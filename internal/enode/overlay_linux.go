//go:build linux

package enode

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// overlayFP 는 이 노드가 오버레이 워크스페이스를 세울 수 있는가를 잰다 (ADR-070 §5.7).
//
// 왜 노드가 재는가 — 마운트 권한은 그 환경이 이미 정해놨고 우리가 나중에 못
// 바꾼다. 컨테이너의 capability 상한은 생성 시점에 고정이라 안에서 무엇을
// 설치해도 못 넘는다. 실측에서 그것이 무엇에 달렸는지까지 나왔다: 기본 도커는
// 못 하고, --cap-add SYS_ADMIN 으로 뜬 같은 컨테이너는 한다. 우리가 정하는
// 값이 아니므로 「할 수 있는가는 노드가 판단한다」(ADR-017 결정 3) 가 그대로 걸린다.
//
// 깔려 있는지가 아니라 되는지를 본다 — 실제로 한 번 마운트하고 층이 갈리는지까지
// 확인한다 (ADR-059 「설치된 것과 쓸 수 있는 것은 다르다」).
//
// ADR-068 의 Detector 가 자기 시계로 이것을 다시 부른다. 그래서 호스트에서
// 그 옵션을 되돌리면 그날 광고가 알아서 달라진다 — 막을 방법이 없는 것을
// 감당하는 방법이 이것뿐이다.
type overlayFP struct{}

func (overlayFP) Kind() string { return "overlay" }

// 사다리 넷. 위에서부터 시도하고 처음 되는 것을 답으로 낸다.
const (
	overlayKernel = "kernel" // mount(2) 를 직접. CAP_SYS_ADMIN 이 있을 때
	overlayUserns = "userns" // unshare 로 연 user namespace 안에서. 특권이 0 이다
	overlayFuse   = "fuse"   // fuse-overlayfs. /dev/fuse 가 이미 있어야 한다
)

func (overlayFP) Probe(ctx context.Context, l Local, log *slog.Logger) (map[string]string, error) {
	// 워크스페이스가 없으면 잴 자리가 없다.
	//
	// 임시 디렉터리에 재면 안 된다 — 컨테이너의 /var/tmp 는 도커 자기 overlay2 라
	// 윗 층을 거기 못 둔다. 실제로는 되는데 안 된다고 답하게 된다. 실측에서 밟았다.
	// 워크스페이스 옆에 재야 진짜 워크스페이스가 앉을 파일시스템을 본다.
	if l.Workspace == "" {
		return nil, nil
	}

	// 워크스페이스 안이 아니라 옆에 만든다 — 안에 만들면 clean -df 가 지우고
	// 관찰 범위에도 들어간다.
	root, err := os.MkdirTemp(filepath.Dir(l.Workspace), ".enode-overlay-probe-")
	if err != nil {
		return nil, fmt.Errorf("cannot make a probe directory next to the workspace: %w", err)
	}
	defer os.RemoveAll(root)

	for _, rung := range []struct {
		name string
		try  func(context.Context, string) error
	}{
		{overlayKernel, mountOverlayDirectly},
		{overlayUserns, mountOverlayInUserNamespace},
		{overlayFuse, mountOverlayWithFuse},
	} {
		err := rung.try(ctx, root)
		if err == nil {
			return map[string]string{"overlay": rung.name}, nil
		}
		log.Debug("this node cannot build an overlay that way", "how", rung.name, "err", err)
	}

	// 못 하면 속성을 뺀다. 값으로 none 을 싣지 않는다 —
	// 광고에서 빠지는 것이 곧 "못 한다" 다 (ADR-012). 그리고 none 을 실으면
	// hasCapability 가 그것을 능력으로 세어, 아무것도 할 줄 모르는 기계가
	// 광고하게 된다.
	log.Info("this node cannot build an overlay workspace; contracts that need one will not pick it",
		"workspace", l.Workspace)
	return nil, nil
}

// layout 은 탐침 한 판이 쓰는 네 디렉터리를 만든다.
func layout(root, tag string) (lower, upper, work, merged string, err error) {
	base := filepath.Join(root, tag)
	lower = filepath.Join(base, "lower")
	upper = filepath.Join(base, "upper")
	work = filepath.Join(base, "work")
	merged = filepath.Join(base, "merged")
	for _, d := range []string{lower, upper, work, merged} {
		if err = os.MkdirAll(d, 0o755); err != nil {
			return "", "", "", "", err
		}
	}
	err = os.WriteFile(filepath.Join(lower, "probe"), []byte("baseline"), 0o644)
	return lower, upper, work, merged, err
}

func overlayOpts(lower, upper, work string) string {
	return fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lower, upper, work)
}

// splits 는 층이 실제로 갈리는지 본다 — 마운트가 섰다는 것만으로는 모자라다.
//
// 윗 층에 쓰고, 아래 층이 안 변했고, 쓴 것이 윗 층에 있어야 한다. 셋 중
// 하나라도 아니면 그 위에서 도는 Run 이 원본 워크스페이스를 고치게 된다.
func splits(lower, upper, merged string) error {
	if err := os.WriteFile(filepath.Join(merged, "probe"), []byte("changed"), 0o644); err != nil {
		return fmt.Errorf("the merged view is not writable: %w", err)
	}
	b, err := os.ReadFile(filepath.Join(lower, "probe"))
	if err != nil || string(b) != "baseline" {
		return errors.New("the write reached the lower layer; this would modify the original workspace")
	}
	if _, err := os.Stat(filepath.Join(upper, "probe")); err != nil {
		return errors.New("the write did not land in the upper layer")
	}
	return nil
}

// mountOverlayDirectly 는 CAP_SYS_ADMIN 이 있는 자리다 — VM · 베어메탈 · 특권 컨테이너.
func mountOverlayDirectly(_ context.Context, root string) error {
	lower, upper, work, merged, err := layout(root, overlayKernel)
	if err != nil {
		return err
	}
	if err := unix.Mount("overlay", merged, "overlay", 0, overlayOpts(lower, upper, work)); err != nil {
		return err
	}
	defer unix.Unmount(merged, unix.MNT_DETACH)
	return splits(lower, upper, merged)
}

// mountOverlayInUserNamespace 는 컨테이너의 길이다. 특권이 0 이다.
//
// unshare 로 재는 이유는 실행 경로가 같은 것을 쓰기 때문이다 — 마운트는 새
// 네임스페이스 안에서 일어나야 하고 그 안에서 하네스가 떠야 하므로, 재는 방법과
// 실제로 하는 방법이 같아야 둘이 안 어긋난다. unshare 가 없으면 이 칸은 못 하는
// 것이 맞다 — 실행 시점에도 못 한다.
func mountOverlayInUserNamespace(ctx context.Context, root string) error {
	lower, upper, work, merged, err := layout(root, overlayUserns)
	if err != nil {
		return err
	}
	script := fmt.Sprintf("mount -t overlay overlay -o %s %s || exit 11",
		overlayOpts(lower, upper, work), merged)
	// 검증도 네임스페이스 안에서 한다 — 마운트가 그 안에만 보이기 때문이다.
	script += fmt.Sprintf(`
printf changed > %s/probe || exit 12
[ "$(cat %s/probe)" = baseline ] || exit 13
[ -f %s/probe ] || exit 14`, merged, lower, upper)

	cmd := child(exec.CommandContext(ctx, "unshare",
		"--user", "--map-root-user", "--mount", "sh", "-c", script))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("unshare: %w: %s", err, out)
	}
	return nil
}

// mountOverlayWithFuse 는 /dev/fuse 가 이미 있을 때만 선다.
//
// 장치를 새로 만들려면 컨테이너를 다시 띄워야 하는데 그것이 금지된 일이다.
// 실측에서는 네 구성 모두 이 칸이 비어 있었다.
func mountOverlayWithFuse(ctx context.Context, root string) error {
	if _, err := os.Stat("/dev/fuse"); err != nil {
		return fmt.Errorf("no /dev/fuse: %w", err)
	}
	bin, err := exec.LookPath("fuse-overlayfs")
	if err != nil {
		return err
	}
	lower, upper, work, merged, err := layout(root, overlayFuse)
	if err != nil {
		return err
	}
	cmd := child(exec.CommandContext(ctx, bin, "-o", overlayOpts(lower, upper, work), merged))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("fuse-overlayfs: %w: %s", err, out)
	}
	defer func() {
		if err := child(exec.Command("fusermount", "-u", merged)).Run(); err != nil {
			_ = unix.Unmount(merged, unix.MNT_DETACH)
		}
	}()
	return splits(lower, upper, merged)
}
