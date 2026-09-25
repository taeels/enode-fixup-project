# v4-run-finalize-bake — 회차의 산출물 자리

**지금은 비어 있다. 이것이 정상이다.** 이 회차가 읽는 요구 팩은 `requirements/finalize-bake/`다.
AI-DLC는 Workspace Detection으로 브라운필드임을 확인하고, 공용 Reverse Engineering
(`aidlc-docs/inception/reverse-engineering/`)의 신선도를 판정한 뒤 Requirements Analysis로
간다. 상태 파일 `aidlc-state.md`와 감사 로그 `audit.md`는 Workspace Detection이 여기에 만든다.

| 항목 | 값 |
|---|---|
| 팩 | `requirements/finalize-bake/` 다섯 파일 |
| 출발 | 2026-09-21~23 설계 검토. 사내 증상 「결정론적 BitBake는 끝났는데 임대가 안 풀린다」 |
| 정본 | 서브모듈 `enode-design` `a2c4ac6`(enode-design `main`, #15). ADR-075(결정), ADR-076(초안), ADR-077(초안) |
| 코드 기준선 | `main` `826b40f`. `unit/runtime-environment-profile`이 #59로 `main`에 들어갔고 이 브랜치가 그것을 합쳤다(`4facffb`) |
| Office Task | `f806768d` |
| 문서 루트 | 여기. 상태 파일과 감사 로그도 여기(`CLAUDE.md`의 회차별 layering) |

## 코드 기준선

이 회차 브랜치는 `main`에서 땄다. 이 팩이 넓히는 ADR-073 구현(`internal/enode/runc_overlay_linux.go`,
`internal/environment/`, `cmd/enode`의 runtime 선택)은 `unit/runtime-environment-profile`에만
있어서, Workspace Detection이 코드가 없는 자리를 보지 않도록 그 브랜치를 여기에 직접 합쳤다
(`b5659ae`, 2026-09-23). 합친 뒤 `go build ./...`와 `go vet ./...`가 통과했다.

`unit/runtime-environment-profile`이 나중에 PR로 `main`에 따로 병합되면 이 회차의 PR에서 그만큼이
빠진다. 먼저 `main`에 병합하는 쪽을 고르면 이 합치기를 되돌리고 `main`을 합친다.

**2026-09-23에 먼저 `main`에 병합하는 쪽으로 닫았다** (Requirements 재질문의 답 A). 정본은
enode-design #15(`a2c4ac6`), 구현은 #59(`826b40f`)로 둘 다 병합 커밋으로 들어갔다. 병합
커밋이라 `b5659ae`를 되돌리지 않고 `main`을 합쳤다(`4facffb`) — 나무가 한 줄도 안 바뀐다.
핀은 README 3.1대로 enode-design `main`인 `a2c4ac6`으로 옮겼다. `369270a`와 나무가 같다.

## 낡은 산출물을 미리 실어 두지 않는다

실어 두면 그 단계를 실행한 것이 아니라 물려받은 것이 되고, 물려받은 것은 코드가 움직인 만큼
조용히 거짓이 된다. 이 설계 검토의 기록(원장 `f806768d`, 인계 상태 `EN-048df7c4`)은 요구
팩이 이미 옮겼다.
