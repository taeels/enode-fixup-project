# rc1 배포와 로컬 실행

사용자가 2026-09-09 macOS·Windows용 rc1 게시와 로컬 Mediator·enode 기동을
지시했다. 기존 구현을 배포하는 작업이며 공동 하드웨어 장면 완료와는 구분한다.

## 배포

- [x] 기존 태그와 버전 확인: 이 저장소의 첫 릴리스는 `v0.1.0-rc1`이다.
- [x] 소스 고정: `2d7bf7524d9c9f9b25bb40d93ac56938115d934b`, PR #6.
- [x] Windows amd64 MSI·ZIP 빌드와 실제 Windows 러너 설치·실행 검증.
- [x] macOS arm64·amd64 노드 도구와 Mediator를 각각 tar.gz로 배포.
- [x] 체크섬 대조와 prerelease 게시.

[릴리스](https://github.com/taeels/enode-fixup-project/releases/tag/v0.1.0-rc1)는
2026-09-09T02:14:42Z에 게시했다. 실행 파일의 판은 `0.1.0-rc1`, 커밋은
`2d7bf7524d9c`다. 노드 묶음은 enode·runctl·enodectl이고 Mediator는 별도다.
기존 workflow가 만든 Linux x64 묶음도 포함해 설치 파일 14개와 SHA256SUMS 1개다.

[소스 CI](https://github.com/taeels/enode-fixup-project/actions/runs/34301526415)의
test·cross와 [패키지 CI](https://github.com/taeels/enode-fixup-project/actions/runs/34302170013)의
빌드·Windows 검증·Linux 네 배포판 설치가 통과했다. Mac은 같은
`packaging/build-installers.sh`에 `PKG_VERSION=0.1.0-rc1`과
`TARGETS='darwin/arm64 darwin/amd64'`를 주어 빌드했다. arm64 압축은 스크립트의
기존 amd64 묶음 루프 밖이므로 명시 파일 목록으로 별도 구성했다.
Mac amd64 묶음도 AppleDouble 부속 파일 없이 같은 방식으로 구성했다.

Mac 실행 파일 8개는 arm64 네이티브와 amd64 Rosetta 2에서 --version을 확인했다.
arm64 4개의 ad-hoc 서명, 압축 4개의 파일·권한·해시, 고정 example 2개의 lint도
통과했다. 업로드된 설치 파일 14개를 다시 받아 GitHub digest·CI 체크섬·Mac 원본과
대조했다. 전체 SHA256SUMS를 올린 뒤 게시본을 다시 받아 일치를 확인했다.
Apple 공증·Windows Authenticode 서명은 포함하지 않는다.

## 로컬 실행

- [x] PostgreSQL 16.14의 별도 `enode_runixs_rc1` DB 준비.
- [x] rc1 Mediator와 실제 Mac enode 기동, 광고 확인.
- [x] shell 작업 성공과 봉인 record·greeting.txt 내용 확인.
- [x] LAN 주소의 UI·API 접속 확인과 사용자 토큰 전달.

설정·바이너리·로그는 Git에서 제외된 `local/rc1-runtime/`에 있다. 토큰 값은
저장소와 릴리스에 싣지 않는다. `config/token`과 설정 파일은 0600이다.
Mediator는 `0.0.0.0:8080`, DB는 로컬 5432를 사용한다. 실제 UI 경로는
`/ui/`, `/ui/fleet/`, `/ui/demo/`이며 API 기준 주소의 `/` 자체는 404다.

노드 `9b2fab72b8fc`는 Mac에서 탐지한 Claude·darwin·arm64 능력을 광고한다.
`runixs-rc1-local-smoke-20260909`는 SUCCEEDED이며 봉인된 greeting.txt 내용은
`enode rc1 local runtime ok`다. 실제 하드웨어 능력을 임의 광고하거나
LED·음원·방송의 공동 게이트를 통과로 기록하지 않았다.

처음 만든 Orca 터미널들이 세션 중 닫힌 뒤, Mediator는 독립 프로세스,
enode는 enodectl start로 다시 기동했다. 터미널 수명과 분리된 상태에서
LAN의 UI 3개와 API 3개가 200이고 같은 노드·성공한 Run이 유지됨을 확인했다.
관리 명령은 `python3 local/rc1-runtime/manage.py start|stop|status`다.
상세 근거는 `local/release-v0.1.0-rc1/`와 `local/rc1-runtime/validation.json`에 있다.
