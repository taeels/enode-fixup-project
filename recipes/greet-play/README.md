# greet-play — 한 기계에서 만들고 다른 기계에서 튼다

문장 하나를 받아서 mp3 로 읽고, 그것을 스피커가 달린 노드로 보내 재생한다.

```bash
./submit.sh "안녕하세요 손신입니다"          # board 노드의 기본 오디오 장치
./submit.sh --pi "안녕하세요 손신입니다"     # 그 노드에 물린 파이의 3.5mm 잭
```

말로 하면 한 줄이지만, 이 계약이 보여 주려는 것은 따로 있다.
**만드는 능력과 트는 능력이 다른 기계에 있고, 계약은 그 둘을 이름으로
부르지 않는다.** 어느 기계인지는 매처가 속성으로 고른다.

## 두 역할

```json
"requires": [
  { "as": "tts",   "capability": "agent.reason",
    "service": "tts", "tts_typecast": "yes", "voice_typecast": "Sanghyun" },
  { "as": "board", "capability": "agent.reason",
    "audio_playback": "true", "board": "rpi2b-v1.1" }
]
```

`tts` 는 Typecast 로 읽을 수 있다고 광고한 노드다. `tts_typecast` 가 핵심이다 —
API 키가 있는 기계만 그 라벨을 싣고, 없는 기계는 안 싣는다. 그래서 키 없는
기계는 **실행 시점이 아니라 매칭에서** 걸러진다. 계약은 "상현 목소리로 읽을
수 있는 아무나" 라고만 적는다. `voice_typecast` 까지 요구하는 이유는 목소리가
곧 이 인사의 정체이기 때문이다 — 다른 목소리로 성공하는 것은 성공이 아니다.

처음에는 `say` 와 `lame` 으로 냈다 (`tts: say-macos`, `fmt_mp3: yes`). 노드가
그 라벨도 그대로 광고하므로 그 계약도 여전히 이 노드에 닿는다. 엔진을 바꾼 것은
목소리를 고르기 위해서다 — `say` 의 한국어 음성은 아홉이고 속도를 못 고른다.

`board` 는 소리를 낼 수 있다고 광고한 노드다. 속성의 존재가 곧 능력이다.

`as` 로 붙인 이름은 계약 안에서만 쓰는 별명이다. 스텝의 `uses` 가 이 별명을
가리키고, 실제 노드는 매처가 채운다. 두 역할이 같은 기계로 떨어질 수도 있고
다른 기계로 갈라질 수도 있다 — 계약은 그것을 모르고, 알 필요도 없다.

## 파일이 건너가는 법

```json
{ "id": "make", "uses": "tts",   "out": ["greeting.mp3"] },
{ "id": "play", "uses": "board", "needs": ["make"],
  "in": { "from": ["greeting.mp3"] } }
```

`make` 가 `$OUT` 에 낸 것을 `out` 으로 신고하고, `play` 가 `needs` 로 순서를
잡고 `in.from` 으로 받는다. 받는 쪽에서는 `$IN` 아래에 놓인다. 두 기계의
경로 모양이 달라도 (`/var/folders/...` 와 `C:\Users\...`) 스텝은 `$OUT` 과
`$IN` 만 알면 된다.

## 문장이 들어가는 자리

문장은 argv 배열의 한 칸으로 들어간다.

```json
"run": ["/bin/bash", "-c", "…$1…", "greet-play", "__TEXT__"]
```

스크립트 문자열 안에 문장을 박아 넣지 않는다. `bash -c` 의 위치 인자로
넘기고 안에서 `"$1"` 로 받는다. 셸을 한 번 덜 거치므로 따옴표가 들어 있는
문장에도 명령이 안 갈린다. `submit.sh` 가 `__TEXT__` 칸만 바꿔 끼운다.

## 재생 쪽이 하는 일

`play` 스텝은 `mpg123`, `ffplay`, .NET `MediaPlayer` 순으로 찾아서 있는 것을
쓴다. 앞의 둘이 없는 기계가 있어서 그렇다. `MediaPlayer` 는 비동기라 길이를
읽어서 그만큼 기다린 뒤에야 끝낸다 — 안 기다리면 스텝이 먼저 끝나고 소리는
중간에 잘린다.

## 실측

2026-09-09 에 돌린 것이다.

```
greet-play-20260909-132909  SUCCEEDED
  tts    620abcbb7e47  thstls110534@MacBook-Pro.local
  board  270c97c94415  taeels@sunnybook
```

`make` 가 낸 것 (당시는 `say`):

```
File type ID:   MPG3
Data format:    1 ch, 22050 Hz, .mp3
estimated duration: 1.933000 sec
```

Typecast 로 바꾼 뒤 파이로 낸 것 (`greet-play-pi-20260909-154201-76258`):

```
make   wrote greeting.mp3  (voice=tc_69fc0cff784968297fb45daa model=ssfm-v30)
       1 ch, 44100 Hz, 2.429388 sec
play   ssh=C:\Program Files\Git\usr\bin\ssh.exe
       target=sunny@192.168.137.50 dev=plughw:1,0
       mpg123 rc=0        스텝 4.5초 — 클립 2.43초에 scp 와 ssh 왕복
```

파이가 `/proc/asound/cards` 로 답한 배치는 `0 vc4hdmi`, `1 bcm2835 Headphones`
다. `plughw:1,0` 이 잭이다.

`play` 가 한 것:

```
file=C:\Users\sunny\AppData\Local\Temp\enode-in-719428209\greeting.mp3 exists=True
player=MediaPlayer
played seconds=1.959
```

만든 길이 1.933 초와 재생한 길이 1.959 초가 맞는다. 파일이 열리기만 한 것이
아니라 끝까지 흘러갔다는 뜻이다.

## 어느 스피커였나 — 답이 났다

처음 이 문서는 여기를 모른다고 적었다. `audio_playback` 은 소리를 낼 수 있다고만
말하고 **어느** 스피커인지는 안 말한다. 그 뒤 실측으로 갈렸다.

```text
   greet-play (기본)   board 노드 자신의 기본 오디오 장치.  위 실행에서는
                       Windows 노트북 스피커였다.  player=MediaPlayer 가 그 증거다

   greet-play --pi     그 노드에 ssh 로 물린 라즈베리파이의 3.5mm 잭
```

라벨은 여전히 못 가른다. **가르는 것은 스텝이다.** `--pi` 가 바꾸는 것은 `play`
스텝 하나이고 `requires` 는 그대로다 — 매칭은 노드를 고를 뿐, 소리가 어느
구멍으로 나가는지는 그 노드 위에서 도는 명령이 정한다.

라벨을 쪼개는 것이 옳은 방향이라는 판단은 그대로다. 다만 그때는 계약이 아니라
노드의 광고가 먼저 바뀌어야 한다.

## 파이로 낼 때 걸리는 것

```text
   장치를 명시한다        이 파이에는 HDMI 싱크가 붙어 있다 (card 0, SyncMaster).
                          기본 라우팅에 기대면 소리가 조용히 모니터로 간다.
                          그래서 언제나 plughw:1,0 을 적는다

   mpg123 을 쓴다         aplay 는 mp3 를 못 읽는다.  장치 플래그도 다르다 —
                          mpg123 은 -a, aplay 는 -D

   새 이름으로 올린다      /tmp 에 444 로 남은 파일들이 있어 덮어쓰기가 거부된다.
                          계약이 매번 uuid 를 섞은 이름을 쓴다

   볼륨을 올린다           한 번 안 들린 적이 있고 원인이 이것이었다.
                          -19.88dB 로 exit 0 이 났다.  amixer 줄이 멱등하게 들어 있다

   Git 의 ssh 를 쓴다       Windows 내장 ssh.exe 는 enode 스텝 안에서 뜨지 못한다.
                          아래에 따로 적었다
```

## Windows 내장 ssh 는 스텝 안에서 뜨지 못한다

보드를 다른 Windows 기계로 옮긴 뒤 `scp` 가 매번 `rc=255` 로 죽었다. 같은
기계의 터미널에서는 같은 명령이 붙었다. 키도 주소도 계정도 문제가 아니었다 —
진단 스텝으로 좁혀 보니 **`ssh -V` 조차 rc=255 에 출력 0 바이트**였다. 네트워크도
키도 안 건드리는 명령이 그렇다는 것은 `ssh.exe` 가 뜨자마자 죽는다는 뜻이다.
stdin 을 `NUL` 로 주어도, 콘솔을 새로 붙여도 (`Start-Process`) 같았다.

같은 기계의 Git for Windows 가 가진 ssh 는 스텝 안에서 정상이다.

```text
   C:\Windows\System32\OpenSSH\ssh.exe -V      rc=255  출력 없음
   C:\Program Files\Git\usr\bin\ssh.exe -V     rc=0    OpenSSH_10.0p2
```

그래서 `play` 스텝이 Git 의 ssh 와 scp 가 있으면 그것을 쓰고, 없으면 예전처럼
PATH 의 것을 쓴다. 어느 쪽을 골랐는지 첫 줄에 찍는다 (`ssh=...`). 원인은 못
밝혔다 — 내장 ssh 가 어떤 프로세스 컨텍스트에서 죽는지는 그 기계에서 더 파야
한다. 여기 적는 것은 증상과 우회다.

`~/.ssh/config` 가 그 파이 주소를 특정 키(`IdentityFile`, `IdentitiesOnly yes`)에
고정하고 있으면 새로 만든 키는 쓰이지 않는다. 이번에도 새 키를 심었지만 실제로
쓰인 것은 config 의 `id_rpi_sunnypi` 였다. 그 항목을 적는 쪽이
`recipes/pi-stage/Initialize-RpiAccess.ps1` 이고, 왜 그렇게 적는지는
`recipes/pi-stage/README.md` 의 「키 이름이 기본값이 아니면 ssh 가 안 집는다」에 있다.

## 안 될 때 먼저 던지는 것

```bash
./submit.sh --diag
```

소리 없이 `board` 노드 위에서 ssh 환경만 찍어 온다 — 누구로 도는지, `HOME` 이
있는지, `ssh` 가 어느 파일인지, 내장 `ssh -V` 와 Git `ssh -V` 가 각각 뜨는지,
Git ssh 로 파이에 붙는지, 파이의 사운드 카드 배치와 `mpg123` 유무. 로그를 그
자리에서 보여 준다. 위의 `rc=255` 를 좁히는 데 여섯 번 던진 것을 하나로 접었다.

읽는 법 — `builtin_ssh_V` 가 `rc=255 out_bytes=0 err_bytes=0` 이면 위 증상이다.
`git_ssh_whoami` 의 `out:` 에 계정 이름이 찍히면 파이까지는 됐고, 남은 것은
소리 쪽이다.

## tts 쪽

`make` 가 부르는 `bin/tts-typecast.sh` 는 노드의 워크스페이스에 있다. 그 사본과
노드가 광고하는 라벨, 키 자리, `say` 쪽 함정은 `tts-node/` 에 있다.

주소와 장치는 환경변수로 덮는다.

```bash
PI_HOST=pi@raspberrypi.local PI_DEV=plughw:0,0 ./submit.sh --pi "문장"
```

## 재생을 무엇으로 확인하나

`exit 0` 은 부족하다. 위의 볼륨 사고가 정확히 그 경우였다. 파이 쪽 스텝은
끝나면서 커널의 재생 카운터를 한 줄 찍는다.

```text
   /proc/asound/card1/pcm0p/sub0/status
```

`hw_ptr` 이 단조 증가하고 초당 증가량이 클립의 표본율과 맞으면 카드가 실제로
프레임을 뽑아낸 것이다. 끝이 `DRAINING` 이면 끝까지 갔다는 뜻이고, 이 값이
card 1 의 것이므로 HDMI 로 새지 않았다는 증거도 된다.

실측 (`pi-play-3`, 2026-09-09):

```text
   mpg123 -q -o alsa -a plughw:1,0     exit 0 · 2.244초 (클립 2.220초)
   hw_ptr    초당 22,029 프레임          클립 22050Hz, 오차 0.1%
   card 0    재생 전후 모두 closed       HDMI 로 안 샜다
```

여기까지가 확인되는 전부다. **DAC 에서 끝난다** — 잭에 스피커가 꽂혀 있는지,
전원이 켜져 있는지는 기계가 알 방법이 없다. 사람이 방에 있어야 한다.

## 미리 보기

`dry-run` 은 아무것도 할당하지 않고 어느 노드가 걸리는지만 보여 준다.
소리도 안 난다.

```bash
runctl dry-run greet-play.json
```
