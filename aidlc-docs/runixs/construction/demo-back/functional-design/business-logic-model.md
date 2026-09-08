# demo-back 업무 흐름

**상태**: 검토안. `Config.Demo`와 obs의 기존 submitter 연결을 재사용한다.

1. 데모 모드일 때만 POST 라우트를 등록한다. 전용 요청 한도를 먼저 적용한다.
2. 본문 크기·Content-Type·단일 JSON·세 필드의 형식을 검증한다.
3. scenario_id가 고정 매핑에 없으면 400, 매핑의 실제 픽스처가 준비되지 않았으면
   503으로 거절한다. 문자열을 파일 경로로 직접 사용하지 않는다.
4. [제출 계약](submission-contract.md)의 식별을 계산하고 고정 example의 독립
   사본을 준비한다. run_id·manual Work 식별과 승인된 이름 위치만 치환한다.
5. 같은 submitter를 이름 입력과 컨텍스트에 넣는다. Contract.Validate로 결과를
   확인한다. 텍스트 전체 검색 치환이나 사용자 문자열의 셸 결합을 하지 않는다.
6. 브라우저 요청을 그대로 고치지 않고 내부 요청을 복제한다. 본문은 준비한
   Contract JSON, Authorization은 서버 cfg.Token, submitterKey는 검증한 이름으로
   설정한다. 브라우저가 보낸 Authorization·X-Enode-Principal은 전달하지 않는다.
   Content-Type·ContentLength도 새 JSON과 일치시킨다. 내부 본문은 호출 뒤 닫고
   원래 요청 헤더를 변경하지 않는다.
7. 내부 HTTP 네트워크 요청 없이 기존 `s.auth(s.postRuns)`를 호출한다. 같은 서버의
   submit·DB 트랜잭션·queue 접수를 재사용하며 응답을 그대로 돌려준다.
8. 에러에서도 요청 본문·DB 작업의 기존 정리 경로를 유지한다. 이름과 토큰을
   로그에 넣지 않는다. 실제 하드웨어 동작은 접수된 계약을 실행하는 노드가 맡는다.

같은 요청의 동시 접수는 같은 run_id를 만들며 기존 트랜잭션과 중복 처리로 모인다.
이 성질은 queue가 병합된 코드로 다시 검증한다. 한 요청 안에서 임의 계약을
추가 제출하거나 서버가 큐 승격·LED 상태를 직접 계산하지 않는다.

최초 접수 뒤 응답 유실은 [제출 계약](submission-contract.md)의 같은 요청 재시도로
복구한다. 실패한 고정 계약을 다른 계약으로 자동 바꾸지 않는다. 허용 목록·이름
검증 실패는 submit을 호출하기 전에 끝낸다.
