# U3 `advert` — Code Generation 요약

**유닛** `advert` · **브랜치** `unit/advert` · **담당** taeels ·
**닫는 게이트** CA2 (사람) · CA3 완결 · CA0 · **선행** U1 · U2

계획은 `../../plans/advert-code-generation-plan.md`, 설계는
`../functional-design/` 의 셋이다. 규칙 번호 R1 ~ R14 는 그 `business-rules.md`.

---

## 1. 낸 것

```text
   제품 파일 넷    config.go · mcp.go · detect.go · format.mjs
   시험 파일       새 파일 셋 + 고친 파일 둘
   패키지 둘       internal/enode · internal/api/ui.  cmd/ 는 소스 diff 0
```

| 파일 | 무엇 | 규칙 |
|---|---|---|
| `internal/enode/config.go` | `Local.MCP` · `validateMCP` · `validateEnvNames` · `isEnvName` · `SampleLocal` 한 줄 | R1 ~ R6 |
| `internal/enode/mcp.go` | 태그 다섯 · `Extra` · `UnmarshalYAML` · `kind` · `mcpUp` · `mcpFP` · `envRefs` · `allowlistEntry` | R3 · R9 |
| `internal/enode/detect.go` | `Fingerprinter` · `fingerprinters` · `harnessFP` · `repoFP` · 순회 · `hasCapability` · `warnMCPWithoutCapability` | R7 · R8 · R10 ~ R12 |
| `internal/api/ui/static/shared/fleet/format.mjs` | `attributeFamilies` · `attributeFamily` · `attributeName` · `attributeValue` | R13 · R14 |

시험은 `config_mcp_test.go`(새) · `detect_mcp_test.go`(새) ·
`tests/format.test.mjs`(절 하나 추가) 셋이 새 코드의 짝이고,
`mcp_test.go` 와 `enode_test.go` 둘은 **옛 시험이 새 사실을 만나 고친 것**이다 (5절).

---

## 2. CA0 — 전부 초록

```text
   go test ./... -count=1                 exit 0 · 패키지 18 초록 · 실패 0
   커버리지 (표준 명령 · 패키지별 80%)      미달 0 · 전체 6887/7902 = 87.2%
                                          internal/enode 1795/2086 = 86.0%
                                          internal/api/ui 61/62 = 98.4%
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

**`internal/enode` 가 85.6% 에서 86.0% 로 올랐다.** FD 5절이 「하한에 가장 가까운
자리가 여기다」로 적었는데, 새 코드가 들어간 뒤에도 여유가 여섯 포인트다.

---

## 3. 실측 — `env` 의 참조를 하네스가 편다

계획 Step 6 이 이 단계에 맡긴 미정 하나다. **이 기계의 `claude 2.1.266`** 으로 쟀다.

```text
   준 것    mcp.json 에 stdio 서버 하나.  command 는 자기 환경을 파일로 쏟는
            스크립트이고 env 는 {"PROBE_X": "${SRC}", "PROBE_LITERAL": "plain"}
            하네스 환경에 SRC=expanded-value-42
   명령     claude -p ... --output-format stream-json --verbose
            --strict-mcp-config --mcp-config=<경로>
   본 것    PROBE_X=expanded-value-42     편다
            PROBE_LITERAL=plain           원문은 그대로 간다
```

**FD 의 대안 하나가 닫혔다.** FD `domain-entities.md` 3절이 「안 펴면 `env` 를
허용목록에 안 쓰고 노드 환경 상속에 맡긴다」를 열어 두었는데, 펴므로 참조 꼴이
확정이다. `envRefs` 가 값 `V` 를 `"${V}"` 로 적는다.

**덤으로 U1 의 ⑳ 이 다시 섰다** — 첫 줄이 `system/init` 이 아니라
`system/hook_started` 였다. 개인 홈의 SessionStart 훅이 앞에 사건을 낸다는
그 실측이 이 기계에서도 그대로다.

---

## 4. CA3 의 뒤 절반 — 합성 함대로 쟀다

Mediator 를 시험 DB 위에 띄우고 노드 둘을 `POST /v1/nodes` 로 심었다 —
하나는 `mcp.probe` 를 가졌고 하나는 안 가졌다.

```text
   runctl capabilities        agent.reason  nodes: 2
                                harness         claude
                                harness.claude  1
                                mcp.probe       1
                              -> 계약 작성자가 보는 면에 새 키 둘이 나온다.  코드 0

   requires: mcp.probe        ca3-probe RUNNING · brain -> node-with-probe
                              -> 그 노드에만 간다

   requires: mcp.nope         422 need 1, fleet has 0 -
                                agent.reason harness=claude mcp.nope=1

   mcp 를 안 요구하는 계약     brain -> node-plain
                              -> 희소한 것이 저절로 남는다 (ADR-027 §4.2).
                                 mcp.<이름> 이 attrCount 를 늘려 그 노드가
                                 뒤로 밀린다 — 매처 코드 diff 0 으로 얻은 성질
```

**셋 다 코드 0 으로 닫혔다.** `store.Capabilities` 가 키 이름을 안 가리고
`Capability.Satisfies` 가 완전 일치를 보므로, 매처도 저장소도 라우트도 새 이름을
알 필요가 없다 — `ADR-012` 의 「어휘는 창발한다」가 실물로 섰다.

**게이트를 돌릴 사람이 밟을 자리 하나를 적어 둔다.** 같은 `run_id` 로 두 번
제출하면 둘째는 `422` 가 아니라 `200` 이고 본문이 `FAILED` 와 저장된 `reject` 다
(멱등이다). `422` 를 보려면 **새 `run_id`** 로 넣어야 한다.

---

## 5. 변이 다섯 — 다 빨개졌다. 구멍 0

```text
   ① hasCapability 의 mcp. 접두 되돌리기      빨강
   ② allowlistEntry 의 얹는 순서 뒤집기        빨강
   ③ mcpUp 의 remote 갈래를 언제나 참으로      빨강
   ④ harnessFP 에서 옛 harness 키 빼기         빨강
   ⑤ repoFP 의 WorkspaceID fallback 빼기       빨강
```

U1 은 변이 넷 중 하나에서 구멍을 찾았고(배선을 안 재는 시험) 여기는 0 이다.
이유는 U2 와 같다 — 규칙이 전부 작은 순수 함수 하나를 지나고, 그 함수를 부르는
자리가 하나씩이다.

### 5.1 옛 시험 둘이 확정 빨강이었다

계획이 예고한 것이 아니라 **돌려서 만났다.** 둘 다 고치는 것이 옳았다.

```text
   enode_test.go TestDetectEmpty
     광고에 실릴 수 있는 키를 harness · os · host_arch 로 못 박는다.
     harness.<이름> 이 늘면서 빨개졌다 — 그 접두를 함께 허용했다

   mcp_test.go TestMCP_AServerIsCopiedIntoTheAllowlist
     env 가 값 그대로 실리는 것을 쟀다 (env["MCP_FS_MODE"] == "ro").
     답 3=A 가 그것을 ${이름} 참조로 바꾸므로 픽스처와 단언을 함께 고쳤다
```

**둘 다 행렬 밖이었고 행렬 3절에 행으로 더했다.**

### 5.2 시험 픽스처 하나가 검사의 한계를 드러냈다

`env` 의 값으로 `ghp_secret` 을 적고 거절을 기대했는데 **통과했다** —
`[A-Za-z_][A-Za-z0-9_]*` 를 그대로 만족한다. FD `business-rules.md` 2절이
이미 「꼴 검사는 값을 적지 말라를 다 못 잰다」로 적은 그 자리다. 픽스처를
실제로 걸리는 값(`sk-ant-api03-abc` · 공백이 든 문장)으로 고치고,
**그 한계를 재는 시험을 따로 더했다** — `ghp_looks_like_a_name` 이 통과하는
것을 단언한다. 그 줄이 빨개지면 검사가 넓어진 것이고, 그때 `features.md` 3.2 와
함께 다시 본다.

---

## 6. 계획에 없던 자리 둘

```text
   ① 거절의 검사 순서       FD 의 표는 R1 ~ R5 를 번호로만 적었다.  좁은 것을
                           먼저 본다 — args 만 적힌 서버에 「needs command or url」
                           을 주면 사람이 args 를 지우려 하고 실은 command 를
                           빠뜨린 것이다.  순서는 R1 · R3 · R4 · R5 · R2 다

   ② mcpUp 의 문구가 둘     계획은 「command %q not found in PATH」 하나였다.
                           절대경로는 PATH 를 안 보므로 그 문구가 거짓이 된다.
                           경로 구분자가 있으면 「is not an executable file」 이다
```

---

## 7. 이 유닛이 회차 밖으로 낸 것

```text
   decisions.md 6절   실측 행 ㉒ — 넷을 한 행에 (yaml 이 종류만 잡는다 ·
                      「그 밖의 키」가 오늘 안 생긴다 · env 의 뜻과 그 실측 ·
                      현황판이 원문 키로 낸다).  6절 머리의 시각도 고쳤다
   decisions.md 2절   노드 선언 형식 행에 env 의 뜻을 박았다
   파일 행렬           2.1 의 U3 칸을 「U1 의 것을 고친다」로 (4.5 절이 그 이유) ·
                      format.mjs 와 시험 넷을 행으로 · 만지는 파일 16 -> 17 ·
                      만지는 패키지 3 -> 4 · 3절에 옛 시험 둘 · 4.4 · 4.5 절
   scene-gates.md     안 고쳤다.  답 어느 것도 게이트 명령을 안 바꾼다
   component-methods.md  안 고쳤다.  답 4=A 가 Probe 의 시그니처를 그대로 뒀다
   짝 팩과의 접점      0.  runner.go 도 claude.go 도 안 만졌다
```

---

## 8. 진행자에게 넘기는 것 — 여섯

```text
   ①  CA2 는 사람이 잰다           집행자는 이 유닛을 구현하지 않은 사람이다
                                  (scene-gates.md 2절 머리).  눈 검증 S1 도 그 사람이
                                  본다 — 보류로 안 넘긴다 (그 문서 4절)

   ②  완료 조건 하나와 어긋난다     unit-of-work.md 3절의 「cheapAttrs · capabilities ·
                                  Detector 에 diff 0」이 글자로는 서고 뜻으로는 안 선다 —
                                  답 6=B 가 hasCapability 를 고쳐 capabilities 의
                                  동작이 바뀐다.  회차 문서라 안 고쳤다

   ③  R12 가 광고 주기마다 나온다   capabilities 가 광고마다 불린다 (탐지 주기가 아니다).
                                  mcp: 를 적고 하네스가 없는 노드에서만 나오고,
                                  그것은 사람이 설정을 고쳐야 끝나는 상태다.
                                  실제 빈도는 광고 주기에 달렸다

   ④  Local 의 필드가 열둘이 됐다   그 형식의 주석이 「열 줄을 넘기지 않는 것이
                                  원칙이다」로 적는데 오늘 이미 열하나였다.
                                  원칙을 고칠지는 그 주석의 주인이 정한다

   ⑤  라벨로 선언 없이 광고할 수 있다  labels: { "mcp.probe": "1" } 를 적으면 탐지가
                                  그 키를 안 냈으므로 라벨이 그대로 실린다.
                                  features.md 3.3 의 「노드 설정에 안 적힌 MCP 는
                                  광고되지 않는다」는 mcp: 절을 두고 한 말이고
                                  라벨도 소유자의 선언이라 같은 신뢰 경계 안이다.
                                  오늘 그대로의 성질이고 이 유닛이 안 좁혔다

   ⑥  Extra 는 읽기만 있다          MarshalYAML 이 없다.  setup 이 설정을 새로 지어
                                  쓰기만 하고 읽어서 다시 쓰는 경로가 저장소에 0 이라
                                  오늘 잃을 것이 없다.  그 0 이 깨지는 날 함께 서야
                                  한다 — 안 그러면 소유자가 적은 모르는 키가
                                  다시 쓰기에서 조용히 사라진다
```

**작은 것 하나** — `runctl capabilities` 의 서식이 `%-10s` 라
`harness.claude`(14자)가 칸을 넘어 값이 한 칸 밀려 찍힌다. 읽는 데 지장은 없고
`cmd/runctl` 의 소스 diff 0 을 지키려 안 고쳤다.

---

## 9. 다음 유닛이 딛는 자리

```text
   U4 sources   Local.MCP 를 출처 하나로 읽는다.  형식(필드 여섯)과 거절 여섯을
                그대로 쓰고 그 위에 agent.mcp 필터와 이름 없음의 거절을 얹는다.
                allowlistEntry 는 이 유닛이 최종형으로 세웠다 — Extra 를 얹고
                아는 키가 덮는다

   U5 pack      팩이 노드 선언 이름을 덮으면 거절한다 (decisions.md 6절 ⑦).
                그 「노드 선언」이 Local.MCP 이고 이름 공간의 주인이 여기다
```
