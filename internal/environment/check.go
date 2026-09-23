package environment

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type State string

const (
	StateReady           State = "ready"
	StateInstallable     State = "installable"
	StateAdminRequired   State = "admin-required"
	StateExternalBlocked State = "external-blocked"
	StateUnsupported     State = "unsupported"
	StateInvalid         State = "invalid"
	StateStale           State = "stale"
)

type Binding struct {
	Store     string
	Scratch   string
	Workspace string
	SSHDir    string
}

type Fact struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Required    string `json:"required,omitempty"`
	Observed    string `json:"observed,omitempty"`
	State       State  `json:"state"`
	Remediation string `json:"remediation,omitempty"`
}

type Operation struct {
	Kind            string     `json:"kind"`
	Source          string     `json:"source"`
	Mutates         bool       `json:"mutates"`
	Packages        []string   `json:"packages,omitempty"`
	Components      []string   `json:"components,omitempty"`
	Release         string     `json:"release,omitempty"`
	Arch            string     `json:"arch,omitempty"`
	Mirror          string     `json:"mirror,omitempty"`
	Locale          string     `json:"locale,omitempty"`
	User            RootFSUser `json:"user,omitempty"`
	WorkspaceTarget string     `json:"workspace_target,omitempty"`
	Executables     []string   `json:"executables,omitempty"`
	RuntimeDriver   string     `json:"runtime_driver,omitempty"`
}

type Report struct {
	State         State       `json:"state"`
	ProfileSHA256 string      `json:"profile_sha256"`
	Facts         []Fact      `json:"facts"`
	Operations    []Operation `json:"operations"`
}

type Inspector interface {
	GOOS() string
	RuntimeDriverAvailable(name string) bool
	CurrentUser() (string, error)
	LookPath(name string) bool
	PackageInstalled(ctx context.Context, name string) (bool, error)
	SubordinateRange(path, username string) (int, error)
	SmokeUserNS(ctx context.Context) error
}

// RuntimeVerifier는 준비된 rootfs와 profile binding을 실제 product runtime으로
// 한 번 여닫는다. package environment가 StepRuntime 구현을 알지 않도록 이 좁은
// seam만 두며, check와 apply가 같은 smoke를 사용한다.
type RuntimeVerifier interface {
	Verify(context.Context, Document, Binding, string, Manifest) error
}

type OSInspector struct{}

func (OSInspector) GOOS() string { return runtime.GOOS }

// RuntimeDriverAvailable은 이 product build가 실제 StepRuntime adapter를
// 연결했는지를 답한다. runc binary가 설치됐다는 사실과 product wiring을 섞지
// 않는다.
func (OSInspector) RuntimeDriverAvailable(name string) bool { return runtimeDriverAvailable(name) }

func (OSInspector) CurrentUser() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.Username, nil
}

func (OSInspector) LookPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func (OSInspector) PackageInstalled(ctx context.Context, name string) (bool, error) {
	cmd := exec.CommandContext(ctx, "dpkg-query", "-W", "-f=${Status}", name)
	b, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return false, nil
		}
		return false, err
	}
	return strings.TrimSpace(string(b)) == "install ok installed", nil
}

func (OSInspector) SubordinateRange(path, username string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	best := 0
	s := bufio.NewScanner(f)
	for s.Scan() {
		parts := strings.Split(s.Text(), ":")
		if len(parts) != 3 || parts[0] != username {
			continue
		}
		size, err := strconv.Atoi(parts[2])
		if err == nil && size > best {
			best = size
		}
	}
	return best, s.Err()
}

func (OSInspector) SmokeUserNS(ctx context.Context) error {
	// 실제 runc-overlay helper와 같은 mapping을 연다. --map-root-user만 되지만
	// subordinate range를 newuidmap/newgidmap으로 붙일 수 없는 중첩 userns를
	// ready로 광고하면 Open에서 뒤늦게 죽는다.
	b, err := exec.CommandContext(ctx, "unshare", "--user", "--map-root-user", "--map-auto", "true").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(b)))
	}
	return nil
}

func Check(ctx context.Context, doc Document, binding Binding, inspector Inspector) Report {
	return CheckWithRuntime(ctx, doc, binding, inspector, nil)
}

// CheckWithRuntime은 선언과 host/prepared fact를 읽은 뒤, 모두 ready일 때만 실제
// runtime smoke를 수행한다. verifier가 nil인 호출은 순수 environment package
// 시험과 embedding을 위한 것이며 제품 CLI는 항상 verifier를 넘긴다.
func CheckWithRuntime(ctx context.Context, doc Document, binding Binding, inspector Inspector, verifier RuntimeVerifier) Report {
	if inspector == nil {
		inspector = OSInspector{}
	}
	r := Report{State: StateReady, ProfileSHA256: doc.SHA256, Facts: []Fact{}, Operations: []Operation{}}
	add := func(f Fact) {
		r.Facts = append(r.Facts, f)
		if stateRank(f.State) > stateRank(r.State) {
			r.State = f.State
		}
	}

	if err := validateBinding(doc.Profile, binding); err != nil {
		add(Fact{Name: "binding", Source: "/runtime", State: StateInvalid, Observed: err.Error()})
		return r
	}
	if inspector.GOOS() != "linux" && doc.Profile.Runtime.Driver == "runc-overlay" {
		add(Fact{Name: "host.os", Source: "/runtime/driver", Required: "linux",
			Observed: inspector.GOOS(), State: StateUnsupported})
		return r
	}
	if !inspector.RuntimeDriverAvailable(doc.Profile.Runtime.Driver) {
		add(Fact{Name: "runtime.product_driver", Source: "/runtime/driver",
			Required: doc.Profile.Runtime.Driver, Observed: "not linked in this enode build",
			State:       StateUnsupported,
			Remediation: "install an enode build whose StepRuntime implements this driver"})
	}
	if !inspector.LookPath("apt-get") {
		add(Fact{Name: "host.provider", Source: "/host/provider", Required: "apt-get",
			Observed: "not found", State: StateUnsupported,
			Remediation: "install on a supported Debian or Ubuntu host"})
	} else {
		var missing []string
		installedPackages := make(map[string]bool, len(doc.Profile.Host.Packages))
		for _, name := range doc.Profile.Host.Packages {
			installed, err := inspector.PackageInstalled(ctx, name)
			if err != nil {
				add(Fact{Name: "host.package." + name, Source: "/host/packages",
					Required: "installed", Observed: err.Error(), State: StateUnsupported})
				continue
			}
			installedPackages[name] = installed
			state, observed := StateReady, "installed"
			if !installed {
				state, observed = StateInstallable, "missing"
				missing = append(missing, name)
			}
			add(Fact{Name: "host.package." + name, Source: "/host/packages",
				Required: "installed", Observed: observed, State: state})
		}
		if len(missing) > 0 {
			r.Operations = append(r.Operations,
				Operation{Kind: "host.apt.update", Source: "/host/packages", Mutates: true},
				Operation{Kind: "host.apt.ensure-packages", Source: "/host/packages", Mutates: true,
					Packages: missing})
		}
		surfaces := map[string][]string{
			"debootstrap": {"debootstrap"},
			"runc":        {"runc"},
			"uidmap":      {"newuidmap", "newgidmap"},
			"util-linux":  {"unshare", "mount", "umount", "findmnt", "mountpoint"},
		}
		for _, packageName := range doc.Profile.Host.Packages {
			for _, executable := range surfaces[packageName] {
				state, observed, remediation := StateReady, "found", ""
				if !inspector.LookPath(executable) {
					observed = "not found"
					if installedPackages[packageName] {
						state = StateUnsupported
						remediation = "repair or reinstall the declared host package " + packageName
					} else {
						state = StateInstallable
					}
				}
				add(Fact{Name: "host.executable." + executable, Source: "/host/packages",
					Required: packageName, Observed: observed, State: state, Remediation: remediation})
			}
		}
	}

	username, err := inspector.CurrentUser()
	if err != nil {
		add(Fact{Name: "host.user", Source: "/host/require", State: StateUnsupported,
			Observed: err.Error()})
	} else {
		checkSubID := func(name, path, source string, required int) {
			observed, err := inspector.SubordinateRange(path, username)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				add(Fact{Name: name, Source: source, Required: strconv.Itoa(required),
					Observed: err.Error(), State: StateAdminRequired})
				return
			}
			state := StateReady
			remediation := ""
			if observed < required {
				state = StateAdminRequired
				remediation = fmt.Sprintf("allocate a non-overlapping subordinate range of at least %d for user %s", required, username)
			}
			add(Fact{Name: name, Source: source, Required: strconv.Itoa(required),
				Observed: strconv.Itoa(observed), State: state, Remediation: remediation})
		}
		checkSubID("host.subuid_size", "/etc/subuid", "/host/require/subuid_size", doc.Profile.Host.Require.SubUIDSize)
		checkSubID("host.subgid_size", "/etc/subgid", "/host/require/subgid_size", doc.Profile.Host.Require.SubGIDSize)
	}

	if doc.Profile.Host.Require.UnprivilegedUserNS {
		var missingUserNSTools []string
		for _, name := range []string{"unshare", "newuidmap", "newgidmap"} {
			if !inspector.LookPath(name) {
				missingUserNSTools = append(missingUserNSTools, name)
			}
		}
		if len(missingUserNSTools) > 0 {
			add(Fact{Name: "host.unprivileged_userns", Source: "/host/require/unprivileged_userns",
				Required: "smoke succeeds", Observed: strings.Join(missingUserNSTools, ", ") + " not found", State: StateInstallable,
				Remediation: "install the declared uidmap and util-linux host packages"})
		} else if err := inspector.SmokeUserNS(ctx); err != nil {
			add(Fact{Name: "host.unprivileged_userns", Source: "/host/require/unprivileged_userns",
				Required: "smoke succeeds", Observed: err.Error(), State: StateExternalBlocked,
				Remediation: "enable nested user namespaces in the outer host or container policy"})
		} else {
			add(Fact{Name: "host.unprivileged_userns", Source: "/host/require/unprivileged_userns",
				Required: "smoke succeeds", Observed: "ready", State: StateReady})
		}
	}

	preparedState, observed := inspectPrepared(binding.Store, doc.Profile.Name, doc.SHA256)
	add(Fact{Name: "prepared_environment", Source: "/rootfs", Required: doc.SHA256,
		Observed: observed, State: preparedState})
	if preparedState != StateReady {
		r.Operations = append(r.Operations, rootFSOperations(doc.Profile)...)
	} else if r.State == StateReady && verifier != nil {
		manifest, err := currentManifest(binding.Store, doc.Profile.Name)
		rootfs := PreparedRootFS(binding.Store, manifest.PreparedEnvironmentID)
		if err == nil {
			err = verifier.Verify(ctx, doc, binding, rootfs, manifest)
		}
		if err != nil {
			add(Fact{Name: "runtime.smoke", Source: "/runtime/driver",
				Required: "open/run/close succeeds", Observed: err.Error(), State: StateExternalBlocked,
				Remediation: "resolve the reported namespace, mount, or runc failure and run env check again"})
		} else {
			add(Fact{Name: "runtime.smoke", Source: "/runtime/driver",
				Required: "open/run/close succeeds", Observed: "ready", State: StateReady})
		}
	}
	return r
}

func validateBinding(profile Profile, b Binding) error {
	for name, path := range map[string]string{"store": b.Store, "scratch": b.Scratch} {
		if !filepath.IsAbs(path) {
			return fmt.Errorf("environment.%s must be an absolute path", name)
		}
	}
	if profile.Runtime.Driver == "runc-overlay" && !filepath.IsAbs(b.Workspace) {
		return errors.New("workspace must be an absolute path for runc-overlay")
	}
	if profile.Runtime.Credentials.SSH == "readonly" && !filepath.IsAbs(b.SSHDir) {
		return errors.New("credentials.ssh_dir must be absolute when SSH projection is readonly")
	}
	return nil
}

func rootFSOperations(p Profile) []Operation {
	packages := append([]string(nil), p.RootFS.APT.Packages...)
	return []Operation{
		{Kind: "rootfs.debootstrap", Source: "/rootfs/builder", Mutates: true,
			Release: p.RootFS.Release, Arch: p.RootFS.Arch, Mirror: p.RootFS.Mirror,
			Components: append([]string(nil), p.RootFS.APT.Components...)},
		{Kind: "rootfs.apt.ensure-packages", Source: "/rootfs/apt/packages", Mutates: true,
			Packages: packages},
		{Kind: "rootfs.locale.ensure", Source: "/rootfs/locale", Mutates: true,
			Packages: []string{"locales"}, Locale: p.RootFS.Locale},
		{Kind: "rootfs.user.ensure", Source: "/rootfs/user", Mutates: true, User: p.RootFS.User},
		{Kind: "rootfs.runtime-targets.ensure", Source: "/runtime", Mutates: true,
			User: p.RootFS.User, WorkspaceTarget: p.Runtime.WorkspaceTarget},
		{Kind: "rootfs.verify", Source: "/verify", Mutates: false,
			Executables: append([]string(nil), p.Verify.Executables...), Locale: p.Verify.Locale,
			RuntimeDriver: p.Runtime.Driver},
	}
}

func stateRank(s State) int {
	switch s {
	case StateInvalid:
		return 7
	case StateUnsupported:
		return 6
	case StateExternalBlocked:
		return 5
	case StateAdminRequired:
		return 4
	case StateInstallable:
		return 3
	case StateStale:
		return 2
	case StateReady:
		return 1
	default:
		return 0
	}
}

type profileIndex struct {
	Name                  string `json:"name"`
	ProfileSHA256         string `json:"profile_sha256"`
	PreparedEnvironmentID string `json:"prepared_environment_id"`
}

func inspectPrepared(store, name, profileSHA string) (State, string) {
	b, err := os.ReadFile(profileIndexPath(store, name))
	if errors.Is(err, os.ErrNotExist) {
		return StateInstallable, "not prepared"
	}
	if err != nil {
		return StateStale, err.Error()
	}
	var index profileIndex
	if json.Unmarshal(b, &index) != nil || index.Name != name || index.PreparedEnvironmentID == "" {
		return StateStale, "profile index is invalid"
	}
	if index.ProfileSHA256 != profileSHA {
		return StateStale, index.ProfileSHA256
	}
	dir := environmentDir(store, index.PreparedEnvironmentID)
	manifest, err := ReadManifest(filepath.Join(dir, "manifest.json"))
	if err != nil || manifest.Profile.SHA256 != profileSHA {
		return StateStale, "manifest is missing or does not match the profile"
	}
	if st, err := os.Stat(filepath.Join(dir, "rootfs")); err != nil || !st.IsDir() {
		return StateStale, "rootfs is missing"
	}
	return StateReady, index.PreparedEnvironmentID
}

func profileIndexPath(store, name string) string {
	return filepath.Join(store, "profiles", name+".json")
}
