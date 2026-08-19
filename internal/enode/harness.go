package enode

import (
	"encoding/json"
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
	h := HarnessResult{Turns: e.NumTurns, CostUSD: e.TotalCost}
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
