# `checkpoint` — 논리 구성요소

패턴은 `nfr-design-patterns.md` (「패턴 N절」) 에 있다. 칸과 타입은 FD `domain-entities.md` (「엔티티 N절」) 다. 작성
2026-09-30T14:55:43Z.

---

## 1. 구성요소

| 구성요소 | 자리 | 하는 일 | 부르는 때 |
|---|---|---|---|
| Store | `internal/scratch` 새 파일 | 예약 · 확정 · 버림 · 판정 ② ~ ④ · 보고 뒤 판정 · 조정 · 목록 · 요약 읽기와 쓰기. `contract` · `lower` 를 모른다 | Keeper 가 부른다 |
| Keeper | `internal/enode` 새 파일 | 정책 · 규칙 1절 · 신원 모으기 · `scratch.Capture` 를 `contract.CheckpointCapture` 로 · 보고의 성패 · 기동 조립 | 세션 열기 · closeOut · report · 기동 |
| 예약 | Keeper 가 단계마다 드는 값 | ID · 항목 폴더 · 항목 잠금 · 열 때의 오류 | 세션을 연 뒤에 생기고 닫을 때나 보고 뒤에 끝난다 (패턴 1절) |
| 판정 고루틴 | `Keeper.Run` | 조정 (기동 때 한 번) · 보고 뒤 판정 · 예약 버리기. 깸 하나짜리 채널 | 보고 뒤 · 10분 · 조정 뒤 (패턴 4절) |
| 측정 helper | `enode trash-helper` 의 측정 동작 · `scratch.Measure` 가 spool 뿌리도 받는다 | 항목 하나를 namespace 안에서 걷는다 · idle IO · CPU 19 | 판정의 차례 2 (패턴 2절) |
| 요약 | `<spool>/usage.json` (0600) | `kept` 의 바이트 합 · inode 합 · 시각 | 판정이 쓰고 받아들임이 읽는다 (패턴 3절) |
| spool 잠금 | `<spool>/.lock` (flock) | 판정의 읽기 · 적기와 조정을 한 번에 하나로 | 패턴 2절 |
| 항목 잠금 | `<spool>/<ID>/.enode-session.lock` (flock) | 예약이 살아 있는 동안 쥔다 — 조정과 조회가 도는 단계의 예약을 알아본다 | 예약부터 확정이나 버림까지 |
| 조회 명령 | `internal/enode` 새 파일 · `cmd/enode` 는 넘기는 줄만 | `list` · `show` — 기록을 읽는다. 항목 잠금을 쥘 수 없는 `reserved` 는 뺀다 | 소유자가 부른다 |

---

## 2. 부르는 차례

```text
   기동         설정 검증 -> spool 자리 확인 (NFR C8) -> Keeper -> 상태 파일의 checkpoint 블록 · 기동 로그
                -> Keeper.Run: 조정 (spool 잠금) -> 판정 -> 깸을 기다린다        광고는 기다리지 않는다
   세션 열기     Keeper.Reserve -> Store: mkdir · 항목 잠금 · reserved 기록          실패하면 errno 만 든다
   닫기         Finalize -> diagnostics -> Keeper.Decide: 규칙 1절 · stat 둘 · statfs · usage.json 읽기
                -> session.Close(Keep{Upper, By, Result}) -> closedAt
                -> Keeper.Finish: kept 기록 · 항목 잠금 놓기 · 또는 버림 (항목 폴더째 trash)
   보고         Worker.report -> Keeper.Reported(ID, 성패) -> AfterReport: 삭제자 Kick · Keeper Kick
   보고 뒤       Keeper.Run: 남은 예약 버리기 -> spool 잠금 (읽기) -> 측정 (잠금 밖) -> spool 잠금 (적기 · 퇴출 · usage.json)
                -> 상태 파일 -> 삭제자 Kick (trash 에 넣었으면)
   조회         enode checkpoint list | show -> 기록 읽기 · 항목 잠금 시험 (쥐지 못하면 도는 단계라 뺀다)
```

---

## 3. 잠금

| 잠금 | 쥐는 쪽 | 길이 | 기다리는 쪽 | 안 기다리는 쪽 |
|---|---|---|---|---|
| spool 잠금 | 판정의 읽기 · 적기 · 조정 | 기록 읽기와 적기 (ms) | 형제의 판정 · 조정 | 예약 · 닫을 때의 판정 · 조회 · 측정 |
| 항목 잠금 | 예약한 Worker | 세션을 열 때부터 확정이나 버림까지 (단계의 길이) | 없음 — 모두 `LOCK_NB` 로 시험만 한다 | 조정 (쥘 수 없으면 건드리지 않는다) · 조회 (쥘 수 없으면 뺀다) |

두 잠금을 함께 쥐는 자리가 없다 — 판정은 항목 잠금을 쥐지 않고 `reserved` 를 건드리지 않는다 (규칙 6절은 `kept` · `evicted` 만 본다).
조정만 둘을 차례로 본다 — spool 잠금 안에서 항목 잠금을 `LOCK_NB` 로 시험한다.

---

## 4. 없는 것

| 무엇 | 까닭 |
|---|---|
| 큐 · 메시지 버스 | 깸은 채널 하나다. 노드 밖으로 나가는 것이 없다 |
| 캐시 | 요약 `usage.json` 이 그 자리다. 메모리의 보존 총량은 두지 않는다 (패턴 3절) |
| 회로 차단기 · 재시도 정책 | helper 를 못 띄우면 다음 깸까지 기다린다 (패턴 4절). 보고의 재시도는 오늘의 `Worker.report` 다 |
| 데이터베이스 · 색인 | 기록은 항목 폴더마다 JSON 하나다 (NFR S1) |
| Mediator 쪽 구성요소 | Mediator 는 `checkpoint_capture` 를 받아 봉인할 뿐이다 (`store/claim.go:933`) |
