package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

func cmdEnvironment(args []string) error {
	if len(args) < 2 || (args[0] != "check" && args[0] != "apply") {
		return errors.New("usage: enodectl env check|apply <name> [--json]")
	}
	action, name := args[0], args[1]
	if _, err := oneName([]string{name}); err != nil {
		return err
	}
	bin := enodeBin()
	if st, err := os.Stat(bin); err != nil || st.IsDir() {
		return fmt.Errorf("enode binary not found: %s (set ENODE_BIN)", bin)
	}
	childArgs := []string{"env", action, "--config", confOf(name)}
	childArgs = append(childArgs, args[2:]...)
	c := exec.Command(bin, childArgs...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c.Run()
}
