package environment

import (
	"path/filepath"
	"testing"
	"time"
)

func TestManifestIdentityExcludesTimeAndSortsPackages(t *testing.T) {
	base := Manifest{
		Schema:     ManifestSchema,
		Profile:    ManifestProfile{Name: "samsung-eabsp", SHA256: "profile"},
		Builder:    ManifestBuilder{Kind: "debootstrap", Version: "1.0"},
		Base:       ManifestBase{Release: "noble", Arch: "amd64", Mirror: "https://archive.ubuntu.com/ubuntu"},
		RootFS:     ManifestRootFS{User: RootFSUser{Name: "enode", UID: 1000, GID: 1000}, Locale: "en_US.UTF-8"},
		Packages:   []ManifestPackage{{Name: "git", Version: "2"}, {Name: "gcc", Version: "1"}},
		VerifiedAt: time.Unix(1, 0),
	}
	one, err := base.Identity()
	if err != nil {
		t.Fatal(err)
	}
	base.VerifiedAt = time.Unix(2, 0)
	base.Packages[0], base.Packages[1] = base.Packages[1], base.Packages[0]
	two, err := base.Identity()
	if err != nil {
		t.Fatal(err)
	}
	if one != two {
		t.Fatalf("volatile time or package order changed identity: %s %s", one, two)
	}
}

func TestManifestRoundTripVerifiesIdentity(t *testing.T) {
	m := Manifest{
		Schema:  ManifestSchema,
		Profile: ManifestProfile{Name: "p", SHA256: "x"},
		Builder: ManifestBuilder{Kind: "debootstrap", Version: "1"},
		Base:    ManifestBase{Release: "noble", Arch: "amd64"},
		RootFS:  ManifestRootFS{User: RootFSUser{Name: "enode", UID: 1000, GID: 1000}, Locale: "C"},
	}
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := WriteManifest(path, m); err != nil {
		t.Fatal(err)
	}
	got, err := ReadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.PreparedEnvironmentID == "" {
		t.Fatal("identity was not written")
	}
}
