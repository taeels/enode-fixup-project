package enode

import "os/exec"

// noConsole 은 자식이 부모의 콘솔을 건드리지 못하게 한다.
//
// 이 패키지가 외부 프로그램을 띄우는 자리는 여덟 곳이고 앞으로 더 는다.
// 자리마다 SysProcAttr 을 손으로 붙이면 언젠가 하나를 빠뜨리는데, 빠뜨린
// 자리는 윈도우에서만 티가 나므로 리눅스에서 개발하는 동안 안 보인다.
// 그래서 한 곳으로 모으고, console_test.go 가 감싸지 않은 자리를 센다.
//
// 왜 필요한지는 console_windows.go 의 childAttr 이 적는다.
func noConsole(cmd *exec.Cmd) *exec.Cmd {
	cmd.SysProcAttr = childAttr()
	return cmd
}
