# U2 `node-stream` — 도메인 개체

```text
   유닛    node-stream (U2) · 웨이브 W-b · 브랜치 unit/node-stream
   계획    ../../plans/node-stream-functional-design-plan.md (답 일곱 전부 A)
   기준선   5d53e52.  W-a 가 병합된 main
```

이 문서는 **무엇이 있는가**다. 순서는 `business-logic-model.md` · 불변식은
`business-rules.md`.

---

## 1. 새로 생기는 것 하나 — 줄 배출기

```go
// lineEmitter 는 tee 의 한 갈래다. 쓰기 경로에서 줄만 가르고, 사건으로 읽는
// 것은 자기 고루틴이 한다.
//
// io.Writer 라 io.MultiWriter 한 겹으로 꽂힌다 — 링과 같은 모양이다.
type lineEmitter struct {
    lines   chan []byte  // 유계. 차면 버린다 (R7)
    dropped int64        // 버린 줄 수. Close 가 로그로 낸다
    emit    func(Event)
}

func newLineEmitter(emit func(Event)) *lineEmitter
func (e *lineEmitter) Write(p []byte) (int, error)  // 언제나 len(p), nil (R6)
func (e *lineEmitter) Close() error                 // 채널을 닫고 고루틴을 기다린다
```

**왜 채널인가** — `transcript.ParseLine` 이 줄마다 `map[string]json.RawMessage`
를 짓는다. 그것을 하네스의 쓰기 경로에서 하면 도구 결과 한 줄(수십 KB)마다
복사가 나고, 그 지연이 자식 프로세스의 stdout 에 되먹힌다. **관측이 실행을
느리게 만드는 자리**라 쓰기 경로에서 뺀다.

**왜 유계이고 버리나** — 이 겹이 지는 것은 관측이다. 채널이 차서 쓰기가 막히면
그 순간 실행이 막힌다. 링은 안 막히므로 **버려도 화면은 안 다친다** — 버린 것은
노드의 디버그 로그뿐이다.

---

## 2. 뜻이 바뀌는 것 — `EventKind`

```go
// 오늘
type EventKind string
const ( EventFinal EventKind = "final" )

// 이 유닛 뒤
type EventKind = transcript.Kind          // 별칭.  어휘가 한 벌이다 (답 2 = A)
const ( EventFinal EventKind = "final" )  // 남긴다 — Decode 의 마지막 사건이다
```

**별칭인 이유** — 정의 타입으로 두면 경계마다 변환이 생기고 그 변환이 하는 일이
0 이다. `transcript.Fields` 가 같은 이유로 별칭이다 (`transcript.go:33` 의 주석).

**`final` 은 일곱에 없다.** 하네스가 내는 종류가 아니라 `Decode` 가 봉투를 읽고
찍는 것이다. 그래서 상수를 남기고, 값이 일곱과 안 겹치는 것을 시험이 고정한다
(R11).

| | 값 | 누가 찍나 |
|---|---|---|
| 일곱 | `init` `text` `tool_use` `tool_result` `result` `raw` `capped` | `transcript.Parse` |
| 하나 | `final` | `Decode` 가 봉투에서 |

---

## 3. 모양이 바뀌는 것 — `Event`

```go
type Event struct {
    Kind EventKind
    Text string
}
```

**필드가 안 는다.** 바뀌는 것은 **누가 무엇을 채우는가**다.

```text
   배출기가 내는 사건    Kind 만 채운다.  Text 는 언제나 빈 문자열 (R5)
   Decode 가 내는 사건   오늘 그대로 — Kind=final · Text=봉투의 message
```

**배출기가 `Text` 를 안 채우는 이유** — FR-1 이 노드 로그에 본문을 못 싣게 했다.
소비자가 안 찍는 것에 기대면 다음 소비자가 찍는다. **담기지 않으면 찍힐 수 없다.**

---

## 4. 배선이 바뀌는 것 — `runHarness` 의 stdout

```text
   오늘        cmd.Stdout = &stdout

   이 유닛 뒤   cmd.Stdout = io.MultiWriter(&stdout, j.Transcript, emitter)
              j.Transcript 가 nil 이면 그 갈래를 뺀다
```

```text
   갈래 셋     &stdout       봉투 파서가 읽을 바이트.  전체를 든다 (답 3 = A)
              j.Transcript  링.  화면이 1초로 읽는다 (FR-2)
              emitter       사건.  노드 로그가 종류만 읽는다 (FR-1)
```

**`emit` 이 두 자리에서 날 뻔했다.** 모순 검사가 그것을 잡았다 —
`business-rules.md` R4 가 답이다.

---

## 5. 안 바뀌는 것 — 이름으로 적는다

```text
   Decode 의 시그니처        agent-runtime R3.  FR-1 이 명시로 걸었다
   ParseClaude · lastJSONObject   봉투 파서.  FR-1 이 명시로 뺐다
   claudeHarness.Argv        짝 팩이 가져갔다 (claude.go:343)
   Ring 의 형식 · 크기 · 권한   앞 팩 §6.2 · §6.3.  내용만 NDJSON 이 된다
   Ring.Reset 의 시점         단계 시작 (claim.go:455)
   명령 단계의 tee            claim.go:632 의 io.MultiWriter(&buf, w.ring)
   Worker.transcript()       링 하나를 그대로 낸다.  업로더 갈래는 U7 (답 4 = A)
   internal/transcript       U1 이 냈다.  이 유닛은 임포트만 한다 — diff 0
```

---

## 6. 이 유닛이 만지는 파일

```text
   internal/enode/runner.go    tee 갈래 셋 · Decode 에 넘기는 emit
   internal/enode/claude.go    Decode 안을 줄 단위로
   internal/enode/emit.go      신규.  lineEmitter
   시험                        claude_test.go · runner 쪽 시험 · 왕복 시험 하나 (R12)
```

`claim.go` 는 **안 만진다.** 계획 1.4 의 실측대로 `Transcript` 도 `Emit` 도 이미
넘어오고 있다. 파일 행렬이 U2 에 `claim.go` 의 `transcript()` 를 줬으나 **답 4 = A
가 그 자리를 U7 로 미뤘다** — 행렬을 고칠 자리로 적는다.
