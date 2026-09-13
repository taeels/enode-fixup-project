# U3 `advert` — 선언과 광고의 형식

이 문서는 **무엇이 어떤 모양인가**만 적는다. 거절과 판정은 `business-rules.md`,
값이 지나가는 길은 `business-logic-model.md` 다.

계획과 답 일곱은 `../../plans/advert-functional-design-plan.md` —
**1=A · 2=A · 3=A · 4=A · 5=A · 6=B · 7=B**.

---

## 1. `Local.MCP` — 노드가 선언하는 자리

`internal/enode/config.go` 의 `Local` 에 맵 하나가 는다.

```go
// MCP 는 이 기계가 가진 MCP 서버다 (ADR-012 · ADR-035 §4.4).
//
// 중앙에 안 적는다 — 그 기계에서만 아는 것이다. 여기 없는 서버는
// 광고되지 않고, 광고되지 않으면 계약이 그 노드를 못 고른다.
MCP map[string]MCPServer `yaml:"mcp,omitempty"`
```

**맵의 키가 서버의 이름이고 그 이름이 곧 광고 키의 꼬리다** — `probe` 를 적으면
`mcp.probe` 로 광고된다. 이름은 사람이 고르고 계약이 그 글자를 그대로 적는다
(`requires` 의 `mcp.probe` · `agent.mcp` 의 `["probe"]`).

`SampleLocal()` 에 주석 한 줄이 는다. 「없다」만 말하고 무엇을 적어야 하는지
안 말하면 사람이 또 찾는다 — 그것이 이 함수가 있는 이유다.

```text
    # mcp: { probe: { command: /opt/tools/probe-mcp } }   이 기계에만 있는 MCP
```

**`Local` 의 필드가 열둘이 된다.** 그 형식의 주석이 「열 줄을 넘기지 않는 것이
원칙이다」로 적고 오늘 이미 열하나다. 이 유닛이 그 수를 다시 정하지 않는다 —
`mcp:` 는 `ADR-035` §4.4 가 이름과 모양까지 정본으로 박은 자리이고, 원칙을 고칠지는
그 주석의 주인이 정한다. `code-summary` 가 넘기는 것으로 적는다.

---

## 2. `MCPServer` — 서버 하나의 모양

U1 이 `internal/enode/mcp.go` 에 세웠다. 이 유닛이 **필드 하나를 더하고 둘의
뜻을 못 박는다**. 나머지는 그대로다.

```go
type MCPServer struct {
	Command    string            `yaml:"command,omitempty"`    // stdio
	Args       []string          `yaml:"args,omitempty"`       // stdio
	URL        string            `yaml:"url,omitempty"`        // remote
	Credential string            `yaml:"credential,omitempty"` // remote. 환경변수 이름이다
	Env        map[string]string `yaml:"env,omitempty"`        // 이름에서 이름으로 (3절)

	// Extra 는 우리가 모르는 키다 — 그대로 허용목록에 옮긴다 (decisions.md 2절).
	//
	// yaml 이 모르는 키를 말없이 버리므로 담을 자리가 없으면 그 결정이
	// 코드로 안 선다. 하네스의 새 키(transport · headers 같은)가 우리 판을
	// 갈아끼우지 않고 지나가는 길이 이 필드다.
	Extra map[string]any `yaml:"-"`
}
```

**태그를 명시로 단다.** yaml 은 태그가 없으면 필드 이름을 소문자로 낮춰 맞추므로
오늘도 다섯이 다 맞는다 — 그러나 맞는 이유가 규약이 아니라 우연이면 필드 이름을
고치는 사람이 설정 형식을 함께 고친 줄 모른다.

**`Extra` 는 `yaml:"-"` 다.** 읽기는 2.1 의 `UnmarshalYAML` 이 직접 하고, 쓰기는
이 유닛이 안 짓는다 — `setup` 이 설정을 **새로 지어 쓰기만 하고**
(`setup.go:94` 가 `Local{}` 를 세운다) 읽어서 다시 쓰는 경로가 저장소에 0 이라
오늘 잃을 것이 없다. **그 0 이 깨지는 날 `MarshalYAML` 이 함께 서야 한다** —
그때 없으면 소유자가 적은 모르는 키가 다시 쓰기에서 조용히 사라진다.
`business-rules.md` 6.2 가 이 한계를 이름으로 진다.

### 2.1 두 번 푼다 — 종류 검사를 안 잃는다

`MCPServer` 가 `UnmarshalYAML(*yaml.Node)` 를 든다. **같은 노드를 두 번 푼다.**

```text
   ① 그림자 구조체로   필드 다섯을 그대로 받는다.  yaml 의 종류 검사가 여기서
                      돈다 — args: "--x" 도 command: [a,b] 도 이 자리에서 오류다
   ② map[string]any 로 키 전부를 받는다.  아는 키 다섯을 지우면 남는 것이 Extra 다
```

한 번만 풀면 둘 중 하나를 잃는다 — 구조체로만 풀면 모르는 키가 사라지고,
맵으로만 풀면 `args: "--x"` 가 오류 대신 문자열로 들어와 **거절 다섯**
(`business-rules.md` 1절)이 못 잡는 자리가 는다. 실측 표가 그 경계를 적는다
(계획 1.1).

**`Extra` 가 비면 `nil` 이다** — 빈 맵을 안 만든다. `allowlistEntry` 가
`len(s.Extra) > 0` 로 갈라도 되고 `range` 가 `nil` 에서 0 바퀴 돌아도 된다.

---

## 3. `Env` 는 이름에서 이름으로 간다 (답 3=A)

```yaml
mcp:
  gerrit:
    url: https://gerrit.example/mcp
    credential: GERRIT_TOKEN
    env: { GERRIT_TOKEN: GERRIT_TOKEN }
```

**키도 값도 환경변수 이름이다.** 키는 하네스가 서버에게 줄 변수의 이름이고,
값은 **노드 환경에서 그 값을 길어 올 변수의 이름**이다. 값을 직접 적지 않는다 —
`features.md` 3.2 가 「파일에는 이름만 적는다. 값은 적지 않는다」로 박았고,
값을 적으면 그것이 계장 디렉터리의 `mcp.json` 에 그대로 남는다.

허용목록에는 참조 꼴로 나간다.

```json
"env": { "GERRIT_TOKEN": "${GERRIT_TOKEN}" }
```

**실제 값은 R1 이 넘긴다** — 노드 환경의 화이트리스트가 하네스 프로세스에 그
변수를 실어 주고, 하네스가 자기 설정에서 그 참조를 편다.

**그 참조 꼴은 이 문서가 아직 못 잰다.** `${이름}` 이 하네스가 실제로 펴는
꼴인지는 실측이 없고, 저장소에 픽스처가 0 이다. **Code Generation 이 잰다** —
계획 물음 3 의 A 가 그 확인을 그 단계에 맡겼다. 안 펴면 대안은 `env` 를 아예 안
쓰는 것이다 (R1 이 이미 노드 환경으로 그 변수를 넘기므로 자식이 물려받는다).
**어느 쪽이든 값이 파일에 안 실린다는 규칙은 안 바뀐다.**

`Credential` 과 `Env` 는 다른 것을 뜻한다.

```text
   Credential   뜨나 판정이 보는 이름 하나.  이 이름의 변수가 노드 환경에
                있으면 remote 가 뜬다고 본다.  허용목록에 안 나간다
   Env          하네스가 서버에게 줄 변수들.  허용목록에 참조로 나간다.
                판정은 이것을 안 본다
```

---

## 4. 광고 키

```text
   mcp.<이름>       "1"      뜨는 서버마다 하나
   harness.<이름>   "1"      쓸 수 있는 하네스마다 하나.  오늘은 harness.claude 뿐
   harness          <이름>   옛 키.  한동안 같이 싣는다 (ADR-035 §6)
```

**값이 언제나 `"1"` 인 이유는 매처가 완전 일치만 보기 때문이다** —
`contract.Capability.Satisfies` 가 `got != want` 로 자른다. 버전이나 경로를 값에
실으면 계약이 그 글자를 맞춰 적어야 하고, 그 순간 「그 MCP 를 가진 노드」가
「그 판의 그 MCP 를 가진 노드」가 된다. 존재만 말하는 값이 `"1"` 이다.

**옛 `harness` 를 같이 싣는 것이 되돌리기를 싸게 만든다.** 오늘 도는 계약이
`requires` 에 `harness: claude` 를 적고, 새 키만 싣고 옛 키를 걷으면 그 계약의
매칭이 같은 배포에서 끊긴다. **걷는 날은 이 회차가 안 정한다** — 걷는 날
매처의 `attrCount` 가 노드마다 1 씩 줄어 정렬이 움직인다
(`application-design.md` 6.4).

**광고에 안 실리는 것 넷**을 이름으로 적는다.

```text
   credential 의 변수 이름   decisions.md 2절.  mcp.gerrit 이 이미 그 사실을 낸다
   모든 값                  env 의 값도 command 의 경로도 안 실린다
   워크스페이스 .mcp.json    U4 의 것이고 광고에 안 간다 (features.md 3.4)
   팩의 서버                U5.  팩은 계약이 나르는 것이라 노드 속성이 아니다
```

---

## 5. `Fingerprinter` — 비싼 사실 한 종류

`internal/enode/detect.go` 에 선다.

```go
// Fingerprinter 는 비싼 사실 한 종류다 (ADR-035 §4.2).
//
// 이름이 Detector 가 아닌 이유 — 그 이름은 ADR-068 의 주기 장치가 쓴다.
// Nomad 의 fingerprint 가 §4.2 가 든 유비이고 그 단어를 따른다.
type Fingerprinter interface {
	Kind() string // "harness" | "repo" | "mcp"

	// Probe 는 logger 를 받는다. 오늘 costlyAttrs 안에서 도는 log.Warn
	// (ADR-059) 과 FR-3 이 요구하는 서버별 누락 사유가 갈 자리다.
	Probe(ctx context.Context, l Local, log *slog.Logger) (map[string]string, error)
}

var fingerprinters = []Fingerprinter{harnessFP{}, repoFP{}, mcpFP{}}
```

```text
   harnessFP   harness.<이름> 과 옛 harness.  오늘 costlyAttrs 의 첫 갈래다
   repoFP      repo.  DetectRepo 와 WorkspaceID fallback (ADR-036) 을 그대로 옮긴다
   mcpFP       mcp.<이름>.  이 유닛이 새로 짓는 하나
```

**`Kind()` 가 쓰이는 자리는 로그 하나다** — 어느 종류가 무엇을 냈고 무엇이
실패했는지를 사람이 읽는 자리다. 매칭에도 광고에도 안 실린다.

**`Probe` 의 오류는 「못 물어봤다」이지 「못 한다」가 아니다** (답 4=A).
그 종류가 낸 것만 버리고 순회는 계속한다 — 규칙은 `business-rules.md` 4절.

**셋 중 오늘 오류를 내는 것은 0 이다.** 못 쓰는 하네스도 안 뜨는 서버도 「빼고
로그」이지 오류가 아니다. 반환값이 지금 죽어 있는 것을 알면서 남긴다 — 종류가
느는 날(예: 사내 등록부를 묻는 탐지기) 그 자리가 필요하고, 그때 규칙을 새로
정하지 않으려는 것이다.

---

## 6. 이 유닛이 안 만드는 이름

```text
   Detector (시계)   ADR-068 의 주기 장치.  Every · refresh · Capabilities 를 안 고친다
   cheapAttrs       안 고친다.  MCP 는 호출 수가 노드 설정에 비례해 비싼 쪽이다
   capabilities     안 고친다.  다만 그것이 부르는 hasCapability 는 고친다 (답 6=B)
   resolveComponents  U4 의 몸통.  이 유닛은 Local.MCP 라는 출처만 세운다
   Pack             U5
```
