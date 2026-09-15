# 컴포넌트 메서드 — 겉면과 입출력

**여기는 겉면이다.** 필드의 값과 그리는 규칙은 Functional Design 이 닫는다
(`execution-plan.md` 3절). 여기서 정하는 것은 **무엇이 들어오고 무엇이 나가며
누가 그것을 부르는가**다.

---

## 1. `internal/transcript`

### 1.1 읽는 쪽

```go
// Kind 는 사건 종류다. decisions.md 2절의 여섯이다.
type Kind string

// Event 는 화면이 그리는 한 조각이다. 필드는 Functional Design 이 닫는다.
type Event struct{ Kind Kind /* ... */ }

// Result 는 한 번 읽은 결과다.
type Result struct {
	Events  []Event
	Elided  *Elided // enode.elided 줄이 있었으면. 없으면 nil
	Raw     int     // raw 로 넘어간 줄 수
}

// Elided 는 걷힌 양이다. NC 없이 CB2 가 이것을 재고 화면이 한 줄로 그린다.
type Elided struct{ Events, Bytes int }

// Parse 는 바이트를 사건 열로 읽는다.
//
// truncated 가 참이면 첫 개행 뒤부터 읽는다 — 링이 감겨 첫 줄이 잘렸을 때다.
// 총량이 용량 안이면 호출자가 거짓을 넘긴다.
func Parse(b []byte, truncated bool) Result
```

**시계를 안 받는다** (Q3 = A). `Parse` 는 같은 입력에 언제나 같은 값을 낸다 —
커버리지 80% 를 하네스 없이 채우는 시험이 그래서 싸다.

### 1.2 짓는 쪽 — `internal/enode` 에서 옮겨 온 셋 (Q4 = A)

```go
// ParseLine 은 줄 하나에서 아는 키만 읽는다. 옛 parseEventLine.
// 파서의 입력은 신뢰할 수 없다 (SECURITY-13).
func ParseLine(line []byte) (obj map[string]any, typ string, ok bool)

// Shell 은 본문을 걷은 껍데기를 짓는다. 옛 eventShell. selectLogs 가 쓴다.
func Shell(obj map[string]any, typ string) []byte

// ElidedMarker 는 「N 개 · B 바이트를 걷었다」한 줄이다. 옛 elidedMarker.
func ElidedMarker(events, bytes int) []byte
```

**짓는 함수와 읽는 함수가 한 파일을 본다.** `Shell` 이 바뀌면 `Parse` 의 시험이
그 자리에서 빨개진다 — 오늘은 두 패키지라 조용히 갈릴 수 있다.

---

## 2. `internal/record`

### 2.1 진행 파일 — 새 겉면 넷

```go
// Progress 는 진행 파일의 지금이다. 화면과 노드가 이 셋으로 갈린다.
type Progress struct {
	Total   int64 // 파일의 총 길이. 노드가 이 값으로 오프셋을 맞춘다
	Attempt int   // 지금 시도. 바뀌면 화면이 카드를 비운다 (Q1 = A 의 대가)
	Capped  bool  // 총 길이가 상한에 닿았다. NC-4 가 이것을 그린다
}

// AppendProgress 는 청크를 진행 파일에 잇는다.
//
// attempt 가 파일에 적힌 것과 다르면 먼저 비운다 (Q1 = A) — 링이 단계 시작에
// Reset 하는 것과 같은 답이다. 그때 Total 이 0 부터 다시 센다.
// 상한은 총 길이로 건다 (D4) — 닿으면 더 안 쓰고 Capped 가 참이다.
func (s *Store) AppendProgress(runID string, seq int, name string,
	attempt int, r io.Reader, limit int64) (Progress, error)

// ReadProgress 는 from 바이트부터 읽는다. 파일이 없으면 Total 0 에 빈 본문이다.
func (s *Store) ReadProgress(runID string, seq int, name string,
	from int64) (Progress, io.ReadCloser, error)

// DropProgress 는 그 Run 의 진행 트리를 통째로 지운다. Seal 이 부른다.
// 없으면 아무것도 안 한다 — 멱등이다 (I4 와 같은 결).
func (s *Store) DropProgress(runID string) error
```

**자리**: `<Root>/progress/run-<safe(id)>/NN-<safe(name)>.log` (Q2 = A).
`safe` 와 `%02d-%s` 는 `logs/` 와 같은 규칙을 쓴다 — 두 벌로 안 둔다.

**시도를 어디에 적나**: 진행 파일 옆의 작은 곁파일이다. 이름과 형식은
Functional Design 이 닫는다. **`internal/store` 에 안 묻는다** — `record` 가
`store` 를 임포트하지 않는 오늘의 방향을 안 바꾼다.

### 2.2 `AppendLog` — 뜻만 바뀐다 (D4)

```go
// AppendLog 는 그대로다. 반환값의 뜻이 「이번 호출의 바이트」에서
// 「붙인 뒤의 총 길이」로 바뀌고, 상한도 총 길이로 건다.
func (s *Store) AppendLog(runID string, seq int, name string,
	r io.Reader, limit int64) (int64, error)
```

**기존 호출자가 안 깨진다** — `api.go:887` 이 `if _, err := ...` 로 값을 버린다
(실측). 시그니처가 같으므로 컴파일도 안 깨진다.

### 2.3 `Seal` — 한 줄이 는다

```go
// Seal 은 오늘 그대로 봉인한다. 다만 tar 로 굳기 전에 진행 트리를 지운다.
// 순서가 뒤집히면 원문이 0444 로 굳어 아무도 못 지운다 (record.go:156 · :196).
func (s *Store) Seal(runID string, manifest, verdict any, steps []StepFile) error
```

---

## 3. `internal/enode`

```go
// Decode 는 시그니처가 안 바뀐다 (agent-runtime R3). 안이 배치에서 줄 단위로
// 바뀐다 — io.ReadAll 을 걷고 읽으면서 emit 한다.
func (claudeHarness) Decode(r io.Reader, exitCode int, emit func(Event)) HarnessResult

// Uploader 는 청크를 Mediator 에 올린다. io.Writer 다 — tee 의 한 갈래로 꽂힌다.
//
// Write 는 버퍼에 담고 곧 돌아온다. 올리는 것은 자기 고루틴이다 —
// **PUT 이 느리거나 실패해도 하네스의 stdout 이 안 막힌다** (FR-5).
type Uploader interface {
	io.Writer
	Close() error // 남은 것을 마지막으로 올린다
}

// newUploader 는 한 단계의 업로더를 만든다. 주기와 임계는 decisions.md 2절 —
// 새 바이트가 있으면 2초, 64 KiB 가 쌓이면 즉시.
func newUploader(c *Client, runID string, seq, attempt int, name string) Uploader

// transcript 는 tee 의 갈래를 합친다. 오늘은 링 하나를 돌려준다.
//
//	ring 만 있으면      ring
//	uploader 만 있으면  uploader
//	둘 다 있으면        io.MultiWriter(ring, uploader)
//	둘 다 없으면        nil
func (w *Worker) transcript() io.Writer
```

**`Client` 에 메서드 하나가 는다.**

```go
// PutLogChunk 는 진행 파일에 청크를 잇고 그 뒤의 총 길이를 받는다.
// 노드는 돌아온 총 길이로 자기 오프셋을 맞춘다 — 서버가 진실이다 (FR-5).
func (c *Client) PutLogChunk(ctx context.Context, runID string, seq, attempt int,
	name string, body []byte) (total int64, capped bool, err error)
```

---

## 4. `internal/api`

### 4.1 GET — 신규 라우트 하나 (FR-6)

```text
   GET /v1/runs/{run}/steps/{seq}/log?from=<바이트>&as=<raw|events>&name=<단계 이름>

   200   본문은 from 부터의 바이트 (as=raw) 또는 사건 배열 (as=events)
   400   seq · from · as 가 규칙 밖이다 (SECURITY-05)
   404   그 Run 이 없다.  파일만 없으면 200 에 총 길이 0 이다
```

응답 헤더 넷. **셋이 이 회차가 새로 다는 것이다.**

```text
   X-Enode-Log-Bytes     총 길이.  노드와 화면이 다음 from 을 여기서 얻는다
   X-Enode-Log-Source    progress | sealed.  **봉인 전후의 갈림이다** (D3 · NC-5)
   X-Enode-Log-Attempt   지금 시도.  값이 바뀌면 화면이 카드를 비운다 (Q1 = A 의 대가)
   X-Enode-Log-Capped    1 이면 상한에 닿았다.  NC-4 가 이것을 그린다
```

**보호 규칙은 `GET /v1/runs` 와 같다. 판정을 안 싣는다** (`ADR-065`).
**봉인 전에도 답한다** — 이것은 Record 가 아니다 (`I4` · `ADR-025` §5).
`GET record` 의 `409` 는 그대로다.

### 4.2 PUT — 기존 라우트의 갈림 (D2)

```text
   PUT /v1/runs/{run}/steps/{seq}/log                  오늘 그대로.  logs/ 에 붙인다
   PUT /v1/runs/{run}/steps/{seq}/log?progress=1&attempt=<n>   진행 파일에 붙인다

   응답 헤더    X-Enode-Log-Bytes · X-Enode-Log-Attempt · X-Enode-Log-Capped
```

**새 라우트를 안 만든다.** `api.go` 의 `mux.HandleFunc` 셈이 17 에서 **18** 이
된다 — 는 것은 GET 하나뿐이다 (CB0).

---

## 5. `internal/runctl` — 메서드 하나

```go
// StepLog 는 한 단계의 로그를 받는다. 제어판이 지난 것을 그릴 때 쓴다.
// tar 를 통째로 푸는 우회를 이것이 대신한다 (FR-4).
func (c *Client) StepLog(ctx context.Context, runID string, seq int,
	name string, from int64, as string) (StepLog, error)

type StepLog struct {
	Body    []byte
	Total   int64
	Source  string // progress | sealed
	Attempt int
	Capped  bool
}
```

---

## 6. `internal/panel` · `internal/api/ui`

**새 겉면이 없다.** 둘 다 이미 있는 핸들러와 화면이 파서를 쓰게 바뀐다.

```text
   handleTranscript    오늘 그대로 링을 읽는다.  내용이 NDJSON 이라 카드가
                       transcript.Parse 를 거쳐 그린다 (transcript.go:20)
   handleRecord        출처를 바꾼다.  Status 로 단계 목록을 받고 단계마다
                       StepLog 을 부른다.  **archive/tar 임포트가 사라진다**
   page.go:15          보안 헤더 다섯을 건다 (requirements.md 5.5)
   static/fleet        Run 상세에 단계 카드.  2초 타이머는 목록의 5초와 별개다
```

**화면이 지는 계산은 뺄셈 하나다** (Q3 = A) — 「지금 빼기 그 시각」. NC-1 과
NC-6 이 그것이고, 나머지 넷은 파서와 헤더가 낸 값을 그대로 그린다.
