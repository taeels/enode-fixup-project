# queue 병합 인수 — 2026-09-09

## 기준과 범위

queue PR [#5](https://github.com/taeels/enode-fixup-project/pull/5)는
2026-09-09T01:00:41Z에 `main@310c22daab72ef00b6de38ef3a1ba34be90752ff`로
병합됐다. fetch 뒤 `unit/runixs-ui`의 두 커밋을 충돌 없이 rebase했다.
`9c72500`은 `f3a82b9`, `7e4b4c7`은 `4c1265d`가 됐다. 작성자와 커미터는
Taewan Kim <runixs92@gmail.com>이다. 기존 audit와 미추적 영상 파일은 보존했다.
이 작업에서 push·PR·공용 문서 변경은 하지 않았다.

공식 단계는 CONSTRUCTION → Code Generation이다. 기존 UI 계획 9·10과
demo-back 계획 1의 queue 인수 검증이며 실제 하드웨어 장면 완료가 아니다.

## 실제 API와 소비 UI

로컬 PostgreSQL 16.14 + 실제 `api.New(...).Handler()`를 사용했다.
실 모드는 18092, 공개 모드는 18093, DB는 `runixs_ui_preview`다.
검증용 광고/계약을 API로 제출했고 하드웨어 명령은 실행하지 않았다.

| 검증 | 결과 |
|---|---|
| 같은 노드에 첫 Run | 201 · RUNNING |
| 둘째 Run | 202 · QUEUED. assigned·steps 없음 |
| QUEUED의 동일 계약 재제출 | 200 · 같은 Run, 새 행 없음 |
| 함대에 없는 능력 | 422. 후속 GET에서 FAILED |
| 첫 Run claim·성공 결과 보고 | SUCCEEDED 뒤 같은 요청에서 둘째 RUNNING |
| 승격 뒤 재제출·claim | 200 · 기존 RUNNING, 둘째 작업의 단계 수령 |
| UI 관측 파서·클라이언트 | 실제 queued/promoted/claimed 응답 모두 수용. 공개 모드는 Bearer/asks 요청 없음, 실 모드는 기존 인증 유지 |
| 실제 브라우저 | QUEUED를 노드 미배정·단계 없음으로 표시. 같은 선택을 유지하고 3D의 prepare/finish 및 needs 간선으로 자동 갱신. Run 목록 유지 |
| 공개 일반 쓰기·asks | 무인증 401 유지 |
| 공개 데모 제출 | 아직 라우트가 없어 404. 픽스처 연결 완료로 기록하지 않음 |

별도 DB `runixs_queue_intake_20260909`에서 `CreateQueuedRun`에 표시 이름을
전달하고 기존 API로 재제출·첫 작업 취소·승격을 확인했다. `GET /v1/runs`에서
이름과 단일 Run 행을 유지했다. `GetRun`은 원래 submitter를 읽지 않는 접점이므로
이름 검증은 목록에서 했다. 내부 컨텍스트 전달은 `enqueue`의 Submitter 전달을
코드로 대조했다. 이 검증은 아직 없는 공개 demo-back 라우트의 E2E 검증이 아니다.

## 자동 검사

Go 1.26.6, CI claude 스텁과 시스템 도구 PATH,
`scripts/testdb.sh`가 인수한 전용 DB `runixs_ui_tests`를 사용했다.

- `go test ./... -count=1 -coverpkg=./... -coverprofile=... -json`: 1,108 통과,
  0 실패·0 스킵, 16 패키지 통과. 블록 키별 최대 count로 집계한 모든 패키지가
  80% 이상이다. api 80.632%, store 82.108%, UI 98.387%, 최저 build 80.0%.
- rebase 직후 Node 42 통과. 이후 UI 개선은 별도 [검증 기록](ui-feedback.md) 참조.
- vet·전체 빌드·Windows amd64·Linux arm/GOARM=7 빌드 통과.
- glyphscan 88 Go 파일 통과. 이후 로컬 검증 도구 추가 뒤에도 89파일로 재검사해 통과했다.

첫 측정에서 미추적 영상 폴더의 node_modules 안 Go 파일이 패키지 탐색에
포함됐다. `output/enode-video/go.mod`로 로컬 산출물을 분리한 뒤 위 표준 명령을
다시 실행했다. 최초 측정은 패키지 기준선으로 사용하지 않는다. 테스트가 바꾼
`cmd/enodectl/probe.lock`의 PID만 원복했다. PostgreSQL 17 CI나 깨끗한 워킹트리를
포함한 CP0 전체 완료를 선언하지 않는다.

원본 응답·전체 테스트 JSON/coverage·브라우저 캡처·검증 도구는 ignored
`local/queue-integration-20260909/`에 보존했다.

## 다음 입력

queue 선행은 인수했다. main의 contract examples는 agent/command/multi뿐이다.
진행자에게서 LED 두 변형의 고정 매핑, mac wav → rpi 재생 계약, Guest 이름의
안전한 주입 위치·Work/ledger 정책을 받아야 demo-back 계획 1을 닫을 수 있다.
웹캠 공개 URL/운영 입력과 sandbox 출처의 공용 확인도 별도다.
CP2의 로컬 API·UI 증거만 추가하며 실제 보드·갈래 계약·공동 장면과 CP9/10/11은
보류다. 영상 작업은 사용자 지시로 보류했고 기존 산출물과 재개 지점을 보존했다.
