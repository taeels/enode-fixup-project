# ADR-068 질문 — 제어판이 Capabilities{Caps, At} 를 어디서 읽나

panel Functional Design 을 막는 열린 미정이다 (roster §6 · 유닛 정본
`unit-of-work.md` §5 「열린 미정」 · obs `business-rules.md` §5). **진행자
결정**이고 담당(nacl1119)이 정하지 않는다. 답이 오면 진행자가
`requirements/decisions.md` 에 행을 더해 닫고, 그 값으로 panel FD 를 낸다.

답은 `[Answer]:` 뒤에 글자 하나로 적는다.

---

## 걸려 있는 것 (실측)

제어판의 「탐지 능력 읽기 전용」 카드는 **무엇을 할 수 있나**(Caps)와 **언제 잰
값인가**(At)를 함께 보여야 한다 (`enode-features.md` 3.1.2). 그런데 그 둘의
정본인 `Capabilities{Caps, At}` 는 **데몬 메모리에만 있다** —
`internal/enode/detector.go:78` 의 `Detector.Capabilities()` 가 돌려주고,
`cmd/enode/main.go:167` 이 그것을 기동 로그에만 쓴다.

```text
   Caps   광고에 실린다.  Mediator 의 GET /v1/nodes 로 읽을 수 있다
          (runctl.Client.Nodes — panel 이 이미 딛는 표면)
   At     광고에 안 실린다.  ADR-068 이 거절했다
          (detector.go:31 「매처가 쓸 것이 아니고 싣는 순간 프로토콜이 는다」)
```

제어판은 데몬과 **다른 프로세스**다 — `enodectl serve` 가 `enode` 제어판
하위명령으로 exec 위임하고(심볼 상한 회피 · decisions §2 「제어판 서버 위치」),
그 하위명령이 `internal/panel` 을 net/http 로 띄운다. 데몬(`enode`)은 나가는
`http.Client` 만 있고 자기 서버가 없다 (`cmd/enode/main.go:155`). 그래서 같은
기계에 있어도 제어판이 데몬 메모리의 `At` 을 읽을 통로가 지금 없다.

`seen_at`(광고 시각)으로 대신할 수 없는 이유 — 광고 주기(기본 60초)와 탐지
주기(기본 5분)가 달라 최대 5분 어긋난다 (obs `business-rules.md` §5 ·
`main.go:67,76`).

---

## Question 1
제어판의 「탐지 능력」 카드가 이 노드의 능력 목록(Caps)과 그것을 잰
시각(At)을 어디서 읽나.

A) 데몬이 로컬 상태 파일에 쓰고 제어판이 읽는다. 정책 파일 옆
   `<stem>.status.yaml` 한 장 — 데몬이 탐지할 때마다 `Caps` 와 `At` 을 적고,
   제어판이 그 파일을 읽는다. 새 프로토콜 0 · Mediator 무변경 · 유닉스와
   윈도우가 정책 파일과 같은 권한 규칙으로 돈다. 대가는 데몬에 파일 쓰기
   표면 하나(`detector.go` 또는 `advertise.go`)와 파일 형식이 는다는 것

B) 데몬이 127.0.0.1 읽기 전용 엔드포인트를 열고 제어판이 조회한다.
   `Capabilities{Caps, At}` 를 그대로 낸다. 대가는 데몬(`enode`)이 지금 없는
   net/http 서버와 리스너·포트·로컬 인증을 새로 진다는 것 (심볼 상한은
   enodectl.exe 에만 걸리므로 데몬 크기는 안 걸린다)

C) At 을 광고에 싣는다 (ADR-068 을 뒤집는다). 제어판은 GET /v1/nodes 로 이
   노드 행을 읽어 Caps 와 At 을 함께 받는다. 대가는 obs 가 진행자에게 남긴
   표면 넷 — `advert.go` · `schema.sql` · `UpsertAdvert` · `NodeView` 가 W3 에
   함께 열리고(obs `business-rules.md` §5), ADR-068 이 피한 「프로토콜이
   는다」로 돌아간다

D) At 을 이번 회차에 미룬다. 제어판은 GET /v1/nodes 로 Caps 를 읽고, 시각은
   `seen_at`(마지막 광고 시각)을 「마지막 광고」로 정직하게 이름 붙여 보인다 —
   탐지 시각이 아님을 명시한다. 정확한 탐지 시각 At 은 뒤 회차로 남긴다.
   새 표면 0 이지만 카드가 요구한 「언제 잰 값인가」를 정확히는 못 채운다

X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## 닫힘 — 2026-09-09

진행자가 A 로 결정했다. `requirements/decisions.md` §2 에 행을 더해 닫았다
(「탐지 능력 읽기 (ADR-068) · At 의 출처」). 데몬이 정책 파일 옆
`<stem>.status.yaml` 에 `Caps` 와 `At` 을 쓰고 제어판이 읽는다. 광고와 Mediator
는 안 바뀐다 (ADR-068 유지). 형식·자리 세부는 panel FD 가 정한다.

기록해 둔 긴장 — decisions §2 「S3 현재 작업의 출처」가 상태 파일을 「새 기계」로
한 번 물렸으나, 그것은 lease 에 한정된 판단이다. lease 는 Mediator 가 이미
들지만 `At` 은 안 든다. FD 의 domain-entities · business-rules 가 이 근거를
다시 적는다.
