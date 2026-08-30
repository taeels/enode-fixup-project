//go:build !darwin

package main

import "testing"

// 짝 파일 caffeinate_other.go 가 약속하는 것 하나 - 맥이 아닌 기계에서
// keepAwake 는 아무것도 하지 않는다.
//
// 왜 이것을 시험하는가. 이 no-op 은 커버리지 분모에 문장을 0개 더하므로
// 커버리지로는 있는지 없는지도 안 보인다. 그런데 여기서 무언가 하기
// 시작하면(경고를 찍는다든가) cmdStart 의 출력이 플랫폼마다 갈리고,
// 그것을 아무도 못 잡는다. 그래서 "아무것도 안 한다" 를 단언으로 둔다.
func TestKeepAwake_DoesNothingOffDarwin(t *testing.T) {
	stdout, stderr := captureOutput(t, func() { keepAwake(4242) })
	if stdout != "" || stderr != "" {
		t.Fatalf("keepAwake off darwin wrote %q / %q, want silence - "+
			"sleep is held off by caffeinate, which only exists on macOS", stdout, stderr)
	}
}
