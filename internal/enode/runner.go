package enode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// R3 — 프로세스를 띄우는 코드는 여기 하나다
//
// 어댑터마다 exec 을 두면 R1 의 환경 화이트리스트를 N 번 지켜야 하고,
// 그게 구멍이 나는 방식이다. 타임아웃·임대 워치독·cwd·계장도 마찬가지다.
// 그래서 Harness 는 순수 함수만 내놓고 실행은 안 한다.
//
//	           ┌──────────────────────────────────────┐
//	runner ────┤ 환경 화이트리스트 (R1)                │
//	 (공용)    │ 개인 설정 차단 (R6)                   │
//	           │ 타임아웃 · cwd · stdin               │
//	           └──────────────┬───────────────────────┘
//	                          │ Argv() / Decode() / Instrument() 만 물어본다
//	              ┌───────────┴───────────┐
//	         claude.go    codex.go    opencode.go

// IOPaths 는 ①사출·④수확이 쓰는 경로들이다 (ADR-013).
type IOPaths struct {
	Dir string // cwd — 워크스페이스 (ADR-017)
	In  string // $IN  — 앞 단계 산출물
	Out string // $OUT — 이 단계가 낼 곳
}

// Job 은 한 번의 기동에 필요한 전부다.
//
// 인자를 늘리지 않고 구조체로 묶는 이유 — 여기 자랄 것이 남아 있다
// (스킬 · MCP · 샌드박스 강도). 시그니처를 매번 고치면 어댑터가 따라 흔들린다.
type Job struct {
	Params AgentParams
	Prompt string
	IO     IOPaths
	Expect []string // 계약이 요구한 산출물 이름 — 훅이 이걸 짚는다
	Stamp  Stamp    // git 이 못 보는 변경의 기준 시각 (R5②')
	// Plan 은 계획 산출물의 이름이다 — expands 단계에만 채워진다 (ADR-046).
	// 훅이 그 파일을 열어 계약 문법을 어겼는지 본다.
	Plan string
	// Roles 는 uses 에 쓸 수 있는 이름이다 (ADR-045).
	Roles  []string
	Inject map[string]string // Credentials 가 돌려준 것 (R1)
	Emit   func(Event)       // 스트림 사건. nil 이면 버린다.
	// Transcript 는 하네스 원문 stdout 을 tee 할 곳이다 (decisions §6.2 Q2).
	// nil 이면 안 흘린다 — 설정이 없는 시험은 링 파일을 안 만든다. 트랜스크립트
	// 링(internal/enode/transcript.go)이 여기 앉아 제어판이 도는 동안 읽는다.
	Transcript io.Writer

	// NodeMCP 는 노드 소유자가 enode.yaml 에 선언한 MCP 서버다 (U4).
	//
	// Job 이 들고 오는 이유 — resolveComponents 가 정하는 함수이고 무엇을
	// 들고 올지는 부르는 쪽이 안다. 여기서 안 실으면 노드 선언이 조용히 안 실린다.
	NodeMCP map[string]MCPServer

	// WorkspaceMCP 는 워크스페이스 .mcp.json 의 mcpServers 를 원문 그대로 담는다.
	//
	// 우리 형식이 아니다 — 저장소가 하네스에게 쓴 것이고 우리 어휘에 없는
	// 키(type · headers)가 있다. 그래서 map 인 채로 나르고 허용목록에 그대로 간다.
	WorkspaceMCP map[string]map[string]any

	// WorkspaceMCPErr 는 그 파일을 읽거나 푸는 데 실패한 이유다.
	//
	// 왜 오류가 필드인가 — 등급을 정하는 것이 정책이고 정책은
	// resolveComponents 에 있다. 읽는 쪽이 등급까지 정하면 그 거절이
	// HarnessResult 를 안 타고 나가서 단계 오류의 꼴이 경로마다 달라진다.
	WorkspaceMCPErr error

	// Log 는 Components.Notes 가 나갈 자리다 (U4).
	//
	// logs/ 에는 안 싣는다 — 그 파일은 허용목록이고 첫 줄이 system/init 이어야
	// 한다 (게이트 CA1 · CA4 · CA5 가 head -1 로 읽는다). nil 이면 안 찍고
	// 그때도 판정은 같다.
	Log *slog.Logger
}

// runHarness 는 ②기동이다. 유일한 exec 지점.
//
// 치명이 전부 exec 앞에 모인다 (components.md 3절)
//
//	①  계장 디렉터리          못 만들면 단계 실패
//	②  resolveComponents     무엇을 열지 여기서 정해진다.  파일을 안 만진다.
//	                        Notes 를 찍는 것이 그 바로 뒤다 — 거절로 끝날 때도 찍는다
//	③  Argv
//	④  Instrument            errAux 만 삼킨다.  삼킬 때도 얻은 플래그는 붙인다
//	⑤  Fixed(tmp)
//	⑥  exec
//	⑦  Decode · Version · 자백
//	⑧  logs/ 에 실을 것을 고른다
//	⑨  defer 가 계장 디렉터리를 지운다
//
// 앞에 모으는 이유는 격리가 안 선 채로 하네스가 뜨는 경로를 없애는 것이다.
// 뒤에서 알면 그때는 이미 개인 설정과 계정 커넥터를 본 프로세스가 돈 뒤다.
func runHarness(ctx context.Context, h Harness, bin string, j Job) ([]byte, HarnessResult) {
	// ① R6 — 계장. 파일은 $OUT 밖에 둔다. 안에 두면 ④수확이 걷어 올린다.
	//
	// 오늘 이 자리는 if 블록의 조건이었다 — 실패하면 계장을 통째로 건너뛰고
	// 그대로 기동했다. 그 경로로 가면 가짜 홈이 없고 개인 설정과 계정 커넥터가
	// 다 보이는 하네스가 돈다. 디스크가 찬 노드에서 격리가 조용히 풀리는 길이다.
	tmp, err := os.MkdirTemp("", "enode-inst-*")
	if err != nil {
		return nil, HarnessResult{Reason: ReasonError,
			Message: "cannot create instrumentation directory: " + err.Error()}
	}
	// 보존 스위치를 안 둔다 — 복사한 자격증명이 계장 디렉터리와 함께 단계 끝에
	// 지워지는 것이 보안 요구다. 대가는 사람이 손으로 띄우는 눈 검증이 실물이
	// 아니라 복제본을 잰다는 것이고, 그 한계를 게이트가 명시로 안다.
	defer os.RemoveAll(tmp) //nolint:errcheck

	// ② 무엇을 열지는 여기서 정해진다. 실패하면 하네스를 안 띄운다 —
	// 요청한 것을 조용히 빼고 도는 것이 「없음이 실패보다 나쁘다」의 그 자리다.
	c, err := resolveComponents(j)
	// 오류 검사보다 앞이다 — 왜 실패했는지를 아는 데 필요한 사실이 실패와
	// 함께 사라지면 안 된다. 겹침과 빠짐이 여기로 나간다.
	if j.Log != nil {
		for _, n := range c.Notes {
			j.Log.Info("mcp allowlist note", "note", n)
		}
	}
	if err != nil {
		return nil, HarnessResult{Reason: ReasonError, Message: err.Error()}
	}

	// ③
	args := h.Argv(j.Params, j.IO)

	// ④ 계장. self 가 비어도 부른다 — 오늘은 os.Executable() 하나가 허용목록까지
	// 떨어뜨렸다. 훅 블록만 그것에 달린다 (WriteHookSettings).
	self, _ := os.Executable()
	a := HookArgs{Out: j.IO.Out, Workspace: j.IO.Dir, Expect: j.Expect,
		Plan: j.Plan, Roles: j.Roles}
	// 훅은 별도 프로세스라 기준 시각을 파일로 넘긴다.
	// 워크스페이스 밖(계장 임시 폴더)에 둔다 — 안에 두면 자기가 걷힌다.
	if !j.Stamp.At.IsZero() {
		p := filepath.Join(tmp, "stamp")
		if writeStamp(p, j.Stamp) == nil {
			a.Stamp = p
		}
	}
	flags, err := h.Instrument(tmp, self, a, c)
	// 오류에도 이미 얻은 플래그를 붙인다 — 보조 실패 하나가 격리의 겹을
	// 함께 떨어뜨리면 안 된다.
	args = append(args, flags...)
	if err != nil && !errors.Is(err, errAux) {
		// 기본이 치명이다. 감싸지 않은 오류는 전부 단계를 죽인다 —
		// 빠뜨림이 열리는 쪽으로 틀리면 허용목록이 안 쓰인 채 exit 0 으로
		// 성공이 봉인된다.
		return nil, HarnessResult{Reason: ReasonError, Message: err.Error()}
	}
	// 보조 실패는 삼킨다 — 훅은 모델 협조가 필요한 셋째 겹이고
	// 진짜 안전망은 워크스페이스 diff 다.

	// ⑤ 우리가 못 박는 값 — 구조적인 것(OUT·IN)과 하네스가 정하는 것(Fixed)을
	// 한 자리에서 합친다. Fixed 는 재현성용이고 OUT·IN 과 이름이 겹칠 일이 없다.
	fixed := map[string]string{"OUT": j.IO.Out, "IN": j.IO.In}
	for k, v := range h.Fixed(tmp) {
		fixed[k] = v
	}
	env := harnessEnv(h.Env(), fixed, j.Inject)

	cmd := child(exec.CommandContext(ctx, bin, args...))
	cmd.Dir = j.IO.Dir
	cmd.Stdin = strings.NewReader(j.Prompt)
	// 화이트리스트로 조립된 것만 넘어간다 — os.Environ() 을 얹지 않는다.
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	// ⑥ 하네스 단계의 링 tee 를 안 건다 (decisions.md 6절 ⑲)
	//
	// 예전에는 여기서 j.Transcript 로 원문을 흘렸다. stream-json 아래서는 그
	// 원문에 도구 입력과 도구 결과가 통째로 실리고, 이 tee 는 아래 선별보다
	// 앞이라 걸러야 할 것이 그대로 링으로 나간다. 링은 노드 디스크에 파일로
	// 남으므로 logs/ 만 걸러서는 안 닫힌다.
	//
	// Job.Transcript 필드는 남긴다 — 명령 단계의 tee 가 지금도 그것을 쓰고,
	// 짝 팩이 사건 스트림과 함께 노출 정책을 정해 이 자리를 되살린다.
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	code := -1
	if cmd.ProcessState != nil {
		code = cmd.ProcessState.ExitCode()
	}
	emit := j.Emit
	if emit == nil {
		emit = func(Event) {}
	}
	// ⑦
	res := h.Decode(bytes.NewReader(stdout.Bytes()), code, emit)
	// 버전은 runner 가 채운다 — 어댑터마다 잊을 수 있는 일을 한 곳에 둔다.
	// Version 이 실패해도 실행은 이미 됐으므로 결과를 버리지 않는다.
	//
	// 캐시하지 않는다 — enode 는 며칠씩 살아 있어서, 그 사이 하네스가
	// 업그레이드되면 캐시된 버전은 거짓이 된다. 그런데 우리가 버전을 남기는
	// 이유가 바로 그 드리프트를 잡기 위해서다. 캐시는 잡으려는 것을 숨긴다.
	// 비용은 단계당 프로세스 하나(~50ms)이고, 단계는 실측 74초였다 — 0.07% 다.
	//
	// 그 「하나」가 예전에는 둘이었다 — Probe 가 버전을 알아내기 전에
	// 「쓸 수 있는가」를 먼저 확인했고, 이 자리에는 그 확인이 필요 없다.
	// 실행은 이미 끝났고 결과가 손에 있다.
	if v, err := h.Version(ctx, bin); err == nil {
		res.Version = v
	}

	// 자백을 읽는다 (ADR-038) — 하네스는 정상 종료했지만 모델이
	// "못 하겠다" 를 남겼을 수 있다. 어댑터는 그것을 못 본다(봉투만 읽는다).
	//
	// 자백은 검증하지 않는다 — 성공 주장과 비대칭이다. 못 했다고 말해서
	// 얻을 것이 없으므로 거짓말할 유인이 없다.
	// 그리고 판정하지도 않는다 — HarnessResult 는 Record 에 남고
	// 계약 판정에는 안 들어간다(harness.go). produced·changed 가 판정한다.
	if why, ok := readCannot(j.IO.Out); ok && res.Reason == ReasonOK {
		res.Reason = ReasonCannot
		if res.Message == "" {
			res.Message = why
		}
	}

	// 종료코드보다 ctx 가 우선이다 — 임대가 끝나 죽인 것을
	// 하네스 오류로 적으면 Record 가 거짓을 남긴다.
	if ctx.Err() != nil {
		res = HarnessResult{Reason: ReasonTimeout, Message: "lease expired or aborted"}
	} else if err != nil && res.Reason == ReasonError && res.Message == "" {
		res.Message = err.Error()
	}
	// ⑧ logs/ 는 허용목록이다 — 아는 것만 남는다 (ADR-005 의 logs/).
	return selectLogs(stdout.Bytes(), stderr.Bytes()), res
}

// logs/ 의 허용목록 — 아는 것만 남기고 나머지는 걷는다 (decisions.md 6절 ⑱)
//
// 왜 지우는 쪽이 아니라 남기는 쪽인가 — 실측이 그 이유를 보였다. assistant
// 사건은 도구 입력을 message.content[] 의 tool_use.input 에 싣고 최상위
// wire_tool_inputs 에 한 번 더 싣는다. user 사건은 도구 결과를
// message.content[] 와 최상위 tool_use_result 에 싣는다. 블록만 지우는
// 구현은 안 닫힌다. 그리고 하네스가 필드를 늘리면 지우는 쪽은 열리는 쪽으로
// 틀린다 — 이 저장소가 env.go 의 환경변수에서 이미 고른 규율이다 (R1).
//
//	전문      첫 system/init · 마지막 type=="result" · stderr.  셋뿐이다
//	껍데기    type 이 문자열인 그 밖의 모든 사건
//	안 남는다  JSON 객체가 아닌 줄 · type 이 문자열이 아닌 줄.  세기만 한다
//
// 경로에 예외가 없다 — 크래시 · 임대 만료 · 플래그 오류에서도 같다. 한 번
// 「봉투가 안 나오는 경로는 원문 그대로」로 예외를 뒀다가 걷었다: 임대 만료는
// 긴 에이전트 단계의 일상이라 그 예외가 곧 정규 경로였다.
//
// 어댑터가 아니라 여기서 한다 — 이 팩은 Decode 를 못 만진다(짝 팩의 것이다).
// claude 와이어 포맷 지식이 runner.go 에 처음 올라오는 자리이고, 두 번째
// 하네스가 오는 날 이 줄이 인터페이스로 올라간다.
func selectLogs(stdout, stderr []byte) []byte {
	lines := splitLines(stdout)

	// 두 자리를 먼저 찾는다 — 마지막 result 를 고르려면 끝까지 봐야 하고
	// 조립은 순서대로 해야 한다. 한 번에 못 한다.
	//
	// lastJSONObject 로 대신 뽑으면 안 된다 — 크래시 때 마지막 완결 객체가
	// assistant 사건이라 그 본문이 전문으로 실린다. 봉투를 고르는 자리가 둘이고
	// 규칙이 다르다 (ParseClaude 는 줄 단위가 아니다).
	initAt, resultAt := -1, -1
	for i, ln := range lines {
		obj, typ, ok := parseEventLine(ln)
		if !ok {
			continue
		}
		if initAt < 0 && typ == "system" && eventString(obj, "subtype") == "init" {
			initAt = i
		}
		if typ == "result" {
			resultAt = i // 여럿이면 마지막 하나다
		}
	}

	var out bytes.Buffer
	var events, elided int
	for i, ln := range lines {
		if i == initAt || i == resultAt {
			out.Write(ln)
			out.WriteByte('\n')
			continue
		}
		events++
		elided += len(ln) + 1 // 개행을 포함한다
		obj, typ, ok := parseEventLine(ln)
		if !ok {
			continue // 모르는 줄은 세기만 한다
		}
		out.Write(eventShell(obj, typ))
		out.WriteByte('\n')
	}
	// 걷었음을 한 줄로 남긴다. 걷은 것이 0 이어도 쓴다 — 그래야 읽는 사람이
	// 이 파일이 걸러진 것임을 안다.
	//
	// stdout 이 통째로 비면 안 쓴다. 그때 이 줄을 쓰면 그것이 첫 줄이 되어
	// 「init 이 없으면 stderr 가 첫 줄」이 깨진다.
	if len(lines) > 0 {
		out.Write(elidedMarker(events, elided))
		out.WriteByte('\n')
	}
	// stderr 는 원문 그대로 뒤에 붙는다 — 오늘과 같다.
	out.Write(stderr)
	return out.Bytes()
}

// splitLines 는 stdout 을 줄로 가른다. 마지막 빈 조각은 버린다.
func splitLines(b []byte) [][]byte {
	if len(b) == 0 {
		return nil
	}
	lines := bytes.Split(b, []byte("\n"))
	if n := len(lines); n > 0 && len(lines[n-1]) == 0 {
		lines = lines[:n-1]
	}
	return lines
}

// parseEventLine 은 줄 하나를 사건으로 읽는다.
//
// 구조체로 한 번에 안 받는다 — 모르는 필드의 모양 하나가 줄 전체를
// 떨어뜨리기 때문이다. type 을 먼저 집고 아는 키만 따로 읽으면 모르는 것은
// 안 실리고 아는 것은 남는다. 허용목록의 규율이 여기서도 같다.
func parseEventLine(ln []byte) (map[string]json.RawMessage, string, bool) {
	var obj map[string]json.RawMessage
	if json.Unmarshal(ln, &obj) != nil {
		return nil, "", false
	}
	typ := eventString(obj, "type")
	if typ == "" {
		// type 이 없거나 문자열이 아니다 — 무엇인지 모르므로 안 싣는다.
		return nil, "", false
	}
	return obj, typ, true
}

// logShell 은 사건 하나가 logs/ 에 남기는 전부다.
//
// 필드를 짓는 쪽이 허용목록이다 — 원본에서 지우는 것이 아니라 새 객체를
// 지으므로 하네스가 필드를 늘려도 안 샌다. 본문은 어느 사건에서도 안 남는다:
// assistant 의 text 도 thinking 도 도구 결과도 같다.
//
// 시각을 안 넣는다 — exec 이 끝난 뒤 한 번에 선별하므로 사건마다의 시각을
// 못 찍고, 넣으면 모든 줄이 같은 값이라 정보량이 0 이다.
type logShell struct {
	Type    string         `json:"type"`
	Subtype string         `json:"subtype,omitempty"`
	Tools   []string       `json:"tools,omitempty"`
	OK      *bool          `json:"ok,omitempty"`
	Tokens  map[string]int `json:"tokens,omitempty"`
}

// elidedMark 는 걷었음을 표시하는 줄이다.
//
// type 에 점을 넣는다 — 실측한 하네스의 type 은 전부 홑단어라
// (system · assistant · user · result · rate_limit_event) 부딪칠 수 없다.
type elidedMark struct {
	Type   string `json:"type"`
	Events int    `json:"events"`
	Bytes  int    `json:"bytes"`
}

func elidedMarker(events, size int) []byte {
	b, err := json.Marshal(elidedMark{Type: "enode.elided", Events: events, Bytes: size})
	if err != nil {
		return nil
	}
	return b
}

// eventShell 은 사건 하나를 껍데기 한 줄로 짓는다.
func eventShell(obj map[string]json.RawMessage, typ string) []byte {
	sh := logShell{Type: typ}
	if typ == "system" {
		sh.Subtype = eventString(obj, "subtype")
	}
	var msg struct {
		Content []map[string]json.RawMessage `json:"content"`
		Usage   map[string]json.RawMessage   `json:"usage"`
	}
	if json.Unmarshal(obj["message"], &msg) == nil {
		ok, sawResult := true, false
		for _, blk := range msg.Content {
			switch eventString(blk, "type") {
			case "tool_use":
				// 배열로 둔다 — 실측은 사건마다 블록 하나였지만 그것이
				// 보증은 아니다. 하나일 때도 배열이면 뒤에 모양이 안 갈린다.
				sh.Tools = append(sh.Tools, eventString(blk, "name"))
			case "tool_result":
				sawResult = true
				// 성공하면 is_error 키가 아예 없다 (실측).
				if v, has := eventBool(blk, "is_error"); has && v {
					ok = false
				}
			}
		}
		if sawResult {
			// 있을 때만 쓴다 — 도구를 안 부른 사건에 ok: true 를 박으면
			// 「성공한 도구가 있었다」로 읽힌다. 없음과 참을 가른다.
			sh.OK = &ok
		}
		sh.Tokens = usageTokens(msg.Usage)
	}
	// system/thinking_tokens 의 estimated_tokens — usage 밖의 정수 하나다.
	// 같은 종류의 값이라(본문이 없는 정수 하나) 싣는 쪽으로 정했고,
	// 범위를 넓힌 자리라 이름으로 적는다. 빼려면 이 블록을 지운다.
	if n, has := eventInt(obj, "estimated_tokens"); has {
		if sh.Tokens == nil {
			sh.Tokens = map[string]int{}
		}
		sh.Tokens["thinking"] = n
	}
	b, err := json.Marshal(sh)
	if err != nil {
		return nil
	}
	return b
}

// usageTokens 는 usage 에서 정수 넷만 집는다 (답 2 = C).
//
// 통째로 못 옮긴다 — 실측이 usage 안에 문자열 둘(service_tier ·
// inference_geo)과 객체 하나(cache_creation)를 보였다. 그래서 이름으로 집고,
// 정수가 아니면 그 키를 건너뛴다. 모르는 것은 안 싣는다.
func usageTokens(usage map[string]json.RawMessage) map[string]int {
	if len(usage) == 0 {
		return nil
	}
	names := [...]struct{ out, in string }{
		{"in", "input_tokens"},
		{"out", "output_tokens"},
		{"cache_write", "cache_creation_input_tokens"},
		{"cache_read", "cache_read_input_tokens"},
	}
	var tk map[string]int
	for _, n := range names {
		v, has := eventInt(usage, n.in)
		if !has {
			continue
		}
		if tk == nil {
			tk = map[string]int{}
		}
		tk[n.out] = v
	}
	return tk
}

// eventString · eventBool · eventInt 는 아는 키 하나를 아는 모양으로만 읽는다.
// 모양이 다르면 없는 것으로 본다 — 판정을 짐작으로 메우지 않는다.
func eventString(obj map[string]json.RawMessage, key string) string {
	var s string
	if json.Unmarshal(obj[key], &s) != nil {
		return ""
	}
	return s
}

func eventBool(obj map[string]json.RawMessage, key string) (bool, bool) {
	var v bool
	if json.Unmarshal(obj[key], &v) != nil {
		return false, false
	}
	return v, true
}

func eventInt(obj map[string]json.RawMessage, key string) (int, bool) {
	var n int
	if json.Unmarshal(obj[key], &n) != nil {
		return 0, false
	}
	return n, true
}

// readCannot 은 $OUT 의 자백 파일을 읽는다 (ADR-038).
//
// 상한을 둔다 — 이유를 적으라고 했는데 로그를 통째로 붓는 경우가 있다.
// 전문은 어차피 $OUT 에 남아 봉인된다.
func readCannot(out string) (string, bool) {
	b, err := os.ReadFile(filepath.Join(out, cannotName))
	if err != nil {
		return "", false
	}
	why := strings.TrimSpace(string(b))
	if why == "" {
		// 빈 파일도 자백이다 — 이유를 안 적었을 뿐 못 했다고 말한 것이다.
		return "no reason given", true
	}
	return trimTo(why, 2000), true
}
