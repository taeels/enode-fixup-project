// landing.js 는 랜딩 화면의 유일한 상호작용 — Guest Login 클릭 —
// 을 처리한다.
//
// enterGuest() 가 카드뉴스를 건너뛸지 정하는 유일한 판정 지점이다
// (aidlc-docs/construction/cardnews-guest-login/functional-design/
// business-rules.md BR-1). 카드뉴스 화면 자신은 이 판정을 하지 않는다.

(function () {
  function enterGuest() {
    if (guest.hasOnboarded()) {
      window.location.href = "/ui/demo/";
    } else {
      window.location.href = "/ui/cardnews/";
    }
  }

  document
    .querySelector('[data-testid="landing-guest-login-button"]')
    .addEventListener("click", enterGuest);
})();
