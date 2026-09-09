# Mac mini 공개 데모 실행 기록

2026-09-09. 사용자의 “한번 해봐”로 Mac mini 공개 실행을 구성했다.
공식 단계는 Construction의 Code Generation이고 공동 실제 장면 CP10은 남아 있다.

## 실행 상태

실행 호스트는 `Runixsui-Macmini.local`(arm64), 경로는
`/Users/runixs/enode-public-demo`다. 기존 SSH `MacMini` 별칭으로 관리한다.
MacBook의 앞서 종료한 Mediator·노드·데모 제어판·전용 DB는 중지 상태를 유지했다.

현재 공개 주소는 https://deutsche-football-tract-necklace.trycloudflare.com 이다.
`/`는 `/ui/`로 이동하며 게스트 데모는 `/ui/demo/`, 실 함대는 `/ui/fleet/`다.
방문자는 별도 앱·VPN·API 토큰 없이 데모 화면에 접속할 수 있다.
Quick Tunnel의 자동 생성 주소다. 사용자는 도메인이 없으며 그대로 진행하도록 했다.
Tailscale 이름과 기존 태양님 공개 서버의 DNS는 변경하지 않았다.

## 구성

| 항목 | 배치·동작 |
|---|---|
| 제품 바이너리 | `166c935b7795`, macOS arm64, `main@bc41f26`과 제품 코드 동일 |
| Mediator | `127.0.0.1:8080`, demo 모드, claim long poll 30초 |
| PostgreSQL | 기존 16 서비스 안의 별도 DB `enode_runixs_public_20260909`, 전용 역할 |
| 실제 노드 | `3ae3a256ec68`, `runixs92@Runixsui-Macmini.local:demo` |
| 작업 공간 | 전용 `workspace/` Git 저장소 |
| 공개 경로 | cloudflared 2026.8.3 Quick Tunnel에서 loopback으로 연결 |
| 비밀 값 | 새 API 토큰·DB 암호, Mac mini의 0600 설정 파일에만 저장 |

실제 노드가 광고한 `agent.reason`과 해당 호스트 속성만 사용했다. 하드웨어 광고를
만들지 않았으며 shell 검증으로 유료 에이전트 호출을 실행하지 않았다.
공개 연결 검사 후 자체 노드는 loopback 연결로 복구했다. 다른 PC의 노드는 현재
공개 HTTPS 주소와 **이 Mac mini 환경의 토큰**으로 접속한다. 태양님 서버와는
DB·토큰·함대가 다른 별도 데모다.

## 확인한 결과

| 검사 | 결과 |
|---|---|
| 내부 실제 shell Run | `macmini-public-smoke-2577b230` SUCCEEDED, 봉인 파일 내용 확인 |
| 공개 시작·데모 화면 | 루트 302, `/ui/`·`/ui/demo/` 200, 브라우저 실제 노드·Run 표시 |
| 공개 조회·인증 | nodes/runs 200, capabilities 무인증 401·새 토큰 200 |
| 공개 등록 POST | 빈 node_id로 인증 후 400 확인, POST가 GET으로 바뀌지 않음 |
| 공개 쓰기 보호 | 일반 runs 무인증 401, 잘못된 demo 요청 400 |
| 실제 노드의 공개 등록·claim | 동일 실제 노드를 공개 HTTPS에 재연결해 새 instance 확인 |
| idle 이후 공개 실행 | 35초 대기 후 MacBook에서 제출, `macmini-public-e2e-4900838a` SUCCEEDED |
| 공개 봉인 다운로드 | `public-proof.txt` 내용 `public https to mac mini ok` 확인 |
| 복구 | 자체 노드를 loopback으로 재시작, Mediator·터널 계속 실행 확인 |

로컬 증거는 무시 경로 `local/macmini-public-20260909/`의 `http-check.json`,
`public-e2e.json`, `public-e2e-record.tar`, `binary-sha256.json`에 있다.
바이너리의 로컬·원격 SHA256을 대조했다. 제품 소스를 바꾸지 않았으므로 기존 CI와
이번 실제 배치 검증을 근거로 삼으며 전체 Go 검사를 다시 실행하지 않았다.
rc1 릴리스 태그와 배포 자산은 변경하지 않았다.

## 관리와 남은 조건

Mac mini 실행 디렉터리에 `README.md`와 `manage.py`를 배치했다.

```sh
ssh MacMini '/opt/homebrew/bin/python3 /Users/runixs/enode-public-demo/manage.py status'
ssh MacMini '/opt/homebrew/bin/python3 /Users/runixs/enode-public-demo/manage.py start'
ssh MacMini '/opt/homebrew/bin/python3 /Users/runixs/enode-public-demo/manage.py stop'
```

`stop`은 이 환경의 터널·노드·Mediator만 종료하고 DB·파일을 보존한다. SSH 종료 후
유지되지만 재부팅 자동 기동은 미설정이다. 최신 URL은 원격 `public-url.txt`,
노드 등록 토큰은 원격 `config/token`에 있다. 설정·토큰은 Git에 포함하지 않는다.

Quick Tunnel 재생성 시 주소가 바뀔 수 있다. 사용자 지정 주소는 도메인이 준비된
뒤 별도로 연결한다. 임시 서비스의 한도는 [Cloudflare 공식 문서](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/trycloudflare/)를 따른다.

공동 LED·음원·방송 CP10은 실제 장비와 공개 webcam 입력이 필요하다. 이번 shell
성공을 그 장면의 완료로 기록하지 않는다. 앞서 보고된 태양님 서버의 다른 PC 등록
문제도 해당 PC의 HTTPS 변경·재시작 결과는 여전히 미수령이다.
