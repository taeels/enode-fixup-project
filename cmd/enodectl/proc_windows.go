//go:build windows

package main

import "syscall"

// 프로세스를 띄우는 자리만 여기 남는다 — 살았는지·멈춤·소유 확인은
// internal/proc 로 내렸다(panel 과 공유). detachAttr 와 그 생성 플래그는
// start 가 프로세스를 띄우는 자리라 여기 산다.

const (
	// 프로세스 생성 플래그 (Win32 procthread).
	createNewProcessGroup = 0x00000200 // CREATE_NEW_PROCESS_GROUP
	detachedProcess       = 0x00000008 // DETACHED_PROCESS
)

// detachAttr 는 부모에서 떼어낸다 — Setsid 의 윈도우 대응이다.
// DETACHED_PROCESS 로 콘솔을 끊고, 새 프로세스 그룹으로 부모의 Ctrl 신호를 피한다.
// 끊지 않으면 enodectl 의 콘솔이 닫힐 때 노드가 함께 죽는다.
func detachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: createNewProcessGroup | detachedProcess}
}

// exeSuffix 는 실행 파일 이름에 붙는 것이다.
const exeSuffix = ".exe"
