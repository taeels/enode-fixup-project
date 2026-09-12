# 장면 조각 게이트 — 진행의 단위는 유닛 완료가 아니라 장면 완주다

장면 하나를 조각 일곱으로 잘랐다. **조각마다 실행 명령이 있고, 조각 게이트가
초록이 아니면 다음 유닛을 착수하지 않는다.** 이것은 요구사항의 **수용 기준**이다.

집행자는 **그 유닛을 구현하지 않은 사람**이다. 사내 MCP 가 필요한 조각은
사내에서만 돌고, 격리 조각은 가짜 stdio 서버(`true` 같은 실행파일)로
스크래치에서도 돈다.

---

# 1. 장면 — 사내 MCP 하나를 요구한 계약

```text
   ①  노드 소유자가 enode.yaml 의 mcp: 에 사내 MCP 하나를 적는다.  실행파일 또는 주소와
      자격증명 이름이다.  그 기계에는 개인 MCP 서버와 계정 커넥터도 있다
   ②  광고에 mcp.<이름> 이 실린다.  현황판의 노드 카드에 그 키가 보인다
   ③  계약이 requires 에 mcp.<이름> 을 적고, 단계가 agent.mcp 로 그 이름을 요청한다.
      첫 단계는 팩 저장소에서 tar 를 받는 명령 단계다
   ④  그 노드에만 배정된다.  키가 없는 노드는 후보에서 빠진다
   ⑤  하네스가 뜬다.  init 줄의 mcp_servers 에 agent.mcp 가 적은 이름뿐이다 —
      노드 것이든 팩 것이든 요청한 것만 실린다 (decisions.md 6절 ⑥).
      개인 서버 0 · 계정 커넥터 0.  팩의 스킬이 slash_commands 에 있다
   ⑥  에이전트가 그 도구를 불러 산출물을 낸다.  success_when 이 그것을 판정한다
   ⑦  runctl record 로 푼 묶음에 서버 이름과 팩 다이제스트가 있다
```

⑤ 와 ⑥ 이 이 장면의 가치다. **사내 실측에서 거부되던 것이 계약이 요청한 만큼만
열리고, 열린 것으로 실제 일이 된다.**

---

# 2. 조각 일곱

| | 조각 | 확인 (실동작) | 집행자 | 대상 | 재는 기능 | 먼저 서는 기능 |
|---|---|---|---|---|---|---|
| CA0 | 기동이 안 깨졌다 | 앞 팩의 CP0 그대로 — `scripts/testdb.sh` 뒤 기존 테스트 전부 초록 · 커버리지 · glyphscan · 포맷 · vet · 크로스 빌드 · 심볼 상한 · 워킹트리 청결. 기존 라우트 수가 그대로다 | 기계 | 스크래치 | 바닥 | (없음) |
| CA1 | 끊긴다 | `logs/` 의 `system/init` 줄로 잰다 — ⑮ 이 그 줄을 내게 했다 (옛 대안 「`enode` 가 남긴 프로브 출력」은 그것을 내는 유닛이 0 이라 기각됐다). **계정에 로그인된 기계에서** 개인 MCP 서버와 계정 커넥터가 있는 채로 아무것도 요청하지 않은 에이전트 단계를 돌린다. 로그아웃 홈에서만 재면 안 된다 — 이 설계가 `.credentials.json` 을 가짜 홈에 복사하므로 계정이 돌아온다 (`application-design.md` 4.6). `logs/` 의 `system/init` 줄(또는 `enode` 가 남긴 프로브 출력)에 `mcp_servers` 가 비어 있다. OAuth 로그인 노드와 게이트웨이 노드 둘 다에서 단계가 `Not logged in` 없이 돈다 | 사람 | 스크래치 | 3.1 · 3.2 | (없음) |
| CA2 | 광고한다 | `enode.yaml` 에 `mcp:` 하나를 적고 노드를 띄운다. `GET /v1/nodes` 의 그 노드 `attrs` 에 `mcp.<이름>` 과 `harness.claude` 가 있고 옛 `harness` 도 있다. 실행파일을 치우고 탐지 주기 뒤에 보면 `mcp.<이름>` 이 빠져 있고 노드 로그에 이유가 있다 | 사람 | 스크래치 | 3.3 | 3.1 |
| CA3 | 고른다 | `runctl capabilities` 에 `mcp.<이름>` 이 나온다 — 계약 작성자가 보는 면이다. `requires` 에 그 이름을 적은 계약이 그 노드에만 간다. 없는 키를 적으면 `422` 다. `agent` 에 모르는 키를 적으면 `400` 이고 `agent.mcp` 를 문자열로 적어도 `400` 이다. `runctl lint` 와 `runctl example` 의 새 예시가 통과한다 | 사람 | 함대 | 3.3 · 3.5 | 3.3 |
| CA4 | 연다 | 노드가 서버 둘을 선언하고 워크스페이스 `.mcp.json` 에 하나가 더 있을 때, 계약이 하나만 요청하면 `mcp_servers` 에 그 하나뿐이다. `.mcp.json` 의 것을 요청하면 나타난다. 없는 이름을 요청하면 하네스가 안 뜨고 단계가 그 사유로 실패한다 | 사람 | 스크래치 | 3.2 · 3.4 · 3.5 | 3.1 |
| CA5 | 싣는다 | 팩 단계가 `$OUT/pack` 에 tar 를 낸다. 다음 에이전트 단계의 `init` 줄에 팩의 스킬이 `slash_commands` 로, 팩의 서버가 `mcp_servers` 로 있다. 워크스페이스 `.claude/skills/` 가 스스로 읽히는지 같은 줄로 판정해 `decisions.md` 에 적는다 | 사람 | 스크래치 | 3.6 | 3.1 · 3.2 |
| CA6 | 한 장면 | 1절 ① ~ ⑦ 을 끝까지. 사내 MCP 의 도구가 실제로 불리고 산출물이 나오며 `record` 에 서버 이름과 팩 다이제스트가 있다 | 사람 | 사내 함대 | 전부 | 전부 |

**CA1 이 가장 앞에 있는 것이 이 팩의 핵심이다.** 실측이 찾은 것이 「거꾸로
막힌다」였으므로, 무엇을 열기 전에 무엇이 끊기는지를 먼저 잰다.

**CA6 만 사내에서 돈다.** 나머지는 가짜 stdio 서버로 스크래치에서 돈다 — `true`
는 뜨기는 하나 MCP 로 답하지 않으므로 `mcp_servers` 에 `failed` 로 나타나며, 그
이름이 목록에 **있다는 것**이 판정 재료다.

## 2.1 화면 검증

이 팩은 화면을 만들지 않는다. 현황판 카드는 새 키를 다른 속성과 같은 칩으로
보인다 — 자동이다. 제어판의 「탐지 능력」 카드도 같다. 눈으로 볼 것은 하나다.

```text
   CA2   S1   노드 카드에 mcp.<이름> 칩이 있다.  제어판 탐지 능력 카드에도 있다
```

---

# 3. 조각마다의 명령

값은 시작값이다. 유닛의 Code Generation 계획이 이것을 실제 스크립트로 굳힌다.

```text
   게이트를 돌리기 전에

     eval "$(scripts/testdb.sh)"
     export M=http://<Mediator 주소>:8080
     export T=<bootstrap 토큰>
     jq 가 필요하다.  가짜 서버는 PATH 의 true 로 족하다

   init 줄을 읽는 법 — 이 팩의 U1 이 Argv 에 --output-format stream-json --verbose
   를 더하므로 (decisions.md 6절 ⑮) 그 줄이 logs/ 에 그대로 남는다:
     runctl record <id> -o r.tar && tar -xf r.tar
     head -1 run-<id>/logs/*-<단계>.log | jq '{mcp: .mcp_servers, skills: .slash_commands}'

   runctl record 는 tar 를 stdout 에 붓는다 — 푸는 코드가 runctl 에 없으므로 -o 로
   받아 tar 로 푼다 (그 플래그는 이미 있다).  tar 안의 경로는 run-<id>/logs/ 이고
   파일 이름은 NN-<단계>.log 다

   이것이 실물을 잰다 — enode 가 실제로 지은 가짜 홈과 실제로 쓴 허용목록으로
   돈 하네스의 증언이다.  사람이 손으로 띄우는 복제본 경로는 환경(harnessEnv 가
   os.Environ() 을 안 얹는다) · 플래그(--settings · --permission-mode · --add-dir) ·
   게이트웨이 인증(apiKeyHelper)에서 실물과 갈리므로 쓰지 않는다.
   claude mcp list 도 쓰지 않는다 — strict 와 mcp-config 를 반영하지 않는다
```

```text
   CA0   decisions.md 6.1 의 열쇠 표를 돈다 — 기대값이 0 인 줄에서 0 이 아니면 빨갛다.
         뒤집기가 반쯤 내려간 채로 다음 유닛이 착수되는 것을 막는 검사다.
         그리고 앞 팩의 CP0 명령 그대로.  Mediator 라우트 수가 그대로다 —
         internal/api/*.go 전체에서 mux.HandleFunc 와 mux.Handle( 을 함께 센다.
         오늘 값은 26 이다 (api.go 의 HandleFunc 17 + Handle 3 · demo_gallery.go 6).
         api.go 한 파일만 HandleFunc 로 세면 17 이 나와 나머지 아홉을 놓친다

   CA1   개인 서버가 있는 기계에서 (claude mcp list 로 있음을 먼저 본다)
         runctl example agent > a.json  (agent.mcp 없음)
         runctl submit a.json -> 끝난 뒤 runctl record <id> 로 logs/ 를 푼다
         init 줄의 mcp_servers == []
         OAuth 노드 · 게이트웨이 노드에서 각각 한 번.  둘 다 Not logged in 이 없다

   CA2   enode.yaml 에 mcp: { probe: { command: true } } 를 적고 노드를 띄운다
         curl -H "Authorization: Bearer $T" $M/v1/nodes \
           | jq '.nodes[] | .capabilities[].attrs | {"mcp.probe", "harness.claude", harness}'
         true 를 PATH 에서 못 찾게 한 뒤 (command 를 없는 경로로) 탐지 주기를 기다린다
           -> mcp.probe 가 없다.  노드 로그에 dropped 사유

   CA3   requires 에 "mcp.probe": "1" 을 적은 계약 -> 그 노드에 배정 (GET /v1/runs/{id} 의 assigned)
         "mcp.nope": "1" -> 422 (runctl 은 stderr 에 낸다)
         agent 에 "mcp_servers": [] 같은 모르는 키 -> 400
         runctl example mcp > m.json && runctl lint m.json -> ok

   CA4   노드 선언 둘(probe · probe2) + 워크스페이스 .mcp.json 에 probe3
         agent.mcp: ["probe"]          -> init 의 mcp_servers 이름이 probe 하나
         agent.mcp: ["probe3"]         -> probe3 이 있다 (워크스페이스에서 옮겨 적힘)
         agent.mcp: ["nope"]           -> 하네스가 안 뜨고 단계 error 가
                                          "mcp server nope is not available on this node"

   CA5   팩 tar 를 만든다 — skills/hello/SKILL.md · mcp.json 에 probe4
         첫 단계 run: ["curl", "-o", "$OUT/pack", "<tar 주소>"]  (또는 git archive --remote)
         둘째 단계 agent.pack: "pack" · agent.mcp: ["probe4"] · in.from: ["fetch.pack"]
         init 의 slash_commands 에 hello · mcp_servers 에 probe4
         agent.mcp 를 빼고 같은 팩으로 한 번 더 -> mcp_servers 가 비어 있다
           (팩도 필터를 탄다.  스킬은 그대로 뜬다)
         팩의 mcp.json 에 노드가 선언한 이름(probe)을 넣어 한 번 더
           -> 단계 error 에 pack redefines node-declared mcp server probe
         워크스페이스에 .claude/skills/ws-skill/SKILL.md 를 두고 팩 없이 돌려
           ws-skill 이 slash_commands 에 있는지 본다 -> decisions.md 에 적는다

   CA6   사내 함대에서 1절 그대로.  runctl record <id> 로 푼 steps/NN-*.json 에
         harness.mcp 와 harness.pack 이 있다
```

---

# 4. 게이트가 빨간 채로 다음 유닛을 착수하면

착수하지 않는다. 빨간 이유가 그 유닛 밖(사내 MCP 서버가 죽었다 · 게이트웨이가
막혔다)이면 진행자가 그 사실을 이 회차의 `audit.md` 에 적고 조각을 **보류**로
표시한 뒤 넘어간다. **보류는 통과가 아니다.**

**눈 검증을 보류로 넘기지 않는다.** 앞 팩의 CP6 은 눈 검증이 보류로 남은 채
닫혔고, 그 결함이 사내 실측에서야 드러났다. 이 팩의 CA1 · CA4 · CA5 는 사람이
실제 하네스로 한 번은 띄워야 초록이다.
