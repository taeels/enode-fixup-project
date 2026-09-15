# U1 `transcript` — 흐름과 이음매

**형식은 `domain-entities.md` 가 들고 여기는 순서다.** 무엇이 어떤 순서로
지나가며, 짓는 쪽과 읽는 쪽이 왜 한 패키지에 사는지를 적는다.

---

## 1. `Parse` 의 파이프라인 — 여덟

```go
func Parse(b []byte, truncated bool) Result
```

```text
   ①  머리 자르기    truncated 가 참이면 첫 개행 뒤부터 읽는다.
                    버린 바이트 수를 Result.Head 에 적는다.
                    개행이 하나도 없으면 읽을 것이 0 이다 — 빈 Result 를 낸다

   ②  꼬리 자르기    b 가 개행으로 안 끝나면 마지막 조각을 떼고 그 바이트 수를
                    Result.Partial 에 적는다 (Q6 = A).  떼고 나서 줄을 나눈다

   ③  줄 나누기      SplitLines.  selectLogs 와 같은 함수다 (Q8 = A)

   ④  줄마다 판정    2절의 사다리.  여기서 Kind 와 Sub 가 정해진다

   ⑤  사건 짓기      3절.  한 줄이 사건 하나 이상을 낸다 (Q1 = A)

   ⑥  표시 줄 가르기  type 이 enode. 로 시작하는 줄은 enode 가 찍은 것이다.
                    enode.elided 는 사건이 아니다 — Result.Elided 에 담는다.
                    enode.capped 는 사건 하나다 — 그 자리에 선다 (2절 ②)
                    둘 다 Lines 에는 센다

   ⑦  붙이기         tool_result 의 Name 을 같은 ID 의 tool_use 에서 채운다 (5절)

   ⑧  세기           Lines · Raw 를 채운다.  불변식은 business-rules.md R1
```

**①과 ②가 ③ 앞인 것이 순서의 값이다.** 뒤에 두면 잘린 조각이 줄로 세어져
`Lines` 가 부풀고, 그 수로 세는 불변식(R1)이 거짓이 된다.

**시계를 안 받는다.** 같은 입력에 언제나 같은 `Result` 를 낸다 — NC-1 의 경과와
NC-6 의 갱신 시각은 화면이 뺀다 (`application-design.md` 6절 · Q3 = A).

---

## 2. 줄 하나의 판정 사다리 — 세 계단

**순서가 값이다.** 위에서 걸리면 아래를 안 본다.

```text
   ① JSON 객체로 풀리나 · type 이 문자열인가          ParseLine 이 그대로 답한다
        아니다  ->  text · Sub = "plain" · Text 는 줄 원문        (Q3 = A)

   ② type 이 "enode." 로 시작하나 — enode 가 찍은 표시 줄이다   (R3.5)
        enode.elided  ->  사건이 아니다.  Result.Elided 로 간다
        enode.capped  ->  capped 사건 하나.  Info.Bytes 에 bytes —
                          그 값은 **닿은 상한**이다 (R3.6 · R3.9)
        그 밖         ->  raw.  아는 둘에만 자리를 준다            (R3.7)

   ③ type 이 assistant · user 인가
        그렇다  ->  message 키가 있으면 원문, 없으면 껍데기         (Q4 = A)
        아니다  ->  껍데기냐 원문이냐를 안 묻는다 (2.2)

   ④ 3.2 의 사상표로 Kind 를 고른다
        표에 없다  ->  raw · Sub = 그 type                        (Q2 = A)
```

### 2.1 ①이 `raw` 가 아니라 `text` 인 이유

계획 1.5 가 그런 줄이 오늘 세 경로에 있음을 쟀다 — 명령 단계의 링과 로그(평문
전부)와 선별본 꼬리의 stderr 다. 셋 다 **사람이 읽으라고 있는 글자**다.
`raw` 로 떨어뜨리면 CB1 의 「읽을 수 있는 문장이 흐른다」가 명령 단계에서 안 선다.

**`Sub == "plain"` 이 그 대가를 갚는다.** 화면은 그것이 에이전트의 말이 아니라
JSON 이 아니었던 줄임을 알고 다르게 그린다 (`business-rules.md` R3).

**`text` 가 두 사실을 지게 된 것을 이름으로 적는다** — 「에이전트가 말했다」와
「JSON 이 아니었다」다. 가르는 것은 `Sub` 하나다.

### 2.2 ③을 `assistant` · `user` 에만 거는 이유

`Shell()` 이 본문을 걷는 대상이 그 둘뿐이다 (`runner.go:434` 의 `message`
블록). 그 둘의 원문에는 `message` 키가 언제나 있고, 그 둘의 껍데기에는 절대
없다 — `logShell` 의 필드 다섯에 `message` 가 없다.

`system` · `result` · `rate_limit_event` 에 안 거는 이유는 **가를 것이 없기
때문**이다. `system/hook_response` 의 원문은 `stdout` 을 들고 껍데기는 안 드는데,
그 Kind 가 `raw` 이고 `raw` 의 `Text` 는 줄 원문 그대로다. **원문이면 긴 줄이
보이고 껍데기면 짧은 줄이 보인다 — 둘 다 그 줄의 진실이다.**

`init` 과 최종 `result` 는 `selectLogs` 가 전문으로 남기므로 (`runner.go:341-344`)
껍데기로 올 수 없다. 둘째 `init` 은 껍데기로 떨어지는데
(`logs_test.go:266` 이 그것을 잰다) 그때 `Info` 가 제로값이고 그것이 옳다 —
그 줄에 값이 실제로 없다.

### 2.3 ②가 ④보다 위인 이유 — `capped` 가 `raw` 로 떨어지면 NC-4 가 반만 선다

`enode.capped` 는 JSON 이고 `type` 이 문자열이다. 사다리에 ②가 없으면 ④의
사상표에서 못 찾아 `raw` 로 떨어지고, `raw` 의 `Text` 는 줄 원문이다
(`domain-entities.md` 4.1).

```text
   ②가 있으면   화면이 「상한 N 에 닿아 멈췄다」를 문장으로 그린다.
                N 은 Info.Bytes 이고 **닿은 상한**이다 — 총 길이가 아니다 (R3.9)
   ②가 없으면   화면이 {"type":"enode.capped","bytes":10485760} 을 그대로 그린다
```

**US-7 이 막으려는 오독이 그 자리에서 돌아온다** — 「모르면 단계가 멈춘 줄
안다」. 원문 JSON 한 줄은 그것을 못 막는다.

**같은 이유로 ②가 ③보다도 위다.** `enode.*` 는 `assistant` · `user` 가 아니라
③에 안 걸리지만, 순서를 붙여 두면 뒤에 enode 가 다른 표시 줄을 더해도 그것이
껍데기 판정을 먼저 지나는 일이 없다.

**모르는 `enode.*` 를 `raw` 로 두는 것은 반대 방향으로 옳다** (R3.7). 우리가
뜻을 모르는 줄에 화면이 문장을 지어 붙이면 그것이 거짓이 된다. 허용목록의 규율이
여기서도 같다 — **아는 것에만 자리를 준다** (R2).

---

## 3. 줄 하나가 사건 여럿이 된다 (Q1 = A)

### 3.1 원문 줄기

```text
   assistant   message.content[] 의 블록마다 사건 하나.
               thinking · text · tool_use 순서 그대로.  모르는 블록 종류는 건너뛴다 —
               버리는 것이 아니다.  그 줄의 원문은 원문 토글이 언제나 든다

   user        message.content[] 의 tool_result 블록마다 사건 하나

   init        사건 하나.  Info 를 채운다
   result      사건 하나.  Info 를 채운다
   raw         사건 하나
```

**블록이 하나도 없는 `assistant` 원문**(`message.content` 가 빈 배열)도 사건
하나를 낸다 — `text` · `Text` 빈 문자열 · `Shell` 거짓. 「말을 안 했다」가 사실로
남아야 한다 (`business-rules.md` R5).

**`wire_tool_inputs` 와 `tool_use_result` 를 안 읽는다.** 실측이 도구 입력과
결과가 `message.content[]` **밖에도** 실림을 보였는데 (⑳), 그것은 같은 값의 두
번째 사본이다. 읽으면 사건이 두 벌이 된다.

### 3.2 껍데기 줄기

```text
   tools 한 칸마다   tool_use 하나.  Name 만 있고 Text 는 비었다.  Shell 이 참
   ok 하나          tool_result 하나.  OK 만 있고 Text 는 비었다.  Shell 이 참
   블록이 0         사건 하나.  text · Text 빈 문자열 · Shell 참        (R5)
   tokens           그 줄이 낸 첫 사건에 싣는다                        (R6)
```

**`tokens` 를 첫 사건에만 싣는 이유** — 와이어의 `usage` 는 **줄 하나의**
예산이지 블록 하나의 예산이 아니다. 사건마다 복사하면 화면이 합계를 내다가
같은 값을 여러 번 더한다.

원문 줄기도 같다 — `message.usage` 는 그 줄이 낸 첫 사건이 든다.

---

## 4. 두 줄기가 같은 Kind 로 모인다

**이것이 「같은 모양으로 읽힌다」의 기계적 뜻이다.**

| 사실 | 원문 줄기 | 껍데기 줄기 | 나오는 Event |
|---|---|---|---|
| 도구를 불렀다 | `content[].tool_use.name` | `tools[]` 한 칸 | `tool_use` · `Name` 있음 |
| 도구 입력 | `content[].tool_use.input` | **없다** | `Text` 있음 / `Text` 비고 `Shell` 참 |
| 도구가 실패했다 | `content[].tool_result.is_error` | `ok` 가 거짓 | `tool_result` · `OK` 거짓 |
| 도구 결과 본문 | `content[].tool_result.content` | **없다** | `Text` 있음 / `Text` 비고 `Shell` 참 |
| 에이전트가 말했다 | `content[].text` | **없다** | `text` |
| 예산 | `message.usage` | `tokens` | `Tokens` |

**갈리는 것은 `Text` 와 `Shell` 뿐이다.** Kind 의 열은 두 줄기에서 같다 —
`tools` 가 둘이면 `tool_use` 사건이 둘이고, 원문에서 블록이 둘이어도 둘이다.

**FR-3 의 수용 기준이 여기까지 참이다** (9절이 그 범위를 좁혀 적는다).

---

## 5. `tool_result` 를 `tool_use` 에 붙인다

```text
   붙이는 값   tool_result 사건의 Name.  같은 ID 의 tool_use 에서 가져온다
   붙이는 때   ⑦.  사건을 다 지은 뒤 한 번 훑는다
   못 붙이면   Name 이 빈 채로 선다.  그대로 사건이다 (FR-3 의 글자)
```

**배치는 안 한다.** 파서는 「이 결과가 어느 호출의 것인가」를 **값으로** 주고,
그 둘을 화면에서 어떻게 겹쳐 그릴지는 카드의 것이다 (U5 · U6 · U8).

### 5.1 못 붙이는 경로 셋

```text
   ①  링이 감겨 호출 줄이 잘렸다      tool_use 가 입력에 없다.  ID 는 있고 짝이 없다
   ②  껍데기 줄기                    ID 가 아예 없다 — logShell 에 자리가 없다.
                                    OK 가 사건 하나에 하나뿐이라 도구가 여럿이면
                                    어느 것의 결과인지 **알 수 없다**.  안 붙인다
   ③  tool_use_id 가 줄에 없다        아래 5.2
```

**②에서 자리로 짝짓지 않는다.** 「가장 가까운 앞의 `tool_use`」로 붙이면
도구가 둘인 턴에서 절반이 틀리고, **틀린 붙이기는 안 붙인 것보다 나쁘다** —
읽는 사람이 그 줄을 근거로 쓴다.

### 5.2 `tool_use_id` 의 존재를 이 단계가 실측으로 못 댄다

`requirements/harness-components/decisions.md` 6절 ⑳ 의 실측 표에 그 키의 행이
없다. `internal/enode/logs_test.go` 의 픽스처도 손으로 지은 최소형이라
`tool_use_id` 가 없다.

```text
   이 단계가 정한 것   있으면 그것으로 붙이고, 없으면 안 붙인다.
                     규칙은 키의 존재에 안 기댄다 — 없으면 ①과 같은 자리로 떨어진다

   Code Generation 이 할 것   실물 stream-json 에서 그 키를 확인하고,
                            없으면 붙이기 경로가 원문에서도 안 도는 것을 기록한다
```

**짐작으로 메우지 않는다.** 이 문장이 그 빈자리를 이름으로 진다.

---

## 6. 입력 넷 — 각각이 타는 줄기

`components.md` 1절이 둘로 적었다. 코드를 따라가면 넷이다 (계획 1.5).

| 입력 | 어디서 오나 | `truncated` | 타는 줄기 |
|---|---|---|---|
| 하네스 단계의 링 | `claim.go:794` 의 `Job.Transcript` (U2 가 되살린다) | `Total > capacity` | 원문. 머리가 잘린다 |
| 하네스 단계의 `logs/` · 진행 파일 | `runner.go:261` 의 `selectLogs` 결과 | 거짓 | **껍데기 + 전문 둘 + 표시 줄 + stderr 평문** |
| **명령 단계의 링** | `claim.go:633` 의 `io.MultiWriter(&buf, w.ring)` | `Total > capacity` | **평문 전부** — `text` · `Sub = "plain"` |
| **명령 단계의 로그** | `claim.go:647` 이 `buf` 를 원문 그대로 올린다 | 거짓 | 같다 |

**명령 단계가 오늘 이미 링에 흐르고 있다** — 이 회차가 만드는 것이 아니다.
파서는 단계 종류를 모르고, 알 필요도 없다. **사다리 ①이 그것을 대신한다.**

**둘째 줄의 입력이 한 파일 안에서 줄기를 섞는다** — 앞은 JSON(껍데기와 전문),
꼬리는 평문(stderr)이다. 사다리가 줄마다 돌기 때문에 섞여도 옳게 갈린다.
그것이 Q4 = A 를 파일 단위가 아니라 줄 단위로 고른 이유다.

### 6.1 `enode.` 표시 줄이 어느 입력에 섞이나

**파서는 안 가린다 — 어느 입력에서 오든 같게 읽는다.** 아래는 오늘 코드에서
읽히는 사실이고, **찍는 조건과 찍는 파일은 U3 의 것**이다 (`domain-entities.md` 3.3).

```text
   enode.elided   selectLogs 를 지난 바이트에 있다.  하네스 단계의 logs/ 와
                  그것을 그대로 받는 진행 파일.  stderr 앞에 한 번

   enode.capped   U3 이 찍는다.  이 문서는 그 줄을 읽는 규칙만 지고
                  언제 찍히는지와 무엇을 상한으로 삼는지를 안 적는다 (R3.8).
                  bytes 를 **닿은 상한**으로 읽는 것만 여기 규칙이다 (R3.9)
```

**오늘 `record.go:70` 이 평문 한 줄을 박는다** — `... log truncated at %d bytes`.
그것은 JSON 이 아니므로 R3.1 로 `text` · `Sub = "plain"` 이 된다. **그 줄을
어떻게 바꾸는지는 U3 의 FD 다** (`application-design.md` D4 가 그 자리를
`AppendLog` 의 뜻 변경으로 이미 열었다). 이 문서가 그 값을 안 정한다.

---

## 7. `selectLogs` 와의 이음매 (Q8 = A)

### 7.1 남는 것과 옮긴 것

```text
   internal/enode 에 남는다   selectLogs 하나.  줄을 나누고 · 두 자리(첫 init ·
                            마지막 result)를 찾고 · 전문과 껍데기를 조립하고 ·
                            표시 줄을 끝에 붙이고 · stderr 를 잇는다

   internal/transcript 로 간다   그 조립이 쓰는 재료 전부 —
                               SplitLines · ParseLine · String · Bool · Int ·
                               Shell · ElidedMarker · usageTokens · logShell · elidedMark
```

**`selectLogs` 가 안 옮겨 가는 이유는 그 함수가 파서가 아니기 때문이다.**
그것은 「`logs/` 에 무엇을 남길 것인가」라는 **정책**이고, 정책은 그 파일을
쓰는 쪽의 것이다. `runner.go:312-314` 의 주석이 그 자리를 이미 적었다 —
「어댑터가 아니라 여기서 한다. 두 번째 하네스가 오는 날 이 줄이 인터페이스로
올라간다」. 옮기면 그 근거가 갈 곳을 잃는다.

### 7.2 임포트의 방향

```text
   internal/enode  ->  internal/transcript     허용.  이 유닛이 만든다
   internal/transcript  ->  internal/enode     금지.  business-rules.md R9
```

**방향이 유일하게 금지를 안 건드리는 쪽이다** (`application-design.md` D5).

### 7.3 `selectLogs` 가 고쳐지는 자리 넷

```text
   runner.go:316   lines := splitLines(...)          -> transcript.SplitLines
   runner.go:326   parseEventLine(ln)                -> transcript.ParseLine
   runner.go:330   eventString(obj, "subtype")       -> transcript.String
   runner.go:352   eventShell(obj, typ)              -> transcript.Shell
   runner.go:361   elidedMarker(events, elided)      -> transcript.ElidedMarker
```

**`selectLogs` 의 로직은 한 줄도 안 바뀐다.** 부르는 이름만 바뀐다 — 그래서
`logs_test.go` 의 `selectLogs` 시험 아홉이 그대로 초록이어야 한다. 그것이 이
옮기기가 옳게 됐는지를 재는 가장 싼 자리다.

### 7.4 도우미 넷을 공개하는 것이 표면을 안 늘린다

`Shell(obj, typ)` 을 부르려면 호출자가 `Fields` 를 손에 들어야 한다. 그 맵이
이미 공개 표면에 있으므로 **그것을 읽는 넷을 공개하는 것이 새 표면이 아니다** —
안 공개하면 호출자가 `json.Unmarshal` 로 같은 일을 다시 짓고, 그것이
CONVENTIONS 1.4 가 막는 두 벌이다.

---

## 8. 짓는 쪽과 읽는 쪽이 한 패키지인 구조가 무엇을 막나

```text
   오늘        eventShell 은 internal/enode 에 있고 그것을 읽는 코드는 없다.
              키 이름이 한 자리에만 있어 갈릴 수 없다

   이 회차 뒤   읽는 코드가 생긴다.  Shell 이 "tools" 로 쓰고 Parse 가 "tools" 로
              읽는다.  둘이 다른 패키지면 한쪽만 고쳐도 컴파일이 되고,
              틀리는 순간은 봉인된 Run 을 화면에서 열 때다 — 몇 주 뒤다
```

**Q4 = A 가 그 구조를 고른 값이다** (`components.md` 1절 — 「짓는 쪽과 읽는
쪽이 한 패키지에 산다. 두 벌로 두면 조용히 갈린다」).

**시험이 그것을 잰다.** `Shell` 이 지은 바이트를 `Parse` 에 그대로 먹여 사건이
나오는 왕복 시험 하나가 이 구조의 값을 지킨다 — `Shell` 의 JSON 태그를 바꾸면
그 시험이 같은 패키지 안에서 빨개진다.

---

## 9. FR-3 의 수용 기준이 어디까지 참인가

팩과 회차 요구가 이렇게 적었다.

```text
   features.md 3.2   「같은 로그를 링에서 읽든 record 에서 읽든 같은 사건 열이
                     나온다 — 테스트가 그것을 센다」
```

**「같은 사건 열」은 참이고 「같은 사건」은 거짓이다.** 봉인된 `logs/` 는
선별본이라 본문이 없다 (`requirements.md` 2.3).

```text
   같다       Kind 의 열.  순서.  도구 이름.  성공 여부.  토큰 수
              -> 4절의 표가 그 목록이다

   다르다     Text.  원문 줄기는 본문이 있고 껍데기 줄기는 비었다
              Shell.  껍데기 줄기에서만 참이다
              ID.  껍데기에는 없다 — 그래서 붙이기가 안 돈다 (5.1 ②)
```

**시험이 재는 것을 그 범위로 적는다** — 같은 stdout 을 ① 원문 그대로 `Parse`
하고 ② `selectLogs` 를 지나 `Parse` 해서, **Kind 의 열과 `Name` 과 `OK` 가
같고 `Text` 만 다름**을 잰다. 「전부 같다」로 적으면 그 시험을 쓸 수가 없고,
쓸 수 없는 수용 기준은 다음 사람에게 규칙이 아니라 분위기로 간다.

**회차 문서를 고쳐야 할 자리다** (`plans/transcript-functional-design-plan.md` 6절).
