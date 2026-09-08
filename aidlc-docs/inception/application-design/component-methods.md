# 컴포넌트 메서드 — 시그니처와 겉면

AI-DLC Application Design 산출물이다. **여기는 겉면(시그니처·입출력·책임)까지다.
메서드 안의 비즈니스 로직은 유닛별 Functional Design 이 정한다.** 시그니처는
Go 표기로 적되 인자·반환의 세부 타입은 FD 가 굳힐 수 있다 — 이 문서는 계약의
모양을 정한다.

기존 코드 사실은 `aidlc-docs/inception/reverse-engineering/api-documentation.md`
에서 온다. 결정은 `application-design-plan.md` 4절 (Q1~Q5 = A).

---

## 1. internal/panel

```go
type Config struct {
    Node         string // enodectl serve <name> 의 그 이름 (유일 식별자)
    Listen       string // 기본 "127.0.0.1:8081". LAN 은 명시
    MediatorBase string // Mediator HTTP 주소
    Token        string // Mediator 토큰 (현재 작업·지난 작업 조회용)
    PanelToken   string // LAN 노출 시 필수. 루프백이면 빈 값 허용
    // 정책 파일 경로 · 링 파일 경로는 FD 가 굳힌다 (decisions §1·§6.3)
}

type Server struct { /* runctl.Client · proc 원장 · 설정 경로를 쥔다 */ }

func New(cfg Config) (*Server, error)   // LAN 인데 PanelToken 이 비면 error
func (s *Server) Handler() http.Handler // /api/* + 제어판 정적 자산
```

핸들러 겉면 (경로·동사는 FD 가 굳힌다 · 여기는 책임까지):

```text
   신원 · 능력 표시     node_id · label · principal · instance + 탐지 능력·잰 시각
                       (능력 출처는 열린 미정 — components.md §6)
   현재 작업           runctl.Client 로 GET /v1/nodes 자기 행의 lease ->
                       그 run_id 로 Status.  데몬 로그를 안 읽는다 (decisions §2)
   drain 토글          정책 파일 쓰기 — 걸기 · 모드(graceful|at-boundary) · 풀기
   트랜스크립트         링 파일 1초 폴링.  지난 작업은 GET /v1/runs 를 node_id 로
                       거르고 GET /v1/runs/{id}/record 의 tar 에서 로그를 꺼낸다
   프로세스 제어        proc 원장으로 status · start · stop · logs.
                       stop 은 POST /v1/runs/{id}/cancel 을 먼저 부른다 (decisions §7.2)
```

**임포트**: `internal/runctl` · `internal/proc`. `store` · `api` 안 함.

---

## 2. internal/mcp

```go
type Server struct { client *runctl.Client } // Q2 = A: 재사용

func New(client *runctl.Client) *Server
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error
// stdio JSON-RPC 2.0 루프. initialize · tools/list · tools/call 을 처리한다
```

도구 디스패치 표 — 열 개 전부 기존 REST 를 감싼다 (`decisions §5.3·§7.3`).
새 의미 0 (`ADR-022 §9.3`).

```text
   도구             REST                        내려가는 방법
   capabilities.list  GET /v1/capabilities       Client.Capabilities (타입)
   fleet.list         GET /v1/nodes              Client.Nodes  (원문 그대로 — Q3)
   runs.list          GET /v1/runs               Client.Runs   (원문 그대로 — Q3)
   run.submit         POST /v1/runs              Client.Submit(dry=false)
   run.plan           POST /v1/runs/dry-run      Client.Submit(dry=true)
   run.cancel         POST /v1/runs/{id}/cancel  Client.Cancel
   record.get         GET /v1/runs/{id}/record   Client.Record
   run.get            GET /v1/runs/{id}          Client.Status
   asks.list          GET /v1/asks               Client.Asks
   run.answer         POST .../steps/{seq}/answer Client.Answer
```

**글자 일치 (Q3 = A)**: `fleet.list` · `runs.list` 는 `Client.Nodes` ·
`Client.Runs` 가 돌려준 응답 본문(원문 JSON)을 그대로 도구 결과로 실어 보낸다.
디코드·재직렬화가 없어 `GET /v1/nodes` 와 글자까지 같다 (CP5).

**임포트**: `internal/runctl`. 새 Go 의존 0 — 프로토콜을 손으로 짠다.

---

## 3. internal/api/ui

```go
//go:embed assets
var assets embed.FS

func Handler() http.Handler // http.FileServer(fs.Sub(assets, "assets"))
```

서버 로직·메서드가 없다. `store` 를 안 부른다. `cmd/mediator` 가 `/ui/` 아래
마운트한다.

---

## 4. internal/proc (Q1 = A)

빌드 태그 짝으로 갈린 원장. 오늘 `cmd/enodectl/proc_*.go` 에 있는 것을 내린다.
`net/http` 를 안 문다.

```go
// proc_unix.go (!windows) / proc_windows.go (windows) — 짝
func processAlive(pid int) bool
func signalStop(pid int) error
func ownsConfig(path string) (bool, error)
// (현재 cmd/enodectl 시그니처를 그대로 옮긴다 — FD 가 export 여부·이름을 굳힌다)
```

`cmd/enodectl` 과 `internal/panel` 이 임포트한다.

---

## 5. internal/runctl.Client (Q2 = A)

기존 `Client{Base, Token, Principal, HTTP}` 에 두 메서드를 더한다. 나머지
(`Submit` · `Status` · `Cancel` · `Record` · `Capabilities` · `Asks` ·
`Answer`)는 그대로 재사용한다.

```go
// 원문 응답 본문을 그대로 돌려준다 — MCP 글자 일치(Q3)를 이 자리에서 보장.
// panel 은 필요하면 로컬에서 json.Unmarshal 한다 (typed 접근을 클라이언트에
// 새로 안 만든다 — 얇게).
func (c *Client) Nodes(ctx context.Context) (json.RawMessage, error)
func (c *Client) Runs(ctx context.Context, q RunsQuery) (json.RawMessage, error)

type RunsQuery struct {
    State string // runs.state
    Since string // runs.created_at 하한
    Work  string // runs.work_id 일치
    Limit int    // 기본 100
}
```

`do(ctx, method, path, body)` 를 그대로 지난다 — `Authorization: Bearer` ·
`X-Enode-Principal` 이 붙는다 (`client.go:64~65`).

---

## 6. internal/store

### 6.1 WakeQueued (Q4 = A)

```go
// QUEUED 를 FIFO(created_at 오름차순)로 훑어 지금 매칭되는 것을 RUNNING 으로
// 올리고, 올린 run_id 들을 돌려준다. 호출자 트랜잭션에서 동기로 돈다 (CP2).
func (s *Store) WakeQueued(ctx context.Context, tx pgx.Tx) ([]string, error)

// 기동 시 크래시 복구용 — 자기 트랜잭션을 열어 WakeQueued 를 부른다
// (decisions §2 · ADR-064 §6). cmd/mediator/main.go 가 한 번 부른다.
func (s *Store) WakeQueuedNow(ctx context.Context) ([]string, error)
```

여섯 호출 지점 (임대가 지워지는 자리 · `decisions §1`) — `services.md` 3절이
오케스트레이션으로 적는다.

### 6.2 QUEUED 생성·전이

```go
// 점유 실패에서 QUEUED 행을 남긴다. 매처 거절(api.go:398~408)과
// ErrNodeTaken 롤백(api.go:457~463) 두 자리가 부른다 — 뒤 자리는 오늘 행조차
// 안 만든다. requires 는 ADR-069 표시를 위해 함께 적는다.
func (s *Store) CreateQueuedRun(ctx context.Context, tx pgx.Tx, c contract.Contract, submitter string) error
```

`runs.state` 상수에 `QUEUED` 를 더한다 (`store.go:156~163`). `RESOLVING`·
`ALLOCATING` 은 여전히 죽은 상수다 — `QUEUED` 가 그 자리에 실제로 저장되는 첫
중간 상태다.

### 6.3 관측 스냅숏 · 목록 · 표시 필드

```go
// GET /v1/nodes 재료 (ADR-065 모양). expires_at > now() 만. draining · lease 포함.
func (s *Store) Nodes(ctx context.Context) ([]NodeView, error)

// GET /v1/runs 목록 재료. 필터 넷 · created_at 역순 · limit 기본 100.
func (s *Store) Runs(ctx context.Context, q RunFilter) ([]RunRow, error)

// 매칭에서 draining 노드를 빼는 재료. api submit 이 busy 에 합친다 (if !dry).
func (s *Store) DrainingNodes(ctx context.Context, tx pgx.Tx) (map[string]bool, error)
```

- `runs` 에 `submitter text` 를 additive 로 더한다 (`ALTER ... ADD COLUMN IF NOT
  EXISTS` · `decisions §8.2 Q3`). `Runs` 목록 행과 스텝 주입이 읽는다
- `StepView` 에 `chosen bool` 을 더해 select 한다 (`ADR-060` — DB 엔 있고 뷰에
  없다). `Steps`/`StepView`/`StepFiles` 의 select 목록을 넓힌다
- getRun 의 `requires` · `as` — `describeWant`(verdict.go:47)가 이미 함대 조건을
  사람이 읽게 렌더한다. 그것을 봉인 시점이 아니라 조회 시점에도 내도록 넓힌다
  (`ADR-069` — QUEUED 는 정의상 종료 전이라 봉인 렌더를 못 쓴다)

---

## 7. internal/match

순수 매처는 시그니처를 안 바꾼다. draining 제외는 호출자(api submit)가 `busy`
집합에 draining 노드를 합쳐 넘기는 것으로 이룬다 — **`if !dry` 안**이다
(`ADR-063 §6` · `api.go:389~395`). `match.Match` 는 이미 `busy` 를 받는다.

---

## 8. internal/api — 새 핸들러

```go
func (s *Server) getNodes(w http.ResponseWriter, r *http.Request) // GET /v1/nodes
func (s *Server) getRuns(w http.ResponseWriter, r *http.Request)  // GET /v1/runs 목록
func (s *Server) postDemo(w http.ResponseWriter, r *http.Request) // 데모 제출 (Q5)
// getRun 은 기존. runView 에 requires · as · steps[].chosen 을 더한다.
```

- 셋 다 기존 `s.auth(...)` 규칙을 따른다. `getNodes`·`getRuns` 는 Mediator
  토큰으로 보호 — 지금 표면과 같은 규칙 (`3.1.1`)
- `api.go` 에는 등록 줄만 는다. 핸들러는 새 파일 (`nodes.go`·`runs.go`·`demo.go`)
- `submit` 응답에 `202 Accepted`(QUEUED) 분기를 더한다 — `201`(새로 만듦)·
  `200`(멱등 재제출)과 나란히. `409` 는 더 이상 「전부 점유」의 뜻이 아니다
  (`mediator-api.md §1` 개정)

**postDemo (Q5 = A)**:

```text
   입력 검증    본문의 시나리오 id 가 allow-list(고정 시나리오 이름 집합)에
                있나.  없으면 거부 (임의 계약 불가)
   토큰 주입    서버측 Mediator 토큰으로 내부 submit 을 부른다 —
                브라우저에 실 토큰 없음 (SECURITY-08 가둠)
   제출자       submitter(표시 라벨)를 받아 CreateRun/CreateQueuedRun 에 넘긴다
   픽스처 출처   internal/contract 의 example (runctl example · decisions §8.3)
```

---

## 9. cmd/enodectl · cmd/enode

```text
   cmd/enodectl serve <name>   HTTP 서버를 직접 안 세운다. 형제 enode 의 제어판
                               프로세스를 exec 하고 stdin/stdout/stderr 를 잇고
                               종료 코드를 그대로 쓴다 (setup.go 와 같은 모양).
                               proc 원장으로 status/start/stop/logs 는 그대로
   cmd/enode <panel 하위명령>   internal/panel.New(...).Handler() 를 net/http 로
                               띄운다. enode 는 이미 net/http 를 링크한다
```

이름(`serve` 가 exec 하는 enode 하위명령이 무엇인지)은 FD·Units Generation 이
굳힌다.

---

## 10. 비즈니스 로직은 여기서 안 짠다

아래는 **Functional Design(유닛별) 몫**이다 — 이 문서는 시그니처까지만 둔다.

```text
   정책 파일의 위치·형식·enum 값        drain 유닛 FD (decisions §1)
   at-boundary 취소의 정확한 트랜잭션    큐/ drain 유닛 FD
   링 파일 머리·몸통·WriteAt 감김 로직    트랜스크립트 유닛 FD (decisions §6.3)
   QUEUED 재매칭 시 tx 재구성 세부        큐 유닛 FD
   Capabilities 읽기 계약 · sandbox 출처  진행자 decisions · 해당 유닛 FD
```
