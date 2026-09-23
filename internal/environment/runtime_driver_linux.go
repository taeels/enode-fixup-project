//go:build linux

package environment

func runtimeDriverAvailable(name string) bool {
	return name == "native" || name == "runc-overlay"
}
