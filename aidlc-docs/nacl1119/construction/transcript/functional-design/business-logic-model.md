# transcript — 업무 흐름 (business-logic-model)

하네스가 뱉는 글자를 도는 동안 노드의 링 파일에 흘리고, 제어판이 그것을 1초로
읽어 카드에 그린다. 지난 작업은 Mediator 가 이미 가진 것을 제어판이 읽어 온다.
정본은 `decisions.md` §6.2/§6.3/§6.5 · `unit-of-work.md` §6.

---

## 1. tee — 한 겹 (쓰기)

```text
   하네스 단계 (runner.go)
     기존   var stdout bytes.Buffer; cmd.Stdout=&stdout; cmd.Run(); h.Decode(stdout)
     바뀜   cmd.Stdout = io.MultiWriter(&stdout, j.Transcript)   // j.Transcript!=nil 일 때
            Decode 와 로그 반환은 그대로 &stdout 을 읽는다 — 계약 무변경
     흘리는 것  하네스 원문 stdout (decisions §6.2 Q2).  --output-format 을 안 건드린다

   명령 단계 (claim.go)
     기존   var buf bytes.Buffer; cmd.Stdout,cmd.Stderr=&buf,&buf; cmd.Run()
     바뀜   w := io.MultiWriter(&buf, ring); cmd.Stdout,cmd.Stderr=w,w
```

데몬은 제어판을 모른다 — 파일 하나로 이야기할 뿐이다(새 포트·IPC 없음 ·
decisions §6.2 Q1).

---

## 2. 링 쓰기 · 비우기 (데몬)

```text
   단계 시작   Worker 가 ring.Reset() — total=0 · generation+1.  파일·몸통은 그대로.
              끝날 때가 아니라 시작할 때 비운다 — 떠나 있던 사람에게도 방금 끝난 것이 남게 (decisions §6.2)
   쓰기        ring.Write(p): off = total % capacity 자리에 WriteAt.
              off+len 이 capacity 를 넘으면 두 조각으로 갈라 쓴다(끝에서 감김).
              머리의 total 을 len 만큼 올려 WriteAt.  Truncate·rename·삭제 안 함
   상한 초과   total 이 capacity 를 넘어도 안 깨진다 — 오래된 바이트부터 덮인다
```

Worker 배선 — 노드마다 Ring 을 한 번 열고(`OpenRing(TranscriptPath(Ident.Config), 512*1024)`),
단계가 CLAIMED 로 시작할 때 `Reset()`, 하네스에는 `Job.Transcript=ring`, 명령
단계에는 같은 ring 을 MultiWriter 에 넣는다. 한 노드는 한 번에 한 단계라 쓰는
쪽이 하나다.

---

## 3. 링 읽기 (제어판)

```text
   ReadRing(path):
     1. 머리를 읽는다 (magic 확인 · capacity · total · generation)
     2. 몸통(capacity)을 읽는다
     3. 머리를 다시 읽는다 — total 이 많이 움직였으면(>capacity 등) 2 를 한 번 더.
        찢긴 읽기 완화 (잠금이 없으므로 · decisions §6.3)
     4. total<=capacity 면 Data=몸통[:total]; 넘으면 start=total%capacity,
        Data=몸통[start:]+몸통[:start] (오래된 것 -> 새것 순서)
```

파일만 있으면 데몬이 죽어 있어도 읽힌다. 없거나 magic 이 틀리면 카드가 「아직
없음」.

---

## 4. 카드 폴링 (화면)

```text
   GET /api/transcript  -> {generation, total, data}
   화면이 1초 타이머로 폴링(열려 있을 때만).  Mediator 폴링(5초)과 별개다 —
   로컬 파일 읽기라 네트워크 부하 이유가 안 걸린다 (decisions §6.2).
   generation 이 바뀌면 화면 버퍼를 비우고 새로 채운다 (새 단계)
```

데몬 로그 카드(handleLogs · <name>.log)는 그대로 옆에 있다 — 둘이 다른 물건임이
화면에서 보인다 (CP6).

---

## 5. 지난 작업 (제어판 · Mediator 가 가진 것)

```text
   GET /api/runs
     -> runctl.Client.Runs(ctx, RunsQuery{}) 원문
     -> assigned 에 자기 node_id 가 있는 Run 만 걸러 목록(run_id·state·ended_at·verdict.checks)
   목록의 한 줄을 누르면
   GET /api/record?run=<id>
     -> runctl.Client.Record(ctx, id, w) 로 tar 를 받아
     -> archive/tar 로 logs/NN-<step>.log 만 꺼낸다 + 그 Run 의 verdict.checks 를 보인다
```

Mediator 변경 0 (decisions §6.4/§6.5). `node=` 서버 필터를 안 만든다 — 제어판이
`assigned` 로 거른다.

---

## 6. 흐름 한눈 (ASCII)

```text
   데몬(enode)                              노드 파일                제어판(internal/panel)
   ───────────                              ─────────                ──────────────────────
   단계 시작 -> ring.Reset()          ->    <stem>.transcript   <-   GET /api/transcript (1초)
   하네스 stdout --MultiWriter-->  ring          (링)                  세대 바뀌면 화면 비움
   명령 stdout/stderr --MultiWriter-->            |
                                                  v
                                            카드에 흐른다

   지난 것:  제어판 --GET /v1/runs--> Mediator  (assigned 로 거름)
             제어판 --GET /v1/runs/{id}/record--> tar --> logs/NN-*.log + verdict.checks
```
