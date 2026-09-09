# Construction 상태 — runixs (김태완)

**현재 공식 단계**: CONSTRUCTION → Code Generation 진행 중.
담당은 **ui**(실 함대·공개 데모)와 **demo-back**(공개 제출·CP9)이다.
담당 FD와 UI 11단계·demo-back 7단계 계획은 2026-09-08T20:11:36Z
사용자의 “응 시작해”로 승인됐다. Inception이나 FD 승인을 다시 기다리지 않는다.

## 현재 기준 — 2026-09-09

`unit/runixs-ui`는 queue PR #5가 병합된 `origin/main@310c22d` 위에 있다.
2026-09-09 재확인한 main도 같은 커밋이다. rebase된 담당 커밋은 `f3a82b9`,
`4c1265d`이며, UI 개선·queue 인수 기록 `ce8bc14`까지 커밋·push했다.
작성자·커미터 이메일은 사용자가 지정한 `runixs92@gmail.com`이다.

**queue는 더 이상 선행 대기가 아니다.** `CreateQueuedRun`·`WakeQueued`가 있고,
실제 API의 201/202/200·QUEUED 재접수·승격·목록 submitter 보존과 UI 갱신을
확인했다. [queue 인수 기록](construction/ui/code/queue-integration-review.md).
이전 설계 리뷰와 obs 인수 문서에 남아 있던 미구현 문구는 정정했다.

**sandbox 출처 승인도 수령했다.** 사용자가 최태양님의 승인을 전달했다.
능력별 `capabilities[].attrs.sandbox`의 광고 원문을 표시하고 없으면 “미제공”이다.
요구 값을 노드 값으로 복사하거나 격리 보장으로 바꾸지 않는다.
실제 장비의 광고·설정 대조는 공동 장면에서 확인한다. 공용 roster는 진행자 소유다.

## 담당 진행

| 범위 | 완료한 것 | 남은 것 |
|---|---|---|
| UI 1~8 | 실 함대 관측, D1/D2/D5, 2D/3D, 투어, 제출 의도·재시도, 웹캠 조작 | 공동 실제 장면은 9~10에서 확인 |
| UI 개선 | 웹캠 닫기·복원, 한글 두 단어 이름, 상세·모달·투어의 바깥 조작 닫기 | 구현·로컬 검증 완료 |
| UI 9 | queue 실제 API·UI 대기→실행, sandbox 표시 출처 승인 | 실제 LED/음원 계약, demo-back POST 연결, 공개 방송 주소 |
| UI 10~11 | 독립 검사·구현 및 검증 문서·커밋·push | 공동 CP 장면과 전체 생성 결과 리뷰·PR 게이트 |
| demo-back 1 | obs·queue 코드와 이름 보존 인수 | 실제 LED/음원 파일·버튼 매핑·이름 주입·Work/ledger 확인 |
| demo-back 2~7 | FD·제출 계약·코드 생성 계획 승인 | 공개 라우트 구현·DB/UI 연결·공동 검증 |

**다음 구현인 demo-back도 runixs 담당이다.** drain·panel·mcp PR 전체를
기다릴 필요가 없다. 큐 접수 경로의 이름 보존은 담당 인수 증거로 확인하며,
진행자의 같은 검사를 또 받아야만 전진하는 승인 단계로 만들지 않는다.

현재 main의 `internal/contract/examples`에는 agent/command/multi만 있다.
진행자를 통해 LED 두 변형의 버튼 연결, mac wav → rpi 재생 계약과 안전한
Guest 이름 주입 위치·Work/ledger 정책을 인수한다. 임의 하드웨어 명령을
만들어 실제 계약으로 대신하지 않는다. 방송 공급자·공개 embed 주소와 실제
송출·장비 운영 담당도 아직 지정된 입력이 없다.

공개 `POST /v1/demo/runs`는 아직 구현되지 않아 404다. 승인된 제출 계약은
scenario_id·submitter·request_id 세 필드와 같은 요청 재시도다.
UI 9와 demo-back 1의 queue 부분 완료가 전체 CP9/10/11 통과를 뜻하지는 않는다.

## 검증 근거

- queue rebase 후 Go 1,108 통과·실패/스킵 0, 16패키지 모두 커버리지 80% 이상.
  UI 98.4%, vet·build·Windows amd64·Linux arm 빌드 통과.
- 후속 UI 개선은 Node 46·Go UI 98.4%·브라우저 24개 동작 검증 통과.
- 전용 로컬 PostgreSQL 16.14를 `scripts/testdb.sh`로 연결했다.
  PostgreSQL 17 CI와 실제 장비·방송을 포함한 공동 장면의 완료 기록은 아니다.
- 이번 코멘트 후 API 전체 140 통과·실패/스킵 0, API vet·glyphscan 통과.
  submitterKey부터 queue·목록·승격까지 회귀 검증과 sandbox 2D/3D 브라우저
  13개 단언 통과를 [queue 인수 기록](construction/ui/code/queue-integration-review.md)에 남겼다.

[구현 요약](construction/ui/code/implementation-summary.md),
[Build and Test](construction/ui/code/build-and-test.md),
[UI 개선](construction/ui/code/ui-feedback.md),
[현재 설계·접점](construction/design-review.md),
[UI 코드 계획](construction/plans/ui-code-generation-plan.md),
[demo-back 코드 계획](construction/plans/demo-back-code-generation-plan.md).

## 소유 경계와 재개 기준

담당 문서는 `aidlc-docs/runixs/`에 기록하고 자기 브랜치에서 작업한다.
공용/run 상태와 `design/*.pen`은 진행자 소유이며, 공유 API 파일은 등록 줄만
수정한다. `internal/api/ui`는 store를 import하지 않는다. 담당 게이트가 초록인
뒤 PR로 main에 직렬 병합한다. 독립 구현의 커밋·push 승인은 전체 유닛 완료나
main 병합 승인이 아니다. 사용자 원문과 승인은 [audit](audit.md)에 보존했다.

Codex는 AGENTS.md에서 vendored AI-DLC v1.0.1 정본을 읽는다. 별도 스킬 목록
노출이나 세션 재시작은 전제조건이 아니다. 설계 gitlink는 `29c89cd`다.
이전 obs 사전 준비·W1 설계·3d-view 검토의 진행 당시 상태는 각 문서와 audit에
남아 있다. 현재 작업 위치는 이 파일과 승인된 코드 계획을 따른다.
팀원의 demo pen D1~D5와 `3d-view@8985694`의 선별 재사용은 이미 UI에 반영했다.

영상·캐릭터 시트 작업은 사용자의 요청으로 **보류**했다. 미추적 영상 산출물과
별도 계획·이미지는 보존하며 UI/demo-back 커밋에 포함하지 않는다.

## Extension Configuration

공용 실행 계획과 requirements/decisions.md §1·§3·§8의 결정을 상속한다.

| Extension | Enabled |
|---|---|
| security-baseline | Yes |
| resiliency-baseline | No |
| property-based-testing | No |

NFR Requirements·NFR Design·Infrastructure Design은 승인대로 SKIP한다.
공용 기준은 construction-roster, 유닛/의존/게이트 매핑, requirements/scene-gates다.
