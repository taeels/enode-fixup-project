# U5 `panel-live` — 불변식

```text
   유닛    panel-live (U5) · 웨이브 W-c
   답      1=A · 2=A · 3=A · 4=A · 5=C · 6=A · 7=A · 8=A
```

**세는 것으로 지킨다.** 각 줄에 무엇이 재는지를 적는다.

---

## 1. 라우트와 경계

```text
   R1   라우트가 안 는다.  internal/panel/panel.go 의 HandleFunc 가 10 그대로다
   R2   이 카드는 Mediator 를 안 탄다.  handleTranscript 에 s.client 가 0 번 나온다
   R3   panel 은 internal/transcript 를 임포트한다.  금지 표에 없다
        (boundary_test.go 의 여덟 줄이 그대로 초록이다)
   R4   handleRecord 를 이 유닛이 안 만진다.  U6 의 것이다 (파일 행렬 2절)
```

---

## 2. 응답이 무엇의 함수인가 — 캐시가 성립하는 근거

```text
   R5   응답에 **기간을 안 싣는다.**  시각(last_write)만 싣고 브라우저가 뺀다
        재는 것 — 응답 JSON 에 초 단위 수 필드가 0 이다
   R6   봉투는 (링 바이트, mtime, 링 경로) 의 순수 함수다.  시계를 안 읽는다
   R7   캐시 열쇠는 (generation, total, mtime) 셋이다.  둘로 줄이지 않는다 —
        링이 다시 만들어져 (0,0) 으로 돌아온 자리가 옛 항목에 맞는다
   R8   캐시는 한 칸이다.  링이 하나다
```

**R5 가 모순 검사가 막은 자리다** (business-logic-model 1.1). **재는 법이
「필드가 없다」인 것이 값이다** — 있으면 언젠가 누가 거기 값을 채운다.

---

## 3. 잘림 — NC-2

```text
   R9    truncated = Total > Capacity.  **엄격 부등호다**
         Total == Capacity 면 몸통이 정확히 찬 것이고 머리가 안 잘렸다
         (ReadRing 이 total <= capacity 에서 body[:total] 을 그대로 낸다)
   R10   truncated 를 Go 가 정해서 싣는다.  브라우저가 다시 계산하지 않는다 —
         규칙이 두 자리에 살면 갈린다
   R11   잃은 양은 Total - Capacity 다.  Result.Head 가 아니다 —
         Head 는 잘린 첫 줄의 바이트일 뿐이라 감긴 총량보다 훨씬 작다
```

---

## 4. 경과 — NC-1

```text
   R12   시각의 출처는 링 파일의 mtime 하나다.  제어판의 메모리도 브라우저의
         타임스탬프도 아니다 — 탭을 다시 열어도 제어판을 다시 띄워도 같은 값이다
   R13   단계 시작이 경과 0 이다.  Ring.Reset() 이 머리를 써 mtime 을 올린다
   R14   경과는 폴링과 **별개 타이머**로 1초마다 다시 그린다.
         폴링이 늦거나 죽어도 경과가 흐른다
```

---

## 5. 세 상태 — 5 = C

```text
   R15   「링이 없다」 · 「읽었다」 · 「폴링이 실패했다」 셋이 화면에서 안 합쳐진다
   R16   폴링이 실패하면 **마지막 값을 안 지운다.**  카드를 회색으로 두고
         값이 낡았음을 적는다.  빈 화면으로 떨어뜨리면 「단계가 끝났다」로 읽힌다
   R17   .catch 가 조용히 삼키는 자리가 0 이다.  오늘 page.go 에 셋 있다
         재는 법 — 이 카드의 경로에 빈 catch 가 없다
```

---

## 6. 그리기

```text
   R18   본문은 textContent 와 esc() 로만 그린다.  innerHTML 에 하네스 바이트가
         0 번 닿는다 (SECURITY-05 · FR-4).  **CSP 가 'unsafe-inline' 이라
         이 줄이 유일한 방어다** (7절)
   R19   펼침 상태는 Event.ID 로 든다.  Line 으로 안 든다 — 링이 감기면 밀린다
   R20   세대가 바뀌면 펼침 · 스크롤 · 원문 토글을 비운다
   R21   바닥에 있었을 때만 바닥을 따라간다.  **그리기 전에 잰다** —
         갈고 나서 재면 언제나 바닥이 아니다
   R22   그리는 종류는 transcript.Kind 일곱뿐이다.  모르는 것은 raw 로 온다
   R23   원문 토글은 같은 응답의 data 를 그린다.  두 번째 요청을 안 한다 —
         링이 감기므로 두 번째 읽기는 다른 창이다
```

**R21 이 오늘의 동작을 뒤집는다** — `page.go:267` 이 매 폴링마다 무조건 바닥으로
간다. **도는 동안 위로 올려 읽는 것이 지금 불가능하다.**

---

## 7. 보안 헤더 — 1 = A

```text
   R24   GET / 에 다섯을 건다.  넷은 ui.go:64-71 과 글자 그대로 같다
   R25   CSP 만 갈린다 —
         default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'
         **어긋남으로 적는다.**  requirements.md 5.5 는 「같은 다섯 줄」이라 적었다
   R26   갈린 값으로도 밖으로 나가는 길은 막힌다 —
         connect-src · img-src · font-src 가 default-src 를 물려받는다
   R27   이 페이지가 밖에서 가져오는 것이 0 이다 (data: 0 · 외부 URL 0 · @import 0).
         재는 법 — page.go 에 그 셋의 문자열이 없다
   R28   /api/* 에는 이 헤더를 안 건다.  JSON 이고 브라우저가 문서로 안 읽는다
```

**`'unsafe-inline'` 이 script 에 붙는 대가를 이름으로 적는다** — 주입된 스크립트
자체는 CSP 가 못 막는다. **막는 것은 R18 하나다.**

---

## 8. 잔여 — 이 유닛이 안 닫는 것

```text
   ①  ReadRing 이 「파일이 없다」와 「파일이 깨졌다」를 둘 다 available:false 로
      접는다.  깨진 링이 「아직 없음」으로 보인다 — 셋째 거짓말이다.
      고치면 겉면이 는다.  이 유닛이 만든 것이 아니다

   ②  LAN 노출에서 GET / 에 토큰이 필요해 브라우저로 페이지를 못 연다
      (requireToken 이 mux 전체를 감싼다).  앞 팩의 성질이고 이 유닛 밖이다

   ③  512 KiB 를 매초 파싱하는 비용을 실측으로 안 쟀다.  침묵에서는 0 이고
      (R7 의 캐시) 말이 많을 때만 든다.  **CB1 이 512 KiB 를 넘기는 프롬프트로
      그 자리를 잰다** — 빨개지면 답 2 를 B 로 다시 연다

   ④  사건이 수천일 때 매초 DOM 을 다시 짓는 비용.  링 용량이 상한을 준다.
      CB1 이 같은 프롬프트로 함께 잰다 — 빨개지면 답 8 을 B 로 다시 연다
```

---

## 9. 회차 밖으로 낼 것 — 넷

```text
   requirements.md 5.5    「ui.go:64-71 과 같은 다섯 줄」이 page.go 의 모양을 안 보고
                          쓰였다.  통짜 인라인 페이지에 default-src 'self' 는
                          같은 다섯 줄이 아니다.  R25 가 그 어긋남이다

   features.md 3.3 ·       「밀렸다는 표시는 앞 팩 그대로」가 거짓이다.  앞 팩에 0 이고
   requirements.md FR-4    CB1 의 화면 검증이 그것을 본다.  이 유닛이 새로 짓는다

   unit-of-work.md U5 절    NFR 요구를 스킵이라 적고 그 근거로 U6 의 문장을 댔다.
                          같은 문서 9절의 표는 돈다로 적는다.  한 문서 안에서 두 값이다

   파일 행렬 1절            internal/enode/transcript.go 가 어느 유닛에도 안 붙어 있다.
                          U5 가 Snapshot 에 Capacity 를 더한다.  동작 diff 0 ·
                          U2 가 이미 병합돼 충돌 0
```
