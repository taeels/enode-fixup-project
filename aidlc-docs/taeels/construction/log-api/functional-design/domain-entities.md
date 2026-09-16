# U4 `log-api` — 도메인 개체

```text
   유닛    log-api (U4) · 웨이브 W-b · 브랜치 unit/log-api
   계획    ../../plans/log-api-functional-design-plan.md (답 여덟 전부 A)
   기준선   5d53e52.  W-a 가 병합된 main.  U3 의 진행 파일 겉면이 이미 있다
```

---

## 1. 신규 라우트 하나

```text
   GET /v1/runs/{run}/steps/{seq}/log?from=<바이트>&as=<raw|events>&name=<이름>
```

**라우트 수가 17 에서 18 이 된다** — CB0 이 그 수를 잰다. 는 것은 이 GET 하나뿐이다.

### 1.1 입력

| 자리 | 규칙 | 어기면 |
|---|---|---|
| `{run}` | 경로 성분 | 없는 Run 은 404 |
| `{seq}` | 양의 정수 | 400 |
| `from` | 음이 아닌 정수. 없으면 0 | 400 |
| `as` | `raw` · `events` 둘뿐. 없으면 `raw` | 400 |
| `name` | 없으면 `step` (PUT 과 같은 기본값) | — |

**`name` 의 기본값이 PUT 과 같은 것이 값이다** (`api.go:884`). 한쪽만 기본값이
있으면 **PUT 이 쓴 것을 GET 이 못 찾는 조합**이 생기고, 그 조합은 시험이 아니라
운영에서 난다.

### 1.2 응답 헤더 넷

```text
   X-Enode-Log-Bytes     총 길이.  이 응답의 본문 길이가 아니다
   X-Enode-Log-Source    progress | sealed        D3 · NC-5
   X-Enode-Log-Attempt   지금 시도                 D7 의 대가를 닫는다
   X-Enode-Log-Capped    1 이면 상한에 닿았다       NC-4
```

**`Bytes` 가 본문 길이가 아닌 것이 이 설계의 축이다.** 본문은 `from` 부터 상한
까지의 조각이고, `Bytes` 는 파일 전체의 길이다. 폴링하는 쪽이 「더 있나」를 그
둘의 차로 안다.

### 1.3 응답 코드

```text
   200   본문이 있거나 · 비었거나.  파일이 아직 없어도 200 에 Bytes: 0 이다
   400   seq · from · as 가 규칙 밖 (SECURITY-05)
   404   그 Run 이 없다.  그것 하나다
   503   저장소가 답을 못 한다
```

**없는 `seq` 는 404 가 아니다** (답 7 = A). 화면은 단계 목록을 이미 들고 폴링을
걸고, **틀린 `seq` 를 404 로 가르는 값이 화면에 0 이다.**

---

## 2. 출처 둘

```text
   progress   <Root>/progress/run-<safe(id)>/NN-<safe(name)>.log     원문
   sealed     <Root>/run-<safe(id)>/logs/NN-<safe(name)>.log         선별본
```

**가르는 것은 `records.Sealed(runID)` 한 줄이다** (답 4 = A · D3). Run 의 상태가
아니다 — 헤더가 말하는 것은 **어느 파일을 읽었나**이지 Run 이 끝났나가 아니다.

`Sealed` 는 `verdict.json` 의 쓰기 권한이 꺼졌는가다 (`record.go:91`). Run 이
종료 상태인데 `Seal` 이 아직 안 돈 창에서는 **`progress` 가 맞는 답이다** —
`Seal` 이 `DropProgress` 를 가장 먼저 부르므로 그 창에서는 진행 파일이 아직 있다.

---

## 3. 새로 생기는 겉면 하나 — `record.OpenLog`

```go
// OpenLog 는 봉인된 단계 로그를 연다. OpenBlob 과 대칭이다.
func (s *Store) OpenLog(runID string, seq int, name string) (io.ReadCloser, int64, error)
```

**`internal/api` 가 경로를 직접 조립하지 않는다** (답 1 = A). 조립에 필요한
`safe()` 가 비공개이고 (`record.go:44`), 공개하거나 베끼면 **경로 규칙이 두 벌이
된다** — `record` 가 이름을 바꾸는 날 `api` 가 조용히 빈 본문을 낸다.

**파일 행렬이 `internal/record/record.go` 를 U3 하나에만 줬다.** U3 는 이미
병합돼 병합 충돌은 0 이지만 **행렬이 틀리다** — `business-rules.md` 7절이 든다.

---

## 4. 이미 있는 겉면 셋 — U3 가 낸 것

```go
   ReadProgress(runID, seq, name string, from int64) (Progress, io.ReadCloser, error)
   Sealed(runID string) bool
   AppendProgress(runID, seq, name string, attempt int, r io.Reader, limit int64) (Progress, error)

   type Progress struct { Total int64; Attempt int; Capped bool }
```

**`Progress` 의 셋이 헤더 넷 중 셋과 1 대 1 이다.** 빈 칸이 0 이다 — U3 가
API 가 물을 것을 미리 냈다.

U3 가 이미 닫은 규칙은 이 유닛이 다시 안 정한다.

```text
   R17   읽는 쪽은 언제나 가장 큰 시도를 본다.  attempt 인자가 없다
   R18   아직 아무것도 안 온 단계는 빈 본문 + 빈 상태.  404 가 아니다
   R19   from 이 Total 보다 커도 오류가 아니다
   R32   from 이 음수면 0 으로 본다.  검증은 HTTP 표면의 일이다 (R33)
```

---

## 5. 갈리는 기존 라우트 하나

```text
   PUT /v1/runs/{run}/steps/{seq}/log                        logs/ 에 붙인다
   PUT /v1/runs/{run}/steps/{seq}/log?progress=1&attempt=N   진행 파일에 붙인다
```

**라우트가 안 는다** (D2). 갈래는 `progress` 쿼리 하나가 정한다.

```text
   두 갈래 다      X-Enode-Log-Bytes 를 단다 (답 3 = A).  204 가 200 이 된다
   진행 갈래만     X-Enode-Log-Attempt · X-Enode-Log-Capped
```

**비진행 갈래도 헤더를 다는 이유** — D4 가 `AppendLog` 의 반환값 뜻을 총 길이로
바꿨는데 `api.go:887` 이 그 값을 버린다. 안 내보내면 **그 변경이 코드에만 있고
표면에는 없다.**

---

## 6. `as=events` 의 선 위 모양

**`internal/transcript` 의 `Event` 에 json 태그를 단다** (답 5 = A).

```go
type Event struct {
    Kind Kind   `json:"kind"`
    Sub  string `json:"sub,omitempty"`
    Line int    `json:"line"`
    Text string `json:"text,omitempty"`
    Name string `json:"name,omitempty"`
    ID   string `json:"id,omitempty"`
    OK   *bool  `json:"ok,omitempty"`
    Cut  int    `json:"cut,omitempty"`
    Shell bool  `json:"shell,omitempty"`
}
```

**`OK` 에 `omitempty` 를 다는 것이 뜻을 지킨다** — 포인터라 nil 이면 키가 빠지고,
`false` 와 「없음」이 선 위에서도 갈린다. 없으면 「실패한 도구가 있었다」로 읽힌다
(`transcript.go` 의 `OK` 주석).

**동작이 안 바뀐다.** 태그는 마샬 이름뿐이고 U1 의 왕복 시험이 그대로 받친다.
**파일 행렬이 `internal/transcript/**` 를 U1 하나에 줬다** — 3절과 같은 모양의
자리이고 7절이 함께 든다.

응답 몸통.

```json
{ "events": [ ... ], "elided": {...}, "raw": 0, "lines": 12, "head": 0, "partial": 41 }
```

`transcript.Result` 를 그대로 낸다. **다음 `from` 은 `partial` 로 고치는 것이
아니라 언제나 `from + len(본문)` 이다** — 서버가 상한에 걸릴 때 마지막 개행에서
끊기 때문이다 (`business-rules.md` R7 · R8).

`partial` 이 0 이 아닌 것은 **한 줄이 상한보다 길다**는 뜻이고 그때만 난다.
화면이 그 자리를 다르게 그린다.
