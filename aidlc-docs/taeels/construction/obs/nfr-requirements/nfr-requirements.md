# obs — 지켜야 하는 것

AI-DLC Construction · obs 유닛(W0 · CP1)의 NFR Requirements 산출물 하나다.
**무엇을 지켜야 하는가**이고 어떻게는 여기가 아니다. 기술 선택은
`tech-stack-decisions.md`, 계획과 답 셋은
`aidlc-docs/taeels/construction/plans/obs-nfr-requirements-plan.md`.

정본 — 확장
`.aidlc/aidlc-rules/aws-aidlc-rule-details/extensions/security/baseline/security-baseline.md`
(차단성) · 팩 `requirements/decisions.md` 2절 · 3절 · `constraints.md` ·
이 유닛의 Functional Design 산출물 셋. **어긋나면 `enode-design` 이 이긴다**
(`requirements/canon.md` 머리말).

---

## 1. 답 셋이 부른 것

물음 셋의 답이 **B · B · A** 다. 셋 다 obs 안으로 들어왔다.

```text
   답      정한 것                                    이 유닛이 지는 일
   ─────   ────────────────────────────────────────   ──────────────────────────
   1 = B   무인증 읽기 셋에 요청 한도를 붙인다          internal/api 에 파일 하나와
           SECURITY-11 을 준수로 닫는다                 시험.  값은 3절
   2 = B   runs_created_idx 를 이 유닛이 더한다        schema.sql 한 줄.
           정렬이 인덱스를 탄다                         변경이 셋이 된다 (ALTER 둘 + 이것)
   3 = A   서버가 Cache-Control: no-store 를 싣는다    읽기 셋 셋의 응답 헤더.
           읽는 쪽 셋이 각자 안 짠다                    본문의 글자는 안 바뀐다
```

**셋의 공통점 하나** — 하류가 각자 정하지 않게 W0 이 값을 세운다. Functional
Design 이 `runctl.Step.StartedAt` 과 `assigned` 계약을 W0 에 세운 것과 같은
이유다: 담당이 갈린 뒤에 고치면 남의 파일을 여는 일이 된다.

**Functional Design 의 파일 목록이 는다** (`business-rules.md` 7.5).

```text
   internal/api/<한도>.go     신규.  요청 한도 하나 — 답 1=B          내부 · 단독 obs
   internal/store/schema.sql  CREATE INDEX 한 줄이 더 붙는다 — 답 2=B  접점 (queue 와 공유)
   internal/api/nodes.go      Cache-Control 한 줄 — 답 3=A            신규 (이미 목록에 있다)
   internal/api/runs.go       같다                                    신규 (이미 목록에 있다)
   internal/api/api.go        getRun 에 같은 한 줄 · 한도 미들웨어 등록   접점 (이미 목록에 있다)
```

---

## 2. security-baseline — 열다섯 규칙의 판정

`decisions.md` 3절이 처리를 미리 적은 것은 **다섯**(01 · 03 · 07 · 08 · 12)이다.
나머지 열도 차단성 확장의 규칙이므로 판정을 비워 두지 않는다. **해당 없음에는
이유를 적는다** — 빈 칸은 판정이 아니다.

| 규칙 | 판정 | 근거 |
|---|---|---|
| SECURITY-01 암호화 | 해당 없음 (기존 코드 사실) | obs 가 저장소를 새로 안 만든다. 평문 HTTP 와 `sslmode=disable` 은 `constraints.md` 6절이 뺐고 `decisions.md` 3절이 기록만으로 닫았다 |
| SECURITY-02 중간자 접근 로깅 | 해당 없음 | 로드밸런서 · API 게이트웨이 · CDN 이 없다. obs 가 새로 두지도 않는다 |
| SECURITY-03 애플리케이션 로깅 | 준수 (새 표면 범위) | 새 로그에 토큰과 정책 값 원문을 안 찍는다. 로거는 기존 `s.log` · `Store.Log` 를 그대로 탄다. **중앙 로그 서비스와 상관관계 ID 가 없는 것은 기존 코드 사실**이고 `decisions.md` 3절이 기록만으로 닫은 자리다 |
| SECURITY-04 HTTP 보안 헤더 | 해당 없음 | obs 의 라우트는 JSON 이고 HTML 을 안 낸다. HTML 표면(`GET /ui/`)은 `internal/api/ui` 가 다섯 헤더를 이미 싣는다 (실측 `ui.go:34~42`) — ui 유닛의 자리다 |
| SECURITY-05 입력 검증 | 준수 | `business-rules.md` 2절 — 질의 인자 넷의 형 · 길이 상한 200 · 빈 값 규칙, 광고 정책 값의 어휘 검사. SQL 은 전부 파라미터 바인딩이고 문자열 연결이 0 이다. 읽기 셋에 요청 본문이 없고 `POST /v1/nodes` 는 기존 `io.LimitReader` 를 그대로 탄다 |
| SECURITY-06 최소 권한 | 해당 없음 | IAM · 클라우드 정책 대상이 아니다. RE 가 클라우드 자원 0 을 확인했고 `constraints.md` 6절이 배포를 뺐다 |
| SECURITY-07 네트워크 구성 | 해당 없음 (기존 코드 사실) | 새 리스너가 없다. 기존 `:8080` 에 라우트를 더한다 |
| SECURITY-08 접근 제어 | **조건부** | 실 함대는 전부 `s.auth` 라 deny-by-default 다. **데모 인스턴스의 읽기 셋 셋은 문서화된 예외**이고 `ADR-065` 2절을 벗어난다 (`business-logic-model.md` 7.1 · 되돌리는 값과 노출 목록이 거기 있다). 객체 수준 권한은 해당 없음 — `ADR-015` 1절이 `principal` 을 식별로 정했고 소유 개념이 없다. CORS 도 해당 없음 — 설정이 없고 화면과 API 가 같은 프로세스 · 같은 포트라 동일 출처다 (실측) |
| SECURITY-09 하드닝 | 준수 | 기본 자격증명이 0 이다 (토큰은 설정에서 온다). 오류 응답이 `{"error":{code,reason}}` 고정 문구라 스택 트레이스 · 내부 경로 · 프레임워크 판이 안 나간다. 디렉터리 목록은 이 표면에 없다 (JSON). Go 1.26 은 현재 판이다 |
| SECURITY-10 공급망 | 준수 | **새 의존 0.** 답 1=B 도 직접 짜므로 `go.mod` 가 안 움직인다 (`tech-stack-decisions.md` 2절). `go.sum` 이 잠금 파일이고 커밋돼 있다. 툴체인이 `go1.26.6` 로 고정이다 |
| SECURITY-11 보안 설계 | **답 1=B 로 준수** | 요청 한도를 붙인다 (3절). 한도가 새 파일 하나에 격리되고, 방어 층이 둘이며 (가둠 셋 + 한도), 오용 사례를 3.3 에 적는다. **전역 한도의 대가도 3.2 에 적는다** — 준수를 대가 없이 적으면 그 표가 심사의 근거가 될 때 거짓이 된다 |
| SECURITY-12 인증 · 자격증명 | **조건부** | 새 자격증명이 0 이고 하드코딩이 0 이다. 토큰은 설정 파일에 산다 (`ADR-015` 1절). 그러나 데모 인스턴스에서 읽기 셋 셋이 인증을 안 거친다 — Functional Design 6절이 이미 조건부로 적었고 그대로 잇는다. 비밀번호 · 세션 · MFA 는 해당 없음이다: 로그인 흐름이 없고 게스트 이름은 표시 라벨이지 신원이 아니다 (`constraints.md`) |
| SECURITY-13 무결성 검증 | 준수 | 역직렬화가 `encoding/json` 의 고정 타입 바인딩뿐이다 — 임의 타입을 푸는 자리가 0 이다. 외부 CDN 스크립트가 0 이다 (새 의존 0). 이 유닛의 유일한 쓰기(`UpsertAdvert`)는 광고 갱신이라 감사 대상 변경이 아니다 — 그 자리는 봉인(Record)이고 `ADR-025` 가 진다 |
| SECURITY-14 경보 · 모니터링 | 해당 없음 (범위 밖) | 실 함대의 `401` 은 기존 경로 그대로이고 데모 읽기 셋에는 인증이 없어 인증 실패 이벤트 자체가 안 난다. 로그 보존 · 대시보드 · 경보는 `constraints.md` 6절이 뺀 운영이다. **기록이지 통과가 아니다** |
| SECURITY-15 예외 처리 | 준수 | 모든 외부 호출(DB)에 오류 분기가 있다. 실패는 닫는 쪽이다 — 조회 실패 `503` · 한도 초과 `429`. 사용자에게 가는 문구가 일반 문구다. 계약 읽기 실패만 `200` 인데 그것은 **권한 판정을 우회하는 자리가 아니라 부분 응답**이고 빠진 사실이 `warnings` 에 실린다 (`business-rules.md` 4절) |

**조건부가 둘이다** — SECURITY-08 과 SECURITY-12. 같은 하나의 사실에서 나온다:
데모 인스턴스의 읽기 셋 셋이 무인증이다. **둘 다 준수로 칠하지 않는다** —
`business-rules.md` 6절이 그것을 SECURITY-08 칸에만 적어 두 칸을 초록으로 보이게
하지 말라고 적었고, 이 표도 같은 규약이다.

---

## 3. 요청 한도 (SECURITY-11) — 값과 대가

답 1=B 다. **값을 여기서 정한다** — NFR Design 이 SKIP 이므로 값이 여기 없으면
Code Generation 이 짐작한다.

### 3.1 값

```text
   적용 범위    데모 모드에서 무인증으로 등록되는 읽기 셋 셋뿐 —
                GET /v1/nodes · GET /v1/runs · GET /v1/runs/{id}.
                실 함대(ENODE_DEMO_MODE 기본값 거짓)의 s.auth 경로는 안 걸린다.
                POST /v1/nodes 도 안 걸린다 — 잠긴 채이고 하트비트를 겸한다

   나누는 키    없다.  읽기 셋 전체에 전역 한도 하나다

   값           초당 120 요청 · 버스트 240

   유도         시청자 30 (계획 2.2) 이 5초마다 다섯을 부른다 —
                GET /v1/nodes 1 + GET /v1/runs 1 + 두 홉 getRun 3
                (임대된 노드만 돈다 · domain-entities 2.2).
                = 시청자당 1 rps · 30 명이면 30 rps 가 정상 상태.
                한도는 그 4배다 — 정상 폴링이 한도에 안 닿는 것이 첫 조건이다

   넘으면       429 + reason "rate limited" + Retry-After: 1
                기존 fail(w, code, reason) 규약 그대로다 (api.go:144)
```

**한도의 상태는 프로세스 메모리에 산다.** 재기동하면 비어서 시작한다 — 일회용
데모 인스턴스이므로 그것이 문제가 아니고, 그 사실이 곧 「Mediator 를 늘리면
한도도 나뉜다」는 4.2 의 제약이다.

### 3.2 IP 로 안 나누는 이유 — 그리고 그 대가

```text
   IP 로 나누면   대회장 관객이 한 NAT 뒤에 있다.  출발지 IP 로 나누면
                 관객 전체가 한 클라이언트로 묶여 정상 폴링이 막힌다.
                 CP10(데모 완주)을 직접 깨는 쪽이다

   전역으로 두면  한 클라이언트가 한도를 다 먹으면 나머지가 굶는다.
                 공정 분배가 아니다 — 이 한도가 막는 것은 폭주 루프와
                 스크립트 홍수가 일회용 인스턴스를 넘어뜨리는 것이다
```

**뒤쪽 대가를 수락한다.** 앞쪽은 정상 동작을 막고 뒤쪽은 비정상 동작 하나가
남을 막는다. 데모 표면이 실 제품 표면이 아니라는 `constraints.md` 의 선이 이
저울의 근거다. **진행자에게 남기는 표시**이기도 하다 (6절).

### 3.3 오용 사례 하나 (SECURITY-11 의 요구)

```text
   공개된 데모 URL 을 본 사람이 GET /v1/runs?limit=1000 을 루프로 부른다.
   페이지네이션이 없으므로 매 요청이 최대 1000 행을 정렬해 낸다.

   막는 것 셋
     ①  limit 상한 1000 이 한 요청의 크기를 자른다 (business-rules 2.1)
     ②  runs_created_idx 가 정렬을 인덱스로 옮긴다 (답 2=B)
     ③  전역 한도 120 rps 가 요청 수를 자른다 (이 절)

   남는 것
     한도 안에서 도는 루프는 정상 관객의 몫을 먹는다.  3.2 의 수락한 대가다
```

---

## 4. 비-보안 NFR

### 4.1 성능

**수치 목표를 안 적는다** (계획 2.2 — 일회용 인스턴스라 지킬 자리가 없다).
대신 **비용이 어디서 나는지**와 그것을 무엇이 누르는지를 적는다.

```text
   GET /v1/nodes      질의 하나.  nodes LEFT JOIN leases.
                      leases.node_id 가 PRIMARY KEY 라 행이 안 불어난다 (실측
                      schema.sql:58).  행 수는 만료 전 노드 수이고 데모 함대는
                      한 자리 수다

   GET /v1/runs       답 2=B 로 runs_created_idx (created_at) 가 선다.
                      ?work= 없는 기본 목록과 ?since= 가 그 인덱스를 탄다.
                      ?work= 는 기존 runs_work_idx (work_id, created_at) 그대로다.
                      btree 는 역방향으로도 훑으므로 인덱스 방향이 정렬 방향을
                      가르지 않는다 — created_at DESC 정렬이 그대로 인덱스를 탄다

   GET /v1/runs/{id}  기존 경로에 LiveContract 하나가 붙는다.  새 질의가 아니고
                      그 메서드는 이미 세 자리가 부른다 (domain-entities 8절)

   폴링 부하           시청자 30 · 5초 · 요청 다섯 = 30 rps.  한도 120 이 그 4배다
```

**쓰기 비용이 는다.** 인덱스가 하나 늘면 `runs` 의 INSERT 마다 그 인덱스도
갱신된다. 제출 빈도가 사람 손이라 무시할 수 있고, 대신 폴링 쪽이 초당 여섯 번
정렬을 안 한다 — 저울이 읽기 쪽으로 크게 기운다.

### 4.2 확장

```text
   수평 확장 안 한다   Mediator 가 하나다 (constraints 6절 · HA 제외).
                     한도가 프로세스 안의 상태이므로 인스턴스를 늘리면 한도도
                     나뉜다.  늘릴 계획이 없다는 것이 그 값을 쓸 수 있는 조건이다

   커넥션 풀          pgxpool 기본값 그대로 둔다 — max(4, NumCPU) (실측 store.go:66).
                     이 유닛이 안 바꾼다.  올리면 기존 쓰기 경로의 대기 시간이
                     함께 움직이고, 그것은 이 유닛이 재는 값이 아니다

   페이지네이션 없음   limit 상한 1000 이 사실상의 상한이다.  넘겨야 하면
                     정본(ADR-065 4절)을 먼저 연다 — 코드가 아니다
```

### 4.3 가용

```text
   목표 수치 없다      일회용 데모 인스턴스 · 단일 Mediator (계획 2.2)

   저장소 조회 실패     503 query failed.  서버는 재시도하지 않는다 —
                      화면이 5초마다 다시 부르고 연속 3회의 처리는
                      decisions 2절이 이미 S1b 로 정했다.  서버 안의 재시도는
                      그 카운트를 흐린다

   계약 읽기 실패      200 + requires 생략 + warnings 한 줄.  조회를 안 막는다
                      (business-rules 4절)

   한도 초과          429 + Retry-After: 1

   기동               스키마 마이그레이션이 멱등이라 ALTER 둘과 INDEX 하나가
                      매 기동 돈다.  CREATE INDEX IF NOT EXISTS 는 이미 있는
                      인덱스에 아무 일도 하지 않는다
```

### 4.4 신뢰

```text
   판정을 안 낸다       서버가 「왜 안 맞나」를 안 내므로 낡은 판정이 안 산다
                      (ADR-069 5절 · business-rules 5절)

   시계가 하나다        observed_at 이 DB 시계이고 만료 필터가 같은 값을 쓴다
                      (business-logic-model 2절).  행이 0 이어도 값이 선다

   순서가 고정이다      nodes 는 node_id 오름차순 · runs 는 created_at 내림차순.
                      같은 함대를 두 번 읽으면 카드가 자리를 안 바꾼다

   캐시가 안 낀다       답 3=A 로 읽기 셋 셋이 Cache-Control: no-store 를 낸다.
                      GET /v1/nodes 는 ?t= 로 캐시를 피할 수 없으므로
                      (질의 인자가 400 이다 · business-rules 2.4) 서버가 진다.
                      본문의 글자는 안 바뀌므로 mcp 의 글자 일치(CP5)와 무관하다

   한도의 실패 모드     메모리 안의 값 하나.  재기동하면 비어서 시작한다
```

### 4.5 유지보수 — 커버리지가 걸리는 자리

`internal/api` 의 여유가 이 유닛의 병목이다. 수는 Functional Design 7.6 의
실측(2026-09-08)이다.

```text
   패키지            실측                여유
   ───────────────   ────────────────    ──────────
   internal/api      328/403 = 81.4%     5.6 문장
   internal/store    1232/1501 = 82.1%   31.2 문장
   internal/runctl   78/81 = 96.3%       13.2 문장
```

답 셋이 이 수에 미치는 영향:

```text
   답 1=B 의 한도    열여섯 문장 남짓 (버킷 · allow · 미들웨어).
                    DB 가 필요 없다 — 순수 로직과 httptest 로 전부 덮인다.
                    전부 덮이면 (328+16)/(403+16) = 82.1% 로 하한에서 멀어진다.
                    안 덮으면 새 문장 n 의 미덮 허용은 0.2n + 5.6 = 8.8 문장

   답 3=A 의 헤더    라우트마다 한 줄이거나 헬퍼 하나.  기존 라우트 시험이 덮는다

   답 2=B 의 인덱스  Go 문장이 0 이다.  schema.sql 한 줄이다
```

**답 1=B 가 커버리지를 나쁘게 하지 않는다** — 덮이는 문장은 비율을 올린다.
비용은 여유가 아니라 **시험을 쓰는 손**이다.

`.coverage-contract.yml` 은 게이트 입력이 아니라 기준선 스냅숏이고 낡았다 —
`internal/api` 에서 한 문장 어긋나 있고 `internal/api/ui` 가 아예 없다. 이
유닛은 그 파일을 안 고치되 유닛을 닫을 때 낡은 정도를 한 줄로 적는다
(`business-rules.md` 7.6 과 같은 값이다).

---

## 5. 재는 자리

`business-rules.md` 7절의 완료 조건에 **이 단계가 더하는 것 셋**이다. CP1 의
curl 두 줄과 CP0 은 그대로다 — 이 단계가 게이트를 다시 정하지 않는다.

```text
   요청 한도    데모 모드로 띄우고 초당 120 을 넘겨 429 를 받는다.
                실 함대 모드(기본)에서는 같은 부하에 429 가 안 난다 —
                두 번째 줄이 「무인증 경로에만」을 재는 자리다

   인덱스       EXPLAIN 으로 ?work= 없는 목록이 runs_created_idx 를 타는지 본다.
                Seq Scan + Sort 가 나오면 빨간 것이다

   캐시 헤더    curl -i 로 읽기 셋 셋 다 Cache-Control: no-store 를 낸다.
                셋이다 — getRun 도 두 홉으로 폴링에 들어간다
```

---

## 6. 진행자에게 남기는 표시

```text
   ①  429 가 정본의 코드 표에 없다.  mediator-api.md 1.2 는 여덟이다 —
      400 · 401 · 404 · 409 · 410 · 413 · 422 · 503.  Functional Design 이
      이미 넷을 표시로 남겼고(400 의 질의 검증 뜻 · 503 의 질의 실패 뜻 등)
      이것이 다섯째다.  정본의 코드 표를 넓히는 것은 진행자의 일이다

   ②  SECURITY-11 의 첫 판정이 이 유닛에서 났다.  demo-back(W2)과 ui 가
      같은 데모 인스턴스의 같은 공개 표면이므로 그 표를 이어 쓴다.
      유닛마다 다르게 닫으면 표가 갈리고, 갈린 표는 심사의 근거가 못 된다

   ③  전역 한도의 대가 — 한 클라이언트가 한도를 다 먹으면 나머지가 굶는다.
      IP 로 나누면 NAT 뒤의 관객 전체가 한 클라이언트가 되어 정상 폴링이
      막힌다.  뒤쪽이 CP10 을 직접 깨므로 앞을 수락했다 (3.2)

   ④  schema.sql 의 변경이 셋이 됐다 — ALTER 둘 + CREATE INDEX 하나.
      queue 도 같은 파일을 만지므로 진행자의 직렬 병합 대상 그대로다

   ⑤  NFR Design 이 SKIP 인데 답 1=B 가 「어떻게」를 남긴다 — 한도의 자료구조와
      미들웨어를 어디에 거는가.  이 문서 3절이 값을 다 적었으므로 Code Generation
      계획이 지고 갈 수 있다.  그 단계를 따로 열지는 진행자가 정한다

   ⑥  조건부 둘(SECURITY-08 · SECURITY-12)은 Functional Design 에서 이어진
      같은 사실이다 — 데모 읽기 셋의 무인증이 ADR-065 2절을 벗어난다.
      되돌리는 값은 Handler() 의 조건부 등록과 config 스위치 하나다
      (business-logic-model.md 7.1)
```
