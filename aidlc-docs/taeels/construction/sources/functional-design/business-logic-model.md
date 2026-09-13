# U4 `sources` — 값이 지나가는 길

형식은 `domain-entities.md`, 규칙은 `business-rules.md`. 여기는 **그 규칙이
언제 어디서 적용되나**다.

---

## 1. 한 줄기

```text
   enode.yaml  mcp:            .mcp.json  mcpServers        계약  agent.mcp
        |                           |                            |
        | LoadLocal (U3 가 검증)      | readWorkspaceMCP           | parseAgentParams (U2)
        v                           v                            v
   Local.MCP                  원문 객체들                      Params.MCP
        |                           |                            |
        +------ claim.go 가 Job 에 싣는다 (요청이 0 이면 파일을 안 연다) ------+
                                    |
                                    v
                        resolveComponents(j)      <- 정한다.  파일을 안 만진다
                                    |
                    +---------------+---------------+
                    |                               |
              Components.Servers               Components.Notes
                    |                               |
                    | Instrument 가 쓴다             | runHarness 가 Info 로 찍는다
                    v                               v
        <계장>/mcp.json                        노드 로그
                    |
                    | --strict-mcp-config --mcp-config=<경로>
                    v
             하네스의 init 줄 mcp_servers      <- 게이트가 읽는 자리
```

**세 출처가 만나는 자리는 하나이고 파일을 여는 자리는 그 밖이다.** 이것이 답
1=A 가 지킨 가름이다 — 여는 것은 가장자리(`claim.go`), 고르는 것은
`resolveComponents`, 쓰는 것은 `Instrument`.

---

## 2. `runHarness` 에서 자라는 자리는 ② 하나다

U1 이 호출 순서의 최종형을 이미 세웠다 (`unit-of-work-file-matrix.md` 2.2).

```text
   ①  계장 디렉터리       그대로
   ②  resolveComponents  몸통이 자란다.  여기가 이 유닛이다
   ③  Argv               그대로
   ④  Instrument         그대로.  c.Servers 의 타입만 바뀐다
   ⑤  Fixed(tmp)         그대로
   ⑥  exec               그대로
   ⑦  Decode             그대로
   ⑧  logs/ 선별          그대로.  Notes 를 여기 안 싣는다
   ⑨  defer 가 지운다      그대로
```

**②와 ③ 사이에 로그 한 줄이 는다** — resolve 가 끝나면 Notes 를 찍는다.
거절로 끝날 때도 찍고 그 뒤에 실패를 낸다 (`business-rules` R11).

**치명이 전부 exec 앞이라는 불변식이 이 유닛에서 실제로 쓰인다.** U1 때는
`resolveComponents` 가 오류를 낼 수 없었다 — 언제나 빈 것을 냈다. 이 유닛이
거절 셋(R4 · R7 · R8)을 그 자리에 넣는 것이 그 불변식의 첫 소비다.

---

## 3. `resolveComponents` 의 걸음

```text
   1   want = agent.mcp 의 이름 집합.  비면 빈 Components 를 낸다 (R2)
   2   WorkspaceMCPErr 가 있으면 거절한다 (R8)
   3   want 를 이름 순으로 돈다
         3a  노드에 있다        allowlistEntry() 로 항목을 짓는다.
                              워크스페이스에도 있으면 Note (R3)
         3b  워크스페이스에 있다   원문을 그대로 집는다.  command 도 url 도
                              없으면 거절한다 (R7)
         3c  어디에도 없다       missing 에 담는다
   4   집은 항목마다 type 이 없으면 채운다 (R9)
   5   워크스페이스가 선언했으나 want 에 없는 이름을 Note 에 담는다 (R10)
   6   missing 이 있으면 첫 이름으로 거절하고 나머지를 Note 에 담는다 (R4 · R5)
```

**순서가 뜻을 가진다.**

```text
   2 가 3 보다 앞     깨진 파일이 「없는 이름」으로 둔갑하지 않는다.
                    원인이 파일이면 문구도 파일을 가리킨다
   3 이 4 보다 앞     type 은 우리가 채우는 유일한 키다.  집기 전에 채우면
                    무엇을 채웠는지가 출처와 섞인다
   6 이 맨 뒤        거절로 끝나도 Notes 가 다 채워진 뒤다 (R11)
```

**1 이 파일을 안 여는 것과 짝이다.** `claim.go` 가 같은 조건(`len(Params.MCP) == 0`)
으로 `readWorkspaceMCP` 를 건너뛴다. 조건이 두 자리에 있지만 값은 하나다 —
**요청이 없으면 출처도 없다.**

---

## 4. 거절이 봉인까지 가는 길

```text
   resolveComponents  error
        |
        v
   runner.go:96   HarnessResult{Reason: ReasonError, Message: err.Error()}
        |
        v
   claim.go:782   h.Reason.Completed() 가 거짓 -> $OUT 을 안 수확한다
        |
        v
   claim.go:784   res.Error = "harness: error mcp server nope is not available on this node"
        |
        v
   report -> Mediator -> steps/NN-*.json 의 봉인
```

**새 `Reason` 을 안 만든다** (`unit-of-work.md` 4절의 완료 조건). 하네스가 안
뜬 것은 하네스 오류의 한 종류이고, `Completed()` 가 거짓이라 반쯤 쓴 산출물이
수확되지 않는다. 게이트는 `res.Error` 안에서 **부분 문자열**로 문구를 찾는다.

---

## 5. CA4 의 세 줄이 어느 규칙으로 초록이 되나

노드가 `probe` · `probe2` 를 선언하고 워크스페이스 `.mcp.json` 에 `probe3` 이
있는 판이다 (`scene-gates.md` 3절).

```text
   agent.mcp: ["probe"]    R1 이 probe2 를 떨어뜨리고 R2 가 워크스페이스를
                           안 읽는 조건이 아니므로 파일은 읽되 probe3 은 3 의
                           want 밖이라 Note 로만 남는다.  init 의 mcp_servers
                           이름이 probe 하나다

   agent.mcp: ["probe3"]   3b 가 원문을 그대로 집고 R9 가 type 을 채운다.
                           init 에 probe3 이 나타난다.  광고에는 여전히 안 실린다
                           (R12) — 그 노드의 attrs 에 mcp.probe3 이 없다

   agent.mcp: ["nope"]     3c -> 6 -> R4.  하네스가 안 뜨고 단계 error 가
                           "mcp server nope is not available on this node" 를 담는다
```

**셋째 줄이 이 유닛의 값을 가장 잘 보인다** — 요청한 것이 없으면 **안 돌린다.**
조용히 빼고 도는 것이 `ADR-035` §3 이 「없음이 실패보다 나쁘다」로 부른 실패다.

---

## 6. 오늘 도는 계약은 안 바뀐다

```text
   agent.mcp 가 없는 계약    R2 로 빈 허용목록.  U1 뒤와 완전히 같다 —
                           파일도 안 열고 Note 도 없다
   워크스페이스에 .mcp.json 이 없는 노드   빈 것이고 오류가 아니다
   워크스페이스가 없는 노드    R6 으로 출처가 없다
   광고                     코드 diff 0.  detect.go 를 안 만진다
```

**동작 중립이 이 유닛의 안전망이다.** 기존 시험이 그대로 초록이어야 하고,
빨개지는 것은 `Components.Servers` 의 타입이 바뀌어 닿는 자리뿐이다
(`mcp_test.go` · `instrument_test.go`).

---

## 7. U5 가 딛는 이음매

```text
   필터        R1 이 팩에도 그대로 적용된다.  U5 는 출처를 더할 뿐 필터를 안 만든다
   집는 자리    3 의 이름 순회.  팩은 3a 앞에 선다 — 팩이 노드 선언 이름을
              덮으면 거절이기 때문이다 (decisions.md 6절 ⑦)
   나르는 꼴    팩의 mcp.json 도 하네스 어휘다.  워크스페이스와 같은 길로
              원문 그대로 간다 (답 2=A)
   여는 자리    가장자리다.  팩 tar 를 읽는 것도 claim.go 나 runner.go 이고
              resolveComponents 는 파일을 계속 안 만진다 (답 1=A 의 대가)
   Notes       「실었으나 요청 안 한 서버」가 R10 의 넷째 줄로 들어간다
```

---

## 8. 시험이 무엇을 잴 수 있나

`resolveComponents` 가 파일도 프로세스도 안 쓰므로 **`Job` 하나를 넣고 결과와
오류를 재는 것으로 규칙 전부가 덮인다** (`unit-of-work.md` 4절의 완료 조건).

```text
   Job 만으로 잰다   R1 · R2 · R3 · R4 · R5 · R7 · R9 · R10 · R11
   디스크가 필요하다  readWorkspaceMCP 의 넷 — 없는 파일 · 봉투 없는 파일 ·
                   깨진 파일 · 성한 파일.  t.TempDir() 하나면 된다
   함대가 필요하다    CA4.  사람이 실제 하네스로 띄운다 (scene-gates.md 4절 —
                   눈 검증을 보류로 안 넘긴다)
```

목록과 파일별 diff 는 이 유닛의 Code Generation 계획이 낸다.
