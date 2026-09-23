package environment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeInspector struct {
	goos      string
	runtimes  map[string]bool
	username  string
	paths     map[string]bool
	installed map[string]bool
	subuid    int
	subgid    int
	userNSErr error
}

func (f fakeInspector) GOOS() string { return f.goos }
func (f fakeInspector) RuntimeDriverAvailable(name string) bool {
	return f.runtimes[name]
}
func (f fakeInspector) CurrentUser() (string, error) { return f.username, nil }
func (f fakeInspector) LookPath(name string) bool    { return f.paths[name] }
func (f fakeInspector) PackageInstalled(context.Context, string) (bool, error) {
	return false, errors.New("unexpected package")
}
func (f fakeInspector) SubordinateRange(path, _ string) (int, error) {
	if path == "/etc/subuid" {
		return f.subuid, nil
	}
	return f.subgid, nil
}
func (f fakeInspector) SmokeUserNS(context.Context) error { return f.userNSErr }

type packagesInspector struct{ fakeInspector }

func (f packagesInspector) PackageInstalled(_ context.Context, name string) (bool, error) {
	return f.installed[name], nil
}

type fakeRuntimeVerifier struct {
	calls  int
	rootfs string
	err    error
}

func (v *fakeRuntimeVerifier) Verify(_ context.Context, _ Document, _ Binding, rootfs string, _ Manifest) error {
	v.calls++
	v.rootfs = rootfs
	return v.err
}

func testBinding(t *testing.T) Binding {
	t.Helper()
	root := t.TempDir()
	return Binding{
		Store: filepath.Join(root, "store"), Scratch: filepath.Join(root, "scratch"),
		Workspace: filepath.Join(root, "workspace"), SSHDir: filepath.Join(root, "ssh"),
	}
}

func readyInspector() packagesInspector {
	return packagesInspector{fakeInspector: fakeInspector{
		goos: "linux", username: "builder",
		runtimes: map[string]bool{"runc-overlay": true},
		paths: map[string]bool{
			"apt-get": true, "debootstrap": true, "runc": true, "unshare": true,
			"newuidmap": true, "newgidmap": true, "mount": true, "umount": true,
			"findmnt": true, "mountpoint": true,
		},
		installed: map[string]bool{"debootstrap": true, "runc": true, "uidmap": true, "util-linux": true},
		subuid:    65536, subgid: 65536,
	}}
}

func TestCheckDoesNotClaimReadyWhenTheProductRuntimeIsMissing(t *testing.T) {
	doc, _ := Parse([]byte(validProfile))
	inspector := readyInspector()
	inspector.runtimes["runc-overlay"] = false
	r := Check(context.Background(), doc, testBinding(t), inspector)
	if r.State != StateUnsupported {
		t.Fatalf("want unsupported, got %s", r.State)
	}
	found := false
	for _, fact := range r.Facts {
		if fact.Name == "runtime.product_driver" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing product runtime fact: %+v", r.Facts)
	}
}

func TestCheckDerivesEveryOperationFromAProfileField(t *testing.T) {
	doc, err := Parse([]byte(validProfile))
	if err != nil {
		t.Fatal(err)
	}
	inspector := readyInspector()
	inspector.installed["uidmap"] = false
	r := Check(context.Background(), doc, testBinding(t), inspector)
	if r.State != StateInstallable {
		t.Fatalf("want installable, got %s: %+v", r.State, r.Facts)
	}
	if len(r.Operations) == 0 {
		t.Fatal("check did not produce a plan")
	}
	for _, op := range r.Operations {
		if op.Source == "" || op.Source[0] != '/' {
			t.Fatalf("operation has no profile source: %+v", op)
		}
	}
	if got := r.Operations[1].Packages; len(got) != 1 || got[0] != "uidmap" {
		t.Fatalf("host plan contains undeclared or installed packages: %v", got)
	}
}

func TestCheckNeverPlansSubIDMutation(t *testing.T) {
	doc, _ := Parse([]byte(validProfile))
	inspector := readyInspector()
	inspector.subuid = 0
	r := Check(context.Background(), doc, testBinding(t), inspector)
	if r.State != StateAdminRequired {
		t.Fatalf("got %s", r.State)
	}
	for _, op := range r.Operations {
		if op.Kind == "host.subid.ensure" {
			t.Fatal("subid allocation escaped the administrator boundary")
		}
	}
}

func TestCheckSeparatesAnExternalUserNSBlock(t *testing.T) {
	doc, _ := Parse([]byte(validProfile))
	inspector := readyInspector()
	inspector.userNSErr = errors.New("operation not permitted")
	r := Check(context.Background(), doc, testBinding(t), inspector)
	if r.State != StateExternalBlocked {
		t.Fatalf("got %s", r.State)
	}
}

func TestCheckTreatsMissingUserNSHelpersAsInstallableBeforeSmoke(t *testing.T) {
	doc, _ := Parse([]byte(validProfile))
	inspector := readyInspector()
	inspector.installed["uidmap"] = false
	inspector.paths["newuidmap"] = false
	inspector.userNSErr = errors.New("smoke must not run before helpers exist")
	r := Check(context.Background(), doc, testBinding(t), inspector)
	if r.State != StateInstallable {
		t.Fatalf("want installable, got %s: %+v", r.State, r.Facts)
	}
	found := false
	for _, fact := range r.Facts {
		if fact.Name == "host.unprivileged_userns" {
			found = true
			if !strings.Contains(fact.Observed, "newuidmap") {
				t.Fatalf("missing helper was not named: %+v", fact)
			}
		}
	}
	if !found {
		t.Fatalf("missing userns fact: %+v", r.Facts)
	}
}

func TestCheckSaysReadyForThePublishedManifest(t *testing.T) {
	doc, _ := Parse([]byte(validProfile))
	binding := testBinding(t)
	manifest := Manifest{
		Schema:                ManifestSchema,
		PreparedEnvironmentID: "sha256-ready",
		Profile:               ManifestProfile{Name: doc.Profile.Name, SHA256: doc.SHA256},
	}
	id, err := manifest.Identity()
	if err != nil {
		t.Fatal(err)
	}
	manifest.PreparedEnvironmentID = id
	dir := environmentDir(binding.Store, manifest.PreparedEnvironmentID)
	if err := os.MkdirAll(filepath.Join(dir, "rootfs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteManifest(filepath.Join(dir, "manifest.json"), manifest); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(profileIndexPath(binding.Store, doc.Profile.Name)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(profileIndexPath(binding.Store, doc.Profile.Name), profileIndex{
		Name: doc.Profile.Name, ProfileSHA256: doc.SHA256,
		PreparedEnvironmentID: manifest.PreparedEnvironmentID,
	}); err != nil {
		t.Fatal(err)
	}
	r := Check(context.Background(), doc, binding, readyInspector())
	if r.State != StateReady || len(r.Operations) != 0 {
		t.Fatalf("want ready with no plan, got %s %+v", r.State, r.Operations)
	}
}

func TestCheckRunsTheProductRuntimeSmokeForAPublishedEnvironment(t *testing.T) {
	doc, _ := Parse([]byte(validProfile))
	binding := testBinding(t)
	manifest, err := (Preparer{Inspector: readyInspector(), Runner: &applyRunner{}}).
		Apply(context.Background(), doc, binding)
	if err != nil {
		t.Fatal(err)
	}
	verifier := &fakeRuntimeVerifier{}
	r := CheckWithRuntime(context.Background(), doc, binding, readyInspector(), verifier)
	if r.State != StateReady || verifier.calls != 1 {
		t.Fatalf("runtime smoke was not part of readiness: state=%s calls=%d facts=%+v", r.State, verifier.calls, r.Facts)
	}
	want := PreparedRootFS(binding.Store, manifest.PreparedEnvironmentID)
	if verifier.rootfs != want {
		t.Fatalf("verified %q, want %q", verifier.rootfs, want)
	}
	if r.Facts[len(r.Facts)-1].Name != "runtime.smoke" {
		t.Fatalf("missing runtime smoke fact: %+v", r.Facts)
	}
}

func TestCheckFailsClosedWhenTheProductRuntimeSmokeFails(t *testing.T) {
	doc, _ := Parse([]byte(validProfile))
	binding := testBinding(t)
	if _, err := (Preparer{Inspector: readyInspector(), Runner: &applyRunner{}}).
		Apply(context.Background(), doc, binding); err != nil {
		t.Fatal(err)
	}
	verifier := &fakeRuntimeVerifier{err: errors.New("overlay mount denied")}
	r := CheckWithRuntime(context.Background(), doc, binding, readyInspector(), verifier)
	if r.State != StateExternalBlocked {
		t.Fatalf("want external-blocked, got %s: %+v", r.State, r.Facts)
	}
}

func TestOSInspectorReadsTheActualHostSurfaces(t *testing.T) {
	inspector := OSInspector{}
	if inspector.GOOS() == "" || !inspector.RuntimeDriverAvailable("native") || inspector.RuntimeDriverAvailable("unknown") {
		t.Fatal("product runtime linkage is inconsistent")
	}
	if _, err := inspector.CurrentUser(); err != nil {
		t.Fatal(err)
	}
	if !inspector.LookPath("sh") || inspector.LookPath("enode-definitely-missing-command") {
		t.Fatal("PATH inspection is inconsistent")
	}
	if installed, err := inspector.PackageInstalled(context.Background(), "enode-definitely-missing-package"); err != nil || installed {
		t.Fatalf("missing package installed=%v err=%v", installed, err)
	}
	ranges := filepath.Join(t.TempDir(), "subids")
	if err := os.WriteFile(ranges, []byte("builder:100000:100\nbuilder:200000:200\nother:1:999\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := inspector.SubordinateRange(ranges, "builder"); err != nil || got != 200 {
		t.Fatalf("subordinate range=%d err=%v", got, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = inspector.SmokeUserNS(ctx)
}
