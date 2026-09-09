package enode

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/taeels/enode/internal/contract"
)

// 소유자 정책 (ADR-063).
//
// 정본은 노드의 파일이다 — 그 기계에 접속했다는 것이 소유의 증거이므로(§3) 정책도
// 거기 산다. 중앙은 광고에 실려 온 복사본을 둘 뿐이고 판정하지 않는다. 데몬은
// 광고 직전마다 이 파일을 읽어 싣는다. 캐시가 없는 이유 — 값이 파일에 있다.

// Policy 는 소유자가 정책 파일에 적는 것이다. 키 하나다 — 모르는 키는 무시한다
// (제어판의 panel_token 이 뒤에 같은 파일에 앉는다).
type Policy struct {
	Drain string `yaml:"drain"`
}

// PolicyPath 는 설정 파일 옆의 정책 파일이다 — <dir>/<stem>.policy.yaml.
//
// 설정 경로가 곧 노드의 신원이므로(ADR-017) 정책도 그 옆에 하나씩이다.
// 한 기계에 노드가 여럿이면 각각 자기 것을 본다.
func PolicyPath(configPath string) string {
	return strings.TrimSuffix(configPath, filepath.Ext(configPath)) + ".policy.yaml"
}

// policyReader 는 정책 파일을 읽고 어휘 안으로 접는다.
//
// 틀린 값으로 노드를 세우지 않는다 — 광고는 하트비트를 겸한다 (ADR-016).
// 틀리면 안 걸린 것으로 접고 로그에 남기되, 같은 원인이 이어지면 다시 안 찍는다:
// 광고가 5초마다 돌면 같은 줄이 5초마다 쌓이고 그러면 기록이 사실보다 커진다.
// 값 원문을 적는 것은 소유자의 기계 · 소유자의 파일이라서다 — 적어야 고친다.
type policyReader struct {
	Path string
	Log  *slog.Logger

	lastWarn    string
	permsWarned bool
}

func (r *policyReader) log() *slog.Logger {
	if r.Log == nil {
		return slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	return r.Log
}

// Read 는 지금 파일이 말하는 정책이다. 없으면 안 걸린 것이다.
func (r *policyReader) Read() contract.Policy {
	b, err := os.ReadFile(r.Path)
	if errors.Is(err, os.ErrNotExist) {
		r.warn("")
		return contract.Policy{}
	}
	if err != nil {
		r.warn("cannot read the policy file; treating the node as not draining: " + err.Error())
		return contract.Policy{}
	}
	r.checkPerms()
	var p Policy
	if err := yaml.Unmarshal(b, &p); err != nil {
		r.warn("cannot parse the policy file; treating the node as not draining: " + err.Error())
		return contract.Policy{}
	}
	switch p.Drain {
	case contract.DrainNone, contract.DrainGraceful, contract.DrainAtBoundary:
	default:
		r.warn("unknown drain policy folded to none: " + p.Drain)
		return contract.Policy{}
	}
	r.warn("")
	return contract.Policy{Drain: p.Drain}
}

// warn 은 원인이 바뀔 때만 찍는다. 빈 원인은 「지금은 괜찮다」이고 기억만 되돌린다.
func (r *policyReader) warn(reason string) {
	if reason == r.lastWarn {
		return
	}
	r.lastWarn = reason
	if reason != "" {
		r.log().Warn(reason, "path", r.Path)
	}
}

// checkPerms 는 남이 쓸 수 있는 정책 파일을 알린다 — 그 사람이 이 노드를 뺄 수 있다.
//
// 막지 않는다. drain 은 파괴가 아니라 회수이고(도는 단계는 끝까지 · 산출은 Record),
// 소유자가 권한을 모르는 채 「걸었는데 안 걸린다」가 되는 쪽이 더 나쁘다.
// 윈도우는 안 본다 — 권한이 모드 비트가 아니라 디렉터리의 ACL 이다 (decisions §6.3).
func (r *policyReader) checkPerms() {
	if runtime.GOOS == "windows" {
		return
	}
	fi, err := os.Stat(r.Path)
	if err != nil {
		return
	}
	loose := fi.Mode().Perm()&0o022 != 0
	if loose && !r.permsWarned {
		r.log().Warn("the policy file is writable by others; anyone on this machine can drain this node",
			"path", r.Path, "mode", fi.Mode().Perm().String())
	}
	r.permsWarned = loose
}
