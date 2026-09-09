# ui 자료 모델 — 관측과 화면 상태

**상태**: 검토안. 서버에 테이블·필드·새 상태를 추가하는 모델이 아니다.
wire 근거는 [API 계약표](../preparation/api-contract.md)와
[obs 인수 기록](../preparation/obs-integration-review.md)에 있다.
아래는 표시 모듈이 소비할 의미 모델이다. 실제 직렬화는 `0a159a4`와 대조했다.

## 1. 관측 자료

| 자료 | 식별자·주요 값 | 관계·부재 의미 |
|---|---|---|
| FleetSnapshot | observedAt, nodes | 성공한 스냅샷에 포함된 광고만 표시 |
| NodeObservation | nodeId, label, instance, capabilities, seenAt, expiresAt, lease?, draining | nodeId 로만 결합. instance 변경은 재기동 구별 |
| Capability | capability, attrs | attrs 는 문자열 맵. node 안에서 각각 표시 |
| LeaseObservation | runId, notAfter | 현재 관측 임대. 완료 예상 시각이 아님 |
| RunSnapshot | observedAt, runs | 서버가 보낸 created_at 내림차순 |
| RunSummary | runId, state, workId, createdAt, endedAt?, verdict?, assigned, submitter | assigned 는 과거 배정도 포함, 현재 점유는 lease 로만 판단 |
| RunDetail | runId, state, requires?, steps, assigned, verdict?, warnings | 요구 읽기 실패의 부분 응답을 보존. submitter는 목록에서 결합 |
| Requirement | as, capability, count?, attrs | as 는 상세 안에서 steps[].uses 와 연결 |
| StepObservation | seq, id, state, uses, needs, node?, attempt?, startedAt?, endedAt?, chosen | id 는 그래프, seq 는 질문 연결. chosen 의 미제공은 false 와 다름 |
| AskObservation | runId, seq, step, prompt, schema, answerers?, askedAt?, deadline?, canAnswer | 질문 하나의 키는 runId/seq. canAnswer 가 참이어도 중앙 답변 기능 없음 |
| Verdict | state, checks, fleet? | check 의 step/what/want/got/ok/note 를 표시, 원시 문자열로 변환하지 않음 |

현재 구현의 Run 상태 RUNNING·VERIFYING·SUCCEEDED·FAILED 에 예정된 QUEUED 를
더해 그대로 표시한다. 그 밖의 문자열 상태는 원문과 알 수 없음 표시를 함께 낸다.
단계 어휘는 PENDING·CLAIMED·DONE·FAILED·SKIPPED·ASKED 다.

## 2. 화면이 소유하는 상태

| 상태 | 값·수명 |
|---|---|
| AuthSession | token, generation, 진입/확인 중/인증됨/거절됨. 탭 수명 |
| ViewSelection | 격자/그래프, 선택 runId/stepId, 목록 state 필터. 메모리 |
| ResourceState | 마지막 정상 자료, observedAt?, receivedAt, 실패 횟수, 진행 중 요청, 오류 분류 |
| DetailCache | runId 별 ResourceState. 선택 Run 과 현재 임대 Run 집합으로 제한 |
| ClockAnchor | 서버 관측 시각 + 수신 당시 단조 시계. 카운트다운의 기준 |
| DashboardMode | real/demo. 요청 경계와 인증 처리를 분리하는 앱 입력 |
| SceneState | fleet/run, 2d/3d, selectedNodeId, selectedRunId, selectedStepId. 메모리 |
| SceneViewport | 장면·표현별 확대·스크롤, Fit/100% 선택. DOM 재생성과 분리 |
| GuestIdentity | 기존 helper가 반환한 이름. 기존 localStorage 키 사용 |
| TourState | 닫힘/1~4, 완료 여부, 대상·포커스. enode.demo.tour.v1에 완료만 저장 |
| DemoSubmissionState | idle/pending/uncertain/rejected/accepted, 요청 세 값·세대, 응답 ID/오류 |
| WebcamState | 설정/로드/확인 상태, 정사각형 크기, 배율, pan, 이동 모드 |
| DemoSettings | webcam=null 또는 공개 embedUrl. 정적 embed 자산이며 토큰/송출키 없음 |

ResourceState 는 nodes/runs/asks 와 상세 runId 별로 둔다. 데이터 없음(최초 조회),
정상 빈 결과, 이전 결과가 있으나 실패, 계약 불일치는 서로 다른 상태다.
현재 목록 필터는 최대 100개 관측 결과 안에서만 동작하며 전체 이력 검색처럼
안내하지 않는다. W1 에 페이지네이션·since/work 입력은 추가하지 않는다.

## 3. 이번에 좁힌 출처

**sandbox (D6)**: 명시적으로 존재하는 `capabilities[].attrs.sandbox` 를 그 능력
행에 “노드 광고 값”으로 표시한다. 근거는 현재 `contract.Capability.Attrs` 의
문자열 맵, `internal/enode/detect.go` 의 label 속성 병합,
`enode-design/protocol/agent-runtime.md` 의 sandbox 광고 속성 설명이다.
값이 없으면 “미제공”이다. 노드 안의 능력마다 값이 다르면 각각 보인다.
요구 능력의 sandbox 속성은 “요구 값”으로 별도 표시하고 노드 값으로 복사하지
않는다. none/tools/os 를 강도 순으로 정렬하거나 격리 보장으로 바꾸지 않는다.
출처 선택은 사용자가 전달한 최태양님의 승인으로 확정했다(2026-09-09).
obs의 승인된 FD도 같은 labels→capabilities[].attrs.sandbox 통로를 설명한다
(`aidlc-docs/taeels/construction/obs/functional-design/business-rules.md`).
실제 장비 설정·광고 대조는 [통합 입력](../../design-review.md)에 남겼다.
출처 승인과 실제 장비 검증을 구분하며 roster는 진행자가 정리한다.

**verdict (D3)**: 현재 `internal/store/verdict.go` 의 객체와
`internal/store/reap.go` 의 Cancel 저장을 확인했다. 취소 사유는 checks 안의
`what: cancelled`, `note: cancelled by: ...` 에 기록된다. drain 취소도 이 note 를
그대로 표시한다. obs 목록이 이 객체를 보존하는 것을 코드·테스트에서 확인했다.

## 4. obs 인수로 확정한 어댑터 입력

| ID | 표시 모델 결정 | 실제 응답에서 확인할 것 |
|---|---|---|
| D1 | 중첩 attrs를 속성 맵으로 사용. count 미지정/0은 기존 계약의 1 의미 | 평탄 형태는 관측 응답으로 거부 |
| D2 | lease:null은 관측 임대 없음 | 키 생략·빈 객체는 계약 불일치 |
| D3 | Verdict 객체/null 유지, checks[].note를 원문 표시 | 실제 drain 종료 장면은 후속 확인 |
| D4 | 빈 문자열은 이름 없음 | 목록의 키 누락/null은 계약 불일치. 상세에서 이름을 요구하지 않음 |
| D5 | chosen은 필수 boolean. requires는 정상 상세에서 필수 | 200+warnings+requires 생략은 부분 응답, 요구 정보 미확인 |

이 단계의 독립 화면 검증은 명시적인 의미 모델 입력으로 한다. 합성 JSON은
실제 캡처가 아니며, 평탄 requires·lease 생략 후보는 오류 입력으로 바꿨다.
실제 어댑터는 확인한 직렬화 하나를 소비한다. QUEUED의 생성·승격은 queue 인수 때
별도로 검증한다. 공개 모드는 asks를 조회하지 않고 상세의 ASKED 상태만 표시한다.

## 5. 데모 저장과 복구

투어 완료와 제출 중 요청은 Guest 이름·카드뉴스 완료와 별개다. 제출 요청은
`enode.demo.pending.v1` sessionStorage에 세 필드와 시작 시각만 둔다. 서버 토큰·
계약 전문·상세 캐시는 저장하지 않는다. 다른 탭은 이 요청을 자동 실행하지 않는다.
확정 응답이면 지우고 미확인일 때는 남긴다. 저장 불가 상태에서는 메모리로 유지한다.

웹캠 크기·배율·pan과 그래프 위치는 해당 페이지 수명에만 둔다. 화면 크기가
바뀌면 유효 범위로 제한한다. UI 새로고침에서 실제 방송이나 Run 상태를 복원했다고
표시하지 않고 서버/플레이어를 다시 읽는다. 임의 브라우저 값을 서버 관측으로 승격하지 않는다.


## 사용자 개선 — 2026-09-09

[UI 개선 변경 계획](../../plans/ui-feedback-plan.md)을 적용한다. 웹캠 가시성은
페이지 수명에만 유지하고 닫기/재열기를 제공한다. 노드·단계 상세는 바깥 조작과
장면 전환에서 닫고, 모달/투어도 배경 클릭으로 닫는다. Guest는 한글 두 단어로
이전하되 저장된 미확인 제출의 원래 이름과 ID는 보존한다.
