# 스토리 생성 계획 — User Stories Part 1

`user-stories-assessment.md` 가 「실행한다 · 깊이 minimal」로 판정한 뒤의 계획이다.
**Part 2 는 이 문서의 질문 넷이 닫힌 뒤에 돈다** — 짝 팩은 Part 2 가 먼저 돌아
계획이 뒤늦게 규칙 안으로 들어갔고, 이 회차는 순서를 지킨다.

---

# 1. 산출물 계획 (체크박스)

- [ ] `user-stories/personas.md` — 페르소나. 새로 만들지 않고 짝 팩의 셋을 잇는다
- [ ] `user-stories/user-stories.md` — 스토리. **오독 경로와 침묵 경로만**
- [ ] 스토리마다 **FR 과 확인(게이트 조각 또는 새 완료 조건)** 을 붙인다
- [ ] 스토리가 낳은 새 완료 조건을 목록으로 모은다 — Units Generation 이 유닛에 내린다
- [ ] INVEST 를 확인한다 (Independent · Negotiable · Valuable · Estimable ·
      Small · Testable). 각 스토리에 수용 기준이 붙어 있는지 센다
- [ ] 페르소나를 스토리에 사상한다. 사상 안 된 페르소나가 0 임을 확인한다

# 2. 안 하는 것

```text
   행복 경로 스토리    scene-gates 의 조각 일곱이 더 촘촘하다.  약한 사본이 된다
   수용 기준 재작성    scene-gates.md 와 requirements.md 6절이 정본이다.
                      스토리는 그것을 가리킨다
   유닛 사상          유닛이 아직 없다.  Units Generation 의 몫이다.
                      스토리는 FR 을 가리키고 유닛은 그 뒤에 붙는다
   에픽 · 포인트 추정   이 저장소는 그것으로 진행을 안 잰다.  조각 게이트가 잰다
   requirements.md 수정  승인된 문서다.  4절 Q4 가 그 처리를 묻는다
```

# 3. 이 회차가 지키는 질문 형식

`common/question-format-guide.md` 는 **모든 질문의 마지막 선택지로 `Other` 를
MANDATORY** 로 요구한다. 짝 팩이 두 파일에서 어겼고 그 회차가 스스로 기록으로
남겼다 (`story-generation-plan.md` 3절).

**이 문서의 질문 넷은 전부 `Other` 를 단다.** 앞 단계의
`requirement-verification-questions.md` 도 달았다.

---

# 4. 결정이 필요한 것 — `[Answer]:` 태그

각 질문의 `[Answer]:` 뒤에 글자를 적는다.

## Q1. 페르소나를 셋으로 두나 넷으로 두나

짝 팩이 셋을 세웠다 — **P1 노드 소유자 · P2 계약 작성자 · P3 진행자.** 그
문서는 P3 를 「무엇을 쓰나: 없다. 이 팩이 만드는 표면을 안 쓴다」로 적었다.

**이 팩에서는 그것이 거짓이다.** 진행자가 현황판을 연다 — `scene-gates.md`
1절의 장면 ③ 이 그 자리다. 그리고 「함대를 보는 사람」이 `features.md` 3.6 에
이름으로 나온다.

A) **셋 그대로.** P3 진행자의 「무엇을 쓰나」를 고쳐 쓴다 — 이 팩에서는 현황판과
제어판을 둘 다 연다. 「함대를 보는 사람」은 진행자의 한 얼굴이지 다른 사람이
아니다. 페르소나가 안 늘어 짝 팩의 문서와 이름이 계속 맞는다. (권장)

B) **넷으로.** P4 「함대 관찰자」를 더한다 — 회차를 안 돌리고 게이트를 안
집행하면서 현황판만 보는 사람. 현황판이 `/ui/` 의 무인증 정적 파일이고 데이터는
토큰으로 받으므로(`static/fleet/login.mjs`) 실제로 그런 사람이 있을 수 있다.

X) Other (please describe after [Answer]: tag below)

[Answer]:

## Q2. 스토리를 무엇에 쓰나

`user-stories-assessment.md` 가 빈자리를 「오독 경로와 침묵 경로」로 짚었다.
그 판정을 그대로 쓸 것인지 묻는다.

A) **오독 경로와 침묵 경로만.** 게이트 조각이 안 재는 자리다 — 화면이 안 자랄
때 사람이 무엇을 보는가, 봉인 뒤 본문이 없는 것을 무엇으로 아는가, 상한에
닿았을 때 무엇이 적히는가, 내 기계에 원문이 남는다는 것을 아는가. (권장)

B) **행복 경로까지 전부.** CB0 ~ CB6 을 스토리로 다시 쓴다. 문서가 두꺼워지고
판정 기준이 두 벌이 된다 — 갈리면 어느 쪽이 이기는지 정할 자리가 없다.

C) **저작 경로를 포함한다.** 이 팩에도 사람이 적는 자리가 있다 — 원문 토글을
켜는 것, 폴링 간격, `curl` 의 `from` 과 `as`. 얇지만 0 은 아니다.

X) Other (please describe after [Answer]: tag below)

[Answer]:

## Q3. 스토리를 무엇으로 가르나

`user-stories.md` Step 5 가 다섯 갈래를 제시한다.

```text
   User Journey-Based   사용자 흐름을 따라간다.  장면 ① ~ ⑦ 이 이미 그 모양이다
   Feature-Based        FR-1 ~ FR-7 마다 스토리.  구현 단위와 1:1 이라 유닛에 잘 앉는다
   Persona-Based        사람마다 묶는다.  짝 팩의 선례
   Domain-Based         업무 영역으로.  이 팩은 영역이 하나라 안 갈린다
   Epic-Based           계층으로.  이 저장소는 에픽으로 진행을 안 잰다
```

A) **Persona-Based.** 짝 팩의 선례이고, 이 팩의 값이 「누가 무엇을 읽는가」라
사람으로 갈리는 것이 뜻에 맞는다. 페르소나 사상이 공짜로 끝난다. (권장)

B) **Feature-Based.** FR 마다 스토리라 Units Generation 이 유닛에 앉히기 가장
쉽다. 대신 같은 사람이 여러 FR 에 흩어진다.

C) **Persona-Based 위에 FR 을 열로 단다.** A 의 묶음에 B 의 추적을 더한다.
표가 한 열 는다. 짝 팩이 「유닛」 열로 이미 한 모양이다.

X) Other (please describe after [Answer]: tag below)

[Answer]:

## Q4. 스토리가 낳는 새 요구를 어디에 두나

오독 경로 스토리는 **요구를 낳는다.** 지금 보이는 것만 셋이다.

```text
   ①  「본문 N 개가 걷혔다」    봉인 뒤 화면이 그것을 적어야 한다.
                              requirements.md FR-4 에 이미 있다
   ②  「상한에 닿아 멈췄다」    진행 파일이 MaxBlobBytes 를 넘으면 청크가 멈춘다.
                              화면은 자라기를 멈춘다.  requirements.md 에 **없다**
   ③  「청크가 밀렸다」         업로드가 계속 실패하면 현황판이 멈춘 채로 있다.
                              그것을 「단계가 멈췄다」로 읽는다.  requirements.md 에 **없다**
```

`requirements.md` 는 방금 승인됐다. 새로 나온 요구를 어디에 둘 것인가.

A) **스토리 문서가 「새 완료 조건」으로 지고 `requirements.md` 는 안 고친다.**
Units Generation 이 그것을 유닛의 완료 조건으로 내린다. 짝 팩이 쓴 경로다.
승인된 문서를 안 건드려 재승인이 없다. (권장)

B) **`requirements.md` 에 FR 을 더하고 재승인을 받는다.** 요구의 자리가 한
곳이라 나중에 찾기 쉽다. 단계를 한 번 되돌린다.

C) **`decisions.md` 에 더할 행으로만 적는다.** `requirements.md` 10절이 이미
아홉 행을 들고 있으므로 거기 붙인다. 완료 조건으로는 안 내린다.

X) Other (please describe after [Answer]: tag below)

[Answer]:

---

# 5. 답이 닫히면 — Part 2 가 도는 순서

```text
   ①  personas.md          Q1 의 답이 셋이냐 넷이냐를 정한다
   ②  user-stories.md      Q2 가 범위를, Q3 이 묶는 법을 정한다
   ③  새 완료 조건 목록      Q4 의 답이 그것을 어디에 둘지 정한다
   ④  INVEST 와 사상 확인   사상 안 된 페르소나 0 · 수용 기준 없는 스토리 0
```

각 단계가 끝나면 1절의 체크박스를 그 자리에서 `[x]` 로 바꾼다.
