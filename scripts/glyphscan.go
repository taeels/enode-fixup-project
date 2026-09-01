//go:build ignore || glyphscan

// Command glyphscan 은 비테스트 Go 파일의 문자열 리터럴에서 장식 문자를 찾는다.
//
//	go run ./scripts/glyphscan.go [-root .]
//
// 왜 이것이 필요한가 — CI 에 이미 있던 장식 문자 검사는 grep -rlIP '\x{2605}'
// 하나였고, 그것은 U+2605 한 글자만 본다. 저장소에 실제로 있던 장식 문자는
// 전부 다른 글자였으므로 그 검사는 고치기 전에도 통과했다. 규약을 집행하는
// 것처럼 보이면서 아무것도 집행하지 않는 항진명제였다.
//
// 왜 주석을 안 보는가 — 실측이 이것을 정했다. 비테스트 Go 파일에서 기호류가
// 나오는 줄이 269줄인데 그중 문자열 리터럴은 18줄뿐이고, 나머지 251줄이
// 전부 설계 논거를 적은 주석이다. 팀 규칙도 표 괘선과 기각 표시 ✗ 와
// 도메인 어휘로 쓰는 원문자를 장식 문자 대상에서 명시적으로 뺐다. 문자열과
// 주석을 못 가르는 검사는 이 저장소에서 251줄을 지적하거나 검사를 포기한다.
//
// 무엇을 기각했나 — grep -P 는 의존이 0이지만 바로 그 251줄을 잡는다.
// 정규식으로 문자열 리터럴을 흉내내는 것도 같은 함정을 다른 모양으로 다시
// 부른다. go/parser 가 낸 AST 에서 *ast.BasicLit 만 걸으면 그 구분이
// 휴리스틱이 아니라 구조로 보장된다.
//
// 왜 빌드 태그가 둘인가 — ignore 는 go build ./... 와 go vet ./... 와
// go test ./... 와 크로스 빌드와 golangci-lint 가 이 파일을 안 보게 한다
// (커버리지 분모에도 안 들어가므로 패키지별 하한의 분모를 이 도구가 늘리지
// 않는다). glyphscan 은 go test -tags glyphscan ./scripts/ 가 이 파일과
// 그 테스트를 함께 보게 한다. gofmt 는 태그와 무관하게 이 파일을 본다.
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// exemptionMarker 는 면제를 얻는 유일한 통로다. 팀 규칙이 표 괘선을 장식
// 문자 대상에서 뺐지만, 빼는 근거를 그 줄에 사람이 손으로 적어야 면제가
// 성립하게 한다. 마커가 없으면 괘선도 보고된다 - 새 기호가 어디로 들어오든
// 사람이 이 문장을 의식하지 않는 한 검사가 빨개진다.
const exemptionMarker = "괘선이라 대상이 아님"

// glyphRanges 는 훑는 구간이다. 가운뎃점 U+00B7 과 줄표 U+2014 가 어느
// 구간에도 안 들어가는 것은 일부러다 - 이 저장소의 출력 문자열이 둘을
// 구분자로 널리 쓰고, 팀 규칙이 그것을 장식이 아니라 구두점으로 본다.
var glyphRanges = [...]struct{ lo, hi rune }{
	{0x2190, 0x21FF},   // 화살표
	{0x2300, 0x23FF},   // 기술 기호
	{0x2460, 0x24FF},   // 원문자
	{0x2500, 0x257F},   // 괘선
	{0x2580, 0x259F},   // 블록
	{0x25A0, 0x25FF},   // 기하 도형
	{0x2600, 0x26FF},   // 기타 기호. U+2605 가 여기 든다
	{0x2700, 0x27BF},   // dingbat
	{0x2B00, 0x2BFF},   // 기호와 화살표
	{0x1F300, 0x1FAFF}, // 이모지
}

// skippedDirs 는 훑기에서 빼는 디렉터리다. testdata 를 빼는 이유는 둘이다 -
// Go 자신이 빌드에서 빼는 자리이고, CLAUDE.md 2.2 가 「testdata 안의 기록은
// 실제로 보냈던 것의 기록이라 고치면 기록이 아니다」로 편집을 금한다.
var skippedDirs = map[string]bool{
	".git":         true,
	"dist":         true,
	"vendor":       true,
	"node_modules": true,
	"testdata":     true,
}

// finding 은 한 줄이 한 기호를 들고 있다는 사실 하나다. 같은 줄의 같은
// 기호가 여러 번 나와도 하나로 접는다 - 판정의 단위가 「이 줄이 이 기호를
// 쓴다」이지 등장 횟수가 아니기 때문이다.
type finding struct {
	file string
	line int
	ch   rune
}

// scanResult 는 지적과 함께 실제로 몇 파일을 읽었는지를 낸다. 훑은 파일 수를
// 같이 내지 않으면 「0건」이 「깨끗하다」와 「아무것도 안 읽었다」 둘 다를
// 뜻하게 되고, 그것이 갈아치우려던 항진명제와 같은 실패다.
type scanResult struct {
	findings []finding
	files    int
}

func isGlyph(r rune) bool {
	for _, g := range glyphRanges {
		if r >= g.lo && r <= g.hi {
			return true
		}
	}
	return false
}

// exemptLines 는 면제 마커를 든 주석이 놓인 줄을 모은다. 마커가 앞줄 doc
// 주석에 있으면 면제가 아니다 - 같은 줄이어야 한다는 것이 합격 기준의
// 문면이고, 그래야 면제가 그 한 자리에만 걸린다.
func exemptLines(fset *token.FileSet, f *ast.File) map[int]bool {
	out := make(map[int]bool)
	for _, group := range f.Comments {
		for _, c := range group.List {
			if !strings.Contains(c.Text, exemptionMarker) {
				continue
			}
			start := fset.Position(c.Pos()).Line
			end := fset.Position(c.End()).Line
			for line := start; line <= end; line++ {
				out[line] = true
			}
		}
	}
	return out
}

// scanFile 은 파일 하나의 문자열 리터럴과 rune 리터럴을 본다.
//
// 원문이 아니라 값을 보는 것이 요점이다. "▲" 로 이스케이프해서 써도
// 터미널에 나가는 것은 같은 글자이므로, 이스케이프가 우회로가 되면 검사가
// 규약이 아니라 표기법을 지키는 것이 된다.
func scanFile(fset *token.FileSet, path, display string) ([]finding, error) {
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", display, err)
	}

	exempt := exemptLines(fset, file)
	seen := make(map[finding]bool)
	var out []finding

	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || (lit.Kind != token.STRING && lit.Kind != token.CHAR) {
			return true
		}

		value, err := strconv.Unquote(lit.Value)
		if err != nil {
			// 파서가 통과시킨 리터럴이라 여기 오는 것은 사실상 없다. 그래도
			// 조용히 건너뛰면 검사에 구멍이 나므로 원문 그대로 훑는다.
			value = lit.Value
		}
		// 백틱 문자열만 값 안의 줄바꿈이 원문의 줄바꿈과 일치한다. 따옴표
		// 문자열은 원문에 날 줄바꿈을 못 담으므로 전부 시작 줄이다.
		raw := strings.HasPrefix(lit.Value, "`")
		start := fset.Position(lit.Pos()).Line

		for i, r := range value {
			if !isGlyph(r) {
				continue
			}
			line := start
			if raw {
				line += strings.Count(value[:i], "\n")
			}
			if exempt[line] {
				continue
			}
			f := finding{file: display, line: line, ch: r}
			if seen[f] {
				continue
			}
			seen[f] = true
			out = append(out, f)
		}
		return true
	})

	return out, nil
}

// scanDir 은 root 아래의 비테스트 Go 파일을 전부 본다. 경로는 root 기준
// 상대경로로 보고해 출력이 실행 위치에 안 흔들리게 한다.
func scanDir(root string) (scanResult, error) {
	var res scanResult
	fset := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}
		if d.IsDir() {
			if path != root && skippedDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}

		name := d.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}

		display := path
		if rel, relErr := filepath.Rel(root, path); relErr == nil {
			display = filepath.ToSlash(rel)
		}

		found, scanErr := scanFile(fset, path, display)
		if scanErr != nil {
			return scanErr
		}
		res.files++
		res.findings = append(res.findings, found...)
		return nil
	})
	if err != nil {
		return scanResult{}, err
	}

	sort.Slice(res.findings, func(i, j int) bool {
		a, b := res.findings[i], res.findings[j]
		if a.file != b.file {
			return a.file < b.file
		}
		if a.line != b.line {
			return a.line < b.line
		}
		return a.ch < b.ch
	})
	return res, nil
}

func main() {
	root := flag.String("root", ".", "directory to scan")
	flag.Parse()

	res, err := scanDir(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "glyphscan: %v\n", err)
		os.Exit(2)
	}

	if len(res.findings) == 0 {
		fmt.Printf("glyphscan: %d files scanned, no decorative glyph in any string literal\n", res.files)
		return
	}

	for _, f := range res.findings {
		fmt.Printf("%s:%d: decorative glyph %q (U+%04X) in a string literal\n", f.file, f.line, f.ch, f.ch)
	}
	fmt.Fprintf(os.Stderr, "\nglyphscan: %d line(s) carry a decorative glyph in a string literal (%d files scanned)\n",
		len(res.findings), res.files)
	fmt.Fprintf(os.Stderr, "replace it with a word, or say on the same line why it stays: %s\n", exemptionMarker)
	os.Exit(1)
}
