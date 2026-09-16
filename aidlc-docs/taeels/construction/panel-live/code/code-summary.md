# U5 `panel-live` — Code Generation 요약

```text
   유닛    panel-live (U5) · 브랜치 unit/panel-live · 담당 taeels
   계획    ../../plans/panel-live-code-generation-plan.md
   잰 날    2026-09-16 · 기준선 05ee710
   지는 게이트  CB1 — 사람이 실제 하네스로 본다.  이 문서는 그 앞까지다
```

**잰 것만 적는다.** 안 돈 것과 못 잰 것은 6절이 이름으로 든다.

---

## 1. 무엇이 생겼나

| 파일 | 새것/고침 | 줄 |
|---|---|---|
| `internal/panel/headers.go` | **새 파일** | 42 |
| `internal/panel/headers_test.go` | **새 파일** | 104 |
| `internal/panel/transcript_test.go` | 고침 | +408 |
| `internal/panel/page.go` | 고침 | +199 −6 |
| `internal/panel/transcript.go` | 고침 | +120 −8 |
| `internal/enode/transcript_test.go` | 고침 | +57 |
| `internal/enode/transcript.go` | 고침 | +9 −1 |
| `internal/panel/panel.go` | 고침 | +8 −2 |

```text
   go.mod · go.sum        diff 0.  새 의존 0
   internal/api           diff 0
   internal/transcript    diff 0
   internal/record        diff 0
   handlers.go · view.go   diff 0
   boundary_test.go       diff 0 — 금지 여덟 줄이 그대로 초록이다
   라우트                  api.go 18 (CB0 그대로) · panel.go 10 (R1)
```

---

## 2. 실측이 계획을 고친 것 — 하나. 그런데 그것이 크다

### 2.1 불변식 하나를 시험으로 못 지켜 **구조로 닫았다**

계획 5절의 변이 ⑥ 이 **「`Parse` 에 넘기는 바이트와 `data` 를 다른 읽기에서
가져온다」**였고, Step 8 의 「같은 바이트」 줄이 그것을 잡을 예정이었다.

**안 잡혔다.** 조용한 링에서는 두 읽기가 같은 바이트라 어떤 단언도 안 걸린다.

```text
   1차   사건 수를 비교했다                    변이에서 초록.  아무것도 안 쟀다
   2차   쓰는 고루틴을 띄워 경합으로 쟀다        열 번에 세 번 빨강
   3차   len(Data) == Total 산수로 좁혔다       열 번에 일곱 번 빨강
```

**열에 셋을 놓치는 시험은 게이트가 아니다.** 초록이 「괜찮다」를 뜻하지 않는데
사람은 그것을 괜찮다고 읽는다.

**그래서 시험을 버리고 함수를 갈랐다** — `liveBody(snap, mtime, path)` 가
스냅샷 **하나만** 받는다. 나눠 볼 둘이 없으므로 **그 갈래가 아예 없다.**

```text
   전   핸들러 안에서 ReadRing -> Parse -> 봉투.  둘째 읽기를 넣을 자리가 있었다
   후   핸들러가 한 번 읽어 liveBody 에 넘긴다.  liveBody 는 시계도 파일도 안 만진다
```

**W-b 의 U2 가 배운 것과 같은 모양이다** — 「`Write` 의 패닉 방어는 `recover` 가
아니라 구조다」. 막을 수 없는 것을 시험으로 지키려 하지 말고 **못 일어나게**
만든다. 부작용으로 `liveBody` 가 순수 함수가 되어 **결정적으로 시험된다**
(`TestLiveBodyFillsBothViewsFromTheOneSnapshotItWasGiven` · `TestLiveBodyIsPure`).

---

## 3. 변이 여섯 — 전부 빨강. 셋이 처음에 살아남았다

```text
   ①  truncated 를 언제나 false 로       ReportsAWrappedRing              FAIL
   ②  엄격 부등호를 >= 로                 TreatsAnExactlyFullRingAsWhole   FAIL
   ③  응답에 기간(초)을 싣는다             CarriesAnInstantAndNeverADuration  **처음에 살아남았다**
   ④  캐시 열쇠에서 mtime 을 뺀다          DoesNotServeAStaleBodyToARebuiltRing  **처음에 살아남았다**
   ⑤  헤더를 GET / 에만 건다              SecurityHeadersAreOnEveryResponse  FAIL (둘)
   ⑥  liveBody 안에서 링을 다시 읽는다      FillsBothViewsFromTheOneSnapshot  **구조로 닫은 뒤에야**
```

### 3.1 ③ 이 살아남은 이유 — 기간이 0 이라 `omitempty` 가 키를 지웠다

시험이 금지어 목록(`age_seconds` 등)으로 키의 유무를 봤다. **방금 쓴 링은 경과가
0 이므로** 변이가 더한 `AgeSeconds` 를 `omitempty` 가 통째로 지웠고, 시험은
없는 키를 못 찾아 초록이었다.

**고친 것 둘.** `os.Chtimes` 로 링을 90초 과거로 늙히고, 금지어 목록 대신
**봉투가 이름으로 인정한 키 밖의 수를 전부** 막았다. 뒤쪽이 센 이유는 새로
더해지는 기간 필드가 **이름을 뭐라 짓든** 걸리기 때문이다.

**한 번 헛디뎠다** — 처음에는 「60 에서 120 사이의 수」로 걸렀는데 `total` 이
76 바이트라 그 범위에 들었다. **바이트 수와 초가 같은 자리에 온다.** 범위로는
못 가르고 아는 키를 빼는 쪽이 맞다.

### 3.2 ④ 가 살아남은 이유 — 열쇠가 안 부딪히는 시험이었다

시험이 링을 지우고 다시 만든 뒤 **한 줄을 썼다.** 그러면 `total` 이 0 이 아니라
줄 길이가 되어 **캐시의 `(0, 0)` 과 애초에 안 부딪힌다.** mtime 이 열쇠에 있든
없든 통과한다.

**고친 것** — **길이가 같고 내용이 다른 두 줄**을 쓴다 (`hello` · `world`).
그러면 `generation` 도 `total` 도 앞것과 똑같고 **갈리는 것이 mtime 하나뿐**이다.
시험이 그 전제를 스스로 단언한다 (`len(lineA) != len(lineB)` 면 `t.Fatalf`).

**④ 가 `CarriesAnInstant` 도 함께 빨갛게 한다** — 그쪽이 `os.Chtimes` 뒤 다시
읽으므로, mtime 이 열쇠에서 빠지면 옛 `last_write` 가 캐시에서 나온다.

---

## 4. 게이트 — 진행자가 직접 쟀다

```text
   build · vet         초록          gofmt          0 줄
   glyphscan           112 파일 · 0   전체 시험      초록 · **스킵 0**
   커버리지 (정본 awk)    19 패키지 **미달 0** · 전체 7746/8854 = 87.5%
                       internal/panel  84.9% -> **85.9%**
                       internal/enode  87.0% -> 86.9% (구조체 필드 하나)
   라우트               api.go **18 그대로** (CB0 을 안 건드린다) · panel.go **10** (R1)
   크로스 빌드           windows/amd64 OK · net/http T 6 · crypto/tls T 1
   diff 0 이어야 할 곳    api · transcript · record · go.mod · go.sum ·
                       handlers.go · view.go · boundary_test.go  전부 0
   git status          probe.lock 만 바뀌어 되돌렸다 (기준선의 성질 · 세 웨이브 연속)
```

**`internal/panel` 이 올라간 것이 계획 1.4 의 답이다** — `page.go` 의 증가분
199줄이 문자열 상수라 문장 수에 0 을 더하고, Go 쪽 새 코드는 시험 Step 셋이
덮었다.

---

## 5. 불변식이 어디서 서 있나

```text
   R5 기간 없음        구조체에 초 필드가 0.  변이 ③ 이 잰다
   R6 순수 함수        liveBody 가 시계도 파일도 안 만진다.  TestLiveBodyIsPure
   R7 열쇠 셋          (generation, total, mtime).  변이 ④ 가 잰다
   R9 엄격 부등호       Total > Capacity.  변이 ② 가 잰다
   R10 Go 가 정한다     truncated 가 응답에 실린다.  브라우저가 재계산 안 한다
   R23 한 읽기          **구조로 닫혔다** (2.1).  liveBody 가 스냅샷 하나만 받는다
   R24 ~ R28 헤더      다섯이 모든 응답에.  변이 ⑤ 가 잰다
   R29 경계            boundary_test.go 가 그대로 초록.  ui.securityHeaders 를 안 쓴다
   R27 밖에서 0        TestPanelPageFetchesNothingFromOutside 가 indexHTML 을 센다
```

---

## 6. 안 돈 것과 못 잰 것

```text
   CB1              이 유닛이 진다.  **사람이 실제 하네스로 본다.**  코드 게이트가
                    초록인 것으로 대신하지 않는다 (scene-gates §4)

   DOM 규칙 넷        R14 별개 타이머 · R19 펼침 열쇠 · R20 세대 비우기 ·
                    R21 바닥 따라가기.  **자동 시험이 0 이다** — 제어판의 JS 가
                    page.go 의 문자열 상수 안이라 .mjs 하네스에 못 들어간다
                    (계획 1.1).  **변이를 못 걸었다.**  전부 CB1 이 진다

   실물 하네스        안 돌렸다.  가짜 링 바이트로만 쟀다

   512 KiB 매초 파싱  안 쟀다.  캐시가 침묵에서 0 으로 만드는 것만 시험이 잰다.
                    말이 많은 구간의 비용은 CB1 이 512 KiB 프롬프트로 밟는다

   사건 수천의 DOM    안 쟀다.  같은 프롬프트가 함께 잰다

   경합 시험          **버렸다.**  열에 셋을 놓쳐 게이트가 아니었다 (2.1).
                    그 자리를 구조가 졌다
```

---

## 7. 진행자에게 넘기는 것

계획 6절의 다섯 그대로이고 **하나가 는다.**

```text
   business-rules R23 의   「한 번 읽은 같은 바이트」를 시험으로 지키려 했으나
   재는 법이 바뀌었다       경합이라 못 잰다.  구조로 닫았고 (liveBody 가 스냅샷
                         하나만 받는다) 그 판단의 정본이 아직 이 요약뿐이다
```
