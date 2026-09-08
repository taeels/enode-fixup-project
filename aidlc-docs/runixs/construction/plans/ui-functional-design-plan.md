# ui Functional Design 계획 — 실 함대·공개 데모

**상태**: 실 함대·데모 전체 설계 검토안 작성. 공용 연결 입력과 사용자 리뷰는 남았다.
**기준**: 구현 `0a159a4`, 설계 gitlink `29c89cd`, AI-DLC `1.0.1`.

사용자의 “다음 진행해줘”에 따라 사전 준비를 상세 설계로 발전시켰다. ui 는 W1·W3
두 모드가 있는 한 유닛이다. 네 FD 문서에 W1·CP7과
[구체화한 담당 범위](../runixs-scope.md)의 D1~D5 데모를 함께 정리했다.
User Stories 대신 승인된 유닛-게이트-기능 매핑을 쓴다.

## 진행

- [x] 최신 main 의 배정·파일 행렬, 담당 상태와 실제 obs 유무 확인.
- [x] 팩·정본·기존 UI 와 `design/README.md`, `enode-ux.pen`, S1·S2 export 대조.
- [x] 도메인·사용자 흐름·표시/검증 규칙과 화면 구성 작성.
- [x] 위임된 sandbox 표시 출처와 실 모드 진입 경로 결정안 작성.
- [x] 공유 파일 접점과 obs 에 의존하는 계약 확정 작업 분리.
- [x] 검토 가능한 파일별 코드 생성 계획 작성.
- [x] 기존 UI Go 테스트 실행: 92.3% statements, 기준 80% 충족.
- [x] UI-01~06·DEMO-01~02의 파일·담당 접점·완료 조건 구체화.
- [x] origin/main의 obs 병합 확인·rebase·D1~D5 코드/관측 테스트 대조와 설계 갱신.
- [x] 데모 D1~D5의 투어·모달·웹캠·공용 표시를 네 FD 문서에 보완.
- [x] demo-back의 별도 FD에서 제출 계약·allow-list·이름 전달 구체화.
- [ ] sandbox 출처·고정 시나리오·방송 설정의 필요한 공용 접점 확인.
- [ ] 보완한 담당 설계를 리뷰하고 그 결과 기록. 기존 Q1은 W1 초안의 미응답 기록.
- [x] 설계에 맞춰 ui 11단계/demo-back 7단계 코드 계획 보완.

## 산출물

1. [업무 흐름](../ui/functional-design/business-logic-model.md)
2. [표시·검증 규칙](../ui/functional-design/business-rules.md)
3. [자료 모델](../ui/functional-design/domain-entities.md)
4. [화면과 모든 조작](../ui/functional-design/frontend-components.md)
5. [코드 생성 계획](ui-code-generation-plan.md)
6. [실 함대/팀원 데모 pen의 타깃 대조](../ui/functional-design/design-targets.md)
7. [3d-view 구현 재사용 검토](../ui/functional-design/3d-view-reuse-review.md)
8. [담당 범위와 완료 조건](../runixs-scope.md)
9. [obs 인수 결과](../ui/preparation/obs-integration-review.md)
10. [현재 설계·구현 순서 리뷰](../design-review.md)

## 검토 Q1 — 설계와 구현 순서

**이전 W1 초안의 검토 질문**: obs 전달 전을 전제한 아래 질문은 아직 미응답이다.
obs 인수와 범위 보완 뒤의 전체 설계 리뷰를 대신하지 않는다. 현재 설계 보완을
진행하며, 이 질문에 다시 답해야만 obs 인수를 할 수 있는 것은 아니다.

**현재 검토 질문은 [통합 리뷰](../design-review.md) §4로 대체했다.** 아래는
이전 제안의 기록이며 두 질문을 중복으로 승인받지 않는다.

위 W1 설계와 코드 생성 계획의 전체 순서를 승인하는가? 제안은 obs 전달 전에는
단계 1~3의 독립 표시 모델·화면·로컬 검증까지만 만들고, 실제 API 어댑터·랜딩
연결은 obs 계약 대조 뒤 단계 4~8에서 수행하는 것이다. 사전 분리 구현은 이번에
새로 제안하는 범위다. 공용 roster 의 W1 선행 조건·실제 게이트·병합 조건은 유지한다.

[Answer]: 미응답. 현재 요청을 아직 제시하지 않은 설계/코드 계획의 승인으로 기록하지 않는다.

Q1은 runixs 담당 유닛의 설계/작업 계획 검토다. 팀 전체 요구·다른 유닛의 API 계약·
공용 착수/병합 조건을 승인하는 질문이 아니다. roster §6의 sandbox 출처는 진행자와
협의할 결정안으로 남겨 두며, 담당자의 계획 검토만으로 공용 미정을 닫지 않는다.

업무 규칙·엔티티·흐름·오류·화면 조작의 선택은 승인된 팩과 코드에서 정했다.
관측 API 의 미정은 사용자에게 wire 형식을 추측하게 묻지 않고 D1~D5의 실제
직렬화 대조 항목으로 유지한다. NFR Requirements·NFR Design·Infrastructure
Design 은 기존 실행 계획대로 건너뛴다.
