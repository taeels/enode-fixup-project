# U4 `sources` — Code Generation 요약

**유닛** `sources` · **브랜치** `unit/sources` · **담당** taeels ·
**닫는 게이트** CA4 (사람) · CA0 · **선행** U1 · U2 · U3

계획은 `../../plans/sources-code-generation-plan.md`, 설계는
`../functional-design/` 의 셋이다. 규칙 번호 R1 ~ R14 는 그 `business-rules.md`.

---

## 1. 낸 것

```text
   제품 파일 셋    mcp.go · runner.go · claim.go.  새 제품 파일 0
   시험 파일       새 파일 하나 + 고친 파일 둘
   패키지 하나     internal/enode.  cmd/ 와 internal/api 는 소스 diff 0
```

| 파일 | 무엇 | 규칙 |
|---|---|---|
| `internal/enode/mcp.go` | `Components.Servers` 타입 · `resolveComponents` 의 걸음 여섯 · `readWorkspaceMCP` · `ensureType` · `copyEntry` · `hasText` · `wantedMCP` · `entryNames` · `notAvailable` | R1 ~ R11 |
| `internal/enode/runner.go` | `Job` 의 필드 넷 · ② 뒤의 Notes 로그 | R10 · R11 |
| `internal/enode/claim.go` | 워크스페이스를 읽는 조건 둘 · `Job` 리터럴 네 줄 | R2 · R6 |

시험은 `resolve_test.go`(새)가 규칙의 짝이고, `worker_unix_test.go` 에 배선
시험 둘을 더했다. `mcp_test.go` 의 셋은 **타입이 바뀌어 확정 빨강이던 옛 시험**이다.

**`claude.go` 는 소스 diff 가 0 이다.** `Instrument` 가 `c.Servers` 를
`writeMCPAllowlist` 에 그대로 넘기므로 타입이 바뀌어도 그 줄이 안 바뀐다 —
계획 1절의 예상 그대로다.

---

## 2. CA0 — 전부 초록

```text
   go test ./... -count=1                 exit 0 · 패키지 18 초록 · 실패 0
   커버리지 (표준 명령 · 패키지별 80%)      미달 0 · 전체 6961/7974 = 87.3%
                                          internal/enode 1865/2158 = 86.4%
                                          (U3 뒤 86.0% 에서 올랐다)
   스킵                                   0
   go vet ./...                           exit 0
   gofmt -l .                             0 줄
   장식 문자 (U+2605 전수 grep)             0 히트
   go run ./scripts/glyphscan.go           104 파일 · 문자열의 장식 문자 0
   크로스 빌드                             windows/amd64 · linux/arm · darwin/arm64
   심볼 상한                               enodectl crypto/tls=1 · net/http=6 (상한 10 · 50)
   Mediator 라우트                          26 (안 늘었다)
   node --test (ui)                       75 통과 · 실패 0
   워킹트리 청결                            커버리지 실행이 바꾼
                                          cmd/enodectl/probe.lock 을 되돌렸다
```

---

## 3. 실측 — 우리가 쓴 파일을 하네스가 어떻게 읽나

계획 Step 6 이다. **복제본이 아니라 제품 코드가 낸 파일로 쟀다** —
`resolveComponents` 에 `Job` 하나를 넣고 `writeMCPAllowlist` 로 쓴 뒤 그 파일을
`claude 2.1.266` 에 `--strict-mcp-config --mcp-config=` 로 물렸다.

```text
   넣은 것    노드 probe (stdio) · 노드 gerrit (url + credential) ·
             워크스페이스 probe3 (command) · 워크스페이스 wsremote (type + headers)
   나온 파일   {"gerrit":{"type":"http","url":...},
              "probe":{"command":"/usr/bin/true","type":"stdio"},
              "probe3":{"command":"/usr/bin/true","type":"stdio"},
              "wsremote":{"headers":{...},"type":"http","url":...}}
   init 줄    mcp_servers 에 넷이 전부 있다 (가짜 주소라 status 는 failed)
```

**R9 를 안 넣었으면 `gerrit` 이 빠진 채로 초록이었다.** `ADR-035` §4.4 의 예시
그대로 적은 원격 선언이 `{"url": ...}` 가 되고, 그 항목은 목록에 이름조차
안 나온다 (FD 계획 1.3). 광고는 서고 서버는 안 열리는 조합이다.

**워크스페이스 항목의 `headers` 가 한 글자도 안 바뀌고 살아 나갔다** — 답 2=A 의
원문 통과가 실물에서 성립한다.

---

## 4. CA4 — 코드로 닫은 것과 사람 몫

```text
   닫았다    없는 이름 -> 하네스가 안 뜨고 res.Error 에 팩의 문구가 든다.
            worker_unix_test.go 의 배선 시험이 스텁의 마커 파일 없음으로 잰다
   닫았다    요청된 것만 허용목록에 실린다 (노드 probe · 워크스페이스 probe3 ·
            요청 안 한 probe2 는 0).  같은 시험이 그 단계가 실제로 쓴 파일을
            $OUT 으로 받아 읽는다
   남았다    실 함대에서 runctl record 로 푼 logs/ 의 첫 줄.  scene-gates.md
            3절의 명령이고 집행자는 이 유닛을 구현하지 않은 사람이다
```

3절의 실측이 그 사람 몫의 **위험을 미리 덜었다** — 우리 파일이 하네스에게
읽히는 것까지는 이미 참이다. 남은 것은 노드와 Mediator 를 지나는 배선이다.

---

## 5. 변이 여섯 — 다 빨개졌다. 구멍 0

넣고 `go test ./internal/enode/` 를 돌렸다.

```text
   ① 노드 우선순위를 뒤집는다        RED  TheNodeWinsOverTheWorkspace · NotesSurvive
   ② 종류 채우기를 걷는다            RED  TheTypeIsFilledIn · TheAllowlistCarries(배선)
   ③ 없는 이름의 거절을 걷는다        RED  AMissingNameStops · NotesSurvive ·
                                        AMissingMCPServerDoesNotLaunch(배선)
   ④ 워크스페이스 모양 검사를 걷는다    RED  AWorkspaceEntryWithoutCommandOrURL
   ⑤ 요청 필터를 걷는다              RED  OnlyRequestedNamesAreOpened · 배선
   ⑥ 얕은 복사를 걷는다              RED  TheJobsWorkspaceEntriesAreNotModified
```

**②와 ③과 ⑤는 순수 시험과 배선 시험이 함께 빨개졌다.** U1 에서 「함수만 부르고
배선을 안 쟀다」로 났던 구멍이 여기서는 안 생긴다 — 계획 Step 7 이 그것을 미리 걸었다.

---

## 6. 실측이 계획의 문장 둘을 고쳤다

```text
   ①  「종류 오류 하나가 파일 전체를 죽인다」가 지금 판에서는 안 일어난다
      계획 1.2 는 MCPServer 로 받던 판을 잰 것이다.  원문(map)으로 받으면
      {"command": 5} 가 파일을 안 죽이고, 그 항목이 요청될 때 R8 이 문장으로
      거절한다.  번역을 안 하니 번역의 실패도 없다 — 답 2=A 가 덤으로 닫았다.
      R7 이 남는 자리는 JSON 이 아닌 파일 · mcpServers 가 목록인 파일 ·
      항목이 객체가 아닌 파일이고 시험이 셋을 잰다

   ②  features.md 3.2 에 종류가 이미 있었다
      계획 Step 10 은 「종류를 더한다」로 적었는데 그 줄은 이미 있었다.
      더한 것은 그것이 type 키이고, 선언이 안 적으면 계장이 채우며, 안 채우면
      하네스가 말없이 뺀다는 줄이다.  그리고 워크스페이스와 팩의 항목을
      원문 그대로 옮긴다는 줄을 함께 넣었다
```

**FD 문서 한 줄도 고쳤다** — `domain-entities.md` 2.1 이 종류를
`allowlistEntry` 가 짓는 것처럼 읽혔다. 실제로는 합친 뒤의 공용 자리다
(계획 4절 ③). 최종 항목에 종류가 있다는 사실은 그대로 참이다.

---

## 7. 이 유닛이 회차 밖으로 낸 것

```text
   decisions.md 6절   실측 행 ㉓ — 워크스페이스 파일의 어휘 · url 만 적힌 항목의
                      침묵 · ${} 가 한 겹만 펴진다 · 모르는 키는 항목을 안 죽인다.
                      제품 코드가 낸 파일로 다시 잰 것까지 한 행에
   decisions.md 2절   허용목록 행에 「종류가 없으면 계장이 채운다」
   features.md 3.2    종류가 type 키라는 줄 · 원문 그대로 옮긴다는 줄
   파일 행렬           1절의 새 시험 파일 · 2.1 의 U4 칸 · 2.2 의 U4 칸 ·
                      3절의 옛 시험 둘 · 4.1 의 U4 문단
   scene-gates.md     안 고쳤다 — 답 어느 것도 CA4 의 명령을 안 바꾼다
   짝 팩과의 접점       runner.go 하나.  Job 에 필드를 더했을 뿐 스트림 처리
                      자리(⑱ · ⑲)는 안 건드렸다
```

---

## 8. 진행자에게 넘기는 것 — 다섯

```text
   1  원격 노드 선언만으로는 인증이 안 실린다 (답 7=A)
      credential 은 허용목록에 안 나가고 원격 MCP 는 헤더로 인증한다.
      소유자가 headers 를 enode.yaml 에 적어야 돌고(Extra 로 지나간다)
      그 사실이 ADR-035 §4.4 의 예시에도 팩에도 없다.  정본 개정 후보다

   2  깨진 워크스페이스 파일이 MCP 를 쓰는 단계를 전부 죽인다 (R7)
      요청한 이름이 전부 노드 선언에 있어도 그렇다 — 읽는 시점이 이름을
      고르기 전이다.  답 4=A 가 고른 값이고 대가를 business-rules 5절이 적는다

   3  워크스페이스 파일에 크기 상한이 없다 (계획 4절 ①)
      같은 워크스페이스를 collectDeclared 와 diff 가 이미 통째로 읽으므로
      규율을 맞춘 것이다.  그 규율 자체를 바꾸려면 세 자리를 함께 본다

   4  종류를 잘못 채우는 경우가 남는다 (R9)
      sse 서버를 url 만으로 적으면 http 로 채워지고 failed 로 나타난다.
      침묵보다는 낫지만 옳은 것은 아니다 — 소유자가 type 을 적으면 우리가 안 덮는다

   5  Notes 는 노드 로그에만 남는다 (답 5=A)
      단계가 끝난 뒤 봉인을 읽는 사람은 겹침과 빠짐을 못 본다.
      U5 의 HarnessResult.MCP 가 그 자리를 일부 받는다
```

---

## 9. 다음 유닛이 딛는 자리

```text
   U5 팩     걸음 3 앞에 팩 출처를 더한다 — 팩이 노드 선언 이름을 덮으면 거절이다.
            팩의 mcp.json 도 하네스 어휘라 워크스페이스와 같은 길로 원문이 간다.
            여는 것은 가장자리라는 규율도 그대로다 — tar 를 읽는 것은
            resolveComponents 가 아니다
   Notes    「팩이 실었으나 요청 안 한 이름」이 R10 의 넷째 줄로 들어간다
   필터      R1 은 안 바뀐다.  출처가 느는 것은 선택지가 느는 것이지
            권한이 느는 것이 아니다
```
