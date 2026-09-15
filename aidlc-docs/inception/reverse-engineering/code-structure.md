# 코드 구조 — 오늘의 파일

**2026-09-15 전면 재측정.** 파일 목록은 고칠 후보의 목록이다 — 브라운필드
회차가 이 표에서 자기 파일 행렬을 뽑는다.

---

## 빌드 시스템

```text
   모듈        github.com/taeels/enode
   Go          go 1.26 · toolchain go1.26.6.  둘을 같은 값으로 묶어 슬랙을 없앴다
   빌드        go build ./...   별도 빌드 도구가 없다
   테스트      go test ./... -count=1 -coverpkg=./...  (커버리지 계약이 명령을 고정한다)
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
   internal/api             HTTP 표면 ------- internal/api/ui (정적.  store 를 안 문다)
     |                                    \
   internal/store  ---- internal/match (순수) \
     |                  internal/contract ---- internal/schema
   internal/record

   internal/enode           노드 로직.  internal/contract · internal/build 만 문다
     |
   internal/panel  ---- internal/runctl · internal/proc
```

**두 나무가 안 만난다.** Mediator 쪽(`api` · `store` · `record`)과 노드 쪽
(`enode` · `panel` · `proc`)은 `contract` 를 공유할 뿐 서로를 임포트하지 않는다.
경계 검사 테스트가 그것을 센다 (`internal/panel/boundary_test.go`).

---

## 파일 목록 — 고칠 후보

### cmd/

```text
   cmd/mediator/main.go        배선 · reaper 고루틴 · serve/shutdown
   cmd/mediator/setup.go       DB role/database 대화형 프로비저닝
   cmd/enode/main.go           데몬.  하위명령 셋을 앞에서 가로챈다
   cmd/enode/hook.go           enode hook — 하네스 훅 진입점
   cmd/enode/setup.go          enode setup
   cmd/enode/panel.go          enode panel — 제어판을 띄운다
   cmd/enodectl/main.go        하위명령 디스패치 · 라이브니스
   cmd/enodectl/serve.go       형제 enode panel 을 exec 한다
   cmd/enodectl/setup.go       형제 enode setup 을 exec 한다
   cmd/enodectl/proc_*.go      플랫폼 쌍
   cmd/enodectl/caffeinate_*.go  darwin 에서 잠들기를 막는다
   cmd/runctl/main.go          하위명령 열하나 · 종료코드 0-3
   cmd/runctl/shape.go         계약 문법과 팩 모양을 사람에게 설명한다
   cmd/iapadapter/*.go         브리지 일곱 — main · config · contract · itsaplan ·
                               mediator · orchestrator · comment
```

### internal/api — Mediator 표면

```text
   api.go                     라우팅 + 핵심 핸들러 열일곱.  1,200 줄
   nodes.go                   GET /v1/nodes (ADR-065)
   runs.go                    GET /v1/runs + 질의 인자 넷
   ratelimit.go               데모 모드의 전역 토큰버킷
   demo.go                    POST /v1/demo/runs — 고정 시나리오
   demo_gallery.go            갤러리 라우트 여섯
   demo_gallery_history.go    갤러리 이력
```

### internal/api/ui — 화면

```text
   ui.go                          임베드 + 보안 헤더 + webcam origin 검증
   static/index.html · landing.*  랜딩
   static/cardnews/               온보딩 카드뉴스 (이미지 넷 · 영상 하나)
   static/demo/                   데모 번들 — 제출 · 갤러리 · 투어 · 웹캠
   static/fleet/                  함대 현황판 진입점 (app.mjs · login.mjs)
   static/shared/fleet/           현황판 본체 — client · model · view · scene ·
                                  format · identity · fleet.css
   tests/*.test.mjs               브라우저 테스트 열넷
```

### internal/panel — 노드 제어판

```text
   panel.go       Config · Server · New · Handler · 토큰 미들웨어 · loopback 판정
   handlers.go    state · drain · undrain · stop · start · logs
   transcript.go  transcript(링) · runs(Mediator 목록 거르기) · record(tar 풀기)
   view.go        State 와 그 다섯 묶음.  각 묶음이 독립 실패한다
   page.go        화면 한 장.  시안 design/enode-ux.pen 의 다크 토큰
```

### internal/enode — 노드 로직

```text
   claim.go        Worker 루프 · 단계 실행 · 업로드.  가장 큰 파일
   runner.go       유일한 exec 관문 · 계장 · logs/ 선별(selectLogs)
   claude.go       하네스 어댑터 — Argv · Decode · Instrument · Usable · Version
   harness.go      Reason · HarnessResult · ParseClaude · Event · 레지스트리
   transcript.go   고정 크기 링 파일 — OpenRing · Write · Reset · ReadRing
   mcp.go          MCP 허용목록 · 팩 tar 읽기 · 노드 선언
   policy.go       소유자 정책 파일 (drain · panel_token)
   status.go       상태 파일 (탐지 능력과 그 시각)
   advertise.go    광고 = 하트비트 = 갱신
   detect.go · detector.go   능력 탐지와 그 시계
   workspace.go · diff.go · changed.go · collect.go   워크스페이스와 산출물
   env.go          환경 화이트리스트 (R1)
   hook.go         훅 설정 쓰기
   identity.go · config.go · paths.go · repoid.go     신원과 경로
   leases.go · lock.go · lock_*.go                    임대와 단일 인스턴스
   agent.go · argv.go · child.go · setup.go
   console_*.go · disk_*.go                           플랫폼 쌍
```

### internal/store — 상태 층

```text
   store.go     Store · Run · 상태 상수 · UpsertAdvert · CreateRun
   claim.go     ClaimStep · ReportStep · RenewLeases · FailRestarted
   observe.go   Steps · Nodes · Runs · Ledger (ADR-025 · ADR-065)
   queue.go     QUEUED — CreateQueuedRun · WakeQueued · WakeQueuedNow (ADR-064)
   reap.go      만료 회수 · Cancel · SettleIfDone · RunReaper
   ask.go       되묻기 — raiseAsks · AnswerStep · PendingAsks · ExpireAsks
   acquire.go · dispatch.go · expand.go · release.go · rollback.go
   seal.go · verdict.go
   schema.sql   임베드.  Migrate 가 통째로 exec 한다
```

### 나머지

```text
   internal/contract/contract.go    1,797 줄.  계약 문법의 정본
   internal/contract/grammar.go     기계용 문법 산출물
   internal/contract/advert.go      광고 어휘 · drain 상수 셋
   internal/contract/examples/      임베드된 예시 계약
   internal/record/record.go        Open · AppendLog · Seal · Tar · blob
   internal/match/match.go          Match — 2-pass all-or-nothing
   internal/schema/schema.go        form-only 검증
   internal/runctl/client.go        Mediator HTTP 클라이언트
   internal/config/config.go        Config 와 기본값
   internal/config/write.go         토큰 · 설정 쓰기
   internal/proc/proc.go            PidFromLock
   internal/build/build.go          버전 한 줄
```

---

## 설계 패턴

### 조건부 래퍼 (조건부 등록이 아니라)

- **자리**: `internal/api/api.go` 의 `read` 함수.
- **목적**: 데모 모드에서 읽기 셋을 무인증 + 한도로 바꾼다.
- **구현**: 라우트 등록은 모드와 무관하게 같고 핸들러를 감싸는 것만 갈린다. 그래서
  라우트 표의 개수와 패턴이 안 변하고 ServeMux 의 중복 등록 패닉이 생길 자리가 없다.

### 순수 함수 코어 + 부작용 가장자리

- **자리**: `internal/match` · `Harness.Argv` · `contract.Validate`.
- **목적**: 시험이 싸다. `Argv` 는 프로세스를 안 띄우고 `Match` 는 광고를 안 읽는다.
- **구현**: 부르는 쪽이 값을 모아 넘긴다.

### 단일 exec 관문

- **자리**: `internal/enode/runner.go` 의 `runHarness`.
- **목적**: 환경 화이트리스트(R1)와 격리를 어댑터 수와 무관하게 한 번만 지킨다.
- **구현**: 어댑터는 순수 함수만 내놓고 실행하지 않는다. 치명 검사가 전부 exec 앞에 모인다.

### 허용목록 (지우는 쪽이 아니라 남기는 쪽)

- **자리**: `env.go` 의 환경변수 · `runner.go` 의 `selectLogs` · `mcp.go` 의 서버 목록.
- **목적**: 하네스가 필드를 늘려도 안 샌다. 지우는 쪽은 열리는 쪽으로 틀린다.
- **구현**: 원본에서 빼지 않고 새 객체를 짓는다.

### 파일이 프로세스 사이의 통로

- **자리**: 정책 파일 · 상태 파일 · 트랜스크립트 링 · 잠금 파일.
- **목적**: 제어판과 데몬은 다른 프로세스이고 데몬은 나가는 클라이언트만 있다.
- **구현**: 설정 파일 경로가 신원이므로 그 옆의 이름 규칙으로 자리가 정해진다
  (`<stem>.policy` · `<stem>.status` · `<stem>.transcript` · `<config>.lock`).

### 링 파일 (회전 없는 고정 크기)

- **자리**: `internal/enode/transcript.go`.
- **목적**: 윈도우가 `FILE_SHARE_DELETE` 를 안 줘서 읽는 쪽이 열고 있으면 rename ·
  삭제가 막히고, `Truncate` 와 읽기가 겹치면 읽는 쪽이 깨진 것을 본다.
- **구현**: 회전 · 자르기 · 삭제를 아예 안 하고 `WriteAt` 만 쓴다. 머리를 마지막에
  갱신해 반쯤 쓰인 꼬리가 화면에 안 나온다. **빌드 태그 쌍이 하나도 안 는다.**

### 형제 바이너리 exec (링크가 아니라)

- **자리**: `enodectl setup` -> `enode setup` · `enodectl serve` -> `enode panel`.
- **목적**: `enodectl.exe` 에서 `crypto/tls` 와 `net/http` 를 뺀다.
- **구현**: `os.Executable` 의 디렉터리에서 형제를 찾아 exec 한다. CI 가 심볼 상한으로
  재링크를 막는다 (avprobe 사건).

---

## 핵심 의존

```text
   github.com/jackc/pgx/v5  v5.10.0   internal/store 만 쓴다.  pool · tx · SKIP LOCKED
   golang.org/x/sys         v0.47.0   internal/enode 와 internal/proc 의 플랫폼 쌍
   gopkg.in/yaml.v3         v3.0.1    설정 · 정책 · 상태 파일
```

간접 일곱은 전부 pgx 와 테스트 도구의 것이다. **HTTP 라우터도 웹 프레임워크도
로깅 라이브러리도 없다** — `net/http` · `log/slog` · `encoding/json` 이 표준
라이브러리로 그 자리를 진다.
