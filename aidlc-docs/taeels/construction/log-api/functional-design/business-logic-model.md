# U4 `log-api` — 순서

**이 유닛이 사는 값은 갈림의 순서다.** 무엇이 있는가는 `domain-entities.md` ·
불변식은 `business-rules.md`.

---

## 1. 모순 검사가 막은 것 하나 — 다음 오프셋을 못 구한다

**답 6 = A (한 응답에 1 MiB 상한)와 `as=events` 를 그대로 합치면 폴링이 깨진다.**

```text
   상한이 걸린다      본문이 1 MiB 에서 끊긴다
   그 자리가 줄 한가운데  transcript.Parse 가 그 꼬리를 안 읽고 Partial 에 센다
   다음 from 을 못 구한다  from + len(본문) 은 줄 한가운데를 가리킨다.
                       그 자리에서 다음 응답을 파싱하면 첫 줄이 반쪽이다
```

**읽는 쪽이 자기가 읽은 만큼을 셀 수 없다** — `as=events` 의 본문은 사건 배열이라
바이트 수가 안 보인다.

```text
   답    상한에 걸리면 서버가 마지막 개행에서 끊는다.
        그러면 다음 from 은 언제나 from + len(본문) 이다 — 갈래가 0 이다
        한 줄이 상한보다 길면 개행이 없다.  그때만 상한 그대로 끊고
        Partial 이 0 이 아니게 나간다 (R7 · R8)
```

**헤더를 다섯째로 안 늘린다.** `component-methods.md` 4.1 이 넷으로 못 박았고,
개행에서 끊는 규칙 하나가 같은 것을 해낸다.

---

## 2. GET 이 도는 순서

```text
   ①  needRecords            기록 저장소가 없으면 503
   ②  seq · from · as 를 검증  규칙 밖이면 400.  여기가 R33 이 가리킨 자리다
   ③  GetRun                 없으면 404.  이것 하나가 404 다
   ④  Sealed(runID)          출처를 가른다.  Run 의 상태가 아니다
   ⑤  본문을 연다
        progress   ReadProgress(run, seq, name, from)  -> Progress · 본문
        sealed     OpenLog(run, seq, name)             -> 본문 · 총 길이
                   from 만큼 Seek 한다
   ⑥  상한을 건다             마지막 개행에서 끊는다 (1절)
   ⑦  헤더 넷을 쓴다          본문보다 먼저다.  쓰기 시작하면 못 고친다
   ⑧  as 로 갈린다
        raw      바이트를 그대로 흘린다
        events   Parse(조각, truncated) 를 JSON 으로 낸다
```

**④ 가 ⑤ 앞이고 ⑦ 이 ⑧ 앞이다.** 둘 다 되돌릴 수 없는 순서다 — 본문을 한 바이트
라도 쓰면 헤더를 못 고치고, 출처를 모르면 어느 파일을 열지 모른다.

### 2.1 `truncated` 를 서버가 정한다

`Parse(b, truncated)` 의 둘째 인자는 **앞이 잘려 왔다**는 뜻이다.

```text
   from == 0        truncated = false
   from  > 0        from - 1 바이트가 개행이면 false · 아니면 true
```

**부르는 쪽이 줄 경계에서 폴링하면 언제나 false 다.** 1절의 규칙이 그것을
보장한다. `true` 가 되는 것은 **사람이 손으로 `from` 을 주거나 한 줄이 상한보다
길어 중간에서 끊긴 뒤**뿐이고, 그때 `Result.Head` 가 버린 바이트를 센다.

---

## 3. 봉인 전후가 갈리는 순간

```text
   Run 이 돈다              Sealed=false · 진행 파일이 있다      progress
   Run 이 종료 상태가 된다    Sealed=false · 진행 파일이 아직 있다  progress
   Seal 이 돈다
     ①  DropProgress       진행 파일이 사라진다
     ②  logs/ 를 굳힌다
     ③  verdict.json 을 0444 로                                 여기서 Sealed=true
   봉인 뒤                  Sealed=true · logs/ 를 읽는다        sealed
```

**①과 ③ 사이의 창에서는 `Sealed` 가 거짓이고 진행 파일이 없다.** 그때 응답은
`Source: progress` · `Bytes: 0` · 빈 본문이다. **틀린 값이 아니다** — 진행 파일을
봤고 없었다는 사실 그대로다.

**화면이 그 창을 「멈췄다」로 읽지 않는 것은 NC-1 의 경과 표시가 진다** (U5 · U6).

---

## 4. PUT 이 도는 순서

```text
   ①  needRecords
   ②  seq 를 검증
   ③  GetRun.  없으면 404
   ④  종료 상태면 410 — 두 갈래 다 (api.go:879 오늘 그대로)
   ⑤  progress 쿼리로 갈린다
        없다    AppendLog(run, seq, name, body, MaxBlobBytes)    -> 총 길이
        있다    AppendProgress(run, seq, name, attempt, body, MaxBlobBytes)
                                                                -> Progress
   ⑥  헤더를 쓰고 200
        없다    X-Enode-Log-Bytes
        있다    X-Enode-Log-Bytes · X-Enode-Log-Attempt · X-Enode-Log-Capped
```

**④ 가 ⑤ 앞이다.** 봉인된 Run 에 청크를 미는 것은 두 갈래 다 410 이고, 그것이
오용 시나리오 ⑤ 의 답이다 (`requirements.md` 5.6).

**`attempt` 가 없거나 규칙 밖이면 400 이다.** `progress=1` 인데 시도를 모르면
`AppendProgress` 가 어느 시도의 파일에 붙일지 못 정한다 — R9 가 앞 시도를 걷는
자리라 **틀린 값이 앞 시도를 지운다.**

---

## 5. 실패 경로

```text
   저장소가 답을 못 함      503.  본문을 안 쓴 채로 낸다
   본문을 흘리는 중 실패     헤더는 이미 나갔다.  로그에 남기고 끊는다 —
                         부분 응답이 되고 읽는 쪽이 Bytes 와 견주어 안다
   Parse 가 못 읽는 줄      오류가 아니다.  raw 나 plain 으로 남는다
                         (transcript 에 error 를 내는 경로가 0 이다)
   OpenLog 가 없는 파일     200 에 Bytes 0.  404 가 아니다 (답 7 = A)
```

---

## 6. 무인증 노출이 어디까지인가 — 답 2 = A 의 값

```text
   실 함대    read() 가 s.auth 다.  오늘의 GET /v1/runs 와 같다
   데모       read() 가 s.limit 다 — 무인증 + 한도 (api.go:82)
```

**데모에서는 이 라우트가 토큰 없이 하네스 원문을 낸다.** `requirements.md` 5.4 의
잔여 ② 는 「토큰 하나라 주체를 못 가른다」로 **토큰이 있다는 전제**로 쓰였고,
데모에는 그 전제가 없다. **잔여가 둘로 갈린다** — `business-rules.md` 6절이 든다.

**잠그면 CB4 가 죽는다.** 데모 현황판이 그 라우트로 단계 카드를 그리고, 데모에는
토큰 입력 화면이 아예 없다. 값을 보고 A 를 골랐다.
