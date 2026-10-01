# `checkpoint` — Code Generation 요약

계획은 `construction/plans/checkpoint-code-generation-plan.md` (승인 · 물음 1 = A) 이다. 설계는 FD 셋 (`functional-design/`) · NFR
둘 (`nfr-requirements/`) · NFR Design 둘 (`nfr-design/`) 이고 아래에서 「규칙 · 엔티티 · 흐름 · NFR · 패턴 N절」로 가리킨다.
기준은 `03b6eef` 이고, 이 기록은 커밋 전의 작업 트리다. 작성 2026-10-01T00:26:42Z.

---

## 1. 파일

**새 파일**

| 파일 | 줄 | 행렬 | 무엇 |
|---|---|---|---|
| `internal/scratch/checkpoint.go` | 451 | 안 | 어휘 · `Policy` · `Admit` · `PlanSettle` · `PlanReconcile` · `NewID` · `Store` · `Reservation` 의 모양 |
| `internal/scratch/checkpoint_unix.go` | 610 | 안 | spool 자리 확인 · 예약 · 확정 · 버림 · 보고의 성패 · 보고 뒤 판정 · 조정 · 목록 · 기록과 요약 |
| `internal/scratch/checkpoint_other.go` | 52 | 안 | 「지원하지 않음」 |
| `internal/enode/checkpoint.go` | 508 | **밖** | Keeper (규칙 1절 · 판정 · 확정 · 버림 · 판정 고루틴) · `StartScratch` |
| `internal/enode/checkpoint_cmd.go` | 251 | **밖** | `RunCheckpointCmd` — `enode checkpoint list \| show` |
| `scripts/finalize-bake/slice-9.sh` | 236 | 안 | 조각 9 |
| 시험 — `internal/scratch/checkpoint_test.go` 171 · `checkpoint_unix_test.go` 473 · `internal/enode/checkpoint_test.go` 696 · `checkpoint_cmd_test.go` 217 · `config_checkpoint_test.go` 57 | | 시험 | 3절 |

**고친 파일** (`git diff --stat` · 22 파일 · +734 · -59)

| 파일 | 행렬 | 무엇 |
|---|---|---|
| `internal/contract/result.go` (+3) | **밖** | `Diagnostics.Checkpoint` |
| `internal/scratch/deleter.go` (+7) | **밖** | `Usage` 의 spool 칸 넷 |
| `internal/enode/runtime.go` (+42 · -2) | 안 | `Keep.By` · `Keep.Result` · `KeepResult` · `CaptureSupport` · native 의 Capture |
| `internal/enode/runc_overlay_linux.go` (+40 · -6) · `runc_overlay_other.go` | 안 | Close 의 `keepUpper` · Capability 의 Capture |
| `internal/enode/claim.go` (+64 · -11) | 안 | `Result.CheckpointCapture` · `Worker.Checkpoints` · 세션을 연 뒤 예약 · `closing` 칸 셋 · closeOut 의 차례 · report 의 네 끝 |
| `internal/enode/bake_build.go` (+9 · -2) | **밖** | 세션을 연 뒤 예약 · `failEnd` 와 `succeed` 의 `closing` |
| `internal/enode/config.go` (+70) | 안 | `Local.Checkpoint` · `CheckpointPolicy` · 검증 |
| `internal/enode/status.go` (+45 · -3) | 안 | `Status.Checkpoint` · `SetSpool` · `SetCheckpoint` · `SetScratch` 가 spool 칸을 지킨다 |
| `internal/enode/trash_linux.go` (+43 · -4) · `trash_other.go` (+7) | **밖** | trash-helper 의 `--measure` · `MeasureLauncher` |
| `internal/panel/view.go` · `page.go` (+18) | 안 · **밖** | `State.Checkpoint` · spool 줄 · 정책 줄 · 안내 줄 |
| `cmd/enode/main.go` (+13 · -21) | 안 | `checkpoint` 입구 한 줄 · runc-overlay 기동을 `StartScratch` 한 줄로 |
| 시험 — `finalize_worker_test.go` · `runc_overlay_linux_test.go` · `runc_overlay_integration_test.go` · `status_test.go` · `trash_test.go` · `panel_test.go` · `result_test.go` · `main_test.go` | 시험 | 3절 |

---

## 2. 규칙의 자리 (설계 → 함수)

| 설계 | 자리 |
|---|---|
| 규칙 1절 (요구하나) | `CheckpointKeeper.requested` · closeOut 의 `failed` (굽기는 부르는 쪽의 것만) |
| 규칙 2절 ① ~ ⑤ | `Keeper.Decide` · `Policy.Admit` · `Store.SameFilesystem` |
| 규칙 2절 ⑤ ~ ⑦ | `runcOverlaySession.keepUpper` · `Keeper.Finish` · `finishKeep` · `Store.Commit` · `Abandon` |
| 규칙 3절 (keep 의 실패만 뗀다) | `Keep.Result` 가 있으면 `keepUpper` 가 결과에만 적는다 · 나머지 Close 오류는 오늘 그대로 |
| 규칙 4절 (칸이 있는 때) | closeOut 만 `res.CheckpointCapture` 를 채운다 · Keeper 가 nil 이면 칸 없음 |
| 규칙 5 · 6절 · 패턴 2 · 3절 | `Store.Settle` (잠금 → 측정은 밖 → 다시 읽고 적기) · `PlanSettle` · `writeSummary` · `Store.Kept` |
| 규칙 7 · 8절 | `Store.Commit` 의 `expires_at` · `Worker.report` 의 `outcome` · `Keeper.Reported` · `Store.Report` |
| 규칙 9절 | `Store.Reconcile` · `PlanReconcile` · `probeLock` |
| 규칙 10절 · NFR 1절 | `validateCheckpoint` · `Local.CheckpointPolicy` |
| 규칙 11절 · NFR C1 · C7 · C8 | `Store.Open` · `Reserve` (O_NOFOLLOW · Fchmodat) · `readJSON` (1 MiB · 보통 파일) · `writeJSON` (0600) |
| 규칙 12 · 14절 | `StartScratch` 의 기동 로그 · `page.go` 의 `checkpointLine` · `checkpointNote` · Keeper 의 로그 |
| 규칙 13절 | `RunCheckpointCmd` · `Store.List` · `Lookup` |
| 패턴 1절 (예약을 세션을 열 때로) | `claim.go` 의 세션 열기 뒤 · `bake_build.go` 의 세션 열기 뒤 · 확정은 `closedAt` 뒤 |
| 패턴 4절 | `Keeper.Kick` · `Keeper.Run` · `MeasureLauncher` (idle · CPU 19) |

---

## 3. 코드 검사와 시험

| 검사 | 결과 |
|---|---|
| `gofmt -l .` | 빈 출력 |
| `go vet ./...` · `go vet -tags integration ./internal/enode/ ./internal/scratch/ ./internal/merge/ ./internal/lower/` · `go build ./...` | exit 0 |
| 시험 (`ci.yml:216` · 시험 DB · 가짜 claude 스텁) · 커버리지 단계 (`ci.yml:267` 의 명령과 awk) | exit 0 (69초) · 통과 2,675 · 실패 0 · 스킵 0 · 미달 0. 주석과 린트 고침 뒤 한 번 더 — 같다 |
| 스킵 감시 (`ci.yml:341` 의 스텝) | 패키지 24 · 허용목록 항목 0 · 허용목록 밖의 스킵 0 |
| 경계 (`TestImportBoundaries`) | 통과 — `internal/scratch` 가 새로 부르는 것은 표준 라이브러리뿐 |
| 크로스 빌드 셋 (`ci.yml:456-458`) | exit 0 |
| `enodectl.exe` 심볼 | crypto/tls 1 · net/http 6 (상한 10 · 50) |
| U+2605 · `glyphscan` | 0 · 「173 files scanned, no decorative glyph in any string literal」 |
| `golangci-lint run ./...` | 38 — 기준선과 같은 목록. 새 코드에서 난 하나 (드모르간 · `ValidID`) 를 고쳤다 |
| 조각 0 | 위 초록 · `mux.HandleFunc` 19 |

**커버리지** (CI 명령 · 병합 전 `03b6eef` 과 뒤)

| 패키지 | 전 | 뒤 |
|---|---|---|
| `cmd/enode` | 80.8% (198/245) | 83.7% (200/239) |
| `internal/enode` | 86.7% (4,584/5,286) | 86.8% (4,960/5,715) |
| `internal/scratch` | 93.5% (231/247) | 88.1% (609/691) |
| `internal/panel` | 89.1% (245/275) | 89.1% (246/276) |
| `internal/contract` | 92.5% (760/822) | 92.5% (760/822) |
| 전체 | 87.7% (12,554/14,319) | 87.6% (13,307/15,187) |

파일마다 — `internal/scratch/checkpoint.go` 94.5 · `checkpoint_unix.go` 82.7 · `internal/enode/checkpoint.go` 84.4 · `checkpoint_cmd.go` 85.8 ·
`config.go` 100 · `status.go` 96.9 · `claim.go` 90.3. `cmd/enode` 는 runc-overlay 기동을 `StartScratch` 로 옮겨 문장이 여섯 줄고 80% 까지의
여유가 문장 둘에서 여덟으로 늘었다 (계획 3.2).

**새 시험의 흔들림** — 새로 쓰거나 고친 시험 53 을 `-count=20` 으로: 통과 1,060 · 실패 0 · 스킵 0. `go test -race -count=1 -timeout 30m
./internal/enode/ ./internal/scratch/` — exit 0 (148초).

**시험 이름** (계획 6절) — 계획의 이름 그대로 지었고 셋을 더했다 (`TestReserve_FailsWithoutASpool` · `TestReport` ·
`TestCheckpointFinish_RecordFailure`). P2 는 `TestCheckpoint_SlowReservationKeepsFinalize` (예약과 확정에 300 ms 를 끼우고 Finalize 예산
100 ms — finalize `ok` · error 없음 · exit 그대로 · 보존만 `failed`(`lease_budget`)). C2 는 `TestCheckpoint_NoHostPathLeaves` (scratch 경로에
표지 글자 · 갈래 열 — 옮김 · 늦음 · rename 실패 · abort · native · 다른 filesystem · 여유 · 몫 · 예약 실패 · 거절한 spool — result JSON 과 올린
단계 로그에 그 글자가 없다).

**integration 태그** (CI 밖 · 사람이 SunnyVM 에서) — `TestIntegrationCheckpoint_CapturesSubuidAndWhiteouts` · `_MeasureInHelper` ·
`_UnshareReadsRootFiles` (NFR C4) · `_WindowIndependentOfSize` (NFR P1 · P3). **이 기계에서는 컴파일만 확인했다** — `ENODE_RUNC_*` 가 없어
스킵이고, 있어도 이 기계는 `newuidmap` 이 uid_map 을 못 쓴다 (FD 계획 2.2). 측정 helper 가 시험 바이너리를 다시 실행하도록 integration 의
`TestMain` 에 `trash-helper` 갈래를 더했다.

---

## 4. 조각 9 — 돌리는 법 (사람 · SunnyVM)

runc-overlay 노드의 설정에 `checkpoint: {ttl_hours: 1, max_gb: 1}` 을 적고 띄운다. native 노드 하나를 같이 띄워 두면 4 를 돈다.

```sh
export M=http://<mediator>:8080 T=<bootstrap token>
export NODE_CONFIG=<runc-overlay 노드의 설정> WS=<그 워크스페이스>
export NATIVE_WS=<native 노드의 워크스페이스> ENODE=<그 노드의 enode>
scripts/finalize-bake/slice-9.sh
```

스크립트는 판정하지 않는다 — 여덟 줄마다 무엇을 볼지 출력하고 Enter 를 기다린다. 설정을 고치고 데몬을 다시 띄우는 셋 (5 · 6 · 7) 은
`do:` 로 멈춘다. 첫 보존본을 가장 먼저 잡아 TTL 시계를 걸고 나머지를 도는 동안 기다리므로, 8 까지 한 시간 반쯤 걸린다. 7 은 `min_free_gb`
를 지금 여유보다 1 GB 작게 두고 단계가 2 GB 를 써서 하한 아래로 내려간다 — 미리 채우면 노드가 drain 해 단계를 못 집는다. 끝나면
`min_free_gb` 를 되돌린다.

---

## 5. 계획과 다른 자리

| # | 자리 | 무엇 · 까닭 |
|---|---|---|
| ① | `remove_unix.go` · `remove_other.go` | **고치지 않았다.** `Measure` 는 이미 어느 뿌리든 `O_NOFOLLOW` 로 열어 받는다 — 뿌리는 항목 폴더, 이름은 `upper` |
| ② | trash-helper 의 쓰는 법 | `--measure` 는 첫 인자로 받고, 쓰는 법 문장은 그대로 두었다 — 숨은 입구이고 기존 시험이 그 문장을 본다 |
| ③ | native 노드의 Keeper | FD 흐름 4절은 「native 면 Keeper 없음」 이다. 규칙 2절 ② 와 조각 9 의 4 는 native 의 요구한 단계에 `unsupported`(`runtime`) 를 싣는다 — 그래서 Store 없는 Keeper 를 둔다 (`StartScratch`) |
| ④ | `checkpointPause` | 시험이 예약 뒤와 확정 앞에 늦음을 끼우는 이음매 — 제품에서는 할 일이 없다 (`lowerTrashPriority` 와 같은 모양) |
| ⑤ | 기다리는 항목 잠금 | 예약은 자기 항목 잠금을 기다려 쥔다 (`LOCK_EX`). 조정과 조회는 `LOCK_NB` 로 한순간만 쥐어 보므로 그 사이에 예약이 실패하지 않게 |
| ⑥ | statfs 를 못 할 때 | 받아들임은 여유와 몫을 둘 다 거르지 않고 (광고의 여유 부족 drain 과 같다), 판정은 두 몫을 안 본다 (`Filesystem.Known`) — 여유 0 으로 읽으면 보존본을 모두 퇴출한다 |
| ⑦ | spool 을 못 만들 때 | 기동 로그 warn `checkpoint reconcile failed` (err) 로 적는다. 요구한 단계는 예약 실패의 `failed`(`io`) 가 된다 |
| ⑧ | 측정 실패의 로그 | debug `cannot measure a checkpoint; the next settle measures it again` (id · err) — 규칙 14절에 없던 줄이다. helper 를 못 띄우면 규칙 14절대로 warn `checkpoint settle failed` |
| ⑨ | `go test` 픽스처 | `TestReserve_DisposedWhenNotRequested` 는 예약이 세션을 열 때 생기는지 `checkpointPause("reserved")` 로 본다 |

**사용자에게 보이는 문구가 초안과 달라진 곳**

| 자리 | 초안 | 지금 | 까닭 |
|---|---|---|---|
| 조회의 크기 | `4.2 GiB` 만 | 1 GiB 아래는 `MiB` · `KiB` | 작은 보존본이 `0.0 GiB` 로 적힌다 |
| `show` 의 `evicted` 줄 | 열쇠 뒤 빈칸 다섯 | 다른 줄과 같은 칸 (열쇠 12 + 빈칸 하나) | 초안만 한 칸 어긋났다 |
| scratch 가 없는 노드의 `show` | 정하지 않았다 | `list` 와 같은 문장 · exit 1 | 보일 것이 없다 |
| diagnostics 의 예비 문장 | 없다 | `reserving a place in the spool failed: no reservation was made` · `…: unexpected error` · `moving the upper into the spool failed: the runtime did not move it` | 일어나지 않아야 하는 갈래다. errno 가 없을 때 경로를 담은 오류 문장을 싣지 않으려고 둔다 |
| 제어판의 spool 줄 | `측정 13:02:15` | 판정 전이면 `측정 없음` | trash 줄과 같은 규칙 |

---

## 6. 파일 행렬 밖의 diff

| 파일 | diff |
|---|---|
| `internal/contract/result.go` | `Diagnostics` 에 ``Checkpoint string `json:"checkpoint,omitempty"` `` 한 칸과 주석 둘 (FD 답 9) |
| `internal/scratch/deleter.go` | `Usage` 에 `SpoolBytes` · `Checkpoints` · `SpoolUnsized` · `SpoolMeasuredAt` (FD 엔티티 7절 · FD 흐름 9절에서 빠졌다) |
| `internal/enode/bake_build.go` | `buildRun.slot` · 세션을 연 뒤 `Reserve` 와 `defer Dispose` · `failEnd` 의 `closing{…, failed: true, bake: true, slot}` · `succeed` 의 `bake: true, slot` |
| `internal/enode/trash_linux.go` | `RunTrashHelper` 의 `--measure` (지우지 않고 측정 줄 하나 · exit 0) · `measureHelperCommand` · `MeasureLauncher` · `runTrashHelper` 의 `measureOnly` |
| `internal/enode/trash_other.go` | `MeasureLauncher` 의 「띄울 수 없다」 |
| `internal/panel/page.go` | `measured` · `checkpointLine` · `checkpointNote` · spool 줄과 정책 줄 · 안내 줄 (규칙 12절의 한국어) |
| `internal/enode/checkpoint.go` · `checkpoint_cmd.go` | 새 파일 — 행렬의 ⑧ 칸에 `internal/enode` 새 파일이 없다 |
| `internal/enode/finalize_worker_test.go` | 물음 1 의 답 — `TestFinalize_ACommandKilledByASignal` 이 종료 보고 0 이나 1 을 받는다 (1 이면 signal 9) |

---

## 7. 넘기는 것

| 누구 | 무엇 |
|---|---|
| 진행자 · 커밋 | 계획 Step 17 의 둘째와 셋째 줄 (`aidlc-state.md` · `audit.md` · 한 커밋). 이 계획의 체크박스는 Step 1 ~ 16 과 Step 17 의 첫 줄을 채웠다 |
| 진행자 · FD 문서 | 흐름 4절 「native 면 Keeper 없음」 → Store 없는 Keeper (5절 ③) · 흐름 9절의 행렬 밖 표에 `internal/scratch/deleter.go` 를 더하고 `remove_unix.go` 를 뺀다 · 엔티티 3절의 `Lower` 칸 뜻 (lower-state 의 Identity → `lower.ReadRoot` 의 키 · 계획 3.4 ①) · 규칙 13절에 scratch 없는 `show` 의 exit 1 · 규칙 14절에 측정 실패의 debug 줄 |
| 진행자 · 정본 | FD 흐름 12절의 열둘과 `nfr-requirements.md` 8절의 다섯 그대로. 코드가 더한 것은 없다 |
| 진행자 · bake | `bake/functional-design/business-rules.md:118-120` 의 규칙 3 근거 문장 (FD 흐름 2절) |
| 사람 · 조각 9 | 4절. integration 넷도 같은 자리에서 돈다 — `ENODE_RUNC_ROOTFS` · `ENODE_RUNC_TEST_ROOT` 를 적고 `go test -tags integration -run TestIntegrationCheckpoint ./internal/enode/` |
