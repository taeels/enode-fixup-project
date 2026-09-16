# U7 `chunk-push` — 도메인 개체

```text
   유닛    chunk-push (U7) · 브랜치 unit/chunk-push
   계획    ../../plans/chunk-push-functional-design-plan.md
   답      여덟이 전부 A
   기준선   05ee710
```

**받는 쪽이 이미 다 서 있다.** 이 유닛은 **보내는 쪽 하나**를 짓고, 그 길을
막고 있던 **검증 한 줄**을 고친다.

---

## 1. 고치는 한 줄 — `attempt 0` 을 받는다

답 1 = A. **이 유닛의 착수 조건이다.**

```go
   internal/api/api.go   putProgress
   전    if err != nil || attempt <= 0 { fail(w, 400, "invalid attempt") }
   후    if err != nil || attempt < 0  { fail(w, 400, "invalid attempt") }
```

```text
   왜 그 줄이 틀렸나   바로 위의 seq 검사를 따라 썼는데 두 수의 바닥이 다르다 —
                    seq 는 1 부터이고 attempt 는 0 부터다 (store/claim.go:156)
   누가 0 을 보내나    노드의 첫 시도.  대부분의 단계다
   record 는 어떤가    0 을 정상으로 다룬다.  progressFile 이 그냥 쓰고
                    maxAttempt 가 best, found := 0, false 로 시작한다
   행렬              internal/api/api.go 는 U4 의 것이다.  **행렬 밖**이고
                    U4 가 병합됐으므로 충돌 0 (W-b 가 같은 모양을 둘 겪었다)
```

**시험이 하나 는다** — `attempt=0` 을 밟는 줄이 저장소에 0 이다. 그것이 U4 가
이 자리를 못 본 이유다.

---

## 2. 새 겉면 하나 — `Client.PutLogChunk`

```go
   // internal/enode
   func (c *Client) PutLogChunk(ctx context.Context, runID string, seq int,
       name string, attempt int, body []byte) (LogChunkAck, error)

   type LogChunkAck struct {
       Total   int64  // X-Enode-Log-Bytes.   서버가 든 진실이다
       Attempt int    // X-Enode-Log-Attempt. 내 것보다 크면 내 청크가 버려졌다
       Capped  bool   // X-Enode-Log-Capped.  상한에 닿았다
   }
```

**`UploadLog` 를 안 고친다** — 그쪽은 단계 끝의 선별본을 `logs/` 에 올린다
(`claim.go:796`). 쿼리가 다르고 파일이 다르므로 **두 벌이 안 생긴다.**

```text
   PutLogChunk   PUT .../log?name=<n>&progress=1&attempt=<a>   도는 동안 · 원문
   UploadLog     PUT .../log?name=<n>                          단계 끝 · 선별본
```

**응답 셋을 다 읽는 것이 이 겉면의 값이다.** 앞의 것(`UploadLog`)은 상태 코드만
보고 몸통을 버린다. 이쪽은 셋이 다 규율을 진다 (business-rules R6 · R8 · R10).

---

## 3. 새 타입 하나 — `uploader`

답 3 · 4 · 6 · 7 이 함께 짓는다. `internal/enode/upload.go` 에 산다.

```go
   type uploader struct {
       // 단계 하나가 수명이다 (답 6 = A).  링의 Reset 과 같은 자리
       runID   string
       seq     int
       name    string
       attempt int

       mu   sync.Mutex
       buf  []byte   // 아직 못 보낸 것.  상한 1 MiB
       done bool     // 넘쳤거나 상한에 닿았다.  더 안 받는다

       ch chan struct{}  // 64 KiB 가 쌓이면 즉시 깨운다
   }
```

**`Write` 가 이 타입의 유일한 공개 면이다** (`io.Writer`). 버퍼에 넣고 바로
돌아온다 — **네트워크를 절대로 안 기다린다** (답 7 = A).

**오프셋 필드가 여기 없는 것이 값이다.** 보내는 고루틴의 지역 변수로만 산다
(business-rules R9). 공유 상태가 하나 줄고, `Write` 가 오프셋을 볼 일이 0 이다.

---

## 4. 넓어지는 비공개 면 하나 — `transcript()`

```go
   전   func (w *Worker) transcript() io.Writer
   후   func (w *Worker) transcript(step *Step) (io.Writer, func())
```

```text
   왜 인자가 필요한가   업로더가 runID · seq · name · attempt 넷을 알아야 한다.
                    링은 노드에 하나라 몰라도 됐다 (계획 1.3)
   왜 둘을 내나       업로더는 단계 끝에 마지막으로 비우고 멈춰야 한다.
                    그 정리를 부르는 쪽이 defer 로 든다
   겉면인가          아니다.  소문자다 — 패키지 밖에서 안 보인다
```

**부르는 자리가 둘이다** (답 2 = A) — `claim.go:794` 의 에이전트 단계와
`claim.go:632` 의 명령 단계. **규칙이 하나다: 링으로 가는 것은 Mediator 로도 간다.**

---

## 5. 선 위 모양 — 이 유닛이 새 타입을 0 개 들인다

```text
   보내는 것    하네스 stdout 의 바이트 그대로.  줄도 안 나눈다
   받는 것      헤더 셋 (Bytes · Attempt · Capped).  U4 가 이미 낸다
   새 type     0 개.  enode.capped 는 **서버가 박는다** (record/progress.go:53)
```

**어휘가 안 넓어지는 것이 이 유닛의 성질이다.** `internal/transcript` 의 Kind
일곱이 그대로이고, 화면 셋이 그릴 것이 하나도 안 는다.

---

## 6. 값 둘 — NFR 단계가 없으므로 여기서 선다

```text
   버퍼 상한    **1 MiB**
               아래에서 민 것 — CB3 이 「Mediator 를 10초 멈췄다 켠다」를 잰다.
                              그 10초가 안 들어가면 게이트가 이 값 때문에 빨개진다
               위에서 누른 것 — 노드의 힙.  관측하는 겹이 실행의 메모리를 위협하면 안 된다
               안 고른 값 — 링과 같은 512 KiB.  한 숫자로 맞추는 이득보다
                          CB3 의 여유가 크다.  **CB3 이 이 값을 실제로 밟는다**

   주기 · 문턱   2초 · 64 KiB.  decisions.md 2절 그대로.  이 유닛이 안 정한다
```

---

## 7. 안 짓는 것

```text
   내려받는 쪽        U4 가 닫았다.  이 유닛은 PUT 만 한다
   Mediator 쪽 저장   U3 가 닫았다.  internal/record 의 diff 가 0 이다
   단계 끝 선별본      claim.go:796 그대로
   화면             U5 · U6 · U8
   새 라우트          0.  기존 PUT 에 쿼리 하나 (U4 가 닫았다)
   새 설정 키         0.  버퍼 상한은 상수다 — 조절 손잡이가 아니라 보호다
```
