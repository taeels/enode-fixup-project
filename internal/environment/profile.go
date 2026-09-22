// Package environment는 노드 로컬 실행 환경 profile과 준비 산출물을 다룬다.
package environment

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	APIVersion = "enode.dev/v1alpha1"
	Kind       = "execution-environment"
)

type Profile struct {
	APIVersion string        `yaml:"api_version" json:"api_version"`
	Kind       string        `yaml:"kind" json:"kind"`
	Name       string        `yaml:"name" json:"name"`
	Host       HostProfile   `yaml:"host" json:"host"`
	RootFS     RootFSProfile `yaml:"rootfs" json:"rootfs"`
	Runtime    RuntimePolicy `yaml:"runtime" json:"runtime"`
	Verify     VerifyPolicy  `yaml:"verify" json:"verify"`
}

type HostProfile struct {
	Provider string          `yaml:"provider" json:"provider"`
	Packages []string        `yaml:"packages" json:"packages"`
	Require  HostRequirement `yaml:"require" json:"require"`
}

type HostRequirement struct {
	SubUIDSize         int  `yaml:"subuid_size" json:"subuid_size"`
	SubGIDSize         int  `yaml:"subgid_size" json:"subgid_size"`
	UnprivilegedUserNS bool `yaml:"unprivileged_userns" json:"unprivileged_userns"`
}

type RootFSProfile struct {
	Builder string     `yaml:"builder" json:"builder"`
	Release string     `yaml:"release" json:"release"`
	Arch    string     `yaml:"arch" json:"arch"`
	Mirror  string     `yaml:"mirror,omitempty" json:"mirror,omitempty"`
	APT     APTProfile `yaml:"apt" json:"apt"`
	Locale  string     `yaml:"locale" json:"locale"`
	User    RootFSUser `yaml:"user" json:"user"`
}

type APTProfile struct {
	Components []string `yaml:"components" json:"components"`
	Packages   []string `yaml:"packages" json:"packages"`
}

type RootFSUser struct {
	Name string `yaml:"name" json:"name"`
	UID  int    `yaml:"uid" json:"uid"`
	GID  int    `yaml:"gid" json:"gid"`
}

type RuntimePolicy struct {
	Driver          string           `yaml:"driver" json:"driver"`
	WorkspaceTarget string           `yaml:"workspace_target" json:"workspace_target"`
	Tmp             TmpPolicy        `yaml:"tmp" json:"tmp"`
	Credentials     CredentialPolicy `yaml:"credentials" json:"credentials"`
}

type TmpPolicy struct {
	Size       string `yaml:"size" json:"size"`
	Executable bool   `yaml:"executable" json:"executable"`
}

type CredentialPolicy struct {
	SSH string `yaml:"ssh,omitempty" json:"ssh,omitempty"`
}

type VerifyPolicy struct {
	Executables []string `yaml:"executables" json:"executables"`
	Locale      string   `yaml:"locale" json:"locale"`
}

type Document struct {
	Profile Profile
	Path    string
	SHA256  string
	Bytes   []byte
}

var (
	namePattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	packagePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9+.-]*$`)
	tokenPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)
	localePattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.@-]*$`)
)

var supportedReleases = map[string]bool{"noble": true, "jammy": true, "bookworm": true}

func Load(path string) (Document, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Document{}, fmt.Errorf("read profile %s: %w", path, err)
	}
	doc, err := Parse(b)
	if err != nil {
		return Document{}, fmt.Errorf("profile %s: %w", path, err)
	}
	doc.Path = path
	return doc, nil
}

func Parse(b []byte) (Document, error) {
	var tree yaml.Node
	if err := yaml.Unmarshal(b, &tree); err != nil {
		return Document{}, err
	}
	if err := rejectIndirectYAML(&tree); err != nil {
		return Document{}, err
	}

	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	var p Profile
	if err := dec.Decode(&p); err != nil {
		return Document{}, err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Document{}, errors.New("multiple YAML documents are not allowed")
		}
		return Document{}, err
	}
	if err := p.Validate(); err != nil {
		return Document{}, err
	}
	sum := sha256.Sum256(b)
	return Document{
		Profile: p,
		SHA256:  hex.EncodeToString(sum[:]),
		Bytes:   append([]byte(nil), b...),
	}, nil
}

func rejectIndirectYAML(n *yaml.Node) error {
	if n == nil {
		return nil
	}
	if n.Kind == yaml.AliasNode || n.Anchor != "" {
		return errors.New("YAML aliases and anchors are not allowed")
	}
	if n.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == "<<" {
				return errors.New("YAML merge keys are not allowed")
			}
		}
	}
	for _, child := range n.Content {
		if err := rejectIndirectYAML(child); err != nil {
			return err
		}
	}
	return nil
}

func (p Profile) Validate() error {
	if p.APIVersion != APIVersion {
		return fmt.Errorf("api_version must be %q", APIVersion)
	}
	if p.Kind != Kind {
		return fmt.Errorf("kind must be %q", Kind)
	}
	if !namePattern.MatchString(p.Name) {
		return errors.New("name must match [a-z0-9][a-z0-9._-]*")
	}
	if p.Host.Provider != "apt" {
		return fmt.Errorf("unsupported host provider %q", p.Host.Provider)
	}
	if err := validateSortedSet("host.packages", p.Host.Packages, packagePattern); err != nil {
		return err
	}
	requiredHostPackages := []string{"debootstrap"}
	if p.Runtime.Driver == "runc-overlay" {
		requiredHostPackages = append(requiredHostPackages, "runc", "uidmap", "util-linux")
	}
	for _, name := range requiredHostPackages {
		if i := sort.SearchStrings(p.Host.Packages, name); i >= len(p.Host.Packages) || p.Host.Packages[i] != name {
			return fmt.Errorf("host.packages must declare %s for the selected builder/runtime", name)
		}
	}
	if p.Host.Require.SubUIDSize <= 0 || p.Host.Require.SubGIDSize <= 0 {
		return errors.New("host.require subuid_size and subgid_size must be positive")
	}
	if p.RootFS.Builder != "debootstrap" {
		return fmt.Errorf("unsupported rootfs builder %q", p.RootFS.Builder)
	}
	if !namePattern.MatchString(p.RootFS.Release) || !namePattern.MatchString(p.RootFS.Arch) {
		return errors.New("rootfs release and arch must be safe tokens")
	}
	if !supportedReleases[p.RootFS.Release] {
		return fmt.Errorf("unsupported rootfs release %q", p.RootFS.Release)
	}
	if p.RootFS.Mirror != "" && !strings.HasPrefix(p.RootFS.Mirror, "http://") &&
		!strings.HasPrefix(p.RootFS.Mirror, "https://") {
		return errors.New("rootfs mirror must use http or https")
	}
	if err := validateSortedSet("rootfs.apt.components", p.RootFS.APT.Components, packagePattern); err != nil {
		return err
	}
	if err := validateSortedSet("rootfs.apt.packages", p.RootFS.APT.Packages, packagePattern); err != nil {
		return err
	}
	if !localePattern.MatchString(p.RootFS.Locale) {
		return errors.New("rootfs.locale is invalid")
	}
	if !namePattern.MatchString(p.RootFS.User.Name) || p.RootFS.User.UID <= 0 || p.RootFS.User.GID <= 0 {
		return errors.New("rootfs.user needs a safe name and positive uid/gid")
	}
	if p.Runtime.Driver != "native" && p.Runtime.Driver != "runc-overlay" {
		return fmt.Errorf("unsupported runtime driver %q", p.Runtime.Driver)
	}
	if p.Runtime.Driver == "runc-overlay" {
		if !path.IsAbs(p.Runtime.WorkspaceTarget) || path.Clean(p.Runtime.WorkspaceTarget) == "/" {
			return errors.New("runtime.workspace_target must be an absolute path other than /")
		}
		if _, err := ParseSize(p.Runtime.Tmp.Size); err != nil {
			return fmt.Errorf("runtime.tmp.size: %w", err)
		}
		if p.Runtime.Credentials.SSH != "" && p.Runtime.Credentials.SSH != "readonly" {
			return errors.New("runtime.credentials.ssh must be readonly when set")
		}
		if !p.Host.Require.UnprivilegedUserNS {
			return errors.New("runc-overlay requires host.require.unprivileged_userns")
		}
		if p.RootFS.User.UID >= p.Host.Require.SubUIDSize || p.RootFS.User.GID >= p.Host.Require.SubGIDSize {
			return errors.New("rootfs.user uid/gid must fit inside the declared subordinate id ranges")
		}
	}
	if err := validateSortedSet("verify.executables", p.Verify.Executables, tokenPattern); err != nil {
		return err
	}
	if p.Verify.Locale != "" && p.Verify.Locale != p.RootFS.Locale {
		return errors.New("verify.locale must equal rootfs.locale")
	}
	return nil
}

func validateSortedSet(field string, values []string, pattern *regexp.Regexp) error {
	if len(values) == 0 {
		return fmt.Errorf("%s must not be empty", field)
	}
	for i, value := range values {
		if !pattern.MatchString(value) || strings.HasPrefix(value, "-") {
			return fmt.Errorf("%s contains invalid value %q", field, value)
		}
		if i > 0 && values[i-1] >= value {
			return fmt.Errorf("%s must be sorted and contain no duplicates", field)
		}
	}
	return nil
}

func ParseSize(value string) (int64, error) {
	var multiplier int64
	switch {
	case strings.HasSuffix(value, "MiB"):
		multiplier = 1 << 20
		value = strings.TrimSuffix(value, "MiB")
	case strings.HasSuffix(value, "GiB"):
		multiplier = 1 << 30
		value = strings.TrimSuffix(value, "GiB")
	default:
		return 0, errors.New("use MiB or GiB")
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n <= 0 || n > (1<<63-1)/multiplier {
		return 0, errors.New("size must be a positive bounded integer")
	}
	return n * multiplier, nil
}

func sortedCopy(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}
