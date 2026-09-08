# Build and Test Summary — cardnews-guest-login

**2026-09-08 갱신** — 카드 1~4 일러스트, 카드 5(영상) 추가 뒤 재검증한
결과를 아래 두 절에 더한다. 최초 검증(원격 세션, Mediator+PostgreSQL+
Playwright 환경)의 결과는 그대로 남기고, 이번 로컬 환경(PostgreSQL
없음)에서 다시 확인한 항목만 갱신한다.

## 게이트 결과 (최초 검증 — 원격 세션, 2026-09-08 이른 시각)

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

## 게이트 재검증 (로컬 세션, 2026-09-08, 카드 일러스트·영상 추가 뒤)

| 게이트 | 결과 | 비고 |
|---|---|---|
| `go build ./...` | 통과 | |
| `go vet ./...` | 통과 | 지적 0 |
| `gofmt -l .` | **환경 요인으로 실패** | 이 유닛이 안 건드린 파일(`ui.go` 포함 저장소 전체)도 전부 걸린다 — Windows 체크아웃의 CRLF 개행 특성이지 이 회차의 diff 가 만든 문제가 아니다. `.gitattributes` 로 고칠 문제이나 이 유닛의 스코프 밖 |
| `go run ./scripts/glyphscan.go` | 통과 | 84개 파일(임시 프리뷰 바이너리 포함), 장식 문자 0 |
| `go test ./internal/api/ui/...` | 통과 | 일곱 테스트 그대로(테스트 수는 안 늘었다 — 기존 테스트에 새 `data-testid`/자산 경로를 추가), 커버리지 92.3% 유지 |
| `go test ./internal/api/...` (회귀, DB 의존) | **재실행 안 함** | 이 로컬 환경에 PostgreSQL 이 없다(원격 세션의 네이티브 설치와 다른 기계). 이번 회차의 diff 는 `internal/api/ui` 와 정적 자산에만 있고 `internal/api/api.go` 등 DB 의존 경로는 손대지 않았으므로 위험은 낮지만, **이 로컬 세션이 직접 재확인하지는 못했다** — 병합 전 CI 또는 DB 가 있는 환경에서 한 번 더 돌리길 권한다 |
| CP8 실동작(브라우저) | **재확인함** | `scene-gates.md` CP8 자신이 "함대도 Mediator 토큰도 필요 없다. 정적 파일 서버(mediator 나 standalone 바이너리)만 띄우면 된다" 고 명시하므로, `internal/api/ui.Handler()` 만 올린 standalone 바이너리 + 실제 브라우저로 여섯 확인 항목(사생활 창 상태 · 버튼 분리 · Guest Login 클릭 시 카드뉴스 경로 · 끝까지 넘기면 데모 경로+DOM 무잔존 · 재방문 생략 · `enode.guest.onboarded` 삭제 후 재노출)을 전부 다시 통과시켰다. 다섯 장(그림 넷 + 영상 하나) 모두 정상 표시·재생을 확인했다

## 최초 검증 세션에서 실제로 검증한 범위 (경과, 원격 세션)

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

유닛 `cardnews-guest-login` 은 이미 `v1-run-dhseo-cardnews` 로
병합됐다(2026-09-08, 최초 검증 세션). 이번 갱신은 병합 뒤에 그 위에
쌓인 콘텐츠 추가(일러스트 · 영상 · 문구)를 뒤늦게 Build and Test 로
다시 확인한 것이다. 남은 것 — 병합 전 CI 또는 DB 가 있는 환경에서
`internal/api/...` 회귀를 한 번 더 돌려 위 표의 "재실행 안 함" 을
닫는 것.
