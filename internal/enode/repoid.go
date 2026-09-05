package enode

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CanonicalRepoID 는 저장소 주소를 정규화한다 (ADR-017 결정 1).
//
// 정규화를 안 하면 조용히 매칭이 실패하고 이유가 안 보인다
//
//	ssh://git@gerrit.corp:29418/kernel/linux  ┐
//	https://gerrit.corp/kernel/linux.git      ├─▶ gerrit.corp/kernel/linux
//	git@gerrit.corp:kernel/linux.git          ┘
//
// 포트를 지우는 이유는 같은 서버를 다른 프로토콜로 접근하면 포트가 다르기
// 때문이다 (Gerrit 은 ssh 29418 · https 443).
func CanonicalRepoID(remote string) string {
	s := strings.TrimSpace(remote)
	if s == "" {
		return ""
	}
	// scheme 제거
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	// 자격증명 제거. user@ 도 user:pass@ 도 있고, : 가 @ 보다 먼저 올 수 있다.
	// 그래서 "첫 / 앞에 있는 마지막 @" 를 기준으로 자른다 — 경로의 @ 는 안 건드린다.
	head := s
	if i := strings.IndexByte(s, '/'); i >= 0 {
		head = s[:i]
	}
	if j := strings.LastIndexByte(head, '@'); j >= 0 {
		s = s[j+1:]
	}
	// scp 형식의 host:path 를 host/path 로. 포트(숫자)면 지운다 —
	// 같은 서버를 다른 프로토콜로 접근하면 포트가 다르기 때문이다.
	if i := strings.IndexByte(s, ':'); i >= 0 {
		host, rest := s[:i], s[i+1:]
		if j := strings.IndexByte(rest, '/'); j >= 0 && isDigits(rest[:j]) {
			rest = rest[j+1:] // ssh://host:29418/path
		}
		s = host + "/" + strings.TrimPrefix(rest, "/")
	}
	s = strings.TrimSuffix(s, "/")
	s = strings.TrimSuffix(s, ".git")
	s = strings.TrimSuffix(s, "/")

	if i := strings.IndexByte(s, '/'); i >= 0 {
		return strings.ToLower(s[:i]) + s[i:] // host 만 소문자화. 경로는 대소문자를 구분한다.
	}
	return strings.ToLower(s)
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// DetectRepo 는 워크스페이스에서 canonical id 를 유도한다.
//
//	.repo 가 있으면   manifest 주소 + manifest 브랜치   (repo-id#branch)
//	없으면            .git 의 origin remote
//
// 사람이 저장소 주소를 적지 않는다 — runctl 은 cwd 에서, enode 는
// 워크스페이스에서 같은 방식으로 유도한다 (ADR-015 가 git config 를 읽는 것과 같은 트릭).
func DetectRepo(ctx context.Context, workspace string) string {
	if id := detectRepoManifest(ctx, workspace); id != "" {
		return id
	}
	out, err := gitIn(ctx, workspace, "config", "--get", "remote.origin.url")
	if err != nil {
		return ""
	}
	return CanonicalRepoID(out)
}

func detectRepoManifest(ctx context.Context, workspace string) string {
	manifests := filepath.Join(workspace, ".repo", "manifests")
	if _, err := os.Stat(manifests); err != nil {
		return ""
	}
	url, err := gitIn(ctx, manifests, "config", "--get", "remote.origin.url")
	if err != nil || url == "" {
		return ""
	}
	id := CanonicalRepoID(url)
	// 같은 manifest 의 다른 브랜치는 다른 트리다 (ADR-017).
	if br, err := gitIn(ctx, manifests, "rev-parse", "--abbrev-ref", "HEAD"); err == nil && br != "" && br != "HEAD" {
		return id + "#" + br
	}
	return id
}

// gitIn 은 ctx 를 받는다 — 워크스페이스가 네트워크 파일시스템 위에 있거나
// 자격증명 헬퍼가 안 돌아오면 이 git 이 무기한 걸릴 수 있고, 그때 부르는
// 쪽(광고 루프)이 함께 멈춘다. Detect 의 주석이 그 사슬을 적는다.
func gitIn(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := child(exec.CommandContext(ctx, "git", args...))
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
