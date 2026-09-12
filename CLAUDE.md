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
상태 파일과 감사 로그도 회차마다 그 아래 따로 둔다 — `aidlc-docs/<브랜치>/aidlc-state.md` ·
`aidlc-docs/<브랜치>/audit.md`. **루트 `aidlc-docs/` 에 상태·감사·단계 문서를
다시 쓰지 않는다** — 그러면 회차끼리 같은 파일을 두고 부딪친다.

공용은 하나다 — Reverse Engineering 분석은 회차와 무관하게 참이라
`aidlc-docs/inception/reverse-engineering/` 에 공용으로 둔다.

`aidlc-state.md` 와 `design/*.pen` 은 통째로 다시 쓰거나 자동 병합할 수 없는
파일이다. **`aidlc-state.md` 는 그 문서 루트의 소유자가 고친다** — 회차 것은 그
회차의 진행자가, Construction 것은 그 담당이. 소유자가 하나라 병합에서 안
부딪친다. `design/*.pen` 은 루트가 갈리지 않으므로 **진행자 한 사람이** 고치고,
시안은 회차마다 새 `.pen` 파일로 만든다. 이어 붙이는 `audit.md` 는
`.gitattributes` 의 `merge=union` 으로 git 이 합친다. `CONVENTIONS.md` 3.2 가 이
layering 위에서 부딪히는 자리를 적는다 — 남는 것은 `design/` 하나다.

**Construction 은 담당별로 나눈다.** Construction 산출물의 문서 루트는 담당
handle 로 **`aidlc-docs/<handle>/`** 다 (예: `aidlc-docs/taeels/`). 각자 자기 유닛의
functional-design · nfr · code 요약과 자기 `aidlc-state.md` · `audit.md` 를 거기 쓴다.
유닛은 `unit/<유닛>` 브랜치에서 돌고 **PR 로 `main` 에 병합**한다 — 그 유닛의 장면
게이트가 초록인 뒤에만 (`CONVENTIONS.md` 3.1 · 3.3 이 브랜치 이름과 시점의 정본이다).
배정과 handle 은 `aidlc-docs/construction-roster.md`.

**가르는 기준은 사람 수가 아니라 소유자다.** 대회 때는 넷이 동시에 돌아서 handle
루트가 생겼지만, 한 손이 도는 회차도 handle 루트를 쓴다 — 그래야 회차가 바뀌어도
그 사람의 Construction 산출물 주소가 안 바뀐다. 문서 루트는 언제나 그 산출물의
소유자 하나로 갈린다 — Inception 회차는 회차(브랜치) 이름, Construction 은 담당 handle.
