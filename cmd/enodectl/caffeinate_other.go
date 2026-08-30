//go:build !darwin

package main

// keepAwake 는 맥 밖에서 아무것도 하지 않는다.
//
// 짝 파일(caffeinate_darwin.go)이 왜 이 쌍으로 갈렸는지를 적는다. 여기가
// 빈 것은 잊어서가 아니라 결론이다 — caffeinate 는 맥에만 있고, 리눅스와
// 윈도우에서 잠자기를 막는 방법은 각각 다른 물건이며(systemd-inhibit ·
// SetThreadExecutionState) 그 둘이 함대를 끊는 것을 실제로 관측한 적이
// 아직 없다. 관측되면 그때 이 자리에 대응하는 구현이 들어온다.
//
// 인자에 이름을 안 붙인 것도 같은 뜻이다 — 받기는 하지만 쓰지 않는다.
func keepAwake(int) {}
