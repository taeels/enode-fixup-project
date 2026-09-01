// 대조군 — 아무 능력도 링크하지 않는다.
//
// 이것까지 지워지면 방아쇠는 코드가 아니라 「서명 없는 Go 실행 파일」
// 자체다. 그때는 링크 표면을 줄여도 소용이 없고 코드 서명만 남는다.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("neither: no network, no process control")
	fmt.Println("pid:", os.Getpid())
}
