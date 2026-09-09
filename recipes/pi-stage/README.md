# pi-stage — 어느 기계에 물려도 보드의 LED 를 흔든다

라즈베리파이 2 Model B 를 랜선으로 직결한 기계에서, 온보드 LED 두 개를
SSH 로 켜고 끄고 깜빡인다.

```powershell
./Initialize-RpiAccess.ps1                      # 기계마다 한 번
./Set-RpiLed.ps1 -Action Blink -Hz 2            # 초당 2회 점멸
./Set-RpiLed.ps1 -Action Stop                   # 세우고 원상복구
```

말로 하면 한 줄이지만 이 recipe 가 보여 주려는 것은 따로 있다. **보드를
찾는 일과 보드를 부리는 일이 다른 문제이고, 앞의 것이 훨씬 어렵다.**
직결 링크에는 DHCP 도 DNS 도 라우터도 없다.

## 주소를 모르고 붙는다

처음에는 고정 IP 로 붙었다. 보드는 `192.168.137.50` 을 들고 있고 그 대역은
Windows 인터넷 연결 공유가 쓰는 자리라, 호스트가 `192.168.137.1` 이 되어
있을 때만 통한다. 그 전제가 무너지는 경우가 실제로 셋이었다.

```text
   공유가 꺼진다        재부팅하면 풀린다.  이더넷이 169.254 로 떨어진다
   기계가 바뀐다        어댑터 GUID 도 인터페이스 인덱스도 다른 값이다
   OS 가 바뀐다         macOS 에는 그 공유도 그 대역도 없다
```

그래서 주 경로를 **mDNS** 로 바꿨다. 보드의 `avahi-daemon` 이 자기 이름을
링크에 광고하고, Windows 10 이상과 macOS 는 둘 다 mDNS 를 기본으로 안다.

```bash
ssh sunny@sunnypi.local
```

이 한 줄에는 주소도 서브넷도 인터페이스도 안 들어간다. 이름이 링크로컬
주소로 풀리고, 링크로컬은 리눅스가 언제나 갖고 있어서 DHCP 와 무관하다.
**설정할 것이 없는 경로가 가장 이식성이 높다.**

`Connect-Rpi.ps1` 은 세 경로를 순서대로 시도한다.

```text
   1  mDNS                sunny@sunnypi.local          설정이 필요 없다
   2  고정 IPv4           sunny@192.168.137.50         같은 서브넷일 때만
   3  IPv6 링크로컬 탐색   MAC 이 b8:27:eb 인 이웃      이름이 안 풀릴 때
```

3번은 OS 마다 이웃 캐시를 읽는 도구가 다르다 — Windows 는 `Get-NetNeighbor`,
macOS 는 `ndp -an`, 리눅스는 `ip -6 neigh` 다. 스코프 표기도 갈린다.
Windows 는 인터페이스 인덱스를(`%9`), 나머지는 이름을(`%en0`) 붙인다.

판정은 핑이 아니라 **실제 SSH 로그인**으로 한다. 핑은 근거가 못 된다 —
리눅스는 브로드캐스트 핑을 무시하고 Windows 방화벽은 ICMP 응답을 막아서,
멀쩡히 붙는 링크에서도 핑이 죽은 것처럼 보인다.

## 트리거를 떼지 않으면 안 먹는다

LED 는 커널의 LED 클래스가 sysfs 로 내준다. 각 LED 에는 트리거가 붙어
있어서 커널이 값을 계속 쓴다. ACT 는 SD카드 활동(`mmc0`), PWR 은 입력
전원(`input`)을 따라간다.

**트리거를 뗀 뒤에야 `brightness` 에 쓴 값이 남는다.** 안 떼면 쓴 값이
곧바로 덮여서 아무 일도 안 일어난 것처럼 보인다.

```bash
echo none > /sys/class/leds/ACT/trigger    # 먼저 뗀다
echo 1    > /sys/class/leds/ACT/brightness # 그 다음에야 먹는다
echo mmc0 > /sys/class/leds/ACT/trigger    # 끝나면 되돌린다
```

되돌리는 것이 예의가 아니라 기능이다. 안 되돌리면 SD카드 표시와 전원
표시가 죽은 채로 남아서, 다음 사람이 보드 상태를 눈으로 못 읽는다.
`Set-RpiLed.ps1` 은 `Stop` 과 시간제한 `Blink` 가 끝날 때 자동으로 되돌린다.

`sudo` 는 안 쓴다. `99-led-permissions.rules` 가 LED 파일을 `gpio` 그룹에
열어 주고 계정이 그 그룹에 있다. 부팅할 때마다 udev 가 다시 적용한다.

## 원격 스크립트는 base64 로 싣는다

여러 줄 셸 스크립트를 `ssh` 인자로 넘기면 따옴표가 PowerShell, ssh, 원격
셸을 거치며 세 번 해석된다. 그래서 본문을 base64 로 감싸서 보낸다.

부수적으로 사고 하나가 사라진다. 점멸 루프를 세울 때 `pkill -f` 를 쓰는데,
스크립트 본문이 원격 명령줄에 남으면 **그 패턴이 자기 명령줄까지 잡아서
SSH 세션이 스스로 죽는다.** 실제로 두 번 당했다. base64 로 실으면 원격
명령줄에는 `echo <b64> | base64 -d | sh` 만 남아서 매칭될 본문이 없다.

## 세 스크립트

```text
   Initialize-RpiAccess.ps1   기계마다 한 번.  키를 만들어 보드에 심고
                              ssh config 에 적고 udev 규칙을 깐다
                              비밀번호를 두 번 묻는다

   Connect-Rpi.ps1            보드를 찾아 SSH 로그인까지 확인한다
                              -PassThru 로 Target 과 Method 를 돌려준다

   Set-RpiLed.ps1             On Off Blink Stop Restore Status
                              -Led 로 ACT PWR Both,  -Hz 로 점멸 속도
```

`Initialize-RpiAccess.ps1` 은 `authorized_keys` 와 로컬 `ssh config` 를 둘 다
**덮어쓰지 않고 이어 붙인다.** 다른 기계의 키와 다른 호스트 항목이 이미 들어
있을 수 있어서다. 이미 적힌 항목이 있으면 그냥 넘어간다.

## 키 이름이 기본값이 아니면 ssh 가 안 집는다

이 함정에 실제로 걸렸다. 키를 심어 놓고도 이렇게 하면 거절당한다.

```text
debug1: identity file C:\Users\MSI/.ssh/id_ed25519 type -1
debug1: Will attempt key: C:\Users\MSI/.ssh/id_rsa
...
sunny@192.168.137.50: Permission denied (publickey,password).
```

`type -1` 은 그 파일이 없다는 뜻이다. ssh 는 `id_rsa` · `id_ed25519` 같은
**기본 이름만** 후보로 올리는데, 이 키는 `id_rpi_sunnypi` 라 목록에 아예 안
들어간다. 서버는 `publickey` 를 받겠다고 했지만 클라이언트가 내밀 것이
없었던 것이다. **네트워크가 끊긴 것처럼 보이지만 주소와는 무관하다.**

그래서 부트스트랩이 `~/.ssh/config` 에 항목을 적는다. 적고 나면 짧은 이름
하나로 붙는다.

```bash
ssh sunnypi
```

```text
Host sunnypi
  HostName sunnypi.local

Host sunnypi sunnypi.local 192.168.137.50
  User sunny
  IdentityFile ~/.ssh/id_rpi_sunnypi
  IdentitiesOnly yes
  StrictHostKeyChecking accept-new
```

키 경로를 물결표로 적는 것이 중요하다. 홈 경로도 경로 구분자도 기계마다
다르므로, 절대 경로로 적으면 그 파일이 그 기계에서만 맞는다.

무한 점멸은 보드에 떼어 놓고 SSH 는 빠진다. 그래서 노트북을 닫아도 계속
돈다. `Stop` 이 세운다.

## 기계마다 다른 것

```text
   Windows      mDNS 가 그대로 된다.  추가 설치가 없다
   macOS        mDNS 가 그대로 된다.  Bonjour 가 이미 있다
   PowerShell   7 이상이 필요하다.  $IsWindows 와 $IsMacOS 로 갈린다
```

보드에 인터넷을 물려야 할 때만 호스트 쪽 공유가 필요하다. LED 를 흔드는
데는 필요 없다. Windows 는 인터넷 연결 공유, macOS 는 인터넷 공유이고,
둘 다 호스트가 게이트웨이가 되는 구조지만 대역이 다르다 — macOS 는
`192.168.2.0/24` 를 쓰므로 보드의 고정 IP 와 안 맞는다. 그 경우에도
mDNS 경로는 그대로 산다.

## 실측

2026-09-09 에 Windows 11 기계에서 돌린 것이다.

```text
Raspberry Pi link check
  ..    trying mDNS: sunny@sunnypi.local
  OK    sunny@sunnypi.local answered as 'sunnypi'
Connected.
  path : mDNS
```

초당 2회 점멸이 실제로 진동하는지 보려고, 보드에서 125ms 간격으로 두 LED
의 `brightness` 를 16번 읽었다.

```text
255255 255255 00 00 255255 2550 00 255255 255255 00 255255 255255 00 00 255255 2550
```

두 칸마다 뒤집힌다. 표본 간격이 125ms 이므로 반주기가 250ms, 곧 2 Hz 다.
읽는 시점과 쓰는 시점이 안 맞아 한쪽만 잡힌 표본이 둘 있는데(`2550`),
그것도 두 LED 가 같은 루프에서 연달아 쓰이기 때문이지 어긋난 것이 아니다.

명령별 결과다.

```text
Blink -Hz 2          blink running pid=11080
Status (돌 때)        ACT trigger=none   PWR trigger=none   loops=1
Stop                 stopped 1 loop(s)
Status (세운 뒤)      ACT trigger=mmc0   PWR trigger=input  loops=0
```

## led-toggle 을 대신한다

이 recipe 는 `recipes/led-toggle` 을 지우고 그 자리를 받는다. 그쪽은 외장
LED 가 GPIO 17 과 27 에 물려 있다고 **가정**했고, 그 문서 스스로 그 값을
「미확인」이라고 적어 두었다. 실제로 읽어 보면 두 핀은 출력으로 잡혀 있고
값은 LOW 다.

```text
17: op -- -- | lo // GPIO17 = output
27: op -- -- | lo // GPIO27 = output
```

핀이 출력으로 서 있다는 것은 누군가 그렇게 세웠다는 뜻일 뿐, 거기 LED 가
물려 있다는 증거가 아니다. 배선을 모르는 채로 핀을 흔들면 무엇이 켜지는지
확인할 방법이 없다.

온보드 ACT 와 PWR 은 그 문제가 없다. 커널이 LED 클래스로 내주므로 이름으로
부르고, 색과 위치가 보드에 붙박여 있어서 눈으로 확인하는 데 배선 지식이
필요 없다. **확인된 것을 남기고 가정한 것을 지운다.**

바깥 핀에 LED 를 물리는 갈래가 다시 필요해지면, 그때는 배선을 먼저 적고
`pinctrl get` 으로 확인한 값을 문서에 넣은 뒤에 되살리는 것이 맞다.

## 아직 모르는 것

`brightness` 를 읽으면 `max_brightness` 가 1인데도 켜진 값이 255 로 나온다.
쓰기는 0과 1로 하고 실제로도 켜지고 꺼지므로 동작에는 문제가 없지만,
드라이버가 어느 단계에서 그 값을 바꾸는지는 안 봤다. 밝기 단계를 쓸 일이
생기면 그때 봐야 한다.

점멸 주기는 사용자 공간 루프가 만든다. `sleep` 이 정확하지 않아서 부하가
걸리면 흔들린다. 정확한 주기가 필요하면 커널 `timer` 트리거로 넘기는 것이
맞다 — `delay_on` 과 `delay_off` 에 밀리초를 쓰면 커널이 재운다.

```bash
echo timer > /sys/class/leds/ACT/trigger
echo 250 > /sys/class/leds/ACT/delay_on
echo 250 > /sys/class/leds/ACT/delay_off
```

이 recipe 는 두 LED 를 **같은 위상으로** 흔든다. 서로 엇갈리게 하려면
루프를 나눠야 하는데, 그 모양이 필요해진 적이 아직 없어서 안 넣었다.
