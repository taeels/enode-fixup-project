package enode

import (
	"github.com/taeels/enode/internal/contract"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// ★ ADR-017 결정 1 — 정규화를 안 하면 조용히 매칭이 실패하고 이유가 안 보인다 ★
func TestCanonicalRepoID(t *testing.T) {
	const want = "gerrit.corp/kernel/linux"
	same := []string{
		"ssh://git@gerrit.corp:29418/kernel/linux",
		"ssh://gerrit.corp/kernel/linux.git",
		"https://gerrit.corp/kernel/linux.git",
		"https://gerrit.corp/kernel/linux",
		"http://user:pass@gerrit.corp/kernel/linux.git",
		"git@gerrit.corp:kernel/linux.git",
		"git://gerrit.corp/kernel/linux",
		"https://GERRIT.CORP/kernel/linux/",
	}
	for _, in := range same {
		if got := CanonicalRepoID(in); got != want {
			t.Errorf("%s → %q, 기대 %q", in, got, want)
		}
	}

	// 다른 것은 달라야 한다 — 과하게 뭉개면 엉뚱한 노드에 배정된다.
	diff := map[string]string{
		"https://gerrit.corp/kernel/linux-next": "gerrit.corp/kernel/linux-next",
		"https://other.corp/kernel/linux":       "other.corp/kernel/linux",
		"https://gerrit.corp/Kernel/Linux":      "gerrit.corp/Kernel/Linux", // 경로는 대소문자 구분
	}
	for in, want := range diff {
		if got := CanonicalRepoID(in); got != want {
			t.Errorf("%s → %q, 기대 %q", in, got, want)
		}
	}
	if CanonicalRepoID("") != "" {
		t.Error("빈 입력은 빈 출력이어야 한다")
	}
}

// ★ ADR-015 §2 — 설정 파일이 곧 신원이다 ★
func TestIdentity(t *testing.T) {
	const email, host = "taeels@gmail.com", "thinkpad"

	a := deriveFrom(email, host, "/etc/enode/ws-a.yaml")
	b := deriveFrom(email, host, "/etc/enode/ws-b.yaml")
	if a.NodeID == b.NodeID {
		t.Fatal("★ 설정이 다른데 신원이 같다 ★ — 두 enode 가 같은 자원을 광고하게 된다")
	}
	// 재시작에 안정적이어야 한다 (무작위 UUID 를 기각한 이유)
	if again := deriveFrom(email, host, "/etc/enode/ws-a.yaml"); again.NodeID != a.NodeID {
		t.Fatal("★ 같은 입력인데 신원이 바뀐다 ★ — 재시작마다 유령 노드가 쌓인다")
	}
	// 기계가 다르면 달라야 한다
	if c := deriveFrom(email, "mbp", "/etc/enode/ws-a.yaml"); c.NodeID == a.NodeID {
		t.Fatal("기계가 다른데 신원이 같다")
	}
	// 사람이 다르면 달라야 한다 (공용 빌드 서버)
	if d := deriveFrom("other@corp.com", host, "/etc/enode/ws-a.yaml"); d.NodeID == a.NodeID {
		t.Fatal("사람이 다른데 신원이 같다")
	}
	if len(a.NodeID) != 12 {
		t.Fatalf("node_id 길이 %d", len(a.NodeID))
	}
}

func TestDeriveLabel(t *testing.T) {
	cases := map[string]string{
		"/etc/enode/local.yaml": "taeels@thinkpad",      // 기본 이름이면 안 붙인다
		"/etc/enode/ws-a.yaml":  "taeels@thinkpad:ws-a", // 워크스페이스가 보인다
		"/etc/enode/board.yml":  "taeels@thinkpad:board",
	}
	for path, want := range cases {
		if got := deriveLabel("taeels@gmail.com", "thinkpad", path); got != want {
			t.Errorf("%s → %q, 기대 %q", path, got, want)
		}
	}
}

// ★ ADR-015 §2 — 중복 실행은 로컬에서 막는다 ★
// Mediator 는 재시작과 중복을 구분할 정보가 없다.
func TestLockRejectsSecond(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "local.yaml")
	if err := os.WriteFile(cfg, []byte("mediator: x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := Acquire(cfg)
	if err != nil {
		t.Fatalf("첫 잠금이 실패했다: %v", err)
	}
	if _, err := Acquire(cfg); err == nil {
		t.Fatal("★ 같은 설정으로 두 번 띄워졌다 ★ — 같은 자원을 둘 다 광고하게 된다")
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
	// 풀면 다시 잡을 수 있어야 한다 — 재시작이 막히면 안 된다
	again, err := Acquire(cfg)
	if err != nil {
		t.Fatalf("★ 재시작이 막혔다 ★: %v", err)
	}
	again.Release()
}

// ★ ADR-017 결정 3 — 여유공간은 매칭 조건이 아니라 광고 조건이다 ★
// "할 수 있는가" 는 노드가 판단하고, 못 하면 그 항목을 빼고 광고한다.
func TestDetectDropsBuildWhenDiskLow(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ws := t.TempDir()

	l := Local{Workspace: ws, Arch: "armv7", MinFreeGB: 1}
	caps := Detect(l, log)
	if len(caps) == 0 || caps[0].Attrs["arch"] != "armv7" {
		t.Fatalf("여유가 있으면 빌드 능력이 있어야 한다: %+v", caps)
	}

	// 임계값을 현실적으로 불가능하게 올린다 → 빠져야 한다
	l.MinFreeGB = 1 << 20 // 1 PB
	caps = Detect(l, log)
	for _, c := range caps {
		if _, ok := c.Attrs["arch"]; ok {
			t.Fatal("★ 디스크가 모자란데 빌드 능력을 광고했다 ★")
		}
	}
}

// ★ 오케스트레이션 노드는 agent.reason 도 arch 도 광고하지 않는다 ★ (ADR-022 §5)
//
// ① agent.reason 을 안 내는 이유 — 어휘를 나눈 목적이 ★ 배제 ★ 다.
//
//	속성은 부분집합 매칭이라 평범한 Run 이 이 노드를 잡아가고,
//	임대 키가 (노드)라 그 순간 오케스트레이션이 막힌다.
//
// ② arch 를 안 내는 이유 — 이 노드는 Mediator 머신에 놓인다.
//
//	"명령 단계에는 파일시스템 경계가 없다"(INVARIANTS)이므로 거기서 명령 단계가
//	돌면 ★ DB·아티팩트·토큰에 무경계 argv 가 닿는다 ★.
func TestDetect_오케스트레이션은_배제되게_광고한다(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ws := t.TempDir()

	l := Local{Workspace: ws, Arch: "armv7", Orchestration: true}
	caps := Detect(l, log)
	if len(caps) == 0 {
		t.Skip("하네스도 저장소도 없어 광고할 것이 없다")
	}
	for _, c := range caps {
		if c.Capability != contract.CapabilityOrchestration {
			t.Fatalf("★ %q 를 광고했다 — 평범한 Run 이 잡아간다 ★: %+v",
				c.Capability, caps)
		}
		if _, ok := c.Attrs["arch"]; ok {
			t.Fatalf("★ arch 를 광고했다 — 명령 단계가 Mediator 머신에서 돈다 ★: %+v", caps)
		}
	}

	// ★ 음성 대조 ★ — 같은 설정에서 플래그만 끄면 arch 가 돌아온다
	l.Orchestration = false
	caps = Detect(l, log)
	if len(caps) == 0 || caps[0].Capability != contract.CapabilityAgentReason ||
		caps[0].Attrs["arch"] != "armv7" {
		t.Fatalf("플래그를 껐는데 평범한 노드가 아니다: %+v", caps)
	}
}

// 아무것도 못 하면 아무것도 광고하지 않는다.
//
// ★ 기계 사실은 능력이 아니다 ★ (ADR-055) — os · host_arch 는 어느 기계에나
// 있으므로 설정이 비어도 실린다. 그것들만 남으면 광고 자체를 안 한다.
func TestDetectEmpty(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if caps := Detect(Local{}, log); len(caps) != 0 {
		// claude 가 설치된 기계에서는 harness 가 잡힐 수 있다 — 그건 정상이다.
		for _, c := range caps {
			for k := range c.Attrs {
				switch k {
				case "harness", "os", "host_arch":
				default:
					t.Fatalf("빈 설정인데 %s 를 광고했다: %+v", k, caps)
				}
			}
		}
	}
}

// ★ 기계 사실만으로는 광고하지 않는다 ★ (ADR-055)
//
// os · host_arch 를 무조건 싣게 되면서 "아무것도 못 하는 노드" 가 능력을
// 가진 것처럼 보일 수 있다. ★ 광고가 곧 능력이다 ★ (ADR-012) — 그 뜻을 지킨다.
func Test기계사실만_있으면_광고하지_않는다(t *testing.T) {
	if hasCapability(map[string]string{"os": "linux", "host_arch": "amd64",
		"ws": "/w"}) {
		t.Fatal("★ 기계 사실만 있는데 능력이 있다고 했다 ★")
	}
	if !hasCapability(map[string]string{"os": "linux", "harness": "claude"}) {
		t.Fatal("★ 하네스가 있는데 능력이 없다고 했다 ★")
	}
	if !hasCapability(map[string]string{"os": "linux", "arch": "arm64"}) {
		t.Fatal("★ 빌드 능력이 있는데 없다고 했다 ★")
	}
}

// ★ 광고가 os · host_arch · ws 를 싣는다 ★ (ADR-055)
//
// 실측(vm-scratch-1..5): 계획이 매 판 uname · sw_vers 를 돌려 이것을 알아냈고,
// 그 답을 보려면 ★ 판이 하나 더 필요했다 ★.
func Test광고에_기계_사실이_실린다(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ws := t.TempDir()
	caps := Detect(Local{Workspace: ws, Arch: "arm64", MinFreeGB: 0}, log)
	if len(caps) == 0 {
		t.Fatal("★ arch 가 있는데 광고가 비었다 ★")
	}
	a := caps[0].Attrs
	if a["os"] != runtime.GOOS {
		t.Fatalf("★ os 가 %q 다 ★ — %q 여야 한다", a["os"], runtime.GOOS)
	}
	if a["host_arch"] != runtime.GOARCH {
		t.Fatalf("★ host_arch 가 %q 다 ★", a["host_arch"])
	}
	if a["ws"] != ws {
		t.Fatalf("★ ws 가 %q 다 ★ — %q 여야 한다", a["ws"], ws)
	}
	// ★ arch 는 빌드 대상이고 host_arch 는 이 기계다 ★ — 섞이면 안 된다.
	if a["arch"] != "arm64" {
		t.Fatalf("★ 빌드 대상 arch 가 사라졌다 ★: %+v", a)
	}
}

// ★ 이름표가 광고에 실린다 ★ (docs/elastic-nodes.md §3.2)
//
// 탄력 노드가 자기 몫의 Run 만 잡으려면 구별할 것이 있어야 한다.
// ADR-012 가 "속성 어휘는 창발한다" 로 열어둔 자리다.
func Test이름표가_광고에_실린다(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	caps := Detect(Local{
		Workspace: t.TempDir(),
		Arch:      "arm64",
		Labels:    map[string]string{"issue": "PROJ-42", "pool": "cloud-builders"},
	}, log)
	if len(caps) == 0 {
		t.Fatal("★ 광고가 비었다 ★")
	}
	a := caps[0].Attrs
	if a["issue"] != "PROJ-42" || a["pool"] != "cloud-builders" {
		t.Fatalf("★ 이름표가 안 실렸다 ★: %+v", a)
	}

	// ★ 탐지한 것을 못 덮는다 ★ — 사람이 적은 것이 기계가 본 것을 이기면
	// 둘이 어긋났을 때 조용히 틀린다.
	caps = Detect(Local{
		Workspace: t.TempDir(),
		Arch:      "arm64",
		Labels:    map[string]string{"os": "그럴듯한거짓말", "arch": "x86"},
	}, log)
	a = caps[0].Attrs
	if a["os"] == "그럴듯한거짓말" {
		t.Fatal("★ 이름표가 탐지한 os 를 덮었다 ★")
	}
	if a["arch"] != "arm64" {
		t.Fatalf("★ 이름표가 arch 를 덮었다 ★: %s", a["arch"])
	}
}
