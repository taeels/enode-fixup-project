# 정본 — `enode-design`과 이 팩의 연결

설계 정본은 이 저장소의 서브모듈 `enode-design/`이다. **어긋나면
`enode-design/protocol/INVARIANTS.md`가 이긴다.**

| 확인 | 명령 |
|---|---|
| 서브모듈 고정 | `git ls-tree HEAD enode-design` → `369270a` |
| 정본 브랜치 | enode-design `unit/runtime-environment-profile`. ADR-075~077은 enode-design `main`에 아직 없다 |
| 갱신 | `git submodule update --remote`는 진행자만 |

아래 링크는 이 저장소 안의 서브모듈 경로다. GitHub에서는 서브모듈 안 경로가 열리지 않으므로
웹에서 볼 때는 `https://github.com/taeels/enode-design/blob/369270a/<경로>`로 연다.

---

# 1. 이 팩이 딛는 정본

| 문서 | 상태 | 이 팩에서 쓰는 자리 |
|---|---|---|
| [ADR-075](../../enode-design/adr/ADR-075-a-step-returns-a-commit-set-not-a-workspace.md) 단계는 commit set을 반환한다 | 결정 · 개정 (2026-09-23) | §4 봉투 · §5 actor와 effect · §8 discovery · §9 Finalize · §10 결정 7 종료 보고와 phase와 예산 · §15 검증 장면 · §16 이월 |
| [ADR-076](../../enode-design/adr/ADR-076-a-checkpoint-preserves-runtime-state-not-a-step-result.md) checkpoint는 결과가 아니다 | 초안 | §2 상태와 `reason` · §4 임대 창 · §4.1 trash · §5 spool과 보존 판정 · §9 검증 장면 · §10 기본값 |
| [ADR-077](../../enode-design/adr/ADR-077-a-bake-builds-in-the-upper-and-merges-into-the-lower.md) 굽기는 upper에서 짓고 lower에 합친다 | 초안 | 전체. §12 실측이 합치기 비용을 잰다 |
| [ADR-073](../../enode-design/adr/ADR-073-the-profile-prepares-the-environment-the-runtime-runs-the-step.md) profile이 환경을 준비하고 runtime이 단계를 돌린다 | 초안 · 구조 수락 | `StepRuntime` 수명 · 코드 기준선 · §11 잔여 |
| [ADR-072](../../enode-design/adr/ADR-072-the-node-says-which-ir-it-stands-on.md) 노드가 어느 IR 위에 섰는지 말한다 | 초안 | §6.4 `repo.built` 모양 · §8 미결 · §9 결정 조건 |
| [ADR-070](../../enode-design/adr/ADR-070-a-node-is-a-machine-and-a-workspace.md) 노드는 (머신, 워크스페이스)다 | 초안 | §4 준비도 Run · §5.5 창 · §5.6 갈래 나 |
| [ADR-074](../../enode-design/adr/ADR-074-windows-isolation-is-a-host-sandbox-not-a-rootfs.md) Windows 격리 | 초안 | 이 팩 밖. 순서만 딛는다 |
| [ADR-063](../../enode-design/adr/ADR-063-the-node-says-what-it-is-the-owner-says-who-may-use-it.md) drain | 결정 | graceful drain과 후보 제외 |
| [ADR-037](../../enode-design/adr/ADR-037-the-predicate-must-not-be-authored-by-the-agent.md) 판정 증거 | 결정 | `Result.Changed`를 유지한다 |
| [ADR-030](../../enode-design/adr/ADR-030-lost-answers-return-when-asked-again.md) 재시작 판정 | 결정 | 종료 보고의 키, 재개 뒤 원래 Run의 실패 |
| [ADR-032](../../enode-design/adr/ADR-032-ask-is-a-step-the-answer-is-an-artifact.md) ask는 단계다 | 결정 | 내장 단계 kind의 선례 |
| [ADR-016](../../enode-design/adr/ADR-016-heartbeat-carries-the-lease.md) 하트비트가 임대를 나른다 | 결정 | 임대는 풀지 않는다 |
| [ADR-018](../../enode-design/adr/ADR-018-mediator-issues-keys-not-bytes.md) 직접 전송 | 결정 | 큰 산출물 업로드가 나갈 자리 |
| [mediator-api](../../enode-design/protocol/mediator-api.md) | 초안 | `POST /v1/steps/{id}/exited` · result의 `exited_at`, `finalize`, `upload` · 진행 조회의 phase |
| [INVARIANTS](../../enode-design/protocol/INVARIANTS.md) | 정본 | `I1` · `I4` · §1.2 상태를 늘리려면 ADR을 먼저 쓴다 |

---

# 2. 정본이 코드와 어긋나 있는 자리

| 정본 | 코드 (`unit/runtime-environment-profile`) |
|---|---|
| ADR-075는 결정이다 | 구현이 0이다. `claim.go:692-693`이 `RecordDiff`와 `Discover: true`로 수확하고 종료 보고, phase, 예산, `effect`가 없다 |
| ADR-076 §4.1 trash | `runRoot`를 `RemoveAll`한다 |
| ADR-077 | 구현이 0이다. 시제품은 SunnyVM의 `merge.py`뿐이다 |
| ADR-072 §6.4 `repo.built` | `ir`, `repo.built` 광고가 없다 |
| ADR-073 §11 「남은 것」의 E5 | SunnyVM 제품 경로에서 이미 완주했다(`EN-bf4045e7`). 문서가 늦다 |

---

# 3. 정본에 더하는 것

`decisions.md` 7절이 목록이다. 계약 문법은 `protocol/run-contract.md`에 아직 없다.
