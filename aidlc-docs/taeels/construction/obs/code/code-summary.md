# obs — 코드 요약

AI-DLC Construction · **obs 유닛**(W0 · CP1 · 토대)의 Code Generation Part 2 산출물이다.
코드는 저장소 루트에 있고 여기는 **무엇이 어디에 섰고 무엇으로 쟀는가**다.
계획은 `aidlc-docs/taeels/construction/plans/obs-code-generation-plan.md`.

---

## 1. 만진 파일

계획 1절의 세 갈래를 병렬로 짰다. **파일이 하나도 안 겹친다.**

```text
   갈래   파일                              무엇
   ────   ───────────────────────────────   ──────────────────────────────────────
   A      internal/contract/advert.go        Policy{Drain} · Advert.Policy
          internal/store/schema.sql          ALTER 둘 + CREATE INDEX 하나
          internal/store/store.go            Run.Submitter · INSERT 둘 ·
                                             UpsertAdvert(CTE) · DrainPolicy
          internal/store/observe.go          NodeView · LeaseView · Nodes ·
                                             RunRow · RunFilter · Runs ·
                                             StepView.Chosen · RequireView · RequiresOf
          internal/store/observe_obs_test.go 신규 · 시험 열셋

   B      internal/config/config.go          Demo · ENODE_DEMO_MODE
          internal/api/ratelimit.go          신규 · 토큰 버킷 하나
          internal/api/nodes.go              신규 · getNodes
          internal/api/runs.go               신규 · getRuns · 질의 파싱 넷
          internal/api/api.go                등록 · read 래퍼 · submitter ·
                                             advertResponse.Drain · getRun 의 requires
          internal/api/observe_test.go       신규
          internal/api/ratelimit_test.go     신규 · DB 없음

   C      internal/runctl/client.go          RunsQuery · Nodes · Runs ·
                                             Run.Requires · Step.Chosen · Step.StartedAt
          internal/runctl/client_test.go     시험 일곱 · DB 없음
```

`internal/api/api_test.go` 는 **안 열었다** — 3,402줄이고 모든 유닛이 딛는다.

---

## 2. 계획이 정한 셋이 코드에서 어떻게 섰나

```text
   store.DrainPolicy      순수 함수로 내보냈다.  store 가 쓸 때 접고 로그를 남기고,
                          api 는 접기만 불러 응답에 싣는다.  응답의 drain 이
                          저장된 값과 언제나 같은 것을 같은 함수가 보장한다

   RequireView 는 store   internal/api 의 여유가 5.6 문장이라 사상 루프를 그쪽에
                          안 얹었다.  결과로 api 가 82.9% 가 됐다 (4절)

   조건부 래퍼            read() 하나가 모드에 따라 s.auth 나 s.limit 로 감싼다.
                          등록 줄이 모드와 무관하게 같아 CP0 의 라우트 검사가
                          데모 모드에서도 참이다.  실측 15 -> 17 (새 둘만 늘었다)
```

---

## 3. 실측 — 게이트와 재는 자리

### 3.1 CP0

```text
   go build ./...                    OK
   go vet ./...                      OK
   go test ./...                     16개 패키지 전부 ok
   go run ./scripts/glyphscan.go     86 파일 · 장식 문자 0
   gofmt -l .                        비어 있다
   스킵                              0 (.ci-allowed-skips 의 실제 항목도 0)
   패키지별 커버리지 80% 하한          16개 전부 통과 · 미달 0
   GOOS=windows 빌드                 OK
   net/http T 심볼                   6 (상한 50)
   crypto/tls T 심볼                 1 (상한 10)
   cmd/enodectl/probe.lock           커버리지 실행 뒤 되돌렸다 (기존 결함)
```

### 3.2 CP1 — `scene-gates.md` 3절의 curl 두 줄

실 Mediator 를 띄우고 합성 함대(노드 넷)로 쟀다. **실 데몬이 아니므로
부분 측정이다** — 3.4 가 그 한계를 적는다.

```text
   첫째 줄   GET /v1/nodes
             노드 넷이 나온다                                    초록
             임대된 노드에 lease 가 있다 (n4 · r-running)         초록
             not_after 가 하트비트마다 앞으로 간다 (+3.0s 실측)     초록
             draining 노드가 값을 낸다 (n3 · "graceful")          초록
             임대 없는 노드는 null 이다 — 빈 객체가 아니다          초록

   둘째 줄   GET /v1/runs?limit=5
             상태 셋이 나온다 (RUNNING · SUCCEEDED · FAILED)      초록
             최신이 앞이다 (r-ok · r-reject · r-running)          초록
             submitter 키가 모든 행에 있다 (값은 "")               초록  CP9 지분
             거절된 Run 의 assigned 가 [] 다 — null 이 아니다      초록  CP2 재료
```

**`not_after` 를 재다 한 번 잘못 읽었다.** 최초 발급은 `TTLSeconds`(3600)이고
하트비트 갱신은 `RenewSeconds x NotAfterFactor`(180)라 공식이 다르다. 둘을
비교하면 시각이 앞당겨진 것처럼 보인다. **갱신끼리 비교해야 한다** — 기존
동작이고 이 유닛이 바꾼 것이 아니다. 게이트를 집행하는 사람이 같은 자리를
밟을 수 있어 적어 둔다.

### 3.3 NFR Requirements 5절의 재는 자리 셋

```text
   요청 한도   데모 인스턴스에 400 요청 -> 200 이 263 · 429 가 137 ·
              429 마다 Retry-After: 1
              실 함대 인스턴스에 같은 400 요청 -> 200 이 400 · 429 가 0
              「무인증 경로에만」이 그 둘째 줄로 서 있다

   인덱스     EXPLAIN (ANALYZE) · runs 5만 행
              ?work= 없는 목록   Index Scan Backward using runs_created_idx
                                Seq Scan 도 Sort 도 없다
              ?since=           같은 인덱스 · Index Cond 로 걸린다
              ?work=            기존 runs_work_idx 그대로 (접기 전 열에 등호를
                                건 결정이 값을 했다)

   캐시 헤더   Cache-Control: no-store 가 읽기 셋 셋 다에 실린다
              /v1/nodes · /v1/runs · /v1/runs/{id}
```

### 3.4 함께 확인한 것

```text
   정책 왕복        POST /v1/nodes 응답의 drain 이 언제나 실린다 ("" 도 나온다)
                   policy.drain=graceful 이 그대로 돌아온다
                   어휘 밖 값(bogus-value)은 "" 로 접힌다

   SECURITY-03     접을 때의 로그가 node=n4 만 찍는다.  원문 bogus-value 의
                   등장 횟수 0 (실측)

   requires        getRun 에만 실리고 attrs 로 감싸여 나온다
                   제출 응답에는 없다 — view() 를 안 넓혔다 (has("requires") = false)

   질의 검증        limit=0 · -1 · abc -> 400 invalid limit
                   limit= (빈 값) -> 기본 100 · limit=99999 -> 자른다
                   since=notatime -> 400 invalid since
                   GET /v1/nodes?t=123 -> 400 unknown query parameter
                   GET /v1/runs?bogus=1 -> 무시한다

   데모 스위치      ENODE_DEMO_MODE=1 에서 읽기 셋 셋이 200 (무인증)
                   POST /v1/runs · POST /v1/nodes · GET /v1/asks 는 401 그대로
                   기본값(거짓)에서는 읽기 셋 셋이 401 — 실 함대 무변경
```

**CP1 이 통째로 초록인 것은 아니다.** 화면 다섯이 ui 몫이고 상태 넷째(`QUEUED`)는
queue(W1)가 만든다 (`business-rules.md` 7.1). 그리고 위 측정은 **합성 함대**다 —
실 데몬 여럿이 45초마다 광고하는 함대에서 다시 재는 것은 게이트 집행자의 몫이다.

---

## 4. 커버리지

```text
   패키지            전                    후                    여유
   ───────────────   ───────────────────   ───────────────────   ──────────
   internal/api      328/403 = 81.4%       392/473 = 82.9%       5.6 -> 13.6
   internal/store    1232/1501 = 82.1%     1310/1588 = 82.5%     31.2 -> 39.7
   internal/runctl   78/81 = 96.3%         100/103 = 97.1%       13.2 -> 17.6
   internal/config   -                     103/121 = 85.1%       6.2
```

**병목이던 `internal/api` 가 하한에서 멀어졌다.** 새 문장 70 중 안 덮인 것이
여섯이고 전부 `503 query failed` 분기다 — 기존의 안 덮인 503 분기와 같은 모양이다.
허용치는 `0.2 x 70 + 5.6 = 19.6` 이었다.

전체는 5600/6396 = 87.6% 이고 미달 패키지가 0 이다.

---

## 5. 알아 둘 것 둘

```text
   gofmt 의 doc comment 포매터가 주석 안의 '' 를 타이포그래픽 따옴표로 바꾼다.
   SQL 의 빈 문자열 리터럴을 주석에 쓸 때 "" 로 적어야 한다.
   저장소 전체에 걸리는 함정이라 여기 적는다

   시험 격리는 Truncate 가 아니라 scratchDB 다.  internal/api 의 스위트가
   기본 DB 를 Truncate 로 비우므로 store 의 새 시험은 판을 따로 판다 —
   reap_deterministic_test.go 가 같은 이유로 이미 그 모양이다
```

---

## 6. 진행자에게 넘기는 것

**이 단계가 새로 더하는 것은 없다.** Functional Design 여섯(계획 8.5)과
NFR Requirements 여섯(`nfr-requirements.md` 6절)이 그대로 산다 — 코드가 그
표시들을 닫지 않는다.

다만 실행에서 확인된 것 하나를 덧붙인다.

```text
   scripts/testdb.sh 가 Docker 를 강제한다.  대회장에서 넷이 Docker 를 못 깔면
   CP0 의 첫 줄이 안 서고, 그러면 아무도 병합 자격을 못 갖춘다.
   이 파일은 obs 의 행렬 밖이고 모든 유닛이 딛는 공용이라 이 유닛이 안 고쳤다.
   격리 단위가 「사람마다 기본 데이터베이스 하나」라는 것도 함께 넘긴다 —
   한 DB 를 여럿이 쓰면 internal/api 의 Truncate 가 남의 픽스처를 지운다
```
