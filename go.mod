module github.com/taeels/enode

// go       이보다 낮은 툴체인은 빌드를 거절한다 (하한 + 언어 버전)
// toolchain 실제로 쓸 것. GOTOOLCHAIN=auto(기본) 면 없을 때 자동으로 받아온다
//
// 둘을 같은 버전으로 묶어 슬랙을 없앴다 — 넷이 서로 다른 컴파일러로 짜면
// "내 기계에서는 되는데" 가 나온다. 명시적 실패가 조용한 드리프트보다 낫다
go 1.26

toolchain go1.26.6

require (
	github.com/jackc/pgx/v5 v5.10.0
	golang.org/x/sys v0.47.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/rogpeppe/go-internal v1.16.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/text v0.39.0 // indirect
)
