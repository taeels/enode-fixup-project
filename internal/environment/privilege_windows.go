//go:build windows

package environment

func privilegedCommand(name string, args []string) (string, []string) {
	return name, args
}
