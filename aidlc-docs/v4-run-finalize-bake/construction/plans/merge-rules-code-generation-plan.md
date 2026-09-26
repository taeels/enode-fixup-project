# `merge-rules` — Code Generation 계획

**유닛** `merge-rules` (합치기 규칙 · 한 줄 순서의 다섯째) · **브랜치** `unit/merge-rules` · **기준** `f155faf`
(Functional Design 커밋) · **맡는 조각** 7 의 기계 부분 (가짜 트리 재개 시험) · **병합 조건** 코드 검사 + 재개 시험

**이 계획이 이 유닛 Code Generation 의 기준이다.** 여기 없는 것은 짓지 않는다. 설계는
`construction/merge-rules/functional-design/` 의 셋이다. 아래의 **FD** 는 그 셋의 줄임이다 — 「FD 규칙 5절」은
`business-rules.md` 5절, 「FD 흐름 4.1」은 `business-logic-model.md` 4.1절, 「FD 엔티티 2절」은 `domain-entities.md` 2절이다.

- **작성 시각**: 2026-09-26T16:11:17Z 이후 (Functional Design 승인 뒤)
- **입력**: FD 셋 · 유닛 정의 `unit-of-work.md` 5절 · 10절 · 파일 행렬 (⑤ 열) · 팩 `scene-gates.md` 조각 0 · 7 ·
  `requirements.md` FR-7 · 5.3 · 5.4 · 5.6
- **NFR 두 단계** (NFR Requirements · NFR Design — 비기능 요구와 그 설계): 사용자 결정으로 건너뛰었다
  (2026-09-26T16:11:17Z). 유닛 정의가 NFR 에 적은 보안 · 성능을 이 계획의 3절이 받는다

---

## 1. 유닛 맥락

**하는 일** — 새 패키지 `internal/merge` 가 대기 upper 를 lower 에 합친다. 시작 전 확인(`Preflight`)과 합치기(`Apply`) 두
함수다. 부르는 쪽(merge 단계 · merge-helper · 시작 때 재개)은 bake 유닛이 짓는다 — 이 유닛은 그 둘을 부르는 자리를
만들지 않는다.

**완성하는 스토리** — 없다 (스토리 지도의 ⑤ 줄이 0 이다). **맡는 조각** — 7 의 기계 부분. 조각 7 전체(merge 단계를 SIGKILL
로 끊고 다른 노드가 시작 때 잇는다)는 bake 유닛이 맡는다.

**의존** — 없다 (의존 그래프에서 의존 0 인 유닛). **뒤 유닛이 기대는 것** — bake 유닛이 `Preflight` · `Apply` · `Options.Discard`
· `PreflightError.Check` · `Result` 를 쓴다 (FD 흐름 1절 · 7절).

**새 이름**

```text
   internal/merge (새 패키지)   Paths · Options · Preflight · Apply · Call (아홉 값 · String) · Op · Result ·
                               Check (다섯 값) · PreflightError · MarkError · OpError · ErrUnsupported
                               (안 내보내는 것) Kind · LowerKind · Marks · classify · decide · step · tally ·
                               checkOverlap · checkDevices · stashXattr
```

**경계** — `internal/merge` 는 표준 라이브러리와 `golang.org/x/sys` 만 쓴다. `internal/scratch` 도 안 쓴다 (FD 규칙 11절).
Mediator(`cmd/mediator`)는 `internal/merge` 를 안 가져다 쓴다. `go.mod` 를 안 움직인다 — `x/sys` 는 이미 있다.

**크로스 빌드** — 판정(`classify` · `decide` · 경로 겹침 · st_dev 판정)은 시스템 호출이 없어 모든 플랫폼에서 빌드된다.
걷기와 호출은 `merge_linux.go` 에만 있고, `merge_other.go` 가 `ErrUnsupported` 를 돌려준다 (`unit-of-work.md` 10절).
32비트 arm 의 `Timespec` · `Rdev` 타입 차이는 FD 엔티티 7절대로 옮긴다.

---

## 2. 고치는 파일

| 파일 | 새 · 고침 | 행렬 | 무엇 |
|---|---|---|---|
| `internal/merge/merge.go` | 새 | 있음 | 패키지 문서 · `Paths` · `Options` · `Call` · `Op` · `Result` · 오류 타입 · `stashXattr` |
| `internal/merge/decide.go` | 새 | 있음 | `Kind` · `LowerKind` · `Marks` · `classify` · `step` · `decide` · `checkOverlap` · `checkDevices` |
| `internal/merge/merge_linux.go` | 새 | 있음 | `Preflight` · `Apply` — 디렉터리 fd 로 걷는 호출 |
| `internal/merge/merge_other.go` | 새 | 있음 | `Preflight` · `Apply` 가 `ErrUnsupported` |
| `internal/merge/decide_test.go` | 새 | 시험 | 판정 표 시험 (모든 플랫폼) |
| `internal/merge/tree_linux_test.go` | 새 | 시험 | 가짜 트리 짓기 · 목록 읽기 · 모형 `view` |
| `internal/merge/merge_linux_test.go` | 새 | 시험 | 한 번에 · 재개 N · ctx · 오류 · Preflight · 경계 · 벤치마크 둘 |
| `internal/merge/merge_integration_test.go` | 새 | 시험 | `integration` 태그 · linux · CI 밖 — 진짜 overlay 로 (FD 흐름 4.2) |
| `internal/panel/boundary_test.go` | 고침 | 있음 | 금지 하나 · 봉인 하나 (FD 규칙 11절) |

그 밖의 파일을 만지지 않는다 (FD 흐름 6절). 파일을 나누는 모양은 FD 엔티티의 표와 같다 — 시험 파일만 셋으로 나눴다.

---

## 3. NFR 두 단계를 건너뛰어서 이 계획이 받는 것

유닛 정의가 NFR 에 적은 둘이다. 설계는 FD 가 닫았다. 여기는 **시험과 측정으로 확인**하는 자리다.

**보안 — 합치는 경로가 lower 밖으로 안 나간다 · 같은 filesystem** (팩 보안 표의 합치기 줄 · FD 규칙 3절 · 7절)

- 가짜 트리에 **lower 밖을 가리키는 symlink** 를 둔다 — lower 의 `escape -> <시험 임시 폴더의 밖 디렉터리>` 위에 upper 의
  디렉터리 `escape/` (파일 하나). 합친 뒤 밖 디렉터리가 그대로이고(항목 수 · 내용), lower 의 `escape` 가 디렉터리다
- upper 의 symlink 가 가리키는 곳이 바뀌지 않는다 (링크 자체가 옮겨진다)
- `Preflight` 표 — 겹침 셋 · st_dev 판정 (Step 8)

**성능 — 비용이 바뀐 항목 수를 따르고 byte 수와 무관하다** (`requirements.md` 5.4)

- **byte 와 무관** — 한 번에 시험이 새 파일 · 대체한 파일의 inode 가 upper 의 것인지 본다 (복사하지 않았다)
- **새 디렉터리는 한 번** — 파일 1,000 개를 담은 새 디렉터리가 `Ops` 를 1 (rename) 만 더한다
- **하루치 규모의 시간** — 벤치마크 둘을 이 기계에서 한 번 돌리고 숫자를 `code-summary.md` 에 적는다. 기본 `go test` 에서는
  안 돈다
  - `BenchmarkPreflight` — upper 150,000 항목 (디렉터리 100 x 파일 1,500) 을 걷는 시간. ADR-077 §12 의 1.00 초(147,893 파일)와 댄다
  - `BenchmarkApply` — 같은 모양의 lower 위에 파일 대체 52,900 · 새 파일 1,600 · 새 디렉터리 160 · whiteout 330 · opaque 200 ·
    양쪽 디렉터리 3,300 (§12 의 B 에 가깝게). §12 의 1.49 초와 댄다. `Ops` 는 호출마다라 §12 의 연산 수보다 크다 (FD 엔티티 2절)
- 값이 §12 보다 한 자리 이상 크면 원인을 찾아 `code-summary.md` 에 적는다. 값을 맞추려고 규칙을 바꾸지 않는다

---

## 4. 이 계획이 정한 것 — FD 에 적히지 않은 자리

**① 디렉터리가 아닌 항목의 표시를 읽는 경로.** `fmt.Sprintf("/proc/self/fd/%d/%s", parentFd, name)` 에 `unix.Llistxattr` · `unix.Lgetxattr`.
사용자 xattr 은 symlink · 장치 파일에 붙지 않으므로 목록이 비어 돌아온다 — 그것도 그대로 판정에 넘긴다 (`ENODATA` · `ENOTSUP` 는 없음).

**② 호출 자리의 함수 값.** `Apply` 의 호출(renameat · renameat2 · unlinkat · fchown · fchmod · utimensat · fsetxattr · fremovexattr)은 걷기 구조체의
함수 칸을 지난다. 기본값은 x/sys 의 그 함수다. 시험이 실패를 끼울 때만 바꾼다 — 가짜 트리로 못 밟는 오류 갈래를 덮으려는 것이다 (FD 흐름 5절).
칸을 바꾸는 시험이 없어도 80% 를 넘으면 칸을 두지 않는다 — Step 9 에서 정한다.

**③ 시험의 Discard.** 시험은 `internal/scratch` 를 쓰지 않는다. 시험 자신의 함수가 버릴 항목을 시험의 trash 폴더로 옮기고(이름은 순번)
그 inode 를 적는다. `scratch.Trash.Move` 로 잇는 것은 bake 유닛이다.

**④ 재개 시험의 끊기.** OnOp 가 k 번째에 시험의 오류 값(`errStop`)을 돌려준다. `Apply` 가 그 값을 그대로 돌려주는지(`errors.Is`) 본다.
k 는 1 부터 「한 번에」의 `Ops` 까지 모두다 — 가짜 트리가 바뀌면 N 도 바뀌고, 시험은 N 을 매번 센다.

**⑤ 모형의 목록.** 목록은 `entry` 의 조각을 경로순으로 정렬한 것이다. 디렉터리 크기 · atime · ctime 은 안 본다. uid · gid 는 본다 (기본
`go test` 에서는 모두 시험 사용자라 같다). 두 목록이 다르면 다른 줄만 실패 메시지에 싣는다 (영어).

**⑥ integration 시험의 모양.** 시험이 자기 바이너리를 `unshare --user --map-root-user --map-auto --mount` 뒤에서 다시 실행한다 (환경 변수
`ENODE_MERGE_IT_CHILD=1` 로 갈린다). 아이 쪽이 overlay 를 마운트하고 세션 동작을 한 뒤 목록을 적고, 마운트를 내리고 `Preflight` · `Apply` 를
부른다. `unshare` 가 uid 매핑을 못 하면 시험이 실패한다 — 건너뛰지 않는다. 폴더는 `t.TempDir()` 아래다 (버려도 되는 자리).

**⑦ 벤치마크의 트리.** `b.TempDir()` 에 짓고, 짓는 시간은 `b.StopTimer` 로 뺀다. `Apply` 벤치마크는 반복마다 트리를 새로 짓는다.

**⑧ 오류 문구의 꼴.** FD 규칙 10절 그대로다. `PreflightError.Error()` 는 `merge preflight: ` 뒤에 Check 마다의 문장, `MarkError.Error()` 는
`upper entry <rel> carries <name>` 꼴, Apply 가 돌려줄 때 `merge: ` 를 앞에 붙인다.

---

## 5. 단계 — 열둘

### Step 1 — 기준선

- [x] `unit/merge-rules` 가 `f155faf` 위에 있다
- [x] `golangci-lint run ./...` 경고 수를 적는다 · `internal/panel` 커버리지를 적는다 (CI 의 측정 명령과 awk)
      — 린트 38 건 (errcheck 24 · staticcheck 10 · govet 2 · ineffassign 1 · unused 1) · `internal/panel` 89.1% · x/sys v0.47.0

### Step 2 — 겉면 타입 (FD 엔티티 1 · 2 · 4 · 5절)

- [x] `merge.go` — 패키지 문서 주석 (담는 것 · 모르는 것 — 상태 · 잠금 · 광고 · namespace · Run · trash 의 이름 규칙 / 봉인 / Mediator 가 링크하지 않는다)
- [x] `Paths` · `Options` · `Call` 아홉과 `String` · `Op` · `Result` (일곱 칸 · `Ops` · `Discarded`) · `tally`
- [x] `Check` 다섯 · `PreflightError` · `MarkError` · `OpError` (셋 다 `Error` · `Unwrap`) · `ErrUnsupported` · `stashXattr = "user.enode.merge-mtime"`

### Step 3 — 판정 (FD 엔티티 3절 · FD 규칙 1 · 2절)

- [x] `decide.go` — `classify(mode, rdev, marks, root)` — 금지 표시 다섯과 뿌리의 opaque 는 `*MarkError`. 문자 장치 0/0 만 whiteout
- [x] `decide(kind, lowerKind) step` — FD 규칙 1절의 열 줄. 양쪽 디렉터리는 「안으로」
- [x] `lowerKindOf(mode)` — lstat 의 종류로. symlink 는 `LowerOther`
- [x] `checkOverlap(upper, lower, trash)` — 경로 조각 단위 앞부분 비교 · 같음도 겹침 · `checkDevices(u, l, t)` — 셋이 같지 않으면 오류

### Step 4 — 판정 시험 (모든 플랫폼)

- [x] `decide_test.go` — `classify` 표 (종류 넷 · 금지 다섯 · 뿌리 opaque · opaque 값 `x` · 디렉터리가 아닌 항목의 opaque · 0/0 이 아닌 문자 장치는
      `KindOther` · escape 한 이름은 Marks 에 안 들어온다) · `decide` 표 (종류 넷 x lower 셋 = 열둘 칸이 FD 규칙 1절의 열 줄로) ·
      `checkOverlap` 표 (같음 · 안 · 밖 · 이름이 앞부분만 같은 형제 `/a/up` 과 `/a/upper`) · `checkDevices` · `Call.String` · 오류 문구 셋

### Step 5 — `Preflight` (FD 규칙 3절 · FD 흐름 2절)

- [x] `merge_linux.go` — 절대 경로 · `EvalSymlinks` · lstat 으로 진짜 디렉터리 · `checkOverlap` · `checkDevices` · upper 를 뿌리부터 fd 로 걸으며 표시를 읽어
      `classify` (첫 금지에서 `*PreflightError{Check: mark}`). 아무것도 안 바꾼다
- [x] 표시 읽기 — 디렉터리는 연 fd 로 `Flistxattr` · `Fgetxattr`, 아닌 항목은 4절 ① 의 경로로. 이름은 정확히 맞춘다

### Step 6 — `Apply` (FD 규칙 1 · 4 · 5 · 6 · 8 · 9절 · FD 흐름 2절)

- [x] `Discard` 가 없으면 오류 (아무것도 안 바꿈) · 뿌리 둘을 `O_NOFOLLOW | O_DIRECTORY` 로
- [x] `walk` — 이름을 바이트 순으로 · 항목마다 `classify` 와 `lowerKindOf` · `decide` 의 차례대로 호출 · 뿌리는 속성 · stash · rmdir 이 없다
- [x] 옮기기 — 파일 · symlink 는 `Renameat`, 디렉터리는 `Renameat2(RENAME_NOREPLACE)`
- [x] `mergeDir` — stash 읽기 (없으면 fstat 의 mtime 을 `Fsetxattr` 로 적는다) · 안으로 · `Fchown` · `Fchmod(mode & 07777)` · `UtimesNanoAt` (atime 은
      `UTIME_OMIT`) · `Unlinkat(AT_REMOVEDIR)`
- [x] `call` — 성공하면 `Ops++` · OnOp · ctx 확인. 실패는 `*OpError`, OnOp 의 오류와 ctx 는 그대로. 되돌리는 defer 를 두지 않는다 (fd 닫기만)
- [x] 끝나면 upper 뿌리가 비었는지 본다
- [x] `merge_other.go` — 두 함수가 `ErrUnsupported`

### Step 7 — 가짜 트리와 모형 (FD 흐름 4.1 · FD 엔티티 6절 · 4절 ③ ⑤)

- [x] `tree_linux_test.go` — 가짜 트리 짓기 (FD 흐름 4.1 의 표 그대로 · 3절의 `escape` · 디렉터리마다 다른 mtime) — 표시를 못 만들면
      `t.Fatal` (건너뛰지 않는다)
- [x] 트리 읽기 — 뿌리 아래의 항목과 표시를 합치기 전에 읽어 둔다 · 목록 `entry` 읽기
- [x] 모형 `view(lower, upper)` — 경로마다 「누가 보이나」로. `decide` 를 부르지 않는다
- [x] 시험의 Discard — 시험 trash 로 옮기고 inode 를 적는다 (4절 ③)

### Step 8 — 규칙 시험 (FD 흐름 4.1 의 시험 여섯 · 3절)

- [x] 한 번에 — lower 목록 = 모형 · upper 뿌리가 비었다 · 버린 inode 모임 = 모형의 가려진 lower · 새 파일과 대체한 파일의 inode 가 upper 의 것 ·
      Result 칸 · 밖 디렉터리가 그대로 (3절 보안) · escape 한 xattr 과 origin 이 옮겨졌다 · 하드 링크 둘이 한 inode · 디렉터리 mtime 이 모형대로
- [x] 재개 — k = 1 .. N 모두 (4절 ④) — 같은 lower 목록 (디렉터리 mtime 포함) · 같은 버린 모임 · 빈 upper · 끊긴 자리의 오류가 `errors.Is(errStop)`
- [x] ctx — k 번째에 cancel → `context.Canceled` → 다시 → 같다
- [x] 오류 — Discard 가 처음 한 번 실패 → `*OpError` (Call 이 discard · 경로) → 다시 → 같다
- [x] Discard 없음 — 오류이고 트리가 그대로다
- [x] 새 디렉터리 한 번 — 파일 1,000 개의 새 디렉터리가 `Ops` 를 1 더한다 (3절 성능)
- [x] Preflight — 통과하는 트리 (origin · escape 한 이름 · 0/0 whiteout) · 상대 경로 · 없는 trash · 파일인 lower · 겹침 · 금지 다섯 · 뿌리 opaque ·
      Preflight 뒤 트리가 그대로 (목록이 같다 · atime 은 안 본다) · symlink 로 준 뿌리가 풀린다
- [x] Apply 가 걷다 금지 표시를 만나면 그 항목 전에 멈춘다 (Preflight 를 건너뛴 경우)

### Step 9 — 커버리지와 벤치마크 (FD 흐름 5절 · 3절 · 4절 ② ⑦)

- [x] CI 의 측정 명령으로 `internal/merge` 가 80% 이상인지. 모자라면 4절 ② 의 함수 칸으로 오류 갈래를 덮는다
      — 89.1% (303/340). 함수 칸은 두지 않았다
- [x] `BenchmarkPreflight` · `BenchmarkApply` 를 한 번 돌리고 숫자를 적는다 — 이 기계 0.63 초 · 3.80 초, SunnyVM 0.23 초 · 1.11 초
      (code-summary 6절)

### Step 10 — integration 시험 (FD 흐름 4.2 · 4절 ⑥)

- [x] `merge_integration_test.go` (`//go:build integration && linux`) — 진짜 표시 · 0555 와 000 디렉터리 · 세 목록 (커널 merged view = 모형 = 합친 lower) ·
      재개 k = 1 .. N · 합친 lower 위의 다음 overlay (목록이 같다 · 덧붙이기가 된다) · 권한 비트가 안 바뀌었다
- [x] `go vet -tags integration ./internal/merge/` 로 컴파일을 본다
- [x] SunnyVM 이 켜져 있으면 에이전트가 거기서 돌리고 결과를 적는다. 꺼져 있으면 보류로 적는다 — 병합 조건이 아니다 (병합 조건은 기본
      `go test` 의 재개 시험) — 2026-09-26 SunnyVM (커널 7.0) 에서 초록 · 재개 44 자리

### Step 11 — 경계 시험과 코드 검사 (FD 규칙 11절 · 조각 0 · 조각 7 기계 부분)

- [x] `internal/panel/boundary_test.go` — `{"cmd/mediator", "internal/merge"}` · 봉인에 `{"internal/merge", []string{"golang.org/x/sys/"}}`
- [x] `gofmt -l .` · `go vet ./...` · `go build ./...` · U+2605 0 · `go run ./scripts/glyphscan.go`
- [x] `go test ./... -count=1` — 실패 0 · 스킵 감시 통과 · 패키지마다 80% 이상 (`internal/merge` 포함) — 통과 2,178 · 실패 0 · 스킵 0 ·
      전체 86.4%
- [x] 크로스 빌드 셋 (windows/amd64 · linux/arm GOARM=7 · darwin/arm64) · 린트 수가 Step 1 보다 늘지 않는다 — 린트 38 그대로
- [x] **조각 0** — build · vet · test
- [x] **조각 7 의 기계 부분** — 재개 시험이 기본 `go test` 에서 초록 (Step 8)
- [x] `git status` 로 2절 밖의 파일이 없는지 — 시험이 바꾼 `cmd/enodectl/probe.lock` 을 되돌렸다

### Step 12 — 요약 · 상태 · 감사 · 커밋

- [x] `construction/merge-rules/code/code-summary.md` — 파일 · 규칙의 자리 · 4절의 결정 · 계획과 다른 자리 · 코드 검사 숫자 · 벤치마크 숫자 ·
      integration 결과 · 넘기는 것 (FD 흐름 7절) · 정본 되돌림 (FD 흐름 8절)
- [x] 표기 검사 · 사용자가 싫어한 말투 · 사내 이름
- [x] 이 계획의 체크박스 · `aidlc-state.md` 의 U5 절 · `audit.md`
- [x] 한 커밋 — 코드 · 시험 · 이 계획 · code-summary · 상태 · 감사. 승인 뒤에 넣는다 (CONVENTIONS 3.3)

---

## 6. 이 단계가 하지 않는 것

```text
   merge 단계 · merge-helper 입구 · 시작 때 재개 · Discard 를 Trash.Move 로 잇기      bake 유닛
   Preflight 가 어긋났을 때의 상태 되돌림 · 되풀이되는 오류의 처리                     bake 유닛
   lower 잠금 · 상태 파일 · 「마운트 0 의 증거」                                    lower-state 유닛
   .enode-metadata.json 쓰기                                                   bake 유닛 (FR-9)
   Mediator                                                                   바뀌지 않는다
   조각 7 전체 (merge 단계를 SIGKILL 로 끊고 다른 노드가 잇는다)                      bake 유닛 · 사람 · SunnyVM
   정본(enode-design) 되돌림                                                     진행자가 올린다 (FD 흐름 8절)
   PR 과 병합                                                                  코드 검사와 재개 시험이 초록인 뒤 · CI 뒤.  올리기 전에 묻는다
```

---

## 7. 확장 준수 — Code Generation

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 합치기 줄은 FD 규칙 3절 · 7절이 닫고 3절의 시험이 확인한다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 표 시험과 가짜 트리 재개 시험 |
