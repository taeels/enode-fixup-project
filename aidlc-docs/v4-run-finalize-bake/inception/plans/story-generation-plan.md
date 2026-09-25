# 스토리 생성 계획 — User Stories Part 1

`user-stories-assessment.md` 가 「실행한다 · 깊이 minimal」로 판정한 뒤의 계획이다.
**Part 2 는 4절의 질문 다섯이 닫힌 뒤에 돈다.**

---

# 1. 산출물 계획 (체크박스)

- [x] `inception/user-stories/personas.md` — 페르소나 넷. 앞 회차의 셋을 잇고 네 칸을
      이 팩에 대고 다시 썼다. P4 굽기 담당을 더했다 (Q1 = B)
- [x] `inception/user-stories/stories.md` — 스토리 열아홉. 두 갈래(Q2 = B), 사람으로 묶었다(Q3 = A)
- [x] 스토리마다 FR 열과 확인 열을 단다 — 확인 열은 조각 번호 또는 완료 조건 번호를 가리킨다.
      가리킬 것이 없는 스토리는 안 싣는다
- [x] 스토리가 낳은 새 완료 조건을 한 절에 모은다 — 열(stories.md 2절). 자리는 Q4.
      `requirements.md` 8절의 Application Design 항목과 겹치면 새로 쓰지 않고 그 번호를 가리킨다
- [x] INVEST 를 확인한다. 수용 기준(확인 열) 없는 스토리 0 (stories.md 4.1)
- [x] 페르소나를 사상한다. 스토리가 없는 페르소나 0 · 스토리가 없는 FR 0 (stories.md 4.2 · 4.3)

# 2. 안 하는 것

```text
   행복 경로 스토리      조각 0 ~ 12 가 더 촘촘하다.  약한 사본이 된다
   수용 기준 재작성      scene-gates.md 와 requirements.md 6절이 정본이다.  스토리는 가리킨다
   유닛 사상            유닛이 아직 없다.  Units Generation 의 몫이다
   에픽 · 포인트 추정     이 저장소는 그것으로 진행을 안 잰다.  조각 게이트가 잰다
   화면과 필드 설계       「누가 무엇을 알아야 하는가」만 적는다.  어디에 어떤 이름으로 싣는지는
                        Application Design 이 닫는다 (Step 11)
```

# 3. 계획이 딛는 실측

판정 문서의 표 둘이 스토리의 재료다. 질문에 쓰인 사실만 여기 다시 적는다.

```text
   ①  drain 중인 노드는 점유된 노드와 같은 편이다     internal/store/queue.go:233 ~ :239
       진행 조회에 대기 사유 칸이 없다               internal/store/observe.go:21 (StepView)
   ②  재개로 합친 lower 와 FAILED 로 봉인된 Run       ADR-077 §7.  잇는 흔적은 노드 기계 안의
                                                  .enode-metadata.json 에만 있다
   ③  노드 정책 파일에서 데몬이 읽는 것은 drain 하나    internal/enode/policy.go:22 ~ :29
       팩은 누가 굽기를 낼 수 있는지 적지 않는다      features.md · decisions.md 에 그 문장 0
   ④  명령 단계의 workspace.diff 는 사람이 리뷰한다    internal/enode/changed.go:36.  저장소 안에
                                                  그것을 기계가 적용하는 코드는 0 이다
   ⑤  checkpoint 기본 정책이 on-failure 다            decisions.md 2-10.  켜는 사람이 없어도 돈다
```

---

# 4. 결정이 필요한 것 — `[Answer]:` 태그

각 질문의 `[Answer]:` 뒤에 글자를 적는다. 모든 질문의 마지막 선택지는 `X) Other` 다.

## Q1. 굽기를 내는 사람을 따로 세나

앞 회차가 세운 셋이 있다 — **P1 노드 소유자 · P2 계약 작성자 · P3 진행자.** 굽기는
정본이 「일반 Run」이라 적었으므로(ADR-077 §2) 기제로는 P2 의 일이다. 그런데 사람으로
보면 네 칸이 다 다르다.

```text
               굽기를 내는 사람                          형제에 일반 Run 을 내는 사람
   쓰나        sync · builds[] · merge 대기 상한            effect · 예산
   보나        waiting · bake_in_progress · metadata · ir   QUEUED · finalizing · checkpoint_capture
   모르나      FAILED 인 Run 이 lower 를 바꿨는지             자기 Run 이 누구의 굽기 뒤에 서 있는지
   틀리면      재개로 합친 lower 위에 같은 굽기를 다시 낸다   Mediator 가 멈춘 줄 안다
```

**한 사람의 굽기가 다른 사람의 Run 을 기다리게 한다** — 판정 문서가 짚은 기다림의
절반이 이 둘 사이에 있다.

A) **셋 그대로.** 굽기를 내는 사람은 P2 의 한 얼굴이다. 정본의 「일반 Run 이고 새
실행 기계가 0개」와 뜻이 맞고 페르소나 문서가 앞 회차와 같은 모양으로 남는다.
대신 P2 아래 스토리에 「계약 작성자로서, 다른 계약 작성자의 굽기 때문에」가 생겨
기다리게 하는 쪽과 기다리는 쪽이 한 이름 안에 섞인다.

B) **넷으로.** P4 굽기 담당을 더한다 — 한 lower 의 신선도를 책임지고 주기 계기를
세우는 사람. P1 ~ P3 의 이름과 번호는 그대로 둔다. 기다리게 하는 쪽과 기다리는 쪽이
다른 이름이 되어 스토리가 그 갈등을 그대로 적는다. (권장)

X) Other (please describe after [Answer]: tag below)

[Answer]: B

## Q2. 스토리를 무엇에 쓰나

판정 문서가 조각이 안 재는 자리를 둘로 짚었다.

```text
   기다림의 이유    새 상태의 대부분이 「아직 안 된다」다.  기다리게 된 사람이
                   그것이 어느 기다림이고 누구 때문인지 아는가
   바뀐 기본값      계약도 설정도 안 고친 사람의 동작이 바뀐다.  그 사람이 바뀐 줄 아는가
```

A) **기다림의 이유만.** 앞 회차의 「오독 경로와 침묵 경로」와 같은 결이다. 스토리가
얇고 완료 조건이 적다.

B) **기다림의 이유와 바뀐 기본값.** 둘째는 이 팩에만 있는 빈자리다 — 앞 회차는 새
표면을 더했고 이 회차는 기존 표면의 기본값을 바꾼다. `workspace.diff` 를 리뷰하던
사람과 여유 부족에도 agent 를 받던 노드 소유자가 여기 선다. (권장)

C) **행복 경로까지 전부.** 조각 0 ~ 12 를 스토리로 다시 쓴다. 판정 기준이 두 벌이 되고
갈리면 어느 쪽이 이기는지 정할 자리가 없다.

X) Other (please describe after [Answer]: tag below)

[Answer]: B

## Q3. 스토리를 무엇으로 가르나

`user-stories.md` Step 5 가 다섯 갈래를 제시한다.

```text
   Persona-Based        사람마다 묶는다.  앞 두 회차의 선례.  이 팩의 값이 「누가 기다리나」라 맞는다
   User Journey-Based   장면 1 · 2 · 3 을 따라간다.  장면이 이미 여정이라 조각과 겹친다
   Feature-Based        FR-1 ~ FR-13 마다.  유닛에 앉히기 쉽지만 같은 사람이 흩어진다
   Domain-Based         단계 결과 · 실행 상태 · 굽기 세 갈래(features.md 1.1).  사람이 섞인다
   Epic-Based           계층으로.  이 저장소는 에픽으로 진행을 안 잰다
```

A) **Persona-Based.** FR 열과 확인 열은 묶는 법이 아니라 추적으로 단다 — Units
Generation 이 그 열로 유닛에 앉힌다. 앞 회차와 같은 모양이다. (권장)

B) **User Journey-Based.** 장면 셋을 뼈대로 쓰고 장면마다 기다리는 사람을 단다.
장면 밖의 스토리(바뀐 기본값)가 들어갈 자리가 없어 넷째 묶음을 따로 둬야 한다.

C) **Feature-Based.** FR 마다 묶는다. 한 사람이 FR 여러 곳에 흩어지고, 기다리게 하는
쪽(FR-8)과 기다리는 쪽(FR-2)이 다른 묶음에 앉는다.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

## Q4. 스토리가 낳는 새 요구를 어디에 두나

기다림의 이유 스토리는 **요구를 낳는다.** 판정 문서에서 이미 보이는 것이 셋이다.

```text
   ①  QUEUED 인 Run 이 형제의 굽기 drain 뒤에 서 있다는 것      requirements.md 에 없다
   ②  FAILED 인 굽기 Run 에서 재개로 합친 lower 에 닿는 길      requirements.md 에 없다
   ③  노드가 스스로 건 drain 과 소유자 drain 이 밖에서 갈린다     8절 ④ 가 Application Design 에 넘겼다
```

`requirements.md` 는 방금 승인됐다(2026-09-23T14:56:31Z).

A) **스토리 문서가 새 완료 조건으로 지고 `requirements.md` 는 안 고친다.**
Units Generation 이 그것을 유닛의 완료 조건으로 내린다. ③ 처럼 8절에 이미 있는 것은
그 조건이 8절 번호를 가리키고 사람 쪽 기준만 더한다. 앞 회차가 쓴 경로라 재승인이 없다. (권장)

B) **`requirements.md` 에 FR 을 더하고 재승인을 받는다.** 요구의 자리가 한 곳이라 나중에
찾기 쉽다. 승인된 단계를 한 번 되돌린다.

C) **`requirements.md` 10절의 decisions 행으로만 적는다.** 완료 조건으로는 안 내린다.
Units Generation 이 받을 것이 없다.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

## Q5. 노드 소유자가 굽기를 받을지 정하는 자리를 스토리로 쓰나

3절 ③ 이다. **overlay 노드에서 lower 를 바꾸는 Run 은 굽기 하나다.** 그런데
받을지 정할 자리가 없다 — 정책 파일에서 데몬이 읽는 것은 `drain` 하나이고, 팩은 누가
굽기를 낼 수 있는지 적지 않는다. ADR-077 §1 의 전제(profile 이 적용된 노드는 전부
overlay 노드이고 굽기는 overlay 노드가 받는다)대로면 **lower 를 둔 노드는 굽기를 받는
노드다.** prepare 와 merge 를 담은 계약이 그 노드의 광고에 맞으면 형제 모두가 서는
lower 가 바뀐다.

A) **「누가 바꿨는지 안다」만 스토리로 쓴다.** metadata 의 `bake.run` · `bake.node` 로
노드 소유자가 추적할 수 있으면 완료 조건으로 삼고, 받을지 정하는 자리는 이 팩 밖으로 둔다 —
decisions 에 순연 행을 더한다. 토큰을 가진 제출자를 믿는 오늘의 모델을 이 회차가
안 바꾼다. (권장)

B) **받을지 정하는 자리까지 스토리와 완료 조건으로 낸다.** 노드 소유자가 prepare 를 받을지
정책으로 끌 수 있어야 한다. 이 팩에 새 요구가 하나 늘고 FR-8 · FR-9 의 유닛이 그것을
진다.

C) **스토리로 안 쓴다.** 토큰을 가진 사람이 곧 믿는 사람이고 lower 는 그 신뢰 안에
있다. 추적도 metadata 가 이미 담으므로 더할 것이 없다.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

# 5. 답 — 2026-09-23T23:42:13Z

사용자 「권장대로」. 다섯 다 권장값이다 — **1 = B · 2 = B · 3 = A · 4 = A · 5 = A.**

**계획 승인으로도 읽었다.** 이 문서에서 승인할 것이 질문 다섯의 답 말고 없고, 사용자가
그것을 권장대로 닫았다. Step 8 의 답과 Step 13 의 승인이 같은 한 문장이다 — 그 읽기를
`audit.md` 에 적었다.

## 답의 모호함 분석 (Step 9 의 의무)

추가 질문 없음. 「mix of」 · 「depends」 · 「hybrid」 같은 말이 0 이고 다섯 다 선택지 라벨
하나로 닫혔다. 답끼리 부딪치는 자리도 없다 — Q1 = B 가 P4 를 세우고 Q3 = A 가 사람으로
묶으므로 기다리게 하는 쪽(P4)과 기다리는 쪽(P2)이 다른 묶음에 앉는다. Q2 = C 를 안
골랐으므로 행복 경로 스토리는 없다.

## 답이 뜻하는 것

```text
   Q1 = B   페르소나 넷.  P1 ~ P3 의 이름과 번호는 앞 회차 그대로, P4 굽기 담당을 더한다
   Q2 = B   두 갈래 — 기다림의 이유 · 바뀐 기본값
   Q3 = A   사람마다 묶고 FR 열과 확인 열을 추적으로 단다
   Q4 = A   새 요구는 stories.md 의 완료 조건으로 지고 requirements.md 는 안 고친다.
            8절에 이미 있는 항목은 번호를 가리킨다
   Q5 = A   노드 소유자 스토리는 「누가 바꿨는지 안다」까지.  받을지 정하는 자리는 순연 행으로
            stories.md 에 적고 decisions.md 에 옮기는 것은 팩을 고치는 쪽의 몫이다
```

## 표기 하나를 고쳤다 — 답 뒤, Part 2 에서

이 문서의 질문은 새 완료 조건에 앞 회차의 두 글자 약칭을 붙였다. Part 2 에서 그
약칭을 지웠다 — 푼 말이 저장소에 없고(`GLOSSARY.md` 가 없다), 그 약칭의 1 ~ 6 번이
Construction 루트 `aidlc-docs/taeels/` 에서 이미 앞 회차의 조건을 가리킨다. 짐작한
확장을 적을 수 없으므로(`CLAUDE.md`) 글자를 빼고 「완료 조건 N」으로 풀어 쓴다. 질문의
뜻과 선택지는 안 바뀌었다.

---

# 6. 답이 닫히면 — Part 2 가 도는 순서

```text
   ①  personas.md       Q1 이 셋이냐 넷이냐를 정한다
   ②  stories.md        Q2 가 범위를, Q3 이 묶는 법을, Q5 가 노드 소유자 스토리 하나의 폭을 정한다
   ③  완료 조건 목록       Q4 가 자리를 정한다
   ④  INVEST 와 사상     스토리 없는 페르소나 0 · 확인 열 없는 스토리 0 · 스토리 없는 FR 을 이름으로
```

각 단계가 끝나면 1절의 체크박스를 그 자리에서 `[x]` 로 바꾼다.
