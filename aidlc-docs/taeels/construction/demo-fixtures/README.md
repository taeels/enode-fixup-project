# 데모 고정 시나리오 픽스처 — 진행자 산출물

`decisions.md` 8.3 이 진행자에게 맡긴 고정 시나리오 픽스처다. demo-back 의
공개 별칭 두 개(`led-toggle` · `welcome-audio`)가 서버측 allow-list 로 겨누는
승인된 example 이 이 둘이다.

```text
   공개 별칭        example 이름               파일
   led-toggle       demo-led-toggle           internal/contract/examples/demo-led-toggle.json
   welcome-audio    demo-welcome-audio        internal/contract/examples/demo-welcome-audio.json
```

`runctl example demo-led-toggle` · `runctl example demo-welcome-audio` 로 뽑힌다.
example 이름과 공개 별칭을 가른 것은 submission-contract 1절의 요구다 —
공개 별칭은 브라우저가 보내는 값이고 example 이름은 서버 매핑의 값이다.

## 왜 별도 capability 가 아니라 attrs 인가

계약 어휘는 닫혀 있다 — `agent.reason` 과 `orchestration` 둘뿐이고 모르는
이름은 422 다(`contract.go` knownCapability). 그래서 LED · 스피커 · tts 는
capability 가 아니라 `agent.reason` 의 attrs 로 겨눈다. 매처가 부분집합
일치이므로(`Capability.Satisfies`) 노드가 그 attr 를 광고하면 잡힌다.

```text
   요구 (requires[].<attr>)      광고하는 노드            담당
   device: led                   windows (ssh -> rpi)     nacl1119
   device: speaker               windows (rpi 스피커)      nacl1119
   tts: higgsfield               mac                      shin-son
```

sandbox 표시도 같은 자리다 — 노드 설정의 `labels: {sandbox: <값>}` 한 줄이
`capabilities[].attrs.sandbox` 로 화면에 닿는다(obs business-rules 239, 코드 변경 0).
픽스처는 sandbox 를 요구하지 않는다 — 표시 전용이라 매칭에 넣지 않는다.

## led-toggle — 한 Run 이 두 패턴을 보인다

진행자 결정. `decisions.md` 8.3 의 「1초 heartbeat + persistent-on, 둘」을 두
별칭이나 서버 메모리 번갈기로 나누지 않는다(submission-contract 4절이 번갈기를
금지). 한 Run 이 두 스텝으로 heartbeat 를 먼저, 그다음 persistent-on 을 보인다.
버튼 하나 · example 하나 · 결정적이고 상태가 없다.

## welcome-audio — cross-node 는 봉인 blob 으로

`synthesize`(mac, voice)는 agent 스텝이다 — shin-son macbook 에 뜬 Claude 가
higgsfield 도구로 wav 를 합성해 `$OUT/welcome.wav` 로 낸다(2026-09-09 결정).
agent 스텝이므로 성패는 exit_code 가 아니라 produced 로 판정한다
(`contract.go` ErrExitOnAgent — 하네스는 헛소리를 하고도 0 으로 끝난다).

그 out 은 Run 의 봉인(Record)에 artifact 로 들어간다. `play`(windows, board)가
`needs`+`in.from` 으로 그 산출에 닿으면 러너가 봉인 blob 을 GET 해 rpi 스피커로
재생한다 — mediator demo 가 봉인을 꺼낼 수 있어야 하는 자리가 이것이고, 기존
PUT/GET blob 라우트(`api.go`)가 이미 진다. 계약은 multi.json 과 같은 모양이고
Mediator 변경도 새 경로도 없다(`decisions.md` 8.3).

## 제출자 이름 주입 계약 (demo-back 과의 접점)

`synthesize` 의 `in.prompt` 에 `{{submitter}}` 자리표시자가 있다. demo-back 이
example 을 제출로 옮길 때 이 자리표시자를 게스트 이름으로 치환한다 —
`decisions.md` 8.6 의 「제출자는 Guest 로그인 이름을 그대로 쓴다」와
submission-contract 4절의 이름 주입 함수가 여기서 만난다.

```text
   치환 전   ... says exactly "{{submitter}}님 환영합니다", and write ...
   치환 후   ... says exactly "밝은 수달님 환영합니다", and write ...
```

raw 로(`runctl example ... | runctl submit`) 내면 자리표시자가 그대로 나가지만
데모 경로가 아니라 무해하다. 치환은 demo-back 의 몫이고, 서버는 받은 원문으로
식별을 계산한다(submission-contract 2절).

## 노드가 채우는 자리 — 명령 이름은 계약, 구현은 노드 소유자

계약이 겨누는 것은 명령의 이름과 성패 판정(exit_code 0 + produced)이다. 실제
argv 뒤의 하드웨어 동작은 노드 소유자가 자기 기계에 얹는다. 각 명령은 얇은
래퍼면 된다.

```text
   자리                        노드        담당        비고
   enode-demo-led heartbeat    windows     nacl1119    run 스텝.  ssh -> rpi.  1초 heartbeat 점멸
   enode-demo-led on           windows     nacl1119    run 스텝.  ssh -> rpi.  persistent-on
   enode-demo-play <file>      windows     nacl1119    run 스텝.  rpi 스피커로 봉인 wav 재생
   synthesize (agent)          mac         shin-son    agent 스텝.  Claude + higgsfield 로 wav 합성
```

LED 는 nacl1119 PC 에서 연결·제어가 확인됐다(2026-09-09). rpi 가 nacl1119 PC 에
물려 있으므로 wav 재생도 거기서 난다. 남은 것은 확인된 그 제어를 위 이름의
명령으로 노출하는 것뿐이다 — nacl1119 의 enode 워크스페이스 CLAUDE.md 가이드가
그 자리를 진다. 명령 이름이 위와 다르면 그 이름을 여기 정본으로 고치고 픽스처의
`run` argv 를 맞춘다(한 파일 · 진행자 병합).

음성 합성은 run 헬퍼가 아니라 agent 스텝이다 — shin-son macbook 의 Claude 가
higgsfield 로 낸다. shin-son 의 자리는 그 mac 노드가 `harness: claude` 와
`tts: higgsfield` 를 광고하고 higgsfield mcp 가 붙어 있는 것이다.

## 검증

계약 스키마(`contract.go` Validate)에 정적으로 맞췄다 — 닫힌 capability 어휘 ·
attrs 형제 키 · 역할 선언 · needs 후방 간선 · 모든 스텝의 success_when 커버리지 ·
lint 경고 0. 이 기계에 go 툴체인이 없어 `go test` 는 실행하지 못했다.
실행 확인이 필요하다 — go 가 있는 곳에서:

```bash
go test ./internal/contract/... ./cmd/runctl/...
```

`TestExamples_ParseAndValidate` · `TestExamples_EveryStepHasASuccessCondition` ·
`TestExamples_LintClean` 이 새 example 둘도 함께 돈다.

## runixs 세 물음의 답 (audit 401)

```text
   Q1 LED·음원 시나리오 파일 경로   위 두 파일.  led-toggle -> demo-led-toggle,
                                    welcome-audio -> demo-welcome-audio 로 매핑한다
   Q2 웹캠 방송 공급자·주소          호스트형 브로드캐스트 임베드 (decisions 8.2 Q4).
                                    P2P 아님.  실제 공개 임베드 URL·허용 origin 은
                                    진행자가 별도로 넣는다 (settings.json 의 webcam)
   Q3 sandbox capabilities[].attrs   합의됨 (decisions 8.5).  표시 전용 · 노드 labels
       .sandbox 표시                 에서 온다 · 코드 변경 0
```
