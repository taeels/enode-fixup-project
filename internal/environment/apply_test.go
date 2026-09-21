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

type applyRunner struct {
	calls  []string
	failAt string
	after  func(string)
}

func (r *applyRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	if name == "sudo" {
		name, args = args[0], args[1:]
	}
	call := name + " " + strings.Join(args, " ")
	r.calls = append(r.calls, call)
	if r.after != nil {
		r.after(call)
	}
	if r.failAt != "" && strings.Contains(call, r.failAt) {
		return nil, errors.New("injected failure")
	}
	switch name {
	case "debootstrap":
		if len(args) == 1 && args[0] == "--version" {
			return []byte("debootstrap 1.0\n"), nil
		}
		rootfs := args[len(args)-1]
		for _, path := range []string{"bin/bash", "usr/bin/gcc", "usr/bin/git", "usr/bin/make", "usr/bin/rpcgen"} {
			full := filepath.Join(rootfs, path)
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(full, []byte("stub"), 0o755); err != nil {
				return nil, err
			}
		}
	case "chroot":
		if len(args) >= 3 && args[1] == "locale" && args[2] == "-a" {
			return []byte("C\nen_US.utf8\n"), nil
		}
		if len(args) >= 2 && args[1] == "dpkg-query" {
			return []byte("gcc\t1\ngit\t2\nlocales\t3\nmake\t4\n"), nil
		}
	}
	return nil, nil
}

func TestApplyExecutesThePlanAndPublishesManifestLast(t *testing.T) {
	doc, err := Parse([]byte(validProfile))
	if err != nil {
		t.Fatal(err)
	}
	binding := testBinding(t)
	runner := &applyRunner{}
	manifest, err := (Preparer{
		Inspector: readyInspector(), Runner: runner,
		Now: func() time.Time { return time.Unix(123, 0) },
	}).Apply(context.Background(), doc, binding)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Profile.SHA256 != doc.SHA256 || manifest.PreparedEnvironmentID == "" {
		t.Fatalf("bad manifest: %+v", manifest)
	}
	dir := environmentDir(binding.Store, manifest.PreparedEnvironmentID)
	if _, err := os.Stat(filepath.Join(dir, "rootfs")); err != nil {
		t.Fatalf("rootfs was not published: %v", err)
	}
	got, err := currentManifest(binding.Store, doc.Profile.Name)
	if err != nil || got.PreparedEnvironmentID != manifest.PreparedEnvironmentID {
		t.Fatalf("profile index does not select the manifest: %+v %v", got, err)
	}
	joined := strings.Join(runner.calls, "\n")
	for _, forbidden := range []string{"add-apt-repository", "software-properties-common", "/etc/subuid", "/etc/subgid"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("apply ran a hidden or administrator operation %q:\n%s", forbidden, joined)
		}
	}
}

func TestApplyDoesNotPublishAFailedBuild(t *testing.T) {
	doc, _ := Parse([]byte(validProfile))
	binding := testBinding(t)
	runner := &applyRunner{failAt: "apt-get install"}
	_, err := (Preparer{Inspector: readyInspector(), Runner: runner}).Apply(context.Background(), doc, binding)
	if err == nil {
		t.Fatal("apply succeeded after a rootfs operation failed")
	}
	if _, err := os.Stat(profileIndexPath(binding.Store, doc.Profile.Name)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed build was published: %v", err)
	}
	entries, err := os.ReadDir(binding.Store)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".build-") {
			t.Fatalf("failed build escaped the diagnostic temporary namespace: %s", entry.Name())
		}
	}
}

func TestApplyRefusesAdministratorRequiredStateBeforeMutation(t *testing.T) {
	doc, _ := Parse([]byte(validProfile))
	inspector := readyInspector()
	inspector.subgid = 0
	runner := &applyRunner{}
	_, err := (Preparer{Inspector: inspector, Runner: runner}).Apply(context.Background(), doc, testBinding(t))
	if err == nil || !strings.Contains(err.Error(), "admin-required") {
		t.Fatalf("got %v", err)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("apply mutated the host before administrator remediation: %v", runner.calls)
	}
}

func TestTwoNodeBindingsShareOnePreparedRootFS(t *testing.T) {
	doc, _ := Parse([]byte(validProfile))
	binding := testBinding(t)
	manifest, err := (Preparer{Inspector: readyInspector(), Runner: &applyRunner{}}).
		Apply(context.Background(), doc, binding)
	if err != nil {
		t.Fatal(err)
	}
	other := binding
	other.Workspace = filepath.Join(t.TempDir(), "other-workspace")
	other.Scratch = filepath.Join(t.TempDir(), "other-scratch")
	r := Check(context.Background(), doc, other, readyInspector())
	if r.State != StateReady || len(r.Operations) != 0 {
		t.Fatalf("second node did not reuse %s: %s %+v", manifest.PreparedEnvironmentID, r.State, r.Operations)
	}
}

func TestChangedProfileIsStaleAndNeverAutoAppliedByCheck(t *testing.T) {
	doc, _ := Parse([]byte(validProfile))
	binding := testBinding(t)
	runner := &applyRunner{}
	if _, err := (Preparer{Inspector: readyInspector(), Runner: runner}).Apply(context.Background(), doc, binding); err != nil {
		t.Fatal(err)
	}
	before := len(runner.calls)
	changed, err := Parse([]byte(validProfile + "\n# operator revision\n"))
	if err != nil {
		t.Fatal(err)
	}
	r := Check(context.Background(), changed, binding, readyInspector())
	if r.State != StateStale {
		t.Fatalf("want stale after profile byte change, got %s", r.State)
	}
	if len(runner.calls) != before {
		t.Fatal("read-only check executed the apply runner")
	}
}

func TestApplyAbortsWhenProfileChangesAfterPlanning(t *testing.T) {
	dir := t.TempDir()
	profilePath := filepath.Join(dir, "profile.yaml")
	if err := os.WriteFile(profilePath, []byte(validProfile), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := Load(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	changed := false
	runner := &applyRunner{after: func(call string) {
		if !changed && strings.HasPrefix(call, "debootstrap ") {
			changed = true
			if err := os.WriteFile(profilePath, []byte(validProfile+"\n# changed during apply\n"), 0o644); err != nil {
				t.Errorf("change profile: %v", err)
			}
		}
	}}
	binding := testBinding(t)
	_, err = (Preparer{Inspector: readyInspector(), Runner: runner}).Apply(context.Background(), doc, binding)
	if err == nil || !strings.Contains(err.Error(), "profile changed while apply was running") {
		t.Fatalf("apply did not discard the stale plan: %v", err)
	}
	if _, statErr := os.Stat(profileIndexPath(binding.Store, doc.Profile.Name)); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("changed profile was published: %v", statErr)
	}
}

func TestResolvedPackagesIncludesTheInstalledDependencyClosure(t *testing.T) {
	runner := &applyRunner{}
	rootfs := t.TempDir()
	got, err := resolvedPackages(context.Background(), runner, rootfs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 || got[0].Name != "gcc" || got[3].Name != "make" {
		t.Fatalf("unexpected package closure: %+v", got)
	}
	joined := strings.Join(runner.calls, "\n")
	if strings.Contains(joined, " -- ") {
		t.Fatalf("dpkg-query was narrowed to declared package names: %s", joined)
	}
}
