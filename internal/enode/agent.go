package enode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// AgentParams 는 계약의 steps[].agent 다 (ADR-019 에서 with 를 개명한 것).
//
// ★ 매칭 조건이 아니라 실행 파라미터다 ★ (ADR-013 결정 4) —
// model 을 requires 에 넣으면 그 모델이 없는 노드가 매칭 실패가 된다.
type AgentParams struct {
	Model     string `json:"model,omitempty"`
	MaxTurns  int    `json:"max_turns,omitempty"`
	MaxTokens int    `json:"max_tokens,omitempty"`
	// Ask 는 ★ 계약이 못 박는다 ★ — 무인 실행이므로 하네스가 권한을 물으면
	// 타임아웃까지 매달린다. ACP 가 기본으로 묻는 프로토콜이라 명시해야 한다.
	Ask string `json:"ask,omitempty"`
	// Harness 는 ★ 어느 어댑터로 돌릴지 ★ 다. 비면 claude.
	// 광고의 harness 속성과 같은 어휘를 쓴다 — 매처가 노드를 고르고,
	// 이 값이 그 노드 위에서 어느 어댑터를 부를지 고른다.
	Harness string `json:"harness,omitempty"`
}

// ★ 배출 규약을 프롬프트에 심는다 ★ — ADR-013 이 [미정] 으로 남긴 자리다.
//
// stdout JSON 은 로그와 섞이고 구조화 출력 API 는 하네스마다 다르다.
// ★ 파일은 어떤 하네스든 쓸 수 있고 셸로 검사된다 ★.
const outContract = `## 배출 규약 (이 형식을 지켜야 결과가 채택된다)

★ 다른 무엇보다 먼저 ★ 아래 파일들을 만들어라. 표준출력으로 내지 않는다.
경로는 ★ 있는 그대로 ★ 쓴다 — 환경변수가 아니라 실제 경로다.
`

// buildPrompt 는 ①사출의 일부다 — 규약 · 스키마 · 되먹임 · 요청을 이 순서로 쌓는다.
//
// 순서에 이유가 있다: 규약을 먼저 두면 모델이 마지막 지시(요청)를 수행하면서도
// 형식을 유지하고, ★ 되먹임을 요청 바로 앞에 두면 ★ 무엇을 고쳐야 하는지가
// 가장 가깝게 놓인다.
func buildPrompt(req, outDir string, outNames []string, schema map[string]json.RawMessage, feedback map[string]string, attempt int) string {
	var b strings.Builder
	b.WriteString(outContract)
	for _, n := range outNames {
		// ★ 실제 경로를 박는다 ★ — $OUT 을 문자 그대로 주면 모델이 확장하지 않는다.
		// 실물 claude 에서 밟았다: 6턴을 쓰고도 아무 파일도 안 만들었다.
		// ★ 어댑터는 경로를 아는데 모델은 모른다. 아는 쪽이 적어준다. ★
		b.WriteString("  " + filepath.Join(outDir, n))
		if _, ok := schema[n]; ok {
			b.WriteString("   ← 아래 스키마를 만족하는 JSON 한 덩어리")
		}
		b.WriteString("\n")
	}
	if len(schema) > 0 {
		b.WriteString("\n### 스키마\n\n")
		names := make([]string, 0, len(schema))
		for n := range schema {
			names = append(names, n)
		}
		sortStrings(names)
		for _, n := range names {
			b.WriteString(filepath.Join(outDir, n) + ":\n```json\n" + string(schema[n]) + "\n```\n")
		}
		// ★ 스키마가 있으면 "못 하겠다" 를 값으로 말할 수 있어야 한다 ★ (ADR-020)
		b.WriteString("\n결론이 없으면 ★ 파일을 안 내는 것이 아니라 ★ 스키마가 허용하는\n" +
			"형태로 그 사실을 적는다. 부재는 크래시와 구분되지 않는다.\n")
	}
	if attempt > 0 && len(feedback) > 0 {
		b.WriteString("\n### 앞 시도가 실패했다 (" + strconv.Itoa(attempt) + "회차)\n\n")
		names := make([]string, 0, len(feedback))
		for n := range feedback {
			names = append(names, n)
		}
		sortStrings(names)
		for _, n := range names {
			b.WriteString(n + ":\n```\n" + trimTo(feedback[n], 4000) + "\n```\n")
		}
	}
	b.WriteString("\n### 요청\n\n")
	b.WriteString(req)
	b.WriteString("\n")
	return b.String()
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// trimTo 는 되먹임이 프롬프트를 압도하지 않게 한다. 뒤쪽이 오류에 가깝다.
func trimTo(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "… (앞부분 생략)\n" + s[len(s)-n:]
}

// writePromptFile 은 프롬프트를 작업 폴더에도 남긴다 — 무엇을 물었는지가
// 사람 눈에 보여야 디버깅이 된다. Record 에는 logs/ 로 들어간다.
func writePromptFile(dir, prompt string) {
	_ = os.WriteFile(filepath.Join(dir, ".enode-prompt.md"), []byte(prompt), 0o600)
}

func parseAgentParams(raw json.RawMessage) (AgentParams, error) {
	var p AgentParams
	if len(raw) == 0 {
		return p, fmt.Errorf("agent 파라미터가 없다")
	}
	return p, json.Unmarshal(raw, &p)
}
