# `step-phase` — 규칙

Mediator 가 종료 보고를 받고, phase 를 바꾸고, 결과를 봉인하고, 대기 사유를 세는 규칙이다. 응답
문구는 영어다 (`CONVENTIONS.md` 2.1 — 에러 문자열은 밖으로 나간다). 타입과 이름은
`domain-entities.md` 에 있다.

**원칙 둘**

- **종료 보고는 판정이 아니다** (결정 1-7 · 1-8). `success_when` 대조 · 다음 단계 생성 · 정산 ·
  임대 해제를 하지 않는다. 종결 전이는 result 하나다
- **result 는 종료 보고의 수락에 기대지 않는다** (결정 1-9). 종료 보고가 유실되거나 거절돼도 result
  하나로 봉인된 사실이 같다

---

## 1. 종료 보고의 수락 (답 1 = A)

`POST /v1/runs/{run}/steps/{seq}/exited`. 인증은 오늘의 bearer 토큰이다 (result 와 같다).
위에서부터 처음 맞는 줄이 결과다. 「키」는 Run · seq · attempt · 노드 · instance 다.

| 순서 | 경우 | 결과 | 바뀌는 것 |
|---|---|---|---|
| 1 | Run 이 없다 | 404 `no such run` | 없음 |
| 2 | 단계가 없다 | 404 `no such step` | 없음 |
| 3 | Run 이 RUNNING 이 아니다 (취소 · 종결 · QUEUED) | 200 · 받지 않음 | 없음 |
| 4 | 단계가 종결됐다 (DONE · FAILED · SKIPPED) | 200 · 받지 않음 | 없음 |
| 5 | ask · acquire 단계다 | 409 | 없음 |
| 6 | merge 단계다 | 200 · 받지 않음 — merge 는 명령이 없다 | 없음 |
| 7 | 보낸 attempt 가 지금 회차보다 작다 | 200 · 받지 않음 — 늦은 도착 | 없음 |
| 8 | 보낸 attempt 가 지금 회차보다 크다 | 409 | 없음 |
| 9 | 단계가 CLAIMED 가 아니다 (PENDING · ASKED) | 409 | 없음 |
| 10 | 노드가 `steps.node_id` 와 다르다 | 409 | 없음 |
| 11 | instance 가 `steps.claimed_instance` 와 다르다 | 409 | 없음 |
| 12 | phase 가 이미 finalizing 이다 (같은 키의 재전송) | 200 · 받지 않음. 처음 값이 남는다 | 없음 |
| 13 | 그 밖 — phase 가 running 이거나 NULL | 200 · **받음** | phase · phase_since · exit |

- 13 의 NULL — 이 코드가 들어오기 전에 claim 된 단계다. running 과 같이 받는다
- 12 에서 본문의 `outcome` 이나 `exited_at` 이 처음 값과 다르면 로그에 경고 한 줄을 남긴다. 응답은 같다
- 3 의 QUEUED — 단계가 아직 없으므로 실제로는 2 에 걸린다. 적어 둔 것은 순서를 분명히 하려고다

**거절 문구 (409)**

| 순서 | 문구 |
|---|---|
| 5 | `step %s is an %s step; only a step that runs a command reports an exit` |
| 8 | `step %s attempt %d has not started; the current attempt is %d` |
| 9 | `step %s is %s, not claimed; nothing ran that could exit` |
| 10 | `step %s is claimed by another node` |
| 11 | `step %s is claimed by another instance of this node; a restarted node cannot report the exit of an earlier life` |

`%s` 의 단계 표기는 오늘의 `run_id#NN` 이다. 10 은 다른 노드의 id 를 문구에 싣지 않는다 — 보낸
쪽이 모르는 노드의 이름을 알려 줄 까닭이 없다.

**응답 본문 (200)** — `{"run_id": …, "seq": …, "accepted": true | false}`. `accepted` 는 이 보고가
행을 바꿨는가다. 노드는 둘 다 성공으로 보고 재시도를 멈춘다. 409 에서도 멈춘다 (finalize 유닛이 따른다).

---

## 2. 종료 보고의 본문 검사 (답 2 = A)

`contract.Exited.Check` 가 본다. 형식만 본다. 틀리면 400 이다. 1절보다 먼저 본다.

| 규칙 | 문구 (400) |
|---|---|
| JSON 이 아니다 · `exited_at` 이 RFC 3339 가 아니다 | `cannot parse exited: %v` |
| seq 가 양의 정수가 아니다 | `invalid step sequence: %s` (result 와 같다) |
| `node` 가 비었다 | `exited: node is empty` |
| `instance` 가 비었다 | `exited: instance is empty; the report is matched against the instance that claimed the step` |
| `attempt` 가 음수다 | `exited: attempt must not be negative` |
| `exited_at` 이 없다 (영의 시각) | `exited: exited_at is missing` |
| `outcome.kind` 가 exit · signal · timeout 밖이다 | `exited: outcome.kind %q is not exit, signal or timeout` |
| `outcome.kind` 가 exit 인데 `code` 가 없다 | `exited: outcome.kind exit needs a code` |

- `exited_at` 의 값(미래 · 과거 · started_at 보다 앞)은 보지 않는다 (3절)
- signal 의 `code` 는 신호 번호이고 있어도 되고 없어도 된다. timeout 도 같다
- `exit` 칸에는 `outcome` 만 적는다. `exited_at` 은 `phase_since` 에 이미 있다

---

## 3. 두 시계 (답 3 = A)

칸마다 시계를 하나로 정한다. 두 시계의 차이를 재지도 고치지도 않는다 — 두 기계의 시각은 NTP 가
맞춘다고 본다.

| 자리 | 칸 | 시계 | 적는 때 |
|---|---|---|---|
| 진행 조회 · Record | `started_at` | Mediator | claim (재전달이면 다시) — 오늘 그대로 |
| 진행 조회 · Record | `ended_at` | Mediator | result 를 받은 때 — 오늘 그대로 |
| 진행 조회 | `phase_since` (running · waiting) | Mediator | claim 때. `started_at` 과 같은 값 |
| 진행 조회 | `phase_since` (finalizing) | **노드** | exited 의 `exited_at` |
| Record | `exited_at` | **노드** | result 의 `exited_at` · 없으면 exited 로 받은 값 |
| Record | `finalized_at` | **노드** | result 의 `finalized_at` |
| result | `build` · `merge` 안의 시각 | **노드** | 노드가 적는다 |
| 진행 조회 | `candidates_at` | Mediator (DB) | 셈을 한 때 |

**`phase_since` 는 phase 에 따라 시계가 바뀐다.** 이 한 줄을 `StepView` 의 godoc 과 정본 진행 조회 절에
적는다. finalizing 이 노드 시계인 까닭은 US-8 의 상한 「`phase_since` + finalize 예산 + upload 예산」을
노드가 자기 시계로 지키기 때문이다 (실행 계획 7절 ② — 두 시계).

---

## 4. phase 의 전이 (답 4 = A)

| 사건 | 조건 | phase | phase_since | exit |
|---|---|---|---|---|
| claim | kind 가 merge 가 아님 | running | `now()` (started_at 과 한 문장) | NULL |
| claim | kind 가 merge | waiting | `now()` | NULL |
| 재전달 (같은 생 · `claim.go:510`) | | claim 과 같이 다시 | `now()` | NULL |
| exited 수락 | 1절 13 | finalizing | 본문의 `exited_at` | 본문의 `outcome` |
| 되돌림 · 되감기 (PENDING 으로) | 셋 (`rollback.go:76` · `:130` · `ask.go:660`) | NULL | NULL | NULL |
| result (DONE · FAILED) | | 그대로 | 그대로 | 그대로 |
| 재시작 실패 (`FailRestarted`) · 임대 만료 · 취소 · 계획 거절 | | 그대로 | 그대로 | 그대로 |

- **종결 전이는 phase 칸을 건드리지 않는다.** 마지막 값이 남는다. 그래서 종결 경로 다섯을 하나씩 안
  고친다. 남은 값은 Record 의 `last_phase` · `exit` 가 된다 (6절)
- 되돌림 셋은 오늘 `result` · `started_at` · `ended_at` 을 지운다. 새 칸 셋도 같은 문장에서 지운다.
  **이 셋은 파일 행렬에 없던 자리다** — 안 지우면 다음 회차가 옛 회차의 finalizing 을 물려받는다
- 재전달은 「시작도 못 한 단계」의 claim 응답이 유실된 경우다 (`claim.go:250` 의 주석). 명령이 안
  돌았으므로 claim 과 같이 다시 적는다
- ask 단계의 ASKED 는 phase 를 안 적는다 — 노드가 수행하지 않는다

---

## 5. result 에 더하는 칸 (답 5 = C · 되물음 4 = A)

- **값으로 거절하지 않는다.** 모르는 `finalize` · `upload` · `reason` · `checkpoint_capture.state` ·
  `diagnostics.changes` 도 그대로 봉인한다. 노드는 result 를 임대가 죽을 때까지 재시도하므로 400 은
  그 보고를 잃게 한다. 어휘 밖의 값은 로그에 경고 한 줄을 남긴다 —
  `result carries an unknown value` 와 칸 이름 · 값
- 모양이 틀린 JSON(예: `build` 가 배열)은 오늘처럼 파싱 오류 400 이다 (`api.go:412`). 모양은 타입이
  정하고, 어휘는 막지 않는다
- result 의 `exited_at` 과 exited 의 `exited_at` 이 달라도 대조하지 않는다. Record 는 result 의 값을
  쓴다 (6절)
- result 를 받는 조건(노드 · CLAIMED)은 바꾸지 않는다. 인스턴스 대조는 잔여다 (FR-2 · 5.3)

---

## 6. Record 의 단계 기록 (답 3 · 4 = A)

| 칸 | 규칙 |
|---|---|
| `exited_at` | result 에 있으면 그 값. 없고 `steps.exit` 가 있으면 `steps.phase_since` (exited 로 받은 값). 둘 다 없으면 없음 |
| `finalized_at` | result 에 있으면 그 값. 없으면 없음 |
| `exit` | `steps.exit` 가 있으면 그 값 |
| `last_phase` | `steps.phase` 가 있으면 그 값 |

**읽는 법** — 결과 없이 FAILED 로 끝난 단계가 `last_phase: finalizing` 과 `exit` 를 가지면 「명령은
끝났고 결과 확정은 못 끝냈다」다 (services.md 「Finalize 중 노드 죽음」). `exit` 가 없고
`last_phase: running` 이면 명령이 끝났다는 보고가 오지 않았다 — 옛 노드이거나 명령 도중에 죽었다.

---

## 7. 진행 조회 (답 4 · 6 = A)

| 칸 | 싣는 조건 |
|---|---|
| `steps[].phase` · `steps[].phase_since` | 단계가 CLAIMED 이고 값이 있을 때 |
| `steps[].exit` | 값이 있을 때. 상태와 무관 |
| `requires[].candidates` · `candidates_at` | Run 이 QUEUED 일 때. `GET /v1/runs/{id}` 에서만 |

- 목록 `GET /v1/runs` 와 제출 응답에는 후보 수를 안 싣는다 — 폴링하는 목록이 Run 마다 셈을 하게 된다
- 후보 수를 못 세면 `candidates` 와 `candidates_at` 을 빼고 `warnings` 에 한 줄을 적는다 —
  `candidate counts could not be read; candidates is omitted` (requires 를 못 읽을 때와 같은 방식)

---

## 8. 대기 사유 — 후보 수 (답 6 = A)

요구 줄 `r` 마다 센다. 판단은 매처와 같은 `contract.Advert.Satisfies` 다. `match.Match` 는 부르지
않는다 — 수를 돌려주지 않고 첫 실패에서 멈춘다. `internal/match` 는 안 바뀐다.

```text
   live       만료 안 된 광고 a 가운데 a.Satisfies(r)
   busy       live 가운데 a.NodeID 가 임대를 쥐었다 (leases 표 — 어느 Run 이든)
   draining   live 가운데 busy 가 아니고 nodes.draining 이 빈 문자열이 아니다
```

- **배타다.** busy 를 먼저 센다. 점유된 노드는 drain 이 없어도 못 받으므로, `draining > 0` 이 곧
  「drain 만 아니면 받을 수 있는 노드가 있다」다 (US-7)
- 요구 줄마다 따로 센다. 한 노드가 두 줄을 만족하면 두 줄에 다 든다. 원하는 수는 오늘의
  `requires[].count` 다
- 남는 수(`live - busy - draining`)를 칸으로 내지 않는다. ADR-065 가 기각한 `free` 다 — 확인 뒤 행동
  경쟁을 공식 표면으로 만든다. 셋 다 관측 사실이고 배정 판정이 아니다
- 광고 · 임대 · drain 을 **한 스냅샷**에서 읽고 그 스냅샷의 DB 시각을 `candidates_at` 으로 낸다
  (`observe.go` 의 규칙 — 목록과 시각이 같은 시점에서 나온다)
- drain 의 출처(소유자 · 여유 부족 · 굽기)는 가르지 않는다. 완료 조건 4 가 요구하지 않고, 출처는
  광고 어휘를 늘린다 (US-7 의 비고)

---

## 9. `Claimed` 에 계약 칸을 싣기 (답 7 = A)

- `fillFromContract` 가 계약의 단계에서 칸 일곱을 그대로 옮긴다. claim 과 재전달이 같은 함수를
  지나므로 같은 칸이 실린다
- 기본값을 채우지 않는다. 안 적은 칸은 JSON 에서 빠진다 (`omitempty`)
- 계약이 contract-grammar 의 `Validate` 를 지났으므로 칸의 값은 이미 맞다. 여기서 다시 검사하지 않는다
