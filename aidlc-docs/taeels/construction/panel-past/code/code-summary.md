# U6 `panel-past` — Code Generation 요약 (Part 2)

```text
   유닛    panel-past (U6) · 웨이브 W-d · 브랜치 unit/panel-past
   계획    ../../plans/panel-past-code-generation-plan.md
   기준선   8b1e8ed (계획 커밋).  그 아래가 unit/fleet-card
   지는 게이트  **CB2** — 사람이다.  아직이다
```

**Step 여덟을 돌았고 체크박스 쉰셋이 실측으로 찼다.** 새 파일이 0 이고 새 Go
패키지가 0 이다 — 이 유닛은 **출처만 바꾼다.**

---

## 1. 무엇이 없어지고 무엇이 그 자리에 왔나

```text
   없어진 것   internal/panel 의 archive/tar 임포트.  tar 를 푸는 코드 전부
             recordLog{name, content} 타입
             GET /v1/runs/{id}/record 호출 하나 (제어판 쪽만)

   온 것      runctl.Client.StepLog  — 원문 바이트와 헤더를 함께 낸다
             pastRecord · pastStep   — 단계마다 한 칸인 봉투
             page.go 의 지난 것 경로 — 도는 것과 같은 렌더러를 부른다

   안 없어진 것 runctl 의 Client.Record.  cmd/runctl 이 쓴다 (runctl record)
             GET /api/record.  라우트 수가 11 그대로다
```

**그리는 코드를 0 줄 지었다.** `window.enodeCard.renderEvents` 를 부른다 —
U8 이 렌더러를 한 벌로 빼 놓은 값이 여기서 돌아온다. 앞 판이었으면 지난 것을
그리는 코드를 `page.go` 안에 또 지어야 했고, 그것이 U5 와 같은 모양의 두 번째
벌이었다.

---

## 2. 계획의 고침 셋이 코드가 됐다

### 2.1 한 읽기로 봉투 하나 (계획 1.1)

```go
   StepLog(ctx, runID, seq, name)   as=raw 하나.  ?from= 이 0 번
   transcript.Parse(body, false)    제어판이 부른다.  같은 바이트에서 사건 열이 난다
   out.Data = string(body)          원문 토글이 그것을 그린다
```

**서버의 `as=events` 와 글자까지 같은 답이다** — 그쪽도 같은 조각을 같은 파서에
넣고, `?from=` 이 없으면 첫 줄을 버릴 일이 없다. 둘을 따로 부르면 봉인 전
진행 파일이 그사이 자라 사건 열과 원문이 다른 창을 보인다.

### 2.2 `sealed` 대신 `state` (계획 1.2)

봉투에 `sealed` 가 없다. 봉인 전과 뒤는 **단계마다의 `source`** 가 말하고,
실측 화면에서 한 봉투 안에 `sealed` 단계와 `progress` 단계가 함께 섰다.

### 2.3 받은 길이를 서버가 센다 (계획 1.3)

```text
   total      파일의 총 길이.  헤더 그대로
   received   이 응답이 받은 바이트 수.  둘이 다르면 잘린 것이다
```

**브라우저가 문자열 길이로 대신 못 센다** — JS 문자열의 길이는 UTF-16 단위라
하네스가 한국어를 한 줄만 내도 바이트 수와 갈린다. 계획에 없던 필드 하나이고
이유가 그것이다.

---

## 3. 계획의 한 줄을 실측이 고쳤다

**`internal/panel` 의 tar 를 `go list -deps` 로 못 잰다.** `internal/enode` 가
`archive/tar` 를 딛고(봉인이 tar 를 짓는다) 제어판이 그것을 임포트하므로
**의존 그래프에는 언제나 보인다.** 걷은 것은 제어판 자신의 임포트이므로
`go list -f '{{join .Imports ...}}'` 로 **직접 임포트**를 잰다.

**「파일에 tar 라는 낱말이 0」도 지키지 않았다.** 남은 넷이 전부 **걷힌 이유를
적는 주석**이다 — 어디서 왔고 무엇을 무르고 옮겼는지. 낱말 수는 임포트가
사라졌는지의 대용이었고, 그 자리는 이제 경계 시험이 직접 잰다.

---

## 4. 변이 일곱 — 전부 죽었다

```text
   ①  source 를 안 옮기고 sealed 를 박는다        죽었다
   ②  elided 를 봉투에서 버린다                   죽었다
   ③  단계 하나의 실패를 전체 502 로 접는다        죽었다
   ④  SKIPPED 의 chosen 을 안 싣는다              죽었다
   ⑤  로그 0 바이트인 단계를 목록에서 뺀다         죽었다
   ⑥  data 를 두 번째 읽기로 채운다               죽었다
   ⑦  404 를 502 로 접는다                       죽었다
```

**④ 를 재려고 시험의 픽스처를 늘렸다.** 처음 판은 `SKIPPED · chosen 거짓`
단계 하나뿐이라 `Chosen` 을 통째로 버려도 안 죽었다. 같은 상태에 `chosen` 만
다른 단계를 하나 더 세워 **R76 과 R77 의 구별 자체**를 재게 했다.

**⑥ 이 이 유닛의 값을 지키는 줄이다.** 가짜 Mediator 가 읽을 때마다 다른 것을
내므로 두 번 읽으면 사건 열과 원문이 어긋난다.

---

## 5. 선 위에서 본 것 — 화면을 실제로 돌렸다

**CB2 는 사람이고 아직이다.** 그 전에 화면 쪽 JS 가 실제 렌더러 위에서 도는지를
한 번 봤다 — `page.go` 의 HTML 을 꺼내 실제 `card.mjs` 와 함께 올리고, 봉투는
**실제 파서(`transcript.Parse`)가 낸 것**을 먹였다.

```text
   단계 1   봉인된 로그 · 501 바이트 · 「본문 98 개가 걷혔다 (408989 바이트)」
           사건 넷 — 말 · 도구 Bash · 접힌 결과 · 「success · 턴 21 · $0.6609222」
   단계 2   「봉인 전 — 진행 파일이다」 · 127117 바이트
           「처음 231 바이트만 왔다 — 한 응답의 상한이다」
   단계 3   SKIPPED · 0 바이트 · **「경로가 갈려 안 갔다」**
   단계 4   SKIPPED · 0 바이트 · **「골랐는데 못 닿았다」**   상태가 같고 chosen 만 다르다
   단계 5   「이 단계의 로그를 못 불러왔다 — 503 read failed」.  앞뒤 넷은 그대로 그렸다
   verdict  checks 가 그대로 보인다 (이 유닛이 안 만졌다)
```

```text
   펼침      [펼치기] 를 누르면 [접기] 가 되고 본문이 선다.  열쇠가 tool_use_id
   단계 독립   단계 1 을 펼쳐도 단계 2 는 안 움직인다
   원문 토글   pre 의 자식 요소가 **0 개**다 — 하네스 바이트가 textContent 로만 들어갔다
   되돌림     원문을 껐다 켜도 펼침이 살아 있다
```

**이것은 게이트가 아니다.** 실제 하네스도 실제 Mediator 도 안 탔다 — 그 둘이
CB2 이고 사람이 본다.

---

## 6. 게이트 — 기계가 재는 것은 전부 초록

```text
   go test ./... -count=1              초록 (testdb.sh 로 DB 를 띄웠다)
   패키지별 커버리지 80% 하한            **미달 0** · 전체 7783/8878 = 87.7%
     internal/panel    86.0% -> 88.5%  (221/257 -> 232/262)
     internal/runctl   97.1% -> 97.3%  (100/103 -> 110/113)
   node --test internal/api/ui/tests/*  99 중 99 초록 (**안 건드렸다**를 잰다)
   glyphscan                           통과 (113 파일)
   gofmt                               깨끗
   mux.HandleFunc 수                    **11 그대로** (R65 · CB0)
   경계 시험                            열둘 + 봉인 둘 + tar 금지 하나, 전부 초록
```

```text
   diff 0    internal/transcript/**  ·  internal/transcriptui/card.mjs
             internal/api/**  ·  internal/api/ui/**
             internal/panel/panel.go  ·  cmd/runctl/main.go
```

---

## 7. 잔여 — 이 유닛이 안 닫는 것

```text
   ①  봉인 전 Run 을 누르면 진행 파일 전체가 한 응답에 실린다.  누를 때 한 번이고
      폴링이 아니다.  **CB2 가 이 자리를 밟는다**

   ②  1 MiB 를 넘는 단계는 처음 조각만 온다.  화면이 그 말을 하지만 나머지를
      가져오지는 않는다 — ?from= 이어받기는 U4 의 라우트를 다시 여는 일이다

   ③  GET log 의 name 기본값이 아직 step 이다.  U8 에 이어 이 유닛이 두 번째로
      화면에서 Step.ID 를 넘겨 비켜 간다.  **화면 둘이 같은 자리를 비켜 가면
      기본값이 틀린 것이다** (회차 밖으로 낸다)

   ④  R81 을 재는 자동 시험이 0 이다.  제어판의 JS 를 재는 하네스가 없다 —
      5절은 사람이 한 번 돌린 것이지 CI 가 도는 것이 아니다.  **CB2 가 그 자리다**

   ⑤  로그가 0 바이트인 단계에도 원문 단추가 선다.  없애면 「단추가 없다」가
      실패와 빈 것 둘을 뜻하게 되므로 그대로 뒀다
```

## 8. 어긋남 — 적고 넘어간 것

```text
   unit-of-work.md U6 절        경로에 internal/panel/page.go 가 빠져 있다.
                               이 유닛이 그 파일을 만졌다 — 안 만지면 출처만
                               바뀌고 화면이 안 바뀌어 CB2 가 안 선다

   unit-of-work-dependency.md   U6 의 코드 의존에 U8 칸이 없다.  2026-09-17 의
   의 U6 행                      설계 변경이 그 행렬보다 뒤다

   파일 행렬                    page.go 는 U5 · U8 의 것이다.  **U8 이 병합된
                               뒤에 만지므로 충돌 0** — 이 브랜치가 unit/fleet-card
                               에서 났다
```
