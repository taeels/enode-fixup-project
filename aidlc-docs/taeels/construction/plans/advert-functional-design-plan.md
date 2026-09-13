# U3 `advert` — Functional Design 계획

**유닛** `advert` · **브랜치** `unit/advert` · **담당** taeels ·
**닫는 게이트** CA2 · CA3 완결 (+ CA0) · **선행** U1 `isolation` · U2 `contract-vocab`

정본 입력은 `aidlc-docs/v3-run-harness-components/inception/application-design/`
의 넷과 `requirements/harness-components/` 팩이다. 이 계획은 그 위에서
**노드 선언의 형식과 광고의 규칙**만 짓는다 — 코드는 다음 단계다.

---

## 0. 이 단계가 닫는 것과 안 닫는 것

```text
   닫는다    enode.yaml 의 mcp: 절이 받는 필드와 각 필드의 뜻
             그 절의 무엇이 「모양이 틀렸다」이고 그때 노드가 어떻게 되나
             뜨나 판정의 규칙 둘 (stdio 는 resolve · remote 는 환경변수 이름)
             그 판정이 못 잡는 것 셋을 문장으로 (decisions.md 6절 ⑭)
             광고 키의 형식 — mcp.<이름>: "1" · harness.<이름>: "1" · 옛 harness
             Fingerprinter 의 계약 — Kind · Probe · 오류가 뜻하는 것
             누락 사유가 어디에 어떤 빈도로 남나 (FR-3)

   안 닫는다  허용목록 합치기와 출처 우선순위     U4 의 resolveComponents
             워크스페이스 .mcp.json              U4.  광고에는 안 실린다
             팩 tar 와 그 서버                   U5
             agent.mcp · agent.pack 의 어휘      U2 가 닫았다
             매처와 정렬                        코드 diff 0.  CA3 이 동작으로 잰다
             시험 목록과 파일별 diff              이 유닛의 Code Generation
```

---

## 1. 실측 — 코드가 지금 어떻게 생겼나 (2026-09-13)

FD 를 짓기 전에 이 유닛이 만지는 자리를 실제로 읽고 돌렸다. **팩이 값으로 박은
문장 하나가 오늘 코드로 성립하지 않고**, **행렬과 U1 의 이음매 주석이 서로
다른 말을 한다.**

### 1.1 `mcp:` 를 오늘 yaml 이 어디까지 거절하나

`Local` 에 `MCP map[string]MCPServer` 를 얹은 것과 같은 모양으로 열일곱 가지를
실제로 파싱했다 (`gopkg.in/yaml.v3`, 저장소가 쓰는 그 판).

| 적은 것 | 오늘 | 무엇을 뜻하나 |
|---|---|---|
| `mcp: probe` (스칼라) | 오류 | 노드가 안 뜬다. 새 코드 0 |
| `mcp: [probe]` (배열) | 오류 | 같다 |
| `probe: "true"` (서버가 스칼라) | 오류 | 같다 |
| `command: [a, b]` | 오류 | 같다 |
| `args: "--x"` | 오류 | 같다 |
| `command: 5` | **통과** | `Command` 가 `"5"` 다. 경로로 쓰인다 |
| `probe: {}` · `probe:` (null) | **통과** | 빈 서버가 맵에 산다 |
| `command` 와 `url` 이 둘 다 | **통과** | 어느 종류인지 모호하다 |
| `transport: sse` · `headers: {...}` | **통과하고 사라진다** | 1.2 |
| `URL: https://...` (대문자) | **통과하고 사라진다** | 키가 소문자로만 맞는다 |
| `"": { command: ... }` (빈 이름) | **통과** | 광고 키가 `mcp.` 가 된다 |
| `a.b: { command: ... }` | **통과** | 광고 키가 `mcp.a.b` 다 |

**yaml 이 잡는 것은 종류(kind)뿐이다.** 스칼라를 문자열 필드에 넣는 것은 yaml 의
타입 강제라 `command: 5` 가 조용히 `"5"` 가 된다. `decisions.md` 2절의
「**아는 키의 모양은 검증한다 — 틀리면 노드가 안 뜬다**」가 요구하는 것은 이
표의 아래 일곱 줄이고, 그것을 세는 코드가 오늘 0 이다. 물음 1 이 범위를 정한다.

**「노드가 안 뜬다」는 이미 참이다** — `cmd/enode/main.go:112-116` 이 `LoadLocal`
오류를 `cannot read config` 로 내고 `1` 로 죽는다. 검증을 `LoadLocal` 안에 두면
새 배선이 0 이다. `identity.go:54` 가 같은 함수의 오류를 무시하므로 신원 유도는
안 깨진다.

### 1.2 「그 밖의 키는 그대로 허용목록에 옮긴다」가 오늘 불가능하다

`decisions.md` 2절의 노드 선언 형식 행이 그렇게 적고, U1 이 `mcp.go` 의
`allowlistEntry` 주석에 이음매로 남겼다 — 「**U3 이 「그 밖의 키」를 그대로
옮긴다. 그 맵을 여기 먼저 얹고 아는 키를 그 위에 덮는다**」.

그런데 `MCPServer` 는 필드 다섯의 구조체이고 yaml 은 모르는 키를 **말없이
버린다** (1.1 의 `transport` · `headers`). 옮길 맵이 애초에 안 생긴다.
가지려면 `MCPServer` 에 필드가 하나 더 서야 하고, 그 필드는 U1 이 세운 형식이다.

**그리고 행렬이 반대로 적는다** — `unit-of-work-file-matrix.md` 2.1 은
`internal/enode/mcp.go` 를 만지는 넷을 갈라 적으면서 U3 의 몫을
「`mcpUp` · `mcpFP` · `Fingerprinter` 순회. **U1 의 것을 안 고친다**」로 못 박는다.
U1 의 이음매 주석과 이 줄이 같은 자리를 두고 다르게 말한다. 물음 2 가 정한다.

### 1.3 `env` 가 이름인가 값인가 — 요구와 형식이 갈린다

```text
   features.md 3.2   파일에는 서버의 이름 · 종류 · 실행 경로나 주소 ·
                     환경변수 이름만 적는다.  값은 적지 않는다

   mcp.go (U1)       Env map[string]string   // 이름만 적는다. 값은 노드 환경이
                                             // R1 로 넘긴다
                     allowlistEntry 가 이 맵을 통째로 mcp.json 에 얹는다

   ADR-035 §4.4      예시에 env 가 없다.  command · args · url · credential 넷뿐이다
```

`map[string]string` 은 **이름에서 값으로** 가는 모양이다. 노드 소유자가
`env: { GERRIT_TOKEN: ghp_... }` 를 적으면 그 값이 그대로 계장 디렉터리의
`mcp.json` 에 간다 — `0600` 이고 기계 밖으로 안 나가지만, 3.2 가 「값은 적지
않는다」로 박은 줄은 그때 거짓이 된다. 반대로 이름만 적는 뜻이라면 그 맵의
값 자리에 무엇을 적어야 하는지가 저장소 어디에도 없다.

`enode.yaml` 을 읽는 것이 이 유닛이므로 뜻을 정하는 것도 여기다. 물음 3.

### 1.4 탐지기 순회는 오늘 동작 중립이다 — 셋 다 쟀다

```text
   harnesses       harness.go:282 가 []Harness{claudeHarness{}} 하나다.
                   그래서 break 를 걷어도 오늘 도는 결과가 안 바뀐다 —
                   harness 키는 여전히 claude 하나이고 harness.claude 가 는다

   Detector        detector.go 가 시계를 이미 들고 있다.  refresh 가
                   costlyAttrs 만 다시 부르고 cheapAttrs 는 광고마다 새로 본다.
                   CA2 의 「실행파일을 치우면 탐지 주기 뒤 빠진다」가
                   이 갈래로 저절로 선다 — 새 배선 0

   hasCapability   detect.go 의 그 함수가 os · host_arch · ws 밖의 키를 하나라도
                   보면 참이다.  그래서 mcp.<이름> 하나만으로 광고가 나간다 —
                   하네스가 없는 기계도 agent.reason 을 광고한다.  물음 6
```

`TestDetectSpawnsOneProcessPerHarness`(`detect_cost_test.go:41`)가
`caps[0].Attrs["harness"] == "claude"` 를 재므로 **옛 키를 같이 싣는 한 초록이다.**
이 시험이 이 리팩터의 안전망이고, `ADR-035:246-248` 의 전제(「오늘 동작을 안
바꾸고 된다」)가 깨졌는지를 여기가 알려준다.

### 1.5 열거 면 둘은 이미 있다. 화면은 원문 키로 낸다

```text
   GET /v1/capabilities   store.go:513 Capabilities 가 광고의 attrs 를 키마다
                          합집합으로 모은다.  키 이름을 안 가린다 —
                          mcp.probe 가 저절로 나온다 (decisions.md 6절 ⑬ 확인)

   runctl capabilities    main.go:177-193 이 그 맵을 정렬해 찍는다.  코드 0
                          다만 서식이 %-10s 라 harness.claude(14) 가 칸을 넘는다

   제어판                 panel/page.go:214 가 attrs 를 통째로 칩으로 만든다.  코드 0

   현황판                 ui/static/shared/fleet/format.mjs:48 attributeName 이
                          표에 없는 키를 원문으로 낸다.  그래서 칩은 보이되
                          mcp.probe = 1 · harness.claude = 1 로 보인다 —
                          바로 옆의 harness 는 실행 도구 = claude 로 보인다.
                          같은 사실이 한글 이름과 원문 키로 두 줄이 된다
```

CA2 의 눈 검증 S1 이 「칩이 보인다」이므로 **보이기는 한다.** 행렬은
`internal/api/ui` 를 0 으로 센다. 물음 7 이 그 0 을 지킬지 정한다.

### 1.6 안 만져도 되는 것을 확인했다

```text
   매처            contract.go 의 Capability.Satisfies 가 키와 값의 완전 일치를
                   부분집합으로 돈다.  mcp.<이름>: "1" 은 새 어휘일 뿐이고
                   match 의 코드 diff 는 0 이다 (행렬 그대로)

   옛 계약          requires 의 harness: claude 는 옛 키가 남는 동안 그대로 산다.
                   harness.claude 를 함께 싣는 것이 그 호환을 만드는 줄이다

   POST /v1/nodes   attrs 는 map[string]string 이고 키 이름을 안 검사한다.
                   점이 든 키가 그대로 오간다 — 라우트도 저장소도 안 는다

   Detector(시계)   ADR-068 의 것.  Every · refresh · Capabilities 를 안 고친다

   cheapAttrs       안 고친다.  MCP 는 호출 수가 노드 설정에 비례하므로 비싼 쪽이다
                   (components.md 2.2 가 그 근거를 이미 적었다)
```

---

## 2. 물음 일곱

답을 `[Answer]:` 뒤에 적는다. 권장이 있는 물음은 권장을 **A** 에 둔다.

### Question 1

**`enode.yaml` 의 `mcp:` 에서 무엇을 「모양이 틀렸다」로 보고 노드를 안 띄우나.**
1.1 이 오늘 yaml 이 종류만 잡고 일곱 가지가 그대로 통과함을 보였다.
US-2 와 U3 완료 조건과 SECURITY-05 가 이 자리를 가리킨다.

A) **다섯을 거절한다.** 이름이 빈 것 · `command` 와 `url` 이 둘 다 없는 것 ·
둘 다 있는 것 · `args` 만 있고 `command` 가 없는 것 · `credential` 만 있고
`url` 이 없는 것. 「어느 종류인가」가 한 값으로 정해지는 것을 강제하는 셈이고,
`mcpUp` 이 그 종류로 갈라 판정하므로 판정 코드가 모호함을 안 만난다.
문구는 서버 이름을 든다 — `mcp server <이름>: ...`

B) **둘만 거절한다.** 이름이 빈 것과 `command` · `url` 이 둘 다 없는 것.
나머지는 `mcpUp` 이 판정에서 다룬다 (둘 다 있으면 `command` 가 이긴다 등).
설정을 덜 까다롭게 두고 광고에서 드러나게 한다. 대가는 「틀리면 노드가 안 뜬다」가
모호한 선언에는 안 걸리는 것이다

C) **거절을 안 더한다.** yaml 이 잡는 종류 오류로 족하다고 보고 나머지는
`mcpUp` 이 「안 뜬다」로 처리한다. 새 검증 코드 0. 대가는 US-2 의 완료 조건과
`decisions.md` 2절의 「아는 키의 모양은 검증한다」가 이 회차에 안 서는 것이다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2

**`MCPServer` 에 「그 밖의 키」를 담을 자리를 이 유닛이 세우나.** 1.2 가 오늘
그 맵이 안 생김을 보였고, U1 의 이음매 주석과 행렬 2.1 이 서로 다르게 적는다.

A) **이 유닛이 세우고 이 유닛이 쓴다.** `MCPServer` 에 필드 하나를 더해
모르는 키를 담고(`yaml.Node` 나 `map[string]any` 로 직접 푼다),
`allowlistEntry` 가 그 맵을 먼저 얹고 아는 키를 그 위에 덮는다 — U1 의 주석
그대로다. 행렬 2.1 의 「U1 의 것을 안 고친다」를 이 유닛이 고치고 그 사실을
파장에 적는다. 값은 `decisions.md` 2절이 이미 박았으므로 여기서 다시 정하지 않는다

B) **형식만 세우고 쓰는 것은 U4 로 미룬다.** 필드는 이 유닛이 더하고
(`enode.yaml` 을 읽는 것이 여기라서), `allowlistEntry` 는 U4 가 만진다.
U4 가 `resolveComponents` 의 몸통을 짓는 유닛이라 허용목록의 내용이 그쪽에서
한 번에 결정된다. 대가는 이 유닛이 쓰이지 않는 필드를 남기는 것이다

C) **이번 회차는 아는 키만 옮긴다.** `decisions.md` 2절의 그 줄을 5절
(정본에 되돌려 올리는 것)으로 올리고, 모르는 키는 노드 로그에 이름만 남기고
버린다. 대가는 `transport` 같은 하네스의 새 키가 이 회차 동안 못 지나가는 것이다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3

**`enode.yaml` 의 `env` 는 이름인가 값인가.** 1.3 이 요구(`features.md` 3.2 의
「값은 적지 않는다」)와 형식(`map[string]string`)이 갈림을 보였다.

A) **이름에서 이름으로 읽는다.** `env: { GERRIT_TOKEN: GERRIT_TOKEN }` 처럼
값 자리에도 **환경변수 이름**을 적고, 허용목록에는 `${이름}` 꼴의 참조로 나간다.
값이 파일에 안 실리고 실제 값은 R1 이 노드 환경으로 넘긴 것을 하네스가 편다.
검증은 「값 자리가 환경변수 이름 꼴인가」를 본다 — 실측으로 어느 꼴을 하네스가
펴는지 확인이 필요하면 그 확인을 Code Generation 이 진다

B) **이름 목록으로 좁힌다.** `env: [GERRIT_TOKEN]` 로 적고 허용목록에서 펴는
방식은 A 와 같다. U1 이 세운 필드의 타입을 바꾼다 — 1.2 와 같은 자리라
물음 2 의 답과 함께 움직인다. 모양이 요구와 같아서 오해가 안 생기는 것이 값이다

C) **오늘 형식 그대로 두고 문서로만 적는다.** `map[string]string` 을 그대로
옮기고, 「값을 적지 말라」는 `SampleLocal` 의 주석과 `domain-entities.md` 가
말한다. 코드가 안 잰다. 대가는 `features.md` 3.2 의 그 줄이 코드로는 안 지켜지고
소유자가 값을 적으면 그것이 계장 디렉터리의 파일에 남는 것이다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4

**`Fingerprinter.Probe` 의 오류는 무엇을 뜻하고 `costlyAttrs` 가 그것을 어떻게
다루나.** `component-methods.md` 6절이 `(map[string]string, error)` 로 시그니처를
박았는데, 오늘 셋(하네스 · 저장소 · MCP)은 **한 종류도 오류를 안 낸다** —
못 쓰는 하네스도 안 뜨는 서버도 「빼고 로그」이지 오류가 아니다.

A) **그 종류가 낸 것만 버리고 순회는 계속한다.** 오류는 「그 종류를 못 물어봤다」
이고, 나머지 종류의 속성은 그대로 광고에 실린다. 사유는 로그에 남는다.
`refresh` 가 취소 중의 결과를 안 쓰는 것(`detector.go`)과 같은 결이다 —
못 물어본 것을 「못 한다」로 광고하면 노드가 스스로를 지운다

B) **오류를 안 받는다.** 시그니처를 `Probe(ctx, l, log) map[string]string` 으로
좁힌다. 오늘 셋 다 안 내고, 안 내는 반환값은 다음 사람에게 거짓을 가르친다.
`component-methods.md` 6절의 시그니처를 벗어나므로 파장에 적고 그 문서를 고친다

C) **오류면 그 탐지 주기 전체를 버린다.** 이전 값을 유지하고 다음 주기에 다시
잰다. 「일부만 실린 광고」가 안 생긴다. 대가는 한 종류의 실패가 나머지 둘의
갱신을 막는 것이다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5

**안 뜨는 서버의 사유를 얼마나 자주 로그에 내나.** FR-3 이 「빠진 이유는 노드
로그에 남긴다」로 요구하고 CA2 가 그 줄을 눈으로 본다. 탐지는 5분마다 돈다.

A) **탐지마다 낸다.** 오늘 하네스가 그렇게 한다 (`detect.go` 의
`harness is installed but not usable`). 상태를 안 들고, CA2 를 재는 사람이
기다리지 않고 바로 본다. 대가는 안 뜨는 서버 하나가 하루에 288 줄을 내는 것이다

B) **전이에서만 낸다.** 뜨던 것이 안 뜨게 된 그 주기에 한 번 낸다.
`mcpFP` 가 직전 결과를 들어야 하므로 탐지기가 상태를 갖는다. 로그가 조용하고
사건이 눈에 띈다. 대가는 노드를 다시 띄우면 첫 주기에 한 번 내고 조용해져서,
CA2 를 나중에 재는 사람이 아무 줄도 못 보는 것이다

C) **탐지마다 내되 등급을 가른다.** 처음 빠진 주기는 `Warn` 이고 이어지는
주기는 `Debug` 다. 기본 로그는 조용하고 `--debug` 로 언제든 다시 본다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6

**하네스가 없고 `mcp:` 만 있는 노드가 광고를 내보내나.** 1.4 가 오늘
`hasCapability` 로 그렇게 됨을 보였다 — `mcp.probe` 하나가 「무언가 할 줄 안다」로
읽혀 그 기계가 `agent.reason` 을 광고하고, 계약이 `requires: mcp.probe` 로 잡으면
실행 시점에 하네스가 없어 죽는다.

A) **오늘 그대로 둔다.** `board` 와 `arch` 가 이미 같은 자리다 — 광고가 곧
능력이라는 `ADR-012` 아래서 「하네스가 없다」는 `harness` 키의 부재로 이미
드러나고, 계약이 그것까지 요구하면 후보에서 빠진다. `capabilities` 와
`hasCapability` 에 diff 0 이라는 완료 조건도 그대로 선다

B) **`mcp.*` 만으로는 광고하지 않는다.** `hasCapability` 가 `mcp.` 접두를
능력으로 안 세도록 고친다. 하네스 없는 기계가 MCP 를 선언해도 조용하다.
대가는 완료 조건의 「`cheapAttrs` · `capabilities` 에 diff 0」이 깨지는 것이고,
그 둘을 안 건드린다는 것은 `components.md` 1.4 가 박은 값이다

C) **광고는 오늘 그대로 두고 노드 기동 로그에 경고만 낸다** —
「`mcp:` 를 선언했는데 이 기계에 하네스가 없다」. 광고 규칙은 안 바뀐다

D) Other (please describe after [Answer]: tag below)

[Answer]: B

### Question 7

**현황판과 제어판이 새 키를 어떻게 보이나.** 1.5 가 오늘 칩은 보이되 현황판이
`mcp.probe = 1` · `harness.claude = 1` 이라는 **원문 키**로 냄을 보였다.
바로 옆에 `실행 도구 = claude` 가 한글 이름으로 있어 같은 사실이 두 줄이 된다.
CA2 의 눈 검증 S1 이 이 화면이다.

A) **코드 0 으로 둔다.** 원문 키로도 칩은 보이고 S1 은 초록이다. 행렬의
`internal/api/ui` 0 과 `panel` 0 이 그대로 선다. 표시의 손질은 이 팩 밖의 일로
남기고 넘기는 것에 적는다

B) **현황판에 이름 둘을 더한다.** `format.mjs` 의 `attributeName` 이 접두를 보게
고쳐 `mcp.<이름>` 과 `harness.<이름>` 에 한글 이름을 준다. 행렬에
`internal/api/ui` 행이 늘고 그 패키지의 시험(`tests/model.test.mjs` 계열)이 닿는다.
S1 을 재는 사람이 화면에서 바로 읽는다

C) **옛 `harness` 키를 현황판에서만 숨긴다.** `technicalAttribute` 에 넣어
같은 사실이 두 줄로 안 보이게 한다. 광고와 API 는 그대로다. 대가는 화면과
`GET /v1/nodes` 가 다른 것을 보이는 것이다

D) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## 3. 답을 받은 뒤 낼 산출물

`aidlc-docs/taeels/construction/advert/functional-design/` 에 셋이다.

- [ ] `domain-entities.md` — 선언과 광고의 형식
  - [ ] `Local.MCP` 의 자리와 뜻. `enode.yaml` 의 `mcp:` 가 받는 필드와 각 뜻
  - [ ] `MCPServer` 의 필드 — 오늘 다섯과 물음 2 · 3 이 더하거나 고치는 자리
  - [ ] 종류 둘(stdio · remote)을 무엇이 가르나. 모호한 선언의 처리 (물음 1)
  - [ ] 광고 키의 형식 — `mcp.<이름>: "1"` · `harness.<이름>: "1"` · 옛 `harness`.
        값이 언제나 `"1"` 인 이유 (매처가 완전 일치를 본다 · 1.6)
  - [ ] `Fingerprinter` 의 계약 — `Kind()` 의 값 셋과 `Probe` 의 반환 (물음 4)
  - [ ] 이 유닛이 안 만드는 이름 — `Detector`(시계) · `cheapAttrs` · `capabilities`
- [ ] `business-rules.md` — 거절과 판정과 사유
  - [ ] 설정 거절 규칙과 문구 (물음 1). 그 오류가 `LoadLocal` 에서 나고
        `cmd/enode/main.go:112` 가 노드를 안 띄운다는 경로를 코드 자리로 적는다
  - [ ] 뜨나 판정 둘 — stdio 는 절대경로이거나 `PATH` resolve · remote 는
        `credential` 이름의 환경변수 존재. **연결은 안 한다**
  - [ ] 못 잡는 것 셋을 문장으로 (`decisions.md` 6절 ⑭) — 틀린 자격증명 ·
        죽은 엔드포인트 · 실행은 되나 MCP 가 아닌 바이너리
  - [ ] 안 뜨는 서버의 사유가 어디에 어떤 빈도로 남나 (물음 5)
  - [ ] 광고에 실리지 않는 것 — `credential` 의 변수 이름 · 모든 값 ·
        워크스페이스 `.mcp.json` 의 서버 (그것은 U4 이고 광고에 안 간다)
  - [ ] 순회의 동작 중립이 무엇으로 판정되나 — 기존 탐지 시험이 그대로 초록
  - [ ] 하네스 없는 노드의 광고 (물음 6)
- [ ] `business-logic-model.md` — 값이 지나가는 길
  - [ ] `enode.yaml` -> `LoadLocal` -> `Local.MCP` -> `mcpFP.Probe` -> `attrs`
        -> `POST /v1/nodes` -> 매처 -> `GET /v1/capabilities` 의 한 줄기
  - [ ] 시계 둘이 가르는 자리 — `Detector.refresh`(5분)가 `costlyAttrs` 를 부르고
        `cheapAttrs` 는 광고마다 새로 본다. CA2 의 「탐지 주기 뒤」가 이 줄이다
  - [ ] `costlyAttrs` 의 순회 — 셋의 순서와 키가 겹칠 때의 규칙, 그리고
        오류의 처리 (물음 4)
  - [ ] CA3 의 뒤 절반이 어디서 닫히나 — `requires` 의 `mcp.<이름>` 이
        `Capability.Satisfies` 의 완전 일치를 타고, 없는 키가 `422` 가 되는 경로
  - [ ] U4 · U5 가 `Local.MCP` 를 어디서 집는가 — 이음매를 이름으로

---

## 4. 이 단계가 안 만드는 것

```text
   코드            다음 단계다.  이 단계는 규칙과 형식만 낸다
   시험 목록        Code Generation 계획이 낸다
   NFR             회차 실행 계획이 SKIP 으로 닫았다 (셋 다)
   인프라           배포 변경 0
   옛 harness 를 걷는 날   이 회차가 안 정한다 (components.md 1.4)
```

---

## 5. 파장 — 이 유닛 밖으로 가는 것

```text
   행렬            2.1 의 U3 칸.  물음 2 의 답이 A 면 「U1 의 것을 안 고친다」가
                   거짓이 되므로 그 줄을 고친다.  물음 7 의 답이 B 나 C 면
                   internal/api/ui 의 0 이 함께 바뀐다

   decisions.md    6절에 실측 행 — yaml 이 잡는 것과 안 잡는 것 (1.1) ·
                   「그 밖의 키」가 오늘 안 생긴다는 것 (1.2) · env 의 갈림 (1.3).
                   2절의 env 줄은 물음 3 의 답이 A 나 B 면 함께 고친다

   component-methods.md   물음 4 의 답이 B 면 6절의 Probe 시그니처를 고친다

   features.md 3.2  물음 3 의 답이 C 면 「값은 적지 않는다」가 코드로는 안
                   지켜진다는 사실을 그 절이나 5절(되돌리기)에 적는다

   짝 팩과의 접점    0.  이 유닛은 runner.go 도 claude.go 도 안 만진다

   U4              Local.MCP 를 출처 하나로 집는다.  이 유닛이 정한 형식과
                   거절 규칙을 그대로 쓴다
   U5              팩의 서버 이름이 노드 선언 이름과 부딪치는 것을 거절한다
                   (decisions.md 6절 ⑦) — 그 「노드 선언」이 이 유닛의 Local.MCP 다
```

**CA0 의 한 값을 미리 적어 둔다.** `internal/enode` 는 U1 뒤 85.6% 이고 하한이
80% 다. 이 유닛이 그 패키지에만 코드를 더하므로 **하한에 가장 가까운 자리가
여기다** — Code Generation 이 표준 명령으로 다시 잰다.

---

## 6. 브랜치를 회차가 아니라 U2 에서 땄다

`CONVENTIONS.md` 3.1 은 유닛 브랜치를 **회차 브랜치에서** 따라고 적는다. 이
유닛은 `unit/contract-vocab` 에서 땄다. U2 가 `unit/isolation` 에서 딴 것과
같은 이유이고, 이 유닛은 그 위에 하나가 더 있다.

```text
   선행이 둘이다        unit-of-work.md 3절이 U3 의 선행을 U1 · U2 로 적는다.
                      CA3 의 앞 절반(400 · lint)이 U2 의 것이고 CA2 의
                      「먼저 서는 기능」이 U1 의 3.1 이다.  회차에서 따면
                      그 둘이 없는 나무 위에서 게이트를 재게 된다

   aidlc-state.md      같은 문서 루트(aidlc-docs/taeels/)의 파일이고 통째로
                      다시 쓰거나 자동 병합할 수 없다 (CLAUDE.md).  회차에서
                      따면 세 유닛의 절이 파일 꼬리에서 각자 자라 부딪친다
```

코드에서도 부딪히지 않는다 — 행렬이 U3 의 제품 파일을 셋
(`mcp.go` · `detect.go` · `config.go`)으로 세고, 그중 `mcp.go` 만 U1 과 겹치며
겹치는 자리가 물음 2 의 답이다. **진행자에게 넘긴다** — U1 · U2 의 PR 이 순서대로
`main` 에 들어가면 이 브랜치의 PR 차이는 U3 것뿐이다.
