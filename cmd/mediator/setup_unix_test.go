//go:build !windows

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/taeels/enode/internal/store"
)

// setup 하위 명령을 안에서 몬다 - U8 이 뽑아 놓은 runSetupWith 와 setupIO 가
// 그 자리다. 프로세스 밖에서 보이는 계약은 entrypoint_unix_test.go 가 이미
// 들고 있고, 이 파일이 드는 것은 그 계약이 지나가는 157 문장이다.
//
// 왜 가짜 adminDB 인가 - setup 이 관리자 자격으로 하는 것은 셋뿐이고
// (setup.go 의 adminDB), 그 셋을 흉내 내면 롤도 데이터베이스도 없는 판 ·
// 둘 다 있는 판 · pg_roles 를 못 읽는 판을 진짜 Postgres 없이 만들 수 있다.
// 진짜 서버로는 만들 수 없는 것이 그중 셋째다.
//
// 진짜 Postgres 를 그래도 쓰는 자리가 있다 - 스키마를 올리는 구간부터다.
// deps.open 이 돌려주는 것이 *store.Store 인터페이스가 아니라 구체 타입이고,
// Migrate 는 진짜 판에 SQL 을 보낸다. 그 자리는 U8 의 scratchDB 를 그대로
// 쓴다 (U8 인계 §4 가 같은 판을 쓰라고 적었다).
//
// 왜 빌드 태그로 가르는가 - 이 파일이 쓰는 scratchDB · freePort ·
// mediatorConfig · isolate 가 전부 !windows 인 entrypoint_unix_test.go 에
// 산다. 베껴 오는 안을 기각했다: 같은 사실이 두 곳에 살면 한쪽만 낡고,
// 그중 하나가 "판을 따로 파는 이유" 같은 판정을 바꾸는 사실이다.
// 이 패키지는 오늘 이미 유닉스에서만 시험되고, 커버리지 계약도
// platform: linux/amd64 를 명시한다 (.coverage-contract.yml).

// captureOutput 은 os.Stdout 과 os.Stderr 를 파이프로 바꿔 fn 이 찍은 것을
// 돌려준다.
//
// setup 은 fmt.Println 으로 os.Stdout 에 직접 쓴다. 출력 대상을 io.Writer 로
// 바꾸는 안은 setupIO 를 만든 U8 이 이미 기각했다 (setup.go 의 setupIO 주석) -
// 이음매를 뽑는 일이 CLI 의 모양을 바꾸기 시작하면 무엇을 시험하는지가
// 흐려진다. cmd/runctl · cmd/enodectl 이 같은 이유로 같은 것을 들고 있다.
//
// 고루틴으로 비우는 이유 - 파이프 버퍼가 차면 fn 이 쓰기에서 멈추고,
// 멈추면 왜 멈췄는지 출력에 아무것도 안 남는다.
func captureOutput(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe for stdout: %v", err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe for stderr: %v", err)
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW

	var ob, eb bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _, _ = io.Copy(&ob, outR) }()
	go func() { defer wg.Done(); _, _ = io.Copy(&eb, errR) }()

	func() {
		defer func() {
			os.Stdout, os.Stderr = oldOut, oldErr
			outW.Close()
			errW.Close()
		}()
		fn()
	}()
	wg.Wait()
	outR.Close()
	errR.Close()
	return ob.String(), eb.String()
}

// ── 가짜 관리자 연결 ─────────────────────────────────────────────────────

// fakeAdmin 은 adminDB 셋을 그대로 흉내 낸다.
//
// 무엇이 있는가를 필드로 정하고, 무엇을 실행했는가를 sql 에 남긴다. 두 번째가
// 이 파일의 가장 강한 단언이다 - "--check 는 아무것도 안 바꾼다" 는 종료
// 코드로는 안 보이고 실행된 SQL 의 목록으로만 보인다.
type fakeAdmin struct {
	haveRole, haveDB bool
	roleErr, dbErr   error  // exists() 안의 Scan 이 돌려줄 오류
	execErr          error  // Exec 이 돌려줄 오류
	execFailOn       string // 그 접두사로 시작하는 SQL 에서만 execErr 를 낸다
	sql              []string
	closed           int
}

var _ adminDB = (*fakeAdmin)(nil)

func (f *fakeAdmin) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	f.sql = append(f.sql, sql)
	if f.execErr != nil && strings.HasPrefix(sql, f.execFailOn) {
		return pgconn.CommandTag{}, f.execErr
	}
	return pgconn.CommandTag{}, nil
}

func (f *fakeAdmin) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	switch {
	case strings.Contains(sql, "pg_roles"):
		return fakeRow{found: f.haveRole, err: f.roleErr}
	case strings.Contains(sql, "pg_database"):
		return fakeRow{found: f.haveDB, err: f.dbErr}
	}
	return fakeRow{err: fmt.Errorf("the setup asked something this fake does not know: %s", sql)}
}

func (f *fakeAdmin) Close(_ context.Context) error { f.closed++; return nil }

// fakeRow 는 exists() 가 읽는 한 행이다.
//
// 없음을 pgx.ErrNoRows 로 내는 것이 계약이다 - exists() 가 그것만 "없다" 로
// 읽고 나머지 오류는 전부 위로 올린다.
type fakeRow struct {
	found bool
	err   error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if !r.found {
		return pgx.ErrNoRows
	}
	if len(dest) == 1 {
		if p, ok := dest[0].(*int); ok {
			*p = 1
		}
	}
	return nil
}

// refusingReader 는 읽히면 그 자체가 실패다.
//
// --yes 와 --check 가 stdin 을 안 읽는다는 것은 반환값에도 출력에도 안
// 나타난다. 읽는 쪽에 덫을 놓는 것이 그것을 보는 유일한 방법이다.
type refusingReader struct{ t *testing.T }

func (r refusingReader) Read([]byte) (int, error) {
	r.t.Errorf("stdin was read although this run is not interactive")
	return 0, io.EOF
}

// setupDeps 는 runSetupWith 에 넘길 주입이자, 그것이 무엇을 조립했는지를
// 되읽는 창이다 - adminURL 과 appURL 이 그것이다.
type setupDeps struct {
	in         io.Reader
	admin      *fakeAdmin
	connectErr error
	openErr    error
	st         *store.Store

	adminURL, appURL string
	connects         int
}

func (d *setupDeps) io() setupIO {
	return setupIO{
		in: d.in,
		connect: func(_ context.Context, url string) (adminDB, error) {
			d.adminURL, d.connects = url, d.connects+1
			if d.connectErr != nil {
				return nil, d.connectErr
			}
			return d.admin, nil
		},
		open: func(_ context.Context, url string) (*store.Store, error) {
			d.appURL = url
			if d.openErr != nil {
				return nil, d.openErr
			}
			return d.st, nil
		},
	}
}

// scratchStore 는 그 판에 붙은 Store 하나다. runSetupWith 가 defer 로 닫으므로
// 부를 때마다 새로 연다.
func scratchStore(t *testing.T, url string) *store.Store {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	st, err := store.Open(ctx, url)
	if err != nil {
		t.Fatalf("cannot open the scratch database: %v", err)
	}
	return st
}

func wantIn(t *testing.T, what, got string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Fatalf("%s said %q, want %q in it", what, got, w)
		}
	}
}

// ── 아무것도 안 바꾸는 구간 ──────────────────────────────────────────────

func TestSetup_CheckReportsWhatIsMissingAndRunsNoStatement(t *testing.T) {
	// --check 의 계약은 "보고만 한다" 이고, 그것은 종료 코드로 안 보인다.
	// 실행된 SQL 이 0 개라는 것이 그 계약의 유일한 관측이다.
	cfg := filepath.Join(t.TempDir(), "config.yaml")
	d := &setupDeps{in: refusingReader{t}, admin: &fakeAdmin{}}

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = runSetupWith([]string{"--check", "--config", cfg,
			"--host", "db1", "--port", "6543"}, d.io())
	})

	if code != 0 {
		t.Fatalf("exit code contract: setup --check = %d, want 0; stderr %q", code, stderr)
	}
	wantIn(t, "setup --check", stdout,
		"postgres  db1:6543",
		"  role      enode   missing",
		"  database  enode   missing",
		"config    "+cfg+"   missing",
		"schema    applied on every run; it is idempotent")
	if len(d.admin.sql) != 0 {
		t.Fatalf("--check contract: it ran %v, want nothing changed", d.admin.sql)
	}
	if d.admin.closed != 1 {
		t.Fatalf("the admin connection was closed %d times, want exactly 1", d.admin.closed)
	}
	if want := "postgres://postgres@db1:6543/postgres?sslmode=disable"; d.adminURL != want {
		t.Fatalf("the admin URL it composed is %q, want %q", d.adminURL, want)
	}
	if stderr != "" {
		t.Fatalf("setup --check wrote %q to stderr, want nothing", stderr)
	}
}

func TestSetup_CheckSaysExistsForWhatIsAlreadyThere(t *testing.T) {
	// mark() 의 반대편. 셋 다 있는 판에서 셋 다 "exists" 여야 한다 - 하나라도
	// 반대로 읽히면 setup 이 있는 것을 다시 만들려 든다.
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("listen: 127.0.0.1:8080\n"), 0o600); err != nil {
		t.Fatalf("write %s: %v", cfg, err)
	}
	d := &setupDeps{in: refusingReader{t}, admin: &fakeAdmin{haveRole: true, haveDB: true}}

	var code int
	stdout, _ := captureOutput(t, func() {
		code = runSetupWith([]string{"--check", "--config", cfg}, d.io())
	})

	if code != 0 {
		t.Fatalf("exit code contract: setup --check on a ready machine = %d, want 0", code)
	}
	wantIn(t, "setup --check on a ready machine", stdout,
		"  role      enode   exists",
		"  database  enode   exists",
		"config    "+cfg+"   exists")
}

func TestSetup_PostgresUnreachable_ExitsOneAndSaysNothingChanged(t *testing.T) {
	// 못 붙었을 때 "nothing was changed" 를 말하는 것이 계약이다. 안 말하면
	// 사람이 반쯤 세워진 판을 의심하며 손으로 뒤진다.
	d := &setupDeps{in: refusingReader{t}, connectErr: errors.New("connection refused")}

	var code int
	_, stderr := captureOutput(t, func() {
		code = runSetupWith([]string{"--check"}, d.io())
	})

	if code != 1 {
		t.Fatalf("exit code contract: an unreachable postgres = %d, want 1; stderr %q", code, stderr)
	}
	wantIn(t, "setup against an unreachable postgres", stderr,
		"cannot reach postgres: connection refused",
		"nothing was changed.")
}

func TestSetup_CatalogQueryFails_ExitsOneAndNamesTheCatalog(t *testing.T) {
	// exists() 는 pgx.ErrNoRows 만 "없다" 로 읽는다. 다른 오류를 없음으로
	// 읽으면 setup 이 이미 있는 롤을 만들려다 다른 메시지로 죽는다.
	for _, tc := range []struct {
		name  string
		admin *fakeAdmin
		want  string
	}{
		{"pg_roles", &fakeAdmin{roleErr: errors.New("permission denied")}, "cannot read pg_roles: permission denied"},
		{"pg_database", &fakeAdmin{dbErr: errors.New("permission denied")}, "cannot read pg_database: permission denied"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &setupDeps{in: refusingReader{t}, admin: tc.admin}
			var code int
			_, stderr := captureOutput(t, func() {
				code = runSetupWith([]string{"--check", "--config",
					filepath.Join(t.TempDir(), "config.yaml")}, d.io())
			})
			if code != 1 {
				t.Fatalf("exit code contract: an unreadable %s = %d, want 1", tc.name, code)
			}
			wantIn(t, "setup with an unreadable "+tc.name, stderr, tc.want)
			if len(d.admin.sql) != 0 {
				t.Fatalf("it ran %v after failing to read the catalog, want nothing", d.admin.sql)
			}
		})
	}
}

func TestSetup_WithoutTheConfigFlagItNamesTheFirstSearchPath(t *testing.T) {
	// --config 를 안 주면 setup 은 찾아본 자리 중 첫째를 쓰겠다고 말한다.
	// 그 자리를 안 말하면 사람이 setup 이 어디에 쓸지 모르는 채로 y 를 친다.
	noSystemConfig(t)
	e := isolate(t)
	t.Setenv("HOME", e.home)
	t.Setenv("ENODE_MEDIATOR_CONFIG", "")
	d := &setupDeps{in: refusingReader{t}, admin: &fakeAdmin{}}

	var code int
	stdout, _ := captureOutput(t, func() { code = runSetupWith([]string{"--check"}, d.io()) })

	if code != 0 {
		t.Fatalf("exit code contract: setup --check without --config = %d, want 0", code)
	}
	wantIn(t, "setup --check without --config", stdout, "config    "+e.userConfigPath()+"   missing")
}

// ── 대화형 ───────────────────────────────────────────────────────────────

func TestSetup_InteractiveDeclineChangesNothing(t *testing.T) {
	// 물어보고 아니라고 하면 아무것도 안 바꾼다. 프롬프트는 일곱이고
	// (host · port · db · user · password · admin role · admin password)
	// 여덟째가 확인이다 - --password 를 안 주었을 때의 개수다.
	for _, tc := range []struct{ name, in string }{
		{"an explicit no", "\n\n\n\n\n\n\nn\n"},
		{"an empty line", "\n\n\n\n\n\n\n\n"},
		{"stdin ends early", "\n\n\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &setupDeps{in: strings.NewReader(tc.in), admin: &fakeAdmin{}}
			var code int
			stdout, _ := captureOutput(t, func() {
				code = runSetupWith([]string{"--config",
					filepath.Join(t.TempDir(), "config.yaml")}, d.io())
			})
			if code != 0 {
				t.Fatalf("exit code contract: declining the confirmation = %d, want 0", code)
			}
			wantIn(t, "declining the confirmation", stdout,
				"mediator setup — answer or press enter to keep the default in brackets.",
				"postgres host [127.0.0.1]: ",
				"admin password: ",
				"create what is missing and write the config? [y/N]: ",
				"nothing was changed.")
			if len(d.admin.sql) != 0 {
				t.Fatalf("declining ran %v, want nothing", d.admin.sql)
			}
		})
	}
}

func TestSetup_InteractiveAnswersBecomeBothURLs(t *testing.T) {
	// 사람이 친 값이 어디로 가는지가 이 갈래의 전부다. 화면의 표만 보면
	// 조립된 URL 이 안 보이고, URL 이 틀리면 setup 은 성공한 것처럼 끝난 뒤
	// mediator 가 다른 판에 붙는다.
	for _, tc := range []struct{ name, yes string }{
		{"y", "y"},
		{"yes", "yes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &setupDeps{
				in:      strings.NewReader("db2\n5433\nmydb\nmyuser\nsecret\nadmin1\nadminpw\n" + tc.yes + "\n"),
				admin:   &fakeAdmin{haveRole: true, haveDB: true},
				openErr: errors.New("no route to host"),
			}
			var code int
			stdout, stderr := captureOutput(t, func() {
				code = runSetupWith([]string{"--config",
					filepath.Join(t.TempDir(), "config.yaml")}, d.io())
			})
			// 확인까지 통과했으므로 바꾸는 구간에 들어갔고, 앱 롤로 붙는
			// 자리에서 멈춘다. 그 1 이 여기서 보는 것은 "대화형 답이
			// 종료 코드를 삼키지 않는다" 다.
			if code != 1 {
				t.Fatalf("exit code contract: a refused app connection = %d, want 1; stderr %q", code, stderr)
			}
			wantIn(t, "the interactive summary", stdout, "postgres  db2:5433")
			wantIn(t, "a refused app connection", stderr, "cannot connect as myuser: no route to host")
			if want := "postgres://admin1:adminpw@db2:5433/postgres?sslmode=disable"; d.adminURL != want {
				t.Fatalf("the admin URL it composed is %q, want %q", d.adminURL, want)
			}
			if want := "postgres://myuser:secret@db2:5433/mydb?sslmode=disable"; d.appURL != want {
				t.Fatalf("the app URL it composed is %q, want %q", d.appURL, want)
			}
		})
	}
}

func TestSetup_PasswordFlagRemovesThatOnePrompt(t *testing.T) {
	// --password 를 주면 프롬프트가 여섯이고 일곱째가 확인이다. 개수가
	// 어긋나면 답이 한 칸씩 밀려 admin 비밀번호가 롤 이름이 된다.
	d := &setupDeps{
		in:      strings.NewReader("\n\n\n\n\n\ny\n"),
		admin:   &fakeAdmin{haveRole: true, haveDB: true},
		openErr: errors.New("no route to host"),
	}
	var code int
	stdout, _ := captureOutput(t, func() {
		code = runSetupWith([]string{"--password", "given", "--config",
			filepath.Join(t.TempDir(), "config.yaml")}, d.io())
	})
	if code != 1 {
		t.Fatalf("exit code contract: a refused app connection = %d, want 1", code)
	}
	if strings.Contains(stdout, "password for that role") {
		t.Fatalf("--password was given yet setup still asked for it: %q", stdout)
	}
	if want := "postgres://enode:given@127.0.0.1:5432/enode?sslmode=disable"; d.appURL != want {
		t.Fatalf("the app URL it composed is %q, want %q", d.appURL, want)
	}
}

// ── 바꾸는 구간 ──────────────────────────────────────────────────────────

func TestSetup_CreatesTheRoleAndTheDatabaseWhenBothAreMissing(t *testing.T) {
	// CREATE ROLE 과 CREATE DATABASE 는 바인딩 파라미터를 못 받아 문자열로
	// 만들어진다 (setup.go 의 quoteIdent 주석). 그래서 무엇을 보냈는지를
	// 글자 그대로 건다 - 인용이 깨지는 것은 여기서만 보인다.
	for _, tc := range []struct {
		name string
		args []string
		want []string
	}{
		{
			"with a password",
			[]string{"--password", "p@ss"},
			[]string{`CREATE ROLE "enode" LOGIN PASSWORD 'p@ss'`, `CREATE DATABASE "enode" OWNER "enode"`},
		},
		{
			"without a password",
			nil,
			[]string{`CREATE ROLE "enode" LOGIN`, `CREATE DATABASE "enode" OWNER "enode"`},
		},
		{
			"with quotes in the names",
			[]string{"--user", `we"ird`, "--db", `da"ta`, "--password", "it's"},
			[]string{`CREATE ROLE "we""ird" LOGIN PASSWORD 'it''s'`, `CREATE DATABASE "da""ta" OWNER "we""ird"`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &setupDeps{in: refusingReader{t}, admin: &fakeAdmin{},
				openErr: errors.New("no route to host")}
			args := append([]string{"--yes", "--config",
				filepath.Join(t.TempDir(), "config.yaml")}, tc.args...)

			var code int
			stdout, _ := captureOutput(t, func() { code = runSetupWith(args, d.io()) })

			if code != 1 {
				t.Fatalf("exit code contract: a refused app connection = %d, want 1", code)
			}
			if len(d.admin.sql) != len(tc.want) {
				t.Fatalf("it ran %v, want %v", d.admin.sql, tc.want)
			}
			for i, want := range tc.want {
				if d.admin.sql[i] != want {
					t.Fatalf("statement %d was %q, want %q", i, d.admin.sql[i], want)
				}
			}
			wantIn(t, "creating what is missing", stdout, "created role ", "created database ")
		})
	}
}

func TestSetup_CreationFailure_ExitsOneAndStopsThere(t *testing.T) {
	// 롤을 못 만들면 데이터베이스를 만들려 들지 않는다. 진행하면 소유자가
	// 없는 데이터베이스가 남고, 그 판은 setup 을 다시 돌려도 안 낫는다.
	for _, tc := range []struct {
		name, failOn, want string
		wantRun            int
	}{
		{"the role", "CREATE ROLE", "cannot create role enode: insufficient privilege", 1},
		{"the database", "CREATE DATABASE", "cannot create database enode: insufficient privilege", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &setupDeps{in: refusingReader{t}, admin: &fakeAdmin{
				execErr: errors.New("insufficient privilege"), execFailOn: tc.failOn}}
			var code int
			_, stderr := captureOutput(t, func() {
				code = runSetupWith([]string{"--yes", "--config",
					filepath.Join(t.TempDir(), "config.yaml")}, d.io())
			})
			if code != 1 {
				t.Fatalf("exit code contract: a refused %s = %d, want 1", tc.name, code)
			}
			wantIn(t, "a refused "+tc.name, stderr, tc.want)
			if len(d.admin.sql) != tc.wantRun {
				t.Fatalf("it ran %v, want it to stop after %d statement(s)", d.admin.sql, tc.wantRun)
			}
			if d.appURL != "" {
				t.Fatalf("it went on to open %q although the creation failed", d.appURL)
			}
		})
	}
}

// ── 스키마와 설정 파일 ───────────────────────────────────────────────────

func TestSetup_AppliesTheSchemaAndWritesTheConfig(t *testing.T) {
	// 끝까지 가는 유일한 갈래다. 화면에 나온 것과 파일에 남은 것을 둘 다
	// 본다 - 토큰은 화면에만 있고 파일에 안 남으면 다음 기동이 다른 토큰을
	// 만들어 노드가 전부 401 로 거절된다.
	url := scratchDB(t)
	cfg := filepath.Join(t.TempDir(), "config.yaml")
	d := &setupDeps{in: refusingReader{t},
		admin: &fakeAdmin{haveRole: true, haveDB: true}, st: scratchStore(t, url)}

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = runSetupWith([]string{"--yes", "--config", cfg}, d.io())
	})

	if code != 0 {
		t.Fatalf("exit code contract: a complete setup = %d, want 0; stderr %q", code, stderr)
	}
	wantIn(t, "a complete setup", stdout,
		"schema applied",
		"wrote "+cfg,
		"token    ",
		"different token is rejected with 401",
		"start it with:  mediator --config "+cfg)
	if len(d.admin.sql) != 0 {
		t.Fatalf("it ran %v although the role and the database were already there", d.admin.sql)
	}

	written, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("read back %s: %v", cfg, err)
	}
	wantIn(t, "the config it wrote", string(written), `url: "`+d.appURL+`"`, "token: ", "artifacts:")

	// 화면의 토큰과 파일의 토큰이 같아야 한다. 다르면 사람이 화면을 보고
	// 노드에 넣은 값이 mediator 가 쓰는 값과 어긋난다.
	tok := strings.TrimSpace(strings.SplitN(strings.SplitN(stdout, "token    ", 2)[1], "\n", 2)[0])
	if tok == "" {
		t.Fatalf("no token on the screen: %q", stdout)
	}
	if !strings.Contains(string(written), "token: "+tok) {
		t.Fatalf("the screen said token %q but %s holds %q", tok, cfg, string(written))
	}
}

func TestSetup_KeepsTheTokenTheConfigAlreadyHolds(t *testing.T) {
	// 있는 토큰을 갈아치우면 이미 그 값으로 도는 노드 전부가 401 이 된다.
	// 그래서 "kept the one already in the config" 가 계약이다.
	url := scratchDB(t)
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("listen: 127.0.0.1:8080\ntoken: already-here\n"), 0o600); err != nil {
		t.Fatalf("write %s: %v", cfg, err)
	}
	d := &setupDeps{in: refusingReader{t},
		admin: &fakeAdmin{haveRole: true, haveDB: true}, st: scratchStore(t, url)}

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = runSetupWith([]string{"--yes", "--config", cfg}, d.io())
	})

	if code != 0 {
		t.Fatalf("exit code contract: setup over an existing config = %d, want 0; stderr %q", code, stderr)
	}
	wantIn(t, "setup over an existing config", stdout, "token    kept the one already in the config")
	if strings.Contains(stdout, "wrote "+cfg) {
		t.Fatalf("it claimed to have written a config that was already there: %q", stdout)
	}
	written, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("read back %s: %v", cfg, err)
	}
	wantIn(t, "the config it kept", string(written), "token: already-here")
}

func TestSetup_SchemaCannotBeApplied_ExitsOne(t *testing.T) {
	// 스키마를 못 올리면 설정 파일을 쓰지 않는다. 쓰면 사람은 setup 이
	// 끝났다고 믿고 mediator 를 띄우며, 그때 나오는 오류는 여기와 무관한
	// 자리를 가리킨다.
	//
	// 닫힌 풀로 몬다 - 진짜 판을 열고 바로 닫으면 Migrate 의 Exec 가
	// 결정적으로 실패한다. 판을 지우는 안보다 이쪽이 경합이 없다.
	url := scratchDB(t)
	cfg := filepath.Join(t.TempDir(), "config.yaml")
	st := scratchStore(t, url)
	st.Close()
	d := &setupDeps{in: refusingReader{t},
		admin: &fakeAdmin{haveRole: true, haveDB: true}, st: st}

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = runSetupWith([]string{"--yes", "--config", cfg}, d.io())
	})

	if code != 1 {
		t.Fatalf("exit code contract: a schema that cannot be applied = %d, want 1", code)
	}
	wantIn(t, "a schema that cannot be applied", stderr, "cannot apply schema")
	if strings.Contains(stdout, "schema applied") {
		t.Fatalf("it announced a schema it did not apply: %q", stdout)
	}
	if _, err := os.Stat(cfg); err == nil {
		t.Fatalf("%s was written although the schema never went up", cfg)
	}
}

func TestSetup_ConfigCannotBeWritten_ExitsOneAndSaysWhy(t *testing.T) {
	// 못 쓰는 이유가 권한이면 그 자리에서 sudo 를 가리킨다. 안 가리키면
	// 사람이 --config 를 의심하며 경로를 바꿔 본다.
	url := scratchDB(t)

	t.Run("a path under a regular file", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "blocker")
		if err := os.WriteFile(blocker, []byte("not a directory\n"), 0o600); err != nil {
			t.Fatalf("write %s: %v", blocker, err)
		}
		cfg := filepath.Join(blocker, "config.yaml")
		d := &setupDeps{in: refusingReader{t},
			admin: &fakeAdmin{haveRole: true, haveDB: true}, st: scratchStore(t, url)}
		var code int
		_, stderr := captureOutput(t, func() {
			code = runSetupWith([]string{"--yes", "--config", cfg}, d.io())
		})
		if code != 1 {
			t.Fatalf("exit code contract: a config path under a file = %d, want 1", code)
		}
		wantIn(t, "a config path under a file", stderr, "cannot write "+cfg)
		if strings.Contains(stderr, "try again with sudo") {
			t.Fatalf("it offered sudo for a failure that is not about permission: %q", stderr)
		}
	})

	t.Run("a directory this user cannot write", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "readonly")
		if err := os.Mkdir(dir, 0o500); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		// 되돌리는 정리를 먼저 등록한다 - t.Cleanup 은 LIFO 라 TempDir 이
		// 등록한 삭제보다 나중에 등록한 이것이 먼저 돈다
		// (internal/enode/claude_test.go 가 같은 함정을 먼저 밟았다).
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
		if os.Geteuid() == 0 {
			t.Fatalf("this test runs as root, and root writes into a 0500 directory; " +
				"the permission branch cannot be measured here")
		}
		cfg := filepath.Join(dir, "config.yaml")
		d := &setupDeps{in: refusingReader{t},
			admin: &fakeAdmin{haveRole: true, haveDB: true}, st: scratchStore(t, url)}
		var code int
		_, stderr := captureOutput(t, func() {
			code = runSetupWith([]string{"--yes", "--config", cfg}, d.io())
		})
		if code != 1 {
			t.Fatalf("exit code contract: a config in an unwritable directory = %d, want 1", code)
		}
		wantIn(t, "a config in an unwritable directory", stderr,
			"cannot write "+cfg,
			"try again with sudo, or pass --config with a path you can write.")
	})
}

func TestSetup_DatabaseURLCannotBeStored_ExitsOne(t *testing.T) {
	// 디렉터리를 --config 로 주면 os.Stat 은 성공하므로 setup 은 "설정이
	// 이미 있다" 로 읽고 만들기를 건너뛴다. 그다음 database.url 을 넣는
	// 자리에서 처음으로 실패한다 - 그 갈래가 여기다.
	url := scratchDB(t)
	dir := t.TempDir()
	d := &setupDeps{in: refusingReader{t},
		admin: &fakeAdmin{haveRole: true, haveDB: true}, st: scratchStore(t, url)}

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = runSetupWith([]string{"--yes", "--config", dir}, d.io())
	})

	if code != 1 {
		t.Fatalf("exit code contract: a directory as the config path = %d, want 1", code)
	}
	wantIn(t, "a directory as the config path", stdout, "config    "+dir+"   exists")
	wantIn(t, "a directory as the config path", stderr, "cannot set database.url:")
}

// ── 주입의 기본값 ────────────────────────────────────────────────────────

func TestRunSetup_UsesTheLivePostgresConnector(t *testing.T) {
	// runSetup 은 liveSetupIO() 를 넘기는 감싸개다 (setup.go). 기본값이
	// 오늘 동작과 같아야 main 쪽 호출이 안 바뀌므로, 진짜 pgx 로 나가는
	// 것을 한 번은 건다 - 아무도 안 듣는 포트로 보내면 그 자리에서 돌아온다.
	var code int
	_, stderr := captureOutput(t, func() {
		code = runSetup([]string{"--check", "--admin-url",
			"postgres://nobody:nobody@127.0.0.1:1/postgres?sslmode=disable"})
	})
	if code != 1 {
		t.Fatalf("exit code contract: runSetup against a dead port = %d, want 1; stderr %q", code, stderr)
	}
	wantIn(t, "runSetup against a dead port", stderr, "cannot reach postgres:", "nothing was changed.")
}
