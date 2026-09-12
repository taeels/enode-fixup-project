# 예외사항 — 이 팩이 구현하지 않는 것

요구 문서가 말하지 않은 것을 「빠뜨린 것」이 아니라 「뺀 것」으로 읽게 한다.
끝의 구조 불변식과 접점 절은 유닛과 무관하게 참인 것이다.

## 제외 기능 (구현하지 않음)

### 1. 계정 커넥터를 여는 경로
- 사람의 claude.ai 로그인이 주는 MCP 를 하네스에 물리는 값
- 노드를 나가는 도구(`SendMessage` · `ListAgents` · `RemoteTrigger`)의 이름 차단

근거 — 앞은 `ADR-067` §2 가 「노드를 나간다」로 막았고, 뒤는 같은 문서 §3 이
「셸이 열려 있으면 이름으로 막는 것은 연극」으로 순연했다. 이 팩은 앞을 값으로
닫고 뒤를 건드리지 않는다.

### 2. 자격증명 주입
- 요청자 신원이나 팀 공용 신원으로 도는 것 (R2)
- 자격증명 값을 계약 · 허용목록 · 광고에 싣는 것

근거 — `Credentials` 인터페이스가 자리를 들고 있고 `Transparent` 가 결정이다.
「그 기계가 자격을 가졌나」(매칭)와 「누구의 자격으로 도나」(주입)는 다르다
(`ADR-035` §8). 이 팩은 앞만 한다.

### 3. 부칠 수 있는 것의 광고
- 스킬 · 서브에이전트 · npm MCP 정의 · 워크스페이스 `.mcp.json` 항목을 광고 키로 싣는 것
- 워크스페이스 안의 것을 매칭 조건으로 쓰는 것

근거 — `ADR-035` §4.1. 노드마다 다르면 재현성이 샌다. 워크스페이스 안의 것은
같은 rev 의 노드마다 똑같아서 후보를 가르지 못한다.

### 4. MCP 의 내용을 아는 것
- `tools/list` 를 물어 도구 설명을 속성으로 사상하는 것
- 광고에 MCP 의 버전 · 도구 수 · 도구 이름을 싣는 것

근거 — `ADR-012` 「`tools/list` 는 존재 확인에만 쓴다」. 매처가 동등 비교뿐이라
버전은 매칭에 못 쓴다.

### 5. 팩의 운반 경로 추가
- `runctl submit --pack` (제출 시점 blob)
- 계약 인라인 팩 · 노드에 미리 둔 팩 · 팩 캐시
- 새 전송 · 새 라우트

근거 — `ADR-034` §2.2 가 인라인과 노드 사전 배치를 기각했다. 제출 시점 blob 은
값이 없어 이월이다 (`decisions.md` 4절). 팩 단계 하나로 오늘 도는데 경로를
늘릴 이유가 없다.

### 6. 두 번째 하네스
- Codex · Gemini 어댑터. Gemini 의 `trustedFolders.json`
- 하네스별 팩 펴기 형식의 일반화

근거 — `agent-runtime` R6 「두 번째 하네스가 생기기 전에는 인터페이스의 모양을
정할 근거가 없다」. `Instrument` 안에 가둔다는 결정으로 족하다.

### 7. 중앙 설정
- Mediator 에 하네스 · MCP · 팩 항목을 두는 것
- 중앙에서 노드의 `mcp:` 를 적거나 승인하는 라우트

근거 — `ADR-012` 「그 기계에서만 아는 것은 그 기계에만」. Mediator 의 하네스 · MCP
항목은 여전히 0 줄이다 (`ADR-035` §6).

### 8. 기존 계약 변경
- `success_when` 문법 · Record 봉인 규칙 · 임대 키 · 매처(`Satisfies` · `Match` · 정렬)
- `max_tokens` 개명 (`ADR-034` §4) — 이 팩과 무관하다

---

## 보안 확장이 하는 것과 안 하는 것

`security-baseline` 을 켠다. 규칙은 새로 만드는 표면(가짜 홈 · 허용목록 · 노드
선언 · 계약 키)에만 건다. 기존 코드의 사실(평문 HTTP · 토큰 로그 · `:8080`)은
기록만 하고 고치지 않는다 — `decisions.md` 3절.

---

## 구조 불변식 — 유닛을 어떻게 가르든 참이다

**여기 있는 것은 제약이지 계획이 아니다.** 어느 유닛이 무엇을 맡는지는 Units
Generation 이 정한다.

### 새 코드가 사는 자리

```text
   internal/enode       가짜 홈 · 허용목록 · 팩 펴기 · 탐지기 · 노드 선언.  실행 층이다
   internal/contract    agent 의 알려진 키 · 예시.  Mediator 와 enode 가 함께 본다
   cmd/iapadapter       설정 키 하나 · 템플릿의 팩 단계
   cmd/runctl           소스 diff 0.  예시는 internal/contract 의 임베드 FS 에 산다.
                        schema 는 Step.Agent 가 map 이라 안 는다
```

**Mediator 라우트는 0 개 는다.** `internal/api/api.go` 는 등록 줄조차 안 는다.
`internal/store` · `internal/panel` · `internal/api/ui` 는 안 건드린다.

### 임포트 금지 — 앞 팩의 넷 그대로

```text
   internal/panel  ->  internal/store     금지
   internal/panel  ->  internal/api       금지
   internal/api/ui ->  internal/store     금지
   internal/enode  ->  internal/panel     금지
```

이 팩은 새 패키지를 만들지 않으므로 표에 줄이 늘지 않는다. 경계 검사 테스트는
그대로 돈다.

### 실행은 여전히 한 자리에서

**하네스를 띄우는 코드는 `runHarness` 하나다.** 「프로세스를 띄우는 코드가
`runner.go` 하나」로 읽으면 거짓이다 — `internal/enode` 만 봐도 여덟 자리가
프로세스를 띄운다(`claim.go` 의 명령 단계 · `claude.go` 의 `--version` 과
`auth status` · `repoid` · `identity` · `diff` · `workspace`). 유닛마다 도는
검사로 굳힐 때 좁은 형태를 쓴다. 허용목록 파일과 가짜 홈은
`Instrument` 가 짓고 `Fixed` 가 값을 박는다. 어댑터에 exec 을 두지 않는다 —
R1 화이트리스트를 N 번 지키게 되는 것이 구멍이다.

### 차단 게이트 다섯 — 그대로 걸린다

```text
   crypto/tls T 심볼 (enodectl.exe)      상한 10
   net/http  T 심볼 (enodectl.exe)       상한 50
   패키지별 커버리지                      하한 80%      internal/enode 가 걸린다
   허용목록 밖의 스킵                     상한 0
   U+2605 을 담은 파일 수                 상한 0
```

`internal/enode` 의 새 코드는 하네스 실행파일 없이 도는 테스트로 채운다 —
가짜 홈 · 허용목록 · 팩 펴기 · 뜨나 는 전부 파일과 문자열이다.

---

## 접점 — 다른 손과 부딪히는 자리

**팩 둘이 같은 파일을 만진다.** 진행자가 직렬로 병합한다 (`CONVENTIONS.md` 3.1).

```text
   internal/enode/claude.go   Argv       이 팩은 MCP 플래그와 출력 형식 플래그를
                                         더한다 (decisions.md 6절 ⑮ — 게이트가 재는
                                         system/init 줄이 그래야 나온다).
                                         transcript 팩이 그 위에서 스트림 처리를
                                         자라게 한다.  겹친다
                              Fixed      이 팩만
                              Instrument 이 팩만
                              Decode     transcript 팩만
   internal/enode/runner.go   Job        이 팩은 팩 · MCP 를 Job 에 싣는다.
                                         transcript 팩은 tee 와 사건 배출을 만진다
                              logs/ 조립  이 팩이 도구 사건의 내용을 걷는다 (6절 ⑱).
                                         exec 뒤의 자리다 — 짝 팩의 스트림 처리와 겹친다
   internal/enode/hook.go     WriteHookSettings 를 가짜 홈 안으로 옮긴다.  이 팩만
```

**Units Generation 에 거는 요구 하나 — 파일 행렬을 반드시 낸다.** 유닛마다 만지는
파일을 표로 세고, 한 파일을 둘 이상이 만지면 그 자리를 접점으로 적고 처리를 정한다.
