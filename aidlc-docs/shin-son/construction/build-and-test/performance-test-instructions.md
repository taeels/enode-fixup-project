# Performance Test — queue

**수치 목표가 없다** (`nfr-requirements.md` 4.1). 잰 것은 비용이 어디서 나는가다.

```text
   빈 큐의 훑기      임대 해제마다 한 번.  runs_queued_idx (부분 인덱스) 조회 하나.
                    실측 — 대기 0 인 함대에서 정산 tx 에 붙는 문장이 SELECT 한 줄
   찬 큐의 훑기      행 수 x match.Match (메모리 · 순수 함수).  대기 행은 사람 손이 낸 수
   잠금 대기         정산 · 취소 · 부분 반납 · 대기 진입이 한 advisory 키 뒤에 선다.  임계 구간 ms
   CP2 의 지연       a 의 결과 보고 응답 안에서 b 가 승격됐다 (로그 2ms 차이)
```

부하 시험을 따로 만들지 않는다 — 팩에 없는 값이다. 대기 행이 수백을 넘는 함대가
생기면 `EXPLAIN ANALYZE` 로 `runs_queued_idx` 가 타는지를 먼저 본다.
