# Run 요청·배정·실행 흐름 — 2026-09-09

담당 runixs의 기존 UI Functional Design·Code Generation에 대한 사용자 후속
보완이다. 현재 `Runixs/UI-Update-2`의 ID 축약 변경을 보존하고 수행한다.

## 설계와 범위

- 게스트/제출자 아바타, Mediator 허브, 실제 enode 실행 단계 영역을 연결한다.
- 게스트·Mediator는 역할을 나타내는 시각 요소다. 함대 등록 노드 수나 API의
  steps/needs에 삽입하지 않는다. 실제 단계 간선은 needs만 사용한다.
- 게스트 이름은 선택 Run의 submitter다. 현재 관람객의 이름으로 대체하지 않는다.
- Mediator는 Run 상태에 따른 배정 대기·실행·응답 대기·검증·성공·실패를 보인다.
  CLAIMED 단계의 연결만 RUNNING 중 움직인다. 상세 오류·15초 지연·종료·
  reduced-motion에서는 움직임을 멈춘다. 네트워크 패킷·진행률을 추정하지 않는다.
- 단계 없는 QUEUED도 요청자와 Mediator를 보이고 enode 배정 대기를 안내한다.
  아직 없는 실행 노드를 만들어 배정하지 않는다. 잘못된 DAG는 오류 안내를 유지한다.
- 게스트·Mediator 선택 시 역할과 원본 Run 상태·시각을 상세에 표시한다.
- 데스크톱은 상단 요청·배정, 하단 실제 실행 DAG로 배치한다. 좁은 화면에서는
  위에서 아래로 배치하고 자동 맞춤은 폭 기준으로 적용해 세로로 읽을 수 있게 한다.
  기존 확대·이동·함대 복귀·목록·전체 ID·키보드 선택을 유지한다.

## 실행

- [x] 1. 기존 승인·FD·관측 모델·게이트·소유 경계를 복원하고 보완 설계를 기록한다.
- [x] 2. `internal/api/ui/static/shared/fleet/model.mjs`에 관측 기반 흐름 상태를
  추가하고 `scene.mjs`, `view.mjs`, `fleet.css`에 시각·상세·좁은 화면 배치를 연결한다.
- [x] 2a. 후속 사용자 지시로 상단의 네 버튼을 보기 설정 그룹의 화면·표현
  드롭다운 두 개로 교체한다. 선택 Run이 없으면 작업 그래프 옵션만 비활성화하고,
  목록 선택·새 제출·함대 복귀·2D/3D 전환 시 실제 상태와 선택값을 동기화한다.
- [x] 3. 상태/종료/지연 회귀를 Node로 확인하고 브라우저에서 2D/3D·두 모드·
  좁은 화면·단계 없는 Run·선택·키보드·자동 갱신·원본 ID·동작 축소를 검증한다.
  Go UI·vet·glyphscan·Mediator embed 빌드도 확인한다.
  드롭다운의 단일 선택·키보드·포커스 복귀·작업 자동 전환도 함께 검사한다.
- [x] 4. FD 보완과 구현·검증 결과, 담당 state/audit를 갱신한다.
- [x] 5. 사용자 PR 요청에 따라 변경을 커밋하고 최신 main을 인수한 뒤 검증·push·PR을 제출한다.
  `main@c51d923` 인수, 코드 `8b4d452`, [PR #19](https://github.com/taeels/enode-fixup-project/pull/19).

[구현·검증](../ui/code/ui-run-flow.md): Node 63개, 관측 모형 4환경과 실제 데모
진입점 2환경, Go UI 98.4%·vet·glyphscan·Mediator 빌드 통과.

관련 수용 기준은 CP1·CP2·CP7·CP9의 화면 의미다. 이번 합성 관측 UI 검사는
DB·실제 장비·공동 CP6/CP10의 통과를 의미하지 않는다. 새 API·DB·의존·배포는
없으며 승인된 NFR/인프라 SKIP과 기존 담당 유닛의 공식 리뷰 상태를 유지한다.

## 보안 확장

SECURITY-04·05·08·11·12·13·15: 기존 CSP·인증·API 검증과 textContent 삽입을
유지한다. SECURITY-01·03·07은 decisions §3 예외를 상속한다.
SECURITY-02·06·09·10·14는 인프라·권한·배포·의존·운영 변경이 없어 N/A다.
resiliency-baseline·property-based-testing은 Disabled로 건너뛴다.
