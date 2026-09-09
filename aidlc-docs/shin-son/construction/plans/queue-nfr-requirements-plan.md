# queue — NFR Requirements 계획

AI-DLC Construction · **queue 유닛**(W1 · CP2)의 NFR Requirements Part 1 이다. 요구
본문이 아니라 **무엇을 재고 무엇을 물을지의 계획**이다. 답이 오면 그 값으로
`aidlc-docs/shin-son/construction/queue/nfr-requirements/` 아래 산출물 둘을 낸다.

정본 — 팩 `requirements/`(decisions 1절·2절 권장값 · 3절 보안 확장 취급 · constraints
차단 게이트 다섯) · 확장 `security-baseline.md`(차단성) · 이 유닛의 Functional Design
산출물 셋 · obs 의 NFR 산출물 둘(같은 표를 잇는다 —
`aidlc-docs/taeels/construction/obs/nfr-requirements/`).

---

## 0. 이 단계를 도는 이유

회차 실행 계획은 NFR Requirements 를 SKIP 으로 승인했고, obs 가 사용자 지시로 다시
들였다(obs 계획 0절). queue 도 같은 자리에 선다 — **보안 확장이 켜져 있어 규칙
열다섯의 판정을 유닛마다 비워 둘 수 없고**, Functional Design 이 끝난 지금 팩이
값으로 안 닫은 자리가 셋 남아 있다(3절). 짧게 돈다 — 새 HTTP 표면이 없어 obs 보다
물을 것이 적다.

## 1. 착수 전에 실측한 것 (기존 코드)

```text
   확인한 사실                                                       자리
   ────────────────────────────────────────────────────────────────  ─────────────────────────
   runs 의 인덱스는 둘 — runs_work_idx (work_id, created_at) ·        schema.sql (obs 가 둘째를 더함)
     runs_created_idx (created_at).  state 열 인덱스는 없다
   advisory lock 을 쓰는 자리가 저장소에 없다 (grep 0건)               internal/
   claim 의 SELECT ... FOR UPDATE SKIP LOCKED 가 유일한 행 잠금이다     claim.go (ADR-015 §3)
   pgxpool 은 기본값 — max(4, NumCPU).  이 유닛이 안 바꾼다             store.go:66
   statement_timeout · lock_timeout 설정이 없다                         internal/config · store.Open
   reaper 는 실패를 Error 로그로 남기고 계속 돈다                        reap.go RunReaper
   기동 경로의 실패는 전부 return 1 (os.Exit 아님) — defer 가 돈다        cmd/mediator/main.go run()
   go.mod 직접 의존 셋 — pgx/v5 · x/sys · yaml.v3.  새 의존 0            go.mod
   패키지별 커버리지 하한 80% 는 awk 가 집행한다.  스킵 허용 0            ci.yml:269 · .ci-allowed-skips
   internal/store 의 시험은 DB 없으면 t.Fatal 이다                       reap_deterministic_test.go
   이 기계에 Postgres · docker · brew 가 없다                            실측 2026-09-09
```

## 2. 이미 값으로 닫힌 것 — 다시 묻지 않는다

### 2.1 팩과 정본이 진 값

```text
   항목                   값                                  근거
   ────────────────────   ─────────────────────────────────   ──────────────────────────
   대기 상한 · 타임아웃     없다.  출구는 cancel 뿐               decisions 1절 · ADR-064 §5
   공정성                 FIFO 하나.  정책 틀 없음              ADR-064 §2.1 · constraints §2
   깨우기의 동기성          부르는 tx 안에서 동기                 decisions 1절 (CP2 의 경쟁 조건)
   크래시 복구             기동 시 한 번 훑는다                   ADR-064 §6 · decisions 2절
   새 Go 의존              0.  Go 1.26 고정                     실행 계획 「기술 스택 고정」
   커버리지 하한           패키지별 80%                          ci.yml:269
   TLS · 비밀 관리자 · HA   안 넣는다                             constraints §6 · decisions 3절
   보안 규칙 01·03·07·08·12  decisions 3절의 취급 그대로           decisions 3절
```

### 2.2 이 계획이 채우는 권장값 — 물음으로 안 올린다

**벗어나면 근거를 적는다.** obs 계획 2.2 와 같은 규약이다.

```text
   한 훑기의 행 수 상한      없다.  QUEUED 전부를 도착순으로 본다
                           근거 — 「대기 상한 없음」과 「FIFO 전체 훑기」가 정본이다.
                           상한을 두면 뒤쪽 Run 이 자원이 있어도 다음 지점까지 선다.
                           비용은 행당 match.Match 한 번(순수 함수 · 광고 수 x 요구 수)이고
                           대기 행은 사람 손이 낸 수다 — 데모에서 열을 안 넘는다

   advisory lock 대기       막고 기다린다 (pg_advisory_xact_lock).  try-lock 으로 건너뛰지 않는다
                           근거 — 건너뛰면 「풀렸는데 못 보는 창」이 다시 열린다.  임계 구간은
                           훑기 하나(ms 단위)라 정산·취소 요청이 그 뒤에 서는 대가가 작다.
                           lock_timeout 을 새로 안 건다 — 저장소에 그 설정이 없고 claim 의
                           롱폴이 같은 자리에서 같은 선택을 했다

   잠금 키                  int64 상수 하나.  값은 Code Generation 이 정하되 store.go 에
                           주석으로 「이 저장소의 advisory lock 키 목록」을 두어 다음 키가
                           부딪치지 않게 한다.  오늘은 하나뿐이다

   승격 로그                Info 한 줄 — "promoted from queue" run 만.  대기 진입도 Info 한 줄.
                           계약 본문 · principal 을 안 찍는다 (SECURITY-03 · obs 와 같은 규칙)

   커넥션 풀                기본값 그대로.  훑기는 부르는 tx 의 커넥션을 쓰므로 새 커넥션 0.
                           WakeQueuedNow 만 자기 tx 하나 — 기동 · Reap · drain 해제뿐이다

   가용성 수치 목표          안 적는다 (obs 계획 2.2 와 같은 근거 — 단일 Mediator · HA 제외).
                           대신 실패 모드와 그때의 상태를 적는다 (FD business-logic §5)

   접근 로그 · 메트릭        안 남긴다.  decisions 3절 「기존 코드 사실로 기록만」
```

---

## 3. 물음 셋

답은 `[Answer]:` 뒤에 글자 하나로 적는다. 셋 다 **되돌리는 값이 크거나 남의 손이
필요한 것**이다.

### 물음 1
`wakeQueuedIn` 의 첫 줄은 `SELECT ... FROM runs WHERE state = 'QUEUED' ORDER BY
created_at, run_id FOR UPDATE` 다. 임대가 지워지는 **모든** 지점(종료 · 취소 · 회수 ·
부분 반납)에서 돈다 — 대기가 0 인 함대에서도 매 종료마다 한 번씩이다. `runs` 에
`state` 인덱스가 없어 그 한 줄이 표 전체를 훑는다. 데모는 수백 행이라 안 보이지만,
실 함대에서 `runs` 는 지우지 않고 쌓인다.

A) `runs_queued_idx ON runs (created_at) WHERE state = 'QUEUED'` — 부분 인덱스
   한 줄을 `schema.sql` 에 더한다. 대기가 0 이면 인덱스가 비어 빠른 길이 O(1) 이고,
   대기가 있으면 도착순이 인덱스 순서다. 크기는 대기 행 수만큼이다. `schema.sql` 은
   접점이라 진행자 직렬 병합에 한 줄이 는다 (권장)

B) 안 더한다. 임계 행 수를 산출물에 적고 미룬다 — `runs` 가 만 행을 넘으면 그때
   붙인다. `schema.sql` 을 이 유닛이 안 만진다

C) `state` 전체 인덱스 `runs_state_idx ON runs (state, created_at)`. 목록의
   `?state=` 필터(obs)도 같이 탄다. 다만 종료 행이 대부분이라 크기가 표만큼 자란다

X) Other (please describe after [Answer]: A tag below)

[Answer]:

### 물음 2
기동 시 `WakeQueuedNow` 를 한 번 부른다(`main.go` · migrate 뒤). 그것이 실패하면
(DB 오류 — migrate 가 방금 성공했으므로 드물다) 어떻게 하나. SECURITY-15 의
「fail closed」와 「기동이 안 깨졌다」(CP0)가 서로 다른 쪽을 가리킨다.

A) Error 로그 한 줄을 남기고 기동을 계속한다. 대기 Run 은 다음 임대 해제 지점이나
   다음 Reap 에서 다시 본다 — 훑기가 못 돌아도 상태는 참이고(QUEUED 는 아무것도
   안 쥔다) 잃는 것은 지연뿐이다. reaper 의 실패 처리와 같은 결 (권장)

B) 기동을 중단한다(`return 1`). DB 가 방금 마이그레이션에 답했는데 훑기에 답을 못
   한 것은 이상 상태이므로 사람이 봐야 한다

C) A 에 더해 Reap 의 매 주기 끝에도 `WakeQueuedNow` 를 건다 — 기동 훑기가 빠져도
   주기(임대 갱신 주기)로 잡힌다. 「주기에 얹지 않는다」(decisions 1절)를 좁게 벗어난다

X) Other (please describe after [Answer]: A tag below)

[Answer]:

### 물음 3
`internal/store` 와 `internal/api` 의 시험이 진짜 Postgres 를 요구하고, 이 기계에는
Postgres · docker · brew 가 없다. Code Generation 의 시험과 CP0 회귀를 여기서 돌리려면
DB 가 있어야 한다. `docs/testdb-setup.md` 가 값(역할 enode · 판 enode_test · 포트 55434)을
정했고 「판을 남과 안 나눈다」를 못 박았다.

A) 사용자가 직접 깐다 — `docs/testdb-setup.md` 4절. brew 가 없으므로 Postgres.app
   또는 EDB 설치기로 17 을 깔고 포트 55434 · 역할 · 판을 맞춘 뒤, 이 세션에
   `ENODE_TEST_DATABASE_URL` 을 준다. 시스템 설치는 사용자 손이다 (권장)

B) 에이전트가 이 기계에 깐다 — 방법(Homebrew 설치 포함)과 권한을 사용자가 허락한다.
   시스템 변경이라 그 허락이 먼저다

C) 다른 기계의 Postgres 를 쓴다 — 서버는 나눠도 판은 내 것(`enode_test_shinson`)으로
   따로 판다. 사용자가 DSN 을 준다

X) Other (please describe after [Answer]: A tag below)

[Answer]:

---

## 4. 실행 계획 — 완료 (2026-09-08T23:59:38Z · 답 A · A · A)

- [x] 4.1 물음 셋의 답을 받는다. 모호하면 확인 물음 파일을 따로 낸다
- [x] 4.2 SECURITY 열다섯 규칙을 queue 의 산출물에 하나씩 대고 준수 · 해당 없음 · 조건부를
      가른다. obs 의 표를 잇는다 — 같은 규칙에 다른 판정을 내면 근거를 적는다
- [x] 4.3 비-보안 NFR 을 축 다섯으로 적는다 — 성능 · 확장 · 가용 · 신뢰 · 유지보수.
      값마다 근거와 재는 자리를 함께
- [x] 4.4 기술 스택 결정 — 새 의존 0 · pgx 의 advisory lock 은 SQL 함수라 의존이 안 는다 ·
      안 들이는 것
- [x] 4.5 산출물 둘을 낸다 (5절)
- [x] 4.6 정합 검사 — `emphasis-check.py` · `scripts/glyphscan.go`
- [x] 4.7 `aidlc-docs/shin-son/aidlc-state.md` 갱신 · `audit.md` 에 답과 결과 추가

## 5. 낼 파일

```text
   aidlc-docs/shin-son/construction/queue/nfr-requirements/nfr-requirements.md
     확장 열다섯 규칙의 적용 표 · 비-보안 NFR 다섯 축 · 재는 자리 · 진행자에게 넘기는 것

   aidlc-docs/shin-son/construction/queue/nfr-requirements/tech-stack-decisions.md
     새 의존 0 · advisory lock · 안 들이는 것 · 차단 게이트 다섯에 미치는 영향
```

## 6. 이 단계가 안 하는 것

```text
   NFR Design 을 겸하지 않는다      어떻게는 Code Generation 계획이 진다 (obs 와 같다)
   팩이 닫은 값을 다시 열지 않는다    2.1 의 표
   기존 코드의 보안 사실을 안 고친다  SECURITY-01 · 03 · 07 은 기록만 (decisions 3절)
   인프라를 설계하지 않는다          Infrastructure Design 은 SKIP (constraints §6)
   게이트를 다시 정하지 않는다        CP2 의 명령은 scene-gates 3절 그대로
```
