# ui — obs 전달 전 준비

**상태**: 설계 초안과 테스트 입력. 실제 obs 와 연결한 결과가 아니다.
사용자가 승인한 사전 준비 범위를 기록한 계획은
[`ui-preparation-plan.md`](../../plans/ui-preparation-plan.md)다.

**후속 진행**: [상세 설계](../../plans/ui-functional-design-plan.md)와
[코드 생성 계획](../../plans/ui-code-generation-plan.md)은 승인됐고 UI 1~8을 구현했다.
obs·queue 인수와 sandbox 출처 승인을 완료했다. 이 디렉터리는 준비 당시의 계약
후보·합성 자료를 보존하며 현재 대기 조건이 아니다. [담당 현재 상태](../../../aidlc-state.md)와
[queue 인수](../code/queue-integration-review.md)를 우선한다.

## 읽는 순서

1. [UI 설계 초안](ui-design-draft.md) — 실 함대 화면과 상태 전이, 데이터 흐름.
2. [소비 API 계약표](api-contract.md) — 정본·현재 코드·미확정 후보의 구분.
3. [샘플 응답 안내](fixtures.md) — JSON 파일과 화면 기대값, 시각 기준.
4. [통합 검증 항목](integration-checklist.md) — obs 를 받은 뒤 확인할 것.

JSON 은 저장소 루트 기준 `internal/api/ui/testdata/obs-contract/` 에 있다.
공개되는 `static/` 과 분리했다. 실패한 실제 API 를 샘플 데이터로 대신하는
동작이나 새 mock 서버는 만들지 않았다.

## 사전 준비 당시의 접점

- `requires[].as` 와 `steps[].uses` 로 요구 능력과 단계를 잇는다.
- ADR-069 의 `requires[].attrs` 예시는 중첩 객체지만 기존
  `contract.Require.MarshalJSON` 은 속성을 평평하게 낸다. obs 응답 형식을
  확인해야 한다. 두 후보 JSON 을 제공하며 어느 쪽도 구현됐다고 주장하지 않는다.
- `lease` 부재의 null/생략과 목록의 종료 `verdict` 형태는 obs 구현에서 확인한다.
- `chosen` 이 없다고 false 로 간주하면 기존 서버의 미구현을 숨긴다.
- `sandbox`는 사전 준비 당시 출처 미정이었다. 후속 FD의
  capabilities[].attrs.sandbox 광고값/미제공 표시안은 최태양님의 승인을 전달받았다.
  전용 서버 필드를 추가하지 않으며 실제 장비 광고 대조는 공동 장면에 남아 있다.

## Codex 설정

AI-DLC v1.0.1 규칙은 `.aidlc/aidlc-rules/` 에 있고 Codex 진입점은 루트
`AGENTS.md` 다. 별도 `aidlc` 스킬은 설치되지 않았다. 이 세션에서 규칙을
명시적으로 읽어 사전 준비에 적용했다. 재개 시 담당 상태는
`aidlc-docs/runixs/aidlc-state.md` 를 읽는다.

## 완료의 경계

사전 준비의 검증은 샘플과 문서의 일관성까지다. 정식 Functional Design 리뷰,
Code Generation, CP0·CP1·CP2·CP7 및 데모 게이트의 통과를 대신하지 않는다.
이후 obs·queue 응답 대조는 각각의 인수 기록에 남겼다. 공동 장면의 남은 조건은
현재 담당 상태와 코드 계획을 따른다.

## 준비 산출물 검증 — 2026-09-08

- JSON 18개 구문·manifest 대응, 노드/Run 연결, QUEUED 의 요구 자원 점유,
  중첩/평탄 requires 후보의 의미 일치, 그래프 의존, ASKED 와 시각 경계 검사 통과.
- 문서 링크·펜스·공백 검사, `enode-design/scripts/emphasis-check.py`,
  `git diff --check` 통과.
- `go test -count=1 -cover ./internal/api/ui` 시도는 `go: command not found` 로
  실행되지 않았다. 표준 설치·캐시 위치에서도 Go 실행 파일을 찾지 못했다.
  이후 로컬 Go 1.26.6을 설치해 같은 검사를 실행했고 **92.3% statements로 통과**했다.
  정확한 경로·명령은 [코드 계획](../../plans/ui-code-generation-plan.md)에 있다.
- 추적된 제품 코드·공유 접점·공용 상태 변경 없음. 테스트 JSON 은
  `//go:embed static` 대상 밖에 있음을 소스 경로와 기존 UI 패키지 빌드로 확인했다.
  실제 obs 연결·브라우저 동작·전체 CP 게이트는 남아 있다.
