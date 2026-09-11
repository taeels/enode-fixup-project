# 정본 — `enode-design` 과 이 팩의 연결

설계 정본은 이 저장소의 서브모듈 `enode-design/` 이다. **어긋나면
`enode-design/protocol/INVARIANTS.md` 가 이긴다.**

```text
   서브모듈 고정       git ls-tree HEAD enode-design
   정본과의 거리       cd enode-design && git rev-list --count HEAD..origin/main
   갱신                git submodule update --remote 는 진행자만
```

---

# 1. 이 팩이 딛는 정본

| 문서 | 상태 | 이 팩에서 쓰는 자리 |
|---|---|---|
| `protocol/INVARIANTS.md` | 정본 | `I4` 봉인되지 않은 것은 Record 가 아니다 — 진행 로그가 Record 가 아닌 이유. `GET record` 의 `409` 는 그대로 |
| `ADR-025` 실행 중 관측 | 결정 · **§7 ③ 을 연다** | §3 새 표면을 안 만든다(라우트 하나는 「이미 있는 조회의 연장」) · §5 진행 상태는 Record 가 아니다 · §7 ③ 실행 중 로그의 여는 조건 |
| `ADR-014` Mediator 가 시퀀싱 · 노드가 당긴다 | 확정 | 결정 2 — Mediator 는 노드를 부르지 않는다. 청크는 노드가 민다 |
| `ADR-005` Run Record | 확정 | `logs/NN-*.log` 는 원문 그대로. 성질 4 자기충족. 파서는 읽을 뿐이다 |
| `ADR-013` 에이전트 단계 계약 | 확정 | 「stdout JSON 은 로그와 섞인다」 — 봉투는 마지막 줄로 온다. 산출물은 여전히 `$OUT` 파일이다 |
| `ADR-020` 종료 사유 어휘 | 결정 · 구현됨 | `Reason` 은 안 바뀐다. 봉투 파서가 그대로 채운다 |
| `ADR-038` 못 하겠다고 말한다 | 결정 · 구현됨 | `_cannot` 은 `$OUT` 파일. 트랜스크립트와 무관하다 |
| `ADR-032` · `ADR-047` 되묻기 | 확정 | 되묻기는 단계를 끊는다. 입력 스트림을 안 여는 이유 |
| `ADR-065` 운용 관측은 배정 판정이 아니다 | 결정 | GET log 는 판정을 싣지 않는다. 보호 규칙은 `GET /v1/runs` 와 같다 |
| `ADR-022` §9.3 | 확정 | MCP 도구는 감싸기만 한다 — `run.log` 가 `GET` 과 글자까지 같은 이유 |
| `agent-runtime` R3 · R4 | 정본 | `Decode(r, exitCode, emit)` — 사건을 나를 수 있는 시그니처가 이미 있다. R4 가 stream-json 을 「열 때」로 잡아둔 자리 |

---

# 2. 정본이 코드와 어긋나 있는 자리

```text
   agent-runtime R3   「배치는 스트리밍의 퇴화형이다」 — 코드는 아직 배치다.  Decode 가
                      EOF 까지 읽고 사건 하나(final)만 낸다
   ADR-025 §7 ③        순연.  코드에 실행 중 로그를 내는 표면이 없다.  올리는 PUT 만 있고
                      내려받는 GET 이 없다 (앞 팩 §6.5 가 그래서 tar 를 골랐다)
```

# 3. 앞 팩과 어긋나는 자리

앞 팩(루트의 다섯 파일)은 정본이 아니라 그 회차의 요구다. 어느 문장을 대체하는지는
`decisions.md` 5절이 센다. 정본과는 어긋나지 않는다 — `ADR-025` 가 여는 조건을 적어
둔 자리를 여는 것이다.

# 4. 정본에 더하는 것 (역방향)

`decisions.md` 6절이 정본이다. 요약하면 셋이다.

```text
   ADR-025 §7 ③     열림.  폴링.  근거와 일자
   mediator-api     GET log 절.  PUT log 의 응답 본문
   agent-runtime    R3 · R4 — Decode 가 사건을 배출하고 stream-json 이 기본이다
```
