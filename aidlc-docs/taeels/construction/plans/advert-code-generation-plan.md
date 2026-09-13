# U3 `advert` — Code Generation 계획

**유닛** `advert` · **브랜치** `unit/advert` · **담당** taeels ·
**닫는 게이트** CA2 · CA3 완결 (+ CA0) · **선행** U1 `isolation` · U2 `contract-vocab`

**이 계획이 Code Generation 의 정본이다.** 여기 없는 것은 안 짓는다.
설계는 `construction/advert/functional-design/` 의 셋이고 규칙 번호 R1 ~ R14 는
그 `business-rules.md` 다.

---

## 0. 앞 단계의 승인과 이 단계의 착수

사용자가 「승인」으로 Functional Design 을 닫았다 (커밋 `668dcaa`).

**승인이 열어 준 것 셋을 이 단계가 싣는다** — 요구 팩과 회차의 문서다.
U1 이 ⑳ 에서, U2 가 ㉑ 에서 같은 순서를 밟았다.

```text
   decisions.md 6절        실측 행 하나.  넷을 한 행에 —
                           yaml 이 종류만 잡는다 · 「그 밖의 키」가 오늘 안 생긴다 ·
                           env 의 갈림 · 현황판이 원문 키로 낸다
   decisions.md 2절        노드 선언 형식 행의 env 를 답 3=A 로 고친다
   파일 행렬               2.1 의 U3 칸 (답 2=A) · internal/api/ui 행 (답 7=B).
                           만지는 파일 16 -> 17
```

`scene-gates.md` 는 **안 고친다** — 답 어느 것도 게이트 명령을 안 바꾼다.
`component-methods.md` 도 안 고친다 — 답 4=A 가 `Probe` 의 시그니처를 그대로 둔다.

---

## 1. 이 유닛이 내는 diff 의 모양

```text
   제품 파일 넷     행렬 그대로 셋 + 답 7=B 가 더한 하나(format.mjs).  새 파일 0
   시험 파일        새 파일 둘(config_mcp_test.go · detect_mcp_test.go)과
                   기존 셋에 붙인다.  실행에서 옛 시험 둘이 확정 빨강이라
                   함께 고쳤다 (code-summary 5.1)
   문서            행렬 둘 · 팩 둘 · code-summary.md · 상태와 감사
   패키지 둘        internal/enode · internal/api/ui.  cmd/ 는 소스 diff 0
```

| 파일 | 무엇 | 스토리 |
|---|---|---|
| `internal/enode/config.go` | `Local.MCP` · 거절 여섯 (R1 ~ R6) · `SampleLocal` 한 줄 | US-2 |
| `internal/enode/mcp.go` | 태그 다섯 · `Extra` · `UnmarshalYAML` · `allowlistEntry` · `mcpUp` · `mcpFP` | US-1 · US-3 |
| `internal/enode/detect.go` | `Fingerprinter` · `harnessFP` · `repoFP` · 순회 · `hasCapability` | US-1 · US-4 |
| `internal/api/ui/static/shared/fleet/format.mjs` | `attributeName` · `attributeValue` 의 접두 (R13 · R14) | CA2 의 S1 |

**이 유닛이 닫는 스토리는 넷이다** (`unit-of-work-story-map.md` 0절) —
US-1(선언이 함대에 보인다) · US-2(잘못 적으면 노드가 안 뜨고 사유를 본다) ·
US-3(안 뜨는 서버의 사유를 노드 로그에서 본다) · US-4(함대의 `mcp.<이름>` 을
열거해 본다).

**US-4 는 코드가 0 이다** — `store.Capabilities` 가 키 이름을 안 가리므로
`runctl capabilities` 에 저절로 나타난다. 게이트가 그것을 보는 것이 이 줄의 값이다.

---

## 2. 단계 — 아홉

### Step 1 — `internal/enode/config.go`

- [x] `Local` 에 `MCP map[string]MCPServer \`yaml:"mcp,omitempty"\`` 와 주석
      (`domain-entities.md` 1절의 문장)
- [x] `LoadLocal` 이 `MinFreeGB` 기본값 다음에 `validateMCP(l.MCP)` 를 부른다.
      오류면 `fmt.Errorf("config %s: %w", path, err)` — 오늘의 두 오류와 같은 모양
- [x] `validateMCP` 가 R1 ~ R6 을 본다. **이름 순으로 돈다** — 같은 설정이면
      같은 문구가 나와야 한다 (`ADR-014` 결정 3 의 결)
- [x] `SampleLocal()` 에 `mcp:` 주석 한 줄. 실제 값이 아니라 주석이다 —
      그 문자열이 `examples_test.go` 의 파싱 대상이 아니고 화면 안내다
- [x] `MinFreeGB` 기본값 채우기보다 **뒤**에 둔다. 검증이 먼저 죽으면 그 기본이
      안 채워진 채로 오류만 나가는데, 둘은 무관하다

### Step 2 — `internal/enode/mcp.go` — 형식

- [x] `MCPServer` 의 필드 다섯에 yaml 태그를 명시로 단다
- [x] `Extra map[string]any \`yaml:"-"\`` 와 주석 (`domain-entities.md` 2절)
- [x] `UnmarshalYAML(*yaml.Node)` — **같은 노드를 두 번 푼다.**
      ① 그림자 타입(`type shadow MCPServer`)으로 풀어 yaml 의 종류 검사를 지키고
      ② `map[string]any` 로 풀어 아는 키 다섯을 지운 나머지를 `Extra` 에 담는다.
      남는 것이 0 이면 `Extra` 는 `nil` 이다
- [x] `allowlistEntry` 가 `Extra` 를 **먼저** 얹고 아는 키를 그 위에 덮는다.
      아는 키가 이겨야 소유자가 적은 선언과 우리가 읽은 선언이 안 갈린다 —
      U1 이 그 주석에 적은 순서 그대로다
- [x] `env` 를 참조 꼴로 낸다 — 값 `V` 를 `"${V}"` 로 적는다 (R6 이 `V` 가
      환경변수 이름임을 보장한다). **Step 6 의 실측이 이 줄을 확정한다**
- [x] U1 의 `Credential` 주석에서 「그 이름이 사는 자리는 `mcpUp` 의 remote
      판정 하나다 (U3)」의 괄호를 걷는다 — 이 유닛이 그 자리를 지었다

### Step 3 — `internal/enode/mcp.go` — 판정

- [x] `mcpUp(s MCPServer) error` — stdio 는 절대경로이거나 `exec.LookPath`,
      remote 는 `os.LookupEnv(s.Credential)`. `Credential` 이 비면 `nil`
- [x] 종류는 `Command != ""` 로 가른다. R3 이 둘 다인 선언을 이미 막았으므로
      이 갈래가 모호를 안 만난다
- [x] 오류 문구가 곧 로그의 `err` 다 — `command %q not found in PATH` ·
      `credential %s is not set in the node environment`
- [x] `mcpFP` 와 그 둘 — `Kind() string` 이 `"mcp"` ·
      `Probe(ctx, l, log)` 가 `business-logic-model.md` 4절의 넷을 그대로 돈다
- [x] `Probe` 가 **오류를 안 낸다** (`nil`). 서버 하나가 안 뜨는 것은 이 종류의
      실패가 아니다
- [x] 안 뜨면 R9 의 줄 — `log.Warn("mcp server is not up; dropping it from the
      advertisement", "server", name, "kind", kind, "err", err)`

### Step 4 — `internal/enode/detect.go`

- [x] `Fingerprinter` 인터페이스와 `fingerprinters` 슬라이스
      (`domain-entities.md` 5절 그대로)
- [x] `harnessFP` — 오늘 `costlyAttrs` 의 하네스 갈래를 **그대로 옮긴다**.
      `break` 를 걷고 `harness.<이름>` 을 싣되 **옛 `harness` 는 첫 것으로 남긴다**.
      `HarnessBin` 덮어쓰기가 `claude` 만인 것도 `ADR-059` 의 `Warn` 도 그대로
- [x] `repoFP` — `DetectRepo` 와 **`WorkspaceID` fallback 을 함께** 옮긴다.
      안 옮기면 git 없는 노드에서 `repo` 가 사라져 중립이 깨진다
- [x] `costlyAttrs` 를 순회로 바꾼다 (`business-logic-model.md` 3절의 코드).
      R7 의 `continue` 와 R8 의 덮어쓰기가 그 안이다
- [x] `hasCapability` 가 `mcp.` 접두를 능력으로 안 센다 (R10). `switch` 의
      `os` · `host_arch` · `ws` 옆이 아니라 **접두 검사**라 `strings.HasPrefix` 다
- [x] R12 의 줄을 `capabilities` 가 아니라 **`Detect` 와 `Detector.Capabilities`
      가 부르는 자리 밖**에 두지 않는다 — `capabilities` 안에서 `nil` 을 돌려주기
      직전에 낸다. 거기가 「선언은 있는데 능력이 0」을 아는 유일한 자리다
- [x] `capabilities` 의 라벨 규칙 · 오케스트레이션 갈래 · `cheapAttrs` 는 안 건드린다

### Step 5 — `internal/api/ui/static/shared/fleet/format.mjs`

- [x] `attributeName` 이 접두 둘을 본다 (R13) — `mcp.` -> `MCP 서버 · <이름>` ·
      `harness.` -> `실행 도구 · <이름>`. 표 조회가 먼저이고 접두가 그다음이다
- [x] `attributeValue` 가 그 둘의 `"1"` 을 `있음` 으로 낸다 (R14).
      `"1"` 이 아니면 오늘 그대로 — 값을 해석하지 않는다
- [x] `technicalAttribute` 는 안 건드린다. 새 키는 「제공 기능」 절에 남는다

### Step 6 — 실측 하나 (`env` 의 참조 꼴)

- [x] 이 기계의 `claude`(2.1.266)로 잰다. `mcp.json` 에 stdio 서버 하나를 두되
      `command` 를 **자기 환경을 파일로 쏟는 스크립트**로 하고
      `env: {"PROBE_X": "${SRC}"}` 를 적는다. 하네스 환경에 `SRC` 를 준다
- [x] `-p --output-format stream-json --verbose --strict-mcp-config --mcp-config=<경로>`
      로 한 번 돌리고, 쏟아진 파일의 `PROBE_X` 가 `SRC` 의 값인지 `${SRC}` 원문인지 본다
- [x] **편다.** Step 2 의 참조 꼴이 확정이다 (`PROBE_X=expanded-value-42`). 안 폈다면 `env` 를 허용목록에
      안 쓰고(그 키를 아예 빼고) 노드 환경의 상속에 맡긴다 — 값이 파일에 안
      실린다는 규칙은 어느 쪽이든 산다 (`domain-entities.md` 3절)
- [x] 결과를 `decisions.md` 6절의 실측 행과 `code-summary.md` 에 적는다.
      **이 실측이 FD 를 뒤집으면 FD 의 그 줄을 고친다** — U2 가 CA5 에서 밟은 순서다

### Step 7 — 시험

- [x] `config_test.go` (또는 새 `config_mcp_test.go`) — R1 ~ R6 마다 한 줄.
      **어긴 설정이 실제로 `LoadLocal` 에서 거절되는가**를 잰다. 문구도 함께 잰다
- [x] 같은 파일에 **통과해야 하는 것** — stdio 하나 · remote 하나 · `Extra` 가
      담기는 것 · 빈 `mcp:` · `mcp:` 가 아예 없는 설정
- [x] `mcp_test.go` 에 붙인다 — `allowlistEntry` 가 `Extra` 를 얹고 아는 키가
      **덮는지**. 덮는 방향을 안 재면 순서가 뒤집혀도 초록이다
- [x] 새 파일 `detect_mcp_test.go` — `mcpUp` 의 갈래 넷(절대경로 · PATH ·
      없는 실행파일 · remote 의 있음/없음), `mcpFP.Probe` 가 뜨는 것만 싣는지,
      **안 뜨는 것의 사유가 로그에 나오는지**(`slog` 를 버퍼로 받아 잰다)
- [x] 같은 파일에 R10 · R12 — `mcp:` 만 있는 `Local` 이 능력 0 을 내고 그
      사유 줄이 나오는지. 하네스가 함께 있으면 광고가 나가는지
- [x] `detect_cost_test.go` 의 둘이 **그대로 초록**이어야 한다 — 옛 `harness`
      키와 프로세스 수 1. 그것이 이 리팩터의 안전망이다 (`business-rules.md` 8절)
- [x] `internal/api/ui/tests/format.test.mjs` 에 R13 · R14 와
      **`attributeName('custom') === 'custom'` 이 그대로 초록**임을 잰다

### Step 8 — 변이

- [x] 다섯을 돌려 시험이 무는지 본다 — ① `hasCapability` 의 접두 검사 되돌리기
      ② `allowlistEntry` 의 얹는 순서 뒤집기 ③ `mcpUp` 의 remote 갈래를 언제나
      참으로 ④ `harnessFP` 에서 옛 `harness` 키 빼기 ⑤ `repoFP` 에서
      `WorkspaceID` fallback 빼기
- [x] **하나라도 안 빨개지면 그 자리가 구멍이다** — U1 이 `selectLogs` 에서
      그것을 밟았다. 구멍이면 시험을 더하고 `code-summary` 에 적는다

### Step 9 — 게이트

- [x] CA0 을 표준 명령으로 돈다 — `eval "$(scripts/testdb.sh)"` 뒤
      `go test ./... -count=1` · 커버리지(`-coverpkg=./...` · 패키지별 80%) ·
      `go vet ./...` · `gofmt -l cmd internal` · U+2605 전수 grep ·
      `go run ./scripts/glyphscan.go` · 크로스 빌드 셋 · 심볼 상한 ·
      **Mediator 라우트 26** · 워킹트리 청결
- [x] `internal/api/ui` 의 시험도 돈다 (`node --test`). 그 패키지에 diff 가 생겼다
- [x] CA3 의 **뒤 절반**을 합성 함대로 잰다 — `runctl capabilities` 에
      `mcp.probe` 가 나오고, `requires` 에 그 키를 적은 계약이 그 노드에 배정되고,
      `mcp.nope` 가 `422` 다
- [x] CA2 는 **사람이 잰다** (6절). 이 단계는 그 준비까지다

### Step 10 — 문서와 팩

- [x] `unit-of-work-file-matrix.md` — 2.1 의 U3 칸 · `internal/api/ui` 행 ·
      만지는 파일 수 (그 문서 6절 ② 의 절차)
- [x] `decisions.md` 6절에 실측 행 하나 · 2절의 `env` 줄 (답 3=A)
- [x] `construction/advert/code/code-summary.md`
- [x] `aidlc-state.md` 와 `audit.md`

---

## 3. 갈래를 안 나눈다

Go 셋이 서로를 기다린다 — `detect.go` 의 `mcpFP` 가 `mcp.go` 의 `mcpUp` 을 부르고,
`config.go` 의 검증이 `mcp.go` 의 `MCPServer` 를 본다. **컴파일되는 시점이 하나뿐**
이라 갈래마다 초록을 못 본다. U1 이 같은 이유로 안 나눴다.

`format.mjs` 는 진짜로 독립이다 — 다른 패키지 · 다른 언어 · 컴파일 결합 0.
그런데 **여섯 줄짜리 변경**이라 갈래를 여는 비용이 이득보다 크다.

---

## 4. 이 계획이 정하는 것 — FD 에 없던 자리 셋

```text
   ①  검증 함수의 자리    validateMCP 를 config.go 에 둔다.  mcp.go 가 아니다 —
                         LoadLocal 이 부르는 것이고 그 함수가 config.go 에 산다.
                         mcp.go 는 형식과 판정이다

   ②  R12 를 내는 자리    capabilities 안에서 nil 을 돌려주기 직전.  거기가
                         「선언은 있는데 능력이 0」을 아는 유일한 자리다.
                         Detect 와 Detector.Capabilities 가 둘 다 그 함수를
                         지나므로 경로가 하나다

   ③  로그의 키 이름      server · kind · err 셋.  오늘 하네스가 harness · err 를
                         쓰므로 같은 결이고, kind 가 느는 것은 stdio 와 remote 를
                         가르는 사유가 다르기 때문이다
```

**②는 대가가 있다** — `capabilities` 는 광고마다 불리므로 그 줄이 광고 주기로
나온다 (탐지 주기가 아니다). R9 보다 잦다. 받아들이는 이유는 그 상태가 **사람이
설정을 고쳐야 끝나는 상태**이고, 조용하면 소유자가 빈 노드만 보고 멈추기
때문이다. Part 2 가 실제 빈도를 재서 `code-summary` 에 적는다.

---

## 5. 이 단계가 안 하는 것

```text
   resolveComponents 의 몸통   U4.  이 유닛은 Local.MCP 라는 출처만 세운다
   워크스페이스 .mcp.json      U4
   팩                        U5
   scene-gates.md            안 고친다.  답 어느 것도 게이트 명령을 안 바꾼다
   component-methods.md       안 고친다.  답 4=A 가 시그니처를 그대로 둔다
   옛 harness 를 걷는 것       이 회차가 날을 안 정한다
   GLOSSARY.md               새 글자를 안 들여온다.  그 빚은 진행자의 것이다
```

---

## 6. CA2 는 사람이 잰다

```text
   집행자    이 유닛을 구현하지 않은 사람 (scene-gates.md 2절 머리)
   기계      스크래치.  가짜 서버는 PATH 의 true 로 족하다
   명령      scene-gates.md 3절의 CA2 두 줄
   눈 검증   S1 — 현황판 노드 카드와 제어판 「탐지 능력」 카드의 칩.
            보류로 안 넘긴다 (그 문서 4절)
```

**이 단계가 내는 것은 그 게이트가 돌 수 있는 코드까지다.** 초록 판정은
집행자의 것이고, 빨가면 이 브랜치가 고친다.
