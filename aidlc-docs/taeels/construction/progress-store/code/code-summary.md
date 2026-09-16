# U3 `progress-store` — Code Generation 요약

**유닛** `progress-store` (U3) · **브랜치** `unit/progress-store` ·
**담당** taeels · **웨이브** W0 · **지는 값** N2

계획은 `aidlc-docs/taeels/construction/plans/progress-store-code-generation-plan.md`
이고 이 문서는 그 Part 2 의 산출 요약이다. **잰 값만 적는다** — 안 돈 것과
못 잰 것을 9절이 이름으로 적는다.

---

## 0. 이 단계가 중간에 끊겼다가 이어졌다

**앞 회차가 구독 만료로 죽었다.** 그때 커밋된 코드가 0 이었고 작업 트리에만
살아 있었다 — 제품 코드 수정 셋(102 줄)과 새 파일 둘(1,507 줄). 이어받은
회차가 한 일의 순서가 이렇다.

```text
   ①  있는 것을 먼저 커밋했다      빨간 시험째로.  bdb13a3
                                 날아갈 뻔한 것을 역사에 먼저 넣었다
   ②  빨간 시험 하나를 닫았다       26bf7b4.  판정은 2절
   ③  안 지어진 Step 15 를 지었다   efc0042.  시험 여덟 · DB 를 탄다
   ④  계획이 요구한 주석 둘을 채웠다  ebb7d7a.  2.6 과 8.9
```

**앞 회차가 계획의 체크박스를 하나도 안 채웠다.** 규칙이 「끝낸 그 자리에서
`[x]`」로 걸었는데 안 지켜졌고, 이 회차가 실측으로 되짚어 채웠다 — 그 되짚기가
Step 15 가 통째로 비어 있다는 것을 찾은 경로다.

---

## 1. 낸 파일 — 여섯

| 파일 | 새것/고침 | 줄 |
|---|---|---|
| `internal/record/progress.go` | **새 파일** | 713 |
| `internal/record/progress_test.go` | **새 파일** | 823 |
| `internal/store/progress_test.go` | **새 파일** | 291 |
| `internal/record/record.go` | 고침 | +45 −7 |
| `internal/store/reap.go` | 고침 | +55 |
| `internal/store/seal.go` | 고침 | +9 |

**계획 1.1 의 표와 파일 하나까지 같다.** 제품 파일 넷(새 파일 하나) · 시험
파일 둘 · 이 요약 하나.

```text
   diff 전체   6 파일 · 1,929 삽입 · 7 삭제   (53ef404..ebb7d7a)
```

---

## 2. 빨간 시험을 어느 쪽으로 닫았나 — **시험이 틀렸다**

이어받은 시점에 `internal/record` 가 빨갰다.

```text
   TestTheFourGapsConverge/before_the_cut_write
   progress_test.go:529  the gap did not converge: capped=false body="aaaa\nzzzz\n"
```

**코드가 아니라 시험이 틀렸다.** 근거는 산수 하나다.

```text
   씨앗    "aaaa\n"    5 바이트
   청크    "zzzz\n"    5 바이트
   상한    20
   합      10  <  20   ->  상한에 닿은 적이 없다
```

닿은 적이 없으니 표시 줄이 안 나고 `Capped` 가 거짓인 것이 **맞는 답**이다.
나머지 세 씨앗은 전부 상한 위였다 (20 · 24 · 45) — 첫째만 그 자리를 안 지났다.

정본은 `nfr-design-patterns.md` 1.3 의 첫 행이다.

```text
   ③ 앞 | 상한 이하 · 표시 줄 없음 | Capped 거짓 | 같은 판정을 다시 해서 ③ ~ ⑤ 를 돈다
```

**「같은 판정을 다시 해서」가 그 호출이 상한을 넘긴다는 뜻이다.** 씨앗만 두고
청크를 안 키운 것이 시험의 틈이었다. 틈마다 청크를 따로 두고 첫째를 예산 안에
개행이 있는 청크로 넘기게 고쳤다 — **③ 이 실제로 자르는 유일한 자리**가 됐다
(나머지 셋은 예산이 0 이하라 ③ 이 0 바이트다).

함께 강화한 것 — 네 자리 모두 **기대 바이트를 통째로 대조**한다. 자름(R11) ·
개행 보장(R43) · 닿은 뒤 안 쓰기(R15) 가 한 대조에 걸린다.

**제품 코드를 0 줄 고쳤다.**

---

## 3. 잰 값 — 계획 6절의 순서 그대로

```text
    2  go build ./...  ·  go vet ./...            초록
    3  go test -race ./internal/record/...        초록  (1.2초)
    4  go test ./...                              rc=0 · ok 18 패키지
    5  go run ./scripts/glyphscan.go              초록 · 105 파일 · 0 건
    6  test -z "$(gofmt -l .)"                    빔
    7  test -z "$(git status --porcelain)"        빔
    8  커버리지 (패키지별 80% 하한)                  미달 0
    9  git checkout -- cmd/enodectl/probe.lock    되돌렸다
   10  grep -c 'mux.HandleFunc' internal/api/api.go   **17** — 안 늘었다
```

### 3.1 커버리지 — 하한 미달 0

```text
   전체              7,401/8,468  =  87.4%
   internal/record     384/455    =  84.4%     기준선 82.4% (131/159)
   internal/store    1,478/1,788  =  82.7%     기준선 82.5% (1,453/1,762)
```

**계획 6.1 의 예산 안이다.**

```text
   record   새 문장 296.  예산이 허용한 새 미달 0.2n + 3.8 = 63.0
            실제 새 미달 43  (미달 28 -> 71).  **여유 20 문장**
   store    새 문장 26.   예산 0.2n + 43.4 = 48.6
            실제 새 미달 1   (미달 309 -> 310)
```

`internal/record` 의 여유가 3.8 문장이라 얇다고 적었던 자리인데, 시험이 오류
가지를 함께 덮어 **82.4 -> 84.4 로 올랐다**. 권한으로 실패를 만드는 시험은 안
썼다 — `t.TempDir()` 안에 파일과 디렉터리를 자리바꿈해 `MkdirAll` 과
`OpenFile` 을 실패시켰다 (root 로 돌아도 발동한다 · 새 `t.Skip` 0).

### 3.2 차단 게이트 다섯 (품질 게이트 1)

```text
   커버리지 80%    미달 0 (3.1)
   스킵            **0**.  .ci-allowed-skips 의 비주석 줄도 0 — 빈 것이 결론이다
   U+2605          **0**  (grep -rlIP 가 빈다)
   심볼 상한 둘     enodectl  crypto/tls=1 · net/http=6   (상한 10 · 50)
                   **안 움직였다** — internal/record 가 그 둘을 안 쓴다 (실측 0 줄)
```

### 3.3 품질 게이트 2 — diff 0 과 깨지는 하나

```text
   internal/contract   0      internal/api      0      internal/enode   0
   internal/panel      0      internal/api/ui   0      cmd/ 다섯         0
   go.mod              0      go.sum            0      schema.sql        0

   internal/store      **0 이 아니다** — seal.go +9 · reap.go +55
                       FD 10절이 확정으로 적은 자리다.  기준선이 바뀐 것이지
                       게이트가 빨간 것이 아니다.  고치는 사람은 진행자다
```

### 3.4 게이트 4 에서 본 흔들림 하나 — **숨기지 않는다**

`go test ./...` 를 이 단계에서 **아홉 번** 돌렸다.

```text
   초록  일곱      그중 마지막 다섯 번이 연속이다
   빨강  **둘**    연속으로 났고 **전부 internal/api** 다.  그 뒤로 재현 안 됐다
                 증상 — deadlock detected (SQLSTATE 40P01) 과
                 통합 시험들의 404 · 422 · 503 이 무더기로
```

**이 유닛의 것이라고 볼 근거와 아니라고 볼 근거를 둘 다 적는다.**

```text
   아니라고 볼 근거
     internal/api 는 이 유닛의 코드 diff 가 **0** 이다 (3.3)
     DB 를 타는 묶음(api · store · cmd/mediator)만 여섯 번 돌려 **6/6 초록**
     저장소가 이미 이 실패 양식을 적어 뒀다 — reap_deterministic_test.go 의
       머리주석이 「internal/api 의 통합 78개가 같은 enode_test 를 쓰고
       함께 돌리면 3회 중 2회가 깨졌다」로 실측을 남겼다.  같은 증상이다
     찌꺼기 scratch DB 가 0 개다 (실측 — pg_database 에 u7_reap% 가 0)

   그렇다고 볼 근거
     이 유닛이 internal/store 에 scratch DB 를 **여덟 개 더** 만든다
     (progressTestStore -> reapTestStore -> CREATE DATABASE).
     그 패키지의 기존 넷에서 **셋 배로 늘었다.**  CREATE/DROP DATABASE 는
     서버 전체에 부하를 주므로 enode_test 를 쓰는 internal/api 를 흔들 수 있다
```

**어느 쪽인지 못 갈랐다.** 빨강 둘이 연속으로 난 뒤 다섯 번 연속 초록이라
재현 경로를 못 잡았고, **못 잡은 것을 잡았다고 안 적는다.** 진행자에게 넘긴다
(8절 ⑤).

### 3.5 게이트 7 — 장식 문자

```text
   이 유닛이 새로 넣은 것   **0**  (새 파일 셋 · 고친 파일 셋의 + 줄 전부)
   저장소의 기존 위반       **15 · 파일 여덟.  그대로 15 다** — 안 건드렸다
                          여덟이 전부 이 유닛의 파일 행렬 밖이고, 고치면
                          게이트 2 의 diff 0 이 깨진다 (계획 6.3 · 9.4)
```

---

## 4. 수명 끊는 자리 셋 — 전부 섰다

```text
   ①  봉인이 지운다        record.go 의 Seal 첫 줄.  Sealed() 조기 반환보다 앞이다
                          시험 TestSealDropsTheProgressTreeEvenWhenAlreadySealed
   ②  종료 상태가 지운다    store/seal.go 의 sealRecord.  GetRun 과 StepFiles 보다 앞
                          시험 TestSealRecordDropsTheTreeEvenWhenStepFilesFails
   ③  회수기가 쓴다        store/reap.go 의 Reap.  조건 없이 매 주기
                          시험 TestTheSweepRunsInACycleThatReclaimsNothing
```

**셋이 서로를 안 믿는다.** ① 이 안 걸린 Run 을 ② 가 줍고, ② 도 못 간 자리를
③ 이 줍는다. 변이 ⑤ 와 ⑦ 이 그 배치를 재고 각각 빨개졌다 (5절).

### 4.1 개행 보장(R43)과 틈 넷 — 돈다

```text
   TestTheMarkIsPrecededByANewline          초록.  변이 ① 에 빨개진다
   TestTheFourGapsConverge  (부분 시험 넷)    **넷 다 초록**
     before the cut write                   ③ 이 실제로 자른다
     between the cut write and the guard    예산 0 이라 ③ 이 0 바이트
     inside the mark write                  ④ 가 반쪽 조각을 개행으로 닫는다
     after the mark write                   R15 로 더 안 쓴다
```

네 자리 모두 **기대 바이트를 통째로 대조**한다.

---

## 5. 변이 여덟 — **여덟 다 빨개졌다**

넣고 빨개지는지 봤다. 안 빨개지면 그 규칙을 재는 시험이 없는 것이다.

| | 변이 | 빨개진 시험 |
|---|---|---|
| ① | R43 의 개행 보장을 걷는다 | `TestTheMarkIsPrecededByANewline` · `TheFourGapsConverge/inside_the_mark_write` |
| ② | R14 를 크기 비교로 바꾼다 | `TestCappedIsJudgedByTheMarkNotTheSize` · `TestTheMarkIsPrecededByANewline` |
| ③ | R6 의 오른쪽 파싱을 왼쪽으로 | `TestProgressNameIsReadFromTheRight` |
| ④ | R9 의 앞 시도 걷기를 걷는다 | `TestANewAttemptSweepsTheOlderFile` |
| ⑤ | R22 의 `DropProgress` 를 조기 반환 뒤로 | `TestSealDropsTheProgressTreeEvenWhenAlreadySealed` |
| ⑥ | R27 의 잘림 판정을 총 길이로 | `TestAppendLogJudgesByWrittenAndReturnsTheTotal` |
| ⑦ | R37 의 순서를 뒤집는다 (나이 먼저) | 여섯 — `store` 다섯 · `record` 하나 |
| ⑧ | R1 의 자리를 기록 안으로 옮긴다 | 열아홉 — **14.1 과 14.2 가 둘 다 포함**된다 |

⑧ 이 계획 17.8 의 요구(`TestTarHasNoProgressEntries` 와
`TestTheProgressTreeStaysWritableAfterSealing` 이 **둘 다** 빨개져야 한다)를
글자 그대로 만족했다.

**변이 뒤 트리를 되돌렸고 `git status --porcelain` 이 빈다.**

---

## 6. 이음매 — `enode.capped` · U1 이 읽을 값

**이 저장소에서 그 글자가 나는 자리는 하나다** —
`internal/record/progress.go` 의 `cappedMark` 구조체와 `cappedMarker` 함수.

```text
   찍는 바이트    {"type":"enode.capped","bytes":<상한>}  뒤에 개행 하나
   기본 상한 10 MiB
                {"type":"enode.capped","bytes":10485760}
                40 바이트 + 개행 1 = **41 바이트**
   필드 둘       type 은 문자열 · bytes 는 정수(int64).  그 밖의 필드가 0 개다
   bytes 의 뜻    **상한이다.**  잘린 자리가 아니다 — 멈춘 자리는 Total 이 말한다
```

맞대는 법 넷 중 이 유닛이 실제로 선 것과 못 선 것을 가른다.

```text
   ① 글자 그대로의 시험    **섰다.**  TestTheCappedMarkIsByteForByte 가 위 40 바이트를
                         한 글자도 안 다른지 재고 41 도 함께 잰다.  필드 이름 ·
                         순서 · 타입이 바뀌면 빨개진다
   ② 값으로 넘긴다        **섰다.**  이 절이 그 값이다.  7절이 진행자에게 넘긴다
   ③ 왕복                **못 섰다.**  찍는 쪽이 U3 이고 읽는 쪽이 U1 이라
                         **두 패키지가 한 바이너리에 처음 드는 것은 U4** 다.
                         **여기서 초록이라고 안 적는다**
   ④ 사람 눈              CB3.  U3 의 자리가 아니다
```

**이음매가 늦게 닫혀도 데이터가 안 죽는다.** U1 이 그 글자를 모르면 `raw`
사건 하나로 뜬다 — 버리지도 실행하지도 않는다. **표시의 질이 떨어지는 것이지
손실이 아니다.**

**U3 이 안 짓는 것** (R46) — `Kind` · 파싱 규칙 · 헤더 이름
(`X-Enode-Log-Capped`) · 화면 문장. U3 의 경계는 `Progress` 를 반환하는 순간
끝난다.

### 6.1 `AppendLog` 의 뜻이 바뀌어도 빨개진 시험이 0 이다

D4 가 반환값을 「이번 호출의 바이트」에서 「붙인 뒤의 총 길이」로 바꿨다.
**실측으로 그 값을 읽는 호출자가 0 이었다.**

```text
   제품 코드   internal/api/api.go:887 하나이고 `if _, err :=` 로 버린다
   시험        record_test.go 의 셋이 전부 `_` 다
   그래서      뜻을 바꿔도 빨개지는 시험이 0 이었다.  새로 지은
              TestAppendLogJudgesByWrittenAndReturnsTheTotal 이 처음으로 그 값을 읽는다
```

**그 사실 자체가 잔여다** — 아무도 안 읽는 값의 뜻을 바꾼 것이라 컴파일러도
시험도 안 막아 준다. 다음에 읽는 쪽은 U4 다.

---

## 7. 계획이 실측으로 틀려 고친 자리

### 7.1 시험 하나 — 틈 넷의 첫째 씨앗 (2절)

계획 13.6 이 「네 자리에서 멈춘 파일을 손으로 지어 놓고」로만 적었고 **씨앗이
상한에 닿아 있어야 한다는 것**을 안 적었다. 첫째 씨앗이 그 자리를 안 지났다.
고친 근거는 `nfr-design-patterns.md` 1.3 의 첫 행이다.

### 7.2 계약에 함수 하나가 늘었다 — `HasProgress()`

**계획 2절이 못 박은 서명 여섯은 한 글자도 안 바뀌었다** (실측 — `Progress` ·
`AppendProgress` · `ReadProgress` · `DropProgress` · `SweepProgress` ·
`ProgressMaxAge`). 그런데 **일곱째가 늘었다.**

```go
// internal/record
func (s *Store) HasProgress() bool
```

```text
   왜 늘었나   계획 5절 ④ 가 「쓸기는 디렉터리를 먼저 읽고 비어 있으면 DB 를
              아예 안 친다」를 요구했다.  그런데 **그 질의는 SweepProgress 를
              부르기 전에 일어난다** — 살아 있는 Run 목록이 SweepProgress 의
              인자이기 때문이다.  그래서 SweepProgress 안의 검사로는 그 요구를
              못 지킨다.  디렉터리 검사를 부르는 쪽이 먼저 할 수 있어야 하고,
              progressRoot 의 자리를 아는 것은 record 다

   계획의 어디가 틀렸나   Step 8.1 이 「SweepProgress 가 디렉터리를 먼저 읽는다」로
              적었다.  그 문장은 그대로 참이지만 (SweepProgress 도 os.ReadDir 로
              시작한다) **그것만으로는 질의를 못 막는다**.  가르는 함수가 하나 더
              필요하다는 것을 계획이 안 봤다

   재는 자리   TestTheSweepSkipsTheQueryWithoutATree 가 짝으로 잰다 —
              트리가 없으면 취소된 컨텍스트에서도 로그가 0 줄이고,
              트리가 있으면 같은 컨텍스트에서 질의가 돌아 로그가 난다
```

### 7.3 Step 15 의 나이 만들기 — `now` 인자가 아니라 `mtime`

계획 15.3 이 「`now` 인자를 앞으로 당겨 잰다」로 적었다. **`internal/store` 의
시험에서는 못 한다** — 거기서는 `Reap()` 을 통과해서 재고 `Reap` 이 안에서
`time.Now()` 를 부르기 때문이다. `os.Chtimes` 로 파일의 mtime 을 과거로 당겨
같은 것을 만들었다. **둘 다 슬립이 0 이다.** `now` 인자를 쓰는 길은
`internal/record` 쪽 시험이 그대로 쓴다.

---

## 8. 진행자에게 넘기는 것

계획 9절의 여섯이 **그대로 산다** (9.1 줄 번호 일곱 · 9.2 `unit-of-work.md`
U3 절 · 9.3 이음매 글자 · 9.4 게이트 7 의 도는 스크립트 0 · 9.5 쓸기의 질의
비용 · 9.6 앞 단계가 넘긴 것). 이 단계가 **새로 더하는 것은 셋**이다.

```text
   ①  계약에 HasProgress() 가 늘었다 (7.2)
      component-methods.md 2.1 과 계획 2절이 서명 여섯만 적는다.
      일곱째를 그 표에 넣을지는 진행자가 본다

   ②  이음매의 왕복이 U4 까지 안 닫힌다 (6절 ③)
      U3 은 못 잰다.  **초록이라고 안 적었다.**
      U1 에 줄 값은 6절의 41 바이트

   ③  AppendLog 의 뜻을 바꿨는데 막아 주는 것이 없었다 (6.1)
      읽는 호출자가 0 이라 뜻이 바뀌어도 아무것도 안 빨개진다.
      다음에 읽는 쪽이 U4 다 — 그쪽이 「총 길이」로 읽는지 확인해야 한다

   ④  계획 19.3 을 이 유닛이 안 했다 — 체크박스 하나가 빈 채로 남는다
      aidlc-docs/taeels/aidlc-state.md 의 U3 절과 audit.md 다.
      계획은 이 단계가 쓰는 것으로 적었는데 진행자가 회차 브랜치에서
      직접 쓰기로 정했다.  **소유자가 갈린 것이지 빠뜨린 것이 아니다**

   ⑤  게이트 4 에서 본 흔들림 둘 (3.4)
      internal/api 가 전체 스위트에서 두 번 연속 빨갰다가 그 뒤 다섯 번
      연속 초록이다.  **이 유닛의 것인지 못 갈랐다** — 그 패키지는 diff 0 이고
      DB 묶음만 여섯 번 돌리면 6/6 초록인데, 이 유닛이 scratch DB 를 여덟 개
      더 만드는 것도 사실이다.  줄이는 길이 있으면 그것이 값이다:
      여덟 시험이 판을 하나씩 파는 대신 하나를 나눠 쓰면 CREATE/DROP 이
      여덟에서 하나로 준다 (nodes 를 감추는 시험 하나만 따로 판을 판다)
```

---

## 9. 안 한 것 · 못 잰 것 — 이름으로 적는다

```text
   이음매의 왕복              **못 잰다.**  U4 의 자리다 (6절 ③)
   게이트 3 (경계 검사 여섯)    이 유닛이 안 건드린다.  U1 의 자리다
   게이트 4 · 5 (CB1 ~ CB6)   사람 눈이다.  U3 이 지는 조각이 0 이고 CB3 의 재료다
   게이트 7 의 기존 위반 열다섯   **안 고쳤다.**  여덟 파일이 전부 파일 행렬 밖이고
                            고치면 게이트 2 의 diff 0 이 깨진다.  진행자다 (9.4)
   AppendLog 의 n == limit 오판  안 고쳤다 (FD 6.3).  이 회차가 안 산 변경이고
                            record.go 에 주석 한 줄로 잔여임을 적었다
   쓸기의 질의 비용            안 막았다.  진행 트리가 있는 동안의 전수 스캔이
                            남는다 (계획 9.5).  인덱스는 schema.sql 이라 diff 0
   디스크 봉투 10 GiB          안 줄였다.  Q4 = A 가 그 대가를 샀다 (R40 · R41)
   GLOSSARY.md 의 CB · N1 · N2  **푼 말을 U3 이 모른다.  안 짐작했다** (CLAUDE.md)
                            새 축약어를 0 개 들여왔으므로 이 유닛이 적을 줄이 0 이다
```

---

## 10. 센 것

```text
   계획의 체크박스    **122 중 121**.  안 채운 하나가 19.3 이다 (8절 ④)
   제품 파일         넷 (새 파일 하나 · 고치는 파일 셋)
   시험 파일         둘 (둘 다 새 파일)
   새 시험 함수       record 30 · store 8  =  **38**
   변이             여덟.  **여덟 다 빨개졌다**
   수명 끊는 자리     셋.  **셋 다 섰다**
   라우트            17 -> **17**
   새 설정 키 · 새 타이머 · 새 고루틴 · 새 의존   **전부 0**
   커밋             넷  (bdb13a3 · 26bf7b4 · efc0042 · ebb7d7a)
```
