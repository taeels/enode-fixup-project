# U7 `chunk-push` — Code Generation 계획 (Part 1)

```text
   유닛    chunk-push (U7) · 브랜치 unit/chunk-push
   설계    ../chunk-push/functional-design/  (domain-entities · business-logic-model ·
          business-rules).  불변식 스물여섯을 R1 ~ R26 으로 센다
   답      여덟이 전부 A
   기준선   05ee710
   지는 게이트  **CB3** (올라간다) — 사람이 실 함대에서 본다
```

---

## 0. NFR 스킵이 이 계획에 넘긴 값 — 설계가 이미 졌다

U5 와 다르다. **값 둘을 Functional Design 이 이미 정했으므로** 이 계획은 그것을
옮겨 적기만 한다.

```text
   버퍼 상한 1 MiB    domain-entities 6절 · R19.  CB3 의 10초 정지가 밟는다
   오프셋의 자리      보내는 고루틴의 지역 변수 · R9
```

**남은 값이 하나 있다 — 재전송 간격이다.** 실패했을 때 다음 시도까지 얼마나
기다리나. 설계가 「다음 주기에 새 바이트와 함께 간다」(R11)로만 적었다.

```text
   고른 값   주기 그대로 2초.  따로 백오프를 안 둔다
   왜       버퍼가 1 MiB 라 재전송이 오래 못 간다 — 넘치면 R14 가 멈춘다.
            백오프를 두면 「멈추기까지의 시간」이 늘어 CB3 의 10초 창을 오히려
            좁힌다.  그리고 실패 로그는 연속의 첫 번째만 적으므로 (R 5절)
            2초 간격이 로그를 안 더럽힌다
   대가     Mediator 가 오래 죽어 있으면 2초마다 헛왕복이 돈다.  버퍼가 찰 때까지만이다
```

---

## 1. 실측이 계획 단계에서 찾은 것 — 둘

### 1.1 `step` 이 두 자리 다 스코프에 있다 — 시그니처를 넓히면 닿는다

```go
   claim.go:441   func (w *Worker) execute(ctx context.Context, step *Step)
   claim.go:632     sink = io.MultiWriter(&buf, w.ring)   // 명령.  step 있음
   claim.go:794     Transcript: w.transcript(),           // 에이전트.  step 있음
```

**둘 다 `execute` 안이라 `step` 이 그대로 있다.** `runID` · `seq` · `name` ·
`attempt` 넷을 따로 나를 필요가 0 이다.

### 1.2 `Client` 는 `advertise.go` 에 산다 — 파일 행렬이 claim.go 로 적었다

```text
   행렬이 준 것   internal/enode/claim.go · internal/enode/upload.go (신규)
   실제           Client 구조체는 advertise.go:17.  UploadLog · PutBlob 은 claim.go
```

**메서드를 `upload.go` 에 두면 파일이 안 는다** — `Client` 는 같은 패키지다.
`PutLogChunk` 를 `upload.go` 에 놓아 **행렬이 준 파일 둘만 만진다.**

### 1.3 시험의 모양이 이미 있다 — `httptest` + 가짜 `Client`

```go
   claim_test.go:21   srv := httptest.NewServer(...)
                      w := &Worker{Client: &Client{Base: srv.URL, HTTP: srv.Client()}, ...}
```

**같은 모양으로 업로더를 잰다.** 서버를 죽였다 살리는 것도, 상한을 흉내 내는
것도 이 핸들러 안에서 한다 — 실 Mediator 가 필요 없다.

---

## 2. 이 유닛이 내는 diff 의 모양

```text
   internal/api/api.go              attempt 검증 한 줄 (R6).  **행렬 밖** (6절)
   internal/enode/upload.go (신규)   uploader · Client.PutLogChunk
   internal/enode/claim.go          transcript() 의 시그니처 · 배선 둘

   시험
   internal/api/getlog_test.go      attempt=0 을 밟는 줄 (저장소에 0 이었다)
   internal/enode/upload_test.go (신규)  업로더의 규율 전부
```

```text
   diff 0 이어야 하는 곳   internal/record/**  ·  internal/transcript/**  ·
                        internal/panel/**  ·  internal/store/**  ·
                        go.mod  ·  go.sum  ·  UploadLog 의 동작
   라우트                internal/api/api.go 의 18 그대로 (R25 · CB0 을 안 건드린다)
```

---

## 3. 못 박는 겉면 — Step 1 이 먼저 한다

```go
   // internal/enode/upload.go
   type LogChunkAck struct {
       Total   int64
       Attempt int
       Capped  bool
   }
   func (c *Client) PutLogChunk(ctx context.Context, runID string, seq int,
       name string, attempt int, body []byte) (LogChunkAck, error)

   type uploader struct { ... }          // 3절 domain-entities
   func newUploader(cl *Client, step *Step, log *slog.Logger) *uploader
   func (u *uploader) Write(p []byte) (int, error)   // 절대로 안 막힌다
   func (u *uploader) Close()                        // 마지막으로 비우고 멈춘다

   // internal/enode/claim.go
   func (w *Worker) transcript(step *Step) (io.Writer, func())
```

**`Close()` 가 `error` 를 안 내는 이유** — 부르는 쪽이 검사할 것이 없다. 실패는
전부 노드 로그로 가고 실행을 안 막는다 (R1). W-b 의 U2 가 `Close() int64` 로 같은
판단을 했다 — **언제나 nil 인 error 를 돌려주면 부르는 쪽이 없는 값을 검사한다.**

---

## 4. 단계 — 열둘

### Step 1 — 겉면을 못 박는다
- [x] `LogChunkAck` · `PutLogChunk` 시그니처 (몸통은 Step 3)
- [x] `uploader` 와 `newUploader` · `Write` · `Close` 시그니처
- [x] `transcript(step)` 의 새 시그니처
- [x] `go build ./...` 가 선다

### Step 2 — `api.go` 의 `attempt` 검증 (R6) — **착수 조건이다**
- [x] `attempt <= 0` 을 `attempt < 0` 으로. 음수는 여전히 400
- [x] 주석에 왜인지 적는다 — `seq` 는 1 부터이고 `attempt` 는 0 부터다
- [x] 시험: `attempt=0` 이 200 이고 진행 파일이 생긴다 (**저장소에 0 이던 줄**)
- [x] 시험: `attempt=-1` 은 그대로 400
- [x] 기존 진행 청크 시험 다섯이 그대로 초록

### Step 3 — `Client.PutLogChunk` (R5 · R10)
- [x] `PUT .../log?name=<n>&progress=1&attempt=<a>` · `text/plain`
- [x] 응답 헤더 셋을 다 읽어 `LogChunkAck` 로 낸다
- [x] 2xx 가 아니면 오류. 몸통의 `reason` 을 싣는다
- [x] `UploadLog` 를 **한 글자도 안 고친다** (R23)

### Step 4 — `uploader` 의 `Write` (R1 · R2 · R3)
- [x] 버퍼에 넣고 **바로 돌아온다.** 네트워크를 0 번 기다린다
- [x] 언제나 `len(p), nil`. 오류를 0 번 낸다
- [x] 1 MiB 를 넘으면 `done` 을 세우고 더 안 받는다. **그래도 바로 돌아온다**
- [x] 64 KiB 가 쌓이면 보내는 고루틴을 깨운다 (막히지 않게 — non-blocking)

### Step 5 — 보내는 고루틴 (R9 · R11)
- [x] 2초 주기. 즉시 문턱으로도 깨어난다
- [x] 버퍼를 통째로 떼어 온다. 비었으면 다음 주기로
- [x] 실패하면 **뗀 것을 버퍼 앞에 도로 붙인다** (R11)
- [x] 오프셋은 **이 고루틴의 지역 변수**다. 구조체 필드로 안 둔다 (R9)
- [x] `recover` 로 패닉을 잡는다. 노드를 안 죽인다 (R4)

### Step 6 — 응답 셋을 읽는 순서 (R7 · R12)
- [x] `Attempt` 를 `Total` 보다 먼저 본다 — **실측으로 이 순서가 공허함을
      확인했다** (변이 ⑤).  코드는 그대로 두되 R8 을 걷었다
- [x] `Attempt` 가 내 것보다 크면 **멈춘다.** 오프셋을 안 옮긴다 (R7)
- [x] `Capped` 면 `done` 을 세우고 멈춘다. **실행은 계속된다** (R12)
- [x] 그 밖이면 오프셋을 `Total` 로 맞춘다 (R10)

### Step 7 — 실패 로그의 규율 (R 5절 · 답 8 = A)
- [x] 연속 실패의 **첫 번째만** 적는다
- [x] 회복한 첫 번째를 적는다 — 몇 번 실패하고 몇 초 만인지
- [x] 버퍼가 넘쳐 멈춘 것을 한 줄로 적는다 (R14 — 중앙은 이것을 못 안다)
- [x] 상한에 닿아 멈춘 것을 한 줄로 적는다 (R12)

### Step 8 — `transcript(step)` 와 배선 둘 (R18)
- [x] 링이 nil 이어도 업로더는 돈다. 업로더가 nil 이어도 링은 돈다
- [x] 에이전트 단계 (`claim.go:794`) 에 잇는다
- [x] 명령 단계 (`claim.go:632`) 에 잇는다 — **규칙이 하나다**
- [x] 정리를 `defer` 로 든다. **`UploadLog` 보다 먼저 비운다** (R17)

### Step 9 — 시험: 올라간다
- [x] 도는 동안 청크가 실제로 PUT 된다. 쿼리 셋이 맞다
- [x] 두 번 부르면 서버의 총 길이가 **늘어 있다** (CB3 의 첫 줄이다)
- [x] 명령 단계도 올라간다 (R18)
- [x] 단계 끝에 꼬리가 비워진 뒤 `UploadLog` 가 간다 (R17 — 순서를 잰다)

### Step 10 — 시험: 안 막는다 · 안 잃는다
- [x] 서버를 죽여 놓고 `Write` 의 시간을 잰다 — **막히지 않는다** (R2)
- [x] 서버를 10초 죽였다 살리면 **다음 청크가 이어 붙고 두 벌이 안 생긴다** (CB3)
- [x] 버퍼가 넘치면 멈추고 `Write` 는 여전히 바로 돌아온다 (R3 · R14)
- [x] 넘친 뒤 서버가 살아나도 **재개하지 않는다** (R15)

### Step 11 — 시험: 시도와 상한
- [x] `attempt 0` 으로 보낸다 (R5) — `+1` 을 안 한다
- [x] 응답의 `Attempt` 가 크면 멈춘다 (R7)
- [x] `Capped` 면 멈추고 **실행은 계속된다** (R12)
- [x] 상한 뒤로 왕복이 0 이다 (전역 한도를 안 갉는다)

### Step 12 — 변이와 게이트
- [x] 변이 일곱 (5절)
- [x] `eval "$(scripts/testdb.sh)"` 뒤 CP0 계열 전부
- [x] 라우트 18 그대로 · `diff 0` 이어야 할 곳 전부 0
- [x] 코드 요약을 쓰고 이 계획의 체크박스를 **실측으로** 채운다

---

## 5. 변이 일곱

```text
   ①  attempt 검증을 <= 0 으로 되돌린다        FAIL (시험 넷)
   ②  노드가 attempt+1 로 보낸다              FAIL (시험 여덟)
   ③  Write 에서 동기로 보낸다                 FAIL (시험 여섯)
   ④  실패한 청크를 도로 안 붙인다              FAIL (시험 일곱)
   ⑤  Total 을 Attempt 보다 먼저 본다          **살아남았다.  안 고쳤다**
   ⑥  버퍼 상한을 뺀다                        FAIL (시험 넷)
   ⑦  Close 를 UploadLog 뒤로                 **처음에 살아남았다** -> FAIL
```

**⑤ 는 규칙이 공허했다.** `ack.Attempt` 가 크면 **멈추므로** 그 뒤에 오프셋을
읽는 자리가 0 이다 — 멈추는 것이 순서를 삼킨다. **시험을 지어 억지로 빨갛게
만들지 않았다.** 그러면 코드가 아니라 시험이 규칙을 만든다.
`business-rules` 의 R8 을 걷고 그 자리에 이 사실을 적었다.

**⑦ 은 Step 9 의 순서 줄을 안 지어 살아남았다.** 지으면서 **가짜 Mediator 가
진행 청크와 선별본을 한 자리에 덮고 있던 것**도 드러났다 — 그러면 「두 벌이
안 생긴다」가 「한 벌만 생긴다」로 조용히 바뀐다. 갈라서 순서를 기록하게 했다.


**③ 과 ⑥ 이 W-b 의 U2 변이 ⑤ 와 같은 자리다** — 거기서 `Write` 가 영영 막혀
**실패가 아니라 600초 멈춤**으로 빨갰다. 업로더는 네트워크라 더 세다.
**시험에 타임아웃을 걸어 멈춤이 멈춤으로 보이게 한다** — 안 걸면 변이가
「빨강」이 아니라 「영영 안 끝남」이 되고, 그 둘은 읽는 사람에게 다르다.

---

## 6. 진행자에게 넘기는 것 — 셋

```text
   **api.go 의 attempt 검증**   이 유닛이 고친다.  **행렬 밖**이다 — 행렬은 그 파일을
                              U4 에만 줬고 U4 는 병합됐으므로 충돌 0.
                              W-b 가 같은 모양을 둘 겪었고 실제로 0 이었다

   unit-of-work.md U7 절       「tee 의 갈래 하나」가 실제로는 둘이다 (에이전트 · 명령)

   unit-of-work.md U7 절       「Client.PutLogChunk 하나」로 적었는데
                              transcript() 의 시그니처도 함께 넓어진다
```

---

## 7. 짓지 않는 것

```text
   현황판 카드        U8
   제어판            U5 · U6
   내려받는 쪽        U4 가 닫았다
   Mediator 쪽 저장   U3 가 닫았다
   새 라우트 · 설정 키  0.  버퍼 상한은 상수다 (조절 손잡이가 아니라 보호다)
   넘친 것을 중앙에    못 한다.  business-rules 8절 ① 의 잔여다
   알리는 길
```

---

## 8. 물음 — 0

설계의 답 여덟이 값을 다 정했고 0절이 재전송 간격 하나를 마저 졌다.
**새로 묻는 것이 없다.**
