# U2 `contract-vocab` — Code Generation 요약

계획은 `construction/plans/contract-vocab-code-generation-plan.md` 이고 설계는
`construction/contract-vocab/functional-design/` 의 셋이다. 규칙 번호 R1 ~ R6 은
그 `business-rules.md` 1절이다.

---

## 1. 무엇이 들어갔나

| 파일 | 무엇 |
|---|---|
| `internal/contract/contract.go` | `agentKeys` 에 `mcp` · `pack` · 새 함수 `agentValues` (R2 ~ R6) · `Validate` 의 단계 순회에 한 줄 |
| `internal/contract/grammar.go` | `Grammar` 에 절 하나 — 「A step may ask for MCP servers and for a pack」 |
| `internal/contract/planshape.go` | `agent` 키 목록에 둘 · 설명 두 줄 |
| `internal/contract/examples/mcp.json` | 새 예시 (MCP 만) |
| `internal/enode/agent.go` | `AgentParams` 에 `MCP` · `Pack` · 새 함수 `checkComponentTypes` |
| `internal/contract/grammar_test.go` | `bad` 셋 (R2 · R4 · R5) + `PlanShape` 낡음 방지 하나 |
| `internal/contract/contract_test.go` | `TestValidate_agentComponents` (R3 · R6 · 통과 경로 · 빈 배열) |
| `internal/enode/agent_test.go` | `parseAgentParams` 의 문구와 왕복 둘 |

**새 시험 파일이 0 이다.** 기존 셋에 붙었다 — 행렬 3절이 예고한 바로 그 셋이다.
`PlanShape` 의 시험은 `grammar_test.go` 에 뒀다 (같은 낡음 방지 장치의 자리).

**`cmd/runctl` 의 소스 diff 가 0 이다** (답 3=A). 행렬 3절의 값이 그대로 남았다.

---

## 2. CA0 — 전부 초록

```text
   go test ./... -count=1                 exit 0 · 패키지 18 초록 · 실패 0
   커버리지 (표준 명령 · 패키지별 80%)      미달 0 · 전체 6795/7804 = 87.1%
                                          internal/contract 526/589 = 89.3%
                                          internal/enode 1702/1988 = 85.6%
   go vet ./...                           exit 0
   gofmt -l cmd internal                  0 줄
   장식 문자 (U+2605 전수 grep)             0 히트
   go run ./scripts/glyphscan.go           104 파일 · 문자열의 장식 문자 0
   크로스 빌드                             windows/amd64 · linux/arm7 · darwin/arm64
   심볼 상한                               enodectl crypto/tls=1 · net/http=6 (상한 10 · 50)
   Mediator 라우트                          26 (안 늘었다)
   워킹트리 청결                            커버리지 실행이 바꾼
                                          cmd/enodectl/probe.lock 을 되돌렸다
```

**`internal/contract` 가 89.3% 다.** FD 5절이 「단독으로 재면 79.5% 라 하한에
가장 가까운 자리」로 적었는데, 표준 명령(`-coverpkg=./...`)은 다른 패키지의
시험이 덮은 문장을 함께 세므로 값이 다르다. 새 코드가 들어간 뒤에도 여유가
아홉 포인트다.

---

## 3. CA3 의 앞 절반 — 실측

게이트 문서의 두 줄과, 답 1=A · 2=A 가 더한 두 줄을 `runctl lint` 로 쟀다
(그 명령이 `Validate` 를 부른다).

```text
   runctl example            목록에 mcp 가 있다.  여섯이 되었다
   runctl example mcp        exit 0 · 그대로 붙여넣을 수 있는 JSON
   runctl lint m.json        ok  (1 steps, 1 conditions) — 경고 0

   모르는 agent 키 (mcpp)     step "ask_the_server": unknown field "mcpp" in agent
                            (allowed: model, max_turns, max_tokens, ask, harness,
                            mcp, pack); the task for the agent goes in in.prompt …
                            ⇒ 허용 목록이 일곱으로 함께 늘었다.  답 3=A 가
                              기댄 성질이 실물로 섰다

   agent.mcp: "probe"        step "ask_the_server": agent.mcp must be an array of
                            server names                                    (R2)

   agent.pack 만 적었다       step "ask_the_server": agent.pack names "kernel-review",
                            but in.from does not carry it; add "kernel-review" to
                            in.from so the pack is placed in $IN             (R5)
```

**CA3 의 뒤 두 줄은 U3 이 닫는다** (`requires` 의 `mcp.<이름>` 이 그 노드에만
가고, 없는 키는 후보 0 으로 `422`).

**`runctl schema steps` 도 실측했다** — `agent  map[string]any  optional` 한 줄
그대로이고 하위 키가 안 나온다. 팩의 「저절로 는다」가 거짓임을 출력으로 확인했고
`decisions.md` 6절 ㉑ 에 적었다.

---

## 4. 초록이 곧 시험이 문다는 뜻은 아니라서 — 변이 다섯

```text
   ① agentKeys 를 옛 다섯으로 되돌린다      빨강 셋 (TestValidate_agentComponents ·
                                         TestExamples_ParseAndValidate ·
                                         TestGrammar_WhatItForbidsIsActuallyRejected)
   ② Validate 에서 agentValues 호출을 뺀다   빨강 — 문법 셋과 값 검사 표가 함께
   ③ PlanShape 의 목록에서 둘을 뺀다         빨강 — "agent key \"mcp\" is missing
                                         from PlanShape"
   ④ parseAgentParams 에서 타입 검사를 뺀다   빨강 — Go 의 기본 문구가 나와
                                         원하는 문장과 안 맞는다
   ⑤ Grammar 의 절을 지운다                 빨강 — "the grammar lacks …" 셋
```

**다섯이 다 물었다.** U1 에서 구멍 하나(배선을 안 재는 시험)를 찾은 것과 달리
이번에는 0 이다. 이유는 규칙이 전부 순수 함수 하나를 지나기 때문이다 —
배선이 한 자리(`Validate` 의 순회)이고 변이 ② 가 그 자리를 직접 잰다.

---

## 5. 실측이 앞 판의 주장 하나를 뒤집었다

**FD 가 「CA5 의 이름 불일치는 조용한 실패다」로 적었고 그것이 틀렸다.**
`Validate` 에 `in.from` 의 정적 검사가 **이미 있다**
(`contract.go:1109-1126` · `ADR-023` §6.2.1). CA5 의 계약을 그대로 넣어 쟀다.

```text
   CA5 를 그대로            step "work": in.from refers to "fetch.pack",
                          which no step produces          <- 오늘도 제출에서 거절
   in.from 을 아예 안 적으면  (R5 앞에는) 통과했다.  R5 가 닫는 것이 이 하나다
```

**게이트가 안 도는 것은 그대로 참이고 이유가 다르다.** 그 거절 문구는 팩을 안
가리키므로 게이트를 돌리는 사람이 팩 코드의 결함으로 읽는다 — `scene-gates.md`
CA5 의 이름을 고친 근거가 약해진 것이 아니라 **더 구체적이 되었다.**

**R5 의 값도 좁아졌지만 남았다.** 오타는 있던 검사가 막고, 남은 것은 빠뜨림이다.
오타는 거절로 오고 빠뜨림은 성공으로 온다 — `decisions.md` 6절 ⑨ 의
「빠뜨림이 닫히는 쪽으로 틀린다」와 같은 결이다.

고친 자리 넷 — FD 계획 1.2 · `business-rules.md` 2절 ·
`business-logic-model.md` 8절 · `domain-entities.md` 4.1, 그리고
`contract.go` 의 `agentValues` 주석.

---

## 6. 이 유닛이 회차 밖으로 낸 것

```text
   decisions.md 6절        실측 행 ㉑ 을 실었다 — 셋을 한 행에 (열거 면 ·
                          blob 이름 공간 · CA5).  6절 제목의 날짜에 09-13 을 더했다
   scene-gates.md CA5      첫 단계에 out: ["pack"] 을 명시하고 둘째 단계의
                          in.from 을 ["pack"] 으로 고쳤다.  왜인지도 그 자리에 적었다
   파일 행렬               planshape.go 행을 1절에 더했다 (그 문서 6절 ② 의 절차).
                          만지는 파일 15 -> 16 · 4절 제목이 「둘」에서 「셋」으로
   짝 팩과의 접점           0 이다.  이 유닛이 만진 파일에 runner.go 도 claude.go 도 없다
```

---

## 7. 진행자에게 넘기는 것 다섯

```text
   ① agent 의 나머지 다섯 키는 타입 검사가 여전히 노드뿐이다
      답 1=A 가 새 키 둘로 범위를 정했다.  model · max_turns · max_tokens ·
      ask · harness 는 Go 의 기본 문구로 노드에서 죽는다.  같은 자리에 다섯을
      더하는 것은 열 줄이고, 안 한 이유는 범위이지 비용이 아니다

   ② 회차 문서 루트의 두 값이 낡았다
      aidlc-docs/v3-run-harness-components/aidlc-state.md 가 decisions.md 의
      행을 「열아홉」으로 적는데 U1 이 ⑳ 을, 이 유닛이 ㉑ 을 실어 스물하나다.
      같은 파일의 「설계 요약 표에 없던 파일 둘」도 셋이 되었다.
      그 루트는 회차 진행자의 것이라 이 유닛이 안 고쳤다 (CONVENTIONS 3.4)

   ③ GLOSSARY.md 가 아직 없다 (답 7=B)
      CP 와 CA 의 푼 말이 저장소에 0 이고 규약이 짐작을 금지한다.
      커밋 하나로 따로 올린다 — 규약 자체가 그랬다 (76e2201)

   ④ features.md 3.5 의 문장 하나가 거짓으로 남아 있다
      「runctl schema steps 가 새 키를 낸다 — 구조체에서 뽑으므로 저절로 는다」.
      decisions.md 6절 ㉑ 이 뒤집었다고 적었지만 features.md 의 그 줄 자체는
      안 고쳤다 — 1 ~ 5절의 번호를 다른 문서가 참조하는 것과 같은 이유로
      팩의 본문을 이 유닛이 다시 쓰지 않는다.  꼬리표를 달지는 진행자의 것이다

   ⑤ U3 이 딛는 문장 하나가 먼저 섰다
      Grammar 의 「역할 옆에 보이는 mcp.<이름> 속성이 agent.mcp 에 적을 수 있는
      이름이다」가 U3 의 광고를 전제한다.  조건으로 적어 속성이 0 인 동안에도
      거짓이 안 되게 했지만, U3 이 광고를 지으면 그 문장을 한 번 다시 읽어야 한다
```
