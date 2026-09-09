// Package config 는 Mediator 설정을 읽는다 (ADR-015 §4).
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen    string    `yaml:"listen"`
	Token     string    `yaml:"token"` // 인증. 이메일에 권한을 걸지 않는다 (ADR-015 §1)
	Database  Database  `yaml:"database"`
	Artifacts Artifacts `yaml:"artifacts"`
	Claim     Claim     `yaml:"claim"`
	Contract  Contract  `yaml:"contract"`
	Notify    Notify    `yaml:"notify"`
	Lease     Lease     `yaml:"lease"`

	// Demo 는 이 Mediator 가 일회용 데모 인스턴스인가다.
	//
	// 참이면 읽기 셋(GET /v1/nodes · GET /v1/runs · GET /v1/runs/{id})이
	// 무인증으로 열리고 그 셋에만 요청 한도가 걸린다. 데모 화면에는
	// 토큰 입력 자리가 없고(게스트 로그인이 그 자리를 대신한다) 브라우저에
	// 실 토큰을 박는 것이 가둠의 셋째를 깨기 때문이다. 쓰기는 잠긴 채다.
	//
	// 기본값이 거짓이라 실 함대는 오늘 그대로다.
	//
	// decisions.md 3절의 "internal/config 를 보안 이유로 고치지 말 것" 은
	// 이 필드를 막지 않는다. 그 문장이 막은 것은 TLS 설정을 여기에 싣는
	// 일이고, 이것은 decisions.md 8절이 범위로 들인 데모 모드의 스위치다.
	// 이유가 다르므로 같은 파일이어도 같은 결정이 아니다.
	//
	// 이 필드는 접점이다 — 데모 쓰기 라우트를 여는 demo-back 이 같은
	// 스위치를 읽는다. 담당이 갈리므로 진행자의 직렬 병합 대상이다.
	Demo bool `yaml:"demo"`
	// DemoGallery opens the bounded gallery demo only when Demo is also enabled.
	DemoGallery bool `yaml:"demo_gallery"`
}

type Database struct {
	URL string `yaml:"url"` // $DATABASE_URL 이 있으면 그것이 이긴다
}

type Artifacts struct {
	Root         string `yaml:"root"`
	MaxBlobBytes int64  `yaml:"max_blob_bytes"`
}

type Claim struct {
	LongPollSeconds int `yaml:"long_poll_seconds"`
}

// Notify 는 알림 웹훅이다 (ADR-032 §4). 푸시는 보조이고 인박스가 정본이다.
type Notify struct {
	// AsksURL 로 질문이 올라올 때마다 AskEvent JSON 하나를 POST 한다.
	// 비면 알림 없음. 재시도 없음 — 유실돼도 인박스에 남는다.
	AsksURL string `yaml:"asks_url"`
}

// Contract 는 계약이 실행 중에 얼마나 자랄 수 있는가다 (ADR-031).
//
// 계약이 못 건드리는 자리여야 한다 — 계약을 짓는 것이 기계이므로,
// 계약 안에 상한을 두면 기계가 자기 상한을 늘린다. 종료 보장은 시스템이 쥔다
// (ADR-022 §7.8 이 "종료 보장이 계약 밖으로 나간다" 로 예고한 자리다).
type Contract struct {
	// MaxVersions 는 계약의 열이 가질 수 있는 판의 최대 개수다. v1(제출본)을 포함한다.
	// 2 면 계획 위임 한 번까지, 3 이면 그 위에 재계획 한 번까지다.
	MaxVersions int `yaml:"max_versions"`
}

// Lease 의 값 셋은 서로 묶여 있다 (ADR-016).
// 하트비트 주기 = 광고 만료 = 임대 갱신 주기이고, 그 값이 취소가 enode 에 닿는
// 지연(창의 크기)도 동시에 정한다. 지금은 임시치이며 S4 에서 실측으로 고친다.
type Lease struct {
	TTLSeconds   int `yaml:"ttl_seconds"`
	RenewSeconds int `yaml:"renew_seconds"`
	// NotAfterFactor 는 not_after 를 갱신 주기의 몇 배로 둘 것인가다.
	// 1 이면 하트비트 한 번만 놓쳐도 죽는다 — 그건 ADR-016 이 금지한 동작이다
	// ("실패한 하트비트 하나는 중단 신호가 아니다").
	NotAfterFactor int `yaml:"not_after_factor"`
	// MaxPerRun 은 한 Run 이 동시에 쥘 수 있는 노드 수다 (ADR-024 §4.2).
	//
	// 임대가 (노드) 단위이고 워커가 직렬이므로 이것이 곧 폭의 상한이다.
	// 오늘의 자연 상한("requires 의 개수" — ADR-023 §12)은 사람이 선언할 때
	// 이야기이고, acquire 를 계획이 짓기 시작하면 그 상한이 사라진다.
	// max_versions 와 같은 이유로 계약이 못 건드리는 자리에 둔다.
	// 0 이면 무제한 — 오늘 동작 그대로다. 값은 돌려보고 정한다.
	MaxPerRun int `yaml:"max_per_run"`
}

func Default() Config {
	return Config{
		Listen:    ":8080",
		Database:  Database{URL: "postgres:///enode"},
		Artifacts: Artifacts{Root: defaultArtifactsRoot(), MaxBlobBytes: 10 << 20},
		Claim:     Claim{LongPollSeconds: 7200}, // 2h (ADR-015 §5)
		// v1 제출본 + 계획 + 재계획 둘 — 돌려보고 정할 값이다 (INVARIANTS §4).
		Contract: Contract{MaxVersions: 4},
		Lease:    Lease{TTLSeconds: 3600, RenewSeconds: 60, NotAfterFactor: 3},
	}
}

// Load 는 ADR-015 §4 의 우선순위를 따른다.
//
//	--config <경로>  >  $ENODE_MEDIATOR_CONFIG  >  Paths() 의 순서
//
// 사용자 경로가 시스템 경로를 이긴다 — 그래야 시연에서 Mediator 가
// 발표자 노트북에 평범한 사용자로 sudo 없이 뜬다 (ADR-007 D3).
func Load(flagPath string) (Config, error) {
	c := Default()
	path, _, err := Resolve(flagPath)
	if err != nil {
		return c, err
	}
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return c, fmt.Errorf("config %s: %w", path, err)
		}
		if err := yaml.Unmarshal(b, &c); err != nil {
			return c, fmt.Errorf("config %s: %w", path, err)
		}
		if err := warnIfWorldReadable(path); err != nil {
			return c, err
		}
	}
	// 비밀은 환경변수가 이긴다 — 파일에 DB 비밀번호를 안 적을 수 있게 (ADR-015 §4)
	if v := os.Getenv("DATABASE_URL"); v != "" {
		c.Database.URL = v
	}
	if v := os.Getenv("ENODE_MEDIATOR_TOKEN"); v != "" {
		c.Token = v
	}
	// 데모 인스턴스는 컨테이너 하나로 뜬다 — 설정 파일을 굽지 않고
	// 환경변수 하나로 켠다. 비밀 둘과 같은 통로에 두는 이유는 그것뿐이고,
	// 값 자체는 비밀이 아니다.
	if v := os.Getenv("ENODE_DEMO_MODE"); v != "" {
		c.Demo = truthy(v)
	}
	return c, nil
}

// truthy 는 환경변수 하나를 불리언으로 읽는다.
//
// 참으로 읽는 값을 셋으로 못 박는다. 안 맞으면 거짓이다 — 오타 하나로
// 데모 인스턴스가 되는 것보다 안 켜지는 쪽이 낫다.
func truthy(v string) bool {
	return strings.EqualFold(v, "1") || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

// Paths 는 설정 파일을 찾아볼 자리를 순서대로 돌려준다.
//
// 경로를 여기 한 곳에만 적는다 — 예시 파일과 패키지와 코드가 서로 다른
// 경로를 말하던 것이 실측에서 사람을 엉뚱한 곳으로 보냈다. 찾는 쪽과
// 알려주는 쪽이 같은 목록을 봐야 그 어긋남이 안 생긴다.
//
// 사용자 자리가 시스템 자리보다 앞이다 (ADR-007 D3).
func Paths() []string {
	var ps []string
	if home, err := os.UserHomeDir(); err == nil {
		if runtime.GOOS == "windows" {
			ps = append(ps, filepath.Join(home, "AppData", "Roaming", "enode-mediator", "config.yaml"))
		} else {
			ps = append(ps, filepath.Join(home, ".config", "enode-mediator", "config.yaml"))
		}
	}
	if runtime.GOOS == "windows" {
		// %ProgramData% 가 유닉스의 /etc 자리다. 없을 때를 대비해 박아 둔다.
		pd := os.Getenv("ProgramData")
		if pd == "" {
			pd = `C:\ProgramData`
		}
		ps = append(ps, filepath.Join(pd, "enode-mediator", "config.yaml"))
	} else {
		ps = append(ps, "/etc/enode-mediator/config.yaml")
	}
	return ps
}

// Resolve 는 쓸 설정 파일과, 못 찾았을 때 찾아본 자리들을 돌려준다.
//
// 찾아본 자리를 함께 내는 이유 — 예전에는 못 찾아도 조용히 기본값으로 갔고,
// 그다음 줄에서 "토큰이 없다" 로 죽었다. 진짜 원인은 "설정을 못 찾았다" 인데
// 메시지가 그 말을 안 해서 사람이 토큰만 들여다봤다.
func Resolve(flagPath string) (path string, tried []string, err error) {
	if flagPath != "" {
		return flagPath, nil, nil // 명시했으면 없을 때 조용히 넘어가지 않는다
	}
	if v := os.Getenv("ENODE_MEDIATOR_CONFIG"); v != "" {
		return v, nil, nil
	}
	tried = Paths()
	for _, p := range tried {
		if _, err := os.Stat(p); err == nil {
			return p, nil, nil
		}
	}
	return "", tried, nil // 설정 파일이 없으면 기본값 + 환경변수로 돈다
}

// ADR-015 §4 — 파일에 DB 비밀번호가 들어가므로 0600 을 요구한다.
// MVP 는 비밀 관리 부품을 만들지 않는다. 경고만 한다.
func warnIfWorldReadable(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.Mode().Perm()&0o077 != 0 {
		fmt.Fprintf(os.Stderr, "warning: %s has mode %04o; 0600 is recommended (it contains database credentials)\n",
			path, fi.Mode().Perm())
	}
	return nil
}

// defaultArtifactsRoot 는 Record 가 쌓일 자리의 기본값이다.
//
// 윈도우에는 /var/lib 가 없다. 크로스 빌드는 CI 가 지키는데(ADR-015 가 Go 를
// 고른 핵심 이유) 기본값이 유닉스만 알면, 깔리기는 하고 뜨지는 않는다.
func defaultArtifactsRoot() string {
	if runtime.GOOS == "windows" {
		pd := os.Getenv("ProgramData")
		if pd == "" {
			pd = `C:\ProgramData`
		}
		return filepath.Join(pd, "enode-mediator", "artifacts")
	}
	return "/var/lib/enode-mediator/artifacts"
}
