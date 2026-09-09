# queue — 기술 선택

AI-DLC Construction · queue 유닛(W1 · CP2)의 NFR Requirements 산출물 하나다.
지켜야 하는 값은 `nfr-requirements.md`. 여기는 **무엇으로 짓는가**다.

**고를 것이 하나다** — 큐 변경을 직렬화하는 잠금. 나머지는 브라운필드와 팩이
고정했다.

---

## 1. 고정된 것 — 이 유닛이 안 고른다

```text
   언어 · 툴체인    Go 1.26 · toolchain go1.26.6            go.mod
   DB 드라이버      github.com/jackc/pgx/v5 v5.10.0         go.mod
   DB              PostgreSQL 17                            docs/testdb-setup.md · obs 의 CTE 선택
   HTTP            표준 net/http · ServeMux                  api.go
   JSON            표준 encoding/json
   로깅            표준 log/slog                             s.log · Store.Log
   매처            internal/match.Match — 순수 함수 · 무변경   ADR-014 결정 3
   시험            표준 testing + 진짜 Postgres               답 3=A 로 이 기계에 선다
```

`go.mod` 의 직접 의존 셋 — `pgx/v5` · `x/sys` · `yaml.v3`. **이 유닛이 안 움직인다.**

---

## 2. 이 단계가 정한 것 하나 — 잠금은 PostgreSQL advisory lock

큐 변경(넣기 · 훑기 · 승격)을 직렬화해야 한다 (FD rules §4 · 경쟁 창).

```text
   후보                                   판정      이유
   ────────────────────────────────────   ───────   ─────────────────────────────────────
   Go 의 sync.Mutex (프로세스 안)          버린다     경계가 프로세스다.  DB 가 직렬화의 주인이어야
                                                    tx 와 같이 풀린다.  I5 가 tx 로 얻어지는 것과
                                                    같은 자리 — 저장 계층이 막는다 (ADR-019)
   runs 행 잠금 (SELECT ... FOR UPDATE)    부족하다   훑기끼리는 직렬화되나, 「임대 삭제 뒤 훑기」와
                                                    「넣기 뒤 훑기」가 아직 없는 행을 못 잠근다 —
                                                    창이 그대로다.  FOR UPDATE 는 훑기 안에서
                                                    행 보호로만 쓴다
   큐 전용 표 한 행을 잠근다                버린다     표 하나가 는다.  스키마 변경이 하나 더 되고
                                                    잠금 행이 「데이터」로 보인다
   pg_advisory_xact_lock(key)             고른다     SQL 함수 하나.  의존 0.  tx 끝에 저절로 풀린다 —
                                                    누수 없음 · 오류 경로 정리 없음 (SECURITY-15).
                                                    인스턴스가 늘어도 DB 안의 것이라 유지된다
```

**세션 잠금(`pg_advisory_lock`)이 아니라 xact 잠금인 이유** — 세션 잠금은 풀어 주는
호출이 필요하고, 오류 경로에서 빠뜨리면 큐가 영영 선다. xact 잠금은 커밋 · 롤백이
푼다. 그것이 「자원 정리를 오류 경로에서」를 코드가 아니라 DB 가 지게 하는 방법이다.

**키** — int64 상수 하나. `store.go` 에 「이 저장소의 advisory lock 키」 주석 목록을
두고 첫 항목으로 등재한다. 값은 Code Generation 이 정한다(충돌만 없으면 된다 —
오늘 다른 키가 0 이다).

**pgxpool 과의 관계** — xact 잠금은 tx 의 커넥션에 묶인다. `WakeQueued(ctx, tx)` 는
부르는 tx 의 커넥션을 쓰므로 새 커넥션이 0 이고, `WakeQueuedNow` 만 자기 tx 하나다.
풀 기본값(`max(4, NumCPU)`)을 안 만진다.

---

## 3. 부분 인덱스 (답 1=A)

```sql
CREATE INDEX IF NOT EXISTS runs_queued_idx ON runs (created_at) WHERE state = 'QUEUED';
```

`schema.sql` 의 기존 규약 그대로 — `IF NOT EXISTS` · 멱등 · 주석에 「왜」. 훑기의
첫 줄(`WHERE state = 'QUEUED' ORDER BY created_at, run_id`)이 이 인덱스를 타고,
승격 UPDATE 가 행을 인덱스에서 뺀다. 마이그레이션 도구는 안 들인다(obs 와 같다).

---

## 4. 안 들이는 것

```text
   메시지 큐 · 작업 큐 라이브러리    큐는 runs 표의 상태 하나다 (ADR-064).  중간 저장소를 들이면
                                  「재기동 후에도 QUEUED 가 참」이 두 곳의 참이 된다
   주기 스케줄러                   깨우기는 지점이다 (decisions 1절).  reaper 가 이미 있고 거기 안 얹는다
   분산 잠금 라이브러리 (etcd 등)    DB 가 하나이고 그 안의 잠금으로 족하다
   lock_timeout · statement_timeout  저장소에 없는 설정을 이 유닛이 첫 번째로 들이지 않는다.
                                  임계 구간이 ms 라 필요가 안 보인다 — 보이면 그때 값으로
   메트릭 · 추적                   constraints §6 · SECURITY-14 해당 없음과 같은 근거
   property-based testing         확장이 꺼져 있다 (decisions 1절).  FIFO 순서는 결정적 시험으로 잡는다
```

---

## 5. 차단 게이트에 미치는 영향

```text
   게이트                          이 유닛                        근거
   ─────────────────────────────   ─────────────────────────────  ───────────────────────────
   crypto/tls T 심볼 상한 10        안 움직인다                    cmd/enodectl 이 internal/store ·
   net/http  T 심볼 상한 50         안 움직인다                    internal/api 를 안 딛는다 (obs 실측)
   패키지별 커버리지 하한 80%        걸린다 — internal/store ·      새 문장은 전부 DB 시험으로 덮는다
                                   internal/api · cmd/mediator
   허용목록 밖 스킵 상한 0          안 움직인다                    DB 없으면 t.Fatal.  스킵 0 그대로
   U+2605 파일 수 상한 0            안 움직인다                    glyphscan
```

**커버리지의 실측은 Code Generation 이 한다** — 이 기계에 Postgres 가 서는 것(답 3=A)이
그 앞이다.
