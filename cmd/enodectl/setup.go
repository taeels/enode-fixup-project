package main

import (
	"os"

	"github.com/taeels/enode/internal/enode"
)

// cmdSetup 은 노드 설정을 하나 만든다.
//
// 여기가 제자리다 — enodectl 이 이미 설정 디렉터리를 쥐고 있고(list · id ·
// start), 설정을 새로 만드는 것이 정확히 그 일이다. enode setup 은 같은
// 것을 부르는 별칭이다: 윈도우에서 enode.exe 만 보고 있을 수 있다.
func cmdSetup(args []string) error {
	if code := enode.SetupCLI("enodectl setup", args); code != 0 {
		os.Exit(code)
	}
	return nil
}
