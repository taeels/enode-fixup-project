# `lower-state` — 흐름

타입은 `domain-entities.md`, 규칙과 문구는 `business-rules.md` 에 있다. 여기는 그 규칙이 어느 자리에서 어느 차례로 도나, 어떻게
확인하나, 그리고 다른 유닛에 무엇을 넘기나다.

---

## 1. 한눈에 — 하루치 굽기 동안 누가 무엇을 쥐나

형제 A 는 긴 Run 을 도는 중이고, 형제 B 는 쉬는 후보다. 굽는 노드 K 가 build 를 끝내 pending 이 된 뒤다.

```text
 시간 ─────────────────────────────────────────────────────────────────────────────────────────────▶

 K (굽기)   build claim: 공유 놓음 · 굽기 잠금 ── building ── pending ── merge claim ── 배타 대기 ········ 잡음 ─ 훑기 ─ merging ─ committed
 A (Run)    공유 (run) ··························· drain 실음 · 받아 적힘 ··· Run 끝 · 응답 1 · 응답 2 · 세션 닫힘 -> 놓음        공유 다시 · 새 ir
 B (후보)   공유 (candidate) ····················· drain 실음 · 응답 1 (acks 1) · 응답 2 -> 놓음                            공유 다시 · 새 ir
 smoke      (누가 env check 를 돌리면 공유를 몇 초 쥐고 놓는다.  합치는 중이면 끝날 때까지 기다린다)
 Mediator                                         A · B draining.  새 매칭 없음 (실행 중 획득도 — 6절)
```

- K 의 배타는 A 와 B 가 둘 다 놓아야 잡힌다. B 는 drain 을 싣고 두 응답 뒤(약 한 광고 주기)에 놓고, A 는 Run 이 끝나고 두 응답
  뒤에 놓는다 — 기다림은 A 의 Run 이 정한다 (ADR-077 §12)
- 합치기가 끝나면 A · B 는 공유를 다시 쥐고 · metadata 에서 새 ir 을 읽고 · drain 없는 그 광고에 싣는다 (Application Design Q1)

---

## 2. 광고 주기 한 바퀴 (`Advertiser.Run` · `LowerGuard`)

```text
   snap   := Caps()                                                    탐지 능력 (오늘 그대로)
   others := 소유자 출처 · 여유 부족 출처                                 trash 유닛의 drain() 그대로
   own, keys := guard.BeforeAdvert(others)
     dir 가 없으면 Open 을 다시 해 본다        실패 -> own += lower 출처
     st := ReadState()                        실패 -> own += lower 출처
     st 가 pending · merging                  -> own += bake 출처 (run)
     others + own 을 합친 값이 비어 있고 공유를 안 쥐었으면
       TryShared(candidate)                   못 쥐면 -> own += bake 출처 (being merged)
     공유를 쥐었으면 기록을 맞춘다             역할 · Run · acks
     md := ReadMetadata(lower)                -> keys (ir · repo.built.* · bake.run · bake.resumed)
   drain := combine(others + own)
   caps  := snap 에서 예약 키를 빼고 (라벨) + workspace.writes + keys
   상태 파일 (drain 과 출처)                                             trash 유닛 그대로
   POST /v1/nodes
   응답이 오면
     Held.Set(leases) · Held.SetDrain(drain)                           오늘 그대로
     guard.AfterResponse(drain, leases)
       leases 가 있다    공유를 쥐었으면 역할 run · acks 0.  안 쥐었으면 늦게 보인 임대 (4절)
       leases 가 0      drain 이 받아 적혔으면 acks++ .  acks 가 2 이고 도는 단계가 없으면 놓는다 (표지를 적는다)
                        drain 이 없으면 acks 0
       굽기 잠금을 쥐었고 pending 인데 그 Run 의 임대가 없다 -> 치운다 (6절)
   응답이 없으면  acks 를 그대로 둔다
```

- **공유를 먼저 정하고 metadata 를 뒤에 읽는다.** 공유를 쥔 뒤에 읽은 ir 은 그 공유를 놓을 때까지 바뀌지 않는다
- LowerGuard 는 runc-overlay 노드에만 있다. 그 밖의 노드는 `workspace.writes: in-place` 만 더한다

---

## 3. claim 과 단계 (`Worker.execute` 의 처음)

```text
   step := Claim()                                     오늘 그대로
   Held.Add(step.Lease)
   execute(step)
     임대가 유효한가                                     오늘 그대로
     err := guard.OnClaim(step)
       effect 가 prepare 이거나 종류가 merge   -> 공유를 놓는다 (결정 3-9) · nil
       거절한 Run                              -> *LowerChangedError
       공유를 쥐었다                            -> 역할 run · nil
       안 쥐었다 (놓은 뒤)                      -> 늦게 보인 임대 (4절) — 쥐면 nil, 못 쥐면 오류
       자리를 못 열었다                          -> 오류 (cannot open the lower state directory)
     err 가 있으면  준비 없이 보고한다 — lower_changed 면 reason 을 단다.  exited 는 안 보낸다 (명령이 돌지 않았다)
     ... 워크스페이스 · $IN · 세션 · 명령 · Finalize · 닫기 · 보고 (오늘 그대로)
     guard.StepDone(step)                               세션을 닫은 뒤 — 놓는 울타리가 도는 단계를 센다
```

- 거절은 보고 전에 아무것도 안 한다 — 워크스페이스를 sanitize 하지 않고 세션을 열지 않는다
- `StepDone` 은 성공 · 실패 · 패닉(`safeExecute`) 모두에서 부른다

---

## 4. 놓은 뒤에 보인 임대 — 장면마다 (답 1)

| 장면 | 보이는 곳 | state.json | 표지 | 결과 |
|---|---|---|---|---|
| drain 과 거의 같은 때 들어온 제출이 늦게 커밋했다 · 합치기 전 | 다음 광고 응답 | pending | 같다 | 공유를 다시 쥔다 (run). merge 가 그 Run 을 더 기다린다 |
| 같은 늦은 커밋 · 합치는 중 | claim | merging | 같다 | 거절 — lower_changed |
| 같은 늦은 커밋 · 합친 뒤 | claim | committed | 다르다 | 거절 — lower_changed |
| 실행 중 획득 (6절 뒤에는 drain 중 노드를 안 잡는다) | 광고 응답 | 무엇이든 | - | 위 셋과 같다 |
| 기동 뒤 첫 응답에 남은 Run (ADR-030 이 안 끝낸 것) | 첫 응답 | committed | 같다 | 이미 쥐었으면 그대로.  못 쥐었으면 위 규칙 |
| 굽기 Run 의 임대 (prepare) | claim | 무엇이든 | - | 거절하지 않는다 — build 단계 claim 에서 공유를 놓는다 |

---

## 5. merge 가 기다릴 때 — bake 가 부르는 모양

이 유닛은 아래 흐름의 잠금 · 기록 · 훑기를 만든다. 흐름과 로그 문장은 bake 유닛이다.

```text
   부르는 쪽 (bake — merge 단계 · 기동 때 재개)                  internal/lower
   -------------------------------------------------         -----------------------------------------------
   굽기 잠금을 쥐고 있다 (building 부터)
   state.json 이 pending 이고 주인이 이 Run 인가
   ex := dir.Exclusive(ctx(마감 = 받은 시각 + merge.wait),      1초마다 LOCK_EX|LOCK_NB
                        every, watch)                          every 마다 watch(Waiting{Holders, Unnamed})
     watch -> 단계 로그 한 줄: 노드 · 라벨 · 역할 · Run · since ·
              candidate 면 acks (「다음 광고에서 놓는다」) ·
              Unnamed 면 「기록 없는 쥔 쪽 (smoke 나 옛 판)」
   scan := ForeignMounts(root)                                 같은 uid 의 namespace 를 한 번씩
     찾았으면 -> ex.Release() · 로그 · 기다렸다 다시 Exclusive     (상한은 그대로)
   bake.WriteState(merging)                                    임시 파일 · fsync · rename · 디렉터리 fsync
   merge-helper: Preflight · Apply  (merge-rules)
   WriteMetadata · bake.WriteState(committed)
   ex.Release() · bake 잠금 Release
```

- 마감이면 `ctx.Err()` — bake 가 `merge_wait_timeout` 로 적는다 (결정 3-11)
- 기동 때 재개는 마감이 없다 — 찾으면 기다렸다 다시 본다

---

## 6. 굽기 Run 이 합치기 없이 끝나면 (`business-rules.md` 10절)

```text
   bake (build 단계 끝)     guard.HoldBake(run, abandon)          굽기 잠금 · pending 을 쓴 뒤
   광고 응답                guard.AfterResponse(drain, leases)
                             HoldBake 가 있고 · state 가 pending · 주인 Run 이 목록에 없다
                               -> go abandon() (한 번)          upper 를 trash 로 · committed · 굽기 잠금 놓기 (bake 의 몸통)
                               -> DropBake
   bake (merge 단계 끝)     guard.DropBake()                      committed 를 쓰고 잠금을 놓은 뒤
```

---

## 7. 노드 기동 (`cmd/enode/main.go`)

```text
   1   env check (오늘 그대로 · main.go:147)          점검 셋이 더해졌다 (8절).  smoke 가 공유를 몇 초 쥔다
   2   lowers := LowersDir() · root := ReadRoot(ws)   runc-overlay 만
   3   dir := Open(lowers, root, now)                 lower.json 대조와 쓰기.  실패는 LowerGuard 가 들고 광고마다 다시 연다
   4   (bake) 굽기 잠금 시도 · 낡은 상태 정리 · 재개      bake 유닛 — 재개는 배경에서 배타를 기다린다
   5   guard := NewLowerGuard(...)                    「놓은 상태」로 시작 · 그때의 metadata 표지
   6   삭제자 첫 회 · (checkpoint) 조정                 오늘 그대로 · checkpoint 유닛
   7   광고 시작                                       첫 BeforeAdvert 가 공유를 쥔다 (2절)
```

- `cmd/enode` 는 80.4% 다 (trash 유닛의 code-summary) — 하한에 가깝다. 조립은 `internal/enode` 의 함수 하나(예: LowerGuard 를 짓는
  함수)로 두고 main.go 에는 부르는 줄만 더한다

---

## 8. env check 와 smoke

```text
   CheckWithRuntime(ctx, doc, binding, inspector, verifier)
     binding · os · driver · host 패키지 · subordinate id · user namespace       오늘 그대로
     verifier 가 FactSource 면 Facts(...) 를 더한다                              새로 (runc-overlay 만)
       binding.scratch_filesystem   st_dev · 마운트
       lower.owner_uid              루트의 소유 uid
       lower.identity               Peek · lower.json · state.json
     prepared_environment                                                     오늘 그대로
     모두 ready 면 smoke                                                        오늘 그대로 + 공유 잠금
       dir := Peek(...)   없으면 잠그지 않는다
       s := dir.WaitShared(ctx, 1s, notice)        합치는 중이면 기다린다 · Notice 에 한 줄 (처음 · 30초마다)
       세션 열기 · 스크립트 · 닫기
       s.Release()
```

`cmd/enode/environment.go` 는 verifier 의 `Notice` 를 표준 오류로 채운다. 데몬 기동은 노드 로그로 채운다.

---

## 9. 확인의 모양

이 유닛은 맡는 조각이 없다 — 코드 검사로 병합한다 (`unit-of-work.md` 6절). 조각 8 의 준비도 점검 부분은 bake 유닛이 조각 8 을
돌릴 때 확인한다.

### 9.1 기계 (기본 `go test` · CI 에서 돈다)

```text
   internal/lower
     키            String · ParseKey 표 — stat -f 순서 · 32비트 부호 (Val 이 음수인 int32)
     뿌리          임시 폴더 ReadRoot — btime 이 0 이 아니다 (ext4) · MountID 가 0 이 아니다
     신원          identityVerdict 표 — New · Match · Reused · Foreign · Broken · btime 0 은 대조 안 함
                  rmdir 뒤 mkdir 로 같은 inode 가 나오면 Reused/Foreign 을 실제로 확인한다 (안 나오면 표 시험만으로)
     자리          Open 이 만드는 권한 (0700 · 0600) · 두 번 Open 이 paths 를 더한다 · Peek 이 아무것도 안 만든다
     상태          없으면 committed · WriteState 뒤 ReadState · 깨진 JSON 은 오류
     잠금          한 프로세스의 Dir 둘로 형제를 흉내 낸다 (fd 가 다르면 부딪친다 — 계획 2.1)
                    공유 + 공유 · 공유 중 TryBake 는 됨 · 공유 중 Exclusive 는 기다림 -> 공유 Release 뒤 잡힘 ·
                    Exclusive 의 ctx 마감 · TryBake 둘째는 false · WaitShared 가 배타 Release 뒤 잡힘
     기록          살아 있는 기록만 · Release 뒤 죽은 기록 · 차례 (기록이 살아 있으면 lower.lock 이 막혀 있다) ·
                    Unnamed (기록 없이 공유를 쥔 Dir)
     mountinfo    풀기와 overlay 찾기 표 — 측정한 helper 두 줄 (bind + overlay) 은 찾음 · 호스트의 bind 별칭 (/work 모양) 은
                    안 찾음 · lowerdir 이 lower 의 부모 · \040 이 든 경로 · lowerdir+= · 다른 장치
     훑기          이 프로세스로 ForeignMounts — Found 0 · Namespaces 1 이상
     점검          scratch 가 워크스페이스와 같은 폴더 아래 · uid 판정 표 · identity 판정 연결
   internal/enode
     LowerGuard   임시 폴더의 진짜 Dir 로 — 후보면 쥔다 · pending 이면 bake 출처 · 배타가 쥐어져 있으면 bake 출처 ·
                    응답 두 번 뒤 놓는다 · 한 번이면 안 놓는다 · 도는 단계가 있으면 안 놓는다 · 광고 실패는 셈을 안 바꾼다 ·
                    늦게 보인 임대 네 장면 (4절) · prepare claim 에서 놓는다 · 거절 뒤 보고 모양 (reason · error) ·
                    합치기 없이 끝난 굽기 (pending · building · merging)
     광고          workspace.writes (runc-overlay · native · 런타임 없음) · 예약 키를 라벨이 못 덮는다 ·
                    metadata 키 · 깨진 metadata · 어긋난 이름과 IR 은 뺀다
     drain        combineDrain 에 bake · lower 출처
     점검 이음매    Facts 가 셋을 내고 CheckWithRuntime 이 smoke 앞에 더한다 · 어긋나면 smoke 를 안 돈다
   internal/environment    FactSource 를 구현한 가짜 verifier — 자리와 State 의 순위
   internal/panel          drainName · drainLift 두 줄 (페이지 문자열 시험이 오늘 있으면 그 자리)
   internal/api            실행 중 획득 — drain 중 노드만 있으면 unavailable 갈래 · drain 없는 노드는 그대로 잡힌다 (시험 DB)
   internal/store          어휘 목록에 lower_changed — 경고가 안 남는다
```

### 9.2 사람이 SunnyVM 에서 도는 시험 (`integration` 태그 · CI 밖)

시험 바이너리를 `unshare --user --map-root-user --map-auto` 뒤에서 다시 실행한다. 버려도 되는 폴더에서만 돈다 — `/srv/yocto` 는
읽지도 않는다.

```text
   훑기          다른 프로세스가 helper 모양 (bind + overlay) 으로 임시 lower 를 마운트 -> ForeignMounts 가 찾는다 · 끝나면 0
   마운트 점검    namespace 안에서 워크스페이스를 bind 별칭으로 -> binding.scratch_filesystem 이 invalid (같은 filesystem · 다른 마운트)
   smoke        다른 프로세스가 배타를 쥔 동안 smoke -> 기다리다 놓으면 돈다
```

이 결과로 조각 8 (사람 · bake 유닛) 을 대신하지 않는다.

---

## 10. 순수 함수와 커버리지

```text
   internal/lower     새 패키지 80% 이상.  판정(judge.go)은 모든 플랫폼 · 시스템 호출은 linux 파일 — linux/amd64 로 측정한다
   internal/enode     83.6% (trash 유닛 뒤).  LowerGuard 는 새 파일 하나에 모으고 시험이 임시 폴더의 진짜 잠금을 쓴다
   cmd/enode          80.4% — 하한에 가깝다.  조립 함수를 internal/enode 에 두고 main.go 에는 부르는 줄만
   internal/store · internal/api · internal/environment · internal/panel · internal/contract    한두 줄씩과 그 시험
```

`_linux.go` 의 오류 갈래 중 시험이 못 밟는 것(예: `/proc` 을 못 읽음)은 남는다. 80% 를 넘지 못하면 Code Generation 이 호출 자리를
함수 값으로 떼어 실패를 끼운다 (merge-rules 와 같은 방법).

---

## 11. 파일 행렬 밖의 자리

행렬(`unit-of-work-file-matrix.md` 1절의 ⑥ 열)이 이 유닛에 준 파일은 `internal/lower/` · `runtime.go` · `runc_overlay_linux.go` ·
`advertise.go` · `detect.go` · `leases.go` · `status.go` · 후보 잠금과 준비도 점검의 새 파일 · `cmd/enode/main.go` ·
`cmd/enode/environment.go` · `internal/environment/check.go` · `internal/panel/view.go` · `internal/panel/boundary_test.go` 다.

```text
   internal/store/acquire.go        실행 중 획득이 drain 을 본다 (답 2 · ADR-063 §2.1 · §6) — Mediator
   internal/store/claim.go          result 어휘 목록에 lower_changed (한 줄) — Mediator
   internal/api 의 시험              실행 중 획득과 drain (시험 DB)
   internal/contract/result.go      원인 코드 ReasonLowerChanged
   internal/contract/bake.go        이름과 IR 판정을 내보낸다 (ValidBuildName · IRProblem)
   internal/enode/claim.go          execute 의 처음에 OnClaim · 끝에 StepDone
   internal/enode/policy.go         Kind 상수 둘 (bake · lower)
   internal/enode/paths.go          LowersDir
   internal/enode/runc_overlay_other.go   Capability · Facts 의 짝
   internal/panel/page.go           drainName · drainLift 두 줄 (행렬은 view.go 로 적었다 — trash 유닛도 page.go 를 고쳤다)
   안 바꿀 수 있는 것                 detect.go (예약 키는 advertise.go 가 뺀다) · status.go (새 칸이 없다) — Code Generation 이 확인한다
```

---

## 12. 다른 유닛에 넘기는 것

```text
   bake          merge 의 Preflight 에 마운트 확인을 더한다 (답 6) — merge-rules 코드 한 곳
                   (internal/merge/merge_linux.go:81 · :86 의 st_dev 비교 옆에 마운트 번호).  행렬 밖이라 bake 의 Code Generation 계획이 적는다.
                   점검 셋이 막아도 Preflight 가 스스로 확인해야 재개 때도 막힌다
                 merge 단계와 기동 때 재개의 흐름 — Exclusive · watch 의 로그 문장과 간격 · ForeignMounts 를 배타 뒤에 부르고
                   찾으면 놓고 다시 기다림 (5절)
                 state.json 을 쓰는 때와 차례 — merging 을 첫 합치기 동작 전에 (business-rules.md 3절)
                 HoldBake · DropBake 와 합치기 없이 끝난 굽기의 몸통 · building 에서 임대가 사라지면 build 단계 끝에서 치움 (6절)
                 WriteMetadata 를 부르는 자리 — lower 뿌리의 임시 파일 + rename (symlink 를 따라가지 않는다)
                 옛 판의 노드가 있는 lower 에서 굽는 장면 — 조각 6 · 8 에서 본다.  규칙 「모두 새 판으로 올린 뒤 굽는다」를
                   조각 스크립트의 확인(완료 조건 8)과 함께 둔다
                 이 계획이 측정하지 못한 것 — ext4 밖 filesystem 의 f_fsid 가 재부팅 뒤에도 같은지 · 전원이 나간 뒤 state.json ·
                   옛 판 노드가 있는 lower 의 굽기 (계획 2.11)
                 lower_changed 로 끝난 Run 이 굽기 조각에서 보이면 그 뜻 (다시 내면 된다)

   checkpoint    RuntimeCapability 의 Capture 절반 · StepRuntime.Capability() 에 더한다

   이 유닛의 NFR  보안 — 상태 자리 권한 (이미 있는 lowers 의 권한을 어떻게 다루나)
                 동시성 — 같은 기계의 여러 노드 · smoke · 재개가 같은 파일과 잠금을 쓴다 (4 · 5 · 7 · 8절)
                 비용 — 광고마다 metadata 읽기 · 배타 대기의 1초 · 훑기 (이 기계 4.7 ms)
```

---

## 13. 정본과 회차 문서에 되돌려 올리는 것

**정본** — 진행자가 `enode-design` 에 올린다.

```text
   ADR-077 §4     「같은 filesystem」은 st_dev 만으로 안 된다 — 같은 마운트여야 rename 이 된다 (EXDEV 측정).  env check 도 같다
   ADR-077 §4 §6  「마운트 0」의 증거는 lower 배타 잠금이다 — 단계 세션과 smoke 가 모두 공유를 쥔다.  배타 뒤 마운트 훑기는 그물이다.
                  한 lower 의 노드를 모두 새 판으로 올린 뒤에 굽는다
   ADR-077 §5     lower.json 에 btime (inode 재사용) · 키 글자는 stat -f -c %i 와 같은 순서 · 상태 자리는 ENODE_STATEDIR 를 따르지 않는다 ·
                  state.json 은 굽기 잠금의 주인만 쓴다 · fsync 차례.  (Application Design 이 이미 올린 「부모가 root 소유인 기계」 정정)
   ADR-077 §6     공유 잠금은 후보인 동안 쥔다 · 놓는 울타리는 두 번 연속 · 도는 단계가 없어야 놓는다 · 놓은 뒤에 보인 임대와 lower_changed ·
                  결정 3-24 의 문구 (매칭된 순간부터 임대를 놓을 때까지)
   ADR-077 §6 §7  pending 에서 굽기 Run 의 임대가 사라지면 굽는 노드가 치운다
   ADR-077 §8     bake.run · bake.resumed (true · false) · 예약 키 다섯은 라벨이 못 덮는다 · metadata 가 깨지면 안 싣는다 ·
                  이름과 IR 은 계약의 규칙으로 거른다
   ADR-063 §4 §6  노드가 스스로 거는 drain 출처 bake · lower (소유자가 못 푼다) · 실행 중 획득도 drain 을 본다 (코드가 어긋나 있었다)
   ADR-024        획득은 drain 중인 노드를 잡지 않는다 — 못 잡음 갈래로 간다
   ADR-073        env check 의 점검 셋 (binding.scratch_filesystem · lower.owner_uid · lower.identity) 과 State ·
                  FactSource 이음매 · smoke 가 lower 공유 잠금을 쥔다
   mediator-api   result 의 reason 어휘에 lower_changed
```

**회차 문서** — 이 단계의 커밋이 함께 고친다.

```text
   unit-of-work.md 6절          준비도 점검의 첫째를 「같은 filesystem 이고 같은 마운트인가」로 · 만지는 자리에 Mediator 두 파일과
                                contract 두 파일
   components.md 2.1            env check 의 확인 셋에 마운트 · 아는 것에 mountinfo 읽기
   components.md 3.3            store 에 「실행 중 획득이 drain 을 본다」
   components.md 5절            실패 등급에 「놓은 뒤에 보인 임대」 줄
   component-methods.md 1절     Key 의 붙이는 차례 · Identity 의 birth_ns 와 fsid 글자 · WriteState 가 *Bake 로 · Exclusive 의 모양 ·
                                Finding 의 Cause (겉면은 domain-entities.md 가 바꿨다고 적는다)
   component-methods.md 4.1     Capability() 를 StepRuntime 으로
   component-methods.md 4.2     Reason 에 lower_changed
   component-methods.md 4.3     LowerGuard 의 겉면 (BeforeAdvert 가 출처 목록을 받는다 · OnClaim 이 오류 · StepDone)
```

`requirements.md` FR-8 의 「scratch 와 워크스페이스의 st_dev 일치」는 고치지 않았다 — 거짓이 되지 않았고 (st_dev 는 여전히 본다),
답 6 이 마운트를 더했다. 고칠지는 진행자가 정한다.

---

## 14. 확장 준수 — Functional Design

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 상태 자리 줄은 `business-rules.md` 1절이 닫는다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 한 줄 |

다음 단계는 유닛 정의대로면 이 유닛의 NFR Requirements (최소 — 보안 · 동시성)다.
