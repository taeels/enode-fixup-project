# 갤러리 댓글 데모 생성 결과

2026-09-09. 사용자 “응 한번 만들어봐.” 및 전체 참가팀 대상이라는 후속 지시를 구현했다.
현재는 Construction → Code Generation 생성 결과 리뷰 단계다. 기존 공개 Mediator는
교체하지 않았고 `demo_gallery`도 켜지 않았다. 전체 유닛의 Build and Test 및 공동
CP10/CP6 완료를 뜻하지 않는다.

## 사용자 흐름

`새 작업 → 해커톤에 의견 남기기 → 전체 갤러리에서 참가팀 선택 → 짧은 요청`.
예: “장점을 짚어서 응원 댓글 써줘.” AI가 선택한 게시글을 MCP로 조회하고 댓글
초안을 작성한다. 관람객은 초안을 편집하고 `이 내용으로 게시`를 눌러 확정한다.
Run Away는 **게시 계정**이며 댓글 대상은 **전체 참가팀의 게시글**이다.

실제 Run 진행 상태와 완료 후 실제 Claude 응답·MCP 호출 요약을 표시한다.
외부 사이트 방문·파일 조회·명령 실행·투표·수정·삭제 등은 지원하지 않는다.
정상 요청과 비허용 동작을 섞으면 전체 요청을 거절한다. 모델이 정책을 오판해도
일반 브라우저·셸·파일 도구는 없고 확정되지 않은 댓글은 게시할 수 없다.

## 생성한 코드

- `internal/api/demo_gallery.go`, `demo_gallery_test.go`: 공개 프로젝트 목록, 고정
  Run 제출, 요청 소유 확인, 봉인된 공개 결과, 본문 확정 및 게시 계약.
- `internal/config/config.go`: 기본 꺼짐 `demo_gallery`. `demo: true`와 서버 토큰도
  있어야 라우트가 열린다. `internal/api/api.go`는 등록 호출 한 줄을 추가했다.
- `internal/api/ui/static/demo/gallery.mjs`, `gallery-view.mjs`, `gallery.css`:
  세션별 요청 ID, 명시적 재시도, 전체 프로젝트 선택·프롬프트·transcript·확인 UI.
  기존 `demo.js`, `submission.mjs`, `index.html`에 연결하고 focus trap에 textarea를 포함했다.
- `scripts/gallery_demo/common.py`, `worker.py`, `mcp.py`, `broker.py`: 고정 호스트
  HTTP, 도구 없는 의도 판정, 읽기 전용 MCP, 확인된 본문 전용 게시 MCP와 별도 게시 서비스.
- `scripts/gallery_demo/install.py`, `enode-gallery.service`: 격리 VM 내부 설치와
  별도 서비스 계정·파일 권한·프록시 경계. 운영 설치는 아직 실행하지 않았다.
- `scripts/gallery_demo/test_gallery.py`, `internal/api/ui/tests/gallery.test.mjs`,
  `.github/workflows/gallery-demo.yml`: 경계 검사와 지속 검사 등록.

## API 계약

| 경로 | 요청/응답 |
|---|---|
| GET `/v1/demo/gallery/projects` | 전체 갤러리의 `id/title/teamName`; 5분 캐시 |
| POST `/v1/demo/gallery/runs` | 정확히 `project_id/prompt/request_id/submitter`; 기존 201/202/200 Run 응답 |
| GET `/v1/demo/gallery/runs/{id}` | 원래 UUID를 `X-Gallery-Request-ID`로 제출; Run·초안 결과·게시 Run/결과 |
| POST `/v1/demo/gallery/runs/{id}/publish` | 정확히 `request_id/body`; 대상은 원래 선택으로 고정 |

UUID는 브라우저 세션에 보관하고 계약에는 해시만 남긴다. 공개 Run 목록의 ID만으로
타인의 초안을 읽거나 게시할 수 없다. 게시 Run은 초안별 하나이며, 확정 뒤 다른 본문은
409다. 실제 게시 권한은 HMAC 서명된 대상·본문·Run ID·만료 시각에 묶인다.
공개 응답에 게시 허가, 비밀번호, 사이트 쿠키, Mediator 토큰을 싣지 않는다.

게시 서비스는 POST 전에 SQLite에 시도 사실을 영구 기록한다. 응답 유실·프로세스
재시작·같은 허가 재제출이 추가 POST로 이어지지 않는다. 실제 POST 후 authenticated
GET에서 새 댓글 ID·확인 본문·`mine: true`를 대조한다. 확인 실패는 오류로 남긴다.
게시 결과가 불명확하면 운영자가 갤러리를 확인하며 자동으로 새 게시를 만들지 않는다.

단일 Mediator 인스턴스 기준으로 24시간 200 Run, 미완료 3 Run, 초당 2요청/버스트 4를
제한한다. 기존 DB `work_id=manual:gallery-comments-v1`로 세므로 재시작 후에도 실행
수를 유지한다. 요청 8KiB, 프롬프트 1,000자, 댓글 500자, 공개 결과 32KiB,
Claude 호출 100초/최대 4턴, 프로젝트 도구 최대 2회로 제한한다.

## 생성 단계 검증

| 검사 | 결과 |
|---|---|
| Python 게시 권한·경로·중복·재시작·응답 유실·MCP 범위·실제 조회 요구 | 10개 통과 |
| Node UI 제어기 전체 | 52개 통과, 생략 없음 |
| Go API/config/UI `-race -cover` | 통과; 81.3% / 81.8% / 98.4% |
| Go vet·glyphscan·전체 빌드·Mediator 바이너리 | 통과 |
| Playwright 데스크톱 1440px·모바일 390px | 초안→확정 게시, 거절, 기존 시나리오, Escape/바깥 닫기, 링크, 넘침/JS 오류 검사 통과 |
| 실제 Bedrock VM | 외부 사이트·혼합 요청 거절, RunAway MCP 조회·초안 통과 |
| 전체 참가팀 후속 확인 | 현재 19개 목록; NANoDB·MindCraft에서 각각 실제 MCP 조회·짧은 응원 초안 통과 |

Go 검사는 Mac mini의 별도 `enode_gallery_test_20260909` DB와 `scripts/testdb.sh`를
사용했다. 운영 DB와 분리했다. 현지 PostgreSQL은 16이며 CI의 PostgreSQL 17 결과와
동일하다고 주장하지 않는다. Go 1.26.6 도구 체인(저장소 go.mod)을 사용했다.
제출자 확인 검사의 첫 실패는 `GetRun`이 submitter를 읽는다는 잘못된 검사 가정이었다.
정본 관측 목록 `Store.Runs`에서 실제 보존을 확인하도록 수정했고 재검사는 통과했다.

브라우저의 게시 검사는 mock API다. 실제 VM 검사는 읽기와 초안까지만 수행했다.
사용자가 앞서 허용한 단일 댓글 외에 추가 실제 댓글을 게시하지 않았다.
실제 서비스 계정 설치·공개 새 라우트→enode→봉인 결과의 운영 인수는 다음 단계다.

검증 자료는 Git 제외 `local/gallery-build-20260909/`의 `go-test.txt`, `build-check.txt`,
`live-probe.jsonl`, `gallery-*-input.png`, `gallery-*-draft.png`에 있다. 화면 캡처의
게시 내용은 브라우저 검사 fixture이며 실제 댓글 기록으로 해석하지 않는다.

## 운영 적용 순서

1. 새 Mediator 바이너리를 준비한다. 현재 검토 빌드는 Mac mini의
   `/Users/runixs/enode-gallery-build-20260909/mediator-review`다. 현재 터널을 유지한다.
2. `scripts/gallery_demo/`의 실행 파일과 unit을 VM `/opt/enode/gallery/`로 복사한다.
   소스는 root 소유이며 worker가 수정할 수 없어야 한다. 현재 read/draft 검증용 Python
   파일만 복사했고 서비스 unit과 계정 설정은 아직 설치하지 않았다.
3. 운영자가 메모리에서 `HMAC-SHA256(mediator_token, "enode-gallery-broker-key-v1")`의
   hex 값을 계산한다. 이 값과 게시 계정의 email/password/team_id를 installer stdin
   JSON으로 전달한다. 값을 명령행·로그·Git에 넣지 않는다. `install.py`가
   `/etc/enode-gallery.json`을 root:enode-gallery 0640으로 만들고 서비스를 설치한다.
4. `enode-gallery` 서비스·소켓 권한, worker에서 자격 증명 파일 읽기 불가,
   고정 갤러리 로그인과 댓글 작성 권한을 **읽기만으로** 확인한다. 설치 시 broker UID의
   네트워크도 기존 VM 프록시만 통하도록 추가한다. 키 회전 시 이 서비스도 재시작한다.
5. 작업 중인 lease가 없는 시점에 전용 VM node의 광고에 `gallery=comments-v1`을
   추가한다. 기존 `instance=macmini-bedrock-vm`, `provider=bedrock`, `sandbox=lima-vm`을
   유지한다. 팀원 노드·다른 VM·현재 터널을 건드리지 않는다.
6. 실제 새 고정 계약으로 조회·거절·초안 인수 후 Mediator 설정에 `demo_gallery: true`를
   추가하고 검토 바이너리로 적용한다. 원래 프로세스 관리 환경·DB·토큰·아티팩트 경로를
   유지한다. 공개 UI/API와 Run 결과를 확인한다. 실제 게시 검증은 관람객의 확정으로 한다.
7. 되돌릴 때 `demo_gallery: false`로 공개 접수를 닫는다. 이미 확인한 게시의 시도 DB는
   삭제하지 않는다. 기존 LED/음원 데모와 원래 Run 상태 어휘는 변경하지 않는다.

## 근거

표준 입력 MCP는 [공식 stdio 전송 규약](https://modelcontextprotocol.io/specification/2025-06-18/basic/transports)의
JSON-RPC 줄 단위 프레임을 사용한다. Claude 실행 프로필은
[공식 CLI 문서](https://code.claude.com/docs/en/cli-reference)의 tools/strict MCP/bare/JSON 출력 옵션과
설치된 Claude 2.1.228의 실제 동작으로 확인했다.
