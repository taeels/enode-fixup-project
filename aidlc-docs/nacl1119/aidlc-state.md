# Construction 상태 — nacl1119 (문태호)

담당 — **card-news**(온보딩 카드뉴스·게스트 로그인 · construction 완료 · 계속 업데이트) ·
**panel**(W3 · CP4) · **transcript**(W4 · CP6). 자기 브랜치에서 작업하고 PR 로 main 에
병합한다(게이트 초록 뒤). 산출물은 이 디렉터리 `aidlc-docs/nacl1119/` 아래.
(panel·transcript 는 2026-09-08 에 shin-son 에서 재배정됨.)

## 유닛

- [x] card-news 초판 — CP8 (construction 완료 · 회차 `v1-run-dhseo-cardnews`)
  - 코드 `internal/api/ui/`(landing · guest · cardnews · demo 정적) · api.go `/ui/` 등록
  - 시안 `design/enode-cardnews.pen` · 완료본 문서 `aidlc-docs/v1-run-dhseo-cardnews/`
- [x] card-news 업데이트 2026-09-08 — CP8 재검증 초록. 상세는
  `aidlc-docs/nacl1119/construction/card-news/update-2026-09-08.md`
  - 카드 1~4 실사 일러스트, 카드 4 보드 사진 추가, 카드 5(영상) 신설
  - [ ] 다음 업데이트는 이 파일에 새 항목으로 이어 쌓는다
- [ ] panel 제어판 — CP4 · 의존 obs · drain
  - internal/proc 추출 · internal/panel · enodectl serve · cmd/enode panel · 경계 검사 테스트
  - **완료 조건에 조작 넷(status·start·stop·logs) + drain 걸기·모드·풀기 전부 나열**
  - 심볼 상한(enodectl.exe net/http ≤50 · crypto/tls ≤10) 재측정 · 커버리지 80%(internal/panel)
- [ ] transcript — CP6 · 의존 panel · obs
  - enode 링 파일 tee(runner.go·claim.go) · panel 카드 · 지난 작업(runs 필터+record tar)
  - 링 파일 로직(머리·몸통·감김)은 FD (decisions §6.3)

## 열린 미정

- panel 착수 전 — 제어판이 데몬 Capabilities{Caps,At} 를 읽는 계약(ADR-068).
  진행자가 `requirements/decisions.md` 에 행을 더해 닫는다.

## 겹치는 자리 (조율)

```text
   internal/api/ui   card-news(nacl1119) vs ui(runixs) — 같은 패키지.  PR 직렬 병합 ·
                     커버리지 80% 공유
   internal/panel    panel·transcript 둘 다 nacl1119 — 한 손 안이라 접점 아님
   internal/enode    drain(shin-son) vs transcript(nacl1119) — 다른 파일(광고 vs 링 tee)
   drain 정책 파일    shin-son 의 drain 이 정하는 형식을 panel 의 drain 토글이 쓴다 — 맞춘다
   cmd/mediator      queue(shin-son)·ui(runixs)와 접점 — 진행자 직렬 병합
```

## 선행 · 공용

- 유닛 정본 `aidlc-docs/v1-run-dhseo/inception/application-design/unit-of-work.md`
- 완료본 `aidlc-docs/v1-run-dhseo-cardnews/` · 배정 `aidlc-docs/construction-roster.md`
- 팩 `requirements/` · RE `aidlc-docs/inception/reverse-engineering/`

## 다음

panel 은 obs·drain 병합 뒤 W3 · transcript 는 panel 뒤 W4. panel Functional Design
전에 Capabilities 읽기 계약(ADR-068)을 진행자와 닫는다. card-news 업데이트는 생기면
이 브랜치에서 작업 후 PR.
