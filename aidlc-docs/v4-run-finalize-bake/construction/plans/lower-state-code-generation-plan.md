# `lower-state` — Code Generation 계획

**유닛** `lower-state` (아래층 상태와 잠금 · 한 줄 순서의 여섯째) · **브랜치** `unit/lower-state` · **기준** `a3b1536`
(Functional Design 커밋) · **맡는 조각** 없음 · **병합 조건** 코드 검사 (`unit-of-work.md` 0절)

**이 계획이 이 유닛 Code Generation 의 기준이다.** 여기 없는 것은 짓지 않는다. 설계는
`construction/lower-state/functional-design/` 의 셋이다. 아래의 FD 는 그 셋의 줄임이다 — 「FD 규칙 5.2」는
`business-rules.md` 5.2절, 「FD 흐름 9.1」은 `business-logic-model.md` 9.1절, 「FD 엔티티 4절」은 `domain-entities.md` 4절이다.
「답 N」은 FD 계획(`plans/lower-state-functional-design-plan.md`)의 물음 N 에 받은 답이고 (아홉 모두 A), 「FD 계획 2.1」은 그 계획의
실측 절이다.

- **작성 시각**: 2026-09-27T01:45:37Z (Functional Design 승인 2026-09-27T01:27:20Z 뒤)
- **입력**: FD 셋과 FD 계획 · 유닛 정의 `unit-of-work.md` 0절 · 6절 · 10절 · 파일 행렬의 ⑥ 열 · `component-methods.md` 1절 · 4.3 · 6절 ·
  `requirements.md` FR-8 (기능 8 — 아래층의 상태 · 잠금 · drain · 준비도 점검) · FR-9 (기능 9 — 광고 키) · 5.1 ~ 5.3 · 코드 (줄 번호는 `a3b1536` 기준)
- **NFR 두 단계** (NFR Requirements · NFR Design — 비기능 요구와 그 설계): 사용자 결정으로 건너뛰었다
  (2026-09-27T01:27:20Z). 유닛 정의 6절이 NFR 에 둔 보안 · 동시성을 이 계획의 3절이 받는다

---

## 1. 유닛 맥락

**하는 일** — 새 패키지 `internal/lower` 가 lower 의 신원 · 상태 자리 · 잠금 셋 · 쥔 사람 기록 · metadata 읽기와 쓰기 · 마운트 훑기 ·
준비도 점검 셋의 판정을 맡는다. `internal/enode` 의 `LowerGuard` 가 광고 주기와 claim 에서 공유 잠금을 쥐고 놓고, 굽기 출처(bake)와
상태 자리 출처(lower)의 drain 과 광고 키 다섯을 싣는다. 준비도 점검은 이음매(`FactSource`) 하나로 Fact 셋을 더하고, smoke 도 공유를
쥔다. Mediator 는 실행 중 획득이 drain 을 보는 한 줄과 원인 코드 하나만 바뀐다 (답 2 — 이 유닛이 획득 경로를 고친다).

**완성하는 스토리** — US-1 (누가 건 drain 인지 — 이 유닛은 굽기 출처와 상태 자리 출처) · US-4 (not ready 가 무엇이 어긋나서인지).
**완료 조건** 1 · 3 ② (scratch 가 다른 filesystem 이면 not ready 이고 이유를 이름으로 말한다).

**맡는 조각** — 없다. 조각 8 (배타와 대기 · 사람 · SunnyVM) 의 준비도 점검 부분은 bake 유닛이 조각 8 을 돌릴 때 확인한다.

**의존** — 앞 유닛 다섯은 `main` 에 있다. 기대는 것은 trash 의 `DrainSource` · `combineDrain` · `StatusBook` · 제어판의 출처 줄,
contract-grammar 의 `workspace.writes` 요구 (굽기 예시가 이 키를 요구한다), finalize 의 `StepRuntime` 이다.

**뒤 유닛이 기대는 것** — bake 가 `lower.Open` · `TryBake` · `Bake.WriteState` · `Exclusive` · `Waiting` · `ForeignMounts` · `WriteMetadata` ·
`LowerGuard.HoldBake` · `DropBake` 를 쓴다 (FD 흐름 5 · 6 · 12절). checkpoint 가 `RuntimeCapability` 에 Capture 칸을 더한다.

**새 이름**

```text
   internal/lower (새 패키지)   Key · ParseKey · Root · ReadRoot · Dir · Open · Peek · Identity · Verdict (다섯 값) ·
                               Phase (넷) · State · Owner · LastAttempt · BuildRecord · Role (둘) · Holder · Shared · Exclusive ·
                               Bake · Waiting · Metadata · Source · Pinned · BakeRecord · ReadMetadata · WriteMetadata · Marker ·
                               Mount · MountScan · ForeignMounts · Finding · Cause (둘) · Check · ErrUnsupported
                               (안 내보내는 것) identityVerdict · permVerdict · parseMountinfo · mountLine · overlaysOn ·
                               pollEvery · 자리와 파일을 여는 함수 · 임시 파일 쓰기
   internal/enode              RuntimeCapability · StepRuntime.Capability · WorkspaceWrites · DrainBake · DrainLower ·
                               KeyWorkspaceWrites · KeyIR · KeyRepoBuilt · KeyBakeRun · KeyBakeResumed · LowersDir ·
                               LowerGuard · StartLowerGuard · LowerChangedError · LogNotice ·
                               ExecutionRuntimeVerifier.Notice · ExecutionRuntimeVerifier.Facts · Advertiser.Guard ·
                               Advertiser.Writes · Worker.Guard
   internal/environment        FactSource
   internal/contract           ReasonLowerChanged · ValidBuildName · IRProblem (둘은 이름만 바뀐다 — 4절 ⑲)
```

**경계** — `internal/lower` 는 표준 라이브러리와 `golang.org/x/sys` 만 쓴다. `internal/scratch` · `internal/merge` · `internal/environment` ·
`internal/contract` 도 안 된다 (봉인 · FD 규칙 14절). Mediator(`cmd/mediator`)는 `internal/lower` 를 안 가져다 쓴다. `go.mod` 를 안
움직인다 — `x/sys` v0.47.0 에 `STATX_MNT_ID` · `Statx_t.Mnt_id` · `Flock` 이 있다.

**크로스 빌드** — 판정(`judge.go`)은 모든 플랫폼에서 빌드된다. statfs · statx · flock · /proc 읽기는 `_linux.go` 에만 있고,
`lower_other.go` 가 같은 겉면을 `ErrUnsupported` 로 채운다 (`unit-of-work.md` 10절). linux/arm 은 32비트다 — `Statfs_t.Fsid.Val` 은
`[2]int32` 라 uint32 로 바꿔 붙인다 (FD 엔티티 11절). 유닛 정의 10절이 이 계획에 맡긴 확인은 Step 17 의 크로스 빌드 셋이다.

---

## 2. 고치는 파일

| 파일 | 새 · 고침 | 행렬 | 무엇 |
|---|---|---|---|
| `internal/lower/lower.go` | 새 | 있음 | 패키지 문서 · 타입 전부 · 오류 문구 (FD 규칙 13절) · `ErrUnsupported` |
| `internal/lower/judge.go` | 새 | 있음 | 키 글자 · `ParseKey` · `identityVerdict` · `permVerdict` · mountinfo 풀기 · `overlaysOn` · 점검 판정 · metadata 거르기 |
| `internal/lower/root_linux.go` | 새 | 있음 | `ReadRoot` (statfs · statx · 마운트 번호가 없을 때 /proc/self/mountinfo) |
| `internal/lower/perm_linux.go` | 새 | 있음 | 자리와 파일을 symlink 없이 열기 · 주인과 권한 확인 · 좁히기 · 임시 파일 쓰기 (3.1 · 4절 ① ②) |
| `internal/lower/dir_linux.go` | 새 | 있음 | `Open` · `Peek` · lower.json · `ReadState` · `ReadMetadata` · `WriteMetadata` |
| `internal/lower/lock_linux.go` | 새 | 있음 | `TryShared` · `Update` · `Release` · `WaitShared` · `Exclusive` · `TryBake` · `Bake.WriteState` · `Holders` |
| `internal/lower/mounts_linux.go` | 새 | 있음 | `ForeignMounts` |
| `internal/lower/check_linux.go` | 새 | 있음 | `Check` (점검 셋) |
| `internal/lower/lower_other.go` | 새 | 있음 | linux 밖 — 같은 겉면이 `ErrUnsupported` |
| `internal/lower/*_test.go` | 새 | 시험 | `judge_test.go` (모든 플랫폼) · `root` · `perm` · `dir` · `lock` · `mounts` · `check` · `concurrency` 의 `_linux_test.go` · `lower_integration_test.go` (`integration` 태그) |
| `internal/enode/lowerguard.go` | 새 | 있음 (새 파일) | `LowerGuard` · `StartLowerGuard` · 출처 둘 · 늦게 보인 임대 · `LowerChangedError` |
| `internal/enode/advertkeys.go` | 새 | 있음 (새 파일) | 예약 키 · metadata 에서 광고 키 · 광고 능력의 복사본에 싣기 |
| `internal/enode/lowercheck.go` | 새 | 있음 (새 파일) | `ExecutionRuntimeVerifier` 의 type (옮김) · `Facts` · smoke 의 잠금 · `LogNotice` |
| `internal/enode/runtime.go` | 고침 | 있음 | `RuntimeCapability` · `StepRuntime.Capability()` · native 는 in-place · `WorkspaceWrites` |
| `internal/enode/runc_overlay_linux.go` | 고침 | 있음 | `Capability()` 는 isolated · `Verify` 가 smoke 앞뒤로 공유를 쥐고 놓는다 · type 줄을 옮긴다 |
| `internal/enode/runc_overlay_other.go` | 고침 | 밖 (FD 흐름 11절) | `Capability()` · type 줄을 옮긴다 |
| `internal/enode/advertise.go` | 고침 | 있음 | `Guard` · `Writes` · 출처와 키 · 응답 뒤 `AfterResponse` |
| `internal/enode/claim.go` | 고침 | 밖 (FD 흐름 11절) | `Worker.Guard` · `execute` 처음의 `OnClaim` 과 `StepDone` · 거절 보고 |
| `internal/enode/policy.go` | 고침 | 밖 (FD 흐름 11절) | `DrainBake` · `DrainLower` · 172행 주석 |
| `internal/enode/paths.go` | 고침 | 밖 (FD 흐름 11절) | `LowersDir` |
| `internal/environment/check.go` | 고침 | 있음 | `FactSource` · host 점검 뒤 · prepared environment 앞 (286행 앞) |
| `internal/contract/result.go` | 고침 | 밖 (FD 흐름 11절) | `ReasonLowerChanged` |
| `internal/contract/bake.go` | 고침 | 밖 (FD 흐름 11절) | `buildName` -> `ValidBuildName` · `irProblem` -> `IRProblem` |
| `internal/store/acquire.go` | 고침 | 밖 · Mediator | `tryGrab` 이 drain 목록을 busy 에 합친다 |
| `internal/store/claim.go` | 고침 | 밖 · Mediator | `OutOfVocabulary` 의 reason 목록에 `lower_changed` (962행) |
| `internal/panel/page.go` | 고침 | 밖 (FD 흐름 11절) | `drainName` · `drainLift` 에 bake · lower (308 ~ 311행) |
| `internal/panel/boundary_test.go` | 고침 | 있음 | 금지 하나 · 봉인 하나 |
| `cmd/enode/main.go` | 고침 | 있음 | `StartLowerGuard` 한 줄 · 칸 넷 · verifier 의 `Notice` (4절 ⑧) |
| `cmd/enode/environment.go` | 고침 | 있음 | verifier 의 `Notice: os.Stderr` |
| 시험 (고침 · 새) | 고침 · 새 | 시험 | `internal/enode`: `runtime_test.go` · `finalize_worker_test.go` (가짜 런타임 넷에 `Capability`) · `drain_test.go` · 새 `lowerguard_linux_test.go` · `advertkeys_test.go` · `lowercheck_linux_test.go` · `lowerclaim_linux_test.go` · `paths_test.go` / `internal/environment/check_test.go` / `internal/contract` 의 굽기 시험 / `internal/store/exited_test.go` / `internal/api/api_test.go` (획득) / `internal/panel/panel_test.go` / `cmd/enode/environment_test.go` |

**확인만 하고 안 고친다** — `internal/enode/detect.go` (예약 키는 광고 주기가 거른다 · 4절 ⑤) · `leases.go` (응답의 임대 목록을 그대로
넘긴다) · `status.go` (새 칸이 없다 · 4절 ④) · `internal/panel/view.go` (출처의 Kind 는 문자열 그대로 지난다). 넷 다 행렬에 있다.

행렬 밖 파일은 아홉이다 (시험 파일 빼고 · 표의 「밖」 줄). 그 diff 를 `code-summary.md` 에 따로 적는다 (CONVENTIONS 3.5 의 가져갈 것).

---

## 3. NFR 두 단계를 건너뛰어서 이 계획이 받는 것

유닛 정의 6절이 NFR 에 둔 둘이다 — 보안 (상태 자리 권한) · 동시성 (같은 기계의 여러 노드가 같은 파일과 잠금을 쓴다). FD 규칙 1절이
「이미 있는 것의 권한을 어떻게 확인할지는 NFR 이다」로 넘긴 것을 3.1 이 정한다.

**측정** — 2026-09-27 · 이 기계 (커널 6.5 · ext4 · uid 1000 · 특권 없음). 측정 프로그램은 스크래치 폴더에 두고 끝나고 지웠다. 작업
트리는 바꾸지 않았다. SunnyVM (커널 7.0) 은 ssh 로 읽기만 했다.

```text
   측정 1  umask 0000 · 0002 · 0022 · 0077 에서 MkdirAll(0700) · OpenFile(0600) · CreateTemp     늘 0700 · 0600 · 0600
   측정 2  lowers 가 밖 디렉터리를 가리키는 symlink 일 때 MkdirAll(lowers/<key>)                  오류 없음 — 밖에 <key> 를 만들었다
   측정 3  잠금 파일이 symlink 일 때 OpenFile(O_CREATE|O_NOFOLLOW)                                  ELOOP.  가리키는 곳이 없어도 ELOOP 이고 아무것도 안 만든다
           같은 것을 O_NOFOLLOW 없이 · 가리키는 곳이 없을 때                                        성공 — 밖에 파일을 만들었다
   측정 4  state.json 이 symlink 일 때 임시 파일을 rename                                            링크가 보통 파일로 바뀐다.  가리키던 파일은 그대로
   측정 5  0755 디렉터리를 O_DIRECTORY|O_NOFOLLOW 로 열어 fchmod(0700)                               0700.  symlink 를 이렇게 열면 ENOTDIR.  파일 0644 도 fchmod 로 0600
   측정 6  쓰는 쪽 16 이 lower.json 을 500 번씩 임시 파일 + rename · 읽는 쪽 4                         임시 이름이 하나: rename 실패 2,101 · 깨진 읽기 6,955 / 108,802
                                                                                                 CreateTemp (쓰는 쪽마다 이름이 다르다): 실패 0 · 깨진 읽기 0 / 180,539
   측정 7  잠금을 쥔 채 자식 프로세스를 exec                                                         os.OpenFile 로 연 fd 는 자식에게 안 보인다.  O_CLOEXEC 없는 syscall.Open 은 보인다
   측정 8  flock LOCK_SH|LOCK_NB 와 풀기                                                            0.6 µs
           CreateTemp + rename 한 번 (기록 파일)                                                   0.12 ms.  파일과 디렉터리 fsync 를 더하면 4.9 ms
           Holders 모양의 읽기 — 기록 여덟 · 살아 있는 것 넷                                          0.07 ms
           O_RDONLY 로 연 fd 에 LOCK_EX                                                          된다
   측정 9  go test -race -count=1 ./internal/enode/ (지금 코드)                                     통과 · 41 초 — 경합 0 에서 시작한다
   SunnyVM ~/.local/state 0700 · ~/.local/state/enode 0755 · lowers 없음 · ssh 셸 umask 0002 ·
           노드 둘 (bench · yocto 설정) 이 2026-09-22 판 (8be73d2) 으로 떠 있다 · /srv/yocto 는 마운트 / (sunny 0755)
   이 기계  ~/.local/state 0755 · ~/.local/state/enode 0755 · umask 0022 · /tmp 와 /dev/shm 에서 btime 이 0 이 아니다
```

### 3.1 보안 — 상태 자리 권한

**요구** — 「`~/.local/state/enode/lowers/…` 는 노드 사용자 전용 권한이다」 (`requirements.md` 5.3 보안 표). 만들 때 0700 · 0600 은
FD 규칙 1절이 정했고 측정 1 이 umask 와 무관함을 보였다. 이미 있는 것은 아래 표대로 확인한다.

```text
   자리                                   데몬 (Open)                                         env check · smoke (Peek · WaitShared)
   lowers · <key> · holders   디렉터리    MkdirAll 뒤 O_DIRECTORY|O_NOFOLLOW 로 다시 연다       같게 연다.  만들지 않는다
                                          (측정 2 — MkdirAll 은 symlink 를 따라간다) · fstat
     symlink · 디렉터리가 아님             오류 — 자리를 열지 않는다 (lower 출처 drain)           lower.identity external-blocked (cannot read <path>)
     주인이 노드 uid 가 아님               오류 — 고치지 않는다                                   같다
     남에게 열린 비트 (0o077)              좁힌다 — fchmod 0700 · 자리마다 로그 한 줄              ready 그대로.  observed 끝에 한 마디 (4절 ⑯ 아래)
   lower.lock · bake.lock ·               os.OpenFile(O_RDWR|O_CREATE|O_NOFOLLOW, 0600) · fstat   WaitShared 만 lower.lock 을 연다 —
   holders/<node>.lock    파일            보통 파일 · 주인이 노드 uid · 열린 비트는 fchmod 0600    O_RDONLY|O_NOFOLLOW.  없으면 잠그지 않는다 (4절 ⑰)
   lower.json · state.json ·              읽기 — O_NOFOLLOW · 보통 파일 · 주인 · 1 MiB 까지.  어긋나면 lower.json 은 Broken (FD 규칙 2절),
   holders/*.json                           state.json 은 못 읽음 (lower 출처 drain · FD 규칙 3절)
                                          쓰기 — 같은 디렉터리의 CreateTemp (0600) 뒤 rename.  덮어쓰면 늘 0600 이다 (측정 1 · 4)
   부모 ($HOME/.local/state/enode)          안 본다.  부모가 남에게 열려 있으면 lowers 를 바꿔치기할 수 있지만 주인과 symlink
                                          확인에 걸려 거절로 끝난다 — 안의 것이 새지 않는다.  만들 때만 0700 (MkdirAll)
```

**좁히고 거절하지 않는 까닭** — 요구가 말하는 것은 권한의 상태다. 자리는 데몬만 만들고 쓴다 (FD 규칙 1절). 느슨한 비트는 그 자체로
공격이 아니고 (옛 도구 · 사람의 chmod) 데몬이 특권 없이 고칠 수 있다. 거절하면 형제가 lower 출처 drain 에 남아 사람이 올 때까지
빠진다. 주인이 다른 것과 symlink 는 특권 없이 고칠 수 없고 남이 놓았을 수 있어서 거절한다 — 측정 3 은 그 symlink 를 따라가면 밖에
파일이 생긴다는 것을 보였다. 정책 파일의 권한은 경고만 하는 선례가 있다 (`policy.go:143`) — 그 파일은 소유자의 것이라 데몬이 안
고친다. 상태 자리는 데몬의 것이다.

**Peek 이 느슨한 비트를 보면** — 판정은 그대로 두고 lower.identity 의 observed 끝에 `; the node narrows loose permissions on start` 를
붙인다. Peek 은 고치지 않는다 (ADR-073 — 점검은 고치지 않는다).

**시험** (Step 5) — 새 자리의 권한이 0700 · 0600 · 0755 자리와 0644 잠금 파일을 Open 이 좁힌다 · Peek 은 안 좁힌다 · lowers · `<key>` ·
holders · lower.lock 이 symlink 면 오류이고 밖에 아무것도 안 생긴다 (가리키는 곳이 없는 symlink 포함) · 주인이 다른 자리 (4절 ⑳) ·
`permVerdict` 표 (주인 · 비트 · 종류 · 디렉터리와 파일).

### 3.2 동시성 — 같은 기계의 여러 노드가 같은 파일과 잠금을 쓴다

```text
   같은 lower 를 쓰는 쪽                      무엇을                                           부딪히는 자리
   형제 데몬 (노드마다 하나 · 프로세스 따로)    lower.json 쓰기 · lower.lock 공유 · 쥔 사람 기록     두 데몬이 거의 같은 때 lower.json 을 쓴다 (FD 규칙 2절)
   한 데몬 안의 두 고루틴                      LowerGuard (광고 주기 · Worker)                     Advertiser 와 Worker 가 같은 구조체를 부른다
   env check 의 smoke (다른 프로세스)           lower.lock 공유 몇 초 (기록 없음)                    배타를 기다리는 merge
   굽는 노드 (bake 유닛)                       bake.lock · lower.lock 배타 · state.json 쓰기        형제의 읽기 · 형제의 공유 · 자기 공유 (4절 ⑪)
```

**이 계획이 정하는 규칙**

- **임시 파일 이름은 쓰는 쪽마다 다르다** — `os.CreateTemp` 로 같은 디렉터리에 (측정 6 · 4절 ①)
- **잠금 fd 는 자식에게 안 샌다** — `os.OpenFile` 로 연다 (측정 7 · 4절 ③)
- **LowerGuard 는 mutex 하나다.** 잡은 동안 하는 일은 파일 호출뿐이다 (측정 8 — µs 에서 1 ms 아래). 막히는 호출(배타 대기)은 하지
  않는다. 합치기 없이 끝난 굽기의 몸통(abandon)은 따로 도는 고루틴이다 (FD 규칙 10절)
- **flock 은 열린 파일마다다** (FD 계획 2.1). 시험은 한 프로세스의 `Dir` 둘로 형제를 흉내 내고, 진짜 자식 프로세스 하나를 더한다

**시험** (Step 8 · Step 10)

```text
   자식 프로세스 형제       시험 바이너리를 다시 띄운 자식이 공유를 쥔다 -> 부모의 Exclusive 가 기다린다 -> 자식을 SIGKILL ->
                           pollEvery 한 번 안에 잡힌다.  걸린 시간을 적는다
   Open 여럿 동시          고루틴 16 이 같은 키를 Open (Dir 마다 fd 따로) -> 모두 성공 · lower.json 이 JSON 이고 신원 칸이 같다
   쓰기와 읽기 동시        Bake.WriteState 200 번 · ReadState 넷 -> 오류 0 · 깨진 읽기 0.  기록 Update 와 Holders 도 같은 모양
   fd 새지 않음            공유 · 배타 · 굽기 · 기록 잠금을 쥔 채 /bin/sh 로 /proc/self/fd 를 읽는다 -> 상태 자리의 파일이 없다
   배타가 굶지 않는다       state 가 pending 이면 LowerGuard 가 공유를 새로 안 잡는다 -> 기다리던 Exclusive 가 잡힌다 (Step 10)
   경합 검사               go test -race 를 internal/lower 와 internal/enode 에 (Step 17).  CI 는 -race 를 안 돈다 — 이 단계에서 한 번
```

**측정 자리** (Step 16) — 벤치마크 셋을 이 기계에서 한 번 돌려 `code-summary.md` 에 적는다. 기본 `go test` 에서는 안 돈다.

- `BenchmarkBeforeAdvert` — 임시 폴더의 진짜 자리에서 광고 한 번의 몫 (ReadState · 기록 맞추기 · ReadMetadata). 측정 8 로 보면 1 ms
  아래다. FD 흐름 12절이 NFR 에 둔 「광고마다 metadata 읽기」의 값이다
- `BenchmarkHolders` — 기록 여덟
- `BenchmarkForeignMounts` — 이 기계의 같은 uid 프로세스 전부 (FD 계획 2.6 은 4.7 ms)

값이 이 추정보다 한 자리 이상 크면 원인을 찾아 적는다. 값을 맞추려고 규칙을 바꾸지 않는다.

---

## 4. 이 계획이 정한 것 — FD 에 적히지 않은 자리

**① 임시 파일 이름.** state.json · lower.json · 기록 파일은 같은 디렉터리의 `os.CreateTemp` (이름 `.<파일>.tmp-<무작위>`) 에 쓰고 rename
한다. 측정 6 — 이름이 하나면 두 데몬의 쓰기가 서로의 임시 파일을 덮어 rename 이 실패하고 반쯤 쓴 파일이 읽힌다. 읽는 쪽은 점으로
시작하는 이름을 건너뛴다. 죽은 쓰기가 남긴 임시 파일은 지우지 않는다 — 지우는 쪽이 다른 데몬의 쓰는 중인 파일과 경쟁한다. 예외는
⑭ 의 metadata 다.

**② fsync.** state.json · lower.json · metadata 는 파일과 디렉터리를 fsync 한다 (FD 규칙 3절 · 4.9 ms · 전이 때만 쓴다). 쥔 사람 기록은
fsync 하지 않는다 (0.12 ms) — 살아 있음은 잠금이 말하고, 전원이 나가면 잠금도 기록도 함께 무의미해진다.

**③ 잠금 파일을 여는 함수.** `os.OpenFile(path, O_RDWR|O_CREATE|syscall.O_NOFOLLOW, 0600)` — Go 가 O_CLOEXEC 를 붙인다 (측정 7).
디렉터리를 `unix.Open` 으로 여는 자리는 O_CLOEXEC 를 직접 적는다. 시험이 자식 프로세스의 fd 목록을 본다 (3.2).

**④ 광고 키를 더하는 자리와 상태 파일.** 광고 주기는 `snap.Caps` 를 복사한 뒤(`maps.Clone`) 키를 더한다. 원본은 `a.status().SetCaps(snap)`
(`advertise.go:186`) 가 상태 파일 장부에 넘긴 것이라, 원본에 쓰면 삭제자 고루틴이 상태 파일을 쓰며 같은 map 을 읽는 경합이 된다.
상태 파일은 탐지 능력 그대로다 — `SetCaps` 는 시각(At)이 같으면 안 쓰므로 광고 키를 넣으면 ir 이 바뀌어도 파일이 안 바뀐다.
그래서 `status.go` 를 안 고치고 제어판에 ir · bake 키를 보이지 않는다 (FD 규칙 9절 — 상태 파일은 새 칸이 없다).

**⑤ 예약 키 거르기.** 예약 키 다섯(`workspace.writes` · `ir` · `repo.built.` 로 시작하는 키 · `bake.run` · `bake.resumed`)은 탐지가 안 낸다
— 능력에 있으면 라벨에서 왔다. 광고 주기가 복사본에서 빼고 키마다 처음 한 번 `label ignored; <key> is set by the node` 를 쓴다
(FD 규칙 11절). 뺀 뒤 `hasCapability` 가 거짓이면 그 능력을 뺀다 — `detect.go` 의 「할 줄 아는 것이 없으면 광고하지 않는다」 그대로다.
`detect.go` 는 안 고친다.

**⑥ workspace.writes 의 값과 자리.** `Advertiser.Writes` — `""` 면 `in-place` (런타임이 없는 노드 · ADR-077 §8). `main.go` 가
`enode.WorkspaceWrites(stepRuntime)` 로 채운다 (nil 이면 in-place). 능력 목록의 모든 능력(agent.reason 또는 오케스트레이션)에 싣는다.
runc-overlay 키 넷(ir · repo.built.* · bake.*)은 `Guard.BeforeAdvert` 가 준다. 능력이 0 이면 싣지 않는다.

**⑦ `ExecutionRuntimeVerifier` 의 type 자리.** 오늘은 linux 와 그 밖 파일에 따로 선언돼 있다 (`runc_overlay_linux.go:1308` ·
`runc_overlay_other.go:25`). `Notice io.Writer` 칸을 한 번만 두려고 type 을 `lowercheck.go` 로 옮기고 `Verify` 는 제자리에 둔다.
데몬은 `enode.LogNotice(log)` 를 넘긴다 — 줄마다 노드 로그의 Info 한 줄이 되는 io.Writer 다.

**⑧ `main.go` 의 모양.** `cmd/enode` 는 80.4% 로 하한에 가깝다 (FD 흐름 7절). runc-overlay 갈래는 기본 시험이 안 지나므로 거기에
문장을 더하지 않는다.

```text
   바꾸는 문장    scratchDir = binding.Scratch          ->  scratchDir, lowerRoot = binding.Scratch, binding.Workspace
                 enode.ExecutionRuntimeVerifier{}       ->  enode.ExecutionRuntimeVerifier{Notice: enode.LogNotice(log)}
   더하는 문장    guard := enode.StartLowerGuard(lowerRoot, ident, log)     lowerRoot 가 "" 면 nil.  오늘의 기동 시험이 지난다
   칸만 더함      Advertiser{..., Guard: guard, Writes: enode.WorkspaceWrites(stepRuntime)} · Worker{..., Guard: guard}
```

`StartLowerGuard` 는 `LowersDir` · `ReadRoot` · `Open` · 첫 metadata 표지 · 「놓은 상태」를 한 번에 한다 (FD 흐름 7절 2 · 3 · 5).
자리를 못 열면 nil 이 아니라 `openErr` 를 든 LowerGuard 를 돌려준다 — 광고마다 다시 연다 (FD 규칙 9절).

**⑨ `OnClaim` 의 자리와 짝.** `claim.go:509` 의 임대 확인 바로 뒤 · 링 비우기(518행) 앞이다. 부른 곧바로 `defer w.Guard.StepDone(step)`
을 건다 — 거절이어도 짝을 맞춘다. defer 차례가 거꾸로라 세션의 `defer Close`(673행) 보다 늦게 돈다 (FD 흐름 3절 「세션을 닫은 뒤」) ·
패닉에도 돈다. nil 수신자는 아무것도 안 한다 (native 노드 · 시험). 굽기 단계의 판정은 `step.Effect == contract.EffectPrepare` 이거나
`step.Kind` 가 `build` · `merge` 다 — 둘은 계약이 짝으로 묶는다 (`effect.go:151`).

**⑩ 거절 보고의 모양.** `Result{Node, Error, Reason}` — `Reason` 은 `*LowerChangedError` 일 때만 `lower_changed`. 워크스페이스 준비 ·
$IN · 세션 · exited 가 없다. `Environment` 는 `report()` 가 오늘처럼 채운다. 문구는 FD 규칙 13절의 둘 그대로다.

**⑪ HoldBake 부터 DropBake 까지 공유를 쥐지 않는다.** `HoldBake` 를 부를 때 쥐고 있으면 놓고, `DropBake` 전에는 `BeforeAdvert` 도
늦게 보인 임대도 공유를 잡지 않는다. FD 계획 2.1 다섯째 줄 — 한 프로세스의 fd 둘이 부딪친다. 기동 때 재개(bake 유닛)는 광고 주기와
따로 돌며 배타를 기다리는데, 그동안 광고 주기가 공유를 쥐면 자기 합치기를 막는다. FD 흐름 1절의 build claim 에서 놓는 것(FD 규칙
5.3)은 그대로이고, build 가 끝나 `HoldBake` 가 오기 전까지는 광고 주기가 공유를 다시 쥘 수 있다 — 굽기 잠금을 이 노드가 쥐고 있어
다른 합치기가 없으므로 해가 없다. bake 에 넘긴다: 배타를 기다리기 전에 `HoldBake`.

**⑫ 임대가 0 이 되면 역할을 candidate 로.** 쥔 채로 임대 목록이 비면 기록의 역할을 candidate 로 · Run 을 비워 다시 쓴다. FD 규칙 7절의
「다시 쓰는 때」에 역할이 바뀔 때가 있어 이 경우를 포함한다고 읽었다.

**⑬ Reused 에서 last_attempt 를 지우는 법.** state.json 을 쓰는 쪽은 굽기 잠금의 주인이다 (FD 엔티티 3절). `Open` 이 Reused 를 보면
`TryBake` 로 잠깐 쥐고 `WriteState` 로 지운 뒤 놓는다. 못 쥐면(다른 노드가 막 굽기를 시작했다) 지우지 않고 로그 한 줄 — 그 굽기가
building 을 쓰며 상태를 새로 쓴다.

**⑭ metadata 의 타입 두 벌.** `internal/lower` 의 `BuildRecord` · `Pinned` 는 봉인 때문에 `contract` 의 것과 따로 둔다 (JSON 칸은 같다 ·
`component-methods.md` 1.4). 같은 모양인지 `internal/enode` 의 시험이 본다 — `contract.BuildRecord` 를 JSON 으로 쓰고
`DisallowUnknownFields` 로 `lower.BuildRecord` 에 푼 뒤 되돌려 같은지. `WriteMetadata` 는 lower 뿌리에 남은 옛
`.enode-metadata.json.tmp-*` 를 먼저 지운다 — 배타 아래에서만 불리므로 경쟁이 없고, 합친 lower 에 남으면 다음 빌드가 그 파일을 본다.

**⑮ bake.run 의 상한.** 비어 있지 않고 128 바이트 이하일 때만 싣는다 (IR 의 상한과 같다 · `bake.go:167`). metadata 1 MiB 안에서 광고
하나가 커지지 않게 한다. 넘으면 `bake.run` · `bake.resumed` 를 빼고 로그 한 줄 (원인이 바뀔 때만).

**⑯ `lower.Check` 와 home.** `Check(lowers, lowerRoot, scratch, uid)` 의 `lowers` 가 `""` 면 lower.identity 를 내지 않는다. `Facts` 가
`LowersDir` 의 오류를 보면 `cannot find the home directory: <원인>` 의 lower.identity 를 스스로 낸다 (external-blocked · FD 규칙 12.3).
`internal/lower` 는 home 을 모른다 — 뿌리는 부르는 쪽이 준다 (답 9).

**⑰ smoke 의 잠금 자리.** `Verify` 안에서 `newRuncOverlayRuntime` 뒤 · `MkdirAll(scratch)` 앞에서 `Peek` 과 `WaitShared` · 세션을 닫은
뒤 놓는다 (defer). 자리가 없으면 안 잠근다 (FD 규칙 8.3). 자리는 있는데 lower.lock 이 없어도 안 잠근다 — `Peek` 은 만들지 않고, 데몬
기동 때 점검이 `Open` 보다 먼저 돌므로 lower.lock 을 요구하면 데몬이 영영 못 뜬다. 잠금이 없는 파일은 누구도 쥘 수 없으므로 그 순간
합치기도 없다. `LowersDir` 오류는 smoke 오류다 — 합치기가 없다는 것을 보일 수 없다. Notice 줄은 기다리기 시작할 때와 30초마다
(FD 규칙 13절 smoke).

**⑱ 배타 대기의 1초.** 패키지 변수 `pollEvery` 다. 시험만 줄인다. `WaitShared` 는 FD 대로 간격을 인자로 받는다.

**⑲ contract 판정 내보내기.** `buildName` · `irProblem` 을 이름만 바꾼다 (부르는 곳 둘 · `bake.go:118` · `:133`). 감싸는 함수를 따로 두면
같은 규칙에 이름이 둘이 된다.

**⑳ 주인이 다른 자리의 시험.** 특권 없이 남의 파일을 만들 수 없다. root 소유의 `/` 를 lowers 로 넘겨 `Open` 이 아무것도 만들지 않고
거절하는지 본다. 시험이 root 로 돌면 `permVerdict` 표만 본다 — 갈래만 바뀌고 스킵하지 않는다 (CI 의 스킵 감시).

---

## 5. 단계 — 열여덟

### Step 1 — 기준선

- [x] `unit/lower-state` 가 `a3b1536` 위에 있고 작업 트리가 깨끗하다 (계획 · 상태 · 감사 셋 밖)
- [x] 시험 DB 를 띄운다 (`scripts/testdb.sh` · `docs/testdb-setup.md`). CI 의 측정 명령과 awk 로 `internal/enode` · `cmd/enode` ·
      `internal/environment` · `internal/panel` · `internal/store` · `internal/api` · `internal/contract` 커버리지와 통과 수를 적는다
      — 통과 2,178 · 실패 0 · 스킵 0 · `internal/enode` 83.6% · `cmd/enode` 80.4% · `internal/environment` 82.2% · `internal/panel` 89.1% ·
      `internal/store` 83.1% · `internal/api` 82.5% · `internal/contract` 92.5% · 전체 86.4% · 라우트 19
- [x] `golangci-lint run ./...` 경고 수를 적는다 (기준 38) — 38 건 (errcheck 24 · staticcheck 10 · govet 2 · ineffassign 1 · unused 1)
- [x] SunnyVM 이 닿는지 본다 — Step 16 의 integration 시험 자리 (2026-09-27T02:09:01Z 닿는다 · 커널 7.0)

### Step 2 — contract 와 store 의 어휘 (FD 엔티티 8절 · 답 1 · FD 규칙 11절)

- [x] `result.go` — `ReasonLowerChanged = "lower_changed"` (원인 코드 목록의 끝)
- [x] `bake.go` — `ValidBuildName` · `IRProblem` (4절 ⑲)
- [x] `store/claim.go:962` — reason 목록에 `contract.ReasonLowerChanged`
- [x] 시험 — 오늘의 굽기 계약 시험이 그대로 초록 · 내보낸 둘의 표 몇 줄 · `TestStepResult_OutOfVocabulary` 옆에 원인 코드 다섯이 모두
      어휘 안인지 (`TestValidBuildNameAndIRProblem` · `TestStepResult_ReasonsAreInVocabulary`)

### Step 3 — 실행 중 획득이 drain 을 본다 (답 2 · FD 규칙 6절)

- [x] `acquire.go` `tryGrab` — `busyIn` 뒤에 `drainingIn(ctx, tx)` 를 busy 에 합친다 (`queue.go:302` 와 같은 모양)
- [x] `internal/api` 시험 (시험 DB) — graceful drain 을 광고한 노드만 있으면 acquire 가 unavailable 갈래로 · at-boundary 도 같다 ·
      drain 없는 노드는 그대로 잡힌다 (`TestAcquire_ADrainingNodeIsNotGrabbed` · 합치는 줄을 빼면 두 갈래가 빨갛다)
- [x] 오늘의 획득 시험 (`TestAcquire_*` · `TestWidthCap_*`) 이 그대로 초록

### Step 4 — `internal/lower` 겉면과 판정 (FD 엔티티 1 ~ 7절 · 모든 플랫폼)

- [x] `lower.go` — 패키지 문서 (담는 것 · 모르는 것 — environment · contract · merge · scratch · 광고 · Mediator / 봉인 / Mediator 가
      링크하지 않는다) · FD 엔티티의 타입과 JSON 칸 · `Metadata` 와 그 칸 (`component-methods.md` 1.4) · 오류 문구 (FD 규칙 13절)
- [x] `judge.go` — `Val[0]` · `Val[1]` (int32) 과 ino 에서 `Key` · `Key.String` · `ParseKey` · `identityVerdict` · `permVerdict` (3.1) ·
      `parseMountinfo` (`\040` 풀기 · 선택 칸 · 구분자 `-`) · `overlaysOn` (lowerdir 의 `:` · `lowerdir+=` · `datadir+=` · 경로가 L 이거나
      안이거나 L 을 담는다 · 같은 장치) · 점검 판정 (st_dev 와 마운트 · 소유 uid · 신원) 과 문구 (FD 규칙 12.3) · metadata 거르기 (schema 1)
- [x] `lower_other.go` — 같은 겉면이 `ErrUnsupported` (`lower is supported on linux only`)
- [x] `judge_test.go` — 키 표 (`stat -f -c %i` 순서 · 음수 int32 · `ParseKey` 거절) · 신원 표 (다섯 판정 · btime 0 은 대조 안 함 · 자리
      이름과 다른 fsid · ino) · 권한 표 · mountinfo 표 (FD 흐름 9.1 — helper 모양 두 줄은 찾는다 · /work 모양 bind 별칭은 안 찾는다 ·
      lowerdir 이 lower 의 부모 · `\040` · `lowerdir+=` · 다른 장치) · 점검 판정 표

### Step 5 — 뿌리 · 자리 · 권한 (FD 규칙 1 · 2 · 3절 · 3.1 · 4절 ① ② ⑬ ⑭)

- [x] `root_linux.go` — `ReadRoot`: `EvalSymlinks` · 디렉터리 · statfs · statx (`STATX_INO|STATX_BTIME|STATX_MNT_ID`) · 마운트 번호가 0 이면
      `/proc/self/mountinfo` 에서 경로가 가장 길게 겹치는 마운트 (답 6)
- [x] `perm_linux.go` — 디렉터리 열기 (O_DIRECTORY|O_NOFOLLOW|O_CLOEXEC · fstat · 주인 · 좁히기 여부) · 잠금 파일 열기 (4절 ③) · JSON
      읽기 (O_NOFOLLOW · 보통 파일 · 주인 · 1 MiB) · 쓰기 (CreateTemp · fsync 여부 · rename · 디렉터리 fsync)
- [x] `dir_linux.go` — `Open` (lowers · `<key>` · holders 0700 · lower.lock · bake.lock 0600 · lower.json 판정표대로 쓰기 · Reused 는 4절 ⑬) ·
      `Peek` (아무것도 안 만든다 · 자리가 없으면 nil, nil · 느슨한 비트는 알린다) · `ReadState` (없으면 committed · 깨졌으면 오류) ·
      `ReadMetadata` · `WriteMetadata` (lower 뿌리 · O_NOFOLLOW · 보통 파일 · 1 MiB · schema 1 · 옛 임시 파일 먼저)
- [x] `root_linux_test.go` — 임시 폴더의 btime 과 마운트 번호가 0 이 아니다 (ext4 · tmpfs 에서. 다른 filesystem 이면 0 을 받는 갈래) ·
      symlink 로 준 뿌리가 풀린다 · rmdir 뒤 mkdir 로 같은 inode 가 나오면 Reused 와 Foreign 을 실제로 본다 (안 나오면 표 시험만)
      — 이 기계에서 같은 inode 가 두 번 다 나왔다
- [x] `dir_linux_test.go` — 두 번 Open 이 paths 를 더한다 · Peek 이 아무것도 안 만든다 · 판정 다섯의 동작 (Foreign · Broken 은 오류) ·
      state 없음 · 깨진 JSON · metadata 없음 · 상한 · symlink · 옛 임시 파일
- [x] `perm_linux_test.go` — 3.1 의 시험 · 4절 ⑳

### Step 6 — 잠금 셋과 쥔 사람 기록 (FD 엔티티 4절 · FD 규칙 4 · 7절 · 4절 ⑱)

- [x] `lock_linux.go` — `TryShared` (lower.lock `LOCK_SH|LOCK_NB` 다음 기록 · node_id 가 `[0-9a-z-]` 밖이면 기록 없이 로그) · `Update` (바뀐
      칸이 있을 때만) · `Release` (기록 다음 lower.lock) · `WaitShared` (기록 없음 · `every` 마다 `notice(ReadState())`) · `Exclusive`
      (`pollEvery` 마다 `LOCK_EX|LOCK_NB` · `every` 마다 `watch(Waiting{Holders, Unnamed})` · 마감이면 `ctx.Err()`) · `TryBake` ·
      `Bake.WriteState` (fsync 둘) · `Holders` (`.json` 마다 짝 `.lock` 에 `LOCK_SH|LOCK_NB` · 막히면 산 기록 · 지우지 않는다 · 점으로
      시작하는 이름은 건너뛴다)
- [x] `lock_linux_test.go` — FD 흐름 9.1 의 잠금 줄 (공유 + 공유 · 공유 중 TryBake 는 된다 · 공유 중 Exclusive 는 기다리다 Release 뒤
      잡힌다 · Exclusive 의 마감 · TryBake 둘째는 false · WaitShared 가 배타 Release 뒤 잡힌다 · 기다리는 동안 notice) · 기록 줄 (산
      기록만 · Release 뒤 죽은 기록 · 기록이 살아 있으면 lower.lock 이 막혀 있다 · Unnamed · 이름이 규칙 밖인 node)

### Step 7 — 마운트 훑기와 점검 셋 (FD 엔티티 6 · 7절 · FD 규칙 8.2 · 12절 · 4절 ⑯)

- [x] `mounts_linux.go` — `ForeignMounts`: /proc/self/mountinfo 에서 lower 를 담은 마운트 (장치 · filesystem 안의 경로 L) · /proc 의 pid
      중 주인이 이 uid 인 것 · ns/mnt 링크마다 한 번 · 권한으로 못 읽으면 `Unreadable` · 사라졌거나 좀비는 건너뛴다 · `overlaysOn`
- [x] `check_linux.go` — `Check`: scratch 가 없으면 가장 가까운 있는 조상 · st_dev 와 마운트 · 루트의 소유 uid · `lowers` 가 있으면
      `Peek` 과 `identityVerdict` · 문구는 FD 규칙 12.3 그대로
- [x] `mounts_linux_test.go` — 이 프로세스로 `ForeignMounts` (Found 0 · Namespaces 1 이상) · 읽기 실패의 갈래는 함수 값으로 끼운다
      (Step 17 에서 커버리지가 모자랄 때만 · FD 흐름 10절) — 함수 값 대신 가짜 /proc 폴더를 넘겨 좀비 · 사라짐 · 권한 · 같은 namespace 를
      밟는다 (`scanMounts(proc, ...)`)
- [x] `check_linux_test.go` — scratch 가 워크스페이스와 같은 폴더 아래 · scratch 가 아직 없음 · 다른 filesystem (`/dev/shm` 이 있으면 tmpfs 로
      · 없으면 판정 표) · uid 가 다름 · identity 판정 연결 · `lowers` 가 `""`

### Step 8 — 동시성 시험 (3.2)

- [x] `concurrency_linux_test.go` — 3.2 표의 앞 다섯 줄. 자식 프로세스는 `TestMain` 이 환경 변수 (`ENODE_LOWER_TEST_CHILD`) 로 나눈다
      — 앞 넷이 여기다. 다섯째 (배타가 굶지 않는다) 는 LowerGuard 의 줄이라 Step 10 이 본다. SIGKILL 뒤 배타까지 1.8 ms (pollEvery 2 ms) ·
      깨진 읽기 0 / 103,724 · CLOEXEC 를 빼는 변이는 fd 시험이 잡는다
- [x] `BenchmarkHolders` · `BenchmarkForeignMounts` (기본 `go test` 에서는 안 돈다)

### Step 9 — `internal/enode` 의 겉면 (FD 엔티티 8절 · 4절 ⑥)

- [x] `runtime.go` — `RuntimeCapability{Writes}` · `StepRuntime` 에 `Capability()` · `NativeRuntime` 은 in-place · `WorkspaceWrites(rt)`
- [x] `runc_overlay_linux.go` · `runc_overlay_other.go` — `Capability()` 는 isolated
- [x] `policy.go` — `DrainBake` · `DrainLower` · 172행 주석 (bake · lower 가 더해졌다)
- [x] `paths.go` — `LowersDir()` (`$HOME/.local/state/enode/lowers` · `ENODE_STATEDIR` 를 안 본다 · home 을 못 찾으면 오류)
- [x] 시험 — 가짜 런타임 넷에 `Capability` · `LowersDir` (`ENODE_STATEDIR` 를 둬도 같다 · `HOME` 이 비면 오류) · `WorkspaceWrites` 표

### Step 10 — `LowerGuard` (FD 엔티티 9절 · FD 규칙 5 · 9 · 10절 · FD 흐름 2 · 3 · 4 · 6절 · 4절 ⑧ ⑪ ⑫)

- [x] `lowerguard.go` — `StartLowerGuard` · `BeforeAdvert` · `AfterResponse` · `OnClaim` · `StepDone` · `HoldBake` · `DropBake` ·
      `LowerChangedError` · 출처 문구 (FD 규칙 9절) · 로그 문구 (FD 규칙 13절 · 원인이나 phase 가 바뀔 때만) · 시각은 `now` 칸
- [x] `lowerguard_linux_test.go` — 임시 폴더의 진짜 자리로 (FD 흐름 9.1 의 LowerGuard 줄):
      후보면 쥔다 · pending 이면 bake 출처이고 공유를 새로 안 잡는다 · 배타가 쥐어져 있으면 bake 출처 (being merged) · 자리를 못 열면
      lower 출처이고 다음 광고에 다시 연다 · 응답 두 번 뒤 놓는다 · 한 번이면 안 놓는다 · 도는 단계가 있으면 안 놓는다 · 광고 실패는
      셈을 안 바꾼다 · 늦게 보인 임대 네 장면 (FD 흐름 4절 — 다시 쥔다 · merging 은 거절 · 표지가 바뀌면 거절 · 못 쥐면 거절) ·
      거절 표는 임대가 사라지면 지운다 · prepare 와 merge claim 에서 놓고 거절하지 않는다 · 임대가 0 이면 candidate (⑫) ·
      HoldBake 중에는 안 쥔다 (⑪) · 합치기 없이 끝난 굽기 (pending 이면 abandon 한 번 · building · merging 이면 안 부른다) ·
      nil 수신자 — 3.2 표 다섯째 줄 (배타가 굶지 않는다) 도 여기다 (`TestLowerGuard_PendingDrainsAndTheMergeGetsTheLock`)

### Step 11 — 광고 (FD 규칙 11절 · FD 흐름 2절 · 4절 ④ ⑤ ⑥ ⑭ ⑮)

- [x] `advertkeys.go` — 예약 키 · metadata 에서 키 (`contract.ValidBuildName` · `contract.IRProblem` · bake.run 상한 · bake.resumed 는
      bake.run 과 늘 함께) · 복사본에 싣기 (예약 키 빼기 · `hasCapability` 다시 · 모든 능력에)
- [x] `advertise.go` — `Guard` · `Writes` · 라벨 경고의 기억 · `drain()` 이 `Guard.BeforeAdvert(sources)` 의 출처를 더하고 키를 돌려준다 ·
      광고 본문은 복사본 · 응답이 오면 `Held` 뒤에 `Guard.AfterResponse(resp.Drain, resp.Leases)` · 실패하면 부르지 않는다 (FD 규칙 5.2)
- [x] `advertkeys_test.go` (모든 플랫폼 · 표) — 예약 키 · 어긋난 이름과 IR 은 그 키만 뺀다 · 깨진 metadata · bake.run 상한 · 복사본이라
      원본이 그대로 · 능력이 0 이면 안 싣는다
- [x] `drain_test.go` — 시험 서버의 광고 본문에 `workspace.writes` · bake · lower 출처가 합친 값과 상태 파일에 · 예약 라벨이 빠지고 로그가
      한 번 · `Guard` 가 nil 이면 오늘 그대로 · metadata 와 contract 의 JSON 모양 (4절 ⑭) — 진짜 잠금이 드는 줄 (bake 출처 ·
      metadata 키 · 실패한 광고는 셈을 안 바꾼다) 은 `lowerguard_linux_test.go` 의 `TestDrain_BakeSourceKeysAndFailedAdverts` 다

### Step 12 — claim (FD 흐름 3절 · FD 규칙 5.3 · 5.4 · 4절 ⑨ ⑩)

- [x] `claim.go` — `Worker.Guard` · 509행 임대 확인 뒤 `OnClaim` 과 `defer StepDone` · 오류면 보고하고 돌아간다 · 그 밖은 오늘 그대로
- [x] `lowerclaim_linux_test.go` — 시험 서버로: 거절 보고의 모양 (reason · error · exited 가 오지 않는다 · 워크스페이스를 준비하지 않고
      세션을 열지 않는다 — `trackingRuntime` 이 0 번) · 자리를 못 연 노드의 보고 (원인 코드 없음) · prepare claim 에서 놓는다 · 패닉에도
      StepDone · 도는 단계가 있는 동안 응답 두 번에도 안 놓는다

### Step 13 — 준비도 점검 (FD 규칙 8.3 · 12절 · FD 흐름 8절 · 4절 ⑦ ⑯ ⑰)

- [x] `environment/check.go` — `FactSource` · 286행 앞에서 verifier 가 `FactSource` 면 그 Fact 를 `add` 로 더한다 (State 순위가 그대로
      합친다 · 하나라도 ready 가 아니면 smoke 를 안 돈다 — 291행)
- [x] `lowercheck.go` — type 옮김 · `Facts` (runc-overlay 만 · Source `/runtime` · `Cause` binding 은 invalid · state 는 external-blocked) ·
      smoke 잠금 함수 · `LogNotice`
- [x] `runc_overlay_linux.go` `Verify` — 4절 ⑰ 의 자리에서 잠그고 닫은 뒤 놓는다
- [x] `check_test.go` — 가짜 `FactSource` 로 자리 (host 뒤 · prepared 앞) · State 순위 · 어긋나면 smoke 를 안 부른다 · native 는 Fact 없음 ·
      `FactSource` 가 아닌 verifier 는 오늘 그대로
- [x] `lowercheck_linux_test.go` — `Facts` 셋 (같은 마운트 · 다른 filesystem · uid · 신원) · home 오류 · smoke 잠금 (배타를 쥔 `Dir` 이 놓으면
      풀린다 · Notice 가 처음 한 줄 · 자리가 없으면 안 잠근다 · lower.lock 이 없으면 안 잠근다) · `LogNotice` 가 줄마다 한 번

### Step 14 — 기동 (FD 흐름 7절 · 4절 ⑧)

- [x] `cmd/enode/main.go` — 4절 ⑧ 의 표 그대로
- [x] `cmd/enode/environment.go` — `enode.ExecutionRuntimeVerifier{Notice: os.Stderr}`
- [x] 오늘의 `cmd/enode` 시험이 그대로 초록 · 커버리지가 Step 1 과 같은 자리에 있다 (80% 이상) — 시험 하나를 더했다
      (`TestEnvironmentCheckReportsTheLowerFacts` · env check 가 셋을 낸다). 수치는 Step 17

### Step 15 — 제어판과 경계 시험 (FD 규칙 9 · 14절)

- [x] `page.go` — `drainName` 에 bake 「굽기」 · lower 「아래층 상태」, `drainLift` 에 「저절로 풀린다 — 합치기가 끝나면」 · 「상태 자리를
      고치면 다음 광고에서 풀린다」
- [x] `panel_test.go` — `TestIndexDrawsDrainSourcesAndTrash` 에 네 문구
- [x] `boundary_test.go` — 금지 `{"cmd/mediator", "internal/lower"}` · 봉인 `{"internal/lower", []string{"golang.org/x/sys/"}}` · 봉인 주석에 한 문단
      — `internal/lower` 가 `internal/scratch` 를 임포트하는 변이를 봉인이 잡는다

### Step 16 — integration 시험과 측정 (FD 흐름 9.2 · 3.2)

- [x] `internal/lower/lower_integration_test.go` (`//go:build integration && linux`) — 시험 바이너리를 `unshare --user --map-root-user --mount`
      뒤에서 다시 띄운 자식이 helper 모양 (bind + overlay) 으로 임시 lower 를 마운트한다 -> `ForeignMounts` 가 찾는다 · 자식이 끝나면 0 ·
      자식 namespace 안에서 워크스페이스의 bind 별칭 -> `binding.scratch_filesystem` 판정이 같은 filesystem · 다른 마운트
- [x] `internal/enode/runc_overlay_integration_test.go` 에 한 줄 — 다른 `Dir` 이 배타를 쥔 동안 `Verify` 가 기다리다 놓으면 돈다
      — `TestRuncOverlaySmokeWaitsForAMerge`. `Verify` 는 `os.Executable()` 을 helper 로 쓰므로 이 파일에 `TestMain` 을 두어 시험
      바이너리가 `runtime-helper` 입구를 안다
- [x] `go vet -tags integration ./internal/lower/ ./internal/enode/` 로 컴파일을 본다
- [x] 이 기계에서 돌린다. SunnyVM 이 켜져 있으면 에이전트가 거기서도 돌리고 결과를 적는다 — 버려도 되는 폴더에서만 · `/srv/yocto` 는 읽지도
      않는다 · 둔 파일은 지운다. 꺼져 있으면 보류로 적는다 — 병합 조건이 아니다 — 이 기계: lower 둘 초록 (runc 장면은 이 기계가
      `--map-auto` 를 못 해 못 돈다). SunnyVM: 셋 다 초록 (버려도 되는 `~/lower-it-*` · busybox 로 지은 rootfs · 끝나고 지웠다)
- [x] `BenchmarkBeforeAdvert` (`internal/enode`) · `BenchmarkHolders` · `BenchmarkForeignMounts` 를 한 번 돌려 숫자를 적는다 (3.2)
      — 이 기계 25 µs · 0.12 ms · 5.2 ms / SunnyVM 9 µs · 0.037 ms · 3.6 ms (셋 모두 추정의 한 자리 안)

### Step 17 — 코드 검사

- [x] `gofmt -l .` · `go vet ./...` · `go vet -tags integration ./internal/lower/ ./internal/enode/` · `go build ./...` — 빈 출력 · exit 0 넷
- [x] `go test ./... -count=1` (CI 의 명령 · 시험 DB · 가짜 claude 스텁) — 실패 0 · 스킵 0 · 패키지마다 80% 이상 (`internal/lower` 포함 ·
      `internal/enode` · `cmd/enode` 는 Step 1 과 댄다). 모자라면 `_linux.go` 의 호출 자리를 함수 값으로 떼어 실패를 끼운다 (FD 흐름 10절)
      — 통과 2,271 · 실패 0 · 스킵 0 (스킵 감시 24 패키지) · 스물세 패키지 전부 80% 이상 · 전체 87.1% · `internal/lower` 93.2% ·
      `internal/enode` 83.6% -> 84.5% · `cmd/enode` 80.4% -> 80.5%. 함수 값으로 떼지 않았다
- [x] `go test -race -count=1 ./internal/lower/ ./internal/enode/` — 경합 0 (측정 9 의 기준) — 통과 607 · 경합 0
- [x] 크로스 빌드 셋 (windows/amd64 · linux/arm GOARM=7 · darwin/arm64 — linux/arm 은 유닛 정의 10절의 확인) · `enodectl.exe` 심볼 상한 ·
      U+2605 0 · `go run ./scripts/glyphscan.go` · 린트 수가 Step 1 보다 늘지 않는다 — 셋 exit 0 · 심볼 1 · 6 · U+2605 0 · glyphscan
      160 파일 0 · 린트 38 (Step 1 과 같은 목록. 처음에 110 이었다 — 새 코드가 버린 오류 71 을 `_ =` 로 드러내고 staticcheck 하나를 고쳤다)
- [x] **조각 0** — build · vet · test · 라우트 수가 Step 1 과 같다 — 초록 · 라우트 19
- [x] `git status` 로 2절 밖의 파일이 없는지 · 시험이 바꾼 `cmd/enodectl/probe.lock` 은 되돌린다 — 2절 밖은 Step 18 이 고치는
      `component-methods.md` 하나 · probe.lock 되돌림

### Step 18 — 요약 · 상태 · 감사 · 커밋

- [x] `construction/lower-state/code/code-summary.md` — 파일 · 규칙의 자리 · 4절의 결정 · 계획과 다른 자리 · 코드 검사 숫자 · 3절의 측정과
      시험 결과 · integration 결과 · 행렬 밖 파일의 diff · 넘기는 것 (FD 흐름 12절 + 4절 ⑪ 의 HoldBake 차례) · 정본 되돌림 (FD 흐름 13절)
- [x] 코드의 겉면이 `component-methods.md` 와 달라진 자리가 있으면 그 줄을 고친다 (예: `StartLowerGuard` · verifier 의 `Notice` · `Check` 의
      `lowers` 가 `""`) — 1.1 · 1.3 · 1.5 · 4.1 · 4.3 · 4.4 에 줄을 더했다
- [x] 표기 검사 · 사용자가 싫어한 말투 · 사내 이름 · 새 축약어
- [x] 이 계획의 체크박스 · `aidlc-state.md` 의 U6 블록 · `audit.md`
- [x] 한 커밋 — 코드 · 시험 · 이 계획 · code-summary · 고친 회차 문서 · 상태 · 감사. 승인 뒤에 넣는다 (CONVENTIONS 3.3)

---

## 6. 이 단계가 하지 않는 것

```text
   build · merge 단계 · 기동 때 정리와 재개 · 배타 대기의 로그 문장과 간격            bake 유닛
   WriteMetadata 를 부르는 때 · abandon 의 몸통 · HoldBake 를 부르는 자리              bake 유닛.  이 유닛은 함수와 부르는 틀만
   merge Preflight 의 마운트 확인 (답 6 · internal/merge/merge_linux.go:81 · :86 옆)   bake 유닛
   「한 lower 의 노드를 모두 새 판으로 올린 뒤 굽는다」의 확인                          bake 유닛의 조각 스크립트 · 정본 (진행자)
   RuntimeCapability 의 Capture                                                   checkpoint 유닛
   제어판에 ir · bake 키를 보이기                                                  하지 않는다 (4절 ④)
   Mediator 의 다른 매칭 경로                                                      제출 · 승격은 이미 drain 을 본다 (FD 계획 2.7)
   조각 8                                                                        bake 유닛 · 사람 · SunnyVM
   정본(enode-design) 되돌림                                                       진행자가 올린다 (FD 흐름 13절)
   PR 과 병합                                                                     코드 검사가 초록인 뒤 · CI 뒤.  올리기 전에 묻는다
```

---

## 7. 확장 준수 — Code Generation

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 상태 자리 줄은 3.1 이 규칙과 시험으로 받는다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 한 줄 · 동시성은 3.2 의 시험 |

---

## 8. 물음

없다. NFR 의 보안은 요구 5.3 의 문장 (노드 사용자 전용 권한이다) 과 FD 규칙 1절 (데몬만 자리를 만든다) 에서 규칙이 나오고, 측정
2 · 3 이 거절할 것 (symlink · 남의 자리) 을 보였다. 동시성은 설계가 FD 에서 닫혔고 여기서는 시험과 측정 자리다. 4절의 스물은 코드에
대 보다 새로 보인 것이고, 정본과 FD 의 결정을 바꾸는 것이 없다. 반대하는 자리가 있으면 승인 대신 그 번호를 알려 주세요.
