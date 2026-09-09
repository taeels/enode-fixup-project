# runixs 담당 범위 — ui · demo-back

**상태**: 2026-09-09 KST 담당 범위 구체화. 승인된 유닛 배정을 구현 가능한 작업으로
풀어 쓴 문서다. Functional Design 진행 중이며 설계 승인·코드 생성 완료를 뜻하지 않는다.

**도착점**: 관람객이 팀원 시안의 데모 화면에 Guest로 들어와 보드 상태를 보고,
고정 작업을 제출하고, 3D 그래프와 웹캠으로 결과를 확인한다. 실 함대 읽기 UI와
공개 제출 백엔드까지 runixs가 담당한다.

범위 근거는 [배정표](../../construction-roster.md),
[유닛 정의](../../v1-run-dhseo/inception/application-design/unit-of-work.md),
[기능 팩](../../../requirements/enode-features.md),
[결정표](../../../requirements/decisions.md),
[장면 게이트](../../../requirements/scene-gates.md)다. 공용 배정을 변경하지 않는다.

## 1. 구현할 것과 완료 조건

아래 번호는 담당 내부의 작업 식별자다. 새 유닛이나 새로운 게이트를 추가하지 않는다.

| 작업 | 구현 범위 | 완료 시 확인할 행동·결과 |
|---|---|---|
| UI-01 실 함대 진입 | S0·S0b, 토큰 입력·재입력·세션·인증 종료 | `/ui/`에서 입력, 잘못된 토큰 거절, 성공 뒤 함대 표시, 새 탭에서 재입력 |
| UI-02 관측·상태 표시 | 노드 격자, Run 목록, 요구 능력, ASKED, 임대·draining·만료·갱신 오류 | 노드 수만큼 카드, QUEUED는 목록에만 표시, 요구 능력 확인, 사람 대기와 임대 시간 구분, 미갱신/회복 표시 |
| UI-03 공용 2D/3D 표시 | 함대 모형, steps/needs DAG, chosen, 노드/단계 상세, 보기 전환·탐색 | 같은 관측/선택을 유지하며 전환, 그래프에서도 Run 목록 유지, 긴 그래프 Fit·확대/축소·100%·스크롤 복원 |
| UI-04 데모 진입·투어 | D1·D3~D3d, 기존 Guest/카드뉴스 진입 연결, 이름 표시, 4단계 투어 | 첫 방문 카드뉴스→데모, 재방문 분기 보존, 웹캠→함대→목록→새 작업 순서, 다음·건너뛰기·시작하기·다시 보기 |
| UI-05 데모 제출 화면 | D4 모달, LED/음원 고정 시나리오 선택, 제출 중/실패/접수, 제출자 표시 | 이름 입력 없이 Guest 이름 사용, 요청 중 중복 클릭 차단, 응답의 Run을 D5에서 선택, QUEUED도 접수 결과로 표시 |
| UI-06 웹캠 영역 | D2~D5 좌하단 방송 임베드·resize·zoom/pan/fit·미니맵·상태 | 격자/목록 유지, 플레이어 조작 가능, 송출 상태 구분, Mediator 밖 전송 |
| DEMO-01 데모 제출 라우트 | 시나리오 ID allow-list, 서버측 토큰 주입, 기존 submit 경로 재사용 | 미등록 시나리오·임의 계약 거부, 브라우저에 실제 토큰 없음, 실행/QUEUED 결과를 기존 접수 의미대로 반환 |
| DEMO-02 제출자 전달·저장 | Guest 이름을 기존 submitter 컨텍스트·생성 경로에 전달, 고정 계약의 이름 입력에 주입 | 즉시 실행과 QUEUED 양쪽에서 이름 보존, 목록·RUN 카드·welcome 음원이 같은 이름 사용 |

UI-01~06은 한 `ui` 유닛의 실 함대/데모 두 모드다. DEMO-01~02는 `demo-back`이다.
`demo-back`은 기존
CreateRun/CreateQueuedRun 접수 흐름을 사용하고 스케줄러를 새로 만들지 않는다.

## 2. 화면과 기존 코드의 사용 기준

- **최종 데모 화면 기준**: `design/enode-demo.pen`의 D1~D5. D2는 함대 3D(S6),
  D5는 작업 그래프 3D(S7)다. 카드뉴스 콘텐츠는 nacl1119의 기존 구현을 연결한다.
- **실 함대 기준**: `design/enode-ux.pen`의 S0·S0b·S1·S1b·S2와 확정된 표시 규칙.
- **재사용 구현**: `3d-view` 워크트리의 `unit/ux-runtime@8985694`에서 SVG 모형,
  graphLayout, 확대/맞춤/위치 보존을 선별한다. 근거는
  [재사용 검토](ui/functional-design/3d-view-reuse-review.md)다.
- **두 모드의 공유 부분**: 노드·Run·단계 의미 모델, 상태 표시, DAG, 2D/3D 선택·탐색.
  실 함대 인증과 공개 데모 읽기/제출 경계는 모드별로 분리한다.
- **원본과 확정 요구의 차이**: Guest 이름은 랜덤 2단어, submitter는 그 이름,
  단계 RUNNING 예시는 CLAIMED로 표시한다. sandbox는 합의한 출처의 명시적 값만
  읽고, 미제공을 격리 보장으로 채우지 않는다. 페이지네이션은 추가하지 않는다.

기존 `guest.guestName()`·`hasOnboarded()`·`markOnboarded()`의 역할을 보존한다.
가이드투어 완료 상태는 카드뉴스 완료 상태와 분리한다. 기존
`landing-guest-login-button`과 `demo-replay-cardnews-link`는 유지한다.
현재 demo 화면은 placeholder이므로 완성된 데모로 취급하지 않는다.

## 3. 손댈 파일과 공유 접점

| 경로 (저장소 루트 기준) | 작업 원칙 |
|---|---|
| `internal/api/ui/static/fleet/` | 실 함대 진입·표시·조회 조정. 기존 코드 계획의 model/scene/view를 활용 |
| `internal/api/ui/static/demo/index.html`, `demo.js`, `demo.css` 및 신규 demo 모듈 | placeholder를 실제 D2~D5 화면과 투어·모달·웹캠으로 발전 |
| `internal/api/ui/tests/`, `testdata/`, 신규 `fleet_test.go` | 표시/상태/조작·embed·계약 검증. 합성 자료는 공개 static 밖 |
| `internal/api/ui/static/index.html`, `landing.css` | 관리자 S0와 D1 접점. nacl1119의 최신 변경과 대조한 뒤 최소 수정 |
| `internal/api/ui/static/shared/guest.js`, `landing.js`, `cardnews/` | 현재 인터페이스를 우선 소비. 변경이 필요하면 nacl1119와 접점을 확인 |
| `internal/api/demo.go`, `demo_test.go` | allow-list·서버측 토큰·접수/오류·submitter 연결 |
| `internal/api/api.go` | 데모 라우트 등록 줄만. 기존 `/ui/` 마운트 재사용, 공용 접점 조율 |
| `internal/store` | obs의 기존 submitter 저장 재사용. queue 생성 경로가 이름을 보존하는지 대조하고 필요한 접점만 변경 |
| `internal/config`·데모 설정 | obs의 Config.Demo 재사용. 방송 주소는 UI의 정적 demo/settings.json에 두며 config.go는 추가 수정하지 않음 |
| `aidlc-docs/runixs/` | 담당 범위·FD·코드 계획·검증·상태·감사 기록 |

공용 표시 모듈을 fleet 아래에서 demo가 가져올지 별도 공용 경로로 옮길지는
전체 UI FD에서 파일별로 확정한다. 같은 로직을 두 모드에 복사하지 않는다.
기존 Go embed와 Handler를 활용하고 `ui → store` 임포트 금지를 유지한다.
`3d-view`의 전체 ui.go·앱 진입점·전역 CSS·서버 구현은 통째로 이식하지 않는다.

## 4. 다른 담당에게서 받는 것

| 상대 | 받을 것 | 우리가 하는 일 |
|---|---|---|
| taeels / obs | nodes/runs/detail, requires·chosen, submitter 저장/조회, 공개 읽기 설정 — 0a159a4에 병합됨 | 인수 완료. UI 어댑터와 제출 시 이름 전달 구현 |
| shin-son / queue | 202/QUEUED 접수·승격·취소와 기존 생성 경로 | 그 결과를 화면/데모 접수에 연결, 이름 보존 검증 |
| shin-son / drain | draining 모드와 자원 회수 동작 | 카드의 진행 중/대기 중을 실제 관측과 대조 |
| nacl1119 | 카드뉴스·공유 Guest 흐름, panel과 transcript의 표시 접점 | 기존 진입을 보존하고 연결, 같은 노드 사실을 표시하는지 협력 검증 |
| 진행자·디자인 담당 | pen 원본, 고정 시나리오 픽스처·ID, sandbox 출처 합의 | 시안을 코드로 구현하고 고정 계약을 제출 경로에 연결 |
| 시연 운영·하드웨어 담당(확인 필요) | 데모 인스턴스, 방송 URL/허용 origin, 실제 rpi/mac 실행 환경 | 임베드/설정 연결, CP10에서 화면과 실제 결과를 함께 확인 |

우리가 소유하지 않는 구현은 obs/MCP, 큐 승격 알고리즘, drain 엔진, 호스트
프로세스 제어, transcript 수집, 카드뉴스 콘텐츠 제작이다. 방송 송출 서비스·
실제 LED/음원 실행 환경의 운영 담당도 이 문서만으로 runixs에게 배정하지 않는다.
다만 우리의 화면/제출 기능이 그 결과에 연결되는지는 공동 장면에서 확인한다.

## 5. FD에서 닫을 접점

이 항목들은 새 요구가 아니라 기존 배정을 구현하기 위해 남아 있는 연결 정보다.

| 접점 | 준비된 방향 | 닫는 방법 |
|---|---|---|
| obs D1~D5 | 병합된 코드·관측 테스트로 소비 형식 확정 | UI 어댑터·실제 함대/queue 장면 검증은 후속. [인수 기록](ui/preparation/obs-integration-review.md) |
| 공개 데모 읽기 | Config.Demo에서 nodes·runs·detail 무인증 | 기존 스위치 소비, asks는 공개 조회하지 않음. obs의 진행자 인계 항목은 유지 |
| 데모 제출 계약 | 요청은 고정 시나리오 선택+Guest 이름, 응답은 접수된 Run 식별·상태 | ui/demo-back FD에서 경로·정확한 JSON·오류를 함께 정의하고 기존 submit 결과와 대조 |
| 고정 시나리오 | c7a237d의 LED·welcome 음원 인수·연결 완료 | LED 한 Run heartbeat→persistent, 음원 synthesize.in.prompt에 이름 주입. 실제 장비는 공동 검증 |
| sandbox | capabilities[].attrs.sandbox의 능력별 광고값, 없으면 미제공 | 2026-09-09 사용자가 최태양님 승인 전달. 실제 장비 대조는 공동 장면에 남김 |
| 방송 임베드 | 좌하단 플로팅, Mediator 밖 전송 | 공급자·주소·허용 origin/iframe 정책을 운영 담당과 확인 |
| 공유 파일 | 기존 Guest 인터페이스·static embed 재사용 | nacl1119 및 진행자와 실제 파일 diff·등록/설정 접점 대조 |

우리의 단위 구현 선택은 담당 FD에서 정한다. 팀 공용 API·시안 원본·의존성·게이트를
바꾸는 선택은 진행자 조율 대상으로 남긴다. 검토 자료는 우리가 작성하며 이 문서를
다른 사람에게 전송하거나 합의가 끝났다고 기록하지 않는다.

## 6. 지금부터의 진행 순서

1. **이번 작업 — 범위 구체화**: 위 작업·파일·담당 접점·완료 조건을 담당 문서에 연결한다.
2. **현재 FD 보완**: D1~D5, 모달/투어/웹캠의 상태·조작, 공용 표시 모델을 UI 네 FD
   문서로 정리한다. demo-back의 제출 계약·허용 목록·이름 저장도 별도 FD로 구체화한다.
3. **구현 전 검토**: 확정 가능한 담당 설계와 남은 공용 접점을 구분해 리뷰한다.
   ui/demo-back 설계와 각각 11단계·7단계 코드 계획을 함께 검토한다.
   기존 W1 8단계 부분 초안은 전체 UI 계획으로 대체했다.
4. **독립 표시 구현**: obs가 병합되어 전달 대기는 끝났다. 설계·계획에 맞춰 SVG/
   화면 상태·모달/투어·웹캠 영역을 로컬 입력으로 검증하고 실제 조회와 연결한다.
   이번 범위 문서 작성이나 rebase 요청을 미완성 설계의 승인으로 기록하지 않는다.
5. **W1 연결**: 실제 응답으로 실 함대 UI를 연결·검증한다. queue/공동 CP7
   증거가 필요한 항목은 담당별 결과와 함께 확인한다.
6. **obs·queue 이후 demo-back, 계약에 맞춘 데모 UI**: 서버 접수·submitter를 구현하고
   데모 화면을 연결한다. 두 쪽은 제출 계약을 정하면 작업할 수 있고 CP9 실연동은
   demo-back 병합 뒤 검증한다.
7. **시연 검증·병합**: CP8·9·11과 하드웨어 포함 CP10, 필요한 회귀 검사를 완료한다.
   외부 선행이 준비되지 않은 검사는 보류 이유/재실행 조건을 적으며 통과로 표시하지 않는다.

## 7. 완료 증거의 책임

| 확인 | runixs가 남길 증거 | 공동 증거 |
|---|---|---|
| CP1·CP2 | S0/S1/S1b/S2 화면·모든 조작, QUEUED/chosen 표시 | obs의 API·queue의 실제 승격 |
| CP3·CP4 협력 | draining 진행 중/대기 중 배지와 관측 일치 | drain/panel의 실제 제어 |
| CP7 표시 | ASKED·answerers·기한/경과·CLI 안내·답변 후 갱신 | mcp/CLI 답변 도구와 Run 재개 |
| CP8 | Guest→카드뉴스 연결·재방문·가이드투어·3D 함대 | 카드뉴스 담당의 기존 산출물 |
| CP9 | 모달, allow-list 거부, submitter 저장/표시, D5 전환·Run 목록 유지 | obs 조회·queue 접수·진행자 시나리오 |
| CP11 | 방송 임베드·resize·레이아웃 유지·Mediator 밖 전송 | 실제 방송 연결 |
| CP10 | 두 시나리오의 제출→화면/그래프/웹캠까지 일치 | 실제 LED·welcome 음원·봉인 결과와 하드웨어 |

Go UI 커버리지 하한 80%와 필요한 CP0 검사는 유지한다. JS 상태·화면 조작은 별도
Node/브라우저 검증으로 확인한다. 기존 코드의 테스트 결과나 합성 화면으로 신규
기능의 장면 게이트를 완료하지 않는다. 데모 전체 완료는 CP10까지 필요하다.
