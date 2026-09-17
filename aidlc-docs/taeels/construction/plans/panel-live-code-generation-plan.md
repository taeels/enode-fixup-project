# U5 `panel-live` — Code Generation 계획 (Part 1)

```text
   유닛    panel-live (U5) · 웨이브 W-c · 브랜치 unit/panel-live
   설계    ../panel-live/functional-design/  (domain-entities · business-logic-model ·
          business-rules).  불변식 스물아홉을 R1 ~ R29 로 센다
   답      1=A · 2=A · 3=A · 4=A · 5=C · 6=A · 7=A · 8=A
   기준선   05ee710.  W-b 가 병합된 main
   지는 게이트  **CB1** — 사람이 실제 하네스로 본다
```

---

## 0. NFR 스킵이 이 계획에 넘긴 값 둘

NFR Requirements 와 NFR Design 이 둘 다 SKIP 이므로 (사용자 지시) **값을 정할
자리가 이 계획이다.** W-b 의 U4 와 같은 모양이다.

### 0.1 폴링 간격 1초 — 안 바꾼다

앞 팩의 값이고 `requirements.md` 5.7 이 그대로 진다. **이 유닛이 파싱을 더하지만
간격을 안 늘린다** — 늘리면 CB1 이 재는 「단계가 끝나기 전에 흐른다」가 그만큼
둔해진다. 대신 **캐시로 침묵 구간의 비용을 0 으로 만든다** (R7).

### 0.2 매초 512 KiB 파싱의 봉투 — 안 닫고 산수만 남긴다

```text
   최악      링이 꽉 찬 채로 계속 새 바이트가 온다 -> 매초 512 KiB 를 파싱한다
   파서      줄 단위 · 표준 라이브러리만 · 할당은 사건 수에 비례
   완화      캐시가 (generation, total, mtime) 로 걸린다.  침묵에서는 0 이다
   천장      노드 한 대에 제어판 탭 몇 개인가.  루프백이고 사람이 여는 화면이라
            함대 규모가 안 곱해진다 — **U4 의 N1 과 다른 자리다**
```

**아픈 자리는 노드의 CPU 를 하네스와 나눠 쓴다는 것이다.** 하네스가 도는 동안
파싱이 같은 기계에서 돈다. **CB1 이 512 KiB 를 넘기는 프롬프트로 이 자리를
실제로 밟는다** — 빨개지면 답 2 를 B 로 다시 연다 (business-rules 8절 ③).

---

## 1. 실측이 계획 단계에서 찾은 것 — 넷

**W-b 의 계획이 둘을 찾았다. 이번에는 넷이다.**

### 1.1 제어판의 JS 는 이 저장소의 어떤 시험도 못 잰다

```text
   .mjs 시험 열둘   internal/api/ui/tests/*.test.mjs
   도는 자리        .github/workflows/gallery-demo.yml:22  node --test
   제어판의 JS      page.go 의 indexHTML 문자열 상수 안이다 — **임포트가 안 된다**
```

**R19 · R20 · R21 · R14 를 재는 자동 시험이 0 이다.** 그 넷은 전부 DOM 규칙이고,
이 저장소에서 그것을 재는 길은 `internal/api/ui` 의 `.mjs` 하네스뿐인데 제어판의
JS 는 거기 못 들어간다.

```text
   즉  그 넷의 유일한 검사는 CB1 이다.  사람이다
```

**이것이 답 1 = B (page.go 를 static 으로 가른다) 의 값에 하나를 더한다** —
CSP 뿐 아니라 **시험 가능성도 거기 붙어 있었다.** 사용자가 A 를 골랐으므로 그
대가를 이름으로 적고 넘긴다 (6절).

**이 계획이 그 대가를 줄이는 법** — **Go 쪽으로 옮길 수 있는 판정은 전부 옮긴다.**
`truncated` 는 Go 가 정하고 (R10), 봉투도 Go 가 짓고 (R6), 캐시도 Go 다 (R7).
**브라우저에 남는 것은 그리기뿐이다.**

### 1.2 `internal/panel` 은 `ui.securityHeaders` 를 못 쓴다 — 경계가 막는다

`boundary_test.go:46` 이 `internal/panel -> internal/api` 를 금지한다. **다섯 줄을
제어판 안에 다시 쓴다.**

**`CONVENTIONS.md` 1.4 와 안 부딪힌다** — CSP 값이 애초에 갈렸다 (R25). 같은 것
두 벌이 아니라 **다른 것 둘**이다. business-rules R29 가 그 판단의 자리다.

### 1.3 기존 시험의 픽스처가 사건을 0 개 낸다 — 개행이 없다

```go
   transcript_test.go:31   r.Write([]byte("agent is thinking..."))
   transcript_test.go:40   if out2["data"] != "agent is thinking..."
```

**W-b 가 배운 것이 그대로 여기 있다** — `transcript.Parse` 는 개행 없이 끝나는
꼬리를 안 읽고 `Partial` 에 센다. 이 픽스처는 **사건 0 개 · `partial` 20** 을 낸다.

```text
   답 6 = A 가 data 를 남기므로 이 시험은 그대로 초록이다 — 고칠 필요가 없다
   그러나 사건을 재는 새 시험은 **개행을 붙여야 한다.**  안 붙이면 0 을 세고
   그 0 이 「파서가 안 불렸다」와 구별이 안 된다
```

**W-b 의 U2 가 정확히 이 자리에서 「사건이 0 이었다」를 겪었다.** 같은 함정을
두 번 밟지 않도록 Step 에 못 박는다.

### 1.4 `internal/panel` 의 커버리지가 84.9% 다 — 하한에서 4.9 포인트

W-b 의 합본 측정값이다. 이 유닛이 `handleTranscript` 를 통째로 다시 쓰고 헤더
래퍼를 더한다. **`page.go` 의 증가분은 문자열 상수라 문장 수에 0 을 더한다** —
커버리지에 안 잡히고, 그래서 **Go 쪽 새 코드가 안 덮이면 바로 내려간다.**

시험 Step 을 셋으로 나눈 이유가 이것이다 (Step 8 · 9 · 10).

---

## 2. 이 유닛이 내는 diff 의 모양

```text
   internal/enode/transcript.go     Snapshot 에 Capacity 한 필드.  **행렬 밖** (6절)
   internal/panel/transcript.go     handleTranscript 를 다시 쓴다.  handleRecord 는 0
   internal/panel/headers.go (신규)  보안 헤더 다섯
   internal/panel/panel.go          Handler() 가 헤더 래퍼를 감싼다.  라우트는 0 개 증가
   internal/panel/page.go           카드 HTML 과 그리는 JS

   시험
   internal/enode/transcript_test.go   Capacity 하나
   internal/panel/transcript_test.go   봉투 · 캐시 · 잘림 · 경과
   internal/panel/headers_test.go (신규)  다섯이 모든 응답에 붙는다
```

```text
   diff 0 이어야 하는 곳   internal/api/**  ·  internal/transcript/**  ·
                        internal/record/**  ·  go.mod  ·  go.sum
                        internal/panel/handlers.go  ·  view.go  ·  boundary_test.go
   라우트                internal/api/api.go 의 18 그대로 (CB0 을 안 건드린다)
                        internal/panel/panel.go 의 10 그대로 (R1)
```

---

## 3. 못 박는 겉면 — Step 1 이 먼저 한다

```go
   // internal/enode
   type Snapshot struct { Data []byte; Generation, Total uint64; Capacity int64 }

   // internal/panel
   type liveTranscript struct {
       Available  bool               `json:"available"`
       Generation uint64             `json:"generation,omitempty"`
       Total      uint64             `json:"total,omitempty"`
       Capacity   int64              `json:"capacity,omitempty"`
       Truncated  bool               `json:"truncated,omitempty"`
       LastWrite  string             `json:"last_write,omitempty"`
       RingPath   string             `json:"ring_path,omitempty"`
       Transcript *transcript.Result `json:"transcript,omitempty"`
       Data       string             `json:"data,omitempty"`
   }
```

**`Transcript` 가 포인터인 이유** — `available:false` 일 때 키가 통째로 빠져야
한다. 값 타입이면 `omitempty` 가 구조체에 안 듣는다 (W-b 의 U4 가 `Info` 에서
같은 자리를 밟았고 거기서는 `omitzero` 로 닫았다 — 여기는 없으면 `nil` 이 맞다).

**기간 필드가 0 인 것이 R5 를 재는 법이다.** 이 구조체에 초 단위 수가 없다.

---

## 4. 단계 — 열하나

### Step 1 — 겉면을 못 박는다
- [x] `enode.Snapshot` 에 `Capacity int64`
- [x] `panel.liveTranscript` 를 3절 그대로
- [x] `go build ./...` 가 선다

### Step 2 — `enode.Snapshot.Capacity` 를 채운다 (R9)
- [x] `ReadRing` 이 이미 읽은 `capacity` 를 `Snapshot` 에 넣는다
- [x] **동작 diff 0** — 기존 시험 전부 그대로 초록
- [x] 시험 — 감긴 링과 안 감긴 링에서 `Capacity` 가 파일 머리의 값과 같다

### Step 3 — `handleTranscript` 의 읽기와 캐시 (R6 · R7 · R8)
- [x] `os.Stat` 으로 mtime 을 **먼저** 얻는다 (`ReadRing` 앞이다 — 캐시가 512 KiB 를 아낀다)
- [x] 캐시 열쇠 `(generation, total, mtime)`. **셋이다** — 둘로 안 줄인다
- [x] 캐시는 한 칸. `sync.Mutex` 하나로 감싼다 (폴링 탭이 여럿일 수 있다)
- [x] `Stat` 실패 · `ReadRing` 실패 -> `available:false` (R15 ①)

### Step 4 — 파싱과 봉투 (R9 · R10 · R11)
- [x] `truncated = snap.Total > uint64(snap.Capacity)` — **엄격 부등호** (R9)
- [x] `transcript.Parse(snap.Data, truncated)` -> `Result`
- [x] `Result` 를 `transcript` 키에 **통째로**. 펼치지 않는다 (교차 검사 · U6 이 받는다)
- [x] `data` 에 **같은 읽기의 같은 바이트** (R23)
- [x] `last_write` 는 RFC 3339 **시각**. 기간이 아니다 (R5)
- [x] `ring_path` 는 `enode.TranscriptPath(s.cfg.ConfigPath)` (R 7절 · 답 7 = A)

### Step 5 — 보안 헤더 다섯 (R24 ~ R29)
- [x] `internal/panel/headers.go` 신규. 넷은 `ui.go:64-71` 과 글자 그대로
- [x] CSP 는 갈린 값 — `default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'`
- [x] `Handler()` 에서 **모든 응답**을 감싼다 (R28). `requireToken` 과의 순서를 정한다
- [x] `ui.securityHeaders` 를 임포트하지 않는다 — 경계가 막는다 (R29)

### Step 6 — 카드의 HTML (R18)
- [x] 트랜스크립트 카드에 자리 넷 — 사건 열 · 경과 · 잘림 · 잔여 한 줄
- [x] 원문 토글 버튼 하나. `<pre>` 는 그대로 쓰되 기본이 접힘
- [x] 데몬 로그 카드는 **한 글자도 안 고친다** (FR-4)
- [x] `TestHandleIndexServesButtons` 가 세는 다섯 문자열이 그대로 있다

### Step 7 — 그리는 JS (R14 · R16 · R17 · R18 · R19 · R20 · R21 · R22 · R23)
- [x] 사건을 `textContent` 로만 그린다. `innerHTML` 에 하네스 바이트가 0 번 닿는다 (R18)
- [x] 펼침 상태를 `Event.ID` 로 든다. `Line` 으로 안 든다 (R19)
- [x] **그리기 전에** 바닥 여부를 재고, 바닥이었을 때만 따라간다 (R21)
- [x] 세대가 바뀌면 펼침 · 스크롤 · 토글을 비운다 (R20)
- [x] 경과는 **별개 타이머** 1초. 폴링이 죽어도 흐른다 (R14)
- [x] 폴링이 실패하면 카드를 회색으로 두고 마지막 값을 **안 지운다** (R16)
- [x] 이 카드 경로의 빈 `.catch` 를 0 으로 (R17)
- [x] 종류 일곱만 그린다. 모르는 것은 `raw` 로 온다 (R22)

### Step 8 — 시험: 봉투와 캐시
- [x] 링이 없다 -> `available:false` 이고 나머지 키가 **없다**
- [x] 링을 읽었다 -> `transcript.events` 가 찬다. **픽스처에 개행을 붙인다** (1.3)
- [x] `data` 가 사건과 **같은 바이트**다
- [x] 응답 JSON 에 초 단위 기간 필드가 **0 개**다 (R5 를 세는 법)
- [x] 같은 링을 두 번 부르면 두 번째가 파싱을 안 한다 (캐시가 듣는다)
- [x] 링에 새로 쓰면 캐시가 안 듣는다

### Step 9 — 시험: 잘림과 경과
- [x] `Total == Capacity` -> `truncated:false` (**엄격 부등호를 재는 줄이다** · R9)
- [x] `Total > Capacity` -> `truncated:true` 이고 `Parse` 가 머리를 버렸다
- [x] `last_write` 가 링 파일의 mtime 과 같다
- [x] `Ring.Reset()` 뒤 `generation` 이 오르고 `last_write` 가 새것이다

### Step 10 — 시험: 헤더 다섯
- [x] `GET /` 에 다섯이 다 있다
- [x] **`GET /api/transcript` 에도 다 있다** (R28 이 고친 자리다)
- [x] CSP 값이 갈린 값 그대로다 — 이 줄이 R25 의 어긋남을 고정한다
- [x] `boundary_test.go` 의 여덟 줄이 그대로 초록 (R29 를 안 깼다)

### Step 11 — 변이와 게이트
- [x] 변이 여섯 (5절)
- [x] `eval "$(scripts/testdb.sh)"` 뒤 CP0 계열 전부
- [x] 라우트 둘 다 안 늘었다 — `api.go` 18 · `panel.go` 10
- [x] `diff 0` 이어야 할 곳 전부 0
- [x] 코드 요약을 쓰고 이 계획의 체크박스를 **실측으로** 채운다

---

## 5. 변이 여섯 — 안 죽으면 시험이 없는 것이다

**실측 결과를 옆에 적는다. 셋이 처음에 살아남았다.**

```text
   ①  truncated 를 언제나 false 로            ReportsAWrappedRing            FAIL
   ②  엄격 부등호를 >= 로                     TreatsAnExactlyFullRingAsWhole  FAIL
   ③  응답에 기간(초)을 싣고 캐시한다           **처음에 살아남았다** -> FAIL
   ④  캐시 열쇠에서 mtime 을 뺀다              **처음에 살아남았다** -> FAIL (둘)
   ⑤  헤더를 GET / 에만 건다                  SecurityHeadersAreOnEvery...   FAIL (둘)
   ⑥  Parse 에 넘기는 바이트와 data 를 다른    **시험으로는 끝내 못 잡았다** ->
      읽기에서 가져온다                        **구조로 닫았다** -> FAIL
```

**③ 이 살아남은 이유** — 방금 쓴 링은 경과가 0 이라 변이가 더한 `age_seconds` 를
`omitempty` 가 지웠다. 시험이 없는 키를 못 찾아 초록이었다. `os.Chtimes` 로 링을
90초 늙히고, **금지어 목록 대신 봉투가 인정한 키 밖의 수를 전부** 막았다.
한 번 헛디뎠다 — 「60 ~ 120 사이의 수」로 걸렀더니 `total` 이 76 바이트라 걸렸다.
**바이트 수와 초가 같은 자리에 온다.**

**④ 가 살아남은 이유** — 시험이 링을 다시 만든 뒤 한 줄을 써서 `total` 이 앞것과
달랐다. **캐시의 `(0,0)` 과 애초에 안 부딪힌다.** 길이가 같고 내용이 다른 두 줄로
고쳤다 — 그러면 갈리는 것이 mtime 하나뿐이다.

**⑥ 은 시험으로 끝내 못 잡았다.** 조용한 링에서는 두 읽기가 같은 바이트다.
경합으로 재니 열에 셋, 산수로 좁혀도 열에 일곱이었다. **열에 셋을 놓치는 시험은
게이트가 아니다.** 시험을 버리고 `liveBody(snap, mtime, path)` 로 갈라
**스냅샷 하나만 받게** 했다 — 그 갈래가 아예 없다. 코드 요약 2.1 이 전문이다.

**DOM 규칙 넷(R19 · R20 · R21 · R14)에 변이를 못 건다** — 1.1 이 그 이유다.
**그 넷은 CB1 이 진다.** 변이를 못 거는 것을 숨기지 않고 여기 적는다.

---

## 6. 진행자에게 넘기는 것 — 다섯

```text
   requirements.md 5.5    「같은 다섯 줄」이 page.go 의 모양을 안 보고 쓰였다.
                          R25 가 그 어긋남이다 (CSP 만 갈렸다)

   features.md 3.3 ·      「밀렸다는 표시는 앞 팩 그대로」가 거짓이다.
   requirements.md FR-4   앞 팩에 0 이고 이 유닛이 새로 짓는다

   unit-of-work.md U5 절   NFR 요구가 한 문서 안에서 두 값이다 (U5 절 SKIP ·
                          9절 표 EXECUTE).  SKIP 의 근거가 U6 의 문장이다

   파일 행렬 1절           internal/enode/transcript.go 가 어느 유닛에도 안 붙어 있다.
                          U5 가 Capacity 를 더한다.  동작 diff 0 · U2 병합돼 충돌 0

   **새로 는 것**          제어판의 JS 를 재는 시험이 이 저장소에 0 이다 (1.1).
                          답 1 = B 를 골랐으면 CSP 와 함께 이것도 닫혔다.
                          A 를 고른 대가이고, R19 · R20 · R21 · R14 의 유일한
                          검사가 CB1 이다
```

---

## 7. 짓지 않는 것

```text
   from= 오프셋      답 2 = A 가 뺐다
   지난 것           U6.  handleRecord 를 안 만진다
   노드 업로더        U7
   현황판 카드        U8
   새 .pen          S3 의 기존 카드를 고치는 것이다
   ReadRing 의       「없다」와 「깨졌다」를 가르는 일.  겉면이 는다 (잔여 ①)
   상태 둘 가르기
```

---

## 8. 물음 — 0

설계 단계의 답 여덟이 값을 다 정했고, 0절이 NFR 스킵이 남긴 둘을 이 계획에서
졌다. **새로 묻는 것이 없다.**
