# U4 `sources` — Functional Design 계획

**유닛** `sources` · **브랜치** `unit/sources` · **담당** taeels ·
**닫는 게이트** CA4 (+ CA0) · **선행** U1 `isolation` · U2 `contract-vocab` ·
U3 `advert`

정본 입력은 `aidlc-docs/v3-run-harness-components/inception/application-design/`
의 넷과 `requirements/harness-components/` 팩이다. 이 계획은 그 위에서
**출처를 합치는 규칙과 허용목록 항목의 최종 꼴**만 짓는다 — 코드는 다음 단계다.

---

## 0. 이 단계가 닫는 것과 안 닫는 것

```text
   닫는다    resolveComponents 가 출처 둘을 합치는 규칙 — 요청 필터 · 우선순위 ·
             겹침과 빠짐
             허용목록 항목의 최종 꼴.  어디까지가 우리 어휘이고 어디부터
             하네스의 원문인가
             워크스페이스 .mcp.json 을 읽는 자리 · 시점 · 못 읽을 때
             요청한 이름이 어디에도 없을 때의 거절 문구와 그것이 타는 길
             Notes 가 무엇을 담고 어디로 나가나

   안 닫는다  팩 출처 · tar 검증 · 상한             U5
             HarnessResult 의 mcp · pack 기록       U5
             광고와 노드 선언의 형식                 U3 가 닫았다
             agent.mcp · agent.pack 의 어휘         U2 가 닫았다
             가짜 홈 · 자격증명 복사 · 플래그         U1 이 닫았다
             시험 목록과 파일별 diff                 이 유닛의 Code Generation
```

---

## 1. 실측 — 코드와 하네스가 지금 어떻게 생겼나 (2026-09-13)

FD 를 짓기 전에 이 유닛이 만지는 자리를 읽고 `claude 2.1.266` 으로 돌렸다.
**허용목록에 실을 것의 어휘가 출처마다 다르고**, 그 다름을 오늘 형식이 못
받는다. 그리고 **오늘 코드가 내는 원격 항목은 하네스가 말없이 버린다.**

### 1.1 워크스페이스 `.mcp.json` 의 실제 모양

지어내지 않고 하네스가 스스로 쓴 것을 읽었다 — 스크래치에서
`claude mcp add --scope project` 로 원격 하나와 stdio 하나를 넣었다.

```json
{"mcpServers": {
  "corridor": {"type": "http", "url": "https://app.example/mcp",
               "headers": {"Authorization": "Bearer x"}},
  "stdioone": {"type": "stdio", "command": "/usr/bin/true", "args": ["--flag"],
               "env": {"API_KEY": "xxx", "REF": "${HOST_TOKEN}"}}
}}
```

**셋이 우리 어휘에 없다.**

```text
   type       우리 MCPServer 에 없는 필드.  하네스가 종류를 여기서 읽는다
   headers    없는 필드.  원격 인증이 여기로 간다
   env 의 값   값이다.  U3 가 노드 선언에 못 박은 「이름에서 이름으로」가 아니다
```

이 파일은 **저장소가 하네스에게 쓴 것**이고 우리가 정의한 형식이 아니다.
옮겨 적는 쪽이 뜻을 바꾸면 그것은 번역이 아니라 손상이다.

### 1.2 `MCPServer` 로 받으면 셋을 잃거나 망친다

같은 JSON 을 `MCPServer` 로 풀고 `allowlistEntry()` 를 거쳐 다시 냈다.

```text
   잃는다    type 과 headers 가 사라진다.  Extra 는 UnmarshalYAML 이 채우고
             JSON 경로에는 그 함수가 없다 — 모르는 키가 말없이 버려진다
             {"type":"http","url":...,"headers":{...}} -> {"url":...}

   망친다    envRefs 가 값을 한 번 더 감싼다
             {"API_KEY":"${HOST_TOKEN}"} -> {"API_KEY":"${${HOST_TOKEN}}"}
             {"LIT":"literal-value"}     -> {"LIT":"${literal-value}"}

   죽는다    종류 오류 하나가 파일 전체를 죽인다.  {"command":5} 서버 하나에
             json.Unmarshal 이 오류를 내고 그 뒤 항목이 안 채워진다
```

`envRefs` 도 `Extra` 도 U3 가 **노드 선언을 위해** 지은 것이라 옳다. 틀린 것은
다른 어휘로 적힌 파일을 그 형식에 밀어 넣는 일이다.

### 1.3 `url` 만 적힌 항목은 하네스가 말없이 버린다

허용목록을 손으로 지어 `--strict-mcp-config --mcp-config=` 로 넘기고 `init`
줄의 `mcp_servers` 를 읽었다. 가짜 주소와 `true` 를 썼으므로 뜨는 것은 하나도
없고, **목록에 이름이 있는가**가 판정 재료다 (`scene-gates.md` 2절 머리와 같은 법).

```text
   {"url": "https://127.0.0.1/mcp"}                      목록에 없다
   {"transport": "http", "url": "..."}                   목록에 없다
   {"type": "http", "url": "...", "headers": {...}}      failed 로 있다
   {"type": "sse", "url": "..."}                         failed 로 있다
   {"command": "/usr/bin/true"}                          failed 로 있다 (type 없이)
   {"type":"stdio","command":"...","credential":"...","wibble":42}   failed 로 있다
```

**종류를 정하는 것은 `type` 이거나 `command` 다.** `url` 만 있으면 목록에서
아예 빠진다 — 실패조차 아니다. 그리고 **모르는 키는 항목을 안 죽인다** — U3 의
「그 밖의 키는 그대로 옮긴다」가 하네스 쪽에서도 성립한다.

**이것이 정본 예시를 그대로 적은 노드 선언에 걸린다.** `ADR-035` §4.4 의
`gerrit: {url: ..., credential: GERRIT_TOKEN}` 은 오늘 `allowlistEntry` 를 거쳐
`{"url": "..."}` 가 되고 하네스가 그것을 조용히 버린다. 광고는 `mcp.gerrit` 로
서 있고 매처는 그 노드를 고르는데 서버는 안 열린다 — `ADR-035` §3 의
**「없음이 실패보다 나쁘다」** 가 가리키는 바로 그 모양이다.

### 1.4 `${}` 는 한 겹만 펴진다

stdio 서버 자리에 환경을 파일로 떨구는 스크립트를 놓고 그 파일을 읽었다.

```text
   "C": "${HOST_TOKEN}"      ->  C=secretvalue        U3 의 ㉒ ③ 이 이 기계에서도 섰다
   "A": "${${HOST_TOKEN}}"   ->  A=${secretvalue}     한 겹만 편다
   "B": "literal"            ->  B=literal
```

1.2 의 이중 감싸기가 **조용한 손상**인 이유가 이것이다 — 오류가 아니라 틀린
값이 서버에 간다.

### 1.5 부르는 자리 — `Job` 이 들고 오는 것과 안 오는 것

```text
   runner.go:94    ② c, err := resolveComponents(j).  j 밖의 입력이 0 이다
   claim.go:765    Job 리터럴은 제품 코드에 한 자리뿐이다
   claim.go:558    dir := w.Local.Workspace.  비면 os.TempDir() 이고 그것이 j.IO.Dir 다
   w.Local.MCP     노드 선언.  오늘 Job 에 안 실린다 — 행렬이 예고한 Job.NodeMCP 다
   j.Params.MCP    요청.  U2 가 배열과 빈 이름을 제출에서 이미 걸렀다
```

**로거가 없다.** `runner.go` 는 `log/slog` 를 import 하지 않는다.
`Components.Notes` 의 소비자가 오늘 0 이고, `j.Emit` 은 `claim.go:775` 에서
`log.Debug("harness event", "kind", e.Kind)` 로 **종류만** 찍는다.

### 1.6 `logs/` 는 Notes 가 갈 자리가 아니다

`selectLogs` 의 허용목록이 전문 셋(첫 `system/init` · 마지막 `result` · stderr)
이고, 게이트 CA1 · CA4 · CA5 가 그 파일을 `head -1` 로 읽는다. 우리 줄을 앞에
실으면 **게이트 셋이 함께 깨진다.** Notes 는 다른 면으로 나가야 한다 (물음 5).

### 1.7 안 만져도 되는 것을 확인했다

```text
   agent.mcp 의 모양     U2 가 닫았다.  checkComponentTypes 가 배열인가와
                        원소가 빈 문자열이 아닌가를 보고 제출에서 400 이다
   우선순위              features.md 3.2 와 decisions.md 2절이 닫았다 —
                        노드가 워크스페이스를 이긴다.  다시 안 정한다
   팩이 노드 이름을 덮을 때  decisions.md 6절 ⑦ 이 닫았고 자리는 U5 다
   요청이 0 일 때의 빈 파일   U1 이 이미 짓는다.  mcpAllowlistJSON 이 봉투를
                        언제나 쓴다 — 빈 것도 {"mcpServers":{}} 다
   광고                  워크스페이스의 것은 광고에 안 간다 (features.md 3.4).
                        U3 의 mcpFP 는 Local.MCP 만 본다 — 코드 0 이다
```

---

## 2. 물음 일곱

답을 `[Answer]:` 뒤에 적는다. 권장이 있는 물음은 권장을 **A** 에 둔다.

### Question 1

**워크스페이스 `.mcp.json` 을 누가 읽나.** U1 이 `resolveComponents` 를
「파일을 하나도 안 만진다」로 세웠고 (`mcp.go` 의 머리 주석 · `unit-of-work.md`
4절의 완료 조건), 이 유닛이 처음으로 디스크에 있는 출처를 들인다.

A) **가장자리에서 읽고 `Job` 이 들고 온다.** `claim.go` 가 `Job.NodeMCP` 옆에
`Job.WorkspaceMCP` 를 채운다 — 요청이 0 이면 파일을 아예 안 연다.
`resolveComponents` 는 파일을 안 만지고 **정하는 일만** 한다. U1 이 세운
「정한다 / 쓴다」의 가름이 그대로 살고 시험이 디스크 없이 돈다. 대가는 U5 의
팩도 같은 규율을 따라야 한다는 것이다 — 여는 것은 가장자리, 고르는 것은 여기

B) **`resolveComponents` 가 직접 읽는다.** `j.IO.Dir/.mcp.json` 을 요청이 있을
때만 연다. 출처 셋이 한 함수에 모이고 배선이 0 이다. 대가는 U1 의 주석과 완료
조건을 이 유닛이 고치는 것이다 — 시험이 `t.TempDir()` 을 쓰게 되고, 「파일도
프로세스도 안 쓰는 시험으로 덮인다」가 거짓이 된다

C) **읽는 함수를 `Job` 에 주입한다.** `Job.Workspace func() (map[string]any, error)`
같은 자리를 두어 시험은 순수하고 제품은 게으르다. 대가는 배선 하나와 nil
검사이고, `Job` 이 데이터가 아니라 동작을 들고 다니게 된다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2

**워크스페이스 항목(그리고 U5 의 팩 항목)을 어떤 꼴로 나르나.** 1.1 이 그 파일의
어휘가 우리 것과 다름을 보였고 1.2 가 오늘 형식으로 받으면 잃거나 망가짐을 보였다.

A) **원문 그대로 나른다.** 그 출처의 항목은 `map[string]any` 로 읽어 허용목록에
**한 글자도 안 고치고** 옮긴다. `Components.Servers` 의 타입을 허용목록 항목
(`map[string]map[string]any`)으로 올리고, 노드 선언은 `allowlistEntry()` 를
거쳐 같은 꼴이 되어 합쳐진다. 1.2 의 손상 셋이 구조적으로 못 생긴다 — 번역하는
코드가 없으므로. 대가는 `Components.Servers` 의 타입이 바뀌어
`instrument_test.go` · `mcp_test.go` 와 `writeMCPAllowlist` 가 함께 닿는 것이다

B) **`MCPServer` 에 `UnmarshalJSON` 을 더한다.** 모르는 키를 `Extra` 로 받고
`env` 의 뜻은 출처로 가른다 (노드 것은 이름에서 이름 · 워크스페이스 것은 그대로).
형식이 하나로 남는 것이 값이다. 대가는 같은 구조체가 두 어휘를 지는 것이다 —
`Env` 필드만 보고는 그 값이 이름인지 값인지 못 푼다

C) **워크스페이스 것도 노드 어휘로 좁힌다.** `command` · `args` · `url` 만
옮기고 나머지는 버린다. 코드가 가장 작다. 대가는 1.1 의 실제 파일 둘 중
원격 하나가 통째로 못 건너오고 `env` 가 사라지는 것이다 — 저장소가 적은 것과
노드가 물리는 것이 갈린다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3

**`url` 만 적힌 항목이 조용히 사라지는 것(1.3)을 이 유닛이 막나.** 오늘
`allowlistEntry` 가 내는 원격 항목이 그 모양이다.

A) **허용목록에 `type` 을 박는다.** 항목에 `type` 이 없으면 채운다 —
`command` 가 있으면 `stdio`, `url` 이면 `http`. 이미 있으면(`Extra` 나 워크스페이스
원문) 손대지 않는다. 조용히 사라지는 경로가 없어지고, 종류를 잘못 채운 경우는
`mcp_servers` 에 `failed` 로 **보인다** — 침묵이 아니라 실패다. 대가는 `sse`
서버를 `url` 만으로 적은 소유자가 `failed` 를 보고 `type: sse` 를 적어야 하는 것이다

B) **resolve 에서 거절한다.** `url` 만 있고 `type` 이 없으면 단계를 실패시키고
문구가 `type` 을 요구한다. 가장 닫히는 쪽이다. 대가는 `ADR-035` §4.4 의 예시를
그대로 적은 선언이 거절되는 것이다 — 정본의 예시와 코드가 갈린다

C) **안 막는다.** 실측만 `decisions.md` 에 적고 소유자가 `type` 을 적게 한다
(`Extra` 로 지나간다). 코드 0. 대가는 광고는 서고 서버는 안 열리는 조합이 그대로
남는 것이다 — 이 팩이 고치러 온 실패를 이 팩이 다시 만든다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4

**워크스페이스 `.mcp.json` 이 있는데 못 읽을 때 무엇을 하나.** 1.2 가 종류 오류
하나로 파일 전체가 죽는 것을 보였고, 그 파일은 우리가 아니라 저장소가 쓴 것이다.

A) **요청이 있을 때만 치명이다.** `agent.mcp` 가 이름을 하나라도 적었는데 파일이
있고 파싱이 안 되면 단계를 실패시키고 문구에 파싱 오류를 담는다. 요청이 0 이면
Note 만 남기고 넘어간다. 진단이 정확해지고(오타와 깨진 파일이 다른 문구다),
MCP 를 안 쓰는 단계가 남의 저장소 파일 때문에 안 죽는다

B) **언제나 치명이다.** 파일이 있는데 못 읽으면 요청과 무관하게 단계 실패다.
「실수는 닫히는 쪽으로 틀린다」(⑨)를 가장 곧이 지킨다. 대가는 `.mcp.json` 이
깨진 저장소에서 MCP 와 아무 상관 없는 에이전트 단계까지 전부 죽는 것이다

C) **언제나 비치명이다.** Note 만 남기고 빈 것으로 본다. 요청한 이름은
「not available」로 떨어진다. 대가는 깨진 파일과 오타가 같은 문구로 보이는
것이다 — 사람이 파일을 고칠 단서를 못 받는다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5

**`Components.Notes` 가 어디로 나가나.** 1.5 가 소비자 0 을, 1.6 이 `logs/` 가
그 자리가 아님을 보였다. 완료 조건은 「겹침과 빠짐을 Notes 에 남긴다」이고 그것을
**사람이 읽는 자리**가 아직 없다.

A) **`Job` 에 로거를 실어 노드 로그로 낸다.** `Job.Log *slog.Logger` 를 더하고
`claim.go` 가 이미 가진 단계 로거를 넘긴다. `runHarness` 가 resolve 뒤에 Note
마다 한 줄을 `Info` 로 찍는다 (nil 이면 안 찍는다). 노드 소유자와 게이트 집행자가
보는 자리가 그것이고, U3 가 `Probe(logger)` 로 이미 쓴 결이다. 대가는 `Job` 이
필드 하나 더 느는 것이다

B) **`j.Emit` 으로 낸다.** 배선이 이미 있다. 대가는 `claim.go:775` 가 `Kind` 만
`Debug` 로 찍어 사실상 안 보이는 것이고, 보이게 하려면 그 줄도 고쳐야 한다.
그리고 `Event` 는 하네스 사건의 어휘라 우리 사실을 싣는 자리가 아니다

C) **이번 회차엔 안 낸다.** `Notes` 를 채우기만 하고 U5 의
`HarnessResult.MCP` 가 봉인에 실을 때 함께 보이게 한다. 코드 0. 대가는 겹침과
빠짐이 단계가 끝나야 보이고, 단계가 exec 앞에서 죽은 경우에는 어디에도 안 남는 것이다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6

**요청한 이름이 여럿 없을 때 어떻게 거절하나.** 문구는 팩이
`mcp server <이름> is not available on this node` 로 박았고 (`features.md` 3.2 ·
`scene-gates.md` CA4) 게이트가 그것을 부분 문자열로 찾는다.

A) **첫 이름으로 거절하고 나머지는 Note 에 남긴다.** 정렬한 순서의 첫 이름으로
문구를 짓는다 — 같은 계약이면 같은 문구다. 못 찾은 나머지 이름은 Note 로 나가
(물음 5 의 답이 정한 면에) 한 번에 보인다. 게이트의 문자열이 안 바뀐다

B) **첫 이름 하나로만 거절한다.** Note 도 안 남긴다. 가장 작다. 대가는 셋을
틀리게 적은 계약이 세 번 돌아야 다 고쳐지는 것이다

C) **전부 나열한다.** `mcp servers nope, nope2 are not available on this node`.
한 번에 고친다. 대가는 팩의 문구와 게이트 명령을 함께 고쳐야 하고, 이름이
하나일 때 단수형을 유지하는 갈래가 코드에 생기는 것이다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 7

**원격 노드 선언의 `credential` 을 허용목록에서 어떻게 다루나.** 1.3 이 원격
항목이 종류 없이 사라지는 것을 보였고, 서더라도 인증은 안 실린다 —
`allowlistEntry` 가 `Credential` 을 안 내보낸다. 원격 MCP 는 HTTP 헤더로
인증하므로 노드 환경변수만으로는 서버에 안 닿는다.

A) **`type` 만 박고 인증은 소유자에게 남긴다.** 소유자가
`headers: {Authorization: "Bearer ${GERRIT_TOKEN}"}` 를 `enode.yaml` 에 적으면
`Extra` 로 그대로 지나간다 (1.3 이 모르는 키의 통과를 쟀다). `credential` 은
오늘대로 「뜨나」 판정에만 쓰이고 파일에 안 나간다. 그 한계를 `business-rules` 가
이름으로 적고 `decisions.md` 에 올린다. 코드는 물음 3 의 `type` 한 줄뿐이다

B) **`credential` 을 헤더로 편다.** `Authorization: Bearer ${이름}` 을 이 유닛이
짓는다. 소유자가 한 줄만 적으면 도는 것이 값이다. 대가는 인증 방식을 추정해
코드에 박는 것이다 — `X-Api-Key` 나 Basic 을 쓰는 서버에는 틀린 헤더가 가고
그 틀림이 `failed` 로만 보인다

C) **원격 선언을 이 회차에서 거절한다.** stdio 만 받는다. 형식이 좁아 못 틀린다.
대가는 `ADR-035` §4.4 의 예시가 절반 죽고 사내 원격 MCP 가 CA6 밖으로 나가는 것이다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## 3. 답을 받은 뒤 낼 산출물

`aidlc-docs/taeels/construction/sources/functional-design/` 에 셋이다.

- [x] `domain-entities.md` — 출처와 허용목록 항목의 형식
  - [x] 출처 셋의 이름과 사는 자리 — 노드 선언(`Local.MCP`) · 워크스페이스
        (`.mcp.json`) · 팩(U5). 각각 누가 쓰는 파일인가와 그래서 어느 어휘인가
  - [x] 허용목록 항목의 최종 꼴 (물음 2 · 3). 어디까지가 우리가 짓는 키이고
        어디부터 원문 그대로인가
  - [x] `Components` 의 필드 — `Servers` 의 타입 (물음 2) · `Notes` 가 담는 것
  - [x] `Job` 이 들고 오는 출처 (물음 1) — `NodeMCP` 와 워크스페이스 자리의 이름
  - [x] 이 유닛이 안 만드는 이름 — `Pack` 의 필드 · `HarnessResult` 의 두 필드
- [x] `business-rules.md` — 필터와 우선순위와 거절
  - [x] 요청 필터 — `agent.mcp` 에 적힌 이름만 실린다. 안 적으면 0 이고
        팩이 실었어도 0 이다 (`decisions.md` 6절 ⑥). U5 가 이 필터를 탄다
  - [x] 우선순위 — 이름이 겹치면 노드가 워크스페이스를 이기고 Note 에 남는다
  - [x] 없는 이름의 거절 (물음 6) — 문구 · 판정 시점 · `ReasonError` 로 타는 길
  - [x] 워크스페이스 파일을 못 읽을 때 (물음 4)
  - [x] 원격 항목의 종류와 인증 (물음 3 · 7). `credential` 이 파일에 안 나가는
        것과 그 한계
  - [x] 광고에 안 실리는 것 — 워크스페이스의 서버 (`features.md` 3.4).
        이 유닛이 `mcpFP` 를 안 만진다는 것으로 지킨다
  - [x] Notes 가 담는 사실 목록과 담지 않는 것 (값은 안 담는다 · 이름만)
- [x] `business-logic-model.md` — 값이 지나가는 길
  - [x] `enode.yaml` · `.mcp.json` · `agent.mcp` 가 `resolveComponents` 에서
        만나 `<계장>/mcp.json` 이 되고 `--mcp-config` 로 하네스에 닿는 한 줄기
  - [x] `runHarness` 의 ① ~ ⑨ 중 이 유닛이 자라는 자리 — ② 하나다.
        치명이 exec 앞이라는 불변식이 이 거절로 지켜진다
  - [x] 거절이 `HarnessResult{Reason: ReasonError}` 에서 `claim.go:784` 의
        `res.Error` 까지 가는 길. 새 `Reason` 을 안 만든다
  - [x] CA4 의 세 줄이 각각 어느 규칙으로 초록이 되나 — 요청 하나 · 워크스페이스의
        것 · 없는 이름
  - [x] U5 가 이 자리에 팩을 더할 때 무엇을 딛나 — 필터와 병합의 이음매를 이름으로

---

## 4. 이 단계가 안 만드는 것

```text
   코드          다음 단계다.  이 단계는 규칙과 형식만 낸다
   시험 목록      Code Generation 계획이 낸다
   NFR           회차 실행 계획이 SKIP 으로 닫았다 (셋 다)
   인프라         배포 변경 0
   팩의 상한      U5 의 Functional Design (unit-of-work.md 7절)
```

---

## 5. 파장 — 이 유닛 밖으로 가는 것

```text
   행렬          2.1 의 U4 칸이 「resolveComponents 의 몸통」뿐인데 물음 2 · 3 의
                답이 A 면 allowlistEntry 와 Components 의 타입까지 넓어진다.
                물음 1 의 답이 A 면 claim.go 의 줄이 하나가 아니라 둘이고,
                물음 5 의 답이 A 면 runner.go 에 필드가 하나 더 는다

   decisions.md  6절에 실측 행 — 워크스페이스 파일의 실제 어휘 (1.1) ·
                url 만 적힌 항목이 조용히 사라지는 것 (1.3) · ${} 가 한 겹만
                펴지는 것 (1.4).  1.3 은 U3 가 낸 항목의 결함이므로 2절의
                허용목록 줄도 함께 본다

   scene-gates.md  CA4 의 명령은 안 고친다 — 워크스페이스의 probe3 을 stdio 로
                적으면 1.3 에 안 걸린다.  물음 6 의 답이 C 면 그 절의 문구를 고친다

   features.md 3.2  「서버의 이름 · 종류 · 실행 경로나 주소 · 환경변수 이름만」에
                종류(type)와 헤더가 빠져 있다.  물음 3 · 7 의 답이 그 줄을 넓힌다

   짝 팩과의 접점   runner.go 하나.  U1 이 이미 만진 파일이고 이 유닛은 Job 의
                필드만 더한다 — 스트림 처리 자리(⑱ · ⑲)를 안 건드린다

   U5            팩 출처가 이 유닛의 필터와 병합 위에 선다.  물음 1 · 2 의 답이
                팩을 읽는 자리와 나르는 꼴을 미리 정한다
```

**CA0 의 한 값을 미리 적어 둔다.** `internal/enode` 는 U3 뒤 86.0% 이고 하한이
80% 다. 이 유닛도 그 패키지에만 코드를 더하므로 하한에 가장 가까운 자리가
여기다 — Code Generation 이 표준 명령으로 다시 잰다.

---

## 6. 브랜치를 회차가 아니라 U3 에서 땄다

`CONVENTIONS.md` 3.1 은 유닛 브랜치를 **회차 브랜치에서** 따라고 적는다. 이
유닛은 `unit/advert` 에서 땄다. U2 가 `unit/isolation` 에서, U3 가
`unit/contract-vocab` 에서 딴 것과 같은 이유이고 이 유닛은 그 위에 셋이 있다.

```text
   선행이 셋이다     unit-of-work.md 4절이 U4 의 선행을 U1 · U2 · U3 로 적는다.
                   허용목록을 쓰는 자리가 U1 이고, agent.mcp 가 400 이 아닌 것이
                   U2 이며, Local.MCP 가 있는 것이 U3 다.  회차에서 따면 그 셋이
                   없는 나무 위에서 CA4 를 재게 된다

   aidlc-state.md   같은 문서 루트(aidlc-docs/taeels/)의 파일이고 통째로 다시
                   쓰거나 자동 병합할 수 없다 (CLAUDE.md).  회차에서 따면 네
                   유닛의 절이 파일 꼬리에서 각자 자라 부딪친다
```

코드에서도 부딪히지 않는다 — 이 유닛의 제품 파일 셋(`mcp.go` · `runner.go` ·
`claim.go`) 중 앞 둘을 U1 이 이미 만졌고 **같은 자리를 차례로** 자라는 것이
행렬 2.1 · 2.2 의 값이다. **진행자에게 넘긴다** — U1 · U2 · U3 의 PR 이 순서대로
`main` 에 들어가면 이 브랜치의 PR 차이는 U4 것뿐이다.
