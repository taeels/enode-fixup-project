# drain — Code Generation 계획

AI-DLC Construction · **drain 유닛**(W2 · CP3)의 Code Generation Part 1 이다. **Part 2 는 이
문서만 본다.** 정본 — FD 셋(`construction/drain/functional-design/`) · NFR 둘
(`.../nfr-requirements/`) · 팩 · `CONVENTIONS.md`. 어긋나면 `enode-design` 이 이긴다.

---

## 0. 유닛 맥락

```text
   지는 기능    3.2.2 소유자 자원 제어(drain)
   재는 게이트  CP3 「돌려받는다」 (+ CP0 회귀)
   의존         queue — 병합됨 (310c22d).  DrainingNodes · Cancel 의 깨우기 · 해제 깨우기를 그대로 쓴다
   딛는 유닛    panel(nacl1119) — 이 FD 의 정책 파일 형식을 토글이 쓴다
   브랜치       unit/drain.  접점 — internal/store(queue.go 한 함수 · store.go 상수) · internal/api/api.go ·
                cmd/enode/main.go 한 줄(조율) · internal/enode/claim.go 한 분기(조율)
   시험 환경    Postgres.app 55434 (그대로).  CP3 는 enode_cp2 판을 다시 쓴다
```

## 1. 갈래 셋 — 순서 A → B → C

```text
   A   internal/contract · internal/store        어휘 상수 한 벌 · NodeDrain                 의존 없음
   B   internal/enode · cmd/enode                policy.go · advertise · leases · claim · main 한 줄   A 의 상수
   C   internal/api                              postResult 갈래                              A 의 NodeDrain
```

## 2. 실행 순서

- [x] 2.1 갈래 A (4.A)
- [x] 2.2 갈래 B (4.B)
- [x] 2.3 갈래 C (4.C)
- [x] 2.4 `go build` · `go vet` · `gofmt` · glyphscan · 크로스 빌드 넷
- [x] 2.5 `go test ./internal/enode ./internal/contract ./cmd/enode` (DB 없음) · `./internal/store ./internal/api` (DB) · 전체 커버리지 awk
- [x] 2.6 코드 요약(냄) · 상태 · 감사 · 커밋은 승인 뒤 — 코드 요약 `construction/drain/code/code-summary.md` · 상태 · 감사 · 커밋(승인 뒤)

## 3. 못 박는 겉면

```go
// internal/contract/advert.go — 어휘 상수 (새로 든다 · store 의 셋은 이것의 별칭이 된다)
const (
    DrainNone       = ""
    DrainGraceful   = "graceful"
    DrainAtBoundary = "at-boundary"
)

// internal/store/queue.go
func (s *Store) NodeDrain(ctx context.Context, nodeID string) (string, error)   // nodes.draining · 없으면 ""

// internal/enode/policy.go (신규)
type Policy struct{ Drain string `yaml:"drain"` }
func PolicyPath(configPath string) string                       // <dir>/<stem>.policy.yaml
type policyReader struct{ path string; log *slog.Logger; lastWarn string }
func (r *policyReader) Read() contract.Policy                   // 없음 → 빈 값 · 틀림 → 빈 값 + Warn(원인이 바뀔 때만) · 권한 경고

// internal/enode/advertise.go
type AdvertResponse struct{ Leases []Lease; RenewSeconds int; Drain string `json:"drain"` }
type Advertiser struct{ ...; Held *Held }                       // nil 이면 안 나른다 (기존 시험 그대로)

// internal/enode/leases.go
func (h *Held) SetDrain(mode string) (changed bool)
func (h *Held) Drain() string
func (h *Held) SetRenew(d time.Duration)
func (h *Held) Renew() time.Duration                            // 0 이면 5초
```

## 4. 갈래별 산출

### 4.A internal/contract · internal/store
- [x] A1 `internal/contract/advert.go` — 상수 셋을 `Policy` 옆에. 주석에 어휘와 접는 자리
- [x] A2 `internal/store/store.go` — `DrainNone` 등 셋을 `= contract.DrainNone` 별칭으로 (한 벌). `DrainPolicy` 무변경
- [x] A3 `internal/store/queue.go` — `NodeDrain`. `SELECT draining FROM nodes WHERE node_id=$1` · `ErrNoRows` 면 `""`
- [x] A4 시험 — `queue_test.go` 에 `TestQueue_NodeDrainReadsTheCopy` (광고 뒤 값 · 없는 노드 "")

### 4.B internal/enode · cmd/enode
- [x] B1 `policy.go` — `PolicyPath` · `policyReader.Read`. yaml.v3 · 모르는 키 무시 · 어휘 검사 · 유닉스 권한 비트 검사(`runtime.GOOS != "windows"` · `mode&0o022 != 0` 면 Warn 한 번)
- [x] B2 `advertise.go` — `Advertiser.Held` · `Policy *policyReader`(nil 이면 `Ident.Config` 로 만든다) · 광고 직전 `ad.Policy = contract.Policy{Drain: r.Read().Drain}` · 응답 뒤 `Held.SetDrain(resp.Drain)`(바뀌면 Info "drain acknowledged by mediator") · `Held.SetRenew(a.Every)`
- [x] B3 `leases.go` — `Held` 에 `drain string` · `renew time.Duration` 과 겉면 넷
- [x] B4 `claim.go` `Worker.Run` — claim 직전: `if w.Held.Drain() == contract.DrainAtBoundary { (바뀔 때 Info) select ctx / time.After(w.Held.Renew()); continue }`
- [x] B5 `cmd/enode/main.go` — `Advertiser{..., Held: held}` 한 줄 (조율 · panel 의 파일)
- [x] B6 시험 — `policy_test.go`(경로 · 없음 · graceful · at-boundary · 파싱 실패 · 어휘 밖 · 권한 경고 · 같은 원인 한 번만) ·
      `drain_test.go`(httptest: 광고 본문에 policy 실림 · 응답 drain 이 Held 로 · 바뀔 때만 로그 · Worker 가 at-boundary 에서 claim 을 안 건다 · graceful 은 건다)

### 4.C internal/api
- [x] C1 `api.go` `postResult` — `rolled` 갈래와 정산 뒤 갈래 둘 다에서 `s.drainAtBoundary(ctx, runID, body.Node)` 를 부른다.
      함수 하나: `NodeDrain` → at-boundary 면 `Cancel(runID, "drain:"+node)` → Info · 실패는 Error(답 A). 돌려준 상태를 응답 `run_state` 에
- [x] C2 시험 — `drain_test.go`(newServerFast · 단계 둘 계약 `oneStepRun` · 광고에 policy):
      `TestDrain_AtBoundaryClosesTheRunAfterTheStepReport` (FAILED · verdict note 에 drain:n1 · lease 0 · steps 2 는 FAILED · Record 에 step 1 산출) ·
      `TestDrain_GracefulLetsTheRunFinish` · `TestDrain_LastStepIsSettledNotCancelled` (SUCCEEDED) ·
      `TestDrain_ReleaseInAdvertPromotesTheWaiter` (202 QUEUED → 해제 광고 → RUNNING · queue 의 것과 겹치지만 drain 경로로 다시 잰다) ·
      `TestDrain_RolledBackStepStillClosesAtBoundary` (loop 계약이 있으면 · 없으면 생략하고 이유를 적는다)

## 5. 짓지 않는 것
```text
   정책 파일 쓰기 · 토글 · 배지        panel · ui
   중앙 라우트 · 재시도 큐             정본 · NFR 답 A
   Worker 의 파일 직접 읽기            FD Q2
   runner.go                         transcript 의 파일.  안 연다
```

## 6. 재는 순서 (CP0 · CP3)
```text
   CP0   queue 와 같다 — go test ./... (DSN) · 커버리지 awk · glyphscan · vet · 크로스 빌드
   CP3   enode_cp2 판 · Mediator · 노드 하나 (local.yaml 옆에 local.policy.yaml)
         단계 둘 계약 a (각 sleep) → RUNNING
         정책 파일에 drain: at-boundary → 다음 광고 → GET /v1/nodes draining: at-boundary
         단계 1 결과 보고 뒤 → GET /v1/nodes lease 없음 · runctl status a → FAILED · verdict note drain:<node_id> ·
         runctl record a → 단계 1 산출이 있다
         새 계약 b → 202 QUEUED (후보 제외)
         정책 파일에서 drain 지움 → 다음 광고 → b RUNNING
         (graceful 도 한 번 — 도는 Run 이 끝까지 간다)
```

## 7. 진행자에게
```text
   ①  조율 둘 — cmd/enode/main.go 한 줄 · internal/enode/claim.go 한 분기
   ②  store.go 의 상수 셋이 contract 의 별칭이 된다 (obs 파일 한 줄)
   ③  internal/api 커버리지 여유 3 문장 — C1 의 갈래를 C2 가 전부 덮는다
```
