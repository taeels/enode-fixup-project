# 원격 실행 가치와 장비 식별 UI

사용자가 새 작업 카드에서 enode의 필요성을 알기 어렵고 그래프의 같은 색 때문에
하나의 PC에서 실행하는 것처럼 보인다고 요청했다. 기존 승인된 ui Construction의
[후속 계획](../../plans/ui-remote-identity-plan.md)에 따라 구현했다.

## 사용자에게 보이는 변화

- LED Toggle: 타인의 에이전트로 device 컨트롤! 결과는 웹캠으로 확인
- 사운드 재생: 타인의 에이전트로 음성을 만들고 원격 장비에서 재생! 결과는 웹캠으로 확인
- 해커톤에 의견 남기기: 타인의 sandbox 내 에이전트로 해커톤 댓글남기기

각 카드에 브라우저→실행 enode 또는 음성 합성→재생 enode 경로도 표시한다.
작업 목록과 선택 Run 그래프는 같은 작업색을 쓴다. 실행 단계와 함대 노드는
호스트별 색을 공유하며 소유자 표시·hostname·OS/CPU·담당 역할을 읽을 수 있다.
브라우저 요청자, Mediator 서버, 실행 장비를 독립된 요소로 그린다. 좁은 화면은
세로로 스크롤하며 첫 관측 장면에서 처음 장비부터 보이도록 스크롤 위치를 보완했다.

## 관측 의미와 구현 경계

`shared/fleet/identity.mjs`가 결정적 작업/호스트 색, 표시용 identity와 역할을
계산한다. `scene.mjs`·`view.mjs`가 2D/3D·목록·상세에 연결한다. 소유자는 owner
광고, 없으면 canonical label의 handle이며 인증된 소유권으로 주장하지 않는다.
hostname은 명시적 광고 또는 label에서 읽고, 장비 종류는 실제 광고로만 판별한다.
CPU는 host_arch이며 빌드 대상 arch를 호스트 CPU로 표시하지 않는다.

board는 연결된 보드 광고다. Windows+board를 Raspberry Pi 호스트로 그리지 않는다.
VM은 sandbox/device_type의 명시적 광고로 구분한다. 같은 hostname은 같은 색이며
계산한 호스트 수를 물리 PC 수로 단정하지 않는다. 단순 단계 연결에는 같은/다른
호스트를 표시한다. 상태색과 상태 문자열은 작업/장비색과 독립적으로 유지한다.

현재 nodes에 없는 과거 단계는 assigned의 당시 label만 사용하고 상세 장비는
미관측으로 표시한다. step.node가 없으면 배정 대기이며 후보 노드를 대입하지 않는다.
역할은 requires에서 찾고 임의의 단계 이름에서 추정하지 않는다. 전체 Run/단계 ID,
needs·선택/지연/실패 의미, 제출 동작, 기존 gallery 표시와 대화 이력을 보존한다.
API/DB/노드 광고·인증·MCP 권한을 변경하지 않는다.

## 검증

- Node 74/74, skip 0. owner/host/config, 결정적 색, 연결 장치/VM/CPU, 과거 label,
  미배정, 역할과 모순된 광고 사례를 포함한다.
- `go test -race -cover ./internal/api/ui`: 통과, 98.4%. `go vet ./internal/api/ui`,
  gofmt·glyphscan 및 Mediator 빌드 통과.
- demo/fleet × desktop/mobile 네 진입 환경에서 2D/3D, 작업·장비색, 긴 host명,
  역할·연결 보드·VM, 첫 모바일 스크롤, 미배정/과거 배정, 같은 호스트,
  분기 DAG·지연 애니메이션, overflow·브라우저 오류 없음을 fixture로 검증했다.
- 기존 갤러리 history 조회와 작성 중 메시지 보존을 fixture로 확인했다. 실제
  댓글 게시·Claude·음원·LED 작업은 제출하지 않았다.

main `2d3f804`의 YouTube Live 설정 인수 후 기존 보안 헤더 테스트의 고정 기대값이
실패했다. 데모 HTML만 설정된 embed origin을 허용하고 다른 페이지는 self를 유지하는
기대값으로 보완해 재통과했다. production CSP와 origin 검증은 변경하지 않았다.
iframe 내부에서 storage를 만지던 로컬 브라우저 harness도 최상위 프레임으로 한정했다.
근거는 Git 제외 `local/gallery-build-20260909/identity-node-tests.txt`,
`browser-identity.cjs`와 `identity-*.png`다. 운영 적용은 추가 검사 없이 빌드·교체한다.
기존 공동 CP6/CP10은 별도 미완료다.
