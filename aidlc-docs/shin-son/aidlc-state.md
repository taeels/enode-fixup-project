# Construction 상태 — shin-son (손신)

담당 유닛 — **queue**(W1 · CP2) · **drain**(W2 · CP3) · **panel**(W3 · CP4) ·
**transcript**(W4 · CP6). 임계 경로 등뼈를 진다. 자기 브랜치에서 작업하고 PR 로
main 에 병합한다(게이트 초록 뒤). 산출물은 이 디렉터리 `aidlc-docs/shin-son/` 아래.

## 유닛 (의존 순)

- [ ] queue 대기열 — CP2 · 의존 obs
  - QUEUED · CreateQueuedRun · WakeQueued(ctx,tx) 여섯 지점 · WakeQueuedNow ·
    DrainingNodes · submit 202 분기 · mediator 기동 wake
- [ ] drain — CP3 · 의존 queue
  - enode 정책 파일·광고 경로 · postResult at-boundary Cancel(drain:<node_id>) + WakeQueued
- [ ] panel 제어판 — CP4 · 의존 obs · drain
  - internal/proc 추출 · internal/panel · enodectl serve · cmd/enode panel · 경계 검사 테스트
  - **완료 조건에 조작 넷(status·start·stop·logs) + drain 걸기·모드·풀기 전부 나열**
- [ ] transcript — CP6 · 의존 panel · obs
  - enode 링 파일 tee(runner.go·claim.go) · panel 카드 · 지난 작업(runs 필터+record tar)

## 열린 미정

- panel 착수 전 — 제어판이 데몬 Capabilities{Caps,At} 를 읽는 계약(ADR-068).
  진행자가 `requirements/decisions.md` 에 행을 더해 닫는다.

## 선행 · 공용

- 유닛 정본 `aidlc-docs/v1-run-dhseo/inception/application-design/unit-of-work.md`
- 배정 `aidlc-docs/construction-roster.md` · 팩 `requirements/` · RE `aidlc-docs/inception/reverse-engineering/`

## 다음

**queue Functional Design 부터** (obs 병합 뒤 W1). 접점 `internal/store` ·
`internal/panel`(transcript 와) · `cmd/mediator/main.go` 는 진행자 직렬 병합.
