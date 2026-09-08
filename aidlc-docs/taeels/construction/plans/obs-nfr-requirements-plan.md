# obs — NFR Requirements 계획

AI-DLC Construction · **obs 유닛**(W0 · CP1 · 토대)의 NFR Requirements Part 1 이다.
요구 본문이 아니라 **무엇을 재고 무엇을 물을지의 계획**이다. 답이 오면 그 값으로
`aidlc-docs/taeels/construction/obs/nfr-requirements/` 아래 산출물 둘을 낸다.

정본 — 팩 `requirements/`(decisions 2절 권장값 · 3절 보안 확장 취급 ·
constraints 차단 게이트 다섯) · 확장
`.aidlc/aidlc-rules/aws-aidlc-rule-details/extensions/security/baseline/security-baseline.md` ·
이 유닛의 Functional Design 산출물 셋
(`aidlc-docs/taeels/construction/obs/functional-design/`).

---

## 0. 이 단계는 회차 계획에서 SKIP 이었다

`aidlc-docs/v1-run-dhseo/inception/plans/execution-plan.md` 가 NFR Requirements 를
**SKIP** 으로 승인받았다. 근거는 「팩이 값으로 닫았다 — 이 단계를 돌리면 팩을
옮겨 적기만 한다」였다. 사용자가 2026-09-08 에 이 단계를 다시 들였고
(`nfr requirements 시작`), AI-DLC 의 「User Control — 사용자가 단계를 넣고 뺀다」가
그 자리다. **이 계획은 그 지시로 선다.**

**담당 상태 파일이 이미 회차 계획과 어긋나 있었다.** `aidlc-docs/taeels/aidlc-state.md`
는 NFR Requirements 와 NFR Design 을 **대기**로 적었는데 회차 계획은 둘 다 SKIP
이다. 어긋남을 이 계획이 닫는다 — 담당 문서는 여기서 정리하고, **회차 계획과
회차 상태 파일을 고칠지는 진행자의 것**이다.

### 0.1 옮겨 적기가 아닌 것 셋

SKIP 판정은 회차 전체에 대한 것이었고 유닛별 Functional Design 앞에서 내려졌다.
obs 의 Functional Design 이 끝난 지금, **팩이 값으로 닫지 않은 자리 셋**이 이
유닛에 남아 있다. 이 단계가 내는 값은 그 셋이다.

```text
   ①  SECURITY-11 의 요청 한도    팩 3절이 처리를 미리 적은 SECURITY 규칙은
                                 다섯이다 — 01 · 03 · 07 · 08 · 12.  11 이 그
                                 목록에 없다.  Functional Design 의 답 4=A 로
                                 읽기 셋 셋이 데모 인스턴스에서 무인증 공개
                                 표면이 되면서 11 이 이 유닛에 처음 걸린다

   ②  정렬 비용                   runs 표의 인덱스가 runs_work_idx 하나이고
                                 화면과 MCP 의 기본 사용법(?work= 없는 목록)이
                                 그것을 못 탄다.  Functional Design 은 「안
                                 붙인다」로 적었으나 그 근거로 든 mediator-api.md
                                 의 문장은 API 표면을 그은 것이지 DB 인덱스를
                                 금한 것이 아니다

   ③  캐시 헤더의 주인            business-rules.md 2.4 가 GET /v1/nodes 에
                                 질의 인자를 금하고(?t= 도 400 이다) 「캐시를
                                 피해야 하면 헤더로 한다」고 적은 뒤 누가
                                 붙이는지를 안 정했다.  읽는 쪽이 셋이고
                                 담당이 셋으로 갈린다
```

셋 다 **나중에 되돌리는 값이 크다.** ①은 차단성 확장의 판정이라 비워 둘 수
없고, ②는 `schema.sql` 이라 진행자의 직렬 병합 대상이며, ③은 안 정하면 하류
셋이 각자 다르게 짓는다.

---

## 1. 착수 전에 실측한 것 (기존 코드)

물음의 선택지가 짐작이 아니라 사실 위에 서게 하려고 먼저 읽었다.

```text
   확인한 사실                                                      자리
   ─────────────────────────────────────────────────────────────   ──────────────────────────
   요청 한도 · throttle 코드가 저장소에 없다 (grep 0건)               internal/ · cmd/
   CORS 설정이 없다.  필요도 없다 — 화면(/ui/)과 읽기 셋이 같은      internal/api/api.go:80
     Mediator 프로세스 · 같은 포트라 동일 출처다                     internal/api/ui/ui.go:22
   demo-back 도 같은 패키지 · 같은 포트다 (internal/api/demo.go)     unit-of-work.md 8절
   http.Server 에 ReadHeaderTimeout 10초만 걸려 있다.               cmd/mediator/main.go:103~107
     WriteTimeout 을 일부러 안 건다 — claim 이 최대 2시간 매달린다
   pgxpool.New(ctx, url) 을 옵션 없이 부른다.  MaxConns 는          internal/store/store.go:66
     pgx 기본값이다 (max(4, NumCPU))
   요청 접근 로그가 없다.  s.log 는 오류 자리에만 걸려 있다           internal/api/api.go 전체
   본문 크기 상한은 쓰기 둘에만 있다 (io.LimitReader).               internal/api/api.go:540 · 744
     읽기 셋은 본문이 없다
   runs 표의 인덱스가 runs_work_idx (work_id, created_at) 하나다     internal/store/schema.sql:41
   leases.node_id 가 PRIMARY KEY 다 — 노드마다 임대가 최대 하나라    internal/store/schema.sql:58
     GET /v1/nodes 의 LEFT JOIN 이 행을 안 불린다
   go.mod 에 golang.org/x/time 이 없다.  한도를 붙이면 직접 짜거나   go.mod
     의존이 하나 는다.  팩은 「새 Go 의존 0」이다
   패키지별 커버리지 하한 80% 는 awk 가 집행한다                     ci.yml:269
   심볼 상한 둘(crypto/tls 10 · net/http 50)도 같은 파일이다          ci.yml:482~484
   internal/api/ui 가 SECURITY-04 다섯 헤더를 이미 싣는다 —          internal/api/ui/ui.go:34~42
     정적 화면 표면이고 obs 의 JSON 라우트는 그 대상이 아니다
```

**동일 출처가 확인된 것에 값이 있다.** SECURITY-08 의 CORS 항목이 이 유닛에
해당 없음으로 닫힌다 — 무인증 읽기 셋이라도 교차 출처 요청을 여는 설정이 없다.

---

## 2. 이미 값으로 닫힌 것 — 다시 묻지 않는다

### 2.1 팩과 정본이 진 값

```text
   항목                  값                                    근거
   ──────────────────   ─────────────────────────────────────  ─────────────────────────
   폴링 간격             둘 다 5초                              decisions 2절
   limit 기본 · 상한      100 · 1000 (넘으면 자른다)             기본 decisions 2절 ·
                                                               상한 business-rules 2.1
   페이지네이션          없다.  limit 뿐                        decisions 2절 · ADR-065 4절
   만료 임박 임계값      expires_at 까지 30초                    decisions 2절
   갱신 실패 임계값      연속 3회면 S1b                          decisions 2절
   새 Go 의존            0.  Go 1.26 고정                        실행 계획 「기술 스택 고정」
   커버리지 하한         패키지별 80%                            ci.yml:269
   TLS · 전송 암호화      안 넣는다                               constraints 6절 · decisions 3절
   비밀 관리자           안 넣는다.  토큰은 설정 파일에 산다       decisions 3절 · ADR-015 1절
   HA · 다중 Mediator     안 한다                                constraints 6절
```

### 2.2 이 계획이 채우는 권장값 — 물음으로 안 올린다

**벗어나면 근거를 적는다.** 팩 2절과 같은 규약이다. 물음 셋에 안 넣은 이유는
되돌리는 값이 작거나 정본이 이미 방향을 정했기 때문이다.

```text
   동시 시청자 규모     대회장 관객 30 내외를 가정한다.  폴링 5초면 라우트당 6 rps
                       근거 — 데모 Mediator 는 실 함대와 분리된 일회용 인스턴스이고
                       (decisions 8.3) CP10 의 관객은 한자리에 모인 사람이다.
                       이 수를 크게 넘길 의도가 있으면 물음 1 과 2 의 답이 달라지므로
                       그때는 Other 로 적는다

   조회 실패의 재시도   서버는 재시도하지 않는다.  503 을 그대로 내고 폴링이 재시도다
                       근거 — 화면이 5초마다 다시 부르고 연속 3회 실패의 처리는
                       decisions 2절이 이미 S1b 로 정했다.  서버 안의 재시도는
                       그 카운트를 흐린다

   접근 로그            안 남긴다.  오류 로그만 오늘 그대로다
                       근거 — decisions 3절이 SECURITY-03 을 「기존 코드 사실로
                       기록만」으로 닫았다.  무인증 읽기 셋에만 접근 로그를 새로
                       다는 것은 그 줄을 넘고, internal/api 의 커버리지 여유
                       5.6 문장을 로그 문장에 쓰는 일이다

   가용성 수치 목표     안 적는다
                       근거 — 일회용 데모 인스턴스와 단일 Mediator 다.  HA 를
                       constraints 6절이 뺐으므로 수치를 적어도 지킬 자리가 없다.
                       대신 실패 모드와 그때의 응답을 적는다 (business-rules 4절)

   응답 본문 크기       상한을 새로 안 만든다.  limit 상한 1000 이 사실상의 상한이다
                       근거 — 읽기 셋에 요청 본문이 없고 응답 크기는 limit 이 진다.
                       화면이 기본 100 을 쓰므로 폴링 한 번의 본문은 수십 KB 대다
```

---

## 3. 물음 셋

답은 `[Answer]:` 뒤에 글자 하나로 적는다. **0.1 의 셋이 그대로 물음 셋이다** —
팩이 안 닫았고, 되돌리는 값이 크고, obs 가 정하지 않으면 하류가 각자 정한다.

### 물음 1
`security-baseline` 은 **차단성**이다. 그 SECURITY-11 이 「public-facing
endpoints MUST implement rate limiting」을 요구하는데, 팩 3절이 처리를 미리 적은
규칙 다섯(01 · 03 · 07 · 08 · 12)에 **11 이 없다.** Functional Design 의 답 4=A 로
`GET /v1/nodes` · `GET /v1/runs` · `GET /v1/runs/{id}` 가 데모 인스턴스에서
무인증이 되면서 이 규칙이 obs 에 처음 걸린다. 판정을 비워 둘 수 없다.

A) 붙이지 않는다. 가둠 셋(실 함대와 분리된 일회용 데모 Mediator · 서버측
   allow-list · 브라우저에 실 토큰 없음)을 완화 수단으로 적고 SECURITY-11 을
   **조건부**로 남긴다. `constraints.md` 의 「수락한 위험」이 SECURITY-08 을
   값으로 닫아 수락한 것과 같은 방식이다. 코드 변경 0

B) obs 가 붙인다. 무인증으로 등록되는 경로에만 프로세스 안의 한도 하나.
   `golang.org/x/time/rate` 를 못 쓰므로(새 의존 0) 직접 짠다 — `internal/api` 에
   파일 하나와 시험. 덮이는 문장이면 커버리지 여유 5.6 문장을 안 먹는다

C) demo-back(W2 · runixs)에 넘긴다. 데모 표면 전체가 그 유닛의 책임이라 한 자리에
   붙고 obs 는 요구만 적는다. 다만 obs 가 병합된 뒤 W2 까지 그 창이 열려 있다

X) Other (please describe after [Answer]: tag below)

[Answer]: B

### 물음 2
`runs` 표의 인덱스는 `runs_work_idx (work_id, created_at)` 하나다. 화면과 MCP 의
기본 사용법은 `?work=` 없는 목록이라 그것을 못 타고, `created_at DESC` 정렬이
폴링마다 전체 정렬이다. Functional Design 은 「이 유닛은 인덱스를 안 붙인다」로
적었으나 그 근거로 든 `mediator-api.md` 의 문장(「검색 계층이 아니다. 인덱스도
질의 언어도 없다」)은 **API 표면**을 그은 것이지 DB 인덱스를 금한 것이 아니다.
값을 여기서 정한다.

A) 붙이지 않는다. 임계 행 수를 산출물에 적고 미룬다 — 데모는 일회용이라 `runs` 가
   수백 행을 안 넘고 실 함대도 오늘 그 규모다. 넘으면 그때 붙인다

B) obs 가 `runs_created_idx (created_at DESC)` 를 `schema.sql` 에 더한다.
   ALTER 가 이미 둘이라 셋째이고, 같은 파일을 queue 가 만지므로 진행자의 직렬
   대상에 한 줄이 는다

C) queue(W1 · shin-son)에 넘긴다 — `QUEUED` 조회가 같은 표를 더 자주 읽으므로
   필요를 그쪽이 먼저 본다

X) Other (please describe after [Answer]: tag below)

[Answer]: B

### 물음 3
`business-rules.md` 2.4 가 `GET /v1/nodes` 에 질의 인자를 금했다 — 캐시 무력화용
`?t=` 도 `400` 이다. 같은 절이 「폴링이 캐시를 피해야 하면 헤더로 한다」고 적고
**누가 그 헤더를 붙이는지**를 안 정했다. 읽는 쪽이 셋(ui · panel · mcp)이고
담당이 셋으로 갈린다.

A) obs 가 읽기 셋 응답에 `Cache-Control: no-store` 를 싣는다. 서버가 한 번 지면
   셋이 각자 안 짠다. 응답 헤더가 늘 뿐 **본문의 글자는 안 바뀌므로** mcp 의
   글자 일치(CP5)와 부딪치지 않는다

B) 부르는 쪽이 진다. 화면은 `fetch` 의 캐시 옵션으로, mcp 는 `runctl.Client` 의
   요청 헤더로. obs 는 응답을 안 바꾼다

C) 안 정한다. 오늘 관측된 문제가 아니므로 값을 만들지 않는다

X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## 4. 실행 계획

- [x] 4.1 물음 셋의 답을 받는다. 모호하거나 서로 어긋나면 확인 물음 파일을 따로 낸다
- [x] 4.2 SECURITY 열다섯 규칙을 obs 표면에 하나씩 대고 적용 · 해당 없음 · 조건부를
      가른다. **해당 없음에는 이유를 적는다** — 빈 칸은 판정이 아니다
- [x] 4.3 Functional Design 6절의 판정 다섯(01 · 03 · 07 · 08 · 12)을 열다섯 칸으로
      넓히고, 그때 조건부로 남긴 SECURITY-12 를 그대로 잇는다
- [x] 4.4 비-보안 NFR 을 축 다섯으로 적는다 — 성능 · 확장 · 가용 · 신뢰 · 유지보수.
      값마다 근거와 재는 자리를 함께 적는다
- [x] 4.5 기술 스택 결정을 적는다. 새 의존 0 의 실측 근거(`go.mod` 의 직접 의존 셋)와
      물음 1 의 답이 그것을 바꾸는지
- [x] 4.6 산출물 둘을 낸다 (5절)
- [x] 4.7 정합 검사 — `enode-design/scripts/emphasis-check.py` · `scripts/glyphscan.go`
- [x] 4.8 `aidlc-docs/taeels/aidlc-state.md` 갱신 · `audit.md` 에 답과 결과 추가

---

## 5. 낼 파일

```text
   aidlc-docs/taeels/construction/obs/nfr-requirements/nfr-requirements.md
     확장 열다섯 규칙의 적용 표 · 비-보안 NFR 다섯 축 · 재는 자리

   aidlc-docs/taeels/construction/obs/nfr-requirements/tech-stack-decisions.md
     새 의존 0 · Go 1.26 · pgx · 한도를 붙인다면 직접 짜는 근거
```

문서 루트가 `aidlc-docs/taeels/` 인 것은 `CLAUDE.md` 의 Construction layering 이다.
AI-DLC 규칙 문서의 `aidlc-docs/construction/{unit}/` 경로를 그것이 대신한다.

---

## 6. 이 단계가 안 하는 것

```text
   NFR Design 을 겸하지 않는다   여기는 무엇을 지켜야 하는가다.  어떻게는 다음
                                자리다 — 회차 계획은 NFR Design 도 SKIP 이다

   팩이 닫은 값을 다시 열지 않는다   2.1 의 표.  폴링 간격 · limit · 페이지네이션 ·
                                커버리지 하한 · TLS 제외

   기존 코드의 보안 사실을 안 고친다  SECURITY-01 · 03 · 07 은 기록만이다
                                (decisions 3절).  이 단계가 그 줄을 안 넘는다

   인프라를 설계하지 않는다        Infrastructure Design 은 SKIP 그대로다
                                (constraints 6절 · 새 인프라 0)

   게이트를 다시 정하지 않는다     obs 의 병합 조건은 business-rules 7.1 이
                                이미 못 박았다 — scene-gates 3절의 curl 두 줄 + CP0
```

---

## 7. 진행자에게 남기는 표시

```text
   ①  회차 실행 계획과 이 단계의 실행이 어긋난다.  execution-plan.md 와
      v1-run-dhseo/aidlc-state.md 의 SKIP 줄을 누가 고칠지는 진행자의 것이다.
      담당 문서(taeels/aidlc-state.md)는 이 계획이 정리한다

   ②  NFR Design 도 SKIP 이다.  물음 1 의 답이 B 나 C 면 「어떻게」가 남는다 —
      그 단계를 다시 볼지, 아니면 Code Generation 계획이 지고 갈지를 진행자가 정한다

   ③  물음 1 이 SECURITY-11 의 첫 판정이다.  obs 가 어떻게 닫든 그 판정은
      demo-back(W2)과 ui(W1·W3)에 그대로 이어진다 — 같은 데모 인스턴스의
      같은 공개 표면이다.  유닛마다 다르게 닫으면 표가 갈린다
```

---

## 8. 답이 부른 것 (2026-09-08)

답 **B · B · A**. 셋 다 obs 안으로 들어왔다. 산출물 둘을
`aidlc-docs/taeels/construction/obs/nfr-requirements/` 에 냈다.

### 8.1 답이 정한 값

```text
   1 = B   무인증 읽기 셋에 요청 한도.  SECURITY-11 을 준수로 닫는다
           전역 하나 · 초당 120 · 버스트 240 · 넘으면 429 + Retry-After: 1
           IP 로 안 나눈다 — 대회장 관객이 한 NAT 뒤라 정상 폴링이 막힌다
           직접 짠다 — golang.org/x/time/rate 는 go.mod 에 없고 새 의존 0 이다

   2 = B   runs_created_idx (created_at) 를 schema.sql 에 더한다
           ?work= 없는 기본 목록과 ?since= 가 그것을 탄다
           btree 는 역방향으로도 훑으므로 DESC 정렬이 그대로 인덱스를 탄다

   3 = A   읽기 셋 셋의 응답에 Cache-Control: no-store
           본문의 글자는 안 바뀌므로 mcp 의 글자 일치(CP5)와 무관하다
           셋이다 — getRun 도 두 홉으로 폴링에 들어간다
```

### 8.2 답이 넓힌 파일

`business-rules.md` 7.5 의 목록에 더한다.

```text
   internal/api/<한도>.go     신규 파일 하나 — 답 1=B          내부 · 단독 obs
   internal/store/schema.sql  CREATE INDEX 한 줄 — 답 2=B      접점 (queue 와 공유)
   internal/api/nodes.go      Cache-Control 한 줄 — 답 3=A     이미 목록에 있다
   internal/api/runs.go       같다                             이미 목록에 있다
   internal/api/api.go        getRun 의 같은 한 줄 · 한도 등록   이미 목록에 있다
```

**행렬이 넓어지지는 않는다.** 새 파일은 `internal/api` 안이고 그 패키지는 이미
접점이다. `schema.sql` 도 이미 접점으로 올려 두었다 (`business-rules.md` 7.5).

### 8.3 커버리지 — 답 1=B 가 비율을 나쁘게 하지 않는다

```text
   한도 파일이 열여섯 문장 남짓이고 DB 가 필요 없다.  전부 덮이면
   (328+16)/(403+16) = 82.1% 로 internal/api 가 하한에서 멀어진다.
   비용은 여유가 아니라 시험을 쓰는 손이다.
```

### 8.4 진행자에게 넘기는 것 여섯

`nfr-requirements.md` 6절이 정본이다. 무거운 셋 —

```text
   429 가 정본 코드 표에 없다 (mediator-api.md 1.2 는 여덟이다).
     Functional Design 이 이미 넷을 남겼고 이것이 다섯째다

   SECURITY-11 의 첫 판정이 이 유닛에서 났다.  demo-back(W2)과 ui 가 같은
     데모 인스턴스의 같은 공개 표면이므로 그 표를 이어 써야 한다

   전역 한도의 대가 — 한 클라이언트가 한도를 다 먹으면 나머지가 굶는다.
     IP 로 나누는 쪽이 CP10 을 직접 깨므로 이쪽을 수락했다
```
