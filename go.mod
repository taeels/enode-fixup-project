module github.com/taeels/enode

// go       이보다 낮은 툴체인은 ★ 빌드를 거절한다 ★ (하한 + 언어 버전)
// toolchain 실제로 쓸 것. GOTOOLCHAIN=auto(기본) 면 없을 때 ★ 자동으로 받아온다 ★
//
// 둘을 같은 버전으로 묶어 슬랙을 없앴다 — 넷이 서로 다른 컴파일러로 짜면
// "내 기계에서는 되는데" 가 나온다. ★ 명시적 실패가 조용한 드리프트보다 낫다 ★
go 1.26

toolchain go1.26.6
