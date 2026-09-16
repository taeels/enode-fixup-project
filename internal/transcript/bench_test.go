package transcript

import (
	"strings"
	"testing"
	"unsafe"
)

// Event 하나의 크기를 못 박는다.
//
// 이 수가 예산의 곱셈에 들어간다 — 10 MiB 입력이 최악에서 사건 약 50 만 개를
// 내고, 그 배열만 224 바이트 x 50 만 = 약 112 MB 다. 필드를 하나 더하면 이
// 시험이 빨개지고, 빨개지는 것이 값이다: 화면이 아니라 파서가 그 곱셈을 진다.
//
// 재는 것이 amd64 다. 다른 정렬의 기계에서는 이 수가 다르다.
func TestEvent_StaysTheSizeTheBudgetAssumed(t *testing.T) {
	const want = 224
	if got := unsafe.Sizeof(Event{}); got != want {
		t.Fatalf("unsafe.Sizeof(Event{}) is %d, want %d — the memory budget in "+
			"nfr-requirements 5.1 was computed from %d", got, want, want)
	}
}

// 최악의 사건 배열은 곱셈으로 적고 안 돌린다.
//
// 10 MiB 입력을 벤치마크로 돌리면 CI 가 아니라 사람의 기계가 멈춘다.
// 그 수는 code-summary.md 가 든다.

// benchInput 은 실측 모양의 줄을 섞어 한 뭉치로 짓는다.
func benchInput(n int) []byte {
	lines := []string{
		line("init.json"),
		line("assistant-tool.json"),
		line("assistant-text.json"),
		line("user-result.json"),
		line("plain.txt"),
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(lines[i%len(lines)])
		b.WriteByte('\n')
	}
	b.WriteString(line("result.json"))
	b.WriteByte('\n')
	return []byte(b.String())
}

// BenchmarkParse 는 증폭의 수를 잰다 — B/op 가 입력의 몇 배인가.
//
// -benchmem 으로 돌린다:
//
//	go test ./internal/transcript -run=XXX -bench=BenchmarkParse -benchmem
func BenchmarkParse(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		in := benchInput(n)
		b.Run(strings.Repeat("x", 0)+itoa(n)+" lines", func(b *testing.B) {
			b.SetBytes(int64(len(in)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = Parse(in, false)
			}
		})
	}
}

// BenchmarkShell 은 짓는 쪽의 비용이다 — selectLogs 가 줄마다 이것을 부른다.
func BenchmarkShell(b *testing.B) {
	obj, typ, ok := ParseLine([]byte(line("assistant-tool.json")))
	if !ok {
		b.Fatalf("the fixture was not read as an event")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Shell(obj, typ)
	}
}

// itoa 는 strconv 를 시험에 들이지 않으려고 둔 한 줄이다.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}
