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
eval "$(scripts/testdb.sh)"   # docker 데몬이 없으면 로컬 Postgres 로 대체
go test ./internal/api/... -v
```

**처음 시도 (docker)**: 이 환경에는 `docker` CLI 는 있지만 데몬이 없다
(`/var/run/docker.sock` 없음) — `scripts/testdb.sh` 가 실패한다.

**대체 경로로 실제 확인함 (2026-09-08)**: 이 환경에 **PostgreSQL 16 이
네이티브로 이미 설치돼 있어** `service postgresql start` 로 띄우고,
`scripts/testdb.sh` 와 같은 자격증명(`enode`/`enode`)으로 `enode` ·
`enode_test` 역할과 데이터베이스를 만들었다. 그 뒤:

```bash
export ENODE_TEST_DATABASE_URL='postgres://enode:enode@127.0.0.1:5432/enode_test?sslmode=disable'
go test ./internal/api/... -v
```

**실행 결과**: **스킵 0. 전부 통과.** 이전에 DB 가 없어 스킵되던
테스트(`TestBlob_UnknownRunAndUnknownStepAreBoth404WithDifferentReasons`
등 십수 개, `TestClaim_WaitsOutTheWindowThenAnswers204` 포함)가 전부
돌아서 통과했다 — `.ci-allowed-skips` 의 분석("이 스킵은 CI 에서 발동
하지 않는다")이 실측으로 확인됐다.

## 참고 — 저장소 전체 테스트에서 남은 실패 셋 (이 유닛과 무관)

`go test ./...` 를 같은 DB 로 전체 돌리면 이 유닛과 무관한 실패가
셋 남는다 — 전부 **이 실행 환경이 root 로 도는 것**이 원인이다.

```text
   cmd/mediator  TestRun_StopsAtTheFirstThingItCannotDo
                 (a config with no token...)          root 는 0500
   cmd/mediator  TestSetup_ConfigCannotBeWritten...    디렉터리에도 쓸 수
                                                       있어 "못 쓴다" 를
                                                       전제한 테스트가 깨진다
   internal/record  TestSealMakesItImmutable           같은 이유 — 봉인은
                                                       파일 권한(0400 등)
                                                       으로 막는데 root 는
                                                       그 권한을 무시한다
```

`.ci-allowed-skips` 의 "root 라 권한이 무의미(1곳)" 항목과 같은
종류의 원인이다(그 항목은 스킵 하나만 짚었지만, 실측해 보니 같은
원인으로 실패하는 자리가 실제로는 셋이다). 이 유닛은 `cmd/mediator` 도
`internal/record` 도 안 건드리므로 **이 실패들은 이 유닛이 만든 것이
아니다** — 비루트 CI 러너에서는 셋 다 원래대로 통과한다.
