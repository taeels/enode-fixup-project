# U7 `chunk-push` — Code Generation 요약

```text
   유닛    chunk-push (U7) · 브랜치 unit/chunk-push · 담당 taeels
   계획    ../../plans/chunk-push-code-generation-plan.md
   잰 날    2026-09-16 · 기준선 05ee710
   지는 게이트  CB3 — 사람이 실 함대에서 본다.  이 문서는 그 앞까지다
```

**잰 것만 적는다.** 안 돈 것과 못 잰 것은 6절이 이름으로 든다.

---

## 1. 무엇이 생겼나

| 파일 | 새것/고침 | 줄 |
|---|---|---|
| `internal/enode/upload.go` | **새 파일** | 300 |
| `internal/enode/upload_test.go` | **새 파일** | 309 |
| `internal/enode/claim.go` | 고침 | +51 −8 |
| `internal/enode/worker_test.go` | 고침 | +22 −1 |
| `internal/enode/worker_unix_test.go` | 고침 | +42 |
| `internal/api/getlog_test.go` | 고침 | +55 −2 |
| `internal/api/api.go` | 고침 | **+9 −1** |

```text
   go.mod · go.sum        diff 0.  새 의존 0
   internal/record        diff 0    internal/transcript  diff 0
   internal/panel         diff 0    internal/store       diff 0
   라우트                  api.go 18 그대로 — CB0 을 안 건드린다
   UploadLog              동작 diff 0.  부르는 자리만 한 줄 뒤로 밀렸다
```

---

## 2. 이 유닛이 고친 한 줄과 그것이 드러낸 것

```go
   전   if err != nil || attempt <= 0 { fail(w, 400, "invalid attempt") }
   후   if err != nil || attempt < 0  { fail(w, 400, "invalid attempt") }
```

**노드의 첫 시도가 `attempt 0` 이다** (`store/claim.go` 의 `Claim.Attempt` —
「0 부터. 재시도면 1 이상」). 그 한 줄이 **대부분의 단계가 청크를 한 바이트도
못 올리게** 막고 있었다.

**시험 하나가 그 버그를 정본으로 박고 있었다.** `TestPutLog_TheProgressBranch`
의 400 목록에 `attempt=0` 이 들어 있었다 — 이 유닛이 미는 쪽을 지으면서
**전체 시험이 그 줄에서 빨개졌고**, 그래서 뒤집었다.

```text
   뒤집은 것   400 목록의 attempt=0  ->  attempt=-1
   더한 것     TestPutProgress_TheFirstAttemptIsZero (200 이고 실제로 읽힌다)
              TestPutProgress_ANegativeAttemptIsStillRefused (음수 · 빈값 · 글자)
```

**왜 아무도 못 봤나** — 진행 청크를 미는 코드가 **이 유닛 전까지 0** 이었고,
U4 의 시험 다섯이 전부 `attempt=1` 이었다. 쓰는 쪽이 없는 표면은 자기가
틀렸다는 것을 말할 수 없다.

---

## 3. 실측이 계획을 고친 것 — 하나

### 3.1 전송 중인 바이트가 상한 계산에서 빠져 있었다

버퍼 상한 시험이 **넘칠 만큼 썼는데 안 넘쳤다.** 원인이 구조에 있었다.

```text
   take 가 버퍼를 통째로 떼어 간다   ->  그동안 u.buf 가 비어 보인다
   그 틈에 상한만큼이 또 쌓인다      ->  실제 메모리가 상한의 두 배가 된다
```

**시험이 잡은 진짜 결함이다.** `inflight` 필드를 더해 `take` 에서 세고
`putBack` · `settled` · `halt` 에서 지운다. 상한 검사가 `buf + inflight + 새것`
을 본다.

**W-b 의 U2 가 배운 것이 여기서 또 나왔다** — 「수를 세는 시험은 시점을 못
잰다」의 사촌이다. **양을 세는 시험은 그 양이 어디 있는지를 안 본다.**

---

## 4. 변이 일곱 — 다섯이 바로 빨갰고 둘이 살아남았다

```text
   ①  attempt 검증을 <= 0 으로 되돌린다     FAIL (시험 넷)
   ②  노드가 attempt+1 로 보낸다            FAIL (시험 여덟)
   ③  Write 에서 동기로 보낸다               FAIL (시험 여섯)
   ④  실패한 청크를 도로 안 붙인다            FAIL (시험 일곱)
   ⑤  Total 을 Attempt 보다 먼저 본다        **살아남았다. 안 고쳤다** (4.1)
   ⑥  버퍼 상한을 뺀다                      FAIL (시험 넷)
   ⑦  Close 를 UploadLog 뒤로               **처음에 살아남았다** -> FAIL (4.2)
```

### 4.1 ⑤ 는 규칙이 공허했다 — 시험을 안 짓고 규칙을 고쳤다

계획이 **「`Attempt` 를 `Total` 보다 먼저 본다」**(R8)를 불변식으로 적었고,
근거는 「버려진 청크의 `Total` 은 앞 시도의 총 길이라 그것으로 오프셋을 맞추면
뒤로 간다」였다.

**구현에서 그 갈래는 오프셋을 쓰지 않는다.** `ack.Attempt > 내 것` 이면
`halt()` 로 **멈추기 때문이다** — 멈춘 뒤에는 오프셋을 읽는 자리가 0 이다.

```text
   즉  멈추는 것이 순서를 삼켰다.  R8 은 R7(새 시도를 보면 멈춘다) 의 따름이지
      따로 지킬 것이 아니다
```

**시험을 지어 억지로 빨갛게 만들지 않았다.** 그러면 코드가 아니라 시험이 규칙을
만든다. **`business-rules` 의 R8 을 걷고 그 자리에 이 사실을 적었다.**

### 4.2 ⑦ 은 시험을 안 지어 살아남았다 — 지어서 잡았다

계획 Step 9 가 「단계 끝에 꼬리가 비워진 뒤 `UploadLog` 가 간다 (R17 — 순서를
잰다)」를 적었는데 **그 줄을 안 지었다.** 변이를 걸고서야 알았다.

**가짜 Mediator 가 두 갈래를 한 자리에 덮고 있었던 것도 함께 드러났다** —
`m.logs[name]` 하나에 진행 청크와 선별본이 같이 쓰였다. 그러면 **「두 벌이
안 생긴다」가 「한 벌만 생긴다」로 조용히 바뀐다.** 갈라서 순서를 기록하게 하고
`TestExecute_TheProgressTailLandsBeforeTheSealedLog` 를 지었다.

**그 시험이 두 가지를 함께 잰다** — 순서와, **두 파일이 실제로 다르다는 것.**

---

## 5. 게이트 — 진행자가 직접 쟀다

```text
   build · vet         초록          gofmt          0 줄
   glyphscan           112 파일 · 0   전체 시험      초록 · **스킵 0**
   커버리지 (정본 awk)    19 패키지 **미달 0** · 전체 7845/8954 = 87.6%
                       internal/enode  87.0% -> **87.4%**
                       internal/api    82.3% 그대로
   라우트               api.go **18 그대로** (CB0 을 안 건드린다)
   크로스 빌드           windows/amd64 OK · net/http T 6
   diff 0 이어야 할 곳    record · transcript · panel · store · go.mod · go.sum  전부 0
   git status          probe.lock 만 바뀌어 되돌렸다 (기준선의 성질 · 네 웨이브 연속)
```

---

## 6. 안 돈 것과 못 잰 것

```text
   CB3              이 유닛이 진다.  **사람이 실 함대에서 본다** —
                    curl 로 총 길이가 느는 것 · Mediator 를 10초 멈췄다 켜는 것 ·
                    봉인 뒤 tar 와 GET 이 같은 바이트인 것

   실물 하네스        안 돌렸다.  가짜 Mediator 와 셸 스크립트로만 쟀다

   1 MiB 의 적정성    쟀다기보다 골랐다.  CB3 의 10초가 실제 출력 속도에서
                    들어가는지는 실 함대가 잰다

   전역 한도          안 쟀다.  함대의 노드마다 2초에 한 번씩 민다 —
                    U4 가 N1 에서 봉투만 남긴 자리에 실제 부하가 여기서 붙는다.
                    값은 CB6 이 실 함대에서 잰다

   넘쳐서 멈춘 것을    **못 한다.**  넘치는 조건이 「Mediator 에 못 닿는다」라
   중앙에 알리는 길    못 닿는 채로 표시를 보낼 수 없다.  business-rules 8절 ①
```

---

## 7. 진행자에게 넘기는 것

계획 6절의 셋에 **하나가 는다.**

```text
   **api.go 의 attempt 검증**   이 유닛이 고쳤다.  **행렬 밖**이고 U4 가 병합돼
                              충돌 0 이었다 (W-b 가 같은 모양을 둘 겪었다)

   unit-of-work.md U7 절       「tee 의 갈래 하나」가 실제로는 둘이다

   unit-of-work.md U7 절       「Client.PutLogChunk 하나」로 적었는데
                              transcript() 의 시그니처도 함께 넓어졌다

   **business-rules R8 을      「Attempt 를 Total 보다 먼저 본다」가 공허했다.
   걷었다**                     멈추는 것이 순서를 삼킨다 (4.1).  설계 문서를
                              코드가 고친 자리이고, 그 판단의 정본이 이 요약뿐이다
```
