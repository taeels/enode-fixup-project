# v4-run-finalize-bake — 회차의 산출물 자리

**지금은 비어 있다. 이것이 정상이다.** 이 회차가 읽는 요구 팩은 `requirements/finalize-bake/`다.
AI-DLC는 Workspace Detection으로 브라운필드임을 확인하고, 공용 Reverse Engineering
(`aidlc-docs/inception/reverse-engineering/`)의 신선도를 판정한 뒤 Requirements Analysis로
간다. 상태 파일 `aidlc-state.md`와 감사 로그 `audit.md`는 Workspace Detection이 여기에 만든다.

| 항목 | 값 |
|---|---|
| 팩 | `requirements/finalize-bake/` 다섯 파일 |
| 출발 | 2026-09-21~23 설계 검토. 사내 증상 「결정론적 BitBake는 끝났는데 임대가 안 풀린다」 |
| 정본 | 서브모듈 `enode-design` `369270a`. ADR-075(결정), ADR-076(초안), ADR-077(초안) |
| 코드 기준선 | `unit/runtime-environment-profile`(상위 `0a67716`). 아직 `main`에 없다 |
| Office Task | `f806768d` |
| 문서 루트 | 여기. 상태 파일과 감사 로그도 여기(`CLAUDE.md`의 회차별 layering) |

## Workspace Detection 전에 할 일

**이 회차 브랜치는 `main`에서 땄기 때문에 이 팩이 넓히는 코드가 아직 없다.** ADR-073의 실행
환경 구현(`internal/enode/runc_overlay_linux.go`, `internal/environment/`, `cmd/enode`의 runtime
선택)이 `unit/runtime-environment-profile`에만 있다. 둘 중 하나를 먼저 한다.

1. `unit/runtime-environment-profile`을 PR로 `main`에 병합하고, 이 브랜치가 `main`을 합친다.
2. 이 브랜치에 `unit/runtime-environment-profile`을 직접 합친다.

그러지 않고 Workspace Detection을 돌리면 코드가 없는 자리를 보고 요구를 쓰게 된다.

## 낡은 산출물을 미리 실어 두지 않는다

실어 두면 그 단계를 실행한 것이 아니라 물려받은 것이 되고, 물려받은 것은 코드가 움직인 만큼
조용히 거짓이 된다. 이 설계 검토의 기록(원장 `f806768d`, 인계 상태 `EN-048df7c4`)은 요구
팩이 이미 옮겼다.
