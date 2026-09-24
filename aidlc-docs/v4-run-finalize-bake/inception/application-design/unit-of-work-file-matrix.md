# 파일 행렬 — 어느 유닛이 어느 파일을 고치나

실행 계획이 이 단계에 필수로 건 표다 (`execution-plan.md` 3절). **설계 문서를 기준으로 한 예상**이다.
각 유닛의 Code Generation 계획이 실제 파일로 고치고, 이 표에 없던 파일을 만지면 그 계획에 적는다
(`CONVENTIONS.md` 3.5 의 「행렬 밖 파일을 만진 diff」).

- **작성 시각**: 2026-09-24T12:14:10Z
- **열 이름**: 순서 번호와 유닛 — ① contract-grammar 계약 문법 · ② step-phase 진행 구간 ·
  ③ finalize 결과 확정 · ④ trash · ⑤ merge-rules 합치기 규칙 · ⑥ lower-state 아래층 상태 ·
  ⑦ bake 굽기 · ⑧ checkpoint 보존

---

## 1. 행렬

### 1.1 계약과 Mediator

| 파일 | ① | ② | ③ | ④ | ⑤ | ⑥ | ⑦ | ⑧ |
|---|---|---|---|---|---|---|---|---|
| `internal/contract/contract.go` | ● | | | | | | | |
| `internal/contract/grammar.go` | ● | | | | | | | |
| `internal/contract/examples/bake.json` (새 · 공개 도구로 쓴 예시) | ● | | | | | | | |
| `cmd/runctl/shape.go` (lint 의 판정 조건 제안) | ● | | | | | | | |
| `internal/store/schema.sql` | | ● | | | | | | |
| `internal/store/claim.go` (claim · 종료 수락 · 결과 봉인) | | ● | | | | | | |
| `internal/store/observe.go` (진행 조회) | | ● | | | | | | |
| `internal/store/queue.go` (대기 사유의 후보 수) | | ● | | | | | | |
| `internal/api/api.go` (라우트 등록 · 결과 · 진행 조회 핸들러) | | ● | | | | | | |
| `internal/record/record.go` (단계 기록의 시각 둘) | | ● | | | | | | |

### 1.2 노드 — 새 패키지

| 파일 | ① | ② | ③ | ④ | ⑤ | ⑥ | ⑦ | ⑧ |
|---|---|---|---|---|---|---|---|---|
| `internal/scratch/` trash · 지우기 · 삭제자 · 양 (새) | | | | ● | | | | |
| `internal/scratch/` Checkpoint Store (새 파일 · 같은 패키지) | | | | | | | | ● |
| `internal/merge/` (새) | | | | | ● | | | |
| `internal/lower/` (새) | | | | | | ● | | |

### 1.3 노드 — `internal/enode` 와 `cmd/enode`

| 파일 | ① | ② | ③ | ④ | ⑤ | ⑥ | ⑦ | ⑧ |
|---|---|---|---|---|---|---|---|---|
| `internal/enode/runtime.go` (세션 수명 · native · 런타임 능력) | | | ● | | | ● | | ● |
| `internal/enode/runc_overlay_linux.go` | | | ● | ● | | ● | ● | ● |
| `internal/enode/runc_overlay_other.go` | | | ● | ● | | | | ● |
| `internal/enode/claim.go` | | | ● | | | | ● | ● |
| `internal/enode/changed.go` (명시 훑기 · 「바뀐 파일이 없다」) | | | ● | | | | | |
| `internal/enode/upload.go` (업로드 client · 흘려 보내기) | | | ● | | | | | |
| `internal/enode/advertise.go` (drain 합치기 · 후보 잠금 · 광고 키) | | | | ● | | ● | | |
| `internal/enode/detect.go` (arch 조건 제거 · 광고 능력) | | | | ● | | ● | | |
| `internal/enode/policy.go` (소유자 정책과 스스로의 drain) | | | | ● | | | | |
| `internal/enode/leases.go` (후보 잠금과 임대) | | | | | | ● | | |
| `internal/enode/status.go` (상태 파일의 칸) | | | | ● | | ● | | ● |
| `internal/enode/config.go` (trash 와 checkpoint 설정) | | | | ● | | | | ● |
| `internal/enode/` 후보 잠금 · 준비도 점검 (새 파일) | | | | | | ● | | |
| `internal/enode/` 굽기 build · merge 단계 · merge-helper (새 파일) | | | | | | | ● | |
| `internal/enode/` trash-helper (새 파일) | | | | ● | | | | |
| `cmd/enode/main.go` | | | ● | ● | | ● | ● | ● |
| `cmd/enode/environment.go` (준비도 점검의 이음매 연결) | | | | | | ● | | |

### 1.4 그 밖

| 파일 | ① | ② | ③ | ④ | ⑤ | ⑥ | ⑦ | ⑧ |
|---|---|---|---|---|---|---|---|---|
| `internal/environment/check.go` (Fact 를 더 내는 이음매) | | | | | | ● | | |
| `internal/panel/view.go` (drain 출처 · 양 · 보존 수) | | | | ● | | ● | | ● |
| `internal/panel/boundary_test.go` (금지 넷 · 봉인 셋) | | | | ● | ● | ● | | |
| `packaging/macos/examples/*.yaml` 넷 (`min_free_gb` 주석) | | | | ● | | | | |
| 조각 스크립트 (새 · 자리는 Code Generation 이 정한다) | | | ● | ● | | | ● | ● |

조각 스크립트는 유닛마다 자기가 맡은 조각의 파일만 만든다 — ③ 조각 1 · 2 · 3, ④ 조각 4,
⑦ 조각 5 ~ 8 (운영 아래층이면 멈추는 가드 포함), ⑧ 조각 9. 같은 파일을 두 유닛이 고치지 않는다.

---

## 2. 여러 유닛이 고치는 파일과 병합 순서

**열하나다.** 혼자 한 줄 순서로 하므로 동시에 부딪치지 않는다. 먼저 병합된 유닛 위에서 다음
유닛을 이어 고친다.

| 파일 | 고치는 유닛 (병합 순서) | 겹치는 곳 |
|---|---|---|
| `runc_overlay_linux.go` | ③ → ④ → ⑥ → ⑦ → ⑧ | 세션의 결과 확정과 닫기 · 지우는 다섯 자리 · 준비도 점검 · helper 여는 코드 · 보존 능력 |
| `cmd/enode/main.go` | ③ → ④ → ⑥ → ⑦ → ⑧ | 업로드 client · 삭제자와 trash-helper · 후보 잠금 연결 · 시작 순서와 merge-helper · 조회 명령과 재시작 조정 |
| `claim.go` | ③ → ⑦ → ⑧ | 명령과 agent 단계의 결과 확정 · 굽기 단계 분기 · 보존과 영수증 |
| `runtime.go` | ③ → ⑥ → ⑧ | 세션 인터페이스 · 광고할 쓰기 방식 · 보존 지원 여부 |
| `status.go` | ④ → ⑥ → ⑧ | drain 출처와 trash 양 · 굽기 출처 · spool 양과 보존 수 |
| `panel/view.go` | ④ → ⑥ → ⑧ | 같은 셋을 그린다 |
| `boundary_test.go` | ④ → ⑤ → ⑥ | 새 패키지마다 금지 한 줄과 봉인 한 줄 |
| `runc_overlay_other.go` | ③ → ④ → ⑧ | Linux 밖에서 돌려줄 「지원 안 함」 |
| `advertise.go` | ④ → ⑥ | drain 을 합치는 틀 · 후보 잠금과 광고 키 |
| `detect.go` | ④ → ⑥ | arch 조건 제거 · 광고할 능력 |
| `config.go` | ④ → ⑧ | trash 설정 · checkpoint 블록 |

**실행 계획이 짚은 다섯과 비교하면**

```text
   contract.go        ① 하나만 고친다.  계약 문법을 한 유닛에 모았다
   api.go             ② 하나만 고친다.  Mediator 를 한 유닛에 모았다
   schema.sql         ② 하나만 고친다
   claim.go           남는다 (③ ⑦ ⑧)
   runc_overlay_linux.go   남는다 (③ ④ ⑥ ⑦ ⑧) — 가장 많이 겹친다
```

Application Design 이 더한 셋(`advertise.go` · `detect.go` · `status.go`)도 남는다. 새로 찾은 것은
여섯이다 — `runtime.go` · `runc_overlay_other.go` · `config.go` · `panel/view.go` ·
`boundary_test.go` · `cmd/enode/main.go`.

---

## 3. 만지지 않는 것

품질 게이트 3 · 4 (`execution-plan.md` 6절)를 여기서 센다.

```text
   internal/match       0.  매칭 규칙을 안 바꾼다.  대기 사유는 Match 를 두 번 부를 뿐이다
   cmd/mediator         0.  그리고 새 패키지 셋과 internal/enode 를 링크하지 않는다 —
                        boundary_test.go 의 금지 넷이 막는다
   internal/schema · internal/proc · internal/transcript · internal/transcriptui ·
   internal/runctl · internal/build · cmd/enodectl · cmd/iapadapter      0
   cmd/runctl           shape.go 한 파일만 (①).  Application Design 은 「안 만진다」에 뒀고
                        Units Generation 이 고쳤다 (계획 2.2)
```

---

## 4. 커버리지 여유

`internal/enode` 는 81.5% 이고 하한이 80% 다. 이 패키지를 고치는 유닛이 다섯(③ ④ ⑥ ⑦ ⑧)이다.
규칙은 새 패키지 셋에 두고 `internal/enode` 에는 잇는 코드만 둔다 (Application Design 질문 2).
유닛마다 Code Generation 계획에 이 패키지의 커버리지를 병합 전과 뒤로 적는다.

새 패키지 셋(`internal/scratch` · `internal/merge` · `internal/lower`)도 각자 80% 이상이어야
한다. namespace 가 필요한 경로는 helper 입구만 남기고, 규칙은 기본 `go test` 에서 시험한다
(`requirements.md` 5.1 · 5.6).
