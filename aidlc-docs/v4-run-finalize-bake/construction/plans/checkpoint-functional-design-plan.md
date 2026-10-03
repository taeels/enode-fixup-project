# `checkpoint` — Functional Design 계획

**유닛** `checkpoint` (실패한 단계 보존) · **브랜치** `unit/checkpoint` (`main` `3c33730` 에서 땄다) · **회차**
`v4-run-finalize-bake` (굽기) · **순서** 여덟 중 여덟째 · **앞 유닛** bake (PR #68 로 병합) · **맡는 조각** 9 (보존 — 사람 ·
SunnyVM) · **기대는 조각** 4 (trash — 2026-09-26 초록) · **스토리** US-2 (trash 와 보존본이 차지한 양) · US-5 (실패한 단계의 upper 가
48시간 남는다) · US-11 (captured 의 뜻) · **완료 조건** 2 (spool 양) · 3 ③ (기본값을 켜는 자리에 적는다) · 7 (captured 의 뜻) ·
**요구** FR-10 (기능 10 · checkpoint capture 와 Checkpoint Store)

- 작성 시각 2026-09-30T13:11:54Z · 규칙 `.aidlc/aidlc-rules/aws-aidlc-rule-details/construction/functional-design.md` Step 1 ~ 4
- 짓는 것 — 상태 다섯과 원인 여섯의 전이 · 받아들임 · 보고 뒤 판정과 퇴출 · TTL · 재시작 조정 · 조회 (유닛 정의 8절).
  안 짓는 것 — N2 (NFR 값 둘째 · checkpoint 하나의 상한과 inode 한도의 값 · NFR Requirements 단계) · 코드 (Code Generation)

---

## 1. 받는 일

| # | 일 | 출처 |
|---|---|---|
| 1 | Checkpoint Store (`internal/scratch` 의 새 파일) — 받아들임 · upper 를 spool 로 rename 한 번 · 보고 뒤 크기 판정과 퇴출 · TTL · 재시작 조정 · 목록 | `unit-of-work.md:314` · FR-10 · `components.md:101` |
| 2 | 런타임이 보존 지원을 낸다 — 지원 여부 · 범위 `workspace-upper` · 보장 `inspect-only`. native 는 `unsupported`(`runtime`) | FR-10 첫 줄 · lower-state `code-summary.md:273` (`RuntimeCapability` 에 Capture 칸) |
| 3 | result 에 `checkpoint_capture` 를 채운다 — 상태 · 원인 코드 · 불투명 ID · 범위 · 보장 · 노드 · 만료 시각. host 경로 없음 | `unit-of-work.md:317` · step-phase `code-summary.md:148` |
| 4 | 실패 상세는 diagnostics 에 둔다 | FR-10 · ADR-076 §2 (상태는 receipt, 상세는 diagnostics) |
| 5 | 노드 설정의 checkpoint 블록 — 기본 on-failure · 48시간 · 보존 용량 20%. 원격 계약과 agent 출력은 못 바꾼다 | `unit-of-work.md:319` · 결정 2-10 (첫 기본값) · 2-11 (정책은 노드 소유자의 것) |
| 6 | `enode checkpoint list \| show` — host 경로는 소유자 화면에만 | `unit-of-work.md:320` · `component-methods.md:646` |
| 7 | 상태 파일과 제어판에 spool 양과 보존 수 | 완료 조건 2 · `component-methods.md:278` |
| 8 | 바뀐 기본값을 겪는 사람이 읽는 자리에 적는다 — 켜는 자리가 곧 끄는 자리. 도구의 자격증명 캐시도 남는다는 것까지 | 완료 조건 3 ③ (`stories.md:125`) · US-5 |
| 9 | captured 의 뜻 — 그 노드에만 있다 · inspect-only · 누가 여나 · TTL 에 없앤다 | 완료 조건 7 · US-11 |
| 10 | 단계 outcome 과 commit set 은 capture 성패와 무관하다 | FR-10 끝 줄 · ADR-076 §4 |
| 11 | 받아들임과 spool 은 닫기 안이고 Finalize 예산의 남은 몫을 쓴다. `Keep.Upper` 에 spool 자리 | finalize `code-summary.md:208` · trash `code-summary.md:313` |
| 12 | 퇴출은 `Trash.Move` · `Usage` 에 `spool_bytes` · `checkpoints` | trash `code-summary.md:313` |
| 13 | `Close(Keep{Upper})` 는 RENAME_NOREPLACE — 자리가 이미 있으면 실패하고 upper 는 trash 로. keep 은 첫 Close 에서만 본다 | bake `code-summary.md:855` (10절) |
| 14 | 실패한 굽기 build 의 upper 를 spool 로 받을지. 받으면 bake 규칙 3 (첫 실패에서 멈춤) 의 근거를 다시 본다 | bake `code-summary.md:855` · bake 흐름 12절 · bake `business-rules.md:118` |
| 15 | 이어 고칠 파일 — `claim.go` (closeOut) · `runc_overlay_linux.go` · `runc_overlay_other.go` · `runtime.go` · `config.go` · `status.go` · `panel/view.go` · `cmd/enode/main.go` | bake 10절 · 파일 행렬 2절 |
| 16 | 기동 차례 — 삭제자 첫 회 뒤 checkpoint 조정, 그 뒤 광고 | `services.md` 4절 · lower-state `business-logic-model.md:144` |
| 17 | 크로스 빌드 — statfs 여유와 보존 지원을 `_linux.go` · `_other.go` 짝으로 | `unit-of-work.md` 10절 |
| 18 | 보안 — spool 은 소유자만 읽는다 · TTL 에 지운다 · 바깥 descriptor 에 host 경로와 비밀 없음 · Mediator 가 링크하는 패키지에 spool 코드 없음 | `requirements.md` 5.3 · 5.2 |
| 19 | 조각 9 (보존) — 장면 3 의 1 ~ 4 · native `unsupported`(`runtime`) · 여유 하한 미달 `rejected`(`free_space`) · outcome 불변 · 재시작 조정 | `scene-gates.md` 2절 · `unit-of-work.md:328` |
| 20 | NFR 은 다음 단계 — N2 · 보안 · 성능 (보고 전에는 rename 한 번) | `unit-of-work.md:338` |

---

## 2. 오늘의 코드 (`main` `3c33730`) 와 측정

### 2.1 코드

| 자리 | 오늘 | 이 유닛에 뜻하는 것 |
|---|---|---|
| `internal/contract/result.go:147-166` | `CheckpointCapture` 칸 일곱 · 상태 상수 다섯. 원인 코드 상수 0 | 모양이 있다 |
| `internal/store/claim.go:933` · `:965-968` | Mediator 가 칸을 받아 봉인한다. 어휘는 state 만 보고 로그만 남긴다 | Mediator 코드를 안 고친다 |
| `runtime.go:101-106` · `runc_overlay_linux.go:546-601` | helper 가 끝난 뒤 `Renameat2(RENAME_NOREPLACE)`. 실패는 Close 오류에 섞여 돌아온다. abort 된 세션은 keep 을 못 한다 (`:553`) | 옮기는 기제는 있다. 실패가 Close 오류와 섞인다 |
| `finalize.go:230-235` (settle) | Close 오류가 있으면 finalize 칸 error · error 문장 — 단계가 완주가 아니다 | 보존 실패가 그대로 들어오면 outcome 이 바뀐다 (3절 2번) |
| `claim.go:867-909` (closeOut) | Finalize → Close(keep) → after → diagnostics. `diagnosticsFor` 는 Close 뒤 (`:886`) | 보존 판정은 Close 앞이다 |
| `claim.go:777-780` · `:1082` | 임대 만료 · 실행 실패 끝은 Finalize 없이 닫는다. 「칸이 없는 것이 곧 그 구간에 닿지 않았다」 | 칸을 안 싣는 끝의 선례 |
| `runtime.go:115-119` · `:159` · `runc_overlay_linux.go:130` | `RuntimeCapability{Writes}` 뿐 · Capture 자리 주석 | Capture 칸을 더한다 |
| `runtime.go:253-260` (onceSession) | 둘째 Close 는 첫 결과를 돌려준다 | 이미 spool 로 옮긴 upper 를 뒤 Close 가 안 건드린다 (ADR-076 §5 끝) |
| `bake_build.go:386-399` (failEnd) · `:338-341` (stopEarly) | 굽기 build 의 실패 끝은 `Keep{}` | 물음 3 |
| `scratch/trash.go:43` · `remove_unix.go:17-26` · `deleter.go:20-26` | `Trash.Move` · `Measure` 는 trash 안만 걷는다 · `Usage` 에 spool 칸 없음 | 보고 뒤 걷기는 helper 안이다 (`components.md` 6절). spool 을 걷는 입구가 없다 |
| `internal/panel/boundary_test.go:102` | `internal/scratch` 는 표준 라이브러리와 x/sys 만 | Store 는 `contract` · `lower` 를 모른다 |
| `status.go:36` · `panel/view.go:28` · `panel/page.go:273-277` | 상태 파일과 제어판의 scratch 줄 — trash 양 · 측정 전 수 · 지우는 중 | spool 줄을 같은 자리에 |
| `config.go:97` · `:141` · `:161` | `min_free_gb` 기본 10 · checkpoint 블록 없음 · 틀린 mcp 설정은 노드를 안 띄운다 | 설정의 선례 |
| `cmd/enode/main.go:48-80` · `:317-324` · `environment.go:21` | 하위 명령에 checkpoint 없음 · 기동 청소 뒤 삭제자 · `env check\|apply --config PATH [--json]` | 조정 자리와 CLI 의 선례 |

없는 것 — Checkpoint Store · spool · 원인 코드 상수 · 설정 블록 · 조회 명령 · 재시작 조정 · diagnostics 의 보존 칸.

### 2.2 측정

| 무엇 | 값 | 출처 |
|---|---|---|
| 노드 uid 소유 부모 사이의 upper rename | SunnyVM 에서 돈다 — 같은 기제로 대기 자리에 옮긴다 | bake `code-summary.md` 8.6 (조각 6 · 7 · 8) |
| 보고 뒤 upper 걷기 | 증분 157,982 파일 2.6초 · 처음부터 681,670 파일 3.95초 | ADR-076 §5 · ADR-077 §12 |
| trash-helper 의 걷기 | 초당 약 811,700 항목 | `aidlc-state.md:257` (조각 4) |
| 커버리지 | `internal/enode` 86.7% · `cmd/enode` 80.8% (하한 80%) | bake `code-summary.md` 6절 |

**못 한 것.** namespace 안에서 subordinate uid 소유 0600 파일을 읽는 것 — 이 기계는 `unshare --map-auto` 에서 `newuidmap` 이
`Operation not permitted` 로 실패했다 (임시 자리에서 시도하고 지웠다). 물음 6 의 A 가 기대는 사실이다. 찬 캐시와 사내 규모의 걷기
시간은 정본도 확인하지 않았다 (ADR-076 §4.1). SunnyVM 에는 접속하지 않았다.

---

## 3. 묻지 않고 정한 것

| # | 정한 것 | 근거 |
|---|---|---|
| 1 | **판정 차례** — ① 정책이 이 단계를 요구하나 (물음 1 · 2 · 3) · 아니면 `not_requested` ② 런타임 지원 — native 면 `unsupported`(`runtime`) · spool 과 upper 의 filesystem 번호가 다르면 `unsupported`(`cross_filesystem`) ③ Finalize 가 마감으로 끝났거나 마감이 지났으면 `rejected`(`lease_budget`) ④ 여유 < `min_free_gb` 면 `rejected`(`free_space`) · 보존 총량 ≥ 몫이면 `rejected`(`quota`) ⑤ 자리를 예약하고 `Close(Keep{spool})` · keep 을 못 하면 `failed`(`io`) ⑥ rename 뒤 확정 전에 마감이 지나면 `failed`(`lease_budget`) ⑦ 확정하면 `captured` | ADR-076 §2 표 · §4 의 3 ~ 6 · 결정 2-2 (상태 다섯) · 2-3 (원인 여섯). ④ 는 여유를 먼저 본다 — 노드 전체의 여유를 보존 몫보다 먼저 지킨다 |
| 2 | keep 실패를 Close 오류에서 뗀다 — `failed`(`io`) 로만 남고 finalize 칸 · error · exit 은 그대로. 굽기 대기 자리의 keep 실패는 오늘처럼 오류다 | ADR-076 §4 (capture 실패가 exit status 를 안 바꾼다) · FR-10 끝 줄 · 2.1 의 settle |
| 3 | 칸이 있는 때 — 세션을 열고 closeOut 에 닿은 단계만. 닿지 않은 끝 (임대 만료 · 실행 실패 · 하네스 앞 거절 · runtime open 실패) 은 칸이 없다. merge 단계는 물음 3 | `claim.go:777-780` 의 선례 |
| 4 | 받아들임은 상수 시간 둘 — statfs 의 여유와 store 가 들고 있는 측정한 크기의 합. 측정 전 항목은 0 으로 센다. 몫 = 보존 용량 비율 × (보존 총량 + 여유) | 결정 2-7 (보존 판정을 보고 전과 뒤로 나눈다) · 2-10 · ADR-076 §5 |
| 5 | 보고 뒤 판정 — 새 항목을 helper 안에서 걷는다 (크기와 inode 수). 하나의 상한을 넘으면 그것을 퇴출. 그 뒤 몫이나 inode 한도를 넘으면 오래된 것부터 퇴출. inode 한도는 **보존본 전체**에 건다. 퇴출은 `Trash.Move` 이고 receipt 는 안 바뀐다 | 결정 2-7 · 2-8 (upper 순회 · quota 안 씀) · 2-10 (오래된 것부터) · ADR-076 §5 (「node 전체의 보존 용량 · inode 한도」) · §6. `component-methods.md:295` 의 `MaxInodes` 는 범위를 안 적었다 |
| 6 | TTL — `expires_at` = 잡은 시각 + 그때의 TTL. 설정을 뒤에 바꿔도 이미 잡은 것의 만료는 그대로. 만료도 `Trash.Move` | ADR-076 §6 (captured 는 당시의 사실) |
| 7 | 재시작 조정 — 미완료 (예약만 · rename 뒤 확정 전) 는 trash 로. 완료된 것은 보고가 닿았는지와 무관하게 TTL 까지 (물음 10). 만료된 것은 trash. 측정 전인 완료 항목은 다시 판정에 넣는다. scratch 를 나눠 쓰는 형제의 미완료 항목은 잠금으로 지킨다 | ADR-076 §2 끝 · §4 (보고가 거절되거나 임대가 만료되면 정책에 따라 만료) · §5 끝 · trash `business-rules.md` 3절 (세션 잠금) |
| 8 | 정책은 노드 설정에만 — 계약에 칸이 없다. 값은 `off` · `on-failure` · `always`. 블록이 없거나 값이 0 이면 기본값. 틀린 값이면 노드가 안 뜬다 | 결정 2-11 · `application-design.md:129` · `component-methods.md:288` · US-5 (켠 적 없이도) · `config.go:141` · `:161` 선례 |
| 9 | 「세션이 끝나기 전 운영자의 명시 요청」 정책은 짓지 않는다 | FR-10 과 유닛 정의에 없다. ADR-076 §4 는 정책이 그것을 표현할 수 있다고만 적었다 |
| 10 | spool 은 `<scratch>/spool/` 고정 · 0700. host 경로는 receipt · diagnostics · 광고 · Mediator 에 없고 `show` 에만. diagnostics 문장은 파일 경로를 빼고 오류의 뜻만 싣는다 | `component-dependency.md:124` · `requirements.md` 5.3 · `component-methods.md:646` |
| 11 | store 가 적는 것 — ID · 노드 · Run · 단계 · attempt · runtime · 범위 · 보장 · lower 와 environment 신원 · 잡은 시각과 만료 시각 · 상태 · lower metadata 가 있으면 그 head 와 ir. 신원은 rename 앞에 정한다 | ADR-076 §5 · §4 의 3 |
| 12 | 코드 자리 — Store 와 원인 상수는 `internal/scratch`. enode 가 `scratch.Capture` 를 `contract.CheckpointCapture` 로 옮긴다. Mediator 는 원인을 안 본다 | `boundary_test.go:102` · step-phase `business-logic-model.md:214` · `store/claim.go:965` |
| 13 | 조회 — `enode checkpoint list \| show <ID> --config PATH [--json]`. 출력은 영어. 칸은 11번의 기록 | `environment.go:21` 선례 · CONVENTIONS 2.1 |
| 14 | 보존하지 않는 것 — `enode env check` 의 smoke 는 단계가 아니다. 매칭 · 광고 어휘를 안 고친다 | FR-4 (smoke 는 trash) · ADR-076 §5 (capability 는 새 매칭 속성이 아니다) |
| 15 | 조각 9 스크립트는 `scripts/finalize-bake/slice-9.sh`. 명령은 Code Generation 이 굳힌다 | finalize `code-generation-plan.md:221` · `scene-gates.md` 3절 |

---

## 4. 물음 열

답을 `[Answer]:` 뒤에 적는다. 권장을 **A** 에 둔다.

**내기 전에 대 본 것** — 네 흠에 물음마다 대 봤다.

| 흠 | 대 본 결과 |
|---|---|
| 이미 정한 것을 다시 묻기 | 상태와 원인 어휘 (결정 2-2 · 2-3) · 기본값 셋 (결정 2-10) · spool 자리 · 정책 값 셋 · 설정 블록이라는 자리 · 상태 파일의 칸 이름 (`component-methods.md:278`) 은 묻지 않는다. 물음은 정본과 요구가 값을 안 준 것만이다 |
| 측정 없는 근거 | 물음마다 코드 줄이나 정본 절을 단다. 물음 6 의 A 가 기대는 namespace 안 읽기는 이 기계에서 확인하지 못했다 (2.2) |
| 앞 유닛 넘김과 어긋남 | 물음 3 은 bake 넘김이 그린 두 갈래 (build 끝 · 대기 자리 뒤) 를 선택지로 두고, 넘김이 뺀 merging 뒤는 어느 선택지에도 없다. 모든 선택지가 keep 의 RENAME_NOREPLACE 와 첫 Close 규칙을 지킨다 |
| 요구를 빼는 선택지 | 모든 선택지가 FR-10 (outcome 불변 · diagnostics 상세 · host 경로 없음) · 완료 조건 2 · 3 ③ · 7 을 지킨다. 「퇴출을 곧바로 잊는다」(ADR-076 §9 의 「조회가 퇴출을 보인다」를 뺀다) 와 「diagnostics 에 안 싣는다」(FR-10 을 뺀다) 는 넣지 않았다 |

### Question 1 — on-failure 가 보존하는 「실패」

노드는 Run 의 성공을 모른다 — 성공은 계약의 조건이 정한다 (불변식 I3 · `claim.go:820`). 그래서 on-failure 는 노드가 닫기 전에 아는
사실로 정한다. 보존 여부만 정하고 결과에는 안 들어간다. 오늘 노드가 내는 끝은 exit 와 signal 둘이다 (`finalize.go:92`).

A) 노드가 아는 실패 전부 — 명령 단계는 exit ≠ 0 · signal, agent 단계는 하네스가 완주하지 못함 (`claim.go:1077`), 둘 다 계약이 적은
산출물이 빠짐 (`diagnostics.missing` · `finalize.go:269`) 이나 Finalize 오류

B) 프로세스의 끝만 — 명령 단계는 exit ≠ 0 · signal, agent 단계는 하네스 미완주. exit 0 인데 산출물이 빠진 단계는 보존하지 않는다

C) Other (please describe after [Answer]: tag below)

**권장 A.** 발단이 실패 조사다 (ADR-076 §1 · §10 의 on-failure 근거). exit 0 인데 산출물이 빠진 단계는 흔한 실패이고 까닭이 upper 에
있다. 늘어나는 양은 20% 몫과 TTL 이 막는다. 대가 — 빠진 산출물을 Close 앞에서 본다 (`diagnosticsFor` 를 앞으로 · 2.1).

[Answer]: A

### Question 2 — 요구하지 않은 단계와 native 노드에서 receipt 의 상태

3절 1번은 정책을 먼저 본다. 반대 차례도 정본과 어긋나지 않는다 — ADR-076 §2 는 unsupported 를 「이 노드와 runtime 에서는 어느
단계든 같은 원인 · 실행 전에 capability 로 안다」로 적었다.

A) 정책이 먼저 — 요구하지 않았으면 `not_requested` (off 인 노드 · on-failure 에서 성공한 단계 · native 도). 요구했을 때만
`unsupported`(`runtime` · `cross_filesystem`)

B) 런타임이 먼저 — native 노드의 모든 단계가 `unsupported`(`runtime`). overlay 노드는 A 와 같다

C) Other (please describe after [Answer]: tag below)

**권장 A.** ADR-076 §4 의 3 이 「정책이 요구하고 runtime 이 지원하면」 차례다. B 면 보존을 끈 native 노드의 성공 단계마다
unsupported 가 실려, 계약 작성자가 무엇을 시도했다가 못 한 것으로 읽는다. 조각 9 의 「native 는 unsupported(runtime)」 는 둘 다
실패한 단계에서 성립한다.

[Answer]: A

### Question 3 — 실패한 굽기 build 의 upper (bake 유닛이 넘긴 일)

굽기 upper 는 9 ~ 23 GB 다 (ADR-077 §12). build 의 실패 끝 넷 — 명령 실패 · IR 어긋남 (둘 다 DONE) · IR 대조를 못 함 · pinned 실패
(둘 다 FAILED) — 은 `failEnd` 가 `Keep{}` 로 upper 를 trash 에 보낸다 (`bake_build.go:386`). 대기 자리에 간 뒤의 끝 (merge 대기 상한 ·
Preflight 어긋남 · 임대가 사라짐) 은 merge 단계의 몸통이 대기 자리를 trash 로 옮긴다. merging 뒤는 넘김이 보존 대상에서 뺐다.

A) build 끝의 실패만 — failEnd 넷이 `Keep{spool}` 로 닫고 build 단계 result 에 칸. 대기 자리 뒤 끝은 보존하지 않고 merge 단계에는
칸이 없다. bake 규칙 3 (첫 실패에서 멈춤) 은 그대로 — 보존본이 첫 실패 자리의 트리를 담는다. 규칙의 근거 문장만 고친다

B) 굽기 단계는 보존하지 않는다 — build 단계에 `not_requested`. 정책은 작업 단계 (명령 · agent) 에만 건다

C) A 에 대기 자리 뒤 끝을 더한다 — merge 단계 몸통이 대기 자리를 spool 로 옮기고 merge 단계 result 에 칸

D) Other (please describe after [Answer]: tag below)

**권장 A.** 넘김의 두 갈래 가운데 조사할 것이 있는 쪽이다 — 빌드가 실패한 트리. 크기는 보고 뒤 판정이 막는다 (하나의 상한 · 20%
몫). 넘으면 걷기 몇 초 뒤 trash 로 가므로 오늘보다 조금 늦게 지워질 뿐이다. C 의 대기 자리 upper 는 빌드가 성공한 트리라 조사할 것이
적고, merge 단계에는 세션이 없어 칸의 뜻이 바뀐다. 대가 — A · C 는 `bake_build.go` (C 는 `bake_merge.go` 도) 를 고친다. 행렬에서
bake 만의 파일이다.

[Answer]: A

### Question 4 — 설정 블록의 키 이름과 단위

자리는 노드 설정의 `checkpoint:` 블록이다 (`application-design.md:61`). 하나의 상한과 inode 한도의 값은 N2 가 정한다 — 여기서는 이름만.

A) 단위를 이름에 — `min_free_gb` (`config.go:97`) 와 같은 꼴

```yaml
checkpoint:
  policy: on-failure      # off | on-failure | always
  ttl_hours: 48
  capacity_percent: 20    # (보존 총량 + 여유) 의 몫
  # max_gb:               하나의 상한.  값은 N2
  # max_total_inodes:     보존본 전체의 inode 한도.  값은 N2
```

B) 값에 단위 — `ttl: 48h` · `capacity: 20%` · `max_size` 는 크기 문자열 (profile 의 `tmp.size` 와 같은 꼴 · `execenv.ParseSize`) ·
`max_total_inodes`

C) Other (please describe after [Answer]: tag below)

**권장 A.** 같은 파일의 선례를 따른다 — 소유자가 한 파일에서 두 표기를 배우지 않는다. B 는 파서 셋 (시간 · 백분율 · 크기) 과 그
오류 문장이 는다.

[Answer]: A

### Question 5 — 기본값과 양을 노드 소유자가 보는 자리 (완료 조건 3 ③ · 2 · US-5)

오늘 소유자가 노드 쪽에서 보는 자리는 상태 파일의 scratch 칸과 제어판의 trash 줄이다 (`status.go:36` · `page.go:273`). 블록을 안
적은 소유자는 설정 파일에서 기본값을 못 본다.

A) 상태 파일과 제어판 — spool 양 · 보존 수 · 측정 전 수 (trash 줄과 같은 모양) 와 실효 정책 한 줄 (`on-failure · 48시간 · 20%`) ·
「바꾸려면 설정의 checkpoint: 블록」 · 「실패한 단계의 upper 에는 도구가 쓴 자격증명 캐시가 남을 수 있다」. 데몬 기동 로그에 영어 한 줄.
native 노드는 「보존하지 않는다 (runtime)」 한 줄

B) A 에 더해 설정이 없을 때 보이는 예시 (`SampleLocal` · `config.go:242`) 에 주석 한 줄

C) A 에 더해 `packaging/macos/examples/*.yaml` 넷에 주석 블록 — 행렬 밖이고, 그 노드들은 native 라 보존을 못 한다

D) Other (please describe after [Answer]: tag below)

**권장 A.** 여유 부족 drain 의 출처를 보는 화면과 같은 자리다 (완료 조건 1). `status.go` · `panel/view.go` 는 행렬 안이다. 그리는
`panel/page.go` 는 행렬 밖이지만 trash · lower-state 도 같은 방식으로 고쳤다. B 의 예시는 설정이 아예 없는 사람에게만 보인다.

[Answer]: A

### Question 6 — 보존본을 여는 길과 일찍 버리는 길

upper 에는 컨테이너 root 가 쓴 subordinate uid 소유 항목이 있어 노드 uid 로는 못 여는 것이 있고 못 지운다
(`application-design-plan.md` 1.2 · `components.md` 6절). spool 의 항목 폴더는 노드 uid 소유라 통째 rename 은 된다.

A) 새 명령 없이 `show` 가 알린다 — host 경로 · 「노드 uid 로 못 여는 항목은 `unshare --user --map-root-user --map-auto` 아래에서
연다 (helper 와 같은 매핑)」 · 「일찍 버리려면 그 폴더를 `<scratch>/trash/` 로 옮긴다 — 삭제자가 지운다」. store 는 사라진 항목을
조정에서 지운다

B) `enode checkpoint drop <ID>` 를 더한다 — rename 한 번으로 trash 로 · store 기록을 고친다. 여는 법은 A 와 같은 안내

C) B 에 `enode checkpoint shell <ID>` 를 더한다 — 같은 매핑의 셸을 연다

D) Other (please describe after [Answer]: tag below)

**권장 A.** store 가 스스로 줄어든다 (20% 몫 · TTL) — 일찍 버릴 일이 드물다. 명령 하나는 `cmd/enode` 커버리지 여유 0.8 점 (2.2) 에
부담이다. 대가 — A 의 여는 줄은 이 기계에서 확인하지 못했다 (2.2). Code Generation 의 integration 시험과 조각 9 가 확인한다.

[Answer]: A

### Question 7 — 퇴출되거나 만료된 보존본을 조회가 얼마나 기억하나

ADR-076 §9 — 조회는 퇴출을 보인다. 만료 뒤에는 Record 의 captured 와 지금 없음이 구별된다.

A) 퇴출된 것은 원래 만료 시각까지 목록에 `evicted` 와 사유 (`larger than max_gb` · `over capacity_percent` · `over max_total_inodes`)
로 남고 만료 시각에 기록째 없앤다. 만료된 것은 곧바로 기록째. 모르는 ID 의 `show` 는
`no checkpoint <ID> on this node: it expired or was never captured here` 로 exit 1

B) 퇴출 · 만료 모두 만료 시각 뒤 TTL 한 번 더 기록을 남긴다 — `expired` 도 목록에 보인다

C) Other (please describe after [Answer]: tag below)

**권장 A.** 시계가 하나다 — receipt 의 `expires_at` 이 기록의 끝이다. Record 가 captured 를 들고 있으므로 만료는 Record 와 목록을 대
보면 안다. 사유 이름은 물음 4 의 답을 따른다.

[Answer]: A

### Question 8 — 불투명 ID 의 모양

receipt 의 `id` 이고 사람이 `show <ID>` 에 옮겨 적는 값이다. spool 의 항목 폴더 이름도 된다.

A) 무작위 12 hex 글자 (48 bit) — 예 `3f9a1c0b7d2e`. store 가 겹침을 확인한다

B) 무작위 32 hex 글자 (128 bit)

C) 시간순 — 잡은 시각과 무작위 (26 글자 · 이름순이 곧 오래된 순)

D) Other (please describe after [Answer]: tag below)

**권장 A.** 한 노드의 spool 안에서만 유일하면 된다 — 다른 노드에서 복원하지 않는다 (ADR-076 §8 순연). 짧아 옮겨 적기 쉽다. 오래된
순은 store 기록의 잡은 시각으로 정렬한다.

[Answer]: A

### Question 9 — 실패 상세의 자리 (diagnostics)

FR-10 이 「diagnostics 에 상세」를 요구하는데 `contract.Diagnostics` 에 그 칸이 없다 (`contract/result.go:109`). 어느 선택지든
`contract/result.go` 에 칸 하나를 더한다 — 행렬 밖 파일이다 (더하기만 · step-phase 규칙). Mediator 는 같은 타입으로 봉인하므로
코드는 안 고치고 다시 짓기만 한다.

A) `diagnostics.checkpoint` — 영어 문장 하나. `captured` 와 `not_requested` 면 없다. 예 `free space 8 GiB is below min_free_gb 10` ·
`moving the upper into the spool failed: file exists`. 경로는 빼고 오류의 뜻만 (3절 10번)

B) `diagnostics.checkpoint` — 구조 (`detail` 문장 · `free_bytes` · `retained_bytes` · `capacity_bytes`)

C) Other (please describe after [Answer]: tag below)

**권장 A.** diagnostics 는 사람이 읽는 칸이다 (`contract/result.go:108` 「판정 재료가 아니다 — 계약 작성자가 읽는다」). 수는 문장에
든다. 숫자 칸을 읽을 코드가 없다.

[Answer]: A

### Question 10 — 결과 보고가 거절되거나 임대가 만료된 단계의 보존본 (진행자가 더함)

3절 7번은 「완료된 것은 보고가 닿았는지와 무관하게 TTL 까지」로 정했다. ADR-076 §4 끝은 「보고가 거절되거나 임대가 만료되면
store 는 **정책에 따라** 이 보존본을 만료 · 정리」한다고 적어, 곧바로 치우는 것으로도 읽힌다. 보고가 안 닿으면 Mediator 에
receipt 가 없어 계약 작성자는 보존본이 있는 줄 모르고, 노드 소유자만 `list` 로 본다.

A) TTL 까지 남긴다 — 보고가 안 닿은 단계도 조사할 값이 있다 (멈춘 단계 · 임대를 놓친 단계). 양은 20% 몫과 오래된 것부터의 퇴출이
막는다. 목록에 「보고가 닿지 않았다」를 함께 보인다

B) 곧바로 trash 로 — 보고 거절 · 임대 만료를 store 가 알면 그 보존본을 만료로 처리한다. 재시작 뒤 보고 여부를 모르는 항목도 치운다

C) Other (please describe after [Answer]: tag below)

**권장 A.** 발단이 실패 조사다 (ADR-076 §1). 임대를 놓친 단계는 조사할 까닭이 더 많다. 「정책에 따라」의 정책을 TTL 로 읽는
것이고, 정본 되돌림 목록에 한 줄 더한다. 대가 — store 가 보고의 성패를 기록해야 목록에 보일 수 있다 (3절 11번에 칸 하나).

[Answer]: A

---

## 5. 산출물 계획 (Step 5 이후)

답이 들어오고 모호함이 풀린 뒤에 채운다. 자리는 `aidlc-docs/v4-run-finalize-bake/construction/checkpoint/functional-design/` 이다.

- [x] 답을 읽고 모호함과 서로 막히는 짝을 본다 (물음 1 과 3 · 2 와 5 의 native 줄 · 4 와 7 의 사유 이름 · 7 과 10 의 목록) — 있으면
      `checkpoint-functional-design-clarification-questions.md` (2026-09-30 · 열 모두 A · 짝 넷 모두 막히지 않아 되물음 파일 없음 —
      1 과 3 은 굽기 build 에 3 이 좁게 걸린다 · 규칙 1절)
- [x] `domain-entities.md` — 정책 · receipt 의 Capture · store 기록 (3절 11번) · 항목의 생애 (예약 · 확정 · 보관 · 퇴출 · 만료) ·
      원인 상수 · 설정 블록 (답 4) · `Usage` 의 spool 칸 · `RuntimeCapability.Capture` · ID (답 8) · diagnostics 칸 (답 9)
- [x] `business-rules.md` — 판정 차례와 전이 표 (3절 1번 · 답 1 · 2 · 3) · keep 실패의 분리 · 칸이 있는 때 · 받아들임 · 보고 뒤
      판정과 퇴출 순서 · TTL · 재시작 조정과 형제의 잠금 · 설정 검증 · 보이는 자리 (답 5) · 조회 (답 6 · 7) · 오류와 로그 문구
      (영어 · 경로 없는 diagnostics)
- [x] `business-logic-model.md` — closeOut 안의 흐름 · 굽기 build 끝 (답 3) · 보고 뒤 판정 · 기동 조정 · 조회 흐름 · 실패 갈래 표 ·
      시험 모양 (기본 `go test` 의 판정 · integration · 조각 9 스크립트) · 행렬 밖 자리 (답 9 의 `contract/result.go` · 답 3 의 `bake_build.go` ·
      답 5 의 `panel/page.go` · 답 6 이 B · C 면 `cmd/enode` 의 새 파일) · 넘기는 것 (NFR 의 N2 · 보안 · 성능 · bake 규칙 3 의 근거 문장 — 진행자) · 정본 되돌림 (ADR-076
      §10 이 미정으로 둔 descriptor · 조회 · reason 의 wire 표기 · 답 8 의 ID · 3절 5번 inode 한도의 범위)
- [x] 커버리지 대책 — 규칙은 `internal/scratch` 의 순수 함수로, 조립은 `internal/enode` 의 함수 하나, `main.go` 에는 부르는 줄만
- [x] 검사 — `enode-design/scripts/emphasis-check.py` · 말투 grep · 사내 이름 grep (새 문서 셋)

---

## 6. 확장 준수 — 이 단계

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 spool 줄 (`requirements.md` 5.3) 은 3절 10번이 닫고, 확인은 이 유닛의 NFR 이다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 — 저장소의 관례다 |
