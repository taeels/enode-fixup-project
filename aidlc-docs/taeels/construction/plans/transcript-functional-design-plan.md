# U1 `transcript` — Functional Design 계획

**유닛** `transcript` · **브랜치** `unit/transcript` · **담당** taeels ·
**회차** v3-run-transcript · **닫는 게이트** 없다 (코드 게이트만) · **선행** 없다

정본 입력은 `aidlc-docs/v3-run-transcript/inception/application-design/` 의 넷과
`requirements/transcript/` 팩이다. 이 계획은 그 위에서 **사건의 형식과 읽는
규칙**만 짓는다 — 코드는 다음 단계다.

---

## 0. 이 단계가 닫는 것과 안 닫는 것

```text
   닫는다    Event · Kind · Result · Elided 의 필드와 뜻
             줄 하나와 사건 하나의 관계 (하나인가 여럿인가)
             실측한 와이어 사건 다섯이 Kind 에 어떻게 떨어지나
             enode 가 찍는 표시 줄 둘(enode.elided · enode.capped)을 읽는 규칙.
               **찍는 조건은 안 적는다** — elided 는 selectLogs 가 · capped 는 U3 이 진다
             껍데기 줄과 원문 줄을 가르는 규칙
             표시 상한(200 · 500)을 파서가 지나 화면이 지나
             잘린 입력의 규칙 — 머리(링이 감겼다)와 꼬리(미완 줄)
             옮겨 오는 함수의 시그니처와 이름, 그리고 enode 에 남는 쪽의 이음매
             경계 검사에 더할 금지 줄과 그 검사가 못 보는 자리

   안 닫는다  카드가 그리는 모양과 접기          U5 · U6 · U8 의 FD
             as=events 응답의 JSON 표현         U4 의 FD
             Decode 가 줄 단위로 배출하는 흐름   U2 의 FD
             진행 파일의 형식과 시도 곁파일      U3 의 FD
             검사기 파일의 자리                 파일 행렬 6.3 이 Code Generation 에 넘겼다
             시험 목록과 파일별 diff            이 유닛의 Code Generation
             NFR 요구                          이 유닛의 NFR Requirements 단계
```

---

## 1. 실측 — 코드가 지금 어떻게 생겼나 (2026-09-15)

FD 를 짓기 전에 이 유닛이 만지는 자리를 실제로 읽고 돌렸다. **설계 문서가
「셋을 옮긴다」고 적은 자리에 실제로는 아홉이 있었고**, **시그니처 하나가 코드와
갈렸으며**, **요구와 결정이 testdata 한 자리에서 서로를 부정한다.**

### 1.1 옮기는 것이 셋이 아니라 일곱이다 — 타입 둘까지 아홉

`components.md` 1절과 `unit-of-work.md` U1 절이 옮길 것을 셋으로 적었다
(`parseEventLine` · `eventShell` · `elidedMarker`). 그 셋은 저희끼리 안 선다.

| `internal/enode/runner.go` | 무엇인가 | 셋이 부르나 |
|---|---|---|
| `315` `selectLogs` | 짓는 쪽의 오케스트레이션 | **enode 에 남는다** |
| `370` `splitLines` | 줄 나누기 | `selectLogs` 가 쓴다. **`Parse` 도 같은 일이 필요하다** |
| `386` `parseEventLine` | 줄 하나를 아는 키만 읽는다 | 옮긴다 |
| `407` `logShell` (타입) | 껍데기 줄의 형식 | `eventShell` 이 마샬한다 |
| `419` `elidedMark` (타입) | 표시 줄의 형식 | `elidedMarker` 가 마샬한다. **`Result.Elided` 의 와이어 꼴이 이것이다** |
| `425` `elidedMarker` | 표시 줄을 짓는다 | 옮긴다 |
| `434` `eventShell` | 껍데기를 짓는다 | 옮긴다 |
| `487` `usageTokens` | `usage` 에서 정수 넷만 집는다 | `eventShell` 만 부른다 |
| `513` `eventString` | 아는 키를 문자열로 | **`selectLogs` 도 직접 부른다** (`runner.go:330`) |
| `521` `eventBool` | 아는 키를 불리언으로 | `eventShell` 만 부른다 |
| `529` `eventInt` | 아는 키를 정수로 | `eventShell` · `usageTokens` 가 부른다 |

**`elidedMark` 가 옮겨 가는 것이 이 유닛의 값 하나다.** 짓는 쪽
(`ElidedMarker`)과 읽는 쪽(`Result.Elided`)이 같은 구조체를 보게 되어,
`{"type":"enode.elided","events":N,"bytes":B}` 의 키 이름이 한 자리에서만 정해진다.
오늘은 짓는 쪽만 있고 읽는 쪽이 없어서 갈릴 수 없었을 뿐이다.

### 1.2 시그니처가 설계 문서와 갈린다 — `map[string]any` 대 `json.RawMessage`

`component-methods.md` 1.2 가 겉면을 이렇게 적었다.

```go
func ParseLine(line []byte) (obj map[string]any, typ string, ok bool)
func Shell(obj map[string]any, typ string) []byte
```

**오늘 코드는 `map[string]json.RawMessage` 다** (`runner.go:386` · `:434`).
그리고 그 선택에 근거 주석이 붙어 있다 — 「구조체로 한 번에 안 받는다. 모르는
필드의 모양 하나가 줄 전체를 떨어뜨리기 때문이다」. `map[string]any` 로 바꾸면
세 가지가 함께 바뀐다.

```text
   ①  모르는 값까지 전부 푼다.  도구 결과 본문(수십 KB 문자열)이 파싱 시점에
      메모리로 올라온다.  RawMessage 는 아는 키를 읽을 때만 푼다
   ②  eventString · eventBool · eventInt 의 규율이 사라진다.  셋은
      「모양이 다르면 없는 것으로 본다」인데 any 는 타입 단언으로 바뀐다
   ③  SECURITY-13 의 「아는 키만 읽는다」가 「전부 읽고 아는 것만 쓴다」가 된다.
      결과는 같아도 규율의 이름이 바뀐다
```

**설계 문서의 잘못이 아니다** — `component-methods.md` 는 겉면을 적었고 「필드의
값은 Functional Design 이 닫는다」고 스스로 적었다. 이 단계가 그 자리다.
질문 7 이 그것을 정한다.

### 1.3 `selectLogs` 는 옮긴 뒤에도 `eventString` 을 쓴다

```go
// runner.go:326-336 — 옮겨도 남는 자리
obj, typ, ok := parseEventLine(ln)
if initAt < 0 && typ == "system" && eventString(obj, "subtype") == "init" {
```

셋만 옮기면 `selectLogs` 가 남의 패키지의 맵을 들고 자기 패키지에는 없는
`eventString` 을 부르게 된다. 길은 셋뿐이고 **하나는 규약 위반이다**.

```text
   내보낸다        transcript 가 String · Bool · Int 를 공개한다.  이음매가 넓어진다
   되돌린다        ParseLine 이 subtype 을 넷째 값으로 돌려준다.  이음매는 좁고
                  subtype 이 특별 대우를 받는다
   두 벌로 둔다    enode 가 세 줄짜리 자기 eventString 을 유지한다.
                  **CONVENTIONS 1.4 의 「도구는 두 벌로 두지 않는다」를 어긴다**
```

`splitLines` 도 같은 모양이다 — `selectLogs` 와 `Parse` 가 둘 다 줄을 나눈다.
질문 8 이 이 이음매를 정한다.

### 1.4 사건 종류 — 실측 다섯과 팩의 여섯이 다른 축이다

실측은 짝 팩의 U1 이 이미 했고 `requirements/harness-components/decisions.md`
6절 ⑳ 에 행으로 있다 (2026-09-12 · `claude 2.1.266`). 그 표와 이 팩의 Kind 를
나란히 놓으면 **사상되지 않는 것이 넷 남는다.**

| 와이어에서 본 것 | 팩의 Kind (`decisions.md` 2절) |
|---|---|
| `system` / `init` | `init` |
| `system` / `hook_started` · `hook_response` · `thinking_tokens` | **없다** |
| `assistant` (`text` · `thinking` · `tool_use` 블록) | `text` · `tool_use` |
| `user` (`tool_result` 블록) | `tool_result` |
| `rate_limit_event` | **없다** |
| `result` | `result` |
| `enode.elided` (우리가 짓는 줄) | **Event 가 아니다 — `Result.Elided` 다** |

`raw` 는 팩이 「파싱 실패 줄의 원문」으로 정의했다. 위의 빈칸 넷은 **파싱에
실패하지 않는다** — JSON 이고 `type` 이 문자열이다. `raw` 로 떨어뜨리면 그
단어가 두 뜻을 지고, 안 떨어뜨리면 파서가 사건을 버린다 (`constraints.md` §4 가
금지한 것이다). 질문 2 가 그 자리다.

### 1.5 파서가 받는 입력이 둘이 아니라 넷이다

`components.md` 1절이 입력을 둘로 적었다 (원문 스트림 · 선별본). **코드를 따라가면
넷이다.**

| 입력 | 어디서 | 모양 |
|---|---|---|
| 하네스 단계의 링 | `claim.go:794` 가 넘기는 `Job.Transcript` (U2 가 되살린다) | NDJSON 원문. 머리가 잘릴 수 있다 |
| 하네스 단계의 `logs/` · 진행 파일 | `runner.go:261` 의 `selectLogs` 결과 | **선별본** — 전문 둘 · 껍데기 · 표시 줄 · 그 뒤에 **stderr 평문** |
| **명령 단계의 링** | `claim.go:633` 의 `io.MultiWriter(&buf, w.ring)` | **평문이다.** stdout 과 stderr 가 한 줄기로 섞인다 |
| **명령 단계의 로그** | `claim.go:647` 이 `buf` 를 원문 그대로 올린다 | 같은 평문 |

**명령 단계가 오늘 이미 링에 흐르고 있다** — 이 회차가 만드는 것이 아니다.
`decisions.md` 2절이 「명령 단계의 stdout · stderr 는 `text`」로 적었는데,
파서는 바이트만 받으므로 **명령 단계인지 하네스 단계인지 모른다.** 같은 규칙이
선별본 꼬리의 stderr 에도 걸린다 — 그 줄들도 JSON 이 아니다. 질문 3 이 그 자리다.

### 1.6 경계 검사 실측 — `go list -deps` 는 시험 임포트를 안 본다

파일 행렬 6절이 검사기를 열어 본 것까지가 Units Generation 의 몫이었다. **그
검사기가 무엇을 못 보는지를 여기서 쟀다.**

```text
   must not import 를 담은 시험 파일    internal/panel/boundary_test.go 하나
                                       (grep -rln --include=*_test.go 가 하나를 낸다)

   go list -deps ./internal/api/ui      내부 패키지 0.  api/ui -> store 넷째 줄은
                                       오늘 세워도 공짜다

   go list -deps 에 새 패키지가 어떻게 잡히나
     내부 의존이 자기 하나뿐인 패키지가 이미 일곱이다 — api/ui · build · config ·
     proc · record · runctl · schema.  transcript 가 표준 라이브러리만 쓰면
     여덟째가 되고 금지 넷이 **자동으로 초록**이다

   go list -deps 와 go list -test -deps 의 차이 (internal/enode 로 실측)
     -test 를 붙이면 .test 패키지 둘이 더 나온다.  붙이지 않은 목록에는
     **시험 파일의 임포트가 아예 안 들어간다**
```

**그래서 `internal/transcript/*_test.go` 가 `internal/enode` 를 임포트해도
검사기는 초록이다.** 이 유닛에서 그것이 이론이 아닌 이유는 FR-3 의 수용 기준이
「같은 로그를 링에서 읽든 `record` 에서 읽든 같은 사건 열이 나온다」이고, 그
`record` 쪽 바이트를 만드는 `selectLogs` 가 `enode` 에 남기 때문이다. 질문 10 이
그 자리다.

### 1.7 커버리지 실측 — 옮기면 `internal/enode` 가 85.2 에서 84.8 로 간다

`go test -coverpkg=./... -count=1 ./internal/enode/` 로 쟀다 (저장소 표준 명령은
모든 패키지의 시험을 함께 돌리므로 실제 값은 이보다 높거나 같다).

```text
   internal/enode        문장 2,314 · 덮임 1,971 · 85.2%
   옮길 후보 (386-540)    문장 65 · 덮임 63 · 96.9%
   옮긴 뒤의 enode        1,908 / 2,249 = 84.8%   하한 80% 와 4.8%p
   옮겨 간 transcript     63 / 65 = 96.9% 에서 시작한다.  Parse 가 여기에 더해진다
```

**옮기는 것이 하한을 위협하지 않는다.** 다만 `internal/transcript` 는 새
패키지라 80% 하한이 그대로 걸리고, 시험이 따라오지 않으면 그 자리에서 빨갛다
(`.coverage-contract.yml` 의 `default_pct: 80`).

함수별로는 `elidedMarker` 가 75.0% 다 — `json.Marshal` 의 오류 가지가 안 밟힌다.
옮긴 뒤 그 가지를 어떻게 다룰지는 Code Generation 이 정한다.

### 1.8 `testdata` 가 저장소에 0 이다 — 요구와 결정이 갈린다

```text
   requirements.md 5.1 · constraints.md   「실측한 stream-json 줄을 testdata 에 둔다.
                                          그것은 실제로 받았던 것의 기록이므로 고치지
                                          않는다 (CONVENTIONS.md 2.2)」

   harness-components/decisions.md ⑳       「원문은 저장소에 안 싣는다 — 개인 홈의
                                          서버 이름과 절대경로가 들어 있다」
```

**둘이 같은 파일을 두고 반대를 말한다.** 오늘의 값은 ⑳ 쪽이다 — 저장소 전체에
`testdata` 디렉터리가 둘뿐이고(`internal/contract` · `internal/api/ui`) 둘 다
stream-json 이 아니다. `internal/enode/logs_test.go` 는 실측한 **모양**을 손으로
지은 Go 상수 다섯으로 들고 있다 (`initLine` · `assistantToolLine` · …).

그 상수들은 `package enode` 의 것이다. 껍데기 시험이 `transcript` 로 옮겨 가면
**같은 픽스처를 두 패키지가 본다** — 1.3 과 같은 종류의 자리다. 질문 11.

### 1.9 안 만져도 되는 것을 확인했다

```text
   internal/enode/claude.go 의 Decode      U2 의 것이다.  오늘 io.ReadAll 로 배치다
                                          (claude.go:412).  U1 은 안 만진다
   ParseClaude · lastJSONObject           harness.go:118 · :172 에 있다.
                                          runner.go 가 아니라 harness.go 라
                                          U1 의 파일 행렬 밖이다.  ⑯ 의 type 검사도 거기다
   Argv 의 stream-json 전환                짝 팩이 이미 main 에 넣었다 (claude.go:343)
   Ring · ReadRing                        형식이 안 바뀐다 (constraints.md 5).
                                          Snapshot 의 Total > capacity 가 곧 truncated 다
   internal/api/ui                        내부 의존 0.  U1 이 금지 줄을 세워도 코드 diff 0
```

---

## 2. 물음 열둘

답을 `[Answer]:` 뒤에 적는다. 권장이 있는 물음은 권장을 **A** 에 둔다.

### Question 1

**줄 하나가 사건 하나인가 여럿인가.** 실측한 `assistant` 줄은
`message.content[]` 에 블록을 여럿 담는다 — `thinking` 과 `text` 와 `tool_use`
둘이 한 줄에 같이 올 수 있다. 껍데기도 `tools` 를 **배열**로 들고 있다
(`runner.go:447-450` 의 주석이 그 이유를 적었다).

A) **블록마다 사건 하나다.** `assistant` 한 줄이 `text` 하나와 `tool_use` 둘로
펴진다. 화면이 도구 이름을 하나씩 그리고 `tool_result` 를 그 호출에 붙일 자리가
생긴다. 대가는 `Result.Events` 의 길이가 줄 수와 다른 것이고, 껍데기에서는 블록
종류를 모르므로 `tools` 배열의 길이만큼만 펴진다

B) **줄마다 사건 하나다.** `Event` 가 블록 목록을 필드로 든다. 줄과 사건이
일대일이라 원문 토글과 자리가 맞고 `Raw` 를 세는 규칙이 단순하다. 대가는 화면이
사건 안을 다시 훑어야 하는 것이고, 「파서가 들고 화면은 그린다」가 한 겹 얇아진다

C) **본문 블록은 펴고 도구는 묶는다** — `text` 와 `thinking` 은 사건 하나씩,
`tool_use` 는 그 줄의 배열 하나. 실측이 「사건마다 블록 하나였다」이므로 오늘은
차이가 안 나고, 뒤에 여럿이 오면 도구만 뭉친다. 규칙이 블록 종류마다 달라진다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2

**Kind 여섯 밖의 사건을 무엇으로 내나.** 1.4 가 넷을 이름으로 적었다 —
`system/hook_started` · `system/hook_response` · `system/thinking_tokens` ·
`rate_limit_event`. 넷 다 JSON 이고 `type` 이 문자열이라 **파싱에 실패하지
않는다.** 함께 묻는다 — `Result.Raw` 가 `Events` 안의 raw 를 다시 센 값인가.

A) **`raw` 로 낸다. 그리고 `raw` 의 정의를 「못 읽은 줄」에서 「Kind 로 사상되지
않은 줄」로 넓힌다.** `Raw` 는 그 사건의 **개수**이고 `Events` 안에도 같은 것이
있다 (세는 값이지 따로 담는 통이 아니다). Kind 가 안 늘어 화면 셋이 안 바뀌고,
넓힌 정의를 `business-rules.md` 가 이름으로 진다. 대가는 「파서가 틀렸다」와
「파서가 모른다」가 화면에서 같아 보이는 것이다

B) **Kind 를 일곱으로 늘린다 — `meta` 하나를 더한다.** 훅 · 사고 토큰 · 요율
제한이 거기로 가고 `raw` 는 「못 읽은 줄」로 좁게 남는다. 화면이 `meta` 를 접어
두면 사람이 읽는 줄기가 안 흐려진다. 대가는 팩의 `decisions.md` 2절이 못 박은
여섯을 이 유닛이 늘리는 것이고, 그 근거를 6절에 행으로 적어야 한다

C) **와이어 `type` 을 그대로 Kind 로 쓴다 — 닫힌 어휘를 안 만든다.** 하네스가
종류를 늘려도 파서가 안 바뀐다. 대가는 화면 셋이 모르는 Kind 를 받게 되는 것이고
「표시 규칙을 파서가 든다」가 깨진다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3

**JSON 이 아닌 줄을 `text` 로 내나 `raw` 로 내나.** 1.5 가 그런 줄이 오늘 두
경로에 있음을 보였다 — 명령 단계의 링과 로그(평문 전부), 그리고 선별본 꼬리의
stderr. `decisions.md` 2절은 한 줄에서 「명령 단계의 stdout · stderr 는 `text`」
라 하고 다른 줄에서 「`raw` 는 파싱 실패 줄의 원문」이라 한다.

A) **JSON 이 아닌 줄은 `text` 다. 다만 `text` 에 출처 표시를 단다.**
파서는 단계 종류를 모르지만 「이 줄은 JSON 이 아니었다」는 안다. 명령 단계가
통째로 읽을 수 있는 문장으로 흐르고 (CB1 의 「읽을 수 있는 문장」이 명령 단계에도
선다), 선별본 꼬리의 stderr 도 같은 모양으로 붙는다. 대가는 `raw` 가 「JSON 인데
Kind 를 모르는 줄」만 지게 되는 것이다 (질문 2 의 답과 맞물린다)

B) **JSON 이 아니면 전부 `raw` 다.** 팩의 `raw` 정의를 글자 그대로 지킨다.
명령 단계의 화면은 `raw` 사건이 줄줄이 흐르고 화면이 그것을 접는다 — 읽을 수
있는 문장이 접힌 채로 온다. 대가는 CB1 의 값이 명령 단계에서 안 서는 것이다

C) **호출자가 힌트를 넘긴다** — `Parse(b, truncated, plain bool)`. 부르는 쪽은
단계 종류를 안다 (제어판은 상태를, API 는 `steps` 를 본다). 정확하지만 겉면이
늘고, 힌트가 틀린 경로가 하나 생기면 화면이 갈린다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4

**껍데기 줄과 원문 줄을 무엇으로 가르나.** 같은 `type` 아래 둘이 온다 —
원문은 `{"type":"assistant","message":{...}}` 이고 껍데기는
`{"type":"assistant","tools":["Read"],"tokens":{...}}` 다. 껍데기에는 본문이
아예 없으므로 화면이 「걷혔다」로 그려야 한다 (US-6).

A) **줄마다 본다 — `message` 키가 있으면 원문, 없으면 껍데기.**
`eventShell` 이 짓는 객체에 `message` 가 절대 없고 (`logShell` 의 필드 다섯이
전부다) 실측한 원문 `assistant` · `user` 에는 언제나 있다. 호출자가 아무것도 안
넘겨도 되고, 한 파일에 둘이 섞여도 (진행 파일이 봉인 뒤 선별본으로 바뀌는 자리)
줄마다 옳게 갈린다. 대가는 `message` 가 없는 원문 사건(`system/init` ·
`result` · `rate_limit_event`)을 껍데기로 오인할 수 있다는 것이고, 그 셋은
**전문으로 남는 줄이라 본문이 있어 구별된다** — 규칙을 그렇게 적는다

B) **표시 줄로 본다 — `enode.elided` 줄이 그 입력에 있으면 그 파일은 선별본이다.**
`selectLogs` 는 stdout 이 비지 않는 한 언제나 그 줄을 쓴다 (`runner.go:360`).
파일 단위의 판정이라 흔들림이 없다. 대가는 `from` 오프셋으로 잘라 읽으면 그 줄이
안 보이는 것이고 (그 줄은 stderr 앞, 즉 거의 끝이다) 링에는 아예 없다

C) **호출자가 넘긴다** — `Parse(b, truncated, selected bool)`. 부르는 쪽 셋이
전부 자기가 무엇을 읽는지 안다. 겉면이 늘고, 틀리게 넘기는 경로가 생기면
화면이 「본문이 없다」와 「걷혔다」를 바꿔 그린다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5

**표시 상한(도구 입력 200자 · 도구 결과 500자)을 파서가 자르나 화면이 자르나.**
`decisions.md` 2절이 「값은 화면이 아니라 파서가 든다」고 적었는데,
`constraints.md` §4 는 「파서가 사건을 고치거나 버리는 것」을 제외로 적었다.

A) **파서가 자르고, 자른 사실을 필드로 남긴다.** `Event` 가 잘린 문자열과
`Truncated bool`(또는 원래 길이)을 든다. 화면 셋이 같은 곳에서 잘리고,
`as=events` 응답이 본문 크기에 안 휘둘린다 — 도구 결과 하나가 수십 KB 여도
사건 배열이 작다. 「고치지 않는다」는 원문 토글이 지킨다 (원문은 `as=raw` 와
제어판의 `data` 로 언제나 그대로 나간다)

B) **파서는 전문을 들고 화면이 자른다.** 팩의 「고치지 않는다」가 글자 그대로
선다. 대가는 `as=events` 응답이 원문만큼 커지는 것이고 (중앙 폴링이 2초마다
그것을 받는다) 화면 둘이 같은 상한을 각자 들어 갈릴 수 있다

C) **파서가 전문과 잘린 것을 둘 다 든다.** 화면이 고르고 응답은 안 준다 —
`as=events` 는 잘린 쪽만 싣는다. 가장 안 잃지만 사건 하나가 본문을 두 벌로
들어 메모리가 두 배다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6

**꼬리의 미완 줄을 어떻게 하나.** 링과 진행 파일은 **쓰는 중에** 읽힌다. 마지막
줄이 개행 없이 끊겨 있을 수 있고, 1초 뒤 다시 읽으면 그 줄이 완성되어 있다.
오늘 `splitLines` 는 마지막 빈 조각만 버리고 미완 줄은 그대로 넘긴다
(`runner.go:370`).

A) **개행으로 안 끝나는 마지막 줄은 버린다 — 다만 그 사실을 `Result` 에
적는다.** 폴링이 같은 줄을 먼저 `raw` 로 그렸다가 다음 회차에 제대로 된 사건으로
바꾸는 깜빡임이 없어진다. 대가는 개행 없이 끝난 파일(크래시)의 마지막 줄이
사건으로 안 나오는 것이고, `Result` 의 표시와 원문 토글이 그것을 덮는다

B) **미완 줄도 사건으로 낸다** (파싱되면 사건, 안 되면 질문 2 · 3 의 규칙대로).
아무것도 안 잃는다. 대가는 폴링 화면의 깜빡임이고, 그 깜빡임이 US-1 이 막으려던
「멈춘 것인지 도는 것인지」의 오독과 같은 자리에서 난다

C) **호출자가 정한다** — 스트림이면 버리고 봉인이면 남긴다. 정확하지만 겉면이
또 하나 늘고, 질문 3 · 4 의 힌트와 합치면 인자가 넷이다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 7

**`ParseLine` 과 `Shell` 이 무슨 맵을 주고받나.** 1.2 가 설계 문서
(`map[string]any`)와 코드(`map[string]json.RawMessage`)의 갈림을 보였다.

A) **`map[string]json.RawMessage` 를 그대로 옮긴다.** 오늘 코드가 그것이고
근거 주석이 붙어 있으며, 도구 결과 본문을 파싱 시점에 안 푼다. SECURITY-13 의
「아는 키만 읽는다」가 글자 그대로 남는다. `component-methods.md` 1.2 의
`map[string]any` 를 이 단계의 산출물이 고쳐 적는다 (겉면 문서가 FD 에 넘긴 자리다)

B) **`map[string]any` 로 바꾼다 — 설계 문서를 코드에 맞추지 말고 코드를 문서에
맞춘다.** 표준 라이브러리만 쓰는 패키지의 겉면이 더 평범해지고 화면 쪽에서 쓰기
쉽다. 대가는 1.2 의 ① ~ ③ 셋이다

C) **맵을 겉면에서 없앤다** — `ParseLine` 이 내부 타입을 돌려주고
(`transcript.Line` 같은 불투명 값) `Shell` 이 그것을 받는다. 호출자가 맵을
못 만지므로 규율이 패키지 안에 갇힌다. 대가는 새 타입 하나와, `selectLogs` 가
`subtype` 을 읽을 길을 따로 줘야 하는 것이다 (질문 8 과 맞물린다)

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 8

**`selectLogs` 가 계속 쓰는 것을 어떻게 잇나.** 1.3 이 `eventString` 과
`splitLines` 둘을 이름으로 적었다. 두 벌로 두는 길은 CONVENTIONS 1.4 가 막는다.

A) **`transcript` 가 읽기 도우미를 공개한다** — `String` · `Bool` · `Int` 와
`SplitLines`. `enode` 가 그것을 부른다. 이음매가 넓지만 **`Shell` 을 부르려면
어차피 그 맵을 손에 들어야 하므로** 공개 표면이 실제로 늘지 않는다.
`Parse` 와 `selectLogs` 가 같은 줄 나누기를 써서 「링에서 읽든 record 에서
읽든」의 전제가 코드로 선다

B) **`ParseLine` 이 subtype 을 넷째 값으로 돌려준다.** 이음매가 좁고 도우미
셋이 패키지 안에 남는다. 대가는 `subtype` 만 특별 대우를 받는 것이고, 뒤에
`selectLogs` 가 다른 키를 읽어야 하면 같은 협상을 다시 한다. `splitLines` 는
따로 공개해야 한다

C) **`selectLogs` 까지 `transcript` 로 옮긴다.** 도우미 문제가 통째로 사라지고
짓는 쪽이 한 패키지에 전부 모인다. 대가 둘 — `runner.go` 가 결과 바이트만 받게
되어 「어댑터가 아니라 여기서 한다」는 `runner.go:312-314` 의 근거 주석이 갈 곳을
잃고, `Q4 = A` 가 옮기라고 한 셋의 범위를 이 단계가 혼자 넓힌다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 9

**`transcript.Kind` 와 `enode.EventKind` 가 어떤 사이인가.** `enode` 에 이미
`EventKind`(`harness.go:285`)와 `Event{Kind, Text}`(`:291`)가 있고 값이 `final`
하나다. U2 가 `Decode` 를 줄 단위로 바꾸면서 그 `emit` 으로 사건을 흘린다 —
시그니처는 안 바뀐다 (`application-design.md` 3절).

A) **둘을 안 섞는다. `enode.Event` 는 오늘 그대로 두고 U2 가 필요하면 자기
Kind 를 늘린다.** `transcript` 는 화면이 그릴 모형이고 `enode.Event` 는
데몬 로그의 신호다 (`claim.go:793` 이 `log.Debug` 로 버린다). 이 유닛은 U2 의
자리를 안 정하고 **이음매만 이름으로 적는다**

B) **`enode.Event` 가 `transcript.Event` 를 싣는다** — 필드 하나를 더한다.
사건의 모형이 저장소에 하나가 된다. 대가는 `enode -> transcript` 의존이
`Decode` 겉면까지 올라오는 것이고, 그 겉면은 `agent-runtime` R3 의 정본이다

C) **`enode.EventKind` 를 없애고 `transcript.Kind` 로 통일한다.** 「파서는
하나다」가 가장 강하게 선다. 대가는 U2 와 정본 문서를 이 유닛이 건드리는 것이고,
파일 행렬 밖이다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 10

**경계 검사에 무엇을 세우나.** 요구는 `transcript` 의 금지 넷이다
(`requirements.md` 5.2). 1.6 이 그 검사가 못 보는 자리와, 옆에 비어 있는 자리를
쟀다. **검사기 파일의 자리는 Code Generation 이 정한다** (파일 행렬 6.3) —
여기서 정하는 것은 **무엇을 재는가**다.

A) **금지 넷 + `api/ui -> store` 넷째 줄까지 다섯. 시험 임포트는 안 잰다.**
넷째 줄은 `requirements.md` 5.2 가 규칙으로 적어 두고 검사기에 없던 자리이고
오늘 세워도 코드 diff 0 이다 (1.6). 시험 임포트를 안 재는 이유를
`business-rules.md` 가 이름으로 진다 — 시험은 제품 바이너리에 안 실리고,
`go list -test -deps` 로 재면 `.test` 패키지가 목록을 흐린다

B) **금지 넷만. 넷째 줄은 안 건드린다.** `api/ui` 는 이 유닛의 파일 행렬 밖이고
그 빈자리는 이 회차가 만든 결함이 아니다. 대가는 그 자리가 계속 비는 것이다

C) **금지 넷 + 넷째 줄 + 시험 임포트까지 잰다** (`go list -test -deps` 로
`transcript` 만). 「파서는 아래를 모른다」가 시험에서도 선다. 대가는 FR-3 의
수용 기준을 재는 시험이 `enode` 쪽에만 살 수 있게 되는 것이고 (`selectLogs` 가
거기 남는다), 검사 명령이 자리마다 달라진다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 11

**픽스처를 어디에 두나.** 1.8 이 요구(`testdata` 에 둔다)와 결정(원문은 안
싣는다)의 갈림을 보였다. 오늘의 픽스처는 `logs_test.go` 의 Go 상수 다섯이고
`package enode` 의 것이다. 껍데기 시험이 옮겨 가면 두 패키지가 같은 줄을 본다.

A) **`internal/transcript/testdata/` 에 파일로 둔다. 원문이 아니라 실측한
모양을 손으로 지은 것임을 그 디렉터리의 README 가 적는다.** `enode` 의 시험은
같은 파일을 상대 경로로 읽는다 (`../transcript/testdata` 가 아니라, Code
Generation 이 자리를 정한다). `requirements.md` 5.1 의 글자가 서고 두 벌이 안
생긴다. 대가는 시험 픽스처를 파일로 읽는 선례가 `internal/enode` 에 없다는 것이다

B) **`internal/transcript` 의 공개 시험 도우미로 둔다** — 픽스처 상수를 내보내
`enode` 의 시험이 임포트한다. 파일이 안 늘고 컴파일러가 오타를 잡는다. 대가는
시험용 심볼이 제품 패키지의 공개 표면에 서는 것이다

C) **두 벌을 허용하고 그 사실을 적는다.** 각 패키지가 자기 상수를 든다.
가장 단순하고, 갈리면 두 시험 중 하나가 빨개져 곧 드러난다. 대가는
CONVENTIONS 1.4 와 같은 결의 중복이다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 12

**`GLOSSARY.md` 가 이 저장소에 아직 커밋되지 않았다.** `git ls-files GLOSSARY.md`
가 0 을 낸다 — 파일은 진행자의 워킹 트리에만 있고 worktree 에는 없다. 그런데
이 회차의 게이트 이름 `CB0` ~ `CB6` 은 이미 커밋된 문서 스무 곳이 쓰고 있고
(`git grep -l "CB0" HEAD`) 글자를 푼 줄이 저장소에 0 이다 — `CLAUDE.md` 가
`CP` · `CA` 에 대해 지목한 것과 같은 자리다.

**이 유닛은 새 글자를 안 들여온다.** `transcript` 는 패키지 이름이고 축약이
아니며, `U1` ~ `U8` · `CB` · `NC` 는 전부 앞선 커밋이 들여왔다. 그래서 묻는
것은 **빚을 누가 지금 지는가**다.

A) **이 유닛 밖에서 진행자가 한 커밋으로 올린다.** `GLOSSARY.md` 를 `main` 에
넣고 같은 커밋이 `CB` 행을 더한다 — U2 가 같은 자리를 그렇게 처리했다
(`contract-vocab` 질문 7 의 답 B). `CONVENTIONS.md` 3.4 의 「한 커밋은 그 단계의
산출물」이 그 근거다. **`CB` 의 푼 말은 짐작하지 않는다** — `CP` · `CA` 와 같이
「없다」로 적거나 사람이 값을 준다

B) **이 유닛의 커밋이 `GLOSSARY.md` 를 만들고 `CB` 행을 더한다.** 회차의
Construction 첫 유닛이 그 자리를 지는 것으로 본다. 대가는 U1 의 diff 에 이
유닛과 무관한 파일이 실리는 것이다

C) **미룬다.** 다음에 새 글자를 들여오는 커밋이 함께 한다. 대가는
`CLAUDE.md` 가 지목한 빚이 회차 하나를 더 넘어가는 것이다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## 3. 질문 범주 여덟 — 묻는 것과 안 묻는 것

규칙(`construction/functional-design.md` Step 3)이 평가하라고 한 여덟을 전부
쟀다. **안 묻는 범주는 왜 안 묻는지를 적는다** — 안 적으면 다음 사람이 빠뜨린
것으로 읽는다.

| 범주 | 묻나 | 근거 |
|---|---|---|
| Business Logic Modeling | **묻는다** — Q1 · Q3 · Q6 | 줄과 사건의 관계가 이 패키지의 뿌리다. 코드를 읽어도 안 나온다 — 오늘 읽는 쪽이 없다 |
| Domain Model | **묻는다** — Q2 · Q7 · Q9 | Kind 의 닫힌 어휘와 맵의 타입. 팩과 코드와 설계 문서가 서로 다른 값을 든다 |
| Business Rules | **묻는다** — Q4 · Q5 · Q10 | 껍데기 판정 · 상한 · 금지 목록. 셋 다 규칙이고 셋 다 대가가 갈린다 |
| Data Flow | **안 묻는다** | 코드가 답한다. 입력 넷의 출처와 모양을 1.5 가 실측으로 적었고, 파서는 `[]byte` 하나를 받아 `Result` 하나를 낸다 — 지속화도 외부 호출도 0 이다 |
| Integration Points | **안 묻는다** | 이 패키지는 표준 라이브러리만 임포트한다 (`constraints.md` 구조 불변식). 외부 계면이 0 이라 물을 것이 없다. 패키지 사이의 이음매는 Q8 · Q9 · Q10 이 이미 든다 |
| Error Handling | **안 묻는다** | 팩이 값으로 닫았다 — 「파서는 사건을 버리지 않는다. 못 읽은 줄은 `raw` 다」(`constraints.md` §4). 실패 등급이 아예 없다 (`components.md` 1절). 남은 모호(무엇이 `raw` 인가)는 Q2 · Q3 가 든다 |
| Business Scenarios | **묻는다** — Q6 · Q11 | 잘린 꼬리와 픽스처. 둘 다 경계 사례이고 둘 다 실측이 찾았다 |
| Frontend Components | **해당 없음** | 이 유닛에 UI 가 0 이다. 카드는 U5 · U6 · U8 의 것이고 화면 자리는 이 회차의 새 `.pen` 으로 진행자가 그린다 (`scene-gates.md` 2.1) |

**규칙 밖에서 하나를 더 물었다** — Q12 (`GLOSSARY.md`). `CLAUDE.md` 가 단계마다
거는 규약이고 짐작한 확장을 금지하므로 질문으로 냈다.

---

## 4. 실행 단계 — 답을 받은 뒤

### 4.1 순서

- [x] 답 열둘을 읽고 **모순을 먼저 센다.** 맞물리는 쌍이 셋이다 —
      Q2 와 Q3 (`raw` 의 뜻) · Q5 와 Q1 (사건 하나가 무엇을 드나) ·
      Q7 과 Q8 (맵의 타입과 도우미의 공개 범위). 어긋나면
      `transcript-functional-design-clarification-questions.md` 를 낸다
      -> **모순 0.** 결과는 4.1.1. 해명 질문 파일을 안 냈다
- [x] 산출물 셋을 쓴다 (4.2)
- [x] 6절의 「회차 밖으로 낼 것」을 후보에서 **목록**으로 좁혀 산출물에 적는다
- [x] 완료 메시지를 규칙의 형식으로 낸다. **커밋하지 않는다** — 진행자가 한다

### 4.1.1 모순 검사 — 0 (2026-09-15)

답 열둘이 전부 **A** 다. 맞물리는 쌍 셋과, 검사 중에 더 본 쌍 넷을 함께 쟀다.

| 쌍 | 판정 | 근거 |
|---|---|---|
| **Q2 - Q3** | 모순 0 | Q2 = A 가 `raw` 를 「Kind 로 사상되지 않은 줄」로 넓히고 Q3 = A 가 「JSON 인데 Kind 를 모르는 줄」로 다시 좁힌다. **Q3 의 A 가 그 좁힘을 대가 줄에 미리 적었다** — 「(질문 2 의 답과 맞물린다)」. 사다리 하나로 둘 다 선다 |
| **Q1 - Q5** | 모순 0 | Q1 = A 는 사건을 **나누는** 규칙이고 Q5 = A 는 사건 하나의 `Text` 를 **자르는** 규칙이다. 축이 다르다. 껍데기 줄기에는 자를 본문이 아예 없어 부딪칠 자리도 없다 |
| **Q7 - Q8** | 모순 0 · **서로를 세운다** | Q8 = A 의 근거(「`Shell` 을 부르려면 어차피 그 맵을 손에 들어야 한다」)가 Q7 = A 를 전제한다. Q7 이 C(불투명 타입)였으면 Q8 = A 가 안 섰다 |
| Q4 - Q2 | 모순 0 | Q4 = A 의 대가(「`message` 없는 원문 셋을 껍데기로 오인할 수 있다」)를 **판정을 `assistant` · `user` 에만 거는 것**으로 닫았다. `business-rules.md` R4.1 |
| Q6 - Q4 | 모순 0 | `enode.elided` 줄은 stderr **앞**이라 꼬리가 아니다 (`runner.go:360`). Q6 = A 가 그 줄을 안 먹는다 |
| Q6 - Q2 | 모순 0 | 안 읽은 꼬리는 `Lines` 에도 `Raw` 에도 안 센다. `Result.Partial` 이 그 사실을 든다 |
| Q10 - Q11 | 모순 0 | 픽스처를 파일로 읽는 것은 임포트가 아니다. 경계 검사와 무관하다 |

**모순은 0 이고 파생 규칙 다섯이 남았다.** 답에서 바로 안 나오고 FD 가 이름으로
지어야 하는 것들이다 — 전부 산출물에 규칙으로 박았다.

```text
   ①  비JSON 과 미사상 JSON 의 2단 사다리          business-logic-model 2절 · R3.1 · R3.2
   ②  블록이 0 인 껍데기도 사건 하나를 낸다         R5
   ③  껍데기 판정을 assistant · user 에만 건다     R4.1
   ④  안 읽은 꼬리 바이트를 Result 가 적는다        R7.2
   ⑤  도우미 넷의 공개 이름과 Fields 별칭           domain-entities 2절 · 7절
```

**새 모호는 0 이고 문서 갈림이 하나 늘었다** — `user-stories.md` US-4 의 확인
글자(「`raw` 사건」)가 R3.1 · R3.2 아래서 「`plain text` 또는 `raw`」 둘로
갈린다. **스토리의 값은 선다** (둘 다 줄 원문을 `Text` 에 그대로 든다).
6절이 그 문장을 고칠 자리와 사람을 적는다.

### 4.1.2 되열림 하나 — U3 의 답이 `Kind` 를 하나 늘렸다 (2026-09-15)

**4.1.1 의 「모순 0」은 그대로 참이다.** 이것은 U1 의 답끼리의 모순이 아니다.
**U1 의 답 열둘은 한 글자도 안 바뀌었다** — 전부 `A` 그대로다.

```text
   무엇이 있었나   U3 progress-store 의 모순 검사가 그 유닛의 Q3 = A(상한 앞
                  마지막 개행까지만 쓴다)와 Q4 = A(파일에 표시 줄을 안 박는다)를
                  합치면 **진행 파일 안에 상한 도달의 흔적이 0** 임을 찾았다.
                  GET 이 「닿아서 멈춘 것」과 「그냥 아직 작은 것」을 못 가르고
                  NC-4 와 US-7 이 그 자리에서 죽는다

   사용자가 고른 것  진행 파일이 상한에 닿으면 NDJSON 표시 줄을 한 줄 박는다.
                  {"type":"enode.capped","bytes":<상한>}.  상한은 MaxBlobBytes
                  이고 기본이 10 MiB 다 (config.go:51 의 int64 · :100 의 기본값)

   U1 에 오는 것    그 줄을 **읽는** 규칙.  Kind 가 여섯에서 일곱이 된다.
                  bytes 는 **닿은 상한**이고 총 길이가 아니다 (R3.9) —
                  그 값의 뜻이 한 번 갈렸다 (4.1.3)
```

**되여는 값이 문서 셋 한 번인 것이 이 길을 고른 근거다.** 실측 —
`git rev-list --count main..HEAD` 가 **13** 이다. `unit/transcript` 가 아직
`main` 에 안 갔고, 그래서 되여는 것이 코드가 아니라 이 문서 넷이다.

**이 계획이 그 대가를 미리 적어 두지는 않았다.** 6.4 가 적은 것은 이 유닛이
**다음 유닛에 주는** 이음매 넷이고, **다음 유닛이 이 유닛에 돌려보내는** 길은
한 줄도 없다. 4.1.3 이 그 빈자리를 진다.

- [x] `Kind` 를 일곱으로 늘린다 — `domain-entities.md` 3절 · 3.2 사상표
- [x] `enode.` 갈래를 규율 한 줄로 세운다 — `domain-entities.md` 3.3
- [x] 판정 사다리에 자리를 준다 (`raw` 로 안 떨어지게) —
      `business-logic-model.md` 2절 ② · 2.3
- [x] 규칙 번호를 새로 부여한다 — **R3.5 · R3.6 · R3.7 · R3.8**.
      기존 R3.1 ~ R3.4 의 뜻을 안 흔들었다 (R3.3 의 「여섯」이 「일곱」이 된
      것 하나가 그 갈래를 센 결과다)
- [x] 찍는 조건을 **안 적는다** — R3.8 이 U3 을 가리키기만 한다
- [x] 6절의 고칠 문서 목록을 갱신한다

### 4.1.3 이 회차가 배운 것 — 유닛 경계를 넘는 조합은 한 유닛이 못 본다

**U1 의 모순 검사가 부실했던 것이 아니다.** 그 검사가 볼 수 있는 것은
**이 유닛의 답끼리**이고, 이번에 막은 것은 **U3 의 답 둘이 합쳐져 U1 의 입력에서
사라진 사실**이다. U1 의 답 열둘을 아무리 맞대어도 그 조합은 시야에 안 들어온다.

```text
   한 유닛의 모순 검사가 보는 것   자기 답끼리의 쌍.  이 계획 4.1.1 의 일곱
   못 보는 것                    다른 유닛의 답이 자기 입력의 모양을 바꾸는 것
   오늘 막은 자리                U3 이 자기 검사에서 찾았다 — 그 유닛의 입력이
                                줄어든 것이 그 유닛의 눈에는 보였다
   남는 위험                     **어느 유닛의 눈에도 안 보이는 조합.**
                                회차의 검증(aidlc-verify)이 그 자리다
```

**그 「남는 위험」이 곧바로 실물로 났다** (2026-09-15 · 같은 날 두 번째).
`enode.capped` 의 `bytes` 를 **U1 은 총 길이로, U3 은 상한으로** 정의했다.
**두 유닛 다 자기 모순 검사에서 0 이었다** — 각자의 문서 안에서는 아무것도
안 어긋났기 때문이다. **웨이브를 닫는 자리에서 진행자가 둘을 대 보고서야
보였다.** 예측이 맞았다는 증거이고, 그래서 이 표의 마지막 줄이 분위기가 아니라
절차다 — 고친 자리는 R3.9 와 `domain-entities.md` 4.4.

**한 벌로 못 만드는 값은 맞대는 절차가 따로 있어야 한다.** `enode.elided` 는
짓는 함수가 이 패키지에 있어 왕복 시험이 잡지만 `enode.capped` 는 그렇지
않다 — `business-rules.md` 16.1 이 Code Generation 에 맞댈 것 둘을 이름으로 넘긴다.

**웨이브 배치가 이 위험을 줄인다** — `unit-of-work-dependency.md` 2절이 U1 ·
U2 · U3 을 같은 W0 에 뒀고, 그래서 U3 의 검사가 U1 의 병합 **전**에 돌았다.
뒤 웨이브였으면 되여는 값이 문서 셋이 아니라 코드였다. **같은 W0 이 두 번 다
값을 냈다.**

### 4.2 산출물 셋

자리는 `aidlc-docs/taeels/construction/transcript/functional-design/` 다.

- [x] **`domain-entities.md`** — 이 유닛이 드는 형식
  - [x] `Kind` 의 값과 각각의 뜻 (Q2 가 개수를 정한다). **닫힌 어휘인가**를
        한 줄로 적는다
  - [x] `Event` 의 필드 — 이름 · 타입 · 비었을 때의 뜻. 도구 이름 · 성공 여부 ·
        토큰 수 · 본문 · 잘림 표시 (Q1 · Q5 가 정한다)
  - [x] `Result` 의 필드 셋과 `Raw` 의 정확한 뜻 (Q2)
  - [x] `Elided` 와 그 와이어 꼴 `{"type":"enode.elided","events":N,"bytes":B}`.
        **짓는 쪽과 읽는 쪽이 같은 구조체를 본다**는 것을 자리로 적는다 (1.1)
  - [x] `logShell` 의 필드 다섯과 그것이 읽히는 규칙 — `tools` 배열 ·
        `ok` 의 삼상태(없음 · 참 · 거짓) · `tokens` 의 키 넷
  - [x] 옮겨 오는 일곱과 타입 둘의 최종 이름과 시그니처 (Q7 · Q8)
  - [x] `transcript.Kind` 와 `enode.EventKind` 의 관계 (Q9)

- [x] **`business-logic-model.md`** — 읽는 순서와 흐름
  - [x] `Parse` 의 파이프라인 — 머리 자르기(`truncated`) · 줄 나누기 ·
        꼬리 판정(Q6) · 줄마다의 판정 · 사건 짓기 · `Elided` 걷기
  - [x] 원문 줄기와 껍데기 줄기가 갈리는 자리 하나와 그 판정식 (Q4)
  - [x] `tool_result` 를 그 `tool_use` 에 붙이는 규칙과, **붙일 호출이 없을 때**
        (링이 감겨 호출 줄이 잘렸다 · 껍데기라 `id` 가 없다)의 흐름
  - [x] `selectLogs` 가 `transcript` 를 부르는 뒤의 모양 — 남는 것과 옮긴 것의
        경계선을 코드 자리로 (Q8)
  - [x] 입력 넷(1.5)이 각각 어느 줄기를 타는지의 표
  - [x] 「링에서 읽든 record 에서 읽든 같은 사건 열」이 **어디까지 참인가** —
        선별본은 본문이 없으므로 사건의 **열**은 같고 **본문**은 다르다.
        FR-3 수용 기준의 글자를 그 값으로 좁혀 적는다

- [x] **`business-rules.md`** — 규칙과 불변식
  - [x] 사건을 안 버린다 — 입력의 줄 수와 `len(Events) + Elided` 의 관계를
        **셀 수 있는 문장**으로 (Q1 의 답이 이 식을 정한다)
  - [x] 마스킹 금지와 상한 자르기가 어떻게 같이 서나 (Q5). 원문 토글이 그
        보증을 어디서 지는지를 자리로
  - [x] 아는 키만 읽는 규율 (SECURITY-13) — 모르는 키 · 모르는 모양 · 모르는
        Kind 의 셋이 각각 어디로 떨어지나
  - [x] 잘린 입력의 규칙 둘 — 머리(첫 개행 뒤부터)와 꼬리(Q6)
  - [x] 임포트 금지 넷(+ 넷째 줄)과 **그 검사가 시험 임포트를 안 본다**는
        한계를 이름으로 (1.6 · Q10)
  - [x] 파서에 실패 등급이 0 이라는 것과, 그래서 오류를 안 돌려준다는 것
  - [x] 커버리지 하한 80% 가 이 새 패키지에 어떻게 걸리나 (1.7 의 실측값)

**셋 다 `frontend-components.md` 를 안 낸다** — 3절 표의 마지막 줄이 그 근거다.

---

## 5. 이 단계가 안 만드는 것

```text
   코드            한 줄도 안 쓴다.  다음 단계다
   시험 목록        Code Generation 계획이 낸다.  픽스처의 자리도 거기다 (Q11 의 답 위에서)
   검사기 파일       파일 행렬 6.3 이 Code Generation 에 넘겼다.  이 단계는 규칙만 낸다
   NFR             U1 은 NFR Requirements 를 **돈다** (unit-of-work.md 10절) —
                   다음 단계이고 이 계획이 그 값을 미리 안 적는다
   인프라           배포 변경 0
```

---

## 6. 회차 밖으로 낼 것 — 확정

**이 유닛이 지금 고치지 않는다.** 회차 루트(`aidlc-docs/v3-run-transcript/`)와
요구 팩(`requirements/`)은 **진행자의 것**이고, `CONVENTIONS.md` 3.4 가
「안 싣는 것 — 남의 문서 루트」로 그 선을 적었다. 여기 적는 것은 **무엇을 ·
누가 · 언제**다.

### 6.1 고쳐야 할 문서 일곱

| | 문서 | 무엇을 | 누가 | 언제 |
|---|---|---|---|---|
| ① | `.../application-design/unit-of-work-file-matrix.md` | 1절 U1 행의 「경계 검사 표에 **두 줄**이 는다」 -> **넷**(R9.1 · R9.2)이고 `api/ui -> store` 까지 **다섯**(R9.3). 6.3 의 「자리는 Code Generation 이 정한다」는 그대로 참이다 | 진행자 | **U2 착수 전.** U2 가 같은 행렬을 읽고 `runner.go` 순서를 딛는다 |
| ② | `.../application-design/component-methods.md` | 1.2 의 `map[string]any` -> `Fields`(= `map[string]json.RawMessage`). 1.1 의 `Event` 자리표(`/* ... */`)는 **안 고쳐도 된다** — 그 문서가 「필드는 Functional Design 이 닫는다」고 넘겼고 이 유닛의 `domain-entities.md` 4절이 그 자리다 | 진행자 | 이 FD 승인 뒤. **U4 착수 전** — U4 가 `as=events` 를 그 겉면 위에 짓는다 |
| ③ | `.../application-design/components.md` | 1절 「옮겨 오는 것 **셋**」 -> 함수 일곱 + 타입 둘 (계획 1.1). 「들어오는 입력이 **둘**」 -> 넷 (계획 1.5). 「아는 것」 줄의 `enode.elided` 옆에 **`enode.capped`** 가 는다 (R3.5) | 진행자 | 같다 |
| ④ | `.../requirements/requirements.md` (회차) | **두 자리다.** 5.1 의 「실측한 stream-json 줄을 `testdata` 에 둔다 — 실제로 받았던 것의 기록이므로 고치지 않는다」 -> 자리는 `testdata` 로 두되 「고치지 않는다」를 뺀다 (R15.2 · 원문은 저장소에 안 싣는다 · ⑳). FR-3 의 사건 종류 목록 **여섯 -> 일곱** (`capped` 가 는다 · R3.3) | 진행자 | 이 FD 승인 뒤. **이 유닛의 Code Generation 전** — 픽스처 파일이 그때 생긴다 |
| ⑤ | `.../user-stories/user-stories.md` (회차) | US-4 의 확인 글자 「`raw` 사건」 -> 「`plain text` 또는 `raw`」. 스토리의 값은 안 바뀐다 — 둘 다 줄 원문을 `Text` 에 그대로 든다 (R3.1 · R3.2) | 진행자 | 같다 |
| ⑥ | `requirements/transcript/decisions.md` (팩) | **두 자리다.** 2절의 `raw` 정의 「파싱 실패 줄의 원문」 -> 「JSON 인데 Kind 를 모르는 줄」 (파싱 실패 줄은 `text`/`plain` 이다 · Q3 = A). 같은 절의 사건 종류 **여섯 -> 일곱** — `capped` 는 하네스가 내는 것이 아니라 **enode 가 찍는 것**임을 함께 적는다 (R3.5 · R3.6) | 진행자 | 같다 |
| ⑦ | `.../application-design/unit-of-work.md` (회차) | U1 절의 「껍데기를 짓는 **셋**(`ParseLine` · `Shell` · `ElidedMarker`)」 -> 함수 일곱 + 타입 둘 (계획 1.1). ①과 같은 사실이 유닛 정본에도 적혀 있다 | 진행자 | **U2 착수 전.** ①과 같은 때다 |

**실측 행을 팩에 안 더한다.** `requirements/harness-components/decisions.md`
6절 ⑳ 이 이미 사건 종류의 실측을 진다. 이 유닛이 행을 더하면 실측이 두 벌이
되고 그것이 CONVENTIONS 1.4 가 막는 자리다 (R11.3).

### 6.2 안 고치는 문서 둘

```text
   requirements/transcript/scene-gates.md   U1 은 지는 게이트가 0 이고 CB0 ~ CB6 의
                                            명령 중 이 유닛이 깨는 것이 0 이다

   enode-design/ (정본)                      이 유닛이 정본에 닿는 자리가 0 이다.
                                            agent-runtime R3 · R4 는 Decode 의 것이라
                                            U2 가 지고, 되돌려 올리는 것은
                                            팩 decisions.md 6절이 진행자에게 넘겼다
```

### 6.3 `GLOSSARY.md` — Q12 = A

```text
   이 유닛이 들여오는 새 글자    0.  transcript 는 패키지 이름이고 축약이 아니다.
                               U1 ~ U8 · CB · NC 는 앞선 커밋이 들여왔다

   지는 빚                      CB0 ~ CB6 의 행이 없다.  그리고 GLOSSARY.md 자체가
                               아직 커밋되지 않았다 (git ls-files 가 0 을 낸다)

   누가 · 언제                  **진행자**가 이 유닛 밖에서 한 커밋으로 main 에 올린다.
                               U2 가 같은 자리를 그렇게 처리한 선례가 있다
                               (contract-vocab 질문 7 의 답 B)

   푼 말                        **짐작하지 않는다.**  CP · CA 와 같이 「없다」로 적거나
                               사람이 값을 준다.  CLAUDE.md 가 그것을 명시로 금지한다
```

### 6.4 다음 유닛으로 가는 이음매 넷

```text
   U2 node-stream    Q9 = A — enode.Event 를 안 섞는다.  U2 가 자기 Kind 를 정한다.
                     파일 행렬 2절이 runner.go 를 U1 -> U2 로 못 박았으므로
                     이 유닛이 먼저 덜어낸다.  U2 는 그 뒤의 파일 위에서 tee 를 잇는다

   U4 log-api        Event 의 JSON 태그와 as=events 의 응답 형식.  이 유닛이 안 정한다.
                     R8.1 이 파서에서 자르므로 그 응답이 원문만큼 커지지 않는다 —
                     N1 의 부하 계산이 그 값을 쓴다

   U5 · U6           Shell · Sub · Cut · Result.Elided · Result.Partial 다섯이
                     카드가 그릴 값이다 (business-rules 16.2).  US-6 이 Shell 로 선다

   U8 fleet-card     같은 다섯을 as=events 로 받는다.  「같은 모양」의 기계적 뜻은
                     business-logic-model 4절의 표다
```

**코드 게이트의 한 값을 미리 적어 둔다.** `internal/transcript` 는 새 패키지라
패키지별 80% 하한이 처음부터 걸리고, 옮겨 오는 65 문장이 96.9% 로 시작한다
(1.7). **하한에 가장 가까워지는 자리는 `Parse` 다** — 그 함수의 가지를 시험이
안 밟으면 65 문장의 여유를 금방 먹는다. Code Generation 이 표준 명령으로 다시 잰다.
