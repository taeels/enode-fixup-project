# ui 화면 구성 — 실 함대·팀원 데모 D1~D5

**상태**: 구현 전 검토안. [업무 흐름](business-logic-model.md)과
[규칙](business-rules.md)을 소비한다. W1과 아래 D1~D5 설계를 한 ui 유닛으로 다룬다.

**디자인 기준**: W1은 enode-ux.pen, 데모는 enode-demo.pen D1~D5다.
[반영 범위 대조](design-targets.md)의 이전 부분 계획을 이번 설계로 보완했다.
디자인 설계 반영과 실제 화면 구현·시연 완료는 구분한다.

**재사용 입력**: [3d-view 검토](3d-view-reuse-review.md)의 SVG 모형·DAG 배치·
확대/스크롤 처리를 공용 표시 모듈 후보로 삼는다. 2D/3D 표현 선택은 현재
격자/작업 그래프 선택과 구분해 관리한다. 어느 조합에서도 Run 목록을 유지하며
소스 앱의 인증·전역 render·폴링을 그대로 가져오지 않는다. 아래 조작 목록은
W1 기본 목록이고, 3D 조작 검증은 코드 계획 3단계에 추가했다.

## 1. 진입과 파일 접점 (D7)

`/ui/` 기본은 D1 Guest 카드, '실 함대 접속' 링크는 같은 페이지의 `#fleet`에서
기존 관리자 입력을 S0 토큰 폼으로 활성화한다. '#demo'로 Guest 카드에 돌아온다.
hash는 보기 선택만 하며 인증/서버 모드를 바꾸지 않는다. nodes/runs 인증 확인
성공 뒤 `/ui/fleet/`으로 이동한다. 401은 `/ui/#fleet`의 S0b로 돌아가고 토큰은 남기지
않는다. `/ui/fleet/`의 직접 방문·새로고침도 세션과 필수 조회를 다시 확인한다.
오류 안내 상태는 토큰과 별도의 세션 값이며 표시 후 지운다.

`static/fleet/`는 실 함대 진입 자산, `static/shared/fleet/`은 두 모드의 공용
표시·관측 모듈이다. `static/index.html` 과
`static/landing.css` 의 관리자 폼 접점만 obs 연결 단계에서 변경한다. 별도 외부
모듈 스크립트를 추가하면 기존 landing.js 의 Guest 진입 함수를 수정할 필요가 없다.
기존 `landing-admin-token-input`, `landing-guest-login-button` 식별자는 유지한다.
공유 파일 변경은 nacl1119의 최신 카드뉴스 변경과 대조한 뒤 PR 접점으로 명시한다.

`/ui/` 마운트는 이미 `internal/api/api.go` 에 있다. `ui.Handler()`와 static embed를
재사용한다. 방송 설정은 정적 demo/settings.json에 두고 기존 Handler가 읽어
데모 HTML의 frame-src를 설정한다. store 임포트·별도 UI 서버는 만들지 않는다.

## 2. 구성·입력·상태

| 구성 | 입력 | 소유 상태·조작 | HTTP 연결 |
|---|---|---|---|
| TokenEntry | 인증 결과, 오류 종류 | token 입력, 제출 중, 재입력 | nodes/runs 최초 확인 |
| FleetShell | AuthSession, ResourceState | 선택 Run·step·격자/그래프·필터 | 자원별 polling 조정 |
| ConnectionStatus | 마지막 성공·실패 횟수 | 재시도·인증 종료 | 실패 자원 재조회 |
| FleetGrid / NodeCard | nodes, 임대 Run 상세, asks | 임대 Run 선택 | nodes + 관련 상세 |
| RunList / RunRow | runs, 현재 선택, 필터 | Run 선택, 상태 필터 | runs |
| RequirementList | 상세 requires, 노드 광고 | 읽기 전용 | 상세 |
| RunGraph / StepInspector | steps·needs·chosen·assigned | 단계 선택, 격자로 복귀 | 상세 |
| AskNotice | ASKED, 인박스, 수신 상태 | 읽기 전용 명령·질문 안내 | asks + 상세 |

Run 목록은 격자·그래프 양쪽에서 유지한다. 좁은 화면에서는 목록을 아래로 쌓되
선택을 바꿀 수 있는 위치에 남긴다. 그래프 안의 단계는 키보드로 선택할 수 있는
버튼이며, 간선 장식과 별도로 needs·state 를 텍스트 목록에서도 확인할 수 있다.

## 3. 조작 전체 목록과 완료 기준

실제 구현 완료 시 아래 항목마다 검증 결과를 기록한다. 지금의 체크박스는 미실행이다.

- [ ] S0 토큰 입력 `landing-admin-token-input`: 가림·빈 값/줄바꿈 검증·Enter 제출.
- [ ] S0 제출 `landing-admin-login-button`: 조회 중 중복 제출 방지, 성공 후 S1.
- [ ] S0b 재입력: 401 안내, 보호 자료 미표시, 같은 폼으로 다시 인증.
- [ ] S0 조회 실패 재시도 `landing-admin-retry-button`: 503과 401 구분.
- [ ] 기존 Guest 버튼 `landing-guest-login-button`: 기존 카드뉴스/데모 분기 유지.
- [ ] `landing-admin-mode-button`, `landing-guest-mode-button`: D1/S0 표시·hash 직접 진입·포커스·입력 보존 경계.
- [ ] S1 인증 종료 `fleet-logout-button`: 세션 토큰·상태 삭제 후 S0.
- [ ] S1 재시도 `fleet-retry-button`: 실패 자원 재조회, 중복 요청 없음.
- [ ] 격자 `fleet-grid-button` / 그래프 `fleet-graph-button`: 선택 유지, 미선택 그래프 비활성.
- [ ] Run 필터 `fleet-run-state-filter`: 전체·QUEUED·RUNNING·VERIFYING·SUCCEEDED·FAILED,
  현재 관측 목록 안에서 필터. 선택 상세를 임의로 없애지 않음.
- [ ] Run 선택 `fleet-run-select-button`: 행의 안정적인 data-run-id 와 함께 식별.
- [ ] 노드 임대 Run 선택 `fleet-node-run-button`: data-node-id 로 식별, 유휴 노드에는 없음.
- [ ] 단계 선택 `fleet-step-select-button`: data-step-id 로 식별, 실제 단계 정보 표시.
- [ ] 탭 닫기 후 새 탭: 다시 토큰 입력. 새로고침: 세션 값으로 인증 재확인.

위 ID 는 역할별로 고정하고 반복 항목은 서버의 안정적인 식별자 속성을 함께 쓴다.
노드 start/stop/drain, 작업 submit/cancel, 질문 answer 버튼은 읽기 전용 범위에 없다.

## 4. 화면 시안 반영

`design/enode-ux.pen` 의 S0/S1/S1b/S2 프레임과 S1·S2 export 를 읽었다.
짙은 바탕의 운영 현황판, 왼쪽 함대 격자/그래프와 오른쪽 Run 목록 구조를 따른다.

| 토큰 | pen 값 |
|---|---|
| 바탕 / 표면 / 둘째 표면 | #0B0D10 / #14181D / #1B2027 |
| 경계 / 강조 경계 | #262D36 / #39434F |
| 본문 / 보조 글자 | #E6EAF0 / #98A4B3 |
| 유휴 / 임대 / drain | #3FBF87 / #4B9CFF / #B98CFF |
| 대기 / 실패 / 만료 임박 | #E2A33C / #F0574B / #7A8698 |
| 간격 / 모서리 | 8·16·24px / 10px |

pen의 Inter·JetBrains Mono를 첫 글꼴 후보로 두고 로컬 system/monospace 대체
글꼴을 쓴다. 이를 위해 외부 폰트를 내려받지 않는다. 본문은 시안의 작은 주석
크기를 그대로 확대 복제하지 않고 읽을 수 있는 크기로 조정한다. 낮은 대비의
text-dim 색은 필수 상태 안내에 쓰지 않는다.

색과 함께 상태 문구를 표시하고 포커스 테두리·label·aria-live 오류 영역을 둔다.
폴링은 포커스를 빼앗지 않는다. 필수 정보는 hover 에만 숨기지 않는다. S2 export의
단계 RUNNING 표기는 실제 CLAIMED로, 조회에 없는 produced 표시는 자료 미제공으로
다룬다. 페이지네이션 장식도 구현하지 않는다. `.pen` 파일은 수정하지 않는다.

## 5. 검증과 의존

사전 독립 화면은 의미 모델을 주입하는 로컬 검증에서만 합성 입력을 사용한다.
배포 자산에 자동 mock 모드·샘플 로딩 버튼·장애 시 fixture 대체를 넣지 않는다.
실제 API 연결은 [코드 계획](../../plans/ui-code-generation-plan.md)의 obs 확인 뒤다.
CP1·CP2·CP7의 API/화면 증거는 각각 기록한다. C4·C5 실시간 일치는 drain/panel
장면을 기다린다. W1 표시 완료가 W3 데모·CP10 완료를 뜻하지 않는다.

## 6. 데모 구성·입력·상태

| 구성 | 입력 | 소유 상태·조작 | 연결 |
|---|---|---|---|
| DemoEntry / GuestBadge | 기존 guest 이름/온보딩 판정 | Guest 진입·이름 표시 | 기존 landing/guest/cardnews 흐름 |
| DemoShell | 공개 관측·SceneState·투어·제출 상태 | 상단 상태·표현 선택·접수 안내 | 토큰 없는 조회 셋과 제출 계약 |
| FleetScene / RunScene | 공용 모델·SVG·표현 상태 | 노드/단계 선택·뷰 복귀·Fit/확대/위치 | 기존 의미 모델 재사용 |
| NodeInspector | 선택 광고·실제 임대/Run | label·capability·sandbox·시간·상태 | nodes와 관련 detail |
| StepInspector | 선택 단계·연결 노드 | 상태·needs·chosen·attempt·시각 | detail, 필요 시 nodes |
| DemoRunList | 실제 runs·submitter·선택·필터 | 행 선택·새 작업 | runs. 모달/투어 이외 계속 접근 가능 |
| TourOverlay | 네 대상 요소·투어 상태 | 다음·건너뛰기·시작하기 | 투어 완료 키만 저장 |
| NewTaskDialog | Guest 이름·제출 상태 | LED/음원 요청·닫기·재시도 | 공동 POST 계약 |
| WebcamWindow | 공개 설정·WebcamState | resize·zoom·pan·fit·재연결 | 브라우저→방송 공급자 |

공용 코드는 전역 앱 싱글턴을 만들지 않는다. 앱별 root DOM·mode·client·state를
주입하며 실 함대 세션과 공개 데모의 캐시/오류 처리를 분리한다. 공용 CSS는
dashboard root 아래로 범위를 제한해 기존 카드뉴스·랜딩의 스타일을 바꾸지 않는다.

## 7. D1~D5의 배치

D1은 시안의 가운데 610px 카드에 실제 Guest 이름·Guest 로그인 버튼을 표시한다.
실 함대 모드 전환은 카드 아래의 작은 링크다. 관리자 토큰 입력을 D1 카드 안에
함께 노출하지 않는다. 기존 버튼과 입력의 testid·Guest 진입 핸들러는 유지한다.

1920×1080 시안 기준 상단 72px, 바깥/열 간격 24px, 오른쪽 목록 460px,
나머지 왼쪽은 3D 평면이다. 노드·단계 상세는 왼쪽 평면 우상단 덮개 카드다.
별도 상시 상세 열을 만들지 않는다. webcam은 왼쪽 아래에 떠 있고 평면은 그
아래까지 이어진다. 3D 워크스테이션/보드 형태는 시안과 기존 SVG를 대조해 다듬는다.

좁은 화면에서는 왼쪽 장면 다음에 Run 목록을 쌓는다. 목록은 숨기지 않고
화면 세로 스크롤로 접근한다. 두 축은 장면 선택(fleet/run)과 표현 선택(2d/3d)이며,
표현 전환으로 선택 Run/노드/단계를 지우지 않는다. 데모의 최초 표현은 3D다.
새 제출 직후 run+3d로 전환한다. 긴 그래프의 Fit/100%와 스크롤 복원을 유지한다.

투어는 시안 네 대상에 실제 spotlight를 맞춘다. 빈 함대/오류에서도 대상 컨테이너가
존재한다. resize 때 위치를 다시 계산하고 좁은 화면에서는 대상을 먼저 보이게 한다.
말풍선은 화면 밖으로 나가지 않게 제한한다. D4는 폭 최대 720px·60% 배경 덮개,
두 시나리오 버튼·닫기·상태 안내이며 이름 칸이나 별도 최종 제출 버튼은 없다.

## 8. 웹캠 조작

기본 360px 정사각형, 최소 200px·최대 720px다. 실제 최대는 왼쪽 패널의
폭/높이에서 48px를 뺀 값까지로 제한한다. 패널이 248px 미만이면 최소값보다
패널 안에 들어오는 것을 우선한다. 왼쪽 아래 위치는 고정하고 우상단 손잡이로
크기를 바꾼다. 조절 중 영상 비율 1:1과 레이아웃 바깥 배치를 유지한다.

영상 표시는 1~4배, +/-는 0.25배, 맞춤은 1배·중앙이다. 확대 중 pan은 프레임이
빈 영역을 보이지 않는 범위로 제한한다. 1배를 넘으면 전체 사각형과 현재 보이는
범위를 나타내는 미니맵을 보인다. 외부 프레임을 캡처한 섬네일을 만들지 않는다.

iframe의 원래 재생/음소거 조작을 보존한다. 별도 '화면 이동' 토글을 켰을 때만
iframe 앞의 입력 레이어로 드래그·휠/핀치·더블클릭 1↔2배를 받는다. 토글을 끄면
플레이어로 입력을 돌려준다. 외부 버튼의 +/-/맞춤은 항상 사용 가능하다.
키보드로도 크기 손잡이와 pan/zoom을 조작할 수 있고 Escape로 이동 모드를 종료한다.
공급자가 확정되면 실제 iframe·음성 재생·gesture 충돌을 검증한다.

## 9. 데모 조작 전수와 구현 후 검증

아래는 구현 완료 체크이며 아직 모두 미실행이다. 반복 항목은 data-node-id,
data-run-id, data-step-id로 식별하고 testid 자체에 무작위 값을 넣지 않는다.

- [ ] 기존 `landing-guest-login-button`: 첫 방문/재방문 분기·정상 Guest 이름 보존.
- [ ] 기존 `demo-replay-cardnews-link`: 카드뉴스 다시 보기·기존 마지막 이동 보존.
- [ ] `demo-tour-replay-button`: 1단계부터 시작, 선택 Run 유지.
- [ ] `demo-tour-next-button`, `demo-tour-skip-button`, `demo-tour-start-button`: 네 단계·완료 기록·Escape·포커스 복귀.
- [ ] `demo-2d-button`, `demo-3d-button`, `demo-fleet-button`, `demo-run-button`: 두 선택 축·선택 보존·미선택 Run 비활성.
- [ ] `demo-node-select-button`, `demo-node-run-button`: 광고 상세·현재 임대 Run 이동.
- [ ] `demo-run-state-filter`, `demo-run-select-button`: VERIFYING 포함 상태 필터·실제 행 선택.
- [ ] `demo-step-select-button`, `demo-step-node-button`: 단계 상세·실제 노드로 이동, 노드 미제공이면 이동 비활성.
- [ ] `demo-graph-fit-button`, `demo-graph-zoom-in-button`, `demo-graph-zoom-out-button`, `demo-graph-actual-size-button`: Fit·100%·긴 그래프·표현별 위치 복원.
- [ ] `demo-retry-button`: 실패 자원만 재조회, 429 대기·중복 요청 제한.
- [ ] `demo-new-task-button`, `demo-task-close-button`: 함대/그래프에서 열기·이전 화면 복귀·focus trap·Escape.
- [ ] `demo-task-led-button`, `demo-task-audio-button`: 한 클릭 한 의도·중복 차단·Guest 자동 전달.
- [ ] `demo-task-retry-button`, `demo-task-new-intent-button`: 동일 요청 재시도/새 작업 구분·응답 유실·새로고침.
- [ ] `demo-webcam-resize-handle`: 드래그·키보드 크기 조절·창 경계·목록 고정.
- [ ] `demo-webcam-zoom-in-button`, `demo-webcam-zoom-out-button`, `demo-webcam-fit-button`: 배율·pan 제한·미니맵.
- [ ] `demo-webcam-pan-button`: 이동 모드·드래그/휠/핀치/더블클릭·플레이어 입력 복귀.
- [ ] `demo-webcam-reconnect-button`: 실제 공급자 재연결, iframe load와 LIVE 구분.
- [ ] 공급자 재생/음소거: 실제 방송·음성·사용자 재생 허용을 확인하고 공급자별 결과 기록.

모달/투어의 포커스 trap·aria-live 접수 결과·축소 동작 설정을 확인한다.
실제 방송·시나리오를 연결하지 않은 조작 테스트만으로 CP8~CP11을 완료하지 않는다.
