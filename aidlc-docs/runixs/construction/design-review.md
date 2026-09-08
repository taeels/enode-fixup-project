# runixs 상세 설계·구현 순서 검토

**상태**: 2026-09-09 담당 설계 검토안. 공식 단계는 CONSTRUCTION의 Functional
Design 검토다. 구현 계획도 함께 구체화했으며 제품 코드 생성은 아직 시작하지 않았다.

## 1. 검토할 결과

| 범위 | 설계 선택 | 다음 코드 |
|---|---|---|
| 실 함대·데모 공통 | 관측 모델·SVG·클라이언트 공유, 모드별 인증/상태 분리 | shared/fleet 모듈, 실/데모 앱 |
| 팀원 pen | D1~D5, 우측 Run 목록·상세 덮개·4단계 투어·두 요청 버튼 | demo placeholder 확장 |
| 제출 | POST /v1/demo/runs, scenario_id·submitter·request_id | 서버 allow-list·기존 submitter 연결 |
| 재시도 | 같은 의도는 같은 Run, 미확인 자동 POST 없음 | 요청 상태·sessionStorage·기존 run_id 중복 처리 |
| 웹캠 | 좌하단 정사각형 resize·zoom/pan/fit·미니맵, 브라우저 직접 임베드 | 정적 공개 설정·CSP의 정확한 frame origin |
| Guest | 기존 이름·카드뉴스 흐름 보존, 훼손값만 복구 | 공유 helper의 좁은 변경 |

세부 문서: [UI FD 계획](plans/ui-functional-design-plan.md),
[demo-back FD 계획](plans/demo-back-functional-design-plan.md),
[제출 계약](demo-back/functional-design/submission-contract.md),
[UI 11단계 구현 계획](plans/ui-code-generation-plan.md),
[demo-back 7단계 구현 계획](plans/demo-back-code-generation-plan.md).

## 2. 공용 입력과 담당 결정의 경계

| 입력 | 현재 근거/상태 | 영향 |
|---|---|---|
| obs 응답·공개 읽기 | 0a159a4 인수 완료 | UI 소비 구현 가능 |
| queue | 현재 CreateQueuedRun/WakeQueued 없음 | demo-back의 실제 접수·CP2/9는 인수 뒤 |
| sandbox 출처 | obs FD도 labels→capabilities[].attrs.sandbox 설명. 실제 설정값/진행자 확인 미수령 | 표시 모듈은 광고값/미제공을 처리, 공용 확정/실제 시연 증거는 남음 |
| 고정 시나리오 | 현재 generic example 셋뿐, 실제 LED/음원 파일 미확인 | 두 공개 별칭/검증 경계 설계, 실제 계약 활성화는 파일 인수 뒤 |
| 방송 | 호스트형 임베드라는 결정만 있음. 공급자·공개 embed 주소 미확인 | 영역/조작/설정 없음 상태는 구현 가능, 실제 플레이어·CSP·음성 검증은 인수 뒤 |

진행자와의 확인을 runixs의 승인으로 대신하지 않는다. 위 대기 조건이 남아 있어도
공용 모델·W1·데모 화면/상태 구현을 계속할 수 있다. 임의 hardware 계약·샘플
방송·추측한 sandbox를 실제 연결로 만들지 않는다.

이번 대화에서 기존에 받은 연결 정보를 비동기로 요청했다. 회신이 없으면 정보를
받았다고 간주하지 않고 위 상태를 유지한다. 이 문서를 다른 사람에게 전송하지 않았다.

## 3. 설계 선택의 이유와 검증

- 독립 표시와 API 소비를 공유해 W1과 데모가 서로 다른 state/chosen 의미를 갖지 않게 한다.
- request_id와 기존 Run 중복 처리를 연결해 응답 유실이 같은 하드웨어 작업의 이중
  접수로 이어지지 않게 한다. 새 DB 테이블은 없다.
- 방송 URL은 같은 정적 설정에서 iframe과 CSP에 사용한다. 별도 설정 API·미디어
  중계·카메라 수집·외부 script를 만들지 않는다.
- 공급자 플레이어의 조작과 화면 pan을 구분하는 토글을 추가한다. 시안의 gesture와
  실제 iframe 재생/음소거를 함께 쓰기 위한 구현 선택이다.
- 공개 제출 한도는 읽기 버킷과 분리한 2 req/s·burst 4다. 공개 호출 반복을 제한하고
  관측 polling을 막지 않는다. 실제 동시 시연에서 검증한다.

이번 검증은 요구/pen/코드 대조·문서 참조/표기·정규 요청 식별과 이름 경계의
설계 검산이다. 실제 앱·DB 제출·브라우저·하드웨어 장면 통과는 아니다.
security-baseline의 관련 대응은 UI/business-rules와 demo-back/business-rules에 있다.
disabled 확장과 SKIP 단계는 그대로다. 새로운 배포·인증 체계의 승인을 요청하지 않는다.

## 4. 검토 결과

Q1. 위 담당 설계와 두 코드 생성 계획의 전체 순서로 진행하는가?
공용 입력은 §2대로 인수 시까지 유지하고, 준비된 UI 구현부터 시작하는 순서다.

[Answer]: 승인 — 2026-09-08T20:11:36Z 사용자 “응 시작해”.
담당 설계와 두 구현 계획의 순서를 승인했다. 공용 입력/외부 선행 조건은 유지한다.

수정 요청은 이 문서와 해당 설계/계획에 반영한다. 승인 시 담당 설계의 검토 결과를
기록하고 코드 생성 계획의 첫 실행 가능한 단계부터 진행한다.
