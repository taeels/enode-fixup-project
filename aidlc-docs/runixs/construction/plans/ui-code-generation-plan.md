# ui 코드 생성 계획 — 실 함대·공개 데모

**승인**: 2026-09-08T20:11:36Z “응 시작해”. Code Generation 진행.

**작성 당시 상태**: 2026-09-09 FD와 함께 검토할 전체 계획 초안. 실행 체크는 모두 미착수다.
기존 W1 8단계를 이 순서로 대체했다. 구현 기준 `0a159a4`, 설계 gitlink `29c89cd`.
[FD 계획](ui-functional-design-plan.md)과 [통합 입력·리뷰](../design-review.md)를 따른다.

## 1. 파일과 소유 경계

| 경로 | 내용 |
|---|---|
| static/shared/fleet/model.mjs | 관측 계약 검사·카드/단계/시간 의미 모델 |
| static/shared/fleet/scene.mjs, view.mjs, fleet.css | 선별 SVG·배치·탐색·DOM 표시. dashboard root에 스타일 제한 |
| static/shared/fleet/client.mjs | 모드별 GET·폴링·오류·세대·취소·공개 한도 처리 |
| static/fleet/index.html, app.mjs, login.mjs | 실 함대 S0~S2 진입·앱 조정 |
| static/demo/index.html, demo.js, demo.css | D2/D5 실제 화면과 Guest·공용 모듈 연결 |
| static/demo/tour.mjs, submission.mjs, webcam.mjs, settings.json | 투어·제출 의도·방송 표시·공개 설정 |
| static/index.html, landing.css | 기존 관리자 폼 연결과 D1 Guest 이름 표시 |
| static/shared/guest.js | 정상 저장 이름 보존, 훼손된 이름만 기존 생성기로 복구 |
| ui.go, 신규 fleet_test.go·demo_test.go | 기존 embed/Handler 재사용, 설정 검증·데모 frame-src |
| tests/*.test.mjs, testdata/obs-contract | Node 계약/상태/경쟁 검증·합성 입력. 공개 static 밖 |

위 경로는 `internal/api/ui/` 기준이다. 기존 landing.js·cardnews 콘텐츠/마지막
이동·Guest 저장 키를 보존한다. 공유 변경은 nacl1119의 최신 파일과 대조한다.
API/DB를 UI 패키지로 옮기지 않는다. 3d-view 전체 앱·서버·ui.go는 복사하지 않는다.

## 2. 실행 순서

- [ ] **1 — 공용 모델과 계약 검증** (UI-02·03, CP1·CP2·CP7)
  model.mjs와 tests/model.test.mjs에 BR-01~12, 중첩 attrs·lease/null·submitter·chosen,
  부분 응답·needs의 중복/미존재/순환·ASKED·서버 시각·15초 갱신 지연을 구현·검증한다.
- [ ] **2 — 공용 2D/3D 표시** (UI-03, D2/D5)
  scene/view/CSS에 3d-view@8985694의 graphLayout·SVG·탐색을 선별 이식한다.
  팀원 pen의 모형·상세 덮개·Run 목록 배치를 반영한다. 모든 조작 testid,
  키보드 선택, Fit/100%/스크롤 복원과 긴/빈/잘못된 그래프를 로컬 브라우저로 검증한다.
- [ ] **3 — obs 클라이언트** (UI-02)
  client.mjs와 tests/client.test.mjs에 인수한 wire 검사·GET·5초/4초 요청 수명·
  자원별 오류·401/404/429/503·Retry-After·늦은 응답 격리를 구현한다.
  실 모드만 asks와 Bearer를 사용한다. nodes 질의 인자와 fixture fallback은 없다.
- [ ] **4 — 실 함대 진입·연동** (UI-01·02)
  fleet 앱과 기존 관리자 폼을 연결한다. 인증 성공 전 자료 숨김·401/교체 시 정리,
  탭 수명·조회 실패·실제 obs 응답 표시를 검증한다. 기존 Guest 진입도 함께 확인한다.
- [ ] **5 — 데모 틀·Guest·공용 장면** (UI-03·04)
  demo placeholder를 D2/D5로 바꾼다. 3D 기본·Run 목록 유지·submitter·선택/필터를
  연결한다. Guest 정상값 보존/훼손값 복구와 기존 카드뉴스 흐름을 회귀 검증한다.
- [ ] **6 — 투어** (UI-04, CP8)
  tour.mjs와 tests/tour.test.mjs에 네 단계·완료/건너뛰기/Escape·다시 보기·
  storage 예외·빈 함대·resize·focus trap을 구현한다. 카드뉴스 키는 건드리지 않는다.
- [ ] **7 — 제출 모달·재시도 상태** (UI-05, CP9 화면)
  submission.mjs와 tests/submission.test.mjs가 공동 제출 계약을 소비한다.
  두 버튼·이름 자동 전달·한 의도 한 ID·10초 미확인·same-request 재시도·
  새로고침·늦은 응답·닫기·목록 갱신 지연을 검증한다. 합성 전송은 테스트에서만 쓴다.
- [ ] **8 — 웹캠 영역·공개 설정** (UI-06, CP11 화면)
  webcam.mjs/settings.json과 tests/webcam.test.mjs에 resize·zoom·pan·fit·미니맵·
  플레이어 입력 복귀·설정 없음/오류를 구현한다. ui.go는 같은 embed 설정을 검증해
  데모 HTML의 frame-src만 좁혀 연다. 기존 Header/Handler 인터페이스를 보존한다.
- [ ] **9 — 실제 데모 연결** (선행: 실제 픽스처·demo-back·방송 입력)
  진행자 값에 맞춰 고정 별칭 매핑·공개 방송 URL·sandbox 광고를 대조한다.
  demo-back 병합 뒤 실제 POST/GET·QUEUED·submitter·그래프 선택·방송을 확인한다.
  입력이 없으면 해당 연결은 보류하고 활성화/성공 장면을 만들어 내지 않는다.
- [ ] **10 — 전체 조작·Build and Test**
  FD의 실/데모 조작 전수를 브라우저로 확인한다. Node 테스트·UI Go 테스트·
  기존 Guest/cardnews·CP0 표준 검사와 필요한 공동 CP1/2/7/8/9/11을 기록한다.
  CP10은 실제 LED/음원·웹캠·봉인까지 공동 확인한다. 미실행 항목은 보류로 남긴다.
- [ ] **11 — 결과·리뷰·커밋**
  implementation-summary와 build-and-test, 담당 state/audit를 갱신한다.
  단계 승인 시 담당 산출물을 커밋한다. PR/main 병합은 담당 게이트가 충족된 뒤다.

## 3. 검증과 진행 조건

코드 생성은 설계와 이 순서의 검토 뒤 시작한다. 독립 표시·투어·모달/방송 영역은
외부 입력 없이 만들 수 있다. sandbox 출처의 공용 확인과 실제 시나리오·방송은
진행자 입력을 기다리며, 그 사실을 UI 구현 전부의 대기로 확대하지 않는다.

Go 1.26.6 로컬 경로와 Node 기본 테스트 실행기를 사용한다. UI 패키지 80% 하한,
테스트 자산 비공개·모듈 MIME·CSP를 확인한다. 전체 커버리지는 ci.yml의
`go test ./... -count=1 -coverpkg=./... -coverprofile=... -json` 측정과 집계로 판정한다.
단독 runctl 커버리지나 기존 Handler 테스트를 전체 게이트 증거로 사용하지 않는다.

로컬 미리보기·캡처는 ignored local/ 아래다. 외부 선행이 없는 검증을 먼저 하되
현재/합성/실제 하드웨어의 증거를 구분한다. 새 라이브러리·UI 프레임워크는 추가하지 않는다.
