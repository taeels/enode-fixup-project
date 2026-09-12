# U1 `isolation` — 흐름

**치명이 전부 exec 앞에 모이는 순서**와 그 안의 네 흐름이다. 형식은
`domain-entities.md`, 규칙은 `business-rules.md` 가 진다.

---

## 1. `runHarness` — 열 걸음

`component-methods.md` 3절의 순서를 U1 의 몫으로 좁힌 것이다. **U1 이 세우는
걸음은 ① · ④ · ⑥ · ⑨ 이고, ② 와 ⑧ 은 자리만 나고 U4 · U5 가 채운다.**

```text
   ①  계장 디렉터리를 짓는다          MkdirTemp.  실패 -> 단계 실패 (④ 의 결정)
                                    오늘은 if 블록의 조건이었다.  함수 몸통으로 올린다

   ②  resolveComponents(j)          U1 에서는 언제나 빈 Components 를 낸다.
                                    요청도 팩도 없는 경우만 다룬다 (U4 가 자란다)

   ③  Argv 를 조립한다               --output-format stream-json --verbose (⑮)

   ④  Instrument(tmp, self, a, c)    2절.  errAux 만 삼킨다.  삼킬 때도 플래그는 붙인다

   ⑤  Fixed(tmp) 를 합친다           CLAUDE_CONFIG_DIR=<tmp>/home 이 여기서 환경에 든다

   ⑥  exec                          **링 tee 를 안 건다** (⑲ · 4절)

   ⑦  Decode · Version · 자백 읽기    오늘 그대로.  Decode 안의 ParseClaude 만 바뀐다 (3절)

   ⑧  HarnessResult 에 MCP · Pack    U5.  U1 은 필드도 안 만든다

   ⑨  logs/ 에 실을 것을 고른다        5절.  오늘은 stdout 전체를 낸다 (runner.go:155-156)

   ⑩  defer 가 계장 디렉터리를 지운다   오늘 그대로.  보존 스위치를 안 만든다 (⑩)
```

**① 이 오늘과 갈리는 자리다.** 오늘 `runner.go:63` 은
`if tmp, err := os.MkdirTemp(...); err == nil` 이라 **실패하면 계장을 통째로
건너뛰고 그대로 기동한다.** 그 경로로 가면 가짜 홈이 없고 개인 설정과 계정
커넥터가 다 보이는 하네스가 돈다. 몸통으로 올리고 실패를 단계 실패로 만든다.

**`self` 가 비어도 ④ 로 간다.** 오늘 `runner.go:66` 의 `if self != ""` 가
계장 전체를 감싸고 있어 `os.Executable()` 하나가 허용목록까지 떨어뜨린다
(`application-design.md` 4.2). 그 조건을 `Instrument` 안으로 옮긴다 — **훅 블록
하나만** 그것에 달린다.

---

## 2. `Instrument` — 다섯을 쓰고 플래그를 낸다

**조기 반환하지 않는다** (불변식 ③). 보조 실패를 만나도 남은 쓰기를 끝까지 하고
마지막에 감싼다.

```text
   들어온다   dir · self · a HookArgs · c Components

   ①  <dir>/home 을 만든다                     실패 -> 치명.  **가장 먼저다** (불변식 ①)
   ②  훅 설정을 <dir>/home/settings.json 로     실패 -> errAux 로 담아 두고 **계속 간다**
        self 가 비면 hooks 키를 빼고 쓴다         gatewayAuthFields() 는 그때도 얹는다
        성공했을 때만 --settings <경로> 를 얻는다
   ③  팩을 <dir>/home/ 아래 편다                U5.  c.Pack 이 nil 이면 아무것도 안 한다
   ④  <dir>/mcp.json 을 쓴다                   실패 -> 치명
        --strict-mcp-config --mcp-config=<경로> 를 얻는다
   ⑤  .credentials.json 을 복사한다             원본 없음 -> 정상.  있는데 실패 -> 치명

   나간다     플래그와 오류.  **오류가 있어도 플래그는 함께 나간다** (불변식 ②)
```

### 2.1 플래그는 산출물마다 붙는다

불변식 ② 는 「보조 실패에도 이미 얻은 플래그를 돌려준다」다. **그 「이미 얻은」을
파일 단위로 읽는다** — 못 쓴 파일을 가리키는 플래그는 안 붙인다.

```text
   --setting-sources ""                 **언제나** 붙는다.  파일을 안 가리킨다.
                                        「아무것도 읽지 마라」라서 우리 파일과 무관하다
   --settings <경로>                     ② 가 성공했을 때만
   --strict-mcp-config --mcp-config=…    ④ 를 지났으면 언제나.  ④ 는 치명이라
                                        실패하면 여기까지 안 온다
```

**없는 파일을 가리키는 `--settings` 를 붙이면 하네스가 아예 안 뜬다.** 보조
실패 하나가 치명이 되는 길이라 붙이지 않는다. 그때도 `--setting-sources ""` 는
남아 개인 설정을 끊는다 — **격리의 겹이 안 사라지는 것이 불변식 ② 의 뜻**이고
그것은 이 셋 중 첫째와 셋째가 진다.

### 2.2 `self` 가 빌 때

```text
   오늘     Instrument 를 통째로 안 부른다 (runner.go:66)
   이 유닛   부른다.  settings.json 에서 **hooks 키 자체를 뺀다** —
            빈 값을 쓰는 것이 아니다.  shellJoin 이 앞이 빈 명령을
            만드는 것을 막는다 (application-design.md 4.2)
   남는 것   gatewayAuthFields() 의 두 필드 · 허용목록 · 플래그 전부
   등급     훅이 없는 것은 보조 실패다.  단계는 돈다
```

---

## 3. `ParseClaude` — `type` 검사가 switch 앞에 선다

```text
   ①  lastJSONObject(stdout)          오늘 그대로.  여러 줄 봉투를 집으려면 필요하다
   ②  빈 문자열이면                    ReasonError · "no result envelope (exit N)"
   ③  Unmarshal 실패면                 ReasonError · "cannot parse result envelope: …"
   ④  h = {Turns, CostUSD, Session}    **셋을 먼저 채운다**
   ⑤  e.Type != "result" 이면          ReasonError · Message 는 **봉투의 type 만**
                                      type 이 비면 "no result envelope type"
                                      여기서 h 를 들고 돌아간다
   ⑥  switch                          오늘 그대로
```

**⑤ 가 switch **앞**에 서야 한다.** 뒤에 두면 둘째 case 가 `subtype` 에 `token`
이 든 무엇이든 `ReasonMaxTokens` 로 떨어뜨리고, `Completed()` 가 참이 되어
반쯤 쓴 `$OUT` 이 수확된다.

**④ 가 ⑤ 앞에 서야 한다.** `TestHarnessRecordsBudget`
(`harness_test.go:64-69`)의 픽스처에 `type` 이 없어 이 분기를 탄다. 조기
반환으로 짜면 `Turns` · `CostUSD` 가 0 이 되어 그 시험이 빨갛다. `Session` 은
그 시험이 안 재지만 같이 채운다 — 안 재는 것과 안 채워도 되는 것은 다르다.

**기존 시험 표는 그대로 초록이다.** `harness_test.go:9-47` 의 일곱 중 `Reason` 을
재는 것은 전부 `type:"result"` 를 들었거나 봉투가 아예 없다.

---

## 4. 링 tee 를 끄는 자리 (⑲)

```text
   끈다        runner.go:103-105.  cmd.Stdout 을 언제나 &stdout 하나로 둔다.
              j.Transcript 는 하네스 단계에서 안 쓰인다
   안 건드린다  claim.go:632-635 의 명령 단계 tee.  링에 기록자가 둘이고
              이 유닛이 끄는 것은 하나다
   왜          그 tee 가 5절의 선별 **앞**이라 안 끄면 걸러야 할 것이
              링으로 그대로 나간다.  링은 512 KiB 파일로 노드 디스크에 남는다
```

`Job.Transcript` 필드는 **남긴다.** 짝 팩이 선별과 함께 되살리고, 명령 단계가
지금도 쓴다.

---

## 5. `logs/` 선별 — 다섯 걸음

오늘은 `runner.go:155-156` 이 `stdout + stderr` 를 원문 그대로 낸다.

```text
   ①  줄로 나눈다        stdout 을 개행으로.  마지막 빈 조각은 버린다

   ②  두 자리를 먼저 찾는다  **첫** system/init 의 줄 번호 ·
                          **마지막** type=="result" 의 줄 번호.
                          못 찾으면 없는 것이다 (-1)

   ③  순서대로 조립한다    원래 줄 순서를 지킨다
                          ② 가 찾은 두 줄     원문 그대로
                          type 이 있는 그 밖    껍데기 한 줄로 짓는다.  센다
                          그 밖의 모든 줄      **안 싣는다.**  센다
                                             JSON 이 아니거나 type 이
                                             문자열이 아닌 줄이다

   ④  표시 줄을 붙인다     stdout 에 줄이 하나라도 있었을 때만.  5.1

   ⑤  stderr 를 붙인다    원문 그대로.  오늘과 같다
```

### 5.1 표시 줄을 언제 쓰나

```text
   쓴다      stdout 에 줄이 하나라도 있으면.  걷은 것이 0 이어도 쓴다 —
            이 파일이 걸러진 것임을 events: 0 으로 말한다
   안 쓴다   stdout 이 통째로 비면.  그때 표시 줄을 쓰면 **그것이 첫 줄이 되어**
            「init 이 안 나오면 stderr 가 첫 줄」이 깨진다
```

### 5.2 두 자리를 **먼저** 찾는 이유

마지막 `result` 를 고르려면 끝까지 봐야 하고, 조립은 순서대로 해야 한다. 한
번에 못 한다. **`lastJSONObject` 로 대신 뽑으면 안 된다** — 크래시 때 마지막
완결 객체가 `assistant` 사건이라 그 본문이 전문으로 실린다.

**`result` 가 여럿이면 마지막 하나만 전문**이고 나머지는 껍데기다. `init` 이
여럿이면 첫 하나만 전문이다. 실측은 각각 하나였지만 세는 쪽으로 짠다.

---

## 6. 오늘과 갈리는 자리 — 여덟

| 자리 | 오늘 | 이 유닛 |
|---|---|---|
| `runner.go:63` | `MkdirTemp` 실패 -> 계장 건너뛰고 기동 | 단계 실패 |
| `runner.go:66` | `self == ""` -> 계장 전체 건너뜀 | `Instrument` 를 부르고 훅 블록만 뺀다 |
| `runner.go:77-80` | `err == nil` 일 때만 플래그 | 오류에도 얻은 플래그를 붙인다. 치명은 단계 실패 |
| `runner.go:103-105` | `Transcript` 로 tee | 안 건다 |
| `runner.go:155-156` | stdout 전문 | 5절의 선별 |
| `claude.go` `Argv` | `--output-format json` | `stream-json --verbose` |
| `claude.go` `Fixed` | 인자 없음 · 값 하나 | `Fixed(dir)` · `CLAUDE_CONFIG_DIR` 이 는다 |
| `hook.go:326` | `<dir>/enode-settings.json` | `<dir>/home/settings.json` |

**확정으로 빨개지는 시험 다섯**은 `unit-of-work.md` 1절이 이미 세었다 —
`claude_test.go:14-15` 와 `hook_test.go` 의 `:215` · `:247` · `:277` · `:300`.
이 유닛이 함께 고친다.
