# demo-back Functional Design 계획

**상태**: ui와 함께 FD·코드 계획 승인 완료. obs·queue 인수를 완료했다.
진행자 시나리오 연결은 남아 있으며, 공개 라우트는 아직 구현하지 않았다.

## 진행

- [x] 배정·고정 시나리오·공개 쓰기 수락 범위와 obs 접점 확인.
- [x] 제출 본문·응답·재시도 식별·Guest 이름의 검증 규칙 설계.
- [x] allow-list에서 기존 submit으로 이어지는 흐름·실패·저장 책임 설계.
- [x] ui가 같은 계약을 소비하도록 상태와 완료 조건 연결.
- [ ] 진행자 LED/음원 픽스처와 이름 주입 위치 확인.
- [x] queue 인수: 같은 ID 재접수·202/QUEUED·submitter 저장·승격 대조.
  [인수 증거](../ui/code/queue-integration-review.md), 내부 컨텍스트부터의 DB 회귀 테스트 포함.
- [x] 사용자 설계 검토 결과 기록. 2026-09-08T20:11:36Z “응 시작해”.

## 산출물

- [업무 흐름](../demo-back/functional-design/business-logic-model.md)
- [자료 모델](../demo-back/functional-design/domain-entities.md)
- [검증 규칙](../demo-back/functional-design/business-rules.md)
- [ui/demo-back 제출 계약](../demo-back/functional-design/submission-contract.md)
- [통합 입력과 리뷰](../design-review.md)

NFR Requirements·NFR Design·Infrastructure Design은 기존 SKIP을 유지한다.
적용 보안 요구는 FD 규칙에 반영한다. 새 큐·인증·DB 스키마를 만들지 않는다.
