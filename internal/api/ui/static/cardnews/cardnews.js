// cardnews.js 는 카드 넉 장의 넘김 · 진행 표시 · 종료를 맡는다.
//
// 이 화면은 guest.hasOnboarded() 를 절대 읽지 않는다 — 로드되면
// 무조건 0번 카드부터 보인다. 재방문 시 카드를 건너뛰는 판정은
// landing.js 하나에만 있다 (business-logic-model.md 「카드뉴스 화면은
// onboarded 플래그를 읽지 않는다」).

(function () {
  var cards = document.querySelectorAll("#card-stage .card");
  var dots = document.querySelectorAll("#progress-dots .dot");
  var progressText = document.getElementById("progress-text");
  var prevButton = document.querySelector('[data-testid="cardnews-prev-button"]');
  var nextButton = document.querySelector('[data-testid="cardnews-next-button"]');
  var closeButton = document.querySelector('[data-testid="cardnews-close-button"]');

  var total = cards.length;
  var index = 0;

  function renderCard(i) {
    for (var c = 0; c < cards.length; c++) {
      cards[c].classList.remove("active", "exiting-left");
      if (c < i) {
        cards[c].classList.add("exiting-left");
      }
    }
    cards[i].classList.add("active");

    for (var d = 0; d < dots.length; d++) {
      dots[d].classList.toggle("active", d === i);
    }

    progressText.textContent = (i + 1) + " / " + total;
    prevButton.disabled = i === 0;
    nextButton.textContent = i === total - 1 ? "현황판 보기" : "다음";
  }

  function next() {
    if (index === total - 1) {
      finish();
      return;
    }
    index += 1;
    renderCard(index);
  }

  function prev() {
    if (index === 0) {
      return;
    }
    index -= 1;
    renderCard(index);
  }

  function finish() {
    guest.markOnboarded();
    window.location.href = "/ui/demo/";
  }

  nextButton.addEventListener("click", next);
  prevButton.addEventListener("click", prev);
  closeButton.addEventListener("click", finish);

  document.addEventListener("keydown", function (event) {
    if (event.key === "ArrowRight") {
      next();
    } else if (event.key === "ArrowLeft") {
      prev();
    } else if (event.key === "Escape") {
      finish();
    }
  });

  renderCard(index);
})();
