# obs — 규칙 · 검증 · 완료 조건

AI-DLC Construction · obs 유닛(W0 · CP1)의 Functional Design 산출물 하나다.
개념은 `domain-entities.md`, 흐름은 `business-logic-model.md`. 여기는 **값이
나쁠 때 무엇이 일어나는가**와 **언제 이 유닛이 끝났다고 말할 수 있는가**다.

---

## 1. 인증과 식별

```text
   읽기 셋은 기존 s.auth 를 거친다           Authorization: Bearer <token>
   토큰이 틀리면 401.  기존 표면과 같은 규칙
   X-Enode-Principal 은 읽지 않는다          이 유닛은 식별로 아무것도 안 가른다
```

`GET /v1/nodes` · `GET /v1/runs` 는 Mediator 토큰으로 보호한다
(`enode-features.md` 3.1.1 보안 요구 · SECURITY-12). 별도 운영 권한을 만들지
않는다 — MVP 는 신뢰 경계가 하나다(`ADR-010`).

**데모 인스턴스는 읽기 셋만 무인증이다** (답 4=A · `business-logic-model.md` §7).
가둠 셋(`decisions.md` §8.4)의 「브라우저에 실 토큰 없음」이 그 근거이고, 토큰을
페이지에 박는 길이 그것을 깬다. obs 가 여는 쓰기(`POST /v1/nodes`)는 잠긴 채다 —
demo-back 이 자기 제출 라우트를 무인증으로 여는 것은 별개이고 allow-list 가 지킨다
(`business-logic-model.md` §7.4).

**그리고 이것은 정본을 벗어난다.** `ADR-065` 2절이 「인증은 다른 Mediator 표면과
같은 Bearer 토큰을 쓴다」로 못 박았고 `enode-features.md` 3.1.1 의 보안 요구도
같다. 벗어나는 근거는 팩의 데모 전환(S0 자체가 없다)이지만, `canon.md` 서열대로면
`enode-design` 이 이긴다. **이 유닛은 A 로 가되 그 어긋남을 준수로 적지 않는다** —
근거와 되돌리는 값은 `business-logic-model.md` §7.1, 노출되는 값의 목록은 §7.2,
정본 개정 여부는 진행자의 것이다.

정적 파일(`GET /ui/`)은 **무인증**이고 이 유닛 밖이다.

---

## 2. 입력 검증 (SECURITY-08)

새 표면이라 검증을 붙인다. 검증할 것은 **질의 문자열 넷**과 **광고 본문의 정책
값 하나**다.

### 2.1 limit (`GET /v1/runs`)

```text
   없음 · 빈 문자열(?limit=)   100 (decisions.md 2절의 기본값)
   1 ~ 1000                   그대로
   1000 초과                   1000 으로 자른다
   0 · 음수 · 수가 아님          400 invalid limit
```

**상한 1000 은 이 문서가 정한다.** 팩이 기본값(100)만 정하고 상한을 안 정했다.
1000 인 근거 — 이 표면이 답하는 것은 「지금 무엇이 도는가」이고, 그보다 큰 값을
받는 것은 지난 것을 뒤지는 일이라 Record 의 자리다.

**넘으면 자르고 거부하지 않는다.** 페이지네이션이 없으므로 「전부 받았다」를
보장하는 값이 애초에 없다 — 기본값 100 이 이미 그렇다. 거부하면 상한이 서버와
화면 두 곳에 살고, 어긋나면 화면이 빈 목록을 받는다. **자른 것이 응답에 안
보인다** — 회복 수단도 없다(페이지네이션 부재). mcp 가 `runs.list` 도구 설명에
그 사실을 싣는다.

**0 과 음수는 거부한다.** 「0줄을 달라」는 요청은 오타이지 뜻이 아니다. 빈
문자열은 거부하지 않는다 — 인자를 붙였다가 지운 흔적이지 값이 아니다.

**빈 문자열의 규칙은 넷에 다 같다.**

```text
   ?limit=    미지정.  기본 100
   ?since=    미지정.  파싱을 안 시도한다 — 400 invalid since 가 아니다
   ?state=    미지정.  좁히지 않는다
   ?work=     미지정.  좁히지 않는다
```

넷을 같게 두는 이유 — **필터 칸을 비우는 것이 화면의 가장 흔한 조작**이고, 넷이
서로 다르게 굴면 ui 가 인자마다 예외를 짓는다. 「지정됐다」의 뜻은 **인자가 왔고
값이 비어 있지 않다**로 하나다. `business-logic-model.md` §3 의 SQL 에서
「지정되면」이 가리키는 것이 이 판정이다.

**길이 상한 위반은 400 인데 `limit` 초과는 자른다.** 비대칭이고 의도다 —
`limit` 은 「얼마나 볼까」라 좁혀도 뜻이 남지만, `state`·`work` 는 **찾는 값
자체**라 잘라 내면 다른 것을 찾은 것이 된다.

### 2.2 since (`GET /v1/runs`)

```text
   RFC 3339        created_at >= since (포함 하한)
   빈 문자열        미지정.  좁히지 않는다 (§2.1 의 넷째 규칙)
   그 밖의 값        400 invalid since
```

**포함이다.** 정본이 「`created_at` 하한」이라고만 적어 경계가 열려 있었다.
포함으로 못 박는 이유 — 같은 초에 만들어진 Run 이 폴링 사이에 사라지지 않는다.

시각 형식을 하나로 못 박는다. `created_at` 이 `timestamptz` 이므로 표준시가
붙은 형태만 받는다 — 표준시 없는 값을 서버 지역시로 짐작하면 경계에서 조용히
어긋난다.

### 2.3 state · work (`GET /v1/runs`)

```text
   그대로 등호 비교에 넘긴다.  길이 상한 200.  넘으면 400
```

**`state` 의 어휘를 이 유닛이 검사하지 않는다.** `schema.sql` 이 `runs.state` 에도
`steps.state` 에도 CHECK 를 안 거는 것과 같은 이유다 — 어휘가 늘 때 마이그레이션을
강요하지 않는다. 그리고 이 유닛에는 더 직접적인 이유가 있다: **queue 유닛(W1)이
`QUEUED` 를 더한다.** 여기서 어휘를 박으면 W1 이 W0 을 다시 열어야 하고, 그것은
토대 유닛이 해서는 안 되는 일이다.

모르는 값이 오면 빈 목록이 나간다. 그것이 오답이 아니다 — 「그 상태인 Run 이
없다」가 참이다.

### 2.4 모르는 질의 인자 — 라우트마다 다르다

```text
   GET /v1/runs     무시한다
   GET /v1/nodes    400 unknown query parameter
```

**갈라 두는 것이 정본이다.** `mediator-api.md` 는 `GET /v1/nodes` 에 대해
「필터를 받지 않는다. `?free=true` 같은 질의는 `400` 이다」라고 적었고, 그것이
`ADR-065` §1.1 이 막으려던 바로 그 모양이다 — 여유 질의를 만들지 않겠다는 결정이
표면에서 눈에 보이는 자리가 여기다. 「좁히기만 하므로 무시해도 된다」는 논거는
필터가 넷인 `/v1/runs` 의 것이고, 필터가 **0** 인 `/v1/nodes` 로 옮겨가지 않는다.

**화면에 넘기는 제약 한 줄** — `GET /v1/nodes` 에 캐시 무력화용 `?t=...` 같은
인자를 붙이면 `400` 이다. 폴링이 캐시를 피해야 하면 헤더로 한다.

### 2.5 광고 본문의 정책 값 (POST /v1/nodes)

답 1=C 로 이 유닛이 본문을 읽어 DB 에 쓰게 됐다. `decisions.md` 3절이 새 표면에
요구한 「입력 검증(`limit` 상한 · **정책 값 enum**)」의 뒤쪽이 이 자리다.

```text
   "" · graceful · at-boundary   그대로 저장한다
   그 밖의 값                     "" 로 접어 저장하고 로그에 남긴다.  400 이 아니다
   policy 키 자체가 없음           "" 다.  광고는 매번 전부다 (ADR-012 · ADR-017)
```

**`400` 을 안 내는 이유** — 광고는 하트비트를 겸한다(`ADR-016`). 정책 값 하나로
광고 전체를 거절하면 노드가 함대에서 사라지고, 증상이 원인보다 훨씬 크다.

**검사가 필요한 이유** — queue 의 `DrainingNodes` 가 이 열을 `map[string]bool` 로
접는다. 어휘 밖 문자열이 들어오면 그 노드가 매칭 후보에서 **조용히 빠지고**,
증상은 「능력은 있는데 계속 QUEUED」이며 원인은 `GET /v1/nodes` 의 그 문자열을
눈으로 봐야만 보인다.

DB CHECK 를 안 거는 것과 이 검사는 다른 것이다. 앞은 스키마의 경직을 피하는
결정이고, 뒤는 `decisions.md` 3절이 이름으로 지목한 애플리케이션 검증이다.

---

## 3. 응답 규약

```text
   빈 결과            200 + 빈 배열.  null 도 404 도 아니다
   observed_at        행이 0 이어도 언제나 있다 (business-logic-model §2)
   키는 언제나 있다     읽기 라우트 둘의 규칙.  값이 없으면
                        null          verdict · ended_at · lease
                        빈 문자열      submitter · draining · instance · work_id
                        빈 배열        assigned · capabilities · nodes · runs
   시각                RFC 3339 (encoding/json 의 time.Time 기본)
   정렬                runs 는 created_at 내림차순 · nodes 는 node_id 오름차순
```

**`assigned` 는 빈 배열이다.** `null` 도 키 생략도 아니다 — 정본 예시가 QUEUED 행에
`"assigned": []` 로 적었고, CP2 의 「QUEUED 행이 어느 카드에도 안 얹혀 있다」를
화면이 이 값으로 판정한다. `StepView.Needs` 가 빈 배열을 언제나 내보내는 이유와
같다: 안 내보내면 읽는 쪽이 기본값 규칙을 추측한다.

**`nodes` 를 `node_id` 로 정렬하는 이유**는 같은 함대를 두 번 읽으면 같은 순서
여야 하기 때문이다. 정렬을 안 주면 화면의 카드가 폴링마다 자리를 바꾼다.

**`GET /v1/runs/{id}` 의 `requires` 는 이 규칙 밖이다** — 기존 표면이라 글자를
안 바꾸는 것이 먼저이고, `omitempty` 로 붙어 `getRun` 에서만 채워진다
(`business-logic-model.md` §4). 그래서 이 키가 없다는 것은 **계약 읽기 실패**이고,
그 사실은 같은 응답의 `warnings` 에 한 줄로 실린다 — 생략과 실패가 한 글자가
되지 않게 하는 것이 그 한 줄의 값이다.

**`POST /v1/nodes` 의 `drain` 도 이 규칙 밖이지만 반대 방향이다** — 기존 표면인데
**언제나 싣는다.** 생략하면 「해제됐다」와 「이 필드를 모르는 중앙이다」가 한 글자가
되고, 그것을 받아 Worker 에 넘기는 것이 drain(W2)이다(`domain-entities.md` §7.3).
요청 본문의 `policy` 가 `omitzero` 인 것과 다른 자리다.

---

## 4. 오류

기존 `fail(w, code, reason)` 규약을 그대로 쓴다 — 본문은
`{"error":{"code":..,"reason":..}}`. 문자열은 영어다(`CONVENTIONS.md` 2.1).

```text
   상황                                          코드   reason
   ───────────────────────────────────────────   ────   ──────────────────────────
   토큰 없음 · 틀림                               401    기존 s.auth
   limit 이 0 이하 · 수가 아님                     400    invalid limit
   since 가 RFC 3339 아님                        400    invalid since
   state · work 가 200자 초과                     400    invalid filter
   GET /v1/nodes 에 질의 인자가 옴                 400    unknown query parameter
   저장소 조회 실패                               503    query failed
   GET /v1/runs/{id} 의 id 가 없음                404    no such run (오늘 그대로)
```

`GET /v1/nodes` 와 `GET /v1/runs` 에는 `404` 가 없다 — 목록은 비어 있어도 있다.

**코드는 기존 규약이지만 뜻이 넷 는다.** `mediator-api.md` §1.2 의 코드 표는
`400` 을 「계약이 문법적으로 틀렸다」로, `503` 을 「Mediator 가 아직 준비 안 됨」
으로 적었다. 위의 다섯 중 `GET /v1/nodes` 의 `unknown query parameter` 만 정본에
글자가 있고(그 절이 「`?free=true` 같은 질의는 `400` 이다」라 적었다), 나머지 넷은
**질의 인자 검증**과 **실행 중 질의 실패**라는 새 뜻이다. 상식적인 배치이지만
「기존 규약 그대로」가 덮는 범위가 아니므로 적어 둔다 — 정본의 코드 표를 넓히는
것은 진행자의 일이고, 넓히지 않으면 다음 사람이 그 표로 검증할 때 갈린다.

**계약을 못 읽어도 `GET /v1/runs/{id}` 는 `200` 이다.** `requires` 가 빠진 채로
나가고, **그 사실이 `warnings` 에 한 줄로 실린다**(§3). 로그에도 남는다. 관측이
조회를 막으면 안 된다 — 오늘 `Steps` 실패가 같은 규칙으로 다뤄지고 있다.

**광고의 정책 값이 나빠도 `POST /v1/nodes` 는 오늘 그대로다** (§2.5).

로그 메시지와 테스트 이름의 문자열은 Code Generation 이 짓는다. 영어로 짓고,
토큰이나 정책 값 원문을 찍지 않는다(SECURITY-03).

---

## 5. 안 하는 것 (선을 못 박는다)

```text
   판정                「이 노드는 왜 안 맞나」를 서버가 안 낸다 (ADR-069 §5)
   매처                관측 경로에서 안 돈다 (ADR-065)
   nodes 의 필터        안 준다.  free · available · 순위도 없다 (ADR-065 §2)
   principal           응답에도 필터에도 없다 (constraints §4 의 제외 목록)
   node 필터            runs 에 없다.  화면이 assigned 로 거른다 (3.1.3)
   페이지네이션          limit 뿐.  뒤로 걸어갈 수단이 없다 (domain-entities §4)
   카드의 단계           GET /v1/nodes 를 안 넓힌다.  두 홉이다 (domain-entities §2.2)
   submitter 의 상세     목록에만.  화면이 목록에서 이어 붙인다 (domain-entities §3.2)
   TLS                 constraints §6.  기존 코드 사실로 기록만
```

**`sandbox` 는 새 필드를 안 만든다 — 자리가 이미 있기 때문이다.**
`decisions.md` §8.5 가 「있는 값을 읽는다」고 한 그 값의 통로는 노드 설정의
`labels` 다: `Local.Labels` 가 광고 속성에 합쳐지고(탐지가 만든 키만 이기며
`sandbox` 는 탐지 목록에 없다) `nodes.capabilities` 에 그대로 앉는다. 이 유닛이
`capabilities` 를 **그대로** 내므로 데모 노드 설정에 `labels: {sandbox: <값>}`
한 줄이면 `capabilities[].attrs.sandbox` 로 화면에 닿는다 — **코드 변경 0.**
값이 아직 아무도 안 실을 뿐 자리는 있다. 화면은 노드 전체가 아니라 `capabilities`
항목마다 붙는다는 것만 알면 된다. 출처 결정은 ui(runixs)의 것이고, 이 결정이
어느 쪽으로 나든 **obs 를 다시 열 이유는 없다.**

**ADR-068 의 `At` 은 자리가 없다 — 진행자에게 남기는 표시다.** panel 의 「탐지
능력 표시」는 **언제 잰 값인가**를 함께 보여야 하는데(`enode-features.md` 3.1.2),
그 시각은 데몬 메모리의 `Capabilities{Caps, At}` 에만 있고 `ADR-068` 이 광고에
안 싣기로 못 박았다. `seen_at` 은 광고 시각이지 탐지 시각이 아니다 — 기본 주기가
60초 대 5분이라 최대 5분 어긋난다. 이 미정이 「광고에 싣는다」로 닫히면
`advert.go` · `schema.sql` · `UpsertAdvert` · `NodeView` **넷이 전부 이 유닛의
표면**이라 W3 에 함께 열린다. 담당은 panel(nacl1119)이고, 진행자가 이 미정을
닫을 때 그 비용을 보고 닫으라는 표시다.

---

## 6. security-baseline 준수

`decisions.md` 3절의 처리를 이 유닛에 적용한 결과다. **답 1=C 로 이 유닛은
읽기 전용이 아니다** — 본문을 받아 DB 에 쓰는 자리가 하나 있고, 그것이 아래
판정의 전제다.

```text
   규칙            이 유닛에서                              판정
   ────────────   ─────────────────────────────────────   ───────────
   SECURITY-01    평문 HTTP.  기존 표면과 같다              기록만 (N/A)
   SECURITY-03    새 로그에 토큰·정책 원문을 안 찍는다        준수
   SECURITY-07    새 리스너가 없다.  기존 :8080 에 붙음       기록만 (N/A)
   SECURITY-08    읽기 셋에 질의 인자 검증 (2.1~2.4)         준수
                  광고 본문의 정책 값 enum 검증 (2.5)
                  객체 수준 권한은 해당 없음 — ADR-015 §1
   SECURITY-12    실 함대는 Mediator 토큰 하나.               조건부
                  새 자격증명 없음                          (아래)
                  데모 인스턴스는 읽기 셋이 무인증이다 —
                  ADR-065 2절을 벗어난다 (§1)
```

**SECURITY-12 를 무조건 준수로 닫지 않는다.** 새 자격증명을 안 만든 것은 참이지만
읽기 셋 셋이 데모 인스턴스에서 인증을 안 거치고, 그것이 이 규칙이 요구하는 값이다.
예외를 SECURITY-08 칸에 적어 두 칸을 다 초록으로 보이게 하면 승인 심사가 그 표를
근거로 읽는다. **범위는 데모 인스턴스뿐이고 실 함대는 무변경이다** — 되돌리는 값과
진행자가 정할 것은 `business-logic-model.md` §7.1 에 있다.

공격면은 셋이다 — 질의 문자열 넷(`GET /v1/runs`) · 질의 인자의 존재 자체
(`GET /v1/nodes`) · 광고 본문의 정책 값 하나. 2절이 그 셋을 전부 적는다.

---

## 7. 완료 조건

### 7.1 CP1 — obs 가 재는 것

CP1 은 obs(API)와 ui(화면)가 나눠 진다(스토리 맵 §1). **obs 가 지는 것은
`scene-gates.md` §3 의 curl 두 줄**이고, 화면 다섯(S0 · S0b · 세션 저장소 · S1 ·
S1b)은 ui 유닛이 진다.

```text
   재는 것                                                무엇이 초록인가
   ────────────────────────────────────────────────────   ──────────────────────
   curl -H "Authorization: Bearer $T" $M/v1/nodes         노드 여럿이 나온다
     | jq '.nodes[] | {label, lease, draining}'           임대된 노드에 lease 가
                                                         있고 not_after 가 뒤로
                                                         밀린다
                                                         draining 노드가 값을 낸다
   curl -H "Authorization: Bearer $T"                     상태 셋의 Run 이 나온다
     "$M/v1/runs?limit=5"                                 (아래) · 최신이 앞이다
     | jq '.runs[] | {run_id, state}'
```

**인증 헤더는 정본 그대로다.** 게이트를 돌리는 사람은 이 유닛을 구현하지 않은
사람이고(`scene-gates.md`), 실 함대 Mediator 에는 토큰이 걸려 있다. 데모 인스턴스의
무인증은 §1 의 예외이지 이 명령의 모양이 아니다.

**빈 출력은 초록이 아니라 잰 것이 없는 것이다** (`scene-gates.md` §3). 함대는
45초마다 광고를 다시 보내는 노드가 있어야 산다 — 광고는 180초에 만료된다.

#### 「상태 넷」은 W0 에 셋뿐이다 — 병합 조건을 여기서 못 박는다

`scene-gates.md` 의 CP1 판정 재료는 「노드 여럿 · 임대 하나 · draining 하나 ·
**상태 넷의 Run**」이다. **넷째가 없다.**

```text
   저장소가 실제로 저장하는 Run 상태
   ───────────────────────────────────────────────────────────────────
   RUNNING      api.go 가 쓴다
   SUCCEEDED    reap 의 정산이 쓴다
   FAILED       같다
   VERIFYING    한 함수 안에서 곧바로 덮인다 — curl 로 못 잡는다
   RESOLVING    코드 어디에서도 안 쓴다 (grep 0건)
   ALLOCATING   같다.  enode-features 3.2.1 이 「그 한 함수 안의 순간」이라 적었다
   QUEUED       queue(W1)가 만든다.  같은 절이 「실제로 저장되는 첫 중간 상태다」
```

**넷째는 `QUEUED` 이고 그것은 W1 이다.** obs 가 그 절반을 재는 것은 불가능하다.

**그래서 이 유닛의 병합 조건은 「`scene-gates.md` §3 의 curl 두 줄 + CP0」이다.**
`curl "$M/v1/runs?limit=5"` 가 **상태 셋**(RUNNING · SUCCEEDED · FAILED)을 최신순으로
내면 obs 지분은 초록이다. 넷째 상태는 CP2(W1)가 잰다 — §7.2 의 표가 그 자리다.

**CP1 이 통째로 초록이 되는 것은 W0 이 아니다.** 화면 다섯이 ui 몫인데 ui 는 obs 가
병합되어야 착수한다. 순환이므로 게이트 전체를 병합 조건으로 두면 아무도 못 선다.
**이 부분 통과를 진행자가 승인한다** — `scene-gates.md` §4 의 「유닛 밖 이유로
빨간 것」인 보류와 다르다. 여기는 「아직 안 만든 유닛」이 이유다.

덧붙여 `scene-gates.md` 는 데모 전환 뒤 **CP0~CP4 를 전환 전 기록**으로 두고
완결성 판정을 CP10(데모 완주)에 넘겼다. CP1 은 여전히 유효한 수용 기준이지만
**이 유닛의 완결성 정본은 아니다.**

#### draining 하나를 만드는 절차

`draining` 하나는 답 1=C 로 이 유닛 안에서 만들 수 있다 — 광고에 `policy.drain` 을
실어 보내면 열에 복사되고 응답에 나온다. drain 유닛(W2)을 기다리지 않는다.
**그런데 그냥 한 번 쏘면 안 된다.**

```text
   실 데몬과 node_id 를 나눈다   같은 node_id 로 데몬이 돌면 그 데몬의 다음
                               하트비트가 draining 을 지운다.  광고는 매번
                               전부이고 W0 의 enode 는 policy 를 안 싣는다
   180초 안에 다시 쏜다          광고는 180초에 만료된다.  합성 노드는 스스로
                               갱신하지 않으므로 사람이 루프를 돈다
   임대는 이 노드에 안 걸린다     데몬이 없으므로 일을 안 받는다.  「임대 하나」는
                               실 데몬 노드에서 나온다 — 다른 노드다
```

즉 시연 함대는 **실 데몬 여럿 + 합성 draining 노드 하나**다. 이 절차를 안 적으면
게이트 집행자가 빈 `draining` 을 보고 obs 의 결함으로 적는다 — 그것이
`scene-gates.md` §4 의 보류로 오분류될 자리다.

### 7.2 CP2 에 넘기는 재료 — obs 가 세우고 CP2 가 잰다

`requires` · `as` · `chosen` 은 **CP2 의 수용 기준**이다 — `scene-gates.md`
§2.1 의 CP2 항목(「QUEUED 행을 누르면 무슨 능력을 기다리는지」 · 「안 간 갈래가
SKIPPED 이고 chosen 이 갈린다」)이고 `canon.md` 도 `ADR-069` 를 CP2 에 걸었다.
소유 유닛은 `queue + ui` 이고 obs 는 협력이다. **CP1 초록으로 이것을 닫지 않는다.**

```text
   obs 가 W0 에서 실증하는 것                         CP2 가 W1 에서 잰다
   ────────────────────────────────────────────    ──────────────────────────
   RUNNING Run 의 GET /v1/runs/{id} 에               QUEUED 행을 눌러 기다리는
     requires · as 가 나온다                          능력을 화면에서 본다
   갈래 계약 하나로 chosen 이 true/false 로 갈린다     안 간 갈래가 화면에서 갈린다
     (dispatch.go 가 이미 SET chosen = true 를 쓴다)
   거절된 Run 의 assigned 가 [] 로 나온다             QUEUED 행이 어느 카드에도
     (null 도 키 생략도 아니다)                        안 얹혀 있다
   상태 셋(RUNNING·SUCCEEDED·FAILED)이 나온다         넷째인 QUEUED 가 나온다
```

**`assigned` 를 이 표에 넣는 이유** — CP2 의 판정이 그 값으로 서는데 값을 넣는 것은
queue 다. obs 는 **읽기 쪽이 `[]` 를 낸다는 것**까지만 W0 에서 보이고, 쓰는 쪽이
jsonb `null` 을 넣으면 그 보장이 깨진다(`domain-entities.md` §3.1). 두 쪽을 갈라
적지 않으면 queue 가 이미 증명된 줄로 믿는다.

**QUEUED 상태는 W1 이전에 존재하지 않으므로** obs 가 그 절반을 재는 것은 불가능
하다. 이 표를 안 적으면 queue 가 W1 에 도착해 이미 증명된 줄로 믿는다.

### 7.3 CP9 에 지는 지분 (3.4.4)

`submitter` 는 `demo-back(쓰기) + obs(컬럼·목록) + ui(카드)` 로 갈린다
(스토리 맵 §2). obs 의 지분은 **실 함대 Run 으로도 잴 수 있다** — 값이 `""` 라도
키는 나가야 하기 때문이다.

```text
   curl -H "Authorization: Bearer $T" "$M/v1/runs?limit=5" \
     | jq '.runs[] | has("submitter")'
     -> 모든 행이 true.  실 함대 Run 의 값은 ""
```

**거절된 Run 도 이 표를 지킨다.** `submit()` 의 `store.Run` 리터럴이 셋이고 그중
둘이 `CreateRejectedRun` 을 타므로 그쪽 INSERT 에도 열을 넣는다
(`domain-entities.md` §7.2). 데모에서 가장 흔한 실패가 「노드가 꺼져 있어 422」이고,
그 행에 게스트 이름이 없으면 CP9 가 재는 화면에서 그 Run 만 이름이 빈다.

### 7.4 CP0 — 회귀 (모든 유닛이 착수·병합 전 재실행)

```text
   eval "$(scripts/testdb.sh)"        먼저 친다.  안 치면 아래가 빨갛다
   go test ./... && go vet ./... && go run ./scripts/glyphscan.go
   test -z "$(gofmt -l .)"
   test -z "$(git status --porcelain)"
   go test ./... -count=1 -coverpkg=./... -coverprofile=/tmp/cover.out
     + ci.yml 커버리지 awk (패키지별 하한 80)
   GOOS=windows GOARCH=amd64 go build -o /tmp/enodectl.exe ./cmd/enodectl
     go tool nm /tmp/enodectl.exe | grep -c ' T net/http\.'    -> 50 이하
     go tool nm /tmp/enodectl.exe | grep -c ' T crypto/tls\.'  -> 10 이하
   curl 로 기존 라우트 하나 — GET /v1/capabilities
   기존 열다섯 라우트가 그대로 등록돼 있다 (api.go:62~76)
```

**심볼 상한은 둘이다.** `ci.yml` 의 판정이 `tls > 10 || http > 50` 이고
`scene-gates.md` 의 CP0 도 「심볼 상한 통과」로 둘을 함께 센다. 게이트 블록에는
`net/http` 만 적혀 있으나 차단 조건은 둘이고, 이 목록을 뒤 유닛이 베낀다.

**`git status --porcelain` 을 커버리지 줄보다 먼저 친다 — 순서가 규칙이다.**
`cmd/enodectl/probe.lock` 은 체크인된 추적 파일인데 `-coverpkg -coverprofile` 을
붙인 실행이 그것을 바꾼다(재현 2/2 · 플래그 없이는 안 바뀐다). 위 순서면 통과하고,
커버리지 뒤에 다시 재면 빨갛다. **행렬 밖 파일이므로 실수로 커밋되면 유닛과 무관한
diff 가 남는다** — 커버리지 스텝 뒤에는 `git checkout -- cmd/enodectl/probe.lock`
으로 되돌린다. 기존 결함이고 `constraints.md` 가 이미 적어 두었다.

린트는 이 저장소에서 차단이 아니다. 지적 수가 20 에서 늘었는지만 본다.

**심볼 상한은 이 유닛이 못 움직인다.** `go list -deps ./cmd/enodectl` 에
`internal/store` · `internal/api` · `internal/runctl` 이 **없다.** 이 유닛이 그
그래프 안에서 만지는 것은 `internal/contract/advert.go` 의 구조체 하나뿐이고
임포트도 함수도 안 는다. 지금 실측값은 `net/http` 6 · `crypto/tls` 1 이다.

### 7.5 이 유닛이 만지는 파일

```text
   internal/store/store.go       읽기 둘(Nodes · Runs) + UpsertAdvert(CTE·검사) +
                                 Run.Submitter + CreateRun 의 열 하나.  전부 additive
   internal/store/observe.go     StepView 에 chosen 한 필드 + Steps SELECT 한 열
   internal/store/schema.sql     ALTER 둘 (nodes.draining · runs.submitter)
   internal/api/api.go           등록 두 줄 + 데모 모드의 조건부 등록(읽기 셋 셋) +
                                 getRun 의 requires·warnings + advertResponse 의
                                 drain + submitter 컨텍스트 키 + submit() 의
                                 store.Run 리터럴 세 자리.  핸들러 본문은
                                 새 파일 둘(nodes.go · runs.go)에 짓는다
   internal/runctl/client.go     Nodes · Runs · RunsQuery + Run.Requires ·
                                 Step.Chosen · Step.StartedAt
   internal/contract/advert.go   Policy{Drain} 필드 하나        행렬 밖 · 단독 obs
   internal/config/config.go     ENODE_DEMO_MODE 하나           행렬 밖 · demo-back 과 공유
```

**데모 조건부 등록이 기존 라우트의 등록 줄을 만진다** — `GET /v1/runs/{id}` 는
새 라우트가 아니다. CP0 의 「기존 열다섯 라우트가 그대로」와 맞닿으므로 **기본값이
거짓**인 것이 그 회귀를 지킨다(`business-logic-model.md` §7.3). 데모 스위치를 켠
채로 CP0 을 돌리면 그 줄이 달라진다.

**행렬에 더할 줄이 넷이다** — `internal/store/observe.go` · `internal/store/schema.sql`
(둘 다 store 접점의 일부인데 행렬이 `store.go` 만 세었다) · `internal/contract/advert.go` ·
`internal/config/config.go`. 앞의 둘 중 `schema.sql` 은 queue 가 인덱스를 붙일
개연이 있어 **같은 꼬리에 붙는다** — 진행자가 직렬 대상으로 본다.

**`internal/config/config.go` 도 단독이 아니라 접점이다.** obs 가 `ENODE_DEMO_MODE`
를 만들고 demo-back(runixs · W2)이 같은 값을 읽어 데모 쓰기 라우트를 등록한다.
`internal/contract/advert.go`(단독 obs)와 다른 처리이므로 진행자가 직렬 대상에 넣는다.

`internal/runctl/client.go` 는 행렬이 「단독 obs」로 세웠으나 **둘이 더 만진다.**

```text
   mcp (W1 · 같은 담당)   asks.list 의 글자 일치(CP7)에 원문 반환이 하나 더 필요하다.
                        기존 AskItem 이 서버의 AskView 와 어긋나(asked_at 이 없고
                        omitempty 넷이 빠졌다) 재마셜에서 값이 사라진다.
                        obs 가 안 쓸 메서드를 미리 짓지 않고 mcp 가 자기 필요만큼 넓힌다

   panel (W3 · 다른 담당)  Step.StartedAt 이 없으면 S3 의 「시작 시각」이 안 선다.
                        담당이 갈리므로 미루지 않고 W0 이 세운다 —
                        domain-entities §5 · §8
```

**어느 쪽이든 이 파일은 「단독 obs」가 아니다.** 행렬 옆에 그 사실을 적는다.

**`api.go 는 등록 줄만`(constraints)을 이렇게 읽는다** — 그 제약은 **새 핸들러**를
그 파일에 짓지 말라는 것이고, 새 핸들러 둘은 새 파일에 짓는다. `getRun` ·
`advertResponse` 는 기존 핸들러의 확장이라 그 파일에 남는다.

**W0 이므로 이 유닛이 접점에 처음 선다** — 뒤에 오는 queue · demo-back 이 이 diff
위에 얹는다.

### 7.6 커버리지 — `internal/api` 가 걸리는 자리다

새 패키지가 없다. 하한 80% 가 걸리는 자리는 기존 셋이고, 기준선(`.coverage-contract.yml`
스냅숏)에서 **여유가 크게 다르다.**

```text
   패키지            실측 (2026-09-08)   하한까지의 여유   스냅숏 파일의 값
   ───────────────   ────────────────   ──────────────   ────────────────
   internal/api      328/403 = 81.4%     5.6 문장         327/402
   internal/store    1232/1501 = 82.1%   31.2 문장        같다
   internal/runctl   78/81 = 96.3%       13.2 문장        같다
   internal/api/ui   12/13 = 92.3%       1.6 문장         파일에 없다
```

새 문장 `n` 개를 더할 때 안 덮여도 되는 것은 `0.2n + 여유` 다.
**`internal/api` 가 이 유닛의 병목이다** — 여유가 5.6 문장뿐이라 새 핸들러 둘과
질의 파싱과 `requires` 사상의 대부분이 덮여야 한다. 계획 단계에서 이 수를 안 적어
두면 구현자가 마지막에 발견한다. **정책 어휘 검사를 store 에 두는 이유가 이 수다**
(§2.5 · `domain-entities.md` §7.3) — 여유 31.2 쪽에 얹는다.

**수는 스냅숏이 아니라 오늘 실측이다.** `.coverage-contract.yml` 은 게이트 입력이
아니라 기준선이고 `internal/api` 에서 한 문장 어긋나 있다. `internal/api/ui` 는
여유 1.6 문장으로 저장소에서 가장 빡빡한데 그 파일에 아예 없다 — obs 는 안 만지지만
누구든 그 패키지를 만지면 거기서 먼저 걸린다.

테스트는 진짜 Postgres 를 쓴다 — 기존 `internal/api/api_test.go` 의 `newServer`
와 같은 방식이고, CI 가 postgres 서비스를 붙여 `t.Skip` 이 CI 에서 발동하지
않는다(`.ci-allowed-skips` 는 비어 있고 그것이 결론이다). 스크래치 DB 헬퍼를
새로 만들지 않는다.

**`internal/runctl` 은 DB 없이 덮는다.** `client_test.go` 에 이미
`newClient(http.HandlerFunc)` 가 `httptest` 로 있고, `TestHeaders` ·
`TestFailCarriesWireCode` 가 필요한 모양 그대로다 — 경로와 질의 문자열을 단언하고
본문을 그대로 비교하면 `Nodes` · `Runs` 와 `Fail` 갈래까지 스무 줄 남짓이다.

`.coverage-contract.yml` 은 게이트 입력이 아니라 기준선 스냅숏이고 **지금 낡았다** —
`decisions.md` 2절이 적은 것(측정 커밋이 이 저장소에 없다 · 5,979에서 6,200 문장)에
더해, 패키지가 열다섯으로 적혀 있는데 `go list ./...` 는 **열여섯**을 낸다
(`internal/api/ui` 가 cardnews 병합으로 들어왔고 이 파일에 없다). **이 유닛은 그
파일을 안 고치되, 유닛을 닫을 때 이 낡은 정도를 한 줄로 적는다.**
