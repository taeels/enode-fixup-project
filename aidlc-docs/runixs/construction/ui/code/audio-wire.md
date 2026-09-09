# 공개 사운드 연결 운영 반영

2026-09-09, `Runixs/fix-wire`. 사용자가 즉시 수정·서버 우선 반영·관련 PR
전체 병합을 지시했다. 기존 이름 주입과 서버 allow-list를 유지했다.

## 원인과 최종 구성

공개 계약은 없는 `tts=higgsfield`·`device=speaker`를 요구해 배정 전에 422로
실패했다. 성공한 수동 Run은 실제 Typecast 노드와 Mac 직결 Pi를 사용했다.
먼저 해당 음성/보드 광고로 계약을 수정하고 PR #29의 `enode-demo-play`를
운영 PATH에 설치했다. 이 버전의 공개 제출 `demo-8150200e…`가 SUCCEEDED였다.

이후 사용자 지정 PR #30을 검토하고 최종안으로 인수했다. 음성 노드는
`harness=claude, service=tts, tts_typecast=yes, voice_typecast=Sanghyun`,
보드 노드는 `board=rpi2b-v1.1`로 고른다. agent가 기존 `bin/tts-typecast.sh`로
이름이 포함된 welcome.wav를 만들고 봉인 blob을 통해 보드 단계에 전달한다.
보드 단계가 SCP로 Pi에 올려 `aplay -D plughw:1,0`으로 재생한다.
이 최종 계약은 별도 enode-demo-play 설치에 의존하지 않는다.

현재 해당 보드 광고는 macOS 노드 85ccc712ee24다. 향후 Windows에 같은 보드
광고를 추가하려면 POSIX sh/scp/ssh 실행 가능 여부도 확인해야 한다.
두 장치 기능은 같은 enode에 있으므로 기존 노드 임대로 LED·음성이 직렬 실행된다.

## 배포와 실제 확인

- 선행 수정 0253f42를 Mac mini에서 빌드·적용(PID 80169).
- 최종 PR #30 12f318c를 다시 빌드·적용(PID 81093).
- 최종 빌드/백업: `/Users/runixs/enode-audio-build.SnAhjA/pr30`.
- 직전 바이너리: 해당 디렉터리의 `previous-mediator`.
- 기존 mediator.yaml·DB·공개 터널 PID 70718과 URL을 유지했다.
- 노드에 설치한 PR #29 명령과 audio_playback 광고는 유지한다.
- 최종 공개 요청: `demo-b301d99a71ac73835e8ff044c7f158b03841e77b753dbc82c9ba5e92cf152790`.
- 실제 voice=620abcbb7e47, board=85ccc712ee24로 배정,
  synthesize·play 모두 DONE, 전체 SUCCEEDED 및 산출물 조건 성공.

사용자에게 새로고침 후 새 작업 → 사운드 재생을 직접 사용할 수 있다고 알렸다.
이 기록은 재생 명령 성공이며 사용자의 실제 청취를 대신하지 않는다.
전체 로컬 테스트는 기존 지시대로 생략했다. PR #29·#30은 GitHub CI의
test/cross/bounded-demo 전체 성공을 확인한 뒤 main에 병합했다.
PR #28은 LED 레시피와 이 운영 기록을 모으며 최종 CI 뒤 병합하도록 승인됐다.
SECURITY-08/12의 기존 이름 주입·고정 시나리오·토큰 비노출 경계를 유지한다.
