# U2 `node-stream` — 불변식

**번호를 단다.** Code Generation 이 이 번호로 시험을 걸고, 계획의 체크박스가
이 번호를 가리킨다.

---

## 1. 배선

```text
   R1   cmd.Stdout 의 갈래는 셋이다 — &stdout · j.Transcript · 배출기.
        j.Transcript 가 nil 이면 그 갈래만 뺀다.  배출기는 언제나 있다
   R2   갈래의 순서는 &stdout · 링 · 배출기다.  앞의 둘이 싼 쪽이고,
        느린 쪽을 앞에 두면 화면이 늦게 본다
   R3   Decode 가 ParseClaude 에 넘기는 것은 stdout 전부다.  꼬리가 아니다 —
        lastJSONObject 가 뒤에서부터 전체를 훑는다 (harness.go:172)
```

## 2. 배출

```text
   R4   한 단계의 사건은 정확히 한 번 난다.  내는 쪽은 배출기다.
        runner 가 Decode 에 넘기는 emit 은 no-op 이다
   R5   배출기가 내는 Event 의 Text 는 언제나 빈 문자열이다.
        본문은 담기지 않으므로 찍힐 수 없다 (FR-1 · SECURITY-03)
   R6   배출기의 Write 는 언제나 (len(p), nil) 이다.  emit 이 패닉을 내도
        Write 는 안 깨진다.  링과 같은 규율이다 (transcript.go:103)
   R7   채널이 차면 줄을 버리고 센다.  Close 가 그 수를 로그로 낸다.
        버린 수가 0 이 아니어도 단계는 성공이다
   R8   Close 는 미완성 꼬리를 마지막 줄로 밀고 채널을 닫고 고루틴을 기다린다.
        Close 뒤에 나는 사건이 0 이다
   R9   Close 는 Decode 앞이다.  단계가 끝난 뒤에 사건이 나는 창이 0 이다
```

## 3. 어휘

```text
   R10  EventKind 는 transcript.Kind 의 별칭이다.  종류 이름의 정본이 하나다
   R11  EventFinal 의 값 "final" 은 transcript 의 일곱 중 어느 것과도 안 같다.
        시험이 그것을 고정한다 — 겹치면 화면이 봉투를 하네스 사건으로 읽는다
```

## 4. 이음매 — 한 패키지 시험이 못 잡는 자리

```text
   R12  링이 감긴 뒤의 왕복이 선다.
        링에 capacity 를 넘겨 쓰고 -> ReadRing -> transcript.Parse 에 먹인다.
        truncated 는 Snapshot.Total > len(Snapshot.Data) 로 유도한다.
        Result.Head 가 0 보다 크고 Result.Events 가 0 보다 크다
   R13  U7 이 Mediator 에 실을 값은 attempt 이지 Ring.gen 이 아니다.
        gen 은 그 노드가 시작한 단계마다 오르고 attempt 는 한 단계의 재시도다
```

**R12 가 이 유닛이 지는 이음매다.** 짓는 쪽이 링(U2)이고 읽는 쪽이 파서(U1)라
어느 한 패키지의 시험도 왕복을 안 잡는다. W-a 가 `enode.capped` 에서 같은 모양을
한 번 겪었고, **거기서 배운 것이 「이음매를 만든 유닛이 잰다」였다.**

---

## 5. 범위

```text
   R14  이 유닛은 internal/transcript 를 임포트만 한다.  그 패키지의 diff 가 0 이다
   R15  이 유닛은 claim.go 를 안 만진다.  Transcript 도 Emit 도 이미 넘어온다
   R16  링에 흘리는 것은 stdout 뿐이다.  stderr 는 안 섞는다 — 링의 내용이
        NDJSON 이고 stderr 는 JSON 이 아니며, 두 파이프의 순서를 쓰는 쪽이 정한다
   R17  라우트가 안 는다.  grep -c 'mux.HandleFunc' internal/api/api.go 가 17 그대로다
```

---

## 6. 잔여 — 이 유닛이 안 닫는 것

```text
   ①  노드가 하네스 stdout 을 통째로 메모리에 든다 (답 3 = A).
       오늘도 그렇다 (cmd.Stdout = &stdout) — 이 유닛이 그 성질을 안 바꾼다.
       자르면 여러 줄 봉투를 못 집어 FR-1 의 「봉투 파서를 안 바꾼다」가 깨진다

   ②  배출기가 버린 줄은 노드 디버그 로그에서 사라진다.  링은 안 다치므로
       화면은 온전하다.  버린 수는 남는다 (R7)

   ③  배출기가 줄마다 transcript.Parse 를 부른다 — 붙이기(attachNames)가
       줄을 넘어가지 못한다.  나르는 것이 종류뿐이라 오늘은 값이 0 이다.
       emit 에 이름이 필요한 소비자가 생기면 그때 다시 연다
```

**셋 다 이름으로 적었다.** 안 적으면 다음 사람이 빠뜨린 것으로 읽는다.

---

## 7. 회차 밖으로 낼 것

```text
   unit-of-work.md U2 절        「Decode 도 파서를 안 쓴다」가 거짓이다.
                               R10 · R14 가 그 자리를 명시로 뒤집는다

   component-methods.md 3절     「io.ReadAll 을 걷고 읽으면서 emit 한다」만으로는
                               사건이 흐르는 시점이 안 바뀐다 (runner.go:209).
                               바뀌는 자리는 cmd.Stdout 이다

   component-methods.md 3절     transcript() 의 네 갈래는 U7 의 것이다 (답 4 = A).
                               이 유닛 뒤에도 링 하나를 그대로 낸다

   unit-of-work-file-matrix.md  claim.go 의 U2 칸이 빈다.  transcript() 를 U7 이 만진다
```
