# `bake` — Code Generation 요약

**유닛** `bake` (굽기 단계 · 이 회차 유닛 한 줄 순서의 일곱째) · **브랜치** `unit/bake` · **기준** `fb86ad2` (Functional Design 커밋) ·
**계획** `construction/plans/bake-code-generation-plan.md` (스물두 단계) · **맡는 조각** 5 (기계) · 6 · 7 · 8 (사람 · SunnyVM) —
넷이 병합 게이트다

노드가 굽기 계약의 두 단계를 받는다. build 단계는 굽기 잠금 · building · 세션 안의 bash 확인 · sync · IR 대조 · pinned manifest ·
builds 를 차례로 돌리고, 성공하면 upper 를 대기 자리로 옮긴 뒤 초안과 pending 을 쓴다. merge 단계는 claim 확인 · 형제를 기다리는
배타 (대기 로그) · 그물 · merge-helper 의 시작 전 확인 · merging · merge-helper 의 합치기 · metadata · committed 다. 주인이 죽은 낡은
상태와 끊긴 합치기는 기동 · 광고 주기 · build claim 이 한 표로 치우거나 잇는다. 격리 노드의 Prepare 가 호스트에서 reset · clean 을
안 돌리고, 격리 노드의 Finalize 가 workspace.diff 를 세션 안에서 만든다 (굽기 뒤 lower 에 남는 계약의 파일을 호스트가 실행하지 않게).

아래에서 쓰는 줄임은 계획 머리와 같다 — FD 는 Functional Design (FD 규칙 N절 = `business-rules.md` · FD 흐름 N절 =
`business-logic-model.md` · FD 엔티티 N절 = `domain-entities.md`), CG 는 Code Generation (CG 물음 N 답 = 계획 10절의 물음), 4절 N번 =
계획 4.1 표의 번호, 결정 N = audit 의 U7 항목들이 적은 설계 결정, 받는 일 N = FD 계획 1절 표의 번호.

---

## 1. 파일

| 파일 | 새 · 고침 | 무엇 |
|---|---|---|
| `internal/enode/bake.go` | 새 | `Baker` · `StartBaker` · `Wait` · 기동 정리 `startUp` · 정리의 표 `clean` · `sweepOwn` · `openResume` · `heldBake` 와 몸통 `abandon` · `letGo` · `draftBuilds` · `startMerging` · `cancelMerging` · 시험이 바꿔 끼우는 `writeLowerState` |
| `internal/enode/bake_build.go` | 새 | `runBuildStep` 과 `buildRun` (`command` · `nodeCommand` · `interrupted` · `stopEarly` · `manifest` · `exited` · `failEnd` · `succeed`) · `takeForBuild` · `previousIR` · 단계 로그 `bakeLog` · `uploadStepLog` |
| `internal/enode/bake_merge.go` | 새 | `runMergeStep` 과 `mergeNode` (`exclusive` · `gaveUp` · `fail` · `stopped` · `finish` · `merged`) · `foundLines` · 시험이 바꿔 끼우는 `foreignMounts` · `writeMetadata` · `movePending` |
| `internal/enode/bake_resume.go` | 새 | `onStale` · 배경 일 `stale` · `resume` · `resumeExclusive` · `completeMerge` · `noteErr` · `retryLater` |
| `internal/enode/bakerule.go` | 새 | 순수 함수 — 셸 한 줄 셋 (`bashCheck` · `pinCommand` · `probeScript`) · 문장 · IR 대조 (`parseProbe` · `irVerdict` · `tagList`) · `repoIDOf` · `stripPassword` · `Draft` · `metadataOf` · `buildManifestOf` · `mergeOpsOf` · `attemptReason` · `abandonedReason` · `mergeClaim` · `finished` · `pendingShape` · `waitLog` · merge-helper 의 주고받는 타입과 `helperErrorOf` · `helperResponseOf` |
| `internal/enode/bakefile.go` | 새 | `makePending` · `discardPending` · `sweepPending` · `writeDraft` · `readDraft` · `readPinned` · 둘이 함께 쓰는 `openPlain` (시험이 바꿔 끼우는 `afterLstat`) |
| `internal/enode/mergehelper_linux.go` | 새 | `RunMergeHelper` · `mergeHelperArgv` · `mergeHelperCommand` · `callMergeHelper` · `fileOwner` |
| `internal/enode/mergehelper_other.go` | 새 | linux 밖 — `RunMergeHelper` 는 exit 1 과 `merge-helper is supported on linux only` · `callMergeHelper` 는 같은 오류 · `fileOwner` 는 모른다 |
| `internal/enode/claim.go` | 고침 | `Step` 의 칸 넷 (`Sync` · `Builds` · `IR` · `Merge`) · `Result` 의 칸 둘 (`Build` · `Merge`) · `Worker.Bake` · `execute` 의 kind 분기 · `afterExit` 를 나눈 `closeOut` 과 `closing` |
| `internal/enode/runc_overlay_linux.go` | 고침 | `Close(Keep{Upper})` 의 `RENAME_NOREPLACE` · `Open` 이 helper 응답을 못 받은 길 (보내기 · 받기) 에서 abort 뒤 stderr 꼬리 · `Finalize` 가 diff 를 세션 안에서 (`diffInSession`) · helper 의 finalize 는 Diff 를 끈다 |
| `cmd/enode/main.go` | 고침 | 부르는 줄 넷 — `merge-helper` 입구 · `StartBaker` · `Worker` 의 `Bake` 칸 · `baker.Wait()` |
| `scripts/finalize-bake/` | 새 | `bake-common.sh` · `slice-5.sh` · `slice-6.sh` · `slice-7.sh` · `slice-8.sh` |
| 행렬 밖 열하나 | 고침 | 9절 — `result.go` · `store/claim.go` · merge 셋 · `finalize.go` · `lowerguard.go` · `runtime.go` · `lower.go` · `diff.go` · `workspace.go` |
| 시험 (새) | 새 | `internal/enode` — `bakerule_test.go` · `bake_test.go` (모든 플랫폼) · `bakerule_linux_test.go` · `bake_fake_linux_test.go` (공용 틀) · `bake_linux_test.go` · `bake_build_linux_test.go` · `bake_merge_linux_test.go` · `bake_resume_linux_test.go` · `bakefile_linux_test.go` · `mergehelper_linux_test.go` · `workspace_linux_test.go` · `bake_integration_test.go` (`integration` 태그) / `internal/api/bake_test.go` (시험 DB) |
| 시험 (고침) | 고침 | `internal/enode` 의 `lowerguard_linux_test.go` · `runc_overlay_linux_test.go` · `runc_overlay_integration_test.go` · `finalize_test.go` / `internal/merge` 의 `decide_test.go` · `merge_linux_test.go` · `merge_integration_test.go` / `internal/contract/result_test.go` / `internal/store/exited_test.go` / `internal/api/api_test.go` / `cmd/enode/main_test.go` |

**계획 2절 밖의 파일** — 코드는 없다. 시험 파일 하나 (`internal/enode/finalize_test.go` · `TestSettle` 에 sealErr 두 줄 · Step 11 이 적었다) 와 Step 22 가
고친 회차 문서 `component-methods.md` 다. 시험이 바꾼 `cmd/enodectl/probe.lock` 은 되돌렸다.

**확인만 하고 안 고친 것** — `runc_overlay_other.go` (linux 밖에는 세션이 없고 native 의 `Close` 는 once 만 돈다) · `upload.go` · `env.go` ·
`repoid.go` · `runtime.go` 의 `finalizeLocal` · `trash_linux.go` (기동 청소는 `enode-runc-` 로 시작하는 이름만 · 대기 자리 `pending/` 을 안
건드린다) · `advertise.go` · `internal/lower/*` (주석 밖) · `internal/scratch/trash.go` · `internal/store/verdict.go` · `internal/contract/bake.go` ·
`examples/bake.json` · `internal/panel/boundary_test.go`. 경계 시험의 금지는 Mediator 쪽 한 줄뿐이라 `internal/enode` 가 `internal/merge` 를
처음 가져와도 고칠 줄이 없다. `go.mod` 는 그대로다.

---

## 2. 규칙의 자리 (FD 규칙 절 -> 함수)

```text
   1   단계 분기와 거절                 execute 의 kind 분기 (claim.go) · runBuildStep · runMergeStep 첫 줄의 WorkspaceWrites ·
                                       takeForBuild · refusedText · bakeInProgressText
   2   전이와 쓰는 때                   takeForBuild (building · pending_upper 를 building 때) · succeed (pending) ·
                                       startMerging 뒤의 merging · abandon · finish · completeMerge (committed) · writeLowerState
   3   build 의 차례                    runBuildStep · buildRun.command (bash -c) · nodeCommand (bash 확인 · 대조 · pinned) ·
                                       closeOut 의 closing (Keep{Upper} · after) · succeed
   3.1 초안의 칸                        Draft · writeDraft · readDraft · stripPassword · repoIDOf · previousIR
   4   IR 대조                          probeScript · parseProbe · irProbe · irVerdict · tagList · irOutcome
   5   exited 와 예산                   buildRun.exited · closeOut (FinalizedAt 은 after 뒤) · settleIn.sealErr · uploadStepLog
   6   합칠 upper 를 못 남기면           failEnd · stopEarly · attemptReason · heldBake.abandon
   7   merge 의 차례                    runMergeStep · mergeNode.finish · merged · metadataOf · mergeOpsOf · movePending
   7.1 claim 확인                       mergeClaim (표를 위에서부터) · claimVerdict
   8   기다림                           mergeNode.exclusive · waitLog.lines · waitShape · leftText · clockText · tookLine ·
                                       foundLines · foreignRetry · gaveUp
   9   merge-helper 와 trash            RunMergeHelper · callMergeHelper · mergeHelperArgv · helperErrorOf · helperResponseOf
   10  시작 전 확인                     merge.checkMounts · merge resolve 의 STATX_MNT_ID · helperPreflightError ·
                                       runMergeStep (어긋나면 upper 를 trash) 과 completeMerge (어긋나도 upper 를 남긴다)
   11  몸통 · HoldBake · DropBake        heldBake.abandon · letGo · startMerging · cancelMerging · draftBuilds
   12  기동                             StartBaker · startUp · clean · sweepOwn · openResume · resume · resumeExclusive ·
                                       completeMerge · finished · pendingShape · metadataOf (resumed)
   13  살아 있는 동안                    LowerGuard.OnStale · Baker.onStale · stale · retryLater · noteErr
   14  last_attempt                    attemptReason · abandonedReason · draftBuildsOf
   15  결과 칸과 $OUT                   buildManifestOf · BuildManifest.HeadTags · succeed 의 올리는 폴더 (manifest 하나) ·
                                       mergeNode.merged (merged 하나)
   16  문구                             bakerule.go 의 문장 함수와 상수 (영어 그대로)
```

---

## 3. 계획 4.1 의 결정이 어떻게 들어갔나

- **1번 파일 배치** — FD 엔티티의 새 파일 다섯과 merge-helper 둘 그대로 · 초안 파일 일만 `bakefile.go` (모든 플랫폼)
- **2번 분기와 겉면** — `runBuildStep(runCtx, ctx, step, dir, in, out, log)` · `runMergeStep(runCtx, ctx, step, log)`. 두 함수의 첫 줄이
  isolated 가 아니면 거절한다 (Bake 가 nil · Runtime nil · `NativeRuntime{}` 세 줄의 시험)
- **3번 `closeOut`** — `closing{keep, after, upload}`. `closeOut` 은 단계 로그를 업로드할 때 읽는 `logBody func() []byte` 를 더 받는다 — after 가
  쓴 pending 줄이 단계 로그에 든다 (5절)
- **4번 `settleIn.sealErr`** — finalize 칸 error 와 그 문장. 행렬 밖 기록에 따로 (9절)
- **5번 올리는 폴더** — build 는 `enode-bake-out-` 임시 폴더에 manifest 하나 · merge 는 merged 하나
- **6번 `Close(Keep{Upper})`** — helper 의 `cmd.Wait` 뒤 · release 앞에 `Renameat2(RENAME_NOREPLACE)`. 오류는 `keep the upper: <원인>` ·
  aborted 면 `keep the upper: the session was aborted and its upper went to trash`
- **7번 초안 읽기** — Lstat · SameFile · 주인 uid · 1 MiB · schema 1. `syscall.O_NOFOLLOW` 를 안 쓴다 (windows 에 없다)
- **8번 대조 한 줄** — 계획의 글자 그대로 `probeScript`. IR 은 `ENODE_IR` 로만
- **9번 노드 명령의 출력** — 대조의 stdout 64 KiB · stderr 는 마지막 줄 1 KiB 만 문장에
- **10번 시간 글자** — `3m12s` · `(3h52m left)` · 마감은 UTC 초와 ` node clock`
- **11번 상수와 변수** — `mergeWatchEvery` 10초 · `waitLogEvery` 5분 · `foreignRetry` 60초 · `staleRetry` 10분 · `mergeHelperCommand`
- **12번 잠금의 차례** — `heldBake.mu` -> `Baker.mu` · `LowerGuard.mu` -> `Baker.mu` (onStale). 반대 차례는 없다
- **13번 몸통의 모양** — `abandon(reason, builds)` 은 한 번만 돈다 (`TestHeldBake_OneBodyUnderRace` · 고루틴 서른둘 · `-race -count=10`)
- **14번 claim 시각** — `runMergeStep` 의 첫 줄에서 잡는다 (거절 확인보다 먼저)
- **15번 merge 의 업로드 예산** — 단계 로그와 merged 에 업로드 예산 (기본 3분) · 넘으면 FAILED `upload_timeout` (CG 물음 1 답 A)
- **16번 merge-helper 의 주고받기** — 한 줄씩 · 응답을 `helperErrorOf` 로 오류에 · 응답이 없으면 `cannot run the merge helper:` 와 stderr 꼬리 (Wait 뒤)
- **17번 마운트 번호** — `resolve` 가 statx `STATX_MNT_ID` · 판정 `checkMounts` 는 `checkDevices` 다음
- **18번 OnStale** — guard 가 g.mu 아래에서 곧바로 부른다 · `Baker.onStale` 은 판단만 하고 `b.wg.Go` 로 연다
- **19번 정리의 몸통 하나** — `clean(dir, lock, st, from)` (계획의 세 인자에 `*lower.Dir` 하나를 더했다 · 키를 읽는다)
- **20 · 21 · 22번** — 정리의 builds 는 초안에서 · url 비밀번호 · 대기 로그의 모습 글자
- **23번 시험 틀** — `bake_fake_linux_test.go` (가짜 런타임 · 진짜 lower · namespace 없는 merge-helper · 가짜 Mediator)
- **24 · 25번** — 조각 5 의 왕복은 `internal/enode` 시험이 `internal/store` 를 가져와 · 제출 경로의 400 셋은 `TestSubmitRejectsBadContract`
- **26번 조각 스크립트** — 8절
- **27번 FD 16절 밖의 오류 글자** — `cannot create the pending upper directory:` · `cannot create the trash of the pending upper:` ·
  `cannot run the merge helper:` · `keep the upper:` · `cannot pin the manifest:` · `cannot write the bake draft:` · `cannot read the bake draft:`
- **28번 Result 칸** — DONE 의 exit_code 는 마지막으로 돈 계약 명령의 값. 명령 앞의 거절과 오류 · 끊긴 명령은 exit_code 를 싣지 않는다
- **29번 주석 둘** — `runtime.go` 의 `Keep` · `FinalizeSpec.Diff` · `lower.go` 의 `PendingUpper`
- **30번 pinned 읽기** — `readPinned` (보통 파일 · 16 MiB · SameFile · LimitReader)
- **31번 기준선 흔들림의 원인** — 시험 helper 의 eof 갈래를 `os.Exit(1)` 로 · 가짜 helper 의 모든 갈래를 `defer os.Exit(0)` 로 · `Open` 이
  응답을 못 받은 길은 abort 뒤 stderr. 셋째 원인 (보내기 실패 · 5절) 도 같은 길로
- **32번 권한** — `pending/` 과 `<키>` 는 0700 · `<이름>` 은 MkdirTemp · 초안 0600
- **33번 격리 노드의 Prepare** — isolated 면 저장소 확인만 · `PrepClean` · 노드 로그 how `fresh upper`. native 는 `reset and clean`
- **34번 격리 노드의 Finalize diff** — `sessionDiffScript` 를 세션의 Run 으로 · 출력은 stdout 으로만 · 호스트가 `$OUT/workspace.diff` 를
  임시 파일과 rename 으로 놓는다 · helper 와 호스트 둘 다 요청의 Diff 를 끈다

---

## 4. 계획의 단계 결과

단계마다 결과 한 줄을 계획의 체크박스 옆에 적었다. 요약만 옮긴다.

```text
   Step 2   ReasonIRMismatch · BuildManifest.HeadTags (omitempty 없음) · 어휘 한 줄                     시험 셋 초록
   Step 3   CheckMount · checkMounts · resolve 의 마운트 번호                                             TestCheckMounts · TestPreflight 줄
   Step 4   Step 의 칸 넷 · contractStep · Result 의 칸 둘                                                 시험 셋
   Step 5   bakerule.go                                                                                 표 시험 열일곱
   Step 6   bakefile.go                                                                                 시험 넷 (Step 20 이 셋을 더했다)
   Step 7   Close(Keep{Upper}) · 기준선 흔들림 · 세션 안 diff                                              시험 아홉 · 300 번에 빨강 0
   Step 8   OnStale · Dir                                                                               시험 둘
   Step 9   Baker · 몸통 · 정리                                                                          시험 넷 · -race -count=10
   Step 10  merge-helper                                                                                시험 넷 · 답 없음 300 번에 빨강 0
   Step 11  build 단계                                                                                  실패 표 열아홉 줄 포함
   Step 12  merge 단계                                                                                  claim 확인 여덟 · 실패 열둘 포함
   Step 13  재개 · 광고 주기의 정리                                                                       12.4 의 넷 포함 (Step 20 이 둘을 더했다)
   Step 14  기동의 네 줄                                                                                 TestMergeHelperEntrySkipsTheConfigSearch
   Step 15  Mediator 판정 시험 셋 · 제출 경로의 400 셋                                                     시험 DB
   Step 16  격리 노드의 Prepare · 보안 시험                                                               source 훑기 · PATH 표지
   Step 17  측정                                                                                        6절
   Step 18  integration                                                                                 7절
   Step 19  조각 스크립트                                                                                8절
```

---

## 5. 계획과 다른 자리

정본과 FD 의 결정 · 실패 등급 · 보안 경계 · 사용자가 답한 다섯 물음의 결과를 바꾼 자리는 없다. 아래는 이름 · 인자 · 글자 · 시험 방법이다.

- **대조의 exit 125 의 글자** — FD 규칙 4절은 대조를 못 하면 「git exited <n>」 이라 적었다. 125 는 셸이 `.repo/manifests` 에 못 들어간 것이라
  git 이 낸 것이 아니다 — `cannot verify ir: cannot enter .repo/manifests` 로 적는다 (Step 5)
- **`pendingShape` 는 scratch 가 `/` 인 경로도 모양 밖으로 본다** — 네 단계 위가 뿌리면 trash 가 `/trash` 가 된다 (Step 5)
- **이름 목록 밖의 작은 도우미** — `helperResponseOf` · `joinCause` · `lastLine` · `leftText` · `clockText` · `tookLine` · `waitShape` ·
  `contractRecord` · `mergedLine` · `schemeShaped` · `startedLine` · `exitedLine` · `skipLine` · `buildLabel` · `ownerRun` 등 (Step 5)
- **`discardPending` 은 없는 자리를 Lstat 으로 먼저 본다** — 없으면 trash 를 만들지 않고 옮겨진 것으로 본다. 기동 청소의 `sweepPending` 과
  두 읽기가 함께 쓰는 `openPlain` 을 더했다 (Step 6)
- **Step 20 이 `openPlain` 에 시험 틈 하나를 더했다** — `afterLstat` 함수 값 (제품은 아무것도 안 한다). Lstat 과 Open 사이에 symlink 로
  바꿔 끼우거나 파일을 키우는 시험이 그 틈을 쓴다 (계획 Step 20 의 「모자라면 호출 자리를 함수 값으로 떼어 실패를 끼운다」)
- **`Open` 의 셋째 원인** — 계획의 둘을 고친 뒤 300 번에 빨강 1 이 남았다. `send runtime helper request: write |1: broken pipe` — 먼저 죽은
  helper 에 open 요청을 쓰는 길이다. 보내기 실패도 abort 뒤 stderr 를 붙인다 (같은 뿌리 · Step 7)
- **세션 안 diff 의 바이트를 짓는 몫은 `sessionDiff` (`diff.go` · 모든 플랫폼)** — 세션과 시험이 같이 쓴다. 호스트도 helper 에 보내는
  요청의 Diff 를 끈다 (helper 도 끈다 — 둘 다). 요약 (stat) 은 상한을 `maxBlobBytes` 로 둔다 — 호스트처럼 잘리지 않게 (Step 7)
- **`OnStale` · `Dir` 은 nil 수신자에 안전하다** (Step 8)
- **`clean` 은 `*lower.Dir` 을 첫 인자로 더 받는다** · 몸통 밖으로 놓는 길은 `letGo` 하나 · 재개를 배경에 여는 `openResume` · state.json 쓰기는
  root 로 도는 시험이 바꿔 끼우는 `writeLowerState` (Step 9)
- **`callMergeHelper(req) (merge.Result, error)`** — 첫 줄 뒤의 stdout 은 버리고 Wait 한다 (Step 10)
- **pinned 를 못 읽은 실패의 exit_code 는 마지막 build 의 것이다** — 4절 30번은 「sync 의 것」 이라 적었으나 그것은 builds 앞의
  `repo manifest -r` 실패 줄과 겹쳐 적힌 것이다. sha256 은 builds 뒤에 읽으므로 28번의 원칙 (마지막으로 돈 계약 명령) 대로 둔다 (Step 11)
- **merge 단계가 시험 틈 셋을 더 둔다** — `foreignMounts` · `writeMetadata` · `movePending` 함수 값. Apply 오류는 lower 를 0500 으로 막는 대신
  가짜 helper 의 `apply-fails` 모드로 끼운다 — root 로 돌아도 같은 갈래 (Step 12)
- **몸통이 먼저 돌아 `startMerging` 이 false 면 보고하지 않는다** — FD 흐름 4절 첫째 줄의 「보고는 닿지 않는다」 · 노드 로그 한 줄 (Step 12)
- **FD 16.3 에 없던 노드 로그 한 줄** — 광고 주기 정리의 committed 쓰기 실패 `bake: cannot write the lower state while cleaning a stale bake;
  trying again in 10m` (Step 13)
- **merge-helper 입구의 시험은 stdin 을 `/dev/null` 로 바꿔 끼운다** (Step 14)
- **판정 시험의 「어휘 밖 경고가 없다」 는 봉인된 결과의 `OutOfVocabulary()` 가 비었는지로 본다** — 시험 서버의 로그는 버려진다 (Step 15)
- **source 훑기의 금지 이름에 `gitOut` · `gitOutName` 을 더했다** · 되돌림 모양의 글에서 아홉 자리를 찾는 대조군을 둔다. 가짜 세션이 호스트
  쪽 `$OUT` 에 쓰는 `doOut` 을 틀에 더했다 (Step 16)
- **`BenchmarkBeforeAdvert` 의 pending 줄은 부른 수만 세는 가짜 f 다** (Step 17)
- **integration** — rootfs 는 SunnyVM 의 버려도 되는 자리 안에 호스트의 dash · bash · git · mktemp 등과 라이브러리만 복사해 지었다. 끊고
  잇기는 helper 를 여는 명령을 감싼 sh 가 upper 의 절반이 옮겨진 순간에 unshare 를 SIGKILL 한다. bind 별칭 시험은 `--map-auto` 없이 돈다 (Step 18)
- **조각 6 의 took 시각** — 그 줄에 시각이 없어 merge 단계의 시작 (Record) 에 `took the lower lock after` 의 길이를 더해 셈한다. 조각 8 의
  두 번째 굽기는 형제 노드 (`SIBLING_WS`) 에 낸다 — 노드마다 임대가 하나라 같은 노드에 내면 기다린다 (Step 19)
- **조각 스크립트의 형제 모양 (Step 21 에서 고침)** — Step 19 는 형제를 bind 별칭 하나로 두고 `check_target` 이 `readlink -f` 로 같은 경로를
  막았다. 그런데 bind 별칭의 형제는 굽는 노드의 합치기를 잇지 못한다 (FD 규칙 10절 끝 — 시작 전 확인의 마운트 줄). 조각 7 (다른 노드가
  잇는다) 이 그 모양으로는 성립하지 않는다. 같은 경로 확인을 글자 비교로 바꿔 symlink 형제를 받는다 — 계약은 노드가 광고하는 ws 글자로
  고르고, lower 키와 merge 의 시작 전 확인은 symlink 를 푼다. `slice-7.sh` 는 형제가 다른 마운트면 멈추고, `slice-8.sh` 의 4 는 bind 별칭으로
  본다고 출력한다 (8.1). 계획 4.1 26번의 SIBLING_WS 는 「같은 lower 를 다른 경로로 가리키는」 이라 글자와 어긋나지 않는다. 판정 · 실패 등급 ·
  보안 경계는 그대로다
- **runc 시험의 한국어 경로 픽스처** — Step 7 은 「한국어 경로」 로 적었다. CONVENTIONS 2.1 이 시험 픽스처 문자열을 영어로 두므로 ASCII 밖의
  라틴 글자 경로 (`résumé/naïve.txt`) 로 바꿨다 — quotePath 를 보는 목적은 같다 (Step 20)
- **새 코드의 주석 구분선을 뺐다** — 저장소에 이미 있는 모양 (괘선 두 글자로 감싼 제목) 을 따랐다가 괘선 문자라 뺐다 (CONVENTIONS 1.2). 앞
  코드의 구분선은 안 건드렸다 (Step 20)
- **`BakeFlows` 시험과 재개 시험의 공유 변수 읽기** — 경합 검출기가 잡은 시험 쪽 읽기 둘 (`m.results` · `b.resuming`) 을 잠금 아래로 옮겼다.
  제품 코드의 경합은 아니다 (Step 20 전의 `-race`)

---

## 6. 코드 검사 (CI 와 같은 명령) · 측정

| 검사 | 결과 |
|---|---|
| `gofmt -l .` | 빈 출력 |
| `go vet ./...` · `go vet -tags integration ./internal/enode/ ./internal/merge/ ./internal/lower/` · `go build ./...` | exit 0 |
| 시험 단계 `go test ./... -count=1` (`ci.yml:216` · 시험 DB · 가짜 claude 스텁) | exit 0 (77초) |
| 커버리지 단계 (`ci.yml:267` 의 명령과 `:269` ~ `:300` 의 awk 그대로) | 통과 2,562 (Step 1 2,271 · 하위 시험 포함) · 실패 0 · 스킵 0 · 스물세 패키지 모두 80% 이상 (미달 0) · 전체 87.0% -> 87.6% (12,546/14,318) |
| Step 1 과 댄 패키지 | `internal/enode` 84.4 -> 86.7 (4,583/5,285) · `cmd/enode` 80.5 -> 80.8 (198/245) · `internal/merge` 89.1 -> 88.9 (311/350 — 새 문장 가운데 statx 실패와 번호 없는 커널의 두 갈래는 못 닿는다) · `internal/contract` 92.5 · `internal/store` 82.9 · `internal/api` 82.5 (셋 그대로) · `internal/lower` 93.2 -> 93.3 |
| 스킵 감시 (`ci.yml:343` ~ `:441` 의 awk 그대로) | 패키지 24 의 결과 · 허용목록 항목 0 · 허용목록 밖의 스킵 0 |
| `go test -race -count=1 -timeout 30m ./internal/enode/ ./internal/lower/ ./internal/merge/` | exit 0 (135초) · 통과 977 · 경합 0 |
| `go test -race -count=10 -run '^TestHeldBake_OneBodyUnderRace$' ./internal/enode/` | exit 0 |
| 흔들림 — 새 시험 `-count=20 -timeout 30m` (다섯 패키지 · `-run` 은 Step 1 의 목록과 지금 `go test -list .` 의 차이 · 이름 91) | exit 0 (377초) · 통과 1,820 (91 x 20) · 실패 0 · 스킵 0 |
| 흔들림 — CPU 부하 아래 (`taskset -c 0,1` 로 새 시험 `-count=3` · 옆에서 `taskset -c 0,1 go test -count=3 ./internal/lower/ ./internal/scratch/ ./internal/merge/` 를 끝날 때까지 되풀이) | exit 0 (71초) · 통과 264 (88 x 3) · 실패 0 · 옆의 다섯 번 실패 0 |
| 흔들림 — `internal/enode` 전체 `-count=5 -timeout 30m` | exit 0 (226초) · 윗 시험 통과 2,730 (546 x 5) · 실패 0 |
| 크로스 빌드 셋 (windows/amd64 · linux/arm GOARM=7 · darwin/arm64) | exit 0 |
| `GOOS=windows go test -c -o /dev/null ./internal/enode/` | Step 1 과 같은 오류 셋 (`overlay_test.go:41` · 이 유닛 전부터) |
| `enodectl.exe` 심볼 | crypto/tls 1 · net/http 6 (상한 10 · 50) |
| U+2605 · `go run ./scripts/glyphscan.go` | 0 · 168 파일에 장식 문자 없음 |
| `golangci-lint run ./...` | 38 — Step 1 과 같은 목록 (줄 번호를 뗀 파일 · 문장으로 diff). 새 코드에서 난 셋 (버린 `os.RemoveAll` 오류 둘 · 드모르간 하나) 을 고쳤다 |
| 조각 0 | build · vet · test 초록 · 라우트 19 (`internal/api/api.go` 안 고침) |
| 조각 5 (`scripts/finalize-bake/slice-5.sh` · 시험 DB) | `slice 5: green` — 목록의 스물하나 (하위 시험 포함) 가 모두 pass · 스킵 0 |

**파일마다의 커버리지** (CI 명령의 프로파일 · 계획의 「굽기 흐름 파일은 85% 이상을 겨눈다」) — `bakerule.go` 98.3 · `bake_resume.go` 93.5 ·
`bake_merge.go` 93.4 · `bake_build.go` 92.0 · `bake.go` 91.6 · `mergehelper_linux.go` 91.0 · `bakefile.go` 86.2. Step 20 이 처음 돈 때
`bakefile.go` 75.7 · `bake_resume.go` 84.6 이라 실패를 끼우는 시험을 더했다 — 쓸 자리가 없는 초안 · JSON 이 아닌 초안 · 옮길 수 없는 버려진
자리 · Lstat 뒤에 symlink 로 바뀌거나 자란 초안과 pinned (`afterLstat`) · 사라진 초안 · 막힌 키 자리 · 재개의 그물 (찾음 · 오류 · 기다리는
중 멈춤) · 못 읽는 state.json. 남은 것은 CreateTemp 뒤의 Write · Sync · Close 실패 같은 오류 갈래다. 고친 파일 — `finalize.go` 98.6 -> 99.3 ·
`lowerguard.go` 92.8 -> 93.2 · `diff.go` 67.9 -> 90.7 · `claim.go` 89.3 -> 89.7 · `workspace.go` 78.3 -> 84.9 · `runc_overlay_linux.go`
62.7 -> 63.9 (helper 안의 코드는 기본 시험이 못 닿는다 — integration 이 본다).

**기준선 흔들림** — Step 1 에 다섯 번 중 한 번 (`TestRuncOverlayOpenIncludesHelperStderr`) · 그 시험 혼자 300 번에 빨강 4. 원인 셋을 고친 뒤
300 번에 빨강 0 (`-coverpkg=./...` 로도 0). 이번 흔들림 확인에서 빨강은 하나도 없었다.

**측정** (계획 3.2 · Step 17 · 이 기계 Intel N100 4코어 · 커널 6.5 · ext4 · `-benchtime 2s` 세 번)

```text
   벤치마크                               값                  계획의 추정 · 대 본 값             뜻
   BenchmarkBeforeAdvert/committed        23.2 ~ 24.1 µs      26.7 ~ 29.8 µs (3절 측정 1)        광고 한 번의 몫 — 주기 60초에 견주어 무시할 만하다
   BenchmarkBeforeAdvert/pending          34.3 ~ 34.8 µs      같은 자리                          공유를 안 쥐고 OnStale 을 부른다 (가짜 f)
   BenchmarkOnStaleWhileTheOwnerLives     13.0 ~ 13.4 µs      TryBake 헛시도 7.7 ~ 8.4 µs         배경 일 한 번 — 고루틴 하나와 wg 기다림이 더 든다
   BenchmarkStaleCleanup                  4.3 ~ 5.3 ms        1.5 ms 근처                        정리 한 번 — 자기 scratch 의 pending/<키>/ 훑기 ·
                                                                                                trash 의 MkdirAll · rename · fsync 두 번
   BenchmarkMergeStepFixedCost            17.7 ~ 19.2 ms      그물 5.2 ms + helper 두 번 +       빈 upper 의 merge 단계 전부 (namespace 없는 helper)
                                                              state 쓰기 둘과 metadata
```

한 자리 넘게 벗어난 값은 없다. 형제가 기다리는 시간은 형제 Run 과 광고 주기가 정하고, merge 가 배타를 쥐는 고정 비용은 20 ms 안이다 —
하루치 Preflight · Apply 는 merge-rules 의 값 (SunnyVM 0.23 초 · 1.11 초 · 이 기계 0.63 초 · 3.80 초) 을 옮겨 적는다 (다시 돌리지 않았다).

**다시 돌리는 명령** (진행자 · 시험 DB 는 `scripts/testdb.sh` · 스크래치 자리는 바꿔 쓴다)

```bash
export ENODE_TEST_DATABASE_URL='postgres://enode:enode@127.0.0.1:<port>/<db>?sslmode=disable'
export PATH="$PWD/.github/ci-stubs:$HOME/go/bin:$PATH"
go test ./... -count=1
go test ./... -count=1 -coverpkg=./... -coverprofile=<scratch>/cover.out -json > <scratch>/test.json
go test -race -count=1 -timeout 30m ./internal/enode/ ./internal/lower/ ./internal/merge/
go test -race -count=10 -run '^TestHeldBake_OneBodyUnderRace$' ./internal/enode/
go test -count=5 -timeout 30m ./internal/enode/
go test -count=300 -run '^TestRuncOverlayOpenIncludesHelperStderr$' ./internal/enode/
go test -run '^$' -bench 'BeforeAdvert|OnStaleWhileTheOwnerLives|StaleCleanup|MergeStepFixedCost' -benchtime 2s -count 3 \
  ./internal/enode/
SLICE_DIR=<scratch>/slice-5 scripts/finalize-bake/slice-5.sh
git checkout -- cmd/enodectl/probe.lock    # 시험이 바꾼다
```

---

## 7. integration 시험 (`integration` 태그 · CI 밖)

```text
   시험                                                   이 기계                               SunnyVM (2026-09-30 · ~/bake-it-<시각>)
   TestBindAliasPreflightIntegration (internal/merge)       초록                                  초록 — 마운트 196 대 8031 · Check mount
   TestMergeHelperIntegration                               빨강 — newuidmap: write to uid_map     초록 — subordinate uid 파일 · 000 디렉터리 ·
                                                            failed (--map-auto 가 막힌다)          종류가 바뀐 항목 하나가 trash 로
   TestResumeAfterAKilledHelperIntegration                  빨강 — 같은 까닭                       초록 — 100,000 가운데 46,405 가 남았을 때 죽임 ·
                                                                                                  재개가 46,409 연산을 1.159 초에 · resumed true
   TestRuncOverlayCloseKeepsTheUpperIntegration             건너뜀 — rootfs 가 없다                 초록 — 단계 사용자가 쓴 파일이 노드 uid 소유로
   TestRuncOverlayFinalizeDiffInTheSessionIntegration       건너뜀 — 같은 까닭                      초록 — workspace.diff 127 바이트 · 호스트 표지 없음
```

- SunnyVM (커널 7.0.0-31 · 8코어 · git 2.43.0 · runc 1.3.4 · `kernel.apparmor_restrict_unprivileged_userns` 0) 에서는 버려도 되는
  `~/bake-it-<시각>` 안에서만 돌렸다. rootfs 는 그 안에 지었다 (9.5 MB). `/srv/yocto` 와 떠 있는 노드의 설정 · 상태 자리는 읽지 않았다.
  끝나고 자리를 지웠다 (`~/bake-it-*` 0)
- 이 결과로 조각 6 · 7 · 8 을 대신하지 않는다 — rootfs 안의 git 소유 확인 · 진짜 굽기 계약 · 옛 판 노드는 조각이 처음 본다

---

## 8. 사람 조각 6 · 7 · 8 — 돌리는 법 (병합 게이트 · 에이전트는 돌리지 않았다)

**에이전트는 조각 6 · 7 · 8 을 돌리지 않았다** — 사람이 보는 조각이다 (US-14). 코드 시험이 초록이라는 것으로 대신하지 않는다. SunnyVM 은
닿는다 (2026-09-30 · Step 1 · 18) — 보류가 아니다. 조각이 아직 안 돌았으므로 병합 지점도 아니다 (CONVENTIONS 3.3). 사용자가 돈 결과는
PR 전에 audit 과 이 절에 적는다.

### 8.1 준비

```text
   기계        SunnyVM.  떠 있는 노드 둘 (~/bin/enode · 2026-09-22 판) 은 lower-state 전의 판이라 굽기를 모른다 — 쓰지 않는다
   바이너리    이 브랜치에서 짓는다 — go build -o <자리>/enode ./cmd/enode
   노드 둘     runc-overlay 실행 환경 (workspace.writes=isolated) 의 노드 둘.  설정 파일을 따로 둔다 (노드 신원이 설정 경로에서
               나온다).  둘 다 lower 의 주인 uid 로 돈다.  scratch 는 노드마다 따로 두고 lower 와 같은 filesystem · 같은 마운트에
               둔다 (env check 의 binding.scratch_filesystem 이 ready 여야 한다)
   rootfs      준비된 실행 환경의 rootfs 에 bash 와 git 이 있어야 한다 — build 는 bash 가 없으면 sync 전에 멈추고, IR 대조와 pinned 는
               세션 안의 git 을 쓴다
   lower       버려도 되는 자리를 새로 만든다.  그 뿌리가 굽는 노드의 워크스페이스 (WS) 다
                 mkdir -p <자리>/lower && touch <자리>/lower/.enode-disposable
               표지 .enode-disposable 이 보통 파일로 없으면 스크립트가 운영 lower 로 보고 멈춘다.  표지는 lower 를 복사하면 함께
               간다 — 스크립트가 장치와 inode 를 출력하므로 사람이 뜻한 자리인지 본다
   형제        같은 lower 를 다른 경로로 가리키는 형제의 워크스페이스 (SIBLING_WS).  글자가 WS 와 같으면 계약이 두 노드를 나누지
               못해 멈춘다 (계약은 노드가 광고하는 ws 글자로 고른다)
                 조각 6 · 7   symlink — ln -s <자리>/lower <자리>/sibling.  같은 마운트라 형제가 굽는 노드의 합치기를 이을
                              수 있다.  slice-7.sh 는 형제가 다른 마운트면 멈춘다
                 조각 8       bind 별칭 — sudo mount --bind <자리> <별칭 자리> 로 부모를 걸고 SIBLING_WS=<별칭 자리>/lower ·
                              형제의 scratch 는 <별칭 자리>/scratch-b (형제의 워크스페이스와 같은 마운트).  8 의 4 (bind 별칭 두
                              노드가 같은 상태 자리) 는 이 모양으로 본다.  bind 별칭의 형제는 굽는 노드의 합치기를 잇지 못한다
                              (FD 규칙 10절 끝 · 시작 전 확인의 마운트 줄) — 조각 8 에서 형제는 합치지 않고 정리만 한다
   저장소      SYNC_URL — sync 가 받아 올 git 저장소 · IR — 그 저장소의 태그.  sync 는 세션 (컨테이너) 안에서 돈다 — 호스트의
               경로는 컨테이너에 안 보이므로 세션 안에서 닿는 주소를 준다
   Mediator    M (주소) · T (bootstrap 토큰) — slice-4.sh 와 같은 이름
   도구        go · jq · curl · git · stat · unshare · flock 과 git config user.email (runctl 이 쓴다).  합치기 전의 merged view
               목록은 unshare --map-auto 가 되는 기계여야 한다 (SunnyVM 은 된다)
```

### 8.2 돌리는 명령

저장소 뿌리에서 돈다. 스크립트는 판정하지 않는다 — 대상을 출력하고 (`== target ==` · `== nodes on this lower ==`), 차례대로 돌리고,
`>> look:` 에서 무엇을 볼지 출력한 뒤 Enter 를 기다리고, `>> do:` 에서 사람이 할 일 (노드 멈추기 · 띄우기) 을 기다린다.
`PAUSE=0` 이면 look 에서 기다리지 않는다.

```bash
export M=http://<mediator>:8080 T=<bootstrap token>
export WS=<place>/lower SIBLING_WS=<place>/sibling
export SYNC_URL=<git repository reachable from the session> IR=<a tag in it>
export BUILD_A='<your build command for config-a>' BUILD_B='<your build command for config-b>'
export SIBLING_LOG=<the sibling node log file>        # slice 6: the released time; empty means read it by hand
export NODE_CONFIG=<the baking node config file>      # slices 7 and 8: the pid is the first line of <config>.lock
scripts/finalize-bake/slice-6.sh
scripts/finalize-bake/slice-7.sh                      # KILL_AFTER="0 0.3 1" and MANY=50000 by default
SIBLING_WS=<alias place>/lower scripts/finalize-bake/slice-8.sh
```

`BUILD_A` · `BUILD_B` 를 비우면 워크스페이스에 파일 하나를 쓰는 짧은 명령이다. 스크래치는 `SLICE_DIR` (기본 `${TMPDIR:-/tmp}/enode-slice-bake`).

### 8.3 무엇을 보나

```text
   조각 6   1      빈 lower 에 굽기 A — build · merge DONE · Run SUCCEEDED · .enode-metadata.json 의 칸 전부 · bake.run · bake.node.
                   처음 확인 — build 가 pending 까지 갔으면 rootfs 에 bash 와 git 이 있고 세션 안 git 이 lower 의 주인을 받아들였다
                   (safe.directory 거절 없음)
            2 · 3  형제에 긴 Run (SIBLING_SECONDS · 기본 300) 뒤 굽기 B — build 는 곧바로 끝나고 merge 가 기다린다 · 단계 로그에
                   쥔 쪽 줄 (형제 노드 · 그 Run) · 노드 시계의 마감 · 남은 시간 (완료 조건 5) · 형제는 draining
            4      네 시각 — 형제 Run 이 끝난 때 · 형제의 released the lower lock · merge 의 took the lower lock (시작 + 길이) ·
                   committed.  기대 창 — released 는 형제 Run 끝에서 두 광고 주기 (기본 60초씩) 안 · took 는 released 뒤 1초 안 ·
                   장면 2 의 4 의 「몇 초」 는 took 부터 committed 까지 (CG 물음 4 답)
            5      두 노드가 새 ir 과 repo.built.config-a · config-b 를 광고한다
            6      합치기 전 merged view 목록과 합친 lower 목록이 같다 (경로 · 종류 · 권한 · .enode-metadata.json 은 뺀다)
            7      빌드 하나를 일부러 실패시킨 굽기 — build DONE exit 3 · config-b skipping · merge 는 합칠 것 없음 DONE ·
                   Run FAILED (produced 조건) · last_attempt.  sync 가 IR 에 닿지 않는 굽기 — build DONE reason ir_mismatch ·
                   head · head_tags · manifest 없음 · 단계 로그 끝의 문장 · Run FAILED (완료 조건 10 · US-19)

   조각 7   1      형제를 멈춘 채 merging 을 본 뒤 KILL_AFTER 초마다 굽는 노드를 SIGKILL -> 형제를 띄운다 -> 형제 로그에
                   bake: resuming an interrupted merge (from=start) 와 resumed 줄 · state committed · metadata bake.resumed true 와
                   원래 Run · 합친 목록이 합치기 전 merged view 목록과 같다.  끊길 틈이 있게 구성 하나가 파일 MANY 개를 쓴다
                   (처음 한 번 구워 두고 뒤의 굽기가 모두 다시 쓴다)
            2      형제를 띄워 둔 채 SIGKILL -> 형제가 한 광고 주기 안에 잇는다 (from=advert)

   조각 8   1      굽기 중 형제에 두 번째 굽기 -> 곧바로 FAILED bake_in_progress 와 주인 문장 (US-16)
            2      pending 에서 굽는 노드를 SIGKILL -> 떠 있는 형제가 두 광고 주기 (기본 120초) 안에 정리 (bake: cleaned a stale
                   bake) · last_attempt 는 abandoned: no process held the bake while it was pending · 대기 자리는 그 scratch 의
                   trash · drain 이 풀린다
            3      merge.wait 1m 굽기 + 형제에 5분 Run -> build DONE · merge FAILED merge_wait_timeout · Record 에서 둘이 따로
                   (완료 조건 9 · US-17) · upper trash · committed
            4      bind 별칭 형제 — 두 키가 같고 상태 자리 하나에 두 노드의 쥔 사람 기록
            5      env check 의 not ready 셋 — 다른 uid 의 노드 (lower.owner_uid) · 다른 filesystem 이나 마운트의 scratch
                   (binding.scratch_filesystem) · 다른 디렉터리를 적은 lower.json (lower.identity).  사람이 노드를 그 모양으로
                   세워 enodectl env check 를 돈다
            6      늦게 매칭된 형제 Run — 합치기 전에 매칭됐다가 뒤에 돌면 lower_changed (다시 내면 된다) · 뒤에 매칭됐으면 새
                   lower 에서 SUCCEEDED.  바뀐 lower 에서 lower_changed 없이 돈 Run 이 틀린 것이다
```

### 8.4 받는 일 44 의 셋째 — 옛 판 노드가 있는 lower

잔여가 아니라 이 조각에서 처음 본다 (FD 흐름 12절). 둘을 본다.

- **스크립트가 멈춘다** — lower-state 전의 판 노드 (`workspace.writes` 를 광고하지 않는다 · 예를 들어 SunnyVM 의 `~/bin/enode`) 를 같은 lower
  에 (또 다른 symlink 경로로) 띄운 채 조각 스크립트를 돌리면 `check_same_lower` 가 `advertises no workspace.writes: an older enode is on
  this lower` 와 `move every node on this lower to a build that bakes before baking; stopping` 으로 멈춘다
- **그물이 옛 판의 마운트를 찾는다** — 옛 판은 lower 잠금을 모른다. 확인이 끝난 뒤 옛 판 노드를 띄우고 그 노드에 긴 명령 단계를 낸 채
  (runc-overlay 면 lower 위에 overlay 를 건다) 굽기를 내면, merge 가 배타를 잡은 뒤 단계 로그에 `found 1 overlay mount of this lower in
  another mount namespace; releasing the lock and waiting 60s` 와 그 마운트 줄이 나오고 그 명령이 끝날 때까지 합치지 않는다. 그물은 증거가
  아니다 — 옛 판을 모두 올린 뒤 굽는 것이 규칙이다 (스크립트의 첫 확인)

---

## 9. 행렬 밖 파일의 diff (계획 2절의 열하나)

```text
   internal/contract/result.go      +9 -1    ReasonIRMismatch = "ir_mismatch" 와 주석 · BuildManifest.HeadTags (json head_tags · omitempty 없음) ·
                                             IR 칸의 주석
   internal/store/claim.go          +2 -1    OutOfVocabulary 의 reason 목록에 contract.ReasonIRMismatch · Mediator
   internal/merge/merge.go          +3       CheckMount Check = "mount"
   internal/merge/decide.go         +12      checkMounts(upper, lower, trash uint64) — 문장은 FD 규칙 10절 그대로
   internal/merge/merge_linux.go    +20 -5   resolve 가 statx STATX_MNT_ID 로 마운트 번호 셋 · roots.mounts [3]uint64 ·
                                             번호가 없는 커널은 merge preflight: <path>: the kernel does not report a mount id
   internal/enode/finalize.go       +21 -6   contractStep 이 굽기 칸 넷을 옮기고 kind 를 switch 로 (FD 흐름 10절) ·
                                             settleIn.sealErr 와 settle 의 여섯 줄 (아래)
   internal/enode/lowerguard.go     +29      onStale 칸 · OnStale · Dir · sourcesLocked 의 부르는 줄
   internal/enode/runtime.go        +4 -3    주석 셋 — Keep (대기 자리가 쓴다) · FinalizeSpec.Diff (runc-overlay 는 세션 안에서)
   internal/lower/lower.go          +1 -1    State.PendingUpper 의 주석 — building 부터
   internal/enode/diff.go           +171 -13 repoDiff 안의 스크립트 둘을 repoDiffScript · repoStatScript 로 (native 바이트 그대로) ·
                                             sessionDiffScript · sessionDiffEnv · sessionDiff · sessionDiffError · placeWorkspaceDiff ·
                                             diffLimit · diffBuffer · tailBuffer
   internal/enode/workspace.go      +17 -2   Prepare 가 WorkspaceWrites(w.Runtime) 를 본다 — isolated 면 저장소 확인만 · PrepClean ·
                                             로그 how (fresh upper · reset and clean)
```

**`runc_overlay_linux.go` 의 finalize 부분** (행렬 안의 파일이나 finalize 유닛의 수확을 고친다 · 4절 34번) — `runcOverlaySession.Finalize`
가 helper 에 보내는 요청의 Diff 를 끄고, helper 의 답 뒤에 `diffInSession` (세션의 Run 으로 `sessionDiffScript` · 마감이 먼저 오면
`the finalize deadline passed before the workspace diff`) 을 돈다. helper 의 `h.finalize` 도 Diff 를 끈다. native 의 `finalizeLocal` 과
`workspaceDiff` 는 그대로다.

**`finalize.go` 의 `settleIn.sealErr` 한 칸은 FD 가 허락한 것이 아니다** — FD 흐름 10절이 finalize.go 에 적은 것은 `contractStep` 하나다. 이
계획이 더했고 (4절 4번) finalize FD 3.2 의 settle 표에 한 줄이 는다 — 「sealErr 가 있으면 finalize 는 error (ok 였으면) · 그 문장을 error 에
잇는다」.

```go
	if in.sealErr != nil {
		if finalize == contract.StageOK {
			finalize = contract.StageError
		}
		errs = append(errs, in.sealErr.Error())
	}
```

Mediator 쪽은 `result.go` · `store/claim.go` 의 둘이다. 원인 코드 상수 하나와 어휘 한 줄이고 라우트는 19 그대로다.

---

## 10. 다른 유닛과 진행자에게 넘기는 것

```text
   checkpoint     실패한 굽기의 upper 를 spool 로 받을지 (FD 흐름 12절 그대로).  온전한 upper 를 trash 로 보내는 길 —
                    build 의 명령 실패 · IR 어긋남 (Close(Keep{})) · 몸통의 대기 상한 · Preflight 어긋남 · 예산 초과 · 임대가 사라짐.
                    보존하면 Keep{Upper: spool} (build 끝) 이나 몸통이 옮길 곳 (대기 자리 -> spool) 을 바꾼다.  merging 뒤의 upper 는 보존
                    대상이 아니다
                  (더함) Close(Keep{Upper}) 는 RENAME_NOREPLACE 라 keep 자리가 이미 있으면 실패하고 upper 는 trash 로 간다 — spool 자리를 새
                    이름으로 준다.  keep 은 첫 Close 에서만 본다 (manageSession 이 감싼 세션의 둘째 Close 는 옮기지 않는다)
                  (더함) claim.go 의 closeOut · runc_overlay_linux.go · main.go · runtime.go 를 이어 고친다 — 한 줄 순서라 이 유닛 위에서

   사람 · PR 전    조각 6 · 7 · 8 (8절).  그 안에서 처음 확인하는 것 — 세션 안 git 의 소유 확인 · rootfs 의 git 과 bash · 합치는 중 SIGKILL 과
                    재개 · 조각 8 의 준비도 점검 · 옛 판 노드가 있는 lower (받는 일 44 의 셋째)

   진행자 (잔여)   조각 6 ~ 8 로도 닿지 않는 셋 — ext4 밖 filesystem 의 fsid 가 재부팅에 같은지 · 전원이 나간 뒤 state.json ·
                    repo init -b 뒤 manifests 의 로컬 태그 (FD 흐름 12절)
                  사람의 수단이 state.json 손 고침뿐인 두 갈래 (초안이 없음 · pending_upper 의 모양이 틀림) 를 운영 문서에 적을지
                  팩 decisions.md · features.md · scene-gates.md · requirements.md 10절 (FD 흐름 13.2)
```

**잔여 · 후속 과제 — 계약의 env 이름으로 노드의 비밀이 명령에 실리는 길** (CG 물음 2 답 · 2026-09-27T14:41:18Z). 계약의 `env` 는 이름에
제한이 없어 `ENODE_TOKEN` 같은 노드 비밀의 이름을 부르면 그 값이 굽기 명령과 명령 단계의 환경에 실린다 (`contract/bake.go:96` ·
`env.go:107` ~ `:111` · `main.go:179`). 오늘 명령 단계에도 있는 길이라 이 유닛이 만든 것이 아니다. 이 유닛의 시험
(`TestBakeBuild_UnnamedHostVariablesDoNotReachTheCommands`) 은 「계약이 이름으로 부르지 않은 호스트 환경 변수는 닿지 않는다」 까지만 본다.

---

## 11. 정본과 회차 문서

**정본 되돌림 — 진행자가 `enode-design` 에 올린다** (FD 흐름 13.1 그대로 · 이 단계는 정본을 고치지 않았다). 옮길 때 13.1 의 글자 둘을 고친다.

- **run-contract 의 미정 항목 주소** — `:1068` 이 아니라 `:1069` 다 (「명령 단계의 셸 — sh -c 인가 argv 배열인가」)
- **「bash -c 가 그 래퍼다」 는 정본과 반대다** — run-contract 의 yocto 절 첫 항목 (`:1122` ~ `:1127` · `:1146`) 의 답은 노드가 가진 래퍼
  스크립트이고 계약에 셸을 여는 것이 아니다. 13.1 의 그 줄은 「**굽기는 계약의 문자열을 bash -c 로 연다 · run-contract 의 그 항목과 다르다**」
  로 적는다

13.1 에 두 줄을 더한다.

```text
   mediator-api    :472 「exit_code — 명령 단계만」 — build 단계의 결과도 exit_code 를 싣는다 (마지막으로 돈 계약 명령의 값).
                   판정 재료가 아니다 — 판정은 계약의 produced 조건이다
   execution-      :610 ~ :612 「process 가 직접 $OUT 에 쓴 파일은 지금처럼 자동 수확한다」 — build 단계는 세션의 $OUT 을 올리지
   environment     않는다.  노드가 만든 폴더의 manifest 하나만 올린다 (4절 5번)
```

4절 15번 (merge 의 업로드 예산) · 33번 (격리 노드의 Prepare) 은 정본을 따르게 한 것이라 되돌림이 없다.

**정본에 보탤 것 (되돌림이 아니다 · 진행자가 정한다)** — 「격리 노드에서 호스트 쪽 helper 는 워크스페이스가 적은 설정을 실행하는 명령을 돌리지
않는다」 (4절 34번 · FD 흐름 13.1 끝). execution-environment.md §10.3 과 ADR-073 §6 은 diff 를 Close 전에 merged view 에서 만든다고만 적었다 —
세션 안의 작업 폴더가 그 merged view 라 어긋나지 않는다.

**회차 문서** — `component-methods.md` 4.2 (`runBuildStep` · `runMergeStep` 의 인자 · `Worker.Bake` · `StartBaker`) · 4.3 (`OnStale` 을 guard 가
곧바로 부르고 f 는 막지 않는다 · 두 함수는 nil 수신자에 안전) · 4.4 (`RunMergeHelper` 의 주고받는 한 줄 · 여는 명령) 를 코드의 겉면에 맞췄다.
계획 단계가 고친 FD 두 장과 `unit-of-work.md` 7절은 계획 2절이 적은 그대로다.
