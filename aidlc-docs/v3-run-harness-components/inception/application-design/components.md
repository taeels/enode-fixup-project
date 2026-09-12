# 컴포넌트 — 무엇이 늘고 무엇이 확장되나

정본은 `requirements/harness-components/` 팩이고 이 문서는 그 요구가 앉을 **자리와
책임**을 정한다. 비즈니스 규칙은 유닛별 Functional Design 이 짓는다.

**새 패키지는 0 이다.** 임포트 금지 넷의 표에 줄이 늘지 않는다
(`constraints.md` 구조 불변식).

---

## 1. 확장하는 기존 컴포넌트 여섯

### 1.1 `Harness` 인터페이스 — `internal/enode/harness.go`

어댑터가 순수 함수만 내놓고 실행은 `runner.go` 가 한다는 갈래는 안 바뀐다.
**두 메서드의 시그니처가 바뀐다** (Q1 · Q4 의 답).

```text
   Fixed(dir string) map[string]string          인자가 는다 (Q1 = A)
                                                dir 은 계장 임시 디렉터리다
   Instrument(dir, self string,                 인자가 는다 (Q4 = A)
              a HookArgs, c Components)         c 는 이미 해소된 구성요소다
              ([]string, error)                 이 메서드가 결정을 안 한다
```

책임 — `Fixed` 는 「노드 환경에 무엇이 있든 우리가 정하는 값」이고, 이제 그중
하나가 계장 디렉터리 아래를 가리킨다. `Instrument` 는 「하네스의 사적인 세계를
파일로 짓고 플래그를 낸다」로 자란다 — 훅 설정 · 가짜 홈 · 팩 · 허용목록.

**두 번째 하네스가 없으므로 이주 비용이 0 이다** (`agent-runtime` R6).

### 1.2 `claudeHarness` 어댑터 — `internal/enode/claude.go`

```text
   Fixed(dir)      CLAUDE_CODE_DISABLE_AUTO_MEMORY=1  (오늘 그대로)
                   CLAUDE_CONFIG_DIR=<dir>/home       (새로)
   Instrument      ① 가짜 홈 <dir>/home 을 가장 먼저 짓는다 (application-design.md 4.3 ①)
                   ② 훅 설정을 그 안의 settings.json 으로 쓴다 — 오늘은 <dir> 옆이다
                   ③ Components 의 팩 파일을 그 아래 편다
                   ④ Components 의 서버 맵을 <dir>/mcp.json 으로 쓴다
                   ⑤ 실제 홈의 .credentials.json 이 있으면 0600 으로 복사한다
                   플래그가 는다 — --strict-mcp-config --mcp-config=<경로>
                   보조 실패에도 이미 얻은 플래그를 돌려준다 (4.3 ②)
   Argv            --output-format stream-json --verbose 로 바꾼다 (decisions.md 6절 ⑮).
                   그때 하네스 단계의 링 tee 를 끈다 (⑲ · runner.go:103-105) —
                   그 tee 가 logs/ 선별 앞이라 안 끄면 누출이 링으로 간다
                   게이트가 재는 system/init 줄이 그래야 나온다.
                   Decode 는 안 건드린다 — 짝 팩의 것이다.
                   하네스 단계의 tee 는 이 팩이 끈다 (⑲)
   Decode          안 바뀐다
```

`--mcp-config` 는 **등호 형태**다. 가변인자라 공백으로 주면 뒤따르는 인자를
삼킨다 (`ADR-034` §3 ②).

**인증 경로가 둘이라는 것을 이 컴포넌트가 안다.** `gatewayAuthFields()` 는
`os.UserHomeDir()` 를 읽으므로 가짜 홈과 무관하게 그대로 돈다 — 게이트웨이
노드의 인증은 안 끊긴다. 가짜 홈이 끊는 유일한 경로가 OAuth 의
`.credentials.json` 이고 ④ 가 그것을 받는다.

### 1.3 `runHarness` — `internal/enode/runner.go`

유일한 exec 지점이라는 것은 안 바뀐다. **순서와 실패 규칙이 바뀐다.**

```text
   계장 디렉터리    if 블록 안에서 살던 tmp 를 함수 몸통으로 끌어올린다.
                   못 만들면 단계 실패다 (Q2 = A) — 오늘은 그냥 기동했다
   해소            exec 전에 resolveComponents(j) 를 부른다 (Q4 = A).
                   오류면 하네스를 안 띄우고 그 사유로 단계를 실패로 낸다
   Instrument      언제나 부른다.  os.Executable() 이 비어도 건너뛰지 않는다
   실패 등급        기본이 치명이다.  errAux 로 감싼 훅 설정 쓰기 하나만 보조다
   logs/ 조립      허용목록이다 (⑱) — system/init 과 최종 result 는 전문이고
                   stderr 도 전문이다.  그 밖의 모든 사건은 껍데기만 남긴다
                   (사건 종류 · 도구 이름 · 성공 여부).  assistant 의 text 도
                   thinking 도 도구 결과도 같다.  init 이 나오는 경로에서
                   첫 줄은 언제나 그것이다
   Fixed           h.Fixed(tmp) 로 부른다
```

### 1.4 탐지 — `internal/enode/detect.go` · `detector.go`

`Detector`(시계)는 **안 건드린다.** 그것은 `ADR-068` 이 세운 주기 장치이고,
광고 루프가 탐지에서 멈출 수 없다는 값은 이미 서 있다.

**`ADR-035` §4.2 는 다른 것을 말한다** — 탐색기를 하네스 밖에 세우고 하네스도 그
한 종류로 만든다. 그 값은 아직 안 서 있고 **이 회차가 세운다** (Q3 = B).
`ADR-035:246-248` 이 「`ADR-034` 를 구현하면서 `detect.go` 를 탐지기 순회로 바꾸는
것까지는 오늘 동작을 안 바꾸고 된다」로 이 시점을 지목했다.

```text
   Fingerprinter   새 인터페이스.  Kind() string ·
                   Probe(ctx, l Local, log *slog.Logger) (map, error)
                   logger 를 받는 이유 — FR-3 의 서버별 누락 사유가 갈 자리다
                   이름이 Detector 가 아닌 이유 — ADR-068 의 시계가 그 이름을 쓴다.
                   Nomad 의 단어를 따른다 (§4.2 가 든 유비다)
   costlyAttrs     셋을 순회한다 — harnessFP · repoFP · mcpFP.  동작 중립이다
   하네스 순회      break 가 저절로 걷힌다 (§4.3 의 부수 효과).  harness.<이름>: "1" 을 싣고
                   옛 harness: <이름> 도 같이 싣는다
   cheapAttrs      안 건드린다
```

**옛 `harness` 키를 걷는 시점은 이 회차가 안 정한다.** 「이 회차 동안」으로 적었던
것을 되돌린다 — 팩도 `ADR-035` §6 도 기간을 안 박았고, 걷는 날 매처의 attrCount 가
노드마다 1 씩 줄어 정렬이 움직인다 (`application-design.md` 6.4).

**옛 키를 같이 싣는 것이 되돌리기를 싸게 만든다.** 새 키만 싣고 옛 키를 걷으면
오늘 도는 계약의 매칭이 같은 배포에서 끊긴다.

### 1.5 `Local` — `internal/enode/config.go`

`mcp:` 맵 하나가 는다. 「그 기계에서만 아는 것은 그 기계에만」(`ADR-012`)의
자리이고, 소유자만 쓰는 설정 파일에 산다 (`ADR-063`).

### 1.6 계약 어휘 — `internal/contract` · `internal/enode/agent.go` · `harness.go`

```text
   agentKeys          mcp · pack 두 이름을 더한다.  검증기는 이미 있다 (ADR-057)
   Grammar            agent.mcp · agent.pack 을 계획에게 가르치는 줄을 더한다.
                      계획이 안 적으면 그 단계는 팩 없이 돈다
   AgentParams        MCP []string · Pack string
   HarnessResult      MCP []string · Pack string — Record 에 남는다
   examples/mcp.json  붙여넣으면 도는 예시 하나
```

---

## 2. 새로 서는 겉면 다섯

전부 `internal/enode` 안이다. 실행 층이기 때문이다.

### 2.1 노드 MCP 선언 — `MCPServer`

`enode.yaml` 의 `mcp:` 가 읽히는 타입이다. stdio 와 remote 두 모양을 한 구조체가
받는다 — 하네스가 읽는 형식과 같아서 옮겨 적기가 복사가 된다 (`ADR-035` §4.4).

**책임** — 형식을 든다. 판정도 광고도 안 한다.

### 2.2 뜨나 판정 — `mcpUp`

```text
   stdio    command 가 절대경로로 있거나 PATH 에서 resolve 된다
   remote   credential 이름의 환경변수가 노드 환경에 있다
   공통     실제 연결은 안 한다.  남의 서버를 광고마다 두드리지 않는다
```

**책임** — 「지금 뜨겠는가」 하나. 프로세스를 안 띄운다.

비싼 쪽에 앉는 이유는 **선언 수만큼 파일시스템을 훑기 때문**이다. 「`PATH` 를
훑어서」가 아니다 — `cheapAttrs` 도 `detectArch` 로 `LookPath` 를 두 번 돈다
(`detect.go:214-227`). 싼 쪽과 비싼 쪽을 오늘 실제로 가르는 것은 **프로세스 기동**
(`Usable()` · `DetectRepo()`)이고, `mcpUp` 은 그 선을 안 넘지만 **호출 수가 노드
설정에 비례**해서 비싼 쪽에 둔다. 그것이 `requirements.md` 4.4 의 확정이다.

### 2.3 허용목록 해소 — `resolveComponents`

**이 회차의 정책이 전부 여기 있다.** 파일을 하나도 안 만진다.

```text
   합친다    셋 다 agent.mcp 가 적은 이름만 집는다 — 팩 mcp.json · 노드 mcp: ·
             워크스페이스 .mcp.json.  팩도 필터를 탄다 (application-design.md 4.4)
   이긴다    이름이 겹치면 팩 · 노드 · 워크스페이스 순으로 앞이 이긴다.  겹침은 Notes 에
   거절한다  요청한 이름이 셋 어디에도 없으면 오류다 —
             mcp server <이름> is not available on this node
   거절한다  노드가 선언한 이름을 팩이 덮으면 오류다 (4.5) —
             pack redefines node-declared mcp server <이름>
   비운다    요청이 없으면 빈 목록이다.  팩이 서버를 실었어도 그렇다.  0 은 의도다
```

**책임** — 결정. 파일을 안 만지므로 시험이 싸고, 오류가 곧 단계 실패다.

### 2.4 팩 읽기 — `readPack`

tar 를 **검증하면서 메모리로 읽는다.** 쓰지 않는다 — 쓰는 것은 어댑터의 몫이라
거부가 파일을 남기기 전에 일어난다.

```text
   거부한다   절대경로 항목 · .. 를 담은 항목 · 심볼릭 링크 항목 ·
             크기 상한 초과 · 개수 상한 초과            (SEC-A)
   건너뛴다   settings.json 은 이름으로.  팩이 정책을 덮는 것을 막는다
   무시한다   skills/ · agents/ · mcp.json 밖의 것.  이름만 Notes 에 남긴다
   센다      전체 tar 의 sha256.  Record 에 남는다 (FR-7)
```

**거부는 조용하지 않다** — 그 단계를 실패로 보고한다. 상한의 실제 값은 팩 유닛의
Functional Design 이 닫는다.

### 2.5 기록 — `HarnessResult` 의 두 필드

봉인된 묶음만 보고 그 단계가 어느 서버와 어느 팩으로 돌았는지 안다
(`ADR-005` 성질 4).

```text
   mcp   허용목록에 실제로 실린 서버 이름들.  요청한 것이 아니라 실린 것
   pack  팩 tar 의 sha256.  팩이 없으면 빈 값
```

하네스 자신의 증언(`init` 줄의 `mcp_servers`)은 짝 팩의 몫이다. 이 팩은 **우리가
무엇을 줬는가**만 적는다.

---

## 3. 실패 등급 — 무엇이 단계를 죽이나

`requirements.md` 4.2 의 SECURITY-15(「실패는 닫히는 쪽으로」)를 코드의 자리마다
값으로 옮긴 표다.

**기본이 치명이다** (`application-design.md` 4.1). 표에서 「보조」로 적힌 하나만
`errAux` 로 감싸고 나머지는 감싸는 일 없이 치명으로 올라간다 — 빠뜨림이 닫히는
쪽으로 틀리게 하려는 것이다.

| 자리 | 실패 | 등급 | 근거 |
|---|---|---|---|
| 계장 디렉터리 생성 | `MkdirTemp` 오류 | **치명** | 가짜 홈이 앉을 자리가 없다. 그대로 기동하면 개인 설정과 계정 커넥터가 다 보인다 (Q2 = A) |
| 구성요소 해소 | 요청한 이름이 없다 | **치명** | FR-2 확정. 조용히 빼면 exit 0 으로 성공이 봉인된다 |
| 팩 읽기 | 경로 · 크기 · 개수 거부 | **치명** | SEC-A. `ADR-035` §3 「없음이 실패보다 나쁘다」 |
| 가짜 홈 짓기 | 디렉터리 · 파일 쓰기 오류 | **치명** | 기본값이다. 감쌀 것이 없다 |
| 허용목록 쓰기 | 파일 쓰기 오류 | **치명** | 같다. 안 쓰이면 요청한 서버가 조용히 없다 |
| 팩 펴기 | 파일 쓰기 오류 | **치명** | 같다. 스킬이 조용히 없는 채로 돈다 |
| 구성요소 해소 | 팩이 노드 선언 이름을 덮는다 | **치명** | 4.5. 매칭과 실행이 다른 값을 쓰는 것을 막는다 |
| 자격증명 복사 | 원본이 없다 | 정상 | FR-1 확정 — 없으면 아무것도 안 한다 |
| 자격증명 복사 | 있는데 복사 실패 | **치명** | OAuth 노드에서 `Not logged in` 으로 늦게 죽는다. 늦게 아는 실패를 앞으로 당긴다 |
| 훅 설정 쓰기 | 오류 | 보조 | `errAux` 로 감싼다. **감싸는 자리는 이것 하나다** (`decisions.md` 6절 ⑪). 진짜 안전망은 워크스페이스 diff 다 |
| 기준 시각 파일 | 오류 | (이 표 밖) | `writeStamp` 는 `Instrument` 앞(`runner.go:71-76`)에서 불리고 오류가 이미 버려진다. 등급을 매길 자리가 아니다 |
| MCP 뜨나 | 안 뜬다 | 정상 | 광고에서 빼고 이유를 노드 로그에 남긴다 (FR-3) |

**치명은 전부 exec 전이다.** 하네스가 뜬 뒤에 이 표의 어떤 줄도 발동하지 않는다.

**보조 실패도 플래그는 안 떨어뜨린다.** 훅을 못 써도 `--strict-mcp-config` 와
`--setting-sources ""` 는 붙어야 한다 — 그 둘이 격리의 확실한 겹이기 때문이다
(`application-design.md` 4.3 ② · 4.6).

---

## 4. 새 로그가 찍는 것 — 이름만 (SECURITY-03)

```text
   빠진 MCP        mcp.<이름> 이 광고에서 빠졌다 · 사유
   겹친 이름       어느 출처가 이겼나
   안 실린 팩 서버   팩이 실었으나 agent.mcp 가 요청 안 한 이름 (4.4)
   무시한 팩 파일   tar 안에서 안 읽은 항목의 이름
   거부한 팩 항목   무엇을 왜 거부했나
```

값은 안 찍는다. 자격증명의 **환경변수 이름**은 노드 자신의 로그에만 나오고
광고에는 안 실린다 (`ADR-035` §8 · FR-3 확정).
