//go:build windows

// 둘 다 링크한다 — rc13 의 enodectl 과 같은 능력 조합이다.
//
// 이것만 지워지면 방아쇠는 어느 한쪽이 아니라 둘이 한 파일에 있는 것이고,
// 그러면 enodectl 에서 네트워크를 걷어내는 것으로 풀린다.
package main

import (
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"
)

func main() {
	c := &http.Client{Timeout: 5 * time.Second}
	if _, err := c.Get("https://127.0.0.1:1/"); err != nil {
		fmt.Println("both: linked net/http and crypto/tls")
	}
	h, err := syscall.OpenProcess(0x1000, false, uint32(os.Getpid()))
	if err == nil {
		var code uint32
		_ = syscall.GetExitCodeProcess(h, &code)
		_ = syscall.CloseHandle(h)
	}
	// proconly 와 같다 — 실행되지 않고 심볼만 남는다.
	if len(os.Args) > 99 {
		hh, _ := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, 0)
		_ = syscall.TerminateProcess(hh, 1)
	}
	_ = &syscall.SysProcAttr{CreationFlags: 0x00000200 | 0x00000008}
	fmt.Println("both: linked OpenProcess, TerminateProcess, DETACHED_PROCESS")
}
