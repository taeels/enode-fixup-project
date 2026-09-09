# 갤러리 Run 대화 기록 후속 계획

기존 ui·demo-back의 승인된 Construction 후속 변경이다. 사용자의
“galley로 run을 한 대화내용을 run history에서도 보게 할 방법없나??? origin/main이 업데이트됐으니까 rebase해서 작업해”를 구현 지시로 받았다.
화면 확인 질문에 응답이 없어 공개 웹 작업 목록을 기준으로 진행한다.

## 기능과 경계

사용자는 작업 목록에서 이전 갤러리 Run을 골라 대화·도구 사건·게시 결과를 다시
읽는다. 봉인된 `gallery-result.json`을 재사용하므로 Claude나 댓글 게시를 재실행하지
않는다. 진행 중·실패·기록 없음·권한 없음은 결과를 지어내지 않고 상태로 보여준다.

게스트는 브라우저에 보존한 요청 증명으로 자신의 결과만 읽는다. 새 메시지/reset이
이 증명을 지우지 않으며 기존 sessionStorage의 현재 요청도 인수한다. 본문과 토큰은
이력 저장소에 저장하지 않는다. 이전에 잃어버린 증명은 복구할 수 없다.
토큰 인증을 마친 실 함대 화면은 별도 읽기 API로 기존 Run도 조회한다. 서버는 기존
갤러리 계약·봉인 결과 검증을 거친 공개 transcript 필드만 반환한다.

## 구현 순서

- [x] 1. `origin/main@57b8a1a` rebase. 새 Run 흐름·보기 설정을 보존하고 audit union 충돌의 원문 블록을 복원.
- [x] 2. `internal/api/demo_gallery.go`와 별도 `demo_gallery_history.go`에 인증된 읽기 경로 추가. 기존 결과 검증·legacy 게시 결과 재사용.
- [x] 3. UI shared gallery history controller/dialog, 요청 증명 보존, demo/fleet 작업 목록의 대화 기록 연결. 기존 입력/실행 상태와 독립.
- [x] 4. 권한·기존 봉인 결과·reset/reload·GET 전용·오래된 응답·HTML 본문 표시를 집중 검증. 외부 댓글은 게시하지 않음.
- [ ] 5. 담당 상태·구현/검증 문서·커밋·PR 준비.

변경 파일은 runixs의 ui·demo-back 안에 둔다. `internal/panel`과 nacl1119의
공동 CP6, CP10은 범위 밖이다. 원격 운영 업데이트 시 추가 검사를 하지 말라는
사용자 지시를 계승한다. 이번 코드 검증은 격리된 개발 fixture로 수행한다.
security-baseline은 기존 인증/출력 제한을 유지한다. 비활성 resiliency-baseline과
property-based-testing은 재선택하지 않는다.
