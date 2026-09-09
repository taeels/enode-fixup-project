# 이 Mac의 Raspberry Pi LED 연결

2026-09-09, `Runixs/fix-wire`. 기존 승인된 ui/demo-back의 운영 보완이다.

원인은 기존 공개 계약의 `device=led`를 광고하는 실행 노드가 없는 것이다.
기존 Windows 보드 광고에는 다른 속성만 있었고, Pi를 이 Mac에 직결한 현재
배치에서는 이 Mac에 POSIX LED 명령과 노드를 제공하는 것으로 해결했다.
mediator/API/예제 계약을 변경하거나 서버를 재시작하지 않았다.

- 운영 루트: `/Users/runixs/enode-led-bridge` (0700).
- runctl: `/Users/runixs/enode-led-bridge/runctl`.
- 제어 명령: `/Users/runixs/enode-led-bridge/bin/enode-demo-led`.
- Pi: `ssh sunny`, 실제 대상 `sunny@sunnypi.local`, 사용자 sunny.
- 실행 노드: `85ccc712ee24`, `runixs92@Runixsui-MacBookPro.local:led`.
- 연결: 이 Mac `127.0.0.1:18080`에서 SSH로 Mac mini `127.0.0.1:8080`.
- 자동 재시작: LaunchAgents `com.runixs.enode-led-tunnel`, `com.runixs.enode-led-node`.
- 로그: 운영 루트의 `logs/node.log`, `logs/tunnel.log`.
- 인증: Pi에 기존 Mac 공개키만 추가. 비밀번호를 파일에 저장하지 않았다.
  mediator 토큰은 운영 루트의 0600 파일에서 환경변수로 읽는다.

노드는 시작 전에 Pi SSH와 두 LED의 sysfs 쓰기 권한을 확인한다.
명령은 온보드 ACT·PWR을 쓰며 매번 쓰기값을 읽어 검증한다.
heartbeat는 기본 10초·2 Hz이며 종료/오류 시 기본 트리거로 복원한다.
계약의 persistent 단계는 두 LED를 켜 둔다. 직접 제어와 계약 실행을 동시에
하면 경합할 수 있으므로 수동 명령은 해당 노드의 작업이 없을 때 사용한다.

## 실제 확인

`led-bridge-20260909-0736`은 heartbeat·persistent 모두 DONE, 전체 SUCCEEDED다.
Mac mini에 SSH로 접속해 동일한 서버 상태와 산출물 조건 4개 성공을 확인했다.
Pi에서 ACT·PWR 각각 `trigger=none brightness=255`를 읽었다.
사용자가 실제 점멸을 세 차례 확인했다.
봉인 기록은 `/Users/runixs/enode-led-bridge/verified-record.tar`에 저장했다.

공개 제출 라우트 `/v1/demo/runs`로 접수한
`demo-f9929b15a7ac3f1ca317262c7fddbcaab70469a6ccf776251aefb86ef54861f1`도
앞선 LED 작업 뒤 QUEUED에서 승격되어 두 단계 모두 SUCCEEDED다.
기존 웹 계약을 변경하지 않고 실제 API 제출부터 하드웨어 실행·봉인까지 확인했다.

enode/runctl은 Mac mini의 운영 바이너리를 복사하고 양쪽 SHA256 일치를 확인했다.
셸 구문·LaunchAgent plist·계약 lint를 확인했다. 전체 테스트 스위트·DB 게이트는
사용자 지시로 생략했다. 이 결과는 음성 시나리오나 공동 CP10 전체 통과가 아니다.

## 운영

```sh
/Users/runixs/enode-led-bridge/runctl capabilities
/Users/runixs/enode-led-bridge/runctl status led-bridge-20260909-0736
/Users/runixs/enode-led-bridge/bin/enode-demo-led status
```

웹의 LED Toggle이 같은 노드를 사용한다. Pi 랜선과 이 Mac의 실행 상태가
필요하다. `restore`는 ACT=mmc0·PWR=input 기본 표시로 돌린다.
자동 기동을 중지하려면 해당 사용자 LaunchAgent 두 개를 `launchctl bootout`한다.
