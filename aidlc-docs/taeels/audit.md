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
