# 파일 행렬 — 유닛마다 무엇을 만지나

`constraints.md` 끝 절이 이 단계에 건 요구다. **유닛마다 만지는 파일을 표로 세고,
한 파일을 둘 이상이 만지면 그 자리를 접점으로 적고 처리를 정한다.**

**세는 것이지 옮겨 적는 것이 아니다.** `application-design.md` 3절의 요약 표를
실제 호출자와 대조했고 **둘이 빠져 있었다** — 3.1 이 그것이다.

---

## 1. 행렬

`x` 는 만진다, 빈칸은 안 만진다.

| 파일 | U1 | U2 | U3 | U4 | U5 | 접점 |
|---|:---:|:---:|:---:|:---:|:---:|---|
| `internal/enode/mcp.go` (새 파일) | x | | x | x | x | **넷** |
| `internal/enode/mcp_test.go` (새 파일) | x | | | | | 시험. 직렬화와 빈 파일의 모양 |
| `internal/enode/harness.go` | x | | | | x | **둘** |
| `internal/enode/harness_test.go` | x | | | | | 시험. `TestHarnessRecordsBudget` 이 ⑯ 의 경계다 |
| `internal/enode/claude.go` | x | | | | x | **둘 · 짝 팩과도 겹친다** |
| `internal/enode/runner.go` | x | | | x | x | **셋 · 짝 팩과도 겹친다** |
| `internal/enode/hook.go` | x | | | | | |
| `internal/enode/instrument_test.go` (새 파일) | x | | | | | 시험. 계장의 다섯 쓰기 · 실패 등급 · 불변식 셋 |
| `internal/enode/logs_test.go` (새 파일) | x | | | | | 시험. `logs/` 허용목록 (⑱) |
| `internal/enode/config.go` | | | x | | | |
| `internal/enode/detect.go` | | | x | | | |
| `internal/enode/claim.go` | | | | x | | |
| `internal/enode/agent.go` | | x | | | | |
| `internal/contract/contract.go` | | x | | | | |
| `internal/contract/grammar.go` | | x | | | | |
| `internal/contract/planshape.go` | | x | | | | 요약 표에 없던 셋째. 4.3 절 |
| `internal/contract/examples/mcp.json` (새 파일) | | x | | | | |
| `internal/enode/config_mcp_test.go` (새 파일) | | | x | | | 시험. 거절 여섯과 성한 선언 |
| `internal/enode/resolve_test.go` (새 파일) | | | | x | | 시험. 합치는 규칙 R1 ~ R11 과 `readWorkspaceMCP` 의 넷 |
| `internal/enode/detect_mcp_test.go` (새 파일) | | | x | | | 시험. 뜨나 판정 · 순회 · 문턱 |
| `internal/api/ui/static/shared/fleet/format.mjs` | | | x | | | 4.4 절. U3 의 답 7=B |
| `internal/api/ui/tests/format.test.mjs` | | | x | | | 시험. 그 짝 |
| `cmd/iapadapter/config.go` | | | | | x | |
| `cmd/iapadapter/contract.go` | | | | | x | |

**시험 파일은 제품 파일과 따로 센다.** 아래 열여섯은 제품 파일이고, U1 이 낸
시험 파일 셋(`mcp_test.go` · `instrument_test.go` · `logs_test.go`)과 U3 이 낸
셋(`config_mcp_test.go` · `detect_mcp_test.go` · `tests/format.test.mjs`)과
U4 가 낸 하나(`resolve_test.go`)는 새 코드의 짝이라 표에만 적고 이 수에 안 넣는다 — 그래야 「만지는 패키지」와
「`cmd/runctl` 0」이라는 값이 흐려지지 않는다. 이미 있던 시험이 빨개지는 자리는
3절이 따로 센다.

```text
   만지는 파일     17  (새 파일 둘 · 고치는 파일 열다섯).  U2 가 planshape.go 를
                       더하고 U3 이 format.mjs 를 더했다 — 4.3 · 4.4 절.
                       첫 판은 15 였다.  U4 는 제품 파일을 안 더했다 —
                       셋 다 이미 행렬에 있다
   만지는 패키지    4  internal/enode · internal/contract · cmd/iapadapter ·
                       internal/api/ui.  넷째는 U3 의 답 7=B 가 더했다
   cmd/runctl       0  소스 diff 가 없다.  3절이 닿는 시험을 따로 적는다
   internal/api     0  라우트도 등록 줄도 안 는다.  ui/static 은 그 패키지의
                       정적 자산이라 따로 센다 (4.4 절)
   internal/match   0  코드 diff 0.  다만 정렬 기준이 광고 attr 개수라
                       동작은 CA3 이 잰다 (application-design.md 6.4)
   internal/store · panel · api/ui   0
```

---

## 2. 심볼까지 — 접점 파일 넷

행렬이 쓰이려면 파일 이름으로는 모자란다. **같은 파일의 다른 자리**를 만지는지,
**같은 자리**를 만지는지가 갈리기 때문이다.

### 2.1 `internal/enode/mcp.go` — 넷이 만진다

```text
   U1   MCPServer · Components 형식.  resolveComponents 의 뼈대
        (요청도 팩도 없으면 빈 Components 를 낸다)
   U3   mcpUp · mcpFP · MCPServer 의 태그와 Extra 와 UnmarshalYAML ·
        allowlistEntry 가 Extra 를 얹는 줄
        **U1 의 것을 고친다** — 첫 판은 「안 고친다」였고 그것이 틀렸다 (4.4 절)
   U4   resolveComponents 의 몸통           출처 둘과 거절 셋을 더한다.
        Components.Servers 의 타입을 MCPServer 에서 최종 허용목록 항목으로 올린다
        (답 2=A) — mcpAllowlistJSON · writeMCPAllowlist 의 인자가 함께 바뀐다.
        새 이름 다섯 — readWorkspaceMCP · ensureType · copyEntry · hasText ·
        wantedMCP · notAvailable · entryNames.
        **allowlistEntry 는 안 고친다** — 종류를 채우는 것은 두 출처가 함께
        지나는 공용 자리다
   U5   Pack · PackFile · PackLimits · readPack
        resolveComponents 에 팩 출처를 더한다
```

**같은 자리를 만지는 것은 `resolveComponents` 하나**이고 U1 · U4 · U5 가 차례로
자란다. 직렬이라 충돌이 아니다 — 앞 유닛이 병합된 뒤 다음이 그 위에서 돈다.

### 2.2 `internal/enode/runner.go` — 셋이 만진다. 짝 팩과도 겹친다

```text
   U1   MkdirTemp 를 함수 몸통으로 · 실패 등급 · Instrument 를 언제나 호출 ·
        resolveComponents 호출 자리 · h.Fixed(tmp)
   U4   Job 의 필드 넷 (NodeMCP · WorkspaceMCP · WorkspaceMCPErr · Log) ·
        ② 뒤에서 Notes 를 찍는 줄.  오류 검사보다 앞이다
   U1   링 tee 를 끄는 자리 (⑲ · :104-105) · logs/ 에 실을 것을 고르는 자리 (⑱ · :155-156).
        둘 다 exec 뒤다 — 짝 팩의 스트림 처리와 겹친다
   U5   ⑧ 에서 HarnessResult.MCP · .Pack 을 채운다
```

**U1 이 호출 순서의 최종형을 세운다.** ② 의 `resolveComponents` 호출을 U1 이
미리 놓으므로 U4 는 `Job` 에 필드만 더하고 순서를 다시 안 만진다.

### 2.3 `internal/enode/claude.go` — 둘이 만진다

```text
   U1   Fixed(dir) · Instrument 의 가짜 홈 · 자격증명 복사 · mcp.json 쓰기 · 플래그
   U5   Instrument 안에서 팩을 펴는 블록
```

`Decode` 는 **어느 유닛도 안 만진다.** `Argv` 는 U1 이 플래그 한 줄을 더한다 (⑮) —
그래서 짝 팩과의 겹침이 하나가 아니라 **둘**이다.

### 2.4 `internal/enode/harness.go` — 둘이 만진다

```text
   U1   Harness 인터페이스의 Fixed · Instrument 시그니처 · errAux ·
        ParseClaude 의 type 검사 (⑯).  harness_test.go 에 그 시험을 더한다
   U5   HarnessResult 의 MCP · Pack 필드
```

**다른 자리다.** U1 은 인터페이스를, U5 는 결과 구조체를 만진다.

---

## 3. 닿는 시험 — 만지지는 않는데 빨개질 수 있는 자리

**「만지는 파일」과 섞지 않는다.** 섞으면 `cmd/runctl` 의 0 이라는 값이 흐려진다.

| 시험 | 어느 유닛이 건드리나 | 왜 |
|---|---|---|
| `cmd/runctl/shape_test.go` `TestExamples_LintClean` | U2 | `contract.ExampleNames()` 를 돌므로 새 예시가 lint 경고를 내면 빨개진다 |
| `internal/contract/example_test.go` | U2 | 같은 이유. 파싱 · `Validate` · `success_when` 유무를 잰다 |
| `internal/contract/grammar_test.go` | U2 | `Grammar` 의 문장마다 「어긴 계약이 거절되는가」를 잰다 |
| `internal/enode/hook_test.go` | U1 | ⑧ 이 훅 설정의 경로와 이름을 `<tmp>/enode-settings.json` 에서 `<tmp>/home/settings.json` 으로 옮기므로 **넷이 확정적으로 빨개진다** — `:215` · `:247` · `:277` · `:300`. U1 이 함께 고친다 |
| `internal/enode/claude_test.go` · `env_test.go` | U1 | `Job{...}` 리터럴 셋이 여기 있다. 그리고 `claude_test.go:14-15` 의 `TestAdapter_ArgvIsPure` 가 `--output-format json` 을 완전 일치로 재므로 ⑮ 가 그것을 **확정적으로 빨갛게** 만든다 |
| `internal/enode/worker_unix_test.go` | U1 | `stubHarness` 로 도는 시험들이 새 순서를 탄다 |
| `internal/enode/enode_test.go` `TestDetectEmpty` | U3 | 광고에 실릴 수 있는 키를 `harness` · `os` · `host_arch` 로 못 박는다. `harness.<이름>` 이 늘면서 **확정 빨강**이라 U3 이 그 접두를 함께 허용한다 |
| `internal/enode/mcp_test.go` `TestMCP_AServerIsCopiedIntoTheAllowlist` | U3 | `env` 가 값 그대로 실리는 것을 재는데, 답 3=A 가 그것을 `${이름}` 참조로 바꾼다. U3 이 함께 고친다 |
| `internal/enode/mcp_test.go` 셋 | U4 | `mcpAllowlistJSON` · `writeMCPAllowlist` 의 인자가 `MCPServer` 에서 항목으로 바뀌어 **확정 빨강**이다. U4 가 함께 고친다 — 항목을 짓는 것은 `allowlistEntry`, 직렬화는 `mcpAllowlistJSON` 으로 갈렸다 |
| `internal/enode/worker_unix_test.go` | U4 | 시험 둘을 더한다 — 없는 이름이 하네스를 안 띄우는 것과, 허용목록에 요청된 것만 실리는 것. 스텁이 `--mcp-config` 의 파일을 `$OUT` 으로 옮겨 **그 단계가 실제로 쓴 파일**을 잰다 |

**`cmd/runctl` 의 소스 diff 는 0 이다.** `runctl example` 이 임베드 FS 를 읽고
`runctl schema steps` 는 구조체에서 뽑으므로 예시 하나로 목록과 출력이 함께 는다.
그래도 그 패키지의 시험이 새 예시 위로 돈다 — U2 가 그것을 자기 완료 조건에 진다.

---

## 4. 설계 요약 표에 없던 파일 넷

`application-design.md` 3절의 표를 실제 호출자와 대조해 둘을 찾았고, **셋째는
U2 가 자기 FD 에서 찾았다** (4.3) — 표 대조가 아니라 「그 목록이 늘면 어느 문장이
거짓이 되나」로 찾은 것이라 이 절의 방법이 둘로 는다. **넷째는 U3 의 답이
들여왔다** (4.4) — 설계가 안 빠뜨린 자리이고 결정이 새로 연 자리다.

### 4.1 `internal/enode/claim.go` — U4

```text
   claim.go:765    logBytes, h := runHarness(runCtx, ha, bin, Job{
```

`Job{...}` 리터럴은 **제품 코드에 이 한 자리뿐이다** (나머지 셋은 시험).
`NodeMCP: w.Local.MCP` 를 여기서 안 실으면 언제나 nil 이고 **노드 선언이 허용목록에
조용히 안 실린다.** 그 침묵이 이 회차가 고치려는 실패와 같은 종류다.

**U4 가 이 파일에 한 자리를 더했다** — 워크스페이스 `.mcp.json` 을 읽는 호출이다
(답 1=A). 여는 것은 가장자리이고 고르는 것은 `resolveComponents` 다. 조건이 둘이고
(요청이 0 이면 안 열고, 워크스페이스가 없으면 출처도 없다) 읽기 실패는 `Job` 에
실려 나간다 — 등급을 정하는 것이 정책이기 때문이다.

### 4.2 `internal/enode/agent.go` — U2

```text
   agent.go:36     type AgentParams struct      MCP []string · Pack string
   agent.go:500    parseAgentParams             steps[].agent 를 풀어 담는다
```

`components.md` 1.6 은 `agent.go` 를 적었고 요약 표만 안 적었다. 계약 어휘의
enode 쪽 끝이라 U2 가 함께 진다 — **계약이 받는 키와 그것을 푸는 코드가 갈리면
`400` 이 아닌데 값이 안 실리는 구멍이 난다.**

### 4.3 `internal/contract/planshape.go` — U2

```text
   planshape.go:45   agent   max_turns, max_tokens, ask, model, harness — nothing else
   planshape.go:52   Any other key under agent or in is rejected
```

`PlanShape` 는 계획에게 **필드의 모양**을 실제 값으로 보여주는 상수다
(`ADR-057`). 첫 줄이 허용 키를 「nothing else」로 못 박으므로 `agentKeys` 에
`mcp` · `pack` 이 느는 순간 **그 문장이 계획에게 거짓을 가르친다** — 「적으면 계획
전체가 거절된다」로 읽히고, 그것은 `features.md` 3.5 의 「계획이 짓는 단계도 같은
문법을 쓴다」와 정반대다.

`components.md` 1.6 은 `Grammar` 만 적었고 이 파일은 행렬의 첫 판에도 없었다.
**U2 가 찾아서 더했다** (2026-09-13 · 그 유닛의 FD 1.3). `Grammar` 에는
`grammar_test.go` 라는 낡음 방지 장치가 있는데 `PlanShape` 에는 없었으므로
같은 유닛이 그 시험도 세웠다.


### 4.4 `internal/api/ui/static/shared/fleet/format.mjs` — U3

```text
   format.mjs:48   attributeName    표에 없는 키를 원문으로 낸다
   format.mjs:49   attributeValue   값을 해석하는 갈래 넷
```

**설계가 빠뜨린 것이 아니라 결정이 연 자리다.** `scene-gates.md` 2.1 은
「현황판 카드는 새 키를 다른 속성과 같은 칩으로 보인다 — 자동이다」로 적었고
그것은 참이다. 다만 자동으로 보이는 모양이 **원문 키**라서, 같은 사실이
`실행 도구 = claude` 와 `harness.claude = 1` 두 줄로 보인다. U3 의 FD 물음 7 이
그것을 물었고 답 7=B 가 이름을 주기로 했다.

그래서 `internal/api/ui` 의 0 이 이 회차에서 깨진다. `internal/api` 자체는
그대로 0 이다 — 라우트도 핸들러도 안 는다.

### 4.5 U3 이 만진 U1 의 코드 — 2.1 의 교정

`mcp.go` 의 U3 칸이 첫 판에 「U1 의 것을 안 고친다」였는데 **그것이 틀렸다.**
U1 이 `allowlistEntry` 주석에 「U3 이 「그 밖의 키」를 그대로 옮긴다」로 이음매를
남겼고, `decisions.md` 2절이 그 통과를 값으로 박았다. 그런데 `MCPServer` 는
필드 다섯의 구조체이고 yaml 은 모르는 키를 말없이 버리므로 **옮길 맵이 애초에
안 생긴다.** 담을 필드(`Extra`)와 그것을 푸는 `UnmarshalYAML`, 그리고 얹는
`allowlistEntry` 가 전부 U1 의 파일에 서야 한다.

**행렬이 이음매 주석과 갈려 있었던 것이고**, 갈린 자리를 U3 의 FD 가 찾았다.

---

## 5. 짝 팩(transcript)과의 접점 — 둘

`constraints.md` 의 접점 절이 정본이고, 이 회차의 답 둘이 그것을 줄였다.

| 파일 | 이 팩 | 짝 팩 | 겹치나 |
|---|---|---|---|
| `claude.go` `Argv` | `stream-json --verbose` 한 줄 (U1 · ⑮) | 출력 형식과 스트림 처리 | **예** |
| `claude.go` `Fixed` · `Instrument` | U1 · U5 | 안 건드린다 | 아니오 |
| `claude.go` `Decode` | 안 건드린다 | 스트림을 훑게 바꾼다 | 아니오 |
| `hook.go` | U1 | 안 건드린다 | 아니오 |
| `harness.go` `HarnessResult` | U5 | 안 건드린다 | 아니오 |
| `runner.go` `Job` | U4 (`NodeMCP`) · U1 (하네스 단계 tee 끄기 · ⑲) | 사건 배출 · 링 되살리기 | **예** |
| `runner.go` `runHarness` | U1 · U5 | stdout 처리 | **예** |

**처리** — 팩 단위로 이 팩이 먼저다 (확인 질문 Q3 = A). U1 ~ U5 가 전부 `main` 에
들어간 뒤 짝 팩이 그 위에서 착수한다. **중간 상태의 `runner.go` 위로 짝 팩이
올라오지 않는다.**

`Job` 은 이미 `Transcript io.Writer` 를 들고 있다 — 짝 팩이 그 필드를 쓰고 이 팩이
`NodeMCP` 를 더한다. **같은 구조체의 다른 필드**라 병합이 기계적이다. 실제로
겹치는 것은 `runHarness` 의 몸통과 `Argv` 둘이다. 이 팩은 exec **앞**과 `logs/`
조립(⑱)을 만지고, 짝 팩은 exec **뒤**의 스트림 처리와 tee 를 만진다.

---

## 6. 행렬 밖을 만지면

`CONVENTIONS.md` 3.4 의 「싣는 것」을 벗어난 diff 다. **가져갈 기록에 들어간다**
(`CONVENTIONS.md` 3.5 — 「행렬 밖 파일을 만진 diff」).

```text
   유닛이 행렬 밖 파일을 고쳐야 하면
     ①  왜 필요한지를 그 유닛의 audit.md 에 적는다
     ②  이 문서에 행을 더한다.  행렬이 사실과 갈리면 접점을 못 센다
     ③  그것이 다른 유닛의 자리면 그 유닛으로 넘긴다
```

**행렬은 계획이 아니라 측정이다.** 틀린 채로 두면 다음 사람이 접점을 모르고 만진다.
