# 시험용 Postgres — Docker 없이 각자 기계에 깐다

`go test ./...` 이 진짜 Postgres 를 쓴다. `scripts/testdb.sh` 는 그것을 Docker 로
띄우지만 **대회장에서는 Docker 설치가 병목**이라, 각자 기계에 직접 깐다.

WSL2 는 3절, macOS 는 4절이다. 값과 확인은 공통이다.

---

## 1. 왜 각자 깔고, 왜 하나를 같이 안 쓰나

한 대에 세워 넷이 붙는 길이 있는데 **그 길은 막혀 있다.**

`internal/api` 의 시험 107개가 `ENODE_TEST_DATABASE_URL` 이 가리키는 판을 직접
쓰고, 시험마다 `TRUNCATE steps, leases, runs, nodes` 를 친다. 두 사람이 같은
판을 보면 한쪽의 TRUNCATE 가 다른 쪽의 픽스처를 지운다.

**이 저장소가 이미 밟았다.** `internal/store/reap_deterministic_test.go` 머리말이
그 기록이다 — 두 패키지가 같은 판을 쓰다 3회 중 2회가 깨졌고, `deadlock detected
(SQLSTATE 40P01)` 과 외래키 위반으로 났다. 「서로의 결과를 바꾸는 모양이고,
그것은 아무도 재현하지 못하는 실패다」. `scripts/testdb.sh` 머리말도 같은 종류의
사고를 하나 더 적어 두었다.

증상이 남의 커밋으로 보이는 실패다. 대회장에서 가장 비싸다.

**그리고 DB 없이 도는 선택지가 없다.** `internal/store` 의 시험은 URL 이 없으면
`t.Skip` 이 아니라 **`t.Fatal`** 이고, `.ci-allowed-skips` 의 실제 항목은 0 이다.

**부득이 한 서버를 나눠 써야 하면 사람마다 판을 따로 판다** — `enode_test_taeels`
처럼. 서버가 하나인 것은 괜찮고 **판이 하나인 것이 안 된다.**

---

## 2. 값 — `scripts/testdb.sh` 의 규약 그대로

```text
   PostgreSQL   17          RETURNING OLD.* 를 안 쓰고 CTE 로 도는 것이 이 판 기준이다
   역할          enode
   비밀번호      enode
   판            enode_test
   포트          55434       5432 를 피한다 — 이미 깔린 Postgres 와 안 부딪치게
```

**정본은 DSN 한 줄이다.** 네 기계에서 글자까지 같아야 한다.

```text
   postgres://enode:enode@127.0.0.1:55434/enode_test?sslmode=disable
```

**권한은 `CREATEDB` 면 된다. 슈퍼유저가 아니어도 된다** — 실측으로 확인했다.
시험이 `CREATE DATABASE` 를 치는 것은 `scratchDB` 때문이다: 패키지 바이너리가
병렬로 돌아 서로를 밟지 않게 판을 따로 파고, 끝나면 지운다.

---

## 3. WSL2 (Ubuntu)

### 3.1 설치

우분투 기본 저장소는 17 이 아닐 수 있으므로 PGDG 를 붙인다.

```bash
sudo apt update && sudo apt install -y curl ca-certificates
sudo install -d /usr/share/postgresql-common/pgdg
sudo curl -fsSL -o /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc \
  https://www.postgresql.org/media/keys/ACCC4CF8.asc
echo "deb [signed-by=/usr/share/postgresql-common/pgdg/apt.postgresql.org.asc] https://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" \
  | sudo tee /etc/apt/sources.list.d/pgdg.list >/dev/null
sudo apt update && sudo apt install -y postgresql-17
```

### 3.2 포트를 55434 로 옮기고 다시 띄운다

패키지가 클러스터 `main` 을 5432 로 만들어 두므로 고친 뒤 재시작한다.

```bash
sudo sed -i 's/^port = .*/port = 55434/' /etc/postgresql/17/main/postgresql.conf
sudo service postgresql restart
```

### 3.3 역할과 판

```bash
sudo -u postgres psql -p 55434 -c "CREATE ROLE enode LOGIN PASSWORD 'enode' CREATEDB;"
sudo -u postgres createdb -p 55434 -O enode enode_test
```

### 3.4 WSL2 는 재시작하면 서비스가 안 올라온다

`systemd` 가 기본이 아니라서 WSL 을 다시 켤 때마다 아래를 쳐야 한다.

```bash
sudo service postgresql start
```

매번 치기 싫으면 `systemd` 를 켠다. `/etc/wsl.conf` 에 아래를 넣고 윈도우 쪽에서
`wsl --shutdown` 한 뒤 다시 들어와 `sudo systemctl enable --now postgresql` 이다.

```ini
[boot]
systemd=true
```

---

## 4. macOS

### 4.1 Homebrew (권장)

```bash
brew install postgresql@17
sed -i '' 's/^#*port = .*/port = 55434/' "$(brew --prefix)/var/postgresql@17/postgresql.conf"
brew services start postgresql@17
```

`postgresql@17` 은 keg-only 라 PATH 에 안 잡힌다. 셸 프로파일에 넣는다.

```bash
echo 'export PATH="'"$(brew --prefix postgresql@17)"'/bin:$PATH"' >> ~/.zshrc
exec "$SHELL" -l
```

### 4.2 역할과 판

Homebrew 의 `initdb` 는 macOS 사용자 이름으로 슈퍼유저를 하나 만들어 두므로,
그 신원으로 아래를 친다.

```bash
createuser -p 55434 -d enode
psql -p 55434 -d postgres -c "ALTER ROLE enode LOGIN PASSWORD 'enode';"
createdb -p 55434 -O enode enode_test
```

### 4.3 Homebrew 를 안 쓰면 — Postgres.app

관리자 권한도 brew 도 없이 가는 길이다.

```text
   1  Postgres.app (PostgreSQL 17 포함) 을 받아 Applications 에 넣는다
   2  실행하고 Initialize 를 누른다
   3  Server Settings 에서 Port 를 55434 로 바꾼다
   4  CLI 도구를 PATH 에 넣는다 — 아래 한 줄을 ~/.zshrc 에
   5  4.2 의 역할과 판을 그대로 만든다
```

```bash
echo 'export PATH="/Applications/Postgres.app/Contents/Versions/latest/bin:$PATH"' >> ~/.zshrc
```

---

## 5. DSN 을 셸에 박고 확인한다

`bash` 면 `~/.bashrc`, `zsh` 면 `~/.zshrc` 다.

```bash
echo "export ENODE_TEST_DATABASE_URL='postgres://enode:enode@127.0.0.1:55434/enode_test?sslmode=disable'" \
  >> ~/.zshrc
exec "$SHELL" -l
```

확인은 셋이다.

```bash
psql "$ENODE_TEST_DATABASE_URL" -c 'SELECT version();'   # 17 이 나온다
git submodule update --init                              # enode-design/ 을 받는다
go test ./...                                            # 전부 ok
```

`scripts/testdb.sh` 는 **이미 잡힌 `ENODE_TEST_DATABASE_URL` 을 그대로 되돌려
준다.** 그래서 CP0 의 첫 줄은 Docker 를 깐 기계와 로컬로 깐 기계에서 같은 글자다.

```bash
eval "$(scripts/testdb.sh)"
```

---

## 6. 대회장에서 실제로 밟는 것

```text
   Go 툴체인을 미리 받아 둔다   go.mod 가 go1.26.6 을 고정하고 GOTOOLCHAIN=auto 가
                              첫 빌드에서 그것을 내려받는다.  현장 네트워크로 받으려
                              하면 거기서 막힌다 — 집에서 go build ./... 을 한 번
                              돌려 캐시에 넣어 온다

   서브모듈도 미리 받는다       enode-design/ 이 비면 INVARIANTS.md 가 통째로 없다

   WSL2 는 리눅스 파일시스템에   저장소를 /mnt/c 아래 두면 Go 빌드와 git 이 몇 배
                              느리다.  ~/ 아래에 클론한다

   포트 충돌                   5432 에 이미 뭔가 떠 있어도 55434 를 쓰므로 안 부딪친다.
                              바꿔야 하면 DSN 도 같이 바꾼다 — 정본은 DSN 한 줄이다

   판을 둘이 안 나눠 쓴다       1절.  같은 DSN 을 두 사람이 쓰면 TRUNCATE 가 서로를 지운다
```

---

## 7. 그래도 안 되면

순서대로 본다.

```bash
pg_isready -h 127.0.0.1 -p 55434            # 서버가 떠 있나
psql "$ENODE_TEST_DATABASE_URL" -c 'SELECT 1'  # 붙을 수 있나 (역할·비밀번호·판)
echo "$ENODE_TEST_DATABASE_URL"             # 셸에 실제로 박혔나
```

`t.Fatal` 로 죽으면서 `ENODE_TEST_DATABASE_URL is unset` 이 나오면 셋째 줄이
비어 있는 것이다. `connection refused` 면 첫째 줄이고, WSL2 라면 3.4 다.
