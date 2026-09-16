# U4 `log-api` — Functional Design 계획 (Part 1)

```text
   유닛     log-api (U4) · 웨이브 W-b · 브랜치 unit/log-api
   맡는 것   FR-6 (GET 진행 로그 하나) · FR-5 의 api 쪽 (PUT 의 갈림)
   지는 게이트  CB0 — grep -c 'mux.HandleFunc' internal/api/api.go 가 17 -> 18
   NFR      돈다.  N1 (함대 규모에서의 청크 PUT 과 폴링 부하)을 진다
   기준선    5d53e52.  W-a 가 병합된 main.  U3 의 진행 파일 겉면이 이미 있다
```

이 문서는 **묻는 것**이다. 답이 오면 4절의 순서로 산출물 셋을 낸다.

---

## 0. 이 단계가 닫는 것과 안 닫는 것

```text
   닫는다     라우트의 입력 검증과 응답 코드 · 헤더 넷의 값과 시점 ·
             봉인 전후를 가르는 술어 · 봉인 쪽 본문을 누가 여나 ·
             as=events 의 선 위 모양 · 한 번에 내는 바이트의 상한
   안 닫는다   코드.  Code Generation 이 낸다
             N1 의 값 — NFR Requirements 가 잰다
             노드 쪽 업로더 (U7) · 화면 (U5 · U6 · U8) · 파서의 안 (U1, 병합됨)
```

---

## 1. 실측 — 코드가 지금 어떻게 생겼나 (2026-09-16)

### 1.1 라우트가 17 이다 — CB0 의 앞 값을 쟀다

```text
   grep -c 'mux.HandleFunc' internal/api/api.go   ->  17
```

W-a 병합 뒤의 합본에서 쟀다. **이 유닛이 18 로 만든다.** 는 것은 GET 하나뿐이고
PUT 의 갈림은 라우트를 안 늘린다 (D2).

### 1.2 `putLog` 는 값을 버리고 204 를 낸다

```go
   api.go:887   if _, err := s.records.AppendLog(runID, seq, name, r.Body, s.cfg.Artifacts.MaxBlobBytes); err != nil {
   api.go:893   w.WriteHeader(204)
```

D4 가 **`AppendLog` 의 반환값 뜻을 총 길이로 바꿨고** U3 가 그것을 냈다
(`record.go:62` 의 주석 — 「붙인 뒤의 총 길이를 낸다 (D4)」). **그런데 이 자리가
그 값을 버린다.** 시그니처가 같아 컴파일이 안 깨지므로 아무도 안 본다.

물음 3 이 그 자리다 — 비진행 갈래도 헤더를 다나.

### 1.3 봉인 쪽 본문을 여는 겉면이 **없다**

`internal/record` 의 공개 겉면을 셌다.

```text
   있다   OpenBlob(runID, name) (io.ReadCloser, int64, error)   record.go:357
         ReadProgress(runID, seq, name, from) (Progress, io.ReadCloser, error)  progress.go:516
         Sealed(runID) bool                                     record.go:91
         Tar(runID, w)                                          record.go:242
   없다   logs/NN-<name>.log 를 여는 것.  0 개다
```

**봉인 뒤의 답은 `logs/NN-*.log` 에서 온다** (FR-6). 그것을 여는 자리가 없으므로
이 유닛이 만들거나, `internal/api` 가 경로를 직접 조립해야 한다. 경로 규칙은
`record.go:37` 의 `dir()` 과 `:64` 의 `fmt.Sprintf("%02d-%s.log", seq, safe(name))`
이고 **`safe()` 는 비공개다** (`record.go:44`). 직접 조립하면 그 함수가 두 벌이 된다.

**파일 행렬이 `internal/record/record.go` 를 U3 에만 줬다.** U3 는 이미 병합됐으니
충돌은 안 나지만 **행렬이 틀린 채로 남는다.** 물음 1 과 6절이 그 자리다.

### 1.4 U3 가 낸 진행 파일 겉면이 헤더 넷과 거의 1 대 1 이다

```go
   progress.go:516   func (s *Store) ReadProgress(runID string, seq int, name string,
                         from int64) (Progress, io.ReadCloser, error)
   Progress{ Total int64; Attempt int; Capped bool }
```

```text
   X-Enode-Log-Bytes    Progress.Total
   X-Enode-Log-Attempt  Progress.Attempt
   X-Enode-Log-Capped   Progress.Capped
   X-Enode-Log-Source   Sealed(runID) 가 가른다 — 이것만 record 의 다른 자리다
```

**U3 가 API 가 물을 것을 미리 낸 것이다.** 맞춰 보니 빈 칸이 0 이다.

U3 가 답을 이미 정한 자리도 함께 실측했다 — 이 유닛이 다시 안 묻는다.

```text
   R17   읽는 쪽은 언제나 가장 큰 시도를 본다.  attempt 인자가 없다
   R18   아직 아무것도 안 온 단계는 빈 본문 + 빈 상태.  404 가 아니다
   R19   from 이 Total 보다 커도 오류가 아니다.  빈 본문과 지금 상태를 준다
   R32   from 이 음수면 0 으로 본다.  검증은 HTTP 표면의 일이다 (R33)
```

**R33 이 이 유닛을 이름으로 가리킨다** — 검증은 여기서 한다.

### 1.5 봉인 술어가 둘이고 서로 다른 순간을 잰다

```go
   record.go:91    func (s *Store) Sealed(runID string) bool   // verdict.json 의 쓰기 권한이 꺼졌나
   api.go:879      if run.State == store.StateSucceeded || run.State == store.StateFailed  // 410
```

**둘 사이에 창이 있다** — Run 이 종료 상태가 됐지만 `Seal` 이 아직 안 돈 순간.
그 창에서 `Sealed` 는 거짓이고 진행 파일은 아직 있다 (`Seal` 이 `DropProgress` 를
가장 먼저 부른다 — `record.go:107` 의 주석).

D3 은 `Sealed(runID)` 로 가르라고 적었다. **그러면 그 창에서 `Source: progress`
가 나가고 그것이 맞는 답이다** — 파일이 실제로 그것이기 때문이다. 물음 4 가 이
자리를 명시로 닫는다.

### 1.6 보호 래퍼가 **데모에서 무인증**이다

```go
   api.go:82   read := func(h http.HandlerFunc) http.HandlerFunc {
                   if s.cfg.Demo { return s.limit(h) }   // 무인증 + 한도
                   return s.auth(h)                      // 실 함대는 오늘 그대로
               }
```

FR-6 이 「보호 규칙은 `GET /v1/runs` 와 같다」로 적었고 그것은 이 래퍼다.
**데모 모드에서는 토큰이 아예 없다.**

`requirements.md` 5.4 의 잔여 ② 는 **「토큰 하나라 주체를 못 가른다」**로 적혀
있다. 그 문장은 토큰이 있다는 전제다. **데모에서는 그 전제가 없다** — 무인증
GET 으로 하네스 원문이 나간다. 잔여 ② 가 덮지 않는 자리이고, 이 실측이 물음 2 다.

### 1.7 `transcript.Event` 에 json 태그가 **0** 이다

```go
   type Event struct { Kind Kind; Sub string; Line int; Text string; Name string
                       ID string; OK *bool; Cut int; Shell bool; ... }
```

`grep -c 'json:' internal/transcript/transcript.go` 가 **0** 이다. 그대로 마샬하면
선 위의 키가 `Kind` · `Sub` · `OK` 가 된다 — 이 저장소의 다른 응답은 전부
`snake_case` 다 (`api.go` 의 `run_id` · `step_id`).

**태그를 다는 것은 `internal/transcript` 를 만지는 일이고 행렬이 그 패키지를
U1 에만 줬다** (1.3 과 같은 모양의 자리다). 물음 5 가 그 자리다.

### 1.8 안 만져도 되는 것을 확인했다

```text
   GET /v1/runs/{id}/record 의 409    FR-6 이 명시로 안 건드린다 (api.go:1037)
   putBlob · getBlob                  이 유닛 밖
   internal/store                     이 유닛은 GetRun 만 부른다.  스키마 diff 0
   internal/transcript 의 파싱         U1 이 냈다.  이 유닛은 부르기만 한다
   MaxBlobBytes                       config.go:51 · 기본 10 MiB.  값을 안 바꾼다
```

---

## 2. 물음 여덟

형식은 `common/question-format-guide.md` 다. **답은 `[Answer]:` 뒤에 글자 하나.**

### Question 1 — 봉인된 `logs/` 를 누가 여나

1.3 의 자리다. 여는 겉면이 0 이고 경로 조립에 필요한 `safe()` 가 비공개다.

```text
   A   internal/record 에 OpenLog(runID, seq, name) (io.ReadCloser, int64, error)
       를 더한다.  OpenBlob 과 대칭이다.  파일 행렬이 record.go 를 U3 에만
       준 것을 이 유닛이 틀렸다고 적는다
   B   internal/api 가 경로를 직접 조립한다.  safe() 와 "%02d-%s.log" 가 두 벌이 된다
   C   Tar 를 풀어서 뽑는다.  제어판이 오늘 하는 우회이고 FR-4 가 그것을 없애려 한다
```

**권장은 A 다.** 경로 규칙의 정본이 하나여야 한다 — B 는 `record` 가 파일 이름을
바꾸는 날 `api` 가 조용히 빈 본문을 낸다. C 는 이 회차가 없애려는 바로 그 우회다.

**A 의 대가** — 행렬이 틀린 것으로 확정되고 6절에 는다. U3 가 이미 병합돼 있어
**병합 충돌은 0** 이다.

[Answer]:

### Question 2 — 데모 모드의 무인증 노출을 받아들이나

1.6 의 실측이다. `read()` 를 쓰면 **데모 인스턴스에서 하네스 원문이 무인증 GET
으로 나간다.** `requirements.md` 5.4 의 잔여 ② 는 토큰이 있다는 전제로 쓰였다.

```text
   A   read() 를 쓴다.  FR-6 의 글자 그대로다.  잔여를 하나 새로 적는다 —
       「데모 모드에서는 무인증이다」
   B   이 라우트만 s.auth 로 잠근다.  데모 현황판이 카드를 못 그려 CB4 가 죽는다
   C   데모에서는 as=events 만 열고 as=raw 를 잠근다.  본문은 여전히 나간다
```

**권장은 A 다.** B 는 이 회차가 사려는 값(CB4 · CB1)을 직접 깬다. C 는 막는 것이
거의 없으면서 갈래를 하나 늘린다 — `events` 의 `Text` 가 곧 본문이다.

**A 를 고르면 이름으로 적는 것** — 데모 인스턴스에 올리는 계약은 비밀을 안 담는
것이어야 한다. 이것은 코드가 아니라 운영의 규칙이라 문서가 진다.

[Answer]:

### Question 3 — PUT 의 비진행 갈래도 헤더를 다나

1.2 의 자리다. `AppendLog` 가 이제 총 길이를 내는데 `api.go:887` 이 버린다.

```text
   A   두 갈래 다 X-Enode-Log-Bytes 를 단다.  비진행 갈래는 Attempt · Capped 가
       없으므로 그 둘은 안 단다
   B   진행 갈래만 헤더를 단다.  오늘의 204 를 그대로 둔다
```

**권장은 A 다.** D4 가 뜻을 바꾼 값이 아무 데도 안 나가면 **그 변경이 코드에만
있고 표면에는 없다.** 그리고 `logs/` 쪽도 폴링하는 쪽이 다음 `from` 을 알아야
한다 — 봉인 전에 `logs/` 를 읽는 경로는 없지만 PUT 을 보내는 노드는 자기가 쓴
총 길이를 알 이유가 있다.

[Answer]:

### Question 4 — 봉인 전후를 무엇으로 가르나

1.5 의 창이다.

```text
   A   Sealed(runID) 하나로 가른다 (D3 그대로).  Run 이 종료 상태여도 Seal 이
       아직 안 돌았으면 Source: progress 다 — 파일이 실제로 그것이다
   B   run.State 로 가른다.  종료 상태면 sealed 라고 말한다
```

**권장은 A 다.** 헤더가 말하는 것은 **어느 파일을 읽었나**이지 Run 의 상태가
아니다. B 를 고르면 그 창에서 `Source: sealed` 인데 본문은 진행 파일에서 온
원문이고, 읽는 쪽이 「선별본이다」로 알고 `from` 을 이어 쓴다.

[Answer]:

### Question 5 — `as=events` 의 선 위 모양

1.7 의 자리다. `transcript.Event` 에 json 태그가 0 이다.

```text
   A   internal/transcript 의 Event 에 json 태그를 단다.  화면 셋이 같은 키를 본다.
       행렬이 그 패키지를 U1 에만 준 것을 이 유닛이 틀렸다고 적는다
   B   internal/api 가 자기 DTO 로 옮겨 적는다.  필드가 늘 때마다 두 자리를 고친다
   C   태그 없이 그대로 마샬한다.  키가 Kind · OK 가 되어 이 저장소의 다른
       응답과 모양이 갈린다
```

**권장은 A 다.** 물음 1 과 같은 이유다 — 모양의 정본이 하나여야 한다. B 는
`Event` 에 필드가 하나 늘 때 API 가 조용히 그것을 안 내는 길이다.

**A 의 대가** — `internal/transcript` 를 U1 이 아닌 유닛이 만진다. 값은 태그뿐이고
동작이 안 바뀌며, U1 의 왕복 시험이 그대로 받친다.

[Answer]:

### Question 6 — 한 번에 내는 바이트에 상한을 거나

`from` 이 0 이고 진행 파일이 10 MiB 면 한 응답이 10 MiB 다. `as=events` 면
Mediator 가 그것을 통째로 파싱한다. **N1 이 재는 부하가 여기서 난다.**

```text
   A   한 번에 내는 바이트에 상한을 건다 (시작값 1 MiB).  X-Enode-Log-Bytes 는
       총 길이 그대로라 화면이 from 을 밀어 이어 받는다
   B   상한을 안 건다.  진행 파일의 총 길이 상한(10 MiB)이 이미 천장이다
```

**권장은 A 다.** B 의 천장은 **한 요청당**이 아니라 파일당이고, 폴링하는 화면이
여럿이면 곱해진다. A 는 이미 있는 `from` 규약 위에 얹히므로 새 개념이 0 이다.

**값(1 MiB)은 시작값이고 NFR Requirements 가 N1 을 잴 때 굳힌다.**

[Answer]:

### Question 7 — 없는 단계의 `seq` 에 무엇을 답하나

`ReadProgress` 는 R18 로 「빈 본문 + 빈 상태」를 낸다 — 그 층은 늦은 노드와 없는
단계를 못 가른다. **API 는 계약을 안다** (`GetRun` 이 단계 목록을 준다).

```text
   A   200 에 총 길이 0.  design 4.1 의 「파일만 없으면 200 에 총 길이 0」 그대로
   B   그 Run 의 단계 수를 넘는 seq 는 404.  아직 안 온 단계만 200 에 0
```

**권장은 A 다.** B 는 화면이 단계 목록을 먼저 받아야 폴링을 시작할 수 있다는
뜻인데 화면은 이미 그렇게 돌고 있고, **틀린 seq 를 404 로 가르는 값이 화면에
0 이다.** A 는 갈래가 하나 적다.

**B 를 고르면 얻는 것** — 오타를 부르는 쪽이 바로 안다.

[Answer]:

### Question 8 — `name` 이 없을 때

`putLog` 는 빈 `name` 을 `"step"` 으로 채운다 (`api.go:884`). GET 은 같은 규칙을
따를 것인가.

```text
   A   같은 기본값 "step" 을 쓴다.  PUT 과 GET 이 같은 파일을 가리킨다
   B   name 이 없으면 400 이다
```

**권장은 A 다.** 두 표면이 같은 파일 이름 규칙을 쓰는데 한쪽만 기본값이 있으면
**PUT 이 쓴 것을 GET 이 못 찾는 조합이 생긴다.** 그 조합은 시험이 아니라 운영에서
난다.

[Answer]:

---

## 3. 안 묻는 것 — 이미 값이 있는 자리

```text
   헤더 이름 넷            component-methods.md 4.1 이 값으로 닫았다
   라우트 문자열           GET /v1/runs/{run}/steps/{seq}/log · PUT 의 ?progress=1&attempt=
   from 의 경계 동작       U3 의 R19 · R32 가 닫았다.  API 는 음수를 400 으로 막는다 (R33)
   가장 큰 시도만 읽는다    U3 의 R17
   판정을 안 싣는다         ADR-065.  FR-6 이 명시로 적었다
   GET record 의 409       안 건드린다 (FR-6)
   as 의 값                raw · events 둘뿐.  그 밖은 400 (SECURITY-05)
   N1 의 값                NFR Requirements 가 잰다.  여기서는 모양만 정한다
```

---

## 4. 실행 단계 — 답을 받은 뒤

### 4.1 순서

```text
   ①  답 여덟을 계획 2절에 적는다
   ②  모순 검사 — 답끼리 부딪치는 자리를 센다 (Functional Design Step 5)
   ③  domain-entities.md      요청 · 응답 · 헤더 넷 · 출처 둘 · 검증 규칙
   ④  business-logic-model.md 갈림의 순서 (Sealed -> 출처 -> 본문 -> 헤더)
                              PUT 의 두 갈래 · 실패 코드의 사다리
   ⑤  business-rules.md       불변식에 번호를 단다.  CB0 이 잴 것을 값까지
   ⑥  W-b 교차 검사 — U2 의 답과 맞댄다 (아래 4.2)
```

### 4.2 웨이브를 닫는 자리의 교차 검사 — W-a 가 배운 것

**유닛 경계를 넘는 답의 조합은 어느 한 유닛의 모순 검사도 못 본다.** 맞댈 자리를
미리 이름으로 적는다.

```text
   사건 종류의 어휘   U4 의 물음 5 (as=events 의 키)와 U2 의 물음 2 (EventKind).
                    U2 가 자기 어휘를 따로 만들면 노드 로그와 API 가 다른 글자를 쓴다
   총 길이의 뜻      U2 의 링 total 과 U4 의 진행 파일 Total 은 다른 값이다.
                    U5 가 한 화면에 둘 다 그린다 — 이름이 같으면 그 화면이 틀린다
   시도(attempt)     U2 의 Ring gen 과 U4 의 X-Enode-Log-Attempt.  U7 이 둘을 잇는다
   상한의 단위       U4 의 물음 6 (한 응답의 바이트)과 U3 의 총 길이 상한(파일).
                    W-a 가 단위가 갈려 한 번 틀렸던 자리와 같은 모양이다
```

### 4.3 산출물 셋

```text
   construction/log-api/functional-design/domain-entities.md
   construction/log-api/functional-design/business-logic-model.md
   construction/log-api/functional-design/business-rules.md
```

---

## 5. 이 단계가 안 만드는 것

```text
   코드 · 시험        Code Generation
   N1 의 값           NFR Requirements (이 유닛이 곧 돈다)
   인프라 설계         회차 계획 SKIP
   노드 쪽 업로더      U7
   화면              U5 · U6 · U8
   runctl 의 StepLog  U6 (application-design 1.2 가 찾은 일곱째 경로)
```

---

## 6. 회차 밖으로 낼 것 — 지금까지 확정된 것 셋

```text
   unit-of-work-file-matrix.md 1절   internal/record/record.go 가 U3 하나로 적혀 있다.
                                    U4 가 봉인 쪽 본문을 열려면 거기를 만진다 (물음 1 = A).
                                    U3 가 이미 병합돼 충돌은 0 이지만 행렬이 틀리다

   unit-of-work-file-matrix.md 1절   internal/transcript/** 가 U1 하나로 적혀 있다.
                                    as=events 의 키를 정본에 두려면 거기를 만진다 (물음 5 = A)

   requirements.md 5.4 의 잔여 ②     「토큰 하나라 주체를 못 가른다」는 토큰이 있다는
                                    전제다.  데모 모드에는 토큰이 없다 (1.6).
                                    잔여가 둘로 갈린다 — 실 함대와 데모
```

**답이 오면 는다.** 물음 6 의 답이 A 면 응답 상한이라는 값 하나가 새로 생기고,
그 값의 정본 자리를 NFR Requirements 가 정해야 한다.
