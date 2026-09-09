# UI 구현 결과 — 실 함대·공개 데모

**함대 탐색·가독성 보완 — 2026-09-09**: 휠 확대·축소, 단계 카드형 그래프·함대 복귀,
임대 작업의 제출자, 기능별 정보 카드·한국어 날짜를 구현했다.
[변경·검증 결과](ui-readability.md). 기존 공동 장면 보류는 유지한다.

**후속 진행 — 2026-09-09**: 진행자 c7a237d의 실제 LED/음원 fixture를 인수하고
공개 demo-back 구현·API/DB/UI 연동을 완료했다. 아래 이전 픽스처/라우트 대기는
[최신 demo-back 구현·검증](../../demo-back/code/implementation-summary.md)으로 대체한다.
실제 장비·방송 공동 장면은 남아 있다.

2026-09-09, `unit/runixs-ui`, 기준 `origin/main@0a159a4`.
사용자의 “응 시작해”(2026-09-08T20:11:36Z)가 승인한
[11단계 코드 계획](../../plans/ui-code-generation-plan.md)의 1~8을 구현했다.
실제 데모 연결을 포함한 전체 유닛 완료와 공동 게이트 통과는 아직 아니다.

## 동작과 코드

- `static/index.html`, `landing.css`, `fleet/login.mjs`: D1 Guest 카드와 S0 토큰
  입력을 분리했다. Guest 진입은 기존 landing.js의 카드뉴스/재방문 분기를 유지한다.
  실 함대는 nodes/runs 인증 확인 후 입장하며 탭 세션만 사용한다.
- `static/shared/fleet/model.mjs`, `client.mjs`: obs의 중첩 requires.attrs,
  명시적 lease/null·submitter·chosen을 소비한다. 5초 조회·4초 본문 포함 제한시간,
  자원별 오류·선택 상세 취소·401 자료 삭제·공개 읽기 Retry-After를 처리한다.
  관측 시각은 단조 시계에 고정한다. 15초 이상 갱신되지 않은 상세/인박스의
  관련 시간은 다른 조회가 성공해도 다시 움직이지 않는다.
- `static/shared/fleet/scene.mjs`, `view.mjs`, `fleet.css`: 함대/Run과 2D/3D의
  두 선택 축, 상시 Run 목록, 상단 상세 덮개, Fit·100%·확대·스크롤 복원을 제공한다.
  잘못된 needs 그래프는 텍스트 단계 목록으로 표시한다. 광고 만료로 노드를
  임의 삭제하거나 과거 assigned를 현재 점유로 표시하지 않는다.
- `static/demo/`: Guest 공개 관측 화면, 별도 완료 키의 네 단계 투어,
  두 시나리오 직접 제출 모달, 미확인 요청의 같은 ID 재시도를 구현했다.
  202 응답은 실제 run_id를 선택하되 서버 목록에 가짜 행을 넣지 않는다.
- `static/demo/webcam.mjs`, `settings.json`, `ui.go`: 방송 창 크기·배율·이동·맞춤·
  미니맵을 제공한다. 설정은 현재 `webcam: null`이며 방송 준비 중으로 표시한다.
  설정된 공개 HTTPS origin만 데모 HTML의 frame-src에 허용한다.
  iframe load는 방송 중이라는 증거로 사용하지 않는다.

HTTP 마운트·API·store·queue·config·DB 스키마는 수정하지 않았다. UI의 store
import는 없다. 공용 변경은 기존 landing과 guest의 좁은 접점, UI 정적 Handler다.
cardnews 자산과 landing.js는 변경하지 않았다. 테스트 데이터는 static 밖에 둔다.
신규 모듈 디렉터리도 디렉터리 목록을 노출하지 않는다.

## 디자인 입력 반영

팀원 `design/enode-demo.pen`과 D1~D5 export의 색상·간격·우측 목록·상세 덮개·
좌하단 방송 배치를 따랐다. 사용자가 지정한 `3d-view@8985694`의 SVG 등각 표현,
DAG level/lane·확대/스크롤 복원 방식을 선별 재사용했다. 원본 워크트리·pen은
변경하지 않았고 전체 브랜치 병합은 하지 않았다.

워크스테이션·보드 모형은 명시적 advertised device_type/model/kind/device 또는
board 속성에만 의존한다. 모르는 기기는 일반 모형, 미배정 단계는 빈 평면이다.
label·harness·requires로 기기 종류나 sandbox 보장을 추론하지 않는다.

## 검증과 남은 연결

[검증 기록](build-and-test.md)에 단위 테스트·실제 obs 연결·합성 상태·브라우저
조작을 구분했다. 외부 입력이 필요 없는 구현과 검증을 마쳤으며 다음 입력이 남았다.

1. queue 유닛 병합과 실제 QUEUED 재접수·승격 계약 인수.
2. 진행자가 확정한 LED/음원 픽스처, Guest 이름을 넣을 데이터 필드.
3. 공개 방송 공급자와 embed 주소, 실제 iframe 재생/음성 검증.
4. `capabilities[].attrs.sandbox` 표시 출처의 진행자 합의와 실제 광고 대조.

이 입력 없이 demo-back의 실제 allow-list 매핑·서버 제출 실행을 만들지 않았다.
현재 실제 `/v1/demo/runs`는 404이며 화면도 그 상태를 알린다.
CP9/CP10/CP11의 실제 하드웨어·방송·봉인 장면은 보류다.


## 후속 기록 — queue 인수·UI 개선

2026-09-09 queue@310c22d로 rebase한 결과와 최신 검증은
[queue 인수 기록](queue-integration-review.md), [UI 개선](ui-feedback.md)에 있다.
위의 queue 미수령·영문 이름·닫기 동작 기록은 당시 기준이며 후속 기록이 대체한다.
