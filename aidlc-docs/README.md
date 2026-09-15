# aidlc-docs — AI-DLC 산출물이 여기 쌓인다

문서는 **소유자 하나**로 나눈다. Inception 산출물의 소유자는 회차(브랜치)이고,
Construction 산출물의 소유자는 담당(handle)이다. 소유자가 갈리면 두 손이 같은
파일을 두고 안 부딪친다.

## 무엇이 어디에 쌓이나

```text
   aidlc-docs/
   ├── inception/
   │   └── reverse-engineering/     공용 — 브라운필드 분석 (회차 무관하게 참)
   ├── <브랜치 이름>/                INCEPTION — 회차 하나의 산출물
   │   ├── aidlc-state.md            단계 진행 (그 회차의 진행자가 고친다)
   │   ├── audit.md                  사용자 입력·결정 기록 (merge=union)
   │   └── inception/
   │       ├── plans/
   │       ├── requirements/
   │       ├── user-stories/
   │       └── application-design/
   ├── <handle>/                     CONSTRUCTION — 담당 하나의 산출물
   │   ├── aidlc-state.md            그 담당이 고친다
   │   ├── audit.md                  merge=union
   │   └── construction/
   │       ├── plans/
   │       ├── <유닛>/
   │       └── build-and-test/
   ├── construction-roster.md        배정과 handle (공용)
   ├── operations/
   └── README.md                     이 문서 (공용)
```

지금 있는 회차 — `v1-run-dhseo`(대회 데모 전환) · `v2-run-shin_pen_drawing`(데모
시안) · `v1-run-dhseo-cardnews`(온보딩 카드뉴스 · 게스트 로그인) ·
`v3-run-harness-components`(하네스 구성요소).

지금 있는 handle — `shin-son` · `nacl1119` · `taeels` · `runixs`.

## 규칙 셋

- **문서 루트는 소유자 하나로 갈린다** — Inception 은 `aidlc-docs/<브랜치 이름>/`,
  Construction 은 `aidlc-docs/<handle>/`. 루트 `aidlc-docs/` 에 상태·감사·단계
  문서를 다시 쓰지 않는다. 회차끼리 같은 파일을 두고 부딪치기 때문이다
  (`CLAUDE.md` 의 문서 루트 규약).
- **RE 만 공용** — `reverse-engineering` 은 회차와 무관하게 참이라 한 벌만 둔다.
- **자기 것은 자기가 고친다** — `aidlc-state.md` 는 자동 병합이 안 되지만 루트가
  갈려서 소유자가 하나다. 그 소유자가 고치고 자기 브랜치에 싣는다. `audit.md` 는
  `merge=union` 으로 git 이 합친다. 루트가 안 갈리는 `design/*.pen` 만 진행자가
  고치고, 시안은 회차마다 새 `.pen` 파일로 만든다 (`CONVENTIONS.md` 3.2 · 3.4).

## 애플리케이션 코드는 여기 안 들어온다

저장소 뿌리에 들어간다 (`cmd/` · `internal/`). 요구 팩은 `requirements/` 와
`design/` 이고 AI-DLC 의 Requirements Analysis 가 그것을 입력으로 읽는다.

**낡은 산출물을 미리 실어 두지 않는다.** 실어 두면 그 단계를 실행한 것이 아니라
물려받은 것이 되고, 물려받은 것은 코드가 움직인 만큼 조용히 거짓이 된다.
