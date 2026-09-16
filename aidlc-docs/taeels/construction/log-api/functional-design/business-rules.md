# U4 `log-api` — 불변식

**번호를 단다.** Code Generation 이 이 번호로 시험을 걸고, 계획의 체크박스가
이 번호를 가리킨다.

---

## 1. 라우트

```text
   R1   라우트가 하나 는다.  grep -c 'mux.HandleFunc' internal/api/api.go 가
        17 에서 18 이다.  는 것은 GET 하나뿐이고 PUT 의 갈림은 안 센다 (CB0)
   R2   핸들러는 새 파일 internal/api/log.go 에 산다.  api.go 는 등록 줄 하나만 는다
   R3   보호 래퍼는 read() 다.  GET /v1/runs 와 같다 (FR-6)
```

## 2. 검증 — SECURITY-05

```text
   R4   seq 는 양의 정수다.  아니면 400
   R5   from 은 음이 아닌 정수다.  없으면 0.  아니면 400
   R6   as 는 raw 와 events 둘뿐이다.  없으면 raw.  아니면 400
   R6.1 name 이 없으면 step 이다.  PUT 과 같은 기본값이다 (api.go:884)
   R6.2 progress=1 인 PUT 에 attempt 가 없거나 규칙 밖이면 400 이다.
        모르는 채로 붙이면 R9 의 걷기가 앞 시도를 지운다
```

**검증이 이 층의 일이라고 U3 가 이름으로 적었다** (R33). 아래층은 `from` 이
음수면 0 으로 보고 넘어간다 — 그 층의 전제가 R32 하나다.

## 3. 조각과 오프셋

```text
   R7   한 응답의 본문에 상한이 있다.  시작값 1 MiB.  X-Enode-Log-Bytes 는
        그 상한과 무관하게 파일의 총 길이다
   R8   상한에 걸리면 마지막 개행에서 끊는다.  그래서 다음 from 은 언제나
        from + len(본문) 이다.  조각 안에 개행이 0 이면 (한 줄이 상한보다 길다)
        상한 그대로 끊는다 — 그때만 Partial 이 0 이 아니다
   R9   as=events 의 truncated 는 서버가 정한다.
        from == 0 이면 false.  from > 0 이면 from-1 바이트가 개행인가로 정한다
```

**R8 이 없으면 R7 이 폴링을 깬다.** `as=events` 의 본문은 사건 배열이라 읽는
쪽이 자기가 읽은 바이트를 셀 수 없다.

## 4. 출처

```text
   R10  출처는 records.Sealed(runID) 하나로 가른다.  Run 의 상태가 아니다
   R11  X-Enode-Log-Source 는 본문을 어느 파일에서 읽었나다.  Run 이 끝났나가 아니다
   R12  봉인된 본문은 record.OpenLog 로 연다.  internal/api 가 경로를 조립하지
        않는다 — safe() 와 "%02d-%s.log" 가 두 벌이 되면 안 된다
   R13  파일이 없으면 200 에 Bytes 0 · 빈 본문이다.  404 는 Run 이 없을 때 하나다
```

## 5. PUT 의 갈림

```text
   R14  라우트가 안 는다.  갈래는 progress 쿼리가 정한다 (D2)
   R15  종료 상태면 두 갈래 다 410 이다 (오용 시나리오 ⑤)
   R16  두 갈래 다 X-Enode-Log-Bytes 를 단다.  204 가 200 이 된다.
        진행 갈래만 Attempt 와 Capped 를 더 단다
   R17  상한은 AppendProgress 가 총 길이로 건다 (U3 · MaxBlobBytes).
        이 층이 따로 세지 않는다
```

## 6. 어휘

```text
   R18  as=events 의 키는 internal/transcript 의 json 태그가 정한다.
        internal/api 가 자기 DTO 를 안 짓는다 — 모양의 정본이 하나다
   R19  Event.OK 의 태그에 omitempty 가 붙는다.  포인터라 nil 이면 키가 빠지고
        false 와 「없음」이 선 위에서도 갈린다
   R20  응답 몸통은 transcript.Result 를 그대로 낸다 (events · elided · raw ·
        lines · head · partial)
```

## 7. 잔여 — 이 유닛이 안 닫는 것

```text
   ①  데모 모드에서 이 라우트가 무인증이다 (api.go:82 의 read).
       requirements.md 5.4 의 잔여 ② 는 「토큰 하나라 주체를 못 가른다」로
       토큰이 있다는 전제로 쓰였다.  **데모에는 그 전제가 없다.**
       잠그면 CB4 가 죽는다 (데모에 토큰 입력 화면이 없다).
       막는 것은 코드가 아니라 운영의 규칙이다 — 데모에 올리는 계약은
       비밀을 안 담는 것이어야 한다

   ②  주체별 권한이 없다.  누구의 토큰이든 남의 Run 의 로그를 읽는다.
       5.4 의 잔여 ② 그대로다.  좁히려면 권한 모델을 새로 만들어야 하고 팩 밖이다

   ③  **Source: progress 에 Bytes 0 이 나가는 길이 셋이다** (교차 검사가 찾았다).
       ㄱ  노드가 아직 첫 청크를 안 올렸다 (U3 의 R18)
       ㄴ  N2 의 쓸기가 마지막 쓰기로부터 6시간 뒤에 걷었다 (U3 의 Reap)
       ㄷ  Seal 의 DropProgress 와 verdict 굳히기 사이의 창
       **셋이 한 값으로 나간다.**  셋 다 틀린 값이 아니다 — 진행 파일을 봤고
       없었다는 사실이다.  가르는 것은 이 라우트가 아니라 NC-1 의 경과 표시다
       (U5 · U6).  어느 유닛의 모순 검사도 이것을 못 본다 — 셋이 다른 문서에 산다

   ④  한 응답의 상한 1 MiB 는 시작값이다.  NFR Requirements 가 N1 을 재고 굳힌다
```

## 8. 회차 밖으로 낼 것

```text
   unit-of-work-file-matrix.md 1절   internal/record/record.go 가 U3 하나다.
                                    U4 가 OpenLog 를 더한다 (R12).
                                    U3 가 병합돼 충돌은 0 이지만 행렬이 틀리다

   unit-of-work-file-matrix.md 1절   internal/transcript/** 가 U1 하나다.
                                    U4 가 json 태그를 단다 (R18).  동작 diff 0

   requirements.md 5.4 잔여 ②        토큰이 있다는 전제로 쓰였다.  데모에는 없다.
                                    잔여가 둘로 갈린다 — 실 함대와 데모

   component-methods.md 4.1          응답 본문의 상한과 개행 규칙(R7 · R8)이 없다.
                                    헤더 넷만으로는 폴링이 안 닫힌다
```
