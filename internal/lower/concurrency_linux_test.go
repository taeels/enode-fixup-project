package lower

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// 동시성 (계획 3.2) — 같은 기계의 여러 노드가 같은 파일과 잠금을 쓴다. 형제는 한 프로세스의 Dir 둘로 흉내 내고
// (flock 은 열린 파일마다다), 진짜 자식 프로세스 하나를 더한다.

// childEnv 는 자식 프로세스 형제의 입구다. 값은 "<lowers>\n<lower 루트>" 다.
const childEnv = "ENODE_LOWER_TEST_CHILD"

func TestMain(m *testing.M) {
	if spec := os.Getenv(childEnv); spec != "" {
		os.Exit(holdAsChild(spec))
	}
	os.Exit(m.Run())
}

// holdAsChild 는 자식 프로세스에서 공유를 쥐고 "held" 를 쓴 뒤 죽을 때까지 기다린다.
func holdAsChild(spec string) int {
	lowers, lowerRoot, _ := strings.Cut(spec, "\n")
	root, err := ReadRoot(lowerRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	d, err := Peek(lowers, root)
	if err != nil || d == nil {
		fmt.Fprintln(os.Stderr, "peek:", err)
		return 1
	}
	s, ok, err := d.TryShared(Holder{Node: "child", Label: "child", Role: RoleCandidate})
	if err != nil || !ok {
		fmt.Fprintln(os.Stderr, "shared:", ok, err)
		return 1
	}
	defer func() { _ = s.Release() }()
	fmt.Println("held")
	time.Sleep(10 * time.Minute)
	return 0
}

// 자식 프로세스 형제가 공유를 쥔다 -> 부모의 Exclusive 가 기다린다 -> 자식을 SIGKILL -> pollEvery 한 번 안에 잡힌다.
func TestChildSiblingIsKilled(t *testing.T) {
	fastPoll(t)
	f := newFixture(t)
	d := f.open(t)
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), childEnv+"="+f.lowers+"\n"+f.lower)
	cmd.Stderr = os.Stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Wait() //nolint:errcheck
	defer func() { _ = cmd.Process.Kill() }()
	line, err := bufio.NewReader(out).ReadString('\n')
	if err != nil || line != "held\n" {
		t.Fatalf("the child said %q, %v", line, err)
	}
	if hs, _ := d.Holders(); len(hs) != 1 || hs[0].Node != "child" || hs[0].PID != cmd.Process.Pid {
		t.Fatalf("holders = %+v", hs)
	}

	got := make(chan error, 1)
	var ex *Exclusive
	go func() {
		var err error
		ex, err = d.Exclusive(t.Context(), time.Hour, nil)
		got <- err
	}()
	select {
	case err := <-got:
		t.Fatalf("Exclusive did not wait for the child: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	start := time.Now()
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := <-got; err != nil {
		t.Fatal(err)
	}
	took := time.Since(start)
	defer func() { _ = ex.Release() }()
	t.Logf("the exclusive lock was taken %v after SIGKILL (poll every %v)", took, pollEvery)
	if took > time.Second {
		t.Errorf("the kernel released the child's lock %v after SIGKILL", took)
	}
	if hs, _ := d.Holders(); len(hs) != 0 {
		t.Errorf("the killed child is still a holder: %+v", hs)
	}
}

// 고루틴 16 이 같은 키를 Open 한다 (Dir 마다 fd 따로) — 모두 성공하고 lower.json 은 JSON 이고 신원 칸이 같다.
func TestOpenConcurrently(t *testing.T) {
	f := newFixture(t)
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			root := f.root
			if i%2 == 1 {
				root.Path = fmt.Sprintf("/alias/%d", i) // bind 별칭 — paths 를 더하는 쓰기가 겹친다
			}
			_, err := Open(f.lowers, root, time.Now())
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}
	b, err := os.ReadFile(filepath.Join(f.lowers, f.root.Key.String(), identityFile))
	if err != nil {
		t.Fatal(err)
	}
	var rec Identity
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatalf("lower.json is not JSON: %v\n%s", err, b)
	}
	if !rec.matchesKey(f.root.Key) || rec.BirthNs != f.root.BirthNs || rec.OwnerUID != os.Getuid() || len(rec.Paths) == 0 {
		t.Errorf("lower.json = %+v", rec)
	}
}

// 쓰는 쪽 하나와 읽는 쪽 넷 — 오류 0 · 깨진 읽기 0. state.json 과 쥔 사람 기록이 같은 모양이다.
func TestWritesAndReadsConcurrently(t *testing.T) {
	f := newFixture(t)
	d := f.open(t)
	bake, _, _ := d.TryBake()
	defer func() { _ = bake.Release() }()
	s := mustShared(t, d, Holder{Node: "writer", Role: RoleCandidate})
	defer func() { _ = s.Release() }()
	reader := sibling(t, f)

	stop := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	reads, broken := 0, 0
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				st, err := reader.ReadState()
				hs, herr := reader.Holders()
				mu.Lock()
				reads++
				if err != nil || herr != nil || st.Schema != 1 || len(hs) != 1 {
					broken++
					if broken == 1 {
						t.Errorf("a broken read: %+v %v %+v %v", st, err, hs, herr)
					}
				}
				mu.Unlock()
			}
		}()
	}
	phases := []Phase{PhaseBuilding, PhasePending, PhaseMerging, PhaseCommitted}
	for i := range 200 {
		if err := bake.WriteState(State{Phase: phases[i%4], Since: time.Now()}); err != nil {
			t.Fatal(err)
		}
		if err := s.Update(Holder{Role: RoleCandidate, Acks: i % 2}); err != nil {
			t.Fatal(err)
		}
	}
	close(stop)
	wg.Wait()
	t.Logf("reads %d, broken %d", reads, broken)
	if reads == 0 {
		t.Error("no read ran")
	}
}

// 잠금 fd 는 자식에게 안 샌다 (계획 4절 ③) — 공유 · 기록 · 굽기 · 배타를 쥔 채 띄운 셸이 자기 fd 를 읽는다.
func TestLockFdsDoNotLeak(t *testing.T) {
	a, b := newFixture(t), newFixture(t)
	da, db := a.open(t), b.open(t)
	s := mustShared(t, da, Holder{Node: "node-a"})
	defer func() { _ = s.Release() }()
	bake, _, _ := da.TryBake()
	defer func() { _ = bake.Release() }()
	ex, err := db.Exclusive(t.Context(), time.Hour, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ex.Release() }()
	out, err := exec.Command("/bin/sh", "-c", `for f in /proc/$$/fd/*; do readlink "$f"; done; true`).Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{da.Path, db.Path} {
		if strings.Contains(string(out), dir) {
			t.Errorf("a child process holds a file under %s:\n%s", dir, out)
		}
	}
}

// BenchmarkHolders 는 기록 여덟 가운데 넷이 살아 있을 때 Holders 한 번이다 (계획 3.2 · 측정 8 은 0.07 ms).
func BenchmarkHolders(b *testing.B) {
	f := newFixtureB(b)
	d, err := Open(f.lowers, f.root, time.Now())
	if err != nil {
		b.Fatal(err)
	}
	for i := range 8 {
		s, ok, err := d.TryShared(Holder{Node: fmt.Sprintf("node-%d", i), Role: RoleCandidate})
		if err != nil || !ok {
			b.Fatal(err)
		}
		if i%2 == 0 {
			_ = s.Release()
		} else {
			defer func() { _ = s.Release() }()
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if hs, err := d.Holders(); err != nil || len(hs) != 4 {
			b.Fatalf("holders %d, %v", len(hs), err)
		}
	}
}

// BenchmarkForeignMounts 는 이 기계의 같은 uid 프로세스 전부를 훑는 한 번이다 (FD 계획 2.6 은 4.7 ms).
func BenchmarkForeignMounts(b *testing.B) {
	f := newFixtureB(b)
	var scan MountScan
	for i := 0; i < b.N; i++ {
		var err error
		if scan, err = ForeignMounts(f.root); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(scan.Namespaces), "namespaces")
	b.ReportMetric(float64(scan.Unreadable), "unreadable")
}

func newFixtureB(b *testing.B) fixture {
	b.Helper()
	dir := b.TempDir()
	f := fixture{dir: dir, lower: filepath.Join(dir, "ws"), lowers: filepath.Join(dir, "lowers")}
	if err := os.Mkdir(f.lower, 0o755); err != nil {
		b.Fatal(err)
	}
	root, err := ReadRoot(f.lower)
	if err != nil {
		b.Fatal(err)
	}
	f.root = root
	return f
}
