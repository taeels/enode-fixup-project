# U3 `advert` — 거절과 판정과 사유

규칙마다 **어긴 것이 실제로 거절되는가**를 잴 수 있는 형태로 적는다 — 문장 하나에
거절 사유 하나다. 형식은 `domain-entities.md`, 지나가는 길은
`business-logic-model.md`.

**에러 문자열과 로그는 영어다** (`CONVENTIONS.md` 2.1). 주석과 이 문서는 한국어다.

---

## 1. 설정 거절 다섯 (답 1=A)

`LoadLocal` 이 `mcp:` 를 읽은 뒤 검사한다. **하나라도 어기면 오류이고 노드가
안 뜬다** — `cmd/enode/main.go:112-116` 이 `cannot read config` 로 내고 `1` 로
죽는다. 새 배선이 0 인 것이 이 자리를 고른 이유다 (계획 1.1).

| | 어긴 것 | 문구 |
|---|---|---|
| R1 | 서버 이름이 비었다 | `mcp: a server name must not be empty` |
| R2 | `command` 도 `url` 도 없다 | `mcp server %q: needs command (stdio) or url (remote)` |
| R3 | `command` 와 `url` 이 둘 다 있다 | `mcp server %q: command and url are both set; a server is either stdio or remote` |
| R4 | `args` 는 있고 `command` 가 없다 | `mcp server %q: args without command` |
| R5 | `credential` 은 있고 `url` 이 없다 | `mcp server %q: credential without url` |

**다섯이 하는 일은 하나다 — 「어느 종류인가」를 한 값으로 만든다.** `mcpUp` 이
종류로 갈라 판정하므로(3절) 모호한 선언이 있으면 판정 코드가 어느 갈래로 갈지를
스스로 정하게 되고, 그 정함은 설정 파일에 안 보인다. R2 · R3 이 그 모호를
설정 시점으로 올린다. R4 · R5 는 **딸린 값이 주인 없이 남은 것**이라 사람이
지우다 만 자리다 — 조용히 무시하면 그 서버가 의도와 다른 종류로 뜬다.

**거절이 서버 이름을 든다.** `enode.yaml` 에 서버가 여럿일 수 있고, 이름이 없으면
사람이 어느 줄인지 다시 찾는다.

### 1.1 이 다섯이 못 잡는 것

```text
   command: 5       yaml 이 스칼라를 문자열로 강제해 "5" 가 된다.  모양은 맞다.
                    PATH 에도 절대경로에도 없으므로 mcpUp 이 안 뜬다고 판정하고
                    사유가 로그에 남는다 (5절).  거절이 아니라 누락으로 드러난다

   a.b 같은 이름     광고 키가 mcp.a.b 가 된다.  점은 이름의 글자이고 매칭은
                    완전 일치라 동작한다.  접두어 예약은 ADR-035 §8 의 열린
                    질문이고 이 회차가 안 정한다

   URL: (대문자)     키가 소문자로만 맞으므로 Extra 로 간다 — 버려지지 않고
                    허용목록으로 그대로 옮겨진다 (답 2=A).  그 서버는 url 이
                    없는 셈이라 R2 로 거절된다.  사람은 「URL 을 적었는데
                    needs url 이라고 한다」를 보고 대소문자를 의심한다
```

**모양 검사는 존재와 짝을 본다. 값이 참인지는 안 본다** — 그것이 3절의 판정이고
판정도 존재까지만 본다.

---

## 2. `env` 의 검증 (답 3=A)

```text
   R6   env 의 값이 환경변수 이름 꼴이 아니다
        mcp server %q: env[%q] must name an environment variable, not a value
```

꼴은 `[A-Za-z_][A-Za-z0-9_]*` 다. **키는 안 잰다** — 키는 하네스가 서버에게 줄
변수의 이름이고 그 어휘는 서버의 것이다. 값만 잰다 — 값이 **노드 환경에서
길어 올 변수의 이름**이기 때문이다 (`domain-entities.md` 3절).

**이 규칙이 닫는 것과 못 닫는 것을 함께 적는다.** 토큰이 우연히 그 꼴이면
(`GHPTOKEN` 같은 글자) 통과한다. **꼴 검사는 「값을 적지 말라」를 다 못 잰다** —
그럼에도 두는 이유는 흔한 실수(`ghp_...` · `https://...` · 공백이 든 문장)가
전부 걸리고, 걸린 사람이 문구에서 규칙을 배우기 때문이다.

---

## 3. 뜨나 판정 — `mcpUp`

```text
   stdio    command 가 절대경로로 있거나 PATH 에서 resolve 된다
   remote   credential 이름의 환경변수가 노드 환경에 있다.
            credential 이 비면 뜬다고 본다 — 인증 없는 endpoint 가 있다
   공통     실제 연결은 안 한다
```

**`mcpUp` 은 오류를 사유로 돌려준다** — 「안 뜬다」의 이유가 로그에 그대로 간다
(5절). `error` 가 `nil` 이면 뜬다.

**연결을 안 하는 근거는 순연이다** — 5분마다 전 노드가 남의 서버를 두드리면 그
서버가 죽을 때 함대가 함께 죽는다 (`requirements.md` 4.4). `tools/list` 로 내용을
아는 것도 범위 밖이다 (`constraints.md` 제외 4 · `ADR-012`).

### 3.1 못 잡는 것 셋 (`decisions.md` 6절 ⑭)

```text
   못 잡는다   자격증명이 틀렸다          환경변수 이름만 본다.  값을 안 본다
   못 잡는다   엔드포인트가 죽었다        접근을 안 한다
   못 잡는다   실행은 되나 MCP 가 아니다   CA2 의 가짜 서버 true 가 그 경우다
```

**「뜨나」는 존재이지 동작이 아니다.** 이 셋을 문장으로 적는 이유는 다음 사람이
`mcp.<이름>` 을 「검증됐다」로 읽지 않게 하려는 것이다. 설계는 이미 알고 있었고
(`scene-gates.md` 가 가짜 서버로 `true` 를 쓴다) 그 한계가 어느 문서에도 문장으로
없었다.

---

## 4. 순회와 오류 (답 4=A)

`costlyAttrs` 가 `fingerprinters` 셋을 순서대로 돌며 낸 맵을 합친다.

```text
   R7   Probe 가 오류를 내면 그 종류가 낸 것만 버리고 순회는 계속한다
        사유를 로그에 남긴다 — kind 와 err

   R8   키가 겹치면 나중 것이 이긴다.  셋의 키 공간이 안 겹치므로 오늘 이 규칙이
        걸리는 자리는 0 이다 (harness · harness.* / repo / mcp.*).
        규칙을 적는 이유는 종류가 느는 날 조용히 갈리지 않게 하려는 것이다
```

**오류가 「못 한다」가 아닌 이유** — 광고가 곧 능력이므로(`ADR-012`) 못 물어본
것을 빼고 보내면 노드가 스스로를 지운다. `Detector.refresh` 가 취소 중에 얻은
결과를 안 쓰는 것과 같은 결이다. **대가는 한 종류가 실패한 주기 동안 그 종류의
속성이 광고에서 빠지는 것이고**, 그것이 오늘 실제로 일어나지는 않는다 — 셋 중
오류를 내는 종류가 0 이다 (`domain-entities.md` 5절).

**이 유닛은 라벨 규칙을 안 건드린다.** `capabilities` 가 사람이 적은 라벨을
탐지값 위에 못 덮게 막는 줄은 그대로다 — `labels: { "mcp.probe": "1" }` 로
선언 없이 광고하는 길은 그 규칙이 이미 막지 **않는다**(탐지가 그 키를 안 냈으면
라벨이 실린다). **이것은 오늘 그대로의 성질이다** — `features.md` 3.3 의
「노드 설정에 안 적힌 MCP 는 광고되지 않는다」가 `mcp:` 절을 두고 한 말이고,
라벨은 소유자가 자기 기계에 적는 또 하나의 선언이라 같은 신뢰 경계 안이다.
`code-summary` 가 넘기는 것으로 적는다.

---

## 5. 누락 사유는 탐지마다 로그에 남는다 (답 5=A)

```text
   R9   안 뜨는 서버마다 한 줄.  수준은 Warn
        mcp server is not up; dropping it from the advertisement
        server=<이름> kind=stdio|remote err=<사유>
```

**오늘 하네스가 그렇게 한다** — `detect.go` 의
`harness is installed but not usable; dropping it from the advertisement` 가
탐지마다 나온다. 같은 자리에 같은 모양으로 두는 것이 값이다. 상태를 안 들므로
탐지기가 순수해지고, **CA2 를 재는 사람이 언제 봐도 그 줄을 본다** — 전이에서만
내면 노드를 띄운 지 한참 뒤에 게이트를 재는 사람이 아무 줄도 못 본다.

**대가는 안 뜨는 서버 하나가 하루에 288 줄을 내는 것이다.** 받아들인다 —
안 뜨는 서버는 사람이 고칠 것이고, 고칠 때까지 조용한 것보다 시끄러운 것이 낫다
(`ADR-035` §3 의 「없음이 실패보다 나쁘다」와 같은 결).

**이 줄이 CA2 의 둘째 절반을 잰다** — 실행파일을 치우고 탐지 주기를 기다리면
`mcp.<이름>` 이 광고에서 빠지고 **그 사유가 노드 로그에 있다**.

---

## 6. 광고의 문턱 (답 6=B)

```text
   R10  mcp.<이름> 은 그것만으로 능력이 아니다.  hasCapability 가 mcp. 접두를
        os · host_arch · ws 와 같이 「능력이 아닌 사실」로 센다

   R11  harness.<이름> 은 능력이다.  옛 harness 도 그대로 능력이다
```

**왜 가르나** — MCP 서버는 **하네스가 물어야 쓸 수 있는 것**이다. 하네스가 없는
기계가 `mcp:` 만 적고 광고를 내면 계약이 `requires: mcp.probe` 로 그 노드를 잡고,
실행 시점에 하네스가 없어 죽는다. 광고가 곧 능력이라는 `ADR-012` 아래서 그것은
**할 줄 모르는 것을 광고한 것**이다.

**대가를 이름으로 적는다.** `unit-of-work.md` 3절의 완료 조건이
「`cheapAttrs` · `capabilities` · `Detector`(시계)에 diff 0」이다. 이 규칙은 그
셋에 diff 를 안 내지만 **`capabilities` 가 부르는 `hasCapability` 를 고친다** —
함수 셋의 글자는 그대로이고 `capabilities` 의 **동작**이 바뀐다. 완료 조건의
글자를 지키면서 그 뜻을 벗어나는 것이므로 **어긋남으로 기록한다**
(`business-logic-model.md` 7절 · 계획 5절).

### 6.1 노드가 사라지지 않는다 — 조용해질 뿐이다

`hasCapability` 가 거짓이면 `capabilities` 가 `nil` 을 돌려주고, 광고는 그대로
나간다 — `advertise.go:174` 가 `Capabilities: nil` 로 실어 보낸다. **노드는
함대에 보이고 능력이 0 이다.** 매처는 그 노드를 못 고른다 (`Advert.Satisfies` 가
빈 목록에서 거짓이다).

```text
   R12  mcp: 를 선언했는데 능력이 하나도 안 서면 그 사유를 로그에 남긴다.  Warn
        node declares mcp servers but advertises no capability; is a harness installed?
        mcp=<선언 수>
```

**침묵이 이 답의 가장 큰 대가다.** 소유자는 `mcp:` 를 적었는데 함대에서 자기
노드가 아무것도 못 하는 것으로 보이고, 이유가 「하네스가 없다」임을 말해 주는
줄이 없으면 그 자리에서 멈춘다. R12 가 그 줄이다 — FR-3 의 「빠진 이유는 노드
로그에 남긴다」가 이 경우까지 덮는다.

---

## 7. 화면 (답 7=B)

`internal/api/ui/static/shared/fleet/format.mjs` 하나를 고친다.
**제어판(`internal/panel/page.go`)은 안 고친다** — 거기는 attrs 를 `키=값` 원문
칩으로 내는 것이 그 화면의 성질이고 `harness=claude` 도 그렇게 나온다.

```text
   R13  attributeName 이 접두를 본다
          mcp.<이름>       -> "MCP 서버 · <이름>"
          harness.<이름>   -> "실행 도구 · <이름>"
        표에 있는 키와 그 밖의 키는 오늘 그대로다

   R14  attributeValue 가 그 둘의 "1" 을 "있음" 으로 낸다.
        "1" 이 아닌 값은 그대로 낸다 — 값을 해석하지 않는다
```

**`attributeName('custom') === 'custom'` 이 그대로 초록이어야 한다**
(`tests/format.test.mjs:50`). 점이 없는 키는 접두에 안 걸린다.

**중복은 남는다.** 같은 사실이 `실행 도구 = claude` 와
`실행 도구 · claude = 있음` 두 줄로 보인다. 옛 `harness` 키를 화면에서 숨기는
길(계획 물음 7 의 C)은 안 골랐다 — 화면과 `GET /v1/nodes` 가 다른 것을 보이게
된다. **옛 키가 걷히는 날 이 중복이 함께 사라지고, 그 날은 이 회차가 안 정한다.**

---

## 8. 동작 중립 — 무엇이 그것을 판정하나

`costlyAttrs` 를 순회로 바꾸는 것은 `ADR-035:246-248` 이 「오늘 동작을 안 바꾸고
된다」로 적은 리팩터다. **그 전제가 깨졌는지는 기존 탐지 시험이 말한다.**

```text
   TestDetectSpawnsOneProcessPerHarness   caps[0].Attrs["harness"] == "claude"
   (detect_cost_test.go:41)               옛 키를 같이 싣는 한 초록이다.
                                          그리고 프로세스 수 1 을 함께 센다 —
                                          순회가 Usable 을 두 번 부르면 빨개진다

   TestDetectComesBackWhen…Cancelled      취소가 순회를 뚫고 닿는지
   probe_test.go 의 넷                    하네스 판정의 갈래들
```

**`harnesses` 가 하나라 `break` 를 걷어도 오늘 결과가 안 바뀐다**
(`harness.go:282`). 그래서 이 리팩터의 위험은 「여러 하네스」가 아니라
**순회로 옮기면서 빠뜨리는 것**이고, 특히 `repoFP` 의 `WorkspaceID` fallback
(`ADR-036`)이 그렇다 — 안 옮기면 git 없는 노드에서 `repo` 속성이 사라진다.

---

## 9. 한계와 잔여 위험

```text
   ①  검증은 모양까지다               command: 5 도 죽은 엔드포인트도 설정 시점에
                                     안 걸린다.  누락으로 드러나고 사유가 로그에 간다

   ②  env 의 참조 꼴이 미확인         ${이름} 을 하네스가 펴는지 Code Generation 이
                                     잰다.  안 펴면 env 를 안 쓰는 쪽으로 간다
                                     (domain-entities.md 3절)

   ③  Extra 는 읽기만 있다            MarshalYAML 이 없다.  오늘 설정을 읽어 다시
                                     쓰는 경로가 0 이라 잃을 것이 없고, 그 0 이
                                     깨지는 날 함께 서야 한다

   ④  R7 과 R10 이 겹치면 노드가 조용해진다
                                     mcpFP 가 오류를 내고 그 기계의 유일한 능력이
                                     MCP 였으면 그 주기의 광고가 능력 0 이다.
                                     오늘 mcpFP 는 오류를 안 내므로 일어나지 않고,
                                     종류가 느는 날 이 조합을 다시 본다

   ⑤  라벨로 선언 없이 광고할 수 있다   4절.  오늘 그대로의 성질이고 이 유닛이
                                     안 좁힌다 — 라벨도 소유자의 선언이다
```
