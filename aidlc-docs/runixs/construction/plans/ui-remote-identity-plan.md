# 원격 실행 장비와 작업 구분 — UI 후속 계획

runixs의 기존 ui Functional Design/Code Generation에 대한 사용자 후속 요청이다.
PR #22가 병합된 main `4290cee`에서 `unit/runixs-remote-identity`로 진행한다.

## 사용자에게 보일 결과

새 작업 카드에서 브라우저 요청이 연결된 장비의 enode로 전달된다는 점과 각
시나리오의 실행 장소/효과를 설명한다. 그래프는 Run마다 일관된 작업 색을 갖고,
실행 장비 카드는 호스트별 색·소유자·호스트명·OS/CPU·담당 역할을 표시한다.
상태는 기존 상태색과 상태 문자열로 별도 표시한다. 같은 호스트는 같은 장비색이며
단계/Run ID는 바꾸지 않는다. 색만으로 의미를 전달하지 않는다.

## 관측 경계

소유자는 명시적 owner 광고, 없으면 enode의 `handle@hostname:config` label에서
표시용으로 읽는다. 이 값은 사용자 실명이나 인증된 소유권으로 추정하지 않는다.
현재 노드가 사라진 이전 Run은 assigned의 당시 label을 사용하며 장비 상세는
미관측으로 표시한다. 실행 노드가 없는 단계는 배정 대기를 유지한다.
호스트 수는 관측된 hostname 기준이며 물리 PC 수로 주장하지 않는다.
board/device=led/speaker는 연결 장치 광고다. Windows+board 광고를 Raspberry Pi
호스트로 그리지 않는다. VM도 명시적 sandbox/device_type 광고로 표시한다.

## 실행 순서

- [x] 1. 기존 FD·PR #22 병합·광고 계약/공개 노드의 표시 필드·시나리오 계약 확인.
- [x] 2. `shared/fleet/identity.mjs`에 결정적 색·장비/역할/과거 label 표시 함수를 추가.
- [x] 3. `scene.mjs`, `view.mjs`, `fleet.css`에서 함대/Run 카드·목록·상세·간선·반응형 배치를 연결.
- [x] 4. `demo/submission.mjs`, `demo.css`에서 원격 실행 가치와 장비 경로가 드러나는 카드 설명을 추가.
- [x] 5. 같은/다른 호스트·VM·연결 보드·미관측·지연·DAG와 desktop/mobile 2D/3D를 검증.
- [ ] 6. 담당 FD/state/audit·검증 기록 갱신, 커밋·PR·Mac mini 운영 반영.

UI 범위에서 API/DB/enode 광고와 계약 ID·인증을 변경하지 않는다. 추측한 실명이나
장비 모델을 하드코딩하지 않는다. Gallery 이름 복원을 유지한다.
운영 업데이트는 기존 지시에 따라 추가 검사 없이 빌드·교체한다. 개발 검증은
fixture로만 수행하고 실제 작업을 제출하지 않는다. 기존 보안 확장·비활성 확장·
공동 CP6/CP10 보류를 계승한다.
