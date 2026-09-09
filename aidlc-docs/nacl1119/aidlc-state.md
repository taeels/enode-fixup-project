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
  - [x] ADR-068 닫힘(A · 진행자) — 데몬이 `<stem>.status.yaml` 에 Caps·At 쓰고 제어판이 읽음.
    `decisions.md` §2 「탐지 능력 읽기(ADR-068)」 행 추가
  - [ ] Functional Design — 계획·ADR-068(A) 커밋됨(04aa712). FD 산출물 셋 냄
    (`panel/functional-design/` domain-entities · business-logic-model · business-rules) · 승인 대기
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
  - [x] PR — unit/panel push 됨 · PR #9 (main). 눈 검증(S3/S3b/S1/S5 · LED)은 진행자·하드웨어가 CP4 최종 확정 후 병합
- [ ] transcript — CP6 · 의존 panel · obs
  - enode 링 파일 tee(runner.go·claim.go) · panel 카드 · 지난 작업(runs 필터+record tar)
  - 링 파일 로직(머리·몸통·감김)은 FD (decisions §6.3)

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

panel 은 obs·drain 병합 뒤 W3 · transcript 는 panel 뒤 W4. panel Functional Design
전에 Capabilities 읽기 계약(ADR-068)을 진행자와 닫는다. card-news 업데이트는 생기면
이 브랜치에서 작업 후 PR.

**의존이 닫혔다 (2026-09-09)** — obs(PR #4) · drain(PR #7) 이 main 에 있다. panel
착수 조건이 섰고 `unit/panel` 을 `origin/main`(50af6cf)에서 땄다. 착수 길잡이는
`construction/plans/panel-transcript-handoff.md` 다 — 규약 · 문서 루트 · 의존이 남긴
표면 · 게이트 판정 기준 · 승인 지점이 거기 모여 있다. 정본과 어긋나면 정본이 이긴다.
이 유닛 둘은 AWS Bedrock 위의 Claude 가 이어받는다(전달 프롬프트는 같은 폴더의
`panel-transcript-handoff-prompt.md`).
