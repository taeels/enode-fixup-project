# 갤러리 확장 Build and Test

2026-09-09. runixs UI/demo-back의 갤러리 확장과 최신 main 통합을 검증하고 Mac mini에
적용했다. 현재 사용자 흐름은 메시지 한 번 → AI의 팀 탐색·글 조회·댓글 게시다.
팀 선택·제약 목록·두 번째 게시 확인은 없고 입력칸 placeholder에 안내/예시만 둔다.
기존 ui/demo-back 공동 CP6/CP10의 전체 완료를 뜻하지 않는다.

## 빌드와 검사

| 검사 | 결과 |
|---|---|
| main 통합 | origin/main 16ef08c로 rebase; transcript·UI 가독성·greet-play 예제 인수 |
| Mac mini 전체 Go -race -coverpkg=./... | rebase 후 1,244 통과·실패/생략 0; Go 1.26.6·전용 PostgreSQL 16 |
| rebase 후 패키지별 커버리지 | 18개 모두 80% 이상, 전체 86.56%; API 81.18%, UI 98.39%, enode 84.57%, panel 84.89% |
| 제어판 검사 보완 | macOS stop 검사 20회 반복 통과; cmd/enode 진입점 추가 뒤 97.2% |
| 자율 댓글 후 Go API/UI -race -cover | 통과; 81.2% / 98.4% |
| Python 게시·MCP·서명·중복 경계 | 14개 통과 |
| Node UI 전체 | rebase 후 59개 통과·생략 0 |
| vet·gofmt·glyphscan | rebase 후 통과, glyphscan 102파일 |
| 운영 빌드 | macOS arm64 Mediator·enode·enodectl, Linux arm64 enode, Windows amd64 enode |
| 이전 CI | f4a9b1b의 test·cross·bounded-demo 모두 통과; PostgreSQL 17, 패키지별 80%·skip 감시 포함. rebase 후 PR #14에서 재실행 |
| 브라우저 mock | 1440px/390px 입력·게시·되묻기·거절·닫기·기존 시나리오·넘침/JS 오류 통과 |
| 실제 Bedrock VM | MindCraft를 직접 찾아 글 조회·초안만 반환, 팀 미지정이면 목록 확인 후 되묻기 |
| 공개 실제 연결 | 단일 메시지·새로고침·동일 Run 완료·실제 댓글·외부 요청 거절 통과 |

전체 검사 최초에는 main의 새 panel 검사에서 셸 exec 최적화로 설정 argv를 잃는
문제를 발견했다. 전용 Go 보조 프로세스로 바꾸고 packaging 예제를 원격 검증
복사에 포함해 생략을 없앴다. CI의 cmd/enode 75.4%는 새 panel 진입점의 검사가
없었기 때문이었다. 설정/정책 오류·인증 없는 외부 바인딩·포트 충돌을 실제 진입점에서
검사해 97.2%로 보완했다. 게이트 하한과 제품 제어판 코드는 바꾸지 않았다.

## 16ef08c rebase 인수

기존 PR #14를 최신 main 위로 재배치했다. upstream 제품 파일 17개는 main과
바이트 단위로 같으며 갤러리 제품 코드는 이전 b01274b의 구현을 유지한다.
감사 union 병합으로 뒤섞인 항목을 복원해 main 감사 원문 전체와 이전 브랜치의
고유 21개 항목을 보존했다. 담당 상태의 PR #16 병합 여부와 Git/운영 버전을 구분했다.
위 전체 Go·Python·Node·브라우저·5개 실행 파일 빌드는 rebase 뒤 다시 통과했다.
이 검증은 별도 소스 복사와 테스트 DB에서 수행했고 실제 댓글을 추가 게시하지 않았다.
운영 Mediator는 b01274b, VM enode는 56f2f99이며 추가 main 변경은 배포 전이다.

## 공개 실제 실행

- 공개 UI: https://deutsche-football-tract-necklace.trycloudflare.com/ui/demo/
- 사용자가 앞서 요청했지만 초안에서 멈춘 “응원 댓글 달아줘”를 Run Away 대상으로
  한 번 재실행했다. 새 요청의 submitter는 같은 “꾸준한 단풍”이다.
- Run: `gallery-dc443969c6631068f24a3bb8ad3e1de93a76dd9c9d7f542845cf593938ea4d0e`.
- 실제 도구 순서: list_projects → get_project → post_comment. 당시 목록 20개.
- 댓글 ID: `398167b8-ffb8-4c22-bb88-b6b55bcec63a`.
- 대상: [RunAway 원문](https://main.d3gkmtkue9o7ly.amplifyapp.com/projects/mttmqvyjaw46r).
  게시 후 공개 GET에서 댓글 ID·동일 본문이 정확히 한 개 존재함을 확인했다.
- 실행 중 새로고침한 뒤 같은 Run을 읽어 완료했다. 브라우저 접수 POST는 한 번이고
  별도 게시 확인 API 요청은 없었다. 기존 초안 Run을 소급 자동 게시하지 않았다.
- 외부 페이지 방문 요청은 `gallery-fc9a2b6f6da63aacf6248d31dc71684c57544b38b5312ec56ccc9c7d5002a04c`
  에서 refused, MCP 호출 없음. 브라우저 JS 오류 0. 추가 실제 댓글 게시 없음.

## 운영 배치·복구

Mac mini 루트는 /Users/runixs/enode-public-demo다. 초기 main 통합 제품 56f2f99를
적용했고, 자율 댓글 API/UI는 f4a9b1b로 갱신했다. 해당 Mediator 교체는 10.70초였고
기존 터널 PID 70718·공개 주소·DB·토큰·아티팩트를 유지했다. enode 바이너리는
56f2f99이며 이후 변경은 API/UI와 VM Python 코드이므로 실행 노드 코드는 동일하다.
입력칸 안내와 최종 문서의 후속 적용 버전은 Git/운영 version 출력을 따른다.

Mediator 백업: /Users/runixs/enode-public-demo/backup-gallery-auto-20260909T051958Z.
VM 소스 백업: /opt/enode/gallery-before-autonomous-20260909T052052Z.
프로세스/PID·직전 백업은 운영 루트의 gallery-autonomous-deployment.json에 있다.
VM 원래 drain 정책을 복원했으며 enode-bedrock/enode-gallery/enode-egress가 활성이다.
worker에서 별도 게시 자격 증명을 읽을 수 없고, HMAC 위조 거부와 실제 팀 계정의
댓글 작성 권한을 확인했다. 게시 시도 DB는 영구 보존한다.

팀원 노드를 중지하지 않았다. pi-mpg123-1과 앞선 greet-play 대기 작업의 SUCCEEDED를
확인했고 팀원 3개와 전용 VM의 heartbeat가 갱신됐다. MacBook 서비스는 계속 중지다.

## 범위와 자료

[빌드](build-instructions.md), [단위 검사](unit-test-instructions.md),
[통합 검사](integration-test-instructions.md)를 재현 절차로 사용한다.
Git 제외 local/gallery-build-20260909의 integrated-build.txt, integrated-coverage.txt,
go-autonomous.txt, node-autonomous.txt, autonomous-live-probe.jsonl,
autonomous-public-e2e.json, autonomous-public-*.png와 운영 배포 기록이 증거다.
rebase 후 node-rebased.txt, coverage-rebased.json과 Mac mini 격리 빌드 경로의
integrated-tests.jsonl도 후속 검사 증거다.
raw 요청 UUID·설정·자격 증명·바이너리는 Git/PR에 포함하지 않는다.

수용된 security-baseline 범위와 예외를 유지한다. 고정 목적/URL·허용 도구·요청별
한 번 게시·권한 분리·입력/출력/시간/호출 수 제한·실제 오류/거절 표시를 검사했다.
resiliency/PBT와 NFR/Infra의 기존 선택은 변경하지 않았다. 별도 부하 시험·제품
전체 OS sandbox·공동 하드웨어/방송 장면은 이 결과에 포함하지 않는다.

표준 CI: [Go·coverage·cross](https://github.com/taeels/enode-fixup-project/actions/runs/34314414892), [Python·UI](https://github.com/taeels/enode-fixup-project/actions/runs/34314414875).

## DDTHON 내부 댓글 탐색 보완 검증

main `50899b0`(PR #14 병합)에서 후속 수정했다. Python 18·Node 59, API/UI
race(81.5%/98.4%), vet·gofmt·glyphscan, 데스크톱/모바일의 댓글 조회 표시와
조회 완료 후 새 메시지·게시 링크 없음 검사를 통과했다. 기존 게시·되묻기·거절
흐름도 브라우저 mock으로 재확인했다.

실제 Bedrock VM에서 후보 소스를 별도 root 소유 디렉터리에 두고 다음을 검증했다.

- Runaway의 AI 댓글 작성 팀 찾기→MindCraft 과제 `mttorge8ag7kv` 조회→
  AI 작성 표시·실제 장점과 응원을 담은 초안 반환. 첫 댓글 조회 실패 후 재조회에 성공했다.
- 허용된 DDTHON URL을 포함한 댓글 조회 전용 요청→answered, MindCraft 식별.
- Runaway 댓글 조회와 example.com 방문의 혼합 요청→도구 호출 없는 refused.

19개 팀인 당시 목록에서 댓글의 작성 팀을 직접 찾았으며 대상 팀 이름을 검증
프롬프트에 미리 제공하지 않았다. 테스트는 초안/조회만 요청했고 게시 호출도
차단해 추가 실제 댓글을 쓰지 않았다. 근거는 Git 제외 discussion-live-probe.jsonl이다.
운영 적용과 후속 PR 결과는 담당 상태·감사 로그에 기록한다.

운영 Mediator·VM Python을 `1b5d8aa`로 적용했다. Mediator 교체는 10.65초,
PID 43479이며 터널 PID 70718과 공개 주소를 유지했다. VM 3개 서비스는 active,
원래 drain 정책을 복원했고 배치한 Python 3개 파일의 SHA-256이 검사 소스와 같다.
Mediator 백업은 backup-gallery-discussion-20260909T060132Z, VM 소스 백업은
/opt/enode/gallery-before-autonomous-20260909T060202Z다. VM enode 바이너리는 56f2f99다.

공개 UI Run `gallery-63cf3ef1302c002a166fc5ed000fd457f4872587994d9241456a482007e53121`은
현재 참가팀 20개→Runaway 댓글 18개→MindCraft 식별→answered로 완료했다.
접수 POST 한 번, 게시 도구 호출·게시 링크 없음, 댓글 읽기 표시·새 메시지 정상,
브라우저 오류 0이다. discussion-public-result.json·discussion-health.json이 증거다.
후속 [PR #17](https://github.com/taeels/enode-fixup-project/pull/17)에서 검토한다.
