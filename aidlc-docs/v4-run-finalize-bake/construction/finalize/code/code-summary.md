# `finalize` — 코드 요약

**유닛** `finalize` (결과 확정 · 한 줄 순서의 셋째) · **브랜치** `unit/finalize` · **기준** `572ec87`
(Functional Design 커밋) · **계획** `construction/plans/finalize-code-generation-plan.md` (단계 열일곱)

노드가 명령이 끝난 뒤의 구간(결과 확정 · 닫기 · 업로드 · 보고)을 두 예산 안에서 닫는다. Mediator 는 한 줄도
안 바뀌었다 — 받는 쪽은 앞 유닛 step-phase 가 지었다. 아래의 **FD** 는 이 유닛의 Functional Design 산출물 셋
(`construction/finalize/functional-design/`)의 줄임이고, 「계획 4절 ③」은 계획 파일의 4절 셋째 결정을 뜻한다.

---

## 1. 파일

| 파일 | 새 · 고침 | 무엇 |
|---|---|---|
| `internal/enode/finalize.go` | 새 (372 줄) | 규칙의 순수 함수 — `contractStep` · `stepBudgets` · `finalizeSpecFor` · `exitOutcome` · `exitedStop` · `exitReportLog` · `exitBackoff` · `settle` · `diagnosticsFor` · `logTail` · `withTail` · `logFinalize`. 종료 보고 goroutine — `exitReporter` · `Worker.startExitReport` · `Stop` |
| `internal/enode/runtime.go` | 고침 (+91 −38) | `HarvestSpec` · `HarvestResult` 를 `FinalizeSpec` · `FinalizeResult` · `Discovery` 로 · `Keep` · `StepSession.Finalize` · `Close(ctx, Keep)` · `nativeSession.Finalize` · `finalizeLocal` (native 와 overlay helper 가 함께 쓴다) |
| `internal/enode/changed.go` | 고침 (+188 −9) | 상한 넷과 `discoverTime` · `discoverLimits` · `walker` · `walkWorkspace` · `walkUpper` · 세는 자리 `statPath` · `walkDir`. `changedName` 을 지웠다 |
| `internal/enode/claim.go` | 고침 (+224 −204) | `Step` 칸 셋 · `Result` 칸 여섯 · `Client.Exited` · `exitRejected` · `Worker.budgets` · `Worker.exitWait` · 명령 단계와 agent 단계의 흐름 · `afterExit` · `closeAndUploadLog`. `uploadProduced` · `logHarvest` · `writeChangedNote` · `writeHarvestNote` 를 지웠다. `UploadLog` · `PutBlob` 은 `upload.go` 로 옮겼다 |
| `internal/enode/upload.go` | 고침 (+170) | `Client.upload` · `BlobRejected` · `UploadLog` · 흘려 보내는 `PutBlob` · `put` · `Worker.upload` · `uploadStopped` · `uploadNames` · `putFile` |
| `internal/enode/runc_overlay_linux.go` | 고침 (+103 −26) | helper 요청 `finalize` · `finalized` · `helperGrace` · 세션 `Finalize` (마감 뒤 5초에 abort) · `Close(ctx, Keep)` 와 abort 표시 · helper `finalize` (요청의 마감으로 ctx · upper 훑기) · `isOverlayWhiteout` |
| `internal/enode/collect.go` | 고침 (+35 −4) | `collectDeclared(ctx, …)` · `copyFile(ctx, …)` · `copyWithin` (1 MiB 마다 마감을 본다) |
| `internal/enode/runner.go` | 고침 (+10) | `Job.Exited` 콜백 한 칸과 부르는 한 줄 |
| `internal/enode/advertise.go` | 고침 (+6) | `Client.Upload` 한 칸 (4절 ①) |
| `cmd/enode/main.go` | 고침 (+5) | `Upload: &http.Client{}` 와 주석 |
| `internal/enode/runc_overlay_other.go` | 안 고침 | 세션 타입이 없다 — `Open` 이 늘 실패한다. 확인만 했다 |
| `scripts/finalize-bake/slice-1.sh` | 새 (178 줄) | 조각 1 (걷지 않는다) |
| `scripts/finalize-bake/slice-2.sh` | 새 (199 줄) | 조각 2 (보인다) — 사람이 돈다 |
| 시험 | 새 둘 · 고침 여덟 | 새 `finalize_test.go` (552 줄) · `finalize_worker_test.go` (495 줄 · `!windows`). 고침 `changed_test.go` · `collect_test.go` · `upload_test.go` · `worker_test.go` · `runtime_test.go` · `runtime_projection_test.go` · `runc_overlay_linux_test.go` · `runc_overlay_integration_test.go` (`integration` 태그 · CI 밖) |

**행렬 밖 파일** — `finalize.go` (계획 4절 ①) · `runner.go` · `collect.go` (FD 흐름 10절) · `advertise.go` (4절 ①).
`internal/store` · `internal/api` · `internal/contract` · `internal/enode/diff.go` · `hook.go` 는 안 고쳤다.

**없앤 이름** — `HarvestSpec` · `HarvestResult` · `StepSession.Harvest` · `changedName` · `uploadProduced` ·
`logHarvest` · `writeChangedNote` · `writeHarvestNote`. `workspace.changed` 라는 산출물 이름이 저장소에서 사라졌다.

---

## 2. 규칙이 어디에 있나

| 규칙 (FD) | 함수 |
|---|---|
| effect 가 정하는 수확 (규칙 1절) | `finalizeSpecFor` |
| 두 예산과 기본값 (규칙 2절) | `stepBudgets` · `Worker.budgetsFor` (`contract.Step.Budgets` 를 부른다) |
| 예산의 경계 — Finalize · 닫기 · finalized_at · 업로드 (규칙 2절 · 흐름 1 · 2절) | `Worker.afterExit` |
| 넘었을 때 멈추는 것 (규칙 3.1) | `finalizeLocal` · `collectDeclared` · `copyWithin` · `walker.visit` · 세션 `Finalize` (overlay) · `Worker.upload` |
| finalize · upload · reason · error (규칙 3.2) | `settle` |
| 종료 보고의 조건 · 응답 · 재전송 (규칙 4절) | `exitOutcome` · `exitedStop` · `exitReportLog` · `exitBackoff` · `startExitReport` · `Stop` |
| 명시 훑기의 곳 · 상한 넷 (규칙 5.1 · 5.2) | `walkWorkspace` · `walkUpper` · `defaultDiscoverLimits` · `discoverTime` |
| changes 의 값 · 진단 칸 (규칙 5.3 · 6.1) | `diagnosticsFor` |
| 단계 로그 끝 줄 · 안내 (규칙 6.3 · 7절) | `logTail` · `withTail` · 상수 `noticeRunDefault` · `noticeAgentChanged` |
| 업로드 규칙 (규칙 8절) | `Worker.upload` · `PutBlob` · `BlobRejected` · `uploadNames` |
| 경로마다의 새 칸 (규칙 9절) | `Worker.execute` 뒷부분 · `runAgentStep` 뒷부분 · `closeAndUploadLog` |

---

## 3. 계획 4절의 결정 열하나 — 지은 모양

```text
   ①  파일 자리          finalize.go 새 파일 · 업로드는 upload.go · Client.Exited 는 claim.go 의 Report 옆
   ②  DiscoverFor        FinalizeSpec 한 칸.  Worker 가 discoverTime(finalize) 를 싣는다.  0 이면 30초
   ③  overlay 의 기다림   ctx 가 끝난 뒤 helperGrace(5초).  넘으면 abort · aborted 표시 · ctx 의 오류.
                         helper 가 제 마감에 답하면 모은 것과 DeadlineExceeded 를 돌려준다
   ④  whiteout 판정       walkUpper 가 판정 함수를 받는다.  linux helper 는 isOverlayWhiteout (문자 장치 0:0)
   ⑤  시험 자리           Worker.budgets · Worker.exitWait · helperGrace · statPath · walkDir
   ⑥  instance 없음       startExitReport 가 nil 을 돌려주고 debug 한 줄.  nil 의 Stop 은 할 일이 없다
   ⑦  finalized 로그      effect · checked · discover · visited · limit · took.  조각 1 스크립트가 읽는다
   ⑧  changedSince        그대로다.  훅의 걷기가 안 바뀐다
   ⑨  ctx 를 보는 복사     1 MiB 마다.  멈추면 .part 를 지우고 남은 이름을 안 옮긴다
   ⑩  못 뜬 하네스        exited_at 없음 · Finalize 예산은 runHarness 가 돌아온 때부터
   ⑪  조각 스크립트       scripts/finalize-bake/slice-N.sh.  M · T 를 받는다
```

---

## 4. 계획과 다르게 된 자리

**① `advertise.go` 를 고쳤다.** `Client` 구조체가 거기 있어 `Upload` 칸 하나를 더했다. 계획 2절의 표에 없던
파일이다.

**② `Discovery.Paths` 가 경로와 크기다** (`[]Changed`). FD 엔티티 2.2 는 `[]string` 이었다. 단계 로그 끝의
목록이 크기를 함께 적으므로(FD 규칙 6.3) 크기를 들고 온다. 진단 칸 `discovered` 에는 경로만 옮긴다.

**③ Finalize 의 마감으로 멈춘 걷기도 `time` 으로 적는다.** 비워 두면 진단 칸이 부분 목록을 `measured` 로 적는다.
native 에서 마감이 걷기 도중에 오면 `changes: partial` · `discovery_limit: time` 이다. overlay 에서 helper 를 죽였으면
결과가 없어 `not_measured` 다 (FD 규칙 3.1 그대로).

**④ `UploadLog` 의 4xx 도 `BlobRejected` 다.** 단계 로그도 산출물과 같은 표를 따른다 (FD 규칙 8절). 거절은 upload
칸을 바꾸지 않고, 5xx 와 끊김은 error 다.

**⑤ 명령 단계의 tee 를 Run 이 돌아온 바로 뒤에 멈춘다.** 오늘은 수확 뒤였다. 명령의 출력은 Run 이 돌아온 때 이미
다 왔고, 순서(진행 청크 먼저 · 선별본 나중)는 그대로다.

**⑥ 임대가 끝난 단계와 못 뜬 프로세스도 단계 로그를 올린다** — Worker 의 ctx 에 업로드 예산을 걸어서
(`closeAndUploadLog`). 오늘도 올렸다. 산출물은 안 올린다 — 오늘 임대가 끝난 단계는 올렸다.

**⑦ agent 단계가 임대 만료로 끝나면 오류 문구는 하네스의 것이다** (`harness: …`). 하네스가 완주했는데 임대가
끝났으면 `aborted: lease expired`. 오늘 agent 흐름에는 임대 확인이 따로 없었다.

**⑧ collect 시험 하나를 옮겼다.** `worker_test.go` 의 collect 실패 시험이 이제 진짜 프로세스를 띄우므로
`finalize_worker_test.go` (`!windows`) 로 옮겼다.

---

## 5. 코드 검사 (`unit-of-work.md` 0절 · CI 와 같은 명령)

| 검사 | 결과 |
|---|---|
| `gofmt -l .` | 빈 출력 |
| `go vet ./...` · `go vet -tags integration ./internal/enode/` · `go build ./...` | exit 0 |
| `go test ./... -count=1` (`-coverpkg=./...` · `-json`) | 통과 2,025 · 실패 0 · 스킵 0 (U2 병합 때 1,960) |
| 패키지마다 커버리지 (CI 의 awk 그대로) | 스무 패키지 전부 80% 이상 · 전체 85.9% (앞 85.7%). `internal/enode` **81.5% -> 82.3%** (2,957/3,593) · `cmd/enode` 83.0% |
| 새 함수 | `finalize.go` 의 함수 열다섯 중 열넷 100% · `startExitReport` 96.2% · `afterExit` 100% · `settle` 100% · `finalizeLocal` 94.7% · `walkWorkspace` 90.0% · `walkUpper` 82.6% · overlay 세션 `Finalize` 88.9% · helper `finalize` 90.9% · `isOverlayWhiteout` 28.6% (진짜 whiteout 은 `integration` 태그 시험) |
| 크로스 빌드 셋 | windows/amd64 · linux/arm (GOARM=7) · darwin/arm64 exit 0 |
| `enodectl.exe` 심볼 | crypto/tls 1 · net/http 6 (상한 10 · 50) |
| U+2605 · `glyphscan` | 0 · 「136 files scanned, no decorative glyph」 |
| `golangci-lint` v2.13.2 | 39 건 — Step 1 의 기준선과 목록이 같다. 이 유닛이 더한 경고 0 |
| **조각 0** (기동이 안 깨졌다) | build · vet · test 통과 · `mux.HandleFunc` 19 (step-phase 그대로) |
| `cmd/enodectl/probe.lock` | 측정이 바꿔 되돌렸다 (Reverse Engineering 이 적은 알려진 부채) |

---

## 6. 조각

**조각 1 (걷지 않는다 · 기계)** — 두 겹이다.

- 기본 `go test` — `TestFinalize_ABuildStepDoesNotWalkTheWorkspace`: 파일 500 개 워크스페이스의 build effect 단계에서
  걷기 0 번 · stat 3 번 (지목 경로 셋) · `$OUT/result` 가 올라감 · 「no files changed」와 `workspace.changed` 가 없음 ·
  `changes: not_measured` · 두 시각이 순서대로 있음 · 종료 보고의 `exited_at` 이 result 와 같음. 부분 관찰은
  `TestWalkWorkspace_StopsAtEachLimit` (상한 넷 · 결과 크기 둘). 초록
- 스크립트 — `slice-1.sh` 를 이 브랜치에서 빌드한 스크래치 Mediator (`127.0.0.1:18080`) 로 돌렸다 (2026-09-25).
  **`slice 1: green`**. 빈 파일 3,000,000 개(디렉터리 1,000 x 3,000)의 워크스페이스와 파일 셋의 워크스페이스에서
  봉인된 단계 기록의 `finalized_at - exited_at` 이 둘 다 0.000 초(1 ms 아래) · 두 노드 로그 모두 `checked=3 visited=0` ·
  `$OUT/result` 가 봉인에 있고 `changes: not_measured`. 첫 실행은 계약이 400 이었다 — `changed` 는 워크스페이스를
  적은 단계에만 쓸 수 있다 (ADR-037). 단계에 `"workspace": {"repo": ""}` 를 더해 노드가 되돌리지 않은 채로 돌게 했다
  (ADR-036 · git 을 안 부르고 트리를 안 걷는다)

**조각 3 (예산 · 기계)** — 기본 `go test`. `TestFinalize_OverTheFinalizeBudget` (예산 50 ms · `finalize: timeout` ·
`reason: finalize_timeout` · 문구 `finalize budget of 50ms exceeded` · `exit_code` 1 이 남음 · 업로드는 계속) ·
`TestFinalize_OverTheUploadBudget` (예산 300 ms · 둘째 이름에서 멈춤 · `upload_timeout` · produced 는 첫 이름뿐) ·
`TestFinalize_TheContractBudgetSetsTheDeadline` (계약의 `"finalize": "5m"` 이 마감 = exited_at + 5분 · 훑기 150초) ·
`TestSettle` (경로마다 한 줄). 초록

**조각 2 (보인다 · 사람)** — 스크립트만 준비했다. 사람이 돈다.

```bash
export M=http://<스크래치 Mediator>:8080 T=<bootstrap 토큰>
export MEDIATOR_PID=<그 Mediator 의 pid>          # ② 에서 10초 멈춘다
export OLD_ENODE=<main 0c0370c 에서 빌드한 enode>    # ④ 이 유닛 전의 노드
scripts/finalize-bake/slice-2.sh
```

**2026-09-26 에이전트가 돌렸다 (사용자 지시 「해봐」). 판정은 사람의 몫이다.** 스크래치 Mediator (`127.0.0.1:18080`) ·
옛 노드는 `main` `0c0370c` 을 `git archive` 로 풀어 빌드했다.

```text
   ①  12:30:50.37  running (exit 없음)
      12:30:58.76  finalizing · exit {kind: exit, code: 1} · phase_since 12:30:58.51 (= 노드의 exited_at)
      12:30:59.30  단계 종결 (phase 없음 · exit 는 남는다) -> Run FAILED (success_when 이 exit 0 을 요구)
      업로드 8 MiB x 32 = 256 MiB 가 약 0.3 초.  produced 32
   ②  Mediator 를 12:31:08 ~ 12:31:18 멈췄다.  재개 직후 finalizing · exit 1 · phase_since 12:31:10.46.
      단계 기록 둘 — state DONE · exit {exit, 1} · last_phase finalizing · finalize ok · upload ok · produced 32 ·
      exited_at 과 finalized_at 이 둘 다 있다.  시각만 다르다.  종료 보고는 유실이 아니라 멈춘 동안 늦게 닿았다
   ③  다른 instance 의 종료 보고 -> 409 「step … is claimed by another instance of this node; a restarted node
      cannot report the exit of an earlier life」
   ④  옛 노드 — 끝날 때까지 running · last_phase running · exited_at 없음 · finalize · upload 칸 없음
   ⑤  exited_at 03:30:58.509853 < finalized_at 03:30:58.509915 (Finalize 가 한 일이 없어 62 µs)
```

첫 실행에서 ① 이 finalizing 을 못 보였다 — 256 MiB 파일 하나가 Mediator 의 blob 상한(`max_blob_bytes` 기본
10 MiB)에 413 으로 곧바로 끊겨 업로드가 짧았고, 1초 간격의 조회가 그 구간을 놓쳤다. 노드는 그것을 거절로 받아
`upload: ok` · produced 없음으로 적었다 (FD 규칙 8절 그대로). 스크립트를 8 MiB 파일 여럿과 0.2초 간격(바뀔 때만
찍는다)으로 고쳐 다시 돌렸다.

Mediator 는 step-phase 이후(`main` `0c0370c` 이상)여야 한다. 이 기계의 개발용 Mediator(`:8080`)는 2026-09-17
빌드라 종료 보고에 404 를 준다 — 조각 1 은 이 브랜치에서 빌드한 스크래치 Mediator(`127.0.0.1:18080`, DB
`enode_slice`)로 돌렸다. 스크립트가 보이는 것 — ① 명령이 도는 동안 running, 끝나면 finalizing 과 exit 1 ·
③ 다른 instance 의 종료 보고가 409 · ② Mediator 를 멈춘 Run 과 안 멈춘 Run 의 단계 기록 · ④ 옛 노드의 running ·
⑤ 두 시각.

---

## 7. 측정 — NFR 을 건너뛰어 계획 3절이 받은 것

| 값 | 측정 (2026-09-25 · 이 기계 · 캐시가 데워진 디스크) | 판단 |
|---|---|---|
| 30초에 훑는 항목 수 | `BenchmarkWalkWorkspace` — 초당 약 304,000 방문 (항목 100,100 을 5 번). 조각 1 스크립트의 discover Run (파일 3,000,000 개) — 방문 2,000,000 에서 멈춤 · 27.1 초 · `changes: partial` · `discovery_limit: visits` | 데워진 트리 (벤치마크) 에서는 방문 상한이 약 6.6 초에 닿는다. 300만 파일 트리를 처음 걸을 때는 27.1 초 — 시간 상한 30 초보다 3 초 먼저다. 더 느린 디스크에서는 시간 상한이 먼저 닿는다. 어느 쪽이든 `partial` 로 적히고 Finalize 예산(1분)을 안 넘는다. 값은 그대로 둔다 |
| helper 여유 5초 | `TestFinalize_ReturnsSoonAfterItsDeadline` — 마감 뒤 돌아오기까지 훑는 중 0.19 ms · 256 MiB collect 중 0.89 ms | 5초가 세 자리 이상 크다. 값은 그대로 둔다 |

SunnyVM (노트북 VM · 꺼져 있을 수 있다) 의 디스크에서는 측정하지 않았다. 사람 조각 4 (trash) 를 돌 때 함께 보려면
노드 설정의 워크스페이스에서 이 한 줄을 돈다.

```bash
go test ./internal/enode/ -run '^$' -bench WalkWorkspace -benchtime 5x
```

overlay 에서의 helper 여유는 `integration` 태그 시험에 한 줄(`helper answered … after its deadline`)을 더해 두었다.
CI 밖이고 사람이 SunnyVM 에서 돈다.

---

## 8. 뒤 유닛에 넘기는 것 (FD 흐름 11절 그대로)

```text
   trash        닫기를 rename 으로 바꾸는 커밋에서 afterExit 의 session.Close(ctx, Keep{}) 에 Finalize ctx 를
                넘긴다 (지금은 Worker 의 ctx).  Keep.Upper 가 "" 면 runRoot 를 trash 로.  finalized_at 의 뜻은 그대로.
                runcOverlaySession.Close 의 os.RemoveAll(s.runRoot) 두 자리 (보통 · abort 뒤)가 바뀔 자리다
   checkpoint   받아들임과 spool 은 닫기 안이고 Finalize 예산의 남은 몫을 쓴다.  Keep.Upper 에 spool 자리.
                result 의 checkpoint_capture
   bake         build 단계의 종료 보고 — Client.Exited 와 startExitReport 를 쓴다.  merge 단계는 안 보낸다.
                build 단계의 두 예산 (contractStep 이 build 종류를 아직 모른다 — Kind 가 "agent" 가 아니면
                Run 으로 채운다.  bake 가 build · merge 를 더할 때 여기를 넓힌다)
```

---

## 9. 정본에 되돌려 올릴 것 (FD 흐름 12절 · 진행자가 올린다)

```text
   protocol/run-contract.md   discover 의 상한 넷과 그 값은 노드가 정한다 (방문 2,000,000 · 들고 있는 항목 200,000 ·
                              경로 2,000 · 글자 합 256 KiB · 시간은 Finalize 예산의 절반) ·
                              discover 는 노드에 워크스페이스가 있어야 목록이 나온다 · workspace.changed 가 더는 안 나온다 ·
                              workspace.diff 는 diff 가 있는 단계에서만 · 단계 로그 끝의 안내 줄
   ADR-075 §9                 HarvestSpec 의 칸마다 판정 · Harvest 가 Finalize 가 됐다 · Discovery.Paths 는 경로와 크기
   ADR-075 §10.2 ⑤           Finalize 예산이 닫기를 덮는 것은 닫기가 rename 이 된 뒤다 (trash 전의 창)
   ADR-075 §10.4              노드의 재전송 — 2xx · 4xx 에서 멈춤 · 5xx 와 끊김에서 1 · 2 · 4 · 8 · 10 초 ·
                              result 를 보내기 직전에 그만 · instance 가 없으면 안 보냄
   mediator-api.md exited 절   옛 Mediator 는 라우트가 없어 404 를 준다
```

---

## 10. 확장 준수 — Code Generation

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표에서 이 유닛에 닿는 줄은 없다 — 노드는 종료 보고에 자기 instance 를 싣기만 하고 대조는 step-phase 가 Mediator 에서 한다. 조각 2 의 ③ 이 사람 눈으로 다시 확인한다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 한 줄 |
