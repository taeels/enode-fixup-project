package enode

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	execenv "github.com/taeels/enode/internal/environment"
	"github.com/taeels/enode/internal/lower"
)

// ExecutionRuntimeVerifier 는 env check/apply 가 제품과 같은 adapter 로 도는 smoke 다. native 는 추가 격리 경계가
// 없으므로 no-op 이고 runc-overlay 만 실제로 namespace 를 연다. Verify 는 linux 와 그 밖 파일에 따로 있다.
//
// 준비도 점검 셋도 낸다 — environment.FactSource (lower-state 유닛 · 답 8). type 을 여기 한 번만 두는 것은 Notice
// 칸을 두 파일에 두 벌로 두지 않으려는 것이다.
type ExecutionRuntimeVerifier struct {
	// Notice 는 smoke 가 합치기를 기다린다는 줄을 쓰는 자리다 — enodectl env 는 표준 오류, 데몬 기동은 노드 로그
	// (LogNotice). nil 이면 안 쓴다.
	Notice io.Writer
}

// smokePoll 은 smoke 가 공유를 다시 보는 간격이고, smokeNotice 는 기다린다는 줄을 다시 쓰는 간격이다
// (business-rules.md 8.3 · 13절). 시험만 줄인다.
var (
	smokePoll   = time.Second
	smokeNotice = 30 * time.Second
)

// Facts 는 준비도 점검 셋이다 (business-rules.md 12절). runc-overlay 일 때만 낸다. Source 는 /runtime 이다 —
// binding Fact 의 선례. 설정이 쓸 수 없는 자리를 가리키면 invalid, profile 밖에 남은 상태가 막으면
// external-blocked 로 옮긴다. 아무것도 만들지 않는다.
func (ExecutionRuntimeVerifier) Facts(_ context.Context, doc execenv.Document, b execenv.Binding) []execenv.Fact {
	if doc.Profile.Runtime.Driver != "runc-overlay" {
		return nil
	}
	lowers, homeErr := LowersDir()
	var facts []execenv.Fact
	for _, f := range lower.Check(lowers, b.Workspace, b.Scratch, os.Getuid()) {
		facts = append(facts, factOf(f))
	}
	// internal/lower 는 home 을 모른다 — 뿌리를 못 주면 신원 줄을 여기서 낸다 (계획 4절 ⑯)
	if homeErr != nil {
		facts = append(facts, execenv.Fact{Name: "lower.identity", Source: "/runtime",
			Required: "the recorded identity of this lower", Observed: "cannot find the home directory: " + homeErr.Error(),
			State:       execenv.StateExternalBlocked,
			Remediation: "set HOME for the node user; the lower state lives under $HOME/.local/state/enode/lowers"})
	}
	return facts
}

// factOf 는 lower 의 판정을 environment 의 Fact 로 옮긴다 (답 8).
func factOf(f lower.Finding) execenv.Fact {
	state := execenv.StateReady
	switch {
	case f.OK:
	case f.Cause == lower.CauseBinding:
		state = execenv.StateInvalid
	default:
		state = execenv.StateExternalBlocked
	}
	return execenv.Fact{Name: f.Name, Source: "/runtime", Required: f.Required, Observed: f.Observed,
		State: state, Remediation: f.Remediation}
}

// smokeLock 은 smoke 가 세션을 열기 전에 lower 공유를 쥔다 (business-rules.md 8.3 · 계획 4절 ⑰). 합치는 중이면
// ctx 까지 기다리고, 기다리기 시작할 때와 30초마다 notice 에 한 줄을 쓴다. 자리가 없으면 잠그지 않는다 — 이
// lower 에 굽기가 한 번도 없었고 합치기는 자리 없이 일어나지 않는다. lower.lock 이 없어도 잠그지 않는다.
// 돌려준 것은 부르는 쪽이 세션을 닫은 뒤 놓는다 (nil 이어도 된다).
//
// home 을 못 찾으면 smoke 오류다 — 합치기가 없다는 것을 보일 수 없다.
func smokeLock(ctx context.Context, workspace string, notice io.Writer) (*lower.Shared, error) {
	lowers, err := LowersDir()
	if err != nil {
		return nil, fmt.Errorf("cannot find the home directory: %w", err)
	}
	root, err := lower.ReadRoot(workspace)
	if err != nil {
		return nil, err
	}
	dir, err := lower.Peek(lowers, root)
	if err != nil || dir == nil {
		return nil, err
	}
	var noted time.Time
	return dir.WaitShared(ctx, smokePoll, func(st lower.State) {
		if notice == nil || !noted.IsZero() && time.Since(noted) < smokeNotice {
			return
		}
		noted = time.Now()
		run, node := "unknown", "unknown"
		if st.Owner != nil {
			run, node = st.Owner.Run, st.Owner.Node
		}
		fmt.Fprintf(notice, "env check: waiting for the lower merge to finish (run %s on node %s, since %s)\n",
			run, node, st.Since.UTC().Format(time.RFC3339))
	})
}

// LogNotice 는 줄마다 노드 로그의 Info 한 줄이 되는 io.Writer 다 — 데몬 기동의 점검이 smoke 의 Notice 로 쓴다.
// 끝나지 않은 줄은 다음 쓰기까지 들고 있다.
func LogNotice(log *slog.Logger) io.Writer { return &logNotice{log: log} }

type logNotice struct {
	mu   sync.Mutex
	log  *slog.Logger
	rest []byte
}

func (w *logNotice) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.rest = append(w.rest, p...)
	for {
		i := bytes.IndexByte(w.rest, '\n')
		if i < 0 {
			return len(p), nil
		}
		if line := strings.TrimSpace(string(w.rest[:i])); line != "" {
			w.log.Info(line)
		}
		w.rest = w.rest[i+1:]
	}
}
