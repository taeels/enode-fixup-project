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
## Code Generation (Part 1 계획 · Part 2 착수)
**Timestamp**: 2026-09-08T15:12:00Z
**User Input**: "이제 코드 작성해라. 병렬로 할 수 있는 것들은 병렬로 해라."
**AI Response**: "이 지시가 둘을 겸한다 — 앞 단계(NFR Requirements)의 승인과
Code Generation 착수다. NFR Requirements 는 산출물 둘을 낸 채 승인을 기다리고
있었고, 다음 단계로 가라는 지시가 곧 그 승인이다. 상태 파일에 그렇게 적는다.

Part 1 계획을 construction/plans/obs-code-generation-plan.md 에 냈다. 이 계획이
NFR Design 의 자리를 지고 간다 — 회차 계획이 SKIP 이고 nfr-requirements 6절
표시 5 가 「한도의 자료구조와 미들웨어를 어디에 거는가」를 이 단계에 넘겼다.
값은 nfr-requirements 3.1 이 다 적었으므로 계획 5절이 어떻게만 정한다.

병렬 지시를 파일 축으로 갈랐다 — 단계로 가르면 같은 파일을 둘이 열어 병렬이
아니라 충돌이 된다. 세 갈래가 파일을 하나도 안 겹친다.
  A  internal/store + internal/contract
  B  internal/api + internal/config
  C  internal/runctl
A 와 B 사이에만 의존이 있고(B 가 A 의 시그니처를 부른다) 그 시그니처는
domain-entities 8절이 이미 못 박아 두었으므로 계획 3절에 옮겨 고정하고 셋이
동시에 그 계약 위에서 짰다. C 는 json.RawMessage 라 어느 쪽도 안 딛는다.

계획이 정한 것 셋이 Functional Design 에 없던 자리다.
  DrainPolicy 를 내보낸다   UpsertAdvert 의 반환은 이전 값이고 응답의 drain 은
                           저장된 값이라 둘이 다르다. 시그니처를 안 넓히고 같은
                           값을 얻는 길이 접는 함수를 순수하게 내보내는 것이다.
                           응답이 저장된 값과 언제나 같다는 것을 같은 함수를
                           쓰는 것으로 보장한다
  RequireView 를 store 에   internal/api 의 커버리지 여유가 5.6 문장이고 store 는
                           31.2 다. 정책 어휘 검사를 store 에 둔 것과 같은 수 ·
                           같은 논거다
  조건부 등록이 아니라 래퍼  라우트 표가 모드와 무관하게 같은 줄이고 감싸는 것만
                           갈린다. CP0 의 「기존 열다섯 라우트가 그대로」가 데모
                           모드에서도 참이 되고 ServeMux 중복 등록 패닉도 없다

시험용 Postgres 를 scripts/testdb.sh 로 올렸다 (포트 55434 · enode_test)."
**Context**: CONSTRUCTION · obs 유닛 · Code Generation Part 1 완료 · Part 2 병렬 착수

---
## Code Generation (Part 2 · 실행과 검증)
**Timestamp**: 2026-09-09T00:20:00Z
**User Input**: (앞 항목의 지시로 계속 — 새 입력 없음)
**AI Response**: "세 갈래를 병렬로 짓고 합쳐서 쟀다. 파일이 하나도 안 겹쳤고
합칠 때 충돌이 0 이었다 — 가르는 축을 단계가 아니라 파일로 잡은 값이다.

CP0 이 전부 초록이다. 16개 패키지가 다 통과했고 커버리지 미달이 0 이다.
병목이던 internal/api 가 81.4% 에서 82.9% 로 올라 여유가 5.6 에서 13.6 문장이
됐다 — 새 문장 70 중 안 덮인 것이 여섯이고 전부 503 query failed 분기다.
전체는 5600/6396 = 87.6%. 심볼 상한 둘은 안 움직였다 (net/http 6 · crypto/tls 1).
스킵이 0 이고 glyphscan 과 gofmt 가 깨끗하다. probe.lock 은 커버리지 실행이
바꾼 것을 되돌렸다.

CP1 을 실 Mediator 로 쟀다 — 전용 DB 를 파고 합성 함대 넷을 세웠다.
scene-gates 3절의 curl 두 줄이 둘 다 초록이다. 노드 넷이 나오고, 임대된 노드에
lease 가 있고, draining 노드가 graceful 을 내고, Run 상태 셋이 최신순으로 나온다.
submitter 키가 모든 행에 있고 거절된 Run 의 assigned 가 [] 다 — 앞은 CP9 지분이고
뒤는 CP2 재료다.

재다 한 번 잘못 읽었고 그것을 산출물에 적었다. not_after 의 최초 발급은
TTLSeconds(3600)이고 하트비트 갱신은 RenewSeconds x NotAfterFactor(180)라 공식이
다르다. 둘을 비교하면 시각이 앞당겨진 것처럼 보인다. 갱신끼리 비교하니 하트비트마다
+3.0초로 앞으로 갔다. 기존 동작이고 이 유닛이 바꾼 것이 아니다 — 게이트를 집행하는
사람이 같은 자리를 밟을 수 있어 code-summary 3.2 에 적었다.

NFR 의 재는 자리 셋도 다 쟀다. 데모 인스턴스에 400 요청을 던져 200 이 263 ·
429 가 137 이고 429 마다 Retry-After: 1 이 실렸다. 같은 400 을 실 함대 인스턴스에
던지니 429 가 0 이다 — 「무인증 경로에만」이 그 둘째 줄로 섰다. runs 5만 행에
EXPLAIN 을 걸어 ?work= 없는 목록이 Index Scan Backward using runs_created_idx 를
타는 것을 봤다 — Seq Scan 도 Sort 도 없다. ?work= 는 기존 runs_work_idx 그대로다.
Cache-Control: no-store 가 읽기 셋 셋 다에 실린다.

SECURITY-03 을 실행으로 확인했다 — 어휘 밖 정책을 접을 때 로그가 node 만 찍고
원문의 등장 횟수가 0 이다. 데모 스위치도 확인했다: ENODE_DEMO_MODE=1 에서 읽기
셋 셋이 무인증 200 이고 POST /v1/runs · POST /v1/nodes · GET /v1/asks 는 401
그대로이며, 기본값에서는 읽기 셋도 401 이라 실 함대가 무변경이다.

CP1 이 통째로 초록인 것은 아니다. 화면 다섯이 ui 몫이고 넷째 상태 QUEUED 는
queue(W1)가 만든다. 그리고 위 측정은 합성 함대라 실 데몬 함대의 재측정은
게이트 집행자의 몫이다 — 그 한계를 code-summary 3.2 가 적는다."
**Context**: CONSTRUCTION · obs 유닛 · Code Generation 완료 · CP0 초록 · CP1 부분 측정

---

## CONSTRUCTION — U1 isolation · Functional Design 착수
**Timestamp**: 2026-09-12T11:55:49Z
**User Input**: "이제 functional design"
**AI Response**: "v3-run-harness-components 회차의 CONSTRUCTION 을 연다. 첫 유닛은 U1 isolation 이고 선행이 없다 (unit-of-work.md 1절 · 회차 aidlc-state.md 의 Next Stage). 브랜치 unit/isolation 을 회차 브랜치에서 땄다 (CONVENTIONS 3.1). functional-design.md 의 Step 1 ~ Step 4 를 돌려 계획과 물음 다섯을 aidlc-docs/taeels/construction/plans/isolation-functional-design-plan.md 에 냈다.

계획을 짓기 전에 decisions.md 6절 ⑱ 이 U1 에 남긴 숙제를 먼저 쟀다 — stream-json 의 사건 종류다. claude 2.1.266 에 -p --output-format stream-json --verbose 로 두 줄기(도구 성공 · 도구 실패)를 돌렸다. 본 것은 다섯이다 — system(init · hook_started · hook_response · thinking_tokens) · assistant · user · rate_limit_event · result. 원문은 저장소에 안 싣는다 (개인 홈의 서버 이름과 절대경로가 들어 있다).

실측이 문서의 전제 하나를 뒤집었다. ⑱ 과 unit-of-work.md 1절이 「init 이 나오는 경로에서 첫 줄은 언제나 system/init 이다 — 게이트가 head -1 로 읽는다」로 못 박았는데, 실측에서 첫 줄은 system/hook_started 였고 init 은 아홉째 줄이었다. SessionStart 훅 넷이 앞에 사건 여덟을 냈다. 가짜 홈 아래서는 --setting-sources '' 가 개인 설정을 끊고 우리 settings.json 에 Stop 훅 하나뿐이라 안 날 가능성이 높지만, 그것은 하네스의 보증이 아니라 우리 설정 파일의 내용에 기댄 전제다. 질문 3 이 그 전제를 어디에 둘지 묻는다.

실측이 굳힌 것도 셋이다 — init 의 mcp_servers 는 맵이 아니라 [{name,status}] 배열이라 CA1 의 「비어 있다」는 [] 다. 도구 이름은 assistant 의 message.content[] 중 tool_use 의 name 이고 성공 여부는 user 의 tool_result 의 is_error 인데 성공하면 그 키가 아예 없다. 그리고 본문이 content 블록 밖에도 있다 — assistant 의 wire_tool_inputs 와 user 의 tool_use_result 가 도구 입력과 결과를 최상위에 한 번 더 싣는다. 블록만 지우는 구현은 안 닫힌다는 뜻이고 ⑱ 이 허용목록을 고른 것이 옳았다는 증거다.

물음 다섯 — ① 껍데기 한 줄의 표현(새 JSON 객체 · 텍스트 · 원본에서 지우기) ② 껍데기가 담는 범위(셋만 · 블록 종류 추가 · 토큰 수 추가) ③ 「첫 줄은 init」 전제를 어디에 두나(게이트가 순서에 안 기댄다 · 불변식 유지 · runner 가 순서를 만든다) ④ 걷었음을 표시하는 줄의 자리 ⑤ U1 이 mcp.json 직렬화를 어디까지 짓나. 다섯 다 Other 선택지를 든다."
**Context**: CONSTRUCTION · U1 isolation · Functional Design Step 1 ~ 4 완료 · 답 대기

---

## CONSTRUCTION — U1 isolation · Functional Design 산출
**Timestamp**: 2026-09-12T12:15:13Z
**User Input**: "답했음."
**AI Response**: "계획 파일의 [Answer] 다섯을 읽었다 — 1=A · 2=C · 3=B · 4=A · 5=A. 모호한 답이 0 이라 추가 질문 없이 Step 6 으로 갔다. 산출물 셋을 construction/isolation/functional-design/ 에 냈다.

답 2=C 를 짓기 전에 usage 의 실제 모양을 다시 쟀다. usage 안에 숫자가 아닌 것이 있다 — service_tier 와 inference_geo 는 문자열이고 cache_creation 은 객체다. 그래서 usage 를 통째로 못 옮긴다. 이름으로 넷만 집는다 (input_tokens · output_tokens · cache_creation_input_tokens · cache_read_input_tokens). system/thinking_tokens 의 estimated_tokens 하나를 더 실었고 그것이 답 2 의 글자 밖이라 domain-entities 5.3 에 이름으로 적었다.

답 3=B 를 받아 「첫 줄은 init」을 불변식으로 유지했다. 그 불변식이 서는 근거 셋과 시험 둘을 business-rules 6절에 적었고, 잔여 위험을 6.3 에 이름으로 남겼다 — 근거 둘째(하네스가 init 앞에 우리 설정과 무관한 사건을 안 낸다)는 우리가 못 재고 하네스의 계약도 아니다. 깨지면 게이트의 head -1 이 mcp_servers 키가 없는 줄을 읽어 「비어 있다」로 오독될 수 있다. 답 3=A 가 그 위험을 없앴을 길이고 사용자가 B 를 골랐다.

답 5=A 를 받아 직렬화를 U1 이 전부 짓는다. 그러면서 셋을 정했다 — Credential 은 mcp.json 에 안 나간다 (하네스가 모르는 키이고 갈 이유가 0 이며 ADR-035 §8 이 같은 값을 이미 골랐다. 사는 자리는 U3 의 mcpUp 이다) · 빈 파일은 {} 가 아니라 {\"mcpServers\":{}} 다 (키 없는 파일을 하네스가 어떻게 읽는지 실측이 없다) · Env 의 표현과 미지 키 통과는 U3 의 것이고 U1 은 운반만 지고 이음매를 이름으로 적었다.

설계가 답 밖에서 더 정한 것 둘. 플래그를 산출물마다 붙인다 — 훅 설정을 못 쓰면 --settings 는 안 붙이고 --setting-sources '' 와 --strict-mcp-config 는 붙인다. 없는 파일을 가리키는 플래그는 보조 실패를 치명으로 만든다. 그리고 표시 줄은 stdout 이 통째로 빌 때 안 쓴다 — 쓰면 그것이 첫 줄이 되어 「init 이 안 나오면 stderr 가 첫 줄」이 깨진다.

한계 하나를 찾아 적었다 (business-rules 8.1). R3(훅 설정 쓰기)이 보조인데 그 파일이 gatewayAuthFields 도 든다. 디스크 실패로 못 쓰면 게이트웨이 노드가 Not logged in 으로 늦게 죽는데, 같은 회차가 R6(자격증명 복사)을 치명으로 올린 근거가 바로 「늦게 아는 실패를 앞으로 당긴다」였다. 등급은 안 바꿨다 — ⑨ 와 ⑪ 이 닫은 값이고 FD 가 뒤집을 자리가 아니다. 조건부 등급(gatewayAuthFields 가 무엇을 돌려줬는지로 가른다)은 그 함수가 실제 홈을 읽어 시험이 조건을 못 세우므로 안 골랐다. 진행자에게 넘긴다.

남은 것 하나 — decisions.md 6절의 실측 행. 팩 문서라 이 단계의 승인 뒤에 싣는다."
**Context**: CONSTRUCTION · U1 isolation · Functional Design Step 5 ~ 7 완료 · 승인 대기

---

---

## CONSTRUCTION — U1 isolation · Code Generation (Part 1 계획 · Part 2 실행)
**Timestamp**: 2026-09-12T15:40:00Z
**User Input**: "isolation 코드 써"
**AI Response**: "이 지시가 둘을 겸한다 — 앞 단계(Functional Design)의 승인과 Code Generation 착수다. FD 는 산출물 셋을 낸 채 승인을 기다리고 있었고 다음 단계로 가라는 지시가 곧 그 승인이다. 같은 문서 루트의 obs 유닛이 NFR Requirements 에서 같은 판단을 했다.

Part 1 계획을 construction/plans/isolation-code-generation-plan.md 에 냈다. obs 때와 달리 병렬로 안 갈랐다 — 다섯 파일이 한 패키지 안에서 한 시그니처 변경(Fixed(dir) · Instrument(..., c))을 함께 받아서, 갈라 짜면 세 갈래가 컴파일되는 시점이 하나뿐이라 병렬이 아니라 대기가 된다.

계획이 FD 에 없던 자리 셋을 정했다. WriteHookSettings 가 계장 디렉터리가 아니라 가짜 홈을 받는다(배치는 claude 의 것이고 hook.go 는 CLAUDE_CONFIG_DIR 을 모르는 편이 낫다) · --setting-sources '' 를 hook.go 에서 어댑터로 옮긴다(그것은 파일을 안 가리키므로 훅 쓰기의 성패와 무관하고, 두 곳에 두면 불변식 ② 가 깨진다) · 선별이 사건을 구조체가 아니라 map[string]json.RawMessage 로 읽는다(모르는 필드의 모양 하나가 줄 전체를 떨어뜨리지 않게).

Part 2 를 돌렸다. 제품 파일은 행렬 그대로 다섯이고 새 파일은 mcp.go 하나다. 행렬 밖은 새 시험 파일 셋(mcp_test.go · instrument_test.go · logs_test.go)뿐이라 파일 행렬 1절에 행으로 더했다 — 그 문서 6절 ② 의 절차다. 행렬이 빨개진다고 예고한 여섯 중 다섯은 실제로 빨갰고(claude_test.go 하나 · hook_test.go 넷) worker_unix_test.go 는 안 빨갰다 — stubHarness 가 이미 type:\"result\" 봉투를 찍는다.

CA0 이 전부 초록이다. go test ./... · vet · glyphscan(104 파일 · 장식 문자 0) · gofmt · 워킹트리 청결 · 크로스 빌드와 심볼 상한(net/http 6, 상한 50). 커버리지 미달 0 이고 전체 87.0% · internal/enode 85.6% 다. 커버리지 실행이 cmd/enodectl/probe.lock 을 바꾼 것은 되돌렸다 — obs 때와 같은 자리다.

초록이 곧 시험이 문다는 뜻은 아니라서 변이 넷을 돌렸다. type 검사를 끄면 ⑯ 의 시험 둘이 빨개지고, 링 tee 를 되살리면 ⑲ 의 시험이 빨개진다. 그런데 selectLogs 를 원문 반환으로 되돌렸을 때 아무것도 안 빨개졌다 — logs_test.go 가 함수를 직접 부르고 배선을 안 쟀기 때문이다. 함수가 있어도 runHarness 가 안 부르면 원문이 그대로 Record 로 올라가 봉인된다. 배선을 재는 시험(TestRunHarness_TheUploadedLogIsTheFilteredOne)을 더했고 같은 변이가 그 뒤 빨개진다.

FD 가 승인에 걸어 둔 하나를 실었다 — requirements/harness-components/decisions.md 6절에 실측 행 ⑳ 이다. 사건 종류 다섯과, 실측이 뒤집은 전제(첫 줄이 init 이 아니었다)와, 굳힌 것 셋(mcp_servers 는 배열 · usage 에 숫자 아닌 것이 있다 · 본문이 블록 밖에도 있다)을 적었다. 플래그도 실물로 쟀다 — claude 2.1.266 의 --help 가 --mcp-config <configs...> 로 가변인자임을 보여 등호 형태를 고른 근거(ADR-034 §3 ②)가 섰다.

넘기는 것이 다섯이고 그중 하나가 새로 드러났다 — decisions.md 6.1 의 문자열 검사 스크립트가 「자리는 scripts/ 다. 짓는 것은 Construction 의 몫」으로 자리만 적힌 채 어느 유닛에도 배정되지 않았다. U1 의 완료 조건에도 파일 행렬에도 없어서 이 유닛이 안 지었다. 오늘은 진행자가 손으로 돈다.

CA1 은 이 유닛이 안 닫는다 — 집행자는 이 유닛을 구현하지 않은 사람이다(scene-gates.md 2절 머리)."
**Context**: CONSTRUCTION · U1 isolation · Functional Design 승인 · Code Generation 완료 · CA0 초록 · CA1 대기

---

## CONSTRUCTION — U2 contract-vocab · Functional Design 착수 (Step 1 ~ 4)
**Timestamp**: 2026-09-12T15:52:00Z
**User Input**: "다음할일해라"
**AI Response**: "U1 isolation 은 단계가 다 돌았고 남은 둘(CA1 측정 · PR 병합)이 전부 사람의 것이다 — CA1 의 집행자는 이 유닛을 구현하지 않은 사람이고(scene-gates.md 2절 머리) 병합은 그 게이트가 초록인 뒤다(CONVENTIONS 3.3). 그래서 이 회차의 다음 유닛으로 갔다. 착수 순서가 U1 -> U2 이고 U2 contract-vocab 의 선행은 없다(unit-of-work.md 0절).

브랜치를 unit/isolation 에서 땄다 — CONVENTIONS 3.1 은 회차 브랜치를 적는데 그러지 않은 이유 둘을 계획 6절에 적었다. U1 이 아직 main 에 없고, aidlc-docs/taeels/aidlc-state.md 가 자동 병합이 안 되는 파일이라 회차 브랜치에서 따면 U1 의 절과 U2 의 절이 파일 꼬리에서 각자 자라 병합에서 부딪친다. 코드는 안 부딪친다 — 행렬이 U1 과 U2 의 파일 교집합을 0 으로 센다.

계획을 짓기 전에 이 유닛이 만지는 자리를 실제로 읽었고 갈린 문장 셋을 찾았다.

① features.md 3.5 의 「runctl schema steps 가 새 키를 낸다 — 구조체에서 뽑으므로 저절로 는다」가 거짓이다. printFields(shape.go:172)는 reflect 로 Step 의 최상위 필드만 찍고 Step.Agent 는 map[string]interface{} 라(contract.go:447) 출력이 언제나 agent map[string]any 한 줄이다. cmd/runctl 은 AgentParams 도 agentKeys 도 import 하지 않는다.

② in.from 의 「<단계>.<blob 이름>」이 절반만 참이다. blob 이름 공간은 Run 하나에 평평하고(api.go:107-108) claim.go:520 이 그 이름을 그대로 $IN 의 파일 이름으로 쓴다. 점은 이름의 한 글자이고 단계 참조가 아니다.

③ ②가 게이트 하나를 깬다. scene-gates.md 3절 CA5 는 첫 단계가 $OUT/pack 을 내는데 둘째 단계가 in.from: [\"fetch.pack\"] 을 적는다. 앞이 내는 blob 이름은 pack 이고, 없는 입력은 값으로 취급되므로(claim.go:505-535) 단계가 죽지 않고 팩 없이 돈다 — CA5 를 그대로 돌리면 빨간 이유가 팩 코드가 아니라 이름 불일치가 된다.

그리고 행렬에 없는 파일 하나가 이 유닛의 것이었다 — internal/contract/planshape.go 다. PlanShape 의 두 줄이 「agent 는 max_turns, max_tokens, ask, model, harness — nothing else」와 「그 밖의 키는 계획 전체를 거절한다」를 못으로 박고 있어서, agentKeys 에 둘을 더하면 그 문장이 계획에게 거짓말을 한다. components.md 1.6 은 Grammar 만 적었고 행렬에는 행이 없다.

타입 검증의 자리가 어긋나 있는 것도 찾았다. decisions.md 2절은 검증 자리를 contract.Validate 로 적었는데 그것이 참인 것은 키 이름이고 값의 타입은 아니다 — agent.mcp 를 문자열로 적은 계약은 400 을 안 받고 노드까지 가서 json.Unmarshal 로 죽는다. 스토리 US-7 은 「제출에서 걸린다」이고 unit-of-work.md 2절의 완료 조건은 「parseAgentParams 가 문구로 낸다」로, 둘이 같은 자리를 안 가리킨다.

물음 일곱을 aidlc-docs/taeels/construction/plans/contract-vocab-functional-design-plan.md 에 냈다 — ① 타입 위반을 어디서 잡나 ② agent.pack 과 in.from 의 대조를 제출에서 하나 ③ runctl schema 에 열거 면을 만드나 ④ 계획에게 이름의 출처를 가르치나 ⑤ examples/mcp.json 이 무엇을 담나 ⑥ 갈린 문장 셋을 어디에 적나 ⑦ GLOSSARY.md 를 이 단계가 만드나. 일곱 다 Other 선택지를 든다.

⑦ 을 물은 이유 — CLAUDE.md 의 새 규약이 CP 와 CA 를 이름으로 지목하고 자리를 루트 GLOSSARY.md 로 박았는데 그 파일이 main 에도 없다. 같은 규약이 「푼 말을 모르면 적지 말고 묻는다」로 짐작을 금지하므로 답으로 받는다."
**Context**: CONSTRUCTION · U2 contract-vocab · Functional Design Step 1 ~ 4 완료 · 답 대기

---

## CONSTRUCTION — U2 contract-vocab · Functional Design 산출 (Step 5 ~ 7)
**Timestamp**: 2026-09-12T16:20:00Z
**User Input**: "다음으로"
**AI Response**: "계획 파일의 [Answer] 일곱이 다 비어 있어서 먼저 답을 받았다. 물음은 파일에 있고 (질문은 채팅에 안 쓴다) 받은 답을 그 파일의 [Answer] 뒤에 적었다 — A · A · A · A · A · A · B. 모호한 답이 0 이고 서로 어긋나는 답도 0 이라 추가 질문 없이 Step 6 으로 갔다.

답 여섯이 권장이고 하나가 갈렸다 — 7=B 는 GLOSSARY.md 를 이 유닛 밖의 커밋으로 뺐다 (규약 자체가 그랬다: 76e2201). 푼 말은 여전히 사람에게서 와야 한다.

답 1=A 와 2=A 가 검증 자리를 Mediator 로 올렸다. contract.Validate 가 agent.mcp 의 타입(문자열 배열)과 agent.pack 의 타입(문자열)을 보고, agent.pack 이 적은 이름이 같은 단계의 in.from 에 있는지를 대조한다. 그러면 US-7 의 「제출에서 걸린다」가 참이 되고, 계획이 지은 단계도 같은 검사를 받는다 — 근거를 코드로 확인했다: checkplan.go:82 의 빠른 훅과 store/expand.go:206 의 applyExpands 가 둘 다 같은 Validate 를 부른다. 노드 쪽 parseAgentParams 의 문구도 같은 문장으로 고쳐 둘째 겹으로 남긴다.

답 2=A 가 CA5 의 값을 바꿨다. 오늘은 in.from 의 이름 불일치가 조용한 실패다 — 없는 입력은 값이라 단계가 안 죽고 팩 없이 돈다 (claim.go:505-535). Validate 가 대조하면 그 계약은 제출에서 400 이므로, scene-gates.md CA5 의 명령을 고치지 않으면 그 게이트를 시작조차 못 한다. 답 6=A 가 그것을 함께 고치게 했다.

산출물 셋을 construction/contract-vocab/functional-design/ 에 냈다. 규칙은 여섯이고 (R1 모르는 키 400 은 오늘 그대로 · R2~R6 이 새로 선다) 문구는 영어로 지었다. grammar_test.go 가 재는 형태에 맞춰 한 문장에 한 거절 사유로 썼고, 문법 대조가 필요한 셋(R2 · R4 · R5)의 mustSay 와 wantErr 를 표로 적었다.

예시를 오늘 코드에 실측으로 걸었다 — 임시 시험으로 examples/mcp.json 의 내용을 Contract 로 풀고 Validate 에 태웠다. requires 의 mcp.probe 는 Attrs 로 들어가고(map[string]string{\"mcp.probe\":\"1\"}) 유일한 거절은 unknown field \"mcp\" 이며 그것을 뺀 같은 계약은 통과한다. 그래서 예시는 agentKeys 가 느는 같은 커밋에서 초록이 된다 — 먼저 넣으면 example_test 가 확정적으로 빨갛다. 임시 시험 파일은 지웠다.

답 밖에서 더 정한 것 다섯을 business-rules 6절에 이름으로 적었다 — in.from 의 타입 검사(답 2=A 의 대조가 그 값을 읽어야 한다) · 빈 문자열 이름 거절 · PlanShape 와 agentKeys 가 갈리지 않는 것을 시험이 잰다(Grammar 에는 그 장치가 있고 PlanShape 에는 없었다) · 팩은 하나다 · requires 와 agent.mcp 를 묶어 검증하지 않는다(워크스페이스와 팩에서 온 이름은 광고에 없어 requires 에 적을 수 없고, 묶으면 정당한 계약이 거절된다).

한계 둘을 남겼다 (7절). agent 의 나머지 다섯 키는 타입 검사가 여전히 노드뿐이라 비대칭이 남는다 — 답 1=A 가 새 키 둘로 범위를 정했고 features.md 3.5 의 범위도 새 어휘다. 그리고 max_tokens 의 개명이 이 팩 밖이라 알려진 키 일곱에 옛 이름이 그대로 있다.

팩 문서 둘(decisions.md 6절의 실측 행 · scene-gates.md CA5)은 이 단계의 승인 뒤에 싣는다 — U1 이 ⑳ 에서 같은 순서를 밟았다."
**Context**: CONSTRUCTION · U2 contract-vocab · Functional Design Step 5 ~ 7 완료 · 승인 대기

---

## CONSTRUCTION — U2 contract-vocab · Code Generation (Part 1 계획 · Part 2 실행)
**Timestamp**: 2026-09-12T17:05:00Z
**User Input**: "다음"
**AI Response**: "이 지시가 둘을 겸한다 — Functional Design 의 승인과 Code Generation 착수다. U1 과 obs 가 같은 판단을 했다. 승인이 열어 준 팩 문서 둘도 이 단계가 실었다.

Part 1 계획을 construction/plans/contract-vocab-code-generation-plan.md 에 냈다. 단계 여덟이고 체크박스 스물여섯이 전부 [x] 다. 갈래를 안 갈랐다 — 제품 파일 다섯 중 넷이 한 패키지이고, 예시가 agentKeys 와 같은 커밋이어야 해서 갈래가 서로를 기다린다.

Part 2 를 돌렸다. contract.go 에 agentKeys 둘과 새 함수 agentValues(R2~R6) · grammar.go 에 절 하나 · planshape.go 의 목록과 설명 · examples/mcp.json · agent.go 의 AgentParams 둘과 checkComponentTypes 다. 새 시험 파일이 0 이고 기존 셋에 붙었다 — 행렬 3절이 예고한 그 셋이다. cmd/runctl 의 소스 diff 는 0 이다 (답 3=A).

CA0 이 전부 초록이다. go test ./... exit 0 · 커버리지 미달 0(전체 6795/7804 = 87.1% · internal/contract 89.3% · internal/enode 85.6%) · vet · gofmt · U+2605 전수 grep 0 · glyphscan 104 파일 · 크로스 빌드 셋 · 심볼 상한(tls 1 · http 6) · 라우트 26 · 워킹트리 청결. 커버리지 실행이 cmd/enodectl/probe.lock 을 바꾼 것은 되돌렸다 — obs · U1 과 같은 자리다.

FD 5절이 internal/contract 를 「하한에 가장 가까운 자리」로 적었는데 표준 명령으로 재면 89.3% 다. 단독 측정(79.5%)과 다른 이유는 -coverpkg=./... 가 다른 패키지의 시험이 덮은 문장을 함께 세기 때문이다.

CA3 의 앞 절반을 runctl lint 로 쟀다 — 그 명령이 Validate 를 부른다. 모르는 키를 적으면 허용 목록이 일곱으로 함께 늘어 나오고(답 3=A 가 기댄 성질이 실물로 섰다), agent.mcp 를 문자열로 적은 계약과 agent.pack 만 적은 계약이 R2 · R5 의 문장으로 거절된다. runctl example mcp 는 exit 0 이고 lint 경고 0 이다. runctl schema steps 도 쟀다 — agent map[string]any optional 한 줄 그대로이고 하위 키가 안 나온다.

변이 다섯을 돌려 다 빨개졌다 — agentKeys 되돌리기 · agentValues 호출 빼기 · PlanShape 목록에서 둘 빼기 · parseAgentParams 의 타입 검사 빼기 · Grammar 의 절 지우기. U1 과 달리 구멍이 0 인 이유는 규칙이 전부 순수 함수 하나를 지나기 때문이다 — 배선이 한 자리이고 변이 ② 가 그 자리를 직접 잰다.

실측이 FD 의 주장 하나를 뒤집었고 고쳤다. FD 는 「CA5 의 이름 불일치는 조용한 실패다」로 적었는데, Validate 에 in.from 의 정적 검사가 이미 있다(contract.go:1109-1126 · ADR-023 §6.2.1). CA5 의 계약을 그대로 넣어 재니 step \"work\": in.from refers to \"fetch.pack\", which no step produces 로 오늘도 제출에서 거절된다. 게이트가 안 도는 것은 그대로 참이고 이유가 침묵이 아니라 그 거절이며, 그 문구가 팩을 안 가리켜서 게이트를 돌리는 사람이 팩 코드의 결함으로 읽는다. R5 가 닫는 것은 in.from 을 아예 안 적은 경우 하나로 좁아졌다 — 오타는 있던 검사가 막고 빠뜨림만 성공으로 온다. 고친 자리는 FD 계획 1.2 · business-rules 2절 · business-logic-model 8절 · domain-entities 4.1 과 contract.go 의 주석이다.

팩 문서 둘을 실었다. decisions.md 6절에 실측 행 ㉑(열거 면 · blob 이름 공간 · CA5 를 한 행에)과 6절 제목의 날짜, scene-gates.md CA5 의 명령(out: [\"pack\"] · in.from: [\"pack\"] 과 왜인지). 파일 행렬에는 planshape.go 행을 더해 만지는 파일이 16 이 되고 4절 제목이 「둘」에서 「셋」이 되었다.

넘기는 것 다섯이고 그중 하나가 새로 드러났다 — 회차 문서 루트의 aidlc-state.md 가 decisions.md 의 행을 「열아홉」으로 적는데 실제로는 스물하나다(U1 의 ⑳ · 이 유닛의 ㉑). 같은 파일의 「요약 표에 없던 파일 둘」도 셋이다. 그 루트는 회차 진행자의 것이라 안 고쳤다 (CONVENTIONS 3.4)."
**Context**: CONSTRUCTION · U2 contract-vocab · Functional Design 승인 · Code Generation 완료 · CA0 초록 · CA3 앞 절반 초록

---

## CONSTRUCTION — U3 advert · Functional Design (Part 1 계획 · 물음 일곱)
**Timestamp**: 2026-09-13T00:00:00Z
**User Input**: "u3 advert 시작"
**AI Response**: "U3 advert 를 착수했다. 브랜치 unit/advert 를 unit/contract-vocab 에서 땄다 — 선행이 U1 · U2 둘이고 aidlc-state.md 가 자동 병합이 안 되는 파일이라 회차에서 따면 세 유닛의 절이 부딪친다 (계획 6절).

계획을 짓기 전에 만지는 자리를 코드로 읽고 yaml 실측 열일곱을 돌렸다. 갈린 자리 셋을 찾았다. ① decisions.md 2절의 「그 밖의 키는 그대로 허용목록에 옮긴다」가 오늘 코드로 성립하지 않는다 — yaml 이 모르는 키를 말없이 버리므로 옮길 맵이 안 생기고, U1 의 allowlistEntry 이음매 주석과 파일 행렬 2.1 의 「U3 은 U1 의 것을 안 고친다」가 같은 자리를 두고 서로 다르게 말한다. ② env 가 이름인지 값인지가 갈린다 — features.md 3.2 는 「값은 적지 않는다」인데 형식은 map[string]string 이고 allowlistEntry 가 통째로 옮기며 ADR-035 §4.4 의 예시에는 env 가 아예 없다. ③ 현황판이 새 키를 원문으로 낸다 — 같은 사실이 실행 도구 = claude 와 harness.claude = 1 두 줄로 보인다.

yaml 실측이 검증 범위를 쟀다 — 잡는 것은 종류뿐이고 일곱이 그대로 통과한다 (command: 5 가 \"5\" 가 되고, 빈 서버 · 빈 이름 · command 와 url 이 둘 다인 선언이 다 선다). 동작 중립의 근거 셋도 쟀다 — harnesses 가 하나라 break 를 걷어도 결과가 안 바뀌고, Detector 의 시계가 CA2 의 「탐지 주기 뒤」를 배선 0 으로 세우며, hasCapability 가 mcp.<이름> 하나로 참이 되어 하네스 없는 노드가 광고를 낸다.

계획을 construction/plans/advert-functional-design-plan.md 에 냈고 물음이 일곱이다 — 설정 검증의 범위 · 「그 밖의 키」의 자리 · env 의 뜻 · Fingerprinter.Probe 의 오류 · 누락 사유 로그의 빈도 · 하네스 없는 노드의 광고 · 현황판 표시. 답 대기다."
**Context**: CONSTRUCTION · U3 advert · Functional Design Step 1 ~ 4 완료 · 답 대기

---

## CONSTRUCTION — U3 advert · Functional Design (답 수집 · 산출물 셋)
**Timestamp**: 2026-09-13T00:30:00Z
**User Input**: "6=B, 7=B,  나머지 A"
**AI Response**: "답 일곱을 계획 2절에 적었다 — 1=A · 2=A · 3=A · 4=A · 5=A · 6=B · 7=B. 모순도 모호도 없어 확인 질문 파일을 안 만들었다.

산출물 셋을 construction/advert/functional-design/ 에 냈다.

domain-entities.md — Local.MCP 와 MCPServer 의 형식. 답 2=A 로 Extra 필드가 서고 UnmarshalYAML 이 같은 노드를 두 번 푼다(그림자 구조체로 종류 검사를 지키고 맵으로 모르는 키를 집는다). 답 3=A 로 env 가 이름에서 이름으로 가고 허용목록에 ${이름} 참조로 나간다. 광고 키 셋과 값이 언제나 \"1\" 인 이유(매처가 완전 일치만 본다). Fingerprinter 의 계약과 종류 셋.

business-rules.md — 거절 다섯(R1~R5)이 「어느 종류인가」를 한 값으로 만든다. env 의 꼴 검사 R6. 뜨나 판정 둘과 못 잡는 것 셋. 순회의 오류 규칙 R7~R8(답 4=A). 누락 사유를 탐지마다 내는 R9(답 5=A). 광고의 문턱 R10~R11(답 6=B)과 그 침묵을 막는 R12. 화면의 R13~R14(답 7=B). 동작 중립을 판정하는 기존 시험. 한계 다섯.

business-logic-model.md — enode.yaml 에서 현황판까지의 한 줄기. 시계 둘이 가르는 자리와 CA2 의 「탐지 주기 뒤」가 배선 0 으로 서는 이유. costlyAttrs 의 순회 코드. CA2 · CA3 이 어디서 초록이 되나. 완료 조건 여덟과의 대조.

어긋남 하나를 기록했다 — 답 6=B 가 hasCapability 를 고치므로 unit-of-work.md 3절의 「cheapAttrs · capabilities · Detector 에 diff 0」이 글자로는 서고 뜻으로는 안 선다. capabilities 의 글자는 그대로이고 동작이 바뀐다. 그 대가는 물음 6 의 B 가 미리 적었다.

파장 둘이 승인 뒤로 간다 — 파일 행렬 2.1 의 「U3 은 U1 의 것을 안 고친다」가 답 2=A 로 거짓이 되고, internal/api/ui 의 0 이 답 7=B 로 깨진다. 만지는 파일이 16 에서 17 이 된다. decisions.md 6절의 실측 행과 2절의 env 줄도 같은 시점이다 — U1 · U2 가 밟은 순서 그대로 승인 뒤에 싣는다."
**Context**: CONSTRUCTION · U3 advert · Functional Design Step 5 ~ 7 완료 · 승인 대기

---

## CONSTRUCTION — U3 advert · Functional Design 승인
**Timestamp**: 2026-09-13T00:45:00Z
**User Input**: "승인."
**AI Response**: "Functional Design 을 닫고 커밋했다 — 계획(답 일곱 포함) · 산출물 셋 · 상태 · 감사 여섯 파일이다. 팩 문서와 회차 문서의 파장은 Code Generation 이 싣는다 (U1 의 ⑳ · U2 의 ㉑ 과 같은 순서).

Code Generation Part 1 로 간다."
**Context**: CONSTRUCTION · U3 advert · Functional Design 승인 · Code Generation 착수

---

## CONSTRUCTION — U3 advert · Code Generation Part 1 (계획)
**Timestamp**: 2026-09-13T00:50:00Z
**User Input**: (없음 — 앞의 「승인」이 이 단계의 착수다)
**AI Response**: "Code Generation 계획을 construction/plans/advert-code-generation-plan.md 에 냈다. 단계 열이고 체크박스 마흔둘이다.

제품 파일 넷이다 — config.go(Local.MCP · 거절 여섯 · SampleLocal) · mcp.go(태그 · Extra · UnmarshalYAML · allowlistEntry · mcpUp · mcpFP) · detect.go(Fingerprinter · harnessFP · repoFP · 순회 · hasCapability) · format.mjs(접두 둘). 새 시험 파일 하나(detect_mcp_test.go)와 기존 셋에 붙인다.

갈래를 안 나눈다 — Go 셋이 서로를 기다려 컴파일되는 시점이 하나뿐이다. format.mjs 는 진짜 독립이지만 여섯 줄이라 갈래 비용이 이득보다 크다.

Step 6 이 실측 하나다 — 이 기계의 claude 2.1.266 으로 env 의 ${이름} 참조를 하네스가 펴는지 잰다. 안 펴면 env 를 허용목록에서 빼고 노드 환경 상속에 맡긴다. 어느 쪽이든 값이 파일에 안 실린다는 규칙은 산다.

Step 8 이 변이 다섯이다 — hasCapability 의 접두 · allowlistEntry 의 순서 · mcpUp 의 remote 갈래 · 옛 harness 키 · repoFP 의 WorkspaceID fallback.

FD 에 없던 자리 셋을 이 계획이 정했다 — validateMCP 를 config.go 에 두는 것, R12 를 capabilities 가 nil 을 돌려주기 직전에 내는 것(그 대가로 광고 주기마다 나온다), 로그 키 이름 셋.

승인이 열어 준 문서 셋(decisions.md 6절과 2절 · 파일 행렬 둘)은 Step 10 이 싣는다."
**Context**: CONSTRUCTION · U3 advert · Code Generation Part 1 완료 · 계획 승인 대기

---

## CONSTRUCTION — U3 advert · Code Generation Part 2 (실행)
**Timestamp**: 2026-09-13T01:30:00Z
**User Input**: "생성해"
**AI Response**: "계획 열 단계를 돌렸고 체크박스 마흔여덟이 전부 [x] 다.

제품 파일 넷이다 — config.go(Local.MCP · validateMCP · validateEnvNames · isEnvName · SampleLocal 한 줄) · mcp.go(태그 다섯 · Extra · UnmarshalYAML · kind · mcpUp · mcpFP · envRefs · allowlistEntry) · detect.go(Fingerprinter · fingerprinters · harnessFP · repoFP · 순회 · hasCapability · warnMCPWithoutCapability) · format.mjs(접두 둘). 새 시험 파일 셋과 고친 옛 시험 둘이다.

CA0 이 전부 초록이다 — go test ./... 18 패키지 · 커버리지 미달 0(전체 6887/7902 = 87.2% · internal/enode 1795/2086 = 86.0% · internal/api/ui 98.4%) · vet · gofmt · U+2605 전수 grep 0 · glyphscan 104 파일 · 크로스 빌드 셋 · 심볼 상한(tls 1 · http 6) · 라우트 26 · node --test 75 통과 · 워킹트리 청결. 커버리지 실행이 바꾼 cmd/enodectl/probe.lock 은 되돌렸다 — obs · U1 · U2 와 같은 자리다.

Step 6 의 실측이 FD 의 미정 하나를 닫았다. claude 2.1.266 에 env: {PROBE_X: ${SRC}} 를 주고 --strict-mcp-config 로 돌리니 서버 프로세스의 PROBE_X 가 SRC 의 값(expanded-value-42)이었다. 하네스가 참조를 편다 — 그래서 envRefs 의 ${이름} 꼴이 확정이고 「안 펴면 env 를 안 쓴다」는 대안이 닫혔다. 덤으로 U1 의 ⑳ 이 이 기계에서도 섰다 — 첫 줄이 system/init 이 아니라 system/hook_started 였다.

CA3 의 뒤 절반을 합성 함대로 쟀다. Mediator 를 시험 DB 위 127.0.0.1:18080 에 띄우고 노드 둘을 심었다. runctl capabilities 에 mcp.probe 와 harness.claude 가 나오고, requires: mcp.probe 를 적은 계약이 node-with-probe 에만 가고, mcp.nope 는 422 다. 셋 다 코드 0 으로 닫혔다 — store.Capabilities 가 키 이름을 안 가리고 Capability.Satisfies 가 완전 일치를 본다. 덤으로 희소성 정렬도 실물로 섰다: mcp 를 안 요구하는 계약은 node-plain 으로 갔다. 게이트를 돌릴 사람이 밟을 자리 하나를 적었다 — 같은 run_id 로 두 번 제출하면 둘째는 422 가 아니라 200 이고 본문이 FAILED 와 저장된 reject 다(멱등).

변이 다섯을 돌려 다 빨개졌다. 구멍 0 이다. 다만 계획이 예고 안 한 옛 시험 둘이 확정 빨강이었다 — enode_test.go 의 TestDetectEmpty(광고에 실릴 수 있는 키를 셋으로 못 박는다. harness. 접두를 함께 허용했다)와 mcp_test.go 의 allowlistEntry 시험(env 가 값 그대로 실리는 것을 잰다. 답 3=A 가 참조로 바꿨다). 둘 다 행렬 밖이라 행렬 3절에 행으로 더했다.

시험 픽스처 하나가 검사의 한계를 드러냈다 — env 값으로 ghp_secret 을 적고 거절을 기대했는데 통과한다. [A-Za-z_][A-Za-z0-9_]* 를 그대로 만족한다. FD business-rules 2절이 이미 적은 한계이고, 픽스처를 실제로 걸리는 값으로 고치고 그 한계를 재는 시험을 따로 더했다.

계획에 없던 자리 둘을 이 단계가 정했다 — 거절의 검사 순서(좁은 것을 먼저: R1 · R3 · R4 · R5 · R2)와 mcpUp 의 문구 둘(절대경로는 PATH 를 안 보므로 not found in PATH 가 거짓이 된다).

회차 밖으로 낸 것은 decisions.md 6절의 실측 행 ㉒ 와 2절의 env 줄, 그리고 파일 행렬(2.1 의 U3 칸 · format.mjs 와 시험 넷 · 만지는 파일 16 -> 17 · 패키지 3 -> 4 · 3절의 옛 시험 둘 · 4.4 · 4.5 절)이다. scene-gates.md 와 component-methods.md 는 안 고쳤다.

넘기는 것 여섯이다. 무거운 셋: 완료 조건의 「capabilities 에 diff 0」이 글자로는 서고 뜻으로는 안 선다(답 6=B) · R12 가 탐지 주기가 아니라 광고 주기마다 나온다 · 라벨로 선언 없이 mcp.<이름> 을 광고할 수 있다(오늘 그대로의 성질이고 이 유닛이 안 좁혔다).

CA2 는 사람이 잰다 — 집행자는 이 유닛을 구현하지 않은 사람이고 눈 검증 S1 도 그 사람이 본다."
**Context**: CONSTRUCTION · U3 advert · Code Generation 완료 · CA0 초록 · CA3 뒤 절반 초록 · CA2 사람 대기

## CONSTRUCTION — U4 sources · Functional Design (계획)
**Timestamp**: 2026-09-13T03:16:24Z
**User Input**: "u4 시작"
**AI Response**: "U3 가 코드까지 닫혔으므로 회차의 넷째 유닛 U4 sources 를 착수했다. 브랜치 unit/sources 를 unit/advert 에서 땄다 — 선행이 U1 · U2 · U3 셋이고 aidlc-state.md 가 자동 병합이 안 되는 파일이라 회차에서 따면 네 유닛의 절이 파일 꼬리에서 부딪친다 (계획 6절).

계획을 짓기 전에 만지는 자리를 코드로 읽고 claude 2.1.266 으로 돌렸다. 갈린 자리 넷을 찾았다 — ① 워크스페이스 .mcp.json 은 하네스가 스스로 쓰는 파일이라 우리 어휘에 없는 키 셋(type · headers · 값으로서의 env)을 담는다 ② 그것을 MCPServer 로 받으면 셋을 잃거나 망친다 (Extra 가 yaml 전용이라 type 과 headers 가 사라지고, envRefs 가 값을 두 번 감싸 ${${X}} 가 되며, 종류 오류 하나가 파일 전체를 죽인다) ③ url 만 적힌 항목을 하네스가 말없이 버린다 — 실측에서 목록에 이름조차 안 나왔고, 그것이 ADR-035 §4.4 예시 그대로 적은 원격 노드 선언의 오늘 모양이다 ④ ${} 는 한 겹만 펴진다.

물음 일곱을 냈다 — 워크스페이스 파일을 누가 읽나(순수성) · 원문을 어떤 꼴로 나르나 · 조용히 사라지는 원격 항목을 막나 · 못 읽는 파일의 등급 · Notes 가 나가는 면 · 없는 이름이 여럿일 때의 거절 · 원격 인증. 계획은 aidlc-docs/taeels/construction/plans/sources-functional-design-plan.md 다."
**Context**: CONSTRUCTION · U4 sources · Functional Design Part 1 완료 · 답 일곱 대기

## CONSTRUCTION — U4 sources · Functional Design (산출물)
**Timestamp**: 2026-09-13T03:38:12Z
**User Input**: "모두 권장대로"
**AI Response**: "답 일곱을 전부 A 로 적고 계획 3절의 체크박스 스물을 [x] 로 닫았다. 산출물 셋을 construction/sources/functional-design/ 에 냈다.

domain-entities 는 출처를 「누가 쓰는 파일인가」로 갈랐다 — 노드 선언은 우리 어휘(MCPServer)이고 워크스페이스와 팩은 하네스 어휘다. 그래서 Components.Servers 의 타입을 map[string]MCPServer 에서 허용목록 항목(map[string]map[string]any)으로 올렸다. 노드 것은 allowlistEntry 를 거쳐 들어오고 워크스페이스 것은 원문 그대로 들어온다 — 번역하는 코드가 없으므로 계획 1.2 의 손상 셋이 구조적으로 못 생긴다. Job 에 넷을 더한다 (NodeMCP · WorkspaceMCP · WorkspaceMCPErr · Log).

business-rules 는 열넷이다. R1 요청 필터 · R2 요청 0 이면 파일도 안 연다 · R3 노드가 워크스페이스를 이긴다(Note) · R4 없는 이름은 하네스를 안 띄운다 · R5 나머지는 Note · R6 워크스페이스가 없으면 출처도 없다(os.TempDir 의 파일을 안 믿는다) · R7 못 읽는 파일은 거절 · R8 command 도 url 도 없는 요청된 항목은 거절 · R9 type 은 없을 때만 채운다 · R10 Notes 는 이름만 · R11 거절로 끝나도 Notes 를 채워 돌려준다 · R12 광고 불변 · R13 credential 은 파일에 안 나간다 · R14 거절이 전부 exec 앞이다.

답 둘이 만나 규칙이 하나로 줄었다 — 4=A 의 「요청이 있을 때만 치명」이 1=A 아래서 조건문이 아니라 호출 자리로 보장된다. 요청이 0 이면 파일을 아예 안 열어 「요청 0 인데 깨진 파일」이 구조적으로 안 생긴다.

business-logic-model 은 resolveComponents 의 걸음 여섯과 그 순서의 뜻을 적었다 (파일 오류가 이름 해결보다 앞 · type 채우기가 집기보다 뒤 · 거절이 맨 뒤). runHarness 의 ① ~ ⑨ 중 자라는 자리는 ② 하나이고 ②와 ③ 사이에 Notes 로그 한 줄이 는다. CA4 의 세 줄을 규칙으로 각각 매핑했고, U5 가 딛는 이음매 다섯(필터 · 집는 자리 · 나르는 꼴 · 여는 자리 · Notes)을 이름으로 적었다.

한계 넷을 이름으로 졌다 — 워크스페이스 서버가 실제로 뜨는지는 안 본다 · 원격 선언만으로는 인증이 안 실려서 소유자가 headers 를 Extra 로 적어야 한다 · 저장소가 env 에 평문 값을 적으면 그대로 옮겨진다 · 저장소가 새 이름을 심는 것은 계약의 요청이 있어야만 열린다."
**Context**: CONSTRUCTION · U4 sources · Functional Design Part 2 완료 · 단계 승인 대기

## CONSTRUCTION — U4 sources · Functional Design 승인
**Timestamp**: 2026-09-13T03:44:36Z
**User Input**: "승인, 코드 써"
**AI Response**: "Functional Design 을 닫고 담당 상태 파일에 U4 절을 더했다. 커밋 하나로 계획 · 산출물 셋 · 상태 · 감사를 함께 실었다 (CONVENTIONS 3.3 의 단계 승인마다 커밋). 다음은 Code Generation Part 1 — 계획을 내고 승인을 받는다."
**Context**: CONSTRUCTION · U4 sources · Functional Design 승인 · Code Generation Part 1 착수
