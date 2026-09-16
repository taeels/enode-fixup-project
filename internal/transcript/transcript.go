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
	Kind Kind
	Sub  string // Kind 안의 갈래. 없으면 빈 문자열
	Line int    // 입력에서 몇 번째 줄인가. 1 부터. 한 줄에서 난 사건이 같은 값을 든다

	Text string // 본문. Kind 마다 무엇인지는 아래 표
	Name string // 도구 이름. tool_use 와 tool_result 만 채운다
	ID   string // tool_use_id. 붙이기의 열쇠다. 껍데기에는 없다
	// OK 는 tool_result 의 성공 여부다. nil 은 「없음」이고 false 와 다르다 —
	// 도구를 안 부른 사건에 false 를 박으면 「실패한 도구가 있었다」로 읽힌다.
	OK *bool
	// Cut 은 상한과 룬 경계에 잘려 Text 에 안 실린 바이트 수다. 0 이면
	// 안 잘렸다. 원래 길이가 아니라 잘려 나간 양인 이유는 「안 잘렸다」가
	// Cut == 0 한 비교이기 때문이다.
	Cut int

	// Shell 이 참이면 이 사건은 selectLogs 가 지은 껍데기 줄에서 왔다.
	// Text 가 빈 것이 「말을 안 했다」가 아니라 「걷혔다」다 — 그 둘을 가르는
	// 것이 US-6 이고 이 필드가 그것을 세운다.
	Shell bool

	Tokens map[string]int // 예산 신호. 키는 in · out · cache_write · cache_read · thinking
	Info   Info           // init · result · capped 만 채운다. 나머지는 제로값
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
	Model   string   // 모델 이름
	Version string   // 하네스 버전. 줄에 없으면 빈 문자열
	Tools   int      // 도구 수. tools 배열의 길이다
	Servers []Server // MCP 서버. 비어 있으면 길이 0 이다 - 배열이고 맵이 아니다

	// result
	Reason  string  // 와이어의 subtype 그대로. 어휘를 우리가 안 바꾼다
	Turns   int     // num_turns
	CostUSD float64 // total_cost_usd

	// Bytes 는 enode.capped 의 bytes 다. 닿은 상한이고 총 길이가 아니다 —
	// 표시 줄도 총 길이에 들어가므로 자기가 든 총 길이를 담을 수 없다.
	// 총 길이는 Progress.Total 이 따로 나른다. int64 인 것은 그 상한이
	// MaxBlobBytes 이고 그 필드가 int64 이기 때문이다.
	//
	// 언제 찍히는지를 이 패키지가 안 적는다 — 찍는 쪽은 internal/record 이고
	// 상한의 값과 조건은 그쪽 문서가 진다. 두 벌로 들면 갈린다.
	Bytes int64
}

// Server 는 init 줄의 mcp_servers 한 칸이다.
//
// JSON 태그를 안 단다 — encoding/json 이 키를 대소문자 무시로 맞추므로
// {"name":...,"status":...} 가 태그 없이 그대로 찬다.
type Server struct{ Name, Status string }

// Result 는 Parse 가 한 번 읽은 결과다.
type Result struct {
	Events []Event
	Elided *Elided // enode.elided 줄이 있었으면. 없으면 nil

	Raw int // Kind 가 raw 인 사건의 수. Events 안에도 있다 - 세는 값이다
	// Lines 는 읽은 줄 수다. 버린 머리와 안 읽은 꼬리는 안 센다.
	Lines int
	Head  int // 잘린 머리로 안 읽은 바이트 수. truncated 가 거짓이면 0
	// Partial 은 개행 없이 끝나 안 읽은 꼬리의 바이트 수다. 0 이면 없다.
	//
	// 「버렸다」가 아니라 「안 읽었다」인 이유 - 링과 진행 파일은 쓰는 중에
	// 읽힌다. 그 바이트는 다음 폴링에서 개행이 붙어 완전한 줄이 된다.
	// 읽어서 plain 으로 그렸다가 1초 뒤 사건으로 바꾸면 화면이 깜빡이고,
	// 그 깜빡임이 「멈춘 것인지 도는 것인지」의 오독과 같은 자리에서 난다.
	Partial int
}

// Elided 는 걷힌 양이다. 짓는 쪽(ElidedMarker)과 읽는 쪽(Parse)이 같은 값을
// 본다.
//
// JSON 태그를 안 단다 — Server 와 같은 이유다.
type Elided struct{ Events, Bytes int }
