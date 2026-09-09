# 갤러리 Run의 저장된 대화 조회

## 결과

웹 데모/실 함대의 작업 목록에서 갤러리 Run을 선택하면 작업 그래프 상단에
**대화 기록** 버튼이 나타난다. 요청·Claude 응답·도구 사건·게시 본문·프로젝트
링크를 별도 읽기 창에서 보여준다. 실행 중에는 완료 대기, 기록 없는 실패/취소는
해당 상태, 조회 실패는 재조회 안내를 낸다. 대화 기록을 열어도 작성 중인 새
메시지나 현재 실행 상태를 바꾸지 않는다. 이 기능은 댓글을 다시 게시하지 않는다.

`gallery-result.json`은 기존 Run에도 봉인돼 있다. 현재 로그에는
`gallery result: <outcome>`만 쓰므로 로그 뷰를 넓히는 대신 기존 결과 검증을
재사용했다. 구형 초안 Run과 `gallery-post-*` 게시 Run도 함께 표시한다.
이번 범위는 웹 ui/demo-back이며 로컬 enode 제어판은 변경하지 않는다.

## 조회 경계

- 게스트: 기존 `GET /v1/demo/gallery/runs/{id}`와 `X-Gallery-Request-ID`.
  자기 요청 증명만 `enode.demo.gallery.history.v1` localStorage에 최대 200개
  보존한다. 기존 sessionStorage의 현재 요청도 자동 인수한다. 본문·토큰은
  이력 저장소에 저장하지 않는다. 저장소 차단 시 현재 탭 메모리만 사용한다.
- 실 함대: `GET /v1/demo/gallery/history/{id}`는 Mediator 토큰 인증이 필요하다.
  갤러리 고정 계약인지 확인하고 기존 봉인 artifact의 허용 필드만 반환한다.
  Demo와 DemoGallery가 켜진 서버에서 제공한다.
- 읽기는 GET만 사용한다. 원본 계약·worker 인자·게시 ticket은 반환하지 않는다.
  요청 증명을 잃은 과거 대화는 토큰 인증된 실 함대에서 조회한다. 다른 브라우저의
  게스트에게 임의 공개하지 않는다. 링크는 검증한 프로젝트 ID와 고정 DDTHON
  origin으로 재생성하고 메시지는 DOM textContent로 표시한다.

## 검증

2026-09-09 로컬 격리 fixture 기준:

- `node --test internal/api/ui/tests/*.test.mjs`: 69개 통과, 실패/생략 없음.
  reset/reload/구형 session 인수, 최대 개수·복수 탭, 증명 없음, 토큰 인증 경로,
  잘못된 응답·오래된 응답·창 닫기·GET 전용을 포함한다.
- 별도 생성 후 삭제한 `enode_history_test_*` PostgreSQL DB에서
  `scripts/testdb.sh`를 거쳐 `go test -race -cover ./internal/api ./internal/api/ui`:
  81.6%/98.4%. 저장된 결과·토큰 없음/오류·게스트 증명 유지·POST 거절·구형 게시
  기록·위조 계약·고정 링크·비밀 필드 비노출을 확인했다.
- Playwright로 실제 demo/fleet 진입점의 1440×1000, 390×844 네 환경을 확인했다.
  저장된 메시지·HTML 문자열·게시 링크·미작성 메시지 보존·새로고침·증명 없음·
  키보드/배경 닫기·포커스 복귀·모바일 넘침·원래 갤러리 표기를 확인했다.
  네트워크는 mock GET만 허용했으며 실제 댓글이나 Claude 실행은 하지 않았다.
- `go vet ./internal/api ./internal/api/ui`, glyphscan 107파일, Mediator 빌드 통과.
  담당 문서와 변경 코드의 diff 공백 검사 통과. audit 원문은 역사대로 보존한다.

보안 확장은 기존 인증/출력 제한을 유지한다. 비활성 확장은 그대로 건너뛰며
공동 하드웨어 CP6/CP10을 완료로 바꾸지 않는다. 운영 업데이트의 추가 검사를
하지 말라는 사용자 지시는 유지한다.

## 표시명 철회

사용자의 후속 지시로 표시명 별칭을 전부 제거했다. 단계/용도는 `gallery`,
목록 ID는 기존 일반 해시 축약, 새 작업/입력창은 **해커톤에 의견 남기기**다.
복원 커밋 `597a784`는 Mac mini에 빌드·교체했다. PID 58117,
백업 `backup-server-update-20260909T064309Z`, 터널 PID 70718이다.
운영 상태/API/브라우저 검사는 실행하지 않았다.
