# demo-back 코드 생성 계획

**승인**: 2026-09-08T20:11:36Z “응 시작해”. 계획 승인. queue 인수 완료, 실제 시나리오 픽스처 인수 대기.

**작성 당시 상태**: FD와 함께 검토할 초안. queue·진행자 픽스처 의존을 유지한다.
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

- [ ] **1 — 선행 인수**
  queue 병합·202/QUEUED·CreateQueuedRun의 submitter·같은 ID 재접수를 확인한다.
  진행자 픽스처와 LED 두 변형의 버튼 연결, 음원 이름 주입·Work/ledger를 대조한다.
  실제 접점이 바뀌면 제출 계약/이 계획을 갱신한다. 선행 부족은 기록하고 UI 독립 작업을 계속한다.
  2026-09-09: queue PR #5/main@310c22d를 인수했다. 실제 202/QUEUED, 같은 ID
  재접수, 승격, 목록 submitter 보존을 확인했다. [인수 기록](../ui/code/queue-integration-review.md).
  실제 LED/음원 픽스처와 버튼 매핑·이름 주입은 아직 없어 이 단계 전체는 미완료다.
  UI 개선의 한글 이름 계약도 인수 시 적용한다. backend 생성은 픽스처를 기다린다.
  인수 상태를 다음처럼 나눈다. queue 구현이나 진행자의 중복 확인을 기다리지 않는다.

  - [x] queue PR #5/main@310c22d 병합·실제 접수·승격·목록 이름 인수.
  - [x] demo_queue_test.go에서 내부 submitterKey를 넣은 기존 제출 경로의 이름 보존을 자동 검증.
    201/202·재접수 200·승격 후 목록 이름 유지, 외부 헤더 무시 통과.
  - [ ] 실제 LED/음원 계약·버튼 매핑·이름 주입·Work/ledger 인수.

- [ ] **2 — 고정 요청 준비**
  demo.go의 엄격한 세 필드 검사·정규 ID·고정 매핑·Contract 사본 준비를 구현한다.
  이름/시나리오/UUID 경계, unknown/duplicate key, body/Content-Type, 입력을
  경로/셸 명령으로 사용하지 않음, 원본 픽스처 불변을 demo_test.go에서 검증한다.
- [ ] **3 — 기존 submit 연결**
  서버 토큰과 submitterKey를 복제 요청에 넣어 기존 auth/postRuns를 호출한다.
  Config.Demo에서만 등록하고 2 req/s·burst 4 전용 버킷·Retry-After를 적용한다.
  브라우저 헤더의 token/principal은 전달하지 않는다. 신규 DB 스키마를 만들지 않는다.
- [ ] **4 — DB 포함 접수 검증**
  전용 테스트 DB에서 201/200/202·거절·동시 재시도·응답 유실을 검증한다.
  같은 의도 Run 하나, 다른 이름/의도 분리, QUEUED 승격 뒤 이름 보존과
  기존 실 함대 쓰기·공개 읽기·asks 인증을 확인한다. 스킵을 통과로 세지 않는다.
- [ ] **5 — UI와 실제 연동**
  공동 계약의 세 값·응답 상태를 실제 브라우저로 대조한다. 오류/미확인/재시도,
  목록 지연·submitter와 음원 입력의 일치, 임의 계약 거부를 CP9 증거로 남긴다.
- [ ] **6 — Build and Test·장면**
  CP0 표준 build/vet/test/커버리지·정본/표기 검사와 CP9 서버측 검증을 실행한다.
  실제 두 시나리오·웹캠·봉인 CP10은 ui/하드웨어와 공동 확인한다.
- [ ] **7 — 기록·리뷰·커밋**
  구현 요약·테스트 명령/환경/결과·공유 파일 diff를 담당 문서에 남긴다.
  단계 승인 시 커밋하고, 필요한 게이트가 충족된 뒤 PR/직렬 병합을 진행한다.

이 계획은 새로운 시나리오 제작·운영 담당 배정·다른 유닛 변경을 승인하는 문서가
아니다. 입력이 없는 항목은 미설정으로 처리하며 CP9/10 완료로 표시하지 않는다.
