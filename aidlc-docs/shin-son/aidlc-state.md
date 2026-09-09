# Construction 상태 — shin-son (손신)

담당 유닛 — **queue**(W1 · CP2) · **drain**(W2 · CP3). 임계 경로 앞머리(obs 뒤
queue·drain)를 진다. 자기 브랜치에서 작업하고 PR 로 main 에 병합한다(게이트 초록 뒤).
산출물은 이 디렉터리 `aidlc-docs/shin-son/` 아래. (panel·transcript 는 nacl1119 로
재배정됨 — 2026-09-08.)

## 진행 (Stage Progress)

- **Current Phase**: CONSTRUCTION
- **Current Unit**: queue (브랜치 `unit/queue` · 2026-09-08T13:55:15Z 착수 · 2026-09-09 에 `main` `0a159a4`(obs 병합) 위로 옮김)
- **Current Stage**: Build and Test — CP0 통과(2026-09-09T00:47:40Z) · CP2 진행 중(실제 노드 필요)
  (`construction/plans/queue-code-generation-plan.md`)
  (`construction/plans/queue-nfr-requirements-plan.md`)
  (`construction/plans/queue-functional-design-plan.md`)
- **Last Completed**: queue Code Generation (2026-09-09T00:36:11Z)
- **Extension Configuration**: `decisions.md` §1 이 닫음 — security-baseline 켬 ·
  resiliency-baseline 끔 · property-based-testing 끔. 취급은 `decisions.md` §3
- **Blockers**: 없음. Postgres.app 18.6 이 55434 에 섰다 (DSN 은 docs/testdb-setup.md 의 한 줄).

### queue
- [x] Functional Design — 승인 2026-09-08T17:37:00Z (Q1=A · Q2=A · 산출물 `construction/queue/functional-design/`)
- [x] NFR Requirements — 승인 2026-09-09T00:11:46Z (답 A·A·A · `construction/queue/nfr-requirements/`)
- [x] NFR Design — SKIP (회차 계획 · 어떻게는 Code Generation 계획이 진다)
- [x] Infrastructure Design — SKIP (새 인프라 0 · constraints §6)
- [x] Code Generation — 승인 2026-09-09T00:36:11Z (시험 미실행 상태 · `construction/queue/code/code-summary.md`)
- [ ] CP0 회귀 통과 (2026-09-09T00:47:40Z) · CP2 게이트 · PR to main

## 유닛 (의존 순)

- [ ] queue 대기열 — CP2 · 의존 obs
  - QUEUED · CreateQueuedRun · WakeQueued(ctx,tx) 여섯 지점 · WakeQueuedNow ·
    DrainingNodes · submit 202 분기 · mediator 기동 wake
- [ ] drain — CP3 · 의존 queue
  - enode 정책 파일·광고 경로 · postResult at-boundary Cancel(drain:<node_id>) + WakeQueued
  - 정책 파일 위치·형식·enum 과 at-boundary tx 경계는 FD (decisions §1)

## 선행 · 공용

- 유닛 정본 `aidlc-docs/v1-run-dhseo/inception/application-design/unit-of-work.md`
- 배정 `aidlc-docs/construction-roster.md` · 팩 `requirements/` · RE `aidlc-docs/inception/reverse-engineering/`

## 다음

**queue Functional Design 부터** (obs 병합 뒤 W1). 접점 `internal/store` ·
`cmd/mediator/main.go`(기동 wake)는 진행자 직렬 병합. drain 은 queue 병합 뒤 W2 ·
`internal/enode`(광고·정책)를 만지고 postResult 취소 경로를 넓힌다. drain 이 넘겨받는
자리는 nacl1119 의 panel(제어판 drain 토글이 쓰는 정책 파일)이다 — 형식을 맞춘다.
