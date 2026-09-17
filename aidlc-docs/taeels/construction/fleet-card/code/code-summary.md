# U8 `fleet-card` — Code Generation 요약

```text
   유닛    fleet-card (U8) · 웨이브 W-e (마지막) · 브랜치 unit/fleet-card
   계획    ../../plans/fleet-card-code-generation-plan.md  (단계 열둘 · 변이 일곱)
   설계    ../functional-design/  (R31 ~ R61.  U5 의 R14 ~ R30 도 이 유닛의 규칙이다)
   답      설계 1~8 = 전부 A · 코드 1 = A (카드는 실 함대에서만)
   기준선   69a142a
```

---

## 1. 무엇이 생겼나

```text
   internal/transcriptui/card.mjs      신규 174줄.  그리는 것 한 벌
   internal/transcriptui/embed.go      신규.  //go:embed 와 var 뿐 — **함수 0**
   internal/api/ui/ui.go               /ui/shared/transcriptui/card.mjs 한 자리
   internal/panel/panel.go             GET /static/card.mjs · handleCard.  10 -> **11**
   internal/panel/page.go              그리는 함수 셋이 빠지고 모듈 태그 하나가 들었다
   internal/panel/boundary_test.go     금지 넷 + **봉인 하나**
   .../shared/fleet/transcript-poller.mjs   신규 150줄.  2초 타이머
   .../shared/fleet/client.mjs         규율 셋을 함수로 뺐다 (timeLimit · describeFailure · FREEZE_AFTER)
   .../shared/fleet/view.mjs           카드 구역 · 주입 자리 · 카드 그리기 셋
   .../shared/fleet/fleet.css          카드 스타일.  사건 줄의 클래스 이름은 제어판과 같다
   .../static/fleet/app.mjs            URL 동적 임포트와 폴러 수명

   시험
   tests/card.test.mjs (신규)                9개.  **DOM 규칙의 첫 자동 검사다**
   tests/transcript-poller.test.mjs (신규)    12개.  가짜 타이머 · 가짜 fetch
   internal/api/ui/fleet_test.go             경로 둘 · 디렉터리 404 둘 · 같은 바이트 하나
   internal/panel/panel_test.go              모듈이 서는 것 · 페이지가 함수를 안 든 것
```

**Go 의 새 문장은 라우트 둘뿐이다.** 나머지는 `.mjs` 와 시험이다.

---

## 2. 계획이 예언한 것이 실측으로 그대로 나왔다 — 둘

### 2.1 `internal/transcriptui` 가 커버리지 표에 아예 안 나온다

계획 1.2 가 **「문장이 0 이면 그 패키지가 표에 안 나온다」**로 걸었고, 그래서
`embed.go` 를 함수 0 개로 지었다. CI 와 같은 명령으로 재 봤다.

```text
   패키지 19 -> 20.  표에 나온 것 19.  transcriptui 는 **없다**
   미달 0.  internal/panel 86.0% · internal/api/ui 98.5%
   스킵 게이트  "Action":"skip" 하나인데 "Test" 필드가 없다 —
               패키지 레벨 스킵이라 규칙이 먼저 소비한다.  위반 0
```

**`func Bytes() []byte` 를 지었으면 그 한 문장을 덮는 시험이 이 패키지에 따로
필요했다.** 파일 하나를 내는 데 필요한 것은 변수뿐이었다.

### 2.2 `view.mjs` 에 정적 임포트를 안 적은 것이 하네스를 살렸다

계획 1.1 이 「디스크 경로와 URL 경로가 다르다」를 냈고, 렌더러를 **주입**으로
받게 했다. 실측으로 그 경로를 확인했다.

```text
   시험이 view.mjs 를 임포트하는 길   **전이다.**  직접 임포트하는 시험은 0 이고
                                  demo/tour.mjs · webcam.mjs · submission.mjs ·
                                  gallery-view.mjs 가 view.mjs 를 임포트하며
                                  그 넷을 시험 넷이 임포트한다
```

**계획의 「기존 시험 열둘이 view.mjs 를 임포트한다」는 정확하지 않았다** — 열둘이
아니라 넷이고 전이다. 결론은 같다: 정적 임포트를 적었으면 그 넷이 터진다.

---

## 3. 변이 일곱 — 전부 빨강. **⑦ 이 처음에 살아남았다**

```text
   ①  폴러의 간격을 5000 으로               the card timer is its own      FAIL
   ②  ?name= 을 "step" 고정으로             the log URL carries the step id FAIL
   ③  ?from= 으로 창을 자른다                같은 줄                        FAIL
   ④  statusLine 의 문(live)을 뺀다          the status line has a gate     FAIL
   ⑤  raw 본문을 그대로 그린다                raw stays one line             FAIL
   ⑥  스크롤 판정을 그린 뒤로 옮긴다           follows the bottom only when   FAIL
   ⑦  「총 길이가 세 번 안 움직이면 끝」        **처음에 살아남았다** -> FAIL
```

### 3.1 ⑦ 이 살아남은 이유 — 바퀴가 셋이라 규칙이 발동을 안 했다

계획 5절이 **「⑦ 이 이 유닛에서 가장 중요한 변이다」**로 미리 이름을 붙여
뒀다. 그 시험이 바퀴 셋으로 재고 있었다.

```text
   첫 바퀴    총 길이가 null 에서 10 으로 **움직였다**.  안 움직인 것으로 안 센다
   둘째 · 셋째  안 움직였다.  센 것이 둘
   변이의 문턱  셋.  그래서 넷째 바퀴부터 발동한다 — 시험이 거기까지 안 갔다
```

**바퀴를 다섯으로 늘려 고쳤다.** 시험이 재려던 것은 「폴러가 스스로 끝을
판정하지 않는다」인데, 셋으로는 그 판정이 일어나기 전에 시험이 끝났다.

**이것이 변이를 거는 값이다** — 그 시험은 변이 전에도 초록이었고, 재려던 것을
안 재고 있다는 사실이 변이 하나로만 보였다.

### 3.2 ④ 를 걸려고 함수 하나를 더 뺐다

앞 판은 상태 줄의 문(`step.state === 'CLAIMED'`)이 `view.mjs` 안에 있었고,
**현황판의 그리기를 재는 하네스가 없어 변이 ④ 를 걸 자리가 없었다.**

`statusLine(events, live)` 를 `card.mjs` 로 빼서 닫았다.

```text
   현황판이 넣는 것   step.state === 'CLAIMED'
   제어판이 넣는 것   임대 여부 (wk.has_lease)
   같은 모양인 것     불리언 하나.  무엇이 「돈다」인지는 화면마다 다르지만
                   그 답은 양쪽 다 예/아니오다
```

**한 벌이 하나 늘었고 재는 자리가 하나 늘었다.** 계획 3절의 세 `export` 가
넷이 된 것이 이 자리다.

---

## 4. 계획에 없던 것 — 봉인 하나를 더했다

`boundary_test.go` 에 **금지 넷**(계획 Step 3)과 함께 **봉인 하나**를 더했다.

```text
   금지 넷   transcriptui -> panel · api · store · enode.  이름을 아는 넷만 막는다
   봉인      transcriptui 의 의존에 점이 있는 경로가 0 이다 — 표준 라이브러리뿐
```

**파일 행렬 6.5 가 「잎이어야 하고 그것을 검사기가 진다」로 적었는데 금지 넷은
잎임을 못 잰다** — 이름을 안 적은 다섯째 패키지를 임포트하면 그대로 통과한다.
`internal/transcript` 에 이미 같은 모양의 봉인이 있어 그 루프를 둘로 돌렸다.

---

## 5. 불변식이 어디서 서 있나

```text
   R31   봉인 + 금지 넷        boundary_test.go (열둘 + 봉인 둘)
   R32   panel.go 11 줄       mux.HandleFunc 세면 11
   R33   api.go 18 줄 그대로   CB0 을 안 건드렸다
   R34   같은 바이트          fleet_test.go · panel_test.go 가 embed 와 댄다
   R36 ~ R41  렌더러의 성질    card.test.mjs 아홉
   R42 · R43  URL             the log URL carries the step id (②③ 이 죽는 자리)
   R44 · R45  시도            a changed attempt drops what the card held
   R46 ~ R51  폴링            transcript-poller.test.mjs 열둘
   R52       DOM 아끼기       paintTranscriptCard 의 signature.  **시험 0** (6절)
   R53       스크롤           the view follows the bottom only when (⑥ 이 죽는다)
   R54       textContent      가짜 DOM 이 innerHTML 대입을 터뜨린다
   R55       계약 검사        a wrongly shaped events body is refused
   R56 · R57  상태 줄          the status line has a gate (④ 가 죽는다)
   R58 ~ R61  안 올 때         a failed round keeps the last value · a missing run folds
```

---

## 6. 안 돈 것과 못 잰 것

```text
   ①  **DOM 이 가짜다.**  노드에 브라우저가 없고 이 저장소에 DOM 하네스가 0 이라
      시험 파일 안에서 만들어 썼다.  재는 것은 규칙이고 (무엇을 그리나 · 어디에
      넣나) 실제 배치 · 실제 스크롤 · CSS 는 **CB4 와 CB1 이 사람 눈으로 잰다**

   ②  **R52 에 시험이 0 이다.**  「총 길이가 안 움직이면 DOM 을 안 건드린다」가
      view.mjs 의 paintTranscriptCard 안에 있고, 그 함수를 재려면 DashboardView
      전체를 가짜 DOM 에 세워야 한다.  이 유닛이 거기까지 안 갔다 —
      변이를 못 거는 것을 숨기지 않고 여기 적는다

   ③  **전체 재수신의 값을 안 쟀다.**  R43 이 2초마다 전체를 나르고, U4 가 잰
      N1 에 카드가 없다.  **CB4 에서 처음 밟는다**

   ④  **CB4 · CB6 · CB1 재확인이 아직이다.**  셋 다 사람이다 (6절 아래)

   ⑤  제어판의 상태 줄(`txLive`)이 임대 여부로 열린다.  단계가 끝나고 다음
      단계가 링을 덮기 전 구간에서 임대는 아직 있고 링은 앞 단계 것이다 —
      그 짧은 구간에 「생각 중」이 남는다.  현황판은 step.state 라 안 그렇다.
      **NC-1(경과)이 그 자리에서 자라는 것이 참말이다**
```

---

## 7. 게이트 — 사람이 재야 하는 셋

```text
   CB4   보인다 (중앙)   현황판에서 Run 을 연다 -> 카드가 자란다.
                       목록 폴링을 막아도 자란다.  원문 토글.  그래프를 안 가린다
   CB6   한 장면        scene-gates 1절 ① ~ ⑦
   CB1   재확인         렌더러를 옮긴 뒤 제어판의 카드 줄을 다시 본다
```

**측정의 조건이 하나 있다** — `main` 의 Mediator 는 노드의 첫 시도(`attempt 0`)
청크를 400 으로 막아 실시간 경로가 죽는다. 그 한 줄 수정이 `unit/chunk-push`
에 있으므로 **CB4 를 눈으로 재려면 그 나무가 필요하다.** 이 유닛의 코드와
무관하고 진행자가 병합 순서로 푸는 자리다.

---

## 8. 진행자에게 넘기는 것 — 넷

```text
   requirements.md        「GET log 는 s.auth 뒤다」가 실 함대만이다.
   SECURITY-08            데모는 read() 가 무인증 + 한도.  **코드 답 1 = A 가
                          화면 쪽은 닫았다** — 카드가 실 함대에서만 뜬다.
                          라우트가 열려 있는 것은 그대로 남는다

   component-methods.md   GET log 의 name 기본값이 step 인데 그 기본값이 맞는
                          Run 이 실제로 몇인지 아무도 안 셌다

   N1 의 둘째 자리          R43 이 2초마다 전체를 나른다.  U4 의 N1 측정에 없다.
                          CB4 가 빨개지면 답 8 을 C 로 다시 연다 (서버가
                          「마지막 완전한 줄의 끝」을 헤더로 낸다 — U4 를 다시 연다)

   U5 의 R1               「panel.go 의 HandleFunc 가 10 그대로다」가 11 이 됐다.
                          그 유닛의 문서를 안 고쳤다 — R32 가 이 유닛의 자리다
```

---

## 9. 시험이 남기는 부산물 하나

`cmd/enodectl/probe.lock` 이 시험을 돌리면 바뀐다 (`1094256` -> `1409515`).
이 유닛이 만든 것이 아니고 커밋에 안 실었다. **`go test ./...` 뒤에 그 파일이
`M` 으로 보이는 것은 정상이다** — 되돌리고 넘어간다.
