# queue — Business Rules

규칙마다 **정본 출처**를 단다. 출처가 없는 규칙은 이 유닛이 정한 것이고 그렇게 적는다.

---

## 1. 상태 규칙

| 규칙 | 출처 |
|---|---|
| `QUEUED` 에서 참인 것 셋 — 임대가 0 · 노드를 쥐지 않는다 · 재기동 후에도 그대로 참 | `INVARIANTS` §1.1 |
| `QUEUED` 로 가는 전이는 `ALLOCATING -> QUEUED` 하나. 자리 둘(매처 AllBusy · ErrNodeTaken)이 같은 전이다 | `INVARIANTS` §2 · `enode-features` 3.2.1 |
| `QUEUED` 에서 나가는 길은 둘 — 승격(`-> RUNNING`) · 취소(`* -> FAILED`). 시간은 보내지 않는다 (대기 상한 없음) | `ADR-064` §5·§6 · `decisions` §1 |
| 승격은 후보 집합을 다시 계산한다 — 광고가 바뀌었을 수 있다 | `INVARIANTS` §2 (`QUEUED -> ALLOCATING`) |
| `422`(함대에 없다 · 폭 상한)는 그대로 `FAILED`. 기다려도 안 되는 것을 기다리게 하지 않는다 | `ADR-064` §2 |
| `READY` 는 만들지 않는다 | `ADR-064` §1.1 · `constraints` §7 |

## 2. 정산 · 회수가 `QUEUED` 를 건드리지 않는다

| 규칙 | 출처 |
|---|---|
| `Reap` 의 만료 회수는 `leases` 가 있는 Run 만 본다 — `QUEUED` 는 대상이 아니다 | 코드 사실 (`reap.go` 의 `run_id IN (SELECT run_id FROM leases ...)`) |
| **`SettleIfDone` 은 `QUEUED` 를 정산하지 않는다.** 단계 0 인 Run 이 들어오면 `broke == 0 && pending == 0` 으로 판정 갈래에 들어 `SUCCEEDED` 로 닫힐 수 있다. 오늘 호출자 셋(postResult · 재기동 감지 · ask 만료)은 `QUEUED` 를 안 넘기지만, 조건이 코드에 없다. `state = 'RUNNING'` 인 Run 만 정산하도록 앞에 보호를 두고 테스트로 잡는다 | **이 유닛이 정한다.** 근거 — 상태가 하나 늘면 크래시 복구 검증 대상이 는다(`INVARIANTS` §1.2). 그 검증이 이 줄이다 |
| 승격 실패(후보 0 · 전부 busy · 폭 초과)는 오류가 아니고 상태를 안 바꾼다 | `ADR-064` §5 · `decisions` §1 「출구는 cancel 뿐」 |

## 3. 응답 규칙

| 규칙 | 출처 |
|---|---|
| 대기 응답은 `202 Accepted`. 본문은 `201` 과 같은 모양 · `state:"QUEUED"` · `assigned` 비움 | `mediator-api` §2 · `decisions` §1 |
| `steps` 는 승격 전엔 비어 있다 (Q2 = A). `view()` 의 「배정 전이면 단계가 없는 것이 정상」 그대로 | 계획 Q2 답 |
| 같은 `run_id` 재제출은 `QUEUED` 여도 `200` + 기존 Run | `INVARIANTS` §2 첫 행 |
| dry-run 은 busy 도 draining 도 안 본다 — `202` 가 나올 수 없다. 합치는 자리는 `if !dry` 안 | `ADR-014` 결정 3 · `ADR-063` §6 · `decisions` §2 |
| `CreateQueuedRun` 의 첫 훑기가 그 자리에서 승격했으면 응답은 `201` + `RUNNING` (다시 읽는다) | **이 유닛이 정한다.** 근거 — 응답이 DB 보다 낡으면 껍데기가 한 폴링을 헛돈다. `mediator-api` §2 의 「202 = 대기에 섰다」가 거짓이 되지 않게 한다 |
| 취소된 `QUEUED` 는 `FAILED` · `verdict.checks[0].what = "cancelled"` · 봉인까지 `Cancel` 그대로. 단계 0 인 Record 가 생긴다 | `ADR-009` · `ADR-005`(왜 안 돌았는지가 Record 에 남는다) · 계획 §4 |
| `runs.reject` 는 `QUEUED` 에서 NULL. `409` 는 매처 안의 이름으로만 남는다 | `INVARIANTS` §2 개정 · `mediator-api` §1 |
| `QUEUED` 가 무엇을 기다리는지는 `GET /v1/runs/{id}` 의 `requires` (obs) — 이 유닛은 새 열을 안 낸다 | `ADR-069` · `decisions` §7 |

## 4. 순서 · 동시성 규칙

| 규칙 | 출처 |
|---|---|
| 공정성은 **FIFO 하나** — `created_at, run_id` 오름차순. 정책 틀 · 우선순위 · 상한 없음 | `ADR-064` §2.1 · `constraints` §2 |
| **전체를 훑고, 못 가는 것은 건너뛴다.** 맨 앞이 보드를 기다린다고 뒤의 다른 자원 요구까지 세우지 않는다. 같은 자원을 두고 겨루면 도착순이다. 한 훑기 안에서 앞 행이 잡은 노드는 뒤 행의 busy 에 든다 | 유닛 정본 「FIFO 전체 훑기(Reap·drain 해제 대응)」 · 기아는 `ADR-064` §6 이월 |
| **큐 변경은 advisory xact lock 하나로 직렬화한다** — `CreateQueuedRun` 과 `wakeQueuedIn` 이 같은 키를 잡는다. 넣기와 첫 훑기가 한 tx 이고, 임대 해제와 훑기가 한 tx 다. 그래야 「둘 다 서로를 못 본」 Run 이 안 생긴다 | **이 유닛이 정한다.** 근거 — 계획 §3.5 의 경쟁 창. `enode-features` 3.2.1 「풀렸는데 못 보는 창도 없다」를 코드로 만드는 방법 |
| 깨우기는 **지점**에서, **부르는 tx 안에서 동기**로. 지점 여섯 + 기동. 광고마다 훑지 않는다 | `decisions` §1 「깨우기」 · `enode-features` 3.2.1 · 계획 §4 |
| `Reap` 은 tx 가 없으므로 `WakeQueuedNow`(자기 tx). 요청 경쟁 밖이라 CP2 의 동기 요구와 무관하다 | 계획 §3.4 |
| 승격의 임대 INSERT 가 unique 위반이면(잠금 아래서는 안 난다) 훑기 전체가 tx 오류로 롤백된다. 부분 승격은 없다 | `I5` |

## 5. 보안 확장 (security-baseline · 켜짐) — 이 유닛에서의 취급

`decisions.md` §3 의 취급표를 따른다. 이 유닛은 **새 HTTP 표면을 안 만든다** — 기존
`POST /v1/runs` 의 응답 코드 하나가 늘고 store 함수가 는다.

| 규칙 | 판정 | 근거 |
|---|---|---|
| SECURITY-03 로깅 | 준수 | 승격·대기 로그는 `run_id` · 노드만. 계약 본문·principal 을 안 찍는다 (obs 와 같은 규칙) |
| SECURITY-05 입력 검증 | N/A | 새 입력 없음. `QUEUED` 는 서버가 정하는 값이고 `state` 필터는 obs 의 것 |
| SECURITY-08 접근 제어 | 준수 | `POST /v1/runs` 는 `auth` 그대로. 202 가 인증 경계를 바꾸지 않는다 |
| SECURITY-14 감시 | 준수 | 승격은 Info 로그, 기동 훑기 실패는 Error 로그로 남는다 |
| SECURITY-15 fail-safe | 준수 | 승격 오류는 tx 롤백 → 부분 승격 0. 기동 훑기 실패는 기동을 안 막는다(대기 Run 은 다음 지점에서 본다) |
| SECURITY-01 · 07 · 12 | 기록만 | `decisions` §3 — 기존 코드 사실. 이 유닛이 새 표면을 안 내므로 닿는 곳 없음 |
| 나머지(02 · 04 · 06 · 09 · 10 · 11 · 13) | N/A | 네트워크 중간장치 · 브라우저 헤더 · 인프라 권한 · 공급망 · 무결성 검증 — 이 유닛의 산출물에 해당 자리 없음 |

## 6. 안 하는 것

```text
   우선순위 · 기아 방지 · 대기 상한 · 정책 틀     constraints §2 · ADR-064 §6·§7
   광고 도착마다 훑기                            지점 여섯 + 기동만.  계획 §4 에 기록
   GET /v1/runs 목록 열 · requires                obs
   drain 정책 파일 · at-boundary 취소              drain (W2 · 같은 담당)
   match 시그니처 · runctl 변경                   호출자가 합친다 · 2xx 성공
```
