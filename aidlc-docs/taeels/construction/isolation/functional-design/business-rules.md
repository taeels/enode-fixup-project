# U1 `isolation` — 규칙과 불변식

**잴 수 있는 문장으로만 적는다.** 못 재는 문장은 다음 사람에게 규칙이 아니라
분위기로 간다.

---

## 1. 실패 등급 — 기본이 치명이다

`components.md` 3절의 표를 U1 의 자리마다의 규칙으로 옮긴 것이다.

| | 자리 | 실패 | 등급 |
|---|---|---|---|
| R1 | 계장 디렉터리 생성 | `MkdirTemp` 오류 | **치명** |
| R2 | 가짜 홈 짓기 | `<dir>/home` 의 `MkdirAll` 오류 | **치명** |
| R3 | 훅 설정 쓰기 | Marshal · WriteFile 오류 | 보조 (`errAux`) |
| R4 | 허용목록 쓰기 | `<dir>/mcp.json` 의 WriteFile 오류 | **치명** |
| R5 | 자격증명 복사 | 원본이 **없다** | 정상 (아무것도 안 한다) |
| R6 | 자격증명 복사 | 원본이 **있는데** 읽기 · 쓰기 실패 | **치명** |
| R7 | 기준 시각 파일 | `writeStamp` 오류 | 이 표 밖 |

**R7 이 표 밖인 이유** — `writeStamp` 는 `runner.go:71-76` 에서 `Instrument`
**앞**에 불리고 그 오류가 `== nil` 비교로 이미 버려진다. 등급을 매길 자리가
구조적으로 아니다 (⑪).

**R5 의 「없다」를 넓게 읽는다.** `os.UserHomeDir()` 이 오류를 내는 노드
(`HOME` 이 없는 환경)도 **없는 것과 같이 본다.** 홈을 못 읽는 노드에는 OAuth
자격증명이 있을 수 없고, `gatewayAuthFields()` 가 이미 같은 판단을 한다
(`hook.go:359-362`). **없음과 실패를 가르는 선은 「원본 파일에 닿았는가」다.**

**시험이 잰다** — R1 · R2 · R4 · R6 을 실패시키면 단계가 실패로 보고되고,
R3 을 실패시키면 단계가 **돈다**.

---

## 2. `errAux` — 감싸는 자리는 하나다

```text
   감싼다      hook.go 의 훅 설정 Marshal 과 WriteFile.  그 둘이 한 자리다
   안 감싼다   Instrument 안의 나머지 오류 반환 전부
   가르는 법   errors.Is(err, errAux) 하나
```

**시험이 개수를 잰다** — `Instrument` 가 내는 오류 중 `errors.Is(err, errAux)`
가 참인 경로가 **훅 설정 하나**임을 잰다. 「둘이다」로 적었던 앞 판은 잴 수가
없었다 (⑪ 이 고쳤다).

**기본이 치명인 이유는 실수의 방향이다.** 기본이 보조이면 감쌀 자리를 하나
빠뜨리는 실수가 **열리는 쪽으로** 틀린다 — 허용목록이 안 쓰였는데 단계가 돌고
`exit 0` 으로 성공이 봉인된다. 기본이 치명이면 그 실수가 **닫히는 쪽으로**
틀린다 (SECURITY-15 · `ADR-034` §2.1).

---

## 3. 불변식 셋 — 시험이 직접 잰다

`application-design.md` 4.3 의 셋을 잴 수 있는 문장으로 옮긴다.

### 3.1 `<dir>/home` 을 가장 먼저 만든다

```text
   잰다   Instrument 가 오류로 나간 뒤에도 <dir>/home 이 디렉터리로 있다.
          R3 을 실패시킨 경우와 R4 를 실패시킨 경우 둘 다에서
   왜     CLAUDE_CONFIG_DIR 이 없는 경로를 가리킬 때 하네스가 무엇을 하는지
          실측이 없다.  진짜 홈으로 되돌아가면 격리가 조용히 풀린다
```

### 3.2 보조 실패에도 이미 얻은 플래그를 돌려준다

```text
   잰다   R3 을 실패시켜도 돌려받은 플래그에
          --strict-mcp-config 와 --setting-sources "" 가 **둘 다** 있다
   안 잰다  --settings 가 있는지.  그 파일을 못 썼으므로 없는 것이 맞다
          (business-logic-model.md 2.1)
```

### 3.3 보조 오류로 조기 반환하지 않는다

```text
   잰다   R3 을 실패시켜도 <dir>/mcp.json 이 쓰여 있다
   왜     조기 반환하면 허용목록이 아예 안 쓰이고 치명도 안 난다 —
          3절 표가 치명으로 잡으려던 것이 보조 경로로 되돌아온다
```

**셋이 한 시험에서 같이 걸린다** — R3 을 실패시킨 뒤 `<dir>/home` 이 있고
`<dir>/mcp.json` 이 있고 플래그 둘이 붙어 있는지를 한 번에 본다.

---

## 4. 허용목록 규칙 — 모르는 것은 안 싣는다

`logs/` 의 선별은 **아는 것만 남기는 쪽**이다. `env.go` 의 환경변수 화이트리스트와
같은 규율이다 (`agent-runtime` R1).

```text
   전문으로 남는다   첫 system/init · 마지막 type=="result" · stderr.  **셋뿐이다**
   껍데기로 남는다   type 이 문자열인 그 밖의 모든 사건
   아예 안 남는다    JSON 객체가 아닌 줄 · type 이 문자열이 아닌 줄.  세기만 한다
   경로에 예외가 없다  크래시 · 임대 만료 · 플래그 오류에서도 같다
```

**껍데기는 새로 짓는다 — 원본에서 지우지 않는다.** 실측이 그 이유를 보였다.
`assistant` 사건은 도구 입력을 `message.content[]` 의 `tool_use.input` 에
싣고 **최상위 `wire_tool_inputs` 에 한 번 더** 싣는다. `user` 사건은 도구 결과를
`message.content[]` 의 `tool_result.content` 와 **최상위 `tool_use_result`** 에
싣는다. **블록만 지우는 구현은 안 닫힌다.**

**새 사건 종류가 와도 규칙이 안 바뀐다.** `type` 만 남고 나머지는 안 실린다.
목록이 늘어도 기본 동작이 그대로인 것이 허용목록의 값이다.

### 4.1 껍데기가 담는 것 — 답 2 = C

```text
   type · subtype    사건 종류.  subtype 은 system 일 때만
   tools             tool_use 블록의 name.  배열이다
   ok                tool_result 가 있을 때만.  is_error 가 참인 것이 하나라도
                     있으면 false
   tokens            정수 넷 (in · out · cache_write · cache_read) +
                     thinking_tokens 의 thinking
```

**정수가 아니면 그 키를 건너뛴다.** 실측이 `usage` 안에 문자열 둘
(`service_tier` · `inference_geo`)과 객체 하나(`cache_creation`)를 보였다 —
`usage` 를 통째로 옮기는 구현은 이 규칙을 깬다.

**본문은 어느 사건에서도 안 남는다.** `assistant` 의 `text` 도 `thinking` 도
도구 결과도 같다.

---

## 5. 봉투를 고르는 자리가 **둘**이고 규칙이 다르다

같은 「최종 봉투」라는 말이 두 자리에서 다른 것을 뜻한다. 섞으면 한쪽이 깨진다.

```text
   ParseClaude   lastJSONObject 로 집는다 + type 검사 (⑯)
                 왜 lastJSONObject 인가 — 봉투가 여러 줄로 예쁘게 찍혀 올 수 있고
                 stdout 에 로그가 섞일 수 있다 (harness.go:126-130).  줄 단위로
                 바꾸면 그 둘이 깨진다

   logs/ 선별     줄을 돌며 type == "result" 인 **마지막** 줄을 고른다
                 왜 lastJSONObject 가 아닌가 — 크래시 때 마지막 완결 객체가
                 assistant 사건이라 그 본문이 전문으로 실린다.  4절이 닫으려던
                 길이 그대로 열린다
```

**⑯ 이 둘을 같은 검사로 묶는다** — 어느 쪽이든 `type` 이 `"result"` 가 아니면
봉투가 아니다. 다만 **집는 방법이 다르다.**

---

## 6. 순서 규칙 — 「첫 줄은 `init`」 (답 3 = B)

**불변식** — `init` 이 나오는 경로에서 `logs/` 의 첫 줄은 언제나
`system/init` 의 원문이다. 게이트가 `head -1` 로 읽는다.

### 6.1 그 불변식이 서는 근거 셋

```text
   ①  우리 settings.json 에 Stop 외의 훅이 0 이다
      실측이 본 pre-init 사건 여덟은 전부 SessionStart 훅에서 왔다.
      우리가 쓰는 설정에는 그 자리가 없다

   ②  --setting-sources "" 가 개인 · 프로젝트 설정을 끊는다
      남의 훅이 실릴 경로가 그것뿐이다

   ③  선별이 원래 순서를 지키고 표시 줄은 끝에 붙는다
      우리 코드가 init 앞에 아무것도 안 끼운다.  stdout 이 비면
      표시 줄도 안 쓴다 (business-logic-model.md 5.1)
```

### 6.2 시험 둘이 ① 과 ③ 을 잰다

```text
   ①  WriteHookSettings 가 짓는 settings 의 hooks 키에 **Stop 하나뿐**이다.
      self 가 빈 경우에는 hooks 키 자체가 없다
   ③  사건 스무 줄짜리 픽스처를 넣어 조립한 logs/ 의 첫 줄이
      init 의 원문과 **바이트로 같다**
```

### 6.3 잔여 위험을 이름으로 적는다

**② 는 우리가 못 잰다.** 하네스가 `init` 앞에 우리 설정과 무관한 사건을 내지
않는다는 보증은 하네스에 없다. 실측에서 본 pre-init 사건이 전부 훅에서 왔다는
것은 **한 번의 관측**이지 계약이 아니다.

```text
   깨지면      게이트 CA1 · CA4 · CA5 의 head -1 이 엉뚱한 줄을 읽는다
   최악의 모양   그 줄에 mcp_servers 키가 없어 「비어 있다」로 오독된다 —
              **차단을 못 했는데 초록이 난다**
   안 고른 길   답 3 = A 가 그 위험을 없앴다 (게이트가 첫 init 줄을 골라 읽는다).
              사용자가 B 를 골랐고 게이트 명령은 안 바뀐다
   줄이는 것    6.2 의 시험 둘.  유닛 안에서 잴 수 있는 만큼은 잰다
```

**이 줄을 진행자에게 넘긴다** (8절).

---

## 7. 오류 문구 규칙

`CONVENTIONS.md` 2.1 대로 **영어**다. 호출자와 로그가 읽는다.

```text
   새 Reason 을 안 만든다   치명은 HarnessResult{Reason: ReasonError, Message: <사유>} 다.
                          Record 어휘를 늘리는 것은 팩의 제외 여덟 중
                          「기존 계약 변경」에 걸리고, 판정은 어차피 같다

   Message 에 안 싣는 것    봉투의 원문 줄.  ⑯ 의 분기에서는 봉투의 type 만 싣는다.
                          Message 는 claim.go:784 의 res.Error 와
                          steps/NN-*.json 으로 봉인에 들어간다

   게이트가 찾는 문구       U1 에는 없다.  CA4 · CA5 의 부분 문자열 둘은
                          U4 · U5 의 것이다 (component-methods.md 2.2)
```

U1 이 내는 치명의 문구는 **무엇을 못 했는가로 시작한다** — 예:
`cannot create instrumentation directory` · `cannot write mcp allowlist` ·
`cannot copy credentials`. 원인 오류를 `%w` 로 잇는다.

---

## 8. 이 유닛이 남기는 한계와 넘기는 것

### 8.1 게이트웨이 인증이 보조 실패에 달려 있다

**규칙 R3 은 보조인데 그 파일이 두 가지를 든다.**

```text
   훅 블록              보조다.  모델 협조가 필요한 셋째 겹이고
                       진짜 안전망은 워크스페이스 diff 다 (hook.go 머리)
   gatewayAuthFields   보조가 아니다.  사내 게이트웨이 노드는 인증이
                       이 두 필드(apiKeyHelper · env)로 온다
```

그래서 **디스크 실패로 `settings.json` 을 못 쓰면 게이트웨이 노드에서
`Not logged in` 으로 늦게 죽는다.** 같은 회차가 자격증명 복사 실패(R6)를
치명으로 올린 근거가 바로 「늦게 아는 실패를 앞으로 당긴다」였다 — **R3 과 R6 의
등급이 같은 논거 위에서 갈린다.**

```text
   이 유닛이 한 것   등급을 안 바꾼다.  ⑨ 와 ⑪ 이 「감싸는 자리는 하나」로
                   닫은 값이고 FD 가 뒤집을 자리가 아니다
   본 다른 길       gatewayAuthFields() 가 무엇을 돌려줬는지로 등급을 가른다 —
                   있으면 치명 · 없으면 보조.  **안 골랐다** — 그 함수가 실제
                   홈을 읽어서 시험이 조건을 세울 수 없다.  못 재는 규칙은
                   규칙이 아니다
   넘긴다          진행자에게.  값을 바꾸려면 hook.go 의 겉면이 먼저 갈라져야 한다
```

### 8.2 눈 검증이 복제본을 잰다

`defer os.RemoveAll(tmp)` 를 그대로 둔다 (⑩). 대가는 사람이 손으로 띄우는
경로가 실물과 갈린다는 것이다 — 환경(`harnessEnv` 가 `os.Environ()` 을 안
얹는다) · 플래그(`--settings` · `--permission-mode` · `--add-dir` 이 빠진다) ·
게이트웨이 인증(`apiKeyHelper` 가 빠져 `Not logged in` 이 난다).

**⑮ 가 그 대가를 줄였다** — `init` 줄이 `logs/` 에 남으므로 CA1 은 복제본이
아니라 실물을 읽는다. 남는 것은 사람이 그 줄을 **보고 판정하는 일**이다.

### 8.3 넘기는 것 넷

```text
   6.3  「첫 줄은 init」의 잔여 위험.  답 3 = B 의 대가다
   8.1  R3 과 R6 의 등급이 같은 논거 위에서 갈린다
   5.3  domain-entities.md 의 thinking_tokens — 답 2 = C 의 글자 밖이다.
        빼려면 그 한 줄을 지운다
   1.4  domain-entities.md 의 미지 키 이음매.  U3 이 그 위에 자란다
```
