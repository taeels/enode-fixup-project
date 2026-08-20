package enode

import (
	"context"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

// claudeHarness 는 ★ MVP 의 유일한 하네스 ★ 다.
//
// 하네스마다 어댑터를 쓰기로 한 이유는 NIH 가 아니다 —
// ★ 우리 계약이 사는 층에 그 프로토콜이 없다 ★:
//
//	ACP 가 나르는 것       프롬프트 · 툴콜 · 되묻기 · 편집 리뷰
//	우리 계약이 요구하는 것 $OUT 규약 · 완주/성공 2비트 · 워크스페이스 준비
//	                       blob 수거 · 회차 표기 · 예산 신호
//
// 겹치는 칸이 없다. 그래서 ACP 를 채택해도 오른쪽은 여전히 하네스마다 쓴다 —
// ★ 50 줄이 다른 50 줄로 바뀌고 위층은 그대로다 ★.
// claudeEnv 는 claude 가 ★ 추가로 ★ 필요로 하는 이름이다 (R1 화이트리스트에 더해진다).
//
// MVP 는 transparent 인증이라 보통 ~/.claude 의 로그인을 쓴다 — 그건 HOME 이
// 통과하는 것으로 이미 된다. 아래는 ★ 그 대신 환경변수로 붙이는 구성 ★ 을 위한 것이다.
//
// ★ 여기 없는 이름은 안 넘어간다 ★. 그래서 하네스가 인증을 못 찾으면
// droppedNotable() 이 무엇을 버렸는지 이름만 로그에 남긴다 — 조용히 실패하지 않게.
var claudeEnv = []string{
	"ANTHROPIC_API_KEY",
	"ANTHROPIC_AUTH_TOKEN",
	"ANTHROPIC_BASE_URL", // 사내 게이트웨이를 거치는 구성
	"CLAUDE_CODE_USE_BEDROCK",
	"CLAUDE_CODE_USE_VERTEX",
}

type claudeHarness struct{}

// execLookPath 는 시험이 같은 경로 해석을 쓰게 하려고 뺀 것이다.
var execLookPath = exec.LookPath

func (claudeHarness) Name() string { return "claude" }

// Env 는 claude 가 ★ 추가로 ★ 필요로 하는 이름이다 (R1 화이트리스트에 더해진다).
func (claudeHarness) Env() []string { return claudeEnv }

// Probe 는 ★ 설치 확인이자 executable resolve 다 ★.
//
// 없으면 광고에 안 실리고 → 후보에서 빠지고 → 계약이 요구하면 422 다.
// "설치 안 된 하네스의 실행은 MVP 에서 제외한다" 가 ★ 별도 코드 없이 ★ 성립한다.
//
// ★ 버전을 돌려주는 이유 ★ — 우리가 의존하는 건 문서화된 프로토콜이 아니라
// CLI 출력 형태이고, claude 의 봉투는 비공개 계약이다. 필드명이 바뀌면
// num_turns 를 못 읽어 Turns=0 이 되고 ★ 예산 신호가 조용히 죽는다 ★.
// 버전이 기록에 있어야 봉인된 묶음만 보고 드리프트를 알 수 있다 (ADR-005 성질 4).
func (claudeHarness) Probe(ctx context.Context, bin string) (string, error) {
	if bin == "" {
		bin = "claude"
	}
	path, err := execLookPath(bin)
	if err != nil {
		return "", err
	}
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		// ★ 있는데 --version 이 실패하면 「있다」로 본다 ★.
		// 버전을 모르는 것과 없는 것은 다르다 — 없다고 하면 광고가 빠져
		// 시연 직전에 노드가 통째로 사라진다.
		return "unknown", nil
	}
	return strings.TrimSpace(string(out)), nil
}

// Instrument 는 enode 전용 종료 훅을 심는다 (R5③ · R6).
//
// ★ claude 는 --settings 로 받는다 ★ — 훅 하나뿐이라 플러그인 디렉터리까지
// 만들 필요가 없고, 파일이 적을수록 정리도 확실하다.
func (claudeHarness) Instrument(dir, self string, a HookArgs) ([]string, error) {
	return WriteHookSettings(dir, self, a)
}

// Argv 는 ★ 순수 함수다 ★ — 프로세스를 안 띄운다. 그래서 시험이 싸다.
func (claudeHarness) Argv(p AgentParams, _ IOPaths) []string {
	args := []string{"-p", "--output-format", "json"}
	if p.Model != "" {
		args = append(args, "--model", p.Model)
	}
	if p.MaxTurns > 0 {
		args = append(args, "--max-turns", strconv.Itoa(p.MaxTurns))
	}
	// ask 는 계약이 못 박는다. 무인 실행이므로 물으면 매달린다 (ADR-013 결정 5).
	if p.Ask == "" || p.Ask == "never" {
		args = append(args, "--permission-mode", "bypassPermissions")
	}
	return args
}

// Decode 는 ★ 스트림을 훑는다 ★ — 끝난 뒤의 덩어리를 받지 않는다.
//
// Parse(stdout []byte, …) 로 두면 ★ 대화가 원천봉쇄된다 ★. claude 는
// --output-format stream-json / --input-format stream-json 으로 실시간을
// 지원하므로(실측), 막는 것은 하네스가 아니라 이쪽 추상화가 된다.
//
// ★ 배치는 스트리밍의 퇴화형이다 ★ — 지금은 EOF 까지 읽고 사건 하나를 낸다.
// R4 를 열 때 stream-json 으로 바꿔도 ★ 이 시그니처는 안 바뀐다 ★.
func (claudeHarness) Decode(r io.Reader, exitCode int, emit func(Event)) HarnessResult {
	b, err := io.ReadAll(r)
	if err != nil {
		return HarnessResult{Reason: ReasonError, Message: "출력을 못 읽었다: " + err.Error()}
	}
	h := ParseClaude(b, exitCode)
	emit(Event{Kind: EventFinal, Text: h.Message})
	return h
}
