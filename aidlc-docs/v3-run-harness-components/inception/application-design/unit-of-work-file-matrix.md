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
| `internal/enode/harness.go` | x | | | | x | **둘** |
| `internal/enode/claude.go` | x | | | | x | **둘** |
| `internal/enode/runner.go` | x | | | x | x | **셋 · 짝 팩과도 겹친다** |
| `internal/enode/hook.go` | x | | | | | |
| `internal/enode/config.go` | | | x | | | |
| `internal/enode/detect.go` | | | x | | | |
| `internal/enode/claim.go` | | | | x | | |
| `internal/enode/agent.go` | | x | | | | |
| `internal/contract/contract.go` | | x | | | | |
| `internal/contract/grammar.go` | | x | | | | |
| `internal/contract/examples/mcp.json` (새 파일) | | x | | | | |
| `cmd/iapadapter/config.go` | | | | | x | |
| `cmd/iapadapter/contract.go` | | | | | x | |

```text
   만지는 파일     14  (새 파일 둘 · 고치는 파일 열둘)
   만지는 패키지    3  internal/enode · internal/contract · cmd/iapadapter
   cmd/runctl       0  소스 diff 가 없다.  3절이 닿는 시험을 따로 적는다
   internal/api     0  등록 줄조차 안 는다
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
   U3   mcpUp · mcpFP · Fingerprinter 순회    U1 의 것을 안 고친다
   U4   resolveComponents 의 몸통           출처 둘과 거절을 더한다
   U5   Pack · PackFile · PackLimits · readPack
        resolveComponents 에 팩 출처를 더한다
```

**같은 자리를 만지는 것은 `resolveComponents` 하나**이고 U1 · U4 · U5 가 차례로
자란다. 직렬이라 충돌이 아니다 — 앞 유닛이 병합된 뒤 다음이 그 위에서 돈다.

### 2.2 `internal/enode/runner.go` — 셋이 만진다. 짝 팩과도 겹친다

```text
   U1   MkdirTemp 를 함수 몸통으로 · 실패 등급 · Instrument 를 언제나 호출 ·
        resolveComponents 호출 자리 · h.Fixed(tmp)
   U4   Job.NodeMCP 필드
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
   U1   Harness 인터페이스의 Fixed · Instrument 시그니처 · errAux
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
| `internal/enode/claude_test.go` · `env_test.go` | U1 | `Job{...}` 리터럴 셋이 여기 있다. 시그니처가 바뀌면 함께 고친다 |
| `internal/enode/worker_unix_test.go` | U1 | `stubHarness` 로 도는 시험들이 새 순서를 탄다 |

**`cmd/runctl` 의 소스 diff 는 0 이다.** `runctl example` 이 임베드 FS 를 읽고
`runctl schema steps` 는 구조체에서 뽑으므로 예시 하나로 목록과 출력이 함께 는다.
그래도 그 패키지의 시험이 새 예시 위로 돈다 — U2 가 그것을 자기 완료 조건에 진다.

---

## 4. 설계 요약 표에 없던 파일 둘

`application-design.md` 3절의 표를 실제 호출자와 대조해 찾았다.

### 4.1 `internal/enode/claim.go` — U4

```text
   claim.go:765    logBytes, h := runHarness(runCtx, ha, bin, Job{
```

`Job{...}` 리터럴은 **제품 코드에 이 한 자리뿐이다** (나머지 셋은 시험).
`NodeMCP: w.Local.MCP` 를 여기서 안 실으면 언제나 nil 이고 **노드 선언이 허용목록에
조용히 안 실린다.** 그 침묵이 이 회차가 고치려는 실패와 같은 종류다.

### 4.2 `internal/enode/agent.go` — U2

```text
   agent.go:36     type AgentParams struct      MCP []string · Pack string
   agent.go:500    parseAgentParams             steps[].agent 를 풀어 담는다
```

`components.md` 1.6 은 `agent.go` 를 적었고 요약 표만 안 적었다. 계약 어휘의
enode 쪽 끝이라 U2 가 함께 진다 — **계약이 받는 키와 그것을 푸는 코드가 갈리면
`400` 이 아닌데 값이 안 실리는 구멍이 난다.**

---

## 5. 짝 팩(transcript)과의 접점 — 하나

`constraints.md` 의 접점 절이 정본이고, 이 회차의 답 둘이 그것을 줄였다.

| 파일 | 이 팩 | 짝 팩 | 겹치나 |
|---|---|---|---|
| `claude.go` `Argv` | `stream-json --verbose` 한 줄 (U1 · ⑮) | 출력 형식과 스트림 처리 | **예** |
| `claude.go` `Fixed` · `Instrument` | U1 · U5 | 안 건드린다 | 아니오 |
| `claude.go` `Decode` | 안 건드린다 | 스트림을 훑게 바꾼다 | 아니오 |
| `hook.go` | U1 | 안 건드린다 | 아니오 |
| `harness.go` `HarnessResult` | U5 | 안 건드린다 | 아니오 |
| `runner.go` `Job` | U4 (`NodeMCP`) | tee 와 사건 배출 | **예** |
| `runner.go` `runHarness` | U1 · U5 | stdout 처리 | **예** |

**처리** — 팩 단위로 이 팩이 먼저다 (확인 질문 Q3 = A). U1 ~ U5 가 전부 `main` 에
들어간 뒤 짝 팩이 그 위에서 착수한다. **중간 상태의 `runner.go` 위로 짝 팩이
올라오지 않는다.**

`Job` 은 이미 `Transcript io.Writer` 를 들고 있다 — 짝 팩이 그 필드를 쓰고 이 팩이
`NodeMCP` 를 더한다. **같은 구조체의 다른 필드**라 병합이 기계적이다. 실제로
겹치는 것은 `runHarness` 의 몸통과 `Argv` 둘이고, 이 팩은 exec **앞**을, 짝 팩은 exec
**뒤**(stdout 처리)를 만진다.

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
