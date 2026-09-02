//go:build windows

package enode

import "syscall"

// CREATE_NO_WINDOW — 자식에게 콘솔을 주지 않는다 (Win32 procthread).
const createNoWindow = 0x08000000

// childAttr 는 자식이 부모의 콘솔을 물려받지 못하게 한다.
//
// 왜 필요한가 — 윈도우에서 콘솔은 표준 입출력과 별개의 자원이다. 우리는
// 자식의 stdout·stderr 를 전부 버퍼로 받고 있는데도, 자식은 부모의 콘솔에
// 그대로 붙어 SetConsoleTitleW 같은 콘솔 API 를 직접 부를 수 있다.
//
// 실측 — 윈도우에서 enode.exe 를 띄우면 창 제목이 claude 로 바뀌었다.
// 기동할 때 detect.go 가 하네스를 찾느라 claude --version 을 돌리고, 그것이
// 제목을 자기 이름으로 바꾼 뒤 복원하지 않기 때문이다. 노드를 여럿 띄우면
// 창이 전부 같은 제목이 되어 어느 것이 어느 노드인지 사람이 못 가린다.
//
// 유닉스에는 이 경로가 없다. 거기서 제목은 자식이 stdout 으로 보내는 이스케이프
// 문자열로만 바뀌는데, 그 stdout 은 이미 버퍼로 막혀 있다. 그래서 윈도우에서만
// 났고 지금까지 안 걸렸다.
//
// 잃는 것이 없다 — 출력은 이미 파이프로 받고 있고, 하네스는 프롬프트를 stdin
// 으로 받는 비대화형이다. 오히려 GUI 로 띄웠을 때 자식마다 콘솔 창이 깜빡이며
// 뜨던 것도 함께 사라진다.
func childAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: createNoWindow}
}
