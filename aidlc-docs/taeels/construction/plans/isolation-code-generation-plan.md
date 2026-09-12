# U1 `isolation` — Code Generation 계획

**유닛** `isolation` · **브랜치** `unit/isolation` · **담당** taeels ·
**닫는 게이트** CA1 (+ CA0) · **선행** 없음

이 계획이 이 단계의 정본이다. 값은 Functional Design 셋이 이미 닫았고
(`construction/isolation/functional-design/`) 여기는 **그것을 어느 파일의 어느
줄로 만드는가**와 **무엇으로 재는가**다.

---

## 0. 앞 단계의 승인과 이 단계의 착수

**사용자 지시 「isolation 코드 써」가 둘을 겸한다** — Functional Design 의 승인과
Code Generation 의 착수다. FD 는 산출물 셋을 낸 채 승인을 기다리고 있었고,
다음 단계로 가라는 지시가 곧 그 승인이다. 같은 회차의 obs 유닛이 같은 자리에서
같은 판단을 했다 (`aidlc-state.md` 의 NFR Requirements 행).

FD 가 승인에 걸어 둔 것 하나가 이 단계에서 풀린다 — `decisions.md` 6절의
사건 종류 실측 행이다 (8절 Step 8).

---

## 1. 이 유닛이 내는 diff 의 모양

```text
   새 파일    internal/enode/mcp.go             형식 · 직렬화 · resolveComponents 뼈대
             internal/enode/mcp_test.go        직렬화와 빈 파일의 모양
             internal/enode/instrument_test.go Instrument 의 다섯 쓰기 · 등급 · 불변식 셋
             internal/enode/logs_test.go       logs/ 선별 (⑱)

   고치는 것   internal/enode/harness.go        인터페이스 둘 · errAux · ParseClaude (⑯)
             internal/enode/claude.go          Fixed(dir) · Argv (⑮) · Instrument
             internal/enode/hook.go            훅 설정을 가짜 홈 안으로 (⑧)
             internal/enode/runner.go          열 걸음 · 등급 · tee (⑲) · 선별 (⑱)
             internal/enode/harness_test.go    ⑯ 의 시험
             internal/enode/claude_test.go     ⑮ 로 확정 빨강인 하나 + Fixed
             internal/enode/hook_test.go       ⑧ 으로 확정 빨강인 넷
```

**시험 파일 셋이 행렬 밖이다.** `unit-of-work-file-matrix.md` 는 제품 파일을
세고 시험은 3절에 「닿는 시험」으로만 적는다. 새 시험 파일 셋은 새 코드의 짝이라
행렬 1절에 행으로 더한다 (그 문서 6절 ②).

---

## 2. 단계 — 여덟

### Step 1 — `internal/enode/mcp.go` 를 만든다

- [x] `MCPServer` · `Components` (`component-methods.md` 2.1 의 시그니처 그대로)
- [x] `Pack` 은 **이름만** 세운다 — 필드와 읽는 코드는 U5 다 (`domain-entities.md` 8절).
      `Components.Pack` 이 그 형식을 가리키므로 이름이 먼저 서야 컴파일된다
- [x] `mcpAllowlistJSON` — 서버 맵을 `{"mcpServers":{...}}` 한 줄로.
      비면 `{"mcpServers":{}}` 다. `Credential` 은 안 나간다 (`domain-entities.md` 1.2)
- [x] `allowlistEntry` — 아는 키 넷(`command` · `args` · `url` · `env`).
      **비면 키를 안 쓴다.** U3 이 미지의 키를 얹을 이음매를 주석으로 적는다 (1.4)
- [x] `writeMCPAllowlist` — `0600` 으로 쓴다
- [x] `resolveComponents(j Job) (Components, error)` — U1 은 언제나 빈 것을 낸다.
      출처 셋과 거절은 U4 · U5 가 이 자리에 자란다

### Step 2 — `internal/enode/harness.go`

- [x] `Fixed(dir string) map[string]string` 로 시그니처를 넓힌다
- [x] `Instrument(dir, self string, a HookArgs, c Components) ([]string, error)`
- [x] `var errAux = errors.New("auxiliary instrumentation failure")`
- [x] `ParseClaude` 가 switch **앞에서** `type` 을 본다 (⑯).
      `Turns` · `CostUSD` · `Session` 을 **먼저** 채운다 —
      `TestHarnessRecordsBudget` 이 그 경계다.
      `Message` 는 봉투의 `type` 만. 비면 `no result envelope type`

### Step 3 — `internal/enode/hook.go`

- [x] `WriteHookSettings` 가 **가짜 홈 안**에 쓴다 (⑧) — `<home>/settings.json`
- [x] `self` 가 비면 `hooks` 키 자체를 안 쓴다. `gatewayAuthFields()` 는 그때도 얹는다
- [x] Marshal 과 WriteFile 의 오류를 `errAux` 로 감싼다. **감싸는 자리는 여기뿐이다** (⑪)
- [x] 돌려주는 플래그는 `--settings <경로>` **하나**다 —
      `--setting-sources ""` 는 파일을 안 가리키므로 어댑터가 언제나 붙인다
      (`business-logic-model.md` 2.1)

### Step 4 — `internal/enode/claude.go`

- [x] `Fixed(dir)` 가 `CLAUDE_CONFIG_DIR=<dir>/home` 을 값으로 박는다.
      `CLAUDE_CODE_DISABLE_AUTO_MEMORY=1` 은 그대로
- [x] `Argv` 에 `--output-format stream-json --verbose` (⑮)
- [x] `Instrument` 가 다섯을 쓴다 — 홈 · 훅 · (팩은 U5) · 허용목록 · 자격증명.
      **조기 반환하지 않는다.** 보조 오류는 들고 가서 마지막에 낸다
- [x] `copyCredentials` — 없으면 정상 · 있는데 못 읽거나 못 쓰면 치명 (R5 · R6)

### Step 5 — `internal/enode/runner.go`

- [x] 계장 디렉터리를 함수 몸통으로. 못 만들면 단계 실패 (④)
- [x] `resolveComponents` 를 exec **앞**에서 부른다
- [x] `Instrument` 를 **언제나** 부르고 등급으로 가른다 — `errAux` 만 삼킨다.
      **삼킬 때도 얻은 플래그는 붙인다**
- [x] `h.Fixed(tmp)` 로 넓어진 시그니처를 받는다
- [x] 하네스 단계의 링 tee 를 끈다 (⑲). `Job.Transcript` 필드는 남긴다
- [x] `selectLogs` — 전문 셋(첫 `system/init` · 마지막 `result` · stderr)과
      껍데기와 표시 줄 (⑱ · 답 1=A · 2=C · 4=A)

### Step 6 — 기존 시험 다섯을 고친다

- [x] `claude_test.go:14-15` `TestAdapter_ArgvIsPure` — ⑮ 로 확정 빨강
- [x] `hook_test.go` 넷 (`:215` · `:247` · `:277` · `:300`) — ⑧ 으로 확정 빨강.
      `--setting-sources` 를 재던 자리는 어댑터 시험으로 옮긴다

### Step 7 — 새 시험을 짓는다

**완료 조건이 「시험이 잰다」로 적은 것만 센다.** 잴 수 없게 적힌 것은 규칙이
아니라 분위기다 (`business-rules.md` 머리).

```text
   R1 · R2 · R4 · R6 이 치명       각각 실패시키고 단계가 실패로 보고되는지
   R3 이 보조                      실패시켜도 단계가 도는지
   errAux 의 자리가 하나            치명 셋이 errors.Is(err, errAux) 로 거짓인지
   불변식 셋을 한 시험에서          R3 을 실패시킨 뒤 home 이 있고 mcp.json 이 있고
                                  플래그 둘이 붙어 있는지 (3.1 · 3.2 · 3.3)
   ⑯ 의 크래시 경로                줄 경계에서 끊긴 stdout 의 마지막 완결 객체가
                                  assistant 인 것.  Message 에 원문이 없는지
   ⑯ 의 예산 신호                  type 없는 봉투에서도 Turns · CostUSD · Session
   ⑲ 의 tee                        Job.Transcript 를 준 채 돌려도 0 바이트
   ⑱ 의 넷                         ① 도구 본문 · ② assistant 의 text ·
                                  ③ 봉투 없이 끊긴 stdout · ④ 껍데기와 전문이 남는지
   6.2 의 둘                        settings 의 hooks 에 Stop 하나뿐 ·
                                  logs/ 의 첫 줄이 init 원문과 바이트로 같다
   직렬화 (답 5 = A)                Components{Servers: ...} 를 직접 넣어 잰다
   빈 파일의 모양                    {"mcpServers":{}} 이고 {} 가 아니다
```

### Step 8 — 게이트와 문서

- [x] CA0 (= 앞 팩 CP0) 을 돌린다 — `go test ./...` · `go vet` · `glyphscan` ·
      `gofmt -l` · 워킹트리 청결 · 커버리지 패키지별 80 · 크로스 빌드와 심볼 상한
- [x] `construction/isolation/code/code-summary.md`
- [x] `decisions.md` 6절에 사건 종류 실측 행 — FD 가 승인에 걸어 둔 하나
- [x] `unit-of-work-file-matrix.md` 에 새 시험 파일 셋을 행으로
- [x] `aidlc-state.md` · `audit.md` (자기 문서 루트의 것만 — `CONVENTIONS.md` 3.4)

---

## 3. 병렬을 안 쓴다

obs 는 파일 축으로 셋을 갈랐다 — 겹치는 파일이 0 이었기 때문이다. **여기는
갈리지 않는다.** 다섯 파일이 한 패키지 안에서 한 시그니처 변경을 함께 받는다
(`Fixed(dir)` 와 `Instrument(..., c)` 가 `harness.go` · `claude.go` ·
`runner.go` 를 동시에 빨갛게 만든다). 갈라서 짜면 세 갈래가 컴파일되는 시점이
하나뿐이라 병렬이 아니라 대기다.

---

## 4. 이 계획이 정하는 것 — FD 에 없던 자리 셋

```text
   WriteHookSettings 가 홈을 받는다   dir 이 아니라 <dir>/home 을 받는다.
                                     가짜 홈의 배치는 claude 의 것이고
                                     (CLAUDE_CONFIG_DIR) hook.go 는 그 이름을
                                     모르는 편이 낫다.  불변식 ① 이 그 디렉터리를
                                     이미 만들어 두므로 받는 쪽이 안 만든다

   --setting-sources 가 어댑터로       hook.go 가 돌려주던 둘 중 하나를 옮긴다.
                                     2.1 이 「파일을 안 가리키므로 언제나 붙는다」로
                                     가른 자리이고, 두 곳에 두면 그 규칙이 갈린다

   선별이 map[string]json.RawMessage  사건 하나를 구조체로 받으면 모르는 필드의
                                     모양 하나가 줄 전체를 떨어뜨린다.  type 을
                                     먼저 집고 아는 키만 따로 읽으면 모르는 것은
                                     안 실리고 아는 것은 남는다 — 허용목록의 규율이다
```

---

## 5. 이 단계가 안 하는 것

```text
   게이트 CA1 의 집행    사람이 재고, 집행자는 이 유닛을 구현하지 않은 사람이다
                        (scene-gates.md 2절 머리)
   scene-gates.md 수정   답 3 = B 라 게이트 명령이 안 바뀐다
   U3 · U4 · U5 의 자리   미지 키 · 출처 합치기 · 팩.  이음매만 이름으로 적는다
   짝 팩의 자리          Decode · 사건 배출 · 링 되살리기
```
