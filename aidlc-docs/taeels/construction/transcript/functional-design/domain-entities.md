# U1 `transcript` — 이 유닛이 드는 형식

**겉면은 `component-methods.md` 1절이 적었고 필드는 여기서 닫는다.** 그 문서가
`Event struct{ Kind Kind /* ... */ }` 로 자리표를 남기고 이 단계에 넘겼다.

물음 열둘의 답은 전부 **A** 다 (2026-09-15). 답이 정한 자리마다 질문 번호를 단다.

---

## 1. 옮겨 오는 범위 — 함수 일곱과 타입 둘

`components.md` 1절과 `unit-of-work.md` U1 절이 **셋**으로 적었다. 실측이 아홉을
찾았다 (계획 1.1). **셋만 옮기면 컴파일이 안 된다** — 나머지가 그 셋의 몸통이다.

| 오늘 자리 | 옮긴 뒤 이름 | 공개 | 왜 옮기나 |
|---|---|---|---|
| `runner.go:386` `parseEventLine` | `ParseLine` | 공개 | 겉면 문서가 이름을 정했다 |
| `runner.go:434` `eventShell` | `Shell` | 공개 | 같다 |
| `runner.go:425` `elidedMarker` | `ElidedMarker` | 공개 | 같다 |
| `runner.go:370` `splitLines` | `SplitLines` | 공개 | `Parse` 와 `selectLogs` 가 둘 다 쓴다 (Q8) |
| `runner.go:513` `eventString` | `String` | 공개 | `selectLogs` 가 `subtype` 을 읽는다 (Q8) |
| `runner.go:521` `eventBool` | `Bool` | 비공개로 둘 수 없다 | 셋이 한 벌이다. 하나만 공개하면 다음 사람이 나머지를 다시 짓는다 |
| `runner.go:529` `eventInt` | `Int` | 공개 | 같다 |
| `runner.go:487` `usageTokens` | `usageTokens` | **비공개** | `Shell` 과 `Parse` 만 부른다. 밖에서 부를 일이 없다 |
| `runner.go:407` `logShell` (타입) | `logShell` | **비공개** | 와이어 형식이고 `Shell` 이 마샬한다. 밖은 바이트만 본다 |
| `runner.go:419` `elidedMark` (타입) | `elidedMark` | **비공개** | 같다. 밖이 보는 것은 `Elided` 다 (5절) |

**`selectLogs` 는 안 옮긴다.** `internal/enode` 에 남는다 — 이유는
`business-logic-model.md` 7절이 진다.

**셋이 아니라 일곱인 것을 문서가 안 적으면 다음 사람이 컴파일 오류로 안다.**
`components.md` 1절과 `unit-of-work.md` U1 절이 고쳐야 할 자리다
(`plans/transcript-functional-design-plan.md` 6절).

---

## 2. `Fields` — 맵의 이름 (Q7 = A)

```go
// Fields 는 아직 안 푼 JSON 객체다. 아는 키만 그때그때 푼다.
//
// map[string]any 가 아닌 이유 — any 로 받으면 도구 결과 본문(수십 KB 문자열)이
// 파싱 시점에 통째로 메모리로 올라오고, 「아는 키만 읽는다」가
// 「전부 읽고 아는 것만 쓴다」로 바뀐다 (SECURITY-13).
type Fields = map[string]json.RawMessage
```

**정의 타입이 아니라 별칭이다.** 별칭이면 `json.Unmarshal(ln, &obj)` 가 그대로
돌고 `internal/enode` 에 남는 코드가 변환 없이 같은 값을 넘긴다. 정의 타입으로
두면 경계마다 변환이 생기고, 그 변환이 하는 일이 0 이다.

`component-methods.md` 1.2 가 `map[string]any` 로 적은 자리다 —
**그 문서를 고쳐야 한다** (계획 6절).

---

## 3. `Kind` — 일곱. 닫힌 어휘다 (Q2 = A)

```go
// Kind 는 사건 종류다. 하네스가 내는 여섯(decisions.md 2절)과
// enode 가 찍는 하나(3.3)를 합해 일곱이고, 이 목록은 닫혀 있다.
type Kind string

const (
	// 하네스가 내는 여섯
	KindInit       Kind = "init"
	KindText       Kind = "text"
	KindToolUse    Kind = "tool_use"
	KindToolResult Kind = "tool_result"
	KindResult     Kind = "result"
	KindRaw        Kind = "raw"

	// enode 가 찍는 하나 (3.3)
	KindCapped Kind = "capped"
)
```

**어휘를 닫는 것이 「파서는 하나다」의 값이다.** 화면 셋이 이 일곱만 그리면
되고, 하네스가 새 `type` 을 내도 화면이 안 바뀐다 — 새 것은 `raw` 로 온다.

**일곱째는 U3 의 답이 들여온 것이다** (2026-09-15). `Q2 = A` 는 여섯을 그대로
두었고 U1 의 답은 한 글자도 안 바뀌었다 — 경위는
`plans/transcript-functional-design-plan.md` 4.1.2.

### 3.1 `raw` 의 뜻이 넓어졌다

팩은 `raw` 를 「파싱 실패 줄의 원문」으로 적었다. **Q2 = A 가 그것을
「Kind 로 사상되지 않은 줄」로 넓혔고, Q3 = A 가 다시 좁혔다.**

```text
   raw 다      JSON 객체로 풀리고 type 이 문자열인데 사상표에 없는 줄.
               실측으로는 rate_limit_event · system/hook_started ·
               system/hook_response · system/thinking_tokens 넷이다.
               우리가 모르는 enode.* 도 여기로 온다 (3.3 · R3.7)

   raw 가 아니다  JSON 이 아닌 줄.  그것은 text 이고 Sub 가 "plain" 이다 (Q3 = A)
```

**두 답이 서로를 부정하지 않는다** — Q3 의 선택지 A 가 그 대가를 「`raw` 가
「JSON 인데 Kind 를 모르는 줄」만 지게 되는 것」으로 미리 적었다. 판정 사다리는
`business-logic-model.md` 2절이 든다.

### 3.2 사상표 — 와이어에서 Kind 로

실측은 `requirements/harness-components/decisions.md` 6절 ⑳ 이다
(2026-09-12 · `claude 2.1.266`). 이 유닛이 다시 재지 않는다 — 두 벌로 두지 않는다.

| 와이어 | Kind | `Sub` |
|---|---|---|
| `system` / `init` | `init` | `""` |
| `assistant` 의 `text` 블록 | `text` | `""` |
| `assistant` 의 `thinking` 블록 | `text` | `"thinking"` |
| `assistant` 의 `tool_use` 블록 | `tool_use` | `""` |
| `user` 의 `tool_result` 블록 | `tool_result` | `""` |
| `result` | `result` | 와이어의 `subtype` |
| `system` / `hook_started` · `hook_response` · `thinking_tokens` | `raw` | `"system/<subtype>"` |
| `rate_limit_event` | `raw` | `"rate_limit_event"` |
| 그 밖의 `type` (문자열) | `raw` | 그 `type` |
| JSON 이 아닌 줄 | `text` | `"plain"` |
| `enode.elided` | **사건이 아니다** | `Result.Elided` 로 간다 (5절) |
| `enode.capped` | `capped` | `""`. 값은 `Info.Bytes` — **닿은 상한**이다 (4.4) |
| 그 밖의 `enode.*` | `raw` | 그 `type`. 아는 둘에만 자리를 준다 (R3.7) |

**`thinking` 이 `text` 로 가되 `Sub` 로 갈린다.** 같은 Kind 로 두는 이유는
어휘가 닫혀 있기 때문이고, 갈라 적는 이유는 **화면이 생각을 기본으로 접을 수
있어야 하기 때문**이다 — 실측 픽스처의 `thinking` 블록에 자격증명 문자열이
들어 있었다 (`logs_test.go:27`). 파서는 마스킹을 안 하므로 (`constraints.md` §4)
접는 일은 화면의 몫이고, **접을 근거를 파서가 값으로 준다.**

### 3.3 점이 있는 `type` 은 enode 의 것이다

하네스가 내는 `type` 은 전부 홑단어다 — `system` · `assistant` · `user` ·
`rate_limit_event` · `result` (실측 ⑳). **그래서 점을 넣은 `type` 은 우리가
찍었다는 표지다.** 근거는 `runner.go:415-418` 의 주석이 `enode.elided` 에 대해
이미 적어 둔 그것이고, `enode.capped` 에 같은 근거가 선다.

```text
   enode.elided   selectLogs 가 찍는다.  「이 파일에서 N 개 · B 바이트가 걷혔다」
   enode.capped   U3 의 진행 파일 쪽이 찍는다.  「여기서 상한에 닿아 멈췄다」.
                  bytes 는 **닿은 상한**이다 — 총 길이가 아니다 (4.4)
```

**갈래는 하나이고 목적지가 둘이다. 가르는 기준은 위치가 값인가다.**

| 줄 | 목적지 | 왜 |
|---|---|---|
| `enode.elided` | `Result.Elided` | **파일 전체의 집계다.** 어느 줄에 있든 뜻이 같다 — `selectLogs` 는 stderr 앞에 한 번 쓴다 |
| `enode.capped` | **`Events` 의 한 자리** (`capped`) | **위치가 값이다.** 「그 사건 다음부터 없다」가 US-7 의 전부이고, 집계로 빼면 사건 열 어디서 끊겼는지를 잃는다 |

**US-7 이 이 갈림을 정했다** — 「진행 로그가 상한에 닿아 멈췄을 때 그것을 알고
싶다. 모르면 단계가 멈춘 줄 안다」. 사람이 그것을 아는 자리는 **마지막 사건
다음**이다. `Result` 의 필드로만 두면 화면이 그 줄을 카드 머리에 그리게 되고,
그러면 「멈춘 것」과 「끊긴 것」이 다시 섞인다.

**응답 헤더로도 온다** — `X-Enode-Log-Capped` (`component-methods.md` 4.1).
헤더는 **지금 상태**이고 이 사건은 **그 위치**다. 둘이 겹치는 것이 아니라
다른 사실이고, 봉인 뒤 tar 에서 읽을 때는 헤더가 없다.

**언제 찍히는지는 이 문서가 안 적는다.** 상한의 값과 찍는 조건은
`internal/record` 의 것이고 **U3 의 FD 가 진다** — 두 문서가 같은 값을 두 벌로
들면 갈린다 (R11.1 과 같은 결).

---

## 4. `Event` — 사건 하나

```go
// Event 는 화면이 그리는 한 조각이다.
//
// 줄 하나가 사건 여럿이 될 수 있다 (Q1 = A) — assistant 한 줄에 thinking 과
// text 와 tool_use 가 함께 오면 사건 넷이다.
type Event struct {
	Kind Kind
	Sub  string // Kind 안의 갈래. 3.2 의 표가 값을 정한다. 없으면 빈 문자열
	Line int    // 입력에서 몇 번째 줄인가. 1 부터. 한 줄에서 난 사건이 같은 값을 든다

	Text string // 본문. Kind 마다 무엇인지는 4.1 의 표
	Name string // 도구 이름. tool_use 와 tool_result 만 채운다
	ID   string // tool_use_id. 붙이기의 열쇠 (business-logic-model 5절). 껍데기에는 없다
	OK   *bool  // tool_result 의 성공 여부. nil 은 「없음」이고 false 와 다르다
	Cut  int    // 상한과 룬 경계에 잘려 Text 에 안 실린 바이트 수. 0 이면 안 잘렸다

	Shell bool // 껍데기 줄에서 왔다 — 본문이 걷혔다 (Q4 = A)

	Tokens map[string]int // 예산 신호. 키는 in · out · cache_write · cache_read · thinking
	Info   Info           // init · result · capped 만 채운다. 나머지는 제로값
}
```

### 4.1 `Text` 가 Kind 마다 무엇인가

| Kind | `Text` | 상한 |
|---|---|---|
| `init` | 빈 문자열 | — (값은 `Info` 가 든다) |
| `text` (`Sub == ""`) | `text` 블록의 본문 그대로 | 없다 (`decisions.md` 2절 — 「`text` 는 그대로」) |
| `text` (`Sub == "thinking"`) | `thinking` 블록의 본문 그대로 | 없다 |
| `text` (`Sub == "plain"`) | 그 줄의 바이트 그대로 | 없다 |
| `tool_use` | 도구 입력을 JSON 으로 다시 적은 요약 | **200 바이트** (룬 경계) |
| `tool_result` | 도구 결과 본문 | **500 바이트** (룬 경계) |
| `result` | 빈 문자열 | — (값은 `Info` 가 든다) |
| `raw` | 그 줄의 바이트 그대로 | 없다 |
| `capped` | 빈 문자열 | — (값은 `Info.Bytes` = 닿은 상한 · 4.4) |

**숫자 둘은 팩의 값이고 단위는 이 유닛이 정했다** — 팩은 「200자 · 500자」이고
여기는 바이트다. 그 갈림과 근거는 `business-rules.md` 8.1 이 진다.
**「룬 경계」는 상한을 넘지 않는 가장 긴 접두 중 마지막 룬이 온전한 데까지라는
뜻이다** (`business-rules.md` 8.2 · R8.1).

**`Cut` 은 잘려 나간 바이트 수이지 원래 길이가 아니다.** 원래 길이는
`len(Text) + Cut` 이다. 그렇게 두는 이유는 「안 잘렸다」가 `Cut == 0` 한
비교이기 때문이다 — 원래 길이로 두면 `Cut == len(Text)` 를 매번 비교해야 한다.

### 4.2 `Shell` — 본문이 걷혔다

```text
   Shell 이 참이면   이 사건은 selectLogs 가 지은 껍데기 줄에서 왔다.
                    Text 가 비어 있는 것은 「말을 안 했다」가 아니라 「걷혔다」다.
                    US-6 이 가르라고 한 두 사실이 이것이다

   Shell 이 거짓이면  원문 줄에서 왔다.  Text 가 비어 있으면 실제로 비어 있었다
```

**판정은 `assistant` 와 `user` 에만 건다.** 그 둘만 `Shell()` 이 본문을 걷는
대상이고, 그 둘의 원문에는 `message` 키가 언제나 있다. 규칙은
`business-rules.md` R4 가 든다.

### 4.3 `Info` — `init` · `result` · `capped` 가 드는 값

```go
// Info 는 Kind 가 값을 드는 셋이 쓰는 주머니다. 나머지 Kind 에서는 제로값이다.
//
// 파서가 문장을 안 짓는다 — 값만 낸다. 말과 배치는 화면의 것이다.
// 그 선을 여기 둔 이유는 CONVENTIONS 2.3 이다: 문장을 파서가 지으면 그것이
// 어느 언어여야 하는지를 이 패키지가 정하게 되고, 화면 셋이 서로 다른 언어다.
type Info struct {
	// init
	Model   string   // 모델 이름
	Version string   // 하네스 버전. 줄에 없으면 빈 문자열
	Tools   int      // 도구 수. tools 배열의 길이다
	Servers []Server // MCP 서버. 비어 있으면 길이 0 이다 (실측 — 배열이고 맵이 아니다)

	// result
	Reason  string  // 와이어의 subtype 그대로. 어휘를 우리가 안 바꾼다
	Turns   int     // num_turns
	CostUSD float64 // total_cost_usd

	// capped
	Bytes int64 // enode.capped 의 bytes. 닿은 상한이다 — 총 길이가 아니다 (4.4)
}

// Server 는 init 줄의 mcp_servers 한 칸이다.
type Server struct{ Name, Status string }
```

**`Servers` 가 배열인 것이 실측이다** (⑳ — `[{"name":…,"status":…}]`).
비어 있음은 `[]` 이고 `nil` 과 같이 다룬다 — 길이 0 하나로 본다.

**`Reason` 을 `HarnessResult.Reason` 어휘로 안 옮긴다.** 그 번역은
`ParseClaude` 가 하고 (`harness.go:153-162`) 그 자리는 이 유닛 밖이다
(계획 1.9). 두 벌로 두면 갈린다.

### 4.4 `Info.Bytes` 는 **닿은 상한**이지 총 길이가 아니다

**이 값의 정본은 U3 이다** — 찍는 쪽이 `internal/record` 다. U3 의 `R12` 가 줄의
모양을, `R11` 과 `R13` 이 그 값이 총 길이일 수 없는 이유를 진다. **여기서는
가리키기만 하고 그 규칙을 베껴 적지 않는다** (R3.8 · R11.1 과 같은 결).

**총 길이로 읽으면 안 되는 이유 셋.**

```text
   ①  자기참조다        U3 의 R13 이 「표시 줄도 총 길이에 든다」로 못 박았다.
                       그러면 표시 줄은 자기가 든 총 길이를 담을 수 없다 —
                       쓰는 순간 그 값이 바뀐다.  「닿은 시점의 총 길이」는
                       파일을 읽는 쪽이 보는 총 길이와 언제나 다르다

   ②  겹친다            총 길이는 Progress.Total 이 이미 따로 나른다
                       (component-methods.md 2.1 · 응답 헤더 X-Enode-Log-Bytes).
                       표시 줄이 그것을 또 실으면 두 벌이 되고 갈릴 자리가 생긴다.
                       상한은 표시 줄에만 있는 값이라 안 겹친다

   ③  집 안의 선례       같은 파일의 기존 잘림 표시가
                       "... log truncated at %d bytes" 이고 그 %d 가 limit 이다
                       (record.go:70).  U3 도 그것을 근거로 댔다
```

**`int64` 인 것은 그 상한이 `MaxBlobBytes` 이고 그 필드가 `int64` 이기
때문이다** (`config.go:51` · 기본값 `10 << 20` 이 `config.go:100`).
파일 총 길이라서가 **아니다**.

**`elidedMark.Bytes` 가 `int` 인 것과 다른 이유도 그것이다** — 거기 `bytes` 는
**한 파일 안에서 걷힌 양**이고 이쪽은 **멈춘 지점의 값**이다. 둘이 같은 이름을
쓰되 같은 종류의 수가 아니다.

---

## 5. `Result` · `Elided` — 한 번 읽은 결과

```go
// Result 는 Parse 가 한 번 읽은 결과다.
type Result struct {
	Events []Event
	Elided *Elided // enode.elided 줄이 있었으면. 없으면 nil

	Raw     int // Kind 가 raw 인 사건의 수. Events 안에도 있다 — 세는 값이다 (Q2 = A)
	Lines   int // 읽은 줄 수. 버린 꼬리와 버린 머리는 안 센다
	Head    int // 잘린 머리로 안 읽은 바이트 수. truncated 가 거짓이면 0
	Partial int // 개행 없이 끝나 안 읽은 꼬리의 바이트 수. 0 이면 없다 (Q6 = A)
}

// Elided 는 걷힌 양이다. 짓는 쪽(ElidedMarker)과 읽는 쪽이 같은 값을 본다.
type Elided struct{ Events, Bytes int }
```

**`Raw` 가 따로 담는 통이 아니라 세는 값인 것을 못 박는다.** 겉면 문서가
「`Raw int // raw 로 넘어간 줄 수`」로만 적어 두 가지로 읽혔다. Q2 = A 가
「`Events` 안에도 같은 것이 있다」로 닫았다.

**`Partial` 이 「버렸다」가 아니라 「안 읽었다」인 이유** — 그 바이트는 다음
폴링에서 개행이 붙어 완전한 줄이 된다. 잃은 것이 아니라 아직 안 온 것이다
(Q6 = A). 규칙은 `business-rules.md` R7.

### 5.1 `elidedMark` 가 함께 옮겨 오는 것이 이 유닛의 값 하나다

```go
// elidedMark 는 표시 줄의 와이어 형식이다. 비공개다 — 밖이 보는 것은 Elided 다.
type elidedMark struct {
	Type   string `json:"type"`   // 언제나 "enode.elided"
	Events int    `json:"events"`
	Bytes  int    `json:"bytes"`
}
```

**짓는 쪽(`ElidedMarker`)과 읽는 쪽(`Parse`)이 같은 구조체를 본다.**
오늘은 짓는 쪽만 있어 갈릴 수 없었을 뿐이고, 읽는 쪽이 생기는 순간 키 이름이
두 자리에 적힐 뻔했다. `type` 에 점을 넣는 근거(`runner.go:415-418` 의 주석 —
실측한 하네스의 `type` 은 전부 홑단어라 부딪칠 수 없다)도 함께 옮긴다.

### 5.2 `enode.capped` 는 읽기만 한다

```text
   찍는 쪽    internal/record.  U3 progress-store 의 것이다
   읽는 쪽    이 패키지.  Parse 가 capped 사건으로 낸다 (3.3)
   짓는 함수  **이 패키지에 없다.**  CappedMarker 같은 것을 안 만든다
```

**`enode.elided` 와 비대칭인 것이 옳다.** `elided` 는 짓는 쪽
(`selectLogs`)이 이 패키지의 함수를 부르므로 형식이 여기 살아야 한다.
`capped` 는 짓는 쪽이 `internal/record` 이고 그 패키지는 **이 패키지를 임포트할
이유가 없다** — 진행 파일을 쓰는 코드가 파서를 딛으면 의존이 거꾸로 하나 는다.

그래서 이 패키지는 `{"type":"enode.capped","bytes":<상한>}` 을 **읽는 쪽에만**
적는다. 그 이름의 정본은 **U3 의 산출물**이고, 갈리면 왕복 시험이
없으므로 **Code Generation 이 두 문서의 글자를 맞대어 확인한다**
(`business-rules.md` 16.1).

---

## 6. `logShell` — 껍데기 줄의 형식 (짓는 쪽)

```go
// logShell 은 사건 하나가 logs/ 에 남기는 전부다. 비공개다.
type logShell struct {
	Type    string         `json:"type"`
	Subtype string         `json:"subtype,omitempty"`
	Tools   []string       `json:"tools,omitempty"`
	OK      *bool          `json:"ok,omitempty"`
	Tokens  map[string]int `json:"tokens,omitempty"`
}
```

**필드 다섯이 오늘 그대로다.** 이 유닛은 껍데기의 **형식을 안 바꾼다** — 옮기기만
한다. 바꾸면 이미 봉인된 Run 의 `logs/` 를 못 읽는다.

### 6.1 껍데기가 읽힐 때 무엇이 되나

| `logShell` 필드 | 어떤 Event 가 되나 |
|---|---|
| `Type` + `Subtype` | Kind 와 `Sub` (3.2 의 표와 같은 사상) |
| `Tools` 한 칸 | `tool_use` 하나. `Name` 은 채우고 `Text` 는 비었고 `Shell` 이 참 |
| `OK` | `tool_result` 하나. `OK` 를 채우고 `Text` 는 비었고 `Shell` 이 참 |
| `Tokens` | 그 줄이 낸 **첫** 사건의 `Tokens` 에 싣는다 (R6) |
| 블록이 하나도 없다 | 사건 **하나**를 낸다 — `text` · `Text` 빈 문자열 · `Shell` 참 (R5) |

**`{"type":"assistant"}` 가 사건 0 이 되면 US-6 이 못 선다.** 「에이전트가 아무
말도 안 했다」와 「걷혔다」를 가르는 것이 그 스토리이고, 사건이 0 이면 화면에
아무것도 안 나와 둘이 같아 보인다.

**`ID` 가 껍데기에 없다.** `logShell` 에 `tool_use_id` 자리가 없어서다. 붙이기가
껍데기 경로에서 어떻게 도는지는 `business-logic-model.md` 5절이 든다.

---

## 7. 이 패키지의 공개 표면 — 전부

```go
// 읽는 쪽
func Parse(b []byte, truncated bool) Result

// 짓는 쪽 (internal/enode 의 selectLogs 가 쓴다)
func ParseLine(line []byte) (Fields, string, bool)
func Shell(obj Fields, typ string) []byte
func ElidedMarker(events, bytes int) []byte

// 줄과 키를 읽는 도우미 (Q8 = A — 두 벌로 두지 않는다)
func SplitLines(b []byte) [][]byte
func String(obj Fields, key string) string
func Bool(obj Fields, key string) (bool, bool)
func Int(obj Fields, key string) (int, bool)

// 형식
type Fields = map[string]json.RawMessage
type Kind string
type Event struct{ ... }
type Info struct{ ... }
type Server struct{ Name, Status string }
type Result struct{ ... }
type Elided struct{ Events, Bytes int }
```

**임포트는 표준 라이브러리만이다** — `encoding/json` · `bytes` ·
`strconv` 정도다. 금지 넷은 `business-rules.md` R9 가 든다.

`ElidedMarker` 의 인자를 `Elided` 하나로 안 바꾼다. 오늘 호출자
(`runner.go:361`)가 정수 둘을 손에 들고 있고, 구조체로 바꾸면 이 유닛이
`selectLogs` 의 줄을 하나 더 고친다 — 옮기는 일에 모양 바꾸기를 섞지 않는다.

### 7.1 `transcript.Kind` 와 `enode.EventKind` 는 안 섞는다 (Q9 = A)

`internal/enode` 에 이미 같은 이름의 것이 있다.

```go
// internal/enode/harness.go:285 — 오늘 그대로 둔다
type EventKind string
const EventFinal EventKind = "final"
type Event struct{ Kind EventKind; Text string }
```

**둘은 다른 물건이다.**

```text
   enode.Event         Job.Emit 으로 흐르는 데몬의 신호다.  claim.go:793 이
                       log.Debug 로 받아 버린다.  사람의 화면에 안 간다
   transcript.Event    화면 셋이 그리는 모형이다.  Parse 가 낸다
```

**이 유닛은 `enode.Event` 를 한 글자도 안 건드린다.** `Decode` 가 줄 단위로
배출할 때 무엇을 `emit` 할지는 **U2 의 자리**이고, 그 시그니처는 정본
`agent-runtime` R3 이 든다.

**이음매만 이름으로 적는다** — U2 가 `enode.EventKind` 를 늘리면 저장소에
사건 어휘가 둘이 된다. 그것이 옳은지는 U2 의 FD 가 재고, **이 유닛은 그 판단의
재료로 `Kind` 일곱이 닫힌 어휘임을 값으로 준다** (3절).

---

## 8. 이 문서가 안 드는 것

```text
   카드가 그리는 모양과 접기        U5 · U6 · U8 의 FD
   as=events 의 JSON 표현          U4 의 FD.  Event 의 JSON 태그를 여기서 안 정한다 —
                                   응답의 형식은 라우트의 것이다
   Decode 가 배출하는 사건의 모양    U2 의 FD.  enode.Event 와 안 섞는다 (Q9 = A)
   시험 목록과 픽스처 파일 이름       이 유닛의 Code Generation
```
