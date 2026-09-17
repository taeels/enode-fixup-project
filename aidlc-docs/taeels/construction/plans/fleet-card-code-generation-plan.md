# U8 `fleet-card` — Code Generation 계획 (Part 1)

```text
   유닛    fleet-card (U8) · 웨이브 W-e (마지막) · 브랜치 unit/fleet-card
   설계    ../fleet-card/functional-design/  (domain-entities · business-logic-model ·
          business-rules).  불변식 서른하나를 R31 ~ R61 로 센다.
          **U5 의 R14 ~ R30 도 이 유닛의 규칙이다** (렌더러가 한 벌이다)
   답      1=A · 2=A · 3=A · 4=A · 5=A · 6=A · 7=A · 8=A
   기준선   ec6f48b.  U5 가 병합된 main 위 (PR #46)
   지는 게이트  **CB4** · **CB6** · **CB1 의 카드 줄 재확인**
```

---

## 0. NFR 스킵이 이 계획에 넘긴 값 셋

NFR Requirements 와 NFR Design 이 둘 다 SKIP 이다 (사용자 지시 2026-09-17).
**`unit-of-work.md` 는 U8 을 「돈다」로 적었으므로 어긋남이고**, 그래서 값을
정할 자리가 이 계획이다. W-b 의 U4 · W-c 의 U5 와 같은 모양이다.

### 0.1 전체 재수신의 봉투 — 안 닫고 산수만 남긴다

R43 이 `?from=` 을 안 쓴다. 대가가 바이트다.

```text
   최악      진행 파일이 MaxBlobBytes(10 MiB)까지 자란 단계를 2초마다 통째로 받는다
   현실      CB3 의 실측에서 한 단계가 6,785 바이트였다.  76 KiB 짜리도 봤다
   곱하기    도는 단계 x 보는 사람 x 0.5 req/s.  **도는 단계는 보통 하나다** (R49)
   완화      R52 가 DOM 을 총 길이가 움직였을 때만 간다.  바이트는 그대로 든다
   천장      U4 의 전역 한도 120 req/s.  카드가 여는 것은 사람이 Run 상세를 열었을
            때뿐이고 (R51) 닫으면 0 이다
```

**아픈 자리는 바이트이지 요청 수가 아니다.** 요청은 초당 0.5 이고, 나르는 양이
파일 크기에 비례한다. **CB4 가 이 자리를 처음 밟는다** — 빨개지면 답 8 을 C 로
다시 연다 (서버가 「마지막 완전한 줄의 끝」을 헤더로 낸다). **U4 의 라우트를 다시
여는 일이라 회차 밖이다** (business-rules 9절).

### 0.2 사건이 수천일 때 — 카드 수와 DOM

```text
   카드 수    단계마다 하나.  그리는 것은 전부이나 **폴링은 CLAIMED 만** (R49)
   DOM       총 길이가 안 움직이면 0 (R52).  움직이면 그 카드 하나를 다시 짓는다
   천장      진행 파일의 상한이 사건 수의 상한을 준다
   안 재는 것  수천 사건에서의 재구성 비용.  U5 가 링에서 같은 자리를 남겼고
            (그쪽 잔여 ④) 여기서도 **CB4 가 재는 자리**로 둔다
```

### 0.3 데모 모드의 노출 — 물음 하나로 낸다

`GET log` 는 데모에서 무인증 + 한도다 (`api.go:82` 의 `read`). **라우트는 오늘도
열려 있었고 이 유닛이 그것을 화면에 올린다.** 값을 여기서 정하지 않고 8절의
물음 하나로 낸다 — 이 유닛이 새로 여는 것이 아니라 **보이게 만드는** 것이라
사람이 고를 자리다.

---

## 1. 실측이 계획 단계에서 찾은 것 — 다섯

**W-c 의 계획이 넷, W-d 가 셋을 찾았다. 이번에는 다섯이고 첫째가 무겁다.**

### 1.1 `view.mjs` 가 렌더러를 정적 임포트하면 `.mjs` 하네스가 깨진다

**디스크 경로와 URL 경로가 다르다.** 답 1 = A 가 렌더러를 `internal/transcriptui`
로 뺐고, 그 파일은 `internal/api/ui/static/` 트리 **밖**이다.

```text
   브라우저가 보는 것   /ui/shared/transcriptui/card.mjs   ui.go 의 새 라우트가 낸다
   디스크에 있는 것     internal/transcriptui/card.mjs     static 트리 밖이다
   시험이 도는 법       node --test internal/api/ui/tests/*.test.mjs  (디스크다)
```

`view.mjs` 에 `import ... from '../transcriptui/card.mjs'` 를 적으면 **브라우저에서는
서고 `node --test` 에서는 그 경로가 디스크에 없어 터진다.** 기존 시험 열둘이
`view.mjs` 를 임포트하므로 **하네스가 통째로 빨개진다.**

```text
   고르는 법   view.mjs 가 렌더러를 **주입받는다.**  정적 임포트를 안 쓴다
              app.mjs · demo.js 가 URL 로 동적 임포트해 DashboardView 에 넘긴다
              시험은 디스크 경로로 임포트해 같은 자리에 넣는다
   근거       DashboardView 가 이미 옵션 객체로 의존을 받는다
              (onRetry · onLogout · onRunSelection · onRunHistory)
   덤        렌더러를 갈아 끼울 수 있으므로 카드 시험이 DOM 만 재고
              네트워크를 안 탄다
```

**이것이 답 1 = A 를 살리는 유일한 모양이다.** 복사는 답 1 = B 이고 사용자가
안 골랐다.

### 1.2 새 Go 패키지가 커버리지 게이트에 들어온다

```text
   ci.yml:265   go test ./... -coverpkg=./... 뒤 **패키지별 80% 하한**
                「한 패키지라도 미달이면 exit 1」
   ci.yml:345   want="$(go list ./... | wc -l)" — 동적이라 패키지가 늘어도 안 깨진다
   지금         패키지 19.  U8 이 20 으로 만든다
```

**문장이 0 이면 그 패키지가 표에 안 나온다** (블록이 없으므로 `t[p]` 가 안 선다).
`internal/transcriptui` 를 **함수 0 개**로 짓는다 — `//go:embed` 와 `var` 뿐이다.

```text
   짓는다     //go:embed card.mjs   ·   var Files embed.FS
   안 짓는다   func Bytes() []byte { ... }   문장 하나가 늘고 그것을 덮어야 한다
   덤        「테스트 파일 없음」은 패키지 레벨 스킵이라 스킵 게이트가 안 잡는다
             (ci.yml 의 그 규칙이 Test 필드 없는 skip 을 먼저 소비한다)
```

**여유는 두 패키지 다 있다** — 실측으로 `internal/api/ui` 98.4% ·
`internal/panel` 85.9% 다. 그래도 새 Go 문장은 라우트 둘뿐이라 얇게 는다.

### 1.3 `fleet_test.go` 가 내는 모듈 목록을 이미 세고 있다

```go
   fleet_test.go:8    열셋의 경로를 돌며 200 과 javascript MIME 을 잰다
   fleet_test.go:22   /ui/tests/... · /ui/testdata/... · /ui/shared/fleet/ 은 404 여야 한다
```

**새 경로가 그 표에 오른다.** 그리고 둘째 시험이 **디렉터리 목록이 안 나오는 것**을
재므로, 새 라우트도 `/ui/shared/transcriptui/` 를 404 로 내야 한다.

### 1.4 카드의 자리는 사이드바다 — 인스펙터는 그래프를 가린다

```text
   .dashboard-layout   2열 격자 — 장면 패널 + 사이드바 (fleet.css:29)
   사이드바가 든 것      필터 · runStatus · runList · askList · requirements · stepList
   인스펙터            장면 패널 **위에 떠 있는 것**.  닫기 버튼이 있다
```

FR-7 과 CB4 가 「그 카드가 그래프와 목록을 가리지 않는다」로 적었다. **인스펙터에
넣으면 그 줄이 구조적으로 거짓이 된다.** 사이드바의 `stepList` 아래가 자리다.

**최종 자리와 모양은 `.pen` 이다** — 진행자가 그린다. 이 계획이 정하는 것은
**어느 컨테이너에 붙느냐**뿐이고, 그것이 게이트 한 줄을 가른다.

### 1.5 그리는 함수가 전역 둘에 붙어 있다 — 콜백으로 끊는다

```text
   page.go   drawEvent 가 txOpen[e.id] 를 읽고 onclick 에서 drawTranscript() 를 부른다
   즉        렌더러가 제어판의 전역 상태와 다시 그리기 함수에 묶여 있다
```

**옮기면서 그 둘을 인자로 뺀다.** 그것 말고는 글자 그대로 옮긴다 (R40).

```text
   renderEvents(container, events, { open, onToggle })
   activeTool(events)  ->  { name } | null
```

`open` 은 펼친 `tool_use_id` 의 집합이고 `onToggle(id, next)` 는 부르는 쪽이 다시
그린다. **제어판은 `txOpen` 과 `drawTranscript` 를 넘기고, 현황판은 자기 상태와
`render` 를 넘긴다.**

---

## 2. 이 유닛이 내는 diff 의 모양

```text
   internal/transcriptui/card.mjs     신규.  그리는 것 한 벌
   internal/transcriptui/embed.go     신규.  //go:embed 와 var 뿐 — 함수 0 (1.2)
   internal/api/ui/ui.go              /ui/shared/transcriptui/ 라우트 하나
   internal/panel/panel.go            GET /static/card.mjs 하나.  **10 -> 11** (R32)
   internal/panel/page.go             그리는 함수가 빠지고 모듈 태그 하나가 든다
   internal/panel/boundary_test.go    금지 넷 (R31)
   internal/api/ui/static/shared/fleet/view.mjs      카드 · 주입 자리
   internal/api/ui/static/shared/fleet/transcript-poller.mjs   신규.  2초 폴러
   internal/api/ui/static/shared/fleet/client.mjs    규율을 함수로 뺀다 (R50)
   internal/api/ui/static/shared/fleet/fleet.css     카드 스타일
   internal/api/ui/static/fleet/app.mjs              동적 임포트와 주입
   internal/api/ui/static/demo/demo.js               같은 자리 (8절의 답에 달렸다)

   시험
   internal/api/ui/tests/card.test.mjs (신규)        R40 · R41 · R54 · R57 · 펼침
   internal/api/ui/tests/transcript-poller.test.mjs (신규)  R44 ~ R52 · R58 ~ R61
   internal/api/ui/fleet_test.go                     새 경로 둘 (1.3)
   internal/panel/panel_test.go                      /static/card.mjs 가 선다
```

```text
   diff 0 이어야 하는 곳   internal/transcript/**  ·  internal/record/**  ·
                        internal/store/**  ·  internal/enode/**  ·
                        internal/api/*.go (ui 제외)  ·  go.mod  ·  go.sum
   라우트                internal/api/api.go 의 수 그대로 — **CB0 을 안 건드린다** (R33)
```

---

## 3. 못 박는 겉면 — Step 1 이 먼저 한다

```go
   // internal/transcriptui — 함수 0 개다 (1.2)
   //go:embed card.mjs
   var Files embed.FS
```

```js
   // internal/transcriptui/card.mjs — 부수효과 0.  window 를 안 건드린다
   export function renderEvents(container, events, { open, onToggle })
   export function activeTool(events)          // { name } | null
   export function parseEvents(body)           // 계약 검사.  틀리면 던진다 (R55)
```

**제어판은 인라인 모듈 한 줄로 창에 건다** — `card.mjs` 자체는 부수효과가 0 이라
현황판에서 전역을 안 더럽힌다.

```html
   <script type="module">
     import * as card from "/static/card.mjs";
     window.enodeCard = card;
   </script>
```

**CSP 가 이것을 허락한다** — 제어판은 `script-src 'self' 'unsafe-inline'` 이다 (R25).

---

## 4. 단계 — 열둘

### Step 1 — 겉면과 빈 패키지
- [x] `internal/transcriptui/embed.go` — `//go:embed` 와 `var` 뿐. **함수 0** (1.2)
- [x] `card.mjs` 에 3절의 세 `export` 를 빈 몸으로 세운다
- [x] `go build ./...` 가 서고 `go list ./...` 가 20 이다

### Step 2 — 라우트 둘 (R32 · R33 · R34)
- [x] `ui.go` 가 `/ui/shared/transcriptui/card.mjs` 를 낸다. MIME 이 javascript 다
- [x] `/ui/shared/transcriptui/` 는 **404** — 디렉터리 목록을 안 낸다 (1.3)
- [x] `panel.go` 가 `GET /static/card.mjs` 를 낸다. 보안 헤더 래퍼 안이다
- [x] 두 응답의 **바이트가 같다** (R34 를 재는 시험 하나)
- [x] `internal/api/api.go` 의 `HandleFunc` 수가 **안 늘었다** (R33 · CB0)

### Step 3 — 경계 넷 (R31)
- [x] `boundary_test.go` 의 표에 네 줄 — `transcriptui -> panel · api · store · enode`
- [x] 여덟이 열둘이 되고 전부 초록이다
- [x] 파일 행렬 6.1 의 빈 넷째 줄(`api/ui -> store`)이 섰는지 보고, 안 섰으면 함께 세운다

### Step 4 — 렌더러를 옮긴다 (R36 ~ R41 · U5 의 R18 · R19 · R21 · R22 · R30)
- [x] `evLabel` · `evSummary` · `drawEvent` 를 **글자 그대로** 옮긴다 (R40)
- [x] 전역 둘을 인자로 뺀다 — `open` 과 `onToggle` (1.5)
- [x] 본문은 `textContent` 로만. `innerHTML` 에 하네스 바이트가 0 번 (R54)
- [x] 펼침 열쇠는 `tool_use_id` (R39)
- [x] **그리기 전에** 바닥을 재고 바닥이었을 때만 따라간다 (R53 · U5 의 R21)
- [x] `Date.now` · `new Date` · `fetch` 가 `card.mjs` 에 **0 번** (R37 · R38)
- [x] `activeTool` — 마지막 `tool_use` 의 짝짓기 (R57)
- [x] `parseEvents` — 계약 검사. 모르는 모양이면 던진다 (R55)

### Step 5 — 제어판을 그 모듈 위에 세운다 (답 2 = A)
- [x] `page.go` 에서 그리는 함수 셋이 빠진다
- [x] 인라인 모듈 한 줄로 `window.enodeCard` 에 건다 (3절)
- [x] 기존 인라인 스크립트는 **그대로 둔다.** `onclick` 아홉을 안 걷는다
- [x] 폴링(1초) · 세대 리셋(R20) · 경과(R14) · 잘림 줄 · 링 경로는 `page.go` 에 남는다
- [x] 제어판이 **상태 줄을 얻는다** — 오늘 없는 줄이다 (도메인 5절 · `statusLine`)

### Step 6 — 규율을 함수로 뺀다 (R50)
- [x] `client.mjs` 에서 4초 제한시간 · `AbortController` · 연속 실패 얼리기를 뺀다
- [x] `retryDelay` 는 이미 `export` 다. 그대로 쓴다
- [x] `ObservationClient` 의 동작이 **안 바뀐다** — 기존 `client.test.mjs` 가 그대로 초록

### Step 7 — 폴러 (R44 ~ R51)
- [x] `transcript-poller.mjs` 신규. `setInterval` 2초 — **목록의 5초와 별개** (R46)
- [x] 화면이 `{seq, name, state}` 목록을 넘긴다. 폴러가 상세를 **안 읽는다** (R47)
- [x] `CLAIMED` 만 2초. 끝난 단계는 한 번 (R49)
- [x] URL 은 `?name=` 에 `steps[].id` (R42). `?from=` 이 0 번 (R43)
- [x] 멈추는 신호 둘 — 상세의 `DONE`·`FAILED`, 응답의 `source=sealed` (R48)
- [x] 열쇠 `(run, seq, attempt)`. `X-Enode-Log-Attempt` 가 다르면 안 잇는다 (R44 · R45)
- [x] Run 상세가 안 열려 있으면 요청 0 (R51)

### Step 8 — 카드를 화면에 붙인다 (R52 · R53 · R56 · R58 ~ R61)
- [x] 자리는 **사이드바**의 `stepList` 아래다 (1.4)
- [x] 렌더러를 **주입받는다.** `view.mjs` 에 정적 임포트가 0 (1.1)
- [x] 총 길이가 안 움직이면 DOM 을 안 건드린다 (R52)
- [x] 사건 열은 `replaceContents` 를 **안 탄다** (R53). 나머지는 탄다
- [x] 상태 줄은 `state === 'CLAIMED'` 일 때만 (R56)
- [x] 세 상태를 안 합친다 · 실패해도 마지막 값을 안 지운다 · 빈 `catch` 0 (R58 ~ R60)
- [x] 404 는 카드를 접는다. 총 길이 0 과 다르다 (R61)
- [x] NC-4 · NC-5 · NC-6 을 그린다. NC-6 이 이 유닛이 지는 완료 조건이다

### Step 9 — 원문 토글 (답 7 = A)
- [x] 켤 때 `as=raw` 를 한 번 더 부른다
- [x] 끈 동안 `as=raw` 요청이 **0** 이다
- [x] 원문은 `pre.textContent` 한 줄. **렌더러 밖이다** (R36)

### Step 10 — 시험: 렌더러
- [x] `card.test.mjs` 신규. 디스크 경로로 임포트한다 (1.1)
- [x] 종류 일곱이 각각 그려진다. 모르는 `type` 은 `raw` 로 온다
- [x] `raw` 가 **한 줄**이다 — 본문 JSON 이 카드에 0 번 (R40 · U5 의 R30)
- [x] `innerHTML` 이 이 경로에 0 번이고 태그가 글자로 남는다 (R54)
- [x] 펼침이 `tool_use_id` 로 든다. `line` 이 밀려도 안 따라 밀린다 (R39)
- [x] 바닥이 아니었으면 안 따라간다 (R53 을 재는 줄이다)
- [x] `activeTool` — 짝이 없으면 이름, 있으면 `null` (R57)
- [x] **이 파일이 R14 · R19 · R20 · R21 · R30 의 첫 자동 검사다** (계획 1.1 의 값)

### Step 11 — 시험: 폴러
- [x] 가짜 타이머와 가짜 `fetch` 로 돈다. 네트워크를 안 탄다
- [x] 2초와 5초가 **다른 타이머**다 — 목록을 멈춰도 카드 요청이 난다 (**CB4 를 재는 줄**)
- [x] URL 에 `name=<id>` 가 있고 `from=` 이 없다
- [x] `DONE` 을 받으면 한 번 더 읽고 멈춘다. **안 받으면 계속 돈다** (R48)
- [x] `attempt` 가 바뀌면 든 것을 버린다
- [x] 429 가 `Retry-After` 만큼 민다 — `client.mjs` 와 **같은 함수**를 쓴다 (R50)
- [x] 실패해도 마지막 값이 안 지워진다 (R59)

### Step 12 — 변이와 게이트
- [x] 변이 일곱 (5절). 실측 결과를 옆에 적는다
- [x] `go test ./... -count=1` 전부 초록 · 패키지별 커버리지 80% 하한 통과
- [x] `node --test internal/api/ui/tests/*.test.mjs` 전부 초록
- [x] `go run ./scripts/glyphscan.go` 통과
- [x] `diff 0` 이어야 할 곳 전부 0 (2절)
- [ ] **CB1 의 카드 줄을 사람이 다시 본다** (6절)
- [x] 코드 요약을 쓰고 이 계획의 체크박스를 **실측으로** 채운다

---

## 5. 변이 일곱 — 안 죽으면 시험이 없는 것이다

**계획 단계에서는 목록만 세운다. 결과는 Step 12 에서 옆에 적는다.**

```text
   ①  폴러의 간격을 5000 으로 바꾼다            타이머가 하나가 된다 -> CB4 의 줄
   ②  ?name= 을 "step" 고정으로 바꾼다          빈 카드로 돌아간다 (R42)
   ③  ?from= 에 X-Enode-Log-Bytes 를 넣는다     경계의 줄이 사라진다 (R43)
   ④  상태 줄에서 step.state 갈림을 뺀다        끝난 단계에 「쓰는 중」이 선다 (R56)
   ⑤  raw 의 본문을 그대로 그린다               카드의 절반이 장부가 된다 (R40)
   ⑥  스크롤 판정을 그리기 **뒤로** 옮긴다       언제나 바닥이 된다 (R53)
   ⑦  「총 길이가 세 번 안 움직이면 끝」을 더한다  CB4 가 우연히 초록이 된다 (R48)
```

**⑦ 이 이 유닛에서 가장 중요한 변이다.** 그 규칙을 넣어도 시험이 초록이면
**CB4 가 재려던 동작을 아무도 안 재고 있는 것**이다.

---

## 6. 게이트 — 셋을 진다

```text
   CB4   보인다 (중앙)   사람.  현황판에서 Run 을 열면 카드가 자란다.
                       **목록 폴링을 막아도 자란다** (브라우저 네트워크 탭)
                       원문 토글이 있다.  그래프와 목록을 안 가린다
   CB6   한 장면        사람.  scene-gates 1절의 ① ~ ⑦ 을 끝까지
   CB1   재확인         사람.  **렌더러를 옮기고 나서** 제어판의 카드 줄을 다시 본다
```

**CB1 재확인이 이 유닛의 값이 아니라 대가다.** 이미 서명된 게이트가 뒤에서
빨개질 수 있어서 드는 것이다 (`unit-of-work.md` U8 절).

**CB4 를 재려면 한 나무에 U7 이 있어야 한다.** `main` 의 Mediator 는 노드의 첫
시도(`attempt 0`) 청크를 400 으로 막아 실시간 경로가 죽는다 — 그 한 줄 수정이
`unit/chunk-push` 에 있다. **눈 검증 전에 진행자가 U7 을 병합하거나 그 나무로
빌드한다.** 이 유닛의 코드와는 무관하고 **측정의 조건**이다.

---

## 7. 짓지 않는 것

```text
   파서 · 라우트 · 봉투   U1 · U4 가 닫았다
   from= 오프셋         R43 이 뺐다
   서버의 판정           「쓰는 중」도 「마지막 완전한 줄」도 서버가 안 낸다
   name 기본값 수정      답 5 = A 가 화면에서 닫았다
   새 .pen             진행자가 그린다
   제어판 페이지 재작성     답 2 = A.  onclick 아홉을 안 걷는다
   scene.mjs · identity.mjs   안 만진다
   U6 panel-past        이 회차에 안 지었다.  이 유닛이 대신 안 한다
```

---

## 8. 물음 — 하나

**권장대로 닫혔다 (2026-09-17 · 사용자 「실함대에서만」).**

형식은 `common/question-format-guide.md` 다.

### Question 1 — 데모 모드에도 카드를 보이나

0.3 이 낸 물음이다. `shared/fleet/view.mjs` 를 데모와 실 함대가 같이 쓴다.

```text
   A   실 함대에서만 보인다.  mode !== 'fleet' 이면 카드가 숨는다 —
       askList 가 이미 그 모양이다 (view.mjs:107).  business-rules 잔여 ③ 이 닫힌다
   B   데모에서도 보인다.  FR-7 의 「Run 상세에 단계마다」를 모드로 안 가른다.
       손님이 자기 작업의 하네스 출력을 그 자리에서 읽는다
   C   데모에서는 보이되 원문 토글만 막는다.  절반이라 규칙이 하나가 안 된다
   X   Other (아래 [Answer]: 뒤에 적는다)
```

**권장은 A 다.** `GET log` 가 데모에서 **무인증**이라 (`read` 의 갈림) B 는
공개 화면에 하네스 원문을 올린다. `requirements.md` 5.4 ②가 **「누가 GET log 를
남의 Run 에 부른다 -> 오늘의 권한 모델에서 막히지 않는다」**를 이미 잔여로 적어
뒀고, B 는 그 잔여를 화면으로 끌어올린다.

**A 가 FR-7 을 줄이지 않는다** — FR-7 은 「중앙 현황판」이고 CB4 도 실 함대에서
잰다. 데모는 그 요구가 가리키는 화면이 아니다.

**A 의 대가** — `view.mjs` 에 모드 갈림이 한 줄 는다. `askList` 와 같은 모양이라
새 개념이 0 이다.

[Answer]: A
