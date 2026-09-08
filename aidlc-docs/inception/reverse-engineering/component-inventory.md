# 컴포넌트 목록

이 문서는 **오늘의 코드가 담은 패키지**를 센다. 모듈은
`github.com/taeels/enode` 하나이고 (`go.mod`), 그 아래 `package main` 실행파일
다섯과 `internal/*` 공유 패키지 열이 있다. 계획 팩이 더할 것 —
`GET /v1/nodes` · `QUEUED` · `WakeQueued` · `enodectl serve` — 은 아직 없다.
없는 자리는 해당 패키지 줄에서 짚는다.

`scripts/glyphscan.go` 와 `scripts/avprobe/` 는 이 셈에서 뺀다. 앞엣것은 빌드
태그 `ignore || glyphscan` 이라 `./...` 에 안 잡히고, 뒤엣것은 자기 `go.mod`
를 따로 가져 본 모듈 밖이다.

## Application Packages

`cmd/` 아래 다섯 개가 각각 `package main` 이고 바이너리 하나로 빌드된다.

| 패키지 | 역할 | 파일 |
|---|---|---|
| `cmd/mediator` | Mediator 서버 프로세스. 설정 로드 · 토큰 부트스트랩 · `store`/`record` 열기 · 스키마 마이그레이션 · 임대 회수 고루틴을 엮고 `api.New(...).Handler()` 를 유일한 HTTP 핸들러로 올린다 | `main.go` · `setup.go` |
| `cmd/enode` | 실행 노드 데몬. detect · advertise · worker 세 고루틴을 띄우고 전부 Mediator 로 바깥 방향 HTTP 를 건다. 인바운드 포트를 열지 않는다 (ADR-014) | `main.go` · `hook.go` · `setup.go` |
| `cmd/enodectl` | 노드-로컬 제어 CLI. 서브커맨드는 `setup` · `list` · `id` · `start` · `stop` · `logs` · `status` · `version` 이다. **`serve` 는 오늘 없다** | `main.go` · `setup.go` · `proc_unix.go` · `proc_windows.go` · `caffeinate_darwin.go` · `caffeinate_other.go` |
| `cmd/runctl` | 무상태 클라이언트 CLI. 서브커맨드 열하나 — `submit` · `status` · `record` · `cancel` · `example` · `lint` · `schema` · `capabilities` · `dry-run` · `asks` · `answer` | `main.go` · `shape.go` |
| `cmd/iapadapter` | "It's a Plan" 트래커를 enode 함대에 잇는 유일한 바깥-대면 바이너리. DB 를 안 쥐고 run-ID 를 이슈 키에서 파생해 재기동에 안전하다 | `main.go` · `itsaplan.go` · `mediator.go` · `config.go` · `contract.go` · `comment.go` · `orchestrator.go` |

## Shared Packages

`internal/` 아래 열 개. 실행파일들이 이것들을 링크해서 쓴다.

| 패키지 | 한 줄 역할 |
|---|---|
| `internal/api` | Mediator 의 HTTP 표면. `http.ServeMux` 라우트 표 · Bearer 인증 미들웨어 · 핸들러들. `mux.HandleFunc` 로 15 개 라우트를 직접 등록한다 (`GET /v1/nodes` 는 없다 — `POST /v1/nodes` 만) |
| `internal/store` | PostgreSQL 상태 계층. `nodes`/`runs`/`leases`/`steps` 를 담고 시퀀싱 · `Verify` · 봉인 · `Reap` 을 한다. run 상태로 `QUEUED` 를 쓰지 않고 `WakeQueued` 도 없다 |
| `internal/match` | 순수 매처. `requires` 를 광고 노드에 얹어 배정(`[]Assignment`) 또는 타입 있는 거부(`*Reject`, 422/409)를 낸다. 부수효과 없음 |
| `internal/contract` | run 계약 문법 모델(`Contract`/`Step`/`Require`/`Condition` 등)과 `Grammar` · `PlanShape` · `CheckPlan` · 내장 `Example` |
| `internal/schema` | 폼 검사로만 좁힌 JSON Schema. `CheckBoundary`(경계 게이트) · `Validate`(문서 검사). 품질 판정 키워드는 하드 거부 |
| `internal/record` | 파일시스템 Run Record 를 짓고 로그 · blob 을 붙이고 `Seal`(chmod)로 봉인 · `Tar` 로 내보낸다 |
| `internal/enode` | 실행 노드 데몬 로직. `Detector` · `Advertiser` · `Worker` · `Harness`(claude 어댑터) · `workspace` 준비 |
| `internal/runctl` | runctl CLI 의 HTTP 클라이언트. `Submit` · `Status` · `Cancel` · `Record` · `Asks` · `Answer` · `Capabilities` 와 DTO |
| `internal/config` | Mediator 설정 로드 · 기록 (ADR-015 §4). `Lease` 기본값(`RenewSeconds`/`NotAfterFactor`) 등 |
| `internal/build` | 실행파일의 신원(커밋 · 판 · 날짜)을 담고 `<cmd> --version` 이 찍는 `Version` 을 낸다. 광고에는 안 싣는다 |

## Test Packages

별도의 테스트 패키지 트리는 없다. 테스트는 **각 패키지 안에 콜로케이트된
`*_test.go`** 로 산다 (`_unix_test.go`/`_windows_test.go` 는 플랫폼별). 예로
`Worker` 는 `internal/enode/claim.go` 에 있고 `worker.go` 라는 파일은 없지만
`worker_test.go`/`worker_unix_test.go` 는 이름을 쓴다.

| 패키지 | `*_test.go` 개수 |
|---|---|
| `cmd/enode` | 2 |
| `cmd/enodectl` | 4 |
| `cmd/iapadapter` | 9 |
| `cmd/mediator` | 3 |
| `cmd/runctl` | 3 |
| `internal/api` | 4 |
| `internal/build` | 1 |
| `internal/config` | 2 |
| `internal/contract` | 4 |
| `internal/enode` | 28 |
| `internal/match` | 1 |
| `internal/record` | 1 |
| `internal/runctl` | 1 |
| `internal/schema` | 2 |
| `internal/store` | 3 |

**DB 의존 테스트는 Postgres 가 없으면 스킵된다.** `ENODE_TEST_DATABASE_URL`
이 비어 있으면 테스트 헬퍼가 `t.Skip("ENODE_TEST_DATABASE_URL is unset ...")`
로 빠진다 (`internal/api/api_test.go:33`). `internal/api` 는 테스트 함수 103 개
가운데 **약 102 개가 스킵**되어 Postgres 없이는 사실상 아무 것도 검증하지
못한다. CI 는 이래서 `postgres:17` 서비스 컨테이너를 붙이고
(`.github/workflows/ci.yml`), 로컬은 `scripts/testdb.sh` 가 같은 컨테이너를
띄운다.

## Total Count

- **Application packages (`cmd/*`)**: 5
- **Shared packages (`internal/*`)**: 10
- **합계**: **15 패키지** (본 모듈에서 빌드/테스트되는 것 기준)

`scripts/glyphscan.go`(빌드 태그로 제외)와 `scripts/avprobe/`(별도 모듈)는
이 15 에 들어가지 않는다.
