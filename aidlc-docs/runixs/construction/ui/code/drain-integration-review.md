# Drain 인수와 기본 주소 접속 보완

2026-09-09, runixs. CONSTRUCTION → Code Generation의 기존 UI 관측·접속 범위다.
승인된 [실행 계획](../../plans/drain-integration-plan.md)을 따른다.

## 기준과 변경

- UI/demo-back PR #6은 main `666126a`, Drain PR #7은 main `50af6cf`로 병합됐다.
  현재 담당 브랜치는 두 병합과 card-news의 새 5장 구성을 fast-forward했다.
- `166c935`의 제품 변경은 `internal/api/api.go` 등록 한 줄이다.
  `GET /{$}`를 `/ui/`로 302 이동시켜 기본 Mediator 주소의 404를 해결한다.
  Go ServeMux의 HEAD 일치도 함께 검증했다. 상대 Location이므로 요청 Host를
  이동 대상으로 복사하지 않는다. UI와 공개/실 모드의 인증 경계를 유지한다.
- 새 `internal/api/entry_test.go`는 실/데모 양쪽의 GET·HEAD `/` 302,
  `/ui/` 200, 미등록 일반/API 경로 404, POST `/` 405와 인증 없는 제출 401을 확인한다.
- Drain 구현은 shin-son, 제어판은 nacl1119의 범위다. UI의 기존 읽기 모델이
  실제 동작을 그대로 표시해 추가 제어 기능이나 API 변경은 필요하지 않았다.
- 후속 변경은 [Draft PR #8](https://github.com/taeels/enode-fixup-project/pull/8)로 제출했다.
  영상 세션의 자산과 state/audit 변경은 작업 디렉터리에 보존하고 이 PR에 싣지 않는다.

## 합친 코드의 검사

Go 1.26.6, Node 기본 실행기, macOS arm64, PostgreSQL 16.14를 사용했다.
테스트 DB는 별도 포트 57567의 `runixs_drain_intake`다. 실행 중인 데모 DB
`enode_runixs_rc1`과 분리했고 `scripts/testdb.sh`로 선행 조건을 확인했다.

```sh
export PATH=/Users/runixs/Library/Caches/enode-runixs/go1.26.6/bin:$PWD/.github/ci-stubs:/usr/bin:/bin:/usr/sbin:/sbin
export GOTOOLCHAIN=local
export ENODE_TEST_DATABASE_URL='postgres://enode_test@127.0.0.1:57567/runixs_drain_intake?sslmode=disable'
eval "$(scripts/testdb.sh)"
go test ./... -count=1 -coverpkg=./... -coverprofile=local/drain-integration-20260909/cover.out -json > local/drain-integration-20260909/go-tests.jsonl
go vet ./...
go run scripts/glyphscan.go
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build ./...
```

Node가 설치된 원래 PATH에서 `node --test internal/api/ui/tests/*.test.mjs`도 실행했다.
네 CLI의 macOS arm64 실행 바이너리를 빌드해 실제 런타임에 적용했다.

| 검사 | 결과 |
|---|---|
| Go 전체 | 1,190 통과, 실패 0, 스킵 0 |
| 표준 커버리지 블록 병합 | 16패키지 모두 80% 이상, 최저 build 80.000% |
| API / UI / enode / store | 84.320% / 98.387% / 84.877% / 82.350% |
| Node | 46 통과, 실패·스킵 0 |
| vet·glyphscan·변경 Go 포맷 | 통과 |
| Mac 실행 빌드·Windows amd64·Linux arm7 | 통과 |

첫 전체 실행은 이 Mac의 Homebrew 크로스 컴파일러가 탐지되어 기존
`TestDetectEmpty`의 빈 설정 가정과 충돌했다. 실제 탐지는 정상적으로 arch를
광고했지만 테스트는 그 도구가 없다고 가정한다. CI처럼 크로스 컴파일러가 없는
PATH와 저장소의 claude stub으로 재실행해 위 최종 결과를 얻었다. 제품이나
테스트 조건을 완화하지 않았으며 최초 실패 로그도 로컬에 보존했다.
검사로 바뀐 `cmd/enodectl/probe.lock` PID는 테스트 종료 확인 후 원상 복구했다.

## 실제 Mac 노드와 UI 인수

노드 `9b2fab72b8fc`의 실제 광고와 enode 실행기를 사용했다. shell 작업은
`agent.reason`과 이 노드의 `instance=rc1-local`을 요구한다. 하드웨어 광고를
만들거나 claim/finish API로 실행 성공을 대신하지 않았다. 별도 이름의 Run만
제출했고 기존 다른 사용자의 Run은 변경하지 않았다.

| 단계 | 관측 |
|---|---|
| A 실행 + B 대기 | A 첫 단계 CLAIMED, B 제출 202/QUEUED |
| at-boundary 광고 | lease 유지, UI `draining · 진행 중`, B는 QUEUED |
| A 첫 단계 완료 | 첫 단계 DONE·다음 단계 FAILED, A FAILED, lease 해제 |
| 중단 사유·봉인 | `cancelled by: drain:9b2fab72b8fc`, record의 첫 결과물 내용 일치 |
| draining 유지 | UI `draining · 대기 중`, B 계속 QUEUED, 작업 목록 유지 |
| 정책 해제 | B RUNNING/CLAIMED로 승격된 뒤 SUCCEEDED |
| graceful | 첫 단계가 끝나도 다음 단계 실행, 두 단계 DONE과 Run SUCCEEDED |
| 최종 UI | A의 drain 사유와 DONE/FAILED 그래프, B·graceful 성공, 가용 노드 표시 |

Run ID는 `runixs-drain-a-7b0b06d5`, `runixs-drain-b-7b0b06d5`,
`runixs-drain-graceful-7b0b06d5`다. 검증용 정책을 제거했고 노드는 draining 빈 값,
lease 없음으로 돌아왔다. API 원문·브라우저 DOM·봉인 tar·검사 로그는
ignored `local/drain-integration-20260909/`에 보존한다.
이것은 runixs의 Drain 소비자 통합 증거이며 다른 담당자의 CP4 제어판이나
LED·음원·방송 CP10 전체 완료를 선언하는 기록은 아니다.

## 현재 실행 환경과 접속

로컬 Mediator·enode·runctl·enodectl을 `166c935b7795` 빌드로 갱신했다.
기존 DB·토큰·node_id를 유지했다. 이전 rc1 바이너리는 로컬 백업에 보존했다.
이미 게시된 v0.1.0-rc1 태그와 다운로드 자산은 `2d7bf7524d9c` 그대로다.
따라서 새 다운로드에 이번 Drain과 기본 경로 수정이 포함됐다는 뜻은 아니다.

2026-09-09 en0의 `172.24.82.142:8080`에서 같은 Mac이 확인한 결과:

| 경로 | 응답 |
|---|---|
| `/` | 302, Location `/ui/` |
| `/ui/`·`/ui/fleet/`·`/ui/demo/` | 200 |
| `/v1/nodes`·`/v1/runs` | 200 |
| `/missing` | 404 |

Mediator는 `0.0.0.0:8080`에 바인딩돼 있고 macOS 방화벽은 해당 바이너리의
인바운드를 허용한다. 방화벽 설정을 변경하지 않았다. 다른 팀원 기기의 URL·
오류·IP는 요청했으나 결과를 아직 받지 못했다. 같은 Mac의 LAN 응답으로
무선망의 다른 기기 간 통신까지 확인했다고 쓰지 않는다.

## 남은 공동 입력과 보안 범위

실물 CP10은 문태호의 LED/음원 래퍼와 보드, 손신의 mac Claude+higgsfield,
최태양의 공개 webcam embed URL을 연결해 확인한다. 기본 webcam 설정은 null이다.
PR #6 병합은 이 실물 검증을 대신하지 않는다. 공식 단계는 Code Generation을 유지한다.

security-baseline의 이번 적용 범위는 충족했다. 고정 상대 리다이렉트와 쓰기 인증
회귀를 확인했고 토큰은 사용자 전용 로컬 파일에만 보존했다. 새 입력 파싱·
데이터 저장·의존성·공개 송신 대상은 없어 해당 신규 통제는 N/A다.
배포·TLS는 기존 승인된 로컬 데모 범위를 유지한다. resiliency-baseline과
property-based-testing은 상속된 비활성 선택을 유지한다.
