# demo-back 코드 생성 계획

**승인**: 2026-09-08T20:11:36Z “응 시작해”. 2026-09-09 담당 구현 우선 지시에
따라 백엔드 생성을 진행했다. queue와 진행자 실제 픽스처 c7a237d 인수 완료.

**진행 기준**: 실제 LED·음원 example을 인수해 공개 매핑을 연결했다.
원본 fixture·진행자 문서는 c7a237d 그대로이며, 실제 장비·방송 장면은 공동 검증이다.
[FD 계획](demo-back-functional-design-plan.md), [제출 계약](../demo-back/functional-design/submission-contract.md).

## 파일

| 경로 | 변경 |
|---|---|
| internal/api/demo.go | 입력 검증·고정 매핑·내부 요청·제출 한도 |
| internal/api/demo_test.go | 잘못된 요청·같은 의도 재시도·기존 submit/DB 연결 |
| internal/api/demo_queue_test.go | 선행 인수: submitterKey부터 실제 queue 저장·목록·승격까지 회귀 검증 |
| internal/api/api.go | Config.Demo 조건부 등록 줄. submit 본문은 재사용 |
| internal/contract/examples | 진행자가 제공한 LED/음원 example 소비. 임의 하드웨어 명령 생성 없음 |
| internal/store | 기존 이름 저장 재사용. queue 대조 결과 추가 변경이 필요하면 계획에 구체화 |
| aidlc-docs/runixs/construction/demo-back/code | 구현 요약·실행 증거 |

Config.Demo·token bucket·submitterKey·기존 submit을 재사용한다.
데모 공개 설정은 UI의 정적 settings.json이며 config.go에 방송 필드를 추가하지 않는다.
제출 한도는 등록할 handler가 소유하는 별도 버킷으로 두어 Server 공용 필드를 늘리지 않는다.

## 순서

- [x] **1 — obs·queue 인수**
  queue 병합·202/QUEUED·CreateQueuedRun의 submitter·같은 ID 재접수를 확인한다.
  실제 접점이 바뀌면 제출 계약/이 계획을 갱신한다. 선행 부족은 기록하고 UI 독립 작업을 계속한다.
  2026-09-09: queue PR #5/main@310c22d를 인수했다. 실제 202/QUEUED, 같은 ID
  재접수, 승격, 목록 submitter 보존을 확인했다. [인수 기록](../ui/code/queue-integration-review.md).
  한글 두 단어 이름 계약을 적용한다. 실제 LED/음원 파일 인수는 4a의 활성화
  조건으로 옮긴다. 미설정 시나리오를 503으로 막는 승인된 FD를 먼저 구현한다.
  인수 상태를 다음처럼 나눈다. queue 구현이나 진행자의 중복 확인을 기다리지 않는다.

  - [x] queue PR #5/main@310c22d 병합·실제 접수·승격·목록 이름 인수.
  - [x] demo_queue_test.go에서 내부 submitterKey를 넣은 기존 제출 경로의 이름 보존을 자동 검증.
    201/202·재접수 200·승격 후 목록 이름 유지, 외부 헤더 무시 통과.

- [x] **2 — 고정 요청 준비**
  demo.go의 엄격한 세 필드 검사·정규 ID·고정 매핑·Contract 사본 준비를 구현한다.
  이름/시나리오/UUID 경계, unknown/duplicate key, body/Content-Type, 입력을
  경로/셸 명령으로 사용하지 않음, 원본 픽스처 불변을 demo_test.go에서 검증한다.
  결과: 엄격한 본문·이름/UUID 경계·정규 ID·사본/원본 불변·미설정 거절 검사 통과.
  내부 시나리오 정의는 고정 example 이름과 이름 주입 함수다. 인수 전에는 비운
  매핑으로 공통 경로를 검증했고 c7a237d 인수 뒤 실제 두 example을 연결했다.
  미설정/잘못된 주입 위치는 503으로 막는다. generic example은 테스트 전용이다.
- [x] **3 — 기존 submit 연결**
  서버 토큰과 submitterKey를 복제 요청에 넣어 기존 auth/postRuns를 호출한다.
  Config.Demo에서만 등록하고 2 req/s·burst 4 전용 버킷·Retry-After를 적용한다.
  브라우저 헤더의 token/principal은 전달하지 않는다. 신규 DB 스키마를 만들지 않는다.
  결과: demo.go 구현, api.go 조건부 등록 3줄 추가. 모드·인증·헤더 보존·전용 한도 검사 통과.
- [x] **4 — DB 포함 접수 검증**
  전용 테스트 DB에서 201/200/202·거절·동시 재시도·응답 유실을 검증한다.
  같은 의도 Run 하나, 다른 이름/의도 분리, QUEUED 승격 뒤 이름 보존과
  기존 실 함대 쓰기·공개 읽기·asks 인증을 확인한다. 스킵을 통과로 세지 않는다.
  결과: 전용 PostgreSQL에서 테스트 전용 command 매핑의 201/202/200·422·승격·
  이름 보존·동시 같은 요청 4개의 단일 Run·응답 유실 후 재시도 검사 통과.
  실제 LED/음원 매핑과 공개 기본 설정의 성공 장면은 4a 뒤 별도 검증이다.
- [x] **4a — 실제 시나리오 인수·활성화**
  진행자가 제공한 LED/음원 계약 파일과 LED 두 변형의 버튼 연결, 음원 이름의
  안전한 주입 위치·Work/ledger를 대조한다. 고정 매핑을 채우고 실제 계약 불변을
  재검증한다. generic example을 실제 LED/음원 대체로 연결하지 않는다.
  결과: c7a237d를 충돌 없이 fast-forward, 두 example의 Validate·success_when·lint
  통과. LED heartbeat→persistent, 음원 synthesize.in.prompt 한 자리표시자만 치환.
  실제 파일의 requires·needs·성공 조건·argv 보존과 요청별 Work/기본 ledger 격리 통과.
- [x] **5 — UI와 실제 연동**
  5a: 공통 구현의 미설정 503·잘못된 요청·한도·기존 UI 재시도는 먼저 브라우저로 확인한다.
  5b: 실제 픽스처 연결 뒤 공동 계약의 세 값·응답 상태를 실제 브라우저로 대조한다. 오류/미확인/재시도,
  목록 지연·submitter와 음원 입력의 일치, 임의 계약 거부를 CP9 증거로 남긴다.
  결과: 실제 공개 Handler+인수한 fixture+로컬 테스트 광고+전용 DB를 브라우저로
  검증했다. 201 LED·202 음원·QUEUED→RUNNING 선택 유지·이름·그래프·목록
  13개 단언 통과, 실제 claim의 음원 prompt 이름도 일치. 하드웨어 명령은 실행하지 않았다.
- [ ] **6 — Build and Test·장면**
  CP0 표준 build/vet/test/커버리지·정본/표기 검사와 CP9 서버측 검증을 실행한다.
  실제 두 시나리오·웹캠·봉인 CP10은 ui/하드웨어와 공동 확인한다.

  - [x] Go 1,173·0 스킵·16패키지 80% 이상, race·vet·build·교차 빌드·govulncheck.
  - [x] CP9 로컬 소프트웨어: 실제 fixture API/DB/UI, Node 46·브라우저 14개 단언.
  - [ ] 실제 LED·음원·웹캠·봉인 공동 장면.

- [ ] **7 — 기록·리뷰·커밋**
  구현 요약·테스트 명령/환경/결과·공유 파일 diff를 담당 문서에 남긴다.
  단계 승인 시 커밋하고, 필요한 게이트가 충족된 뒤 PR/직렬 병합을 진행한다.

  - [x] 구현·검증 문서와 담당 state/audit 작성.
  - [x] PR #6은 2026-09-09T02:25:03Z main@666126a로 병합됐다.
  - [ ] 전체 유닛 결과 리뷰·실물 공동 게이트 확인. 병합 사실이 CP10 검증을 대신하지 않는다.


이 계획은 새로운 시나리오 제작·운영 담당 배정·다른 유닛 변경을 승인하는 문서가
아니다. 입력이 없는 항목은 미설정으로 처리하며 CP9/10 완료로 표시하지 않는다.
