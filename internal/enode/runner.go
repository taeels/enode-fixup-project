package enode

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/taeels/enode/internal/transcript"
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
//	①    계장 디렉터리        못 만들면 단계 실패
//	①.5  openPack             팩이 있을 때만 $IN 을 연다.  가장자리다
//	②    resolveComponents    무엇을 열지 여기서 정해진다.  파일을 안 만진다.
//	                          Notes 를 찍는 것이 그 바로 뒤다 — 거절로 끝날 때도 찍는다
//	③    Argv
//	④    Instrument           errAux 만 삼킨다.  삼킬 때도 얻은 플래그는 붙인다
//	⑤    Fixed(tmp)
//	⑥    exec
//	⑦    Decode · Version · 자백
//	⑧    logs/ 선별 + 무엇을 물렸나를 기록한다
//	⑨    defer 가 계장 디렉터리를 지운다
//
// ①.5 가 ② 앞인 이유 — 검증 실패가 곧 단계 실패이고 그 판정을 하는 것이 ② 다.
// 파일을 여는 것은 가장자리이고 등급과 문구를 정하는 것은 정책이다.
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

	// ①.5 팩을 연다. agent.pack 이 비면 파일을 아예 안 만진다.
	pk := openPack(j)

	// ② 무엇을 열지는 여기서 정해진다. 실패하면 하네스를 안 띄운다 —
	// 요청한 것을 조용히 빼고 도는 것이 「없음이 실패보다 나쁘다」의 그 자리다.
	c, err := resolveComponents(j, pk)
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
	// ⑧ 무엇을 물렸나를 봉인에 남긴다 (ADR-005 성질 4).
	//
	// 여기가 exec 을 실제로 지난 유일한 자리다. ②나 ④에서 채우면 exec 앞에
	// 죽은 단계의 봉인에도 값이 남아 「실린 것」이라는 필드의 뜻이 흐려진다 —
	// 안 실린 팩의 sha256 이 harness.pack 에 있으면 봉인을 읽는 사람이
	// 실렸다고 읽는다. 타임아웃은 채운다: 하네스는 이미 떴다.
	//
	// 요청한 것이 아니라 실린 것이다. 팩이 실었으나 요청 안 한 이름은
	// Notes 에만 있고 이 필드에는 없다.
	if len(c.Servers) > 0 {
		res.MCP = entryNames(c.Servers)
	}
	if c.Pack != nil {
		res.Pack = c.Pack.SHA256
	}
	// logs/ 는 허용목록이다 — 아는 것만 남는다 (ADR-005 의 logs/).
	return selectLogs(stdout.Bytes(), stderr.Bytes()), res
}

// openPack 은 $IN 의 팩을 연다 (R1 · R2).
//
// 파일을 여는 유일한 자리다. claim.go 가 아니라 여기인 이유는 claim.go 가 이
// 유닛의 파일 행렬 밖이기 때문이고, 그래서 가장자리가 둘로 갈린다 —
// 워크스페이스 선언은 claim.go 가 열고 팩은 여기가 연다. 그 비대칭을 이 주석이
// 진다. 같은 파일의 readCannot 이 이미 같은 종류의 자리다.
//
// 등급과 문구를 안 정한다. Err 를 그대로 담아 resolveComponents 가 정한다 —
// Job.WorkspaceMCPErr 와 같은 규율이고, 읽는 쪽이 등급까지 정하면 그 거절이
// HarnessResult 를 안 타고 나가서 단계 오류의 꼴이 경로마다 달라진다.
//
// 없는 파일이 값이 아니라 오류인 것이 ADR-058 의 예외다. 「없는 입력은 값이다」는
// in.from 의 규칙이고 그 근거는 dispatch 로 안 간 가지를 가리킬 수 있다는
// 것이었다. 팩은 다르다 — agent.pack 은 이 단계가 그것으로 돌겠다고 적은
// 이름이고, 없는 채로 돌면 스킬 없이 도는 단계가 exit 0 으로 봉인된다.
func openPack(j Job) packInput {
	name := j.Params.Pack
	if name == "" {
		return packInput{}
	}
	// Base 로 좁힌다 — blob 이름 공간은 평평하지만 여는 쪽이 그것을 가정하지
	// 않는다. 가정이 깨지는 날 $IN 밖을 여는 것보다 파일을 못 찾는 것이 낫다.
	f, err := os.Open(filepath.Join(j.IO.In, filepath.Base(name)))
	if err != nil {
		return packInput{Err: fmt.Errorf("pack blob %s was not produced by this run", name)}
	}
	defer f.Close() //nolint:errcheck
	p, notes, err := readPack(name, f, defaultPackLimits)
	return packInput{Pack: p, Notes: notes, Err: err}
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
	lines := transcript.SplitLines(stdout)

	// 두 자리를 먼저 찾는다 — 마지막 result 를 고르려면 끝까지 봐야 하고
	// 조립은 순서대로 해야 한다. 한 번에 못 한다.
	//
	// lastJSONObject 로 대신 뽑으면 안 된다 — 크래시 때 마지막 완결 객체가
	// assistant 사건이라 그 본문이 전문으로 실린다. 봉투를 고르는 자리가 둘이고
	// 규칙이 다르다 (ParseClaude 는 줄 단위가 아니다).
	initAt, resultAt := -1, -1
	for i, ln := range lines {
		obj, typ, ok := transcript.ParseLine(ln)
		if !ok {
			continue
		}
		if initAt < 0 && typ == "system" && transcript.String(obj, "subtype") == "init" {
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
		obj, typ, ok := transcript.ParseLine(ln)
		if !ok {
			continue // 모르는 줄은 세기만 한다
		}
		out.Write(transcript.Shell(obj, typ))
		out.WriteByte('\n')
	}
	// 걷었음을 한 줄로 남긴다. 걷은 것이 0 이어도 쓴다 — 그래야 읽는 사람이
	// 이 파일이 걸러진 것임을 안다.
	//
	// stdout 이 통째로 비면 안 쓴다. 그때 이 줄을 쓰면 그것이 첫 줄이 되어
	// 「init 이 없으면 stderr 가 첫 줄」이 깨진다.
	if len(lines) > 0 {
		out.Write(transcript.ElidedMarker(events, elided))
		out.WriteByte('\n')
	}
	// stderr 는 원문 그대로 뒤에 붙는다 — 오늘과 같다.
	out.Write(stderr)
	return out.Bytes()
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
