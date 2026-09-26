# `finalize` — Functional Design 계획

**유닛** `finalize` (결과 확정) · **브랜치** `unit/finalize` · **담당** taeels ·
**회차** `v4-run-finalize-bake` (굽기) · **순서** 여덟 중 셋째 · **앞 유닛** step-phase (`main` `0c0370c`
에 병합) · **맡는 조각** 1 (걷지 않는다 · 기계) · 2 (보인다 · 사람) · 3 (예산 · 기계) ·
**완료 조건** 3 ④ (명령 단계가 diff 를 안 낸다) · 3 ⑤ (agent 단계의 전체 훑기가 꺼진다) ·
**스토리** US-8 (finalizing 이 언제 끝나는지) · US-9 (명령 단계의 diff 가 사라진 것) · US-10 (agent 의
workspace.changed 가 사라진 것)

입력은 유닛 정의(`unit-of-work.md` 3절) · 요구(`requirements.md` FR-1 Finalize 로 닫기 · FR-2 종료 보고 ·
FR-3 예산 둘 · 5.4 임대 창) · 설계(`services.md` 1절 명령 단계 흐름 · `components.md` 3.1 · 3.8 · 5절 실패 등급 ·
`component-methods.md` 4.1 · 4.2) · 조각 정의(`requirements/finalize-bake/scene-gates.md` 2절) ·
정본 ADR-075 (단계는 commit set 을 돌려준다) §5 · §8 · §9 · §10 · 앞 두 유닛이 넘긴 일
(`construction/contract-grammar/functional-design/business-logic-model.md` 8절 ·
`construction/step-phase/functional-design/business-logic-model.md` 10절)이다. 이 계획은 그 위에서
**노드가 명령이 끝난 뒤의 구간을 어떻게 닫나**만 짓는다. 코드는 다음 단계다.

---

## 0. 이 단계가 닫는 것과 안 닫는 것

```text
   닫는다     두 예산이 덮는 구간 — 어디서 시작해 어디서 끝나고, 넘으면 무엇이 멈추고 무엇이 계속되나
             종료 보고 — 보내는 조건 · 보내는 자리 · 재전송과 멈추는 응답
             effect 마다 무엇을 거두나 · 명시 훑기의 상한 넷과 부분 관찰 표기
             진단 칸을 채우는 법 · 사람이 읽는 자리(단계 로그 끝) · workspace.changed 를 어떻게 하나
             바뀐 기본값의 안내 (완료 조건 3 ④ ⑤)
             업로드 client 와 흘려 보내기
             result 의 새 칸 여섯을 노드가 채우는 규칙 — exited_at · finalized_at · finalize · upload ·
               reason · diagnostics
             세션 겉면 — Harvest 를 Finalize 로 · Close 가 행선지(Keep)를 받는다
             조각 1 · 2 · 3 의 확인 모양

   안 닫는다   runRoot 를 trash 로 옮기고 닫기를 rename 으로                  trash 유닛
             Keep.Upper 를 실제로 쓰는 일 (대기 자리 · spool)                bake · checkpoint 유닛
             checkpoint_capture 칸                                        checkpoint 유닛
             Git changeset adapter · producer adapter                      순연 (FR-11)
             build · merge 단계의 수확과 종료 보고                           bake 유닛
             값의 크기 — 업로드 예산과 요청의 관계 · 흘려 보내기의 상한 (N3) ·
               임대 창이 트리 크기와 무관한지                                 이 유닛의 NFR Requirements
             Mediator                                                     바뀌지 않는다 (step-phase 가 받았다)
```

---

## 1. 실측 — 코드가 지금 어떻게 생겼나 (2026-09-25 · `0c0370c`)

### 1.1 명령 단계의 끝 (`internal/enode/claim.go`)

```text
   session.Run(runCtx)       :688   임대를 감시하는 runCtx (:604).  임대가 끝나면 명령을 죽인다
   session.Harvest(ctx)      :691   RecordDiff 와 Discover: true 를 늘 켠다 — effect 를 모른다.
                                    ctx 는 임대를 모른다.  예산이 없다
   session.Close()           :702   overlay 는 helper 를 닫고 runRoot 를 지운다 — upper 크기에 비례한다
   uploadProduced(ctx)       :705   $OUT 의 파일마다 os.ReadFile 로 통째로 읽어 PutBlob (:917)
   UploadLog(ctx)            :713   단계 로그 (Record 의 logs/ 에 들어가는 선별본)
   report                    :731   result.  임대가 죽을 때까지 재시도
```

HTTP 는 모두 `Client.HTTP` 이고 요청마다 30초 제한이 있다 (`cmd/enode/main.go:206`). 롱폴만 제한 없는
client 를 따로 쓴다.

### 1.2 agent 단계의 끝 (`claim.go:866` ~ `:905`)

- 단계 로그를 **수확 전에** 올린다 (`:874`). 그래서 수확이 알게 된 것을 로그 끝에 붙일 수 없다
- 하네스가 완주하지 못했으면 collect · diff · 훑기를 안 하고 지목 경로 stat 만 한다 (`:876` ~ `:881`)
- 하네스 프로세스는 `runHarness` 안에서 끝난다 (`runner.go:234`). 그 뒤 봉투 해석과 하네스 버전 확인
  (프로세스 하나)을 한 다음 돌아온다

### 1.3 수확 (`runtime.go:132` · `nativeSession.Harvest`)

- **diff** — 워크스페이스에서 git diff (`diff.go:174`). 노드에 워크스페이스가 있고 계약이 `workspace`
  를 적었을 때만 (`claim.go:692`). 결과는 `$OUT/workspace.diff` 로 올라간다
- **collect** — glob 으로 찾아 `.part` 에 복사하고 rename 한다 (`collect.go:177`). 반쯤 복사한 파일이
  그 이름으로 남지 않는다. 다만 프로세스가 죽으면 `.part` 파일이 `$OUT` 에 남는다. ctx 를 받지 않는다
- **지목 경로 stat** — `CheckChanged`. 계약이 적은 경로만 본다 (ADR-037). 안 바꾼다
- **전체 훑기** — `changedSince` 가 워크스페이스를 끝까지 걷는다. 상한은 들고 있는 항목 200,000
  (`changed.go:76`)과 목록 2,000 뿐이다. **방문 수와 시간 상한이 없다**
- **workspace.changed** — `writeHarvestNote` (`claim.go:961`)가 `$OUT` 에 쓰고 produced 로 올라간다.
  빠진 산출물 · collect 실패 · 바뀐 파일 목록 셋을 싣는다. 목록이 비면 「no files changed in the
  workspace」라고 쓴다 (`:1002`). 이 파일을 읽는 코드는 0 이다 (`requirements.md` FR-1). 시험 하나가
  그 blob 을 찾는다 (`worker_test.go:616`)
- **훅의 걷기** (`hook.go:283`)는 하네스가 도는 동안 산출물이 빠졌을 때만 돈다. Finalize 와 무관하다.
  안 바꾼다

### 1.4 overlay 세션 (`runc_overlay_linux.go`)

- `Harvest` 가 ctx 를 버리고(`:388`) helper 에 harvest 요청을 보낸다. helper 는
  `context.Background()` 로 native 수확을 돈다 (`:980`)
- **helper 는 요청을 하나씩 처리한다** (`:657`). 수확 중에는 cancel 요청을 읽지 못한다. 그래서 Worker 가
  ctx 를 끊어도 helper 안의 수확은 멈추지 않는다 — 마감을 요청에 실어야 helper 안에서 멈춘다
- `Close` — helper 를 닫고(unmount), 3초 안에 안 끝나면 SIGKILL, 그다음 `os.RemoveAll(runRoot)`
  (`:406` ~ `:441`). upper 는 `runRoot/upper` 다 (`:801`) — 이 단계가 쓴 것만 들어 있다

### 1.5 Mediator 가 이미 받는 것 (step-phase · PR #62)

- `POST /v1/runs/{run}/steps/{seq}/exited` — 200 `{accepted}` · 409 와 문구 · 400 · 404 · 503.
  **옛 Mediator 는 라우트가 없어 404 를 준다** (Go 의 mux). step-phase 가 넘긴 「405」는 실제로 안 나온다
- result 의 새 칸 아홉을 값으로 거절하지 않는다. `error` 가 비어 있지 않으면 완주가 아니다 → 그 단계는
  실패 쪽으로 간다 (`internal/api/api.go:432`)
- claim 응답(`Claimed`)이 `effect` · `budget` · `discover` 를 싣는다. **노드의 `Step` 구조체가 아직 안
  받는다** (`claim.go:22`)
- blob 업로드 — Mediator 는 상한+1 바이트까지 읽고 413 을 준다 (`api.go:1056` 이후 · `record.ErrTooBig`).
  Content-Length 를 미리 보지 않는다
- 계약 쪽 도구 — `Step.EffectOrDefault` · `Step.Budgets` · 기본값 상수 (`internal/contract/effect.go`) ·
  `Diagnostics` · `Stage` · 원인 코드 상수 (`internal/contract/result.go`)

### 1.6 정본이 이미 정한 것 (ADR-075)

```text
   §9   Finalize 의 순서 — 프로세스를 멈춘다 · named output 확정 · 지목 경로만 관찰 · edit 일 때만
        source change · 진단 · cache 제외 · 닫기 · 봉투.  모든 연산은 임대를 아는 ctx 와 시간 · 방문 ·
        바이트 상한을 받는다.  payload 를 통째로 버퍼링하지 않는다
   §10.2 ①  프로세스가 뜨기 전에 실패한 단계는 종료 보고 없이 결과 보고로 간다
   §10.2 ⑤  예산 둘 — Finalize 1분 · 업로드 3분.  넘으면 FAILED 와 원인 코드
   §10.4    종료 보고는 Finalize 를 막지 않는다 · 실패하면 나란히 재시도 · result 가 같은 사실을 싣는다
   §8   훑기는 진단이다.  상한 넷 · 부분 관찰을 적는다 · produced 가 아니다
   §5   실패 · timeout 중의 반쪽 파일을 produced 로 올리지 않는다.  의도해서 완성한 진단
        (테스트 보고서)은 남길 수 있다
```

### 1.7 한 줄 순서에서 생기는 창

trash 유닛이 들어오기 전까지 닫기는 runRoot 를 **지운다**. 조각 4 (trash) 가 쓰는 규모(15만 파일 ·
9 GB upper)면 분 단위다. `services.md` 1절은 「닫기가 rename 뿐이라 Finalize 예산 안에 든다」를 전제로
예산이 닫기까지 덮게 했다. 그 전제는 trash 유닛 뒤에야 성립한다 — 물음 1.

---

## 2. 물음 아홉

답을 `[Answer]:` 뒤에 적는다. 권장을 **A** 에 둔다. 기호 옆에 뜻을 적었다.

### Question 1 — 두 예산이 덮는 구간

```text
                  종료 status                                                 result
                       |                                                        |
   -- 명령 ------------+-- Finalize --+-- 닫기 --+-- 업로드 --------+-- 보고 ---->
   A (권장)            [ Finalize 예산 ]          [ 업로드 예산      ]
   B                   [ Finalize 예산 ---------- ] [ 업로드 예산      ]
                       exited_at                 finalized_at (A · B 둘 다 닫기 뒤)
```

A) **Finalize 예산은 종료 status 가 정해진 때부터 Finalize (collect · 지목 경로 stat · diff · 명시 훑기)가
   끝날 때까지다. 이 유닛에서 닫기는 예산 밖이다.** 오늘 닫기는 runRoot 를 지우므로(1.7) 예산 안에 두면
   trash 가 들어오기 전까지 큰 upper 를 남기는 overlay 단계가 `finalize_timeout` 으로 실패한다. trash
   유닛이 닫기를 rename 으로 바꾸는 그 커밋에서 닫기를 예산 안으로 옮긴다 — 그때 `services.md` 1절의
   모양이 된다. `finalized_at` 은 닫기가 끝난 시각이다 (그 사이의 지우는 시간도 있는 그대로 적힌다).
   업로드 예산은 닫기 뒤부터 마지막 업로드가 끝날 때까지다. **단계 로그를 먼저, `$OUT` 의 이름들을 뒤에**
   올린다 — 로그가 가장 작고 실패를 읽는 재료라서 예산이 모자라도 남게. result 보고는 두 예산 밖이다 —
   오늘처럼 임대가 죽을 때까지 재시도한다

B) 처음부터 닫기까지 Finalize 예산 안에 둔다 (`services.md` 그대로). trash 가 들어오기 전까지 큰 upper 를
   남기는 단계는 `finalize_timeout` 으로 실패할 수 있다. 한 줄 순서라 그 창은 trash 병합까지다

C) A 와 같되 `finalized_at` 을 Finalize 가 끝난 때(닫기 전)로 적는다 — 예산과 시각이 같은 구간을
   가리킨다. 대신 trash 유닛 뒤에 `finalized_at` 의 뜻이 한 번 바뀐다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2 — 예산을 넘겼을 때

A) **아래 표대로.**

```text
   Finalize 의 ctx   runCtx (임대를 안다) 에 마감을 건다.  임대가 끝나 멈춘 것은 finalize_timeout 이
                     아니다 — 오늘의 "aborted: lease expired" 그대로
   native           그 ctx 가 collect (파일 사이와 복사 중) · git diff · 훑기 · stat 을 멈춘다
   overlay          마감 시각을 helper 요청에 싣는다.  helper 가 그 시각으로 자기 ctx 를 만든다.
                    Worker 는 마감 + 5초를 기다리고, 답이 없으면 helper 를 죽이고(오늘의 abort) 닫는다
   Finalize 초과    finalize=timeout · reason finalize_timeout · error "finalize budget of 1m0s exceeded".
                    닫기 · 업로드 · 보고는 계속한다.  $OUT 에 있는 것은 올린다 — collect 는 .part 에 쓰고
                    rename 하므로 반쯤 복사한 것은 그 이름으로 없다.  이름이 .part 로 끝나는 파일은 안 올린다
   업로드 초과       진행 중 요청을 끊고 남은 이름을 안 올린다.  못 올린 이름은 produced 에 없다.
                    upload=timeout · reason upload_timeout · error "upload budget of 3m0s exceeded"
   둘 다            reason 은 먼저 난 finalize_timeout.  error 는 한 줄에 둘을 적는다
```

   error 칸을 채우므로 Mediator 는 완주가 아닌 것으로 받고(`api.go:432`) 단계는 실패한다. 명령의 exit code
   는 result 에 그대로 남아 명령 실패와 나뉜다 (조각 3 — 예산)

B) A 와 같되 Finalize 예산을 넘기면 `$OUT` 을 안 올리고 로그만 올린다 — 어차피 실패한 단계다. 대신 실패한
   단계가 남긴 테스트 보고서 같은 것도 안 남는다

C) A 와 같되 overlay 에 마감을 싣지 않는다 — Worker 가 마감에 곧바로 helper 를 죽인다. helper 안의 수확이
   스스로 멈출 기회가 없고 helper 를 죽이는 일이 더 잦다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3 — 종료 보고를 언제 · 어디서 보내고 언제 멈추나

A) **아래 표대로.**

```text
   보낸다          프로세스가 떴고 끝났다.
                   종료 코드가 0 이상              exit {code}
                   native 가 signal 로 죽었다      signal {신호 번호}  (exec.ExitError 의 WaitStatus)
                   overlay                        runc 가 128+n 을 종료 코드로 준다 — 그대로 exit
   안 보낸다        프로세스가 안 떴다 (ADR-075 §10.2 ①) · 임대가 끝나 죽였다 (Run 이 이미 회수됐다) ·
                   명령 앞에서 실패했다 (워크스페이스 준비 · $IN · runtime open)
   timeout 종류     오늘 명령 timeout 이 없어 이 유닛은 안 쓴다
   agent 단계       하네스 프로세스가 끝난 때.  Job 에 종료 콜백 하나를 더해 runner.go 가 session.Run
                   바로 뒤에 부른다 (파일 행렬 밖 한 곳)
   exited_at       종료 status 를 받은 순간의 노드 시계.  result 의 exited_at 도 같은 값
   재전송          goroutine 이다.  Finalize 를 막지 않는다
                     멈춘다   200 (accepted 가 false 여도) · 409 · 400 · 404 · 그 밖의 4xx
                     다시     5xx · 끊김 — 1초에서 두 배씩, 간격은 최대 10초
                     그만     result 를 보내기 시작하면 — result 가 같은 사실을 싣는다
   client          오늘의 30초 client
```

   멈춤은 그 단계에만 걸린다 — 단계마다 한 번 보내므로 Run 단위로 기억하지 않는다. step-phase 는
   「404 · 405 면 그 Run 동안 다시 안 보낸다」로 넘겼는데, 옛 Mediator 에 한 단계마다 요청 하나를 더
   쓰는 비용이라 기억을 두지 않는다

B) A 와 같되 agent 단계는 `runHarness` 가 돌아온 뒤 보낸다 — runner.go 를 안 고친다. 하네스 버전 확인
   (프로세스 하나 · 약 50 ms)만큼 `exited_at` 이 늦다

C) A 와 같되 result 를 보낸 뒤에도 재시도를 이어 간다 (늦게 닿아도 200 이다)

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4 — agent 가 완주하지 못했을 때 무엇을 거두나

effect 마다 거두는 것은 앞 유닛이 정했다 (contract-grammar — read 는 build 와 같다).

```text
   단계 · effect           diff (workspace.diff)         훑기                collect · stat · $OUT
   run · build (기본)       없음                           discover 일 때만     있다
   run · read              없음                           discover 일 때만     있다
   run · edit              오늘 조건 그대로 (주)            discover 일 때만     있다
   agent · edit (기본)      오늘 조건 그대로 (주)            discover 일 때만     있다
   agent · read            없음                           discover 일 때만     있다

   (주) 노드에 워크스페이스가 있고 계약이 workspace 를 적었을 때 (claim.go:692).  agent 는 Git changeset
        adapter 가 생길 때까지 둔다 (Units Generation 물음 7)
```

남는 것은 하네스가 완주하지 못했을 때(크래시 · max_turns · 하네스 오류)다. 오늘은 stat 만 한다.

A) **오늘처럼 collect 와 diff 는 안 하고 stat 만 한다. discover 를 켰으면 훑기는 돈다** — 실패한
   sandbox 를 조사하는 것이 훑기의 둘째 쓰임이다 (ADR-075 §8)

B) 오늘처럼 stat 만 한다. 훑기도 완주했을 때만 돈다

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5 — 명시 훑기의 상한 · 훑는 곳 · 부분 관찰

계약은 `"discover": true` 로 켜기만 하고 상한은 노드가 정한다 (contract-grammar 물음 5). 정본은 상한 넷
(방문 수 · 시간 · 메모리 · 결과 크기)을 요구한다.

A) **아래 표대로.**

```text
   방문          2,000,000 항목 (디렉터리 포함)
   시간          Finalize 예산의 절반 — 기본 30초.  계약이 finalize 예산을 늘리면 같이 는다.
                 훑기가 예산을 다 써서 단계를 finalize_timeout 으로 만들지 않게 하려고
   메모리        들고 있는 항목 200,000 (오늘 changedSince 의 상한 그대로).  Go 에서 힙을 직접 막을 수
                 없으므로 항목 수로 막는다
   결과 크기     diagnostics.discovered 에 경로 2,000 개 · 합 256 KiB 까지.  큰 파일부터 (오늘 순서)
   닿으면        먼저 닿은 하나를 discovery_limit 에 적는다 (visits · time · memory · size).
                 changes 는 partial
   다 훑으면     changes 는 measured
   안 켰으면     changes 는 not_measured.  워크스페이스가 없는 노드에서 켰어도 not_measured 와 로그 한 줄
   훑는 곳       native — 워크스페이스를 기준 시각(mtime)으로 (오늘 방법)
                 overlay — upper 만.  upper 가 곧 이 단계가 쓴 것이라 비용이 바뀐 수를 따른다.
                 지운 항목(whiteout)은 목록에 안 넣고 수만 로그에 적는다
```

B) A 와 같되 overlay 도 merged 를 기준 시각으로 훑는다 — native 와 같은 방법이고, lower 까지 걷는다

C) A 와 같되 시간 상한을 고정 30초로 둔다 — finalize 예산과 무관하다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6 — 진단이 가는 자리와 workspace.changed

A) **`$OUT` 에 workspace.changed 를 더 쓰지 않는다.** 그 파일이 싣던 셋(1.3)은 두 자리로 옮긴다.

```text
   result 의 diagnostics 칸    missing · collect · changes · effect · discovered · discovery_limit.  봉인된다
   단계 로그 끝의 몇 줄         사람이 Record 의 logs/ 에서 읽는다.  "enode: " 로 시작하는 영어 줄
```

   「no files changed in the workspace」는 사라진다. agent 단계의 로그 업로드를 수확 뒤로 옮긴다 (1.2).
   workspace.diff 는 edit 에서 오늘처럼 `$OUT` 의 산출물이다. `writeChangedNote` 와 `writeHarvestNote` 는
   지운다

B) workspace.changed 를 계속 쓰되 바뀐 파일 목록만 뺀다 — 빠진 산출물과 collect 실패가 오늘 자리(produced
   의 이름 하나)에도 남는다. diagnostics 칸과 같은 것이 두 곳에 있게 된다

C) A 와 같되 단계 로그 끝의 줄은 쓰지 않는다 — diagnostics 칸만 쓴다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 7 — 바뀐 기본값의 안내 (완료 조건 3 ④ ⑤)

완료 조건 3 은 「바뀐 기본값이 그것을 겪는 사람이 읽는 자리 하나에 적힌다」다. ④ 는 effect 를 안 적은
명령 단계가 diff 를 안 내는 것, ⑤ 는 agent 단계가 workspace.changed 를 안 내는 것이다.

A) **단계 로그 끝에 한 줄.** 조건과 문구는 아래다. effect 를 적은 run 단계와 discover 를 켠 단계에는 안
   쓴다. 기한을 두지 않는다.

```text
   run 단계가 effect 를 안 적었고 discover 도 안 켰다
     enode: this run step used the default effect build, so it returns no workspace.diff and no
     list of changed files. Write "effect": "edit" if the command changes source files, or
     "discover": true to list what changed.
   agent 단계가 discover 를 안 켰다
     enode: agent steps no longer return workspace.changed. Write "discover": true to list what
     changed.
```

B) A 의 두 줄을 노드 로그에, 노드가 뜬 뒤 처음 한 번씩만 쓴다 — 계약 작성자는 노드 로그를 못 본다

C) A 와 같되 effect 를 적은 run 단계에도 쓴다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 8 — 업로드 client 와 흘려 보내기

A) **아래 표대로.**

```text
   client        Client 에 Upload 칸 — 요청마다의 제한이 없는 http.Client.  마감은 업로드 예산의 ctx 가
                 준다.  PutBlob 과 UploadLog 가 쓴다.  진행 청크 · exited · result · claim · GetBlob 은
                 30초 client 그대로
   흘려 보내기    PutBlob 이 파일을 열어 크기(Content-Length)와 함께 보낸다.  통째로 읽지 않는다
   크기 검사      노드가 미리 거르지 않는다.  Mediator 가 상한+1 바이트에서 413 으로 끊으므로 보내는 양이
                 상한을 안 넘는다 (1.5).  노드는 Mediator 의 상한 값을 모른다
   재시도         없다 (오늘 그대로)
   upload 칸      ok · timeout · error.  error 는 예산 안에서 전송이 실패한 이름이 하나라도 있을 때다.
                 판정은 안 바꾼다 — error 칸을 안 채우고, 그 이름이 produced 에 없을 뿐이다 (오늘 그대로).
                 413 · 422 는 Mediator 의 거절이라 업로드 실패가 아니다
```

B) A 와 같되 노드가 10 MiB (`diff.go:168` 의 값)를 넘는 파일은 안 보낸다 — Mediator 설정이 다르면
   어긋난다

C) A 와 같되 전송이 실패한 이름을 예산 안에서 한 번 더 보낸다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 9 — 조각 1 (걷지 않는다) 을 기계로 확인하는 모양

조각 1 은 「합성 300만 파일 workspace 에서 결정론 no-op 단계의 시간이 빈 workspace 와 같은 수준 · 지목한
N 개만 stat · `$OUT` named output 은 그대로 봉인」이다. 집행자는 기계다.

A) **둘로 한다.** (1) 기본 `go test` — build effect 의 Finalize 가 워크스페이스를 한 항목도 방문하지 않고,
   지목 경로가 N 개면 stat 이 N 번인지를 센다 (걷는 함수와 stat 앞에 세는 자리를 둔다). CI 에서 돈다.
   (2) 조각 스크립트 — 스크래치 디렉터리에 빈 파일 300만 개를 만들고 native 노드에 no-op 계약을 내어
   `finalized_at` 과 `exited_at` 의 차를 빈 워크스페이스와 비교한다. 기준은 차이 1초 안쪽. CI 밖이다
   (파일 300만 개를 만드는 데 수 분이 걸린다). 판정은 스크립트가 한다

B) 스크립트만 둔다

C) `go test` 만 둔다 — 방문 0 이 크기와 무관함을 보인다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## 3. 산출물 계획 (체크박스)

답이 들어오고 모호함이 풀린 뒤에 채운다. 자리는
`aidlc-docs/v4-run-finalize-bake/construction/finalize/functional-design/` 이다.

- [x] 답을 읽고 모호함을 확인한다 — 있으면 되물음 파일을 만든다 (2026-09-25T12:26:32Z · 답 아홉 모두 A · 답끼리 막는 자리 없음 · 되물음 없음)
- [x] `domain-entities.md` — 노드 `Step` 의 새 칸 셋 (effect · budget · discover) · `FinalizeSpec` ·
      `FinalizeResult` · `Keep` · `StepSession` 의 새 겉면 · 노드 `Result` 의 새 칸 여섯 · 종료 보고를
      보내는 쪽의 모양 · `Client.Upload` · 훑기 상한 상수 · helper 요청의 마감 칸
- [x] `business-rules.md` — effect 표 · 예산의 경계와 넘었을 때 · finalize 와 upload 칸과 reason 을 정하는 표 ·
      종료 보고를 보내는 조건과 멈추는 응답 · 훑기 상한과 부분 관찰 · 진단 칸 채우기 · 로그 끝 줄과 안내 문구
      (영어) · 업로드 규칙
- [x] `business-logic-model.md` — 명령 단계의 흐름 · agent 단계의 흐름 · overlay 에 마감 전달 · 종료 보고
      goroutine · 업로드 순서 · 옛 Mediator 와 함께 사는 법 · 한 줄 순서의 창 (trash 전) · 조각 1 · 2 · 3 의
      확인 모양 · 파일 행렬 밖 자리 · 다른 유닛에 넘기는 것 · 정본 되돌림
- [x] 파일 행렬 밖 자리 — `runner.go` (물음 3) · `collect.go` (ctx) · `diff.go` · `hook.go` (안 바꾼다)가
      필요한지 확인해 적는다
- [x] 커버리지 — `internal/enode` 의 여유가 1.5점이다. 규칙을 순수 함수로 떼는 자리를 적는다
- [x] 표기 검사 (`enode-design/scripts/emphasis-check.py`) · 사용자가 싫어한 말투 검사 · 사내 이름 검사

---

## 4. 확장 준수 — 이 단계

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표에서 이 유닛에 닿는 줄은 없다 — 종료 보고의 인스턴스 대조는 Mediator 쪽이고 step-phase 가 닫았다. 노드는 자기 instance 를 싣기만 한다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 저장소의 표 시험 관례를 따른다 |

유닛 정의는 이 유닛의 NFR Requirements 를 「한다」로 적었다 — 성능 (임대 창이 트리 크기와 무관한지 ·
N3 업로드 예산과 요청의 관계 · 흘려 보내기의 상한). Functional Design 이 닫힌 뒤에 한다.
