package enode

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Local 은 ★ 그 기계에서만 아는 것 ★ 이다 (ADR-012).
// 중앙에는 아무것도 안 적는다 — 그것이 LAVA 와 갈리는 자리다.
// 열 줄을 넘기지 않는 것이 원칙이다.
type Local struct {
	Mediator string `yaml:"mediator"`
	Token    string `yaml:"token"`

	// 이 enode 가 서 있는 워크스페이스. ★ 경로가 곧 신원의 일부다 ★ (ADR-017).
	// repo canonical id 는 여기서 유도한다 — 사람이 저장소 주소를 안 적는다.
	Workspace string `yaml:"workspace"`

	// WorkspaceID 는 ★ 자동 유도가 실패할 때만 ★ 적는다.
	//
	// DetectRepo 가 .repo · .git 에서 신원을 유도하지만, ★ 둘 다 없는 워크스페이스 ★
	// (풀어놓은 소스 트리 · 문서 디렉터리 · 파일서버 마운트)는 기계가 알아낼 방법이
	// 없다. 그러면 repo 속성이 안 실리고 ★ 그런 노드 둘이 매처에게 똑같아 보인다 ★.
	//
	// ★ Board 와 같은 자리다 ★ (ADR-012) — "포트에 무엇이 달렸는지는 기계가 모른다".
	// 그래서 「사람이 저장소 주소를 안 적는다」의 예외가 되지만, 이유가 같다.
	// ★ 유도가 성공하면 이 값은 무시된다 ★ — 사람이 적은 것이 기계가 본 것을 못 이긴다.
	WorkspaceID string `yaml:"workspace_id,omitempty"`

	// 자동으로 못 알아내는 것만 적는다. 포트에 무엇이 달렸는지는 기계가 모른다.
	Board *Board `yaml:"board,omitempty"`

	// 크로스 툴체인 탐지가 애매할 때만 명시한다. 비우면 자동 탐지한다.
	Arch string `yaml:"arch,omitempty"`

	// HarnessBin 은 하네스 실행 파일이다. 비우면 claude.
	// 시험용 스텁을 가리키게 할 수 있다.
	HarnessBin string `yaml:"harness_bin,omitempty"`

	// 이 값 아래로 떨어지면 빌드 능력을 ★ 광고에서 뺀다 ★ (ADR-017 결정 3).
	// 매칭 조건이 아니라 광고 조건이다 — "할 수 있는가" 는 노드가 판단한다.
	MinFreeGB int `yaml:"min_free_gb,omitempty"`

	// Orchestration 은 ★ 이 노드가 계약을 짓는 자리 ★ 라는 선언이다 (ADR-022 §5).
	//
	// ★ 사람이 적는 이유 ★ — 기계가 알아낼 수 없다. 하네스가 있다는 사실만으로는
	// "이 노드에 오케스트레이션을 맡겨도 되는가" 를 판정할 수 없고, 그건
	// ★ 노드 소유자의 결정 ★ 이다. ADR-012 가 "포트에 무엇이 달렸는지는 기계가
	// 모른다 — 그 기계에만 적는다" 로 board 를 다룬 것과 같은 자리다.
	Orchestration bool `yaml:"orchestration,omitempty"`
}

type Board struct {
	SoC  string `yaml:"soc"`
	Tag  string `yaml:"tag"`
	Port string `yaml:"port"`
}

func LoadLocal(path string) (Local, error) {
	var l Local
	b, err := os.ReadFile(path)
	if err != nil {
		return l, fmt.Errorf("설정 %s: %w", path, err)
	}
	if err := yaml.Unmarshal(b, &l); err != nil {
		return l, fmt.Errorf("설정 %s: %w", path, err)
	}
	if l.MinFreeGB == 0 {
		l.MinFreeGB = 10
	}
	return l, nil
}
