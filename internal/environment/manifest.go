package environment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const ManifestSchema = "enode.dev/prepared-environment/v1alpha1"

type Manifest struct {
	Schema                string            `json:"schema"`
	PreparedEnvironmentID string            `json:"prepared_environment_id"`
	Profile               ManifestProfile   `json:"profile"`
	Builder               ManifestBuilder   `json:"builder"`
	Base                  ManifestBase      `json:"base"`
	RootFS                ManifestRootFS    `json:"rootfs"`
	Packages              []ManifestPackage `json:"packages"`
	VerifiedAt            time.Time         `json:"verified_at"`
}

type ManifestProfile struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

type ManifestBuilder struct {
	Kind    string `json:"kind"`
	Version string `json:"version"`
}

type ManifestBase struct {
	Release string `json:"release"`
	Arch    string `json:"arch"`
	Mirror  string `json:"mirror"`
}

type ManifestRootFS struct {
	User   RootFSUser `json:"user"`
	Locale string     `json:"locale"`
}

type ManifestPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (m Manifest) Identity() (string, error) {
	packages := append([]ManifestPackage(nil), m.Packages...)
	sort.Slice(packages, func(i, j int) bool {
		if packages[i].Name == packages[j].Name {
			return packages[i].Version < packages[j].Version
		}
		return packages[i].Name < packages[j].Name
	})
	projection := struct {
		Schema   string            `json:"schema"`
		Profile  ManifestProfile   `json:"profile"`
		Builder  ManifestBuilder   `json:"builder"`
		Base     ManifestBase      `json:"base"`
		RootFS   ManifestRootFS    `json:"rootfs"`
		Packages []ManifestPackage `json:"packages"`
	}{m.Schema, m.Profile, m.Builder, m.Base, m.RootFS, packages}
	b, err := json.Marshal(projection)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func ReadManifest(path string) (Manifest, error) {
	var m Manifest
	b, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return Manifest{}, fmt.Errorf("manifest %s: %w", path, err)
	}
	if m.Schema != ManifestSchema || m.PreparedEnvironmentID == "" {
		return Manifest{}, fmt.Errorf("manifest %s has an unsupported schema or empty identity", path)
	}
	want, err := m.Identity()
	if err != nil {
		return Manifest{}, err
	}
	if want != m.PreparedEnvironmentID {
		return Manifest{}, fmt.Errorf("manifest %s identity mismatch", path)
	}
	return m, nil
}

func WriteManifest(path string, m Manifest) error {
	if m.Schema == "" {
		m.Schema = ManifestSchema
	}
	id, err := m.Identity()
	if err != nil {
		return err
	}
	if m.PreparedEnvironmentID == "" {
		m.PreparedEnvironmentID = id
	}
	if m.PreparedEnvironmentID != id {
		return errorsIdentityMismatch(path)
	}
	return writeJSON(path, m)
}

func errorsIdentityMismatch(path string) error {
	return fmt.Errorf("manifest %s identity does not match its stable fields", path)
}

func writeJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".json-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) //nolint:errcheck
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close() //nolint:errcheck
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close() //nolint:errcheck
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close() //nolint:errcheck
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func environmentDir(store, id string) string {
	return filepath.Join(store, strings.ReplaceAll(id, ":", "-"))
}
