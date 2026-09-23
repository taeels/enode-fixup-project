# 코드 구조 — 오늘의 파일

**2026-09-23 전면 재측정.** 기준 커밋 `195a5d0`. 파일 목록은 고칠 후보의
목록이다 — 브라운필드 회차가 이 표에서 자기 파일 행렬을 뽑는다.

---

## 빌드 시스템

```text
   모듈        github.com/taeels/enode
   Go          go 1.26 · toolchain go1.26.6.  둘을 같은 값으로 묶어 슬랙을 없앴다
   빌드        go build ./...   별도 빌드 도구가 없다
   테스트      go test ./... -count=1 -coverpkg=./...  (커버리지 계약이 명령을 고정한다)
   격리 시험    go test -tags integration ./internal/enode  (환경변수 셋이 있어야 돈다)
   DB          scripts/testdb.sh 가 postgres:17 컨테이너를 띄운다 (enode_test · 55434)
   패키징      packaging/{linux,macos,windows}.  nfpm · install.sh · wixl
   CI          .github/workflows/ci.yml — 잡 둘 (test · cross)
```

설정 파일 넷이 게이트의 입력이다.

```text
   .coverage-contract.yml   커버리지 측정 명령과 하한 80% 의 정본
   .ci-allowed-skips        허용된 t.Skip 의 목록.  그 밖은 상한 0
   .golangci.yml            린트 설정 (경고 전용)
   .gitattributes           audit.md 의 merge=union
```

---

## 모듈 계층

```text
   cmd/*                    껍데기.  플래그 · 종료코드 · 배선만
     |
   internal/api             HTTP 표면 ------- internal/api/ui ---- internal/transcriptui
     |          \                                                  (카드 한 장)
     |           internal/transcript (파서.  표준 라이브러리만)
   internal/store  ---- internal/match (순수)
     |          \       internal/contract ---- internal/schema
   internal/record  \
                     internal/environment (profile · 준비 산출물 · 결과 기록 타입)
                    /
   internal/enode           노드 로직.  contract · environment · transcript 를 문다
     |
   internal/panel  ---- internal/runctl · internal/proc · internal/transcript · transcriptui
```

**두 나무가 `contract` 말고 한 자리에서 더 만난다.** Mediator 쪽(`api` · `store` ·
`record`)과 노드 쪽(`enode` · `panel` · `proc`)이 `internal/environment` 를 함께
문다 — `store` 는 결과 JSON 의 `execenv.Record` 타입 하나 때문이다
(`dependencies.md` 의 「두 나무가 한 자리에서 만난다」). 경계 검사
(`internal/panel/boundary_test.go`)의 금지 열두 줄에 `environment` 가 없다.

---

## 파일 목록 — 고칠 후보

### cmd/

```text
   cmd/mediator/main.go        배선 · reaper 고루틴 · serve/shutdown
   cmd/mediator/setup.go       DB role/database 대화형 프로비저닝
   cmd/enode/main.go           데몬.  하위명령 다섯을 앞에서 가로챈다.
                               profile 이 있으면 기동 전에 CheckWithRuntime 이 ready 여야
                               하고, driver 로 NativeRuntime 또는 RuncOverlayRuntime 을 고른다
   cmd/enode/environment.go    enode env check|apply --config PATH [--json]
   cmd/enode/hook.go           enode hook — 하네스 훅 진입점
   cmd/enode/setup.go          enode setup
   cmd/enode/panel.go          enode panel — 제어판을 띄운다
   cmd/enodectl/main.go        하위명령 디스패치 · 라이브니스.  start 가 env check 를 먼저 exec 한다
   cmd/enodectl/environment.go 형제 enode env 를 exec 한다
   cmd/enodectl/serve.go       형제 enode panel 을 exec 한다
   cmd/enodectl/setup.go       형제 enode setup 을 exec 한다
   cmd/enodectl/proc_*.go      플랫폼 쌍
   cmd/enodectl/caffeinate_*.go  darwin 에서 잠들기를 막는다
   cmd/runctl/main.go          하위명령 열하나 · 종료코드 0-3
   cmd/runctl/shape.go         계약 문법과 팩 모양을 사람에게 설명한다
   cmd/iapadapter/*.go         브리지 일곱 — main · config · contract · itsaplan ·
                               mediator · orchestrator · comment
```

`enode runtime-helper` 는 사람이 부르는 하위명령이 아니다 — runc-overlay 가
`unshare` 안에서 자기 실행파일을 다시 띄우는 사적인 진입점이고, 설정과 플래그
파싱보다 앞에서 가로챈다.

### internal/api — Mediator 표면

```text
   api.go                     라우팅 + 핵심 핸들러.  1,122 줄.  PUT log 가 ?progress=1 로 갈린다
   log.go                     GET .../log — 봉인 전은 진행 파일, 봉인 뒤는 logs/ (FR-6)
   nodes.go                   GET /v1/nodes (ADR-065)
   runs.go                    GET /v1/runs + 질의 인자 넷
   ratelimit.go               데모 모드의 전역 토큰버킷
   demo.go                    POST /v1/demo/runs — 고정 시나리오
   demo_gallery.go            갤러리 라우트 여섯
   demo_gallery_history.go    갤러리 이력
```

### internal/api/ui — 화면

```text
   ui.go                          임베드 + 보안 헤더 + webcam origin 검증 + 카드 모듈 경로
   static/index.html · landing.*  랜딩
   static/cardnews/               온보딩 카드뉴스
   static/demo/                   데모 번들 — 제출 · 갤러리 · 투어 · 웹캠
   static/fleet/                  함대 현황판 진입점 (app.mjs · login.mjs)
   static/shared/fleet/           현황판 본체 — client · model · view · scene · format ·
                                  identity · transcript-poller · fleet.css
   tests/*.test.mjs               브라우저 테스트 열넷
```

### internal/panel — 노드 제어판

```text
   panel.go       Config · Server · New · Handler · 토큰 미들웨어 · loopback 판정.  라우트 열하나
   handlers.go    state · drain · undrain · stop · start · logs
   transcript.go  transcript(링) · runs(Mediator 목록 거르기) · record(tar 풀기) · card
   headers.go     모든 응답에 보안 헤더 다섯.  하네스 출력을 그리게 되며 생겼다
   view.go        State 와 그 다섯 묶음.  각 묶음이 독립 실패한다
   page.go        화면 한 장.  시안 design/enode-ux.pen 의 다크 토큰
```

### internal/enode — 노드 로직

```text
   claim.go        Worker 루프 · 단계 실행 · 업로드 · 결과 보고.  1,030 줄
   runtime.go      StepRuntime · StepSession 인터페이스 · NativeRuntime · HarvestSpec/Result
   runc_overlay_linux.go   RuncOverlayRuntime · 세션 · helper · OCI config · 준비도 smoke.
                           1,285 줄로 이 패키지에서 가장 큰 파일이다
   runc_overlay_other.go   linux 가 아니면 생성자가 실패한다
   environment.go  노드 설정의 environment 바인딩 + 공유 profile 을 읽는다
   overlay_linux.go · overlay_other.go   overlay 탐침 — kernel · userns · fuse 사다리
   runner.go       유일한 하네스 exec 관문 · 계장 · 봉인 로그 선별(selectLogs)
   emit.go         하네스 stdout 줄을 사건으로 흘린다 (도는 중)
   upload.go       도는 동안의 원문을 PUT log?progress=1 로 민다.  실행을 안 막는다
   claude.go       하네스 어댑터 — Argv · Decode · Instrument · Usable · Version
   harness.go      Reason · HarnessResult · ParseClaude · Event · 레지스트리
   transcript.go   고정 크기 링 파일 — OpenRing · Write · Reset · ReadRing
   mcp.go          MCP 허용목록 · 팩 tar 읽기 · 노드 선언
   policy.go       소유자 정책 파일 (drain · panel_token)
   status.go       상태 파일 (탐지 능력과 그 시각)
   advertise.go    광고 = 하트비트 = 갱신
   detect.go · detector.go   능력 탐지와 그 시계.  machine · arch.<이름> · overlay 가 신규
   workspace.go    워크스페이스 준비 — reset --hard 뒤 clean -df.  checkout 이 없어졌다
   diff.go · changed.go · collect.go   수확의 재료 — git diff · 기준 시각 이후 변경 · collect
   env.go          환경 화이트리스트 (R1)
   hook.go         훅 설정 쓰기
   identity.go · config.go · paths.go · repoid.go     신원과 경로
   leases.go · lock.go · lock_*.go                    임대와 단일 인스턴스
   agent.go · argv.go · child.go · setup.go
   console_*.go · disk_*.go                           플랫폼 쌍
```

빌드 태그 쌍이 다섯이다 — `console_*` · `disk_*` · `lock_*` · `overlay_*` ·
`runc_overlay_*`. 뒤의 둘은 `linux` 와 `!linux` 로 갈린다(나머지는 unix 와 windows).

### internal/environment — 실행 환경 (신규)

```text
   profile.go     Profile 타입 · Parse (YAML 별칭 · 앵커 · merge 키 거절, KnownFields) · Validate
   check.go       Binding · Fact · Operation · Report · 상태 일곱 · Check · CheckWithRuntime
   apply.go       Preparer.Apply — host 패키지 · debootstrap · rootfs 봉인 · manifest 게시
   manifest.go    Manifest (prepared_environment_id = 정규화 투영의 sha256) · 읽기 · 쓰기
   record.go      Record — 단계 결과에 실리는 profile · 준비 산출물 · runtime 식별자
   exec.go        명령 실행 얇은 겹
   privilege_unix.go · privilege_windows.go         euid 0 이 아니면 sudo 를 붙인다
   runtime_driver_linux.go · runtime_driver_other.go   runc-overlay 를 이 빌드가 지원하는가
```

### internal/transcript — 파서 (신규)

```text
   transcript.go  Fields · Kind (닫힌 어휘 일곱) · Event · Result
   parse.go       Parse — NDJSON 을 사건 열로.  줄 단위 상한과 표시 상한
   shell.go       Shell — 봉인 로그의 껍데기 줄을 짓는다.  짓는 쪽과 읽는 쪽이 한 패키지다
   line.go        SplitLines · ParseLine · 아는 키 하나를 아는 모양으로만 읽는 String · Bool · Int
```

### internal/store — 상태 층

```text
   store.go     Store · Run · 상태 상수 · UpsertAdvert · CreateRun
   claim.go     ClaimStep · ReportStep · RenewLeases · FailRestarted.  StepResult 에 Environment
   observe.go   Steps · Nodes · Runs · Ledger (ADR-025 · ADR-065)
   queue.go     QUEUED — CreateQueuedRun · WakeQueued · WakeQueuedNow (ADR-064)
   reap.go      만료 회수 · Cancel · SettleIfDone · RunReaper · 고아 진행 트리 쓸기
   seal.go      봉인.  진행 트리를 먼저 걷는다
   ask.go       되묻기 — raiseAsks · AnswerStep · PendingAsks · ExpireAsks
   acquire.go · dispatch.go · expand.go · release.go · rollback.go · verdict.go
   schema.sql   임베드.  Migrate 가 통째로 exec 한다.  기준선 뒤로 안 바뀌었다
```

### internal/record — 파일시스템

```text
   record.go      Open · AppendLog(총 길이를 낸다) · Seal · Tar · blob
   progress.go    진행 파일 — <Root>/progress/run-<id>/ (기록 디렉터리의 형제).
                  AppendProgress · 시도가 바뀌면 앞 시도를 걷는다 · 총 길이 상한 표시 줄 ·
                  DropProgress · 고아 쓸기.  ProgressMaxAge 6시간
```

### 나머지

```text
   internal/contract/contract.go    1,799 줄.  계약 문법의 정본.  Workspace 에서 Rev 가 빠졌다
   internal/contract/grammar.go     기계용 문법 산출물
   internal/contract/advert.go      광고 어휘 · drain 상수 셋
   internal/contract/examples/      임베드된 예시 계약
   internal/match/match.go          Match — 2-pass all-or-nothing
   internal/schema/schema.go        form-only 검증
   internal/runctl/client.go        Mediator HTTP 클라이언트.  StepLog 가 신규
   internal/config/config.go        Config 와 기본값
   internal/config/write.go         토큰 · 설정 쓰기
   internal/proc/proc.go            PidFromLock
   internal/build/build.go          버전 한 줄
   internal/transcriptui/embed.go   card.mjs 를 embed.FS 로 든다.  문장 0
   scripts/overlay-probe.sh         overlay 가 되는 환경인지 잰다.  환경을 안 고친다
   scripts/nested-runc-overlay-probe.sh   중첩 user namespace 안에서 overlay 와 runc 를 함께 잰다
```

---

## 설계 패턴

### 조건부 래퍼 (조건부 등록이 아니라)

- **자리**: `internal/api/api.go` 의 `read` 함수.
- **목적**: 데모 모드에서 읽기 넷을 무인증 + 한도로 바꾼다.
- **구현**: 라우트 등록은 모드와 무관하게 같고 핸들러를 감싸는 것만 갈린다.

### 순수 함수 코어 + 부작용 가장자리

- **자리**: `internal/match` · `Harness.Argv` · `contract.Validate` · `transcript.Parse` ·
  `environment.Parse`.
- **목적**: 시험이 싸다. `Argv` 는 프로세스를 안 띄우고 `Match` 는 광고를 안 읽는다.
- **구현**: 부르는 쪽이 값을 모아 넘긴다.

### 단계의 수명을 세션 하나가 소유한다 (신규)

- **자리**: `internal/enode/runtime.go` 의 `StepRuntime` · `StepSession`.
- **목적**: agent 와 command 가 같은 실행 경계를 지나고, 명령이 끝난 뒤에도 수확이
  같은 namespace 에서 merged view 를 본다 (ADR-073).
- **구현**: `Open` → `Paths` → `Project`(agent 만) → `Run` → `Harvest` → `Close`.
  native 는 `exec.Cmd` 를 그대로 감싸고, runc-overlay 는 helper 에 요청을 보낸다.
  `manageSession` 이 `Close` 를 한 번으로 묶는다 — `defer` 와 명시 호출이 둘 다 부른다.

### 줄 단위 JSON 요청 · 응답 (신규)

- **자리**: `runc_overlay_linux.go` 의 `runtimeWireRequest` · `runtimeWireResponse`.
- **목적**: 부모 노드와 namespace 안 helper 가 파일 하나 없이 말한다. 부모가 죽으면
  `Pdeathsig` 와 `unshare --kill-child` 가 helper 와 runc 를 함께 거둔다.
- **구현**: `open` · `project` · `run` · `cancel` · `harvest` · `close` 여섯 요청.
  `run` 동안 helper 가 `stdout` · `stderr` 조각(32 KiB)을 흘리고 `run-done` 으로 닫는다.

### 읽는 명령과 고치는 명령을 가른다 (신규)

- **자리**: `enode env check` 와 `enode env apply`. 데몬 기동.
- **목적**: 기동은 환경을 고치지 않는다. 준비되지 않았으면 이유를 말하고 멈춘다.
- **구현**: `check` 는 사실만 재서 판정하고, `apply` 는 계획한 연산을 실행해 manifest 를
  게시한다. 데몬 기동과 `enodectl start` 가 `check` 를 다시 부른다.

### 불변 준비 산출물 (신규)

- **자리**: `internal/environment/apply.go` · `manifest.go`.
- **목적**: 같은 profile 은 같은 rootfs 를 가리키고, 단계 결과가 그것을 식별자로 남긴다.
- **구현**: rootfs 를 다 지은 뒤 `chmod -R a-w` 로 굳히고, 정규화한 투영의 sha256 을
  `prepared_environment_id` 로 쓴다. profile 이 바뀌면 `stale` 로 판정된다.

### 조건부 래퍼 · 허용목록 · 파일 통로 · 링 · 형제 exec

2026-09-15 판의 다섯 패턴은 그대로 산다. 바뀐 자리만 적는다.

```text
   허용목록            봉인 로그(selectLogs)는 허용목록에서 크기 절단으로 바뀌었다 (ADR-071).
                       환경변수와 MCP 서버는 여전히 허용목록이다
   파일이 통로          진행 파일이 하나 더 는다 — Mediator 쪽이고 기록 디렉터리의 형제다
   형제 exec           enodectl env -> enode env 가 는다.  enodectl.exe 심볼 상한을 지킨다
```

### 한 벌만 둔다

- **자리**: `internal/transcript`(파서) · `internal/transcriptui`(카드 렌더러).
- **목적**: 봉인 로그를 짓는 쪽과 읽는 쪽, 제어판과 현황판이 갈리지 않게 한다.
- **구현**: 짓는 `Shell` 과 읽는 `Parse` 가 한 패키지에 살고 왕복 시험이 잡는다.
  카드는 `embed.FS` 한 장을 두 서버가 각자 `http.FileServerFS` 로 낸다.

---

## 핵심 의존

```text
   github.com/jackc/pgx/v5  v5.10.0   internal/store 만 쓴다.  pool · tx · SKIP LOCKED
   golang.org/x/sys         v0.47.0   internal/enode 와 internal/proc 의 플랫폼 쌍 · mount 계열
   gopkg.in/yaml.v3         v3.0.1    설정 · 정책 · 상태 파일 · 실행 환경 profile
```

간접 일곱은 전부 pgx 와 테스트 도구의 것이다. **HTTP 라우터도 웹 프레임워크도
로깅 라이브러리도 컨테이너 라이브러리도 없다** — `net/http` · `log/slog` ·
`encoding/json` 이 표준 라이브러리로 그 자리를 지고, runc 는 실행파일로 부른다.
