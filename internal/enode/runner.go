package enode

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
)

// ★ R3 — 프로세스를 띄우는 코드는 여기 하나다 ★
//
// 어댑터마다 exec 을 두면 R1 의 환경 화이트리스트를 ★ N 번 지켜야 하고,
// 그게 구멍이 나는 방식이다 ★. 타임아웃·임대 워치독·cwd·계장도 마찬가지다.
// 그래서 Harness 는 ★ 순수 함수 ★ 만 내놓고 실행은 안 한다.
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
// ★ 인자를 늘리지 않고 구조체로 묶는 이유 ★ — 여기 자랄 것이 남아 있다
// (스킬 · MCP · 샌드박스 강도). 시그니처를 매번 고치면 어댑터가 따라 흔들린다.
type Job struct {
	Params AgentParams
	Prompt string
	IO     IOPaths
	Expect []string          // 계약이 요구한 산출물 이름 — 훅이 이걸 짚는다
	Inject map[string]string // Credentials 가 돌려준 것 (R1)
	Emit   func(Event)       // 스트림 사건. nil 이면 버린다.
}

// runHarness 는 ②기동이다. ★ 유일한 exec 지점 ★.
func runHarness(ctx context.Context, h Harness, bin string, j Job) ([]byte, HarnessResult) {
	args := h.Argv(j.Params, j.IO)

	// ★ R6 — 계장 ★. 파일은 $OUT ★ 밖 ★ 에 둔다. 안에 두면 ④수확이 걷어 올린다.
	if tmp, err := os.MkdirTemp("", "enode-inst-*"); err == nil {
		defer os.RemoveAll(tmp) //nolint:errcheck
		self, _ := os.Executable()
		if self != "" {
			flags, err := h.Instrument(tmp, self, HookArgs{
				Out: j.IO.Out, Workspace: j.IO.Dir, Expect: j.Expect,
			})
			if err == nil {
				args = append(args, flags...)
			}
			// ★ 실패해도 기동은 한다 ★ — 계장은 보조이고,
			// 진짜 안전망은 워크스페이스 diff 다 (모델 협조가 필요 없다).
		}
	}
	env := harnessEnv(h.Env(), map[string]string{"OUT": j.IO.Out, "IN": j.IO.In}, j.Inject)

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = j.IO.Dir
	cmd.Stdin = strings.NewReader(j.Prompt)
	// ★ 화이트리스트로 조립된 것만 넘어간다 ★ — os.Environ() 을 얹지 않는다.
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()

	code := -1
	if cmd.ProcessState != nil {
		code = cmd.ProcessState.ExitCode()
	}
	emit := j.Emit
	if emit == nil {
		emit = func(Event) {}
	}
	res := h.Decode(bytes.NewReader(stdout.Bytes()), code, emit)
	// ★ 버전은 runner 가 채운다 ★ — 어댑터마다 잊을 수 있는 일을 한 곳에 둔다.
	// Probe 가 실패해도 실행은 이미 됐으므로 결과를 버리지 않는다.
	//
	// ★ 캐시하지 않는다 ★ — enode 는 며칠씩 살아 있어서, 그 사이 하네스가
	// 업그레이드되면 캐시된 버전은 거짓이 된다. 그런데 우리가 버전을 남기는
	// 이유가 바로 그 드리프트를 잡기 위해서다. 캐시는 잡으려는 것을 숨긴다.
	// 비용은 단계당 프로세스 하나(~50ms)이고, 단계는 실측 74초였다 — 0.07% 다.
	if v, err := h.Probe(ctx, bin); err == nil {
		res.Version = v
	}

	// ★ 종료코드보다 ctx 가 우선이다 ★ — 임대가 끝나 죽인 것을
	// 하네스 오류로 적으면 Record 가 거짓을 남긴다.
	if ctx.Err() != nil {
		res = HarnessResult{Reason: ReasonTimeout, Message: "임대 만료 또는 중단"}
	} else if err != nil && res.Reason == ReasonError && res.Message == "" {
		res.Message = err.Error()
	}
	// 로그는 stdout + stderr 를 합쳐 원문 그대로 (ADR-005 의 logs/).
	return append(stdout.Bytes(), stderr.Bytes()...), res
}
