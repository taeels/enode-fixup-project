# U1 `transcript` — Code Generation 계획

**유닛** `transcript` (U1) · **브랜치** `unit/transcript` · **담당** taeels ·
**지는 게이트** 없다 (코드 게이트만) · **선행** 없다 (W0 · 의존 0)

**이 계획이 Code Generation 의 정본이다.** 여기 없는 것은 안 짓는다.
설계는 `construction/transcript/functional-design/` 셋이고 규칙 번호
R1 ~ R15 는 그 `business-rules.md` 다. 값은 `nfr-requirements.md` 가,
규율이 무엇에 의해 지켜지는지는 `nfr-design/nfr-design-patterns.md` 가 든다.
**어긋나면 `enode-design` 이 이긴다** (`requirements/canon.md` 머리말).

이 계획의 숫자는 전부 **이 브랜치의 코드에 대고 다시 잰 것**이다. 앞 문서와
어긋난 자리는 1.3 이 이름으로 적는다.

---

## 0. 앞 단계가 이 계획에 넘긴 것 — 넷

앞 단계들이 **판단을 미룬 자리를 이름으로 적고** 넘겼다. 이 계획이 그 넷을 닫는다.

```text
   ①  경계 검사기의 자리와 명령    파일 행렬 6.3 · business-rules.md 9.3
                                 -> 5절
   ②  enode.capped 를 무엇으로 맞대나   business-rules.md 16.1 · NFR Design 3.2
                                 -> 6절
   ③  tool_use_id 의 실측         business-logic-model.md 5.2 · R12
                                 -> Step 2
   ④  elidedMarker 의 안 밟히는 가지를 어떻게 다루나   business-rules.md 14절
                                 -> 8.3.  안 고치고 안 덮는다
```

**NFR Design 은 SKIP 이 아니다** — 이 유닛은 돌았다 (`nfr-design/` 둘).
그래서 이 계획은 값을 새로 고르지 않고 **파일 · 함수 · 순서 · 재는 명령**만 정한다.

---

## 1. 이 유닛이 내는 diff 의 모양

### 1.1 새 파일과 고치는 파일

```text
   새 패키지 하나    internal/transcript.  제품 파일 넷 · 시험 파일 다섯 · testdata
   고치는 파일 셋    internal/enode/runner.go · internal/enode/logs_test.go ·
                   internal/panel/boundary_test.go
   새 시험 파일 하나  internal/enode/logs_events_test.go
   go.mod · go.sum   diff 0.  새 의존 0 (tech-stack-decisions.md 2.2)
```

| 파일 | 신규 · 고침 | 무엇 | 규칙 |
|---|---|---|---|
| `internal/transcript/transcript.go` | 신규 | 패키지 주석 · `Kind` 일곱 · `Event` · `Info` · `Server` · `Result` · `Elided` · `Fields` | R3.3 · R2.1 |
| `internal/transcript/line.go` | 신규 (옮김) | `SplitLines` · `ParseLine` · `String` · `Bool` · `Int` | R2 · R11.1 |
| `internal/transcript/shell.go` | 신규 (옮김) | `Shell` · `logShell` · `ElidedMarker` · `elidedMark` · `usageTokens` | R13 · R10.3 |
| `internal/transcript/parse.go` | 신규 (새 코드) | `Parse` 의 파이프라인 여덟 · 판정 사다리 넷 · 룬 경계 자르기 | R1 · R3 ~ R8 |
| `internal/enode/runner.go` | 고침 | 여덟과 타입 둘을 덜어내고 `selectLogs` 의 다섯 자리를 다시 부른다 | R11.2 |
| `internal/enode/logs_test.go` | 고침 | 픽스처를 파일로 · 한 줄 · 표 시험 하나 이사 | R15.3 |
| `internal/enode/logs_events_test.go` | 신규 | FR-3 의 두 경로 시험 (`selectLogs` 가 비공개라 여기여야 한다) | 9절 |
| `internal/panel/boundary_test.go` | 고침 | 금지 표로 바꾸고 다섯 줄을 더한다 | R9 · 5절 |

**행렬 밖 파일을 안 만진다.** `internal/store` · `internal/contract` ·
`internal/api` · `cmd/` 다섯의 diff 가 0 이다 (`execution-plan.md` 6절 품질 게이트 2).

### 1.2 덜어내는 것 — 실측으로 센 여덟과 타입 둘

`components.md` 1절과 `unit-of-work.md` U1 절이 **셋**으로 적었다. **코드에 대고
다시 셌고 열이다** (함수 여덟 + 타입 둘). 줄 번호는 이 브랜치의 `runner.go` 다.

| 오늘 자리 | 옮긴 뒤 | 공개 | 가는 파일 |
|---|---|---|---|
| `runner.go:370` `splitLines` | `SplitLines` | 공개 | `line.go` |
| `runner.go:386` `parseEventLine` | `ParseLine` | 공개 | `line.go` |
| `runner.go:513` `eventString` | `String` | 공개 | `line.go` |
| `runner.go:521` `eventBool` | `Bool` | 공개 | `line.go` |
| `runner.go:529` `eventInt` | `Int` | 공개 | `line.go` |
| `runner.go:425` `elidedMarker` | `ElidedMarker` | 공개 | `shell.go` |
| `runner.go:434` `eventShell` | `Shell` | 공개 | `shell.go` |
| `runner.go:487` `usageTokens` | `usageTokens` | **비공개** | `shell.go` |
| `runner.go:407` `logShell` (타입) | `logShell` | **비공개** | `shell.go` |
| `runner.go:419` `elidedMark` (타입) | `elidedMark` | **비공개** | `shell.go` |

**덜어내는 구간은 `runner.go:369` 부터 `runner.go:535` 까지 한 덩어리다** —
사이에 남는 것이 없다. `selectLogs`(315-367)가 그 앞이고 `readCannot`(541-)이
그 뒤다. **`selectLogs` 는 안 옮긴다** (R11.2 · `business-logic-model.md` 7.1).

**실측 — 그 구간은 문장 64 개이고 62 개가 덮여 있다** (2026-09-15 ·
`go test ./internal/enode/ -count=1 -coverpkg=./internal/enode/`).

### 1.3 실측이 앞 문서와 어긋난 자리 — 넷

**전부 코드에 다시 대서 찾은 것이고, 넷 다 값의 방향을 안 바꾼다.**

| | 앞 문서 | 실측 | 무엇이 달라지나 |
|---|---|---|---|
| ① | `business-rules.md` 14절 「옮겨 가는 문장 **65** · 덮임 **63**」 · 「옮긴 뒤 1,908 / 2,249」 | **64 · 62** · 옮긴 뒤 **1,909 / 2,250 = 84.8%** | 비율(96.9% · 84.8%)이 안 바뀐다. 절대 수만 하나씩 어긋났다 |
| ② | `tech-stack-decisions.md` 2.3 「커버리지 분모 — `FuzzParse` 의 몸통 문장이 몇 개 는다」 | **안 는다.** 커버리지 프로파일에 `_test.go` 블록이 **0** 개다 (실측) | 퍼즈와 벤치마크가 커버리지 예산을 한 문장도 안 먹는다. 8.2 가 그 예산을 다시 센다 |
| ③ | `nfr-requirements.md` 2.2 · `business-rules.md` 8.2 「그 둘의 `Text` 는 JSON 을 거쳐 오므로 표준 라이브러리가 잘못된 UTF-8 을 이미 `U+FFFD` 로 바꾼다」 | **`tool_result` 만 참이고 `tool_use` 는 거짓이다** (7.2 의 실측) | **불변식 F2 의 글자를 고쳐야 한다.** 7.2 가 고친 글자를 든다 |
| ④ | `domain-entities.md` 1절의 제목 「옮겨 오는 범위 — **함수 일곱**과 타입 둘」 · FD 계획 6절 ③ ⑦ 의 「함수 일곱 + 타입 둘」 | **함수 여덟과 타입 둘 — 합 열.** `sed -n '369,535p' \| grep -c '^func '` 가 **8**, `'^type '` 가 **2** | 1.2 의 표가 옮길 이름을 정한다. **구간도 문장 수도 안 바뀐다** — 셈 하나다 |

**③ 이 이 계획에서 가장 무거운 발견이다.** 안 고치면 `FuzzParse` 가 첫 변이에서
빨개지고, 그때 고칠 자리가 시험이 아니라 **불변식의 정의**라 Part 2 가 멈춘다.

**④ 는 앞 문서 안에서 이미 어긋나 있었다.** `domain-entities.md` 1절은 제목이
「일곱」인데 **그 절의 표가 열 행**이다 (실측 — `grep -c "^| \`runner.go"` 가 10).
표가 옳고 제목이 하나 모자랐다. 회차 문서의 「셋」을 고치면서 한 번 고친 값인데
**고친 값도 하나 모자랐고**, 이 계획이 그 제목을 그대로 옮기다가 **자기 표(여덟
행)와 어긋났다.** 진행자의 표본 검증이 그것을 잡았다.

**세는 자리를 하나로 줄여 적는다** — 옮기는 것의 개수를 글자로 세지 않고
**1.2 의 표 행이 정본**이다. 글자는 그 표를 가리키기만 한다.

---

## 2. 병렬 — 파일 축으로 셋. 겹치는 파일 0

**가르는 축은 단계가 아니라 파일이다** — 같은 파일을 둘이 열면 병렬이 아니라
충돌이다 (obs 계획 1절과 같은 판정).

```text
   갈래   파일                                              겹치는 파일
   ────   ───────────────────────────────────────────────   ───────────
   A      internal/transcript/**                            없다
   B      internal/enode/runner.go · logs_test.go ·          없다
          logs_events_test.go
   C      internal/panel/boundary_test.go                   없다
```

```text
   Step 1 (겉면) ──> A ──┐
                        ├──> Step 18 (게이트)
                   B ───┤
                   C ───┘
```

**셋 다 Step 1 의 겉면을 딛는다.** 그 글자는 `domain-entities.md` 7절이 이미
못 박았으므로 3절에 그대로 옮겨 고정하고, 셋이 동시에 그 계약 위에서 짠다.

**B 와 C 는 A 가 컴파일돼야 초록이 된다** — B 는 `transcript.X` 를 부르고
C 는 `go list -deps ./internal/transcript` 를 돌린다. **짜는 것은 병렬이고
재는 것은 A 뒤다.** 그것이 이 갈래가 실제로 버는 것이다.

**Step 2 (실물 하네스 실측)가 A 보다 앞이다.** 그 답이 `parse.go` 의 한 자리와
`testdata` 의 내용을 정한다 (11절 물음 1).

---

## 3. 못 박는 겉면 — 세 갈래가 같은 글자를 본다

`domain-entities.md` 7절 그대로다. **여기서 짐작하지 않는다.**

```go
// 읽는 쪽
func Parse(b []byte, truncated bool) Result

// 짓는 쪽 (internal/enode 의 selectLogs 가 쓴다)
func ParseLine(line []byte) (Fields, string, bool)
func Shell(obj Fields, typ string) []byte
func ElidedMarker(events, bytes int) []byte

// 줄과 키를 읽는 도우미
func SplitLines(b []byte) [][]byte
func String(obj Fields, key string) string
func Bool(obj Fields, key string) (bool, bool)
func Int(obj Fields, key string) (int, bool)

// 형식
type Fields = map[string]json.RawMessage   // 별칭이다. 정의 타입이 아니다
type Kind string
type Event struct{ Kind, Sub, Line, Text, Name, ID, OK, Cut, Shell, Tokens, Info }
type Info struct{ Model, Version, Tools, Servers, Reason, Turns, CostUSD, Bytes }
type Server struct{ Name, Status string }
type Result struct{ Events, Elided, Raw, Lines, Head, Partial }
type Elided struct{ Events, Bytes int }
```

**`ElidedMarker` 의 인자를 `Elided` 하나로 안 바꾼다.** 오늘 호출자
(`runner.go:361`)가 정수 둘을 손에 들고 있다 — 옮기는 일에 모양 바꾸기를 안 섞는다.

**`Elided` 에 JSON 태그를 안 단다.** 필드 이름이 `Events` · `Bytes` 이고
`encoding/json` 이 키를 대소문자 무시로 맞추므로 태그 없이도 `{"events":N,"bytes":B}`
가 그대로 들어온다. Step 13 이 그 성질을 한 줄로 쓴다.

---

## 4. 단계 — 열아홉

### Step 1 — 겉면을 못 박는다

- [ ] 3절의 글자를 `internal/transcript/transcript.go` 의 선언으로 먼저 놓는다.
      몸통은 비어도 된다 — **세 갈래가 같은 시점에 같은 이름을 본다**
- [ ] 패키지 주석에 적는 것 넷 — 파서가 아래를 모른다는 방향(R9) ·
      짓는 쪽과 읽는 쪽이 한 패키지인 이유(NFR Design 2.1) ·
      `Kind` 가 닫힌 어휘라는 것(R3.3) · 금지 넷의 검사기가 어디 있는지(5.4)

### Step 2 — 실물 하네스로 재는 것 둘 (A 보다 앞이다)

**환경에 `claude 2.1.271` 이 있다** (실측 · 표 ⑳ 의 `2.1.266` 보다 나중이다).

- [ ] `-p --output-format stream-json --verbose` 로 도구를 부르는 한 턴을 돌린다
- [ ] ① `user` 의 `tool_result` 블록에 **`tool_use_id` 키가 있는가** (R12 · 물음 밖)
- [ ] ② 그 블록의 **`content` 가 문자열인가 배열인가** (물음 1 — 11절)
- [ ] 결과를 `code-summary.md` 에 적는다. **원문은 저장소에 안 싣는다** (⑳ 의 규율)
- [ ] 실측한 모양을 손으로 지어 `testdata/lines/` 에 넣는다 (R15.2)
- [ ] **못 돌리면 멈추지 않는다** — ① 은 R12.2 로 떨어지고(규칙이 키의 존재에
      안 기댄다) ② 는 물음 1 의 A 로 간다. **못 쟀다는 것을 `code-summary.md` 에
      그대로 적는다** — 잰 것이 없는 것이지 초록이 아니다

### Step 3 — 이음매의 글자를 못 박는다 (6절)

- [ ] `shell.go` 에 `const capped = "enode.capped"` 하나. **글자가 한 자리다**
- [ ] `testdata/lines/capped.json` 에 `{"type":"enode.capped","bytes":10485760}`
- [ ] `transcript.go` 의 `Info.Bytes` 주석에 **닿은 상한**임을 적고 U3 을 가리킨다
      (R3.8 — 찍는 조건을 이 유닛이 안 적는다)

---

### 갈래 A — `internal/transcript/**`

### Step 4 — `transcript.go` — 형식

- [ ] `Kind` 일곱과 상수. **닫힌 어휘임을 주석이 적는다** (R3.3)
- [ ] `Event` 의 필드 열하나. `OK *bool` 의 삼상태 · `Cut` 이 잘린 바이트 수이지
      원래 길이가 아님 · `Shell` 이 US-6 을 세운다는 것을 주석이 적는다
- [ ] `Info` · `Server` · `Result` · `Elided` · `Fields`
- [ ] `Fields` 가 **별칭**인 이유를 적는다 — 정의 타입이면 경계마다 변환이 생기고
      그 변환이 하는 일이 0 이다 (`domain-entities.md` 2절)
- [ ] `map[string]any` 를 안 쓰는 이유를 적는다 (R2.1 · SECURITY-13)

### Step 5 — `line.go` — 읽는 통로 (옮기기)

- [ ] 다섯을 `runner.go` 에서 그대로 옮긴다. **몸통을 한 줄도 안 바꾼다**
- [ ] 이름만 바꾼다 — 1.2 의 표 그대로. `map[string]json.RawMessage` 를
      `Fields` 로 적는다 (별칭이라 뜻이 같다)
- [ ] `runner.go:381-385` 의 근거 주석(구조체로 한 번에 안 받는 이유)과
      `runner.go:511-512` 의 주석(모양이 다르면 없는 것으로 본다)을 **함께 옮긴다**

### Step 6 — `shell.go` — 짓는 통로 (옮기기)

- [ ] 다섯을 그대로 옮긴다. `logShell` 의 필드 다섯과 JSON 태그를 안 바꾼다 (R13)
- [ ] `elidedMark` 의 `"enode.elided"` 도 안 바꾼다 (R13)
- [ ] `runner.go:399-406` 의 근거 주석(지우는 쪽이 아니라 짓는 쪽) ·
      `runner.go:415-418`(점 있는 `type` 은 우리 것) · `runner.go:466-468`
      (`estimated_tokens` — 범위를 넓힌 자리)을 **함께 옮긴다**
- [ ] Step 3 의 `capped` 상수를 여기 둔다 — `enode.` 표시 줄의 글자가 한 파일이다

### Step 7 — `parse.go` — `Parse` 의 파이프라인 여덟

`business-logic-model.md` 1절의 순서 그대로다. **순서가 값이다.**

- [ ] ① 머리 자르기 — `truncated` 면 첫 개행 뒤부터. 버린 수를 `Head` 에 (R7.1).
      개행이 하나도 없으면 빈 `Result`
- [ ] ② 꼬리 자르기 — 개행으로 안 끝나면 떼고 `Partial` 에 (R7.2)
- [ ] ③ `SplitLines`. **`selectLogs` 와 같은 함수다** (R11.1)
- [ ] ④ 판정 사다리 넷 — 2절의 순서. **②가 ③ · ④보다 위다** (R3.5 · 2.3)
- [ ] ⑤ 사건 짓기 — 원문 줄기와 껍데기 줄기 (3.1 · 3.2). 블록 0 도 사건 하나 (R5)
- [ ] ⑥ 표시 줄 가르기 — `enode.elided` 는 `Result.Elided`, `enode.capped` 는
      사건 하나, 모르는 `enode.*` 는 `raw` (R3.6 · R3.7). **둘 다 `Lines` 에 센다**
- [ ] ⑦ 붙이기 — 같은 `ID` 의 `tool_use` 에서 `Name` 을 채운다. 짝이 없으면
      빈 채로 선다 (R12.2). **껍데기 줄기에서는 안 붙인다** (R12.3)
- [ ] ⑧ 세기 — `Lines` · `Raw`
- [ ] `Tokens` 는 그 줄이 낸 **첫 사건**에만 (R6)
- [ ] 시계를 안 받는다 · `error` 를 안 돌려준다 (R10.1) · 로거를 안 받는다 (SECURITY-03)

### Step 8 — `parse.go` — 룬 경계 자르기 (7절)

- [ ] 자르는 함수 하나. `tool_use` 200 · `tool_result` 500 바이트 (R8.1)
- [ ] 상한 자리에서 시작해 `utf8.RuneStart` 가 거짓인 동안 뒤로 물린다. **최대 셋**
- [ ] 못 물리면 빈 문자열이고 `Cut` 이 전체 길이다. **멈추지 않는 경로가 0 이다**
- [ ] `len(Text) + Cut == 원래 길이` 가 언제나 참이다 (R8.2)
- [ ] `text` · `thinking` · `plain` · `raw` 는 안 자른다 (R8.3)
- [ ] **안 고친다** — 원래 깨진 바이트는 깨진 채다. 치환도 마스킹도 없다 (R8.4)

### Step 9 — `testdata/` — 픽스처와 README

- [ ] `testdata/lines/` 에 줄 하나씩 파일로 (R15.1). 다섯은 `logs_test.go` 의
      오늘 상수 그대로 — `init` · `assistant-tool` · `assistant-text` ·
      `user-result` · `result`
- [ ] 이 유닛만 쓰는 것 — `elided` · `capped` · `unknown-enode` · `rate-limit` ·
      `hook-response` · `type-not-string` · `plain` · `shell-assistant`
- [ ] `testdata/README.md` — **실측 원문이 아니라 실측한 모양을 손으로 지은
      것**임을 적는다 (R15.2). 고치면 실측과 갈린다는 경고도 같이
- [ ] 읽는 도우미는 `internal/transcript` 와 `internal/enode` **양쪽**에 한 줄씩.
      **파일이 한 벌이다** (R15.3). 꼬리 개행을 떼는 자리가 그 도우미다

### Step 10 — 시험 — 규칙마다 하나

- [ ] `shell_test.go` — `logs_test.go` 의 표 시험 일곱 갈래를 그대로 옮긴다
      (`TestShell_CarriesOnlyWhatWasAllowed`). **기대 문자열을 한 글자도 안 바꾼다** (R13)
- [ ] `shell_test.go` — 누출 시험. 고정 문자열 셋 `/etc/shadow` ·
      `sk-ant-secret` · `root:x:0:0` 이 따라온다 (NFR Design B5)
- [ ] `parse_test.go` R1 — `Lines == 집계 줄 수 + 사건을 낸 줄 수` · 줄 하나를
      지우면 `Lines` 가 하나 준다
- [ ] `parse_test.go` R2 — `usage` 에 문자열과 객체를 섞고 `Tokens` 에 정수 넷만
- [ ] `parse_test.go` R3 — 줄 여섯 (비JSON · `type` 이 숫자 · `rate_limit_event` ·
      `system/hook_response` · `enode.capped` · `enode.wibble`). **`capped` 의
      `Info.Bytes` 가 줄의 `bytes` 그대로**임을 잰다 (R3.9 — 총 길이로 안 고친다)
- [ ] `parse_test.go` R4 — 같은 `assistant` 를 원문과 `Shell()` 결과로 넣어
      `Shell` 이 거짓/참으로 갈리는지
- [ ] `parse_test.go` R5 — `{"type":"assistant"}` 하나에 `len(Events) == 1`
- [ ] `parse_test.go` R6 — `tool_use` 블록 둘인 줄에서 `Tokens` 가 첫 사건에만
- [ ] `parse_test.go` R7 — 같은 바이트를 개행 있이/없이. `len(Events)` 가 하나
      차이나고 `Partial` 이 그 줄의 길이
- [ ] `parse_test.go` R8 — ① 600 바이트 도구 결과에 `len(Text)==500` · `Cut==100`
      ② 같은 길이의 `text` 블록은 **안 잘린다** ③ 500 번째가 한글 룬 가운데면
      `len(Text)` 가 498 로 물러나고 `len(Text)+Cut` 이 원래 길이와 같다
- [ ] `parse_test.go` R10.2 — 입력 다섯(빈 바이트 · 개행만 · 잘린 JSON ·
      거대한 한 줄 · 이어바이트로만 시작하는 줄)에 패닉 0
- [ ] `parse_test.go` R12 — 붙는 경로 하나와 못 붙는 경로 셋 (5.1 의 ①②③)
- [ ] `roundtrip_test.go` — **왕복** (NFR Design B4). `Shell` 이 지은 바이트를
      `Parse` 에 먹여 `Kind` · `Name` · `OK` · `Tokens` 가 서고 `Shell` 이 참인지.
      **`Shell` 의 JSON 태그를 바꾸면 이 시험이 같은 패키지 안에서 빨개진다**

### Step 11 — `FuzzParse` 와 불변식 넷

- [ ] `fuzz_test.go` 에 대상 하나. `FuzzParseLine` · `FuzzShell` 을 **안 만든다**
      (코퍼스가 두 벌이 된다 · `nfr-requirements.md` 4.1)
- [ ] 시드는 Step 9 의 픽스처 그대로 `f.Add` (4.3). **두 벌이 0 이다**
- [ ] F1 — 어떤 입력에도 패닉하지 않는다
- [ ] F2 — **입력이 올바른 UTF-8 일 때만** `Cut > 0` 인 `Text` 가 올바른
      UTF-8 이다. **조건이 붙는 이유는 7.2 다** — 이 글자가 고친 값이다
- [ ] F3 — 줄을 안 버린다. 집계 줄 수는 `Elided != nil` 이면 1, 사건을 낸 줄 수는
      `Event.Line` 의 서로 다른 값의 수
- [ ] F4 — `Head` · `Partial` 이 음수가 아니고 입력 길이를 안 넘는다
- [ ] 추가로 언제나 참인 것 하나 — `len(Text) + Cut` 이 음수가 아니다.
      **입력의 유효성과 무관한 유일한 자르기 불변식이다** (7.2)

---

### 갈래 B — `internal/enode`

### Step 12 — `runner.go` — 덜어내고 다시 부른다

- [ ] `runner.go:369` ~ `:535` 를 지운다. **한 덩어리라 사이에 남는 것이 없다**
- [ ] `selectLogs` 의 다섯 자리를 바꾼다. **로직은 한 줄도 안 바뀐다** (R11.2)

```text
   runner.go:316   splitLines(stdout)              -> transcript.SplitLines
   runner.go:326   parseEventLine(ln)              -> transcript.ParseLine
   runner.go:330   eventString(obj, "subtype")     -> transcript.String
   runner.go:348   parseEventLine(ln)              -> transcript.ParseLine
   runner.go:352   eventShell(obj, typ)            -> transcript.Shell
   runner.go:361   elidedMarker(events, elided)    -> transcript.ElidedMarker
```

- [ ] 임포트를 더한다. **방향이 하나다** — `enode -> transcript` 만 (R9.1 · 7.2)
- [ ] `encoding/json` 이 `runner.go` 에서 아직 쓰이는지 확인하고, 안 쓰이면 뺀다
- [ ] `selectLogs` 머리 주석의 「어댑터가 아니라 여기서 한다」를 **안 지운다** —
      옮기면 그 근거가 갈 곳을 잃는다 (`business-logic-model.md` 7.1)

### Step 13 — `logs_test.go` — 세 자리

**실측 — 이 파일에 시험 함수가 열이고 그중 아홉이 `selectLogs` 를 부른다.**

- [ ] 상수 다섯을 `testdata/lines/` 읽기로 바꾼다 (R15.3). **시험 함수 아홉의
      몸통은 한 줄도 안 바뀐다** — 이름이 그대로이기 때문이다
- [ ] `logs_test.go:134` 의 `var mark elidedMark` 를 `var mark transcript.Elided`
      로 **한 줄** 바꾼다. `Elided` 에 태그가 없어도 `encoding/json` 이 키를
      대소문자 무시로 맞춰 `Events` · `Bytes` 가 그대로 찬다 (3절)
- [ ] `TestLogs_TheShellCarriesOnlyWhatWasAllowed`(`:175-234`)를 **통째로 옮긴다**
      — `parseEventLine` 과 `eventShell` 을 직접 부르므로 이 패키지에 못 남는다.
      간 자리는 Step 10 의 `shell_test.go`
- [ ] **R11.2 의 「한 줄도 안 고친 채」가 그대로는 거짓이다** — 아홉 중 하나가
      `elidedMark` 타입을 쓴다. 고치는 것이 **정확히 한 줄**이고 나머지 여덟은
      손대지 않는다. 그 사실을 `code-summary.md` 에 적는다

**상대 경로의 대가를 이름으로 적는다** — `internal/enode` 의 시험이
`../transcript/testdata/` 를 읽으므로, 파일 이름이 바뀌면 **컴파일이 아니라 실행
때** 깨진다. `go list` 가 못 보는 결합이다. 실패 메시지에 경로를 싣는 것으로 갚는다.

### Step 14 — `logs_events_test.go` (신규) — FR-3 의 두 경로

**`selectLogs` 가 비공개라 이 시험은 `internal/enode` 에만 설 수 있다.**
`internal/transcript` 는 `internal/enode` 를 임포트할 수 없다 (R9.1).

- [ ] 같은 stdout 을 ① 원문 그대로 `transcript.Parse` ② `selectLogs` 를 지나
      `transcript.Parse`
- [ ] **재는 범위를 좁혀 적는다** (`business-logic-model.md` 9절) —
      `Kind` 의 열 · 순서 · `Name` · `OK` 가 **같고** `Text` 와 `Shell` 과 `ID` 는
      **다르다**. 「전부 같다」로 적으면 쓸 수 없는 수용 기준이 된다

---

### 갈래 C — 경계 검사

### Step 15 — `boundary_test.go` — 표로 바꾸고 다섯 줄을 더한다 (5절)

- [ ] 금지 쌍을 **표 하나**로 바꾼다. 오늘은 슬라이스 하나와 `if` 둘이다 (행렬 6.2)
- [ ] 여덟 줄 — 기존 셋 + `api/ui -> store`(빈자리 · 6.3) + `transcript` 의 넷
- [ ] `internal/transcript` 가 **표준 라이브러리만** 쓰는지 한 줄로 잰다 (R9.4)
- [ ] `cmd/enode -> internal/panel` 의 **있어야 한다** 검사는 표 밖에 그대로 둔다 —
      금지 표가 못 담는 모양이다

---

### 닫는 것

### Step 16 — 변이 다섯 — 시험이 실제로 재는지

넣고 **빨개지는지** 본다. 안 빨개지면 그 규칙을 재는 시험이 없는 것이다.

- [ ] ① 사다리에서 ②(`enode.` 갈래)를 걷는다 -> `capped` 가 `raw` 로 떨어져야 빨강
- [ ] ② 껍데기 판정을 `assistant` · `user` 밖으로 넓힌다 -> R4.3 이 빨강
- [ ] ③ `Tokens` 를 사건마다 복사한다 -> R6 이 빨강
- [ ] ④ 룬 경계 물리기를 걷고 바이트로 그냥 자른다 -> R8 ③ 과 F2 가 빨강
- [ ] ⑤ 블록 0 인 줄을 사건 0 으로 넘긴다 -> R1 · R5 가 빨강

### Step 17 — 벤치마크 — 증폭의 수를 잰다

`nfr-requirements.md` 7절이 「벤치마크가 아직 없다. Code Generation 이 잰다」로
넘긴 자리다. **커버리지 분모를 안 먹는다** (1.3 ②).

- [ ] 자리는 `internal/transcript/bench_test.go` 하나다. **시험 파일 다섯째다**
- [ ] `unsafe.Sizeof(Event{})` 를 시험 하나로 못 박는다. **실측 224 바이트**
      (amd64 · 2026-09-15). `nfr-requirements.md` 5.1 의 계산값과 같다
- [ ] `BenchmarkParse` — 파싱 비용이 입력의 약 2배인지 (`-benchmem` 의 `B/op`)
- [ ] 최악의 사건 배열 약 112 MB 는 **곱셈으로 적고 안 돌린다** —
      10 MiB 입력을 벤치마크로 돌리면 CI 가 아니라 사람의 기계가 멈춘다
- [ ] 잰 값을 `code-summary.md` 에 적고 **U4 에 넘긴다** (`nfr-requirements.md` 5절)

### Step 18 — 게이트 (8절)

- [ ] 8.1 의 순서대로 열둘을 돌린다
- [ ] 미달이 하나라도 있으면 **그 자리에서 멈춘다.** 하한을 낮추지 않는다

### Step 19 — 문서

- [ ] `construction/transcript/code/code-summary.md` — 옮긴 열 · Step 2 의 실측
      둘 · Step 17 의 수 · **Step 3 의 이음매를 맞댄 결과**(6절) · 못 잰 것
- [ ] 12절의 표시를 진행자에게 넘긴다. **회차 문서와 팩을 이 유닛이 안 고친다**
- [ ] 이 계획의 체크박스를 전부 `[x]` 로 (규칙 Step 12)
- [ ] `aidlc-docs/taeels/aidlc-state.md` 와 `audit.md` — **진행자의 것이 아니라
      이 담당의 것이다** (`CONVENTIONS.md` 3.4). Part 2 가 싣는다

---

## 5. 경계 검사 — 자리와 명령을 이 계획이 정한다

파일 행렬 6.3 과 `business-rules.md` 9.3 이 이 판단을 여기로 넘겼다.

### 5.1 고른 것 — `internal/panel/boundary_test.go` 에 넣는다

**근거는 실측 하나다** — 그 파일은 **`internal/panel` 을 한 번도 참조하지 않는다.**
임포트가 `os/exec` · `strings` · `testing` 셋뿐이고, 하는 일은 `go list` 를
돌려 출력을 보는 것이다. 이름과 디렉터리만 `panel` 이지 **내용은 이미 저장소
전체의 검사기다** — 오늘도 `internal/enode -> internal/panel` 과
`cmd/enode -> internal/panel` 을 함께 잰다.

```text
   행렬 6.2 가 적은 대가   package panel_test 가 남의 패키지 경계를 검사하게 된다
   실측이 좁힌 것          이미 그렇게 하고 있다.  transcript 넷을 더해도
                        그 성질이 한 칸도 안 나빠진다
```

### 5.2 안 고른 쪽과 그 대가

| | 안 고른 쪽 | 대가 |
|---|---|---|
| ① | `internal/transcript/boundary_test.go` 를 새로 낸다 | **검사기가 두 벌이 된다.** `deps` 와 `has` 도우미가 두 파일에 같은 모양으로 앉는다 — `CONVENTIONS.md` 1.4 가 막는 그것이고 R9.3 이 이름으로 금지했다 |
| ② | 검사기를 중립 패키지(`internal/boundary` 같은)로 옮긴다 | 시험 파일만 있는 패키지가 생겨 `go build ./...` · 스킵 감시의 패키지 수 · 커버리지 표의 행이 **셋 다 흔들린다.** 이 유닛의 diff 모양이 「파서 하나」에서 「빌드 구성 변경」으로 커진다. **옮기는 일에 구조 바꾸기를 안 섞는다** |

**①의 대가가 ②보다 싸 보이지만 아니다** — 두 벌이 된 검사기는 **갈릴 때 아무도
안 본다.** 한쪽만 고쳐도 둘 다 초록이기 때문이다. 그것이 이 유닛이 파서를 한
패키지에 모으는 것과 **같은 근거**다 (NFR Design B4).

**고른 쪽의 남는 대가를 이름으로 적는다** — `internal/transcript` 를 읽는 사람이
그 디렉터리 안에서 자기 경계 규칙을 못 찾는다. **패키지 주석 한 줄이 그것을
가리키게 한다** (Step 1).

### 5.3 명령 — `go list -deps` 다. `-test` 가 아니다

```text
   쓴다        go list -deps <패키지>          오늘 boundary_test.go 가 쓰는 그대로
   안 쓴다     go list -test -deps <패키지>
```

**실측으로 갈랐다** (2026-09-15).

```text
   go list -deps ./internal/api/ui        내부 의존이 자기 하나다
   go list -test -deps ./internal/api/ui  같은 하나에 셋이 더 붙는다 —
                                          ui [ui.test] · ui_test [ui.test] · ui.test
```

**`-test` 는 시험 임포트를 보여주는 대신 목록을 `.test` 패키지로 흐린다.**
Q10 = A 가 시험 임포트를 **안 재기로** 골랐고 근거는 둘이다 — 시험은 제품
바이너리에 안 실리고, `-test` 로 재면 걸러야 할 줄이 는다.

**못 보는 것을 규칙 옆에 적는다** — `internal/transcript/*_test.go` 가
`internal/enode` 를 임포트해도 이 검사는 초록이다. **그 자리는 사람이 진다**
(`business-rules.md` 9.1 · NFR Design 6절 ⑤).

### 5.4 재는 것 — 금지 여덟 줄과 봉인 하나

```text
   금지 표 여덟 줄
     internal/panel      -> internal/store        오늘 있다
     internal/panel      -> internal/api          오늘 있다
     internal/enode      -> internal/panel        오늘 있다
     internal/api/ui     -> internal/store        **빈자리를 메운다** (행렬 6.1)
     internal/transcript -> internal/enode        새로 선다  R9.1
     internal/transcript -> internal/api          새로 선다  R9.2
     internal/transcript -> internal/store        새로 선다  R9.2
     internal/transcript -> internal/panel        새로 선다  R9.2

   봉인 하나 (R9.4)
     go list -deps ./internal/transcript 의 출력에서 첫 경로 조각에 점이 있는
     줄(= 표준 라이브러리가 아닌 것)이 **자기 하나**여야 한다
```

**봉인 한 줄이 금지 넷보다 강하다** — 금지는 이름을 아는 넷만 막고, 봉인은
`internal/match` 든 새 외부 모듈이든 **전부** 막는다. 넷을 그래도 두는 이유는
`requirements.md` 5.2 가 그 넷을 이름으로 적었고 **실패 메시지가 어느 줄인지를
말해야 하기 때문**이다. 두 벌이 아니라 **다른 단언 둘이고 파일은 하나다**.

**실측으로 방법이 선다** (2026-09-15).

```text
   go list -deps ./internal/schema | awk -F/ '$1 ~ /\./'   -> 자기 하나만 나온다
   go list -deps ./internal/config | awk -F/ '$1 ~ /\./'   -> yaml.v3 와 자기
   go list -deps ./internal/store  | awk -F/ '$1 ~ /\./'   -> pgx 무리와 자기
```

### 5.5 오늘 넷이 자동으로 초록인 것이 쓸모없다는 뜻이 아니다

`internal/transcript` 가 표준 라이브러리만 쓰면 넷이 다 초록이다. **값은 나중에
누가 임포트를 더했을 때 빨개지는 데 있다** (행렬 6.3 이 같은 말을 적었다).
`api/ui -> store` 도 같다 — 오늘 코드 diff 가 0 이다 (5.3 의 실측).

---

## 6. 이음매 — `enode.capped` 를 무엇으로 맞대나

**`business-rules.md` 16.1 이 맞댈 것 둘을 이름까지 적어 넘겼다.** 이 절이
그것을 무엇으로 맞대는지를 정한다.

### 6.1 왜 왕복 시험이 못 잡나

| 표시 줄 | 짓는 쪽 | 읽는 쪽 | 왕복 시험 |
|---|---|---|---|
| `enode.elided` | `ElidedMarker` — **이 패키지** | `Parse` | **잡는다** (Step 10) |
| `enode.capped` | `internal/record` — **U3** | `Parse` | **못 잡는다** |

**비대칭이 옳다** — `internal/record` 가 파서를 임포트하면 진행 파일을 쓰는 코드가
화면의 모형을 딛고 의존이 거꾸로 하나 는다 (`domain-entities.md` 5.2).

### 6.2 U3 의 산출물이 이 worktree 에 없다 — 그래서 반쪽이다

**실측 — 이 브랜치에 `enode.capped` 라는 글자가 `internal/` 아래에 0 개다.**
U3 은 아직 `main` 에 없고, 착수 순서가 ①U1 ... ④U3 이라 **U1 이 먼저 병합된다**
(`unit-of-work-dependency.md` 3절). **U1 은 U3 의 코드를 볼 수 없는 시점에 선다.**

그래서 이 계획은 이음매를 **반으로 갈라** 닫는다.

```text
   U1 이 닫는 반쪽   글자를 한 자리에 못 박고 · 그 자리를 시험이 잡고 ·
                   맞댈 대상을 파일로 남긴다.  Step 3 · Step 10 · Step 19
   U3 · 진행자가 닫는 반쪽   U3 이 실제로 그 바이트를 쓰는가.  6.4 의 명령
```

### 6.3 U1 이 무엇에 대고 맞대나 — 사용자가 고른 값

짐작이 아니다. **사용자가 2026-09-15 에 고른 값이 이 worktree 의 문서에 있다**
(`plans/transcript-functional-design-plan.md` 4.1.2).

```text
   와이어    {"type":"enode.capped","bytes":<상한>}
   상한      MaxBlobBytes.  기본 10 MiB — 실측 config.go:51 의 int64 ·
            config.go:100 의 10 << 20
   bytes 의 뜻   **닿은 상한**이다.  총 길이가 아니다 (R3.9)
```

**그 뜻이 한 번 실제로 갈렸다** — U1 이 총 길이로, U3 이 상한으로 적었고 두 유닛
다 자기 모순 검사에서 0 이었다. 웨이브를 닫는 자리에서 진행자가 대 보고서야
보였다 (같은 문서 4.1.3). **이 절이 그 한 번 때문에 있다.**

### 6.4 맞대는 절차 — 셋

- [ ] ① **U1 안에서** — 글자가 `shell.go` 의 상수 하나이고
      `testdata/lines/capped.json` 이 그 바이트를 든다. 시험이 그 파일을 읽어
      `Kind == KindCapped` 이며 `Info.Bytes == 10485760` 임을 잰다 (Step 10 R3)
- [ ] ② **`code-summary.md` 에** — 맞댈 것 둘(`type` 의 글자 · `bytes` 의 뜻)과
      U1 이 든 값을 그대로 적는다. **U3 이 읽을 자리가 그것이다**
- [ ] ③ **U3 이 병합된 뒤 진행자가** — 아래 한 줄을 돌려 두 자리의 글자를 맞댄다

```text
   grep -rn 'enode\.capped' internal/record internal/transcript
```

**③ 은 사람이 집행한다.** 도는 스크립트가 없는 채로 기계 검사라고 적으면 그것이
거짓 초록이다 (`CLAUDE.md` 의 같은 결). **U1 이 ③ 을 초록이라고 적지 않는다** —
잰 것이 없는 자리다. 12절이 그것을 진행자에게 넘긴다.

---

## 7. 단위 갈림 — 숫자는 팩이고 단위는 이 유닛이다

### 7.1 값

```text
   숫자      200 · 500.  팩이 진다 (decisions.md:50).  이 유닛이 안 고른다
   단위      **바이트**.  이 유닛이 정했다 (NFR Requirements 물음 1 = A)
   경계      상한을 넘지 않는 가장 긴 접두 중 **마지막 룬이 온전한 데까지**
   Cut       바이트.  룬 경계로 물러난 바이트도 든다.  len(Text) + Cut == 원래 길이
```

한국어는 UTF-8 에서 글자당 3 바이트라 **같은 숫자가 3 배 다른 값**이다. 팩의
「200자 · 500자」를 진행자가 고친다 (12절 ①). **이 유닛이 팩을 안 만진다.**

### 7.2 실측이 F2 의 글자를 고쳤다 — `tool_use` 는 UTF-8 보증이 없다

`nfr-requirements.md` 2.2 와 `business-rules.md` 8.2 가 이렇게 적었다.

```text
   앞 문서    상한이 걸리는 Kind 가 둘뿐이고 그 둘의 Text 는 JSON 을 거쳐 온다.
             표준 라이브러리가 그 두 길에서 잘못된 UTF-8 을 이미 U+FFFD 로 바꾼다
```

**길 둘을 각각 돌려 봤다** (2026-09-15 · `go1.26.6`).

```text
   json.Unmarshal 로 문자열을 받는다      잘못된 바이트가 U+FFFD 가 된다   맞다
   json.Marshal 로 문자열을 낸다          같다                            맞다
   json.Marshal 로 json.RawMessage 를 낸다   **잘못된 바이트가 그대로 산다**  틀리다
```

```text
   tool_result   JSON 문자열을 디코드한 것이다 -> 자르기 앞에서 이미 올바르다   그대로
   tool_use      도구 입력 객체를 다시 마샬한 것이다 -> compact 를 지날 뿐이라
                 잘못된 UTF-8 이 살아남는다.  **앞 문서의 근거가 이 길에는 안 선다**
```

### 7.3 그래서 F2 를 어떻게 적나 — 고치지 않고 조건을 단다

**세 갈래가 있었고 하나만 규칙을 안 깬다.**

| | 길 | 판정 |
|---|---|---|
| ① | 자른 뒤 `utf8.Valid` 가 거짓이면 파서가 고친다 | **안 된다.** 그것이 마스킹이다 (R8.4 · `constraints.md` §4 · `nfr-requirements.md` 2.3 「원래 깨진 것은 그대로」) |
| ② | `tool_use` 의 입력을 `any` 로 풀었다가 다시 마샬한다 | **안 된다.** 도구 입력을 통째로 메모리에 올린다 — R2.1 이 막은 그 모양이다 |
| ③ | **F2 에 조건을 단다** | **이쪽이다.** 파서의 코드가 한 줄도 안 바뀐다 |

```text
   고친 F2   입력이 올바른 UTF-8 이면, Cut > 0 인 사건의 Text 도 올바른 UTF-8 이다
   왜 재나    FuzzParse 는 입력 바이트를 손에 들고 있다 —
             if utf8.Valid(b) { ... } 한 줄이 그 조건이다.  잴 수 있다
   무엇을 지키나   as=events 의 **조용한 손상**.  바이트로 그냥 자르면 꼬리 룬이
             쪼개지고 json.Marshal 이 그것을 오류 없이 U+FFFD 로 바꾼다.
             그 문은 그대로 닫힌다 — 닫는 조건이 정확해졌을 뿐이다
```

**입력이 이미 깨진 경우에 남는 보장 하나를 따로 적는다** — `len(Text) + Cut` 이
원래 길이와 같고 자른 자리가 `RuneStart` 다 (R8.2). **그것은 입력의 유효성과
무관하게 참**이고 Step 11 의 마지막 줄이 그것을 잰다.

**이것은 값의 변경이 아니라 문장의 교정이다.** 상한도 단위도 자르는 규칙도 안
바뀌었다. 12절 ② 가 고칠 문서 둘을 적는다.

---

## 8. 무엇으로 재는가

### 8.1 순서 — 열둘. 순서가 규칙이다

```text
    1  eval "$(scripts/testdb.sh)"                 커버리지 표준 명령의 env 다
    2  go build ./...
    3  go vet ./...
    4  go test ./... -count=1
    5  go run ./scripts/glyphscan.go                출력 문자열의 장식 문자
    6  grep -rlIP '\x{2605}' --exclude-dir=.git .   상한 0
    7  test -z "$(gofmt -l .)"
    8  test -z "$(git status --porcelain)"          커버리지보다 먼저다
    9  go test ./... -count=1 -coverpkg=./... -coverprofile=/tmp/cover.out
       + ci.yml 의 awk 로 패키지별 80% 하한
   10  git checkout -- cmd/enodectl/probe.lock      9 가 그것을 바꾼다
   11  GOOS=windows 빌드 + 심볼 상한 둘 (net/http 50 · crypto/tls 10)
   12  git diff --stat main -- internal/store internal/contract   **0 이어야 한다**
```

**9 가 이 저장소의 유일한 게이트 입력이다** — `.coverage-contract.yml` 의
`command` 블록이 그렇게 못 박았다. 다른 명령으로 잰 수치는 게이트 입력이 아니다.

**12 가 품질 게이트 2 의 뒷절이다** (`execution-plan.md` 6절). CB0 의 라우트
셈(17 -> 18)은 **U4 의 것이고 U1 에서는 17 그대로**여야 한다.

### 8.2 커버리지 예산 — 다시 쟀다

```text
   오늘의 internal/enode          1,971 / 2,314 = 85.2%     실측
   옮겨 가는 것                      62 /    64 = 96.9%     실측 (runner.go:369-535)
   옮긴 뒤의 internal/enode        1,909 / 2,250 = 84.8%     하한 80% 와 4.8%p
   새 internal/transcript 의 시작      62 /    64 = 96.9%
```

**안 덮인 둘의 자리가 정확히 밝혀졌다** — `runner.go:427`(`elidedMarker` 의
`json.Marshal` 오류 가지)과 `runner.go:476`(`eventShell` 의 같은 가지)이다.

**`Parse` 가 P 문장을 더할 때 허용되는 안 덮인 수**는
`0.2 x (64 + P) - 2` 다. `P = 150` 이면 **40 문장**까지 안 덮여도 초록이다.
**여유가 넉넉하다** — 그래도 Step 10 이 규칙마다 시험을 두는 이유는 커버리지가
아니라 Step 16 의 변이가 빨개지게 하기 위해서다.

**퍼즈와 벤치마크는 예산을 한 문장도 안 먹는다** — 커버리지 프로파일에
`_test.go` 블록이 0 개다 (1.3 ②).

### 8.3 안 덮인 둘을 어떻게 다루나 — 안 고치고 안 덮는다

`business-rules.md` 14절이 이 판단을 넘겼다.

```text
   안 고친다    R10.3 이 「ElidedMarker · Shell 은 마샬 실패에 nil 을 돌려준다.
               옮기면서 안 바꾼다」로 못 박았다
   안 덮는다    그 가지는 **구조상 못 밟는다** — 두 함수가 마샬하는 것이
               string · int · bool · []string · map[string]int · *bool 뿐이고
               encoding/json 이 그 종류에서 오류를 안 낸다.
               밟게 하려면 시험이 타입을 바꿔야 하고 그것이 R13 위반이다
   적는다       그 두 문장이 안 덮이는 이유를 code-summary.md 에 적는다.
               예산 안에 있으므로 게이트는 초록이다
```

### 8.4 퍼즈가 닫는 것과 **안 닫는 것**

**`nfr-design/nfr-design-patterns.md` 4절이 이 표를 세웠다. 이 계획이 그것을
명령으로 옮긴다.**

| | 닫는다 | 무엇으로 |
|---|---|---|
| F1 패닉 0 | 퍼즈 시드 (CI) + 변이 (사람) | `go test ./internal/transcript` 가 시드를 실제로 돌린다 |
| F2 자른 `Text` 의 UTF-8 | 퍼즈 + 표 시험 (Step 10 R8 ③) | 7.3 의 조건 붙은 글자로 |
| F3 줄을 안 버린다 | 퍼즈 + 표 시험 (Step 10 R1) | |
| F4 경계 산술 | 퍼즈 | |

| | **퍼즈가 안 닫는 것** | 무엇이 닫나 |
|---|---|---|
| ① | 누출 — 본문이 껍데기에 실렸나 | `assertNoLeak` 의 고정 문자열 셋 (Step 10). **무엇이 본문인지 아는 것은 사람이다** |
| ② | 짓는 쪽과 읽는 쪽의 일치 | 왕복 시험 (Step 10 `roundtrip_test.go`) |
| ③ | `Fields` 를 `map[string]any` 로 바꾸는 것 | **아무 기계도 안 막는다.** 공개 겉면의 diff 로 사람이 본다 (NFR Design B1) |
| ④ | `enode.capped` 의 글자와 뜻 | 사람이 맞댄다 (6.4 ③) |
| ⑤ | 시험 파일의 임포트 | 사람이 본다 (5.3) |

**그래서 SECURITY-13 을 닫는 것은 퍼즈 하나가 아니라 넷이다** — F1 · F3 ·
왕복 시험 · 누출 시험. **퍼즈만으로 닫았다고 적으면 그것이 거짓 초록이다.**

### 8.5 CI 에 `-fuzz` 를 안 붙인다

```text
   go test ./internal/transcript            시드만 한 번씩 돈다.  **스킵이 아니다**
   go test ... -fuzz=FuzzParse -fuzztime=60s  변이.  **사람이 돌린다**
```

`internal/transcript` 를 고치는 커밋이 아래를 한 번 돌린다. **집행은 사람이다.**

```text
   go test ./internal/transcript -run=XXX -fuzz=FuzzParse -fuzztime=60s
```

**`ci.yml` 은 이 유닛의 파일 행렬 밖이다** — 넣을지는 진행자에게 넘긴다 (12절 ④).

---

## 9. 스토리 추적 — 어느 Step 이 무엇을 닫나

**U1 이 지는 스토리와 완료 조건이 0 이고 기능 하나(FR-3)를 진다**
(`unit-of-work-story-map.md` 4절). **빠뜨린 것이 아니다** — 파서는 사람이 보는
표면이 없고 전부 재료다.

| | 무엇 | 지는 유닛 | U1 이 주는 것 | Step |
|---|---|---|---|---|
| **FR-3** | 공용 파서 | **U1** | 패키지 전부 | 4 ~ 15 |
| FR-3 수용 기준 | 같은 로그를 링에서 읽든 record 에서 읽든 같은 사건 열 | **U1** | 두 경로 시험 | **14** |
| US-4 | 못 읽은 줄을 원문으로 | U5 | `Sub=="plain"` 과 `raw` 의 `Text` 가 줄 원문 (R3.1 · R3.2) | 7 · 10 R3 |
| US-6 | 본문이 없는 이유 | U6 | `Event.Shell` · `Result.Elided` (R4 · R5) | 7 · 10 R4 · R5 |
| US-7 · NC-4 | 상한에 닿아 멈췄다 | U4 | `Kind == capped` 가 **사건 열의 그 자리**에 (R3.6) | 3 · 7 · 10 R3 |
| US-10 | 봉인된 묶음만 열어 안다 | U6 | 껍데기 줄기가 도구 이름 · 성공 여부 · 토큰을 낸다 (3.2) | 6 · 10 |

**US-4 의 확인 글자가 갈린다** — 그 스토리는 확인란이 「`raw` 사건」인데
R3.1 · R3.2 아래서 못 읽은 줄은 **`plain text` 이거나 `raw`** 다. **둘 다 줄
바이트를 그대로 `Text` 에 든다**. 스토리의 값은 서고 글자만 갈린다 (12절 ③).

**화면에 넘기는 값 여섯**(`business-rules.md` 16.2)이 전부 Step 4 의 필드다 —
`Shell` · `Sub` · `Cut` · `Kind == capped` · `Result.Elided` · `Result.Partial`.

---

## 10. 짓지 않는 것

```text
   selectLogs 를 옮기는 것       파서가 아니라 정책이다.  internal/enode 에 남는다
   CappedMarker 같은 짓는 함수   찍는 쪽이 U3 이다.  읽기만 한다 (5.2 절)
   Event 의 JSON 태그           as=events 의 형식은 U4 의 것이다
   enode.Event 를 건드리는 것     U2 의 자리다 (Q9 = A)
   ElidedMarker 의 인자 모양      정수 둘 그대로.  옮기기에 모양 바꾸기를 안 섞는다
   마스킹 · 검색 · 필터 · 요약     constraints.md §4 가 금지했다
   로거 · 캐시 · 타이머 · 시계     논리 컴포넌트 0 (logical-components.md 1절)
   새 Go 의존                   go.mod · go.sum diff 0
   .coverage-contract.yml       주인이 따로다.  낡은 정도만 12절 ⑤ 로 넘긴다
   ci.yml                       파일 행렬 밖이다 (12절 ④)
   회차 문서와 팩                이 유닛이 안 만진다.  12절이 표시만 넘긴다
```

---

## 11. 물음 — 하나

**나머지는 전부 앞 단계의 답이 정했거나 이 계획이 실측으로 닫았다.**
아래 하나만 값이 없고, **짐작하면 `tool_result` 의 본문이 통째로 비는 판이 나온다.**

## Question 1

`tool_result` 블록의 `content` 가 **문자열이 아니라 배열**로 올 때 `Event.Text` 를
무엇으로 채우나. 실측 표 ⑳ 의 픽스처는 문자열 하나뿐이고
(`{"type":"tool_result","is_error":true,"content":"..."}`), 배열 꼴을 이 저장소가
한 번도 기록하지 않았다. Step 2 가 실물로 확인하지만 **못 돌릴 수도 있어** 값을
먼저 정해 둔다.

A) 문자열이면 그대로 쓰고, **그 밖의 모양이면 `Text` 를 비운다.** R2.2 의
「아는 모양이 아니면 없는 것으로 본다」를 글자 그대로 따른다. 대가 — 하네스가
배열로 내는 순간 봉인 전 트랜스크립트의 도구 결과 본문이 **전부 빈다.**
`Shell` 은 거짓이므로 화면이 「걷혔다」가 아니라 「비어 있었다」로 그려 US-6 이
그 자리에서 거짓을 말한다

B) 문자열이면 그대로 쓰고, **배열이면 그 안의 `type=="text"` 블록의 `text` 를
순서대로 잇는다.** 그 밖의 모양은 비운다. 대가 — 아는 모양이 하나 는다.
`content` 의 원소가 `image` 처럼 본문이 아닌 것이면 건너뛰므로 「안 버린다」(R1)가
**블록 수준에서는** 안 서고, 그 사실을 규칙으로 적어야 한다

C) 문자열이 아니면 그 `content` 의 **원문 JSON 바이트**를 `Text` 에 넣고 상한을
건다. 대가 — 화면에 JSON 이 그대로 나온다. 팩이 「JSON 을 그대로 보이지 않는다」로
막은 자리다 (`decisions.md` §1)

X) Other (please describe after [Answer]: tag below)

[Answer]:

**이 물음이 Part 2 를 안 막는다** — Step 2 가 실측으로 답을 주면 그것이 이기고,
못 돌리면 답이 올 때까지 A 로 짓되 그 자리에 규칙 번호를 비워 둔다.

---

## 12. 진행자에게 넘기는 것

앞 단계의 표시가 그대로 산다 (`nfr-requirements.md` 8절의 다섯). **이 계획이
더하는 것은 ② · ⑥ · ⑧ 셋과 ⑦ 의 뒤 절반이고, 나머지는 자리만 다시 가리킨다.**

```text
   ①  팩 requirements/transcript/decisions.md:50 의 「200자 · 500자」를
      「200바이트 · 500바이트」로.  단위를 이 유닛이 정했고 (물음 1 = A)
      팩은 이 유닛이 안 만진다.  회차 requirements.md 5.6 ② 도 같은 단위다

   ②  **새것.**  F2 의 글자를 고친다 — 두 자리다.
        nfr-requirements/nfr-requirements.md 2.2 · 4.2
        functional-design/business-rules.md 8.2
      고칠 문장은 「그 둘의 Text 는 JSON 을 거쳐 오므로 이미 올바르다」이고,
      실측은 **tool_result 만 참이고 tool_use 는 거짓**이다 (7.2).
      고친 글자는 「입력이 올바른 UTF-8 이면」 한 조건이 앞에 붙는 것이다.
      **값은 안 바뀐다** — 상한도 단위도 자르는 규칙도 그대로다

   ③  회차 user-stories.md US-4 의 확인란 「raw 사건」을
      「plain text 또는 raw — 둘 다 줄 원문을 Text 에 든다」로 (9절)

   ④  ci.yml 에 변이 퍼즈를 넣을지.  이 계획의 판단은 「넣지 말 것」이고
      근거는 무한히 도는 스텝이 이 저장소 게이트의 성질과 다르다는 것이다 (8.5)

   ⑤  .coverage-contract.yml 의 packages: 기준선이 15 -> 16 행이 된다.
      게이트는 그 표를 안 읽으므로 빨개지지 않는다.  그 파일은 주인이 따로다

   ⑥  **새것.**  U3 이 병합된 뒤 6.4 ③ 의 한 줄을 돌려 enode.capped 의
      글자와 bytes 의 뜻을 맞댄다.  **U1 은 그 절반을 못 닫는다** —
      착수 순서가 U1 을 먼저 세우고 U3 의 코드가 그때 없다 (6.2)

   ⑦  옮겨 오는 것의 개수 — **고칠 자리가 넷이고 값은 함수 여덟 + 타입 둘이다**
      (1.2 의 표가 정본 · 1.3 ④).  회차 문서 둘은 「셋」이고 유닛 문서 둘은
      「일곱」이라 **틀린 값이 두 벌로 있다**

        application-design/components.md 1절     「셋」      -> 여덟 + 둘
        application-design/unit-of-work.md U1 절  「셋」      -> 여덟 + 둘
        construction/.../domain-entities.md 1절   제목 「일곱」 -> 여덟.
                                                 **그 절의 표는 이미 열 행이다**
        plans/transcript-functional-design-plan.md 6절 ③ ⑦ · 691행
                                                 「일곱 + 타입 둘」 -> 여덟 + 둘

      앞 둘은 FD 계획 6절이 이미 진행자에게 넘긴 자리이고, **뒤 둘이 이 단계가
      새로 찾은 것이다** — 「셋」을 고치면서 쓴 값이 하나 모자랐다

   ⑧  **새것.**  construction/.../business-rules.md 14절의 커버리지 수를
      고친다 — 「문장 65 · 덮임 63」 -> **64 · 62**,
      「옮긴 뒤 1,908 / 2,249」 -> **1,909 / 2,250**.  비율(96.9% · 84.8%)은
      그대로다 (1.3 ① · 8.2).  같은 절의 「옮긴 뒤 그 가지를 어떻게 다룰지는
      Code Generation 이 정한다」는 8.3 이 닫았다
```

**⑦ 과 ⑧ 이 같은 결이다** — 둘 다 **글자로 센 수가 표와 어긋난 자리**이고,
둘 다 값의 방향을 안 바꾼다. 고치는 사람이 같으므로 한 커밋에 묶어도 된다.

**이 유닛이 회차 문서도 팩도 안 고친다** (`CONVENTIONS.md` 3.4 의 「안 싣는 것」).
