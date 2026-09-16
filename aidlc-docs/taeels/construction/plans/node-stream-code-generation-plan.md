# U2 `node-stream` — Code Generation 계획 (Part 1)

```text
   유닛     node-stream (U2) · 웨이브 W-b · 브랜치 unit/node-stream
   앞 단계   Functional Design 승인 2026-09-16T02:05:00Z.  답 일곱 전부 A
   NFR      **스킵** — 회차 계획이 U2 를 원래 스킵으로 걸었다 (새 외부 표면 0)
   기준선    5d53e52 + b115b25.  라우트 17 · internal/enode 커버리지 86.7%
```

**이 계획은 Part 1 이다.** 승인이 오면 Part 2 가 이 Step 을 돌며 체크박스를 채운다.

---

## 0. 앞 단계가 이 계획에 넘긴 것 — 셋

```text
   ①  배출은 한 번만 난다 (R4).  runner 가 Decode 에 넘기는 emit 이 no-op 이다.
      **이것이 구현의 축이다** — 빠뜨리면 U7 이 같은 바이트를 두 번 올린다
   ②  배출기는 쓰기 경로에서 줄만 가른다.  파싱은 자기 고루틴이고 채널이 차면
      버린다 (R7).  채널 깊이는 값이고 이 계획이 고른다
   ③  왕복 시험 하나가 이 유닛의 몫이다 (R12) — 링이 감긴 뒤 U1 의 파서가 읽는다
```

---

## 1. 이 유닛이 내는 diff 의 모양

| 파일 | 새것/고침 | 무엇 |
|---|---|---|
| `internal/enode/emit.go` | **신규** | `lineEmitter` · `newLineEmitter` · `Write` · `Close` |
| `internal/enode/harness.go` | 고침 | `EventKind` 를 `transcript.Kind` 별칭으로 |
| `internal/enode/runner.go` | 고침 | tee 갈래 셋 · `Close` 가 `Decode` 앞 · no-op emit |
| `internal/enode/claude.go` | 고침 | `Decode` 를 줄 단위로 |
| `internal/enode/emit_test.go` | **신규** | 배출기의 규칙 |
| `internal/enode/ring_roundtrip_test.go` | **신규** | R12 의 왕복 |
| `internal/enode/claude_test.go` | 고침 | `Decode` 의 배출 |

```text
   go.mod · go.sum   diff 0.  새 의존 0
   claim.go          diff 0 (R15).  Transcript 도 Emit 도 이미 넘어온다
   internal/transcript  diff 0 (R14).  임포트만 한다
   U4 와의 파일 교집합   **0**
```

**`harness.go` 가 파일 행렬에 없다.** 행렬은 U2 에 `runner.go` · `claude.go` ·
`claim.go` 를 줬는데 `EventKind` 와 `Event` 가 `harness.go` 에 산다
(`:285` · `:291`). **행렬이 틀린 자리로 적는다** (6절).

---

## 2. 못 박는 겉면 — Step 1 이 먼저 한다

```go
// harness.go
type EventKind = transcript.Kind
const EventFinal EventKind = "final"

// emit.go
type lineEmitter struct { ... }
func newLineEmitter(emit func(Event)) *lineEmitter
func (e *lineEmitter) Write(p []byte) (int, error)
func (e *lineEmitter) Close() error
```

**겉면을 먼저 못 박는 이유** — Step 이 병렬로 안 돌아도 시험과 제품이 같은
글자를 봐야 한다. 나중에 정하면 시험이 제품을 따라가고, 그 순간 시험이 규칙을
안 재고 구현을 왜곡해 받아 적는다.

**채널 깊이 = 256.** 근거는 산수 하나다 — `claude` 의 stream-json 이 한 턴에
내는 줄이 실측 수십이고, 256 이면 고루틴이 한 턴 뒤처져도 안 찬다. 값이 작아서
막는 것이 아니라 **차면 버리는 것이 설계**라 (R7) 이 값은 안전 여유일 뿐이다.

---

## 3. 단계 — 열둘

### Step 1 — 겉면을 못 박는다
- [x] `harness.go` 에 `EventKind = transcript.Kind` 별칭
- [x] `EventFinal` 상수를 남긴다
- [x] `emit.go` 에 `lineEmitter` 의 필드와 넷의 시그니처만 (몸통은 다음 Step)
- [x] `go build ./...` 가 선다

### Step 2 — `emit.go` 의 쓰기 경로
- [x] `Write` 가 미완성 꼬리를 이어 붙이고 개행마다 줄을 만든다
- [x] 줄을 채널에 민다. 차면 `dropped++` 하고 버린다 (R7)
- [x] **언제나 `(len(p), nil)`** (R6). `recover` 로 패닉까지 삼킨다
- [x] 빈 `p` 는 아무것도 안 한다

### Step 3 — `emit.go` 의 고루틴
- [x] 채널에서 줄을 꺼내 `transcript.Parse(line, false)` 를 부른다
- [x] 나온 사건마다 `emit(Event{Kind: e.Kind})` — **`Text` 를 안 채운다** (R5)
- [x] 채널이 닫히면 돌아간다
- [x] `Close` 가 미완성 꼬리를 마지막 줄로 밀고 · 채널을 닫고 · 기다린다 (R8)
- [x] `Close` 가 `dropped` 를 돌려준다. 부르는 쪽이 로그로 낸다

### Step 4 — `runner.go` 의 tee 갈래 셋
- [x] `cmd.Stdout = io.MultiWriter(...)` — `&stdout` · `j.Transcript` · 배출기 (R1 · R2)
- [x] `j.Transcript` 가 nil 이면 그 갈래만 뺀다
- [x] `runner.go:187` 의 ⑥ 주석을 **되살린 이유로 다시 쓴다** (지우지 않는다)

### Step 5 — `runner.go` 의 순서
- [x] 배출기를 `cmd.Run` 앞에 만든다. `emit` 은 `j.Emit`
- [x] `cmd.Run` 뒤에 `Close` 를 부른다 — **`Decode` 앞이다** (R9)
- [x] 버린 줄이 0 이 아니면 `j.Log.Warn` 으로 수를 낸다
- [x] `h.Decode(..., code, noop)` — no-op 을 넘긴다 (R4)

### Step 6 — `claude.go` 의 `Decode`
- [x] `io.ReadAll` 을 걷고 `bufio` 로 줄 단위로 읽으며 바이트를 이어 담는다
- [x] 줄마다 `transcript.Parse` 로 읽고 사건마다 `emit`
- [x] 다 읽은 **전체** 바이트로 `ParseClaude(b, exitCode)` (R3)
- [x] `emit(Event{Kind: EventFinal, Text: h.Message})` — 오늘 그대로
- [x] 줄이 매우 길어도 안 깨진다 (`bufio.Scanner` 의 기본 상한을 안 쓴다)

### Step 7 — 시험: 배출기의 규칙
- [x] `Write` 가 오류를 안 낸다 — `emit` 이 패닉을 내도 (R6)
- [x] 채널이 차면 버리고 세고 단계는 안 죽는다 (R7)
- [x] `Close` 뒤에 나는 사건이 0 이다 (R8)
- [x] 개행 없이 끝난 꼬리가 `Close` 에서 마지막 줄이 된다
- [x] 사건의 `Text` 가 언제나 빈 문자열이다 (R5)

### Step 8 — 시험: 배출이 한 번이다 (R4)
- [x] `runHarness` 를 가짜 하네스로 돌려 **사건 수를 센다**
- [x] 줄 N 개를 흘리면 사건이 N 계열 + `final` 하나다. **두 배가 아니다**
- [x] `Decode` 를 직접 부르면 그때는 사건이 난다 (경로가 죽지 않았다)

### Step 9 — 시험: 어휘 (R10 · R11)
- [x] `EventKind` 가 `transcript.Kind` 와 같은 타입이다 (대입으로 고정)
- [x] `EventFinal` 의 값이 일곱 중 어느 것과도 안 같다 — 표로 돈다

### Step 10 — 시험: 왕복 (R12) — **이 유닛이 지는 이음매**
- [x] 링에 `DefaultTranscriptCapacity` 를 넘겨 쓴다 (감긴다)
- [x] `ReadRing` 으로 읽는다
- [x] `truncated = Snapshot.Total > uint64(len(Snapshot.Data))` 로 유도한다
- [x] `transcript.Parse(data, truncated)` 가 `Head > 0` 이고 `len(Events) > 0`
- [x] 안 감긴 경우도 함께 돈다 — `Head == 0`

### Step 11 — 변이: 시험이 실제로 재는지
- [x] `Close` 를 `Decode` 뒤로 옮긴다 -> Step 8 이 빨개져야 한다
- [x] no-op 대신 `j.Emit` 을 넘긴다 -> Step 8 이 빨개져야 한다
- [x] 배출기가 `Text` 를 채운다 -> Step 7 이 빨개져야 한다
- [x] `truncated` 를 언제나 false 로 -> Step 10 이 빨개져야 한다
- [x] 다섯째로 `Write` 가 오류를 내게 한다 -> Step 7 이 빨개져야 한다

### Step 12 — 게이트와 문서
- [x] `go build ./... && go vet ./... && gofmt -l .` 가 빈다
- [x] `go run ./scripts/glyphscan.go` 가 0 이다
- [x] 전체 시험이 초록이고 **스킵 0**
- [x] 커버리지 정본 명령으로 **미달 0**. `internal/enode` 가 80% 위다
- [x] 라우트가 **17 그대로** (R17)
- [x] `internal/transcript` 와 `claim.go` 의 diff 가 **0** (R14 · R15)
- [x] `git status --porcelain` — `probe.lock` 만 나오면 되돌린다
- [x] `construction/node-stream/code/code-summary.md` 를 쓴다. **잰 것만 적는다**
- [x] 이 계획의 체크박스를 **끝낸 그 자리에서** 채운다

---

## 4. 무엇으로 재는가

```text
   기계    go test ./... -count=1 -coverpkg=./... -coverprofile=/tmp/cover.out
          + ci.yml 의 awk (패키지별 하한 80)
          eval "$(scripts/testdb.sh)" 를 먼저 친다
   사람    없다.  이 유닛은 화면을 안 만든다 — CB1 은 U5 가 받는다
   이음매   Step 10 의 왕복.  링(U2)과 파서(U1)가 다른 패키지라 이것이 유일한 자리다
```

**이 유닛이 CB1 을 안 진다.** 재료를 낼 뿐이고, 재는 것은 U5 가 화면을 낸 뒤다.
**그래서 Step 10 이 이 유닛이 낼 수 있는 가장 가까운 증거다.**

---

## 5. 짓지 않는 것

```text
   업로더             U7.  transcript() 는 링 하나를 그대로 낸다 (답 4 = A)
   화면               U5 · U8
   봉투 파서           FR-1 이 명시로 뺐다
   stdout 버퍼의 상한   잔여 ① 로 이름만 적는다.  오늘도 전체를 든다
   설정 키            채널 깊이도 상한도 상수다.  키를 만들면 이름과 문서와
                     시험이 함께 생기고, 이 값들은 조절 손잡이가 아니다
```

---

## 6. 진행자에게 넘기는 것 — 다섯

```text
   unit-of-work-file-matrix.md 1절   internal/enode/harness.go 가 행렬에 없다.
                                    EventKind 와 Event 가 거기 산다
   unit-of-work-file-matrix.md 2절   claim.go 의 U2 칸이 빈다 — transcript() 는 U7 이 만진다
   unit-of-work.md U2 절            「Decode 도 파서를 안 쓴다」가 거짓이다
   component-methods.md 3절         Decode 안만 고치면 사건이 흐르는 시점이 안 바뀐다
   잔여 셋                          business-rules.md 6절 — stdout 을 통째로 든다 ·
                                    버린 줄은 로그에서 사라진다 · 줄 넘는 붙이기가 없다
```

---

## 7. 물음 — 0

**이 단계는 안 묻는다.** Functional Design 이 답 일곱으로 갈래를 다 닫았고,
남은 것은 값 둘(채널 깊이 256 · 변이 다섯)인데 둘 다 **되돌리는 값이 0 이라**
묻는 대신 적고 실측으로 고친다.
