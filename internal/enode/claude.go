package enode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// claudeHarness 는 MVP 의 유일한 하네스다.
//
// 하네스마다 어댑터를 쓰기로 한 이유는 NIH 가 아니다 —
// 우리 계약이 사는 층에 그 프로토콜이 없다:
//
//	ACP 가 나르는 것       프롬프트 · 툴콜 · 되묻기 · 편집 리뷰
//	우리 계약이 요구하는 것 $OUT 규약 · 완주/성공 2비트 · 워크스페이스 준비
//	                       blob 수거 · 회차 표기 · 예산 신호
//
// 겹치는 칸이 없다. 그래서 ACP 를 채택해도 오른쪽은 여전히 하네스마다 쓴다 —
// 50 줄이 다른 50 줄로 바뀌고 위층은 그대로다.
// claudeEnv 는 claude 가 추가로 필요로 하는 이름이다 (R1 화이트리스트에 더해진다).
//
// MVP 는 transparent 인증이라 보통 ~/.claude 의 로그인을 쓴다 — 그건 HOME 이
// 통과하는 것으로 이미 된다. 아래는 그 대신 환경변수로 붙이는 구성을 위한 것이다.
//
// 여기 없는 이름은 안 넘어간다. 그래서 하네스가 인증을 못 찾으면
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

// Env 는 claude 가 추가로 필요로 하는 이름이다 (R1 화이트리스트에 더해진다).
func (claudeHarness) Env() []string { return claudeEnv }

// Fixed 는 재현성을 위해 우리가 값으로 박는 것이다.
//
// --setting-sources ” 만으로는 개인 설정이 안 막힌다 (2026-08-20)
//
// R6 이 재현성 방어로 --setting-sources ” 를 걸었는데(hook.go), Agent SDK 호스팅
// 문서가 그것만으로 부족하다고 명시한다 — 자동 메모리는 settingSources 와
// 무관하게 시스템 프롬프트로 들어간다:
//
//	~/.claude/projects/<워크스페이스 경로 인코딩>/memory/
//	     │
//	     ├─ R1 이 HOME 을 통과시킨다 (MVP 의 transparent 인증이 곧 그것이다)
//	     └─ 노드 주인이 그 워크스페이스에서 claude 를 한 번이라도 썼으면 파일이 있다
//	     ▼
//	노드마다 시스템 프롬프트가 달라진다 = Run 이 재현 불가능
//	  그리고 Record 에는 그 사실이 안 남는다 — 조용히 갈린다.
//
// 이름이 아니라 값인 것이 핵심이다 — Env 에 넣으면 노드 환경에 그 변수가
// 없을 때 안 걸리고, 그건 「기본이 통과」라서 막은 R1 의 실수를 되풀이하는 것이다.
//
// 가짜 홈도 같은 자리에서 박는다 (features.md 3.1)
//
// CLAUDE_CONFIG_DIR 이 계장이 지은 홈을 가리킨다. 그래야 개인 설정과 계정
// 커넥터가 실행에 안 섞이고, 팩이 펴질 자리가 생긴다. 자동 메모리를 끄는 값은
// 그대로 둔다 — 가짜 홈에는 메모리가 없지만 값 하나가 두 겹을 만든다.
//
// dir 은 계장 임시 디렉터리다. 그것을 못 만들면 단계가 이미 실패했으므로
// (runner.go 의 ①) 여기 빈 값이 오는 경로가 없다.
func (claudeHarness) Fixed(dir string) map[string]string {
	return map[string]string{
		"CLAUDE_CODE_DISABLE_AUTO_MEMORY": "1",
		"CLAUDE_CONFIG_DIR":               claudeHome(dir),
	}
}

// claudeHome 은 계장 임시 디렉터리 아래 가짜 홈의 자리다.
//
// 자리를 한 함수가 진다 — Fixed 가 박는 값과 Instrument 가 만드는 디렉터리가
// 갈리면 하네스가 없는 홈을 가리키고, 그때 무엇을 하는지는 실측이 없다.
func claudeHome(dir string) string { return filepath.Join(dir, "home") }

// Usable 은 설치 확인이자 executable resolve 이자 「쓸 수 있는가」다 (ADR-059).
//
// 없으면 광고에 안 실리고 → 후보에서 빠지고 → 계약이 요구하면 422 다.
// "설치 안 된 하네스의 실행은 MVP 에서 제외한다" 가 별도 코드 없이 성립한다.
//
// 버전을 안 알아낸다 — 광고가 버전을 안 싣기 때문이다(detect.go).
// 예전에는 이 자리가 Probe 하나였고 「쓸 수 있는가」와 버전을 함께 돌려줬다.
// 그런데 광고 경로는 버전을 받자마자 버렸고, 버리는 값을 위해 프로세스가
// 60초마다 하나씩 더 떴다. 부르는 쪽이 필요한 것만 부르게 가른다.
func (claudeHarness) Usable(ctx context.Context, bin string) error {
	path, err := claudePath(bin)
	if err != nil {
		return err
	}
	// 「있다」와 「쓸 수 있다」를 가른다 (ADR-059)
	//
	// 예전에는 --version 만 봤다. 그런데 --version 은 로그인 없이도 답한다.
	//
	//	실측 (2026-08-24) vm-scratch-7 이 colima VM 에 세운 노드가
	//	harness=claude 를 광고했는데, agent 단계가 1턴 1초에 죽었다:
	//	  terminal_reason: api_error
	//	  result: "Not logged in · Please run /login"
	//	매칭은 통과하고 실행 시점에 죽는다 — 가장 늦게 아는 실패다.
	//
	// ADR-012 가 적은 그대로다 — "못 하는 것을 빼고 보내는 것이
	// 「지금은 못 한다」를 표현하는 방법이다".
	return claudeUsable(ctx, path)
}

// Version 은 기록에 남길 버전 문자열이다 (ADR-005 성질 4).
//
// 우리가 의존하는 건 문서화된 프로토콜이 아니라 CLI 출력 형태이고,
// claude 의 봉투는 비공개 계약이다. 필드명이 바뀌면 num_turns 를 못 읽어
// Turns=0 이 되고 예산 신호가 조용히 죽는다. 버전이 기록에 있어야
// 봉인된 묶음만 보고 드리프트를 알 수 있다.
//
// 실행 경로만 부른다 — 광고는 이것을 안 부른다.
func (claudeHarness) Version(ctx context.Context, bin string) (string, error) {
	path, err := claudePath(bin)
	if err != nil {
		return "", err
	}
	out, err := child(exec.CommandContext(ctx, path, "--version")).Output()
	if err != nil {
		// 있는데 --version 이 실패하면 「모른다」다. 버전을 모르는 것과
		// 못 쓰는 것은 다르다 — 여기서 오류를 내면 부르는 쪽이 이미 끝난
		// 실행의 기록을 통째로 버릴 수 있다.
		return "unknown", nil
	}
	return strings.TrimSpace(string(out)), nil
}

// claudePath 는 빈 이름을 기본값으로 채우고 실행 파일을 해석한다.
func claudePath(bin string) (string, error) {
	if bin == "" {
		bin = "claude"
	}
	return execLookPath(bin)
}

// errNotUsable 은 있는데 못 쓴다는 뜻이다 (ADR-059).
var errNotUsable = errors.New("harness is installed but not usable")

// claudeUsable 은 이 claude 로 실제로 일을 시킬 수 있는지 본다.
//
// 값싸야 한다 — Detect 는 광고마다 돈다(기본 60초). 그래서
// `claude auth status` 를 쓴다: 로컬 확인이고 0.5 초 안에 답한다.
// 짧은 프롬프트를 실제로 돌려보는 방법은 매 분 토큰을 태운다.
//
// 모르면 「쓸 수 있다」로 본다 — 이 하위명령이 없는 옛 CLI 나 형식이
// 바뀐 새 CLI 에서 노드가 통째로 사라지면 안 된다. 확실히 아니라고
// 말할 때만 뺀다. (틀리면 닫히는 쪽이 아니라 열리는 쪽 인데, 그 이유는
// 여기서는 조용한 사라짐이 조용한 실패보다 나쁘기 때문이다 —
// 못 쓰는 노드는 실행 시점에 _cannot 으로 드러나지만, 사라진 노드는
// 「왜 매칭이 안 되지」로 남는다.)
func claudeUsable(ctx context.Context, path string) error {
	// 종료코드를 안 본다. 나온 것을 본다
	//
	// 처음에는 Output() 을 썼고, 그것이 종료코드가 0 이 아니면 실패로 읽는다.
	// 그런데 로그아웃 상태의 claude 는 auth status 가 0 이 아닐 수 있다 —
	// 그러면 "모르는 것" 으로 떨어져 이 결정이 아무 일도 안 한다.
	//
	//	실측 (2026-08-24) colima VM 을 갱신했는데 harness 가 그대로 실렸다.
	//	그 VM 에서 `claude auth status --json` 은 {"loggedIn":false} 를
	//	분명히 찍고 있었다 — 우리가 그것을 안 읽은 것이다.
	//
	// ⇒ stdout 에 판정할 것이 있으면 종료코드와 무관하게 읽는다.
	cmd := child(exec.CommandContext(ctx, path, "auth", "status", "--json"))
	var buf bytes.Buffer
	cmd.Stdout = &buf
	_ = cmd.Run() // 종료코드는 안 본다
	var st struct {
		LoggedIn *bool `json:"loggedIn"`
	}
	if json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &st) != nil || st.LoggedIn == nil {
		return nil // 형식을 모른다 — 모르는 것이지 아닌 것이 아니다
	}
	if !*st.LoggedIn {
		return fmt.Errorf("%w: claude is not logged in", errNotUsable)
	}
	return nil
}

// Instrument 는 이 하네스의 사적인 세계를 파일로 짓고 플래그를 돌려준다 (R5③ · R6).
//
//	<dir>/home/                   가짜 홈.  가장 먼저 만든다.  CLAUDE_CONFIG_DIR 이 가리킨다
//	<dir>/home/settings.json      훅과 게이트웨이 인증 필드
//	<dir>/home/.credentials.json  실제 홈에 있으면 0600 으로 복사
//	<dir>/mcp.json                허용목록.  홈 밖이다
//	<dir>/home/skills · agents    U5 가 짓는다.  없는 것이 오늘의 참이다
//
// 조기 반환하지 않는다 — 보조 실패를 만나도 남은 쓰기를 끝까지 하고 마지막에
// 낸다. 훅 쓰기 하나가 실패했다고 나가면 허용목록이 아예 안 쓰이고 치명도
// 안 나서, 등급 표가 치명으로 잡으려던 것이 보조 경로로 되돌아온다.
//
// 오류에도 이미 얻은 플래그를 함께 돌려준다. 격리는 겹이 여럿이고, 보조 실패
// 하나가 --strict-mcp-config 를 떨어뜨리면 그중 확실한 겹 하나가 사라진다.
func (claudeHarness) Instrument(dir, self string, a HookArgs, c Components) ([]string, error) {
	// ① 가짜 홈을 가장 먼저 만든다. 실패는 치명이다 —
	// CLAUDE_CONFIG_DIR 이 없는 경로를 가리키면 하네스가 진짜 홈으로
	// 되돌아갈 수 있고, 그러면 격리가 조용히 풀린다.
	home := claudeHome(dir)
	if err := os.MkdirAll(home, 0o700); err != nil {
		return nil, fmt.Errorf("cannot create the harness home: %w", err)
	}
	// --setting-sources "" 는 파일을 안 가리킨다 — 「아무것도 읽지 마라」라서
	// 우리 파일의 성패와 무관하다. 그래서 언제나 붙는다: 개인 · 프로젝트 설정과
	// 워크스페이스 .mcp.json 을 끊는 겹이 이것이고, 가짜 홈이 끊는 것과 다르다.
	flags := []string{"--setting-sources", ""}

	// ② 훅 설정. 유일한 보조 등급이다 — 실패해도 남은 쓰기를 마저 한다.
	// 없는 파일을 가리키는 --settings 를 붙이면 하네스가 아예 안 뜨므로
	// 이 플래그는 성공했을 때만 붙는다.
	hookFlags, aux := WriteHookSettings(home, self, a)
	flags = append(flags, hookFlags...)

	// ③ 팩을 홈 아래 편다 — U5 다. c.Pack 이 nil 이면 아무것도 안 한다.

	// ④ 허용목록. 실패는 치명이다 — 안 쓰이면 요청한 서버가 조용히 없고,
	// 그 단계는 exit 0 으로 성공이 봉인된다.
	mcpPath := filepath.Join(dir, mcpAllowlistName)
	if err := writeMCPAllowlist(mcpPath, c.Servers); err != nil {
		return flags, fmt.Errorf("cannot write mcp allowlist: %w", err)
	}
	// 등호 형태다 — --mcp-config 는 가변인자라 띄어 쓰면 뒤따르는 인자를 삼킨다.
	flags = append(flags, "--strict-mcp-config", "--mcp-config="+mcpPath)

	// ⑤ 자격증명. 홈을 옮기면 OAuth 로그인 노드가 Not logged in 이 되므로
	// 그 파일을 가짜 홈으로 복사한다. 계장 디렉터리와 함께 단계 끝에 지워진다.
	if err := copyCredentials(home); err != nil {
		return flags, err
	}
	return flags, aux
}

// credentialsName 은 OAuth 세션이 사는 파일이다 (ADR-034 §3 ①).
const credentialsName = ".credentials.json"

// copyCredentials 는 노드의 OAuth 자격증명을 가짜 홈으로 복사한다.
//
// 세 갈래다 — 원본이 없으면 정상이고(환경변수 노드 · 게이트웨이 노드는 이
// 파일에 인증을 안 둔다), 있는데 못 읽거나 못 쓰면 치명이다. 가르는 선은
// 「원본 파일에 닿았는가」다.
//
// 치명인 이유는 실패를 앞으로 당기는 것이다 — 복사가 조용히 실패하면
// 하네스가 떠서 Not logged in 으로 늦게 죽고, 그때는 임대와 예산을 이미 썼다.
//
// 홈을 못 읽는 노드(HOME 이 없는 환경)는 「없는 것」과 같이 본다. 그런 노드에
// OAuth 자격증명이 있을 수 없고, gatewayAuthFields() 가 이미 같은 판단을 한다.
func copyCredentials(home string) error {
	real, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	b, err := os.ReadFile(filepath.Join(real, ".claude", credentialsName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("cannot copy credentials: %w", err)
	}
	// 0600 이다 — 복사본이 원본보다 느슨하면 이 설계가 표면을 넓힌 것이 된다.
	if err := os.WriteFile(filepath.Join(home, credentialsName), b, 0o600); err != nil {
		return fmt.Errorf("cannot copy credentials: %w", err)
	}
	return nil
}

// Argv 는 순수 함수다 — 프로세스를 안 띄운다. 그래서 시험이 싸다.
func (claudeHarness) Argv(p AgentParams, io IOPaths) []string {
	// stream-json 으로 받는다 (decisions.md 6절 ⑮)
	//
	// 게이트 CA1 · CA4 · CA5 가 전부 system/init 줄의 mcp_servers 를 재는데,
	// --output-format json 은 그 줄을 아예 안 낸다 — 끝의 봉투 하나만 낸다.
	// --verbose 가 있어야 -p 아래서 사건이 흐른다.
	//
	// 최종 result 사건은 그대로 집힌다 — ParseClaude 가 lastJSONObject 로
	// 마지막 JSON 객체를 집는다. 크래시 경로는 그 함수의 type 검사가 막는다 (⑯).
	// logs/ 에 실을 것은 runner.go 가 따로 고른다 (⑱).
	args := []string{"-p", "--output-format", "stream-json", "--verbose"}
	if p.Model != "" {
		args = append(args, "--model", p.Model)
	}
	if p.MaxTurns > 0 {
		args = append(args, "--max-turns", strconv.Itoa(p.MaxTurns))
	}
	// 경계는 단계가 아니라 노드에 있다 (2026-08-22, ADR-042)
	//
	// 이 자리는 두 번 바뀌었고 값은 처음으로 돌아왔지만 근거가 다르다.
	//
	//	1차  bypassPermissions   ADR-013 의 ask:never 를 「모든 권한 우회」로
	//	                        잘못 번역한 것이었다 (R7 이 지적했다)
	//	2차  acceptEdits+add-dir R7 실측이 고른 값. 근거는 포함관계 였다:
	//	                        모델이 쓸 수 있는 곳 ⊆ 훅이 볼 수 있는 곳
	//	3차  bypassPermissions   그 포함관계가 살 자리가 아니었다
	//
	// 왜 3차인가 — 명령 단계가 이미 무경계다
	//
	// INVARIANTS 가 적어둔 그대로다: *명령 단계에는 파일시스템 경계가 없다.
	// argv 는 무엇이든 되므로 걸 자리가 없다. 그래서 노드는 소유자가
	// 신뢰하는 계약만 받는다.* 그런데 같은 계약이
	//
	//	run:   ["brew", "install", "colima"]     오늘 그냥 된다
	//	agent: "colima 를 설치해라"               거부된다
	//
	// 로 갈렸다. 위협이 같은데 규칙이 달랐다 — 그리고 좁은 쪽이 산 것은
	// 안전이 아니라 에이전트가 명령을 못 돌린다였다.
	//
	// enode 를 띄운 것이 곧 허가다 — 토큰을 쥐여 미디에이터를 가리킨 순간
	// 노드 주인은 이 기계가 임의의 argv 를 받는다고 선언한 것이다. 그 선언 뒤에
	// 에이전트만 가두는 것은 위협 모델을 좁히지 못하고 능력만 좁힌다.
	//
	// R7 의 실측은 틀리지 않았다 — 각 모드가 무엇을 여는지는 그대로 맞다.
	// 바뀐 것은 어느 대가를 받을 것인가이고, R7 은 명령 단계와의 불일치를
	// 저울에 안 올렸다.
	//
	// 그래서 무엇이 남아서 지키나
	//   sealInput()  $IN 을 0444·0555 로 잠근다 (collect.go).
	//                권한 모드가 아니라 파일시스템이 거는 것이라 여기 안 묶인다.
	//                오히려 이제 그것이 유일한 기계적 방어다.
	//   harnessEnv() R1 환경 화이트리스트. ENODE_TOKEN 은 여전히 안 넘어간다 —
	//                열쇠가 하나 라는 전제는 I1 이 서 있는 자리라 안 건드린다.
	//
	// 도구는 여전히 나열하지 않는다 — --allowed-tools 는 제한이 아니라
	// 자동승인 목록이고(R7 실측), 여기서는 아무 일도 안 하면서 이름만 늘린다.
	if p.Ask == "" || p.Ask == "never" {
		args = append(args, "--permission-mode", "bypassPermissions")
	}
	// --add-dir 을 남긴다 — bypassPermissions 아래서 권한상으로는 불필요하다.
	// 남기는 이유는 의도가 argv 에 남아야 하기 때문이다: 이 단계가 어디를
	// 쓸 셈이었는지가 Record 의 하네스 로그에 그대로 찍힌다. 지우면 그 사실이
	// 코드에만 있고 기록에는 없다.
	for _, d := range []string{io.Out, io.In} {
		if d != "" && d != io.Dir {
			args = append(args, "--add-dir", d)
		}
	}
	return args
}

// Decode 는 스트림을 훑는다 — 끝난 뒤의 덩어리를 받지 않는다.
//
// Parse(stdout []byte, …) 로 두면 대화가 원천봉쇄된다. claude 는
// --output-format stream-json / --input-format stream-json 으로 실시간을
// 지원하므로(실측), 막는 것은 하네스가 아니라 이쪽 추상화가 된다.
//
// 배치는 스트리밍의 퇴화형이다 — 지금은 EOF 까지 읽고 사건 하나를 낸다.
// R4 를 열 때 stream-json 으로 바꿔도 이 시그니처는 안 바뀐다.
func (claudeHarness) Decode(r io.Reader, exitCode int, emit func(Event)) HarnessResult {
	b, err := io.ReadAll(r)
	if err != nil {
		return HarnessResult{Reason: ReasonError, Message: "cannot read output: " + err.Error()}
	}
	h := ParseClaude(b, exitCode)
	emit(Event{Kind: EventFinal, Text: h.Message})
	return h
}
