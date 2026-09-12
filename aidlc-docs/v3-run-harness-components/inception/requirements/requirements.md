# Requirements — 하네스 구성요소 (v3-run-harness-components)

요구 팩 `requirements/harness-components/` 다섯 파일이 입력이다. 이 문서는 그
팩을 **오늘의 코드에 대서** 확정한 것이고, 팩을 대체하지 않는다. 어긋나면
`enode-design/protocol/INVARIANTS.md` 가 이긴다 (`canon.md`).

깊이는 **Comprehensive** 다. 근거 셋 — 새로 만드는 표면이 자격증명과 실행 경계에
걸리고, 차단성 확장(security-baseline)이 켜져 있고, 짝 팩과 같은 파일을 만진다.

---

# 1. 의도 분석

```text
   사용자 요청      "하네스 구성요소 aidlc 하자"
   요청 종류        Enhancement — 정본이 이미 결정한 것을 코드로 내린다.
                    ADR-034 · ADR-035 는 결정인데 코드가 0 이고, ADR-067 은 초안이다
   범위             Multiple Components — internal/enode · internal/contract ·
                    cmd/iapadapter · cmd/runctl 넷.  Mediator 라우트는 0 개 는다
   복잡도           Complex — 자격증명 · 실행 경계 · 탐지 주기 · 계약 문법이 얽힌다
   위험             High — 가짜 홈이 하네스 인증을 끊을 수 있고(OAuth 노드),
                    허용목록이 틀리면 사내 MCP 가 다시 막힌다.
                    롤백은 Moderate — 회차 브랜치라 되돌리기가 한 번이다
```

## 1.1 왜 지금인가

사내 실측(2026-09-11)이 「하네스 환경에 설치된 MCP 가 거부된다」를 봤다. 로컬
실측이 그 거부가 **거꾸로** 나 있음을 보였다 (`features.md` 1.2 의 표) —
`ADR-067` 이 막으려는 계정 커넥터는 통과하고, 열어야 한다고 한 것(노드 ·
워크스페이스가 명시한 서버)은 막힌다. 오늘 코드에 있는 것은
`--setting-sources ""` 한 겹뿐이다.

## 1.2 가치의 고정점

`scene-gates.md` 1절의 장면 하나다.

**사내 MCP 하나를 요구한 계약이 그 MCP 를 가진 노드에서만 돌고, 하네스는 계약이
요청한 서버와 팩의 스킬만 본다. 계정 커넥터는 0 이다. 무엇을 물렸는지가 Record 에
남는다.**

완결성은 그 장면(CA6)이 끝까지 도는가로 판단한다. 유닛 완료 개수로 재지 않는다.

---

# 2. 착수 전 실측 — 팩을 코드에 댔다

팩이 `features.md` 1.3 과 `canon.md` 2절에 세는 명령을 적어 두었다. 그대로
돌렸다. 기준선은 `06215ff` (공용 R/E 스캔 직전 커밋), 오늘은 `7d92bfd` 다.

| 팩의 기술 | 실측 | 판정 |
|---|---|---|
| `grep -rn 'strict-mcp-config\|CLAUDE_CONFIG_DIR\|"pack"' internal cmd` 가 0 줄 | 0 줄 | 맞다 |
| `detect.go` 가 하네스 하나에서 `break` 한다 | `detect.go` 의 `costlyAttrs` 가 `attrs["harness"] = h.Name()` 뒤 `break` | 맞다 |
| `Fixed()` 는 `CLAUDE_CODE_DISABLE_AUTO_MEMORY` 하나 · `hook.go` 가 `--settings` 와 `--setting-sources ""` 를 낸다 | `claude.go:69` · `hook.go:343` | 맞다 |
| **`AgentParams` 의 알려진 키 검증이 없다** | `internal/contract/contract.go:902` 의 `agentKeys` 와 같은 파일의 `knownKeys` 가 이미 거절한다 (ADR-057) | **틀리다** |

## 2.1 교정 하나 — 알려진 키 검증은 이미 있다

`agentKeys = []string{"model", "max_turns", "max_tokens", "ask", "harness"}` 이
정본 자리로 이미 서 있고, `Contract.Validate` 가 `knownKeys(st.ID, "agent",
st.Agent, agentKeys)` 로 부른다. 모르는 키는 이미 거절되고 Mediator 에서 `400`
이다.

그래서 **`features.md` 3.5 의 그 항목은 새 검증을 만드는 일이 아니라 목록에
이름 둘(`mcp` · `pack`)을 더하는 일**이다. `canon.md` 2절의 `ADR-034` §5 ④ 줄도
「이 팩」이 아니라 「이미 있다」로 바뀐다.

범위가 주는 방향이므로 팩의 게이트 규칙대로 묻지 않았다. `decisions.md` 에
행으로 적는다 (7절).

## 2.2 실측이 새로 보인 것 — 계장 임시 디렉터리는 이미 있다

`runner.go:63` 이 `os.MkdirTemp("", "enode-inst-*")` 로 계장 임시 디렉터리를
짓고 `defer os.RemoveAll(tmp)` 로 단계 끝에 지운다. 훅 설정도 이미 거기 산다
(`Instrument(tmp, self, a)` · `runner.go:77`). 가짜 홈이 앉을 자리가 이미
있다는 뜻이고, 팩의 「계장 임시 디렉터리와 함께 단계 끝에 지워진다」가 새
정리 코드를 요구하지 않는다.

**그런데 `Fixed()` 가 그 경로를 모른다.** 인터페이스가
`Fixed() map[string]string` 이라 인자가 없고, `runner.go:88` 이
`for k, v := range h.Fixed()` 로 부른다. `CLAUDE_CONFIG_DIR=<tmp>/home` 을
「값으로 박는다」(3.1)는 요구는 **인터페이스의 모양을 건드린다** —
`Fixed(dir string)` 로 바꾸거나, 홈 변수만 `runHarness` 가 합치거나 둘 중
하나다. `constraints.md` 의 구조 불변식은 이 자리를 안 적었다. Application
Design 이 고른다 (8절 D1).

## 2.3 공용 R/E 의 부분 재측정

질문 1 의 답이 B 였다. 이 팩이 딛는 네 경로만 다시 쟀고
`aidlc-docs/inception/reverse-engineering/` 의 `code-structure.md` ·
`component-inventory.md` ·`reverse-engineering-timestamp.md` 에 그 결과를 실었다.

```text
   internal/enode      비테스트 소스 일곱이 바뀌고 셋이 새 파일 —
                       policy.go · status.go · transcript.go
   internal/contract    advert.go 에 Advert.Policy · Drain 어휘 셋
   cmd/iapadapter       무변경
   cmd/runctl           무변경
```

**`transcript.go` 가 이미 있다는 것이 이 팩에 걸린다.** `Job` 에 `Emit` 과
`Transcript` 가 이미 있고 `runner.go` 가 하네스 stdout 을 링 파일로 tee 한다.
짝 팩(transcript)이 딛는 바닥의 일부가 이미 서 있다는 뜻이고, 접점의 크기가
`constraints.md` 가 적은 것보다 작다.

안 잰 자리의 낡음은 관측값으로만 적었다 — `internal/api` 라우트 15 -> 17 ·
`internal/` 패키지 10 -> 12 · Go 파일 143 -> 190. 이 팩이 그 자리를 안 만진다.

---

# 3. 기능 요구사항

팩 `features.md` 3절이 정본이다. 여기는 그것을 **확정 · 교정 · 추가**로 가른 것이고
번호를 그대로 잇는다.

## FR-1 (팩 3.1) 가짜 홈

계장이 홈을 짓고 고정값이 그 주소를 박는다. 개인 설정 · 계정 커넥터 · 자동
메모리가 실행에 안 섞이고, 팩이 펴질 자리가 생긴다.

```text
   확정   CLAUDE_CONFIG_DIR=<계장 임시 디렉터리>/home 을 값으로 박는다.
          통과 목록에 안 넣는다 — 넣으면 노드에 그 변수가 없을 때 조용히 안 걸린다
   확정   --setting-sources "" 를 유지한다.  가짜 홈과 끊는 것이 다르다
   확정   CLAUDE_CODE_DISABLE_AUTO_MEMORY=1 을 유지한다
   확정   계장 실패와 독립이다.  계장이 실패해도 홈은 갈아끼워져 있고 비어 있다
   확정   OAuth 노드의 ~/.claude/.credentials.json 을 가짜 홈에 0600 으로 복사.
          없으면 아무것도 안 한다.  Usable() 은 실제 홈으로 재는 오늘 그대로
   교정   계장 임시 디렉터리와 그 정리는 이미 있다 (runner.go:63 · defer RemoveAll)
   추가   Fixed() 가 그 경로를 모른다.  인터페이스 모양이 걸린다 (2.2 · D1)
```

**수용 기준**: CA1.

## FR-2 (팩 3.2) MCP 허용목록

파일 하나가 정책이다. `<계장 임시 디렉터리>/mcp.json` 을 계장이 쓰고 어댑터가
`--strict-mcp-config --mcp-config=<경로>` 를 넘긴다. 등호 형태다 — `--mcp-config`
가 가변인자라 뒤따르는 인자를 삼킨다.

내용은 셋의 합집합이고 **셋 다 `agent.mcp` 필터를 탄다**. 이름이 겹치면
**팩 · 노드 · 워크스페이스** 순으로 앞이 이기고 로그에 남긴다.

```text
   팩의 서버 중 요청된 것    팩 tar 의 mcp.json 중 agent.mcp 가 이름을 적은 것
   노드 선언 중 요청된 것    enode.yaml mcp: 중 agent.mcp 가 이름을 적은 것
   워크스페이스 중 요청된 것  .mcp.json 중 agent.mcp 가 이름을 적은 것
```

**팩만 「적힌 것 전부」였던 것을 고쳤다** (`decisions.md` 6절 ⑥ · 2026-09-12).
그대로 두면 계약 작성자가 `agent.mcp` 에 이름을 안 적고도 임의의 stdio 서버를
물릴 수 있고 `--strict-mcp-config` 도 가짜 홈도 그것을 안 막는다 — 허용목록
자체가 싣기 때문이다. 아래 4.2 의 SECURITY-06 과 4.3 의 「요청 안 한 서버 0」과
정면으로 어긋났다. **팩은 정의의 출처이지 허가의 출처가 아니다.**

```text
   확정   요청이 없으면 빈 목록이다.  팩이 서버를 실었어도 그렇다.  0 은 의도다
   확정   요청한 이름이 셋 어디에도 없으면 하네스를 안 띄우고 단계를 실패로 보고한다.
          사유는 mcp server <이름> is not available on this node
   확정   노드가 선언한 이름을 팩이 덮으면 거절한다 (decisions.md 6절 ⑦).
          사유는 pack redefines node-declared mcp server <이름>
   확정   파일에는 이름 · 종류 · 실행 경로나 주소 · 환경변수 이름만 적는다.  값은 없다
```

**수용 기준**: CA4.

## FR-3 (팩 3.3) 노드 선언과 광고

`enode.yaml` 에 `mcp:` 절을 더하고, 뜨는 것만 `mcp.<이름>: "1"` 로 광고한다.
매처는 안 고친다 — 속성 키 하나가 늘 뿐이다.

```text
   확정   stdio 는 command 가 resolve 되는가 · remote 는 credential 이름의
          환경변수가 노드 환경에 있는가.  실제 연결은 안 한다
   확정   비싼 탐지 자리(Detector · 기본 5분)에 앉는다
   확정   안 뜨면 광고에서 빠지고 빠진 이유는 노드 로그에 남긴다
   확정   detect.go 를 Fingerprinter 순회로 바꾼다 (decisions.md 6절 ⑤).
          하네스 순회의 break 가 그 부수 효과로 걷히고 harness.<이름>: "1" 을 싣는다.
          옛 harness 키는 한동안 같이 싣는다 — 걷는 시점은 이 회차가 안 정한다
   확정   노드 설정에 안 적힌 MCP 는 광고되지 않는다
   확정   광고에 자격증명의 이름조차 안 싣는다.  mcp.<이름> 만 실린다
```

**수용 기준**: CA2 · CA3.

## FR-4 (팩 3.4) 워크스페이스 감지

실행 시점에 워크스페이스 `.mcp.json` 을 읽어 **계약이 요청한 이름만** 허용목록으로
옮겨 적는다. 광고에는 안 싣는다 — 같은 rev 의 노드마다 똑같아서 후보를 못 가른다.

```text
   확정   project 설정 소스는 켜지 않는다.  켜면 저장소의 .claude/settings.json 의
          훅 · 권한까지 딸려온다 (실측 F2)
   확정   노드마다 달라지는 항목은 노드 소유자가 enode.yaml 의 mcp: 로 옮겨 적는다.
          그것이 그 항목을 광고하는 유일한 길이다
   실측   워크스페이스 .claude/skills/ 를 하네스가 스스로 읽는가.  CA5 가 잰다.
          안 읽으면 팩으로 나른다
```

**수용 기준**: CA4 의 둘째 줄 · CA5.

## FR-5 (팩 3.5) 계약 문법

매칭은 `requires` 에, 실행은 `agent` 에 적는다 (`ADR-013` 결정 4).

```text
   확정   requires[].mcp.<이름>: "1"   이미 있는 문법.  매처 무변경
   확정   agent.mcp: ["gerrit"]        이 단계가 물릴 서버 이름
   확정   agent.pack: "<blob 이름>" + in.from: ["<단계>.<blob 이름>"]
   교정   알려진 키 검증은 이미 있다 (2.1).  agentKeys 에 mcp · pack 을 더한다.
          400 은 이미 난다.  max_tokens 는 그대로 허용된다 (개명은 이 팩 밖)
   확정   runctl example 에 사내 MCP 하나를 요구하고 요청하는 예시를 더한다.
          runctl schema steps 는 구조체에서 뽑으므로 저절로 는다
   확정   계획(expands)이 짓는 단계도 같은 문법.  Grammar 에 agent.mcp · agent.pack
          이 들어간다.  계획이 안 적으면 그 단계는 팩 없이 돈다
```

**수용 기준**: CA3 · CA4.

## FR-6 (팩 3.6) 팩 — 전달과 출처

팩은 tar 하나다. `skills/<이름>/SKILL.md` · `agents/<이름>.md` · `mcp.json`.

```text
   확정   계장이 $IN 의 팩을 가짜 홈에 편다.  skills/ agents/ 는 그대로,
          mcp.json 은 허용목록에 합친다
   확정   팩의 settings.json 은 이 회차에 안 읽는다 (훅 설정과 충돌한다)
   확정   그 밖의 파일은 무시하고 로그에 남긴다
   확정   출처는 팩 단계다.  계약의 첫 단계가 명령 단계로 $OUT/pack 에 tar 하나를
          낸다 (git archive --remote · curl -o 등 argv 하나).  새 전송 0
   확정   runctl submit --pack 은 이월.  매칭 전 blob 쓰기의 API 자리가 없다
   확정   cmd/iapadapter 설정에 executor.pack: { fetch: [argv], name: pack }.
          비면 오늘 그대로.  템플릿이 팩 단계와 agent.pack 을 더한다
   추가   tar 풀기가 경로를 검증한다 (4.2 SEC-A).  팩이 안 적은 자리다
```

**수용 기준**: CA5.

## FR-7 (팩 3.7) 기록

`HarnessResult` 에 허용목록의 서버 이름(`mcp: [이름]`)과 팩의
다이제스트(`pack: <sha256>`)를 남긴다. 봉인된 묶음만 보고 그 단계가 어느
서버와 어느 팩으로 돌았는지 안다 (`ADR-005` 성질 4).

하네스 자신의 증언(`system/init` 줄의 `mcp_servers` · `slash_commands`)도
**이 팩이 나오게 한다** — `decisions.md` 6절 ⑮ 가 `Argv` 에
`--output-format stream-json --verbose` 를 더한다. 스트림 처리와 tee 와 사건
배출은 짝 팩의 몫이다.

**수용 기준**: CA6.

---

# 4. 비기능 요구사항

## 4.1 차단 게이트 다섯 — 그대로 걸린다

`constraints.md` 가 정본이다. 이 회차가 값을 바꾸지 않는다.

```text
   crypto/tls T 심볼 (enodectl.exe)      상한 10
   net/http  T 심볼 (enodectl.exe)       상한 50
   패키지별 커버리지                      하한 80%     internal/enode 가 걸린다
   허용목록 밖의 스킵                     상한 0
   U+2605 을 담은 파일 수                 상한 0
```

`internal/enode` 의 새 코드는 **하네스 실행파일 없이 도는 테스트**로 채운다 —
가짜 홈 · 허용목록 · 팩 펴기 · 뜨나 는 전부 파일과 문자열이다.

## 4.2 security-baseline — 규칙마다의 처리

확장이 켜져 있다 (`decisions.md` §1). 규칙은 차단성이다. `decisions.md` §3 이
다섯을 미리 적었고, 이 회차가 **둘을 더한다**.

| 규칙 | 적용 | 처리 |
|---|---|---|
| SECURITY-01 전송 암호화 | 기존 코드 사실 | 기록만. 수정 안 함 |
| SECURITY-02 중간자 접근 로그 | N/A | 이 팩이 네트워크 중간자를 안 만든다 |
| SECURITY-03 애플리케이션 로깅 | 새 표면 | 새 로그는 이름만 찍는다 — 빠진 MCP 의 이름 · 겹친 이름 · 팩에서 무시한 파일. 값은 안 찍는다. **`logs/` 도 이 규칙의 대상이다** — `runner.go` 가 stdout 을 원문으로 싣고 그 묶음은 `ADR-005` 성질 2 로 봉인된다. ⑮ 이 그 표면을 넓혔고 **⑱ 이 닫는다** — `logs/` 에 `system/init` 줄과 최종 `result` 봉투만 싣는다 |
| SECURITY-04 HTTP 보안 헤더 | N/A | 새 웹 표면 0 |
| SECURITY-05 입력 검증 | 새 표면 | `enode.yaml` 의 `mcp:` · 계약의 `agent.mcp` · `agent.pack` · 팩 tar 의 경로. 아래 SEC-A |
| SECURITY-06 최소 권한 | 새 표면 | 허용목록이 요청한 이름만 연다. 와일드카드가 없다 |
| SECURITY-07 네트워크 | 기존 코드 사실 | 기록만. 새 포트 0 · 새 전송 0 |
| SECURITY-08 접근 제어 | 새 표면 | 허용목록이 곧 접근 제어다 — 요청 안 한 서버 0 · 계정 커넥터 0. 노드 선언은 소유자만 쓰는 설정 파일에 산다 (ADR-063) |
| SECURITY-09 하드닝 | 새 표면 | 가짜 홈이 기본값이다. 여는 값을 두지 않는다 |
| SECURITY-10 공급망 | 새 표면 | 아래 SEC-B |
| SECURITY-11 보안 설계 | 새 표면 | 겹이 둘이다 — 가짜 홈과 strict 허용목록이 계정 커넥터를 각각 끊는다. 오용 시나리오는 4.3 |
| SECURITY-12 인증 · 자격증명 | 새 표면 | 자격증명 복사본은 `0600` · 단계 끝 삭제 · 임시 디렉터리 밖으로 안 나간다. 허용목록과 광고에는 이름만. R1 화이트리스트는 그대로 — `ENODE_TOKEN` 은 안 넘어간다 |
| SECURITY-13 무결성 | 새 표면 | 아래 SEC-B |
| SECURITY-14 알림 | N/A | 이 회차가 알림 경로를 안 만든다 |
| SECURITY-15 예외 처리 | 새 표면 | 실패는 닫히는 쪽으로 — 계장 실패 시 홈은 비어 있고, 요청한 서버가 없으면 하네스를 안 띄운다 |

### SEC-A 팩 tar 를 푸는 것은 신뢰할 수 없는 입력을 푸는 것이다

팩은 계약이 적은 주소에서 받은 tar 이고, 계장이 그것을 **가짜 홈 안에 편다.**
팩이 규약한 것은 「그 밖의 파일은 무시하고 로그에 남긴다」뿐이라 **경로**를 안
적었다. 다음을 요구로 세운다.

```text
   절대경로 항목을 거부한다            /etc/... 가 홈 밖으로 나간다
   .. 를 담은 항목을 거부한다          ../../.claude/settings.json 이 훅 설정을 덮는다
   심볼릭 링크 항목을 거부한다          링크를 따라 쓰면 홈 밖에 닿는다
   크기와 개수에 상한을 둔다            푸는 쪽이 디스크를 채우는 길을 막는다
   settings.json 은 이름으로 건너뛴다   팩이 정책을 덮는 것을 막는다 (팩 3.6 이 이미 적었다)
```

거부는 조용하지 않다 — 그 단계를 실패로 보고한다. `ADR-035` §3 의 「없음이
실패보다 나쁘다」가 여기도 같다.

### SEC-B 팩의 무결성은 기록이지 검증이 아니다

FR-7 이 `pack: <sha256>` 을 Record 에 남긴다. 그것은 **사후 감사**다.
SECURITY-13 이 요구하는 것은 **실행 전 검증**이다. 오늘 값이 없다.

이 회차의 판단 — **기록으로 족하다고 명시한다.** 근거 셋.

```text
   blob 이 봉인된다        $IN 은 0444 · 0555 로 잠긴다 (collect.go).  받은 뒤에는 안 바뀐다
   출처가 계약에 박힌다     tar 주소와 rev 가 argv 하나에 있고 manifest 에 남는다
   경계가 노드다           INVARIANTS — 명령 단계에는 파일시스템 경계가 없다.
                          같은 계약이 이미 임의의 argv 를 돌린다.  팩만 더 잠그는 것은
                          위협 모델을 안 좁히고 능력만 좁힌다 (claude.go 의 3차 판단과 같다)
```

**받는 쪽이 기대 다이제스트를 적는 자리(`agent.pack_sha256` 같은 것)는 이월**로
둔다. 값이 생기는 때는 팩 저장소가 여럿이 되거나 팩을 캐시할 때다.
`decisions.md` 에 행으로 적는다 (7절).

## 4.3 오용 시나리오 (SECURITY-11)

```text
   남의 자격으로 도는 MCP    계약이 노드의 mcp: 에 없는 이름을 요청한다
                            -> 셋 어디에도 없으면 단계 실패.  조용히 안 빠진다 (FR-2)
   정책을 덮는 팩            팩이 settings.json 이나 ../ 경로를 담는다
                            -> 이름으로 건너뛰고 경로로 거부한다 (SEC-A)
   노드를 나가는 도구        계정 커넥터가 붙어 하네스가 밖으로 나간다
                            -> 가짜 홈과 strict 가 두 겹으로 막는다.  언제나 0
   자격증명이 광고에 샌다     mcp: 의 credential 이름이 함대 전체에 보인다
                            -> 광고에 mcp.<이름> 만 싣는다.  이름도 값도 안 싣는다
   소유자의 이름을 팩이 덮는다  계약이 requires 로 소유자 선언을 보고 노드를 고른 뒤
                            자기 팩의 정의로 그 이름을 덮어 실행한다.
                            매칭은 소유자의 값으로 하고 실행은 계약의 값으로 한다
                            -> 덮으면 거절한다 (decisions.md 6절 ⑦).
                            팩이 새 이름을 싣는 것은 그대로 허용한다
   팩이 요청 없이 서버를 싣는다  팩 mcp.json 의 서버가 agent.mcp 를 안 거치고 실린다
                            -> 팩도 필터를 탄다 (6절 ⑥).  요청 안 한 이름은 0 이다
```

## 4.4 성능

새 비용은 탐지 하나다. MCP 「뜨나」는 비싼 탐지 자리(5분)에 앉는다 — 실행파일
resolve 가 프로세스를 안 띄우지만 파일시스템을 훑는다. **광고 루프는 이것을
직접 부르지 않는다.** 부르면 여기서 멈출 때 하트비트가 함께 멈추고 노드가
프로세스도 로그도 정상인 채로 함대에서 사라진다.

---

# 5. 수용 기준 — 장면 조각 게이트

`scene-gates.md` 2절이 정본이다. 조각마다 실행 명령이 3절에 있다.

| | 조각 | 재는 기능 | 대상 | 집행자 |
|---|---|---|---|---|
| CA0 | 기동이 안 깨졌다 | 바닥 | 스크래치 | 기계 |
| CA1 | 끊긴다 | FR-1 · FR-2 | 스크래치 | 사람 |
| CA2 | 광고한다 | FR-3 | 스크래치 | 사람 |
| CA3 | 고른다 | FR-3 · FR-5 | 함대 | 사람 |
| CA4 | 연다 | FR-2 · FR-4 · FR-5 | 스크래치 | 사람 |
| CA5 | 싣는다 | FR-6 | 스크래치 | 사람 |
| CA6 | 한 장면 | 전부 | 사내 함대 | 사람 |

**CA1 이 가장 앞이다.** 실측이 찾은 것이 「거꾸로 막힌다」였으므로 무엇을 열기
전에 무엇이 끊기는지를 먼저 잰다.

**눈 검증을 보류로 넘기지 않는다.** CA1 · CA4 · CA5 는 사람이 실제 하네스로 한
번은 띄워야 초록이다. 앞 팩의 CP6 이 눈 검증을 보류로 남긴 채 닫혔고 그 결함이
사내 실측에서야 드러났다.

## 5.1 이 회차의 게이트 읽는 법 — 2026-09-12 에 뒤집혔다

답이 A 였다. 이 팩을 먼저 끝내고 transcript 를 뒤에 둔다. 그때는 `init` 줄을
사람이 직접 띄워 읽기로 했다.

**그 판정이 뒤집혔다** (`decisions.md` 6절 ⑮). U1 이 `Argv` 에
`--output-format stream-json --verbose` 를 더하므로 그 줄이 `logs/` 에 남고
게이트가 **실물**을 잰다. `scene-gates.md` 3절이 사람이 손으로 띄우는 복제본
경로를 **명시로 배제한다** — 환경(`harnessEnv` 가 `os.Environ()` 을 안 얹는다) ·
플래그(`--settings` · `--permission-mode` · `--add-dir`) · 게이트웨이 인증
(`apiKeyHelper`)에서 실물과 갈리기 때문이다. 아래 명령줄은 뒤집기 전의 기록이다.

```text
   CLAUDE_CONFIG_DIR=<가짜 홈> claude -p --output-format stream-json --verbose \
     --setting-sources "" --strict-mcp-config --mcp-config=<허용목록> "say hi" < /dev/null \
     | head -1 | jq '{mcp: .mcp_servers, skills: .slash_commands}'
```

`claude mcp list` 는 안 쓴다 — strict 와 `--mcp-config` 를 반영하지 않는다
(실측 C · G).

**다만 `transcript.go` 가 이미 있다** (2.3). `Job.Transcript` 가 링 파일로 tee
하므로, 노드 설정이 그것을 켜면 `logs/` 없이도 하네스 원문을 읽을 수 있다.
Application Design 이 이 자리를 본다 — 게이트가 더 싸질 수 있다.

---

# 6. 범위

## 6.1 이 회차가 하는 것 — 일곱

FR-1 ~ FR-7. `features.md` 4절의 필수 일곱과 같고, FR-5 의 크기가 2.1 만큼
줄었다.

## 6.2 안 하는 것

`constraints.md` 의 제외 여덟 범주가 정본이다. 요약하면 이렇다.

```text
   계정 커넥터를 여는 경로 · 도구 이름 차단      ADR-067 §2 · §3
   자격증명 주입 (R2)                          Credentials 는 Transparent 그대로
   부칠 수 있는 것의 광고                       ADR-035 §4.1
   MCP 의 내용을 아는 것 (tools/list 속성 매핑)  ADR-012
   팩의 운반 경로 추가 (submit --pack · 캐시)    ADR-034 §2.2
   두 번째 하네스 (Codex · Gemini)              agent-runtime R6
   중앙 설정 (Mediator 의 하네스 · MCP 항목)     ADR-012
   기존 계약 변경 (success_when · 봉인 · 임대 · 매처 · max_tokens 개명)
```

**Mediator 라우트는 0 개 는다.** `internal/api/api.go` 는 등록 줄조차 안 는다.
`internal/store` · `internal/panel` · `internal/api/ui` 는 안 건드린다.

## 6.3 새 코드가 사는 자리

```text
   internal/enode       가짜 홈 · 허용목록 · 팩 펴기 · 탐지기 · 노드 선언.  실행 층이다
   internal/contract    agent 의 알려진 키 · 예시.  Mediator 와 enode 가 함께 본다
   cmd/iapadapter       설정 키 하나 · 템플릿의 팩 단계
   cmd/runctl           소스 diff 0.  example 이 임베드 FS 를 돌고 예시 파일은
                        internal/contract 에 산다.  schema 는 Step.Agent 가 map 이라 안 는다
```

임포트 금지 넷은 그대로다. 이 팩이 새 패키지를 만들지 않으므로 표에 줄이 안 는다.

**프로세스를 띄우는 코드는 `runner.go` 하나다.** 어댑터에 exec 을 두지 않는다 —
R1 화이트리스트를 N 번 지키게 되는 것이 구멍이다.

---

# 7. `decisions.md` 에 더할 행

팩의 게이트 규칙은 「권장을 벗어나면 근거를 적는다」다. 이 회차가 벗어나거나
교정한 자리는 **열넷**이고(이 단계가 셋 · Application Design 이 둘 · 2026-09-12 의
설계 검증과 설계 질문이 아홉), 승인 뒤 진행자가 `requirements/harness-components/decisions.md`
에 2026-09-11 날짜 절로 싣는다.

```text
   ①  알려진 키 검증은 이미 있다               2.1.  3.5 의 범위가 목록 추가로 준다
   ②  팩 tar 풀기의 경로 검증                  SEC-A.  절대경로 · .. · 심볼릭 링크 거부 ·
                                              크기 상한.  거부는 단계 실패
   ③  팩의 기대 다이제스트는 이월              SEC-B.  기록으로 족한 근거 셋을 적는다
```

---

# 8. Application Design 이 닫을 것

**요구가 아니라 모양**이다. 여기서 정하면 팩 밖의 결정이 된다.

```text
   D1  Fixed() 가 계장 디렉터리를 어떻게 아나    인터페이스를 바꾸나 runHarness 가 합치나 (2.2)
   D2  Detector 가 MCP 를 어떻게 드나           ADR-035 §4.2 의 탐지기 종류.  오늘 Detector 는
                                               고루틴 하나다 (detector.go)
   D3  허용목록을 누가 쓰나                     Instrument 가 쓰나 runHarness 가 쓰나.
                                               팩 펴기와 같은 자리인가
   D4  게이트가 transcript 링을 쓸 수 있나       2.3 · 5.1.  쓸 수 있으면 CA1 · CA4 · CA5 가 싸진다
```

---

# 9. 회차 운영 — 질문 2 의 답이 정했다

답이 B 였다. **Inception 을 돌고 그대로 Construction 까지 한 손(taeels)으로
간다.** 유닛을 직렬로 민다.

```text
   담당          taeels 하나.  construction-roster.md 에 행을 더하지 않는다
   병렬          없다.  이 팩이 만지는 자리가 한 손 안이라 충돌이 0 이다.
                 짝 팩과의 접점은 둘이다 — runner.go 와 claude.go 의 Argv (⑮)
   문서 루트      Inception 은 aidlc-docs/v3-run-harness-components/
                 Construction 은 aidlc-docs/taeels/ (CLAUDE.md)
   브랜치         Inception 은 v3-run-harness-components 에서 직렬.
                 유닛은 unit/<유닛> 을 따고 그 유닛의 조각 게이트가 초록인 뒤 병합
   짝 팩          transcript 는 이 팩 뒤다 (질문 3 = A)
```

**Units Generation 에 거는 요구 하나 — 파일 행렬을 반드시 낸다.** 유닛마다
만지는 파일을 표로 세고, 한 파일을 둘 이상이 만지면 그 자리를 접점으로 적고
처리를 정한다. 한 손이라도 유닛 순서가 그 표에서 나온다.

---

# 10. 정본에 되돌려 올리는 것

`decisions.md` 5절이 정본이다. 회차가 끝나면 진행자가 `enode-design` 에 올린다.
서브모듈은 진행자만 올린다.

```text
   ADR-067    초안 -> 결정.  §5 미결 둘의 답
   ADR-034    §5 ①② 착수 기록 · §3 ① OAuth 파일 복사 값 · 팩 tar 규약과 팩 단계.
              ④(알려진 키)는 「이미 있다」로 고친다 (2.1)
   ADR-035    §7 「지금 열지 않는다」 -> 열렸다.  여는 조건 충족 일자와 근거
   R6         「스킬 · MCP 는 자리만」 -> 구현됨.  플래그 목록 갱신
   실측 표     features.md 1.2 의 네 줄을 ADR-067 에 함께 올린다
```
