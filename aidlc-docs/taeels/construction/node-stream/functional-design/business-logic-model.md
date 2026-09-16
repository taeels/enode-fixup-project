# U2 `node-stream` — 순서

**이 유닛이 사는 값은 순서다.** 무엇이 있는가는 `domain-entities.md` ·
불변식은 `business-rules.md`.

---

## 1. 모순 검사가 막은 것 하나 — 사건이 두 번 날 뻔했다

**답 1 = A 와 FR-1 을 그대로 합치면 배출이 두 벌이다.**

```text
   배출기    도는 중에 줄마다 emit 을 부른다            답 1 = A
   Decode   끝난 뒤에 줄 단위로 읽으며 emit 을 부른다    FR-1 의 글자
```

두 자리가 같은 `j.Emit` 을 받으면 **한 단계의 사건이 정확히 두 번 난다.** 오늘은
소비자가 `log.Debug` 하나라 로그가 두 줄이 되는 것으로 끝나지만, U7 이 이 자리에
업로더를 물리면 **같은 바이트가 두 번 올라간다.**

**가르는 것은 취향이 아니다** — 흐르는 것은 도는 중에만 값이 있고, 끝난 뒤의
배출은 아무도 안 기다린다.

```text
   답    사건은 흘리는 쪽이 낸다.  runner 가 Decode 에 넘기는 emit 은 no-op 이다
        Decode 의 emit 경로는 살아 있다 — 시험과 다른 호출자가 쓴다 (R4 · R11)
```

**`Decode` 의 시그니처는 안 바뀐다.** 바뀌는 것은 runner 가 무엇을 넘기느냐뿐이다.

---

## 2. 단계가 도는 순서

```text
   ①  execute 가 링을 Reset 한다                     claim.go:455.  오늘 그대로
   ②  runHarness 가 배출기를 만든다                   emit 은 j.Emit
   ③  cmd.Stdout 에 갈래 셋을 건다                    &stdout · 링 · 배출기
   ④  cmd.Run                                       자식이 돈다
        │  쓰기마다   버퍼에 담기고 · 링에 감기고 · 배출기가 줄을 채널에 민다
        │  고루틴     채널에서 줄을 꺼내 transcript.Parse 로 읽고 emit 을 부른다
        │  화면       제어판이 링을 1초로 읽는다 (앞 팩 그대로)
   ⑤  배출기를 Close 한다                            채널을 닫고 고루틴을 기다린다
        버린 줄이 있으면 그 수를 로그로 낸다 (R7)
   ⑥  Decode(bytes.NewReader(stdout.Bytes()), code, noop)
        줄 단위로 읽고 · 봉투를 ParseClaude 로 읽고 · final 하나를 낸다
        그 final 은 no-op 으로 간다 (R4)
   ⑦  자백 · 버전 · logs/ 선별                        오늘 그대로
```

**⑤ 가 ⑥ 앞이다.** 뒤에 두면 고루틴이 아직 줄을 들고 있는 채로 `Decode` 가 돌고,
단계가 끝난 뒤에 사건이 나는 창이 생긴다. **그 창은 U5 가 「끝났는데 아직 흐른다」
로 그리는 자리다.**

---

## 3. 배출기 안의 순서

```text
   Write(p)      ①  p 를 이어 붙인다 (미완성 꼬리를 든다)
                 ②  개행마다 잘라 줄을 만든다
                 ③  줄을 채널에 민다.  차면 버리고 dropped++
                 ④  언제나 len(p), nil 을 돌려준다            R6

   고루틴        ①  채널에서 줄 하나를 꺼낸다
                 ②  transcript.Parse(line + "\n", false) 를 부른다
                 ③  나온 사건마다 Event{Kind: e.Kind} 로 emit 한다   R5
                 ④  채널이 닫히면 돌아간다

   Close()       ①  미완성 꼬리가 있으면 마지막 줄로 민다      R8
                 ②  채널을 닫는다
                 ③  고루틴이 끝나기를 기다린다
```

**② 가 `Parse` 인 이유** — 줄 하나를 사건으로 읽는 규칙이 이미 거기 있다.
배출기가 자기 사상표를 지으면 **같은 줄을 화면과 노드가 다른 종류로 부른다.**

**한 줄이 사건 여럿이 될 수 있다** — `assistant` 한 줄에 thinking 과 text 와
tool_use 가 함께 오면 사건 셋이다 (`transcript.go` 의 `Event` 주석). 배출기는
그 셋을 다 낸다.

**붙이기(`attachNames`)는 줄을 넘어간다** — `tool_result` 가 `tool_use` 의 이름을
줄 경계 너머에서 얻는다. 줄마다 부르면 그 붙이기가 안 산다. **배출기는 그것을
안 쓴다** — 나르는 것이 종류뿐이라 이름이 필요 없다 (R5). 이름이 필요한 자리는
화면이고, 화면은 뭉치를 한 번에 읽는다.

---

## 4. `Decode` 안의 순서

```text
   오늘         ①  io.ReadAll
               ②  ParseClaude(b, exitCode)
               ③  emit(Event{Kind: final, Text: h.Message})

   이 유닛 뒤    ①  줄 단위로 읽으며 바이트를 이어 담는다 (답 3 = A)
               ②  줄마다 transcript.Parse 로 읽고 사건마다 emit
               ③  다 읽은 바이트로 ParseClaude(b, exitCode)     R3
               ④  emit(Event{Kind: final, Text: h.Message})
```

**③ 이 전체 바이트를 받는 것이 R3 이다.** `lastJSONObject` 가 뒤에서부터 전체를
훑는다 — 봉투가 여러 줄로 예쁘게 찍혀 올 수 있기 때문이다 (`harness.go:172` 의
주석). **꼬리만 들면 그 경우를 못 집는다.**

그래서 **`io.ReadAll` 을 걷는다는 것이 「덜 든다」는 뜻이 아니다.** 드는 양은
같고, 읽으면서 배출한다는 것이 바뀐다.

---

## 5. 실패 경로 — 무엇이 실행을 죽이나

```text
   링 쓰기 실패        삼킨다.  오늘 그대로 (transcript.go:103 의 주석)
   배출기 채널 참      버린다.  센다.  Close 가 로그로 낸다              R7
   emit 이 패닉        삼킨다.  emit 은 이 유닛의 코드가 아니다          R6
   고루틴이 안 끝남     Close 가 기다린다.  채널이 닫히면 반드시 끝난다
   stdout 버퍼 OOM     오늘 그대로.  잔여로 적는다 (business-rules §6)
```

**죽이는 것이 0 이다.** 이 유닛이 더하는 겹은 전부 관측이고, 관측이 실행을 죽이면
그것은 관측이 아니다.

---

## 6. 재시도가 갈리는 순서

```text
   ①  Mediator 가 같은 단계를 다시 준다        attempt 가 오른다
   ②  execute 가 다시 돈다                    claim.go:440
   ③  ring.Reset() — total 0 · gen++          claim.go:455
   ④  화면이 gen 이 바뀐 것을 보고 카드를 비운다   앞 팩 그대로
```

**`gen` 과 `attempt` 는 같은 수가 아니다.** `gen` 은 그 노드가 시작한 단계마다
오르고 `attempt` 는 한 단계의 재시도 횟수다. **U7 이 Mediator 에 실어 보낼 값은
`attempt` 이지 `gen` 이 아니다** — 이음매로 적는다 (R13).
