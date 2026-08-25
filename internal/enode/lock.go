package enode

import (
	"fmt"
	"os"
)

// Lock 은 설정 파일 옆의 잠금이다.
//
// 중복 실행은 로컬에서 막는다 (ADR-015 §2)
//
// Mediator 에게 재시작과 중복은 똑같이 "같은 node_id 의 새 광고" 로 보여서
// 구분할 정보가 없다. 막으려 들면 재시작한 enode 가 광고 만료까지 못 붙는다 —
// 더 나쁘다. 로컬 잠금은 즉시 실패하고 Mediator 없이도 동작한다.
type Lock struct {
	f    *os.File
	path string
}

// Acquire 는 설정 경로에 대한 배타 잠금을 잡는다.
// 같은 설정으로 두 번째 프로세스를 띄우면 그 자리에서 실패한다.
func Acquire(configPath string) (*Lock, error) {
	path := configPath + ".lock"
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("lock file %s: %w", path, err)
	}
	if err := lockFile(f); err != nil {
		f.Close()
		return nil, fmt.Errorf("another enode is already running with this config (%s): %w", configPath, err)
	}
	_ = f.Truncate(0)
	fmt.Fprintf(f, "%d\n", os.Getpid())
	return &Lock{f: f, path: path}, nil
}

func (l *Lock) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	err := unlockFile(l.f)
	l.f.Close()
	return err
}
