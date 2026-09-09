# demo-back Build and Test — 2026-09-09

## 환경·기준

진행자 픽스처 c7a237d + 이번 demo-back 구현. Go 1.26.6, PostgreSQL 16.14,
Node 20.18.1. CI claude 스텁을 PATH에 두고 scripts/testdb.sh로 전용 DB를 연결했다.
표준 DB는 runixs_ui_tests, race는 runixs_demo_race_20260909, 브라우저는
runixs_demo_browser_20260909다. 기존 실제 사용자 데이터와 공유하지 않았다.

## 실행 결과

| 검사 | 결과 |
|---|---|
| go test ./... -count=1 -coverpkg=./... -coverprofile=... -json | 1,173 통과, 실패·스킵 0, 16패키지 모두 통과 |
| 패키지 커버리지 | 모두 80% 이상. api 84.539%, store 82.336%, UI 98.387%, 최저 build 80.0% |
| 신규 demo.go | 96.0% statements |
| go test -race ./internal/api -run '^TestDemo_ConcurrentRetryCreatesOneRun$' -count=1 -v | 통과, 경쟁 보고 없음 |
| go vet ./... / go build ./... | 통과 |
| Windows amd64 / Linux arm GOARM=7 전체 빌드 | 통과 |
| govulncheck ./... | No vulnerabilities found |
| glyphscan | 90파일, 통과 |
| Node UI 전체 테스트 | 46 통과, 실패·스킵 0 |
| 최종 UI 안내 수정 뒤 Go UI | 통과, 98.4% |
| actual fixture 브라우저 | 접수·이름·큐·그래프·목록·안내 갱신 14개 단언 통과 |

전체 Go 측정은 JSON의 skip을 별도로 세었고, coverage 블록별 최대 실행 count로
패키지를 집계했다. UI 안내 수정은 JS 변경으로 이후 Node·Go UI·브라우저를
재검증했다. PostgreSQL 17의 CI 결과나 전체 CP0 완료로 확대하지 않는다.

## 실제 fixture의 API·DB·브라우저

overlay로 추가한 로컬 시험 시작 코드가 **실제 s.Handler()**를 127.0.0.1:18094에
띄웠다. 제출 응답·DB·UI 전송은 mock으로 바꾸지 않았다. 승인된 두 example을
사용하고 필요한 광고만 로컬 시험용으로 API에 넣었다. 장비 명령은 실행하지 않았다.

1. 모달의 LED 버튼이 세 필드만 전송, 토큰/principal 없음. 201/RUNNING,
   heartbeat·persistent 두 단계와 이름을 표시했다.
2. 같은 board를 요구하는 음원은 202/QUEUED. 기존 Run 목록이 남고 새 Run을 선택했다.
3. 시험용 LED Run 취소로 board를 해제하자 음원이 RUNNING으로 승격됐다.
   같은 선택에서 synthesize·play·needs가 갱신됐다.
4. 실제 voice claim 응답의 prompt가 목록의 한글 이름과 일치했다.
5. 현재 목록에 나온 뒤 예전 QUEUED 접수 안내가 숨겨지는 것을 재검증했다.

DB 회귀는 첫 201/대기 202/같은 요청 200·422 후 기존 FAILED·응답 유실 재시도·
핸들러 재생성·동시 4개 재접수에서 단일 Run·취소 컨텍스트 후 같은 요청 복구를
검증한다. 일반 submit·asks 인증, 임의 이름 헤더 무시, 원본 요청 헤더/URL 보존도
확인했다. fixture 주입 검사는 이름/Work 외 값이 원본과 같음을 대조했다.

## 검사 과정의 보정

처음 broad 검사에서 기존 로컬 queue 인수 도구가 Go 패키지 탐색에 포함됐고,
새 브라우저 시험 파일도 vet 대상에 들어갔다. 각 ignored 도구 폴더에 go.mod 경계를
추가해 16 제품 패키지만 탐색함을 확인하고 표준 검사를 다시 실행했다. 예비 결과는
최종 기준선에서 제외했다. 원격 fixture 인수 전 검사도 별도 보존하고 인수 뒤 재실행했다.
테스트가 바꾼 cmd/enodectl/probe.lock의 PID는 테스트 전 값으로 복구한다.

브라우저 초기 캡처에서 승격 후 낡은 QUEUED 안내를 발견해 수정하고 같은 실제
제출·승격 흐름을 다시 검증했다. 원본 JSON·coverage·브라우저 스크립트·캡처와
로그는 ignored local/demo-back-20260909에 보존한다. 서버는 검증 뒤 종료했다.

## 게이트와 보안 범위

CP9의 공개 allow-list·이름·제출/queue·UI 연결에 대한 로컬 증거를 추가했다.
CP10의 실제 LED·higgsfield wav·스피커·봉인과 CP11 방송 플레이어/음성은 팀 장비와
공개 URL을 기다린다. 미실행을 통과로 세지 않으며 PR은 Draft다.

SECURITY-04는 기존 UI 헤더, 05/13은 입력/고정 매핑/SQL, 08/12는 데모 모드·
서버 토큰·기존 인증, 11은 본문·전용 한도, 15는 실패/취소·재시도 회귀로 확인했다.
01/03/07은 기존 기록·예외 범위와 토큰/본문 미로깅을 유지한다. 02/06/09/10/14의
새 인프라·역할·의존 추가는 없다. resiliency/PBT disabled와 NFR/Infra SKIP을 유지한다.
