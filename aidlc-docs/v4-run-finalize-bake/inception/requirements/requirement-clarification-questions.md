# Requirements 재질문 — 굽기 (v4-run-finalize-bake)

질문 3 의 답이 선택이 아니라 되물음이었다 — 「병합을 안 하면 회차 진행에 문제가
되는지?」. 먼저 그 물음에 답하고, 그 답을 딛고 다시 묻는다. 나머지 다섯 질문의 답은
`requirement-verification-questions.md` 끝의 「답」 절이 분석했고 재질문이 없다.

---

# 되물음에 대한 답 — 진행은 안 막힌다. PR 세 자리에서 걸린다

**Inception 의 남은 단계와 Construction 의 코드 작업은 안 막힌다.** 회차 브랜치가
실행 환경 구현을 이미 품고 있고(`b5659ae`), 유닛 브랜치는 회차 브랜치에서 딴다
(`CONVENTIONS.md` 3.1). 모든 작업이 그 코드를 딛고 돈다.

실리는 양은 이렇다 (`origin/main...origin/unit/runtime-environment-profile`, 2026-09-23).
원 질문 3 이 「19 커밋의 코드」라고 적은 것은 부풀린 말이었다 — 19 커밋 중 코드를
싣는 것은 다섯이다.

```text
   코드 커밋 다섯         d7e21e3  실행 환경 profile 과 공통 StepRuntime 경계
                         1a52bab  runtime projection 과 in-session Harvest
                         3247b63  runc-overlay StepRuntime 을 제품 실행 경로에 연결
                         1117fcc  nested runc namespace 수명
                         8be73d2  env apply 진행 알림
   서브모듈 커밋 열넷      enode-design 포인터만 바꾼다 (ADR-073 · 075 ~ 077 정본)

   비테스트             24 파일  +3,326 -118   runc_overlay_linux.go 1,285 · internal/environment 1,400
   테스트               15 파일  +2,292 -4
   main 에 있나          없다.  origin/main 에 internal/environment 도 runc_overlay_linux.go 도 없다
```

걸리는 곳은 `main` 으로 가는 PR 이다.

```text
   1  Inception PR        3.1 은 Inception 이 닫히면 회차 브랜치를 올린다.
                          그대로 올리면 위의 코드가 함께 실린다 (질문 3 의 B).
                          안 실으려면 그 PR 을 보류해야 한다 — 3.1 의 시점을 어긴다

   2  유닛 PR             유닛 브랜치가 회차 브랜치에서 따지므로 main 과의 차이에 위의 코드가
                          함께 보인다.  첫 유닛 PR 이 그 코드를 main 에 넣는다.
                          리뷰할 diff 가 유닛의 몇 배가 되고, 그 코드의 게이트 판정은
                          그 PR 에 안 보인다

   3  서브모듈            main 의 enode-design 포인터가 enode-design main 에 없는 커밋
                          (369270a) 을 가리키게 된다.  그 브랜치가 지워지거나 다시 쓰이면
                          main 을 받는 사람이 서브모듈을 못 받는다
```

**미룰수록 느는 것 하나** — 이 회차가 `internal/enode/claim.go` 와
`runc_overlay_linux.go` 를 크게 고친다. 그 사이 `main` 이 같은 파일을 만지면 어느 PR
에서든 충돌이 커진다. 이 비용은 어느 길을 골라도 PR 시점에 한 번 치르고, 미루는 만큼
커진다.

**그래서 이 선택이 Requirements 의 내용을 바꾸지 않는다.** 요구 · 설계 · 유닛 분해는
어느 길이든 같다. 달라지는 것은 Inception PR 의 모양과 시점이다.

---

## Clarification Question 1 — 실행 환경 구현이 `main` 에 가는 길 (원 질문 3)

A) **실행 환경 브랜치를 먼저 자기 PR 로 `main` 에 올린다** — enode-design 의 같은 이름
브랜치도 enode-design `main` 으로 먼저 간다. 그 뒤 회차 브랜치가 `main` 을 합치고
Inception PR 에는 이 회차의 문서만 실린다. 셋 다 풀린다. 대가는 그 브랜치의 PR 이 이
회차보다 먼저 서야 한다는 것이다 — 그 브랜치 자신의 게이트 판정이 선행 조건이다.

B) **회차의 Inception PR 에 실어 보낸다** — 한 번에 끝난다. 1 과 3 이 그 PR 한 번에
몰리고, 2 는 사라진다(유닛 PR 은 그 뒤 main 을 딛는다).

C) **Inception PR 직전까지 미룬다** — 지금 고르지 않는다. Workflow Planning 이 이
질문을 실행 계획의 미결로 들고 가고, Units Generation 승인 때(Inception PR 직전) A 나
B 로 닫는다. 그때까지 진행은 안 막힌다. 그 사이 그 브랜치의 게이트 판정이 나오면 A 를
고를 재료가 선다. (권장 — 이 선택이 요구를 안 바꾸고, A 의 선행 조건이 이 회차 밖에
있다)

X) Other (please describe after [Answer]: tag below)

[Answer]: A — 채팅 답 2026-09-23T14:41:53Z 「그럼 먼저 main에 기준선을 병합하고 진행하도록 하지.」
