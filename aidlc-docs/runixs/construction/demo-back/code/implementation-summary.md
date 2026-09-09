# demo-back 구현 결과 — 2026-09-09

## 결과

`Config.Demo`에서만 공개 `POST /v1/demo/runs`를 등록했다. UI의 두 버튼이 진행자
c7a237d의 실제 LED·음원 example을 제출하고, 같은 요청의 재시도는 같은 Run으로
모인다. 브라우저 토큰 없이 기존 auth/postRuns·queue·submitter 저장을 재사용한다.

| 구현 파일 | 책임 |
|---|---|
| internal/api/demo.go | 엄격한 세 필드 입력, 정규 Run ID, 고정 example·이름 주입, 전용 한도, 내부 요청 |
| internal/api/demo_test.go | 본문·식별·격리·한도·인증·동시 재시도·실제 fixture 접수·이름/queue 회귀 |
| internal/api/api.go | Config.Demo 조건부 라우트 등록 3줄 |
| internal/api/ui/static/demo/demo.js | 목록 관측 이후 오래된 QUEUED 접수 안내 숨김 |

진행자 c7a237d를 PR #6의 브랜치에서 fast-forward했다. 진행 중이던 파일 6개의
해시가 그대로인 것을 확인했다. 진행자의 두 JSON과 taeels 문서는 수정하지 않았다.
별도 DB 스키마·Server 필드·공개 토큰·설정 API는 없다. 영상은 현재 워크트리에
보존하고 사용자 지시대로 담당 구현 이후로 미룬다.

## 고정 시나리오

- `led-toggle` → `demo-led-toggle`: 한 Run의 heartbeat 다음 persistent. 이름을
  실행 argv에 넣지 않는다.
- `welcome-audio` → `demo-welcome-audio`: synthesize의 agent prompt 한 필드에서
  `{{submitter}}` 한 곳만 바꾼다. play의 argv·needs·in.from은 그대로다.
- 각 요청은 example JSON을 새로 읽어 독립 Contract를 만든다. RunID와 manual
  Work 식별만 요청별로 정하며 원본 requires·steps·성공 조건·기본 run ledger 범위를
  보존한다. 주입 위치가 없거나 형식이 바뀌면 503으로 막는다.

실제 fixture의 prompt·목록 submitter·claim 입력 이름이 일치했다. 이름은 표시
데이터이고 인증 신원이 아니다. 과거 영문 Guest의 미확인 요청도 원문으로 재시도한다.

## 입력·접수 경계

본문 4 KiB, application/json, JSON 객체 하나와 정확한 문자열 키 셋만 받는다.
중복·알 수 없는 키, 다른 타입, 뒤따르는 JSON, 잘못된 UTF-8, 임의 계약/경로는
거절한다. 이름은 한글 두 단어 또는 이전 Guest 형식, request_id는 소문자 UUID v4다.
정규 JSON 배열 `[scenario_id, request_id, submitter]`의 SHA-256을 Run ID로 쓴다.

handler가 전용 2 req/s·burst 4 버킷을 갖는다. 초과 시 429/Retry-After: 1이며
공개 읽기 버킷과 독립적이다. 서버 토큰이 없으면 503이다. 내부 요청을 복제해
Authorization·Content-Type만 새로 설정하고 브라우저 principal·헤더를 전달하지
않는다. 요청 취소 수명은 유지한다. 접수 응답과 트랜잭션은 기존 코드가 담당한다.

## UI 연결에서 확인한 수정

접수 당시 QUEUED 안내가 승격 뒤에도 남던 문제를 실제 통합 화면에서 확인했다.
Run이 관측 목록에 나타나면 접수 안내를 숨긴다. 목록 반영 전에는 현재 실행 상태를
추측하지 않고 접수 확인과 목록 반영 중임을 안내한다. 3D 선택·그래프·Run 목록은
실제 GET 응답에 따라 갱신한다.

## 남은 공동 검증

소프트웨어 제출·관측 연결은 구현·로컬 검증했다. 현재 공식 단계는 CONSTRUCTION
Code Generation이며, 전체 유닛 리뷰와 실제 장비·방송 장면을 완료했다고 선언하지 않는다.
PR #6은 Draft로 유지한다.

| 공동 입력/검증 | 담당·위치 |
|---|---|
| LED heartbeat/on, 봉인 wav 재생 래퍼 | 문태호(nacl1119): Windows→rpi, enode-demo-led / enode-demo-play |
| Claude+higgsfield 음원 합성·광고 | 손신(shin-son): mac 노드의 harness:claude, tts:higgsfield |
| 공개 방송 embed URL·허용 origin | 최태양(taeels): settings.json의 webcam. 현재 null |
| UI·두 시나리오·음성·봉인 공동 장면 | 위 입력 준비 뒤 runixs가 담당 UI/demo-back과 함께 검증 |

실제 광고·sandbox 값도 장비 설정과 함께 대조한다. 로컬 테스트 광고의 결과를
장비 격리 보장이나 실제 LED/음원 실행 증거로 사용하지 않는다.

[검증 기록](build-and-test.md), [진행자 인수 안내](../../../../taeels/construction/demo-fixtures/README.md).
