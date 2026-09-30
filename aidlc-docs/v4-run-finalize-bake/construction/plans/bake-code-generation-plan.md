# `bake` — Code Generation 계획

**유닛** `bake` (굽기 단계 · 한 줄 순서의 일곱째) · **브랜치** `unit/bake` · **기준** `fb86ad2` (Functional Design 커밋 ·
`main` `0262155` 위) · **맡는 조각** 5 (굽기 계약 — 기계) · 6 (굽기가 끝까지 돈다) · 7 (끊겨도 된다) · 8 (배타와 대기) — 6 · 7 · 8 은
사람 · SunnyVM · 버려도 되는 lower · **병합 조건** 조각 5 · 6 · 7 · 8 이 초록 (`unit-of-work.md` 0절)

이 계획이 이 유닛 Code Generation 의 기준이다. 여기 없는 것은 짓지 않는다. 설계는 `construction/bake/functional-design/` 의 셋이다.
아래에서 쓰는 줄임은 앞 유닛 계획과 같다.

```text
   FD            Functional Design 의 줄임
   FD 규칙 N절    business-rules.md N절
   FD 흐름 N절    business-logic-model.md N절
   FD 엔티티 N절  domain-entities.md N절
   FD 계획        plans/bake-functional-design-plan.md
   물음 N 답      FD 계획 4절의 물음 넷 — 1 B (IR 어긋남은 build 의 DONE · 판정은 계약의 조건) · 2 A (끊긴 합치기는 시작할
                 때와 살아 있는 동안 잇는다) · 3 A (builds 는 첫 실패에서 멈춘다) · 4 A (조각 스크립트의 허용 표지)
   되물음 N 답    plans/bake-functional-design-clarification-questions.md — 1 A (광고 주기도 낡은 상태를 치운다) ·
                 2 B (합칠 것 없음은 DONE) · 3 A (bash -c) · 4 A (build claim 이 낡은 merging 을 만나면 재개를 연다) ·
                 5 A (답 뒤에 정한 다섯을 산출물대로)
   U7            이 회차 유닛 한 줄 순서의 일곱째 (bake) — audit · 상태 파일의 절 이름.  .coverage-contract.yml:1 의 U7 은
                 앞 회차의 다른 유닛이다
   결정 N         audit.md 의 U7 항목들이 적은 「답에 없던 설계 결정」 (1) ~ (55)
   받는 일 N      FD 계획 1절 표의 번호 (55 · 56 은 FD 흐름 11.2 가 더했다)
   팩 결정 N-M    요구 팩 requirements/finalize-bake/decisions.md 의 행 (예 — 3-10 은 pending · merging 이면 형제가 drain 을 싣는다)
   lower-state 답 N  lower-state 유닛 FD 계획의 물음 N 의 답 — 답 1 은 형제가 임대가 0 인 drain 응답을 두 번 받은 뒤에 공유
                 잠금을 놓는다 · 답 6 은 같은 filesystem 확인에 마운트까지 본다
   ADR-077 §N    정본 enode-design/adr 의 굽기 결정 (「굽기는 upper 에서 짓고 lower 에 합친다」) 의 절.  §12 는 실측 절이다
   M · T         조각 스크립트의 입력 — Mediator 주소와 bootstrap 토큰 (앞 조각 스크립트 slice-4.sh 머리와 같은 이름)
   4절 N번        이 계획 4.1 표의 번호
   CG 물음 N 답   이 계획 (Code Generation 계획) 10절 물음 N 의 답 — FD 파일에 고친 줄의 근거로도 이 글자를 단다
```

- **작성 시각**: 2026-09-27T13:42:42Z (NFR 건너뜀 2026-09-27T13:14:53Z 뒤) · 줄 번호 넷을 고친 판 2026-09-27T13:54:10Z · QA 1회
  지적을 고친 판 2026-09-27T15:02:00Z · 물음 넷의 답을 반영한 판 2026-09-29T14:10:40Z · 물음 5 의 답을
  반영한 판 (마지막으로 고친 시각) 2026-09-29T15:15:25Z
- **입력**: FD 셋 · FD 계획과 되물음 · audit 의 U7 항목 열여섯 · 유닛 정의 `unit-of-work.md` 0 · 7 · 10절 · 파일 행렬의 ⑦ 열 (bake 가
  고치는 칸) · `components.md` 5절 (실패 등급) · `component-methods.md` 4.2 · 4.3 · 4.4 · 9절 · `services.md` 2 ~ 5절 ·
  `component-dependency.md` · `requirements.md` FR-5 ~ FR-9 (굽기 계약 · upper 에 짓기 · 합치기 · lower 상태와 재개 · metadata 와 IR
  대조) · 5.1 · 5.3 · 5.4 · 5.7 · 6절 · 팩 `scene-gates.md` 조각 5 ~ 8 · `decisions.md` 3-7 ~ 3-22 · `constraints.md` 2절 · 앞 유닛
  셋 (lower-state · merge-rules · trash) 의 Code Generation 계획과 code-summary · 코드 (줄 번호는 `fb86ad2` 기준 — 4.2)
- **NFR 두 단계** (NFR Requirements · NFR Design — 비기능 요구와 그 설계): 사용자 결정으로 건너뛰었다 (2026-09-27T13:14:53Z ·
  「건너뛰고 다음으로.」). 유닛 정의 7절이 NFR 에 둔 보안 · 성능을 이 계획의 3절이 받는다

---

## 1. 유닛 맥락

**하는 일** — 노드가 굽기 계약의 두 단계를 받는다. build 단계는 굽기 잠금 · building · 세션 안의 bash 확인 · sync · IR 대조 ·
pinned manifest · builds 를 차례로 돌리고, 성공하면 upper 를 대기 자리로 옮긴 뒤 초안과 pending 을 쓴다. merge 단계는 claim 확인 ·
형제를 기다리는 배타 (대기 로그) · 그물 · 시작 전 확인 · merging · merge-helper 로 합치기 · metadata · committed 다. 주인이 죽은 낡은
상태와 끊긴 합치기는 기동 · 광고 주기 · build claim 이 한 표로 치우거나 잇는다 (물음 2 답 · 되물음 1 답 A). merge 의 시작 전 확인에
마운트 줄을 더한다 (lower-state 답 6 — 같은 filesystem 확인에 마운트까지). 조각 6 · 7 · 8 의 스크립트는 운영 lower 면 멈춘다
(물음 4 답). 격리 노드의 워크스페이스 준비 (Prepare) 가 호스트에서 git · repo 를 돌리지 않게 한다 — 굽기 뒤 lower 에 남는 계약의 파일을
호스트가 실행하지 않게 하는 것이고, 정본 ADR-072 결정 3 을 코드가 따르게 하는 것이다 (CG 물음 3 답 A · 4절 33번). 같은 까닭으로 격리 노드의
Finalize 가 만드는 workspace.diff 를 세션 안 (준비된 rootfs 의 git · repo) 으로 옮긴다 — 굽기 전부터 있던 구멍이고 finalize 유닛의 수확을
고친다 (CG 물음 5 답 · 4절 34번).

**완성하는 스토리** — US-6 (lower 를 마지막으로 바꾼 굽기 Run 과 노드) · US-12 · US-15 (merge 가 누구를 언제까지 기다리나) ·
US-13 (대상 lower 가 버려도 되는 것인가) · US-16 (bake_in_progress 는 다시 내면 되는 거절) · US-17 (merge_wait_timeout 이면 빌드는
성공했다) · US-18 (FAILED 로 봉인된 굽기가 재개로 합쳐졌나) · US-19 (IR 대조가 어긋나면 sync 가 어디에 닿았나). 완료 조건 5 (merge 가
누구를 언제까지 기다리나) · 6 (재개로 합쳐졌나) · 8 (운영 lower 면 멈춤) · 9 (build 성공과 merge 사유가 따로 보인다) · 10 (계약의
IR 과 sync 뒤 HEAD 가 어긋난 이유).

**맡는 조각** — 5 (기계 · 기본 `go test` 와 제출 경로의 400) · 6 · 7 · 8 (사람 · SunnyVM · 버려도 되는 lower). 넷이 병합 게이트다
(CONVENTIONS 3.3 — 장면 게이트가 초록인 뒤에 병합). 기대는 조각 1 (걷지 않는다) · 2 (보인다) · 4 (trash) 는 초록이다. 조각 7 의 기계
부분 (가짜 트리 재개 시험) 은 merge-rules 가 이미 초록으로 두었다 (`TestApplyResume`).

**의존** — 앞 유닛 여섯이 `main` 에 있다.

```text
   contract-grammar   Step 의 굽기 칸 · MergeWait · ArtifactManifest · ArtifactMerged · EnvIR · examples/bake.json 의 produced 조건
   step-phase         Mediator 가 build 의 exited 를 받고 merge 의 것은 안 받는다 · result 의 build · merge 칸 · reason 어휘 검사
   finalize           Finalize · 두 예산 · startExitReport · settle · 업로드 client (Worker.upload)
   trash              scratch.Trash.Move · TrashIn · Keep · 삭제자 · 기동 청소 (enode-runc- 로 시작하는 이름만)
   merge-rules        merge.Preflight · Apply · Result · PreflightError · OpError
   lower-state        lower.Open · TryBake · Bake.WriteState · Exclusive · Waiting · ForeignMounts · ReadMetadata · WriteMetadata ·
                      LowerGuard 의 HoldBake · DropBake · 광고 키
```

**뒤 유닛이 기대는 것** — checkpoint 가 `Close(Keep{Upper})` 의 rename (4절 6번) 을 spool 로 쓴다. 실패한 build 의 upper 를 보존하게
되면 몸통이 옮길 곳을 바꾼다 (FD 흐름 12절).

**새 이름**

```text
   internal/enode (내보냄)     Baker · StartBaker · Baker.Wait · Worker.Bake · Draft · RunMergeHelper ·
                               LowerGuard.OnStale · LowerGuard.Dir · Step.Sync · Step.Builds · Step.IR · Step.Merge ·
                               Result.Build · Result.Merge
   internal/enode (안 내보냄)   heldBake · abandon · draftBuilds · startMerging · cancelMerging · runBuildStep · runMergeStep ·
                               resume · onStale · clean · irProbe · irOutcome · irVerdict · parseProbe · probeScript ·
                               attemptReason · abandonedReason · buildManifestOf · mergeOpsOf · metadataOf · waitLog ·
                               pendingShape · stripPassword · repoIDOf · finished · mergeClaim · closing · closeOut · readPinned ·
                               mergeHelperRequest · mergeHelperResponse · mergeHelperArgv · mergeHelperCommand ·
                               callMergeHelper · helperPreflightError · helperOpError · helperErrorOf · fileOwner ·
                               beforeStartMerging (시험 훅 · 제품은 nil) · sessionDiffScript · diffInSession · sessionDiffError ·
                               placeWorkspaceDiff · diffLimit (시험이 줄이는 상한 · 제품은 maxBlobBytes) · repoDiffScript ·
                               repoStatScript (diff.go 의 repoDiff 안 상수를 패키지 상수로 올린 것)
   internal/contract           ReasonIRMismatch · BuildManifest.HeadTags
   internal/merge              CheckMount · checkMounts
```

**경계** — `internal/enode` 가 `internal/merge` 를 처음 가져온다. 경계 시험의 금지는 Mediator 쪽 한 줄뿐이라
(`{"cmd/mediator", "internal/merge"}`) 고칠 줄이 없다. `internal/merge` 의 봉인 (표준 라이브러리와 `golang.org/x/sys`) 은 그대로다.
`go.mod` 를 안 움직인다 — `x/sys` 에 `Statx` · `STATX_MNT_ID` 가 있다 (lower-state 가 이미 쓴다). Mediator 에 닿는 것은 원인 코드
상수 하나와 어휘 한 줄이다.

**크로스 빌드** — linux 에만 있는 것 넷 (merge-helper 입구 `RunMergeHelper` · 여는 argv `mergeHelperArgv` · `mergeHelperCommand` ·
helper 를 띄우고 한 줄씩 주고받는 호스트 함수 `callMergeHelper` · 파일 주인 uid 를 읽는 `fileOwner`) 을 `mergehelper_linux.go` 에 두고,
`mergehelper_other.go` 가 부르는 쪽이 쓰는 이름 셋 (`RunMergeHelper` · `callMergeHelper` · `fileOwner`) 의 짝으로 지원 안 함을 돌려준다
(FD 엔티티 12절 · `unit-of-work.md` 10절 · 선례 `trash_other.go` 가 `RunTrashHelper` 와 함께 `TrashLauncher` · `SweepOrphanSessions` 를
둔다). 부르는 쪽 `bake_merge.go` · `bake_resume.go` · `bake.go` 는 모든 플랫폼 파일이다. 주고받는 타입과 응답을 오류로 되돌리는 것
(`mergeHelperRequest` · `mergeHelperResponse` · `helperPreflightError` · `helperOpError` · `helperErrorOf`) 은 모든 플랫폼 파일
`bakerule.go` 에 둔다 — 부르는 쪽이 errors.As 로 본다. 초안 파일은 모든 플랫폼에서 빌드되는 방법으로 읽고 쓴다 (4절 7번 — `syscall.O_NOFOLLOW` 가 windows 에
없다). `cmd/enodectl` 이 `internal/enode` 를 링크하므로 `internal/merge` 가 딸려 들어간다 — Step 20 의 크로스 빌드 셋과
`enodectl.exe` 심볼 상한이 확인한다. 32비트 arm 에 새로 드는 정수는 마운트 번호 (`uint64` · statx 의 `stx_mnt_id`) 하나다.

---

## 2. 고치는 파일 — 행렬

「행렬」은 `unit-of-work-file-matrix.md` 1절이고 ⑦ 열이 bake 유닛이 고치는 칸이다. 「겹치는 유닛」은 같은 파일을 고치는 다른 유닛과
병합 순서다 (① contract-grammar · ② step-phase · ③ finalize · ④ trash · ⑤ merge-rules · ⑥ lower-state · ⑧ checkpoint).

| 파일 | 새 · 고침 | 행렬 | 겹치는 유닛 | 무엇 |
|---|---|---|---|---|
| `internal/enode/bake.go` | 새 | 있음 (굽기 새 파일 칸) | — | `Baker` · `StartBaker` · `Wait` · `heldBake` · 몸통 `abandon` · `draftBuilds` · `startMerging` · `cancelMerging` · 정리 `clean` (FD 규칙 12.2 의 표) |
| `internal/enode/bake_build.go` | 새 | 있음 | — | `runBuildStep` — 굽기 잠금 · 명령 · IR 대조 · pinned · 닫기 · 초안 · pending |
| `internal/enode/bake_merge.go` | 새 | 있음 | — | `runMergeStep` — claim 확인 · 기다림 · 그물 · helper 두 번 · metadata · committed |
| `internal/enode/bake_resume.go` | 새 | 있음 | — | `resume` · `onStale` |
| `internal/enode/bakerule.go` | 새 | 있음 | — | 순수 함수 — IR 판정과 문장 · 대조 출력 풀기 · `Draft` 타입 · 결과 칸 옮기기 · 대기 로그 · last_attempt · 대기 자리 모양 · url 비밀번호 · merge claim 표 · merge-helper 의 주고받는 타입과 응답을 오류로 되돌리기 (`helperErrorOf`) |
| `internal/enode/bakefile.go` | 새 | 있음 | — | 대기 자리 만들기 (0700) · 초안 쓰기와 읽기 · pinned 파일의 sha256 (`readPinned` · 4절 30번) · 대기 자리를 그 scratch 의 trash 로 (모든 플랫폼 · 4절 7번) |
| `internal/enode/mergehelper_linux.go` | 새 | 있음 (merge-helper 칸) | — | `RunMergeHelper` · 여는 argv (`mergeHelperArgv` · `mergeHelperCommand`) · 띄우고 한 줄씩 주고받기 (`callMergeHelper`) · `fileOwner` |
| `internal/enode/mergehelper_other.go` | 새 | 있음 | — | `RunMergeHelper` 가 exit 1 과 `merge-helper is supported on linux only` · `callMergeHelper` 가 같은 문장의 오류 · `fileOwner` 는 모른다 (false — 초안 읽기가 거절한다) |
| `internal/enode/claim.go` | 고침 | 있음 | ③ ④ ⑥ (병합됨) · ⑧ (뒤) | `Step` 의 칸 넷 · `Result` 의 칸 둘 · `Worker.Bake` · `execute` 의 분기 · `afterExit` 를 나눈 `closeOut` (4절 3번) |
| `internal/enode/runc_overlay_linux.go` | 고침 | 있음 | ③ ④ ⑥ (병합됨) · ⑧ (뒤) | `Close` 가 `Keep.Upper` 를 본다 (RENAME_NOREPLACE · 4절 6번) · `Open` 이 helper 응답을 못 받은 길의 stderr 꼬리를 helper 를 기다린 뒤 싣는다 (4절 31번) · `Finalize` 가 diff 를 세션 안에서 만들고 helper 의 `finalize` 는 Diff 를 끈다 (4절 34번 — 이 부분은 finalize 유닛 ③ 의 수확을 고친다 · 행렬 밖 기록에 따로) · 주석 |
| `cmd/enode/main.go` | 고침 | 있음 | ③ ④ ⑥ (병합됨) · ⑧ (뒤) | 부르는 줄 넷 — merge-helper 입구 · `StartBaker` · `Worker` 의 칸 · `baker.Wait()` |
| `scripts/finalize-bake/bake-common.sh` · `slice-5.sh` · `slice-6.sh` · `slice-7.sh` · `slice-8.sh` | 새 | 있음 (조각 스크립트 칸) | ③ ④ 는 다른 파일 (slice-1 · 2 · 4) | 조각 5 ~ 8 · 운영 lower 확인 (4절 26번) |
| `internal/contract/result.go` | 고침 | 밖 (FD 흐름 10절) | ② ⑥ 이 고친 파일 | `ReasonIRMismatch` · `BuildManifest.HeadTags` · `IR` 칸의 주석 |
| `internal/store/claim.go` | 고침 | 밖 · Mediator (FD 흐름 10절) | ② · ⑥ 이 같은 줄을 고쳤다 | reason 어휘 목록 (:962 ~ :963) 에 `contract.ReasonIRMismatch` |
| `internal/merge/merge.go` · `decide.go` · `merge_linux.go` | 고침 | 밖 (FD 흐름 10절) | ⑤ | `CheckMount` · `checkMounts` · `resolve` 가 `STATX_MNT_ID` 를 읽는다 (4절 17번) |
| `internal/enode/finalize.go` | 고침 | 밖 (`contractStep` 은 FD 흐름 10절 · `settleIn` 한 칸은 이 계획이 더했다) | ③ ④ | `contractStep` 이 굽기 칸 넷을 옮긴다 · `settleIn` 에 `sealErr` 한 칸 (4절 4번 — finalize FD 3.2 의 settle 표에 한 줄이 는다) |
| `internal/enode/lowerguard.go` | 고침 | 밖 (FD 흐름 10절) | ⑥ | `OnStale` · `Dir` · `sourcesLocked` 의 부르는 줄 (4절 18번) |
| `internal/enode/runtime.go` | 고침 (주석 셋) | 밖 (4절 29 · 34번) | ③ ④ ⑥ · ⑧ | `Keep` 의 주석 — 「아직 늘 비어 있고」가 거짓이 된다 · `FinalizeSpec.Diff` 의 주석 — runc-overlay 는 세션 안에서 만든다 (`finalizeLocal` 은 안 고친다) |
| `internal/enode/diff.go` | 고침 | 밖 (4절 34번 · CG 물음 5 답) | ③ (이 회차에는 고친 유닛 없음 · 앞 회차의 finalize 수확) | `repoDiff` 안의 스크립트 둘을 패키지 상수로 · `sessionDiffScript` · `sessionDiffError` · `placeWorkspaceDiff` · `diffLimit`. native 의 `workspaceDiff` 는 그대로 |
| `internal/lower/lower.go` | 고침 (주석 한 줄) | 밖 (4절 29번) | ⑥ | `State.PendingUpper` 의 주석 — building 때부터 적힌다 |
| `internal/enode/workspace.go` | 고침 | 밖 (4절 33번 · CG 물음 3 답 A) | — (이 회차에 고친 유닛 없음) | `Prepare` 가 `WorkspaceWrites(w.Runtime)` 를 본다 — isolated 면 저장소 확인만 하고 호스트 reset · clean · `repo forall` 을 안 돌린다 · `PrepClean` |
| 시험 (새) | 새 | 시험 | — | `internal/enode`: `bakerule_test.go` · `bake_test.go` (모든 플랫폼) · `bakerule_linux_test.go` (호스트 셸의 대조 한 줄) · `bake_fake_linux_test.go` (공용 틀) · `bake_linux_test.go` · `bake_build_linux_test.go` · `bake_merge_linux_test.go` · `bake_resume_linux_test.go` · `bakefile_linux_test.go` · `mergehelper_linux_test.go` · `workspace_linux_test.go` (격리 노드의 Prepare · PATH 표지 · 4절 33번) · `bake_integration_test.go` (`integration` 태그) / `internal/api/bake_test.go` (시험 DB) |
| 시험 (고침) | 고침 | 시험 | — | `internal/enode/lowerguard_linux_test.go` · `runc_overlay_linux_test.go` · `runc_overlay_integration_test.go` / `internal/merge/decide_test.go` · `merge_linux_test.go` · `merge_integration_test.go` / `internal/contract/result_test.go` / `internal/store/exited_test.go` / `internal/api/api_test.go` / `cmd/enode/main_test.go` |

**행렬 밖은 열하나다** (시험 빼고) — FD 가 적은 일곱 (result.go · store/claim.go · merge 셋 · finalize.go · lowerguard.go) 과
이 계획이 더한 주석 둘 (runtime.go · lower.go) 과 물음 3 의 답이 더한 workspace.go 와 물음 5 의 답이 더한 diff.go. 행렬 안의
runc_overlay_linux.go 도 finalize 부분은 finalize 유닛의 수확을 고치는 것이라 행렬 밖 기록에 따로 적는다. unit-of-work.md 7절의 행렬 밖 줄과
FD 흐름 10절에 둘 다 적었다. 그 diff 를 code-summary 에 따로 적는다 (CONVENTIONS 3.5 의 「행렬 밖 파일을 만진 diff」).
**finalize.go 의 `settleIn.sealErr` 한 칸은 FD 가 허락한 것이 아니다** — FD 흐름 10절이 finalize.go 에 적은 것은 `contractStep` 하나다.
이 계획이 더한 한 칸이고 finalize FD 3.2 의 settle 표에 한 줄이 는다. 행렬 밖 기록에 그렇게 따로 적는다.

**이 계획 단계가 고친 회차 문서** (물음 넷의 답 · Step 22 의 커밋이 함께 싣는다) — FD `business-rules.md` 5 · 7 · 16.1절 (merge 의 업로드
예산 · CG 물음 1 답 A) · FD `business-logic-model.md` 3 · 7 · 10 · 13.1절 (같은 답 · 물음 3 의 답과 ADR-072 결정 3 · 물음 5 의 답과 굽기
전부터 있던 구멍) · `unit-of-work.md` 7절 (행렬 밖 workspace.go · diff.go · runc_overlay_linux.go 의 finalize). FD 계획 (`bake-functional-design-plan.md:303`) 에도 「merge 는 예산을 안 쓴다」 가 있으나 그 단계의 기록이라 두었다.

**여러 유닛이 고치는 파일** — 한 줄 순서라 동시에 부딪치지 않는다. 먼저 병합된 유닛 위에서 이어 고친다. `internal/api/api.go` 의 등록
줄은 이 유닛이 만지지 않는다 — 라우트 19 그대로 (조각 0). Mediator 쪽 시험 (`internal/api/bake_test.go`) 은 새 파일이라 ② 의 파일과
겹치지 않는다. 뒤의 checkpoint 가 claim.go · runc_overlay_linux.go · main.go · runtime.go 를 이어 고친다.

**확인만 하고 안 고친다** — `internal/enode/runc_overlay_other.go` (native 와 linux 밖의 `Close` 는 `Keep` 을 안 본다) · `upload.go`
(`Worker.upload` 를 그대로 부른다 — 4절 5번) · `env.go` (`commandEnv` · `harnessEnv`) · `repoid.go` (`CanonicalRepoID` · `DetectRepo` 의
차례 · 읽기만) · `runtime.go` 의 `finalizeLocal` (helper 가 Diff 를 끄므로 안 고친다 · native 는 오늘처럼 호스트 git 으로 diff — 4절 34번) ·
`finalize.go` (`finalizeSpecFor` 의 Diff 조건과 `logTail` 의 「workspace.diff was not produced」 줄 그대로) · `trash_linux.go` (기동 청소는 `enode-runc-` 로 시작하는 이름만 옮긴다 — 대기 자리 `pending/` 을 안 건드린다 · :23 · :202) ·
`advertise.go` (BeforeAdvert 를 부르는 길) · `internal/lower/*` (lower.go 의 주석 밖) · `internal/scratch/trash.go` (Move 의 이름 규칙과
없는 자리의 ENOENT) · `internal/store/verdict.go` (`Verify` 불변) · `internal/contract/bake.go` · `examples/bake.json` ·
`internal/panel/boundary_test.go` (고칠 줄 없음).

---

## 3. NFR 두 단계를 건너뛰어서 이 계획이 받는 것

유닛 정의 7절이 NFR 에 둔 둘이다 — 보안 (계약의 명령이 격리 실행 환경 안에서만 돈다) · 성능 (형제가 기다리는 시간은 합치기가 아니라
형제 Run 이 정한다). 설계는 FD 가 닫았다. 여기는 **무엇이 그것을 지키고 어떤 시험이 되돌아가는 것을 막나 · 무엇을 측정하나** 다.

**측정** — 2026-09-27 · 이 기계 (Intel N100 4코어 · 커널 6.5.11-8-pve · ext4 · uid 1000 · 특권 없음). 측정 파일은 스크래치 폴더에 두고
끝나고 지웠다. 제품 코드로 측정할 때는 `go test -overlay` 로 시험 파일 하나를 끼웠다 — 작업 트리는 바꾸지 않았다. SunnyVM 은 ssh 로
읽기만 했다.

```text
   측정 1  BenchmarkBeforeAdvert (지금 코드 · 3회 · -benchtime 2s)                         26.7 ~ 29.8 µs  (lower-state 때 24 ~ 26 µs)
   측정 2  TryBake — 다른 Dir 이 bake.lock 을 쥔 채 (주인이 살아 있다)                        7.7 ~ 8.4 µs    (open · flock · close)
           TryBake — 비어 있을 때 잡고 놓기                                                8.0 ~ 8.2 µs
           ReadState + WriteState(committed) — 임시 파일 · fsync · rename · 디렉터리 fsync  1.44 ~ 1.54 ms
   측정 3  IR 대조의 셸 한 줄 (4절 8번 · 호스트의 dash · git 2.39) — 다섯 모양               모두 FD 규칙 4절의 판정대로
             git 모양 · init + fetch (origin 없음) · detached    head · tagged · 태그 둘 · url 빈 값 · branch HEAD
             git 모양 · origin 있음 · 브랜치                      url 그대로 (비밀번호가 든 채 — 초안에 쓸 때 지운다)
             뿌리에 .git 도 .repo 도 없음                         mode=none
             repo 모양 (.repo/manifests)                          clone 이 태그를 받아 온다 — 대조가 된다
             .git 이 깨진 파일                                    exit 128 · stderr 마지막 줄 fatal: invalid gitfile format
           한 번 도는 데 평균 10 ms (20회) — 세션 안이면 runc 의 exec 가 더 든다
   측정 4  GOOS=windows go doc syscall O_NOFOLLOW                                         no symbol — 초안 읽기를 모든 플랫폼 방법으로 (4절 7번)
   측정 5  go test -coverprofile (패키지 셋 · CI 의 -coverpkg 없이)                          internal/enode 3,429/4,090 = 83.8% ·
                                                                                          cmd/enode 194/241 = 80.5% · internal/merge 303/340 = 89.1%
           (CI 의 명령으로는 lower-state 뒤 internal/enode 84.5% · cmd/enode 80.5% — Step 1 이 다시 측정한다)
   측정 6  기준선 흔들림 — internal/enode 를 여덟 번                                        두 번 빨갛다 (나무는 main + FD 문서뿐).  이름을 잡은 한 번은
                                                                                          TestRuncOverlayOpenIncludesHelperStderr
           그 시험 혼자 -count=300 (QA 검수 뒤 다시 측정 · 2026-09-27)                        11 번 빨갛다 (QA 가 측정한 14 번 · 7 번과 같은 자리).
                                                                                          부하 탓이 아니다 — 원인 둘.  가짜 helper 의 eof 갈래가
                                                                                          os.Exit 없이 돌아와 시험 바이너리가 stdout 에 PASS 를 찍는다
                                                                                          (runc_overlay_linux_test.go:27 ~ :30) · Open 의 receive 가
                                                                                          helper 가 끝나기 전에 stderr 버퍼를 읽는다
                                                                                          (runc_overlay_linux.go:293 ~ :301).  CI 의 둘째 실행
                                                                                          (-coverpkg) 에서는 그 stdout 뒤에 coverage 줄까지 붙는다.
                                                                                          lower-state code-summary 5절 끝의 「부하 흔들림」 은 틀린
                                                                                          규정이다 — 4절 31번이 원인을 고친다
   SunnyVM 2026-09-27T13:24:22Z 닿는다 · 커널 7.0.0-31 · 8코어 · /bin/sh 는 dash · git 2.43 · runc 1.3.4 ·
           kernel.apparmor_restrict_unprivileged_userns 0 (unshare 가 된다) · subuid 한 줄 · ext4 · 떠 있는 enode 둘은
           ~/bin/enode (2026-09-22T22:36+0900 판 — lower-state 전) · ~/lower-it-* 남은 것 0.  노드의 설정 · 상태 자리 · /srv/yocto 는 안 읽었다
   이 기계  /bin/sh 는 dash · git 2.39.2 · runc 1.1.5 · subuid 세 줄 · unshare --map-auto 가 막힌다 (lower-state code-summary 7절) ·
           golangci-lint 는 ~/go/bin · shellcheck 없음
```

### 3.1 보안 — 계약의 명령은 격리 실행 환경 안에서만 돈다

**요구** — 팩 보안 표의 「계약의 명령」 줄 (`requirements.md` 5.3 — `sync` 와 `builds[].command` 는 격리 runtime 안에서 돈다 · 계약이 노드
설정이나 host 경로를 지정하지 못한다). 세션 (`runcOverlaySession.Run` · `runc_overlay_linux.go:346`) 이 helper 를 거쳐 runc 컨테이너 안에서
돈다는 것은 앞 유닛 (실행 환경 · finalize) 이 짓고 확인했으며 SunnyVM 의 integration 시험이 다시 봤다. 이 유닛은 **굽기의 명령이 모두 그 길로만
가고 밖으로 새지 않는 것** 과 **계약이 lower 에 남긴 것을 호스트가 실행하지 않는 것** 을 지킨다. 뒤의 것은 굽기가 처음 만든 길이다 —
굽기 전에는 runc-overlay 노드에서 계약이 쓴 것이 모두 버려졌고, 굽기 뒤에는 계약의 sync 가 쓴 `.git/config` · `.repo/` 가 lower 에 남는다
(CG 물음 3 답 A · 4절 33번).

```text
   도는 것 (누가 지은 글자)                 argv                                          어디서                 막는 시험 (Step)
   sync · builds[].command (계약)           ["bash", "-c", <계약의 문자열>]                  session.Run            TestBakeBuild_CommandsGoThroughTheSession (16)
   bash 확인 (노드의 고정 한 줄)             ["sh", "-c", "command -v bash >/dev/null"]     session.Run            같은 시험 — argv 의 차례와 모양 전부
   IR 대조 (노드의 고정 한 줄)               ["sh", "-c", <probeScript>]                    session.Run            TestIRProbe_TheIRGoesOnlyThroughTheEnvironment (16)
                                            IR 은 환경 변수 ENODE_IR 로만 — 셸 글자에 넣지 않는다                       TestProbeScript_OnAHostShell (16)
   pinned (노드의 고정 한 줄)                ["bash", "-c", "repo manifest -r -o .enode-manifest.xml"]  session.Run  CommandsGoThroughTheSession (repo 모양)
   호스트가 읽는 계약의 산출물               pinned 파일 (builds 가 바꿀 수 있다 — pinned 는 9 · builds 는 10)          호스트 (노드 uid)     TestReadPinned (6) ·
     — pinned 파일의 sha256                 닫은 뒤 대기 자리에서 Lstat 이 보통 파일 · 16 MiB 안일 때만 열고              TestBakeBuild_APinnedFileThatIsNotRegularFails (11)
                                            SameFile · LimitReader.  symlink · FIFO · 장치 · 없음은 cannot pin the manifest
                                            (4절 30번)
   호스트가 lower 에 돌리는 명령              isolated 노드의 Prepare 는 저장소 확인 (git config --get · rev-parse ·      TestPrepare_AnIsolatedNodeRunsNoHostGitWrite (16) ·
     (workspace.repo 를 적은 run · agent 단계)  읽기만) 만 하고 reset · clean · repo forall 을 안 돌린다 · PrepClean 은 새    TestPrepare_AnIsolatedNodeDoesNotRunTheLowersFsmonitor (16) ·
                                            upper 가 준다.  native 는 오늘처럼 reset · clean (계약의 명령이 이미 호스트에서  TestPrepare_ANativeNodeStillResetsAndCleans (16) ·
                                            돈다).  저장소가 어긋나면 그대로 거절 (4절 33번).  그 밖의 자리는 아래 표       TestPrepare_AnIsolatedNodeStillRefusesAnotherRepo (16)
   Finalize 의 workspace.diff (isolated ·     ["sh", "-c", sessionDiffScript] — 준비된 rootfs 의 git · repo · 출력은 Run 의     TestRuncOverlayFinalizeMakesTheDiffInTheSession (7) ·
     effect edit 이고 workspace 를 적은 단계)  stdout 흐름으로만 호스트에 온다 · 호스트가 $OUT 에 새 파일로 놓는다 (4절 34번)  TestRuncOverlayHelperFinalizeRunsNoHostGit (7) ·
                                            helper 는 Diff 를 끈다 · native 는 오늘처럼 호스트 git                         TestRuncOverlayHelperFinalizeDoesNotRunTheTreesConfig (7) ·
                                                                                                              TestSessionDiffScript_MatchesTheHostDiff (7) ·
                                                                                                              TestRuncOverlayFinalizeDiffInTheSessionIntegration (18)
   환경                                     commandEnv + 계약의 env 이름 + OUT · IN · ENODE_IR    세션 환경            TestBakeBuild_UnnamedHostVariablesDoNotReachTheCommands (16)
                                            계약이 이름으로 부르지 않은 호스트 환경 변수는 닿지 않는다.  계약이 env 로
                                            부른 이름은 닿는다 — 노드 비밀의 이름 (ENODE_TOKEN 등) 도 · 잔여 (8절 ·
                                            CG 물음 2 답)
   작업 폴더                                session.Paths().Dir (워크스페이스 자리)                                     CommandsGoThroughTheSession
   호스트 프로세스                           굽기 흐름의 파일 여섯은 exec. · os.StartProcess · syscall. · NativeRuntime ·   TestBakeSources_StartNoHostProcess (16)
                                            DetectRepo · detectRepoManifest · gitIn · child 를 부르지 않는다 (go/ast 로
                                            selector 를 본다).  예외 mergehelper_linux.go — unshare 로 merge-helper 만 연다
                                            build · merge · 재개를 가짜 런타임으로 돌려 PATH 의 bash · sh · git · repo 가      TestBakeFlows_RunNoHostProgram (16)
                                            한 번도 불리지 않는다 — 같은 패키지 함수와 claim.go · finalize.go 의 길까지
   제자리에 쓰는 노드 (native)               runBuildStep · runMergeStep 의 첫 줄이 WorkspaceWrites(w.Runtime) 가          TestBakeBuild_ANodeWithoutAnUpperRefuses (11 · 16) —
                                            isolated 가 아니면 거절한다 — Bake 가 있어도 · 세션을 열지 않고 명령을 안       Bake nil · Bake 가 있고 Runtime nil ·
                                            돌린다 · Runtime 이 nil 이어도 NativeRuntime 으로 떨어뜨리지 않는다 (4절 2번)   Bake 가 있고 NativeRuntime{}
   merge-helper                             unshare --user --map-root-user --map-auto --fork --kill-child -- <enode>    TestMergeHelperArgv (10) — 바꿔 끼우기 전의
                                            merge-helper  (--mount 없음 · namespace 안의 root · 권한을 내려놓지 않는다)    제품 기본값이 unshare 로 시작한다
   merge-helper 가 받는 경로                 upper 는 state.json 의 pending_upper (노드가 적은 값) · 모양이 틀리면        TestPendingShape (5) ·
                                            옮기지도 합치지도 않는다 (재개는 실패 · 정리는 경로를 로그에 남기고 상태만      TestResume_AnOddPendingPathIsNotMerged (13) ·
                                            committed) · lower 는 이 노드의 lower 루트 · trash 는 pending_upper 에서        TestBakerClean_AnOddPendingPathIsLeft (9)
                                            네 단계 위의 scratch 의 trash — 계약은 어느 것도 못 정한다
   합치기의 같은 마운트                       시작 전 확인이 bind 별칭을 거절한다 (4절 17번)                               TestCheckMounts (3) · TestBindAliasPreflightIntegration (18)
   초안 · metadata                           초안 0600 · pending/ 과 <키> 0700 · 초안을 읽을 때 주인이 노드 uid (4절 32번) ·  TestDraft_IsPrivate (6) · TestStripPassword (5)
                                            url 의 비밀번호를 지운다 · metadata 0644 (lower-state 의 WriteMetadata)
   build 의 $OUT                             명령이 쓴 것은 올리지 않는다 — 성공이어도 실패여도 노드의 manifest 하나      TestBakeBuild_TheCommandsOutIsNotUploaded (16)
                                            (4절 5번)
   단계 로그 · 진행 청크                     host 경로 (대기 자리 · scratch) 를 쓰지 않는다 — 노드 로그에만.  그물 줄은     TestBakeBuild_TheStepLogCarriesNoHostPath (11) ·
                                            예외 (FD 규칙 16.2 끝)                                                     TestMergeStep_TheStepLogCarriesNoHostPath (12)
```

- **왜 source 를 훑는 시험인가** — 호스트에서 `exec.Command("bash", "-c", sync)` 를 부르는 되돌림은 가짜 세션 시험이 못 잡는다 (가짜
  세션은 불리지 않았을 뿐이다). import 만 보면 가장 그럴듯한 되돌림을 못 잡는다 — `os.StartProcess` 는 os 패키지이고, 같은 패키지의
  `NativeRuntime` (`runtime.go:183` 의 exec) · `gitIn` · `DetectRepo` (`repoid.go:77` ~ `:116` — 호스트에서 git 을 돈다) 는 import 가 없다.
  그래서 굽기 흐름의 파일 여섯 (`bake.go` · `bake_build.go` · `bake_merge.go` · `bake_resume.go` · `bakerule.go` · `bakefile.go`) 을
  `go/parser` 로 읽고 `go/ast` 로 selector 와 부르는 식을 훑어 `exec.` · `os.StartProcess` · `syscall.` · `NativeRuntime` · `DetectRepo` ·
  `detectRepoManifest` · `gitIn` · `child` 를 쓰는 자리가 없는지 본다 (import 의 `os/exec` · `syscall` 도). 값은 나중에 누가 그 이름을 부르게 했을 때
  빨개지는 데 있다 (경계 시험의 논리와 같다)
- **행동 시험을 더하는 까닭** — `closeOut` 과 `settleIn` 이 들어가는 `claim.go` · `finalize.go` 는 위 검사 밖이다 (`finalize.go:8` 은 이미
  `os/exec` 를 가져온다). `TestBakeFlows_RunNoHostProgram` 은 PATH 를 임시 폴더로 바꾸고 그 자리에 bash · sh · git · repo 라는 이름으로
  표지 파일을 쓰는 스크립트를 둔 뒤, build (git 모양 · repo 모양) · merge · 재개를 가짜 런타임과 namespace 없는 helper 로 돌려 표지
  파일이 하나도 없는지 본다. `repoIDOf` 는 세션 안 대조의 출력 (url · branch · mode) 만 쓰고 `DetectRepo` 를 부르지 않는다 — 부르면
  호스트 git 이 pending upper 에서 돈다
- **IR 을 셸 글자에 넣지 않는 까닭** — contract-grammar 의 `IRProblem` 이 이미 글자를 좁히지만 (`bake.go`), 셸 글자에 이어 붙이지 않으면
  글자 규칙이 느슨해져도 주입이 생기지 않는다. 시험은 받은 argv 어디에도 IR 글자가 없고 환경에 `ENODE_IR=<IR>` 이 있는지 본다
- **integration** — 진짜 세션의 upper 가 `Close(Keep{Upper})` 로 옮겨지고 파일 주인이 노드 uid 인지 (`TestRuncOverlayCloseKeepsTheUpperIntegration`
  · Step 18). rootfs 안의 git · bash · safe.directory 는 조각 6 이 처음 확인한다 (FD 계획 2.6 · 되물음 3 답 A)

**호스트 git · repo 가 도는 자리** (`fb86ad2` 에서 `exec.Command` · `gitIn` · `gitOutName` · `run(` · `DetectRepo(` 를 모두 찾았다 · CG 물음 3 답 A).
「실행 위험」 은 계약이 쓴 파일 (`.git/config` 의 core.fsmonitor · filter 와 diff 드라이버 · `.repo/repo` 의 런처 코드) 을 그 명령이 실행할 수 있나다.
native 노드는 계약의 명령이 이미 호스트에서 돌므로 새로 넓어지는 길이 아니다 — 표는 isolated (runc-overlay) 노드를 본다.

```text
   자리                                   도는 것                              어디 · 어느 트리                    lower 에 쓰기 · 실행 위험         이 계획
   workspace.go:94 ~ :118                 git reset --hard · git clean -df ·     호스트 · lower 뿌리                 쓴다 (형제가 공유를 쥔 lower 의    4절 33번 — isolated 에서
   (Prepare 의 reset · clean)             repo forall -c ...                   (Local.Workspace · environment.go:26)  추적 파일을 되돌린다) · 있다      안 돈다
                                                                                                            (QA 측정 — reset · clean 에서
                                                                                                            core.fsmonitor 가 실행됐다 ·
                                                                                                            repo 런처는 .repo/repo 를 실행)
   repoid.go:77 ~ :116 (DetectRepo —       git config --get remote.origin.url ·   호스트 · lower 뿌리                 없다 · 없다 (읽기만 · QA 측정       남긴다
   Prepare 의 저장소 확인)                  (repo 모양) 같은 것과 rev-parse          (repo 모양은 .repo/manifests)        git 2.39.2 — fsmonitor · hooks 를
                                          --abbrev-ref HEAD                                                    안 부른다)
   detect.go:197 ~ :209 (광고 탐지 repoFP)  DetectRepo 그대로                      호스트 · lower 뿌리 · 탐지 주기마다   위와 같다                        그대로 — Prepare 의 확인을
                                                                                                                                         남겨도 새 길이 아니다
   diff.go:51 ~ :150 (Finalize 의          git read-tree · add -N . · diff        runtime helper — 호스트 바이너리 ·   없다 (쓰기는 upper · 인덱스는       4절 34번 — 세션 안 (rootfs 의
   workspace.diff · effect edit 이고       --binary (임시 인덱스) · repo forall   user · mount namespace 의 root =     임시 파일) · 있다 (그 단계가 쓴     git · repo) 으로 옮긴다 ·
   workspace 를 적은 단계 · finalize.go:72)  -c <script>                          노드 uid · merged view               것과 굽기 뒤 lower 에 남은           helper 는 Diff 를 끈다 ·
                                                                               (runc_overlay_linux.go:1086 ~ :1090)  .git/config · .repo/repo · 진행자     native 는 그대로
                                                                                                                    측정 — core.fsmonitor 넷 · filter
                                                                                                                    clean 둘이 실행됐다)
   identity.go:121                        git config --global --get user.email   호스트 · 데몬의 작업 폴더           없다 · 없다 (전역 설정만)          그대로
   claude.go:138 · :183                   하네스 --version · auth status         호스트 · lower 아님                 없다                             그대로
   runc_overlay_linux.go:1014 · :1034 · :1066  runc run · delete · kill          노드가 지은 bundle                  없다                             그대로
   trash_linux.go:135 · 새 mergehelper     trash-helper · merge-helper           namespace · 지우기와 rename 만      트리 안의 것을 실행하지 않는다       그대로
   changed.go · collect.go (collect ·     git 없음 — Lstat 로 걷기와 복사 ·       helper · merged view · upper       없다 (두 파일은 os/exec 를           그대로 — 34번 뒤 helper 가
   지목 경로 stat · 명시 훑기 walkUpper)    훑기는 upper 를 걷기만                                                   가져오지 않는다)                    merged 에서 하는 일은 이것뿐
   굽기의 bash 확인 · IR 대조 · pinned      session.Run                           컨테이너 안                        호스트가 아니다                    3.1 표
```

- **merge 쪽** — merge 단계와 재개는 호스트 git 을 부르지 않는다. `repoIDOf` 는 세션 안 대조의 출력만 쓴다 (Step 5 · 16)
- **표 밖으로 가지 않게** — Step 16 의 `TestBakeSources_StartNoHostProcess` (굽기 흐름 파일 여섯) 와 `TestBakeFlows_RunNoHostProgram` · 4절
  33번의 PATH 표지 시험 · 4절 34번의 helper 시험 둘 (Step 7) 이 되돌림을 잡는다
- **이 구멍은 굽기 전부터 있었다** — isolated 노드의 edit 단계 (agent 포함) 가 upper 에 `.git/config` 를 쓰면 Finalize 의 diff 때 helper 의 호스트
  git 이 컨테이너 밖에서 노드 uid 로 그 설정을 실행했다 (진행자 측정 · git 2.39.2 · audit 2026-09-29T14:13:46Z). 굽기는 lower 에 남은 설정을 뒤
  Run 의 diff 가 읽게 해 그 길을 넓힌다. repo 모양은 런처가 트리의 `.repo/repo` 를 실행하므로 git 설정을 끄는 것으로는 못 막는다 — 그래서 세션
  안으로 옮긴다 (CG 물음 5 답)
- **세션 안 git 도 설정을 실행한다** — 세션 안의 diff (4절 34번) 와 IR 대조 · pinned 의 git · repo 는 계약과 그 단계가 쓴 `.git/config` ·
  `.repo/repo` 를 실행할 수 있다. 격리 실행 환경 안이라 계약의 명령과 같은 등급이다 — 계약이 sync · builds 로 이미 그 자리에서 무엇이든 돌릴
  수 있다. 지키는 선은 「호스트 쪽 (helper · 데몬) 이 그 파일을 실행하지 않는다」 다

### 3.2 성능 — 형제가 기다리는 시간은 형제 Run 이 정한다

**형제가 새 일을 못 받는 구간** 은 굽는 노드가 pending 을 쓴 때부터 committed 를 쓴 뒤의 다음 광고까지다 (굽기 drain · `lowerguard.go:212` ~
`:216`). 그 구간을 이루는 것과 크기다.

```text
   조각                                         누가 정하나                              크기 (측정 · 실측)
   형제의 도는 Run 이 끝나기까지                   형제 Run                                  빌드 하나 30 ~ 80 분 (ADR-077 §12 — 1,892 초 · 4,760 초)
   drain 응답 두 번 뒤 공유를 놓기까지              광고 주기                                 두 주기 안 (주기 기본 60초 · 최대 120초 · lower-state 답 1)
   배타를 잡는 데까지                              lower 의 pollEvery                        1초 안 — 1초마다 다시 건다 (lower-state 의 시험은 간격을 2 ms 로 줄여 1.8 ms)
   merge 가 배타를 쥐는 시간                        합치기                                   고정 비용 (그물 5.2 ms · helper 여는 데 약 5 ms 두 번 · state 쓰기 둘
                                                                                           (merging · committed) 과 metadata 쓰기 — 각 1.5 ms 안팎) + Preflight + Apply.
                                                                                           하루치 Preflight 0.23 초 (SunnyVM) · 0.63 초 (이 기계) ·
                                                                                           Apply 1.11 초 (SunnyVM) · 3.80 초 (이 기계) — merge-rules code-summary
   drain 이 풀리는 다음 광고                         광고 주기                                 한 주기 안
```

합치기가 드는 몫은 초 단위이고 나머지는 형제 Run 과 광고 주기다. 이 유닛이 새로 들이는 세 가지가 형제의 claim 과 광고를 막지 않는지가
확인할 자리다.

```text
   새로 드는 것            형제에게 닿는 길                                         막는가                          확인 (Step)
   굽기 잠금               형제는 bake.lock 을 기다리지 않는다.  광고 주기의 onStale 이      광고 고루틴 밖 · 8 µs 안팎        BenchmarkOnStaleWhileTheOwnerLives (17)
                          주인이 살아 있는 동안 광고마다 TryBake 한 번을 헛되이 한다         (측정 2)
   광고 주기의 낡은 상태 정리  BeforeAdvert 는 state 를 이미 읽는다 — committed 가 아니면     광고는 기다리지 않는다             TestOnStale_ReturnsAtOnce (13) ·
                          OnStale 을 곧바로 부르고, Baker 는 표지만 보고 정리를 배경       정리 한 번 1.5 ms 안팎 (측정 2)     TestLowerGuard_OnStaleSeesStaleStates (8) ·
                          (Baker.wg) 으로 연다 (4절 18번).  정리 (옮기기 + committed)                                        BenchmarkBeforeAdvert (committed · pending —
                          는 따로 돈다                                                                                      f 는 부른 수만 세는 가짜) (17) ·
                                                                                                                 BenchmarkStaleCleanup (17)
                          정리를 연 광고는 이미 drain 을 싣고 나가고 다음 광고에서 풀린다   두 광고 주기 안 (기본 120초)       TestOnStale_CleansAPendingWhoseOwnerDied (13) · 조각 8
   끊긴 합치기의 재개        merging 이면 형제가 drain 이고 공유를 새로 안 잡는다 — 재개의     재개 시간 = 고정 비용 + Preflight   BenchmarkMergeStepFixedCost (17) ·
                          배타는 보통 곧바로 잡힌다                                      + 남은 Apply                      integration 의 끊고 잇기 (18)
                          실패하면 그 노드는 10분 뒤 · 다른 노드는 자기 광고에서           하루치 upper 재시도 Preflight 만으로  (FD 규칙 13절 · 계산)
                                                                                     시간당 1.4 ~ 3.8 초 · 노드 N 개면 N 배
   merge 의 대기 로그        단계 로그 · 진행 청크                                       4시간에 바뀜을 빼고 48 줄           TestWaitLog (5)
```

- **측정 자리** (Step 17) — 벤치마크 넷을 이 기계에서 한 번 돌려 code-summary 에 적는다. 기본 `go test` 는 안 돌린다. SunnyVM 이 켜져
  있으면 integration 시험 (Step 18) 이 끊고 잇기의 재개 시간을 적는다
- **값이 위 추정보다 한 자리 이상 크면** 원인을 찾아 적는다. 값을 맞추려고 규칙을 바꾸지 않는다
- **merge 는 형제를 기다리는 동안 형제를 더 붙잡지 않는다** — 배타는 1초마다 다시 걸 뿐이고 (`lock_linux.go:163` ~ `:180`) 형제의 공유와
  광고에 닿지 않는다. 형제를 붙잡는 것은 pending 의 drain 이고, 그 drain 은 FD 가 정한 뜻 그대로다 (팩 결정 3-10 — pending 이나 merging 이면
  형제가 graceful drain 을 싣는다)

---

## 4. 이 계획이 정한 것

### 4.1 FD 에 적히지 않은 자리

정본과 FD 의 결정을 바꾸는 자리는 여기서 정하지 않고 10절 물음으로 올렸고, 물음 다섯의 답을 받아 15 · 33 · 34번과 3.1 · 7.1 · 8절에 옮겼다.
답이 거짓으로 만든 FD 문장은 FD 파일에서 고쳤다 (2절 「이 계획 단계가 고친 회차 문서」). **FD 의 글자와 다른 수단을 고른 자리는 여섯이다** — 2번
(겉면 · FD 엔티티 2절의 세 인자 겉면과 다르다) · 4번 (finalize FD 3.2 의 settle 표에 한 줄이 는다) · 5번 (올리는 폴더 · FD 규칙 15절 안의
두 문장을 목적 쪽으로 푼다) · 7번 (초안 읽기의 방법) · 18번 (FD 흐름 6절은 `go OnStale(st)` 로 적었다 — guard 는 곧바로 부르고 Baker 가 wg 로 연다) ·
29번 (FD 흐름 10절은 runtime.go 가 안 바뀐다고 적었다 — 주석만 고친다). 각 근거 칸에 그 대조를 적었다. 15번 (merge 의 업로드 예산) 은
FD 를 답에 맞춰 고쳤으므로 이 목록에서 뺐고, 33번 (격리 노드의 Prepare) 과 34번 (격리 노드의 Finalize diff) 은 FD 가 다루지 않은 파일이다.

| 번호 | 자리 | 정한 것 | 근거 |
|---|---|---|---|
| 1 | 파일 배치 | FD 엔티티 머리 표의 새 파일 다섯 (`bake.go` · `bake_build.go` · `bake_merge.go` · `bake_resume.go` · `bakerule.go`) 과 merge-helper 둘 그대로. 초안 파일 일만 `bakefile.go` 로 뗀다 — 모든 플랫폼 (7번) | FD 엔티티 머리가 지키라 한 셋 (규칙은 순수 함수 · unshare 만 `_linux.go` · main.go 는 부르는 줄만) 을 지킨다. `_linux.go` 에는 unshare 를 여는 코드와 함께 linux 에만 있는 것 (`callMergeHelper` · `fileOwner`) 을 두고 `_other.go` 가 짝을 둔다 (1절 크로스 빌드) |
| 2 | 분기 자리와 겉면 | `execute` 의 임대 감시 고루틴 (`claim.go:655` ~ `:672`) 뒤 · 빈 argv 확인 (`:676`) 앞에 `kind` 가 `contract.KindBuild.String()` 이면 `w.runBuildStep(runCtx, ctx, step, dir, in, out, log)` · `KindMerge.String()` 이면 `w.runMergeStep(runCtx, ctx, step, log)` 하고 돌아간다. 겉면은 `runAgentStep` 과 같은 모양이다. 두 함수의 첫 줄은 `WorkspaceWrites(w.Runtime)` 가 isolated 가 아니면 FD 규칙 1절의 거절 문장으로 끝낸다 — `Worker.Bake` 가 있어도 본다. 세션은 `w.Runtime` 으로만 연다 — Runtime 이 nil 이어도 `NativeRuntime` 으로 떨어뜨리지 않는다 (`claim.go:681` ~ `:684` 의 모양을 옮기지 않는다) | `component-methods.md` 4.2 와 FD 엔티티 2절의 세 인자 겉면 (ctx · step · log) 은 임대 ctx 와 워크스페이스 · `$IN` · `$OUT` 을 못 받는다. Step 22 가 그 줄을 고친다. 제품 배선에서 Baker 가 있으면 runc-overlay 다 (`main.go:161` ~ `:171` · `lowerguard.go:78` ~ `:81`) — 그 사실이 main.go 에만 있으므로 흐름이 스스로 본다. 떨어뜨리면 계약의 명령이 호스트에서 돈다 (`runtime.go:183` 의 exec) |
| 3 | `afterExit` 를 나눈다 | `afterExit` 의 겉면은 그대로 두고 안에서 `w.closeOut(..., closing{})` 를 부른다. `closing` 의 칸 셋 — `keep Keep` (Close 에 넘긴다) · `after func(deadline time.Time, closeErr, finErr error) (late bool, err error)` (Close 뒤 · 업로드 앞) · `upload string` (올릴 폴더 · 비면 `spec.Out`). 차례 — Finalize · `Close(ctx, keep)` · Close 가 끝난 시각 · `after` · `FinalizedAt` (after 가 돌아온 뒤) · 업로드 · settle. `closedLate` 는 `late` 이거나 「Close 가 끝난 시각이 마감 뒤이고 runCtx 가 살아 있음」이다. `after` 의 규칙 — closeErr 나 finErr 가 있으면 성공 쪽 대신 몸통을 부르고 `(false, nil)` (문장은 settle 이 closeErr · finErr 로 낸다). 성공 쪽 (sha256 · 초안 · pending · HoldBake · manifest) 은 pending 을 쓰기 직전에 마감을 보고, 지났으면 pending 을 안 쓰고 몸통 · `(true, nil)`. 성공 쪽의 노드 쪽 오류 (sha256 · 초안 · pending 쓰기) 는 몸통 · `(false, err)`. build 의 실패 끝 (명령 실패 · IR 어긋남 · 대조를 못 함 등) 은 `after` 가 몸통만 부르고, 업로드는 blobs false 로 단계 로그만 올린다 | 오늘 `afterExit` 는 `Close(ctx, Keep{})` 를 박아 두었고 `spec.Out` 전부를 올린다 (`claim.go:827` · `:838`). 명령 · agent 단계는 `closing{}` (after 없음 · 올릴 폴더는 `spec.Out`) 이라 오늘과 같다 — 오늘의 시험이 그대로 초록인 것으로 확인 (Step 11). 오늘 closedLate 는 Close 바로 뒤의 시각으로 정한다 (`:827` ~ `:829`) — 그 시각만으로 정하면 pending 을 건너뛴 build 가 finalize ok 인 DONE 이 되고, after 뒤의 시각만으로 정하면 마감 직전에 pending 을 쓴 build 가 finalize_timeout 이 되는데 pending 과 HoldBake 는 살아 있다. 그래서 pending 을 썼는지를 아는 `after` 가 `late` 를 말한다. FD 규칙 6절 — pending 을 쓰기 전의 노드 쪽 오류 (닫기 · 초안 · pending 쓰기) 는 FAILED 이고 pending 을 쓰지 않는다 (business-rules.md:250) |
| 4 | 예산과 pending (finalize.go) | `settleIn` 에 `sealErr error` 한 칸 — `after` 가 돌려준 `err` (초안 · pending 쓰기 · sha256 의 노드 쪽 오류) 는 finalize 칸 `error` 이고 그 문장을 그대로 error 에 잇는다. pending 을 쓰기 전에 Finalize 마감이 지났으면 `after` 가 pending 을 쓰지 않고 `late` 를 참으로 돌려주고 `closeOut` 이 그것을 `closedLate` 에 싣는다 (3번) — 오늘의 `finalize_timeout` 문구와 reason. reason 이 둘이면 settle 의 것 (FAILED) 이 이기고 `ir_mismatch` 는 last_attempt 에만 남는다 | FD 규칙 5 · 6절 (Finalize 예산이 닫기 · sha256 · 초안 · pending 쓰기를 덮는다 · 결정 32). `closeErr` 에 실으면 「runtime cleanup:」이 붙어 거짓 문장이 된다. 이 칸은 FD 흐름 10절이 finalize.go 에 허락한 `contractStep` 밖이다 — 이 계획이 더한 것이고 finalize FD 3.2 의 settle 표에 한 줄이 는다 (2절 행렬 밖 기록) |
| 5 | 올리는 폴더 | build 단계는 세션에 준 `$OUT` 을 올리지 않고 노드가 만든 임시 폴더 (`enode-bake-out-`) 에 `manifest` 하나를 써 그 폴더를 `Worker.upload` 로 올린다. merge 단계도 같은 모양 (`merged` 하나) | FD 규칙 15절 안의 두 문장 — 「명령이 `$OUT` 에 쓴 것은 올리지 않는다」 와 「굽기가 성공하지 않았으면 `$OUT/manifest` 를 지운 뒤 올린다」 — 은 세션의 `$OUT` 을 올리느냐에서 서로 어긋난다. 목적 (명령이 쓴 manifest 가 build 의 조건을 참으로 만들지 못한다 · 결정 9 · 되물음 5 답 A) 쪽으로 푼다 — 세션의 `$OUT` 을 한 번도 올리지 않으므로 지울 것도 없다. FD 의 수단과 같은 것이 아니라 두 문장의 어긋남을 푼 것이다. 정본 `execution-environment.md:610` ~ `:612` (「process 가 직접 `$OUT` 에 쓴 파일은 자동 수확한다」) 와 다르다 — 정본 되돌림 (Step 22). `upload.go` 를 안 고친다 |
| 6 | `Close(Keep{Upper})` | helper 가 끝난 뒤 (close 응답 뒤가 아니라 `cmd.Wait` 뒤) · `release` (runRoot 를 trash) 앞에 `unix.Renameat2(unix.AT_FDCWD, <runRoot>/upper, unix.AT_FDCWD, keep.Upper, unix.RENAME_NOREPLACE)` 한 번. keep.Upper 의 부모는 있어야 하고 keep.Upper 는 없어야 한다 (대기 자리 `<이름>/upper`). 실패하면 `keep the upper: <원인>` 이고 upper 는 runRoot 와 함께 trash. Finalize 가 helper 를 죽였으면 (`aborted`) upper 는 이미 trash 에 있다 — `keep the upper: the session was aborted and its upper went to trash`. Close 는 처음 부른 한 번만 돈다 (`runtime.go:252` ~ `:259` 의 onceSession · `runc_overlay_linux.go:487` 의 closeOnce) — runBuildStep 은 `Keep{Upper}` 를 넘기는 Close 를 첫 부름으로 둔다. 안전망으로 둔 `defer Close(Keep{})` 는 그 뒤에 불려 아무것도 안 한다 | FD 규칙 3절 「닫기」 · trash 가 넘긴 일 (받는 일 21). 오늘 `Close` 는 `Keep` 을 받기만 한다 (`runc_overlay_linux.go:486` · 주석 :483 을 고친다). linux 의 rename(2) 은 대상이 빈 디렉터리면 덮고 성공한다 — `os.Rename` 으로는 「덮지 않는다」 를 못 지킨다 (선례 `merge_linux.go:274` 의 RENAME_NOREPLACE). helper 의 unmount 는 MNT_DETACH 다 (`:1122`) — helper 가 끝나 namespace 가 사라진 뒤에 옮긴다 |
| 7 | 초안 파일의 읽기와 쓰기 | 읽기 — `os.Lstat` 가 보통 파일이고, 연 파일의 `Stat` 과 `os.SameFile` 이고, 주인이 기대한 uid (노드 uid · 인자로 받는다) 이면 읽는다 · 1 MiB 까지 · schema 1. uid 는 `fileOwner` 가 읽는다 — linux 밖에서는 모른다고 돌려주고 읽기가 거절한다. 쓰기 — 같은 디렉터리의 `os.CreateTemp` (0600) · fsync · rename · 디렉터리 fsync. 모든 플랫폼에서 빌드된다 (Baker 는 linux 에서만 만든다) | FD 엔티티 4절은 「O_NOFOLLOW · 보통 파일」이라 적었다. `syscall.O_NOFOLLOW` 가 windows 에 없다 (3절 측정 4). Lstat 와 연 파일이 다르면 거절하므로 사이에 symlink 로 바뀌어도 그 내용을 읽지 않는다 — 내용 보호는 같다. 다만 Lstat 과 open 사이에 FIFO 를 가리키는 symlink 로 바뀌면 open 에서 멈출 수 있다 (O_NOFOLLOW 면 곧바로 ELOOP 다). 대기 자리가 0700 이라 (32번) 그 틈을 쓸 수 있는 것은 같은 uid 뿐이고 위협은 작다. 주인 확인은 선례 lower 의 `readJSON` (`perm_linux.go:134` ~ `:150`) 을 따른다. linux 에만 있는 것은 `mergehelper_linux.go` 에 둔다 (1절 크로스 빌드) |
| 8 | IR 대조의 셸 한 줄과 출력 | 아래 글 (4.1 끝) 이 Go 상수 `probeScript` 다. 출력은 줄마다 `mode=` · `head=` · `tagged=` · `tag=` (여럿) · `url=` · `branch=`. exit — 0 이면 읽었다 · `git rev-parse -q --verify` 의 1 은 태그가 로컬에 없다 (실패가 아니다) · 그 밖의 0 아닌 값은 그 git 의 exit 로 「대조를 못 함」· 125 는 `.repo/manifests` 로 못 들어갔다. `parseProbe(stdout, exit, stderr) irProbe` 가 풀고 `branch=HEAD` 는 "" 로 | FD 규칙 4절 (명령 다섯 · 앞의 셋이 실패하면 못 함 · 뒤의 둘은 빈 값) · FD 엔티티 5절. 이 기계의 dash 로 다섯 모양을 돌려 확인했다 (3절 측정 3) |
| 9 | 노드 명령의 출력 | bash 확인 · 대조 · pinned 는 단계 로그에 흘리지 않는다. stdout 은 대조만 읽고 (64 KiB 까지), stderr 는 마지막 줄 (1 KiB 까지) 만 문장에 싣는다. 계약의 명령 (sync · builds) 만 오늘처럼 버퍼와 진행 청크로 | FD 규칙 3절 「출력」은 계약의 명령만 말한다. 대조의 출력은 기계가 읽는 줄이다 |
| 10 | 시간 글자 | 단계 로그의 걸린 시간은 `Round(time.Second)` (`3m12s`). 대기 로그의 남은 시간은 1분 이상이면 `Round(time.Minute)` 의 글자에서 끝의 `0s` 를 뗀다 (`3h52m0s` -> `3h52m` · `4h0m0s` -> `4h0m`) · 아래면 `Round(time.Second)`. 마감은 UTC · 초 단위 · 뒤에 ` node clock`. `took the lower lock after` 는 `Round(time.Second)` | FD 규칙 8.3 · 16.2 의 예시 글자 (`(3h52m left)`). `Round(time.Minute).String()` 은 `3h52m0s` 를 낸다 |
| 11 | 상수와 시험이 바꾸는 값 | 패키지 변수 — `mergeWatchEvery` 10초 (Exclusive 의 every) · `waitLogEvery` 5분 · `foreignRetry` 60초 · `staleRetry` 10분 · `mergeHelperCommand` (여는 argv). 상수 — `pendingDirName` "pending" · `draftName` "bake.json" · `draftMax` 1 MiB · `pinnedFile` ".enode-manifest.xml" · `pinnedMax` 16 MiB · `pinCommand` · `bashCheck` · `reasonBakeRunEnded` "the bake run ended before the merge" · `abandonedReason(phase)`. Baker 의 시계는 `now` 칸. 시험 훅 `beforeStartMerging func()` — 제품은 nil 이고, 시험이 merge 단계의 시작 전 확인과 `startMerging` 사이에 몸통을 끼운다 (Step 12). 시험은 패키지 변수를 t.Parallel 없는 시험에서만 바꾸고 `t.Cleanup` 으로 되돌린다 | FD 규칙 8.2 · 8.4 · 13 · 14절의 값. 7.1 의 재시작 줄은 `abandonedReason(lower.PhasePending)` 과 같은 함수로 대 본다 (결정 52) |
| 12 | 잠금의 차례 | `heldBake.mu` 를 먼저, `Baker.mu` 를 나중에 잡는다 (몸통이 held 를 비울 때). onStale · runBuildStep 은 `Baker.mu` 만. `LowerGuard.mu` 다음에 `Baker.mu` 인 차례가 하나 있다 — guard 가 g.mu 아래에서 `f(st)` 를 부르고 onStale 이 Baker.mu 를 잡는다 (18번). 그 반대 — `Baker.mu` 나 `heldBake.mu` 를 쥔 채 g.mu 를 잡는 것 (`guard.Dir()` · HoldBake · DropBake) — 은 없다. 몸통은 `done` 을 mu 아래에서 적고 파일 일은 mu 밖에서 — `startMerging` 은 mu 아래에서 `done` 을 보고, `cancelMerging` 은 mu 아래에서 merging 표지를 거둔다 (13번). **`abandon` 은 `Baker.mu` 를 쥔 채 부르지 않는다** — abandon 이 heldBake.mu 다음 Baker.mu 를 잡으므로 자기와 교착한다. runBuildStep 이 Baker.mu 아래에서 held 를 둔 직후의 오류 길은 mu 를 놓은 뒤 몸통을 부른다 | 세 고루틴 (Worker · 광고 응답의 몸통 · 배경 재개) 이 같은 굽기를 본다. 차례를 하나로 두어 교착을 막는다. `-race` 시험 (Step 9 · 20). lowerguard 의 `bakeGoneLocked` 는 `go abandon()` 으로 g.mu 밖에서 부른다 (`lowerguard.go:419` ~ `:434`) |
| 13 | 몸통의 모양 | `abandon(reason string, builds []lower.BuildRecord)` 은 FD 그대로. HoldBake 에 넘기는 함수는 `func() { h.abandon(reasonBakeRunEnded, h.draftBuilds()) }` (`draftBuilds` 는 held 가 쥔 초안의 builds). `heldBake` 가 `*lower.Bake` 참조를 쥐고 있는다. merge 단계가 merging 을 못 쓰면 합치기 전이다 — `cancelMerging` 으로 표지를 거두고 배타 Release · 몸통 · FAILED | FD 규칙 3 · 11절 · FD 엔티티 3절 · 결정 44 (참조가 끊기면 finalizer 가 fd 를 닫아 flock 이 풀린다) · FD 규칙 7절 9 — 표지를 안 거두면 몸통이 startMerging 표지를 보고 아무것도 안 해, 잠금과 held 가 남고 그 노드는 재시작 전까지 모든 굽기를 bake_in_progress 로 거절한다 |
| 14 | claim 시각과 마감 | `runMergeStep` 첫 줄의 `time.Now()` (Worker 에 시계 칸이 없다 · 노드 시계). `context.WithDeadline(runCtx, claimedAt.Add(contractStep(step).MergeWait()))`. 끝의 갈래 — 마감 ctx 가 DeadlineExceeded 이고 runCtx 가 살아 있으면 마감 · runCtx 가 끝났고 Worker 의 ctx 가 살아 있으면 임대 · Worker 의 ctx 가 끝났으면 데몬 | FD 규칙 7절 3 · 8.5절 (끝나는 셋) |
| 15 | merge 단계의 업로드 (CG 물음 1 답 A) | 단계 로그와 `merged` 의 업로드에 업로드 예산을 건다 — 명령 단계의 `afterExit` 와 같은 길 (`context.WithTimeout(runCtx, uploadBudget)` · `w.upload`). 값은 `w.budgetsFor(step)` 의 업로드 몫이고, merge 단계는 계약이 budget 을 적지 못하므로 늘 기본 3분이다 (`contract.DefaultUploadBudget` · `effect.go:103` ~ `:108` 이 merge 의 budget 을 거절한다 — 「merge.wait sets how long it waits」 · 시험 `TestValidate_MergeStep/budget`). 결과 칸은 `settle(settleIn{upload, leaseEnded, uploadBudget})` 로 짓고 finalize 몫은 버린다 — 넘으면 FAILED · error `upload budget of 3m0s exceeded` · reason `upload_timeout` · `Upload` 칸 `timeout`. 제시간이면 `Upload` 칸 `ok` (한 건이 실패하면 `error`) 이고 DONE. 임대가 끝나 멈추면 upload `error` 이고 문구는 명령 단계처럼 `aborted: lease expired` (보고는 닿지 않을 수 있다). merge 칸은 합쳤으면 늘 싣는다 — 넘겨도 lower 는 이미 committed 이고 metadata 의 bake.run 이 이 Run 이다. `Finalize` · `FinalizedAt` · `ExitedAt` · exit_code 는 싣지 않는다 (merge 는 Finalize 와 명령이 없다). 넘긴 사실은 노드 로그의 `upload budget exceeded; not uploaded` (`upload.go:474`) 와 result 에 남는다 — 단계 로그는 먼저 올라가므로 거기에는 없다 (명령 단계와 같다) | 정본 ADR-075 결정 5 (ADR-075:437 ~ :442 — 업로드 예산을 넘기면 FAILED `upload_timeout` · 명령 실패와 구분되는 원인) · ADR-077:261 ~ :263 (merge 에서 빼는 것은 Finalize 예산뿐) · 원인 코드는 이미 있다 (`internal/contract/result.go:94`). 명령 단계의 모양 — `claim.go:837` ~ `:847` (`uctx` · `w.upload` · `settle`) · `finalize.go:227` ~ `:241` (upload 칸 · 문구 · reason) · `upload.go:472` ~ `:479` (uploadStopped). **Mediator 는 막지 않는다** — `api.go:432` 는 error 가 비었는지만 보고 (`completed`) · reason 어휘에 `upload_timeout` 이 있다 (`store/claim.go:962` ~ `:963`) · `ReportStep` (`store/claim.go:1001` ~ `:1030`) 은 단계 종류를 안 보고 result 를 그대로 봉인한다 · merge 에 대한 거절은 exited 에만 있다 (`:847`). lower 는 합쳐졌는데 Record 는 실패를 말한다 — ADR-077:290 ~ :292 의 재개와 같은 모양. FD 규칙 5 · 7 · 16.1절과 FD 흐름 3 · 7절을 고쳤다 |
| 16 | merge-helper 의 주고받기 | 입력 한 줄 · 출력 한 줄 (FD 엔티티 8절). 호스트가 응답을 오류로 되돌린다 — kind `preflight` 는 `*helperPreflightError{Check, Msg}` (문장은 helper 가 보낸 merge 의 문장 그대로) · `op` 는 `*helperOpError` · `io` 는 보통 오류. 응답이 없거나 JSON 이 아니면 `cannot run the merge helper: <원인> (<stderr 꼬리>)`. helper 안의 Apply 는 `context.Background()`. 프로세스 그룹 · Pdeathsig SIGKILL. 호스트는 helper 를 끝까지 기다린다 — 데몬의 ctx 로 죽이지 않는다. 호스트는 stdout 의 첫 줄만 읽고, stderr 꼬리는 `cmd.Wait()` 가 돌아온 뒤에 읽는다. 자리 — 띄우고 주고받는 `callMergeHelper` 는 `mergehelper_linux.go` (linux 밖은 `mergehelper_other.go` 의 짝이 지원 안 함) · 타입과 응답을 오류로 되돌리는 `helperErrorOf` 는 `bakerule.go` (모든 플랫폼) | FD 규칙 7 · 9 · 10절 (본체에 상한 없음 · SIGTERM 이면 Apply 를 끝낸다 · 읽기 실패는 PreflightError 가 아니다). trash-helper 와 같은 여는 모양 (`trash_linux.go:104`) · Wait 뒤의 stderr (`trash_linux.go:168` ~ `:176`). Wait 전에 stderr 를 읽으면 기준선 흔들림과 같은 흠이 된다 (3절 측정 6) |
| 17 | 마운트 번호 읽기 | `resolve` 의 Lstat 옆에 `unix.Statx(AT_FDCWD, real, AT_SYMLINK_NOFOLLOW, STATX_MNT_ID)`. mask 에 `STATX_MNT_ID` 가 없으면 `merge preflight: <path>: the kernel does not report a mount id` (PreflightError 가 아니다). 판정 `checkMounts(upper, lower, trash uint64)` 는 `decide.go` (모든 플랫폼). 차례는 `checkDevices` 다음 — 다른 filesystem 이면 filesystem 문장이 먼저 | FD 규칙 10절 · FD 엔티티 9절 · FD 계획 2.2 (bind 별칭은 st_dev 가 같고 마운트 번호가 다르다) |
| 18 | OnStale 을 부르는 자리 | `sourcesLocked` 에서 ReadState 가 된 뒤 phase 가 building · pending · merging 이고 `g.bake == nil` 이고 `g.onStale != nil` 이면 `f(st)` 를 곧바로 부른다 (g.mu 아래 · 고루틴을 띄우지 않는다). f 는 막지 않아야 한다 — `Baker.onStale` 은 Baker.mu 아래에서 닫힘 (`closed` — `Wait` 가 적는다) · held · resuming · retryAt 을 보고, 할 일이 있으면 resuming 을 세운 뒤 배경 일 (TryBake · state 다시 읽기 · 정리나 재개) 을 `b.wg.Go` 로 연다. 한 번에 배경 일 하나 — resuming 이 광고 주기의 정리도 덮는다. `Dir()` 은 mu 아래에서 `g.dir` | FD 엔티티 10절 · FD 흐름 6절. 광고 루프를 막지 않는다 (FD 규칙 13절). `go f(st)` 로 띄우면 그 고루틴은 Baker.Wait 가 못 기다리고, 정리 (Trash.Move · fsync) 가 멈추면 광고마다 하나씩 쌓인다. `go func(){ b.wg.Add(1) ... }()` 모양은 go vet 이 잡는다 (CI 의 vet 은 막는 단계다). `closed` 는 FD 엔티티 2절의 칸에 이 계획이 더한 것이다 — Wait 뒤에 wg 에 더하지 않게 한다. FD 흐름 6절 (business-logic-model.md:175) 과 FD 엔티티 10절은 `go OnStale(st)` · 「따로 돈다」 로 적었다 — 광고를 막지 않는다는 목적은 같고 수단이 다르다 |
| 19 | 정리의 몸통 하나 | `Baker.clean(lock, st, from)` 이 FD 규칙 12.2 의 표를 한 벌로 — 기동 · 광고 주기 · build claim 이 부른다. committed 줄 (자기 scratch 의 버려진 대기 자리) 은 `from` 이 기동일 때만. building · pending 줄은 pending_upper 의 모양 (`pendingShape`) 을 먼저 본다 — 틀리면 옮기지 않고 경로를 노드 로그에 한 줄 (`the pending upper path does not look like one this node writes; leaving it`) 쓰고 상태만 committed + last_attempt | FD 규칙 12.2 끝 줄과 13절 · FD 규칙 12.2 의 모양 줄 (business-rules.md:541 ~ :542 — 확인을 안 거치면 state.json 에 적힌 아무 경로가 trash 로 가고 삭제자가 지운다. state.json 은 uid 를 보므로 쓸 수 있는 것은 같은 사용자뿐이다 · `dir_linux.go:193` ~ `:206`) · 결정 42 (광고 주기는 committed 면 TryBake 를 안 한다) |
| 20 | last_attempt 의 builds (정리) | 초안이 있으면 초안의 builds · 없으면 비운다. 초안을 못 읽으면 비우고 노드 로그 한 줄. 몸통은 부른 쪽이 넘긴 것 | FD 규칙 12.2 (낡은 상태 정리의 last_attempt) |
| 21 | url 의 비밀번호 | `scheme://` 모양일 때만 `net/url` 로 풀어 `User.Password` 가 있으면 `url.User(username)` 로 바꿔 다시 쓴다. `scheme://` 모양인데 못 풀면 "" (비밀이 새지 않는 쪽). scp 모양과 경로는 그대로 | FD 규칙 3.1 · 결정 10 |
| 22 | 대기 로그의 「모습」 | 살아 있는 기록을 node 순으로 세워 node · role · run · acks 를 잇고 Unnamed 를 더한 글자. 바뀌면 쓴다. 쥔 쪽 줄의 노드 이름은 Label (없으면 node) | FD 규칙 8.2 · 8.3 · FD 엔티티 11절 |
| 23 | 시험 틀 | 가짜 런타임 (`Capability().Writes` 는 isolated · argv 마다 exit · stdout · stderr · 지연을 적어 두고 · upper 로 쓸 임시 폴더를 만들어 `Close(Keep{Upper})` 에서 rename · 받은 `ProcessSpec` 을 모두 적는다) · 진짜 lower 자리와 잠금 (`newGuardFixture` 모양) · namespace 없는 merge-helper (`TestMergeHelperProcess` — 시험 바이너리를 다시 띄워 진짜 `merge.Preflight` · `Apply` 를 임시 폴더에 돈다 · trash-helper 선례) · 가짜 Mediator (finalize 시험의 `mediator`) · 잠금 핸들을 t.Cleanup 으로 놓는 도우미 (5절 규칙 9). 틀은 `bake_fake_linux_test.go` 에 두고 Step 9 가 만든다 | FD 흐름 8.1 · 9절 |
| 24 | Mediator 왕복과 판정 시험의 자리 | 조각 5 의 왕복 (store.Claimed 를 JSON 으로 · `enode.Step` 으로 풀어 `contractStep(step).MergeWait()`) 은 `internal/enode` 의 시험이 `internal/store` 를 가져와 한다. 판정 시험 둘은 시험 DB 가 있는 `internal/api/bake_test.go` | 경계 시험은 `go list -deps` 에 `-test` 를 안 쓴다 (`boundary_test.go` 머리 주석) — 시험 import 는 제품 경계와 무관하다. `contractStep` 은 안 내보낸 함수라 `internal/api` 에서 못 부른다 |
| 25 | 400 을 제출 경로로 | 조각 5 의 400 셋은 contract 의 Validate 시험이 이미 초록이다. 제출 경로의 400 은 `TestSubmitRejectsBadContract` 에 굽기 줄 셋 (merge 없음 · 이름 규칙 · 이름 겹침) 을 더해 확인한다 | FD 흐름 8.1 끝 (Code Generation 계획이 시험 이름을 조각 5 목록에 적는다) · 7.1 절 |
| 26 | 조각 스크립트 | 이름 `bake-common.sh` (공용 · source 로 읽는다) · `slice-5.sh` (기계 시험을 이름으로 돌리고 `slice 5: green` 또는 `red`) · `slice-6.sh` · `slice-7.sh` · `slice-8.sh`. 입력 `M` · `T` · `WS` (굽는 노드의 워크스페이스 = lower 루트) · `SIBLING_WS` (같은 lower 를 다른 경로로 가리키는 형제의 워크스페이스 — 글자가 같으면 계약이 두 노드를 나누지 못하므로 멈춘다) · `SYNC_URL` · `IR` · `BUILD_A` · `BUILD_B` (구성 명령 — 기본은 워크스페이스에 파일 하나를 쓰는 짧은 명령 · 사람이 자기 명령을 넣는다 · 주석의 예시는 `<your build command>` 자리표시). 계약의 sync 는 FD 흐름 8.3 대로 `git init` · `git remote add origin` · `git fetch` · `git checkout` | FD 흐름 8.3 · 결정 20 · 36 · 38. 사내 스크립트 · 구성 이름 · 명령을 싣지 않는다 (제품은 사내 내용을 모른다) |
| 27 | FD 16절 밖에 더하는 오류 글자 | `cannot create the pending upper directory: <원인>` · `cannot create the trash of the pending upper: <원인>` · `cannot run the merge helper: <원인>` · `keep the upper: <원인>` (settle 이 `runtime cleanup:` 을 앞에 붙인다) · `cannot pin the manifest: <원인>` (sha256 을 못 읽음 · 보통 파일이 아님 · 16 MiB 넘음 — 30번) · rmdir 과 metadata 의 오류도 `merge stopped: <원인>; the lower stays merging and a node on this lower resumes it` 모양 · 재개의 `cannot read the bake draft: <원인>` (노드 로그) | FD 규칙 16절이 적지 않은 갈래 (대기 자리 만들기 · trash 만들기 · helper 를 못 띄움 · 닫기의 rename · sha256) |
| 28 | Result 칸 | DONE 의 exit_code 는 마지막으로 돈 계약 명령의 값 (IR 어긋남이면 sync 의 0). 명령 앞의 거절과 오류 (거절 · bake_in_progress · 상태 자리 · 대기 자리 · building 쓰기 · 세션을 못 엶 · bash 확인) 는 exit_code 를 싣지 않는다. 명령이 끊긴 것 (임대 만료 · 명령 중 helper 죽음 `runtime run:`) 도 싣지 않는다 — 오늘 명령 단계의 규칙 (`claim.go:782` ~ `:789`). 명령 뒤의 오류와 예산 초과 (대조를 못 함 · pinned 실패 · 닫기 · 초안 · pending 쓰기 · finalize_timeout · upload_timeout) 는 마지막으로 돈 계약 명령의 exit_code 를 싣는다 (대조 · pinned 실패면 sync 의 것). merge 단계는 exit_code 가 없다. build 칸은 계약의 명령이 하나라도 돌았으면 FAILED 에도 싣는다 | FD 규칙 6절 표의 exited 칸 (business-rules.md:250 ~ :252 — 닫기 · 예산 초과 줄이 「마지막 build」) · 15절 · 병합된 finalize 규칙 (finalize FD business-rules.md:116 ~ :117 — 예산을 넘겨도 exit_code 는 남는다 · 조각 3) · `claim.go:794` ~ `:796` 의 주석 (「예산을 넘긴 것은 error 에 남아 완주가 아니게 되지만 exit_code 는 그대로 남는다」) |
| 29 | 주석 두 곳 (행렬 밖 · 주석만) | `runtime.go:101` ~ `:102` 의 「아직 늘 비어 있고」를 대기 자리 (bake) 가 쓴다로 · `lower.go:113` 의 「pending · merging 일 때」를 building 부터로 | FD 규칙 2절 (pending_upper 를 building 때 적는다) · 4절 6번. 거짓이 된 주석을 남기지 않는다. FD 흐름 10절 끝은 「runtime.go 는 안 바뀐다 · CG 가 확인한다」 였다 — `Keep` 은 그대로이고 거짓이 된 주석만 고친다 |
| 30 | pinned 파일 읽기 | 닫은 뒤 대기 자리의 `upper/.enode-manifest.xml` 을 호스트가 읽는다 (`readPinned` · `bakefile.go`). `os.Lstat` 이 보통 파일이고 크기가 `pinnedMax` (16 MiB) 안일 때만 열고, 연 파일의 `Stat` 과 `os.SameFile` 이면 `io.LimitReader(pinnedMax+1)` 로 sha256 을 계산한다 (7번의 초안 읽기와 같은 방법). 아니면 `cannot pin the manifest: <원인>` 으로 몸통 · FAILED (exit_code 는 sync 의 것 · 28번) | FD 규칙 3절 — pinned 는 9, builds 는 10 이라 계약의 명령이 그 파일을 symlink · FIFO · 장치로 바꿀 수 있다 (business-rules.md:88 ~ :90). :125 는 「닫은 뒤 대기 자리에서 읽는다」 만 적었다. `os.Open` 에 sha256 을 붙인 그대로면 symlink 를 따라 호스트 파일을 읽고 (그 sha256 이 0644 metadata 에 실린다), FIFO 에서는 돌아오지 않는다 (QA 측정 · mkfifo 에는 권한이 필요 없다). Worker 가 굽기 잠금을 쥔 채 멈추면 그 lower 의 다음 굽기가 모두 bake_in_progress 이고 데몬을 멈출 때 `wg.Wait` 가 안 돌아온다. 닫은 뒤라 계약은 더 쓰지 못하고 대기 자리는 0700 이다 |
| 31 | 기준선 흔들림의 원인 | 이 유닛이 runc_overlay 두 파일을 고치므로 원인 둘도 고친다 — 시험 helper 의 eof 갈래를 `os.Exit(1)` 로 끝낸다 (`runc_overlay_linux_test.go:27` ~ `:30`) · `Open` 이 helper 응답을 못 받은 길은 abort (helper 를 죽이고 Wait) 뒤에 stderr 꼬리를 문장에 붙인다 (지금은 receive 가 Wait 전에 읽는다 · `runc_overlay_linux.go:293` ~ `:301`). 새 merge-helper 는 처음부터 그 모양이다 (16번) | 3절 측정 6 — 혼자 `-count=300` 에 11 번 빨강. CI 의 둘째 실행 (`-coverpkg`) 에서는 return 으로 끝난 자식의 stdout 뒤에 coverage 줄까지 붙는다 (QA 측정 · Go 1.26.6). trash-helper 는 둘 다 옳다 (`trash_test.go:25` ~ `:51` 의 os.Exit · `trash_linux.go:168` ~ `:176` 의 Wait 뒤 stderr). 고치지 않으면 이 유닛의 PR 이 CI 에서 가끔 빨갛다 |
| 32 | 대기 자리의 권한과 초안의 주인 | `<scratch>/pending/` 과 `<키>` 는 `os.MkdirAll(…, 0o700)` (선례 `scratch/trash.go:44`), `<이름>` 은 `MkdirTemp` (0700). 초안을 읽을 때 주인 uid 가 노드 uid 인지 본다 (7번) | FD 엔티티 4절은 `<이름>` 의 MkdirTemp (0700) 만 적었다. 선례 `readJSON` (`perm_linux.go:134` ~ `:150`) 은 주인을 본다 |
| 33 | 격리 노드의 Prepare (CG 물음 3 답 A) | `Prepare` (`workspace.go:30`) 가 `WorkspaceWrites(w.Runtime)` 를 본다. isolated 면 — 저장소 확인 (`DetectRepo == spec.Repo` · 어긋나면 오늘 문장 그대로 거절) 만 하고 reset · clean · `repo forall` 을 돌리지 않는다 · `PrepClean` (새 upper 는 뒤의 `Runtime.Open` 이 연다) · 노드 로그 `workspace prepared` 에 `how` 가 `fresh upper` (native 는 `reset and clean`). spec.Repo 가 비었으면 isolated 는 `PrepClean` 이고 native 는 오늘처럼 `PrepUnprepared` 와 경고. native (in-place · Runtime nil 포함) 는 오늘과 같다. `claim.go` 의 준비 분기 (`:554` ~ `:565`) 는 안 바뀐다 — 나눔은 Prepare 안이다. runtime 증거는 더하지 않는다 — 모든 결과의 `environment.runtime` 이 이미 `runc-overlay` 인지 `native` 인지를 싣는다 (`claim.go:393` ~ `:394` 의 `w.RuntimeRecord` · `store/claim.go:916` ~ `:917` 이 Record 에 봉인). environment 칸이 없는 노드는 프로필이 없는 노드이고 native 다 | 정본 ADR-072 결정 3 (§5 「오버레이 노드에서 Prepare 는 윗 층을 버리는 것이다」 · §5.1 「윗 층을 버리는 것이 그 되돌림이고 저장소가 없어도 된다」 · §5.2 「방법은 달라도 PrepClean 의 뜻은 같다」 · §8 「오버레이 upper 폐기인지 reset/clean 인지는 별도 runtime evidence 가 말한다」) · ADR-073 §8 (Record 의 runtime kind). 오늘 Prepare 는 runtime 을 안 보고 lower 뿌리 (`internal/enode/environment.go:26`) 에서 호스트 reset · clean 을 돈다 (`workspace.go:94` ~ `:118`) — 굽기 뒤에는 계약의 sync 가 쓴 `.git/config` · `.repo/` 가 lower 에 남아 계약의 코드가 호스트에서 돈다 (QA 측정 — core.fsmonitor 가 reset --hard · clean -df 에서 실행됐다 · repo 런처는 그 트리의 `.repo/repo` 를 실행하는 설계) · reset --hard 는 형제가 공유 잠금을 쥔 lower 를 바꾼다. 저장소 확인을 남기는 까닭 — config --get · rev-parse 는 fsmonitor 와 hooks 를 안 부르고 (QA 측정 git 2.39.2) 쓰지 않는다 · 광고 루프가 같은 `DetectRepo` 를 lower 에 이미 돈다 (`detect.go:203`). 이 회차의 팩 · Inception · FD 는 ADR-072 의 §6.4 · §8 · §9 만 받았고 결정 3 은 받지 않았다 (FD 흐름 13.1 에 한 줄). 정본 되돌림은 없다 — 정본이 정한 것을 코드가 따른다 |
| 34 | 격리 노드의 Finalize diff (CG 물음 5 답) | runc-overlay 의 `runcOverlaySession.Finalize` 가 차례를 둘로 한다. (1) helper 의 `finalize` (collect · 지목 경로 stat · 명시 훑기) 는 오늘처럼 보내되 helper 는 `spec.Diff` 를 끈다 — `h.finalize` 가 `false` 로 바꾼 뒤 `finalizeLocal` 을 부른다 (호스트가 켜서 보내도 helper 는 git · repo 를 안 돈다). (2) `spec.Diff` 면 같은 세션에서 명령 하나를 더 돌린다 — 굽기의 IR 대조 · pinned 와 같은 `Run` 길이다 (callMu 를 쥔 채 Run 의 몸통을 뗀 안쪽 run 을 부른다 · 같은 잠금을 두 번 잡지 않는다). argv `["sh", "-c", sessionDiffScript]` · 작업 폴더 `Paths().Dir` · 환경은 `ENODE_DIFF_MODE` (`diff` · `stat`) 와 repo 모양의 스크립트 둘 (`ENODE_REPO_DIFF_SCRIPT` · `ENODE_REPO_STAT_SCRIPT` — `repoDiffScript` · `repoStatScript` 그대로) 뿐이다 (호스트 환경 변수를 넘기지 않는다). **때** — helper 의 답 뒤 · Close 전 · Finalize 의 ctx (예산) 안. helper 의 답이 이미 마감이면 diff 를 건너뛰고 DiffError `the finalize deadline passed before the workspace diff` · 도는 중 마감이면 Run 이 cancel 되고 DiffError `runtime run: <원인>` · 둘 다 Finalize 는 ctx 의 오류 (finalize_timeout). helper 를 죽였으면 (`aborted`) diff 를 안 돈다. 차례가 collect · stat · 훑기 · diff 로 바뀐다 (native 는 collect · stat · diff · 훑기) — stat 을 diff 앞에 두는 까닭 (느린 diff 가 마감을 넘겨도 changed 는 남는다 · `runtime.go:207` ~ `:208`) 은 지킨다. **출력** — stdout 은 호스트의 세는 버퍼로 받는다 (`diffLimit` 바이트까지 담고 전체 길이는 센다) · stderr 는 마지막 줄 (1 KiB). 전체가 `diffLimit` 안이면 그 바이트 · 넘으면 `ENODE_DIFF_MODE=stat` 으로 한 번 더 돌려 `diffNote` 머리 (전체 길이 · 상한) 와 stat 을 잇는다 — 오늘 `workspaceDiff` 와 같은 바이트다. 빈 diff 면 파일을 안 만든다. **놓기** — `placeWorkspaceDiff` 가 호스트의 `$OUT` (FinalizeSpec.Out) 에 `os.CreateTemp(out, ".enode-diff-*")` (O_EXCL 새 파일) 로 쓰고 fsync 뒤 `workspace.diff` 로 rename 한다 — 단계가 그 이름에 미리 둔 symlink · FIFO 를 따라가지 않는다 · 그 이름이 디렉터리면 rename 이 실패하고 DiffError `cannot place workspace.diff: <원인>` · 임시 파일은 지운다 (`.enode-` 로 시작해 올라가지도 않는다 · `upload.go:490`). diff 는 파일로 오지 않으므로 호스트가 컨테이너가 쓴 파일을 읽는 일이 없다 — 30번의 걱정이 생기지 않는다. **DiffError 의 글자** (영어) — exit 125 `the prepared rootfs has no git` · 124 `the prepared rootfs has no repo` · 121 `cannot prepare temporary index: <stderr 마지막 줄>` · 122 `intent-to-add: <stderr 마지막 줄>` · 그 밖의 0 아닌 값 `workspace diff exited <n>: <stderr 마지막 줄>` · -1 `runtime run: <원인>`. 판정은 오늘 diff 실패와 같다 — DiffError 는 진단이고 단계는 그것으로 실패하지 않는다 (단계 로그 끝의 `workspace.diff was not produced: <DiffError>` · `finalize.go:305` ~ `:306`). native 노드는 오늘처럼 `finalizeLocal` 의 호스트 git 이다 | 진행자 측정 (audit 2026-09-29T14:13:46Z) — helper 는 `unshare --user --map-root-user --mount` 뿐이고 PID · network namespace 가 없다 (`runc_overlay_linux.go:233` ~ `:237`) · helper 가 merged 에서 `finalizeLocal` 을 부르고 (`:1077` ~ `:1090`) `spec.Diff` 면 `writeWorkspaceDiff` 를 돈다 (`runtime.go:219` ~ `:220` · `diff.go:51` ~ `:150`) · diff.go 와 같은 차례를 git 2.39.2 로 돌리니 core.fsmonitor 넷 · filter clean 둘이 실행됐다. repo 모양은 런처가 트리의 `.repo/repo` 를 실행하므로 git 설정을 끄는 것 (`-c core.fsmonitor=`) 으로는 못 막는다 — 그래서 세션 안이다. 정본과 어긋나지 않는다 — execution-environment.md §10.3 (:597 ~ :608 — Close 전에 merged view 에서 diff) 과 ADR-073 §6 (Harvest 의 기존 diff) 은 그 git 이 어디서 도는지 적지 않았고, 세션의 작업 폴더가 그 merged view 다. 「호스트 쪽 helper 는 워크스페이스의 설정을 실행하지 않는다」 는 정본에 없는 규칙이라 보탬 제안으로 FD 흐름 13.1 에 올렸다. 스크립트는 `probeScript` 처럼 노드의 고정 글자이고 LC_ALL=C · GIT_CEILING_DIRECTORIES 를 안에서 export 한다 (FD 규칙 4절과 같은 까닭) · 임시 인덱스는 컨테이너의 mktemp (작업 폴더 밖). finalize 유닛 (③ · 병합됨) 의 수확을 고친다 |

**4절 8번의 셸 한 줄** — 노드가 짓는 고정 글자다. 계약은 못 바꾼다. 이 기계의 dash 로 돌려 본 모양이다 (3절 측정 3).

```text
   export LC_ALL=C GIT_TERMINAL_PROMPT=0
   GIT_CEILING_DIRECTORIES=$(cd .. && pwd); export GIT_CEILING_DIRECTORIES
   if [ -d .repo/manifests ]; then echo mode=repo; cd .repo/manifests || exit 125
   elif [ -e .git ]; then echo mode=git
   else echo mode=none; exit 0; fi
   h=$(git rev-parse HEAD) || exit $?
   echo "head=$h"
   t=$(git rev-parse -q --verify "refs/tags/$ENODE_IR^{commit}"); rc=$?
   [ $rc -le 1 ] || exit $rc
   echo "tagged=$t"
   g=$(git tag --points-at HEAD) || exit $?
   printf '%s\n' "$g" | sed -n 's/^./tag=&/p'
   echo "url=$(git config --get remote.origin.url)"
   echo "branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)"
```

- 작업 폴더는 세션의 워크스페이스 자리다. `GIT_CEILING_DIRECTORIES` 는 워크스페이스 자리의 부모 (FD 규칙 4절). `ENODE_IR` 은 세션 환경으로
  넘긴다 — 셸 글자에 IR 이 없다 (3.1)
- `git tag --points-at HEAD` 는 이름순이다 (refname 기본). 판정과 문장은 `irVerdict` 가 한다 (Step 5)

**4절 34번의 셸 한 줄** (`sessionDiffScript`) — 노드가 짓는 고정 글자다. git 모양은 `diff.go` 의 차례 (임시 인덱스 · read-tree HEAD · add -N . ·
diff --binary · 요약은 diff HEAD --stat · 모든 git 에 `-c core.quotePath=false`) 를 그대로 옮긴다. repo 모양은 `repoDiffScript` · `repoStatScript` 를
환경 변수로 받아 `repo forall -c` 에 넘긴다 — 호스트의 `repoDiff` 와 같은 글자다.

```text
   export LC_ALL=C GIT_TERMINAL_PROMPT=0
   GIT_CEILING_DIRECTORIES=$(cd .. && pwd); export GIT_CEILING_DIRECTORIES
   command -v git >/dev/null 2>&1 || exit 125
   if [ -e .repo ]; then
     command -v repo >/dev/null 2>&1 || exit 124
     if [ "$ENODE_DIFF_MODE" = stat ]; then exec repo forall -c "$ENODE_REPO_STAT_SCRIPT"; fi
     exec repo forall -c "$ENODE_REPO_DIFF_SCRIPT"
   fi
   i=$(mktemp) || exit 121
   trap 'rm -f "$i"' EXIT
   GIT_INDEX_FILE=$i; export GIT_INDEX_FILE
   git -c core.quotePath=false read-tree HEAD >/dev/null || exit 121
   git -c core.quotePath=false add -N . >/dev/null || exit 122
   if [ "$ENODE_DIFF_MODE" = stat ]; then git -c core.quotePath=false diff HEAD --stat
   else git -c core.quotePath=false diff --binary HEAD; fi
```

- 차례는 오늘 `isRepo` (`.repo` 가 있으면 repo 모양) 와 같다. git 이 없으면 repo 모양이어도 125 가 먼저다
- 같은 바이트인지는 Step 7 의 `TestSessionDiffScript_MatchesTheHostDiff` 가 호스트 sh 로 이 글을 돌려 `workspaceDiff` 와 댄다. 실제 rootfs 에서는
  Step 18 의 integration 이 처음 본다

### 4.2 FD 의 코드 주소를 `fb86ad2` 에 대 본 결과

FD 셋이 적은 코드 주소 예순 남짓을 열어 봤다 (`claim.go` · `lowerguard.go` · `advertise.go` · `runc_overlay_linux.go` · `finalize.go` · `env.go` ·
`repoid.go` · `trash_linux.go` · `lock_linux.go` · `merge.go` · `decide.go` · `merge_linux.go` · `scratch/trash.go` · `deleter.go` · `contract/bake.go` ·
`result.go` · `examples/bake.json` · `contract.go` · `store/claim.go` · `store.go` · `verdict.go` · `queue.go` · `schema.sql` · `observe.go` · `api.go` ·
`cmd/runctl/shape.go` · `cmd/enode/main.go` · INVARIANTS.md · ADR-077 (굽기 결정)). **주소는 모두 그 자리를 가리킨다** — `result.go:92` 는
reason 묶음의 `const (` · `result.go:164` ~ `:174` 는 주석 한 줄과 `BuildManifest` · `deleter.go:48` ~ `:60` 은 삭제자의 주석과 칸이다.

어긋난 것은 주소가 아니라 코드의 모양이다.

```text
   코드의 모양                                                    FD 가 기대한 것                         이 계획
   afterExit 가 Close(ctx, Keep{}) 를 박고 $OUT 전부를 올린다          build 가 Keep.Upper 로 닫고 manifest 만     4절 3 · 5번
   (claim.go:819 ~ :850 · upload.go:458)
   syscall.O_NOFOLLOW 가 windows 에 없다                            초안 읽기의 O_NOFOLLOW · 모든 플랫폼 파일   4절 7번
   lower 의 pollEvery (1초) 는 internal/lower 의 안 내보낸 변수다       -                                     5절 — enode 시험이 배타를 1초 안에 잡는 것을
                                                                                                          기다리게 둔다
   runtime.go:101 ~ :102 의 Keep 주석 (「아직 늘 비어 있고」)         runtime.go 는 안 바뀐다 · CG 가 확인한다   4절 29번 — Keep 은 그대로 · 거짓이 된
                                                                   (FD 흐름 10절 끝)                       주석만 고친다
   lower.go:113 의 PendingUpper 주석 (「pending · merging 일 때」)   적지 않았다 (FD 규칙 2절이 building 때     4절 29번
                                                                   부터 적는다고 정했다)
   linux 의 rename(2) 은 대상이 빈 디렉터리면 덮고 성공한다            rename 한 번 · 대상이 없어야 한다          4절 6번 — RENAME_NOREPLACE
   (os.Rename · QA 가 스크래치에서 확인)
   DetectRepo 는 .repo/manifests 를 os.Stat 으로 본다 (repoid.go:90)   FD 규칙 4절 「디렉터리면」               대조는 [ -d ] (FD 대로).  .repo/manifests 가 보통
                                                                                                          파일인 트리에서만 둘이 다르다 — 그대로 둔다
   runc 판 — SunnyVM 은 runc 1.3.4 (FD 규칙 3절은 이 기계의 1.1.5 로 측정)  bash 가 없으면 bash -c 가 exit 1          bash 확인은 1 이상이면 모두 「bash 없음」이라 판에
                                                                                                          기대지 않는다.  조각 6 이 1.3.4 에서 처음 본다
```

---

## 5. 시험의 시간 경쟁 · CI 러너와 이 기계

**lower-state 의 교훈** — `TestLocks` 가 여섯 번 중 다섯 번 빨갰다. 제품이 놓는 차례 (기록을 먼저 · lower.lock 을 나중에) 의 사이를 시험이
「마지막 관찰」로 잡았다. 고친 것은 시험이었다 — 놓기 전에 관찰을 떠 둔다. 이 유닛의 시험도 고루틴 셋 (Worker · 광고 응답의 몸통 · 배경
재개) 과 파일 잠금 둘을 오가므로 같은 흠이 생기기 쉽다. 규칙을 먼저 정한다.

```text
   규칙                                                              어디에
   1  고정된 sleep 으로 기다리지 않는다.  조건을 짧은 간격으로 다시 보는         배타를 잡기까지 (lower 의 pollEvery 1초 — enode 시험이 못 줄인다) ·
      waitFor(t, cond, msg) 로 기다린다 (upload_test.go:319 · 5초 · 5 ms       임대 감시 (claim.go 의 1초 ticker) · 광고 응답 뒤의 몸통
      간격 — 있는 도우미).  상한은 넉넉히, 판정은 조건으로
   2  몸통의 차례 (옮기기 -> committed -> 굽기 Release -> DropBake ->        몸통 · 합침 · merging 뒤 멈춤의 시험
      held 비움) 에서 시험은 마지막 (held == nil) 을 기다린 뒤 앞의 것을 본다.
      중간 상태를 「끝」으로 잡지 않는다 (TestLocks 의 흠)
   3  배경 고루틴은 Baker.Wait 로 기다린다.  onStale 은 시험이 직접 부른다      재개 · 광고 주기의 정리 · build claim 이 연 재개
      (LowerGuard 는 f 를 곧바로 부른다 — 부른 것은 lowerguard 시험이, 막지
      않는 것은 onStale 시험이 본다 · 4절 18번)
   4  10분 · 5분 · 60초 · 10초는 패키지 변수와 Baker.now 로 바꾼다.            retryAt · 대기 로그 · 그물 재시도 · watch 간격
      바꾸는 시험은 t.Parallel 을 안 쓰고 t.Cleanup 으로 되돌린다
   5  시각을 비교하는 시험은 하나 — 조각 5 의 merge.wait (300 ms · 900 ms).       TestMergeStep_WaitsForTheContractValue
      시각을 보는 것은 포기한 순간이다 — 보고가 닿은 때가 아니다.  포기한 순간은
      노드 로그의 마감 줄 (FD 규칙 16.3 「waiting for the lower lock (merge
      step)」 의 마감 한 줄) 의 slog Record 시각이다.  시험의 시작 시각은
      execute 를 부르기 전에 잡는다 — 노드의 claim 시각보다 먼저다.
      아래 끝은 엄격하다 (ctx 의 마감은 일찍 오지 않는다 · lock_linux.go:177 ·
      :274 ~ :283).  위 끝은 < wait + 300 ms (두 값 차이의 절반 · FD 흐름 8.1 ·
      결정 37).  값을 넓히지 않는다 — 넓히면 결정 37 을 바꾼다.  끝을 보고가
      닿은 때로 잡으면 몸통의 fsync 두 번과 결과 보고 HTTP 가 들어 아래
      「파일 일의 시간」 과 어긋난다
   6  로그 줄은 lockedBuffer 에 모으고 조건으로 기다린다 (몸통과 재개가 따로 쓴다)
   7  -race 는 CI 가 안 돈다 — 이 단계에서 돈다 (Step 20).  몸통의 동시성 시험
      (TestHeldBake_OneBodyUnderRace) 은 -race -count=10
   8  새 시험을 -count=20 (Step 1 이 떠 둔 시험 목록과의 차이로 -run 을 짓는다) ·   Step 20.  새 흔들림이면 시험이나 코드를 고친다.  기준선의 흔들림
      CPU 부하 아래 한 번 (taskset -c 0,1 로 두 코어에 묶고 옆에서 다른 패키지      (3절 측정 6) 은 4절 31번이 원인을 고친다 — 그 시험 -count=300 에 빨강 0.
      시험을 함께 돈다) · internal/enode 전체를 -count=5.  모두 -timeout 30m        그 밖의 것은 Step 1 의 목록과 대 보고 이 유닛이 들인 것이 아님을 적는다
      (부하 아래 internal/enode -count=5 는 기본 10분에 가깝다 — QA 측정 · 혼자 한 번 30.6초)
   9  시험의 잠금 핸들 (lower.Bake · Exclusive · Shared) 은 defer 나 t.Cleanup    모든 흐름 시험 · 재시작 장면 · 광고 주기의 정리
      으로 명시적으로 놓는다.  flock 은 같은 프로세스 안에서도 fd 가 다르면
      부딪치고, 놓지 않은 핸들은 GC finalizer 차례에 풀린다 — 새 Baker 의
      TryBake 성패가 GC 에 달린다.  lower.Dir 에는 Close 가 없다.  주인이 죽은
      것은 Bake.Release() 로 흉내 낸다.  재시작 장면은 옛 Bake 를 Release 하고
      새 LowerGuard 와 Baker 로 짓는다
   10 새 시험은 스스로 t.Skip 을 부르지 않는다 — helper 시험의 t.Skip("helper")    새 기본 태그 시험 전부
      관용구도 쓰지 않는다 (helper 입구는 환경 변수가 없으면 return · 선례
      TestTrashHelperProcess).  CI 의 스킵 감시 (ci.yml:341 ~ :441) 는 허용목록
      (.ci-allowed-skips · 주석 밖의 줄 0) 밖의 스킵 하나에도 빨갛다.  시험 DB 가
      없을 때의 newServer 스킵은 CI 에 DB 가 있어 안 걸린다 — Step 20 도 DB 를
      켜고 돈다
```

**CI 러너와 이 기계와 SunnyVM**

```text
                        CI (ubuntu-latest)                       이 기계                          SunnyVM
   /bin/sh              dash                                     dash                             dash
   git                  러너 이미지의 판 (측정 안 함)               2.39.2                           2.43.0
                        전역 설정은 러너의 것
   unshare (user ns)    막힐 수 있다 (Ubuntu 24.04 의 기본값 ·       --map-auto 가 막힌다 ·            된다 (apparmor_restrict 0)
                        측정 안 함)                               -rm 은 된다
   runc                 기대지 않는다 (측정 안 함)                  1.1.5                            1.3.4
   -race                안 돈다                                   Step 20 이 돈다                   -
   시험 DB              services 의 postgres                      scripts/testdb.sh                -
   스킵                 허용목록이 비어 있다 — 스킵 하나면 빨갛다      -                                -
   코어 · 속도           2 ~ 4 vCPU (저장소 종류에 따라) · 느릴 수 있다  4 코어 N100                      8 코어
```

- **시험이 기대지 않는 것** — unshare · runc · subordinate uid (integration 태그 뒤로만). 기본 시험의 merge-helper 는 namespace 없이 시험
  바이너리를 다시 띄운다 (4절 23번)
- **git 을 쓰는 시험** (`TestProbeScript_OnAHostShell`) — 러너의 전역 설정과 기본 브랜치 이름에 기대지 않는다: `GIT_CONFIG_GLOBAL=/dev/null` ·
  `GIT_CONFIG_NOSYSTEM=1` · 작성자 환경 변수 · `git init -b main`. `sh` 와 `git` 은 러너에 있다. linux 시험 파일에 둔다
- **키 글자를 박지 않는다** — lower-state 의 CI 빨강 (러너의 fsid 는 위 네 비트가 0 이라 열다섯 자리였다 · audit 의 PR #66 항목) 을 따른다.
  순수 표 (`TestPendingShape` · 모든 플랫폼 파일) 에는 열다섯 자리 키 (`lower.Key{FSID: 0x35b60f8473d0c15, Ino: 2}.String()` — 러너의
  모양) 와 열여섯 자리 키를 둘 다 싣는다. 이 기계의 fsid 는 열여섯 자리라 「늘 ReadRoot」 만으로는 열다섯 자리를 못 본다. 흐름 시험
  (linux 파일) 은 늘 `lower.ReadRoot` 의 `Key.String()` 으로 얻는다 — linux 밖의 ReadRoot 는 ErrUnsupported 다 (`lower_other.go`)
- **root 로 도는 경우** — 쓰기 실패를 디렉터리 0500 으로 만드는 시험은 root 면 막히지 않는다. 그때는 쓰기 함수 값을 바꿔 끼우는 갈래로
  간다 — 스킵하지 않는다 (CI 의 스킵 감시 · lower-state 4절 ⑳ 과 같은 방법)
- **파일 일의 시간** — fsync 가 드는 시험에 시간 상한을 걸지 않는다 (러너의 디스크가 다르다)

---

## 6. 단계 — 스물둘

- **체크박스** — 체크박스 하나의 일을 마치면 그 자리에서 [x] 로 바꾸고 끝에 결과 한 줄을 적는다 (core-workflow 의 계획 체크박스
  규칙 — 일을 마친 같은 자리에서). Step 22 는 모두 [x] 인지 다시 볼 뿐이다
- **컴파일 차례** — 단계마다 끝에서 `go build ./...` 와 `go vet ./...` 가 초록이다. 뒤 단계가 채울 몸통은 앞 단계가 겉면만 둔다 —
  Step 9 가 `onStale` · `resume` 의 겉면 (아무것도 안 한다) 을 두고 Step 13 이 채운다. `Worker.Bake` 는 `Baker` 가 생기는 Step 9 에서 더한다

### Step 1 — 기준선

- [x] `unit/bake` 가 `fb86ad2` 위에 있고 작업 트리가 깨끗하다 (이 계획 · 상태 · 감사 셋 밖) — HEAD `fb86ad2` (부모 `0262155`) · 커밋 안 된
      것은 계획 단계가 고친 여섯 (계획 · 상태 · 감사 · FD 두 장 · unit-of-work.md) 뿐이다
- [x] 시험 DB (`scripts/testdb.sh`) · 가짜 claude 스텁 (`.github/ci-stubs`). CI 의 측정 명령과 awk 로 `internal/enode` · `cmd/enode` ·
      `internal/merge` · `internal/contract` · `internal/store` · `internal/api` · `internal/lower` 커버리지 · 전체 · 통과 수 · 라우트 19 를 적는다
      — 시험 DB 는 다른 세션과 안 부딪치게 `ENODE_TEST_DB=enode_test_bakecg` · enode 84.4% (3454/4090) · cmd/enode 80.5% (194/241) ·
      merge 89.1% (303/340) · contract 92.5% (760/822) · store 82.9% (1575/1899) · api 82.5% (907/1100) · lower 93.2% (671/720) ·
      전체 87.0% (11404/13109) · 통과 2271 (하위 시험 포함) · 실패 0 · 스킵 0 · 미달 0 · 라우트 19 (`mux.HandleFunc(` 의 수)
- [x] `golangci-lint run ./...` 의 경고 목록을 스크래치 파일로 떠 둔다 (수만이 아니라 목록 — Step 20 이 diff 로 댄다 · lower-state 뒤 38)
      — 38 (줄 번호를 뗀 파일 · 문장 목록으로 떠 두었다 — 편집으로 줄이 밀려도 diff 가 된다)
- [x] 시험 목록 — `go test -list . ./internal/enode/ ./internal/merge/ ./internal/lower/ ./internal/contract/ ./cmd/enode/` 를 스크래치 파일로
      떠 둔다 (Step 20 이 그 차이로 새 시험의 -run 을 짓는다). `GOOS=windows go test -c -o /dev/null ./internal/enode/` 의 오류 목록도 떠 둔다
      (지금은 `overlay_test.go:41` 의 셋) — 이름 620 · windows 시험 컴파일 오류는 `overlay_test.go:41` 의 셋 그대로
- [x] 기준선 흔들림 — node 패키지 넷 (`internal/enode` · `lower` · `merge` · `scratch`) 을 함께 `-count=1 -json` 으로 다섯 번 · 빨간 시험의
      이름을 모은다. `TestRuncOverlayOpenIncludesHelperStderr` 혼자 `-count=300` 의 빨강 수를 적는다 (4절 31번 전의 값 · 3절 측정 6)
      — 다섯 번 중 한 번 빨강 (셋째 · `TestRuncOverlayOpenIncludesHelperStderr` 하나) · 혼자 `-count=300` 에 빨강 4
- [x] SunnyVM 이 닿는지 본다 (ConnectTimeout 25 · 읽기만) — Step 18 의 자리 — 닿는다 (2026-09-30 · 커널 7.0.0-31 · 8코어 · git 2.43.0 ·
      runc 1.3.4 · apparmor_restrict 0 · `~/bake-it-*` 0)

### Step 2 — 원인 코드와 build 칸 (FD 엔티티 6절 · 물음 1 답 B · 결정 11 · 35 · 54)

- [x] `internal/contract/result.go` — `ReasonIRMismatch = "ir_mismatch"` 와 주석 (완주한 결과에 실리는 첫 원인 코드) · `BuildManifest` 끝에
      `HeadTags []string` (`json:"head_tags"` · omitempty 없음 — null 과 [] 가 다르다) · `IR` 칸의 주석 (맞았을 때 그 값 · 아니면 null)
      — 더했다 · build · vet 초록
- [x] `internal/store/claim.go:963` — reason 목록에 `contract.ReasonIRMismatch` — 더했다 (`:962` ~ `:964`)
- [x] 시험 — `TestBuildManifest_HeadTagsNullAndEmpty` (`result_test.go` · nil 은 null · 빈 슬라이스는 []) · `TestStepResult_ReasonsAreInVocabulary`
      에 한 줄 (`store/exited_test.go`) · `TestMetadataRecordsMatchTheContract` 가 그대로 초록 — 셋 초록.  `TestResultVocabulary_FieldNames` 의
      build manifest 줄에 head_tags 를 더했다 (칸을 더하면 그 줄이 깨져야 하는 시험이다)

### Step 3 — 시작 전 확인의 마운트 줄 (FD 규칙 10절 · FD 엔티티 9절 · 받는 일 34 · 4절 17번)

- [x] `merge.go` — `CheckMount Check = "mount"` (묶음 끝) — 더했다
- [x] `decide.go` — `checkMounts(upper, lower, trash uint64) error` (`checkDevices` 옆 · 문장은 FD 규칙 10절 그대로) — 더했다
- [x] `merge_linux.go` — `resolve` 가 셋마다 statx 로 마운트 번호를 읽고 `checkDevices` 다음에 `checkMounts`. 푼 뿌리 `roots` 에 마운트
      번호 셋의 칸을 더한다 — 지금 `roots` 는 경로 셋뿐이다 (`merge_linux.go:54` ~ `:57`) — `roots.mounts [3]uint64` · statx 가 실패하거나
      mask 에 번호가 없으면 `merge preflight: <path>: ...` 의 보통 오류
- [x] 시험 — `TestCheckMounts` (`decide_test.go` · 셋이 같으면 nil · 하나라도 다르면 `*PreflightError{Check: mount}` 과 번호 셋이 든 문장) ·
      `TestPreflight` 에 줄 — 한 마운트의 임시 폴더 셋이 통과하고 `resolve` 가 채운 `roots` 의 마운트 번호가 0 이 아니고 셋이 같다. bind 별칭
      장면은 namespace 가 들어 Step 18 이다 — 둘 다 초록 (`TestPreflight/reads_one_mount_id_for_the_three_roots`)
- [x] `internal/merge` 커버리지가 80% 이상 (89.1% 기준) — 패키지 혼자 88.6% (못 닿는 두 갈래는 statx 실패와 번호 없는 커널 · CI 명령의
      값은 Step 20)

### Step 4 — 노드 Step 의 칸 넷 · contractStep · Result 의 칸 둘 (FD 엔티티 1 · 6절 · 받는 일 7 · 9 · 11 · 19 · 20 · 4절 2 · 24번)

- [x] `claim.go` — `Step` 에 `Sync` · `Builds` · `IR` · `Merge` (이름과 모양은 `store.Claimed:187` ~ `:190` 과 같다) · `Result` 에 `Build` ·
      `Merge` (`store.StepResult:934` ~ `:935` 와 같다). `Worker.Bake *Baker` 는 Step 9 (Baker 가 그때 생긴다) — 더했다
- [x] `finalize.go` — `contractStep` 이 넷을 옮긴다. build · merge 종류는 `Run` 을 채우지 않는다 — 넷을 옮기고 종류 셋을 switch 로 나눴다
- [x] `bake_test.go` (모든 플랫폼) — `TestContractStep_CarriesTheBakeFields` (merge.wait 이 `MergeWait` 에 · 없으면 4시간 · build 의 `Budgets`
      는 그대로) · `TestClaim_MergeWaitReachesTheNode` (조각 5 의 (1) — `store.Claimed` 에 merge.wait "2s" 와 sync · builds · ir 을 싣고
      JSON 으로 옮겨 `Step` 으로 풀면 `contractStep(step).MergeWait()` 가 2초 · 칸 셋이 같다) · `TestResult_TheBakeFieldsUseTheMediatorNames`
      (`build` · `merge` 의 JSON 이름 · 없으면 안 나간다) — 셋 초록.  셋째는 `store.StepResult` 로 되읽어 Mediator 쪽 이름과도 댄다

### Step 5 — 순수 규칙 (`bakerule.go` · FD 규칙 3.1 · 4 · 7.1 · 8 · 12.4 · 14 · 16절 · FD 엔티티 4 ~ 7 · 11절 · 4절 8 · 10 · 11 · 20 · 21 · 22번)

- [x] `probeScript` · `parseProbe` · `irVerdict` (판정 넷과 문장 · 태그 목록은 빈칸으로 잇고 없으면 `none`) · bash 확인의 exit 갈래 ·
      `attemptReason` · `abandonedReason` · `buildManifestOf` · `mergeOpsOf` · `metadataOf` · `waitLog.lines` · `pendingShape` · `stripPassword` ·
      `repoIDOf` (세션 안 대조의 출력 url · branch · mode 만 쓴다 — `DetectRepo` · `detectRepoManifest` · `gitIn` 을 부르지 않는다 · 3.1) ·
      `finished` (FD 규칙 12.4) · `mergeClaim` (FD 규칙 7.1 의 표를 위에서부터) · 단계 로그 줄 · bake_in_progress 두 문장 (주인이
      없으면 `run unknown`) · `Draft` 타입 (FD 엔티티 4절) · merge-helper 의 주고받는 타입 (`mergeHelperRequest` · `mergeHelperResponse`) 과
      오류 타입 (`helperPreflightError` · `helperOpError`) · 응답을 오류로 되돌리는 `helperErrorOf` (4절 16번 · 1절 크로스 빌드)
      — 모두 `bakerule.go` 에 · 이름 목록 밖의 작은 도우미 (`helperResponseOf` · `joinCause` · `lastLine` · `leftText` 등) 는 code-summary 에
      적는다.  대조의 exit 125 는 FD 의 「git exited <n>」 대신 `cannot verify ir: cannot enter .repo/manifests` (git 이 낸 것이 아니다)
- [x] `bakerule_test.go` (모든 플랫폼 · 표 시험 — FD 흐름 8.1 의 「순수 함수」 목록) — 시험 열일곱 초록 (TestIRVerdict · TestParseProbe ·
      TestLastLine · TestBashCheckText · TestAttemptReason · TestBuildManifestOf · TestMergeOpsOf · TestMetadataOf · TestWaitLog · TestLeftText ·
      TestPendingShape · TestStripPassword · TestRepoIDOf · TestHelperErrorOf · TestFinished · TestMergeClaim · TestBakeTexts).  pendingShape 는
      scratch 가 `/` 인 경로도 모양 밖으로 본다
      irVerdict — 맞음 (태그 둘 · annotated · 가벼운) · 로컬에 없음 · 다른 커밋 · 태그 없음은 `none` · git 실패 · mode none — 문장 글자까지
      parseProbe — 3절 측정 3 의 출력 다섯 (origin 없음은 url "" · detached 는 branch "" · 줄 끝 공백 · 모르는 줄은 무시)
      attemptReason — 원인 코드 · 명령 exit (`sync exited 1` · `build config-a exited 2`) · 대조 · 멈춤 · abandoned (결정 49 의 글자)
      buildManifestOf · mergeOpsOf — 칸 옮기기 (replaced = Replaced + TypeChanged 등 일곱) · 대조를 못 했으면 head "" · head_tags null ·
        태그가 없으면 []
      metadataOf — merge 단계 (resumed false · node 는 이 노드) · 재개 (resumed true · node 는 초안의 Node · previous_ir 은 초안) · 칸 전부 —
        source 여덟 (url · branch · repo_id · head · ir · pinned · sync_command · synced_at) · builds · environment · workspace_target ·
        bake 다섯 (run · node · merged_at · resumed · previous_ir) 을 초안과 인자에서 옮긴다 (FD 규칙 3.1 · 팩 결정 3-16 · scene-gates.md
        조각 6 의 「필드가 다 있다」). 반사로 빈 칸이 없는지도 본다 (TestMetadataRecordsMatchTheContract 와 같은 방법)
      waitLog (TestWaitLog) — 처음 · 모습 바뀜 · 5분 · Unnamed · 재개는 마감 줄 없음 · 4시간에 모습이 안 바뀌면 48 줄 (가짜 시계) · 남은
        시간의 글자 `(3h52m left)` (끝의 0s 를 뗌) · 1분 아래는 초
      pendingShape (TestPendingShape) — 맞는 모양 · 다른 키 · 상대 경로 · upper 가 아닌 끝 · .. 이 든 경로 · Clean 이 아닌 글자 · 위로 세
        단계 조각이 pending 이 아닌 경로 · 키는 열다섯 자리 (`lower.Key{FSID: 0x35b60f8473d0c15, Ino: 2}.String()`) 와 열여섯 자리 둘 다
        (5절 「키 글자를 박지 않는다」)
      stripPassword (TestStripPassword) — https://u:p@h/x -> https://u@h/x · scp 모양 그대로 · 비밀번호 없는 url 그대로 · 못 푸는 scheme 모양은 ""
      repoIDOf — repo 모양 + 브랜치 · repo 모양 브랜치 없음 · git 모양 · url 빈 값 (DetectRepo 의 규칙을 대조 출력에 옮긴 값 · 부르지는 않는다)
      helperErrorOf — kind preflight 는 `*helperPreflightError` (Check 와 merge 의 문장 그대로) · op 는 `*helperOpError` · io 는 보통 오류 ·
        오류가 없으면 nil
      finished — bake.run 과 synced_at · head 가 모두 같을 때만 참 · run_id 만 같으면 거짓 (다시 쓴 run_id · FD 규칙 12.4 끝)
      mergeClaim — FD 규칙 7.1 표의 모든 줄 · 재시작 두 줄이 합칠 것 없음보다 먼저 · 주인이 이 Run 인 building 과 building 을 치운 abandoned
        기록은 합칠 것 없음 (결정 52) · last_attempt 의 abandoned pending 은 지금 phase 와 무관 (결정 49)
      bash 확인 — exit 0 통과 · 1 · 127 은 문장 (결정 45) · -1 은 세션 오류

### Step 6 — 대기 자리와 초안 (`bakefile.go` · FD 엔티티 4절 · FD 규칙 2 · 3.1 · 9 · 12.2절 · 결정 5 · 33 · 46 · 4절 7번)

- [x] 대기 자리 만들기 (`<scratch>/pending/` 과 `<lower 키>` 는 `MkdirAll` 0700 · 그 아래 `MkdirTemp` 0700 · 4절 32번) · 초안 쓰기 · 초안
      읽기 (기대 uid 를 인자로 받는다 · 주인은 `fileOwner`) · 대기 자리를 그 자리의 scratch 의 trash 로 옮기기 (pending_upper 에서 네 단계 위 ·
      없는 자리는 옮겨진 것 — `scratch/trash.go:54` ~ `:60` 의 ENOENT) — `makePending` · `writeDraft` · `readDraft` · `discardPending` (없는
      자리는 Lstat 으로 먼저 보고 trash 를 만들지 않는다) · 기동 청소의 `sweepPending` · 두 읽기가 함께 쓰는 `openPlain`
- [x] `readPinned` (4절 30번 — Lstat 이 보통 파일 · `pinnedMax` 안 · SameFile · LimitReader) 와 `fileOwner` 의 linux · 그 밖 짝
      (`mergehelper_linux.go` · `mergehelper_other.go`). `bakefile.go` 는 `syscall` 을 가져오지 않는다 (Step 16 의 검사) — 그대로.  두
      helper 파일은 Step 10 이 채운다
- [x] `bakefile_linux_test.go` — `TestDraft_IsPrivate` (0600 · 되읽기 · `pending/` 과 `<키>` 가 0700) · symlink · 보통 파일이 아님 · 1 MiB 넘음 ·
      schema 가 1 이 아님 · 주인이 다름 (기대 uid 로 다른 값을 넘긴다) 을 거절 · scratch 둘 — 다른 scratch 의 대기 자리는 그 scratch 의 trash 로 ·
      없는 자리는 옮겨진 것. `TestReadPinned` — 보통 파일의 sha256 · symlink (호스트의 다른 파일을 가리킨다 — 그 내용을 읽지 않는다) · FIFO
      (`syscall.Mkfifo` · 막히지 않고 곧바로 오류) · `pinnedMax` 넘음 (Truncate 로 만든 성긴 파일) · 없음 — 모두 `waitFor` 의 상한 안에 돌아온다
      — 넷 초록 (`TestDraft_IsPrivate` · `TestDraft_RefusesWhatItShouldNotRead` 여섯 · `TestDiscardPending_GoesToItsOwnScratch` ·
      `TestReadPinned`) · `-race` 초록

### Step 7 — 세션의 닫기와 수확 — 닫기가 upper 를 남기고 격리 노드의 diff 는 세션 안에서 (`runc_overlay_linux.go` · `diff.go` · FD 규칙 3절 「닫기」 · 받는 일 21 · 4절 6 · 29 · 31 · 34번 · CG 물음 5 답)

- [x] `Close(ctx, keep)` — keep.Upper 가 있으면 `cmd.Wait` 뒤 · release 앞에 `unix.Renameat2(…, RENAME_NOREPLACE)`. 실패 · aborted 의 오류
      (4절 6번). 주석 (:478 ~ :485) — 고쳤다 · 오류 글자는 `keep the upper: <원인>` 과 aborted 의 한 줄
- [x] `runtime.go` · `lower.go` 의 주석 (4절 29번) — `Keep` · `FinalizeSpec.Diff` · `State.PendingUpper` 의 주석 셋
- [x] `runc_overlay_linux_test.go` (가짜 helper 로 여는 세션 · `protocolTestRuntime`) — 가짜 helper 는 upper 를 만들지 않으므로
      (`runc_overlay_linux.go:901` ~ `:908` 은 helper 안의 코드다) 시험이 Open 뒤에 `<runRoot>/upper` 와 그 안의 파일을 만든다.
      `TestRuncOverlayCloseKeepsTheUpper` (upper 의 파일이 keep 자리로 · 나머지 runRoot 는 trash · 세션 잠금이 풀린다) ·
      `TestRuncOverlayCloseKeepAfterAnAbort` (오류 · upper 는 trash 에) · `TestRuncOverlayCloseKeepOntoAnExistingPath` (keep 자리에 빈 디렉터리가
      이미 있다 — os.Rename 이면 덮고 성공하는 모양 · 오류 · upper 는 trash 에 · 빈 디렉터리는 그대로) · `TestRuncOverlayCloseKeepOnlyOnTheFirstClose`
      (manageSession 으로 감싼 세션에서 `Keep{}` 로 먼저 닫으면 뒤의 `Keep{Upper}` 는 옮기지 않는다 — runBuildStep 이 첫 부름을 지키는 까닭) ·
      오늘의 `TestRuncOverlayCloseMovesTheSessionToTrash` 가 그대로 초록 — 다섯 초록
- [x] 기준선 흔들림의 원인 (4절 31번) — 시험 helper 의 eof 갈래를 `os.Exit(1)` 로 · `Open` 이 helper 응답을 못 받은 길은 abort 뒤에 stderr
      꼬리를 붙인다. `TestRuncOverlayOpenIncludesHelperStderr` 혼자 `-count=300` 에 빨강 0 (Step 1 의 값과 댄다) — 둘을 고친 뒤 300 에 빨강 1 이
      남았다 (`send runtime helper request: write |1: broken pipe` — 먼저 죽은 helper 에 open 요청을 쓰는 길).  그 길도 abort 뒤 stderr 를 붙이게
      고쳤다 (셋째 원인 · 같은 뿌리).  고친 뒤 `-count=300` 빨강 0 · `-coverpkg=./...` 로도 300 에 빨강 0 (Step 1 의 4 와 댄다).  가짜 helper 는
      모든 갈래가 `defer os.Exit(0)` 로 끝난다
- [x] 확인 — `runc_overlay_other.go` · `nativeSession.Close` 는 `Keep` 을 안 본다 (안 고친다) — 확인했다 (linux 밖에는 세션이 없고 native 는
      once 만 돈다)
- [x] 격리 노드의 Finalize diff (4절 34번) — `diff.go`: `repoDiff` 안의 스크립트 둘을 `repoDiffScript` · `repoStatScript` 로 올린다 (native 의
      `workspaceDiff` 바이트는 그대로) · `sessionDiffScript` · `sessionDiffError` (exit 와 stderr 마지막 줄 -> DiffError 글자) · `placeWorkspaceDiff`
      (CreateTemp · fsync · rename) · `diffLimit`. `runc_overlay_linux.go`: `h.finalize` 가 `spec.Diff` 를 끈다 · `runcOverlaySession.Finalize` 가
      helper 의 답 뒤에 `diffInSession` (안쪽 run · 세는 버퍼 · 넘으면 stat 한 번 더 · 마감과 aborted 의 갈래) · `runtime.go` 의 `FinalizeSpec.Diff`
      주석. 오늘의 `diff_test.go` 열하나 (native · 호스트 git) 가 그대로 초록 — 했다.  바이트를 짓는 몫은 `sessionDiff` (diff.go · 모든
      플랫폼) 로 떼어 세션과 시험이 같이 쓴다 · 호스트도 helper 에 보내는 요청의 Diff 를 끈다 (helper 도 끈다 — 둘 다) · 요약 (stat) 은 호스트처럼
      자르지 않는다 (상한 maxBlobBytes) · 열하나 초록
- [x] 시험 (`runc_overlay_linux_test.go`) — `TestRuncOverlayFinalizeMakesTheDiffInTheSession` (가짜 helper 에 모드 하나를 더한다 — run 의 argv 가
      `sh -c sessionDiffScript` 면 정해 둔 stdout · stderr · exit 로 답하고, finalize 요청의 Diff 가 false 인지 적어 둔다. 본다 — helper 가 받은
      Diff 는 false · 세션의 Run 한 번 · 환경은 ENODE_DIFF_MODE 와 repo 스크립트뿐 · 호스트 $OUT 의 workspace.diff 가 stdout 바이트와 같다 ·
      $OUT 에 미리 둔 `workspace.diff` symlink (호스트 파일을 가리킨다) 는 따라가지 않고 바뀌며 그 호스트 파일은 그대로 · 그 이름이 디렉터리면
      `cannot place workspace.diff:` · `diffLimit` 을 줄이면 stat 으로 한 번 더 돌고 머리에 전체 길이 · exit 125 · 124 · 121 · 122 · 3 · -1 의 DiffError
      글자 · 그때도 Finalize 는 오류가 아니다 · helper 의 답이 마감이면 diff 를 안 돈다 · 도는 중 마감이면 DeadlineExceeded) ·
      `TestRuncOverlayHelperFinalizeRunsNoHostGit` (PATH 표지의 가짜 git · repo · git 모양과 repo 모양 트리에 `h.finalize` 를 Diff true 로 부른다 ·
      표지가 없다 · DiffBytes 0) · `TestRuncOverlayHelperFinalizeDoesNotRunTheTreesConfig` (진짜 git · git 환경은 5절대로 · 트리의 `.git/config` 에
      core.fsmonitor 와 filter.<이름>.clean (`.gitattributes` 로 건다) 이 호스트 표지 파일을 쓰는 스크립트 · `h.finalize` 뒤 표지가 없다 · 대조군 —
      같은 트리에 호스트 `workspaceDiff` 를 부르면 표지가 생긴다 — 오늘의 구멍이 있었다는 것과 fixture 가 살아 있다는 확인) ·
      `TestSessionDiffScript_MatchesTheHostDiff` (호스트 sh 로 `sessionDiffScript` 를 그 트리에서 돌린 stdout 과 `workspaceDiff` 의 바이트가 같다 —
      고친 파일 · 새 파일 · 지운 파일 · 이진 · 한국어 경로 · 상한을 넘은 요약 · 둘 다 LC_ALL=C · repo 모양은 PATH 의 가짜 `repo` (프로젝트
      둘을 돌며 REPO_PATH 를 넣고 -c 의 글을 sh 로 돈다) 로 호스트 `repoDiff` 와 댄다). native 는 그대로 — 오늘의 `diff_test.go` 와
      `TestFinalizeLocal_DiffAndTheDeadline` (`finalize_test.go:506` · `nativeSession.Finalize` 가 Diff true 로 호스트 git 을 돈다). t.Skip 을 안 부른다 (5절 규칙 10)
      — 넷 초록 (첫째는 하위 시험 열넷 · 가짜 helper 는 받은 요청을 파일에 적고 세션 안 diff 에 파일로 답한다) · 둘째와 넷째의 대조군이 표지를
      남겨 fixture 가 살아 있다 · native 두 시험 초록

### Step 8 — LowerGuard 의 OnStale · Dir (`lowerguard.go` · FD 엔티티 10절 · FD 규칙 13절 · 되물음 1 답 A · 결정 19 · 40 · 4절 18번)

- [x] `OnStale(f func(lower.State))` · `Dir() *lower.Dir` · `sourcesLocked` 의 부르는 줄 — f 를 곧바로 부른다 (고루틴을 띄우지 않는다 · 4절 18번)
      — 더했다 (`LowerGuard.onStale` 칸 하나 · 부르는 줄은 ReadState 가 된 뒤 · 둘 다 nil 수신자에 안전)
- [x] `lowerguard_linux_test.go` — `TestLowerGuard_OnStaleSeesStaleStates` (building · pending · merging 이면 그 state 로 부른다 · committed 면 안
      부른다 · `g.bake` 가 있으면 안 부른다 · f 가 nil 이면 안 부른다 · ReadState 오류면 안 부른다 · BeforeAdvert 가 돌아오기 전에 부른 수가
      1 이다 — 곧바로 부른다) · `TestLowerGuard_Dir` (못 열었으면 nil · 연 자리) · 오늘의 guard 시험 그대로 초록. 막지 않는 것은 f 쪽의
      일이라 Step 13 의 `TestOnStale_ReturnsAtOnce` 가 본다 — 둘 초록 · guard · drain · lowerclaim 시험 스물일곱 그대로 초록

### Step 9 — Baker · 치우는 몸통 · 정리 (`bake.go` · FD 엔티티 2 · 3 · 10절 · FD 규칙 11 · 12.1 · 12.2 · 13 · 14절 · FD 흐름 4 · 5절 · 결정 15 · 25 · 28 · 29 · 33 · 34 · 42 · 44 · 46 · 49 · 53 · 4절 12 · 13 · 19 · 20번)

- [x] `Baker` · `StartBaker` (첫 줄이 `guard.OnStale(b.onStale)` · Dir 이 nil · TryBake 오류 · ReadState 오류면 로그 한 줄로 기동 정리를 건너뛴다 ·
      guard 가 nil 이면 nil) · `Wait` (`closed` 를 적고 wg 를 기다린다 · 4절 18번) · `heldBake` · `abandon` · `draftBuilds` · `startMerging` ·
      `cancelMerging` · `clean` (building · pending 줄은 pending_upper 의 모양을 먼저 본다 · 4절 19번). `onStale` · `resume` 은 겉면만 —
      아무것도 안 한다 (몸통은 Step 13). `claim.go` 의 `Worker.Bake *Baker` — 모두 `bake.go` 에.  `clean` 은 키를 읽으려고 `*lower.Dir` 을
      첫 인자로 더 받는다 (`clean(dir, lock, st, from)`) · 몸통 밖으로 놓는 길 (합침 · merging 뒤 멈춤 · 7.1 어긋남) 은 `letGo` 하나 ·
      재개를 배경에 여는 `openResume` · state.json 쓰기는 root 시험이 바꿔 끼우는 `writeLowerState` 함수 값
- [x] 공용 틀 `bake_fake_linux_test.go` (4절 23번) — 가짜 런타임 (`Capability().Writes` isolated · argv 마다 exit · stdout · stderr · 지연 ·
      받은 ProcessSpec 기록 · `Close(Keep{Upper})` 에서 rename) · 진짜 lower 자리 (`newGuardFixture` 모양) · 가짜 Mediator 잇기 · 잠금 핸들을
      t.Cleanup 으로 놓는 도우미 (5절 규칙 9) — `fakeRuntime` · `bakeFixture` · `holdBakeLock` · `putState` · `hold` · `failStateWrites` 등
- [x] `bake_linux_test.go` — 넷 초록 (`TestHeldBake_OneBodyUnderRace` · `TestHeldBake_Body` 일곱 · `TestStartBaker` 여덟 ·
      `TestBakerClean_AnOddPendingPathIsLeft` 셋) · `-race` 초록 · 몸통 시험 `-race -count=10` 초록.  모양 밖 경로 가운데 「upper 가 아닌 끝」 은
      다른 scratch 에 둔다 — 자기 scratch 의 pending/<키>/ 아래는 모양과 무관하게 버려진 자리라 기동이 치운다
      몸통 — 고루틴 서른둘이 abandon 과 startMerging 을 함께 불러도 몸통은 한 번 (`TestHeldBake_OneBodyUnderRace` · `-race`) · startMerging 뒤의
        몸통은 아무것도 안 한다 · cancelMerging 뒤의 몸통은 돈다 ·
        몸통 뒤의 startMerging 은 false · 대기 자리를 못 옮겨도 committed 를 쓴다 · committed 를 못 써도 잠금을 놓는다 (결정 28 — 쓰기 실패는
        자리 디렉터리 0500 · root 면 함수 값) · held 는 그 굽기일 때만 비운다 · DropBake 가 불린다 (guard 의 bake 가 빈다) · last_attempt 칸
      기동 (FD 규칙 12.2 의 표) — committed (자기 scratch 의 `pending/<키>/` 남은 것을 trash 로) · building · pending (기록된 자리를 그 scratch 의
        trash 로 · 나머지를 자기 trash 로 · committed + abandoned · 놓는다) · TryBake 가 안 됨 (아무것도 안 함) ·
        TryBake 오류 (로그 한 줄) · 어느 경우든 OnStale 이 등록돼 있다 · Dir 이 nil (로그 한 줄) · guard 가 nil 이면 Baker 가 nil.
        merging 줄 (재개를 배경에 연다) 은 재개를 짓는 Step 13 이 본다
      정리의 모양 확인 (`TestBakerClean_AnOddPendingPathIsLeft`) — pending_upper 가 모양 밖의 경로 (다른 lower 키 · pending 이 아닌 조각 ·
        upper 가 아닌 끝) 인 building · pending 이면 그 경로의 파일이 그대로 남는다 · committed + abandoned · 노드 로그에 경로 한 줄 (FD 규칙
        12.2 끝 줄 · 16.3)

### Step 10 — merge-helper (`mergehelper_linux.go` · `mergehelper_other.go` · FD 엔티티 8 · 12절 · FD 규칙 9절 · 받는 일 23 ~ 26 · 32 · 4절 16번)

- [x] `mergehelper_linux.go` — `RunMergeHelper` (stdin 한 줄 · stdout 한 줄 · `Options.Discard` 는 `scratch.Trash{Dir: trash}.Move` · exit 0 과 1) ·
      `mergeHelperArgv` · `mergeHelperCommand` · `callMergeHelper` (띄우고 stdout 의 첫 줄만 읽는다 · stderr 꼬리는 `cmd.Wait()` 뒤에 읽는다 ·
      응답은 `helperErrorOf` 로 오류로 되돌린다). `mergehelper_other.go` — `RunMergeHelper` (exit 1) · `callMergeHelper` (지원 안 함의 오류) 의 짝
      (4절 16번 · 1절 크로스 빌드) — 했다 · `callMergeHelper(req) (merge.Result, error)` · 첫 줄 뒤의 stdout 은 버리고 Wait 한다 · 크로스 빌드 셋 초록
- [x] `mergehelper_linux_test.go` — `TestMergeHelperArgv` (unshare 모양 · `--mount` 없음 · 바꿔 끼우기 전의 제품 기본값 `mergeHelperCommand` 가
      `mergeHelperArgv(os.Executable())` 와 같고 unshare 로 시작한다 — 시험이 바꿔 끼운 명령만 보면 기본값을 unshare 없는 명령으로 바꿔도
      초록이다) · `TestMergeHelperProcess` (시험이 띄우는 helper · 가짜 갈래와 진짜 · **모든 갈래를 os.Exit 로 끝낸다** — return 으로 끝나면 시험
      바이너리가 stdout 에 PASS 와 coverage 줄을 찍는다 · 4절 31번) · preflight 통과 · preflight 어긋남 (kind preflight · Check · 호스트의
      `helperPreflightError` 가 merge 의 문장 그대로) · apply 가 임시 트리를 합치고 lower 쪽 항목을 항목마다 trash 로 (같은 끝 조각은 `-1` · `-2`) ·
      op 오류 (kind op) · io 오류 · JSON 이 아닌 요청 (exit 1) · 답이 없음 (helper 가 stderr 에 쓰고 죽음 · stderr 꼬리가 든 문장 — `-count=300` 에
      빨강 0 으로 확인) · helper 의 stderr 가 stdout 의 줄을 흐리지 않는다 — 넷 초록 (`TestMergeHelperProcess` · `TestMergeHelperArgv` ·
      `TestMergeHelper_Calls` 열 · `TestRunMergeHelper_ABadRequest`) · 답이 없는 갈래 `-count=300` 빨강 0 · `-coverpkg` 로 `-count=100` 빨강 0.
      합치기 트리는 종류가 바뀐 둘과 opaque 하나로 lower 쪽 셋을 버린다 (같은 끝 조각은 same · same-1)

### Step 11 — build 단계 (`bake_build.go` · claim.go 의 분기와 `closeOut` · finalize.go 의 settle · FD 규칙 1 · 2 · 3 · 3.1 · 4 · 5 · 6 · 11 · 14 · 15 · 16절 · FD 흐름 2 · 7절 · 4절 2 ~ 5 · 8 · 9 · 10 · 12 · 27 · 28 · 30번)

- [x] `runBuildStep` — 첫 줄의 `WorkspaceWrites(w.Runtime)` 확인 (4절 2번) · FD 규칙 3절 표의 1 ~ 11 · 실패 끝 · 노드 쪽 오류의 몸통 · `execute` 의
      분기 · `closeOut` 과 `closing` (4절 3번 — `after(deadline, closeErr, finErr) (late, err)` · `FinalizedAt` 은 after 뒤) · `settleIn.sealErr` ·
      성공 끝의 첫 Close 가 `Keep{Upper}` (4절 6번) · pinned 의 sha256 은 `readPinned` (4절 30번). held 를 둔 직후의 오류 길은 Baker.mu 를 놓은
      뒤 몸통을 부른다 (4절 12번). 단계 로그와 진행 청크에 host 경로 (대기 자리 · scratch) 를 쓰지 않는다 — 노드 로그에만 (FD 규칙 16.2 끝)
      — `bake_build.go` 에 (`takeForBuild` · `buildRun` 의 `command` · `nodeCommand` · `stopEarly` · `failEnd` · `succeed`).  `closeOut` 은
      단계 로그를 업로드할 때 읽도록 `logBody func() []byte` 를 받는다 — after 가 쓴 pending 줄이 단계 로그에 들고 진행 청크의 꼬리는 after 가
      비운다.  readPinned 실패의 exit_code 는 마지막 build 다 (4절 30번의 「sync 의 것」 은 builds 앞의 `repo manifest -r` 실패 줄과 겹쳐 적힌
      것 · 28번의 원칙대로) — code-summary 의 계획과 다른 자리
- [x] `bake_build_linux_test.go` (가짜 런타임 · 진짜 lower · 가짜 Mediator) — 초록 (분기 · 거절 셋 · 상태를 못 읽음 셋 · 잠금 넷 · 성공 ·
      마감 직전 pending · repo 모양 · pinned 파일 바꿔치기 둘 · 단계 로그 둘 · 실패 표 열아홉 · 데몬 멈춤 · held 동안의 오류 둘) · `-race` 초록.
      실패 표의 줄마다 다음 굽기가 bake_in_progress 로 거절되지 않는 것까지 본다
      분기 — kind build · merge 가 빈 argv 확인에 안 닿는다 · 거절 문장이고 세션을 안 연다 (`TestBakeBuild_ANodeWithoutAnUpperRefuses` — Bake 가
        nil · Bake 가 있고 Runtime 이 nil · Bake 가 있고 `NativeRuntime{}` 의 세 줄. 뒤의 둘은 NativeRuntime 으로 떨어뜨리는 되돌림을 잡는다) ·
        Dir 이 nil 이면 `cannot open the lower state directory` · TryBake · ReadState 오류가 bake_in_progress 가 아니고 잡은 잠금을 놓는다 (결정 29)
      잠금 — 다른 Dir 이 bake.lock 을 쥐고 state 가 pending 이면 bake_in_progress 와 주인 문장 (`TestBakeBuild_AnotherBakeHoldsTheLower`) · held · 재개
        중이면 bake_in_progress · 낡은 building · pending 은 정리 뒤 진행 (옛 대기 자리는 그 scratch 의 trash · abandoned) · 낡은 merging 줄은
        재개를 짓는 Step 13 이 본다
      성공 — 명령의 차례 (bash 확인 · sync · 대조 · builds) · 단계 로그의 머리와 끝 줄 · exited 한 번 (마지막 build) · building 을 쓸 때 owner ·
        pending_upper · since 가 적힌다 (FD 규칙 2절 — pending_upper 를 building 때 적는다) · pending (owner · pending_upper · since · 앞의
        last_attempt 를 그대로) · 초안의 칸 전부 (0600 · source 여덟 — url 은 비밀번호를 지움 · branch · repo_id · head · ir · pinned ·
        sync_command · synced_at · builds · environment · workspace_target · run · node · previous_ir — 못 읽으면 null 과 경고 한 줄) · HoldBake 뒤
        공유를 안 잡는다 · 올라간 이름은 manifest 하나 · build 칸 · DONE exit 0 · 마감 직전에 pending 을 쓴 build 는 finalize ok 인 DONE (after 가
        late false · pending 과 HoldBake 가 살아 있다)
      repo 모양 — pinned 의 argv · sha256 을 닫은 뒤 대기 자리에서 읽는다 · builds 가 pinned 파일을 symlink 나 FIFO 로 바꾸면 제한 시간
        (`waitFor` 의 상한) 안에 FAILED `cannot pin the manifest:` · 몸통 · committed (`TestBakeBuild_APinnedFileThatIsNotRegularFails` · 4절 30번)
      단계 로그 (`TestBakeBuild_TheStepLogCarriesNoHostPath`) — 성공 · 실패 끝의 단계 로그 버퍼와 가짜 Mediator 가 받은 진행 청크 어디에도 scratch
        경로 (대기 자리) 가 없다. 노드 로그에는 있다
      실패 (FD 규칙 6절 표 · 줄마다 부분 시험 하나) — bash 확인 exit 1 · 127 · -1 · 확인 중 임대 끝 · 세션을 못 엶 · sync 가 0 아님 (builds 안 돎 ·
        남은 구성마다 skipping 줄 · DONE exit 은 sync 의 값 · last_attempt `sync exited 1` · committed · upper trash) · build 하나가 0 아님 (첫 실패에서
        멈춤 · last_attempt 의 builds 는 거기까지) · IR 로컬에 없음 · 다른 커밋 (DONE · exit 0 · reason ir_mismatch · 문장은 단계 로그 끝 · 구성마다
        skipping 줄 · head · head_tags · manifest 없음 · last_attempt ir_mismatch · exited 는 sync 의 것 · 결정 22) · 대조를 못 함 (FAILED · head "" ·
        head_tags null · exited 와 exit_code 는 sync 의 것) · pinned 실패 (FAILED · exited 와 exit_code 는 sync 의 것 · 결정 22) · 명령 중 임대 만료
        (FAILED aborted · exited 없음 · exit_code 없음) · 명령 중 helper 죽음 (`runtime run:` · exited 없음 · exit_code 없음 · 결정 48) · keep 의
        rename 실패 (FAILED `runtime cleanup: keep the upper:` · pending 을 안 씀 · HoldBake 없음 · 몸통 · committed · exit_code 는 마지막 build ·
        4절 3번) · pending 전 Finalize 마감 (finalize_timeout · pending 을 안 씀 · committed · exit_code 는 마지막 build) · pending 쓰기 실패 (FAILED
        `cannot write the lower state:` · 몸통 · committed · exit_code 는 마지막 build) · pending 뒤 업로드 예산 (upload_timeout · 몸통 · DropBake ·
        exit_code 는 마지막 build) · 초안 쓰기 실패 (FAILED · pending 을 안 씀 · exit_code 는 마지막 build) · 명령 중 데몬 멈춤 (보고 없음 ·
        last_attempt `node stopped`) · bash 확인 · 세션 실패는 exit_code 없음 (4절 28번)
      held 가 있는 동안의 오류 길 (결정 44) — 대기 자리 만들기 · building 쓰기가 실패해도 몸통이 돌고 held 와 g.bake 가 빈다 — 다음 굽기가
        bake_in_progress 로 거절되지 않는다
- [x] `afterExit` 를 나눈 뒤 오늘의 명령 · agent 시험 (`finalize_worker_test.go` 등) 이 그대로 초록 — `internal/enode` 전체 · `cmd/...`
      초록 · `TestSettle` 에 sealErr 두 줄을 더했다

### Step 12 — merge 단계 (`bake_merge.go` · FD 규칙 7 · 7.1 · 8 · 9 · 10 · 11 · 15 · 16절 · FD 흐름 3 · 4 · 7절 · 4절 2 · 11 · 12 · 13 · 14 · 15 · 16 · 22번)

- [x] `runMergeStep` — 첫 줄의 `WorkspaceWrites(w.Runtime)` 확인 (4절 2번) · FD 규칙 7절 표의 1 ~ 16 (6 의 trash 는 `MkdirAll` 0700 · 9 의
      merging 쓰기가 실패하면 `cancelMerging` 으로 표지를 거두고 배타 Release · 몸통 · FAILED — 4절 13번) · 시작 전 확인과 startMerging 사이의
      시험 훅 `beforeStartMerging` (4절 11번) · merging 뒤 멈춤 · metadata 뒤 오류 · 끝나는 셋 · 16 의 업로드에 업로드 예산 (`settle` 의 upload 몫 ·
      4절 15번). 단계 로그와 진행 청크에 host 경로를 쓰지 않는다 — 그물 줄만 예외 (FD 규칙 16.2 끝) — `bake_merge.go` 에.  claim 시각은
      거절 확인보다 먼저 잡는다 (첫 줄).  시험이 끼우는 함수 값 셋 (`foreignMounts` · `writeMetadata` · `movePending`) 을 더했다.  몸통이
      먼저 돌아 startMerging 이 false 면 보고하지 않는다 (FD 흐름 4절 첫째 줄 「보고는 닿지 않는다」) · 노드 로그 한 줄
- [x] `bake_merge_linux_test.go` (진짜 lower · namespace 없는 helper · 가짜 Mediator · 대기 자리는 가짜 런타임의 build 단계가 만든다) —
      초록 (합침 · 기다림 · claim 확인 여덟 · 그물 둘 · 실패 열둘 · 단계 로그 · 조각 5 의 (2) 둘) · `-race` 초록.  Apply 오류는 lower 를 0500 으로
      막는 대신 가짜 helper 의 `apply-fails` 모드로 끼운다 (root 로 돌아도 같은 갈래) · 옮기기 실패는 `movePending` 함수 값
      분기 — Bake 가 있어도 런타임이 isolated 가 아니면 거절 문장 (4절 2번)
      합침 (`TestMergeStep_Merges`) — lower 에 upper 의 항목 · 버린 lower 항목은 trash 에 · metadata 의 칸 전부 (초안에서 옮긴 source 여덟 ·
        builds · environment · workspace_target · bake.run · node · merged_at · resumed false · previous_ir) ·
        committed · last_attempt 지움 · 대기 자리는 committed 뒤에 trash 로 (merge 가 만든 trash 는 0700) · 두 잠금이 풀린다 · DropBake (표지가 새로) · held nil · 올라간 이름은
        merged 하나 · merge 칸의 셈 · `Upload` 칸 `ok` · `Finalize` · `FinalizedAt` · `ExitedAt` · exit_code 없음 · exited 를 안 보낸다
      기다림 — 형제를 흉내 낸 Dir 이 공유를 쥔 동안 단계 로그와 진행 청크에 쥔 쪽 줄 · 형제가 놓으면 합친다
      claim 확인 (FD 규칙 7.1) — 쥐었고 pending 이고 주인이 이 Run 이면 진행 · 쥐었는데 그 밖이면 어긋남 FAILED (state 를 안 쓰고 잠금을 놓고
        DropBake 가 불리고 held 를 비운다 · last_attempt 를 안 남긴다) · 안
        쥐었고 주인이 이 Run 인 pending · merging 이면 재시작 문장 (left for the cleanup) · 안 쥐었고 last_attempt 가 이 Run 의 abandoned pending 이면
        재시작 문장 (discarded) — 다른 굽기가 building 을 써도 · 그 밖 (명령 실패 · ir_mismatch · 다른 굽기 · 주인이 이 Run 인 building · building 을
        치운 기록) 은 합칠 것 없음 DONE · error 없음 · 단계 로그 한 줄 · 재시작 줄이 합칠 것 없음보다 먼저
      재시작 장면 — build 보고 뒤 Baker 를 새로 짓고 (재시작) 기동 정리 뒤 merge claim 이 오면 FAILED 재시작 문장
      명령 실패 뒤의 merge — 몸통이 held 를 비웠으므로 합칠 것 없음 (둘째 줄의 어긋남이 아니다 · FD 흐름 4절 표 여섯째 줄)
      그물 — 찾으면 배타를 놓고 줄을 쓰고 `foreignRetry` 뒤 다시 · 오류면 경고 줄을 쓰고 합친다 · 못 읽은 수 줄 (그물은 함수 값으로 끼운다)
      시작 전 확인이 어긋남 (upper 에 금지 표시) — committed · upper trash · FAILED `merge preflight:` · last_attempt
      몸통이 대기 자리를 먼저 옮김 — preflight 가 directory 로 어긋나고 abandon 은 이미 돌았다 · 배타를 놓고 끝 (FD 흐름 4절 표 다섯째 줄)
      held 가 있는 동안의 오류 길 (결정 44 · FD 흐름 8.1 의 오류 길) — merge 의 ReadState 오류 (몸통 · held 와 g.bake 가 빈다 · FAILED `cannot open
        the lower state directory:`) · trash 만들기 실패 (trash 자리의 부모를 0500 · root 면 함수 값 · 몸통 · FAILED `cannot create the trash of
        the pending upper:`) · merging 쓰기 실패 (표지를 거두고 몸통이 돈다 · committed · 두 잠금이 풀린다 · held 와 g.bake 가
        빈다 · FAILED `cannot write the lower state:`) — 셋 다 다음 굽기가 bake_in_progress 로 거절되지 않는다
      Apply 오류 (lower 의 디렉터리를 쓰지 못하게 해 rename 이 실패) — merging 에 둔다 · 두 잠금이 풀린다 · DropBake 가 불린다 · held nil ·
        FAILED `merge stopped:` · retryAt 을 안 건다
      metadata 뒤 committed 쓰기 실패 — DONE merged · 대기 자리가 남는다 · state 는 merging (재개가 초안과 metadata 로 끝났나를 본다 · FD 규칙
        12.4)
      committed 뒤 대기 자리 옮기기 실패 — DONE merged · 노드 로그 경고 한 줄 · state 는 committed (남은 자리는 기동의 committed 정리가 거둔다 ·
        FD 규칙 7절 끝)
      합친 뒤 업로드 예산 (`TestMergeStep_UploadBudgetFailsAfterTheMerge` · 4절 15번) — 가짜 Mediator 가 `merged` 업로드를 붙잡고 `w.budgets` 로
        업로드 예산을 줄인다 · FAILED · error 는 `upload budget of <예산> exceeded` · reason `upload_timeout` · `Upload` 칸 `timeout` · merge 칸이
        있다 · state 는 committed · metadata 의 bake.run 이 이 Run · 두 잠금이 풀린다 · 노드 로그에 `upload budget exceeded; not uploaded`.
        예산의 값은 계약이 바꾸지 못한다 — 이미 초록인 `TestValidate_MergeStep/budget` (`internal/contract/bake_test.go:173` ~ `:174`) 이 본다
      끝나는 셋 — 기다리는 중 임대가 사라짐 (FAILED aborted · 몸통) · 데몬 멈춤 (보고 없음 · 몸통 `node stopped`)
      startMerging 이 false — 시험 훅 `beforeStartMerging` 이 시작 전 확인과 startMerging 사이에서 광고 응답의 몸통 (abandon) 을 부르면
        startMerging 이 false 라 배타를 놓고 끝 · merging 을 안 쓴다 (FD 흐름 4절 표 첫째 줄 · 시각에 기대지 않는다)
      단계 로그 (`TestMergeStep_TheStepLogCarriesNoHostPath`) — 단계 로그 버퍼와 진행 청크에 대기 자리 경로가 없다 · 그물 줄만 마운트 자리를 보인다
      조각 5 의 (2) (`TestMergeStep_WaitsForTheContractValue`) — 형제 Dir 이 공유를 쥔 채 merge.wait "300ms" · "900ms" 인 merge 단계 둘이 각각
        claim 시각 + wait 에서 merge_wait_timeout 으로 끝난다 — 포기한 순간 (노드 로그의 마감 줄의 시각) 이 시작 시각 + wait 이상이고 + wait +
        300 ms 미만 (5절 규칙 5) · 그 뒤 upper 가 trash · state 가 committed · 기다리는 동안 쥔 쪽 줄

### Step 13 — 재개 · 광고 주기의 정리 (`bake_resume.go` · FD 규칙 10 · 12.3 · 12.4 · 12.5 · 13절 · FD 흐름 6절 · FD 엔티티 10절 · 결정 1 · 6 · 7 · 16 · 26 · 27 · 30 · 42 · 46 · 53 · 4절 11 · 18 · 19번)

- [x] Step 9 가 겉면만 둔 `resume` (FD 규칙 12.3 의 1 ~ 12) · `onStale` (4절 18번 — Baker.mu 아래에서 closed · held · resuming · retryAt 을 보고
      `b.wg.Go` 로 연다 · 막지 않는다) 의 몸통. 재개가 helper 의 preflight 결과를 다루는 길은 merge 단계와 반대다 — 재개는 어긋나도 upper 를
      버리지 않는다 (FD 규칙 10절 표). 두 길이 helper 결과를 다루는 코드를 함께 쓰면 부르는 쪽에서 갈래를 나눈다 — `bake_resume.go` 에
      (`onStale` · 배경 일 `stale` · `resume` · 재개의 배타 `resumeExclusive` · 12.3 의 3 ~ 11 인 `completeMerge` · `noteErr` · `retryLater`).
      helper 를 부르는 코드는 merge 단계와 나눠 두었다 — 갈래가 반대라서.  광고 주기 정리의 committed 쓰기 실패 로그는
      `bake: cannot write the lower state while cleaning a stale bake; trying again in 10m` (FD 16.3 에 없던 글자)
- [x] `bake_resume_linux_test.go` — 초록 (12.4 의 넷과 run_id 만 같은 것 · metadata · 모양 · 시작 전 확인 실패 둘 · 되풀이 로그 · onStale
      이 아무것도 안 하는 여섯 · 곧바로 돌아옴 · 광고 주기의 정리 다섯 · DropBake 안 부름 · Apply 멈춘 뒤 곧바로 · 노드 로그의 대기와 데몬 멈춤 ·
      여는 쪽 (기동 · build claim)) · `-race` 초록
      FD 규칙 12.4 의 넷 — 초안 없음 (실패 · merging 그대로 · retryAt 10분 뒤 · 두 잠금 풀림) · 끝났다 (committed 와 옮기기만) · upper 뿌리가 없음
        (metadata 부터) · upper 뿌리가 있음 (비어 있어도 · Preflight 와 Apply)
      run_id 만 같은 metadata (다시 쓴 run_id) 는 끝나지 않은 것 — Apply 가 돈다
      모양이 틀린 pending_upper — 실패 · 옮기지도 합치지도 않는다 (`TestResume_AnOddPendingPathIsNotMerged`)
      재개의 Preflight 어긋남 (upper 에 금지 표시) — upper 가 남는다 · merging 그대로 · 두 잠금이 풀린다 · retryAt 이 10분 뒤 (FD 규칙 10절 표 ·
        12.3 · 답 2 — merge 단계는 같은 helper 결과에서 upper 를 버린다)
      재개의 Preflight 읽기 실패 (helper 를 못 띄움 · 읽을 수 없는 디렉터리) — 위와 같다 · upper 가 남고 merging 그대로 · retryAt 10분 뒤
      실패가 되풀이 — 같은 오류는 로그 한 번 · 바뀌면 한 번 더
      onStale — held · 재개 중 · retryAt 전이면 아무것도 안 한다 · TryBake 가 안 됨 (형제가 쥠) 도 · TryBake 오류는 로그 한 줄 (retryAt 없음) ·
        잠금 뒤 state 를 다시 읽는다 (그 사이 committed 면 놓는다)
      onStale 은 곧바로 돌아온다 (`TestOnStale_ReturnsAtOnce`) — 배경 일이 채널에서 막힌 채 onStale 과 BeforeAdvert 가 돌아온다 · 막힌 동안의
        다음 onStale 은 아무것도 안 연다 (resuming) · 풀면 끝나고 Baker.Wait 가 돌아온다 · Wait 뒤의 onStale 은 아무것도 안 연다 (closed)
      광고 주기의 정리 (`TestOnStale_CleansAPendingWhoseOwnerDied`) — 형제 Dir 이 bake.lock 을 쥔 채 pending 이면 아무것도 안 한다 · 그 Bake 를
        Release 하면 (주인이 죽음 · 5절 규칙 9) committed · abandoned pending · 대기 자리를 그 scratch 의 trash 로 · 기록된 자리가 없어도
        committed · 못 옮겨도 committed 와 로그 · committed 를 못 쓰면 retryAt 과 잠금 Release · building (주인이 죽음) 도 같은 표로 치운다
        (abandoned building · 초안이 없으면 builds 는 비운다 · FD 규칙 12.2 · 13절)
      재개의 metadata — resumed true · node 는 초안의 Node · previous_ir 은 초안 · run 은 주인 Run
      재개는 DropBake · HoldBake 를 안 부른다 — guard 의 표지가 그대로라 합치기 전에 매칭된 늦은 임대가 lower_changed 로 거절된다
      merge 단계의 Apply 가 멈춘 뒤의 onStale 은 곧바로 재개한다 (retryAt 없음)
      재개의 대기 줄은 노드 로그 (마감 줄 없음) · 데몬이 멈추면 두 잠금을 놓고 merging 그대로
      재개를 여는 셋 — 기동의 merging 줄 (재개가 배경에서 · Baker.Wait) · build claim 이 낡은 merging 을 만나면 잡은 잠금을 넘겨 재개를 열고
        build 는 둘째 문장의 bake_in_progress (되물음 4 답 A) · 광고 주기의 onStale

### Step 14 — 기동 (`cmd/enode/main.go` · FD 흐름 5절 · FD 규칙 12.1)

- [x] 부르는 줄 넷 — `os.Args[1] == "merge-helper"` 면 `enode.RunMergeHelper(os.Stdin, os.Stdout, os.Stderr)` (trash-helper 의 `:53` 옆) ·
      `baker := enode.StartBaker(ctx, guard, scratchDir, ident, client.Instance, log)` (`:249` 뒤) · `Worker{..., Bake: baker}` · `wg.Wait()` 뒤
      `baker.Wait()` (nil 이면 할 일이 없다) — 넷 그대로
- [x] `main_test.go` — `TestMergeHelperEntrySkipsTheConfigSearch` (빈 stdin 으로 exit 1 · 설정을 찾지 않는다 · 오류 한 줄이나 linux 밖의 문장) ·
      오늘의 기동 시험이 그대로 초록 — 초록 (stdin 은 시험이 /dev/null 로 바꿔 끼운다)
- [x] `cmd/enode` 커버리지가 80% 이상 (80.5% 기준 · 194/241 이라 여유가 1.2 문장이다). `StartBaker` 와 `baker.Wait()` 는 기동 시험이 닿는
      한 문장씩으로 둔다 — 갈래는 `internal/enode` 안에 둔다 — 패키지 혼자 80.8% · CI 명령의 값은 Step 20

### Step 15 — Mediator 판정 시험 (`internal/api` · 시험 DB · FD 흐름 8.1 의 Mediator 줄 둘 · 되물음 2 답 B · 물음 1 답 B · 결정 54 · 4절 24 · 25번)

- [x] `internal/api/bake_test.go` — `TestBake_NoConditionsAndASyncFailureSucceeds` (판정 조건 없는 굽기 계약 · build DONE exit 1 과 build 칸 ·
      merge DONE merged 없음 -> Run SUCCEEDED · Record 의 build 칸) · `TestBake_IRMismatchFailsByTheManifestCondition` (예시 계약 · build DONE reason
      ir_mismatch · head_tags · merge DONE merged 없음 -> Run FAILED · Record 에 reason 과 head_tags · 어휘 밖 경고가 없다) ·
      `TestBake_MergeUploadTimeoutFailsTheRun` (build DONE 뒤 merge FAILED · reason upload_timeout · merge 칸 -> Run FAILED · Record 에 merge 칸과
      reason 이 남는다 · 어휘 밖 경고가 없다 · 4절 15번 · CG 물음 1 답 A) — 셋 초록 (시험 DB).  「어휘 밖 경고가 없다」 는 봉인된 결과의
      `OutOfVocabulary()` 가 비었는지로 본다 (시험 서버의 로그는 버려진다)
- [x] `api_test.go` 의 `TestSubmitRejectsBadContract` 에 굽기 줄 셋이 400 — 하위 시험 이름은 `bake without merge` · `bake build name` ·
      `bake build name twice` (go test 의 이름으로는 `TestSubmitRejectsBadContract/bake_without_merge` 등 — slice-5.sh 가 이 이름으로 본다)
      — 초록 · 세 줄은 400 의 까닭 글자까지 본다 (다른 칸 때문의 400 을 막는다)
- [x] 라우트 19 그대로 — `mux.HandleFunc(` 19

### Step 16 — 보안 — 격리 노드의 Prepare 와 시험 (3.1 · 4절 33번 · CG 물음 2 · 3 답)

- [x] `workspace.go` — `Prepare` 가 `WorkspaceWrites(w.Runtime)` 를 본다 (4절 33번). isolated 면 저장소 확인만 · `PrepClean` · 로그 `how` 가
      `fresh upper` · spec.Repo 가 비었으면 `PrepClean`. native (Runtime nil 포함) 는 오늘과 같다. 오늘의 `workspace_test.go` 넷 (Runtime nil ·
      진짜 git) 이 그대로 초록 — 했다 · native 의 로그 `how` 는 `reset and clean` · 넷 초록
- [x] `workspace_linux_test.go` — PATH 표지 방식 (가짜 `git` 스크립트는 argv 를 표지 파일에 한 줄씩 적고 `config --get remote.origin.url` 에는
      시험의 url · `rev-parse --abbrev-ref HEAD` 에는 `main` 을 답한다 · 가짜 `repo` 스크립트는 argv 를 적기만 한다 · Worker 는 Step 9 틀의 가짜
      isolated 런타임). `TestPrepare_AnIsolatedNodeRunsNoHostGitWrite` (git 모양 `.git` · repo 모양 `.repo/manifests` 둘 — 표지에는 config --get 과
      repo 모양의 rev-parse --abbrev-ref HEAD 만 · reset · clean 없음 · repo 는 한 번도 안 불린다 · PrepClean · spec.Repo 가 비었으면 git 도 안
      불리고 PrepClean) · `TestPrepare_AnIsolatedNodeDoesNotRunTheLowersFsmonitor` (진짜 git · git 환경은 5절대로 · lower 의 `.git/config` (git
      모양) 와 `.repo/manifests/.git/config` (repo 모양) 에 core.fsmonitor 로 표지 파일을 쓰는 스크립트 · Prepare 뒤 표지가 없다 · 대조군 — 같은
      트리에서 시험이 `git status` 를 돌리면 표지가 생긴다 · fixture 가 살아 있다는 확인) · `TestPrepare_ANativeNodeStillResetsAndCleans` (Runtime
      nil 과 `NativeRuntime{}` — git 모양은 reset --hard 다음 clean -df · repo 모양은 repo forall -c "git reset --hard" 다음 "git clean -df" · 표지의
      차례로 본다) · `TestPrepare_AnIsolatedNodeStillRefusesAnotherRepo` (가짜 git 이 다른 url 을 답한다 · `workspace repository mismatch` 문장 그대로
      · reset · clean 없음). t.Setenv 를 쓰므로 t.Parallel 을 안 쓴다 · t.Skip 을 안 부른다 (5절 규칙 10) — 넷 초록 (하위 아홉) · 대조군이
      표지를 남겨 fixture 가 살아 있다 (진짜 git 2.39.2 · DetectRepo 의 config --get 과 rev-parse 는 fsmonitor 를 안 부른다)
- [x] `bake_test.go` (모든 플랫폼) — `TestBakeSources_StartNoHostProcess` (굽기 흐름 파일 여섯을 `go/parser` 로 읽고 `go/ast` 로 훑는다 —
      import 에 `os/exec` · `syscall` 이 없고, `exec.` · `os.StartProcess` · `syscall.` · `NativeRuntime` · `DetectRepo` · `detectRepoManifest` ·
      `gitIn` · `child` 를 쓰는 자리가 없다 · 3.1) · `mergehelper_linux.go` 의 argv 는 unshare 로 시작한다 (기본값은 Step 10 의 TestMergeHelperArgv)
      — 초록 · 금지 이름에 `gitOut` · `gitOutName` 둘을 더했다 · 되돌림 모양의 글에서 아홉 자리를 찾는 대조군을 함께 둔다.  argv 는
      linux 파일의 `TestMergeHelperArgv` 가 본다
- [x] `bake_linux_test.go` — `TestBakeFlows_RunNoHostProgram` (PATH 를 임시 폴더로 바꾸고 bash · sh · git · repo 자리에 표지 파일을 쓰는
      스크립트를 둔다 · build (git 모양 · repo 모양) · merge · 재개를 가짜 런타임과 namespace 없는 helper 로 돌린 뒤 표지 파일이 없다 · t.Setenv 라
      t.Parallel 을 안 쓴다 · 3.1 「행동 시험을 더하는 까닭」) — 초록 (build 둘 · merge 둘 · 재개 하나 뒤 표지 없음)
- [x] `bake_build_linux_test.go` — `TestBakeBuild_CommandsGoThroughTheSession` (받은 argv 의 차례와 모양 · Dir 이 세션의 워크스페이스 자리 ·
      repo 모양의 pinned 포함) · `TestBakeBuild_UnnamedHostVariablesDoNotReachTheCommands` (t.Setenv 로 둔 호스트 변수 가운데 계약이 env 로 부르지 않은 것은 ProcessSpec.Env 에 없다 ·
      계약이 부른 이름은 있다 — 노드 비밀의 이름도 부르면 닿는 것은 잔여다 · 8절 · CG 물음 2 답) ·
      `TestIRProbe_TheIRGoesOnlyThroughTheEnvironment` · `TestBakeBuild_TheCommandsOutIsNotUploaded` (실패 갈래 — 명령이 세션의 `$OUT` 에 쓴
      manifest 가 produced 에 없다 · FD 흐름 8.1 의 지우는 시험을 올리지 않는 것으로 확인. 성공 갈래 — 명령이 가짜 manifest 와 다른 파일을
      써도 produced 는 manifest 하나이고 내용은 노드의 build 칸이다 · 4절 5번) — 넷 초록 (가짜 세션이 호스트 쪽 $OUT 에 쓰는 `doOut` 을
      틀에 더했다)
- [x] `bakerule_linux_test.go` — `TestProbeScript_OnAHostShell` (호스트 `sh -c probeScript` 를 임시 git 저장소 다섯 모양에 · git 환경은 5절대로)
      — 초록 (다섯 모양에 부모 저장소를 안 읽는 줄 하나를 더했다 · annotated 와 가벼운 태그가 한 커밋)

### Step 17 — 측정 (3.2)

- [x] `lowerguard_linux_test.go` 의 `BenchmarkBeforeAdvert` 를 committed 와 pending 으로 — pending 줄의 OnStale 은 부른 수만 세는 가짜 f 다 (진짜
      onStale 을 두면 반복마다 배경 일이 떠 측정을 흐린다). 3절 측정 1 (26.7 ~ 29.8 µs) 과 댄다 — committed 23.2 ~ 24.1 µs · pending 34.3 ~
      34.8 µs (공유를 안 쥐고 OnStale 을 부른다 · 부른 수가 광고 수와 같은지 벤치마크가 본다)
- [x] `bake_resume_linux_test.go` 의 `BenchmarkOnStaleWhileTheOwnerLives` (형제 Dir 이 bake.lock 을 쥔 채 배경 일 한 번 — TryBake 헛시도 — 을
      끝까지 · 측정 2 의 7.7 ~ 8.4 µs 근처) · `BenchmarkStaleCleanup` (옮기기 + committed · 1.5 ms 근처). `bake_merge_linux_test.go` 의
      `BenchmarkMergeStepFixedCost` (빈 upper · namespace 없는 helper · 배타가 비어 있음 — 그물 · helper 두 번 · state 쓰기 둘 · metadata · 옮기기)
      — 13.0 ~ 13.4 µs · 4.3 ~ 5.3 ms · 17.7 ~ 19.2 ms (-benchtime 2s · 세 번 · 이 기계)
- [x] 숫자는 Step 22 에서 code-summary 에. 하루치 Preflight · Apply 는 merge-rules 의 값 (3.2 표) 을 인용하고 다시 돌리지 않는다 — 옮겨 적는다
- [x] 한 자리 이상 크면 원인을 적는다 · 값을 맞추려고 규칙을 바꾸지 않는다 — 한 자리를 넘은 값은 없다.  정리 한 번이 추정의 세 배쯤이다
      (자기 scratch 의 pending/<키>/ 훑기 · trash 의 MkdirAll · rename · fsync 두 번이 든다) · 배경 일이 TryBake 의 1.6 배쯤이다 (고루틴 하나와
      wg 기다림)

### Step 18 — integration 시험 (`integration` 태그 · CI 밖 · FD 흐름 8.2)

- [x] `internal/enode/bake_integration_test.go` (`//go:build integration && linux`) — 진짜 unshare 로 여는 merge-helper 의 preflight · apply
      (subordinate uid 소유 항목 · 권한 000 디렉터리) · 끊고 잇기 (helper 를 apply 도중 SIGKILL -> merging 이 남는다 -> `resume` 이 끝낸다 ·
      metadata resumed true · committed) — `TestMergeHelperIntegration` · `TestResumeAfterAKilledHelperIntegration` (helper 를 여는 명령을
      감싼 sh 가 upper 의 절반이 옮겨진 순간에 unshare 를 SIGKILL 한다 · 항목 100,000)
- [x] `internal/merge/merge_integration_test.go` — `TestBindAliasPreflightIntegration` (user · mount namespace 안에서 lower 를 bind 별칭으로 걸고 그
      경로로 Preflight -> Check mount) — 더했다 (`--map-auto` 없이 돈다)
- [x] `internal/enode/runc_overlay_integration_test.go` — `TestRuncOverlayCloseKeepsTheUpperIntegration` (진짜 세션이 단계 사용자로 쓴 파일이
      `Close(Keep{Upper})` 로 옮겨지고 호스트에서 노드 uid 소유 · 나머지는 trash). 그 파일의 `TestMain` 이 `merge-helper` 입구도 안다 — 더했다
- [x] `internal/enode/runc_overlay_integration_test.go` — `TestRuncOverlayFinalizeDiffInTheSessionIntegration` (4절 34번 · 진짜 runc 세션의 edit 단계가
      git 워크스페이스의 추적 파일을 고치고 `.git/config` 에 core.fsmonitor 로 호스트 절대 경로 (시험의 임시 폴더 · 컨테이너에는 없다) 에 표지를 쓰는
      스크립트를 적는다 · Finalize 뒤 workspace.diff 에 그 고친 줄이 있고 호스트 표지가 없다 · rootfs 에 git 이 없는 갈래는 Step 7 의 단위
      시험이 본다). 이 기계는 `--map-auto` 가 막혀 못 돈다 — SunnyVM 에서 돈다 — 더했다 · rootfs 는 시험이 받는 것 (ENODE_RUNC_ROOTFS) 이라
      git 이 든 것이어야 한다
- [x] `go vet -tags integration ./internal/enode/ ./internal/merge/ ./internal/lower/` 로 컴파일을 본다 — 초록
- [x] 이 기계에서 돌린다 — bind 별칭 줄은 돈다 · runc 와 subordinate uid 장면은 `--map-auto` 가 막혀 못 돈다 (그렇다고 적는다). SunnyVM 이 켜져
      있으면 에이전트가 거기서도 돌리고 결과를 적는다 — 버려도 되는 `~/bake-it-*` 에서만 · `/srv/yocto` 와 떠 있는 두 노드의 자리는 읽지도 않는다 ·
      둔 파일은 지운다. 꺼져 있으면 보류로 적는다 (병합 조건은 사람 조각이다) — 이 기계: bind 별칭 초록 · merge-helper 둘은 `newuidmap: write to
      uid_map failed` 로 빨강 (막힘) · runc 둘은 rootfs 가 없어 건너뜀.  SunnyVM (2026-09-30 · `~/bake-it-<시각>`): 다섯 모두 초록 — bind 별칭
      (마운트 196 대 8031) · merge-helper (subordinate uid 파일 · 000 디렉터리 · 종류 바뀜 하나 trash) · 끊고 잇기 (100,000 가운데 46,405 가 남았을
      때 죽임 · 재개가 46,409 연산을 1.159 초에) · keep (노드 uid 소유) · 세션 안 diff (127 바이트 · 호스트 표지 없음).  rootfs 는 그 자리 안에 호스트의
      dash · bash · git · mktemp 등과 라이브러리만 복사해 지었다 (9.5 MB).  끝나고 자리를 지웠다 (`~/bake-it-*` 0)

### Step 19 — 조각 스크립트 (FD 흐름 8.3 · 물음 4 답 · 결정 20 · 21 · 36 · 38 · 4절 26번)

- [x] `bake-common.sh` — 대상 출력 (WS 의 경로 · 장치 maj:min · inode) · 허용 표지 (`<WS>/.enode-disposable` 이 보통 파일 · 없으면 운영 lower 로
      보고 멈춘다) · 같은 lower 확인 (`GET /v1/nodes` 에서 machine 이 이 기계이거나 없는 노드마다 ws 를 stat · 같은 장치 · inode 인 노드가 모두
      `workspace.writes=isolated` · 키가 없으면 「옛 판이 있다」· in-place 면 「native 노드가 있다」· stat 을 못 하면 멈춘다) · 대상 노드의 판은
      사람이 확인한다는 출력 · 못 잡는 셋의 출력 (다른 Mediator · 꺼진 노드 · 확인 뒤 뜬 노드) · 표지가 복사됐을 수 있다는 경고 · 결과에
      lower_changed 가 보이면 「다시 내면 된다」 · 목록 비교 (user namespace 의 ro overlay 로 본 merged view 와 합친 lower · `.enode-metadata.json`
      은 뺀다) — 했다.  공용 도우미 (계약 짓기 · 제출 · Run 과 lower 의 phase 기다리기 · Record 의 단계 칸 · 공유 잠금을 잠깐 쥐어
      merge 를 세우는 `hold_lower` · 목록) 를 함께 둔다.  계약 두 모양 (굽기 · 형제의 긴 명령) 은 `runctl lint` 초록
- [x] `slice-5.sh` — 조각 5 의 시험 이름 (7.1 절의 정확한 하위 시험 이름) 을 `go test -json -run` 으로 돌린다. 시험 DB 를 요구한다 —
      `ENODE_TEST_DATABASE_URL` 이 없으면 곧바로 `slice 5: red` (DB 가 없으면 internal/api 의 시험이 t.Skip 하고 go test 는 exit 0 이다 ·
      `api_test.go:31` ~ `:34`). 목록의 시험 (하위 시험 포함) 마다 JSON 의 `pass` 줄이 있어야 한다 — 스킵이거나 없거나 (`-run` 이 아무것도 못
      맞춰도 exit 0 이다) 실패면 red 와 그 이름. 모두 pass 일 때만 `slice 5: green` — 했다 (목록에 Mediator 판정 시험 셋과
      `TestValidate_MergeStep/budget` · `TestContractStep_CarriesTheBakeFields` 를 더했다 · 결과는 Step 20)
- [x] `slice-6.sh` — 장면 2 의 1 ~ 6 · 빌드 하나를 일부러 실패시킨 굽기 · sync 가 IR 에 닿지 않는 굽기 · 처음 확인 셋 (세션 안 git 의 소유 확인 ·
      rootfs 의 git 과 bash). 계약은 `runctl example bake` 처럼 build · merge 에 produced 조건을 건다. 사람이 보는 조각이라 판정하지 않고 멈춰 무엇을
      볼지 출력한다. 출력에 싣는 것 — `.enode-metadata.json` 의 칸 목록과 `bake.run` · `bake.node` (US-6 · 7.1 조각 6) · merge 단계 로그의 쥔
      쪽 줄과 남은 시간 (완료 조건 5) · 장면 2 의 4 의 네 시각 — 형제 Run 이 끝난 때 (Record) · 형제 노드 로그의 `released the lower lock` ·
      merge 단계 로그의 `took the lower lock after` · committed (state.json 의 since 와 metadata 의 merged_at) — 를 기대 창과 함께 (released 는
      형제 Run 끝에서 두 광고 주기 안 · took 는 released 뒤 1초 안 · committed 는 took 뒤 합치기 시간). 장면 2 의 4 의 「몇 초」 는 took 부터
      committed 까지로 따로 한 줄 보인다 — 앞의 60 ~ 120 초는 lower-state 답 1 의 설계값이다 (CG 물음 4 답 · 7.1). 형제 노드 로그의 자리는
      입력 `SIBLING_LOG` (비우면 그 시각을 사람이 적는다) — 했다.  took 의 시각은 merge 단계의 시작 (Record) 에 `took the lower lock
      after` 의 길이를 더해 셈한다 (그 줄에 시각이 없다).  IR 이 어긋나는 굽기는 계약의 ir 을 없는 태그로 두고 sync 는 IR 로 간다
- [x] `slice-7.sh` — 같은 lower 의 다른 노드를 멈춘 채 merge 단계를 여러 자리에서 SIGKILL -> 다른 노드를 시작 -> committed · resumed true 와 원래
      Run · 목록이 한 번에 끝낸 것과 같다 · 둘째 경우 (형제를 띄워 둔 채 SIGKILL -> 한 광고 주기 안에 잇는다) — 했다.  끊길 틈이 있게 구성 하나가
      파일 MANY 개를 쓰고 (먼저 한 번 구워 둔다) merging 을 본 뒤 KILL_AFTER 초마다 죽인다
- [x] `slice-8.sh` — 굽기 중 두 번째 굽기 (bake_in_progress) · pending 에서 SIGKILL -> 떠 있는 형제가 두 광고 주기 안에 정리하고 drain 이 풀린다 ·
      merge.wait 1m + 형제에 5분 Run (merge_wait_timeout · Record 에서 build DONE 과 merge 의 reason 이 따로) · bind 별칭 두 노드가 같은 상태 자리 ·
      env check 의 not ready 사유 셋 · 늦게 매칭된 형제 Run 의 lower_changed — 했다.  두 번째 굽기는 형제 노드 (SIBLING_WS) 에 낸다 —
      노드마다 임대가 하나라 같은 노드에 내면 줄을 선다.  env check 셋과 늦은 매칭은 사람이 하는 일과 볼 것을 출력한다
- [x] 다섯 파일 — `bash -n` · (있으면) `shellcheck` · 사내 이름 0 · 출력 영어 · 주석 한국어 · 계약 명령의 예시는 자리표시 (`<your sync command>`)
      — `bash -n` 다섯 초록 · shellcheck 는 이 기계에도 SunnyVM 에도 없다 · 사내 이름 0 · 출력 줄의 한국어와 장식 문자 0 (출력의 — 와 · 도
      ; 와 , 로) · 계약의 sync 는 git 명령이라 자리표시가 없고 BUILD_A · BUILD_B 의 예시가 `<your build command for config-a>`.  생성한 sync 를
      로컬 저장소에 돌려 HEAD 가 태그와 같은 것을 봤다

### Step 20 — 코드 검사 (유닛 정의 0절 · CI 와 같은 명령)

- [x] `gofmt -l .` · `go vet ./...` · `go vet -tags integration ./internal/enode/ ./internal/merge/ ./internal/lower/` · `go build ./...` — 빈 출력 · exit 0
      — 빈 출력 · 셋 모두 exit 0
- [x] CI 의 시험 · 판정 명령을 그대로 돈다 (시험 DB · 가짜 claude 스텁) — 시험 단계 `go test ./... -count=1` (`ci.yml:216`) · 커버리지 단계의
      `go test ./... -count=1 -coverpkg=./... -coverprofile=/tmp/cover.out -json > /tmp/test.json` (`:267`) 과 그 awk (`:269` ~ `:300` · 패키지마다
      80%) · 스킵 감시의 awk (`:343` ~ `:441` · `/tmp/test.json`). 실패 0 · 스킵 0 · 미달 0. `internal/enode` · `cmd/enode` ·
      `internal/merge` · `internal/contract` · `internal/store` · `internal/api` 는 Step 1 과 댄다. 새 파일의 줄마다 시험이 닿는지 `go tool cover -func`
      로 본다 — 굽기 흐름 파일은 85% 이상을 겨눈다 (`internal/enode` 의 여유가 얇다 · 3절 측정 5). 모자라면 호출 자리를 함수 값으로 떼어 실패를
      끼운다 (merge-rules · lower-state 와 같은 방법)
      — 시험 단계 exit 0 (77초) · 커버리지 단계 통과 2,562 (Step 1 2,271) · 실패 0 · 스킵 0 · 미달 0 (스물세 패키지) · 전체 87.0 -> 87.6% ·
      enode 84.4 -> 86.7 · cmd/enode 80.5 -> 80.8 · merge 89.1 -> 88.9 (statx 실패 두 갈래는 못 닿는다) · contract 92.5 · store 82.9 ·
      api 82.5 (셋 그대로) · lower 93.2 -> 93.3 · 스킵 감시 패키지 24 · 허용목록 밖 0.  굽기 흐름 파일 — bakefile.go 75.7 -> 86.2 (실패를
      끼우는 시험 넷과 openPlain 의 함수 값 `afterLstat` 하나) · bake_resume.go 84.6 -> 93.5 (그물 시험 · 못 읽는 state) · 나머지 91.0 ~ 98.3
- [x] `go test -race -count=1 -timeout 30m ./internal/enode/ ./internal/lower/ ./internal/merge/` — 경합 0 · 몸통의 동시성 시험은
      `go test -race -count=10 -run '^TestHeldBake_OneBodyUnderRace$' ./internal/enode/`
      — exit 0 (135초) · 통과 977 · 경합 0 · 몸통 시험 `-count=10` exit 0
- [x] 흔들림 (5절 규칙 8) — 새 시험 `-count=20 -timeout 30m` (패키지 `./internal/enode/ ./internal/merge/ ./internal/lower/ ./internal/contract/
      ./cmd/enode/` · `-run` 은 Step 1 이 떠 둔 목록과 지금 `go test -list .` 의 차이로 짓는다 — 이름을 손으로 고르지 않는다) · CPU 부하 아래 한 번
      (`taskset -c 0,1 go test -count=3 -timeout 30m -run <새 시험> ./internal/enode/` 를 돌리는 동안 옆에서 `taskset -c 0,1 go test -count=3
      ./internal/lower/ ./internal/scratch/ ./internal/merge/`) · `internal/enode` 전체 `-count=5 -timeout 30m`. 새 빨강은 고친다 · 기준선의
      흔들림은 Step 1 의 목록과 댄다
      — Step 1 의 목록과 달라진 이름 94 가운데 시험 91 (벤치마크 셋은 뺐다) `-count=20` exit 0 · 통과 1,820 · 실패 0 · 부하 아래 `-count=3` 통과 264 (enode 의
      88 x 3) · 실패 0 · 옆의 다섯 번 실패 0 · `internal/enode` 전체 `-count=5` exit 0 (226초) · 윗 시험 통과 2,730 · 실패 0.  새 빨강 0 ·
      기준선의 흔들림 (Step 1 의 `TestRuncOverlayOpenIncludesHelperStderr`) 도 이번에는 0
- [x] 크로스 빌드 셋 (windows/amd64 · linux/arm GOARM=7 · darwin/arm64) · `GOOS=windows go test -c -o /dev/null ./internal/enode/` 의 오류가
      Step 1 의 목록 (`overlay_test.go:41` 의 셋) 밖에 새 것이 없다 (크로스 빌드는 시험 파일을 컴파일하지 않는다) · `enodectl.exe` 심볼 상한
      (crypto/tls 10 · net/http 50) · U+2605 0 · `go run ./scripts/glyphscan.go` (시험이 아닌 Go 파일의 문자열만 본다 · `scripts/glyphscan.go:50`
      ~ `:64` — 시험 메시지와 스크립트 출력은 눈으로 본다) · 린트 목록을 Step 1 의 목록과 diff 로 대 새 경고가 없다 (수만 보면 하나 고치고
      하나 들인 것을 못 본다 · 린트 단계는 CI 를 막지 않는다 — `ci.yml:116` ~ `:117` · 새 코드가 버린 오류는 `_ =` 로 드러낸다)
      — 셋 exit 0 · windows 시험 컴파일 오류는 `overlay_test.go:41` 의 셋 그대로 · crypto/tls 1 · net/http 6 · U+2605 0 · glyphscan 168 파일
      0 · 린트 38 (Step 1 과 같은 목록 — 새로 난 셋 · 버린 `os.RemoveAll` 둘과 드모르간 하나 · 을 고쳤다).  눈으로 본 것 — 새 시험의 문자열에
      한국어와 장식 문자 없음 (runc 시험의 한국어 경로 픽스처를 ASCII 밖의 라틴 글자 경로로 바꿨다) · 새 코드의 주석 구분선 (괘선) 을 뺐다
- [x] **조각 0** (기동이 안 깨졌다) — build · vet · test · 라우트 19 — 초록 · `mux.HandleFunc(` 19 (`internal/api/api.go` 안 고침)
- [x] **조각 5** (굽기 계약) — Step 4 · 12 · 15 의 시험과 contract 의 시험이 초록 · `slice-5.sh` 가 `slice 5: green`
      — `slice 5: green` (목록 스물하나 모두 pass · 스킵 0 · 시험 DB)
- [x] `git status` 로 2절 밖의 파일이 없는지 · 시험이 바꾼 `cmd/enodectl/probe.lock` 은 되돌린다
      — 2절 밖은 시험 파일 `internal/enode/finalize_test.go` (Step 11 의 sealErr 두 줄) 와 Step 22 가 고친 회차 문서 `component-methods.md` ·
      계획 단계가 고친 여섯 (계획 · 상태 · 감사 · FD 두 장 · unit-of-work.md).  probe.lock 은 되돌렸다

### Step 21 — 사람 조각 6 · 7 · 8 (준비만 · 병합 게이트)

- [x] 스크립트 셋의 돌리는 법 (입력 · 버려도 되는 lower 를 만드는 법 · 허용 표지 · 형제의 bind 별칭) 을 code-summary 에 적는다. 에이전트는 돌리지
      않는다 — 사람이 보는 조각이다 (US-14 · 코드 시험이 초록이라는 것으로 대신하지 않는다)
      — code-summary 8절 (준비 · 명령 · 무엇을 보나).  적다가 형제 모양의 구멍을 찾아 고쳤다 — bind 별칭의 형제는 굽는 노드의 합치기를
      잇지 못한다 (FD 규칙 10절 끝) 라 조각 7 은 symlink 형제 (같은 마운트) 로 · 조각 8 의 4 는 bind 별칭으로 돈다.  `check_target` 의 같은
      경로 확인을 글자 비교로 · `slice-7.sh` 는 다른 마운트면 멈춤 · `slice-8.sh` 의 4 는 형제 모양을 출력 (code-summary 5절)
- [x] SunnyVM 이 꺼져 있으면 보류로 적는다 — 보류는 통과가 아니고 병합 지점도 아니다 (scene-gates.md 4절)
      — 닿는다 (2026-09-30 · Step 1 · 18) — 보류가 아니다.  조각 6 · 7 · 8 은 아직 안 돌았으므로 병합 지점도 아니다
- [x] 받는 일 44 의 셋째 (옛 판 노드가 있는 lower) 는 잔여가 아니라 이 사람 조각에서 처음 본다 (FD 흐름 12절 · business-logic-model.md:605 ~
      :611) — 돌리는 법에 그 확인을 적는다 (bake-common.sh 가 「옛 판이 있다」 로 멈추는 것 · 그물이 옛 판의 마운트를 찾는 것)
      — code-summary 8.4 (멈추는 두 줄의 글자 · 확인 뒤 옛 판에 긴 명령을 낸 채 굽기 · 그물 줄의 글자)
- [x] 사용자가 돈 결과는 PR 전에 audit 과 code-summary 에 적는다 — 사용자 지시로 집행 에이전트 (코드를 쓰지 않은 에이전트) 가 SunnyVM 에서
      돌리고 사용자가 판정했다. 첫 실행 (7116e8d) 은 조각 7 에서 흠을 찾아 멈췄고 고친 뒤 (6d9e6c7) 모두 돌았다 — code-summary 8.5 · 8.6.
      사용자 판정 초록 2026-09-30T12:50:12Z

### Step 22 — 요약 · 회차 문서 · 상태 · 감사 · 커밋

- [x] `construction/bake/code/code-summary.md` — 파일 · 규칙의 자리 (FD 규칙 절 -> 함수) · 4절의 결정 · 계획과 다른 자리 · 코드 검사 숫자 ·
      3절의 측정과 시험 결과 · integration 결과 · 행렬 밖 파일의 diff (finalize.go 의 settleIn 한 칸은 계획이 더한 것으로 따로) · 조각 6 · 7 · 8 의
      돌리는 법 · 넘기는 것 (checkpoint · FD 흐름 12절) · 정본 되돌림 (FD 흐름 13.1 · 진행자가 올린다). 옮길 때 FD 13.1 의 글자를 고친다 —
      run-contract 의 미정 항목 주소 `:1068` 은 `:1069` 다 · 「bash -c 가 그 래퍼다」 는 정본과 반대다 (run-contract 의 yocto 절 첫 항목
      `:1122` ~ `:1127` · `:1146` — 답은 노드가 가진 래퍼 스크립트이고 계약에 셸을 여는 것이 아니다) — 「굽기는 계약의 문자열을 bash -c 로 연다 ·
      run-contract 의 그 항목과 다르다」 로 적는다. 두 줄을 더한다 — `mediator-api.md:472` (「exit_code 는 명령 단계만」 — build 결과도
      exit_code 를 싣는다 · 판정 재료가 아니다) · `execution-environment.md:610` ~ `:612` (「process 가 직접 `$OUT` 에 쓴 파일은 자동 수확」 —
      build 단계는 세션의 `$OUT` 을 올리지 않는다 · 4절 5번). 4절 15 · 33번은 정본을 따르게 한 것이라 되돌림이 없다. 잔여 · 후속 과제 절에 —
      계약 env 이름으로 노드 비밀 (ENODE_TOKEN 등) 이 명령에 실리는 길 (CG 물음 2 답 · 8절). 정본에 보탤 것 (되돌림이 아니다) — 「격리 노드에서
      호스트 쪽 helper 는 워크스페이스가 적은 설정을 실행하는 명령을 돌리지 않는다」 (4절 34번 · FD 흐름 13.1 · 진행자가 정한다)
      — 썼다 (열한 절 · 13.1 의 글자 둘을 고쳐 옮기고 두 줄을 더했다 · 잔여 · 보탬 · 조각 6 · 7 · 8 의 돌리는 법은 8절)
- [x] 코드의 겉면이 회차 문서와 달라진 자리를 고친다 — `component-methods.md` 4.2 (`runBuildStep` · `runMergeStep` 의 인자 · 4절 2번) · 4.3
      (`OnStale` — guard 가 f 를 곧바로 부르고 f 는 막지 않는다 · 4절 18번 · `Dir`) · 4.4 (`RunMergeHelper`)
      — 고쳤다 (4.2 두 겉면과 `Worker.Bake` · `StartBaker` · 4.3 부르는 자리와 nil 수신자 · 4.4 주고받는 한 줄과 여는 명령)
- [x] 표기 검사 · 사용자가 싫어한 말투 · 사내 이름 · 새 축약어 (저장소에 GLOSSARY.md 가 없다 — 새 글자를 들이지 않는다)
      — emphasis-check.py exit 0 (문서 셋 · 스크립트 · 고친 Go 파일) · 말투 0 (claim.go 주석의 「갈라진다」 를 「나뉜다」 로) · 사내 이름 0 ·
      장식 문자 0 (새 코드의 괘선 구분선을 뺐다) · 새 축약어 0 (FD · CG 는 code-summary 머리에서 푼다)
- [x] 이 계획의 체크박스가 모두 [x] 이고 결과 한 줄이 있는지 다시 본다 (단계마다 그 자리에서 적었다 · 6절 머리) · `aidlc-state.md` 의 U7 절 ·
      `audit.md`
      — 88 가운데 [x] 86 · 남은 둘은 Step 21 의 「사용자가 돈 결과」 (조각 6 · 7 · 8 을 사용자가 돈 뒤) 와 이 단계의 커밋.  `aidlc-state.md` 의
      U7 절과 `audit.md` 는 진행자가 고친다 — 이 단계는 건드리지 않았다
- [x] 한 커밋 — 코드 · 시험 · 스크립트 · 이 계획 · code-summary · 고친 회차 문서 (이 계획 단계가 고친 FD 두 장과 unit-of-work.md 7절 포함 ·
      2절) · 상태 · 감사. **승인 뒤에 넣는다** (CONVENTIONS 3.3). PR 은
      조각 6 · 7 · 8 이 초록인 뒤이고 올리기 전에 묻는다 — 승인 2026-09-30 (「승인. 커밋해」) 뒤 진행자가 unit/bake 에 한 커밋으로 넣었다 ·
      push 와 PR 은 하지 않았다

---

## 7. 추적

### 7.1 조각 5 ~ 8 의 수용 기준 (`scene-gates.md` 2절 · `requirements.md` 6절의 바뀐 줄 · FD 흐름 8.3)

```text
   조각 5 (굽기 계약 · 기계)
     prepare 단계 뒤에 merge 가 없는 계약이 400      이미 초록 — TestValidate_BakeShape/build_alone ·
                                                   TestGrammar_WhatItForbidsIsActuallyRejected/a_bake_has_both_steps ·
                                                   제출 경로 — Step 15 (TestSubmitRejectsBadContract/bake_without_merge)
     구성 이름이 규칙을 어긴 계약이 400              이미 초록 — TestValidate_BuildStep/upper-case_name · /empty_name · /name_of_65 ·
                                                   TestGrammar_WhatItForbidsIsActuallyRejected/builds_names ·
                                                   Step 15 (TestSubmitRejectsBadContract/bake_build_name)
     이름이 겹친 계약이 400                         이미 초록 — TestValidate_BuildStep/name_twice ·
                                                   TestGrammar_WhatItForbidsIsActuallyRejected/builds_names_do_not_repeat ·
                                                   Step 15 (TestSubmitRejectsBadContract/bake_build_name_twice)
     merge 대기 상한을 바꾼 계약이 그 값으로 기다린다   이미 초록 — TestMergeWait_FillsDefault · Step 4 TestClaim_MergeWaitReachesTheNode ·
                                                   Step 12 TestMergeStep_WaitsForTheContractValue
     게이트                                         Step 20 — slice-5.sh 가 slice 5: green

   조각 6 (굽기가 끝까지 돈다 · 사람 · SunnyVM · 버려도 되는 lower)             기계의 앞선 시험 (Step)                    사람 (Step 19 slice-6.sh)
     장면 2 의 1 — 빈 lower 처음부터 굽기 A · 합치기는 rename 몇 번         BakeBuild 성공 (11) · MergeStep_Merges (12)          1
     장면 2 의 2 · 3 — 형제 긴 Run 중 굽기 B · build 는 곧바로 · merge 는      MergeStep 기다림 (12) · WaitLog (5)                  2 · 3 — 단계 로그의 쥔 쪽 줄과
       waiting · 형제는 draining                                                                                          남은 시간 (완료 조건 5) · GET /v1/nodes
     장면 2 의 4 — 형제 Run 이 끝나자 몇 초에 합친다.  「몇 초」 는 배타      MergeStepFixedCost (17) — 배타를 쥔 뒤의 고정 비용    4 — slice-6.sh 의 네 시각.  판정하는
       잠금을 잡은 때 (took the lower lock after) 부터 committed 까지다                                                     수는 took 부터 committed 까지 ·
       (CG 물음 4 답).  형제 Run 끝에서 took 까지의 60 ~ 120 초는 lower-state                                              앞의 60 ~ 120 초는 기대 창 안인지만
       답 1 (drain 응답 두 번) 의 설계값이다                                                                               본다
     장면 2 의 5 — 형제가 새 ir 과 repo.built.<이름> 을 광고                 metadataOf (5) · lower-state 의 광고 키                5 — GET /v1/capabilities
     장면 2 의 6 — 합친 lower 목록 == 합치기 전 merged view                  -                                                   6 — bake-common.sh 의 목록 비교
     .enode-metadata.json 의 칸이 다 있다 (팩 결정 3-16 — metadata 가 담는 것)   metadataOf (5) · MergeStep_Merges (12)                칸 목록 · bake.run · bake.node (US-6)
     빌드 하나 실패 -> 합치지 않고 last_attempt                              BakeBuild 실패 줄 (11) · Mediator 판정 (15)             build DONE · Run FAILED (조건)
     (바뀜) sync 가 IR 에 닿지 않음 -> manifest 없음 · 합치지 않음 ·          irVerdict (5) · BakeBuild IR 두 갈래 (11) ·            head · head_tags · Run FAILED
       HEAD 의 태그와 커밋 · Run 은 계약의 조건이 FAILED                       TestBake_IRMismatchFailsByTheManifestCondition (15)    (완료 조건 10 · US-19)
     처음 확인 — 세션 안 git 의 소유 · rootfs 의 git 과 bash                 ProbeScript_OnAHostShell (16)                         처음 확인 (FD 계획 2.6)

   조각 7 (끊겨도 된다 · 기계 + 사람)
     여러 지점 강제 종료 -> 같은 lower 의 다른 노드가 시작 때 재개 ·           이미 초록 — merge-rules TestApplyResume (41 자리) ·      slice-7.sh 첫 경우
       결과 목록이 한 번에 끝낸 것과 같다                                      Step 13 재개 시험 · Step 18 끊고 잇기 (integration)
     metadata 에 resumed 와 원래 Run                                        metadataOf (5) · 재개의 metadata (13)                  slice-7.sh
     가짜 트리 1 ~ 30번째 연산 뒤 강제 종료가 Go 시험                          이미 초록 — TestApplyResume
     (바뀜 · 요구 6절) 가짜 트리에 종류가 바뀐 항목 · 기본 go test              이미 초록 — TestApplyResume 의 mainFixture
                                                                          (TypeChanged 4 · internal/merge/tree_linux_test.go:197 ~ :199)
     (더함 · FD 흐름 8.3) 살아 있는 형제가 한 광고 주기 안에 잇는다            OnStale (8) · 재개 (13)                                slice-7.sh 둘째 경우

   조각 8 (배타와 대기 · 사람 · SunnyVM)
     굽기 중 두 번째 굽기가 곧바로 bake_in_progress (US-16)                  TestBakeBuild_AnotherBakeHoldsTheLower (11)             slice-8.sh
     pending 에서 죽으면 다른 노드가 정리 · drain 이 풀린다                    TestOnStale_CleansAPendingWhoseOwnerDied (13) ·         떠 있는 형제 · 두 광고 주기 안
                                                                          OnStale (8)                                          (되물음 1 답 A)
     짧은 대기 상한 · 형제 Run -> merge_wait_timeout · upper trash ·          TestMergeStep_WaitsForTheContractValue (12)             Record 에서 build DONE 과 merge 의
       committed · drain 풀림                                                                                               reason 이 따로 (완료 조건 9 · US-17)
     bind 별칭 두 노드가 같은 상태 자리                                      lower-state (키 · TestBindAliasCheckIntegration)         slice-8.sh
     다른 사용자의 노드 · 다른 filesystem 이나 마운트의 scratch -> not ready    lower-state 의 점검 셋                                 사유를 이름으로 (요구 6절)
     (더함 · 요구 6절) 형제가 임대를 받은 뒤 첫 단계 전에 lower 가 안 바뀐다     lower-state TestLowerGuard_LateLeases ·                 늦게 매칭된 Run 의 lower_changed
                                                                          재개가 DropBake 를 안 부른다 (13)
```

### 7.2 FD 규칙의 절 -> 단계

```text
   원칙 셋 (merging 에서 committed 는 합치기로만 · 주인 하나와 몸통 한 번 · 판정은 계약의 조건)    9 · 11 · 12 · 15
   1  단계 분기와 거절                  4 · 11            10 시작 전 확인과 어긋났을 때           3 · 10 · 12 · 13 · 18
   2  전이와 쓰는 때                    9 · 11 · 12 · 13  11 몸통 · HoldBake · DropBake            9 · 11 · 12
   3  build 의 차례                     5 · 6 · 7 · 11 · 13 · 16  12.1 기동의 자리                        9 · 14
   3.1 초안의 칸                        5 · 6 · 11        12.2 TryBake 가 되면                    6 · 9 · 13
   4  IR 대조                           5 · 11 · 15 · 16  12.3 재개의 차례                        13
   5  exited 와 예산                    11 · 12 · 15      12.4 끝났는지 먼저                      5 · 13
   6  합칠 upper 를 못 남기면            9 · 11 · 15       12.5 재개의 metadata                    5 · 13
   7  merge 의 차례                     12                13 살아 있는 동안 정리와 재개             8 · 9 · 13 · 17
   7.1 claim 확인                       5 · 12            14 last_attempt                        5 · 9 · 11 · 12
   8.1 ~ 8.3 시계 · 간격 · 줄           5 · 12            15 결과 칸과 $OUT                       2 · 4 · 5 · 11 · 12
   8.4 그물                             12                16.1 ~ 16.3 오류와 로그 문구            5 · 11 · 12 · 13
   8.5 끝나는 셋                        12                17 확장 준수                           9절
   8.6 어디에 쓰나                      12 · 13
   9  merge-helper 와 trash             6 · 10 · 12
```

### 7.3 FD 흐름의 절 -> 단계

```text
   1  한눈에 — 하루치 굽기             11 · 12 · 17 · 19          8.1 기계 시험                  2 ~ 16 (시험 이름은 각 단계)
   2  build 단계                       11                         8.2 integration                 18
   3  merge 단계                       12                         8.3 사람 조각 6 · 7 · 8         19 · 21
   4  몸통과 다른 길이 만나는 자리       9 · 12 (표의 여섯 줄)       9  순수 함수와 커버리지          5 · 20
   5  기동                             9 · 13 · 14                10 행렬 밖                      2절 · 2 · 3 · 4 · 8 · 11 · 7 (주석 · diff.go · runc 의 finalize) · 16 (workspace.go)
   6  살아 있는 동안 정리와 재개         8 · 13                     11 답 대 보기와 추적              7절
   7  실패 갈래 한 장                   11 · 12 · 13 (줄마다 시험)   12 넘기는 것                    8절 · 22
                                                                 13 정본과 회차 문서              7 (굽기 전부터 있던 구멍 · 보탬) · 16 (ADR-072 결정 3) · 22 (회차 문서 · code-summary 에 넘김)
                                                                 14 확장 준수                    9절
```

### 7.4 FD 엔티티의 절 -> 단계

```text
   머리 (파일 표 · 지키는 셋)     2절 · 4절 1 · 7번           6  결과 칸 · 원인 코드          2 · 4 · 5 · 11 · 12
   1  Step 의 칸 넷 · contractStep  4                         7  last_attempt                5 · 9
   2  Baker                        9 · 14                     8  merge-helper 의 입출력       10
   3  heldBake                     9 · 11 · 12                9  merge.Check 의 mount         3
   4  대기 자리와 초안               6 · 11 · 13                10 낡은 상태의 정리와 재개        8 · 13
   5  IR 대조                       5 · 11 · 16                11 대기 로그                    5 · 12
                                                             12 linux 와 그 밖               1절 크로스 빌드 · 20
```

### 7.5 스토리와 완료 조건 -> 단계

```text
   US-6   lower 를 마지막으로 바꾼 굽기 Run 과 노드                     5 (metadataOf) · 12 · 13 · 19
   US-12 · US-15 · 완료 조건 5   merge 가 누구를 언제까지 기다리나       5 (waitLog) · 12 · 19
   US-13 · 완료 조건 8   대상 lower 가 버려도 되는가 · 운영이면 멈춤       19
   US-14  사람 조각을 사람 눈으로                                      21
   US-16  bake_in_progress 는 다시 내면 되는 거절                       5 (문장 둘) · 11
   US-17 · 완료 조건 9   merge_wait_timeout 이면 build 는 성공          12 · 19
   US-18 · 완료 조건 6   재개로 합쳐졌나                               12 · 13 · 18 · 19
   US-19 · 완료 조건 10  IR 대조가 어긋나면 sync 가 어디에 닿았나        5 · 11 · 15 · 19
```

### 7.6 받는 일 쉰여섯 -> 단계

```text
    1 셸로 도는 sync · builds       11 · 16    20 노드가 받는 칸 넷              4            39 state.json 을 쓰는 때            9 · 11 · 12
    2 ENODE_IR                     11 · 16    21 Keep.Upper 에 대기 자리          7 · 11       40 HoldBake · DropBake 의 차례      9 · 11 · 12
    3 sync 뒤 대조                  5 · 11     22 대기 상한의 upper 는 Trash.Move   9 · 12       41 몸통 · building 은 build 끝에서   9 · 11
    4 .repo 없는 git                5 · 16     23 lower 쪽 항목도 Trash.Move        10           42 WriteMetadata 를 부르는 자리      12 · 13
    5 성공했을 때만 manifest         11         24 helper 안에서 Preflight 뒤 Apply  10 · 12 · 13  43 옛 판 노드와 스크립트 확인         19
    6 합쳤을 때만 merged             12         25 Discard 와 trash 먼저             10 · 12      44 lower-state 가 못 측정한 셋        21 (셋째) · 8절 (둘)
    7 merge.wait 을 센다            4 · 12     26 namespace 안의 root              10 · 18      45 lower_changed 가 보이면           19
    8 native 거절                   11         27 Check 마다 상태                  12 · 13      46 조각 8 의 준비도 점검             19 · 21
    9 병합 뒤의 창 (empty argv)      4 · 11     28 읽기 실패는 다시 해 볼 일          13           47 실패 경로와 상태 되돌림           11 · 12 · 13
   10 build 실패 뒤 merge 가 알리는 법 12 · 15    29 되풀이되는 오류                  13           48 환경 변수 이름과 원인 코드         2 · 11
   11 Grammar 의 굽기 절             11         30 Apply 뒤 upper 뿌리               12 · 13      49 대기 로그와 시계                  5 · 12
   12 build 의 exited                11         31 Result 를 로그와 metadata 에      5 · 12 · 13   50 먼저 잡은 하나가 재개              9 · 13
   13 merge 는 exited 를 안 보냄      12         32 trash 항목을 모을지 (항목마다)     10           51 시작 전 확인이 어긋났을 때         12 · 13
   14 result 의 build · merge 칸      2 · 4 · 5  33 조각 7 전체                     13 · 18 · 19 · 21  52 소유자의 굽기 Run 취소        9 · 12
   15 실패한 build 에도 build 칸      11         34 Preflight 의 마운트 확인          3 · 18       53 같은 커밋에 IR 태그 둘             5 · 16
   16 result.go 는 더하기만           2          35 merge 와 재개의 흐름 · watch 로그   5 · 12 · 13  54 운영 lower 확인                  19
   17 merge 가 합치기를 알리는 법      12         36 watch 의 때 · Unnamed             5 · 12       55 build · merge 의 수확              11
   18 build 의 종료 보고              11         37 ForeignMounts 를 배타 뒤에          12           56 실패의 기록 셋                    11 · 15
   19 build 의 두 예산                4 · 11     38 ForeignMounts 가 오류면            12
```

### 7.7 설계 결정 (1) ~ (55) -> 단계

```text
    1 재개는 HoldBake 를 안 부른다          13        29 TryBake · ReadState 오류는 FAILED       9 · 11 · 12
    2 merging 뒤 오류는 Apply 와 같은 길      12        30 재개는 잠금 뒤 state 를 다시 읽는다      13
    3 metadata 뒤 정리 오류는 DONE           12        31 대조 한 줄 · 그 안에서 export            5 · 16
    4 previous_ir 은 build 가 읽어 초안에     6 · 11    32 Finalize 예산이 pending 까지            11 (4절 4번)
    5 sha256 먼저 · 초안 fsync               6 · 11    33 대기 자리는 그 scratch 의 trash 로       6 · 9
    6 초안이 없으면 재개 안 함                13        34 StartBaker 첫 줄이 등록                 9
    7 pending_upper 의 모양                  5 · 13    35 대조를 못 하면 head_tags null            5 · 11
    8 합치면 last_attempt 지움               12 · 13   36 스크립트 — 모두 isolated                 19
    9 build 의 $OUT 은 노드의 것              11        37 조각 5 왕복 · 허용 오차                  4 · 12
   10 url 의 비밀번호                        5         38 조각의 sync · 표지 복사 경고             19
   11 head_tags 의 null 과 []                2 · 5     39 previous_ir 을 못 읽으면 null 과 경고     11
   12 merge 대기 줄은 단계 로그               12        40 OnStale 이름                           8
   13 데몬이 멈추면                          11 · 12 · 13 · 14   41 bash 확인                      5 · 11
   14 merge claim 의 표                     5 · 12    42 광고 주기 정리의 간격 · committed 는 기동  9 · 13
   15 기동의 버려진 자리 · abandoned reason   9         43 재시작 문장 넓힘 (49 가 바꿈)            5
   16 재시도 간격 · 실패가 아닌 셋            13        44 held 동안의 오류 길은 몸통               9 · 11 · 12
   17 bake_in_progress 둘째 문장             5 · 11 · 13  45 bash 확인의 갈래                        5 · 11
   18 보는 git 의 차례 · safe.directory       5 · 11 · 16  46 기록된 자리가 없으면 옮겨진 것        6 · 9 · 13
   19 LowerGuard 에 OnStale · Dir            8         47 origin 이 없어도 대조                   5 · 16
   20 조각 스크립트의 규칙                   19        48 helper 가 죽으면 runtime run:           11
   21 조각 7 의 두 경우                      19        49 재시작 줄 · reason 글자                  5 · 12
   22 exited 는 sync 의 결과로 (IR · pinned)  11        50 7.1 의 차례                            5 · 12
   23 마감과 잡힘이 같은 순간                 12 (lower 의 Exclusive 가 flock 을 먼저 건다 · 결정대로 둔다)
   24 재시작 문장 (abandoned 기록)            5 · 12    51 합칠 것 없음은 error 없는 DONE           12
   25 몸통이 held 를 비운다                  9         52 재시작 줄은 pending 갈래뿐               5 · 12
   26 끝났나 · 정리 차례                     5 · 12 · 13  53 정리는 못 옮겨도 committed             9 · 13
   27 재개는 DropBake 도 안 부른다            13        54 ir_mismatch 는 DONE 에                  2 · 11 · 15
   28 몸통의 쓰기 실패                       9         55 IR 어긋남은 skipping 줄 · 문장은 단계 로그  5 · 11
```

---

## 8. 이 단계가 하지 않는 것

```text
   실패한 build 의 upper 를 spool 로 보존                           checkpoint 유닛 (FD 흐름 12절)
   조각 6 · 7 · 8 을 돌리는 일                                      사람의 조각이다.  사용자가 SunnyVM 에서 돈다.  꺼져 있으면 보류
   lower-state 가 측정 못 한 셋 (받는 일 44) 가운데 둘 — ext4 밖       잔여 (FD 흐름 12절).  진행자가 적는다.  셋째 (옛 판 노드가 있는
     fsid 의 재부팅 · 전원이 나간 뒤 state.json — 과 repo init -b 뒤    lower) 는 잔여가 아니다 — Step 21 의 사람 조각에서 처음 본다
     manifests 의 로컬 태그
   정본(enode-design) 되돌림 · 팩 decisions.md · features.md ·        진행자가 올린다 (FD 흐름 13.1 · 13.2 — 13.2 가 넘긴 넷 가운데
     scene-gates.md (조각 7 · 8 의 문장) · requirements.md 10절          scene-gates.md 와 requirements.md 10절 을 이름으로 적었다)
     (decisions.md 에 더할 행)
   사람의 수단이 state.json 손 고침뿐인 두 갈래의 운영 문서             진행자 (FD 흐름 12절 끝)
   계약의 env 이름으로 노드의 비밀 (ENODE_TOKEN 등) 이 굽기 명령과      잔여 · 후속 과제 (CG 물음 2 답 · 2026-09-27T14:41:18Z).  오늘 명령 단계에도
     명령 단계의 환경에 실리는 길 — 이름에 제한이 없다                  있는 길이라 이 유닛이 만든 것이 아니다.  3.1 의 환경 줄과 Step 16 의 시험은
     (contract/bake.go:96 · env.go:107 ~ :111 · main.go:179)          「계약이 이름으로 부르지 않은 호스트 환경 변수는 닿지 않는다」 로 좁혔다.
                                                                    code-summary 의 잔여 · 후속 과제 절에 옮긴다 (Step 22)
   Mediator 의 새 라우트 · 매칭 규칙                                  바뀌지 않는다 (라우트 19)
   PR 과 병합                                                       조각 5 · 6 · 7 · 8 이 초록인 뒤 · CI 뒤.  올리기 전에 묻는다
```

---

## 9. 확장 준수 — Code Generation

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 계약 명령 줄은 3.1 이 규칙과 시험으로 받는다 (Step 6 · 10 · 11 · 12 · 16 · 18). 계약이 lower 에 남긴 것을 호스트가 실행하지 않는 것은 4절 33번과 Step 16 (CG 물음 3 답 A) · 격리 노드의 Finalize diff 는 4절 34번과 Step 7 · 18 (CG 물음 5 답). 합치기 줄 (같은 filesystem · 같은 마운트 · lower 밖으로 안 나감) 은 Step 3 · 18 과 merge-rules 가 받는다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 한 줄 (Step 5) · 흐름은 가짜 런타임과 진짜 잠금 |

---

## 10. 물음 — 다섯 모두 답을 받았다

QA 검수 (1회) 가 사용자에게 되물을 것으로 올린 넷과, 물음 3 의 답을 옮기며 호스트 git 이 도는 자리를 모두 찾다가 새로 보인 물음 5 는
**모두 답을 받았다** (audit 의 2026-09-27T14:41:18Z · 2026-09-29T13:53:38Z · 2026-09-29T15:05:21Z 항목). 답은 `[Answer]:` 에 시각과 함께
적었고, 반영한 자리는 물음마다 끝에 적었다. 기호 옆에 뜻을 적었다.

4절의 서른넷 가운데 FD 의 글자와 다른 수단을 고른 여섯은 4.1 머리에 모았고 근거 칸에 그 대조를 적었다 (2 · 4 · 5 · 7 · 18 · 29번).
반대하는 자리가 있으면 승인 대신 그 번호를 알려 주세요.

### Question 1 — merge 업로드의 3분 상한 (4절 15번)

물음을 낸 때의 계획은 merge 의 업로드에 기본 업로드 예산 3분을 전송 상한으로만 걸었다. 넘으면 노드 로그 경고 · DONE 그대로 · Upload 칸을 싣지 않는다.
정본 ADR-075 (단계는 워크스페이스가 아니라 commit set 을 돌려준다) 결정 5 는 업로드 예산을 넘기면 FAILED `upload_timeout` 이고 명령
실패와 구분되는 원인으로 남는다 (ADR-075:437 ~ :442 · 원인 코드는 이미 `internal/contract/result.go:94`). ADR-077:261 ~ :263 이 merge 에서
빼는 것은 Finalize 예산뿐이다. FD 규칙 5절 (business-rules.md:228 ~ :229) 은 「merge 는 예산을 안 쓴다」 의 근거로 팩 결정 3-11 을 댔는데,
3-11 (decisions.md:100) 은 대기 상한과 합치기 본체만 말한다. 계획대로면 상한을 넘었을 때 `merged` 가 produced 에 없어 계약의 조건이 Run
을 FAILED 로 판정하지만 Record 에는 그 까닭이 어디에도 남지 않는다. 상한이 필요한 까닭은 맞다 — `Client.Upload` 에는 요청마다의 상한이
없다 (`upload.go:350` ~ `:358`).

A) 정본 글자대로 — merge 단계 FAILED · 원인 `upload_timeout`. lower 는 합쳐졌는데 Record 는 실패를 말한다 (ADR-077:290 ~ :292 의 재개 때와
같은 모양)

B) DONE 을 두고 result 의 `upload` 칸에 `timeout` 을 싣는다. 정본 되돌림 (ADR-075 §10.2 · ADR-077 §6 · mediator-api) 에 올린다

C) FD 글자대로 상한 없이 runCtx 에만 묶는다. Mediator 가 멈추면 Worker 도 임대가 끝날 때까지 멈춘다

D) 계획 그대로 — 상한은 두고 Record 에는 아무 칸도 싣지 않는다

E) Other (please describe after [Answer]: tag below)

[Answer]: A (2026-09-27T14:41:18Z · 「물음 1 = A 정본대로 (권장)」)

반영한 자리 — 4절 15번 (명령 단계와 같은 모양 · 계약은 merge 의 예산을 못 바꾼다 · Mediator 는 막지 않는다) · Step 12 의
`TestMergeStep_UploadBudgetFailsAfterTheMerge` · Step 15 의 `TestBake_MergeUploadTimeoutFailsTheRun` · FD 규칙 5 · 7 · 16.1절과 FD 흐름 3 · 7절
(근거 「CG 물음 1 답 A」)

### Question 2 — 계약의 env 이름으로 노드의 비밀이 굽기 명령에 실리는 길

계약의 env 이름에는 제한이 없다 (`contract/bake.go:96` · `env.go:107` ~ `:111`). `env: ["ENODE_TOKEN"]` 이면 노드가 읽는 토큰
(`cmd/enode/main.go:179`) 이 sync 의 환경에 실린다. 컨테이너는 network namespace 를 열지 않는다 (`runc_overlay_linux.go:1262`). 오늘의
명령 단계 (`claim.go:737`) 에도 있는 길이라 이 유닛이 새로 만든 것은 아니다. 다만 물음을 낸 때의 계획 3.1 은 시험 이름으로 「호스트 비밀이 닿지 않는다」 고 적었다.

A) 잔여로 적고, 3.1 의 문장과 시험 이름을 좁힌다 (t.Setenv 로 둔 호스트 변수 가운데 계약이 이름을 적지 않은 것이 닿지 않는다)

B) 계약 env 에서 `ENODE_` 로 시작하는 이름을 거절한다 — 오늘 명령 단계의 뜻까지 바뀐다

C) Other (please describe after [Answer]: tag below)

[Answer]: A (2026-09-27T14:41:18Z · 「물음 2 = 잔여로 적는다 (권장)」)

반영한 자리 — 3.1 표의 「환경」 줄 (「계약이 이름으로 부르지 않은 호스트 환경 변수는 닿지 않는다」) · Step 16 의 시험 이름과 뜻
(`TestBakeBuild_UnnamedHostVariablesDoNotReachTheCommands`) · 8절의 잔여 줄 · Step 22 (code-summary 의 잔여 · 후속 과제 절)

### Question 3 — 굽기 뒤 lower 에 남은 계약의 파일로 호스트의 git · repo 가 도는 길

굽기 전에는 runc-overlay 노드에서 계약이 쓴 것이 모두 버려졌다. 굽기 뒤에는 계약의 sync 가 쓴 `.git/config` 와 `.repo/` 가 lower 에
남는다. runc-overlay 노드의 워크스페이스는 lower 루트다 (`internal/enode/environment.go:26` — Binding 의 Workspace 가 노드 설정의 Workspace). workspace.repo 를 적은
단계가 오면 Prepare 가 호스트에서 그 자리에 `git reset --hard` · `git clean -df` 나 `repo forall` 을 돈다 (`claim.go:555` ~ `:557` ·
`workspace.go:94` ~ `:118`). QA 의 측정 (git 2.39.2) — core.fsmonitor 가 reset --hard · clean -df 에서 실행됐다 · DetectRepo 가 쓰는
config --get · rev-parse 에서는 실행되지 않았다. repo 런처가 lower 의 `.repo/repo` 코드를 실행하는지는 이 기계에 repo 가 없어 측정하지
못했다. 물음을 낸 때의 계획 3.1 은 「명령이 도는 길」 만 다루고 「명령이 남긴 것을 호스트가 읽는 길」 은 다루지 않는다 — 유닛 정의 7절의 보안 문장
(계약의 명령은 격리 안에서만 돈다) 을 비켜 가는 길이다.

A) 이 유닛에서 막는다 — isolated 노드에서 Prepare 가 lower 에 호스트 git · repo 를 돌리지 않게 한다

B) 이 유닛에서 좁힌다 — 호스트 git 을 fsmonitor 와 hooks 를 끈 채 돌린다 (repo 런처의 길은 남는다)

C) 잔여로 두고 정본 되돌림에 올린다

D) Other (please describe after [Answer]: tag below)

**다시 물은 판** (진행자 · 2026-09-29) — 첫 답은 되물음이었다 (「Repo 일 때는 어떻게 되는데?」 · 2026-09-27T14:41:18Z). 진행자가 repo 트리의
길 (`repo forall` · 런처가 그 트리의 `.repo/repo` 를 실행) 과 정본 ADR-072 결정 3 (§5 · §5.2 · §8 의 닫힌 칸) · ADR-073 을 들어 고쳐 물었다 —
A 이 유닛에서 ADR-072 결정 3 을 코드로 옮긴다 (runc-overlay 노드의 Prepare 는 호스트에서 git · repo 를 돌리지 않고 새 upper 로 PrepClean 을
얻는다 · 저장소 확인 (읽기만) 은 남긴다 · native 는 오늘처럼 reset 과 clean · 행렬 밖 workspace.go · 정본 되돌림 없음) · B 후속 유닛으로 뗀다.

[Answer]: A (2026-09-29T13:53:38Z · 다시 물은 판의 A — 「계속해.」 를 권장 A 로 받았다)

반영한 자리 — 3.1 머리 (계약이 lower 에 남긴 것을 호스트가 실행하지 않는 것) · 3.1 표의 「호스트가 lower 에 돌리는 명령」 줄과 「호스트
git · repo 가 도는 자리」 표 · 4절 33번 · Step 16 (workspace.go 와 `workspace_linux_test.go` 의 시험 넷) · 2절 (행렬 밖 workspace.go) ·
unit-of-work.md 7절 · FD 흐름 10 · 13.1절 · 7.3. 표를 짓다 새로 보인 길은 물음 5 (답 — 4절 34번)

### Question 4 — 장면 2 의 4 「형제의 Run 이 끝나자 합치기가 몇 초에 끝난다」 를 어디부터 세나

형제는 임대가 0 인 drain 응답을 두 번 받은 뒤에 공유를 놓는다 (`lowerguard.go:346` ~ `:349` · lower-state 답 1). 그래서 형제 Run 이 끝나고
60 ~ 120 초 뒤에 합치기가 시작되고, 합치기 자체는 몇 초다 (scene-gates.md:28). 물음을 낸 때의 계획 7.1 은 이 기준에 고정 비용 벤치마크 하나만 댔다.
조각 6 을 사람이 판정하기 전에 「몇 초」 가 어디부터인지 정해야 한다. 답과 무관하게 slice-6.sh 는 네 시각 (형제 Run 끝 · released the lower
lock · took the lower lock after · committed) 을 기대 창과 함께 보인다 (Step 19).

A) 배타 잠금을 잡은 때 (`took the lower lock after`) 부터 committed 까지 — 앞의 60 ~ 120 초는 lower-state 답 1 의 설계값이다

B) 형제 Run 이 끝난 때부터 committed 까지 — 두 광고 주기를 줄이는 설계 변경이 든다

C) Other (please describe after [Answer]: tag below)

[Answer]: A (2026-09-27T14:41:18Z · 「물음 4 = 배타 잠금부터 (권장)」)

반영한 자리 — 7.1 조각 6 의 「장면 2 의 4」 줄 (「몇 초」 는 took the lower lock after 부터 committed 까지 · 앞의 60 ~ 120 초는 lower-state
답 1 의 설계값) · Step 19 의 slice-6.sh (took 부터 committed 까지를 따로 한 줄)

### Question 5 — isolated 노드의 Finalize diff 가 runtime helper 안에서 호스트 git · repo 를 도는 길

effect edit 이고 workspace 를 적은 run · agent 단계는 Finalize 에서 workspace.diff 를 만든다 (`finalize.go:72`). runc-overlay 노드에서는 그
일을 runtime helper 가 한다 — helper 는 호스트의 enode 바이너리이고 `unshare --user --map-root-user --mount` 뿐인 namespace 의 root (노드 uid 로
매핑 · PID · network namespace 없음 · `runc_overlay_linux.go:233` ~ `:237`) 이며 컨테이너 밖이다. helper 는 merged view 에서 호스트의 git
(`git read-tree` · `git add -N .` · `git diff --binary` · 임시 인덱스) 이나 `repo forall -c <script>` 를 돈다 (`runc_overlay_linux.go:1077` ~
`:1090` · `runtime.go:219` ~ `:220` · `diff.go:51` ~ `:150`). merged view 에는 그 단계의 명령이 방금 upper 에 쓴 것과, 굽기 뒤라면 lower 에 남은
계약의 `.git/config` · `.repo/repo` 가 보인다. 진행자가 스크래치에서 diff.go 와 같은 차례를 git 2.39.2 로 돌리니 저장소의 core.fsmonitor 가 넷 ·
filter.<이름>.clean 이 둘 실행됐다 (audit 2026-09-29T14:13:46Z). repo 모양은 런처가 트리의 `.repo/repo` 코드를 실행하므로 git 설정을 끄는 것으로
막을 수 없다. lower 에는 쓰지 않는다 (merged 의 쓰기는 upper 로 · 인덱스는 임시 파일). 단계 하나 안의 길은 굽기 전에도 있었고, 굽기는 그것을
다른 Run 으로 넓힌다 — 굽기 계약이 lower 에 남긴 설정을 뒤의 편집 단계의 diff 가 읽는다. 물음 3 의 답은 Prepare 만 말했다.

진행자가 물은 선택지 넷이다 (2026-09-29).

A) 이 유닛에서 세션 안으로 — isolated 노드의 Finalize diff 를 세션 안 (준비된 rootfs 의 git · repo 모양은 rootfs 의 repo) 에서 만든다. 호스트의
runtime helper 는 merged view 에 git · repo 를 돌리지 않는다. native 는 그대로. finalize 유닛의 수확을 고친다

B) 따로 유닛으로 바로 다음에 — 이 유닛은 그대로 두고 바로 뒤에 수확을 고치는 유닛을 둔다

C) 격리 노드에서 diff 끄기 — isolated 노드의 Finalize 는 diff 를 만들지 않고 진단 칸에 까닭을 적는다. runc-overlay 노드의 workspace.diff 가
나오지 않는다

D) 잔여로 둔다 — 후속 과제로 적는다 (8절과 code-summary)

E) Other (please describe after [Answer]: tag below)

[Answer]: A — 이 유닛에서 세션 안으로 (2026-09-29T15:05:21Z · 사용자가 진행자의 넷 가운데 권장을 골랐다 · 「물음 5 = 이 유닛에서 세션 안으로 (권장)」)

반영한 자리 — 4절 34번 · 4절 끝의 `sessionDiffScript` · 3.1 의 표 둘 (「Finalize 의 workspace.diff」 줄 · 「호스트 git · repo 가 도는 자리」 의
diff.go 줄과 collect · stat · 훑기 줄) 과 글 셋 (굽기 전부터 있던 구멍 · 세션 안 git 의 등급 · 표 밖으로 가지 않게) · Step 7 (코드와 시험 넷) ·
Step 18 (SunnyVM 의 진짜 runc 한 줄) · 2절 (행렬 밖 diff.go · runc_overlay_linux.go 의 finalize · runtime.go 의 주석) · unit-of-work.md 7절 ·
FD 흐름 10 · 13.1절 (굽기 전부터 있던 구멍 · 정본과 어긋나지 않음 · 보탬 제안) · 8 · 9절 · Step 22
