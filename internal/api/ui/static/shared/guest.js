// guest.js 는 랜딩 · 카드뉴스 · 데모 세 화면이 공유하는 유일한 코드다.
// 클라이언트 로컬 저장소만 읽고 쓴다 — 네트워크 호출을 하지 않는다.
// (aidlc-docs/construction/cardnews-guest-login/functional-design/
// business-rules.md BR-5, BR-6)

var guest = (function () {
  var ONBOARDED_KEY = "enode.guest.onboarded";
  var ID_KEY = "enode.guest.id";

  var adjectives = [
    "vivid", "quiet", "brisk", "gentle", "swift",
    "steady", "keen", "bold", "calm", "bright"
  ];
  var nouns = [
    "otter", "falcon", "maple", "harbor", "ember",
    "cedar", "heron", "meadow", "ridge", "comet"
  ];

  var memoryName = null;

  function randomName() {
    var a = adjectives[Math.floor(Math.random() * adjectives.length)];
    var n = nouns[Math.floor(Math.random() * nouns.length)];
    return "guest-" + a + "-" + n;
  }

  function hasOnboarded() {
    try {
      return window.localStorage.getItem(ONBOARDED_KEY) !== null;
    } catch (e) {
      return false;
    }
  }

  function markOnboarded() {
    try {
      window.localStorage.setItem(ONBOARDED_KEY, "1");
    } catch (e) {
      // 저장에 실패해도 다음에 카드가 다시 뜨는 것뿐이다 — 안전한 실패.
    }
  }

  function guestName() {
    try {
      var stored = window.localStorage.getItem(ID_KEY);
      if (stored) {
        return stored;
      }
      var name = randomName();
      window.localStorage.setItem(ID_KEY, name);
      return name;
    } catch (e) {
      if (!memoryName) {
        memoryName = randomName();
      }
      return memoryName;
    }
  }

  return {
    hasOnboarded: hasOnboarded,
    markOnboarded: markOnboarded,
    guestName: guestName
  };
})();
