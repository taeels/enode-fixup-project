# Mac mini Bedrock VM 실행 기록

2026-09-09. 사용자 요청으로 기존 `ccb`의 Bedrock 연결을 enode에 적용했다.
개인 Mac에서 실행하던 노드를 종료하고 별도 Linux VM으로 전환했다.
공식 AI-DLC 단계는 Construction → Code Generation이며 공동 CP10은 남아 있다.
제품의 OS sandbox 기능 추가가 아닌, 기존 `harness_bin`을 이용한 운영 구성이다.

## 실행 구성

| 항목 | 적용 값 |
|---|---|
| VM | Lima 2.2.0, VZ, Ubuntu 24.04 arm64, 2 CPU·4 GiB RAM·24 GiB 디스크 |
| 노드 | `95262f8783e9`, `runixs92@lima-enode-bedrock:bedrock` |
| enode | `166c935b7795` 소스의 Linux arm64 빌드 |
| Claude Code | 공식 native 2.1.228, 배포 manifest의 SHA256 대조 |
| Bedrock | `ap-northeast-2`, `global.anthropic.claude-haiku-4-5-20251001-v1:0` |
| 작업 사용자 | VM의 `enode-worker`, UID 1001, sudo 권한 없음 |
| 작업 공간 | `/var/lib/enode-worker/workspace`, 별도 Git 저장소 |
| 하네스 | `/opt/enode/claude-bedrock` → `/opt/enode/claude` |
| 광고 | `agent.reason`: `instance=macmini-bedrock-vm`, `provider=bedrock`, `sandbox=lima-vm` |

`ccb` 전체나 사용자 셸 설정을 실행하지 않는다. 필요한 Bedrock 환경만 래퍼에서
읽어 enode의 환경 필터 뒤에 적용한다. 새 키를 Mac mini의 `ccb`와 VM 설정에
반영했고, 다른 Claude 공급자 설정은 래퍼에서 제거한다.

VM은 plain 모드이며 호스트 파일 공유, SSH agent 전달, 호스트 공개 키 자동 반영,
게스트 에이전트와 자동 포트 전달이 없다. 기존 다른 Lima VM은 유지했다.
enode systemd 서비스는 권한 상승 금지, Linux capability 제거, 시스템 경로 읽기
전용, 홈 디렉터리 보호, 개별 임시 디렉터리·장치 제한과 자원 한도를 적용한다.

작업 UID의 외부 연결은 nftables로 차단하고 로컬 HTTPS CONNECT 프록시만 허용한다.
프록시는 현재 Mediator 호스트와 해당 리전 Bedrock 두 호스트의 443 연결만 받는다.
목적지 DNS 결과도 전역 IP인지 검사한다. 호스트 사설망과 일반 인터넷에 직접
연결하는 기능은 이 노드에서 사용할 수 없다.

## 검증 결과

- 실제 서비스와 같은 권한 설정에서 격리 검사 15개 통과: 일반 UID, 권한 상승
  금지, capability 없음, 호스트 공유·홈 없음, VM 관리자 홈 비공개, 시스템 파일
  쓰기 차단, 작업 공간 쓰기 허용, sudo 차단, 사설망·직접 인터넷 차단, 미허용
  프록시 목적지 차단, Mediator HTTPS, Bedrock 인증 설정·공급자 확인.
- 처음 발견한 기존 키는 실제 호출에서 AWS 403을 반환했다. 로컬 `auth status`는
  키 유효성 검사가 아니므로 이 결과만으로 성공 처리하지 않았다. 자체 첫 시험은
  취소했고, 사용자가 제공한 새 키로 다시 실제 실행했다.
- `bedrock-vm-proof-b8f111c2`가 `SUCCEEDED`. 첫 shell 단계에서 실제 작업의 UID,
  권한 상승 금지, 호스트 경로 없음, 시스템 쓰기·사설망 차단을 확인했다.
  다음 단계에서 enode가 Bedrock Claude를 실행해 `bedrock-proof.txt`를 만들었다.
  Mediator에서 봉인 record를 내려받아 내용이 `BEDROCK_VM_OK\n`인지 대조했다.
- 기존 Mac 직접 실행 노드 `3ae3a256ec68`를 종료했다. VM 서비스는 부팅 시 실행을
  활성화했고 공개 런타임 관리 명령도 VM을 사용하도록 전환했다.
- 전환 후 VM과 팀원 두 노드의 heartbeat 갱신·lease 해제, 기존 호스트 노드의
  광고 만료·프로세스 종료, 공개 UI 세 경로의 HTTP 200을 확인했다. Mediator와
  터널의 프로세스 및 공개 주소는 유지됐다.

증거는 Git 제외 경로 `local/macmini-bedrock-20260909/`의 `proof-contract.json`,
`proof-result.json`, `proof-record.tar`, `final-health.json`에 있다. 운영 스크립트는
Mac mini 전용 실행 디렉터리에 배치했다. 제품 코드와 rc1 자산은 변경하지 않았다.
Python 문법 검사와 실제 운영 경로를 검증했으며 전체 Go 검사는 다시 실행하지 않았다.

## 관리

MacBook에서는 다음 SSH 명령을 사용한다. Mac mini 터미널에서는 SSH 부분을 뺀다.

```sh
ssh MacMini '/opt/homebrew/bin/python3 /Users/runixs/enode-public-demo/manage.py status'
ssh MacMini '/opt/homebrew/bin/python3 /Users/runixs/enode-public-demo/manage.py start'
ssh MacMini '/opt/homebrew/bin/python3 /Users/runixs/enode-public-demo/manage.py stop'
```

전체 `start`는 Mediator·터널·VM 노드를 시작하고 현재 주소와 노드 등록 토큰을 VM에
반영한다. 이미 실행 중이면 유지한다. 전체 `stop`은 터널·VM·Mediator를 종료하고
DB와 파일을 보존한다. Mac mini 재부팅 후 전체 자동 기동은 설정하지 않았다.
VM 안의 서비스 자동 기동과 Mac 자체의 자동 기동은 별개다.

VM 작업만 중지하거나 재개할 때는 아래 명령을 쓴다. 실행 중인 작업을 먼저
확인한다. VM `stop`은 실행 작업도 중단하며 디스크는 보존한다.

```sh
ssh MacMini '/opt/homebrew/bin/python3 /Users/runixs/enode-bedrock-sandbox/manage.py status'
ssh MacMini '/opt/homebrew/bin/python3 /Users/runixs/enode-bedrock-sandbox/manage.py stop'
ssh MacMini '/opt/homebrew/bin/python3 /Users/runixs/enode-bedrock-sandbox/manage.py start'
```

키를 다시 갱신할 때는 Mac mini의 `.zshrc`에 있는 `ccb` 키를 편집한 뒤 실행한다.
키 값은 명령 인자에 넣지 않는다. VM이 실행 중이어야 한다.

```sh
ssh MacMini '/opt/homebrew/bin/python3 /Users/runixs/enode-bedrock-sandbox/sync-key.py'
```

이 명령은 키만 표준 입력으로 VM에 전달해 원자적으로 저장한다. 다음 Claude
실행부터 적용하며 진행 중인 작업을 재시작하지 않는다. 기존 셸에서 `ccb`를 직접
사용할 때는 `.zshrc`를 다시 읽어 갱신된 함수를 로드한다. 초기 설치용
`configure-vm.py`는 키 갱신용으로 재실행하지 않는다.

VM 서비스 로그는 `journalctl -u enode-bedrock`, 프록시 로그는
`journalctl -u enode-egress`로 확인한다. 공개 터널의 최신 주소는 공개 런타임의
`public-url.txt`다. 터널을 재생성하면 주소가 바뀔 수 있다.

## 적용 범위와 한계

Bedrock 키는 VM의 root 소유·worker 그룹 읽기 전용 0640 파일에 있고, `ccb`가 있는
호스트 `.zshrc`는 0600이다. 키와 노드 등록 토큰은 Git에 저장하지 않았다.
Claude는 Bedrock을 호출하기 위해 키를 읽을 수 있다. 이 VM은 작업마다 새로 만드는
구성이 아니므로 같은 worker의 다른 작업·상태를 서로 격리하지 않는다.

공유 Mediator 토큰의 등록·작업 제출 권한은 그대로다. 프록시는 CONNECT 호스트와
목적지 IP를 제한하지만 TLS 내부 SNI·HTTP 경로까지 검사하지 않는다. 허용된
서비스로의 데이터 전송이나 Bedrock 비용을 별도로 통제하는 구성은 아니다.
검증한 것은 현재 VM 작업의 호스트 접근·권한·네트워크 제한이며 모든 VM 탈출이나
모든 악성 작업에 대한 안전 보장은 아니다. 공동 LED·음원·방송 CP10은 미완료다.

공급자 설정은 [Claude Code Bedrock 문서](https://code.claude.com/docs/en/amazon-bedrock),
VM 설정은 [Lima 구성 문서](https://lima-vm.io/docs/config/)와
[보안 문서](https://lima-vm.io/docs/security/)를 확인했다.
