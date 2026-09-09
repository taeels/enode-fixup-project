# LED 연결 운영 보완

사용자가 테스트 스위트 생략과 즉시 연결·Mac mini SSH 실검증을 지시했다.
기존 ui/demo-back 승인과 고정 LED 계약을 이어받는 운영 보완이다.
현재 담당 topic branch `Runixs/fix-wire`에서 작업하며 공동 CP10은 별도 유지한다.

- [x] 1. pi-stage 레시피·기존 계약과 실제 Mac mini mediator·Pi 연결 확인.
- [x] 2. POSIX SSH LED 명령 제공. heartbeat/on 계약을 유지하고 쓰기값을 읽어 검증.
- [x] 3. 이 Mac에 enode/runctl·SSH 터널·자동 재시작 구성을 설치하고 Pi 인증 연결.
- [x] 4. runctl 및 공개 LED 제출을 실제 실행하고 Mac mini Run 상태·Pi 읽기값 확인.
- [x] 5. 결과·운영 명령·남은 제한을 담당 기록에 남김.

전체 자동 테스트와 DB 장면 게이트는 사용자 요청으로 생략한다.
기존 mediator 바이너리·API·계약은 변경하지 않는다. 토큰은 private 운영 디렉터리에서만 읽는다.
