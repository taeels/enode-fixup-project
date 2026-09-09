package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// cmdServe 는 이 노드의 제어판을 띄운다 — enode 에 exec 위임한다.
//
// 왜 넘기나 — cmdSetup 과 같은 이유다(setup.go). 제어판은 net/http 서버라,
// internal/panel 을 enodectl 이 직접 임포트하면 링커가 net/http 와 crypto/tls 를
// enodectl.exe 로 끌어와 심볼 상한(ci.yml:479 · net/http<=50 · crypto/tls<=10)이
// 깨진다. 앞 회차 실측으로 백신이 enodectl.exe 를 지운 사건이 그 근거다
// (scripts/avprobe/README.md). 그래서 제어판은 enode panel 하위명령이 지고,
// serve 는 os/exec 로만 그것을 띄운다 — enodectl 의 링크 그래프가 net/http 에
// 안 닿는다.
func cmdServe(args []string) error {
	n, err := oneName(args)
	if err != nil {
		return err
	}
	listen := "127.0.0.1:8081"
	rest := args[1:]
	for i := 0; i < len(rest); i++ {
		if rest[i] == "--listen" && i+1 < len(rest) {
			listen = rest[i+1]
			i++
		}
	}
	bin := enodeBin()
	if st, err := os.Stat(bin); err != nil || st.IsDir() {
		return fmt.Errorf("enode binary not found: %s (set ENODE_BIN)", bin)
	}
	c := exec.Command(bin, "panel", "--config", confOf(n), "--node", n, "--listen", listen)
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
