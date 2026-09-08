# 이 저장소의 규약을 먼저 읽는다

@CONVENTIONS.md

표기 규약(강조는 마크다운 굵게로만 · 코드에 장식 문자 금지) · 언어 규약 ·
커밋 규약이 거기 있다. AI-DLC 가 내는 모든 산출물과 커밋에 그대로 적용된다 —
`aidlc-docs/` 아래 문서도 예외가 아니다.

---

# AI-DLC 워크플로

이 저장소는 **AI-DLC v1.0.1** 로 돈다. 판과 규칙 전문이 `.aidlc/aidlc-rules/`
에 있고 `VERSION` 이 그 판을 진다. 아래는 그 워크플로 정본을 그대로 가져온
것이다 — 사본을 두 벌로 두지 않는다.

@.aidlc/aidlc-rules/aws-aidlc-rules/core-workflow.md

---

# AI-DLC 문서 루트 — 회차별 layering

다중 사용자가 회차마다 브랜치를 따로 돌리므로, **AI-DLC 산출물 문서 루트는
`aidlc-docs/<브랜치 이름>/`** 다 (예: `aidlc-docs/v1-run-dhseo/`). 상태 파일과
감사 로그도 회차마다 그 아래 따로 둔다 — `aidlc-docs/<브랜치>/aidlc-state.md` ·
`aidlc-docs/<브랜치>/audit.md`. **루트 `aidlc-docs/` 에 상태·감사·단계 문서를
다시 쓰지 않는다** — 그러면 회차끼리 같은 파일을 두고 부딪친다.

공용은 하나다 — Reverse Engineering 분석은 회차와 무관하게 참이라
`aidlc-docs/inception/reverse-engineering/` 에 공용으로 둔다.

`aidlc-state.md` 와 `design/*.pen` 은 통째로 다시 쓰거나 자동 병합할 수 없는
파일이라 **진행자 한 사람이** 고친다 — 회차 브랜치를 병합한 뒤 진행자가 상태를
정리하고, 시안은 회차마다 새 `.pen` 파일로 만든다. 이어 붙이는 `audit.md` 는
`.gitattributes` 의 `merge=union` 으로 git 이 합친다. 이 layering 이
`CONVENTIONS.md` 3.2 의 flat 충돌 모델(상태·감사 한 장씩)을 대신한다.

**Construction 은 담당별로 나눈다.** 여러 사람이 동시에 유닛을 맡으므로,
Construction 산출물의 문서 루트는 담당 handle 로 **`aidlc-docs/<handle>/`** 다
(예: `aidlc-docs/taeels/`). 각자 자기 유닛의 functional-design · nfr · code 요약과
자기 `aidlc-state.md` · `audit.md` 를 거기 쓴다. 각자 **자기 브랜치 위에서 작업하고
PR 로 `main` 에 병합**한다 — 그 유닛의 장면 게이트가 초록인 뒤에만. 배정과 handle 은
`aidlc-docs/construction-roster.md`. 문서 루트는 언제나 그 산출물의 소유자 하나로
갈린다 — Inception 회차는 회차(브랜치) 이름, Construction 은 담당 handle.
