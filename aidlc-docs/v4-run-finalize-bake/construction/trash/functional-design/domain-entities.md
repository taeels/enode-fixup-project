# `trash` — 도메인 엔티티

단계가 남긴 작업 폴더(runRoot)를 버리는 자리와, 여유가 모자랄 때 노드가 스스로 거는 drain 의 칸과 타입이다.
입력은 계획의 답 열(`construction/plans/trash-functional-design-plan.md` 2절)이다. 규칙은 `business-rules.md`,
흐름은 `business-logic-model.md` 에 있다.

```text
   답   1 A   닫기를 Finalize 예산 안으로.  닫기는 마감으로 끊지 않는다
        2 A   trash 는 <scratch>/trash/ · 이름은 작업 폴더 이름 · 세션 잠금으로 남은 폴더를 알아본다
        3 A   항목 하나에 helper 하나 · 차례로 · idle IO · 단계가 돌아도 멈추지 않는다
        4 A   이름 하나만 받는다 · symlink 를 안 따라감 · 권한 000 을 풀고 · 다른 filesystem 으로 안 넘어간다
        5 A   지우기 전에 같은 helper 안에서 크기와 항목 수를 먼저 측정한다
        6 A   min_free_gb 를 가진 모든 노드가 여유 부족 drain 을 건다
        7 A   graceful · 센 쪽 · 걸 때 < min · 풀 때 >= min + 1 GB
        8 A   상태 파일의 칸을 한 자리에 모아 바뀌면 쓴다
        9 A   제어판 — 실린 drain · 출처마다 누가 풀 수 있나 · 버튼은 소유자 출처가 있을 때만 · trash 의 양
       10 A   삭제자 첫 회는 배경 · 광고는 기다리지 않는다 · 조각 4 는 사람과 기계
```

**새 패키지는 하나다** — `internal/scratch` (Application Design 의 자리). trash 와 삭제의 규칙만 담고, namespace 를
여는 일과 광고는 `internal/enode` 가 한다. 표준 라이브러리와 `golang.org/x/sys` 만 쓴다.

---

## 1. `internal/scratch` — trash

```go
// Trash 는 <scratch>/trash 다. 단계가 남긴 작업 폴더가 rename 한 번으로 들어온다.
type Trash struct {
	Dir string
}

// Move 는 path 를 trash 로 옮긴다. rename 한 번이라 크기와 무관하다 (requirements.md 5.4).
// 이름은 path 의 마지막 조각이다. 같은 이름이 trash 에 있으면 "-1" · "-2" … 를 붙인다.
// trash 가 없으면 0700 으로 만든다. 돌려주는 것은 trash 안의 이름이다.
func (t Trash) Move(path string) (string, error)

// Entries 는 trash 의 바로 아래 이름들이다. 노드 uid 로 읽는다 — trash 자체는 노드 소유다.
func (t Trash) Entries() ([]string, error)
```

- `Move` 가 실패하면 fallback 으로 지우지 않는다 — 지우기는 그 비용을 보고 전 창에 다시 들인다. 오류는 세션 닫기
  오류로 결과에 남는다 (`components.md` 5절 · 오늘 규칙)

---

## 2. `internal/scratch` — 세션 잠금 (답 2 = A)

```go
// SessionLockName 은 작업 폴더 안의 잠금 파일 이름이다.
const SessionLockName = ".enode-session.lock"

// HoldSession 은 작업 폴더를 만든 프로세스가 쥐는 잠금이다 (flock · 배타 · 기다리지 않음).
// 세션이 닫히면 Release 한다. 프로세스가 죽으면 커널이 푼다.
func HoldSession(runRoot string) (*SessionLock, error)
func (l *SessionLock) Release() error

// Orphans 는 scratch 에서 쥔 프로세스가 없는 작업 폴더를 찾는다 (prefix = "enode-runc-").
// 잠금 파일이 있고 잠금을 쥘 수 있으면 남은 것이다.  잠금 파일이 없으면 만든 지 1시간이 지난
// 것만 남은 것으로 친다 — 이 유닛 전의 enode 가 남긴 폴더이고, 막 만든 폴더의 짧은 창을 비킨다.
func Orphans(scratch, prefix string, now time.Time) (orphans, skipped []string, err error)
```

- 잠금은 작업 폴더를 따라 trash 로 간다 (inode 를 따른다). 닫기는 **rename 뒤에** 잠금을 놓는다
- `enode env check` 의 준비도 smoke 도 같은 `Open` 을 지나므로 잠금을 쥔다 — 데몬의 기동 청소가 도는 smoke 를 안 건드린다

---

## 3. `internal/scratch` — 삭제와 측정 (답 4 · 5 = A)

helper 안(runtime 과 같은 uid 매핑의 user namespace)에서 돈다. 노드 uid 는 subordinate uid 소유 디렉터리와 권한 000
디렉터리 안을 못 걷기 때문이다.

```go
// CheckEntry 는 helper 가 받은 이름을 본다. trash 의 바로 아래 이름 하나만 받는다.
// 빈 이름 · "." · ".." · "/" 가 든 이름은 거절한다. trash 자체가 디렉터리가 아니거나 symlink 면 거절한다.
func CheckEntry(trashDir, entry string) error

// Size 는 항목의 크기(블록 수 x 512)와 항목 수(디렉터리 포함)다.
type Size struct {
	Bytes   int64 `json:"bytes"`
	Entries int64 `json:"entries"`
}

// Measure 는 지우기 전에 한 번 걷는다. Remove 와 같은 경계를 지킨다.
func Measure(trashDir, entry string) (Size, error)

// Remove 는 항목을 지운다. symlink 는 링크 자체만 지운다. 권한 000 디렉터리는 들어가기 전에
// 0700 으로 푼다. trash 와 다른 filesystem(st_dev)의 디렉터리는 들어가지 않고 남긴다 — 그러면
// 그 항목은 다 못 지운 것이고 Left 에 남긴 경로가 있다.
func Remove(trashDir, entry string) (Left []string, err error)
```

---

## 4. `internal/scratch` — 배경 삭제자와 양 (답 3 · 5 · 8 = A)

```go
type Deleter struct {
	Trash Trash
	// Launch 는 항목 하나에 helper 하나를 연다 (enode 가 채운다 — namespace 를 여는 쪽).
	// measured 는 helper 가 지우기 전에 낸 측정이다. 지우기가 끝나면 돌아온다.
	Launch func(ctx context.Context, entry string, measured func(Size)) error
	// Changed 는 Usage 가 바뀔 때 불린다 — 상태 파일을 쓰는 자리가 받는다.
	Changed func(Usage)
	Log     *slog.Logger
	Every   time.Duration // trash 에 항목이 남아 있을 때 다시 깨는 간격.  기본 10분
}

func (d *Deleter) Run(ctx context.Context) // 곧바로 한 번 돌고 Kick · Every 마다 깬다
func (d *Deleter) Kick()                   // 결과 보고 뒤.  막지 않는다 (채널 하나 · 겹치면 하나로)
func (d *Deleter) Usage() Usage

// Usage 는 scratch 의 양이다. 늦은 값이다 — 측정한 시각을 함께 싣는다 (Application Design Q3).
type Usage struct {
	TrashBytes   int64     `yaml:"trash_bytes" json:"trash_bytes"`     // 측정한 항목들의 합
	TrashEntries int       `yaml:"trash_entries" json:"trash_entries"` // trash 에 남은 항목 수
	TrashUnsized int       `yaml:"trash_unsized" json:"trash_unsized"` // 그 가운데 아직 측정하지 않은 수
	Deleting     bool      `yaml:"deleting" json:"deleting"`
	MeasuredAt   time.Time `yaml:"measured_at" json:"measured_at"`
}
```

- `SpoolBytes` · `Checkpoints` (component-methods 3.1) 는 checkpoint 유닛이 더한다. 이 유닛은 칸을 만들지 않는다
- 측정값은 삭제자의 메모리에만 있다. 데몬이 재시작하면 남은 항목은 다시 「크기 모름」이다

---

## 5. trash-helper — 입구와 줄 (`internal/enode` · `cmd/enode`)

```text
   명령      unshare --user --map-root-user --map-auto --fork --kill-child -- <enode> trash-helper <trash> <entry>
             runtime-helper 와 같은 uid 매핑이다.  --mount 는 안 연다 — 삭제는 마운트하지 않는다
   먼저      IO 우선순위를 idle 로 (ioprio_set · IOPRIO_CLASS_IDLE) · CPU 우선순위를 19 로 (setpriority)
   stdout    JSON 한 줄씩
             {"measured": {"bytes": N, "entries": M}}
             {"removed": true}                          다 지웠다
             {"removed": false, "left": ["…"]}           다른 filesystem 이라 남긴 경로가 있다
             {"error": "…"}                              CheckEntry · Measure · Remove 의 오류
   exit      0 다 지웠다 · 1 그 밖
```

```go
func RunTrashHelper(args []string, out, errOut io.Writer) int // internal/enode (linux) · 다른 OS 는 1
```

`internal/enode` 의 새 파일 `trash_linux.go` · `trash_other.go` 에 둔다 (파일 행렬 1.3 의 「trash-helper (새 파일)」).
`Launch` 도 여기서 짓는다 — `os.Executable` 로 자기 바이너리를 찾는 것은 runtime-helper 와 같다.

---

## 6. drain 의 출처와 합치기 (`internal/enode/policy.go` · 답 6 · 7 = A)

```go
// DrainSource 는 drain 의 출처 하나다 (완료 조건 1).
type DrainSource struct {
	Kind   string `yaml:"kind" json:"kind"`     // owner | disk.  bake 는 lower-state 유닛이 더한다
	Mode   string `yaml:"mode" json:"mode"`     // graceful | at-boundary
	Detail string `yaml:"detail" json:"detail"` // 영어 한 줄 — business-rules.md 6절
	Owner  bool   `yaml:"owner" json:"owner"`   // 소유자가 정책 파일에서 풀 수 있나
}

// DrainStatus 는 이번 광고에 실은 drain 과 그 출처다.
type DrainStatus struct {
	Effective string        `yaml:"effective" json:"effective"` // "" | graceful | at-boundary
	Sources   []DrainSource `yaml:"sources" json:"sources"`
	At        time.Time     `yaml:"at" json:"at"`               // 이 값을 정한 광고 주기의 시각
}

// combineDrain 은 출처 목록에서 센 쪽을 고른다 — at-boundary > graceful > "".  순수 함수다.
func combineDrain(sources []DrainSource) string

// diskDrain 은 여유 부족 출처를 정한다. held 는 앞 광고에서 걸었는가 — 거는 값과 푸는 값이 다르다.
// 순수 함수다. 출처가 없으면 ok 가 거짓이다.
func diskDrain(freeBytes uint64, minFreeGB int, held bool) (src DrainSource, ok bool)
```

- 소유자 출처는 정책 파일의 값이다 (`policyReader.Read` 그대로). 값이 없으면 출처가 없다
- 광고의 `policy.drain` 에는 `combineDrain` 의 결과 하나만 실린다 — 새 광고 어휘도 Mediator 변경도 없다 (FR-4)

---

## 7. 상태 파일의 새 칸 (`internal/enode/status.go` · 답 8 = A)

```go
type Status struct {
	Caps    []contract.Capability `yaml:"caps"`
	At      time.Time             `yaml:"at"`
	Drain   *DrainStatus          `yaml:"drain,omitempty"`   // 새로
	Scratch *scratch.Usage        `yaml:"scratch,omitempty"` // 새로.  scratch 가 없는 노드(native)는 없다
}

// statusBook 은 상태 파일의 칸을 한 자리에 모은다. 어느 칸이든 바뀌면 파일을 쓴다.
// Advertiser 와 Deleter 가 다른 고루틴에서 부르므로 잠금 하나를 쥔다.
type statusBook struct { /* mu · path · Status · 마지막으로 쓴 값 · 쓰기 실패 경고 */ }

func (b *statusBook) SetCaps(c Capabilities)
func (b *statusBook) SetDrain(d DrainStatus)
func (b *statusBook) SetScratch(u scratch.Usage)
```

- 옛 상태 파일(칸 둘)도 그대로 읽힌다 — 새 칸은 없으면 nil 이다

---

## 8. 제어판 State 의 새 칸 (`internal/panel` · 답 9 = A)

```go
// State 에 더하는 칸
//   Drain        string              // 오늘 그대로의 이름.  상태 파일의 effective 를 싣는다 (데몬이 돌 때)
//   DrainSources []enode.DrainSource `json:"drain_sources"`
//   DrainFrom    string              `json:"drain_from"` // "status" | "policy" — 어디서 읽은 값인가
//   Scratch      *scratch.Usage      `json:"scratch,omitempty"`
```

- 데몬이 돌고(잠금 파일 · proc) 상태 파일에 drain 칸이 있으면 `status` 에서, 아니면 오늘처럼 정책 파일에서 읽는다
- `internal/panel` 이 `internal/scratch` 를 가져다 쓰는 것은 경계 표의 금지와 안 부딪친다 (`component-dependency.md` 2절)

---

## 9. 세션 겉면에서 바뀌는 것 (`internal/enode/runc_overlay_linux.go`)

```text
   runcOverlaySession    lock *scratch.SessionLock · trash scratch.Trash 를 든다
   RuncOverlayRuntime    trash scratch.Trash (<binding.Scratch>/trash)
   helper cleanup        unmount 만 한다.  runRoot 를 지우지 않는다 — 밖의 rename 이 맡는다
```

`StepSession` 의 겉면(`Close(context.Context, Keep) error`)은 finalize 유닛이 이미 바꿨다. 이 유닛은 그 안의 동작만
바꾼다 — `Keep.Upper` 는 여전히 늘 비어 있다.

---

## 10. 설정

새 칸이 없다. trash 의 자리는 `environment.scratch` 에서 온다(`<scratch>/trash`). `min_free_gb` 의 기본값 10 은 그대로이고
뜻이 바뀐다 — 「빌드 능력을 뺀다」에서 「노드 전체가 drain 한다」로 (완료 조건 3 ①). 한 번에 지우는 양과 속도의 값은
NFR (N1) 이 정한다 — 칸이 필요하면 그때 더한다.
