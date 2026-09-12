# Unit of Work — v3-run-harness-components

`requirements/harness-components/` 팩을 **유닛 다섯**으로 가른다 (계획 Q1 = A).
가른 기준은 기능(`features.md` 3.1 ~ 3.7)이고, **유닛의 끝을 정하는 것은 조각
게이트**(`scene-gates.md` CA0 ~ CA6)다.

담당은 `taeels` 하나다 (Requirements 확인 질문 Q2 = B). 문서 루트는
`aidlc-docs/taeels/` 이고 유닛은 `unit/<유닛>` 브랜치에서 **직렬로** 돈다.

```text
   착수 순서   U1 isolation -> U2 contract-vocab -> U3 advert
               -> U4 sources -> U5 pack
```

---

## 0. 다섯을 한눈에

| | 유닛 | 맡는 기능 | 닫는 게이트 | 선행 |
|---|---|---|---|---|
| U1 | `isolation` | 3.1 · 3.2 의 최소 | CA1 | 없음 |
| U2 | `contract-vocab` | 3.5 | CA3 의 절반 | 없음 |
| U3 | `advert` | 3.3 | CA2 · CA3 완결 | U1 · U2 |
| U4 | `sources` | 3.4 · 3.2 의 완성 | CA4 | U1 · U2 · U3 |
| U5 | `pack` | 3.6 · 3.7 | CA5 · CA6 | U1 · U2 · U4 |

**모든 유닛의 완료 조건에 CA0 가 들어간다** (계획 Q5 = A). 근거는 4절.

---

## 1. U1 — `isolation`

**책임** — 하네스의 사적인 세계를 계장 임시 디렉터리 아래에 짓고, 그 세계 밖이
안 보이게 한다. **무엇을 열지는 안 정한다** — 언제나 빈 목록으로 연다.

겹이 둘이라는 것이 이 유닛의 값이다 (`services.md` 1.2) — 가짜 홈이 사람의
`~/.claude` 를 끊고, `--strict-mcp-config` 가 우리가 준 파일 밖을 안 뜨게 한다.
**첫 겹은 로그아웃 홈에서만 쟀다** — CA1 이 로그인된 홈에서 다시 잰다.

```text
   internal/enode/mcp.go       새 파일.  MCPServer · Components ·
                               resolveComponents (요청도 팩도 없는 경우만)
   internal/enode/harness.go   Fixed(dir) · Instrument(..., c Components) 시그니처 · errAux
   internal/enode/harness.go   ParseClaude 가 switch 앞에서 type 을 본다 (⑯).
                               "result" 가 아니면 ReasonError.  그때도 Turns ·
                               CostUSD · Session 은 채운다.  switch 뒤에 두면
                               subtype 에 token 이 든 사건이 max_tokens 로 떨어져
                               Completed() 가 참이 된다
   internal/enode/claude.go    Argv 에 --output-format stream-json --verbose 한 줄 (⑮).
                               게이트가 재는 system/init 줄이 그래야 logs/ 에 남는다.
                               Decode 는 안 건드린다 — 최종 result 사건은 그대로
                               집힌다.  크래시 경로는 ⑯ 이 막는다.
                               Fixed 가 CLAUDE_CONFIG_DIR 을 박는다.
                               Instrument 가 <dir>/home 을 가장 먼저 짓고
                               훅 설정을 그 안의 settings.json 으로 쓰고
                               .credentials.json 을 0600 으로 복사하고
                               <dir>/mcp.json 을 쓴다.
                               플래그 --strict-mcp-config --mcp-config=<경로>
   internal/enode/runner.go    logs/ 선별 (⑱).  허용목록 — init 과 최종 result 와
                               stderr 만 전문이고 나머지는 껍데기다.
                               계장 디렉터리를 함수 몸통으로.  못 만들면 단계 실패.
                               resolveComponents 를 exec 전에 부른다.
                               Instrument 를 언제나 부르고 오류를 등급으로 가른다
                               (기본이 치명 · errAux 만 보조 · 삼켜도 플래그는 붙인다)
   internal/enode/hook.go      훅 파일을 가짜 홈 안으로 옮긴다.
                               self 가 비면 hooks 키 자체를 안 쓴다 —
                               shellJoin 이 앞이 빈 명령을 만드는 것을 막는다
```

**완료 조건**

- `components.md` 3절의 실패 등급 표 중 계장 디렉터리 · 가짜 홈 · 허용목록 쓰기 ·
  자격증명 복사 네 줄이 코드로 있다. **치명이 전부 exec 앞이다**
- **기본이 치명이다** (`application-design.md` 4.1). `errAux` 로 감싼 자리가
  **훅 설정 쓰기 하나뿐**임을 시험이 잰다 (`decisions.md` 6절 ⑪ — 기준 시각은
  `Instrument` 앞에서 불려 구조적으로 이 등급에 안 닿는다)
- **불변식 셋이 코드로 있다** (4.3) — `<dir>/home` 을 가장 먼저 만든다 ·
  보조 실패에도 이미 얻은 플래그를 돌려준다 · **보조 오류로 조기 반환하지 않는다**.
  시험이 직접 잰다: 훅 쓰기를 실패시켜도 `<dir>/mcp.json` 이 쓰이고
  `--strict-mcp-config` 가 argv 에 있다
- **계장 보존 스위치를 안 만든다.** 한 번 넣었다가 뺐다 — `features.md` 3.1 의
  「복사한 자격증명은 계장 디렉터리와 함께 단계 끝에 지워진다」를
  뚫기 때문이다. 그 대가로 **눈 검증이 복제본을 잰다**는 한계가 남고, U1 의
  완료 조건이 그 한계를 이름으로 적는다 — 사람이 손으로 띄우는 경로는 환경 ·
  플래그 · 게이트웨이 인증에서 실물과 갈린다
- **크래시가 성공으로 안 봉인된다** (⑯). `ParseClaude` 가 switch **앞에서**
  `type` 을 보고 `"result"` 가 아니면 `ReasonError` 다. **시험 입력은 줄 경계에서
  끊긴 stdout** 이어야 한다 — 마지막 완결 객체가 `{"type":"assistant",...}` 인
  것. 객체 중간에서 끊으면 `lastJSONObject` 가 `}` 로 안 끝나 오늘 코드도 이미
  `harness_error` 라 **⑯ 을 안 재는 시험**이 된다.
  `TestHarnessRecordsBudget` 이 그대로 초록이어야 한다 — `type` 없는 픽스처로
  `Turns` · `CostUSD` 를 재므로 조기 반환으로 짜면 빨갛다.
  `harness.go:98-99` 의 「종료코드 0 을 믿지 않는다」가 ⑮ 뒤에도 참이어야 한다
- **`logs/` 가 허용목록이다** (⑱). `system/init` 과 최종 `result` 와 stderr 는
  **전문**이고 그 밖의 모든 사건은 **껍데기**(사건 종류 · 도구 이름 · 성공 여부)만
  남는다. `assistant` 의 `text` 도 `thinking` 도 도구 결과도 같다. **경로에 예외가
  없다** — 봉투가 안 나오는 크래시 · 임대 만료에서도 같다. `runner.go` 가 직접
  선별하고 이 팩은 `Decode` 를 못 만진다. **시험이 넷을 잰다** —
  ① `tool_use` · `tool_result` 의 본문이 안 남는지 · ② **`assistant` 의 `text` 본문**도
  안 남는지(「도구 사건」만 거르는 구현을 잡는 줄이다) · ③ **봉투 없이 끊긴 stdout**
  으로도 같은지(「예외 없음」을 재는 줄) · ④ **껍데기와 `init` 전문과 stderr 전문이
  남는지**(통째로 버리는 구현을 잡는 줄)
- **`init` 이 나오는 경로에서 첫 줄은 언제나 `system/init` 이다.** 걷었음을 표시하는
  줄은 첫 줄이 아니다 — 게이트 CA1 · CA4 · CA5 가 `head -1` 로 읽는다. `init` 이 아예
  안 나오는 경로(플래그 오류 · 기동 실패)에서는 stderr 가 첫 줄이고 그 단계는
  `harness_error` 다 (⑯)
- **`type == "result"` 로 최종 봉투를 고른다.** `lastJSONObject` 로 뽑으면 크래시 때
  `assistant` 사건이 잡혀 ⑱ 이 닫으려던 길이 다시 열린다 (⑯ 과 같은 검사다)
- **`stream-json` 의 사건 종류를 실측해 `decisions.md` 에 행으로 적는다.** 저장소에
  그 픽스처가 0 이고 `harness.go:212-214` 가 「실물을 보고 늘린다」로 모른다고
  적는다. 허용목록이라 목록이 늘어도 기본 동작은 안 바뀌지만, **무엇을 봤는지는
  기록으로 남는다**
- **`TestAdapter_ArgvIsPure`(`claude_test.go:14-15`)가 `--output-format json` 을 완전
  일치로 잰다.** ⑮ 가 그것을 확정적으로 빨갛게 만든다 — U1 이 함께 고친다.
  이것이 SECURITY-03 을 준수로 만드는 줄이다
- **`init` 줄이 `logs/` 에 남는다** (⑮). `runctl record <id> -o r.tar && tar -xf r.tar`
  로 푼 `run-<id>/logs/NN-<단계>.log` 의 첫 줄이
  `system/init` 이고 거기 `mcp_servers` 와 `slash_commands` 가 있다. **이것이
  CA1 · CA4 · CA5 를 복제본이 아니라 실물로 재게 하는 줄이다** — 그 셋이
  이 유닛 뒤에야 집행 가능해진다
- **CA1 이 초록이다** — 개인 MCP 서버와 계정 커넥터가 있는 기계에서 아무것도
  요청하지 않은 단계를 돌려 `init` 줄의 `mcp_servers` 가 비어 있다.
  **OAuth 노드와 게이트웨이 노드 둘 다에서** `Not logged in` 없이 돈다
  (인증 경로가 둘이라 한 번으로 안 끝난다 — `application-design-plan.md` 1.4)
- **CA1 이 「로그인된 가짜 홈에서도」를 잰다** (4.6). 팩의 실측은 로그아웃 홈에서
  커넥터가 끊긴 것을 봤고, 이 설계는 `.credentials.json` 을 복사해 계정을 되돌린다.
  확실히 끊는 겹은 `--strict-mcp-config` 하나이므로 그 상태에서 재야 참이다
- **집행자는 U1 을 구현하지 않은 사람이다** (`scene-gates.md` 2절 머리). 한 손이
  전부 구현하면 자격자가 0 이 된다 — `unit-of-work-dependency.md` 8절이 그 배정을 진다
- CA0 가 초록이다. `internal/enode` 커버리지 80% 를 넘는다
- 하네스 실행파일 없이 도는 시험으로 채운다 — `Fixed(dir)` 는 문자열 비교,
  `Instrument` 는 `t.TempDir()` 에 쓰고 난 파일을 읽는다 (`application-design.md` 6.2)

**어디서 왔나** — `features.md` 3.1 · 3.2 · `requirements.md` 2.2 · Q1 = A · Q2 = A ·
`decisions.md` 6절 ④ · `scene-gates.md` CA1.

**선행** — 없다. CA1 이 아무것도 요청하지 않은 단계를 재므로 계약 어휘 없이 선다.

---

## 2. U2 — `contract-vocab`

**책임** — 계약이 `agent.mcp` · `agent.pack` 을 적을 수 있게 한다. **빌드 시점
의존의 뿌리다** — 이 유닛 전에는 그 키를 적은 계약이 `400` 이다.

```text
   internal/contract/contract.go       agentKeys 에 mcp · pack
   internal/contract/grammar.go        agent.mcp · agent.pack 을 계획에게 가르치는 줄
   internal/contract/examples/mcp.json 새 예시 하나
   internal/enode/agent.go             AgentParams.MCP · .Pack.
                                       parseAgentParams 가 받고 타입을 검증한다
```

**`internal/contract` 를 만지는 유일한 유닛이다** (계획 Q2 = A). 그 패키지의
접점이 0 이고, 계약 어휘가 흔들리면 되돌릴 자리가 한 곳이다.

**완료 조건**

- `agent` 에 모르는 키를 적은 계약이 `400` 이다 (오늘 그대로 — 검증기가 이미 있다)
- **타입이 틀린 값이 노드에서 안 죽는다.** `agentKeys` 는 키 이름만 보므로
  `agent.mcp` 를 문자열로 적으면 `400` 이 아니라 노드 위 `json.Unmarshal` 에서
  죽는다. `parseAgentParams` 가 그것을 문구로 낸다 —
  `agent.mcp must be an array of server names` · `agent.pack must be a blob name`
- `runctl example mcp` 가 나오고 `runctl lint` 가 경고 0 으로 통과한다
- **닿는 시험 둘이 초록이다** — `internal/contract/example_test.go` 와
  `cmd/runctl/shape_test.go` 의 `TestExamples_LintClean`. 둘 다
  `ExampleNames()` 를 돌므로 새 예시가 저절로 걸린다
- `grammar_test.go` 가 새 문장마다 「어긴 계약이 실제로 거절되는가」를 잰다
- CA0 가 초록이다

**어디서 왔나** — `features.md` 3.5 · `components.md` 1.6 · `component-methods.md` 7절 ·
`scene-gates.md` CA3 의 뒤 두 줄.

**선행** — 없다. U1 과 독립이다 (다른 패키지).

**`cmd/runctl` 의 소스 diff 는 0 이다.** `runctl example` 이 임베드 FS 를 읽고
`runctl schema steps` 는 구조체에서 뽑는다. 시험만 닿는다 — 행렬 3절.

---

## 3. U3 — `advert`

**책임** — 노드에 묶인 것만 `mcp.<이름>` 으로 광고한다. 뜨나를 판정하되
**프로세스를 안 띄운다.**

```text
   internal/enode/config.go   Local.MCP map[string]MCPServer · SampleLocal 주석 한 줄.
                              mcp: 절의 값 검증 — 모양이 틀리면 노드가 안 뜬다
   internal/enode/mcp.go      mcpUp (stdio 는 PATH · remote 는 환경변수 이름) ·
                              mcpFP (Fingerprinter 의 한 종류.  옛 이름 mcpAttrs)
   internal/enode/detect.go   Fingerprinter 인터페이스와 순회.  harnessFP · repoFP · mcpFP.
                              Probe 가 logger 를 받는다 — 서버별 누락 사유(FR-3)와
                              오늘의 log.Warn(ADR-059)이 갈 자리다.
                              repoFP 는 WorkspaceID fallback(ADR-036)을 그대로 옮긴다 —
                              git 없는 노드에서 repo 속성이 사라지면 동작 중립이 깨진다.
                              break 가 순회로 저절로 걷힌다.
                              harness.<이름> 과 옛 harness 를 같이 싣는다
```

**`costlyAttrs` 를 탐지기 순회로 바꾼다** (Q3 = B · `decisions.md` 6절 ⑤).
`ADR-035` §4.2 의 정본 결정이고 `ADR-035:246-248` 이 「`ADR-034` 를 구현하면서
바꾸는 것까지는 오늘 동작을 안 바꾸고 된다」로 이 시점을 지목했다. 이름은
`Fingerprinter` 다 — `Detector` 는 `ADR-068` 의 시계가 이미 쓴다.

**`Detector`(시계)와 `cheapAttrs` 는 안 건드린다.** 광고 루프가 탐지에서 멈출 수
없다는 값은 `ADR-068` 이 이미 세웠고 순회는 그 갈래를 안 건드린다.

**완료 조건**

- **CA2 가 초록이다** — `enode.yaml` 의 `mcp:` 하나가 `GET /v1/nodes` 의 `attrs` 에
  `mcp.<이름>` 으로 나오고 `harness.claude` 와 옛 `harness` 가 함께 있다.
  실행파일을 치우면 **탐지 주기(5분) 뒤** 빠지고 노드 로그에 사유가 있다
- **CA2 의 눈 검증 S1** — 현황판 노드 카드와 제어판 「탐지 능력」 카드에
  `mcp.<이름>` 칩이 보인다. **보류로 안 넘긴다** (`scene-gates.md` 4절)
- **CA3 이 완결된다** — `requires` 에 `mcp.<이름>` 을 적은 계약이 그 노드에만 가고,
  없는 키는 `422` 다. U2 가 닫은 `400` · lint 와 합쳐 CA3 한 조각이 된다
- `cheapAttrs` · `capabilities` · `Detector`(시계)에 diff 가 0 이다
- **순회가 동작 중립이다.** 기존 탐지 시험이 그대로 초록이다. 그것이 이 리팩터의
  안전망이고, 안 그러면 `ADR-035:246-248` 의 전제가 깨진 것이다
- **`runctl capabilities` 에 `mcp.<이름>` 이 나온다** (`decisions.md` 6절 ⑬ · US-4).
  계약 작성자가 보는 면이 그것이다 — `GET /v1/nodes` 는 운영자 면이다. 코드는 안 는다
  (attrs 집계가 자동으로 싣는다). **게이트가 그것을 본다는 것이 이 줄의 값이다**
- **`enode.yaml` 의 `mcp:` 가 검증된다** (SECURITY-05 · US-2). 모양이 틀리면 노드가 안 뜨고
  사유를 로그에 낸다. `decisions.md` 2절이 「그 밖의 키는 그대로 허용목록에 옮긴다」로
  미지의 키 통과를 값으로 박았으므로, **검증하는 것은 아는 키의 모양**이다
- CA0 가 초록이다

**옛 `harness` 키를 언제 걷는지는 이 유닛이 안 정한다.** 걷는 날 매처의 attrCount 가
노드마다 1 씩 줄어 정렬이 움직인다 (`application-design.md` 6.4).

**어디서 왔나** — `features.md` 3.3 · `requirements.md` 4.4 · `ADR-012` · `ADR-035` §4.2 ·
`scene-gates.md` CA2 · CA3 · 2.1.

**선행** — U1 (CA2 의 「먼저 서는 기능」이 3.1) · U2 (CA3 의 lint 와 400).

---

## 4. U4 — `sources`

**책임** — 이 단계가 무엇을 열지 **실행 직전에 한 번 정한다.** 파일을 하나도
안 만진다. 요청한 이름이 어디에도 없으면 하네스를 안 띄운다.

```text
   internal/enode/mcp.go      resolveComponents 가 출처 둘을 합친다 —
                              노드 선언 중 요청된 것 · 워크스페이스 .mcp.json 중
                              요청된 것.  겹치면 노드가 이긴다.  없으면 거절한다.
                              겹침과 빠짐을 Notes 에 남긴다.
                              요청 필터를 여기서 세운다 — U5 의 팩 출처가 그것을 탄다
   internal/enode/runner.go   Job.NodeMCP 필드
   internal/enode/claim.go    Job 리터럴에 NodeMCP: w.Local.MCP 한 줄
```

**`claim.go` 가 여기 있는 이유** — `Job{...}` 리터럴은 제품 코드에 `claim.go:765`
한 자리뿐이고, 거기서 안 실으면 `Job.NodeMCP` 가 언제나 nil 이라 **노드 선언이
조용히 안 실린다.** 설계의 요약 표에 없던 줄이다 (계획 1.1).

**완료 조건**

- **CA4 가 초록이다** — 노드가 둘을 선언하고 워크스페이스 `.mcp.json` 에 하나가
  더 있을 때, 하나만 요청하면 `mcp_servers` 에 그 하나뿐이다. 워크스페이스의
  것을 요청하면 나타난다. **없는 이름을 요청하면 하네스가 안 뜨고** 단계 error 가
  `mcp server nope is not available on this node` 를 담는다
- 그 문구가 `HarnessResult{Reason: ReasonError}` 를 타고 `claim.go` 의
  `res.Error` 안에 부분 문자열로 나온다. **새 Reason 을 안 만든다**
- 워크스페이스 `.mcp.json` 이 **광고에는 안 실린다** (`ADR-035` §4.1). `project`
  설정 소스를 안 켠다 (실측 F2)
- `resolveComponents` 가 파일도 프로세스도 안 쓰는 시험으로 덮인다 — `Job` 하나를
  넣고 오류를 잰다
- CA0 가 초록이다

**어디서 왔나** — `features.md` 3.2 · 3.4 · `requirements.md` 8절 D3 · Q4 = A ·
`scene-gates.md` CA4.

**선행** — U1 (허용목록을 쓰는 자리가 서 있어야 한다) · U2 (`agent.mcp` 가 `400` 이
아니어야 한다) · U3 (`Local.MCP` 가 있어야 한다).

---

## 5. U5 — `pack`

**책임** — 부칠 수 있는 것을 싣고, **무엇을 실었는지 봉인에 남긴다.** 팩 tar 는
쓰기 전에 검증한다 — 거부가 파일을 남기기 전에 일어난다.

```text
   internal/enode/mcp.go        Pack · PackFile · PackLimits · readPack.
                                resolveComponents 에 팩 출처를 더한다.  팩도
                                agent.mcp 필터를 탄다 (application-design.md 4.4).
                                노드 선언 이름을 덮으면 거절한다 (4.5)
   internal/enode/claude.go     Instrument 가 <dir>/home/skills/ · agents/ 를 편다
   internal/enode/harness.go    HarnessResult.MCP · .Pack
   internal/enode/runner.go     ⑧ 에서 두 필드를 채운다
   cmd/iapadapter/config.go     ExecutorConfig.Pack · PackConfig{Fetch, Name}
   cmd/iapadapter/contract.go   BuildContract 가 팩 단계와 agent.pack · in.from 을 더한다
```

**기록(3.7)이 여기 있는 이유** (계획 Q3 = A) — `HarnessResult` 를 한 번만 만진다.
두 필드의 값이 갈리는 곳(허용목록 · 팩)은 둘이지만 **쓰는 쪽이 `Record` 봉인
하나**이고 그것은 유닛 밖이다. CA6 이 둘을 한 자리에서 판정한다.

**완료 조건**

- **CA5 가 초록이다** — 팩 단계가 `$OUT/pack` 에 tar 를 내고, 다음 단계의 `init`
  줄에 팩의 스킬이 `slash_commands` 로 · 팩의 서버가 `mcp_servers` 로 있다
- **워크스페이스 `.claude/skills/` 가 스스로 읽히는지**를 같은 줄로 판정해
  `decisions.md` 에 적는다 (열린 미정 하나가 여기서 닫힌다)
- **SEC-A 가 코드로 있다** — 절대경로 · `..` · 심볼릭 링크 · 크기 상한 · 개수 상한을
  거부하고 **파일을 쓰기 전에** 한다. `settings.json` 은 이름으로 건너뛴다.
  거부는 조용하지 않다 — 그 단계를 실패로 보고한다
- 상한의 **실제 값**을 이 유닛의 Functional Design 이 닫는다 (`PackLimits`).
  **전송은 이미 막혀 있다** — Mediator 의 `MaxBlobBytes` 기본 10 MiB 가 tar 를
  막는다. 안 막힌 것은 **푼 뒤의 크기**(tar bomb)이고 `MaxBytes` 가 그것을 진다
- **팩이 노드 선언 이름을 덮으면 거절한다** (4.5). 문구는
  `pack redefines node-declared mcp server <이름>` 이고 CA5 가 그것을 찾는다
- **팩이 실었으나 요청 안 한 서버는 0 이다** (4.4). 팩의 `mcp.json` 에 있어도
  `agent.mcp` 가 이름을 안 적으면 안 실린다. Notes 에 남는다
- `readPack` 이 `io.Reader` 하나로 덮인다 — 악성 tar 를 메모리에서 지어 넣는다
- **CA6 이 초록이다** — 사내 함대에서 `scene-gates.md` 1절 ① ~ ⑦ 을 끝까지.
  `runctl record` 로 푼 `steps/NN-*.json` 에 `harness.mcp` 와 `harness.pack` 이 있다
- 새 전송 0 · 새 라우트 0. Mediator 라우트 수가 그대로다 (세는 법은 6절 · 오늘 26)
- CA0 가 초록이다

**어디서 왔나** — `features.md` 3.6 · 3.7 · `ADR-034` §2.2 · `ADR-005` 성질 4 ·
`scene-gates.md` CA5 · CA6.

**선행** — U1 (가짜 홈 아래 편다) · U2 (`agent.pack`) · U4 (`resolveComponents` 가
서 있어야 팩 출처를 더한다).

---

## 6. 유닛마다 CA0 를 도는 이유

계획 Q5 = A 다. `constraints.md` 의 차단 게이트 다섯 중 하나가 **패키지별 커버리지
80%** 이고 `internal/enode` 가 거기 걸린다.

**이 팩은 그 패키지에 새 코드를 다섯 유닛에 걸쳐 붓는다.** 마지막에만 재면 어느
유닛이 하한을 깼는지 모르는 채로 다섯 유닛치 코드를 받는다.

```text
   유닛마다 도는 것   scripts/testdb.sh 뒤 기존 테스트 전부 · 커버리지 ·
                     허용목록 밖의 스킵 0 · glyphscan · U+2605 0 ·
                     포맷 · vet · 크로스 빌드 · 심볼 상한 · 워킹트리 청결 ·
                     Mediator 라우트 수
   비용              CA0 이 여섯 번 돈다 (착수 전 한 번 + 유닛 다섯)
```

**라우트 수를 세는 명령이 부족하다.** `grep -c 'mux.HandleFunc' internal/api/api.go`
는 17 을 내지만 실제 라우트는 **26** 이다 — `api.go` 가 `mux.Handle` 로 셋을 더 걸고
`internal/api/demo_gallery.go` 가 여섯을 더 등록한다. 「라우트 0 개 는다」를 집행하려면
`internal/api/*.go` 전체에서 `mux.HandleFunc` 와 `mux.Handle(` 을 함께 세야 한다.

```text
   세는 명령   grep -rho 'mux\.HandleFunc\|mux\.Handle(' internal/api/*.go | wc -l
   오늘 값     26   (api.go 20 · demo_gallery.go 6)
```

이 팩은 `internal/api` 를 안 만지므로 오늘은 안 물리지만, **세는 명령을 그 값으로
굳힌다.** `scene-gates.md` 3절의 CA0 줄도 같이 고쳤다.

---

## 7. 유닛 밖으로 넘기는 것

**여기서 안 정한다.** 유닛별 Functional Design 과 Code Generation 의 몫이다.

```text
   PackLimits 의 실제 값        U5 의 Functional Design.  SEC-A 가 값을 안 줬다
   enode.yaml mcp: 절의 필드    U3 의 Functional Design.  Env 의 표현을 포함한다
   HarnessResult 직렬화         U5 의 Functional Design
   scene-gates.md 3절의 명령    유닛마다의 Code Generation 계획이 스크립트로 굳힌다
```

**`mcp.json` 의 모양은 FD 로 안 넘긴다 — 팩이 이미 닫았다.**
`decisions.md` 2절이 `{"mcpServers": {...}}` 로 값을 박았다. 하네스가 읽는 형식과
같아야 하므로 지어낼 자리가 아니다. 노드 선언의 「그 밖의 키는 그대로 허용목록에
옮긴다」도 같은 절이 닫았다 — `MCPServer` 가 아는 필드만 들면 소유자가 적은 미지의
키가 YAML 언마샬에서 조용히 사라진다. **U3 의 FD 가 그 통과 경로를 든다.**

**형식을 안 만드는 유닛은 Functional Design 을 그 자리에서 스킵한다**
(실행 계획). U2 가 그 후보다 — 계약 어휘는 `contract.go` 의 기존 구조체를 늘릴 뿐
새 형식을 안 만든다.
