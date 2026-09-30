# `checkpoint` — NFR Design 계획

**유닛** `checkpoint` (실패한 단계 보존) · **브랜치** `unit/checkpoint` · **앞 단계** NFR Requirements (승인 · `cd7d6a3`) ·
**다음** Code Generation

- 작성 시각 2026-09-30T14:31:43Z · 규칙 `.aidlc/aidlc-rules/aws-aidlc-rule-details/construction/nfr-design.md` Step 1 ~ 4
- 범위 — `nfr-requirements.md` 6절의 D1 ~ D4. 문서는 `construction/checkpoint/` 아래 FD 셋과 NFR 둘이고 「규칙 · 엔티티 · 흐름 ·
  NFR N절」로 가리킨다. P2 는 NFR 2절의 요구 (보존은 단계의 finalize 판정을 바꾸지 않는다) 다

---

## 1. 받는 일

| # | 일 | 출처 |
|---|---|---|
| D1 | P2 를 지키는 모양 — 후보 셋 (예약을 세션을 열 때로 · 남은 예산에 여유 · finalize 판정에서 보존에 든 시간 빼기) | NFR 6절 · 2절 P2 |
| D2 | spool 잠금의 범위 — 측정을 잠금 밖에서 할지 | NFR 6절 · P5 · 5절 (찬 캐시) |
| D3 | 받아들임의 보존 총량이 형제 (scratch 를 나눠 쓰는 노드) 의 보존본을 따라잡는 법 | NFR 6절 · P7 |
| D4 | 판정 깸의 합치기 · 측정 helper 를 여는 모양 | NFR 6절 · 흐름 5절 |
| 범주 | 회복 · 규모 · 성능 · 보안 · 논리 구성요소 — D1 ~ D4 와 NFR 에 대 본다 | 규칙 Step 3 |

---

## 2. 코드와 측정

### 2.1 코드 자리 (`main` `3c33730` 위 · 이 브랜치는 문서만 바뀌었다)

| 자리 | 오늘 | D1 에 뜻하는 것 |
|---|---|---|
| 세션 열기 — `claim.go:712` (명령 · agent) · `bake_build.go:196` (굽기 build) | 명령보다 앞이다. Finalize 예산 밖 | 예약을 여기로 옮기면 창 밖이다 |
| Finalize 예산의 시작 — `claim.go:868-871` | `spec.Deadline = from.Add(finalizeBudget)` · `from` 은 exited_at | 예산은 명령이 끝난 때부터다 |
| 마감 판정 — `claim.go:875-889` · `finalize.go:223` (settle) | Close 가 끝난 시각 (`closedAt`) 이 마감 뒤면 finalize_timeout | 예약이 Finalize 와 Close 사이에 있으면 이 판정에 든다 (흐름 1절) |
| 굽기 성공 끝의 마감 — `bake_build.go:491` | pending 을 쓰기 전에 마감을 다시 본다 | 성공 끝은 보존하지 않으므로 (규칙 1절) 영향 없음 |
| 예산이 닫기까지 덮는다 — `services.md:38` | 「capture 의 남은 예산은 Finalize 예산의 남은 몫」 | 후보 C 는 이 문장의 예외가 된다 |

### 2.2 측정 (2026-09-30 · 이 기계 · ext4 · 임시 자리에서 하고 지웠다)

| 무엇 | p50 | p90 | p99 | 최대 | n |
|---|---|---|---|---|---|
| 세션을 열 때 예약 (mkdir · 잠금 · 기록) | 67 µs | 81 µs | 168 µs | 2.5 ms | 2000 |
| 성공한 단계의 예약 버리기 (기록 · 잠금 파일 · 폴더 지움) | 36 µs | 43 µs | 91 µs | 2.4 ms | 2000 |
| 작은 요약 파일 읽기 (JSON 약 80 바이트) | 15 µs | 20 µs | 39 µs | 0.27 ms | 2000 |

NFR 계획 2절의 값 — 창 안의 보존 일 전부 p99 0.46 ms (한가함) · 쓰기 부하에서 예약이 한 번 4.2 초 · rename 은 부하에서 최대 60 ms ·
Close 안의 작업 폴더 rename (오늘 있는 것) 과 같은 종류다.

### 2.3 못 한 것

| 무엇 | 까닭 |
|---|---|
| 찬 캐시 · 사내 규모의 측정 시간 (D2 의 잠금 길이) | 정본도 측정하지 않았다 (ADR-076 §4.1 · NFR 5절) |
| 쓰기 부하에서 세션을 열 때 예약의 시간 | 이 기계의 부하 측정 (4.2 초) 과 같은 종류로 보고 다시 측정하지 않았다 — 옮기면 그 멈춤이 명령 앞으로 간다 |

---

## 3. 묻지 않고 정한 것

### 3.1 D2 — spool 잠금은 읽고 적을 때만 쥔다

| 차례 (규칙 6절) | 잠금 |
|---|---|
| 1 기록 읽기 · 만료 · 측정할 목록 | 쥔다 → 놓는다 |
| 2 측정 (helper) | **쥐지 않는다** |
| 3 ~ 7 측정값 적기 · 퇴출 · 요약 쓰기 | 다시 쥔다 — 기록을 다시 읽고, 그새 형제가 퇴출하거나 만료한 항목이면 측정값을 버린다 |

**까닭** — 측정은 upper 하나에 2.6 ~ 3.95 초이고 (ADR-077 §12) 찬 캐시는 모른다 (2.3). 쥔 채 측정하면 형제의 판정과 기동 조정이 그만큼
기다린다 (P5). 측정은 읽기만이라 잠금이 없어도 트리를 바꾸지 않는다. 형제가 같은 항목을 동시에 측정하면 일이 두 번일 뿐 결과는 같다.
옮긴 트리를 측정한 값은 3 의 다시 읽기에서 버린다.

### 3.2 D3 — 판정이 요약을 적고 예약이 읽는다

- 판정이 끝날 때 잠금 안에서 `<spool>/usage.json` 을 쓴다 — `kept` 의 측정한 바이트 합 · inode 합 · 시각 (임시 파일과 rename · 0600 ·
  fsync 없음)
- 예약이 그것을 읽어 보존 총량으로 쓴다. 어느 노드의 판정이든 spool 전체의 기록을 읽으므로 (규칙 6절 1) 형제의 보존본이 든다
- 없거나 못 읽으면 0 으로 센다 — 오늘 규칙 5절과 같이 느슨할 뿐 판정이 막는다

**까닭** — 오늘 규칙 5절은 자기 판정의 값만 들고 있어, 형제가 채운 spool 에 새 것을 받아들인 뒤 판정이 형제의 오래된 것을 퇴출한다.
받아들임이 `quota` 로 거절해야 할 자리다 (ADR-076 §2 표). 읽기는 15 µs (2.2) 이고 D1 이 A 면 창 밖에서 읽는다. 조회 (`list`) 는 이 파일을
보지 않는다.

### 3.3 D4 — 깸은 삭제자와 같은 모양이다

- Keeper 의 판정 고루틴 하나 · 깸은 자리 하나짜리 채널 (`scratch/deleter.go:68` · `:81` 과 같다). 돌 때 온 깸은 끝난 뒤 한 번으로 합친다
  — 보고 뒤 · 10분 · 조정 뒤 어느 것이든
- 측정 helper 는 항목 하나에 하나 · 차례로 · IO 우선순위 idle 과 CPU 19 (`trash_linux.go:53` 과 같다). 입구는 trash-helper 의 측정 동작
  (흐름 5절). 삭제자의 helper 와 겹쳐 돌 수 있다 — 둘 다 idle 이다
- helper 를 못 띄우면 그 판정을 멈추고 다음 깸을 기다린다 (규칙 6절 · 삭제자와 같다)

### 3.4 범주

| 범주 | 판정 | 근거 |
|---|---|---|
| 회복 | 해당 — 새 패턴 없음 | 오류는 로그와 다음 깸 (R3) · 주인 없는 예약은 기동 조정이 거둔다 (규칙 9절 · R2). D1 이 A 면 거둘 예약이 도는 단계마다 하나 늘 뿐 길은 같다 |
| 규모 | N/A | 항목 수는 몫과 TTL 이 막는다 (S1). D3 로 받아들임이 형제 수와 무관한 한 번의 읽기다 |
| 성능 | 해당 | D1 · D2 · D3 |
| 보안 | 새 패턴 없음 | C1 ~ C8 그대로. D1 이 A 면 spool 의 `O_NOFOLLOW` 열기 (C8) 가 세션을 열 때로 간다. `usage.json` 은 0600 이고 경로를 담지 않는다 |
| 논리 구성요소 | 해당 | Keeper (`internal/enode`) · Store (`internal/scratch`) · 판정 고루틴 · 측정 helper · spool 의 `usage.json`. 큐 · 캐시 · 회로 차단기는 없다 — 노드 한 대 안의 파일 일이다 |

---

## 4. 물음 하나

답을 `[Answer]:` 뒤에 적는다. 권장을 **A** 에 둔다.

**내기 전에 대 본 것**

| 흠 | 대 본 결과 |
|---|---|
| 이미 정한 것을 다시 묻기 | 판정 차례 (규칙 2절) · 받아들임을 닫을 때 보는 것 (결정 2-7) · By 의 마감 (규칙 2절 ③ · ⑤) · 32 GiB 와 몫 (NFR 1절) 은 묻지 않는다. D2 ~ D4 는 답이 하나라 3절에 정했다 |
| 측정 없는 근거 | 선택지마다 2.1 의 코드 줄과 2.2 · NFR 계획 2절의 값을 단다 |
| 앞 단계 넘김과 어긋남 | 세 선택지가 NFR 6절 D1 의 후보 셋 그대로다. 셋 다 ADR-076 §4 의 「rename 앞이면 `rejected`(`lease_budget`)」를 By 로 지킨다 |
| 요구를 빼는 선택지 | **B 는 P2 를 온전히 지키지 못한다** — 예약이 여유보다 오래 멈추면 finalize_timeout 이 날 수 있다. 진행자가 넘긴 후보라 선택지에 두고 그 흠을 선택지 안에 적었다 |

### Question 1 — 보존이 단계의 finalize 판정을 바꾸지 않게 하는 모양 (D1)

오늘 흐름 1절의 예약 (mkdir · 잠금 · 기록) 은 Finalize 와 Close 사이에 있어 마감 판정 (`claim.go:884`) 에 든다. 쓰기 부하에서 한 번
4.2 초 멈췄다. 어느 선택지든 받아들임 (여유 · 보존 총량 · 결정 2-7) 과 rename 과 확정은 닫을 때 그대로다.

A) 예약을 세션을 열 때로 옮긴다 (`claim.go:712` · `bake_build.go:196`) — 정책이 off 가 아니고 runtime 이 지원하고 spool 을 받아들인
노드에서 단계마다 예약하고, 닫을 때 규칙 1 · 2절대로 쓰거나 버린다. 보이는 것 — 성공한 단계마다 spool 에 폴더 하나를 만들었다가 보고
뒤에 지운다 (67 µs · 36 µs) · 도는 단계의 예약은 `list` 에 안 보인다 (잠금이 쥐어진 `reserved` 를 뺀다 · 주인 없는 것만 `incomplete`) ·
예약이 세션을 열 때 실패하면 단계는 돌고, 닫을 때 요구하면 `failed`(`io`) 에 그때의 errno · 부하에서 예약이 멈추면 명령의 시작이 그만큼
늦는다

B) 남은 예산에 여유를 둔다 — 남은 Finalize 예산이 10초 아래면 예약하지 않고 `rejected`(`lease_budget`). 예약 자리는 오늘 그대로.
보이는 것 — Finalize 가 50초를 넘긴 실패 단계는 보존되지 않는다 · 예약이 10초보다 오래 멈추면 여전히 finalize_timeout 이 날 수 있다
(측정 최대 4.2 초)

C) finalize 판정에서 보존에 든 시간을 뺀다 — `closedLate` 를 Close 가 끝난 시각에서 보존 시간을 뺀 값으로 본다 (`claim.go:884` ·
`finalize.go:223` · finalize 유닛의 파일). 보이는 것 — receipt 의 `finalized_at - exited_at` 이 예산을 넘어도 finalize 는 `ok` 일 수 있다 ·
`services.md:38` 「예산이 닫기까지 덮는다」의 예외

D) Other (please describe after [Answer]: tag below)

**권장 A.** 보존의 느린 일이 판정의 창에서 빠지고, 창 안에 남는 것은 오늘도 있는 종류의 rename 하나와 µs 의 stat 이다. receipt 의
시각과 판정이 어긋나지 않는다 (C 의 흠). 여유 값을 고를 일이 없고 P2 를 온전히 지킨다 (B 의 흠). on-failure 의 뜻은 그대로다 — 예약은
판정이 아니고, 요구하는지는 닫을 때 규칙 1절이 정한다. 대가 — 성공한 단계의 디스크 일 (폴더 하나 · 0.1 ms) 과 `list` 의 규칙 한 줄.

[Answer]: A

---

## 5. 산출물 계획 (Step 5 이후)

자리는 `aidlc-docs/v4-run-finalize-bake/construction/checkpoint/nfr-design/` 이다.

- [x] 답을 읽고 모호함을 본다 — 있으면 `checkpoint-nfr-design-clarification-questions.md` (2026-09-30 · A · 모호함 없음 · 되물음 파일 없음 ·
      규칙 1 · 2절과 ADR-076 §4 에 대 봤다 — `nfr-design-patterns.md` 1절)
- [x] `nfr-design-patterns.md` — D1 (답) · D2 · D3 · D4 의 패턴 · 범주 (3.4) · P1 ~ P7 과 C1 ~ C8 이 어느 패턴에 기대는지 한 표
- [x] `logical-components.md` — Keeper · Store · 판정 고루틴 · 측정 helper · spool 의 자리 (`usage.json` 더함) · 부르는 차례 (세션 열기 · 닫기 ·
      보고 뒤 · 기동)
- [x] FD · NFR 에서 고칠 곳을 고쳤다 (진행자 지시 · 표지 「NFR Design … 로 고침」 · FD 흐름 10절 표) — 처음 계획은 진행자에게 넘긴다였다 — 흐름 1절 · 규칙 2절 ⑤ · 13절 (답 1) · 규칙 5 · 6절 · 엔티티 4절 (D2 · D3) · NFR 2절 P5 의
      문장 (D2)
- [x] 검사 — `enode-design/scripts/emphasis-check.py` · 말투 grep · 사내 이름 grep

---

## 6. 확장 준수 — 이 단계

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 보안 패턴은 3.4 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B. 회복은 3.4 |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X |
