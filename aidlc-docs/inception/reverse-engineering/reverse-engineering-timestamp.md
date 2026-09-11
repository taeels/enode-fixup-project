# Reverse Engineering Metadata

**Analysis Date**: 2026-09-08T07:17:58Z
**Analyzer**: AI-DLC (워크플로 — 리더 7 · 라이터 6, 13/13 성공 · 오류 0)
**Workspace**: /home/sunny/enode-fixup-project
**Total Files Analyzed**: 비테스트 소스 75 · 테스트 포함 143 (`cmd/` · `internal/`) + `enode-design` 정본(`INVARIANTS.md` · `mediator-api.md` · ADR 다수)

## Artifacts Generated
- [x] business-overview.md
- [x] architecture.md
- [x] code-structure.md
- [x] api-documentation.md
- [x] component-inventory.md
- [x] technology-stack.md
- [x] dependencies.md
- [x] code-quality-assessment.md

## 검증 (라이터 자기보고를 다시 실측함)
- `enode-design/scripts/emphasis-check.py` 통과 — 위반 0
- U+2605 · 이모지 · 장식 화살표 없음 (전 산출물 grep)
- 코드펜스 짝 맞음 · Mermaid 4개(architecture 2 · business-overview 1 · dependencies 1) 문법 유효
- 라우트 15개가 `internal/api/api.go:61-75` 원본과 정확히 일치 (`POST /v1/nodes` · `GET /v1/runs/{id}` 상세만 · 목록 라우트 없음 확인)

**커밋**: 파일은 `1fe2145` ("Adding R/E") 가 실었다 — 사용자가 공유용으로 먼저 커밋.

---

## 부분 재측정 — 2026-09-11 (v3-run-harness-components)

**Refresh Date**: 2026-09-11T13:13:54Z
**Scope**: 네 경로만 — `internal/enode` · `internal/contract` · `cmd/iapadapter` ·
`cmd/runctl`. `harness-components` 회차의 Requirements 질문 1 의 답이 B 였다.
**Artifacts Touched**: `code-structure.md` · `component-inventory.md` 두 장.
나머지 여섯 문서는 2026-09-08 판 그대로다.

기준선 `06215ff` (2026-09-08T07:17:58Z 직전 커밋) 과 오늘 HEAD 를 대서 세었다.

```text
   internal/enode      비테스트 소스 일곱 파일이 바뀌었고 셋이 새 파일이다 —
                       policy.go (ADR-063 소유자 정책) · status.go (ADR-068 상태 파일) ·
                       transcript.go (하네스 stdout 링).  advertise.go · claim.go ·
                       leases.go · runner.go 가 그에 맞춰 늘었다
   internal/contract    advert.go 에 Advert.Policy 와 Drain 어휘 상수 셋이 늘었다.
                       examples/ 에 시연 계약 둘.  파일 수는 그대로
   cmd/iapadapter       비테스트 소스 무변경
   cmd/runctl           비테스트 소스 무변경
```

**안 잰 자리의 알려진 낡음** — 두 문서에 관측값으로만 적고 본문은 안 고쳤다.
`internal/api` 라우트 15 -> 17 · `internal/` 패키지 10 -> 12 (`panel` · `proc`) ·
Go 파일 143 -> 190. 그 자리를 이 회차가 안 만진다.
