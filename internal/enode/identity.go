// Package enode 는 실행 노드다. 광고하고, 일을 당겨가고, 실행한다.
package enode

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrNoEmail 은 git 이메일이 없을 때다.
//
// ★ 조용한 대체를 하지 않는다 ★ (ADR-015 §1) —
// $USER 로 대체하면 한 사람이 기계마다 다른 신원을 갖고,
// hostname 으로 대체하면 같은 기계의 두 사람이 한 신원이 된다.
// 어긋남은 Run Record 의 노드 귀속이 쓸모없어질 때까지 안 보인다.
var ErrNoEmail = errors.New("git 이메일이 없다: git config --global user.email 을 설정하라")

// Identity 는 이 enode 가 누구인가다.
type Identity struct {
	NodeID    string // hash(email ∥ hostname ∥ realpath(config))
	Label     string // 사람이 읽는 이름. 유도된 것이라 사람이 안 적는다.
	Principal string // 이메일. ★ 식별이지 인증이 아니다 ★
	Config    string // realpath 로 정규화된 설정 경로
}

// Derive 는 신원을 계산한다 (ADR-015 §2).
//
// ★ 설정 파일이 곧 신원이다 ★
// 한 기계에서 enode 를 둘 띄우려면 설정이 이미 둘이어야 한다 — 같은 local.yaml 을
// 두 프로세스가 읽으면 같은 자원을 둘 다 광고해 I1 이 깨지기 때문이다.
// 이미 유일해야 하는 것을 신원으로 쓰는 데는 비용이 0 이다.
func Derive(configPath string) (Identity, error) {
	email, err := gitEmail()
	if err != nil {
		return Identity{}, err
	}
	host, err := os.Hostname()
	if err != nil {
		return Identity{}, fmt.Errorf("hostname: %w", err)
	}
	abs, err := filepath.Abs(configPath)
	if err != nil {
		return Identity{}, err
	}
	// 심링크를 뚫는다 — 같은 파일을 다른 경로로 가리켜도 같은 신원이어야 한다.
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		abs = real
	}

	return deriveFrom(email, host, abs), nil
}

// deriveFrom 은 순수 함수다 — 같은 입력이면 같은 신원이다.
// ★ 재시작에 안정적이어야 한다 ★ (ADR-015 §2 가 무작위 UUID 를 기각한 이유):
// 신원이 재시작마다 바뀌면 유령 노드가 쌓이고, Run Record 의 노드 귀속이
// 재시작을 넘어 추적되지 않는다.
func deriveFrom(email, host, configAbs string) Identity {
	sum := sha256.Sum256([]byte(email + "\x00" + host + "\x00" + configAbs))
	return Identity{
		NodeID:    hex.EncodeToString(sum[:])[:12],
		Label:     deriveLabel(email, host, configAbs),
		Principal: email,
		Config:    configAbs,
	}
}

// deriveLabel 은 사람이 읽는 이름을 만든다.
// Record 의 노드 귀속이 해시로만 남으면 Case D 의 "서로 다른 기계였다" 를
// 사람이 못 읽는다. 유일성은 보장하지 않는다 — 그건 NodeID 의 몫이다.
func deriveLabel(email, host, configPath string) string {
	local := email
	if i := strings.IndexByte(email, '@'); i > 0 {
		local = email[:i]
	}
	base := strings.TrimSuffix(filepath.Base(configPath), filepath.Ext(configPath))
	if base == "local" || base == "" {
		return local + "@" + host
	}
	return local + "@" + host + ":" + base
}

// gitEmail 은 ★ 전역 ★ 설정을 읽는다.
// runctl 은 저장소 맥락이 적용되는 형태를 쓰지만(그 저장소에서 커밋할 신원과
// 같아야 하므로), enode 는 데몬이라 저장소 맥락이 무의미하고 시작 디렉터리에
// 따라 신원이 흔들리면 안 된다.
func gitEmail() (string, error) {
	out, err := exec.Command("git", "config", "--global", "--get", "user.email").Output()
	if err != nil {
		return "", ErrNoEmail
	}
	email := strings.TrimSpace(string(out))
	if email == "" {
		return "", ErrNoEmail
	}
	return email, nil
}
