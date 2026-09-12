# User Stories 실행 판정

`.aidlc/aidlc-rules/aws-aidlc-rule-details/inception/user-stories.md` 가 요구하는
평가서다. **이 회차는 이것을 안 쓰고 스킵했다가 되살렸다** — 아래 1절이 그 경위이고
2절이 규칙과의 실제 대조다.

---

## 1. 경위 — 절차가 거꾸로 돌았다

```text
   2026-09-11   Workflow Planning 이 User Stories 를 SKIP 으로 닫았다.
                근거는 「scene-gates 의 조각 일곱이 더 촘촘하다」였고
                그것은 규칙의 SKIP 조건 목록에 없는 근거다
   2026-09-12   설계 검증이 규칙 위반으로 판정했다
   2026-09-12   사용자가 「최소 형태로 돌린다」로 결정했다
   2026-09-12   Part 2(생성)를 먼저 돌려 personas.md · user-stories.md 를 냈다
   2026-09-12   이 문서와 story-generation-plan.md 로 Part 1 을 뒤늦게 닫는다
```

**Part 2 가 Part 1 보다 먼저 돌았다.** 그것 자체가 절차 결함이고 여기 적어 둔다 —
지우면 기록이 아니다. 유닛 분해도 스토리보다 먼저 났다 (`5cce66f` 대 `92697aa`).

---

## 2. 규칙과의 대조

`core-workflow.md` 의 User Stories 절과 `inception/user-stories.md` 의 판정 기준이다.

### ALWAYS Execute IF — 다섯이 걸린다

```text
   걸린다   New user-facing features           노드 소유자의 mcp: · 계약의 agent.mcp
   걸린다   Multiple user types or personas     셋 (노드 소유자 · 계약 작성자 · 진행자)
   걸린다   Complex business requirements       수용 기준이 조각 일곱이다
   걸린다   Customer-facing API changes         계약 문법이 는다 (라우트는 0)
   걸린다   New product capabilities            부칠 수 있는 것을 싣는 능력
```

### SKIP ONLY IF — 0 이 걸린다

여섯이 전부 「사용자 영향 0」을 전제한다. 이 회차는 사람이 손으로 적는 표면을
셋 만든다.

### 판정

**실행한다. 깊이는 minimal.** 규칙의 `Default Decision Rule`(「When in doubt,
include」)과 `Complexity Assessment Factors` 의 Risk 항목(이 회차는 스스로 위험을
High 로 적었다)이 함께 가리킨다.

**minimal 인 근거** — `scene-gates.md` 의 조각 일곱이 행복 경로의 수용 기준을
실행 명령으로 이미 적었다. 스토리가 그것을 다시 쓰면 약한 사본이 된다. 그래서
**조각 일곱이 안 재는 자리만** 짓는다 — 저작 경로와 오류 경로다.

---

## 3. 이 판정이 낳은 것

```text
   inception/user-stories/personas.md      페르소나 셋
   inception/user-stories/user-stories.md  스토리 열 (US-1 ~ US-10)
   story-map 0절                           스토리 -> 유닛 사상
```

스토리 열 중 **셋이 새 완료 조건을 낳았다** — 그것이 이 단계를 되살린 값이다.

```text
   US-2   enode.yaml 의 mcp: 를 잘못 적었을 때 노드가 안 뜨고 사유를 본다   U3
   US-4   runctl capabilities 에 mcp.<이름> 이 나온다                    U3
   US-7   agent.mcp 를 문자열로 적으면 제출에서 400 이다                  U2
```

셋 다 **스킵했으면 안 물어졌을 것**이고, 셋 다 이 회차가 고치려는 「조용히
틀린다」와 같은 모양이다.
