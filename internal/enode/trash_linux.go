//go:build linux

package enode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/taeels/enode/internal/scratch"
	"golang.org/x/sys/unix"
)

// runcSessionPrefix 는 runc-overlay 작업 폴더 이름의 머리다. 기동 청소가 이것으로 찾는다.
const runcSessionPrefix = "enode-runc-"

// trashLine 은 trash-helper 가 stdout 에 쓰는 JSON 한 줄이다 (domain-entities.md 5절).
//
//	{"measured": {"bytes": N, "entries": M}}
//	{"removed": true}                         다 지웠다
//	{"removed": false, "left": ["…"]}          다른 filesystem 이라 남긴 경로가 있다
//	{"error": "…"}                             CheckEntry · Measure · Remove 의 오류
type trashLine struct {
	Measured *scratch.Size `json:"measured,omitempty"`
	Removed  *bool         `json:"removed,omitempty"`
	Left     []string      `json:"left,omitempty"`
	Error    string        `json:"error,omitempty"`
}

// IO 우선순위 (linux/ioprio.h). x/sys 에 이름이 없다.
const (
	ioprioWhoPgrp    = 2
	ioprioClassIdle  = 3
	ioprioClassShift = 13
)

// lowerTrashPriority 는 helper 의 IO 우선순위를 idle 로 · CPU 우선순위를 19 로 내린다.
//
// 둘 다 프로세스 그룹에 건다 — linux 에서 두 값은 스레드마다이고 Go 런타임은 스레드가
// 여럿이다. 그룹은 launcher 가 Setpgid 로 새로 만든 것이라 unshare 와 helper 만 든다.
// idle 은 BFQ 스케줄러의 디스크에서만 먹는다 (business-rules.md 4.2). 시험이 바꿔 끼운다 —
// 시험 프로세스에서 부르면 go test 의 그룹 전체가 느려진다.
var lowerTrashPriority = func() error {
	var errs []error
	if _, _, e := unix.Syscall(unix.SYS_IOPRIO_SET, ioprioWhoPgrp, 0, ioprioClassIdle<<ioprioClassShift); e != 0 {
		errs = append(errs, fmt.Errorf("set idle IO priority: %w", e))
	}
	if err := unix.Setpriority(unix.PRIO_PGRP, 0, 19); err != nil {
		errs = append(errs, fmt.Errorf("set CPU priority 19: %w", err))
	}
	return errors.Join(errs...)
}

// RunTrashHelper 는 unshare 가 다시 실행한 숨은 입구다 — enode trash-helper <trash> <entry>.
//
// runtime 과 같은 uid 매핑의 user namespace 안이라 subordinate uid 소유 항목과 권한 000
// 디렉터리를 지울 수 있다. 마운트 namespace 는 안 연다 — 삭제는 마운트하지 않는다. stdout 은
// 줄 프로토콜에만 쓰고 진단은 stderr 로 보낸다. exit 0 은 다 지웠다 · 1 은 그 밖이다.
func RunTrashHelper(args []string, out, errOut io.Writer) int {
	enc := json.NewEncoder(out)
	fail := func(err error) int {
		_ = enc.Encode(trashLine{Error: err.Error()})
		return 1
	}
	if len(args) != 2 {
		return fail(errors.New("usage: enode trash-helper <trash> <entry>"))
	}
	if err := lowerTrashPriority(); err != nil {
		// 막지 않는다 — 우선순위가 그대로여도 지우기는 맞다
		fmt.Fprintln(errOut, err)
	}
	trash, entry := args[0], args[1]
	if err := scratch.CheckEntry(trash, entry); err != nil {
		return fail(err)
	}
	size, err := scratch.Measure(trash, entry)
	if err != nil {
		return fail(err)
	}
	_ = enc.Encode(trashLine{Measured: &size})
	left, err := scratch.Remove(trash, entry)
	if err != nil {
		return fail(err)
	}
	removed := len(left) == 0
	_ = enc.Encode(trashLine{Removed: &removed, Left: left})
	if !removed {
		return 1
	}
	return 0
}

// trashHelperArgv 는 trash-helper 를 띄우는 명령이다. runtime-helper 와 같은 uid 매핑이고
// --mount 가 없다.
func trashHelperArgv(helper, trash, entry string) []string {
	return []string{"unshare", "--user", "--map-root-user", "--map-auto", "--fork", "--kill-child",
		"--", helper, "trash-helper", trash, entry}
}

// trashHelperCommand 는 항목 하나를 지우는 argv 다. 시험이 namespace 없는 명령으로 바꿔 끼운다.
var trashHelperCommand = func(trash, entry string) ([]string, error) {
	helper, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve enode executable for trash helper: %w", err)
	}
	return trashHelperArgv(helper, trash, entry), nil
}

// TrashLauncher 는 삭제자의 Launch 다 — 항목 하나에 helper 하나를 연다 (business-logic-model.md 3절).
func TrashLauncher(trash scratch.Trash) func(context.Context, string, func(scratch.Size)) error {
	return func(ctx context.Context, entry string, measured func(scratch.Size)) error {
		argv, err := trashHelperCommand(trash.Dir, entry)
		if err != nil {
			return &scratch.LaunchError{Err: err}
		}
		return runTrashHelper(ctx, argv, measured)
	}
}

// runTrashHelper 는 helper 를 띄우고 줄을 읽는다. 프로세스 그룹 · Pdeathsig 는 runtime-helper 와
// 같다 — 데몬이 죽으면 같이 죽는다. ctx 가 끝나면 그룹에 SIGKILL 을 보낸다. 반쯤 지운 항목은
// trash 에 남고 다음 기동이 거둔다.
//
// 한 줄도 없이 끝났으면 helper 가 뜨지 못한 것이다 (unshare 가 uid 매핑을 못 했다) — 항목의
// 잘못이 아니므로 LaunchError 로 돌려준다.
func runTrashHelper(ctx context.Context, argv []string, measured func(scratch.Size)) error {
	cmd := child(exec.Command(argv[0], argv[1:]...))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	var stderr lockedRuntimeBuffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return &scratch.LaunchError{Err: err}
	}
	if err := cmd.Start(); err != nil {
		return &scratch.LaunchError{Err: fmt.Errorf("start trash helper: %w", err)}
	}
	stopped := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		case <-stopped:
		}
	}()
	var last trashLine
	lines := 0
	dec := json.NewDecoder(stdout)
	for {
		var line trashLine
		if err := dec.Decode(&line); err != nil {
			break
		}
		lines++
		if line.Measured != nil {
			measured(*line.Measured)
			continue
		}
		last = line
	}
	waitErr := cmd.Wait()
	close(stopped)
	if err := ctx.Err(); err != nil {
		return err
	}
	switch {
	case lines == 0:
		return &scratch.LaunchError{Err: trashHelperFailure("trash helper did not start", waitErr, stderr.String())}
	case last.Error != "":
		return errors.New(last.Error)
	case last.Removed != nil && *last.Removed:
		return nil
	case last.Removed != nil:
		return &scratch.LeftError{Left: last.Left}
	}
	return trashHelperFailure("trash helper ended without a result", waitErr, stderr.String())
}

func trashHelperFailure(what string, waitErr error, stderr string) error {
	switch {
	case waitErr != nil && stderr != "":
		return fmt.Errorf("%s: %w: %s", what, waitErr, stderr)
	case waitErr != nil:
		return fmt.Errorf("%s: %w", what, waitErr)
	case stderr != "":
		return fmt.Errorf("%s: %s", what, stderr)
	}
	return errors.New(what)
}

// SweepOrphanSessions 는 데몬이 기동할 때 한 번 scratch 의 남은 작업 폴더를 trash 로 옮긴다
// (business-rules.md 3절). 데몬이 죽으면 helper 는 Pdeathsig 로 죽고 작업 폴더가 남는다.
// 잠금을 쥔 폴더(다른 데몬의 단계 · 도는 env check 의 smoke)는 건드리지 않는다.
func SweepOrphanSessions(scratchDir string, log *slog.Logger) {
	orphans, skipped, err := scratch.Orphans(scratchDir, runcSessionPrefix, time.Now())
	if err != nil {
		log.Warn("cannot look for orphaned runtime sessions", "scratch", scratchDir, "err", err)
		return
	}
	trash := scratch.TrashIn(scratchDir)
	for _, dir := range orphans {
		name := filepath.Base(dir)
		if _, err := trash.Move(dir); err != nil {
			log.Warn("cannot move an orphaned runtime session to trash", "name", name, "err", err)
			continue
		}
		log.Info("orphaned runtime session moved to trash", "name", name)
	}
	for _, dir := range skipped {
		log.Debug("runtime session without a lock is too young to judge; left in place", "name", filepath.Base(dir))
	}
}
