package environment

import (
	"strings"
	"testing"
)

const validProfile = `api_version: enode.dev/v1alpha1
kind: execution-environment
name: samsung-eabsp
host:
  provider: apt
  packages: [debootstrap, runc, uidmap, util-linux]
  require:
    subuid_size: 65536
    subgid_size: 65536
    unprivileged_userns: true
rootfs:
  builder: debootstrap
  release: noble
  arch: amd64
  apt:
    components: [main, universe]
    packages: [gcc, git, make]
  locale: en_US.UTF-8
  user: {name: enode, uid: 1000, gid: 1000}
runtime:
  driver: runc-overlay
  workspace_target: /srv/workspaces/product
  tmp: {size: 256MiB, executable: true}
  credentials: {ssh: readonly}
verify:
  executables: [bash, gcc, git, make]
  locale: en_US.UTF-8
`

func TestParseAcceptsTheTypedProfile(t *testing.T) {
	doc, err := Parse([]byte(validProfile))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Profile.Name != "samsung-eabsp" || len(doc.SHA256) != 64 {
		t.Fatalf("unexpected document: %+v", doc)
	}
}

func TestParseRejectsUnknownFieldsAndShellHooks(t *testing.T) {
	for _, extra := range []string{
		"\nhooks: [echo bad]\n",
		"\ncommands: [apt update]\n",
		"\nrootfs:\n  unknown: true\n",
	} {
		body := validProfile + extra
		if _, err := Parse([]byte(body)); err == nil {
			t.Fatalf("accepted an unknown execution escape: %q", extra)
		}
	}
}

func TestParseRejectsAliasesMergeKeysAndMultipleDocuments(t *testing.T) {
	cases := []string{
		strings.Replace(validProfile, "packages: [debootstrap, runc, uidmap, util-linux]", "packages: &pkgs [debootstrap, runc, uidmap, util-linux]", 1),
		validProfile + "---\n" + validProfile,
	}
	for _, body := range cases {
		if _, err := Parse([]byte(body)); err == nil {
			t.Fatal("accepted indirect or multiple YAML documents")
		}
	}
}

func TestParseRequiresSortedUniqueSets(t *testing.T) {
	body := strings.Replace(validProfile,
		"packages: [debootstrap, runc, uidmap, util-linux]",
		"packages: [uidmap, debootstrap, uidmap]", 1)
	if _, err := Parse([]byte(body)); err == nil || !strings.Contains(err.Error(), "sorted") {
		t.Fatalf("want sorted-set error, got %v", err)
	}
}

func TestParseSize(t *testing.T) {
	if got, err := ParseSize("256MiB"); err != nil || got != 256<<20 {
		t.Fatalf("got %d, %v", got, err)
	}
	for _, value := range []string{"", "0MiB", "256MB", "-1GiB"} {
		if _, err := ParseSize(value); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
}

func TestParseRequiresTheRuntimeUserToFitTheSubordinateRange(t *testing.T) {
	body := strings.Replace(validProfile, "subuid_size: 65536", "subuid_size: 1000", 1)
	if _, err := Parse([]byte(body)); err == nil || !strings.Contains(err.Error(), "subordinate") {
		t.Fatalf("accepted an unmappable runtime uid: %v", err)
	}
}

func TestSortedCopyDoesNotMutateTheProfileValue(t *testing.T) {
	source := []string{"z", "a"}
	got := sortedCopy(source)
	if strings.Join(got, ",") != "a,z" || strings.Join(source, ",") != "z,a" {
		t.Fatalf("sorted=%v source=%v", got, source)
	}
}

func TestParseRejectsInvalidTypedProfileCombinations(t *testing.T) {
	cases := []string{
		strings.Replace(validProfile, APIVersion, "enode.dev/v9", 1),
		strings.Replace(validProfile, "kind: execution-environment", "kind: shell-script", 1),
		strings.Replace(validProfile, "name: samsung-eabsp", "name: BAD NAME", 1),
		strings.Replace(validProfile, "provider: apt", "provider: dnf", 1),
		strings.Replace(validProfile, "packages: [debootstrap, runc, uidmap, util-linux]", "packages: []", 1),
		strings.Replace(validProfile, "subgid_size: 65536", "subgid_size: 0", 1),
		strings.Replace(validProfile, "builder: debootstrap", "builder: dockerfile", 1),
		strings.Replace(validProfile, "release: noble", "release: unstable", 1),
		strings.Replace(validProfile, "arch: amd64", "arch: BAD/ARCH", 1),
		strings.Replace(validProfile, "components: [main, universe]", "components: []", 1),
		strings.Replace(validProfile, "locale: en_US.UTF-8", "locale: bad locale", 1),
		strings.Replace(validProfile, "user: {name: enode, uid: 1000, gid: 1000}", "user: {name: enode, uid: 0, gid: 1000}", 1),
		strings.Replace(validProfile, "driver: runc-overlay", "driver: magic", 1),
		strings.Replace(validProfile, "workspace_target: /srv/workspaces/product", "workspace_target: relative", 1),
		strings.Replace(validProfile, "tmp: {size: 256MiB, executable: true}", "tmp: {size: huge, executable: true}", 1),
		strings.Replace(validProfile, "credentials: {ssh: readonly}", "credentials: {ssh: writable}", 1),
		strings.Replace(validProfile, "unprivileged_userns: true", "unprivileged_userns: false", 1),
		strings.Replace(validProfile, "verify:\n  executables: [bash, gcc, git, make]\n  locale: en_US.UTF-8", "verify:\n  executables: [bash, gcc, git, make]\n  locale: C.UTF-8", 1),
	}
	for i, body := range cases {
		if _, err := Parse([]byte(body)); err == nil {
			t.Fatalf("case %d accepted invalid profile", i)
		}
	}
}
