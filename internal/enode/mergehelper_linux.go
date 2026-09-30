//go:build linux

package enode

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/taeels/enode/internal/merge"
	"github.com/taeels/enode/internal/scratch"
)

// merge-helper 는 합치기를 도는 숨은 입구다 (bake 유닛 · FD 엔티티 8절 · FD 규칙 9절). trash-helper 와 같은 uid
// 매핑의 user namespace 안이라 subordinate uid 소유 항목과 권한 000 디렉터리를 옮긴다. 마운트 namespace 는 안 연다 —
// 호스트의 마운트 번호를 그대로 본다 (시작 전 확인의 마운트 줄이 뜻을 가진다). 한 합치기에 두 번 연다 — preflight ·
// apply. 그 사이에 호스트가 merging 을 쓴다. 입력은 stdin 의 JSON 한 줄, 출력은 stdout 의 JSON 한 줄이다.

// RunMergeHelper 는 unshare 가 다시 실행한 숨은 입구다 — enode merge-helper. stdout 은 응답 한 줄에만 쓰고 진단은
// stderr 로 보낸다. exit 0 은 오류가 없었다 · 1 은 그 밖이다. namespace 안의 root 로 돌고 권한을 내려놓지 않는다.
// Apply 는 끝까지 간다 — 본체에 상한이 없다 (결정 3-11). lower 쪽 항목은 항목마다 그 trash 로 rename 한다.
func RunMergeHelper(in io.Reader, out, errOut io.Writer) int {
	enc := json.NewEncoder(out)
	var req mergeHelperRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		fmt.Fprintln(errOut, "merge helper: cannot read the request:", err)
		_ = enc.Encode(mergeHelperResponse{Error: "merge helper: cannot read the request: " + err.Error(), Kind: "io"})
		return 1
	}
	paths := merge.Paths{Upper: req.Upper, Lower: req.Lower, Trash: req.Trash}
	var res *merge.Result
	var err error
	switch req.Op {
	case "preflight":
		err = merge.Preflight(paths)
	case "apply":
		trash := scratch.Trash{Dir: req.Trash}
		r, aerr := merge.Apply(context.Background(), paths, merge.Options{Discard: func(p string) error {
			_, err := trash.Move(p)
			return err
		}})
		res, err = &r, aerr
	default:
		err = fmt.Errorf("merge helper: unknown op %q", req.Op)
	}
	if err := enc.Encode(helperResponseOf(res, err)); err != nil {
		fmt.Fprintln(errOut, "merge helper: cannot write the response:", err)
		return 1
	}
	if err != nil {
		return 1
	}
	return 0
}

// mergeHelperArgv 는 merge-helper 를 띄우는 명령이다 — trash-helper 와 같은 모양이고 --mount 가 없다.
func mergeHelperArgv(helper string) []string {
	return []string{"unshare", "--user", "--map-root-user", "--map-auto", "--fork", "--kill-child",
		"--", helper, "merge-helper"}
}

// mergeHelperCommand 는 merge-helper 를 여는 argv 다. 시험이 namespace 없는 명령으로 바꿔 끼운다.
var mergeHelperCommand = func() ([]string, error) {
	helper, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve enode executable for merge helper: %w", err)
	}
	return mergeHelperArgv(helper), nil
}

// callMergeHelper 는 helper 를 띄우고 요청 한 줄을 보내 응답 한 줄을 읽는다. 프로세스 그룹 · Pdeathsig 는
// trash-helper 와 같다 — 데몬이 죽으면 같이 죽는다. 호스트는 helper 를 끝까지 기다린다 — 데몬의 ctx 로 죽이지
// 않는다 (Apply 는 끝까지 간다). stdout 의 첫 줄만 읽고, stderr 꼬리는 Wait 가 돌아온 뒤에 읽는다 — 그 전에
// 읽으면 helper 가 아직 다 쓰지 않았을 수 있다. 응답은 helperErrorOf 로 오류로 되돌린다.
func callMergeHelper(req mergeHelperRequest) (merge.Result, error) {
	fail := func(cause string, stderr string) (merge.Result, error) {
		if tail := lastLine(stderr); tail != "" {
			cause += " (" + tail + ")"
		}
		return merge.Result{}, fmt.Errorf("cannot run the merge helper: %s", cause)
	}
	argv, err := mergeHelperCommand()
	if err != nil {
		return fail(err.Error(), "")
	}
	cmd := child(exec.Command(argv[0], argv[1:]...))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	stderr := &tailBuffer{max: 4 << 10}
	cmd.Stderr = stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fail(err.Error(), "")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fail(err.Error(), "")
	}
	if err := cmd.Start(); err != nil {
		return fail(err.Error(), "")
	}
	sendErr := json.NewEncoder(stdin).Encode(req)
	_ = stdin.Close()
	var resp mergeHelperResponse
	decodeErr := json.NewDecoder(stdout).Decode(&resp)
	_, _ = io.Copy(io.Discard, stdout)
	waitErr := cmd.Wait()
	if decodeErr != nil {
		cause := decodeErr.Error()
		for _, e := range []error{sendErr, waitErr} {
			if e != nil {
				cause += "; " + e.Error()
			}
		}
		return fail(cause, stderr.String())
	}
	var res merge.Result
	if resp.Result != nil {
		res = *resp.Result
	}
	return res, helperErrorOf(resp)
}

// fileOwner 는 파일의 주인 uid 다. 초안을 읽을 때 노드 uid 인지 본다 (계획 4.1 7 · 32번).
func fileOwner(fi os.FileInfo) (int, bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return int(st.Uid), true
}
