# Requirements 확인 질문 — 하네스 구성요소 (v3-run-harness-components)

요구 팩이 값을 거의 다 갖고 있다. `decisions.md` 1 ~ 2절이 홈 변수 · 허용목록
모양 · 광고 키 · 계약 문법 · 팩 tar 규약까지 못 박았고, `features.md` 5절의
미정 둘은 **사람에게 물을 것이 아니라 게이트가 재는 것**이다 (CA1 의 OAuth 파일
복사 · CA5 의 워크스페이스 스킬). 그래서 여기서 묻는 것은 셋뿐이고, 셋 다
**팩이 대답할 수 없는 자리** — 회차 운영과 손의 배분이다.

각 질문의 `[Answer]:` 뒤에 글자를 적는다. 맞는 것이 없으면 `Other` 를 고르고
설명을 적는다. 답이 없으면 권장값으로 간다.

권장을 벗어나는 자리는 묻지 않고 `decisions.md` 에 행으로 적는다 (팩의 게이트
규칙). 이 문서 끝의 「확인된 사실」이 그 후보 하나를 이미 들고 있다.

---

## Question 1 — 공용 Reverse Engineering 을 갱신하나

공용 R/E 산출물(`aidlc-docs/inception/reverse-engineering/`)은
**2026-09-08T07:17:58Z** 것이다. 그 뒤 `cmd/` · `internal/` 에 커밋 **43** 이
들어왔다. Mediator 라우트가 **15 -> 17**, Go 파일이 **143 -> 190** 이다.
워크플로 규칙(`workspace-detection.md` Step 3)은 낡았으면 다시 돌리라고 적는다.

그런데 **움직인 곳이 이 팩의 밖이다.** 경로별로 세면 이렇다.

```text
   internal/api        커밋 32     이 팩은 라우트를 0 개 늘린다 (constraints)
   internal/enode      커밋  4     이 팩의 주무대
   internal/contract   커밋  4     이 팩이 키 둘을 더한다
   cmd/iapadapter      커밋  0
   cmd/runctl          커밋  0
```

A) **그대로 쓴다** — 공용 R/E 를 갱신하지 않고 Requirements Analysis 로 간다.
이 팩이 딛는 경로는 거의 안 움직였고, 움직인 `internal/api` 는 팩이 안 건드린다.
낡은 줄을 만나면 그 자리에서 코드를 직접 읽어 대고 근거를 남긴다. (권장)

B) **이 팩이 딛는 경로만 다시 잰다** — `internal/enode` · `internal/contract` ·
`cmd/iapadapter` · `cmd/runctl` 네 경로의 `code-structure.md` ·
`component-inventory.md` 절만 갱신하고 나머지는 그대로 둔다. 회차가 한 단계 는다.

C) **전부 다시 돌린다** — R/E 여덟 문서를 새로 낸다. 가장 정확하지만 이 회차의
값이 아니라 공용 자산의 값이고, 열린 Task 가 따로 있다 (R/E 산출물 신선도 조사).

X) Other (please describe after [Answer]: tag below)

[Answer]: 

---

## Question 2 — 이 회차를 어디까지 도나, 그리고 몇 손인가

`CLAUDE.md` 가 Inception 과 Construction 의 문서 루트를 가른다 — Inception 은
회차 브랜치(`aidlc-docs/v3-run-harness-components/`), Construction 은 담당
handle(`aidlc-docs/<handle>/`)이다. 그 갈림이 **언제 일어나는가**가 이 질문이다.
팩의 기능 일곱은 `internal/enode` 에 몰려 있고 접점 표가 같은 파일 셋을 가리키므로
(`claude.go` · `runner.go` · `hook.go`), 손이 여럿이면 병합 순서가 곧 일정이 된다.

A) **Inception 을 끝까지 돌고 멈춘다** — Requirements Analysis · Workflow
Planning · (필요하면) Application Design · Units Generation 까지 내고, 유닛
분해와 파일 행렬을 본 뒤에 담당을 배정한다. Construction 착수는 별도 승인이다.
(권장 — 팩이 유닛 분해를 안 주므로 배정할 대상이 아직 없다)

B) **Inception 을 돌고 그대로 Construction 까지 한 손(taeels)으로 간다** —
배정 없이 유닛을 직렬로 민다. 접점 충돌이 없고 가장 빠르지만 병렬 이득이 0 이다.

C) **지금 담당을 정하고 Inception 과 Construction 을 겹친다** — 유닛 분해가
나오는 즉시 `construction-roster.md` 에 행을 더하고 각자 브랜치를 딴다.

X) Other (please describe after [Answer]: tag below)

[Answer]: 

---

## Question 3 — 짝 팩(transcript)과의 순서

`constraints.md` 의 접점 절이 두 팩이 같은 파일을 만진다고 적는다.

```text
   internal/enode/claude.go   Argv        이 팩은 MCP 플래그를 더한다
                                          transcript 팩은 출력 형식을 바꾼다
                              Decode      transcript 팩만
   internal/enode/runner.go   Job         이 팩은 팩 · MCP 를 싣는다
                                          transcript 팩은 tee 와 사건 배출을 만진다
```

그리고 **이 팩의 게이트가 짝 팩에 기댄다.** `scene-gates.md` 3절이 적은 그대로다 —
`init` 줄을 읽는 정식 경로는 `logs/` 이고, transcript 팩이 먼저 병합돼 있으면
그것이 있다. 아니면 사람이 같은 환경과 플래그로 `claude` 를 한 번 직접 띄워야 한다.
`v3-run-transcript` 브랜치는 지금 팩 커밋 하나뿐이고 AI-DLC 산출물이 0 이다.

A) **이 팩을 먼저 끝내고 transcript 를 뒤에** — 게이트는 사람이 직접 띄우는
경로(`scene-gates.md` 3절의 `claude -p --output-format stream-json`)로 잰다.
`claude.go` · `runner.go` 의 접점은 transcript 가 나중에 받아 푼다. (권장 —
사내 실측이 막고 있는 것이 이 팩이고, 직접 띄우기가 이미 게이트에 적혀 있다)

B) **transcript 를 먼저 돌리고 이 팩을 뒤에** — `logs/` 에 `init` 줄이 있는
상태에서 이 팩의 게이트를 잰다. 게이트가 편해지지만 사내에서 막힌 것이 그만큼 늦다.

C) **둘을 병렬로 돌리고 진행자가 직렬로 병합한다** — 손이 둘 이상일 때만 뜻이
있다. Question 2 의 답과 묶인다.

X) Other (please describe after [Answer]: tag below)

[Answer]: 

---

# 확인된 사실 — 팩의 코드 상태 기술 하나가 낡았다

착수 전에 팩이 적어둔 세는 명령을 그대로 돌렸다. 셋은 맞고 하나가 틀렸다.

```text
   맞다   grep -rn 'strict-mcp-config|CLAUDE_CONFIG_DIR|"pack"' internal cmd  ->  0 줄
   맞다   detect.go 의 하네스 순회가 attrs["harness"] = h.Name() 뒤 break 한다
   맞다   claude.go Fixed() 는 CLAUDE_CODE_DISABLE_AUTO_MEMORY 하나뿐이고
          hook.go 가 --settings 와 --setting-sources "" 를 낸다
   틀리다  features.md 1.3 의 「없다」 목록 마지막 줄 — AgentParams 의 알려진 키 검증
```

`agent` 맵의 알려진 키 검증은 **이미 있다.** `internal/contract/contract.go:902`
가 `agentKeys = []string{"model", "max_turns", "max_tokens", "ask", "harness"}`
를 들고 있고, 같은 파일의 `knownKeys` 가 모르는 키를 거절한다. `ADR-057` 이
그 자리를 정본으로 적어 두었고 코드가 따라와 있다.

그래서 `features.md` 3.5 의 그 항목은 **새로 만드는 일이 아니라 목록에 이름 둘을
더하는 일**이다 — `mcp` · `pack`. 400 은 이미 난다. 바뀌는 범위가 줄어드는
방향이므로 묻지 않고 `decisions.md` 에 행으로 적는다 (팩의 게이트 규칙).

이 교정은 `canon.md` 2절의 `ADR-034` §5 줄에도 걸린다 — ④(알려진 키)가 「이 팩」이
아니라 「이미 있다」로 바뀐다.
