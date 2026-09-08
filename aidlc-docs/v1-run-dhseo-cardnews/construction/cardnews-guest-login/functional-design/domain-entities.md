# Domain Entities — cardnews-guest-login

이 유닛은 함대 도메인(노드 · 임대 · Run)을 하나도 다루지 않는다. 아래
둘은 **영속 도메인 모델이 아니라 브라우저 로컬 저장소 안의 값**이다 —
서버는 존재를 모른다.

## GuestSession (클라이언트 로컬 전용)

```text
   필드              타입      비고
   ────────────────  ───────  ──────────────────────────────
   id                string   "guest-" + 형용사 + "-" + 명사.
                              guestName() 가 처음 호출될 때 생성
   onboardedAt       boolean  저장소 키 enode.guest.onboarded 의 존재
                              여부로만 판정한다 — 값의 내용은 안 본다
                              ("1" 이든 다른 문자열이든 true)
```

- 저장 위치: `window.localStorage`, 키 `enode.guest.id` /
  `enode.guest.onboarded`
- 생성 규칙: `guestName()` 최초 호출 시 형용사 목록과 명사 목록에서
  각각 하나를 고르고 `-` 로 잇는다(예 `guest-vivid-otter`). 저장에
  성공하면 이후 호출은 저장된 값을 그대로 돌려준다(재생성하지 않는다)
- 수명: 브라우저 저장소가 지워질 때까지. 서버 세션이 아니므로 만료가
  없다 — TTL 개념 자체가 없다(`ADR-015` 의 principal 과 달리 아예
  서버 인지가 없다)
- **인증 자격이 아니다.** 어떤 요청 헤더 · 쿼리 · 바디에도 실리지 않는다

## Card (정적 콘텐츠, 영속 아님)

```text
   필드     타입      비고
   ───────  ───────  ──────────────────────────
   title    string   카드 제목
   body     string   카드 본문(한두 문장)
```

- 저장 위치: `static/cardnews/cardnews.js` 안의 상수 배열. 데이터베이스도
  API 도 거치지 않는다
- 개수: 4(고정). `business-rules.md` BR-4
- 식별자 없음 — 배열 인덱스가 곧 순서이자 식별이다(재정렬은 배열 순서를
  바꾸는 코드 변경으로만 가능하다)
