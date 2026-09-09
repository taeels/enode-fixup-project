# obs — Code Generation 계획

AI-DLC Construction · **obs 유닛**(W0 · CP1 · 토대)의 Code Generation Part 1 이다.
코드가 아니라 **무엇을 어느 순서로 짓고 무엇으로 재는가**의 계획이다.

정본 — 이 유닛의 Functional Design 산출물 셋
(`aidlc-docs/taeels/construction/obs/functional-design/`) · NFR Requirements 둘
(`.../nfr-requirements/`) · 팩 `requirements/` · `CONVENTIONS.md`.
**어긋나면 `enode-design` 이 이긴다** (`requirements/canon.md` 머리말).

---

## 0. 이 계획이 지고 가는 것 — NFR Design 이 SKIP 이다

회차 실행 계획이 NFR Design 을 SKIP 으로 승인받았고, NFR Requirements 6절
표시 ⑤ 가 그 자리를 이 계획에 넘겼다.

```text
   NFR Requirements 가 값을 다 적었다   초당 120 · 버스트 240 · 429 + Retry-After: 1 ·
                                      전역 하나 · 데모 모드의 읽기 셋에만
   이 계획이 정하는 것은 「어떻게」다     자료구조 · 미들웨어를 어디에 거는가 · 파일 이름
                                      5절이 그 자리다
```

**사용자가 「이제 코드 작성해라 · 병렬로 할 수 있는 것들은 병렬로 해라」로
Part 2 를 함께 지시했다.** 그래서 이 계획은 승인을 기다리는 문서가 아니라
**Part 2 의 실행 명세**다. 앞 단계(NFR Requirements)의 승인도 그 지시가 겸한다 —
`audit.md` 에 그대로 적는다.

---

## 1. 병렬로 가른다 — 파일이 안 겹치는 세 갈래

사용자 지시가 병렬이다. **가르는 축은 단계가 아니라 파일**이다 — 같은 파일을
둘이 열면 병렬이 아니라 충돌이다.

```text
   갈래   패키지                              겹치는 파일
   ────   ─────────────────────────────────   ───────────
   A      internal/store · internal/contract   없다
   B      internal/api · internal/config       없다
   C      internal/runctl                      없다
```

**A 와 B 사이에만 의존이 있다** — B 가 A 의 시그니처를 부른다. 그 시그니처는
`domain-entities.md` 8절이 이미 못 박았으므로 **3절에 그대로 옮겨 고정**하고,
셋이 동시에 그 계약 위에서 짠다. C 는 `json.RawMessage` 라 A·B 어느 쪽도 안 딛는다.

```text
   A  ── 시그니처(3절) ──>  B
   C  (독립)
```

병합은 진행자의 직렬 대상 규칙 그대로다 (`business-rules.md` 7.5).

---

## 2. 실행 순서

- [x] 2.1 계약을 못 박는다 (3절). 세 갈래가 같은 글자를 본다
- [x] 2.2 A · B · C 를 병렬로 짓는다 (4절)
- [x] 2.3 합쳐서 빌드 · 시험 · 커버리지 (7절)
- [x] 2.4 CP0 회귀와 CP1 의 curl 두 줄 (8절)
- [x] 2.5 `aidlc-state.md` 갱신 · `audit.md` 추가 · 커밋

---

## 3. 못 박는 계약 — 세 갈래가 동시에 본다

`domain-entities.md` 8절을 그대로 옮긴 것이다. **여기서 짐작하지 않는다.**

```go
// internal/store
func (s *Store) Nodes(ctx context.Context) ([]NodeView, time.Time, error)
func (s *Store) Runs(ctx context.Context, f RunFilter) ([]RunRow, time.Time, error)
func (s *Store) UpsertAdvert(ctx context.Context, a contract.Advert,
        principal string, ttl time.Duration) (prevDrain string, err error)
func DrainPolicy(v string) string          // 어휘 밖이면 "" 를 낸다.  3.1
func RequiresOf(c contract.Contract) []RequireView

// internal/runctl
func (c *Client) Nodes(ctx context.Context) (json.RawMessage, error)
func (c *Client) Runs(ctx context.Context, q RunsQuery) (json.RawMessage, error)
```

### 3.1 `DrainPolicy` 를 왜 내보내는가

`UpsertAdvert` 의 반환은 **이전** 값이고(`WakeQueued` 의 트리거 · drain W2 가 쓴다),
`advertResponse.drain` 은 **저장된** 값이라 둘이 다르다. 시그니처를 안 넓히고
같은 값을 얻는 길은 접는 함수를 순수하게 내보내는 것이다.

```text
   store 안        검사 + 접기 + 로그.  쓰는 자리가 하나이므로 그 자리에 붙는다
   api 가 부르는 것  접기만 (순수).  로그는 안 부른다 — 두 번 남으면 기록이 갈린다
```

응답이 「저장된 값」과 언제나 같다는 것을 **같은 함수를 쓰는 것으로** 보장한다.

### 3.2 `RequireView` 를 store 에 두는 이유

`internal/api` 의 커버리지 여유가 **5.6 문장**이고 `internal/store` 는 31.2 다
(`business-rules.md` 7.6). 사상 루프를 store 에 두면 그 예산을 안 먹는다.
정책 어휘 검사를 store 에 둔 것과 **같은 수 · 같은 논거**다. store 는 이미
`StepView` · `LedgerEntry` 같은 관측 뷰 타입을 지고 있으므로 자리도 맞다.

---

## 4. 갈래별 산출

### 4.A internal/store · internal/contract

- [x] A1 `internal/contract/advert.go` — `Policy{Drain}` 과 `Advert.Policy`
      (`json:"policy,omitzero"`). `LiveAdverts` 가 안 채운다는 비대칭을 주석에
      남긴다 (`domain-entities.md` 7.3)
- [x] A2 `internal/store/schema.sql` — 셋
      `ALTER TABLE nodes ADD COLUMN IF NOT EXISTS draining text NOT NULL DEFAULT ''` ·
      `ALTER TABLE runs ADD COLUMN IF NOT EXISTS submitter text NOT NULL DEFAULT ''` ·
      `CREATE INDEX IF NOT EXISTS runs_created_idx ON runs (created_at)`.
      CHECK 를 안 건다 — `steps.state` 와 같은 이유
- [x] A3 `store.Run` 에 `Submitter string`. `CreateRun` · `CreateRejectedRun` 의
      INSERT 열에 든다. **거절 경로를 빼면 CP9 가 재는 행의 이름이 빈다**
- [x] A4 `UpsertAdvert` — CTE 로 이전 `draining` 을 받으면서 새 값을 쓴다.
      `coalesce` 는 **서브쿼리 밖**이다 (0행이면 NULL 이고 그러면 첫 광고가 죽는다)
- [x] A5 `DrainPolicy` — 셋(`""` · `graceful` · `at-boundary`) 밖이면 `""` 로 접는다.
      `UpsertAdvert` 가 접을 때 `s.log()` 에 남기되 **값 원문을 안 찍는다**
      (SECURITY-03)
- [x] A6 `NodeView` · `LeaseView` · `Store.Nodes` — 1행 `at` CTE 를 `FROM` 에 두고
      `LEFT JOIN`. 노드 0 이어도 `observed_at` 이 선다. 만료 필터도 그 시각으로 건다
- [x] A7 `RunRow` · `RunFilter` · `Store.Runs` — `LEFT JOIN LATERAL`, `LIMIT` 은 안쪽.
      `coalesce(work_id,'')` · `coalesce(assigned,'[]'::jsonb)`. `?work=` 는
      **접기 전 열**에 등호를 건다 (인덱스를 타야 한다)
- [x] A8 `observe.go` — `StepView.Chosen bool` (`json:"chosen"` · 언제나 싣는다) ·
      `Steps` 의 SELECT 에 `chosen` 한 열
- [x] A9 `RequireView` · `RequiresOf` — `attrs` 로 감싼다. `count` 만 `omitempty`,
      `attrs` 는 비면 `{}`
- [x] A10 시험 — `internal/store/observe_obs_test.go` (진짜 Postgres)

### 4.B internal/api · internal/config

- [x] B1 `internal/config/config.go` — `Demo bool` 과 `ENODE_DEMO_MODE`
      (`1` · `true` · `yes` 를 참으로 읽는다). 기본값 거짓
- [x] B2 `internal/api/ratelimit.go` — 토큰 버킷 하나. 의존 0
- [x] B3 `internal/api/nodes.go` — `getNodes`. 질의 인자가 하나라도 오면
      `400 unknown query parameter`
- [x] B4 `internal/api/runs.go` — `getRuns` 와 질의 파싱 넷
- [x] B5 `internal/api/api.go` — 등록 두 줄 · `read()` 래퍼 · `submitterKey` 와
      `submitter(r)` · `submit()` 의 리터럴 **세 자리** · `advertResponse.Drain` ·
      `getRun` 의 `requires` 와 `warnings` · `noStore`
- [x] B6 시험 — `internal/api/observe_test.go`(DB) · `ratelimit_test.go`(DB 없음)

### 4.C internal/runctl

- [x] C1 `RunsQuery` — 빈 값과 0 을 질의에서 **뺀다** (제로값이 `?limit=0` 이 되면
      `runs.list` 의 첫 호출이 400 이다)
- [x] C2 `Client.Nodes` · `Client.Runs` — `json.RawMessage` 원문 그대로.
      파싱해서 다시 마셜하면 mcp 의 글자 일치(CP5)가 깨진다
- [x] C3 `Run.Requires` · `Step.Chosen` · `Step.StartedAt`
- [x] C4 시험 — `client_test.go` 에 이어 붙인다 (`httptest` · DB 없음)

---

## 5. 요청 한도의 「어떻게」 — NFR Design 이 SKIP 이라 여기가 그 자리

값은 `nfr-requirements.md` 3.1 이 다 적었다. 여기는 자료구조와 거는 자리다.

### 5.1 자료구조 — 토큰 버킷 하나

```text
   상태      토큰 수(실수) · 마지막 갱신 시각 · 뮤텍스 하나
   채우기    부를 때마다 경과 시간 x 초당 속도 만큼 채우고 버스트에서 자른다
             타이머도 고루틴도 없다 — 일회용 인스턴스에 배경 작업을 안 만든다
   판정      토큰이 1 이상이면 하나 빼고 통과.  아니면 거절
   시계      time.Now.  시험이 갈아 끼울 수 있게 필드로 둔다 (DB 도 슬립도 없다)
```

### 5.2 거는 자리 — 등록 줄의 래퍼 하나

**조건부 등록이 아니라 조건부 래퍼다.** 라우트 표는 모드와 무관하게 같은 줄이고
감싸는 것만 갈린다.

```go
read := func(h http.HandlerFunc) http.HandlerFunc {
        if s.cfg.Demo {
                return s.limit(h) // 무인증 + 한도
        }
        return s.auth(h) // 실 함대는 오늘 그대로
}
mux.HandleFunc("GET /v1/nodes", read(s.getNodes))
mux.HandleFunc("GET /v1/runs", read(s.getRuns))
mux.HandleFunc("GET /v1/runs/{id}", read(s.getRun))
```

**이 모양이 CP0 을 지킨다.** `business-rules.md` 7.5 가 「데모 조건부 등록이
기존 라우트의 등록 줄을 만진다 · 기본값이 거짓인 것이 그 회귀를 지킨다」고 적었는데,
래퍼로 두면 **등록 줄의 개수와 패턴이 모드와 무관하게 같다.** CP0 의 「기존 열다섯
라우트가 그대로」가 데모 모드에서도 참이 된다 — 조건부 등록보다 강한 성질이라
그쪽으로 간다. 셋이 늘 같은 자리에 있으므로 `ServeMux` 의 중복 등록 패닉도 없다.

**한도는 실 함대 경로에 안 걸린다.** `s.auth` 쪽에는 래퍼가 없다 —
`nfr-requirements.md` 3.1 의 「적용 범위」 그대로이고, 5절 「재는 자리」의
둘째 줄이 그것을 잰다.

### 5.3 429 의 모양

```text
   Retry-After: 1 을 먼저 세우고        fail() 이 WriteHeader 를 하므로 순서가 규칙이다
   fail(w, 429, "rate limited")        기존 규약 그대로 (api.go:144)
```

`429` 가 정본의 코드 표에 없다는 것은 `nfr-requirements.md` 6절 표시 ①로 이미
진행자에게 넘어가 있다. 코드는 그 값으로 간다.

---

## 6. 짓지 않는 것

```text
   새 라우트 셋째        requires 는 GET /v1/runs/{id} 를 넓힌다.  표면 개수가 비용이다
   view() 안의 requires  제출 응답 셋이 함께 넓어진다.  getRun 에서만 채운다
   매처 호출             관측 경로에서 안 돈다 (ADR-065)
   페이지네이션          limit 뿐
   판정                 「왜 안 맞나」를 서버가 안 낸다
   새 Go 의존            go.mod 를 안 움직인다.  한도를 직접 짜는 이유가 그것이다
   스크래치 DB 헬퍼       기존 newServer · newServerFast 를 쓴다
   .coverage-contract.yml  안 고친다.  낡은 정도만 유닛 닫을 때 한 줄로 적는다
```

---

## 7. 시험과 커버리지 예산

`internal/api` 의 여유가 **5.6 문장**이다 (`business-rules.md` 7.6). 새 문장 `n`
개에 대해 안 덮여도 되는 것은 `0.2n + 5.6`.

```text
   덮어야 하는 자리                                    어디서
   ─────────────────────────────────────────────────  ──────────────────────
   질의 파싱 넷의 갈래 (기본 · 상한 자름 · 400 셋)        api/observe_test.go
   GET /v1/nodes 의 질의 인자 거절                      같다
   빈 함대에서 observed_at 이 선다                      같다 · store 시험도
   lease 가 null 로 나간다 · 있으면 값이 나간다           같다
   assigned 가 [] 로 나간다 (거절된 Run)                 같다
   submitter 키가 언제나 있다                           같다
   requires 가 attrs 로 감싸여 나간다                    같다
   chosen 이 언제나 실린다                              같다
   Cache-Control: no-store 가 셋 다 나간다              같다
   한도 — 버킷 순수 로직 · 미들웨어 429 · 실 함대 무영향    api/ratelimit_test.go
   drain 왕복 — 저장 · 되돌림 · 어휘 밖이 "" 로 접힘       store 시험 + api 시험
   UpsertAdvert 의 첫 광고가 안 죽는다 (prevDrain "")    store 시험
   RunsQuery 가 빈 값과 0 을 뺀다                       runctl/client_test.go
   Nodes · Runs 가 원문을 그대로 낸다                    같다
```

**한도 시험에 DB 도 슬립도 없다** — 시계를 필드로 두는 이유가 그것이고,
`.ci-allowed-skips` 가 비어 있으므로 `t.Skip` 을 새로 안 만든다는 뜻이기도 하다.

---

## 8. 재는 순서 (CP0 · CP1)

**순서가 규칙이다** (`business-rules.md` 7.4).

```text
   1  eval "$(scripts/testdb.sh)"
   2  go build ./... && go vet ./...
   3  go test ./...
   4  go run ./scripts/glyphscan.go
   5  test -z "$(gofmt -l .)"
   6  test -z "$(git status --porcelain)"        커버리지보다 먼저다
   7  go test ./... -count=1 -coverpkg=./... -coverprofile=... + 패키지별 80 하한
   8  git checkout -- cmd/enodectl/probe.lock    7 이 그것을 바꾼다
   9  GOOS=windows 빌드 + 심볼 상한 둘 (net/http 50 · crypto/tls 10)
  10  CP1 의 curl 두 줄 (scene-gates 3절)
```

10 은 실 함대 · 실 데몬 · 합성 draining 노드가 필요하다
(`business-rules.md` 7.1). 이 세션이 그 함대를 못 세우면 **잰 것이 없는 것**이지
초록이 아니다 — 그대로 적고 진행자에게 넘긴다.

---

## 9. 진행자에게 넘기는 것 (앞 단계에서 이어짐)

Functional Design 여섯(`계획 8.5`)과 NFR Requirements 여섯
(`nfr-requirements.md` 6절)이 그대로 산다. **이 단계가 더하는 것은 없다** —
코드가 그 표시들을 닫지 않는다. 새로 생기면 유닛을 닫을 때 적는다.
