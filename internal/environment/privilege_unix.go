//go:build !windows

package environment

import "os"

func privilegedCommand(name string, args []string) (string, []string) {
	if os.Geteuid() == 0 {
		return name, args
	}
	return "sudo", append([]string{name}, args...)
}
