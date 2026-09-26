# `finalize` — Code Generation 계획

**유닛** `finalize` (결과 확정 · 한 줄 순서의 셋째) · **브랜치** `unit/finalize` ·
**기준** `572ec87` (Functional Design 커밋) · **맡는 조각** 1 (걷지 않는다 · 기계) · 2 (보인다 · 사람) ·
3 (예산 · 기계) · **병합 조건** 조각 1 · 2 · 3 이 초록

**이 계획이 이 유닛 Code Generation 의 기준이다.** 여기 없는 것은 짓지 않는다. 설계는
`construction/finalize/functional-design/` 의 셋이다 — 타입은 `domain-entities.md`, 규칙과 문구는
`business-rules.md`, 순서와 흐름은 `business-logic-model.md`. 아래의 **FD** 는 이 세 Functional Design
산출물의 줄임이다 — 「FD 규칙 3.2」는 `business-rules.md` 3.2절, 「FD 흐름 1절」은 `business-logic-model.md`
1절, 「FD 엔티티 2.1」은 `domain-entities.md` 2.1절을 뜻한다.

- **작성 시각**: 2026-09-25T12:53:36Z 이후 (Functional Design 승인 뒤)
- **입력**: FD 셋 · 유닛 정의 `unit-of-work.md` 3절 · 파일 행렬 `unit-of-work-file-matrix.md` 1.3절 ·
  스토리 지도 `unit-of-work-story-map.md` · 팩의 `scene-gates.md` 2절 조각 1 · 2 · 3 · 요구 문서
  `requirements.md` 5.4절 (임대 창이 트리 크기와 무관하다) · 6절 (조각 1 과 2 에 더한 확인)
- **NFR 두 단계** (NFR Requirements · NFR Design — 비기능 요구와 그 설계): 사용자 결정으로 건너뛰었다
  (2026-09-25T12:53:36Z). 그 둘이 맡기로 했던 것을 이 계획의 3절이 받는다

---

## 1. 유닛 맥락

**하는 일** — 노드가 명령이 끝난 뒤의 구간(결과 확정 · 닫기 · 업로드 · 보고)을 두 예산 안에서 닫는다.
Mediator 는 한 줄도 안 바뀐다 — 받는 쪽은 앞 유닛 step-phase 가 이미 지었다.

```text
   세션 겉면        Harvest 를 Finalize 로.  Close 가 행선지(Keep)를 받는다 (ADR-075 §9 — 세션 겉면 교체)
   수확             effect(작업 성격) 에 따라.  run 단계의 기본 build 는 diff 를 안 낸다.
                   전체 훑기는 계약이 discover 를 켠 단계에서만, 상한 넷 안에서
   진단             workspace.changed 를 없앤다.  result 의 diagnostics 칸과 단계 로그 끝 줄로
   안내             바뀐 기본값을 단계 로그 끝 한 줄로 알린다
   종료 보고        명령이 끝나면 POST .../exited 를 goroutine 으로 보낸다.  result 직전에 그만둔다
   예산 둘          Finalize 1분 · 업로드 3분 (기본).  넘으면 finalize_timeout · upload_timeout
   업로드 client    요청마다의 30초 제한이 없는 client.  파일을 통째로 읽지 않고 흘려 보낸다
   result 새 칸     exited_at · finalized_at · finalize · upload · reason · diagnostics
```

**완성하는 스토리**

```text
   US-8    finalizing 에 오래 있을 때 언제까지 기다리면 되는지 — phase_since 에 두 예산을 더한 시각이 상한
   US-9    effect 를 안 적은 명령 단계가 workspace.diff 와 변경 목록을 안 낸다는 것을 안다
   US-10   agent 단계가 workspace.changed 를 안 낸다는 것을 안다
```

**완료 조건** — 3 ④ (명령 단계가 diff 를 안 낸다) · 3 ⑤ (agent 단계의 전체 훑기가 꺼진다).

**앞 유닛에서 받는 것**

```text
   contract-grammar (계약 문법 · PR #61)   contract.Effect · Budget · Step.EffectOrDefault · Step.Budgets ·
                                         Step.Discover · DefaultFinalizeBudget · DefaultUploadBudget
   step-phase (Mediator 진행 구간 · PR #62)  contract.Exited · Outcome · Stage · Diagnostics · CollectNote ·
                                         Limit* · Changes* · Reason* 상수 ·
                                         Claimed 가 effect · budget · discover 를 싣는다 ·
                                         POST .../exited 의 수락 표 · StepResult 의 새 칸
```

**뒤 유닛에 넘기는 것** — FD 흐름 11절 그대로다 (trash 는 닫기를 Finalize 예산 안으로 · checkpoint 는
`Keep.Upper` 와 spool · bake 는 build 단계의 종료 보고와 두 예산).

**새 이름** (모두 `internal/enode` 안. 패키지 밖으로 새로 나가는 것은 `Client.Exited` · `Client.Upload` ·
`BlobRejected` · 세션 타입들)

```text
   runtime.go     FinalizeSpec · FinalizeResult · Discovery · Keep · StepSession.Finalize · Close(ctx, Keep)
   changed.go     discoverMaxVisits · discoverMaxHeld · discoverMaxPaths · discoverMaxBytes ·
                  discoverTime · discoverLimits · walkWorkspace · walkUpper
   finalize.go    contractStep · finalizeSpecFor · stepBudgets · exitOutcome · exitedStop · exitBackoff ·
   (새 파일)       settle · diagnosticsFor · logTail · exitReporter · startExitReport
   claim.go       Step.Effect · Budget · Discover · Result 새 칸 여섯 · Client.Exited · Client.Upload
   upload.go      BlobRejected · PutBlob(io.Reader, size) · uploadProduced 의 새 모양
   runner.go      Job.Exited
```

**없애는 이름** — `HarvestSpec` · `HarvestResult` · `StepSession.Harvest` · `changedName` ·
`writeHarvestNote` · `writeChangedNote` · `logHarvest` (FD 규칙 6.2). `HarvestNote` 는 남는다 — collect 가
쓰고 진단 칸의 `collect` 로 옮겨 담는다. `changedSince` 와 `summarize` 는 훅이 쓰므로 그대로다.

**경계** — 새 코드는 표준 라이브러리 · `internal/contract` · 오늘 `internal/enode` 가 이미 쓰는 것만 쓴다.
코드 경계 시험(`boundary_test`)의 금지 표가 바뀌지 않는다. `go.mod` 를 안 움직인다.

---

## 2. 고치는 파일

| 파일 | 새 · 고침 | 행렬 | 무엇 |
|---|---|---|---|
| `internal/enode/runtime.go` | 고침 | 있음 | 세션 겉면 · `nativeSession.Finalize` · `onceSession.Close(ctx, Keep)` |
| `internal/enode/changed.go` | 고침 | 있음 | 상한 넷 · `walkWorkspace` · `walkUpper` · `changedName` 을 지움 |
| `internal/enode/claim.go` | 고침 | 있음 | `Step` 칸 셋 · `Result` 칸 여섯 · `Client.Exited` · `Client.Upload` · 명령 단계와 agent 단계의 흐름 · 옛 진단 함수 셋을 지움 |
| `internal/enode/upload.go` | 고침 | 있음 | `PutBlob` · `UploadLog` 를 옮겨 온다 · 흘려 보내기 · `BlobRejected` · `uploadProduced` (4절 ①) |
| `internal/enode/runc_overlay_linux.go` | 고침 | 있음 | helper 요청 `finalize` · 마감 · upper 훑기 · 마감 뒤 5초의 abort · abort 뒤의 `Close` |
| `internal/enode/runc_overlay_other.go` | 안 고친다 | 있음 | 세션 타입이 없다 — `Open` 이 늘 실패한다. 확인만 한다 (FD 흐름 10절) |
| `cmd/enode/main.go` | 고침 | 있음 | `Upload: &http.Client{}` 한 줄과 주석 |
| `internal/enode/finalize.go` | 새 | 밖 (4절 ①) | 규칙의 순수 함수 · 종료 보고 goroutine |
| `internal/enode/runner.go` | 고침 | 밖 (FD 흐름 10절) | `Job.Exited` 콜백 한 칸과 부르는 한 줄 |
| `internal/enode/collect.go` | 고침 | 밖 (FD 흐름 10절) | `collectDeclared` 와 `copyFile` 이 ctx 를 받는다 |
| `scripts/finalize-bake/slice-1.sh` | 새 | 있음 (조각 스크립트) | 조각 1 — 300만 파일 워크스페이스와 빈 워크스페이스의 비교 |
| `scripts/finalize-bake/slice-2.sh` | 새 | 있음 (조각 스크립트) | 조각 2 — 사람이 보는 순서를 명령으로 굳힌다 |
| `internal/enode/finalize_test.go` | 새 | 시험 | 순수 함수의 표 시험 · 종료 보고 goroutine |
| `internal/enode/finalize_worker_test.go` | 새 | 시험 | Worker 흐름 — 조각 1 의 기계 부분 · 조각 3 · 경로마다의 result 칸 · 옛 Mediator |
| `internal/enode/changed_test.go` | 고침 | 시험 | 상한 넷 · upper 훑기 · `writeChangedNote` 시험을 지움 |
| `internal/enode/collect_test.go` | 고침 | 시험 | ctx 로 멈춘 복사 · `writeChangedNote` 시험을 지움 |
| `internal/enode/upload_test.go` | 고침 | 시험 | 흘려 보내기 · 거절과 전송 실패 · 업로드 예산 |
| `internal/enode/worker_test.go` | 고침 | 시험 | `uploadProduced` 의 새 모양 · workspace.changed 를 찾던 시험을 반대로 |
| `internal/enode/runtime_test.go` · `runtime_projection_test.go` | 고침 | 시험 | 가짜 세션이 새 겉면을 따른다 |
| `internal/enode/runc_overlay_linux_test.go` | 고침 | 시험 | 가짜 helper 의 op 이름 · 늦은 helper 의 abort · abort 뒤 Close |
| `internal/enode/runc_overlay_integration_test.go` | 고침 | 시험 (`integration` 태그) | 새 겉면으로 컴파일되게. CI 밖 |

**행렬 밖 파일 넷** — 셋(`runner.go` · `collect.go` · 시험 파일)은 FD 흐름 10절이 이미 적었다. 이 계획이
더하는 것은 `finalize.go` 하나다 (4절 ①).

`internal/enode/diff.go` 와 `hook.go` 는 안 고친다 (FD 흐름 10절). `internal/store` · `internal/api` ·
`internal/contract` 는 안 고친다.

---

## 3. NFR 두 단계를 건너뛰어서 이 계획이 받는 것

유닛 정의(`unit-of-work.md` 3절)가 NFR 에 적은 것은 둘이었다 — 임대 창이 트리 크기와 무관한지(조각 1) ·
N3 (업로드 예산과 요청의 관계 · 흘려 보내기의 상한). FD 가 둘 다 닫았다.

```text
   조각 1 의 임대 창     FD 답 9 = A.  go test 의 방문 0 과 300만 파일 스크립트 (Step 12 · 14 · 15)
   N3                  FD 답 8 = A.  요청마다 제한 없는 client · 마감은 업로드 예산 · Mediator 가 상한+1
                       바이트에서 끊으므로 보내는 양이 상한을 안 넘는다 (Step 8)
```

FD 엔티티 3절이 「값의 크기는 이 유닛의 NFR Requirements 가 확인한다」로 남긴 값 둘을 여기서 측정한다.
둘 다 틀려도 안전한 쪽으로 틀린다 — 훑기가 일찍 멈춰 목록이 부분이 되거나(`partial` 로 적힌다), helper 를
조금 일찍 죽인다. 그래서 측정 한 번으로 값을 확인하고 숫자를 `code-summary.md` 에 적는다 (Step 13).

**① 30초에 훑는 항목 수** — 방문 상한 2,000,000 과 시간 상한 30초 중 어느 것이 먼저 닿는지.

- `BenchmarkWalkWorkspace` (`changed_test.go`) — 임시 디렉터리에 파일 100,000 개를 만들고 기준 시각 뒤의
  것이 없게 한 채 `walkWorkspace` 를 돌려 초당 방문 수를 낸다. 기본 `go test` 에서는 안 돈다
  (`-bench` 를 줄 때만 돈다)
- 조각 1 스크립트의 세 번째 Run — 300만 파일 워크스페이스에 `"discover": true` 로 같은 no-op 을 낸다.
  노드 로그의 `finalized` 줄(4절 ⑦)이 `visited` 와 걸린 시간을, 봉인된 Record 가 `discovery_limit` 을 준다.
  이 기계의 디스크에서 방문 상한과 시간 상한 중 무엇이 먼저 닿았는지가 곧 답이다
- SunnyVM (노트북 VM · 꺼져 있을 수 있다) 에서의 값은 이 유닛이 측정하지 않는다. 사람 조각 4 (trash) 를 돌 때
  함께 볼 수 있도록 `code-summary.md` 에 명령 한 줄을 남긴다

**② helper 여유 5초** — 마감이 지난 뒤 helper 가 스스로 돌아와 답을 쓰기까지의 시간.

- helper 는 native 와 같은 Finalize 를 돈다 (FD 흐름 3절). 그래서 native Finalize 로 측정한다 —
  큰 트리를 훑는 중과 큰 파일을 collect 하는 중에 마감을 지나게 하고, 마감부터 `Finalize` 가 돌아오기까지를
  측정하는 시험 하나 (`finalize_worker_test.go` · 기본 `go test` 에서 돈다 · 상한 1초로 확인한다)
- 걸린 값의 최대를 `code-summary.md` 에 적는다. 5초가 그 값보다 한 자리 이상 크면 값을 그대로 둔다
- overlay 에서의 실제 값은 `integration` 태그 시험에 한 줄을 더해 두고 사람이 SunnyVM 에서 돈다 (CI 밖)

---

## 4. 이 계획이 정한 것 — FD 에 적히지 않은 자리

**① 파일 자리.** FD 흐름 9절이 규칙을 순수 함수로 떼라고 했고 자리는 안 정했다. `claim.go` 가 이미
1,030 줄이라 새 파일 `finalize.go` 에 둔다 — 순수 함수 여덟과 종료 보고 goroutine. 업로드는 행렬이 적은
`upload.go` 로 모은다 — `PutBlob` 과 `uploadProduced` 는 어차피 다시 쓰므로 옮기는 값이 거의 없고,
`UploadLog` 도 같은 client 를 쓰게 되어 함께 옮긴다. `Client.Exited` 는 `Client.Report` 옆(`claim.go`)에 둔다
— 둘 다 단계 하나에 대한 Mediator 호출이다.

**② `FinalizeSpec.DiscoverFor`.** FD 규칙 5.2 는 훑기의 시간 상한을 「Finalize 예산의 절반 · 훑기를 시작한
때부터」로 정했다. overlay helper 는 요청의 `Deadline` 만 받으므로 예산 값을 모른다. Worker 가
`discoverTime(finalize)` 를 셈해 칸 하나로 싣는다.

```go
	// DiscoverFor 는 명시 훑기의 시간 상한이다 — Finalize 예산의 절반 (business-rules.md 5.2).
	// helper 는 Deadline 만으로는 예산을 모르므로 Worker 가 셈해 싣는다.
	DiscoverFor time.Duration
```

**③ overlay 에서 기다리는 시각.** FD 흐름 3절은 「마감 + 5초」다. Finalize ctx 는 임대가 끝나도 끝나므로
(FD 규칙 2절) 기준을 **ctx 가 끝난 때**(마감이든 임대 끝이든)로 잡고 그로부터 5초를 기다린다. helper 는 요청을
하나씩 처리해 도중에 cancel 요청을 못 읽는다 — 그래서 임대가 끝났을 때도 길은 abort 하나다.

- 값은 패키지 변수 `helperGrace = 5 * time.Second` — 시험이 줄인다
- abort 한 세션에 표시를 남기고, 그 뒤의 `Close` 는 helper 에 말하지 않고 남은 runRoot 만 지운다. 표시가
  없으면 `Close` 가 죽은 helper 에 close 를 보내다 실패해 `runtime cleanup:` 오류가 덧붙는다
- 돌려주는 오류는 `context.DeadlineExceeded` 다 (임대 끝이면 `context.Canceled`). `settle` 이 둘을
  runCtx 의 상태로 나눈다 (FD 규칙 3.2)

**④ whiteout 판정.** overlay 의 지운 항목(whiteout)은 upper 에 문자 장치 0:0 으로 남는다. 보통 사용자 권한의
시험은 그 장치를 만들 수 없다. 그래서 `walkUpper` 가 판정 함수를 인자로 받는다 — linux helper 는 문자 장치
0:0 판정을 넘기고(`runc_overlay_linux.go`), 시험은 이름으로 판정하는 가짜를 넘긴다. 실제 판정은
`integration` 태그 시험이 확인한다.

**⑤ 시험이 바꿔 끼우는 자리.** 모두 내보내지 않는 이름이다.

```text
   Worker.budgets     func(*Step) (finalize, upload time.Duration).  nil 이면 stepBudgets.
                      조각 3 이 1분 아래의 예산을 쓰려면 필요하다 (FD 흐름 8절 — 계약은 1분 아래로 못 내린다)
   Worker.exitWait    func(n int) time.Duration.  nil 이면 exitBackoff.  재전송 시험이 1초를 안 기다리게
   helperGrace        패키지 변수 (③)
   statPath · walkDir 패키지 변수.  os.Stat · filepath.WalkDir 를 가리킨다.  조각 1 의 시험이 stat 수와
                      걷기 호출 수를 센다 (FD 흐름 8절 「세는 자리」).  이 패키지의 시험은 t.Parallel 을
                      안 쓰므로(0 건) 바꿔 끼워도 안전하다
```

**⑥ `Client.Instance` 가 비면 종료 보고를 안 보낸다.** Mediator 가 400 (`instance is empty`) 으로 돌려줄
것을 알고 보내지 않는다. `cmd/enode/main.go` 는 늘 채운다 — 비는 것은 시험과 옛 조립뿐이다. 노드 로그에
debug 한 줄.

**⑦ 노드 로그 `finalized` 한 줄.** Finalize 가 끝나면 info 로 한 줄 — `effect` · `checked` (지목 경로
stat 수) · `discover` (켰나) · `visited` · `limit` · `took`. 조각 1 스크립트가 `checked` 와 `took` 을 읽고,
3절 ① 이 `visited` 를 읽는다. 오늘의 `logHarvest` (diff 크기 · collect 결과) 가 하던 말도 이 자리와 단계 로그
끝 줄이 받는다.

**⑧ `changedSince` 를 그대로 둔다.** 새 걷기 둘은 `changed.go` 에 따로 짓는다. `changedSince` 를 새 걷기
위에 다시 세우면 훅의 걷기에 방문 상한이 생기고 들고 있는 항목이 차면 걷기가 멈춘다 — 오늘은 세기를
계속한다. FD 규칙 6.2 가 훅의 걷기를 그대로 두라고 했으므로 겹치는 스무 줄 남짓을 받아들인다.

**⑨ ctx 를 보는 복사.** `copyFile` 이 `io.Copy` 대신 1 MiB 씩 읽고 쓰며 사이마다 `ctx.Err()` 를 본다. 멈추면
`.part` 를 지우고 ctx 의 오류를 돌려준다. collect 는 그 이름을 `HarvestNote` 로 남기지 않고 멈춘다 —
나머지 이름도 안 옮긴다 (마감이 지났다).

**⑩ 하네스가 뜨지 못한 agent 단계.** FD 흐름 2절대로 `exited_at` 이 없다. Finalize ctx 의 마감은
`runHarness` 가 돌아온 시각 + Finalize 예산이다. `Result.ExitedAt` 은 비고 나머지 새 칸 다섯은 채운다.

**⑪ 조각 스크립트의 자리.** `scripts/finalize-bake/slice-N.sh`. 뒤 유닛(bake · checkpoint) 이 같은 폴더에
자기 조각을 더한다. 스크립트는 환경 변수 `M` (Mediator 주소) · `T` (bootstrap 토큰) 을 받는다 — 팩의
`scene-gates.md` 3절 머리말과 같다. 사내 이름과 경로를 적지 않는다.

---

## 5. 단계 — 열일곱

### Step 1 — 기준선

- [x] `unit/finalize` 가 `572ec87` 위에 있고 작업 트리가 깨끗한지 (계획 · 상태 · 감사만 커밋 전)
- [x] `eval "$(scripts/testdb.sh)"` 뒤 `go test ./internal/enode/ -count=1 -coverprofile=...` — 오늘의
      `internal/enode` 커버리지를 적는다 (Reverse Engineering 은 81.5% 로 적었다).
      **측정 2026-09-25** — CI 의 awk (`-coverpkg=./...`) 로 81.5% (2,653/3,256) · 전체 85.7% ·
      자기 패키지 시험만 세면 80.2%
- [x] `golangci-lint run` 의 경고 수를 적는다 (main 과 같아야 한다 — step-phase 에서 39 건). **39 건** (v2.13.2)

### Step 2 — 노드 `Step` 의 칸 셋과 `Result` 의 칸 여섯 (FD 엔티티 1절 · 4절)

- [x] `claim.go` 의 `Step` 에 `Effect contract.Effect` · `Budget *contract.Budget` · `Discover bool`.
      JSON 이름은 `store.Claimed` 와 같다. 한 덩어리 주석 — 옛 Mediator 는 안 싣고 그때 기본값이 채워진다
- [x] `Result` 에 `ExitedAt *time.Time` · `FinalizedAt *time.Time` · `Finalize contract.Stage` ·
      `Upload contract.Stage` · `Reason string` · `Diagnostics *contract.Diagnostics`. JSON 이름과 `omitempty`
      는 `store.StepResult` 와 같다. 주석 — Finalize 를 돈 단계에만 있다 · 판정 재료가 아니다
- [x] `finalize.go` — `contractStep(step)` (종류를 정하는 칸과 세 칸만 채운 `contract.Step`) ·
      `stepBudgets(step)` (`contract.Step.Budgets` 를 부른다) · `finalizeSpecFor(step, completed, local)`
      (FD 규칙 1절의 표 · 오늘 조건 = `Local.Workspace` 가 있고 계약이 `workspace` 를 적었다)
- [x] `finalize_test.go` — effect 표 다섯 줄 · agent 가 완주 못 한 줄 · 예산의 기본값과 계약이 늘린 값
      (`"budget": {"finalize": "5m", "upload": "10m"}`) · 옛 Mediator 처럼 세 칸이 없는 Step

### Step 3 — 세션 겉면 (FD 엔티티 2절)

- [x] `runtime.go` — `HarvestSpec` · `HarvestResult` 를 `FinalizeSpec` · `FinalizeResult` · `Discovery` 로
      바꾼다. 칸마다의 판정은 FD 엔티티 2.1 의 표 그대로. `DiscoverFor` 한 칸을 더한다 (4절 ②).
      `Keep` · `StepSession.Finalize` · `Close(context.Context, Keep) error`
- [x] `nativeSession.Finalize` — 순서는 collect · 지목 경로 stat · diff (`Diff` 일 때) · 명시 훑기
      (`Discover` 일 때). ctx 를 각 자리에 넘긴다. 마감이 지나면 그 자리에서 멈추고 `ctx.Err()` 를 돌려준다 —
      그때까지 모은 값은 결과에 남긴다. 훑기의 상한은 `discoverLimits` 기본값과 `DiscoverFor`
- [x] 명시 훑기의 곳 — native 는 `Stamp.Root` 를 `walkWorkspace` 로, 워크스페이스가 비었으면 `Skipped` 에
      이유 (FD 규칙 5.1)
- [x] `onceSession.Close(ctx, Keep)` — 오늘처럼 한 번만. `nativeSession.Close` 는 할 일이 없다
- [x] 가짜 세션 둘(`runtime_test.go` 의 `trackingSession` · `runtime_projection_test.go` 의
      `projectingSession`) 을 새 겉면으로. 기존 시험의 뜻은 그대로

      **한 일과 다른 점** — `Discovery.Paths` 는 `[]string` 이 아니라 `[]Changed` (경로와 크기) 다. 단계 로그 끝의
      목록이 크기를 함께 적기 때문이다 (FD 규칙 6.3). 진단 칸에는 경로만 옮긴다

### Step 4 — 명시 훑기와 상한 넷 (FD 엔티티 3절 · FD 규칙 5절)

- [x] `changed.go` — 상수 넷과 `discoverTime`. `discoverLimits` 구조체(방문 · 들고 있는 항목 · 경로 수 ·
      글자 합 · 시간) — 걷는 함수가 인자로 받아 시험이 작게 준다
- [x] `walkWorkspace(ctx, stamp, limits)` — 기준 시각 뒤의 보통 파일. `.git` · `.repo` 아래로 안 들어간다.
      방문마다 `Visited` 를 올리고, 먼저 닿은 상한 하나를 `Limit` 에 적고 멈춘다. ctx 가 끝나도 멈춘다
      (그때는 `Limit` 이 비고 `settle` 이 finalize 의 마감으로 다룬다). 걷기가 끝나면 큰 파일부터 담다가 경로 수나
      글자 합에 닿으면 자르고 `size` 를 적는다 — 다른 상한이 먼저 닿았으면 `size` 를 안 적는다
- [x] `walkUpper(ctx, upper, limits, whiteout)` — upper 의 보통 파일을 모은다 (기준 시각 없음 · upper 에 있는
      것이 곧 이 단계가 쓴 것). whiteout 은 목록에 안 넣고 `Deleted` 를 센다. 디렉터리와 opaque 표시는 안 센다
      (4절 ④)
- [x] `changedName` 을 지운다. `changedSince` · `summarize` · `human` 은 그대로 (4절 ⑧)
- [x] `changed_test.go` — 상한마다 한 시험(방문 · 들고 있는 항목 · 경로 수 · 글자 합 · 시간 — 시간은
      1 ns 상한) · 먼저 닿은 하나만 적힌다 · `.git` · `.repo` · 순서 · upper 의 whiteout 수 · ctx 로 멈춤.
      `writeChangedNote` 를 부르던 시험 셋을 지운다

      **한 일과 다른 점** — ctx (Finalize 예산의 마감) 로 멈춘 걷기도 `time` 으로 적는다. 비워 두면 진단 칸이
      부분 목록을 `measured` 로 적게 된다

### Step 5 — collect 가 ctx 를 받는다 (FD 흐름 10절 · 4절 ⑨)

- [x] `collect.go` — `collectDeclared(ctx, ws, out, spec)` · `copyFile(ctx, src, dst)`. 이름 사이와 복사 중
      (1 MiB 마다) `ctx.Err()` 를 본다
- [x] `collect_test.go` — 끝난 ctx 로 부르면 그 이름이 `$OUT` 에 없고 `.part` 도 없다 · 이름 사이에서 멈추면
      남은 이름을 안 옮긴다. `writeChangedNote` 를 부르던 시험 하나를 지운다

### Step 6 — overlay helper 와 세션 (FD 엔티티 7절 · FD 흐름 3절)

- [x] 요청과 응답의 칸 — `Harvest` 를 `Finalize` 로, op `harvest` · `harvested` 를 `finalize` · `finalized` 로
- [x] helper — `openSession` 이 upper 경로를 기억한다. `finalize` 요청을 받으면
      `ctx := context.WithDeadline(context.Background(), spec.Deadline)` · `waitRun` · merged 에서 collect · stat ·
      diff, upper 에서 `walkUpper` (문자 장치 0:0 판정). `context.Background()` 로 돌던 자리(`:980`)가 없어진다
- [x] 세션 `Finalize` — 요청을 보내고 응답을 goroutine 에서 받는다. ctx 가 끝나면 `helperGrace` 를 더
      기다리고 그래도 답이 없으면 abort 하고 ctx 의 오류를 돌려준다 (4절 ③). abort 표시를 남긴다
- [x] 세션 `Close(ctx, Keep)` — abort 표시가 있으면 runRoot 지우기만. 없으면 오늘 그대로. `Keep` 은 받기만
- [x] `runc_overlay_other.go` — 바꿀 것이 없는지 확인한다 (세션 타입이 없다)
- [x] `runc_overlay_linux_test.go` — 가짜 helper 의 op 이름 · 답하지 않는 가짜 helper 에 짧은 마감 → abort 와
      `DeadlineExceeded` · abort 뒤 `Close` 가 오류 없이 runRoot 를 지운다 · 마감 안에 답한 helper 의 결과가
      그대로 온다
- [x] `runc_overlay_integration_test.go` — 새 겉면으로 고친다. 3절 ② 의 overlay 측정 한 줄과 whiteout 판정
      한 줄을 더한다. `go vet -tags integration ./internal/enode/` 로 컴파일만 확인한다 (CI 밖)

### Step 7 — 종료 보고 (FD 엔티티 5절 · FD 규칙 4절 · FD 흐름 4절)

- [x] `claim.go` — `Client.Exited(ctx, runID, seq, e) (stop bool, err error)`. 30초 client. 응답을
      `exitedStop(status)` 로 나눈다
- [x] `finalize.go` — `exitOutcome(code, runErr)` (FD 규칙 4.1 의 세 줄) · `exitedStop(status)` (FD 규칙 4.2
      의 표) · `exitBackoff(n)` (1 · 2 · 4 · 8 · 10 · 10 초) · `exitReporter` · `Worker.startExitReport` ·
      `Stop`. 응답마다의 노드 로그 문구는 FD 규칙 4.2 그대로
- [x] `Instance` 가 비면 안 보낸다 (4절 ⑥)
- [x] `runner.go` — `Job.Exited func(code int, runErr error, at time.Time)`. `session.Run` 이 돌아온 바로
      다음 줄에서 부른다 (봉투 해석 앞)
- [x] `finalize_test.go` — `exitOutcome` 표 (exit 0 · exit 1 · native 의 signal · 안 뜬 프로세스 · overlay 의
      128+n) · `exitedStop` 표 (200 · 400 · 404 · 409 · 그 밖의 4xx · 5xx) · `exitBackoff` 수열 · 시험 서버로
      goroutine 네 경우 (200 에서 멈춤 · 404 에서 멈춤 · 503 두 번 뒤 200 · 답을 늦추는 서버에 `Stop` 이 오면
      끝남) · 본문이 `contract.Exited.Check` 를 지난다

### Step 8 — 업로드 client 와 흘려 보내기 (FD 엔티티 6절 · FD 규칙 8절 · N3)

- [x] `Client.Upload *http.Client` — 주석은 FD 엔티티 6절. nil 이면 `HTTP`
- [x] `upload.go` 로 `UploadLog` · `PutBlob` 을 옮긴다 (4절 ①). `UploadLog` 는 모양 그대로 `Upload` client 를
      쓴다. `PutBlob(ctx, runID, seq, name, body io.Reader, size int64)` — `ContentLength` 를 싣고 흘려 보낸다.
      4xx 는 `*BlobRejected`
- [x] `uploadProduced(ctx, step, out, log) (produced []string, stage contract.Stage)` — `ReadDir` 순서 ·
      `.enode-` 와 `.part` 를 건너뛴다 · 파일을 열어 stat 으로 크기 · 결과마다 FD 규칙 8절의 표 (거절은 영향
      없음 · 마감이면 남은 이름을 안 올리고 timeout · 그 밖의 전송 실패는 error 로 적고 다음 이름)
- [x] `cmd/enode/main.go` — `Upload: &http.Client{}` 와 주석 한 덩어리 (롱폴 client 와 칸을 나누는 까닭 —
      한쪽을 조이면 다른 쪽이 따라 조여지면 안 된다)
- [x] `upload_test.go` — 시험 서버가 받은 `Content-Length` 와 본문 · 413 이면 `BlobRejected` 이고 produced 에
      없고 stage 는 ok · 늦게 답하는 서버에 짧은 ctx → 남은 이름을 안 올리고 timeout · 끊는 서버 → error 이고
      다음 이름은 올라간다 · `.part` 를 안 올린다

      **한 일과 다른 점** — `Client` 구조체가 `advertise.go` 에 있어 그 파일에 칸 하나를 더했다 (2절의 표에 없던
      파일). `UploadLog` 의 4xx 도 `BlobRejected` 로 돌려준다 — 단계 로그도 같은 표를 따른다 (FD 규칙 8절)

### Step 9 — 진단 · 로그 끝 · 판정 칸 (FD 규칙 3.2 · 5.3 · 6절 · 7절)

- [x] `finalize.go` — `settle(...)` 이 `finalize` · `upload` · `reason` · `error` 를 정한다 (FD 규칙 3.2 의 표와
      문구). 임대 만료와 실행 실패가 덮어쓰는 규칙도 여기
- [x] `diagnosticsFor(step, out, result, finalized bool)` — 칸 여섯 (FD 규칙 6.1). `missing` 은 `$OUT` 을 직접
      읽어 채운다 (`.part` 는 없는 것). Finalize 가 결과를 못 돌려줬으면 `collect` · `discovered` 가 비고
      `changes` 는 `not_measured`
- [x] `logTail(step, result, diag, finalizeTimeout, budget)` — FD 규칙 6.3 의 표 순서대로 줄을 내고 마지막에
      7절의 안내. 목록은 큰 것부터 20 줄 · 크기는 `human`
- [x] `claim.go` — `writeHarvestNote` · `writeChangedNote` · `logHarvest` 를 지운다
- [x] `finalize_test.go` — `settle` 표 (FD 규칙 9절의 경로마다 한 줄) · `diagnosticsFor` 의 `changes` 네 경우
      (FD 규칙 5.3) · `logTail` 의 줄마다 한 경우 · 안내 두 줄의 조건 (effect 를 적은 run · discover 를 켠
      단계에는 없다) · 어느 경우에도 「no files changed」가 없다

### Step 10 — 명령 단계의 흐름 (`Worker.execute` · FD 흐름 1절)

- [x] `session.Run` 뒤 — `exited_at` · runCtx 확인 · `exitOutcome` · `startExitReport` · Finalize ctx
      (`WithDeadline(runCtx, exited_at + finalize)`) · `session.Finalize` · `session.Close(ctx, Keep{})` ·
      `finalized_at` · 진단과 로그 끝 · 업로드 ctx · `UploadLog` · `uploadProduced` · `settle` · `Stop` · `report`
- [x] 임대가 끝났으면 — 닫기와 result (`aborted: lease expired`). 프로세스가 안 떴으면 — 닫기 · 단계 로그 업로드 ·
      result (runErr). 둘 다 새 칸을 안 채운다 (FD 규칙 9절)
- [x] `stopCmdTee` 는 오늘처럼 단계 로그 업로드 앞
- [x] `step finished` 로그 줄은 오늘 모양을 두고 `finalize` · `upload` 를 더한다

### Step 11 — agent 단계의 흐름 (`Worker.runAgentStep` · FD 흐름 2절)

- [x] `Job.Exited` 로 `exited_at` 을 받고 조건이 맞으면 `startExitReport`
- [x] 단계 로그 업로드를 Finalize 뒤로 옮긴다 (오늘 `claim.go:874`)
- [x] 완주했으면 collect · diff (edit) · 훑기 (discover). 못 했으면 stat · 훑기 (discover)
- [x] Finalize 오류는 `finalize: error` 로 적고 업로드와 보고를 계속한다 (오늘은 바로 보고했다 · `claim.go:886`).
      완주 못 했으면 `PutBlob` 은 안 한다 (오늘 그대로). `harness:` 문구는 오늘 그대로
- [x] 하네스가 못 떴을 때 (4절 ⑩)
- [x] 앞에서 실패하는 `report` 클로저의 경로(파라미터 · 하네스 이름 · 자격증명)는 오늘 그대로 — 새 칸 없음

### Step 12 — Worker 시험 (`finalize_worker_test.go` · 조각 1 과 3 의 기계 부분)

시험 서버는 오늘의 worker 시험과 같은 모양이다 — claim · exited · log · blob · result 를 받아 기록한다.

- [x] **조각 1** — 보통 파일 500 개가 있는 워크스페이스에서 build effect 단계: `walkDir` 호출 0 · `statPath`
      호출이 지목 경로 수와 같다 · `$OUT` 의 이름이 PutBlob 으로 올라가 produced 에 있다 · 단계 로그와 `$OUT` 에
      「no files changed」와 `workspace.changed` 가 없다 · diagnostics 의 `changes` 가 `not_measured`
- [x] **조각 1 의 부분 관찰** — discover 를 켜고 방문 상한을 작게 준 `walkWorkspace` 시험은 Step 4 에 있다.
      여기서는 discover 를 켠 단계의 result 에 `changes: measured` 와 `discovered` 가 실리는지
- [x] **조각 3** — `Worker.budgets` 로 예산을 줄인다. Finalize 가 마감을 넘기는 가짜 세션 → `finalize: timeout` ·
      `reason: finalize_timeout` · error 문구 · `exit_code` 가 남는다 · 업로드는 계속한다. PUT 을 늦게 답하는 서버 →
      `upload: timeout` · `reason: upload_timeout` · 남은 이름이 produced 에 없다. 둘 다 명령 실패(exit 1)와 다른
      칸에 남는다
- [x] **종료 보고** — 명령 단계와 agent 단계가 한 번씩 보낸다 · 본문의 node · instance · attempt · outcome ·
      `exited_at` 이 result 의 `exited_at` 과 같다 · 서버가 404 를 주면 다시 안 보내고 단계는 정상으로 끝난다 (옛
      Mediator) · 5xx 를 주는 동안 result 가 나가면 재전송이 멈춘다
- [x] **경로마다의 새 칸** — FD 규칙 9절의 표 여덟 줄. 명령 앞 실패 · 안 뜬 프로세스 · 임대 만료는 새 칸이 없다
- [x] **옛 Mediator 의 claim** — effect · budget · discover 가 없는 claim 응답이 기본값으로 돈다
- [x] **agent** — 완주 못 한 하네스에도 discover 를 켰으면 훑는다 · Finalize 오류에도 단계 로그가 올라간다 ·
      안내 줄이 단계 로그 끝에 있다
- [x] `worker_test.go` — `uploadProduced` 를 부르던 시험 둘을 새 모양으로 · workspace.changed 를 찾던 시험
      (`:608`) 을 「없다」로 뒤집는다

      **한 일과 다른 점** — 프로세스를 띄우는 시험이라 `finalize_worker_test.go` 에 `!windows` 태그를 달았다
      (`worker_unix_test.go` 와 같다). `worker_test.go` 의 collect 시험도 프로세스를 띄우게 되어 이 파일로 옮겼다

### Step 13 — 측정 둘 (3절)

- [x] `BenchmarkWalkWorkspace` 를 한 번 돌리고 초당 방문 수를 적는다
- [x] 마감 뒤 돌아오기 시험의 최대값을 적는다 (`-v` 로 한 번)
- [x] 숫자는 Step 16 에서 `code-summary.md` 에

      **측정 2026-09-25 (이 기계 · 캐시가 데워진 디스크)** — `BenchmarkWalkWorkspace` 초당 약 304,000 방문
      (항목 100,100 을 5 번). 방문 상한 2,000,000 에 약 6.6 초 — 시간 상한 30 초보다 먼저 닿는다.
      마감 뒤 돌아오기 — 훑는 중 0.19 ms · 256 MiB collect 중 0.89 ms. 5 초는 넉넉하다

### Step 14 — 조각 스크립트 (FD 흐름 8절 · 4절 ⑪)

- [x] `scripts/finalize-bake/slice-1.sh` — `M` · `T` 를 받는다. `enode` · `runctl` 을 빌드한다. 스크래치에
      워크스페이스 둘 — 큰 것은 디렉터리 1,000 x 빈 파일 3,000 (이미 있으면 다시 안 만든다), 빈 것은 파일 셋.
      노드 둘을 설정 파일 둘로 띄운다 (`--config` 가 신원을 정한다). 계약은 `requires` 의 `ws` 로 노드를 고른다.
      단계는 `["touch", "d0/f0", "d1/f1", "d2/f2", "$OUT/result"]` (결정론 · 지목 경로 셋) ·
      `success_when` 은 `changed` 셋과 `produced` 하나. 세 Run — 큰 워크스페이스 · 빈 워크스페이스 · 큰
      워크스페이스에 `"discover": true` (3절 ①)
- [x] 판정 — 앞 두 Run 이 DONE · 봉인된 Record(`runctl record`) 의 단계 기록에서 `finalized_at - exited_at` 의
      차가 1초 안쪽 · 두 노드 로그의 `finalized` 줄 `checked=3` · `$OUT/result` 가 봉인에 있다. 셋째 Run 은 판정에
      안 넣고 `visited` · `took` · `discovery_limit` 을 출력한다. 끝나면 노드를 내린다. 출력 끝 줄은 `slice 1: green`
      또는 `slice 1: red` 와 이유
- [x] `scripts/finalize-bake/slice-2.sh` — 사람이 보는 조각이다. 단계마다 무엇을 볼지 영어로 출력하고 멈춘다
      (Enter 로 다음). ① exit 1 로 끝나고 큰 `$OUT` (256 MiB) 을 올리는 단계 — 명령이 끝나는 동안 진행 조회를
      1초마다 출력 (`phase` · `phase_since` · `exit`) ② Mediator 프로세스를 10초 멈춘 채(`MEDIATOR_PID` 를 받는다 ·
      `kill -STOP` 과 `-CONT`) 같은 계약 → 두 Record 의 단계 기록을 나란히 출력 ③ 다른 instance 로 보낸 종료
      보고가 409 인지 `curl` 로 ④ 옛 노드(`OLD_ENODE` — `0c0370c` 에서 빌드한 enode 경로)에 같은 계약 →
      `phase: running` 에 머무는지 ⑤ 단계 기록의 `exited_at` 과 `finalized_at`
- [x] 두 스크립트 — `bash -n` · (있으면) `shellcheck`. 사내 이름 0. 출력 문자열 영어 · 주석 한국어.
      `bash -n` 통과 · 이 기계에 `shellcheck` 이 없어 못 돌렸다

### Step 15 — 코드 검사와 조각 (유닛 정의 0절 · CI 와 같은 명령)

- [x] `gofmt -l .` 빈 출력 · `go vet ./...` · `go vet -tags integration ./internal/enode/` · `go build ./...`
- [x] `go test ./... -count=1` — 실패 0 · 스킵 0
- [x] `go test ./... -count=1 -coverpkg=./... -coverprofile=...` — 패키지마다 80% 이상. `internal/enode` 는
      Step 1 의 값과 나란히 적는다. 새 함수의 줄마다 시험이 닿는지 `go tool cover -func` 로 본다
- [x] 크로스 빌드 셋 — `GOOS=windows GOARCH=amd64` · `GOOS=linux GOARCH=arm GOARM=7` · `GOOS=darwin GOARCH=arm64`
- [x] `enodectl.exe` 의 심볼 상한 (crypto/tls 10 · net/http 50)
- [x] U+2605 0 (`grep -rlIP '\x{2605}'`) · `go run ./scripts/glyphscan.go` (Go 문자열의 장식 문자)
- [x] `golangci-lint` 경고 수가 Step 1 보다 늘지 않는다
- [x] **조각 0** (기동이 안 깨졌다) — build · vet · test 통과 · `grep -c 'mux.HandleFunc' internal/api/api.go` 가 19
- [x] **조각 1** (걷지 않는다) — Step 12 의 시험이 초록 · `slice-1.sh` 를 이 기계의 스크래치 Mediator 로 돌려
      `slice 1: green`. 스크래치 Mediator 는 `packaging/mediator/dev-up.sh` 로 띄운다 (이미 돌면 그것을 쓴다).
      못 띄우면 조각 1 을 보류로 적는다 — 보류는 통과가 아니다
- [x] **조각 3** (예산) — Step 12 의 시험이 초록
- [x] **조각 2** (보인다) — 사람의 조각이다. 스크립트만 준비하고 돌리지 않는다. 사용자가 돈다
- [x] `git status` 로 2절 밖의 파일이 없는지. `cmd/enodectl/probe.lock` 이 바뀌면 되돌린다 (Reverse Engineering 이
      적은 알려진 부채)
      **결과 2026-09-25** — 통과 2,025 · 실패 0 · 스킵 0 · 스무 패키지 전부 80% 이상 · `internal/enode` 81.5% -> 82.3% ·
      전체 85.9% · 크로스 빌드 셋 · 심볼 1 · 6 · U+2605 0 · glyphscan 깨끗 · 린트 39 건 (목록이 Step 1 과 같다) ·
      조각 0 (라우트 19) · 조각 1 `slice 1: green` · 조각 3 시험 초록.
      **한 일과 다른 점** — 이 기계의 개발용 Mediator(`:8080`)가 2026-09-17 빌드라 종료 보고에 404 를 준다. 그래서
      `dev-up.sh` 의 것을 쓰지 않고, 이 브랜치에서 빌드한 Mediator 를 `127.0.0.1:18080` 에 따로 띄웠다 (DB `enode_slice` ·
      같은 Postgres 컨테이너). 조각 1 의 계약은 첫 실행에서 400 이었다 — `changed` 는 워크스페이스를 적은 단계에만 쓸 수
      있어 단계에 `"workspace": {"repo": ""}` 를 더했다

### Step 16 — 코드 요약 문서

- [x] `construction/finalize/code/code-summary.md` — 고친 파일과 새 파일 · 없앤 이름 · 규칙이 어느 함수에
      있나 · 4절의 결정 열하나 · 코드 검사 결과(숫자) · 측정 둘의 숫자 · 조각 1 · 3 의 결과와 조각 2 의 돌리는 법 ·
      뒤 유닛에 넘기는 것 · 정본에 되돌려 올릴 것 (FD 흐름 12절 · 진행자가 올린다)
- [x] 표기 검사 — `enode-design/scripts/emphasis-check.py` 를 새 문서와 이 계획에 돌린다. 사용자가 싫어한 말투 ·
      사내 이름 · U+2605 검사

### Step 17 — 상태 · 감사 · 커밋

- [x] 이 계획의 체크박스를 채운다 (단계를 끝낸 그 자리에서)
- [x] `aidlc-state.md` 의 U3 절 · `audit.md` 에 완료와 승인 요청을 적는다
- [x] 한 커밋 — 코드 · 시험 · 스크립트 · 이 계획 · code-summary · 상태 · 감사 (CONVENTIONS 3.4). 승인 뒤에 넣는다
      (CONVENTIONS 3.3)

---

## 6. 이 단계가 하지 않는 것

```text
   runRoot 를 trash 로 · 닫기를 Finalize 예산 안으로            trash 유닛
   Keep.Upper 를 쓰는 일 (대기 자리 · spool) · checkpoint_capture   bake · checkpoint 유닛
   build · merge 단계의 수확과 종료 보고                        bake 유닛
   Git changeset adapter · producer adapter                    순연 (FR-11 — 결과 adapter 둘)
   Mediator (internal/store · internal/api)                    바뀌지 않는다.  step-phase 가 받았다
   명령 timeout (outcome timeout)                              오늘 명령 timeout 이 없다 (FD 규칙 4.1)
   공용 Reverse Engineering 문서                               다음 전면 갱신이 다시 센다
   정본(enode-design) 되돌림                                    진행자가 올린다 (FD 흐름 12절)
   조각 2 를 돌리는 일                                          사람의 조각이다.  사용자가 돈다
   PR 과 병합                                                  조각 1 · 2 · 3 이 초록인 뒤.  올리기 전에 묻는다
```

---

## 7. 확장 준수 — Code Generation

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표에서 이 유닛에 닿는 줄은 없다 — 노드는 종료 보고에 자기 instance 를 싣기만 하고 대조는 step-phase 가 Mediator 에서 한다. 조각 2 의 ③ 이 사람 눈으로 다시 확인한다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 한 줄 (Step 2 · 4 · 7 · 9) |
