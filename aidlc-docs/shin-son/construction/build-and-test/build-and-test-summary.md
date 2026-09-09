# Build and Test Summary — queue

## 게이트 결과 (2026-09-09 · Postgres.app 18.6 · 포트 55434)

| 게이트 | 결과 | 비고 |
|---|---|---|
| `go build ./...` · `go vet` · `gofmt -l` | 통과 | 출력 없음 |
| `go run ./scripts/glyphscan.go` | 통과 | 87 파일 · 장식 문자 0 |
| 크로스 빌드 넷 (linux/arm64 · linux/amd64 · windows/amd64 · darwin/arm64) | 통과 | |
| `go test ./internal/store ./internal/api ./cmd/mediator` | 통과 | 새 시험 열여덟 포함 |
| `go test ./... -coverpkg` + 80% 하한 awk (CP0) | 통과 | 열여섯 패키지 전부 · 전체 87.0% · `internal/api` 80.6% |
| 스킵 | 0 | `.ci-allowed-skips` 그대로 |
| `enodectl.exe` 심볼 상한 | 해당 없음 | 이 유닛은 `cmd/enodectl` 이 딛는 패키지를 안 만진다 (obs 실측) |
| **CP2 실동작** (실 Mediator + 실 노드 + `runctl`) | **통과** | 202 QUEUED · 목록·상세 · a 끝나며 같은 tx 에서 승격 · 422 FAILED |
| `git status --porcelain` | 깨끗 | `probe.lock` 은 시험이 건드렸다 매번 원복 (기존 결함 · 이 유닛 밖) |

## 무엇을 실제로 검증했나
- 시험은 이 기계의 진짜 PostgreSQL 위에서 돌았다 (NFR 답 3=A · 사용자가 Postgres.app 을 깔았다).
  판이 문서 기준(17)보다 새 18.6 이나 이 코드는 17 문법만 쓴다.
- CP2 는 `enode_cp2` 판을 따로 파고 Mediator · enode 를 이 기계에 띄워 `scene-gates.md`
  3절의 명령을 그대로 쳤다 (`integration-test-instructions.md` 의 실측 절).
- 흔들린 시험 넷은 이 유닛 밖이고 단독 통과했다 (`unit-test-instructions.md` 참고 절).

## 병합 조건 (CONVENTIONS 3.3 · roster §5)
CP0 회귀와 CP2 가 초록이다. `unit/queue` 를 PR 로 `main` 에 낸다. 접점(store · api ·
mediator · schema.sql 한 줄)은 진행자가 직렬로 병합한다.

## 진행자에게
- `internal/api` 커버리지 80.6% — 여유 3 문장.
- `WakeQueued` 의 반환이 `Woken`. drain(W2)은 `Cancel` 만 부르면 된다 — 안에서 깨운다.
- 대기 상한 없음(정본)의 수락 위험은 `nfr-requirements.md` 3절.
