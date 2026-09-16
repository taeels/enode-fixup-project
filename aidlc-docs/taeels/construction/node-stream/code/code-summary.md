# U2 `node-stream` — Code Generation 요약

```text
   유닛    node-stream (U2) · 브랜치 unit/node-stream · 담당 taeels
   계획    ../../plans/node-stream-code-generation-plan.md
   잰 날    2026-09-16 · 기준선 5d53e52
```

**잰 것만 적는다.** 안 돈 것과 못 잰 것은 6절이 이름으로 든다.

---

## 1. 무엇이 생겼나

| 파일 | 새것/고침 | 줄 |
|---|---|---|
| `internal/enode/emit.go` | **새 파일** | 178 |
| `internal/enode/emit_test.go` | **새 파일** | 298 |
| `internal/enode/ring_roundtrip_test.go` | **새 파일** | 89 |
| `internal/enode/runner.go` | 고침 | +57 −9 |
| `internal/enode/claude.go` | 고침 | +50 −5 |
| `internal/enode/harness.go` | 고침 | +22 −6 |
| `internal/enode/claude_test.go` | 고침 | +48 −4 |
| `internal/enode/instrument_test.go` | 고침 | +14 −7 |

```text
   go.mod · go.sum      diff 0.  새 의존 0
   claim.go             diff 0 (R15)
   internal/transcript  diff 0 (R14)
   라우트                17 그대로 (R17)
```

---

## 2. 실측이 계획을 고친 것 — 다섯

### 2.1 `transcript.Parse` 는 개행 없는 줄을 안 읽는다

계획은 「줄마다 `transcript.Parse` 를 부른다」로만 적었다. 그대로 부르니
**사건이 0 이었다.** `parse.go` 의 2번이 개행으로 안 끝나는 꼬리를 떼어
`Partial` 에 센다 — 쓰는 중에 읽히는 링과 진행 파일을 위한 규율이다.

**여기 오는 줄은 이미 완결된 줄이라 그 규율의 대상이 아니다.** 두 자리 다
개행을 붙여서 넘긴다 (`emit.go` 의 `flushLine` · `claude.go` 의 `emitLine`).

### 2.2 `Close() error` 가 아니라 `Close() int64` 다

계획 2절이 `Close() error` 로 적었다. **이 겹에는 실패가 없다** (R6 · R7 이
그렇게 정했다). 언제나 nil 인 `error` 를 돌려주면 부르는 쪽이 검사할 것이
없는 값을 검사한다. 부르는 쪽이 실제로 원하는 값은 **버린 줄 수**다.

### 2.3 `Decode` 에 no-op 을 넘기면 `final` 이 사라진다

계획과 `business-logic-model.md` 가 「no-op 을 넘긴다」로 적었다. 그대로
했더니 **단계마다 한 번씩 찍히던 `final` 이 조용히 없어졌다** — 시험이 그
자리에서 빨갰다.

`final` 은 줄에서 나는 사건이 아니라 **봉투를 읽고 나는 판정**이라 배출기가
낼 수 없다. no-op 이 아니라 **거르개**가 답이다.

```go
   onlyFinal := func(e Event) { if e.Kind == EventFinal { emit(e) } }
```

**R4 의 뜻은 안 바뀐다** — 한 사건이 한 번 난다. 줄 사건은 배출기가, 봉투
사건은 `Decode` 가 낸다.

### 2.4 앞 팩의 시험 하나가 이 유닛의 반대를 재고 있었다

`TestRunHarness_TheRingGetsNothingWhileTheStreamIsRaw` 가 **링이 한 바이트도
안 받는 것**을 재고 있었다 (`decisions.md` 6절 ⑲). 이 유닛이 그 결정을
뒤집으므로 시험을 뒤집었다 — 이름까지 바꿨다 (`TheRingGetsTheRawStream`).

**느슨해진 것이 아니다.** 앞 판은 「한 바이트도 안 받는다」였고 새 판은
**본문 글자까지 확인한다** — 「무언가 받았다」로 재면 링에도 선별을 거는
구현이 통과하고, 그 구현은 화면이 읽을 문장을 0 으로 만든다.

### 2.5 상한 하나를 계획에 없이 더했다 — `maxEmitLineBytes`

개행이 안 오는 동안 배출기가 꼬리를 무한히 모은다. stdout 버퍼가 이미 전체를
들고 있으므로(봉투 파서가 뒤에서부터 전체를 훑는다) **같은 바이트가 세 벌**이
된다. 관측하는 겹이 메모리를 그렇게 쓰면 안 된다. 1 MiB 를 넘으면 그 줄을
버리고 다음 개행까지 건너뛴다 — 버리는 것이 이미 이 겹의 규율이다 (R7).

### 2.6 `Write` 의 패닉 방어는 `recover` 가 아니라 구조다

계획 Step 2 가 「`recover` 로 패닉까지 삼킨다」로 적었다. **그럴 자리가 없다** —
`Write` 는 `emit` 을 안 부른다. 부르는 것은 고루틴이고, `recover` 는 거기 있다
(`safeEmit`). 불변식(R6)은 그대로 서고 **더 세게 선다**: 구조상 `Write` 경로에
소비자의 코드가 아예 없다.

`recover` 를 고루틴에 둔 이유는 다른 것이다 — 고루틴 하나가 패닉으로 죽으면
프로세스가 죽는다. 관측하는 겹이 노드를 죽이면 그것은 관측이 아니다.

---

## 3. 사고 하나와 그것이 드러낸 구멍

**변이를 `git checkout --` 로 되돌리다 커밋 안 된 `runner.go` 작업을 지웠다.**
백업에서 되살렸고, 그 뒤로는 파일 사본으로 되돌린다.

**그 사고가 시험의 구멍을 드러냈다.** 되살리기 전에 돌린 시험이 **초록이었다** —
배출기가 통째로 없는데도 `TestRunHarness_EmitsEachEventOnce` 가 통과했다.
`Decode` 혼자서 같은 종류를 같은 순서로 내기 때문이다.

```text
   수를 세는 시험이 못 재는 것   시점.  이 유닛이 사는 값이 바로 그것이다
```

그래서 시험을 하나 더 지었다 — `TestRunHarness_EventsArriveBeforeTheStepEnds`.
하네스가 **첫 사건이 나야 생기는 문지기 파일**을 기다린다. 배출이 끝난 뒤에만
난다면 하네스는 영영 안 끝나고 `ctx` 가 그것을 죽인다. 변이 1 이 이 시험에서
20초 만에 빨개진다.

---

## 4. 변이 여섯 — 전부 빨강

```text
   ①  배출기를 tee 에서 뺀다              EventsArriveBeforeTheStepEnds  FAIL (20.05s)
   ②  Decode 의 줄 사건을 안 거른다        EmitsEachEventOnce             FAIL
   ③  Close 를 Decode 뒤로                EmitsEachEventOnce             FAIL
   ④  배출기가 본문을 싣는다               CarriesNoBody                  FAIL
   ⑤  대기열이 차면 막는다                 DropsWhenFull                  **멈춤** (600s)
   ⑥  감긴 링을 안 잘린 것으로 읽는다       RingRoundTrip                  FAIL
```

**⑤ 는 실패 메시지가 아니라 멈춤으로 빨개졌다.** `default:` 를 빼면 `Write` 가
대기열이 찬 순간 영영 막히고, **그것이 바로 그 상한이 막는 실패다** — 실물에서는
하네스의 stdout 이 그 자리에서 멈춘다. 되돌리는 데 10분이 들었다.

**③ 이 빨개지게 하려고 시험을 한 번 세웠다** — 줄 사건을 10 ms 씩 늦게 받게
해서 순서가 우연이 아니게 만들었다. 안 늦추면 추월이 대개 안 보인다.

---

## 5. 게이트 — 진행자가 직접 쟀다

```text
   build · vet         초록          gofmt          빔
   glyphscan           110 파일 · 0   전체 시험      초록 · **스킵 0**
   커버리지 (정본 명령)   19 패키지 **미달 0** · 전체 87.5%
                       internal/enode  86.7% -> **87.0%**
                       internal/transcript 94.4% -> **94.9%** (왕복 시험이 밟는다)
   라우트               **17 그대로**
   diff 0 이어야 할 곳    claim.go · internal/transcript · go.mod · go.sum  전부 0
   git status          probe.lock 만 바뀌어 되돌렸다 (기준선의 성질)
```

---

## 6. 안 돈 것과 못 잰 것

```text
   CB1               이 유닛이 안 진다.  화면이 U5 에서 선 뒤에 사람이 잰다.
                     여기서 낼 수 있는 가장 가까운 증거가 §3 의 시점 시험이다
   실물 하네스        안 돌렸다.  가짜 스크립트로만 쟀다 —
                     stream-json 의 줄 모양은 U1 이 실물로 이미 쟀다
   링의 동시 읽기      제어판이 쓰는 중에 읽는 경로는 앞 팩의 시험이 잰다.
                     이 유닛은 쓰는 쪽만 바꿨다
   잔여 셋            business-rules.md 6절 그대로 — stdout 을 통째로 든다 ·
                     버린 줄은 로그에서 사라진다 · 줄 넘는 붙이기가 없다
```

---

## 7. 진행자에게 넘기는 것

계획 6절의 다섯에 하나가 는다.

```text
   decisions.md 6절 ⑲    「하네스 단계의 링 tee 를 끈다」가 이 회차에서 뒤집혔다.
                        그 결정을 적은 자리와 그것을 재던 시험이 둘 다 이 커밋에
                        있고, 팩 문서는 아직 앞 판이다
```
