# 코드 구조 — 오늘의 저장소

이 문서는 **지금 있는 코드**를 적는다. 팩이 더할 것(`GET /v1/nodes` ·
`QUEUED` · `WakeQueued` · `enodectl serve`)은 오늘 없으며, 없다고 그 자리에
적는다. 모든 근거는 리버스 엔지니어링 스캔에 매인 `file:line` 이다.

---

## Build System

### Go 모듈

모듈은 `github.com/taeels/enode` 한 벌이다 (`go.mod`). 언어 하한과 툴체인을
같은 판으로 묶어 슬랙을 없앴다 — 넷이 서로 다른 컴파일러로 짜면 드리프트가
난다.

```text
   module       github.com/taeels/enode
   go           1.26            이보다 낮은 툴체인은 빌드를 거절한다
   toolchain    go1.26.6        실제로 쓸 판.  GOTOOLCHAIN=auto 면 없을 때 받아온다
```

직접 의존은 셋뿐이다 — `github.com/jackc/pgx/v5 v5.10.0` ·
`golang.org/x/sys v0.47.0` · `gopkg.in/yaml.v3 v3.0.1`. 간접 의존은 일곱이다
(`pgpassfile` · `pgservicefile` · `puddle/v2` 가 `pgx` 를 따라 들어오고,
`kr/text` · `rogpeppe/go-internal` 은 테스트 도구, `x/sync` · `x/text`).
자세한 쓰임은 아래 **Critical Dependencies** 에 있다.

### CI — GitHub Actions

정본 CI 는 `.github/workflows/ci.yml` 하나다. `test` 잡과 `cross` 잡으로
갈린다. `package.yml` 과 `release.yml` 은 이 잡 밖의 패키징/릴리스다.

- `test` 잡은 실제 `postgres:17` 서비스를 띄우고(`ci.yml:20-26`, DB `enode` ·
  user `enode`) `ENODE_TEST_DATABASE_URL` 로 붙는다(`ci.yml:31`). 가짜 `claude`
  스텁을 PATH 에 두어(`ci.yml:63`) 커버리지 측정을 안정시킨다.
- `cross` 잡은 세 플랫폼 크로스 빌드 뒤 `enodectl.exe` 의 심볼 상한을 잰다.

트리거는 `push` 가 `main` 에서만 돌고(`ci.yml:4-5`), `pull_request` 는 분기
필터 없이 모든 PR 에서 돈다(`ci.yml:6`).

### 차단 게이트

린트 하나만 경고 전용(`continue-on-error: true`, `ci.yml:117`)이고 나머지는
실패하면 `exit 1` 로 잡을 세운다.

```text
   게이트                                  ci.yml            차단?
   U+2605 장식 문자 grep                    :79 (fail :82)     예
   glyphscan.go AST 스캔                    :92               예
   포맷 (gofmt)                             :95               예
   vet                                      :97               예
   린트 (golangci-lint)                     :116-117          아니오 (경고 전용)
   취약점 (govulncheck)                     :182              예 (DB 불통은 경고 후 exit 0)
   테스트 (go test ./...)                   :215              예
   커버리지 패키지별 80% 하한                :265 (awk floor=80 :269)   예
   스킵 감시 (전 패키지)                     :341 (awk allow :343)      예
   크로스 빌드 (windows/amd64·linux/arm 등) :454              예
   enodectl 심볼 상한                       :479              예
```

심볼 상한은 크로스 빌드한 `enodectl.exe` 를 `go tool nm` 으로 세어
`crypto/tls` 심볼 T 를 10 이하로, `net/http` 를 50 이하로 묶는다
(`grep -c` 는 `ci.yml:482-483`, 임계 비교 `if [ "$tls" -gt 10 ] || [ "$http"
-gt 50 ]` 는 `ci.yml:485`). rc13 에서 `enodectl.exe` 가 AhnLab V3 에
`Trojan/Win.Generic.C5874069` 로 지워진 사고의 방벽이다 —
`setup` 을 형제 `enode` 바이너리로 exec 위임해 `crypto/tls` 를 안 링크하게
만든 것을(`cmd/enodectl/setup.go`) 이 상한이 지킨다. 커버리지 하한은 패키지별
80% 이고, `.coverage-contract.yml` 이 그 **측정 조건**(명령·환경·플랫폼
`linux/amd64`)의 스냅샷을 잡는다 — 게이트 자체는 아니고 게이트의 입력이다.
`.ci-allowed-skips` 는 스킵 면제 명단인데 오늘 비어 있어(주석만) 어떤 스킵도
허용하지 않는다.

---

## Key Classes/Modules

바이너리는 `cmd/` 다섯 벌, 내부 패키지는 `internal/` 열 벌이다.

```text
github.com/taeels/enode
├── cmd/                         실행 바이너리 (package main)
│   ├── mediator/                Mediator HTTP 서버 (Postgres · Record · reaper)
│   ├── enode/                   실행 노드 데몬 (pull-only worker, 인바운드 포트 없음)
│   ├── enodectl/                노드 로컬 제어판 (list/start/stop/logs/status)
│   ├── runctl/                  무상태 클라이언트 CLI (submit/status/record/...)
│   └── iapadapter/              It's a Plan 이슈 트래커 <-> enode 어댑터
└── internal/                    import 경계로 밖에서 못 가져간다
    ├── api/                     Mediator HTTP 표면 (15 라우트)
    ├── store/                   PostgreSQL 상태 계층 + schema.sql
    ├── match/                   순수 매처 (부작용 없음)
    ├── contract/                계약 문법 · Grammar · CheckPlan · 예제
    ├── schema/                  form-only JSON Schema 경계
    ├── record/                  봉인 Run Record (파일시스템 + tar)
    ├── config/                  Mediator 설정 (YAML)
    ├── build/                   실행파일 버전 정보
    ├── runctl/                  runctl 의 HTTP 클라이언트 (Client)
    └── enode/                   노드 데몬 내부 (detect · advertise · worker)
```

층은 한 방향으로 흐른다. `cmd/mediator` 가 `internal/api` 를 마운트하고,
`internal/api` 가 `internal/store` · `internal/match` · `internal/record` 를
부른다. 매칭은 `internal/match` 의 순수 함수이고, 상태 커밋은 `internal/store`
가 트랜잭션으로 한다. `cmd/enode` 는 `internal/enode` 를 엮고, `cmd/runctl` 은
`internal/runctl` 클라이언트를 쓴다.

**오늘 없는 것 (팩이 더한다).** `internal/api` 에 `GET /v1/nodes` 라우트는
없다 — `POST /v1/nodes` (광고) 만 있다. `internal/store` 에 `QUEUED` 상태와
`WakeQueued` 심볼은 없다(`RESOLVING`/`ALLOCATING` 은 정의만 있고 어디에도 안
쓰이는 죽은 상수, 실제로 쓰이는 상태는 `RUNNING`/`VERIFYING`/`SUCCEEDED`/
`FAILED` 넷). `cmd/enodectl` 에 `serve` 서브커맨드는 없다 — daemon 을 띄우는
것은 `start` 가 형제 `enode` 를 exec 하는 것이다.

### Existing Files Inventory

수정 후보인 비테스트 소스 파일 전부를 패키지별로 든다. 테스트 파일
(`*_test.go`)과 `schema.sql` 외의 데이터 파일은 뺀다.

**cmd/mediator/**

```text
   main.go       프로세스 진입점.  설정 로드 · 토큰 부트스트랩 · store/records
                 열기 · migrate · reaper 고루틴 · HTTP serve/shutdown
   setup.go      mediator setup 서브커맨드.  대화형 DB/role/database 프로비저닝 +
                 설정 쓰기 (flag 파싱 전에 돌고 config/DB 가 필요 없다)
```

**cmd/enode/**

```text
   main.go       데몬 진입점.  detect · advertise · worker 세 고루틴을 엮는다
   hook.go       enode hook 서브커맨드 디스패치 (stop 훅)
   setup.go      enode setup 서브커맨드 디스패치 (대화형 설정 부트스트랩)
```

**cmd/enodectl/**

```text
   main.go               커맨드 디스패치 + list/setup/id/start/stop/logs/status
   setup.go              cmdSetup.  setup 을 형제 enode 로 exec 위임 (crypto/tls
                         를 안 링크하려고)
   proc_unix.go          !windows 프로세스 제어 (processAlive/signalStop/ownsConfig ...)
   proc_windows.go       windows 프로세스 제어 (OpenProcess/TerminateProcess ...)
   caffeinate_darwin.go  darwin.  keepAwake 가 caffeinate -i -s -w <pid>
   caffeinate_other.go   !darwin.  no-op keepAwake
```

**cmd/runctl/**

```text
   main.go       CLI 디스패치 · flag permutation · 종료 코드 사상 · 11 개
                 서브커맨드 핸들러 (submit/status/record/cancel/example/lint/
                 schema/capabilities/dry-run/asks/answer)
   shape.go      오프라인 계약 작성 헬퍼 (cmdExample · cmdLint · cmdSchema)
```

**cmd/iapadapter/**

```text
   main.go          어댑터 진입점 + 오케스트레이션 루프 (claim/handle/heartbeat/
                    follow/handOff/finish).  이슈 키에서 run ID 를 파생해 재기동 안전
   itsaplan.go      It's a Plan 트래커 HTTP 클라이언트 (claim/heartbeat/result/...)
   mediator.go      enode Mediator HTTP 클라이언트 (runs/asks/answers/ledger/blobs)
   config.go        adapter.yaml YAML 로더 + duration 헬퍼
   contract.go      고정 템플릿 Run 계약 빌더 (추론 0)
   comment.go       물음/결과를 트래커 코멘트로 렌더 + 결과 마커
   orchestrator.go  enode --once 오케스트레이터 서브프로세스 기동/정지/디태치
```

**internal/api/**

```text
   api.go        Mediator HTTP 표면 전부.  Server · 라우트표 · auth 미들웨어 ·
                 15 개 핸들러 메서드.  SQL 은 없다 — store 에 위임한다
```

**internal/store/**

```text
   store.go      Store 타입 · 연결/Migrate · advert upsert/read · Run/CreateRun/
                 GetRun · run 상태 상수 · capability 뷰
   schema.sql    nodes · runs · leases · steps DDL (embed.  Migrate 가 통째 exec)
   verdict.go    step 상태 상수 · Verify (계약 검사, 순수) · describeWant · StepResults
   reap.go       Reap (만료 임대 회수) · sealExpired · RunReaper · Cancel · SettleIfDone
   ask.go        ASK 생명주기 (raiseAsks/AnswerStep/PendingAsks/ExpireAsks/PushAsks)
                 + 제안 adopt/retire/rewind
   claim.go      RenewLeases · ClaimStep/redeliver/FailRestarted · ReportStep ·
                 afterStep/applyStepEffects · StepResult 타입
   expand.go     applyExpands (plan -> 계약 버전 + step 행) · readPlan · skipAdopters
   dispatch.go   applyDispatch (가지 선택) · propagateSkips · chosen 표시
   acquire.go    runAcquires/doAcquire/tryGrab/bindRole (acquire 단계)
   release.go    applyRelease (단계의 release 역할에 대한 부분 임대 해제)
   rollback.go   rollBack/loopBack/validateBack (loop · validate_with 재시도)
   observe.go    Steps/StepView · Ledger/LedgerEntry · LiveContract · priorRuns
   seal.go       Manifest/ContractVersion · StepFiles · sealRecord · ByStep
```

**internal/match/**

```text
   match.go      순수 매처.  advert 를 attrCount 오름차순 정렬 후 422 영구 불능 ->
                 409 점유를 검사, []Assignment 또는 *Reject 를 낸다
```

**internal/contract/**

```text
   contract.go   계약 타입 모델 (Contract/Require/Step/Condition/Loop/Dispatch/
                 Acquire/Ask ...) + Contract.Validate().  문법의 단일 정본
   advert.go     Advert/Capability 타입 + subset-match Satisfies (노드 어휘)
   planshape.go  PlanShape 상수 (예시로 보여주는 단계 필드 모양, 스키마가 아니다)
   checkplan.go  CheckPlan + PlanDoc (부모 계약 없이 Contract.Validate 를 재사용)
   example.go    examples/*.json embed.  ExampleNames/Example (붙여넣어 도는 계약)
   grammar.go    Grammar 상수 (plan 작성 에이전트 프롬프트에 실리는 규칙 전문)
```

**internal/schema/**

```text
   schema.go     허용/거부 키워드 표 · CheckBoundary (form-only 게이트) · Validate.
                 질을 재는 키워드를 계약 검증 시점에 하드 거부한다
```

**internal/record/**

```text
   record.go     Store.  Open · AppendLog · WriteBlob/OpenBlob/Blobs · Sealed ·
                 Seal (chmod 봉인) · Tar (봉인 번들 스트리밍)
```

**internal/config/**

```text
   config.go     Mediator 설정 (YAML) 읽기.  Config 구조 + Lease/Claim/Notify 기본값
   write.go      NewToken (노드와 Mediator 가 나눠 갖는 비밀 생성) + 설정 파일 쓰기
```

**internal/build/**

```text
   build.go      실행파일 버전 정보 (자기 갱신 · --version).  광고에는 안 싣는다 —
                 매처는 동등 비교뿐이라 버전 문자열을 못 쓴다
```

**internal/runctl/**

```text
   client.go     runctl 의 HTTP 클라이언트.  Client · Principal · Submit/Asks/
                 Answer/Status/Cancel/Record/Wait/Capabilities + DTO + Terminal
```

**internal/enode/**

```text
   advertise.go        Client (Mediator 로 나가는 유일 방향) + Advertiser 고루틴
   agent.go            AgentParams + buildPrompt/봉투 조립 (출력 계약 · 스키마 ·
                       피드백 · 요청)
   argv.go             커맨드 단계 argv 의 $OUT/$IN 확장 (셸 안 씀)
   changed.go          Stamp/Changed.  타임스탬프 이후 파일시스템 변경 탐지
   child.go            exec.Cmd 를 플랫폼 프로세스 그룹/속성 설정으로 감싼다
   claim.go            Step · Result · Client claim/blob/log/report + Worker
                       (execute/runAgentStep).  Worker 가 여기 산다 (worker.go 는 없다)
   claude.go           claude 하네스 어댑터 (Argv/Decode/Instrument/Version/Usable)
   collect.go          collectDeclared.  계약 선언 워크스페이스 경로를 $OUT 으로 복사
   config.go           Local 설정 구조 + LoadLocal (YAML) + SampleLocal
   console_unix.go     플랫폼별 console/tty 처리 (unix 반쪽)
   console_windows.go  플랫폼별 console/tty 처리 (windows 반쪽)
   detect.go           Detect.  cheap vs costly 속성 탐지 · capabilities 조립
   detector.go         Detector 고루틴 · Capabilities{Caps,At} · DefaultDetectEvery (5분)
   diff.go             writeWorkspaceDiff.  워크스페이스 git diff 를 $OUT 에 담는다
   disk_unix.go        free-disk 바이트 syscall (unix 반쪽)
   disk_windows.go     free-disk 바이트 syscall (windows 반쪽)
   env.go              env 화이트리스트 조립 (harnessEnv) · Credentials/Transparent ·
                       commandEnv · RunIdentity
   harness.go          Harness 인터페이스 · HarnessResult · Reason · ParseClaude ·
                       어댑터 레지스트리 (지금은 claudeHarness 하나)
   hook.go             enode hook stop 구현 (RunStopHook/WriteHookSettings/HookArgs)
   identity.go         Identity · Derive/DeriveFor (node id 를 설정 경로로 키)
   leases.go           Held.  광고 응답이 통째로 갈아끼우는 인메모리 임대 집합
   lock.go             Lock.  설정 파일 옆 단일 실행 잠금 (플랫폼 무관 껍데기)
   lock_unix.go        단일 실행 flock (unix 반쪽)
   lock_windows.go     단일 실행 flock (windows 반쪽)
   paths.go            ConfDir/StateDir/ConfigPaths/ResolveConfig
   repoid.go           CanonicalRepoID · DetectRepo
   runner.go           runHarness.  유일한 exec 지점 (IOPaths · Job)
   setup.go            Setup/SetupCLI 대화형 설정 부트스트랩
   workspace.go        Worker.Prepare/reset/clean/checkout (git sanitize) ·
                       WorkspaceSpec · Prep
```

---

## Design Patterns

### build-tag 플랫폼 쌍

플랫폼별 코드는 런타임 `GOOS` 분기 대신 build-tag 로 가른 파일 쌍으로 둔다.
한쪽을 고치면 다른 쪽도 같이 봐야 하는 자리다.

```text
   cmd/enodectl/proc_unix.go (!windows)      / proc_windows.go (windows)
   cmd/enodectl/caffeinate_darwin.go (darwin) / caffeinate_other.go (!darwin)
   internal/enode/console_unix.go             / console_windows.go
   internal/enode/disk_unix.go                / disk_windows.go
   internal/enode/lock_unix.go                / lock_windows.go   (lock.go 가 공통 껍데기)
```

`proc_unix.go` 의 제약식은 `unix` 가 아니라 `!windows` 다 — 올바른 쌍이지만
표현이 다르다. windows 반쪽(`proc_windows.go` · `lock_windows.go` ·
`disk_windows.go`)은 linux 커버리지 프로필에 안 잡히므로 `.coverage-contract.yml`
이 측정 플랫폼을 `linux/amd64` 로 못 박는다.

### 무상태 클라이언트 (runctl)

`internal/runctl/client.go` + `cmd/runctl` 은 로컬 상태를 두지 않는 "제출하고
잊는" 표면이다. 토큰을 `Authorization: Bearer` 로, 신원을 `X-Enode-Principal`
(git 이메일) 로 실어 Mediator 에 HTTP 로만 말한다. `Wait` 는 상태를 들고 있는
상호작용이 아니라 `--poll` 주기 폴링이고(`client.go:210-213`), 종료 판정은
`Terminal(state)` 가 `SUCCEEDED`/`FAILED` 둘로만 한다(`client.go:208`). 토큰을
flag 기본값에 안 넣어 usage 출력/로그에 새지 않게 한다. `enodectl` 도 같은
결로 privileged 작업(`setup`)을 형제 `enode` 에 exec 위임해 링크 표면을
최소로 둔다.

### 봉인 Record 패턴

`internal/record` 는 Run Record 를 파일시스템에 쌓고 한 번 봉인하면 못 고치게
만든다. `Seal` 은 `manifest.json` · `steps/NN-*.json` · `verdict.json` 을 쓴 뒤
모든 파일을 `0o444`, 디렉터리를 `0o555` 로 chmod 한다(`record.go`). 봉인
여부는 `verdict.json` 의 쓰기 비트가 꺼졌는지로 감지한다
(`Mode().Perm()&0o200 == 0`). 봉인은 멱등이다. `CreateRejectedRun` 은 `runs`
행만 넣고 Record 를 안 건드리므로 `verdict.json` 이 없어 `Sealed` 가 계속
false 이고, 그래서 거부된(FAILED) Run 에 `GET /v1/runs/{id}/record` 는 409 를
낸다. blob 은 `(attempt, seq)` 최댓값으로 되읽어 재시도 루프가 같은 출력 이름을
덮어써도 살아남게 한다.

### 롱폴 claim

`postClaim` 은 1초 ticker 로 `Claim.LongPollSeconds` 기한까지 돌다 비면 204 로
끊는다. `cmd/mediator/main.go` 는 claim 이 최대 2시간까지 매달릴 수 있어
`WriteTimeout` 을 **일부러 설정하지 않는다**(`main.go:107-108`) — 계획자는
여기에 write timeout 을 더하면 안 된다. 노드 쪽에서는 `Worker.Run`
(`internal/enode/claim.go:348`) 이 전용 `poll()` 클라이언트로 롱폴하고, 200 을
받으면 즉시 `Held.Add(step.Lease)` 로 임대를 기록한 뒤 1초 watchdog 으로 임대
만료 시 실행 컨텍스트를 취소한다.

---

## Critical Dependencies

직접 의존은 셋이고, 각자 한 축씩만 맡는다.

**github.com/jackc/pgx/v5 v5.10.0** — Mediator 상태 계층(`internal/store`)
전체가 이 위에 선다. `pgxpool` 연결, `FOR UPDATE ... SKIP LOCKED LIMIT 1`
디스패치(`claim.go`), savepoint 로 감싼 임대 INSERT(`acquire.go`), unique
violation(23505) 을 `store.ErrNodeTaken` 으로 사상하는 것(`store.go:271`)이 모두
pgx 기능이다. `schema.sql` 을 embed 해 `Migrate` 가 통째 exec 한다 — 마이그레이션
도구는 없고 additive `ALTER ... ADD COLUMN IF NOT EXISTS` 를 쓴다. 노드 데몬과
클라이언트 CLI 는 pgx 를 링크하지 않는다.

**golang.org/x/sys v0.47.0** — 플랫폼 syscall 축이다. 단일 실행 flock
(`internal/enode/lock_unix.go`), free-disk statfs (`disk_unix.go`), windows
프로세스 제어(`cmd/enodectl/proc_windows.go` 의 OpenProcess/TerminateProcess)가
여기서 온다. 위 build-tag 반쪽들이 주 소비자다.

**gopkg.in/yaml.v3 v3.0.1** — 설정 파싱 축이다. 노드 `local.yaml`
(`internal/enode/config.go` 의 `LoadLocal`), Mediator 설정
(`internal/config/config.go`), iapadapter 의 `adapter.yaml`
(`cmd/iapadapter/config.go`) 가 모두 이 파서를 쓴다.

간접 의존 일곱은 위 셋을 따라 들어온다 — `pgpassfile`/`pgservicefile`/
`puddle/v2` 는 pgx 가, `kr/text`/`rogpeppe/go-internal` 은 테스트 도구가,
`x/sync`/`x/text` 는 상위 라이브러리가 끌어온다. `scripts/avprobe/` 는 자기
`go.mod` 를 따로 두어 메인 모듈의 `./...` 에 안 잡힌다.
