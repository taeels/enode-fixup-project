# Mac mini 공개 데모 실행

사용자의 “한번 해봐” 지시에 따라 Mac mini에서 공개 접속을 직접 확인한다.
기존 Code Generation·공동 CP10 상태를 바꾸는 승인이 아니라 실행 환경 구성 요청이다.

- [x] MacMini SSH 별칭의 실제 호스트·아키텍처·서비스 확인. MacBook은 중지 상태 유지.
- [x] 별도 실행 디렉터리·DB·토큰 준비. CI 통과한 main과 제품 코드가 같은 166c935 바이너리 배치.
- [x] Mediator·실제 Mac mini 노드 기동. 먼저 loopback HTTP·광고·shell 작업 확인.
- [x] Cloudflare Quick Tunnel로 임시 HTTPS 주소 발급. 계정·고정 도메인은 현재 입력이 없다.
- [x] MacBook에서 공개 UI·API·토큰 인증·POST 보존·실제 클라이언트 연결 확인.
- [x] 실행/종료 명령·주소·환경 한계와 담당 상태·audit 기록.

Mac mini의 별도 PostgreSQL DB와 원래 제품의 demo 모드를 사용한다. 기존 태양님
공개 Mediator의 DB·토큰·노드 등록은 변경하지 않는다. 실제 하드웨어를 가장한 광고를
만들지 않으며 LED·음원·방송 CP10은 입력이 준비돼야 검증한다.
Quick Tunnel 주소는 임시이며 재생성 시 바뀔 수 있다. 설정·토큰은 Mac mini의
사용자 전용 파일에 저장하고 Git에 올리지 않는다. 운영 프로세스는 SSH 접속 수명과
분리해 유지하고 해당 환경만 종료하는 관리 명령을 제공한다.

검증 결과와 관리 명령은 [실행 기록](../macmini-public-demo.md)에 있다.
사용자는 2026-09-09 도메인이 없어 임시 주소로 계속 진행하도록 했다.
