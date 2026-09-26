package enode

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/taeels/enode/internal/contract"
)

// 소유자 정책 (ADR-063).
//
// 정본은 노드의 파일이다 — 그 기계에 접속했다는 것이 소유의 증거이므로(§3) 정책도
// 거기 산다. 중앙은 광고에 실려 온 복사본을 둘 뿐이고 판정하지 않는다. 데몬은
// 광고 직전마다 이 파일을 읽어 싣는다. 캐시가 없는 이유 — 값이 파일에 있다.

// Policy 는 소유자가 정책 파일에 적는 것이다. 데몬은 Drain 만 본다 —
// PanelToken 은 제어판(internal/panel)이 LAN 노출을 켤 때 읽는 값이라 여기
// 자리만 두고 데몬은 안 쓴다. policyReader.Read 가 Drain 만 접으므로 이 키를
// 더해도 데몬 동작은 안 바뀐다.
type Policy struct {
	Drain      string `yaml:"drain"`
	PanelToken string `yaml:"panel_token"`
}

// PolicyPath 는 설정 파일 옆의 정책 파일이다 — <dir>/<stem>.policy.yaml.
//
// 설정 경로가 곧 노드의 신원이므로(ADR-017) 정책도 그 옆에 하나씩이다.
// 한 기계에 노드가 여럿이면 각각 자기 것을 본다.
func PolicyPath(configPath string) string {
	return strings.TrimSuffix(configPath, filepath.Ext(configPath)) + ".policy.yaml"
}

// ReadPolicyFile 은 제어판이 정책 파일을 읽는 자리다 (drain 과 panel_token).
//
// 데몬의 policyReader 는 Drain 만 접어 어휘로 좁히지만, 제어판은 파일에 적힌
// 그대로(panel_token 포함)를 봐야 한다. 파일이 없으면 안 걸린 것이다 — 제로값.
func ReadPolicyFile(configPath string) (Policy, error) {
	b, err := os.ReadFile(PolicyPath(configPath))
	if errors.Is(err, os.ErrNotExist) {
		return Policy{}, nil
	}
	if err != nil {
		return Policy{}, err
	}
	var p Policy
	if err := yaml.Unmarshal(b, &p); err != nil {
		return Policy{}, err
	}
	return p, nil
}

// WritePolicyFile 은 제어판의 drain 토글이 정책 파일을 쓰는 자리다.
//
// tmp 에 쓰고 rename 으로 바꾼다 — 데몬이 광고 직전에 반쯤 쓴 파일을 읽지 않게.
// 권한은 소유자만 쓸 수 있게 0600 (유닉스). 윈도우는 그 디렉터리의 상속 ACL 이라
// 모드 비트를 안 건다(checkPerms 가 윈도우를 안 보는 것과 같은 이유).
// 부르는 쪽이 기존 Policy 를 읽어 Drain 만 바꿔 넘기면 panel_token 이 보존된다.
func WritePolicyFile(configPath string, p Policy) error {
	b, err := yaml.Marshal(p)
	if err != nil {
		return err
	}
	path := PolicyPath(configPath)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
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

// drain 의 출처 (ADR-063 §4 · trash 유닛 · 완료 조건 1).
//
// 광고의 policy.drain 에는 출처를 합친 값 하나만 실린다 — 새 광고 어휘도 Mediator 변경도
// 없다 (FR-4). 누가 건 drain 인지는 노드 쪽(상태 파일과 제어판)에만 있다. 노드가 스스로 건
// drain 도 소유자 drain 과 같은 길을 지난다 — Mediator 가 받아 적고 응답으로 돌려주면 Worker 가
// 새 일을 안 집는다.
const (
	DrainOwner = "owner" // 소유자가 정책 파일에 건 것.  여기서 풀 수 있다
	DrainDisk  = "disk"  // 워크스페이스의 여유가 min_free_gb 아래라 노드가 스스로 건 것
)

// DrainSource 는 drain 의 출처 하나다.
type DrainSource struct {
	Kind   string `yaml:"kind" json:"kind"`     // owner | disk.  bake 는 lower-state 유닛이 더한다
	Mode   string `yaml:"mode" json:"mode"`     // graceful | at-boundary
	Detail string `yaml:"detail" json:"detail"` // 영어 한 줄 (business-rules.md 6절)
	Owner  bool   `yaml:"owner" json:"owner"`   // 소유자가 정책 파일에서 풀 수 있나
}

// DrainStatus 는 이번 광고에 실은 drain 과 그 출처다.
type DrainStatus struct {
	Effective string        `yaml:"effective" json:"effective"` // "" | graceful | at-boundary
	Sources   []DrainSource `yaml:"sources" json:"sources"`
	At        time.Time     `yaml:"at" json:"at"` // 이 값을 정한 광고 주기의 시각
}

// OwnerDrain 은 정책 파일의 값에서 소유자 출처를 짓는다. 값이 없으면 출처가 없다.
// 제어판도 이것을 쓴다 — 데몬이 돌지 않을 때 정책 파일에서 출처 하나를 그린다.
func OwnerDrain(mode string) (DrainSource, bool) {
	if mode == contract.DrainNone {
		return DrainSource{}, false
	}
	return DrainSource{Kind: DrainOwner, Mode: mode, Detail: "set in the policy file", Owner: true}, true
}

// combineDrain 은 출처 목록에서 센 쪽 하나를 고른다 — at-boundary > graceful > 없음
// (business-rules.md 6.3). 출처가 늘어도(bake) 이 규칙이 그대로다.
func combineDrain(sources []DrainSource) string {
	effective := contract.DrainNone
	for _, src := range sources {
		switch src.Mode {
		case contract.DrainAtBoundary:
			return contract.DrainAtBoundary
		case contract.DrainGraceful:
			effective = contract.DrainGraceful
		}
	}
	return effective
}

// diskDrain 은 여유 부족 출처를 정한다 (business-rules.md 6.2). held 는 앞 광고에서 걸었나다 —
// 거는 선과 푸는 선이 다르다. 걸 때는 min 아래 · 풀 때는 min + 1 GB 이상이다. GB 는 2^30
// 바이트의 몫(내림)이다. min 이 0 이하면 걸지 않는다.
func diskDrain(freeBytes uint64, minFreeGB int, held bool) (DrainSource, bool) {
	if minFreeGB <= 0 {
		return DrainSource{}, false
	}
	freeGB, minGB := freeBytes/(1<<30), uint64(minFreeGB)
	src := DrainSource{Kind: DrainDisk, Mode: contract.DrainGraceful}
	switch {
	case freeGB < minGB:
		src.Detail = fmt.Sprintf("free %d GB < min %d GB", freeGB, minGB)
		return src, true
	case held && freeGB < minGB+1:
		src.Detail = fmt.Sprintf("free %d GB < min %d GB + 1 GB to lift", freeGB, minGB)
		return src, true
	}
	return DrainSource{}, false
}
