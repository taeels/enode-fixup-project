package enode

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/taeels/enode/internal/contract"
)

// 상태 파일 (ADR-068 = A).
//
// 탐지 능력과 그것을 알아낸 시각은 데몬 메모리(Detector 의 Capabilities{Caps,At})
// 에만 있고 광고에 안 실린다 — 매처가 쓸 것이 아니고 싣는 순간 프로토콜이 는다
// (ADR-068). 그런데 제어판은 「무엇을 할 수 있나」와 「언제 잰 값인가」를 함께
// 보여야 한다(enode-features 3.1.2). 제어판은 데몬과 다른 프로세스이고 데몬은
// 나가는 http.Client 만 있어 그 값을 읽을 통로가 없다.
//
// 그래서 데몬이 이 파일에 쓰고 제어판이 읽는다. 정책 파일과 같은 자리·같은
// 권한이라 새 개념이 없다 — 같은 기계의 제어판↔데몬 통로를 이 저장소는 이미
// 정책 파일로 정해 뒀고(ADR-063), 이것은 그 방향만 뒤집은 것이다.

// Status 는 상태 파일에 담기는 것이다.
type Status struct {
	Caps []contract.Capability `yaml:"caps"`
	At   time.Time             `yaml:"at"`
}

// StatusPath 는 설정 파일 옆의 상태 파일이다 — <dir>/<stem>.status.yaml.
// PolicyPath 와 같은 규칙이다 (설정 경로가 곧 노드의 신원이므로 · ADR-017).
func StatusPath(configPath string) string {
	return strings.TrimSuffix(configPath, filepath.Ext(configPath)) + ".status.yaml"
}

// WriteStatus 는 지금 알아낸 능력을 상태 파일에 쓴다.
//
// tmp 에 쓰고 rename 으로 바꾼다 — 제어판이 반쯤 쓴 파일을 읽지 않게. 권한은
// 정책 파일과 같다(유닉스 0600 · 윈도우는 그 디렉터리의 상속 ACL 이라 모드
// 비트를 안 건다). 부르는 쪽이 At 이 바뀔 때만 부르므로 자주 안 쓴다.
func WriteStatus(configPath string, c Capabilities) error {
	b, err := yaml.Marshal(Status{Caps: c.Caps, At: c.At})
	if err != nil {
		return err
	}
	path := StatusPath(configPath)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ReadStatus 는 제어판이 읽는다. 없으면 그 사실을 오류로 돌려준다 —
// 부르는 쪽이 「아직 모름」으로 그린다.
func ReadStatus(configPath string) (Status, error) {
	b, err := os.ReadFile(StatusPath(configPath))
	if err != nil {
		return Status{}, err
	}
	var s Status
	if err := yaml.Unmarshal(b, &s); err != nil {
		return Status{}, err
	}
	return s, nil
}
