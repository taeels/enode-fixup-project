# 파일 행렬 — 어느 유닛이 어느 파일을 만지나

**팩이 이것을 명시로 걸었다** (`constraints.md` 접점 절 — 「Units Generation 에
거는 요구 하나 — 파일 행렬을 반드시 낸다」).

한 손이라 사람 충돌은 0 이다. 그래도 낸다 — **유닛 둘 이상이 만지는 파일은
순서가 있고, 그 순서를 안 적으면 두 번째 유닛이 첫 번째의 변경 위에서 재작업한다.**

---

## 1. 행렬

| 파일 | U1 | U2 | U3 | U4 | U5 | U6 | U7 | U8 |
|---|---|---|---|---|---|---|---|---|
| `internal/transcript/**` (신규) | 신규 | | | | | | | |
| `internal/enode/runner.go` | **덜어낸다** | **tee** | | | | | | |
| `internal/enode/claude.go` | | Decode | | | | | | |
| `internal/enode/claim.go` | | `transcript()` | | | | | **업로더 배선** | |
| `internal/enode/upload.go` (신규) | | | | | | | 신규 | |
| `internal/record/record.go` | | | 진행 파일 · `AppendLog` · `Seal` | | | | | |
| `internal/store/seal.go` · `reap.go` | | | **수명 끊기 · 고아 쓸기** | | | | | |
| `internal/api/log.go` (신규) | | | | 신규 | | | | |
| `internal/api/api.go` | | | | 등록 줄 · `putLog` 갈림 | | | | |
| `internal/panel/transcript.go` | | | | | **도는 것** | **지난 것** | | |
| `internal/panel/page.go` | | | | | 카드 · 헤더 다섯 | | | |
| 경계 검사 시험 (6절) | **금지 넷 (+ 빈 넷째 줄)** | | | | | | | |
| `internal/runctl/client.go` | | | | | | `StepLog` | | |
| `internal/api/ui/static/shared/fleet/**` | | | | | | | | 단계 카드 |
| `internal/api/ui/tests/**` | | | | | | | | 시험 |

**시험 파일은 그 유닛이 만진다** — 위 표는 비테스트 파일만 이름으로 적고,
각 유닛의 `_test.go` 는 그 유닛 것이다. 예외가 하나 있어 표에 넣었다 —
경계 검사 시험이다. **그 파일의 자리를 여기서 못 박지 않았고 이유가 6절에 있다.**

---

## 2. 유닛 둘 이상이 만지는 파일 — 셋

**`CONVENTIONS.md` 3.1 이 표시하라고 한 자리다.**

| 파일 | 유닛 | 순서 | 왜 그 순서인가 |
|---|---|---|---|
| `internal/enode/runner.go` | U1 · U2 | **U1 -> U2** | U1 이 `parseEventLine` · `eventShell` · `elidedMarker` 셋을 덜어낸다. U2 가 그 뒤에 tee 를 잇는다. 뒤집으면 U1 의 이동이 U2 의 변경 위를 지나 충돌이 는다 |
| `internal/enode/claim.go` | U2 · U7 | **U2 -> U7** | U2 가 `transcript()` 를 링 하나로 되살리고 U7 이 거기에 `io.MultiWriter` 로 갈래를 더한다. 뒤집으면 U7 이 없는 링에 갈래를 더하게 된다 |
| `internal/panel/transcript.go` | U5 · U6 | **U5 -> U6** | U5 가 `handleTranscript` 를 사건 배열로 바꾸고 U6 이 `handleRecord` 의 출처를 바꾼다. 둘이 같은 파일의 다른 함수라 순서는 충돌이 아니라 **CB1 을 먼저 본다**는 값이다 |

**셋 다 착수 순서(`unit-of-work-dependency.md` 3절)가 이미 그 순서다.**
①U1 ②U2 ③U5 ④U3 ⑤U4 ⑥U6 ⑦U7 ⑧U8 — 표의 화살표를 거스르는 자리가 0 이다.

---

## 3. `CONVENTIONS.md` 가 경고한 자리가 안 부딪친다

```text
   규약이 적은 것   「internal/api/api.go 의 등록 줄처럼 유닛 둘 이상이 만지는 파일은
                   파일 행렬에 표시하고 진행자가 직렬로 병합한다.  한 번에 하나다」
   이 회차의 값     **U4 하나뿐이다.**  라우트를 더하는 것도 putLog 를 가르는 것도
                   같은 유닛이라 직렬이 실제로 안 생긴다
```

**대신 실제로 직렬인 자리가 다른 데 셋 있었다** (2절). 규약이 가리킨 파일이
아니라 `internal/enode` 둘과 `internal/panel` 하나다. **행렬을 안 냈으면 그
셋을 못 봤다** — 팩이 이것을 필수로 건 이유가 이 자리다.

---

## 4. 짝 팩과의 접점 — 0

`constraints.md` 의 접점 절이 팩 둘이 같은 파일을 만진다고 적었다.
**그 접점이 지금 0 이다.**

| 접점 절이 적은 파일 | 오늘 |
|---|---|
| `internal/enode/claude.go` 의 `Argv` | **짝 팩이 이미 가져가 `main` 에 있다** (`claude.go:343`). 이 회차는 `Decode` 만 만진다 |
| `internal/enode/claude.go` 의 `Decode` | 이 팩만. 접점 아니다 |
| `internal/enode/runner.go` 의 tee · Emit | 이 팩만 |
| `internal/enode/claim.go` 의 `UploadLog` 호출 자리 | 이 팩만 |
| `internal/api/api.go` 의 등록 줄 | 이 팩만. 짝 팩은 등록 줄조차 안 늘렸다 |
| `internal/panel` · `internal/api/ui` | 이 팩만 |

**진행자가 직렬로 병합할 자리가 회차 사이에 0 이다.** 짝 팩의 유닛 다섯이 이미
`main` 에 들어왔다 (`aidlc-state.md` 의 기준선 `5716714`).

---

## 5. 안 만지는 경로

```text
   internal/contract   코드 diff 0.  계약 문법이 안 는다
   internal/match · proc · schema · config · build     코드 diff 0
   cmd/ 다섯           코드 diff 0.  새 실행파일도 새 하위명령도 0
   internal/api/ui/ui.go   Go diff 0.  U8 은 정적 파일과 시험만 만진다
```

### 5.1 `internal/store` 가 0 에서 빠졌다 — U3 의 Functional Design 이 고쳤다

**앞 판이 「`internal/store` 코드 diff 0. 그 코드는 `internal/record` 에 산다」로
적었고 그것이 거짓이 됐다.** U3 의 Functional Design 이 N2 를 지면서 진행 파일의
수명을 끊는 자리를 셋으로 갈랐는데, **그중 둘이 Run 상태를 알아야 하고
`internal/record` 는 `internal/store` 를 임포트하지 않는다.**

```text
   종료 상태로 끊기    internal/store/seal.go 의 sealRecord
   고아 쓸기          internal/store/reap.go
   봉인으로 끊기       internal/record.  DropProgress.  여기는 그대로다
```

**이것은 게이트가 빨개진 것이 아니라 기준선이 바뀐 것이다.** 사용자가 대가를
보고 골랐다 (U3 의 물음 5 = A · 2026-09-15). `internal/contract` 와 `cmd/` 다섯은
**0 그대로다** — U3 의 물음 10 = B 가 `ReadProgress` 의 겉면을 안 바꿔
`record.New` 도 `cmd/mediator/main.go` 도 안 바뀐다.

**`internal/contract` 의 diff 가 0 인 것이 품질 게이트 2 로 남는다**
(`execution-plan.md` 6절).

---

## 6. 경계 검사 — 이 단계가 실측에서 찾은 것 둘

`requirements.md` 5.2 가 「앞 팩의 넷에 둘을 더한다 · **경계 검사 테스트의 표에
두 줄이 는다. 검사기는 표를 읽는다**」고 적었다. 그 검사기를 열어 봤다.

### 6.1 넷 중 셋만 검사기에 있다

`internal/panel/boundary_test.go` 가 저장소의 **유일한** 경계 검사다
(`grep -rln "must not import" --include=*_test.go` 가 그 파일 하나를 낸다).

```text
   panel  -> store     검사한다   boundary_test.go:31 의 목록
   panel  -> api       검사한다   같은 목록
   enode  -> panel     검사한다   boundary_test.go:37
   api/ui -> store     **검사가 없다**
```

**`requirements.md` 5.2 의 넷째 줄은 규칙으로만 있고 검사기에 없다.**
이 회차가 만든 결함은 아니다 — 다만 「두 줄이 는다」를 그대로 믿고 두 줄만 더하면
그 자리가 계속 비어 있다.

### 6.2 「표를 읽는다」가 반쯤만 참이다

검사기는 `panel` 의 금지만 슬라이스로 돌고 (`[]string{"internal/store",
"internal/api"}`) 나머지는 `if` 를 하나씩 쓴다. **표가 아니라 목록 하나와
if 둘이다.** `internal/transcript` 의 금지 넷을 그 파일에 넣으면 `package
panel_test` 가 남의 패키지 경계를 검사하게 된다.

### 6.3 U1 에 거는 것

```text
   한다      transcript 의 금지 넷을 검사기에 넣는다.  자리는 Code Generation 이
             정한다 — 오늘 파일에 넣을지 internal/transcript 쪽에 새로 낼지
   같이 본다  api/ui -> store 의 빈자리.  두 줄 더하는 김에 넷째 줄도 선다
   안 한다    검사기를 두 벌로 만드는 것.  CONVENTIONS 1.4 의 같은 결이다
```

`go list -deps` 가 **전이 의존**을 보므로 `internal/transcript` 가 표준
라이브러리만 쓰면 넷이 자동으로 초록이다. 값은 **나중에 누가 임포트를 더했을 때**
빨개지는 데 있다.

### 6.4 그 명령이 시험을 안 본다 — U1 의 Functional Design 이 실측했다

**`go list -deps` 와 `go list -test -deps` 를 떼어 돌린 값이다.** 앞의 것은
시험 임포트를 세지 않는다.

```text
   go list -deps         제품 코드의 전이 의존만
   go list -test -deps   _test.go 의 임포트까지
```

**그래서 금지를 `go list -deps` 로만 세우면 `internal/transcript/*_test.go` 가
`internal/enode` 를 임포트해도 초록이다.** 6.3 이 「나중에 누가 임포트를 더했을 때
빨개지는 데 값이 있다」고 적었는데, **그 「나중」의 절반이 오늘 안 걸린다.**

이 회차가 만든 결함은 아니다 — 오늘 검사기도 같다. **U1 의 Code Generation 이
어느 명령으로 세울지를 정하고 그 판단을 `code-summary.md` 에 적는다.**
