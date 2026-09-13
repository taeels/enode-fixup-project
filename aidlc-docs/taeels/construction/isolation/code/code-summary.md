# U1 `isolation` — Code Generation 요약

**유닛** `isolation` · **브랜치** `unit/isolation` · **담당** taeels ·
**닫는 게이트** CA1 (+ CA0) · **계획** `construction/plans/isolation-code-generation-plan.md`

값은 Functional Design 셋이 닫았고 이 문서는 **그것이 어느 줄이 되었고 무엇으로
재는가**다.

---

## 1. 낸 것

```text
   새 파일    internal/enode/mcp.go              MCPServer · Components · Pack(이름만) ·
                                                resolveComponents · 허용목록 직렬화와 쓰기
             internal/enode/mcp_test.go         직렬화 · 빈 파일의 모양 · 파일 권한
             internal/enode/instrument_test.go  계장의 다섯 쓰기 · 실패 등급 · 불변식 셋 ·
                                                등급의 배선 · 링 tee · 선별의 배선
             internal/enode/logs_test.go        logs/ 허용목록 (⑱)

   고친 것    internal/enode/harness.go     Fixed(dir) · Instrument(..., c) · errAux ·
                                           ParseClaude 의 type 검사 (⑯)
             internal/enode/claude.go      Fixed 가 CLAUDE_CONFIG_DIR 을 박는다 ·
                                           Argv 에 stream-json --verbose (⑮) ·
                                           Instrument 가 가짜 홈 · 허용목록 · 자격증명을 쓴다
             internal/enode/hook.go        훅 설정이 가짜 홈 안으로 (⑧) · self 가 비면
                                           hooks 키를 안 쓴다 · errAux 로 감싼다
             internal/enode/runner.go      열 걸음 · 등급 · 링 tee 를 끈다 (⑲) ·
                                           selectLogs (⑱)
             harness_test.go · claude_test.go · hook_test.go   확정 빨강 다섯과 ⑯ 의 시험 둘
```

**행렬 밖은 시험 파일 셋뿐이다.** 새 코드의 짝이라
`unit-of-work-file-matrix.md` 1절에 행으로 더했다 (그 문서 6절 ②). 제품 파일은
행렬 그대로 다섯이다.

**행렬이 예고한 자리 하나가 안 빨개졌다** — `worker_unix_test.go` 는
`stubHarness` 가 이미 `type:"result"` 봉투를 찍고 있어 ⑯ 과 새 순서를 그대로
통과했다. 고칠 것이 0 이었다.

---

## 2. 잰 것 — CA0

```text
   go test ./...                    전부 초록 (scripts/testdb.sh 를 먼저 쳤다)
   go vet ./...                     깨끗
   go run ./scripts/glyphscan.go    104 파일 · 장식 문자 0
   gofmt -l .                       비었다
   git status --porcelain           추적 파일의 잔여 변경 0
                                    (커버리지 실행이 cmd/enodectl/probe.lock 을
                                     바꾼 것은 되돌렸다 — obs 때와 같은 자리다)
   커버리지 (패키지별 하한 80)        미달 0.  전체 87.0%.
                                    internal/enode 는 85.6% (1687/1971)
   크로스 빌드 + 심볼 상한           GOOS=windows 빌드 통과 · net/http 심볼 6 (상한 50)
```

**라우트 회귀는 이 유닛에서 코드로 잴 것이 없다** — `internal/api` diff 가 0 이고
그 패키지의 시험이 전부 초록이다. 실 Mediator 에 `curl` 로 거는 줄은 게이트
집행자의 몫이다.

---

## 3. 시험이 재는 것 — 완료 조건과의 대응

| 완료 조건 | 재는 시험 |
|---|---|
| R1 치명 (계장 디렉터리) | `TestRunHarness_AnInstrumentationDirectoryThatCannotBeMadeKillsTheStep` |
| R2 · R4 · R6 치명 | `TestAdapter_EverythingButTheHookSettingsIsFatal` 의 하위 셋 |
| R3 보조 · `errAux` 가 한 자리 | `TestAdapter_TheIsolationFlagsSurviveAnAuxiliaryFailure` · 위 셋의 `assertFatal` |
| 등급이 단계의 운명으로 번역된다 | `TestRunHarness_TheGradeDecidesWhetherTheStepRuns` |
| 불변식 ① · ② · ③ | 같은 시험 하나 + 허용목록 실패 경로의 홈 확인 |
| ⑯ 크래시가 성공으로 안 봉인된다 | `TestParseClaude_ACrashIsNotSealedAsSuccess` |
| ⑯ 예산 신호가 남는다 | `TestParseClaude_AnEnvelopeWithNoTypeSaysSo` · 기존 `TestHarnessRecordsBudget` |
| ⑲ 링이 한 바이트도 안 받는다 | `TestRunHarness_TheRingGetsNothingWhileTheStreamIsRaw` |
| ⑱ 의 넷 (도구 본문 · text 본문 · 끊긴 stdout · 껍데기와 전문) | `TestLogs_NoBodySurvivesOnAnyPath` · `TestLogs_TheAllowlistKeepsThreeThingsWhole` |
| ⑱ 이 실제로 걸린다 (배선) | `TestRunHarness_TheUploadedLogIsTheFilteredOne` |
| 첫 줄이 `init` 원문 | `TestLogs_TheFirstLineIsTheInitVerbatim` |
| 우리 설정에 Stop 외의 훅이 0 | `TestAdapter_OnlyTheStopHookIsPlanted` |
| 직렬화 (답 5 = A) · 빈 파일의 모양 | `TestMCP_AServerIsCopiedIntoTheAllowlist` · `TestMCP_TheEmptyAllowlistKeepsTheEnvelope` |
| `self` 가 비면 훅 블록만 빠진다 | `TestAdapter_AnEmptySelfDropsOnlyTheHooksKey` |
| 자격증명 세 갈래 | `TestAdapter_TheCredentialsRideIntoTheFakeHome` · `TestAdapter_NoCredentialsIsNormal` · 위 치명 셋 |

### 3.1 시험이 무는지를 변이로 확인했다

초록은 시험이 있다는 뜻이지 그것이 문다는 뜻이 아니다. 넷을 일부러 되돌려 봤다.

```text
   type 검사를 끈다            TestParseClaude_ACrashIsNotSealedAsSuccess 와
                              ...AnEnvelopeWithNoTypeSaysSo 가 빨개졌다
   링 tee 를 되살린다          ...TheRingGetsNothingWhileTheStreamIsRaw 가 빨개졌다
   selectLogs 를 원문으로      **처음에는 아무것도 안 빨개졌다** — logs_test.go 는
                              함수를 직접 부르고 배선을 안 쟀다.
                              ...TheUploadedLogIsTheFilteredOne 을 더했고 그 뒤
                              같은 변이가 빨개진다
```

**그 하나가 이 단계에서 실제로 찾은 구멍이다.** 함수가 있어도 `runHarness` 가 안
부르면 원문이 그대로 Record 로 올라가 봉인된다.

---

## 4. 계획에 없던 결정 셋

```text
   WriteHookSettings 가 홈을 받는다   dir 이 아니라 <dir>/home 을 받는다.  가짜 홈의
                                     배치는 claude 의 것이고(CLAUDE_CONFIG_DIR)
                                     hook.go 는 그 이름을 모르는 편이 낫다.
                                     디렉터리를 안 만든다 — 불변식 ① 이 이미 만들었다

   --setting-sources 가 어댑터로       hook.go 가 돌려주던 둘 중 하나를 옮겼다.
                                     그것은 파일을 안 가리키므로 우리 파일의 성패와
                                     무관하고, 두 곳에 두면 훅 쓰기 실패가 개인 설정
                                     차단까지 함께 떨어뜨린다 (불변식 ②)

   선별이 map[string]json.RawMessage  사건을 구조체로 한 번에 받으면 모르는 필드의
                                     모양 하나가 줄 전체를 떨어뜨린다.  type 을 먼저
                                     집고 아는 키만 따로 읽는다 — 허용목록의 규율이다
```

---

## 5. 실측으로 굳힌 것 둘

```text
   --mcp-config 는 가변인자다      claude 2.1.266 의 --help 가 <configs...> 로 적는다.
                                 등호 형태를 고른 근거(ADR-034 §3 ②)가 실물로 섰다
   플래그 넷이 다 있다             --strict-mcp-config · --mcp-config ·
                                 --setting-sources · --output-format stream-json
```

사건 종류 실측은 `requirements/harness-components/decisions.md` 6절 ⑳ 에 행으로
실었다 — FD 가 승인에 걸어 둔 하나였다.

---

## 6. 한계와 넘기는 것

```text
   CA1 을 이 유닛이 안 닫는다     집행자는 이 유닛을 구현하지 않은 사람이다
                                (scene-gates.md 2절 머리).  개인 MCP 서버와 계정
                                커넥터가 있는 기계에서, 그리고 OAuth 노드와 게이트웨이
                                노드 **둘 다**에서 재야 한다

   눈 검증이 복제본을 잰다        계장 보존 스위치를 안 만들었다 (⑩).  사람이 손으로
                                띄우는 경로는 환경 · 플래그 · 게이트웨이 인증에서
                                실물과 갈린다.  ⑮ 가 그 대가를 줄였다 — init 줄이
                                logs/ 에 남으므로 CA1 은 실물을 읽는다

   6.3 의 잔여 위험              「첫 줄은 init」은 우리 설정 파일의 내용에 기댄
                                전제다.  하네스가 init 앞에 사건을 내면 게이트의
                                head -1 이 엉뚱한 줄을 읽는다.  유닛 안에서 잴 수
                                있는 근거 둘(훅이 Stop 하나 · 선별이 순서를 지킨다)은
                                시험이 잰다

   8.1 의 등급                   R3 이 보조인데 그 파일이 게이트웨이 인증도 든다.
                                디스크 실패로 못 쓰면 게이트웨이 노드가 Not logged in
                                으로 늦게 죽는다.  값을 바꾸려면 hook.go 의 겉면이
                                먼저 갈라져야 한다 — 진행자의 것이다

   6.1 의 문자열 검사 스크립트     decisions.md 6.1 이 「자리는 scripts/ 다.  짓는 것은
                                Construction 의 몫」으로 자리만 적었고 **어느 유닛에도
                                배정되지 않았다.**  U1 의 완료 조건에도 파일 행렬에도
                                없어서 이 유닛이 안 지었다.  오늘은 진행자가 손으로
                                돈다 — 배정할 자리가 진행자의 것이다
```

---

## 7. 다음 유닛이 딛는 자리

```text
   U3   allowlistEntry 의 이음매 — 미지의 키 맵을 먼저 얹고 아는 키로 덮는다.
        Credential 을 읽는 자리는 mcpUp 하나다
   U4   resolveComponents 의 몸통.  호출 자리와 순서는 U1 이 세워 뒀다
   U5   Instrument 의 ③ 자리(팩을 홈 아래 편다) · Pack 형식 · HarnessResult 의 두 필드
   짝 팩  runner.go 의 링 tee 자리.  Job.Transcript 는 남겨 뒀다
```
