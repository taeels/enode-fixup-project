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
  - [x] 업데이트 2026-09-09 — 이미지/영상을 enode-img/card1~5 로 교체 · 카드뉴스 5 목업 신설
    (`.pen` 에 카드 5 프레임 · 모든 카드 5점·N/5 · 카드 4 버튼 다음). 상세는 update-2026-09-08.md
    - .pen 파일은 진행자가 pen.dev 에서 저장한 뒤 브랜치에 실림(design 은 진행자 몫)
  - [ ] 다음 업데이트는 update-2026-09-08.md 에 새 항목으로 이어 쌓는다
- [x] panel 제어판 — CP4 · 의존 obs · drain (PR #9·#10 병합 · main)
  - internal/proc 추출 · internal/panel · enodectl serve · cmd/enode panel · 경계 검사 테스트
  - **완료 조건에 조작 넷(status·start·stop·logs) + drain 걸기·모드·풀기 전부 나열**
  - 심볼 상한(enodectl.exe net/http ≤50 · crypto/tls ≤10) 재측정 · 커버리지 80%(internal/panel)
  - [x] ADR-068 닫힘(A · 진행자) — 데몬이 `<stem>.status.yaml` 에 Caps·At 쓰고 제어판이 읽음.
    `decisions.md` §2 「탐지 능력 읽기(ADR-068)」 행 추가
  - [x] Functional Design — 계획·ADR-068(A) 커밋됨(04aa712). FD 산출물 셋 냄
    (`panel/functional-design/` domain-entities · business-logic-model · business-rules) · 승인·병합됨
    - 정본 충돌 기록: proc 추출은 internal/proc(net/http 없음)로 간다 — enode-features 3.1.2 는
      internal/panel 이라 했으나 그러면 enodectl.exe 심볼 상한(CP4)이 깨진다. 유닛 정본 §5 가 이김
    - [x] FD 승인됨(진행자 "진행해") · 커밋 04aa712·64b5ae2·75ef4d3 · 인계 요약 1979743
  - [x] NFR Requirements — 승인됨(진행자 "다음 단계"). `panel/nfr-requirements/nfr-requirements.md` (커밋 992d8d7)
    심볼 상한 CP4 · 커버리지 80%(panel·proc) · 바인딩/토큰 거부 경로 · 새 의존 0
  - [x] NFR Design — 승인됨. `panel/nfr-design/nfr-design.md` (커밋 497a1f2)
  - [x] Infrastructure Design — 건너뜀(로컬 프로세스뿐·새 클라우드 자원 없음·결정 불필요)
  - [x] Code Generation — 구현 완료 (커밋 f958ecc·becb42a·d1ca5fc). 요약
    `panel/code/implementation-summary.md`. internal/proc · status.go · policy.go ·
    advertise.go · internal/panel · cmd/enode panel · cmd/enodectl serve · 테스트
    - 진행자 위임: 결정 필요 없으면 게이트에서 "다음 단계"로 기록하고 진행 (2026-09-09)
  - [x] CP4 게이트 코드 재료 초록 — 심볼 상한 13·1(≤50·10) · 커버리지 panel 87.8%·proc 88.9% ·
    경계 테스트 · 크로스 빌드 3종 · vet · glyphscan · gofmt. 버튼 전부(status·start·stop·logs +
    drain 걸기·모드·풀기) 냄
  - [x] PR #9 (main) — **병합됨** 2026-09-09T04:08:41Z (merge 182da58). panel 코드가 main 에 있다
  - [x] PR #10 (main) — **병합됨** 2026-09-09T04:27Z (merge 77f5a83). 제어판 시안 정렬(다크 2단·한국어).
    PR #9 가 이 커밋 앞에 병합돼 옛 기능판이 먼저 들어갔고, #10 이 시안본으로 바꿔 닫음
- [x] transcript — CP6 · 의존 panel · obs (PR #15 병합 · main)
  - enode 링 파일 tee(runner.go·claim.go) · panel 카드 · 지난 작업(runs 필터+record tar)
  - 링 파일 로직(머리·몸통·감김)은 FD (decisions §6.3)
  - [x] Functional Design — 계획·산출물 셋 냄(`construction/transcript/functional-design/`).
    막는 결정 없음(decisions §6 이 값 다 닫음). 링 형식·tee·비우기·지난 작업 확정
  - [x] NFR Requirements · NFR Design — 냄(`transcript/nfr-requirements`·`nfr-design`).
    단일 코드경로(빌드 태그 0) · 링 원자성/찢긴 읽기 · 커버리지 80% · 새 의존 0 · 심볼 상한 무영향
  - [x] Infrastructure Design — 건너뜀(로컬 파일뿐 · 클라우드 자원 없음)
  - [x] Code Generation — 구현 완료 (커밋 a416289). 요약 `transcript/code/implementation-summary.md`.
    링 파일 tee(runner·claim) · Ring(transcript.go) · 제어판 카드+지난 작업 · 화면은 enode-ux.pen 다크 토큰
  - [x] CP6 게이트 코드 재료 초록 — 빌드 태그 0 · 심볼 상한 무영향 · panel 커버리지 86.1% ·
    크로스 빌드 3종 · Mediator 변경 0 · 새 의존 0. 눈 검증은 사람·함대 몫
  - [x] PR #15 (main) — **병합됨** 2026-09-09T05:05Z (merge 1a15760). CP6 최종 눈 검증(도는 카드·봉인 기록)은 사람·함대 몫

## 열린 미정

- (닫힘 2026-09-09) ADR-068 — 제어판이 데몬 Capabilities{Caps,At} 를 읽는 계약.
  진행자가 A 로 결정: 데몬이 `<stem>.status.yaml` 에 Caps·At 쓰고 제어판이 읽음.
  `requirements/decisions.md` §2 「탐지 능력 읽기(ADR-068)」 행으로 닫음.

## 겹치는 자리 (조율)

```text
   internal/api/ui   card-news(nacl1119) vs ui(runixs) — 같은 패키지.  PR 직렬 병합 ·
                     커버리지 80% 공유
   internal/panel    panel·transcript 둘 다 nacl1119 — 한 손 안이라 접점 아님
   internal/enode    drain(shin-son 병합됨) vs panel(nacl1119 status 파일 쓰기 · ADR-068=A)
                     vs transcript(nacl1119 링 tee) — 서로 다른 파일/자리(광고 vs status vs 링 tee)
   drain 정책 파일    shin-son 의 drain 이 정하는 형식을 panel 의 drain 토글이 쓴다 — 맞춘다
   cmd/mediator      queue(shin-son)·ui(runixs)와 접점 — 진행자 직렬 병합
```

## 선행 · 공용

- 유닛 정본 `aidlc-docs/v1-run-dhseo/inception/application-design/unit-of-work.md`
- 완료본 `aidlc-docs/v1-run-dhseo-cardnews/` · 배정 `aidlc-docs/construction-roster.md`
- 팩 `requirements/` · RE `aidlc-docs/inception/reverse-engineering/`

## 다음

**담당(nacl1119) Construction 완료 (2026-09-09)** — 세 유닛이 전부 `main` 에 병합됐다.

```text
   card-news   초판(PR #3) + 미디어 교체·카드 5 목업(PR #18)
   panel       구현(PR #9) + 시안 정렬(PR #10) · CP4
   transcript  링 tee·카드·지난 작업(PR #15) · CP6
```

실질 개발 todo 없음. 남은 것은 사람·하드웨어 몫 — **CP4/CP6 최종 눈 검증**(화면
S3/S3b/S1/S5 · LED · 실제 Run 의 도는 트랜스크립트와 봉인 기록)은 진행자·함대가
닫는다. 코드·자동 게이트(심볼 상한 · 커버리지 80% · 경계 · 크로스 빌드 · glyphscan ·
CI)는 전부 초록. card-news 업데이트는 생기면 새 브랜치에서 작업 후 PR.

착수·설계 길잡이 `construction/plans/panel-transcript-handoff.md` 는 이력으로 남긴다.
