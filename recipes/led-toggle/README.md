# led-toggle — 파이에 물린 LED 둘을 번갈아 켠다

```bash
./submit.sh                    # 17 · 27 · 6회 · 250ms
./submit.sh 17 27 10 120       # 빨강 · 초록 · 반복 · 간격
```

`greet-play` 가 소리를 다뤘다면 이쪽은 핀이다. 계약의 모양은 같다 —
**속성이 노드를 고르고, 스텝이 무엇을 할지 정한다.**

## 고르는 자리

```json
"requires": [
  { "as": "board", "capability": "agent.reason",
    "board": "rpi2b-v1.1", "led_red": "true", "led_green": "true" }
]
```

`led_red` 와 `led_green` 은 노드 설정의 `labels` 에서 온다. 값이 `"true"`
문자열인 것이 중요하다 — 라벨은 `map[string]string` 이라 따옴표 없이 쓰면
YAML 이 불리언으로 읽고 설정 읽기가 실패한다.

능력 이름을 새로 만들지 않는다. 계약이 요구할 수 있는 어휘는 `agent.reason`
과 `orchestration` 둘뿐이고, 주변장치는 **속성의 존재**로 표현한다.

## GPIO 번호는 아직 확인 안 됐다

```text
   기본값   빨강 17 · 초록 27      미확인
```

이 숫자는 처음에 문서에 적혀 있던 값이고, **실제로 켜서 어느 LED 가 들어오는지
본 사람이 아직 없다.** 그래서 이 레시피는 패턴을 돌리기 전에 하나씩 1초씩
켠다.

```text
   identify: red only      1초
   identify: green only    1초
   pattern: alternating    반복 × 간격
```

처음 돌릴 때는 파이 앞에서 눈으로 보라. 다르면 인자로 바로잡고, 그 값을
이 문서와 워크스페이스의 `CLAUDE.md` 에 적어라. 적고 나면 이 절의 「미확인」을
지운다.

핀이 무엇에 물려 있는지 모르는 채로 구동하는 것이라, 배선을 모르면 먼저
`pinctrl get <핀>` 으로 현재 기능부터 보는 편이 안전하다.

## 도구는 둘 중 있는 것을 쓴다

```text
   pinctrl       Raspberry Pi OS 의 요즘 도구
   raspi-gpio    예전 이름.  인자 모양이 같다 - set <핀> op dh|dl · get <핀>
```

둘 다 없으면 그 자리에서 멈추고 이유를 적는다. **`gpioset` 은 안 쓴다** —
프로세스가 끝나면 선을 놓아서 LED 가 그 자리에서 꺼진다. 켜 두는 것이
목적인 명령에는 맞지 않는다.

## 끄고 끝낸다

패턴이 끝나면 둘 다 내리고, 내린 뒤의 상태를 다시 읽어 로그에 적는다.

```text
   before:   패턴 전 상태
   after:    패턴 뒤 상태
```

켠 채로 끝난 단계가 다음 시연의 첫 장면을 망친다. 그래서 이것은 예의가
아니라 계약의 일부다.

## 어디서 도는가

스텝은 **Windows 노드에서** 돌고, 거기서 ssh 로 파이를 부린다. 파이는 enode 를
돌리지 않는다 — `board=rpi2b-v1.1` 은 그 노드에 파이가 물려 있다는 표시다.

주소는 환경변수로 덮는다.

```bash
PI_HOST=pi@raspberrypi.local ./submit.sh 17 27
```

원격 스크립트는 파일로 올려서 돌린다. 명령줄에 이어 붙이지 않는다 — 원격 셸이
`${...}` 를 먼저 먹어서 조용히 빈 값이 된다. 그리고 올릴 때 **BOM 과 CRLF 를
안 남긴다.** 원격 `bash` 가 `\r` 을 명령의 일부로 읽는다. PowerShell 의 기본
출력이 둘 다 붙이므로 `UTF8Encoding($false)` 로 직접 쓴다.

## 미리 보기

```bash
runctl dry-run led-toggle.json    # 핀을 안 건드린다. 노드만 고른다
```
