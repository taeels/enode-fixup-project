# Integration Test Instructions — CP8

정본은 `requirements/scene-gates.md` §3 CP8 이다. 여기는 그 절차를
반복하고, 이 세션에서 실제로 무엇을 확인했고 무엇을 못 했는지를
정직하게 남긴다.

## 절차

```bash
eval "$(scripts/testdb.sh)"        # Mediator 기동에 필요 (store 를 연다)
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

## 이 세션에서 실제로 확인한 것

이 실행 환경에는 브라우저도, `docker` 데몬도 없다(위 `unit-test-
instructions.md` 참조) — Mediator 프로세스를 실제로 띄워 브라우저로
여는 절차 1~8 을 **이 세션에서 직접 수행하지 못했다.** 대신 다음으로
같은 계약을 다른 도구로 검증했다:

```text
   절차 1 · 2   ui_test.go TestHandlerServesLanding — 응답 본문에
                landing-guest-login-button 과 landing-admin-token-input
                이 둘 다 있는지 문자열로 확인(시각적 분리 자체는 CSS
                이므로 이 테스트로 "레이아웃이 실제로 분리돼 보이는가"
                까지는 못 잰다 — 코드 리뷰로 landing.css 의 flex 레이아웃을
                확인했다)
   절차 3       landing.js 를 읽어 enterGuest() 가 hasOnboarded()==false
                일 때 /ui/cardnews/ 로 이동함을 코드로 확인
   절차 4       cardnews.css 의 transform/transition 규칙이 있음을 코드로
                확인. 애니메이션이 실제로 매끄럽게 재생되는지는 브라우저가
                있어야 확인된다 — 이 세션은 못 했다
   절차 5       cardnews.js 의 finish() 가 markOnboarded() 뒤
                location.href 를 바꿈을 코드로 확인. DOM 잔존 여부는
                전체 페이지 이동이 브라우저 문서를 통째로 교체하므로
                구조적으로 안 남는다(SPA 가 아니다) — 실측은 못 했다
   절차 6 · 7   landing.js 의 분기 로직을 코드로 확인. localStorage
                조작 후 재확인은 브라우저가 있어야 한다 — 이 세션은
                못 했다
   절차 8       demo/index.html 의 <a href="/ui/cardnews/"> 를 확인
```

**결론**: 정적 서빙 · 라우팅 · 헤더 · 콘텐츠 존재는 `go test` 로 확실히
검증했다. 애니메이션의 실제 체감 · 브라우저 저장소 조작 후의 실동작 ·
DOM 잔존 여부의 시각적 확인은 **이 세션에서 검증하지 못했다** — 사람이
브라우저와 Postgres 가 있는 환경에서 위 절차 1~8 을 직접 돌려 CP8 을
최종 확인해야 한다. 이 문서는 그 확인을 위한 재현 가능한 절차이지,
"확인했다"는 주장이 아니다.
