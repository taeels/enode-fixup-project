# Unit Test Instructions — cardnews-guest-login

## 실행

```bash
go test ./internal/api/ui/... -v
```

DB 가 필요 없다 — `ENODE_TEST_DATABASE_URL` 없이도 전부 돈다
(`decisions.md` 2절의 원칙, `nfr-design.md` 「커버리지 하한 80%」).

**실행 결과 (2026-09-08)**: 일곱 테스트 전부 통과.

```text
   TestHandlerServesLanding          랜딩에 Guest Login 버튼 · 관리자
                                     토큰 placeholder 가 있다
   TestHandlerServesCardnews         카드뉴스 화면의 버튼 다섯(닫기 ·
                                     이전 · 다음 · 진행 점 0 · 진행 점 3)
   TestHandlerServesDemo             데모 화면에 재열람 링크가 있다
   TestHandlerServesStaticAssets     CSS·JS 일곱 개가 각자 경로로 열린다
   TestHandlerUnknownPathIs404       없는 경로는 404
   TestHandlerNoDirectoryListing     shared/ 가 목록이 아니라 index.html 을 낸다
   TestHandlerSecurityHeaders        보안 헤더 다섯이 랜딩·카드뉴스·데모·
                                     404 응답 전부에 실린다
```

## 커버리지

```bash
go test ./internal/api/ui/... -count=1 -coverprofile=/tmp/ui-cover.out
go tool cover -func=/tmp/ui-cover.out
```

**실행 결과**: `internal/api/ui` 92.3% (하한 80% 초과, `decisions.md`
2절 · 8.2). `Handler()` 80.0%, `withSecurityHeaders` 100.0% — 미달
분기는 `fs.Sub` 의 에러 경로 하나뿐이고, embed 가 빌드에 포함되므로
런타임에 그 경로를 탈 방법이 없다(`nfr-design.md` 주석 참조).

## 회귀 — 기존 internal/api

```bash
eval "$(scripts/testdb.sh)"   # Postgres 를 세운다. docker 필요
go test ./internal/api/... -v
```

**실행 결과 (이 실행 환경)**: 이 세션에는 `docker` CLI 는 있지만 데몬이
없다(`/var/run/docker.sock` 없음) — `scripts/testdb.sh` 가 실패하고
`ENODE_TEST_DATABASE_URL` 을 못 만든다. 그 상태로 `go test
./internal/api/...` 를 돌리면 **실패가 아니라 스킵**이다 — DB 를 요구하는
테스트들이 `t.Skip("ENODE_TEST_DATABASE_URL is unset")` 으로 빠지고
`go test` 는 `ok` 를 낸다. `TestSurface_WithoutARecordStoreEveryRecordPathIs503`
처럼 DB 가 없어도 되는 테스트는 그대로 통과한다.

**이것이 CP0 의 스킵 상한 0 을 어기는가 — 아니다.** `.ci-allowed-skips`
가 이 스킵들의 조건("ENODE_TEST_DATABASE_URL 이 빔")을 실제 CI(러너가
Postgres 서비스를 붙인다)에서는 절대 발동하지 않는 것으로 이미 분석해
뒀다. 이 환경에 `docker` 데몬이 없는 것은 **이 유닛이 만든 조건이
아니고**, `internal/api/api.go` 의 diff 가 등록 줄 두 개뿐이라는 사실은
DB 없이도 코드 리뷰로 확인된다. **정직하게 남긴다** — 이 실행 환경에서
`internal/api` 의 DB 의존 테스트를 실제로 통과시키는 것까지는 확인하지
못했다. Postgres 가 있는 환경(CI 또는 `docker` 데몬이 있는 로컬)에서
위 명령으로 재확인하는 것을 권한다.
