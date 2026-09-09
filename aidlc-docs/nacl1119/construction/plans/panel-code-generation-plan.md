# panel — Code Generation 계획 (Part 1)

FD·NFR 설계가 값을 다 정했으므로 이 계획은 기계적이다 — 결정 사항이 없어
Part 2(구현)로 이어 진행한다. 정본은 이 유닛의 functional-design/ ·
nfr-design/ · `unit-of-work.md` §5 다.

---

## 낼 것 · 순서 (의존 순)

- [x] 1. `internal/proc` 신규 — `cmd/enodectl/proc_*.go` 에서 `ProcessAlive` ·
      `SignalStop` · `OwnsConfig` 를 내린다(빌드 태그 짝). net/http 없음.
      `cmd/enodectl` 은 이 패키지를 부르게 고친다(`detachAttr`·`signalKill`·
      `exeSuffix` 는 남긴다)
- [x] 2. `internal/enode/status.go` 신규 — `Status{Caps, At}` · `StatusPath` ·
      `WriteStatus(configPath, Capabilities)` · `ReadStatus(configPath)`.
      tmp+rename 원자성 · 0600
- [x] 3. `internal/enode/policy.go` — `Policy` 에 `PanelToken` (`panel_token`) additive
- [x] 4. `internal/enode/advertise.go` — `snap := a.Caps()` 뒤에 At 바뀔 때만
      `WriteStatus` 호출(guarded · 실패는 경고만). `Advertiser` 에 `lastStatusAt` 자리
- [x] 5. `internal/panel` 신규 — `Config` · `New` · `Server` · `Handler`.
      화면 값(신원·탐지 능력·현재 작업·drain·프로세스) 읽기 · drain 토글 쓰기 ·
      프로세스 제어(status·start·stop·logs) · stop 은 cancel 먼저 · `isLoopback` 거부 경로
- [x] 6. `cmd/enode` — `panel` 하위명령(os.Args[1]=="panel") 이 `internal/panel` 을
      net/http 로 띄운다
- [x] 7. `cmd/enodectl serve <name>` — `oneName` 으로 설정 경로 풀고 `enodeBin` 을
      exec 위임(setup.go 본뜸). `internal/panel` 을 임포트하지 않는다
- [x] 8. 테스트 — `internal/proc`(플랫폼 짝) · `internal/panel`(httptest 가짜
      Mediator · TempDir) · 임포트 경계 검사 테스트. 커버리지 80%
- [x] 9. 검증 — go build ./... (크로스 셋) · go vet · glyphscan · enodectl.exe 심볼
      상한 재측정 · 커버리지 80% · 포맷

---

## 완료 조건 (게이트 CP4)

business-rules §6 의 버튼 표를 코드가 전부 낸다 — status·start·stop·logs +
drain 걸기·모드·풀기 · S3/S3b/S1/S5. 경계 테스트 초록 · 심볼 상한 안 ·
`internal/panel`·`internal/proc` 커버리지 80%.

---

## 안 하는 것

트랜스크립트 카드(transcript 유닛) · Capabilities 광고 변경(ADR-068=A) ·
LAN 토큰 켜기(거부 경로만) · 화면 레이아웃(시안·진행자).
