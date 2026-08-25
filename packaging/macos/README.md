# 맥에 enode 를 세운다

이 묶음은 **CT103(리눅스)에서 크로스 빌드된 것**이다. 맥에는 Go 도 저장소도
필요 없다 — `ADR-015` 가 Go 를 고른 이유(리눅스 CI 에서 다른 OS 실행파일이
나온다)를 그대로 쓰는 것이고, `.github/workflows/ci.yml` 의 `cross` 잡이
`darwin/arm64` 를 이미 매 커밋 검증하고 있다.

```text
   CT103                                    맥
   ├ mediator  :8080   ◀──── /v1/nodes ───── enode (zephyr)
   ├ postgres                              └ enode (qemu)
   ├ enode                     ▲
   └ runctl ────────────────────┘           colima VM
                                            └ enode (linux/arm64)
```

## 묶음 안에 무엇이 있나

```text
   bin/enode-darwin-arm64    ┐ 맥에 설치되는 것 install.sh 가 골라 넣는다
   bin/enode-darwin-amd64    │
   bin/runctl-darwin-arm64   │
   bin/runctl-darwin-amd64   ┘
   bin/enode-linux-arm64     ┐ colima VM 안에 넣을 것 (2차 시나리오)
   bin/enode-linux-amd64     │ colima 는 맥 위의 리눅스 VM 이라 darwin 이 아니다
   bin/runctl-linux-*        ┘
   install.sh                프리플라이트 + 설치. 아무것도 기동하지 않는다
   enodectl                  인스턴스 관리 (list · id · start · stop · logs · status)
   examples/*.yaml           설정 = 노드. 넷은 각각 시나리오 한 단계다
   MANIFEST                  어느 커밋으로 구웠나
```

## ① 맥으로 옮긴다

**맥에서 당겨간다.** CT103 의 sshd 는 22 가 아니라 **10322** 에 있다 —
기본 포트로 시도하면 연결이 거절되고, 그때 사람은 네트워크를 의심한다.

```console
# 맥에서
$ scp -P 10322 sunny@192.168.219.203:~/enode/dist/enode-macos-\*.tar.gz ~/
$ tar xzf ~/enode-macos-*.tar.gz && cd enode-macos-*
$ ./install.sh
```

맥에 원격 로그인이 켜져 있다면 CT103 에서 밀어도 된다
(`scp dist/enode-macos-*.tar.gz 맥:~/`).

`install.sh` 는 `~/.local/bin` 에 넣는다 — **sudo 가 필요 없다.**
`PREFIX=/usr/local ./install.sh` 로 바꿀 수 있다.

### install.sh 가 재는 네 가지 — **여기서 안 걸리면 맥에서 조용히 실패한다**

```text
   ① git --global user.email   없으면 enode 가 그 자리에서 죽는다
                               (ADR-015 §1 — 조용한 대체를 안 한다)

   ② HostName 고정              맥 고유의 함정이다. node_id 가
                               hash(email ∥ hostname ∥ realpath(config)) 인데
                               맥은 HostName 을 안 박으면 Bonjour·DHCP 로 흔들린다.
                               카페에 가면 같은 기계가 다른 노드가 된다.
                                  sudo scutil --set HostName my-mac

   ③ PATH 의 claude            없으면 harness 속성이 광고에서 빠지고,
                               에이전트를 요구한 계약은 이 노드를 못 고른다(422).

   ④ 시스템 잠자기              디스플레이가 꺼지는 것은 상관없다.
                               시스템이 자면 광고가 멈추고 그 노드를 쥔 Run 이 죽는다.
                               enodectl start 가 caffeinate 를 함께 띄운다.

   ⑤ 크로스 툴체인             자동 탐지는 arm-linux-gnueabihf-gcc ·
                               aarch64-linux-gnu-gcc 둘만 안다.
                               Zephyr SDK 는 그 목록에 없다 → 설정에 arch: 를 적는다.
```

### **잠자기가 함대를 끊는다**

```text
   디스플레이 잠자기   화면만 꺼진다. enode 는 계속 돈다
   시스템 잠자기       CPU·네트워크가 멈춘다
        ▼
   광고가 끊긴다 → not_after(갱신 60초 × 3 = 180초)가 지난다
        ▼
   그 노드를 쥔 Run 이 「임대 만료로 회수」된다 — 실측에서 밟았다.
     승인을 기다리던 Run 이 맥이 조용해진 지 정확히 180초 만에 FAILED 가 됐다
```

`enodectl start` 가 `caffeinate -i -s -w <enode pid>` 를 함께 띄운다 —
**enode 의 수명에 묶여서** 껐다 잊는 일이 없고 유령이 안 남는다.
`-d` 는 안 준다: **화면은 꺼져도 된다.**

`enodectl list` 에 `☕` 가 보이면 붙어 있는 것이다.

영구 설정이 필요하면 사람이 한다 — **`sudo` 를 쓰는 일이라 도구가 대신하지 않는다.**

```text
   sudo pmset -c sleep 0            AC 전원일 때 시스템 잠자기 안 함
   sudo pmset -c displaysleep 10    화면은 10분 뒤 꺼도 된다
   sudo pmset -a tcpkeepalive 1     0 이면 잠자기 중 네트워크가 통째로 끊긴다
   pmset -g assertions              지금 무엇이 잠자기를 막고 있나
```

## ② 노드를 정의한다 — 설정 파일이 곧 신원이다

`~/.config/enode/<이름>.yaml` 하나가 노드 하나다. **여럿이 기본이다.**

`install.sh` 는 예시를 `~/.config/enode/examples/` 에만 풀어둔다 —
**노드를 세우는 것은 고르는 일**이지 설치의 부수효과가 아니다.

```console
$ cp ~/.config/enode/examples/zephyr.yaml ~/.config/enode/zephyr.yaml
$ vi ~/.config/enode/zephyr.yaml     # mediator · token · workspace 를 채운다
$ enodectl id zephyr                 # 띄우기 전에 node_id 를 미리 안다
node_id  3f9a1c22b8de
label    taeels@my-mac:zephyr
config   /Users/taeels/.config/enode/zephyr.yaml
```

`enodectl id` 가 있는 이유 — `runctl capabilities` 는 해시로만 노드를 보여준다.
미리 계산해두지 않으면 **함대 목록의 어느 줄이 이 맥인지 짚을 수 없다.**

## ③ 띄운다

```console
$ enodectl start zephyr
떴다        zephyr  node=3f9a1c22b8de  pid=41822
  로그: /Users/taeels/.local/state/enode/zephyr.log

$ enodectl list
  zephyr    3f9a1c22b8de   taeels@my-mac:zephyr    돈다 pid=41822
  qemu      7c02aa19f4b1   taeels@my-mac:qemu      멈춤
```

**launchd 를 기본으로 안 쓴다.** launchd 의 `PATH` 는 최소 집합이라 `claude` 를
못 찾고, 그러면 harness 가 **광고에서 조용히 빠진다** — 원인이 어디에도 안 남는
실패다. `enodectl` 은 로그인 셸의 환경을 그대로 물려준다.
launchd 로 상시화하려면 plist 에 `PATH` 를 **명시**해야 한다.

**중복 기동은 막을 필요가 없다** — `enode` 가 설정 파일 옆에 `flock` 을 잡고
두 번째를 거절한다. `enodectl` 은 그 잠금 파일의 pid 를 읽을 뿐 별도 장부를
두지 않는다 (장부를 두면 그 장부가 진실과 갈라진다).

## ④ CT103 에서 확인한다

```console
$ runctl capabilities
agent.reason  nodes: 3
  arch       arm-zephyr-eabi      ← 맥이 보인다
  board      qemu_cortex_m3
  harness    claude
  repo       github.com/zephyrproject-rtos/zephyr
  tag        qemu-mac
```

여기 안 보이면 순서대로 본다: `enodectl logs zephyr` → 맥에서
`curl -sf http://192.168.219.203:8080/v1/...` 로 도달성 → CT103 방화벽.

### CT103 쪽 준비물

```text
   □ postgres        scripts/testdb.sh 는 테스트용이다. 실측용은 따로 띄운다
   □ mediator 설정   ~/.config/enode-mediator/config.yaml  (0600)
                     token · database.url · artifacts.root
   □ listen 이 :8080 이어야 한다        127.0.0.1:8080 이면 맥에서 못 닿는다
                     (기본값 ":8080" 이 이미 전 인터페이스다 — 바꾸지 말 것)
   □ 방화벽          192.168.219.0/24 에서 8080 인바운드
```

## 시나리오와 설정의 대응

```text
   examples/local.yaml    하네스만 든 평범한 노드 — 맨 처음 왕복을 재볼 때
   examples/zephyr.yaml   1차 Zephyr 빌드 노드. arch 를 손으로 적는 이유가 안에 있다
   examples/colima.yaml   2차 VM 안에 놓는다. 실행파일이 linux/arm64 다
   examples/qemu.yaml     추가 QEMU 를 board 로 광고한다 — 코드 수정 0 개
```

`qemu.yaml` 이 코드 수정 없이 되는 이유: `board` 는 원래 *"포트에 무엇이
달렸는지는 기계가 모른다"* 라서 사람이 적는 자리다(`ADR-012`). QEMU 도 똑같이
**노드 주인의 선언**이다. `tag: qemu-mac` 으로 실물과 갈라두면, 나중에 실보드가
와도 tag 만 다른 노드가 하나 더 서고 **계약은 tag 로 지명한다.**

## 실측으로 확인된 것 (2026-08-22, CT103)

맥에 넣기 전에 **묶음의 리눅스 실행파일로 CT103 에서 왕복을 한 번 돌렸다.**
맥에서 처음 밟을 자리를 여기서 먼저 밟아보기 위해서다.

```text
   ○ 크로스 빌드 darwin/arm64 에 ad-hoc 서명이 붙는다
     LC_CODE_SIGNATURE · 76,274 바이트. 리눅스에서 구워도 애플 실리콘이 받는다.
     → 맥에서 codesign 을 따로 안 돌려도 된다

   ○ enodectl 의 node_id 계산이 Derive() 와 정확히 일치한다
     GO   deriveFrom(email, host, config) = a331f5876d57
     셸   printf '%s\0%s\0%s' | sha256 | cut -c1-12 = a331f5876d57

   ○ 광고에 다섯 속성이 전부 실린다
     arch=arm-zephyr-eabi  board=qemu_cortex_m3  harness=claude
     repo=github.com/taeels/enode  tag=qemu-probe
     → arch 를 손으로 적는 것과 board 로 QEMU 를 세우는 것이 둘 다 동작한다

   ○ runctl capabilities 가 그 노드를 보여준다 (agent.reason  nodes: 1)

   ○ 중복 기동을 flock 이 거절한다 · SIGTERM 으로 스스로 정리하고 끝난다

   ○ 광고 주기를 Mediator 가 덮어쓴다
     was=15s now=1m0s — enodectl start 에 --every 를 줘도 서버 값이 이긴다
     (ADR-028 — 만료를 계산하는 쪽이 주기를 말한다)
```

**아직 안 재본 것** — 맥 실물(HostName 흔들림 · Zephyr SDK 탐지 · 로제타 감지),
colima VM 안의 enode, 실제 Run 왕복.

## 다시 굽기

```console
$ packaging/macos/build.sh                        # CT103 에서
$ TARGETS="darwin/arm64" packaging/macos/build.sh # 하나만
```

빌드 끝에 **서명을 잰다.** 애플 실리콘 커널은 서명 없는 실행파일을 `killed: 9`
한 줄로 죽이고 이유를 안 남긴다. Go 내부 링커가 크로스 빌드에서도 ad-hoc 서명을
붙이지만 **그건 툴체인의 성질이지 우리 계약이 아니다** — 깨지면 맥이 아니라
빌드에서 알아야 한다.
