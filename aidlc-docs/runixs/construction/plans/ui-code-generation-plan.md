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

- [x] **1 — 공용 모델과 계약 검증** (UI-02·03, CP1·CP2·CP7)
  model.mjs와 tests/model.test.mjs에 BR-01~12, 중첩 attrs·lease/null·submitter·chosen,
  부분 응답·needs의 중복/미존재/순환·ASKED·서버 시각·15초 갱신 지연을 구현·검증한다.
  검증: `node --test internal/api/ui/tests/model.test.mjs` 6 passed, 0 skipped.
- [x] **2 — 공용 2D/3D 표시** (UI-03, D2/D5)
  scene/view/CSS에 3d-view@8985694의 graphLayout·SVG·탐색을 선별 이식한다.
  팀원 pen의 모형·상세 덮개·Run 목록 배치를 반영한다. 모든 조작 testid,
  키보드 선택, Fit/100%/스크롤 복원과 긴/빈/잘못된 그래프를 로컬 브라우저로 검증한다.
  검증: Orca 로컬 합성 미리보기에서 6개 광고, 키보드 선택·표현 전환,
  30단계 그래프·확대·Fit·스크롤 복원·잘못된 그래프 텍스트 대체·빈 함대 통과.
- [x] **3 — obs 클라이언트** (UI-02)
  client.mjs와 tests/client.test.mjs에 인수한 wire 검사·GET·5초/4초 요청 수명·
  자원별 오류·401/404/429/503·Retry-After·늦은 응답 격리를 구현한다.
  실 모드만 asks와 Bearer를 사용한다. nodes 질의 인자와 fixture fallback은 없다.
  검증: `node --test internal/api/ui/tests/client.test.mjs` 9 passed, 0 skipped.
- [x] **4 — 실 함대 진입·연동** (UI-01·02)
  fleet 앱과 기존 관리자 폼을 연결한다. 인증 성공 전 자료 숨김·401/교체 시 정리,
  탭 수명·조회 실패·실제 obs 응답 표시를 검증한다. 기존 Guest 진입도 함께 확인한다.
  검증: 전용 로컬 PostgreSQL 16.14 + 실제 api.Handler에서 401/정상 진입,
  실제 POST 광고 3개·예제 Run 201 후 GET 폴링 표시 확인. 하드웨어 실행 증거 아님.
- [x] **5 — 데모 틀·Guest·공용 장면** (UI-03·04)
  demo placeholder를 D2/D5로 바꾼다. 3D 기본·Run 목록 유지·submitter·선택/필터를
  연결한다. Guest 정상값 보존/훼손값 복구와 기존 카드뉴스 흐름을 회귀 검증한다.
  검증: Guest 상태 4개 테스트, 실제 브라우저 첫 진입→카드 4장→데모,
  재방문→데모 분기와 같은 이름 보존. 기존 landing.js/cardnews 자산은 변경 없음.
- [x] **6 — 투어** (UI-04, CP8)
  tour.mjs와 tests/tour.test.mjs에 네 단계·완료/건너뛰기/Escape·다시 보기·
  storage 예외·빈 함대·resize·focus trap을 구현한다. 카드뉴스 키는 건드리지 않는다.
  검증: 상태 테스트 2개와 브라우저 4단계·완료/다시 보기·취소·선택/포커스 복원.
  390px iframe viewport에서도 spotlight 말풍선이 화면 안에 남는 것 확인.
- [x] **7 — 제출 모달·재시도 상태** (UI-05, CP9 화면)
  submission.mjs와 tests/submission.test.mjs가 공동 제출 계약을 소비한다.
  두 버튼·이름 자동 전달·한 의도 한 ID·10초 미확인·same-request 재시도·
  새로고침·늦은 응답·닫기·목록 갱신 지연을 검증한다. 합성 전송은 테스트에서만 쓴다.
  검증: 상태 테스트 14개. 실제 미구현 라우트의 404와 로컬 테스트 전송의
  503→동일 ID 재시도·새 의도·202 선택을 구분해서 확인. 가짜 목록 행은 생성하지 않음.
- [x] **8 — 웹캠 영역·공개 설정** (UI-06, CP11 화면)
  webcam.mjs/settings.json과 tests/webcam.test.mjs에 resize·zoom·pan·fit·미니맵·
  플레이어 입력 복귀·설정 없음/오류를 구현한다. ui.go는 같은 embed 설정을 검증해
  데모 HTML의 frame-src만 좁혀 연다. 기존 Header/Handler 인터페이스를 보존한다.
  검증: 표시 모델/URL 테스트 3개, Go 설정·헤더 검사, 브라우저 크기·배율·pan·Fit·
  플레이어 입력 복귀·방송 미설정 표시. 실제 공급자·음성은 9단계의 미수령 입력.
- [ ] **9 — 실제 데모 연결** (선행: 실제 픽스처·demo-back·방송 입력)
  진행자 값에 맞춰 고정 별칭 매핑·공개 방송 URL·sandbox 광고를 대조한다.
  demo-back 병합 뒤 실제 POST/GET·QUEUED·submitter·그래프 선택·방송을 확인한다.
  입력이 없으면 해당 연결은 보류하고 활성화/성공 장면을 만들어 내지 않는다.
  [인수 기록](../ui/code/queue-integration-review.md)에 따라 상태를 구분한다.

  - [x] queue 실제 202/QUEUED·재접수·승격·UI 그래프 갱신.
  - [x] sandbox 표시 출처: 최태양님 승인 전달 수령(2026-09-09).
  - [x] 로컬 실제 광고 API→2D/3D 표시: 능력별 값·미제공·추론 금지 검증(13개 단언).
  - [ ] 실제 장비 sandbox 광고·설정 대조.
  - [ ] 실제 LED/음원 계약과 demo-back POST/GET·이름·선택 연결.
  - [ ] 공개 방송 URL·플레이어·음성의 실제 연결.

- [ ] **10 — 전체 조작·Build and Test**
  FD의 실/데모 조작 전수를 브라우저로 확인한다. Node 테스트·UI Go 테스트·
  기존 Guest/cardnews·CP0 표준 검사와 필요한 공동 CP1/2/7/8/9/11을 기록한다.
  CP10은 실제 LED/음원·웹캠·봉인까지 공동 확인한다. 미실행 항목은 보류로 남긴다.
  독립 검사 결과: Node 42 통과, Go 전체 1,090 통과·0 스킵·16패키지 모두 80% 이상,
  UI 98.4%, vet/build/표기/Windows 빌드/취약점 검사 통과.
  [검증 기록](../ui/code/build-and-test.md)의 환경 제한과 공동 장면 보류를 유지한다.
- [ ] **11 — 결과·리뷰·커밋**
  implementation-summary와 build-and-test, 담당 state/audit를 갱신한다.
  단계 승인 시 담당 산출물을 커밋한다. PR/main 병합은 담당 게이트가 충족된 뒤다.
  [구현 요약](../ui/code/implementation-summary.md)과 검증 기록은 작성했다.
  2026-09-08T23:28:22Z 사용자가 현재 독립 구현의 커밋을 승인했다.
  지정 이메일 `runixs92@gmail.com`으로 코드·담당 기록을 커밋한다.
  실제 연결·전체 생성 완료 리뷰·PR 게이트는 남아 있다.

## 사용자 개선 — 2026-09-09

[UI 개선 변경 계획](ui-feedback-plan.md)을 기존 5·6·7·8단계의 보완으로 실행한다.
웹캠 닫기/복원, 한글 두 단어 Guest 이름, 상세/모달의 바깥 조작 닫기를 반영한다.

## 3. 검증과 진행 조건

코드 생성은 설계와 이 순서의 검토 뒤 시작한다. 독립 표시·투어·모달/방송 영역은
외부 입력 없이 만들 수 있다. queue 인수와 sandbox 출처 승인은 완료했다.
실제 시나리오·방송과 장비 광고 대조에 필요한 입력은 진행자를 통해 인수한다.

Go 1.26.6 로컬 경로와 Node 기본 테스트 실행기를 사용한다. UI 패키지 80% 하한,
테스트 자산 비공개·모듈 MIME·CSP를 확인한다. 전체 커버리지는 ci.yml의
`go test ./... -count=1 -coverpkg=./... -coverprofile=... -json` 측정과 집계로 판정한다.
단독 runctl 커버리지나 기존 Handler 테스트를 전체 게이트 증거로 사용하지 않는다.

로컬 미리보기·캡처는 ignored local/ 아래다. 외부 선행이 없는 검증을 먼저 하되
현재/합성/실제 하드웨어의 증거를 구분한다. 새 라이브러리·UI 프레임워크는 추가하지 않는다.
