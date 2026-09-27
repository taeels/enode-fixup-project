# `bake` — Functional Design 계획

**유닛** `bake` (굽기 단계) · **브랜치** `unit/bake` · **담당** taeels · **회차** `v4-run-finalize-bake` (굽기) ·
**순서** 여덟 중 일곱째 · **앞 유닛** lower-state (`main` `0262155` 에 병합) ·
**맡는 조각** 5 (굽기 계약 — 기계) · 6 (굽기가 끝까지 돈다) · 7 (끊겨도 된다) · 8 (배타와 대기) — 6 · 7 · 8 은 사람 · SunnyVM ·
버려도 되는 lower · **기대는 조각** 1 · 2 · 4 (셋 다 초록) ·
**완료 조건** 5 (merge 가 누구를 언제까지 기다리나) · 6 (FAILED 로 봉인된 굽기가 재개로 합쳐졌나) · 8 (합치기 조각이 운영 lower 면
멈춘다) · 9 (merge_wait_timeout 에서 build 의 성공과 merge 의 사유가 따로 보인다) · 10 (계약의 IR 과 sync 뒤 HEAD 가 어긋난 이유) ·
**스토리** US-6 (lower 를 마지막으로 바꾼 굽기) · US-12 · US-15 (merge 가 기다리는 상대와 남은 시간) · US-13 (대상 lower 가 버려도
되는 것인가) · US-16 (bake_in_progress 는 다시 내면 되는 거절) · US-17 (merge_wait_timeout 이면 빌드는 성공) · US-18 (재개로
합쳐졌나) · US-19 (IR 대조로 실패하면 sync 가 어디에 닿았나) ·
**요구** FR-5 (기능 5 · 굽기 계약 — 노드 쪽) · FR-6 (기능 6 · build 는 upper 에 짓는다) · FR-7 (기능 7 · 합치기 — 잇는 쪽) ·
FR-8 (기능 8 · lower 상태 — 재개와 나머지) · FR-9 (기능 9 · metadata 쓰기와 IR 대조)

입력은 유닛 정의(`unit-of-work.md` 7절 · 0절 코드 검사 · 10절 크로스 빌드) · 요구(`requirements.md` FR-5 ~ FR-9 · 5.3 보안 표 ·
5.5 호환 · 6절 조각 6 · 8) · 설계(`services.md` 2 ~ 5절 · `components.md` 3.1 · 3.8 · 5절 실패 등급 · `component-methods.md`
1.2 ~ 1.4 · 2절 · 4.1 · 4.2 · 9절 · Application Design 계획의 Q4 · Q6 · Q7 논의) · 스토리 지도의 ⑦ 줄 (⑦ 은 이 유닛) · 팩(`scene-gates.md` 조각
5 ~ 8 과 3절의 명령 · `decisions.md` 3-7 ~ 3-18 · 굽기의 결정) · 정본(ADR-077 §2 굽기 계약 · §3 upper 에서 짓기 · §4 합치기 · §5 상태와
metadata · §6 합치기만 미룬다 · §7 굽기끼리의 배타와 재개 · §12 실측 · ADR-030 재시작 · ADR-004 완주와 성공) · 앞 유닛 여섯이 넘긴
일(1절) · 지금 코드(2.5)다. 이 계획은 그 위에서 **build · merge 단계와 기동 때 재개가 어떤 차례로 무엇을 부르고, 실패하면 무엇을
어디로 돌리나** 만 짓는다. 상태와 잠금의 모양(lower-state) · 합치기 규칙(merge-rules) · 계약 문법(contract-grammar)은 병합됐고,
코드는 다음 단계다.

---

## 0. 이 단계가 닫는 것과 안 닫는 것

```text
   닫는다     노드가 build · merge 단계를 받는 분기 · 제자리에 쓰는 노드의 거절
             build 의 차례 — sync · IR 대조 · pinned manifest · builds (물음 3) · 명령마다의 기록 · exited · 예산
             IR 대조 — 어느 git 을 보나 · 어디서 도나 · 어긋났을 때의 보고 (물음 1 · 완료 조건 10)
             대기 자리 — upper 와 metadata 초안이 pending 동안 사는 곳
             merge 의 차례 — 배타 · 그물 · 시작 전 확인 · merging · Apply · metadata · committed
             기다림 — 로그 문장 · 간격 · 상한 시각의 시계 (완료 조건 5)
             실패 갈래마다 단계 · 원인 코드 · upper · state.json · last_attempt 를 어디로
             HoldBake · DropBake 와 치우는 몸통 하나 (lower-state 가 넘긴 차례)
             기동 때 낡은 상태 정리와 재개 · 재개가 못 끝날 때 (물음 2)
             merge-helper 의 입출력 · trash 에 넣는 법 · 시작 전 확인의 마운트 줄 (lower-state 답 6 · 같은 마운트여야 rename 이 된다)
             조각 5 의 기계 시험 모양 · 조각 6 · 7 · 8 스크립트가 할 일과 운영 lower 확인 (물음 4 · 완료 조건 8)

   안 닫는다   상태 자리 · 잠금 · 쥔 사람 기록 · 광고 키 · 준비도 점검                  lower-state (병합됨)
             합치기 표와 재개 성질                                                 merge-rules (병합됨)
             계약 문법 · 성공 판정 (produced manifest · merged)                     contract-grammar (병합됨)
             실패한 단계의 upper 를 spool 로 보존                                   checkpoint 유닛
             값의 크기 · 격리 확인 (계약의 명령이 격리 runtime 안에서만 돈다)          이 유닛의 NFR Requirements
             조각 스크립트의 실제 명령                                              이 유닛의 Code Generation
```

---

## 1. 받는 일 — 앞 단계와 앞 유닛이 넘긴 것

일곱 곳에서 쉰넷을 모았다. 넘김 절(`code-summary.md` 의 「넘기는 것」 · Functional Design 의 `business-logic-model.md` 의 「다른 유닛에 넘기는 것」)과
`grep -rn bake construction/` 로 찾은 줄이다. 표에서 FD 는 Functional Design, FD 흐름은 그 `business-logic-model.md`, FD 규칙은
`business-rules.md` 다 (앞 유닛의 문서가 쓰는 줄임 그대로). 오른쪽 칸이 이 계획이 받는 자리다.

| # | 보낸 곳 | 받는 일 | 적힌 자리 | 이 계획 |
|---|---|---|---|---|
| 1 | contract-grammar | sync 와 builds 를 셸로 돈다 | FD 흐름 8절 · code-summary 8절 | 3.3 |
| 2 | contract-grammar | 구울 IR 을 환경 변수 `ENODE_IR` (`contract.EnvIR`) 로 넘긴다 | 같은 곳 · `bake.go:42` | 3.3 |
| 3 | contract-grammar | sync 뒤 HEAD 와 IR 대조 | 같은 곳 | 3.4 · 물음 1 |
| 4 | contract-grammar | SunnyVM 의 시험 lower 는 poky 를 git 으로 받아 `.repo` 가 없다 — 대조가 git 을 알아야 한다 | 되물음 파일 끝 | 3.4 |
| 5 | contract-grammar | 성공했을 때만 `$OUT/manifest` | FD 흐름 3절 · 8절 | 3.6 |
| 6 | contract-grammar | 합쳤을 때만 `$OUT/merged` | 같은 곳 | 3.8 |
| 7 | contract-grammar | merge.wait 을 센다 (`Step.MergeWait`) | FD 흐름 8절 | 3.8 · 3.9 |
| 8 | contract-grammar | 제자리에 쓰는 노드(native)가 굽기 단계를 받으면 거절한다 | 같은 곳 | 3.2 |
| 9 | contract-grammar | 병합 뒤의 창 — 굽기 단계가 `run step has an empty argv` 로 실패한다 | code-summary 7절 | 3.2 |
| 10 | contract-grammar | build 가 실패해 manifest 가 없어도 merge 는 needs 를 따라 온다 — 합칠 것이 없음을 어떻게 알리나 | FD 흐름 3절 | 3.7 · 3.8 |
| 11 | contract-grammar | Grammar 의 굽기 절 — 계획이 지은 굽기를 노드가 받게 되어야 창이 닫힌다 | code-summary 7절 | 3.2 |
| 12 | step-phase | build 단계의 exited — 마지막 명령이 끝난 뒤 한 번 | FD 흐름 10절 · code-summary 8절 | 3.5 |
| 13 | step-phase | merge 단계는 exited 를 안 보낸다 (보내도 받지 않는다) | 같은 곳 · `store/claim.go:847` | 3.8 |
| 14 | step-phase | result 에 build (`BuildManifest`) · merge (`MergeResult`) — lower 타입에서 contract 타입으로 | 같은 곳 | 3.6 · 3.8 · 3.15 |
| 15 | step-phase | 실패한 build 에도 build 칸을 채울지 | FD 흐름 10절 | 3.6 |
| 16 | step-phase | `contract/result.go` 의 모양은 더하기만 한다 | 같은 곳 | 물음 1 |
| 17 | step-phase | merge 단계가 합치기를 어떻게 알리나 | FD 계획 0절 | 3.8 |
| 18 | finalize | build 단계의 종료 보고 — `Client.Exited` · `startExitReport` | FD 흐름 11절 · code-summary 8절 | 3.5 |
| 19 | finalize | build 단계의 두 예산 — `contractStep` 이 build 종류를 모른다 | code-summary 8절 · `finalize.go:28` | 3.5 |
| 20 | finalize | 노드가 받는 칸 `sync` · `builds` · `ir` · `merge` 는 finalize 가 안 받았다 | FD domain-entities 1절 | 3.2 |
| 21 | trash | `Keep.Upper` 에 대기 자리 | FD 흐름 10절 · code-summary 9절 | 3.6 |
| 22 | trash | merge_wait_timeout 의 upper 는 `Trash.Move` 로 | 같은 곳 | 3.9 · 3.12 |
| 23 | trash | 합치기의 lower 쪽 항목도 `Trash.Move` 로 (같은 filesystem) | 같은 곳 | 3.10 |
| 24 | merge-rules | merge-helper 안에서 Preflight 뒤 Apply — 처음과 재개가 같은 두 줄 | FD 흐름 7절 · code-summary 8절 | 3.10 |
| 25 | merge-rules | `Options.Discard` 를 `scratch.Trash.Move` 로 · Preflight 전에 trash 를 만든다 | 같은 곳 | 3.10 |
| 26 | merge-rules | merge-helper 는 namespace 안의 root 로 돈다 · 권한을 내려놓지 않는다 | 같은 곳 | 3.10 |
| 27 | merge-rules | `PreflightError.Check` 마다 상태를 어디로 돌리나 | 같은 곳 | 3.11 |
| 28 | merge-rules | Preflight 가 `*PreflightError` 가 아닌 오류를 주면 확인을 못 한 것 — 다시 해 볼 일 | code-summary 8절 | 3.11 · 물음 2 |
| 29 | merge-rules | 같은 오류가 재개 때마다 되풀이될 때 — 상태가 merging 에 머문다 | FD 규칙 9절 | 물음 2 |
| 30 | merge-rules | Apply 가 끝나면 빈 upper 뿌리를 치운다 | FD 흐름 7절 | 3.8 · 3.10 |
| 31 | merge-rules | Result 를 로그와 metadata 에 — 재개면 더한다 · 칸이 바뀔 수 있다 | 같은 곳 | 3.8 · 3.15 |
| 32 | merge-rules | 한 합치기의 trash 항목을 한 디렉터리에 모을지 (하루치 526 번 · helper 한 번에 약 5 ms) | 같은 곳 | 3.10 |
| 33 | merge-rules | 조각 7 전체 (합치는 중 SIGKILL · 다른 노드가 시작 때 잇는다) | code-summary 7절 | 3.16 |
| 34 | lower-state | merge 의 Preflight 에 마운트 확인 (`merge_linux.go:81` · `:86` 옆) | FD 흐름 12절 · 답 6 (같은 filesystem 확인에 마운트까지) | 3.11 · 2.2 |
| 35 | lower-state | merge 단계와 재개의 흐름 — `Exclusive` · watch 로그의 문장과 간격 | 같은 곳 | 3.9 |
| 36 | lower-state | watch 는 막혀 있는 동안 처음과 every 마다 불린다 · `Waiting.Unnamed` | code-summary 9절 | 3.9 |
| 37 | lower-state | `ForeignMounts` 를 배타 뒤에 · 찾으면 놓고 다시 기다린다 — 문장과 간격 | FD 흐름 12절 · FD 규칙 8.2 | 3.9 |
| 38 | lower-state | `ForeignMounts` 가 오류면 그물을 못 친 것 — 합칠지는 bake 가 | code-summary 9절 | 3.9 |
| 39 | lower-state | state.json 을 쓰는 때와 차례 — merging 은 첫 합치기 동작 전에 | FD 흐름 12절 · FD 규칙 3절 | 3.1 |
| 40 | lower-state | 배타를 기다리기 전에 `HoldBake(run, abandon)` · committed 를 쓰고 두 잠금을 놓은 뒤 `DropBake` | code-summary 9절 · `lowerguard.go:490` · `:504` | 3.12 |
| 41 | lower-state | 치우는 몸통 (pending 에서 임대가 사라짐) · building 이면 build 단계가 끝에서 | FD 흐름 6절 · FD 규칙 10절 | 3.12 |
| 42 | lower-state | `WriteMetadata` 를 부르는 자리 · 옛 임시 파일을 지우고 0644 · schema 0 이면 1 | FD 흐름 12절 · code-summary 9절 | 3.8 |
| 43 | lower-state | 옛 판 노드가 있는 lower 의 장면 · 「모두 새 판으로 올린 뒤 굽는다」를 조각 스크립트 확인과 함께 | FD 흐름 12절 · FD 규칙 8.4 | 물음 4 |
| 44 | lower-state | 측정 못 한 셋 — ext4 밖 fsid · 전원이 나간 뒤 state.json · 옛 판 노드가 있는 lower | FD 흐름 12절 · FD 계획 2.11 | 2.6 · 5절 |
| 45 | lower-state | `lower_changed` 로 끝난 Run 이 굽기 조각에서 보이면 — 다시 내면 된다 | code-summary 9절 | 3.16 |
| 46 | lower-state | 조각 8 의 준비도 점검 부분은 bake 가 조각 8 을 돌릴 때 확인한다 | FD 계획 머리 · 유닛 정의 6절 | 3.16 |
| 47 | 유닛 정의 | build · merge 의 실패 경로와 상태 되돌림 | `unit-of-work.md` 7절 | 3.7 · 3.9 · 3.11 · 물음 1 · 3 |
| 48 | 유닛 정의 | IR 대조의 환경 변수 이름과 원인 코드 | 같은 곳 | 3.3 (이름은 닫혔다) · 물음 1 |
| 49 | 유닛 정의 · Application Design Q4 논의 | 대기 로그의 모양과 상한 시각의 시계 | 같은 곳 · 계획 Q4 | 3.9 |
| 50 | 유닛 정의 | 재개가 굽기 잠금을 먼저 잡은 하나에게 가는 규칙 | 같은 곳 | 3.13 · 물음 2 |
| 51 | 유닛 정의 · `components.md` 5절 | 시작 전 확인이 어긋났을 때 상태를 어디로 | 같은 곳 | 3.11 |
| 52 | Application Design Q6 논의 | 소유자의 수단은 굽기 Run 취소 — upper 는 trash · committed · drain 이 풀린다 | 계획 Q6 | 3.12 |
| 53 | Application Design Q7 논의 | 같은 커밋에 IR 태그 둘 (IR 이 계약 값이 된 뒤 남은 뜻) | 계획 Q7 | 3.4 · 2.1 |
| 54 | 유닛 정의 | 합치기 조각 스크립트 — 굽기 전에 대상 lower 를 출력하고 운영 lower 면 멈춘다 | 같은 곳 · 스토리 지도 | 물음 4 |

12 와 18 은 같은 일이다 (둘이 같은 문장으로 넘겼다). checkpoint 유닛에서 받는 일은 없다 — 아직 오지 않았다.

---

## 2. 실측 (2026-09-27)

이 기계(커널 6.5 · ext4 · uid 1000 · 특권 없음)에서 측정했다. SunnyVM(커널 7.0)은 읽기만 했다. 측정 폴더와 프로그램은 스크래치에
두었고 끝나고 지웠다. 제품 코드로 측정할 때는 `go test -overlay` 로 시험 파일 하나를 끼웠다 — 작업 트리는 바꾸지 않았다.

### 2.1 IR 대조 — git 태그 (git 2.39)

원격 하나에 커밋 셋을 두고, 가벼운 태그 · annotated 태그 · 한 커밋에 태그 둘을 걸었다. 받는 쪽은 poky 모양(clone 뒤
`git checkout --detach refs/tags/<IR>`)과 repo 모양(`.repo/manifests` 에 브랜치 `default` 로 체크아웃)이다.

```text
   경우                                  HEAD 와 refs/tags/<IR>^{commit}    HEAD 에 붙은 태그 (git tag --points-at HEAD)
   annotated 태그                         같다                               그 태그 하나
   가벼운 태그                            같다                               그 태그 하나
   한 커밋에 태그 둘 (ir-3a · ir-3b)       ir-3a 로 대조하면 같다               ir-3a ir-3b
   원격에 없는 태그                        태그의 커밋을 못 찾는다 (빈 출력)     그 커밋의 태그들
   --no-tags 로 받은 clone                HEAD 는 맞는데 로컬 태그가 0 개 —    (없음)
                                         대조가 실패한다
     그 태그 하나만 fetch 한 뒤            같다                               ir-3a
   repo 모양 (브랜치 default)             같다                               ir-3a ir-3b
   워크스페이스에 .git 이 없고 부모에 있다   부모 저장소의 HEAD 가 나온다.  GIT_CEILING_DIRECTORIES 를 두면 오류
```

- `^{commit}` 로 벗기면 두 종류의 태그가 같은 값으로 대조된다
- **태그가 로컬에 없으면 HEAD 가 맞아도 대조가 실패한다.** sync 가 그 태그를 받아 와야 한다. 어긋남의 문장이 「태그가 없다」와
  「태그가 다른 커밋을 가리킨다」를 나눠 말해야 사람이 sync 를 고칠지 IR 을 고칠지 안다 (물음 1)
- 워크스페이스 뿌리의 `.git` 만 봐야 한다 — 부모 저장소의 HEAD 를 읽으면 틀린 값이 조용히 나온다
- 같은 커밋에 태그가 여럿이어도 계약의 IR 이 그중 하나면 맞다. Application Design Q7 논의의 「같은 커밋의 IR 둘」은 IR 이 계약
  값이 된 뒤로 대조를 흔들지 않는다

### 2.2 merge.Preflight 는 bind 별칭을 통과시킨다 — 제품 코드로

`internal/merge` 에 시험 파일 하나를 더한 바이너리를 `unshare -rm`(user · mount namespace) 안에서 돌렸다. lower 를 `mount --bind`
로 다른 자리(alias)에 걸고, upper 와 trash 는 scratch 에 두었다.

```text
   자리      st_dev    STATX_MNT_ID
   upper     252:15    1220
   lower     252:15    1220
   alias     252:15    1423        lower 를 bind 한 자리
   trash     252:15    1220

   Preflight(upper, lower=alias, trash)       nil — 통과한다
   Preflight(upper, lower=원래 경로, trash)     nil
   rename upper/a -> alias/a                  invalid cross-device link (EXDEV)
   rename upper/a -> 원래 경로/a                된다
```

- lower-state 계획 2.4 (st_dev 가 같아도 마운트가 다르면 EXDEV) 를 merge 의 실제 `Preflight` 로 다시 봤다. 별칭 경로를 lower 로
  받으면 확인을 통과하고 Apply 의 첫 rename 에서 멈춘다 — 재개도 같은 자리에서 멈춘다
- 커널이 `STATX_MNT_ID` 를 준다 (mask 에 있다). 별칭과 원래 자리의 번호가 다르다 — 확인할 값으로 쓸 수 있다 (3.11)

### 2.3 LowerGuard 가 굽는 동안 공유 잠금을 어떻게 쥐나 — 제품 코드로

`internal/enode` 에 시험 하나를 끼워 임시 폴더의 진짜 자리와 잠금으로 돌렸다 (lower-state 의 시험 틀 `newGuardFixture`).

```text
   1  쉬는 노드의 광고                      공유를 쥔다 (candidate)
   2  build 단계 claim (OnClaim)            놓는다
   3  building 인 동안의 광고                 다시 쥔다 (candidate · 쥔 사람 기록 1)     lowerguard.go:170 — 실을 drain 도 굽기 표지도 없다
   4  HoldBake 뒤의 광고                     놓는다 · 다시 안 쥔다
   5  pending 인 광고                        bake 출처 drain — lower pending a merge (run R1)
   6  응답에 그 Run 의 임대가 있다             치우는 몸통 0 번
   7  응답에 그 Run 의 임대가 없다             치우는 몸통 1 번 · 굽기 표지가 지워진다
```

- HoldBake 전까지는 굽는 노드가 building 동안 공유를 candidate 로 다시 쥔다. 그동안 배타를 기다리는 쪽이 없으므로 해가 없다 — 굽기
  잠금이 다른 굽기와 재개를 막는다. HoldBake 가 놓는다 (3.12)
- 치우는 몸통은 광고 응답에서 따로 도는 고루틴이다 (`lowerguard.go:431`). 같은 응답이 Held 의 임대 목록을 바꾸고
  (`advertise.go:215`) · Worker 의 임대 감시가 1초 안에 도는 단계의 ctx 를 끊는다 (`claim.go:665`). merge 단계가 배타를 기다리는
  중에 임대가 사라지면 **치우는 길 둘이 함께 돈다** (3.12)

### 2.4 SunnyVM — 읽기만

```text
   커널 7.0.0 · 호스트 git 2.43 · 호스트에 repo 없음
   떠 있는 enode 둘 — 둘 다 ~/bin/enode (2026-09-22 22:36 +0900 판) 로 2026-09-23 08:19 · 19:39 에 떴다
```

- 그 판은 lower-state 전이다 — 잠금을 쥐지 않고 `workspace.writes` 를 광고하지 않는다. 그 키는 `9d7197f` 가 처음 들였다
  (`git log -S`). 새 판으로 굽는 lower 에 이 노드가 형제로 있으면 옛 판의 틈(pending 에도 새 일을 받는다)이 그대로 있다 (물음 4)
- `/srv/yocto` 와 떠 있는 노드의 설정 · 상태 자리는 읽지 않았다. 명령 줄도 읽지 않았다

### 2.5 코드 — 지금 `main` (`0262155`)

```text
   노드의 Step 에 sync · builds · ir · merge 칸이 없다          internal/enode/claim.go:21 ~ :89.  Mediator 는 싣는다
                                                             (store/claim.go:187 ~ :190)
   굽기 단계는 run step has an empty argv 로 끝난다              claim.go:676 ~ :679
   세션 닫기가 Keep 을 안 본다                                  runc_overlay_linux.go:486 (483 의 주석)
   upper · merged 는 helper 의 마운트 namespace 안에만 있다       runc_overlay_linux.go:899 ~ :930.  호스트는 합쳐진 모습을 못 본다
   컨테이너 사용자는 바깥 namespace 의 0 (노드 uid) 이다           runc_overlay_linux.go:1269 ~ :1271
   result 에 error 가 있으면 단계 FAILED · Run 이 곧바로 FAILED    api.go:432 · store/reap.go:326 ~ :343.  error 가 없으면 DONE 이고
                                                             needs 를 따라 다음 단계가 온다 · 판정은 끝에 success_when
   원인 코드가 있는 결과는 모두 error 도 있다                     finalize.go:210 ~ :241 (예산 둘) · claim.go:524 ~ :533 (lower_changed) ·
                                                             components.md 5절 (bake_in_progress · merge_wait_timeout 은 단계 등급)
   exited 는 merge 종류를 받지 않는다                            store/claim.go:793 · :847
   graceful drain 은 claim 을 안 막는다 · claim 은 drain 을 안 본다  claim.go:447 ~ :456 · store/claim.go 의 ClaimStep —
                                                             pending 인 굽는 노드도 자기 merge 단계를 받는다
   run_id 는 계약이 적는 아무 문자열이다                          contract.go:1087 — 경로 조각으로 쓰면 / 와 .. 이 들어온다
   trash 이름은 끝 조각 + -1 · -2                                scratch/trash.go:43
   삭제자는 항목이 쓰이는 중인지 모른다                            scratch/deleter.go:48 ~ :60 — 도는 단계가 있어도 멈추지 않고 10분마다
                                                             깬다.  항목 하나에 helper 하나 (약 5 ms · trash code-summary 6절)
   두 예산                                                     finalize.go:28 contractStep 이 kind 에 쓰이는 칸을 안 옮긴다 —
                                                             Budgets 는 kind 를 안 보므로 build 의 예산은 맞다.  MergeWait 은 merge 칸이 필요하다
   BuildManifest 에 HEAD 의 태그 칸이 없다                        contract/result.go:164 ~ :174 (head 하나 · ir 하나)
   기동 순서                                                    cmd/enode/main.go:249 StartLowerGuard · :312 SweepOrphanSessions · :329 광고 시작
   trash-helper 는 --mount 없이 연다                             trash_linux.go:104 — 호스트의 마운트 번호를 그대로 본다
```

### 2.6 못 한 것

- rootfs 안의 git 과 repo — SunnyVM 의 준비된 rootfs 는 떠 있는 노드의 자리라 안 읽었다. 이 기계는 `unshare --map-auto` 가 막혀
  runc 세션을 못 연다 (lower-state code-summary 7절). IR 대조를 세션 안에서 돌리는 전제(3.4)는 조각 6 (굽기가 끝까지 돈다) 에서 처음 확인한다
- `repo init -b refs/tags/<IR>` 뒤 `.repo/manifests` 에 그 태그가 로컬로 있는지 — repo 가 이 기계와 SunnyVM 호스트에 없다.
  없으면 2.1 의 「--no-tags」 줄처럼 대조가 실패하고 문장이 태그를 받으라고 말한다
- 세션 안 git 의 소유 확인 (safe.directory) — 파일 주인과 컨테이너 사용자가 같은 uid 로 보일 것이다 (2.5 의 매핑). 조각 6 (굽기가 끝까지 돈다) 에서 본다
- 합치는 중 SIGKILL 과 다른 노드의 재개 — 조각 7 (끊겨도 된다 · 사람 · SunnyVM)
- lower-state 가 넘긴 셋 — ext4 밖 filesystem 의 fsid 가 재부팅에 같은지 · 전원이 나간 뒤 state.json · 옛 판 노드가 있는 lower
  (조각 6 굽기가 끝까지 돈다 · 조각 8 배타와 대기 · 물음 4)

---

## 3. 이 계획이 정하는 것 — 묻지 않는다

근거가 정본 · 설계 · 앞 유닛 · 2절에 있다. 반대하면 답에 적어 주세요. 영어 문장은 초안이고 Functional Design 산출물이 굳힌다.

### 3.1 전이와 쓰는 때 (lower-state 규칙 3절의 표 · ADR-077 §5 상태 기계)

```text
   때                                              전이                      state.json 에 적는 것
   build claim · 굽기 잠금을 잡음 · committed          committed -> building     owner (Run · 단계 · 노드 · 인스턴스) · pending_upper
                                                                             (대기 자리를 이때 먼저 잡는다 — 3.6)
   build 성공 · 세션을 닫은 뒤                          building -> pending       since
   merge · 배타 · 그물 · Preflight 뒤 · 첫 Apply 전     pending -> merging        (fsync 까지 · lower-state 규칙 3절)
   metadata 뒤 · 대기 자리를 trash 로 옮긴 뒤            merging -> committed      owner · pending_upper 를 비운다
   build 실패 · IR 어긋남 · 대기 상한 · 치우는 몸통 ·     building · pending ->     last_attempt (3.14)
     Preflight 어긋남 · 기동 때 낡은 상태 정리              committed
```

- **merging 에서 committed 로 가는 길은 합치기가 끝난 것 하나다.** lower 가 반쯤 바뀌었을 수 있으므로 되돌리지 않는다 (물음 2)
- building 에 pending_upper 를 적는 까닭 — 세션을 닫는 rename 과 pending 쓰기 사이에 죽으면 기동 정리가 upper 의 자리를 알아야 한다

### 3.2 단계 분기와 거절

- 노드의 `Step` 에 `sync` · `builds` · `ir` · `merge` 칸을 더한다 (Mediator 는 이미 싣는다 · 2.5). `contractStep` 도 이 넷을 옮긴다
  — 그래야 `MergeWait` 이 계약의 값을 읽는다 (finalize 가 넘긴 일)
- `execute` 는 `Kind` 가 `build` · `merge` 면 빈 argv 확인(`claim.go:676`) 전에 굽기 흐름으로 간다. 계획이 지은 굽기(Grammar 의
  굽기 절)도 같은 분기로 들어온다 — 병합 뒤의 창이 닫힌다
- **LowerGuard 가 없는 노드(native · runtime 없음)는 거절한다.** 준비 없이 곧바로 FAILED — `a bake step needs a node that writes to
  an upper (workspace.writes=isolated); this node writes to its workspace in place`. 원인 코드는 없다 (노드 설정의 일이다)
- 상태 자리를 못 여는 노드도 곧바로 FAILED — `cannot open the lower state directory: <원인>`. lower-state 의 lower 출처 drain 이
  이미 이 노드를 후보에서 뺐으므로 드물다

### 3.3 build 의 명령 (FR-6 · build 는 upper 에 짓는다 · ADR-077 §2 굽기 계약)

- 차례 — sync → IR 대조 (3.4) → pinned manifest (repo 일 때 · 3.6) → builds 를 적힌 차례로. sync 가 0 이 아니면 그 뒤는 안 돈다 —
  맞지 않은 트리 위의 빌드는 뜻이 없다. build 하나가 실패한 뒤는 물음 3
- 명령마다 세션의 `Run` 한 번 — `sh -c <명령>` · 작업 폴더는 워크스페이스 자리 · 시작과 끝 시각(노드 시계)과 exit. 명령에는 시간 상한이
  없다 — 임대가 끝나면 끊긴다 (오늘의 임대 감시)
- 환경 — 오늘의 명령 단계와 같은 목록(`commandEnv` + 계약의 `env` 이름) 에 `OUT` · `IN` · `ENODE_IR` (`contract.EnvIR`). 이름은
  contract-grammar 의 코드가 닫았다
- 단계 로그에 명령마다 머리 한 줄과 끝 한 줄 — `bake: sync started` · `bake: build config-a exited 0 after 41m45s`

### 3.4 IR 대조 (FR-9 · 완료 조건 10)

- **보는 git** — 워크스페이스 뿌리에 `.repo` 가 있으면 `.repo/manifests`, 없고 뿌리에 `.git` 이 있으면 워크스페이스 뿌리, 둘 다
  없으면 대조를 못 한다 (`cannot verify ir: the workspace has neither .repo/manifests nor .git`). 근거 — ADR-077 §5 (상태와 metadata) 가 이미
  「manifest 가 없는 단일 git workspace 면 url · branch · head 만 채우고 pinned 는 null」을 적었고, `DetectRepo`
  (`repoid.go:77`) 가 같은 차례로 본다. SunnyVM 의 시험 lower (poky git) 가 이 갈래로 조각 6 ~ 8 (굽기 · 끊김 · 배타의 사람 조각) 을 돈다. 부모 저장소를 읽지 않게
  `GIT_CEILING_DIRECTORIES` 를 둔다 (2.1 끝 줄)
- **어디서** — 세션 안에서, 노드가 지은 고정된 셸 한 줄로 (`git rev-parse HEAD` · `git rev-parse -q --verify
  refs/tags/$ENODE_IR^{commit}` · `git tag --points-at HEAD` · `git config --get remote.origin.url` · `git rev-parse --abbrev-ref HEAD`).
  합쳐진 모습은 helper 의 마운트 namespace 안에만 있다 (2.5). 새 helper 동작이 필요 없고 격리 runtime 을 벗어나지 않는다. rootfs 에
  git 이 있어야 한다 — sync 가 git 이나 repo 를 쓰면 이미 있다 (확인은 2.6)
- **맞음** — 태그의 커밋이 HEAD 와 같다. HEAD 에 다른 태그가 함께 있어도 된다 (2.1)
- **어긋남의 두 갈래** — 태그가 로컬에 없다 · 태그가 다른 커밋을 가리킨다. 보고의 모양은 물음 1
- 대조는 sync 바로 뒤, builds 전이다 — 빌드 하나가 30 ~ 80 분이다 (ADR-077 §12 실측 · 1,892 초 · 4,760 초)

### 3.5 exited 와 예산

- exited 는 계약의 명령이 끝난 뒤 한 번 — 마지막으로 돈 명령(실패한 것이나 마지막 build)의 결과로 보낸다. 계약의 명령이 하나도
  안 돌았으면 안 보낸다 (거절 · bake_in_progress · 자리를 못 엶). IR 대조 · pinned · git 조회는 노드의 명령이라 세지 않는다
- 두 예산은 오늘 규칙 그대로 — Finalize 예산이 결과 확정 · 닫기(`Keep` 의 rename) · 초안과 pending 쓰기를 덮고, 업로드 예산이 로그와
  manifest 를 덮는다. 계약이 build 의 budget 을 늘릴 수 있다 (contract-grammar)
- merge 단계는 exited 를 안 보내고 예산을 안 쓴다 — 기다림은 merge.wait, 합치기 본체는 상한이 없다 (결정 3-11 · 대기 상한 기본 4시간 · 본체에 상한 없음)

### 3.6 build 이 성공하면 — 대기 자리와 초안

```text
   <scratch>/pending/<lower 키>/<고유 이름>/
     upper/         세션의 upper 를 rename 한 번으로 (Keep.Upper)
     bake.json      metadata 초안 — bake 칸만 빼고 전부 (source · builds · environment · workspace_target) + Run · 노드
```

- **이름에 run_id 를 쓰지 않는다.** 계약이 적는 아무 문자열이다 (2.5). 고유 이름은 `MkdirTemp` 이고 경로는 state.json 의
  pending_upper 가 든다
- lower 키 아래에 두는 까닭 — 한 scratch 를 여러 lower 의 노드가 나눠 쓸 수 있다. 굽기 잠금을 쥔 쪽이 자기 lower 키 아래의 버려진
  자리만 치운다 (3.13)
- 초안을 대기 자리에 두는 까닭 — 재개하는 노드가 metadata 를 쓰려면 build 가 안 사실이 필요하다. state.json 은 형제가 광고마다
  읽는 작은 파일이고 그 타입은 lower-state 의 것이다. 초안은 upper 와 함께 살고 함께 trash 로 간다
- 차례 — Finalize (effect prepare · diff 없음 · 훑기 없음) → `Close(Keep{Upper: <자리>/upper})` (upper 만 rename · 나머지 runRoot
  는 trash) → 초안 → pinned 의 sha256 (대기 자리에서 읽는다 — upper 의 파일은 노드 uid 소유다) → `WriteState(pending)` →
  `HoldBake(run, 치우는 몸통)` → `$OUT/manifest` → 업로드 → 보고 DONE
- 초안의 칸 — source.url · branch (`git rev-parse --abbrev-ref HEAD` · detached 면 빈 값) · repo_id (`DetectRepo` 의 규칙을 세션 안의
  값에 그대로 · 합친 뒤 detect 가 광고할 값과 같다) · head · ir (계약의 IR) · pinned (repo 일 때 `repo manifest -r -o
  .enode-manifest.xml` · 아니면 null) · sync_command · synced_at (sync 의 끝) · builds · environment (`Record.PreparedEnvironment`) ·
  workspace_target (`Record.WorkspaceTarget`)
- pinned 는 IR 이 맞은 바로 뒤, builds 전에 뜬다 — builds 는 저장소의 판을 안 바꾸고, 실패하면 빌드 전에 알린다
- **result 의 build 칸은 명령이 하나라도 돌았으면 늘 채운다** (step-phase 가 넘긴 물음) — 돈 명령의 기록 · head · ir (맞았을 때) ·
  pinned. 완료 조건 9 (build 의 성공이 따로 보인다) · 10 (IR 과 HEAD 의 차이) 이 이 칸을 본다
- `$OUT/manifest` 의 내용은 build 칸과 같은 JSON 이다

### 3.7 build 이 합칠 upper 를 못 남기면

```text
   갈래                           단계                원인 코드          upper    state.json            last_attempt   merge 단계
   sync 나 build 가 0 아님          DONE · exit 는 끝낸  없음               trash    committed            남긴다          온다 — 합칠 것 없음
                                  명령의 값                                                                             (3.8 첫 줄)
   IR 이 어긋남                    물음 1              ir_mismatch        trash    committed            남긴다          물음 1
   굽기 잠금의 주인이 살아 있다       FAILED              bake_in_progress   없음     그대로                안 남긴다        안 온다
   낡은 building · pending          치우고 진행한다 (ADR-077 §7 · 굽기끼리의 배타와 재개) — 3.13 과 같은 몸통
   낡은 merging                    물음 2
   노드 쪽 오류 (git 없음 · pinned    FAILED              없음               trash    committed            남긴다          안 온다
     실패 · 세션을 못 엶)
   임대가 끝남 (도는 중)             FAILED (aborted:     없음               trash    committed            남긴다          안 온다
                                  lease expired)
   pending 을 쓴 뒤 예산을 넘김       FAILED              finalize_timeout   trash    committed            남긴다          안 온다
     (결과 확정 · 업로드)                                또는 upload_timeout — 치우는 몸통을 build 단계가 곧바로 부른다 (3.12)
```

- 명령의 0 아닌 exit 는 완주다 (ADR-004 · 완주와 성공을 나눈다 · 오늘의 명령 단계). 성공은 success_when 의 produced manifest 가 판정한다 (contract-grammar
  답 1 · 판정은 produced). merge 는 needs 를 따라 오고 합칠 것이 없다 (contract-grammar 흐름 3절)
- bake_in_progress 의 문장 (US-16) — `another bake holds this lower: run <Run> on node <노드> since <시각> (<phase>); submit the bake
  again later`. 주인은 state.json 의 owner 에서 읽는다
- 노드 쪽 오류는 오늘의 `runtime open:` · `workspace:` 와 같은 등급이다 — 완주가 아니다

### 3.8 merge 단계

```text
   claim                        Mediator 가 phase waiting (결정 1-12 — 합치는 몇 초는 따로 안 알린다)
   OnClaim                      놓을 공유가 없다 — HoldBake 가 이미 놓았다
   확인                          이 프로세스가 이 Run 의 굽기를 쥐었고 state 가 pending · 주인이 이 Run
     아니고 이 Run 의 대기 자리도 없다   DONE · merged 없음 · 로그 bake: nothing to merge; the build step left no pending upper
     state 는 이 Run 인데 안 쥐었다    FAILED — 재시작 뒤다 (ADR-030 · 재시작한 노드의 Run 을 닫는다 · 기동 정리가 치운다)
   마감                          claim 을 받은 때 (노드 시계) + MergeWait
   Exclusive(마감, 10초, watch)    3.9
   ForeignMounts                3.9
   Preflight                    merge-helper 첫 실행 (3.10).  어긋나면 3.11
   WriteState(merging)
   Apply                        merge-helper 둘째 실행.  ctx 로 끊지 않는다 (본체에 상한 없음)
   빈 upper 뿌리 rmdir
   WriteMetadata                초안 + bake {run · node · merged_at · resumed false · previous_ir}.  합치기의 마지막 동작
   대기 자리를 trash 로           초안과 빈 자리 · rename 한 번
   WriteState(committed) · 배타 Release · 굽기 Release · DropBake
   $OUT/merged · 보고 DONE       merge 칸 (MergeResult)
```

- 「합칠 것 없음」을 DONE 으로 보고하는 까닭 — FAILED 는 「완주하지 못했다」다 (ADR-004 · 완주와 성공). 확인하고 할 일이 없었던 것은 완주이고,
  Run 은 build 의 조건(produced manifest)에서 이미 실패다 (contract-grammar 흐름 3절). merge 의 조건도 merged 가 없어 거짓이다
- previous_ir 은 합치기 전 lower 의 metadata 의 source.ir 이다 (없으면 null)
- `WriteMetadata` 는 lower-state 가 지은 그대로 부른다 — 옛 임시 파일을 지우고 0644 · schema 를 채운다 · 호스트(노드 uid)에서 부른다
- MergeResult 의 셈 (`contract.MergeOps`) 은 `merge.Result` 에서 옮긴다 — replaced ← Replaced + TypeChanged · created ← Added ·
  dirs ← NewDirs · opaque ← OpaqueDirs · whiteouts ← Whiteouts · trashed ← Discarded · attrs ← MergedDirs. 로그에는 `merge.Result`
  를 칸 그대로 쓴다. metadata 에는 안 싣는다 — metadata 의 칸은 결정 3-16 (metadata 가 담는 것) 그대로다
- 합치는 중 임대가 사라지면 — Apply 는 끝까지 간다. 보고는 못 닿을 수 있다. lower 는 committed 로 가고 광고의 bake.run 이 그 Run
  이다 (완료 조건 6 · 재개로 합쳐졌나를 광고로 본다 — 와 같은 길)
- SIGTERM 이면 도는 Apply 를 기다려 끝낸다 (하루치 1 ~ 2 초 · merge-rules code-summary 6절 1.11 초). SIGKILL 이면 helper 도 죽고
  (`--kill-child`) merging 이 남는다 — 재개가 잇는다

### 3.9 기다림 — 로그 · 시계 · 그물 (완료 조건 5 · US-12 · US-15)

- **상한 시각은 노드 시계다.** 마감을 지키는 쪽이 노드이고, 로그는 노드가 언제 포기하는지를 말해야 한다. 한 줄에 마감 시각(UTC ·
  `node clock`)과 남은 시간을 함께 쓴다 — 남은 시간은 시계와 무관하다. 진행 조회의 phase_since 는 Mediator 시계 그대로다
  (Application Design Q4 논의)
- watch 간격 10 초. 로그는 처음 · 쥔 쪽의 모습이 바뀔 때 (노드 · 역할 · Run · acks) · 그 밖에는 5분마다 한 번. 4시간이면 바뀜을 빼고
  48 줄이다

```text
   waiting for the lower lock; deadline 2026-09-27T09:00:00Z node clock (3h52m left)
     node box-a holds it for run R-8790 since 2026-09-27T04:10:00Z
     node box-b holds it as a candidate; its drain was acknowledged 1 of 2 times
     a holder left no record (an env check smoke, or an older enode that takes no lower lock)
```

- 마감 — `merge_wait_timeout`. FAILED · `gave up waiting for the lower lock after 4h0m0s (merge.wait)` · 치우는 몸통 (3.12). build 의
  DONE 은 그대로라 Record 에서 둘이 따로 보인다 (완료 조건 9)
- 임대가 사라짐 — FAILED `aborted: lease expired` · 같은 몸통
- **그물** — 배타를 잡은 뒤 · Preflight 전에 `ForeignMounts`. 찾으면 배타를 놓고 찾은 마운트마다 한 줄 (pid · namespace · 마운트 자리 ·
  lowerdir) 을 쓰고 60 초 뒤 다시 배타를 기다린다 (마감은 그대로). 곧바로 다시 잡으면 잠금을 모르는 옛 판 노드의 마운트를 1초마다 훑는다
- 그물이 오류면 (자기 mountinfo 를 못 읽음) 경고 한 줄을 쓰고 합친다 — 증거는 배타 잠금이다 (lower-state 규칙 8.1). 못 읽은
  프로세스(Unreadable)가 있으면 그 수를 한 번 쓴다
- 재개도 같다 — 마감이 없고 로그는 노드 로그로 간다

### 3.10 merge-helper 와 trash

- 입구 `enode merge-helper` 를 trash-helper 와 같은 모양으로 연다 — `unshare --user --map-root-user --map-auto --fork --kill-child`
  · `--mount` 없음 (`trash_linux.go:104`). namespace 안의 root 로 돌고 권한을 내려놓지 않는다 (merge-rules)
- 한 합치기에 두 번 연다 — preflight · apply. 그 사이에 호스트가 merging 을 쓴다. 한 번 여는 데 약 5 ms 다
- 입력은 stdin 의 JSON 한 줄 (동작 · upper · lower · trash) · 출력은 stdout 의 JSON 한 줄 (Result · 오류 · 어긋난 Check)
- trash 는 **대기 upper 가 있는 scratch 의 trash** 다 (`<scratch>/pending/...` 의 scratch). 재개하는 노드가 다른 scratch 를
  써도 같은 자리로 간다. Preflight 전에 `MkdirAll` 로 만든다
- `Discard` 는 `scratch.Trash.Move` 로 항목마다 한 번 — 합치기 하나의 항목을 한 디렉터리에 모으지 않는다. 삭제자는 항목이 쓰이는
  중인지 모르고 도는 단계가 있어도 10분마다 깬다 (2.5) — 모은 디렉터리는 채우는 중에 지워질 수 있다. 낱개의 값은 하루치 526 번 ×
  약 5 ms = 약 2.6 초의 배경 일이다
- Apply 뒤 빈 upper 뿌리를 rmdir 하고, 대기 자리(초안 포함)를 `Trash.Move` 한 번으로 옮긴다

### 3.11 시작 전 확인 — 마운트 줄과 어긋났을 때

- **마운트 줄을 더한다** (lower-state 답 6 · 같은 filesystem 확인에 마운트까지) — `merge_linux.go:81` · `:86` 의 st_dev 옆에서 셋의 `STATX_MNT_ID` 를 댄다. 새 Check
  `mount` · 문장 `merge preflight: upper, lower and trash must share one mount (upper mount <n>, lower mount <n>, trash mount <n>);
  a bind alias of the lower cannot take a rename`. 커널이 번호를 안 주면 `*PreflightError` 가 아니라 확인을 못 한 오류다 — runc-overlay
  는 user namespace 의 overlay(커널 5.11)가 필요해 번호(5.8)는 늘 있다. merge-rules 의 파일이라 행렬 밖이다
- 어긋났을 때

```text
   때                   *PreflightError (path · directory · overlap · filesystem · mount · mark)   그 밖의 오류 (읽기 실패)
   merge 단계 (pending)  merging 을 안 쓴다.  lower 는 그대로다 — upper 를 trash · committed ·          같다.  다시 해 보지 않는다
                        두 잠금을 놓는다 · DropBake · FAILED (error 에 Preflight 의 문장) ·          (3.7 의 노드 쪽 오류)
                        last_attempt
   재개 (merging)        upper 를 버리지 않는다 — lower 가 반쯤 바뀌었을 수 있다.  merging 에 둔다       물음 2
                        (물음 2)
```

- merge 단계에서 Preflight 를 merging 보다 먼저 두는 까닭 — Preflight 는 아무것도 바꾸지 않는다. merging 을 쓴 뒤에 어긋나면
  committed 로 돌릴 길이 없다 (3.1 첫 줄)

### 3.12 HoldBake · DropBake · 치우는 몸통 하나

- **HoldBake 는 pending 을 쓴 바로 뒤** — lower-state 의 차례 그대로다 (`lowerguard.go:490` 의 주석 · 배타를 기다리기 전). 그 전
  building 동안 공유를 candidate 로 다시 쥐는 것은 해가 없다 (2.3)
- **DropBake 는 HoldBake 뒤 굽기 잠금을 놓는 모든 길의 끝** — 합침 · 대기 상한 · 임대가 사라짐 · Preflight 어긋남 · pending 뒤 예산을
  넘긴 build · 치우는 몸통
- **치우는 몸통은 굽기 하나에 하나다.** 프로세스의 굽기 보관(Run · Dir · 굽기 잠금 · 대기 자리 · 단계)에 mutex 하나와 「합치기
  시작」 표지를 둔다. 광고 응답의 몸통(`bakeGoneLocked`)과 merge 단계의 실패 길이 같은 함수를 부르고, 한 번만 돈다. merge 단계는
  merging 을 쓰기 전에 같은 mutex 아래에서 표지를 적고, 몸통이 먼저 돌았으면 거기서 멈춘다. 몸통은 표지가 있으면 아무것도
  안 한다 — 합치기 본체를 끊지 않는다 (2.3 끝 줄)
- 몸통이 하는 일 — 대기 자리를 trash · `WriteState(committed)` + last_attempt · 굽기 Release · DropBake. 소유자가 굽기 Run 을
  취소한 길이 이것이다 (Application Design Q6 논의)
- building 에서 임대가 사라지면 build 단계가 끝에서 치운다 — 세션이 upper 를 쓰는 중이다 (lower-state 규칙 10절)

### 3.13 기동 — 낡은 상태 정리와 재개 (ADR-077 §7 · 결정 3-12 주인이 살아 있으면 bake_in_progress · 3-13 시작할 때 재개)

- 자리 — `cmd/enode/main.go` 의 `StartLowerGuard` (:249) 뒤 · 광고 시작 (:329) 전. 조립은 `internal/enode` 의 함수 하나이고 main.go
  에는 부르는 줄만 (cmd/enode 80.5% · 하한에 가깝다)
- TryBake 가 되면 — committed 면 자기 scratch 의 `pending/<lower 키>/` 아래 버려진 자리를 trash 로 옮기고 놓는다 · building ·
  pending 이면 그 자리를 (그 자리가 있는 scratch 의) trash 로 · committed · 놓는다 · merging 이면 배경에서 재개하고 잠금은 재개가 쥔다. 안 되면 다른 노드가
  쥐었다 — 그쪽이 한다. **먼저 잡은 하나가 한다** 는 flock 이 준다
- 재개 — 배타 (마감 없음) → 그물 → 끝났는지 본다 → Preflight → Apply → metadata → 대기 자리를 trash → committed → 놓는다
- **끝났는지 먼저 본다** — metadata 의 bake.run 이 주인 Run 이면 합치기와 metadata 가 끝났다 (죽은 자리가 그 뒤다) — committed 만
  쓴다. upper 뿌리가 없고 초안이 있으면 Apply 가 끝났다 — metadata 부터
- 재개의 metadata — resumed true · run 은 주인 Run · node 는 주인 노드 (그 굽기를 한 노드 · US-6) · merged_at 은 재개가 끝난 때 ·
  previous_ir 은 지금 lower 의 metadata. 재개한 노드는 자기 노드 로그에 남는다
- 재개의 로그는 노드 로그다 — 단계가 없다

### 3.14 last_attempt

- **남기는 때** — building 을 쓴 굽기가 합치지 않고 끝난 모든 길 (3.7 · 3.9 · 3.11 · 3.12). bake_in_progress 와 거절은 굽기를
  시작하지 않았으므로 안 남긴다
- 칸 — Run · 때 · reason (원인 코드가 있으면 그것, 없으면 영어 한 줄 — `sync exited 1` · `build config-a exited 2` · `cannot verify
  ir: ...`) · builds (돈 것)

### 3.15 결과 칸

- `lower.BuildRecord` · `lower.Pinned` 를 `contract` 타입으로 옮긴다 — lower-state 의 시험(`TestMetadataRecordsMatchTheContract`)이 두
  벌의 칸을 맞춰 둔다
- `$OUT/manifest` 는 `BuildManifest` JSON · `$OUT/merged` 는 `MergeResult` JSON. 판정은 오늘의 produced 대조다 (Verify 불변)

### 3.16 조각

- 조각 5 (굽기 계약) 의 bake 몫 — 기본 `go test` 에서, 임시 lower 의 진짜 잠금으로. 형제를 흉내 낸 Dir 이 공유를 쥔 채 merge.wait
  을 짧게 준 merge 단계가 claim 시각 + wait 근처에서 `merge_wait_timeout` 으로 끝나는지 · 다른 값이면 그 값으로 기다리는지
- 조각 6 · 7 · 8 — 스크립트 셋 (`scripts/finalize-bake/slice-6.sh` · `slice-7.sh` · `slice-8.sh` 이 후보 · 실제 이름은 Code
  Generation). 굽기를 내기 전에 대상 lower 를 확인한다 (물음 4). 결과에 `lower_changed` 가 보이면 「다시 내면 된다」를 출력한다.
  조각 8 (배타와 대기) 은 lower-state 의 점검 셋(다른 filesystem · 다른 uid · bind 별칭)을 함께 본다
- 조각 7 (끊겨도 된다) 의 기계 부분은 merge-rules 의 재개 시험이 이미 초록이다. 사람 부분은 merge 단계를 SIGKILL 로 여러 자리에서 끊고 다른 노드를
  띄워 3.13 이 잇는지 본다

---

## 4. 물음 넷

답을 `[Answer]:` 뒤에 적는다. 권장을 **A** 에 둔다. 기호 옆에 뜻을 적었다.

**내기 전에 대 본 것** — 네 흠 (앞 단계가 정한 것을 다시 묻기 · 측정 없는 근거 · 앞 유닛의 넘김과 어긋난 선택지 · 요구를 빼는
선택지) 에 물음마다 대 봤다.

```text
   흠                            대 본 결과
   이미 정한 것을 다시 묻기         IR 의 환경 변수 이름 (ENODE_IR · contract-grammar 코드) · 성공 판정 (contract-grammar 답 1 · produced manifest · merged) ·
                                 대기 상한의 기본값과 표기 (결정 3-11 · 기본 4시간) · 스크립트라는 자리 (유닛 정의 7절) · 상태와 잠금의 모양
                                 (lower-state) 은 묻지 않는다.  물음 1 은 유닛 정의가 이 단계에 넘긴 「원인 코드」 · 물음 2 는
                                 merge-rules 가 넘긴 「되풀이되는 오류」 · 물음 3 은 ADR-077 §2 (굽기 계약) 가 안 적은 차례 · 물음 4 는 유닛
                                 정의가 자리만 정한 「운영 lower」의 뜻이다
   측정 없는 근거                  물음마다 2절의 번호나 코드 줄을 단다.  측정하지 못한 것은 2.6 에 적었다
   넘김과 어긋난 선택지             물음 1 의 A 는 contract-grammar 흐름 3절의 「merge 는 needs 를 따라 온다」를 IR 갈래에서만 바꾼다.
                                 그 절은 알리는 법을 bake 에 넘겼으므로 넘김 안이다 — 선택지 안에 그 차이를 적었다.
                                 물음 2 의 A · C 는 결정 3-13 (시작할 때 재개) 에 더할 뿐 빼지 않는다
   요구를 빼는 선택지               모든 선택지가 완료 조건 10 (HEAD 의 태그와 커밋이 결과에 보인다) · FR-8 (시작할 때 재개) ·
                                 결정 3-18 (하나라도 실패하면 합치지 않는다) · 완료 조건 8 (운영 lower 면 멈춘다) 을 지킨다.
                                 물음 4 의 B 는 목록이 비었으면 멈추게 적어 운영 lower 를 통과시키는 갈래를 막았다
```

### Question 1 — IR 이 어긋났을 때의 보고 (원인 코드 · HEAD 의 태그와 커밋의 자리 · 단계의 끝)

유닛 정의 7절이 「IR 대조의 원인 코드」를 이 단계에 넘겼다. 완료 조건 10 (계약의 IR 과 HEAD 가 어떻게 다른지 build 결과에서
본다) · US-19 (sync 가 어디에 닿았나). 2.1 — 어긋남은 두 갈래다 (태그가 로컬에 없다 · 다른 커밋을 가리킨다). 2.5 — 코드에 있는 원인
코드 셋(예산 둘 · lower_changed)과 설계의 둘(bake_in_progress · merge_wait_timeout)은 모두 error 와 함께 오고 단계가 FAILED 다. error 가 있으면 Run 이 곧바로 끝나고, 없으면 needs 를 따라 merge 가 온다.
`BuildManifest` 에 HEAD 의 태그 칸이 없다.

세 선택지에 공통 — builds 를 안 돈다 · upper 는 trash · committed · last_attempt (reason `ir_mismatch`) · `$OUT/manifest` 없음 ·
build 칸에 sync 의 기록과 head · 단계 로그에 한 줄. 문장 둘 —
`ir <IR> is not in the local repository after sync; the sync command must fetch that tag (HEAD is <커밋>, tags at HEAD: <목록>)` ·
`ir <IR> points at <커밋 1>, but sync left HEAD at <커밋 2> (tags at HEAD: <목록>)`.

A) **FAILED · reason `ir_mismatch` (새 wire 값) · error 에 위 문장 · build 칸에 새 칸 `head_tags`** (더하기만 · step-phase 의 규칙).
   Run 이 곧바로 FAILED 로 끝나고 merge 가 오지 않는다 — bake_in_progress · lower_changed 처럼 노드가 스스로 멈춘 갈래와 같은 모양.
   contract-grammar 흐름 3절이 그린 「merge 가 needs 를 따라 와서 합칠 것이 없다」는 명령 실패 갈래(3.7)에만 남는다

B) DONE · error 없음 · exit 0 (sync) · reason `ir_mismatch` · `head_tags` · 문장은 단계 로그 끝. merge 가 needs 를 따라 와서 합칠
   것이 없다 (3.8 첫 갈래) — 명령이 0 아닌 exit 로 끝난 갈래와 같은 모양

C) A 에서 `head_tags` 칸을 두지 않는다 — 태그는 error 문장에만 (`contract/result.go` 를 안 고친다)

D) Other (please describe after [Answer]: tag below)

**권장 A.** 2.5 — 원인 코드가 있으면 노드가 멈췄다는 한 규칙으로 읽힌다 (예산 둘 · lower_changed · 설계의 bake_in_progress ·
merge_wait_timeout). merge 단계가 할 일 없이 Record 에 남지 않는다. `head_tags` 는 「어디에 닿았나」를 기계가 읽는 자리다 — 조각 6 (굽기가 끝까지 돈다)
스크립트가 문장을 풀지 않고 본다. 대가 — Mediator 의 어휘 한 줄 (`store/claim.go:963`) 과 `contract/result.go` 한 칸이 행렬 밖이다.

[Answer]: B (2026-09-27T12:54:46Z 다시 답함 — 처음 답은 A 였다. A 는 정본 불변식 I3 과 부딪치고, 병합된 internal/contract/bake.go:34-35 가 IR 어긋남을 manifest 없음으로 계약 조건이 판정하게 두었다 — 마지막 QA)

### Question 2 — 끊긴 합치기 (merging) 를 누가 언제 잇나

결정 3-13 (끊긴 합치기는 같은 lower 의 어느 노드든 시작할 때 잇는다) · FR-8 (기능 8 · lower 상태와 재개) · ADR-077 §7 (굽기끼리의 배타와 재개). 이것이 덮지
못하는 갈래 셋이다.

```text
   (1) merge 단계의 Apply 가 오류로 멈춤       merge-rules 규칙 9절 — 첫 오류에서 멈춘다.  데몬은 산다 — 「시작」이 안 온다
   (2) 재개가 같은 오류로 되풀이               merge-rules 가 넘긴 일 (예 — 매핑 밖 uid 로 chown)
   (3) build claim 이 낡은 merging 을 만남      ADR-077 §7 은 「낡은 상태 정리를 먼저 하고 진행」이다.  재개는 배타를 기다린다 —
                                            형제 Run 이 끝날 때까지 (ADR-077 §12 의 빌드 1,892 ~ 4,760 초)
```

어느 갈래든 lower 는 merging 에 남고 형제는 drain 이다. merging 을 committed 로 돌리는 길은 없다 (3.1). 재개의 비용 — Preflight 가
하루치 upper 에 0.23 초 (SunnyVM) · 0.63 초 (이 기계 · merge-rules code-summary 6절) · 처음부터 굽기의 upper 에 3.95 초 (ADR-077 §12
의 금지 표시 검사). 굽기 잠금을 해 보는 것은 flock 하나다 (lower-state 의 `BeforeAdvert` 25 µs 안에 든다).

A) **시작할 때 + 살아 있는 동안.** runc-overlay 노드의 광고 주기가 state 가 merging 이고 굽기 잠금이 비어 있으면 (TryBake 가 되면)
   배경에서 재개를 연다. 실패하면 잠금을 놓고 그 노드는 10분 뒤 다시 해 본다 — 오류가 바뀔 때만 로그 한 줄. merge 단계의 Apply 가
   멈추면 FAILED (error 에 호출과 경로 · 원인 코드 없음) 로 보고하고 두 잠금을 놓는다 — 다음 광고에서 보통 그 노드 자신이 잇는다.
   build claim 이 낡은 merging 을 만나면 같은 재개를 배경에 열고 build 는 bake_in_progress 로 끝난다 (ADR-077 §7 의 「정리를 먼저
   하고 진행」을 merging 에서만 바꾼다 — 정본 되돌림)

B) 시작할 때만 (결정 3-13 글자 그대로). merge 단계의 Apply 오류는 FAILED · 두 잠금을 놓고 merging 에 둔다. 사람이 원인을 고치고
   그 lower 의 노드 하나를 다시 띄운다. build claim 이 낡은 merging 을 만나면 ADR-077 §7 대로 그 자리에서 재개한 뒤 build 를 잇는다
   (배타를 기다린다 · 상한 없음)

C) merge 단계가 두 잠금을 쥔 채 제자리에서 다시 해 본다 (1 · 2 · 4 … 분 · 10분 간격에서 멈춤 · 끝없이). 단계가 끝나지 않고 Run 이
   RUNNING 에 남는다. 시작 때 재개와 build claim 은 A 와 같다

D) Other (please describe after [Answer]: tag below)

**권장 A.** 재개는 여러 번 해도 결과가 같다 (merge-rules 의 재개 시험 41 자리 · SunnyVM 44 자리). 사람 없이 풀리는 오류 (여유
공간 · 다시 쓸 수 있게 한 마운트)가 데몬 재시작을 기다리지 않는다. 10분 간격이면 하루치 upper 의 재시도는 시간당 1.4 ~ 3.8 초다.
B 는 원인을 고친 뒤에도 누가 노드를 다시 띄울 때까지 형제가 모두 drain 이고, build claim 이 형제 Run 을 기다리는 동안 굽기 Run 의
임대가 묶인다. C 는 굽기 Run 이 결말을 못 낸다.

[Answer]: A

### Question 3 — build 하나가 실패하면 남은 builds 를 돌리나

ADR-077 §2 (굽기 계약) · FR-6 (build 는 upper 에 짓는다) · 결정 3-18 (실패하면 합치지 않는다) — 하나라도 0 이 아니면 build 단계는 실패이고 합치지 않으며 upper 는 trash 로 간다. 남은 구성을
돌릴지는 적혀 있지 않다. sync 가 실패하면 builds 는 안 돈다 (3.3). ADR-077 §12 (SunnyVM) — 구성 하나의 증분 빌드 1,892 초 · 처음부터
4,760 초.

A) **첫 실패에서 멈춘다.** build 칸에는 돈 명령만 적는다 · 로그에 건너뛴 구성마다 한 줄 (`bake: skipping config-b; config-a failed`)

B) 끝까지 돈다 — 구성마다 성패를 한 번에 본다. 합치지 않는 것과 upper 가 trash 로 가는 것은 같다

C) Other (please describe after [Answer]: tag below)

**권장 A.** 실패한 굽기의 upper 는 trash 로 가므로 뒤 구성의 산출물과 캐시가 남지 않는다. 구성 하나에 30 ~ 80 분 동안 그 노드의
임대가 묶이고 형제는 영향이 없다 — 얻는 것은 성패 한 줄뿐이다. 다른 구성의 성패가 궁금하면 굽기 계약을 다시 낸다.

[Answer]: A

### Question 4 — 조각 스크립트가 「운영 lower」를 어떻게 알아보나 (완료 조건 8 · US-13)

유닛 정의 7절 — 합치기 조각(6 · 7 · 8)의 스크립트가 굽기를 내기 전에 대상 lower 를 출력하고 운영 lower 면 멈춘다. lower-state 가 넘긴
규칙 「한 lower 의 노드를 모두 새 판으로 올린 뒤 굽는다」도 이 확인에 둔다. 2.4 — SunnyVM 에 옛 판 (2026-09-22) 데몬 둘이 떠 있다.
코드 — `GET /v1/nodes` 가 노드마다 능력 (`machine` · `ws` · `workspace.writes`) 을 준다 (`store/observe.go:133` · `detect.go:65` ·
`:78`). `workspace.writes` 가 없으면 lower-state 전의 판이다 (2.4).

세 선택지에 공통 — 스크립트는 그 기계에서 돈다 · 대상 노드를 `ws` 로 고른다 (`slice-4.sh` 모양) · 대상 lower 의 경로 · 장치 · inode 를
출력한다 · 같은 `machine` 에서 `ws` 가 같은 장치 · inode 인 노드 (bind 별칭 포함) 가 모두 `workspace.writes` 를 광고해야 돈다 —
하나라도 없으면 옛 판이 있다고 말하고 멈춘다.

A) **허용 표지.** lower 뿌리에 사람이 둔 빈 파일 `.enode-disposable` 이 있어야 돈다. 없으면 운영 lower 로 보고 멈춘다

B) 금지 목록. 환경 변수 `PROTECTED_LOWERS` (콜론으로 나눈 경로) 가 비었으면 멈추고, 대상이 그중 하나와 같은 장치 · inode 면 멈춘다

C) 확인 입력. 대상 lower 경로를 출력하고 사람이 그 경로를 그대로 쳐야 돈다

D) Other (please describe after [Answer]: tag below)

**권장 A.** 모르는 lower 를 운영으로 본다. 조각 6 (굽기가 끝까지 돈다) 의 lower 는 사람이나 스크립트가 새로 만드는 버려도 되는 자리 (`~/yocto-fresh`
모양) 라 만들 때 표지를 함께 둔다. B 는 목록에서 빠진 운영 lower 를 통과시킨다. C 는 `PAUSE=0` 으로 돌릴 수 없고 사람의 눈에만
기댄다. 표지는 lower 뿌리의 파일이라 합친 목록 비교(조각 6 의 ⑥ · 합친 lower 와 merged view)의 양쪽에 똑같이 있다.

[Answer]: A

---

## 5. 산출물 계획 (체크박스)

답이 들어오고 모호함이 풀린 뒤에 채운다. 자리는 `aidlc-docs/v4-run-finalize-bake/construction/bake/functional-design/` 이다.

- [x] 답을 읽고 모호함을 확인한다 — 있으면 되물음 파일을 만든다 (2026-09-27 · 넷 모두 A · 서로 대 봐 막히는 짝이 없어 되물음 파일 없음 ·
      대 본 짝은 `business-logic-model.md` 11.1절과 audit)
- [x] `domain-entities.md` — 노드 Step 의 새 칸 넷 · 굽기 보관 (Run · Dir · 굽기 잠금 · 대기 자리 · 단계 · 「합치기 시작」 표지 ·
      치우는 몸통) · 대기 자리와 초안 `bake.json` · IR 대조의 입력과 출력 · merge-helper 의 입출력 · 재개 · `BuildManifest` 의 새 칸
      (답 1 · IR 어긋남의 보고) · 원인 코드 (답 1 · IR 어긋남의 보고) · `MergeResult` 옮기기 · last_attempt · `merge.Check` 의 mount
- [x] `business-rules.md` — 분기와 거절 · build 의 차례 (답 3 · 실패 뒤 builds) · IR 대조 (자리 · 명령 · 두 갈래 · 답 1 · IR 어긋남의 보고) · exited · 예산 · 대기 자리 ·
      전이와 쓰는 때 · merge 의 차례 · 기다림 (시계 · 간격 · 그물) · 시작 전 확인이 어긋났을 때 · 치우는 몸통 하나 · HoldBake ·
      DropBake · 기동과 재개 (답 2 · 재개의 때) · last_attempt · 결과 칸 · 오류와 로그 문구 (영어)
- [x] `business-logic-model.md` — build · merge · 기동 · 재개 흐름 · 실패 갈래 표 · 시험 모양 (조각 5 굽기 계약의 기계 시험 · integration · 조각 6 · 7 · 8 (사람 조각)
      스크립트와 운영 lower 확인 · 답 4 · 운영 lower) · 파일 행렬 밖 자리 (`internal/merge` 의 마운트 줄 · `contract/result.go` 와 Mediator 어휘 ·
      답 1 (IR 어긋남의 보고) 에 따라) · 다른 유닛에 넘기는 것 (checkpoint — 실패한 build 단계의 upper 를 spool 로 받을지) · 정본 되돌림 (ADR-077 §4 합치기 · §5 상태와 metadata ·
      §6 기다림 · §7 재개 · run-contract 의 원인 코드 · mediator-api 의 result 칸)
- [x] 커버리지 — `internal/enode` 84.5% · `cmd/enode` 80.5% (lower-state 뒤). 규칙은 순수 함수로 떼고, 조립은 `internal/enode` 의
      함수 하나 · main.go 에는 부르는 줄만
- [x] 표기 검사 (`enode-design/scripts/emphasis-check.py`) · 말투 검사 · 사내 이름 검사 (새 문서 셋과 고친 문서 모두)

---

## 6. 확장 준수 — 이 단계

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 계약 명령 줄 (격리 runtime 안에서만) 은 3.3 · 3.4 가 세션 안에서 도는 것으로 닫고, 확인은 이 유닛의 NFR 이다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 한 줄 — 저장소의 관례다 |

유닛 정의는 이 유닛의 NFR Requirements 를 「한다 (최소)」로 적었다 — 보안 (계약의 명령이 격리 실행 환경 안에서만 돈다) · 성능 (형제가
기다리는 시간은 합치기가 아니라 형제 Run 이 정한다). Functional Design 이 닫힌 뒤에 정한다.
