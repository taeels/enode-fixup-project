# U6 `panel-past` — Code Generation 계획 (Part 1)

```text
   유닛    panel-past (U6) · 웨이브 W-d · 브랜치 unit/panel-past
   설계    ../panel-past/functional-design/  (domain-entities · business-logic-model ·
          business-rules).  불변식 스물여덟을 R62 ~ R89 로 센다.
          **U5 의 R14 ~ R30 과 U8 의 R31 ~ R61 도 이 유닛의 규칙이다** (렌더러가 한 벌이다)
   답      1=A · 2=A · 3=A · 4=A · 5=A · 6=A
   기준선   1ea1b3e.  unit/fleet-card 위 (답 1 = A)
   지는 게이트  **CB2**.  그리고 이것이 초록이면 **U8 의 CB6 보류가 풀린다**
```

---

## 0. NFR 스킵이 이 계획에 넘긴 값 하나

`unit-of-work.md` 10절이 U6 을 **스킵**으로 적었고 근거가 「출처만 바뀐다 · 기존
`do()` 를 탄다」다. 앞의 U4 · U5 · U8 과 달리 **이 스킵은 어긋남이 아니다** —
정본이 그렇게 걸었다. 그래도 값이 하나 남고 그 자리가 이 계획이다.

### 0.1 한 번 누르면 요청이 1 + N 이다

```text
   지금    1        tar 하나
   뒤     1 + N     Status 하나 + 단계마다 로그 하나
   실측    3단계     cb4-card-4.  요청 넷
   언제    누를 때만.  자동 갱신이 0 이라 초당 요청이 0 이다 (R75)
   천장    U4 의 전역 한도 120 req/s.  단계 스물이어도 한 번에 21 이다
```

**아픈 쪽은 브라우저가 아니라 Mediator 다.** 브라우저와 제어판 사이는 루프백이고,
제어판과 Mediator 사이가 N+1 이다. **순차로 부른다** — 동시성 1 이라 한도에
닿는 봉우리가 안 생기고, 20단계 Run 을 누른 사람이 기다리는 것은 한 번이다.

**CB2 가 이 자리를 처음 밟는다.** 아프면 `data` 를 두 번째 쿼리로 미루는 것이
다음 자리다 (U8 이 현황판에서 그렇게 했다 · 그쪽 답 7 = A).

---

## 1. 실측이 계획 단계에서 찾은 것 — 다섯

**W-c 의 계획이 넷, W-d 가 셋, U8 이 다섯을 찾았다. 이번에도 다섯이고
앞의 셋이 설계 문서의 줄을 고친다.**

### 1.1 봉투 하나가 두 읽기에서 나오면 R69 가 봉인 전에 깨진다

설계가 단계마다 **두 번** 부르기로 적었다 (`business-logic-model` 1절 ③ —
`as=events` 와 `as=raw`). **그 둘은 다른 읽기다.**

```text
   봉인 뒤    logs/NN-*.log 는 불변이라 두 읽기가 같은 바이트다.  안 아프다
   봉인 전    진행 파일은 자란다.  **두 읽기 사이에 자라면 사건 열과 원문이
             다른 창을 보인다.**  읽는 사람은 그것을 파서의 버그로 읽는다
```

**같은 자리를 U5 가 이미 이름으로 적어 뒀다** — `liveBody` 의 주석이
「토글을 켤 때 따로 읽으면 링이 그사이 감겨 다른 창을 보이고, 읽는 사람은
그것을 파서의 버그로 읽는다」이고, 그래서 그 함수는 **스냅샷 하나**를 받는다.
R69 (「`data` 는 같은 읽기의 같은 바이트다」)가 그 규율의 이 유닛 판이다.

```text
   고르는 법   단계마다 as=raw 를 **한 번** 부르고 제어판이 transcript.Parse 를
              부른다.  사건 열과 data 가 같은 바이트에서 나온다
   같은가      같다.  서버의 as=events 가 하는 일이 정확히 그것이다
              (log.go — 같은 sl.body 를 Parse 에 넣는다)
   headCut    ?from= 을 안 쓰므로 from 이 0 이고 probe 가 거짓이라 headCut 이
              거짓이다.  Parse(body, false) 가 서버와 **글자까지 같은 답**을 낸다
   덤         요청이 2N+1 에서 **1+N** 으로 준다.  설계 1.1 의 산수
              (「3단계면 4 · 20단계면 21」)가 그 수였다 — 두 줄이 어긋나 있었다
```

**「같은 파서」가 약해지지 않는다.** `internal/transcript` 하나이고, 제어판은
도는 것에서 이미 그것을 부른다 (`handleTranscript`). 지난 것도 같은 모양이 된다.

```text
   고치는 설계 줄   domain-entities 2.1 의 「왜 as 를 받나」
                  business-logic-model 1절 ③ 의 두 줄
   안 고치는 것     R67 ~ R71.  봉투의 키도 타입도 그대로다
```

### 1.2 `sealed` 를 제어판이 낼 독립된 출처가 없다

설계 3절의 봉투에 `sealed` 가 있고 「`steps` 의 `source` 와 같은 답을 내야 한다」
로 적혀 있다. **그 값을 어디서 얻느냐가 안 적혀 있고, 실측하니 자리가 없다.**

```text
   source 에서 유도하면   「같은 답을 내야 한다」가 자동으로 참이다.  재는 값이 0
   Terminal(state) 이면   **봉인 창에서 거짓말을 한다.**  log.go 의 주석이 그
                        창을 이름으로 적는다 — 「Run 이 종료 상태인데 Seal 이
                        아직 안 돈 창에서는 진행 파일이 아직 있다」
   R72 와의 자리         「봉인됐나」를 판정하는 조건문이 0 이어야 한다
```

```text
   고르는 법   봉투에서 sealed 를 뺀다.  Run 의 state 를 그대로 싣는다
   화면        봉인 전과 뒤는 **단계마다의 source 하나가 말한다** (R73 · R74 가
              이미 단계 단위다).  Run 한 줄로 접지 않는다
   값          단계마다 다를 수 있다 — 봉인 창에서 실제로 그렇다.  접으면
              그 한 단계가 거짓이 된다
```

### 1.3 한 응답 상한 1 MiB 가 지난 것에도 걸린다

`maxLogSliceBytes` 가 1 MiB 다. **폴링하는 쪽(U8)은 `?from=` 으로 이어 받지만
이 유닛은 한 번 읽고 끝이라 나머지가 영영 안 온다.**

```text
   봉인 뒤   OpenLog 의 total 이 파일 전체이고 body 는 1 MiB 에서 개행으로 끊긴다.
            **capped 헤더가 안 붙는다** — 그것은 진행 파일 쪽 값이다
   봉인 전   ReadProgress 가 capped 를 내지만 그것은 「진행 파일이 상한에 닿았다」
            (10 MiB)이지 「이 응답이 잘렸다」가 아니다.  다른 물건이다
   실측      cb4-card-4 는 세 단계 합 16,445 바이트라 안 닿는다.
            진행 파일은 한 단계가 127,117 이었고 상한이 10 MiB 다 — **닿을 수 있다**
```

```text
   고르는 법   화면이 total 과 받은 길이를 **비교**해 「처음 N 바이트만 왔다」를
              적는다.  헤더를 다시 계산하는 것이 아니다 (R70 은 그대로다) —
              봉투에 이미 있는 수 둘을 견주는 것이다
   안 하는 것   ?from= 으로 이어 받기.  자동 갱신 0 (R75) 과 부딪치고,
              **U4 의 라우트를 다시 여는 일이라 회차 밖이다**
   왜 지금     안 적으면 사람이 잘린 로그를 통째로 본 것으로 읽는다.
              틀린 줄은 없는 줄보다 나쁘다
```

### 1.4 커버리지 — 두 패키지 다 여유가 있다

표준 측정 명령(`-coverpkg=./...` · 패키지별 80% 하한)으로 이 기계에서 쟀다.

```text
   internal/panel    86.0%   221/257
   internal/runctl   97.1%   100/103
```

```text
   주의   DB 없이 재면 internal/store 5.6% · internal/api 20.6% 로 나온다.
         testdb.sh 없이 그 둘의 시험이 스킵되기 때문이고 이 유닛과 무관하다.
         **게이트 판정은 CI 의 숫자로 한다**
```

새 문장은 `StepLog` 하나와 `handleRecord` 다시 짓기다. **걷히는 문장도 있다**
(tar 풀기). 시험이 같이 서므로 하한에 안 닿는다.

### 1.5 기존 시험 둘이 tar 를 짓는다 — 그 둘이 이 유닛의 첫 빨강이다

```text
   transcript_test.go:79    TestHandleRecordExtractsLogs — tar 를 만들어 먹인다
   transcript_test.go:117   TestHandleRecordNeedsRun — run= 없으면 400.  **그대로 산다**
   transcript_test.go:4     import "archive/tar" — 시험 쪽 임포트다
```

**첫째를 지우지 않고 다시 짓는다.** 재던 것(「`logs/*.log` 만 꺼낸다」)이
없어지고 재는 것이 바뀐다 — 「단계마다 로그 하나를 받아 봉투로 묶는다」.

---

## 2. 이 유닛이 내는 diff 의 모양

```text
   짓는다        (없다).  새 파일이 0 이다

   고친다        internal/runctl/client.go        StepLog 하나
                internal/runctl/client_test.go   그 메서드의 시험
                internal/panel/transcript.go     handleRecord 다시 짓기 · tar 걷힘
                internal/panel/transcript_test.go 시험 다시 짓기
                internal/panel/boundary_test.go  금지 한 줄 (R62)
                internal/panel/page.go           loadRecord 다시 짓기

   문서          aidlc-docs/taeels/construction/panel-past/code/code-summary.md
                aidlc-docs/taeels/aidlc-state.md · audit.md

   diff 0        internal/transcript/**          파서를 안 건드린다
                internal/transcriptui/card.mjs   렌더러를 안 건드린다 (R81)
                internal/api/**                  서버를 안 건드린다
                internal/api/ui/**               현황판을 안 건드린다
                internal/panel/panel.go          라우트 11 그대로 (R65 · CB0)
                cmd/runctl/main.go               Client.Record 가 그대로 산다 (R63)
```

**새 Go 패키지가 0 이라 커버리지 표의 행이 안 는다.** U8 과 다른 자리다.

---

## 3. 못 박는 겉면 — Step 1 이 먼저 한다

```go
// internal/runctl
func (c *Client) StepLog(ctx context.Context, runID string, seq int, name string) ([]byte, http.Header, error)
```

```text
   as 가 없다     1.1 이 지웠다.  이 메서드는 원문 바이트만 낸다 —
                 사건 열은 부르는 쪽이 transcript.Parse 로 만든다
   헤더를 낸다     출처 · 총 길이 · 상한이 거기 있다 (R70).  몸통만 내면
                 부르는 쪽이 봉인 전인지 뒤인지를 못 안다
   do() 를 탄다    인증과 오류 사상이 그대로다.  새 외부 표면이 0 이다 (R66)
   name 은 Step.ID  상세의 단계 객체에 name 이 없다.  U8 이 이미 밟은 자리다
```

```go
// internal/panel — 봉투 (설계 3절 · 1.2 가 sealed 를 state 로 바꿨다)
type pastStep struct {
	Seq        int                `json:"seq"`
	ID         string             `json:"id"`
	State      string             `json:"state"`
	Chosen     bool               `json:"chosen"`
	Source     string             `json:"source,omitempty"`
	Total      int64              `json:"total"`
	Capped     bool               `json:"capped,omitempty"`
	Transcript *transcript.Result `json:"transcript,omitempty"`
	Data       string             `json:"data,omitempty"`
	Error      string             `json:"error,omitempty"`
}

type pastRecord struct {
	RunID string     `json:"run_id"`
	State string     `json:"state"`
	Steps []pastStep `json:"steps"`
}
```

```text
   Transcript 가 포인터   liveTranscript 와 같은 이유다.  실패한 단계에서 키가
                       통째로 빠져야 하고 값 타입이면 omitempty 가 안 듣는다
   Error 는 단계 안      R86.  한 단계의 실패가 화면 전체를 안 비운다
   recordLog 가 사라진다  {name, content} 를 아무도 안 쓴다
```

---

## 4. 단계 — 여덟

### Step 1 — 겉면 하나 (R66)
- [ ] `internal/runctl/client.go` 에 `StepLog`. 기존 `do()` 를 탄다
- [ ] 경로는 `/v1/runs/{id}/steps/{seq}/log?name=<Step.ID>&as=raw`
- [ ] `?from=` 이 **0 번** 나온다 (1.3 · R75)
- [ ] 몸통과 헤더를 함께 낸다. `resp.Body` 를 닫는다
- [ ] `go build ./...` 가 서고 `go list ./...` 가 **안 는다**

### Step 2 — `handleRecord` 를 다시 짓는다 (R62 ~ R72)
- [ ] `archive/tar` 임포트가 사라진다. 파일에 `tar` 라는 낱말이 0 이다
- [ ] `Client.Status` 로 단계 목록을 받는다. 실패하면 `markMediator(false)` (R87)
- [ ] `runctl.Fail` 의 404 를 **404 로 그대로 낸다** (R88). 다른 실패는 502
- [ ] 단계마다 `StepLog` 를 **순차로** 한 번 (0.1). 동시성 1 이다
- [ ] 헤더 셋을 **그대로** 옮긴다 — `source` · `total` · `capped` (R70)
- [ ] `transcript.Parse(body, false)` 로 사건 열을 만든다 (1.1)
- [ ] `Data` 는 **같은 바이트**다. 두 번 안 읽는다 (R69)
- [ ] 단계 하나가 실패하면 그 단계의 `Error` 에 적고 **나머지를 그린다** (R86)
- [ ] 로그가 없는 단계를 목록에서 **안 뺀다** (R79). 상태로 미리 안 거른다 (R80)
- [ ] `sealed` 를 안 낸다. `state` 를 낸다 (1.2)
- [ ] `recordLog` 타입이 사라진다

### Step 3 — 경계 (R62)
- [ ] `boundary_test.go` 가 `internal/panel` 의 의존에 **`archive/tar` 가 없음**을 잰다
- [ ] 표준 라이브러리 이름이라 기존 `has` 와 다른 자리다 — 단언을 따로 둔다
- [ ] 기존 열둘과 봉인 둘이 그대로 초록이다

### Step 4 — 화면 (R73 ~ R89)
- [ ] `page.go` 의 `loadRecord` 가 새 봉투를 그린다
- [ ] 사건 열은 `window.enodeCard.renderEvents` — **그리는 코드가 0 줄** (R81)
- [ ] 펼침은 단계마다 따로. 열쇠는 `tool_use_id` (R83)
- [ ] 원문 토글은 `pre.textContent = step.data` — 렌더러 밖이다 (R84 · U8 의 R36)
- [ ] `statusLine` 을 **안 부른다** (R82)
- [ ] 걷힌 줄 — 「본문 N 개가 걷혔다 (M 바이트)」. **단계마다 하나** (R85)
- [ ] `source` 가 `progress` 면 「봉인 전」을 적는다 (R73)
- [ ] 총 길이 0 의 세 문장 (R76 · R77 · R78)
- [ ] 잘린 응답에 「처음 N 바이트만 왔다」 (1.3)
- [ ] 단계의 `Error` 는 그 단계 자리에 (R86) · 빈 `catch` 가 0 (R89)
- [ ] `verdict` 를 안 만진다. 지금처럼 Run 목록에서 온다
- [ ] 하네스 바이트가 `innerHTML` 에 **0 번** 닿는다 (R84)

### Step 5 — 시험: 클라이언트
- [ ] `StepLog` 이 URL 을 그 모양으로 짓는다 — `name` 이 붙고 `from` 이 없다
- [ ] 헤더 셋이 그대로 올라온다
- [ ] 비 2xx 가 `*Fail` 이고 코드를 든다 (기존 시험과 같은 모양)

### Step 6 — 시험: 제어판
- [ ] 가짜 Mediator 가 단계 셋을 내고 단계마다 다른 바이트를 준다
- [ ] 봉투의 단계 수가 상세와 **같다**. 로그 0 바이트인 단계도 있다 (R79)
- [ ] `source` 가 `progress` 인 단계와 `sealed` 인 단계가 **한 봉투에** 있다 (1.2 의 값)
- [ ] `elided` 가 봉인된 단계에만 있다 (R74 를 재는 줄)
- [ ] 단계 하나만 500 을 내면 **그 단계만** `error` 이고 나머지가 그대로다 (R86)
- [ ] 상세가 404 면 제어판도 404 다 (R88)
- [ ] 상세가 죽으면 502 이고 `markMediator(false)` 다 (R87)
- [ ] **같은 읽기의 같은 바이트** — 같은 단계를 두 번 읽으면 다른 것을 주는
      가짜 서버에서 `data` 와 `transcript.events` 가 어긋나지 않는다 (R69 · 1.1)
- [ ] `/api/record` 에 `run=` 이 없으면 400 (기존 시험 그대로 산다)

### Step 7 — 변이
- [ ] 변이 일곱 (5절). 실측 결과를 옆에 적는다

### Step 8 — 게이트와 문서
- [ ] `go test ./... -count=1` 전부 초록
- [ ] 패키지별 커버리지 80% 하한 통과 (1.4 의 주의를 함께 적는다)
- [ ] `node --test internal/api/ui/tests/*.test.mjs` 전부 초록 (**안 건드렸다**를 잰다)
- [ ] `go run ./scripts/glyphscan.go` 통과 · `gofmt` 깨끗
- [ ] `diff 0` 이어야 할 곳 전부 0 (2절)
- [ ] `grep -c 'mux.HandleFunc' internal/panel/panel.go` 가 **11 그대로** (R65)
- [ ] 코드 요약을 쓰고 이 계획의 체크박스를 **실측으로** 채운다
- [ ] **CB2 는 사람이다** (6절)

---

## 5. 변이 일곱 — 안 죽으면 시험이 없는 것이다

```text
   ①  source 헤더를 안 옮기고 "sealed" 를 박는다        봉인 전 단계 시험이 빨개야
   ②  elided 를 봉투에서 버린다                        걷힌 줄 시험이 빨개야
   ③  단계 하나의 실패를 전체 502 로 접는다             R86 시험이 빨개야
   ④  SKIPPED 의 chosen 을 안 싣는다                   R76 · R77 시험이 빨개야
   ⑤  로그 0 바이트인 단계를 목록에서 뺀다              R79 시험이 빨개야
   ⑥  data 를 두 번째 읽기로 채운다 (as=raw 를 다시)    R69 시험이 빨개야 (1.1)
   ⑦  404 를 502 로 접는다                            R88 시험이 빨개야
```

**⑥ 이 이 유닛의 값을 지키는 줄이다.** 나머지 여섯은 화면이 무엇을 말하는가를
재고, ⑥ 은 **봉투 하나가 한 읽기에서 나오는가**를 잰다. 그 줄이 없으면 앞 판의
설계로 되돌아가도 하네스가 초록이다.

---

## 6. 게이트 — 하나를 지고 하나를 푼다

```text
   CB2   진다.  사람이 실제 하네스로 본다
         끝난 Run 을 제어판 지난 작업에서 누른다 -> 같은 파서의 같은 모양
         브라우저 네트워크 탭 -> GET /v1/runs/<id>/record 가 **0 개**이고
                              .../log 가 **단계 수만큼**
         verdict.checks 가 그대로 보인다
         **안 끝난 Run 을 누르면 「봉인 전」이 적혀 있다** (scene-gates 2.1)

   CB6   푼다.  U8 의 측정에서 장면 ⑥ 의 절반이 보류였고 이유가 U6 이 없어서였다.
         나머지 절반(tar 와 GET 이 같은 바이트)은 이미 초록이다 — 세 단계 전부
         cmp 로 확인했다 (11,553 · 102 · 4,790)

   CB0   라우트 수가 안 는다.  U5 가 11 로 만든 그대로다 (R65)
```

**측정 나무가 필요하다.** `unit/panel-past` 는 `unit/fleet-card` 위에 있고 U8 이
아직 병합 전이다. U8 의 측정 나무(`measure/u8-cb4` — main + U5 + U7 + U8)와 같은
모양으로 짓고, **병합은 그 나무가 아니라 `unit/panel-past` 가 진다.**

---

## 7. 짓지 않는 것

```text
   파서            internal/transcript 를 그대로 쓴다
   렌더러           card.mjs.  이 유닛이 그리는 코드를 0 줄 짓는다 (R81)
   라우트           GET /api/record 가 이미 있다 (R65)
   Client.Record    cmd/runctl 이 쓴다.  안 없앤다 (R63)
   폴링            봉인된 것은 안 자란다 (R75)
   ?from= 이어받기   1.3.  U4 의 라우트를 다시 여는 일이라 회차 밖이다
   GET log 의 name 기본값  아직 step 이다.  화면에서 Step.ID 를 넘겨 비켜 간다
   회차 문서 수정    unit-of-work.md 의 U6 절에 page.go 가 빠진 것과
                   의존 행렬에 U8 칸이 없는 것.  **적고 넘어간다**
                   (business-rules 7절 · 9절)
```

---

## 8. 물음 — 0

**설계 단계의 답 여섯이 값을 다 정했고, 스킵이 남긴 값 하나를 0절이 졌다.**

1절의 발견 셋(1.1 · 1.2 · 1.3)은 물음이 아니라 **고침**이다 — 설계 문서의 줄이
서로 어긋나거나(1.1 의 요청 수), 값을 낼 자리가 없거나(1.2 의 `sealed`), 적히지
않은 경계가 실측으로 나왔다(1.3 의 1 MiB). 셋 다 승인받은 답 여섯을 안 무른다.
**이 계획의 승인이 그 고침의 승인을 겸한다.**
