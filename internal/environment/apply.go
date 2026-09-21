package environment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := commandContext(ctx, name, args...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return b, fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(b)))
	}
	return b, nil
}

type Preparer struct {
	Inspector Inspector
	Runner    CommandRunner
	Now       func() time.Time
}

func (p Preparer) Apply(ctx context.Context, doc Document, binding Binding) (Manifest, error) {
	inspector := p.Inspector
	if inspector == nil {
		inspector = OSInspector{}
	}
	runner := p.Runner
	if runner == nil {
		runner = ExecRunner{}
	}
	now := p.Now
	if now == nil {
		now = time.Now
	}
	report := Check(ctx, doc, binding, inspector)
	switch report.State {
	case StateInvalid, StateUnsupported, StateExternalBlocked, StateAdminRequired:
		return Manifest{}, fmt.Errorf("environment is %s; run env check and resolve its blocking facts", report.State)
	case StateReady:
		return currentManifest(binding.Store, doc.Profile.Name)
	}

	for _, op := range report.Operations {
		if err := ensureProfileUnchanged(doc); err != nil {
			return Manifest{}, err
		}
		if !strings.HasPrefix(op.Source, "/") {
			return Manifest{}, fmt.Errorf("operation %s has no profile source", op.Kind)
		}
		if strings.HasPrefix(op.Kind, "host.") {
			if err := executeHostOperation(ctx, runner, op); err != nil {
				return Manifest{}, err
			}
		}
	}

	if err := os.MkdirAll(binding.Store, 0o755); err != nil {
		return Manifest{}, fmt.Errorf("create environment store: %w", err)
	}
	buildDir, err := os.MkdirTemp(binding.Store, ".build-")
	if err != nil {
		return Manifest{}, fmt.Errorf("create environment build directory: %w", err)
	}
	rootfs := filepath.Join(buildDir, "rootfs")

	for _, op := range report.Operations {
		if !strings.HasPrefix(op.Kind, "rootfs.") {
			continue
		}
		if err := ensureProfileUnchanged(doc); err != nil {
			return Manifest{}, fmt.Errorf("%w; incomplete build kept at %s", err, buildDir)
		}
		if err := executeRootFSOperation(ctx, runner, rootfs, op); err != nil {
			return Manifest{}, fmt.Errorf("%s from %s: %w; incomplete build kept at %s",
				op.Kind, op.Source, err, buildDir)
		}
	}

	packages, err := resolvedPackages(ctx, runner, rootfs)
	if err != nil {
		return Manifest{}, fmt.Errorf("read resolved package versions: %w; incomplete build kept at %s", err, buildDir)
	}
	versionOut, err := runner.Run(ctx, "debootstrap", "--version")
	if err != nil {
		return Manifest{}, fmt.Errorf("read debootstrap version: %w; incomplete build kept at %s", err, buildDir)
	}
	manifest := Manifest{
		Schema:  ManifestSchema,
		Profile: ManifestProfile{Name: doc.Profile.Name, SHA256: doc.SHA256},
		Builder: ManifestBuilder{Kind: doc.Profile.RootFS.Builder, Version: firstLine(versionOut)},
		Base: ManifestBase{
			Release: doc.Profile.RootFS.Release,
			Arch:    doc.Profile.RootFS.Arch,
			Mirror:  effectiveMirror(doc.Profile.RootFS),
		},
		RootFS:     ManifestRootFS{User: doc.Profile.RootFS.User, Locale: doc.Profile.RootFS.Locale},
		Packages:   packages,
		VerifiedAt: now().UTC(),
	}
	id, err := manifest.Identity()
	if err != nil {
		return Manifest{}, err
	}
	manifest.PreparedEnvironmentID = id
	if err := ensureProfileUnchanged(doc); err != nil {
		return Manifest{}, fmt.Errorf("%w; complete build kept at %s", err, buildDir)
	}
	if err := WriteManifest(filepath.Join(buildDir, "manifest.json"), manifest); err != nil {
		return Manifest{}, fmt.Errorf("write manifest: %w; incomplete build kept at %s", err, buildDir)
	}
	if _, err := runPrivileged(ctx, runner, "chmod", "-R", "a-w", rootfs); err != nil {
		return Manifest{}, fmt.Errorf("seal rootfs: %w; incomplete build kept at %s", err, buildDir)
	}

	destination := environmentDir(binding.Store, id)
	if _, err := os.Stat(destination); err == nil {
		if err := discardBuild(ctx, runner, binding.Store, buildDir); err != nil {
			return Manifest{}, fmt.Errorf("prepared environment already exists but duplicate build cleanup failed: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Manifest{}, err
	} else if err := os.Rename(buildDir, destination); err != nil {
		return Manifest{}, fmt.Errorf("publish prepared environment: %w; complete build kept at %s", err, buildDir)
	}

	index := profileIndex{Name: doc.Profile.Name, ProfileSHA256: doc.SHA256, PreparedEnvironmentID: id}
	if err := writeJSON(profileIndexPath(binding.Store, doc.Profile.Name), index); err != nil {
		return Manifest{}, fmt.Errorf("publish profile index: %w", err)
	}
	return manifest, nil
}

func executeHostOperation(ctx context.Context, runner CommandRunner, op Operation) error {
	switch op.Kind {
	case "host.apt.update":
		_, err := runPrivileged(ctx, runner, "apt-get", "update")
		return err
	case "host.apt.ensure-packages":
		if len(op.Packages) == 0 {
			return nil
		}
		args := []string{"install", "-y", "--"}
		args = append(args, op.Packages...)
		_, err := runPrivileged(ctx, runner, "apt-get", args...)
		return err
	default:
		return fmt.Errorf("unsupported host operation %q", op.Kind)
	}
}

func executeRootFSOperation(ctx context.Context, runner CommandRunner, rootfs string, op Operation) error {
	switch op.Kind {
	case "rootfs.debootstrap":
		args := []string{"--arch=" + op.Arch, "--components=" + strings.Join(op.Components, ","), op.Release, rootfs}
		if op.Mirror != "" {
			args = append(args, op.Mirror)
		}
		_, err := runPrivileged(ctx, runner, "debootstrap", args...)
		return err
	case "rootfs.apt.ensure-packages":
		if _, err := runChroot(ctx, runner, rootfs, "apt-get", "update"); err != nil {
			return err
		}
		args := []string{"DEBIAN_FRONTEND=noninteractive", "apt-get", "install", "-y", "--no-install-recommends", "--"}
		args = append(args, op.Packages...)
		_, err := runChroot(ctx, runner, rootfs, "/usr/bin/env", args...)
		return err
	case "rootfs.locale.ensure":
		args := []string{"DEBIAN_FRONTEND=noninteractive", "apt-get", "install", "-y", "--no-install-recommends", "--"}
		args = append(args, op.Packages...)
		if _, err := runChroot(ctx, runner, rootfs, "/usr/bin/env", args...); err != nil {
			return err
		}
		if _, err := runChroot(ctx, runner, rootfs, "locale-gen", op.Locale); err != nil {
			return err
		}
		_, err := runChroot(ctx, runner, rootfs, "update-locale", "LANG="+op.Locale, "LC_ALL="+op.Locale)
		return err
	case "rootfs.user.ensure":
		if _, err := runChroot(ctx, runner, rootfs, "groupadd", "-g", fmt.Sprint(op.User.GID), op.User.Name); err != nil {
			return err
		}
		_, err := runChroot(ctx, runner, rootfs, "useradd", "-u", fmt.Sprint(op.User.UID),
			"-g", fmt.Sprint(op.User.GID), "-m", "-s", "/bin/bash", op.User.Name)
		return err
	case "rootfs.runtime-targets.ensure":
		target, err := beneath(rootfs, op.WorkspaceTarget)
		if err != nil {
			return err
		}
		home, err := beneath(rootfs, "/home/"+op.User.Name+"/.ssh")
		if err != nil {
			return err
		}
		if _, err := runPrivileged(ctx, runner, "install", "-d", "-o", fmt.Sprint(op.User.UID),
			"-g", fmt.Sprint(op.User.GID), "-m", "0755", target); err != nil {
			return err
		}
		_, err = runPrivileged(ctx, runner, "install", "-d", "-o", fmt.Sprint(op.User.UID),
			"-g", fmt.Sprint(op.User.GID), "-m", "0700", home)
		return err
	case "rootfs.verify":
		for _, name := range op.Executables {
			if !rootfsExecutable(rootfs, name) {
				return fmt.Errorf("executable %s is not in the rootfs runtime PATH", name)
			}
		}
		b, err := runChroot(ctx, runner, rootfs, "locale", "-a")
		if err != nil {
			return err
		}
		want := strings.ToLower(strings.ReplaceAll(op.Locale, "-", ""))
		got := strings.ToLower(strings.ReplaceAll(string(b), "-", ""))
		if !strings.Contains(got, want) {
			return fmt.Errorf("locale %s is not available", op.Locale)
		}
		return nil
	default:
		return fmt.Errorf("unsupported rootfs operation %q", op.Kind)
	}
}

func runChroot(ctx context.Context, runner CommandRunner, rootfs, name string, args ...string) ([]byte, error) {
	all := []string{"chroot", rootfs, name}
	all = append(all, args...)
	return runPrivileged(ctx, runner, all[0], all[1:]...)
}

func runPrivileged(ctx context.Context, runner CommandRunner, name string, args ...string) ([]byte, error) {
	executable, all := privilegedCommand(name, args)
	return runner.Run(ctx, executable, all...)
}

func resolvedPackages(ctx context.Context, runner CommandRunner, rootfs string) ([]ManifestPackage, error) {
	// prepared identity는 선언 목록만이 아니라 실제 설치된 dependency closure를
	// 봉인한다. 그래야 직접 package version이 같아도 전이 dependency가 달라진
	// 두 rootfs를 같은 준비 산출물로 오인하지 않는다.
	b, err := runChroot(ctx, runner, rootfs, "dpkg-query", "-W", "-f=${Package}\t${Version}\\n")
	if err != nil {
		return nil, err
	}
	var out []ManifestPackage
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		name, version, ok := strings.Cut(line, "\t")
		if !ok || name == "" || version == "" {
			return nil, fmt.Errorf("unexpected dpkg-query line %q", line)
		}
		out = append(out, ManifestPackage{Name: name, Version: version})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func ensureProfileUnchanged(doc Document) error {
	// Parse로 직접 만든 문서는 파일 경계가 없으므로 immutable value로 취급한다.
	// Load로 읽은 실제 CLI 문서는 plan 생성 뒤 bytes가 바뀌면 폐기한다.
	if doc.Path == "" {
		return nil
	}
	b, err := os.ReadFile(doc.Path)
	if err != nil {
		return fmt.Errorf("re-read profile before apply: %w", err)
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != doc.SHA256 {
		return errors.New("profile changed while apply was running; discard this plan and run env check again")
	}
	return nil
}

func currentManifest(store, name string) (Manifest, error) {
	b, err := os.ReadFile(profileIndexPath(store, name))
	if err != nil {
		return Manifest{}, err
	}
	var index profileIndex
	if err := jsonUnmarshal(b, &index); err != nil {
		return Manifest{}, err
	}
	return ReadManifest(filepath.Join(environmentDir(store, index.PreparedEnvironmentID), "manifest.json"))
}

func firstLine(b []byte) string {
	line, _, _ := strings.Cut(strings.TrimSpace(string(b)), "\n")
	return line
}

func effectiveMirror(rootfs RootFSProfile) string {
	if rootfs.Mirror != "" {
		return rootfs.Mirror
	}
	switch rootfs.Release {
	case "noble", "jammy":
		return "http://archive.ubuntu.com/ubuntu"
	case "bookworm":
		return "http://deb.debian.org/debian"
	default:
		return "distribution-default"
	}
}

func rootfsExecutable(rootfs, name string) bool {
	for _, dir := range []string{"usr/local/sbin", "usr/local/bin", "usr/sbin", "usr/bin", "sbin", "bin"} {
		st, err := os.Stat(filepath.Join(rootfs, dir, name))
		if err == nil && st.Mode().IsRegular() && st.Mode().Perm()&0o111 != 0 {
			return true
		}
	}
	return false
}

func beneath(root, absolute string) (string, error) {
	if !path.IsAbs(absolute) {
		return "", fmt.Errorf("target %q is not absolute", absolute)
	}
	rel := strings.TrimPrefix(path.Clean(absolute), "/")
	if rel == "" || rel == "." || strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("target %q escapes the rootfs", absolute)
	}
	return filepath.Join(root, filepath.FromSlash(rel)), nil
}

func discardBuild(ctx context.Context, runner CommandRunner, store, buildDir string) error {
	rel, err := filepath.Rel(store, buildDir)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) ||
		!strings.HasPrefix(filepath.Base(buildDir), ".build-") {
		return fmt.Errorf("refuse to remove unverified build path %q", buildDir)
	}
	if err := os.RemoveAll(buildDir); err == nil {
		return nil
	}
	_, err = runPrivileged(ctx, runner, "rm", "-rf", "--", buildDir)
	return err
}
