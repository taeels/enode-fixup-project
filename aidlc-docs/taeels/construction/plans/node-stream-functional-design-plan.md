# U2 `node-stream` — Functional Design 계획 (Part 1)

```text
   유닛     node-stream (U2) · 웨이브 W-b · 브랜치 unit/node-stream
   맡는 것   FR-1 (Decode 가 사건을 배출한다) · FR-2 (링 tee 를 되살린다)
   지는 게이트  없다.  CB1 의 재료다
   NFR      스킵 (회차 계획 — 새 외부 표면 0)
   기준선    5d53e52.  W-a 가 병합된 main.  internal/transcript 가 이미 있다
```

이 문서는 **묻는 것**이다. 답이 오면 4절의 순서로 산출물 셋을 낸다.

---

## 0. 이 단계가 닫는 것과 안 닫는 것

```text
   닫는다     사건의 모양과 배출 시점 · tee 갈래의 수와 실패 규율 ·
             링에 흘리는 것의 범위 · 시도가 바뀌는 자리
   안 닫는다   코드.  Code Generation 이 낸다
             업로더의 안 (U7) · 화면 (U5) · 라우트 (U4)
             봉투 파서 ParseClaude · lastJSONObject — FR-1 이 명시로 뺐다
             Argv — 짝 팩이 이미 가져갔다 (requirements.md 2.1)
```

---

## 1. 실측 — 코드가 지금 어떻게 생겼나 (2026-09-16)

**계획을 짓기 전에 읽었다.** W-a 가 배운 것이 이것이다 — 문서가 코드와 갈린
자리는 계획을 짓는 동안에만 싸게 찾힌다.

### 1.1 `Decode` 는 끝난 뒤에 불린다 — 안을 고쳐도 사건이 안 흐른다

```go
   runner.go:196   cmd.Stdout = &stdout          // bytes.Buffer
   runner.go:207   err = cmd.Run()               // 프로세스가 끝난다
   runner.go:209   res := h.Decode(bytes.NewReader(stdout.Bytes()), code, emit)
```

`component-methods.md` 3절은 **「`io.ReadAll` 을 걷고 읽으면서 emit 한다」** 로
적었다. 그 문장만 따르면 `Decode` 안은 줄 단위가 되지만 **부르는 자리가 끝난
뒤라 사건이 흐르는 시점은 한 톨도 안 바뀐다.** 값이 0 이다.

**바뀌어야 하는 자리는 `runner.go:196` 이다.** 이 실측이 물음 1 을 낳았다.

### 1.2 `Decode` 는 종료코드를 인자로 받는다 — 그래서 먼저 못 돈다

```go
   harness.go:267   Decode(r io.Reader, exitCode int, emit func(Event)) HarnessResult
```

`agent-runtime` R3 이 이 시그니처를 못 박았고 FR-1 이 「안 바뀐다」로 이었다.
**종료코드는 `cmd.Wait` 뒤에만 있다.** 그래서 `Decode` 를 파이프에 물려 먼저
돌리는 모양은 시그니처를 깬다 — 고루틴으로 빼도 인자를 못 채운다.

**흐르는 것과 판정하는 것이 다른 일이라는 뜻이다.** 물음 1 의 갈래가 여기서 갈린다.

### 1.3 `lastJSONObject` 가 뒤에서부터 **전체**를 훑는다

```go
   harness.go:172   func lastJSONObject(b []byte) string
                    // 줄 단위로 찾으면 안 된다 — 봉투가 여러 줄로 예쁘게 찍혀 올 수 있다
```

주석이 이유까지 적어 두었다. **그래서 `Decode` 는 줄 단위로 읽어도 바이트를
통째로 들고 있어야 한다** — 꼬리만 들면 여러 줄 봉투를 못 집는다.

`requirements.md` 5.7 이 「파서 — 줄 단위. 전체를 메모리에 다시 담지 않는다」로
적었는데 **그 문장이 가리키는 것은 `internal/transcript` 이지 `Decode` 가 아니다.**
둘을 한 문장으로 읽으면 이 유닛이 봉투 파서를 깨게 된다. 물음 3 이 그 자리다.

### 1.4 링은 이 자리를 위해 지어졌고 지금 끊겨 있다

`internal/enode/transcript.go` 머리말이 그대로 적는다.

```text
   하네스가 도는 동안 뱉는 글자는 지금까지 어디에도 없었다 — runner.go 가
   bytes.Buffer 로 통째로 받아 프로세스가 끝나야 Decode 했다.  이 파일이 그것을
   노드의 고정 크기 링 파일에 흘려, 제어판이 1초로 읽어 카드에 그리게 한다
```

**끊은 것은 앞 팩이다** (`runner.go:187` 의 ⑥ 주석 · `decisions.md` 6절 ⑲).
이유는 「이 tee 가 아래 선별보다 앞이라 걸러야 할 것이 그대로 링으로 나간다」였고,
이 회차의 질문 2 = A 가 그 값을 다시 사서 되살린다 (`requirements.md` 5.4 ③).

```text
   이미 서 있는 것   claim.go:794   Transcript: w.transcript()  가 이미 넘어온다
                    claim.go:455   ring.Reset() 이 단계 시작마다 돈다
                    claim.go:632   명령 단계의 tee 는 지금도 흐른다
   비어 있는 것      runner.go:196  하네스 단계만 tee 가 없다
```

**되살리는 데 필요한 변경이 한 줄이다.** 그것이 물음 1 의 갈래 하나를 싸게 만든다.

### 1.5 `EventKind` 에 값이 하나다

```go
   harness.go:285   type EventKind string
   harness.go:288   EventFinal EventKind = "final"
   harness.go:291   type Event struct { Kind EventKind; Text string }
```

주석이 「종류는 stream-json 을 켤 때 실물을 보고 늘린다」로 적었다. **지금이
그때다.** 늘리는 어휘를 어디서 가져오는지가 물음 2 다.

**소비자가 하나다** — `claim.go:793` 의 `log.Debug("harness event", "kind", e.Kind)`.
FR-1 이 「노드 로그에 본문을 안 싣는다. 그대로 둔다」로 그 자리를 잠갔으므로
**`Event.Text` 는 오늘 아무도 안 읽는다.**

### 1.6 유닛 정의의 한 줄이 이미 거짓이다

`unit-of-work.md` 의 U2 절이 이렇게 적었다.

```text
   딛는 게이트  없다.  U1 을 안 기다린다 — 링 tee 도 Decode 도 파서를 안 쓴다
```

**`internal/enode` 는 이미 `internal/transcript` 를 임포트한다** — U1 이
`selectLogs` 를 그렇게 고쳤다 (`runner.go:331` 의 `transcript.String`). D5 가
정한 것이고 `component-dependency.md` 가 허용 임포트로 적었다. 그래서 「파서를
안 쓴다」는 **오늘 코드에서 거짓**이고, 남는 뜻은 「U1 을 안 기다린다」뿐인데
U1 은 이미 병합됐다. 6절이 고칠 문서로 든다.

### 1.7 링은 감기면 첫 줄을 자른다

`Ring.Write` 가 `capacity` 를 넘으면 꼬리만 남기고 감는다 (`transcript.go:106`).
**줄 경계를 모른다.** 그래서 읽는 쪽이 보는 첫 줄은 반쪽일 수 있고, CB1 이
그것을 명시로 잰다 — 「첫 줄이 잘린 채로도 파싱된다」.

U1 의 `SplitLines` · `ParseLine` 이 그것을 감당하게 지어졌다. **이 유닛이 만드는
이음매다** — 짓는 쪽이 링이고 읽는 쪽이 파서라 어느 한 패키지의 시험도 왕복을
안 잡는다. W-a 가 `enode.capped` 에서 같은 모양을 한 번 겪었다. 물음 7 이 그 자리다.

### 1.8 안 만져도 되는 것을 확인했다

```text
   claude.go 의 Argv        짝 팩이 가져갔다.  실측 — claude.go:343 에 있다
   ParseClaude · lastJSONObject   FR-1 이 명시로 뺐다
   claim.go:632 의 명령 단계 tee   오늘 그대로 흐른다 (FR-2)
   Ring 의 형식 · 크기 · 권한      앞 팩이 값으로 닫았다.  내용만 NDJSON 이 된다
   internal/transcript          U1 이 냈다.  이 유닛은 읽기만 한다
```

---

## 2. 물음 일곱

형식은 `common/question-format-guide.md` 다.

**물음이 전부 A 로 닫혔다 (2026-09-16 · 사용자 「권장대로」).** 권장과 갈린 답이 0 이다.

### Question 1 — 사건을 흐르게 하는 자리를 어디로 잡나

1.1 과 1.2 가 함께 내는 물음이다. `Decode` 는 종료코드를 인자로 받아 프로세스가
끝나기 전에 못 돈다. 그런데 CB1 이 재는 것은 **단계가 끝나기 전에** 문장이
흐르는 것이다.

```text
   A   tee 갈래를 셋으로 만든다 — stdout 버퍼 · 링 · 줄 배출기.
       배출기가 줄 경계에서 emit 을 부른다 (도는 중).  Decode 는 시그니처대로
       끝난 뒤에 돌되 io.ReadAll 을 걷고 줄 단위로 읽는다 (FR-1 의 글자)
   B   tee 갈래를 둘로 두고 (stdout 버퍼 · 링) emit 은 Decode 안에서만 낸다.
       흐르는 것은 링이 지고 emit 은 끝난 뒤에 종류별로 한 번씩 난다
   C   Decode 를 고루틴으로 빼고 종료코드를 나중에 채운다 — 시그니처가 깨진다
```

**권장은 A 다.** 흐르는 것과 판정하는 것이 다른 일이고, A 만 둘을 각자의 자리에
둔다. C 는 R3 을 깨므로 값을 보기 전에 뺄 수 없다 — 이름으로 적는다.

**B 의 대가** — `Event` 가 도는 중에 안 나므로 나중에 누가 emit 에 화면을 물리면
그 자리에서 다시 이 결정을 해야 한다. **A 의 대가** — 배출기가 한 겹 더 생기고
그 겹의 실패 규율을 물음 6 이 정해야 한다.

[Answer]: A

### Question 2 — 사건 종류의 어휘를 어디서 가져오나

`EventKind` 에 값이 하나다 (1.5). FR-1 이 「낯선 줄은 `raw` 사건으로 넘긴다」를
요구하므로 최소 둘이 필요하다.

```text
   A   internal/transcript 의 Kind 일곱을 그대로 쓴다.  EventKind 를
       transcript.Kind 로 바꾸거나 그 값을 그대로 담는다.  어휘가 한 벌이다
   B   internal/enode 안에서 EventKind 상수를 따로 늘린다.  유닛 정의의
       「Decode 도 파서를 안 쓴다」를 글자 그대로 지킨다
   C   final 하나를 그대로 두고 줄마다 Kind=raw 로만 낸다
```

**권장은 A 다.** `CONVENTIONS.md` 1.4 가 「도구는 두 벌로 두지 않는다」로 세운
것과 같은 이유다 — 종류 이름이 두 벌이 되면 화면과 노드 로그가 다른 글자로 같은
것을 부른다. 그리고 `internal/enode -> internal/transcript` 는 이미 서 있는
임포트다 (1.6).

**B 를 고르면 지키는 것** — 유닛 정의의 문장 하나. **잃는 것** — 종류 이름의
정본이 둘이 되고, 늘어날 때마다 두 자리를 함께 고쳐야 한다.

[Answer]: A

### Question 3 — stdout 을 통째로 메모리에 드는 것을 이 유닛이 자르나

1.3 이 낸 자리다. `lastJSONObject` 가 뒤에서부터 전체를 훑으므로 꼬리만 들면
여러 줄 봉투를 못 집는다. 그런데 U3 가 최악 봉투를 **2.3 GB** 로 쟀다.

```text
   A   오늘 그대로 둔다.  cmd.Stdout = &stdout 는 이미 전체를 들고 있고
       이 유닛이 그 성질을 안 바꾼다.  잔여로 이름만 적는다
   B   상한을 걸고 꼬리만 든다 — 여러 줄 봉투가 그 순간 안 집힌다
   C   상한을 걸되 봉투 후보(마지막 '{' 부터)는 따로 들고 간다
```

**권장은 A 다.** 이 회차가 안 산 변경이고, 자르면 봉투 파서를 안 바꾼다는
FR-1 의 전제가 실질적으로 깨진다. **2.3 GB 는 노드가 아니라 Mediator 의 봉투였다**
— 노드에서는 하네스 하나의 stdout 이고 단계마다 끝난다. 그래도 값이 없는 것이
아니므로 잔여로 적는다.

[Answer]: A

### Question 4 — `transcript()` 가 업로더 갈래를 지금 파나

`component-methods.md` 3절이 `transcript()` 의 최종 모양을 넷으로 적었다 —
링만 · 업로더만 · 둘 다 · 둘 다 없음. **업로더는 U7 의 것이다** (파일 행렬 1절).

```text
   A   오늘 모양(링 하나)을 유지한다.  U7 이 갈래를 더한다.
       파일 행렬 2절의 U2 -> U7 순서 그대로다
   B   U2 가 지금 네 갈래 구조와 nil 업로더까지 판다.  U7 은 채우기만 한다
```

**권장은 A 다.** B 는 쓰는 쪽이 0 인 인터페이스를 먼저 굳히는 모양이고, U7 이
실측으로 그 모양을 뒤집으면 되돌릴 것이 는다. **A 의 대가** — U7 이 `transcript()`
를 다시 만진다. 그것은 행렬이 이미 예상한 자리다.

[Answer]: A

### Question 5 — 링에 흘리는 것의 범위

FR-2 가 「선별을 링에는 안 건다」로 정했다. 남은 물음은 stderr 다.

```text
   A   stdout 만 흘린다.  stderr 는 오늘처럼 따로 모은다
   B   stdout 과 stderr 를 둘 다 흘린다
```

**권장은 A 다.** 링의 내용이 NDJSON 이 되는데 (`requirements.md` FR-2) stderr 는
JSON 이 아니다. 섞으면 읽는 쪽이 줄마다 갈래를 하나 더 타야 하고, 그 갈래는
`raw` 로 이미 있으나 **섞는 쪽이 순서를 보장 못 한다** — 두 파이프의 인터리빙은
쓰는 시점이 정한다.

[Answer]: A

### Question 6 — tee 갈래의 실패 규율

`Ring.Write` 는 **언제나 `len(p), nil` 을 돌려준다.** 이유가 주석에 있다 —
「링 쓰기 오류로 MultiWriter 가 멈추면 하네스 실행(`cmd.Run`)까지 멈춘다」.

물음 1 = A 면 갈래가 하나 더 생긴다 (줄 배출기).

```text
   A   같은 규율.  배출기도 오류를 안 낸다.  emit 이 패닉을 내도 삼킨다
   B   배출기는 오류를 낸다.  MultiWriter 가 멈추는 것을 감수한다
```

**권장은 A 다.** 이 겹이 지는 것은 관측이고, 관측 장치가 실행을 못 죽이는 것이
이 저장소의 규율이다 (`nfr-design` §3 · `record` 의 같은 자리). **패닉까지
삼키는 이유** — emit 은 `claim.go` 가 넘긴 클로저이고 이 유닛의 코드가 아니다.

[Answer]: A

### Question 7 — 잘린 첫 줄의 왕복을 누가 재나

1.7 의 이음매다. 링이 감기면 첫 줄이 반쪽이 되고, 그것을 읽는 것은 U1 의 파서다.
**두 패키지에 걸쳐 있어 어느 한쪽의 시험도 왕복을 안 잡는다.**

```text
   A   U2 가 왕복 시험 하나를 짓는다 — 링에 512 KiB 를 넘겨 감고,
       ReadRing 으로 읽어, transcript.SplitLines · ParseLine 에 먹인다
   B   U5 (panel-live) 의 몫으로 넘긴다.  화면이 그리는 자리에서 잡는다
```

**권장은 A 다.** W-a 가 배운 것이 이것이다 — 한 벌로 못 만드는 값은 맞대는
절차가 따로 있어야 하고, **그 절차는 이음매를 만든 유닛이 진다.** B 로 넘기면
U5 가 빨갰을 때 원인이 링인지 파서인지 화면인지 셋으로 갈린다.

[Answer]: A

---

## 3. 안 묻는 것 — 이미 값이 있는 자리

```text
   링의 형식 · 크기 · 세대 · 권한   앞 팩 §6.2 · §6.3 이 값으로 닫았다
   Reset 의 시점                  단계 시작 (claim.go:455).  D7 의 「시도가 바뀌면
                                 비운다」가 그 자리를 이미 탄다 — 재시도는 새 execute 다
   Argv · ParseClaude             FR-1 이 명시로 뺐다
   노드 로그에 본문                FR-1.  log.Debug 이 종류만 적는다.  그대로 둔다
   청크 주기 2초 · 64 KiB          U7 의 것이다 (decisions.md 2절)
   사건 종류의 목록                decisions.md 2절 + U1 의 Kind 일곱.  이 유닛이 안 늘린다
```

---

## 4. 실행 단계 — 답을 받은 뒤

### 4.1 순서

```text
   ①  답 일곱을 계획 2절에 적는다
   ②  모순 검사 — 답끼리 부딪치는 자리를 센다 (AI-DLC Functional Design Step 5)
   ③  domain-entities.md      Event · EventKind · tee 갈래 · Ring 의 계약
   ④  business-logic-model.md 단계가 도는 순서 · 배출기의 자리 · 실패 경로
   ⑤  business-rules.md       불변식에 번호를 단다.  왕복 시험이 잴 것을 값 이름까지
   ⑥  W-b 교차 검사 — U4 의 답과 맞댄다 (아래 4.2)
```

### 4.2 웨이브를 닫는 자리의 교차 검사 — W-a 가 배운 것

**유닛 경계를 넘는 답의 조합은 어느 한 유닛의 모순 검사도 못 본다.** W-a 에서
같은 날 두 번 났고 두 번 다 각 유닛은 모순 0 이었다.

W-b 에서 맞댈 자리를 미리 이름으로 적는다.

```text
   사건 종류의 어휘   U2 의 물음 2 와 U4 의 as=events 응답 모양.
                    U2 가 B 를 고르고 U4 가 transcript.Event 를 그대로 내면
                    노드 로그와 API 가 다른 글자로 같은 종류를 부른다
   총 길이의 뜻      U2 는 링의 total (감긴 뒤에도 안 준다) · U4 는 진행 파일의
                    Total.  같은 이름이 다른 값이다.  둘이 한 화면에 든다
   시도(attempt)     U2 는 Reset 의 세대(gen) · U4 는 X-Enode-Log-Attempt.
                    U7 이 둘을 잇는다.  지금 갈리면 U7 이 못 잇는다
```

### 4.2.1 교차 검사 결과 (2026-09-16) — 모순 0 · 갈릴 자리 넷 · 새로 찾은 것 하나

**둘 다 각자 모순 0 이었다.** W-a 가 배운 것이 「그래도 맞대야 한다」였다.

```text
   어휘        모순 0.  U2 R10 (EventKind = transcript.Kind 별칭)과
              U4 R18 (transcript 에 json 태그) 이 같은 정본을 가리킨다.
              파일 교집합도 0 이다 — U2 는 임포트만 하고 (R14 diff 0)
              태그는 U4 가 단다

   총 길이      **같은 단어가 다른 값이다.**  U2 의 Ring.Total 은 흘린 총량이라
              감긴 뒤에도 안 준다.  U4 의 Progress.Total 은 파일 길이다.
              U5 가 한 화면에 둘 다 그린다 — 이름을 안 가르면 그 화면이 틀린다

   시도        gen 과 attempt 는 같은 수가 아니다 (U2 R13).  U4 는 attempt 가
              없거나 규칙 밖이면 400 이다 (R6.2).  이으는 것은 U7 이다

   상한의 단위   셋이 다르다 — 링 몸통 512 KiB (U2) · 진행 파일 총 길이 10 MiB (U3) ·
              한 응답 1 MiB (U4).  W-a 가 단위가 갈려 한 번 틀렸던 자리와 같은 모양이다
```

**새로 찾은 것 하나 — `Source: progress` 에 `Bytes: 0` 이 나가는 길이 셋이다.**

```text
   ①  노드가 아직 첫 청크를 안 올렸다          U3 의 R18
   ②  N2 의 쓸기가 6시간 뒤에 걷었다           U3 의 Reap
   ③  Seal 의 DropProgress 와 verdict 굳히기 사이의 창   U4 의 잔여 ③
```

**셋이 한 값으로 나간다.** 어느 유닛의 모순 검사도 이것을 못 본다 — ① 은 U3 의
규칙이고 ② 는 U3 의 NFR 이며 ③ 은 U4 의 순서라 셋이 다른 문서에 산다.
**U4 의 잔여 ③ 을 셋으로 넓혔다.**

### 4.3 산출물 셋

```text
   construction/node-stream/functional-design/domain-entities.md
   construction/node-stream/functional-design/business-logic-model.md
   construction/node-stream/functional-design/business-rules.md
```

---

## 5. 이 단계가 안 만드는 것

```text
   코드 · 시험        Code Generation
   NFR               회차 계획이 이 유닛을 스킵으로 걸었다
   인프라 설계         회차 계획 SKIP
   업로더의 안         U7
   화면               U5 · U8
```

---

## 6. 회차 밖으로 낼 것 — 지금까지 확정된 것 둘

```text
   unit-of-work.md 의 U2 절   「링 tee 도 Decode 도 파서를 안 쓴다」가 거짓이다.
                             internal/enode 는 U1 뒤로 transcript 를 임포트한다
                             (runner.go:331).  1.6 이 실측이다

   component-methods.md 3절   「io.ReadAll 을 걷고 읽으면서 emit 한다」만으로는
                             사건이 흐르는 시점이 안 바뀐다.  바뀌어야 하는 자리는
                             runner.go:196 이다.  1.1 이 실측이다
```

**답이 오면 는다.** 물음 3 의 답이 A 면 잔여 하나가 더 붙는다 — 노드가 하네스
stdout 을 통째로 메모리에 든다.
