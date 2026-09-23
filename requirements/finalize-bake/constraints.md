# 예외사항 — 이 팩이 구현하지 않는 것, 지켜야 하는 것, 부딪히는 자리

요구 문서가 말하지 않은 것을 「빠뜨린 것」이 아니라 「뺀 것」으로 읽게 한다. 구조 불변식과
접점 절은 유닛과 무관하게 참인 것이다.

---

# 1. 제외 기능 (구현하지 않음)

| 무엇 | 근거 | 여는 조건 |
|---|---|---|
| Windows `StepRuntime` (AppContainer, Job Object) | Linux 쪽이 먼저 닫혀야 한다. 실제 Windows 노드가 있어야 probe할 수 있다 | 이 팩의 게이트가 초록이고 Windows 노드에서 W1~W7을 돌릴 수 있을 때 ([ADR-074 §6~§7](../../enode-design/adr/ADR-074-windows-isolation-is-a-host-sandbox-not-a-rootfs.md)) |
| 중앙 checkpoint store, 다른 노드에서 복원 | 대용량 직접 전송과 post-step 권한, portable snapshot format이 아직 없다 | [ADR-076 §6, §8](../../enode-design/adr/ADR-076-a-checkpoint-preserves-runtime-state-not-a-step-result.md) |
| checkpoint의 `restorable` 보장 | 불변 lower generation과 `.repo` base 보증이 없다. 첫 보장은 `inspect-only`다 | ADR-076 §3 |
| Mediator가 Finalize 예산을 강제 | 노드가 스스로 지킨다 | 광고를 계속하면서 예산을 넘기는 노드가 관측될 때 ([ADR-075 §10.2](../../enode-design/adr/ADR-075-a-step-returns-a-commit-set-not-a-workspace.md)) |
| 첫 준비를 Run으로 | 빈 lower 노드를 굽기 계약이 고르는 법이 없다 | [ADR-077 §9](../../enode-design/adr/ADR-077-a-bake-builds-in-the-upper-and-merges-into-the-lower.md) |
| 한 lower를 여러 사용자나 환경이 나눈다 | 상태 자리가 사용자 home 아래다 | ADR-077 §9 |
| 다른 filesystem 사이의 합치기 | 사용자 마운트포인트 하나에 둔다 | ADR-077 §9 |
| upper를 층으로 쌓아 두기 | ADR-070 §5.6 갈래 가의 기각 근거 | 롤백이 요구되고 디스크가 두 세대를 받을 때 |
| filesystem project quota로 크기 세기 | root와 관리자 준비가 필요하고 순회가 싸다 | 사내 트리에서 순회가 너무 느릴 때 |
| 노드 로컬 spool의 보존 중 암호화 | 같은 내용이 lower와 scratch에 평문으로 있다 | 기계 전체의 디스크 암호화는 이 팩 밖이다 |
| 저장소 소유의 구성 정의 파일 | 이름과 명령은 굽기 계약이 싣는다 | 이름이 갈라지는 사례 |
| 합치기 대기 중 Run을 경계에서 멈췄다 잇기 | graceful로 기다린다 | ADR-064 §3.4 |

---

# 2. 구조 불변식

| 불변식 | 지키는 법 |
|---|---|
| `I1`을 고치지 않는다. 새 점유 단위를 Mediator에 만들지 않는다 | lower 조율은 기계 안의 `flock`과 광고 drain으로 한다 |
| 단계 상태 어휘와 Run 상태기계를 바꾸지 않는다 | 새 구간은 phase 칸에 적는다. 진행 중 판정(`PENDING`, `CLAIMED`, `ASKED`)은 그대로다 |
| 매칭 규칙을 바꾸지 않는다 | 새 광고 키는 평평한 문자열 키다. 부정 조건이나 술어를 열지 않는다 |
| Record는 append-only이고 봉인 뒤 고치지 않는다 | 합치기는 굽기 Run의 단계로 Record에 남는다. 재개의 흔적은 metadata가 진다 |
| 마운트된 lower를 고치지 않는다 | 전이 상태는 lower 밖에 둔다. `.enode-metadata.json`은 마운트가 0인 합치기 창에서만 쓴다 |
| 임대 창에서 크기에 비례하는 일을 하지 않는다 | trash와 spool은 `rename` 한 번. 크기 측정과 삭제는 보고 뒤다 |
| 합치기는 같은 filesystem 안에서만 한다 | 시작 전에 `st_dev`를 확인하고 어긋나면 시작하지 않는다 |
| 합치기는 재개할 수 있어야 한다 | 표시는 lower에 적용한 뒤에만 치운다 |
| 시험이 실제 운영 lower를 바꾸지 않는다 | 합치기 시험은 버려도 되는 lower에서 한다. SunnyVM의 `/srv/yocto`는 읽기만 한다 |
| 결과 의미는 runtime과 무관하다 | native, runc-overlay에서 같은 actor와 effect는 같은 commit set을 낸다 |

---

# 3. 접점 — 두 기능 이상이 만지는 파일

진행자가 직렬로 병합한다(`CONVENTIONS.md` 3.1). 파일 경로는 `unit/runtime-environment-profile`
기준이다.

| 파일 | 만지는 기능 | 무엇을 |
|---|---|---|
| `internal/enode/claim.go` | 1, 2, 3, 6, 10, 11 | 수확 제거, 종료 보고, 예산, build 단계 기록, capture 봉투, adapter 호출 |
| `internal/enode/runc_overlay_linux.go` | 1, 4, 6, 7, 10 | Finalize, trash `rename`, 대기 upper, 합치기 준비, spool |
| `internal/contract/contract.go` | 1, 3, 5 | `effect`, 예산, `sync`, `builds[]`, merge kind, 검증 |
| `internal/api/api.go` (라우트 등록) | 2 | `POST exited`, 진행 조회의 phase |
| `internal/store/schema.sql`, `internal/store/claim.go` | 2, 3 | phase, `phase_since`, 종료 시각, 원인 |
| `internal/enode/detect.go`, `internal/enode/advertise.go` | 8, 9 | `workspace.writes`, `ir`, `repo.built`, drain |
| `internal/enode/leases.go` | 8 | Run 단위 공유 잠금 |
| `internal/environment/check.go` | 8 | `st_dev`, 소유 uid, `lower.json` 검사 |
| `enode-design/protocol/mediator-api.md`, `run-contract.md` | 2, 3, 5 | 정본 되돌림(`decisions.md` 7절) |

---

# 4. 코드 기준선과 회차 브랜치

- 이 팩이 넓히는 ADR-073 구현은 `unit/runtime-environment-profile`에만 있다. 상위 저장소에서
  `origin/main`보다 18 커밋 앞서 있고 충돌 없이 합쳐진다(2026-09-23 확인).
- 회차 브랜치 `v4-run-finalize-bake`는 `main`에서 땄다(`CONVENTIONS.md` 3.1). **Workspace
  Detection 전에 그 코드가 회차 브랜치에 있어야 한다.** 길은 둘이다.
  - `unit/runtime-environment-profile`을 `main`에 PR로 먼저 병합하고 회차 브랜치가 `main`을 합친다.
  - 회차 브랜치에 그 브랜치를 직접 합친다.
- enode-design에서도 ADR-075~077은 `unit/runtime-environment-profile`에만 있다. 회차 브랜치의
  서브모듈은 그 머리 `369270a`를 가리킨다.
