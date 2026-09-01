package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// cmdSetup 은 enode 에 넘긴다. 여기서 직접 하지 않는다.
//
// 왜 넘기나 — 링크 표면 때문이다. enode.SetupCLI 를 여기서 부르면 Go 링커가
// 거기서 도달하는 코드를 전부 끌어오고, 그 안에 net/http 가 있어 crypto/tls
// 까지 따라 들어온다. 실측으로 enodectl.exe 가 6,223,360 바이트에서
// 10,242,560 바이트로, crypto/tls 심볼이 24 에서 1068 로 늘었다.
//
// 그래서 로컬 프로세스 관리 도구가 「남의 프로세스를 죽이면서 바깥과 암호화
// 통신하는 서명 없는 실행 파일」이 되었다. AhnLab V3 가 그것을
// Trojan/Win.Generic.C5874069 로 진단해 지웠고, 사람이 받은 묶음에서
// enodectl.exe 만 사라졌다. 원본 묶음에서 이 파일만 갈아끼운 통제 실험에서
// setup 을 뺀 판만 살아남았다 (scripts/avprobe/README.md).
//
// 코드는 여전히 한 곳이다 — internal/enode/setup.go 이고 enode setup 이
// 그것을 부른다. 달라진 것은 부르는 방식뿐이므로 둘이 갈릴 일은 없다.
//
// 되돌리는 조건 — enodectl 이 setup 말고 다른 이유로 바깥과 통신해야 하면
// 다시 본다. 그때는 링크 표면을 줄이는 것으로 못 막으므로 코드 서명이 답이다.
func cmdSetup(args []string) error {
	bin := enodeBin()
	if st, err := os.Stat(bin); err != nil || st.IsDir() {
		return fmt.Errorf("enode binary not found: %s (set ENODE_BIN)", bin)
	}
	c := exec.Command(bin, append([]string{"setup"}, args...)...)
	// setup 은 사람에게 묻는다 — 셋을 다 이어야 대화가 성립한다.
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			os.Exit(ee.ExitCode()) // 넘긴 쪽의 종료 코드를 그대로 쓴다
		}
		return err
	}
	return nil
}
