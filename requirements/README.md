# 요구 팩 — 팩마다 디렉터리 하나

`requirements/` 는 AI-DLC 의 Requirements Analysis 가 **입력으로** 읽는 요구
팩이다. 팩이 하나일 때는 루트에 다섯 파일이었다. 이제 팩이 여럿이므로 규칙을
하나 더 둔다 — **팩마다 디렉터리 하나, 다섯 파일 이름은 같다.**

```text
   루트의 다섯 파일     2026-09 회차의 팩 (기능 일곱 + 시연 다섯).  옮기지 않는다.
                        아직 닫히지 않은 기능(3.3.1 Mediator MCP · 3.2.3 되묻기)의
                        입력이고, 지난 회차의 기록 전부가 이 경로를 가리킨다
   <팩 이름>/           그 뒤의 팩.  같은 다섯 파일을 둔다
   archive/             닫힌 팩이 가는 자리.  루트의 다섯은 3.3.1 · 3.2.3 이 닫히면 간다
```

## 목록

| 팩 | 자리 | 상태 | 회차 브랜치 |
|---|---|---|---|
| 2026-09 기능 일곱 + 시연 다섯 | 루트 다섯 파일 | 열려 있다 — 3.3.1 · 3.2.3 이 남았다 | `v1-run-dhseo` · `v1-run-dhseo-cardnews` · `v2-run-shin_pen_drawing` |
| harness-components — 하네스 구성요소 | `harness-components/` | 제안 (2026-09-11) | `v3-run-harness-components` |
| transcript — 트랜스크립트 | `transcript/` | 제안 (2026-09-11) | `v3-run-transcript` |

뒤의 둘은 **사내 실측(2026-09-11) 후속**이고 서로 독립이다. 한 팩이 아니라 둘로
가른 근거는 각 팩의 `features.md` 1절에 있다.

## 다섯 파일의 역할

```text
   features.md      요구.  목적 · 기능 · 보안 · 수용 기준.  옛 팩의 enode-features.md 자리
   decisions.md     이미 정해진 값.  다시 논의하지 않는다
   constraints.md   안 만드는 것 · 구조 불변식 · 접점
   scene-gates.md   장면 조각 게이트.  수용 기준의 실동작판
   canon.md         enode-design 정본과의 연결.  어긋나면 INVARIANTS 가 이긴다
```

## 규칙 넷

- **회차가 어느 팩을 읽는지는 그 회차의 `aidlc-docs/<브랜치>/` 가 적는다.** 이
  문서는 목록만 진다. 목록이 팩을 고르지 않는다.
- **뒤 팩이 앞 팩의 문장을 대체하면 뒤 팩의 `decisions.md` 가 그 자리를 적는다.**
  앞 팩의 파일은 고치지 않는다 — 그 회차의 기록이다. 어느 문장이 대체됐는지는
  뒤 팩에서 찾는다.
- **두 팩이 같은 파일을 만지는 자리는 각 팩의 `constraints.md` 접점 절이 적는다.**
  진행자가 직렬로 병합한다 (`CONVENTIONS.md` 3.1).
- **팩은 유닛 분해를 주지 않는다.** 어느 유닛이 무엇을 맡는지는 Units Generation 이
  정한다. 팩은 파일 행렬을 요구할 뿐 대신 내지 않는다.
