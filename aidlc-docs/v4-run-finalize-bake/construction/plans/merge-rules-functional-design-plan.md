# `merge-rules` — Functional Design 계획

**유닛** `merge-rules` (합치기 규칙) · **브랜치** `unit/merge-rules` · **담당** taeels ·
**회차** `v4-run-finalize-bake` (굽기) · **순서** 여덟 중 다섯째 · **앞 유닛** trash (`main` `b44b46e` 에 병합) ·
**맡는 조각** 7 의 기계 부분 (가짜 트리 재개 시험 — 합치기를 여러 지점에서 끊고 다시 돌려도 결과가 같은가) ·
**병합 조건** 코드 검사 + 재개 시험 · **요구** FR-7 (기능 7 · 합치기) · **스토리** 없음 (스토리 지도의 ⑤ 줄이 0 이다)

입력은 유닛 정의(`unit-of-work.md` 5절 · 10절 크로스 빌드) · 요구(`requirements.md` FR-7 · 5.3 보안 표의 합치기 줄 ·
5.4 성능 · 5.6 시험) · 설계(`components.md` 2.2 · `component-methods.md` 2절) · 정본(ADR-077 §4 합치기 표 · §12 실측) ·
팩의 결정 3-5 (합치기는 파일 단위 rename · 표시는 적용한 뒤에만 치운다) · SunnyVM 의 시제품
(`~/bake-measure/merge.py` 139 줄 · `toy-test.sh` · `listing.py`)이다. 이 계획은 그 위에서
**대기 upper 를 lower 에 어떻게 합치나**만 짓는다. 잠금과 상태와 굽기 단계는 뒤 유닛이고, 코드는 다음 단계다.

---

## 0. 이 단계가 닫는 것과 안 닫는 것

```text
   닫는다     ADR-077 §4 표의 줄마다의 동작과, 표에 없는 줄 — 종류가 바뀐 항목 (FR-7)
             표시를 읽는 법 — whiteout (문자 장치 0/0 · xattr 형식은 물음 8) · opaque (값 y 만) · 금지 표시
             시작 전 확인 — 무엇을 보고, 어긋나면 무엇을 돌려주나
             재개 성질 — 표시는 적용한 뒤에만 치운다.  「연산 한 번」의 단위
             lower 쪽을 trash 로 옮기는 자리와 이름
             lower 에도 있는 디렉터리를 합친 뒤 맞추는 속성
             합치는 경로가 lower 밖으로 안 나가게 (보안 표)
             결과 세기 · 도중 오류 · 멈춤
             가짜 트리 재개 시험의 모양과 기대값 · CI 에서 가짜 표시를 못 만들 때
             코드 경계 시험 두 줄 · 크로스 빌드 짝

   안 닫는다   lower 잠금 · bake 잠금 · 상태 파일 · lower 신원                       lower-state 유닛
             merge 단계 · 대기 · merge-helper 입구 (namespace) · 시작 때 재개        bake 유닛
             .enode-metadata.json 쓰기 (FR-9)                                   bake 유닛
             런타임의 overlay 마운트 옵션                                        바꾸지 않는다 (1.2 — 이미 맞다)
             값의 크기 (NFR)                                                     이 유닛의 NFR Requirements
```

---

## 1. 실측 (2026-09-26)

### 1.1 시제품 `merge.py` — 표의 줄마다 무엇을 했나

SunnyVM 의 `~/bake-measure/merge.py` 를 읽었다. ADR-077 §12 의 실측(하루치 합치기 연산 59,030 번 · 1.49 초 · merged view
765,107 항목과 일치)과 가짜 트리 재개 시험(1 ~ 30번째 연산 뒤 강제 종료)을 이 파일이 했다.

```text
   upper 의 항목                        lower              merge.py 가 한 일
   whiteout (문자 장치 0/0 또는        있다                lower 를 trash 로 · whiteout 을 unlink
     user.overlay.whiteout xattr)      없다                whiteout 을 unlink
   opaque 디렉터리                      있다                lower 를 trash 로 · opaque xattr 을 지움 · rename
                                       없다                opaque xattr 을 지움 · rename
   디렉터리 (opaque 아님)                진짜 디렉터리         안으로 들어간다 · 끝나면 chmod · utime · upper 쪽 rmdir
                                       파일 · symlink       lower 를 trash 로 · rename          (종류가 바뀐 항목)
                                       없다                rename (하위 트리째 한 번)
   파일 · symlink · 그 밖                디렉터리 (진짜)       lower 를 trash 로 · rename          (종류가 바뀐 항목)
                                       파일 · symlink · 없다  rename (있으면 대체)
```

- **FR-7 의 물음 — 종류가 바뀐 항목을 어떻게 다뤘나.** 두 방향 모두 lower 쪽을 trash 로 옮긴 뒤 rename 했다. lower 의
  symlink 가 디렉터리를 가리켜도 디렉터리로 치지 않는다 (`isdir and not islink`). 표에 이 줄을 더하면 된다
- 디렉터리 속성은 mode 와 시각만 맞췄다. ADR-077 §4 표의 「소유」는 안 맞췄다 (물음 4)
- 시작 전 확인은 둘 — upper · lower · trash 의 `st_dev` 가 같다 · upper 전체를 걸어 `user.overlay.metacopy` ·
  `user.overlay.redirect` 가 없다 (§12 의 금지 xattr 사전 검사 1.00 초 · 3.95 초)
- 표시를 읽는 법 — opaque 는 값이 `y` 일 때만 opaque 로 읽었다. whiteout 은 문자 장치 0/0 이거나, `user.overlay.whiteout`
  이 붙은 항목이면 크기와 부모 표시를 보지 않고 whiteout 으로 읽었다. `toy-test.sh` 는 진짜 overlay 마운트로 표시를
  만들었으므로 xattr whiteout 의 길은 시험된 적이 없다 (1.7 · 물음 8)
- 「연산」을 항목 하나마다 셌다 — trash 로 옮기기만 따로 한 번이다 (물음 2). opaque 의 xattr 지우기와 rename 사이 ·
  디렉터리의 chmod · utime · rmdir 사이에서는 안 끊겼다. 앞 판은 「호출마다 셌다」고 적었는데 틀렸다 (답을 받은 뒤
  2026-09-26T16:02:51Z 에 다시 읽고 고쳤다). 답 2 = A (호출마다) 는 시제품보다 촘촘하다
- 디렉터리 mtime 은 들어가기 전에 읽은 값을 맞췄다. 재개하면 upper 디렉터리의 mtime 이 이미 바뀌어 있어 틀어진다 —
  `listing.py` 가 디렉터리 mtime 을 비교에서 뺀 까닭이다
- trash 의 이름은 `<pid>-<시각>-<번호>-<끝 조각>` 이었다 (물음 6)
- `--skip` 은 실측용 표지 파일(`.enode-probe-start-ns`) 하나를 비키려던 것이다. 제품에는 필요 없다

### 1.2 제품 런타임이 upper 에 남기는 표시 (SunnyVM · 커널 7.0)

제품 helper 는 overlay 를 `lowerdir · upperdir · workdir` 만으로 마운트한다 (`runc_overlay_linux.go`). 같은 옵션과
`userxattr` 를 명시한 것을 user namespace 안에서 나란히 측정했다 — **결과가 같다.**

```text
   커널이 붙인 옵션      redirect_dir=nofollow,uuid=on,userxattr   (user namespace 안의 마운트라 스스로 붙인다)
   모듈 기본값          metacopy=N · index=N · redirect_dir=N
   echo >> a.txt        upper 에 a.txt · user.overlay.origin (복사해 올린 표지)
   chmod 700 d1         upper 에 d1 · user.overlay.origin
   rm d1/c.txt          d1/c.txt 문자 장치 0/0
   rm -rf d2; mkdir d2  d2 가 user.overlay.opaque = "y"
   mv d3 d3moved        d3 문자 장치 0/0 · d3moved 는 새 디렉터리 (redirect 없이 통째 복사 — EXDEV)
   rm f_to_dir; mkdir   f_to_dir 가 opaque 디렉터리
   rm -rf d_to_file;    d_to_file 이 보통 파일 (whiteout 없음)
     echo > d_to_file
   ln -sfn new link     link 가 upper 의 symlink
```

- **표시 형식은 ADR-077 §4 가 전제한 그대로다.** 문자 장치 whiteout · `user.overlay.opaque` · redirect 없음 · metacopy 없음.
  런타임의 마운트 옵션을 바꿀 일이 없다
- 복사해 올린 항목에 `user.overlay.origin` 이 붙는다. rename 하면 그 xattr 이 lower 로 따라간다 (물음 5)
- 새 디렉터리(lower 에 없는 것)의 하위에는 표시가 생기지 않는다 — 가릴 lower 가 없기 때문이다. 그래서 하위 트리째
  rename 한 번이 맞다

### 1.3 형제 helper 의 overlay 마운트는 호스트에서 안 보인다

SunnyVM 에서 helper 와 같은 모양(`unshare --user --map-root-user --mount` 안에서 워크스페이스를 bind 한 뒤 그것을
lowerdir 로 overlay)을 띄우고 측정했다.

```text
   호스트의 /proc/self/mountinfo          그 경로가 나오는 줄 0
   helper 의 /proc/<pid>/mountinfo        bind 줄 — root 칸이 lower 경로 · 장치 sda2
                                          overlay 줄 — lowerdir 는 bind 한 자리 (lower 경로가 아니다)
   노드 사용자가 그 파일을 읽을 수 있나      읽힌다 (같은 uid 의 프로세스)
```

「이 lower 의 overlay 마운트 0」을 호스트 mountinfo 로는 못 본다. 프로세스마다의 mountinfo 를 훑으면 본다 (물음 1).

### 1.4 가짜 표시 — 특권 없이 만들어진다 (이 기계 · 커널 6.5 · ext4 · 2026-09-26 다시 측정했다)

`mknod c 0 0` · `setxattr user.overlay.opaque` · `setxattr user.overlay.whiteouts` 모두 성공. Application Design 1.3 과 같다.
CI 의 `ubuntu-latest` 에서는 이 유닛의 첫 CI 가 확인한다 — 못 만들면 시험이 **실패**한다 (스킵하지 않는다).
`user.overlay.whiteouts` 는 지금 커널 문서에 없는 이름이다. 문서는 부모 디렉터리에 `user.overlay.opaque` = `x` 를 적는다 (1.7).

### 1.5 upper 의 뿌리

제품의 upper 는 helper 가 `MkdirAll(0o700)` 로 만든 `runRoot/upper` 다. 그 뿌리의 mode · 시각을 lower 뿌리에 맞추면 lower
뿌리가 0700 이 된다. **뿌리의 속성은 안 맞춘다** (merge.py 도 안 맞췄다).

### 1.6 코드

`internal/merge` 는 없다. `internal/panel/boundary_test.go` 의 봉인은 셋 (`internal/transcript` · `internal/transcriptui` ·
`internal/scratch`) 이고, 넷째가 된다.

### 1.7 upper 의 xattr whiteout — 커널이 반쪽만 읽는다 (SunnyVM · 커널 7.0 · 2026-09-26)

커널 문서(`Documentation/filesystems/overlayfs.rst`)가 적는 것 둘.

- opaque 는 값이 `y` 일 때다. 값 `x` 는 「이 디렉터리 안에 xattr whiteout 이 있다」는 표시이고 **opaque 가 아니다.**
  `x` 를 opaque 로 읽으면 lower 의 그 디렉터리를 통째로 trash 로 보낸다
- xattr whiteout (크기 0 파일 + `overlay.whiteout`)은 **overlayfs 가 만들지 않는다.** 컨테이너 도구가 lower 층을 만들 때
  쓰는 형식이다

제품과 같은 마운트(user namespace · `userxattr`)로 측정했다.

```text
   upper 에 둔 것                                    merged 의 목록 (readdir)    merged 에서 열기 (stat)
   크기 0 파일 + whiteout xattr · 부모 opaque = x      이름이 보인다               없다 (ENOENT)
   크기 0 파일 + whiteout xattr · 부모 표시 없음        이름이 보인다               없다 (ENOENT)
   같은 것을 lower 층에 두면                            안 보인다                   없다

   세션 안에서 파일에 user.overlay.whiteout 과          세션에는 그 이름으로 보인다.  upper 에는
     user.overlay.opaque 를 붙이면                      user.overlay.overlay.* (escape 한 이름)로 적힌다
```

- **upper 의 xattr whiteout 은 세션이 본 모습이 하나가 아니다** — 목록에는 있고 열면 없다. 어느 쪽으로 합쳐도 한쪽과 어긋난다
- **제품 세션은 upper 에 진짜 표시를 못 만든다.** 커널이 escape 한 이름으로 적는다. upper 에 xattr whiteout 이나 opaque
  값 `x` 가 있으면 그 upper 는 제품 마운트가 만든 것이 아니다
- escape 한 이름은 보통 xattr 이다. lower 로 옮기면 다음 overlay 가 한 겹 벗겨 세션에 보여 준다 (커널 문서의 층 겹치기 절)
- ADR-077 §4 의 「커널이 xattr 형식을 쓰는 경우도 whiteout 으로 읽는다」와 유닛 정의 5절의 「whiteout 은 문자 장치 0/0 과
  xattr 형식」이 이 사실과 어긋난다 (물음 8)

### 1.8 origin · impure 가 lower 로 따라가도 다음 overlay 는 정상이다 (SunnyVM · 커널 7.0 · 2026-09-26)

세션에서 기존 파일에 덧붙이기 · 새 디렉터리 만들기 · lower 파일을 새 디렉터리로 `mv` · 하위 파일에 덧붙이기를 하고,
ADR-077 §4 의 규칙대로 손으로 합친 뒤 그 lower 위에 새 overlay 를 올렸다.

```text
   합친 lower 에 남은 표시    a.txt · d/b.txt · newdir/x.txt 에 user.overlay.origin · newdir 에 user.overlay.impure
                             (d 는 양쪽 디렉터리라 upper 쪽을 rmdir 했다 — 그 표시는 안 따라간다)
   새 overlay 에서            목록이 lower 와 같다 · 내용이 맞다 · 덧붙이기(다시 복사해 올리기)가 된다
   새 upper                   새 origin 이 붙는다 (옛 값을 옮기지 않는다)
```

물음 5 의 A 가 이 측정에 기댄다. 측정 폴더는 지웠다.

---

## 2. 물음 아홉

답을 `[Answer]:` 뒤에 적는다. 권장을 **A** 에 둔다. 기호 옆에 뜻을 적었다.

**다시 검토 (2026-09-26)** — 앞 판을 근거 문서와 측정에 다시 대 보고 넷을 고쳤다.

```text
   물음 1   Application Design 이 이미 정한 것이다 (잠금이 증거 · 증거는 lower-state 유닛이 닫는다).  A 를 그 결정대로 바꿨다
   물음 5   A 의 근거가 주장이었다.  측정으로 바꿨다 (1.8)
   물음 6   앞 판의 B 는 Application Design 의 봉인과 어긋났고, trash 유닛이 넘긴 길 (Trash.Move) 이 빠져 있었다
   물음 8   앞 판의 B 는 FR-7 과 어긋났다.  대신 1.7 의 새 사실 — upper 의 xattr whiteout — 을 묻는다
   그대로   2 · 3 · 4 · 7 · 9.  3 · 7 · 9 에는 한 줄씩 더했다
```

### Question 1 — 「이 lower 의 overlay 마운트 0」을 이 유닛이 확인하나

FR-7 은 시작 전 확인 넷에 마운트 0 을 넣었다. Application Design 은 그 증거를 **부르는 쪽이 쥔 배타 잠금**으로 정했고
(`components.md` 2.2 · `component-methods.md` 2절의 `Preflight` 주석), 유닛 정의는 「마운트 0 의 증거」를 **lower-state 유닛의
Functional Design** 에 맡겼다 (`unit-of-work.md` 6절). 1.3 의 측정(형제의 마운트가 호스트 mountinfo 에 안 보인다)은 그 결정의
전제를 확인한 것이다.

남는 틈 하나 — lower-state 유닛이 들어오기 전의 옛 노드는 잠금을 안 쥔다. SunnyVM 의 yocto 노드가 2026-09-23 빌드로
`/srv/yocto` 에 떠 있다.

A) **이 유닛은 확인하지 않는다 (Application Design 그대로).** `Preflight` 는 마운트를 보지 않는다. 1.3 의 측정과 옛 노드의
   틈을 lower-state 유닛의 Functional Design 에 넘긴다 — 그 유닛이 「마운트 0 의 증거」를 닫으면서 옛 노드를 어떻게 막을지
   (프로세스마다의 `/proc/<pid>/mountinfo` 를 훑는다 · 굽기 전에 같은 lower 의 노드를 모두 새 판으로 올린다 등) 정한다

B) `Preflight` 가 노드 사용자 프로세스의 `/proc/<pid>/mountinfo` 를 훑어, lower 와 같은 장치에서 lower 경로나 그 조상 · 자손을
   root 로 가진 마운트가 하나라도 있으면 시작하지 않는다. Application Design 을 고친다 — `internal/merge` 가 다른 프로세스의
   마운트 namespace 를 알게 되고 (`components.md` 2.2 의 「모르는 것」에 namespace 가 있다), lower-state 유닛의 몫을 이 유닛이
   가져온다. 비용은 프로세스 수를 따른다

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2 — 재개 시험의 「연산 한 번」

A) **lower 나 upper 를 바꾸는 호출 하나** — rename · unlink · rmdir · xattr 지우기 · 속성 맞추기. merge.py 가 센 단위와
   같다 (trash 로 옮기기도 한 번). 그래서 항목 하나 안의 두 호출 사이 — 예: lower 를 trash 로 옮긴 뒤 whiteout 을 지우기
   전 — 에서도 끊긴다. 시험은 1번째부터 **마지막 호출까지 모두** 끊어 본다 (팩의 「1 ~ 30」을 넘는다 · 가짜 트리 전체)

B) upper 항목 하나 — 항목 안의 호출 사이에서는 안 끊긴다. 끊는 자리가 적다

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3 — 재개 시험의 기대값을 어디서 얻나

요구 5.6 이 부르는 것은 C 까지다 (끊고 다시 돌린 결과를 한 번에 합친 목록과 비교). A 와 B 는 한 번에 합친 결과가
**맞는지**를 더 본다.

A) **순수 Go 모형이 기대값을 계산한다** — lower 와 upper 를 overlay 가 보일 모습(merged view)으로 합친 목록. 기본
   `go test` 는 가짜 트리(특권 없이 만든 표시)로 「모형 = 한 번에 합친 lower = 끊고 다시 돌린 lower」를 본다.
   `integration` 태그 시험(SunnyVM · CI 밖)은 진짜 overlay 마운트로 표시를 만들고 「진짜 merged view = 모형 = 합친 lower」를
   본다 — 모형이 커널과 같은지를 거기서 확인한다 (merge.py 의 `toy-test.sh` 가 한 대조)

B) 손으로 쓴 기대 목록 하나 — 가짜 트리가 고정이라 충분하다. 트리를 고치면 목록도 고친다

C) `integration` 만 — 기본 `go test` 는 재개만 본다 (한 번에 합친 결과 = 끊고 다시 돌린 결과)

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4 — lower 에도 있는 디렉터리를 합친 뒤 맞추는 속성

A) **mode · 소유 · 시각** — ADR-077 §4 표 그대로. 소유는 lchown 이다. 실측(trash 유닛의 조각 4)으로는 upper 와 lower 가
   모두 노드 uid 소유라 바뀌는 일이 없지만 표를 따른다. merge-helper 가 namespace 안에서 돌므로 매핑된 uid 는 바꿀 수 있다

B) mode · 시각 — merge.py 그대로

C) A 에 더해 사용자 xattr 도 옮긴다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5 — overlay 가 upper 에 남긴 사적 xattr 이 lower 로 따라가는 것

복사해 올린 파일과 디렉터리에 `user.overlay.origin` 이 붙고 (1.2), 디렉터리에는 `user.overlay.impure` 가 붙을 수 있다.

A) **그대로 둔다.** 1.8 의 측정 — 두 표시가 lower 에 남아도 다음 overlay 의 목록 · 내용 · 다시 복사해 올리기가 정상이고,
   새 upper 에는 새 origin 이 붙는다. lower 에 남으면 안 되는 표시 중 opaque · whiteout 은 합치기가 치우고, metacopy ·
   redirect 는 시작 전 확인이 막는다. 비용이 바뀐 항목 수를 따른다는 성질을 지킨다 (새 디렉터리는 하위 트리째 rename
   한 번). 이 사실을 정본에 적는다

B) 옮기는 항목 자신에서만 지운다 — 새 디렉터리의 하위는 그대로다

C) 옮긴 하위 트리 전체에서 지운다 — 비용이 하위 트리의 크기를 따른다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6 — lower 쪽을 trash 로 옮기는 일을 누가 하나

전제 둘. **`internal/merge` 는 `internal/scratch` 를 임포트하지 못한다** — 새 패키지 셋은 서로 임포트하지 않는다
(`components.md` 2절). 그리고 **trash 유닛이 bake 유닛에 넘긴 약속**이 있다 — 「합치기의 lower 쪽 항목도 `Trash.Move` 로」
(`construction/trash/functional-design/business-logic-model.md` 10절). 앞 판의 B (`internal/scratch` 를 임포트한다)는 첫째와
어긋나서 뺐다.

A) **부르는 쪽이 채우는 함수로 옮긴다.** `Options` 에 lower 쪽 항목을 버리는 함수 하나를 둔다. `enode merge-helper` 가 그것을
   `scratch.Trash.Move` 로 채운다 — 삭제자의 `Launch` 를 `internal/enode` 가 채우는 것과 같은 모양이다. 이름 규칙(끝 조각 ·
   겹치면 `-1` · `-2` …)이 `internal/scratch` 한 곳에 있고, trash 유닛의 약속을 그대로 지킨다. `Paths.Trash` 는 시작 전
   확인(같은 filesystem)에만 쓴다. 함수가 비어 있으면 `Apply` 가 시작하지 않는다

B) `internal/merge` 가 `Paths.Trash` 에 직접 넣는다 — `renameat2` 의 덮어쓰지 않기(`RENAME_NOREPLACE`)로, 이름은 trash 유닛과
   같은 규칙으로. 이름 규칙이 두 벌이 된다. trash 유닛이 넘긴 약속을 고친다

C) Other (please describe after [Answer]: tag below)

둘 다 trash 에 항목이 낱개로 들어가고, 삭제자는 항목마다 helper 를 한 번 연다 — ADR-077 §12 의 하루치 합치기(B)는 trash 로
526 번 옮겼다. 한 합치기의 항목을 한 디렉터리에 모을지는 bake 유닛이 정한다.

[Answer]: A

### Question 7 — 합치는 경로가 lower 밖으로 안 나가게 (팩 보안 표의 합치기 줄)

A) **디렉터리 fd 에 기대어 내려간다** — `openat(O_NOFOLLOW)` · `fstatat` · `renameat` · `unlinkat`. trash 유닛의 삭제
   걷기와 같다. lower 쪽이 symlink 면 그것이 디렉터리를 가리켜도 들어가지 않고 종류가 바뀐 항목으로 친다 (trash 로
   옮기고 대체 · merge.py 와 같다). 물음 6 이 A 면 lower 쪽 항목을 버리는 한 걸음만 경로로 간다 — 그 위의 조각은 걷기가
   방금 진짜 디렉터리로 확인했고, 배타 잠금 아래라 사이에 바꿀 프로세스가 없다

B) 경로를 이어 붙이고 lstat 으로 본다 (merge.py 그대로). 배타 잠금 아래라 사이에 바꿀 프로세스가 없다

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 8 — 시작 전 확인이 upper 에서 무엇을 거절하나

upper 를 걷는 것 자체는 물음이 아니다. FR-7 이 metacopy 꺼짐 · redirect 없음을 시작 조건으로 적었고, 합칠 때는 마운트가
이미 없어서 upper 의 xattr 말고는 증거가 없다 (ADR-077 §12 · 15만 항목에 1초 · 68만 항목에 3.95초). 앞 판의 B (걷지 않는다)는
FR-7 의 넷 중 둘을 빼는 것이라 뺐다. 같은 filesystem (upper · lower · trash 의 `st_dev`) 과 lower · trash 가 진짜 디렉터리인지도
그대로 본다.

남는 물음은 1.7 이다 — upper 의 xattr whiteout 과 opaque 의 값.

A) **거절한다.** upper 의 어느 항목에든 `user.overlay.metacopy` · `user.overlay.redirect` · `user.overlay.whiteout` 이 있거나,
   `user.overlay.opaque` 의 값이 `y` 가 아니면 시작하지 않는다. 제품 마운트는 이것들을 만들지 않고 세션도 못 만든다
   (1.2 · 1.7). 있으면 그 upper 는 제품이 만든 것이 아니다. whiteout 은 문자 장치 0/0 하나만 읽는다. 이름은 정확히
   맞춘다 — escape 한 이름(`user.overlay.overlay.*`)은 세션이 붙인 보통 xattr 이라 거절하지 않고 그대로 옮긴다. ADR-077 §4 의
   「xattr 형식도 whiteout 으로 읽는다」와 유닛 정의 5절을 정본과 회차 문서에서 고친다

B) xattr whiteout 을 whiteout 으로 읽는다 (유닛 정의 · ADR-077 §4 · merge.py 그대로). 합친 lower 는 세션의 「열기」와 같고
   「목록」과 다르다. opaque 는 값 `y` 만 opaque 이고 `x` 는 보통 디렉터리로 읽는다

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 9 — 도중 오류와 멈춤

A) **첫 오류에서 멈춘다.** ctx 가 끝나면 연산 사이에서 멈춘다. 남은 upper 가 곧 남은 일이고, 같은 절차를 다시 부르면
   잇는다. 그때까지 센 수와 오류를 함께 돌려준다. 디렉터리의 속성 맞추기와 upper 쪽 rmdir 은 그 하위가 다 끝난 뒤에만
   한다 — 멈춘 자리보다 위의 디렉터리는 손대지 않은 채 남는다. 같은 오류가 재개 때마다 되풀이되면 (예: 매핑 밖의 uid 로
   lchown) 상태가 merging 에 머문다 — 그때 무엇을 하나는 bake 유닛에 넘긴다

B) 오류가 난 항목을 건너뛰고 끝까지 간 뒤 모아서 돌려준다 — 한 번에 더 많이 합친다. 대신 부모 디렉터리의 순서가 흐트러진다

C) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## 3. 산출물 계획 (체크박스)

답이 들어오고 모호함이 풀린 뒤에 채운다. 자리는
`aidlc-docs/v4-run-finalize-bake/construction/merge-rules/functional-design/` 이다.

- [x] 답을 읽고 모호함을 확인한다 — 있으면 되물음 파일을 만든다 (2026-09-26T16:02:51Z · 아홉 모두 A · 모호함 없음)
- [x] `domain-entities.md` — `Paths` · `Kind` (종류가 바뀐 항목 포함) · `Op` · `Result` · `Options` · 시작 전 확인의 오류 ·
      `Apply` · `Preflight` · `Classify` 의 입력과 출력 · 시험 모형(물음 3) · linux 와 그 밖의 짝
- [x] `business-rules.md` — 표의 줄마다의 동작과 종류가 바뀐 항목 두 줄 · 표시를 읽는 법 · 표시를 치우는 순서 · 시작 전
      확인 · trash 로 옮기기 · 디렉터리 속성 · 뿌리 · 경로 경계 · 결과 세기 · 오류와 멈춤 · 오류 문구 (영어)
- [x] `business-logic-model.md` — Apply 의 걷기 순서 · 끊긴 지점마다 다시 돌리면 무엇을 하나 (연산마다의 표) · 시험의 모양
      (가짜 트리의 항목 · 끊는 자리 · 기대값) · integration 시험 · 부르는 쪽(bake)과의 약속 · 다른 유닛에 넘기는 것 · 정본 되돌림
- [x] 코드 경계 시험 두 줄 — Mediator 가 `internal/merge` 를 못 가져다 쓴다 · `internal/merge` 는 표준 라이브러리와 x/sys 만
      (`business-rules.md` 11절에 적었다 · 코드는 Code Generation)
- [x] 커버리지 — 새 패키지 80% 이상 · linux 전용 파일이 기본 `go test` 에서 닿는 자리 (`business-logic-model.md` 5절)
- [x] 표기 검사 (`enode-design/scripts/emphasis-check.py`) · 사용자가 싫어한 말투 검사 · 사내 이름 검사 (2026-09-26T16:09:11Z ·
      넷 모두 exit 0 · 말투 0 · 사내 이름 0)

---

## 4. 확장 준수 — 이 단계

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 합치기 줄(같은 filesystem · 경로가 lower 밖으로 안 나감)은 팩의 요구로 남아 물음 7 · 8 이 닫는다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 저장소의 표 시험 관례를 따른다 |

유닛 정의는 이 유닛의 NFR Requirements 를 「한다 (최소)」로 적었다 — 보안(합치는 경로가 lower 밖으로 안 나감 · 같은
filesystem) · 성능(비용이 바뀐 항목 수를 따르고 바이트 수와 무관). Functional Design 이 닫힌 뒤에 정한다.
