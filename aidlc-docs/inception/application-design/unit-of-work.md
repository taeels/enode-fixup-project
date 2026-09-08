# Unit of Work — v1-run-dhseo-cardnews

이 실행의 스코프(3.4.1)는 유닛 하나로 끝난다. 기존 일곱 기능의 유닛
분해는 이 실행이 안 낸다 — 다른 세션/유닛의 몫이다(`aidlc-docs/inception/
requirements/requirements.md` §2.1).

## 유닛 — `cardnews-guest-login`

**책임**: 온보딩 카드뉴스 + Guest Login 진입점 전체 — 랜딩의 Guest Login
버튼, 카드 시퀀스 화면, 데모 현황판 placeholder, 그리고 그 셋을 잇는
클라이언트 쪽 라우팅. `internal/api/ui` 패키지와 `internal/api/api.go`
의 등록 줄 하나를 포함한다.

**이 유닛이 무엇을 완료로 삼는가**:

- Application Design 이 정한 컴포넌트 다섯(`UIHandler` ·
  `LandingPage` · `CardNewsScreen` · `DemoPlaceholder` · `GuestIdentity`)
  이 전부 코드로 있다
- `scene-gates.md` CP8 이 초록이다 — 5절의 눈으로 보는 항목(랜딩 분리 ·
  카드뉴스 독립 · 종료 시 DOM 유출 없음 · 재방문 생략 · 재열람 경로)을
  전부 포함한 완료 조건이다. **화면이 있는 유닛의 완료 조건은 그 화면의
  버튼을 전부 나열한다**(`constraints.md`) — 이 유닛의 화면 버튼은
  Guest Login · 카드뉴스의 다음/이전/닫기/진행 점 · 데모의 재열람
  링크다. 전부 CP8 의 확인 항목에 있다
- `internal/api/ui` 패키지가 패키지별 커버리지 하한 80% 를 넘는다
  (`decisions.md` 2절의 값을 3.4 표면에도 그대로 적용, 8.2)
- `go build ./... ` · `go vet ./...` · `gofmt` · glyphscan(장식 문자
  상한 0)이 이 유닛이 만든 파일에서 모두 통과한다
- `internal/api` 의 기존 핸들러 15개에 diff 가 없다(회귀 없음, US-5)

**어느 요구사항 절에서 왔는가**: `requirements/enode-features.md`
§3.4.1 전체. `requirements/decisions.md` §8. `requirements/scene-
gates.md` CP8. User Stories US-1 ~ US-5.

**다른 유닛에 의존하는가**: **없다.** 함대 상태를 안 읽고 안 쓰므로
3.1.x · 3.2.x · 3.3.x 어느 유닛의 산출물도 선행 조건이 아니다.
`scene-gates.md` CP8 행의 「먼저 서는 기능」이 비어 있는 것과 같은
근거다. 유일한 전제는 CP0(기동이 안 깨졌다)이고, 그것도 이 유닛이
`internal/api/ui` 를 새로 만드는 행위 자체와는 무관하다(기존 코드를
안 건드리므로 CP0 를 깨뜨릴 표면이 없다).

**다른 유닛이 이 유닛에 의존하는가**: 향후 3.1.1(중앙 현황판) 유닛이
`internal/api/ui` 패키지와 `GET /ui/` 등록 줄을 공유한다 — 새로
만드는 쪽이 아니라 이 유닛이 이미 세운 것 위에 자기 화면(S0 ~ S5)을
더하는 쪽이 된다(`decisions.md` 8.5). 강제 선행은 아니다 — 3.1.1 이
먼저 병합되면 그쪽이 패키지를 만들고 이 유닛이 나중에 랜딩/카드뉴스/
데모 디렉터리를 더하는 순서도 성립한다. 어느 쪽이 먼저든 파일 행렬
접점(패키지 자체 · 등록 줄)에서 사람이 직렬로 병합한다.
