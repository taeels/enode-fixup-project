package main

import (
	"github.com/taeels/enode/internal/enode"
)

// runSetupCmd 는 enodectl setup 의 별칭이다.
//
// 코드는 internal/enode 에 하나뿐이므로 둘이 갈릴 수 없다. 별칭을 두는
// 이유는 배포 모양이다 — 윈도우에 enode.exe 만 풀어 놓은 사람이 설정을
// 만들 방법이 없으면 안 된다.
func runSetupCmd(args []string) int {
	return enode.SetupCLI("enode setup", args)
}
