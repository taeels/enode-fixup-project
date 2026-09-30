# `bake` — 흐름

타입은 `domain-entities.md`, 규칙과 문구는 `business-rules.md` 에 있다. 여기는 그 규칙이 어느 자리에서 어느 차례로 도나, 어떻게
확인하나, 계획이 받은 일이 어디에 닿았나, 그리고 다른 유닛 · 정본에 무엇을 넘기나다. 아래에서 「규칙 N절」은 `business-rules.md`,
「엔티티 N절」은 `domain-entities.md`, 「흐름 N절」은 이 문서다. 「되물음 N 답 A · B」는 `construction/plans/bake-functional-design-clarification-questions.md`
의 물음과 그 답이다 (2026-09-27T10:38:41Z · 다섯 모두 A — 되물음 2 는 2026-09-27T11:59:51Z 에 B 로 다시 답함).

---

## 1. 한눈에 — 하루치 굽기

굽는 노드 K, 긴 Run 을 도는 형제 A, 쉬는 형제 B 가 한 lower 위에 있다. 굽기 계약은 build 와 merge 두 단계다 (FR-5 굽기 계약).

```text
   시간 ------------------------------------------------------------------------------------------------------>

   K (굽기)     build claim: 공유 놓음 . 굽기 잠금 . building -- sync -- IR 대조 -- pinned -- builds --
                닫기(Keep) . 초안 . pending . HoldBake -- merge claim -- 배타 대기 ........ 잡음 . 그물 .
                Preflight . merging . Apply . metadata . committed . 대기 자리 trash . 두 잠금 놓음 . DropBake
   A (형제 Run)  공유 (run) ........................... drain 실음 ......... Run 끝 . 응답 두 번 . 놓음     공유 다시 . 새 ir
   B (후보)      공유 (candidate) ..................... drain 실음 . 응답 두 번 . 놓음                     공유 다시 . 새 ir
   Mediator     build: running -> finalizing (exited) -> DONE    merge: waiting .............. DONE    Run 판정 (produced)
```

- build 는 형제가 일하는 중에도 곧바로 돈다 — lower 를 읽기만 한다 (ADR-077 §6 합치기만 미룬다). merge 만 형제의 Run 을 기다린다
- K 의 배타는 A 와 B 가 공유를 놓아야 잡힌다 — 기다림은 A 의 Run 이 정한다 (ADR-077 §12 · 하루치 합치기 1.49 초)
- 합치면 A · B 는 공유를 다시 쥐고 · metadata 에서 새 ir 을 읽고 · drain 없는 그 광고에 싣는다 (lower-state 흐름 1절)
- K 가 building 인 동안 A · B 에는 drain 이 없다 — pending 부터 drain 이다 (`lowerguard.go:212` ~ `:216`)

---

## 2. build 단계 (`Worker.runBuildStep` · 규칙 3절)

```text
   execute (claim.go:508)
     OnClaim (prepare -> 공유 놓음) · defer StepDone                        오늘 그대로 (lower-state)
     $OUT · $IN 임시 폴더 · 임대 감시 (:655)                                 오늘 그대로
     kind == "build" -> runBuildStep                                       빈 argv 확인 (:676) 앞
       Bake 가 nil · Dir 이 nil                    -> 보고 FAILED (거절 · 규칙 1절)
       held · resuming 이면                        -> 보고 FAILED bake_in_progress
       lock, ok, err := dir.TryBake()   err        -> 보고 FAILED cannot open the lower state directory
                                        ok 아님    -> 보고 FAILED bake_in_progress (주인은 state.json)
       st, err := dir.ReadState()       err        -> lock.Release · 보고 FAILED cannot open the lower state directory
         merging                                   -> go resume(lock, st, "build claim") · 보고 FAILED bake_in_progress (되물음 4 답 A)
         building · pending                        -> 정리 (규칙 12.2 · 옛 대기 자리는 그 scratch 의 trash · 옛 주인의 last_attempt)
       hold := heldBake{run, step, dir, lock}      Baker.held = hold
       previous_ir := ReadMetadata(lower).source.ir   못 읽으면 null · 노드 로그 경고
       hold.pending := MkdirTemp(<scratch>/pending/<키>/)               실패 -> 몸통 · FAILED (결정 44)
       lock.WriteState(building{owner, pending_upper: <pending>/upper, last_attempt 그대로})   실패 -> 몸통 · FAILED (결정 44)
       session := runtime.Open(RuntimeSpec{Dir, In, Out, Record})    실패 -> 몸통 · FAILED runtime open:
       bash:   Run(["sh", "-c", "command -v bash >/dev/null"])        1 이상 -> 몸통 · FAILED cannot run the bake commands (되물음 3 답 A)
                                                                    -1 -> 몸통 · FAILED runtime run: · 임대 -> 임대 갈래 (결정 45)
       sync:   Run(["bash", "-c", sync])  기록 · 머리와 끝 줄             0 아님 -> skipping 줄 · 실패 끝 (DONE)
       probe:  Run(["sh", "-c", <고정된 대조 한 줄>])  irVerdict           어긋남 -> skipping 줄 · 실패 끝 (DONE ir_mismatch · 물음 1 답 B)
                                                                    못 함   -> 실패 끝 (FAILED · head_tags null)
       pinned: repo 모양이면 Run(["bash", "-c", "repo manifest -r -o .enode-manifest.xml"])   실패 -> 실패 끝 (FAILED)
       builds: 차례로 Run(["bash", "-c", command])                   0 아님 -> 남은 것 skipping · 실패 끝 (DONE)
       exited (마지막 build) · Finalize (effect prepare) · Close(Keep{Upper: <pending>/upper})
       pinned 의 sha256 (대기 자리에서) · 초안 쓰기 (임시 파일 · fsync · rename)
       Finalize 예산의 마감을 넘겼으면 -> pending 을 안 쓴다 · 몸통 (finalize_timeout) · 보고 FAILED
       lock.WriteState(pending) · guard.HoldBake(run, func() { hold.abandon("the bake run ended before the merge", 초안의 builds) })
       $OUT/manifest · 업로드 · 보고 DONE (exit 0 · build 칸)
       업로드 예산을 넘겼으면 hold.abandon(upload_timeout) · 보고 FAILED

   실패 끝 (명령 실패 · IR · 노드 쪽 오류)
       exited (마지막으로 돈 명령) · Finalize · Close(Keep{}) · $OUT/manifest 가 있으면 지움 (규칙 15절)
       hold.abandon(reason, 돈 builds)        대기 자리 trash · committed + last_attempt · 굽기 Release · DropBake · Baker.held 비움
       IR 어긋남이면 단계 로그 끝에 문장 한 줄 (규칙 4절 · 16.2)
       업로드 (단계 로그) · 보고 (DONE 또는 FAILED · build 칸 · 규칙 6절의 표)
```

- **초안의 sha256 칸은 닫은 뒤에 채운다** — pinned 파일은 세션의 upper 에 있고, 닫기가 그것을 대기 자리로 옮긴 뒤에야 호스트가 읽는다.
  초안은 한 번에 쓴다 (sha256 을 채운 뒤) — 차례는 닫기 · sha256 · 초안 쓰기 · pending 이다. 계획 3.6 은 「초안 -> sha256」 차례로
  적었다 — 초안이 sha256 을 담으므로 뒤집었다
- 굽기 잠금은 building 부터 pending 까지 이 프로세스가 쥔다. pending 뒤에도 쥔 채로 merge 단계를 기다린다 (ADR-077 §6 굽기 잠금)
- 명령마다 `session.Run` 은 세션의 `callMu` 로 차례가 지켜진다 (`runc_overlay_linux.go:346`)
- 노드의 고정 한 줄 둘 (bash 확인 · IR 대조) 은 POSIX sh 다 — 계약의 명령과 pinned 의 `repo manifest -r` 은 bash 로 돈다 (되물음 3 답 A). LC_ALL 등은 그 한 줄 안에서
  export 한다 (규칙 4절)
- bash 확인이 sync 앞에 있는 까닭 — bash 가 없는 rootfs 에서 `bash -c` 는 runc 가 실행 파일을 못 찾아 exit 1 이 된다 (QA 재검 측정 ·
  runc 1.1.5). sync 가 흔히 내는 실패 exit 1 과 구별되지 않는다. 명령 실패 (DONE) 로 보이지 않게 계약의 명령을 돌리기 전에 노드 쪽
  오류로 멈춘다 (규칙 3절). 처음 판의 「exit 127」은 틀렸다
- 명령 · 대조 · pinned 도중 helper 가 죽으면 `runtime run: <원인>` · exit -1 이다 (`runc_overlay_linux.go:380` ~ `:382`) — 오늘 명령 단계처럼
  FAILED · 몸통 (결정 48 · `claim.go:749` ~ `:761`)

---

## 3. merge 단계 (`Worker.runMergeStep` · 규칙 7 · 8절)

```text
   execute
     OnClaim (merge -> 공유가 있으면 놓음 · 거절하지 않음)                   오늘 그대로
     kind == "merge" -> runMergeStep
       Bake 가 nil · Dir 이 nil · ReadState 오류        -> 보고 FAILED (held 가 있으면 몸통 · 결정 44)
       claim 확인 (규칙 7.1)                         재시작 뒤 -> 보고 FAILED
                                                   합칠 것 없음 -> 보고 DONE · merged 없음 · 단계 로그 한 줄 (되물음 2 답 B)
       deadline := claimedAt + contractStep(step).MergeWait()
       loop:
         ex, err := dir.Exclusive(ctx(runCtx, deadline), 10s, watch -> waitLog -> 단계 로그)
           err: 마감 -> hold.abandon(merge_wait_timeout) · 보고 FAILED
                임대 -> hold.abandon("aborted: lease expired") · 보고 FAILED
                데몬 -> hold.abandon("node stopped") · 보고 없음
         scan := lower.ForeignMounts(root)
           찾음 -> ex.Release · 줄 · 60초 (같은 ctx) · loop
           오류 -> 경고 줄 · 계속
       MkdirAll(trash)                              실패 -> ex.Release · hold.abandon · 보고 FAILED (결정 44)
       helper preflight                            어긋남 · 읽기 실패 -> ex.Release · hold.abandon(문장) · 보고 FAILED
       hold.startMerging()                          false -> ex.Release · 끝 (몸통이 먼저 돌았다)
       lock.WriteState(merging)                     실패 -> 표지 거둠 · ex.Release · hold.abandon · 보고 FAILED
       helper apply                                 오류 -> merging 에 둠 (아래)
       rmdir(upper 뿌리) · WriteMetadata(초안 + bake)   오류 -> merging 에 둠
       lock.WriteState(committed)                   오류 -> 경고 · 대기 자리를 옮기지 않고 놓기로 (합치기는 끝났다 · 되물음 5 답 A 의 (3) ·
                                                   초안이 남아 재개가 「끝났나」를 대 본다 — QA 재검 B2)
       Trash.Move(대기 자리)                         committed 를 쓴 때만.  오류 -> 경고 · 계속 (버려진 자리 — 기동 정리가 거둔다)
       ex.Release · lock.Release · DropBake · Baker.held = nil
       $OUT/merged · 업로드 · 보고 DONE (merge 칸) · AfterReport (삭제자 Kick)
         업로드 예산 (기본 3분) 을 넘겼으면 보고 FAILED upload_timeout · merge 칸은 싣는다 (합치기는 끝났다 · 규칙 5절 · CG 물음 1 답 A —
         Code Generation 계획 10절 물음 1 의 답)

   merging 에 둠 (답 2)
       ex.Release · lock.Release · DropBake · Baker.held = nil (retryAt 은 걸지 않는다)
       보고 FAILED "merge stopped: ...; the lower stays merging and a node on this lower resumes it"
       -> 다음 광고의 OnStale 이 보통 이 노드에서 재개를 연다 (6절)
```

- `Exclusive` 는 잡을 수 있으면 먼저 잡고 그다음에 ctx 를 본다 (`lock_linux.go:163` ~ `:180` — flock 을 건 뒤 sleep 에서 ctx 를 본다).
  마감과 잡힘이 같은 순간이면 잡힌 쪽이 이긴다 — 합친다
- merge 단계의 단계 로그는 버퍼와 진행 청크 둘로 간다 — 기다리는 동안 Mediator 에서 보인다 (규칙 8.6 · US-15)
- **committed 를 쓴 뒤에 대기 자리를 옮긴다** — merging 인 동안 초안이 늘 있어 재개가 「이미 끝났나」를 초안으로 대 본다 (규칙 12.4)

---

## 4. 치우는 몸통과 다른 길이 만나는 자리 (규칙 11절)

계획 2.3 이 측정한 대로 광고 응답 하나가 두 길을 함께 연다 — `bakeGoneLocked` 의 몸통 (`lowerguard.go:419`) 과 Worker 의 임대 감시가
끊는 단계의 ctx (`claim.go:667` 의 cancel). `heldBake` 의 mutex 와 두 표지(`done` · `merging`) 가 차례를 정한다.

| 먼저 | 나중 | 결과 |
|---|---|---|
| 광고 응답의 몸통 (pending 인데 그 Run 의 임대가 없다) | merge 단계의 `startMerging` | 몸통이 done 을 적었다 — merge 단계는 배타를 놓고 끝. 보고는 임대가 없어 닿지 않는다 |
| merge 단계의 `startMerging` | 광고 응답의 몸통 | 몸통은 merging 표지를 보고 아무것도 안 한다 — Apply 는 끝까지 간다 (결정 3-11 본체에 상한 없음) |
| 임대 감시가 대기를 끊음 | 광고 응답의 몸통 | 둘 다 `abandon` 을 부르고 한 번만 돈다 |
| build 단계의 예산 넘김 몸통 | 다음 광고 응답의 몸통 | 한 번만 돈다 — DropBake 가 먼저 불려 광고 쪽은 `g.bake` 가 비어 있다 |
| 몸통이 대기 자리를 옮김 | merge 단계의 helper preflight | preflight 가 directory 로 어긋난다 -> `abandon` 은 이미 돌았다 -> 배타를 놓고 끝 |
| build 의 명령 실패 몸통 (held 를 비움) | 같은 Run 의 merge claim | 7.1 의 「합칠 것 없음」줄로 간다 — held 가 남았으면 둘째 줄 (어긋남) 로 잘못 갔다 (QA S3) |

---

## 5. 기동 (`cmd/enode/main.go` · 규칙 12절)

```text
   1   env check                                   main.go:148 오늘 그대로.  점검 셋이 공유를 몇 초 쥔다 (lower-state)
   2   guard := StartLowerGuard(lowerRoot, ...)      :249 오늘 그대로
   3   baker := StartBaker(ctx, guard, scratchDir, ident, client.Instance, log)       새 한 줄
         guard.OnStale(baker.onStale)                 첫 줄 — 아래가 어디서 끝나도 광고 주기의 정리와 재개는 열려 있다
         dir := guard.Dir()                           nil 이면 기동 정리를 건너뛴다 (로그 한 줄)
         lock, ok, err := dir.TryBake()               err -> 로그 한 줄 · 끝.  ok 아님 -> 끝 (다른 노드가 쥐었다 — 그쪽이 한다)
         st, err := dir.ReadState()                   err -> lock.Release · 로그 한 줄 · 끝
         규칙 12.2 의 표                               committed 정리 · building · pending 정리 · merging 재개 (배경)
   4   worker.Bake = baker                            새 한 줄 (Worker 를 짓는 줄에 칸 하나)
   5   SweepOrphanSessions · 삭제자 (:312)            오늘 그대로 — 정리가 trash 에 넣은 대기 자리를 첫 회가 지운다
   6   탐지 · 광고 · claim · 삭제자 고루틴 (:329)       오늘 그대로.  첫 광고 전에 정리가 끝났다 (재개만 배경)
   7   멈출 때                                        wg.Wait 뒤 baker.Wait — 도는 Apply 를 기다린다
   입구 merge-helper                                  os.Args[1] == "merge-helper" 면 RunMergeHelper (trash-helper 의 :53 옆)
```

- main.go 에 더하는 것은 부르는 줄 넷 (merge-helper 입구 · StartBaker · Worker 의 칸 · Wait) 이다 — `cmd/enode` 가 80.5% 로 하한에 가깝다
  (lower-state code-summary 5절 표 `:144`)
- 기동 순서는 `services.md` 4절과 `components.md` 3.8 그대로다 — 상태 자리 · 굽기 잠금 · 정리와 재개 · 삭제자 첫 회 · 광고

---

## 6. 살아 있는 동안 정리와 재개 (답 2 · 되물음 1 답 A · 규칙 13절)

```text
   광고 주기 (Advertiser.Run -> guard.BeforeAdvert)
     sourcesLocked: st := dir.ReadState()
       st.Phase 가 building · pending · merging 이고 g.bake == nil 이면  go OnStale(st)     광고는 기다리지 않는다 (되물음 1 답 A)
     Baker.onStale(st)
       held · resuming · now < retryAt  -> 끝
       lock, ok, err := dir.TryBake()   err -> 로그 한 줄 (원인이 바뀔 때만) · 끝.  ok 아님 -> 끝 (주인이 살아 있거나 다른 쪽이 쥐었다)
       st2 := dir.ReadState()           잠금 뒤 다시 읽는다
         merging     -> resuming = true · go resume(ctx, lock, st2, "advert")
         building · pending -> 정리 (규칙 12.2 — 대기 자리를 그 scratch 의 trash · committed + last_attempt abandoned) · 놓는다
                               옮기기 실패 -> 로그 한 줄 · 그래도 committed (몸통과 같다 · 결정 53)
                               committed 실패 -> 잠금 Release · retryAt = now + 10분 · 오류가 바뀌었으면 로그
         committed   -> 놓는다 (그 사이 다른 노드가 끝냈다)
     resume (규칙 12.3 의 차례)
       성공 -> 로그 한 줄 · resuming = false · lastErr = ""
       실패 -> merging 그대로 · 두 잠금 Release · retryAt = now + 10분 · 오류가 바뀌었으면 로그 · resuming = false
```

| 장면 | 남는 것 | 누가 언제 잇나 |
|---|---|---|
| merge 단계의 Apply 가 오류로 멈춤 (데몬은 산다) | merging · 두 잠금 풀림 · 그 Run FAILED | 다음 광고에서 보통 그 노드 자신 (retryAt 없음). 같은 오류면 그 뒤 10분마다. 형제도 자기 광고에서 해 본다 |
| 합치는 중 K 가 SIGKILL · K 가 다시 뜸 | merging | K 의 기동 (규칙 12.2) 이나 먼저 TryBake 한 형제의 광고 |
| 합치는 중 K 가 SIGKILL · K 가 안 뜸 | merging | 형제의 광고 주기 — 한 주기(기본 60초) 안에 재개를 연다 · drain 은 재개가 끝난 뒤의 다음 광고에서 풀린다 |
| 재개가 같은 오류로 되풀이 (예 — 매핑 밖 uid 로 chown) | merging · 형제 drain | 사람이 원인을 고치면 다음 시도 (늦어도 10분) |
| build claim 이 낡은 merging 을 만남 | merging | 그 build 노드가 배경에서 연다 · build 는 bake_in_progress (되물음 4 답 A) — 굽기 Run 의 임대가 묶이지 않는다 |
| pending 을 쓴 뒤 merge claim 전에 K 가 SIGTERM · 안 뜸 | pending · 형제 drain | 형제의 광고 주기가 치운다 — committed · last_attempt abandoned. drain 은 두 주기 (기본 120초) 안에 풀린다 — 정리를 연 광고는 이미 drain 을 싣고 나간다 (되물음 1 답 A) |
| building 중에 K 가 죽음 · 안 뜸 | building (형제에 drain 없음) | 형제의 광고 주기가 치운다 (되물음 1 답 A) — 다음 굽기의 build claim 이 먼저 오면 그쪽이 치운다 |

- 재개는 `HoldBake` 도 `DropBake` 도 부르지 않는다 — DropBake 가 표지를 합친 뒤의 것으로 적으면 합치기 전에 매칭된 늦은 임대가 바뀐 lower
  위에서 돈다 (엔티티 10절 · 규칙 11절)
- merging 동안은 어느 노드도 공유를 새로 잡지 않으므로 재개의 배타는 보통 곧바로 잡힌다 — 기다리는 것은 smoke 가 쥐는 몇 초와 기록 없는
  쥔 쪽 (옛 판) 뿐이다 (규칙 13절)

---

## 7. 실패 갈래 한 장

`business-rules.md` 6절 (build) · 8.5절 (기다림) · 10절 (시작 전 확인) · 7절 (merging 뒤) 을 한 곳에 모았다.

| 자리 | 단계 | 원인 코드 | upper | state.json | last_attempt | 뒤 |
|---|---|---|---|---|---|---|
| 굽기를 못 받는 노드 · 상태 자리 · 잠금 파일 · state.json 을 못 엶 | FAILED | 없음 | 없음 | 안 건드림 | 없음 | Run 끝 |
| 주인이 살아 있음 | FAILED | `bake_in_progress` | 없음 | 그대로 | 없음 | Run 끝 · 다시 내면 된다 (US-16) |
| rootfs 에 bash 가 없음 · sh 를 못 띄움 | FAILED | 없음 | trash | committed | 남김 | Run 끝 (되물음 3 답 A · 결정 45) |
| 명령 · 대조 · pinned 도중 helper 가 죽음 | FAILED (`runtime run:`) | 없음 | trash | committed | 남김 | Run 끝 (결정 48) |
| held 가 있는 동안의 그 밖의 오류 (대기 자리 · building 쓰기 · merge 의 ReadState · trash 만들기) | FAILED | 없음 | trash (몸통) | committed | 남김 | Run 끝 · held 와 g.bake 가 비워진다 (결정 44) |
| 낡은 merging | FAILED | `bake_in_progress` | 없음 | merging | 없음 | 이 노드가 재개 (답 2 · 되물음 4 답 A) |
| sync · build 가 0 아님 | DONE | 없음 | trash | committed | 남김 | merge 가 온다 — 합칠 것 없음 · DONE.  Run 은 계약의 조건이 판정한다 (produced manifest · merged 를 건 계약이면 FAILED · 되물음 2 답 B) |
| IR 어긋남 | DONE (exit 0 · sync 의 것) | `ir_mismatch` | trash | committed | 남김 | merge 가 온다 — 합칠 것 없음 · DONE.  build 칸에 head · head_tags (완료 조건 10).  Run 은 계약의 produced manifest 조건이 FAILED 로 판정한다 (물음 1 답 B · I3 · 결정 54) |
| IR 을 못 대 봄 · pinned · 세션 · 닫기 · 초안 · pending 쓰기 | FAILED | 없음 | trash | committed | 남김 | Run 끝 |
| pending 전 Finalize 예산 | FAILED | `finalize_timeout` | trash | committed | 남김 | Run 끝 |
| pending 뒤 업로드 예산 | FAILED | `upload_timeout` | trash (몸통) | committed | 남김 | Run 끝 |
| build 보고 뒤 재시작 (merge claim 전) | 기동 정리 | — | trash | committed | 남김 (`abandoned: … pending`) | merge 가 새 생에게 온다 — FAILED 재시작 문장 (규칙 7.1) |
| 명령 실패 뒤 몸통이 committed 를 못 씀 | DONE (build) | 없음 | trash | building 이 남음 -> 광고 주기가 치움 | 남김 (`abandoned: … building`) | merge 는 합칠 것 없음 DONE — 재시작 줄이 아니다 (결정 52) |
| 대기 상한 | FAILED | `merge_wait_timeout` | trash | committed | 남김 | build 의 DONE 과 따로 보인다 (완료 조건 9 · US-17) |
| 임대가 사라짐 (build · merge 대기) | FAILED | 없음 | trash | committed | 남김 | drain 이 풀린다 |
| Preflight 어긋남 · 읽기 실패 (merge 단계) | FAILED | 없음 | trash | committed | 남김 | lower 는 그대로 |
| Apply · rmdir · metadata 오류 (merging 뒤) | FAILED | 없음 | 대기 자리에 남음 | merging | 없음 | 재개 (답 2) |
| metadata 뒤 committed 쓰기 오류 | DONE (merged) | 없음 | 대기 자리에 남음 (초안 · upper 뿌리는 치웠다) | merging | 그대로 — 재개가 committed 를 쓸 때 지운다 | 재개가 초안으로 끝났나를 보고 committed 만 쓴다 (되물음 5 답 A 의 (3)) |
| committed 뒤 대기 자리 옮기기 오류 | DONE (merged) | 없음 | 버려진 대기 자리 (초안만) | committed | 지움 | 그 scratch 를 쓰는 노드의 기동 정리가 거둔다 |
| 합친 뒤 업로드 예산을 넘김 (merge) | FAILED | `upload_timeout` | 합쳤다 (대기 자리는 trash) | committed | 지움 | Run 끝 · lower 는 합쳐졌고 metadata 의 bake.run 이 이 Run (ADR-077:290 ~ :292 와 같은 모양 · 규칙 5절 · CG 물음 1 답 A) |
| 재개 실패 | 단계 없음 | — | 대기 자리에 남음 | merging | 없음 | 그 노드는 10분 뒤 · 다른 노드는 자기 광고에서 |
| 데몬이 멈춤 (명령 중 · merge 대기 중) | 보고 없음 | — | trash | committed | 남김 (`node stopped`) | 임대 만료나 재시작 (ADR-030) 이 Run 을 닫는다 |
| 데몬이 멈춤 (pending 뒤 · merge claim 전) | 보고 없음 | — | 형제의 광고 주기가 trash | pending -> committed | 남김 (`abandoned: …`) | 형제가 치우고 drain 은 두 주기 안에 풀린다 (되물음 1 답 A) |

---

## 8. 확인의 모양

이 유닛은 조각 5 (굽기 계약 — 기계) 와 6 · 7 · 8 (굽기가 끝까지 돈다 · 끊겨도 된다 · 배타와 대기 — 사람 · SunnyVM · 버려도 되는 lower)
을 맡는다 (`unit-of-work.md` 7절). 넷이 이 유닛의 병합 게이트다 (`unit-of-work.md` 0절 · CONVENTIONS 3.3). 기대는 조각은 1 · 2 · 4 이고
셋 다 초록이다.

### 8.1 기계 (기본 `go test` · CI 에서 돈다)

```text
   internal/enode — 순수 함수 (표 시험)
     irVerdict        맞음 (태그 둘 · annotated · 가벼운) · 로컬에 없음 · 다른 커밋 · none 모양 · git 실패 — 문장 글자까지
     attemptReason    원인 코드 · 명령 exit · 대조 · 멈춤 · abandoned
     buildManifestOf · mergeOpsOf   칸 옮기기 (replaced = Replaced + TypeChanged 등 일곱) · 대조를 못 했으면 head_tags null
     metadataOf       merge 단계 (resumed false · node 는 이 노드) · 재개 (resumed true · node 는 초안의 Node)
     waitLog          처음 · 모습 바뀜 · 5분 · Unnamed · 마감 줄 없음 (재개)
     pending 모양      맞는 모양 · 다른 키 · 상대 경로 · upper 가 아닌 끝 · .. 이 든 경로
     url 비밀번호      https://u:p@h/x -> https://u@h/x · scp 모양 그대로 · 비밀번호 없는 url 그대로
     끝났나            bake.run 과 synced_at · head 가 모두 같을 때만 참 · run_id 만 같으면 거짓 (다시 쓴 run_id)
     contractStep     merge.wait 을 MergeWait 이 읽는다 · 없으면 4시간 · build 의 예산은 그대로
   internal/enode — 흐름 (가짜 세션 · 임시 lower 의 진짜 잠금과 상태 자리 · lower-state 의 newGuardFixture 모양)
     오류 길           held 를 세운 뒤의 오류마다 (MkdirTemp · building 쓰기 · merge 의 ReadState · trash 만들기) 몸통이 돌고 held 와
                      g.bake 가 비워진다 — 다음 굽기가 bake_in_progress 로 거절되지 않는다 (결정 44)
     bash 확인         exit 0 · 127 (dash) · 1 (sh 를 못 띄움 · bash 의 command -v) · -1 (세션 오류) · 임대 끝의 다섯 갈래 문구 (결정 45)
     분기             kind build · merge 가 빈 argv 확인에 안 닿는다 · Bake nil 이면 거절 문구 · Dir nil 이면 오류 ·
                      TryBake · ReadState 오류가 bake_in_progress 가 아니다
     build 성공        pending · 초안 · HoldBake 뒤 공유 안 잡음 · $OUT/manifest · build 칸 · exited 한 번
     build 실패        sync 실패 (builds 안 돎 · skipping 줄) · build 실패 (답 3 · 첫 실패에서 멈춤 · last_attempt 의 builds) ·
                      IR 두 갈래 (물음 1 답 B · DONE · error 없음 · reason ir_mismatch · 남은 구성마다 skipping 줄 · 문장은 단계 로그 끝 ·
                      head_tags · manifest 없음 · last_attempt ir_mismatch) · 대조를 못 함 (FAILED · head_tags null) ·
                      pinned 실패 · 명령이 쓴 $OUT/manifest 가 지워진다 · 명령 중 임대 만료 (exited 없음) ·
                      pending 전 Finalize 예산 (pending 을 안 씀)
     굽기 잠금         다른 Dir 이 bake.lock 을 쥐면 bake_in_progress (주인 문장) · 낡은 building · pending 정리 뒤 진행 ·
                      낡은 merging 이면 재개가 열리고 bake_in_progress
     merge            합침 (merged · merge 칸 · committed 뒤 대기 자리 옮김 · last_attempt 지움 · DropBake · held 비움) ·
                      committed 를 못 쓰면 대기 자리가 남는다 (재개가 초안으로 끝났나를 본다 · QA 재검 B2) ·
                      7.1 의 재시작 줄이 지금 state 의 phase 와 무관 — 다른 굽기가 building 을 써도 last_attempt 가 남는다 (결정 49) ·
                      재시작 줄은 pending 갈래뿐 — 명령 실패 뒤 몸통이 committed 를 못 쓴 building (주인이 이 Run) 과 building 을 치운
                      abandoned 기록은 합칠 것 없음 DONE (결정 52) ·
                      합칠 것 없음 (DONE · merged 없음 · error 없음 · 되물음 2 답 B) · 명령 실패 뒤의 merge 가 합칠 것 없음
                      으로 간다 (몸통이 held 를 비웠다 — 둘째 줄의 어긋남이 아니다) · 재시작 두 줄이 합칠 것 없음보다 먼저 맞는다 ·
                      판정 조건 없는 굽기 계약 + sync 실패 -> Run SUCCEEDED 와 Record 의 build 칸 (Mediator 시험 · 되물음 2 답 B) ·
                      IR 어긋남 뒤의 merge 가 합칠 것 없음 DONE (last_attempt ir_mismatch 는 재시작 줄이 아니다) ·
                      produced manifest 를 건 굽기 계약 + build DONE ir_mismatch -> Run FAILED · Record 에 reason 과 head_tags
                      (Mediator 시험 · 물음 1 답 B · 결정 54) ·
                      재시작 장면 — build 보고 뒤 Baker 를 새로 짓고 (재시작) 기동 정리 뒤 merge claim -> FAILED 재시작 문장 ·
                      그물이 찾으면 놓고 다시 · Preflight 어긋남 (committed · upper trash) ·
                      Apply 오류 (merging · 두 잠금 풀림 · retryAt 없음) · metadata 뒤 committed 오류 (DONE)
     조각 5 (merge.wait) (1) Mediator -> 노드 JSON 왕복 — store.Claimed 에 merge.wait 을 싣고 JSON 으로 옮겨 enode.Step 으로 풀고
                      contractStep(step).MergeWait() 가 그 값이다 (finalize 가 넘긴 틈 · 받는 일 19 · 20 이 이 디코드 길이다)
                      (2) 형제를 흉내 낸 Dir 이 공유를 쥔 채 merge.wait "300ms" · "900ms" 인 merge 단계 둘 — claim 시각 + wait 에서
                      merge_wait_timeout 으로 끝난다.  허용 오차는 두 값 차이의 절반 (300 ms) 보다 작게 — 두 기다림을 구별할 수 있어야 한다.
                      그 뒤 upper 가 trash · state 가 committed.  기다리는 동안 단계 로그에 쥔 쪽 줄이 있다
     몸통              광고 응답의 몸통과 merge 단계의 실패 길이 함께 돌아도 한 번 (-race) · startMerging 뒤의 몸통은 아무것도 안 함 ·
                      몸통의 옮기기 실패 뒤에도 committed · 잠금을 놓는다
     기동              OnStale 등록이 TryBake 실패와 무관 · committed (버려진 대기 자리 정리) · building · pending (정리 · 그 scratch 의
                      trash · last_attempt abandoned) · merging (재개) · TryBake 실패는 아무것도 안 함
     광고 주기 정리     형제 Dir 이 굽기 잠금을 쥔 채 building · pending 이면 아무것도 안 함 (산 주인) · 그 Dir 을 닫으면 (주인이 죽음)
                      다음 onStale 이 정리 · last_attempt abandoned · committed 를 못 쓰면 retryAt (되물음 1 답 A) · 기록된 대기 자리가
                      이미 없거나 못 옮겨도 committed 를 쓴다 (결정 46 · 53)
     IR 대조 (origin)  origin 이 없는 git (init · fetch) 도 맞음 · url 과 repo_id 는 빈 값 (결정 47)
     재개              12.4 의 넷 · 초안 없음 · 모양이 틀림 · 실패 뒤 retryAt · 같은 오류는 로그 한 번 · onStale 이 잠금 뒤 state 를
                      다시 읽는다 · 이 프로세스가 쥔 굽기 · 재개 중에는 아무것도 안 함 · 재개는 DropBake 를 안 부른다 (표지가 남는다)
     merge-helper     가짜 argv (namespace 없이 같은 바이너리) 로 preflight · apply 의 JSON 한 줄 · 오류 셋의 kind
   internal/merge     checkMounts 표 · 임시 폴더 셋이 같은 마운트면 Preflight 통과 (STATX_MNT_ID 를 읽는다)
   internal/contract  ReasonIRMismatch · head_tags 의 JSON — null (대조 전 · 대조를 못 함) 과 [] (태그 없음) 이 구별된다
   internal/store     어휘 목록에 ir_mismatch — DONE 결과에 실려 와도 경고가 안 남고 reason 이 저장된다 (결정 54)
   cmd/enode          merge-helper 입구가 RunMergeHelper 로 간다
```

- 조각 5 의 나머지 (merge 가 없는 계약 · 이름 규칙 · 이름 겹침이 400) 는 contract-grammar 의 `Validate` 시험이 이미 초록이다 — Code
  Generation 계획이 그 시험 이름을 조각 5 의 목록에 적는다
- 조각 7 (끊겨도 된다) 의 기계 부분은 merge-rules 의 재개 시험 (41 자리 · 기본 `go test`) 이 이미 초록이다

### 8.2 사람이 SunnyVM 에서 도는 시험 (`integration` 태그 · CI 밖)

시험 바이너리를 `unshare --user --map-root-user --map-auto` 뒤에서 다시 실행한다 (lower-state 9.2 · trash 의 모양). 버려도 되는 폴더에서만
돈다 — `/srv/yocto` 는 읽지도 않는다. 이 기계는 `--map-auto` 가 막혀 못 돈다 (lower-state code-summary 7절).

```text
   merge-helper     진짜 unshare 로 preflight · apply — subordinate uid 소유 항목 · 권한 000 디렉터리
   마운트 줄         namespace 안에서 lower 를 bind 별칭으로 걸고 그 경로로 preflight -> Check mount
   끊고 잇기         apply 도중 helper 를 SIGKILL -> merging 남음 -> resume 이 끝낸다 (metadata resumed true)
```

이 결과로 조각 6 · 7 · 8 (사람) 을 대신하지 않는다 (US-14 사람 눈으로).

### 8.3 사람 조각 6 · 7 · 8 — 스크립트와 운영 lower 확인 (답 4 · 완료 조건 8 · US-13)

스크립트는 `scripts/finalize-bake/` 아래 셋이다 (이름은 Code Generation · `slice-4.sh` 모양 — `M` · `T` · `WS` 를 받는다). 셋이 같은 확인을
먼저 한다 — 공용 파일 하나로 둔다 (행렬의 조각 스크립트 칸 · 같은 파일을 두 유닛이 고치지 않는다).

```text
   대상 출력      WS (대상 노드의 워크스페이스 = lower 루트) 의 경로 · 장치 (maj:min) · inode 를 출력한다
   허용 표지      <WS>/.enode-disposable 이 보통 파일로 있어야 돈다.  없으면 운영 lower 로 보고 멈춘다 (답 4)
   같은 lower     GET /v1/nodes 에서 machine 이 이 기계의 hostname 이거나 machine 이 없는 노드마다 ws 를 이 기계에서 stat 한다.
                 장치 · inode 가 WS 와 같은 노드 (bind 별칭 포함) 가 모두 workspace.writes=isolated 를 광고해야 돈다.  키가 없으면
                 「옛 판이 있다」, in-place 면 「native 노드가 있다」 (결정 3-1 · 공유 lower 위에 native 를 두지 않는다) 고 말하고
                 멈춘다 (lower-state 가 넘긴 규칙 · 한 lower 의 노드를 모두 새 판으로 올린 뒤 굽는다)
                 ws 를 stat 하지 못한 노드가 있으면 같은 lower 인지 모르므로 멈춘다
   대상 노드      workspace.writes=isolated 여야 한다.  굽기를 아는 판인지는 광고 키로 알아보지 못한다 — lower-state 판 (0262155)
                 도 isolated 를 광고한다.  사람이 대상 노드의 판을 확인한다고 스크립트가 출력한다
   결과          단계에 lower_changed 가 보이면 「다시 내면 된다」를 출력한다 (lower-state 가 넘긴 일)
```

- **옛 판을 알아보는 근거** — `workspace.writes` 는 `9d7197f` (lower-state 유닛 · 2026-09-27) 가 처음 들였다. `ws` 는 `d0438c2`
  (2026-08-23), `machine` 은 `1b1c9dd` (2026-09-19) 부터 광고한다 (`git log -S` · 2026-09-27 측정). SunnyVM 에 떠 있는 2026-09-22 판
  데몬 둘 (계획 2.4) 은 `machine` · `ws` 를 광고하고 `workspace.writes` 를 안 한다 — 그 데몬의 ws 가 대상 lower 와 같으면 이 확인에 걸린다.
  그 ws 는 읽지 않았다 (계획 2.4 — 떠 있는 노드의 설정을 안 읽었다)
- **못 잡는 것** — 다른 Mediator 에 붙은 같은 lower 의 노드 · 꺼져 있는 노드 · 확인 뒤에 뜬 노드 (`GET /v1/nodes` 는 만료 전의 노드만 준다 ·
  `store/observe.go:167`). 스크립트가 출력하고, 사람이 굽는 동안 그 lower 의 노드를 새로 띄우지 않는다
- **표지는 lower 뿌리의 파일이라 합친 목록 비교의 양쪽에 똑같이 있다** (조각 6 의 ⑥ · 합친 lower 와 merged view). 비교에서
  `.enode-metadata.json` 은 뺀다 — 합치기의 마지막 동작으로 새로 쓰인다
- **조각의 sync 는 lower 뿌리를 비울 것을 기대하지 않는다** — 표지가 있어 빈 lower 도 뿌리가 비어 있지 않다. `git clone <url> .` 은 빈 디렉터리를
  요구하므로 쓰지 않고 `git init` · `git remote add origin <url>` · `git fetch` · `git checkout` 으로 받는다 — origin 을 두어 metadata 의 url 과
  repo_id 가 찬다 (조각 6 이 「칸이 다 있다」를 본다 · 결정 47. 대조는 origin 이 없어도 돈다). 뿌리에 `git clean -x` 를 돌리면 upper 에 whiteout 이 생겨 표지가
  첫 굽기에 사라지고 둘째 굽기 앞의 확인이 멈춘다
- 표지는 lower 와 함께 복사된다 — 버려도 되는 lower 를 복사해 운영 자리에 두면 표지도 따라간다. 스크립트가 경고 한 줄로 알린다

```text
   조각 6 (굽기가 끝까지 돈다)
     장면 2 의 1 ~ 6 (scene-gates.md) — 빈 lower 에서 굽기 A, 형제에 긴 Run 을 낸 뒤 굽기 B.  merge 가 waiting 인 동안
       단계 로그에 쥔 쪽 줄과 남은 시간 (완료 조건 5 · US-12 · US-15) · 형제 draining (GET /v1/nodes)
     합치기 전 merged view 목록 (user namespace 의 ro overlay) 과 합친 뒤 lower 목록 -> diff 가 비어야 한다 (metadata 는 뺀다)
     .enode-metadata.json 의 칸이 다 있다 (결정 3-16) · bake.run · bake.node (US-6) · GET /v1/capabilities 에 ir · repo.built.<이름>
     빌드 하나를 일부러 실패시킨 굽기 -> 합치지 않음 · last_attempt · 뒤 구성은 skipping (답 3) · build 는 DONE 이고 Run FAILED
       (조각의 계약은 runctl example bake 처럼 build · merge 에 produced 조건을 건다 — 그 조건이 판정한다 · 되물음 2 답 B)
     sync 가 IR 에 닿지 않는 굽기 -> build 는 DONE · reason ir_mismatch · build 칸의 head · head_tags · manifest 없음 · 합치지 않음 ·
       merge 는 합칠 것 없음 · Run FAILED (produced 조건 · 물음 1 답 B) (완료 조건 10 · US-19) — requirements.md 6절이 바꾼 줄
     처음 확인 — 세션 안 git 의 소유 확인 (safe.directory) · rootfs 의 git 과 bash (되물음 3 답 A) (계획 2.6)
   조각 7 (끊겨도 된다)
     같은 lower 의 다른 노드를 멈춘 채 merge 단계를 여러 자리에서 SIGKILL -> 다른 노드를 시작 -> state committed ·
       metadata 의 resumed true 와 원래 Run · 합친 목록이 한 번에 끝낸 것과 같다 (scene-gates.md 조각 7 그대로)
     둘째 경우 — 형제를 띄워 둔 채 SIGKILL -> 형제가 한 광고 주기 안에 잇는다 (답 2 · 살아 있는 동안)
   조각 8 (배타와 대기)
     굽기 중 두 번째 굽기 -> 곧바로 FAILED bake_in_progress (US-16)
     굽기 노드를 pending 에서 SIGKILL -> 다른 노드가 낡은 상태를 정리하고 drain 이 풀린다 (scene-gates.md:53) — 떠 있는 형제의 광고
       주기에서, 다시 띄우지 않고 (되물음 1 답 A).  drain 이 풀리기까지 두 광고 주기 (기본 120초) 안이다 — 정리를 연 광고는 이미
       drain 을 싣고 나간다
     merge.wait 1m 굽기 + 형제에 5분 Run -> merge_wait_timeout · upper trash · committed · drain 풀림 ·
       Record 에서 build DONE 과 merge 의 reason 이 따로 (완료 조건 9 · US-17)
     bind 별칭으로 같은 lower 를 가리키는 두 노드가 같은 상태 자리
     env check 의 not ready 사유가 어긋난 것을 이름으로 — 다른 uid 의 노드 (lower.owner_uid) · 워크스페이스와 다른 filesystem 이나 다른
       마운트 (bind 별칭 포함) 에 둔 scratch (binding.scratch_filesystem) · lower.json 신원 (lower.identity) (requirements.md 6절 조각 8 ·
       lower-state 의 점검 셋 · 받는 일 46)
     형제 노드가 임대를 받은 뒤 첫 단계 전에 합치기가 lower 를 바꾸지 않는다 — 형제가 drain 을 받아 공유를 놓은 뒤 늦게 매칭된
       Run 은 lower_changed 로 돌려받는다 (requirements.md 6절 조각 8 · lower-state 규칙 5.4)
```

---

## 9. 순수 함수와 커버리지

```text
   internal/enode     84.5% (lower-state 뒤).  규칙은 bakerule.go 의 순수 함수로 떼고, 흐름은 가짜 세션과 임시 lower 의 진짜 잠금으로
                      덮는다.  merge-helper 를 여는 자리는 함수 값이라 시험이 namespace 없는 명령으로 바꿔 끼운다 (trash-helper 선례)
   cmd/enode          80.5% — 하한에 가깝다.  조립은 StartBaker 하나 · main.go 에는 부르는 줄 넷
   internal/merge · internal/contract · internal/store      한두 줄씩과 그 시험
```

`_linux.go` 의 unshare 갈래 중 기본 `go test` 가 못 밟는 것은 8.2 가 덮는다. 80% 를 넘지 못하면 Code Generation 이 호출 자리를 함수
값으로 떼어 실패를 끼운다 (merge-rules · lower-state 와 같은 방법).

---

## 10. 파일 행렬 밖의 자리

행렬(`unit-of-work-file-matrix.md` 1절의 ⑦ 열 — bake 유닛이 고치는 칸)이 이 유닛에 준 파일은 `runc_overlay_linux.go` · `claim.go` ·
굽기 build · merge 단계와 merge-helper 의 새 파일 · `cmd/enode/main.go` · 조각 스크립트다.

```text
   internal/contract/result.go      ReasonIRMismatch · BuildManifest.HeadTags · IR 칸의 주석 (물음 1 답 B) — 노드와 Mediator 가 함께 가져온다
   internal/store/claim.go          reason 어휘 목록 (:962 ~ :963) 에 ir_mismatch 한 줄 (물음 1 답 B · 결정 54) — Mediator
   internal/merge/merge.go          CheckMount (lower-state 답 6 · 받는 일 34) — merge-rules 유닛의 파일
   internal/merge/decide.go         checkMounts (checkDevices 옆)
   internal/merge/merge_linux.go    resolve 가 STATX_MNT_ID 를 읽고 댄다
   internal/enode/finalize.go       contractStep 이 굽기 칸 넷을 옮긴다 (finalize 가 넘긴 일 · 받는 일 19) — finalize 유닛의 파일
   internal/enode/lowerguard.go     OnStale · Dir (답 2 · 되물음 1 답 A) — lower-state 유닛의 파일
   internal/enode/workspace.go      isolated 노드의 Prepare 는 호스트 git · repo 를 돌리지 않는다 (ADR-072 결정 3 · CG 물음 3 답 A — Code
                                    Generation 이 더함)
   internal/enode/diff.go ·         isolated 노드의 Finalize 는 workspace.diff 를 세션 안 (준비된 rootfs 의 git · repo) 에서 만든다 ·
   runc_overlay_linux.go 의 finalize  helper 는 merged view 에 git · repo 를 돌리지 않는다 — finalize 유닛의 수확을 고친다 (CG 물음 5 답 —
   · runtime.go 의 주석              Code Generation 이 더함)
   위 파일들의 시험                    표 시험 한두 줄씩
```

- `runtime.go` · `runc_overlay_other.go` 는 안 바뀐다 — `Keep` 은 이미 있고 native 의 Close 는 Keep 을 안 본다. Code Generation 이 확인한다
- `internal/panel/boundary_test.go` 는 안 바뀐다 — `internal/enode` 가 `internal/merge` 를 가져오는 것은 경계 표의 금지가 아니다
  (`cmd/mediator` 만 막는다)

---

## 11. 답 대 보기와 받는 일의 추적

### 11.1 답 대 보기 (Step 5)

```text
   짝                              본 것                                                           판단
   1 과 2                          IR 어긋남은 build 에서 끝나 merging 에 닿지 않는다 · merge 는 합칠 것 없음  부딪침 없음
   1 과 contract-grammar 흐름 3절  「merge 는 needs 를 따라 와서 합칠 것이 없다」가 명령 실패와 IR     부딪침 없음 (물음 1 답 B — 처음 답 A
                                   어긋남 두 갈래에 그대로다                                           의 넘김 안을 거뒀다)
   1 과 I3                         ir_mismatch 는 완주 (DONE) 에 붙는다.  manifest 가 없으므로         부딪침 없음 (결정 54)
                                   produced 조건이 판정한다 (internal/contract/bake.go:36 ~ :37)
   1 과 Mediator 의 reason         DONE 에 reason 이 실린 적이 없다.  완주는 error 만 본다 (api.go:432)  부딪침 없음 — 어휘 줄은 경고를
                                   reason 은 완주와 무관하게 저장되고 어휘 밖이면 경고만 (store/claim.go:962)  막으려 둔다 (결정 54)
   1 과 step-phase 의 더하기만 규칙   head_tags · ir_mismatch 는 더하기다.  IR 칸은 주석만 고친다         부딪침 없음
   2 와 계획 3.1                    merging 에서 committed 로 가는 길은 합치기가 끝나는 것 하나 —       부딪침 없음
                                  답 2 가 그 길을 지키며 「언제 잇나」만 넓혔다
   2 와 계획 3.7 · 3.11 · 3.12       물음 2 로 비워 둔 칸이 채워졌다.  merging 뒤의 모든 오류를 Apply 와   설계로 닫음 (결정 2)
                                  같은 길로 두는 것은 답 2 를 넓힌 것이다
   2 와 lower-state 의 HoldBake ·    DropBake 가 표지를 합친 뒤로 새로 적어 늦은 임대를 통과시킨다       설계로 닫음 (결정 1 · 27 — 재개는
   DropBake                        (QA S6 이 근거를 고쳤다)                                          둘 다 부르지 않는다)
   2 와 조각 7 의 글자              「다른 노드가 시작 때 재개」 — 살아 있는 형제가 먼저 이을 수 있다      스크립트가 다른 노드를 멈춘 채
                                                                                                 끊는다 · 둘째 경우를 더한다
   2 와 계획 3.13                   광고 주기는 merging 만 잇는다.  주인이 죽은 pending 은 형제를 모두    되물음 1 답 A (QA S1) — 광고
                                  drain 에 묶고 다음 굽기도 매칭되지 않는다                            주기가 셋 모두 기동과 같은 표로
   3 과 계획 3.5 · 3.14             exited 는 마지막으로 돈 명령 · last_attempt 의 builds 는 돈 것      부딪침 없음
   3 과 1                          IR 이 어긋나면 builds 가 하나도 안 돈다 — 구성마다 skipping 줄      부딪침 없음 (결정 55)
   4 와 lower-state 가 넘긴 규칙      옛 판 노드는 workspace.writes 가 없다 — machine · ws 는 그 전부터   조건을 isolated 로 좁혔다 (QA X4)
                                  광고한다.  native 는 in-place 로 통과했다
   4 와 조각 6 의 목록 비교          표지는 양쪽에 있다 · metadata 는 뺀다 · 표지가 있어 뿌리가 비지 않는다   스크립트의 규칙으로 닫음 (8.3)
```

처음 판에서 되물음 파일이 없었다. QA 검수가 사용자가 정할 다섯 자리를 찾아 되물음 파일을 냈다 (QA S1 · S4 · S8 · S9 · 참고의 첫 줄).
답은 2026-09-27T10:38:41Z 에 다섯 모두 A 다 (「권장대로」). 계획 5절 첫 체크박스의 「되물음 파일 없음」은 그때의 판단이다 — 계획은 답을 받은
기록이라 두고 여기에 적는다.

```text
   답 대 보기 (되물음)                본 것                                                                     판단
   되물음 1 답 A 와 규칙 7.1 재시작 줄  정리하는 쪽이 기동만이 아니라 형제의 광고 주기일 수 있다 —              문장을 넓혔다 (결정 43)
                                      「its start-up cleanup」이 거짓이 된다
   되물음 1 답 A 와 산 굽기           주인이 building 부터 committed 까지 잠금을 놓지 않으므로 TryBake 가       부딪침 없음
                                      안 된다 — 형제가 산 굽기를 치울 길이 없다
   되물음 1 답 A 와 되물음 4 답 A     build claim 이 낡은 building · pending 을 만나면 정리하고 진행 ·          부딪침 없음
                                      merging 이면 재개 · bake_in_progress.  광고 주기가 먼저 했으면 committed
   되물음 2 답 B 와 S3 · 결정 24      몸통이 held 를 비우므로 명령 실패 뒤의 merge 는 합칠 것 없음 줄로 가고    부딪침 없음 — 재시작 줄이 합칠 것
                                      DONE 이다.  재시작은 pending 갈래 (주인 · pending 을 치운 abandoned)      없음보다 앞에 있다 (결정 50 · 52)
                                      만 잡아 FAILED 다.  몸통이 committed 를 못 쓴 building 은 합칠 것 없음
   되물음 2 답 B 와 I3                합칠 것 없음은 완주 (DONE) · 판정은 계약의 조건.  재시작 줄과 7.1 의      부딪침 없음 — FAILED 둘은 판정이 아니라
                                      어긋남 줄의 FAILED 는 미완주 (INVARIANTS.md:292)                          입력 · 전제를 잃은 미완주
   되물음 2 답 B 와 물음 1 답 B       IR 어긋남 뒤의 merge 도 합칠 것 없음 줄로 간다 — build 는 DONE ·          부딪침 없음 (결정 54)
                                      reason ir_mismatch · manifest 없음.  판정은 produced 조건
   되물음 2 답 B 와 S8                판정 조건 없는 굽기 계약은 sync 가 실패해도 SUCCEEDED 가 될 수 있다       정본 결정대로 — 예시 계약이 조건을 걸고
                                                                                                                lint 가 경고한다 (규칙 6절)
   되물음 3 답 A 와 IR 대조           노드의 고정 한 줄은 sh 로 남는다 · bash 확인은 sync 앞                    설계로 닫음 (결정 41)
   되물음 5 답 A 와 되물음 4 답 A     (17) 의 문장은 되물음 4 가 A 일 때의 글자다 — 둘 다 A 라 그대로           부딪침 없음
```

### 11.2 받는 일과 받은 자리

계획 1절의 번호다. 55 · 56 은 계획이 빠뜨린 둘이다 (QA 참고). 「넘김」은 이 단계가 받지 않은 몫이고 12절에 까닭을 적었다.

| # | 받는 일 (줄임) | 받은 자리 |
|---|---|---|
| 1 | sync 와 builds 를 셸로 | 규칙 3절 (bash -c · 되물음 3 답 A) · 흐름 2절 |
| 2 | IR 을 `ENODE_IR` 로 넘긴다 | 규칙 3절 (환경) · 엔티티 5절 |
| 3 | sync 뒤 HEAD 와 IR 대조 | 규칙 4절 · 엔티티 5절 |
| 4 | `.repo` 없는 poky git | 규칙 4절 (git 모양 줄) |
| 5 | 성공했을 때만 manifest | 규칙 15절 · 엔티티 6절 |
| 6 | 합쳤을 때만 merged | 규칙 15절 · 7절 |
| 7 | merge.wait 을 센다 | 엔티티 1절 · 규칙 7절 (마감) · 흐름 8.1 (조각 5 · 왕복) |
| 8 | native 거절 | 규칙 1절 |
| 9 | 병합 뒤의 창 (empty argv) | 규칙 1절 |
| 10 | build 실패 뒤 merge 가 알리는 법 | 규칙 6절 · 7.1절 (merge DONE · merged 없음 · 단계 로그 · 되물음 2 답 B) |
| 11 | Grammar 의 굽기 절 | 규칙 1절 |
| 12 | build 의 exited | 규칙 5절 |
| 13 | merge 는 exited 를 안 보냄 | 규칙 5절 |
| 14 | result 의 build · merge 칸 (contract 타입) | 엔티티 6절 · 규칙 15절 |
| 15 | 실패한 build 에도 build 칸 | 규칙 15절 · 6절 |
| 16 | result.go 는 더하기만 | 엔티티 6절 |
| 17 | merge 단계가 합치기를 알리는 법 | 규칙 7절 · 7.1절 · 15절 (되물음 2 답 B) |
| 18 | build 의 종료 보고 (`Client.Exited` · `startExitReport`) | 규칙 5절 |
| 19 | build 의 두 예산 (`contractStep`) | 엔티티 1절 · 규칙 5절 · 흐름 8.1 (왕복 시험) |
| 20 | 노드가 받는 칸 넷 | 엔티티 1절 · 흐름 8.1 (왕복 시험) |
| 21 | `Keep.Upper` 에 대기 자리 | 규칙 3절 (닫기) · 엔티티 4절 |
| 22 | merge_wait_timeout 의 upper 는 `Trash.Move` | 규칙 8.5 · 11절 |
| 23 | 합치기의 lower 쪽 항목도 `Trash.Move` | 규칙 9절 |
| 24 | helper 안에서 Preflight 뒤 Apply | 규칙 9절 · 7절 · 12.3 |
| 25 | `Discard` 를 `Trash.Move` 로 · trash 를 먼저 만든다 | 규칙 9절 · 7절 (6) |
| 26 | helper 는 namespace 안의 root | 규칙 9절 · 엔티티 8절 |
| 27 | `PreflightError.Check` 마다 상태 | 규칙 10절 |
| 28 | Preflight 의 읽기 실패는 다시 해 볼 일 | 규칙 10절 · 13절 (재개에서 받는다) |
| 29 | 재개 때마다 되풀이되는 오류 | 규칙 13절 (답 2) |
| 30 | Apply 뒤 빈 upper 뿌리를 치운다 | 규칙 7절 (11) · 9절 · 12.4 |
| 31 | Result 를 로그와 metadata 에 · 재개면 더한다 | 규칙 7절 · 엔티티 6절 (재개의 셈은 노드 로그 · 더할 보고가 없다) |
| 32 | 한 합치기의 trash 항목을 모을지 | 규칙 9절 (항목마다) |
| 33 | 조각 7 전체 | 흐름 8.3 · 8.2 · 사람 실행은 이 유닛의 Code Generation 끝 · PR 전 (12절) |
| 34 | Preflight 에 마운트 확인 | 규칙 10절 · 엔티티 9절 |
| 35 | merge 와 재개의 흐름 · watch 로그 | 규칙 8절 · 흐름 3절 · 6절 |
| 36 | watch 는 처음과 every 마다 · `Waiting.Unnamed` | 규칙 8.2 · 8.3 |
| 37 | `ForeignMounts` 를 배타 뒤에 · 찾으면 놓고 다시 | 규칙 8.4 |
| 38 | `ForeignMounts` 가 오류면 | 규칙 8.4 |
| 39 | state.json 을 쓰는 때와 차례 | 규칙 2절 |
| 40 | HoldBake · DropBake 의 차례 | 규칙 11절 · 엔티티 3절 |
| 41 | 치우는 몸통 · building 은 build 단계 끝에서 | 규칙 11절 · 엔티티 3절 · 흐름 4절 |
| 42 | `WriteMetadata` 를 부르는 자리 | 규칙 7절 |
| 43 | 옛 판 노드의 장면 · 새 판 규칙과 스크립트 확인 | 흐름 8.3 (답 4) |
| 44 | lower-state 가 측정 못 한 셋 | 넘김 — 조각 6 · 8 에서 볼 수 있는 것과 측정 못 하는 잔여로 나눔 (12절) |
| 45 | `lower_changed` 가 굽기 조각에서 보이면 | 흐름 8.3 |
| 46 | 조각 8 의 준비도 점검 부분 | 흐름 8.3 (조각 8) · 사람 실행은 이 유닛의 Code Generation 끝 · PR 전 (12절) |
| 47 | build · merge 의 실패 경로와 상태 되돌림 | 규칙 6 · 7 · 10 · 11절 (held 가 있는 동안의 오류 길 · 결정 44) · 흐름 7절 |
| 48 | IR 대조의 환경 변수 이름과 원인 코드 | 엔티티 5 · 6절 (`ENODE_IR` · `ir_mismatch`) |
| 49 | 대기 로그의 모양과 상한 시각의 시계 | 규칙 8절 |
| 50 | 재개가 굽기 잠금을 먼저 잡은 하나에게 | 규칙 12.2 · 13절 |
| 51 | 시작 전 확인이 어긋났을 때 상태 | 규칙 10절 |
| 52 | 소유자의 굽기 Run 취소 | 규칙 11절 (pending 에서 굽기를 쥔 프로세스가 살아 있을 때 통한다 · 그 프로세스가 죽었으면 형제의 광고 주기가 치운다 — 되물음 1 답 A) |
| 53 | 같은 커밋에 IR 태그 둘 | 규칙 4절 (판정) |
| 54 | 합치기 조각 스크립트의 운영 lower 확인 | 흐름 8.3 (답 4) |
| 55 | build · merge 단계의 수확 (finalize 계획 0절 · `finalize-functional-design-plan.md:38`) | 규칙 3절 (Finalize 는 effect prepare 의 몫 · merge 는 Finalize 없음) |
| 56 | 실패의 기록은 result 진단과 last_attempt (contract-grammar 흐름 3절) | 규칙 15절 (실패의 기록 셋) |

### 11.3 계획 3절의 열여섯과 답 넷이 받은 자리

```text
   3.1  전이와 쓰는 때                  규칙 2절                  3.9   기다림 · 로그 · 시계 · 그물       규칙 8절
   3.2  단계 분기와 거절                규칙 1절 · 엔티티 1 · 2절    3.10  merge-helper 와 trash            규칙 9절 · 엔티티 8절
   3.3  build 의 명령                   규칙 3절                  3.11  시작 전 확인 · 마운트 줄           규칙 10절 · 엔티티 9절
   3.4  IR 대조                        규칙 4절 · 엔티티 5절        3.12  HoldBake · DropBake · 몸통        규칙 11절 · 엔티티 3절 · 흐름 4절
   3.5  exited 와 예산                  규칙 5절                  3.13  기동 · 낡은 상태 정리 · 재개        규칙 12 · 13절 · 흐름 5 · 6절 · 되물음 1 답 A
   3.6  대기 자리와 초안                 규칙 3 · 3.1절 · 엔티티 4절  3.14  last_attempt                     규칙 14절 · 엔티티 7절
   3.7  합칠 upper 를 못 남기면          규칙 6절 · 흐름 7절          3.15  결과 칸                           규칙 15절 · 엔티티 6절
   3.8  merge 단계                     규칙 7절 · 흐름 3절 · 되물음 2 답 B  3.16  조각                        흐름 8절
   답 1  IR 어긋남의 보고 (B)            규칙 원칙 · 3 · 4 · 6 · 7.1 · 16.2절 · 엔티티 5 · 6절 · 흐름 2 · 7절
   답 2  재개의 때                      규칙 7 · 10 · 12 · 13절 · 엔티티 10절 · 흐름 6절 · 되물음 1 · 4 답 A
   되물음 1 ~ 5 (답 A)                  1 규칙 12 · 13절 · 흐름 6절 · 2 규칙 6 · 7.1절 · 3 규칙 3절 · 흐름 2절 · 4 규칙 3 · 13절 ·
                                      5 규칙 2 · 7 · 14 · 15 · 16.1절 · 엔티티 6절
   답 3  실패 뒤 builds                 규칙 3 · 6 · 14 · 16.2절
   답 4  운영 lower 확인                흐름 8.3
```

**계획과 달라진 자리는 열둘이다** — 셋째부터는 QA 가 찾았다 (아홉째부터는 재검).

```text
   1  previous_ir 을 합칠 때가 아니라 build 가 굽기 잠금을 잡은 뒤 읽어 초안에 둔다      계획 3.8 · 3.13     엔티티 4절
   2  초안을 쓰기 전에 pinned 의 sha256 을 읽는다                                  계획 3.6            흐름 2절
   3  exited 는 명령이 임대로 끊기거나 데몬이 멈추면 안 보낸다                           계획 3.5            규칙 5절
   4  pending 을 쓰기 전의 노드 쪽 오류 (닫기 · 초안 · pending 쓰기) 와 Finalize 예산 줄    계획 3.7            규칙 6절
   5  DropBake 를 HoldBake 전의 길 (building 실패) 에서도 부른다                         계획 3.12           규칙 11절
   6  단계 로그에 host 경로를 쓰지 않는다                                            계획 3.9            규칙 16.2
   7  merge 의 정리 차례 — committed 를 쓴 뒤 대기 자리를 옮긴다 (QA S5)                   계획 3.8            규칙 7절 · 12.4
   8  merge claim 의 재시작 줄 (QA S2)                                             계획 3.8            규칙 7.1
   9  명령의 셸 — sh -c 가 아니라 bash -c · sync 앞의 bash 확인 (되물음 3 답 A)          계획 3.3            규칙 3절
   10 재개의 「끝났나」 — metadata 의 bake.run 만이 아니라 초안의 synced_at · head 까지      계획 3.13           규칙 12.4
   11 전이의 차례 — 「대기 자리를 trash 로 옮긴 뒤 committed」가 아니라 committed 뒤 옮김     계획 3.1            규칙 2 · 7절
   12 재개의 차례 — 같은 뒤집음 (대기 자리 -> committed 에서 committed -> 대기 자리)         계획 3.13           규칙 12.3
```

계획의 사실 줄 하나도 틀렸다 — 계획 2.5 (`계획:208`) 의 「upper · merged 는 helper 의 마운트 namespace 안에만 있다」에서 upper 는 호스트의
보통 디렉터리다. 계획은 답을 받은 기록이라 두고, 규칙 4절이 고친 글자를 적는다.

계획 3.13 의 「그 자리가 있는 scratch 의 trash」는 처음 판의 규칙 12.2 에서 빠졌다가 고쳤다 (규칙 9 · 11 · 12.2절). 나머지는 계획이 비워
둔 자리를 채운 것이다 — 합치면 last_attempt 를 지운다 · metadata 를 쓴 뒤의 정리 오류는 DONE · build 의 $OUT 은 노드의 것 · url 의 비밀번호를
지운다 (앞의 셋은 되물음 5 답 A, 넷째는 audit 의 「답에 없던 설계 결정」). 되물음의 답이 계획을 더 바꾼 자리는 하나다 — 광고 주기도 낡은
building · pending 을 치운다 (계획 3.13 · 되물음 1 답 A). 되물음 2 는 A 로 한 번 계획 3.8 을 바꿨다가 B 로 다시 답해 「합칠 것 없음은 DONE」
으로 돌아갔다 (I3 · QA 재검 D1). 물음 1 도 A 로 한 번 받았다가 2026-09-27T12:54:46Z 에 B 로 다시 답했다 — IR 어긋남은 build 의 DONE 에
reason `ir_mismatch` 를 싣고 판정을 계약의 produced 조건에 맡긴다 (I3 · 결정 54 · 55). 계획의 답 칸이 B 라 계획과 달라진 자리가 아니다.

---

## 12. 이 단계가 받지 않고 넘기는 것

```text
   checkpoint      실패한 굽기의 upper 를 spool 로 받을지.  온전한 upper 를 trash 로 보내는 길이 여럿이다 — build 의 명령 실패 ·
                   IR 어긋남 (Close(Keep{})) 과 몸통이 부르는 대기 상한 · Preflight 어긋남 · 예산 초과 · 임대가 사라짐.  on-failure 로
                   보존하면 그 자리를 Keep{Upper: spool} (build 끝) 이나 몸통의 옮길 곳 (대기 자리 -> spool) 으로 바꾼다.  merging 뒤의
                   upper 는 반쯤 합쳐진 남은 일이라 보존 대상이 아니다.  보존하면 답 3 (첫 실패에서 멈춤) 의 근거 — 「upper 가 trash 로
                   간다」 — 를 다시 본다 (규칙 3절)
   이 유닛의 NFR    보안 — 계약의 명령과 IR 대조가 격리 runtime 안에서만 돈다 (셸이 세션 안) · 초안 0600 · url 비밀번호 ·
                   merge-helper 가 namespace 안의 root.  성능 — 형제가 기다리는 시간은 형제 Run 이 정한다 · 재개 재시도의 비용
                   (노드 N 개면 N 배) · 대기 로그의 줄 수 (4시간 48 줄)
   Code Generation 조각 스크립트의 이름과 명령 · 조각 5 의 시험 이름 목록 (contract-grammar 의 400 시험 포함) · 로그 문구를 굳힌다 ·
                   runtime.go · runc_overlay_other.go 를 안 바꿔도 되는지
   이 유닛의 Code Generation 끝 · PR 전 (사람 · SunnyVM)
                   조각 6 · 7 · 8 을 사람이 돈다 — 이 유닛의 병합 게이트다 (unit-of-work.md 0절 · CONVENTIONS 3.3 · US-14).  그 안에서 처음
                   확인하는 것 — 세션 안 git 의 소유 확인 · rootfs 의 git 과 bash (되물음 3 답 A) · 합치는 중 SIGKILL 과 재개 · 조각 8 의 준비도
                   점검 부분 (받는 일 46) · 옛 판 노드가 있는 lower (받는 일 44 의 셋째)
   측정 못 함 · 잔여 · 진행자
                   조각 6 ~ 8 로도 닿지 않는 것 — ext4 밖 filesystem 의 fsid 가 재부팅에 같은지 (SunnyVM 의 시험 lower 는 ext4 다) ·
                   전원이 나간 뒤 state.json (받는 일 44 의 둘) · repo init -b 뒤 manifests 의 로컬 태그 (조각의 lower 는 poky git 이고
                   SunnyVM 호스트에 repo 가 없다 · 계획 2.6).  진행자가 잔여로 적는다
   진행자           팩 decisions.md · features.md (13.2) · 정본 되돌림 (13.1) · 사람의 수단이 state.json 손 고침뿐인 두 갈래 (초안이 없음 ·
                   pending_upper 의 모양이 틀림 — 규칙 12.4 · 12.2) 를 운영 문서에 적을지
```

---

## 13. 정본과 회차 문서에 되돌려 올리는 것

### 13.1 정본 — 진행자가 `enode-design` 에 올린다 (이 단계는 정본을 고치지 않는다)

```text
   ADR-077 §2      builds 는 첫 실패에서 멈춘다 (답 3).  sync 가 실패하면 builds 를 안 돈다.  build 의 $OUT 은 노드의 것이다 (manifest
                   하나 · 되물음 5 답 A).  sync · builds 는 bash -c 한 줄로 돌고 bash 가 없으면 sync 전에 멈춘다 (되물음 3 답 A).
                   「sync 나 빌드 하나라도 0 이 아니면 build 단계가
                   실패」는 판정의 실패다 — 단계는 DONE (INVARIANTS.md:292 가 이미 말한다 · 되돌림이 아니라 글자 맞춤).
                   합쳐 committed 로 가면 last_attempt 를 지운다 (되물음 5 답 A 의 (8) · last_attempt 의 자리는 §2 와 §10 의 장면 표다)
   ADR-077 §5      IR 대조 — 계약의 IR 태그가 가리키는 커밋이 HEAD 와 같으면 맞다 (HEAD 에 다른 태그가 함께 있어도).  보는 git 은
                   .repo/manifests, 없으면 워크스페이스 뿌리의 .git (정본의 IR 문장 :227 ~ :230 은 .repo/manifests 만 적는다 · 이 단계의
                   FR-9 고침).  세션 안에서 돈다.  어긋나면 builds 를 안 돌리고 manifest 를 안 낸 완주 — DONE 에 reason
                   ir_mismatch · 문장은 단계 로그 (물음 1 답 B · 결정 54 · 55).
                   대기 자리 <scratch>/pending/<lower 키>/<이름>/{upper, bake.json} · pending_upper 는 building 때부터 · previous_ir 은
                   build 가 굽기 잠금을 잡은 뒤 읽는다 · url 의 비밀번호를 지운다 ·
                   재개의 bake.node 는 합친 노드가 아니라 구운 노드다 (예시만으로는 구별되지 않는다)
   ADR-077 §6      대기 로그 (노드 시계의 마감 · 남은 시간 · 쥔 사람 · 바뀔 때와 5분마다) · 그물이 찾으면 60초 뒤 다시 ·
                   merge 가 합칠 것이 없으면 DONE · merged 없음 (되물음 2 답 B — 판정은 계약의 조건) · metadata 를 쓴 뒤의 정리 오류는 DONE
                   (되물음 5 답 A) ·
                   정리 차례 — committed 뒤에
                   대기 자리를 옮긴다.  prepare 를 담은 Run 도 building 동안 후보로 공유를 쥔다 — HoldBake 가 놓는다 (결정 3-9 와 다름)
   ADR-077 §7      (답 2 · 되돌림) 「주인이 없으면 낡은 상태 정리를 먼저 하고 진행한다」를 merging 갈래에서 바꾼다 — 재개를 배경에 열고 새
                   굽기는 bake_in_progress (되물음 4 답 A).  「이 정리와 재개는 … 시작할 때 한다」(:289) 에 살아 있는 동안을 더한다 —
                   광고 주기에 TryBake 가 되면 기동과 같은 표로 building · pending 은 정리하고 merging 은 재개한다 (답 2 · 되물음 1 답 A) ·
                   실패하면 그 노드는 10분 뒤.
                   merge 단계의 Apply 오류는 FAILED (원인 코드 없음) 로 보고하고 두 잠금을 놓는다 · merging 뒤의 모든 오류가 같은 길.
                   재개는 끝났는지 metadata 와 초안 (bake.run · synced_at · head) 으로 먼저 본다.  재개는 HoldBake · DropBake 를 안 부른다.
                   원래 Run 이 FAILED 인 까닭이 재시작 (ADR-030) 만이 아니다 — Apply 오류도 있다 (:290 ~ :291 「원래 Run 은 ADR-030 에 따라
                   재시작 실패로 닫힌다」)
   ADR-077 §10     구현 순서 6 「시작할 때 낡은 상태를 정리하고 merging 을 재개한다」(:382) — 시작할 때와 광고 주기 (되물음 1 답 A)
   ADR-077 §10     검증 장면 표의 「manifest HEAD 에 IR 태그가 없음 — ir 은 null · 광고하지 않음」(:388) — IR 이 계약 값이 된 뒤로
                   「sync 가 IR 에 닿지 않음 — build DONE · reason ir_mismatch · head_tags · manifest 없음 · merge 는 합칠 것 없음 ·
                   Run 은 produced 조건이 FAILED」 (결정 3-26 의 되돌림에 이 줄이 빠졌다 · 물음 1 답 B)
   INVARIANTS      :293 「명령 단계의 셸 — argv 배열」 — 굽기 명령은 그 줄의 예외다.  노드의 래퍼 bash -c 가 계약의 sync · builds
                   문자열을 돌린다 (linux runc-overlay 만 · 준비 환경의 rootfs 에 bash 가 있어야 한다 · 되물음 3 답 A · 2026-09-27T11:59:51Z 유지)
   run-contract    :1146 「계약에 셸을 여는 것이 아니라 노드가 래퍼를 갖는다」 — 되물음 3 답 A 의 bash -c 가 그 래퍼다.  :1068 의 미정 항목
                   「명령 단계의 셸 — sh -c 인가 argv 배열인가」를 굽기 단계에 한해 닫는다 (argv 는 [bash, -c, <계약의 문자열>]).  ADR-077 §2
                   의 셸 문자열과 이렇게 맞춘다 (QA 재검 D2)
   run-contract    원인 코드 ir_mismatch · build 결과의 head_tags · 명령 실패와 IR 어긋남은 DONE (명령 실패는 exit · IR 어긋남은
                   reason ir_mismatch 와 단계 로그의 문장 · 물음 1 답 B) · 노드 쪽 오류는 FAILED ·
                   merge 가 합칠 것이 없으면 DONE · merged 없음 (되물음 2 답 B) · merge 가 멈추면 FAILED (원인 코드 없음) · 재시작 두 문장 ·
                   bake_in_progress 의 두 문장 ·
                   sync · builds 는 bash 로 돈다 (되물음 3 답 A)
   mediator-api    result 의 reason 어휘를 적는 절이 아직 없다 — 새로 두고 ir_mismatch 를 싣는다 · reason 은 DONE 에도 실린다 (완주는
                   error 만 본다 · ir_mismatch 가 처음 · 결정 54) · build.head_tags 의 null (대조 전 · 대조를 못 함) 과 [] (태그 없음)
```

- **되돌림이 아닌 것** — ADR-077:133 은 이미 「한 마운트포인트 안」을 요구한다. merge 의 마운트 줄은 코드가 그것을 지키게 하는 것이다.
  INVARIANTS.md:292 는 이미 「result.error 가 비면 완주」다 — build 단계의 명령 실패와 merge 의 합칠 것 없음이 DONE 인 것은 정본 그대로다.
  INVARIANTS.md:148 (I3 — Run 의 성공과 실패는 계약에 선언된 조건으로만) 은 지킨다 — 합칠 것 없음을 DONE 으로 두고 Run 의 판정을 계약의
  조건에 맡긴다 (되물음 2 답 B · QA 재검 D1). 재시작 줄과 7.1 의 어긋남 줄의 FAILED 는 판정이 아니라 미완주다 (규칙 7.1절). IR 어긋남도
  I3 을 지킨다 — ir_mismatch 는 완주 (DONE) 에 붙고 Run 은 produced ["manifest"] 조건이 판정한다 (`internal/contract/bake.go:36` ~ `:37` ·
  규칙 4절 · 물음 1 답 B)
- **정본을 따르게 하는 것 (Code Generation 이 더함)** — ADR-072 결정 3 (§5 「오버레이 노드에서 Prepare 는 윗 층을 버리는 것이다」 · §5.2 ·
  §8) 을 이 회차의 팩 · Inception · FD 는 받지 않았다 (팩은 ADR-072 의 §6.4 · §8 · §9 만 받았다). 오늘 Prepare 는 runtime 을 안 보고 lower
  뿌리에서 호스트 reset · clean 을 돌린다 — Code Generation 계획 4절 33번이 isolated 노드에서 그것을 멈춘다 (CG 물음 3 답 A ·
  2026-09-29T13:53:38Z). 되돌림은 없다
- **굽기 전부터 있던 격리의 구멍 (Code Generation 이 더함)** — isolated 노드의 edit 단계 (agent 포함) 가 upper 에 `.git/config` 를 쓰면
  Finalize 의 workspace.diff 때 runtime helper (컨테이너 밖 · 노드 uid) 의 호스트 git 이 그 설정 (core.fsmonitor · filter.<이름>.clean) 을
  실행했다 (진행자 측정 git 2.39.2 · 2026-09-29T14:13:46Z). 굽기는 lower 에 남은 설정을 뒤 Run 의 diff 가 읽게 해 길을 넓힌다. Code Generation
  계획 4절 34번이 diff 를 세션 안으로 옮긴다 (CG 물음 5 답 · 2026-09-29T15:05:21Z). **정본과 어긋나지 않는다** — execution-environment.md
  §10.3 (:597 ~ :608) 과 ADR-073 §6 은 diff 를 Close 전에 merged view 에서 만든다고만 적었고 그 git 이 어디서 도는지 적지 않았다. 세션 안의
  작업 폴더가 그 merged view 다. 「격리 노드에서 호스트 쪽 helper 는 워크스페이스가 적은 설정을 실행하는 명령을 돌리지 않는다」 는 정본에
  없는 규칙이라 보탤지 진행자가 정한다 (되돌림이 아니라 보탬)

### 13.2 팩 — 진행자가 정한다 (`requirements/finalize-bake/`)

```text
   decisions.md 3-9    「prepare 를 담은 Run 은 공유 잠금을 잡지 않는다」 — building 동안 후보로 쥐고 HoldBake 가 놓는다 (lower-state 코드 ·
                       계획 2.3)
   decisions.md 3-12   「주인이 없으면 낡은 상태를 정리한다」 — merging 이면 재개를 열고 bake_in_progress (답 2 · 되물음 4 답 A).  계획 4절의
                       흠 대 보기가 이 줄이 바뀌는 것을 빠뜨렸다 (QA S9)
   decisions.md 3-13   「시작할 때 재개한다」 — 시작할 때와 살아 있는 동안 · 낡은 building · pending 의 정리도 (답 2 · 되물음 1 답 A)
   decisions.md 3-18   「실패한 시도는 … last_attempt 에 남는다」 — 합쳐 committed 로 가면 지운다 (되물음 5 답 A 의 (8)).  정본 자리는
                       ADR-077 §2 · §10 이다 (처음 판은 §5 줄에 적었다)
   features.md 기능 6   「하나라도 0 이 아니면 build 단계가 실패」 — 판정의 실패다 (단계는 DONE) · builds 는 첫 실패에서 멈춘다 (답 3)
   features.md 기능 8   bake_in_progress 와 낡은 상태 정리의 merging 갈래 · 살아 있는 동안 재개 (답 2) · 광고 주기의 정리 (되물음 1 답 A)
   scene-gates.md 조각 7 「다른 노드가 시작 때 재개」 — 거짓은 아니다.  살아 있는 형제가 먼저 이을 수 있다 (8.3 의 둘째 경우)
   scene-gates.md 조각 8 「pending 에서 죽으면 다른 노드가 정리」 (:53) — 떠 있는 형제의 광고 주기가 한다 (되물음 1 답 A)
```

`requirements.md` 10절 (「decisions.md 에 더할 행」) 에 넣을지도 진행자가 정한다 — 이 단계는 그 절을 고치지 않았다.

### 13.3 회차 문서 — 이 단계의 커밋이 함께 고친다

```text
   고친 것
     unit-of-work.md 7절            merge 단계의 차례 (그물 · 시작 전 확인 · merging) · 만지는 자리에 행렬 밖 셋
     components.md 5절              「build 의 sync · builds」 줄 (단계는 DONE · 첫 실패에서 멈춤 · merge 는 합칠 것 없음 DONE · 판정은 계약의 조건 · 되물음 2 답 B) · IR 대조 줄 ·
                                    「새 굽기 중복」에 낡은 merging (답 2) · 「합치기 도중」의 등급과 재개의 때 (답 2)
     component-methods.md 4.2       Reason 의 어휘에 ir_mismatch (DONE 에 실린다) · BuildManifest 에 HeadTags (물음 1 답 B)
     component-dependency.md        대기 자리 두 곳 (3절 표 · 5.2) — <scratch>/pending/<lower 키>/<이름>/upper
     services.md 2 · 3절            build 의 차례 (정리의 merging 갈래 · 명령 실패는 DONE · IR 대조의 자리 · 셸 · 대기 자리) ·
                                    merge 의 차례 (합칠 것 없음 · 그물 · 시작 전 확인 · merging · 대기 로그 · committed 뒤 옮김 · 놓는 차례)
     requirements.md FR-6           「하나라도 0 이 아니면 build 단계가 실패」 — 단계는 DONE 이고 판정이 실패다 · 첫 실패에서 멈춤
     requirements.md FR-8           「주인이 없으면 낡은 상태를 정리한다」에 merging 갈래 (답 2) · 시작할 때와 광고 주기의 정리와 재개
                                    (되물음 1 답 A)
     requirements.md 5.7            reason 코드 목록에 lower_changed (lower-state) · ir_mismatch (물음 1 답 B)
     unit-of-work.md 0절 · 7절       하는 일에 광고 주기의 정리와 재개 (되물음 1 답 A) · 만지는 자리의 행렬 밖을 다섯으로 (finalize.go ·
                                    lowerguard.go)
     component-methods.md 4.2 · 4.3  HeadTags 의 「대조를 못 했으면 null」 · LowerGuard 의 OnStale · Dir
     component-dependency.md 3절    bake.lock · state.json 을 쓰는 쪽에 낡은 상태를 정리하는 형제 (광고 주기)
     services.md 4 · 5절            기동의 정리와 재개를 광고 주기도 한다 · 광고 주기에 OnStale 줄 (되물음 1 답 A)
     requirements.md FR-9           대조하는 git — .repo/manifests 가 없으면 워크스페이스 뿌리의 .git (계획 3.4 · 받는 일 4) ·
                                    다르면 manifest 를 내지 않고 합치지 않는다 (물음 1 답 B — 「build 단계가 실패」를 고침)
     requirements.md 6절 조각 6      「build 단계가 실패하고」 — manifest 를 내지 않고 합치지 않으며 Run 은 계약의 조건이 실패로 판정한다
     unit-of-work.md 7절            IR 대조의 「다르면 실패」 — 같은 고침 (물음 1 답 B)
     unit-of-work-story-map.md      조각이 바뀐 자리의 「build 가 실패하고」 — 같은 고침 (물음 1 답 B)
     components.md 5절 · services.md 2절   IR 대조 줄 — FAILED 에서 DONE · reason ir_mismatch · merge 는 합칠 것 없음 (물음 1 답 B)

   앞 유닛 문서와 달라진 자리 — 병합된 유닛의 기록이라 고치지 않는다
     lower-state 규칙 3절            pending_upper 를 building -> pending 에 적는다고 썼다 — 이 단계는 building 때 적는다 (규칙 2절)
     lower-state 흐름 5절            merging 을 쓴 뒤 Preflight 를 그렸다 — 이 단계는 Preflight 뒤에 merging 을 쓴다 (규칙 10절)

   거짓이 아니라 두었다
     unit-of-work.md 7절 의 「노드 시작 순서 … 끊긴 합치기 재개」 · components.md 3.8 의 기동 순서 — 기동 순서로는 맞다. 살아 있는 동안의
     정리와 재개는 FR-8 · 0절 · services.md 5절에 더했다 (답 2 · 되물음 1 답 A).  components.md 5절 의 「merge 시작 전 확인 … 상태를
     어디로 돌리는지는 Functional Design」 — 이 문서가 그 자리다.  requirements.md FR-7 — 답과 맞다.
     contract-grammar 흐름 3절 (:73 ~ :75) 의 「build 가 실패해 manifest 가 없어도 merge 는 needs 를 따라 돈다 … 합치지 않고 merged 도
     없다 … Run 은 build 의 조건에서 이미 실패다」 — 명령 실패와 IR 어긋남 두 갈래 모두 그 절 그대로다 (합칠 것 없음 DONE · merged 없음 ·
     되물음 2 답 B · 물음 1 답 B).  처음 답 A 때는 IR 갈래가 달라졌다고 적었다
```

처음 판은 services.md 를 「이 단계가 고칠 문서 목록 밖」으로 두었다. 그런 목록은 없다 — CONVENTIONS 3.4 가 싣지 말라는 것은 `design/` 과 남의
문서 루트뿐이다 (QA). 이번에 고쳤다.

---

## 14. 확장 준수 — Functional Design

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 계약 명령 줄은 `business-rules.md` 3 · 4절이 세션 안에서 도는 것으로 닫고, 확인은 이 유닛의 NFR 이다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 한 줄 |

되물음 다섯의 답 (모두 A) 을 채웠다. 다음은 QA 재검이고, 그 뒤 유닛 정의대로면 이 유닛의 NFR Requirements (최소 — 보안 · 성능) 다.
