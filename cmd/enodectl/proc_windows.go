//go:build windows

package main

// 프로세스 프리미티브(살았는지·멈춤·소유·분리 실행)는 internal/proc 로 내렸다
// — cmd/enodectl 과 internal/panel 이 같이 쓴다. 여기 남는 것은 실행 파일
// 이름 접미사뿐이다.

// exeSuffix 는 실행 파일 이름에 붙는 것이다.
const exeSuffix = ".exe"
