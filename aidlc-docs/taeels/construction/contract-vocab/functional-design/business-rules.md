# U2 `contract-vocab` — 비즈니스 규칙

**거절과 문구**다. 어휘의 형식은 `domain-entities.md`, 값이 지나가는 길은
`business-logic-model.md` 다.

---

## 0. 규칙의 자리는 `Validate` 하나다

답 1=A 와 2=A 가 그 자리를 정했다. **`contract.Validate` 가 이름뿐 아니라 값의
타입도 본다.**

```text
   오늘        agentKeys 와 knownKeys 가 키 이름만 본다 (contract.go:902 · 1119).
              타입이 틀린 값은 400 을 안 받고 노드까지 가서
              json.Unmarshal 의 기본 문구로 죽는다 (agent.go:500)

   이 유닛 뒤   Validate 가 새 키 둘의 타입을 보고 Mediator 가 400 을 낸다.
              노드의 문구도 같은 문장으로 고쳐 둘째 겹으로 남긴다
```

**왜 Mediator 인가** — 스토리 US-7 이 「타입을 틀리게 적으면 **제출에서**
걸린다」이고, `decisions.md` 2절이 검증 자리를 `contract.Validate` 로 이미
적었다. 그리고 **계획이 지은 단계도 같은 검사를 받는다** (3절). 그것이 노드
쪽만 고쳤을 때 안 생기는 값이다.

**왜 노드도 남기나** — 등급의 기본은 닫히는 쪽이다 (`decisions.md` 6절 ⑨ 와
같은 결). 한 겹만 두면 그 겹을 안 지나는 경로가 생길 때 조용히 열린다.

---

## 1. 규칙 여섯

| | 규칙 | 자리 | 코드 |
|---|---|---|---|
| R1 | `agent` 에 모르는 키가 있으면 거절한다 | `Validate` | `400` |
| R2 | `agent.mcp` 는 문자열 배열이다 | `Validate` | `400` |
| R3 | `agent.mcp` 의 각 원소는 빈 문자열이 아닌 이름이다 | `Validate` | `400` |
| R4 | `agent.pack` 은 문자열이다 | `Validate` | `400` |
| R5 | `agent.pack` 이 적은 이름이 같은 단계의 `in.from` 에 있다 | `Validate` | `400` |
| R6 | `in.from` 은 문자열 배열이다 | `Validate` | `400` |

**R1 은 오늘 그대로다** — 목록에 이름 둘을 더하는 것이 이 유닛의 일이고
검증기는 이미 있다 (`decisions.md` 6절 ①). R2 ~ R6 이 새로 서는 줄이다.

**계획이 낸 계약에서는 같은 규칙이 `422` 로 보인다** — 계획은 파일로 나오고 그
파일이 스키마 검증을 받는 자리에서 먼저 걸린다 (3절).

### 1.1 문구

문구는 **영어**다 (`CONVENTIONS.md` 2.1 — 호출자와 로그가 읽는다). 사람이 다음
행동을 알 수 있게 **무엇이 틀렸는지와 어디에 적어야 하는지**를 함께 낸다.
`knownKeys` 가 이미 그 형태다.

```text
   R1   step "review": unknown field "mcpp" in agent
        (allowed: model, max_turns, max_tokens, ask, harness, mcp, pack);
        the task for the agent goes in in.prompt, and agent carries
        execution parameters only

   R2   step "review": agent.mcp must be an array of server names

   R3   step "review": agent.mcp[1] must be a non-empty server name

   R4   step "review": agent.pack must be a blob name (a string)

   R5   step "review": agent.pack names "kernel-review", but in.from does not
        carry it; add "kernel-review" to in.from so the pack is placed in $IN

   R6   step "review": in.from must be an array of artifact names
```

**R1 의 허용 목록은 손으로 안 적는다** — `strings.Join(allowed, ", ")` 가
`agentKeys` 를 그 자리에서 나열한다. 그래서 목록이 늘면 안내가 함께 는다. 답
3=A 가 새 열거 면을 안 만들고 이 성질에 기댄 근거다.

**R5 가 이름을 두 번 적는 이유** — 고칠 자리가 `in.from` 이라서다. 「없다」만
말하면 `agent.pack` 을 지우는 것도 답이 되어 보이고, 그것은 팩을 안 쓰겠다는
다른 뜻이다.

---

## 2. R5 가 닫는 실패

`agent.pack` 은 **`$IN` 에 깔린 파일의 이름**이고 그 파일은 `in.from` 이 적어야
깔린다 (`domain-entities.md` 4절).

**R5 의 범위를 실측이 좁혔다.** 첫 판은 「이름 불일치도 조용히 실패한다」로
적었는데 그것이 틀렸다 — `Validate` 에 `in.from` 의 **정적 검사가 이미 있다**
(`contract.go:1109-1126` · `ADR-023` §6.2.1). 그래서 두 경우가 갈린다.

```text
   이름을 틀리게 적었다      오늘도 제출에서 거절된다.  R5 가 아니라 있던 검사다
     in.from: ["fetch.pack"]   step "work": in.from refers to "fetch.pack",
     out:     ["pack"]         which no step produces

   in.from 을 아예 안 적었다   오늘은 통과한다.  R5 가 닫는 것이 이 하나다
     agent.pack: "pack"        제출 200 -> 노드 배정 -> $IN 에 팩이 없다
     in.from 없음              -> 계장이 팩 없이 하네스를 띄운다
                               -> 스킬도 팩의 서버도 안 뜬다.  없는 입력은
                                  값이므로 (claim.go:505-535 · ADR-058) 단계가
                                  안 죽고 종료코드 0 으로 끝날 수 있다
                               ⇒ 실패가 「없음」으로 보인다
```

**남은 하나가 가장 조용한 쪽이다.** 오타는 거절로 오고, 빠뜨림은 성공으로 온다.
`decisions.md` 6절 ⑨ 가 「빠뜨림이 닫히는 쪽으로 틀린다」로 같은 결을 적었다.

**R5 는 실행 한 번을 제출 한 번으로 바꾼다.** `Validate` 가 같은 단계 안의 두
필드를 대조하는 것은 새 종류가 아니다 — `dispatch.from` 이 이미 「그 단계가 내는
산출물을 가리켜야 한다」로 같은 대조를 하고, 바로 위의 `in.from` 정적 검사가
이미 계약 전체를 훑어 같은 종류의 오타를 막는다.

**대가를 이름으로 적는다.** `runctl submit --pack` (`decisions.md` 4절 이월)이
제출 시점에 blob 을 심는 길을 고르면, 그 경로는 `in.from` 없이도 팩이 있을 수
있다. 그때 고치는 자리는 **이 규칙 한 줄**이고, `decisions.md` 6절 ⑫ 가 이미
「계약 어휘는 안 는다」로 그 기능의 모양을 못 박아 뒀다.

---

## 3. 계획이 지은 단계도 같은 규칙을 받는다

근거가 코드에 둘 있다.

```text
   internal/contract/checkplan.go:82   빠른 훅.  제안을 얹어 Validate 를 부른다
   internal/store/expand.go:206        applyExpands.  next.Validate() 로 다시 본다
                                       (:222 · :264 에서 제안과 채택도 같은 Validate)
```

`checkplan.go` 의 주석이 그 관계를 이미 적었다 — 「훅은 빠르고 불완전하고
`applyExpands` 가 느리고 완전하다. 권위는 거기 있다」. **그래서 R2 ~ R6 은
사람이 적은 계약과 계획이 지은 계약에 같이 걸린다.** 규칙을 `Validate` 에 두는
것만으로 그렇게 되고, 새 검사 자리가 0 이다.

**그 대신 계획에게 가르쳐야 한다.** 규칙이 늘었는데 문법 안내가 안 늘면 계획은
벽을 못 보고 부딪힌다 — `grammar.go` 의 머리 주석이 실측 넷으로 그것을 적었다
(6차는 `ask` 단계에 `uses` 를 요구해 `422` 로 계획 전체가 버려졌다).
더하는 문장은 `business-logic-model.md` 3절에 있다.

---

## 4. 노드 쪽 둘째 겹 — `parseAgentParams`

```text
   오늘    json.Unmarshal 한 줄.  문구는 Go 의 기본값이다 —
           json: cannot unmarshal string into Go struct field AgentParams.mcp
           of type []string

   규칙    타입이 틀리면 R2 · R4 와 같은 문장을 낸다.
           사람이 봉인에서 읽는 문장과 제출에서 받는 문장이 같아야 한다 —
           두 벌이면 어휘를 두 번 배운다
```

**이 문구가 봉인에 남는다.** `claim.go:677-681` 이 오류를 `Result.Error` 로
보고하고 그것이 `steps/NN-*.json` 으로 들어간다 (`claim.go:784` 와 같은 자리).
그래서 문구는 **로그가 아니라 기록**이다.

**노드는 R5 를 안 본다.** 그 자리에서는 `in.from` 이 이미 파일로 풀렸고
「없는 입력은 값」이 그 위에서 돈다 (`ADR-058`). 팩이 `$IN` 에 없다는 판정과
문구는 **U5 의 것**이다 — 이 유닛은 그 이음매를 이름으로만 적는다.

---

## 5. `grammar_test.go` 가 재는 형태

그 시험은 **문법의 문장마다 「그 규칙을 어긴 계약이 실제로 거절되는가」** 를
잰다 (`mustSay` + `wantErr`). 규칙을 그 형태로 쓸 수 있게 **한 문장에 한 거절
사유**로 짓는다.

| 규칙 | 문법이 해야 하는 말 (`mustSay`) | 거절 사유에 있어야 하는 말 (`wantErr`) |
|---|---|---|
| R2 | `agent.mcp must be an array of server names` | `agent.mcp` |
| R4 | `agent.pack must be a blob name` | `agent.pack` |
| R5 | `must also list that same name in in.from` | `in.from` |

R3 과 R6 은 문법에 별 문장을 안 만든다 — R2 · R5 의 문장이 값의 모양을 이미
말하고, 문장을 쪼개면 안내가 길어져 계획이 읽는 비용만 는다. **시험은 R3 · R6
도 잰다** — `contract_test.go` 쪽이고, 문법 대조가 아니라 거절 대조다.

**시험 하나를 더 세운다** — `PlanShape` 의 `agent` 키 목록과 `agentKeys` 가
갈리지 않는 것을 잰다. 오늘 그 목록은 손으로 적힌 산문이고
(`planshape.go:45`), 갈리면 **계획에게 거짓을 가르친다**. `Grammar` 쪽에는 이미
`grammar_test.go` 가 그 장치를 갖고 있는데 `PlanShape` 에는 없다. 답 밖에서
정한 것이고 6절에 적는다.

---

## 6. 답 밖에서 더 정한 것 다섯

```text
   ① in.from 의 타입 검사 (R6)     답 2=A 의 대조가 그 값을 읽어야 하므로
                                  필요하다.  in.from 이 문자열이면 대조 자체를
                                  못 한다.  in.prompt 는 안 본다 — 7절 ②

   ② 빈 문자열 이름을 거절한다 (R3)  빈 이름은 어느 출처에도 없으므로 U4 에서
                                  「없는 서버」로 죽는다.  제출에서 잡는 쪽이
                                  같은 답을 더 이르게 준다

   ③ PlanShape 와 agentKeys 가      5절.  Grammar 에는 장치가 있고 PlanShape
      갈리지 않는 것을 시험이 잰다     에는 없었다

   ④ 팩은 하나다                    값의 타입이 문자열인 것에서 따라온다.
                                  근거는 domain-entities 4.2 — 이름 충돌의
                                  판정이 하나여야 한다

   ⑤ requires 와 agent.mcp 를       domain-entities 5절.  묶으면 워크스페이스와
      묶어 검증하지 않는다            팩에서 온 이름을 쓰는 정당한 계약이 거절된다
```

---

## 7. 한계와 넘김

```text
   ① max_tokens 가 옛 이름으로 남는다
      ADR-034 4절이 max_budget_usd 로 개명하고 옛 이름을 400 으로 막자고
      적었는데 이 팩 밖이다 (decisions.md 4절).  알려진 키 일곱에 옛 이름이
      그대로 들어간다 — 개명이 오면 그 목록과 AgentParams 가 함께 바뀐다

   ② 타입 검사가 새 키 둘에만 걸린다
      답 1=A 가 정한 범위다.  model · max_turns · max_tokens · ask · harness 는
      여전히 노드의 json.Unmarshal 에서 죽고 문구도 Go 의 기본값이다.
      비대칭을 진행자에게 넘긴다 — 같은 자리에 다섯을 더하는 것은 이 유닛의
      완료 조건 밖이고, features.md 3.5 의 범위도 새 어휘다

   ③ 서버 이름의 글자 규칙
      U3 (광고 속성 mcp.<이름>) 과 U5 (mcp.json 의 키)가 근거를 갖는다.
      이 유닛은 빈 문자열만 본다

   ④ submit --pack 이 오면 R5 를 함께 본다
      2절 끝.  decisions.md 6절 ⑫ 가 그 기능의 모양을 이미 못 박았다

   ⑤ GLOSSARY.md 는 이 유닛 밖이다 (답 7=B)
      CLAUDE.md 의 규약이 CP 와 CA 를 이름으로 지목했고 main 에도 그 파일이
      없다.  커밋 하나로 따로 올린다 — 규약 자체가 그랬다.
      푼 말은 사람에게서 와야 한다.  같은 규약이 짐작을 금지한다
```
