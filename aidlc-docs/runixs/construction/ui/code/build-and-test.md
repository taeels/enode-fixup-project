# UI 코드 생성 중 검증 기록

2026-09-09 · `unit/runixs-ui` · 구현 기준 `0a159a4`.
공식 단계는 Code Generation이다. 승인된 코드 계획 10의 사전 검증이며,
외부 입력이 필요한 공동 Build and Test 장면을 통과했다고 기록하지 않는다.

## 단위·전체 검사

| 검사 | 결과 |
|---|---|
| Node 기본 실행기 `node --test internal/api/ui/tests/*.test.mjs` | 42 통과, 0 실패·0 스킵. 모델·GET 경쟁·Guest·투어·제출·방송·토큰 검증 |
| `go test ./internal/api/ui -count=1 -cover` | 통과, UI 98.4% |
| CI 형식 `go test ./... -count=1 -coverpkg=./... -coverprofile=... -json` | 1,090 테스트 통과, 16 패키지 통과, 0 실패·0 스킵 |
| 블록 키별 최대 count 병합, 패키지별 하한 | 전 패키지 ≥80%. UI 98.4%, api 82.9%, store 82.5%, 최저 build 80.0% |
| `go vet ./...`, `go build ./...` | 통과 |
| glyphscan | 87 Go 파일 검사, 지적 없음 (ignored 로컬 미리보기 1개 포함) |
| 추적/담당 신규 Go 파일 gofmt, `git diff --check` | 통과 |
| Windows amd64 enodectl 빌드·net/http 심볼 | 통과, T 심볼 6개 (상한 50) |
| govulncheck v1.7.0 | `No vulnerabilities found.` |
| golangci-lint v2.13.2 | 로컬 미설치, 미실행. 공용 CI에서 경고 전용인 항목 |

환경: Go 1.26.6, Node 20.18.1, 전용 PostgreSQL 16.14.
`ENODE_TEST_DATABASE_URL`에 검증용 DB `runixs_ui_tests`를 지정하고
`scripts/testdb.sh`로 설정을 인수했다. UI 미리보기 DB `runixs_ui_preview`와 분리했다.
표준 측정의 `-coverpkg=./...`와 스킵 감시는 유지했다. PostgreSQL 17 CI 실행을
대신했다고 주장하지 않는다. 커버리지 원본·JSON 로그는 ignored `local/ui-preview/`에 있다.

첫 전체 검사에서 macOS의 설치된 aarch64 크로스 컴파일러 때문에 기존
`internal/enode/TestDetectEmpty`가 실패했다. CI용 claude 스텁과 시스템 도구만 둔
PATH로 다시 실행해 전부 통과했다. 제품 탐지 코드는 변경하지 않았다.
저장소 아래 설치한 Go 툴체인을 glyphscan과 coverage 패턴이 함께 읽는 문제는
도구 디렉터리를 사용자 캐시로 옮기고 기존 로컬 경로를 symlink로 유지해 해결했다.
최종 전체 측정은 캐시의 실제 Go 경로로 실행했다.

기존 테스트가 추적 파일 `cmd/enodectl/probe.lock`의 PID를 변경했다. 최초 작업
상태에는 없던 이번 테스트의 변경만 원복했다. 사용자의 기존 untracked 파일은
보존했다. 따라서 작업 디렉터리 전체가 깨끗하다는 CP0 조건은 선언하지 않는다.

## 실제 API를 연결한 로컬 브라우저

Orca 전용 검증 탭에서 실제 `api.New(...).Handler()` 두 모드를 사용했다.
실 모드 18092, 공개 모드 18093은 loopback으로만 열었다. UI Handler는 실제 embed다.

- 잘못된 토큰 401 → 입력/세션 정리. 정상 토큰 → nodes/runs 확인 후 실 함대.
- 새 탭의 `/ui/fleet/` 직접 방문 → S0, 토큰·보호 자료 없음.
- 인증 종료 → S0, token과 관측 자료 삭제.
- 전용 DB에 실제 POST한 광고 3개, example-command 기반 Run 201 접수.
  GET 폴링으로 노드 3개·Run 1개, 실제 needs 단계·요구 능력을 표시했다.
- 데모는 같은 공개 GET 셋을 무인증으로 조회한다. asks·Bearer를 보내지 않는다.
- 실제 미등록 `POST /v1/demo/runs`의 404를 “아직 제공되지 않음”으로 표시했다.
- Guest 첫 진입 → 카드 4장 → 데모, 재방문 → 바로 데모. 기존 이름/온보딩 유지.

이 Run의 명령을 하드웨어에서 실행한 것은 아니다. 노드 프로세스의 실제 광고,
QUEUED 승격, ASKED 응답, LED/음원, 방송·봉인 증거와 구분한다.

## 합성 입력·브라우저 조작

합성 입력은 static 밖 testdata와 ignored 로컬 미리보기에서만 사용했다.
배포 코드에 fixture fallback·샘플 경로·mock query flag는 없다.

- 광고 6개 상태, nested requires, SKIPPED+chosen, ASKED와 drain·만료를 검증.
- 키보드 노드/단계 선택, 두 보기 축, Run 목록 유지, 선택 복원.
  서버 문자열의 HTML 비실행과 polling 중 노드 포커스 보존을 확인했다.
- 30단계 긴 그래프의 전체 Fit·100%·확대·표현별 스크롤 복원.
  순환/미존재 needs는 간선을 만들지 않고 단계 텍스트를 유지.
- 투어 4단계·완료·다시 보기·취소·포커스 복원, 모달 포커스 trap·닫기·이전 보기 유지.
- 제출 중 두 버튼 차단, 테스트 전송의 503/미확인 → 같은 UUID 재시도,
  명시적 새 의도 → 새 UUID, 테스트 202 → Run+3D 선택·목록 지연 안내.
  서버 목록에 가짜 첫 행·created_at·submitter를 만들지 않음을 확인.
  실제 새로고침에서도 미확인 의도를 복원하며 POST 자동 재전송은 하지 않는다.
  HTTP LAN 환경의 randomUUID 미지원에는 getRandomValues 기반 UUID v4를 사용한다.
- 방송 크기 키보드 조절·정사각형·배율·미니맵·이동 모드·wheel·방향키·Escape·Fit.
  설정이 없으면 iframe을 만들지 않음. 실제 공급자 재생/음성은 미검증.
  추가로 합성 pointer 이벤트의 resize·두 손가락 pinch·double-click을 확인했다.
- 390×844 iframe viewport: 목록이 장면 아래로 이동, 가로 overflow 없음,
  방송 창이 실제 패널 너비에 맞춰 축소, 투어 말풍선이 화면 안에 남음.

## 공동 장면 게이트

CP1의 실 API/진입과 CP2의 기본 그래프는 위 로컬 증거가 있다. CP2의 실제 QUEUED,
CP7의 실제 ASKED·drain 결합, CP9 실제 제출과 큐, CP10 실제 하드웨어·봉인,
CP11 실제 방송은 필요한 외부 입력이 없어 보류한다. CP0는 로컬 자동 검사를
통과한 항목과 환경/작업 트리 조건을 위처럼 구분한다. 공용 게이트·main 병합 승인으로
승격하지 않으며 공용 상태와 다른 담당 문서를 수정하지 않는다.
