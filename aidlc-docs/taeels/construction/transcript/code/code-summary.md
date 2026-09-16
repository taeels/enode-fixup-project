# U1 `transcript` — Code Generation 요약

**유닛** `transcript` (U1) · **브랜치** `unit/transcript` · **담당** taeels ·
**정본** `plans/transcript-code-generation-plan.md` · **잰 날** 2026-09-16

이 문서는 **잰 것만** 적는다. 안 돈 것과 못 잰 것은 8절이 이름으로 든다.

---

## 1. 무엇이 생겼나

```text
   새 패키지 하나   internal/transcript
                  제품 파일 넷 790 줄 · 시험 파일 다섯 935 줄 · 픽스처 열여덟
   고친 파일 셋     internal/enode/runner.go · internal/enode/logs_test.go ·
                  internal/panel/boundary_test.go
   새 시험 파일 하나 internal/enode/logs_events_test.go  104 줄
   go.mod · go.sum  diff 0.  새 의존 0
```

| 파일 | 줄 | 무엇 |
|---|---|---|
| `internal/transcript/transcript.go` | 151 | `Kind` 일곱 · `Event` · `Info` · `Server` · `Result` · `Elided` · `Fields` |
| `internal/transcript/line.go` | 68 | `SplitLines` · `ParseLine` · `String` · `Bool` · `Int` |
| `internal/transcript/shell.go` | 145 | `Shell` · `logShell` · `ElidedMarker` · `elidedMark` · `usageTokens` · 글자 상수 셋 |
| `internal/transcript/parse.go` | 426 | `Parse` 의 파이프라인 여덟 · 판정 사다리 넷 · 룬 경계 자르기 |

**고친 쪽의 diff** — 81 줄이 들어오고 259 줄이 나갔다.

```text
   internal/enode/runner.go         183 줄 변동.  덜어낸 것이 168 줄이다
   internal/enode/logs_test.go      108 줄 변동.  상수 다섯이 파일 읽기가 됐다
   internal/panel/boundary_test.go   49 줄 변동.  금지 표와 봉인이 들어왔다
```

---

## 2. 옮긴 열 — 함수 여덟과 타입 둘

`runner.go:369` ~ `:536` 한 덩어리를 지웠다. 사이에 남은 것이 0 이다.

| 옛 이름 | 새 이름 | 공개 | 파일 |
|---|---|---|---|
| `splitLines` | `SplitLines` | 공개 | `line.go` |
| `parseEventLine` | `ParseLine` | 공개 | `line.go` |
| `eventString` | `String` | 공개 | `line.go` |
| `eventBool` | `Bool` | 공개 | `line.go` |
| `eventInt` | `Int` | 공개 | `line.go` |
| `elidedMarker` | `ElidedMarker` | 공개 | `shell.go` |
| `eventShell` | `Shell` | 공개 | `shell.go` |
| `usageTokens` | `usageTokens` | 비공개 | `shell.go` |
| `logShell` (타입) | `logShell` | 비공개 | `shell.go` |
| `elidedMark` (타입) | `elidedMark` | 비공개 | `shell.go` |

**공개 표면이 열다섯이다** (실측 — `go doc -all` 의 `^func [A-Z]|^type [A-Z]` 가 15).
`selectLogs` 는 안 옮겼다 — 파서가 아니라 정책이고 `internal/enode` 에 남는다.

**부르는 자리 여섯을 다시 걸었다.** 서로 다른 함수는 다섯이다 (`ParseLine` 이
두 번 불린다). 로직은 한 줄도 안 바뀌었다. `encoding/json` 임포트는 `runner.go`
에서 쓰임이 0 이 되어 뺐다.

**R11.2 의 「한 줄도 안 고친 채」가 그대로는 거짓이다.** `logs_test.go` 의 시험
함수 아홉 중 하나가 `elidedMark` 타입을 쓴다. 고친 것이 정확히 한 줄
(`var mark elidedMark` -> `var mark transcript.Elided`)이고 나머지 여덟의 몸통은
손대지 않았다.

---

## 3. Step 2 의 실측 — 실물 하네스로 쟀다

**돌렸다.** `claude 2.1.271` · 2026-09-16 · `-p --output-format stream-json --verbose`
로 도구를 부르는 턴 둘. **원문은 저장소에 안 싣는다** — 모양만 손으로 지어
`testdata/lines/` 에 넣었다.

### 3.1 물음 밖 ① — `tool_use_id` 는 있다

```text
   물었던 것   user 의 tool_result 블록에 tool_use_id 키가 있는가
   실측        있다.  본 tool_result 블록 다섯 전부에 있었다 —
               content 가 문자열인 것도 배열인 것도 같다
```

**`business-logic-model.md` 5.2 가 이름으로 진 빈자리가 닫혔다.** 그 절은
「이 단계가 실측으로 못 댄다」로 적고 Code Generation 에 넘겼다. 실측이 왔으므로
**붙이기 경로(R12)가 실제 트랜스크립트에서 돈다** — 안 도는 죽은 가지가 아니다.
규칙은 그대로 둔다. R12.2 가 키의 존재에 안 기대는 것이 여전히 옳다.

### 3.2 물음 1 — `content` 는 **둘 다** 온다

```text
   문자열 꼴   도구 결과가 글자일 때.  본 다섯 중 넷
   배열 꼴     **온다.**  이미지를 읽는 턴에서 content 가 배열이었다
```

**앞 문서의 전제가 틀렸다.** 계획 11절은 「배열 꼴을 이 저장소가 한 번도 기록하지
않았다」로 적고 값을 못 정했다. 실측이 배열을 냈다.

**본 배열의 안이 값을 정했다.**

```text
   본 것    [{"type":"image","source":{...}}]  — 원소 하나이고 image 다.
            text 원소가 **0** 이다
   뜻       그 배열에서 건질 본문이 없다.  A 와 B 가 이 모양에서 결과가 같다
```

**A 로 지었다** — 문자열이면 그대로 쓰고 그 밖의 모양이면 `Text` 를 비운다.
짐작이 아니라 실측이 받친다: 본 배열에 본문이 0 이므로 B(텍스트 블록 잇기)가
같은 입력에서 같은 빈 문자열을 낸다. **A 가 아는 모양을 하나 덜 들고 같은 값을
낸다.**

```text
   남는 대가   text 원소를 든 배열이 오면 A 는 그것을 비운다.
              그 모양을 **아직 못 봤다** — 봤다고 적지 않는다
   규칙 번호   비워 둔다.  business-rules.md 는 앞 단계의 문서이고
              이 유닛이 안 고친다.  9절 ⑨ 가 그 자리를 진행자에게 넘긴다
```

시험이 그 판정을 묶는다 —
`TestParse_AToolResultWhoseContentIsAnArrayCarriesNoText` 가
`Text == ""` 이고 `ID` 는 그대로 서는 것을 잰다 (줄을 안 버린다).

### 3.3 덤으로 확인된 셋

```text
   claude_code_version   있다.  "2.1.271".  initEvent 가 읽는 키가 맞다
   mcp_servers           배열이다.  [{"name":...,"status":...}] — 맵이 아니다
   usage                 정수 넷 말고 문자열 둘(service_tier · inference_geo)과
                        객체 하나(cache_creation)가 섞여 온다.
                        usageTokens 가 이름으로 집는 근거가 실측으로 다시 섰다
```

---

## 4. 이음매 — `enode.capped` 의 **반쪽만** 닫았다

### 4.1 U1 이 닫은 반쪽

```text
   글자가 한 자리다   shell.go 의 cappedType = enodePrefix + "capped"
   시험이 그것을 잡는다  TestParse_CappedCarriesTheLimitItWasGiven 이
                     testdata/lines/capped.json 을 읽어 잰다
   잰 값             Kind == KindCapped · Info.Bytes == 10485760 ·
                     Line == 2 (위치가 값이다.  집계로 안 뺀다)
```

**U3 이 읽을 자리가 여기다** — 맞댈 것 둘을 그대로 적는다.

```text
   와이어       {"type":"enode.capped","bytes":<상한>}
   type 의 글자  "enode.capped".  점을 넣는다
   bytes 의 뜻   **닿은 상한**이다.  총 길이가 아니다 (R3.9)
   기본값       10485760 (10 MiB · MaxBlobBytes)
```

### 4.2 U1 이 **못 닫은** 반쪽

**이 브랜치에 `enode.capped` 를 찍는 코드가 없다.** `internal/record` 는 U3 의
것이고 착수 순서가 U1 을 먼저 세운다.

```text
   실측   grep -rn 'enode\.capped' internal/  ->  internal/transcript 안에서만 나온다
   그래서  왕복 시험이 이 줄을 못 잡는다.  짓는 쪽이 이 패키지에 없기 때문이다
```

**U1 은 이 절반을 초록이라고 안 적는다.** 잰 것이 없는 자리다. U3 이 병합된 뒤
진행자가 아래 한 줄을 돌려 두 자리의 글자와 뜻을 맞댄다.

```text
   grep -rn 'enode\.capped' internal/record internal/transcript
```

---

## 5. 게이트 — 실측값

**환경 둘이 있어야 이 값이 나온다.** 하나는 문서에 있고 하나는 없었다 (9절 ⑩).

```text
   ENODE_TEST_DATABASE_URL   .coverage-contract.yml 의 command.env 가 적었다
   .github/ci-stubs 를 PATH 에   **그 파일이 안 적었다.**  없으면 아래가 다르게 나온다
```

| | 잰 것 | 값 | 판정 |
|---|---|---|---|
| 1 | `go build ./...` | rc 0 | 통과 |
| 2 | `go vet ./...` | rc 0 | 통과 |
| 3 | `go test ./... -count=1` | 19 패키지 전부 ok | 통과 |
| 4 | `go run ./scripts/glyphscan.go` | 108 파일, 장식 문자 0 | 통과 |
| 5 | `grep -rlIP '\x{2605}'` | 0 파일 | 통과 |
| 6 | `gofmt -l .` | 빔 | 통과 |
| 7 | 커버리지 (표준 명령 · 하한 80%) | **전체 7258/8296 = 87.5% · 미달 0개** | 통과 |
| 8 | 스킵 감시 | 19 패키지, 허용목록 밖 스킵 0 | 통과 |
| 9 | `GOOS=windows` · `linux/arm` · `darwin/arm64` 빌드 | 셋 다 rc 0 | 통과 |
| 10 | `enodectl` 심볼 상한 | `crypto/tls` **1** (상한 10) · `net/http` **6** (상한 50) | 통과 |
| 11 | 라우트 수 | `grep -c 'mux.HandleFunc' internal/api/api.go` = **17** | 통과 |
| 12 | 금지 경로 diff | `internal/store` · `internal/contract` · `internal/api` · `cmd/` 넷 다 **0** | 통과 |
| 13 | 경계 검사 | `TestImportBoundaries` 통과. 금지 여덟 + 봉인 하나 | 통과 |

**7 의 87.5% 가 `.coverage-contract.yml` 의 `measured_total_pct: 87.5` 와 같다.**

### 5.1 커버리지 — 예산과 실측

```text
   internal/enode    1951 / 2250 = 86.7%
                     계획의 예측은 1909 / 2250 = 84.8% 였다.
                     **분모 2250 이 예측과 정확히 같고** 분자가 42 높다 —
                     logs_events_test.go 가 selectLogs 를 더 밟았다

   internal/transcript  202 / 214 = 94.4%
                     안 덮인 것이 12 문장이다.
                     예산은 0.2 x 214 = 42.8 문장까지이므로 여유가 30.8 이다
```

**안 덮인 둘의 자리는 그대로다** — `ElidedMarker` 와 `Shell` 의 `json.Marshal`
오류 가지다. **안 고치고 안 덮는다.** 그 두 함수가 마샬하는 것이 `string` ·
`int` · `bool` · `[]string` · `map[string]int` · `*bool` 뿐이고 `encoding/json`
이 그 종류에서 오류를 안 낸다 — **구조상 못 밟는 가지**다. 밟게 하려면 시험이
타입을 바꿔야 하고 그것이 R13 위반이다.

### 5.2 경계 검사 — 봉인이 실제로 잰다

```text
   go list -deps ./internal/transcript | awk -F/ '$1 ~ /\./'
     -> github.com/taeels/enode/internal/transcript  (자기 하나뿐이다)
```

**봉인이 공허하지 않다는 것을 옆 패키지로 확인했다** — 같은 한 줄이
`internal/config` 에서 `yaml.v3` 를, `internal/store` 에서 pgx 무리를 낸다.
`internal/enode` 는 이제 `internal/transcript` 를 낸다 (방향이 하나인 그 모서리).

---

## 6. 시험이 실제로 재는가 — 변이 다섯

**넣고 빨개지는지 봤다. 둘이 안 빨개졌고 그 둘이 이 단계의 수확이다.**

| | 변이 | 처음 | 고친 뒤 |
|---|---|---|---|
| ① | 사다리에서 `enode.` 갈래를 걷는다 | **빨강** (4 시험) | — |
| ② | 껍데기 판정을 `assistant` · `user` 밖으로 넓힌다 | **빨강** (2 시험) | — |
| ③ | `Tokens` 를 사건마다 복사한다 | **빨강** (1 시험) | — |
| ④ | 룬 경계 물리기를 걷는다 | R8 ③ 만 빨강. **F2 는 초록이었다** | F2 도 빨강 |
| ⑤ | 블록 0 인 줄을 사건 0 으로 넘긴다 | **전부 초록.** 구멍이다 | R1 · R5 · F3 빨강 |

### 6.1 ④ 가 드러낸 것 — F2 가 공허했다

계획 Step 16 ④ 는 「R8 ③ 과 F2 가 빨강」으로 적었다. **F2 는 안 빨개졌다.**

```text
   왜    F2 의 전제가 Cut > 0 인데 **시드 열여섯 중 그것을 내는 것이 0 이었다** (실측).
         픽스처가 전부 짧아 상한에 안 닿는다
   더    -fuzz 로 45 초를 돌려도 안 잡혔다.  500 바이트가 넘는 온전한 JSON 줄에
         멀티바이트 룬이 정확히 경계에 놓이는 입력은 무작위 변이가 닿기에 깊다
```

**픽스처 하나를 더해 고쳤다** — `testdata/lines/long-tool-result.json` 은 498
바이트 뒤에 한글을 두어 500 번째 바이트가 룬 가운데다. 시드가 픽스처라
(`f.Add` 가 디렉터리를 읽는다) **이제 `-fuzz` 없이 `go test` 만으로 F2 가
④ 를 잡는다.** 확인했다.

**계획의 파일 목록에 없던 픽스처다.** 더한 이유가 이것이고 9절 ⑧ 이 그 갈림을
진행자에게 넘긴다.

### 6.2 ⑤ 가 드러낸 것 — 줄기 둘 중 하나만 재고 있었다

**변이 ⑤ 를 넣어도 시험이 전부 초록이었다.** 되돌림이 두 자리인데 시험이
한 자리만 밟았다.

```text
   껍데기 줄기   shellEvents 의 되돌림.  {"type":"assistant"} 가 그리로 간다 —
                message 키가 아예 없기 때문이다.  **R5 시험이 이쪽만 쟀다**
   원문 줄기     messageEvents 의 되돌림.  message 는 있고 블록이 0 인 줄.
                **아무 시험도 안 밟았다** — 지워도 초록이었다
```

**고친 것 둘** — R5 시험을 줄기 넷(껍데기 하나 · 원문 셋)의 표로 넓혔고,
블록 0 인 줄(`assistant-no-blocks.json`)을 R1 시험의 입력에 넣었다. 이제
변이 ⑤ 가 R1 · R5 · F3 셋을 빨갛게 만든다.

---

## 7. 증폭 — 잰 값과 **계획이 틀린 자리**

`nfr-requirements.md` 7절이 「벤치마크가 아직 없다」로 넘긴 자리다.

```text
   unsafe.Sizeof(Event{})   **224 바이트** (amd64 · go1.26.6).
                            nfr-requirements 5.1 의 계산값과 같다.
                            시험 하나가 못 박는다 — 필드를 더하면 빨개진다
```

### 7.1 증폭 — 재는 값이 둘이고 계획이 하나를 잘못 가리켰다

계획 Step 17 은 「파싱 비용이 입력의 약 2배인지 (`-benchmem` 의 `B/op`)」로
적었다. **`B/op` 로 재면 약 2배가 아니다.**

| | 무엇 | 값 |
|---|---|---|
| 붙든 것 | 사건 배열 (`len(Events) x 224`) | **입력의 1.67 배** |
| 흘린 것 | `B/op` — 파싱 한 번의 총 할당 | **입력의 24.8 배** |

```text
   실측 (1000 줄 · 입력 161,307 바이트)
     retainedEventArray   269,024 바이트   1.67x
     B/op               3,994,509 바이트  24.76x   58,062 allocs/op
```

**둘 다 참이고 다른 것을 잰다.** 「약 2배」는 **붙든 것**에 대해 맞다 —
화면이 들고 있는 양이 그것이다. `B/op` 는 지나가며 버리는 양이라 GC 부담이고,
그 수가 25 배다. **계획이 「약 2배」를 `B/op` 로 재라고 적은 것이 갈림이다**
(9절 ⑧).

### 7.2 최악의 경우 — **계획의 112 MB 가 21 배 적다**

계획은 「최악의 사건 배열 약 112 MB」를 곱셈으로 적었다. **그 곱셈은 가장 짧은
JSON 줄(21 바이트)을 가정한다.** 더 짧은 줄이 있다.

```text
   실측 (사건 하나당 224 바이트 · 10 MiB 입력으로 환산)

     {"type":"assistant"}\n   21 바이트/줄   10.7x   ->   **112 MB**   계획의 값
     a\n                       2 바이트/줄  112.0x   ->  **1,174 MB**
     \n                        1 바이트/줄  224.0x   ->  **2,349 MB**
```

**개행만 든 10 MiB 파일이 사건 약 1,048 만 개를 내고 배열만 2.3 GB 다.**
빈 줄도 사건 하나이기 때문이다 — JSON 이 아니므로 `plain` 으로 떨어진다.
그것은 「줄을 안 버린다」(R1)의 값이 치르는 대가이고, 규칙을 바꿀 자리가 아니다.

**U1 이 여기에 상한을 안 건다** — 계획 10절이 짓지 않는 것에 그것을 넣었고,
`as=events` 의 응답을 지는 것은 U4 다. **수를 U4 에 넘긴다** (9절 ⑦).

### 7.3 벤치마크 실측

```text
   BenchmarkParse/10 lines      92,042 ns/op    18.68 MB/s     42,145 B/op     630 allocs/op
   BenchmarkParse/100 lines    783,565 ns/op    20.71 MB/s    388,217 B/op   5,853 allocs/op
   BenchmarkParse/1000 lines 8,722,434 ns/op    18.49 MB/s  3,994,509 B/op  58,062 allocs/op
   BenchmarkShell               10,373 ns/op                    3,316 B/op      69 allocs/op

   기계   Intel N100 · linux/amd64 · go1.26.6 · -benchtime=200x
```

**10 MiB 입력은 벤치마크로 안 돌렸다** — 곱셈으로 적었다 (7.2). CI 가 아니라
사람의 기계가 멈추는 크기다.

---

## 8. 퍼즈가 닫는 것과 **안 닫는 것**

```text
   go test ./internal/transcript            시드 열여덟이 한 번씩 돈다.  스킵이 아니다
   -fuzz=FuzzParse -fuzztime=60s            변이.  **사람이 돌린다**
```

**변이를 실제로 돌렸다** — 2026-09-16 · 60 초 · **1,298,261 회 실행 · 실패 0**.
새 interesting 31 개. 불변식 넷(F1 패닉 0 · F2 자른 `Text` 의 UTF-8 ·
F3 줄을 안 버린다 · F4 경계 산술)이 전부 섰다.

**퍼즈 하나로 SECURITY-13 을 닫았다고 적지 않는다.** 퍼즈가 안 닫는 다섯이 있다.

| | 퍼즈가 안 닫는 것 | 무엇이 닫나 | 이 단계에서 |
|---|---|---|---|
| ① | 누출 — 본문이 껍데기에 실렸나 | `assertNoLeak` 의 고정 문자열 셋 | `TestShell_NoBodySurvivesOnAnyPath` 가 픽스처 다섯에 건다 |
| ② | 짓는 쪽과 읽는 쪽의 일치 | 왕복 시험 | `roundtrip_test.go` 둘 |
| ③ | `Fields` 를 `map[string]any` 로 바꾸는 것 | **아무 기계도 안 막는다.** 사람이 diff 로 본다 | 안 닫힘 |
| ④ | `enode.capped` 의 글자와 뜻 | 사람이 맞댄다 | **반쪽만.** 4.2 |
| ⑤ | 시험 파일의 임포트 | 사람이 본다 (`go list -deps` 가 `-test` 가 아니다) | 안 닫힘 |

---

## 9. 진행자에게 넘기는 것

**①~⑥ 은 계획 12절이 이미 적은 것이고, ⑦~⑩ 이 Part 2 가 새로 찾은 것이다.**

```text
   ①  팩 requirements/transcript/decisions.md:50 의 「200자 · 500자」를
      「200바이트 · 500바이트」로.  단위를 이 유닛이 정했고 팩은 안 만진다

   ②  F2 의 글자를 고친다 — nfr-requirements.md 2.2 · 4.2 ·
      business-rules.md 8.2.  「입력이 올바른 UTF-8 이면」 조건이 앞에 붙는다.
      값은 안 바뀐다

   ③  회차 user-stories.md US-4 의 확인란 「raw 사건」을
      「plain text 또는 raw — 둘 다 줄 원문을 Text 에 든다」로

   ④  ci.yml 에 변이 퍼즈를 넣을지.  이 계획의 판단은 「넣지 말 것」이다

   ⑤  .coverage-contract.yml 의 packages: 기준선이 15 -> 16 행이 된다

   ⑥  U3 이 병합된 뒤 4.2 의 한 줄로 enode.capped 를 맞댄다.
      **U1 은 그 절반을 못 닫는다**

   ⑦  **새것.**  as=events 의 사건 수 상한을 U4 가 진다.
      최악이 계획의 112 MB 가 아니라 **2.3 GB** 다 (7.2 의 실측).
      개행만 든 10 MiB 입력이 사건 1,048 만 개를 낸다.
      **U1 은 상한을 안 건다** — 계획 10절이 그것을 짓지 않는 것에 넣었다

   ⑧  **새것.**  계획 자신의 두 자리가 실측과 갈렸다.
        Step 17 「파싱 비용이 입력의 약 2배인지 (-benchmem 의 B/op)」
          -> 붙든 것은 1.67x 이고 B/op 는 24.8x 다.  재는 값이 둘이다 (7.1)
        Step 16 ④ 「R8 ③ 과 F2 가 빨강」
          -> F2 는 안 빨개졌다.  시드가 Cut > 0 을 한 번도 안 냈다 (6.1)

   ⑨  **새것.**  물음 1 의 규칙 번호가 비어 있다.
      값은 실측이 정했고(A · 3.2) 시험이 묶었지만, 그것을 business-rules.md 의
      R 번호로 올리는 것은 앞 단계의 문서라 이 유닛이 안 고친다.
      **적을 말** — 「tool_result 의 content 가 문자열이면 그대로 쓰고
      그 밖의 모양이면 Text 를 비운다.  배열 꼴은 실측했고(2026-09-16)
      본 것은 image 원소 하나뿐이라 건질 본문이 0 이었다」

   ⑩  **새것.**  .coverage-contract.yml 의 command.env 가 전제 하나를 빠뜨렸다.
      ENODE_TEST_DATABASE_URL 만 적혀 있는데 **.github/ci-stubs 를 PATH 에**
      두는 것도 전제다.  실측 — 스텁 없이 표준 명령을 돌리면 internal/api 의
      시험 72 개가 실패해 전체가 83.1% 로 읽히고 미달이 2 개가 된다.
      스텁을 넣으면 87.5% · 미달 0 이다.  그 파일은 주인이 따로다 (U7)
```

**이 유닛이 회차 문서도 팩도 안 고쳤다.** `aidlc-state.md` 와 `audit.md` 도
안 건드렸다.

---

## 10. 계획과 갈린 자리 — 다섯

**전부 실측이 찾았고, 넷은 계획이 틀렸고 하나는 재개 지점의 기록이 틀렸다.**

| | 계획 · 기록이 적은 것 | 실측 | 무엇을 했나 |
|---|---|---|---|
| ① | 재개 지점 「`testdata/lines` 가 빈 디렉터리다」 | **픽스처 열 개가 이미 있었다** (앞 회차가 쓴 것) | 안 지웠다. 빠진 다섯을 더했다 |
| ② | Step 12 「`selectLogs` 의 **다섯** 자리」 | 표의 행이 여섯이고 부르는 자리도 **여섯**이다. 서로 다른 함수가 다섯 | 여섯을 다 걸었다 |
| ③ | Step 16 ④ 「F2 가 빨강」 | F2 는 초록이었다 — 시드가 `Cut > 0` 을 0 번 냈다 | 픽스처 하나를 더해 F2 를 세웠다 |
| ④ | Step 16 ⑤ 「R1 · R5 가 빨강」 | **전부 초록이었다** — 줄기 둘 중 하나만 쟀다 | R5 를 표로 넓히고 R1 입력에 그 줄을 넣었다 |
| ⑤ | Step 17 「약 2배 · `B/op`」 · 「최악 112 MB」 | 1.67x(붙든 것) · 24.8x(`B/op`) · 최악 **2.3 GB** | 둘 다 적고 U4 에 넘겼다 (9절 ⑦ ⑧) |

**계획에 없이 더한 것은 픽스처 둘뿐이다.**

```text
   long-tool-result.json    F2 를 공허하지 않게 한다 (6.1).  Step 11 이 요구한
                           불변식이 실제로 서게 하는 데 필요했다
   assistant-no-blocks.json 원문 줄기의 되돌림을 재게 한다 (6.2).
                           Step 16 ⑤ 가 요구한 빨강이 실제로 나게 하는 데 필요했다
   user-result-array.json   Step 2 가 「실측한 모양을 손으로 지어 넣는다」로 이미 요구했다
```

**셋 다 시험이 실제로 재게 하려고 더한 것이고, 제품 코드는 한 줄도 안 바뀌었다.**

---

## 11. 남은 것

```text
   못 닫은 것   enode.capped 의 U3 쪽 절반 (4.2).  **초록이라고 안 적는다**
   안 닫는 것   Fields 를 map[string]any 로 바꾸는 것 · 시험 파일의 임포트 (8절 ③ ⑤)
   안 지은 것   as=events 의 사건 수 상한 — U4 의 것이다 (9절 ⑦)
   안 고친 것   회차 문서 · 팩 · aidlc-state.md · audit.md — 진행자의 것이다
```
