package enode

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
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

// 아무것도 못 하면 아무것도 광고하지 않는다.
func TestDetectEmpty(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if caps := Detect(Local{}, log); len(caps) != 0 {
		// claude 가 설치된 기계에서는 harness 가 잡힐 수 있다 — 그건 정상이다.
		for _, c := range caps {
			for k := range c.Attrs {
				if k != "harness" {
					t.Fatalf("빈 설정인데 %s 를 광고했다: %+v", k, caps)
				}
			}
		}
	}
}
