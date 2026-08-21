// Package config 는 Mediator 설정을 읽는다 (ADR-015 §4).
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen    string    `yaml:"listen"`
	Token     string    `yaml:"token"` // 인증. 이메일에 권한을 걸지 않는다 (ADR-015 §1)
	Database  Database  `yaml:"database"`
	Artifacts Artifacts `yaml:"artifacts"`
	Claim     Claim     `yaml:"claim"`
	Contract  Contract  `yaml:"contract"`
	Lease     Lease     `yaml:"lease"`
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

// Contract 는 ★ 계약이 실행 중에 얼마나 자랄 수 있는가 ★ 다 (ADR-031).
//
// ★ 계약이 못 건드리는 자리여야 한다 ★ — 계약을 짓는 것이 기계이므로,
// 계약 안에 상한을 두면 ★ 기계가 자기 상한을 늘린다 ★. 종료 보장은 시스템이 쥔다
// (ADR-022 §7.8 이 "종료 보장이 계약 밖으로 나간다" 로 예고한 자리다).
type Contract struct {
	// MaxVersions 는 계약의 열이 가질 수 있는 판의 최대 개수다. v1(제출본)을 포함한다.
	// 2 면 계획 위임 한 번까지, 3 이면 그 위에 재계획 한 번까지다.
	MaxVersions int `yaml:"max_versions"`
}

// Lease 의 값 셋은 ★ 서로 묶여 있다 ★ (ADR-016).
// 하트비트 주기 = 광고 만료 = 임대 갱신 주기이고, 그 값이 취소가 enode 에 닿는
// 지연(창의 크기)도 동시에 정한다. 지금은 임시치이며 S4 에서 실측으로 고친다.
type Lease struct {
	TTLSeconds   int `yaml:"ttl_seconds"`
	RenewSeconds int `yaml:"renew_seconds"`
	// NotAfterFactor 는 not_after 를 갱신 주기의 몇 배로 둘 것인가다.
	// 1 이면 하트비트 한 번만 놓쳐도 죽는다 — 그건 ADR-016 이 금지한 동작이다
	// ("실패한 하트비트 하나는 중단 신호가 아니다").
	NotAfterFactor int `yaml:"not_after_factor"`
	// MaxPerRun 은 ★ 한 Run 이 동시에 쥘 수 있는 노드 수 ★ 다 (ADR-024 §4.2).
	//
	// 임대가 (노드) 단위이고 워커가 직렬이므로 ★ 이것이 곧 폭의 상한 ★ 이다.
	// 오늘의 자연 상한("requires 의 개수" — ADR-023 §12)은 사람이 선언할 때
	// 이야기이고, acquire 를 ★ 계획이 짓기 시작하면 ★ 그 상한이 사라진다.
	// max_versions 와 같은 이유로 ★ 계약이 못 건드리는 자리 ★ 에 둔다.
	// 0 이면 무제한 — 오늘 동작 그대로다. 값은 돌려보고 정한다.
	MaxPerRun int `yaml:"max_per_run"`
}

func Default() Config {
	return Config{
		Listen:    ":8080",
		Database:  Database{URL: "postgres:///enode"},
		Artifacts: Artifacts{Root: "/var/lib/enode-mediator/artifacts", MaxBlobBytes: 10 << 20},
		Claim:     Claim{LongPollSeconds: 7200}, // 2h (ADR-015 §5)
		// v1 제출본 + 계획 + 재계획 둘 — ★ 돌려보고 정할 값이다 ★ (INVARIANTS §4).
		Contract: Contract{MaxVersions: 4},
		Lease:    Lease{TTLSeconds: 3600, RenewSeconds: 60, NotAfterFactor: 3},
	}
}

// Load 는 ADR-015 §4 의 우선순위를 따른다.
//
//	--config <경로>  >  $ENODE_MEDIATOR_CONFIG
//	                 >  ~/.config/enode-mediator/config.yaml
//	                 >  /etc/enode-mediator/config.yaml
//
// ★ 사용자 경로가 시스템 경로를 이긴다 ★ — 그래야 시연에서 Mediator 가
// 발표자 노트북에 평범한 사용자로 sudo 없이 뜬다 (ADR-007 D3).
func Load(flagPath string) (Config, error) {
	c := Default()
	path, err := resolve(flagPath)
	if err != nil {
		return c, err
	}
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return c, fmt.Errorf("설정 %s: %w", path, err)
		}
		if err := yaml.Unmarshal(b, &c); err != nil {
			return c, fmt.Errorf("설정 %s: %w", path, err)
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
	return c, nil
}

func resolve(flagPath string) (string, error) {
	if flagPath != "" {
		return flagPath, nil // 명시했으면 없을 때 조용히 넘어가지 않는다
	}
	if v := os.Getenv("ENODE_MEDIATOR_CONFIG"); v != "" {
		return v, nil
	}
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, ".config", "enode-mediator", "config.yaml")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	p := "/etc/enode-mediator/config.yaml"
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	return "", nil // 설정 파일이 없으면 기본값 + 환경변수로 돈다
}

// ADR-015 §4 — 파일에 DB 비밀번호가 들어가므로 0600 을 요구한다.
// MVP 는 비밀 관리 부품을 만들지 않는다. 경고만 한다.
func warnIfWorldReadable(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.Mode().Perm()&0o077 != 0 {
		fmt.Fprintf(os.Stderr, "경고: %s 권한이 %04o 다. 0600 을 권한다 (DB 자격증명이 들어간다)\n",
			path, fi.Mode().Perm())
	}
	return nil
}
