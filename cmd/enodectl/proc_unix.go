//go:build !windows

package main

import "syscall"

// 프로세스를 띄우는 자리만 여기 남는다 — 살았는지·멈춤·소유 확인은
// internal/proc 로 내렸다(panel 과 공유 · net/http 무의존으로 심볼 상한을 지킨다).
// detachAttr 는 start 가 프로세스를 띄우는 자리라 여기 산다.

// detachAttr 는 부모에서 떼어낸다 — enodectl 이 끝나도 노드는 살아 있어야 한다.
func detachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

// exeSuffix 는 실행 파일 이름에 붙는 것이다.
const exeSuffix = ""
