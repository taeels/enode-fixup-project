package enode

import (
	"context"
	"encoding/json"
	"io"
	"strings"
)

// Reason 은 ★ 정규화된 하네스 종료 사유 ★ 다 (ADR-020 결정 2).
//
// ★ ADR-013 결정 3 의 부분 정정 ★
//
//	여전히 맞다   "프로세스 ★ 종료코드 ★ 는 계약이 될 수 없다"
//	              claude 는 헛소리를 하고도 0 으로 끝난다
//	정정          하네스의 ★ 구조화된 결과 봉투 ★ 는 계약이 될 수 있다
//	              error_max_turns 로는 끝나지 ★ 않는다 ★
//
// 하네스마다 봉투가 다르므로 ★ 어댑터가 번역한다 ★ (ADR-013 결정 2 —
// 선언적 매핑은 기각했다. OpenHands 는 플래그가 아니라 Python API 라 설정으로 안 덮인다).
type Reason string

const (
	ReasonOK        Reason = "ok"
	ReasonMaxTurns  Reason = "max_turns"
	ReasonMaxTokens Reason = "max_tokens"
	ReasonError     Reason = "harness_error"
	ReasonTimeout   Reason = "timeout"
)

// Completed 는 ★ 완주했는지 ★ 다. 성공했는지가 아니다.
//
//	harness_error · timeout   ★ 완주가 아니다 ★ — 크래시한 하네스는
//	                          반쯤 쓴 파일을 남길 수 있어 산출물을 믿을 수 없다
//	max_turns · max_tokens    ★ 완주다 ★ — produced 가 판정한다.
//	                          필요한 걸 다 내고 상한에 닿았으면 그건 성공이다.
//	                          다만 Record 에 반드시 남긴다 (예산 신호, ADR-013)
func (r Reason) Completed() bool {
	return r == ReasonOK || r == ReasonMaxTurns || r == ReasonMaxTokens
}

// HarnessResult 는 Record 에 남고 계약 판정에는 안 들어간다.
type HarnessResult struct {
	Reason  Reason  `json:"reason"`
	Turns   int     `json:"turns,omitempty"`
	CostUSD float64 `json:"cost_usd,omitempty"`
	Message string  `json:"message,omitempty"`

	// Session 은 ★ 대화를 이어붙일 손잡이 ★ 다 (R4).
	//
	// 되묻기를 「살아있는 stdin 파이프」로 처리하면 사람이 답할 때까지
	// ★ 임대가 묶인다 ★ — 노드 하나가 사람을 기다리며 논다. ADR-014 가 pull 인 것과도
	// 어긋난다(Mediator 는 enode 를 부를 수 없다).
	//
	// 그래서 ★ 되묻기는 단계를 끊는다 ★ — 프로세스를 끝내고 이 값을 기록에 남긴 뒤,
	// 사람 답이 오면 --resume 으로 ★ 새 단계 ★ 를 연다. Run 은 원래 단계의 열이므로
	// 대화 왕복이 또 하나의 단계가 될 뿐이고 봉인 성질(ADR-005)이 안 깨진다.
	//
	// 실측 (2026-08-20): --resume 이 별개 -p 호출 사이로 문맥을 물고 온다.
	// 첫 호출 $0.1066 → 재개 $0.0099 로 ★ 왕복이 1/10 ★ 이다 (시스템 프롬프트·툴
	// 정의를 다시 안 문다). 대화가 예산에서 새 단계보다 훨씬 싼 항목이라는 뜻이다.
	Session string `json:"session,omitempty"`

	// Version 은 ★ 이 결과를 낸 하네스의 버전 ★ 이다.
	//
	// 우리가 의존하는 건 문서화된 프로토콜이 아니라 CLI 출력 형태이고,
	// claude 의 봉투는 ★ 비공개 계약 ★ 이다. 낯선 subtype 은 harness_error 로
	// 떨어져 닫히는 쪽으로 틀리지만, ★ 필드명이 바뀌면 Turns=0 이 되어
	// 예산 신호가 조용히 죽는다 ★ — 우리가 계속 금지해온 「조용한 무시」다.
	//
	// 버전이 기록에 있어야 봉인된 묶음만 보고 드리프트를 알 수 있다
	// (ADR-005 성질 4 — 자기충족). 비용은 --version 한 번과 이 한 줄이다.
	Version string `json:"version,omitempty"`
}

// claudeEnvelope 는 `claude -p --output-format json` 이 내는 것이다.
type claudeEnvelope struct {
	Type      string  `json:"type"`
	Subtype   string  `json:"subtype"`
	IsError   bool    `json:"is_error"`
	NumTurns  int     `json:"num_turns"`
	TotalCost float64 `json:"total_cost_usd"`
	Result    string  `json:"result"`
	SessionID string  `json:"session_id"`
}

// ParseClaude 는 claude headless 의 봉투를 정규화한다.
//
// 봉투가 아예 안 나오면(크래시·플래그 오류) harness_error 다 —
// ★ 종료코드 0 을 믿지 않는다 ★.
func ParseClaude(stdout []byte, exitCode int) HarnessResult {
	line := lastJSONObject(stdout)
	if line == "" {
		return HarnessResult{Reason: ReasonError,
			Message: "봉투가 없다 (exit " + itoa(exitCode) + ")"}
	}
	var e claudeEnvelope
	if err := json.Unmarshal([]byte(line), &e); err != nil {
		return HarnessResult{Reason: ReasonError, Message: "봉투를 못 읽었다: " + err.Error()}
	}
	h := HarnessResult{Turns: e.NumTurns, CostUSD: e.TotalCost, Session: e.SessionID}
	switch {
	case strings.Contains(e.Subtype, "max_turns"):
		h.Reason = ReasonMaxTurns
	case strings.Contains(e.Subtype, "max_tokens"), strings.Contains(e.Subtype, "token"):
		h.Reason = ReasonMaxTokens
	case e.IsError || e.Subtype != "" && e.Subtype != "success":
		h.Reason, h.Message = ReasonError, e.Subtype
	default:
		h.Reason = ReasonOK
	}
	return h
}

// lastJSONObject 는 출력 끝에 붙은 마지막 유효 JSON 객체를 찾는다.
//
// ★ 줄 단위로 찾으면 안 된다 ★ — 봉투가 여러 줄로 예쁘게 찍혀 올 수 있다.
// stdout 에 로그가 섞여 있어도 봉투를 집을 수 있어야 하는데, 이것이
// ADR-013 이 "stdout JSON 은 로그와 섞인다" 며 ★ 산출물은 $OUT 파일로 ★
// 뺀 이유이기도 하다. 봉투는 어쩔 수 없이 stdout 이지만 산출물은 아니다.
func lastJSONObject(b []byte) string {
	s := strings.TrimSpace(string(b))
	if s == "" || !strings.HasSuffix(s, "}") {
		return ""
	}
	// 뒤에서부터 '{' 를 찾아 거기서 끝까지가 유효한 JSON 인지 본다.
	for i := strings.LastIndexByte(s, '{'); i >= 0; i = strings.LastIndexByte(s[:i], '{') {
		if json.Valid([]byte(s[i:])) {
			return s[i:]
		}
	}
	return ""
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// ★ R3 — 하네스 어댑터 ★
//
// 「하네스마다 어댑터」를 고른 근거는 우리가 만든 것이 좋아서가 아니라
// ★ 우리 계약이 사는 층에 외부 프로토콜이 없어서 ★ 다 (claude.go 주석 참조).
// 어댑터 한 개는 argv 조립 + 봉투 정규화로 ★ 50 줄 안팎 ★ 이고,
// 그게 얇은 이유는 위층(워크스페이스·blob·봉인·임대)을 이미 가지고 있어서다.
//
// ★ exec 이 여기 없다 ★ — runner.go 하나가 띄운다. R1 의 환경 화이트리스트를
// 어댑터 수와 무관하게 한 번만 지키기 위해서다.
type Harness interface {
	Name() string  // 광고의 harness 속성이 된다
	Env() []string // 추가로 통과시킬 환경변수 ★ 이름 ★ (R1)
	Probe(ctx context.Context, bin string) (version string, err error)
	// Instrument 는 ★ 훅·플러그인을 심고 플래그를 돌려준다 ★ (R6).
	//
	// 하네스마다 심는 방법이 다르다 — claude 는 --settings 로 훅을 받고,
	// 다른 하네스는 다른 방식일 것이다. ★ 그 차이가 여기서만 보이게 한다 ★.
	// 심을 것이 없는 하네스는 빈 것을 돌려주면 된다.
	//
	// dir 은 ★ 하네스에 안 보이는 곳 ★ 이어야 한다 — $OUT 에 두면 ④수확이 걷는다.
	// self 는 enode 자기 실행경로다 — 훅이 곧 enode 자신이기 때문이다.
	Instrument(dir, self string, a HookArgs) ([]string, error)

	Argv(p AgentParams, io IOPaths) []string                          // ★ 순수 함수 ★
	Decode(r io.Reader, exitCode int, emit func(Event)) HarnessResult // ★ 순수 함수 ★
}

// EventKind 는 스트림 사건의 종류다.
//
// 지금은 final 하나만 난다 — 배치 봉투에는 중간 사건이 없기 때문이다.
// ★ 미리 여러 종류를 만들지 않는다 ★. 시그니처가 사건을 나를 수 있다는 것이
// 요점이고, 종류는 stream-json 을 켤 때 실물을 보고 늘린다.
type EventKind string

const (
	EventFinal EventKind = "final"
)

type Event struct {
	Kind EventKind
	Text string
}

// harnesses 는 등록된 어댑터다. ★ 지금은 하나뿐이고 그게 맞다 ★ —
// 붙일 두 번째 하네스가 없어서 acp.go 를 안 만들었다 (protocol/agent-runtime.md).
var harnesses = []Harness{claudeHarness{}}

// harnessFor 는 이름으로 어댑터를 찾는다.
func harnessFor(name string) (Harness, bool) {
	for _, h := range harnesses {
		if h.Name() == name {
			return h, true
		}
	}
	return nil, false
}
