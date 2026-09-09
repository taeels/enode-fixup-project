# Integration Test — queue (CP2 「기다린다」)

정본은 `requirements/scene-gates.md` 3절 CP2 다. 실제 Mediator 하나와 실제 노드 하나로
잰다. 시험 DB 와 **다른 판**을 쓴다.

## 준비
```bash
createdb -p 55434 -U enode -O enode enode_cp2
go build -o ./bin/mediator ./cmd/mediator && go build -o ./bin/enode ./cmd/enode && go build -o ./bin/runctl ./cmd/runctl
```

`mediator.yaml` — `listen: "127.0.0.1:18080"` · `token` · `database.url` 을 `enode_cp2` 로 ·
`artifacts.root` · `lease.renew_seconds: 5`. `local.yaml` — `mediator` · 같은 `token` ·
`principal` · `workspace` · `board.soc` 하나(선언만으로 능력이 된다).

```bash
./bin/mediator --config ./mediator.yaml &
./bin/enode --config ./local.yaml --every 5s --ready-file ./ready &
export ENODE_MEDIATOR=http://127.0.0.1:18080 ENODE_TOKEN=<token>
```

계약 셋은 `runctl example command` 에서 만든다 — `a.json` 은 단계가 `sleep 25` 를 하고,
`b.json` 은 같은 요구, `c.json` 은 요구에 함대에 없는 속성(`"board": "no-such-board"`)을
형제 키로 편다. `runctl lint` 가 셋을 통과해야 한다.

## 명령과 기대값 (scene-gates 3절 CP2)
```text
   runctl submit a.json                          -> RUNNING
   curl -i -X POST $M/v1/runs -d @b.json         -> HTTP/1.1 202 Accepted · {"run_id":"cp2-b","state":"QUEUED"}
   runctl status cp2-b                           -> QUEUED
   GET /v1/runs?state=QUEUED                     -> cp2-b 한 행 · assigned []
   GET /v1/runs/cp2-b                            -> state QUEUED · requires (obs)
   a 가 끝난 뒤 runctl status cp2-b              -> RUNNING (또는 이미 SUCCEEDED — 단계가 짧으면)
   runctl submit c.json                          -> stderr 422 need 1, fleet has 0
   runctl status cp2-c                           -> FAILED · reject.Code 422
```

승격이 **같은 트랜잭션**에서 났는지는 Mediator 로그의 순서로 본다 —
`promoted from queue run=cp2-b` 가 `run finished run=cp2-a` 보다 앞이어야 한다.

## 2026-09-09 실측
전부 기대값 그대로. 로그 순서 — promoted 09:50:58.073 · run finished cp2-a 09:50:58.075 ·
run finished cp2-b 09:50:58.084. b 의 단계가 4ms 라 폴링은 RUNNING 대신 SUCCEEDED 를 봤다.
c 는 `422 need 1, fleet has 0 - agent.reason board=no-such-board` · FAILED.

화면 항목(QUEUED 행 · chosen)은 ui(runixs) 몫이다 — 여기서는 API 로만 잰다.

---

# Integration Test — drain (CP3 「돌려받는다」 · 2026-09-09)

정본은 `scene-gates.md` 3절 CP3. queue 와 같은 판(`enode_cp2`) · Mediator · 노드 하나.
노드 설정이 `local.yaml` 이면 정책 파일은 그 옆의 **`local.policy.yaml`** 이다.

## 명령과 기대값
```text
   단계 둘 계약 a 제출 (각 단계가 잠시 돈다)     -> RUNNING
   printf 'drain: at-boundary\n' > local.policy.yaml   다음 광고(≤ 광고 주기) 뒤
   GET /v1/nodes                                -> draining "at-boundary" · lease 있음
   단계 1 이 끝난 경계                            -> runctl status a → FAILED · verdict note "cancelled by: drain:<node_id>"
                                                   · s1 DONE · s2 FAILED · GET /v1/nodes lease 없음
   runctl record a                              -> 단계 1 산출이 있다 (blobs/01.0-*.txt)
   새 계약 b                                    -> 202 QUEUED
   rm local.policy.yaml (해제)                   -> 다음 광고 뒤 b RUNNING · draining ""
   graceful 로 같은 절차                         -> 도는 Run 이 끝까지 SUCCEEDED
```

## 2026-09-09 실측
전부 기대값 그대로. Mediator 로그 `run closed at a step boundary; the node is draining` ·
`queued` · `promoted from queue`. 노드 로그 `drain acknowledged by mediator mode=at-boundary` ·
`draining at-boundary; not claiming until the owner releases it` · `drain released; claiming again`.
