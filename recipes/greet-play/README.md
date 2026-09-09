# greet-play — 한 기계에서 만들고 다른 기계에서 튼다

문장 하나를 받아서 mp3 로 읽고, 그것을 스피커가 달린 노드로 보내 재생한다.

```bash
./submit.sh "안녕하세요 손신입니다"
```

말로 하면 한 줄이지만, 이 계약이 보여 주려는 것은 따로 있다.
**만드는 능력과 트는 능력이 다른 기계에 있고, 계약은 그 둘을 이름으로
부르지 않는다.** 어느 기계인지는 매처가 속성으로 고른다.

## 두 역할

```json
"requires": [
  { "as": "tts",   "capability": "agent.reason",
    "service": "tts", "tts": "say-macos", "fmt_mp3": "yes" },
  { "as": "board", "capability": "agent.reason",
    "audio_playback": "true", "board": "rpi2b-v1.1" }
]
```

`tts` 는 mp3 를 낼 수 있다고 광고한 노드다. `fmt_mp3` 가 핵심이다 —
macOS 에는 mp3 인코더가 없어서 lame 을 따로 깔아야 하고, 안 깐 기계는
그 라벨을 안 싣는다. 그래서 인코더 없는 기계는 **실행 시점이 아니라
매칭에서** 걸러진다. 계약은 "mp3 를 낼 수 있는 아무나" 라고만 적는다.

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

`make` 가 낸 것:

```
File type ID:   MPG3
Data format:    1 ch, 22050 Hz, .mp3
estimated duration: 1.933000 sec
```

`play` 가 한 것:

```
file=C:\Users\sunny\AppData\Local\Temp\enode-in-719428209\greeting.mp3 exists=True
player=MediaPlayer
played seconds=1.959
```

만든 길이 1.933 초와 재생한 길이 1.959 초가 맞는다. 파일이 열리기만 한 것이
아니라 끝까지 흘러갔다는 뜻이다.

## 아직 모르는 것

`audio_playback` 이 **어느 스피커**를 뜻하는지는 라벨에 안 적혀 있다.
위 실행에서 소리는 `board` 노드의 기본 오디오 장치로 나갔는데, 그 노드는
`os=windows` 인 기계이고 `board=rpi2b-v1.1` 은 거기 붙어 있다는 표시일 뿐이다.
보드에 물린 스피커인지 호스트 스피커인지는 로그로 안 갈린다.

가른다면 라벨을 쪼개는 것이 맞다. 출력 장치를 고르는 인자가 필요해지면
그때는 계약이 아니라 노드의 광고가 먼저 바뀌어야 한다.

## 미리 보기

`dry-run` 은 아무것도 할당하지 않고 어느 노드가 걸리는지만 보여 준다.
소리도 안 난다.

```bash
runctl dry-run greet-play.json
```
