# 회사 메신저 승인 장면 자산

사용자가 제공한 `ddthon-6.jpeg`는 메신저 화면, `ddthon-7.jpeg`는 앱 로고다.
서대현님이 휴가지에서 S2 테스트 요청을 승인하는 후속 장면에 사용한다.
완성본의 23–30초에 사용했으며 앞 15초에는 삽입하지 않았다.

## 저장한 자산

- 로고 PNG (`output/enode-video/messenger-assets-v1/messenger-logo.png`): 파란색·남색 리본 형태를 살린 정면 로고. 흰 배경이며 투명 PNG는 아니다.
- 승인 요청 화면 PNG (`output/enode-video/messenger-assets-v1/messenger-approval-screen.png`): 상단 메뉴, 하늘색 채팅 배경, 청록색 봇, 메시지 카드, 하단 입력창을 살린 세로 화면.
- 생성 프롬프트 (`output/enode-video/messenger-assets-v1/prompts.json`)와 로고 배경 수정 프롬프트 (`output/enode-video/messenger-assets-v1/logo-background-fix-prompt.txt`).
- 파일 규격·해시 (`output/enode-video/messenger-assets-v1/manifest.json`).

내장 image_gen으로 제작했다. 원본의 촬영 무늬·손·반사를 정리하고 일정 내용은
영상용 요청 카드로 바꿨다. 사진 형식이 MPO로 감지되어 PNG 사본으로 변환한 뒤
참조했다. 회사 메신저 원본을 대체하는 실제 제품 화면이 아니라 영상용 시안이다.
승인 화면의 한국어 문구와 요청자·대상·작업·버튼을 눈으로 확인했다.

## 후속 승인 장면 적용

1. 휴가지의 서대현님 휴대폰으로 전환하고 익숙한 메신저 로고를 짧게 보여준다.
2. 화면을 충분히 크게 잡아 `갤럭시 S2 테스트 요청`, 요청자, 대상 PC, 작업을 읽을 시간을 준다.
3. 청록색 손가락이 오른쪽의 파란 `승인` 버튼을 한 번 누른다. 버튼 중앙은 화면 폭 약 74%, 높이 약 49% 지점이다.
4. 누른 다음에만 `승인 완료`로 바뀌고 실제 웹 UI의 승인 후 요청 전달로 연결한다.

원본 PNG는 승인 전 상태다. 후속 Kling 7초 영상에 로고와 이 화면을 합성하고,
허용 범위 `/sandbox/s2-verify`와 승인 후 상태를 편집에서 추가했다.
생성 원본의 녹색·임의 문자를 제거하고 손의 가림을 보존했다. 승인 완료는
127프레임, 약 5.292초부터 표시한다. 168프레임 화면 영역의 강한 녹색 잔여는 0픽셀이다.

사용자는 화면 수정본을 채택했다. 손가락과 승인 버튼의 위치가 다른 점은
사소하므로 무시해도 된다고 명시했다. 위 좌표는 설계 목표이며 실제 탭의 정확한
위치를 보증하지 않는다. 최종 제작·비용·검사는 [제작 요약](code/implementation-summary.md)과
[검증 기록](code/build-and-test.md)에 있다.
