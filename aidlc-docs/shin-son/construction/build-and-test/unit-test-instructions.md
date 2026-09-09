# Unit Test Execution — queue

## 진짜 Postgres 가 필요하다
`internal/store` 의 시험은 DSN 이 없으면 `t.Fatal`, `internal/api` 는 건너뛴다. 스킵 허용
목록은 비어 있으므로 CI 는 DSN 없이는 빨갛다. 값은 `docs/testdb-setup.md` — 역할 enode ·
비밀번호 enode · 판 enode_test · 포트 55434.

```bash
export ENODE_TEST_DATABASE_URL='postgres://enode:enode@127.0.0.1:55434/enode_test?sslmode=disable'
```

Docker 가 있으면 `eval "$(scripts/testdb.sh)"` 한 줄이 위와 같다.

## 이 유닛의 시험 열여덟
```bash
go test ./internal/store -run 'TestQueue_' -count=1 -v
go test ./internal/api -run 'TestSubmit_|TestSubmitRejectCodes|TestConcurrentSubmitAllOrNothing|TestRelease_OnceReleased' -count=1 -v
go test ./cmd/mediator -run 'TestWakeQueuedAtStart_' -count=1 -v
```

## 회귀 전체 + 커버리지 (CP0 · ci.yml:269 그대로)
```bash
go test ./... -count=1 -coverpkg=./... -coverprofile=/tmp/cover.out -json > /tmp/test.json
```
그 뒤 `.github/workflows/ci.yml` 의 「커버리지」 스텝 awk 를 `/tmp/cover.out` 에 대면 패키지별
80% 하한을 잰다. 2026-09-09 실측 — 열여섯 패키지 전부 통과 · 전체 87.0% · `internal/api` 80.6%.

## 판을 남과 안 나눈다
`internal/api` 가 시험마다 `TRUNCATE` 를 친다. 손으로 띄운 Mediator 와 같은 판을 쓰면
광고가 살아 있어 「함대에 없다(422)」가 조용히 201 이 된다 — CP2 는 `enode_cp2` 처럼
다른 판을 판다.

## 참고 — 흔들리는 시험 넷 (이 유닛 밖)
`cmd/enodectl` 의 `TestCmdStart_` 셋과 `cmd/iapadapter` 의
`TestAdapter_HandleNewWorkDoesNotSubmitBeforeTheFleetSeesTheNode` 는 1초 안에 프로세스가
서길 기다린다. 전체 병렬 실행에서 한 번 깨졌고 단독 재실행에서 통과했다. 이 유닛이
만지지 않은 패키지다.
