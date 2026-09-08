# 유닛 정의와 책임 — Units Generation

AI-DLC Units Generation 산출물이다. **정본은 `requirements/` 팩**이고, 응용 설계
(`aidlc-docs/inception/application-design/`) 위에서 확정한 **여덟 유닛**의 정의·
책임·경계다. 계획과 답은 `plans/unit-of-work-plan.md` (Q1=A · Q2=C · Q3=A).
의존과 파일 행렬은 `unit-of-work-dependency.md`, 게이트·기능 매핑은
`unit-of-work-story-map.md` 가 진다. 값은 팩·응용 설계를 가리키고 다시 적지 않는다.

메서드 안의 비즈니스 로직은 **유닛별 Functional Design** 몫이다 — 이 문서는
책임과 겉면까지다.

---

## 0. 확정 분해 (한눈)

```text
   obs · queue · drain · ui · panel · transcript · mcp · demo-back

   새 패키지 넷   internal/proc(panel) · internal/panel(panel) ·
                 internal/api/ui(ui) · internal/mcp(mcp)
   접점 넷        internal/store · internal/api/api.go · internal/panel ·
                 cmd/mediator/main.go (진행자 직렬 병합)
```

Q2=C 로 실 함대 현황판과 공개 데모 대시보드가 **ui 한 유닛의 두 모드**다.
Q1=A 로 되묻기는 독립 유닛이 아니라 표시(ui)·도구(mcp)로 접혔다. Q3=A 로 store
스키마는 기능별로 흩되 store 를 접점으로 진행자가 직렬 병합한다.

---

## 1. obs — 관측 API

**목적** — 함대와 Run 을 보이는 읽기 표면. 현황판·제어판·MCP 세 표면이 전부
이 위에 서는 토대다 (임계 경로).

**책임**
- `GET /v1/nodes`(getNodes) · `GET /v1/runs` 목록(getRuns) 신규. 기존 15 라우트 보존
- `GET /v1/runs/{id}` 확장 — `requires`·`as`(ADR-069) · `steps[].chosen`(ADR-060)
- store 읽기 — `Nodes` · `Runs` · `StepView.chosen` select · getRun requires·as ·
  `submitter` 컬럼(additive · Q3=A 로 읽기 경로가 진다)
- `runctl.Client` 에 `Nodes`·`Runs` — 원문 JSON 반환(panel·mcp 공유 · MCP 글자 일치)

**파일** — 신규 `internal/api/nodes.go` · `internal/api/runs.go`. 만지는
`internal/store/store.go`(읽기경로) · `internal/api/api.go`(등록 줄+getRun 확장) ·
`internal/runctl/client.go`.

**겉면**
```go
func (s *Store) Nodes(ctx) ([]NodeView, error)
func (s *Store) Runs(ctx, RunFilter) ([]RunRow, error)
func (c *Client) Nodes(ctx) (json.RawMessage, error)
func (c *Client) Runs(ctx, RunsQuery) (json.RawMessage, error)
```

**지는 기능** 3.1.1(관측 API 부분). **재는 게이트** CP1 (+CP0 회귀).

**의존** 없음 (토대).

**완료 핵심** — `GET /v1/nodes`·`/v1/runs` 가 실데이터 JSON(lease·draining 포함) ·
기존 15 라우트 회귀 · getRun 에 requires·as·chosen. 커버리지는 기존 `internal/api`.

---

## 2. queue — 대기열

**목적** — 점유 실패를 죽지 않고 `QUEUED` 로 받고, 자원이 풀리면 승격한다.

**책임**
- `runs.state` 에 `QUEUED` 추가 · `CreateQueuedRun`(매처 거절·ErrNodeTaken 롤백 두 자리)
- `WakeQueued(ctx, tx)` — 임대 지워지는 여섯 지점 뒤 · **부르는 tx 안 동기**(CP2) ·
  `WakeQueuedNow` 기동 복구(cmd/mediator/main.go 한 번)
- `DrainingNodes` + submit `busy` 합침 — **`if !dry` 안**(api.go:389~395)
- submit 응답에 `202 Accepted`(QUEUED) 분기

**파일** — 만지는 `internal/store/store.go`(QUEUED·CreateQueuedRun·WakeQueued·
WakeQueuedNow·DrainingNodes) · `internal/api/api.go`(submit 분기) ·
`cmd/mediator/main.go`(기동 wake). `internal/match` 시그니처 무변경(호출자가 합친다).

**겉면**
```go
func (s *Store) WakeQueued(ctx, tx pgx.Tx) ([]string, error)
func (s *Store) WakeQueuedNow(ctx) ([]string, error)
func (s *Store) CreateQueuedRun(ctx, tx, contract.Contract, submitter string) error
func (s *Store) DrainingNodes(ctx, tx) (map[string]bool, error)
```

**지는 기능** 3.2.1. **재는 게이트** CP2.

**의존** obs (CP2 는 GET /v1/runs 로 QUEUED 를 본다 · store 접점).

**완료 핵심** — 둘째 제출 `202`·`QUEUED` · a 끝나면 b 가 같은 tx 에서 `RUNNING` ·
함대에 없으면 `422`·`FAILED` · 기동 복구. FIFO 전체 훑기(Reap·drain 해제 대응).

---

## 3. drain — 소유자 자원 회수

**목적** — 소유자가 자기 노드를 돌려받는다. 중앙 라우트 신설 없음, 경로는 광고뿐.

**책임**
- enode 가 광고 직전에 정책 파일을 읽어 광고에 싣는다 · UpsertAdvert 가
  `nodes.draining` 에 복사(열은 이미 있음) · 광고 응답으로 되돌림
- 두 모드 — `graceful`(새 임대만 막음) · `at-boundary`(경계에서 닫음)
- at-boundary — postResult 끝에서 `store.Cancel(run, "drain:<node_id>")` +
  같은 tx 에서 `WakeQueued`
- 해제 — 소유자 명시. 자동 복귀 없음

**파일** — 만지는 `internal/enode`(광고 경로·정책 읽기 — transcript 와 다른 파일) ·
Mediator postResult(`internal/api` 또는 `internal/store` 취소 경로). 정책 파일
위치·형식·enum 과 at-boundary tx 경계는 **FD**(decisions §1).

**지는 기능** 3.2.2. **재는 게이트** CP3.

**의존** queue (at-boundary·해제가 WakeQueued 를 부른다).

**완료 핵심** — at-boundary 로 놓인 Run 이 `FAILED`·verdict `drain:<node_id>` ·
끝난 단계 산출은 Record 에 남음 · 해제 후 대기 Run 이 `RUNNING` · dry-run 은 draining 무시.

---

## 4. ui — 현황판 UI (실 함대 모드 + 공개 데모 모드)

**목적** — 함대와 Run 을 브라우저로 보인다. **한 표면의 두 모드** — 실 함대
(토큰·읽기 전용)와 공개 데모(게스트·쓰기). Q2=C 로 board-ui 와 demo-front 를
이 한 유닛이 진다.

**책임**
- 실 함대 모드 — S0(토큰)·S0b(401)·S1(격자+Run 목록)·S1b(폴링 끊김)·S2(작업 그래프).
  브라우저가 `GET /v1/nodes`·`/v1/runs` 를 부른다. **읽기 전용**(submit·drain 없음)
- 공개 데모 모드 — Guest 로그인(랜덤 2단어 이름 = submitter) · 3D 함대뷰(S6) ·
  3D 작업 그래프(S7) · 새 작업 모달(고정 시나리오 둘 · 이름 칸 없음) · 웹캠 좌하단
  floating resize · 가이드투어 4스텝(웹캠·노드·목록·새작업) · 작업 그래프 전환은
  run 목록을 안 가림 · RUN 카드에 submitter 표시. 제출은 demo-back 라우트를 부른다
- 되묻기 카드 표시 — S1 여섯째 상태(사람을 기다림 · 카운트다운 not_after 아님 ·
  answerers·칠 명령). 중앙은 답을 안 받는다
- `cmd/mediator` 가 `/ui/` 아래 이 embed FS 를 마운트

**파일** — 신규 `internal/api/ui`(embed 자산 + `Handler()`). 만지는
`cmd/mediator/main.go`(/ui/ 마운트). **`internal/api/ui` 를 단독 소유**한다(Q2=C).

**겉면**
```go
//go:embed assets
func Handler() http.Handler   // 정적. store 를 안 부른다
```

**임포트 경계** `internal/api/ui → internal/store` 금지.

**지는 기능** 3.1.1(화면) · 3.2.3(표시) · 3.4.1(투어) · 3.4.2 · 3.4.3(모달·전환) ·
3.4.4(카드 표시) · 3.4.5. **재는 게이트** CP1·CP2·CP7(표시)·CP8·CP9·CP11 화면.

**의존** obs(부르는 라우트 · 착수) · demo-back(데모 제출 라우트 · 완료).

**완료 핵심** — 화면 조작을 전부 나열한다(§2.1·§5.3): 실 모드 S0·S0b·S1(카드 상태·
QUEUED 행·chosen)·S1b·S2 · 데모 모드 게스트 로그인·투어 4스텝·모달(시나리오 둘)·
3D 전환·웹캠 resize·RUN 카드 submitter. 커버리지 80%(`internal/api/ui`).

**소비·보류** — 3D 자산(S6·S7)은 shonsin pen(D2·D5) 소비. 게스트 이름 랜덤 2단어화·
RUN 카드 submitter 표시는 pen 정합(shonsin) 뒤 반영. 온보딩 카드 시퀀스는 이 유닛
밖(외부 브랜치). sandbox 표시 자리는 두되 출처는 FD(§8.5).

---

## 5. panel — 호스트 제어판

**목적** — 노드 소유자가 자기 기계에서 자기 노드 하나를 보고 통제한다.

**책임**
- `internal/proc` 추출 — `processAlive`·`signalStop`·`ownsConfig` 를
  `cmd/enodectl/proc_*.go` 에서 내려 빌드 태그 짝으로. `net/http` 안 씀
- `internal/panel` 서버 — `127.0.0.1:8081` 기본 · 신원 표시 · 탐지 능력 읽기 전용 ·
  현재 작업(runctl.Client 조회) · drain 토글(정책 파일 쓰기) · 프로세스 제어
  (status·start·stop·logs · stop 은 cancel 먼저) · Mediator 마지막 응답 시각
- `cmd/enodectl serve <name>` — enode 제어판 프로세스를 exec 위임(심볼 상한 회피)
- `cmd/enode` 제어판 하위명령 — internal/panel 을 net/http 로 띄운다
- **임포트 경계 검사 테스트를 낸다**(internal/panel 을 처음 만든다)

**파일** — 신규 `internal/proc/*` · `internal/panel/*` · 경계 검사 테스트. 만지는
`cmd/enodectl` · `cmd/enode`.

**겉면**
```go
type Config struct { Node, Listen, MediatorBase, Token, PanelToken string; ... }
func New(cfg Config) (*Server, error)   // LAN 인데 PanelToken 비면 error
func (s *Server) Handler() http.Handler
// proc: processAlive(pid) · signalStop(pid) · ownsConfig(path) (플랫폼 짝)
```

**임포트 경계** `panel → store` 금지 · `panel → api` 금지 · `enode → panel` 금지
(`cmd/enode → panel` 허용). 심볼 상한 — `enodectl.exe` net/http ≤50 · crypto/tls ≤10
(serve 는 exec 위임 · proc 는 net/http 없음).

**지는 기능** 3.1.2. **재는 게이트** CP4.

**의존** obs(runctl.Client.Nodes · 착수) · drain(제어판이 쓰는 정책 파일 형식 · 완료).

**완료 핵심** — 조작 넷(status·start·stop·logs) + drain 걸기·모드(graceful·
at-boundary)·풀기 전부 나열 · S3·S3b·S5 · stop 은 cancel 먼저(verdict 에 cancelled by) ·
경계 검사 테스트 초록 · 심볼 상한 재측정 · 커버리지 80%(`internal/panel`).

**열린 미정** — 제어판이 데몬 `Capabilities{Caps, At}` 를 읽는 계약(ADR-068). 진행자가
`decisions.md` 에 행을 더해 이 유닛 FD 전에 닫는다. 겉면에 자리만 둔다.

---

## 6. transcript — 하네스 트랜스크립트

**목적** — 하네스가 지금 뱉는 글자와 지난 작업의 결과·봉인 기록을 제어판에 보인다.
DB 를 안 만진다.

**책임**
- enode 가 하네스 stdout/stderr 를 노드의 고정 크기 링 파일에 tee — 에이전트 단계
  (runner.go)·명령 단계(claim.go) 둘 다(io.MultiWriter)
- 제어판이 링 파일을 1초 폴링해 카드에 그린다(`GET /api/transcript?node=<이름>`).
  데몬 로그 카드와 나란히(다른 물건)
- 지난 작업 — `GET /v1/runs` 를 자기 `node_id` 로 걸러 목록 · 누르면
  `verdict.checks` + `GET /v1/runs/{id}/record` tar 의 `logs/NN-*.log`

**파일** — 만지는 `internal/enode/runner.go`·`claim.go`(링 tee — drain 과 다른 파일) ·
`internal/panel`(트랜스크립트 카드 — panel 생성 뒤 접점). 링 파일 로직(머리·몸통·
감김·비우기)은 **FD**(decisions §6.3).

**지는 기능** 3.1.3. **재는 게이트** CP6.

**의존** panel(카드가 사는 화면) · obs(GET /v1/runs).

**완료 핵심** — 도는 것(단계 끝나기 전 흐름 · 상한 넘겨도 앞부터 밀림 · 다음 단계
첫 글자에 갈림) · 지난 것(결과 + 봉인 트랜스크립트) · 데몬 로그 카드와 다른 물건임이
화면에 보임 · 커버리지 80%(`internal/enode`).

---

## 7. mcp — Mediator MCP 어댑터

**목적** — 사용자의 Claude 가 stdio 로 함대에 붙는다. 기존 REST 를 도구로 감싼다.
새 의미 0.

**책임**
- `internal/mcp` — stdio JSON-RPC 2.0 루프(initialize·tools/list·tools/call)
- `runctl mcp` 하위명령이 이 패키지를 띄운다. 새 실행파일 0(packaging 무변경)
- 도구 열 — capabilities.list · fleet.list · runs.list · run.submit · run.plan ·
  run.cancel · record.get · run.get · asks.list · run.answer
- fleet.list·runs.list 는 `Client.Nodes`·`Runs` 원문 passthrough(글자 일치 · CP5)
- 토큰은 환경변수(ENODE_MEDIATOR·ENODE_TOKEN). 도구 인자로 안 받음. 새 Go 의존 0

**파일** — 신규 `internal/mcp/*`. 만지는 `cmd/runctl/main.go`(mcp 하위명령).

**겉면**
```go
func New(client *runctl.Client) *Server
func (s *Server) Serve(ctx, in io.Reader, out io.Writer) error
```

**지는 기능** 3.3.1 (+3.2.3 도구 둘 asks.list·run.answer). **재는 게이트** CP5
(+CP7 도구 부분).

**의존** obs(runctl.Client.Nodes·Runs 와 그 라우트).

**완료 핵심** — `tools/list` 가 열 · `run.submit` 낸 Run 이 현황판에 그대로 ·
`fleet.list` = `GET /v1/nodes` 글자까지 같음 · `go.mod`·`packaging/` 무변경 ·
asks.list·run.answer 글자 일치.

---

## 8. demo-back — 데모 제출 백엔드

**목적** — 공개·무인증 게스트가 고정 시나리오만 안전하게 낸다. SECURITY-08 수락
위험을 가둠 셋으로 닫는다.

**책임**
- `internal/api/demo.go`(postDemo) — 본문 시나리오 id 가 allow-list(고정 시나리오
  이름 집합)에 있나 · 없으면 거부(임의 계약 불가)
- 서버측 Mediator 토큰 주입으로 내부 submit — 브라우저에 실 토큰 없음
- `submitter`(Guest 로그인 이름 · 표시 라벨) 를 받아 CreateRun/CreateQueuedRun 에
  넘긴다(store 쓰기 · Q3=A) · 스텝 주입
- 고정 시나리오 픽스처는 `internal/contract` 의 example(decisions §8.3)
- 시나리오 둘 — ① rpi LED 토글 ② mac wav → rpi 스피커(cross-node 봉인 blob)

**파일** — 신규 `internal/api/demo.go`. 만지는 `internal/api/api.go`(등록 줄) ·
`internal/store`(submitter 쓰기).

**겉면**
```go
func (s *Server) postDemo(w http.ResponseWriter, r *http.Request)
```

**지는 기능** 3.4.3(서버측) · 3.4.4(submitter 쓰기). **재는 게이트** CP9 서버측.

**의존** obs(submitter 컬럼) · queue(submit → 202/QUEUED 경로).

**완료 핵심** — 임의 계약 거부(allow-list) · submitter 가 `GET /v1/runs` 에 노출 ·
가둠 셋 보존(일회용 데모 Mediator · 서버측 allow-list · 브라우저 실 토큰 없음) ·
커버리지. 데모 라우트를 데모 모드(config)에서만 등록하는 것은 FD 권장.

---

## 9. 패키지 배정 (한눈)

```text
   새 패키지        만드는 유닛
     internal/proc      panel
     internal/panel     panel (transcript 가 카드를 더함 — 접점)
     internal/api/ui    ui (단독)
     internal/mcp       mcp

   기존 패키지 확장  만지는 유닛
     internal/store     obs(읽기) · queue(QUEUED·wake·draining) · demo-back(submitter) — 접점
     internal/api       obs(nodes·runs·getRun) · queue(submit) · demo-back(demo.go) — api.go 접점
     internal/enode     drain(광고·정책) · transcript(링 tee) — 다른 파일
     internal/runctl    obs(Client.Nodes·Runs)
     cmd/enodectl       panel(serve)
     cmd/enode          panel(제어판 하위명령)
     cmd/runctl         mcp(mcp 하위명령)
     cmd/mediator       queue(기동 wake) · ui(/ui/ 마운트) — 접점
     internal/match     무변경 (호출자가 busy 합침)
```

## 10. 열린 미정 (해당 유닛 FD 전에 닫힌다)

```text
   1  Capabilities{Caps,At} 읽기 계약 (ADR-068)   panel 유닛.  진행자 decisions
   2  sandbox 표시 출처 (decisions §8.5)          ui 유닛.  있는 값을 읽는다 · FD
```
