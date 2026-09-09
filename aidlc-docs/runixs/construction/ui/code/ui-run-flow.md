# Run 흐름과 보기 설정 보완 결과

2026-09-09 `Runixs/UI-Update-2`에서 [보완 계획](../../plans/ui-run-flow-plan.md)을
수행했다. 앞선 Run ID 축약을 보존하고 기존 승인된 UI 구현의 후속 요청을 반영했다.

## 화면

- 상단을 보기 설정 그룹으로 묶었다. 화면(함대/작업 그래프)과 표현(2D/3D)을
  각각 네이티브 드롭다운으로 선택한다. Run 미선택 시 작업 그래프 옵션만
  비활성화하고 목록 선택·새 제출·함대 복귀 시 선택값을 동기화한다.
- Run 그래프에 선택 Run의 제출자 아바타와 Mediator 허브를 추가했다. 각 요소를
  선택하면 역할, Run 상태·접수/종료 시각과 제출자를 상세에서 확인한다.
- 아래 실행 영역에 실제 단계 DAG를 표시한다. 점선은 요청·배정 관계, 실선은
  steps[].needs다. 기존 순번·상태·실행 노드·chosen에 따른 SKIPPED 차이를 유지한다.
- Mediator의 안내는 QUEUED·RUNNING·ASKED·VERIFYING·SUCCEEDED·FAILED를 구분한다.
  실행 중인 CLAIMED 단계의 연결과 허브만 움직인다. 종료 또는 상세 오류·15초
  지연 시 움직임을 멈추고 마지막 관측 상태를 보존한다. reduced-motion을 따른다.
- 단계 없는 QUEUED도 요청자·Mediator·배정 대기를 보인다. 함대 노드나 API의
  steps에 가상 값을 추가하지 않는다. 실제 DAG 오류는 기존 오류·단계 목록으로 남긴다.
- 좁은 화면에서는 흐름을 위에서 아래로 배치하고 폭에 맞춰 표시한다. 일반 휠·
  터치로 세로 이동하고 Ctrl/Cmd+휠 또는 확대 버튼으로 배율을 바꾼다. 넓은 화면과
  함대의 기존 휠 확대 동작을 유지한다.

새 제어의 testid는 `{mode}-scene-select`, `{mode}-representation-select`,
`{mode}-flow-actor-button`이다. 역할 선택 키는 `data-actor-id`이고 실제
`data-node-id`·`data-step-id`·`data-run-id`와 섞지 않는다.

## 검증

| 검사 | 결과 |
|---|---|
| `node --test internal/api/ui/tests/*.test.mjs` | 63 통과, 실패·스킵 0 |
| 흐름 모델 회귀 4개 | 종료/알 수 없는 상태의 잔여 CLAIMED, 신선도, 병렬 ASKED, 빈 단계 구분 |
| Playwright 관측 모형 | demo/fleet × 1440px/390px 4환경, pageerror 0 |
| 실제 demo HTML·JS 진입점 + API fixture | 1440px/390px 2환경 통과, pageerror 0, 쓰기 요청 0 |
| `go test -count=1 -cover ./internal/api/ui` | 통과, 98.4% |
| `go vet ./internal/api/ui` | 통과 |
| `go run ./scripts/glyphscan.go` | 102파일, 위반 없음 |
| `go build -o /tmp/enode-ui-run-flow-mediator ./cmd/mediator` | 통과 |
| 코드·문서 diff 공백 검사 | 통과. audit의 사용자 원문에 있던 후행 공백 1개는 원문 보존 |

Go는 `/tmp/enode-impl-tools/go/bin/go`를 사용했다.
브라우저 검사는 실제 SVG·DOM과 기존 obs-contract fixture를 사용했다. 게스트/
Mediator의 Enter·Space 선택, Escape와 포커스 복귀, 폴링 중 선택 보존,
드롭다운 단일 선택·키보드 문자 선택·새 제출의 run+3d 전환, 실제 선택 ID와
API 조회 URL, 종료·지연·오류·ASKED·QUEUED·잘못된 DAG를 확인했다.
HTML 형태 제출자 문자열은 텍스트로 남고 긴 이름이 카드 밖으로 나가지 않는다.
모바일의 일반 스크롤은 배율을 바꾸지 않으며 문서 가로 넘침이 없다.

실제 데모 진입점에서는 현재 관람객(조용한 강물)과 선택 Run의 제출자(꾸준한 단풍)가
서로 다르게 표시됨을 확인했다. 새 작업 모달과 웹캠 닫기도 회귀 확인했다.
네이티브 드롭다운의 키보드 검사는 Chromium 문자 선택으로 확인했으며 특정 OS의
팝업 화살표 조작까지 검증한 것으로 해석하지 않는다.

로컬 재현 자료는 `/tmp/enode-run-flow-check.khBcul/`의 `check.mjs`,
`full-demo.mjs`, `results.json`, 화면 캡처다. 검사 브라우저는 종료했다.
배포 자산에 fixture나 새 의존성은 추가하지 않았다. 운영 서버 배포 전이며
DB·실물 장면 CP0/CP6/CP10을 이번 UI 검사로 통과 처리하지 않는다.

사용자의 PR 요청에 따라 `main@c51d923`을 인수했다. 코드 충돌 없이 rebase했고
main과 이번 작업의 audit 원문이 모두 보존됐음을 확인했다. Node 63개·Go UI
98.4%·Mediator 빌드·실제 demo 진입점 2환경이 다시 통과했다.
코드 커밋 `8b4d452`, [PR #19](https://github.com/taeels/enode-fixup-project/pull/19).

## 보안 확장

SECURITY-04·05·08·11·12·13·15 준수: 기존 CSP·인증·입력 검증을 보존하고
역할·이름·상태는 SVG textContent와 DOM 속성으로 삽입한다. 새 API·토큰·쓰기
권한을 만들지 않는다. SECURITY-01·03·07은 decisions §3의 예외를 상속한다.
SECURITY-02·06·09·10·14는 인프라·권한·배포·의존·운영 변경이 없어 N/A다.
resiliency-baseline·property-based-testing과 NFR/인프라 SKIP을 유지한다.
