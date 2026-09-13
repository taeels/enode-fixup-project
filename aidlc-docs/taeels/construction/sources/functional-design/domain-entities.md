# U4 `sources` — 출처와 허용목록 항목의 형식

이 유닛이 다루는 것은 **셋이 아니라 둘**이다 — 노드 선언과 워크스페이스.
팩은 U5 다. 규칙은 `business-rules.md`, 지나가는 길은 `business-logic-model.md`.

**에러 문자열과 로그는 영어다** (`CONVENTIONS.md` 2.1). 주석과 이 문서는 한국어다.

---

## 1. 출처는 누가 쓰는 파일인가로 갈린다

같은 「MCP 서버 선언」이지만 **쓰는 사람과 어휘가 다르다.** 이 표가 이 유닛의
설계 전부를 끌고 간다.

| 출처 | 파일 | 누가 쓰나 | 어느 어휘 | 광고 |
|---|---|---|---|---|
| 노드 선언 | `enode.yaml` 의 `mcp:` | 노드 소유자 | **우리 것** — `MCPServer` (U3 가 닫았다) | 실린다 |
| 워크스페이스 | 저장소 루트의 `.mcp.json` | 저장소 | **하네스 것** — 우리가 정의하지 않았다 | 안 실린다 |
| 팩 | tar 안의 `mcp.json` | 계약 작성자 | 하네스 것 | 안 실린다 (U5) |

**워크스페이스 파일의 어휘가 우리 것이 아니라는 사실이 이 유닛의 고정점이다.**
계획 1.1 이 하네스가 스스로 쓴 파일을 읽어 셋을 보였다.

```text
   type       종류를 여기서 읽는다.  우리 MCPServer 에 없는 필드다
   headers    원격 인증이 여기로 간다.  없는 필드다
   env 의 값   값이다.  노드 선언의 「이름에서 이름으로」(U3)가 아니다
```

그래서 **옮겨 적지 않고 그대로 옮긴다** (답 2=A). 뜻을 바꾸는 번역은 계획 1.2
에서 이미 손상으로 드러났다 — 잃거나(`type` · `headers`) 망치거나(`${${X}}`)
죽는다(종류 오류 하나에 파일 전체).

---

## 2. 허용목록 항목의 최종 꼴

`<계장 임시 디렉터리>/mcp.json` 의 `mcpServers` 아래 값 하나가 **항목**이다.
그 꼴은 하네스가 정한 것이고 우리는 거기에 맞춘다.

```text
   출처가 노드     MCPServer -> allowlistEntry() 로 짓는다.  우리가 만드는 유일한 항목
   출처가 워크스페이스  파일에 있던 객체 그대로.  키 하나도 더하거나 빼지 않는다
                  (예외는 3절의 type 한 줄이다)
```

### 2.1 우리가 짓는 키 (노드 출처)

`allowlistEntry()` 가 오늘 내는 것에 종류가 더해진다 (답 3=A). **그 한 줄은
`allowlistEntry` 가 아니라 합친 뒤의 공용 자리에서 붙는다** — 두 출처가 같은
규칙을 받아야 하므로 `resolveComponents` 의 걸음 4 가 한 번에 지난다
(Code Generation 계획 4절 ③).

```text
   type        새로 짓는다.  없을 때만 — command 가 있으면 stdio, url 이면 http
   command     MCPServer.Command.  비면 안 적는다
   args        MCPServer.Args.  비면 안 적는다
   url         MCPServer.URL.  비면 안 적는다
   env         MCPServer.Env 를 ${이름} 참조로 (U3 의 envRefs)
   그 밖의 키    MCPServer.Extra 를 먼저 얹고 아는 키가 그 위를 덮는다 (U3)
   credential  안 나간다.  우리 어휘이고 하네스가 모른다 (7절)
```

### 2.2 그대로 옮기는 것 (워크스페이스 출처)

파일에서 읽은 객체를 **한 글자도 안 고치고** 옮긴다. `env` 도 `headers` 도
`type` 도 저장소가 적은 그대로다. 우리가 더하는 것은 `type` 이 없을 때의 그
한 줄뿐이고, 그것도 **없을 때만** 짓는다.

**왜 그대로인가** — 그 파일은 저장소가 하네스에게 쓴 것이다. 우리는 전달자이고
번역자가 아니다. 하네스가 키를 늘리면 우리 판을 갈아끼우지 않고 지나간다
(U3 가 `Extra` 로 노드 선언에 세운 것과 같은 값이다).

---

## 3. `type` 은 우리가 채우는 유일한 키다

계획 1.3 의 실측이 근거다 — `type` 도 `command` 도 없는 항목을 하네스가
`mcp_servers` 목록에서 **말없이 뺀다.** `failed` 로도 안 나온다.

```text
   command 가 있다        type 을 stdio 로 채운다
   url 이 있다            type 을 http 로 채운다
   type 이 이미 있다      손대지 않는다.  소유자나 저장소가 적은 것이 이긴다
   command 도 url 도 없다  못 채운다 — 그 항목은 거절이다 (business-rules R7)
```

**이 한 줄이 없으면 광고는 서고 서버는 안 열린다.** `ADR-035` §4.4 의 예시
그대로 적은 원격 선언(`url` + `credential`)이 오늘 그 모양이다.

---

## 4. `Components` — 정해진 것을 담는 그릇

U1 이 세운 형식에서 **`Servers` 의 타입이 바뀐다** (답 2=A).

```text
   Servers   map[string]map[string]any    이름 -> 허용목록 항목.
                                          MCPServer 가 아니라 최종 항목이다
   Pack      *Pack                        nil.  U5 가 채운다
   Notes     []string                     사람이 읽을 사실.  이름만 담는다
```

**왜 `MCPServer` 가 아닌가** — 출처마다 어휘가 다르므로(1절) 한 형식으로는 둘을
못 담는다. 최종 항목으로 올리면 **합치는 자리에서 어휘가 하나가 된다** — 노드
것은 `allowlistEntry()` 를 거쳐 들어오고 워크스페이스 것은 그대로 들어온다.

그 대가로 `writeMCPAllowlist` · `mcpAllowlistJSON` 이 항목을 짓지 않고 받는다 —
정책은 `resolveComponents` 하나에 있고 쓰는 쪽은 정책을 다시 판단하지 않는다
(U1 이 `mcp.go` 머리에 세운 가름 그대로다).

### 4.1 `Notes` 가 담는 것

```text
   담는다     서버 이름 · 출처 이름 · 무엇을 했는가 (이겼다 · 없다 · 요청 안 했다)
   안 담는다   값.  url · args · env 의 값과 이름 · headers · 파일 경로
```

**이름만 담는 것이 규칙이다.** Notes 는 노드 로그로 나가고(5절) 노드 로그는
봉인보다 오래 산다.

---

## 5. `Job` 이 출처를 들고 온다

`resolveComponents` 는 **파일을 하나도 안 만진다** (답 1=A). U1 이 세운
「정한다 / 쓴다」의 가름에 「연다」가 하나 더 붙는 셈이고, 여는 자리는
가장자리다.

```text
   Job.NodeMCP          map[string]MCPServer          claim.go 가 w.Local.MCP 를 싣는다
   Job.WorkspaceMCP     map[string]map[string]any     .mcp.json 의 mcpServers 원문
   Job.WorkspaceMCPErr  error                         그 파일을 못 읽은 이유
   Job.Log              *slog.Logger                  Notes 가 나갈 자리 (답 5=A)
```

**`WorkspaceMCPErr` 가 왜 필드인가** — 등급을 정하는 것이 정책이고 정책은
`resolveComponents` 에 있다 (답 4=A). 읽는 쪽이 등급까지 정하면 거절이
`HarnessResult` 를 안 타고 나가서, 단계 오류의 꼴이 경로마다 달라진다.

**`Log` 가 nil 이면 안 찍는다.** 시험이 로거 없이 돌고, 그때도 판정은 같다.

### 5.1 읽는 자리 — `readWorkspaceMCP`

`mcp.go` 에 산다 (MCP 어휘가 거기 있다). 부르는 것은 `claim.go` 한 줄이다.

```text
   부르지 않는 때   요청이 0 일 때 (agent.mcp 가 비었다) — 파일을 아예 안 연다
                   w.Local.Workspace 가 빈 문자열일 때 (business-rules R6)
   읽는 것          <워크스페이스>/.mcp.json 의 mcpServers 아래 객체들
   없는 파일        빈 것이다.  오류가 아니다
   mcpServers 가 없는 파일   빈 것이다.  오류가 아니다 — 다른 목적의 파일일 수 있다
   못 읽거나 못 푸는 파일     오류다.  등급은 resolveComponents 가 정한다
```

---

## 6. 이 유닛이 만들지 않는 이름

```text
   Pack 의 필드            U5.  오늘 빈 구조체 그대로 둔다
   HarnessResult.MCP · Pack  U5.  기록은 이 유닛의 것이 아니다
   새 Reason               안 만든다.  거절은 ReasonError 로 간다
   MCPServer 의 새 필드      안 더한다.  워크스페이스 것을 그 형식으로 안 받으므로
   광고 키                  U3 가 닫았다.  mcpFP 는 Local.MCP 만 본다
```

---

## 7. `credential` 은 파일에 안 나간다 (답 7=A)

`MCPServer.Credential` 은 **우리 어휘**다. 하네스가 모르는 키이고, 원격 MCP 의
인증은 HTTP 헤더로 가므로 이름을 파일에 적어도 하네스가 그것으로 하는 일이 없다.

```text
   쓰이는 자리   mcpUp 의 remote 판정 하나 — 그 이름의 환경변수가 노드에 있는가
   안 쓰이는 자리  허용목록.  오늘도 안 나가고 이 유닛도 안 내보낸다
   인증을 실으려면  소유자가 enode.yaml 에 headers 를 적는다.  모르는 키라
                 Extra 로 그대로 지나가고 하네스가 ${이름} 을 편다
```

**그래서 「선언만으로 원격 MCP 가 도는」 경로는 이 회차에 없다.** 그 한계를
`business-rules` 9절이 이름으로 지고 `decisions.md` 로 올린다.
