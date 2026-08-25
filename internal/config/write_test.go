package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// rc8 이 밟은 자리다 — 상위 디렉터리가 없으면 "no such file or directory" 로
// 죽었다. setup 이 처음 부르는 경로이므로 그 자리가 없는 것이 정상이다.
func TestCreate_MakesTheParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deep", "deeper", "config.yaml")
	if err := Create(path); err != nil {
		t.Fatalf("could not create through a missing directory: %v", err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("the file is not there: %v", err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode %04o, want 0600 — it carries database credentials", fi.Mode().Perm())
	}
}

func TestCreate_DoesNotOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("token: keep-me\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Create(path); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), "keep-me") {
		t.Fatalf("an existing config was overwritten: %s", b)
	}
}

func TestEnsureToken_FillsAnEmptyOneAndKeepsAnExistingOne(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := Create(path); err != nil {
		t.Fatal(err)
	}

	tok, err := EnsureToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if tok == "" {
		t.Fatal("no token was generated for an empty one")
	}
	if strings.ContainsAny(tok, "/+=") {
		t.Fatalf("token %q carries a character that changes meaning in YAML, URLs or a shell", tok)
	}

	// 두 번째는 안 건드린다 — 노드가 이미 그 값을 들고 있다.
	again, err := EnsureToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if again != "" {
		t.Fatalf("an existing token was replaced: %q -> %q", tok, again)
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), tok) {
		t.Fatalf("the token is gone from the file: %s", b)
	}
}

func TestSetDatabaseURL_ReplacesTheValueInPlace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := Create(path); err != nil {
		t.Fatal(err)
	}
	want := "postgres://u:p@h:5432/d?sslmode=disable"
	if err := SetDatabaseURL(path, want); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if strings.Count(string(b), "database:") != 1 {
		t.Fatalf("the database block was duplicated: %s", b)
	}
	if !strings.Contains(string(b), want) {
		t.Fatalf("the url did not land: %s", b)
	}

	c, err := Load(path)
	if err != nil {
		t.Fatalf("what we wrote does not parse back: %v", err)
	}
	if c.Database.URL != want {
		t.Fatalf("Database.URL = %q, want %q", c.Database.URL, want)
	}
}

// Paths 가 정본이다 — 예시 파일과 패키지가 이 목록을 가리킨다.
func TestPaths_AreAbsoluteAndOrdered(t *testing.T) {
	ps := Paths()
	if len(ps) < 2 {
		t.Fatalf("want a user path and a system path: %v", ps)
	}
	for _, p := range ps {
		if !filepath.IsAbs(p) {
			t.Fatalf("%q is not absolute", p)
		}
	}
}
