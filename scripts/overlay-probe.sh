#!/usr/bin/env bash
# 이 환경에서 오버레이 워크스페이스를 세울 수 있는지 잰다. 아무것도 안 고친다.
#
#   bash scripts/overlay-probe.sh [작업경로]
#
# 작업경로는 실제 워크스페이스가 앉을 디스크여야 한다. 파일시스템이 답을
# 가르기 때문이다 — 그 경로가 이미 overlayfs 면 윗 층을 거기 못 둔다.
# 안 주면 /var/tmp 를 쓴다.
#
# 사다리 넷을 위에서부터 시도하고 처음 되는 것을 답으로 낸다.
#
#   kernel   mount -t overlay          CAP_SYS_ADMIN 이 필요하다
#   userns   unshare -Ur 안에서 마운트  특권이 0 이다.  seccomp 와 AppArmor 가 가른다
#   fuse     fuse-overlayfs            /dev/fuse 가 이미 있어야 한다
#   none     셋 다 실패                 그 노드는 오버레이를 광고하지 않는다
#
# 이 스크립트는 환경을 안 건드린다 — 설치하지 않고, 마운트를 안 남기고,
# 커널 파라미터를 안 고친다. 만든 임시 디렉터리는 끝에 지운다.
#
# 되는지가 아니라 실제로 한 번 해보는 것이 요점이다. 설치돼 있는 것과
# 쓸 수 있는 것이 다르다.
set -uo pipefail

BASE="${1:-/var/tmp}"
WORK="$BASE/.overlay-probe.$$"
RESULT="none"
DEPTH_OK="-"
MOUNT_MS="-"

cleanup() {
  mountpoint -q "$WORK/merged" 2>/dev/null && umount "$WORK/merged" 2>/dev/null
  mountpoint -q "$WORK/deep" 2>/dev/null && umount "$WORK/deep" 2>/dev/null
  rm -rf "$WORK" 2>/dev/null
}
trap cleanup EXIT

say() { printf '%s\n' "$*"; }
kv()  { printf '  %-22s %s\n' "$1" "$2"; }

# ----------------------------------------------------------------- 환경

say "== 환경"
kv "커널" "$(uname -r)"
kv "배포판" "$( . /etc/os-release 2>/dev/null && echo "${PRETTY_NAME:-unknown}" )"
kv "uid" "$(id -u) ($(id -un))"

incontainer="no"
[ -f /.dockerenv ] && incontainer="docker (/.dockerenv)"
if [ "$incontainer" = "no" ] && grep -qE 'docker|kubepods|lxc|containerd' /proc/1/cgroup 2>/dev/null; then
  incontainer="container (cgroup)"
fi
if [ "$incontainer" = "no" ] && command -v systemd-detect-virt >/dev/null 2>&1; then
  v="$(systemd-detect-virt --container 2>/dev/null)"
  [ -n "$v" ] && [ "$v" != "none" ] && incontainer="$v"
fi
kv "컨테이너" "$incontainer"
kv "pid1" "$(tr -d '\0' < /proc/1/comm 2>/dev/null || echo unknown)"

capeff="$(awk '/^CapEff:/{print $2}' /proc/self/status 2>/dev/null)"
capbnd="$(awk '/^CapBnd:/{print $2}' /proc/self/status 2>/dev/null)"
capbit() { # <16진 마스크> -> yes/no.  CAP_SYS_ADMIN 은 21번 비트다
  [ -z "${1:-}" ] && { echo unknown; return; }
  if (( 0x$1 & (1 << 21) )); then echo yes; else echo no; fi
}
kv "CAP_SYS_ADMIN (유효)" "$(capbit "$capeff")"
kv "CAP_SYS_ADMIN (상한)" "$(capbit "$capbnd")   <- 컨테이너 생성 때 고정된다"

sysctl_of() { # sysctl 바이너리가 없는 최소 컨테이너가 많다 -- proc 를 직접 읽는다
  local f="/proc/sys/${1//./\/}"
  if [ -r "$f" ]; then cat "$f"; else echo "(없음)"; fi
}
kv "userns 제한 (24.04)" "$(sysctl_of kernel.apparmor_restrict_unprivileged_userns)"
kv "userns clone" "$(sysctl_of kernel.unprivileged_userns_clone)"
kv "max_user_namespaces" "$(sysctl_of user.max_user_namespaces)"

# ----------------------------------------------------------------- 대상 경로

say ""
say "== 대상 경로  $BASE"
mkdir -p "$WORK" 2>/dev/null || { say "  만들 수 없다. 다른 경로를 주라"; exit 1; }
fstype="$(stat -f -c %T "$WORK" 2>/dev/null || echo unknown)"
kv "파일시스템" "$fstype"
if [ "$fstype" = "overlayfs" ]; then
  kv "경고" "이미 overlayfs 다 — 윗 층을 여기 못 둔다.  호스트 볼륨을 주라"
fi
kv "여유" "$(df -h "$WORK" 2>/dev/null | awk 'NR==2{print $4" / "$2}')"

# reflink 는 copy_up 비용을 바꾼다 (XFS reflink · btrfs)
dd if=/dev/zero of="$WORK/reflink.src" bs=4096 count=8 status=none 2>/dev/null
if cp --reflink=always "$WORK/reflink.src" "$WORK/reflink.dst" 2>/dev/null; then
  kv "reflink" "yes"
else
  kv "reflink" "no"
fi
rm -f "$WORK/reflink.src" "$WORK/reflink.dst"

# ----------------------------------------------------------------- 사다리

prepare_dirs() {
  rm -rf "$WORK/lower" "$WORK/upper" "$WORK/work" "$WORK/merged"
  mkdir -p "$WORK/lower" "$WORK/upper" "$WORK/work" "$WORK/merged"
  echo baseline > "$WORK/lower/probe.txt"
}

# 마운트가 실제로 갈라놓는지 본다 — 윗 층에 쓰고 아래 층이 안 변해야 한다
verify_split() { # <합친경로> <아래층경로> <윗층경로>
  echo changed > "$1/probe.txt" 2>/dev/null || return 1
  [ "$(cat "$2/probe.txt" 2>/dev/null)" = "baseline" ] || return 2
  [ -f "$3/probe.txt" ] || return 3
  return 0
}

say ""
say "== 사다리"

# 1. kernel
printf '  %-8s ' "kernel"
prepare_dirs
t0=$(date +%s%N)
err="$(mount -t overlay overlay \
  -o "lowerdir=$WORK/lower,upperdir=$WORK/upper,workdir=$WORK/work" \
  "$WORK/merged" 2>&1)"
rc=$?
if [ $rc -eq 0 ]; then
  MOUNT_MS=$(( ($(date +%s%N) - t0) / 1000000 ))
  if verify_split "$WORK/merged" "$WORK/lower" "$WORK/upper"; then
    say "PASS   (${MOUNT_MS}ms)"
    RESULT="kernel"
  else
    say "FAIL   마운트는 됐는데 층이 안 갈린다"
  fi
  umount "$WORK/merged" 2>/dev/null
else
  say "FAIL   ${err:-rc=$rc}"
fi

# 2. userns.  마운트가 그 네임스페이스 안에만 보이므로 검증도 안에서 한다
printf '  %-8s ' "userns"
if ! command -v unshare >/dev/null 2>&1; then
  say "SKIP   unshare 가 없다"
else
  prepare_dirs
  t0=$(date +%s%N)
  err="$(W="$WORK" unshare --user --map-root-user --mount -- \
    sh -c 'mount -t overlay overlay \
             -o lowerdir=$W/lower,upperdir=$W/upper,workdir=$W/work \
             $W/merged || exit 11
           echo changed > $W/merged/probe.txt || exit 12
           [ "$(cat $W/lower/probe.txt)" = baseline ] || exit 13
           [ -f $W/upper/probe.txt ] || exit 14' 2>&1)"
  rc=$?
  ms=$(( ($(date +%s%N) - t0) / 1000000 ))
  case $rc in
    0)  say "PASS   (${ms}ms)"
        [ "$RESULT" = "none" ] && { RESULT="userns"; MOUNT_MS="$ms"; } ;;
    11) say "FAIL   네임스페이스는 열렸는데 마운트가 거절됐다 — ${err:-}" ;;
    1[2-4]) say "FAIL   마운트는 됐는데 층이 안 갈린다 (rc=$rc)" ;;
    *)  say "FAIL   네임스페이스를 못 연다 — ${err:-rc=$rc}"
        # 24.04 의 기본값이 1 이고 그것이 uid_map 쓰기를 막는다.  맨 VM 에서 이 칸이
        # 닫히는 가장 흔한 이유라 값을 그 자리에서 보여준다.
        if [ "$(sysctl_of kernel.apparmor_restrict_unprivileged_userns)" = "1" ]; then
          say "         apparmor_restrict_unprivileged_userns 가 1 이다 (24.04 의 기본값)"
          say "         이 칸은 그 기계의 보안 정책이 정한다 — 우리가 바꿀 자리가 아니다"
        fi ;;
  esac
fi

# 3. fuse
printf '  %-8s ' "fuse"
if [ ! -c /dev/fuse ]; then
  say "SKIP   /dev/fuse 가 없다 — 컨테이너 재생성 없이는 못 만든다"
elif ! command -v fuse-overlayfs >/dev/null 2>&1; then
  say "SKIP   fuse-overlayfs 를 깔면 열린다 — /dev/fuse 는 이미 있다"
  say "         sudo apt install fuse-overlayfs.  특권도 커널 설정 변경도 필요 없다"
else
  prepare_dirs
  t0=$(date +%s%N)
  err="$(fuse-overlayfs \
    -o "lowerdir=$WORK/lower,upperdir=$WORK/upper,workdir=$WORK/work" \
    "$WORK/merged" 2>&1)"
  rc=$?
  if [ $rc -eq 0 ]; then
    ms=$(( ($(date +%s%N) - t0) / 1000000 ))
    if verify_split "$WORK/merged" "$WORK/lower" "$WORK/upper"; then
      say "PASS   (${ms}ms)"
      [ "$RESULT" = "none" ] && { RESULT="fuse"; MOUNT_MS="$ms"; }
    else
      say "FAIL   마운트는 됐는데 층이 안 갈린다"
    fi
    fusermount -u "$WORK/merged" 2>/dev/null || umount "$WORK/merged" 2>/dev/null
  else
    say "FAIL   ${err:-rc=$rc}"
  fi
fi

# ----------------------------------------------------------------- 깊이

# 주간 스택 안이 깊이 7 을 요구한다. 커널 상한과 마운트 자체가 서는지만 본다 --
# lookup 비용은 실물 빌드에서만 나온다
say ""
say "== 깊이 7"
mk_layers() {
  rm -rf "$WORK/L" "$WORK/deep" "$WORK/du" "$WORK/dw"
  mkdir -p "$WORK/deep" "$WORK/du" "$WORK/dw"
  DEEP_LOWERS=""
  for i in 1 2 3 4 5 6 7; do
    mkdir -p "$WORK/L/$i"; echo "layer$i" > "$WORK/L/$i/probe.txt"
    DEEP_LOWERS="$WORK/L/$i${DEEP_LOWERS:+:}$DEEP_LOWERS"
  done
}
case "$RESULT" in
  kernel)
    mk_layers
    if mount -t overlay overlay \
         -o "lowerdir=$DEEP_LOWERS,upperdir=$WORK/du,workdir=$WORK/dw" \
         "$WORK/deep" 2>/dev/null; then
      say "  PASS   맨 위가 이긴다: $(cat "$WORK/deep/probe.txt" 2>/dev/null)"
      DEPTH_OK="yes"
      umount "$WORK/deep" 2>/dev/null
    else
      say "  FAIL   깊이 7 을 못 쌓는다"
      DEPTH_OK="no"
    fi ;;
  userns)
    mk_layers
    out="$(W="$WORK" L="$DEEP_LOWERS" unshare --user --map-root-user --mount -- \
      sh -c 'mount -t overlay overlay -o lowerdir=$L,upperdir=$W/du,workdir=$W/dw \
               $W/deep || exit 11
             cat $W/deep/probe.txt' 2>&1)"
    if [ $? -eq 0 ]; then
      say "  PASS   맨 위가 이긴다: $out"
      DEPTH_OK="yes"
    else
      say "  FAIL   깊이 7 을 못 쌓는다 — ${out:-}"
      DEPTH_OK="no"
    fi ;;
  fuse)
    mk_layers
    if fuse-overlayfs -o "lowerdir=$DEEP_LOWERS,upperdir=$WORK/du,workdir=$WORK/dw" \
         "$WORK/deep" 2>/dev/null; then
      say "  PASS   맨 위가 이긴다: $(cat "$WORK/deep/probe.txt" 2>/dev/null)"
      DEPTH_OK="yes"
      fusermount -u "$WORK/deep" 2>/dev/null || umount "$WORK/deep" 2>/dev/null
    else
      say "  FAIL   깊이 7 을 못 쌓는다"
      DEPTH_OK="no"
    fi ;;
  *) say "  SKIP   사다리가 전부 실패해서 잴 것이 없다" ;;
esac

# ----------------------------------------------------------------- 답

say ""
say "== 답"
say "RESULT overlay=$RESULT mount_ms=$MOUNT_MS depth7=$DEPTH_OK fs=$fstype container=$incontainer"
say ""
case "$RESULT" in
  kernel) say "이 기계는 오버레이 노드가 된다. enodectl 이 직접 마운트한다" ;;
  userns) say "특권 없이 된다. enodectl 이 자기 네임스페이스 안에서 마운트하고" 
          say "하네스를 그 안에서 돌린다 — 정리는 프로세스가 죽으면 자동이다" ;;
  fuse)   say "FUSE 로 된다. 커널 마운트보다 느리므로 빌드 성능을 따로 재야 한다" ;;
  none)   say "이 환경에서는 오버레이를 못 세운다. 이 노드는 overlay 를 광고하지"
          say "않고, 오버레이를 요구하는 계약의 후보에서 빠진다"
          say ""
          say "여는 길은 환경마다 다르다"
          say "  VM · 베어메탈   fuse-overlayfs 를 깐다.  /dev/fuse 가 이미 있으면 그것뿐이다"
          say "                 또는 그 유닛에 CAP_SYS_ADMIN 을 준다"
          say "  컨테이너        /dev/fuse 도 CAP_SYS_ADMIN 도 생성 때 정해진다."
          say "                 띄우는 쪽이 고쳐야 하고 안에서는 못 넘는다" ;;
esac
