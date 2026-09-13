# U2 `contract-vocab` — 업무 논리 모형

값이 계약에서 하네스까지 가는 **한 줄기**와, 그 줄기에서 이 유닛이 만지는
자리다. 형식은 `domain-entities.md`, 거절은 `business-rules.md` 다.

---

## 1. 한 줄기

```text
   계약 저자 / 계획(expands)
        |
        |  steps[].agent = { "mcp": ["probe"], "pack": "kernel-review" }
        v
   contract.Validate                   R1 ~ R6.  Mediator 가 400 (계획은 422)
        |                              <- 이 유닛이 만든다
        v
   Mediator 가 단계를 실어 보낸다         enode.Step.Agent 는 json.RawMessage
        |
        v
   parseAgentParams                    AgentParams.MCP · .Pack
        |                              <- 이 유닛이 만든다 (필드와 문구)
        v
   runAgentStep
        |
        +--> resolveComponents(요청, 노드 선언, 워크스페이스, 팩)   U4
        |         |
        |         v
        |    Components -> mcp.json 직렬화 -> Instrument 가 쓴다     U1 이 지었다
        |
        +--> 팩 tar 를 $IN 에서 읽어 가짜 홈에 편다                  U5
        |
        v
   claude --strict-mcp-config --mcp-config=<경로> ...               U1 이 지었다
        |
        v
   HarnessResult.MCP · .Pack -> 봉인                                U5
```

**이 유닛은 줄기의 앞 두 칸이다.** 뒤의 셋은 U4 · U5 가 짓고, U1 이 이미 지은
칸(쓰는 자리)은 안 건드린다. 그래서 이 유닛만 병합돼도 동작이 안 바뀐다 —
어휘가 서고 값이 `AgentParams` 까지 실려 오지만 그것을 읽는 코드가 아직 0 이다.

**그것이 이 유닛이 「빌드 시점 의존의 뿌리」인 이유다** (`unit-of-work.md` 2절).
U3 · U4 · U5 는 이 어휘가 없으면 컴파일되지 않는 코드를 쓴다.

---

## 2. 만지는 자리 다섯

```text
   contract.go:902   agentKeys 에 "mcp" · "pack".  목록의 정본이다
   contract.go:1119  knownKeys 다음 줄에 값 검사 (R2 ~ R6).
                     한 함수로 모은다 — Validate 의 단계 순회 안이고,
                     오류는 첫 하나에서 돌아온다 (오늘의 다른 검사와 같다)
   grammar.go        Grammar 에 문장 (3절)
   planshape.go:45   PlanShape 의 두 줄 (4절).  행렬 밖이라 행을 더한다
   agent.go:36 · 500 AgentParams 의 필드 둘 · parseAgentParams 의 문구
   examples/mcp.json 새 예시 (5절)
```

**순서 하나가 Code Generation 에 걸린다.** `examples/mcp.json` 은
`agentKeys` 가 늘기 전에는 `Validate` 에서 `unknown field "mcp"` 로 거절된다 —
`internal/contract/example_test.go` 가 모든 예시를 `Validate` 에 태우므로
**예시를 먼저 넣으면 확정적으로 빨갛다.** 같은 커밋에 함께 들어가야 한다.

---

## 3. `Grammar` 에 더하는 문장

`grammar.go` 의 `Grammar` 상수에 절 하나를 더한다. **계획이 읽는다** — 3절이
적은 대로 계획이 지은 단계도 같은 규칙을 받으므로, 규칙만 늘리고 안내를 안
늘리면 계획이 벽을 못 보고 부딪힌다.

```text
### A step may ask for MCP servers and for a pack

    { "id":"review", "uses":"b",
      "agent": { "mcp": ["probe"], "pack": "kernel-review" },
      "in": { "prompt": "...", "from": ["kernel-review"] } }

    agent.mcp     the names of the MCP servers this step wants. Nothing is
                  opened unless it is named here
    agent.pack    the name of one blob that carries the pack tar

agent.mcp must be an array of server names, and agent.pack must be a blob name.
A step that names a pack must also list that same name in in.from: the pack is
read from $IN, and without it the harness starts with no pack at all.

The names you may put in agent.mcp are the mcp.<name> attributes printed next to
each role above. A name that no node offers fails the step when it runs.
```

**마지막 문단이 답 4=A 다.** 역할 옆의 속성은 `attrLine` 이 이미 전부 찍으므로
(head 여덟 뒤에 나머지를 이름순으로) U3 이 광고를 지으면 **코드 변경 없이**
그 이름들이 프롬프트에 보인다. `Grammar` 가 `uses` 에서 이미 같은 일을 한다 —
「적을 수 있는 역할은 이것뿐」.

**조건으로 적는다** — 「보이는 `mcp.<name>` 속성이」다. U3 이 아직 광고를 안
지었으므로 그 속성이 0 인 동안에도 문장이 거짓이 되지 않아야 한다.

**「capability 는 닫힌 어휘」 줄은 안 건드린다.** 거기 `mcp.<이름>` 을 속성의
예로 더하는 것은 광고를 짓는 유닛(U3)의 자리다 — 이 유닛이 적으면 아직 아무도
광고하지 않는 이름을 예로 드는 문장이 선다.

---

## 4. `PlanShape` 의 두 줄 — 고치지 않으면 거짓말이 된다

```text
   planshape.go:45   agent   max_turns, max_tokens, ask, model, harness — nothing else
   planshape.go:52   Any other key under agent or in is rejected: the whole plan
                     is refused with the name of the offending field
```

`agentKeys` 에 둘을 더하면 **첫 줄이 거짓이 된다** — 계획에게 「`mcp` 를 적으면
계획 전체가 거절된다」고 가르치는 문장이 남고, 그것은 `features.md` 3.5 의
「계획이 짓는 단계도 같은 문법을 쓴다」와 정반대다. 둘째 줄은 참으로 남는다.

```text
   고친다   agent   max_turns, max_tokens, ask, model, harness, mcp, pack — nothing else
   더한다   agent.mcp    names of the MCP servers this step wants
            agent.pack   the name of the blob carrying the pack tar;
                         list that name in in.from as well
```

**예시 블록의 `agent` 맵은 안 건드린다.** `PlanShape` 는 모양 하나를 실제 값으로
보이는 문서이고, 거기 `"mcp": ["probe"]` 를 넣으면 **그 서버가 있는 노드를
전제한 예시**가 된다. 목록과 설명만 늘린다.

**시험을 세운다** — `PlanShape` 의 그 목록과 `agentKeys` 가 갈리지 않는 것을
잰다 (`business-rules.md` 5절 · 6절 ③). `Grammar` 에는 `grammar_test.go` 라는
장치가 있고 `PlanShape` 에는 없었다.

---

## 5. `examples/mcp.json`

답 5=A — **MCP 만** 담는다. 붙여넣으면 도는 계약 하나다.

```json
{
  "run_id": "example-mcp",
  "work": {
    "id": { "system": "manual", "change_id": "example-mcp" },
    "system": "manual"
  },
  "requires": [
    { "as": "brain", "capability": "agent.reason", "mcp.probe": "1" }
  ],
  "steps": [
    {
      "id": "ask_the_server",
      "uses": "brain",
      "agent": { "max_turns": 5, "ask": "never", "mcp": ["probe"] },
      "in": {
        "prompt": "The probe MCP server is available in this session. List the tools it offers and write $OUT/tools.json as {\"tools\": [\"...\"]}."
      },
      "out": ["tools.json"],
      "schema": {
        "tools.json": {
          "type": "object",
          "required": ["tools"],
          "properties": {
            "tools": { "type": "array", "items": { "type": "string" } }
          }
        }
      }
    }
  ],
  "success_when": [
    { "step": "ask_the_server", "produced": ["tools.json"] }
  ]
}
```

### 5.1 예시가 정하는 것 넷

```text
   둘을 함께 보인다   requires 의 mcp.probe(매칭)와 agent.mcp(실행)를 한 계약에.
                    domain-entities 5절이 적은 「하나만 적으면 반쯤 돈다」를
                    예시가 몸으로 막는다

   이름은 probe 다    게이트가 쓰는 가짜 서버의 이름이고 (CA2 · CA3 ·
                    scene-gates.md 3절) CA3 이 runctl capabilities | grep
                    mcp.probe 로 그것을 본다.  gerrit 처럼 실물 같은 이름을
                    쓰면 그 이름을 선언한 노드가 0 이라 붙여넣어도 안 돈다

   네트워크를 안 탄다  단계가 하나이고 팩이 없다.  답 5=B 가 기각된 자리 —
                    팩 예시는 첫 단계가 자리표 주소로 curl 을 돌아야 한다

   판정은 produced 다  agent 단계이므로 exit_code 를 못 쓴다.
                    lintWarnings 와 example_test 가 모든 단계에 조건을 요구한다
```

### 5.2 실측 — 이 예시를 오늘 코드에 걸어 봤다 (2026-09-13)

임시 시험 하나로 위 JSON 을 `Contract` 로 풀고 `Validate` 에 태웠다. 본 것 셋이다.

```text
   파싱과 속성     requires 의 mcp.probe 가 Attrs 로 들어간다 —
                 map[string]string{"mcp.probe":"1"}.  코드 변경 0 이 참이다

   유일한 거절      step "ask_the_server": unknown field "mcp" in agent
                 (allowed: model, max_turns, max_tokens, ask, harness); ...
                 agentKeys 가 늘기 전의 유일한 불만이다.  다른 결함이 0 이다

   agent.mcp 를 뺀 같은 계약   통과한다.  그래서 이 예시는 목록이 는 그 커밋에서
                 초록이 된다 — 2절의 순서 제약이 실측으로 섰다
```

**팩 예시는 U5 로 넘긴다** — `examples/pack.json` 이다. `agent.pack` 은 팩 단계와
`in.from` 이 함께 있어야 뜻이 서고 그 둘이 U5 의 것이다. 이 유닛은 어휘와
R5 만 세운다.

---

## 6. 이음매 — 뒤의 유닛이 어디서 집나

```text
   U3 advert    광고 속성 mcp.<이름>.  이 유닛의 값을 안 읽는다 —
                requires 쪽 어휘이고 코드 변경이 0 이다 (domain-entities 5절).
                다만 3절 마지막 문단이 U3 의 광고를 전제로 선다

   U4 sources   AgentParams.MCP 를 resolveComponents 의 첫 인자로 읽는다.
                빈 배열과 부재를 같게 취급한다 (domain-entities 3절).
                없는 이름의 문구는 U4 의 것 —
                mcp server <이름> is not available on this node

   U5 pack      AgentParams.Pack 으로 $IN 의 파일을 찾는다.
                팩이 $IN 에 없을 때의 문구와 등급은 U5 의 것 (R5 가 제출에서
                거르지만 노드가 그 경로를 안 지나는 계약도 있을 수 있다).
                HarnessResult 의 MCP · Pack 직렬화도 U5
```

---

## 7. CA3 의 앞 절반을 어떻게 재나

`unit-of-work-story-map.md` 2.1 이 이 유닛에 두 줄을 준다.

```text
   agent 에 모르는 키 -> 400                    R1.  목록이 일곱이 된 뒤에도
                                              여덟째 이름은 여전히 400 이다
   runctl example mcp && runctl lint -> ok      5절의 예시.  lint 는 경고 0
```

**여기에 답 1=A 와 2=A 가 줄 둘을 더한다** — 게이트 문서의 글자는 아니지만 이
유닛의 완료 조건이다.

```text
   agent.mcp 를 문자열로 적은 계약 -> 400       R2
   agent.pack 을 적고 in.from 을 빠뜨린 계약 -> 400   R5
```

**CA3 의 뒤 두 줄은 U3 이 닫는다** (`requires` 의 `mcp.<이름>` 이 그 노드에만
가고, 없는 키는 후보 0 으로 `422`). U2 뒤에 절반만 재고 넘어가는 것이 게이트를
빨간 채로 넘기는 것이 아니다 — 같은 문서 2.1 이 그 값을 적었다.

---

## 8. 파장 — 이 유닛 밖으로 가는 것

```text
   unit-of-work-file-matrix.md   internal/contract/planshape.go 에 행을 더한다
                                 (그 문서 6절 ② 의 절차).  U2 만 만진다

   decisions.md 6절               실측 행 하나 — 답 6=A.  셋을 한 행에 적는다
                                 ① runctl schema steps 는 agent 의 하위 키를
                                   안 낸다 (Step.Agent 가 map 이다).  features.md
                                   3.5 의 「저절로 는다」가 거짓이고, 답 3=A 로
                                   열거 면을 안 만든다
                                 ② blob 이름 공간은 Run 하나에 평평하다.
                                   in.from 의 점은 이름의 글자다
                                 ③ 그래서 CA5 의 명령이 안 돈다 — 오늘도 이미
                                   제출에서 거절된다 (in.from 의 정적 검사가
                                   있다).  거절 문구가 팩을 안 가리킨다

   scene-gates.md CA5            답 6=A.  첫 단계가 $OUT/pack 을 내므로
                                 out: ["pack"] · in.from: ["pack"] 이다.
                                 고치지 않으면 그 게이트를 시작조차 못 한다 —
                                 오늘의 거절 문구는
                                 in.from refers to "fetch.pack", which no step
                                 produces 이고, 게이트를 돌리는 사람이 그것을
                                 팩 코드의 결함으로 읽는다

   팩 문서는 이 단계의 승인 뒤에 싣는다   U1 이 ⑳ 에서 같은 순서를 밟았다.
                                 요구 팩은 이 유닛의 산출물이 아니다

   GLOSSARY.md                   답 7=B — U2 밖의 커밋 하나.  푼 말이 아직 없다
                                 (business-rules.md 7절 ⑤)
```
