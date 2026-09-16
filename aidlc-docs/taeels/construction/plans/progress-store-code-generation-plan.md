# U3 `progress-store` — Code Generation 계획

**유닛** `progress-store` (U3) · **브랜치** `unit/progress-store` ·
**담당** taeels · **웨이브** W0 · **지는 게이트** 없다 (코드 게이트만) ·
**딛는 게이트** 없다 · **지는 값** N2

**이 계획이 Code Generation 의 정본이다. 여기 없는 것은 안 짓는다.**

정본 — 이 유닛의 앞 단계 산출물 일곱.
`construction/progress-store/functional-design/` 셋 (규칙 R1 ~ R36) ·
`nfr-requirements/` 둘 (값 V1 ~ V6 · 규칙 R37 ~ R42) ·
`nfr-design/` 둘 (규칙 R43 ~ R50). 회차 정본은
`aidlc-docs/v3-run-transcript/inception/`, 요구 팩은 `requirements/transcript/`.
**어긋나면 `enode-design` 이 이긴다** (`requirements/transcript/canon.md`).

---

## 0. 이 단계가 서 있는 자리

앞 단계 셋이 값을 전부 닫았다. **이 계획이 새로 고르는 값이 0 이고, 정하는
것은 「어떤 자료구조로 · 어느 파일에 · 어떤 차례로」다.**

```text
   Functional Design   형식과 규칙 R1 ~ R36.  물음 열하나의 답
   NFR Requirements    N2 의 값 V1 ~ V6 과 대가 R37 ~ R42.  물음 넷의 답
   NFR Design          순서와 안전장치 R43 ~ R50.  물음 0
   이 계획              자료구조 · 파일 · Step · 시험 · 재는 것.  5절이 「FD 에
                       없던 자리」를 따로 모은다 (선례 — `sources` 계획 4절)
```

**Part 2 는 사용자 승인 뒤에 연다.** 이 계획을 짓는 동안 코드를 한 줄도 안
고쳤다 — `go build` · `go vet` · `go test` 는 재기만 했고 6절의 숫자가 그
실측이다.

---

## 1. 이 유닛이 내는 diff 의 모양

### 1.1 파일

| 파일 | 새것/고침 | 무엇 | 규칙 |
|---|---|---|---|
| `internal/record/progress.go` | **새 파일** | `Progress` · `AppendProgress` · `ReadProgress` · `DropProgress` · `SweepProgress` · 뮤텍스 · 표시 줄 · 이름 파서 | R1 ~ R21 · R28 ~ R31 · R43 ~ R46 |
| `internal/record/record.go` | 고침 | `Seal` 의 첫 줄 (`DropProgress`) · `AppendLog` 의 반환값 뜻 | R22 · R27 · R47 · R48 |
| `internal/store/seal.go` | 고침 | `sealRecord` 가 `DropProgress` 를 부른다 | R23 |
| `internal/store/reap.go` | 고침 | `Reap()` 안의 고아 쓸기 하나 | R24 · R37 ~ R39 · V1 ~ V5 |
| `internal/record/progress_test.go` | **새 파일** | 이 유닛의 시험 대부분 | — |
| `internal/store/progress_test.go` | **새 파일** | 쓸기와 봉인 경로의 시험 (DB 를 탄다) | — |
| `aidlc-docs/taeels/construction/progress-store/code/code-summary.md` | **새 파일** | 코드 요약 (문서) | — |

**제품 파일은 넷이고 그중 새 파일이 하나다.** 시험 파일 둘과 요약 하나가 는다.

### 1.2 diff 0 이 품질 게이트인 경로 — 그리고 깨지는 하나

`execution-plan.md` 6절 품질 게이트 2 와 `unit-of-work-file-matrix.md` 5절이
재는 값이다.

```text
   internal/contract       코드 diff 0.  **그대로 지킨다**
   cmd/ 다섯               코드 diff 0.  **그대로 지킨다** — 물음 10 = B 가
                          ReadProgress 의 서명을 안 넓혀서 record.New 도
                          cmd/mediator/main.go 도 안 바뀐다
   internal/api            코드 diff 0.  **그대로 지킨다** — putLog 의 갈림은 U4 다
   internal/enode          코드 diff 0.  업로더는 U7 이다
   internal/panel · api/ui  코드 diff 0
   go.mod · go.sum         diff 0 (SECURITY-10 · 표준 라이브러리만 쓴다)
   internal/store/schema.sql  diff 0 — DB 스키마를 안 바꾼다
                          (`constraints.md` 6절이 명시로 뺐다)

   internal/store          **0 이 아니다.**  seal.go 와 reap.go 에 줄이 는다.
                          FD 10절이 확정으로 적은 자리이고 근거는 하나다 —
                          R23 과 R24 가 Run 의 상태를 알아야 하는데
                          internal/record 는 internal/store 를 임포트하지 않는다
                          (실측 — `grep -rn internal/store internal/record/` 가 0줄)
```

**게이트 2 의 기준선이 바뀌는 것이지 게이트가 빨간 것이 아니다.** 고치는 사람은
진행자다 (9절).

### 1.3 라우트 수는 안 는다

```text
   오늘      grep -c 'mux.HandleFunc' internal/api/api.go  ->  17  (실측)
   U3 뒤     17.  **안 는다** — 새 라우트는 U4 의 CB0 이 잰다
```

### 1.4 스토리 추적 — U3 은 재료다

`unit-of-work-story-map.md` 1절 · 2절이 낸 값이다.

| | 값 | U3 의 자리 | 지는 유닛 | 어느 Step 이 닫나 |
|---|---|---|---|---|
| FR-5 (med) | 원문 청크를 진행 파일에 — Mediator 쪽 | 진다 | U3 | Step 4 ~ Step 9 |
| US-5 | 봉인 전 원문인지 뒤 선별본인지 | 재료 | U4 | Step 9 · Step 14 (봉인 전후로 자리가 갈린다) |
| US-7 | 진행 로그가 상한에 닿았다 | 재료 | U4 | Step 5 · Step 6 · Step 16 |
| NC-4 | 상한 도달이 응답과 화면에 | 값을 만든다 | U4 | Step 5 (표시 줄) · Step 6 (`Capped`) |

```text
   NC-4 · US-7 이 사는지 죽는지가 이 유닛에 달렸다
     U3 이 Progress.Capped 를 안 내면 U4 가 지어내야 한다.  지어내는 길은
     크기 비교뿐이고 그것이 R14 가 금지한 자리다 (양쪽으로 다 틀린다).
     **Step 5 와 Step 6 이 그 둘의 생사를 진다**

   U3 이 지는 스토리는 0 이다
     빠뜨린 것이 아니다 — 바이트를 다루고 사람이 보는 표면을 안 만든다
     (`unit-of-work-story-map.md` 1.1)
```

### 1.5 디스크 봉투 — 숨기지 않는다

```text
   최악 10 GiB (R = 50 동시 Run · S = 20 단계)      **그대로 서 있다**
   V5 의 6시간이 자르는 것은 노출 시간이고 디스크가 아니다   (NFR Req 4.2)
   Mediator 가 자기 디스크를 안 본다               R40
   그 결과 진행 파일이 봉인을 굶길 수 있다            R41.  막는 코드가 0 이다
```

**이 유닛이 그것을 안 줄인다.** 물음 4 의 B 가 없애는 길이었고 대가가
`freeBytes` 두 벌이었다 — 사용자가 값을 보고 A 를 골랐다.

---

## 2. 못 박는 계약 — 두 갈래가 같은 글자를 본다

**여기서 짐작하지 않는다.** 서명 넷은 `component-methods.md` 2.1 이 이미 닫았고,
`SweepProgress` 와 상수 하나는 이 계획이 정한다 (5절 ①).

```go
// internal/record

type Progress struct {
	Total   int64 // 파일의 크기.  표시 줄도 든다 (R13)
	Attempt int   // 지금 시도.  파일 이름에서 읽는다
	Capped  bool  // 마지막 줄이 표시 줄이다 (R14)
}

func (s *Store) AppendProgress(runID string, seq int, name string,
	attempt int, r io.Reader, limit int64) (Progress, error)

func (s *Store) ReadProgress(runID string, seq int, name string,
	from int64) (Progress, io.ReadCloser, error)

func (s *Store) DropProgress(runID string) error

// SweepProgress 는 고아 진행 트리를 걷는다 (N2 · R24).
// live 는 살아 있는 Run 의 id 다 — 부르는 쪽(internal/store)이 DB 에서 낸다.
func (s *Store) SweepProgress(live []string, maxAge time.Duration,
	now time.Time) (int, error)

// ProgressMaxAge 는 살아 있는 Run 의 진행 파일이 디스크에 앉아 있을 수 있는
// 상한이다 — 마지막 쓰기부터 센다 (N2 의 V5).
const ProgressMaxAge = 6 * time.Hour
```

**`AppendLog` 의 서명은 안 바뀐다.** 반환값의 뜻만 바뀐다 (D4) —
「이번 호출의 바이트」에서 「붙인 뒤의 총 길이」로.

---

## 3. 병렬 — 파일이 안 겹치는 두 갈래

`obs` 계획 1절이 선례다. **가르는 축은 단계가 아니라 파일이다.**

```text
   갈래   패키지            파일                                      겹치는 파일
   ────   ───────────────   ───────────────────────────────────────   ───────────
   A      internal/record   progress.go (신규) · record.go ·           없다
                            progress_test.go (신규)
   B      internal/store    seal.go · reap.go ·                        없다
                            progress_test.go (신규)
```

**A 와 B 사이에 의존이 하나 있다** — B 가 A 의 `DropProgress` 와
`SweepProgress` 와 `ProgressMaxAge` 를 부른다. 그 계약은 2절이 못 박았으므로
**둘이 동시에 그 글자 위에서 짠다.**

```text
   A  ── 2절의 서명 ──>  B
```

**갈래를 더 못 나눈다.** `internal/record` 안에서 `AppendProgress` 와
`ReadProgress` 와 `DropProgress` 가 같은 뮤텍스 자료구조와 같은 이름 파서를
쓰므로 같은 파일에서 같은 시점에 컴파일된다 — `sources` 계획 3절과 같은 판정이다.

---

## 4. Step — 열아홉

**Part 2 가 이 체크박스를 채운다.** 갈래 표시는 `[A]` `[B]` `[—]` 다.

### Step 1 `[—]` — 계약을 못 박는다

- [x] 1.1 2절의 서명 여섯을 그대로 쓴다. **여기서 넓히지 않는다** — 넓히면
      `cmd/` 다섯의 diff 0 (1.2) 이 깨진다
- [x] 1.2 A 와 B 가 같은 글자를 보는지 확인하고 갈래 둘을 연다

### Step 2 `[A]` — `internal/record/progress.go` — 자리와 이름

- [x] 2.1 `progressRoot()` 와 `progressDir(runID)` —
      `filepath.Join(s.Root, "progress", "run-"+safe(runID))` (R1).
      **`safe()` 를 다시 짓지 않는다** (`record.go:38` 를 그대로 쓴다 ·
      `CONVENTIONS.md` 1.4)
- [x] 2.2 주석에 **왜 형제인가**를 적는다 — `seal` 의 `Walk` 이
      `record.go:159`, `Tar` 의 `Walk` 이 `record.go:229` 이고 둘 다
      `s.dir(runID)` 만 돈다. 진행 트리는 그 밖이다 (R35 · R36).
      **`s.Root` 를 도는 코드가 저장소에 0 개**라는 것도 적는다 (실측 —
      `s.Root` 가 나오는 자리는 `record.go:33` 의 `dir()` 하나다)
- [x] 2.3 이름은 `NN-<safe(name)>.<attempt>.log` (R2). `%02d` 와 `%d` 는
      `blobPath` (`record.go:270`) 의 규칙을 따른다
- [x] 2.4 `parseProgressName(n string) (seq, attempt int, name string, ok bool)` —
      오른쪽에서 읽는다 (R6). `.log` 를 떼고, 마지막 `.` 뒤를 `attempt` 로,
      첫 `-` 앞을 `seq` 로 읽는다. 못 읽으면 `ok=false` 이고 부르는 쪽이
      조용히 건너뛴다 (R34)
- [x] 2.5 주석에 **`parseBlobName` 을 재사용 못 하는 이유**를 적는다 —
      그쪽은 `%02d.%d-이름` 이라 왼쪽에서 읽는다 (`record.go:304`)
- [x] 2.6 권한 — 디렉터리 `0700` · 파일 `0600` (R4). 주석에 `logs/` 와 다른
      이유를 적는다 (`record.go:47` 의 `0755` · `record.go:59` 의 `0644`) —
      그쪽은 선별본이고 이쪽은 원문이다

### Step 3 `[A]` — 단계별 뮤텍스

- [x] 3.1 `Store` 에 필드 하나를 더한다. 오늘 `Store` 는 `struct{ Root string }`
      이므로 (`record.go:28`) **`New` 의 서명은 안 바뀐다** — 맵은 게으르게 난다
- [x] 3.2 자료구조는 두 겹이다 (5절 ②)

```go
type progressLocks struct {
	mu   sync.Mutex
	runs map[string]*runLocks
}

type runLocks struct {
	drop  sync.RWMutex          // R31.  쓰기와 읽기는 RLock · 드롭은 Lock
	mu    sync.Mutex            // steps 를 지킨다
	steps map[string]*sync.Mutex // 키는 (seq, name).  attempt 를 안 넣는다.  R28
	refs  int                   // 0 이 될 때만 runs 에서 걷는다.  R30
}
```

- [x] 3.3 `acquire(runID)` 가 `locks.mu` 안에서 `refs++` 하고,
      `release(runID)` 가 `locks.mu` 안에서 `refs--` 해 **0 일 때만 맵에서
      지운다**. 이것이 R31 이 이름 붙인 함정을 막는다 — 누군가 쥐고 있거나
      기다리는 동안에는 `refs >= 1` 이라 지워지지 않고, 그 창에 들어온 새
      호출은 같은 객체를 받는다
- [x] 3.4 주석에 그 함정을 문장으로 적는다 — 「뮤텍스를 맵에서 지우는 순간
      그것을 쥔 쓰기가 있으면 다음 호출이 새 뮤텍스를 만들어 둘이 같은 파일에서
      동시에 돈다」
- [x] 3.5 선례를 가리킨다 — `Ring` 이 같은 자리를 뮤텍스 하나로 풀었다
      (`internal/enode/transcript.go:40`)

### Step 4 `[A]` — `AppendProgress` — 순서 일곱

`business-logic-model.md` 1.1 의 ① ~ ⑦ 그대로 짓는다. 순서가 뜻을 가지므로
주석이 그 순서의 이유를 적는다.

- [x] 4.1 ① 키를 잠근다. ② ~ ⑦ 이 전부 이 안이다 (R28 · 1.4 의 셋)
- [x] 4.2 ② `MkdirAll(progressDir, 0700)`. 게으르다 — `Store.Open`
      (`record.go:45`) 을 안 건드린다 (R3). 주석에 이유를 적는다: `Seal` 이
      `s.Open` 을 부르므로 (`record.go:96`) `Open` 에 넣으면 지우기 직전에
      다시 만든다
- [x] 4.3 ③ 디렉터리를 훑어 그 `(seq, name)` 의 최대 시도를 읽는다 (R5 · R6).
      곁파일이 0 이라 이름이 진실이다
- [x] 4.4 ④ 세 갈래 —
      `attempt < 지금` 버린다 (0 바이트 · 지금의 `Progress`) R8 ·
      `attempt == 지금` 이어 붙인다 R7 ·
      `attempt > 지금` 앞 시도 파일을 전부 걷고 새 이름으로 연다 R9 · R16
- [x] 4.5 ④ 의 셋째에서 **걷기가 실패해도 진행을 안 막는다** (R25). 걷는 범위는
      같은 `(seq, name)` 의 더 작은 `attempt` 뿐이다 — 다른 단계도 다른 Run 도
      안 건드린다
- [x] 4.6 ⑤ 이미 닿았으면 0 바이트를 쓰고 ⑦ 로 (R15). **판정의 근거는 표시
      줄의 존재다. 크기가 아니다** (R14)
- [x] 4.7 ⑥ 두 갈래 — Step 5
- [x] 4.8 ⑦ `Total` 은 쓰기 직후의 파일 크기다. 락 안에서 잰다 (R13 · 7.2)

### Step 5 `[A]` — 상한에 닿는 순간 — 개행 보장과 표시 줄

`nfr-design-patterns.md` 1.1 의 ① ~ ⑦ 그대로다. **쓰기가 둘이고 그 앞에 개행
보장이 하나 붙는다.**

- [x] 5.1 상한 안이면 청크를 통째로 쓴다. 개행에 안 맞춘다 (R10).
      주석에 이유를 적는다 — 맞추면 `Total` 이 노드가 보낸 것보다 뒤처지고
      노드가 그 차이를 다시 보내 루프가 돈다
- [x] 5.2 넘기면 `budget = limit - 지금 크기` 를 재고 그 안의 마지막 개행까지만
      쓴다 (R11). **예산 안에 개행이 없으면 0 바이트를 쓴다**
- [x] 5.3 **[R43] 개행을 보장한다** — 쓰고 난 파일의 마지막 바이트가 개행이
      아니면 개행 하나를 먼저 쓴다. 주석에 **없으면 무슨 일이 나는지**를 적는다:
      반쪽 줄이 표시 줄과 붙어 한 줄이 되고, 파싱이 안 되고, R14 의
      「마지막 줄이 표시 줄인가」가 거짓이 되고, **`Capped` 가 영영 참이 안 되어
      표시 줄이 무한히 쌓인다**
- [x] 5.4 **[R44] 표시 줄과 그 개행을 한 번의 쓰기로** 낸다. 버퍼를 나누지 않는다
- [x] 5.5 표시 줄을 만드는 자리를 `elidedMark` 의 모양으로 짓는다
      (`internal/enode/runner.go:415` ~ `:429`) — 구조체 하나와 마샬 함수 하나

```go
// cappedMark 는 총 길이 상한에 닿았음을 표시하는 줄이다 (R12).
//
// type 에 점을 넣는다 — 실측한 하네스의 type 은 전부 홑단어라
// (system · assistant · user · result · rate_limit_event) 부딪칠 수 없다.
// elidedMark 가 같은 근거로 enode.elided 를 쓴다 (runner.go:415).
//
// bytes 는 상한이다. 잘린 자리가 아니다 — 멈춘 자리는 Total 이 이미 말한다.
type cappedMark struct {
	Type  string `json:"type"`
	Bytes int64  `json:"bytes"`
}
```

- [x] 5.6 `Bytes` 가 `int64` 인 것이 `elidedMark` 의 `int` 와 다른 유일한 자리다.
      근거를 주석에 적는다 — `limit` 이 `int64` 다 (`config.go:51` 의
      `MaxBlobBytes int64`)
- [x] 5.7 `Capped` 를 마지막 줄로 판정한다 (R14). 꼬리 한 번 읽기면 족하다 —
      R15 가 「표시 줄이 언제나 마지막」을 보장한다
- [x] 5.8 **`Total > limit` 이 정상이다** (R45). 주석이 그것을 적고 Step 13 의
      시험이 못 박는다

### Step 6 `[A]` — `ReadProgress`

`business-logic-model.md` 2.1 의 ① ~ ⑤ 그대로다.

- [x] 6.1 ① 키를 잠근다 (쓰기와 같은 키) · ② 최대 시도를 고른다 (R5).
      **`attempt` 인자가 없다** — 그것이 R17 을 이 유닛 안에서 참으로 만든다
- [x] 6.2 ③ `Capped` 를 마지막 줄로 판정한다 (R14)
- [x] 6.3 ④ 세 갈래 — 파일이 없으면 `Total 0 · Attempt 0 · Capped false` 에
      빈 본문이고 404 가 아니다 (R18) · `from >= Total` 이면 빈 본문 ·
      `from < Total` 이면 `from` 부터
- [x] 6.4 **`from > Total` 도 오류가 아니다** (R19). 총 길이가 뒤로 간 뒤의
      폴링이 그 모양이다
- [x] 6.5 본문을 **`Total - from` 에서 자른다** (5절 ③). `io.LimitReader` 로
      감싸고 `Close` 가 파일을 닫는 작은 타입 하나를 둔다. 주석에 근거를 적는다 —
      안 자르면 본문이 `Total` 보다 길어져 폴링하는 쪽이 다음 `from` 을
      `Total` 로 잡았을 때 **같은 바이트를 두 번 받는다.** CB3 이 재는 것이
      「두 벌이 안 생긴다」다
- [x] 6.6 본문은 락 밖에서 읽힌다. 그 사이에 `DropProgress` 가 돌아도
      리눅스에서 열린 fd 는 살아 있으므로 읽기가 안 깨진다 — 주석에 적는다

### Step 7 `[A]` — `DropProgress`

- [x] 7.1 ① 그 Run 의 `drop` 을 Lock 한다 (R31) · ② 진행 트리를 통째로
      지운다 (`os.RemoveAll`) · ③ 뮤텍스 맵에서 그 Run 을 걷는다 (R30)
- [x] 7.2 멱등이다 (R21). 없으면 아무것도 안 하고 `nil` 을 낸다.
      `os.RemoveAll` 이 이미 그 성질이다
- [x] 7.3 **Run 단위다. 단계 단위가 아니다** — 봉인도 종료도 쓸기도 Run 단위로
      일어난다

### Step 8 `[A]` — `SweepProgress` — 고아 쓸기

- [x] 8.1 **디렉터리를 먼저 읽는다.** `<Root>/progress/` 가 없거나 비었으면
      **0 을 내고 끝낸다** — 부르는 쪽이 DB 를 안 치게 한다 (5절 ④)
- [x] 8.2 `live` 를 `safe()` 로 접어 집합을 만든다. **앞으로 가는 사상만
      쓴다** — `safe()` 는 되돌릴 수 없으므로 디렉터리 이름에서 `runID` 를
      복원하지 않는다 (5절 ⑤)
- [x] 8.3 **조건 둘은 배타이고 순서가 규칙이다** (R37) — **Run 상태를 먼저
      보고 종료가 아닐 때만 나이를 본다**. 뒤집으면 종료된 Run 의 트리가 6시간
      남아 V4 = 0 과 어긋난다
- [x] 8.4 조건 ① — 살아 있는 집합에 없는 디렉터리는 통째로 지운다.
      유예 0 (V4). **DB 에 없는 Run 도 여기로 온다** (R38)
- [x] 8.5 조건 ② — 살아 있는 Run 의 디렉터리에서 파일마다 mtime 을 보고
      `now.Sub(mtime) > maxAge` 면 그 파일만 지운다 (V5 · 3.2).
      같은 Run 의 방금 쓴 다른 단계 파일은 안 건드린다
- [x] 8.6 조건 ② 는 그 파일의 `(seq, name)` 뮤텍스를 잡고 지운다.
      **이름이 안 읽히면 건너뛴다** (R34) — 키를 못 만들기 때문이고, 그 Run 이
      끝나면 조건 ① 이 통째로 걷는다
- [x] 8.7 조건 ① 은 그 Run 의 `drop` 을 Lock 하고 지운다 — `DropProgress` 와
      같은 경로를 쓴다
- [x] 8.8 지운 수를 돌려준다. 실패는 보조다 (R25) — 한 항목이 실패해도
      나머지를 계속 돈다
- [x] 8.9 **`Record` 를 안 건드린다** (R39). 주석에 적는다 — 구멍 ① ② 가 낳는
      「종료했는데 안 봉인된 기록」은 이 유닛의 것이 아니다

### Step 9 `[A]` — `internal/record/record.go` — 두 자리

- [x] 9.1 **`Seal` 의 첫 줄** (R22 · R47) — `if s.Sealed(runID) { return nil }`
      **보다 앞이다**

```go
func (s *Store) Seal(runID string, manifest, verdict any, steps []StepFile) error {
	_ = s.DropProgress(runID)   // R47 · R48.  실패해도 봉인을 안 막는다
	if s.Sealed(runID) {
		return nil
	}
	...
	return seal(d)              // record.go:120.  여기를 지나면 못 되돌린다
}
```

- [x] 9.2 주석에 **막아야 할 것이 `seal(d)` 의 chmod** 임을 적는다 —
      `Seal` 은 tar 를 안 짓는다 (`record.go:120` 에서 끝난다). `Tar` 는
      `GET /v1/runs/{id}/record` 때 돈다 (`api.go:1043`, `Sealed()` 게이트
      뒤 — `api.go:1037`). 0444 는 `record.go:174`, 0555 는 `record.go:180`
- [x] 9.3 주석에 **조기 반환보다 앞인 이유**를 적는다 — 늦은 청크의 고아 트리
      (t0 ~ t4). `putLog` 가 DB 의 `run.State` 로 410 을 내고 `Sealed()` 를
      안 본다 (`api.go:879`)
- [x] 9.4 선례를 가리킨다 — `.tmp-*` 를 걷는 줄이 `seal(d)` 바로 앞에 있다
      (`record.go:112` ~ `:119`). 「굳히기 전에 걷는다」가 이 함수의 기존 결이다.
      **`DropProgress` 는 그보다 더 앞이다** (이미 봉인된 Run 에도 돌아야 한다)
- [x] 9.5 **`AppendLog` 의 반환값** (D4 · R27) — 쓴 바이트와 총 길이를 **갈라
      든다**

```go
written, err := io.Copy(f, io.LimitReader(r, limit))
if err != nil { ... }
if written == limit { ... 표시 ... }   // R27.  판정은 written 으로 한다
// 총 길이를 돌려준다 (D4) — 표시 줄까지 든 파일의 크기다
```

- [x] 9.6 주석에 함정을 적는다 — `total` 로 판정하면 재시도가 깨진다.
      `logs/NN-*.log` 는 시도가 이름에 없어 이어 붙기 때문이다
      (`record.go:58` · `internal/enode/claim.go:146` 의 URL 에 `attempt` 가 없다)
- [x] 9.7 **`n == limit` 의 오판은 안 고친다** (FD 6.3). 그 자리에 주석 한 줄로
      **잔여임을 적고** 9절이 그것을 넘긴다 — 이 회차가 안 산 변경이다
- [x] 9.8 **오늘 `AppendLog` 의 반환값을 읽는 호출자가 0 이다** (실측 — 제품
      코드는 `api.go:887` 하나이고 `if _, err :=` 로 버린다. 시험 셋도 전부
      `_` 다). **그래서 뜻을 바꿔도 빨개지는 시험이 0 이다** — 그 사실을
      `code-summary.md` 에 적는다

### Step 10 `[B]` — `internal/store/seal.go`

- [x] 10.1 `sealRecord` 의 **`s.Records == nil` 검사 바로 뒤**에
      `DropProgress` 를 부른다 (`seal.go:176` 의 다음). **`GetRun` 과
      `StepFiles` 보다 앞이다** — 그 둘이 실패하면 `Seal` 이 아예 안 불리고
      (`seal.go:179` ~ `:186`) 트리가 남는다. R23 이 그 자리다
- [x] 10.2 실패는 삼키고 로그로 낸다 (R25 · R48). `s.log()`
      (`store.go:65`) 를 쓴다. 메시지는 영어다 (`CONVENTIONS.md` 2.1)
- [x] 10.3 주석에 **`Seal` 의 성공과 무관하게** 부르는 이유를 적는다

### Step 11 `[B]` — `internal/store/reap.go`

- [x] 11.1 쓸기를 **`Reap()` 안**에 둔다. **`sealExpired` 안이 아니다** (V2) —
      `sealExpired` 는 `if n > 0` 뒤라 (`reap.go:99`) 그 주기에 회수가 0 이면
      아예 안 훑는다
- [x] 11.2 자리는 `if n > 0 { ... }` 블록 뒤, `return n, nil` 앞이다.
      **조건 없이 매 주기 돈다**
- [x] 11.3 살아 있는 Run 을 낸다 —
      `SELECT run_id FROM runs WHERE state NOT IN ('SUCCEEDED','FAILED')`
- [x] 11.4 `s.Records.SweepProgress(live, record.ProgressMaxAge, time.Now())`
- [x] 11.5 **실패가 `Reap` 을 안 죽인다** (R25). 로그만 내고 `n` 을 안 바꾼다
- [x] 11.6 **새 타이머 0 · 새 고루틴 0 · 새 설정 키 0** (V1). 이미 도는
      `RunReaper` 에 얹는다 — `cmd/mediator/main.go:89` 가
      `cfg.Lease.RenewSeconds` 로 띄우고 `reap.go:158` 의 `time.NewTimer(0)` 이
      **기동에 한 번 먼저 돌게** 한다. R24 의 「기동과 주기마다」가 글자 그대로
      이미 거기 있다
- [x] 11.7 `s.Records == nil` 이면 건너뛴다 — `reap.go:114` 의 결과 같다

### Step 12 `[A]` — 시험 — 자리 · 이름 · 시도 · 뮤텍스

새 파일 `internal/record/progress_test.go`.

- [x] 12.1 R1 — 진행 파일이 `<Root>/progress/run-<id>/` 에 나고
      `<Root>/run-<id>/` 안에는 0 개다
- [x] 12.2 R2 · R6 — `01-build.step.2.log` 를 넣으면 단계 이름 `build.step` ·
      시도 `2` 가 나온다. **왼쪽 파싱이면 이 시험이 빨개진다**
- [x] 12.3 R4 — 파일 `0600` · 디렉터리 `0700` 을 `Stat` 으로 잰다
- [x] 12.4 R3 — `Store.Open` 만 부르면 `<Root>/progress/` 가 안 생긴다
- [x] 12.5 R7 · R8 — 같은 시도는 이어 붙고, 작은 시도는 **0 바이트를 쓰고
      지금의 `Progress`** 를 낸다
- [x] 12.6 R9 · R16 · R17 — 시도 1 을 쓰고 시도 2 를 쓰면 `ReadProgress` 가
      시도 2 의 바이트만 내고 시도 1 의 파일이 디스크에 없다
- [x] 12.7 R18 — 파일이 없는 단계는 `Total 0 · Attempt 0 · Capped false` 에
      빈 본문이다. 오류가 아니다
- [x] 12.8 R19 · R20 — 경로 둘의 표시가 다르다. 시도가 바뀌면 `Attempt` 가
      바뀌고, 트리가 걷히면 `Total` 과 `Attempt` 가 함께 0 이다
- [x] 12.9 R21 — `DropProgress` 를 두 번 불러도 오류가 0 이다
- [x] 12.10 R29 — **키가 다르면 안 기다린다.** 단계 둘에 동시에 쓰고 둘 다
      끝나는지 잰다
- [x] 12.11 R28 · R30 · R31 — 고루틴 여럿이 같은 키에 쓰는 동안 `DropProgress`
      를 섞는다. **`go test -race` 로 잰다** (6절 3번)
- [x] 12.12 R32 — `TestPathTraversalIsBlocked` (`record_test.go:300`) 의 결을
      진행 트리에도 건다. `../../etc/passwd` 가 `____etc_passwd` 로 접힌다
- [x] 12.13 R6 · R34 — 못 읽는 이름을 디렉터리에 두고 오류가 0 인지 잰다

### Step 13 `[A]` — 시험 — 상한 · 개행 보장 · 틈 넷

- [x] 13.1 R10 — 상한 안의 청크가 통째로 들어간다 (개행에 안 맞춘다)
- [x] 13.2 R11 — 상한을 넘기면 마지막 개행까지만 남고 반쪽 줄이 0 이다
- [x] 13.3 **R14 의 핵심 시험** — 개행이 없는 큰 청크 하나로 상한을 넘기면
      `Total < 상한` 인데 `Capped` 가 참이다. **크기 비교로 적은 구현은
      여기서 빨개진다**
- [x] 13.4 R12 · R15 — 표시 줄이 정확히 한 번이고 언제나 마지막 줄이다.
      닿은 뒤에 또 PUT 이 와도 안 자란다
- [x] 13.5 **R43 의 시험** — 꼬리가 반쪽 줄인 파일에 상한을 넘기면, 표시 줄
      앞에 개행이 하나 들어가 마지막 줄이 온전한 표시 줄이 된다.
      **개행 보장을 걷으면 이 시험이 빨개진다**
- [x] 13.6 틈 넷 (`nfr-design-patterns.md` 1.3) — 네 자리에서 멈춘 파일을
      손으로 지어 놓고 다음 호출이 **전부 「표시 줄이 온전히 마지막 줄」로
      수렴**하는지 잰다. 특히 셋째 (표시 줄이 반쪽으로 남은 파일) 에서 R43 이
      그 조각을 개행으로 닫는지
- [x] 13.7 R45 — `Total > limit` 을 **정상으로 못 박는 시험**을 둔다.
      `Total <= limit` 을 불변식으로 적지 않는다. 주석이 그 이유를 적는다 —
      **이 시험이 없으면 다음 사람이 `assert Total <= limit` 을 쓴다**
- [x] 13.8 R13 — `Total` 이 언제나 `os.Stat` 의 크기와 같다. 표시 줄과 R43 의
      개행을 포함해서
- [x] 13.9 R27 — `AppendLog` 을 상한의 절반씩 두 번 불러 **총 길이가 상한과
      같아져도** 잘림 표시가 **안 붙는지** 잰다. 총 길이로 판정한 구현은 여기서
      빨개진다. 그리고 반환값이 총 길이인지 잰다 (D4)

### Step 14 `[A]` — 시험 — 봉인 경로 불변식

- [x] 14.1 R35 — 진행 파일을 쓴 Run 을 봉인하고 `Tar` 를 풀어
      `progress/` 항목이 0 개인지 잰다. `TestTarIsSelfSufficient`
      (`record_test.go:109`) 의 결을 따른다
- [x] 14.2 R36 — 봉인 뒤에도 진행 트리를 지울 수 있는지 잰다
      (쓰기 비트가 살아 있다). `newStore` 의 `unseal` 정리
      (`record_test.go:17` ~ `:22`) 를 그대로 쓴다
- [x] 14.3 R22 · R47 — 봉인이 진행 트리를 지운다. 그리고 **이미 봉인된
      Run 에 다시 `Seal` 을 불러도** 늦은 청크가 만든 트리를 지운다
      (조기 반환 뒤에 두면 이 시험이 빨개진다)
- [x] 14.4 R48 — `DropProgress` 가 실패하는 상황에서도 봉인이 선다.
      진행 트리 자리에 지울 수 없는 것을 놓아 실패를 만들고 `Seal` 이
      `nil` 을 내는지 잰다
- [x] 14.5 `TestSealIsIdempotent` (`record_test.go:62`) 가 그대로 초록인지

### Step 15 `[B]` — 시험 — 쓸기 (DB 를 탄다)

새 파일 `internal/store/progress_test.go`.

- [x] 15.1 V4 · R37 — 종료 상태 Run 의 진행 트리가 `Reap()` 한 번에 사라진다
- [x] 15.2 R38 — DB 에 없는 Run 의 트리가 걷힌다
- [x] 15.3 V5 · 3.2 — 살아 있는 Run 의 6시간 넘은 파일이 걷히고
      **같은 Run 의 방금 쓴 파일은 안 걷힌다**. `now` 인자를 앞으로 당겨 잰다 —
      슬립이 0 이다
- [x] 15.4 R37 의 순서 — 종료 Run 의 방금 쓴 파일도 걷힌다 (나이를 먼저
      보는 구현은 여기서 빨개진다)
- [x] 15.5 R39 · R35 — 쓸기 뒤에도 `Record` 가 그대로다. `Tar` 출력에
      `progress/` 가 0 개다
- [x] 15.6 V2 — 쓸기가 **회수가 0 인 주기에도** 돈다 (`sealExpired` 안이면
      이 시험이 빨개진다)
- [x] 15.7 R23 — `StepFiles` 가 실패해 `Seal` 이 안 불리는 경로에서도
      트리가 걷힌다
- [x] 15.8 8.1 — `<Root>/progress/` 가 없으면 DB 질의가 0 번 돈다

### Step 16 `[—]` — 이음매 — `enode.capped` 를 어디서 박고 무엇으로 맞대나

**한 패키지 왕복 시험이 이것을 못 잡는다.** 짓는 쪽이 U3 이고 읽는 쪽이 U1
(`internal/transcript`) 이라 **두 패키지가 한 바이너리에 처음 드는 것은 U4** 다.

- [x] 16.1 박는 자리 — `internal/record/progress.go` 의 `cappedMark`
      구조체 하나와 마샬 함수 하나 (Step 5.5). 이 저장소에서 그 글자가 나는
      자리는 여기 하나다
- [x] 16.2 못 박는 바이트 — 실측이다 (2026-09-15)

```text
   기본 상한 10 MiB    {"type":"enode.capped","bytes":10485760}    40 바이트
   개행을 더해         41 바이트
   가장 긴 경우        51 + 1 = 52 바이트   (bytes 가 int64 의 큰 값일 때)
```

- [x] 16.3 **맞대는 법 하나 — 글자 그대로의 시험.** `progress_test.go` 가
      마샬 결과를 위 40 바이트와 한 글자도 안 다른지 잰다. 필드 이름이나
      순서나 타입이 바뀌면 빨개진다. `Total` 이 41 만큼 는 것도 함께 잰다
- [x] 16.4 **맞대는 법 둘 — 값으로 넘긴다.** 16.2 의 글자를 9절이 진행자에게
      넘긴다. U1 이 `Kind` 를 지을 때 **짐작 안 하게** 한다
- [x] 16.5 **맞대는 법 셋 — 왕복은 U4 의 자리다.** `as=events` 가 U1 의 파서를
      타므로 거기가 처음으로 찍는 쪽과 읽는 쪽이 같은 바이너리에 드는 자리다.
      **U3 은 그 왕복을 못 잰다 — 그 사실을 숨기지 않는다**
- [x] 16.6 **맞대는 법 넷 — 사람 눈은 CB3 이다**
- [x] 16.7 **이음매가 늦게 닫혀도 데이터가 안 죽는다** — U1 이 그 글자를
      모르면 `raw` 사건 하나로 뜬다. 버리지도 실행하지도 않는다
      (`requirements.md` 5.6 ③ · FR-3). **표시의 질이 떨어지는 것이지 손실이
      아니다.** 그 사실을 `code-summary.md` 에 적는다
- [x] 16.8 **U3 이 안 짓는 것을 다시 적는다** (R46) — `Kind` · 파싱 규칙 ·
      헤더 이름 (`X-Enode-Log-Capped`) · 화면 문장. U3 의 경계는
      **`Progress` 를 반환하는 순간** 끝난다

### Step 17 `[—]` — 변이 여덟

넣고 빨개지는지 본다. 안 빨개지면 그 규칙을 재는 시험이 없는 것이다
(선례 — `sources` 계획 Step 8).

- [x] 17.1 ① R43 의 개행 보장을 걷는다
- [x] 17.2 ② R14 를 크기 비교(`Total >= limit`)로 바꾼다
- [x] 17.3 ③ R6 의 오른쪽 파싱을 왼쪽으로 바꾼다
- [x] 17.4 ④ R9 의 앞 시도 걷기를 걷는다
- [x] 17.5 ⑤ R22 의 `DropProgress` 를 조기 반환 뒤로 옮긴다
- [x] 17.6 ⑥ R27 의 잘림 판정을 총 길이로 바꾼다
- [x] 17.7 ⑦ R37 의 순서를 뒤집는다 (나이를 먼저 본다)
- [x] 17.8 ⑧ R1 의 자리를 `<Root>/run-<id>/progress/` 로 옮긴다 —
      14.1 과 14.2 가 둘 다 빨개져야 한다

### Step 18 `[—]` — 게이트

6절의 순서 그대로 돈다.

- [x] 18.1 6절의 1 ~ 10 을 순서대로
- [x] 18.2 안 초록인 것이 있으면 **그대로 적는다.** 초록이라고 안 적는다

### Step 19 `[—]` — 문서

- [x] 19.1 `construction/progress-store/code/code-summary.md` —
      만든 파일 · 고친 파일 · 잰 값 · 16.2 의 글자 · 못 닫은 것
- [x] 19.2 이 계획의 체크박스를 전부 `[x]` 로
- [ ] 19.3 `aidlc-docs/taeels/aidlc-state.md` 의 U3 절과 `audit.md` —
      **Part 2 에서** 쓴다. 이 단계(Part 1)의 커밋에는 안 싣는다

      **안 채운 하나다.** 이어받은 회차의 진행자가 그 둘을 회차 브랜치에서
      직접 쓰기로 정하고 이 유닛에 안 건드리게 걸었다. 계획이 적은 소유자와
      실제 소유자가 갈린 자리이고, **안 한 것을 한 것으로 적지 않는다.**
      남은 자리는 진행자다 (요약 8절)
- [x] 19.4 회차 문서와 팩은 **안 고친다.** 9절이 진행자에게 넘길 글자만 적는다
      (`CONVENTIONS.md` 3.4 의 「안 싣는 것」)

---

## 5. 이 계획이 정하는 것 — 앞 문서에 없던 자리 다섯

```text
   ①  고아 쓸기의 파일시스템 순회는 internal/record 가 진다

      FD 6.3 이 「record 는 DropProgress 만 내주고 부르는 쪽이 store 에 앉는다」
      고 적었다.  글자 그대로면 store 가 순회해야 하는데 **못 한다** —
      safe() 가 비공개이고 (record.go:38) 경로 규칙도 record 의 것이다.
      갈리는 자리는 「정하는 것」과 「도는 것」이다: 살아 있는 Run 이
      무엇인지는 store 가 알고(DB), 그 트리가 어디 있는지는 record 가 안다.
      그래서 SweepProgress(live, ...) 로 가른다 — 정책은 store, 기제는 record.
      FD 6.3 의 문장은 그대로 참이다.  Run 의 상태를 아는 쪽이 store 다

   ②  뮤텍스는 두 겹이고 참조 수로 걷는다

      FD 4.3 이 「자료구조는 Code Generation 이 고른다」로 남긴 자리다.
      키 하나짜리 맵으로는 DropProgress 가 「그 Run 의 키 전부」를 못 잡는다 —
      맵을 훑는 사이에 새 키가 날 수 있다.  Run 마다 RWMutex 를 두면
      드롭(Lock)이 그 Run 의 모든 쓰기(RLock)를 배타로 막는다.  참조 수는
      R31 의 함정을 막는다 (3.3)

   ③  ReadProgress 의 본문을 Total - from 에서 자른다

      FD 2.2 가 「from 부터 끝까지」로만 적었다.  본문은 락 밖에서 읽히므로
      자르지 않으면 끝이 Total 보다 길어질 수 있고, 폴링하는 쪽이 다음 from 을
      Total 로 잡았을 때 같은 바이트를 두 번 받는다.
      **CB3 이 재는 것이 「두 벌이 안 생긴다」다** — 그 조각이 이 갈림을 정한다

   ④  쓸기는 디렉터리를 먼저 읽고, 비어 있으면 DB 를 아예 안 친다

      NFR Requirements 3.3 이 「Reap() 안의 쓸기 하나」까지만 적었다.
      질의는 SELECT run_id FROM runs WHERE state NOT IN (...) 이고
      **runs.state 에 인덱스가 없다** (실측 — schema.sql 의 인덱스 넷 중
      state 를 타는 것은 runs_queued_idx 하나이고 WHERE state = 'QUEUED'
      부분 인덱스다).  60초마다 전수 스캔이 도는 것을 디렉터리 검사가 막는다 —
      진행 트리가 0 이면 (U4 · U7 이 서기 전의 모든 순간이 그렇다) 질의가 0 번이다.
      **인덱스를 더하지 않는다** — schema.sql 은 diff 0 이 규칙이다
      (constraints.md 6절).  남는 대가는 9절이 넘긴다

   ⑤  safe() 를 되돌리지 않는다 — 앞으로 가는 사상만 쓴다

      디렉터리 이름에서 runID 를 복원하면 "/" 를 담은 runID 에서 틀린 답이
      나오고, 그 틀림의 방향이 **살아 있는 Run 의 트리를 지우는 쪽**이다.
      살아 있는 Run 들을 safe() 로 접어 집합을 만들고 그 밖을 고아로 본다.
      충돌(두 runID 가 같은 safe 이름)이 나면 살아 있는 쪽이 이긴다 —
      안 지우는 쪽으로 틀린다
```

---

## 6. 재는 것 — 순서가 규칙이다

**1 ~ 10 을 이 순서로 돈다.** 숫자는 전부 이 계획을 짓는 동안 실측했다
(2026-09-15 · `unit/progress-store` · `ef4d766`).

```text
    1  eval "$(scripts/testdb.sh)"
    2  go build ./...  &&  go vet ./...                    오늘 둘 다 초록
    3  go test -race ./internal/record/...                 R31 을 재는 자리
    4  go test ./...                                       오늘 rc=0
    5  go run ./scripts/glyphscan.go                       오늘 초록
    6  test -z "$(gofmt -l .)"                             오늘 빈다
    7  test -z "$(git status --porcelain)"                 커버리지보다 먼저다
    8  go test ./... -count=1 -coverpkg=./... -coverprofile=... + 패키지별 80 하한
    9  git checkout -- cmd/enodectl/probe.lock             8 이 그것을 바꾼다
   10  grep -c 'mux.HandleFunc' internal/api/api.go        17 그대로여야 한다
```

### 6.1 커버리지 예산 — 실측

`-coverpkg=./...` 로 DB 를 붙여 잰 값이다. 전체 **87.4%** (7,123/8,146).

| 패키지 | 오늘 | 새 문장 `n` 개에 대해 **안 덮어도 되는 것** |
|---|---|---|
| `internal/record` | **82.4%** (131/159) | `0.2n + 3.8` |
| `internal/store` | **82.5%** (1,453/1,762) | `0.2n + 43.4` |

```text
   record 의 여유가 3.8 문장이다 — 얇다
     이 유닛이 새 문장을 가장 많이 넣는 패키지가 바로 거기다.
     오류 가지(MkdirAll · OpenFile · Stat 의 실패)가 커버리지의 늘 새는 자리다

   새 t.Skip 을 안 만든다
     .ci-allowed-skips 가 비어 있고 빈 것이 결론이다.  그래서 권한으로
     실패를 만드는 시험을 안 쓴다 (root 로 돌면 안 발동한다)

   권한 없이 오류를 만드는 길 둘
     MkdirAll 실패    <Root>/progress 자리에 파일 하나를 놓는다 (ENOTDIR)
     OpenFile 실패    그 파일 이름 자리에 디렉터리 하나를 놓는다
     둘 다 t.TempDir() 하나면 되고 권한도 root 도 안 탄다
```

### 6.2 품질 게이트 (`execution-plan.md` 6절) — 이 유닛이 재는 값

| | 게이트 | 이 유닛의 값 |
|---|---|---|
| 1 | 차단 게이트 다섯 | 심볼 상한 둘은 `internal/record` 가 `net/http` · `crypto/tls` 를 안 쓰므로 안 움직인다. 커버리지 80% 는 6.1. 스킵 0. U+2605 0 |
| 2 | 라우트 17 · `store` 와 `contract` 의 diff 0 | 라우트 17 그대로. `contract` 0 그대로. **`store` 는 0 이 아니다** (1.2) |
| 3 | 경계 검사 여섯 | **이 유닛이 안 건드린다.** U1 의 자리다 (`file-matrix` 6.3) |
| 4 · 5 | CB1 ~ CB6 사람 눈 | **U3 이 지는 조각이 0 이다.** CB3 의 재료다 |
| 6 | 새 로그가 본문을 안 싣는다 | 싣지 않는다 — 짓는 로그가 쓸기와 드롭 실패 둘이고 둘 다 종류와 수만 적는다 (SECURITY-03) |
| 7 | 장식 문자 0 | **오늘 `main` 에서 이미 빨갛다** (6.3) |
| 8 | 언어 규약 | 에러 · 로그 · 시험 이름은 영어. 주석은 한국어 |

### 6.3 게이트 7 은 오늘 이미 빨갛다 — 실측을 적어 둔다

```text
   잰 것     grep -rn '\*\*[^*]\+\*\*' --include=*.go internal/ cmd/ scripts/
   나온 것    21 자리 · 파일 열.  **그러나 21 이 전부 위반은 아니다** — 아래가 그 갈림
   재는 도구  **0 이다.**  scripts/glyphscan.go 는 문자열 · rune 리터럴 안의
             유니코드 장식 문자만 본다 (glyphRanges 열하나).  주석도 안 보고
             ASCII 두 별표도 안 본다

   그래서    게이트 7 의 절반에 **도는 스크립트가 0 이다**.  CLAUDE.md 가
             「도는 스크립트가 없는 채로 기계 검사라고 적으면 그것이 거짓
             초록이다」로 적은 바로 그 자리다
```

**21 을 가른다 — 위반 열다섯 · 위반 아님 여섯.** 가르는 기준은 **렌더링되느냐**다.

```text
   위반 (1.2)         **열다섯 · 파일 여덟.  전부 Go 주석 안이다**
     internal/api/refusal_test.go             2
     internal/api/log_test.go                 2
     internal/api/api_test.go                 1
     internal/config/load_test.go             2
     internal/config/write_test.go            1
     internal/record/record_test.go           2      **U3 이 여는 패키지 안이다**
     internal/store/reap_deterministic_test.go 4     같다
     internal/build/build_test.go             1
     왜 위반인가   Go 의 주석에는 강조 문법이 없다.  렌더링이 없는 자리의
                  장식 문자이고 CONVENTIONS.md 1.2 가 그것을 금지한다

   위반 아님 (1.1)    **여섯 · 파일 둘.  전부 문자열 리터럴 안이다**
     cmd/iapadapter/comment.go       3   b.WriteString 으로 코멘트 본문을 짓는다
     cmd/iapadapter/comment_test.go  3   그 본문을 그대로 대조한다
     왜 아닌가     **GitHub 코멘트 본문으로 나가는 마크다운이고 실제로
                  렌더링된다.**  그 자리는 1.1 이 다스리고 1.1 이 고르라고 한
                  수단이 바로 **굵게** 다.  지우면 코멘트의 굵게가 사라진다 —
                  규약을 지키려다 1.1 을 어기는 것이다
```

```text
   제외를 담은 꼴 (그대로 돌아간다.  15 · 파일 여덟이 나온다)
     grep -rn '\*\*[^*]\+\*\*' --include=*.go internal/ cmd/ scripts/ \
       | grep -v '^cmd/iapadapter/'

   U3 이 하는 것    **새로 안 만든다.**  이 유닛이 짓는 코드와 주석에 0 개
   U3 이 안 하는 것  기존 열다섯을 고치는 것.  **여덟 파일이 전부 이 유닛의 파일
                    행렬 밖이다** (1.1 의 표에 그 여덟이 0 개다).  그중 둘만
                    U3 이 여는 패키지 안에 있고, 그래도 고치면 게이트 2 의
                    diff 0 이 깨진다
   어디로 가나       **진행자다** (9.4)
```

---

## 7. 이 단계가 안 하는 것

```text
   HTTP · 라우트 · 헤더 · 검증     U4.  X-Enode-Log-* 넷과 SECURITY-05 의 검증 넷
   Kind · 파싱 · 화면 문장         U1 · U5 · U8.  U3 은 찍는 쪽만 진다 (R46)
   업로더                         U7.  internal/enode 를 0 줄 만진다
   AppendLog 의 n == limit 오판    고치지 않는다 (FD 6.3).  이 회차가 안 산 변경이다
   logs/ 의 상한 규칙 · 자르는 자리  안 바꾼다.  Q2 = A 가 AppendProgress 에만 걸었다
   DB 스키마 · 인덱스              diff 0.  constraints.md 6절
   디스크 여유 보기                Q4 = A.  R40 · R41 이 그 대가다
   새 설정 키 · 새 타이머 · 새 의존   전부 0 (V1 · 2.4 · SECURITY-10)
   회차 문서와 팩 고치기            진행자다.  9절이 글자만 적는다
```

---

## 8. 물음 — **0 이다**

**규칙이 물으라고 한 자리를 전부 지났고 진짜 갈림이 0 이라 억지로 만들지
않았다.** 앞선 세 단계가 물음 열하나 · 넷 · 0 을 이미 지났다.

| 자리 | 왜 안 묻나 |
|---|---|
| 순회를 어디에 두나 (5절 ①) | **도출이다.** `safe()` 가 비공개라 store 가 순회를 못 한다. 고를 수 있는 값이 아니다 |
| 뮤텍스 자료구조 (5절 ②) | FD 4.3 이 **이 단계에 넘긴** 자리다. 요구가 「겹치면 안 된다」 하나이고 그것을 만족하는 최소가 하나다 |
| 본문을 자르나 (5절 ③) | **CB3 이 정한다** — 「두 벌이 안 생긴다」가 수용 기준이다 |
| 질의 비용 (5절 ④) | 인덱스를 더하는 길이 `constraints.md` 6절에 막혀 있다. 고를 값이 없고 남는 대가를 9절이 넘긴다 |
| 6시간의 자리와 이름 | 값은 V5 가 닫았다. 상수를 어느 패키지에 두느냐는 **그 값이 지배하는 것 옆**이 답이라 갈림이 아니다 |
| `bytes` 가 무엇인가 | R12 가 상한으로 닫았다 |
| `attempt` 가 음수면 | R33 이 U4 에 맡겼다. `record` 의 전제는 R32 하나다 |

**갈림이 있었으면 멈추고 물었다.**

---

## 9. 진행자에게 넘기는 것

앞 단계가 넘긴 것은 그대로 산다 (`nfr-design-patterns.md` 5절 ·
`nfr-requirements.md` 9절 · `business-rules.md` 11절). **이 단계가 새로
더하는 것은 아래 여섯이다.**

### 9.1 실측이 앞 문서와 어긋나는 자리 — 줄 번호 일곱

**FD 와 NFR 문서가 가리킨 줄 번호 중 일곱이 오늘 코드와 다르다.** 근거로 쓰인
문장은 전부 참이고 **가리키는 줄만 어긋났다.**

| 적힌 것 | 오늘 | 어디에 적혀 있나 |
|---|---|---|
| `parseBlobName` `record.go:263` | **`:304`** | FD `domain-entities.md` 2.3 · `business-rules.md` R6 |
| `Blobs()` 의 `.tmp-*` `record.go:392` | **`:389`** | FD `business-rules.md` R6 |
| `if n > 0` `reap.go:104` | **`:99`** | FD 6.2 · `business-rules.md` 5.2 ② · NFR Req V2 |
| `interval '1 hour'` `reap.go:126` | **`:118`** | FD 6.2 · `business-rules.md` 5.2 ① |
| `Sealed` 건너뛰기 `reap.go:148` | **`:146`** | FD 5.3 |
| `name` 질의 `api.go:884` | **`:883`** | `business-rules.md` 8.1 |
| `Attempt int` `claim.go:73` | **`internal/store/claim.go:156`** | FD `domain-entities.md` 3.3 |

```text
   함께 못 찾은 것 하나
     FD domain-entities.md 3.3 이 「expands 로 붙은 단계가 시도 0 이라는 것을
     claim.go:712-713 의 주석이 적는다」고 적었다.  그 줄에 그 주석이 없고
     internal/store/claim.go 전체에서 「시도 0」을 적은 주석을 못 찾았다.
     **짐작해 고치지 않는다** — 근거 문장이 참인지를 진행자가 본다

   맞는 것 (확인했다)
     record.go 의 :32 :38 :45 :47 :59 :64 :70 :92 :96 :115 :120 :156 :159
     :174 :180 :270 :287 :326 · api.go 의 :50 :860 :869 :879 :1037 :1043 ·
     reap.go 의 :113 :158 · seal.go:175 · config.go:100 ·
     cmd/mediator/main.go:89 · runner.go 의 :361 :415 :425 ·
     transcript.go 의 :28 :40 :65 · internal/enode/claim.go 의 :145 :647 :796
```

### 9.2 `unit-of-work.md` U3 절의 한 줄

```text
   오늘   Seal 이 tar 보다 먼저 진행 트리를 지운다
   뒤     Seal 이 seal(d) 의 chmod 보다 먼저 진행 트리를 지운다

   근거   NFR Design 2.1 이 이미 낸 교정이다.  Seal 은 tar 를 안 짓고
          record.go:120 의 seal(d) 로 끝난다.  Tar 는 GET /record 때
          돈다 (api.go:1043).  execution-plan.md 의 NFR Design 절에도
          같은 표현이 있고 그쪽은 nfr-design-patterns.md 5절이 이미 넘겼다
```

### 9.3 `enode.capped` 의 글자를 U1 에 준다

```text
   찍는 바이트   {"type":"enode.capped","bytes":<상한>}  뒤에 개행 하나
   기본 상한에서  {"type":"enode.capped","bytes":10485760}  =  40 + 개행 = 41 바이트
   필드 둘       type 은 문자열 · bytes 는 정수(int64).  그 밖의 필드가 0 개다
   bytes 의 뜻    **상한이다.**  잘린 자리가 아니다

   U1 에 주는 이유   짝이 U1 의 business-rules.md 16.1 이고 U3 은 Kind 를 안 짓는다
                   (R46).  왕복은 U4 에서 처음 한 바이너리에 든다 (Step 16.5)
   안 맞으면        U1 이 raw 사건 하나로 올린다 — 버리지도 실행하지도 않는다.
                   표시의 질이 떨어지는 것이지 손실이 아니다
```

### 9.4 게이트 7 의 도는 스크립트가 0 이다

9.1 과 별개의 자리다. **6.3 의 실측이 근거다** — `glyphscan.go` 가 게이트 7 의
절반(ASCII 두 별표 · 주석)을 안 본다. **U3 은 새로 안 만들고 기존 것을 안
고친다** (여덟 파일이 전부 파일 행렬 밖이고 고치면 게이트 2 의 diff 0 이 깨진다).

```text
   넘기는 값   **위반 열다섯 · 파일 여덟** (전부 Go 주석 · 1.2)
              **위반 아님 여섯 · 파일 둘** (cmd/iapadapter 의 코멘트 마크다운 · 1.1)
              21 을 통째로 위반으로 받으면 안 된다

   왜 이 갈림이 값인가
              다음 사람이 이 표를 근거로 고친다.  21 을 통째로 적어 두면
              누군가 cmd/iapadapter 의 여섯을 지우고 **그 순간 GitHub 코멘트의
              굵게가 사라진다** — 규약을 지키려다 1.1 을 어기는 것이다

   도는 스크립트를 짓는다면
              **cmd/iapadapter 를 제외 목록에 둔다** (6.3 의 grep 이 그 꼴이다)

              다만 경로 제외는 뭉툭하다 — 그 패키지의 **주석**에 새로 들어온
              두 별표를 놓친다.  **정확한 규칙은 자리다: 주석 안이면 위반(1.2) ·
              렌더링되는 문자열이면 1.1 의 자리다.**  glyphscan.go 가 문자열과
              rune 리터럴을 보고 주석을 안 보므로 **이 검사기는 그 반대를 봐야
              한다** — 같은 AST 순회에 얹을 수 있고 두 벌이 안 된다
              (CONVENTIONS.md 1.4)
```

### 9.5 쓸기의 질의 비용 — 받아들이는 잔여

```text
   무엇      Reap() 의 쓸기가 진행 트리가 있을 때 60초마다
            SELECT run_id FROM runs WHERE state NOT IN ('SUCCEEDED','FAILED')
            를 친다.  runs.state 에 인덱스가 없어 전수 스캔이다 (실측)

   막은 것    진행 트리가 0 이면 질의가 0 번이다 (5절 ④).  U4 · U7 이 서기
            전에는 언제나 0 이고, 선 뒤에도 조용한 함대에서는 0 이다

   안 막은 것  진행 트리가 있는 동안의 스캔.  runs 는 영원히 쌓이므로
            (ADR-005 「장기 보관 정책은 MVP 밖」) 그 비용이 Run 총수와 함께 자란다

   왜 안 고치나  인덱스를 더하려면 schema.sql 을 만져야 하고 그것은
            constraints.md 6절이 명시로 뺀 자리다.  R41 과 같은 결이다 —
            **없앤 것이 아니라 받아들인 것이고 그 경로를 이름으로 적어 둔다**
```

### 9.6 앞 단계가 넘긴 것 중 그대로 사는 것

```text
   requirements.md 5.4 잔여 ① 과 회차 aidlc-state.md 의 미결
     nfr-requirements.md 9.1 · 9.2 가 낸 글자 그대로
   unit-of-work-file-matrix.md 5절 · execution-plan.md 6절 품질 게이트 2
     FD 10절이 낸 확정 그대로.  **고치는 때는 이 유닛이 병합되기 전이다**
   application-design.md D1 의 줄 번호 · D4 의 근거 문장
     FD 5.1 이 낸 것 그대로
   decisions.md 2절의 사건 종류에 enode.capped 행
     business-rules.md 11절이 낸 것 그대로.  글자는 9.3
   GLOSSARY.md 의 CB · N1 · N2 행
     **푼 말을 U3 이 모른다.  안 짐작한다** (CLAUDE.md)
   코드의 잔여 — AppendLog 의 n == limit 오판
     FD 6.3.  이 회차가 안 산 변경이다.  Step 9.7 이 주석 한 줄로 자리만 적는다
```

---

## 10. 센 것

```text
   Step          열아홉.  갈래 A 열하나 · 갈래 B 셋 · 공통 다섯
   제품 파일      넷 (새 파일 하나 · 고치는 파일 셋)
   시험 파일      둘 (둘 다 새 파일)
   수명 끊는 자리  셋 — Step 9.1 (봉인) · Step 10 (종료 상태) · Step 11 (쓸기)
   변이          여덟 (Step 17)
   물음          **0** (8절)
   진행자에게     여섯 (9.1 ~ 9.6)
   diff 0 이 규칙인 경로   일곱.  깨지는 것 하나 (internal/store)
```
