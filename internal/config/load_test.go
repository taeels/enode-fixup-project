package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 설정 파일이 아예 없어도 죽지 않는다 — 기본값으로 돈다.
//
// 그리고 못 찾았을 때 **어디를 찾아봤는지**를 함께 낸다. 예전에는 조용히
// 기본값으로 갔다가 다음 줄에서 "토큰이 없다" 로 죽었고, 진짜 원인
// ("설정을 못 찾았다")을 아무도 말해주지 않아 사람이 토큰만 들여다봤다.
func TestResolve_ReportsWhereItLookedWhenItFindsNothing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("ENODE_MEDIATOR_CONFIG", "")

	path, tried, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve returned an error when simply finding nothing: %v", err)
	}
	if path != "" {
		t.Fatalf("path=%q, want empty — nothing was placed anywhere", path)
	}
	if len(tried) == 0 {
		t.Fatal("nothing was found and no places were reported; the caller cannot say where it looked")
	}
	for _, p := range tried {
		if !filepath.IsAbs(p) {
			t.Errorf("%q is not absolute, so the message cannot tell a person where to put the file", p)
		}
	}

	// 그 상태로 Load 하면 기본값이 온다.
	c, err := Load("")
	if err != nil {
		t.Fatalf("Load failed with no config file present: %v", err)
	}
	if c.Listen != Default().Listen || c.Contract.MaxVersions != Default().Contract.MaxVersions {
		t.Fatalf("Load did not fall back to the defaults: %+v", c)
	}
}

// 사용자 자리가 시스템 자리를 이긴다 (ADR-007 D3) — 그래야 발표자 노트북에서
// sudo 없이 뜬다.
func TestResolve_PicksTheUserPathThatExists(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ENODE_MEDIATOR_CONFIG", "")

	want := filepath.Join(home, ".config", "enode-mediator", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(want), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(want, []byte("listen: \":9999\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	path, tried, err := Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if path != want {
		t.Fatalf("path=%q, want %q — the user path must win over the system one", path, want)
	}
	if len(tried) != 0 {
		t.Fatalf("tried=%v, want empty — the places looked at are only reported on a miss", tried)
	}
	// Paths 가 정본이라는 것 — 찾는 쪽과 알려주는 쪽이 같은 목록을 본다.
	if ps := Paths(); len(ps) == 0 || ps[0] != want {
		t.Fatalf("Paths()[0]=%v, want %q; the finder and the advertiser disagree", ps, want)
	}
}

// ADR-015 §4 의 우선순위: --config > $ENODE_MEDIATOR_CONFIG > Paths().
func TestResolve_HonoursThePrecedenceOrder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	userPath := filepath.Join(home, ".config", "enode-mediator", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(userPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userPath, []byte("listen: \":1\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(t.TempDir(), "from-env.yaml")
	flagPath := filepath.Join(t.TempDir(), "from-flag.yaml")

	t.Setenv("ENODE_MEDIATOR_CONFIG", envPath)

	// 환경변수가 발견 경로를 이긴다.
	got, _, err := Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if got != envPath {
		t.Fatalf("path=%q, want %q — $ENODE_MEDIATOR_CONFIG must beat the search paths", got, envPath)
	}

	// --config 가 환경변수를 이긴다.
	got, tried, err := Resolve(flagPath)
	if err != nil {
		t.Fatal(err)
	}
	if got != flagPath {
		t.Fatalf("path=%q, want %q — --config must beat $ENODE_MEDIATOR_CONFIG", got, flagPath)
	}
	if len(tried) != 0 {
		t.Fatalf("tried=%v, want empty — an explicit path is not a search", tried)
	}
}

// 명시한 설정 파일이 없으면 조용히 넘어가지 않는다.
//
// Resolve 의 주석이 적은 그대로다 — "명시했으면 없을 때 조용히 넘어가지
// 않는다". 오타가 기본값으로 도는 것보다 죽는 편이 낫다.
func TestLoad_NamedFileThatIsMissingIsAnError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.yaml")

	_, err := Load(missing)
	if err == nil {
		t.Fatal("a --config path that does not exist was silently ignored")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("the error does not name the file it could not read: %v", err)
	}
	if !strings.Contains(err.Error(), "config ") {
		t.Errorf("the error does not say this is about the config file: %v", err)
	}
}

// 망가진 YAML 은 파싱 오류를 그 파일 이름과 함께 낸다.
func TestLoad_BrokenYAMLNamesTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("listen: \":8080\"\ndatabase: [unclosed\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("a malformed config parsed without complaint")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("the error does not name the offending file: %v", err)
	}
}

// 비밀은 환경변수가 파일을 이긴다 (ADR-015 §4) — 파일에 DB 비밀번호를
// 안 적을 수 있어야 한다.
func TestLoad_EnvironmentBeatsTheFileForSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	body := "listen: \":8080\"\ntoken: \"from-file\"\n\ndatabase:\n  url: \"postgres:///from-file\"\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	// 환경변수가 없으면 파일 값이다.
	t.Setenv("DATABASE_URL", "")
	t.Setenv("ENODE_MEDIATOR_TOKEN", "")
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Token != "from-file" || c.Database.URL != "postgres:///from-file" {
		t.Fatalf("the file values did not land: token=%q url=%q", c.Token, c.Database.URL)
	}

	// 환경변수가 있으면 그것이 이긴다.
	t.Setenv("DATABASE_URL", "postgres:///from-env")
	t.Setenv("ENODE_MEDIATOR_TOKEN", "from-env")
	c, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Database.URL != "postgres:///from-env" {
		t.Errorf("Database.URL = %q, want %q — the environment must win", c.Database.URL, "postgres:///from-env")
	}
	if c.Token != "from-env" {
		t.Errorf("Token = %q, want %q — the environment must win", c.Token, "from-env")
	}
	// 비밀이 아닌 항목은 여전히 파일이 정한다.
	if c.Listen != ":8080" {
		t.Errorf("Listen = %q, want %q — only the secrets are overridden", c.Listen, ":8080")
	}
}

// 남이 읽을 수 있는 설정은 **경고**지 실패가 아니다 (ADR-015 §4).
//
// MVP 는 비밀 관리 부품을 만들지 않기로 했다. 여기서 죽이면 이미 깔려서
// 도는 Mediator 가 업그레이드에서 안 뜬다.
func TestLoad_WorldReadableConfigWarnsButStillLoads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("listen: \":8080\"\ntoken: \"t\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}

	c, err := Load(path)
	if err != nil {
		t.Fatalf("a world-readable config was rejected instead of warned about: %v", err)
	}
	if c.Token != "t" {
		t.Fatalf("Token = %q, want %q — the file was not actually read", c.Token, "t")
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o644 {
		t.Fatalf("Load changed the mode to %04o; it warns, it does not repair", fi.Mode().Perm())
	}
}

// 기본 산출물 자리는 절대 경로이고, 이 플랫폼이 실제로 가진 자리를 가리킨다.
//
// 윈도우에는 /var/lib 가 없다 — 기본값이 유닉스만 알면 깔리기는 하고
// 뜨지는 않는다.
func TestDefaultArtifactsRoot_IsAbsoluteAndPlatformShaped(t *testing.T) {
	root := defaultArtifactsRoot()
	if !filepath.IsAbs(root) {
		t.Fatalf("%q is not absolute", root)
	}
	if !strings.Contains(root, "enode-mediator") {
		t.Fatalf("%q does not sit under an enode-mediator directory", root)
	}
	if Default().Artifacts.Root != root {
		t.Fatalf("Default().Artifacts.Root = %q, want %q", Default().Artifacts.Root, root)
	}
	// 기본 상한도 함께 선다 — 0 이면 아무것도 못 올린다.
	if Default().Artifacts.MaxBlobBytes <= 0 {
		t.Fatalf("MaxBlobBytes = %d, want a positive default", Default().Artifacts.MaxBlobBytes)
	}
}
