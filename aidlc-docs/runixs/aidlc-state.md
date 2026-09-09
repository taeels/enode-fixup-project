# Construction 상태 — runixs (김태완)

담당 유닛 — **ui**(W1 실 모드 · W3 데모 모드) · **demo-back**(W2 · CP9). 자기
브랜치에서 작업하고 PR 로 main 에 병합한다(게이트 초록 뒤). 산출물은 이 디렉터리
`aidlc-docs/runixs/` 아래.

**2026-09-09 최신 기준**: queue PR #5의 `origin/main@310c22d`에
`unit/runixs-ui`를 충돌 없이 rebase했다. 담당 두 커밋은 `f3a82b9`, `4c1265d`다.
[queue 인수 기록](construction/ui/code/queue-integration-review.md)에 실제
202/QUEUED·재접수·승격·목록 submitter와 UI 갱신 증거를 남겼다.

**사용자 개선 반영**: 웹캠 닫기/재열기, 한글 두 단어 Guest 이름, 플로팅 상세와
모달/투어의 바깥 조작 닫기를 구현했다. [변경 계획](construction/plans/ui-feedback-plan.md),
[구현·검증](construction/ui/code/ui-feedback.md). Node 46·Go UI 98.4%·브라우저
24개 동작 검증 통과. 영상은 보류하고 산출물과 재개 지점을 보존했다.

**최신 커밋·push 승인**: 2026-09-09T01:25:46.761181+00:00 사용자가 “그러면 일단 commit해서 push할까”로
UI 개선과 queue 인수 기록의 저장·공유를 승인했다. `unit/runixs-ui`를
`runixs92@gmail.com`으로 커밋하고 rebase된 브랜치를 원격에 반영한다.
실제 데모 연결·공동 게이트와 main 병합은 별도로 남아 있다.

## 유닛

**현재 구현**: 승인된 UI 코드 계획 1~8을 생성했다. 실 함대 관측·D1/D2/D5·투어·
제출 의도 상태·방송 표시 조작을 구현했고 로컬 브라우저와 단위 검증을 진행했다.
9단계 실제 데모 연결은 LED/음원 픽스처·demo-back·방송 주소가 없어 보류다.
queue 인수는 완료했고 UI의 실제 대기→실행 전환은 로컬 API로 확인했다.
10단계 중 외부 입력이 필요 없는 검사를 진행했다. queue rebase 뒤 Go 1,108 통과,
후속 UI 개선 뒤 Node 46과 Go UI 검사가 통과했다.
Go 0 스킵, 16 패키지 모두 커버리지 80% 이상(UI 98.4%). Code Generation 전체
완료나 공동 장면 통과를 선언하지 않으며, 아래 이전 준비 기록은 이 진행에 선행한다.

**커밋 승인**: 2026-09-08T23:28:22Z 사용자가 현재 구현의 커밋과
`runixs92@gmail.com` 작성자 이메일을 지정했다. 독립 UI 코드와 담당 기록을
`unit/runixs-ui`에 커밋한다. 실제 연결·공동 게이트의 보류 상태는 유지한다.

- [ ] ui 현황판 UI — 한 유닛 두 모드
  - **실 함대 모드**(CP1·CP2 화면) — obs 만 딛는다 -> **W1 착수·완료**.
    S0·S0b·S1·S1b·S2 · 되묻기 카드 표시(CP7) · 읽기 전용
  - **데모 모드**(CP8·CP9·CP11 화면) — demo-back 제출 라우트를 딛는다 -> **W3 완료**.
    게스트 로그인 · 3D(S6·S7) · 새 작업 모달 · 웹캠 floating · 투어 · RUN 카드 submitter
  - **완료 조건에 화면 조작 전부 나열**(§2.1·§5.3). internal/api/ui 는
    runixs(ui)·nacl1119(card-news) 공유(최신 roster). store 임포트 금지 · 커버리지 80%
- [ ] demo-back — CP9 서버측 · 의존 obs · queue
  - internal/api/demo.go — allow-list · 서버측 토큰 주입 · submitter 쓰기(Guest 로그인 이름)
  - api.go 는 등록 줄만 · 데모 라우트는 데모 모드 config 에서만 등록 권장

## ui 와 demo-back 의 순서

제출 라우트 계약(경로·payload)을 먼저 정하면 **병렬로 짠다**. demo-back(W2)이
병합된 뒤 ui 데모 모드의 CP9 end-to-end 를 검증한다(W3).

## 열린 미정

- ui — sandbox 출처 결정안: 명시적인 capabilities[].attrs.sandbox 를 능력별
  노드 광고 값으로 표시, 없으면 미제공. roster §6의 진행자 협의·FD 리뷰와
  실제 광고 확인은 남아 있다. 담당 설계안만으로 공용 미정을 닫지 않는다.
- obs D1~D5의 소비 형식은 코드·기존 관측 테스트로 확정했다. 실제 UI 소비와
  queue·drain 장면 대조는 후속 검증이다.
- 데모 — 제출 계약 검토와 고정 시나리오·방송 설정 수령이 남았다.
  공개 읽기는 obs의 Config.Demo를 사용하며 asks는 공개 조회하지 않는다.

**FD 보완 결과**: 제출 계약은 POST /v1/demo/runs의 세 필드·같은 요청 재시도까지
설계했다. 실제 픽스처·방송 공급자/주소와 sandbox 진행자 확인은 미수령이다.
데모 화면·모달·투어·웹캠 전체 조작과 별도 demo-back FD가 준비됐다.

## 담당 작업과 진행자 결정의 경계

roster §2·§5에 따라 runixs는 ui(실 함대·데모)와 demo-back의 상세 설계·구현·
테스트·담당 기록을 진행한다. 문서/코드는 에이전트가 작성하고 담당자는 그 범위의
작업 방향을 검토한다. 승인된 팀 전체 Inception을 다시 수행하는 작업은 아니다.

공용 요구·유닛 경계·의존성 변경, 공유 접점 병합, 회차/공용 상태와 pen 원본은
진행자 조율 범위다. sandbox 출처처럼 roster가 명시한 미정은 진행자와 닫는다.
runixs의 유닛 계획 검토가 공용 결정이나 다른 담당의 API 계약 승인을 대신하지 않는다.
사용자가 설명한 팀 리드는 최태양님이며, 별도 구현 배정은 taeels의 obs·mcp다.

## 선행 · 공용

- 유닛 정본 `aidlc-docs/v1-run-dhseo/inception/application-design/unit-of-work.md`
- 배정 `aidlc-docs/construction-roster.md` · 팩 `requirements/` · RE `aidlc-docs/inception/reverse-engineering/`
- 데모 시안 참고 — `design/enode-demo.pen` · export `design/exports/D1~D5`
- 이미 있는 정적 코드 — `internal/api/ui/`(cardnews 회차가 낸 landing·guest·demo·cardnews)

## 다음

**3d-view 검토 반영**: 지정 워크트리의 실제 브랜치 `unit/ux-runtime@8985694`를
읽고 테스트·실행 화면을 확인했다. SVG 3D·DAG 배치·확대/스크롤 보존의 선별
재사용을 권고하며 코드 계획 1~3에 반영했다. 상세는
`construction/ui/functional-design/3d-view-reuse-review.md`. Run 그래프에서 목록이
사라지는 점과 인증·폴링·계약 차이는 이식 시 수정한다. 원본 워크트리·제품 코드는
수정하지 않았으며 브랜치 병합·코드 생성은 아직 없다.

**디자인 범위 확인**: 사용자가 팀원의 demo pen 반영 여부를 물었다. 현재 W1
설계의 직접 타깃은 `enode-ux.pen`이며 최종 데모 타깃 `enode-demo.pen` D1~D5는
8단계 구현 대상에 포함되지 않았음을 명확히 했다. 화면별 대조는
`construction/ui/functional-design/design-targets.md`에 있다. 데모 디자인 준비는
obs 이전에도 가능하다. 이 질문은 기존 계획 승인이나 W1 취소로 기록하지 않는다.

**상세 설계와 전체 구현 순서를 검토한다.**
[현재 리뷰](construction/design-review.md)에 실/데모 UI FD와 demo-back FD,
UI 11단계·demo-back 7단계 구현 계획을 연결했다. 공통 모델·SVG·클라이언트를
shared/fleet에 두며, 두 모드는 인증/상태를 분리한다. 모달 두 버튼·네 단계 투어·
방송 resize/zoom/pan·미확인 제출의 같은 요청 재시도까지 설계했다.
obs 전달 대기는 해제됐고, 리뷰 뒤 준비된 UI 코드부터 생성할 수 있다.
queue·픽스처·방송·sandbox 공용 확인이 필요한 단계는 입력 수령 뒤 진행한다.

## Codex 재개 설정 — 2026-09-08

- [x] Codex 진입점 `AGENTS.md` 를 AI-DLC v1.0.1 정본에 연결.
- [x] 이 체크아웃의 담당을 `local/aidlc-context.md` 에 runixs 로 설정.
- [x] 담당·유닛·승인된 실행 계획과 선행 구현 상태 확인.
- [x] detached HEAD `c010841` 에서 작업 브랜치 `unit/runixs-ui` 생성.
- [x] `enode-design` 서브모듈을 고정 리비전 `29c89cd` 로 초기화.
- [x] 설정 문서의 참조 경로·버전·표기와 `git diff --check` 검증.

**현재 공식 단계**: CONSTRUCTION → ui Code Generation 진행 중.
2026-09-08T20:11:36Z “응 시작해”로 담당 FD·구현 계획 승인을 기록했다.
승인된 UI 계획 1~8을 구현했다. 공용 입력 확인은 9단계 실제 연결의 조건으로 유지한다.
W1 상세 설계 검토안, 팀원 demo pen 대조, 3d-view 재사용 검토, 담당 범위와
obs 소비 계약 인수를 완료했다. 데모 전체 UI와 demo-back의 상세 설계 검토안을
작성했고 공용 입력 확인·승인은 남아 있다.
검토를 위해 작성한 코드 계획은 이제 승인됐다. 코드 생성은 실제 연결을 남겼으며
독립 UI 검증을 완료했다. [구현 요약](construction/ui/code/implementation-summary.md)과
[검증 결과](construction/ui/code/build-and-test.md)에 이번 신규 코드 검증과
공동 장면 보류를 구분했다. 승인된 Inception은 유지한다.

**확인 기준**: 최신 origin/main을 fetch하고 queue 병합 `310c22d`로 rebase했다.
`GET /v1/nodes`·`GET /v1/runs`와 requires/chosen 확장이 존재한다.
`WakeQueued`·`CreateQueuedRun`과 submitter 저장·승격을 인수했다. obs 담당의 부분 CP1 측정과
진행자 병합 조건을 확인했다. 실제 UI·queue가 필요한 CP1/CP2 전체 통과는 아니다.

**이미 있는 코드**: `internal/api/ui/static/` 의 landing · shared/guest · demo ·
cardnews 와 `ui.Handler()`. `/ui/` 마운트는 이미 `internal/api/api.go` 에 있다.
설계 문서에 있는 `cmd/mediator/main.go` 마운트를 중복으로 만들지 않도록 FD 에서
실제 코드와 대조한다.

**통합 착수 조건**: obs 인수는 완료했다. UI 어댑터·브라우저·실제 장면은 설계와
코드 계획에 따라 구현·검증한다. demo-back의 queue 접수 경로는 인수했고,
공동 제출 계약 검토안을 사용하며 고정 시나리오의 실제 연결은 인수 뒤 확정한다.

**2026-09-09 FD 진행**: UI 네 문서와 demo-back 네 문서(공동 제출 계약 포함)를
작성·보완했다. 기존 W1 코드 계획은 전체 UI 11단계로 대체했고 demo-back 7단계를
별도로 작성했다. 현재 검토 질문은 construction/design-review.md §4다.
이전 W1 Q1을 중복 승인 항목으로 사용하지 않는다.

**인수 검증**: 전용 PostgreSQL 16에서 API 17개·store 21개 관측 테스트 통과,
실패·스킵 0. UI 기존 테스트 92.3%, 전체 Go 빌드 통과. 로컬 도구 디렉터리가
Go 패키지 탐색에 포함되던 문제는 ignored `local/toolchains/go.mod`로 분리했다.
PostgreSQL 17의 전체 CP0와 실제 함대·UI 게이트를 완료한 기록은 아니다.

## obs 전달 전 사전 준비 — 2026-09-08

사용자가 사전 준비 제안에 “일단 너가 말한대로 진행해줘”라고 답해 범위를 승인했다.
계획은 `construction/plans/ui-preparation-plan.md`, 결과 안내는
`construction/ui/preparation/README.md` 다. 공용 roster·실제 게이트 조건은 유지한다.

- [x] UI 화면·상태·데이터 흐름 초안 작성.
- [x] 소비 API 계약표와 접점 D1~D7 작성.
- [x] 합성 응답 17개와 manifest 1개 작성 — `internal/api/ui/testdata/obs-contract/`.
- [x] obs 반영 후 실행할 통합 검증 항목 작성.
- [x] JSON·시나리오 의미·문서 표기 검증 통과.
- [x] 기존 UI 패키지 테스트·커버리지 — 로컬 Go 설치 후 통과, 92.3% statements.
- [ ] 실제 obs·queue 연동과 CP 게이트 — 미실행.

**인수 시 우선 확인**: ADR-069 의 중첩 attrs 예시와 기존 Require.MarshalJSON 의
평탄 속성 표현 차이(D1), lease 부재(D2), 목록 verdict(D3), submitter(D4),
requires/chosen 확장(D5). sandbox 출처(D6)와 두 UI 모드의 진입·읽기 경계(D7)는
해당 FD 에서 닫는다. 미확정을 샘플로 확정하거나 API 장애를 샘플로 숨기지 않는다.

**Codex**: 별도 aidlc 스킬 설치가 아니라 AGENTS.md 를 통한 v1 규칙 적용이다.
이번 세션은 참조 규칙을 직접 읽어 진행했다. 새 세션은 AGENTS.md 와 이 상태를
읽으면 되며 스킬 목록 노출은 재개의 전제조건이 아니다.

## 상세 설계와 도구 준비 — 2026-09-08

- [x] W1의 business-logic-model·business-rules·domain-entities·frontend-components 작성.
- [x] sandbox 출처, verdict.checks[].note, VERIFYING/CLAIMED, ASKED의 Run 연결 정리.
- [x] `/ui/` S0 관리자 폼 → 인증 성공 → `/ui/fleet/` 진입으로 결정안 작성.
- [x] 공유 랜딩 접점과 신규 fleet 자산을 나눈 8단계 코드 생성 계획 작성.
- [x] 공식 go.dev 배포 메타데이터의 크기·SHA256 확인 후 Go 1.26.6 darwin/arm64를
  `local/toolchains/go1.26.6/go/`에 설치. 전역 PATH·go.mod 변경 없음.
- [x] `GOTOOLCHAIN=local ./local/toolchains/go1.26.6/go/bin/go test -count=1 -cover ./internal/api/ui`
  통과: 92.3% statements. 기존 정적 UI 기준선이며 JS 기능·CP 게이트 증거는 아니다.
- [ ] 설계/코드 계획 승인 — 설계 계획 Q1에 대기 상태 기록.
- [ ] obs 실제 응답·공동 장면 게이트 — 여전히 미실행.

제품 런타임 파일은 아직 수정하지 않았다. 새 세션에서도 로컬 Go 실행 경로를
명시하면 이어서 테스트할 수 있다.

## Extension Configuration

`aidlc-docs/v1-run-dhseo/aidlc-state.md` 와 `requirements/decisions.md` §1·§3·§8
의 승인된 선택을 상속한다. 재개를 이유로 다시 묻지 않는다.

| Extension | Enabled |
|---|---|
| security-baseline | Yes |
| resiliency-baseline | No |
| property-based-testing | No |

승인된 실행 계획대로 NFR Requirements · NFR Design · Infrastructure Design 은
SKIP 한다. security-baseline 은 해당 단계에서 결정표의 적용 범위대로 집행한다.
