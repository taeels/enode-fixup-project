# AI-DLC Audit Log

모든 사용자 입력과 결정을 시간순으로 기록한다. 사용자 원문은 요약하지 않고 그대로 싣는다.
타임스탬프는 `date -u +%Y-%m-%dT%H:%M:%SZ` (ISO 8601 UTC).

---

## Workspace Detection — 초기 사용자 요청
**Timestamp**: 2026-09-08T05:36:09Z
**User Input**: "enode 에 기능을 더한다. 요구사항은 이미 문서다.

  requirements/decisions.md        정해진 값과 근거
  requirements/enode-features.md   기능 일곱. 목적 · 요구사항 · 수용 기준
  requirements/scene-gates.md      장면 조각 게이트. 이것이 수용 기준이다
  requirements/canon.md            설계 정본과의 연결. 어느 ADR 을 읽을지 적혀 있다
  requirements/constraints.md      안 만드는 것과 구조 불변식
  design/README.md                 화면 아홉 장의 규칙
  design/exports/*.png             그 화면의 그림. .pen 은 못 여니 이것을 본다

설계 정본은 서브모듈 enode-design/ 이고, 어긋나면 protocol/INVARIANTS.md 가
이긴다. aidlc-docs/ 는 비어 있다.

값은 decisions.md 가 다 갖고 있고 미정은 하나뿐이다 (enode-features.md 5절).
그래서 물을 것이 거의 없다 — 물어야 한다면 계약(함수 서명 · 파일 경로 ·
응답 모양)에 걸리는 것만, 단계마다 셋까지. 권장값을 벗어나야 하면 묻지 말고
근거를 decisions.md 에 행으로 적어라.

유닛 분해는 팩이 주지 않는다. Units Generation 이 정하되 유닛마다 어느 유닛에
의존하는지를 적어라 — Construction 을 그 그래프대로 병렬로 돌린다. 화면이 있는
유닛은 완료 조건에 그 화면의 버튼을 전부 나열한다.

완료는 scene-gates.md 가 잰다. 3절의 명령을 실제로 돌리고, 화면이 있는 조각은
2.1 의 눈으로 보는 항목을 따로 본다 — API 검증은 화면 없이도 초록이 된다.
조각이 초록이 아니면 그 유닛을 병합하지 않는다.

커밋은 토픽 브랜치 위에서 한다. Inception 은 v1-run-<이름> 하나에서 직렬로,
Construction 은 유닛마다 unit/<유닛> 을 따서 병렬로 간다. 단계 승인마다
커밋하고, 유닛은 조각이 초록인 뒤에 v1-run 으로 병합한다.

병렬이 부딪히는 자리가 셋이다. 유닛 브랜치는 여기를 안 건드린다.
  aidlc-docs/audit.md · aidlc-state.md   각각 한 장이다. 진행자가 합친다
  design/                                 그림은 진행자만 고친다
유닛 둘이 같은 파일을 만져야 하면 (internal/api/api.go 의 등록 줄이 그 자리다)
파일 행렬에 표시해라. 그 파일은 진행자가 직렬로 병합한다.

타임스탬프는 date -u +%Y-%m-%dT%H:%M:%SZ 로 적는다."
**AI Response**: 요청을 기록했다. 요구 팩 다섯 파일과 `design/README.md`, 화면 그림, AI-DLC 규칙(v1.0.1)을 읽었다. Workspace Detection 을 실행했다 — 브라운필드 확정, `aidlc-docs/` 비어 있음, RE 산출물 없음. Inception 실행 브랜치 `v1-run-dhseo` 를 `main` 에서 땄다(CONVENTIONS.md 3.1). `aidlc-state.md` 를 만들고 다음 단계로 Reverse Engineering 을 시작했다.
**Context**: INCEPTION - Workspace Detection. 초기 요청 로그. 브라운필드 감지.

---

## Workspace Detection — 확장 opt-in (사전 확정)
**Timestamp**: 2026-09-08T05:36:09Z
**User Input**: (요구 팩 `decisions.md` §1 이 착수 전에 확정 — 2026-09-04 사용자 결정)
**AI Response**: 확장 3종의 enable 상태를 `aidlc-state.md` 의 Extension Configuration 에 기록했다 — security-baseline 켬, resiliency-baseline 끔, property-based-testing 끔. Requirements Analysis Step 5.1 에서 재확인한다. security-baseline 전문을 로드했다(켜져 있으므로).
**Context**: INCEPTION - Workspace Detection. 확장 결정은 요구 팩이 이미 닫음.

---
