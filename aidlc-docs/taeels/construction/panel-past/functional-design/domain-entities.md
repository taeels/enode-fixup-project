# U6 `panel-past` — 도메인 개체

```text
   유닛    panel-past (U6) · 웨이브 W-d · 브랜치 unit/panel-past
   계획    ../../plans/panel-past-functional-design-plan.md
   답      1=A · 2=A · 3=A · 4=A · 5=A · 6=A  (여섯이 전부 권장)
   기준선   b3c7a24 — unit/fleet-card 위 (답 1 = A)
```

**새 타입이 하나도 안 생긴다.** 이 유닛이 하는 일은 **같은 것을 다른 문으로
가져오는 것**이다 — tar 를 풀던 자리에서 `GET` 라우트로. 파서도 렌더러도 봉투의
모양도 이미 있다.

그래서 이 문서가 적는 것은 **무엇이 없어지고 무엇이 그 자리에 오나**다.

---

## 1. 없어지는 것 — `archive/tar`

```go
   지금   internal/panel/transcript.go:4    "archive/tar"
          handleRecord 가 Record 로 tar 를 받아 logs/NN-*.log 만 꺼낸다
   뒤     임포트가 사라진다.  파일에 tar 라는 낱말이 0 이다
```

**CB2 가 이것을 잰다** — 「출처가 GET 이고 tar 를 안 푼다」. 브라우저 네트워크
탭에서 `GET /v1/runs/<id>/record` 가 없고 `.../log` 가 있는 것으로 본다.

```text
   안 없어지는 것   internal/runctl 의 Client.Record.  cmd/runctl/main.go:365 이 쓴다
                  (runctl record <id> -o r.tar).  이 유닛이 걷는 것은 제어판의
                  호출 하나이지 그 메서드가 아니다
```

**앞 팩이 tar 를 고른 이유는 「Mediator 를 안 건드린다」였고, 이 팩이 그 경계를
값으로 무른다** (`decisions.md` 2절). 무르는 값이 CB2 다 — 같은 파서의 같은 모양.

---

## 2. 그 자리에 오는 것 — 라우트 둘

```text
   GET /v1/runs/{id}                   단계 목록.  Client.Status
   GET /v1/runs/{id}/steps/{seq}/log   단계마다 하나.  **새 클라이언트 메서드**
```

### 2.1 새 겉면 하나 — `runctl.Client.StepLog`

```go
   func (c *Client) StepLog(ctx context.Context, runID string, seq int, name, as string) ([]byte, http.Header, error)
```

```text
   왜 헤더를 함께 내나   출처 · 총 길이 · 상한이 헤더에 있다 (NC-4 · NC-5).
                     몸통만 내면 부르는 쪽이 「봉인 전인가 뒤인가」를 못 안다
   왜 as 를 받나       events 와 raw 둘을 같은 메서드로 부른다.  U4 가 그 둘만 받는다
   기존 do() 를 탄다    인증 · 오류 사상이 그대로다.  새 외부 표면이 0 인 이유가 이것이다
```

**`?name=` 에는 `Step.ID` 를 넣는다.** 상세의 단계 객체에 `name` 이 없고 `ID` 가
곧 로그의 이름이다 — **U8 이 현황판에서 이미 밟은 자리**이고 (빈 카드가 200 에
0 바이트로 왔다) 답이 같다. `GET log` 의 `name` 기본값은 아직 `step` 이다.

---

## 3. 봉투 — 도는 것과 **같은 키**를 쓴다

답 2 = A. 지금은 `{logs: [{name, content}]}` 다.

```json
{
  "run_id": "cb4-card-4",
  "sealed": true,
  "steps": [
    {
      "seq": 1,
      "id": "survey",
      "state": "DONE",
      "chosen": true,
      "source": "sealed",
      "total": 11553,
      "capped": false,
      "transcript": { "events": [], "elided": { "events": 98, "bytes": 408989 }, "raw": 0, "lines": 0, "head": 0, "partial": 0 },
      "data": "<원문 바이트 그대로>"
    }
  ]
}
```

```text
   transcript   transcript.Result 통째.  **U5 의 liveTranscript 와 같은 키다**
   data         원문 토글이 그린다.  같은 읽기의 같은 바이트
   source       X-Enode-Log-Source 그대로 (progress · sealed).  NC-5
   total        X-Enode-Log-Bytes 그대로
   chosen       Step.Chosen 그대로.  0 바이트의 이유를 가르는 데 쓴다 (5절)
   sealed       이 Run 이 봉인됐나.  steps 의 source 와 같은 답을 내야 한다 (6절)
```

**키를 `transcript` 로 맞추는 것이 이 봉투의 전부다.** U5 의 도메인 문서가 그
이유를 미리 적어 뒀다 — 「U6 이 U4 의 라우트로 출처를 바꿀 때 그쪽 몸통이 이미
`transcript.Result` 다. 같은 키에 그대로 담으면 화면 함수가 하나로 선다」.

```text
   도는 것 (U5)   t.transcript.events
   지난 것 (U6)   s.transcript.events      단계 하나가 t 자리에 온다
```

**꺼내는 식이 같으면 그리는 함수가 하나다.** 그 함수는 U8 이 이미 옮겨 놨다
(`window.enodeCard.renderEvents`).

---

## 4. 걷힌 양 — `Result.Elided`

답 4 = A. **새 타입이 아니다** — 파서가 `enode.elided` 줄을 읽어 이미 낸다.

```text
   실측    cb4-card-4 의 survey — {"events":98,"bytes":408989}
   크기    진행 파일 127,117 바이트 대 봉인된 로그 11,553 바이트.  **9%**
   어디에   단계마다 한 줄.  사건 열 위에 「본문 98 개가 걷혔다 (408,989 바이트)」
```

**봉인된 것은 도는 동안 본 것과 다른 물건이다.** 9% 만 남고 나머지는 수 둘로
접혀 있다. **화면이 그 말을 안 하면 읽는 사람이 같은 물건으로 읽는다.**

### 4.1 이 줄이 둘째 신호다

```text
   진행 파일    enode.elided 가 **없다.**  걷는 일이 봉인 때 일어난다
   봉인된 로그   enode.elided 한 줄이 있다
```

**출처 헤더가 첫째 신호이고 이것이 둘째다.** 둘이 같은 답을 내야 한다 —
`business-rules` 가 그 자리를 불변식으로 적는다.

---

## 5. 총 길이 0 의 세 가지 뜻

답 6 = A 를 모순 검사가 셋으로 넓혔다. `Step.State` 와 `Step.Chosen` 이 가른다.

```text
   SKIPPED · chosen 거짓    경로가 갈려 안 갔다
   SKIPPED · chosen 참      골랐는데 못 닿았다
   그 밖에 total 이 0       돌았고 아무 말도 안 했다
```

**`Step.Chosen` 이 있는 이유가 정확히 이 구별이다** — 그 열의 주석이
「false 를 생략하면 SKIPPED 하나가 두 가지를 다시 뜻하게 된다」로 적는다.
한 문장으로 접으면 그 열이 나른 값을 화면에서 버린다.

**단계를 목록에서 빼지 않는다.** 빼면 화면의 단계 수가 계약과 안 맞고,
**왜 없는지**를 말할 자리가 사라진다.

---

## 6. 봉인 전과 뒤 — 같은 라우트, 다른 파일

답 3 = A. `GET log` 는 봉인 전이면 진행 파일을, 뒤면 봉인된 로그를 낸다.

```text
   봉인 전   source=progress · 걷힌 줄 없음 · 큰 파일   화면에 「봉인 전」을 적는다
   봉인 뒤   source=sealed  · 걷힌 줄 있음 · 9%        그대로 읽는다
```

**같은 Run 을 두 번 누르면 다른 것이 온다.** 진행 파일과 봉인된 로그는 다른
파일이다. **그것을 설명하는 줄이 출처다** — 화면이 헤더 값을 그대로 그린다.

---

## 7. 안 짓는 것

```text
   파서            internal/transcript 를 그대로 쓴다
   렌더러           U8 의 card.mjs.  이 유닛이 그리는 코드를 0 줄 짓는다
   상태 줄          statusLine 을 안 부른다 — 지난 것에 「쓰는 중」이 없다 (R56 과 같은 값)
   폴링            봉인된 것은 안 자란다.  자동 갱신이 0 이다
   verdict         Run 목록에서 온다.  유닛 정본이 「안 한다」로 적었다
   라우트           GET /api/record 가 이미 있다.  panel.go 의 11 이 그대로다
   보안 헤더        U5 가 모든 응답에 건다 (R28)
```
