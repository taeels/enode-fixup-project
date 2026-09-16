# U4 `log-api` — Code Generation 요약

```text
   유닛    log-api (U4) · 브랜치 unit/log-api · 담당 taeels
   계획    ../../plans/log-api-code-generation-plan.md
   잰 날    2026-09-16 · 기준선 5d53e52
   지는 게이트  CB0 — 라우트 17 -> **18**
```

**잰 것만 적는다.** 안 돈 것과 못 잰 것은 6절이 이름으로 든다.

---

## 1. 무엇이 생겼나

| 파일 | 새것/고침 | 줄 |
|---|---|---|
| `internal/api/log.go` | **새 파일** | 238 |
| `internal/api/getlog_test.go` | **새 파일** | 405 |
| `internal/transcript/wire_test.go` | **새 파일** | 86 |
| `internal/api/api.go` | 고침 | +43 −3 |
| `internal/record/record.go` | 고침 | +25 |
| `internal/transcript/transcript.go` | 고침 | +44 −29 (태그와 주석) |
| `internal/record/record_test.go` | 고침 | +72 |
| `internal/api/log_test.go` | 고침 | +6 −5 (204 -> 200) |

```text
   go.mod · go.sum      diff 0
   internal/store       diff 0
   U2 와의 파일 교집합    **0**
```

---

## 2. 실측이 계획을 고친 것 — 넷

### 2.1 태그를 달 타입이 하나가 아니라 다섯이다

계획은 「`Event` 에 json 태그」로 적었다. 실측하니 **선 위로 나가는 타입이
다섯**이다 — `Event` · `Info` · `Server` · `Elided` · `Result`.

`Server` 와 `Elided` 에는 **태그를 안 다는 근거가 주석으로 적혀 있었다** —
「`encoding/json` 이 키를 대소문자 무시로 맞춘다」. **그 관용은 읽을 때만
있다.** 그때 이 타입들은 읽히기만 했고, 이제 `as=events` 가 쓴다. 태그가
없으면 키가 `Name` · `Status` · `Events` 로 나가 이 저장소의 다른 응답
(`run_id` · `step_id`)과 모양이 갈린다. **주석을 지우지 않고 뒤집힌 이유까지
적었다.**

### 2.2 `Info` 에는 `omitempty` 가 안 듣는다 — `omitzero` 다

`Info` 는 구조체라 `omitempty` 가 아무 일도 안 한다. 그대로 두면 **사건마다
`"info":{}` 가 실린다.** 사건 대부분이 그 모양이라 배열 전체에 노이즈가 된다.
Go 1.24 의 `omitzero` 가 그 자리를 닫는다 (`go.mod` 가 `go 1.26`).

### 2.3 `Bytes` 태그 하나가 빠진 것을 시험이 잡았다

`Info.Bytes` 만 태그가 안 붙었고 **선 위에 `"Bytes":9` 로 나갔다.** 시험을
먼저 쓴 덕에 그 자리에서 빨갰다. **필드 이름이 새어 나가는 것을 재는 줄이
같은 시험에 있다** — 태그가 빠진 자리를 그것이 잡는다.

### 2.4 줄 경계를 보려고 한 바이트 앞에서 읽는다

`as=events` 는 조각이 줄 머리에서 시작하는지를 알아야 한다. 계획은 「`from-1`
바이트가 개행인가로 정한다」로만 적었고, **그 바이트를 어떻게 얻느냐**가 비어
있었다. 파일을 두 번 여는 대신 `from-1` 에서 읽기 시작해 첫 바이트를 보고
몸통에서 뺀다 — 여는 횟수가 안 늘고 갈래도 안 는다.

---

## 3. 변이 다섯 — 전부 빨강. 넷째가 한 번 살아남았다

```text
   ①  개행에서 안 끊는다          ASlicedBodyEndsOnANewline · EventsSurvive  FAIL
   ②  봉인 뒤에도 progress        TheSourceSplitsAtTheSeal                  FAIL
   ③  Bytes 에 조각 길이          TotalIsTheFileAndFromResumes              FAIL
   ④  머리가 잘린 것을 안 말한다    **처음에는 살아남았다**                     -> FAIL
   ⑤  OK 의 omitempty 를 뗀다     TestWire_AbsentIsNotFalse                 FAIL
```

**④ 가 살아남은 이유가 값이다.** 폴링이 규칙대로 돌면 다음 `from` 이 언제나
줄 머리라 (①의 개행 규칙이 그렇게 만든다) **그 갈래를 한 번도 안 밟는다.**
시험 전부가 규칙대로 도는 폴링이었다.

밟는 길이 둘이다 — 사람이 손으로 `from` 을 줄 때와, 한 줄이 상한보다 길어
중간에서 끊긴 다음 폴링. 그 둘을 재는 시험을 따로 지었다
(`AMidLineFromIsReportedAsACutHead` · `ALineLongerThanTheCapStillMovesForward`).
**그 뒤 ④ 가 빨개졌다.**

**안 지었으면 반쪽 줄이 `raw` 사건으로 화면에 그려진다** — 깨진 JSON 한 줄이
원문 토글에 나타나고, 그것을 본 사람은 하네스가 깨진 것을 봤다고 생각한다.

---

## 4. 게이트 — 진행자가 직접 쟀다

```text
   **CB0**             라우트 17 -> **18**.  는 것은 GET 하나뿐이다
   build · vet         초록          gofmt          빔
   glyphscan           110 파일 · 0   전체 시험      초록 · **스킵 0**
   커버리지 (정본 명령)   19 패키지 **미달 0** · 전체 87.4%
                       internal/api        82.1% -> **82.3%**
                       internal/transcript 94.4% -> **95.3%**
                       internal/record     84.4% 그대로
   diff 0 이어야 할 곳    internal/store · go.mod · go.sum  전부 0
   git status          probe.lock 만 바뀌어 되돌렸다 (기준선의 성질)
```

---

## 5. 행렬 밖 파일 둘 — 예고한 그대로다

```text
   internal/record/record.go        OpenLog.  행렬은 U3 하나에 줬다
   internal/transcript/transcript.go  json 태그.  행렬은 U1 하나에 줬다
```

**둘 다 이미 병합된 유닛의 파일이라 병합 충돌이 0 이다.** 계획 6절이 이것을
미리 적었고 실제로 그대로 됐다.

`internal/transcript` 의 변경은 **동작이 0 이다** — 태그와 주석뿐이고 U1 의
왕복 시험과 퍼즈가 그대로 초록이다.

---

## 6. 안 돈 것과 못 잰 것

```text
   N1                 안 닫았다.  계획 0.2 의 산수가 봉투이고 실측은 CB4 의 몫이다
   1 MiB 의 적정성      쟀다기보다 골랐다.  한 요청이 가져가는 양을 막는 값이고,
                      진행 파일의 10 MiB 와 다른 단위다.  CB4 가 뒤집으면 상수 하나다
   데모의 무인증        코드로 안 막았다 (답 2 = A).  read() 가 데모에서 s.limit 이라
                      토큰 없이 원문이 나간다.  막는 것은 운영의 규칙이다
   CB1 · CB2 · CB4     화면이 선 뒤다.  이 유닛은 라우트만 낸다
   봉인 쪽 attempt      언제나 0 이다.  logs/NN-*.log 는 이름에 시도가 없다 —
                      값이 없는 것이지 못 잰 것이 아니다
   전역 한도            안 건드렸다.  v1 이 값을 보고 고른 자리다
```

---

## 7. 진행자에게 넘기는 것

계획 6절의 여섯 그대로이고, 하나가 는다.

```text
   internal/transcript 의 주석 둘   「JSON 태그를 안 단다」의 근거가 뒤집혔다.
                                  읽기만 하던 타입이 이제 쓰이기도 한다.
                                  주석을 고쳐 적었으나 그 판단의 정본은
                                  component-methods.md 에 없다
```
