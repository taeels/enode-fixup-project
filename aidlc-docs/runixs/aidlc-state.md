# Construction 상태 — runixs (김태완)

담당 유닛 — **ui**(W1 실 모드 · W3 데모 모드) · **demo-back**(W2 · CP9). 자기
브랜치에서 작업하고 PR 로 main 에 병합한다(게이트 초록 뒤). 산출물은 이 디렉터리
`aidlc-docs/runixs/` 아래.

## 유닛

- [ ] ui 현황판 UI — 한 유닛 두 모드
  - **실 함대 모드**(CP1·CP2 화면) — obs 만 딛는다 -> **W1 착수·완료**.
    S0·S0b·S1·S1b·S2 · 되묻기 카드 표시(CP7) · 읽기 전용
  - **데모 모드**(CP8·CP9·CP11 화면) — demo-back 제출 라우트를 딛는다 -> **W3 완료**.
    게스트 로그인 · 3D(S6·S7) · 새 작업 모달 · 웹캠 floating · 투어 · RUN 카드 submitter
  - **완료 조건에 화면 조작 전부 나열**(§2.1·§5.3). internal/api/ui 단독 소유 ·
    store 임포트 금지 · 커버리지 80%
- [ ] demo-back — CP9 서버측 · 의존 obs · queue
  - internal/api/demo.go — allow-list · 서버측 토큰 주입 · submitter 쓰기(Guest 로그인 이름)
  - api.go 는 등록 줄만 · 데모 라우트는 데모 모드 config 에서만 등록 권장

## ui 와 demo-back 의 순서

제출 라우트 계약(경로·payload)을 먼저 정하면 **병렬로 짠다**. demo-back(W2)이
병합된 뒤 ui 데모 모드의 CP9 end-to-end 를 검증한다(W3).

## 열린 미정

- ui — sandbox 표시 출처(decisions §8.5). 있는 값을 읽는다 · FD 가 정한다.

## 선행 · 공용

- 유닛 정본 `aidlc-docs/v1-run-dhseo/inception/application-design/unit-of-work.md`
- 배정 `aidlc-docs/construction-roster.md` · 팩 `requirements/` · RE `aidlc-docs/inception/reverse-engineering/`
- 데모 시안 참고 — `design/enode-demo.pen` · export `design/exports/D1~D5`
- 이미 있는 정적 코드 — `internal/api/ui/`(cardnews 회차가 낸 landing·guest·demo·cardnews)

## 다음

**ui 실 모드 Functional Design 부터** (obs 병합 뒤 W1). demo-back 은 queue 병합 뒤 W2.
