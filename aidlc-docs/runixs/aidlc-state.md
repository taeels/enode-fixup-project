# Construction 상태 — runixs (김태완)

**현재 공식 단계**: CONSTRUCTION → Code Generation 진행 중.
담당은 **ui**(실 함대·공개 데모)와 **demo-back**(공개 제출·CP9)이다.
담당 FD와 UI 11단계·demo-back 7단계 계획은 2026-09-08T20:11:36Z
사용자의 “응 시작해”로 승인됐다. Inception이나 FD 승인을 다시 기다리지 않는다.

## 현재 기준 — 2026-09-09

`unit/runixs-ui`는 queue PR #5가 병합된 `origin/main@310c22d` 위에 있다.
2026-09-09 재확인한 main도 같은 커밋이다. rebase된 담당 커밋은 `f3a82b9`,
`4c1265d`이며, UI 개선·queue 인수 기록 `ce8bc14`까지 커밋·push했다.
이후 queue/sandbox 접점 정정은 `9ae9c4f`로 push했다. 사용자의 지시로 PR #6에
추가된 진행자 `c7a237d`를 fast-forward하고 실제 LED·음원 계약을 인수했다.
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
| UI 9 | queue·sandbox 출처, c7a237d 두 시나리오의 실제 POST/GET·이름·3D 연결 | 실제 장비 광고·동작, 공개 방송 주소 |
| UI 10~11 | 독립 검사·구현 및 검증 문서·커밋·push | 공동 CP 장면과 전체 생성 결과 리뷰·PR 게이트 |
| demo-back 1~5 | obs·queue·실제 fixture 인수, 공개 라우트·검증·한도·재시도·DB/UI 연결 | 구현·로컬 검증 완료 |
| demo-back 6~7 | 표준 검사·race·브라우저·문서 | 공동 장면과 전체 유닛 리뷰 |

**다음 구현인 demo-back도 runixs 담당이다.** drain·panel·mcp PR 전체를
기다릴 필요가 없다. 큐 접수 경로의 이름 보존은 담당 인수 증거로 확인하며,
진행자의 같은 검사를 또 받아야만 전진하는 승인 단계로 만들지 않는다.

실제 시나리오는 진행자 c7a237d에서 인수했다. LED는 heartbeat→persistent 한 Run,
음원은 synthesize.in.prompt의 지정 자리표시자에만 이름을 넣고 play의 needs/blob
참조를 유지한다. 요청마다 독립 Contract·manual Work를 준비한다.

공개 `POST /v1/demo/runs` 구현과 실제 두 매핑을 연결했다. 세 필드만 받고 서버
토큰/submitterKey로 기존 submit을 호출한다. 201/202/200·같은 의도·이름 보존과
로컬 브라우저의 3D 전환·queue 승격·목록 갱신을 확인했다.

**다음은 공동 실제 장면이다.** 문태호(nacl1119)의 enode-demo-led/enode-demo-play
래퍼와 Windows→rpi, 손신(shin-son)의 mac Claude+higgsfield 광고/합성, 최태양
(taeels)의 공개 webcam embed URL이 필요하다. 준비된 장비와 함께 runixs가
UI·제출·이름·음성·봉인·방송을 확인한다. 현재 webcam 설정은 null이다.
진행자 [인수 안내](../taeels/construction/demo-fixtures/README.md)의 담당 범위를 따른다.

[demo-back 구현](construction/demo-back/code/implementation-summary.md)과
[검증 기록](construction/demo-back/code/build-and-test.md)에 소프트웨어 완료와
실제 하드웨어/방송 게이트를 구분했다. [PR #6](https://github.com/taeels/enode-fixup-project/pull/6)은
Draft로 유지하며 전체 CP9/10/11이나 main 병합 완료를 선언하지 않는다.

## 검증 근거

- 최종 demo-back+실제 fixture 기준 Go 1,173 통과·0 실패·0 스킵, 16패키지 80% 이상.
  api 84.539%, demo.go 96.0%, UI 98.4%. race·vet·build·교차 빌드·govulncheck 통과.
- 최종 안내 수정 뒤 Node 46·Go UI·브라우저 14개 단언 통과. 아래는 선행 단계의 검사다.

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
