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

## CONSTRUCTION — U4 sources · Code Generation Part 1 (계획)
**Timestamp**: 2026-09-13T03:48:34Z
**User Input**: "승인, 코드 써"
**AI Response**: "Code Generation 계획을 냈다 — construction/plans/sources-code-generation-plan.md. 단계 열이고 체크박스 예순여섯이다.

제품 파일은 행렬 그대로 셋이다 (mcp.go · runner.go · claim.go). claude.go 는 소스 diff 0 이다 — Instrument 가 c.Servers 를 그대로 넘기므로 타입이 바뀌어도 그 줄이 안 바뀐다.

Step 6 을 실측으로 뒀다 — 우리 코드가 쓴 허용목록 파일을 claude 에 물려 init 줄의 mcp_servers 를 읽는다. 함대 없이 실물 하네스로 재는 자리이고, 이름이 목록에 안 나오면 R9 의 값이 틀린 것이므로 거기서 멈추고 묻는다.

Step 7 이 배선을 명시로 잰다 — 스텁 하네스를 쓰는 워커 시험으로 「없는 이름을 요청한 단계가 하네스를 안 띄운다」를 파일 없음으로 잰다. U1 의 구멍(함수만 부르고 배선을 안 쟀다)이 여기서 안 생기게.

FD 에 없던 자리 다섯을 이 계획이 정했다 — 워크스페이스 파일에 크기 상한을 안 둔다 · 항목을 얕은 복사 뒤에 고친다 · 종류를 채우는 자리는 공용 헬퍼다 · Notes 는 Info 이고 메시지가 고정이다 · R8 이 문자열 아닌 command 를 「없다」로 본다."
**Context**: CONSTRUCTION · U4 sources · Code Generation Part 1 완료 · 계획 승인 대기

## CONSTRUCTION — U4 sources · Code Generation Part 2 (실행)
**Timestamp**: 2026-09-13T07:54:50Z
**User Input**: "승인"
**AI Response**: "계획 열 단계를 돌렸고 체크박스 예순여섯이 전부 [x] 다.

제품 파일 셋이다 — mcp.go(Components.Servers 타입 · resolveComponents 의 걸음 여섯 · readWorkspaceMCP · ensureType · copyEntry · hasText · wantedMCP · entryNames · notAvailable) · runner.go(Job 의 필드 넷과 Notes 로그) · claim.go(읽는 조건 둘과 리터럴 네 줄). claude.go 는 계획의 예상대로 소스 diff 0 이다. 새 시험 파일 하나(resolve_test.go)와 worker_unix_test.go 의 배선 시험 둘, 그리고 타입 변경으로 확정 빨강이던 옛 시험 셋을 고쳤다.

CA0 이 전부 초록이다 — go test ./... 18 패키지 · 커버리지 미달 0(전체 6961/7974 = 87.3% · internal/enode 1865/2158 = 86.4%, U3 뒤 86.0% 에서 올랐다) · 스킵 0 · vet · gofmt · U+2605 전수 grep 0 · glyphscan 104 파일 · 크로스 빌드 셋 · 심볼 상한(tls 1 · http 6) · 라우트 26 · node --test 75 · 워킹트리 청결. 커버리지 실행이 바꾼 cmd/enodectl/probe.lock 은 되돌렸다.

Step 6 의 실측을 제품 코드가 낸 파일로 했다. resolveComponents 와 writeMCPAllowlist 가 쓴 mcp.json 을 claude 2.1.266 에 물리니 넷(probe · gerrit · probe3 · wsremote)이 전부 init 줄의 mcp_servers 에 섰다. R9(종류 채우기)를 안 넣었으면 gerrit — ADR-035 §4.4 의 예시 그대로 적은 원격 선언 — 이 빠진 채로 초록이었다. 워크스페이스 항목의 headers 도 한 글자도 안 바뀌고 살아 나갔다.

CA4 의 셋째 줄은 코드로 닫혔다 — 스텁 하네스가 마커 파일을 안 남기는 것으로 「안 떴다」를 잰다. 앞 두 줄은 같은 배선 시험이 그 단계가 실제로 쓴 허용목록을 $OUT 으로 받아 읽지만, 실 함대의 init 줄은 집행자의 몫이다.

변이 여섯을 돌려 다 빨개졌다. 구멍 0 이고 셋은 순수 시험과 배선 시험이 함께 빨개졌다.

실측이 계획의 문장 둘을 고쳤다 — 「종류 오류 하나가 파일 전체를 죽인다」는 원문(map)으로 받는 지금 판에서 안 일어나고(답 2=A 가 덤으로 닫았다. R8 이 요청 시점에 문장으로 거절한다), features.md 3.2 에는 종류가 이미 있었다. FD 의 한 줄도 고쳤다 — 종류를 채우는 자리는 allowlistEntry 가 아니라 합친 뒤의 공용 자리다.

회차 밖으로 낸 것은 decisions.md 6절의 실측 행 ㉓ 와 2절의 종류 줄, features.md 3.2 의 두 줄, 파일 행렬의 다섯 자리다. scene-gates.md 는 안 고쳤다.

넘기는 것 다섯이다. 무거운 셋: 원격 노드 선언만으로는 인증이 안 실린다(credential 은 파일에 안 나가고 원격은 헤더로 인증한다 — 정본 개정 후보다) · 깨진 워크스페이스 파일이 MCP 를 쓰는 단계를 전부 죽인다(답 4=A 의 대가) · Notes 가 노드 로그에만 남아 봉인을 읽는 사람은 겹침과 빠짐을 못 본다."
**Context**: CONSTRUCTION · U4 sources · Code Generation 완료 · CA0 초록 · CA4 사람 대기

## CONSTRUCTION — U4 sources · CA4 실측 (합성 함대)
**Timestamp**: 2026-09-13T08:23:30Z
**User Input**: "ca4진행"
**AI Response**: "scene-gates.md 3절의 CA4 명령을 합성 함대로 그대로 돌렸다. Mediator 를 시험 DB(enode_ca4) 위 127.0.0.1:18080 에 띄우고, 노드 하나를 mcp: probe · probe2 선언과 워크스페이스 .mcp.json(probe3)으로 세웠다. 하네스는 실물 claude 2.1.266 이다.

광고부터 맞았다 — runctl capabilities 에 mcp.probe · mcp.probe2 · harness.claude 가 나오고 mcp.probe3 은 없다 (워크스페이스는 광고에 안 실린다).

세 줄이 다 초록이다. ① agent.mcp: [probe] -> init 의 mcp_servers 가 probe 하나다 (probe2 도 probe3 도 없다). ② agent.mcp: [probe3] -> probe3 이 나타난다. 워크스페이스에서 옮겨 적혔고 광고에는 여전히 없다. ③ agent.mcp: [nope] -> 단계 FAILED 이고 result.error 가 'harness: harness_error mcp server nope is not available on this node' 이며 logs/01-work.log 이 0 바이트다 — 하네스가 안 떴다.

노드 로그가 Notes 를 실물로 냈다 — ①과 ③ 에서 'workspace .mcp.json declares probe3, which this step did not request' 한 줄씩이고 ②에서는 안 난다.

이 실측은 게이트의 서명이 아니다. scene-gates.md 2절 머리가 집행자를 이 유닛을 구현하지 않은 사람으로 못 박는다 — 한 것은 그 사람이 밟을 자리를 미리 밟아 위험을 던 것이다. 함대는 내렸고 임시 디렉터리만 남는다."
**Context**: CONSTRUCTION · U4 sources · CA4 세 줄 실측 초록 · 집행자 서명과 병합 대기

## CONSTRUCTION — U5 pack · Functional Design (Part 1 · 계획과 물음)
**Timestamp**: 2026-09-13T13:40:00Z
**User Input**: "u5 시작"
**AI Response**: "회차의 마지막 유닛 U5 pack 을 착수했다. 브랜치 unit/pack 을 main 에서 땄다 — U1 ~ U4 가 PR #33 ~ #36 으로 전부 main 에 있어 U4 처럼 앞 유닛 브랜치에서 딸 이유가 없다.

계획을 짓기 전에 만지는 자리를 코드로 읽고 실물 claude 2.1.270 으로 다섯을 쟀다. 무거운 것이 첫째다 — 오늘의 플래그(--setting-sources \"\")로는 가짜 홈의 skills/ 와 agents/ 를 하네스가 아예 안 읽는다. 파일은 생기는데 init 줄의 skills 에도 agents 에도 slash_commands 에도 안 나타난다. unit-of-work.md 5절의 'Instrument 가 <dir>/home/skills/ 를 편다' 가 펴기만 하고 안 읽히는 자리였고, 그 상태로는 CA5 가 확정 빨강이며 빨간 이유가 팩 코드가 아니라 플래그다.

길 둘을 실측으로 세웠다. ① --setting-sources user 로 바꾸면 hello 와 helper 가 뜨고, 사람의 ~/.claude/skills 수십과 ~/.claude/agents 둘은 하나도 안 샌다 — CLAUDE_CONFIG_DIR 이 user 범위를 통째로 옮기기 때문이다. ② --plugin-dir <계장>/pack 은 --setting-sources \"\" 를 그대로 둔 채 실린다. .claude-plugin/plugin.json 없이도 실렸고 이름이 pack:hello 로 namespace 된다.

셋을 더 쟀다. 워크스페이스 .claude/skills/ 는 어느 경우에도 안 읽힌다 — features.md 5절과 decisions.md 2절의 열린 미정이 '안 읽는다. 팩으로 나른다' 로 닫힌다. --plugin-dir 에 없는 경로를 주면 종료코드 0 에 stderr 도 없이 조용히 무시된다 — 빠짐을 우리가 잡아야 한다. 그리고 가짜 홈의 settings.json 이 user 설정과 --settings 로 두 번 실려도 Stop 훅은 한 번만 뛴다.

계획 파일을 construction/plans/pack-functional-design-plan.md 에 냈다. 물음 여덟이고 권장은 전부 A 다 — 전달 경로(--plugin-dir 대 --setting-sources user) · 팩 mcp.json 의 어휘 · $IN 을 누가 여나 · PackLimits 의 값 · gzip 수용 · 노드 선언 충돌의 범위 · 기록을 채우는 시점 · 오케스트레이터가 agent.pack 을 붙이는 단계."
**Context**: CONSTRUCTION · U5 pack · Functional Design Part 1 · 사용자 답 대기

---

## CONSTRUCTION — U5 pack · Functional Design (Part 2 · 산출물)
**Timestamp**: 2026-09-14T00:20:00Z
**User Input**: "4는 64mb 로 하고 1은 세팅소스 관련 언급이 이번 회차에서 수십개던데 이거 뭔데" / "B가 아닌 a가 권장인 이유는" / "64메가 이외 모두 a로 한다."
**AI Response**: "물음 4 를 A 로 닫되 MaxBytes 를 64 MiB 로 올렸고, --setting-sources 가 이 회차에 왜 그렇게 자주 나오는지를 셌다 — 저장소에서 예순여덟 자리이고 이 회차 것이 쉰둘이다. 이 팩의 출발점이 그 플래그의 실측표(features.md 1.2 「오늘 코드가 거꾸로 막는다」)이고, U1 이 그것을 hook.go 의 반환에서 Instrument 의 무조건 플래그로 옮겼으며, U4 는 그 플래그가 워크스페이스 .mcp.json 을 끊기 때문에 존재한다. 물음 1 에서 A 를 권한 이유 넷을 적었다 — 닫는 방식이 값이냐 조건이냐 · CLAUDE_CONFIG_DIR 하나가 두 겹을 겸하게 되는 것 · 팩 없는 단계의 argv 가 안 바뀌어 CA1 의 서명이 사는 것 · 이 팩이 MCP 에서 이미 고른 규율(범위가 아니라 지목)과 같은 모양인 것.

사용자가 「64메가 이외 모두 a로 한다」로 답 여덟을 닫았다. 계획 파일의 [Answer] 여덟을 채우고 산출물 셋을 냈다 — construction/pack/functional-design/ 의 domain-entities · business-rules · business-logic-model.

형식에서 설계 원안 둘을 고쳤다. PackFile.Mode 를 뺐다 (규약 안이 전부 글자라 실행 비트가 할 일이 0 이고, tar 의 모드를 받으면 팩이 0777 파일을 계장에 남긴다 — 파일 0600 · 디렉터리 0700 으로 우리가 정한다). Pack.MCP 를 map[string]MCPServer 에서 원문 map 으로 올렸다 (답 2=A · U4 와 같은 판단).

규칙 서른다섯을 번호로 적었고 오류 문구를 계약으로 못 박았다. 무거운 것 셋 — R2 는 agent.pack 이 이름을 적었는데 $IN 에 그 파일이 없으면 거절이다(ADR-058 의 「없는 입력은 값이다」의 예외를 이름으로 적었다). R16 은 실을 것이 0 인 팩을 거절한다(하네스가 없는 플러그인 디렉터리를 종료코드 0 으로 조용히 무시하므로 빠짐을 우리가 잡는다). R8 은 64 MiB 를 받은 바이트와 푼 바이트 둘 다에 건다.

걸음의 순서에서 뜻을 가지는 자리 넷을 적었다 — 팩 오류 검사가 요청 조기 반환 앞이라 깨진 팩이 agent.mcp 가 비어도 단계를 죽이고, 팩 담기가 그 앞이라 요청 0 에도 스킬은 펴지며(CA5 의 둘째 줄), 출처 순회에서 팩을 가장 먼저 보는 것은 우선순위가 아니라 노드 선언 충돌을 놓치지 않기 위해서이고, readPack 은 종류를 이름보다 먼저 본다.

상태 표의 낡은 값을 고쳤다 — U1 ~ U4 가 전부 병합됨(PR #33 ~ #36)이고, 병합됨이 게이트 서명이 아니라는 줄을 함께 달았다."
**Context**: CONSTRUCTION · U5 pack · Functional Design 승인 · 다음은 Code Generation

---

## CONSTRUCTION — U5 pack · Code Generation (Part 1 · 계획)
**Timestamp**: 2026-09-14T05:10:00Z
**User Input**: "이제 코드 생성하자"
**AI Response**: "Functional Design 셋과 만지는 자리 여섯을 코드로 읽고 Code Generation 계획을 냈다 — construction/plans/pack-code-generation-plan.md. 단계 열셋이고, 제품 파일 여섯에 새 제품 파일은 0 이며 새 시험 파일이 하나다(pack_test.go).

이 계획이 FD 에 없던 자리 다섯을 정했다. ① readPack 이 팩 이름을 인자로 받는다 — R3 과 R16 의 문구가 팩 이름을 담는데 바깥에서 감싸면 게이트가 찾는 글자가 그대로 안 나온다. ② R4 의 종류 이름 다섯을 글자로 못 박는다(symlink · hard link · device · fifo · unknown entry type). ③ --plugin-dir 을 등호 형태로 붙인다 — Instrument 의 ③ 뒤에 --strict-mcp-config 가 따라붙는데 띄어 쓴 형태가 가변인자면 그것을 삼킨다. 실측 1.2 는 공백 형태로 쟀으므로 Step 9 가 등호 형태를 다시 잰다. ④ planPrompt 가 더하는 두 줄의 영어 글자를 못 박고 이름을 설정에서 끌어온다. ⑤ PackConfig.Name 의 글자를 LoadConfig 가 안 검증한다 — blob 이름의 정본은 contract.Validate 다.

커버리지 기준선을 쟀다 — internal/enode 가 오늘 84.5% 다. 하한이 80% 이고 이 유닛이 새 코드가 가장 많아 pack_test.go 의 표가 그 하한을 지는 자리다.

갈래를 안 나눈다 — resolveComponents 의 인자가 하나 늘어 runner.go 의 호출과 resolve_test.go 의 호출 전부가 같은 시점에 컴파일된다."
**Context**: CONSTRUCTION · U5 pack · Code Generation Part 1 · 계획 승인 대기

---

## CONSTRUCTION — U5 pack · Code Generation (Part 1 승인)
**Timestamp**: 2026-09-14T05:25:00Z
**User Input**: "승인"
**AI Response**: "계획을 승인받고 Part 2 로 간다. Step 1 부터 순서대로 짓고 각 단계를 마치는 즉시 계획의 체크박스를 [x] 로 바꾼다."
**Context**: CONSTRUCTION · U5 pack · Code Generation Part 1 승인 · Part 2 착수

---

## CONSTRUCTION — U5 pack · Code Generation (Part 2 · 산출물)
**Timestamp**: 2026-09-14T06:40:00Z
**User Input**: "승인"
**AI Response**: "계획의 단계 열셋을 그대로 돌렸다. 제품 파일 여섯 · 새 제품 파일 0 · 새 시험 파일 하나(pack_test.go)다. claim.go 는 소스 diff 가 0 이다 — 답 3=A 가 가장자리를 runner.go 로 정했기 때문이고, 계획 5절의 예상 그대로다.

readPack 을 순수 함수로 지었다. 받은 바이트를 세면서 sha256 에 흘리고, 1f 8b 면 gzip 을 한 겹 벗기고, 항목마다 종류를 이름보다 먼저 보고, tar 가 끝난 뒤 남은 바이트를 끝까지 읽어 해시를 마친다. 마지막 걸음을 빠뜨리면 봉인에 남은 값으로 blob 을 대조할 수 없다 — 변이 ① 이 그것을 잰다.

실측 하나를 더 했다. 우리 코드가 지은 <계장>/pack 을 claude 2.1.270 이 --plugin-dir= 로 읽는다 — skills 에 pack:hello · agents 에 pack:helper · slash_commands 에 pack:hello · plugins 에 {name: pack, source: pack@inline} · mcp_servers 에 probe4(failed). 이 단계가 새로 잰 것은 등호 형태다. FD 의 실측은 공백 형태였고, Instrument 의 ③ 자리 바로 뒤에 --strict-mcp-config 가 따라붙으므로 가변인자면 그것을 삼킨다 (ADR-034 §3 ② 의 함정). 안 삼켰다. 그리고 샌 것이 0 이다 — 이 기계의 ~/.claude/skills 수십과 agents 둘이 하나도 안 나타났다.

변이 일곱을 넣어 전부 빨개지는 것을 봤다. 다섯 · 여섯 · 일곱은 순수 시험과 배선 시험이 함께 빨개졌다 — U1 에서 「함수만 부르고 배선을 안 쟀다」로 났던 구멍이 여기서도 안 생긴다.

CA0 전부 초록이다. 표준 커버리지 명령으로 미달 0 이고 internal/enode 가 87.0% 다 (U4 뒤 86.4% 에서 올랐다). 라우트 26 그대로 · 스킵 0 · 크로스 빌드 셋 · 심볼 상한 crypto/tls=1 net/http=6 · node --test 75 통과 · 워킹트리 청결(probe.lock 되돌림).

계획에 없던 것을 둘 더했고 그것을 계획과 요약에 이름으로 적었다. ① command 도 url 도 없는 팩 항목을 요청하면 거절한다 — U4 가 워크스페이스 출처에서 같은 실패를 이미 거절했고(6절 ㉓ ②), 종류를 못 정하는 항목을 하네스가 말없이 뺀다는 실측이 출처를 안 가린다. 안 더하면 이 유닛이 그 침묵을 팩 경로로 되살린다. ② Notes 복사를 걸음 0 의 오류보다 앞으로 옮겼다 — 거기서 곧바로 반환하면 규약 밖 항목의 이름이 거절과 함께 사라지고, 그것은 U4 의 R11 이 막은 자리다.

글자 하나를 계획에서 바꿨다 — 모르는 tar 종류의 이름을 계획은 unknown entry type 으로 적었고 코드는 tar type '<글자>' 로 갔다. 문구가 pack entry %s is a %s 라 is a 뒤에 놓으면 영어가 안 된다. 계획 4절 ② 를 그 값으로 고쳤다.

실측이 계획의 문장 하나를 고쳤다 — 0 바이트 blob 은 「tar 가 아니다」가 아니라 「실을 것이 0」이다. Go 의 tar 가 빈 스트림을 빈 아카이브로 읽기 때문이고, 둘 다 치명이며 이 문구가 curl 이 빈 파일을 남긴 경우를 더 곧게 가리킨다.

회차 밖으로 여덟을 실었다 — decisions.md 6절 ㉔ 와 2절 두 행 · features.md 5절과 3.6 · unit-of-work.md 5절 · scene-gates.md CA5 의 표와 명령 · component-methods.md 2.1 · 파일 행렬 넷. component-methods.md 의 Components.Servers 는 U4 가 남긴 낡은 줄이었고 같은 코드 블록이라 함께 고쳤다.

CA5 와 CA6 은 안 닫혔다. 이 단계가 잰 것은 우리 코드가 지은 디렉터리를 하네스가 읽나이고, CA5 는 노드와 Mediator 를 지나 그 팩이 실제로 실리나다. 집행자는 이 유닛을 구현하지 않은 사람이고 눈 검증을 보류로 안 넘긴다."
**Context**: CONSTRUCTION · U5 pack · Code Generation 완료 · CA5 · CA6 대기

---

## GATE — CA1 실측 (합성 함대 · OAuth 절반)
**Timestamp**: 2026-09-14T02:52:15Z
**User Input**: "Ca1부터 진행해"
**AI Response**: "scene-gates.md 3절의 CA1 명령을 합성 함대로 돌렸다. 시험 Postgres 위에 전용 DB enode_ca1 을 파고 Mediator 를 127.0.0.1:18080 에, 노드 하나를 그 위에 세웠다. 하네스는 실물 claude 2.1.270 이고 이 기계의 진짜 HOME 으로 돈다 — 그것이 CA1 이 요구하는 조건이다.

전제를 먼저 봤다. claude mcp list 가 개인 서버 일곱을 낸다 — claude.ai 계정 커넥터 넷(Expedia · Google Drive · Gmail · Google Calendar)과 개인 stdio 셋(office · playwright · pencil)이고 전부 Connected 다. auth status 는 loggedIn: true · authMethod: claude.ai 다.

runctl example agent 를 그대로 냈다 (agent.mcp 가 없는 예시다). run_id 를 ca1-oauth-1 로 바꿔 제출했고 SUCCEEDED 다. runctl record ca1-oauth-1 -o r.tar 로 푼 logs/01-summarize.log 의 첫 줄이 판정 재료다.

init 줄의 mcp_servers 가 [] 다. 개인 서버 일곱이 전부 끊겼다. slash_commands 는 내장뿐이고 agents 도 내장 다섯뿐이다 — 사람의 ~/.claude/agents 둘(aidlc-verify · aidlc-xhigh)이 안 샜다.

같은 단계가 Not logged in 없이 돌았다. reason ok · turns 4 · $0.1002545 · version 2.1.270 이고 로그에 Not logged in 도 /login 도 0 건이며 summary.json 이 실제 워크스페이스를 읽고 나왔다. 두 겹이 동시에 선 것이다 — 가짜 홈이 개인 설정을 끊었는데도 .credentials.json 복사가 로그인을 살렸다. 둘 중 하나만 되면 이 줄이 안 나온다.

덤으로 U5 의 omitempty 가 실물에서 확인됐다 — steps/01-summarize.json 의 result.harness 에 mcp 키도 pack 키도 없다. 팩도 서버도 없는 단계의 봉인이 오늘과 한 글자도 안 달라진다는 값이 실제 Run Record 에서 선다.

게이트웨이 절반은 안 돌렸다. 이 기계의 ~/.claude/settings.json 에 apiKeyHelper 도 env 도 없고 ANTHROPIC_BASE_URL 도 AUTH_TOKEN 도 API_KEY 도 안 잡혀 있다 — gatewayAuthFields() 가 옮길 필드가 0 이라 그 경로가 애초에 발동하지 않는다. 가짜 게이트웨이로 돌리면 API 호출에서 죽어서 「Not logged in 없이 돈다」를 못 잰다. 그건 초록이 아니라 다른 실패이므로 안 돌렸고, 사용자가 사내에서 확인하기로 했다.

이 실측은 게이트의 서명이 아니다. scene-gates.md 2절 머리가 집행자를 이 유닛을 구현하지 않은 사람으로 못 박고 회차는 진행자로 닫았다 — 한 것은 그 사람이 밟을 자리를 미리 밟아 위험을 던 것이다. 함대는 내렸고 DB 와 임시 디렉터리만 남는다."
**Context**: GATE · CA1 OAuth 절반 초록 · 게이트웨이 절반은 사내 확인 대기

---

## GATE — CA0 ~ CA6 집행 (진행자 서명)
**Timestamp**: 2026-09-15T00:00:00Z
**User Input**: "잠시 돌아가자. Ca 0 ~ ca6 까지 확인 필요하다고 했지? 회사에서 unit/pack 브랜치로 확인했다.

ca0 초록.
ca1 초록. Oauth 절반 + 게이트웨이 절반
ca2 초록. 광고에 실림. 실행파일 없으면 빠지고 사유 남음. 놓으면 5분 뒤 복귀
ca3 초록. 422(없는 키), 400(모르는 에이전트 키), 400(mcp 문자열), lint, dry-run
ca4 초록. 셋 중 요청한 하나만 열림. 워크스페이스 것도 열림. 없는 이름은 하네스 안 뜨고 그 사유로 실패
ca5 초록. pack:hello가 skills/slash_commands에. probe4 failed. 다이제스트 일치
ca6 초록. omab(내가만든 사내 mcp) connected. 다이제스트 blob과 일치."
**AI Response**: "진행자가 사내에서 unit/pack 브랜치로 일곱을 전부 초록으로 서명했다. 집행자는 진행자다 (unit-of-work-dependency.md 8.1). 이것이 서명이고, 그 앞의 합성 함대 실측들은 미리 밟은 자리였다.

무거운 것 셋을 기록에 남긴다.

CA1 의 게이트웨이 절반이 닫혔다. 이 기계에서는 ~/.claude/settings.json 에 apiKeyHelper 도 env 도 없어 gatewayAuthFields() 가 옮길 필드가 0 이라 못 쟀고, 사내 기계가 그 절반을 냈다. 가짜 홈으로 옮겨도 게이트웨이 인증이 사는 경로가 그 두 필드뿐이라는 설계가 실물에서 선다.

CA6 이 실물 MCP 로 초록이다. 스크래치 게이트들은 가짜 서버(true)를 써서 mcp_servers 에 failed 로 나타나는 것이 판정 재료였는데, 사내에서는 omab 이 connected 다 — 뜨기만 하는 것이 아니라 MCP 로 답한다. 그리고 harness.pack 의 다이제스트가 같은 Run 의 blob 과 맞았다. U5 가 SHA256 을 받은 바이트의 것으로 정한 이유(봉인을 읽는 사람이 blob 을 받아 sha256sum 으로 맞출 수 있어야 한다)가 실물 대조로 확인된 것이다.

CA2 에 값 하나가 더 붙었다 — 실행파일을 되돌리면 5분 뒤에 광고로 복귀한다. 그것이 -detect-every 의 기본값이다. 내가 합성 함대에서 잰 것은 15초로 줄인 값이었고, 사내 측정이 기본값의 실제 지연을 냈다.

남은 것은 병합이다. CONVENTIONS.md 3.3 이 「그 유닛의 장면 게이트가 초록인 뒤」를 병합 지점으로 못 박았고 이제 그 조건이 섰다."
**Context**: GATE · CA0 ~ CA6 전부 초록 · 진행자 서명 · unit/pack 병합 가능

---

## GATE — CA2 의 화면 검증 S1 (진행자 서명)
**Timestamp**: 2026-09-15T00:20:00Z
**User Input**: "Mediator 상세 패널에 mcp서버 omab 있음 으로 표시 확인. 노드 제어판에서는 탐지 능력 (읽기 전용) 칸에 mcp.omab=1 이라고 나타남."
**AI Response**: "CA2 의 마지막 줄이 닫혔다. 표가 CA2 를 「(S1 포함)」으로 적고 2.1 절이 「노드 카드에 mcp.<이름> 칩이 있다. 제어판 탐지 능력 카드에도 있다」로 두 화면을 요구하는데, 진행자가 사내 함대에서 둘 다 봤다 — 현황판은 상세 패널에 「MCP 서버 · omab 있음」이고 제어판은 탐지 능력(읽기 전용) 칸에 mcp.omab=1 이다.

칩이 사는 자리를 이름으로 적어 둔다. 현황판에서 그 칩은 **노드 상세 패널**에 있고 함대 격자의 카드 면에는 없다. 카드 면은 identity.mjs 의 nodeIdentity() 가 고정 키 목록으로 짓는데(owner · hostname · os · host_arch · device_type · sandbox · provider · harness · tts · device · board · instance) 그 목록에 mcp.* 가 없다. U3 의 답 7=B 가 만든 attributeName · attributeValue 는 view.mjs 의 상세 패널에서만 돈다. format.test.mjs 의 단언도 그 함수를 직접 부르므로 카드 면을 안 잰다.

그래서 S1 은 **상세 패널로 초록**이다. 카드 면에 올리는 것은 identity.mjs 를 고치는 일이고 이 회차의 파일 행렬 밖이다 — 다음 회차가 정할 자리로 남긴다.

이로써 CA0 ~ CA6 이 하위 줄까지 전부 닫혔다."
**Context**: GATE · CA2 의 S1 초록 · 회차의 게이트가 전부 닫혔다

---

## GATE 이후 — 어휘 발견 가능성 관찰 (진행자 손기록)
**Timestamp**: 2026-09-15T00:40:00Z
**User Input**: "그전에 관찰한 문제점이 있어 손으로 타이핑 해 둔다. 속도를 위해 영어로 썼다.

# Finding - runctl alone is not enough for an agent with no context

# what was tested
A fresh agent was given an empty directory containing only ./runctl, forbidden to read source, and told to make a Windows node install an MCP server named omab and then prove a later step sees it in its session.

## result
It succeeded. It reached the correct three-step contract (install -> mkpack -> prove) and found agent.pack, agent.mcp and in.from unaided, from unctl's own help, example and schema output.

It took 26+ minutes and about 24 runs.

## where the runs went
1 to 2 : ligitimate   Is the node alive, does an agent step run
18 to 19 : doc gaps    15 of them guessing the pack tar layout

The layout guesses, each one a submitted run: mcp.json with a top-level server name, .mcp.json with an mcpServers key, mcp/omab.json, a pack with skills only, a pack with agents only. It also guessed device:\"led\" as a requires key, probed $OUT, CWD and USERPROFILE on the real machine to learn the run-step contract, and hunted through d:\\images\\.claude looking for where MCP config lives.

Only aobut 2 mins of the 26+ was work that documentation could not have removed.

## the refusals were not the problem
R16 returned \"pack p.pack carries no skills, agents, or mcp servers\", which names the three convention slots. The other failure was \"mcp server omab is not available on this node\". the agent was reading error text and narrowing, not guessing blindly. THe messages are good.

What is missing sits one layer below them: the file name mcp.json , and the fact that servers go under an mcpServers key inside it. The error says which three slot exist; it does not say what the mcp slot is called on disk.

## what the gap costs
20 failed runs sealed into the record, every one of them on the real board-attached windows machine, tokens spent on each harness start. The product currently teaches its own conventions by letting an agent measure them on live hardware.

## Fixes, in order of value
1. Add `runctl example pack`. One example shows the tar layout and both keys (agent.pack, in.from) at once. `example mcp` already exists and is precisely why CA6 verdict 1 was cheap to reach - the example was copied and edited. Thepack path had no such starting point.
2. Put the pack tar layout in `runctl schema`: mcp.json carrying an mcpServers object, skills/<name>/SKILL.md, agents/<name>.md. Include the trap that `tar -cf x.tar -C dir .` is rejected with \"pack entry ./ has an unsafe name\" because of the ./ entry; use `tar -cf x.tar skills mcp.json`.
3. Put the $OUT  convention in `runctl schema`: a run step's cwd is the workspace, and the harvest reads $OUT. This is stated in the plan prompt that orchestrators receive but not in schema, so a contract author writing by hand has to discover it by running a probe.
4. Point at `runctl capabilities` as the source of requires attribute values.The help calls it \"attribute vocabulary of the fleet\", which does not make clear those are the values you write into requires.
Items 1 and 2 alone remove the 15 layout runs.

## what already works well
`runctl lint` warned \"no condition judges step mkpack; it can fail and the run sitll succeeds\" and printed the JSON fragment to add . Both 400 refusals name the allowed fields and end with \"see: runctl lint <contract.json> / runctl example\""
**AI Response**: "네 구멍이 오늘 코드에 실제로 있는지 먼저 셌다. 넷 다 있다.

runctl example 의 목록이 여섯이고 pack 이 없다 (agent · command · demo-led-toggle · demo-welcome-audio · mcp · multi). runctl schema 의 네 섹션 전체에서 mcpServers 가 0 회이고 $OUT 도 0 회다. schema steps 는 agent 를 map[string]any 한 줄로만 찍으므로 agent.pack 도 안 보인다 — 6절 ㉑ 이 이미 적은 그 자리다.

./ 함정도 실물로 재현했다. tar -cf x.tar -C dir . 이 첫 항목으로 ./ 를 낸다. U5 의 packEntryName 이 path.Clean(\"./\") == \".\" 을 R7 으로 거절하므로 문구가 정확히 pack entry ./ has an unsafe name 이다 — 관찰한 그대로다.

이 관찰의 무게는 실패가 아니라 비용이다. 게이트는 전부 초록이고 거절 문구도 제 일을 했다 — 에이전트가 문구를 읽고 좁혀 갔다. 없는 것은 그 한 층 아래다: 슬롯 이름은 알려 주는데 그 슬롯이 디스크에서 무엇으로 불리는지를 안 알려 준다. 그래서 스무 번의 실패 Run 이 실물 보드가 달린 윈도우 기계에 봉인됐다.

decisions.md 6절에 실측 행 ㉕ 로 적었다 — 회차가 어휘를 지었는데 그 어휘를 보이게 하는 면이 안 늘었다는 사실이 값이다. features.md 4절에 이월 넷을 이름으로 더했고, 1 과 2 만으로 열다섯 번이 사라진다는 관찰의 판정을 그대로 실었다.

이 회차에서 안 고친다 — cmd/runctl 의 소스 diff 0 이 이 회차의 파일 행렬이 센 값이고(CA0 의 라우트 수와 같은 자리), 지금 고치면 서명이 끝난 게이트 아래의 코드를 건드리는 것이 된다. 다음 회차의 입력으로 넘긴다."
**Context**: GATE 이후 · 어휘 발견 가능성 · decisions ㉕ 와 features 4절 이월로 넘긴다

---

## 회차 밖 — 어휘를 보이게 하는 넷 (㉕ 의 고침)
**Timestamp**: 2026-09-15T02:00:00Z
**User Input**: "맥락없는 에이전트 도움말 강화는 여기서 바로 수정해줘"
**AI Response**: "브랜치 fix/runctl-shows-the-pack-vocabulary 에서 넷을 다 고쳤다. 회차는 PR #37 로 main 에 닫혔고 게이트가 서명된 뒤라 회차 브랜치에 안 실었다.

1 runctl example pack — internal/contract/examples/pack.json. tar 를 짓는 명령 단계 하나와 두 키를 문 에이전트 단계 하나다. 명령이 mktemp 아래에 skills/hello/SKILL.md · agents/helper.md · mcp.json 을 짓고 tar -cf $OUT/pack -C $d skills agents mcp.json 으로 묶는다 — 배치가 argv 에 글자로 보이고 안전한 tar 꼴도 함께 보인다. 워크스페이스에 아무것도 안 남긴다. 팩의 서버 이름을 packed 로 지었다: probe 로 지으면 노드가 그 이름을 선언한 함대에서 R19(팩이 노드 선언을 덮는다)로 죽는다. 그리고 requires 에 mcp.<이름> 을 안 적었다 — 팩이 정의의 출처이므로 노드가 그 이름을 선언할 필요가 없다는 것이 이 예시가 가르치는 값이다.

2 runctl schema pack — 슬롯 셋의 디스크 배치와 mcpServers 키, 두 키가 함께 있어야 한다는 것, tar -cf x.tar -C dir . 이 ./ 항목 때문에 pack entry ./ has an unsafe name 으로 거절된다는 함정, 팩의 스킬이 pack:<이름> 으로 namespace 된다는 것.

3 runctl schema io — cwd · $OUT · $IN. schema steps 의 꼬리가 그것을 가리킨다. 한 자리에만 적고 나머지는 가리킨다 — 두 벌로 두면 갈린다.

4 requires 값의 출처 — schema requires 의 꼬리와 help 의 capabilities 줄이 runctl capabilities 를 가리킨다. 옛 문구 attribute vocabulary of the fleet 가 그 값이 requires 에 적는 값이라는 것을 안 말했다.

cmdSchema 를 글 절이 들어갈 수 있게 고쳤다 — typ 이 nil 이면 글이고, 필드 뒤에 붙는 꼬리를 따로 둔다. 구조체에서 뽑는 것만으로는 안 보이는 값이 있다는 것이 ㉕ 의 내용이다.

시험이 그 글자를 센다. TestSchema_CarriesWhatTheErrorMessagesDoNotSay 가 mcp.json · mcpServers · skills/<name>/SKILL.md · agents/<name>.md · 두 키 · 함정 문구 · runctl example pack 을 이름으로 찾고, io 와 requires 와 steps 의 꼬리도 잰다. TestExamplePack_CarriesBothKeys 는 예시의 agent.pack 과 앞 단계의 out 과 in.from 이 같은 이름인지를 잰다 — 하나만 적으면 그 단계는 팩 없이 돌고 우리가 못 잡는다. 변이 둘을 넣어 빨개지는 것을 봤다 (mcpServers 를 지운다 · steps 의 꼬리를 걷는다).

CA0 전부 초록이다. cmd/runctl 커버리지가 93.9% 에서 94.3% 로 올랐고 라우트 26 그대로다.

features.md 4절의 이월 블록을 「닫혔다」로 고치고 원문을 아래에 남겼다. decisions.md ㉕ 의 근거 칸에도 고친 사실을 달았다."
**Context**: 회차 밖 · ㉕ 의 넷을 고쳤다 · CA0 초록

---
## W-a Functional Design Part 1 — U1 · U3 병렬
**Timestamp**: 2026-09-15T08:50:00Z
**User Input**: (없음 — 「병렬로 돌릴 수 있으면 돌려」의 실행)
**AI Response**: 웨이브 W-a 의 유닛 둘을 **worktree 로 갈라 동시에** 돌렸다 — `unit/transcript` 와 `unit/progress-store` 가 각자 `/home/sunny/enode-wt/` 아래 자기 트리를 쓴다. 산출물은 계획 둘이다 (`transcript-functional-design-plan.md` 673줄 · 물음 열둘 / `progress-store-functional-design-plan.md` 660줄 · 물음 아홉). 커밋 `edda804` · `ac3acac`.

**둘 다 계획을 짓기 전에 코드를 읽었고 갈린 자리를 여덟씩 찾았다.** 진행자가 표본을 코드에 대고 검증했다 — U1 의 1.1(옮기는 것이 셋이 아니라 일곱 + 타입 둘)과 1.2(`map[string]json.RawMessage`), U3 의 1.2(`AppendLog` 는 `n == limit` · 같은 파일의 `WriteBlob` 은 `limit+1` 과 `n > limit`)와 1.4(D1 이 인용한 `record.go:196` 은 `unseal` 안이다)와 1.7(`internal/record` 에 `sync` 임포트 0). **다섯 다 맞다.**

**가장 무거운 것 셋** — ① U1 이 옮길 범위가 문서의 셋에서 **일곱 + 타입 둘**로 는다 (`components.md` · `unit-of-work.md` 가 함께 틀렸다) ② `go list -deps` 가 시험 임포트를 안 보므로 경계 검사를 그 명령으로만 세우면 시험이 남의 패키지를 임포트해도 초록이다 ③ N2 의 오늘 값이 「상한이 없다」가 아니라 **「지우는 코드가 0」**이고 구멍 셋을 코드로 짚었다.

**병렬이 재작업을 안 냈다** — 파일 겹침 0 이 실제로 0 이었고 두 worktree 의 `git status` 가 각자 한 줄이다.
**Context**: CONSTRUCTION — W-a Functional Design Part 1. 답 대기 (물음 스물하나)

---

## W-a Functional Design Part 1 — 답
**Timestamp**: 2026-09-15T09:05:00Z
**User Input**: "권장대로"
**AI Response**: 물음 스물하나를 전부 권장값 **A** 로 닫았다 (U1 열둘 · U3 아홉). 밖으로 번지는 셋을 답 앞에 명시로 냈고 사용자가 그것을 보고 골랐다 — U1 Q7 = A 가 `map[string]json.RawMessage` 를 유지해 SECURITY-13 의 「아는 키만 읽는다」를 지키고, U3 Q2 = A 가 D4 의 상한을 `AppendProgress` 에만 걸어 봉인 기록의 거동을 안 바꾸고, **U3 Q5 = A 가 `internal/store` 에 고아 쓸기를 들여 파일 행렬 4절의 「`internal/store` diff 0」과 품질 게이트 2 를 함께 건드린다.** Part 2 를 유닛 둘에서 병렬로 연다 — 답을 계획에 박고, 맞물리는 쌍(U1 의 Q2-Q3 · Q1-Q5 · Q7-Q8)에 모순 검사를 먼저 돌린 뒤, 산출물 셋씩을 짓는다.
**Context**: CONSTRUCTION — W-a Functional Design Part 1 답. Part 2 착수

---

## W-a Functional Design Part 2 — 모순 검사가 막는 것을 찾았다
**Timestamp**: 2026-09-15T09:40:00Z
**User Input**: "상한이 뭔데" / 물음 10 = **B** (NDJSON 표시 줄) · 물음 11 = **B** (지운다)
**AI Response**: U3 의 모순 검사가 **답끼리의 모순은 0 인데 막는 것 하나**를 찾아 산출물을 안 짓고 멈췄다 — `Q3 = A`(상한 앞 마지막 개행까지)와 `Q4 = A`(파일에 표시 줄 없음)를 합치면 **진행 파일 안에 상한 도달의 흔적이 0** 이고 `ReadProgress` 의 겉면에 `limit` 이 없어 **GET 이 「닿아서 멈춘 것」과 「아직 작은 것」을 못 가른다.** NC-4 와 US-7 이 그 자리에서 죽는다. Part 1 에서는 안 보였다 — 답이 없어 조합이 없었다.

**검사가 되레 하나를 지웠다** — 1.5 의 구멍 ③(`s.Records == nil` 이면 봉인 0)은 없다. `api.go:50` 이 `records: st.Records` 로 같은 포인터를 들고 `needRecords` 가 nil 이면 503 을 내므로 **진행 트리가 애초에 안 생긴다.** N2 의 구멍은 셋이 아니라 **둘**이다 (`reap.go:126` 의 한 시간 창 · `:104` 의 `n > 0`).

**사용자가 값을 묻고 답했다** — 「상한이 뭔데」에 진행자가 코드로 쟀다: `MaxBlobBytes` 기본 **10 MiB** (`config.go:100` 의 `10 << 20`), yaml `max_blob_bytes` 로 바뀐다. **닿는 일이 드물다**는 것을 보고 골랐다 — 「반쪽 줄」의 대가는 찬 경로에서만 무는 반면 표시 줄은 **NC-4 가 요구한 「응답과 화면」 둘을 한 출처에서** 세운다. 진행자가 시점 사실 하나를 보탰다 — 계획이 B 의 대가로 적은 「U1 이 이미 병합됐을 수 있다」가 **지금은 거짓**이라 되여는 값이 문서 셋 한 번이다.

**표시 줄의 모양은 선례를 그대로 쓴다** — `runner.go:419` 의 `elidedMark` 가 `type` 에 점을 넣고 그 근거(실측 하네스 `type` 이 전부 홑단어라 안 부딪친다)를 주석이 적어 뒀다. `enode.capped` 에 그대로 선다.

**11 = B 가 긴장 하나를 닫았다** — 새 시도가 앞 시도 파일을 걷으므로 D7 의 보안 credit(노출 기간이 시도 하나)이 그대로 서고, `application-design.md` 8절과 `requirements.md` 5.4 잔여 ① 을 **안 고친다**.

**U1 은 Part 2 를 모순 0 으로 닫았다** (산출물 셋 · R1 ~ R15 · 체크박스 27 전부 닫힘). 10 = B 때문에 **U1 을 한 번 되연다** — `Kind` 를 하나 늘린다. 찍는 쪽(U3)과 읽는 쪽(U1)에 규칙을 갈라 걸어 두 벌로 안 두게 했다.

**이 회차가 배운 것 하나** — **유닛 경계를 넘는 답의 조합은 한 유닛의 모순 검사가 못 본다.** U1 과 U3 각자의 검사는 둘 다 모순 0 이었고, 막는 것은 U3 의 답이 U1 의 표면을 건드리는 자리에서 났다.
**Context**: CONSTRUCTION — W-a Functional Design Part 2. 물음 10 · 11 답 뒤 재개

---

## W-a 교차 검사 — 유닛 둘이 같은 필드를 다르게 정의했다
**Timestamp**: 2026-09-15T10:20:00Z
**User Input**: (없음 — 진행자가 웨이브를 닫기 전에 돌린 검사)
**AI Response**: U1 과 U3 이 각자 Part 2 를 닫은 뒤, **유닛 사이를 대 보는 검사**를 돌렸다. 앞 항목이 「유닛 경계를 넘는 조합은 한 유닛의 모순 검사가 못 본다」고 적은 그 자리다. **실제로 하나 났다.**

```text
   U3 (찍는 쪽)   R12 — {"type":"enode.capped","bytes":<상한>}
   U1 (읽는 쪽)   Bytes int64 // 상한에 닿은 시점의 총 길이
```

U3 의 R11 이 상한 앞 마지막 개행까지만 쓰고 R13 이 표시 줄을 총 길이에 넣으므로 **총 길이와 상한이 다른 값**이다. 그대로 두면 화면이 틀린 숫자를 그리고 NC-4 가 거짓 수치와 함께 선다.

**상한이 맞고, 취향이 아니라 기계로 갈린다** — ① **자기참조**다. R13 이 「표시 줄도 총 길이에 든다」로 못 박았으므로 **표시 줄은 자기가 든 총 길이를 담을 수 없다.** 쓰는 순간 그 값이 바뀐다 ② 총 길이는 `Progress.Total` 이 이미 따로 나른다. 표시 줄이 또 실으면 두 벌이 된다 ③ 같은 파일의 기존 잘림 표시 `... log truncated at %d bytes` 의 `%d` 가 limit 이다 (`record.go:72`). U1 에 고치라고 보냈다 — `int64` 결론은 유지하고 근거를 바꾼다.

**두 유닛 다 자기 검사에서 모순 0 이었다.** 어느 쪽도 자기 눈으로는 못 봤고, **웨이브를 닫는 자리에서 진행자가 대 보고서야 보였다.** U1 이 Part 2 에서 「한 패키지 시험으로 못 잡는 첫 자리」로 그 위험을 이름으로 적어 뒀는데, 그 예측이 같은 웨이브 안에서 실증됐다.

**U3 의 Functional Design 을 커밋했다** — `d20c26c`. 산출물 셋 · 규칙 서른여섯 · 교차 참조를 기계로 세어 빈 것과 뜬 것이 0 · 체크박스 25 전부 닫힘 · 열린 물음 0.
**Context**: CONSTRUCTION — W-a 교차 검사. U1 수정 대기

---

## W-a Functional Design — 승인
**Timestamp**: 2026-09-15T10:45:00Z
**User Input**: "승인"
**AI Response**: W-a 의 Functional Design 을 닫았다. 유닛 둘 · 물음 스물셋 전부 닫힘 · 산출물 여섯 · 커밋 `edda804` · `ac3acac` · `d20c26c` · `adc93d0` · `e200ec1`. 다음은 **NFR Requirements** 이고 유닛 둘 다 돈다 (회차 계획이 「새 표면을 만드는 유닛만」으로 걸었고 U1 은 파서가 · U3 은 진행 파일이 새 표면이다). **N2 의 숫자가 U3 에서 채워진다** — 회차 상태 파일이 「① 에 상한이 없다」로 남긴 미결이고, Functional Design 이 규칙의 모양(R24 의 주기와 나이)까지만 짓고 숫자를 여기로 넘겼다.
**Context**: CONSTRUCTION — W-a Functional Design 승인. NFR Requirements 착수

---

## W-a NFR Requirements Part 1 — 답
**Timestamp**: 2026-09-15T11:40:00Z
**User Input**: "권장대로"
**AI Response**: 물음 일곱을 전부 권장값 **A** 로 닫았다 (U1 셋 · U3 넷). **SECURITY 미준수가 둘 다 0 이다** — U1 준수 7 · N/A 8 / U3 준수 7 · **조건부 3** · N/A 5. U3 가 조건부를 준수로 안 칠한 것이 옳다: 그중 둘(01 암호화 · 14 경보)이 이 물음의 답에 걸려 있었다.

**실측이 무거운 것 셋을 냈다** — ① **출력 상한이 아무 데도 없다.** 입력 상한(링 512 KiB · 파일 10 MiB)은 있는데 사건 배열의 상한이 0 이라 10 MiB 입력이 사건 약 499,000 · `Event` 배열 약 112 MB 가 된다. 한 줄 길이 상한도 0 이고 파서 경로에 `bufio.Scanner` 가 0 이다 ② **N2 는 Run 을 못 묶는다** — 되묻기 대기가 무한이 설계다 (`contract.go:130` · ADR-032). 파일의 수명을 따로 재야 한다. 노드가 죽은 Run 은 임대로 이미 240초 안에 닫힌다 ③ **쓸기에 새 타이머가 0 이다** — `RunReaper` 가 이미 `RenewSeconds` 60초로 돌고 `reap.go:158` 이 `time.NewTimer(0)` 이라 기동에 한 번 먼저 돈다. `sealExpired` 밖(`Reap()` 안)으로 자리만 옮기면 FD 가 짚은 구멍 둘째(`reap.go:104` 의 `n > 0`)가 닫힌다.

**U1 이 자기가 FD 에서 만든 결함을 찾았다.** 팩 `decisions.md`:50 은 「입력 요약 **200자** · 앞 **500자**」인데 FD 의 R8.1 이 「200 **바이트** · 500 **바이트**」로 적고는 「값은 팩의 것이고 이 유닛이 안 고른다」를 달았다. **한글은 3바이트라 3배 다른 값이고**, 바이트로 자르면 룬이 쪼개져 `encoding/json` 이 오류 없이 `U+FFFD` 로 바꾼다 (`encode.go:1040-1042`) — 파서의 「고치지 않는다」를 그 자리에서 어긴다. 진행자가 코드와 두 문서에 대고 확인했다.

**Q3 = A 가 회차의 미결 하나를 닫는다** — 「① 에 상한이 없다」로 들고 온 자리에 **6시간**이 섰다.

**진행자가 표본 검증한 것** — `RenewSeconds` 기본 60 · `NotAfterFactor` 3 · `reap.go:158` 의 `NewTimer(0)` · `internal/enode/testdata` 가 **없다** · `freeBytes` 는 노드 쪽(`detect.go:334`)에만 있다 · 팩과 FD 의 단위 갈림. **여섯 다 맞다.**

**U3 가 못 잰 것을 못 잰다고 적었다** — 단계 하나가 실제로 내는 바이트. 저장소에 stream-json testdata 가 0 이라 못 잰다. **링의 512 KiB 를 근거로 안 썼다** — 그것은 판단이지 측정이 아니다. 대신 상한 봉투(R=10 · S=20 이면 2 GiB)로 댔다.

커밋 `ab819f5` (U1) · `bd8cc0f` (U3). Part 2 를 병렬로 연다.
**Context**: CONSTRUCTION — W-a NFR Requirements Part 1 답. Part 2 착수

---

## W-a NFR Requirements Part 2 — N2 가 값을 얻었다
**Timestamp**: 2026-09-15T12:30:00Z
**User Input**: (없음 — 「권장대로」의 실행)
**AI Response**: 유닛 둘이 Part 2 를 닫았다. **모순 0 · SECURITY 미준수 0 · 빈 칸 0.** 커밋 `5e9219b` (U3) · `8fc8886` (U1) · 회차와 팩이 `b58906f`.

**N2 의 값 여섯이 섰다.** 핵심은 **무엇의 나이인가**이고 U3 이 그것을 못 박았다 — **진행 파일의 마지막 쓰기**다. Run 이나 단계의 시작이 아니다. 노출은 바이트가 앉아 있는 기간이고 그것은 마지막으로 바이트가 는 순간부터 센다. **끝난 단계의 파일이 여기서 걸린다.**

**봉투를 다시 계산했더니 기대와 달랐다 — 이것이 이 단계가 낸 가장 정직한 값이다.** 「6시간이 자르는 것은 노출 시간이고 디스크가 아니다.」 Run 이 6시간보다 오래 돌 때만 디스크가 주므로 **최악 10 GiB 가 그대로 서 있고 Q4 = A 가 그것을 안 본다.** 없앤 것이 아니라 받아들인 것이라 규칙으로 적었다.

**U3 가 자기 판정을 스스로 교정했다** — Part 1 이 SECURITY-14 를 조건부로 적었는데, 규칙 전문을 읽으니 **넷 중 디스크 용량을 말하는 줄이 0 이다** (보안 사건 경보 · 로그 무결성 · 보존 기간 · 대시보드). 진행자가 규칙 파일에 대고 확인했고 맞다. 상한을 안 만드는 것이 14 위반이 아니고 디스크 잔여는 규칙 밖의 운영 자리다. **SECURITY-01 은 준수로 안 굳혔다** — 6시간이 닫은 것은 「기간에 값이 없다」이고 「평문이다」는 그대로다. 「보상 통제가 완성됐다고 준수로 칠하면 그 표가 심사의 근거가 될 때 거짓이 된다」.

**U1 이 자기가 만든 갈림을 자기 문서에 기록했다** — `business-rules.md` 8.1 이 팩(09-11)과 FD(09-15)의 날짜를 나란히 적고 「나중이 이것이다」를 못 박는다. 그리고 「값은 팩의 것이고 이 유닛이 안 고른다」를 **숫자는 팩 · 단위는 이 유닛**으로 갈라 적었다. **불변식의 범위도 스스로 좁혔다** — 「잘린 `Text` 가 올바른 UTF-8」을 모든 `Text` 로 적으면 거짓이다 (`plain` · `raw` 는 상한이 없어 애초에 안 잘린다). 조건이 `Cut > 0` 에 걸리고 그것이 곧 「출처가 JSON 이다」와 같다.

**교차 검사 — 이번엔 0 이다.** U3 의 NFR 이 파서 표면을 안 건드린다 (`grep` 으로 확인). 앞 웨이브의 두 번과 달리 겹치는 값이 없었다.

**진행자가 친 것** — 팩 `decisions.md`:50 의 「200자 · 앞 500자」를 「200바이트 · 앞 500바이트 · 룬 경계」로 · 회차 `requirements.md` 의 같은 자리와 5.4 잔여 ① 에 6시간 · 회차 `aidlc-state.md` 의 「① 에 상한이 없다」를 값과 대가로 교체. **전부 유닛이 글자를 정확히 적어 준 것을 그대로 쳤다** — 짐작으로 친 것이 0 이다.
**Context**: CONSTRUCTION — W-a NFR Requirements Part 2. 승인 대기

---

## W-a NFR Requirements — 승인
**Timestamp**: 2026-09-15T12:45:00Z
**User Input**: "승인"
**AI Response**: W-a 의 NFR Requirements 를 닫았다. 다음은 **NFR Design** 이고 회차 계획이 「유닛마다 · 최소 · 패턴 다섯」으로 걸었다. 앞 회차(`obs`)는 이 단계를 SKIP 했고 그때 남은 「어떻게」를 Code Generation 계획이 졌다 — 이 회차는 그 자리를 여기서 진다.
**Context**: CONSTRUCTION — W-a NFR Requirements 승인. NFR Design 착수

---

## W-a NFR Design — 물음 0
**Timestamp**: 2026-09-15T13:20:00Z
**User Input**: (없음 — 승인 뒤 자동 진행)
**AI Response**: 유닛 둘이 **물음 0** 으로 산출물까지 한 번에 냈다. 커밋 `5cf1571` (U1) · `ef4d766` (U3). 회차 계획이 이 단계를 **패턴 다섯만**으로 좁혔고 W-a 가 지는 것이 셋이다 — U1 이 ④(파서가 아는 키만 읽는 규율) · U3 가 ①의 찍는 쪽과 ⑤(봉인 때의 순서). **안 지는 것도 이름으로 적게 걸었고 둘 다 그렇게 했다** (「이 유닛이 안 한다」이지 「아무도 안 한다」가 아니다).

**U1 이 ④ 를 「형식」이 아니라 「무엇이 그것을 지키나」로 닫았다** — 규율이 깨지는 길 일곱 중 **여섯을 기계가 막고 하나(`Fields` 를 `map[string]any` 로 되돌리는 것)는 사람이 막는다.** 그 칸에 「사람」이라고 적은 것이 `CLAUDE.md` 의 「도는 스크립트가 없는 채로 기계 검사라고 적으면 그것이 거짓 초록이다」와 같은 결이다. 그리고 **규율에 난 유일한 이음매**를 박았다 — `enode.elided` 는 짓는 쪽이 같은 패키지라 왕복 시험이 잡지만 **`enode.capped` 는 못 잡는다.** 이론이 아니라는 증거가 이 회차 안에 있다. **「퍼즈만으로 SECURITY-13 을 닫았다고 적으면 거짓 초록」** — 재는 것이 넷이고 퍼즈가 둘이다.

**U3 가 설계로 순서 하나를 새로 더했다 — 개행 보장(R43).** 없으면 표시 줄이 반쪽 줄에 붙어 한 줄이 되고 파싱이 깨져 **`Capped` 가 영영 참이 안 되고 표시 줄이 무한히 쌓인다.** 개행 하나가 무한 루프와 `raw` 사건 하나를 가른다. 중간에 죽는 틈 넷이 전부 수렴하는 것도 표로 보였다 — 대가가 「크래시 한 번에 깨진 조각 하나」이고 무한이 아니다.

**U3 의 실측이 회차 계획을 교정했다** — `execution-plan.md` 가 「진행 파일 지우기가 **tar 보다 앞**」으로 적었는데 **`Seal` 은 tar 를 안 짓는다.** `seal(d)` 의 chmod 로 끝나고 (`record.go:120`) `Tar` 는 `GET /record` 때 호출자 하나에서 돈다 (`api.go:1043`). 진행자가 코드로 확인했고 맞다 — 막아야 할 것은 0444 · 0555 다. 그 자리를 고쳤다.

**진행자의 오류 하나** — 지시에서 산출물 이름을 `nfr-design.md` 로 줬는데 규칙 Step 6 은 `nfr-design-patterns.md` 다. 둘 다 규칙 이름으로 고쳤다.
**Context**: CONSTRUCTION — W-a NFR Design. 승인 대기

---

## W-a NFR Design — 승인
**Timestamp**: 2026-09-15T13:35:00Z
**User Input**: "승인한다. 코드 쓰자"
**AI Response**: W-a 의 NFR Design 을 닫았다. 다음은 **Code Generation** 이고 규칙이 Part 1(계획) · Part 2(생성)로 가른다. 사용자의 「코드 쓰자」는 단계 착수의 지시이고, Part 2 는 계획 승인 뒤에 연다. 유닛 둘의 계획을 worktree 둘에서 병렬로 짓는다 (`unit/transcript` · `unit/progress-store`).
**Context**: CONSTRUCTION — W-a NFR Design 승인. Code Generation Part 1 착수

---

## W-a Code Generation Part 1 — 계획 둘
**Timestamp**: 2026-09-15T14:05:00Z
**User Input**: (없음 — 승인 뒤 자동 진행)
**AI Response**: 유닛 둘이 worktree 둘에서 병렬로 계획을 지었다. 커밋 `9d907da` · `c4e6eec` (U1) · `f962d26` · `53ef404` (U3). **Step 열아홉씩 · 체크박스 아흔여섯(U1)과 백스물둘(U3) · 코드 0 줄.**

**진행자가 표본 검증을 돌렸고 유닛마다 하나씩 되돌렸다.**

**U1 이 가장 무거운 것을 찾았다 — 불변식 F2 가 거짓이다.** 「`tool_use` 와 `tool_result` 의 `Text` 는 JSON 을 거쳐 오므로 이미 올바른 UTF-8」이 `tool_result` 에만 참이다. 진행자가 `go1.26.6` 으로 직접 재현했다 — `json.Unmarshal` 로 `string` 을 받는 길만 `U+FFFD` 로 바꾸고 **`json.Marshal(json.RawMessage)` 는 `0xff` 를 그대로 낸다.** `tool_use` 의 `Text` 는 도구 입력 객체를 다시 마샬한 것이라 `compact` 만 지난다. 고치는 길 셋 중 파서를 고치는 것은 마스킹 금지 위반이고 `any` 로 푸는 것은 R2.1 위반이라, **F2 에 「입력이 올바른 UTF-8 이면」 조건을 단다.** 값은 한 글자도 안 바뀐다. 안 고치면 Part 2 가 첫 변이에서 멈추고 그때 고칠 자리가 시험이 아니라 불변식의 정의다.

**되돌린 것 ① U1 의 셈** — 1.2 의 표에 함수가 여덟인데 글자가 「일곱」이었다. 고치는 과정에서 **틀린 값이 두 벌로 있던 것**이 드러났다 — `domain-entities.md` 1절은 제목이 「함수 일곱」인데 **그 절의 표가 이미 열 행**이다. 회차 문서의 「셋」을 고치며 한 번 고친 값인데 고친 값도 하나 모자랐고, 계획이 그 제목을 옮기다 자기 표와 어긋났다. 재발을 막는 줄을 넣었다 — **개수를 글자로 세지 않고 1.2 의 표 행이 정본이다.**

**되돌린 것 ② U3 의 게이트 7** — 스물한 자리를 통째로 위반으로 적었다. **여섯은 위반이 아니다** — `cmd/iapadapter` 의 것은 GitHub 코멘트 본문을 짓는 문자열 리터럴이라 실제로 렌더링되고, 그 자리를 다스리는 것은 `CONVENTIONS.md` **1.1** 이며 1.1 이 고르라고 한 수단이 곧 굵게다. 통째로 위반으로 적어 두면 다음 사람이 그 여섯을 지우고 그 순간 규약 1.1 을 어긴다. **위반은 열다섯 · 파일 여덟 · 전부 주석 안이다.**

**U3 가 진행자의 셈을 되돌렸다** — 진행자가 「여덟 중 일곱이 행렬 밖」으로 적었는데 **여덟 전부가 밖이다.** 계획 1.1 의 표에 그 여덟이 0 개다. 유닛이 짐작으로 따르지 않고 다시 세서 고쳤다. 다만 둘(`record_test.go` · `reap_deterministic_test.go`)이 **U3 이 여는 패키지 안**이라는 사실은 따로 남겼다 — 가져갈지는 진행자 판단이고 가져가면 게이트 2 의 `internal/store` diff 가 더 벌어진다.

**U3 가 제외 규칙의 한계도 적었다** — 경로 제외는 뭉툭해서 그 패키지의 주석에 새로 들어온 두 별표를 놓친다. **정확한 규칙은 자리다** — 주석 안이면 1.2 위반, 렌더링되는 문자열이면 1.1. `glyphscan.go` 가 문자열과 rune 리터럴을 보고 주석을 안 보므로 이 검사기는 그 반대를 봐야 하고, 같은 AST 순회에 얹으면 두 벌이 안 된다.

**착수 시점 기준선을 진행자가 실측했다** — `go build` · `go vet` · `go test ./...` 전부 초록. 단 postgres 가 서야 한다 (`cmd/mediator` · `internal/api` · `internal/store` 셋이 URL 없이 실패). `eval "$(scripts/testdb.sh)"` 로 `enode-test-pg` 가 55434 에 섰다.
**Context**: CONSTRUCTION — W-a Code Generation Part 1. 승인 대기

---

## W-a Code Generation Part 1 — 승인
**Timestamp**: 2026-09-15T14:20:00Z
**User Input**: "넘어가지"
**AI Response**: 계획 둘을 승인으로 받고 Part 2 를 연다. **물음 하나(U1 의 `tool_result` 배열 꼴)에 답이 안 얹혔으므로 계획이 적은 순서를 그대로 간다** — Step 2 가 실물 하네스(`claude 2.1.271`)를 돌려 실측하고, 답이 나오면 그것이 이기고 못 돌리면 A 로 짓되 규칙 번호를 비워 둔다. 짐작한 값을 규칙으로 굳히지 않는다.

Part 2 도 worktree 둘에서 병렬이다. U1 은 불변식 F2 의 글자를 먼저 고치고 시작한다 — 안 고치면 퍼즈가 첫 변이에서 멈춘다.
**Context**: CONSTRUCTION — W-a Code Generation Part 1 승인. Part 2 착수

---

## W-a Code Generation Part 2 — 구독 만료로 강제 종료
**Timestamp**: 2026-09-16T00:10:00Z
**User Input**: "구독만료로 작업 강제종료되었음 재개할 수 있는 지점으로부터 시작"
**AI Response**: 유닛 둘이 Part 2 를 돌던 중 API 가 403 (`oauth_org_not_allowed`) 으로 끊겨 둘 다 중간에 죽었다. 진행자가 **재개 지점을 코드로 실측했다** — 에이전트의 말이 아니라 저장소의 상태로 쟀다.

```text
   U1   커밋 10fd7a2 까지 (불변식 F2 의 조건).  internal/transcript/ 넷이
        커밋 안 된 채로 790 줄 — transcript.go · line.go · shell.go · parse.go
        공개 심볼 열다섯이 다 섰다.  build · vet · test 초록
        **runner.go 는 아직 안 덜어냈다** (금지 셋이 그대로 있다).  시험 0
        testdata/lines 가 빈 디렉터리다 — 죽은 자리가 Step 9 의 픽스처다

   U3   커밋 0.  제품 셋 수정(record.go · seal.go · reap.go · 102 줄) +
        progress.go 703 줄 + progress_test.go 804 줄이 커밋 안 된 채로 있다
        build · vet 초록 · internal/store 시험 초록
        **internal/record 가 빨갛다** — TestTheFourGapsConverge 의
        before_the_cut_write 하나.  capped 가 거짓이고 몸통에 표시 줄이 0 이다
        틈 넷 중 하나가 안 수렴한다 — 죽은 자리가 그 시험을 짓던 중이다
```

**둘 다 계획의 체크박스를 하나도 안 채웠다** — 규칙의 「일을 끝낸 그 자리에서 `[x]` 로 고친다」를 안 지켰다. 그래서 계획으로는 진척을 못 재고 코드로 쟀다. 재개할 때 그 자리도 함께 고치게 건다.

**커밋 안 된 것을 안 버린다** — U1 790 줄 · U3 1,507 줄 + 수정 102 줄이 전부 살아 있고 빌드가 선다. 재개는 이 위에서 이어 간다.
**Context**: CONSTRUCTION — W-a Code Generation Part 2 중단과 재개 지점 실측

---

## W-a Code Generation Part 2 — 유닛 둘이 코드를 냈다
**Timestamp**: 2026-09-16T01:05:00Z
**User Input**: (없음 — 재개 지시의 실행)
**AI Response**: 끊긴 자리에서 이어 둘 다 닫았다. **진행자가 게이트를 다시 쟀고 초록이다.**

```text
   U1   커밋 넷 — c95d5bf (internal/transcript 넷 + 시험 다섯 + 픽스처 18) ·
        583ebf6 (runner.go 에서 368 줄 덜어냄) · 0510edb (경계 검사) ·
        864d5e0 (요약과 체크박스).  체크박스 97 중 94
   U3   커밋 다섯 — bdb13a3 · 26bf7b4 · efc0042 · ebb7d7a · 73fdeb3
        체크박스 122 중 121
```

**진행자가 직접 잰 값** — 전체 시험이 둘 다 초록(18 · 19 패키지) · 커버리지 하한 80% **미달 0** (U1 전체 87.5% · `transcript` 94.4% · `enode` 86.7%. U3 전체 87.4% · `record` 84.4% · `store` 82.6%) · 라우트 **17 그대로** · `gofmt` 빔 · diff 0 이어야 하는 경로 전부 0 · 파일 교집합 **0**.

**커버리지를 처음에 잘못 쟀다** — `-cover` 단독으로 재니 `store` 가 37.7% 로 나왔다. `.coverage-contract.yml` 이 측정 명령을 `-coverpkg=./... -count=1` 로 못 박았고 **그 명령이 아닌 값은 게이트 입력이 아니다.** 정본 명령으로 다시 쟀다.

**이음매를 진행자가 실제로 왕복시켰다** — W-a 가 「병렬의 대가」로 남긴 자리다. U3 의 `cappedMark` 를 글자 그대로 옮겨 마샬한 41 바이트를 U1 의 `Parse` 에 먹였다. **U1 이 그것을 `KindCapped` 로 읽고 `Info.Bytes` 에 10485760 을 얻었다.** 짓는 쪽과 읽는 쪽이 한 패키지에 없어 어느 유닛의 시험도 못 잡는 자리였고, 그것을 웨이브를 닫는 자리에서 쟀다.

**진행자가 U1 의 산출물에서 고친 것 둘** — ① `logs_events_test.go:17` 의 주석에 장식 문자가 새로 들어왔다 (규약 1.2 위반). 에이전트가 「새로 넣은 것 0」이라 보고했으나 하나 있었다. 문장을 앞으로 옮겨 고쳤다 ② `cmd/enodectl/probe.lock` 이 시험을 돌 때마다 바뀌어 diff 0 게이트를 깬다. 되돌렸다 — 이 유닛이 만든 것이 아니고 앞선 회차들이 이미 겪은 자리다.

**U1 이 커밋을 안 했다** — 자기 운영 규칙이 `git commit` 을 금한다고 했다. 진행자가 계획의 갈래대로 넷으로 쌓았다.

**실측이 계획을 뒤집은 것 셋.** ① **물음 1 의 답이 실측으로 나왔다** — 실물 하네스(`claude 2.1.271`)를 돌리니 `tool_result` 의 `content` 가 **배열로 온다.** 계획은 「기록이 0」이라 적었는데 이미지를 읽는 턴에서 실제로 왔고, 그 배열에 `text` 원소가 0 이라 A 와 B 가 같은 값을 낸다. **A 로 지었고 짐작이 아니라 실측이 받친다.** ② **U3 의 빨간 시험은 시험이 틀렸다** — 씨앗 5 바이트 + 청크 5 바이트 = 10 이라 상한 20 에 닿은 적이 없다. 닿은 적이 없으니 `Capped` 가 거짓인 것이 맞는 답이다. **제품 코드 0 줄을 고쳤다.** ③ **최악 봉투가 112 MB 가 아니라 2.3 GB 다** — 그 값은 가장 짧은 JSON 줄을 가정하는데 개행만 든 1 바이트 줄도 사건 하나다. 21배 차이다.

**변이가 시험의 구멍 둘을 찾았다** (U1) — F2 는 `Cut > 0` 을 내는 시드가 0 이라 전제가 한 번도 안 섰고, 되돌림은 줄기 둘 중 껍데기만 밟아 원문 줄기를 지워도 초록이었다. **둘 다 시험만 고쳤고 제품 코드는 안 바꿨다.** U3 의 변이 여덟도 전부 빨개졌다.

**초록이라고 안 적은 것** — `internal/api` 가 전체 스위트 아홉 번 중 둘에서 빨갰다 (deadlock 40P01). U3 가 재현 경로를 못 잡았고 **못 갈랐다고 적었다.** 이 유닛이 scratch DB 를 여덟 개 더 만드는 것이 의심 자리다. 진행자의 측정에서는 안 났다.
**Context**: CONSTRUCTION — W-a Code Generation Part 2. 승인 대기

---


## W-a Code Generation Part 2 승인 — 그리고 병합
**Timestamp**: 2026-09-16T01:08:12Z
**User Input**: "병합하고 w-b 가자"
**AI Response**: 승인으로 받았다 (Continue to Next Stage). **병합 전에 합친 나무에서 게이트를 다시 쟀다** — 유닛 둘의 값은 각자의 브랜치에서 잰 것이고, 합치면 같은 값이라는 보장이 없다.

```text
   합친 나무   origin/main + v3-run-transcript + unit/transcript + unit/progress-store
               충돌 0.  세 갈래가 만지는 파일의 교집합이 0 이다

   build       초록        vet 초록        gofmt 빔
   test        19 패키지 전부 초록 · **스킵 0**
   커버리지     정본 명령(-coverpkg=./... -count=1)으로 19 패키지 · **미달 0**
               전체 87.5% · transcript 94.4% · record 84.4% · store 82.5% ·
               enode 86.7% · 가장 얇은 자리는 build 16/20 = 80.0% 그대로
   라우트       17 그대로.  W-a 는 라우트를 안 늘린다 (18 은 U4 의 몫 · CB0)
   크로스 빌드   windows/amd64 초록 · net/http T 6 · crypto/tls T 1 (상한 50 · 10)
   glyphscan   109 파일 · 장식 문자 0
```

**앞 항목이 「못 갈랐다」고 적은 자리를 다시 쟀다** — `internal/api` 의 deadlock(40P01). 합친 나무에서 전체 스위트 두 번 + `api`·`store`·`record` 다섯 번, **일곱 번 다 초록이고 재현 0 이다.** 재현이 0 인 것은 없어진 것과 다르므로 사라졌다고 안 적는다. 병합을 막지는 않는다.

**`cmd/enodectl/probe.lock` 이 또 바뀐다** — 시험이 추적 파일을 고치므로 `git status --porcelain` 이 빈다는 CP0 조건을 그대로 두면 깨진다. 이 유닛들이 만든 것이 아니다 (둘 다 `cmd/enodectl` 을 안 만진다). 되돌렸고 기준선의 성질로 적는다.

**PR 셋으로 `main` 에 올린다** (`CONVENTIONS.md` 3.1 · 3.3). 회차 브랜치를 먼저 올린다 — 유닛 정의와 파일 행렬이 거기 있고 유닛 PR 둘이 그것을 딛는다.
**Context**: CONSTRUCTION — W-a Code Generation Part 2 승인 · 병합 전 합본 게이트 측정

---

## W-b 착수 — Functional Design Part 1 (U2 · U4)
**Timestamp**: 2026-09-16T01:35:00Z
**User Input**: (없음 — "병합하고 w-b 가자" 의 뒷절 실행)
**AI Response**: 브랜치 둘을 **병합된 `main` 에서** 땄다 (`unit/node-stream` · `unit/log-api` · worktree `/home/sunny/enode-wt/`). 회차 브랜치가 아니라 `main` 인 이유는 둘 다 W-a 의 코드를 딛기 때문이다 — U2 는 U1 이 `runner.go` 에서 덜어낸 자리에 tee 를 잇고, U4 는 U1 의 파서와 U3 의 진행 파일을 둘 다 부른다.

**둘 다 계획을 짓기 전에 코드를 읽었다** (W-a 가 배운 것). 문서가 코드와 갈린 자리를 **다섯** 찾았다.

```text
   U2   ①  component-methods 3절의 「Decode 안에서 io.ReadAll 을 걷는다」만으로는
           사건이 흐르는 시점이 안 바뀐다.  부르는 자리가 cmd.Run 뒤다 (runner.go:209)
       ②  유닛 정의의 「Decode 도 파서를 안 쓴다」가 거짓이다 — U1 이 selectLogs 를
           옮기면서 internal/enode 가 transcript 를 임포트한다 (runner.go:331)
       ③  lastJSONObject 가 뒤에서부터 전체를 훑는다.  줄 단위로 읽어도 바이트를
           통째로 들어야 한다.  requirements 5.7 의 「전체를 메모리에 다시 담지
           않는다」는 internal/transcript 를 가리키지 Decode 가 아니다

   U4   ④  봉인된 logs/NN-*.log 를 여는 공개 겉면이 internal/record 에 0 이다.
           경로 조립에 필요한 safe() 는 비공개다.  파일 행렬은 record.go 를 U3 에만 줬다
       ⑤  데모 모드의 read 래퍼가 무인증이다 (api.go:82).  requirements 5.4 의
           잔여 ② 는 「토큰 하나라 주체를 못 가른다」로 토큰이 있다는 전제로 쓰였다
```

**CB0 의 앞 값을 합본에서 쟀다** — `grep -c 'mux.HandleFunc' internal/api/api.go` 가 **17**. U4 가 18 로 만든다.

**U3 가 U4 의 물음을 미리 닫아 둔 자리가 넷이다** — R17 · R18 · R19 · R32. 맞춰 보니 `Progress{Total, Attempt, Capped}` 가 헤더 넷 중 셋과 1 대 1 이고 빈 칸이 0 이다. 그 넷은 이 유닛이 다시 안 묻는다.

**물음 열다섯을 냈다** — U2 일곱 · U4 여덟. 계획 커밋 77eb3d2 (U2) · f8c7b74 (U4).
**4.2 에 웨이브를 닫는 교차 검사 자리를 미리 이름으로 적었다** — 사건 종류의 어휘 · 총 길이의 뜻 · 시도 · 상한의 단위. W-a 가 「유닛 경계를 넘는 조합은 어느 한 유닛의 모순 검사도 못 본다」를 배웠고 그것을 계획에 박았다.
**Context**: CONSTRUCTION — W-b Functional Design Part 1. 답 대기

---

## W-b Functional Design — 답 열다섯과 산출물 여섯
**Timestamp**: 2026-09-16T02:05:00Z
**User Input**: "권장대로"
**AI Response**: **물음 열다섯이 전부 A 로 닫혔다.** 권장과 갈린 답이 0 이다. 계획 둘의 2절에 답을 적고 산출물 여섯을 냈다. 커밋 b115b25 (U2) · 0d9f14c (U4).

**모순 검사가 유닛마다 하나씩 막았다.**

```text
   U2   답 1 = A 와 FR-1 을 그대로 합치면 배출이 두 벌이다 — 배출기가 도는 중에
        emit 을 부르고 Decode 가 끝난 뒤에 또 부른다.  한 단계의 사건이 정확히
        두 번 난다.  오늘은 로그가 두 줄이 되는 것으로 끝나지만 **U7 이 그 자리에
        업로더를 물리면 같은 바이트가 두 번 올라간다**
        답 — 흘리는 쪽이 낸다.  runner 가 Decode 에 넘기는 emit 이 no-op 이다.
        시그니처는 안 바뀐다 (R3 그대로)

   U4   답 6 = A (한 응답 1 MiB 상한)와 as=events 를 합치면 폴링이 깨진다 —
        본문이 사건 배열이라 읽는 쪽이 자기가 읽은 바이트를 셀 수 없고 다음
        from 을 못 구한다
        답 — 상한에 걸릴 때 **마지막 개행에서 끊는다.**  그러면 다음 from 이
        언제나 from + len(본문) 이다.  헤더를 다섯째로 안 늘려도 닫힌다
```

**교차 검사를 웨이브를 닫기 전에 돌렸다** (W-a 가 배운 것). **둘 다 각자 모순 0 이었다.**

```text
   모순 0 이나 갈릴 자리 넷   어휘(정본이 하나로 맞다) · 총 길이(같은 단어가 다른 값 —
                          Ring.Total 대 Progress.Total) · 시도(gen 과 attempt 가
                          같은 수가 아니다) · 상한의 단위 셋(512 KiB · 10 MiB · 1 MiB)

   새로 찾은 것 하나        Source: progress 에 Bytes 0 이 나가는 길이 **셋**이다 —
                          ① 아직 안 왔다 (U3 R18) ② 6시간 쓸기가 걷었다 (U3 Reap)
                          ③ Seal 의 DropProgress 와 verdict 굳히기 사이의 창.
                          셋이 한 값으로 나가고 **셋이 다른 문서에 살아 어느 유닛의
                          모순 검사도 못 본다.**  U4 의 잔여 ③ 을 셋으로 넓혔다
```

**회차 밖으로 낼 것이 여섯으로 늘었다** — 파일 행렬 둘(`record.go` 가 U3 하나 · `internal/transcript/**` 가 U1 하나인데 U4 가 둘 다 만진다) · `unit-of-work.md` U2 절의 거짓 한 줄 · `component-methods.md` 셋(Decode 만 고치면 시점이 안 바뀐다 · `transcript()` 네 갈래는 U7 의 것 · 응답 상한과 개행 규칙이 없다) · `requirements.md` 5.4 잔여 ②(데모에는 토큰 전제가 없다).
**Context**: CONSTRUCTION — W-b Functional Design. 승인 대기

---

## W-b — NFR 단계 전부 스킵
**Timestamp**: 2026-09-16T02:20:00Z
**User Input**: "nfr 단계를 모두 스킵하고 다음으로 간다."
**AI Response**: 스킵으로 받았다. **회차 계획과 갈린다** — `execution-plan.md` 는 NFR Requirements 와 NFR Design 을 「EXECUTE (유닛마다 · 최소)」로 걸었고 U4 가 **N1 을 지도록** 배정돼 있었다. 사용자가 그것을 걷었고 **어긋남으로 적는다** (준수라고 안 적는다).

**스킵이 안 닫힌 채로 남기는 값이 둘이다.**

```text
   N1    함대 규모에서의 청크 PUT 과 폴링 부하.  회차의 미결 둘 중 남은 하나다
         (N2 는 W-a 에서 U3 가 닫았다)
   1 MiB  한 응답의 본문 상한.  Functional Design 이 시작값으로 적고
         「NFR Requirements 가 N1 을 재고 굳힌다」로 미뤘다 (U4 잔여 ④)
```

**둘 다 Code Generation 계획이 진다.** 선례가 있다 — v1 의 `obs` 에서 NFR Design 을 스킵했을 때 답이 남긴 「어떻게」를 Code Generation 계획 5절이 졌다 (`aidlc-state.md` 의 회차 v1 절). 같은 모양으로 값과 거는 자리를 계획이 적는다.

**U2 는 원래 스킵이었다** — 회차 계획이 「새 표면을 만드는 유닛만 돈다」로 걸었고 U2 는 새 외부 표면이 0 이다. 이 지시로 바뀌는 것은 **U4 뿐이다.**

다음은 **Code Generation Part 1** — 유닛 둘의 계획을 짓는다.
**Context**: CONSTRUCTION — W-b NFR Requirements · NFR Design 스킵

---

## W-b Code Generation Part 1 — 계획 둘
**Timestamp**: 2026-09-16T02:40:00Z
**User Input**: (없음 — "nfr 단계를 모두 스킵하고 다음으로 간다" 의 뒷절 실행)
**AI Response**: 계획 둘을 냈다. 커밋 ec8d7c5 (U2 · Step 열둘 · 체크박스 54) · 24b531f (U4 · Step 열셋 · 체크박스 58). **물음 0 이다** — 갈래는 답 열다섯이 다 닫았다.

**NFR 스킵이 남긴 값 둘을 U4 의 계획 0절이 졌다.**

```text
   1 MiB   한 응답의 본문 상한.  internal/api/log.go 의 **상수**다.
           설정 키를 안 만든다 — 조절 손잡이가 아니라 보호이고, 값을 올릴 수
           있는 것 자체가 보호를 무르는 길이다

   N1     안 닫는다.  대신 **봉투를 산수로 남겼다** — 미는 쪽 S/2 req/s ·
          당기는 쪽 V x C / 2 req/s · 천장은 데모의 전역 한도 120 req/s
```

**N1 이 실제로 아픈 자리를 찾았다** — `ratelimit.go:19-20` 의 전역 한도(초당 120 · 버스트 240)를 **이 폴링 라우트가 `GET /v1/nodes` · `GET /v1/runs` 와 한 바구니에서 나눠 쓴다.** v1 의 `obs` 가 이미 「전역 한도의 대가로 한 클라이언트가 나머지를 굶길 수 있다」로 적은 자리이고, 이 회차가 거기에 폴링을 더한다. **한도는 안 바꾼다** — v1 이 값을 보고 고른 자리를 이 유닛이 되돌리는 것이 된다. 대신 U8 에 규칙 하나를 넘긴다: **보이는 카드만 2초로 폴링한다.**

**계획이 실측으로 찾은 것 둘.**

```text
   U2   internal/enode/harness.go 가 파일 행렬에 없다 — EventKind 와 Event 가
        거기 산다 (:285 · :291).  행렬은 U2 에 runner.go · claude.go · claim.go 만 줬다
   U4   internal/api/log_test.go 가 이미 있다 (PUT 과 tar).  GET 의 시험을
        거기 섞으면 두 시험이 같은 헬퍼를 두고 서로를 고친다 — getlog_test.go 로 가른다
```

**파일 교집합이 여전히 0 이다** — U2 는 `internal/enode` 넷 · U4 는 `internal/api` 둘 + `record.go` + `transcript.go`.
**Context**: CONSTRUCTION — W-b Code Generation Part 1. 승인 대기

---

## W-b Code Generation Part 2 — 유닛 둘이 코드를 냈다
**Timestamp**: 2026-09-16T04:30:00Z
**User Input**: "승인. 넘어가자."
**AI Response**: 둘 다 닫았다. 커밋 0b66780 (U2) · 9685f52 (U4). 체크박스 **U2 54 중 54 · U4 58 중 58**.

**진행자가 합본에서 게이트를 다시 쟀다** — `origin/main` + `unit/node-stream` + `unit/log-api`. 충돌 0 · **파일 교집합 0** · 시험 19 패키지 초록 · **스킵 0** · 커버리지 미달 0 (전체 87.5%) · **라우트 18 (CB0)** · `gofmt` 빔 · glyphscan 111 파일 0 · `internal/store` diff 0.

**이음매를 웨이브 닫는 자리에서 실제로 맞댔다.** 노드가 찍는 종류와 선 위로 나가는 종류가 같은 글자인가를 한 번 돌렸다 (합본에서만 가능하고 커밋하지 않았다).

```text
   양쪽이 같은 여섯    init · text · tool_use · tool_result · capped · result
   capped 이 값이다   찍는 쪽이 U3 · 읽는 쪽이 U1 · 흘리는 쪽이 U2 · 내는 쪽이 U4 다.
                    넷을 지나 같은 글자로 나온다
```

**실측이 계획을 고친 것이 여덟이다 — U2 다섯 · U4 넷** (하나가 겹친다). 무거운 셋.

```text
   U2   Decode 에 no-op 을 넘기면 final 이 사라진다.  final 은 줄에서 나는
        사건이 아니라 봉투를 읽고 나는 판정이라 배출기가 낼 수 없다.  거르개가 답이다
   U2   transcript.Parse 는 개행 없는 줄을 안 읽는다 (쓰는 중에 읽히는 파일을
        위한 규율).  두 자리 다 개행을 붙여 넘긴다.  안 붙이면 사건이 0 이다
   U4   태그를 달 타입이 하나가 아니라 다섯이다.  Server 와 Elided 에는 안 다는
        근거가 주석에 있었는데 **그 관용은 읽을 때만 있다.**  이제 쓰인다
```

**변이 열하나 전부 빨강. 두 번은 값이 있었다.**

```text
   U2 ⑤   대기열의 상한을 빼니 Write 가 영영 막혔다 — 실패 메시지가 아니라
          **멈춤**으로 빨개졌다.  실물에서는 하네스의 stdout 이 그 자리에서 멈춘다
   U4 ④   머리가 잘린 것을 안 말하는 변이가 **처음에는 살아남았다.**  폴링이
          규칙대로 돌면 다음 from 이 언제나 줄 머리라 그 갈래를 한 번도 안 밟는다.
          밟는 길 둘(손으로 준 from · 상한보다 긴 줄)을 재는 시험을 따로 짓고
          나서야 빨개졌다.  안 지었으면 반쪽 줄이 raw 사건으로 화면에 그려진다
```

**사고 하나를 적는다** — U2 의 변이를 `git checkout --` 로 되돌리다 커밋 안 된 `runner.go` 작업을 지웠다. 백업에서 되살렸다. **그 사고가 시험의 구멍을 드러냈다**: 배출기가 통째로 없는데도 수를 세는 시험이 초록이었다 — `Decode` 혼자서 같은 종류를 같은 순서로 내기 때문이다. **수를 세는 시험은 시점을 못 잰다.** 하네스가 첫 사건을 기다리게 하는 시험을 따로 지었고, 그 뒤 변이 ① 이 20초 만에 빨개진다.

**앞 팩의 시험 하나를 뒤집었다** — `TestRunHarness_TheRingGetsNothingWhileTheStreamIsRaw` 가 「링이 한 바이트도 안 받는다」를 재고 있었다 (`decisions.md` 6절 ⑲). 그 결정이 이 회차에서 뒤집혔으므로 시험도 뒤집고 **본문 글자까지 확인하게** 했다.
**Context**: CONSTRUCTION — W-b Code Generation Part 2. 승인 대기

---
## W-b 병합 지시 · W-c 착수 지시
**Timestamp**: 2026-09-16T05:40:00Z
**User Input**: "w-b pr둘고 메인에 올리고 w-c 진행"
**AI Response**: W-b 의 Code Generation Part 2 승인으로 받는다. 병합 전에 합본에서 게이트를 다시 잰다 (W-a 와 같은 규율 — 각 브랜치의 값이 합쳐진 나무에서도 같다는 보장을 안 믿는다). 초록이면 PR 셋을 올린다 — 회차 브랜치 `v3-run-transcript` 먼저, 그 뒤 `unit/node-stream` · `unit/log-api`. 병합 뒤 W-c (U5 `panel-live` · 단독 · CB1) 를 연다.
**Context**: CONSTRUCTION — W-b Code Generation Part 2 승인 · 병합 · W-c 착수

---

## W-b 를 닫는다 — 합본 재측정
**Timestamp**: 2026-09-16T05:52:00Z
**User Input**: (없음 — 위 지시의 실행)
**AI Response**: 합본 `origin/main` + `v3-run-transcript` + `unit/node-stream` + `unit/log-api` 에서 다시 쟀다. **충돌 0 · 파일 교집합 0.**

```text
   build · vet · gofmt 빔 · glyphscan 111 파일 0
   시험 19 패키지 초록 · 스킵 0
   커버리지 정본 awk — 미달 0 · 전체 7722/8824 = 87.5%
   크로스 빌드 windows/amd64 · net/http T 6 · crypto/tls T 1
   CB0 — grep -c 'mux.HandleFunc' internal/api/api.go = 18 (앞 값 17)
```

**패키지 하나가 내려갔다** — `internal/api` 82.9% -> **82.3%**. U4 가 `log.go` 238줄을 들였고 그 갈래 몇이 폴링의 규칙적인 경로에서 안 밟힌다. 하한에서 2.3 포인트 위라 안 막으나 U6 이 같은 패키지를 만지므로 거기서 다시 본다.

**`probe.lock` 이 두 웨이브 연속 같은 자리다** — 기준선의 성질로 확인됐다. **`internal/api` 의 deadlock 은 U4 가 405줄을 더한 뒤에도 재현 0** 이고, 그래도 사라졌다고 안 적는다.

**정본에 갈린 자리 하나를 찾았다** — `unit-of-work.md` 의 U5 절이 NFR 요구를 **스킵**이라 적고 그 근거로 U6 의 문장(「출처만 바뀐다 · 기존 `do()` 를 탄다」)을 댔다. 같은 문서 9절의 표는 U5 를 **돈다**로 적는다. 한 문서 안에서 두 값이다.
**Context**: CONSTRUCTION — W-b 병합 전 합본 측정

---

## W-c 착수 — U5 panel-live Functional Design Part 1
**Timestamp**: 2026-09-16T07:20:00Z
**User Input**: (없음 — 「w-c 진행」의 실행)
**AI Response**: W-b 를 PR #42 · #43 · #44 로 `main` 에 병합했다 (`main` 은 05ee710 · 라우트 18). 브랜치 `unit/panel-live` 를 병합된 `main` 에서 땄다.

**계획을 짓기 전에 코드를 읽었고 문서가 코드와 갈린 자리를 여섯 찾았다.** W-a 가 셋, W-b 가 다섯이었고 이번이 여섯이다. **그중 하나가 이 유닛의 범위를 정면으로 바꾼다.**

```text
   무겁다   5.5 의 「같은 다섯 줄」   page.go 가 통짜 인라인이라 default-src 'self'
                                  한 줄이 제어판을 죽인다.  /ui/ 가 그 헤더로 사는
                                  이유는 static 파일을 따로 내기 때문이다
           「밀렸다는 표시」 앞 팩    앞 팩에 0 이다.  CB1 의 화면 검증이 그것을 본다
           Snapshot 의 용량        ReadRing 이 읽고 버린다.  Parse 의 truncated 가
                                  그 값이고 NC-2 와 CB1 이 둘 다 쓴다

   그 밖    handleTranscript 가 파싱을 0 한다 (오늘 보이는 것은 JSON 원문이다)
           세대 비우기가 사실상 없다 (통째로 갈아치우는 덕에 안 틀렸다)
           5.7 의 파서 줄이 매초 반복을 안 적었다
```

**물음 여덟을 냈다** — 보안 헤더를 이 페이지에 어떻게 거나 · 사건을 어디서 만드나 · NC-1 의 경과를 무엇으로 재나 · NC-2 의 잘림을 무엇으로 아나 · 「밀렸다」가 NC-1 과 같은가 · 원문 토글의 바이트 출처 · NC-3 의 한 줄 · 세대가 바뀔 때 무엇을 비우나. 계획은 `construction/plans/panel-live-functional-design-plan.md`.

**NFR 단계의 실행 여부가 미정이다** — 회차 계획의 9절 표는 U5 를 EXECUTE 로, 같은 문서의 U5 절은 SKIP 으로 적고 그 SKIP 의 근거가 U6 의 문장이다. W-b 의 「nfr 단계를 모두 스킵」이 회차에 이어지는지가 안 정해졌다. 단계 앞에서 묻는다.
**Context**: CONSTRUCTION — W-c Functional Design Part 1. 승인 대기

---

## W-c Functional Design Part 1 — 답 여덟
**Timestamp**: 2026-09-16T07:45:00Z
**User Input**: "권장대로. 여기서도 nfr skip"
**AI Response**: 물음 여덟을 전부 권장으로 닫는다 — 1=A · 2=A · 3=A · 4=A · **5=C** · 6=A · 7=A · 8=A. 다섯째만 권장이 C 였고 나머지 일곱이 A 다. **갈린 답 0.**

NFR Requirements 와 NFR Design 을 **SKIP** 으로 적는다. 회차 계획 9절 표의 EXECUTE 와 갈리므로 **어긋남으로 적는다** — W-b 와 같은 모양이고 이번에는 사용자가 이 회차를 콕 집어 지시했다 (「여기서도」). `unit-of-work.md` 의 U5 절은 원래 SKIP 이라 그쪽과는 안 갈린다.
**Context**: CONSTRUCTION — W-c Functional Design Part 1 승인 · NFR 스킵 지시

---

## W-c Functional Design Part 2 — 산출
**Timestamp**: 2026-09-16T08:05:00Z
**User Input**: (없음 — 「권장대로」의 실행)
**AI Response**: 산출물 셋을 냈다 — `construction/panel-live/functional-design/` (domain-entities · business-logic-model · business-rules).

**모순 검사가 둘을 막았다.**

```text
   ①  2 = A 의 캐시가 3 = A 의 경과를 얼린다.  제어판이 초 단위 수를 실으면
      침묵에서 그 수가 얼어붙는다 — 침묵이 바로 NC-1 이 있는 구간이다.
      시각만 싣고 브라우저가 뺀다.  응답이 바이트의 순수 함수가 되어 캐시가
      성립하고 표시가 폴링 사이에도 흐른다.  캐시 열쇠에 mtime 을 넣는다
   ②  8 = A 의 통째 갈아 그리기가 CB1 의 「접힌 결과」를 못 펴게 한다.
      오늘 안 아픈 이유는 접을 것이 없어서다 — 이 유닛이 접기를 들여오면서
      같이 들어오는 모순이다.  Event.ID (tool_use_id) 로 펼침을 든다.
      Line 은 못 쓴다 — Parse 가 창 안에서 1 부터 세므로 링이 감기면 밀린다
```

**교차 검사가 하나 더 잡았다** — U4 는 메타를 헤더 넷에 싣고 몸통이 `transcript.Result` 통째인데, 제어판의 응답은 객체라 메타를 헤더로 못 옮긴다. U6 이 그 둘을 잇는다. **U5 가 봉투를 정했다** — `transcript` 키에 `Result` 통째, 메타는 형제 키. 안 정했으면 U6 이 둘째 봉투를 짓고 화면 함수가 두 벌이 된다.

**오늘의 동작 하나를 뒤집는다** — `page.go:267` 이 매 폴링마다 무조건 바닥으로 간다. 도는 동안 위로 올려 읽는 것이 지금 불가능하다. 바닥에 있을 때만 따라가고 그 판정을 그리기 전에 한다.

**회차 밖으로 낼 것이 넷이다** — 5.5 의 「같은 다섯 줄」 · 「밀렸다는 표시는 앞 팩 그대로」 · `unit-of-work.md` U5 절의 NFR 두 값 · 파일 행렬에 없는 `internal/enode/transcript.go`.
**Context**: CONSTRUCTION — W-c Functional Design Part 2 완료

---

## W-c Code Generation Part 1 — 계획
**Timestamp**: 2026-09-16T08:30:00Z
**User Input**: "진행"
**AI Response**: Code Generation 계획을 냈다 — `construction/plans/panel-live-code-generation-plan.md`. **Step 열하나 · 체크박스 쉰하나 · 물음 0.** 설계 단계의 답 여덟이 값을 다 정했고 NFR 스킵이 남긴 둘을 계획 0절이 졌다.

**계획이 실측으로 찾은 것 넷. 무거운 것 하나.**

```text
   제어판의 JS 를 재는 시험이 이 저장소에 0 이다.  .mjs 하네스는
   internal/api/ui 의 것이고 (gallery-demo.yml:22 의 node --test) 제어판의 JS 는
   page.go 의 문자열 상수 안이라 임포트가 안 된다.  DOM 규칙 넷 —
   R19 펼침 열쇠 · R20 세대 비우기 · R21 바닥 따라가기 · R14 별개 타이머 —
   의 유일한 검사가 CB1 이다.  변이를 못 건다
```

**답 1 = B 를 골랐으면 CSP 와 함께 이것도 닫혔다.** A 를 고른 대가이므로 숨기지 않고 6절에 적었다. 줄이는 법은 판정을 Go 로 옮기는 것이고 실제로 그렇게 짰다 — `truncated` · 봉투 · 캐시가 전부 Go 이고 브라우저에 남는 것은 그리기뿐이다.

그 밖 셋 — `internal/panel` 이 `ui.securityHeaders` 를 **못 쓴다** (경계가 `panel -> api` 를 막는다. 값이 애초에 갈려 두 벌이 아니다) · 기존 픽스처가 개행이 없어 **사건을 0 개 낸다** (W-b 의 U2 가 같은 자리를 밟았다) · `internal/panel` 커버리지 84.9% 인데 `page.go` 증가분은 문자열 상수라 문장 수에 0 을 더한다.

**설계 문서 한 줄을 계획 단계에서 고쳤다** — R28 이 「`/api/*` 에는 안 건다」였다. `nosniff` 가 정확히 그 전제를 안 믿는 헤더다. 다섯을 모든 응답에 건다. R29 를 더해 경계가 재사용을 막는 것을 적었다.
**Context**: CONSTRUCTION — W-c Code Generation Part 1. 승인 대기

---

## W-c Code Generation Part 2 — 생성
**Timestamp**: 2026-09-16T09:10:00Z
**User Input**: "코드 생성 진행"
**AI Response**: Step 열하나를 다 돌았다. **체크박스 51 중 51.** 요약은 `construction/panel-live/code/code-summary.md`.

```text
   시험 19 패키지 초록 · 스킵 0
   커버리지 미달 0 · 전체 7746/8854 = 87.5%
   internal/panel  84.9% -> 85.9%   (계획 1.4 가 걱정한 자리가 올라갔다)
   라우트  api.go 18 그대로 (CB0 안 건드림) · panel.go 10 (R1)
   gofmt 0 줄 · glyphscan 112 파일 0 · 크로스 빌드 OK · net/http T 6
   diff 0 이어야 할 곳 전부 0 — api · transcript · record · go.mod · boundary_test.go
```

**변이 여섯 전부 빨강. 셋이 처음에 살아남았다.**

**③ 응답에 기간을 싣는다** — 방금 쓴 링은 경과가 0 이라 `omitempty` 가 변이가 더한 키를 지웠고 시험이 없는 키를 못 찾았다. `os.Chtimes` 로 90초 늙히고, 금지어 목록 대신 **봉투가 인정한 키 밖의 수를 전부** 막았다. 한 번 헛디뎠다 — 「60 ~ 120 사이의 수」로 걸렀더니 `total` 이 76 바이트라 걸렸다. **바이트 수와 초가 같은 자리에 온다.**

**④ 캐시 열쇠에서 mtime 을 뺀다** — 시험이 링을 다시 만든 뒤 한 줄을 써서 `total` 이 캐시의 `(0,0)` 과 애초에 안 부딪혔다. **길이가 같고 내용이 다른 두 줄**로 고쳤다. 시험이 그 전제를 스스로 단언한다.

**⑥ 은 시험으로 끝내 못 잡았다 — 구조로 닫았다.** 조용한 링에서는 두 읽기가 같은 바이트다. 경합으로 재니 열에 셋, `len(Data) == Total` 산수로 좁혀도 열에 일곱이었다. **열에 셋을 놓치는 시험은 게이트가 아니다.** 시험을 버리고 `liveBody(snap, mtime, path)` 로 갈라 **스냅샷 하나만 받게** 했다 — 나눠 볼 둘이 없어 그 갈래가 아예 없다. **W-b 의 U2 가 배운 것과 같은 모양이다** (「`Write` 의 패닉 방어는 `recover` 가 아니라 구조다」). 부작용으로 그 함수가 순수해져 결정적으로 시험된다.

**DOM 규칙 넷(R14 · R19 · R20 · R21)은 끝내 변이를 못 걸었다.** 계획 1.1 이 예고한 그대로이고 **넷 다 CB1 이 진다.**
**Context**: CONSTRUCTION — W-c Code Generation Part 2. 승인 대기

---

## 웨이브 순서를 바꾼다 — U7 을 먼저 간다
**Timestamp**: 2026-09-16T09:40:00Z
**User Input**: "그리고 나는 mediator 에 보여지는 실시간 트랜스크립트 만들으라고 했는데?" / "그래. 7번 가자"
**AI Response**: 지적이 맞다. **U5 는 노드 자기 기계의 제어판이고 Mediator 를 안 탄다** (FR-4). Mediator 에 실시간으로 보이게 하는 것은 셋이고 그중 하나만 서 있다 — FR-6 `GET .../log` 는 U4 가 병합했고, **FR-5 의 올리는 쪽(U7)과 FR-7 의 그리는 쪽(U8)이 없다.** Mediator 는 받을 준비도 낼 준비도 끝났는데 **보내는 쪽이 없다.**

**의존 행렬을 확인했다 — U7 의 U5 의존은 게이트뿐이고 코드가 아니다.**

```text
   U7 chunk-push   코드로는 U2 · U4     둘 다 병합됨
                   게이트로는 U5 (CB1)
   U8 fleet-card   코드로는 U4          병합됨
                   게이트로는 U7 (CB3)
```

**어긋남으로 적는다** — 회차 계획의 W-d 는 U6 · U7 병렬이고 W-c 의 CB1 이 그 앞에 있었다. 사용자가 순서를 바꿨다. **U7 을 단독으로 먼저 돌리고 CB1 은 U8 착수 앞에서 CB3 와 함께 잰다** — 둘 다 로그인된 노드에서 계약을 돌려야 하는 게이트라 한 자리에서 보는 것이 싸고, U7 은 U5 의 화면 결정을 하나도 안 물려받는다.

**남는 위험을 이름으로 적는다** — U8 이 「제어판과 같은 모양으로 그린다」라 U5 의 그리는 규칙을 물려받는다. **그 분기점이 U8 착수 자리이고 거기서 CB1 을 본다.** U5 는 코드가 섰고 `unit/panel-live` 에 푸시돼 있다 (커밋 2456ded · 미병합).
**Context**: CONSTRUCTION — 웨이브 순서 변경 · U7 착수

---


## CB1 측정 — U5 `panel-live`
**Timestamp**: 2026-09-16T13:54:11Z
**User Input**: "U5, u8 하자 뭐부터 할지는 알아서."
**AI Response**: U5 를 먼저 골랐다. 근거 셋 — CB1 을 U6 · U7 · U8 이 전부 딛는다 · U5 는 코드가 이미 서 있어 남은 것이 측정과 병합뿐이다 · U8 이 「제어판과 같은 모양」을 베끼므로 그 모양이 초록이기 전에 지으면 틀린 모양이 두 자리에 박힌다. 합본 나무(`measure/u5-cb1` = `unit/chunk-push` + `unit/panel-live`)에서 코드 게이트를 다시 돌렸고 (커버리지 87.6% · 미달 0 · glyphscan 0 · 라우트 18/10) 제어판만 새 빌드로 띄워 계약 둘을 실제로 돌렸다. **CB1 의 다섯 줄이 전부 초록이다** — 도는 동안 문장이 흐르고 · 도구 이름과 접힌 결과가 보이고 · 링이 감겨도 (784128 / 524288 · truncated true · head 7820) 안 깨지고 · 단계 경계에서 카드가 갈리고 · 원문 토글이 JSON 줄을 낸다. 서명은 사람의 것이라 화면 셋을 냈다.
**Context**: CONSTRUCTION — U5 의 장면 게이트 측정

---

## CB1 이 드러낸 것 둘
**Timestamp**: 2026-09-16T13:54:11Z
**User Input**: (없음 — 측정에서 나왔다)
**AI Response**: ① **raw 가 카드의 절반을 넘는다** — 도는 동안 사건 29 중 16 이 raw 였다 (`system/thinking_tokens` 가 토큰 델타마다 한 줄씩 온다). 규칙 R22 는 지켜졌고 파서도 화면도 안 틀렸다. **그리는 규칙이 이 자리를 안 정한 것이고, 그 값이 곧 U8 의 입력이다.** ② **wrap 단계가 하네스 계정의 다섯 시간 창을 태웠다** — base64 스물넷 중 열셋째에서 세션 한도에 닿았고 그 단계 하나가 $1.69 다. 링 감김은 한 번만 재면 되므로 다시 안 돌린다.
**Context**: CONSTRUCTION — U5 의 장면 게이트 측정 · 회차 밖으로 낼 것

---

## U8 설계 변경 — It's a Plan 과의 비교에서 둘을 들였다
**Timestamp**: 2026-09-17T07:30:00Z
**User Input**: "Its a plan의 실시간 트랜스크립트 기능과 비교해라" / "U8 에 들ㅇㅣㄹ것을 activeTool 에 더해 “그리는 것이 한 벌” 을 추가하자. raw접기 추가하고, u8 설계를 변경하고 넘어간다."
**AI Response**: `/home/sunny/itsaplan` 의 코드를 읽어 여덟 축으로 비교했다 — 어휘 · 번역하는 자리 · 나르는 법(SSE 대 폴링) · 쌓는 자리(Postgres 행 대 링 512 KiB) · 이어받기 · 상한 · 잘림을 말하는 법 · 그리는 자리. **enode 가 나은 둘** — 잘림이 값이다(truncated · head · NC-2 · capped), 원문이 남는다. **IAP 가 나은 둘** — 프로토콜 어휘라 장부 줄이 화면에 안 온다, 컴포넌트가 한 벌이다. 사용자가 뒤의 둘 중 하나(한 벌)와 `activeTool` 을 U8 에 들였고, raw 는 버리지 않고 접는 쪽(R30)으로 U5 에 넣었다. 회차 문서 셋(`unit-of-work.md` · 파일 행렬 · 의존 행렬)을 그 값에 맞췄고 `unit/fleet-card` 를 `unit/panel-live` 에서 땄다.
**Context**: CONSTRUCTION — W-e 착수 앞의 설계 변경
## CB1 서명 — U5 `panel-live`
**Timestamp**: 2026-09-17T07:45:00Z
**User Input**: "서명한다"
**AI Response**: CB1 을 초록으로 닫는다. 사람이 실 하네스의 화면을 보고 서명한 것이고 코드 게이트로 대신하지 않았다 (`scene-gates.md` §4). 이 서명이 병합 지점이라 `unit/panel-live` 를 PR 로 `main` 에 올린다. 서명 앞에 고친 것 하나가 R30 (raw 한 줄 접기)이고, 그 규칙의 자동 검사는 U8 이 렌더러를 한 벌로 만들며 선다.
**Context**: CONSTRUCTION — U5 의 장면 게이트 서명 · 병합

---

## U5 병합 · U8 Functional Design Part 1
**Timestamp**: 2026-09-17T08:05:00Z
**User Input**: "서명한다"
**AI Response**: CB1 서명을 받아 PR #46 을 열고 CI 셋(test · cross · bounded-demo)이 초록인 뒤 `main` 에 병합했다 (`3a04b67`). 이어서 U8 의 Functional Design Part 1 을 열었다 — 코드를 먼저 읽어 실측 여덟을 찾았고 물음 여덟을 냈다. 무거운 셋은 별개 타이머가 구조로 안 서는 것 · 데모의 무인증 한도 · 창을 잘라 받으면 경계의 줄이 조용히 사라지는 것이다.
**Context**: CONSTRUCTION — U5 병합 · W-e 착수

---

## Construction · U8 `fleet-card` · Functional Design — 답과 산출물
**Timestamp**: 2026-09-16T23:00:24Z
**User Input**: "U8 의 Functional Design Part 1  질문 모두 권장안으로 답하겠다"
**AI Response**: 계획의 물음 여덟에 `[Answer]: A` 를 박았다 (여덟이 전부 권장 · 갈린 답 0). 답이 다 letter 이고 모호한 것이 0 이라 별도 확인 파일을 안 만들었다. 계획 5.2 가 이름으로 든 모순 검사 자리 넷을 돌려 5.2.1 에 적었다 — **막은 것 하나**(물음 7 의 「렌더러가 두 입력을 받는다」가 물음 1 = A 의 한 벌을 쪼갠다. 렌더러의 입력을 사건 배열 하나로 못 박고 원문 토글을 렌더러 밖에 뒀다) · **값을 정한 것 셋**(끝났다를 폴러가 스스로 정하지 않는다 · 사건 열은 replaceContents 를 안 탄다 · 상태 줄은 step.state 가 연다). 산출물 셋을 `construction/fleet-card/functional-design/` 에 냈다 — domain-entities · business-logic-model · business-rules (규칙은 U5 의 R30 에 이어 R31 ~ R61).

**회차 밖으로 낼 것이 셋 그대로다** — requirements.md SECURITY-08 의 반쪽(데모는 무인증 + 한도) · component-methods.md 의 `name` 기본값 `step` 이 맞는 Run 을 아무도 안 셌다 · N1 에 카드의 전체 재수신이 안 들어 있다.
**Context**: Construction · U8 Functional Design · Part 1 의 답 수집(Step 5)과 산출물 생성(Step 6). 승인 대기.

---

## Construction · U8 `fleet-card` · Functional Design 승인 · NFR 스킵
**Timestamp**: 2026-09-16T23:20:11Z
**User Input**: "승인하고 nfr 건너뛴다"
**AI Response**: Functional Design 을 승인으로 닫고 **NFR Requirements 와 NFR Design 을 둘 다 SKIP** 으로 기록했다 (NFR Design 은 실행 조건이 「NFR Requirements 가 돌았을 때」라 함께 닫힌다). Infrastructure Design 도 SKIP 이다 — 새 인프라가 0 이다 (새 포트 · 새 저장소 · 배포 자원 0).

**어긋남으로 적는다** — `unit-of-work.md` 의 U8 절과 같은 문서 9절의 표가 둘 다 「NFR 요구 **돈다**. 카드가 새 표면이다」로 적었고, 사용자가 그것을 건너뛰기로 정했다. 규칙이 틀린 것이 아니라 사용자가 값을 치르기로 한 것이므로 **숨기지 않고 상태 파일과 계획 0절에 적는다.**

**스킵이 넘긴 값 셋을 Code Generation 계획이 진다** (U5 · U4 와 같은 모양) — ① R43 의 전체 재수신이 N1 에 더하는 양 ② 사건이 수천일 때 매 2초 DOM 비용과 카드 수 ③ 데모 모드에서 카드가 하네스 원문을 무인증으로 보이는 자리 (business-rules 잔여 ③).

다음 단계는 **Code Generation Part 1** 이다.
**Context**: Construction · U8 · Functional Design 승인(Step 8·9) · NFR Requirements/NFR Design/Infrastructure Design SKIP 기록.

---

## Construction · U8 `fleet-card` · Code Generation 계획 승인
**Timestamp**: 2026-09-16T23:40:02Z
**User Input**: "실함대에서만. 넘어가라"
**AI Response**: 계획 8절의 물음 1 을 **A** 로 닫았다 — 카드는 `mode === 'fleet'` 에서만 뜬다. `view.mjs` 의 `askList` 가 이미 그 모양이라 새 개념이 0 이고, `business-rules` 잔여 ③(데모에서 하네스 원문이 무인증으로 보인다)이 함께 닫힌다. Part 1 승인으로 보고 **Part 2 (코드 생성)** 로 간다.
**Context**: Construction · U8 · Code Generation Part 1 승인 · Part 2 착수.

---

## Construction · U8 `fleet-card` · Code Generation Part 2 — 코드가 섰다
**Timestamp**: 2026-09-17T00:15:00Z
**User Input**: (없음 — 앞 승인 「실함대에서만. 넘어가라」로 Part 2 를 돌았다)
**AI Response**: 단계 열둘을 돌았고 체크박스 예순여섯이 실측으로 찼다 (남은 하나는 CB1 재확인 — 사람이다). 변이 일곱 중 **⑦ 이 처음에 살아남았다** — 「총 길이가 세 번 안 움직이면 끝」을 심었는데 시험이 바퀴 셋이라 그 규칙이 발동하기 전에 끝났다. 첫 바퀴는 총 길이가 null 에서 움직인 것으로 세므로 문턱이 넷째 바퀴다. 바퀴를 다섯으로 늘려 고쳤고 일곱이 전부 빨개졌다.

**계획에 없던 것 둘을 했다** — ① `statusLine(events, live)` 를 `card.mjs` 로 뺐다. 상태 줄의 문이 `view.mjs` 안에 있으면 변이 ④ 를 걸 자리가 없어서다 (현황판의 그리기를 재는 하네스가 0 이다). 한 벌이 하나 늘고 재는 자리가 하나 늘었다. ② `boundary_test.go` 에 금지 넷과 함께 **봉인 하나**를 더했다 — 금지는 이름을 아는 넷만 막고 잎임을 못 재는데, 파일 행렬 6.5 가 요구한 것은 잎이다.

**계획의 한 줄을 실측이 고쳤다** — 「기존 시험 열둘이 `view.mjs` 를 임포트한다」가 정확하지 않다. 직접 임포트는 0 이고 `tour` · `webcam` · `submission` · `gallery-view` 넷이 임포트하며 그 넷을 시험 넷이 임포트한다. 전이다. 결론(정적 임포트를 적으면 하네스가 터진다)은 같다.

**게이트** — `go test ./...` 초록 · 패키지별 커버리지 미달 0 (`internal/transcriptui` 는 문장이 0 이라 표에 안 나온다 · 스킵 게이트 위반 0) · `node --test` 98 + 21 = 전부 초록 · glyphscan 통과 · `gofmt` 깨끗 · diff 0 이어야 할 곳 전부 0. **CB4 · CB6 · CB1 재확인은 사람이고 아직이다.**
**Context**: Construction · U8 · Code Generation Part 2 완료. 승인 대기.

---

## Construction · U8 `fleet-card` · Code Generation 승인 · 게이트 측정 착수
**Timestamp**: 2026-09-17T00:25:00Z
**User Input**: "하자"
**AI Response**: Code Generation 을 승인으로 닫고 게이트 측정에 들어간다. CB4 · CB6 · CB1 재확인 셋이다. 측정 나무를 따로 만든다 — main + U5(병합됨) + U7(`unit/chunk-push`) + U8 이 한 나무에 있어야 하고, U7 을 `unit/fleet-card` 에 합치면 U8 의 PR 이 U7 의 커밋을 싣게 되므로 브랜치를 따로 판다.
**Context**: Construction · U8 · Code Generation 승인(Step 6·7) · 게이트 측정 착수.

---

## Construction · U8 `fleet-card` · 게이트 측정 — CB4 초록 · CB1 재확인 초록 · CB6 보류
**Timestamp**: 2026-09-17T03:10:00Z
**User Input**: "계속해"
**AI Response**: 측정 나무(`measure/u8-cb4` — main + U5 + U7 + U8)를 지어 CT103 의 `enode-dev` 에 올리고 실 하네스로 쟀다. Run 넷을 돌렸다 (`cb4-card-1` ~ `cb4-card-4`).

**CB4 초록 — 여섯 줄 전부.** 단계 셋에 카드 셋 · 도는 동안 자란다(40,969 -> 99,997 바이트/15초) · **목록 폴링을 막아도 자란다**(`/v1/nodes` · `/v1/runs` · `/v1/asks` · `/v1/runs/{id}` 거절 12건 동안 로그 호출 8건, 카드는 계속 자랐다) · 원문 토글(`as=raw` 96,866 바이트) · 그래프와 목록을 **안 가린다**(사각형 실측으로 겹침 0) · 끝난 단계가 사유·턴·비용 한 줄로 닫힌다(`success · 턴 21 · $0.6609222`).

**CB1 재확인 초록.** 제어판이 `window.enodeCard` 의 export 넷을 들고 사건 78 을 그린다. raw 한 줄(R30) 그대로 · 원문 토글 230,794 바이트 · 접기/펼치기가 `tool_use_id` 로 든다 · **제어판이 상태 줄을 새로 얻었다**(「생각 중」) · 다음 단계 첫 줄에 카드가 갈린다(장면 ⑤). 제어판과 현황판이 낸 `card.mjs` 가 **같은 바이트다**(R34 를 선 위에서 확인).

**CB6 보류 — 장면 ⑥ 때문이다.** ① ~ ⑤ 와 ⑦ 이 초록이고, ⑥ 의 절반(`runctl record` 의 tar 와 `GET log` 가 세 단계 모두 같은 바이트: 11553 · 102 · 4790)도 초록이다. 나머지 절반인 **「제어판의 지난 작업에서 그 Run 을 누르면 같은 모양으로 읽힌다」가 안 된다** — 아직 판정 JSON 원문을 그린다. **U6 `panel-past` 를 이 회차에 안 지었기 때문이고 U8 밖이다.** `scene-gates.md` §4 대로 보류로 적는다. **보류는 통과가 아니므로 병합 지점도 아니다.**

**측정이 결함 셋을 냈고 전부 고쳤다.** ① `GET log?as=events` 가 빈 로그에 `{"events":null}` 을 냈다 — 카드가 계약 위반으로 섰다. `internal/transcript` 는 U1 의 것이라 회차 밖 브랜치 `fix/transcript-events-is-always-an-array` 에서 고쳤다 (PR #45 의 `verdict.checks` 와 같은 자리 · 같은 방법). ② NC-6 의 시계가 첫 바이트 전에 흘렀다 — 시작도 안 한 단계가 「마지막 갱신 67초 전」이라고 말했다. ③ 한 바이트도 안 온 카드가 「진행 중인 파일」을 들었다. ②③ 은 U8 의 것이라 `unit/fleet-card` 에서 고쳤다.

**운영 기록** — 도는 중에 Mediator 를 갈아 끼우면 그 단계가 `harness_error` 로 죽는다 (`cb4-card-3`). 제품 결함이 아니라 측정 절차의 실수이고, 다음부터는 Run 이 끝난 뒤에 올린다.
**Context**: Construction · U8 · 게이트 측정 (CB4 · CB6 · CB1 재확인).

---
