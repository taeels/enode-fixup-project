package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// NewToken 은 노드와 Mediator 가 나눠 갖는 비밀을 만든다.
//
// 24 바이트다 — 사람이 옮겨 적는 값이므로 길이가 곧 마찰이고, 128 비트보다
// 넉넉하면 그 마찰이 값을 못 한다. base64 에서 / + = 를 뺀다: 이 값이
// YAML 과 URL 과 셸 인용을 지나가는데, 그 셋에서 뜻이 달라지는 글자다.
func NewToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	s := base64.StdEncoding.EncodeToString(b)
	return strings.NewReplacer("/", "", "+", "", "=", "").Replace(s), nil
}

// tokenLine 은 최상위 token 항목이다. 주석과 들여쓴 키는 안 잡는다 —
// database 아래에도 token 이 생길 수 있고, 그것을 건드리면 안 된다.
var tokenLine = regexp.MustCompile(`(?m)^token:[ \t]*(.*)$`)

// EnsureToken 은 설정 파일의 token 이 비었으면 하나 만들어 써넣는다.
//
// 이것은 ADR-015 의 「조용한 대체를 하지 않는다」와 부딪히지 않는다. 그 원칙이
// 막는 것은 비어 있는데 있는 척하는 것이고, 여기서 하는 일은 그 반대다 —
// 만들고, 파일에 남기고, 만들었다고 말한다. 값이 이미 있으면 손대지 않는다.
//
// 파일이 없으면 만들지 않는다. 어느 자리에 만들지는 setup 이 사람에게 묻는
// 일이고, 여기서 골라 버리면 그 물음이 사라진다.
//
// 돌려주는 값 — (만든 토큰, 이미 있었으면 빈 문자열).
func EnsureToken(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	text := string(b)

	if m := tokenLine.FindStringSubmatch(text); m != nil {
		if v := strings.TrimSpace(strings.Trim(strings.TrimSpace(m[1]), `"'`)); v != "" {
			return "", nil // 이미 있다
		}
	}

	tok, err := NewToken()
	if err != nil {
		return "", err
	}
	line := "token: " + tok

	if tokenLine.MatchString(text) {
		text = tokenLine.ReplaceAllString(text, line)
	} else {
		if text != "" && !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		text += line + "\n"
	}
	if err := writeSecret(path, text); err != nil {
		return "", err
	}
	return tok, nil
}

// urlLine 은 database 블록 안의 url 이다. 들여쓰기까지 함께 잡아 그대로 쓴다.
var urlLine = regexp.MustCompile(`(?m)^([ \t]+)url:[ \t]*.*$`)

// SetDatabaseURL 은 database.url 을 바꾸거나, 없으면 database 블록을 더한다.
func SetDatabaseURL(path, url string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(b)
	quoted := `"` + url + `"`

	if i := strings.Index(text, "\ndatabase:"); i >= 0 || strings.HasPrefix(text, "database:") {
		if urlLine.MatchString(text) {
			text = urlLine.ReplaceAllString(text, "${1}url: "+quoted)
			return writeSecret(path, text)
		}
	}
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += "\ndatabase:\n  url: " + quoted + "\n"
	return writeSecret(path, text)
}

// SetArtifactsRoot 는 artifacts.root 가 없을 때만 더한다.
func SetArtifactsRoot(path, root string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(b)
	if strings.Contains(text, "\nartifacts:") || strings.HasPrefix(text, "artifacts:") {
		return nil
	}
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += "\nartifacts:\n  root: " + `"` + root + `"` + "\n"
	return writeSecret(path, text)
}

// writeSecret 은 0600 으로 쓴다. 같은 파일에 DB 자격증명이 들어 있다.
//
// 임시 파일에 쓰고 옮긴다 — 쓰는 도중에 죽으면 반쯤 쓰인 설정이 남고,
// 그러면 다음 기동이 파싱 오류로 죽는다.
func writeSecret(path, text string) error {
	dir := filepath.Dir(path)
	// 0755 다 — nfpm 이 만드는 /etc/enode-mediator 와 같은 값이다.
	// 파일이 0600 이므로 디렉터리를 좁힐 이유가 없고, 좁히면 전용 사용자로
	// 도는 mediator 가 그 디렉터리를 지나가지 못한다.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".config-*.yaml")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name) //nolint:errcheck // 옮겨졌으면 없다
	if _, err := tmp.WriteString(text); err != nil {
		tmp.Close() //nolint:errcheck
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, 0o600); err != nil {
		return err
	}
	return os.Rename(name, path)
}

// Sample 은 새 설정 파일의 뼈대다. setup 이 이것을 놓고 값을 채운다.
func Sample() string {
	return fmt.Sprintf(`# enode mediator configuration.
#
# The token is shared with every node: a node whose token differs is
# silently rejected with 401.

listen: ":8080"
token: ""

database:
  url: ""

artifacts:
  root: %q
`, defaultArtifactsRoot())
}

// Create 는 새 설정 파일을 뼈대로 만든다. 상위 디렉터리가 없으면 만든다.
//
// 이 함수가 따로 있는 이유 — Set* 들은 전부 파일을 먼저 읽는다. 그래서
// 없는 파일에는 못 쓴다. 처음 만드는 자리를 그것들로 대신하려 했다가
// "no such file or directory" 로 죽었다(rc8 실측).
func Create(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil // 이미 있다. 덮지 않는다
	}
	return writeSecret(path, Sample())
}
