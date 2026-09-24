# 이 저장소의 규약을 먼저 읽는다

@CONVENTIONS.md

표기 규약(강조는 마크다운 굵게로만 · 코드에 장식 문자 금지) · 언어 규약 ·
커밋 규약이 거기 있다. AI-DLC 가 내는 모든 산출물과 커밋에 그대로 적용된다 —
`aidlc-docs/` 아래 문서도 예외가 아니다.

---

# 축약어를 들여오면 그 커밋이 glossary 에 적는다

이 저장소는 자기 글자를 만든다 — 게이트 이름(`CP0` · `CA1`) · 웨이브(`W0`) ·
담당 handle 이 그렇다. **짧아서 만드는 것이고, 짧기 때문에 푼 말이 없으면 다음
사람이 못 푼다.** `CP` 와 `CA` 가 지금 그 자리다 — 두 팩의 `scene-gates.md` 가
그 글자로 게이트를 부르는데 무엇의 약자인지 적은 줄이 저장소에 0 이다.

```text
   대상       이 저장소가 지은 글자.  처음 나오는 자리에서 풀어 쓰지 않은 것
   대상 아님   밖에서 온 표준 어휘 — MCP · JSON · HTTP · tar
              정본(enode-design)의 어휘 — ADR-035 · R1 · I3.  거기가 정본이다.
              두 벌로 두면 갈린다 (CONVENTIONS 1.4 의 「도구는 두 벌로 두지 않는다」)
```

**자리는 하나다** — 저장소 루트의 `GLOSSARY.md`. 없으면 그 글자를 들여오는
커밋이 만든다.

```text
   적는 것    푼 말 · 한 줄 뜻 · 그 어휘를 지는 문서의 주소
   때        그 글자를 처음 들여오는 커밋이다.  다음 커밋이 아니다 —
             미루면 지은 사람이 떠나고 아무도 못 푼다
   집행      사람이다.  단계 승인과 PR 에서 본다.  도는 스크립트가 없는 채로
             기계 검사라고 적으면 그것이 거짓 초록이다
```

**푼 말을 모르면 적지 말고 묻는다.** 짐작한 확장을 적으면 glossary 가 틀린 값을
굳히고, 그것은 비어 있는 것보다 나쁘다 — 다음 사람이 그 줄을 근거로 쓴다.

적용 범위는 위 절과 같다. **AI-DLC 산출물도 예외가 아니다** — 회차 문서가
새 게이트 이름이나 유닛 약칭을 들여오면 그 단계의 커밋이 함께 적는다.

---

# AI-DLC 워크플로

이 저장소는 **AI-DLC v1.0.1** 로 돈다. 판과 규칙 전문이 `.aidlc/aidlc-rules/`
에 있고 `VERSION` 이 그 판을 진다. 아래는 그 워크플로 정본을 그대로 가져온
것이다 — 사본을 두 벌로 두지 않는다.

@.aidlc/aidlc-rules/aws-aidlc-rules/core-workflow.md

---

# AI-DLC 문서 루트 — 회차별 layering

회차마다 브랜치를 따로 돌리므로, **AI-DLC 산출물 문서 루트는
`aidlc-docs/<브랜치 이름>/`** 다 (예: `aidlc-docs/v3-run-harness-components/`).
**Inception 과 Construction 이 같은 루트를 쓴다.** 상태 파일과 감사 로그도 회차마다
하나다 — `aidlc-docs/<브랜치>/aidlc-state.md` · `aidlc-docs/<브랜치>/audit.md`.
**루트 `aidlc-docs/` 에 상태·감사·단계 문서를 다시 쓰지 않는다** — 그러면 회차끼리
같은 파일을 두고 부딪친다.

공용은 하나다 — Reverse Engineering 분석은 회차와 무관하게 참이라
`aidlc-docs/inception/reverse-engineering/` 에 공용으로 둔다.

`aidlc-state.md` 와 `design/*.pen` 은 통째로 다시 쓰거나 자동 병합할 수 없는
파일이다. **`aidlc-state.md` 는 그 회차의 진행자가 고친다.** 소유자가 하나라 병합에서
안 부딪친다. `design/*.pen` 은 회차마다 나뉘지 않으므로 **진행자 한 사람이** 고치고,
시안은 회차마다 새 `.pen` 파일로 만든다. 이어 붙이는 `audit.md` 는
`.gitattributes` 의 `merge=union` 으로 git 이 합친다. `CONVENTIONS.md` 3.2 가 이
layering 위에서 부딪히는 자리를 적는다 — 남는 것은 `design/` 하나다.

**Construction 도 회차 루트에 쓴다.** 유닛의 functional-design · nfr · code 요약은
`aidlc-docs/<브랜치>/construction/<유닛>/`, 단계 계획은 `.../construction/plans/<유닛>-*`,
Build and Test 는 `.../construction/build-and-test/` 다. 유닛은 `unit/<유닛>` 브랜치에서
돌고 **PR 로 `main` 에 병합**한다 — 그 유닛의 장면 게이트가 초록인 뒤에만
(`CONVENTIONS.md` 3.1 · 3.3 이 브랜치 이름과 시점의 정본이다). 누가 어느 유닛을 맡는지는
그 회차의 Units Generation 산출물이 적고, handle 은 `aidlc-docs/construction-roster.md` 에 있다.

**한 회차의 문서는 한 폴더에 모은다.** Construction 은 Inception 의 유닛 정의 · 요구 ·
설계를 계속 읽고, 도중의 되물음이 Inception 문서를 고치기도 한다. 루트가 둘이면 한 회차를
읽으려고 두 폴더를 오가고, 상태 파일 둘이 서로를 가리켜야 한다. 굽기 회차
(`v4-run-finalize-bake`, 2026-09-24)에서 이렇게 바꿨다.

**담당 루트 `aidlc-docs/<handle>/` 는 앞 회차의 기록이다.** v1 ~ v3 회차의
Construction 이 거기 있다. 옮기지 않고, 새로 쓰지도 않는다. 대회 때 넷이 한 회차를
동시에 돌면서 상태 파일 한 장을 두고 부딪쳐 생긴 자리다. **여럿이 한 회차의 Construction
을 동시에 돌면 회차의 `aidlc-state.md` 가 다시 부딪친다** — 그런 회차가 오면 그 회차의
Workflow Planning 이 상태를 나눌 자리를 정한다.
