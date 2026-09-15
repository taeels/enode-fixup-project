package enode

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

// 이 파일은 readPack 의 규칙을 잰다 (U5 · business-rules R3 ~ R16).
//
// 디스크를 안 쓴다 — readPack 이 io.Reader 하나를 받으므로 악성 tar 를
// 메모리에서 짓는다. 그것이 이 시그니처의 값이고, 거부가 파일을 남기기 전에
// 일어난다는 SEC-A 를 시험이 그대로 밟는다.

// packEntry 는 tar 항목 하나의 재료다.
type packEntry struct {
	name string
	body string
	typ  byte // 0 이면 정규 파일
	link string
}

func packTar(t *testing.T, entries ...packEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		typ := e.typ
		if typ == 0 {
			typ = tar.TypeReg
		}
		h := &tar.Header{Name: e.name, Mode: 0o644, Typeflag: typ, Linkname: e.link}
		if typ == tar.TypeReg {
			h.Size = int64(len(e.body))
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if typ == tar.TypeReg && e.body != "" {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func packGzip(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(b); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// wideLimits 는 규칙을 재는 시험이 상한에 안 걸리게 하는 값이다.
var wideLimits = PackLimits{MaxBytes: 1 << 20, MaxFiles: 64}

func mustReadPack(t *testing.T, raw []byte, lim PackLimits) (*Pack, []string) {
	t.Helper()
	p, notes, err := readPack("pack", bytes.NewReader(raw), lim)
	if err != nil {
		t.Fatalf("readPack: %v", err)
	}
	return p, notes
}

func readPackErr(t *testing.T, raw []byte, lim PackLimits) string {
	t.Helper()
	p, _, err := readPack("pack", bytes.NewReader(raw), lim)
	if err == nil {
		t.Fatalf("this pack was accepted: %+v", p)
	}
	if p != nil {
		t.Fatalf("a rejected pack came back with a value: %+v", p)
	}
	return err.Error()
}

// 규약 안 셋을 담고 이름 순으로 낸다 (R11 · R12).
func TestPack_ReadsTheConvention(t *testing.T) {
	raw := packTar(t,
		packEntry{name: "skills/hello/SKILL.md", body: "hello"},
		packEntry{name: "skills/hello/notes.md", body: "side file"},
		packEntry{name: "agents/helper.md", body: "helper"},
		packEntry{name: "mcp.json", body: `{"mcpServers":{"probe4":{"command":"true","type":"sse"}}}`},
	)
	p, notes := mustReadPack(t, raw, wideLimits)

	want := []string{"agents/helper.md", "skills/hello/SKILL.md", "skills/hello/notes.md"}
	var got []string
	for _, f := range p.Files {
		got = append(got, f.Name)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("files = %v, want %v", got, want)
	}
	if len(notes) != 0 {
		t.Fatalf("a clean pack left notes: %v", notes)
	}
	// 원문 그대로다 — 우리 어휘에 없는 키가 안 사라진다.
	if p.MCP["probe4"]["type"] != "sse" {
		t.Fatalf("mcp.json was translated: %v", p.MCP)
	}
	if p.SHA256 != sha256Hex(raw) {
		t.Fatalf("sha256 = %s, want %s", p.SHA256, sha256Hex(raw))
	}
}

// gzip 을 투명하게 푼다. 해시는 받은 바이트의 것이다 (R3 · 2.2).
func TestPack_AcceptsGzip(t *testing.T) {
	plain := packTar(t, packEntry{name: "skills/hello/SKILL.md", body: "hello"})
	raw := packGzip(t, plain)
	p, _ := mustReadPack(t, raw, wideLimits)

	if len(p.Files) != 1 || string(p.Files[0].Data) != "hello" {
		t.Fatalf("gzip was not peeled: %+v", p.Files)
	}
	// 압축된 원본의 값이다 — 봉인을 읽는 사람이 blob 을 받아 맞출 수 있어야 한다.
	if p.SHA256 != sha256Hex(raw) {
		t.Fatalf("sha256 is of the expanded bytes: %s", p.SHA256)
	}
	if p.SHA256 == sha256Hex(plain) {
		t.Fatal("sha256 is of the tar inside the gzip")
	}
}

// 꼬리의 0 블록 뒤까지 읽어야 해시가 파일 전체의 값이다 (2.2).
//
// 쓰레기를 bufio 의 버퍼보다 크게 붙인다 — 작으면 미리 읽기가 이미 해시에
// 흘려 버려서 드레인을 걷어도 이 시험이 안 빨개진다.
func TestPack_HashCoversTheWholeFile(t *testing.T) {
	plain := packTar(t, packEntry{name: "skills/hello/SKILL.md", body: "hello"})
	raw := append(append([]byte{}, plain...), bytes.Repeat([]byte("z"), 32<<10)...)

	p, _ := mustReadPack(t, raw, wideLimits)
	if p.SHA256 != sha256Hex(raw) {
		t.Fatalf("sha256 stopped at the end-of-archive marker: %s", p.SHA256)
	}
}

// tar 가 아닌 것은 거절이고 문구가 팩 이름을 담는다 (R3).
func TestPack_RejectsWhatIsNotATar(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  []byte
	}{
		{"plain garbage", bytes.Repeat([]byte("nope"), 400)},
		{"gzip of garbage", packGzip(t, bytes.Repeat([]byte("nope"), 400))},
		{"gzip of gzip", packGzip(t, packGzip(t, packTar(t,
			packEntry{name: "skills/a/SKILL.md", body: "a"})))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if msg := readPackErr(t, tc.raw, wideLimits); msg != "pack pack is not a tar archive" {
				t.Fatalf("msg = %q", msg)
			}
		})
	}
}

// 종류를 이름보다 먼저 본다 — 성해 보이는 이름의 링크가 살아 나가면 안 된다 (R4).
func TestPack_RejectsEntryKinds(t *testing.T) {
	for _, tc := range []struct {
		typ  byte
		want string
	}{
		{tar.TypeSymlink, "symlink"},
		{tar.TypeLink, "hard link"},
		{tar.TypeChar, "device"},
		{tar.TypeBlock, "device"},
		{tar.TypeFifo, "fifo"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			raw := packTar(t, packEntry{
				name: "skills/hello/SKILL.md", typ: tc.typ, link: "SKILL.md"})
			msg := readPackErr(t, raw, wideLimits)
			want := "pack entry skills/hello/SKILL.md is a " + tc.want +
				"; only regular files are read"
			if msg != want {
				t.Fatalf("msg = %q, want %q", msg, want)
			}
		})
	}
}

// 뿌리를 벗어나거나 플랫폼마다 다르게 펴질 이름은 거절이다 (R5 ~ R7).
func TestPack_RejectsUnsafeNames(t *testing.T) {
	for _, tc := range []struct{ name, entry, want string }{
		{"absolute", "/etc/passwd", "pack entry /etc/passwd has an absolute path"},
		{"dotdot", "skills/../../x", "pack entry skills/../../x escapes the pack root"},
		{"bare dotdot", "..", "pack entry .. escapes the pack root"},
		{"backslash", `skills\hello\SKILL.md`, `pack entry skills\hello\SKILL.md has an unsafe name`},
		{"control char", "skills/he\x01lo/SKILL.md", "pack entry skills/he\x01lo/SKILL.md has an unsafe name"},
		{"drive letter", "C:/skills/x", "pack entry C:/skills/x has an unsafe name"},
		{"dot", ".", "pack entry . has an unsafe name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := packTar(t, packEntry{name: tc.entry, body: "x"})
			if msg := readPackErr(t, raw, wideLimits); msg != tc.want {
				t.Fatalf("msg = %q, want %q", msg, tc.want)
			}
		})
	}
}

// 받은 바이트와 푼 바이트 중 먼저 닿는 쪽에서 끊는다 (R8).
func TestPack_RejectsOverTheSizeLimit(t *testing.T) {
	big := packTar(t, packEntry{name: "skills/hello/SKILL.md", body: strings.Repeat("x", 4096)})

	t.Run("received bytes", func(t *testing.T) {
		lim := PackLimits{MaxBytes: 512, MaxFiles: 64}
		if msg := readPackErr(t, big, lim); msg != "pack exceeds the size limit of 512 bytes" {
			t.Fatalf("msg = %q", msg)
		}
	})

	// gzip 폭탄 — 받은 바이트로는 안 걸리고 푼 바이트로 걸린다. 이 갈래가
	// 없으면 압축된 폭탄이 상한을 그대로 지나간다.
	t.Run("expanded bytes", func(t *testing.T) {
		bomb := packGzip(t, packTar(t, packEntry{
			name: "skills/hello/SKILL.md", body: strings.Repeat("\x00", 1<<20)}))
		lim := PackLimits{MaxBytes: 64 << 10, MaxFiles: 64}
		if int64(len(bomb)) > lim.MaxBytes {
			t.Fatalf("the bomb is not compressed enough to isolate the expanded path: %d", len(bomb))
		}
		if msg := readPackErr(t, bomb, lim); msg != "pack exceeds the size limit of 65536 bytes" {
			t.Fatalf("msg = %q", msg)
		}
	})

	// 상한과 같은 크기는 통과한다 — 넘은 뒤에만 끊는다.
	t.Run("exactly at the limit", func(t *testing.T) {
		lim := PackLimits{MaxBytes: int64(len(big)), MaxFiles: 64}
		mustReadPack(t, big, lim)
	})
}

// 규약 밖 항목과 디렉터리까지 함께 센다 (R9).
func TestPack_RejectsOverTheFileLimit(t *testing.T) {
	raw := packTar(t,
		packEntry{name: "skills/", typ: tar.TypeDir},
		packEntry{name: "readme.txt", body: "outside"},
		packEntry{name: "skills/hello/SKILL.md", body: "hello"},
	)
	lim := PackLimits{MaxBytes: 1 << 20, MaxFiles: 2}
	if msg := readPackErr(t, raw, lim); msg != "pack exceeds the file limit of 2 entries" {
		t.Fatalf("msg = %q", msg)
	}
}

// 규약 밖은 무시하되 이름을 남긴다. settings.json 은 문구가 다르다 (R13 · R14).
func TestPack_IgnoresWhatIsOutsideTheConvention(t *testing.T) {
	raw := packTar(t,
		packEntry{name: "skills/hello/SKILL.md", body: "hello"},
		packEntry{name: "skills/", typ: tar.TypeDir},
		packEntry{name: "readme.txt", body: "outside"},
		packEntry{name: "settings.json", body: `{"hooks":{}}`},
	)
	p, notes := mustReadPack(t, raw, wideLimits)

	if len(p.Files) != 1 {
		t.Fatalf("something outside the convention was loaded: %+v", p.Files)
	}
	want := []string{
		"pack carries readme.txt, which is outside the pack convention; ignoring it",
		"pack carries settings.json; this run does not read it",
	}
	if strings.Join(notes, "|") != strings.Join(want, "|") {
		t.Fatalf("notes = %v, want %v", notes, want)
	}
}

// 같은 이름이 두 번이면 뒤가 이기고 그 사실이 남는다 (R15).
func TestPack_LastDuplicateWins(t *testing.T) {
	raw := packTar(t,
		packEntry{name: "skills/hello/SKILL.md", body: "first"},
		packEntry{name: "skills/hello/SKILL.md", body: "second"},
	)
	p, notes := mustReadPack(t, raw, wideLimits)

	if len(p.Files) != 1 || string(p.Files[0].Data) != "second" {
		t.Fatalf("the first one won: %+v", p.Files)
	}
	want := "pack carries skills/hello/SKILL.md more than once; the last one wins"
	if len(notes) != 1 || notes[0] != want {
		t.Fatalf("notes = %v", notes)
	}
}

// 깨진 mcp.json 은 거절이고 mcpServers 가 없는 파일은 서버 0 이다 (R12).
func TestPack_MCPJSON(t *testing.T) {
	t.Run("broken", func(t *testing.T) {
		raw := packTar(t, packEntry{name: "mcp.json", body: "{not json"})
		if msg := readPackErr(t, raw, wideLimits); !strings.HasPrefix(
			msg, "pack mcp.json is not valid json: ") {
			t.Fatalf("msg = %q", msg)
		}
	})
	t.Run("no mcpServers key", func(t *testing.T) {
		raw := packTar(t,
			packEntry{name: "skills/hello/SKILL.md", body: "hello"},
			packEntry{name: "mcp.json", body: `{"somethingElse":1}`},
		)
		p, _ := mustReadPack(t, raw, wideLimits)
		if len(p.MCP) != 0 {
			t.Fatalf("mcp = %v", p.MCP)
		}
	})
}

// 실을 것이 0 인 팩은 거절이다 (R16).
//
// 하네스는 없는 플러그인 디렉터리를 종료코드 0 에 stderr 한 줄 없이 무시한다.
// 그 침묵을 우리가 잡는 자리가 이 규칙이다.
func TestPack_RejectsAnEmptyPack(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  []byte
	}{
		{"no entries", packTar(t)},
		// 0 바이트 blob 이 여기로 온다 — tar 는 빈 스트림을 빈 아카이브로
		// 읽으므로 「tar 가 아니다」가 아니라 「실을 것이 0」이다. 둘 다
		// 치명이고, 이 문구가 curl 이 빈 파일을 남긴 경우를 더 곧게 가리킨다.
		{"an empty blob", nil},
		{"only outside the convention", packTar(t,
			packEntry{name: "readme.txt", body: "outside"})},
		{"only an empty mcp.json", packTar(t,
			packEntry{name: "mcp.json", body: `{"mcpServers":{}}`})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			msg := readPackErr(t, tc.raw, wideLimits)
			if msg != "pack pack carries no skills, agents, or mcp servers" {
				t.Fatalf("msg = %q", msg)
			}
		})
	}
}

// 거부가 파일을 남기기 전에 일어난다 (R10 · SEC-A).
func TestPack_RejectionLeavesNothingBehind(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	raw := packTar(t,
		packEntry{name: "skills/hello/SKILL.md", body: "hello"},
		packEntry{name: "evil.txt", body: "outside"},
		packEntry{name: "../escape.txt", body: "escape"},
	)
	if msg := readPackErr(t, raw, wideLimits); msg != "pack entry ../escape.txt escapes the pack root" {
		t.Fatalf("msg = %q", msg)
	}
	left, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Fatalf("readPack wrote something: %v", left)
	}
}
