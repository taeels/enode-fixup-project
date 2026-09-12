# 응용 설계 — 하네스 구성요소 (통합본)

네 문서를 한 장으로 모은 것이다. 상세는 각 문서가 진다.

```text
   components.md            무엇이 늘고 무엇이 확장되나 · 실패 등급 표
   component-methods.md     시그니처와 오류 계약
   services.md              순서와 실패 지점 · 탐지 · 팩의 운반
   component-dependency.md  호출 방향 · 임포트 금지 · 짝 팩과의 접점
```

---

## 1. 이 설계가 한 문장으로 말하는 것

**하네스의 사적인 세계를 계장 임시 디렉터리 아래에 새로 짓고, 그 세계에 무엇을
넣을지는 실행 직전에 한 번 결정한 뒤 그 결정대로만 쓴다.**

결정과 쓰기가 갈린 것이 이 설계의 뼈대다 (Q4 = A).

```text
   결정   resolveComponents     파일을 안 만진다.  틀리면 하네스가 안 뜬다
   쓰기   Instrument            정책을 다시 판단하지 않는다.  받은 것만 쓴다
```

---

## 2. 답 다섯이 정한 것

```text
   Q1 = A   Fixed(dir string) map[string]string
            하네스별 이름(CLAUDE_CONFIG_DIR)이 어댑터 밖으로 안 나간다

   Q2 = A   계장 디렉터리를 못 만들면 단계 실패
            오늘 동작이 바뀐다.  decisions.md 에 행으로 적는다

   Q3 = A   costlyAttrs 에 mcpAttrs 하나.  Detector 는 시계로 그대로
            decisions.md 2절의 권장값에서 벗어난다.  행으로 적는다

   Q4 = A   결정과 쓰기를 가른다.  정책 실패는 exec 전에 단계를 죽인다

   Q5 = A   출력 형식을 안 건드린다.  게이트는 사람 경로
            짝 팩과의 접점이 여섯에서 하나로 준다
```

---

## 3. 무엇이 바뀌나 — 한눈에

| 경로 | 무엇이 | 종류 |
|---|---|---|
| `internal/enode/mcp.go` | 새 파일 — `MCPServer` · `Components` · `Pack` · `resolveComponents` · `readPack` · `mcpUp` · `mcpAttrs` | 새로 |
| `internal/enode/harness.go` | `Fixed` · `Instrument` 시그니처 · `HarnessResult` 필드 둘 · `errComponents` | 확장 |
| `internal/enode/claude.go` | `Fixed(dir)` · `Instrument` 가 가짜 홈 · 팩 · 허용목록을 쓴다 | 확장 |
| `internal/enode/runner.go` | `Job.NodeMCP` · 순서와 실패 등급 | 확장 |
| `internal/enode/detect.go` | `mcpAttrs` 합침 · `break` 제거 · `harness.<이름>` | 확장 |
| `internal/enode/config.go` | `Local.MCP` · `SampleLocal` 한 줄 | 확장 |
| `internal/enode/hook.go` | 훅 파일이 가짜 홈 옆으로. `self` 가 비어도 돈다 | 확장 |
| `internal/contract/contract.go` | `agentKeys` 에 이름 둘 | 확장 |
| `internal/contract/grammar.go` | `agent.mcp` · `agent.pack` 을 가르치는 줄 | 확장 |
| `internal/contract/examples/mcp.json` | 예시 하나 | 새로 |
| `cmd/iapadapter/config.go` · `contract.go` | `executor.pack` · 템플릿 분기 | 확장 |
| `cmd/runctl` | **없다** — `example` 과 `schema` 가 저절로 는다 | 0 |
| `internal/api` · `store` · `panel` · `api/ui` | **없다** | 0 |

---

## 4. 설계가 답 밖에서 더 정한 것 둘

답 다섯이 안 닫은 자리다. **둘 다 조용히 열리는 길을 막는 것**이고, 그것이
이 회차가 사내 실측에서 고치려는 실패와 같은 종류다.

### 4.1 `Instrument` 의 오류를 등급으로 가른다

오늘 `Instrument` 의 오류는 삼켜진다 — 「계장은 보조이고 진짜 안전망은
워크스페이스 diff 다」가 그 근거였고, 그 판단은 훅에 대해서는 지금도 맞다.

그런데 이제 같은 메서드가 **정책의 산물**도 쓴다. 허용목록이 안 쓰이면 계약이
요청한 서버가 조용히 없고, 팩이 안 펴지면 스킬이 조용히 없다. 둘 다 `exit 0` 으로
성공이 봉인되는 길이다.

```text
   errComponents 로 감싼 오류   치명.  가짜 홈 · 허용목록 · 팩 · 자격증명 복사
   그 밖의 오류                 보조.  훅 · 기준 시각 — 오늘 그대로 삼킨다
```

`errors.Is` 하나로 갈린다. Q4 의 B(전부 치명)가 훅의 기존 판단을 말없이 뒤집는
것과, C(새 메서드)가 두 번째 하네스 없이 인터페이스를 늘리는 것을 둘 다 피한다.

### 4.2 `Instrument` 를 언제나 부른다

오늘은 `os.Executable()` 이 빈 문자열이면 계장을 **통째로** 건너뛴다. 그 경로로
가면 허용목록이 안 쓰이고 팩이 안 펴진다 — 4.1 이 막으려는 바로 그 침묵이
`self` 하나로 다시 열린다.

`self` 가 비면 훅 블록만 빼고 나머지(게이트웨이 인증 필드 · 허용목록 · 팩)는
그대로 쓴다. 훅이 없는 것은 보조 실패다.

---

## 5. 요구와의 대조 — FR 일곱이 어디 앉나

| 요구 | 앉는 자리 | 게이트 |
|---|---|---|
| FR-1 가짜 홈 | `Fixed(dir)` + `Instrument` 의 `<dir>/home` · `.credentials.json` 복사 | CA1 |
| FR-2 허용목록 | `resolveComponents` (결정) + `Instrument` 의 `<dir>/mcp.json` (쓰기) | CA1 · CA4 |
| FR-3 선언과 광고 | `Local.MCP` · `mcpUp` · `mcpAttrs` · `costlyAttrs` | CA2 · CA3 |
| FR-4 워크스페이스 | `resolveComponents` 안의 `<cwd>/.mcp.json` 읽기 | CA4 · CA5 |
| FR-5 계약 문법 | `agentKeys` · `Grammar` · `AgentParams` · `examples/mcp.json` | CA3 · CA4 |
| FR-6 팩 | `readPack` (검증) + `Instrument` (펴기) + `executor.pack` (출처) | CA5 |
| FR-7 기록 | `HarnessResult.MCP` · `HarnessResult.Pack` | CA6 |

---

## 6. NFR 과의 대조

### 6.1 차단 게이트 다섯

```text
   crypto/tls · net/http 심볼 (enodectl.exe)   이 회차가 enodectl 을 안 건드린다.  영향 0
   패키지별 커버리지 80%                        internal/enode 가 걸린다.  아래 6.2
   허용목록 밖의 스킵 0                          새 스킵을 안 만든다
   U+2605 을 담은 파일 0                        emphasis-check.py 가 문서마다 돈다
```

### 6.2 커버리지를 하네스 없이 채운다

이 설계가 그것을 가능하게 만든 자리가 **결정과 쓰기의 분리**다.

```text
   resolveComponents   파일도 프로세스도 안 쓴다.  Job 하나를 넣고 오류를 잰다
   readPack            io.Reader 하나.  악성 tar 를 메모리에서 지어 넣는다
   mcpUp               LookPath · LookupEnv.  PATH 와 환경변수를 시험이 세운다
   mcpAttrs            Local 하나를 넣고 속성 맵을 잰다
   Fixed(dir)          순수 함수.  문자열 비교다
   Instrument          t.TempDir() 에 쓰고 난 파일을 읽는다
```

**하네스 실행파일이 필요한 것은 하나도 없다.** 가짜 stdio 서버(`true`)조차
단위 시험에는 필요 없다 — 실제 연결을 안 하기 때문이다.

### 6.3 security-baseline 준수

| 규칙 | 판정 | 이 설계의 근거 |
|---|---|---|
| SECURITY-05 입력 검증 | 준수 | `readPack` 이 절대경로 · `..` · 심볼릭 링크 · 상한을 거부하고 **파일을 쓰기 전에** 한다 |
| SECURITY-06 최소 권한 | 준수 | 허용목록은 요청한 이름만 연다. 와일드카드가 없다 |
| SECURITY-08 접근 제어 | 준수 | 노드 선언이 소유자만 쓰는 파일에 산다. 계약이 요청 안 한 서버는 0 |
| SECURITY-09 하드닝 | 준수 | 가짜 홈이 기본값이다. 여는 값이 없고, 못 지으면 단계가 실패한다 (Q2 = A) |
| SECURITY-11 보안 설계 | 준수 | 겹이 둘이다 — 가짜 홈과 strict 가 계정 커넥터를 각각 끊는다 (`services.md` 1.2) |
| SECURITY-12 자격증명 | 준수 | 복사본은 `0600` · 계장 디렉터리 안 · 단계 끝 삭제. 광고와 허용목록에는 이름만 |
| SECURITY-13 무결성 | 준수 (기록) | 실행 전 검증 대신 기록으로 족하다. 근거 셋은 `decisions.md` 6절 ③ |
| SECURITY-15 예외 처리 | 준수 | `components.md` 3절의 실패 등급 표. 치명이 전부 exec 앞에 모인다 |
| SECURITY-03 로깅 | 준수 | 새 로그가 이름만 찍는다 (`components.md` 4절) |
| SECURITY-01 · 07 | 기록만 | 기존 코드 사실. 새 포트 0 · 새 전송 0 |
| SECURITY-02 · 04 · 10 · 14 | N/A | 네트워크 중간자 · 웹 표면 · 새 의존 · 알림 경로를 안 만든다 |

---

## 7. 이 단계가 `decisions.md` 에 더하는 행

`requirements.md` 7절이 낸 셋은 이미 실렸다 (`decisions.md` 6절). **이 단계가
둘을 더한다.**

```text
   ④  계장 디렉터리를 못 만들면 단계 실패      Q2 = A.  오늘은 그대로 기동한다.
                                             가짜 홈이 앉을 자리가 없는 채로 띄우는 것은
                                             SECURITY-09 와 정면으로 어긋난다

   ⑤  Detector 인터페이스를 안 세운다          Q3 = A.  ADR-035 §4.2 의 권장값에서 벗어난다.
                                             그 권장이 지키려던 값은 ADR-068 의 Detector
                                             (시계)가 이미 세웠고, 같은 이름의 두 번째
                                             개념을 만드는 비용만 남는다
```

---

## 8. Units Generation 에 넘기는 것

**유닛 분해는 여기서 안 한다.** 넘기는 것은 세 가지다.

```text
   파일 목록      3절의 표.  cmd/runctl 이 0 이라 만지는 경로가 셋이다
   착수 순서의 뿌리  scene-gates.md 2절의 「먼저 서는 기능」 열.
                  3.1 가짜 홈이 3.2 · 3.3 · 3.4 · 3.5 · 3.6 의 앞이다
   접점          runner.go 하나가 짝 팩과 겹친다 (component-dependency.md 5절).
                 파일 행렬이 이 줄을 그대로 받는다
```

**빌드 시점 의존 하나** — `internal/contract` 의 `agentKeys` 가 서기 전에는
`agent.mcp` 나 `agent.pack` 을 적은 계약이 `400` 이다. 그 키를 쓰는 유닛보다
먼저 서야 한다. 반대로 가짜 홈과 허용목록의 뼈대는 계약 키 없이 설 수 있다 —
CA1 이 아무것도 요청하지 않은 단계를 재기 때문이다.
