# `finalize` — 흐름

타입은 `domain-entities.md`, 규칙과 문구는 `business-rules.md` 에 있다. 여기는 그 규칙이 어느 자리에서
어느 순서로 도나, 옛 Mediator 와 어떻게 함께 사나, 조각을 어떻게 확인하나, 그리고 다른 유닛에 무엇을
넘기나다.

---

## 1. 명령 단계의 흐름 (`Worker.execute` 의 뒷부분)

```text
   준비 · 세션 열기 · argv          오늘 그대로
   session.Run(runCtx)             명령
   exited_at := now                 session.Run 이 돌아온 순간
   runCtx 가 끝났나?                예 -> 닫기 -> result ("aborted: lease expired").  끝
   exitOutcome(code, runErr)        ok 가 아니면 (프로세스가 안 떴다)
                                      -> 닫기 -> 단계 로그 업로드 -> result (runErr).  끝
   startExitReport                  goroutine.  Finalize 를 안 막는다
   finalize ctx := WithDeadline(runCtx, exited_at + finalize 예산)
   session.Finalize(ctx, spec)      [Finalize 예산]  collect · 지목 경로 stat · (edit) diff · (discover) 훑기
   session.Close(ctx0, Keep{})      닫기.  예산 밖 (이 유닛)
   finalized_at := now
   진단 칸 · 로그 끝 줄 · 안내        순수 함수 (9절)
   upload ctx := WithTimeout(runCtx, 업로드 예산)
   UploadLog -> PutBlob ...          [업로드 예산]  단계 로그 먼저 -> $OUT 의 이름들
   settle                           finalize · upload · reason · error 를 정한다 (순수 함수)
   exitReporter.Stop()              종료 보고 재시도를 그만둔다
   report(result)                   오늘 그대로.  임대가 죽을 때까지 재시도
```

- `spec` 은 1절의 표에서 온다 — `Diff` 는 effect 가 edit 이고 오늘 조건이 맞을 때, `Discover` 는 계약의 값
- 트랜스크립트 tee 를 멈추는 자리(`stopCmdTee`)는 오늘처럼 단계 로그 업로드 앞이다
- `ctx0` 는 Worker 의 ctx 다. 닫기가 예산 밖이라 마감을 걸지 않는다

---

## 2. agent 단계의 흐름 (`Worker.runAgentStep`)

```text
   runHarness(runCtx, …, Job{Exited: onExit})
      session.Run 이 돌아온 순간 onExit(code, runErr, now)
         runCtx 가 살아 있고 exitOutcome 이 ok 면 startExitReport.  exited_at 을 기억한다
      봉투 해석 · 버전 확인 · 자백 읽기 (오늘 그대로)
   stopTee
   (단계 로그 업로드를 여기서 하지 않는다 — 오늘의 claim.go:874 를 뒤로 옮긴다)
   runCtx 가 끝났나?                 예 -> 닫기 -> result.  끝
   spec — 완주했으면 collect · diff (edit) · 훑기 (discover).  못 했으면 stat · 훑기 (discover)
   session.Finalize(finalize ctx)   [Finalize 예산 — exited_at 부터]
   session.Close                    닫기
   finalized_at · 진단 · 로그 끝 줄 · 안내
   UploadLog -> PutBlob ...          [업로드 예산].  완주 못 했으면 PutBlob 은 안 한다 (오늘 그대로)
   settle · exitReporter.Stop · report
```

- 하네스가 뜨지 못했으면(`exitOutcome` 이 ok 가 아니다) `exited_at` 이 없다. 그때 Finalize 예산은
  `runHarness` 가 돌아온 순간부터 센다 — Finalize 는 돌지만(stat 뿐) 종료 보고와 `exited_at` 은 없다
- 오늘 Harvest 가 오류를 내면 produced 를 안 올리고 바로 보고했다 (`claim.go:886`). 이제 Finalize 오류는
  `finalize: error` 로 적고 업로드와 보고를 계속한다 — 명령 단계와 같은 모양이다

---

## 3. overlay 에서 마감을 지키는 법

```text
   Worker                                   helper (요청을 하나씩 처리한다)
   send {op: finalize, finalize: spec}  -->  ctx := WithDeadline(Background, spec.Deadline)
                                             waitRun -> native Finalize(ctx, spec)
   select                                    (merged 에서 collect · stat · diff /
     응답이 왔다          -> 그대로 돌려준다     upper 에서 훑기)
     마감 + 5초가 지났다  -> abort (SIGKILL)  <--  {op: finalized, finalize: result}
                           DeadlineExceeded
   Close                                     (helper 가 없으면 남은 runRoot 지우기만)
```

- helper 가 스스로 멈출 기회를 먼저 준다. 5초는 helper 가 ctx 를 보고 돌아와 응답을 쓰는 여유다
- abort 뒤의 `Close` 는 helper 에게 말하지 못한다. 남은 runRoot 를 지우는 일만 한다 — 오늘의 abort 가
  이미 그렇게 한다
- helper 의 시계와 Worker 의 시계는 같은 기계의 것이다. 마감을 시각으로 나르는 까닭이다

---

## 4. 종료 보고 goroutine

```text
   startExitReport(ctx, step, body)
      loop
         stop, err := Client.Exited(ctx, run, seq, body)      30초 client
         stop            -> 끝
         Stop 을 받았다    -> 끝
         ctx 가 끝났다     -> 끝
         기다린다 1 · 2 · 4 · 8 · 10 · 10 … 초 (Stop 이 오면 바로 끝)
   Stop()   result 를 보내기 직전.  goroutine 이 끝나기를 기다린다
```

- 기다리는 동안 Finalize · 닫기 · 업로드가 돈다. Stop 은 그것들이 끝난 뒤에 온다. 그래서 재전송은
  길어야 한 단계의 finalizing 구간 동안이다
- 재전송이 result 보다 늦게 닿아도 Mediator 는 200 으로 받고 아무것도 안 바꾼다 (step-phase 수락 표 4)

---

## 5. 업로드

```text
   upload ctx (업로드 예산)
   UploadLog(단계 로그 + 끝 줄)                         Client.Upload
   for name in ReadDir($OUT):
      .enode-* · *.part 는 건너뛴다
      f := open · size := stat
      PutBlob(ctx, …, f, size)                        Client.Upload · Content-Length
         nil          produced 에 넣는다
         BlobRejected 안 넣는다 (Mediator 의 판정)
         마감          안 넣는다 · 남은 이름도 안 올린다 · upload=timeout
         그 밖         안 넣는다 · upload=error · 다음 이름
```

`uploadProduced` 가 이 모양이 된다. 파일을 통째로 읽던 자리(`claim.go:917`)가 사라진다.

---

## 6. 함께 사는 법

```text
   옛 Mediator (step-phase 전)      Claimed 에 effect · budget · discover 가 없다 -> 기본값.
                                   exited 에 404 -> 그 단계는 다시 안 보낸다.
                                   result 의 새 칸 여섯은 모르는 칸이라 버린다 (JSON 풀기가 모르는 칸을
                                   허락한다).  판정은 error 칸으로 하므로 예산 초과는 그대로 실패다
   effect 를 안 적은 옛 계약        run 단계가 build 로 돈다 -> diff 와 목록이 안 나온다 (US-9).
                                   로그 끝의 안내 한 줄이 이유를 알린다 (완료 조건 3 ④)
   agent 단계를 쓰는 옛 계약        workspace.changed 가 안 나온다 (US-10).  in.from 이나 feedback 이 그
                                   이름을 가리켜도 「없으면 안 깔린다」 규칙대로 실패가 아니다
                                   (claim.go 의 missingIn).  안내 한 줄 (완료 조건 3 ⑤)
   predicate (success_when.changed) 바뀌지 않는다 — 지목 경로 stat 은 늘 한다
```

---

## 7. 한 줄 순서의 창 — trash 전

trash 유닛이 병합되기 전까지 닫기는 runRoot 를 지운다.

```text
   Finalize 예산       Finalize 만 덮는다.  닫기가 upper 크기에 비례해도 finalize_timeout 이 안 난다
   finalized_at        닫기가 끝난 때다.  지우는 시간이 들어 있다 — 조각 4 가 없애려는 비용이 그대로 보인다
   조각 1              native 로 확인한다.  닫기가 지울 upper 가 없다
   조각 2              exit 1 과 긴 업로드로 확인한다.  닫기의 시간과 무관하다
```

**trash 유닛에 넘기는 일** — 닫기를 rename 으로 바꾸는 그 커밋에서 닫기를 Finalize 예산 안으로 옮긴다.
`Close(ctx, Keep)` 의 ctx 에 Finalize ctx 를 넘기면 된다. `finalized_at` 의 뜻(닫기가 끝난 때)은 그대로다.

---

## 8. 조각 1 · 2 · 3 의 확인 모양

### 조각 1 — 걷지 않는다 (기계 · 답 9 = A)

**기본 go test** (`internal/enode` · CI 에서 돈다)

- 보통 파일 수백 개가 있는 임시 워크스페이스에서 build effect 의 Finalize 를 돌리고
  `FinalizeResult.Discovery == nil` 과 **훑기 방문 0** 을 확인한다. 방문은 걷는 함수 앞의 세는 자리로 센다
- 지목 경로가 N 개면 stat 이 N 번인지 센다 (`CheckChanged` 앞의 세는 자리)
- `$OUT` 의 named output 이 그대로 올라가는지 (PutBlob 을 받는 시험 서버)
- discover 를 켠 단계가 상한에 닿으면 `changes: partial` 과 `discovery_limit` 이 적히는지 — 상한을 작게 한
  시험용 값으로
- 걷기를 안 켠 단계의 단계 로그와 `$OUT` 에 「no files changed」가 없는지

**조각 스크립트** (CI 밖 · 판정은 스크립트)

- 스크래치 디렉터리에 빈 파일 3,000,000 개를 만든다 (디렉터리 1,000 개 x 파일 3,000 개)
- 그 디렉터리를 워크스페이스로 쓰는 native 노드에 결정론 no-op 계약(`run: ["true"]`)을 낸다
- 빈 워크스페이스 노드에 같은 계약을 낸다
- 두 Run 의 봉인된 Record 에서 `finalized_at` 과 `exited_at` 의 차를 읽어 비교한다. **차이가 1초 안쪽이면
  초록**이다. 노드 로그에서 그 단계의 stat 수가 계약이 지목한 수와 같은지도 본다
- 스크립트의 자리와 이름은 Code Generation 계획이 정한다

### 조각 2 — 보인다 (사람 · 스크래치 Mediator)

Code Generation 계획이 스크립트로 굳힌다. 사람이 보는 것은 `scene-gates.md` 3절 그대로다.

- exit 1 로 끝나고 큰 `$OUT` 을 올리는 명령 단계 — 명령이 끝나는 순간 `GET /v1/runs/{id}` 에
  `phase: finalizing` 과 `exit {kind: exit, code: 1}` 이 보인다
- Mediator 를 10초 멈춘 채 같은 계약 — 종료 보고가 유실되거나 재전송되고, 봉인된 Record 의 단계 기록이
  멈추지 않은 Run 과 같다 (`exited_at` 은 result 가 싣는다)
- 이 유닛 전의 노드(`0c0370c` 의 enode) — `phase: running` 에 머문다
- 봉인된 Record 의 단계 기록에 `exited_at` 과 `finalized_at` 이 따로 있다

### 조각 3 — 예산 (기계)

기본 go test 로 한다. 계약의 Finalize 예산은 1분 아래로 못 내리므로(contract-grammar) Worker 에 예산을
정하는 함수를 바꿔 끼울 자리를 둔다 — 시험만 쓴다.

- Finalize 가 예산을 넘기는 가짜 세션 → `finalize: timeout` · `reason: finalize_timeout` · error 문구 ·
  `exit_code` 가 그대로 남는다
- PUT 을 늦게 답하는 시험 서버 → `upload: timeout` · `reason: upload_timeout` · 남은 이름이 produced 에 없다
- 계약이 늘린 예산(`"budget": {"finalize": "5m", "upload": "10m"}`)이 그 값으로 잡히는지 — 예산 함수의
  표 시험
- 둘 다 명령 실패(exit code)와 다른 칸에 남는지

---

## 9. 커버리지 — 규칙을 순수 함수로

`internal/enode` 의 여유가 1.5점(81.5%)이다. 새 문장의 대부분이 이 패키지에 들어가므로 규칙을 순수
함수로 떼어 표 시험으로 덮는다.

```text
   finalizeSpecFor(step, completed, local)    effect 표 (business-rules.md 1절)
   stepBudgets(step)                          두 예산.  contract.Step.Budgets 를 부른다
   exitOutcome(code, runErr)                  종료 보고의 outcome 과 보낼지
   exitedStop(status)                         응답마다 멈출지
   exitBackoff(n)                             재전송 간격
   settle(finalizeErr, closeErr, uploadErr…)  finalize · upload · reason · error
   diagnosticsFor(step, out, result)          진단 칸
   logTail(step, result, …)                   로그 끝 줄과 안내
   discoverLimits · walkWorkspace · walkUpper  상한 넷.  상한을 인자로 받아 시험이 작게 준다
```

흐름(goroutine · HTTP · helper 요청)은 오늘의 worker 시험과 같은 방식(시험 서버 · 가짜 세션)으로 덮는다.
namespace 가 필요한 overlay 의 훑기와 마감은 `integration` 태그 시험이다 — CI 밖이고 사람이 SunnyVM 에서
돈다 (`requirements.md` 5.6).

---

## 10. 파일 행렬 밖의 자리

행렬(`unit-of-work-file-matrix.md` 1.3)이 이 유닛에 준 파일은 `runtime.go` · `runc_overlay_linux.go` ·
`runc_overlay_other.go` · `claim.go` · `changed.go` · `upload.go` · `cmd/enode/main.go` · 조각 스크립트다.

```text
   internal/enode/runner.go     Job.Exited 콜백 한 칸과 부르는 한 줄 (답 3 = A)
   internal/enode/collect.go    collectDeclared 와 copyFile 이 ctx 를 받는다 (마감에 멈추려고)
   internal/enode/diff.go       바꾸지 않는다 — workspaceDiff 가 이미 ctx 를 받는다
   internal/enode/hook.go       바꾸지 않는다 — 훅의 걷기는 Finalize 와 무관하다
   시험 파일                     worker_test.go (workspace.changed 를 찾던 시험) · changed_test.go ·
                               collect_test.go (writeChangedNote 시험을 지운다)
```

`runc_overlay_other.go` 는 `Finalize` 가 없다 — Open 이 늘 실패하므로 세션이 생기지 않는다. 바꿀 것이 없을
수 있다. Code Generation 계획이 확인한다.

---

## 11. 다른 유닛에 넘기는 것

```text
   trash        닫기를 rename 으로 바꾸는 커밋에서 닫기를 Finalize 예산 안으로 (7절).
                Close(ctx, Keep) 의 Keep.Upper 가 "" 면 runRoot 를 trash 로.  finalized_at 의 뜻은 그대로
   checkpoint   받아들임과 spool 은 닫기 안이고 Finalize 예산의 남은 몫을 쓴다 (ADR-075 §10.2 ⑤ ·
                ADR-076 §4).  Keep.Upper 에 spool 자리.  result 의 checkpoint_capture
   bake         build 단계의 종료 보고 — 마지막 명령이 끝난 뒤 한 번 (Client.Exited · startExitReport 를
                쓴다).  merge 단계는 안 보낸다.  build 단계의 두 예산 (계약이 build 에 budget 을 허락한다)
   이 유닛의     값의 크기 — 30초에 훑는 항목 수 · helper 여유 5초 · 업로드 예산과 요청의 관계 (N3) ·
   NFR          흘려 보내기의 상한 · 임대 창이 트리 크기와 무관한지 (조각 1)
```

---

## 12. 정본에 되돌려 올리는 것

진행자가 `enode-design` 에 올린다.

```text
   protocol/run-contract.md   discover 의 상한 넷과 그 값은 노드가 정한다 (값은 이 문서 5.2) ·
                              discover 는 계약의 workspace 없이 켤 수 있지만 노드에 워크스페이스가 있어야
                              목록이 나온다 · workspace.changed 가 더는 안 나온다 · workspace.diff 는 diff 가
                              있는 단계(business-rules.md 1절)에서만 · 단계 로그 끝의 안내 줄
   ADR-075 §9                 HarvestSpec 의 칸마다 판정 (domain-entities.md 2.1) · Harvest 가 Finalize 가 됐다
   ADR-075 §10.2 ⑤           Finalize 예산이 닫기를 덮는 것은 닫기가 rename 이 된 뒤다 (trash 전의 창)
   ADR-075 §10.4              노드의 재전송 규칙 — 200 · 4xx 에서 멈춤 · 5xx 와 끊김에서 1 ~ 10초 간격 ·
                              result 를 보내면 그만
   mediator-api.md exited 절   step-phase 가 적은 되돌림에 더해 — 옛 Mediator 는 라우트가 없어 404 를 준다
```

---

## 13. 확장 준수 — Functional Design

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표에서 이 유닛에 닿는 줄은 없다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 한 줄 |

다음 단계는 유닛 정의대로면 이 유닛의 NFR Requirements (성능 — 11절의 값)다.
