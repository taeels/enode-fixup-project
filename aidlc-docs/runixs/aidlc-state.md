# Construction 상태 — runixs (김태완)

**현재 공식 단계**: CONSTRUCTION → DDTHON 내부 댓글 탐색 보완 Build and Test 진행.
앞선 갤러리 확장은 검증·Mac mini 운영 적용·PR #14 병합을 완료했다.
기존 공동 CP6/CP10은 별도 미완료 상태를 유지한다.
담당은 **ui**(실 함대·공개 데모)와 **demo-back**(공개 제출·CP9)이다.
담당 FD와 UI 11단계·demo-back 7단계 계획은 2026-09-08T20:11:36Z
사용자의 “응 시작해”로 승인됐다. Inception이나 FD 승인을 다시 기다리지 않는다.

## 현재 기준 — 2026-09-09

**UI 가독성 후속 작업**: `UI-update` 체크아웃의 `unit/runixs-ui-readability`
(`a5d62cf` 기준)에서 휠 확대·축소, 작업 그래프 구분·함대 복귀, 현재 임대의
제출자 표시, 기능별 정보 카드·읽기 쉬운 시각을 구현했다. 긴 노드 이름의 겹침
피드백으로 함대 간격과 두 줄 표시도 보완했다. Node 52·브라우저 45개 단언·12가지 배치·Go UI 98.4%·vet·glyphscan·Mediator 빌드를 확인했다.
[변경·검증 기록](construction/ui/code/ui-readability.md),
[보완 계획](construction/plans/ui-readability-plan.md). 공식 Code Generation과
기존 공동 장면 보류는 유지한다. 사용자 지시로 미리보기를 재개하고 코드 510c6a6·0b422a7을 push했다.
[PR #16](https://github.com/taeels/enode-fixup-project/pull/16)은 main에 병합됐고 이번 rebase로 인수했다. 운영 배포 전이다.

**로컬 실행 상태: 중지.** 사용자 지시로 실행 장소를 다른 호스트로 옮긴다.
이 PC의 Mediator·enode·이전 3d-view 데모 제어판과 전용 테스트 DB를 종료했다.
아래 LAN·Drain 결과는 종료 전 검증 기록이다. 설정·DB 데이터·로그는 보존했다.

**Mac mini 공개 데모는 실행 중이다.** 별도 DB·새 토큰으로 Mediator와 실제 노드를
띄우고 Cloudflare Quick Tunnel의 HTTPS 주소로 공개했다. 외부 UI·토큰 인증과
실제 노드의 공개 등록·claim·shell 실행·봉인 다운로드를 확인했다. 사용자에게
소유 도메인이 없어 현재 임시 주소를 유지한다. [실행·관리 기록](construction/macmini-public-demo.md).
이 MacBook의 서비스는 중지 상태이며 공동 LED·음원·방송 CP10은 남아 있다.

**Mac mini 작업 실행은 Bedrock VM으로 전환했다.** 기존 `ccb` 연결을 별도 Lima
VM의 일반 사용자 enode에 적용했다. 사용자가 제공한 새 키로 실제 Claude Run
`bedrock-vm-proof-b8f111c2` 성공·봉인 결과와 격리 검사 15개를 확인했다.
기존 Mac 직접 실행 노드는 종료했고 VM과 팀원 두 노드의 heartbeat, 공개 UI를
재확인했다. Mediator·터널·공개 주소는 유지한다. [구성·관리·한계](construction/macmini-bedrock-sandbox.md).
제품 OS sandbox 기능이나 공동 CP10 전체 완료로 해석하지 않는다.

후속 요청으로 지정된 해커톤 갤러리 호스트의 HTTPS를 VM에 허용했다. 사용자의
명시적인 한 댓글 지시에 따라 Run Away 게시글에 댓글을 등록하고, 로그인하지
않은 게스트 화면에서도 노출을 확인했다. [갤러리 검증](construction/hackathon-gallery-integration.md).
이후 사용자 승인으로 전체 참가팀 대상 Claude MCP·게스트 프롬프트 UI를 구현하고
최신 main `a5d62cf`를 통합해 Mac mini에 적용했다. 후속 지시로 팀 선택·예시/제약
목록·별도 게시 확인을 제거했다. 이제 메시지 한 번으로 AI가 팀 목록→해당 글→댓글
게시까지 같은 Run에서 처리한다. 입력칸에는 Mac mini·DDTHON·Bedrock 안내와
요청 예시 placeholder만 둔다. 모호한 팀은 되묻고 초안 전용 요청은 게시하지 않는다.

공개 UI의 “Run Away 팀에 응원 댓글 달아줘”에서 실제 새 댓글
`398167b8-ffb8-4c22-bb88-b6b55bcec63a`를 확인했다. 실행 중 새로고침에도 같은
Run과 댓글 한 개로 완료됐으며 외부 URL 요청은 도구 호출 없이 거절됐다.
[구현·배치](construction/demo-back/code/gallery-implementation.md),
[Build and Test 인수](construction/build-and-test/build-and-test-summary.md).

태양님 Windows 노드 `270c97c94415`는 실제 shell Run
`teammate-connectivity-54c76943` 성공·봉인 내용·lease 해제를 확인했다.
추가 Mac 노드 `620abcbb7e47`도 등록·heartbeat 갱신을 확인했으며 해당 노드의
에이전트·하드웨어 작업은 아직 검증하지 않았다.

공개 서버의 등록 실패를 조사해 HTTP→HTTPS 301이 등록 POST를 GET으로 바꾸는
문제를 재현했다. 제공된 토큰은 HTTPS에서 인증됐다. 노드 설정 주소를 HTTPS로
바꿔 재시작하도록 안내했다. [공개 등록 확인 기록](construction/ui/code/public-registration-review.md).
실패한 다른 PC의 재시작 후 등록 성공은 아직 미확인이다.

[PR #14](https://github.com/taeels/enode-fixup-project/pull/14)는 모든 CI 통과 후 main `50899b0`에 병합됐다. 후속 `unit/runixs-gallery-navigation`에서 DDTHON 내부 댓글/작성 팀 탐색의 과잉 거절을 수정한다. 운영 Mediator는 `b01274b`, VM enode는 `56f2f99`이며 수정분은 실제 Bedrock 검증 후 적용한다. 아래는 앞선 Drain 인수 기록이다.
UI/demo-back PR #6은 `666126a`로 병합됐다. card-news의 새 5장 구성도 인수했다.
미커밋 영상 작업을 보존하며 fast-forward했고, 후속 `166c935`에서 기본 주소의
404를 `/ui/` 리다이렉트로 보완해 push했다. 실제 Mac 노드의 at-boundary·graceful,
봉인·queue 승격·화면 표시를 확인했다. [Drain 인수 기록](construction/ui/code/drain-integration-review.md).
기존 queue/sandbox 접점과 진행자 `c7a237d`의 실제 LED·음원 계약을 유지한다.
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
| 갤러리 확장 | 프롬프트 한 번·AI 팀 탐색·실제 댓글 게시·거절/되묻기, main 통합·Mac mini 적용·공개 인수 | 공동 CP6/CP10은 별도 |
| UI 개선 | 웹캠 닫기·복원, 한글 두 단어 이름, 상세·모달·투어의 바깥 조작 닫기 | 구현·로컬 검증 완료 |
| UI 가독성 보완 | 휠 확대·축소, 단계 그래프·함대 복귀, 임대 제출자, 기능별 상세·한국어 시각 | 구현·로컬 검증 완료, 배포 전 |
| UI 9 | queue·sandbox 출처, c7a237d 두 시나리오의 실제 POST/GET·이름·3D 연결 | 실제 장비 광고·동작, 공개 방송 주소 |
| UI 10~11 | 독립 검사·PR #6/#8 병합·실제 Drain 관측과 queue 인수 | 공동 CP 장면과 전체 생성 결과 리뷰 |
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
2026-09-09T02:25:03Z 병합됐다. 병합 사실과 공동 CP10의 실물 검증은 별개다.
기본 주소 접속 보완과 Drain 인수 기록의 [후속 PR #8](https://github.com/taeels/enode-fixup-project/pull/8)은
2026-09-09T02:48:47Z `main@bc41f26`으로 병합됐다.

## 검증 근거

사용자의 별도 지시로 **v0.1.0-rc1 prerelease 게시와 로컬 실행을 완료**했다.
태그는 CI가 통과한 `2d7bf7524d9c`에 고정했다. macOS arm64·amd64,
Windows amd64와 별도 Mediator를 배포했고 설치·실행·다운로드 체크섬을 검증했다.
로컬 Mac의 실제 enode 광고·shell 작업 성공·봉인 record와 LAN 접속도 확인했다.
[rc1 배포 기록](construction/rc1-release.md). 이후 로컬 런타임만 `166c935` 빌드로
갱신했다. rc1 태그·다운로드 자산에는 이번 Drain 인수와 루트 보완이 포함되지 않는다.
현재 기본 주소는 접속 화면으로 이동하며 같은 Mac의 LAN HTTP 검사가 통과했다.
다른 팀원 기기의 URL·오류 응답은 아직 미수령이다. 공동 CP10은 남아 있다.

- Drain 인수 후 Go 1,190 통과·실패/스킵 0, 16패키지 80% 이상.
  API 84.320%, UI 98.387%. Node 46, vet·glyphscan·Mac 빌드/실행·Windows/Linux 교차 빌드 통과.
- 실제 Mac의 at-boundary 첫 단계 DONE·다음 단계 FAILED·drain 사유·봉인 결과물,
  정책 해제 후 QUEUED→RUNNING→SUCCEEDED와 graceful 두 단계 SUCCEEDED를 확인했다.
  UI의 draining 진행 중/대기 중·중단 사유·목록 유지와 해제 후 가용 표시가 일치한다.
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

## 영상 제작 지원

2026-09-09 DDTHON 캐릭터 시트 9종과 고정 참조를 만들고, 첫 15초·요청 및 승인
15초·실제 웹 UI 설명 30초를 연결해 60초 영상을 완성했다.
[영상 문서 인덱스](construction/video/README.md),
[제작 요약](construction/video/code/implementation-summary.md),
[출력·검증](construction/video/code/build-and-test.md)에 최신 기준을 정리했다.

첫 15초와 회사 메신저 폰 합성본은 사용자가 채택했다. 탭 위치의 작은 차이도
수용했다. 후반은 가상 노드 5개로 승인 후 Mediator 전달, 해당 장치의 enode 수신,
Claude Code 실행, 고정된 sandbox 안 작업과 결과 반환을 보여준다.
720p·24fps·60/15/30초와 전체 영상·음성 디코딩, UI 캡처 오류 0,
TypeScript·ESLint를 확인했다. 한국어 대사는 자동 전사로 확인했으며 원어민
청취 검증이나 실제 S2 실행·격리 검증으로 기록하지 않는다.

사용자의 문서화·커밋 지시에 따라 영상 기록만 담당 AI-DLC 문서로 관리한다.
원본·MP4·PNG·ZIP·제작 코드·dist는 로컬에 그대로 보존한다.
기존 제품 유닛 단계와 공동 CP10 상태는 이 영상 완료로 변경하지 않는다.

## Extension Configuration

공용 실행 계획과 requirements/decisions.md §1·§3·§8의 결정을 상속한다.

| Extension | Enabled |
|---|---|
| security-baseline | Yes |
| resiliency-baseline | No |
| property-based-testing | No |

NFR Requirements·NFR Design·Infrastructure Design은 승인대로 SKIP한다.
공용 기준은 construction-roster, 유닛/의존/게이트 매핑, requirements/scene-gates다.
