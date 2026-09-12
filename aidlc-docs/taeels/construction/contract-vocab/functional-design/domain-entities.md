# U2 `contract-vocab` — 도메인 개체

계약이 `agent.mcp` · `agent.pack` 을 적을 수 있게 하는 **어휘의 형식**이다.
값이 무엇이고 어디까지가 그 값의 이름인지를 여기서 정한다. 거절과 문구는
`business-rules.md`, 값이 지나가는 길은 `business-logic-model.md` 다.

답 일곱은 계획 2절에 있다 — `A · A · A · A · A · A · B`.

---

## 0. 이 문서가 드는 것과 안 드는 것

```text
   든다      agent 맵의 알려진 키 일곱과 각 값의 타입
             AgentParams 의 필드 둘과 왕복의 성질
             agent.pack 이 가리키는 이름이 무엇의 이름인가
             빈 배열과 부재가 같은가
             requires[].mcp.<이름> 이 코드를 0 으로 두는 근거

   안 든다   허용목록을 합치는 규칙        U4 의 resolveComponents
             팩 tar 의 디렉터리 규약과 상한  U5
             enode.yaml 의 mcp: 절 필드     U3
             HarnessResult 의 직렬화        U5
```

---

## 1. `agent` 맵 — 알려진 키 일곱

| 키 | 타입 | 누가 읽나 | 안 적으면 |
|---|---|---|---|
| `model` | 문자열 | 어댑터 | 하네스의 기본 모델 |
| `max_turns` | 정수 | 어댑터 | 상한 없음 |
| `max_tokens` | 정수 | 어댑터 | 상한 없음. 개명은 이 팩 밖이다 (`decisions.md` 4절) |
| `ask` | 문자열 | 어댑터 | 하네스의 기본값. 무인 실행이라 계약이 못 박는다 |
| `harness` | 문자열 | `runAgentStep` | `claude` |
| **`mcp`** | **문자열 배열** | `resolveComponents` (U4) | **허용목록이 빈다 — 오늘 그대로** |
| **`pack`** | **문자열** | 계장 (U5) | **팩 없이 돈다 — 오늘 그대로** |

**목록의 정본은 하나다** — `internal/contract/contract.go` 의 `agentKeys`.
두 벌로 두면 갈린다 (`CONVENTIONS.md` 1.4 의 같은 규칙). `knownKeys` 의 거절
문구가 이 목록을 그 자리에서 나열하므로 **목록이 늘면 사람이 읽는 안내도 함께
는다** — 답 3=A 가 새 열거 면을 안 만들고 이 성질에 기댄 이유다.

**「안 적으면 오늘 그대로」가 둘의 공통 성질이다.** 새 키가 필수가 되면 이미 도는
계약이 전부 `400` 이 된다. `ADR-034` 5절이 `agent.pack` 을 「오늘 비어 있어도
된다」로 적은 것이 그 값이다.

---

## 2. `AgentParams` — enode 쪽 끝

```text
   MCP   []string  json:"mcp,omitempty"    계약의 agent.mcp
   Pack  string    json:"pack,omitempty"   계약의 agent.pack
```

`internal/enode/agent.go:36` 의 구조체에 필드 둘이 는다. **계약이 받는 키와
그것을 푸는 구조체가 갈리면 `400` 이 아닌데 값이 안 실리는 구멍이 난다** —
`contract.go:898` 의 주석이 그 짝을 이미 적어 뒀고, 파일 행렬 4.2 가 같은
이유로 이 파일을 U2 에 붙였다.

**왕복의 성질 하나를 적어 둔다.** `omitempty` 는 빈 슬라이스와 빈 문자열을
뺀다. 그래서 `agent.mcp: []` 인 계약이 이 구조체를 지나 다시 JSON 이 되면 키가
사라진다 — 3절의 「빈 배열과 부재가 같다」가 동작뿐 아니라 **기록에서도** 참이
되는 자리다.

---

## 3. `agent.mcp` — 값은 이름의 목록이다

```text
   타입          문자열 배열.  ["probe"] · ["gerrit", "serial"]
   빈 배열과 부재  같다.  둘 다 허용목록이 빈다
   중복          허용한다.  허용목록이 이름으로 키를 지으므로 뒤에서 접힌다 (U4)
   빈 문자열      이름이 아니다.  거절한다 (business-rules R3)
   순서          뜻이 없다.  우선순위는 출처(팩 · 노드 · 워크스페이스)가 정한다 (U4)
```

**빈 배열과 부재가 같은 이유** — U4 의 `resolveComponents` 는 「`agent.mcp` 가
적은 이름만 집는다」다 (`decisions.md` 6절 ⑥). 0 개를 적은 것과 안 적은 것은
집는 이름이 0 이라는 같은 결과를 낸다. **둘을 가르면 갈 곳이 없다** — 「빈
배열이면 전부 연다」는 ⑥ 이 정반대로 닫은 값이고, 「빈 배열이면 오류」는 계약이
「아무것도 안 물린다」를 명시로 적을 길을 없앤다.

**이름의 글자 규칙은 이 유닛이 안 정한다.** 그 이름은 U3 에서 광고 속성
`mcp.<이름>` 의 꼬리가 되고 U5 에서 `mcp.json` 의 키가 된다 — 글자에 제약이
필요하면 그 두 자리가 근거를 갖는다. 이 유닛은 **빈 문자열만** 거절하고 나머지는
넘긴다 (`business-rules.md` 7절).

---

## 4. `agent.pack` — 값은 blob 하나의 이름이다

```text
   타입      문자열.  배열이 아니다 — 팩은 tar 하나다 (features.md 3.6)
   가리키는 것  이 Run 의 blob 이름.  그것이 곧 $IN 에 깔리는 파일의 이름이다
   점         이름의 한 글자다.  단계 참조가 아니다
   있으려면    같은 단계의 in.from 이 그 이름을 적어야 한다 (답 2=A · R5)
```

### 4.1 점이 단계 참조가 아니라는 근거 셋

```text
   ① blob 이름 공간이 Run 하나에 평평하다
      PUT /v1/runs/{run}/steps/{seq}/blob/{name}   올리는 쪽은 자기 seq 를 안다
      GET /v1/runs/{run}/blob/{name}               받는 쪽은 이름만 안다
      (internal/api/api.go:107-108.  주석이 "소비자는 이름만 안다" 를 적었다)

   ② internal/enode/claim.go:520 이 그 이름을 그대로 $IN 의 파일 이름으로 쓴다
      os.Create(filepath.Join(in, name)) 한 줄이다.  쪼개는 코드가 없다

   ③ examples/multi.json 이 이미 평평한 이름으로 돈다
      out: ["lines.txt"] -> in.from: ["lines.txt"]
```

`ADR-034` 2.2 의 `in.from: ["prep.pack"]` 은 **앞 단계가 `out: ["prep.pack"]`
이라고 적었을 때 도는 것**이다. 그 표기가 「단계 이름 + 점 + blob 이름」으로
읽히도록 생긴 것이 `features.md` 3.5 의 `<단계>.<blob 이름>` 이고, 런타임에는
그것을 푸는 코드가 없다. **이 차이가 게이트 하나를 깼다** —
`business-logic-model.md` 8절.

### 4.2 팩 하나만 실을 수 있다

값이 배열이 아니므로 한 단계에 팩은 하나다. 근거는 **이름 충돌의 판정이 하나여야
하기 때문**이다 — 팩의 `mcp.json` 이 노드 선언 이름을 덮으면 거절이고
(`decisions.md` 6절 ⑦), 팩이 둘이면 「어느 팩이 먼저인가」가 그 판정 앞에
끼어든다. 여럿이 필요해지면 값의 타입을 넓히는 것이 아니라 팩을 합치는 것이
`ADR-034` 2.2 의 결에 맞는다 (계약이 아니라 팩 단계가 합친다).

---

## 5. `requires[].mcp.<이름>` — 코드 변경이 0 이다

```text
   Require 의 미지 키는 Attrs 로 들어간다      커스텀 언마샬러.  contract.go
   값은 문자열 "1"                            속성 값은 문자열이다
   매처는 부분집합 비교 그대로                  새 술어가 0 이다
```

`features.md` 3.5 의 「이미 있는 문법이다. 속성 키 하나가 늘 뿐」이 **참임을
확인했다**. 이 유닛이 만지는 파일에 매처도 `Require` 도 없다.

**갈리는 자리를 이름으로 적어 둔다** — `requires` 의 `mcp.<이름>` 은 **매칭
조건**이고 `agent.mcp` 는 **실행 파라미터**다 (`ADR-013` 결정 4). 하나만 적으면
둘 다 조용히 반쯤 돈다.

```text
   requires 만 적었다     그 노드에 배정되지만 서버가 안 뜬다 (허용목록이 빈다)
   agent.mcp 만 적었다   그 서버가 없는 노드에 배정될 수 있고, 그러면 U4 가
                        mcp server <이름> is not available on this node 로 죽인다
```

**둘을 묶어 검증하지 않는다.** `agent.mcp` 의 이름이 `requires` 의
`mcp.<이름>` 에도 있어야 한다는 규칙은 두지 않는다 — 워크스페이스
`.mcp.json` 과 팩이 같은 이름을 줄 수 있고(U4 의 출처 셋), 그 둘은 광고에 안
나오므로 `requires` 에 적을 수 없다. 규칙으로 만들면 **정당한 계약이 거절된다.**

---

## 6. 이 유닛이 만지는 파일

| 파일 | 무엇이 는가 | 행렬 |
|---|---|---|
| `internal/contract/contract.go` | `agentKeys` 에 둘 · 값의 타입 검사 · `agent.pack` 과 `in.from` 대조 | 1절에 있다 |
| `internal/contract/grammar.go` | `agent.mcp` · `agent.pack` 문장과 이름의 출처 한 줄 | 1절에 있다 |
| `internal/contract/examples/mcp.json` | 새 예시 하나 (MCP 만) | 1절에 있다 (새 파일) |
| `internal/enode/agent.go` | `AgentParams` 의 필드 둘 · `parseAgentParams` 의 문구 | 4.2 절에 있다 |
| **`internal/contract/planshape.go`** | **`PlanShape` 의 거짓이 되는 두 줄** | **행이 없다 — 이 유닛이 더한다** |

닿는 시험 셋은 행렬 3절 그대로다 — `grammar_test.go` ·
`internal/contract/example_test.go` · `cmd/runctl/shape_test.go` 의
`TestExamples_LintClean`.

---

## 7. 안 만지는 것을 확인했다

```text
   Require · internal/match     미지 키가 Attrs 로 가고 매처는 부분집합 그대로 (5절)
   cmd/runctl 의 소스           답 3=A.  diff 0 이 남는다
   lintWarnings                success_when 덮임만 본다.  함대를 안 본다 —
                              새 예시가 안 뜬 서버를 요구해도 경고가 0 이다
   attrLine (agent.go)         head 여덟 뒤에 나머지 속성을 이름순으로 전부 붙인다.
                              U3 이 mcp.<이름> 을 광고하면 계획의 프롬프트에
                              저절로 보인다.  답 4=A 가 그 성질에 기댄다
   enode.Step.In               구조체 그대로 (Prompt · From).  in.from 의 타입
                              검사는 Mediator 쪽에서만 는다 (R6)
```
