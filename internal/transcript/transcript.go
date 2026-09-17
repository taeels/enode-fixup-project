// Package transcript 는 하네스의 stdout 을 화면이 그리는 사건 열로 읽는다.
//
// 방향이 하나다 — 이 패키지는 internal/enode 도 internal/api 도
// internal/store 도 internal/panel 도 임포트하지 않는다. 파서는 자기를 부르는
// 쪽을 모르고, 그래서 파일도 소켓도 시계도 못 만진다. 못 하는 것이 규칙이
// 아니라 의존 그래프다.
//
// 그 금지 넷과 「표준 라이브러리만 쓴다」는 봉인은 여기가 아니라
// internal/panel/boundary_test.go 가 go list -deps 로 잰다. 그 파일에 넣은
// 이유는 그것이 이미 저장소 전체의 경계 검사이기 때문이다 — 검사기를 두
// 벌로 두면 한쪽만 고쳐도 둘 다 초록이라 갈린 것을 아무도 못 본다.
//
// 짓는 쪽과 읽는 쪽이 한 패키지에 산다. Shell 이 "tools" 로 쓰고 Parse 가
// "tools" 로 읽는다. 둘이 다른 패키지면 한쪽만 고쳐도 컴파일이 되고, 틀리는
// 순간은 봉인된 Run 을 화면에서 열 때다 — 몇 주 뒤다. 왕복 시험 하나가 같은
// 패키지 안에서 그것을 잡는다 (roundtrip_test.go).
//
// Kind 는 닫힌 어휘 일곱이다. 하네스가 새 type 을 내도 화면이 안 바뀐다 —
// 새 것은 raw 로 오고, 우리는 그것을 버리지도 않고 자리를 지어 주지도 않는다.
package transcript

import "encoding/json"

// Fields 는 아직 안 푼 JSON 객체다. 아는 키만 그때그때 푼다.
//
// map[string]any 가 아닌 이유 — any 로 받으면 도구 결과 본문(수십 KB 문자열)이
// 파싱 시점에 통째로 메모리로 올라오고, 「아는 키만 읽는다」가 「전부 읽고 아는
// 것만 쓴다」로 바뀐다 (SECURITY-13).
//
// 정의 타입이 아니라 별칭이다. 별칭이면 json.Unmarshal(ln, &obj) 가 그대로
// 돌고 internal/enode 에 남는 코드가 변환 없이 같은 값을 넘긴다. 정의 타입으로
// 두면 경계마다 변환이 생기고, 그 변환이 하는 일이 0 이다.
type Fields = map[string]json.RawMessage

// Kind 는 사건 종류다. 하네스가 내는 여섯과 enode 가 찍는 하나를 합해
// 일곱이고, 이 목록은 닫혀 있다.
//
// 어휘를 닫는 것이 「파서는 하나다」의 값이다. 화면 셋이 이 일곱만 그리면 되고,
// 사상표에 없는 type 은 KindRaw 로 떨어진다.
type Kind string

const (
	// 하네스가 내는 여섯
	KindInit       Kind = "init"
	KindText       Kind = "text"
	KindToolUse    Kind = "tool_use"
	KindToolResult Kind = "tool_result"
	KindResult     Kind = "result"
	KindRaw        Kind = "raw"

	// KindCapped 는 enode 가 찍는 하나다. 진행 파일이 상한에 닿아 멈춘 자리를
	// 사건 열의 그 자리에 세운다 — 위치가 값이라 집계로 뺄 수 없다.
	KindCapped Kind = "capped"
)

// Event 는 화면이 그리는 한 조각이다.
//
// 줄 하나가 사건 여럿이 될 수 있다 — assistant 한 줄에 thinking 과 text 와
// tool_use 가 함께 오면 사건 넷이다. 그 넷은 Line 이 같다.
type Event struct {
	Kind Kind   `json:"kind"`
	Sub  string `json:"sub,omitempty"` // Kind 안의 갈래. 없으면 빈 문자열
	Line int    `json:"line"`          // 입력에서 몇 번째 줄인가. 1 부터. 한 줄에서 난 사건이 같은 값을 든다

	Text string `json:"text,omitempty"` // 본문. Kind 마다 무엇인지는 아래 표
	Name string `json:"name,omitempty"` // 도구 이름. tool_use 와 tool_result 만 채운다
	ID   string `json:"id,omitempty"`   // tool_use_id. 붙이기의 열쇠다. 껍데기에는 없다
	// OK 는 tool_result 의 성공 여부다. nil 은 「없음」이고 false 와 다르다 —
	// 도구를 안 부른 사건에 false 를 박으면 「실패한 도구가 있었다」로 읽힌다.
	//
	// omitempty 가 그 셋째 값을 선 위에서도 지킨다 — 포인터라 nil 이면 키가
	// 통째로 빠지고, 받는 쪽이 "없다" 와 "실패했다" 를 그대로 가른다.
	OK *bool `json:"ok,omitempty"`
	// Cut 은 상한과 룬 경계에 잘려 Text 에 안 실린 바이트 수다. 0 이면
	// 안 잘렸다. 원래 길이가 아니라 잘려 나간 양인 이유는 「안 잘렸다」가
	// Cut == 0 한 비교이기 때문이다.
	Cut int `json:"cut,omitempty"`

	// Shell 이 참이면 이 사건은 selectLogs 가 지은 껍데기 줄에서 왔다.
	// Text 가 빈 것이 「말을 안 했다」가 아니라 「걷혔다」다 — 그 둘을 가르는
	// 것이 US-6 이고 이 필드가 그것을 세운다.
	Shell bool `json:"shell,omitempty"`

	Tokens map[string]int `json:"tokens,omitempty"` // 예산 신호. 키는 in · out · cache_write · cache_read · thinking
	Info   Info           `json:"info,omitzero"`    // init · result · capped 만 채운다. 나머지는 제로값
}

// Text 가 Kind 마다 무엇인가
//
//	init          빈 문자열.  값은 Info 가 든다
//	text          text 또는 thinking 블록의 본문 그대로.  상한이 없다
//	text/plain    그 줄의 바이트 그대로.  상한이 없다
//	tool_use      도구 입력의 JSON.  200 바이트에서 룬 경계로 자른다
//	tool_result   도구 결과 본문.  500 바이트에서 룬 경계로 자른다
//	result        빈 문자열.  값은 Info 가 든다
//	raw           그 줄의 바이트 그대로.  상한이 없다
//	capped        빈 문자열.  값은 Info.Bytes 가 든다

// Info 는 Kind 가 값을 드는 셋이 쓰는 주머니다. 나머지 Kind 에서는 제로값이다.
//
// 파서가 문장을 안 짓는다 — 값만 낸다. 말과 배치는 화면의 것이다. 그 선을
// 여기 둔 이유는 CONVENTIONS 2.3 이다: 문장을 파서가 지으면 그것이 어느
// 언어여야 하는지를 이 패키지가 정하게 되고, 화면 셋이 서로 다른 언어다.
type Info struct {
	// init
	Model   string   `json:"model,omitempty"`   // 모델 이름
	Version string   `json:"version,omitempty"` // 하네스 버전. 줄에 없으면 빈 문자열
	Tools   int      `json:"tools,omitempty"`   // 도구 수. tools 배열의 길이다
	Servers []Server `json:"servers,omitempty"` // MCP 서버. 비어 있으면 길이 0 이다 - 배열이고 맵이 아니다

	// result
	Reason  string  `json:"reason,omitempty"`   // 와이어의 subtype 그대로. 어휘를 우리가 안 바꾼다
	Turns   int     `json:"turns,omitempty"`    // num_turns
	CostUSD float64 `json:"cost_usd,omitempty"` // total_cost_usd

	// Bytes 는 enode.capped 의 bytes 다. 닿은 상한이고 총 길이가 아니다 —
	// 표시 줄도 총 길이에 들어가므로 자기가 든 총 길이를 담을 수 없다.
	// 총 길이는 Progress.Total 이 따로 나른다. int64 인 것은 그 상한이
	// MaxBlobBytes 이고 그 필드가 int64 이기 때문이다.
	//
	// 언제 찍히는지를 이 패키지가 안 적는다 — 찍는 쪽은 internal/record 이고
	// 상한의 값과 조건은 그쪽 문서가 진다. 두 벌로 들면 갈린다.
	Bytes int64 `json:"bytes,omitempty"`
}

// Server 는 init 줄의 mcp_servers 한 칸이다.
//
// 앞 판은 태그를 안 달았다. 근거는 "읽을 때 encoding/json 이 키를 대소문자
// 무시로 맞춘다" 였고 그것은 지금도 참이다 — 다만 그때 이 타입은 읽히기만
// 했다. 이제 GET log 의 as=events 가 이것을 선 위로 내보내고, 쓰는 쪽에는
// 그 관용이 없다: 태그가 없으면 키가 Name 과 Status 로 나가 이 저장소의
// 다른 응답(run_id · step_id)과 모양이 갈린다.
type Server struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// Result 는 Parse 가 한 번 읽은 결과다.
//
// 선 위에서 events 는 언제나 배열이다 - 비어 있으면 [] 이고 null 이 아니다.
// MarshalJSON 이 그것을 지킨다 (아래). 짝 팩의 store.Verdict 가 checks 에서
// 같은 자리를 밟았고 같은 방법으로 닫았다 - 타입이 자기 선 위 모양을 진다.
//
// 왜 타입이 지는가: 이 값이 선 위로 나가는 길이 둘이고 (GET log 의 as=events ·
// 제어판의 /api/transcript 봉투) 둘 다 빈 입력을 정상으로 다룬다. 아직 아무도
// 아무 말도 안 한 단계는 흔한 상태이지 예외가 아니다.
//
// 빠뜨리면 무엇이 깨지나 - 화면이 이 자리에 배열을 요구하고, 사건 배열을
// 검사하는 쪽이 null 을 계약 위반으로 거절한다. 실제로 그렇게 깨졌다:
// 아직 안 시작한 단계의 카드가 "events 가 배열이 아니다" 로 섰다.
type Result struct {
	Events []Event `json:"events"`
	Elided *Elided `json:"elided,omitempty"` // enode.elided 줄이 있었으면. 없으면 nil

	Raw int `json:"raw"` // Kind 가 raw 인 사건의 수. Events 안에도 있다 - 세는 값이다
	// Lines 는 읽은 줄 수다. 버린 머리와 안 읽은 꼬리는 안 센다.
	Lines int `json:"lines"`
	Head  int `json:"head"` // 잘린 머리로 안 읽은 바이트 수. truncated 가 거짓이면 0
	// Partial 은 개행 없이 끝나 안 읽은 꼬리의 바이트 수다. 0 이면 없다.
	//
	// 「버렸다」가 아니라 「안 읽었다」인 이유 - 링과 진행 파일은 쓰는 중에
	// 읽힌다. 그 바이트는 다음 폴링에서 개행이 붙어 완전한 줄이 된다.
	// 읽어서 plain 으로 그렸다가 1초 뒤 사건으로 바꾸면 화면이 깜빡이고,
	// 그 깜빡임이 「멈춘 것인지 도는 것인지」의 오독과 같은 자리에서 난다.
	Partial int `json:"partial"`
}

// MarshalJSON 은 events 를 언제나 배열로 낸다.
//
// nil 슬라이스가 null 로 마샬되는 것이 Go 의 기본이고, 그 기본이 이 자리에서는
// 틀린 값이다. 별칭 타입으로 재귀를 끊는다 - 그 줄이 없으면 이 메서드가
// 자기를 다시 부른다.
func (r Result) MarshalJSON() ([]byte, error) {
	if r.Events == nil {
		r.Events = []Event{}
	}
	type wire Result
	return json.Marshal(wire(r))
}

// Elided 는 걷힌 양이다. 짓는 쪽(ElidedMarker)과 읽는 쪽(Parse)이 같은 값을
// 본다.
//
// 태그를 단다 — Server 와 같은 이유다. 선 위로 나가는 순간 대소문자 관용이
// 없어진다.
type Elided struct {
	Events int `json:"events"`
	Bytes  int `json:"bytes"`
}
