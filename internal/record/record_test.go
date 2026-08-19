package record

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newStore 는 봉인된 디렉터리를 t.TempDir 이 지울 수 있게 정리를 걸어둔다.
// ★ 봉인은 삭제까지 막는다 ★ — 그 자체가 I4 가 작동한다는 증거다.
func newStore(t *testing.T) *Store {
	t.Helper()
	root := t.TempDir()
	t.Cleanup(func() { _ = unseal(root) })
	return New(root)
}

func steps() []StepFile {
	return []StepFile{
		{Seq: 1, StepID: "r#01", Name: "build", Uses: "builder", Kind: "run",
			Node: "8d73234fac52", NodeLabel: "taeels@box:ws-a", State: "DONE"},
		{Seq: 2, StepID: "r#02", Name: "observe", Uses: "board", Kind: "run",
			Node: "7a02313b8c10", NodeLabel: "taeels@box:ws-b", State: "DONE"},
	}
}

// ★ I4 — 종료 상태에 이른 Run 의 Record 는 봉인되어 이후 변경되지 않는다 ★
//
// 애플리케이션이 "고치지 않기로 한다" 가 아니라 ★ 파일시스템이 막는다 ★.
// ADR-015 §3 이 Record 를 DB 가 아니라 디렉터리에 둔 논거가 이것이다.
func TestSealMakesItImmutable(t *testing.T) {
	s := newStore(t)
	if err := s.Seal("r", map[string]any{"run_id": "r"}, map[string]any{"state": "SUCCEEDED"}, steps()); err != nil {
		t.Fatal(err)
	}
	if !s.Sealed("r") {
		t.Fatal("봉인 표시가 안 됐다")
	}
	d := s.dir("r")

	// 덮어쓰기가 막혀야 한다
	if err := os.WriteFile(filepath.Join(d, "verdict.json"), []byte("tampered"), 0o644); err == nil {
		t.Fatal("★ I4 위반 ★ 봉인된 파일이 덮어써졌다")
	}
	// 새 파일 주입도 막혀야 한다
	if err := os.WriteFile(filepath.Join(d, "steps", "03-injected.json"), []byte("{}"), 0o644); err == nil {
		t.Fatal("★ I4 위반 ★ 봉인된 디렉터리에 파일이 주입됐다")
	}
	// 읽기는 되어야 한다 — 봉인은 잠그는 것이지 숨기는 것이 아니다
	if _, err := os.ReadFile(filepath.Join(d, "manifest.json")); err != nil {
		t.Fatalf("봉인된 것을 못 읽는다: %v", err)
	}
}

// 봉인은 한 번뿐이고 멱등이다.
func TestSealIsIdempotent(t *testing.T) {
	s := newStore(t)
	m := map[string]any{"run_id": "r", "state": "SUCCEEDED"}
	if err := s.Seal("r", m, map[string]any{"state": "SUCCEEDED"}, steps()); err != nil {
		t.Fatal(err)
	}
	// 두 번째 봉인이 내용을 바꾸면 안 된다
	if err := s.Seal("r", map[string]any{"run_id": "r", "state": "TAMPERED"},
		map[string]any{"state": "TAMPERED"}, nil); err != nil {
		t.Fatalf("두 번째 봉인이 에러를 냈다: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(s.dir("r"), "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "TAMPERED") {
		t.Fatal("★ 봉인된 것이 다시 쓰였다 ★")
	}
}

// ★ 성질 3 — 노드 귀속 ★ Case D 의 "서로 다른 기계였다" 가 여기 걸린다.
func TestStepsCarryNodeAttribution(t *testing.T) {
	s := newStore(t)
	if err := s.Seal("r", map[string]any{}, map[string]any{}, steps()); err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, f := range []string{"01-build.json", "02-observe.json"} {
		b, err := os.ReadFile(filepath.Join(s.dir("r"), "steps", f))
		if err != nil {
			t.Fatal(err)
		}
		var sf StepFile
		if err := json.Unmarshal(b, &sf); err != nil {
			t.Fatal(err)
		}
		if sf.Node == "" || sf.NodeLabel == "" {
			t.Fatalf("★ 노드 귀속이 없다 ★: %s", f)
		}
		seen[sf.Node] = sf.NodeLabel
	}
	if len(seen) != 2 {
		t.Fatalf("★ O1 ★ 서로 다른 노드가 %d 개 — 2 여야 한다", len(seen))
	}
}

// ★ 성질 4 — 자기충족 ★ 묶음 하나를 풀면 그 안에 전부 있다.
func TestTarIsSelfSufficient(t *testing.T) {
	s := newStore(t)
	if err := s.Open("r"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendLog("r", 1, "build", strings.NewReader("  CC foo.o\n"), 1<<20); err != nil {
		t.Fatal(err)
	}
	if err := s.Seal("r", map[string]any{"run_id": "r"}, map[string]any{"state": "SUCCEEDED"}, steps()); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := s.Tar("r", &buf); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"run-r/manifest.json": false, "run-r/verdict.json": false,
		"run-r/steps/01-build.json": false, "run-r/logs/01-build.log": false,
	}
	tr := tar.NewReader(&buf)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := want[h.Name]; ok {
			want[h.Name] = true
		}
	}
	for name, got := range want {
		if !got {
			t.Fatalf("★ 자기충족 위반 ★ 묶음에 %s 가 없다", name)
		}
	}
}

// 봉인되지 않은 것은 Record 가 아니다 (I4).
func TestTarRefusesUnsealed(t *testing.T) {
	s := newStore(t)
	if err := s.Open("r"); err != nil {
		t.Fatal(err)
	}
	if err := s.Tar("r", io.Discard); err != ErrNotSealed {
		t.Fatalf("err=%v 기대 ErrNotSealed", err)
	}
}

// 로그는 상한을 넘으면 잘라 저장하고 ★ 잘렸음을 표시한다 ★ (ADR-015 §5).
func TestLogTruncationIsMarked(t *testing.T) {
	s := newStore(t)
	if err := s.Open("r"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendLog("r", 1, "big", strings.NewReader(strings.Repeat("x", 100)), 10); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(s.dir("r"), "logs", "01-big.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "잘렸") {
		t.Fatalf("잘렸다는 표시가 없다: %q", b)
	}
}

// run_id 로 경로를 탈출할 수 없다.
func TestPathTraversalIsBlocked(t *testing.T) {
	root := t.TempDir()
	t.Cleanup(func() { _ = unseal(root) })
	s := New(root)
	if err := s.Open("../../etc/evil"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "..", "..", "etc", "evil")); err == nil {
		t.Fatal("★ 경로를 탈출했다 ★")
	}
}
