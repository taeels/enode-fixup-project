# U4 `log-api` — Code Generation 계획 (Part 1)

```text
   유닛     log-api (U4) · 웨이브 W-b · 브랜치 unit/log-api
   앞 단계   Functional Design 승인 2026-09-16T02:05:00Z.  답 여덟 전부 A
   NFR      **스킵** (2026-09-16T02:20:00Z · 사용자 지시).  회차 계획과 갈린다 —
            계획은 EXECUTE 이고 U4 가 N1 을 지도록 걸려 있었다
   지는 게이트  **CB0** — 라우트가 17 에서 18
   기준선    5d53e52 + 0d9f14c.  라우트 17 · internal/api 커버리지 82.4%
```

---

## 0. NFR 스킵이 이 계획에 넘긴 값 둘

**선례가 있다** — v1 의 `obs` 에서 NFR Design 을 스킵했을 때 답이 남긴 「어떻게」를
Code Generation 계획 5절이 졌다. 같은 모양으로 여기가 진다.

### 0.1 한 응답의 본문 상한 — **1 MiB. 상수다**

```text
   자리     internal/api/log.go 의 상수.  설정 키를 안 만든다
   근거     조절 손잡이가 아니라 보호다.  키로 두면 이름 · 기본값 · 문서 · 시험이
           함께 생기고, 값을 올릴 수 있는 것 자체가 보호를 무르는 길이다
   대가     10 MiB 짜리 진행 파일을 처음부터 읽으면 응답이 열 번이다.
           from 규약이 이미 그것을 감당한다 — 새 개념이 0 이다
```

### 0.2 N1 — 함대 규모의 부하. **안 닫힌 채로 남되 봉투를 적는다**

**재는 대신 산수를 남긴다.** 안 적으면 다음 사람이 봉투가 없다고 읽는다.

```text
   미는 쪽    청크 PUT — 단계마다 2초에 한 번 (64 KiB 면 즉시).
             도는 단계가 S 개면 S/2 req/s.  S=50 이면 25 req/s

   당기는 쪽   현황판 Run 상세 — 보이는 단계 카드마다 2초.
             관객 V 명 · 카드 C 개면 V x C / 2 req/s.  V=5 · C=8 이면 20 req/s

   천장      **데모의 전역 한도 120 req/s · 버스트 240** (ratelimit.go:19-20).
             이 라우트가 그 한 바구니를 GET /v1/nodes · GET /v1/runs 와 나눠 쓴다
```

**여기가 N1 이 실제로 아픈 자리다.** v1 의 `obs` 가 이미 이름으로 적었다 —
「전역 한도의 대가로 한 클라이언트가 나머지를 굶길 수 있다」. **이 회차가 폴링
라우트를 그 바구니에 더한다.** 굶으면 현황판이 비고, 비는 것과 안 도는 것을
사람이 못 가른다.

```text
   이 유닛이 하는 것    산수를 적는다.  한도를 안 바꾼다 — 바꾸면 v1 이 값을 보고
                     고른 자리를 이 유닛이 되돌리는 것이 된다
   U8 에 넘기는 규칙    **보이는 카드만 2초로 폴링한다.**  접힌 카드는 안 부른다.
                     C 를 줄이는 것이 유일하게 싼 손잡이다
   잔여              N1 은 안 닫힌다.  Build and Test 가 CB4 에서 실측한다
```

---

## 1. 이 유닛이 내는 diff 의 모양

| 파일 | 새것/고침 | 무엇 |
|---|---|---|
| `internal/api/log.go` | **신규** | `getLog` 핸들러 · 상한 상수 · 헤더 넷 |
| `internal/api/api.go` | 고침 | 등록 줄 하나 · `putLog` 의 갈림과 헤더 |
| `internal/record/record.go` | 고침 | `OpenLog` |
| `internal/transcript/transcript.go` | 고침 | `Event` 에 json 태그 |
| `internal/api/getlog_test.go` | **신규** | GET 의 규칙 |
| `internal/api/log_test.go` | 고침 | PUT 의 갈림과 헤더 |
| `internal/record/record_test.go` | 고침 | `OpenLog` |

```text
   go.mod · go.sum   diff 0.  새 의존 0
   internal/store    diff 0.  GetRun 만 부른다
   U2 와의 파일 교집합  **0**
```

**시험 파일 이름이 이미 쓰였다** — `internal/api/log_test.go` 가 있다 (PUT 과 tar).
**GET 의 시험은 새 파일 `getlog_test.go` 에 둔다.** 한 파일에 섞으면 PUT 의 시험과
GET 의 시험이 같은 헬퍼를 두고 서로를 고치게 된다.

**행렬 밖 파일이 둘이다** — `internal/record/record.go` (행렬은 U3 하나) ·
`internal/transcript/transcript.go` (행렬은 U1 하나). **둘 다 이미 병합됐으므로
병합 충돌은 0 이다.** 6절이 든다.

---

## 2. 못 박는 겉면 — Step 1 이 먼저 한다

```go
// record.go
func (s *Store) OpenLog(runID string, seq int, name string) (io.ReadCloser, int64, error)

// log.go
const maxLogSliceBytes = 1 << 20

func (s *Server) getLog(w http.ResponseWriter, r *http.Request)
```

```text
   헤더 넷   X-Enode-Log-Bytes · X-Enode-Log-Source · X-Enode-Log-Attempt ·
            X-Enode-Log-Capped.  다섯째를 안 만든다
```

---

## 3. 단계 — 열셋

### Step 1 — 겉면을 못 박는다
- [ ] `record.OpenLog` 의 시그니처 (몸통은 Step 2)
- [ ] `log.go` 에 상수와 핸들러 시그니처
- [ ] `go build ./...` 가 선다

### Step 2 — `record.OpenLog`
- [ ] `OpenBlob` 과 같은 모양으로 연다 — 경로는 `dir()` 과 `safe()` 를 쓴다
- [ ] 없는 파일은 `os.ErrNotExist` 를 그대로 싼다 (부르는 쪽이 200/0 으로 가른다)
- [ ] 총 길이를 `Stat` 으로 낸다
- [ ] 시험 — 있는 것 · 없는 것 · 이름에 경로 탈출이 든 것

### Step 3 — `transcript.Event` 의 json 태그
- [ ] 필드마다 `snake_case` 태그. `OK` 는 `ok,omitempty` (R19)
- [ ] **동작이 안 바뀐다** — U1 의 왕복 시험이 그대로 초록이어야 한다
- [ ] 마샬 결과를 고정하는 시험 하나 (키 이름의 정본이 여기다)

### Step 4 — `log.go` 의 검증
- [ ] `seq` 양의 정수 아니면 400 (R4)
- [ ] `from` 음이 아닌 정수 아니면 400. 없으면 0 (R5)
- [ ] `as` 가 `raw` · `events` 아니면 400. 없으면 `raw` (R6)
- [ ] `name` 없으면 `step` (R6.1)

### Step 5 — `log.go` 의 갈림과 본문
- [ ] `needRecords` -> `GetRun` (404 는 여기 하나)
- [ ] `records.Sealed(runID)` 로 출처를 가른다 (R10)
- [ ] `progress` 면 `ReadProgress(run, seq, name, from)`
- [ ] `sealed` 면 `OpenLog` 뒤 `from` 만큼 `Seek`
- [ ] 파일이 없으면 200 에 `Bytes: 0` · 빈 본문 (R13)

### Step 6 — `log.go` 의 상한과 개행 (R7 · R8)
- [ ] 조각을 `maxLogSliceBytes` 로 자른다
- [ ] 잘렸으면 **마지막 개행에서 끊는다**
- [ ] 조각에 개행이 0 이면 상한 그대로 끊는다 (한 줄이 상한보다 길다)
- [ ] `X-Enode-Log-Bytes` 는 **총 길이** — 조각 길이가 아니다

### Step 7 — `log.go` 의 `as=events` (R9 · R18 · R20)
- [ ] `truncated` 를 정한다 — `from == 0` 이면 false, 아니면 `from-1` 이 개행인가
- [ ] `transcript.Parse(조각, truncated)` 의 `Result` 를 그대로 JSON 으로
- [ ] 헤더 넷을 **본문보다 먼저** 쓴다

### Step 8 — `api.go` 의 등록 줄과 PUT 의 갈림
- [ ] `mux.HandleFunc("GET /v1/runs/{run}/steps/{seq}/log", read(s.getLog))` — **한 줄**
- [ ] `putLog` 에 `progress` 쿼리 갈림. `attempt` 가 규칙 밖이면 400 (R6.2)
- [ ] 종료 상태는 두 갈래 다 410 (R15) — 오늘 자리 그대로
- [ ] 두 갈래 다 `X-Enode-Log-Bytes` 를 달고 200 (R16). 204 를 걷는다
- [ ] 진행 갈래만 `Attempt` · `Capped` 를 더 단다

### Step 9 — 시험: GET 의 갈래
- [ ] 400 넷 (`seq` · `from` · `as` · 음수)
- [ ] 404 는 없는 Run 하나. 없는 `seq` 는 **200 에 0** (R13 · 답 7 = A)
- [ ] 봉인 전 `Source: progress` · 봉인 뒤 `Source: sealed`
- [ ] `Bytes` 가 총 길이이고 조각 길이가 아니다
- [ ] `from` 을 이어 보내면 같은 바이트가 두 번 안 온다

### Step 10 — 시험: 상한과 개행 (R7 · R8 이 이 유닛의 축이다)
- [ ] 상한보다 긴 파일에서 본문이 **개행으로 끝난다**
- [ ] `from + len(본문)` 을 다음 `from` 으로 써서 **끝까지 이어 읽으면 원본과 같다**
- [ ] 한 줄이 상한보다 길면 개행 없이 끊기고 `partial` 이 0 이 아니다
- [ ] `as=events` 로 같은 왕복을 돌아도 줄이 안 쪼개진다

### Step 11 — 시험: PUT 의 갈림
- [ ] `progress=1` 이 진행 파일로 · 없으면 `logs/` 로 간다
- [ ] 두 갈래 다 `Bytes` 헤더가 온다
- [ ] 봉인된 Run 은 두 갈래 다 410
- [ ] `attempt` 가 없거나 0 이하면 400

### Step 12 — 변이
- [ ] 개행 끊기를 지운다 -> Step 10 이 빨개져야 한다
- [ ] `Sealed` 대신 `run.State` 로 가른다 -> Step 9 가 빨개져야 한다
- [ ] `Bytes` 에 조각 길이를 싣는다 -> Step 9 가 빨개져야 한다
- [ ] `truncated` 를 언제나 false -> Step 10 이 빨개져야 한다
- [ ] `OK` 의 `omitempty` 를 뗀다 -> Step 3 이 빨개져야 한다

### Step 13 — 게이트와 문서
- [ ] **CB0 — `grep -c 'mux.HandleFunc' internal/api/api.go` 가 18 이다**
- [ ] `go build ./... && go vet ./... && gofmt -l .` 가 빈다
- [ ] `go run ./scripts/glyphscan.go` 가 0 이다
- [ ] 전체 시험 초록 · **스킵 0**
- [ ] 커버리지 정본 명령으로 **미달 0**. `internal/api` 가 80% 위다
- [ ] `internal/store` diff 0
- [ ] `git status --porcelain` — `probe.lock` 만 나오면 되돌린다
- [ ] `construction/log-api/code/code-summary.md`. **잰 것만 적는다**
- [ ] 이 계획의 체크박스를 **끝낸 그 자리에서** 채운다

---

## 4. 무엇으로 재는가

```text
   기계    CB0 — 라우트 17 -> 18.  그리고 CP0 계열 전부
          eval "$(scripts/testdb.sh)" 를 먼저 친다 — internal/api 가 DB 를 탄다
   사람    없다.  CB1 · CB2 · CB4 는 화면이 선 뒤다
   봉투    0.2 의 산수.  CB4 가 실측으로 받거나 뒤집는다
```

**`internal/api` 의 커버리지가 얇다** — 오늘 82.4% 이고 하한이 80% 다. 핸들러
하나가 통째로 들어오므로 **Step 9 ~ 11 이 안 서면 이 유닛이 게이트를 깬다.**
그래서 시험 Step 이 셋이다.

---

## 5. 짓지 않는 것

```text
   runctl 의 StepLog     U6 (application-design 1.2 의 일곱째 경로)
   화면                 U5 · U6 · U8
   노드 쪽 업로더         U7
   권한 모델             팩 밖 (requirements.md 5.4 잔여 ②)
   한도 값의 변경         v1 이 값을 보고 고른 자리다.  산수만 적는다 (0.2)
   새 라우트 둘째         D2.  PUT 은 쿼리로 가른다
```

---

## 6. 진행자에게 넘기는 것 — 여섯

```text
   unit-of-work-file-matrix.md 1절   internal/record/record.go 가 U3 하나다 (OpenLog)
   unit-of-work-file-matrix.md 1절   internal/transcript/** 가 U1 하나다 (json 태그)
   component-methods.md 4.1          응답 상한과 개행 규칙이 없다.  헤더 넷만으로는
                                    폴링이 안 닫힌다
   requirements.md 5.4 잔여 ②        토큰이 있다는 전제다.  데모에는 없다
   execution-plan.md                 NFR 둘이 EXECUTE 로 적혀 있다.  실제로 SKIP 이다.
                                    **회차 계획을 고칠지는 진행자의 것이다**
   N1                               안 닫힌다.  봉투는 0.2 · 실측은 CB4 의 몫
```

---

## 7. 물음 — 0

**이 단계는 안 묻는다.** 갈래는 답 여덟이 닫았고, NFR 스킵이 남긴 값 둘은
0절이 값과 근거까지 적었다. **둘 다 되돌리는 값이 작다** — 상한은 상수 하나이고
N1 은 산수라 CB4 의 실측이 뒤집으면 그 자리에서 고친다.
