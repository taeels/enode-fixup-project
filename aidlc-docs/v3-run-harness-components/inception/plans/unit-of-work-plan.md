# 유닛 분해 계획 — Units Generation

AI-DLC Units Generation 의 Part 1 이다. **유닛 분해는 여기서 낸다** —
`application-design.md` 8절이 명시로 넘겼다.

선행 맥락 — `inception/application-design/` 다섯 · 팩 다섯 ·
`inception/plans/execution-plan.md` · 공용 `aidlc-docs/inception/reverse-engineering/`.

**이 단계는 유닛 안의 설계를 안 짠다** — 그것은 유닛별 Functional Design 이다.
여기서는 경계 · 의존 · 착수 순서 · 파일 행렬까지다.

---

## 1. 착수 전 실측 — 설계의 요약 표에 없는 파일 둘

파일 행렬은 설계의 표를 옮겨 적는 일이 아니라 **세는 일**이다
(`constraints.md` 끝 절이 이 단계에 건 요구). 그래서 `application-design.md`
3절의 열두 줄이 실제 호출자와 맞는지 먼저 읽었다. **둘이 빠져 있다.**

### 1.1 `internal/enode/claim.go` — `Job` 이 조립되는 유일한 자리

`Job.NodeMCP` 는 `resolveComponents` 가 exec 전에 읽는 필드다
(`component-methods.md` 3절). 그런데 `Job{...}` 리터럴은 제품 코드에 한 자리뿐이다.

```text
   internal/enode/claim.go:765    logBytes, h := runHarness(runCtx, ha, bin, Job{
   그 밖                          시험 셋 (claude_test.go · env_test.go)
```

거기서 `Local.MCP` 를 안 실어 넘기면 `Job.NodeMCP` 가 언제나 nil 이고 **노드
선언이 허용목록에 조용히 안 실린다.** 요약 표에 `claim.go` 줄이 없다 — 행렬이
이 줄을 세운다.

### 1.2 `internal/enode/agent.go` — `AgentParams` 가 사는 자리

`components.md` 1.6 은 `agent.go` 를 적었고 요약 표만 안 적었다. 자리는 둘이다.

```text
   agent.go:36     type AgentParams struct    MCP []string · Pack string 이 는다
   agent.go:500    parseAgentParams           steps[].agent 를 풀어 담는다
```

### 1.3 `cmd/runctl` 의 0 은 맞다 — 다만 시험이 닿는다

`runctl example` 과 `runctl schema` 가 저절로 는다는 판정은 맞다. 그런데
`cmd/runctl/shape_test.go` 의 `TestExamples_LintClean` 이
`contract.ExampleNames()` 를 돌므로 **새 예시가 lint 경고를 내면 `cmd/runctl` 의
시험이 빨개진다.**

```text
   소스 diff      0      판정 그대로
   닿는 시험      예     예시 하나가 contract 와 runctl 양쪽 시험 위로 돈다
```

행렬은 이것을 「만지는 파일」이 아니라 **「닿는 시험」** 열로 적는다. 둘을 섞으면
0 이라는 값이 흐려진다.

### 1.4 치명이 밖으로 나가는 길은 이미 서 있다

`runHarness` 는 `error` 를 안 돌려준다. 치명은 `HarnessResult` 로 나간다.

```text
   runner.go        HarnessResult{Reason: ReasonError, Message: <사유>}
   claim.go:783     res.Error = "harness: " + string(h.Reason) + " " + h.Message
   CA4 가 찾는 것    mcp server nope is not available on this node   부분 문자열
```

`Completed()` 가 `harness_error` 를 거짓으로 내므로 판정 경로도 오늘 그대로다.
**새 Reason 도 새 필드도 없이 CA4 가 설 수 있다** — 유닛을 어떻게 가르든 참이다.

### 1.5 인터페이스 변경 비용이 세 자리다

`Harness` 구현체가 `claudeHarness` 하나다. `Fixed()` 의 호출자도 하나다.

```text
   harness.go:176   type Harness interface       선언
   claude.go        구현 하나
   runner.go:88     for k, v := range h.Fixed()  호출 하나
```

`Fixed(dir)` · `Instrument(..., c Components)` 로 겉면이 바뀌는 비용이 세 줄이다.
**그래서 시그니처 변경을 한 유닛에 몰 이유가 약하다** — 어느 유닛이 먼저 가도
나머지가 부딪히지 않는다.

---

## 2. 이 단계가 정하는 것 · 안 정하는 것

**정한다.**

```text
   유닛의 개수와 경계        무엇이 한 유닛인가.  무엇으로 갈랐나
   유닛마다의 완료 조건      어느 조각 게이트가 초록이면 그 유닛이 끝나나
   의존과 착수 순서          빌드 시점 의존 · 게이트의 「먼저 서는 기능」 열
   파일 행렬                유닛마다 만지는 파일과 심볼.  둘 이상이 만지는 자리는 접점
   짝 팩과의 접점 처리       runner.go 한 줄을 누가 언제 쥐나
   브랜치와 문서 루트        unit/<유닛> 과 aidlc-docs/taeels/
```

**안 정한다** — 건드리면 옮겨 적기가 된다.

```text
   설계가 닫은 겉면          시그니처 · 실패 등급 표 · 호출 순서.
                            application-design/ 다섯이 진다
   Functional Design 몫      팩 tar 의 상한 값 · mcp.json 의 필드 ·
                            enode.yaml mcp: 절의 필드 · 병합의 세부
   Code Generation 몫        scene-gates.md 3절 명령의 실제 스크립트화
```

---

## 3. 제안하는 분해 — 다섯

질문의 바탕이다. 5절의 답이 이것을 바꿀 수 있다.

### 3.1 유닛 다섯

| | 유닛 | 맡는 기능 | 닫는 게이트 | 사는 경로 |
|---|---|---|---|---|
| U1 | `isolation` | 3.1 가짜 홈 · 3.2 허용목록의 최소 | CA1 | `internal/enode` |
| U2 | `contract-vocab` | 3.5 계약 문법 | (게이트 없음) | `internal/contract` · `internal/enode/agent.go` |
| U3 | `advert` | 3.3 노드 선언과 광고 | CA2 · CA3 | `internal/enode` |
| U4 | `sources` | 3.4 워크스페이스 · 3.2 의 완성 | CA4 | `internal/enode` |
| U5 | `pack` | 3.6 팩 · 3.7 기록 | CA5 · CA6 | `internal/enode` · `cmd/iapadapter` |

### 3.2 왜 이렇게 갈랐나

```text
   U1 이 먼저다        scene-gates.md 가 CA1 을 맨 앞에 둔 것이 이 팩의 핵심이다 —
                      무엇을 열기 전에 무엇이 끊기는지를 먼저 잰다.
                      CA1 은 아무것도 요청하지 않은 단계를 재므로
                      계약 어휘 없이 설 수 있다

   U2 에 게이트가 없다  계약 어휘는 그 자체로 재는 장면이 없다.  대신 U3 의 CA3 과
                      U4 의 CA4 가 이 유닛 위에 선다 — 빌드 시점 의존의 뿌리다
                      (agentKeys 가 서기 전에는 agent.mcp 를 적은 계약이 400)

   U3 이 U4 보다 먼저   CA2 가 CA4 보다 싸다.  광고는 노드 하나로 재고
                      허용목록의 출처 셋은 워크스페이스까지 세워야 잰다

   U5 가 마지막이다     CA6 이 1절 전부를 끝까지 도는 조각이고 사내에서만 돈다.
                      기록(3.7)의 두 필드가 그 자리에서 함께 판정된다
```

### 3.3 의존

```text
   U1  isolation        선행 없음.  CA0 만 전제다
   U2  contract-vocab   선행 없음.  U1 과 독립이다 (다른 패키지)
   U3  advert           U1   (CA2 의 「먼저 서는 기능」이 3.1)
                        U2   (CA3 의 runctl example mcp · 모르는 키 400)
   U4  sources          U1   (허용목록 쓰기가 서 있어야 한다)
                        U2   (agent.mcp 를 적은 계약이 400 이 아니어야 한다)
                        U3   (Local.MCP 가 있어야 읽는다.  생성이 더한 줄이다)
   U5  pack             U1   (가짜 홈 아래 펴므로)
                        U2   (agent.pack)
                        U4   (resolveComponents 가 서야 팩 출처를 더한다.  생성이 더했다)
```

**한 손(taeels)이 직렬로 돈다** (Q2 = B). 병렬 가능성은 U1 과 U2 사이에만 있고
직렬이므로 쓰이지 않는다. 착수 순서 — U1 · U2 · U3 · U4 · U5.

### 3.4 유닛 둘 이상이 만지는 파일 — 미리 센 것

행렬이 이것을 표로 낸다. 여기서는 **접점이 몇 자리인지**만 적는다.

```text
   internal/enode/mcp.go      U1 (타입 · 빈 해소) · U3 (mcpUp · mcpFP) ·
                              U4 (출처 셋 · 거절) · U5 (readPack · Pack)
   internal/enode/claude.go   U1 (가짜 홈 · 빈 허용목록 · 플래그) · U5 (팩 펴기)
   internal/enode/runner.go   U1 (순서 · 실패 등급) · U4 (해소 호출) · U5 (기록 채움)
   internal/enode/harness.go  U1 (시그니처 · errAux) · U5 (HarnessResult 필드 둘)
   internal/enode/agent.go    U2 (AgentParams 필드 둘)
   internal/enode/claim.go    U4 (Job.NodeMCP 를 싣는 줄)
```

**직렬이라 이 여섯은 충돌이 아니다** — 앞 유닛이 병합된 뒤 다음 유닛이 그 위에서
돈다. 진짜 접점은 짝 팩과의 `runner.go` 하나다.

---

## 4. 산출물 계획 (체크박스)

규칙의 필수 셋에 **파일 행렬**을 더한 넷이다 (`constraints.md` 끝 절).
**5절 답이 들어온 뒤 생성한다.**

- [x] `application-design/unit-of-work.md` — 유닛 정의와 책임
  - [x] 유닛마다 — 책임 · 완료 조건 · 어느 요구 절에서 왔나 · 의존 방향
  - [x] 완료 조건에 **그 유닛이 초록으로 만드는 조각 게이트**를 이름으로 박는다
  - [x] 차단 게이트 다섯을 유닛마다의 완료 조건에 건다 (커버리지 80% 포함)
- [x] `application-design/unit-of-work-dependency.md` — 의존 행렬
  - [x] 유닛 사이의 의존 표와 그림 · 텍스트 대안
  - [x] 빌드 시점 의존 하나(`agentKeys`)를 명시로 적는다
  - [x] 착수 순서와 그 근거 — `scene-gates.md` 2절의 「먼저 서는 기능」 열
- [x] `application-design/unit-of-work-story-map.md` — 사상
  - [x] User Stories 를 건너뛰었으므로 **기능 3.1 ~ 3.7 과 조각 게이트 CA0 ~ CA6**
        을 유닛에 사상한다 (실행 계획 3절의 근거)
  - [x] 사상되지 않은 기능이 0 · 담당 없는 게이트가 0 임을 표로 확인
- [x] `application-design/unit-of-work-file-matrix.md` — 파일 행렬 (필수)
  - [x] 유닛마다 만지는 파일과 **심볼**(함수 · 타입 · 필드)
  - [x] 둘 이상이 만지는 파일을 접점으로 적고 처리를 정한다
  - [x] **닿는 시험** 열을 따로 둔다 (1.3 의 `cmd/runctl`)
  - [x] 짝 팩(transcript)과 겹치는 `runner.go` 한 줄의 병합 처리
  - [x] 1.1 · 1.2 의 파일 둘(`claim.go` · `agent.go`)이 행렬에 있음을 확인
- [x] 검증 — 구조 불변식(새 패키지 0 · 임포트 금지 넷 · 라우트 0) 과 대조 ·
      제외 여덟 범주와 대조 · 표기(`emphasis-check.py`)

---

## 5. 결정이 필요한 것 — `[Answer]:` 태그

**다섯이다.** 각 질문에 권장(A)과 근거를 붙였다. `[Answer]:` 뒤에 고른 기호를
적어 주세요. 직접 서술해도 된다.

---

### Q1. 유닛을 무엇으로 가르나

3절의 제안은 **기능과 조각 게이트를 함께** 본 것이다. 다른 가름도 선다.

```text
   A (권장)  다섯 — 3절 그대로.  기능을 묶되 조각 게이트가 유닛의 끝을 정한다
             U1 isolation · U2 contract-vocab · U3 advert · U4 sources · U5 pack

   B         여섯 — 조각 게이트마다 하나 (CA1 ~ CA6).
             기능 3.7 기록이 CA6 과 함께 독립 유닛이 된다

   C         셋 — 패키지마다 하나 (internal/enode · internal/contract · cmd/iapadapter).
             파일 행렬이 거의 비고 접점이 0 이 된다

   D         일곱 — 기능 3.1 ~ 3.7 마다 하나
```

근거 — **A 가 「끝났다」를 실행으로 판정할 수 있는 가장 거친 가름이다.**

```text
   C 는 유닛 하나가 너무 크다   internal/enode 유닛이 기능 여섯을 지고 게이트 다섯을
                             한 번에 진다.  중간에 멈출 자리가 없어 게이트가
                             착수 조건으로 안 쓰인다 (scene-gates.md 4절의 전제가 깨진다)

   D 는 게이트가 안 닫힌다      3.2 허용목록만으로는 CA1 도 CA4 도 안 초록이다.
                             CA1 은 3.1 과 함께, CA4 는 3.4 와 함께여야 닫힌다.
                             유닛이 끝나도 잴 것이 없는 유닛이 넷 생긴다

   B 는 A 와 거의 같다         갈리는 것은 기록(3.7) 하나뿐이다 — Q3 이 그것만 따로 묻는다
```

**[Answer]:** A

---

### Q2. 계약 어휘(3.5)를 독립 유닛으로 먼저 세우나

`agentKeys` 가 서기 전에는 `agent.mcp` · `agent.pack` 을 적은 계약이 `400` 이다
(`application-design.md` 8절의 빌드 시점 의존). 그 뿌리를 어디에 두나.

```text
   A (권장)  독립 유닛 U2 로 세운다.  게이트가 없는 유닛이지만 U3 · U4 · U5 가
             그 위에 선다.  internal/contract 를 만지는 유일한 유닛이 된다

   B         U1 isolation 에 붙인다.  유닛이 넷으로 준다.
             다만 U1 이 두 패키지를 지고 CA1 과 무관한 파일을 만진다

   C         그것을 처음 쓰는 유닛(U4 sources)에 붙인다.
             U3 advert 의 CA3 이 U4 뒤로 밀린다
```

근거 — **A 가 행렬을 깨끗하게 가른다.** `internal/contract` 를 만지는 유닛이
하나뿐이면 그 패키지에 접점이 0 이고, 계약 어휘가 바뀔 때 어느 유닛을 되돌려야
하는지가 한 자리다. 게이트가 없는 것은 약점이 아니라 사실이다 — 계약 어휘는
그 자체로 재는 장면이 없고, `runctl lint` 와 예시 시험이 그 자리의 초록이다.
B 는 U1 의 완료 조건에 CA1 과 무관한 줄을 더한다. C 는 CA3 을 CA4 뒤로 미는데,
`scene-gates.md` 2절이 CA3 을 CA4 앞에 둔 순서와 어긋난다.

**[Answer]:** A

---

### Q3. 기록(3.7)을 어디에 두나

`HarnessResult` 의 두 필드(`MCP` · `Pack`)와 그것을 채우는 `runner.go` 의 ⑧ 이다.
값이 작고 **값의 출처가 둘**이다 — `MCP` 는 허용목록에서, `Pack` 은 팩에서 온다.

```text
   A (권장)  U5 pack 에 붙인다.  둘 다 CA6 이 한 자리에서 판정한다

   B         독립 유닛으로 둔다 (여섯 번째).  CA6 과 짝이 된다

   C         값이 생기는 유닛마다 나눈다 — MCP 는 U4 sources 에, Pack 은 U5 pack 에
```

근거 — **A 가 `HarnessResult` 를 한 번만 만진다.** C 는 같은 구조체를 두 유닛이
만지고 직렬 병합 자리를 하나 더 만드는데, 얻는 것은 「필드가 쓰이는 유닛에서
난다」뿐이다. 그 값은 여기서 약하다 — 두 필드 다 쓰는 쪽이 `Record` 봉인이고
그것은 유닛 밖이다. B 는 유닛 하나가 필드 둘과 채움 한 줄만 진다 — 조각 게이트가
착수 조건으로 도는 값에 비해 유닛의 무게가 너무 가볍다.

**C 를 고르면** `MCP` 필드가 U4 에서 나므로 CA4 에 「단계 결과에 실린 서버 이름을
본다」가 더해진다 — 게이트가 하나 두꺼워진다.

**[Answer]:** A

---

### Q4. 유닛이 어느 브랜치에서 돌고 어디로 병합되나

`CONVENTIONS.md` 3.1 은 유닛 브랜치가 `unit/<유닛>` 이고 **v1-run 으로 모인다**고
적었다. `CLAUDE.md` 의 Construction 절은 담당이 **자기 브랜치에서 PR 로 `main` 에
병합**한다고 적었다. 이 회차는 한 손이고 회차 브랜치가 `v3-run-harness-components`
다 — 두 문장이 가리키는 자리가 다르다.

```text
   A (권장)  unit/<유닛> 을 회차 브랜치에서 따고 게이트가 초록이면 PR 로 main 에 올린다.
             회차 브랜치도 Inception 이 닫히면 PR 로 올린다.
             규칙 정리(5cbf171) 뒤의 CONVENTIONS 3.1 · 3.3 그대로다

   B         unit/<유닛> 에서 돌고 유닛마다 PR 로 main 에 병합.
             CLAUDE.md 의 Construction 문장 그대로다

   C         브랜치를 안 딴다.  v3-run-harness-components 위에서 직렬로 커밋만 한다.
             한 손이라 부딪힐 상대가 없다
```

근거 — **A 가 짝 팩과의 병합 순서를 지킨다.** 질문 3 의 답(A)이 「이 팩 먼저 ·
짝 팩이 그 위에」를 정했고, 그 순서는 **팩 단위**로 매겨진 것이다. 짝 팩은 이 팩의
마지막 유닛이 들어간 뒤에 착수하므로, 유닛마다 `main` 에 올려도 중간 상태의
`runner.go` 위로 짝 팩이 올라오지 않는다. C 는 게이트가 빨간 채로 커밋이 쌓여도
되돌릴 경계가 없다 —
`CONVENTIONS.md` 3.3 의 「빨간 채로 병합하면 다음 사람이 서지 않는 나무를 받는다」가
막으려던 자리가 브랜치 경계 그 자체다.

**A 를 고르면** 문서 루트는 `aidlc-docs/taeels/` 다 (`CLAUDE.md` Construction 절).

**이 질문은 규칙 정리가 닫았다** (2026-09-12 · 커밋 `5cbf171`). git 이력이 답을
줬다 — `origin/v1-run-dhseo` 가 `unit/obs` · `panel` · `drain` · `transcript` 의
조상이고(유닛은 회차 브랜치에서 땄다) PR 스물여섯이 전부 `main` 으로 갔다.
**두 문서가 각각 절반씩 맞았고** 그것을 하나로 적었다. A 가 그 결론이다.

**[Answer]:** A

---

### Q5. 유닛의 완료 조건에 CA0 를 넣나

`scene-gates.md` CA0 은 「기동이 안 깨졌다」이고 기계가 돈다 — 기존 테스트 전부 ·
커버리지 · glyphscan · 포맷 · vet · 크로스 빌드 · 심볼 상한 · 워킹트리 청결 ·
라우트 수. 그것을 언제 도나.

```text
   A (권장)  유닛마다 돈다.  그 유닛의 조각 게이트 앞에 CA0 가 먼저 초록이어야
             병합한다.  착수 전 한 번 + 유닛 다섯 = 여섯 번

   B         팩 착수 전에 한 번, 마지막 유닛 뒤에 한 번.  두 번

   C         유닛마다 돌되 무거운 것(크로스 빌드 · 심볼 상한)은 마지막에만 돈다
```

근거 — **A 가 커버리지 하한을 실제로 집행한다.** 차단 게이트 다섯 중 하나가
「패키지별 커버리지 80%」이고 `internal/enode` 가 거기 걸린다
(`constraints.md`). 이 팩은 그 패키지에 새 코드를 다섯 유닛에 걸쳐 붓는다 —
마지막에만 재면 **어느 유닛이 하한을 깼는지 모르는 채로 다섯 유닛치 코드를
받는다.** B 는 그 위험을 지고 C 는 그중 싼 것만 남긴다.

**A 의 비용** — CA0 이 여섯 번 돈다. `scripts/testdb.sh` 가 필요하고 크로스
빌드가 들어 있어 한 번이 싸지 않다. 그 비용을 낼 값이 있다고 본다.

**[Answer]:** A

---

### 답 (2026-09-12)

사용자 답은 **「권장안으로 결정하고 닫는다」** 였다. Q1 ~ Q5 가 모두 A 다.
모호한 답이 없어 후속 질문을 안 냈다 (규칙 7 · 8 의 게이트 통과).

```text
   Q1 = A   유닛 다섯.  기능을 묶되 조각 게이트가 유닛의 끝을 정한다
   Q2 = A   계약 어휘를 독립 유닛 contract-vocab 으로 세운다
   Q3 = A   기록(3.7)을 pack 유닛에 붙인다
   Q4 = A   unit/<유닛> 을 회차 브랜치에서 따고 PR 로 main 에 올린다.
            규칙 정리(5cbf171)가 이 답을 먼저 세웠다
   Q5 = A   유닛마다 CA0 를 돈다.  커버리지 하한을 유닛 단위로 집행한다
```

**생성이 답 밖에서 더 정한 것 셋** — 답이 안 닫은 자리이고 산출물이 근거를 적었다.

```text
   ①  Local.MCP 는 U3 advert 가, Job.NodeMCP 와 claim.go 의 싣는 줄은
      U4 sources 가 낸다.  읽는 쪽이 없는 필드를 먼저 세우지 않는다
   ②  resolveComponents 는 U1 이 빈 경우로 세우고 U4 가 출처 둘을,
      U5 가 팩 출처를 더한다.  runner.go 의 호출 자리가 U1 에서 최종형이 된다
   ③  CA3 이 유닛 둘에 걸린다.  400 과 lint 는 U2 가, requires 매칭과 422 는
      U3 이 닫는다.  게이트를 쪼개 적는다
```

---

## 6. 질문을 안 낸 범주와 그 근거

규칙이 세라고 한 여섯 중 넷을 위에서 물었다. 나머지 둘은 **이미 닫혀 있다.**

```text
   Business Domain          팩 하나가 한 도메인이다 — 하네스 실행 층.
                            constraints.md 의 구조 불변식이 경계를 이미 못 박았다
                            (새 패키지 0 · 임포트 금지 넷 · 라우트 0).
                            유닛이 도메인을 가를 여지가 없다

   Code Organization        Greenfield 다중 유닛에만 해당한다.  이 회차는 brownfield 이고
                            새 패키지가 0 이라 디렉터리 구조를 정할 것이 없다
```

```text
   Story Grouping           Q1 · Q3 이 물었다.  User Stories 를 최소 형태로 되살렸다
                            (2026-09-12).  묶는 단위는 기능 3.1 ~ 3.7 이고
                            스토리 열이 그 위에 사상된다
   Dependencies             Q2 가 물었다 (빌드 시점 의존의 뿌리)
   Team Alignment           Q4 가 물었다 (한 손 · 브랜치와 병합 대상)
   Technical Considerations Q5 가 물었다 (차단 게이트를 언제 집행하나)
```

---

## 7. 답이 들어온 뒤의 순서

```text
   ①  답 다섯을 읽고 모호한 것이 있으면 후속 질문을 이 문서에 더한다 (규칙 7·8)
   ②  4절 체크박스대로 산출물 넷을 낸다
   ③  파일 행렬에 1.1 · 1.2 의 파일 둘이 있는지 확인한다
   ④  audit.md 에 답 원문을 그대로 싣고 aidlc-state.md 를 갱신한다
   ⑤  승인 뒤 CONSTRUCTION — 첫 유닛의 Functional Design
```
