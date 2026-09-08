# obs — 기술 선택

AI-DLC Construction · obs 유닛(W0 · CP1)의 NFR Requirements 산출물 하나다.
지켜야 하는 값은 `nfr-requirements.md`. 여기는 **무엇으로 짓는가**다.

**이 유닛에 고를 것이 거의 없다.** 브라운필드이고 팩이 스택을 고정했다. 그래서
이 문서는 고른 것보다 **안 들이는 것과 그 이유**가 길다 — 답 1=B 가 새 의존을
부를 수 있었기 때문이다.

---

## 1. 고정된 것 — 이 유닛이 안 고른다

```text
   언어 · 툴체인    Go 1.26 · toolchain go1.26.6            go.mod (실측)
   DB 드라이버      github.com/jackc/pgx/v5 v5.10.0         go.mod
   DB              PostgreSQL 17                            RETURNING OLD.* 가 18 부터라
                                                            CTE 를 쓴다 (domain-entities 8절)
   HTTP            표준 net/http · http.ServeMux            api.go:60
   JSON            표준 encoding/json                       고정 타입 바인딩
   설정            환경변수 (internal/config)               DATABASE_URL ·
                                                            ENODE_MEDIATOR_TOKEN 과 같은 모양
   로깅            표준 log/slog                            s.log · Store.Log
   시험            표준 testing + 진짜 Postgres             internal/api/api_test.go 의
                                                            newServer.  CI 가 서비스를 붙인다
```

`go.mod` 의 직접 의존이 셋이다 — `pgx/v5` · `golang.org/x/sys` · `yaml.v3`.
**이 유닛이 그 목록을 안 움직인다.**

---

## 2. 이 단계가 정한 것 하나 — 요청 한도를 직접 짠다

답 1=B 가 요청 한도를 obs 안으로 들였다. 그것을 무엇으로 짓는가가 이 유닛에서
유일하게 남은 기술 선택이다.

```text
   후보                              판정      이유
   ───────────────────────────────   ───────   ──────────────────────────────────
   golang.org/x/time/rate            버린다     go.mod 에 없다 (실측).  들이면
                                               「새 Go 의존 0」이 깨진다 —
                                               팩과 실행 계획이 함께 못 박은 값이다

   리버스 프록시 · 게이트웨이의 한도    버린다     constraints 6절이 리버스 프록시를
                                               제외 범주로 뺐다

   직접 짠다 (토큰 버킷 하나)          고른다     의존 0.  열여섯 문장 남짓.
                                               DB 가 필요 없어 httptest 로 전부 덮인다
```

**직접 짜는 것이 이 자리에서 싼 이유가 셋이다.**

```text
   ①  범위가 좁다        나누는 키가 없다 (전역 하나).  분산도 지속도 필요 없다 —
                        일회용 인스턴스의 프로세스 메모리 하나다

   ②  의존이 0 이다      SECURITY-10 이 새 의존마다 잠금 · 취약점 검사 · SBOM 을
                        요구한다.  안 들이면 지킬 것이 안 는다

   ③  시험이 싸다        순수 로직이라 DB 없이 돈다.  internal/api 의 커버리지
                        여유가 5.6 문장뿐인데 덮이는 문장은 그 여유를 안 먹는다
                        (nfr-requirements 4.5)
```

**자료구조와 미들웨어를 어디 거는가는 여기가 아니다** — 그것이 「어떻게」이고,
NFR Design 이 SKIP 이므로 Code Generation 계획이 진다. 값은
`nfr-requirements.md` 3.1 에 다 있다.

---

## 3. 안 들이는 것

```text
   TLS 라이브러리          constraints 6절 · decisions 3절.  「보안 이유로
                          internal/config 를 고치지 말 것」이 이름으로 막은 자리다

   비밀 관리자             decisions 3절 · ADR-015 1절.  토큰은 설정 파일에 산다

   캐시 (Redis 등)         읽기가 DB 질의 하나씩이고 답 3=A 가 오히려 캐시를 끈다.
                          중간 저장소를 들이면 observed_at 이 「관측된 한 시점」을
                          그친다

   마이그레이션 도구        schema.sql 이 멱등하게 돌고 있다.  「스키마가 굳으면
                          그때 넣는다」가 store.go 의 기존 주석이다

   메트릭 · 추적 라이브러리   constraints 6절이 운영을 뺐다.  SECURITY-14 를
                          해당 없음으로 적은 것과 같은 근거다

   검증 · 스키마 라이브러리   질의 인자 넷과 정책 값 하나뿐이다.  business-rules 2절이
                          규칙을 다 적었고 표준 라이브러리로 짧다
```

---

## 4. 차단 게이트에 미치는 영향

`constraints.md` 의 차단 게이트 다섯 중 이 유닛이 닿는 것은 커버리지 하나다.

```text
   게이트                          이 유닛                    실측 근거
   ─────────────────────────────   ───────────────────────    ─────────────────────
   crypto/tls T 심볼 상한 10        안 움직인다                go list -deps ./cmd/enodectl
   net/http  T 심볼 상한 50         안 움직인다                에 internal/api ·
                                                              internal/store ·
                                                              internal/runctl 이 없다
                                                              (실측 0건).  한도 파일은
                                                              internal/api 다

   패키지별 커버리지 하한 80%        걸린다 — internal/api     nfr-requirements 4.5

   허용목록 밖 스킵 상한 0          안 움직인다                한도 시험은 DB 가
                                                              필요 없어 t.Skip 이 없다

   U+2605 파일 수 상한 0            안 움직인다                glyphscan 통과
```

**심볼 상한 둘의 실측값은 `net/http` 6 · `crypto/tls` 1 이다**
(`business-rules.md` 7.4). 한도를 `internal/api` 에 짓는 것이 그 수를 못 움직이는
이유는 임포트가 아니라 **의존 그래프**다 — `cmd/enodectl` 이 `internal/api` 를
안 딛는다.
