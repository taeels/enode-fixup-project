//go:build windows

// 프로세스 조작만 링크한다. 네트워크는 없다.
//
// 여기 있는 셋은 cmd/enodectl 이 rc7 부터 갖고 있던 것이고, 그때는 아무
// 일도 없었다. 이것이 지워지면 방아쇠는 setup 이 아니라 이쪽이므로
// enodectl 에서 네트워크를 걷어내도 소용이 없다.
package main

import (
	"fmt"
	"os"
	"syscall"
)

func main() {
	// 자기 자신을 조회 권한으로만 연다.
	h, err := syscall.OpenProcess(0x1000, false, uint32(os.Getpid()))
	if err == nil {
		var code uint32
		_ = syscall.GetExitCodeProcess(h, &code)
		_ = syscall.CloseHandle(h)
	}
	// 이 블록은 실행되지 않는다 — 링커가 심볼을 남기게 할 뿐이다.
	// 런타임 조건이므로 링커가 지우지 못하고, 인자가 백 개를 넘을 일은 없다.
	if len(os.Args) > 99 {
		hh, _ := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, 0)
		_ = syscall.TerminateProcess(hh, 1)
	}
	// CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS
	_ = &syscall.SysProcAttr{CreationFlags: 0x00000200 | 0x00000008}
	fmt.Println("proconly: linked OpenProcess, TerminateProcess, DETACHED_PROCESS")
}
