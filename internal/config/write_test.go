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

// artifacts 블록이 없으면 더한다 — 그리고 쓴 것이 다시 읽힌다.
func TestSetArtifactsRoot_AddsTheBlockAndItParsesBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("listen: \":8080\"\ntoken: \"t\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(t.TempDir(), "artifacts")

	if err := SetArtifactsRoot(path, want); err != nil {
		t.Fatalf("could not add the artifacts block: %v", err)
	}

	c, err := Load(path)
	if err != nil {
		t.Fatalf("what we wrote does not parse back: %v", err)
	}
	if c.Artifacts.Root != want {
		t.Fatalf("Artifacts.Root = %q, want %q", c.Artifacts.Root, want)
	}
	// 있던 값은 그대로다 — 블록을 더하는 것이지 파일을 다시 짓는 것이 아니다.
	if c.Token != "t" || c.Listen != ":8080" {
		t.Fatalf("the existing values were lost: token=%q listen=%q", c.Token, c.Listen)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode %04o, want 0600 — the same file carries database credentials", fi.Mode().Perm())
	}
}

// 이미 있으면 손대지 않는다 — 사람이 고른 자리를 덮으면 기존 기록을 잃는다.
func TestSetArtifactsRoot_LeavesAnExistingRootAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	body := "token: \"t\"\n\nartifacts:\n  root: \"/srv/keep-me\"\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := SetArtifactsRoot(path, "/tmp/somewhere-else"); err != nil {
		t.Fatal(err)
	}

	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Artifacts.Root != "/srv/keep-me" {
		t.Fatalf("Artifacts.Root = %q, want %q — an existing root was overwritten", c.Artifacts.Root, "/srv/keep-me")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(b), "artifacts:") != 1 {
		t.Fatalf("the artifacts block was duplicated: %s", b)
	}
}

// 줄바꿈으로 안 끝나는 파일에 이어 붙여도 YAML 이 깨지지 않는다.
//
// 사람이 손으로 고친 설정은 마지막 줄바꿈이 없기 쉽다. 그대로 이어 붙이면
// 마지막 키와 새 블록이 한 줄에 붙어 다음 기동이 파싱 오류로 죽는다.
func TestSetters_AppendCleanlyToAFileWithNoTrailingNewline(t *testing.T) {
	// 각 setter 를 **각자의** 파일에 건다 — 한 파일에 이어 걸면 앞의 setter 가
	// 줄바꿈을 남겨 뒤의 setter 는 이 갈래를 아예 안 밟는다.
	newFile := func() string {
		p := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(p, []byte("token: \"t\""), 0o600); err != nil { // 줄바꿈 없음
			t.Fatal(err)
		}
		return p
	}

	root := filepath.Join(t.TempDir(), "artifacts")
	ap := newFile()
	if err := SetArtifactsRoot(ap, root); err != nil {
		t.Fatal(err)
	}
	c, err := Load(ap)
	if err != nil {
		t.Fatalf("SetArtifactsRoot broke the YAML of a file with no trailing newline: %v", err)
	}
	if c.Token != "t" {
		t.Errorf("Token = %q, want %q — the last line was swallowed", c.Token, "t")
	}
	if c.Artifacts.Root != root {
		t.Errorf("Artifacts.Root = %q, want %q", c.Artifacts.Root, root)
	}

	dp := newFile()
	if err := SetDatabaseURL(dp, "postgres:///d"); err != nil {
		t.Fatal(err)
	}
	c, err = Load(dp)
	if err != nil {
		t.Fatalf("SetDatabaseURL broke the YAML of a file with no trailing newline: %v", err)
	}
	if c.Token != "t" {
		t.Errorf("Token = %q, want %q — the last line was swallowed", c.Token, "t")
	}
	if c.Database.URL != "postgres:///d" {
		t.Errorf("Database.URL = %q, want %q", c.Database.URL, "postgres:///d")
	}

	// 그리고 둘을 이어 걸어도 깨지지 않는다.
	both := newFile()
	if err := SetArtifactsRoot(both, root); err != nil {
		t.Fatal(err)
	}
	if err := SetDatabaseURL(both, "postgres:///d"); err != nil {
		t.Fatal(err)
	}
	if c, err = Load(both); err != nil {
		t.Fatalf("the two setters together broke the YAML: %v", err)
	}
	if c.Token != "t" || c.Artifacts.Root != root || c.Database.URL != "postgres:///d" {
		t.Fatalf("a value was lost when both setters ran: %+v", c)
	}
}

// database 블록이 아예 없으면 만들어 준다 — url 만 갈아끼우는 것이 아니다.
func TestSetDatabaseURL_AddsTheBlockWhenThereIsNone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("listen: \":8080\"\ntoken: \"t\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	want := "postgres://u:p@h:5432/d?sslmode=disable"

	if err := SetDatabaseURL(path, want); err != nil {
		t.Fatal(err)
	}

	c, err := Load(path)
	if err != nil {
		t.Fatalf("what we wrote does not parse back: %v", err)
	}
	if c.Database.URL != want {
		t.Fatalf("Database.URL = %q, want %q", c.Database.URL, want)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(b), "database:") != 1 {
		t.Fatalf("the database block was duplicated: %s", b)
	}
}

// token 줄이 아예 없는 설정에도 써넣는다 — 있는 줄을 갈아끼우는 것만이 아니다.
//
// 만들고, 파일에 남기고, 만들었다고 말한다 (ADR-015 의 「조용한 대체를 하지
// 않는다」의 반대편).
func TestEnsureToken_AddsATokenLineWhenThereIsNone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("listen: \":8080\""), 0o600); err != nil { // token 줄 없음, 줄바꿈 없음
		t.Fatal(err)
	}

	tok, err := EnsureToken(path)
	if err != nil {
		t.Fatalf("could not add a token line: %v", err)
	}
	if tok == "" {
		t.Fatal("no token was generated for a file that has no token line at all")
	}

	c, err := Load(path)
	if err != nil {
		t.Fatalf("what we wrote does not parse back: %v", err)
	}
	if c.Token != tok {
		t.Fatalf("Token = %q, want %q — it was announced but not written", c.Token, tok)
	}
	if c.Listen != ":8080" {
		t.Fatalf("Listen = %q, want %q — the last line was swallowed", c.Listen, ":8080")
	}

	// 두 번째는 안 건드린다.
	again, err := EnsureToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if again != "" {
		t.Fatalf("the token it had just written was replaced: %q -> %q", tok, again)
	}
}

// database 블록 밑의 token 은 최상위 token 이 아니다 — 건드리면 안 된다.
func TestEnsureToken_DoesNotTouchAnIndentedTokenKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	body := "database:\n  token: \"belongs-to-the-database\"\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	tok, err := EnsureToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if tok == "" {
		t.Fatal("no top-level token was generated; the indented key was mistaken for one")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `token: "belongs-to-the-database"`) {
		t.Fatalf("the indented token key was overwritten: %s", b)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Token != tok {
		t.Fatalf("Token = %q, want the newly generated %q", c.Token, tok)
	}
}

// 없는 파일에는 안 쓴다 — 어느 자리에 만들지는 setup 이 사람에게 묻는 일이다.
//
// rc8 이 이 자리에서 죽었다. 조용히 만들어 버리면 그 물음이 사라지고,
// 사람이 고른 적 없는 자리에 비밀이 놓인다.
func TestSetters_RefuseAMissingFileRatherThanCreatingIt(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-there.yaml")

	cases := map[string]func() error{
		"EnsureToken":      func() error { _, err := EnsureToken(missing); return err },
		"SetDatabaseURL":   func() error { return SetDatabaseURL(missing, "postgres:///d") },
		"SetArtifactsRoot": func() error { return SetArtifactsRoot(missing, "/srv/a") },
	}
	for name, call := range cases {
		err := call()
		if err == nil {
			t.Errorf("%s silently accepted a file that does not exist", name)
			continue
		}
		if !strings.Contains(err.Error(), missing) {
			t.Errorf("%s: the error does not name the file: %v", name, err)
		}
		if _, statErr := os.Stat(missing); statErr == nil {
			t.Errorf("%s created the file instead of refusing", name)
		}
	}
}

// 상위 자리가 파일에 막혀 있으면 실패로 말한다 — 조용히 아무것도 안 하지 않는다.
//
// writeSecret 은 임시 파일에 쓰고 옮긴다. 그 앞의 MkdirAll 이 막히면
// 여기서 멈춰야 하고, 그 사실이 호출자에게 올라가야 한다.
func TestCreate_FailsLoudlyWhenTheParentPathIsBlockedByAFile(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(blocker, "config.yaml")

	err := Create(path)
	if err == nil {
		t.Fatal("creating a config under a path blocked by a regular file reported success")
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Fatal("the config file exists even though creation failed")
	}
	// 막은 파일은 그대로다.
	b, readErr := os.ReadFile(blocker)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(b) != "not a directory\n" {
		t.Fatalf("the blocking file was modified: %q", b)
	}
}
