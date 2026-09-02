//go:build !windows

package enode

import "syscall"

// childAttr 는 유닉스에서 아무것도 안 한다.
//
// 여기서는 자식이 부모의 콘솔을 따로 물려받는 개념이 없다 — 터미널 제목은
// 자식이 stdout 으로 보내는 이스케이프 문자열로만 바뀌고, 그 stdout 은 이미
// 버퍼로 막혀 있다. 그래도 자리를 비워 두지 않고 갈래 파일을 두는 이유는
// 이 저장소가 OS 분기를 빌드 태그 쌍으로만 하기로 정했기 때문이다
// (lock_unix.go/lock_windows.go · disk_unix.go/disk_windows.go 선례).
func childAttr() *syscall.SysProcAttr { return nil }
