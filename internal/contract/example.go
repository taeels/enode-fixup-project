package contract

import (
	"embed"
	"fmt"
	"sort"
	"strings"
)

// 예시는 파일로 둔다 — 문자열 상수로 들고 있으면 사람이 그것을 편집할 때
// Go 이스케이프를 통과해야 하고, 그러면 「붙여넣으면 돈다」가 깨지기 쉽다.
//
//go:embed examples/*.json
var examplesFS embed.FS

// Examples 는 붙여넣으면 도는 최소 계약들이다 (ADR-066 §4.1).
//
// 왜 여기 있나 — 이 패키지가 계약 어휘의 원본이고, 예시는 그 어휘로 쓰였다.
// cmd/runctl 에 두면 어휘가 바뀔 때 함께 바뀌어야 한다는 것이 안 보인다.
//
// 왜 시험이 이것을 dry-run 까지 태우나 — example_test.go 가 적는다. 요약하면
// 「문서는 낡아도 아무것도 빨개지지 않는다」이고, 예시를 시험으로 걸어야
// 어휘와 갈릴 수 없다.
func ExampleNames() []string {
	ents, err := examplesFS.ReadDir("examples")
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(ents))
	for _, e := range ents {
		out = append(out, strings.TrimSuffix(e.Name(), ".json"))
	}
	sort.Strings(out)
	return out
}

// Example 은 그 이름의 계약을 JSON 그대로 돌려준다.
//
// 파싱해서 다시 마샬링하지 않는다 — 그러면 사람이 읽으라고 넣은 줄바꿈과
// 순서가 사라지고, 예시의 값이 절반으로 준다.
func Example(name string) ([]byte, error) {
	b, err := examplesFS.ReadFile("examples/" + name + ".json")
	if err != nil {
		return nil, fmt.Errorf("no such example: %s  (have: %s)",
			name, strings.Join(ExampleNames(), " · "))
	}
	return b, nil
}
