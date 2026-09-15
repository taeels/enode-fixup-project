# demo — 공개 데모의 고정 계약이 이 보드에 닿게 한다

`/ui/demo/` 의 「LED Toggle」과 「사운드 재생」은 진행자가 고정한 계약
`demo-led-toggle` · `demo-welcome-audio` 를 보낸다(`internal/contract/examples/`).
계약은 노드에 **이름**을 요구하고 **명령**을 부른다. 노드가 그 이름을 광고하고
그 명령을 갖추면 Mediator 를 안 바꾸고 붙는다 — 진행자가 그렇게 설계했다
(`aidlc-docs/taeels/construction/demo-fixtures/README.md`).

```text
   계약                요구하는 라벨           부르는 명령
   demo-led-toggle     device: led            enode-demo-led heartbeat --seconds 10  ·  enode-demo-led on
   demo-welcome-audio  device: speaker        enode-demo-play $IN/welcome.wav
                       tts: higgsfield (mac)  (agent 스텝 — tts 노드가 welcome.wav 를 낸다)
```

2026-09-09 현재 보드는 macOS 노트북에 물려 있고 그 노드가 `device: led` 와
`enode-demo-led` 를 갖췄다. LED 계약은 실 함대에서 통과했다
(`fx-led-live-163905`, ACT · PWR 2 Hz 10s, readback verified).

## enode-demo-play

사운드 쪽에 빠져 있던 명령이다. 파일을 파이로 올려 3.5mm 잭(card 1)으로 틀고
지운다. wav 는 `aplay`, mp3 는 `mpg123` — aplay 는 mp3 를 못 읽고 장치 플래그도
`-D` 와 `-a` 로 다르다. 파이 주소는 mDNS 이름이 기본이다(`sunny@sunnypi.local`).
`ENODE_PI_TARGET` · `ENODE_PI_CARD` 로 덮는다.

```sh
cp recipes/pi-stage/demo/enode-demo-play ~/enode-led-bridge/bin/   # 노드의 bin — 스텝 PATH 맨 앞
chmod +x ~/enode-led-bridge/bin/enode-demo-play
```

실 함대에서 확인했다 (`fx-play-mac-164010`): tts 노드가 Typecast 로 낸
`welcome.wav` 가 macOS 노드로 건너가 `enode-demo-play $IN/welcome.wav` 로
잭에서 났다 — `played=wav card=1 rc=0`.

## device 는 한 값이다 — 스피커는 두 번째 노드

노드 설정의 `labels:` 는 맵 하나라 `device` 는 값 하나다. 지금 노드는 `led` 를
광고하므로 `speaker` 를 광고할 노드가 하나 더 필요하다. 같은 기계에서 설정 파일을
하나 더 두고 enode 를 하나 더 띄운다 — 설정 파일 경로가 노드 id 의 일부라(ADR-017)
다른 노드가 된다. 둘 다 같은 파이를 부린다.

```yaml
labels:
  device: speaker
  board: rpi2b-v1.1
  tag: sunnypi
```

두 노드의 bin 이 같은 디렉터리여도 된다 — `enode-demo-led` 와 `enode-demo-play`
가 나란히 있으면 어느 계약이 와도 있다.

## tts 쪽

`tts: higgsfield` 는 진행자가 tts 노드의 음성 능력에 붙인 이름이다. 실물은
Typecast 상현이고, 그 노드의 CLAUDE.md 가 그 프롬프트를 받으면
`bin/tts-typecast.sh` 로 `welcome.wav` 를 내도록 적혀 있다
(`recipes/greet-play/tts-node/`). 실 함대에서 확인했다 (`fx-voice-agent-163905`):
픽스처의 프롬프트 그대로 agent 스텝이 2.1 초짜리 `welcome.wav` 를 냈다.
agent 스텝이라 클릭마다 모델 호출이 한 번 들어간다.
