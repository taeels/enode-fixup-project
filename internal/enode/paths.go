package enode

import (
	"os"
	"path/filepath"
	"runtime"
)

// 노드가 쓰는 자리를 한 곳에만 적는다.
//
// Mediator 쪽에서 같은 것이 세 곳에 적혀 있다가 서로 다른 말을 했다
// (코드 · 예시 파일 · 패키지). 노드도 같은 모양이었다 — enode 는
// /etc/enode/local.yaml 을, enodectl 은 ~/.config/enode 를 봤다.
//
// 그리고 둘 다 유닉스만 알았다. 윈도우 크로스 빌드는 CI 가 지키는데
// (ADR-015 가 Go 를 고른 핵심 이유) 기본 경로가 윈도우를 모르면
// 깔리기는 하고 뜨지는 않는다 — rc8·rc9 가 Mediator 에서 밟은 자리다.

// ConfDir 는 노드 설정들이 놓이는 디렉터리다.
//
// $ENODE_CONFDIR 이 이긴다 — enodectl 이 그 이름으로 이미 열어 둔 자리이고,
// 시연에서 한 기계에 여러 벌을 두는 데 쓰인다.
func ConfDir() string {
	if v := os.Getenv("ENODE_CONFDIR"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		if runtime.GOOS == "windows" {
			return filepath.Join(programData(), "enode")
		}
		return "/etc/enode"
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "AppData", "Roaming", "enode")
	}
	return filepath.Join(home, ".config", "enode")
}

// StateDir 는 로그와 pid 가 놓이는 자리다.
func StateDir() string {
	if v := os.Getenv("ENODE_STATEDIR"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		if runtime.GOOS == "windows" {
			return filepath.Join(programData(), "enode", "state")
		}
		return "/var/lib/enode"
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "AppData", "Local", "enode")
	}
	return filepath.Join(home, ".local", "state", "enode")
}

// ConfigPaths 는 --config 없이 떴을 때 찾아볼 자리를 순서대로 돌려준다.
//
// 사용자 자리가 시스템 자리보다 앞이다 — 시연에서 노드가 발표자 노트북에
// 평범한 사용자로 sudo 없이 떠야 한다 (ADR-007 D3).
//
// 신원이 이 경로에 걸려 있다는 것을 잊으면 안 된다 (ADR-017) — 설정 파일의
// 절대경로가 node_id 의 일부다. 그래서 기본값을 옮기면 그 기본값으로 돌던
// 노드는 다른 노드가 된다. 유닉스의 /etc/enode/local.yaml 을 목록에 남겨 둔
// 이유가 그것이다: 이미 그것으로 도는 노드의 신원을 안 바꾼다.
func ConfigPaths() []string {
	ps := []string{filepath.Join(ConfDir(), "local.yaml")}
	if runtime.GOOS == "windows" {
		ps = append(ps, filepath.Join(programData(), "enode", "local.yaml"))
	} else {
		ps = append(ps, "/etc/enode/local.yaml")
	}
	return ps
}

// ResolveConfig 는 쓸 설정과, 못 찾았을 때 찾아본 자리들을 돌려준다.
func ResolveConfig(flagPath string) (path string, tried []string) {
	if flagPath != "" {
		return flagPath, nil // 명시했으면 없을 때 조용히 넘어가지 않는다
	}
	if v := os.Getenv("ENODE_CONFIG"); v != "" {
		return v, nil
	}
	tried = ConfigPaths()
	for _, p := range tried {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", tried
}

func programData() string {
	if v := os.Getenv("ProgramData"); v != "" {
		return v
	}
	return `C:\ProgramData`
}
