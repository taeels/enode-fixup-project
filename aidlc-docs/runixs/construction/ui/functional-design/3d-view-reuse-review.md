# 3d-view 코드 재사용 검토

**판정**: 3D 표시와 그래프 탐색 코드를 선별 재사용할 것을 권고한다. 팀원의
`design/enode-demo.pen`을 최종 화면 기준으로 유지한다. 이번 작업은 검토·계획
보완까지이며 소스 브랜치를 병합하거나 코드를 복사하지 않았다.

## 1. 소스와 범위

| 항목 | 확인값 |
|---|---|
| Orca 워크트리 | `/Users/runixs/orca/workspaces/enode-fixup-project/3d-view` |
| 실제 브랜치 | `unit/ux-runtime` (표시 이름 3d-view) |
| 검토 HEAD | `89856940889444f72bf4d3f14c7355fdaf41b846` |
| 대상 브랜치 | `unit/runixs-ui`, `81b526c` + 담당 설계 변경 |
| 공통 조상 | `06215ff1be927cdf13021a48b1cb05db14dc2802` |
| 주 소스 | `internal/api/ui/assets/app.js`, `model.js`, `app.css` |
| 최근 기능 | `8985694`: 20단계 예시·전체 맞춤·확대/축소·보기별 스크롤 보존 |
| 소스 상태 | 추적된 변경 없음, 기존 `.omc/`만 미추적. 검토 중 소스 수정 없음 |

소스 브랜치는 공통 조상부터 102개 파일에 obs·queue·drain·panel·MCP·transcript·
광고 철회와 시안/요구 변경까지 포함한다. 현재 main의 담당별 계약·게이트를 이
브랜치의 구현이나 과거 검증 결과로 대체할 수 없다.

## 2. 채택할 코드

아래 파일 경로는 위 소스 워크트리의 `internal/api/ui/assets/` 기준이다.

| 원본 | 재사용할 부분 | 적용 방식 |
|---|---|---|
| `model.js:39` graphLayout | needs 기반 깊이·병렬 lane, 2D/등각 좌표 | 현재 의미 모델로 계산. 누락 참조·순환·중복 검증 추가 |
| `app.js:170` isoFleet | SVG 바닥·모형·상태색·노드 선택·키보드 조작 | scene 모듈로 추출. D2의 기기 표현·배치에 맞춤 |
| `app.js:224` graph | 단계/간선·3D 측면·맞춤/확대/100% | D5 중앙 표시 영역에 넣고 Run 목록 유지 |
| `app.js:249` render 일부 | Run/보기별 scrollLeft/scrollTop 저장·복원 | 현재 상태 관리에 옮김. 전체 innerHTML 교체는 그대로 옮기지 않음 |
| `app.js:310` 그래프 조작 | 중심을 보존한 확대/축소 | 고정 testid·키보드·범위 검사 추가 |
| `app.css:447` 이후 관련 규칙 | 모형·간선·포커스·scroll/fit 스타일 | fleet/demo 루트에 범위 제한. 전역 body/button 규칙은 제외 |
| `scripts/test-ux-model.mjs` | DAG·병렬 시작·속성 비교 사례 | 현재 tests에서 계약·시각 경계 사례와 함께 검증 |

3D는 외부 엔진 없는 **SVG 등각 표현**이다. 회전 가능한 WebGL 장면은 아니며
현재 pen의 탑뷰 표현을 구현하는 기반으로 사용할 수 있다. 함대는 같은 상자 모형,
Run은 측면을 더한 단계 카드다. 팀원 D2/D5의 PC·보드 표현과 상세 패널 배치까지
이미 구현됐다고 보지는 않는다.

## 3. 이식 전 수정할 차이

| 대상 | 관찰·근거 | 조치 |
|---|---|---|
| Run 목록 | runPage에서 runList를 호출하지 않음. 실제 그래프 화면에 목록 없음 | 공통 shell에 목록 유지. decisions §8.3·D5 준수 |
| landing/guest | 소스는 assets embed, 현재는 static embed와 landing·cardnews·guest | 기존 ui.go/index를 덮지 않고 신규 자산 선별. 마운트 재사용 |
| 인증 자료 수명 | 토큰 변경 때 nodes/runs/detail은 남고, 다음 입력 때 조회 성공 전 render | 인증 성공 뒤 진입, 세대 변경 시 보호 자료·요청 정리 |
| 폴링 | nodes/runs/asks/detail을 Promise.all로 묶고 failures 하나 사용 | 상세 404·asks 오류가 전체 관측을 막지 않도록 자원별 상태 |
| API 계약 | matches는 평탄 requires 전제. chosen 미제공도 `=== true`로 false 표시 | obs D1~D5 확인 후 어댑터. 미제공 의미 보존 |
| 시각·노드 상태 | lease/drain이 있으면 만료 임박 검사 안 함. 실패 후 now는 과거 observed로 복귀 | 만료 흐림을 독립 표시, 시간 정지·회복은 현재 FD 적용 |
| DAG 오류 | 없는 선행 id 무시, 순환을 오류로 보고하지 않음 | 누락·순환·중복 검증 뒤 그래프 또는 오류 상태 표시 |
| 데모 기능 | Guest·투어·모달·웹캠·submitter 표시 없음. capability/sandbox 강조도 없음 | demo pen과 팩에 따른 데모 FD에서 보완 |

인증 자료 수명과 폴링 문제는 코드 흐름에서 확인한 통합 위험이다. 이번에 서버를
중단하거나 인증 실패를 주입해 재현하지 않았으며 이식 검증 항목으로 남긴다.

## 4. 이번에 직접 실행한 검증

2026-09-08, 읽기 전용 조회와 화면 조작으로 확인했다. 노드 시작/중지나 Run
제출·취소·답변을 하는 스크립트는 실행하지 않았다.

- Node v24.1.0, `node --test scripts/test-ux-model.mjs`: 3개 통과·스킵 0.
- 로컬 Go 1.26.6, `go test -count=1 -cover ./internal/api/ui`: 통과,
  100.0% statements. 정적 Go Handler 수치이며 JS 커버리지가 아니다.
- `http://127.0.0.1:18080/ui/`의 app.js와 검토한 소스 파일의 bytes 일치.
- Orca 별도 검토 탭에서 함대 3D 전환: 노드 모형 6개·Run 목록 표시.
- `release-demo-verified`: SUCCEEDED, 20단계, 18 DONE·2 SKIPPED.
- 3D 그래프 Fit에서 scrollWidth ≤ clientWidth + 1.
- 100%에서 가로 스크롤 500px 설정 후 Refresh에도 500px 유지.
- Graph에서 300px로 옮겼다가 3D로 돌아오면 이전 500px 복원.
- Zoom in으로 100% → 125% 전환.
- 같은 Run 그래프에서 Run 목록 부재 직접 확인.
- 모델 탐침: 평탄 요구는 매칭, 동등한 중첩 attrs는 미매칭. 임대 중 만료 20초 전
  노드는 leased만 반환. 미존재 needs를 오류 없이 배치.

검토 탭은 닫았다. 기존 서비스·데이터·다른 터미널은 변경하지 않았다. 소스의 과거
브라우저 기록은 참고만 했다. `release-demo-review`는 과거 기록의 ASKED와 달리
현재 FAILED였으므로 당시 사람 대기 결과를 현재 재검증으로 사용하지 않는다.

## 5. 계획 반영

코드 생성 계획 1~3에 graphLayout·SVG scene·탐색 처리의 선별 재사용을 반영한다.
소스 HEAD가 바뀌면 차이를 확인한다. 의미 모델과 조회/인증 코드를 분리해 2D/3D가
같은 노드·Run·선택을 소비하게 한다. 팀원 pen에 맞춘 목록·기기·상세 배치를 적용하며
D1·D3·D4·웹캠은 별도 데모 FD 대상으로 유지한다. obs 이후에는 현재 main의 응답
계약에 연결한다. 소스 브랜치의 서버 구현을 함께 들여오지 않는다.

전체 W1/W3 코드 생성 승인 상태는 바뀌지 않았다. 이 검토는 선별 재사용 권고와
계획 보완의 근거이며 기능 통합이나 CP 게이트 완료를 뜻하지 않는다.
