package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/taeels/enode/internal/config"
	"github.com/taeels/enode/internal/store"
)

// setup 은 Postgres 를 세우고 설정 파일을 채운다.
//
// 왜 별도 바이너리가 아니라 하위명령인가 — 스키마가 이 바이너리에 embed 돼
// 있다. 갈라 놓으면 스키마가 두 곳에 살거나(버전을 두 곳에 적지 않는다),
// 세울 도구와 돌 도구의 판이 갈린다. ADR-056 이 `--version` 을 넣은 이유가
// 「도구가 자기를 설명해야 한다」였는데, 판이 둘이면 그 값이 사라진다.
// 그리고 ADR-056 §2 가 이미 같은 모양을 거절했다 — 전용 도구를 만들면
// 그것이 또 배포돼야 한다.
//
// 권한은 바이너리가 아니라 건네는 자격이 정한다. 여기서 묻는 관리자 자격은
// 이 실행 동안만 살고 파일에 안 남는다. 돌 때의 mediator 는 config 의
// database.url 로만 붙는다.
func runSetup(args []string) int {
	fs := flag.NewFlagSet("mediator setup", flag.ExitOnError)
	cfgPath := fs.String("config", "", "config file to write (default: first of the search paths)")
	adminURL := fs.String("admin-url", "", "postgres URL with rights to create roles and databases")
	appURL := fs.String("url", "", "database URL the mediator will use")
	dbName := fs.String("db", "enode", "database to create")
	dbUser := fs.String("user", "enode", "role the mediator connects as")
	dbPass := fs.String("password", "", "password for that role")
	host := fs.String("host", "127.0.0.1", "postgres host")
	port := fs.String("port", "5432", "postgres port")
	check := fs.Bool("check", false, "report what is missing and change nothing")
	yes := fs.Bool("yes", false, "do not ask; assume yes")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `mediator setup — prepare postgres and write the config file

  mediator setup                      ask, then create what is missing
  mediator setup --check              report only; change nothing
  mediator setup --yes --host db1 --password s3cret

It creates the role and the database if they are absent, applies the schema,
and writes the config file. The admin credentials are used for this run only
and are never stored.

`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	in := bufio.NewReader(os.Stdin)
	interactive := !*yes && !*check

	if interactive {
		fmt.Println("mediator setup — answer or press enter to keep the default in brackets.")
		fmt.Println()
		*host = ask(in, "postgres host", *host)
		*port = ask(in, "postgres port", *port)
		*dbName = ask(in, "database to create", *dbName)
		*dbUser = ask(in, "role the mediator connects as", *dbUser)
		if *dbPass == "" {
			*dbPass = ask(in, "password for that role", "")
		}
		fmt.Println()
		fmt.Println("The next credentials only need to exist for this run.")
		adminUser := ask(in, "admin role (may create databases)", "postgres")
		adminPass := ask(in, "admin password", "")
		if *adminURL == "" {
			*adminURL = pgURL(adminUser, adminPass, *host, *port, "postgres")
		}
	}
	if *adminURL == "" {
		*adminURL = pgURL("postgres", "", *host, *port, "postgres")
	}
	if *appURL == "" {
		*appURL = pgURL(*dbUser, *dbPass, *host, *port, *dbName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// ① 아무것도 안 바꾸는 구간 — 붙어 보고, 무엇이 없는지만 본다.
	admin, err := pgx.Connect(ctx, *adminURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot reach postgres: %v\n", err)
		fmt.Fprintf(os.Stderr, "nothing was changed.\n")
		return 1
	}
	defer admin.Close(ctx) //nolint:errcheck

	haveRole, err := exists(ctx, admin, "SELECT 1 FROM pg_roles WHERE rolname=$1", *dbUser)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot read pg_roles: %v\n", err)
		return 1
	}
	haveDB, err := exists(ctx, admin, "SELECT 1 FROM pg_database WHERE datname=$1", *dbName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot read pg_database: %v\n", err)
		return 1
	}

	path := *cfgPath
	tried := []string{}
	if path == "" {
		p, t, _ := config.Resolve("")
		path, tried = p, t
		if path == "" && len(tried) > 0 {
			path = tried[0]
		}
	}
	_, statErr := os.Stat(path)
	haveCfg := statErr == nil

	fmt.Println()
	fmt.Println("postgres  " + *host + ":" + *port)
	fmt.Println("  role      " + *dbUser + mark(haveRole))
	fmt.Println("  database  " + *dbName + mark(haveDB))
	fmt.Println("  schema    applied on every run; it is idempotent")
	fmt.Println("config    " + path + mark(haveCfg))
	fmt.Println()

	if *check {
		return 0
	}
	if interactive && !confirm(in, "create what is missing and write the config?") {
		fmt.Println("nothing was changed.")
		return 0
	}

	// ② 여기서부터 바꾼다.
	if !haveRole {
		q := "CREATE ROLE " + quoteIdent(*dbUser) + " LOGIN"
		if *dbPass != "" {
			q += " PASSWORD " + quoteLiteral(*dbPass)
		}
		if _, err := admin.Exec(ctx, q); err != nil {
			fmt.Fprintf(os.Stderr, "cannot create role %s: %v\n", *dbUser, err)
			return 1
		}
		fmt.Println("created role " + *dbUser)
	}
	if !haveDB {
		// CREATE DATABASE 는 트랜잭션 안에서 못 돈다. 그래서 따로 보낸다.
		q := "CREATE DATABASE " + quoteIdent(*dbName) + " OWNER " + quoteIdent(*dbUser)
		if _, err := admin.Exec(ctx, q); err != nil {
			fmt.Fprintf(os.Stderr, "cannot create database %s: %v\n", *dbName, err)
			return 1
		}
		fmt.Println("created database " + *dbName)
	}

	// ③ 앱 롤로 붙어 스키마를 올린다.
	//
	// 판을 세지 않는다 — schema.sql 은 전부 IF NOT EXISTS 라 매번 다 돌려도
	// 안전하다. 판 개념은 되돌릴 수 없는 변경이 처음 생기는 날에 들어온다.
	st, err := store.Open(ctx, *appURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot connect as %s: %v\n", *dbUser, err)
		return 1
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "cannot apply schema: %v\n", err)
		return 1
	}
	fmt.Println("schema applied")

	// ④ 설정 파일.
	if !haveCfg {
		if err := os.WriteFile(path, []byte(config.Sample()), 0o600); err != nil {
			// 디렉터리가 없을 수 있다 — writeSecret 이 만들어 준다.
			if err := config.SetArtifactsRoot(path, config.Default().Artifacts.Root); err != nil {
				fmt.Fprintf(os.Stderr, "cannot write %s: %v\n", path, err)
				return 1
			}
			if err := os.WriteFile(path, []byte(config.Sample()), 0o600); err != nil {
				fmt.Fprintf(os.Stderr, "cannot write %s: %v\n", path, err)
				return 1
			}
		}
		fmt.Println("wrote " + path)
	}
	if err := config.SetDatabaseURL(path, *appURL); err != nil {
		fmt.Fprintf(os.Stderr, "cannot set database.url: %v\n", err)
		return 1
	}
	if err := config.SetArtifactsRoot(path, config.Default().Artifacts.Root); err != nil {
		fmt.Fprintf(os.Stderr, "cannot set artifacts.root: %v\n", err)
		return 1
	}
	tok, err := config.EnsureToken(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot set token: %v\n", err)
		return 1
	}

	fmt.Println()
	if tok != "" {
		fmt.Println("token    " + tok)
		fmt.Println("         Every node needs this exact value; a node with a")
		fmt.Println("         different token is rejected with 401.")
	} else {
		fmt.Println("token    kept the one already in the config")
	}
	fmt.Println()
	fmt.Println("start it with:  mediator --config " + path)
	return 0
}

func ask(in *bufio.Reader, prompt, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", prompt, def)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	line, err := in.ReadString('\n')
	if err != nil && line == "" {
		return def
	}
	if v := strings.TrimSpace(line); v != "" {
		return v
	}
	return def
}

func confirm(in *bufio.Reader, prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	line, _ := in.ReadString('\n')
	v := strings.ToLower(strings.TrimSpace(line))
	return v == "y" || v == "yes"
}

func mark(have bool) string {
	if have {
		return "   exists"
	}
	return "   missing"
}

func exists(ctx context.Context, c *pgx.Conn, q string, arg any) (bool, error) {
	var one int
	err := c.QueryRow(ctx, q, arg).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func pgURL(user, pass, host, port, db string) string {
	u := url.URL{Scheme: "postgres", Host: host + ":" + port, Path: "/" + db}
	if pass != "" {
		u.User = url.UserPassword(user, pass)
	} else {
		u.User = url.User(user)
	}
	u.RawQuery = "sslmode=disable"
	return u.String()
}

// quoteIdent 와 quoteLiteral — CREATE ROLE 과 CREATE DATABASE 는 이름을
// 바인딩 파라미터로 못 받는다. 그래서 여기서만 문자열로 만들고, 그래서
// 인용을 직접 한다.
func quoteIdent(s string) string   { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }
func quoteLiteral(s string) string { return `'` + strings.ReplaceAll(s, `'`, `''`) + `'` }
