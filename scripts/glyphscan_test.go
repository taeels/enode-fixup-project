//go:build glyphscan

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// write 는 t.TempDir() 안에 Go 소스 하나를 놓는다. 저장소의 실제 파일을
// 읽지 않으므로 이 테스트가 소스 수정에 흔들리지 않는다.
func write(t *testing.T, dir, name, src string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o600); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
}

func scan(t *testing.T, dir string) scanResult {
	t.Helper()
	res, err := scanDir(dir)
	if err != nil {
		t.Fatalf("scanDir(%s) returned an error, want nil: %v", dir, err)
	}
	return res
}

// requireFiles 는 「0건」이 「깨끗하다」인지 「아무것도 안 읽었다」인지를
// 가른다. 이것을 안 걸면 훑기가 갈아치우려던 항진명제가 테스트 안에서
// 되살아난다.
func requireFiles(t *testing.T, res scanResult, want int) {
	t.Helper()
	if res.files != want {
		t.Fatalf("scanned %d files, want exactly %d - a vacuous pass is the failure mode this check exists for", res.files, want)
	}
}

func TestScan_FlagsGlyphInStringLiteral(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "out.go", "package p\n"+
		"\n"+
		"func warn() string {\n"+
		"\treturn \"▲ caffeinate is missing\"\n"+
		"}\n")

	res := scan(t, dir)
	requireFiles(t, res, 1)

	if len(res.findings) != 1 {
		t.Fatalf("AC2.3.4: want exactly 1 finding for a glyph in an output string, got %d: %+v", len(res.findings), res.findings)
	}
	got := res.findings[0]
	if got.file != "out.go" || got.line != 4 || got.ch != '▲' {
		t.Errorf("AC2.3.4: got %s:%d %q, want out.go:4 %q", got.file, got.line, got.ch, '▲')
	}
}

func TestScan_IgnoresComments(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "doc.go", "package p\n"+
		"\n"+
		"// 표를 그리는 괘선 ── 은 대상이 아니다.\n"+
		"//\n"+
		"//\t기각한 대안 ✗ 은 일관된 표기법이다\n"+
		"//\t도메인 어휘 ①②③④ 도 마찬가지다\n"+
		"func label() string {\n"+
		"\treturn \"plain text\"\n"+
		"}\n")

	res := scan(t, dir)
	requireFiles(t, res, 1)

	if len(res.findings) != 0 {
		t.Fatalf("AC2.3.2: comments are out of scope, want 0 findings, got %d: %+v", len(res.findings), res.findings)
	}
}

func TestScan_IgnoresTestFiles(t *testing.T) {
	dir := t.TempDir()
	// 두 파일을 함께 둔다. 하나만 두면 「0건」이 「_test.go 를 걸렀다」인지
	// 「훑기가 아예 안 돌았다」인지 구별되지 않는다.
	write(t, dir, "thing_test.go", "package p\n\nvar inTest = \"▲ from a test file\"\n")
	write(t, dir, "thing.go", "package p\n\nvar inProd = \"▲ from product code\"\n")

	res := scan(t, dir)
	requireFiles(t, res, 1)

	if len(res.findings) != 1 {
		t.Fatalf("AC2.3.4: want exactly 1 finding (the non-test file only), got %d: %+v", len(res.findings), res.findings)
	}
	if got := res.findings[0].file; got != "thing.go" {
		t.Errorf("AC2.3.4: reported %s, want thing.go - _test.go must not be scanned", got)
	}
}

func TestScan_ExemptsLineWithMarkerComment(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "log.go", "package p\n"+
		"\n"+
		"func header(n string) string {\n"+
		"\treturn fmt.Sprintf(\"── %s ── recent log\\n\", n) // 괘선이라 대상이 아님\n"+
		"}\n")

	res := scan(t, dir)
	requireFiles(t, res, 1)

	if len(res.findings) != 0 {
		t.Fatalf("AC2.3.4: a same-line marker comment must exempt the line, want 0 findings, got %d: %+v", len(res.findings), res.findings)
	}
}

func TestScan_FlagsExemptGlyphWithoutMarker(t *testing.T) {
	dir := t.TempDir()
	// 마커가 아예 없는 파일 하나와, 비슷하지만 문면이 다른 주석을 단 파일
	// 하나. 면제가 「괘선이니까」가 아니라 「그 문장을 적었으니까」 성립한다.
	write(t, dir, "bare.go", "package p\n\nvar h = \"── recent log\"\n")
	write(t, dir, "loose.go", "package p\n\nvar h = \"── recent log\" // 괘선이다\n")

	res := scan(t, dir)
	requireFiles(t, res, 2)

	if len(res.findings) != 2 {
		t.Fatalf("AC2.3.4: exemption comes only from the exact marker, want 2 findings, got %d: %+v", len(res.findings), res.findings)
	}
	for _, f := range res.findings {
		if f.ch != '─' {
			t.Errorf("AC2.3.4: reported %q at %s:%d, want the box-drawing rune %q", f.ch, f.file, f.line, '─')
		}
	}
}

func TestScan_UnquotesEscapes(t *testing.T) {
	dir := t.TempDir()
	// 값을 본다, 원문이 아니라. 이스케이프로 써도 터미널에 나가는 것은 같은
	// 글자이므로 이스케이프가 우회로가 되면 안 된다.
	write(t, dir, "esc.go", "package p\n"+
		"\n"+
		"var warn = \"\\u25b2 not on PATH\"\n"+
		"var face = \"\\U0001F600 done\"\n")

	res := scan(t, dir)
	requireFiles(t, res, 1)

	if len(res.findings) != 2 {
		t.Fatalf("AC2.3.4: escaped glyphs must be caught by value, want 2 findings, got %d: %+v", len(res.findings), res.findings)
	}
	want := map[rune]int{'▲': 3, '\U0001F600': 4}
	for _, f := range res.findings {
		line, ok := want[f.ch]
		if !ok {
			t.Errorf("AC2.3.4: unexpected rune %q at %s:%d", f.ch, f.file, f.line)
			continue
		}
		if f.line != line {
			t.Errorf("AC2.3.4: rune %q reported at line %d, want line %d", f.ch, f.line, line)
		}
	}
}

func TestScan_HandlesRawAndRuneLiterals(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "raw.go", "package p\n"+
		"\n"+
		"var banner = `line one\n"+
		"line two ▲ here`\n"+
		"\n"+
		"var mark = '▲'\n")

	res := scan(t, dir)
	requireFiles(t, res, 1)

	if len(res.findings) != 2 {
		t.Fatalf("AC2.3.4: raw strings and rune literals are both in scope, want 2 findings, got %d: %+v", len(res.findings), res.findings)
	}
	// 백틱 문자열의 줄 번호는 리터럴 시작 줄이 아니라 기호가 실제로 놓인
	// 줄이어야 한다. 그래야 면제 주석이 그 한 줄에만 걸린다.
	if got := res.findings[0]; got.line != 4 {
		t.Errorf("AC2.3.4: raw-string glyph reported at line %d, want line 4 (the line the glyph sits on)", got.line)
	}
	if got := res.findings[1]; got.line != 6 {
		t.Errorf("AC2.3.4: rune literal reported at line %d, want line 6", got.line)
	}
}

func TestScan_AllowsMiddotAndEmDash(t *testing.T) {
	dir := t.TempDir()
	// 같은 파일에 허용 글자 줄과 위반 줄을 함께 둔다. 위반 줄이 잡히는 것이
	// 「이 파일의 리터럴을 실제로 훑었다」는 증거이고, 그 위에서만 허용 줄이
	// 안 잡혔다는 사실이 뜻을 갖는다.
	write(t, dir, "sep.go", "package p\n"+
		"\n"+
		"var joined = \"needs · a — separator\"\n"+
		"var warned = \"▲ but this one is decoration\"\n")

	res := scan(t, dir)
	requireFiles(t, res, 1)

	if len(res.findings) != 1 {
		t.Fatalf("AC2.3.4: middot and em dash are separators, not decoration, want exactly 1 finding, got %d: %+v", len(res.findings), res.findings)
	}
	got := res.findings[0]
	if got.line != 4 || got.ch != '▲' {
		t.Errorf("AC2.3.4: got %s:%d %q, want sep.go:4 %q", got.file, got.line, got.ch, '▲')
	}
}
