//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

// 잠자기 방지는 맥에만 있는 일이라 빌드 태그 쌍으로 가른다.
//
// 왜 가르나 — 원래는 한 함수 안에서 runtime.GOOS 로 갈랐다. 그러면 두 가지가
// 동시에 나쁘다. 첫째로 이 저장소는 OS 분기를 빌드 태그 쌍으로만 하기로 정해
// 두었고(lock_unix.go/lock_windows.go · disk_unix.go/disk_windows.go), 런타임
// 분기 하나만 그 규약 밖에 있었다. 둘째로 커버리지를 리눅스에서 재는데
// (.coverage-contract.yml 의 platform: linux/amd64) 가드 뒤의 11 문장이
// 리눅스 프로파일의 분모에는 들어가면서 어떤 테스트로도 도달할 수 없었다 —
// 즉 패키지 하한이 원리상 못 덮는 코드를 상대로 계산되고 있었다.
//
// 기각한 것 — runtime.GOOS 와 caffeinate 실행을 변수로 빼 테스트가 갈아끼우는
// 방식. 도달 범위는 가장 넓어지지만 생산 코드가 시험을 위해 모양을 바꾸고,
// 위의 첫째 이유(빌드 태그 쌍으로만 가른다)와 정면으로 어긋난다.
//
// caffeinated 는 여기로 옮기지 않았다 — 그쪽은 GOOS 가드가 없고 ps 를 돌려
// 볼 뿐이라 리눅스에서도 그대로 도달한다. 함께 옮기면 덮을 수 있는 8 문장을
// 스스로 분모 밖으로 버리는 것이 된다.

// keepAwake 는 시스템 잠자기가 함대를 끊는 것을 막는다.
//
// 디스플레이가 꺼지는 것은 상관없다. 시스템 잠자기가 CPU 와 네트워크를
// 멈추고, 그러면 광고가 끊기고 not_after 가 지나 Run 이 죽는다.
// 실측에서 밟았다: 승인을 기다리던 Run 이 맥이 조용해진 지 180초 만에
// "임대 만료로 Run 을 회수했다" 로 FAILED 가 됐다.
//
// enode 의 수명에 묶는다 (-w) — 껐다 잊는 일이 없고 유령이 안 남는다.
// -d 는 안 준다 — 화면은 꺼져도 된다. 우리가 막는 것은 그것이 아니다.
// sudo 를 안 쓴다 — 전원 설정을 영구히 바꾸지 않는다.
func keepAwake(pid int) {
	if _, err := exec.LookPath("caffeinate"); err != nil {
		fmt.Fprintln(os.Stderr, "  warning: caffeinate is missing — system sleep can cut the fleet")
		return
	}
	c := exec.Command("caffeinate", "-i", "-s", "-w", strconv.Itoa(pid))
	c.SysProcAttr = detachAttr()
	if err := c.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "  warning: could not start caffeinate: %v\n", err)
		return
	}
	_ = c.Process.Release()
	fmt.Printf("  sleep held off (while enode pid=%d lives)\n", pid)
}
