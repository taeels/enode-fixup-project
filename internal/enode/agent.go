package enode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/taeels/enode/internal/contract"
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

// cannotName 은 ★ 못 하겠다는 것을 말하는 자리 ★ 다 (ADR-038).
//
// ★ 밑줄 예약이라 계약이 이 이름을 못 쓴다 ★ (contract.go 가 400 으로 막는다) —
// 계약의 산출물과 절대 안 겹친다.
const cannotName = "_cannot"

// failLane 은 ★ 실패 차선 ★ 이다 — 스키마가 있든 없든 항상 붙는다.
//
// ★ 왜 필요한가 ★ — 실측: --json-schema 에 const:true 를 박으면 모델이
// 아무것도 안 짓고도 built:true 를 뱉는다. ★ 스키마는 진실 장치가 아니라 통로이고,
// 실패 차선을 막으면 그 통로로 거짓이 흐른다 ★ (ADR-037 §1.1).
//
// ★ 자백은 믿어도 된다 ★ — 성공 주장과 비대칭이다. 못 했다고 말해서 얻을 것이 없다.
// 그래서 이 파일은 검증하지 않고 그대로 기록한다.
const failLane = `
## 못 하겠을 때

요구된 것을 낼 수 없으면 ★ 성공을 주장하지 말고 ★ 아래 파일에 이유를 적어라.

  %s

★ 이 파일은 요구된 산출물의 대체물이 아니다 ★ — 단계는 그대로 실패한다.
그래도 ★ 거짓으로 성공을 주장하는 것보다 낫다 ★: 왜 못 했는지가 기록에 남고,
다음 시도가 그것을 읽는다. ★ 검증은 파일이 하지 네 말이 하지 않는다 ★.
`

// buildPrompt 는 ①사출의 일부다 — 규약 · 스키마 · 되먹임 · 요청을 이 순서로 쌓는다.
//
// 순서에 이유가 있다: 규약을 먼저 두면 모델이 마지막 지시(요청)를 수행하면서도
// 형식을 유지하고, ★ 되먹임을 요청 바로 앞에 두면 ★ 무엇을 고쳐야 하는지가
// 가장 가깝게 놓인다.
func buildPrompt(req, outDir string, outNames []string, schema map[string]json.RawMessage,
	feedback map[string]string, attempt int, expands bool,
	roles, owed []string, goal string) string {
	var b strings.Builder
	b.WriteString(outContract)
	// ★ 계약을 짓는 단계에는 계약 문법을 심는다 ★ (ADR-045)
	//
	// outContract 가 ★ 네가 무엇을 어떻게 낼 것인가 ★ 를 말한다면, 이것은
	// ★ 네가 짓는 단계들이 무엇을 지켜야 하는가 ★ 다. 둘은 다른 층이고,
	// 그래서 계획 위임에서는 ★ 둘 다 필요하다 ★ — 배출 규약만 심으면
	// 계획의 형식은 여전히 사람이 자연어로 나른다.
	if expands {
		b.WriteString("\n" + contract.Grammar + "\n")
		// ★ 문법은 uses 를 적으라고만 말한다 ★ — 무엇을 적는지는 그 계약의
		// requires 에 있고 계획을 짓는 쪽은 그것을 못 본다. ★ 아는 쪽이 적어준다 ★.
		if len(roles) > 0 {
			b.WriteString("### ★ uses 에 쓸 수 있는 역할은 이것뿐이다 ★\n\n")
			for _, r := range roles {
				b.WriteString("    " + r + "\n")
			}
			b.WriteString("\n★ 여기 없는 이름을 쓰면 계획 전체가 거절된다 ★ — " +
				"자원은 계약 저자가 선언한다.\n\n")
		}
		// ★ 목표를 나른다 ★ (ADR-049) — 계획이 지은 재계획 단계는 자기 프롬프트가
		// 비어 있을 수 있다. 그러면 ★ 재료를 받고도 무엇을 향해 지을지 모른다 ★.
		// 실측에서 밟았다: "요청 섹션이 비어 있다. 목표는 어디에도 명시돼 있지 않다".
		if goal != "" {
			b.WriteString("### ★ 이 Run 이 처음 받은 목표 ★\n\n" +
				"아래는 이 Run 을 시작한 사람이 적은 것이다. " +
				"★ 네가 짓는 계획은 여전히 이것을 향한다 ★.\n\n" +
				"```\n" + trimTo(goal, 6000) + "\n```\n\n")
		}
		// ★ 무엇이 아직 안 섰는지 ★ (ADR-049) — 계약이 약속한 단계 이름이다.
		if len(owed) > 0 {
			b.WriteString("### ★ 계약이 약속했는데 아직 안 지어진 단계 ★\n\n")
			for _, n := range owed {
				b.WriteString("    " + n + "\n")
			}
			b.WriteString("\n★ 이 이름을 가진 단계를 지어야 한다 ★ — " +
				"success_when 이 이미 이 이름을 가리키고 있고, " +
				"안 지으면 ★ 계획이 거절된다 ★. 그리고 이것이 남아 있는 한 " +
				"★ 빈 계획을 낼 수 없다 ★.\n\n")
		}
	}
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
	// ★ 되먹임은 회차와 무관하게 싣는다 ★ (ADR-048)
	//
	// 예전에는 attempt > 0 일 때만 실었다. 그래서 계획이 지은 재계획 단계가
	// ★ 앞 단계 로그를 하나도 못 봤다 ★ — expands 로 붙은 단계는 attempt 0 이다.
	//
	// ★ 제목이 회차에 따라 달라진다 ★ — 무엇을 보고 있는지가 달라지기 때문이다:
	//	attempt > 0  ★ 앞 시도의 나 ★ 가 남긴 것 (자백 포함)
	//	attempt = 0  ★ 앞 단계들 ★ 이 남긴 것 — 실패했다고 단정하면 안 된다.
	//	             성공한 로그를 보고 「고칠 것이 없다」를 판단하는 것도 이 자리다
	if len(feedback) > 0 {
		if attempt > 0 {
			b.WriteString("\n### 앞 시도가 실패했다 (" + strconv.Itoa(attempt) + "회차)\n\n")
		} else {
			b.WriteString("\n### 앞 단계들이 남긴 것\n\n" +
				"계약이 이 단계에 되먹이라고 지목한 산출물이다. ★ 읽고 판단하라 ★ —\n" +
				"실패했을 수도 있고 아무 문제가 없을 수도 있다.\n\n")
		}
		names := make([]string, 0, len(feedback))
		for n := range feedback {
			names = append(names, n)
		}
		sortStrings(names)
		for _, n := range names {
			b.WriteString(n + ":\n```\n" + trimTo(feedback[n], 4000) + "\n```\n")
		}
	}
	// ★ 실패 차선은 항상 붙는다 ★ — 스키마 유무와 무관하다 (ADR-038).
	// 위의 ADR-020 문구는 ★ 스키마가 있을 때만 ★ 이고 "스키마가 허용하는 형태" 를
	// 요구하는데, ★ 스키마가 차선을 안 뚫었으면 허용하는 형태가 없다 ★ — 순환이다.
	b.WriteString(fmt.Sprintf(failLane, filepath.Join(outDir, cannotName)))
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
		return p, fmt.Errorf("agent parameters are missing")
	}
	return p, json.Unmarshal(raw, &p)
}
