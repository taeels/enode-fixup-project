# 기술 스택

이 문서는 **오늘 저장소가 실제로 쓰는 것**을 적는다. 계획 팩이 더할 축은
(대시보드 · 큐 등) 여기 없다.

## Programming Languages

- **Go 1.26** — `go.mod` 의 `go 1.26`, 툴체인은 `go1.26.6`. 제품 코드는
  전부 Go 하나다 (`cmd/*` 다섯 · `internal/*` 열). 검사 스크립트도 Go 다
  (`scripts/glyphscan.go`).
- 보조로 **bash**(`scripts/testdb.sh`, `packaging/*.sh`)와 설정 형식
  **YAML**(enode `local.yaml` · mediator/adapter 설정)이 있으나, 실행 로직은
  Go 로만 짠다.

## Frameworks

**웹 프레임워크가 없다.** HTTP 표면은 표준 라이브러리 `net/http` 의
`http.ServeMux` 만으로 짠다 — 라우터/미들웨어 라이브러리 의존이 없다.
`internal/api/api.go` 가 `mux.HandleFunc` 로 15 개 라우트를 직접 등록하고
`s.auth(...)` 로 하나씩 감싼다. Go 1.22+ 의 메서드+경로 패턴
(`POST /v1/nodes/{id}/claim` 같은)을 그대로 쓴다.

DB 접근은 드라이버 위에서 직접 SQL 을 쓴다 — **ORM 이 없다.**

| 층 | 의존 | 쓰임 |
|---|---|---|
| DB 드라이버 | `github.com/jackc/pgx/v5 v5.10.0` | `internal/store` 의 pgx 풀. embedded `schema.sql` 을 `Migrate` 가 그대로 exec |
| 설정 파싱 | `gopkg.in/yaml.v3 v3.0.1` | `internal/config`(Mediator) · `internal/enode`(`local.yaml`) · iapadapter 설정 |
| 시스템 콜 | `golang.org/x/sys v0.47.0` | 플랫폼별 lock · disk · console (`*_unix.go`/`*_windows.go`) |

`go.mod` 의 **직접 의존은 이 셋뿐**이고, 나머지 일곱(pgpassfile ·
pgservicefile · puddle/v2 · kr/text · rogpeppe/go-internal · x/sync · x/text)은
간접이다.

## Infrastructure

- **PostgreSQL** — Mediator 상태(`nodes`/`runs`/`leases`/`steps`)의 저장소.
  경합·가변 사실은 DB 에, 봉인된 Run Record 는 파일시스템에 둔다. CI 는
  `postgres:17` 서비스 컨테이너로 돌고(`.github/workflows/ci.yml`), 로컬은
  `scripts/testdb.sh` 가 같은 이미지를 띄워 `ENODE_TEST_DATABASE_URL` 을
  내보낸다.
- **파일시스템** — `internal/record` 가 `run-<id>/` 아래 `manifest.json` ·
  `steps/NN-*.json` · `logs/NN-*.log` · `blobs/` · `verdict.json` 을 쓰고,
  `verdict.json` 의 쓰기 비트가 꺼진 것이 봉인 표식이다. blob 본체도 여기
  산다.
- **Claude CLI 하네스(서브프로세스)** — 에이전트 스텝은 `internal/enode` 가
  `claude` 바이너리를 서브프로세스로 부른다. `internal/enode/runner.go` 의
  `runHarness` 가 유일한 exec 지점이고, stdout 을 `bytes.Buffer` 에 통째로
  모아 프로세스 종료 뒤에 `Decode` 한다(라이브 스트림 없음). 기본 하네스는
  `claude`, `Local.HarnessBin` 으로 덮는다. 하네스 레지스트리에는 지금
  `claudeHarness{}` 하나뿐이다.
- **알림 웹훅(보조)** — asks 알림용 fire-and-forget POST
  (`internal/store/ask.go`, `NotifyURL`). 재시도 없음 — 인박스(`GET /v1/asks`)가
  정본이다.
- **git 서브프로세스** — enode 가 워크스페이스 준비에 `git`(`reset --hard` ·
  `checkout --detach` · `fetch` · `clean` 등)과 `repo forall` 을 부른다.

## Build Tools

| 도구 | 쓰임 |
|---|---|
| `go` | 빌드 · 테스트 · 크로스빌드(`cross` job) · `go tool nm`(심볼 수 측정). 툴체인 `go1.26.6` |
| `gofmt` | 포맷 게이트 (`.github/workflows/ci.yml`, 어긋나면 `exit 1`) |
| `go vet` | 정적 검사 게이트 |
| `golangci-lint` v2 | `.golangci.yml` — `default: none` + 린터 5 개, `max-same-issues: 0`. CI 에서는 경고 전용(`continue-on-error: true`), 유일한 비차단 스텝 |
| 커스텀 glyphscan | `scripts/glyphscan.go` (빌드 태그 `ignore \|\| glyphscan`). 비-test Go 파일의 문자열/문자 리터럴에서 장식 문자를 AST 로 잡는다. 이와 별도로 CI 가 `grep -rlIP` 로 U+2605 를 담은 파일 수가 0 인지 본다 |
| `scripts/testdb.sh` | 로컬 `postgres:17` 컨테이너(포트 55434, DB `enode_test`)를 띄우고 `ENODE_TEST_DATABASE_URL` export 를 출력한다 |
| `govulncheck` | 차단성 취약점 스캔 (DB 못 닿으면 warn + `exit 0`, 3 회 재시도) |
| 패키징 | nfpm(.deb/.rpm) · wixl(MSI) · macOS 스크립트 (`packaging/`). 서비스 매니저 연동(systemd/launchd/Windows 서비스)은 아직 없다 — 설치까지만 |

## Testing Tools

- **`go test ./...`** — 콜로케이트된 `*_test.go` 로 유닛·통합을 한 벌에 돈다.
  CI 는 테스트 게이트로 한 번, 커버리지 계측으로 또 한 번, 총 두 번 돌린다.
- **커버리지: `go test -coverprofile`** — CI 가 `/tmp/cover.out` 으로 뽑는다.
- **커버리지 바닥 게이트(awk)** — 패키지별 80% 를 `awk -v floor=80` 으로
  강제하고(`.github/workflows/ci.yml`), 한 패키지라도 미달이면 `exit 1`.
  awk 는 같은 coverpkg 블록을 최댓값으로 병합한다(단순 합산은 오독).
  `internal/build` 는 16/20 = 80.0% 로 딱 바닥에 붙어 있다.
- **스킵 감시(awk)** — 허용 목록 `.ci-allowed-skips`(현재 비어 있음, 주석뿐)
  밖의 스킵은 0 이어야 한다. `/tmp/test.json` 을 읽어 강제하고, 밖의 스킵이
  있으면 `exit 1`.
- **커버리지 기준선 스냅샷** — `.coverage-contract.yml` 은 측정 조건
  (명령 · 환경 · 플랫폼 `linux/amd64`)을 고정하는 문서이지 게이트 입력이
  아니다. 게이트는 위의 awk 스텝이다.
- **DB 의존 테스트 스킵** — `ENODE_TEST_DATABASE_URL` 이 없으면 DB 를 쓰는
  테스트는 `t.Skip` 으로 빠진다. `internal/api` 는 테스트 함수 103 개 중 약
  102 개가 이 조건으로 스킵된다 — CI 의 Postgres 서비스가 있어야 비로소 이
  표면이 검증된다.
