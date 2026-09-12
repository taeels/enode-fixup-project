# U1 `isolation` — Functional Design 계획

**유닛** `isolation` · **브랜치** `unit/isolation` · **담당** taeels ·
**닫는 게이트** CA1 (+ CA0) · **선행** 없음

정본 입력은 `aidlc-docs/v3-run-harness-components/inception/application-design/`
의 넷과 `requirements/harness-components/` 팩이다. 이 계획은 그 위에서 **비즈니스
규칙과 형식**만 짓는다 — 코드는 다음 단계다.

---

## 0. 이 단계가 닫는 것과 안 닫는 것

```text
   닫는다    mcp.json 의 필드 표현과 빈 파일의 모양
             Instrument 의 쓰기 순서와 불변식 셋을 규칙으로
             실패 등급을 자리마다의 규칙으로 (기본 치명 · errAux 하나)
             logs/ 허용목록의 실제 표현 — 껍데기 한 줄이 무엇으로 적히나
             ParseClaude 의 type 검사와 Message 에 싣는 것
             stream-json 사건 종류의 실측 목록

   안 닫는다  PackLimits 의 값          U5 의 FD
             enode.yaml mcp: 절의 필드  U3 의 FD
             resolveComponents 의 출처 합치기와 거절  U4
             HarnessResult 직렬화       U5 의 FD
             시험 목록과 파일별 diff     이 유닛의 Code Generation
```

---

## 1. 실측 — `stream-json` 의 사건 종류 (2026-09-12)

`decisions.md` 6절 ⑱ 이 「**실물을 아직 못 봤다** — U1 이 실측하고 사건 종류
목록을 `decisions.md` 에 행으로 적는 것까지가 그 유닛의 일이다」로 이 단계에
숙제를 남겼다. FD 를 짓기 전에 먼저 쟀다.

측정은 `claude 2.1.266` 에 `-p --output-format stream-json --verbose` 로 두 번
돌렸다 — 도구가 성공하는 줄기 하나와 도구가 실패하는 줄기 하나다. 원문은
저장소에 안 싣는다 (개인 홈의 서버 이름과 절대경로가 들어 있다). 여기 적는 것은
**모양**이다.

### 1.1 본 사건 종류 — 다섯

| `type` | `subtype` | 본문이 어디 있나 | 허용목록 판정 |
|---|---|---|---|
| `system` | `init` | `mcp_servers` · `slash_commands` · `tools` · `cwd` · `memory_paths` | **전문** |
| `system` | `hook_started` · `hook_response` · `thinking_tokens` | `hook_response` 에 `stdout` · `stderr` · `output` | 껍데기 |
| `assistant` | (없음) | `message.content[]` 의 `text` · `thinking` · `tool_use.input`, 그리고 **최상위 `wire_tool_inputs`** | 껍데기 |
| `user` | (없음) | `message.content[]` 의 `tool_result.content`, 그리고 **최상위 `tool_use_result`** | 껍데기 |
| `rate_limit_event` | (없음) | `rate_limit_info` | 껍데기 |
| `result` | `success` | `result` · `permission_denials` · `usage` · `modelUsage` | **전문** |

### 1.2 실측이 뒤집은 것 — 첫 줄이 `init` 이 아니었다

`decisions.md` 6절 ⑱ 과 `unit-of-work.md` 1절이 「`init` 이 나오는 경로에서 첫
줄은 언제나 `system/init` 이다 — 게이트가 `head -1` 로 읽는다」로 못 박았다.
**실측에서 첫 줄은 `system/hook_started` 였다.** 개인 홈의 SessionStart 훅
넷이 `init` 앞에 사건 여덟을 냈다.

```text
   0-3   system/hook_started     SessionStart 훅 넷
   4-7   system/hook_response    그 넷의 답.  stdout 과 output 이 들어 있다
   8     system/init             아홉째 줄이다
```

가짜 홈 아래서는 `--setting-sources ""` 가 개인 설정을 끊고 우리 `settings.json`
에는 Stop 훅 하나뿐이라 SessionStart 훅이 안 뜬다. 그래서 오늘의 설계로도
`init` 이 첫 줄일 **가능성은 높다**. 다만 그것은 **우리 설정 파일의 내용에 기댄
전제**이고 하네스가 준 보증이 아니다. 질문 3 이 그 전제를 어디에 둘지 묻는다.

### 1.3 실측이 굳힌 것 셋

```text
   mcp_servers 는 배열이다      [{"name":"...","status":"connected"}] 모양이다.
                                맵이 아니다.  CA1 의 「비어 있다」는 [] 다.
                                status 는 connected · needs-auth 를 봤다

   도구 이름과 성공 여부의 자리   이름은 assistant 의 message.content[] 중
                                type=="tool_use" 의 name.
                                성공 여부는 user 의 message.content[] 중
                                type=="tool_result" 의 is_error —
                                **성공하면 그 키가 아예 없다** (거짓이 아니라 없음)

   본문이 블록 밖에도 있다        assistant 의 wire_tool_inputs 는 도구 입력을
                                최상위에 통째로 한 번 더 싣고, user 의
                                tool_use_result 는 도구 결과를 최상위에 싣는다.
                                content 블록만 지우면 안 새는 것이 아니다 —
                                **허용목록을 짓는 쪽이 옳다는 증거다**
```

---

## 2. 물음 다섯

답을 `[Answer]:` 뒤에 적는다. 권장이 있는 물음은 권장을 **A** 에 둔다.

### Question 1

`logs/` 에 남길 **껍데기 한 줄을 무엇으로 적나.** 전문으로 남는 셋(`system/init` ·
최종 `result` · stderr)은 원문 줄이라 JSON 이다. 껍데기가 그 사이에 섞인다.

A) **새 JSON 객체를 지어 한 줄로 적는다.** `{"type":"assistant","tool":"Read"}` ·
`{"type":"user","tool_result":true,"ok":false}`. 파일 전체가 한 줄 한 객체로 남고,
남길 필드를 **짓는 쪽이 허용목록**이라 원본에 새 필드가 생겨도 안 샌다

B) **텍스트 한 줄로 적는다.** `event assistant tool_use Read` 처럼. 사람이 읽기
쉽지만 파일 안에 JSON 줄과 텍스트 줄이 섞여 뒤에 기계로 읽을 때 갈린다

C) **원본 객체에서 허용 필드만 지우고 남긴다.** 줄 모양이 원본과 같아 기존 도구가
그대로 읽지만, 지우는 쪽이라 하네스가 필드를 늘리면 **열리는 쪽으로 틀린다**

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2

껍데기가 **무엇까지 담나.** ⑱ 의 글자는 「사건 종류 · 도구 이름 · 성공 여부」다.
실측이 보인 자리는 1.3 에 있다.

A) **글자 그대로 셋만.** 사건 종류(`type` 과 `system` 의 `subtype`) · 도구 이름 ·
성공 여부. 블록 종류도 토큰 수도 안 남긴다. 가장 닫힌 쪽이다

B) 거기에 **블록 종류 목록**을 더한다 — `["thinking","text"]` 처럼. 본문은 없고
모양만 남아 「생각만 하고 끝난 턴」과 「답을 낸 턴」이 갈린다

C) 거기에 **`usage` 의 토큰 수**도 더한다. 예산 신호가 사건마다 남아 `result` 의
합계와 대조된다. 다만 사건마다 숫자가 늘어 줄이 길어진다

D) Other (please describe after [Answer]: tag below)

[Answer]: C

### Question 3

**「첫 줄은 `init`」 전제를 어디에 두나.** 1.2 가 그 전제가 하네스의 보증이
아니라 **우리 설정 파일의 내용**에 기댄 것임을 보였다.

A) **게이트가 순서에 안 기댄다.** `head -1` 대신 첫 `system/init` 줄을 골라 읽게
게이트 명령을 고치고(`grep -m1` 또는 그에 준하는 것), 불변식은 「`init` 줄이
`logs/` 에 있다」로 약하게 둔다. `scene-gates.md` 3절의 명령이 바뀐다

B) **「첫 줄은 `init`」을 불변식으로 유지한다.** 우리 `settings.json` 에
SessionStart 훅이 없다는 사실이 그 근거이고, U1 의 시험이 「우리가 쓰는 설정에
Stop 외의 훅이 0 이다」를 잰다. 게이트 명령은 안 바뀐다

C) **`runner.go` 가 순서를 만든다.** 선별할 때 `init` 줄을 맨 앞으로 옮겨 적어
`head -1` 을 우리 코드가 참으로 만든다. 대가는 `logs/` 의 줄 순서가 하네스가 낸
순서와 갈리는 것이다

D) Other (please describe after [Answer]: tag below)

[Answer]: B

### Question 4

**걷었음을 표시하는 줄을 어디에 두나.** ⑱ 은 「그 줄은 첫 줄이 아니다」만
정했다.

A) **stdout 선별 결과의 맨 끝 · stderr 앞**에 한 줄. `logs/` 는 오늘도
stdout 다음에 stderr 라 그 경계에 놓인다. 세어 적는다 — 걷은 사건 수와 바이트 수

B) **`init` 줄 바로 뒤** (둘째 줄). 읽는 사람이 파일 머리에서 바로 본다. 다만
`init` 이 없는 경로에서는 자리가 흔들린다

C) **파일 맨 끝** (stderr 뒤). 언제나 마지막 줄이라 찾기 쉽다. 다만 stderr 가
길면 사람이 안 읽는 자리로 간다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5

**U1 이 `mcp.json` 직렬화를 어디까지 짓나.** U1 의 제품 경로는 언제나 빈
목록이다 — 요청도 팩도 없는 경우만 다루기 때문이다 (`unit-of-work.md` 1절).

A) **직렬화를 전부 짓는다.** `MCPServer` 의 필드가 그대로
`{"mcpServers":{...}}` 로 나가고, U1 의 시험이 `Components{Servers: ...}` 를
직접 넣어 잰다. U4 는 **정하는 일**만 가져오고 쓰는 쪽을 안 건드린다

B) **빈 봉투만 짓는다.** `{"mcpServers":{}}` 하나를 쓰고, 서버 하나를 적는 코드는
U4 가 가져온다. U1 이 안 도는 코드를 안 만든다

C) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## 3. 답을 받은 뒤 낼 산출물

`construction/isolation/functional-design/` 아래 셋이다. **셋 다 냈다** (2026-09-12).

- [x] **`domain-entities.md`** — 이 유닛이 드는 형식
  - [x] `MCPServer` — 필드와 `mcp.json` 으로의 사상 (질문 5 가 범위를 정한다)
  - [x] `Components` — U1 이 채우는 것과 안 채우는 것 (`Pack` 은 언제나 nil)
  - [x] 가짜 홈의 자리표 — `<dir>/home` 아래 다섯 경로와 각각의 권한
  - [x] `logs/` 의 줄 종류 셋 — 전문 · 껍데기 · 표시 (질문 1 · 2 · 4)
  - [x] `claudeEnvelope` 가 `type` 을 읽게 되는 것과 `Message` 에 싣는 값 (⑯)

- [x] **`business-logic-model.md`** — 순서와 흐름
  - [x] `runHarness` 의 ① ~ ⑩ — 치명이 전부 exec 앞에 모이는 순서
  - [x] `Instrument` 의 쓰기 순서 다섯과 조기 반환하지 않는 흐름
  - [x] `logs/` 선별 파이프라인 — 줄 나누기 · 판정 · 짓기 · 표시 · stderr 이어붙이기
  - [x] `ParseClaude` 의 판정 흐름 — `type` 검사가 switch 앞에 서는 자리
  - [x] 링 tee 를 끄는 자리 (⑲) 와 명령 단계 tee 가 그대로인 것

- [x] **`business-rules.md`** — 규칙과 불변식
  - [x] 실패 등급 규칙 — 기본 치명 · `errAux` 한 자리 · 자격증명의 세 갈래
  - [x] 불변식 셋 (`application-design.md` 4.3) 을 잴 수 있는 문장으로
  - [x] 허용목록 규칙 — 모르는 사건 종류가 닫히는 쪽으로 떨어지는 규칙
  - [x] 봉투 고르기 규칙 — `logs/` 는 `type == "result"`, `ParseClaude` 는
        `lastJSONObject` + `type` 검사 (자리가 둘이고 규칙이 다르다)
  - [x] 순서 규칙 — `init` 줄과 표시 줄 (질문 3 · 4 의 답)
  - [x] 오류 문구 규칙 — U1 이 내는 치명의 문구와 `Message` 에 안 싣는 것

- [ ] **`decisions.md` 6절에 실측 행 하나** — 1절의 사건 종류 표.
      팩 문서라 **이 단계의 승인 뒤에** 싣는다. 남은 하나다

---

## 4. 이 단계가 안 만드는 것

```text
   코드         한 줄도 안 쓴다.  다음 단계다
   시험 목록     Code Generation 계획이 든다
   게이트 명령    질문 3 의 답이 A 면 scene-gates.md 3절이 바뀐다 —
                그 수정도 Code Generation 에서 한다
```

---

## 5. 파장 — 이 유닛 밖으로 가는 것

```text
   짝 팩 (v3-run-transcript)   claude.go 의 Argv 와 runner.go 둘이 겹친다.
                              이 팩이 먼저 전부 들어간다 (파일 행렬)
   U4 · U5                    질문 5 의 답이 쓰는 쪽의 경계를 정한다
   scene-gates.md             질문 3 의 답이 A 면 CA1 · CA4 · CA5 의 명령이 바뀐다
   decisions.md               1절의 실측 행.  6.1 의 열쇠 표에는 안 걸린다 —
                              뒤집는 값이 아니라 더하는 기록이다
```
