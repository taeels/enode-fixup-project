# Construction 상태 — nacl1119 (문태호)

담당 — **card-news**(온보딩 카드뉴스 · 게스트 로그인). **construction 이 이미
완료**됐고(회차 `v1-run-dhseo-cardnews`) 이후 업데이트를 이 담당이 계속 진다.
자기 브랜치에서 작업하고 PR 로 main 에 병합한다. 이후 산출물은 이 디렉터리
`aidlc-docs/nacl1119/` 아래.

## 상태

- [x] card-news 초판 — CP8 (construction 완료 · `v1-run-dhseo-cardnews` 회차)
  - 코드 `internal/api/ui/`(landing · guest · cardnews · demo 정적) · api.go `/ui/` 등록
  - 시안 `design/enode-cardnews.pen` · 완료본 문서 `aidlc-docs/v1-run-dhseo-cardnews/`
- [ ] 이후 업데이트 — 여기(`aidlc-docs/nacl1119/`)에 쌓는다

## 겹치는 자리 (조율)

card-news 의 `internal/api/ui/` 는 runixs 의 **ui 유닛과 같은 패키지**다. 온보딩
카드뉴스·게스트 로그인 정적은 nacl1119 가, 현황판·데모 모드 화면은 runixs 가 진다 —
같은 패키지라 PR 을 직렬로 병합하고 커버리지 80%(internal/api/ui)를 공유한다.

## 선행 · 공용

- 완료본 `aidlc-docs/v1-run-dhseo-cardnews/` · 배정 `aidlc-docs/construction-roster.md`
- 팩 `requirements/` · RE `aidlc-docs/inception/reverse-engineering/`

## 다음

업데이트가 생기면 이 브랜치에서 작업 후 PR. ui(runixs)와 같은 패키지라 접점 조율.
