package enode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// Fixed 는 ★ 재현성을 위해 우리가 값으로 박는 것 ★ 이다.
//
// ★ --setting-sources ” 만으로는 개인 설정이 안 막힌다 ★ (2026-08-20)
//
// R6 이 재현성 방어로 --setting-sources ” 를 걸었는데(hook.go), Agent SDK 호스팅
// 문서가 그것만으로 부족하다고 명시한다 — ★ 자동 메모리는 settingSources 와
// 무관하게 시스템 프롬프트로 들어간다 ★:
//
//	~/.claude/projects/<워크스페이스 경로 인코딩>/memory/
//	     │
//	     ├─ R1 이 HOME 을 통과시킨다 (MVP 의 transparent 인증이 곧 그것이다)
//	     └─ 노드 주인이 그 워크스페이스에서 claude 를 한 번이라도 썼으면 파일이 있다
//	     ▼
//	★ 노드마다 시스템 프롬프트가 달라진다 ★ = Run 이 재현 불가능
//	  그리고 ★ Record 에는 그 사실이 안 남는다 ★ — 조용히 갈린다.
//
// ★ 이름이 아니라 값인 것이 핵심이다 ★ — Env 에 넣으면 노드 환경에 그 변수가
// 없을 때 안 걸리고, 그건 「기본이 통과」라서 막은 R1 의 실수를 되풀이하는 것이다.
func (claudeHarness) Fixed() map[string]string {
	return map[string]string{"CLAUDE_CODE_DISABLE_AUTO_MEMORY": "1"}
}

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
	// ★ 「있다」와 「쓸 수 있다」를 가른다 ★ (ADR-059)
	//
	// 예전에는 --version 만 봤다. 그런데 ★ --version 은 로그인 없이도 답한다 ★.
	//
	//	★ 실측 ★ (2026-08-24) vm-scratch-7 이 colima VM 에 세운 노드가
	//	harness=claude 를 광고했는데, agent 단계가 ★ 1턴 1초에 죽었다 ★:
	//	  terminal_reason: api_error
	//	  result: "Not logged in · Please run /login"
	//	★ 매칭은 통과하고 실행 시점에 죽는다 ★ — 가장 늦게 아는 실패다.
	//
	// ADR-012 가 적은 그대로다 — "못 하는 것을 ★ 빼고 ★ 보내는 것이
	// 「지금은 못 한다」를 표현하는 방법이다".
	if err := claudeUsable(ctx, path); err != nil {
		return "", err
	}
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		// ★ 쓸 수 있는데 --version 이 실패하면 「있다」로 본다 ★.
		// 버전을 모르는 것과 못 쓰는 것은 다르다 — 없다고 하면 광고가 빠져
		// 시연 직전에 노드가 통째로 사라진다.
		return "unknown", nil
	}
	return strings.TrimSpace(string(out)), nil
}

// errNotUsable 은 ★ 있는데 못 쓴다 ★ 는 뜻이다 (ADR-059).
var errNotUsable = errors.New("harness is installed but not usable")

// claudeUsable 은 이 claude 로 실제로 일을 시킬 수 있는지 본다.
//
// ★ 값싸야 한다 ★ — Detect 는 광고마다 돈다(기본 60초). 그래서
// `claude auth status` 를 쓴다: ★ 로컬 확인이고 0.5 초 안에 답한다 ★.
// 짧은 프롬프트를 실제로 돌려보는 방법은 ★ 매 분 토큰을 태운다 ★.
//
// ★ 모르면 「쓸 수 있다」로 본다 ★ — 이 하위명령이 없는 옛 CLI 나 형식이
// 바뀐 새 CLI 에서 ★ 노드가 통째로 사라지면 안 된다 ★. 확실히 아니라고
// 말할 때만 뺀다. (틀리면 닫히는 쪽이 아니라 ★ 열리는 쪽 ★ 인데, 그 이유는
// 여기서는 ★ 조용한 사라짐이 조용한 실패보다 나쁘기 때문 ★ 이다 —
// 못 쓰는 노드는 실행 시점에 _cannot 으로 드러나지만, 사라진 노드는
// 「왜 매칭이 안 되지」로 남는다.)
func claudeUsable(ctx context.Context, path string) error {
	out, err := exec.CommandContext(ctx, path, "auth", "status", "--json").Output()
	if err != nil {
		return nil // 하위명령이 없거나 못 돌았다 — 모르는 것이지 아닌 것이 아니다
	}
	var st struct {
		LoggedIn *bool `json:"loggedIn"`
	}
	if json.Unmarshal(out, &st) != nil || st.LoggedIn == nil {
		return nil // 형식을 모른다 — 위와 같다
	}
	if !*st.LoggedIn {
		return fmt.Errorf("%w: claude is not logged in", errNotUsable)
	}
	return nil
}

// Instrument 는 enode 전용 종료 훅을 심는다 (R5③ · R6).
//
// ★ claude 는 --settings 로 받는다 ★ — 훅 하나뿐이라 플러그인 디렉터리까지
// 만들 필요가 없고, 파일이 적을수록 정리도 확실하다.
func (claudeHarness) Instrument(dir, self string, a HookArgs) ([]string, error) {
	return WriteHookSettings(dir, self, a)
}

// Argv 는 ★ 순수 함수다 ★ — 프로세스를 안 띄운다. 그래서 시험이 싸다.
func (claudeHarness) Argv(p AgentParams, io IOPaths) []string {
	args := []string{"-p", "--output-format", "json"}
	if p.Model != "" {
		args = append(args, "--model", p.Model)
	}
	if p.MaxTurns > 0 {
		args = append(args, "--max-turns", strconv.Itoa(p.MaxTurns))
	}
	// ★ 경계는 단계가 아니라 노드에 있다 ★ (2026-08-22, ADR-042)
	//
	// 이 자리는 두 번 바뀌었고 ★ 값은 처음으로 돌아왔지만 근거가 다르다 ★.
	//
	//	1차  bypassPermissions   ADR-013 의 ask:never 를 「모든 권한 우회」로
	//	                        ★ 잘못 번역한 것 ★ 이었다 (R7 이 지적했다)
	//	2차  acceptEdits+add-dir R7 실측이 고른 값. 근거는 ★ 포함관계 ★ 였다:
	//	                        모델이 쓸 수 있는 곳 ⊆ 훅이 볼 수 있는 곳
	//	3차  bypassPermissions   ★ 그 포함관계가 살 자리가 아니었다 ★
	//
	// ★ 왜 3차인가 — 명령 단계가 이미 무경계다 ★
	//
	// INVARIANTS 가 적어둔 그대로다: *명령 단계에는 파일시스템 경계가 없다.
	// argv 는 무엇이든 되므로 걸 자리가 없다. ★ 그래서 노드는 소유자가
	// 신뢰하는 계약만 받는다 ★.* 그런데 같은 계약이
	//
	//	run:   ["brew", "install", "colima"]     ★ 오늘 그냥 된다 ★
	//	agent: "colima 를 설치해라"               ★ 거부된다 ★
	//
	// 로 갈렸다. ★ 위협이 같은데 규칙이 달랐다 ★ — 그리고 좁은 쪽이 산 것은
	// 안전이 아니라 ★ 에이전트가 명령을 못 돌린다 ★ 였다.
	//
	// ★ enode 를 띄운 것이 곧 허가다 ★ — 토큰을 쥐여 미디에이터를 가리킨 순간
	// 노드 주인은 이 기계가 임의의 argv 를 받는다고 선언한 것이다. 그 선언 뒤에
	// 에이전트만 가두는 것은 위협 모델을 좁히지 못하고 능력만 좁힌다.
	//
	// ★ R7 의 실측은 틀리지 않았다 ★ — 각 모드가 무엇을 여는지는 그대로 맞다.
	// 바뀐 것은 ★ 어느 대가를 받을 것인가 ★ 이고, R7 은 명령 단계와의 불일치를
	// 저울에 안 올렸다.
	//
	// ★ 그래서 무엇이 남아서 지키나 ★
	//   sealInput()  $IN 을 0444·0555 로 잠근다 (collect.go).
	//                ★ 권한 모드가 아니라 파일시스템이 거는 것 ★ 이라 여기 안 묶인다.
	//                오히려 이제 그것이 ★ 유일한 기계적 방어 ★ 다.
	//   harnessEnv() R1 환경 화이트리스트. ENODE_TOKEN 은 여전히 안 넘어간다 —
	//                ★ 열쇠가 하나 ★ 라는 전제는 I1 이 서 있는 자리라 안 건드린다.
	//
	// ★ 도구는 여전히 나열하지 않는다 ★ — --allowed-tools 는 제한이 아니라
	// 자동승인 목록이고(R7 실측), 여기서는 아무 일도 안 하면서 이름만 늘린다.
	if p.Ask == "" || p.Ask == "never" {
		args = append(args, "--permission-mode", "bypassPermissions")
	}
	// ★ --add-dir 을 남긴다 ★ — bypassPermissions 아래서 권한상으로는 불필요하다.
	// 남기는 이유는 ★ 의도가 argv 에 남아야 ★ 하기 때문이다: 이 단계가 어디를
	// 쓸 셈이었는지가 Record 의 하네스 로그에 그대로 찍힌다. 지우면 그 사실이
	// 코드에만 있고 기록에는 없다.
	for _, d := range []string{io.Out, io.In} {
		if d != "" && d != io.Dir {
			args = append(args, "--add-dir", d)
		}
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
		return HarnessResult{Reason: ReasonError, Message: "cannot read output: " + err.Error()}
	}
	h := ParseClaude(b, exitCode)
	emit(Event{Kind: EventFinal, Text: h.Message})
	return h
}
