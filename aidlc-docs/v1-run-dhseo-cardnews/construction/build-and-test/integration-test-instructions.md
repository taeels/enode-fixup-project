# Integration Test Instructions — CP8

정본은 `requirements/scene-gates.md` §3 CP8 이다. 여기는 그 절차를
반복하고, 이 세션에서 실제로 확인한 것을 남긴다.

## 절차

```bash
eval "$(scripts/testdb.sh)"        # docker 가 없으면 로컬 Postgres 로 대체 가능
export M=http://<host>:8080
go run ./cmd/mediator --config <설정 파일>
```

1. 브라우저 사생활 창(로컬 저장소가 비어 있다)으로 `$M/ui/` 를 연다
2. Guest Login 버튼이 관리자 토큰 입력과 시각적으로 분리돼 보이는지 확인
3. Guest Login 을 누른다 -> 카드뉴스가 뜬다. 주소창이 `/ui/cardnews/`
4. 카드를 넘긴다(다음 버튼 · 화살표 키) -> 슬라이드 애니메이션이 보인다
5. 마지막 카드에서 "현황판 보기" 를 누른다 -> `/ui/demo/` 로 이동.
   개발자 도구로 DOM 에 카드뉴스 요소가 안 남았는지 확인
6. 탭을 새로고침하고 랜딩에서 Guest Login 을 다시 누른다 -> 카드뉴스
   없이 곧장 `/ui/demo/` 로 간다
7. 브라우저 저장소에서 `enode.guest.onboarded` 를 지운다 -> 다시
   누르면 카드뉴스가 다시 뜬다
8. 데모 화면의 "카드뉴스 다시 보기" 링크를 누른다 -> 카드뉴스가 뜬다

## 이 세션에서 실제로 확인했다 (2026-09-08)

처음에는 이 실행 환경에 `docker` 데몬도 브라우저도 없어 절차 1~8 을
직접 못 돌린다고 적었었다. 이후 대안을 찾아 실제로 돌렸다 —

- `docker` 데몬은 없지만 **로컬 PostgreSQL 16 이 이미 설치돼 있어**
  `service postgresql start` 로 띄우고 `enode`/`enode_test` 역할과
  DB 를 만들었다(`scripts/testdb.sh` 와 같은 자격증명, docker 대신
  네이티브 서비스)
- 브라우저는 없지만 **이 환경에 Playwright 와 Chromium 이 미리 설치돼
  있다**(`/opt/pw-browsers`, 전역 npm 패키지 `playwright@1.56.1`).
  실제 Mediator 프로세스(`go run ./cmd/mediator --config ...`, 방금
  만든 Postgres 를 가리킴)를 `127.0.0.1:18080` 에 띄우고 headless
  Chromium 으로 절차 1~8 을 전부 실행했다

실행 스크립트는 이 문서와 함께 남기지 않는다(임시 검증용, `/tmp` 산출물)
— 대신 결과를 아래에 적는다. 재현하려면 위 §절차의 명령대로 Mediator 를
띄우고, Playwright(`chromium.launch()`)로 `data-testid` 셀렉터들을
그대로 조작하면 된다.

## 확인 결과 (18개 확인, 전부 통과)

```text
   랜딩            Guest Login 버튼 · 관리자 토큰 placeholder 둘 다 보인다
                   placeholder 는 disabled — 실제 제출 동작이 없다
   첫 방문          Guest Login 클릭 -> /ui/cardnews/ 로 이동, 1/4 카드부터 시작
   카드 넘김        다음 버튼으로 4장 전부 이동. 마지막 카드에서 버튼 라벨이
                   "현황판 보기" 로 바뀐다
   종료            마지막 카드에서 누르면 /ui/demo/ 로 이동하고, DOM 에
                   .card 요소가 0개 남는다(document.querySelectorAll 로 확인)
   Guest 배지       데모 화면에 "Guest — guest-<형용사>-<명사>" 문구가 보인다
   로컬 저장소      enode.guest.onboarded 가 "1" 로 저장된다
   재방문           같은 브라우저 컨텍스트(저장소 유지)에서 Guest Login 을
                   다시 누르면 카드뉴스를 거치지 않고 곧장 /ui/demo/ 로 간다
   재열람           데모의 "카드뉴스 다시 보기" 링크를 누르면
                   onboarded="1" 인데도 카드가 1/4 부터 다시 보인다
                   (business-logic-model.md 의 설계대로 — 카드뉴스는
                   플래그를 안 읽는다)
   닫기 버튼        어느 카드에서 눌러도 /ui/demo/ 로 이동한다
   키보드           ArrowRight/ArrowLeft 로 카드 이동, Escape 로 닫기 —
                   전부 실제 keydown 이벤트로 확인
   보안 헤더        실제 네트워크 응답(Go httptest 가 아니라 진짜 TCP 요청)
                   에서 Content-Security-Policy · X-Frame-Options 확인
```

스크린샷 다섯 장(랜딩 · 카드 1/4 · 카드 4/4 · 데모 · 재열람)을 이
세션에서 사용자에게 전달했다.

## 결론

CP8 의 눈으로 보는 항목(`scene-gates.md` §2.1)을 실제 브라우저 ·
실제 Mediator 프로세스 · 실제 PostgreSQL 로 확인했다. 관리자 흐름
회귀(US-5)는 이 유닛이 `internal/api` 의 기존 라우트를 안 건드렸다는
사실(코드 리뷰 + `unit-test-instructions.md` 의 회귀 테스트 결과)로
확인된다 — 관리자 로그인 자체가 아직 구현되지 않았으므로(placeholder)
"기존과 동일하게 동작"은 "아직 아무 동작도 없다"는 뜻이고, 그 상태를
이 유닛이 바꾸지 않았다.

## 재검증 (로컬 세션, 2026-09-08, 일러스트 · 영상 카드 추가 뒤)

병합 뒤에 카드 1~4 에 실제 사진 일러스트를, 카드 4 에 보드 사진을
하나 더, 새 5번째 카드로 `story2.mp4` 영상을 더했다. 이 로컬
환경에는 PostgreSQL 도 Playwright 도 없다 — 대신 `scene-gates.md`
CP8 자신이 "함대도 Mediator 토큰도 필요 없다. 정적 파일 서버
(mediator 나 **standalone 바이너리**)만 띄우면 된다" 고 명시하므로,
`internal/api/ui.Handler()` 만 올린 standalone 바이너리 + 실제
브라우저(Claude Code 의 Browser 도구)로 §3 CP8 절차를 그대로 다시
돌렸다.

```text
   사생활 창 상태     localStorage.clear() 로 재현.  랜딩 재확인
   버튼 분리          Guest Login 카드와 관리자 토큰 카드가 시각적으로
                     분리된 두 섹션으로 보인다
   첫 방문            Guest Login 클릭 -> /ui/cardnews/ 로 이동, 1/5 부터 시작
   카드 넘김          다섯 장 전부 이동 확인.  카드 1~4 는 일러스트,
                     카드 4 는 사진 둘(폰 + 보드), 카드 5 는 영상
                     (story2.mp4, 자동재생 · 음소거 · 반복)
   마지막 카드        버튼 라벨이 "현황판 보기" 로 바뀐다(더 이상
                     카드 4 가 아니라 카드 5 에서)
   종료              /ui/demo/ 로 이동.  document.querySelectorAll(".card")
                     가 0, #card-stage 가 없음 — DOM 무잔존 확인
   재방문             같은 컨텍스트에서 Guest Login 을 다시 누르면
                     카드뉴스 없이 곧장 /ui/demo/ 로 간다
   재노출             localStorage.removeItem("enode.guest.onboarded")
                     뒤 다시 누르면 카드가 다시 뜬다(다섯 장 전부 DOM 에)
```

여섯 항목 전부 통과했다. 이번 재검증은 CP8 의 화면 항목만 다시 본다
— 보안 헤더 · 관리자 흐름 회귀는 위 최초 검증(18개 확인)에서 이미
닫혔고 이번 변경이 그 경로를 건드리지 않았으므로 다시 돌 이유가
없다.
