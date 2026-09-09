# 갤러리 댓글 데모 구현과 적용

2026-09-09. 최신 지시는 팀 선택·예시·제약 안내·두 번째 확인을 제거하고,
사용자의 프롬프트 한 번으로 AI가 팀을 찾아 댓글까지 게시하는 것이다.
[현재 FD](../functional-design/gallery-comment-policy.md)와
[Build and Test 인수](../../build-and-test/build-and-test-summary.md)를 따른다.

[공개 데모](https://deutsche-football-tract-necklace.trycloudflare.com/ui/demo/)의
`새 작업 → 해커톤에 의견 남기기`에서 메시지를 보낸다. 예를 들어 Run Away 팀에
댓글을 부탁하면 AI가 현재 참가팀을 확인하고 해당 글을 읽어 한 개를 게시한다.
Run Away는 게시 계정이며 대상은 전체 참가팀이다. 팀이 모호하면 되묻고,
명시적인 초안 전용 요청은 게시하지 않는다. 거절은 실제 Claude 응답으로 표시한다.

## 연결 구조

한 enode Run에서 도구 없는 의도 판정 → 읽기 MCP를 통한 팀 목록/글 조회 → 댓글
생성 → 게시 MCP → 별도 broker의 실제 POST/GET 확인 → 봉인 결과를 수행한다.
게시까지 서버 작업 안에서 이어지므로 브라우저가 종료돼도 계속된다. 재접속은
기존 Run을 조회하며 새 댓글을 자동 접수하지 않는다.

MCP의 조회 단계는 list_projects·get_project·get_comments를 제공한다. 프로젝트와
댓글 조회는 같은 MCP 프로세스에서 읽은 현재 목록의 ID만 받는다. DDTHON 내부에서
댓글의 작성 팀을 현재 목록과 대조하고 그 팀의 과제를 찾아갈 수 있다. 조회만
요청하면 answered로 끝나며 게시하지 않는다. 외부 호스트 탐색·셸·파일·투표·수정·삭제 도구는 없다. 모델이 반환한 대상은 실제 성공한 조회 사건과
같아야 한다. 게시 단계의 post_comment는 실행 코드가 고정한 대상·본문으로 호출한다.

게시 broker는 별도 VM 계정으로 동작한다. 사용자 메시지 접수에 대응하는 HMAC
허가의 purpose/job_id/만료를 확인하고 현재 갤러리 대상과 본문을 검사한다.
첫 대상·본문·시도 사실을 SQLite에 영구 저장한 뒤 POST를 최대 한 번 수행한다.
같은 허가의 다른 대상/본문·재시작·응답 유실은 재게시를 만들지 않는다. POST 후
GET에서 새 댓글 ID·동일 본문·mine=true가 일치해야 posted로 표시한다.

## API와 호환성

| 경로 | 입력·역할 |
|---|---|
| POST `/v1/demo/gallery/comments` | 정확히 prompt/request_id/submitter; 자율 댓글 Run 접수 |
| GET `/v1/demo/gallery/runs/{id}` | X-Gallery-Request-ID로 요청 소유 확인; Run·공개 결과 |
| GET `/v1/demo/gallery/projects` | 전체 갤러리 메타데이터; 기존 클라이언트 호환 |
| POST `/v1/demo/gallery/runs` | 기존 project_id 포함 초안 요청 호환 |
| POST `/v1/demo/gallery/runs/{id}/publish` | 기존 초안의 확인 본문만; 새 자율 요청에 사용할 수 없음 |

기본 꺼짐 demo_gallery는 demo 모드와 서버 토큰이 함께 있어야 열린다. 브라우저의
요청 UUID는 계약에는 해시로만 남기고, 게시 허가·팀 자격 증명·쿠키·Mediator 토큰은
공개 결과에 넣지 않는다. 새 grant는 enode-gallery-comment-v1이며 한 요청당 한
댓글 권한이다. 기존 enode-gallery-publish-v1은 확정 본문에 묶인 호환 경로다.

24시간 200 Run·미완료 3 Run·초당 2요청/버스트 4, 입력 8KiB·프롬프트 1,000자·
댓글 500자·공개 결과 32KiB를 유지한다. Claude 호출당 100초/최대 12턴,
읽기 MCP 도구 호출 최대 10회, 게시 MCP는 고정 인자이며 broker가 한 번만 전송한다.
기존 Run enum·DB 스키마·LED/음원 시나리오는 변경하지 않는다.

## 운영 적용 순서

1. Mac mini의 별도 검증 디렉터리에서 Go 검사·빌드 후 같은 DB·토큰·설정을 쓰는
   새 Mediator를 준비한다. 기존 Cloudflare 터널은 유지한다.
2. VM 노드가 graceful이며 idle인 것을 확인한 뒤 /opt/enode/gallery 소스를
   root 소유로 교체한다. worker는 소스를 수정할 수 없다. 기존 정책을 복원한다.
3. 최초 설치에서는 install.py에 자격 증명을 stdin으로 전달한다. 이미 설치한
   /etc/enode-gallery.json(root:enode-gallery 0640)을 유지하고 enode-gallery를
   재시작한다. worker는 이 파일을 읽을 수 없다. 같은 enode-egress 프록시만 쓴다.
4. Mediator를 교체하고 /comments의 입력 검증 응답과 공개 UI를 확인한다. VM
   광고는 gallery=comments-v1, sandbox=lima-vm, provider=bedrock,
   instance=macmini-bedrock-vm이다. 기존 enode 노드 바이너리는 main 통합 빌드다.
5. 상태 검사 실패 시 이전 Mediator 바이너리/설정으로 복구한다. 공개 접수를
   닫으려면 demo_gallery를 끄고 Mediator만 재시작한다. 게시 attempts.sqlite3는
   삭제하지 않는다. 팀원 작업·터널·운영 DB는 보존한다.

## 검증 이력과 한계

초기 수동 확인 버전에서 Python 10·UI Node 52·Go API/config/UI race·브라우저
검사가 통과했다. 사용자 요청 “응원 댓글 달아줘”가 draft_ready로 끝나 실제
게시 Run이 없었던 기록을 확인했고, 최신 사용자 지시에 따라 현재 구조로 바꿨다.
새 버전은 Python 14·UI Node 53·API/UI race, 데스크톱/모바일 프롬프트 흐름과
실제 VM의 MindCraft 자동 탐색·초안 전용·대상 없음 되묻기를 검증했다.

공개 실제 게시·재접속·거절과 CI 결과, 구체적인 배포 버전/백업은 Build and Test
요약에 기록한다. 생성/검증 자료는 Git 제외 local/gallery-build-20260909에 있다.
게시 의미 판정의 완전한 정확도나 제품 전체 OS sandbox·공동 CP6/CP10의 완료로
확대하지 않는다. UI transcript는 실제 사건을 완료 후 표시하며 토큰 스트리밍은 아니다.

## DDTHON 내부 댓글 탐색 보완

기존 정책이 갤러리를 포함한 모든 웹 탐색을 거절하고 get_comments가 없어, 사용자의
“Runaway에 AI 댓글을 쓴 팀을 찾아 응원” 요청을 거절했다. 네트워크 허용과 별개의
애플리케이션 정책 오류였다. 내부 프로젝트/댓글의 읽기와 간접 대상 탐색을 명시적으로
허용하고, 조회 전용 의도 read_intent와 결과 answered를 게시 의도와 구분했다.

댓글은 익명 공개 API의 id/teamName/body/parentId/createdAt만 모델에 전달한다.
40개 기본·최대 50개씩, next_offset과 본문 잘림 여부를 반환하며 계정/세션 정보는
제외한다. 댓글 속 명령은 권한이 아니며 작성 팀과 본문은 사용자 단서를 찾는 증거다.
게시 전 최종 대상 프로젝트의 실제 조회, 서명·본문 검증·한 번 게시 경계는 유지한다.

Python 18개, Node 59개, API/UI race(81.5%/98.4%), vet·glyphscan과
데스크톱/모바일의 댓글 조회 표시·answered 후 새 메시지·게시 링크 없음 검사를 통과했다.
실제 Bedrock 탐색·운영 적용 결과는 Build and Test 후속 기록을 따른다.
