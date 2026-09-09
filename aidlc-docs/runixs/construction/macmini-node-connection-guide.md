# 태양님 PC를 runixs Mac mini에 연결하기

2026-09-09 기준. Mediator와 DB는 runixs Mac mini에서 실행 중입니다.
태양님 PC에는 enode 노드만 실행하면 됩니다. Tailscale 설치나 공유기 설정은 필요 없습니다.

## 기존 노드의 연결 주소 변경

아래는 노드 설정 이름이 `local`인 예입니다. 이름이 다르면 명령의 `local`을
실제 이름으로 바꾸세요. 설정 이름과 디렉터리는 다음 명령으로 확인합니다.

```sh
enodectl list
enodectl stop local
```

표시된 설정 디렉터리의 `local.yaml`에서 다음 두 항목만 바꿉니다.

```yaml
mediator: https://deutsche-football-tract-necklace.trycloudflare.com
token: <RUNIXS_MACMINI_TOKEN>
```

토큰은 runixs가 전달한 **Mac mini 서버의 새 토큰**입니다. 기존
`mediator.taeels.duckdns.org` 서버의 토큰과 다릅니다. 주소에는 HTTPS를 사용하고
`/ui/`나 `:8080`을 붙이지 않습니다.

기존 `principal`, `workspace`, `board`, `labels` 등은 유지하세요. 설정 파일을
다른 경로로 옮기면 node id가 달라집니다. 최초 실행 PC에 principal과 전역 Git
이메일이 모두 없으면 아래 항목도 실제 본인 이메일로 추가합니다.

```yaml
principal: <태양님 본인 이메일>
```

설정을 저장한 뒤 시작합니다.

```sh
enodectl start local
enodectl status
enodectl logs local
```

Windows에서 명령이 PATH에 없다면 바이너리 폴더의 PowerShell에서
`enodectl` 대신 `./enodectl.exe`를 사용합니다. 예:

```powershell
./enodectl.exe start local
./enodectl.exe status
./enodectl.exe logs local
```

## 처음 설치하는 PC

[rc1 다운로드](https://github.com/taeels/enode-fixup-project/releases/tag/v0.1.0-rc1)에서
노드 패키지를 받습니다. Apple Silicon Mac은 `enode-0.1.0-rc1-darwin-arm64.tar.gz`,
Intel Mac은 `enode-0.1.0-rc1-darwin-amd64.tar.gz`, Windows x64는
`enode-0.1.0-rc1-windows-amd64.zip`입니다. `enode-mediator` 패키지는 필요 없습니다.
`enode`와 `enodectl`을 같은 폴더에 둡니다.

기존 설정이 없는 이름으로 대화형 설정을 실행합니다. 압축을 푼 폴더에서 Mac은
`./enodectl`, Windows PowerShell은 `./enodectl.exe`로 실행합니다.

```sh
./enodectl setup local --mediator https://deutsche-football-tract-necklace.trycloudflare.com
```

주소는 Enter로 유지하고 위 토큰을 입력합니다. workspace에는 작업할 실제 checkout
경로를 넣습니다. 추론만 제공할 경우 비워둘 수 있지만 설치·인증된 하네스가 필요합니다.
이메일을 물으면 본인 이메일을 입력하고 장비 항목은 실제 연결된 값만 넣습니다.
`token accepted`와 node id를 확인한 뒤 저장하고 `./enodectl start local`을 실행합니다.
setup만 실행하면 설정을 준비한 상태이며 start부터 실제 노드를 등록합니다.

## 연결 확인

[실 함대 화면](https://deutsche-football-tract-necklace.trycloudflare.com/ui/fleet/)에서
태양님 PC의 노드가 나타나는지 확인합니다. `enodectl id local`로 로컬 node id와
화면을 대조할 수 있습니다. 현재 이 함대에는 Mac mini의 실제 노드가 있습니다.
태양님 기존 서버의 함대와는 별개입니다.

`running`은 로컬 프로세스 상태입니다. 실제 연결 성공은 함대 표시와 로그로 확인합니다.

| 증상 | 확인할 것 |
|---|---|
| 401 또는 token rejected | Mac mini의 새 토큰인지 확인 |
| 404 | 설정 주소에 `/ui/`가 붙지 않았는지 확인 |
| cannot derive node identity / no identity | principal 또는 전역 Git 이메일 설정 |
| running인데 함대에 없음 | HTTPS 주소·토큰·logs 확인 |
| 함대에 있지만 작업에 매칭되지 않음 | 실제 workspace·하네스·장비 능력 광고 확인 |
| 터널 연결 오류 | runixs에게 서버와 현재 임시 주소 확인 요청 |

Quick Tunnel을 재생성하면 주소가 바뀔 수 있습니다. 그 경우 새 주소를 받아
기존 파일에서 mediator만 바꾸고 같은 노드를 재시작합니다.
연결 실패 시 `enodectl id local`과 `enodectl logs local`의 결과를 전달해 주세요.
