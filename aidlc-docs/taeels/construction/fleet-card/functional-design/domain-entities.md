# U8 `fleet-card` — 도메인 개체

```text
   유닛    fleet-card (U8) · 웨이브 W-e (마지막) · 브랜치 unit/fleet-card
   계획    ../../plans/fleet-card-functional-design-plan.md
   답      1=A · 2=A · 3=A · 4=A · 5=A · 6=A · 7=A · 8=A  (여덟이 전부 권장)
   기준선   10224f8  (main 에 U5 가 병합된 뒤 · PR #46)
```

**이 유닛이 짓는 새 타입은 하나도 없다.** 사건도 봉투도 라우트도 U1 · U4 가
이미 닫았다. 이 유닛이 하는 일은 **이미 있는 것을 한 자리로 옮기고, 그 자리를
두 화면이 같이 쓰게 하는 것**이다.

그래서 이 문서가 적는 것은 타입이 아니라 **자리**다 — 무엇이 어디에 살고,
무엇이 무엇을 들고, 어느 경계를 안 넘나.

---

## 1. 새로 생기는 자리 하나 — `internal/transcriptui`

답 1 = A. **잎 패키지다.** `.mjs` 한 장을 `//go:embed` 로 들고만 있다.

```text
   internal/transcriptui/card.mjs     그리는 것 한 벌
   internal/transcriptui/embed.go     //go:embed card.mjs · Bytes() 하나
```

```text
   금지 넷   transcriptui -> panel · api · store · enode
   왜       제어판과 현황판이 둘 다 이것을 임포트한다.  잎이 아니면 두 화면이
            서로의 의존을 통해 다시 붙는다 — 접은 두 벌이 뒤로 돌아온다
   검사      boundary_test.go 의 표에 네 줄을 더한다 (파일 행렬 6.5)
```

**`go list -deps` 로는 자동으로 초록이다** (의존이 `embed` 뿐이다). 값은 나중에
누가 임포트를 더했을 때 빨개지는 데 있다. 시험 임포트까지 보려면
`go list -test -deps` 다 (파일 행렬 6.4 의 실측).

### 1.1 그 한 장이 지는 것과 안 지는 것

계획 5.2.1 의 「막은 것」이 이 경계를 정했다.

```text
   진다      사건 배열 -> DOM          evLabel · evSummary · drawEvent
             「지금 쓰는 중」 판정       activeTool
             계약 검사                parseEvents.  1.6 이 낸 자리다
             R21 의 스크롤 판정        그리기 전에 바닥을 잰다

   안 진다    원문 토글의 출처와 그 한 줄   pre.textContent = 바이트
             폴링                      화면마다 다르다 (제어판 1초 · 현황판 2초)
             스크롤 컨테이너의 자리       .pen 과 각 화면의 CSS
```

**입력이 사건 배열 하나인 것이 이 패키지가 한 벌로 남는 조건이다.** 원문
문자열까지 받게 하면 함수 안에 갈림이 서고, 두 화면이 그 갈림의 다른 가지만
쓴다 — 한 벌이 함수 안에서 두 벌이 된 것이다.

---

## 2. 카드가 다루는 것 — 다섯

새 타입이 아니라 **이미 선 위에 있는 값들의 이름**이다.

```text
   단계       steps[] 의 한 원소.  GET /v1/runs/{id} 가 낸다
   창         GET .../log 의 한 응답.  본문과 헤더 넷
   사건 배열   transcript.Result.  as=events 의 몸통
   폴러       카드마다가 아니라 **Run 상세에 하나**.  2절 아래 4절
   상태 줄     사건 배열에서 뽑는다.  단계가 CLAIMED 일 때만 있다
```

---

## 3. 단계 — 카드가 무엇으로 URL 을 만드나

답 5 = A. **1.1 의 실측이 이 절의 전부다.**

`model.mjs` 의 계약 검사가 이미 이 모양을 강제한다.

```text
   s.seq       정수 · 1 부터      -> 경로의 {seq}
   s.id        문자열 · 필수      -> 쿼리의 ?name=      **이 줄이 빈 카드를 닫는다**
   s.state     문자열            -> 폴링을 멈추는 신호 (4.2) · 상태 줄의 문 (5절)
   s.attempt   정수 · 0 이상 · 선택 -> 창 열쇠의 한 칸 (3.2)
```

**`steps[]` 에 `name` 이 없고 `id` 가 곧 로그의 이름이다.** `observe.go:54` 의
`SELECT seq, name, ...` 이 `StepView.ID` 에 `steps.name` 을 그대로 싣는다 —
DB 로 확인했고 여러 단계짜리 Run 에서도 같다.

```text
   name 없이         200 · X-Enode-Log-Bytes: 0      빈 카드
   name=<steps[].id> 200 · X-Enode-Log-Bytes: 6785   내용이 온다
```

**404 가 아니라 200 에 0 바이트라 화면이 「에이전트가 아무 말도 안 했다」로
그린다.** 이 유닛이 안 넘기면 카드가 전부 비어 있고, 보는 사람도 짓는 사람도
어디가 틀렸는지 못 짚는다.

### 3.2 시도가 바뀌면 창이 리셋된다

`attempt` 가 선택 필드인 것이 값이다 — 없으면 0 으로 읽는다. 진행 파일은
시도마다 다른 파일이고 `dropOlderAttempts` 가 앞 시도를 걷는다 (U3).

**그래서 카드가 든 것의 열쇠는 `(run, seq, attempt)` 셋이다.** 셋 중 하나가
바뀌면 든 것을 버리고 처음부터 받는다. U5 의 R20(세대가 바뀌면 비운다)과 같은
자리이고, 링의 `generation` 자리에 여기서는 `attempt` 가 선다.

---

## 4. 창 — `GET .../log` 의 한 응답

답 8 = A. **매 폴링마다 처음부터 다 받는다.** `?from=` 을 안 쓴다.

```text
   경로     GET /v1/runs/{run}/steps/{seq}/log?name=<id>&as=events
   몸통     transcript.Result 통째.  U5 의 봉투가 transcript 키에 담은 것과 같은 타입
   헤더     X-Enode-Log-Bytes     총 길이.  NC-6 이 이 값이 움직인 시각을 든다
           X-Enode-Log-Source    progress · sealed.  NC-5 · 멈추는 신호 (4.2)
           X-Enode-Log-Attempt   서버가 아는 시도.  3.2 의 열쇠와 댄다
           X-Enode-Log-Capped    상한에 닿았으면 1.  NC-4
```

**`from=` 을 안 쓰는 이유가 이 유닛에서 가장 비싼 값이다.** 창 `[0, N)` 과
`[N, M)` 을 따로 받으면 **N 을 가로지르는 줄이 양쪽에서 다 빠진다** — 앞 창에서는
끝이 안 닫힌 조각(`partial`)이고 뒤 창에서는 머리가 잘린 조각(`head`)이라 파서가
둘 다 버린다.

```text
   치르는 값   바이트.  진행 파일이 상한까지 자라면 2초마다 그만큼 나른다
   사는 값     **조용히 사라지는 줄이 0 이다.**  이 팩은 잘림을 값으로 낸다
              (NC-2 · truncated · capped).  from= 은 그 성질을 정확히 어긴다
```

**바이트와 DOM 을 가른다** — 받는 것이 매번 전체라는 뜻이고, `X-Enode-Log-Bytes`
가 안 움직였으면 DOM 을 안 갈아 그린다 (계획 5.2.1 ②).

### 4.1 원문 토글은 다른 쿼리다

답 7 = A. 토글을 켤 때 `as=raw` 로 한 번 더 부른다.

```text
   평소       as=events 하나.  바이트가 안 는다
   켠 동안     as=events 와 as=raw 둘.  2초마다 둘 다
   안 하는 것  사건에서 원문을 되짓는 것.  Event.Text 가 종류마다 다른 것을 들어
             raw 말고는 줄 전체가 안 나온다 — 실측으로 불가능하다
```

**U5 와 갈리는 자리다.** 제어판은 한 응답에 `data` 를 함께 받는다 (R23 — 링이
감기므로 두 번째 읽기는 다른 창이다). 진행 파일은 안 감기므로 두 쿼리가 같은
바이트의 앞뒤를 본다. **갈리는 것을 기록하고 한 벌에 안 넣는다** (1.1).

### 4.2 폴러 — Run 상세에 하나

답 3 = A · 4 = A. **`ObservationClient` 밖의 작은 것 하나다.**

```text
   간격      2초.  목록의 5초와 별개 타이머다 (FR-7 · CB4)
   도는 것    state 가 CLAIMED 인 단계만.  보통 한 Run 에 하나다
   한 번만    끝난 단계는 한 번 읽고 멈춘다
   규율      429 의 Retry-After · 4초 제한시간 · AbortController · 연속 실패 얼리기
            **client.mjs 에서 함수로 빼 쓴다.**  두 벌로 안 만든다
```

**`GET log` 는 데모 모드에서 무인증 + 한도다** (`api.go:82` 의 `read`). 카드가
그 모드에서도 돌므로 폴러가 그 규율을 진다 — `requirements.md` SECURITY-08 이
「`s.auth` 뒤다」로 한쪽 모드만 적었고, 이 유닛이 그 반쪽을 진다.

**방향이 한쪽이다** — 폴러는 상세 자원을 구독하지 않는다. 화면이 그릴 때마다
`{seq, name, state}` 목록을 넘기고 폴러는 시각만 진다. 자세한 것은
`business-logic-model` 2절.

---

## 5. 상태 줄 — 「지금 쓰는 중」

답 6 = A. 사용자가 It's a Plan 의 같은 화면을 보고 들인 것이다.

```text
   문        step.state == CLAIMED 일 때만 이 줄이 있다
   판정      마지막 tool_use 의 id 에 짝이 되는 tool_result 가 없으면 「<이름> 쓰는 중」
            짝이 다 있으면 「생각 중」
   출처      사건 배열 하나.  서버를 안 건드린다
```

**짝짓기는 파서가 이미 해 놨다** — `Event.ID` 가 `tool_use_id` 이고 U5 의 접기가
그 열쇠를 쓴다 (R19).

**끝난 단계에 짝 없는 `tool_use` 가 남아 있을 수 있다** — 도구가 도는 중에 단계가
죽으면 그렇다. 그때 「쓰는 중」을 세우면 화면이 끝난 것을 도는 것으로 그린다.
**`step.state` 가 그 줄을 닫는다.**

**제어판도 이 줄을 얻는다** — 한 벌의 덤이다. 오늘 제어판에는 경과(NC-1)만 있어서
조용한 구간에 멈춘 건지 생각 중인지 화면이 말을 안 한다.

---

## 6. 라우트 — 둘이 는다

답 1 = A · 2 = A. **같은 바이트를 두 서버가 낸다.**

```text
   현황판   GET /ui/shared/transcriptui/card.mjs      ui.go 가 낸다
   제어판   GET /static/card.mjs                      panel.go 가 낸다
```

```text
   CB0 에 미치는 것   0.  CB0 은 internal/api/api.go 의 HandleFunc 를 센다.
                    현황판 쪽은 ui.Handler() 안이라 그 수가 안 는다
   U5 의 R1 이 바뀐다  「panel.go 의 HandleFunc 가 10 그대로다」 -> 11 이다.
                    U8 이 그 줄을 바꾸는 유닛이라고 business-rules 가 적는다
```

**제어판 페이지는 `<script type="module" src="/static/card.mjs">` 하나를
더하고 기존 인라인을 그대로 둔다** (답 2 = A). 제어판의 CSP 가
`script-src 'self' 'unsafe-inline'` 이라 같은 출처의 모듈이 그대로 선다 (R25).
모듈이 `export` 한 것을 인라인이 부를 수 있게 `window` 에 건다.

---

## 7. 안 짓는 것

```text
   파서            internal/transcript 를 그대로 쓴다.  사건을 고치거나 버리지 않는다
   라우트 · 봉투     U4 가 닫았다.  이 유닛은 GET 만 한다
   from= 오프셋     답 8 = A 가 뺐다 (4절)
   서버의 판정       「쓰는 중」을 헤더로 내는 것.  답 6 = A 가 화면에서 닫았다
   name 기본값      서버의 step 을 고치는 것.  답 5 = A 가 화면에서 닫았다
   새 .pen         진행자가 그린다 (CLAUDE.md 의 layering).  S2 의 확장이다
   scene.mjs       그래프 배치.  카드가 그래프를 안 가리는 것은 자리의 문제다
   identity.mjs    함대 격자의 카드 면.  이 유닛은 Run 상세만 만진다
```
