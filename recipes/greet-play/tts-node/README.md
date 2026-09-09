# tts-node — 문장을 읽는 쪽이 서 있는 자리

`greet-play` 의 `make` 스텝은 `bin/tts-typecast.sh` 를 부른다. 그 파일은 이
저장소가 아니라 **노드의 워크스페이스**에 있다 — 지금은 macOS 한 대의
`~/tts_workspace/bin/` 이고 그 디렉터리는 git 저장소가 아니다. 여기 있는 것은
그 사본이다. 노드를 다른 기계에 세울 때 여기서 가져간다.

```text
   tts-typecast.sh    Typecast API 로 문장 하나를 mp3 · wav 로 낸다
   labels.yaml        노드가 광고하는 라벨.  local.yaml 의 labels: 자리
```

## 엔진이 둘이다

```text
   say         macOS 내장.  요금 0 · 네트워크 없음 · 키 없음.  기본
   Typecast    API.  글자당 과금 · 네트워크 · 키 필요.  요청이 명시로 부를 때만
```

`say` 를 `greet.sh` 가 쓰고, Typecast 를 `tts-typecast.sh` 가 쓴다. Typecast 로
간 이유는 목소리다 — `say` 의 한국어 음성은 아홉이고 속도를 못 고른다. 상현
(`tc_69fc0cff784968297fb45daa`) 을 다섯 중에 사람이 골라 스크립트 기본값으로 박았다.
바꾸려면 `TYPECAST_VOICE_ID`. 목록은 `--list-voices` (590 개).

## 키

```text
   자리        ~/.config/typecast/api.key   (chmod 600)   또는 TYPECAST_API_KEY
   헤더        X-API-KEY
   엔드포인트  POST https://api.typecast.ai/v1/text-to-speech
```

**키를 명령줄에 적지 않는다.** `ps` 에 남는다. 키가 없으면 스크립트가 어디 두면
되는지 알려 주고 exit 3 으로 죽는다 — 조용히 `say` 로 갈아타지 않는다. 부른
사람이 원한 엔진이 아닌 소리를 성공이라 부르는 것이 제일 나쁘다.

**`HOME` 을 믿지 않는다.** enode 는 걸러진 환경으로 스텝을 띄우고 그때 `HOME` 이
없으면 `$HOME/.config/...` 가 `/.config/...` 이 되어 키를 조용히 못 찾는다.
스크립트가 `id -un` 으로 집을 되짚는다. `env -i PATH=/usr/bin:/bin` 으로 확인했다.

응답은 오디오 바이트 그대로다. 실패해도 본문이 오므로 HTTP 코드를 안 보면 에러
JSON 이 `greeting.mp3` 라는 이름으로 저장된다. 200 이 아니면 파일을 지우고 본문
앞부분을 찍는다.

## say 쪽 함정 둘

이 기계의 한국어 음성은 Eddy · Flo · Grandma · Grandpa · Reed · Rocko · Sandy ·
Shelley · Yuna 아홉이다. **Yuna 를 뺀 여덟은 이름만 주면 안 된다.** 같은 이름이
열네 언어에 있어서 `say -v Sandy` 는 영어 Sandy 를 집고 한국어를 못 읽어
0.016초짜리 빈 파일을 낸다. `say -v "Sandy (한국어(한국))"` 로 언어까지 붙인다.

macOS 에는 mp3 인코더가 없다. `afconvert` 는 읽기만 한다. lame 3.100 을 소스에서
`~/.local` 에 깔았고, `fmt_mp3` 라벨은 그것이 있을 때만 싣는다.

## 라벨

`labels.yaml` 대로다. 설정을 고친 뒤 **enode 를 재기동해야** 광고에 실린다.
`runctl capabilities` 에 `tts_typecast yes` 가 보이면 됐고, `runctl dry-run` 으로
`tts_typecast=yes` 를 요구하는 계약이 이 노드로 걸리는지 본다.
