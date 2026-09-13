# U3 `advert` — 값이 지나가는 길

형식은 `domain-entities.md`, 규칙은 `business-rules.md`. 여기는 **그 규칙들이
어느 순서로 걸리고 게이트가 어디서 초록이 되나**다.

---

## 1. 한 줄기

```text
   enode.yaml 의 mcp:
     |
     v
   LoadLocal            MCPServer.UnmarshalYAML 이 두 번 푼다 (형식 2.1)
     |                  거절 다섯 + env 하나가 여기서 걸린다 (규칙 1 · 2)
     |                  어기면 cmd/enode/main.go:112 가 노드를 안 띄운다
     v
   Local.MCP            그 기계에만 있는 선언.  중앙에 안 적는다 (ADR-012)
     |
     v
   mcpFP.Probe          뜨나 판정.  프로세스를 안 띄우고 연결도 안 한다 (규칙 3)
     |                  안 뜨는 것마다 Warn 한 줄 (규칙 5)
     v
   costlyAttrs          셋을 순회해 합친다 — harnessFP · repoFP · mcpFP (규칙 4)
     |
     v
   Detector             5분마다 이것만 다시 돈다.  cheapAttrs 는 광고마다 새로 본다
     |
     v
   capabilities         hasCapability 가 문턱을 잰다 (규칙 6).
     |                  넘으면 agent.reason 하나에 attrs 를 전부 싣는다
     v
   POST /v1/nodes       매번 전부.  델타가 아니다 (ADR-017 결정 3)
     |
     +---> 매처          Capability.Satisfies 의 완전 일치.  없는 키는 422
     |
     +---> GET /v1/capabilities   store.Capabilities 가 키마다 합집합
     |       |
     |       +---> runctl capabilities      계약 작성자가 보는 면 (⑬)
     |
     +---> GET /v1/nodes          운영자 면
             |
             +---> 현황판          format.mjs 가 이름을 준다 (규칙 7)
             +---> 제어판          attrs 를 원문 칩으로.  코드 0
```

**이 유닛이 만지는 제품 파일은 셋이다** — `config.go` · `mcp.go` · `detect.go`.
답 7=B 가 넷째를 더한다 — `internal/api/ui/static/shared/fleet/format.mjs`.

---

## 2. 시계 둘이 가르는 자리

```text
   광고 주기    Mediator 의 만료 계산에 맞는다 (ADR-028).  cheapAttrs 가 여기 탄다
   탐지 주기    알아내려는 사실이 얼마나 빨리 변하느냐에 맞는다.  기본 5분.
               costlyAttrs 가 여기 탄다 — mcp 는 이쪽이다
```

**`mcp` 가 비싼 쪽인 근거는 프로세스가 아니라 호출 수다.** `mcpUp` 은 프로세스를
안 띄우고 `cheapAttrs` 도 `detectArch` 로 `LookPath` 를 두 번 돈다. 가르는 것은
**선언 수만큼 파일시스템을 훑는다**는 것이다 — 노드 설정에 비례해 는다
(`components.md` 2.2 · `requirements.md` 4.4).

**CA2 의 「실행파일을 치우면 탐지 주기 뒤 빠진다」가 이 갈래로 저절로 선다.**
`Detector.Run` 이 티커마다 `refresh` 를 부르고 `refresh` 가 `costlyAttrs` 를 다시
돈다 — **새 배선이 0 이다.** 이 유닛은 `Detector` 를 한 글자도 안 고친다.

---

## 3. `costlyAttrs` 의 순회

```go
func costlyAttrs(ctx context.Context, l Local, log *slog.Logger) map[string]string {
	attrs := map[string]string{}
	for _, fp := range fingerprinters {
		part, err := fp.Probe(ctx, l, log)
		if err != nil {
			log.Warn("fingerprint failed; its attributes are missing from this advertisement",
				"kind", fp.Kind(), "err", err)
			continue // 규칙 R7 — 그 종류만 버리고 계속한다
		}
		for k, v := range part {
			attrs[k] = v // 규칙 R8 — 나중 것이 이긴다. 오늘 겹치는 키는 0
		}
	}
	return attrs
}
```

**오늘 도는 것과 같은 결과를 낸다.** 셋으로 가른 뒤에도 `harness` 의 `break` 는
`harnessFP` 안에 그대로 있고, `harnesses` 가 하나라 (`harness.go:282`)
그 `break` 가 오늘 도는 결과를 안 바꾼다. 는 것은 `harness.<이름>` 키 하나다.

```text
   harnessFP.Probe   harnesses 를 돌며 Usable() 이 참인 것마다
                     harness.<이름> = "1" 을 싣는다.  첫 것의 이름을
                     옛 harness 키로도 싣는다.  HarnessBin 덮어쓰기는 claude 만
                     (오늘 그대로).  못 쓰는 것의 Warn 도 오늘 그대로 (ADR-059)

   repoFP.Probe      Workspace 가 있으면 DetectRepo.  실패하면 WorkspaceID
                     fallback (ADR-036).  이 fallback 을 안 옮기면 git 없는
                     노드에서 repo 가 사라져 중립이 깨진다

   mcpFP.Probe       Local.MCP 를 돌며 mcpUp 이 참인 것마다 mcp.<이름> = "1".
                     거짓이면 안 싣고 사유를 Warn (규칙 R9)
```

---

## 4. `mcpFP.Probe` 안쪽

```text
   ① Local.MCP 가 비면 빈 맵을 낸다.  로그도 안 낸다 — 선언이 없는 것은 정상이다
   ② 이름을 정렬해 돈다              같은 입력이면 같은 로그 순서 (ADR-014 결정 3 의 결)
   ③ 서버마다 mcpUp                 nil 이면 mcp.<이름> = "1", 아니면 Warn 한 줄
   ④ 오류를 안 낸다                 서버 하나가 안 뜨는 것은 이 종류의 실패가 아니다.
                                    R7 이 걸리는 자리가 아니다
```

**③ 이 「안 뜨는 것」과 「종류가 실패한 것」을 가른다.** 안 뜨는 서버는 정상적인
결과다 — `ADR-017` 결정 3 의 「못 하면 뺀다」가 그것이다. 종류의 실패는 물어보지
못한 것이고, 오늘 `mcpFP` 에는 그런 경로가 없다.

---

## 5. CA2 가 어디서 초록이 되나

```text
   ① enode.yaml 에 mcp: { probe: { command: true } } 를 적고 노드를 띄운다
      -> LoadLocal 의 거절 다섯을 다 통과한다 (command 하나면 stdio 다)

   ② 첫 탐지가 NewDetector 안에서 동기로 돈다 — 첫 광고가 빈 능력으로 안 나간다
      -> mcpUp 이 PATH 에서 true 를 찾는다 -> mcp.probe = "1"

   ③ GET /v1/nodes 의 그 노드 attrs 에 셋이 함께 있다
      mcp.probe = "1" · harness.claude = "1" · harness = "claude"

   ④ command 를 없는 경로로 바꾸고 탐지 주기(5분)를 기다린다
      -> refresh -> costlyAttrs -> mcpFP.Probe -> mcpUp 이 거짓
      -> mcp.probe 가 빠지고 노드 로그에 R9 의 줄

   S1 눈 검증  현황판 노드 카드의 「제공 기능」에 MCP 서버 · probe = 있음 (규칙 R13 · R14)
              제어판 「탐지 능력」 카드에 mcp.probe=1 칩 (코드 0)
```

**집행자는 이 유닛을 구현하지 않은 사람이다** (`scene-gates.md` 2절 머리).
**S1 을 보류로 안 넘긴다** (같은 문서 4절).

---

## 6. CA3 의 뒤 절반이 닫히는 자리

U2 가 앞 절반(`400` · `lint` · 예시)을 닫았다. 이 유닛이 나머지를 닫는다.

```text
   runctl capabilities 에 mcp.probe 가 나온다
     store.Capabilities 가 attrs 를 키마다 합집합으로 모으고 키 이름을 안 가린다.
     코드 0 이다 — decisions.md 6절 ⑬ 의 「자동으로 나타난다」가 실측으로 참이다

   requires 에 "mcp.probe": "1" 을 적은 계약이 그 노드에만 간다
     Require 의 미지 키가 Attrs 로 들어가고 Capability.Satisfies 가
     got != want 로 자른다.  match 의 코드 diff 는 0 이다

   "mcp.nope": "1" -> 422
     Match 의 1차 순회가 countSatisfying 으로 0 을 세고 CodeNoCandidate 를 낸다.
     이것도 코드 0 이다 — 새 어휘가 있는 통로를 탈 뿐이다
```

**이 조각이 코드 0 으로 닫히는 것이 `ADR-012` 가 옳았다는 증거다** — 어휘가
창발하므로 매처도 저장소도 라우트도 새 이름을 알 필요가 없다.

---

## 7. 완료 조건과의 대조 — 어긋남 하나

`unit-of-work.md` 3절의 완료 조건 여덟을 이 설계와 맞췄다.

| 완료 조건 | 이 설계 |
|---|---|
| CA2 가 초록 | 5절 |
| CA2 의 눈 검증 S1 | 5절. 답 7=B 가 이름을 준다 |
| CA3 완결 | 6절 |
| `cheapAttrs` · `capabilities` · `Detector` 에 diff 0 | **어긋난다 — 아래** |
| 순회가 동작 중립 | `business-rules.md` 8절 |
| `runctl capabilities` 에 나온다 | 6절. 코드 0 |
| `enode.yaml` 의 `mcp:` 가 검증된다 | `business-rules.md` 1 · 2절 |
| CA0 초록 | Code Generation |

**어긋나는 것은 넷째다.** 답 6=B 가 `hasCapability` 를 고친다. 그 함수는 완료
조건이 든 셋에 안 들지만 `capabilities` 가 그것을 부르므로 **`capabilities` 의
글자는 그대로이고 동작이 바뀐다.** 조건의 글자를 지키면서 뜻을 벗어나는 것이라
기록으로 남긴다 — 사용자가 답 6 에서 그 대가를 보고 골랐다.

**같은 자리에 `components.md` 1.4 의 「`cheapAttrs` 는 안 건드린다」가 있는데
그것은 그대로 지켜진다** — 고치는 것은 `hasCapability` 하나다.

---

## 8. 이음매 — 이 어휘를 누가 어디서 집나

```text
   U4 sources   resolveComponents 가 Local.MCP 를 출처 셋 중 하나로 읽는다.
                이 유닛이 정한 형식(필드 여섯)과 거절 다섯을 그대로 쓴다.
                U4 는 그 위에 agent.mcp 필터와 이름 없음의 거절을 얹는다

   U5 pack      팩이 노드 선언 이름을 덮으면 거절한다 (decisions.md 6절 ⑦).
                그 「노드 선언」이 이 유닛의 Local.MCP 다 — 이름 공간의 주인이
                여기서 정해진다

   U1 isolation allowlistEntry 가 Extra 를 먼저 얹고 아는 키를 덮는다 (답 2=A).
                U1 의 이음매 주석이 지목한 그 자리이고, 파일 행렬 2.1 의
                「U3 은 U1 의 것을 안 고친다」가 이 답으로 거짓이 된다 — 9절

   짝 팩         접점 0.  이 유닛은 runner.go 도 claude.go 도 안 만진다
```

---

## 9. 이 유닛이 회차 밖으로 낼 것

```text
   파일 행렬        2.1 의 U3 칸을 고친다 — allowlistEntry 를 만진다.
                   internal/api/ui 행을 더한다 (답 7=B).  만지는 파일 16 -> 17

   decisions.md    6절에 실측 행 — yaml 이 종류만 잡는다 · 「그 밖의 키」가 오늘
                   안 생긴다 · env 의 갈림.  2절의 env 줄도 답 3=A 로 고친다

   unit-of-work.md 3절 완료 조건 넷째의 어긋남 (7절).  회차 진행자의 문서다

   component-methods.md  고칠 것 0.  답 4=A 가 Probe 의 시그니처를 그대로 둔다

   GLOSSARY.md     이 유닛이 새 글자를 안 들여온다 — CA2 · CA3 은 U2 가 이미
                   물었고 답 7=B 로 밖에 있다.  그 빚은 그대로 진행자의 것이다
```

**`Local` 의 필드가 열둘이 되는 것**과 **라벨로 선언 없이 광고할 수 있다는 것**
(`business-rules.md` 4절)은 이 유닛이 안 고치고 `code-summary` 가 넘긴다.
