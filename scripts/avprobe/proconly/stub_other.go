//go:build !windows

// 윈도우 밖에서도 빌드는 되게 둔다 — 빌드가 깨지면 실수인지 의도인지 안 보인다.
package main

import "fmt"

func main() { fmt.Println("proconly: windows only") }
