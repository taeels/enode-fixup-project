package enode

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/taeels/enode/internal/contract"
)

// ★ R5③ · R6 — enode 전용 종료 훅 ★
//
// ★ 훅은 판정하지 않는다 ★
//
//	○ "계약이 요구한 X 가 아직 $OUT 에 없다"      ← 알린다
//	✗ "산출물이 부족하다. 실패다."                ← ★ 판정 ★
//
// 판정은 success_when 이 한다 (ADR-004 · I3). 훅이 판정하면 계약이 할 일을
// 플러그인이 가로챈다 — ADR-021 4번이 코드에 대해 지적한 것과 같은 실수다.
//
// ★ 실측이 정한 것 (2026-08-20) ★
//
//	훅이 막으면 claude 는 실제로 계속한다 — stop_hook_active 가 false→true 로 뒤집힌다.
//	그런데 ★ 훅의 reason 은 조언이지 명령이 아니다 ★. 무엇을 말하느냐로 갈렸다:
//
//	  훅이 ★ 계약 밖 ★ 을 시킨다   "증거.txt 를 만들어라"    → ★ 모델이 거절했다 ★
//	                                                            (계약 지시와 충돌하자
//	                                                             모델이 계약을 택했다)
//	  훅이 ★ 계약을 상기 ★ 한다    "요구된 진단 이 아직 없다" → ★ 모델이 따랐다 ★
//	                                                            프롬프트가 "파일은 만들지
//	                                                            마라" 였는데도. 7턴 $0.16.
//
//	⇒ 그래서 아래 stopReason 은 ★ 계약이 이미 요구한 것만 ★ 말한다.
//
//	⇒ 세 겹의 무게가 같지 않다:
//	     ① 선언  모델 협조 필요
//	     ② diff  ★ 모델 협조 불필요 — 진짜 안전망 ★
//	     ③ 훅    모델 협조 필요 — ★ 보조 ★
//
//	그래서 훅은 "빠진 게 없나" 라는 막연한 물음이 아니라
//	★ 계약이 요구한 이름 중 없는 것을 짚는다 ★. 그래야 조언에 값이 있다.

// StopInput 은 Claude Code 가 Stop 훅에 넘기는 것이다.
// ★ 실측으로 확인한 필드만 적는다 ★ — 짐작한 것은 안 넣는다.
type StopInput struct {
	SessionID string `json:"session_id"`
	CWD       string `json:"cwd"`
	// StopHookActive 는 ★ 두 번째 호출부터 true ★ 다. 안 보면 영원히 돈다.
	StopHookActive bool `json:"stop_hook_active"`
}

// StopOutput 은 우리가 돌려주는 것이다. block 이면 멈추는 것을 거부한다.
type StopOutput struct {
	Decision string `json:"decision,omitempty"` // "block" 이거나 비움
	Reason   string `json:"reason,omitempty"`
}

// HookArgs 는 enode 가 훅을 심을 때 명령줄에 박아 넣는 것이다.
//
// ★ 환경변수로 안 넘긴다 ★ — R1 이 하네스 환경을 화이트리스트로 좁혔는데
// 훅 때문에 구멍을 내면 그 결정이 무의미해진다. 명령줄이면 그럴 일이 없다.
type HookArgs struct {
	Out       string   // $OUT — 여기에 이름별로 낸다
	Workspace string   // 변경을 살필 곳. 비면 안 본다.
	Expect    []string // 계약이 요구한 산출물 이름
	// Stamp 는 기준 시각이 든 파일 경로다 (R5②').
	//
	// ★ 이게 있어야 훅이 빌드 산출물을 본다 ★ — git status 는 .gitignore 를
	// 지켜서 zImage 도 .ko 도 안 보여준다. 빌드 단계는 산출물이 전부
	// 무시 목록에 있어 ★ git 만 보면 아무 일도 안 한 것처럼 보인다 ★.
	Stamp string

	// Plan 은 ★ 계획 산출물의 이름 ★ 이다 — expands 단계에만 있다 (ADR-046).
	//
	// ★ 왜 훅이 계획을 보나 ★ — 문법을 프롬프트에 심어도(ADR-045) 모델이 어긴다.
	// 어긴 계획은 계약 적용 시점에 거절되는데 ★ 그때는 하네스가 이미 끝나 있다 ★.
	// 판 하나(약 $0.2~0.45)와 사람의 검토가 통째로 버려진다.
	// 훅이 짚으면 ★ 같은 세션에서 고친다 ★.
	Plan string
	// Roles 는 uses 에 쓸 수 있는 이름이다 (ADR-045). 비면 역할 검사를 건너뛴다.
	Roles []string
}

// RunStopHook 은 `enode hook stop` 의 본체다.
//
// ★ 별도 스크립트가 아니라 enode 자신인 이유 ★
//   - 셸 스크립트는 윈도우에서 안 돈다. enode 는 이미 단일 정적 바이너리다 (ADR-015).
//   - Go 로 시험할 수 있다. 셸이면 못 한다.
//   - 노드에 이미 있는 파일이라 ★ 배포할 것이 늘지 않는다 ★.
func RunStopHook(a HookArgs, in io.Reader, out io.Writer) error {
	var si StopInput
	if err := json.NewDecoder(in).Decode(&si); err != nil {
		// ★ 입력을 못 읽으면 통과시킨다 ★ — 훅이 하네스를 막아 세우면 안 된다.
		// 안전망이 정규 경로를 무너뜨리는 것이 가장 나쁘다.
		return nil
	}
	// ★ 한 번만 되묻는다 ★ — 안 보면 영원히 돈다.
	if si.StopHookActive {
		return nil
	}

	missing := missingOutputs(a.Out, a.Expect)
	if len(missing) == 0 {
		// ★ 다 냈다. 그런데 계획이면 ★ 모양 ★ 까지 본다 ★ (ADR-046).
		//
		// 계약은 이미 ★ 유효한 계획 ★ 을 요구하고 있다 — 어긴 것을 짚는 것은
		// 새 지시가 아니라 ★ 상기 ★ 다. 훅이 계약 밖을 시키면 모델이 거절한다는
		// 실측(위 표)과 어긋나지 않는다.
		if why := planProblem(a); why != "" {
			return json.NewEncoder(out).Encode(StopOutput{
				Decision: "block", Reason: why,
			})
		}
		return nil // 다 냈다. 통과.
	}

	return json.NewEncoder(out).Encode(StopOutput{
		Decision: "block",
		Reason:   stopReason(a, missing),
	})
}

// planProblem 은 계획이 계약 문법을 어겼으면 ★ 그 이유를 그대로 ★ 돌려준다.
//
// ★ 오류 문장을 번역하지 않는다 ★ — Validate() 가 내는 말을 그대로 전한다.
// 사람이 번역하다 축약하고 모순낸 것이 ADR-045 가 닫은 문제이고,
// 여기서 다시 번역하면 ★ 같은 실수를 코드가 되풀이한다 ★.
//
// ★ 못 읽으면 통과시킨다 ★ — 안전망이 정규 경로를 무너뜨리는 것이 가장 나쁘다.
func planProblem(a HookArgs) string {
	if a.Plan == "" || a.Out == "" {
		return ""
	}
	raw, err := os.ReadFile(filepath.Join(a.Out, a.Plan))
	if err != nil {
		return "" // 아직 없거나 못 읽는다 — missingOutputs 가 이미 봤다
	}
	err = contract.CheckPlan(raw, a.Roles)
	if err == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("계약 문법을 어겼다. ★ 이대로는 계획 전체가 거절된다 ★:\n\n  ")
	b.WriteString(err.Error())
	b.WriteString("\n\n")
	b.WriteString(filepath.Join(a.Out, a.Plan))
	b.WriteString(" 를 고쳐서 다시 써라. ")
	b.WriteString("위에 실린 「계약 문법」 절이 규칙 전부다.\n")
	if len(a.Roles) > 0 {
		b.WriteString("uses 에 쓸 수 있는 역할: ")
		b.WriteString(strings.Join(a.Roles, " · "))
		b.WriteString("\n")
	}
	return b.String()
}

// missingOutputs 는 요구된 이름 중 $OUT 에 없는 것이다.
func missingOutputs(outDir string, expect []string) []string {
	have := map[string]bool{}
	for _, n := range harvest(outDir) {
		have[n] = true
	}
	var missing []string
	for _, n := range expect {
		if !have[n] {
			missing = append(missing, n)
		}
	}
	sort.Strings(missing)
	return missing
}

// stopReason 은 모델에게 ★ 알리는 ★ 문장이다.
//
// ★ 새 지시를 주지 않는다 ★ — 실측에서 계약 밖을 시키자 모델이 거절했고,
// 계약을 상기시키자 따랐다(위 표). 그게 옳은 순서다. 새 일을 시키면
// 모델이 거절하거나, ★ 더 나쁘게는 계약을 어긴다 ★.
func stopReason(a HookArgs, missing []string) string {
	var b strings.Builder
	b.WriteString("계약이 요구한 산출물 중 아직 없는 것: ")
	b.WriteString(strings.Join(missing, ", "))
	b.WriteString("\n$OUT = ")
	b.WriteString(a.Out)
	b.WriteString(" 에 그 이름 그대로 파일로 쓰면 된다.\n")

	// ★ git 이 못 보는 것까지 보여준다 ★ — 빌드 산출물은 .gitignore 안에 있다.
	if a.Stamp != "" && a.Workspace != "" {
		if s, err := readStamp(a.Stamp, a.Workspace); err == nil {
			if found, total, err := changedSince(s, 2000); err == nil && total > 0 {
				b.WriteString("\n참고 — ")
				b.WriteString(summarize(found, total, 25))
			}
		}
	}
	b.WriteString("\n낼 것이 없다면 그 이유를 담아서라도 파일을 만들어라.")
	return b.String()
}

// WriteHookSettings 는 훅을 심고 ★ 하네스에 줄 플래그를 돌려준다 ★.
//
// 파일은 ★ $OUT 밖 ★ 에 둔다 — $OUT 에 두면 ④수확이 산출물로 걷어 올린다.
func WriteHookSettings(dir string, self string, a HookArgs) ([]string, error) {
	cmd := []string{self, "hook", "stop", "--out", a.Out}
	if a.Workspace != "" {
		cmd = append(cmd, "--workspace", a.Workspace)
	}
	if a.Stamp != "" {
		cmd = append(cmd, "--stamp", a.Stamp)
	}
	if len(a.Expect) > 0 {
		cmd = append(cmd, "--expect", strings.Join(a.Expect, ","))
	}
	// ★ 계획 단계에만 붙는다 ★ (ADR-046) — 다른 단계에는 검사할 계획이 없다.
	if a.Plan != "" {
		cmd = append(cmd, "--plan", a.Plan)
		if len(a.Roles) > 0 {
			cmd = append(cmd, "--roles", strings.Join(a.Roles, ","))
		}
	}

	settings := map[string]any{
		"hooks": map[string]any{
			"Stop": []any{map[string]any{
				"matcher": "",
				"hooks": []any{map[string]any{
					"type": "command", "command": shellJoin(cmd),
				}},
			}},
		},
	}
	b, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "enode-settings.json")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return nil, err
	}
	// ★ R6 — 개인 설정을 차단한다 ★
	//
	// ~/.claude 의 사람 설정이 섞이면 ★ 노드마다 결과가 달라진다 ★.
	// Run 은 재현 가능해야 하고, 그것이 Record 를 남기는 이유다 (ADR-005).
	// --setting-sources '' 로 아무것도 안 읽게 하고 우리가 준 --settings 만 쓴다.
	//
	// ★ 인증은 안 끊긴다 ★ — 자격증명은 설정이 아니라 ~/.claude 의 별도 파일이고
	// HOME 이 화이트리스트에 있다 (R1). 실측으로 확인했다.
	return []string{"--settings", path, "--setting-sources", ""}, nil
}

// shellJoin 은 훅 명령을 한 줄로 만든다. 하네스가 셸에 넘기기 때문이다.
func shellJoin(argv []string) string {
	q := make([]string, len(argv))
	for i, a := range argv {
		if strings.ContainsAny(a, " \t\"'$`\\") {
			q[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
		} else {
			q[i] = a
		}
	}
	return strings.Join(q, " ")
}
