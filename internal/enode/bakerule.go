package enode

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/taeels/enode/internal/contract"
	"github.com/taeels/enode/internal/lower"
	"github.com/taeels/enode/internal/merge"
)

// 굽기의 규칙이다 (bake 유닛 · business-rules.md). 시스템 호출이 없는 순수 함수만 둔다 — 흐름(bake.go ·
// bake_build.go · bake_merge.go · bake_resume.go)은 이 함수들을 차례로 부를 뿐이고, 무엇을 적고 어떻게 판정하나는
// 여기서 정한다. 규칙마다 표 시험 한 줄로 덮는다 — 이 패키지의 커버리지 여유가 얇다.

// 노드가 세션 안에서 돌리는 고정 글자 (계약은 못 바꾼다)

const (
	// bashCheck 는 sync 앞에 세션 안에서 도는 POSIX sh 한 줄이다 (되물음 3 답 A · 결정 45). 계약의 명령은
	// bash -c 로 돌므로 bash 가 없으면 계약의 명령을 하나도 안 돌리고 멈춘다.
	bashCheck = "command -v bash >/dev/null"

	// pinCommand 는 repo 모양의 manifest 를 고정한다 (business-rules.md 3절 9). 세션 안 · 워크스페이스 뿌리.
	pinCommand = "repo manifest -r -o " + pinnedFile

	// probeScript 는 IR 대조다 (business-rules.md 4절 · 계획 4.1 8번). 세션 안의 POSIX sh 로 돈다. 작업 폴더는
	// 워크스페이스 자리이고 IR 은 환경 변수 ENODE_IR 로만 온다 — 셸 글자에 IR 을 넣지 않는다. LC_ALL 등은 이 한 줄
	// 안에서 export 한다 — 세션 환경이 LC_ALL 을 프로필의 locale 로 덮는다.
	//
	//	exit 0      읽었다.  mode=none 이면 보는 git 이 없다
	//	exit 125    .repo/manifests 로 못 들어갔다
	//	그 밖       앞의 git 셋 중 하나가 실패했다 — 그 exit 그대로
	probeScript = `export LC_ALL=C GIT_TERMINAL_PROMPT=0
GIT_CEILING_DIRECTORIES=$(cd .. && pwd); export GIT_CEILING_DIRECTORIES
if [ -d .repo/manifests ]; then echo mode=repo; cd .repo/manifests || exit 125
elif [ -e .git ]; then echo mode=git
else echo mode=none; exit 0; fi
h=$(git rev-parse HEAD) || exit $?
echo "head=$h"
t=$(git rev-parse -q --verify "refs/tags/$ENODE_IR^{commit}"); rc=$?
[ $rc -le 1 ] || exit $rc
echo "tagged=$t"
g=$(git tag --points-at HEAD) || exit $?
printf '%s\n' "$g" | sed -n 's/^./tag=&/p'
echo "url=$(git config --get remote.origin.url)"
echo "branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)"
`

	// probeOutMax 는 대조의 stdout 을 읽는 상한이다 — 기계가 읽는 줄이라 짧다 (계획 4.1 9번).
	probeOutMax = 64 << 10
	// tailMax 는 노드 명령의 stderr 에서 문장에 싣는 마지막 줄의 상한이다.
	tailMax = 1 << 10
)

// 문장 (영어 · business-rules.md 16절)

const (
	// refusedText 는 굽기를 못 받는 노드의 거절이다 (business-rules.md 1절). 원인 코드가 없다 — 노드 설정의 일이다.
	refusedText = "a bake step needs a node that writes to an upper (workspace.writes=isolated); " +
		"this node writes to its workspace in place"
	// reasonBakeRunEnded 는 pending 인데 굽기 Run 의 임대가 사라져 몸통이 돈 까닭이다.
	reasonBakeRunEnded = "the bake run ended before the merge"
	// restartLeftText 와 restartDiscardedText 는 merge claim 의 재시작 두 줄이다 (business-rules.md 7.1).
	restartLeftText = "the pending upper of this run is left for the stale-bake cleanup; " +
		"the node restarted or could not record the end of the build step"
	restartDiscardedText = "the pending upper of this run was discarded as a stale bake; " +
		"the node restarted or could not record the end of the build step"
	nothingToMergeLine = "bake: nothing to merge; the build step left no pending upper"
	pinnedLine         = "bake: pinned the manifest to " + pinnedFile
	pendingLine        = "bake: the upper is pending the merge step"
	// oddPendingText 는 state.json 의 pending_upper 가 이 노드가 쓰는 모양이 아닐 때다 (business-rules.md 12.2 끝).
	oddPendingText = "the pending upper path does not look like one this node writes"
)

// cannotOpenState 는 상태 자리 · 잠금 파일 · state.json 을 못 읽은 것이다 (business-rules.md 1절). 「주인이 살아
// 있다」로 읽지 않는다.
func cannotOpenState(err error) string {
	return "cannot open the lower state directory: " + err.Error()
}

// bakeInProgressText 는 굽기 잠금의 주인이 살아 있을 때의 거절이다 (US-16). 주인은 state.json 의 owner 에서 읽는다.
func bakeInProgressText(st lower.State) string {
	if st.Owner == nil || st.Owner.Run == "" {
		return fmt.Sprintf("another bake holds this lower: run unknown (%s); submit the bake again later", st.Phase)
	}
	return fmt.Sprintf("another bake holds this lower: run %s on node %s since %s (%s); submit the bake again later",
		st.Owner.Run, st.Owner.Node, clockText(st.Since), st.Phase)
}

// resumeOpenedText 는 build claim 이 낡은 merging 을 만나 재개를 연 거절이다 (되물음 4 답 A · 결정 17).
func resumeOpenedText(run string) string {
	return fmt.Sprintf("an interrupted merge of run %s is left on this lower; this node started resuming it; "+
		"submit the bake again after it finishes", run)
}

// mismatchText 는 이 프로세스가 쥔 굽기와 state.json 이 맞지 않을 때다 (business-rules.md 7.1 둘째 줄).
func mismatchText(st lower.State) string {
	return fmt.Sprintf("the lower state does not match this bake: %s (run %s)", st.Phase, ownerRun(st))
}

// ownerRun 은 state.json 의 주인 Run 이다. 없으면 unknown.
func ownerRun(st lower.State) string {
	if st.Owner == nil || st.Owner.Run == "" {
		return "unknown"
	}
	return st.Owner.Run
}

// mergeWaitText 는 merge.wait 을 넘긴 것이다. 기다린 값은 time.Duration 의 글자다 (4h0m0s · 2s).
func mergeWaitText(wait time.Duration) string {
	return fmt.Sprintf("gave up waiting for the lower lock after %s (merge.wait)", wait)
}

// mergeStoppedText 는 merging 을 쓴 뒤에 멈춘 것이다 — 되돌리지 않고 재개가 잇는다 (답 2).
func mergeStoppedText(err error) string {
	return "merge stopped: " + err.Error() + "; the lower stays merging and a node on this lower resumes it"
}

// runtimeRunText 는 세션 오류(exit -1)의 문장이다 (결정 48). 세션이 이미 「runtime run:」을 붙였으면 그대로다.
func runtimeRunText(err error) string {
	msg := "the runtime ended the process"
	if err != nil {
		msg = err.Error()
	}
	if strings.HasPrefix(msg, "runtime run:") {
		return msg
	}
	return "runtime run: " + msg
}

// bashCheckText 는 bash 확인의 갈래다 (business-rules.md 3절 · 결정 45). 0 이면 "" — sync 로 간다.
// 1 이상은 모두 「bash 없음」이다 — dash 의 command -v 는 127, bash 는 1, sh 를 못 띄우면 runc 가 1 을 낸다.
// 음수는 세션 오류다.
func bashCheckText(code int, err error) string {
	switch {
	case code < 0:
		return runtimeRunText(err)
	case code > 0:
		return fmt.Sprintf("cannot run the bake commands: the prepared rootfs has no bash, or sh cannot run there "+
			"(exit %d)", code)
	}
	return ""
}

// pinFailedText 는 repo manifest -r 이 실패한 것이다.
func pinFailedText(code int, stderr string) string {
	return joinCause(fmt.Sprintf("cannot pin the manifest: repo manifest -r exited %d", code), stderr)
}

// joinCause 는 문장 끝에 원인 한 줄을 「: 」로 잇는다. 원인이 비었으면 문장 그대로다.
func joinCause(msg, cause string) string {
	if cause == "" {
		return msg
	}
	return msg + ": " + cause
}

// lastLine 은 출력의 마지막 빈칸 아닌 줄이다 — 1 KiB 까지 (뒤쪽을 남긴다).
func lastLine(s string) string {
	lines := strings.Split(strings.TrimRight(s, " \t\r\n"), "\n")
	line := strings.TrimSpace(lines[len(lines)-1])
	if len(line) > tailMax {
		line = line[len(line)-tailMax:]
	}
	return line
}

// 단계 로그 줄 (business-rules.md 16.2)

// startedLine · exitedLine 은 계약의 명령마다 머리와 끝 줄이다. label 은 "sync" 이거나 "build <이름>" 이다.
func startedLine(label string) string { return "bake: " + label + " started" }

func exitedLine(label string, code int, took time.Duration) string {
	return fmt.Sprintf("bake: %s exited %d after %s", label, code, took.Round(time.Second))
}

// exitReason 은 명령이 0 아닌 exit 로 끝난 last_attempt 의 reason 이다 (sync exited 1 · build config-a exited 2).
func exitReason(label string, code int) string { return fmt.Sprintf("%s exited %d", label, code) }

// skipLine 은 돌지 않은 구성 하나다 (답 3 · 물음 1 답 B).
func skipLine(name, why string) string { return "bake: skipping " + name + "; " + why }

// buildLabel 은 구성 하나의 이름표다.
func buildLabel(name string) string { return "build " + name }

// IR 대조 (business-rules.md 4절 · FD 엔티티 5절)

// irProbe 는 세션 안에서 돈 probeScript 의 결과다.
type irProbe struct {
	Mode   string   // "repo" (.repo/manifests) | "git" (워크스페이스 뿌리) | "none"
	Head   string   // git rev-parse HEAD
	Tagged string   // refs/tags/$ENODE_IR^{commit} — 로컬에 없으면 ""
	Tags   []string // git tag --points-at HEAD — 이름순.  대조가 HEAD 의 태그를 읽었으면 nil 이 아니다
	URL    string   // remote.origin.url — origin 이 없으면 "" (결정 47)
	Branch string   // rev-parse --abbrev-ref HEAD — detached 면 ""
	Exit   int      // 셸의 exit
	Stderr string   // stderr 의 마지막 줄
}

// parseProbe 는 대조의 stdout 을 푼다. 모르는 줄은 버린다. exit 0 으로 git · repo 모양을 끝까지 읽었으면 태그가
// 없어도 Tags 는 빈 목록이다 — head_tags 의 [] 와 null 을 나눈다 (되물음 5 답 A 의 (11)).
func parseProbe(stdout string, exit int, stderr string) irProbe {
	p := irProbe{Exit: exit, Stderr: lastLine(stderr)}
	for _, line := range strings.Split(stdout, "\n") {
		key, value, ok := strings.Cut(strings.TrimRight(line, " \t\r"), "=")
		if !ok {
			continue
		}
		switch key {
		case "mode":
			p.Mode = value
		case "head":
			p.Head = value
		case "tagged":
			p.Tagged = value
		case "tag":
			p.Tags = append(p.Tags, value)
		case "url":
			p.URL = value
		case "branch":
			if value != "HEAD" {
				p.Branch = value
			}
		}
	}
	if p.Exit == 0 && (p.Mode == "git" || p.Mode == "repo") && p.Tags == nil {
		p.Tags = []string{}
	}
	return p
}

// irOutcome 은 대조의 판정 넷이다.
type irOutcome int

const (
	irMatch      irOutcome = iota + 1 // 태그의 커밋이 HEAD 와 같다 — HEAD 에 다른 태그가 함께 있어도 된다
	irNotLocal                        // 태그가 로컬에 없다 — sync 가 그 태그를 받아 와야 한다
	irElsewhere                       // 태그가 다른 커밋을 가리킨다
	irUnverified                      // 보는 git 이 없다 · git 이 실패했다 — 노드 쪽 오류
)

// irVerdict 는 대조를 판정하고 문장을 낸다. 맞으면 로그 줄이고, 어긋나면 단계 로그 끝의 문장 (물음 1 답 B),
// 못 대 봤으면 error 의 문장이다.
func irVerdict(ir string, p irProbe) (irOutcome, string) {
	tags := tagList(p.Tags)
	switch {
	case p.Exit == 0 && p.Mode == "none":
		return irUnverified, "cannot verify ir: the workspace has neither .repo/manifests nor .git"
	case p.Exit == 125 && p.Mode == "repo":
		return irUnverified, joinCause("cannot verify ir: cannot enter .repo/manifests", p.Stderr)
	case p.Exit != 0:
		return irUnverified, joinCause(fmt.Sprintf("cannot verify ir: git exited %d", p.Exit), p.Stderr)
	case p.Head == "" || (p.Mode != "git" && p.Mode != "repo"):
		return irUnverified, "cannot verify ir: the check printed no HEAD"
	case p.Tagged == "":
		return irNotLocal, fmt.Sprintf("ir %s is not in the local repository after sync; the sync command must fetch "+
			"that tag (HEAD is %s, tags at HEAD: %s)", ir, p.Head, tags)
	case p.Tagged != p.Head:
		return irElsewhere, fmt.Sprintf("ir %s points at %s, but sync left HEAD at %s (tags at HEAD: %s)",
			ir, p.Tagged, p.Head, tags)
	}
	return irMatch, fmt.Sprintf("ir %s matches HEAD %s (tags at HEAD: %s)", ir, p.Head, tags)
}

// tagList 는 태그를 빈칸으로 잇는다. 없으면 none.
func tagList(tags []string) string {
	if len(tags) == 0 {
		return "none"
	}
	return strings.Join(tags, " ")
}

// repoIDOf 는 초안의 source.repo_id 다 — DetectRepo (repoid.go) 의 규칙을 세션 안 대조의 출력에 옮긴다. DetectRepo 를
// 부르지 않는다 — 부르면 호스트 git 이 대기 upper 에서 돈다 (계획 3.1). url 이 없으면 "" — detect 도 origin 이
// 없으면 "" 다.
func repoIDOf(p irProbe) string {
	if p.URL == "" {
		return ""
	}
	id := CanonicalRepoID(p.URL)
	if p.Mode == "repo" && p.Branch != "" {
		return id + "#" + p.Branch
	}
	return id
}

// stripPassword 는 url 의 비밀번호를 지운다 (business-rules.md 3.1 · 결정 10). scheme:// 모양일 때만 풀고
// user 는 둔다. scheme:// 모양인데 못 풀면 "" 이다 — 비밀이 새지 않는 쪽. scp 모양과 경로는 그대로다.
// metadata 는 lower 뿌리의 0644 파일이라 그 lower 위의 모든 Run 이 읽는다.
func stripPassword(u string) string {
	scheme, _, ok := strings.Cut(u, "://")
	if !ok || !schemeShaped(scheme) {
		return u
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return ""
	}
	if parsed.User == nil {
		return u
	}
	if _, has := parsed.User.Password(); !has {
		return u
	}
	parsed.User = url.User(parsed.User.Username())
	return parsed.String()
}

// schemeShaped 는 RFC 3986 의 scheme 글자인가다 — 글자로 시작하고 글자 · 숫자 · + · - · . 만.
func schemeShaped(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		letter := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
		if i == 0 && !letter {
			return false
		}
		digit := r >= '0' && r <= '9'
		if !letter && !digit && r != '+' && r != '-' && r != '.' {
			return false
		}
	}
	return true
}

// 초안 · metadata · 결과 칸 (FD 엔티티 4 · 6절)

// draftSchema 는 초안의 판이다.
const draftSchema = 1

// Draft 는 대기 자리의 bake.json 이다 (FD 엔티티 4절). 재개하는 노드가 metadata 를 쓰려면 build 가 안 사실이
// 필요하다 — state.json 은 형제가 광고마다 읽는 작은 파일이라 여기 둔다. upper 와 함께 살고 함께 trash 로 간다.
type Draft struct {
	Schema          int                 `json:"schema"`
	Run             string              `json:"run"`              // 굽기 Run
	Node            string              `json:"node"`             // 구운 노드 — metadata 의 bake.node (US-6)
	Source          lower.Source        `json:"source"`           // url · branch · repo_id · head · ir · pinned · sync_command · synced_at
	Builds          []lower.BuildRecord `json:"builds"`           // 돈 builds — 성공이면 모두 exit 0
	Environment     string              `json:"environment"`      // Record.PreparedEnvironment
	WorkspaceTarget string              `json:"workspace_target"` // Record.WorkspaceTarget
	PreviousIR      *string             `json:"previous_ir"`      // 굽기 잠금을 잡은 뒤 읽은 lower 의 source.ir
}

// metadataOf 는 초안에 bake 칸을 채운 metadata 다. merge 단계면 resumed false 이고 node 는 이 노드, 재개면
// resumed true 이고 node 는 초안의 Node 다 — 부르는 쪽이 넘긴다. previous_ir 은 초안의 값이다 (반쯤 합친 lower
// 에서 읽지 않는다 · business-rules.md 12.5).
func metadataOf(d Draft, run, node string, mergedAt time.Time, resumed bool) lower.Metadata {
	return lower.Metadata{
		Source: d.Source, Builds: d.Builds, Environment: d.Environment, WorkspaceTarget: d.WorkspaceTarget,
		Bake: lower.BakeRecord{Run: run, Node: node, MergedAt: mergedAt.UTC(), Resumed: resumed,
			PreviousIR: d.PreviousIR},
	}
}

// buildManifestOf 는 lower 의 타입을 contract 의 타입으로 옮긴다 — result 의 build 칸이자 $OUT/manifest 다.
// head 는 대조가 돌았으면 HEAD, 아니면 "" 이고, tags 는 대조가 태그를 읽었을 때만 nil 이 아니다 (null 과 []).
func buildManifestOf(sync lower.BuildRecord, builds []lower.BuildRecord, head string, ir *string,
	pinned *lower.Pinned, tags []string) contract.BuildManifest {
	m := contract.BuildManifest{Sync: contractRecord(sync), Builds: []contract.BuildRecord{}, Head: head, IR: ir,
		HeadTags: tags}
	for _, b := range builds {
		m.Builds = append(m.Builds, contractRecord(b))
	}
	if pinned != nil {
		m.Pinned = &contract.Pinned{File: pinned.File, SHA256: pinned.SHA256}
	}
	return m
}

func contractRecord(r lower.BuildRecord) contract.BuildRecord {
	return contract.BuildRecord{Name: r.Name, Command: r.Command, StartedAt: r.StartedAt, FinishedAt: r.FinishedAt,
		ExitCode: r.ExitCode}
}

// mergeOpsOf 는 merge.Result 를 셈 일곱으로 옮긴다 (계획 3.8).
func mergeOpsOf(r merge.Result) contract.MergeOps {
	return contract.MergeOps{Replaced: r.Replaced + r.TypeChanged, Created: r.Added, Dirs: r.NewDirs,
		Opaque: r.OpaqueDirs, Whiteouts: r.Whiteouts, Trashed: r.Discarded, Attrs: r.MergedDirs}
}

// mergedLine 은 합친 뒤의 단계 로그 줄이다 — merge.Result 의 칸 그대로와 걸린 시간.
func mergedLine(r merge.Result, took time.Duration) string {
	return fmt.Sprintf("bake: merged: ops %d, replaced %d, added %d, new dirs %d, opaque dirs %d, whiteouts %d, "+
		"type changed %d, merged dirs %d, discarded %d in %s", r.Ops, r.Replaced, r.Added, r.NewDirs, r.OpaqueDirs,
		r.Whiteouts, r.TypeChanged, r.MergedDirs, r.Discarded, took.Round(time.Millisecond))
}

// last_attempt (FD 엔티티 7절)

// attemptReason 은 reason 칸이다 — 원인 코드가 있으면 그것, 없으면 영어 한 줄.
func attemptReason(code, fallback string) string {
	if code != "" {
		return code
	}
	return fallback
}

// abandonedReason 은 낡은 상태를 치운 last_attempt 의 reason 이다 (결정 49). 주인이 죽었는지 committed 를 못 쓰고
// 놓았는지는 치우는 쪽이 모르므로 둘 다 참인 글자다. merge claim 의 재시작 줄이 pending 쪽 글자를 이 함수로 대 본다.
func abandonedReason(phase lower.Phase) string {
	return "abandoned: no process held the bake while it was " + string(phase)
}

// merge claim 의 확인 (business-rules.md 7.1)

type claimVerdict int

const (
	claimProceed          claimVerdict = iota + 1 // 쥐었고 pending 이고 주인이 이 Run — 합친다
	claimMismatch                                 // 쥐었는데 state 가 이 굽기의 것이 아니다
	claimRestartLeft                              // 안 쥐었고 주인이 이 Run 인 pending · merging — 정리가 아직 못 돌았다
	claimRestartDiscarded                         // 안 쥐었고 last_attempt 가 이 Run 의 pending 을 치운 기록이다
	claimNothing                                  // 그 밖 — 합칠 것 없음 (DONE)
)

// mergeClaim 은 7.1 의 표를 위에서부터 댄다. held 는 이 프로세스가 이 Run 의 굽기를 쥐었나다. 재시작 두 줄이
// 합칠 것 없음보다 먼저다 (결정 50). 재시작 줄은 pending 갈래뿐이다 (결정 52) — 주인이 이 Run 인 building 과
// building 을 치운 abandoned 기록은 합칠 것 없음이다. last_attempt 의 줄은 지금 phase 와 무관하다 (결정 49).
func mergeClaim(held bool, st lower.State, run string) claimVerdict {
	owner := st.Owner != nil && st.Owner.Run == run
	if held {
		if st.Phase == lower.PhasePending && owner {
			return claimProceed
		}
		return claimMismatch
	}
	if owner && (st.Phase == lower.PhasePending || st.Phase == lower.PhaseMerging) {
		return claimRestartLeft
	}
	if la := st.LastAttempt; la != nil && la.Run == run && la.Reason == abandonedReason(lower.PhasePending) {
		return claimRestartDiscarded
	}
	return claimNothing
}

// finished 는 끊긴 합치기가 이미 끝났나다 (business-rules.md 12.4). metadata 의 bake.run 이 주인 Run 이고
// source.synced_at · source.head 가 초안과 같을 때만 참이다 — run_id 한 칸으로 정하지 않는다 (다시 쓴 run_id).
func finished(md *lower.Metadata, run string, d Draft) bool {
	return md != nil && md.Bake.Run == run && md.Source.SyncedAt.Equal(d.Source.SyncedAt) &&
		md.Source.Head == d.Source.Head
}

// 대기 자리의 모양 (business-rules.md 12.2 끝)

// pendingShape 는 state.json 의 pending_upper 가 <scratch>/pending/<이 lower 의 키>/<이름>/upper 인 절대 경로인가다.
// 맞으면 그 scratch 를 돌려준다 — 대기 자리를 보내는 trash 는 그 scratch 의 trash 다 (다른 마운트면 EXDEV).
// 틀리면 옮기지도 합치지도 않는다 — 확인을 안 거치면 state.json 에 적힌 아무 경로가 trash 로 가고 삭제자가 지운다.
// state.json 의 경로는 linux 의 것이라 path 로 읽는다.
func pendingShape(p, key string) (string, bool) {
	if !path.IsAbs(p) || path.Clean(p) != p || path.Base(p) != "upper" {
		return "", false
	}
	name := path.Dir(p)
	keyDir := path.Dir(name)
	pending := path.Dir(keyDir)
	switch base := path.Base(name); base {
	case "/", ".", "..":
		return "", false
	}
	scratch := path.Dir(pending)
	if path.Base(keyDir) != key || path.Base(pending) != pendingDirName || scratch == "/" {
		return "", false
	}
	return scratch, true
}

// 기다림의 줄 (business-rules.md 8절 · FD 엔티티 11절)

// waitLog 은 배타를 기다리는 동안의 줄이다. Exclusive 의 watch 가 10초마다 부르고, 쓸지는 이것이 정한다 — 처음 ·
// 쥔 쪽의 모습(노드 · 역할 · Run · acks · 기록 없는 쥔 쪽)이 바뀔 때 · 그 밖에는 every 마다. 4시간이면 바뀜을
// 빼고 48 번이다. 시계를 받으므로 순수하게 시험한다.
type waitLog struct {
	deadline time.Time     // 노드 시계.  재개면 제로값 — 마감 대신 resume 의 줄을 쓴다
	resume   string        // 재개하는 Run
	every    time.Duration // 5분
	last     time.Time
	shape    string
}

// lines 는 이번 watch 에 쓸 줄이다. 안 쓸 때면 nil.
func (l *waitLog) lines(now time.Time, w lower.Waiting) []string {
	shape := waitShape(w)
	if !l.last.IsZero() && shape == l.shape && now.Sub(l.last) < l.every {
		return nil
	}
	l.last, l.shape = now, shape
	head := "waiting for the lower lock to resume the merge of run " + l.resume
	if !l.deadline.IsZero() {
		head = fmt.Sprintf("waiting for the lower lock; deadline %s node clock (%s left)", clockText(l.deadline),
			leftText(l.deadline.Sub(now)))
	}
	out := []string{head}
	for _, h := range w.Holders {
		name := h.Label
		if name == "" {
			name = h.Node
		}
		switch h.Role {
		case lower.RoleRun:
			out = append(out, fmt.Sprintf("  node %s holds it for run %s since %s", name, h.Run, clockText(h.Since)))
		default:
			out = append(out, fmt.Sprintf("  node %s holds it as a candidate; its drain was acknowledged %d of 2 times",
				name, h.Acks))
		}
	}
	if w.Unnamed {
		out = append(out, "  a holder left no record (an env check smoke, or an older enode that takes no lower lock)")
	}
	return out
}

// waitShape 는 쥔 쪽의 모습이다 — 바뀌면 줄을 쓴다. 기록은 node 순이다 (lower.Dir.Holders).
func waitShape(w lower.Waiting) string {
	var b strings.Builder
	for _, h := range w.Holders {
		fmt.Fprintf(&b, "%s|%s|%s|%d;", h.Node, h.Role, h.Run, h.Acks)
	}
	if w.Unnamed {
		b.WriteString("unnamed")
	}
	return b.String()
}

// tookLine 은 배타를 잡은 줄이다.
func tookLine(d time.Duration) string {
	return fmt.Sprintf("took the lower lock after %s", d.Round(time.Second))
}

// clockText 는 노드 시계의 시각 글자다 — UTC · 초 단위.
func clockText(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05Z") }

// leftText 는 남은 시간이다. 1분 이상이면 분 단위이고 끝의 0s 를 뗀다 (3h52m · 4h0m), 아래면 초 단위다.
func leftText(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Minute {
		return d.Round(time.Second).String()
	}
	return strings.TrimSuffix(d.Round(time.Minute).String(), "0s")
}

// merge-helper 의 주고받기 (FD 엔티티 8절 · 계획 4.1 16번)

// mergeHelperRequest 는 merge-helper 의 stdin 한 줄이다.
type mergeHelperRequest struct {
	Op    string `json:"op"`    // "preflight" | "apply"
	Upper string `json:"upper"` // pending_upper
	Lower string `json:"lower"` // 이 노드의 lower 루트
	Trash string `json:"trash"` // 대기 upper 가 있는 scratch 의 trash
}

// mergeHelperResponse 는 merge-helper 의 stdout 한 줄이다. exit 0 은 오류가 없었다 · 1 은 그 밖이다.
type mergeHelperResponse struct {
	Result *merge.Result `json:"result,omitempty"` // apply — 오류가 있어도 그때까지 센 것
	Error  string        `json:"error,omitempty"`
	Check  string        `json:"check,omitempty"` // *merge.PreflightError 면 그 Check
	Kind   string        `json:"kind,omitempty"`  // "preflight" | "op" | "io"
}

// helperPreflightError 는 helper 가 보낸 *merge.PreflightError 다 — 시작 전 확인이 어긋났다. 문장은 merge 의 것 그대로다.
type helperPreflightError struct {
	Check merge.Check
	Msg   string
}

func (e *helperPreflightError) Error() string { return e.Msg }

// helperOpError 는 helper 가 보낸 *merge.OpError 다 — 합치기의 호출 하나가 실패했다.
type helperOpError struct {
	Msg string
}

func (e *helperOpError) Error() string { return e.Msg }

// helperErrorOf 는 응답을 오류로 되돌린다 — 부르는 쪽이 errors.As 로 갈래를 본다. 오류가 없으면 nil 이다.
// 읽기 실패 (io) 는 PreflightError 가 아니다 — 다시 해 볼 일이다 (merge-rules 가 넘긴 일 · 받는 일 28).
func helperErrorOf(r mergeHelperResponse) error {
	switch {
	case r.Error == "":
		return nil
	case r.Kind == "preflight":
		return &helperPreflightError{Check: merge.Check(r.Check), Msg: r.Error}
	case r.Kind == "op":
		return &helperOpError{Msg: r.Error}
	}
	return errors.New(r.Error)
}

// helperResponseOf 는 helper 쪽에서 merge 의 결과를 응답 한 줄로 옮긴다 — helperErrorOf 의 짝이다.
func helperResponseOf(res *merge.Result, err error) mergeHelperResponse {
	r := mergeHelperResponse{Result: res}
	if err == nil {
		return r
	}
	r.Error, r.Kind = err.Error(), "io"
	var pe *merge.PreflightError
	var oe *merge.OpError
	switch {
	case errors.As(err, &pe):
		r.Kind, r.Check = "preflight", string(pe.Check)
	case errors.As(err, &oe):
		r.Kind = "op"
	}
	return r
}
