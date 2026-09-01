// TLS 네트워크만 링크한다. 프로세스는 건드리지 않는다.
//
// rc13 의 enodectl 이 setup 을 통해 새로 얻은 것이 정확히 이 부분이다.
package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	c := &http.Client{Timeout: 5 * time.Second}
	// 실제로 나가지 않는다 — 닫힌 포트를 찍어 링크만 시킨다.
	if _, err := c.Get("https://127.0.0.1:1/"); err != nil {
		fmt.Println("netonly: linked net/http and crypto/tls")
	}
}
