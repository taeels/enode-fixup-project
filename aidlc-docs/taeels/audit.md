# AI-DLC Audit Log — taeels (최태양)

Construction 담당 감사 로그다. 담당별 문서 루트가 `aidlc-docs/taeels/` 이므로
회차 감사 로그(`aidlc-docs/v1-run-dhseo/audit.md`)와 따로 쌓인다. 사용자 원문은
요약하지 않고 그대로 싣는다. 타임스탬프는 `date -u +%Y-%m-%dT%H:%M:%SZ`.

---

## Construction 착수 — obs 유닛
**Timestamp**: 2026-09-08T11:12:48Z
**User Input**: "w0 obs construction 시작하자"
**AI Response**: "담당 배정(construction-roster W0 obs = taeels)과 유닛 정본
(unit-of-work.md §1) · 의존 행렬 · 스토리 맵 · 팩(enode-features 3.1.1 ·
decisions 1~3절 · constraints) · 정본 ADR(065 · 069 · 060) · 기존 코드
(internal/store · internal/api · internal/runctl)를 읽고, obs 의 Functional
Design 계획을 `aidlc-docs/taeels/construction/plans/obs-functional-design-plan.md`
에 냈다. 물음 셋을 계획 안에 [Answer]: 로 박았다."
**Context**: CONSTRUCTION · obs 유닛 · Functional Design Part 1 (계획)

---

## Functional Design — obs · 물음 셋의 답
**Timestamp**: 2026-09-08T11:31:00Z
**User Input**: "여기서 차례대로 질문 해줘" (물음을 파일이 아니라 대화에서 하나씩
받겠다는 지시. AI-DLC 의 question-format-guide 가 대화 안 질문을 금지하나 사용자의
명시가 이긴다. 답은 그대로 계획 파일의 [Answer]: 에 옮겨 적었다)
**User Input (물음 1)**: "obs 가 쓰기까지" = C
**User Input (물음 2)**: "ADR-069 그대로 (attrs 중첩)" = A
**User Input (물음 3, 1차)**: "추가설명요청"
**User Input (물음 3, 2차)**: "목록만 · 언제나 (권장)" = A
**AI Response**: "답 셋을 계획 파일에 박고, 답 C 가 부른 파장을 6절에 적었다 —
contract.Advert 에 정책 필드가 없어 obs 가 internal/contract/advert.go 에 필드
하나(Draining string)를 더해야 하고, 그 파일은 어느 유닛의 파일 행렬에도 없다.
행렬 밖 파일을 만진 diff 로 가져간다."
**Context**: CONSTRUCTION · obs 유닛 · Functional Design · 답 수집 완료

---

## Functional Design — obs · 산출물
**Timestamp**: 2026-09-08T11:44:00Z
**AI Response**: "답 C·A·A 로 Functional Design 산출물 셋을 냈다 —
construction/obs/functional-design/ 의 domain-entities.md · business-logic-model.md ·
business-rules.md. 팩이 안 정한 값 하나를 여기서 정했다: GET /v1/runs 의 limit
상한 1000 이고 넘으면 자른다(거부 아님) — 페이지네이션이 없어 「전부 받았다」를
보장하는 값이 애초에 없고, 거부하면 상한이 서버와 화면 두 곳에 산다.
state 어휘는 검사하지 않는다 — queue 유닛(W1)이 QUEUED 를 더하므로 여기서
어휘를 박으면 W1 이 W0 을 다시 열어야 한다."
**Context**: CONSTRUCTION · obs 유닛 · Functional Design 완료 · 승인 대기

---

## Functional Design — obs · 하류 병렬 검토와 재작성
**Timestamp**: 2026-09-08T12:40:00Z
**User Input**: "에이전트 병렬로 풀어서 한번 더 검토해.  이 유닛이 다음 유닛들의 초석인 만큼 하류에서 모순이나 구멍을 발견하면 회귀비용이 크니 꼼꼼히 봐라"
**AI Response**: "검토를 넷으로 갈라 병렬로 돌렸다 — queue·demo-back / drain·panel·transcript /
ui·mcp / 코드·스키마·CI 정합. 스물두 자리를 고쳤고 그중 아홉은 둘 이상이 겹쳐 짚은
것이다. 가장 무거운 셋 — ① 광고 본문의 정책 필드가 정본과 다른 이름이었다
(mediator-api.md:304 · ADR-063 §6 이 policy.drain 으로 못 박았다) ② SECURITY-08
준수 판정의 근거 문장(「본문이 없다」)이 답 C 로 거짓이 됐고 decisions 3절이 이름으로
지목한 정책 값 enum 검증이 주인 없이 떨어져 있었다 ③ UpsertAdvert 가 이전 draining 을
안 돌려줘 WakeQueued 여섯 지점 중 「drain 해제를 받은 광고 처리」가 트리거를 잃었다.
검토가 새로 연 물음 둘을 사용자에게 물어 닫았다. 검토가 반증한 것 여덟도 계획 7.3 에
적었다 — 없는 문제를 지어내지 않기 위해서다."
**User Input (물음 4)**: "데모 모드에서 무인증 등록 (권장)" = A
**User Input (물음 5)**: "두 홉을 수락하고 명시 (권장)" = A
**Context**: CONSTRUCTION · obs 유닛 · Functional Design 재작성 · 승인 대기

---

## Functional Design — obs · 2차 병렬 검토와 반영
**Timestamp**: 2026-09-08T13:50:00Z
**User Input**: "작성된 fd 문서들에 대해 에이전트 풀어서 병렬 검토 해라. 하류에서 구멍이나 모순이 발견되면 비용이 커지므로 한번 더 잡고 간다"
**AI Response**: "1차(계획 §7)는 하류 담당별로 갈랐으므로 같은 축을 다시 돌리면 같은 것이
다시 나올 뿐이라 축을 바꿔 넷으로 갈랐다 — 문서 셋 내부 정합 / 정본(enode-design ·
requirements) 인용 대조 / 코드 실측과 구현 가능성 / 게이트 실증 가능성과 재개방 비용.
원시 발견 일흔둘을 겹침으로 합쳐 마흔 남짓. 실행으로 확정된 치명 둘 — ① UpsertAdvert 의
CTE 가 첫 광고에서 NULL 을 내 모든 노드의 첫 광고가 503 이고 시험 109 자리가 함께 죽는다
(에이전트 둘이 PostgreSQL 17 에 직접 쳐서 재현) ② coalesce(assigned,'[]'::jsonb) 가
jsonb null 을 안 접어 CreateQueuedRun 이 제로값을 쓰면 CP2 의 판정값이 null 로 샌다.
정본 대조에서 지어낸 인용은 0건이었고, 위험은 인용하지 않은 자리에 몰려 있었다 —
답 4=A 가 ADR-065 2절의 인증 줄을 벗어나고, 같은 ADR 이 신원을 label 로 옮겨 두어
무인증 GET /v1/nodes 가 이메일 로컬파트와 호스트명을 공개한다. 게이트 축은 CP1 의
「상태 넷」 중 넷째가 QUEUED(W1)이라 W0 에서 성립하지 않는다는 것을 찾았다."
**User Input (막힌 결정 1 · 데모 인증)**: "A 를 유지하되 어긋남을 기록한다"
**User Input (막힌 결정 2 · CP1 병합조건)**: "scene-gates §3 의 curl 두 줄 + CP0 로 못 박는다"
**AI Response**: "답 둘을 그대로 반영했다. 산출물 셋과 계획을 고쳤다 — CTE 에 coalesce,
assigned 계약 한 줄을 queue 에 넘김, observed_at 을 1행 CTE LEFT JOIN 한 문장으로 통일,
어휘 검사를 store 로 못 박음(api 여유 5.6 문장), requires 읽기 실패를 warnings 로 가름,
빈 값 규칙을 인자 넷에 통일, submit() 리터럴 셋과 CreateRejectedRun, runctl.Step.StartedAt
(panel 되열림 방지), ENODE_DEMO_MODE 이름·기본값·접점 승격, 게이트 curl 의 인증 헤더 복원,
draining 시연 절차, crypto/tls 상한과 probe.lock, 커버리지 실측(328/403 · 여유 5.6).
정본을 벗어난 자리와 진행자에게 넘기는 것 여섯을 계획 §8.5 에 적었다. 1차 고침이
계획 문서로 안 따라온 것 일곱도 함께 고쳤다."
**Context**: CONSTRUCTION · obs 유닛 · Functional Design 2차 검토 반영 · 승인 대기

---
