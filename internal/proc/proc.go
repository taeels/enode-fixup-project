package proc

import (
	"os"
	"strconv"
	"strings"
)

// PidFromLock 은 그 설정을 열고 있는 프로세스의 pid 다. 없으면 0.
//
// 잠금 파일이 곧 상태다 — enode 가 자기 pid 를 <config>.lock 에 쓰고, 죽으면
// 커널이 flock 을 푼다. 파일의 존재만으로는 아무것도 못 말한다(enode 는 끝나도
// 잠금 파일을 안 지운다). 그래서 그 pid 가 살아 있고 이 설정을 열고 있는지를
// OwnsConfig 로 함께 본다. cmd/enodectl 과 internal/panel 이 같이 쓴다.
func PidFromLock(configPath string) int {
	b, err := os.ReadFile(configPath + ".lock")
	if err != nil {
		return 0
	}
	line := strings.TrimSpace(strings.SplitN(string(b), "\n", 2)[0])
	pid, err := strconv.Atoi(line)
	if err != nil || pid <= 0 {
		return 0
	}
	if !OwnsConfig(pid, configPath) {
		return 0
	}
	return pid
}
