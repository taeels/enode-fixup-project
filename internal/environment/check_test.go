package environment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
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
		runtimes:  map[string]bool{"runc-overlay": true},
		paths:     map[string]bool{"apt-get": true, "unshare": true},
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
