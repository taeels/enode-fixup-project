# U5 `pack` — Code Generation 요약

**유닛** `pack` · **브랜치** `unit/pack` · **담당** taeels ·
**닫는 게이트** CA5 · CA6 (사람) · CA0 · **선행** U1 · U2 · U4

계획은 `../../plans/pack-code-generation-plan.md`, 설계는
`../functional-design/` 의 셋이다. 규칙 번호 R1 ~ R35 는 그 `business-rules.md`.

---

## 1. 낸 것

```text
   제품 파일 여섯   mcp.go · runner.go · harness.go · claude.go ·
                   iapadapter 의 config.go · contract.go.  새 제품 파일 0
   시험 파일        새 파일 하나 + 고친 파일 다섯
   패키지 둘        internal/enode · cmd/iapadapter.
                   internal/contract · internal/api 는 소스 diff 0
```

| 파일 | 무엇 | 규칙 |
|---|---|---|
| `internal/enode/mcp.go` | `Pack` · `PackFile` · `PackLimits` · `packInput` · `readPack` 과 헬퍼 여섯(`countingReader` · `overLimit` · `packEntryKind` · `packEntryName` · `packSafeName` · `packMCPServers`) · `resolveComponents` 의 걸음 여덟 | R3 ~ R22 |
| `internal/enode/runner.go` | `openPack` · ①.5 호출 · ⑧ 의 기록 두 필드 | R1 · R2 · R28 ~ R30 |
| `internal/enode/harness.go` | `HarnessResult.MCP` · `.Pack` (둘 다 `omitempty`) | R28 · R29 |
| `internal/enode/claude.go` | `packDirName` · `writePack` · `Instrument` 의 ③ | R23 ~ R27 |
| `cmd/iapadapter/config.go` | `ExecutorConfig.Pack` · `PackConfig` · `LoadConfig` 의 거절 하나 | R31 |
| `cmd/iapadapter/contract.go` | `BuildContract` 의 팩 단계 · `planPrompt` 의 안내 | R32 ~ R35 |

시험은 `pack_test.go`(새)가 `readPack` 의 표이고, `resolve_test.go` 에 팩 출처의
여덟을, `instrument_test.go` 에 펴기 셋과 기록·배선 넷을, `iapadapter` 의
`config_test.go` · `adapter_test.go` 에 넷을 더했다. `mcp_test.go` 와
`resolve_test.go` 의 호출 열하나는 **인자가 늘어 확정 빨강이던 옛 시험**이다.

**`claim.go` 는 소스 diff 가 0 이다.** 팩을 여는 것은 `runHarness` 이고
(답 3=A) 그래서 파일 행렬의 `claim.go` 행에 U5 가 안 붙었다 — 계획 5절의
예상 그대로다.

---

## 2. CA0 — 전부 초록

```text
   go test ./... -count=1                 exit 0 · 실패 0
   커버리지 (표준 명령 · 패키지별 80%)      미달 0 · 전체 7117/8141 = 87.4%
                                          internal/enode 2013/2314 = 87.0%
                                          (U4 뒤 86.4% 에서 올랐다)
   스킵                                   0
   go vet ./...                           exit 0
   gofmt -l .                             0 줄
   장식 문자 (U+2605 전수 grep)             0 히트
   go run ./scripts/glyphscan.go           104 파일 · 문자열의 장식 문자 0
   크로스 빌드                             windows/amd64 · linux/arm · darwin/arm64
   심볼 상한                               enodectl crypto/tls=1 · net/http=6 (상한 10 · 50)
   Mediator 라우트                          26 (안 늘었다)
   node --test (ui)                       75 통과 · 실패 0
   워킹트리 청결                            커버리지 실행이 바꾼
                                          cmd/enodectl/probe.lock 을 되돌렸다
```

---

## 3. 실측 — 우리 코드가 지은 디렉터리를 하네스가 읽나

계획 Step 9 다. **복제본이 아니라 제품 코드가 낸 디렉터리로 쟀다** — tar 를
`readPack` 에 넣고 그 결과를 `Instrument` 가 `<계장>/pack` 에 편 뒤, 그 디렉터리를
`claude 2.1.270` 에 오늘의 플래그 그대로 물렸다.

```text
   플래그      --setting-sources "" --settings <홈>/settings.json
              --plugin-dir=<계장>/pack --strict-mcp-config --mcp-config=<계장>/mcp.json

   skills          [... 내장 열여덟 ...] 안에 pack:hello
   agents          [claude Explore general-purpose pack:helper Plan statusline-setup]
   slash_commands  pack:hello 가 index 1
   mcp_servers     [{"name":"probe4","status":"failed"}]
   plugins         [{"name":"pack","path":"<계장>/pack","source":"pack@inline"}]
```

**등호 형태가 선다.** 이것이 이 단계가 새로 잰 것이다 — FD 의 실측은 공백
형태였고, `Instrument` 의 ③ 자리 바로 뒤에 `--strict-mcp-config` 가 따라붙으므로
가변인자면 그것을 삼킨다 (`ADR-034` §3 ② 의 함정). 삼키지 않았다 —
`mcp_servers` 에 `probe4` 가 그대로 있다.

**샌 것이 0 이다.** 이 기계의 `~/.claude/skills` 수십과 `~/.claude/agents` 둘
(`aidlc-verify` · `aidlc-xhigh`)이 하나도 안 나타났다. `--setting-sources ""` 를
안 건드린 것이 값으로 보인 자리다.

**`.claude-plugin/plugin.json` 없이 실렸다** — 2.1.270 의 값이다. 규약이 굳으면
`writePack` 에 파일 하나가 는다.

---

## 4. 변이 일곱 — 다 빨개졌다. 구멍 0

넣고 `go test ./internal/enode/ -run 'TestPack_|TestResolve_|TestRunHarness_'` 를 돌렸다.

```text
   ① 걸음 4 의 드레인을 걷는다        RED  HashCoversTheWholeFile
   ② R4 의 종류 검사를 걷는다         RED  RejectsEntryKinds (하위 다섯 전부)
   ③ 푼 바이트 상한을 걷는다          RED  RejectsOverTheSizeLimit/expanded_bytes
   ④ R16 의 거절을 걷는다            RED  RejectsAnEmptyPack (하위 넷 전부)
   ⑤ 걸음 0 을 걸음 2 뒤로            RED  ABrokenPackStopsTheStepEvenWithNoRequest ·
                                         AMissingPackBlobNeverStartsTheHarness(배선)
   ⑥ 걸음 1 을 걸음 2 뒤로            RED  AnEmptyRequestStillCarriesThePack ·
                                         APackMayNotRedefine · TheRecordSays(배선)
   ⑦ 팩 서버의 요청 필터를 걷는다      RED  ThePacksServersAreFilteredToo ·
                                         TheRecordSaysWhatWasActuallyLoaded(배선)
```

**⑤ ⑥ ⑦ 은 순수 시험과 배선 시험이 함께 빨개졌다.** U1 에서 「함수만 부르고
배선을 안 쟀다」로 났던 구멍이 여기서도 안 생긴다.

**③ 이 가장 얇았다** — 받은 바이트 상한이 대부분의 큰 팩을 먼저 잡으므로,
푼 바이트 갈래만 재려면 압축률이 높은 폭탄이 필요하다. 시험이 그것을 명시로
짓고(`1 MiB` 의 0 을 gzip 해 `64 KiB` 상한 아래로 들어가는지 먼저 단언한다)
그래서 이 변이가 빨개진다.

---

## 5. 이 단계가 계획에 없던 것을 둘 더했다

```text
   ①  팩 항목의 모양 검사 (mcp.go)
      command 도 url 도 없는 팩 항목이 요청되면 거절한다 —
      pack mcp server %s declares neither command nor url.
      U4 가 워크스페이스 출처에서 같은 실패를 이미 거절했고(6절 ㉓ ②),
      「종류를 못 정하는 항목을 하네스가 말없이 뺀다」는 실측이 출처를 안 가린다.
      안 더하면 이 유닛이 그 침묵을 팩 경로로 되살린다 —
      ADR-035 §3 의 「없음이 실패보다 나쁘다」가 막으려던 바로 그 자리다

   ②  Notes 를 걸음 0 의 오류보다 앞에 옮겼다 (mcp.go)
      FD 는 걸음 0 을 오류 검사로 적었는데, 거기서 곧바로 반환하면 규약 밖
      항목의 이름이 거절과 함께 사라진다.  U4 의 R11(거절로 끝날 때도 그때까지의
      Components 를 돌려준다)과 같은 규율이라 Notes 복사를 오류보다 앞에 뒀다.
      걸음 0 이 요청 조기 반환보다 앞이라는 FD 의 뜻은 그대로다
```

**글자 하나를 계획에서 바꿨다** — 모르는 tar 종류의 이름을 계획은
`unknown entry type` 으로 적었고 코드는 `tar type '<글자>'` 로 갔다. 문구가
`pack entry %s is a %s` 라 앞의 `is a` 뒤에 놓으면 영어가 안 되고, 이름을
지어내면 그 이름으로 찾을 문서가 없다. 계획 4절 ② 를 그 값으로 고쳤다.

---

## 6. 실측이 계획의 문장 하나를 고쳤다

```text
   0 바이트 blob 은 「tar 가 아니다」가 아니라 「실을 것이 0」이다
   계획 Step 10 은 빈 입력을 R3 의 표에 넣었는데, Go 의 tar 는 빈 스트림을
   빈 아카이브로 읽어 R16 으로 떨어진다.  둘 다 치명이고, 이 문구가
   curl 이 빈 파일을 남긴 경우를 더 곧게 가리킨다 — 갈래를 안 더하고
   시험을 R16 의 표로 옮겼다
```

---

## 7. 이 유닛이 회차 밖으로 낸 것

```text
   decisions.md 6절   실측 행 ㉔ — 오늘 플래그로는 팩이 안 읽힌다 · 길 둘의
                      실측값 · 워크스페이스 스킬이 안 읽힌다 · 없는 플러그인
                      디렉터리의 침묵 · 훅이 두 번 안 뛴다.  등호 형태까지 한 행에
   decisions.md 2절   워크스페이스 .claude/skills 행이 미정에서 값으로 ·
                      팩의 형식 행에 gzip 과 상한
   features.md 5절    미정이 둘에서 하나로
   features.md 3.6    펴는 자리가 가짜 홈에서 <계장>/pack 으로
   unit-of-work.md    U5 의 claude.go 줄 · 완료 조건의 「판정」이 「확인」으로
   scene-gates.md     CA5 의 표 한 줄과 3절 명령 넷 — pack:hello · 필드 둘 ·
                      충돌 줄의 agent.mcp · 마지막 줄이 확인이 된 것
   component-methods  2.1 의 Components.Servers(U4 가 남긴 낡은 줄) ·
                      Pack.MCP · PackFile.Mode · PackLimits 의 값
   파일 행렬           1절의 새 시험 파일 행과 resolve_test.go 의 U5 칸 ·
                      2.1 · 2.2 · 2.3 의 U5 칸
   짝 팩과의 접점       runner.go · claude.go 둘.  스트림 처리 자리(⑱ · ⑲)와
                      Argv · Decode 는 안 건드렸다
```

---

## 8. 진행자에게 넘기는 것 — 다섯

```text
   1  스킬 이름이 pack: 으로 namespace 된다 (답 1=A 의 대가)
      계약 저자가 슬래시로 부르는 이름과 CA5 가 읽는 글자가 함께 바뀐다.
      모델이 설명으로 스킬을 고르는 경로는 안 바뀐다.  ADR-034 §1.1 의 팩
      규약과 --plugin-dir 이 갈리는 자리라 정본 개정 후보다

   2  플러그인 규약이 굳으면 매니페스트 한 장이 는다
      .claude-plugin/plugin.json 없이 실리는 것은 2.1.270 의 실측값이다.
      그날 writePack 에 파일 하나가 늘고 CA5 가 그것을 먼저 만난다

   3  계획이 두 줄을 잊으면 팩이 조용히 안 실린다 (R34)
      planPrompt 가 blob 이름과 agent.pack · in.from 을 이름으로 알려주지만,
      계획이 안 적으면 그 단계는 팩 없이 돌고 우리가 못 잡는다.
      decisions.md 6절 ㉑ 이 이미 이름으로 진 침묵이다

   4  팩의 기대 다이제스트가 없다 (이월)
      무엇을 받았는지는 harness.pack 이 남기지만 무엇을 받아야 했는지는
      아무도 안 적는다.  주소가 바뀐 tar 를 우리가 못 잡는다

   5  팩 내용 자체의 악의는 경계 밖이다 (ADR-042)
      스킬 본문이 시키는 것을 안 본다.  경계는 단계가 아니라 노드에 있고,
      enode 를 띄운 것이 곧 임의의 argv 를 받는다는 선언이다
```

---

## 9. CA5 와 CA6 은 아직 안 닫혔다

Step 9 가 잰 것은 **우리 코드가 지은 디렉터리를 하네스가 읽나**이고, CA5 는
**노드와 Mediator 를 지나 그 팩이 실제로 실리나**다. CA6 은 사내 함대에서
`runctl record` 로 푼 `steps/NN-*.json` 의 두 필드를 읽는다.

**집행자는 이 유닛을 구현하지 않은 사람이다** (`scene-gates.md` 2절 머리).
**눈 검증을 보류로 안 넘긴다** (같은 문서 4절).
