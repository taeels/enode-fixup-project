# 메서드 겉면 — 시그니처와 책임

**비즈니스 규칙은 여기 없다.** 상한의 실제 값 · 필드의 전체 목록 · 병합의 세부는
유닛별 Functional Design 이 짓는다. 여기는 무엇이 무엇을 받아 무엇을 내놓는가와
**어떤 오류를 내놓는가**까지다.

---

## 1. `internal/enode/harness.go` — 인터페이스

```go
type Harness interface {
	Name() string
	Env() []string

	// Fixed 는 하네스가 값으로 박는 환경변수다.
	//
	// dir 은 계장 임시 디렉터리다. 이 하네스의 사적인 세계가 그 아래 산다.
	// 인자로 받는 이유 — 하네스마다 그 세계의 이름이 다르고(claude 는
	// CLAUDE_CONFIG_DIR), 그 이름을 runner 가 알면 안 된다.
	Fixed(dir string) map[string]string

	Usable(ctx context.Context, bin string) error
	Version(ctx context.Context, bin string) (string, error)

	// Instrument 는 하네스의 사적인 세계를 파일로 짓고 플래그를 돌려준다.
	//
	// c 는 이미 해소된 구성요소다 — 이 메서드는 무엇을 열지 결정하지 않는다.
	// 결정은 resolveComponents 가 exec 전에 끝냈다.
	//
	// 오류에 등급이 있고 기본이 치명이다. 훅 설정 쓰기만 errAux 로
	// 감싸 보조로 내리고, 감싸지 않은 오류는 전부 단계를 죽인다.
	// 빠뜨림이 닫히는 쪽으로 틀리게 하려는 것이다.
	//
	// 기준 시각은 여기 안 온다. writeStamp 는 Instrument 앞에서 불리고
	// 그 오류는 이미 버려진다 (runner.go:71-76).
	//
	// 오류를 내도 이미 얻은 플래그는 함께 돌려준다 — 보조 실패 하나가
	// --strict-mcp-config 를 떨어뜨리면 격리의 확실한 겹이 사라진다.
	Instrument(dir, self string, a HookArgs, c Components) ([]string, error)

	Argv(p AgentParams, io IOPaths) []string
	Decode(r io.Reader, exitCode int, emit func(Event)) HarnessResult
}

// errAux 는 보조 실패다. 이것으로 감싼 오류만 부르는 쪽이 삼킨다.
// 감싸는 자리는 하나다 — 훅 설정 쓰기(hook.go 의 Marshal 과 WriteFile).
var errAux = errors.New("auxiliary instrumentation failure")
```

`HarnessResult` 에 두 필드가 는다.

```go
type HarnessResult struct {
	// ... 오늘의 여섯 그대로 ...

	// MCP 는 허용목록에 실제로 실린 서버 이름이다 (ADR-005 성질 4).
	// 요청한 것이 아니라 실린 것이다 — 둘이 갈리면 봉인이 그것을 안다.
	MCP []string `json:"mcp,omitempty"`
	// Pack 은 실린 팩 tar 의 sha256 이다. 팩이 없으면 빈 값이다.
	Pack string `json:"pack,omitempty"`
}
```

---

## 2. `internal/enode/mcp.go` — 새 파일

### 2.1 형식

```go
// MCPServer 는 노드가 선언한 MCP 서버 하나다 (ADR-035 §4.4).
//
// stdio 와 remote 를 한 구조체가 받는다 — 하네스가 읽는 형식과 같아서
// 허용목록으로 옮겨 적는 것이 복사가 된다.
type MCPServer struct {
	Command    string            // stdio
	Args       []string          // stdio
	URL        string            // remote
	Credential string            // remote. 환경변수 이름이다. 값이 아니다
	Env        map[string]string // 이름만 적는다 (FD 가 표현을 닫는다)
}

// Components 는 이 단계가 하네스에 실어 줄 것 전부다.
//
// 파일이 아니다 — 메모리다. 그래서 resolveComponents 가 실패해도
// 아무것도 안 남고, 쓰는 쪽은 정책을 다시 판단하지 않는다.
type Components struct {
	Servers map[string]MCPServer // 허용목록에 실릴 것
	Pack    *Pack                // nil 이면 팩 없음
	Notes   []string             // 로그로 낼 사실 — 이름만 담는다
}

// Pack 은 검증을 통과한 팩이다. tar 를 다시 안 연다.
type Pack struct {
	SHA256 string
	Files  []PackFile           // skills/ · agents/ 아래의 것만
	MCP    map[string]MCPServer // 팩의 mcp.json
}

type PackFile struct {
	Name string // 팩 안의 상대경로. 검증을 통과한 것만 여기 온다
	Mode fs.FileMode
	Data []byte
}

// PackLimits 는 푸는 쪽이 디스크를 채우는 길을 막는다 (SEC-A).
// 값은 팩 유닛의 Functional Design 이 닫는다.
type PackLimits struct {
	MaxBytes int64
	MaxFiles int
}
```

### 2.2 결정 — 파일을 안 만진다

```go
// resolveComponents 는 이 단계가 무엇을 열지 정한다 (FR-2 · FR-4 · FR-6).
//
// 파일을 하나도 안 만진다. 그래서 시험이 하네스도 디스크도 안 쓰고,
// 오류가 곧 단계 실패다 — 부르는 쪽이 exec 전에 부른다.
//
// 세 출처 다 agent.mcp 가 적은 이름만 집는다. 팩도 필터를 탄다 —
// 팩은 정의의 출처이지 허가의 출처가 아니다.
//
// 이름이 겹치면 팩 · 노드 · 워크스페이스 순으로 앞이 이긴다. 다만 노드가
// 선언한 이름을 팩이 덮는 것은 오류다. 그것을 허용하면 매칭은 소유자의
// 값으로 하고 실행은 계약의 값으로 하게 된다.
//
// 요청한 이름이 셋 어디에도 없으면 오류다.
func resolveComponents(j Job) (Components, error)

// mcpUp 은 이 서버가 지금 뜨겠는가다 (FR-3).
//
// 실제 연결은 안 한다 — 광고마다 남의 서버를 두드리는 것은 순연이다.
func mcpUp(s MCPServer) error

// readPack 은 tar 를 검증하면서 메모리로 읽는다 (SEC-A).
//
// 쓰지 않는다. 거부가 파일을 남기기 전에 일어나야 하기 때문이다.
// 절대경로 · .. · 심볼릭 링크 · 상한 초과는 오류다. settings.json 은
// 이름으로 건너뛰고, 규약 밖의 항목은 이름만 Notes 로 남긴다.
func readPack(r io.Reader, lim PackLimits) (*Pack, []string, error)

// mcpFP 는 뜨는 서버만 광고 속성으로 낸다 (FR-3).
//
// Fingerprinter 의 한 종류다. 프로세스를 안 띄우지만 호출 수가 노드
// 설정에 비례하므로 값싼 쪽에 안 둔다 (requirements.md 4.4).
type mcpFP struct{}

func (mcpFP) Kind() string
func (mcpFP) Probe(ctx context.Context, l Local, log *slog.Logger) (map[string]string, error)
```

**오류 문구는 계약이다.** 게이트가 단계 error 안에서 **부분 문자열로** 찾는다.

```text
   mcp server <이름> is not available on this node        CA4
   pack redefines node-declared mcp server <이름>          CA5
```

---

## 3. `internal/enode/runner.go` — 순서와 등급

```go
// Job 에 한 필드가 는다.
type Job struct {
	// ... 오늘 그대로 ...

	// NodeMCP 는 이 노드가 enode.yaml 에 선언한 서버다 (ADR-063).
	// Job 이 드는 이유 — resolveComponents 가 exec 전에 여기서 돈다.
	NodeMCP map[string]MCPServer
}

func runHarness(ctx context.Context, h Harness, bin string, j Job) ([]byte, HarnessResult)
```

바뀐 순서다. **치명이 전부 exec 앞에 모인다.**

```text
   ①  계장 디렉터리를 짓는다        실패 -> 단계 실패 (Q2 = A)
   ②  resolveComponents(j)        실패 -> 단계 실패.  하네스를 안 띄운다
   ③  Argv 를 조립한다
   ④  Instrument(tmp, self, a, c)  errAux 로 감싼 오류만 삼킨다.  그 밖은 단계 실패.
                                   삼킬 때도 돌려받은 플래그는 붙인다
   ⑤  Fixed(tmp) 를 합친다
   ⑥  exec
   ⑦  Decode · Version · 자백 읽기  오늘 그대로
   ⑧  HarnessResult 에 MCP · Pack 을 채운다
   ⑨  defer 가 계장 디렉터리를 지운다 — 오늘 그대로
```

**⑨ 를 안 건드린다.** 보존 스위치를 한 번 넣었다가 뺐다 — `features.md` 3.1 이
「복사한 자격증명은 계장 디렉터리와 함께 **단계 끝에 지워진다**」를 보안 요구로
못 박았고(SECURITY-12), 남기는 스위치는 그것을 뚫는다.

**대가는 눈 검증이 복제본을 잰다는 것이다.** 사람이 `init` 줄을 읽으려고 손으로
띄울 때 실물 가짜 홈과 실물 허용목록이 없어서 다시 짓게 되고, 그 복제본은 환경
(`harnessEnv` 는 `os.Environ()` 을 안 얹는다) · 플래그(`--settings` ·
`--permission-mode` · `--add-dir` 이 빠진다) · 게이트웨이 인증(`apiKeyHelper` 가
빠져 `Not logged in` 이 난다)에서 실물과 갈린다. **그 한계를 게이트가 명시로 안다.**

**`Instrument` 를 언제나 부른다.** 오늘은 `os.Executable()` 이 빈 문자열이면 계장을
통째로 건너뛰는데, 그 경로로 가면 허용목록이 조용히 안 쓰인다. `self` 가 비면
`settings.json` 에서 `hooks` 키만 빠지고 나머지는 그대로 돈다 — 보조 실패다.
**키를 빼는 것이지 빈 값을 쓰는 것이 아니다** (`application-design.md` 4.2).

새 Reason 값은 안 만든다. 치명은 `HarnessResult{Reason: ReasonError, Message: <사유>}`
로 보고한다. 근거 — Record 어휘를 늘리는 것은 팩의 제외 여덟 중 「기존 계약
변경」에 걸리고, 판정은 어차피 같다 (`Completed()` 가 거짓이다).

---

## 4. `internal/enode/claude.go` — 어댑터

```go
// Fixed 는 재현성을 위해 우리가 값으로 박는 것이다.
//
// CLAUDE_CONFIG_DIR 이 가짜 홈을 가리킨다. 통과 목록(Env)에 안 넣는다 —
// 넣으면 노드에 그 변수가 없을 때 조용히 안 걸린다.
func (claudeHarness) Fixed(dir string) map[string]string

// Instrument 는 훅 · 가짜 홈 · 팩 · 허용목록을 심고 플래그를 돌려준다.
//
//	<dir>/home/                   가짜 홈.  가장 먼저 만든다.  CLAUDE_CONFIG_DIR 이 가리킨다
//	<dir>/home/settings.json      훅.  features.md 3.1 의 요구대로 홈 안이다
//	<dir>/home/skills/…           팩의 스킬
//	<dir>/home/agents/…           팩의 서브에이전트
//	<dir>/home/.credentials.json  실제 홈에 있으면 0600 으로 복사
//	<dir>/mcp.json                허용목록
//
// self 가 비면 settings.json 에서 hooks 키 자체를 뺀다. 빈 첫 원소를
// shellJoin 이 그대로 이어 붙여 앞이 빈 명령이 실리는 것을 막는다.
// gatewayAuthFields() 는 그때도 얹는다.
//
// 플래그: --settings <경로> --setting-sources "" (오늘)
//         --strict-mcp-config --mcp-config=<경로> (새로. 등호 형태)
func (claudeHarness) Instrument(dir, self string, a HookArgs, c Components) ([]string, error)
```

`Argv` 는 **한 줄 바뀐다** (`decisions.md` 6절 ⑮) — `-p --output-format json` 이
`-p --output-format stream-json --verbose` 가 된다. 게이트 CA1 · CA4 · CA5 가 재는
`system/init` 줄이 그래야 `logs/` 에 남는다.

`Decode` 는 **안 바뀐다.** `ParseClaude` 가 `lastJSONObject` 로 마지막 JSON 객체를
집으므로 stream-json 의 `result` 줄을 그대로 읽는다 — 봉투 파싱이 안 흔들린다.
스트림을 훑는 것과 tee 와 사건 배출은 짝 팩의 것이다.

---

## 5. `internal/enode/config.go` — 노드 선언

```go
type Local struct {
	// ... 오늘 그대로 ...

	// MCP 는 이 기계가 가진 MCP 서버다 (ADR-012 · ADR-035 §4.4).
	//
	// 중앙에 안 적는다 — 그 기계에서만 아는 것이다. 여기 없는 서버는
	// 광고되지 않고, 광고되지 않으면 계약이 그 노드를 못 고른다.
	MCP map[string]MCPServer `yaml:"mcp,omitempty"`
}
```

`SampleLocal()` 에 `mcp:` 주석 한 줄을 더한다 — 「없다」만 말하고 무엇을 적어야
하는지 안 말하면 사람이 또 찾는다.

---

## 6. `internal/enode/detect.go` — 광고

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

// costlyAttrs 가 셋을 순회해 합친다. 동작 중립이다 —
// 하네스 순회의 break 는 harnessFP 안에서 저절로 걷힌다 (ADR-035 §4.3).
//
// repoFP 는 DetectRepo 의 WorkspaceID fallback (ADR-036) 을 그대로 옮긴다.
// 안 옮기면 git 없는 노드에서 repo 속성이 사라져 중립이 깨진다.
func costlyAttrs(ctx context.Context, l Local, log *slog.Logger) map[string]string
```

`cheapAttrs` · `capabilities` · `Detector`(시계)는 **안 건드린다.** 순회로 바꾸는
것은 `ADR-035:246-248` 이 「오늘 동작을 안 바꾸고 된다」고 적은 리팩터이고, 기존
탐지 시험이 그 중립성의 안전망이다.

---

## 7. `internal/contract` — 계약 어휘

```go
// agentKeys 에 두 이름이 는다. 검증기는 이미 있다 (ADR-057).
var agentKeys = []string{
	"model", "max_turns", "max_tokens", "ask", "harness", "mcp", "pack",
}
```

`Grammar` 에 줄이 는다 — 계획이 그 문법을 읽는 유일한 경로이기 때문이다
(`grammar_test.go` 가 이 문장마다 「어긴 계약이 실제로 거절되는가」를 잰다).

```text
   agent.mcp    이 단계가 물릴 MCP 서버 이름들.  노드가 선언했거나 팩이 실은 것만
   agent.pack   $IN 의 blob 이름.  in.from 이 그 blob 을 잇는다
   안 적으면     그 단계는 MCP 도 팩도 없이 돈다.  0 은 의도다
```

`internal/contract/examples/mcp.json` 하나가 는다 — 사내 MCP 하나를 `requires` 로
요구하고 `agent.mcp` 로 요청하며 앞 단계가 팩을 받는 최소 계약이다.

**`agentKeys` 는 키 이름만 본다.** `agent.mcp` 를 배열이 아니라 문자열로 적으면
`400` 이 아니라 노드 위 `parseAgentParams` 의 `json.Unmarshal` 에서 죽는다. 그래서
`parseAgentParams` 가 두 키의 **타입을 검증**하고 사유를 문구로 낸다 —
`agent.mcp must be an array of server names` · `agent.pack must be a blob name`.

**`cmd/runctl` 의 diff 는 0 이다.** `runctl example` 이 `contract.ExampleNames()`
로 임베드 FS 를 읽으므로 파일 하나를 더하면 목록과 출력이 함께 는다.
`runctl schema steps` 도 구조체에서 뽑는다. `requirements.md` 6.3 이
「`cmd/runctl` — example 하나」라고 적은 자리의 실제 위치는 `internal/contract`
다. **만지는 경로가 넷에서 셋으로 준다.**

---

## 8. `cmd/iapadapter` — 오케스트레이터

```go
type ExecutorConfig struct {
	As    string
	Attrs map[string]string

	// Pack 은 이 함대가 에이전트 단계에 실어 보낼 팩이다 (ADR-034 §2.2).
	// 비면 오늘 그대로 — 팩 단계가 안 붙고 agent.pack 도 안 붙는다.
	Pack *PackConfig `yaml:"pack,omitempty"`
}

type PackConfig struct {
	Fetch []string `yaml:"fetch"` // argv 하나. git archive --remote · curl -o 등
	Name  string   `yaml:"name"`  // blob 이름. 비면 pack
}
```

`BuildContract` 는 `Pack` 이 있을 때만 두 가지를 더한다 — 맨 앞에 명령 단계
하나(`$OUT/<name>` 에 tar 를 낸다)와, 실행 단계의 `agent.pack` · `in.from`.
계획이 짓는 단계는 `Grammar` 가 문법을 알려주고, 안 적으면 팩 없이 돈다.

**새 전송은 0 이다.** 팩은 이미 있는 blob 경로로만 흐른다.
