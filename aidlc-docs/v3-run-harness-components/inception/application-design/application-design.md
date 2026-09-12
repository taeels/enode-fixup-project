# 응용 설계 — 하네스 구성요소 (통합본)

네 문서를 한 장으로 모은 것이다. 상세는 각 문서가 진다.

```text
   components.md            무엇이 늘고 무엇이 확장되나 · 실패 등급 표
   component-methods.md     시그니처와 오류 계약
   services.md              순서와 실패 지점 · 탐지 · 팩의 운반
   component-dependency.md  호출 방향 · 임포트 금지 · 짝 팩과의 접점
```

---

## 1. 이 설계가 한 문장으로 말하는 것

**하네스의 사적인 세계를 계장 임시 디렉터리 아래에 새로 짓고, 그 세계에 무엇을
넣을지는 실행 직전에 한 번 결정한 뒤 그 결정대로만 쓴다.**

결정과 쓰기가 갈린 것이 이 설계의 뼈대다 (Q4 = A).

```text
   결정   resolveComponents     파일을 안 만진다.  틀리면 하네스가 안 뜬다
   쓰기   Instrument            정책을 다시 판단하지 않는다.  받은 것만 쓴다
```

---

## 2. 답 다섯이 정한 것

```text
   Q1 = A   Fixed(dir string) map[string]string
            하네스별 이름(CLAUDE_CONFIG_DIR)이 어댑터 밖으로 안 나간다

   Q2 = A   계장 디렉터리를 못 만들면 단계 실패
            오늘 동작이 바뀐다.  decisions.md 에 행으로 적는다

   Q3 = B   costlyAttrs 를 탐지기 순회로 바꾸고 하네스 · 저장소 · MCP 셋을
            그 종류로 둔다.  ADR-035 §4.2 의 정본 결정 그대로다.
            이름은 Fingerprinter — Detector 는 ADR-068 의 시계가 이미 쓴다

            처음에 A 를 골랐다가 검증이 뒤집었다 (2026-09-12).  A 의 근거 둘이
            다 틀렸다 — 「§4.2 가 지키려던 값은 ADR-068 이 이미 세웠다」는
            오독이고(§4.2 의 논거는 등록부이지 광고 루프가 아니다),
            「이름이 부딪친다」는 Nomad 의 단어로 이름을 달면 사라진다.
            ADR-035:246-248 이 이 회차를 시점으로 지목한 것도 안 읽혔다

   Q4 = A   결정과 쓰기를 가른다.  정책 실패는 exec 전에 단계를 죽인다

   Q5 = A*  출력 형식의 대부분은 짝 팩의 것으로 둔다.  Decode · tee · 사건 배출은 안 건드린다

            * 2026-09-12 에 부분으로 뒤집혔다 (decisions.md 6절 ⑮).
              Argv 에 --output-format stream-json --verbose 한 줄만 가져온다 —
              그러지 않으면 CA1 · CA4 · CA5 가 재려는 system/init 줄이
              이 회차에 존재하지 않는다.  짝 팩과의 접점이 하나에서 둘로 는다
```

---

## 3. 무엇이 바뀌나 — 한눈에

| 경로 | 무엇이 | 종류 |
|---|---|---|
| `internal/enode/mcp.go` | 새 파일 — `MCPServer` · `Components` · `Pack` · `resolveComponents` · `readPack` · `mcpUp` · `mcpFP` | 새로 |
| `internal/enode/harness.go` | `Fixed` · `Instrument` 시그니처 · `HarnessResult` 필드 둘 · `errAux` | 확장 |
| `internal/enode/claude.go` | `Fixed(dir)` · `Instrument` 가 가짜 홈 · 팩 · 허용목록을 쓴다 · `Argv` 에 `stream-json --verbose` 한 줄 (⑮) | 확장 |
| `internal/enode/runner.go` | `Job.NodeMCP` · 순서와 실패 등급 | 확장 |
| `internal/enode/claim.go` | `Job` 리터럴에 `NodeMCP` 를 싣는 줄 하나 (`claim.go:765`) | 확장 |
| `internal/enode/detect.go` | `costlyAttrs` 를 `Fingerprinter` 순회로 · `break` 제거 · `harness.<이름>` | 확장 |
| `internal/enode/config.go` | `Local.MCP` · `SampleLocal` 한 줄 | 확장 |
| `internal/enode/agent.go` | `AgentParams.MCP` · `.Pack` · `parseAgentParams` 의 타입 검증 | 확장 |
| `internal/enode/hook.go` | 훅 파일이 **가짜 홈 안으로**. `self` 가 비면 `hooks` 키를 안 쓴다 | 확장 |
| `internal/contract/contract.go` | `agentKeys` 에 이름 둘 | 확장 |
| `internal/contract/grammar.go` | `agent.mcp` · `agent.pack` 을 가르치는 줄 | 확장 |
| `internal/contract/examples/mcp.json` | 예시 하나 | 새로 |
| `cmd/iapadapter/config.go` · `contract.go` | `executor.pack` · 템플릿 분기 | 확장 |
| `cmd/runctl` | **없다** — `example` 이 임베드 FS 를 돈다. `schema` 는 안 는다 | 0 |
| `internal/api` · `store` · `panel` · `api/ui` · `match` | **없다** | 0 |

표를 고친 자리 셋을 적어 둔다 (2026-09-12 검증).

```text
   claim.go · agent.go   빠져 있었다.  Job 리터럴은 제품 코드에 claim.go:765
                         한 자리뿐이고 AgentParams 는 agent.go:36 · 500 이다
   runctl 의 근거         diff 0 은 맞고 근거가 틀렸다.  Step.Agent 가
                         map[string]interface{} 라(contract.go:447)
                         schema steps 는 agentKeys 가 늘어도 한 글자도 안 는다.
                         새 계약 키를 사람에게 가르치는 경로는
                         Grammar 와 examples/mcp.json 둘뿐이다
   internal/match         분류에서 빠져 있었다.  코드 diff 는 0 이지만
                         동작은 6.4 가 따로 잰다
```

---

## 4. 설계가 답 밖에서 더 정한 것

답 다섯이 안 닫은 자리다. **전부 조용히 열리는 길을 막는 것**이고, 그것이
이 회차가 사내 실측에서 고치려는 실패와 같은 종류다.

4.1 ~ 4.2 는 처음부터 있었고 **4.3 ~ 4.6 은 2026-09-12 검증이 더했다.**

### 4.1 `Instrument` 의 오류를 등급으로 가른다

오늘 `Instrument` 의 오류는 삼켜진다 — 「계장은 보조이고 진짜 안전망은
워크스페이스 diff 다」가 그 근거였고, 그 판단은 훅에 대해서는 지금도 맞다.

그런데 이제 같은 메서드가 **정책의 산물**도 쓴다. 허용목록이 안 쓰이면 계약이
요청한 서버가 조용히 없고, 팩이 안 펴지면 스킬이 조용히 없다. 둘 다 `exit 0` 으로
성공이 봉인되는 길이다.

**기본이 치명이다.** 처음 설계는 반대였고 (「`errComponents` 로 감싼 오류만 치명」)
2026-09-12 검증이 방향을 뒤집었다.

```text
   기본                        치명.  가짜 홈 · 허용목록 · 팩 · 자격증명 복사 —
                               Instrument 안의 오류 반환 여덟 자리가 여기다
   errAux 로 감싼 오류          보조.  훅 설정(<tmp>/home/settings.json)의 Marshal · WriteFile.
                               훅을 보조로 두는 오늘 판단을 명시로 유지한다
```

**왜 뒤집었나 — 실수의 방향이다.** 기본이 보조이면 여덟 자리 중 하나를 안 감싸는
실수가 **열리는 쪽으로** 틀린다. 허용목록이 안 쓰였는데 단계가 돌고 `exit 0` 으로
성공이 봉인된다 — 이 회차가 고치려는 실패와 같은 모양이다. 기본을 치명으로 두면
감쌀 자리가 여덟에서 둘로 줄고 빠뜨림이 **닫히는 쪽으로** 틀린다.

이 저장소는 같은 선택을 이미 한 번 했고 이쪽으로 골랐다 — `env.go` 가 블랙리스트
대신 화이트리스트를 쓰는 근거(`ADR-034` §2.1 「기본이 통과면 조용히 안 걸린다」)와
`requirements.md` 4.2 의 SECURITY-15 「실패는 닫히는 쪽으로」다.

`errors.Is` 하나로 갈린다. Q4 의 B(전부 치명)가 훅의 기존 판단을 말없이 뒤집는
것과, C(새 메서드)가 두 번째 하네스 없이 인터페이스를 늘리는 것을 둘 다 피한다.

### 4.2 `Instrument` 를 언제나 부른다

오늘은 `os.Executable()` 이 빈 문자열이면 계장을 **통째로** 건너뛴다. 그 경로로
가면 허용목록이 안 쓰이고 팩이 안 펴진다 — 4.1 이 막으려는 바로 그 침묵이
`self` 하나로 다시 열린다.

`self` 가 비면 훅 블록만 빼고 나머지(게이트웨이 인증 필드 · 허용목록 · 팩)는
그대로 쓴다. 훅이 없는 것은 보조 실패다.

**「훅 블록만 뺀다」를 한 줄로 못 적는다.** `hook.go:291` 이
`cmd := []string{self, "hook", "stop", ...}` 로 시작하고 `shellJoin` 이 빈 첫
원소를 그대로 잇는다. 그냥 부르면 설정에 **앞이 빈 명령**이 실리고 하네스가 매
Stop 마다 그것을 셸에 넘긴다. `WriteHookSettings` 가 `self == ""` 일 때
**`hooks` 키 자체를 안 쓴다.** 함수를 통째로 건너뛰는 것은 답이 아니다 —
`gatewayAuthFields()` 는 그대로 얹혀야 한다.

### 4.3 `Instrument` 의 불변식 셋

Q1 = A(`Fixed(dir)`)가 성립하려면 겉면 밖에 불변식 **셋**이 필요하다. 안 적으면
`ADR-034` §2.1 의 「계장 실패와 독립」이 새 순서에서 조용히 깨진다.

```text
   ①  <dir>/home 을 가장 먼저 만든다
      보조 등급 오류로 중간에 나가면 Fixed(tmp) 가 없는 경로를 가리킨다.
      CLAUDE_CONFIG_DIR 이 없는 경로일 때 하네스가 무엇을 하는지는 팩의 실측 표에
      없다 — 진짜 홈으로 되돌아가면 격리가 조용히 풀린다

   ②  보조 등급일 때도 이미 얻은 플래그를 돌려준다
      오늘 runner.go:77-80 은 if err == nil 일 때만 flags 를 붙인다.
      그대로 두면 훅 쓰기 실패 하나가 --strict-mcp-config 와
      --setting-sources "" 를 함께 떨어뜨린다.  4.6 이 그 값을 잰다

   ③  보조 오류로 조기 반환하지 않는다
      훅 쓰기가 실패해도 남은 쓰기(팩 · 허용목록)를 끝까지 하고
      마지막에 errAux 로 감싸 돌려준다.  조기 반환하면 허용목록이
      아예 안 쓰이고 치명도 안 난다 — 3절 표가 치명으로 잡으려던
      「안 쓰이면 요청한 서버가 조용히 없다」가 보조 경로로 되돌아온다
```

**③ 이 ⑧ 과 함께 열린 자리다.** 훅 설정을 가짜 홈 안으로 옮기면서 보조 쓰기가
순서상 앞으로 당겨졌다 (`services.md` 1.2 — `home/settings.json` 이 `mcp.json`
보다 앞이다). 그래서 ② 의 「이미 얻은 플래그」라는 전제가 혼자서는 안 선다 —
훅이 먼저 실패하면 얻은 플래그가 아직 없다. **쓰기를 끝까지 하는 것이 그 전제를
세운다.**

### 4.4 팩의 서버도 `agent.mcp` 필터를 탄다

`features.md` 3.2 · `requirements.md` FR-2 · `services.md` 1.1 은 「팩의 서버는
팩 `mcp.json` 에 적힌 것 **전부**」로 적었고, 같은 회차의 `requirements.md` 4.2
SECURITY-06 · 4.3 · 6.3 SECURITY-08 은 「**요청한 이름만** · 요청 안 한 서버 0」
으로 적었다. **둘이 정반대이고 정한 자리가 없었다.**

```text
   정한다   팩의 서버도 agent.mcp 가 적은 이름만 실린다.
            팩은 정의의 출처이지 허가의 출처가 아니다
   근거     앞이 참이면 계약 작성자가 이름을 안 적고도 임의의 stdio 서버를
            물릴 수 있다.  --strict-mcp-config 도 가짜 홈도 그것을 안 막는다 —
            허용목록 자체가 싣기 때문이다.  SECURITY-06 의 「와일드카드가 없다」와
            정면으로 어긋난다
   대가     팩이 서버를 싣고도 계약이 이름을 안 적으면 안 뜬다.
            Notes 에 「팩이 실었으나 요청되지 않음」으로 남긴다
```

### 4.5 팩이 노드 선언을 이길 때 이름을 가린다

`decisions.md` 2절의 우선순위(팩 · 노드 · 워크스페이스)는 그대로 둔다. 다만
**소유권이 뒤집히는 경로**를 막는다.

```text
   경로     계약이 requires: mcp.gerrit 으로 소유자 선언을 보고 노드를 고른 뒤
            자기 팩의 gerrit 정의로 그 이름을 덮어 실행한다.
            매칭은 소유자 값으로 하고 실행은 계약 값으로 한다

   막는다   노드가 선언한 이름을 팩이 덮으면 거절한다 —
            pack redefines node-declared mcp server <이름>
            팩이 새 이름을 싣는 것은 그대로 허용한다
   근거     SECURITY-08 의 「노드 선언은 소유자만 쓰는 설정 파일에 산다」가
            이 경로로 무력해진다.  requirements.md 4.3 의 오용 시나리오에 없던 줄이다
```

### 4.6 격리의 겹은 실제로 하나다

`services.md` 1.2 가 계정 커넥터 0 의 근거를 「겹 둘 — 가짜 홈과 strict」로 적었다.
**첫 겹이 실제로 끊는지 잰 적이 없다.**

```text
   쟀다          features.md 1.2 의 실측표에 「가짜 홈만 -> 계정 커넥터 차단」 행이 있다.
                 다만 그 홈은 로그아웃 상태였다
   안 쟀다        이 설계는 FR-1 대로 .credentials.json 을 가짜 홈에 복사한다.
                 로그인된 가짜 홈에서도 커넥터가 안 보이는지는 잰 적이 없다
   확실한 겹      --strict-mcp-config 는 계정 상태와 무관하게 목록 밖을 안 띄운다
                 (ADR-067 §2.1 「항상 켠다」)
```

**겹은 둘로 두고, 첫 겹을 재측정 항목으로 남긴다.** 「하나뿐이다」는 측정이 아니라
가설이었다 — 로그인된 홈에서 안 재고 내린 결론을 차단 판정의 근거로 쓰면 안 된다.

```text
   지금 아는 것    strict 는 확실하다.  가짜 홈은 로그아웃 상태에서 확실하다
   재야 하는 것    로그인된 가짜 홈에서 커넥터가 보이는가.  CA1 이 그것을 잰다
   결과가 「보인다」면  겹이 하나로 준다.  그때 SECURITY-11 을 다시 판정한다
```

그리고 `--strict-mcp-config` 가 `Instrument` 의 반환 플래그에 실려 있으므로
**4.3 ②가 그 겹을 보조 실패에서 지키는 줄이다.**

---

## 5. 요구와의 대조 — FR 일곱이 어디 앉나

| 요구 | 앉는 자리 | 게이트 |
|---|---|---|
| FR-1 가짜 홈 | `Fixed(dir)` + `Instrument` 의 `<dir>/home` · `.credentials.json` 복사 | CA1 |
| FR-2 허용목록 | `resolveComponents` (결정 · 4.4 의 필터 포함) + `Instrument` 의 `<dir>/mcp.json` (쓰기) | CA1 · CA4 |
| FR-3 선언과 광고 | `Local.MCP` · `mcpUp` · `mcpFP` · `Fingerprinter` 순회 | CA2 · CA3 |
| FR-4 워크스페이스 | `resolveComponents` 안의 `<cwd>/.mcp.json` 읽기 | CA4 · CA5 |
| FR-5 계약 문법 | `agentKeys` · `Grammar` · `AgentParams` · `examples/mcp.json` | CA3 · CA4 |
| FR-6 팩 | `readPack` (검증) + `Instrument` (펴기) + `executor.pack` (출처) | CA5 |
| FR-7 기록 | `HarnessResult.MCP` · `HarnessResult.Pack` | CA6 |

---

## 6. NFR 과의 대조

### 6.1 차단 게이트 다섯

```text
   crypto/tls · net/http 심볼 (enodectl.exe)   이 회차가 enodectl 을 안 건드린다.  영향 0
   패키지별 커버리지 80%                        internal/enode 가 걸린다.  아래 6.2
   허용목록 밖의 스킵 0                          새 스킵을 안 만든다
   U+2605 을 담은 파일 0                        emphasis-check.py 가 문서마다 돈다
```

### 6.2 커버리지를 하네스 없이 채운다

이 설계가 그것을 가능하게 만든 자리가 **결정과 쓰기의 분리**다.

```text
   resolveComponents   파일도 프로세스도 안 쓴다.  Job 하나를 넣고 오류를 잰다
   readPack            io.Reader 하나.  악성 tar 를 메모리에서 지어 넣는다
   mcpUp               LookPath · LookupEnv.  PATH 와 환경변수를 시험이 세운다
   mcpFP.Probe         Local 하나를 넣고 속성 맵을 잰다
   Fixed(dir)          순수 함수.  문자열 비교다
   Instrument          t.TempDir() 에 쓰고 난 파일을 읽는다
```

**하네스 실행파일이 필요한 것은 하나도 없다.** 가짜 stdio 서버(`true`)조차
단위 시험에는 필요 없다 — 실제 연결을 안 하기 때문이다.

### 6.3 security-baseline 준수

| 규칙 | 판정 | 이 설계의 근거 |
|---|---|---|
| SECURITY-05 입력 검증 | 준수 | `readPack` 이 절대경로 · `..` · 심볼릭 링크 · 상한을 거부하고 **파일을 쓰기 전에** 한다 |
| SECURITY-06 최소 권한 | 준수 | 허용목록은 요청한 이름만 연다. 와일드카드가 없다 |
| SECURITY-08 접근 제어 | 준수 | 노드 선언이 소유자만 쓰는 파일에 산다. 계약이 요청 안 한 서버는 0 |
| SECURITY-09 하드닝 | 준수 | 가짜 홈이 기본값이다. 여는 값이 없고, 못 지으면 단계가 실패한다 (Q2 = A) |
| SECURITY-11 보안 설계 | 준수 (재측정 대기) | 규칙이 요구하는 것은 **단일 방어선이 아닐 것**이다. 겹 둘(가짜 홈 · strict)로 설계하고 4.3 ②가 strict 를 보조 실패에서 지킨다. 다만 첫 겹은 로그아웃 홈에서만 쟀다 (4.6) — **CA1 이 로그인된 홈에서 재고, 거기서 커넥터가 보이면 겹이 하나가 되므로 이 판정을 다시 낸다** |
| SECURITY-12 자격증명 | N/A | **규칙 오태그였다** (5차 검증). `security-baseline` 의 SECURITY-12 는 사용자 인증 규칙이다 — 비밀번호 정책 · 적응형 해싱 · MFA · 세션 쿠키 · 무차별 대입 · 소스에 자격증명 하드코딩. 계장의 자격증명 파일 복사는 여섯 중 어디에도 안 걸린다. 그 요구의 근거는 `features.md` 3.1 의 보안 요구와 SECURITY-09 · 15 다. 복사본이 `0600` · 계장 디렉터리 안 · 단계 끝 삭제라는 값은 그대로 산다 |
| SECURITY-13 무결성 | 준수 (기록) | 실행 전 검증 대신 기록으로 족하다. 근거 셋은 `decisions.md` 6절 ③ |
| SECURITY-15 예외 처리 | 준수 (⑯ 뒤) | `components.md` 3절의 실패 등급 표. 치명이 전부 exec 앞에 모인다. **⑮ 이 exec 뒤의 봉투 해석에서 이 규칙을 한 번 깼고 ⑯ 이 그것을 막는다** — `type` 이 `"result"` 가 아니면 `ReasonError` 라 크래시가 성공으로 안 봉인된다 |
| SECURITY-03 로깅 | 준수 (⑱ 뒤) | 노드 로그는 이름만 찍는다 (`components.md` 4절). **그런데 ⑮ 가 `logs/` 의 내용을 바꾼다** — `stream-json` 아래 stdout 은 봉투 하나가 아니라 사건 전부이고 거기 도구 입력과 도구 결과가 실린다. `logs/NN-*.log` 는 `ADR-005` 성질 2 로 봉인되어 이후 안 바뀐다. 에이전트가 가짜 홈의 `.credentials.json` 을 한 번 읽으면 그 내용이 봉인 기록에 영구히 남는다 — ⑩ 이 디스크 경계로 막은 값을 다른 문으로 낸다. ⑮ 이 넓힌 몫을 **⑱ 이 닫는다** — **허용목록이다** — `system/init` 과 최종 `result` 와 stderr 는 전문이고 그 밖의 모든 사건은 껍데기(사건 종류 · 도구 이름 · 성공 여부)만 남는다. **경로에 예외가 없다.** 에이전트가 파일을 *읽기만* 해도 새던 길이 닫히고 오늘처럼 *스스로 말해야* 남는다. 남는 잔여는 이 팩보다 먼저 있던 표면이라 `decisions.md` 5절 되돌리기로 올렸다 |
| SECURITY-01 · 07 | 기록만 | 기존 코드 사실. 새 포트 0 · 새 전송 0 |
| SECURITY-10 공급망 | **새 표면** | 팩은 계약이 적은 주소에서 받아 하네스 홈에 펴는 외부 산출물이고 그 안에 실행 경로를 담은 서버 정의가 있다. `readPack` 의 SEC-A 와 4.4 의 필터가 그 표면을 좁히고, 실행 전 다이제스트 검증 대신 기록으로 둔 근거는 `decisions.md` 6절 ③ 이다 |
| SECURITY-02 | N/A | 네트워크 중간자 경로를 안 만든다. 새 포트 0 · 새 전송 0 |
| SECURITY-04 | N/A | 웹 표면을 안 만든다. `internal/api/ui` diff 0 |
| SECURITY-14 | N/A | 알림 경로를 안 만든다 |

### 6.4 매처는 코드가 0 인데 동작은 안 쟀다

`internal/match` 는 코드 diff 가 0 이다. 그런데 **매처의 유일한 정렬 기준이 노드가
광고한 attr 의 개수**이고(`match.go:36-42` · `:81-87`), 공용 R/E 가 다섯 자리에서
「attrCount 휴리스틱은 장식 attr 이 광고되면 깨진다」를 경고한다.

```text
   이 팩이 더하는 키   harness.<이름>   오늘 하네스가 하나뿐이라 harness 와
                                      정보량이 같다 — match.go 의 정의로 장식용이다
                     mcp.<이름>       노드마다 개수가 다르다.  희소한 노드가
                                      뒤로 밀리므로 방향은 유리하다

   판정              오늘 깨지는 장면을 못 찾았다.  그래도 「매처 변경 0」은
                     코드에 대해서만 참이다.  CA3 이 배정 결과를 보므로
                     그 조각이 이 동작을 함께 잰다
```

**옛 `harness` 키를 걷는 날 이 줄을 다시 본다.** 그때 attrCount 가 노드마다 1 씩
줄고 정렬이 움직인다.

---

## 7. 이 단계가 `decisions.md` 에 더하는 행

`requirements.md` 7절이 낸 셋은 이미 실렸다 (`decisions.md` 6절). **이 단계가
④ ~ ⑤ 를 더했고, 2026-09-12 의 검증이 ⑥ ~ ⑩ 을 더했다 — 모두 열이다.**

```text
   ④  계장 디렉터리를 못 만들면 단계 실패      Q2 = A.  오늘은 그대로 기동한다.
                                             가짜 홈이 앉을 자리가 없는 채로 띄우는 것은
                                             SECURITY-09 와 정면으로 어긋난다

   ⑤  탐지기 순회로 바꾼다                     Q3 = B.  ADR-035 §4.2 의 정본 결정 그대로다.
                                             이름만 Fingerprinter 로 단다 — Detector 는
                                             ADR-068 의 시계가 이미 쓴다.
                                             벗어남이 아니라 정본을 따르는 행이다

   ⑥  팩의 서버도 agent.mcp 필터를 탄다        4.4.  문서 둘이 정반대였고 이쪽으로 닫았다

   ⑦  팩이 노드 선언 이름을 덮으면 거절한다      4.5.  소유권이 뒤집히는 경로를 막는다

   ⑧  훅 설정을 가짜 홈 안으로 옮긴다           features.md 3.1 · constraints.md 접점 절의
                                             요구 그대로다.  앞선 판(가짜 홈 옆)이
                                             근거 없이 요구를 뒤집고 있었다

   ⑨  Instrument 의 실패 등급은 기본이 치명이다   4.1.  빠뜨림이 닫히는 쪽으로 틀린다

   ⑱  도구 사건은 껍데기만 남긴다             ⑮ 이 넓힌 누출 표면을 닫는다.
                                             경로에 예외가 없다.  runner.go 가 선별한다

   ⑮  Argv 에 stream-json --verbose 한 줄        게이트가 복제본 대신 실물을 잰다.
                                             정본 MVP 값을 R4 없이 뒤집는다
   ⑯  ParseClaude 가 봉투의 type 을 본다      ⑮ 이 연 fail-open 을 막는다.
                                             크래시가 성공으로 봉인되던 길이다

   ⑩  계장 디렉터리 보존 스위치를 안 둔다        한 번 넣었다가 뺐다.  features.md 3.1 의
                                             보안 요구(복사한 자격증명은 단계 끝에
                                             지워진다)를 뚫기 때문이다.
                                             눈 검증이 복제본을 잰다는 한계는 남고
                                             그것을 게이트에 명시로 적는다
```

---

## 8. Units Generation 에 넘기는 것

**유닛 분해는 여기서 안 한다.** 넘기는 것은 세 가지다.

```text
   파일 목록      3절의 표.  cmd/runctl 이 0 이라 만지는 경로가 셋이다
                 (claim.go · agent.go 가 표에 늘었다 — 패키지는 그대로 셋이다)
   착수 순서의 뿌리  scene-gates.md 2절의 「먼저 서는 기능」 열.
                  3.1 가짜 홈이 3.2 · 3.3 · 3.4 · 3.5 · 3.6 의 앞이다
   접점          runner.go 와 claude.go 의 Argv 둘이 짝 팩과 겹친다 (⑮ · 5절).
                 파일 행렬이 이 줄을 그대로 받는다
```

**빌드 시점 의존 하나** — `internal/contract` 의 `agentKeys` 가 서기 전에는
`agent.mcp` 나 `agent.pack` 을 적은 계약이 `400` 이다. 그 키를 쓰는 유닛보다
먼저 서야 한다. 반대로 가짜 홈과 허용목록의 뼈대는 계약 키 없이 설 수 있다 —
CA1 이 아무것도 요청하지 않은 단계를 재기 때문이다.
