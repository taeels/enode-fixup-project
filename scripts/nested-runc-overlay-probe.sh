#!/usr/bin/env bash
# userns overlay와 runc를 한 덩어리로 세워 uid 1000 쓰기와 실제 명령을 잰다.
#
#   scripts/nested-runc-overlay-probe.sh [--mount-at PATH] [--ssh-config PATH] ROOTFS LOWER SCRATCH [COMMAND]
#
# ROOTFS  Ubuntu rootfs. /bin/bash, uid/gid 1000 계정, --mount-at 디렉터리가 있어야 한다.
# LOWER   읽기 전용 아래 층으로 쓸 준비된 워크스페이스.
# SCRATCH upper/work/merged와 결과를 둘 디렉터리. Docker overlay2가 아닌
#         호스트 bind volume을 줘야 한다.
# COMMAND runc 안에서 --mount-at 경로를 cwd로 실행할 셸 한 줄. 생략하면 쓰기만 검사한다.
#
# 환경을 설치하거나 고치지 않는다. LOWER는 첫 user namespace 안에서 다시 read-only
# bind한 뒤 overlay lowerdir로 쓴다. 결과 디렉터리는 사후 검사를 위해 남긴다.
set -Eeuo pipefail

usage() {
  cat <<'EOF'
usage: nested-runc-overlay-probe.sh [--mount-at PATH] [--ssh-config PATH] ROOTFS LOWER SCRATCH [COMMAND]

--mount-at PATH  runc 안에서 merged workspace를 보일 절대 경로 (기본 /work).
                 BitBake가 마지막으로 기록한 TOPDIR/TMPDIR의 상위 경로여야 한다.
                 ROOTFS 안에 이 디렉터리를 미리 만들어야 한다.
--ssh-config PATH 호스트의 SSH config 파일. rootfs uid 1000 사용자의
                  ~/.ssh/config에 읽기 전용으로 탑재한다. 키와 known_hosts는 탑재하지 않는다.
                  ROOTFS 안에 uid/gid 1000 소유의 대상 빈 파일을 미리 만들어야 한다.

example:
  scripts/nested-runc-overlay-probe.sh \
    "$HOME/rootfs" /srv/yocto "$HOME/ovl-probe" \
    'source /work/poky/oe-init-build-env /work/build >/dev/null && bitbake -C compile virtual/kernel'

# BitBake가 /srv/workspaces/product를 기록했을 때:
  sudo install -d -m 0755 "$HOME/rootfs/srv/workspaces/product"
  scripts/nested-runc-overlay-probe.sh --mount-at /srv/workspaces/product \
    "$HOME/rootfs" /srv/workspaces/product "$HOME/ovl-probe" \
    'source /srv/workspaces/product/poky/oe-init-build-env /srv/workspaces/product/build >/dev/null && bitbake -C compile virtual/kernel'

# SSH host alias가 필요한 private repo:
  scripts/nested-runc-overlay-probe.sh --ssh-config "$HOME/.ssh/config" \
    "$HOME/rootfs" /srv/yocto "$HOME/ovl-probe"
EOF
}

fail() {
  printf 'FAIL %s\n' "$*" >&2
  exit 1
}

MOUNT_AT="/work"
SSH_CONFIG=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --mount-at)
      [ "$#" -ge 2 ] || fail "--mount-at 뒤에 경로가 필요하다"
      MOUNT_AT="$2"
      shift 2
      ;;
    --ssh-config)
      [ "$#" -ge 2 ] || fail "--ssh-config 뒤에 경로가 필요하다"
      SSH_CONFIG="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --)
      shift
      break
      ;;
    -*)
      fail "알 수 없는 옵션이다: $1"
      ;;
    *)
      break
      ;;
  esac
done

[ "$#" -ge 3 ] && [ "$#" -le 4 ] || { usage >&2; exit 2; }

for command_name in runc unshare newuidmap newgidmap python3 mount umount mountpoint findmnt realpath; do
  command -v "$command_name" >/dev/null 2>&1 || fail "$command_name 가 없다"
done

[ "$(id -u)" -ne 0 ] || fail "root가 아니라 개발자 uid로 실행해야 한다"
[ "$(id -u)" -eq 1000 ] || fail "이 probe는 사내 개발자 uid 1000 계약을 잰다 (현재 $(id -u))"
[ "$(id -g)" -eq 1000 ] || fail "이 probe는 사내 개발자 gid 1000 계약을 잰다 (현재 $(id -g))"

ROOTFS="$(realpath "$1")"
LOWER="$(realpath "$2")"
mkdir -p "$3" || fail "scratch를 만들 수 없다: $3"
SCRATCH="$(realpath "$3")"

if [ -n "$SSH_CONFIG" ]; then
  SSH_CONFIG="$(realpath "$SSH_CONFIG")" || fail "SSH config 경로를 해석할 수 없다"
  [ -f "$SSH_CONFIG" ] || fail "SSH config가 일반 파일이 아니다: $SSH_CONFIG"
  [ -r "$SSH_CONFIG" ] || fail "SSH config를 읽을 수 없다: $SSH_CONFIG"
fi

case "$MOUNT_AT" in
  /*) ;;
  *) fail "--mount-at은 절대 경로여야 한다: $MOUNT_AT" ;;
esac
MOUNT_AT_ROOT="$(realpath -m "$ROOTFS$MOUNT_AT")"
case "$MOUNT_AT_ROOT" in
  "$ROOTFS"/*) ;;
  *) fail "--mount-at이 rootfs 밖을 가리킨다: $MOUNT_AT" ;;
esac
MOUNT_AT="${MOUNT_AT_ROOT#"$ROOTFS"}"
[ "$MOUNT_AT" != / ] || fail "--mount-at에 / 자체를 쓸 수 없다"
RUN_COMMAND="${4:-printf 'probe-write-ok\n' > \"\$ENODE_WORKSPACE/.enode-write-test\"; rm \"\$ENODE_WORKSPACE/.enode-write-test\"; echo COMMAND_OK}"

[ -d "$ROOTFS" ] || fail "rootfs가 디렉터리가 아니다: $ROOTFS"
[ -x "$ROOTFS/bin/bash" ] || fail "rootfs에 /bin/bash가 없다: $ROOTFS"
[ -d "$ROOTFS$MOUNT_AT" ] || fail "rootfs에 $MOUNT_AT가 없다: sudo install -d -m 0755 '$ROOTFS$MOUNT_AT'"
[ -d "$LOWER" ] || fail "아래 층이 디렉터리가 아니다: $LOWER"
[ -r "$LOWER" ] || fail "아래 층을 읽을 수 없다: $LOWER"
[ -w "$SCRATCH" ] || fail "scratch에 쓸 수 없다: $SCRATCH"

case "$SCRATCH/" in
  "$LOWER/"*) fail "scratch를 아래 층 안에 둘 수 없다" ;;
esac

for probe_path in "$ROOTFS" "$LOWER" "$SCRATCH"; do
  case "$probe_path" in
    *:*|*,*) fail "overlay 옵션에 쓸 경로에는 ':'나 ','를 둘 수 없다: $probe_path" ;;
  esac
done

scratch_fs="$(stat -f -c %T "$SCRATCH")"
[ "$scratch_fs" != overlayfs ] || fail "scratch가 Docker overlay2 위다. 호스트 bind volume을 주라"

probe_stamp="$(date +%Y%m%d-%H%M%S)-$$"
RUN="$SCRATCH/nested-runc-$probe_stamp"
BUNDLE="$RUN/bundle"
STATE="$RUN/state"
UPPER="$RUN/upper"
OVERLAY_WORK="$RUN/work"
MERGED="$RUN/merged"
LOWER_RO="$RUN/lower-ro"
LOG="$RUN/probe.log"
CONTAINER_ID="enode-probe-$probe_stamp"

mkdir -p "$BUNDLE" "$STATE" "$UPPER" "$OVERLAY_WORK" "$MERGED" "$LOWER_RO"

(
  cd "$BUNDLE"
  runc spec --rootless
)

rootfs_user="$(awk -F: '$3 == 1000 { print $1; exit }' "$ROOTFS/etc/passwd" 2>/dev/null || true)"
rootfs_home="$(awk -F: '$3 == 1000 { print $6; exit }' "$ROOTFS/etc/passwd" 2>/dev/null || true)"
rootfs_group="$(awk -F: '$3 == 1000 { print $1; exit }' "$ROOTFS/etc/group" 2>/dev/null || true)"
[ -n "$rootfs_group" ] || fail "rootfs에 gid 1000 그룹이 없다: sudo chroot '$ROOTFS' /usr/sbin/groupadd -g 1000 enode"
[ -n "$rootfs_user" ] || fail "rootfs에 uid 1000 사용자가 없다: sudo chroot '$ROOTFS' /usr/sbin/useradd -u 1000 -g 1000 -m -s /bin/bash enode"
[ -n "$rootfs_home" ] || fail "rootfs uid 1000 사용자의 home이 비어 있다"
SSH_CONFIG_DEST=""
if [ -n "$SSH_CONFIG" ]; then
  case "$rootfs_home" in
    /*) ;;
    *) fail "rootfs uid 1000의 home이 절대 경로가 아니다: $rootfs_home" ;;
  esac
  SSH_CONFIG_DEST="$rootfs_home/.ssh/config"
  ssh_config_target="$ROOTFS$SSH_CONFIG_DEST"
  ssh_config_dir="$(dirname "$ssh_config_target")"
  [ -f "$ssh_config_target" ] && [ -x "$ssh_config_dir" ] && [ -r "$ssh_config_target" ] && \
    [ -O "$ssh_config_dir" ] && [ -G "$ssh_config_dir" ] && [ -O "$ssh_config_target" ] && [ -G "$ssh_config_target" ] || \
    fail "rootfs SSH config 대상은 uid/gid 1000 소유의 접근 가능한 빈 파일이어야 한다: sudo install -d -o 1000 -g 1000 -m 700 '$ssh_config_dir' && sudo install -o 1000 -g 1000 -m 600 /dev/null '$ssh_config_target'"
fi

PROBE_ROOTFS="$ROOTFS" \
PROBE_MERGED="$MERGED" \
PROBE_MOUNT_AT="$MOUNT_AT" \
PROBE_SSH_CONFIG="$SSH_CONFIG" \
PROBE_SSH_CONFIG_DEST="$SSH_CONFIG_DEST" \
PROBE_COMMAND="$RUN_COMMAND" \
PROBE_USER="$rootfs_user" \
PROBE_HOME="$rootfs_home" \
python3 - "$BUNDLE/config.json" <<'PY'
import json
import os
import sys

path = sys.argv[1]
with open(path) as config_file:
    config = json.load(config_file)

config["root"]["path"] = os.environ["PROBE_ROOTFS"]
config["process"]["terminal"] = False
config["process"]["user"]["uid"] = 1000
config["process"]["user"]["gid"] = 1000
mount_at = os.environ["PROBE_MOUNT_AT"]
config["process"]["cwd"] = mount_at
config["process"]["env"] = [
    "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
    f"HOME={os.environ['PROBE_HOME']}",
    f"USER={os.environ['PROBE_USER']}",
    f"LOGNAME={os.environ['PROBE_USER']}",
    "LANG=en_US.UTF-8",
    "LC_ALL=en_US.UTF-8",
    f"ENODE_WORKSPACE={mount_at}",
]
wrapper = """set -e
start_ns=$(cat "$ENODE_WORKSPACE/.enode-probe-start-ns")
now_ns=$(date +%s%N)
echo READY_MS=$(( (now_ns-start_ns)/1000000 ))
echo INNER_ID
id
echo INNER_UID_MAP
cat /proc/self/uid_map
echo INNER_GID_MAP
cat /proc/self/gid_map
""" + os.environ["PROBE_COMMAND"]
config["process"]["args"] = ["/bin/bash", "-lc", wrapper]
config["mounts"] = [
    mount
    for mount in config["mounts"]
    if mount.get("destination") not in ("/sys", "/sys/fs/cgroup", mount_at)
]
config["mounts"].append(
    {
        "destination": mount_at,
        "type": "bind",
        "source": os.environ["PROBE_MERGED"],
        "options": ["rbind", "rw"],
    }
)
ssh_config = os.environ["PROBE_SSH_CONFIG"]
if ssh_config:
    config["mounts"].append(
        {
            "destination": os.environ["PROBE_SSH_CONFIG_DEST"],
            "type": "bind",
            "source": ssh_config,
            "options": ["rbind", "ro", "nosuid", "nodev", "noexec"],
        }
    )
config["linux"]["uidMappings"] = [
    {"containerID": 0, "hostID": 1, "size": 1000},
    {"containerID": 1000, "hostID": 0, "size": 1},
    {"containerID": 1001, "hostID": 1001, "size": 64536},
]
config["linux"]["gidMappings"] = [
    {"containerID": 0, "hostID": 1, "size": 1000},
    {"containerID": 1000, "hostID": 0, "size": 1},
    {"containerID": 1001, "hostID": 1001, "size": 64536},
]

with open(path, "w") as config_file:
    json.dump(config, config_file, indent=2)
PY

printf '== 환경\n'
printf 'uid=%s gid=%s user=%s\n' "$(id -u)" "$(id -g)" "$(id -un)"
printf 'rootfs=%s\nlower=%s\nmount_at=%s\nscratch=%s\n' "$ROOTFS" "$LOWER" "$MOUNT_AT" "$SCRATCH"
printf 'ssh_config=%s\n' "$([ -n "$SSH_CONFIG" ] && echo mounted-ro || echo absent)"
printf 'lower_fs=%s scratch_fs=%s\n' "$(stat -f -c %T "$LOWER")" "$scratch_fs"
printf 'subuid=%s\n' "$(grep -E "^$(id -un):" /etc/subuid 2>/dev/null | paste -sd, - || true)"
printf 'subgid=%s\n' "$(grep -E "^$(id -un):" /etc/subgid 2>/dev/null | paste -sd, - || true)"

set +e
PROBE_LOWER="$LOWER" \
PROBE_LOWER_RO="$LOWER_RO" \
PROBE_UPPER="$UPPER" \
PROBE_WORK="$OVERLAY_WORK" \
PROBE_MERGED="$MERGED" \
PROBE_STATE="$STATE" \
PROBE_BUNDLE="$BUNDLE" \
PROBE_ID="$CONTAINER_ID" \
unshare --user --map-root-user --map-auto --mount --fork --kill-child bash -c '
  set -Eeuo pipefail

  cleanup_mounts() {
    if mountpoint -q "$PROBE_MERGED"; then
      umount "$PROBE_MERGED" 2>/dev/null || umount -l "$PROBE_MERGED" 2>/dev/null || true
    fi
    if mountpoint -q "$PROBE_LOWER_RO"; then
      umount "$PROBE_LOWER_RO" 2>/dev/null || umount -l "$PROBE_LOWER_RO" 2>/dev/null || true
    fi
  }
  trap cleanup_mounts EXIT

  mount --make-rprivate /
  mount --bind "$PROBE_LOWER" "$PROBE_LOWER_RO"
  mount -o remount,bind,ro "$PROBE_LOWER_RO"
  printf "%s\n" "$(date +%s%N)" > "$PROBE_UPPER/.enode-probe-start-ns"
  mount -t overlay overlay \
    -o "lowerdir=$PROBE_LOWER_RO,upperdir=$PROBE_UPPER,workdir=$PROBE_WORK" \
    "$PROBE_MERGED"

  echo PARENT_UID_MAP
  cat /proc/self/uid_map
  echo PARENT_GID_MAP
  cat /proc/self/gid_map
  findmnt -T "$PROBE_LOWER_RO" -o TARGET,FSTYPE,OPTIONS
  findmnt -T "$PROBE_MERGED" -o TARGET,FSTYPE,OPTIONS

  set +e
  runc --root "$PROBE_STATE" run --bundle "$PROBE_BUNDLE" "$PROBE_ID"
  run_rc=$?
  set -e
  cleanup_mounts
  trap - EXIT
  exit "$run_rc"
' 2>&1 | tee "$LOG"
run_rc="${PIPESTATUS[0]}"
set -e

ready_ms="$(awk -F= '/^READY_MS=/{value=$2} END{print value}' "$LOG")"
upper_size="$(du -sh "$UPPER" 2>/dev/null | awk '{print $1}')"
upper_files="$(find "$UPPER" -xdev -type f 2>/dev/null | wc -l)"
mount_left="no"
if [ "$(findmnt -n -o FSTYPE -T "$MERGED" 2>/dev/null || true)" = overlay ]; then
  mount_left="yes"
fi

printf '== 답\n'
printf 'RESULT nested_userns_runc=%s ready_ms=%s rc=%s upper=%s files=%s mount_left=%s\n' \
  "$([ "$run_rc" -eq 0 ] && [ "$mount_left" = no ] && echo PASS || echo FAIL)" \
  "${ready_ms:--}" "$run_rc" "${upper_size:--}" "$upper_files" "$mount_left"
printf 'RESULT_DIR %s\n' "$RUN"

[ "$run_rc" -eq 0 ] && [ "$mount_left" = no ]
