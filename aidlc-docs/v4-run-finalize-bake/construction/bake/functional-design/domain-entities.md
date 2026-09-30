# `bake` — 도메인 엔티티

노드가 굽기의 두 단계(build · merge)와 끊긴 합치기의 재개를 돌리는 데 드는 타입과 칸이다. 입력은 계획의 답 넷과 3절(묻지
않고 정한 열여섯) · 1절(받는 일 쉰넷)이다 (`construction/plans/bake-functional-design-plan.md`). 규칙과 문구는 `business-rules.md`,
흐름 · 시험 · 넘기는 것 · 정본 되돌림은 `business-logic-model.md` 에 있다.

```text
   답   1 B   IR 이 어긋나면 build 단계는 DONE (error 없음 · exit 0 — sync 의 것) · 원인 코드 ir_mismatch (새 wire 값) ·
              두 갈래 문장은 단계 로그 · build 칸에 새 칸 head_tags.  builds 를 안 돌리고 manifest 를 안 내며 합치지 않는다.
              merge 는 needs 를 따라 와서 합칠 것 없음.  Run 은 계약의 produced ["manifest"] 조건이 판정한다
              (2026-09-27T12:54:46Z 에 A 에서 B 로 다시 답함 — I3)
        2 A   끊긴 합치기(merging)는 시작할 때와 살아 있는 동안 잇는다.  광고 주기에 state 가 merging 이고 굽기 잠금이
              비어 있으면 배경에서 재개하고, 실패하면 그 노드는 10분 뒤에 다시 해 본다.  merge 단계의 Apply 가 멈추면
              FAILED (원인 코드 없음) 로 보고하고 두 잠금을 놓는다.  build claim 이 낡은 merging 을 만나면 재개를 배경에
              열고 build 는 bake_in_progress 로 끝난다
        3 A   builds 는 첫 실패에서 멈춘다.  남은 구성은 단계 로그에 한 줄씩
        4 A   조각 스크립트는 lower 뿌리의 .enode-disposable 이 있어야 돈다.  같은 lower 의 노드가 모두 workspace.writes 를
              광고해야 돈다
```

**새 패키지는 없다.** 흐름은 `internal/enode` 에 새 파일로 두고, 규칙은 같은 패키지의 순수 함수로 뗀다 (`components.md` 3.1 —
이 패키지에는 이어 붙이는 코드만 · 커버리지 여유가 얇다). 상태 · 잠금 · metadata 는 `internal/lower` (lower-state 유닛),
합치기는 `internal/merge` (merge-rules 유닛), trash 는 `internal/scratch` (trash 유닛) 의 것을 부른다.

```text
   internal/enode (새 파일)
     bake.go               Baker · heldBake · StartBaker · 치우는 몸통 · 기동 정리                            모든 플랫폼
     bake_build.go         build 단계 — 굽기 잠금 · 명령 · IR 대조 · pinned · 닫기 · 초안 · pending            모든 플랫폼
     bake_merge.go         merge 단계 — 확인 · 기다림 · 그물 · helper 두 번 · metadata · committed             모든 플랫폼
     bake_resume.go        재개 — 기동 · 광고 주기 · build claim 이 여는 셋이 같은 몸통                        모든 플랫폼
     bakerule.go           순수 함수 — IR 판정과 문장 · 초안 · 대기 로그 · last_attempt · 결과 칸 옮기기 ·       모든 플랫폼
                           pending_upper 의 모양 · url 의 비밀번호
     mergehelper_linux.go  RunMergeHelper · 여는 argv (unshare)                                               linux
     mergehelper_other.go  RunMergeHelper 는 지원하지 않음                                                    linux 밖
   internal/enode (있는 파일)
     claim.go              Step 의 칸 넷 · Result 의 칸 둘 · execute 의 분기                  행렬 ⑦ 열 (파일 행렬에서 bake 가 고치는 칸)
     runc_overlay_linux.go Close 가 Keep.Upper 를 본다                                       행렬 ⑦ 열 (bake 가 고치는 칸)
     finalize.go           contractStep 이 굽기 칸 넷을 옮긴다                                행렬 밖 (finalize 유닛의 파일)
     lowerguard.go         광고 주기의 재개 자리 · Dir 을 빌려준다                             행렬 밖 (lower-state 유닛의 파일)
   cmd/enode/main.go       merge-helper 입구 · StartBaker · Worker 의 Bake                     행렬 ⑦ 열 (bake 가 고치는 칸)
   그 밖 (행렬 밖)          internal/contract/result.go · internal/store/claim.go (물음 1 답 B) ·
                           internal/merge 의 merge.go · decide.go · merge_linux.go (마운트 줄 · lower-state 답 6)
   조각 스크립트            scripts/finalize-bake/ 아래 (이름은 Code Generation)
```

파일 이름과 나눔은 Code Generation 이 바꿀 수 있다. 지키는 것은 셋이다 — 규칙은 시스템 호출 없는 순수 함수다 · unshare 를 여는
코드만 `_linux.go` 다 (`unit-of-work.md` 10절 · `lower` 는 linux 밖에서 같은 겉면이 `ErrUnsupported` 다 — `lower_other.go`) ·
`cmd/enode/main.go` 에는 부르는 줄만 더한다. 행렬 밖 파일의 까닭은 `business-logic-model.md` 10절이다.

---

## 1. 노드 `Step` 의 새 칸 넷 · `contractStep` (계획 3.2 · 받는 일 7 · 19 · 20)

Mediator 는 이미 싣는다 (`internal/store/claim.go:187` ~ `:190`). 노드가 안 받았을 뿐이다 (`internal/enode/claim.go:21` ~ `:89`).

```go
// (claim.go 의 Step 에 더한다) 굽기 단계의 칸 넷 — 이름과 모양은 store.Claimed 와 같다.
Sync   string           `json:"sync,omitempty"`   // build 단계만 — 셸 한 줄
Builds []contract.Build `json:"builds,omitempty"` // build 단계만 — {name, command} 를 적힌 차례로
IR     string           `json:"ir,omitempty"`     // build 단계만 — 구울 IR 태그의 정확한 값
Merge  *contract.Merge  `json:"merge,omitempty"`  // merge 단계만 — "merge": {} 도 nil 이 아니다
```

```go
// contractStep 은 기본값 메서드(EffectOrDefault · Budgets · MergeWait)를 부를 만큼 옮긴다 (finalize.go:28).
// 굽기 칸 넷을 더 옮긴다 — 그래야 MergeWait 이 계약의 merge.wait 을 읽는다 (finalize 가 넘긴 일).
// 종류를 정하는 칸은 build 면 Sync · Builds, merge 면 Merge 가 이미 채운다 — Run 을 채우지 않는다.
func contractStep(step *Step) contract.Step
```

- **kind 는 문자열 그대로 읽는다** — `"build"` · `"merge"` (`contract.KindBuild.String()` · `KindMerge.String()`). Mediator 가 steps.kind
  칸에 그 문자열을 적는다 (`internal/contract/contract.go:428`)
- 예산은 kind 를 안 보므로 오늘도 build 에 맞다 (계획 2.5 · `Budgets`). 바뀌는 것은 `MergeWait` 하나다

---

## 2. `Baker` — 노드의 굽기

runc-overlay 노드에만 있다. `LowerGuard` 와 같은 조건이다 — lower 루트(노드 설정의 워크스페이스)가 있고 scratch 가 있다. 그 밖의
노드는 nil 이고, nil 인 노드가 굽기 단계를 받으면 거절한다 (`business-rules.md` 1절).

```go
// Baker 는 이 노드의 굽기다 — build · merge 단계 · 기동 정리 · 재개. Worker(단계)와 LowerGuard(광고 주기)가 나눠 쓴다.
type Baker struct {
	/* lowerRoot · scratch · node · label · instance · log · now
	   guard    *LowerGuard       Dir 을 빌리고 HoldBake · DropBake 를 부른다
	   helper   helperLauncher    merge-helper 를 여는 자리 — 시험이 namespace 없는 명령으로 바꿔 끼운다
	   mu       sync.Mutex        아래 넷을 지킨다
	   held     *heldBake         이 프로세스가 쥔 굽기 (build · merge 단계).  한 번에 하나 — 노드마다 임대 하나 (I1)
	   resuming bool              배경 재개가 굽기 잠금을 쥐고 도는 중
	   retryAt  time.Time         재개나 광고 주기의 정리가 실패한 뒤 이 노드가 다시 해 볼 때 (답 2 · 되물음 1 답 A · 10분)
	   lastErr  string            마지막 재개 · 정리 오류 — 바뀔 때만 로그
	   wg       sync.WaitGroup    배경 재개 — 데몬이 멈출 때 기다린다 */
}

// StartBaker 는 노드 기동의 한 줄이다 (cmd/enode/main.go · StartLowerGuard 뒤 · 광고 시작 전).
// 첫 줄에서 guard.OnStale(b.onStale) 를 등록한다 — 굽기 잠금을 못 잡아도 광고 주기의 정리와 재개는 열려 있다.
// 그다음 굽기 잠금을 해 보고, 되면 낡은 상태를 정리하거나 재개를 배경에 연다 (business-rules.md 12절).
// guard 가 nil 이면 nil 이다.
func StartBaker(ctx context.Context, guard *LowerGuard, scratch string, ident Identity, instance string,
	log *slog.Logger) *Baker

// Wait 는 배경 재개가 끝나기를 기다린다 — 데몬이 멈출 때 main 이 부른다.
func (b *Baker) Wait()

// Worker 가 kind 로 부른다 (claim.go 의 execute — 빈 argv 확인 앞).
func (w *Worker) runBuildStep(ctx context.Context, step *Step, log *slog.Logger)
func (w *Worker) runMergeStep(ctx context.Context, step *Step, log *slog.Logger)
```

- `Worker` 에 칸 하나 — `Bake *Baker`. nil 이면 굽기 단계를 거절한다
- `runBuildStep` · `runMergeStep` 의 이름은 `component-methods.md` 4.2 그대로다

---

## 3. `heldBake` — 이 프로세스가 쥔 굽기 하나 (계획 3.12 · 받는 일 40 · 41)

build 단계가 굽기 잠금을 잡을 때 만들고, committed 를 쓰고 잠금을 놓을 때 끝난다. 치우는 몸통이 굽기 하나에 한 번만 돌게
하는 자리다.

```go
type heldBake struct {
	mu      sync.Mutex
	run     string       // 굽기 Run
	step    int          // build 단계의 seq
	dir     *lower.Dir
	lock    *lower.Bake  // bake.lock — building 부터 committed 까지.  쥔 동안 이 참조를 붙들어 둔다 — os.File 은 참조가 끊기면
	                     // GC 의 finalizer 가 fd 를 닫아 flock 이 풀린다 ($GOROOT/src/os/file_unix.go:225 · QA 재검).  광고 주기의
	                     // 정리 (되물음 1 답 A) 가 산 굽기를 건드리지 않는 조건이다 (결정 44)
	pending string       // 대기 자리 <scratch>/pending/<lower 키>/<이름> — building 때 만든다 (4절)
	draft   *Draft       // pending 을 쓴 뒤에 있다
	merging bool         // 「합치기 시작」 표지 — merge 단계가 merging 을 쓰기 전에 mu 아래에서 적는다
	done    bool         // 몸통이 돌았거나 committed 를 썼다 — 더 할 일이 없다
}

// abandon 은 치우는 몸통이다 (business-rules.md 11절). 몇 번 불러도 한 번만 돈다. merging 표지가 있으면 아무것도
// 안 한다 — 합치기 본체를 끊지 않는다. 하는 일: 대기 자리를 그 자리의 scratch 의 trash 로 · WriteState(committed +
// last_attempt) · 굽기 잠금 Release · guard.DropBake · Baker.mu 아래에서 Baker.held 를 비운다 (그 굽기일 때만).
// 쓰기가 실패하면 할 수 있는 데까지 한다 — 옮기기가 실패해도 committed 를 쓰고, committed 가 실패해도 잠금은 놓는다.
func (h *heldBake) abandon(reason string, builds []lower.BuildRecord)

// startMerging 은 merge 단계가 merging 을 쓰기 직전에 부른다. 몸통이 먼저 돌았으면 false — merge 단계는 거기서 멈춘다.
func (h *heldBake) startMerging() bool
```

- **부르는 자리** — build 단계의 끝(실패 · pending 을 쓴 뒤 예산을 넘김) · 광고 응답의 몸통(`LowerGuard.HoldBake` 에 넘긴 함수 —
  `bakeGoneLocked`, `lowerguard.go:419`) · merge 단계의 실패 길(대기 상한 · 임대가 사라짐 · 시작 전 확인이 어긋남 · 데몬이 멈춤).
  계획 2.3 이 측정한 대로 광고 응답의 몸통과 Worker 의 임대 감시(`claim.go:655` ~ `:667`)가 같은 응답에서 함께 돌 수 있다 — 한 번만 돈다
- **held 를 비우는 자리는 넷이다** — 합쳐 끝남 · merging 뒤 멈춤 · 7.1 의 둘째 줄 (상태가 이 굽기의 것이 아님) · 몸통. 몸통이 비우지 않으면
  합칠 것 없는 굽기의 merge 가 7.1 의 둘째 줄로 실패하고, 그 노드는 재시작까지 모든 굽기를 bake_in_progress 로 거절한다 (QA S3)
- **held 가 있는 동안의 모든 오류 길은 몸통으로 끝난다** (결정 44 · QA 재검) — build 의 대기 자리 만들기 · building 쓰기 · merge 의
  ReadState · trash 만들기까지. 몸통 밖으로 끝나는 길은 merging 뒤 멈춤과 7.1 둘째 줄 둘이고 둘 다 held 를 비운다
- 재개는 `heldBake` 를 쓰지 않는다 (10절). 재개하는 쪽은 굽기 Run 의 임대가 없다

---

## 4. 대기 자리와 초안 — `bake.json` (계획 3.6 · 받는 일 21)

```text
   <scratch>/pending/<lower 키>/<이름>/          MkdirTemp (0700).  building 을 쓰기 전에 만든다
     upper/                                    세션의 upper 를 rename 한 번으로 (Keep.Upper)
     bake.json                                 초안 — metadata 에서 bake 칸만 빠진 것과 그 굽기의 주인
```

- **state.json 의 `pending_upper` 는 `<이름>/upper` 까지의 절대 경로다.** 합치기의 `merge.Paths.Upper` 가 곧 이 값이다. 대기 자리는
  그 부모, scratch 는 네 단계 위다 (upper -> 이름 -> 키 -> pending -> scratch)
- **이름에 run_id 를 쓰지 않는다** — 계약이 적는 아무 문자열이다 (`contract.go:1087` 은 빈 값만 본다). 경로는 state.json 이 든다
- lower 키 아래에 두는 까닭 — 한 scratch 를 여러 lower 의 노드가 나눠 쓸 수 있다. 굽기 잠금을 쥔 쪽이 자기 lower 키 아래만 치운다

```go
// Draft 는 bake.json 이다. 재개하는 노드가 metadata 를 쓰려면 build 가 안 사실이 필요하다 — state.json 은 형제가
// 광고마다 읽는 작은 파일이고 그 타입은 lower-state 의 것이라 여기 둔다. upper 와 함께 살고 함께 trash 로 간다.
type Draft struct {
	Schema          int                 `json:"schema"`            // 1
	Run             string              `json:"run"`               // 굽기 Run
	Node            string              `json:"node"`              // 구운 노드 — metadata 의 bake.node (US-6 · lower 를 바꾼 굽기)
	Source          lower.Source        `json:"source"`            // url · branch · repo_id · head · ir · pinned · sync_command · synced_at
	Builds          []lower.BuildRecord `json:"builds"`            // 돈 builds — 성공이면 모두 exit 0
	Environment     string              `json:"environment"`       // Record.PreparedEnvironment
	WorkspaceTarget string              `json:"workspace_target"`  // Record.WorkspaceTarget
	PreviousIR      *string             `json:"previous_ir"`       // 굽기 잠금을 잡은 뒤 읽은 lower 의 source.ir — 없으면 null
}

// metadataOf 는 초안에 bake 칸을 채운 metadata 다 — merge 단계면 resumed false · node 는 이 노드, 재개면 resumed true ·
// node 는 초안의 Node. 순수 함수다.
func metadataOf(d Draft, run, node string, mergedAt time.Time, resumed bool) lower.Metadata
```

- **`PreviousIR` 를 초안에 둔다** (계획 3.8 · 3.13 과 다른 자리). 굽기 잠금이 building 부터 committed 까지 쥐어져 있으므로 build 때
  읽은 lower 의 metadata 가 합치기 전의 것과 같다. 재개가 반쯤 합친 lower 에서 읽으면 틀린다 — upper 에 `.enode-metadata.json` 의
  whiteout 이 있으면 (sync 의 `git clean -x` 모양) 그 파일이 이미 없어졌을 수 있다. metadata 를 못 읽었을 때도 null 이다 — 옛 ir 이
  없는 것과 metadata 로는 구별하지 못하고 노드 로그의 경고가 구별한다
- **쓰는 법** — 임시 파일 · fsync · rename · 자리 디렉터리 fsync, 0600. `WriteState(pending)` 보다 먼저 디스크에 있다 — pending 이
  보이면 초안이 있다. 읽는 법 — `O_NOFOLLOW` · 보통 파일 · 1 MiB 까지 · schema 1
- **대기 자리는 committed 를 쓴 뒤에 옮긴다** — merging 인 동안은 초안이 늘 있다. 재개가 「이미 끝났나」를 초안의 synced_at · head 와
  metadata 로 대 본다 (business-rules.md 12.4)

---

## 5. IR 대조 — 입력 · 출력 · 판정 (물음 1 답 B · 계획 3.4 · 받는 일 3 · 4 · 53)

```go
// irProbe 는 세션 안에서 도는 고정된 POSIX sh 한 줄의 결과다 (business-rules.md 4절). 노드가 짓고 계약은 못 바꾼다.
// LC_ALL=C · GIT_TERMINAL_PROMPT=0 · GIT_CEILING_DIRECTORIES 는 그 한 줄 안에서 export 한다 — 세션 환경이 LC_ALL 을
// 프로필의 locale 로 덮는다 (runc_overlay_linux.go:1280 ~ :1284).
type irProbe struct {
	Mode   string   // "repo" (.repo/manifests) | "git" (워크스페이스 뿌리) | "none"
	Head   string   // git rev-parse HEAD
	Tagged string   // git rev-parse -q --verify refs/tags/$ENODE_IR^{commit} — 로컬에 없으면 ""
	Tags   []string // git tag --points-at HEAD — 이름순
	URL    string   // git config --get remote.origin.url — origin 이 없으면 "" (실패로 치지 않는다 · 결정 47)
	Branch string   // git rev-parse --abbrev-ref HEAD — detached 면 "HEAD" · 못 읽으면 ""
	Exit   int      // 셸의 exit — 0 이 아니면 Stderr 의 꼬리와 함께 「대조를 못 했다」
	Stderr string
}

type irOutcome int

const (
	irMatch      irOutcome = iota + 1 // 태그의 커밋이 HEAD 와 같다.  HEAD 에 다른 태그가 함께 있어도 된다 (계획 2.1)
	irNotLocal                        // 태그가 로컬에 없다 — sync 가 그 태그를 받아 와야 한다
	irElsewhere                       // 태그가 다른 커밋을 가리킨다 — sync 가 다른 곳에 닿았다
	irUnverified                      // 보는 git 이 없다 · git 이 실패했다 — 노드 쪽 오류 (원인 코드 없음)
)

// irVerdict 는 순수 함수다. 문장(영어)은 business-rules.md 4절 그대로 낸다 — irMatch 면 "".
func irVerdict(ir string, p irProbe) (irOutcome, string)
```

- 두 어긋남 (`irNotLocal` · `irElsewhere`) 이 원인 코드 `ir_mismatch` 다 — 완주 (DONE) 한 결과에 싣고 문장은 단계 로그에 쓴다 (물음 1 답 B).
  `irUnverified` 는 원인 코드가 없는 노드 쪽 오류 (FAILED) 다 — IR 과 HEAD 를 대 보지 못했으므로 어긋났다고 말할 수 없다. 그때 build 칸의
  head 는 "" · head_tags 는 null 이다 (6절)
- 칸의 이름은 계약 쪽이 이미 닫았다 — 환경 변수 `ENODE_IR` (`contract.EnvIR`, `internal/contract/bake.go:42`)

---

## 6. 결과 칸 — `BuildManifest` · `MergeResult` · 원인 코드 (물음 1 답 B · 계획 3.15 · 받는 일 14 · 16 · 31)

`internal/contract/result.go` 에 둘을 더한다 (더하기만 — step-phase 의 규칙 · 받는 일 16).

```go
// 원인 코드 (result.go:92 의 묶음 끝)
ReasonIRMismatch = "ir_mismatch" // 물음 1 답 B — sync 뒤 HEAD 가 계약의 IR 에 닿지 않았다.  builds 를 안 돌리고 manifest 를 안 낸
                                 // 완주 (DONE) 에 싣는다.  error 와 함께 오지 않는 첫 원인 코드다 (결정 54)

// BuildManifest 에 칸 하나 (result.go:164 ~ :174 의 끝)
// HeadTags 는 sync 뒤 HEAD 에 붙은 태그 전부다 (git tag --points-at HEAD · 이름순). 대조가 HEAD 와 태그를 읽었으면 늘
// 있다 — 태그가 없으면 [] 이다. 대조 전에 끝났거나 대조를 못 했으면 null 이다 — 재지 않은 것을 「없다」로 쓰지 않는다
// (FR-1 · 결과 확정의 교훈 · 되물음 5 답 A 의 (11)).
HeadTags []string `json:"head_tags"`
```

- **있는 칸의 뜻** — `IR` 은 계약의 IR 이 HEAD 와 맞았을 때 그 값이고, 어긋났거나 대조 전이면 null 이다. 「HEAD 에 정확히 붙은
  태그」(오늘 주석) 와 같은 값이 된다 — 계약의 IR 이 그 태그다. 주석만 고친다
- `Head` 는 대조를 돌렸으면 HEAD 의 커밋, 아니면 "" 다

Mediator 의 어휘 목록(`internal/store/claim.go:962` ~ `:963` 의 reason 줄)에 `contract.ReasonIRMismatch` 를 더한다 — 빠지면 받을 때마다
「어휘 밖 값」 경고가 남는다 (lower-state 가 `lower_changed` 를 더한 선례).

노드의 `Result` (`internal/enode/claim.go:185`) 에 칸 둘을 더한다. 이름과 모양은 `store.StepResult` 와 같다 (`store/claim.go:934` ~ `:935`).

```go
Build *contract.BuildManifest `json:"build,omitempty"` // build 단계 — 계약의 명령이 하나라도 돌았으면 늘 (계획 3.6)
Merge *contract.MergeResult   `json:"merge,omitempty"` // merge 단계 — 합쳤을 때만
```

```go
// buildManifestOf 는 lower 타입을 contract 타입으로 옮긴다. lower-state 의 시험(TestMetadataRecordsMatchTheContract)이
// 두 벌의 칸을 맞춰 둔다. 순수 함수다.
func buildManifestOf(sync lower.BuildRecord, builds []lower.BuildRecord, head string, ir *string,
	pinned *lower.Pinned, tags []string) contract.BuildManifest

// mergeOpsOf 는 merge.Result 를 셈 일곱으로 옮긴다 (계획 3.8).
//   replaced = Replaced + TypeChanged · created = Added · dirs = NewDirs · opaque = OpaqueDirs ·
//   whiteouts = Whiteouts · trashed = Discarded · attrs = MergedDirs
func mergeOpsOf(r merge.Result) contract.MergeOps
```

- `$OUT/manifest` 는 `BuildManifest` JSON 이고 `$OUT/merged` 는 `MergeResult` JSON 이다. 이름은 `contract.ArtifactManifest` ·
  `ArtifactMerged` (`bake.go`). 판정은 오늘의 produced 대조가 한다 — `Verify` 는 안 바뀐다 (contract-grammar 답 1)
- 재개로 끝낸 합치기의 셈은 노드 로그에만 남는다 — 보고할 단계가 없다 (받는 일 31 · 「재개면 더한다」는 보고할 자리가 없어 닫힌다)

---

## 7. `last_attempt` (계획 3.14)

타입은 lower-state 의 `lower.LastAttempt` 그대로다 (Run · At · Reason · Builds).

```go
// attemptReason 은 reason 칸이다 — 원인 코드가 있으면 그것, 없으면 영어 한 줄. 순수 함수다.
//   ir_mismatch · merge_wait_timeout · finalize_timeout · upload_timeout
//   sync exited 1 · build config-a exited 2 · cannot verify ir: ... · cannot pin the manifest: ...
//   aborted: lease expired · node stopped · the bake run ended before the merge · merge preflight: ...
//   abandoned: no process held the bake while it was building (또는 pending) — 결정 49 의 글자.  7.1 이 pending 쪽 글자를 같은
//   상수로 대 본다 (결정 52)
func attemptReason(code string, fallback string) string
```

- Builds 는 돈 것만이다 (답 3 — 첫 실패에서 멈추므로 실패한 구성이 끝이다). sync 는 싣지 않는다 — 칸의 모양이 builds 하나다

---

## 8. merge-helper 의 입출력 (계획 3.10 · 받는 일 24 ~ 26)

```go
// mergeHelperRequest 는 stdin 의 JSON 한 줄이다.
type mergeHelperRequest struct {
	Op    string `json:"op"`    // "preflight" | "apply"
	Upper string `json:"upper"` // pending_upper
	Lower string `json:"lower"` // 이 노드의 lower 루트
	Trash string `json:"trash"` // 대기 upper 가 있는 scratch 의 trash
}

// mergeHelperResponse 는 stdout 의 JSON 한 줄이다. exit 0 은 오류가 없었다 · 1 은 그 밖이다.
type mergeHelperResponse struct {
	Result *merge.Result `json:"result,omitempty"` // apply — 오류가 있어도 그때까지 센 것
	Error  string        `json:"error,omitempty"`
	Check  string        `json:"check,omitempty"` // *merge.PreflightError 면 그 Check — 어긋났다
	Kind   string        `json:"kind,omitempty"`  // "preflight" (어긋남) | "op" (*merge.OpError — 호출과 경로) | "io" (그 밖)
}

// RunMergeHelper 는 unshare 가 다시 실행한 숨은 입구다 — enode merge-helper. component-methods.md 4.4 의 겉면 그대로.
func RunMergeHelper(in io.Reader, out, errOut io.Writer) int
```

- **여는 argv** — `unshare --user --map-root-user --map-auto --fork --kill-child -- <enode> merge-helper`. `--mount` 가 없다 — trash-helper
  와 같은 모양이다 (`trash_linux.go:104`). 호스트의 마운트 번호를 그대로 본다 (9절의 확인이 뜻을 가진다)
- 안에서 `Options.Discard` 는 `scratch.Trash{Dir: Trash}.Move` 다. namespace 안의 root 로 돌고 권한을 내려놓지 않는다 (merge-rules)
- 프로세스 그룹 · Pdeathsig 는 trash-helper 와 같다 — 데몬이 죽으면 같이 죽는다

---

## 9. `merge.Check` 의 mount (lower-state 답 6 · 계획 2.2 · 받는 일 34)

`internal/merge` 에 줄 셋 (행렬 밖 — merge-rules 유닛의 파일).

```go
// merge.go:134 의 묶음 끝
CheckMount Check = "mount" // 셋이 같은 filesystem 이지만 같은 마운트가 아니다 — bind 별칭으로는 rename 이 EXDEV 다

// decide.go — checkDevices (:192) 옆의 순수 함수
func checkMounts(upper, lower, trash uint64) error // 다르면 *PreflightError{Check: CheckMount}

// merge_linux.go 의 resolve (:59) — 셋마다 Lstat 옆에서 statx(STATX_MNT_ID) 를 읽고, checkDevices (:86) 다음에 checkMounts
```

- 커널이 `STATX_MNT_ID` 를 mask 에 안 주면 `*PreflightError` 가 아니라 확인을 못 한 오류다 — `merge preflight: <path>: the kernel
  does not report a mount id`. runc-overlay 는 user namespace 의 overlay (커널 5.11) 가 있어야 하고 번호는 5.8 부터라 늘 있다
- 이 기계(커널 6.5)에서 bind 별칭의 번호가 다르고 st_dev 는 같았다 (계획 2.2 · 1423 대 1220)

---

## 10. 낡은 상태의 정리와 재개 (답 2 · 되물음 1 · 4 답 A · 계획 3.13 · 받는 일 29 · 50)

재개는 셋이 연다 — 기동 · 광고 주기 · build claim. 낡은 building · pending 의 정리는 기동 · 광고 주기 · build claim 이 한다 (되물음 1 답 A).
몸통은 각각 하나다 — 재개는 `resume`, 정리는 기동의 표 (business-rules.md 12.2) 한 벌.

```go
// resume 은 굽기 잠금을 쥔 채 도는 배경 고루틴의 몸통이다. 연 쪽이 TryBake 로 잡은 잠금을 넘겨받는다.
// 마감이 없다 — 배타는 형제가 모두 놓을 때까지 기다린다. ctx 는 데몬의 것이다.
func (b *Baker) resume(ctx context.Context, lock *lower.Bake, st lower.State, from string)

// onStale 은 광고 주기의 자리다 — LowerGuard 가 state 가 building · pending · merging 인 것을 본 광고에서 부른다
// (되물음 1 답 A). 막지 않는다.
//   이 프로세스가 굽기를 쥐었거나 재개 중이면        아무것도 안 한다
//   retryAt 전이면                                  아무것도 안 한다 (답 2 · 실패 뒤 10분)
//   TryBake 가 오류면                               로그 한 줄 (원인이 바뀔 때만) — 간격은 걸지 않는다
//   TryBake 가 안 되면                              아무것도 안 한다 — 주인이 살아 있거나 다른 쪽이 쥐었다.  실패가 아니다
//   되면                                            state 를 다시 읽는다 (광고가 읽은 것은 잠금 전의 값이다).
//                                                   merging 이면 resume 을 배경에 연다 · building · pending 이면 12.2 의
//                                                   정리를 하고 놓는다 · committed 면 잠금을 놓는다
func (b *Baker) onStale(st lower.State)
```

`lowerguard.go` 에 둘을 더한다 (행렬 밖 — lower-state 유닛의 파일).

```go
// OnStale 은 Baker 가 기동 때 채운다. BeforeAdvert 가 state.json 을 읽어 committed 가 아니면 (building · pending · merging)
// 부른다 — 광고 루프를 막지 않게 따로 돈다. nil 이면 안 부른다. g.bake 가 있으면 (이 프로세스가 merge 단계 중) 안 부른다.
// 처음 판의 이름은 OnMerging 이었다 — 되물음 1 답 A 로 merging 만이 아니게 되어 바꿨다.
func (g *LowerGuard) OnStale(f func(lower.State))

// Dir 은 연 상태 자리다. 못 열었으면 nil — 광고 주기가 다시 연다.
func (g *LowerGuard) Dir() *lower.Dir
```

- **재개는 `HoldBake` 도 `DropBake` 도 부르지 않는다.** `DropBake` 는 표지를 지금 metadata 의 것으로 새로 적는다
  (`lowerguard.go:504` ~ `:513`). 재개하는 노드가 합친 뒤에 부르면, 합치기 전에 그 노드에 매칭된 늦은 임대가 표지 비교 (`lateLocked` ·
  `:398`) 를 통과해 바뀐 lower 위에서 돈다. 부르지 않으면 표지가 합치기 전의 것으로 남아 그 임대는 lower_changed 로 거절된다.
  `HoldBake` 는 그 짝이라 부르지 않는다 — merging 동안은 어느 노드도 공유를 새로 잡지 않으므로 (광고 `:170` · `:215` · 늦은 임대 `:396` ·
  기동 `:95` ~ `:100`) 놓을 공유도 없다. 처음 판의 근거 (「재개하는 형제가 자기 Run 의 공유를 쥐고 있을 수 있다」) 는 코드에서 생기지
  않는 장면이었다 — merging 은 배타를 쥔 뒤에만 쓰이고 그때 공유는 모두 놓여 있다 (QA S6)
- 재개의 로그는 노드 로그다 — 단계가 없다

---

## 11. 대기 로그 (계획 3.9 · 받는 일 35 · 36 · 49)

```go
// waitLog 은 배타를 기다리는 동안의 줄이다. Exclusive 의 watch 가 10초마다 부른다. 쓸지는 이것이 정한다 — 처음 ·
// 쥔 쪽의 모습(노드 · 역할 · Run · acks · 기록 없는 쥔 쪽)이 바뀔 때 · 그 밖에는 5분마다. 순수하게 시험할 수 있게
// 시계와 쓰는 자리를 받는다.
type waitLog struct {
	deadline time.Time     // 노드 시계.  재개면 제로값 — 마감 줄을 안 쓴다
	every    time.Duration // 5분
	last     time.Time
	shape    string        // 앞 줄의 쥔 쪽 모습
}

// lines 는 이번 watch 에 쓸 줄이다. 안 쓸 때면 nil.
func (l *waitLog) lines(now time.Time, w lower.Waiting) []string
```

---

## 12. linux 와 그 밖

```text
   linux        unshare 로 merge-helper 를 연다 · statx(STATX_MNT_ID) (merge 의 확인)
   linux 밖      RunMergeHelper 는 exit 1 과 "merge-helper is supported on linux only".  Baker 는 만들어지지 않는다 —
                runc-overlay 런타임을 못 짓는다 (runc_overlay_other.go).  lower 는 같은 겉면이 ErrUnsupported 다
   32비트 arm    새 정수 칸이 없다 — 마운트 번호는 uint64 (statx 의 stx_mnt_id)
```

- `cmd/enodectl` 이 `internal/enode` 를 링크한다 (`go list -deps ./cmd/enodectl`). `internal/merge` 가 새로 딸려 들어간다 — 표준
  라이브러리와 `golang.org/x/sys` 만 쓰므로 `enodectl.exe` 의 net/http · crypto/tls 심볼 상한(requirements.md 5.1)과 무관하다. Code
  Generation 의 크로스 빌드가 확인한다
