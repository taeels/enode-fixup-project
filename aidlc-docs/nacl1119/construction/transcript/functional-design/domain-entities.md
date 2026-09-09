# transcript — 자료 모델 (domain-entities)

링 파일과 제어판 카드가 쓰는 것. DB 를 안 만진다. 값은 `decisions.md` §6.2/§6.3/
§6.5 가 닫았고, 여기는 그 값을 Go 표면으로 적는다.

---

## 1. 링 파일 Ring (신규 · internal/enode/transcript.go)

고정 크기 · 이름 안 바뀜 · 안 지움 · 잠금 없음 (decisions §6.3). 윈도우와
유닉스가 같은 코드로 돈다.

```text
   경로     TranscriptPath(configPath) = <dir>/<stem>.transcript
            PolicyPath · StatusPath 와 같은 규칙 (설정 경로가 곧 신원 · ADR-017)
   파일     머리(고정) + 몸통(용량 바이트).  한 번 만들고 크기 안 바뀐다
   권한     유닉스 0600 · 윈도우는 그 디렉터리의 상속 ACL (정책·상태 파일과 같다)
```

머리(고정 32바이트 · 리틀엔디언).

```text
   magic       4B   "ENTR"  — 다른 파일을 링으로 오인하지 않게
   version     2B   판.  형식이 바뀌면 올린다
   (예약)      2B   정렬용 0
   capacity    8B   몸통 바이트 수 = 512*1024
   total       8B   지금까지 이 세대에 쓴 총 바이트 (몸통에 감기는 커서)
   generation  8B   비우기(Reset)마다 +1 — 카드가 「새 단계」를 이것으로 안다
```

몸통 — `capacity` 바이트의 원형 버퍼. 쓰기 커서는 `total % capacity`.

Go 표면.

```go
type Ring struct { /* *os.File · capacity · 내부 상태 */ }

func TranscriptPath(configPath string) string      // <dir>/<stem>.transcript
func OpenRing(path string, capacity int) (*Ring, error) // 없으면 만든다(머리 초기화)
func (r *Ring) Write(p []byte) (int, error)        // 몸통에 WriteAt · 감김 · total 갱신. io.Writer
func (r *Ring) Reset() error                        // total=0 · generation+1. 파일은 그대로
func (r *Ring) Close() error

// 읽기 — 제어판이 쓴다. 파일만 있으면 데몬 없이도 읽힌다.
type Snapshot struct {
    Data       []byte // 순서대로(오래된 것 -> 새것). total<=capacity 면 전부, 넘으면 최근 capacity
    Generation uint64
    Total      uint64
}
func ReadRing(path string) (Snapshot, error)
```

`Write` 는 `io.Writer` 라 runner.go 의 `io.MultiWriter(&stdout, ring)` 한 겹으로
꽂힌다. 몸통 끝을 넘으면 두 조각으로 갈라 WriteAt 한다.

---

## 2. Job.Transcript — tee 대상을 나르는 자리

runHarness 에 링 writer 를 넘기는 이음매. nil 이면 안 흘린다(설정 없는 시험).

```go
// internal/enode/runner.go 의 Job 에 additive
type Job struct {
    // ... 기존 필드 ...
    Transcript io.Writer // 하네스 stdout 을 tee 할 곳 (ADR 없음 · decisions §6.2 Q2)
}
```

Worker 가 노드의 Ring 을 열어 단계 시작 때 `Reset()` 하고 `ring`(io.Writer)을
`Job.Transcript` 로, 명령 단계에는 같은 writer 를 `io.MultiWriter` 에 넣는다.

---

## 3. 트랜스크립트 카드 view (제어판 · 지금 도는 것)

```text
   출처     로컬 링 파일 ReadRing(TranscriptPath(cfg.ConfigPath))
   내용     Snapshot.Data 를 텍스트로.  Generation 이 바뀌면 화면을 비우고 새로 채운다
   폴링     화면 열려 있을 때만 1초.  Mediator 폴링(5초)과 별개 타이머 (decisions §6.2)
   자리     데몬 로그 카드 옆.  다른 물건임이 화면에서 보인다 (CP6 · S3)
   엔드포인트 GET /api/transcript  -> {generation, total, data}
```

제어판은 한 노드를 보므로 `node=` 인자는 자기 노드로 기본값이다 — 새 필터를
안 만든다.

---

## 4. 지난 작업 view (제어판 · Mediator 가 이미 가진 것)

Mediator 를 안 고친다 (decisions §6.5). 제어판이 읽어 온다.

```text
   목록     GET /api/runs -> runctl.Client.Runs(RunsQuery{}) 원문 ->
            assigned([{as, nodes:[{node,label}]}])에 자기 node_id 가 있는 Run 만.
            행이 이미 진다 — run_id · state · ended_at · verdict.checks
   상세     누르면 GET /api/record?run=<id> -> runctl.Client.Record tar ->
            logs/NN-<step>.log 만 꺼낸다(archive/tar 표준 · 새 의존 0) + verdict.checks
   자리     S3 안의 카드.  S4 는 자리만 (decisions §6.1 · scene-gates CP6)
```

파싱 타입 — `runctl.Client.Runs` 는 원문 `json.RawMessage` 다(obs 가 그렇게 남겼다).
제어판이 `run_id` · `state` · `ended_at` · `assigned` · `verdict` 만 뽑는 최소
타입으로 디코드한다.

---

## 5. 안 만드는 자료

```text
   새 DB 표 · 새 store 열              트랜스크립트는 DB 를 안 만진다
   node= 서버 필터                      assigned 로 제어판이 거른다 (decisions §6.5)
   빌드 태그 쌍                         링이 Truncate·rename·삭제를 안 해서 안 는다 (§6.3)
   새 라우트(Mediator)                  record·runs 는 obs 가 이미 냈다
```
