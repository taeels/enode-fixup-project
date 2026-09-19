// guest.js 는 랜딩 · 카드뉴스 · 데모 세 화면이 공유하는 유일한 코드다.
// 클라이언트 로컬 저장소만 읽고 쓴다 — 네트워크 호출을 하지 않는다.
// (aidlc-docs/v1-run-dhseo-cardnews/construction/cardnews-guest-login/
// functional-design/business-rules.md BR-5, BR-6)

var guest = (function () {
  var ONBOARDED_KEY = "enode.guest.onboarded";
  var ID_KEY = "enode.guest.id";

  var adjectives = [
    "선명한", "조용한", "경쾌한", "다정한", "날쌘",
    "꾸준한", "슬기로운", "용감한", "차분한", "밝은",
    "포근한", "잔잔한", "맑은", "든든한", "온화한",
    "씩씩한", "반가운", "유쾌한", "따스한", "여유로운"
  ];
  var nouns = [
    "수달", "매", "단풍", "항구", "불씨",
    "삼나무", "왜가리", "들판", "산등성이", "혜성",
    "여우", "두루미", "소나무", "별빛", "갈대",
    "시냇물", "구름", "산새", "노을", "바람"
  ];
  var previousAdjectives = ["vivid", "quiet", "brisk", "gentle", "swift", "steady", "keen", "bold", "calm", "bright"];
  var previousNouns = ["otter", "falcon", "maple", "harbor", "ember", "cedar", "heron", "meadow", "ridge", "comet"];

  var memoryName = null;

  function randomName() {
    var a = adjectives[Math.floor(Math.random() * adjectives.length)];
    var n = nouns[Math.floor(Math.random() * nouns.length)];
    return a + " " + n;
  }

  // 영문 이름의 이전은 저장이 막혀도 같은 결과다. 표시 이름은 인증 신원이 아니다.
  function migrateName(name) {
    var parts = name.split("-");
    var a = previousAdjectives.indexOf(parts[1]), n = previousNouns.indexOf(parts[2]);
    var hash = 0;
    for (var i = 0; i < name.length; i++) hash = (hash * 31 + name.charCodeAt(i)) >>> 0;
    return adjectives[a < 0 ? hash % adjectives.length : a] + " " + nouns[n < 0 ? Math.floor(hash / adjectives.length) % nouns.length : n];
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
      if (stored && /^[가-힣]{1,12} [가-힣]{1,12}$/.test(stored)) {
        return stored;
      }
      var name = stored && /^guest-[a-z]{1,24}-[a-z]{1,24}$/.test(stored) ? migrateName(stored) : memoryName || randomName();
      memoryName = name;
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
