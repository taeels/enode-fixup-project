# U5 `panel-live` — 도메인 개체

```text
   유닛    panel-live (U5) · 웨이브 W-c · 브랜치 unit/panel-live
   계획    ../../plans/panel-live-functional-design-plan.md
   답      1=A · 2=A · 3=A · 4=A · 5=C · 6=A · 7=A · 8=A
   기준선   05ee710
```

**새 타입을 거의 안 짓는다.** 이 유닛이 하는 일은 이미 있는 것 둘(링과 파서)을
잇고, 그 이음매에 **시각 하나와 용량 하나**를 더하는 것이다.

---

## 1. 라우트는 하나도 안 는다

```text
   GET /api/transcript    있다.  응답의 모양이 바뀐다
   GET /api/runs          그대로
   GET /api/record        U6 의 것이다.  이 유닛이 안 만진다
   GET /                  같다.  헤더가 다섯 는다 (5절)
```

**`grep -c 'mux.HandleFunc' internal/panel/panel.go` 가 10 그대로다.** CB0 이
`internal/api/api.go` 를 세므로 이 유닛은 그 수에 0 을 더한다.

---

## 2. 넓어지는 겉면 하나 — `enode.Snapshot.Capacity`

답 4 = A 다.

```go
   type Snapshot struct {
       Data       []byte
       Generation uint64
       Total      uint64
       Capacity   int64   // 새 필드
   }
```

**`ReadRing` 이 이미 읽어 놓고 버리던 값이다** (`transcript.go:181` 의
`capacity := int64(...)`). 더하는 것이 필드 하나와 대입 한 줄이다.

```text
   왜 필요한가   Parse(b, truncated) 의 truncated 가 Total > Capacity 다.
                NC-2 와 CB1 의 「첫 줄이 잘린 채로도 파싱된다」가 둘 다 이 값을 쓴다
   왜 상수가     OpenRing 이 용량을 인자로 받는다.  오늘 부르는 자리가 하나라
   아닌가        DefaultTranscriptCapacity 와 같을 뿐이고, 달라지는 날 조용히 틀린다
   동작 diff     0.  읽는 값이 하나 더 나갈 뿐이다
```

**`internal/enode/transcript.go` 는 파일 행렬 밖이다** — 행렬이 그 파일을 아무
유닛에도 안 줬다. **U2 가 이미 병합됐으므로 충돌이 0 이다** (W-b 가 `record.go` ·
`transcript.go` 로 같은 모양을 둘 겪었고 실제로 0 이었다).

---

## 3. 시각 하나 — 링 파일의 mtime

답 3 = A. **새 타입이 아니라 이미 파일시스템에 있는 값이다.**

```text
   무엇의 시각   링 파일의 마지막 쓰기.  Ring.Write 가 쓸 때마다 flushHeader 하고
                Ring.Reset 도 머리를 쓴다 — 단계 시작이 경과 0 으로 잡힌다
   읽는 법      os.Stat(TranscriptPath(cfg.ConfigPath)).ModTime()
   선 위 모양    RFC 3339 문자열.  **기간이 아니라 시각이다** (business-rules R2)
```

**기간을 안 싣는 것이 모순 검사가 막은 자리다** — 계획 5.2.1 ①.

---

## 4. 응답의 모양 — 봉투를 여기서 정한다

답 2 = A · 6 = A · 7 = A 가 함께 짓는다. **교차 검사가 이 봉투를 U6 의 것까지
정하게 했다** (계획 5.2.1 교차).

```json
{
  "available": true,
  "generation": 7,
  "total": 918273,
  "capacity": 524288,
  "truncated": true,
  "last_write": "2026-09-16T07:41:02Z",
  "ring_path": "/home/sunny/.enode/node.transcript",
  "transcript": { "events": [], "raw": 0, "lines": 0, "head": 0, "partial": 0 },
  "data": "<원문 바이트 그대로>"
}
```

```text
   transcript   transcript.Result 통째로.  **U4 의 as=events 몸통과 같은 타입이다**
                화면 함수가 보는 것은 이 키 하나다
   data         같은 읽기에서 나온 같은 바이트.  원문 토글이 그린다 (답 6 = A)
   available    링을 못 읽었으면 false 이고 나머지 키가 없다
   ring_path    답 7 = A.  노드마다 다르므로 서버가 낸다
```

**`transcript` 를 형제 키로 안 펼치고 통째로 담는 이유** — U6 이 U4 의 라우트로
출처를 바꿀 때 **그쪽 몸통이 이미 `transcript.Result` 다.** 같은 키에 그대로
담으면 화면 함수가 하나로 선다. 펼치면 U6 이 다시 조립해야 하고, 그 조립이
둘째 봉투다.

---

## 5. 응답 헤더 다섯 — `GET /` 에 건다

답 1 = A. `requirements.md` 5.5 를 닫는다.

```text
   Strict-Transport-Security   max-age=31536000; includeSubDomains   ui.go 그대로
   X-Content-Type-Options      nosniff                               ui.go 그대로
   X-Frame-Options             DENY                                  ui.go 그대로
   Referrer-Policy             strict-origin-when-cross-origin       ui.go 그대로
   Content-Security-Policy     default-src 'self';                   **이 페이지의 모양에 맞췄다**
                               style-src 'self' 'unsafe-inline';
                               script-src 'self' 'unsafe-inline'
```

**넷은 글자 그대로이고 다섯째만 갈렸다.** `page.go` 가 통짜 인라인이라
`default-src 'self'` 한 줄이 이 페이지를 죽인다 (계획 1.2).

```text
   그래도 남는 값   connect-src · img-src · font-src 가 default-src 를 물려받는다.
                  **주입이 일어나도 밖으로 못 보낸다** — 유출 경로가 막힌다
   잃는 값        주입된 스크립트 자체는 CSP 가 못 막는다.
                  막는 것은 esc() 와 textContent 뿐이다 (business-rules R6)
```

**이 페이지가 밖에서 가져오는 것이 0 인 것을 실측으로 확인했다** — `data:` 0 ·
외부 URL 0 · `@import` 0. 폰트는 이름만 적고 대체 스택으로 떨어진다. 그래서
`font-src` 를 따로 안 연다.

---

## 6. 화면이 새로 그리는 것 — 카드 하나 안의 넷

```text
   사건 열      문장 · 도구 이름 · 접힌 결과.  transcript.Kind 일곱만 그린다
   NC-1 경과    「마지막 사건 이후 N초」.  last_write 에서 브라우저가 뺀다
   NC-2 잘림    truncated 가 참이면 「앞 (total - capacity) 바이트가 감겨 나갔다」
   NC-3 잔여    ring_path 와 「단계마다 덮인다」 한 줄
   원문 토글     data 를 <pre> 로.  같은 읽기의 같은 바이트다
```

**데몬 로그 카드는 그대로 옆에 있다** — 이 유닛이 안 만진다 (FR-4 · `features.md`
3.3 의 「둘이 다른 물건임이 화면에서 보인다」).

---

## 7. 안 짓는 것

```text
   파서             internal/transcript 를 그대로 쓴다.  panel 이 파싱을 안 짓는다
   from= 오프셋      답 2 = A 가 뺐다.  브라우저에 경계 상태를 안 들인다
   새 .pen          S3 의 기존 카드를 고치는 것이다.  새 화면이 아니다
   Mediator 왕복     이 카드는 링만 읽는다.  U4 의 라우트를 안 탄다
   봉인된 것         U6 이다.  Elided · Shell 은 이 카드에 안 온다
```
