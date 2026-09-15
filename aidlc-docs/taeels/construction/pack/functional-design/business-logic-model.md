# U5 `pack` — 값이 지나가는 길

형식은 `domain-entities.md`, 규칙은 `business-rules.md`. 여기는 **순서**다 —
어느 걸음이 어느 걸음 앞이어야 하는지와, 그 순서가 무엇을 지키는지.

---

## 1. 한 줄기

```text
   계약        첫 단계  run: ["curl","-o","$OUT/pack", <주소>]  out: ["pack"]
                 │
                 ▼
   Mediator    blob "pack"                     MaxBlobBytes 10 MiB 가 여기서 막는다
                 │
                 ▼
   claim.go    $IN/pack 으로 깐다               0444 · 0555 로 잠근다 (:520 · sealInput)
                 │                              둘째 단계가 in.from: ["pack"] 을 적었다
                 ▼
   runner.go   openPack                        파일을 여는 유일한 자리 (답 3=A)
                 │  readPack — 메모리에서 검증한다.  아무것도 안 쓴다
                 ▼
   mcp.go      resolveComponents               팩 · 노드 · 워크스페이스를 합친다
                 │  팩의 서버는 agent.mcp 필터를 탄다.  스킬은 안 탄다
                 ▼
   claude.go   Instrument                      <계장>/pack/ 에 편다
                 │  --plugin-dir <계장>/pack
                 ▼
   하네스       system/init                     skills 에 pack:<이름> ·
                 │                              mcp_servers 에 요청된 이름
                 ▼
   runner.go   selectLogs + 기록                logs/ 첫 줄이 그 init 이고
                                                HarnessResult 에 mcp · pack 이 찬다
```

**끊기는 자리가 전부 exec 앞이다.** 팩이 잘못된 단계는 하네스를 안 띄운다.

---

## 2. `runHarness` 에서 자라는 자리는 둘이다

U1 이 세운 아홉 걸음에 **하나가 끼고 하나가 자란다.**

```text
   ①    계장 디렉터리          그대로
   ①.5  openPack               새 걸음.  팩이 있을 때만 $IN 을 연다
   ②    resolveComponents      인자가 하나 는다 — packInput
                               Notes 를 찍는 자리는 그대로 오류 검사 앞이다
   ③    Argv                   그대로
   ④    Instrument             c.Pack 이 있으면 펴고 플래그를 하나 더 받는다
   ⑤    Fixed                  그대로
   ⑥    exec                   그대로
   ⑦    Decode · Version · 자백  그대로
   ⑧    logs/ 선별 + 기록       두 필드를 여기서 채운다 (R28 ~ R30)
   ⑨    defer 가 계장을 지운다   팩도 그 아래라 함께 지워진다
```

**①.5 가 ② 앞인 이유** — 검증 실패가 곧 단계 실패이고, 그 판정을 하는 것이 ② 다.
파일을 여는 것은 가장자리이고 등급과 문구를 정하는 것은 정책이다.

**⑧ 에서 채우는 이유** (답 7=A) — 거기가 exec 을 실제로 지난 유일한 자리다.
②나 ④에서 채우면 exec 앞에 죽은 단계의 봉인에도 값이 남아 「실린 것」이라는
필드의 뜻이 흐려진다.

---

## 3. `openPack` 과 `readPack` 의 걸음

```text
   openPack (가장자리 · 파일을 연다)
     1  j.Params.Pack 이 비면 빈 packInput 을 낸다            R1
     2  $IN/<이름> 을 연다.  없으면 Err 에 담는다              R2
     3  readPack 에 넘긴다.  결과와 Notes 와 오류를 담는다

   readPack (순수 · io.Reader 하나를 받는다)
     1  받은 바이트를 sha256 에 흘리면서 읽는다               2.2
     2  첫 두 바이트가 1f 8b 면 gzip 을 벗긴다                 R3
     3  tar 항목마다
          종류를 본다        정규 파일과 디렉터리만            R4
          이름을 본다        절대경로 · .. · 안전하지 않은 글자  R5 ~ R7
          센다              바이트와 개수                     R8 · R9
          가른다            skills/ · agents/ · mcp.json · 그 밖  R11 ~ R15
     4  남은 바이트를 끝까지 읽어 해시를 마친다                  2.2
     5  실을 것이 0 이면 거절한다                              R16
```

**3 의 순서가 뜻을 가진다** — 종류를 이름보다 먼저 본다. 심볼릭 링크의 이름이
성해 보일 수 있고, 그때 이름 검사만 통과시키면 링크가 살아 나간다.

**4 를 빠뜨리면 안 된다.** tar 는 꼬리에 0 블록을 달고, 거기서 멈추면 해시가
파일 전체의 값이 아니다 — 봉인에 남은 값으로 blob 을 대조할 수 없게 된다.

**1 과 4 가 R8 을 두 번 재게 한다** — 받은 바이트와 푼 바이트 중 먼저 닿는 쪽에서
끊는다.

---

## 4. `resolveComponents` 의 걸음 — 팩이 어디에 끼나

U4 가 남긴 여섯 걸음 앞에 둘이 끼고, 걸음 3 이 출처 셋을 본다.

```text
   0  팩 오류면 거절한다                        요청 여부와 무관하다
   1  팩이 있으면 c.Pack 에 담고 팩의 Notes 를 옮긴다
   2  요청 집합을 만든다.  비면 여기서 반환한다   c.Pack 은 담긴 채로 나간다
   3  워크스페이스를 못 읽었으면 거절한다
   4  이름 순으로 출처 셋을 뒤진다               팩 · 노드 · 워크스페이스
        팩에 있고 노드에도 있으면 거절            R19
        팩에만 있으면 팩 것                      R21 로 워크스페이스를 이긴다
        노드에 있으면 노드 것                    U4 의 규칙 그대로
        워크스페이스에만 있으면 그것
   5  집은 항목마다 종류를 채운다                 R22
   6  요청 안 한 이름을 Notes 로 남긴다           워크스페이스 · 팩
   7  못 찾은 이름이 있으면 거절한다
```

**0 이 2 앞인 것이 규칙이다.** 깨진 팩은 `agent.mcp` 가 비어도 단계를 죽인다 —
팩을 적은 것은 계약이고, 그 팩이 안 열리는데 조용히 도는 것이 「없음이 실패보다
나쁘다」의 그 자리다.

**1 이 2 앞인 것도 규칙이다.** 요청이 0 이어도 **스킬은 펴진다.** 계약이 서버를
안 적고 스킬만 쓰는 것이 정상이기 때문이다 — CA5 의 둘째 줄이 정확히 그 경우다.

**4 에서 팩을 가장 먼저 보는 이유**는 우선순위(팩 · 노드 · 워크스페이스)가 아니라
**거절을 놓치지 않기 위해서**다. 노드를 먼저 보고 집으면 팩이 같은 이름을 실은
사실을 못 본 채로 지나간다.

---

## 5. `Instrument` 의 ③ 자리

U1 이 비워 둔 자리에 그대로 앉는다.

```text
   ①  <dir>/home 을 만든다            치명
   ②  훅 설정                        보조 (errAux).  실패해도 아래를 마저 한다
   ③  팩                             치명.  c.Pack 이 nil 이면 아무것도 안 한다
        <dir>/pack 을 0700 으로 만든다
        Files 를 이름 그대로 0600 으로 쓴다
        --plugin-dir <dir>/pack 을 플래그에 붙인다
   ④  허용목록                        치명
   ⑤  자격증명                        치명
```

**②가 실패해도 ③이 돈다** — 조기 반환하지 않는다는 U1 의 불변식이다. 훅 하나가
실패했다고 나가면 팩도 허용목록도 안 쓰이고 치명도 안 난다.

**③이 ④ 앞인 것에 뜻은 없다.** 둘 다 치명이고 서로를 안 본다. 순서를 굳혀 두는
이유는 실패 문구가 어느 것에서 먼저 나오는지가 시험마다 흔들리지 않게 하려는 것뿐이다.

---

## 6. 거절이 봉인까지 가는 길

```text
   resolveComponents 의 오류
     -> HarnessResult{Reason: ReasonError, Message: <문구>}
     -> claim.go:784 의 res.Error
     -> steps/NN-*.json 과 GET /v1/runs/{id} 의 단계 error
     -> logs/NN-<단계>.log 는 0 바이트다 — 하네스가 안 떴다
```

**게이트가 그 문구를 부분 문자열로 찾는다.** 그래서 문구가 계약이다 — `R19` 의
`pack redefines node-declared mcp server <이름>` 이 CA5 의 판정 재료다.

**Notes 는 노드 로그로만 간다** (U4 의 값 그대로). 봉인을 읽는 사람은 겹침과
빠짐을 못 본다 — U4 가 이미 넘긴 한계이고 이 유닛이 늘리지 않는다.

---

## 7. CA5 의 네 줄이 어느 규칙으로 초록이 되나

```text
   ① 팩 단계 + agent.pack + agent.mcp: ["probe4"]
        -> init 의 skills 에 pack:hello · mcp_servers 에 probe4
        R11 이 hello 를 담고 R26 이 --plugin-dir 을 붙이고 R17 이 probe4 를 싣는다
        게이트 문장이 바뀐다 — hello 가 아니라 pack:hello 다 (답 1=A)

   ② agent.mcp 를 빼고 같은 팩으로
        -> mcp_servers 가 비고 스킬은 그대로 뜬다
        걸음 1 이 걸음 2 앞이라 그렇다.  요청 0 이 팩을 안 끈다

   ③ 팩의 mcp.json 에 노드가 선언한 이름(probe)
        -> 단계 error 에 pack redefines node-declared mcp server probe
        R19 다.  다만 답 6=A 라 그 이름을 agent.mcp 에도 적어야 한다 —
        게이트 문장에 그 한 줄을 더한다

   ④ 워크스페이스 .claude/skills/ws-skill
        -> 안 나타난다.  계획 1.3 이 이미 쟀고 게이트는 확인이다.
        판정을 decisions.md 에 적으라는 줄은 이 실측이 먼저 답을 냈다
```

**게이트가 읽는 필드가 둘이다** — `skills` 와 `slash_commands`. 스킬 이름은 둘 다에
있고 `slash_commands` 에는 내장 명령이 섞인다. 명령에 둘을 함께 적는다.

---

## 8. CA6 이 읽는 것

```text
   runctl record <id> -o r.tar && tar -xf r.tar
   jq '.harness | {mcp, pack}' run-<id>/steps/NN-<단계>.json
     -> mcp 는 그 단계에 실제로 실린 이름의 배열
     -> pack 은 64 글자 hex.  같은 Run 의 blob 을 받아 sha256sum 으로 맞춘다
```

**대조가 서는 것이 3.7 의 값이다** — 봉인된 묶음만 보고 그 단계가 어느 서버와
어느 팩으로 돌았는지 안다 (`ADR-005` 성질 4).

---

## 9. 오케스트레이터 템플릿의 단계 배치

```text
   설정이 비면          steps = [plan, gate]           오늘 그대로
   설정이 있으면        steps = [pack, plan, gate]

   pack   uses <executor.as> · run <fetch> · out [<name>]
   plan   needs 를 안 적는다 -> NeedsOf 의 기본값이 직전 단계라 pack 을 기다린다
   gate   needs: [plan]                                 오늘 그대로
```

**`in.from` 은 순서를 안 만든다** — 이름을 가리킬 뿐이다. 순서를 만드는 것은
`needs` 이고, 안 적으면 직전 단계가 기본값이다 (`contract.go:764-792`). 팩 단계를
맨 앞에 놓는 것만으로 그 뒤 전부가 그것을 딛는다.

**계획이 짓는 단계가 두 줄을 적는다** (답 8=A).

```text
   agent.pack: "<name>"     그 blob 을 이 단계에 싣는다
   in.from: ["<name>"]      $IN 에 그 이름으로 깔린다
```

`planPrompt` 가 그 이름과 두 키를 알려준다. 안 적으면 그 단계는 팩 없이 돈다 —
오늘 그대로다.

**`adapter_test.go` 의 `steps[0]`=plan · `steps[1]`=gate 는 그대로 초록이다** —
`testConfig()` 에 팩이 없기 때문이다. 팩이 있는 배치는 새 시험이 잰다.

---

## 10. 시험이 무엇을 잴 수 있나

**하네스 실행파일 없이 도는 시험으로 채운다** (`application-design.md` 6.2).

```text
   internal/enode/pack_test.go (새 파일)
     readPack 을 io.Reader 로 덮는다 — 악성 tar 를 메모리에서 짓는다.
     절대경로 · .. · 심볼릭 링크 · 장치 · 백슬래시 · 상한 둘 · gzip ·
     tar 가 아닌 것 · 깨진 mcp.json · 빈 팩 · 같은 이름 두 번 ·
     해시가 파일 전체의 값인 것 (꼬리 0 블록 뒤까지 읽는다)

   internal/enode/resolve_test.go (U4 의 파일에 더한다)
     팩의 서버가 필터를 타는 것 · 요청 0 에도 c.Pack 이 사는 것 ·
     노드 선언 충돌의 거절 문구 · 팩이 워크스페이스를 이기는 것 · Notes 넷

   internal/enode/instrument_test.go (U1 의 파일에 더한다)
     펴진 파일의 내용과 권한 · 디렉터리 0700 ·
     팩이 없으면 --plugin-dir 이 argv 에 없는 것 ·
     훅 쓰기를 실패시켜도 팩이 펴지는 것 (U1 의 불변식이 안 깨진다)

   internal/enode/harness_test.go 또는 runner 쪽
     기록 두 필드 — 실린 이름 순서 · 팩 없는 단계의 봉인이 오늘과 같은 것

   cmd/iapadapter/config_test.go · adapter_test.go
     설정이 비면 오늘 그대로 · fetch 가 비면 거절 ·
     팩이 있으면 steps[0] 이 팩 단계이고 out 이 그 이름인 것
```

**CA0 를 이 유닛에서도 돈다** — `internal/enode` 커버리지 80%. 새 코드가 가장
많은 유닛이므로 `pack_test.go` 의 표가 그 하한을 지는 자리다.

**시험이 못 재는 것은 게이트가 잰다** — 하네스가 그 디렉터리를 실제로 읽는가는
`--plugin-dir` 의 실측(계획 1.2)이 답했고, CA5 가 우리 코드가 지은 디렉터리로
그것을 다시 잰다.
