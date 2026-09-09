# drain — Domain Entities

새 표 · 새 열 없음(obs 가 냈다). 는 것은 **파일 하나 · 구조 셋 · 함수 셋**이다.

---

## 1. 정책 파일

```text
   경로      <dir>/<stem>.policy.yaml     설정이 /home/u/.config/enode/local.yaml 이면
                                          /home/u/.config/enode/local.policy.yaml
   형식      yaml
   키        drain: "" | graceful | at-boundary     (모르는 키는 무시 — panel_token 이 뒤에 앉는다)
   권한      0600 권장 (유닉스).  이 유닛은 읽기만 하고 경고만 낸다
```

```yaml
# <stem>.policy.yaml — 소유자가 쓴다.  없으면 안 걸린 것이다.
drain: at-boundary
```

## 2. `internal/enode` — 구조와 함수

```go
// policy.go (신규)
type Policy struct {
    Drain string `yaml:"drain"`
}
func PolicyPath(configPath string) string            // <dir>/<stem>.policy.yaml
func ReadPolicy(path string, log *slog.Logger) Policy  // 없음 → 빈 값 · 틀림 → 빈 값 + Warn · 권한 경고 (Q3=A)

// advertise.go
type AdvertResponse struct {
    Leases       []Lease
    RenewSeconds int
    Drain        string `json:"drain"`   // 새로 든다.  서버가 언제나 싣는다 (obs)
}
type Advertiser struct {
    ...
    Held *Held   // 새로 든다 (Q2=A).  응답의 drain 과 주기를 Worker 에 나른다.  nil 이면 안 나른다
}

// leases.go
func (h *Held) SetDrain(mode string)            // 응답이 준 값 그대로.  바뀌면 true 를 돌려준다 (로그용)
func (h *Held) Drain() string
func (h *Held) SetRenew(d time.Duration)        // Worker 가 at-boundary 에서 기다리는 길이
func (h *Held) Renew() time.Duration            // 0 이면 5초

// claim.go — Worker.Run 의 claim 직전
//   if w.Held.Drain() == contract.DrainAtBoundary { wait(w.Held.Renew()); continue }
```

어휘 상수는 `internal/store` 에 있다(`DrainNone` · `DrainGraceful` · `DrainAtBoundary`).
enode 는 store 를 못 딛으므로(임포트 방향) `internal/contract` 에 같은 셋을 둔다 —
`contract.Policy` 옆이 자리다. store 의 상수는 그것을 가리키게 한다(한 벌).

## 3. `internal/store` · `internal/api`

```go
// store/queue.go — 한 함수
func (s *Store) NodeDrain(ctx context.Context, nodeID string) (string, error)   // nodes.draining.  없으면 ""

// api/api.go — postResult 끝의 갈래 (business-logic-model §4)
```

## 4. `cmd/enode/main.go` — 한 줄 (Q2 = A · 조율)

```go
adv := &enode.Advertiser{
    Client: client, Ident: ident, Every: *every, Log: log,
    Caps:     det.Capabilities,
    OnLeases: held.Set,
    Held:     held,   // 새 한 줄 — 응답의 drain 을 Worker 에 나른다 (ADR-063 §4)
}
```

파일 행렬상 `cmd/enode` 는 panel(nacl1119)의 것이다. 한 줄이고 리터럴 필드 하나라 충돌
면적이 작다 — PR 에 「조율」로 적고 진행자가 직렬로 병합한다.

## 5. 파일 행렬 — 재확인

```text
   internal/enode/policy.go        N    단독 drain
   internal/enode/advertise.go     W    단독 drain (transcript 는 runner.go · claim.go 의 tee)
   internal/enode/leases.go        W    단독 drain
   internal/enode/claim.go         W    Worker.Run 한 분기 — transcript 가 같은 파일의 다른 자리를 만질 수 있다.  조율
   internal/contract/advert.go     W    drain 어휘 상수 셋 (obs 의 Policy 옆)
   internal/store/queue.go         W    NodeDrain 하나 (store 접점 · 진행자 직렬)
   internal/store/store.go         W    상수 셋을 contract 의 것으로 (한 벌)
   internal/api/api.go             W    postResult 갈래 (api 접점 · 진행자 직렬)
   cmd/enode/main.go               W    한 줄 (panel 의 파일 · 조율)
   시험   internal/enode  policy_test.go(경로 · 읽기 · 접기 · 권한 경고 · DB 없음) ·
                          advertise 시험에 policy 실림 · drain 응답이 Held 로 · Worker 가 안 집음(httptest)
          internal/api    at-boundary 취소(단계 둘 계약) · graceful 무취소 · 되돌림 뒤 취소 · 해제 후 승격 (DB)
```
