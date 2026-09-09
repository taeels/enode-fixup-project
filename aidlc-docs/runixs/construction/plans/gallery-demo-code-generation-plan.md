# 갤러리 댓글 데모 구현 순서

2026-09-09 사용자 “응 한번 만들어봐.”로 추가 FD와 아래 생성 순서 승인. Construction → Code Generation으로 진행한다.
[요청 정책과 transcript](../demo-back/functional-design/gallery-comment-policy.md)를 기준으로 한다.

- [x] 지정 사이트 조회·팀 로그인·댓글 한 개 게시·게스트 화면 노출 검증.
- [x] Bedrock 의도 분류 시험: 외부 탐색 거절, 혼합 요청 거절, 정상 댓글 의도 인식.
- [x] 새 고정 데모 제출·결과 계약 확정. 기존 LED·음원·request_id 재시도 유지.
- [x] 게스트 전용 Claude 설정과 갤러리 MCP 구현. 실제 노출 도구·대상 범위 검증.
- [x] 팀 세션을 분리한 게시 서비스, 대상·본문 확인과 일회성 게시·중복 방지 구현.
- [x] 실제 Claude 응답·MCP 사건에서 공개 transcript 산출, 비밀 값 제외·크기 제한.
- [x] 새 작업의 프로젝트 선택·프롬프트·진행 상태·거절·초안·게시 결과 UI 연결.
- [x] 서버·MCP 경계 검사: 잘못된 대상·경로·혼합 요청·권한 위조·중복·비밀 값 비노출.
- [x] 브라우저와 API의 정상·거절·오류·대기·재접수 검증. 표준 검사 후 공개 배포 준비.

실제 외부 게시 검증은 사용자가 허용한 대상·본문·횟수 안에서 한다. 앞선 댓글 한 개는
그 지시로 이미 게시했다. 임의의 추가 테스트 댓글을 게시하지 않는다.
공유 제어판 transcript·CP6·제품의 OS sandbox 전체 구현은 이 계획의 완료 대상이 아니다.

## 생성 경로와 순서

1. `scripts/gallery_demo/`에 고정 갤러리 HTTP 클라이언트, stdio MCP, Bedrock worker, 별도 계정 게시 broker와 Python 경계 검사를 생성한다.
2. `internal/api/demo_gallery.go` 및 `_test.go`에 고정 계약 제출·소유 의도 확인·봉인 결과·게시 확정 API를 만든다. `api.go`는 등록, `internal/config/config.go`는 기본 꺼짐 스위치만 추가한다. 기존 queue/record를 재사용하며 DB 스키마는 추가하지 않는다.
3. `internal/api/ui/static/demo/gallery.mjs`, `gallery-view.mjs`, `gallery.css`와 `internal/api/ui/tests/gallery.test.mjs`를 만들고 기존 새 작업 모달에 연결한다. 진행 상태·실제 transcript·초안 편집·명시적 게시 확정을 구현한다.
4. 서버/worker/UI 검사를 실행하고 `construction/demo-back/code/gallery-implementation.md`에 결과·제약·설치 절차를 기록한다. 운영 배포 전 검토 가능한 결과를 만든다.

기능 추적: 게스트 기여(프로젝트 선택/댓글) → 1~3; 범위 거절/실행 제한 → 1~2; 중복 게시 방지/관측 → 1~4. 공유 Run enum과 기존 LED/음원 계약은 그대로 사용한다.

## 생성 결과

생성과 독립 검증을 마쳤다. [구현·검사·운영 적용 기록](../demo-back/code/gallery-implementation.md)을 검토한다.
전체 참가팀 대상이라는 사용자 후속 지시를 반영했다. 후속 “그러면 적용해봐”로
생성 결과 승인을 수령했고 최신 main 통합·Mac mini 공개 적용까지 수행했다.
[Build and Test 결과](../build-and-test/build-and-test-summary.md)에 전체 검사와
공개 실제 조회·초안·거절을 기록한다. 새 검사 workflow는 `.github/workflows/gallery-demo.yml`이다.


## 최신 후속 — 팀 탐색과 한 번에 게시

사용자의 추가 지시로 기존 팀 선택·두 번째 확인을 없애는 변경을 승인받았다.

- [x] 현재 요청이 draft_ready에서 끝나 게시가 없었던 원인 확인.
- [x] 프롬프트 전용 댓글 API와 요청별 한 번 게시 허가 추가.
- [x] AI의 현재 참가팀 목록 조회·후보 판별·실제 글 조회 연결.
- [x] 정상 댓글 요청은 같은 Run에서 게시까지, 초안 전용/모호한 대상은 미게시.
- [x] 입력 화면의 팀 선택·예시·범위 안내·두 번째 게시 확인 제거.
- [x] API·MCP·게시 중복·브라우저 경계 검사.
- [x] Mac mini 재적용과 공개 프롬프트→실제 게시·새로고침 인수.

이 절이 앞선 수동 확인 흐름보다 최신이다. 기존 원문·검사 결과는 역사적 증거로 보존한다.

## 후속 — DDTHON 내부 탐색 허용 정정

사용자의 범위 정정으로 진행한다. 원래 허용한 사이트 내부의 댓글/참가팀 탐색을 금지한 구현을 고친다.

- [x] `common.py`·`mcp.py`: 공개 댓글 본문/작성 팀·답글 관계의 페이지별 조회. 현재 목록의 프로젝트만 읽고 임의 URL/계정 필드는 받지 않는다.
- [x] `worker.py`: DDTHON 내부 탐색 허용, 간접 단서로 대상 찾기, 조회 전용 결과와 게시 의도 분리.
- [x] Go 결과 검증·UI transcript에 댓글 조회와 조회 완료 연결.
- [x] 실제 댓글→작성 팀→그 프로젝트 읽기를 mock/Bedrock 초안으로 검사. 외부 URL·혼합 요청·댓글 주입·중복 게시 경계 회귀.
- [ ] Mac mini 적용·공개 조회 확인·PR #14 갱신. 추가 실제 댓글 게시 없음.
