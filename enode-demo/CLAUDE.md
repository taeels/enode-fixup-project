# enode-demo — 무대 노드 가이드

이 기계는 **enode 노드**이고, 무대 장치(스피커·LED)는 여기 달려 있지 않다.
**라즈베리파이에 달려 있고 SSH 로만 닿는다.** 아래 값은 실제로 돌려서 확인한
것이다. 확인 안 된 것은 그렇게 표시했으니, 그 자리는 믿지 말고 직접 재라.

```text
   이 기계     Windows 11.  enode 가 여기서 돈다.  워크스페이스 D:\enode-demo
   파이        sunnypi · 192.168.137.50 · 직결 이더넷
               Raspbian GNU/Linux 13 (trixie) · armhf · 6.18.34+rpt-rpi-v7
               Raspberry Pi 2 Model B Rev 1.1 · MAC B8:27:EB:72:7D:6A
   스피커      파이의 3.5mm 잭 (card 1)
```

## 접속 정보

```
RPI_HOST=192.168.137.50
RPI_USER=sunny
RPI_PORT=22
```

**공개키 인증이 설정돼 있다.** 이 PC 의 공개키가 파이의 `~/.ssh/authorized_keys`
에 들어 있어, **비밀번호 없이** 붙는다. 자격증명 파일(`ssh_cred`)은 더 이상
필요 없고, 비상용 fallback 으로만 남겨 둔다(`.gitignore` 로 커밋 차단).

## 네트워크 (전제)

- 파이는 이 PC와 **이더넷 직결**, PC 인터넷은 Wi-Fi.
- 파이는 `192.168.137.50` 정적 IP(ICS 대역). 접속하려면 이 PC 이더넷에
  **`192.168.137.1/24`** 가 있어야 한다. 없으면 `192.168.137.x` 가 Wi-Fi 로
  새서 파이에 안 닿는다. 확인: `ping 192.168.137.50`.
- 없을 때(관리자 PowerShell): `New-NetIPAddress -InterfaceAlias '이더넷 3' -IPAddress 192.168.137.1 -PrefixLength 24`

---

# 파이에 붙는 법

공개키 인증이라 **Windows 기본 `ssh.exe` / `scp.exe` 로 비밀번호 없이** 된다.
(이전에 PuTTY `plink`/`pscp` 를 쓰던 이유는 비대화식 비밀번호 문제였는데, 키가
들어간 지금은 필요 없다. `.tools\plink.exe`·`pscp.exe` 는 fallback 으로만 둔다.)

```powershell
ssh sunny@192.168.137.50 "hostname"
ssh -o BatchMode=yes sunny@192.168.137.50 "hostname"   # 스크립트/자동화용
```

호스트키는 `known_hosts` 에 이미 등록돼 있다(참고 지문
`SHA256:zvq+FRVtFDN+RB/r0zf28Clc7cg1B+wPseNW68jD+I8`). 처음 붙는 환경이면
`-o StrictHostKeyChecking=accept-new` 로 한 번 받아 둔다.

원격 명령에 `${...}` 가 들어가면 원격 셸이 먼저 먹는다. 그럴 때는 스크립트를
파일로 만들어 `ssh sunny@192.168.137.50 'bash -s' < script.sh` 로 보낸다.

---

# 소리를 내는 법

파일은 이 기계에 있고 스피커는 파이에 있다. 먼저 옮기고 그다음 튼다.

```powershell
scp welcome.mp3 sunny@192.168.137.50:/tmp/welcome.mp3
```

```sh
amixer -c 1 sset PCM 100% unmute
mpg123 -q -o alsa -a plughw:1,0 /tmp/welcome.mp3
```

## 포맷마다 재생기가 다르다

```text
   mp3    mpg123 -o alsa -a plughw:1,0        1.32.10 설치돼 있다
   wav    aplay  -D plughw:1,0
```

장치를 가리키는 플래그 이름이 `-a`(mpg123) 와 `-D`(aplay) 로 다르다.
**`aplay` 는 mp3 를 아예 못 읽는다.** 파이에 `ffmpeg` 도 있으니 정 필요하면
거쳐 갈 수 있지만, 디코더를 하나 더 끼우면 실패가 그 안에 숨는다.

## 장치는 card 1 이다

```text
   card 0   vc4hdmi      HDMI.  지금 아무것도 안 꽂혀 있다
   card 1   Headphones   bcm2835.  3.5mm 잭.  이쪽이다
```

`~/.asoundrc` 가 기본을 card 1 로 돌려 놓았지만, 재생 명령에 장치를 **명시적으로
적는다.** 기본값에 기대면 그것이 바뀐 날 조용히 HDMI 로 간다. 장치 이름은
`aplay -l` 로 본 것만 쓰고, 지어내지 않는다.

## 볼륨을 확인한다

한 번 안 들린 적이 있고 원인이 이것이었다. `aplay` 가 exit 0 로 끝나는데도
아무도 못 들었다.

```text
   그때        Playback -1988 [78%] [-19.88dB]
   올린 뒤     Playback   400 [100%] [  4.00dB]
```

`alsactl store` 로 저장돼 재부팅해도 남지만, 재생 전에 한 줄 넣는 편이
안전하다 — 멱등이다.

**`amixer cset numid=3 1` 은 이 커널에 없다.** 예전 단일 카드 시절의 라우트
선택자인데, 이 커널은 HDMI 와 Headphones 가 분리된 카드라 그 컨트롤이 아예
없다. 라우팅은 카드를 골라서 한다.

---

# 재생을 확인하는 법

exit 0 만으로는 부족하다. 위의 볼륨 사고가 정확히 그 경우였다. 셋을 본다.

```text
   종료 상태     0 인가
   걸린 시간     클립 길이와 맞는가.  즉시 끝났으면 열고 버린 것이다
   커널 카운터   하드웨어가 실제로 프레임을 먹었는가
```

세 번째가 결정적이다. 재생하는 동안 다른 셸에서 이것을 훑는다.

```sh
cat /proc/asound/card1/pcm0p/sub0/status
```

`hw_ptr` 이 단조 증가하고 초당 증가량이 클립의 표본율과 맞으면 카드가 실제로
소리를 뽑아낸 것이다. 끝이 `DRAINING` 이면 끝까지 갔다는 뜻이다. 이 값은
card 1 의 것이므로 HDMI 로 새지 않았다는 증거도 된다.

---

# LED 를 켜는 법

**아래 GPIO 번호는 확인 안 됐다.** 실제로 켜 보고 어느 LED 가 들어오는지 눈으로
본 사람이 아직 없다. 쓰기 전에 한 번 켰다 꺼서 확인하고, 다르면 이 문서를 고쳐라.

```sh
raspi-gpio set 17 op dh    # red on      (미확인)
raspi-gpio set 17 op dl    # red off
raspi-gpio set 27 op dh    # green on    (미확인)
raspi-gpio set 27 op dl    # green off
```

끄는 것을 잊지 않는다. 켠 채로 끝난 단계가 다음 시연의 첫 장면을 망친다.

---

# 하지 않는 것

```text
   apt 를 쓰지 않는다              파이에 인터넷이 없다.  ICS 경로는 잡혀 있는데
                                   DNS 도 TCP 도 전부 죽는다.  패키지가 필요하면
                                   윈도우에서 .deb 를 받아 pscp 로 넘겨 dpkg 로 깐다

   파이를 재부팅하지 않는다        다시 올라오는 데 시간이 걸리고 그동안
                                   이 노드는 할 수 있다고 광고한 채로 있다

   장치 이름을 지어내지 않는다     aplay -l 로 본 것만 쓴다

   /tmp/welcome.wav 를 쓰지 않는다 예전 시도가 남긴 444 파일이 있어 덮어쓰기가
                                   거부된다.  다른 이름으로 올린다
```

못 하겠으면 지어내지 말고 그 자리에 적는다. **재생하지 않고 재생했다고 적는
것이 이 시연에서 가장 나쁜 결과다.** 그리고 들었다고 적지 마라 — 이 기계도
파이도 소리를 듣지 못한다. 확인할 수 있는 것은 DAC 까지다.
