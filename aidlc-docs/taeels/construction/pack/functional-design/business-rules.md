# U5 `pack` — 검증과 필터와 거절

규칙마다 **어긴 것이 실제로 거절되는가**를 잴 수 있는 형태로 적는다 — 문장 하나에
거절 사유 하나다. 형식은 `domain-entities.md`, 지나가는 길은
`business-logic-model.md`.

**에러 문자열과 로그는 영어다** (`CONVENTIONS.md` 2.1). 주석과 이 문서는 한국어다.

**팩의 실패는 전부 치명이고 전부 exec 앞이다.** `errAux` 로 감싸는 자리가 이
유닛에 0 이다 — 훅 설정 하나뿐이라는 U1 의 값(⑪)이 그대로 산다.

---

## 1. 팩을 여는 조건

```text
   R1   agent.pack 이 비면 팩 경로를 아예 안 탄다.  파일을 안 연다
   R2   agent.pack 이 이름을 적으면 $IN/<이름> 을 연다.  없으면 거절한다
        pack blob %s was not produced by this run
```

**R2 가 `ADR-058` 의 예외다.** 「없는 입력은 값이다」는 `in.from` 의 규칙이고, 그
근거는 dispatch 로 안 간 가지의 산출물을 가리킬 수 있다는 것이었다. 팩은 다르다 —
`agent.pack` 이 **이 단계가 그것으로 돌겠다고 적은 이름**이고, 없는 채로 돌면
스킬 없이 도는 단계가 exit 0 으로 성공이 봉인된다. `decisions.md` 6절 ⑨ 의
「빠뜨림이 닫히는 쪽으로 틀린다」와 ㉑ 이 이름으로 진 그 침묵이 여기서 닫힌다.

오타는 이미 앞에서 걸린다 — `contract.Validate` 가 `in.from` 이 어느 단계도 안
내는 이름을 가리키면 제출에서 거절하고(`contract.go:1109-1126`), U2 가 그 위에
`agent.pack` 과 `in.from` 이 같은 이름을 가리켜야 한다는 규칙을 얹었다
(`contract.go:961-999`). **R2 가 잡는 것은 앞 단계가 실제로 그 blob 을 못 낸 경우**다.

---

## 2. tar 를 읽는 규칙 (SEC-A) — 파일을 쓰기 전에 전부 끝난다

```text
   R3   첫 두 바이트가 1f 8b 이면 gzip 을 한 겹 벗긴다.  그 뒤가 tar 로 안
        읽히면 거절한다
        pack %s is not a tar archive
   R4   정규 파일과 디렉터리 항목만 받는다.  그 밖의 종류는 거절한다 —
        심볼릭 링크 · 하드 링크 · 장치 · FIFO
        pack entry %s is a %s; only regular files are read
   R5   절대경로 항목을 거절한다
        pack entry %s has an absolute path
   R6   .. 를 조각으로 담은 항목을 거절한다
        pack entry %s escapes the pack root
   R7   백슬래시 · 제어문자 · 드라이브 문자를 담은 이름을 거절한다
        pack entry %s has an unsafe name
   R8   받은 바이트나 푼 바이트가 MaxBytes 를 넘으면 거절한다
        pack exceeds the size limit of %d bytes
   R9   항목 수가 MaxFiles 를 넘으면 거절한다.  규약 밖 항목도 함께 센다
        pack exceeds the file limit of %d entries
   R10  readPack 은 파일을 하나도 안 쓴다.  거부가 파일을 남기기 전에 일어난다
```

**R4 의 디렉터리 항목은 무시다.** 우리가 필요한 디렉터리를 직접 만든다 — tar 의
디렉터리 모드를 받을 이유가 없다 (`domain-entities.md` 2.1).

**R7 이 백슬래시를 거절하는 이유는 크로스 빌드다.** tar 의 이름은 `/` 로 나뉘는데
윈도우에서는 `\` 도 구분자다. 같은 팩이 리눅스와 윈도우에서 다른 경로로 펴지면
격리의 경계가 플랫폼마다 달라진다. CA0 가 크로스 빌드 셋을 이미 돈다.

**R8 이 두 곳에 걸리는 이유** — 받은 바이트만 재면 gzip 폭탄이 통과하고, 푼
바이트만 재면 tar 꼬리에 붙은 거대한 쓰레기를 끝까지 읽는다. 먼저 닿는 쪽에서
끊는다.

---

## 3. 규약 안과 밖

```text
   R11  skills/ 또는 agents/ 로 시작하는 항목만 Files 에 담는다
   R12  루트의 mcp.json 은 팩의 서버 정의다.  JSON 이 깨졌으면 거절한다
        pack mcp.json is not valid json: %v
        mcpServers 키가 없으면 서버 0 으로 본다 (readWorkspaceMCP 와 같은 판단)
   R13  규약 밖 항목은 무시하고 이름을 Notes 로 남긴다
        pack carries %s, which is outside the pack convention; ignoring it
   R14  settings.json 은 Notes 의 문구가 다르다
        pack carries settings.json; this run does not read it
   R15  같은 이름의 항목이 두 번 나오면 뒤가 이긴다.  Notes 로 남긴다
        pack carries %s more than once; the last one wins
   R16  규약 안 파일도 팩의 서버도 0 이면 거절한다
        pack %s carries no skills, agents, or mcp servers
```

**R14 는 R11 이 이미 막는 자리다.** 접두가 아니므로 `settings.json` 은 R13 으로
조용히 무시된다. 그래도 문구를 따로 두는 이유는 `ADR-034` §7 과 `decisions.md`
6절 ② 가 그 충돌을 **보이게 하라**고 적었기 때문이다 — 팩 저자가 훅 설정을 덮으려
했다는 사실이 로그에 이름으로 남는다.

**R16 이 1.4 의 침묵을 잡는다.** 하네스는 없는 플러그인 디렉터리를 종료코드 0 에
stderr 한 줄 없이 무시한다 (계획 1.4 의 실측). 팩을 적었는데 실을 것이 0 인 것은
계약 저자의 오해이거나 tar 가 엉뚱한 것이라는 뜻이고, 둘 다 조용히 돌면 안 된다.

---

## 4. 팩의 서버도 `agent.mcp` 필터를 탄다 (⑥)

```text
   R17  agent.mcp 가 이름을 적은 서버만 허용목록에 실린다.  출처가 팩이어도 같다
   R18  팩이 실었으나 요청 안 한 이름은 Notes 로 남긴다
        pack declares mcp server %s, which this step did not request
```

**팩은 정의의 출처이지 허가의 출처가 아니다** (`application-design.md` 4.4). 이
규칙이 없으면 계약 작성자가 이름을 안 적고도 임의의 stdio 서버를 물릴 수 있고,
`--strict-mcp-config` 도 가짜 홈도 그것을 안 막는다 — 허용목록 자체가 싣기 때문이다.

**스킬과 서브에이전트는 필터가 없다.** 계약이 이름으로 고르는 문법이 서버에만
있기 때문이다 (`agent.mcp`). 팩을 적었다는 것이 곧 그 스킬을 쓰겠다는 뜻이다.

---

## 5. 노드 선언과의 충돌 (⑦ · 답 6=A)

```text
   R19  요청된 이름이 팩과 노드 선언 둘 다에 있으면 거절한다
        pack redefines node-declared mcp server %s
   R20  요청 안 한 이름의 겹침은 안 본다.  R18 이 그 이름을 이미 Notes 로 낸다
```

**R19 가 막는 것은 소유권의 뒤집힘이다.** 계약이 `requires: mcp.gerrit` 으로
소유자의 선언을 보고 노드를 고른 뒤 자기 팩의 `gerrit` 정의로 그 이름을 덮으면,
**매칭은 소유자의 값으로 하고 실행은 계약의 값으로 하는** 것이 된다.

**요청된 이름만 보는 이유** (답 6=A) — 그 뒤집힘은 이름이 실제로 허용목록에 실릴
때만 열린다. 요청 안 한 이름은 팩에 있어도 어디에도 안 실리므로 뒤집을 것이 없다.
전부 보면 실행 위험이 0 인 경우까지 단계를 죽인다 — 노드가 흔한 이름을 선언해
두면 그 이름을 담은 팩이 그 노드에서 전부 막힌다.

**팩이 새 이름을 싣는 것은 그대로 허용한다.**

---

## 6. 우선순위와 종류

```text
   R21  요청된 이름이 팩과 워크스페이스에 둘 다 있으면 팩이 이긴다.  Note 를 남긴다
        mcp server %s is declared by both the pack and the workspace; the pack wins
   R22  종류(type)는 출처를 합친 뒤 공용 자리에서 채운다.  팩 항목도 같은 규칙이다 —
        command 면 stdio · url 이면 http · 이미 있으면 안 덮는다 (U4 의 걸음 4)
```

**팩이 워크스페이스를 이기는 근거는 `decisions.md` 2절이다** — 계약이 실어 보낸
것이 가장 재현 가능하다. 노드 선언과의 관계만 R19 가 거절로 바꾼다.

---

## 7. 펴기 — `Instrument` 의 ③ 자리

```text
   R23  c.Pack 이 nil 이면 아무것도 안 한다.  --plugin-dir 도 안 붙는다
   R24  <계장>/pack/ 아래에 Files 를 이름 그대로 편다.
        디렉터리 0700 · 파일 0600.  tar 의 모드를 안 쓴다
   R25  실패는 치명이다.  errAux 로 안 감싼다
        cannot extract the pack: %w
   R26  편 뒤 --plugin-dir <계장>/pack 을 플래그에 붙인다
   R27  자리는 훅 설정 뒤 · 허용목록 앞이다 (Instrument 의 ③).
        보조 실패(훅)로 조기 반환하지 않는 U1 의 불변식이 그대로 산다 —
        훅 쓰기가 실패해도 팩과 허용목록은 끝까지 쓴다
```

**R26 의 플래그가 R23 과 짝이다.** 팩이 없는 단계의 argv 가 오늘과 한 글자도 안
다르다 — CA1 이 이미 사람의 서명을 받은 게이트이고, 그 값이 이 유닛으로 안 흔들린다.

---

## 8. 기록 (3.7 · 답 7=A)

```text
   R28  HarnessResult.MCP 는 허용목록에 실제로 실린 이름이다.  이름 순이다
   R29  HarnessResult.Pack 은 Pack.SHA256 이다.  팩이 없으면 빈 값이다
   R30  둘 다 exec 을 실제로 한 경로에서만 찬다.  exec 앞에서 죽은 단계는 빈 값이다
```

**「요청한 것」이 아니라 「실린 것」이다.** 둘이 갈리는 경우가 실제로 있다 — 워크스페이스
출처로 실린 이름은 계약의 `agent.mcp` 와 같지만, 팩이 실었으나 요청 안 한 이름은
`Notes` 에만 있고 이 필드에는 없다. 봉인을 읽는 사람이 **그 단계의 모델이 실제로
본 서버 목록**을 이 필드로 읽는다.

---

## 9. 실패 등급 — 팩 관련은 전부 치명이고 전부 exec 앞이다

```text
   자리                        등급    걸리는 규칙
   ────────────────────────    ────    ────────────────────
   $IN 의 팩 파일 없음           치명    R2
   tar 가 아님 · 검증 위반        치명    R3 ~ R9
   mcp.json 이 깨짐              치명    R12
   실을 것이 0                   치명    R16
   요청된 이름이 없음             치명    U4 의 R (오늘 그대로)
   팩이 노드 선언을 덮음          치명    R19
   펴다 실패                     치명    R25
```

**보조 등급이 하나도 안 는다.** `decisions.md` 6절 ⑪ 이 「보조 등급은 하나다」로
박은 값이 이 유닛 뒤에도 참이다 — `errAux` 로 감싸는 자리는 훅 설정 쓰기 하나뿐이다.

---

## 10. 오케스트레이터 (답 8=A)

```text
   R31  executor.pack 이 비면 오늘 그대로다.  단계도 키도 안 는다
   R32  있으면 계약의 맨 앞에 명령 단계 하나를 더한다
        id "pack" · uses <executor.as> · run <fetch> · out [<name>]
   R33  plan 단계에 needs 를 안 적는다.  NeedsOf 의 기본값이 직전 단계이므로
        (contract.go:764-792) 계획이 팩 단계를 기다린다.  순서가 그것으로 선다
   R34  plan 단계 자신은 agent.pack 을 안 문다.  planPrompt 가 blob 이름과
        두 키(agent.pack · in.from)를 계획에게 알려주고, 계획이 짓는 단계가 적는다.
        안 적으면 그 단계는 팩 없이 돈다 — 오늘 그대로다
   R35  success_when 은 안 바뀐다.  판정은 report 가 summary 를 냈는가다
```

**R33 이 순서를 지는 자리다.** `in.from` 은 이름을 가리킬 뿐 실행 순서를 안
만든다 — 순서는 `needs` 가 만들고, 안 적으면 직전 단계가 기본값이다. 팩 단계를
맨 앞에 놓는 것만으로 그 뒤 전부가 그것을 딛는다.

**R34 의 대가를 이름으로 적는다** — 계획이 두 줄을 잊으면 팩 없이 도는 단계가
생긴다. 그때는 R1 이 적용되어 아무 일도 안 일어난다 (팩 경로를 안 탄다). 잊은
것을 우리가 못 잡는다.

---

## 11. 이 규칙들이 못 잡는 것

```text
   팩 내용 자체의 악의        스킬 본문이 시키는 것을 안 본다.  경계는 단계가
                            아니라 노드에 있다 (ADR-042).  enode 를 띄운 것이
                            곧 임의의 argv 를 받는다는 선언이다
   주소가 바뀐 tar            기대 다이제스트가 없다 (decisions.md 6절 ③ 이월).
                            무엇을 받았는지는 R29 가 남기지만 무엇을 받아야
                            했는지는 아무도 안 적는다
   모델이 스킬을 안 고르는 것   우리 판정은 init 줄에 이름이 있는 데까지다.
                            그 뒤는 모델의 선택이고 잴 자리가 없다
   플러그인 규약의 변화        plugin.json 없이 실리는 것은 2.1.270 의 실측값이다.
                            규약이 굳으면 매니페스트 한 장이 더 필요해진다 —
                            그때 R24 에 파일 하나가 는다
   팩 안의 이름 충돌          R15 가 뒤를 이기게 하고 Notes 로 남길 뿐이다.
                            어느 것이 맞는지는 우리가 모른다
```
