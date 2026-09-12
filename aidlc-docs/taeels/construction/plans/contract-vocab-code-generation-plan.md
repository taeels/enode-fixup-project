# U2 `contract-vocab` — Code Generation 계획

**유닛** `contract-vocab` · **브랜치** `unit/contract-vocab` · **담당** taeels ·
**닫는 게이트** CA3 의 앞 절반 (+ CA0) · **선행** 없음

**이 계획이 Code Generation 의 정본이다.** 여기 없는 것은 안 짓는다.
설계는 `construction/contract-vocab/functional-design/` 의 셋이고 규칙 번호
R1 ~ R6 은 그 `business-rules.md` 1절이다.

---

## 0. 앞 단계의 승인과 이 단계의 착수

사용자가 「다음」으로 다음 단계를 지시했고 그것이 Functional Design 의 승인을
겸한다. 같은 문서 루트의 U1 · obs 가 같은 판단을 했다.

**승인이 열어 준 것 둘을 이 단계가 싣는다** — 요구 팩의 문서다.

```text
   decisions.md 6절        실측 행 하나.  셋을 한 행에 (답 6=A)
   scene-gates.md CA5      out 과 in.from 의 이름을 pack 으로 (답 6=A)
```

---

## 1. 이 유닛이 내는 diff 의 모양

```text
   제품 파일 다섯    행렬 그대로 넷 + 행렬 밖 하나(planshape.go).  새 파일 하나
   시험 파일          기존 셋을 고치고 새 파일 0.  닿는 시험은 행렬 3절 그대로
   문서               행렬에 행 하나 · 팩 둘 · code-summary.md
   패키지 둘          internal/contract · internal/enode.  cmd/ 는 소스 diff 0
```

| 파일 | 무엇 | 스토리 |
|---|---|---|
| `internal/contract/contract.go` | `agentKeys` + 값 검사 (R2 ~ R6) | US-7 |
| `internal/contract/grammar.go` | `Grammar` 에 절 하나 | US-7 (계획 경로) |
| `internal/contract/planshape.go` | 목록 한 줄 + 설명 둘 | US-7 (계획 경로) |
| `internal/contract/examples/mcp.json` | 새 예시 (MCP 만) | US-4 의 읽는 면 |
| `internal/enode/agent.go` | `AgentParams` 둘 + 문구 | US-7 |

**이 유닛이 닫는 스토리는 US-7 하나다** (`unit-of-work-story-map.md` 0절) —
「타입을 틀리게 적으면 제출에서 걸린다」. 답 1=A 가 그 「제출에서」를 참으로
만든다.

---

## 2. 단계 — 여덟

### Step 1 — `internal/contract/contract.go`

- [x] `agentKeys` 에 `"mcp"` · `"pack"` 을 더한다 (`:902`). 목록의 정본이다
- [x] `knownKeys` 루프 다음에 값 검사를 건다 (`:1119` 뒤). 함수 하나로 모은다 —
      `agentValues(stepID, agent, in)` 이고 R2 ~ R6 을 그 안에서 본다
- [x] 검사 순서를 R6 -> R5 로 둔다. `in.from` 의 타입이 서야 대조를 할 수 있다
- [x] 오류는 첫 하나에서 돌아온다 — 오늘의 다른 검사와 같은 모양

### Step 2 — `internal/contract/grammar.go`

- [x] `Grammar` 에 절 하나. 자리는 「Choosing between run and agent」 다음 —
      에이전트 단계의 문맥이 거기다
- [x] 문장 셋이 `grammar_test.go` 의 `mustSay` 가 된다 (FD `business-rules.md` 5절)
- [x] 이름의 출처 한 줄을 **조건으로** 적는다 (답 4=A) — U3 이 아직 광고를 안
      지었으므로 속성이 0 인 동안에도 거짓이 안 되게

### Step 3 — `internal/contract/planshape.go`

- [x] `agent` 키 목록 줄에 `mcp` · `pack` (`:45`). 안 고치면 계획에게 거짓을 가르친다
- [x] 「Where each thing goes」에 두 줄을 더한다
- [x] 예시 블록의 `agent` 맵은 안 건드린다 (FD `business-logic-model.md` 4절)

### Step 4 — `internal/contract/examples/mcp.json`

- [x] FD `business-logic-model.md` 5절의 JSON 그대로
- [x] **Step 1 과 같은 커밋이다.** `agentKeys` 가 늘기 전에는
      `example_test.go` 가 `unknown field "mcp"` 로 확정 빨강이다 (실측 5.2)

### Step 5 — `internal/enode/agent.go`

- [x] `AgentParams` 에 `MCP []string` · `Pack string` (`:36`)
- [x] `parseAgentParams` 가 타입 오류를 R2 · R4 와 **같은 문장**으로 낸다 (`:500`).
      이 문구는 `Result.Error` 로 봉인에 들어간다 — 로그가 아니라 기록이다

### Step 6 — 시험

- [x] `grammar_test.go` 에 `bad` 셋 — R2 · R4 · R5. 각 줄이 「문법이 그 말을
      하는가」와 「그 말대로 거절되는가」를 함께 잰다
- [x] `contract_test.go` 에 R3 · R6 — 문법 대조가 없는 둘이다
- [x] `planshape` 의 `agent` 목록과 `agentKeys` 가 갈리지 않는 것을 잰다.
      `Grammar` 에는 그 장치가 있었고 `PlanShape` 에는 없었다 (FD 6절 ③)
- [x] `internal/enode` 에 `parseAgentParams` 의 문구와 값 실림을 잰다 —
      `agent.mcp` · `agent.pack` 이 `AgentParams` 까지 오는지, 타입이 틀리면
      어떤 문장이 나오는지

### Step 7 — 게이트

- [x] CA0 을 표준 명령으로 돈다 — `scripts/testdb.sh` 뒤 `go test ./...` ·
      커버리지(패키지별 80%) · `glyphscan` · `gofmt` · `vet` · 크로스 빌드 ·
      심볼 상한 · 워킹트리 청결 · **Mediator 라우트 26**
- [x] CA3 의 앞 절반을 잰다 — 모르는 `agent` 키가 `400` 이고
      `runctl example mcp` 가 나오고 `runctl lint` 가 경고 0 이다
- [x] 답 1=A · 2=A 가 더한 둘도 잰다 — `agent.mcp` 를 문자열로 적은 계약과
      `agent.pack` 을 적고 `in.from` 을 빠뜨린 계약이 거절된다

### Step 8 — 문서와 팩

- [x] `unit-of-work-file-matrix.md` 1절에 `planshape.go` 행 (그 문서 6절 ② 의 절차)
- [x] `decisions.md` 6절에 실측 행 하나 — 셋을 한 행에 (답 6=A)
- [x] `scene-gates.md` CA5 의 이름 (답 6=A)
- [x] `construction/contract-vocab/code/code-summary.md`
- [x] `aidlc-state.md` 와 `audit.md`

---

## 3. 갈래를 안 나눈다

산출이 작다 — 제품 파일 다섯에 늘어나는 줄이 150 안쪽이고 넷이 한 패키지다.
갈래를 나누면 `agentKeys` 가 느는 갈래와 예시를 넣는 갈래가 **서로를 기다린다**
(Step 4 의 순서 제약). U1 이 같은 이유로 안 나눴고 여기서는 그 이유에 규모가
하나 더 붙는다.

`internal/enode/agent.go` (Step 5)만 다른 패키지이고 독립이지만, 열 줄짜리
변경이라 갈래를 여는 비용이 이득보다 크다.

---

## 4. 이 계획이 정하는 것 — FD 에 없던 자리 둘

```text
   ① 검사를 함수 하나로 모은다
      agentValues(stepID, agent, in).  R2 ~ R6 이 한 자리에 있고 Validate 의
      단계 순회는 한 줄만 는다.  키마다 흩으면 R5 가 두 필드를 봐야 하는데
      그 대조가 어디 사는지가 흐려진다

   ② agent.pack 의 빈 문자열도 R4 로 거절한다
      빈 이름은 blob 이름이 아니다.  문구를 (a non-empty string) 으로 적어
      타입 위반과 빈 값이 같은 문장을 받는다 — FD 의 R4 문구를 그 한 마디로
      고친다 (같은 커밋)
```

---

## 5. 이 단계가 안 하는 것

```text
   agent 의 나머지 다섯 키 타입 검사   답 1=A 의 범위 밖.  비대칭을 진행자에게 넘긴다
   서버 이름의 글자 규칙              U3 · U4
   examples/pack.json               U5
   runctl schema 의 열거 면           답 3=A — 안 만든다
   GLOSSARY.md                      답 7=B — 이 유닛 밖의 커밋
   resolveComponents · 팩 펴기        U4 · U5
```
