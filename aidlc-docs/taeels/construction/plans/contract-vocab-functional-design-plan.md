# U2 `contract-vocab` — Functional Design 계획

**유닛** `contract-vocab` · **브랜치** `unit/contract-vocab` · **담당** taeels ·
**닫는 게이트** CA3 의 앞 절반 (+ CA0) · **선행** 없음

정본 입력은 `aidlc-docs/v3-run-harness-components/inception/application-design/`
의 넷과 `requirements/harness-components/` 팩이다. 이 계획은 그 위에서
**계약 어휘의 규칙과 형식**만 짓는다 — 코드는 다음 단계다.

---

## 0. 이 단계가 닫는 것과 안 닫는 것

```text
   닫는다    agent.mcp · agent.pack 의 값 형식과 그 형식을 어겼을 때의 자리
             알려진 키 목록이 일곱이 되는 자리와 그 거절 문구
             계획에게 가르치는 문장 — Grammar 와 PlanShape 의 어느 줄이 는가
             examples/mcp.json 이 무엇을 담나
             AgentParams 의 필드 둘과 parseAgentParams 의 오류 문구
             agent.pack 이 가리키는 이름이 무엇의 이름인가 (blob · $IN 의 파일)

   안 닫는다  허용목록 합치기와 출처 우선순위     U4 의 resolveComponents
             팩 tar 를 펴는 규약과 상한          U5 의 FD
             enode.yaml 의 mcp: 절 필드          U3 의 FD
             mcp.<이름> 광고와 매칭              U3.  CA3 의 뒤 절반
             HarnessResult 의 MCP · Pack 직렬화  U5
             시험 목록과 파일별 diff              이 유닛의 Code Generation
```

---

## 1. 실측 — 코드가 지금 어떻게 생겼나 (2026-09-13)

FD 를 짓기 전에 이 유닛이 만지는 자리를 실제로 읽었다. **팩의 문장 둘이 코드와
갈렸고**, **행렬에 없는 파일 하나가 이 유닛의 것이었다.**

### 1.1 검증이 있는 자리와 없는 자리

| 자리 | 오늘 | 무엇을 보나 |
|---|---|---|
| `contract.go:902` `agentKeys` | `model` · `max_turns` · `max_tokens` · `ask` · `harness` | **키 이름만** |
| `contract.go:1119` `knownKeys` | `Validate` 안에서 거절. Mediator 가 `400` | 이름이 목록에 있나 |
| `claim.go:30` `enode.Step.Agent` | `json.RawMessage` | 노드까지 원문으로 온다 |
| `agent.go:500` `parseAgentParams` | `json.Unmarshal` 한 줄 | **타입**. 문구는 Go 의 기본값 |
| `claim.go:677` 호출 자리 | 오류를 `Result.Error` 로 보고 | 그 문구가 봉인에 남는다 |

**타입을 보는 자리가 노드에만 있다.** `decisions.md` 2절이 「검증 자리는
`contract.Validate` — Mediator 와 enode 가 같은 패키지를 본다」로 적었는데, 그
문장이 참인 것은 **이름**이고 **값의 타입은 아니다**. `agent.mcp` 를 문자열로
적은 계약은 `400` 을 안 받고 제출되어 노드까지 가서
`json: cannot unmarshal string into Go struct field` 로 죽는다. 스토리 US-7 은
「타입을 틀리게 적으면 **제출에서** 걸린다」이고 `unit-of-work.md` 2절의 완료
조건은 「`parseAgentParams` 가 그것을 문구로 낸다」다 — **둘이 같은 자리를 안
가리킨다.** 질문 1 이 그 자리를 정한다.

### 1.2 팩의 문장 둘이 코드와 갈린다

```text
   ① runctl schema steps 가 새 키를 낸다 — 구조체에서 뽑으므로 저절로 는다
      (features.md 3.5)

      거짓이다.  cmd/runctl/shape.go:172 printFields 는 reflect 로 Step 의
      최상위 필드만 찍고, Step.Agent 는 map[string]interface{} 다
      (contract.go:447).  그래서 출력은 언제나 한 줄이다 —
      agent  map[string]any  optional.  하위 키는 오늘도 안 나오고
      mcp · pack 을 더해도 안 나온다.  cmd/runctl 은 AgentParams 도
      agentKeys 도 import 하지 않는다 (grep 0)

   ② in.from: ["<단계>.<blob 이름>"]  (features.md 3.5 · ADR-034 2.2 의 prep.pack)

      절반만 참이다.  blob 이름 공간은 Run 하나에 평평하다 —
      PUT /v1/runs/{run}/steps/{seq}/blob/{name} 로 올리고
      GET /v1/runs/{run}/blob/{name} 이 그 이름의 최신을 준다 (api.go:107-108).
      claim.go:520 은 그 이름을 그대로 $IN 의 파일 이름으로 쓴다.
      그래서 점은 이름의 한 글자이고 단계 참조가 아니다 — prep.pack 이 도는 것은
      앞 단계가 out: ["prep.pack"] 이라고 적었을 때다.
      examples/multi.json 이 이미 평평한 이름(lines.txt)으로 돈다
```

②가 게이트 명령 하나를 깬다. `scene-gates.md` 3절 CA5 는 첫 단계를
`run: ["curl", "-o", "$OUT/pack", ...]` 로 적고 둘째 단계를
`in.from: ["fetch.pack"]` 로 적는다. 앞이 내는 blob 이름은 `pack` 이므로
`fetch.pack` 은 이 Run 에 없다.

**이 단락의 첫 판은 「단계가 죽지 않고 팩 없이 돈다」였고 그것이 틀렸다.**
Code Generation 이 실물로 재서 고쳤다 — `Validate` 에 `in.from` 의 **정적 검사가
이미 있다** (`contract.go:1109-1126` · `ADR-023` §6.2.1). CA5 의 계약을 그대로
넣으면 오늘도 제출에서 거절된다.

```text
   step "work": in.from refers to "fetch.pack", which no step produces
```

**게이트가 안 도는 것은 그대로 참이다** — 다만 이유가 침묵이 아니라 이 거절이고,
그 문구는 팩을 안 가리킨다. 게이트를 돌리는 사람은 「이 팩의 코드가 덜 됐나」가
아니라 「이름을 잘못 적었나」를 먼저 물어야 한다. 고치는 값은 같다 —
`out: ["pack"]` · `in.from: ["pack"]`. **그 게이트는 U5 의 것이지만 어휘의 주인은
이 유닛이다** — 질문 6 이 어디에 적을지 정한다.

### 1.3 행렬에 없는 파일 하나 — `internal/contract/planshape.go`

`PlanShape` 는 계획에게 **필드의 모양**을 보여주는 예시 상수다 (`ADR-057`).
거기 두 줄이 지금 목록을 못으로 박고 있다.

```text
   planshape.go:45   agent        max_turns, max_tokens, ask, model, harness — nothing else
   planshape.go:52   Any other key under agent or in is rejected: the whole plan is refused
```

`agentKeys` 에 둘을 더하면 **첫 줄이 거짓말이 된다** — 계획을 짓는 쪽에게
「`mcp` 를 적으면 계획 전체가 거절된다」고 가르치는 문장이 남는다.
`features.md` 3.5 는 반대로 「계획이 짓는 단계도 같은 문법을 쓴다」다.
`components.md` 1.6 은 `Grammar` 만 적었고 `planshape.go` 는
`unit-of-work-file-matrix.md` 1절의 행렬에 **행이 없다.** 이 유닛이 만진다 —
행렬 6절 ② 의 절차대로 행을 더한다.

### 1.4 안 만져도 되는 것을 확인했다

```text
   requires[].mcp.<이름>   Require 의 미지 키는 Attrs 로 들어간다 (커스텀
                          언마샬러).  코드 변경 0 — 팩의 「이미 있는 문법」이 참이다
   runctl lint            lintWarnings (shape.go:78) 는 success_when 덮임만 본다.
                          함대를 안 본다 — 새 예시가 안 뜬 서버를 요구해도 경고 0
   attrLine (agent.go)    역할 옆에 붙는 속성은 head 여덟 뒤에 나머지를 이름순으로
                          전부 붙인다.  U3 이 mcp.<이름> 을 광고하면 계획의
                          프롬프트에 저절로 보인다 — 코드 변경 0.  질문 4 가 쓸 값
```

---

## 2. 물음 일곱

답을 `[Answer]:` 뒤에 적는다. 권장이 있는 물음은 권장을 **A** 에 둔다.

### Question 1

**타입이 틀린 `agent.mcp` · `agent.pack` 을 어디서 잡나.** 1.1 이 오늘 그 자리가
노드뿐임을 보였고, US-7 과 완료 조건이 서로 다른 자리를 가리킨다.

A) **`contract.Validate` 가 타입도 본다 — 제출에서 `400` 이고, 노드의 문구는
둘째 겹으로 남긴다.** `agent.mcp` 는 문자열 배열 · `agent.pack` 은 문자열임을
Mediator 가 보고 거절한다. US-7 의 글자가 그대로 서고, 계획이 지은 단계도 같은
검사를 받는다 (`applyExpands` 가 `Validate` 를 다시 돈다). `parseAgentParams` 의
문구도 함께 고쳐 두 겹으로 만든다 — 검증 없이 노드에 닿는 경로(직접 claim)가
남아 있고, 등급의 기본은 닫히는 쪽이다 (`decisions.md` 6절 ⑨ 와 같은 결)

B) **노드만 본다.** `agentKeys` 는 이름만 보는 오늘 그대로 두고
`parseAgentParams` 가 문구를 낸다 — `agent.mcp must be an array of server names` ·
`agent.pack must be a blob name`. `unit-of-work.md` 2절의 완료 조건 글자 그대로다.
`Validate` 의 주석(「계약이 문법적으로 성립하는지만 본다」)을 안 넓힌다.
대가는 틀린 계약이 노드와 임대를 한 번 쓰고 죽는 것이다

C) **`runctl lint` 가 제출 전에 경고한다 + 노드가 문구를 낸다.** Mediator 는 안
바뀐다. 대가는 lint 를 안 돌리는 경로(오케스트레이터 · API 직접 제출)에서 아무도
안 잡는 것이다. `cmd/runctl` 의 소스 diff 0 도 깨진다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2

**`agent.pack` 이 가리킨 이름이 그 단계의 `in.from` 에 없으면 제출에서 잡나.**
팩은 `$IN` 에 깔린 파일을 계장이 읽으므로 (`ADR-034` 2.2), `in.from` 에 그 이름이
없으면 팩이 없는 채로 돈다.

A) **`400` 으로 거절한다.** `Validate` 가 같은 단계 안에서 대조한다 —
`agent.pack` 의 이름이 `in.from` 에 없으면 거절. 이미 `dispatch.from` 이
「그 단계가 내는 산출물을 가리켜야 한다」로 같은 종류의 대조를 한다.
1.2 ② 가 보인 실수(이름 불일치)가 가장 흔한 실수이고, 그것이 실행 한 번을
버리는 대신 제출에서 문장으로 온다. 대가는 `runctl submit --pack`(4절 이월)이
`in.from` 없이 blob 을 심는 길을 고르면 이 줄을 함께 고쳐야 하는 것이다 —
한 곳이고, 그 결정이 올 때 그 기능이 고친다

B) **안 잡는다.** `Validate` 는 형식만 보고, 팩이 `$IN` 에 없다는 것은 실행 시에
U5 가 문장으로 낸다. `agent.pack` 과 blob 의 관계를 계약 검증에 못 박지 않아
`submit --pack` 이 열릴 자리가 그대로 남는다 (`decisions.md` 6절 ⑫)

C) **경고만 낸다 (`runctl lint`).** 거절은 아니고, 제출 전에 사람이 본다.
오케스트레이터 경로에서는 아무도 안 본다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3

**`runctl schema` 가 `agent` 의 하위 키를 낼 자리를 만드나.** 1.2 ① 이
「구조체에서 뽑으므로 저절로 는다」가 거짓임을 보였다. 열거 면이 없으면 계약
작성자는 `runctl example` 과 거절 문구로만 어휘를 안다.

A) **안 만든다.** 팩의 그 문장을 `decisions.md` 6절에 행으로 고쳐 적고, 어휘를
읽는 면 셋에 기댄다 — `runctl example mcp` (붙여넣으면 도는 예시) ·
`knownKeys` 의 거절 문구(허용 목록을 그 자리에서 나열한다) ·
`Grammar` 와 `PlanShape`. `cmd/runctl` 의 소스 diff 0 이라는 행렬의 값이 남는다

B) **`runctl schema agent` 를 더한다.** `agentKeys` 를 내보내(`AgentKeys()`)
그것으로 찍는다. 손으로 적는 목록이 아니므로 갈릴 수 없다. 대가는
`cmd/runctl` 의 diff 가 0 이 아니게 되는 것이고 (행렬 1절 · 3절의 값이 바뀐다)
`agentKeys` 가 패키지 밖으로 나가는 것이다

C) **`runctl schema steps` 의 `agent` 줄 아래에 하위 키를 들여써서 찍는다.**
새 섹션을 안 만들고 있는 출력만 는다. 대가는 B 와 같다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4

**계획에게 어디까지 가르치나.** `Grammar` 와 `PlanShape` 에 `mcp` · `pack` 을
더하는 것은 1.3 이 정한 필수다. 그 위에 **이름을 어디서 얻는지**를 적을지가
남는다 — 1.4 가 U3 뒤에 역할 옆에 `mcp.<이름>` 이 저절로 보인다는 것을 쟀다.

A) **한 줄 더한다.** 「역할 이름 옆에 보이는 `mcp.<이름>` 속성이 그 노드에서
`agent.mcp` 에 적을 수 있는 이름이다」. 코드 변경은 0 이고 (`attrLine` 이 이미
찍는다) 계획이 없는 이름을 지어내는 경로가 닫힌다 — `Grammar` 가 `uses` 에서
이미 같은 일을 한다(「적을 수 있는 역할은 이것뿐」)

B) **안 더한다.** 어휘만 가르치고 이름의 출처는 안 적는다. 없는 이름을 적으면
U4 가 실행 시에 `mcp server <이름> is not available on this node` 로 낸다.
근거 — U3 이 광고를 아직 안 지었으므로 이 유닛이 그 줄을 적으면 없는 것을
가리키는 문장이 잠깐 선다

C) **U3 의 FD 로 넘긴다.** 문장의 자리는 이 유닛이 만들고 (어휘 줄까지) 내용은
광고를 짓는 유닛이 채운다. `grammar_test.go` 가 재는 방식과 어긋나지는 않는다 —
그 시험은 금지를 재고 이 문장은 안내다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5

**`examples/mcp.json` 이 무엇을 담나.** 행렬은 이 유닛에 새 예시 **하나**를 준다.

A) **MCP 만.** `requires[].mcp.<이름>: "1"` 로 고르고 `agent.mcp: ["<이름>"]` 로
물리는 단계 하나 + `success_when`. 붙여넣으면 그 서버를 선언한 노드에서 그대로
돌고 네트워크를 안 탄다. 팩 예시는 U5 가 `examples/pack.json` 으로 낸다 —
`agent.pack` 은 팩 단계와 `in.from` 이 함께 있어야 뜻이 서고 그 둘이 U5 의 것이다.
이 유닛은 그 넘김을 기록한다

B) **MCP 와 팩을 한 예시에.** CA5 의 모양이 그대로 복사되어 게이트를 돌리는
사람이 베낀다. 대가 둘 — 첫 단계가 자리표 주소로 `curl` 을 돌아서 「붙여넣으면
돈다」가 깨지고, U5 의 어휘가 U2 의 파일에 먼저 들어온다

C) **MCP 만 담되 팩 줄을 주석 아닌 형태로 함께 보인다** (예시 안의 `in.prompt`
문장으로). JSON 에 주석이 없으므로 어디에 적든 본문이 된다. 계약 예시의 산문이
문법 안내를 겸하는 것은 이 저장소에 선례가 없다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6

**팩과 게이트의 갈린 문장 셋을 어디에 적나.** 1.2 가 둘(`runctl schema steps` ·
`in.from` 의 점)을, 1.2 끝이 하나(CA5 의 `pack` 대 `fetch.pack`)를 찾았다.
`requirements/` 는 요구 팩이고 이 회차의 유닛이 그 6절에 실측 행을 더한 선례가
있다 (U1 이 ⑳ 을 실었다).

A) **`decisions.md` 6절에 행을 더한다 — 셋을 한 행으로.** 「계약 어휘의 열거 면과
blob 이름 공간」 한 행에 셋을 적고, `scene-gates.md` CA5 의 명령도 그 행이
가리키는 대로 고친다(`out: ["pack"]` · `in.from: ["pack"]`). 어휘의 주인이 이
유닛이고, 미루면 게이트를 돌리는 사람이 빨간 이유를 팩 코드에서 찾는다

B) **`decisions.md` 6절에 행만 더하고 `scene-gates.md` 는 안 고친다.** 게이트
문서는 그 게이트를 지는 유닛(U5)이 고친다. 대가는 U5 까지 그 명령이 틀린 채
남는 것이다

C) **이 유닛은 자기 산출물에만 적고 팩은 안 만진다.** 진행자가 팩 개정을 진다
(`code-summary.md` 6절의 넘김 목록과 같은 자리)

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 7

**`GLOSSARY.md` 가 아직 없다.** `CLAUDE.md` 의 규약이 「`CP` 와 `CA` 가 지금 그
자리다 — 무엇의 약자인지 적은 줄이 저장소에 0 이다」로 그 둘을 이름으로 지목하고,
자리를 루트 `GLOSSARY.md` 로 박았다. 이 단계의 산출물이 `CA0` · `CA3` 를 계속
쓴다. **푼 말을 짐작해서 적지 않는다** — 같은 규약이 그것을 금지한다.

A) **지금 푼 말을 답으로 주면 이 단계의 커밋이 `GLOSSARY.md` 를 만든다.**
`[Answer]:` 뒤에 `CP` 와 `CA` 의 푼 말을 적는다. 적을 것은 셋 — 푼 말 · 한 줄 뜻 ·
그 어휘를 지는 문서의 주소(`requirements/scene-gates.md` ·
`requirements/harness-components/scene-gates.md`)

B) **U2 밖에서 따로 돈다.** 커밋 하나로 `main` 에 직접 올린다 — 규약 자체가
그랬다(`76e2201`). 이 유닛의 커밋에는 안 싣는다 (`CONVENTIONS.md` 3.4 의
「한 커밋은 그 단계의 산출물」)

C) **미룬다.** 다음에 새 글자를 들여오는 커밋이 함께 만든다. 대가는 규약이
지목한 빚이 그대로 남는 것이다

D) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## 3. 답을 받은 뒤 낼 산출물

`aidlc-docs/taeels/construction/contract-vocab/functional-design/` 에 셋이다.

- [ ] `domain-entities.md` — 어휘의 형식
  - [ ] `agent` 맵의 알려진 키 일곱과 각 값의 타입. `mcp` 는 문자열 배열 ·
        `pack` 은 문자열
  - [ ] `AgentParams` 의 필드 둘과 JSON 태그. 비었을 때의 뜻 (안 적으면 팩 없이
        돈다 · 오늘 그대로)
  - [ ] `agent.pack` 이 가리키는 것이 무엇의 이름인가 — blob 이름이고 그것이
        `$IN` 의 파일 이름이다. 점은 이름의 글자다 (1.2 ②)
  - [ ] 빈 배열과 부재를 가른다 — `agent.mcp: []` 와 키 없음이 같은가
- [ ] `business-rules.md` — 거절과 문구
  - [ ] 모르는 키는 `400` (오늘 그대로). 허용 목록을 문구가 나열한다
  - [ ] 타입 위반의 자리와 문구 (질문 1 의 답)
  - [ ] `agent.pack` 과 `in.from` 의 대조 (질문 2 의 답)
  - [ ] 계획이 지은 단계에도 같은 규칙이 걸린다 — `applyExpands` 가 `Validate`
        를 다시 돈다는 근거를 코드 자리로 적는다
  - [ ] `grammar_test.go` 가 이 규칙마다 「어긴 계약이 실제로 거절되는가」를 잴 수
        있는 형태로 적는다 — 문장 하나에 거절 사유 하나
- [ ] `business-logic-model.md` — 값이 지나가는 길
  - [ ] 계약 -> `Validate` -> `Job` -> `parseAgentParams` -> `AgentParams` 까지의
        한 줄기와, 그 줄기에서 이 유닛이 만지는 자리 넷
  - [ ] `Grammar` · `PlanShape` 에 더하는 문장과 지우는 문장 (1.3 의 거짓말 줄)
  - [ ] `examples/mcp.json` 의 내용 (질문 5 의 답)
  - [ ] U3 · U4 · U5 가 이 어휘를 어디서 집는가 — 이음매를 이름으로

---

## 4. 이 단계가 안 만드는 것

```text
   코드            다음 단계다.  이 단계는 규칙과 형식만 낸다
   시험 목록        Code Generation 계획이 낸다
   NFR             회차 실행 계획이 SKIP 으로 닫았다 (셋 다)
   인프라           배포 변경 0
```

---

## 5. 파장 — 이 유닛 밖으로 가는 것

```text
   행렬            internal/contract/planshape.go 에 행을 더한다 (1.3).
                   질문 3 의 답이 B 나 C 면 cmd/runctl 의 0 도 함께 바뀐다
   decisions.md    6절에 실측 행 (질문 6 의 답).  팩 문장 둘과 CA5 의 이름 하나
   scene-gates.md  CA5 의 명령 (질문 6 의 답이 A 일 때만)
   U3              Grammar 의 「capability 는 닫힌 어휘」 줄 옆에 mcp.<이름> 이
                   속성으로 선다.  질문 4 의 답이 그 문장의 자리를 정한다
   U5              examples/pack.json 과 agent.pack 의 실행 쪽 (질문 5 의 답이
                   A 일 때 이 유닛이 넘긴다)
   GLOSSARY.md     질문 7 의 답
```

**CA0 의 한 값을 미리 적어 둔다.** `internal/contract` 를 단독으로 재면 오늘
79.5% 다 (표준 명령은 `-coverpkg=./...` 라 값이 다르다). 이 유닛이 그 패키지에
문장을 더하므로 **패키지별 80% 하한에 가장 가까운 자리가 여기다** — Code
Generation 이 표준 명령으로 다시 잰다.

---

## 6. 브랜치를 회차가 아니라 U1 에서 땄다

`CONVENTIONS.md` 3.1 은 유닛 브랜치를 **회차 브랜치에서** 따라고 적는다. 이
유닛은 `unit/isolation` 에서 땄다. 이유는 둘이다.

```text
   U1 이 아직 main 에 없다     CA1 은 사람이 재고 집행자는 구현자가 아니다.
                             그 게이트가 초록이 되기 전에는 병합 지점이 아니다
                             (CONVENTIONS 3.3)
   aidlc-state.md 가 한 장이다  같은 문서 루트(aidlc-docs/taeels/)의 파일이고
                             통째로 다시 쓰거나 자동 병합할 수 없다 (CLAUDE.md).
                             회차 브랜치에서 따면 U1 의 절과 U2 의 절이 파일
                             꼬리에서 각자 자라 병합에서 부딪친다
```

코드에서는 부딪히지 않는다 — 행렬이 U1 과 U2 의 파일 교집합을 0 으로 센다
(`internal/enode` 다섯 대 `internal/contract` 셋 + `agent.go`). **진행자에게
넘긴다** — U1 의 PR 이 먼저 `main` 에 들어가면 이 브랜치의 PR 차이는 U2 것뿐이고,
CA1 이 빨개져 U1 이 고쳐지면 이 브랜치는 그 위로 rebase 한다.
