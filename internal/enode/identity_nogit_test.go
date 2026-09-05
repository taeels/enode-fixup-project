package enode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeConfig 는 설정 파일 하나를 쓰고 경로를 돌려준다.
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "local.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// git 이 없는 기계에서도 노드가 자기 이름을 갖는다 (ADR-015 §1).
//
// ADR-015 는 git 전역 설정의 이메일을 정본으로 삼으며 "커널 개발자는 예외
// 없이 user.email 을 설정해 두었다 — 이미 있는 것을 읽는다" 를 근거로 들었다.
//
//	실측 (2026-09-06) git 이 안 깔린 윈도우 노트북에서 노드가 신원을 못
//	만들고 그 자리에서 죽었다. enodectl 은 "start failed" 만 냈다.
//	그 노드는 추론만 하므로 git 이 할 일이 애초에 없었다.
//
// 조용한 대체가 아니다 — ADR-015 가 막은 것은 $USER 나 hostname 으로 몰래
// 채우는 것이고, 그 비교 목록에 「사람이 적는다」가 없었을 뿐이다.
func TestANodeWithoutGitCanStillHaveAnIdentity(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // git 이 없는 기계

	conf := writeConfig(t, "mediator: http://x\ntoken: t\nprincipal: someone@example.com\n")
	id, err := Derive(conf)
	if err != nil {
		t.Fatalf("a node that needs no git could not name itself: %v", err)
	}
	if id.Principal != "someone@example.com" {
		t.Fatalf("principal = %q", id.Principal)
	}
	if id.NodeID == "" || !strings.HasPrefix(id.Label, "someone@") {
		t.Fatalf("id is not usable: %+v", id)
	}
}

// 설정에 적힌 것이 git 을 이긴다 (ADR-015 §1).
//
// repo 와 반대 방향인데 근거가 같다 — repo 는 기계가 관찰하는 사실이라
// 유도가 이기고, 신원은 사람의 것이라 사람이 이긴다. 한 기계의 git 전역
// 설정이 그 기계 노드 전부의 신원을 강제하면 팀 공용 노드를 세울 자리가 없다.
func TestTheConfiguredPrincipalWinsOverGit(t *testing.T) {
	// 이겨야 할 상대를 여기서 만든다.
	//
	// 이 기계의 git 설정에 기대면 CI 처럼 이메일이 없는 곳에서 시험이
	// 스킵되고, 스킵은 종료코드에 안 나타나므로 「통과」로 보인다. 그러면
	// 이 시험이 지키려던 것을 아무도 안 지킨다.
	home := t.TempDir()
	gc := filepath.Join(home, "gitconfig")
	if err := os.WriteFile(gc, []byte("[user]\n\temail = machine@example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// GIT_CONFIG_GLOBAL 이 정본이고, 옛 git 을 위해 HOME 도 함께 옮긴다.
	t.Setenv("GIT_CONFIG_GLOBAL", gc)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	if got, err := gitEmail(); err != nil || got != "machine@example.com" {
		t.Fatalf("could not stage a git email to win over: %q %v", got, err)
	}

	conf := writeConfig(t, "mediator: http://x\ntoken: t\nprincipal: team@example.com\n")
	id, err := Derive(conf)
	if err != nil {
		t.Fatal(err)
	}
	if id.Principal != "team@example.com" {
		t.Fatalf("git won over what the person wrote: %q", id.Principal)
	}
}

// 둘 다 없으면 두 길을 다 말한다.
//
// 예전 문구는 git config 만 안내했다. git 이 아예 없는 기계에서 그것은
// 따를 수 없는 안내이고, 사람은 무엇을 해야 할지 모른 채 남는다.
func TestTheAdviceNamesBothWaysToSetAnIdentity(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	conf := writeConfig(t, "mediator: http://x\ntoken: t\n")
	_, err := Derive(conf)
	if err == nil {
		t.Fatal("derived an identity out of nothing")
	}
	for _, want := range []string{"principal", "git config"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the advice does not mention %q: %v", want, err)
		}
	}
}
