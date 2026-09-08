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
## Functional Design — obs · 승인
**Timestamp**: 2026-09-08T14:15:57Z
**User Input**: "nfr requirements 시작"
**AI Response**: "다음 단계를 지시한 것을 Functional Design 의 Continue to Next Stage
로 읽었다. 산출물 셋(domain-entities · business-logic-model · business-rules)과
계획을 승인으로 닫고 상태 파일에 적었다."
**Context**: CONSTRUCTION · obs 유닛 · Functional Design 승인

---

## NFR Requirements — obs · 계획 (Part 1)
**Timestamp**: 2026-09-08T14:15:57Z
**User Input**: "nfr requirements 시작"
**AI Response**: "이 단계는 회차 실행 계획에서 SKIP 이었다 — 근거는 「팩이 값으로
닫았다」였다. 사용자 지시로 다시 들였고 그 사실을 계획 0절에 적었다. 담당 상태
파일이 이미 회차 계획과 어긋나 있었다는 것도 함께 적었다 — taeels/aidlc-state.md
는 NFR Requirements 와 NFR Design 을 대기로, 회차 계획은 둘 다 SKIP 으로 적었다.

옮겨 적기가 아닌 자리 셋을 찾아 그대로 물음 셋으로 냈다. ① SECURITY-11 의 요청
한도 — 팩 3절이 처리를 미리 적은 SECURITY 규칙은 다섯(01·03·07·08·12)이고 11 이
그 목록에 없는데, Functional Design 의 답 4=A 로 읽기 셋이 무인증 공개 표면이
되면서 차단성 확장의 그 규칙이 처음 걸린다. ② GET /v1/runs 의 정렬 인덱스 —
runs 표의 인덱스가 runs_work_idx 하나이고 ?work= 없는 목록이 그것을 못 탄다.
Functional Design 이 「안 붙인다」의 근거로 든 mediator-api.md 문장은 API 표면을
그은 것이지 DB 인덱스를 금한 것이 아니다. ③ 캐시 헤더의 주인 — business-rules 2.4
가 헤더로 한다고 적고 누가 붙이는지를 안 정했다.

착수 전에 실측한 것 열둘을 계획 1절에 적었다 — 요청 한도 코드 0건, CORS 없음이자
동일 출처라 필요도 없음(화면과 읽기 셋이 같은 프로세스·같은 포트), ReadHeaderTimeout
10초만 걸린 http.Server, 옵션 없는 pgxpool, 접근 로그 없음, leases.node_id 가
PRIMARY KEY 라 LEFT JOIN 이 행을 안 불림, go.mod 에 golang.org/x/time 없음.

물음에 안 올린 권장값 다섯도 근거와 함께 적었다 — 동시 시청자 30 내외 · 서버 재시도
없음 · 접근 로그 없음 · 가용성 수치 목표 없음 · 응답 본문 상한 없음. 사용자 규칙
「단계마다 셋까지」를 지켜 물음을 셋으로 묶었다.

계획을 aidlc-docs/taeels/construction/plans/obs-nfr-requirements-plan.md 에 냈고
emphasis-check 와 glyphscan 이 둘 다 통과했다."
**Context**: CONSTRUCTION · obs 유닛 · NFR Requirements Part 1 (계획) · 답 대기

---
## NFR Requirements — obs · 물음 셋의 답과 산출물
**Timestamp**: 2026-09-08T14:15:57Z
**User Input**: "B, B, A"
**User Input (물음 1)**: B — obs 가 무인증 읽기 셋에 요청 한도를 붙인다
**User Input (물음 2)**: B — obs 가 runs_created_idx 를 schema.sql 에 더한다
**User Input (물음 3)**: A — obs 가 Cache-Control: no-store 를 싣는다
**AI Response**: "셋 다 obs 안으로 들어왔다. 답에 모순이나 모호가 없어 확인 물음
파일을 안 냈다. 산출물 둘을 construction/obs/nfr-requirements/ 에 냈다 —
nfr-requirements.md · tech-stack-decisions.md.

답 1=B 의 값을 이 단계가 정했다. NFR Design 이 SKIP 이라 값이 여기 없으면
Code Generation 이 짐작하기 때문이다. 전역 한도 하나 · 초당 120 · 버스트 240 ·
넘으면 429 + Retry-After: 1. 유도는 시청자 30 이 5초마다 다섯을 부르는 것
(nodes 1 + runs 1 + 두 홉 getRun 3) = 30 rps 정상이고 한도는 그 4배다.
IP 로 안 나눈다 — 대회장 관객이 한 NAT 뒤에 있어 출발지로 나누면 관객 전체가
한 클라이언트로 묶여 정상 폴링이 막히고 그것이 CP10 을 직접 깬다. 그 대신 전역
한도의 대가(한 클라이언트가 나머지를 굶길 수 있다)를 3.2 에 적고 수락했다.
직접 짠다 — golang.org/x/time/rate 가 go.mod 에 없고 팩이 새 의존 0 을 못 박았다.

답 2=B 는 Go 문장이 0 이고 schema.sql 한 줄이다. btree 가 역방향으로도 훑으므로
created_at DESC 정렬이 그대로 인덱스를 탄다. ?since= 도 같은 인덱스를 탄다.

답 3=A 는 읽기 셋 셋 전부다 — getRun 도 두 홉으로 폴링에 들어간다. 응답 헤더가
늘 뿐 본문의 글자는 안 바뀌므로 mcp 의 글자 일치(CP5)와 부딪치지 않는다.

SECURITY 열다섯을 obs 표면에 하나씩 대고 판정했다. 팩 3절이 처리를 미리 적은
것은 다섯뿐이라 나머지 열의 판정을 새로 냈다. 조건부가 둘이고(SECURITY-08 ·
SECURITY-12) 같은 하나의 사실에서 나온다 — 데모 읽기 셋의 무인증이 ADR-065
2절을 벗어난다. 둘 다 준수로 칠하지 않았다.

실측으로 확인한 것 둘을 근거로 썼다 — go list -deps ./cmd/enodectl 에
internal/api 가 없어서 한도 파일이 심볼 상한 둘을 못 움직인다(0건) ·
leases.node_id 가 PRIMARY KEY 라 GET /v1/nodes 의 LEFT JOIN 이 행을 안 불린다.

정본을 넓히는 자리 하나를 찾았다 — 429 가 mediator-api.md 1.2 의 코드 표
여덟(400·401·404·409·410·413·422·503)에 없다. Functional Design 이 이미 넷을
표시로 남겼고 이것이 다섯째다.

emphasis-check 와 glyphscan 통과. 계획의 체크박스 여덟을 전부 닫았다."
**Context**: CONSTRUCTION · obs 유닛 · NFR Requirements 완료 · 승인 대기

---
