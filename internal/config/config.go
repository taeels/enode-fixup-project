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
}

func Default() Config {
	return Config{
		Listen:    ":8080",
		Database:  Database{URL: "postgres:///enode"},
		Artifacts: Artifacts{Root: "/var/lib/enode-mediator/artifacts", MaxBlobBytes: 10 << 20},
		Claim:     Claim{LongPollSeconds: 7200}, // 2h (ADR-015 §5)
		Lease:     Lease{TTLSeconds: 3600, RenewSeconds: 60, NotAfterFactor: 3},
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
