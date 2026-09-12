# 응용 설계 계획 — Application Design

AI-DLC Application Design 산출물 계획이다. **정본은 `requirements/harness-components/`
팩**이고, 이 단계는 팩이 일부러 안 준 것을 짓는다 — `requirements.md` 8절이
명시로 넘긴 미결 넷(D1 ~ D4)과 그 결정이 낳는 컴포넌트 겉면이다.

선행 맥락 — 팩 다섯 · `inception/requirements/requirements.md` ·
공용 `aidlc-docs/inception/reverse-engineering/` 아홉 · 실행 계획.

**이 단계는 메서드 수준의 비즈니스 로직을 안 짠다** — 그것은 유닛별 Functional
Design 이다. 여기서는 겉면(시그니처 · 책임 · 의존)까지다.

---

## 1. 착수 전 실측 — 코드를 먼저 읽었다

질문을 짓기 전에 D1 ~ D4 가 걸리는 자리를 읽었다. **넷 중 하나는 실측으로 이미
답이 나왔고, 하나는 팩의 권장값이 오늘 코드와 이름이 부딪친다.**

### 1.1 D4 는 답이 나왔다 — 링을 쓸 수 없다

`claim.go:367` 이 노드의 트랜스크립트 링을 **설정이 있으면 언제나** 연다
(`w.Ident.Config != ""`). 링 파일은 설정 파일 옆의 `<stem>.transcript` 이고
`ReadRing` 이 읽는다. 여기까지는 게이트가 쓸 수 있어 보인다.

**그런데 나를 것이 없다.** `claude.go` 의 `Argv` 가 내는 것은
`-p --output-format json` 이다 — 최종 봉투 하나뿐이고 `system/init` 줄이 아예
안 나온다. 링은 stdout 을 그대로 tee 하므로 `mcp_servers` 도 `slash_commands` 도
거기 없다.

출력 형식을 `stream-json --verbose` 로 바꾸는 것은 **짝 팩의 몫**이다
(`constraints.md` 접점 절 — 「transcript 팩은 출력 형식을 바꾼다」). 질문 3 의
답이 A 라 이 팩이 먼저 돈다.

```text
   판정   게이트는 scene-gates.md 3절의 사람 경로 그대로다.
          CA1 · CA4 · CA5 는 사람이 claude 를 한 번 직접 띄워 init 줄을 읽는다
   값     이 팩이 출력 형식을 안 건드린다 -> 짝 팩과의 접점이 줄어든다
```

D4 는 질문으로 안 낸다. 실측이 닫았다. 다만 **뒤집을 선택지**가 하나 있어서
Q5 로 묻는다 — 이 팩이 출력 형식까지 가져올 것인가.

### 1.2 D2 는 이름이 부딪친다

`decisions.md` 2절의 권장값은 「`Detector` 인터페이스를 세우고 하네스 · MCP
둘을 그 종류로 둔다」(`ADR-035` §4.2)다. 그런데 **`internal/enode` 에 이미
`Detector` 가 있다** (`detector.go` · `ADR-068`) — 그것은 종류가 아니라
**시계**다. 비싼 탐지를 광고와 다른 주기로 도는 고루틴이다.

그래서 `ADR-035` §4.2 가 지키려던 값은 **이미 서 있다**.

```text
   Detector.Capabilities()   외부 프로세스를 안 띄운다.  마지막 값을 읽는다
   Detector.refresh()        잠금 밖에서 costlyAttrs 를 돈다
   광고 루프                  Capabilities() 만 부른다 — 탐지에서 멈춰도 하트비트가 산다
```

`requirements.md` 4.4 의 「광고 루프는 탐지를 직접 부르지 않는다」는 **새로 지을
것이 아니라 안 깨뜨릴 것**이다. 남은 물음은 좁다 — MCP 를 `costlyAttrs` 안의
함수 하나로 더하나, 종류 인터페이스를 새로 세우나. Q3 으로 묻는다.

### 1.3 D1 은 인터페이스 모양보다 실패 규칙이 먼저다

`runner.go:63` 의 계장 임시 디렉터리는 **if 블록 안에서만 산다**.

```go
   if tmp, err := os.MkdirTemp("", "enode-inst-*"); err == nil {
       defer os.RemoveAll(tmp)
       ...
       flags, err := h.Instrument(tmp, self, a)
       // 실패해도 기동은 한다 — 계장은 보조다
   }
   fixed := map[string]string{"OUT": ..., "IN": ...}
   for k, v := range h.Fixed() { ... }   // tmp 가 안 보인다
```

`tmp` 를 블록 밖으로 끌어올리는 것은 기계적이다. **진짜 결정은 실패 규칙이다** —
FR-1 이 「계장 실패와 독립이다. 계장이 실패해도 홈은 갈아끼워져 있고 비어
있다」를 확정으로 걸었는데, 오늘 코드에는 실패가 두 종류다.

```text
   MkdirTemp 실패      디렉터리 자체가 없다.  가짜 홈이 앉을 자리가 없다
   Instrument 실패     디렉터리는 있다.  훅만 안 심긴다
```

FR-1 이 말하는 「계장 실패」는 뒤쪽이다. 앞쪽은 팩도 요구도 안 적었다. Q2 로
묻는다.

### 1.4 인증 경로가 둘이라는 것 — 가짜 홈이 하나만 건드린다

`hook.go:346` 의 `gatewayAuthFields()` 가 사람의 `~/.claude/settings.json` 에서
`apiKeyHelper` 와 `env` 만 골라 우리 settings 에 얹는다. 이것은
`os.UserHomeDir()` 를 읽으므로 **`CLAUDE_CONFIG_DIR` 을 바꿔도 안 끊긴다.**

```text
   게이트웨이 노드   apiKeyHelper + env 가 우리 settings 로 들어온다.  가짜 홈과 무관
   OAuth 노드       ~/.claude/.credentials.json 을 하네스가 CLAUDE_CONFIG_DIR 에서 찾는다.
                    가짜 홈이 끊는 유일한 경로 — FR-1 의 복사가 이것을 받는다
```

**CA1 이 노드 둘에서 각각 돌아야 하는 이유가 여기 있다.** 두 경로가 서로 다른
코드로 산다.

---

## 2. 이 단계가 정하는 것 · 안 정하는 것

**정한다.**

```text
   Harness 인터페이스의 모양     Fixed 가 계장 디렉터리를 아는 길 (D1)
   실패 규칙의 경계             어느 실패가 단계를 죽이고 어느 실패가 보조로 남나
   허용목록·팩의 소유자          결정(정책)과 쓰기(파일)가 어느 컴포넌트에 갈리나 (D3)
   MCP 탐지가 앉는 자리          costlyAttrs 안인가 새 종류인가 (D2)
   새 겉면의 시그니처            노드 선언 읽기 · 뜨나 · 허용목록 해소 · 팩 펴기 · 기록
   의존과 통신                  네 경로 사이의 호출 방향.  임포트 금지 넷을 안 깬다
```

**안 정한다** — 건드리면 옮겨 적기가 된다.

```text
   팩이 값으로 닫은 것       CLAUDE_CONFIG_DIR 위치 · 등호 플래그 · 합집합 순서 ·
                            광고 키 모양 · 실패 사유 문구 · 탐지 주기 5분.
                            decisions.md 2절이 진다
   Functional Design 몫      팩 tar 의 크기·개수 상한 실제 값 · mcp.json 의 필드 ·
                            enode.yaml mcp: 절의 필드 · HarnessResult 직렬화
   Units Generation 몫       유닛 분해와 파일 행렬.  여기서 안 가른다
```

---

## 3. 산출물 계획 (체크박스)

규칙이 요구하는 필수 산출물 다섯이다. **4절 질문의 답이 들어온 뒤 생성한다.**

- [x] `application-design/components.md` — 컴포넌트 정의와 책임
  - [x] 기존 컴포넌트 넷의 확장 책임 — `Harness` 어댑터 · `runHarness` ·
        `Detector`/`costlyAttrs` · `Local`
  - [x] 새 겉면 다섯 — 노드 MCP 선언 · 뜨나 판정 · 허용목록 해소 ·
        팩 펴기(안전한 tar) · 기록
  - [x] 컴포넌트마다의 인터페이스 요약과 **실패 등급** (치명 · 보조)
- [x] `application-design/component-methods.md` — 메서드 시그니처 (비즈 규칙은 FD)
  - [x] `Harness` 인터페이스의 최종 모양 (Q1 · Q4 의 답이 정한다)
  - [x] `internal/enode` 의 새 함수 겉면 — 입출력 타입과 오류 타입
  - [x] `internal/contract` — `agentKeys` 추가와 `Grammar` 의 새 필드 자리
  - [x] `cmd/iapadapter` · `cmd/runctl` 의 설정·예시 겉면
- [x] `application-design/services.md` — 오케스트레이션
  - [x] 한 단계의 순서 — 계장 디렉터리 · 가짜 홈 · 팩 펴기 · 허용목록 해소 ·
        플래그 조립 · exec · 기록. **어디서 죽을 수 있나를 함께 적는다**
  - [x] 탐지의 순서 — `Detector` 주기 · `costlyAttrs` · 광고 조립
  - [x] 팩의 순서 — 팩 단계(명령)가 `$OUT/pack` 에 내고 `in.from` 으로 잇는다
- [x] `application-design/component-dependency.md` — 의존과 통신
  - [x] 네 경로의 호출 방향 그림과 텍스트 대안
  - [x] 임포트 금지 넷이 안 깨짐을 표로 확인
  - [x] **짝 팩(transcript)과 겹치는 파일 셋의 접점 표**
- [x] `application-design/application-design.md` — 위 넷의 통합본
- [x] 검증 — 임포트 금지 · 차단 게이트 다섯 · 표기(`emphasis-check.py`) ·
      팩의 「안 하는 것」 여덟 범주와 대조

---

## 4. 결정이 필요한 것 — `[Answer]:` 태그

**다섯이다.** 각 질문에 권장(A)과 근거를 붙였다. `[Answer]:` 뒤에 고른 기호를
적어 주세요. 직접 서술해도 된다.

---

### Q1. `Fixed()` 가 계장 임시 디렉터리를 어떻게 아나 (D1)

`CLAUDE_CONFIG_DIR=<계장 디렉터리>/home` 을 **값으로 박는다**는 것이 FR-1 의
확정이다. 그런데 `Fixed() map[string]string` 은 인자가 없다.

```text
   A (권장)  인터페이스를 Fixed(dir string) map[string]string 으로 바꾼다.
             runHarness 가 계장 디렉터리를 넘기고, claude.go 가 그 아래 home 을
             가리키는 값을 돌려준다

   B         Fixed() 를 그대로 두고 runHarness 가 홈 변수만 합친다.
             인터페이스가 안 바뀐다

   C         Instrument 가 flags 와 함께 env 도 돌려준다.
             홈을 짓는 쪽이 그 주소도 말한다
```

근거 — **A 가 유일하게 셋을 다 지킨다.**

```text
   하네스별 이름이 어댑터에만 있다   B 는 runner.go 가 CLAUDE_CONFIG_DIR 이라는
                                   claude 전용 이름을 알게 된다.  그 파일의 주석이
                                   "구조적인 것(OUT·IN)과 하네스가 정하는 것(Fixed)을
                                   한 자리에서 합친다" 로 그 갈래를 이미 적어 두었다
   계장 실패와 독립이다              C 는 Instrument 가 실패하면 홈이 안 갈린다.
                                   FR-1 의 확정을 정면으로 어긴다
   비용이 거의 0 이다                dir 은 이미 Instrument(dir, ...) 로 인터페이스가
                                   아는 개념이다.  두 번째 하네스가 없어 이주 비용도 0 이다
```

**[Answer]:** A

---

### Q2. 계장 임시 디렉터리를 못 만들면 (D1 의 실패 규칙)

오늘은 `os.MkdirTemp` 가 실패하면 계장을 통째로 건너뛰고 **그대로 기동한다.**
가짜 홈이 거기 앉으면 그 경로는 「개인 설정과 계정 커넥터가 그대로 보이는
하네스를 띄운다」가 된다.

```text
   A (권장)  단계를 실패로 보고한다.  하네스를 안 띄운다.
             사유는 cannot create the instrumentation directory

   B         오늘 그대로 — 계장 없이 기동한다.  가짜 홈도 없다

   C         홈만 따로 짓는다 (os.MkdirTemp 를 두 번).  하나가 실패해도 다른 하나로 간다
```

근거 — A 가 `requirements.md` 4.2 의 SECURITY-09(「가짜 홈이 기본값이다.
여는 값을 두지 않는다」)와 SECURITY-15(「실패는 닫히는 쪽으로」)에 맞는다.
B 는 **디스크가 찬 노드에서 조용히 격리가 풀리는 길**이고, 그것이 이 팩이
사내 실측에서 고치려는 「거꾸로 열린다」와 같은 종류의 실패다.
C 는 실패 확률을 줄이지만 정리 경로가 둘이 되고, 애초에 `MkdirTemp` 가
실패하는 기계는 두 번 해도 실패한다.

**A 를 고르면 오늘 동작이 바뀐다** — 이 줄은 `decisions.md` 에 행으로 적는다.

**[Answer]:** A

---

### Q3. MCP 탐지를 어디에 앉히나 (D2)

> **개정 (2026-09-12).** 이 답이 A 로 닫혔다가 설계 검증에서 **B 로 뒤집혔다.**
> A 의 근거 둘이 다 틀렸다 — 「§4.2 가 지키려던 값은 `ADR-068` 이 이미 세웠다」는
> 오독이고(§4.2 의 논거는 등록부다), 「이름이 부딪친다」는 다른 이름을 달면
> 사라진다. `ADR-035:246-248` 이 이 회차를 시점으로 지목한 문장도 안 읽혔다.
> **지금 값은 `decisions.md` 6절 ⑤ 다** — 아래 본문은 기록이다.

1.2 가 적은 대로, `decisions.md` 2절의 권장값(「`Detector` 인터페이스를 세운다」)은
오늘 코드의 `Detector`(시계)와 이름이 부딪치고, 그 권장이 지키려던 값은 이미 서 있다.

```text
   A (기록)  costlyAttrs 안에 mcpAttrs(ctx, l, log) 하나를 더한다.
             새 타입 0.  Detector 는 시계로 그대로 둔다

   B         CostlyDetector 인터페이스를 세우고 harness · repo · mcp 셋을
             그 종류로 둔다.  ADR-035 §4.2 의 글자 그대로

   C         A 로 하되 mcp 만 별도 파일(mcp_detect.go)로 가른다.
             타입은 안 늘리고 파일로만 가른다
```

근거 — A · C 는 **한 이름이 두 가지를 뜻하는 것을 막는다.** 비싼 사실은 오늘
셋이고(하네스 · 저장소 · MCP) 셋의 모양이 서로 다르다 — 하네스는 어댑터 목록을
돌고, 저장소는 호출 하나이고, MCP 는 설정 맵을 돈다. 모양이 다른 셋 위에 인터페이스를
씌우면 등록부가 하나 늘고 얻는 것이 없다. B 는 `ADR-068` 이 `detector.go` 를
만들기 전에 쓰인 문장을 두 번 구현하는 일이 된다.

**A 나 C 를 고르면 권장값에서 벗어난다** — `decisions.md` 에 행으로 적는다.
B 를 고르면 권장값 그대로이고 적을 것이 없다.

**[Answer]:** A

---

### Q4. 허용목록과 팩을 누가 쥐나 — 그리고 실패를 어떻게 올리나 (D3)

이 팩은 실행 직전에 셋을 한다 — **팩 tar 를 가짜 홈에 편다 · 요청한 MCP 이름을
세 출처에서 해소한다 · 허용목록 파일을 쓰고 플래그를 낸다.** 오늘 그 자리에 있는
것은 `Instrument` 하나이고, 그 실패는 **삼켜진다** (「실패해도 기동은 한다 —
계장은 보조이고 진짜 안전망은 워크스페이스 diff 다」).

**그런데 이 셋은 보조가 아니다.** FR-2 는 「요청한 이름이 셋 어디에도 없으면
하네스를 안 띄우고 단계를 실패로 보고한다」이고, SEC-A 는 「팩 tar 의 거부는
조용하지 않다 — 그 단계를 실패로 보고한다」다.

```text
   A (권장)  결정과 쓰기를 가른다.
             resolveMCP(...) 가 세 출처를 합쳐 서버 목록을 내거나 오류를 낸다 —
             파일을 안 만지므로 시험이 싸다.  runHarness 가 exec 전에 부르고
             오류면 그 자리에서 단계를 실패로 낸다.
             해소된 목록과 팩은 Instrument 가 파일로 쓴다 (형식은 하네스의 것)

   B         Instrument 하나가 셋을 다 하고, Instrument 의 오류를 전부 치명으로 올린다.
             시그니처가 안 바뀐다.  다만 훅 쓰기 실패까지 단계를 죽인다 —
             오늘의 "보조" 판단이 뒤집힌다

   C         Harness 에 Prepare(dir, j Job) ([]string, error) 를 새로 더하고
             정책을 거기 둔다.  Instrument 는 보조로 남는다.
             실패 등급이 시그니처에 보인다.  인터페이스 메서드가 하나 는다
```

근거 — A 가 **실패 등급을 타입이 아니라 호출 순서로 가른다.** 정책은 exec 전에
독립으로 돌고 오류가 곧 단계 실패다. 파일 쓰기는 그대로 어댑터 안에 남아
「하네스마다 심는 방법이 다르다」가 유지된다. B 는 한 줄도 안 늘지만 훅 실패의
기존 판단(모델 협조가 필요 없는 diff 가 진짜 안전망이다)을 말없이 뒤집는다.
C 는 가장 정직하지만 두 번째 하네스가 없는 지금 메서드를 늘리는 값이 약하다
(`agent-runtime` R6).

**[Answer]:** A

---

### Q5. 이 팩이 하네스 출력 형식을 가져오나 (D4 의 뒤집기)

1.1 의 실측 — 오늘 `Argv` 가 `--output-format json` 이라 `system/init` 줄이
안 나오고, 그래서 게이트의 `mcp_servers` · `slash_commands` 는 **사람이 claude 를
한 번 직접 띄워** 읽어야 한다 (`scene-gates.md` 3절).

```text
   A (권장)  안 가져온다.  출력 형식은 짝 팩(transcript)의 것으로 둔다.
             CA1 · CA4 · CA5 는 사람 경로로 잰다

   B         이 팩이 --output-format stream-json --verbose 로 바꾼다.
             게이트가 logs/ 와 트랜스크립트 링에서 init 줄을 읽어 싸진다

   C         게이트 전용으로만 바꾼다 — 계약이나 노드 설정에 스위치를 둔다
```

근거 — A 가 `constraints.md` 의 접점 표(「이 팩은 MCP 플래그를 더한다 ·
transcript 팩은 출력 형식을 바꾼다」)와 질문 3 의 답(A — 이 팩이 먼저)을 지킨다.
B 는 게이트를 싸게 만들지만 `Decode` 와 봉투 파싱이 함께 흔들리고
(`ParseClaude` 는 마지막 JSON 객체를 집는다), 그 흔들림이 짝 팩과 같은 파일에서
동시에 일어난다 — 진행자가 직렬로 병합해야 하는 자리를 하나 더 만든다.
C 는 스위치가 하나 늘고, 「게이트가 재는 것이 실제 실행과 다른 경로」가 되어
게이트의 값 자체를 깎는다.

**A 를 고르면 눈 검증 셋의 비용이 남는다** — 그것을 보류로 안 넘긴다는 것이
`scene-gates.md` 4절의 확정이다.

**[Answer]:** A

---

## 5. 답 (2026-09-11)

사용자 답은 **「전부 권장대로 할게」** 였다. Q1 ~ Q5 가 모두 A 다. 모호한 답이
없어 후속 질문을 안 냈다 (규칙 8 · 9 의 게이트 통과).

```text
   Q1 = A   Fixed(dir string) map[string]string 으로 인터페이스를 바꾼다
   Q2 = A   계장 디렉터리를 못 만들면 단계 실패.  하네스를 안 띄운다
   Q3 = A   costlyAttrs 안에 mcpAttrs 하나.  새 타입 0 -- 2026-09-12 에 B 로 뒤집힘
   Q4 = A   결정(resolve)과 쓰기(write)를 가른다.  정책 실패는 exec 전에 단계를 죽인다
   Q5 = A   출력 형식을 안 건드린다.  게이트는 사람 경로
```

**설계가 A 를 완성하며 더 정한 것 둘** — 답이 안 닫은 자리이고 산출물이 근거를
적었다 (`application-design.md` 4절).

```text
   ①  Instrument 의 오류를 등급으로 가른다.  훅 실패는 오늘처럼 보조로 삼키고,
      가짜 홈·허용목록·팩의 쓰기 실패만 errComponents 로 감싸 치명으로 올린다
      -- 개정 (2026-09-12): 방향이 뒤집혔다.  기본이 치명이고 errAux 로 감싼
         훅 설정 쓰기 하나만 보조다.  지금 값은 decisions.md 6절 ⑨ · ⑪ 이다
   ②  Instrument 를 언제나 부른다.  오늘은 os.Executable() 이 빈 문자열이면
      통째로 건너뛰는데, 그 경로로 가면 허용목록이 조용히 안 쓰인다
```

---

## 6. 답이 들어온 뒤의 순서

```text
   ①  답 다섯을 읽고 모호한 것이 있으면 후속 질문을 이 문서에 더한다 (규칙 8·9)
   ②  3절 체크박스대로 산출물 다섯을 낸다
   ③  권장을 벗어난 답(Q2 의 A · Q3 의 A 나 C)을 decisions.md 행으로 모은다
   ④  audit.md 에 답 원문을 그대로 싣고 aidlc-state.md 를 갱신한다
   ⑤  승인 뒤 Units Generation — 파일 행렬을 낸다
```
