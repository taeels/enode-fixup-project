# U5 `pack` — Functional Design 계획

회차 `v3-run-harness-components` 의 마지막 유닛이다. 맡는 기능은
`features.md` 3.6(팩 — 전달과 출처)과 3.7(기록)이고 닫는 게이트는 CA5 · CA6 다.
선행 셋(U1 `isolation` · U2 `contract-vocab` · U4 `sources`)은 전부 `main` 에 있다.

**계획을 짓기 전에 만지는 자리를 코드로 읽고 실물 하네스로 쟀다.** 그 실측이
이 유닛의 뼈대를 바꿨다 — 1절이 그것이다. 물음 여덟 중 첫째가 거기서 나온다.

---

## 0. 이 단계가 닫는 것과 안 닫는 것

```text
   닫는다     팩 tar 의 검증 규칙과 상한의 실제 값 (PackLimits)
              Pack · PackFile 의 필드와 팩 mcp.json 의 어휘
              팩을 하네스가 실제로 읽게 하는 전달 경로
              팩과 노드 선언이 이름으로 부딪칠 때의 판정 범위
              HarnessResult.MCP · .Pack 의 값과 채우는 시점
              오케스트레이터 설정(ExecutorConfig.Pack)과 계약 템플릿의 모양

   안 닫는다   코드.  Code Generation 이 쓴다
              CA5 · CA6 의 서명.  사람이 실물로 띄운다 (scene-gates.md 2절 머리)
              runctl submit --pack.  이월이다 (features.md 4절)
              팩의 기대 다이제스트 검증.  이월이다 (decisions.md 6절 ③)
```

---

## 1. 실측 — 오늘의 플래그 아래서 팩이 하네스에 안 들어간다

**2026-09-13 · `claude 2.1.270`** 로 쟀다. 명령은 오늘 `enode` 가 실제로 내는 것
그대로다 — `Argv` 의 여섯(`-p --output-format stream-json --verbose
--permission-mode bypassPermissions --add-dir ...`)에 `Instrument` 가 붙이는
넷(`--setting-sources "" --settings <홈>/settings.json --strict-mcp-config
--mcp-config=<계장>/mcp.json`), 환경은 `harnessEnv` 의 화이트리스트에
`CLAUDE_CONFIG_DIR` 과 `CLAUDE_CODE_DISABLE_AUTO_MEMORY=1` 을 얹은 것이다.

판정 재료는 `system/init` 줄이다. 게이트 CA5 가 읽는 그 줄이다.

### 1.1 가짜 홈의 `skills/` 와 `agents/` 를 안 읽는다

`<계장>/home/skills/hello/SKILL.md` 와 `<계장>/home/agents/helper.md` 를 두고
오늘의 플래그 그대로 돌렸다.

```text
   skills          내장 열여덟만.  hello 가 없다
   agents          내장 다섯만.  helper 가 없다
   slash_commands  hello 가 없다
   mcp_servers     [{probe4, failed}]   허용목록은 오늘도 그대로 선다
```

**`--setting-sources ""` 가 그것을 끊는다.** 스킬과 서브에이전트는 user 범위의
것이고 그 플래그는 「아무 범위도 읽지 마라」다. `unit-of-work.md` 5절이 적은
「`Instrument` 가 `<dir>/home/skills/` · `agents/` 를 편다」는 **펴기만 하고 안
읽히는 자리**였다 — 파일은 생기는데 하네스가 안 본다. CA5 는 그 상태로 확정
빨강이고, 빨간 이유가 팩 코드가 아니라 플래그다.

### 1.2 길이 둘이고 둘 다 실측으로 선다

**길 하나 — `--setting-sources user`.**

```text
   skills          hello 가 맨 앞에 선다.  그 뒤는 내장 열여덟
   agents          helper 가 낀다
   slash_commands  hello 가 있다 (index 0)
   샌 것            0.  사람의 ~/.claude/skills 수십과 ~/.claude/agents 둘
                   (aidlc-verify · aidlc-xhigh)이 하나도 안 나타났다
```

**`CLAUDE_CONFIG_DIR` 이 user 범위를 통째로 옮긴다** — 그래서 `user` 를 켜도
읽히는 것은 우리가 쓴 파일뿐이다. 이것이 이 길이 서는 근거다.

**길 둘 — `--plugin-dir <계장>/pack`.** `--setting-sources ""` 를 안 건드린다.

```text
   skills          plug:hello2 가 낀다        디렉터리 이름이 앞에 붙는다
   agents          plug:helper2 가 낀다
   slash_commands  plug:hello2 가 있다
   plugins         [{name: plug, source: plug@inline}]
   필요 없던 것     .claude-plugin/plugin.json.  없이도 실렸다
```

**이름이 `<디렉터리 이름>:<스킬 이름>` 으로 바뀐다.** 디렉터리를 `pack` 으로
지으면 `pack:hello` 다. 모델이 설명으로 스킬을 고르는 경로는 그대로이고,
바뀌는 것은 사람과 게이트가 읽는 글자다.

### 1.3 워크스페이스 `.claude/skills/` 도 안 읽힌다 — 미정 하나가 닫힌다

세 번의 실측 내내 워크스페이스(`cwd`)에 `.claude/skills/ws-skill/SKILL.md` 를
두고 돌렸다. **한 번도 안 나타났다** — `--setting-sources ""` 에서도
`--setting-sources user` 에서도 그렇다. project 범위가 안 켜지기 때문이다.

`features.md` 5절과 `decisions.md` 2절의 열린 미정(「하네스가 스스로 읽는가.
안 읽으면 팩으로 나른다」)이 **안 읽는다**로 닫힌다. CA5 의 마지막 줄이 재려던
값을 이 실측이 먼저 냈고, 게이트는 그것을 확인하는 자리가 된다.

### 1.4 없는 것을 말없이 무시한다 — 두 길 다

`--plugin-dir` 에 없는 경로를 주고 돌렸다. **종료코드 0 · `plugins` 가 빈 배열 ·
stderr 에 한 줄도 없다.** 스킬이 하나도 없는 채로 단계가 성공으로 끝난다.

`ADR-035` §3 의 「없음이 실패보다 나쁘다」가 여기 그대로 있다. 하네스는 팩이
빠진 것을 안 알려주므로 **빠짐을 우리가 잡아야 한다** — 펴는 쪽이 자기가 쓴
파일 수를 세고, 팩이 있는데 실린 것이 0 이면 그 단계를 실패로 본다.

### 1.5 훅은 두 번 안 뛴다

길 하나(`--setting-sources user`)가 `<홈>/settings.json` 을 user 설정으로도
읽고 `--settings` 로도 읽는다. 같은 파일이 두 번 실리면 Stop 훅이 두 번 뛰고
수확이 두 번 돈다. **쟀다 — 마커 파일에 한 줄이다.** 두 번 안 뛴다.

이 값이 없으면 길 하나가 훅 설계(⑧)와 부딪히는 것으로 보인다. 안 부딪친다.

### 1.6 부르는 자리 — `Job` 이 들고 오는 것과 안 오는 것

```text
   있다     Params.Pack     U2 가 넣었다.  blob 이름 하나 (agent.go:58-68)
            IO.In           $IN 디렉터리.  claim.go 가 blob 을 그 이름 그대로
                            파일로 깐다 (claim.go:520) 그리고 0444 · 0555 로 잠근다
            NodeMCP         충돌 판정의 상대편 (U4)

   없다     팩 파일의 내용.  누구도 안 연다
            claim.go 는 이 유닛의 행렬 밖이다 — 워크스페이스 파일과 달리
            가장자리를 새로 못 만든다.  그래서 여는 자리가 물음 3 이다
```

**`in.from` 의 정적 검사는 이미 선다** (`contract.go:1109-1126`). U2 가 그 위에
`agent.pack` 이 적은 이름이 같은 단계의 `in.from` 에 있어야 한다는 규칙을 얹었다
(`contract.go:961-999`). 그래서 이름 오타로 팩이 조용히 빠지는 경로는 닫혀 있고,
남은 것은 **하네스가 안 읽는 자리**(1.1)와 **펴다 실패하는 자리**뿐이다.

### 1.7 게이트의 문장이 오늘 출력과 갈리는 자리

```text
   init 줄의 필드가 둘이다   skills 와 slash_commands.  스킬 이름은 둘 다에 있고
                            slash_commands 에는 내장 명령이 섞인다.  CA5 는
                            slash_commands 만 적었다 — 틀리진 않았으나 좁다
   버전                     U4 는 2.1.266 으로 쟀고 오늘은 2.1.270 이다.
                            위 값은 전부 2.1.270 의 것이다
```

---

## 2. 물음 여덟

각 물음의 `[Answer]:` 뒤에 글자 하나를 적어 주면 된다. 권장은 A 로 두었고
권장을 고르지 않을 이유가 물음마다 함께 적혀 있다.

### Question 1 — 팩의 스킬과 서브에이전트를 무엇으로 들이나

1.1 이 오늘의 플래그로는 안 들어간다고 말한다. 무엇으로 여는가.

- **A. `--plugin-dir <계장>/pack` 을 팩이 있을 때만 더한다** (권장)
  - `--setting-sources ""` 를 안 건드린다. 격리의 그 겹이 그대로 남는다
  - 모양이 허용목록과 같다 — 「아무것도 읽지 마라 + 우리가 지은 이 자리만」
    이 `--strict-mcp-config --mcp-config=<경로>` 와 같은 문장이다
  - 대가 하나 — 이름이 `pack:hello` 로 바뀐다. CA5 의 문장과 계약 저자가
    슬래시로 부르는 이름이 함께 바뀐다
  - 위험 하나 — 플러그인 규약(`.claude-plugin/plugin.json`)이 없이 실리는
    것은 2.1.270 의 실측값이다. 규약이 굳으면 매니페스트 한 장이 더 필요해진다
- **B. `--setting-sources user` 로 바꾸고 팩을 가짜 홈에 편다**
  - `unit-of-work.md` 5절이 적은 원래 설계다. 이름이 `hello` 그대로다
  - 1.2 가 「샌 것 0」을 쟀고 1.5 가 「훅은 한 번」을 쟀다. 오늘 안 부딪친다
  - 대가 — 격리의 한 겹이 값에서 조건으로 바뀐다. `""` 는 무조건 닫히고
    `user` 는 `CLAUDE_CONFIG_DIR` 이 서 있을 때만 닫힌다. 그 변수가 안 먹는
    하네스 판이 오면 사람의 `~/.claude` 가 통째로 열린다 — 열리는 쪽으로
    틀리는 값이다
- **C. 스킬·서브에이전트 전달을 이월하고 이 유닛은 팩의 `mcp.json` 만 싣는다**
  - CA5 의 절반(`mcp_servers` 에 팩의 서버)만 닫고 나머지는 다음 회차로
  - 대가 — 3.6 의 값이 거의 사라진다. 팩의 뜻이 「스킬을 실어 보낸다」다

`[Answer]:` **A** — 사용자 결정 (2026-09-14). `--plugin-dir <계장>/pack` 으로 간다.
`--setting-sources ""` 는 안 건드린다.

### Question 2 — 팩의 `mcp.json` 을 어떤 어휘로 받나

`component-methods.md` 2.1 은 `Pack.MCP map[string]MCPServer` 로 적었다. 그런데
U4 가 같은 물음에서 반대로 갔다 — `Components.Servers` 를 `MCPServer` 가 아니라
최종 허용목록 항목(`map[string]any`)으로 올렸다. 팩의 `mcp.json` 은 워크스페이스
`.mcp.json` 과 같이 **하네스가 정의한 형식**이다.

- **A. 원문 그대로 `map[string]map[string]any` 로 받는다** (권장)
  - U4 와 같은 판단이다. `type` · `headers` 처럼 우리 어휘에 없는 키가 안 사라지고
    `env` 의 뜻이 안 뒤집힌다 (노드 선언의 `env` 는 이름에서 이름으로 가고
    이 파일의 `env` 는 값이다)
  - 번역하는 코드가 없으므로 번역이 못 틀린다
  - `component-methods.md` 2.1 의 한 줄을 고친다 — 이 유닛의 파장으로 적는다
- **B. `MCPServer` 로 받는다**
  - 설계 문서 그대로다. 노드 선언과 같은 검증을 공짜로 받는다
  - 대가 — U4 가 실측으로 잡은 손상 셋(키 소실 · `${${X}}` 이중 감싸기 ·
    종류 오류의 전파)이 팩 경로로 되돌아온다

`[Answer]:` **A** — 사용자 결정 (2026-09-14). 원문 그대로 `map[string]map[string]any` 다.

### Question 3 — `$IN/<팩>` 을 누가 여나

`resolveComponents` 는 파일을 하나도 안 만지는 함수다(U1 의 불변식). 팩은
파일이다. `claim.go` 는 이 유닛의 파일 행렬 밖이다.

- **A. `runHarness` 가 `resolveComponents` 앞에서 연다** (권장)
  - `Job.Params.Pack` 과 `Job.IO.In` 이 이미 거기 있다. 행렬을 안 넘는다
  - `readPack` 이 `io.Reader` 하나를 받으므로 시험은 메모리에서 악성 tar 를
    지어 넣는다 — 디스크가 필요 없다
  - `resolveComponents` 는 이미 검증을 통과한 `*Pack` 을 받아 필터와 충돌만 본다.
    「정하는 함수는 파일을 안 만진다」가 그대로 산다
  - 대가 — 가장자리가 둘로 갈린다. 워크스페이스는 `claim.go` 가 열고 팩은
    `runner.go` 가 연다. 그 비대칭을 주석이 이름으로 진다
- **B. `claim.go` 가 열어 `Job.Pack` · `Job.PackErr` 로 담는다**
  - U4 의 워크스페이스와 대칭이다. 가장자리가 한 자리에 모인다
  - 대가 — 파일 행렬의 `claim.go` 행에 U5 가 붙는다. 접점이 하나 는다

`[Answer]:` **A** — 사용자 결정 (2026-09-14). `runHarness` 가 `resolveComponents` 앞에서 연다.

### Question 4 — 상한의 실제 값 (`PackLimits`)

`decisions.md` 6절 ② 가 이 값을 이 단계로 넘겼다. 전송은 이미 막혀 있다 —
Mediator 의 `MaxBlobBytes` 기본 10 MiB 가 tar 자체를 막는다. 안 막힌 것은
**푼 뒤의 크기**다.

- **A. `MaxBytes` 64 MiB · `MaxFiles` 512 · 파일 하나의 상한은 안 둔다** (권장)
  - 전송 상한(10 MiB)의 여섯 배를 푼 바이트에 준다. gzip 을 받기로 하면
    (물음 5) 압축률 6.4 까지가 그 안에 들어온다 — 글자 팩의 실제 압축률이
    거기 못 미치므로 폭탄만 걸리고 성한 팩은 안 걸린다
  - 스킬 팩의 실제 크기는 킬로바이트 단위다. 512 는 사고를 잡는 값이지
    사람을 막는 값이 아니다
  - 합계로 세므로 파일 하나짜리 상한이 따로 필요 없다
- **B. `MaxBytes` 10 MiB · `MaxFiles` 256** — 전송 상한과 같은 값으로 맞춘다
  - 값이 하나라 설명이 짧다. 대가 — gzip 을 받으면 압축률 1 을 못 넘긴다
- **C. 노드 설정(`enode.yaml`)으로 뺀다**
  - 대가 — 이 회차가 새 설정 키를 늘린다. `constraints.md` 의 「새 표면을
    최소로」와 어긋나고, 소유자가 안 적으면 어차피 기본값이 판단한다

`[Answer]:` **A** — 사용자 결정 (2026-09-14). `MaxBytes` 64 MiB 를 값으로
박았고 `MaxFiles` 512 는 그대로다.

### Question 5 — gzip 을 받나

계약 저자가 쓸 수 있는 것은 argv 하나다. 셸이 없으므로 받은 것을 다시 풀어
tar 로 만들 자리가 없다. 그런데 세상의 tar 주소 상당수가 `.tar.gz` 다.

- **A. 매직 두 바이트(`1f 8b`)를 보고 투명하게 푼다** (권장)
  - `curl -o $OUT/pack <.tar.gz 주소>` 가 그대로 선다. `git archive --remote`
    의 기본 tar 도 그대로 선다
  - 폭탄은 상한이 진다 — 푼 바이트를 세면서 `MaxBytes` 에서 끊는다.
    `unit-of-work.md` 5절이 「푼 뒤의 크기를 `MaxBytes` 가 진다」로 이미 그렇게 적었다
- **B. 순수 tar 만 받고 gzip 은 문구로 거절한다**
  - 압축 해제 경로가 아예 없다. 읽는 코드가 짧아진다
  - 대가 — 계약 저자가 `.tar.gz` 를 만나면 팩을 못 쓴다. 실패는 시끄럽다
    (거절 문구가 나간다)

`[Answer]:` **A** — 사용자 결정 (2026-09-14). 매직 두 바이트를 보고 투명하게 푼다.

### Question 6 — 노드 선언과 이름이 부딪칠 때 어디까지 보나

`decisions.md` 6절 ⑦ 이 「팩이 노드 선언 이름을 덮으면 거절」로 정했다. 범위가
안 정해졌다.

- **A. `agent.mcp` 가 요청한 이름만 본다** (권장)
  - 소유권이 뒤집히는 경로는 **그 이름이 실제로 허용목록에 실릴 때만** 열린다.
    요청 안 한 이름은 팩에 있어도 어디에도 안 실리므로 뒤집을 것이 없다
  - 팩 하나를 여러 단계가 나눠 쓸 때 남의 이름 때문에 안 죽는다
  - 대가 — CA5 의 그 줄을 「`agent.mcp` 에 그 이름을 적고」로 명시해야 한다.
    오늘 게이트 문장이 요청 여부를 안 적었다
- **B. 팩이 실은 모든 이름을 본다**
  - 요청 여부와 무관하게 노드 선언과 겹치면 거절이다. 더 시끄럽다
  - 대가 — 실행 위험이 0 인 경우까지 단계를 죽인다. 노드가 흔한 이름
    (`github` 같은)을 선언해 두면 그 이름을 담은 팩이 그 노드에서 전부 막힌다

`[Answer]:` **A** — 사용자 결정 (2026-09-14). `agent.mcp` 가 요청한 이름만 본다.

### Question 7 — 기록(`HarnessResult.MCP` · `.Pack`)을 언제 채우나

3.7 의 값이다. 「실린 것」과 「실으려던 것」이 갈린다.

- **A. 실제로 exec 한 경로에서만 채운다** (권장)
  - `component-methods.md` 1절이 「요청한 것이 아니라 실린 것이다」로 적었다
  - exec 앞에서 죽은 단계는 아무것도 안 물렸다. 그 사실이 빈 값이다
  - 실패 이유는 `Message` 가 진다 — 문구에 서버 이름과 거절 사유가 있다
  - 대가 — 팩을 읽는 데까지 성공하고 그 뒤에 죽은 단계의 다이제스트가 안 남는다
- **B. `resolveComponents` 가 선 순간부터 채운다**
  - 거절로 끝난 단계의 봉인에도 팩 다이제스트가 남는다. 사후 조사가 쉽다
  - 대가 — 「실린 것」이라는 필드의 뜻이 흐려진다. 안 실린 팩의 sha256 이
    `harness.pack` 에 있으면 봉인을 읽는 사람이 실렸다고 읽는다

`[Answer]:` **A** — 사용자 결정 (2026-09-14). 실제로 exec 한 경로에서만 채운다.

### Question 8 — 오케스트레이터가 `agent.pack` 을 어느 단계에 붙이나

`cmd/iapadapter` 의 템플릿은 단계 둘(`plan` · `gate`)을 짓고 나머지는 `plan` 이
펼친다. 팩을 쓰는 것은 펼쳐진 단계다.

- **A. 팩 단계를 맨 앞에 놓고, `agent.pack` 은 계획이 짓는 단계가 적게 한다** (권장)
  - 설정이 `executor.pack: {fetch: [argv], name: pack}` 이면 템플릿이 명령
    단계 하나를 맨 앞에 더하고 `plan` 이 그것을 `needs` 로 딛는다
  - `planPrompt` 가 blob 이름과 두 키(`agent.pack` · `in.from`)를 이름으로
    알려준다. 안 적으면 팩 없이 돈다 — 오늘 그대로다
  - `plan` 단계 자신은 팩을 안 문다. 오케스트레이션 노드는 스킬이 필요 없다
  - 대가 — 계획이 그 두 줄을 잊으면 팩이 조용히 안 실린다. 그 침묵은
    `decisions.md` 6절 ㉑ 이 이미 이름으로 진 자리다
- **B. `plan` 단계에도 붙인다**
  - 계획을 짓는 에이전트도 팩의 스킬을 본다. 잊을 자리가 하나 준다
  - 대가 — 오케스트레이션 노드가 실행용 스킬을 물고 돈다. 역할이 흐려지고
    `plan` 단계의 `in.from` 이 늘어 계약이 커진다
- **C. 어느 단계에 붙일지를 설정에 적는 키를 하나 더 둔다**
  - 대가 — 설정 표면이 는다. 오늘 답이 하나인 물음에 스위치를 두는 것이다

`[Answer]:` **A** — 사용자 결정 (2026-09-14). 계획이 짓는 단계가 적는다.

---

## 3. 답을 받은 뒤 낼 산출물

`aidlc-docs/taeels/construction/pack/functional-design/` 에 셋이다.

```text
   domain-entities.md      Pack · PackFile · PackLimits 의 필드와 뜻.
                           HarnessResult 의 새 필드 둘.
                           ExecutorConfig.Pack · PackConfig
                           팩 tar 규약 — 무엇이 규약 안이고 무엇이 무시인가

   business-rules.md       규칙에 번호를 붙인다.  경로 검증 다섯(절대경로 ·
                           .. · 심볼릭 링크 · 크기 · 개수) · settings.json 건너뛰기 ·
                           agent.mcp 필터 · 노드 선언 충돌 · 실패 등급 ·
                           빈 팩과 스킬 0 개의 판정 · 오류 문구의 정확한 글자.
                           규칙마다 어느 시험이 그것을 재는지 함께 적는다

   business-logic-model.md 걸음의 순서.  readPack 의 걸음과 resolveComponents 의
                           걸음 어디에 팩이 끼는지 · Instrument 의 어느 자리에서
                           펴는지 · runner.go 가 기록을 채우는 자리 ·
                           오케스트레이터 템플릿의 단계 배치
```

**오류 문구를 규칙으로 적는 이유** — 게이트가 단계 error 안에서 부분 문자열로
찾는다. `pack redefines node-declared mcp server <이름>` 은 CA5 가 그대로 찾는
글자다 (`decisions.md` 6절 ⑦).

---

## 4. 이 단계가 안 만드는 것

```text
   새 라우트 · 새 전송      0.  팩은 이미 있는 blob 경로로만 흐른다
   Decode 의 변경           짝 팩의 것이다 (constraints.md 접점 절)
   계약 어휘의 추가         U2 가 닫았다.  agent.pack 하나이고 안 는다 (6절 ⑫)
   팩 캐시 · system.md      이월 (features.md 4절)
   기대 다이제스트 검증      이월 (decisions.md 6절 ③)
```

---

## 5. 파장 — 이 유닛 밖으로 가는 것

답이 정해지면 함께 고칠 자리다. 유닛 안에서 안 끝난다.

```text
   decisions.md 6절        실측 행 하나.  1.1 ~ 1.5 를 한 행에 싣는다 —
                           오늘 플래그로는 팩이 안 읽힌다는 사실과 길 둘의 실측값

   decisions.md 2절        워크스페이스 .claude/skills 행.  미정이 아니라
                           「안 읽는다.  팩으로 나른다」로 값이 된다 (1.3)

   features.md 5절         미정 둘 중 하나가 빠진다.  같은 이유다

   features.md 3.6         답 1 이 A 면 「가짜 홈에 편다」가 「계장 아래 팩
                           디렉터리에 펴고 --plugin-dir 로 가리킨다」로 바뀐다

   unit-of-work.md 5절     같은 자리.  claude.go 줄의 서술이 답 1 을 탄다

   scene-gates.md 3절      CA5 의 명령 셋.  스킬 이름(답 1 이 A 면 pack:hello) ·
                           충돌 줄에 agent.mcp 를 적는 것(답 6 이 A 면) ·
                           마지막 줄은 판정이 아니라 확인이 된다 (1.3)

   component-methods.md    2.1 의 Pack.MCP 타입 한 줄 (답 2)

   파일 행렬               U5 열에 시험 파일 하나(pack_test.go).  claim.go 행은
                           답 3 이 B 일 때만 는다
```

**정본(`enode-design`)은 이 유닛이 안 고친다.** `ADR-034` §1.1 의 팩 규약과
`--plugin-dir` 이 갈리는 자리가 답 1 에 달렸고, 그것은 5절 되돌리기 목록에
올릴 값이지 이 유닛이 정본을 고칠 자리가 아니다.

---

## 6. 브랜치를 `main` 에서 땄다

U4 는 선행 셋이 안 병합돼 있어 `unit/advert` 에서 땄다. **지금은 U1 ~ U4 가
전부 `main` 에 있다** (PR #33 ~ #36). 그래서 `unit/pack` 은 `main` 에서 딴다 —
선행 셋과 회차의 Inception 산출물이 그 나무에 다 있고, 이 유닛이 만지는
`aidlc-state.md` 의 소유자가 하나라 병합에서 안 부딪친다
(`CONVENTIONS.md` 3.1 · 3.2).
