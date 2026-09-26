//go:build linux

package enode

import (
	"bufio"
	"bytes"
	"context"
	"debug/elf"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	execenv "github.com/taeels/enode/internal/environment"
	"github.com/taeels/enode/internal/scratch"
	"golang.org/x/sys/unix"
)

const (
	runtimeInTarget              = "/run/enode/in"
	runtimeOutTarget             = "/run/enode/out"
	runtimeHarnessTarget         = "/run/enode/bin/harness"
	runtimeEnodeTarget           = "/run/enode/bin/enode"
	runtimeInstrumentationTarget = "/run/enode/instrumentation"
	runtimeCredentialTarget      = "/run/enode/bin/credential-helper"
	maxRuntimeStdin              = 32 << 20
)

// RuncOverlayRuntime은 한 step 동안 user/mount namespace를 유지하는 Linux
// adapter다. helper가 overlay mount, runc process, Harvest를 같은 namespace에서
// 소유하므로 Run이 끝난 뒤에도 merged view를 잃지 않는다.
//
// 세션이 끝나면 작업 폴더를 지우지 않고 trash(<scratch>/trash)로 rename 한 번에 옮긴다
// (ADR-076 §4.1). 지우는 것은 보고 뒤의 배경 삭제자다 (scratch.Deleter).
type RuncOverlayRuntime struct {
	doc      execenv.Document
	binding  execenv.Binding
	manifest execenv.Manifest
	rootfs   string
	helper   string
	trash    scratch.Trash
	command  func() *exec.Cmd
}

func NewRuncOverlayRuntime(doc execenv.Document, binding execenv.Binding, manifest execenv.Manifest) (*RuncOverlayRuntime, error) {
	return newRuncOverlayRuntime(doc, binding, manifest,
		execenv.PreparedRootFS(binding.Store, manifest.PreparedEnvironmentID))
}

func newRuncOverlayRuntime(doc execenv.Document, binding execenv.Binding, manifest execenv.Manifest, rootfs string) (*RuncOverlayRuntime, error) {
	if doc.Profile.Runtime.Driver != "runc-overlay" {
		return nil, fmt.Errorf("runc-overlay runtime cannot serve driver %q", doc.Profile.Runtime.Driver)
	}
	if manifest.Profile.SHA256 != "" && manifest.Profile.SHA256 != doc.SHA256 {
		return nil, errors.New("prepared manifest does not match the execution environment profile")
	}
	if st, err := os.Stat(rootfs); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("prepared rootfs %s is unavailable", rootfs)
	}
	helper, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve enode executable for runtime helper: %w", err)
	}
	return &RuncOverlayRuntime{doc: doc, binding: binding, manifest: manifest, rootfs: rootfs, helper: helper,
		trash: scratch.TrashIn(binding.Scratch)}, nil
}

type runtimeWireRequest struct {
	Op         string                 `json:"op"`
	Open       *runtimeWireOpen       `json:"open,omitempty"`
	Projection *runtimeWireProjection `json:"projection,omitempty"`
	Process    *runtimeWireProcess    `json:"process,omitempty"`
	Finalize   *FinalizeSpec          `json:"finalize,omitempty"`
}

type runtimeWireOpen struct {
	RootFS          string `json:"rootfs"`
	Workspace       string `json:"workspace"`
	In              string `json:"in"`
	Out             string `json:"out"`
	SSHDir          string `json:"ssh_dir,omitempty"`
	RunRoot         string `json:"run_root"`
	WorkspaceTarget string `json:"workspace_target"`
	UserName        string `json:"user_name"`
	UID             int    `json:"uid"`
	GID             int    `json:"gid"`
	SubUIDSize      int    `json:"subuid_size"`
	SubGIDSize      int    `json:"subgid_size"`
	Locale          string `json:"locale"`
	TmpSize         int64  `json:"tmp_size"`
	TmpExecutable   bool   `json:"tmp_executable"`
}

type runtimeWireProjection struct {
	Harness         string `json:"harness,omitempty"`
	Enode           string `json:"enode,omitempty"`
	Instrumentation string `json:"instrumentation,omitempty"`
	Credential      string `json:"credential,omitempty"`
}

type runtimeWireProcess struct {
	Argv  []string `json:"argv"`
	Cwd   string   `json:"cwd,omitempty"`
	Env   []string `json:"env,omitempty"`
	Stdin []byte   `json:"stdin,omitempty"`
}

type runtimeWireResponse struct {
	Op       string          `json:"op"`
	Error    string          `json:"error,omitempty"`
	Data     []byte          `json:"data,omitempty"`
	ExitCode int             `json:"exit_code,omitempty"`
	Finalize *FinalizeResult `json:"finalize,omitempty"`
}

func (r *RuncOverlayRuntime) Open(ctx context.Context, spec RuntimeSpec) (StepSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for name, value := range map[string]string{
		"workspace": spec.Dir, "$IN": spec.In, "$OUT": spec.Out,
	} {
		st, err := os.Stat(value)
		if err != nil || !st.IsDir() {
			return nil, fmt.Errorf("runtime %s directory %s is unavailable", name, value)
		}
	}
	if err := os.MkdirAll(r.binding.Scratch, 0o700); err != nil {
		return nil, fmt.Errorf("create runtime scratch: %w", err)
	}
	runRoot, err := os.MkdirTemp(r.binding.Scratch, runcSessionPrefix)
	if err != nil {
		return nil, fmt.Errorf("create runtime session: %w", err)
	}
	// 곧바로 잠금을 쥔다 — 다른 데몬의 기동 청소가 이 폴더를 남은 것으로 보지 않게.
	// 실패 갈래는 모두 trash 로 옮긴 뒤 잠금을 놓는다 (business-rules.md 1절 ①).
	lock, err := scratch.HoldSession(runRoot)
	if err != nil {
		return nil, r.discard(runRoot, nil, err)
	}
	tmpSize, err := execenv.ParseSize(r.doc.Profile.Runtime.Tmp.Size)
	if err != nil {
		return nil, r.discard(runRoot, lock, err)
	}

	var cmd *exec.Cmd
	if r.command != nil {
		cmd = child(r.command())
	} else {
		argv := runcOverlayHelperArgv(r.helper)
		cmd = child(exec.Command(argv[0], argv[1:]...))
	}
	// parent node가 crash해도 unshare supervisor가 고아로 남지 않는다.
	// --kill-child가 이어서 helper/runc namespace를 거둔다.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, r.discard(runRoot, lock, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, r.discard(runRoot, lock, err)
	}
	var helperErr lockedRuntimeBuffer
	cmd.Stderr = &helperErr
	if err := cmd.Start(); err != nil {
		return nil, r.discard(runRoot, lock, fmt.Errorf("start runtime namespace helper: %w", err))
	}

	s := &runcOverlaySession{
		spec: spec, record: spec.Record, open: runtimeWireOpen{
			RootFS: r.rootfs, Workspace: spec.Dir, In: spec.In, Out: spec.Out,
			RunRoot:         runRoot,
			WorkspaceTarget: r.doc.Profile.Runtime.WorkspaceTarget,
			UserName:        r.doc.Profile.RootFS.User.Name,
			UID:             r.doc.Profile.RootFS.User.UID, GID: r.doc.Profile.RootFS.User.GID,
			SubUIDSize: r.doc.Profile.Host.Require.SubUIDSize,
			SubGIDSize: r.doc.Profile.Host.Require.SubGIDSize,
			Locale:     r.doc.Profile.RootFS.Locale, TmpSize: tmpSize,
			TmpExecutable: r.doc.Profile.Runtime.Tmp.Executable,
		},
		cmd: cmd, stdin: stdin, decoder: json.NewDecoder(bufio.NewReader(stdout)),
		encoder: json.NewEncoder(stdin), helperErr: &helperErr, runRoot: runRoot,
		lock: lock, trash: r.trash,
	}
	if r.doc.Profile.Runtime.Credentials.SSH == "readonly" {
		s.open.SSHDir = r.binding.SSHDir
	}
	if err := s.send(runtimeWireRequest{Op: "open", Open: &s.open}); err != nil {
		return nil, errors.Join(err, s.abort())
	}
	response, err := s.receive()
	if err != nil || response.Op != "opened" || response.Error != "" {
		released := s.abort()
		if err != nil {
			return nil, errors.Join(fmt.Errorf("open runtime namespace: %w", err), released)
		}
		return nil, errors.Join(fmt.Errorf("open runtime namespace: %s", response.Error), released)
	}
	return s, nil
}

// discard 는 helper 를 띄우기 전에 실패한 Open 의 작업 폴더를 버린다. 막 만든 빈 폴더라도
// 같은 길로 간다 — 자리가 하나면 빠지는 경로가 없다. 옮긴 뒤에 잠금을 놓는다.
func (r *RuncOverlayRuntime) discard(runRoot string, lock *scratch.SessionLock, cause error) error {
	errs := []error{cause}
	if _, err := r.trash.Move(runRoot); err != nil {
		errs = append(errs, fmt.Errorf("move runtime session to trash: %w", err))
	}
	_ = lock.Release()
	return errors.Join(errs...)
}

func runcOverlayHelperArgv(helper string) []string {
	// 바깥 helper에는 PID namespace를 열지 않는다. 새 PID namespace와 기존
	// /proc를 함께 쓰면 runc가 nsexec child의 uid_map을 찾을 수 없다. 실제
	// workload의 PID namespace는 OCI config가 만들고 helper는 수명만 지킨다.
	return []string{"unshare", "--user", "--map-root-user", "--map-auto",
		"--mount", "--fork", "--kill-child", "--", helper, "runtime-helper"}
}

type lockedRuntimeBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *lockedRuntimeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *lockedRuntimeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return strings.TrimSpace(b.b.String())
}

type runcOverlaySession struct {
	spec      RuntimeSpec
	record    *execenv.Record
	open      runtimeWireOpen
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	decoder   *json.Decoder
	encoder   *json.Encoder
	helperErr *lockedRuntimeBuffer
	runRoot   string
	lock      *scratch.SessionLock
	trash     scratch.Trash
	sendMu    sync.Mutex
	callMu    sync.Mutex
	closeOnce sync.Once
	closeErr  error
	// aborted 는 Finalize 가 답하지 않는 helper 를 죽였다는 표시다. callMu 아래에서 쓰고 읽는다.
	aborted bool
	// releaseOnce 는 작업 폴더를 trash 로 옮기는 일을 한 번만 하게 한다 — abort 와 Close 가 모두 부른다.
	releaseOnce sync.Once
	releaseErr  error
}

func (s *runcOverlaySession) Paths() RuntimePaths {
	return RuntimePaths{Dir: s.open.WorkspaceTarget, In: runtimeInTarget, Out: runtimeOutTarget}
}

func (s *runcOverlaySession) send(request runtimeWireRequest) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	if err := s.encoder.Encode(request); err != nil {
		return fmt.Errorf("send runtime helper request: %w", err)
	}
	return nil
}

func (s *runcOverlaySession) receive() (runtimeWireResponse, error) {
	var response runtimeWireResponse
	if err := s.decoder.Decode(&response); err != nil {
		detail := s.helperErr.String()
		if detail != "" {
			return response, fmt.Errorf("%w: %s", err, detail)
		}
		return response, err
	}
	return response, nil
}

func (s *runcOverlaySession) Project(_ context.Context, spec FrameworkProjectionSpec) (FrameworkProjection, error) {
	s.callMu.Lock()
	defer s.callMu.Unlock()
	harness, err := resolveProjectedExecutable(s.open.RootFS, spec.HarnessExecutable, false)
	if err != nil {
		return FrameworkProjection{}, fmt.Errorf("harness: %w", err)
	}
	enodeBin, err := resolveProjectedExecutable(s.open.RootFS, spec.EnodeExecutable, false)
	if err != nil {
		return FrameworkProjection{}, fmt.Errorf("enode hook: %w", err)
	}
	credential, err := resolveProjectedExecutable(s.open.RootFS, spec.CredentialHelper, true)
	if err != nil {
		return FrameworkProjection{}, fmt.Errorf("credential helper: %w", err)
	}
	instrumentation, err := resolveProjectedDirectory(spec.Instrumentation)
	if err != nil {
		return FrameworkProjection{}, fmt.Errorf("instrumentation: %w", err)
	}
	projection := runtimeWireProjection{Harness: harness, Enode: enodeBin,
		Instrumentation: instrumentation, Credential: credential}
	if err := s.send(runtimeWireRequest{Op: "project", Projection: &projection}); err != nil {
		return FrameworkProjection{}, err
	}
	response, err := s.receive()
	if err != nil {
		return FrameworkProjection{}, err
	}
	if response.Op != "projected" || response.Error != "" {
		return FrameworkProjection{}, fmt.Errorf("project framework runtime: %s", response.Error)
	}
	result := FrameworkProjection{
		HarnessExecutable: runtimeHarnessTarget, EnodeExecutable: runtimeEnodeTarget,
		Instrumentation: runtimeInstrumentationTarget,
	}
	if credential != "" {
		result.CredentialHelper = runtimeCredentialTarget
	}
	return result, nil
}

func (s *runcOverlaySession) Run(ctx context.Context, spec ProcessSpec) (int, error) {
	s.callMu.Lock()
	defer s.callMu.Unlock()
	if len(spec.Argv) == 0 {
		return -1, exec.ErrNotFound
	}
	stdin, err := readRuntimeStdin(spec.Stdin)
	if err != nil {
		return -1, err
	}
	request := runtimeWireRequest{Op: "run", Process: &runtimeWireProcess{
		Argv: append([]string(nil), spec.Argv...), Cwd: spec.Dir,
		Env: append([]string(nil), spec.Env...), Stdin: stdin,
	}}
	if err := s.send(request); err != nil {
		return -1, err
	}
	cancelDone := make(chan struct{})
	defer close(cancelDone)
	go func() {
		select {
		case <-ctx.Done():
			_ = s.send(runtimeWireRequest{Op: "cancel"})
		case <-cancelDone:
		}
	}()
	stdout, stderr := spec.Stdout, spec.Stderr
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	for {
		response, err := s.receive()
		if err != nil {
			return -1, fmt.Errorf("runtime run: %w", err)
		}
		switch response.Op {
		case "stdout":
			if _, err := stdout.Write(response.Data); err != nil {
				_ = s.send(runtimeWireRequest{Op: "cancel"})
				return -1, err
			}
		case "stderr":
			if _, err := stderr.Write(response.Data); err != nil {
				_ = s.send(runtimeWireRequest{Op: "cancel"})
				return -1, err
			}
		case "run-done":
			if response.Error != "" {
				return response.ExitCode, errors.New(response.Error)
			}
			return response.ExitCode, nil
		case "error":
			return -1, errors.New(response.Error)
		}
	}
}

func readRuntimeStdin(reader io.Reader) ([]byte, error) {
	if reader == nil {
		return nil, nil
	}
	b, err := io.ReadAll(io.LimitReader(reader, maxRuntimeStdin+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxRuntimeStdin {
		return nil, fmt.Errorf("runtime stdin exceeds %d bytes", maxRuntimeStdin)
	}
	return b, nil
}

// helperGrace 는 Finalize 의 ctx 가 끝난 뒤 helper 가 스스로 돌아와 답을 쓰기를 기다리는
// 여유다 (business-logic-model.md 3절). 지나면 helper 를 죽인다. 시험이 줄인다.
//
// 기준이 마감이 아니라 ctx 가 끝난 때인 까닭 — 임대가 끝나도 ctx 가 끝나는데, helper 는
// 요청을 하나씩 처리해 도중에 cancel 요청을 못 읽는다. 그래서 그때도 길은 abort 하나다.
var helperGrace = 5 * time.Second

// Finalize 는 결과 확정을 helper 에 맡긴다. 마감은 요청의 Deadline 으로 helper 에 가고,
// helper 가 그 시각으로 자기 ctx 를 만든다. Worker 쪽은 ctx 가 끝난 뒤 helperGrace 만큼
// 더 기다리고, 그래도 답이 없으면 helper 를 죽이고 ctx 의 오류를 돌려준다.
func (s *runcOverlaySession) Finalize(ctx context.Context, spec FinalizeSpec) (FinalizeResult, error) {
	s.callMu.Lock()
	defer s.callMu.Unlock()
	copySpec := spec
	copySpec.Workspace = ""
	if deadline, ok := ctx.Deadline(); ok && copySpec.Deadline.IsZero() {
		copySpec.Deadline = deadline
	}
	if err := s.send(runtimeWireRequest{Op: "finalize", Finalize: &copySpec}); err != nil {
		return FinalizeResult{}, err
	}
	type answer struct {
		response runtimeWireResponse
		err      error
	}
	got := make(chan answer, 1)
	go func() {
		response, err := s.receive()
		got <- answer{response, err}
	}()
	var a answer
	select {
	case a = <-got:
	case <-ctx.Done():
		select {
		case a = <-got:
		case <-time.After(helperGrace):
			_ = s.abort() // 옮기기의 오류는 뒤의 Close 가 돌려준다
			s.aborted = true
			return FinalizeResult{}, ctx.Err()
		}
	}
	if a.err != nil {
		return FinalizeResult{}, a.err
	}
	if a.response.Op != "finalized" || a.response.Finalize == nil {
		return FinalizeResult{}, errors.New(a.response.Error)
	}
	switch a.response.Error {
	case "":
		return *a.response.Finalize, nil
	case context.DeadlineExceeded.Error():
		// helper 의 마감이다 — 모은 것은 남기고 마감으로 돌려준다
		return *a.response.Finalize, context.DeadlineExceeded
	}
	return *a.response.Finalize, errors.New(a.response.Error)
}

// Close 는 helper 를 닫고(unmount 만) runRoot 를 trash 로 옮긴 뒤 잠금을 놓는다. 보고 전
// 창에는 rename 한 번만 든다 — 지우는 것은 보고 뒤의 삭제자다 (business-rules.md 1절 ②).
// ctx 로 끊지 않는다 — 반쯤 닫은 세션은 helper 와 마운트를 남긴다. 닫기가 마감을 넘었는지는
// Worker 가 본다.
//
// Keep 은 받기만 한다 — Upper 는 늘 비어 있고 행선지를 쓰는 것은 bake · checkpoint 유닛이다.
// Finalize 가 helper 를 죽였으면 helper 에 말하지 않는다 — 죽은 helper 에 close 를 보내면
// 실패가 덧붙는다. 그때 옮기기는 abort 가 이미 했다.
func (s *runcOverlaySession) Close(context.Context, Keep) error {
	s.closeOnce.Do(func() {
		s.callMu.Lock()
		defer s.callMu.Unlock()
		if s.aborted {
			s.closeErr = s.release()
			return
		}
		var errs []error
		if err := s.send(runtimeWireRequest{Op: "close"}); err != nil {
			errs = append(errs, err)
		} else if response, err := s.receive(); err != nil {
			errs = append(errs, err)
		} else if response.Op != "closed" || response.Error != "" {
			errs = append(errs, fmt.Errorf("runtime close: %s", response.Error))
		}
		_ = s.stdin.Close()
		wait := make(chan error, 1)
		go func() { wait <- s.cmd.Wait() }()
		select {
		case err := <-wait:
			if err != nil {
				detail := s.helperErr.String()
				if detail == "" {
					errs = append(errs, fmt.Errorf("runtime helper exit: %w", err))
				} else {
					errs = append(errs, fmt.Errorf("runtime helper exit: %w: %s", err, detail))
				}
			}
		case <-time.After(3 * time.Second):
			if s.cmd.Process != nil {
				_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)
			}
			errs = append(errs, errors.New("runtime helper did not exit after close"))
		}
		if err := s.release(); err != nil {
			errs = append(errs, err)
		}
		s.closeErr = errors.Join(errs...)
	})
	return s.closeErr
}

// abort 는 helper 를 죽이고 작업 폴더를 trash 로 옮긴다 (business-rules.md 1절 ③).
// namespace 가 죽으면 그 안의 마운트도 사라지므로 옮길 때 남은 마운트가 없다.
func (s *runcOverlaySession) abort() error {
	_ = s.stdin.Close()
	if s.cmd.Process != nil {
		_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)
		_ = s.cmd.Wait()
	}
	return s.release()
}

// release 는 작업 폴더를 trash 로 옮기고 잠금을 놓는다 — 잠금은 rename 뒤에 놓는다. 앞에
// 놓으면 그 사이에 다른 데몬의 기동 청소가 같은 폴더를 옮기려 한다. 옮기기가 실패해도
// 지우지 않는다 — 폴더는 scratch 에 남고 다음 기동 청소가 거둔다.
func (s *runcOverlaySession) release() error {
	s.releaseOnce.Do(func() {
		if _, err := s.trash.Move(s.runRoot); err != nil {
			s.releaseErr = fmt.Errorf("move runtime session to trash: %w", err)
		}
		_ = s.lock.Release()
	})
	return s.releaseErr
}

func (s *runcOverlaySession) Environment() *execenv.Record { return s.record }

func resolveProjectedDirectory(source string) (string, error) {
	if source == "" {
		return "", errors.New("source is empty")
	}
	abs, err := filepath.Abs(source)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(resolved)
	if err != nil || !st.IsDir() {
		return "", fmt.Errorf("%s is not a directory", resolved)
	}
	return resolved, nil
}

func resolveProjectedExecutable(rootfs, source string, optional bool) (string, error) {
	if source == "" {
		if optional {
			return "", nil
		}
		return "", errors.New("source is empty")
	}
	path := source
	var err error
	if !strings.ContainsRune(path, filepath.Separator) {
		path, err = exec.LookPath(path)
	} else {
		path, err = filepath.Abs(path)
	}
	if err != nil {
		return "", err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(path)
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("%s is not an executable regular file", path)
	}
	if err := checkProjectedExecutableDependencies(rootfs, path); err != nil {
		return "", err
	}
	return path, nil
}

func checkProjectedExecutableDependencies(rootfs, executable string) error {
	f, err := elf.Open(executable)
	if err == nil {
		defer f.Close()
		if section := f.Section(".interp"); section != nil {
			b, readErr := section.Data()
			if readErr != nil {
				return readErr
			}
			interpreter := strings.TrimRight(string(b), "\x00")
			if interpreter != "" {
				path, pathErr := beneathRoot(rootfs, interpreter)
				if pathErr != nil {
					return pathErr
				}
				if _, statErr := os.Stat(path); statErr != nil {
					return fmt.Errorf("ELF interpreter %s is absent from the prepared rootfs", interpreter)
				}
			}
		}
		libraries, libErr := f.ImportedLibraries()
		if libErr != nil {
			return libErr
		}
		for _, library := range libraries {
			if !rootfsLibraryExists(rootfs, library) {
				return fmt.Errorf("ELF dependency %s is absent from the prepared rootfs", library)
			}
		}
		return nil
	}
	file, openErr := os.Open(executable)
	if openErr != nil {
		return openErr
	}
	defer file.Close()
	line, _ := bufio.NewReader(io.LimitReader(file, 4096)).ReadString('\n')
	if !strings.HasPrefix(line, "#!") {
		return fmt.Errorf("%s is neither ELF nor a script with a shebang", executable)
	}
	fields := strings.Fields(strings.TrimPrefix(line, "#!"))
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "/") {
		return fmt.Errorf("%s has an invalid shebang", executable)
	}
	interpreter, pathErr := beneathRoot(rootfs, fields[0])
	if pathErr != nil {
		return pathErr
	}
	if _, statErr := os.Stat(interpreter); statErr != nil {
		return fmt.Errorf("script interpreter %s is absent from the prepared rootfs", fields[0])
	}
	if fields[0] == "/usr/bin/env" {
		command := ""
		for _, field := range fields[1:] {
			if field == "-S" || strings.HasPrefix(field, "-") || strings.Contains(field, "=") {
				continue
			}
			command = field
			break
		}
		if command == "" || !rootfsExecutableExists(rootfs, command) {
			return fmt.Errorf("script interpreter selected by /usr/bin/env (%s) is absent from the prepared rootfs", command)
		}
	}
	return nil
}

func rootfsExecutableExists(rootfs, name string) bool {
	for _, dir := range []string{"usr/local/sbin", "usr/local/bin", "usr/sbin", "usr/bin", "sbin", "bin"} {
		st, err := os.Stat(filepath.Join(rootfs, dir, name))
		if err == nil && st.Mode().IsRegular() && st.Mode().Perm()&0o111 != 0 {
			return true
		}
	}
	return false
}

func rootfsLibraryExists(rootfs, name string) bool {
	archDir := ""
	switch runtime.GOARCH {
	case "amd64":
		archDir = "x86_64-linux-gnu"
	case "arm64":
		archDir = "aarch64-linux-gnu"
	}
	dirs := []string{"lib", "lib64", "usr/lib", "usr/lib64"}
	if archDir != "" {
		dirs = append(dirs, filepath.Join("lib", archDir), filepath.Join("usr/lib", archDir))
	}
	for _, dir := range dirs {
		if _, err := os.Stat(filepath.Join(rootfs, dir, name)); err == nil {
			return true
		}
	}
	return false
}

func beneathRoot(root, absolute string) (string, error) {
	if !filepath.IsAbs(absolute) {
		return "", fmt.Errorf("path %q is not absolute", absolute)
	}
	rel := strings.TrimPrefix(filepath.Clean(absolute), string(filepath.Separator))
	if rel == "" || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes the prepared rootfs", absolute)
	}
	return filepath.Join(root, rel), nil
}

// RunRuncOverlayHelper는 unshare가 다시 실행한 숨은 entrypoint다. stdout은 오직
// framed JSON protocol에 쓰고, 진단은 stderr로 보낸다.
func RunRuncOverlayHelper(in io.Reader, out, errOut io.Writer) int {
	h := &overlayRuntimeHelper{
		decoder: json.NewDecoder(bufio.NewReader(in)), encoder: json.NewEncoder(out), errOut: errOut,
	}
	err := h.loop()
	h.cancel()
	h.waitRun()
	cleanupErr := h.cleanup()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if cleanupErr != nil {
		fmt.Fprintln(errOut, cleanupErr)
		return 1
	}
	return 0
}

type overlayRuntimeHelper struct {
	decoder    *json.Decoder
	encoder    *json.Encoder
	errOut     io.Writer
	sendMu     sync.Mutex
	mu         sync.Mutex
	open       *runtimeWireOpen
	projection runtimeWireProjection
	active     *exec.Cmd
	activeID   string
	running    bool
	canceled   bool
	runWG      sync.WaitGroup
	counter    atomic.Uint64
	merged     string
	upper      string // 명시 훑기의 곳 — 이 단계가 쓴 것이 여기 있다
	lowerRO    string
	inRO       string
	sshRO      string
	bundle     string
	state      string
}

func (h *overlayRuntimeHelper) loop() error {
	for {
		var request runtimeWireRequest
		if err := h.decoder.Decode(&request); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		switch request.Op {
		case "open":
			if request.Open == nil {
				h.respond(runtimeWireResponse{Op: "opened", Error: "missing open payload"})
				continue
			}
			if err := h.openSession(*request.Open); err != nil {
				h.respond(runtimeWireResponse{Op: "opened", Error: err.Error()})
				continue
			}
			h.respond(runtimeWireResponse{Op: "opened"})
		case "project":
			if h.open == nil || request.Projection == nil {
				h.respond(runtimeWireResponse{Op: "projected", Error: "runtime is not open"})
				continue
			}
			if err := validateHelperProjection(*request.Projection); err != nil {
				h.respond(runtimeWireResponse{Op: "projected", Error: err.Error()})
				continue
			}
			h.projection = *request.Projection
			h.respond(runtimeWireResponse{Op: "projected"})
		case "run":
			if h.open == nil || request.Process == nil {
				h.respond(runtimeWireResponse{Op: "error", Error: "runtime is not open"})
				continue
			}
			h.mu.Lock()
			alreadyRunning := h.running
			if !alreadyRunning {
				h.running = true
				h.canceled = false
				h.runWG.Add(1)
			}
			h.mu.Unlock()
			if alreadyRunning {
				h.respond(runtimeWireResponse{Op: "error", Error: "a runtime process is already active"})
				continue
			}
			go h.runProcess(*request.Process)
		case "cancel":
			h.cancel()
		case "finalize":
			if request.Finalize == nil {
				h.respond(runtimeWireResponse{Op: "finalized", Error: "missing finalize payload"})
				continue
			}
			h.waitRun()
			result, err := h.finalize(*request.Finalize)
			response := runtimeWireResponse{Op: "finalized", Finalize: &result}
			if err != nil {
				response.Error = err.Error()
			}
			h.respond(response)
		case "close":
			h.cancel()
			h.waitRun()
			err := h.cleanup()
			response := runtimeWireResponse{Op: "closed"}
			if err != nil {
				response.Error = err.Error()
			}
			h.respond(response)
			return nil
		default:
			h.respond(runtimeWireResponse{Op: "error", Error: "unknown runtime helper operation " + request.Op})
		}
	}
}

func (h *overlayRuntimeHelper) respond(response runtimeWireResponse) {
	h.sendMu.Lock()
	defer h.sendMu.Unlock()
	if err := h.encoder.Encode(response); err != nil {
		fmt.Fprintln(h.errOut, err)
	}
}

func (h *overlayRuntimeHelper) openSession(open runtimeWireOpen) error {
	if h.open != nil {
		return errors.New("runtime session is already open")
	}
	for name, value := range map[string]string{
		"rootfs": open.RootFS, "workspace": open.Workspace, "$IN": open.In,
		"$OUT": open.Out, "run root": open.RunRoot,
	} {
		if strings.ContainsAny(value, ",:") {
			return fmt.Errorf("%s path contains a character unsafe for overlay options: %s", name, value)
		}
	}
	if open.UID <= 0 || open.GID <= 0 || open.UID >= open.SubUIDSize || open.GID >= open.SubGIDSize {
		return errors.New("rootfs uid/gid cannot be represented by the declared subordinate ranges")
	}
	workspace, err := filepath.EvalSymlinks(open.Workspace)
	if err != nil {
		return fmt.Errorf("resolve workspace: %w", err)
	}
	runRoot, err := filepath.EvalSymlinks(open.RunRoot)
	if err != nil {
		return fmt.Errorf("resolve runtime scratch: %w", err)
	}
	if pathsOverlap(workspace, runRoot) {
		return errors.New("runtime scratch and workspace must not contain each other")
	}
	var statfs unix.Statfs_t
	if err := unix.Statfs(runRoot, &statfs); err != nil {
		return err
	}
	if uint64(statfs.Type) == uint64(unix.OVERLAYFS_SUPER_MAGIC) {
		return errors.New("runtime scratch is on overlayfs; use a host bind volume")
	}
	for _, target := range []string{open.WorkspaceTarget, "/run/enode", "/tmp"} {
		path, err := beneathRoot(open.RootFS, target)
		if err != nil {
			return err
		}
		if st, err := os.Stat(path); err != nil || !st.IsDir() {
			return fmt.Errorf("prepared rootfs target %s is missing", target)
		}
	}
	if open.SSHDir != "" {
		if st, err := os.Stat(open.SSHDir); err != nil || !st.IsDir() {
			return fmt.Errorf("SSH projection source %s is unavailable", open.SSHDir)
		}
		target, err := beneathRoot(open.RootFS, "/home/"+open.UserName+"/.ssh")
		if err != nil {
			return err
		}
		if st, err := os.Stat(target); err != nil || !st.IsDir() {
			return fmt.Errorf("prepared rootfs SSH target for %s is missing", open.UserName)
		}
	}

	h.bundle = filepath.Join(runRoot, "bundle")
	h.state = filepath.Join(runRoot, "state")
	upper := filepath.Join(runRoot, "upper")
	h.upper = upper
	work := filepath.Join(runRoot, "work")
	h.merged = filepath.Join(runRoot, "merged")
	h.lowerRO = filepath.Join(runRoot, "lower-ro")
	h.inRO = filepath.Join(runRoot, "in-ro")
	h.sshRO = filepath.Join(runRoot, "ssh-ro")
	for _, dir := range []string{h.bundle, h.state, upper, work, h.merged, h.lowerRO, h.inRO, h.sshRO} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	if err := unix.Mount("", "/", "", unix.MS_REC|unix.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("make runtime mount namespace private: %w", err)
	}
	if err := bindTreeReadOnly(workspace, h.lowerRO); err != nil {
		return fmt.Errorf("bind workspace lower: %w", err)
	}
	if err := bindTreeReadOnly(open.In, h.inRO); err != nil {
		_ = unix.Unmount(h.lowerRO, unix.MNT_DETACH)
		return fmt.Errorf("bind $IN read-only: %w", err)
	}
	if open.SSHDir != "" {
		if err := bindTreeReadOnly(open.SSHDir, h.sshRO); err != nil {
			_ = unix.Unmount(h.inRO, unix.MNT_DETACH)
			_ = unix.Unmount(h.lowerRO, unix.MNT_DETACH)
			return fmt.Errorf("bind SSH projection read-only: %w", err)
		}
	}
	options := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", h.lowerRO, upper, work)
	if err := unix.Mount("overlay", h.merged, "overlay", 0, options); err != nil {
		if open.SSHDir != "" {
			_ = unix.Unmount(h.sshRO, unix.MNT_DETACH)
		}
		_ = unix.Unmount(h.inRO, unix.MNT_DETACH)
		_ = unix.Unmount(h.lowerRO, unix.MNT_DETACH)
		return fmt.Errorf("mount overlay workspace: %w", err)
	}
	copy := open
	copy.Workspace = workspace
	copy.RunRoot = runRoot
	h.open = &copy
	return nil
}

func bindTreeReadOnly(source, target string) error {
	if err := unix.Mount(source, target, "", unix.MS_BIND|unix.MS_REC, ""); err != nil {
		return err
	}
	attr := &unix.MountAttr{Attr_set: unix.MOUNT_ATTR_RDONLY}
	if err := unix.MountSetattr(unix.AT_FDCWD, target, unix.AT_RECURSIVE, attr); err != nil {
		_ = unix.Unmount(target, unix.MNT_DETACH)
		return fmt.Errorf("make bind tree recursively read-only: %w", err)
	}
	return nil
}

func pathsOverlap(a, b string) bool {
	relAB, errAB := filepath.Rel(a, b)
	relBA, errBA := filepath.Rel(b, a)
	inside := func(rel string, err error) bool {
		return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
	}
	return inside(relAB, errAB) || inside(relBA, errBA)
}

func validateHelperProjection(projection runtimeWireProjection) error {
	for name, source := range map[string]string{
		"harness": projection.Harness, "enode": projection.Enode,
		"instrumentation": projection.Instrumentation, "credential": projection.Credential,
	} {
		if source == "" && (name == "credential") {
			continue
		}
		st, err := os.Stat(source)
		if err != nil {
			return fmt.Errorf("%s source: %w", name, err)
		}
		if name == "instrumentation" && !st.IsDir() {
			return fmt.Errorf("%s source is not a directory", name)
		}
		if name != "instrumentation" && (!st.Mode().IsRegular() || st.Mode().Perm()&0o111 == 0) {
			return fmt.Errorf("%s source is not executable", name)
		}
	}
	return nil
}

func (h *overlayRuntimeHelper) runProcess(process runtimeWireProcess) {
	defer h.runWG.Done()
	defer func() {
		h.mu.Lock()
		h.active = nil
		h.activeID = ""
		h.running = false
		h.canceled = false
		h.mu.Unlock()
	}()
	config, err := h.ociConfig(process)
	if err != nil {
		h.respond(runtimeWireResponse{Op: "run-done", ExitCode: -1, Error: err.Error()})
		return
	}
	b, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		h.respond(runtimeWireResponse{Op: "run-done", ExitCode: -1, Error: err.Error()})
		return
	}
	if err := os.WriteFile(filepath.Join(h.bundle, "config.json"), b, 0o600); err != nil {
		h.respond(runtimeWireResponse{Op: "run-done", ExitCode: -1, Error: err.Error()})
		return
	}
	id := fmt.Sprintf("enode-%d-%d", os.Getpid(), h.counter.Add(1))
	cmd := child(exec.Command("runc", "--root", h.state, "run", "--bundle", h.bundle, id))
	cmd.Stdin = bytes.NewReader(process.Stdin)
	cmd.Stdout = runtimeResponseWriter{helper: h, op: "stdout"}
	cmd.Stderr = runtimeResponseWriter{helper: h, op: "stderr"}
	if err = cmd.Start(); err != nil {
		h.respond(runtimeWireResponse{Op: "run-done", ExitCode: -1, Error: err.Error()})
		return
	}
	h.mu.Lock()
	h.active, h.activeID = cmd, id
	canceled := h.canceled
	h.mu.Unlock()
	if canceled && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	err = cmd.Wait()
	exitCode := -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	_ = child(exec.Command("runc", "--root", h.state, "delete", "--force", id)).Run()
	response := runtimeWireResponse{Op: "run-done", ExitCode: exitCode}
	if err != nil {
		response.Error = err.Error()
	}
	h.respond(response)
}

type runtimeResponseWriter struct {
	helper *overlayRuntimeHelper
	op     string
}

func (w runtimeResponseWriter) Write(p []byte) (int, error) {
	written := len(p)
	for len(p) > 0 {
		n := len(p)
		if n > 32<<10 {
			n = 32 << 10
		}
		w.helper.respond(runtimeWireResponse{Op: w.op, Data: append([]byte(nil), p[:n]...)})
		p = p[n:]
	}
	return written, nil
}

func (h *overlayRuntimeHelper) cancel() {
	h.mu.Lock()
	h.canceled = true
	cmd, id := h.active, h.activeID
	h.mu.Unlock()
	if id != "" {
		_ = child(exec.Command("runc", "--root", h.state, "kill", id, "KILL")).Run()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

func (h *overlayRuntimeHelper) waitRun() { h.runWG.Wait() }

// finalize 는 merged 에서 collect · stat · diff 를, upper 에서 명시 훑기를 한다.
// 마감은 요청이 나른 시각이다 — Worker 의 ctx 는 이 프로세스에 닿지 않는다.
func (h *overlayRuntimeHelper) finalize(spec FinalizeSpec) (FinalizeResult, error) {
	if h.open == nil {
		return FinalizeResult{}, errors.New("runtime is not open")
	}
	ctx, cancel := context.WithCancel(context.Background())
	if !spec.Deadline.IsZero() {
		ctx, cancel = context.WithDeadline(context.Background(), spec.Deadline)
	}
	defer cancel()
	spec.Workspace = h.merged
	spec.Out = h.open.Out
	upper := h.upper
	return finalizeLocal(ctx, spec, func(ctx context.Context, limits discoverLimits) *Discovery {
		return walkUpper(ctx, upper, limits, isOverlayWhiteout)
	})
}

// isOverlayWhiteout 은 overlay 가 upper 에 남기는 지운 표시다 — 문자 장치 0:0.
func isOverlayWhiteout(d fs.DirEntry) bool {
	if d.Type()&fs.ModeCharDevice == 0 {
		return false
	}
	fi, err := d.Info()
	if err != nil {
		return false
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	return ok && st.Rdev == 0
}

// cleanup 은 unmount 만 한다. runRoot 를 지우지 않는다 — 밖의 rename 이 trash 로 옮기고
// 배경 삭제자가 지운다 (business-rules.md 1절 ④). 부모가 죽어 최종 방어 경로로 불렸으면
// 작업 폴더는 scratch 에 남고 다음 기동 청소가 거둔다.
func (h *overlayRuntimeHelper) cleanup() error {
	merged, lowerRO, inRO, sshRO := h.merged, h.lowerRO, h.inRO, h.sshRO
	// close request와 helper의 최종 방어 경로가 모두 cleanup을 부른다.
	// 첫 호출이 소유권을 가져가야 두 번째 호출이 이미 푼 mount path를 다시
	// unmount하고 helper를 실패로 끝내지 않는다.
	h.merged, h.lowerRO, h.inRO, h.sshRO = "", "", "", ""
	h.open = nil

	var errs []error
	if merged != "" {
		if err := unix.Unmount(merged, unix.MNT_DETACH); err != nil && !errors.Is(err, unix.EINVAL) {
			errs = append(errs, fmt.Errorf("unmount merged workspace: %w", err))
		}
	}
	if lowerRO != "" {
		if err := unix.Unmount(lowerRO, unix.MNT_DETACH); err != nil && !errors.Is(err, unix.EINVAL) {
			errs = append(errs, fmt.Errorf("unmount lower workspace: %w", err))
		}
	}
	if inRO != "" {
		if err := unix.Unmount(inRO, unix.MNT_DETACH); err != nil && !errors.Is(err, unix.EINVAL) {
			errs = append(errs, fmt.Errorf("unmount $IN projection: %w", err))
		}
	}
	if sshRO != "" {
		if err := unix.Unmount(sshRO, unix.MNT_DETACH); err != nil && !errors.Is(err, unix.EINVAL) {
			errs = append(errs, fmt.Errorf("unmount SSH projection: %w", err))
		}
	}
	return errors.Join(errs...)
}

type ociConfig struct {
	OCIVersion string     `json:"ociVersion"`
	Process    ociProcess `json:"process"`
	Root       ociRoot    `json:"root"`
	Hostname   string     `json:"hostname"`
	Mounts     []ociMount `json:"mounts"`
	Linux      ociLinux   `json:"linux"`
}

type ociProcess struct {
	Terminal        bool            `json:"terminal"`
	User            ociUser         `json:"user"`
	Args            []string        `json:"args"`
	Env             []string        `json:"env"`
	Cwd             string          `json:"cwd"`
	Capabilities    ociCapabilities `json:"capabilities"`
	Rlimits         []ociRlimit     `json:"rlimits,omitempty"`
	NoNewPrivileges bool            `json:"noNewPrivileges"`
}

type ociUser struct {
	UID uint32 `json:"uid"`
	GID uint32 `json:"gid"`
}
type ociCapabilities struct {
	Bounding    []string `json:"bounding"`
	Effective   []string `json:"effective"`
	Inheritable []string `json:"inheritable"`
	Permitted   []string `json:"permitted"`
	Ambient     []string `json:"ambient"`
}
type ociRlimit struct {
	Type string `json:"type"`
	Hard uint64 `json:"hard"`
	Soft uint64 `json:"soft"`
}
type ociRoot struct {
	Path     string `json:"path"`
	Readonly bool   `json:"readonly"`
}
type ociMount struct {
	Destination string   `json:"destination"`
	Type        string   `json:"type"`
	Source      string   `json:"source"`
	Options     []string `json:"options,omitempty"`
}
type ociLinux struct {
	UIDMappings   []ociIDMapping `json:"uidMappings"`
	GIDMappings   []ociIDMapping `json:"gidMappings"`
	Namespaces    []ociNamespace `json:"namespaces"`
	MaskedPaths   []string       `json:"maskedPaths,omitempty"`
	ReadonlyPaths []string       `json:"readonlyPaths,omitempty"`
}
type ociIDMapping struct {
	ContainerID uint32 `json:"containerID"`
	HostID      uint32 `json:"hostID"`
	Size        uint32 `json:"size"`
}
type ociNamespace struct {
	Type string `json:"type"`
}

func (h *overlayRuntimeHelper) ociConfig(process runtimeWireProcess) (ociConfig, error) {
	if h.open == nil || len(process.Argv) == 0 {
		return ociConfig{}, errors.New("runtime process is incomplete")
	}
	open := *h.open
	cwd := open.WorkspaceTarget
	if process.Cwd != "" {
		if !path.IsAbs(process.Cwd) {
			return ociConfig{}, errors.New("runtime process cwd must be absolute")
		}
		cwd = path.Clean(process.Cwd)
	}
	mounts := []ociMount{
		{Destination: "/proc", Type: "proc", Source: "proc", Options: []string{"nosuid", "noexec", "nodev"}},
		{Destination: "/dev", Type: "tmpfs", Source: "tmpfs", Options: []string{"nosuid", "strictatime", "mode=755", "size=65536k"}},
		{Destination: "/dev/pts", Type: "devpts", Source: "devpts", Options: []string{"nosuid", "noexec", "newinstance", "ptmxmode=0666", "mode=0620", "gid=5"}},
		{Destination: "/dev/shm", Type: "tmpfs", Source: "shm", Options: []string{"nosuid", "noexec", "nodev", "mode=1777", "size=65536k"}},
		{Destination: "/dev/mqueue", Type: "mqueue", Source: "mqueue", Options: []string{"nosuid", "noexec", "nodev"}},
		{Destination: open.WorkspaceTarget, Type: "bind", Source: h.merged, Options: []string{"rbind", "rw", "nosuid", "nodev"}},
		{Destination: "/run/enode", Type: "tmpfs", Source: "tmpfs", Options: []string{"nosuid", "nodev", "mode=0755", "size=16m"}},
		{Destination: runtimeInTarget, Type: "bind", Source: h.inRO, Options: []string{"rbind", "ro", "nosuid", "nodev", "noexec"}},
		{Destination: runtimeOutTarget, Type: "bind", Source: open.Out, Options: []string{"rbind", "rw", "nosuid", "nodev", "noexec"}},
	}
	tmpOptions := []string{"nosuid", "nodev", "mode=1777", "size=" + strconv.FormatInt(open.TmpSize, 10)}
	if !open.TmpExecutable {
		tmpOptions = append(tmpOptions, "noexec")
	}
	mounts = append(mounts, ociMount{Destination: "/tmp", Type: "tmpfs", Source: "tmpfs", Options: tmpOptions})
	if open.SSHDir != "" {
		mounts = append(mounts, ociMount{Destination: "/home/" + open.UserName + "/.ssh", Type: "bind",
			Source: h.sshRO, Options: []string{"rbind", "ro", "nosuid", "nodev", "noexec"}})
	}
	for _, projected := range []struct {
		source, target string
		options        []string
	}{
		{h.projection.Instrumentation, runtimeInstrumentationTarget, []string{"rbind", "rw", "nosuid", "nodev", "noexec"}},
		{h.projection.Harness, runtimeHarnessTarget, []string{"bind", "ro", "nosuid", "nodev"}},
		{h.projection.Enode, runtimeEnodeTarget, []string{"bind", "ro", "nosuid", "nodev"}},
		{h.projection.Credential, runtimeCredentialTarget, []string{"bind", "ro", "nosuid", "nodev"}},
	} {
		if projected.source != "" {
			mounts = append(mounts, ociMount{Destination: projected.target, Type: "bind",
				Source: projected.source, Options: projected.options})
		}
	}
	return ociConfig{
		OCIVersion: "1.0.2", Root: ociRoot{Path: open.RootFS, Readonly: true}, Hostname: "enode-step",
		Process: ociProcess{
			Terminal: false, User: ociUser{UID: uint32(open.UID), GID: uint32(open.GID)},
			Args: append([]string(nil), process.Argv...), Env: runtimeProcessEnv(process.Env, open),
			Cwd: cwd, Capabilities: ociCapabilities{}, NoNewPrivileges: true,
			Rlimits: []ociRlimit{{Type: "RLIMIT_NOFILE", Hard: 1024, Soft: 1024}},
		},
		Mounts: mounts,
		Linux: ociLinux{
			UIDMappings:   runtimeIDMappings(open.UID, open.SubUIDSize),
			GIDMappings:   runtimeIDMappings(open.GID, open.SubGIDSize),
			Namespaces:    []ociNamespace{{Type: "pid"}, {Type: "ipc"}, {Type: "uts"}, {Type: "mount"}, {Type: "cgroup"}, {Type: "user"}},
			MaskedPaths:   []string{"/proc/acpi", "/proc/asound", "/proc/kcore", "/proc/keys", "/proc/latency_stats", "/proc/timer_list", "/proc/timer_stats", "/proc/sched_debug", "/sys/firmware", "/proc/scsi"},
			ReadonlyPaths: []string{"/proc/bus", "/proc/fs", "/proc/irq", "/proc/sys", "/proc/sysrq-trigger"},
		},
	}, nil
}

func runtimeIDMappings(target, total int) []ociIDMapping {
	mappings := []ociIDMapping{{ContainerID: 0, HostID: 1, Size: uint32(target)}}
	mappings = append(mappings, ociIDMapping{ContainerID: uint32(target), HostID: 0, Size: 1})
	// outer namespace의 host id 0은 --map-root-user가 현재 사용자를 위해
	// 더한 한 칸이고 1..total이 --map-auto의 subordinate range다.
	if rest := total - target; rest > 0 {
		mappings = append(mappings, ociIDMapping{ContainerID: uint32(target + 1), HostID: uint32(target + 1), Size: uint32(rest)})
	}
	return mappings
}

func runtimeProcessEnv(source []string, open runtimeWireOpen) []string {
	replace := map[string]string{
		"PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"HOME": "/home/" + open.UserName, "USER": open.UserName, "LOGNAME": open.UserName,
		"LANG": open.Locale, "LC_ALL": open.Locale, "PWD": open.WorkspaceTarget,
		"TMPDIR": "/tmp", "ENODE_WORKSPACE": open.WorkspaceTarget,
	}
	seen := map[string]bool{}
	result := make([]string, 0, len(source)+len(replace))
	for _, item := range source {
		name, _, ok := strings.Cut(item, "=")
		if !ok || name == "" || seen[name] {
			continue
		}
		seen[name] = true
		if value, fixed := replace[name]; fixed {
			result = append(result, name+"="+value)
			delete(replace, name)
		} else {
			result = append(result, item)
		}
	}
	for _, name := range []string{"PATH", "HOME", "USER", "LOGNAME", "LANG", "LC_ALL", "PWD", "TMPDIR", "ENODE_WORKSPACE"} {
		if value, ok := replace[name]; ok {
			result = append(result, name+"="+value)
		}
	}
	return result
}

// ExecutionRuntimeVerifier는 env check/apply가 product와 같은 adapter로 수행하는
// smoke다. native는 추가 격리 경계가 없으므로 no-op이고 runc-overlay만 실제로
// namespace를 연다.
type ExecutionRuntimeVerifier struct{}

func (ExecutionRuntimeVerifier) Verify(ctx context.Context, doc execenv.Document, binding execenv.Binding, rootfs string, manifest execenv.Manifest) error {
	if doc.Profile.Runtime.Driver == "native" {
		return nil
	}
	runtimeImpl, err := newRuncOverlayRuntime(doc, binding, manifest, rootfs)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(binding.Scratch, 0o700); err != nil {
		return err
	}
	probeRoot, err := os.MkdirTemp(binding.Scratch, "enode-smoke-io-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(probeRoot)
	in, out := filepath.Join(probeRoot, "in"), filepath.Join(probeRoot, "out")
	if err := os.MkdirAll(in, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(in, "probe"), []byte("input\n"), 0o644); err != nil {
		return err
	}
	marker := ".enode-runtime-smoke-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	lowerMarker := filepath.Join(binding.Workspace, marker)
	if _, err := os.Stat(lowerMarker); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("runtime smoke marker already exists: %s", lowerMarker)
	}
	session, err := runtimeImpl.Open(ctx, RuntimeSpec{Dir: binding.Workspace, In: in, Out: out})
	if err != nil {
		return err
	}
	runRoot := ""
	if concrete, ok := session.(*runcOverlaySession); ok {
		runRoot = concrete.runRoot
	}
	defer session.Close(ctx, Keep{}) //nolint:errcheck
	tmpCheck := "/tmp/enode-runtime-smoke"
	execExpectation := tmpCheck
	if !doc.Profile.Runtime.Tmp.Executable {
		execExpectation = "! " + tmpCheck
	}
	script := fmt.Sprintf(`set -eu
test "$(id -u)" = %d
test "$(id -g)" = %d
test "$PWD" = %q
test "$(cat "$IN/probe")" = input
! (printf bad > "$IN/probe") 2>/dev/null
printf workspace > %q
printf output > "$OUT/probe"
printf '#!/bin/sh\nexit 0\n' > %s
chmod +x %s
%s
`, doc.Profile.RootFS.User.UID, doc.Profile.RootFS.User.GID,
		doc.Profile.Runtime.WorkspaceTarget, marker, tmpCheck, tmpCheck, execExpectation)
	if doc.Profile.Runtime.Credentials.SSH == "readonly" {
		script += `! (printf bad > "$HOME/.ssh/.enode-runtime-smoke") 2>/dev/null
`
	}
	var stdout, stderr bytes.Buffer
	code, runErr := session.Run(ctx, ProcessSpec{Argv: []string{"/bin/sh", "-c", script},
		Env: []string{"IN=" + runtimeInTarget, "OUT=" + runtimeOutTarget}, Stdout: &stdout, Stderr: &stderr})
	closeErr := session.Close(ctx, Keep{})
	if _, statErr := os.Stat(lowerMarker); !errors.Is(statErr, os.ErrNotExist) {
		if statErr == nil {
			_ = os.Remove(lowerMarker)
		}
		return fmt.Errorf("runtime smoke modified or could not verify the original workspace: %v", statErr)
	}
	if runErr != nil || code != 0 {
		return fmt.Errorf("runtime smoke exited %d: %v: %s%s", code, runErr, stdout.String(), stderr.String())
	}
	if closeErr != nil {
		return closeErr
	}
	// 닫기가 작업 폴더를 trash 로 옮겼다 — 원래 자리에 없어야 한다. 지우는 것은 데몬의 삭제자다
	if runRoot != "" {
		if _, err := os.Stat(runRoot); !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("runtime smoke left session state at %s", runRoot)
		}
	}
	b, err := os.ReadFile(filepath.Join(out, "probe"))
	if err != nil || string(b) != "output" {
		return fmt.Errorf("runtime smoke did not project writable $OUT: %w", err)
	}
	return nil
}
