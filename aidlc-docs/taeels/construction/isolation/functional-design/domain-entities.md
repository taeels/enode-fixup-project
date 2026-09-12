# U1 `isolation` — 도메인 형식

이 유닛이 드는 형식이다. **시그니처는 `component-methods.md` 가 이미 정했고**
여기는 그 안의 **필드 표현과 파일의 모양**을 닫는다.

답 다섯은 계획 2절에 있다 — **1=A · 2=C · 3=B · 4=A · 5=A**.

---

## 1. `MCPServer` — 노드가 선언한 서버 하나

형식 자체는 `component-methods.md` 2.1 이 정했다. **이 절이 닫는 것은
`mcp.json` 으로의 사상**이다 (답 5 = A — U1 이 직렬화를 전부 짓는다).

```go
type MCPServer struct {
	Command    string
	Args       []string
	URL        string
	Credential string
	Env        map[string]string
}
```

### 1.1 `mcp.json` 으로의 사상

```text
   MCPServer     mcp.json 의 키    비고
   Command       command           stdio.  비면 키를 안 쓴다
   Args          args              stdio.  빈 배열이면 키를 안 쓴다
   URL           url               remote.  비면 키를 안 쓴다
   Env           env               비었으면 키를 안 쓴다
   Credential    (안 나간다)        아래 1.2
```

**비면 키를 안 쓴다.** `command: ""` 를 적어 보내면 하네스가 그것을 실행 경로로
읽을 수 있고, 그 실패는 우리 파일이 만든 것이 된다. 빈 값은 「안 적었다」이지
「빈 것을 적었다」가 아니다.

### 1.2 `Credential` 은 `mcp.json` 에 안 나간다

**이 유닛이 정한다.** 셋이 같은 방향을 가리킨다.

```text
   하네스가 모르는 키다      ADR-035 §4.4 의 「옮겨 적기가 복사」는 하네스가 아는
                          키에 대한 말이다.  credential 은 우리 어휘다

   갈 이유가 0 이다         remote 서버의 인증은 노드 환경변수로 간다 (R1 화이트리스트).
                          이름을 파일에 적어도 하네스가 그것으로 하는 일이 없다

   같은 값을 이미 골랐다     ADR-035 §8 · FR-3 이 credential 의 환경변수 이름을
                          광고에 안 싣기로 했다.  mcp.json 은 광고가 아니지만
                          하네스에 실려 나가는 파일이라 더 먼 자리다
```

`Credential` 이 사는 자리는 **`mcpUp` 의 remote 판정**이다 (U3) — 그 이름의
환경변수가 노드 환경에 있는가. 읽는 쪽이 하나뿐이라 직렬화에 안 걸린다.

### 1.3 `Env` 의 **표현**은 U3 이 닫는다

`unit-of-work.md` 7절이 「`enode.yaml` `mcp:` 절의 필드 — **`Env` 의 표현을
포함한다** — 는 U3 의 Functional Design」으로 배정했다. **U1 은 그 값을 안 정한다.**

```text
   U1 이 지는 것   Env 를 mcp.json 의 env 로 **그대로 옮기는 일**.  운반이다
   U3 이 지는 것   그 맵의 키와 값이 무엇을 뜻하나.  의미다
   U1 이 거는 규칙  mcp.json 에는 값이 안 실린다 (features.md 3.2).
                  U3 의 표현은 이 규칙을 만족해야 한다
```

### 1.4 미지의 키 통과는 U3 의 것이고 U1 이 자리를 비워 둔다

`decisions.md` 2절이 「그 밖의 키는 그대로 허용목록에 옮긴다」를 값으로 박았고
`unit-of-work.md` 7절이 그 통과 경로를 U3 에 줬다. **U1 은 필드를 안 만든다.**
대신 **이음매를 이름으로 적어** U3 이 직렬화를 다시 짓지 않게 한다.

```text
   이음매    서버 객체 하나를 짓는 자리.  아는 키 다섯을 먼저 넣고,
            U3 이 더할 미지의 맵을 그 위에 **합친다**
   충돌 규칙  아는 키가 이긴다.  미지의 키가 command 나 url 을 덮으면
            소유자가 적은 선언과 우리가 읽은 선언이 갈린다
```

---

## 2. `Components` — U1 이 채우는 것

```go
type Components struct {
	Servers map[string]MCPServer
	Pack    *Pack
	Notes   []string
}
```

```text
   Servers   U1 의 제품 경로에서 **언제나 빈 맵**이다.  요청도 팩도 없는 경우만
             다루기 때문이다 (unit-of-work.md 1절).  시험은 직접 채워 넣는다
   Pack      U1 에서 **언제나 nil**.  읽는 코드도 U1 이 안 짓는다 (U5)
   Notes     U1 에서 **언제나 빈 목록**.  겹침도 빠짐도 U4 가 만든다
```

**빈 맵과 nil 을 가른다.** `Servers` 는 nil 이어도 빈 맵이어도 같은 파일이 나와야
한다 — 쓰는 쪽이 길이만 본다. 이것이 3절의 「빈 파일의 모양」을 한 갈래로 만든다.

---

## 3. 허용목록 파일 — `<dir>/mcp.json`

`decisions.md` 2절이 자리와 봉투를 박았다. **이 절이 닫는 것은 빈 파일의
모양**이다.

```json
{"mcpServers":{}}
```

```text
   자리      <dir>/mcp.json.  가짜 홈 밖이다 — decisions.md 2절의 값 그대로
   권한      0600.  이름만 든 파일이지만 계장 안의 다른 파일과 같은 값으로 둔다
   줄        한 줄 + 개행 하나.  들여쓰기를 안 한다
   빈 것     {"mcpServers":{}} 다.  {} 가 아니다
```

**`{}` 로 안 쓰는 이유** — `--strict-mcp-config` 는 「이 파일에 적힌 것만」이고
그 「적힌 것」의 자리가 `mcpServers` 다. 키가 없는 파일을 하네스가 어떻게 읽는지는
실측이 없다. **봉투를 언제나 쓰면 그 물음이 사라진다.** 그리고 `0` 이 의도임이
파일에 보인다 (`features.md` 3.2 — 「요청이 없을 때 0 은 의도다」).

**U1 의 제품 경로가 내는 파일은 이것 하나다.** 서버가 실린 파일은 U4 가 처음
만든다. 그래도 직렬화는 U1 이 다 짓는다 (답 5 = A) — 시험이
`Components{Servers: ...}` 를 직접 넣어 잰다.

---

## 4. 가짜 홈의 자리표

```text
   <dir>/                        0700   계장 임시 디렉터리.  MkdirTemp 가 짓는다
   <dir>/home/                   0700   가짜 홈.  **가장 먼저 짓는다** (불변식 ①)
   <dir>/home/settings.json      0600   훅 설정.  오늘의 <dir>/enode-settings.json 이 여기로 (⑧)
   <dir>/home/.credentials.json  0600   실제 홈에 있을 때만.  복사본이다
   <dir>/mcp.json                0600   허용목록.  home 밖이다
   <dir>/stamp                   0600   기준 시각.  writeStamp 가 Instrument **앞**에서 쓴다
   <dir>/home/skills/            0700   U5 가 짓는다.  U1 은 안 만든다
   <dir>/home/agents/            0700   U5 가 짓는다.  U1 은 안 만든다
```

`CLAUDE_CONFIG_DIR` 이 `<dir>/home` 을 가리킨다. `Fixed(dir)` 가 값으로 박는다.

**`skills/` 와 `agents/` 를 U1 이 미리 안 만든다.** 빈 디렉터리를 두면 팩이
없는 단계에서도 하네스가 그 자리를 훑고, 무엇보다 「U5 가 만든다」는 소유가
흐려진다. 없는 것이 오늘의 참이다.

---

## 5. `logs/` 의 줄 종류 셋

답 1 = A (새 JSON 객체) · 답 2 = C (토큰 수까지) · 답 4 = A (표시 줄은 stdout 끝).
**파일 전체가 한 줄 한 JSON 객체다.**

### 5.1 전문 — 셋

```text
   system/init      첫 번째 것 하나.  줄 전체를 원문 그대로
   최종 result      type == "result" 인 것 중 **마지막** 것.  줄 전체를 원문 그대로
   stderr           통째로.  JSON 이 아니어도 그대로.  stdout 다음에 붙는다
```

### 5.2 껍데기 — 그 밖의 모든 사건

**원본에서 지우는 것이 아니라 새 객체를 짓는다** (답 1 = A). 하네스가 필드를
늘려도 안 샌다 — 짓는 쪽이 허용목록이기 때문이다.

```text
   type       언제나 있다.  봉투의 type 을 그대로
   subtype    type == "system" 일 때만.  봉투의 subtype 을 그대로
   tools      tool_use 블록의 name 을 나온 순서대로.  없으면 키가 없다
   ok         tool_result 블록이 있을 때만.  is_error 가 참인 블록이 하나라도
              있으면 false, 아니면 true
   tokens     아래 5.3.  원본에 없으면 키 자체가 없다
```

```json
{"type":"assistant","tools":["Read"],"tokens":{"in":10,"out":3,"cache_write":10128,"cache_read":13551}}
{"type":"user","ok":false}
{"type":"system","subtype":"hook_response"}
{"type":"rate_limit_event"}
```

**`ok` 는 있을 때만 쓴다.** 도구를 안 부른 사건에 `ok: true` 를 박으면 「성공한
도구가 있었다」로 읽힌다. 없음과 참을 가른다.

**`tools` 가 배열인 이유** — 실측은 사건마다 블록 하나였지만 그것이 보증은
아니다. 하나일 때도 배열로 두면 뒤에 모양이 안 갈린다.

### 5.3 토큰 수 — 넷 + 하나 (답 2 = C)

실측이 `message.usage` 에 **숫자가 아닌 것**도 넣는다는 것을 보였다 —
`service_tier` · `inference_geo` 는 문자열이고 `cache_creation` 은 객체다.
**그래서 `usage` 를 통째로 못 옮긴다.** 이름으로 넷만 집는다.

```text
   in           usage.input_tokens
   out          usage.output_tokens
   cache_write  usage.cache_creation_input_tokens
   cache_read   usage.cache_read_input_tokens
```

**정수가 아니면 안 싣는다.** 그 자리에 문자열이나 객체가 오면 그 키를 건너뛴다 —
「모르는 것은 안 싣는다」가 이 유닛 전체의 규칙이다.

**하나를 더 싣는다 — `system/thinking_tokens` 의 `estimated_tokens`.** 답 2 = C 의
글자는 「`usage` 의 토큰 수」이고 이것은 `usage` 밖이다. 같은 종류의 값이라
(본문이 없는 정수 하나) 싣는 쪽으로 정했고, **범위를 넓힌 자리라 여기 이름으로
적는다** — 빼려면 이 줄 하나를 지우면 된다.

```json
{"type":"system","subtype":"thinking_tokens","tokens":{"thinking":50}}
```

### 5.4 표시 — 한 줄 (답 4 = A)

```json
{"type":"enode.elided","events":17,"bytes":48213}
```

```text
   자리    stdout 선별 결과의 **맨 끝**.  stderr 앞이다
   events  껍데기로 바꾼 사건 + 아예 안 실은 줄.  둘을 합쳐 센다
   bytes   그 줄들의 원문 바이트 수.  개행을 포함한다
```

**`type` 에 점을 넣는다.** 실측한 하네스의 `type` 은 전부 홑단어라
(`system` · `assistant` · `user` · `result` · `rate_limit_event`)
`enode.elided` 는 부딪칠 수 없다.

**언제나 쓴다 — 걷은 것이 0 이어도.** 그래야 읽는 사람이 「이 파일은 걸러진
것」임을 세 값 중 하나(`events: 0`)로 안다. 표시 줄이 없는 파일과 있는 파일이
갈리면 그 판단이 파일마다 달라진다.

---

## 6. `claudeEnvelope` — `type` 을 읽는다 (⑯)

오늘 `Type` 필드는 `harness.go:88` 에 **선언만 되어 있고 읽는 자리가 0** 이다.
이 유닛이 그것을 읽게 한다.

```text
   읽는 자리   ParseClaude 의 switch **앞**
   판정        type != "result" 이면 ReasonError
   함께 채운다  Turns · CostUSD · Session 셋.  ReasonError 로 떨어질 때도
   Message     봉투의 **type 만**.  원문 줄을 안 싣는다
```

**`Message` 에 원문을 안 싣는 이유** — 그 분기의 입력이 크래시 때의
`{"type":"assistant",...}` 이고 그것은 5.2 가 본문을 지우기로 한 바로 그
객체다. `Message` 는 `claim.go:784` 의 `res.Error` 와 `steps/NN-*.json` 으로
**봉인에 들어간다.** 원문을 넣으면 5.2 가 닫은 길이 뒷문으로 열린다.

`type` 이 아예 없는 봉투(`""`)도 `"result"` 가 아니므로 `ReasonError` 다.
그때 `Message` 는 빈 문자열이 아니라 **`no result envelope type` 으로 적는다** —
빈 `Message` 는 `runner.go:152-154` 가 `err.Error()` 로 덮는 자리라 값이 갈린다.

---

## 7. `errAux` — 보조 실패의 표지

```go
var errAux = errors.New("auxiliary instrumentation failure")
```

**감싸는 자리는 하나다** (⑪) — `hook.go` 의 훅 설정 Marshal 과 WriteFile.
`errors.Is` 하나로 부르는 쪽이 갈린다. 자세한 규칙은 `business-rules.md` 2절.

---

## 8. 이 유닛이 **안** 드는 형식

```text
   Pack · PackFile · PackLimits   U5.  mcp.go 에 자리만 나고 U1 이 안 채운다
   Local.MCP                      U3
   AgentParams.MCP · .Pack        U2
   HarnessResult.MCP · .Pack      U5 가 필드를 넣고 채운다
   Fingerprinter · mcpUp · mcpFP  U3
```
