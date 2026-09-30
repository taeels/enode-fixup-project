# `checkpoint` — 흐름

칸은 `domain-entities.md` (「엔티티 N절」), 규칙은 `business-rules.md` (「규칙 N절」) 에 있다. 여기는 차례와 자리, 시험의 모양,
넘기는 것이다. **답 N** 은 계획 물음 N 의 답 (모두 A) 이다.

---

## 1. 단계의 닫기 — `Worker.closeOut` (`claim.go:867`)

```text
   Finalize                                  오늘 그대로 [Finalize 예산 시작 = exited_at]
   diagnostics 를 짓는다                        Close 뒤 (claim.go:886) 에서 앞으로 옮긴다 — 빠진 산출물을 규칙 1절이 본다
   Keeper.Decide(사실)                        규칙 1절 (요구하나) -> 규칙 2절 ② ~ ④.  걸리면 그 상태로 끝나고 Keep{} 로 닫는다
     Reserve                                 ID · <spool>/<ID>/ · 항목 잠금 · reserved 기록 (신원 칸 · 규칙 11절)
   session.Close(Keep{Upper, By, Result})    helper 가 끝난 뒤 By 를 보고 rename 한 번 (RENAME_NOREPLACE).  작업 폴더는 trash
   Keeper.Finish(예약, KeepResult)           규칙 2절 ⑤ ~ ⑦ — kept 기록 · 잠금 놓기, 또는 버림 (항목 폴더째 trash)
   after (굽기) · finalized_at               오늘 그대로 [Finalize 예산 끝]
   업로드 · settle                            오늘 그대로.  settle 은 KeepResult 를 모른다 (규칙 3절)
   res.CheckpointCapture · Diagnostics.Checkpoint
```

- **보고 전 창에 더하는 것** — stat 둘 (filesystem 번호) · statfs 하나 · mkdir · flock · 작은 기록 쓰기 둘 · rename 하나. 모두 upper
  크기와 무관하다 (결정 2-4 — 임대 창에서는 O(1) 소유권 이전만)
- **기록은 임시 파일과 rename 으로 쓰고 fsync 하지 않는다** — 보고 전 창에 디스크 대기를 들이지 않는다. 전원이 나가 기록을 잃으면
  조정이 주인 없는 미완료로 거둔다 (규칙 9절). 보존본을 잃을 뿐 단계 결과는 그대로다
- **예약의 자리** — 쓰기 부하에서 예약이 4.2 초 멈췄다. 보존이 finalize 판정을 바꾸지 않게 할 모양을 NFR Design 이 정할 때 바뀐다
  (`nfr-requirements.md` 2절 P2 · 6절 D1). 지금 모양은 그대로다 (NFR P2 로 고침)
- **`closing` 에 두 칸** (엔티티 6절) — 부르는 쪽이 아는 실패 (명령의 exit ≠ 0 · signal, agent 의 미완주) 와 굽기 build 인지. 빠진
  산출물과 Finalize 오류는 closeOut 안에서 더한다

---

## 2. 굽기 build 의 끝 (답 3)

| 끝 | 오늘 | 바뀌는 것 |
|---|---|---|
| `failEnd` (`bake_build.go:388`) — 명령 실패 · IR 어긋남 · IR 대조를 못 함 · pinned 실패 | `closing{after}` · `Keep{}` | `closing{after, failed: true, bake: true}` — 1절의 흐름을 탄다. upper 는 spool 이나 trash |
| `succeed` — 대기 자리 | `closing{keep: Keep{Upper: pending}}` | 그대로. `checkpoint_capture` 는 `not_requested` (규칙 1절) |
| `stopEarly` (`:338`) | `Close(Keep{})` | 그대로 · 칸 없음 |
| merge 단계 | 세션 없음 | 그대로 · 칸 없음 |

bake 규칙 3 (첫 실패에서 멈춤) 은 그대로다. 보존본은 첫 실패 자리의 트리를 담고 lower 에 합치지 않는다. 그 규칙의 근거 문장
(bake `business-rules.md:118-120` 「실패한 굽기의 upper 는 trash 로 가므로」) 만 거짓이 된다 — 진행자에게 넘긴다 (11절).

---

## 3. 보고와 보고 뒤

```text
   Worker.report                              네 끝 가운데 하나 (claim.go:405-433)
   Keeper.Reported(ID, 성패)                   captured 였을 때만.  기록의 report 칸 (규칙 8절)
   AfterReport                                삭제자 Kick (오늘) · Keeper Kick
   판정 고루틴 (Keeper.Run)                    spool 잠금 -> 규칙 6절의 일곱 차례 -> 잠금 놓기
     측정                                     trash-helper 의 측정 입구를 spool 에 연다 (namespace 안 · 5절)
     퇴출 · 만료                              Trash.Move -> 삭제자 Kick
     상태 파일                                 StatusBook.SetSpool (바뀌었을 때만)
```

---

## 4. 기동 — `cmd/enode/main.go`

```text
   설정 읽기 · 검증                            규칙 10절.  틀리면 노드가 안 뜬다
   런타임                                     native 면 Keeper 없음.  상태 파일에 checkpoint 블록 (unsupported: runtime) · 로그 한 줄
   runc-overlay
     SweepOrphanSessions · 삭제자              오늘 그대로 (main.go:317-324)
     StartCheckpoints                          Keeper 를 짓고 상태 파일의 checkpoint 블록 · 기동 로그 한 줄 (규칙 12절)
     Keeper.Run (고루틴 하나 더)                 조정 (규칙 9절) -> 판정 -> 깸을 기다린다 (보고 뒤 · 10분)
   광고 시작                                   조정을 기다리지 않는다 — 삭제자 첫 회와 같다 (trash 답 10)
```

조정 전에 단계를 집어도 된다 — 예약은 spool 잠금을 안 쓰고, 받아들임의 보존 총량이 0 이면 문이 느슨할 뿐 판정이 뒤에 막는다
(규칙 5절). `services.md` 4절의 차례 (삭제자 첫 회 → 조정 → 광고) 는 시작하는 차례로 지킨다.

---

## 5. 측정의 입구

spool 의 upper 에는 subordinate uid 소유 항목이 있어 노드 uid 로는 못 걷는다 (`components.md` 6절). 삭제자와 같은 매핑의 helper 가
걷는다 — `enode trash-helper` 에 측정만 하는 동작을 더하고, `scratch.Measure` 가 trash 가 아닌 뿌리 (spool) 도 받게 한다. 경계는
그대로다 (symlink 를 안 따라감 · 다른 filesystem 에 안 들어감). argv 의 모양은 Code Generation 이 정한다.

---

## 6. 조회 명령

`cmd/enode/main.go` 는 `checkpoint` 를 `enode.RunCheckpointCmd(args, stdout, stderr)` 로 넘기는 줄만 둔다 (`RunTrashHelper` 와 같은
모양). 설정 찾기 (`ResolveConfig`) · scratch 찾기 · 기록 읽기 · 출력은 `internal/enode` 의 새 파일에 있다 (규칙 13절).

---

## 7. 시험

**기본 `go test` (CI)** — 규칙마다 표 시험 한 줄 (저장소의 관례).

| 자리 | 본다 |
|---|---|
| `internal/scratch` | 판정 차례 ② ~ ④ · ID 와 겹침 · 기록의 읽기와 쓰기 · 보고 뒤 판정 (만료 · 하나의 상한 · 몫은 오래된 것부터이고 새 것도 · inode · 측정 실패 · `LaunchError`) · 조정의 다섯 줄 (살아 있는 형제의 잠금 포함) · spool 칸 |
| `internal/enode` | 규칙 1절 표 (명령 · agent · 굽기 build) · 가짜 세션으로 closeOut — `KeepResult` 셋 (옮김 · 늦음 · 실패) 마다 receipt 와 diagnostics 가 맞고 finalize 칸 · error · exit 은 그대로 · 다른 Close 오류는 finalize 칸 error · `Keep.Result` 가 없는 굽기 keep 은 오늘 그대로 · 보고의 네 끝 · 설정 검증 표 · 조회 출력 · 어휘 두 벌이 같다 (엔티티 8절) · diagnostics 문장에 경로가 없다 |
| `runc_overlay_linux_test.go` | 가짜 helper 로 Close 의 keep — `By` 가 지났으면 옮기지 않음 · rename 실패면 `Err` 이고 upper 는 trash · 결과를 Close 오류에 안 싣는다 |
| `internal/panel` | spool 줄 · 정책 줄 · off · native 문구 |

**integration 태그 (사람 · SunnyVM · CI 밖)** — 실제 overlay upper (subordinate uid 항목 · whiteout 포함) 를 spool 로 옮기고 helper 가
측정한다. `show` 의 `unshare` 줄로 컨테이너 root 가 쓴 0600 파일이 읽힌다 — 계획 2.2 가 이 기계에서 확인하지 못한 것이다.

**조각 9 (보존 · 사람 · SunnyVM)** — `scripts/finalize-bake/slice-9.sh`. 장면 3 (실패를 들여다본다) 의 1 ~ 4 와 조각 9 의 줄.

| # | 하는 일 | 보는 것 |
|---|---|---|
| 1 | on-failure overlay 노드에 exit 1 단계와 exit 0 단계 | 앞은 `captured` 와 ID · 뒤는 `not_requested` |
| 2 | `enode checkpoint list · show` | `kept` · 보고 뒤 크기 · `delivered` |
| 3 | `max_gb: 1` 로 그보다 큰 upper | `evicted: larger than max_gb` · receipt 는 `captured` 그대로 |
| 4 | `ttl_hours: 1` | 한 시간과 10분 안에 목록에서 없음 · Record 는 `captured` · `show` 는 exit 1. **조각이 한 시간 넘게 걸린다** |
| 5 | native 노드에 exit 1 단계 | `unsupported`(`runtime`) |
| 6 | 단계 자신이 여유를 `min_free_gb` 아래로 채우고 exit 1 | `rejected`(`free_space`). 미리 채우면 노드가 drain 해 단계를 못 집는다 |
| 7 | 같은 실패 계약을 `policy: off` 와 on-failure 로 | exit_code · produced · finalize 칸이 같다 |
| 8 | 주인 없는 `reserved` 항목을 만들고 데몬 재시작 | 그 항목이 trash 로 가고 삭제자가 지운다 |

---

## 8. 코드 자리와 커버리지

| 파일 | 하는 일 |
|---|---|
| `internal/scratch/checkpoint.go` (새) | Policy · 판정 ② ~ ④ · 예약 · 확정 · 버림 · 기록 · 보고 뒤 판정 · 조정 · 목록 · spool 칸 |
| `internal/enode/checkpoint.go` (새) | Keeper — 규칙 1절 · 신원 모으기 · `scratch.Capture` 를 `contract.CheckpointCapture` 로 · 기동 조립 (`StartCheckpoints`) |
| `internal/enode/checkpoint_cmd.go` (새) | `RunCheckpointCmd` — 조회 |
| `claim.go` · `runtime.go` · `runc_overlay_linux.go` · `runc_overlay_other.go` · `config.go` · `status.go` · `panel/view.go` · `cmd/enode/main.go` | 행렬의 ⑧ 칸 (계획 1절 15번) |

flock · `Renameat2` · statfs 를 쓰는 자리는 `_unix` (또는 `_linux`) 와 `_other` 짝이다 — `_other` 는 지원 안 함을 돌려준다 (유닛 정의
10절 · 크로스 빌드 셋). 규칙은 `internal/scratch` 의 순수 함수에 두고, 조립은 `internal/enode` 의 함수 하나, `main.go` 에는 부르는 줄만 둔다 — `cmd/enode` 가
80.8% (하한 80%) 라서다.

---

## 9. 파일 행렬 밖

| 파일 | 까닭 |
|---|---|
| `internal/contract/result.go` | `Diagnostics.Checkpoint` 한 칸 (답 9 · 더하기만). Mediator 는 코드를 안 고치고 다시 짓는다 |
| `internal/enode/bake_build.go` | `failEnd` 의 `closing` 두 칸 (답 3) |
| `internal/panel/page.go` | spool 줄과 정책 줄 (답 5). trash · lower-state 도 고친 파일이다 |
| `internal/enode/trash_linux.go` · `trash_other.go` | trash-helper 의 측정 입구 (5절) |
| `internal/scratch/remove_unix.go` | `Measure` 가 spool 뿌리도 받는다 (5절) |
| `internal/enode/checkpoint.go` · `checkpoint_cmd.go` (새) | 행렬의 ⑧ 칸에 `internal/enode` 새 파일이 없다 |

---

## 10. 고친 회차 문서

| 문서 | 줄 | 고친 것 |
|---|---|---|
| `inception/application-design/component-methods.md` | 295 | `MaxInodes` 가 보존본 전체의 inode 한도라고 적었다 (규칙 6절) |
| 같은 문서 | 318-321 | `Store.Keep` (Store 가 rename 한다) 을 `Reserve` · `Commit` 으로 — rename 은 세션의 Close 가 한다 |
| 같은 문서 | 417 | result 의 보존 칸을 `CheckpointCapture *contract.CheckpointCapture` 로 (step-phase 가 지은 타입) |
| 이 폴더 `business-rules.md` (NFR 뒤 · 2026-09-30) | 2절 ② · ⑤ · 6절 4 · 6 · 9절 · 10절 · 11절 · 14절 | spool 자리 거절 (NFR 답 3) · 예약 자리 (P2) · inode 한도 (답 2) · 미완료 조정 (R2) · 기본값 (답 1 · 2) · 기록 읽기 (C7) · 로그 두 줄 (답 3) |
| 이 폴더 `domain-entities.md` (NFR 뒤) | 1절 | N2 의 값 (NFR 답 1 · 2) |
| 이 폴더 `business-logic-model.md` (NFR 뒤) | 1 · 11 · 12절 | 예약 자리 (P2) · NFR 이 받은 넘김 · 정본 되돌림의 이어짐 |

---

## 11. 넘기는 것

| 누구 | 무엇 |
|---|---|
| NFR Requirements | (받았다 — `nfr-requirements.md` · 2026-09-30) N2 — `max_gb` · `max_total_inodes` 의 기본값 (ADR-076 §10 — 하루치 upper 9 GB · 처음부터 23 GB). 보안 — 0700 · 0600 · 경로가 나가는 자리 셋 (규칙 11절) · `unshare` 안내 · 자격증명 캐시 안내. 성능 — 1절의 보고 전 창에 더한 것 · fsync 없음 · 판정이 spool 잠금을 쥔 채 측정한다 · 형제와 나눈 spool 에서 받아들임이 느슨하다 |
| Code Generation | 9절의 행렬 밖 · `Keep` 의 두 칸 · `report` 의 네 끝 · `diagnosticsFor` 를 Close 앞으로 · 측정 입구의 argv · 다섯째 고루틴 · `slice-9.sh` · 커버리지 병합 전과 뒤 (`internal/enode` · `cmd/enode` · `internal/scratch`) |
| 조각 9 | 7절 표. 4번이 한 시간 넘게 걸린다 · 6번은 단계가 여유를 채운다 |
| 진행자 | bake `business-rules.md:118-120` 의 규칙 3 근거 문장 (2절) · 12절의 정본 되돌림 · audit 와 상태 파일 |

---

## 12. 정본에 되돌려 올리는 것 (진행자가 올린다)

| 자리 | 한 줄 |
|---|---|
| ADR-076 §2 | 판정 차례 — 정책 → 런타임 → 예산 → 여유 → 총량 → 옮기기 → 확정. 요구하지 않았으면 native 도 `not_requested` (답 2 · 규칙 2절) |
| ADR-076 §4 | on-failure 의 실패는 노드가 닫기 전에 아는 것이다 — exit ≠ 0 · signal · 하네스 미완주 · 빠진 산출물 · Finalize 오류. 성공 판정 (I3) 과 무관하다 (답 1) |
| ADR-076 §4 · ADR-077 | 굽기 build 는 build 끝의 실패만 보존한다. 성공한 upper 와 대기 자리 뒤와 merge 단계는 보존하지 않는다 (답 3) |
| ADR-076 §4 | spool 로 옮기기의 실패만 `failed`(`io`) 로 떼고, 다른 Close 오류는 오늘처럼 단계의 오류다 (규칙 3절) |
| ADR-076 §4 끝 | 「보고가 거절되거나 임대가 만료되면 정책에 따라 만료 · 정리」의 정책은 TTL 이다. store 가 보고의 성패를 적는다 (답 10) |
| ADR-076 §4 | 「세션이 끝나기 전 운영자의 명시 요청」 정책은 짓지 않았다 — FR-10 에 없다. 여는 조건은 요청을 보낼 표면이 요구될 때 |
| ADR-076 §5 | inode 한도는 보존본 전체에 건다. 하나의 상한은 바이트뿐이다 (계획 3절 5번) |
| ADR-076 §10 | 설정 키 `policy` · `ttl_hours` · `capacity_percent` · `max_gb` · `max_total_inodes` (답 4) |
| ADR-076 §10 | descriptor 는 receipt 의 칸 일곱과 `diagnostics.checkpoint` 문장 (답 9). ID 는 무작위 12 hex · 한 노드의 spool 안에서 유일 (답 8) |
| ADR-076 §10 | 조회는 노드의 `enode checkpoint list \| show` · Mediator API 없음 · 퇴출 기록은 만료까지 (답 6 · 7) |
| ADR-076 §10 | reason 의 wire 표기는 여섯 글자 그대로 · 새 코드는 더하기만 · Mediator 는 원인을 확인하지 않는다 (`store/claim.go:965`) |
| ADR-075 §6 · mediator-api 의 result | diagnostics 에 `checkpoint` 문장 한 칸 (답 9) |
| (NFR 이 더한 넷) | `nfr-requirements.md` 8절 — N2 의 값 · spool 자리 · fsync 없음 · 창의 비용 |

---

## 13. 추적 — 계획의 항목이 들어간 자리

| 계획 | 자리 |
|---|---|
| 받는 일 1 (Store) · 11 (닫기 안 · 예산) · 12 (퇴출 · Usage) | 엔티티 3 · 4 · 7절 · 규칙 5 · 6절 · 흐름 1 · 3절 |
| 받는 일 2 (보존 지원) · 17 (크로스 빌드) | 엔티티 5절 · 흐름 8절 |
| 받는 일 3 (receipt) · 4 (diagnostics) · 9 (captured 의 뜻) | 엔티티 2절 · 규칙 2 · 13절 |
| 받는 일 5 (설정) · 8 (기본값을 적는 자리) | 엔티티 1절 · 규칙 10 · 12절 |
| 받는 일 6 (조회) | 규칙 13절 · 흐름 6절 |
| 받는 일 7 (양) | 엔티티 7절 · 규칙 12절 |
| 받는 일 10 (outcome 불변) · 13 (RENAME_NOREPLACE · 첫 Close) | 규칙 3절 |
| 받는 일 14 (굽기 build) | 규칙 1절 · 흐름 2절 |
| 받는 일 15 (이어 고칠 파일) | 흐름 8 · 9절 |
| 받는 일 16 (기동 차례) | 규칙 9절 · 흐름 4절 |
| 받는 일 18 (보안) | 규칙 11절 |
| 받는 일 19 (조각 9) · 20 (NFR) | 흐름 7 · 11절 |
| 3절 1 · 2 · 3 · 4 | 규칙 2 · 3 · 4 · 5절 |
| 3절 5 · 6 · 7 | 규칙 6 · 7 · 9절 |
| 3절 8 · 9 | 규칙 10절 · 흐름 12절 (운영자 요청) |
| 3절 10 · 11 · 12 | 규칙 11절 · 엔티티 3 · 8절 |
| 3절 13 · 14 · 15 | 규칙 13 · 4절 · 흐름 7절 |
| 답 1 · 2 · 3 | 규칙 1 · 2절 · 흐름 2절 |
| 답 4 · 5 | 엔티티 1 · 7절 · 규칙 10 · 12절 |
| 답 6 · 7 · 8 | 규칙 7 · 13절 · 엔티티 4절 |
| 답 9 · 10 | 엔티티 2 · 3절 · 규칙 2 · 8절 |

---

## 14. 확장 준수 — Functional Design

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 spool 줄은 규칙 11절이 닫고 확인은 NFR 이다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 (7절) |
