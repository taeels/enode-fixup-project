// demo.js 는 자리표시 문구 한 줄뿐이다 — 데모 현황판의 실제 내용은
// 이 유닛의 스코프가 아니다 (enode-features.md 3.4.1 범위 경계).

(function () {
  document.getElementById("guest-badge").textContent =
    "Guest — " + guest.guestName() + " 로 둘러보는 중";
})();
