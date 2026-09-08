# Build and Test Summary — cardnews-guest-login

## 게이트 결과

| 게이트 | 결과 | 비고 |
|---|---|---|
| `go build ./...` | 통과 | |
| `go vet ./...` | 통과 | 지적 0 |
| `gofmt -l .` | 통과 | 출력 없음 |
| `go run ./scripts/glyphscan.go` | 통과 | 83개 파일, 장식 문자 0 |
| `go test ./internal/api/ui/...` | 통과 | 일곱 테스트, 커버리지 92.3%(하한 80%) |
| `go test ./internal/api/...` (회귀, 로컬 Postgres) | **통과, 스킵 0** | docker 데몬이 없어 네이티브 PostgreSQL 16 을 대신 세워 확인 |
| `enodectl.exe` 심볼 상한 | 해당 없음 | 이 유닛이 `cmd/enodectl` 을 안 건드린다 |
| `git status --porcelain` | 깨끗 | `probe.lock` 은 테스트 실행 중 건드려졌다가 매번 원복(기존 결함, 이 유닛과 무관) |
| CP8 실동작(브라우저) | **확인함** | 실제 Mediator 프로세스 + 실제 PostgreSQL + headless Chromium(Playwright)으로 18개 확인 항목 전부 통과. 스크린샷 다섯 장을 사용자에게 전달 |

## 이 세션에서 실제로 검증한 범위 (경과)

처음에는 `docker` 데몬과 브라우저가 없어 DB 의존 회귀와 CP8 실동작을
"코드 리뷰로만 확인했다"고 적었다. 이후 대안을 찾았다 —

- **DB**: 이 환경에 PostgreSQL 16 이 네이티브로 이미 설치돼 있었다.
  `service postgresql start` 로 띄우고 `scripts/testdb.sh` 와 같은
  자격증명으로 역할/DB 를 만들어 `internal/api` 전체를 스킵 없이
  돌렸다 — 전부 통과
- **브라우저**: 이 환경에 Playwright 와 Chromium 이 미리 설치돼
  있었다. 실제 Mediator 를 띄우고 headless Chromium 으로
  `scene-gates.md` CP8 의 절차 1~8 을 그대로 수행했다 — 18개 확인
  항목(랜딩 분리 · 첫 방문 진입 · 카드 넘김 · 종료 후 DOM 무잔존 ·
  재방문 생략 · 재열람 · 키보드 내비게이션 · 실제 네트워크 응답의
  보안 헤더) 전부 통과

**남은 것은 저장소 전체 테스트에서 나온, 이 유닛과 무관한 실패 셋뿐이다**
(`unit-test-instructions.md` 「참고」 절 — 이 실행 환경이 root 로 도는
것이 원인이고, `cmd/mediator` · `internal/record` 는 이 유닛이 안
건드린다).

## 커버리지 하한 재측정 (decisions.md 2절 · 8.2)

`internal/api/ui` — 92.3%. 새 패키지가 이 하한을 넘긴다.

## 다음 단계

이 문서로 유닛 `cardnews-guest-login` 의 Construction 이 끝난다.
남은 것은 `v1-run-dhseo-cardnews` 로의 병합과 `aidlc-docs/v1-run-dhseo-cardnews/aidlc-state.md` ·
`audit.md` 갱신(진행자 역할, `CONVENTIONS.md` 3.2 — 이 파일들은 유닛
브랜치가 안 싣는다)이다.
