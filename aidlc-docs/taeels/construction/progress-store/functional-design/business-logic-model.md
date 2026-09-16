# U3 `progress-store` — 순서와 흐름

**여기는 순서다.** 형식은 `domain-entities.md`, 규칙은 `business-rules.md` 가
진다. 이 문서의 모든 순서는 **규칙 번호(R)** 로 근거를 가리킨다.

---

# 1. `AppendProgress` — 청크 하나가 들어오는 길

겉면은 이렇다 (`component-methods.md` 2.1).

```go
func (s *Store) AppendProgress(runID string, seq int, name string,
	attempt int, r io.Reader, limit int64) (Progress, error)
```

## 1.1 순서 일곱

```text
   ①  키를 잠근다              (runID, seq, name).  attempt 를 안 넣는다   R28
       ② ~ ⑦ 이 전부 이 안이다.  「지금 시도 고르기」와 「쓰기」가
       갈리면 시도가 바뀌는 순간이 안 막힌다

   ②  디렉터리를 만든다        MkdirAll(<Root>/progress/run-<id>, 0700)    R3
       게으르다 — 쓸 때만 만든다.  Open 을 안 건드린다

   ③  지금 시도를 읽는다        디렉터리를 훑어 (seq, name) 의 최대 시도    R5 · R6
       곁파일이 없으므로 이름이 진실이다.  오른쪽에서 읽는다

   ④  시도를 가른다            아래 1.2 의 세 갈래                        R7 · R8 · R9

   ⑤  상한을 본다              이미 닿았으면 0 바이트를 쓰고 ⑦ 로         R15
       판정의 근거는 **표시 줄의 존재**다.  크기가 아니다                 R14

   ⑥  쓴다                     아래 1.3 의 두 갈래                        R10 ~ R13

   ⑦  Progress 를 짓는다       Total = 파일 크기 · Attempt = 지금 시도 ·  R13
       Capped = 표시 줄이 있다
```

## 1.2 ④ 의 세 갈래 — 시도를 가른다

```text
   attempt <  지금    **버린다.**  0 바이트를 쓰고 지금의 Progress 를 돌려준다   R8
                     노드는 돌아온 Total 과 Attempt 로 자기가 뒤처진 것을 안다

   attempt == 지금    이어 붙인다.  ⑤ 로 간다                                  R7

   attempt >  지금    **새 시도다.**  앞 시도의 파일을 걷고 새 이름으로 연다     R9 · R16
                     걷기가 실패해도 진행은 안 막는다 — 보조다                  R25
```

**「버린다」가 안전한 이유** — Q1 = A 가 시도마다 파일을 가르므로, 뒤늦은 청크를
받아도 **다른 파일**에 붙는다. 그래도 버리는 것은 그 파일이 곧 걷힐 것이고
(R9) 노드에게 뒤처짐을 알려야 하기 때문이다.

## 1.3 ⑥ 의 두 갈래 — 쓴다

```text
   상한 안이다        청크를 **통째로** 쓴다.  개행에 안 맞춘다              R10
                     맞추면 Total 이 노드가 보낸 것보다 뒤처지고
                     노드가 그 차이를 다시 보내 루프가 돈다

   상한을 넘긴다      상한 앞 **마지막 개행까지만** 쓰고 나머지를 버린다      R11
                     그 뒤 표시 줄 한 줄을 박는다                          R12
                     예산 안에 개행이 없으면 0 바이트를 쓰고 표시 줄만       R11
```

**표시 줄을 박는 것이 이 단계의 마지막 쓰기다.** 그 뒤로 이 파일은 안 자란다
(R15).

## 1.4 왜 ① 이 전부를 감싸나

③ 과 ⑥ 사이에 다른 호출이 끼어들면 세 가지가 깨진다.

```text
   시도 둘이 같은 「지금」을 읽는다     둘 다 새 파일을 만들려 한다
   걷기와 쓰기가 겹친다               방금 만든 새 시도의 파일을 옆 호출이 걷는다
   상한 판정과 쓰기가 갈린다           둘 다 「아직 안 닿았다」를 보고 둘 다 쓴다
```

---

# 2. `ReadProgress` — 읽는 길

```go
func (s *Store) ReadProgress(runID string, seq int, name string,
	from int64) (Progress, io.ReadCloser, error)
```

**`attempt` 인자가 없다.** 그래서 읽는 쪽은 언제나 **최대 시도**를 연다 (R5).
이것이 「앞 시도를 아무도 안 읽는다」를 이 유닛 안에서 참으로 만드는 장치다
(R17).

## 2.1 순서 다섯

```text
   ①  키를 잠근다                    쓰기와 같은 키                       R28
   ②  지금 시도를 고른다              최대 시도.  없으면 ⑤ 의 빈손          R5
   ③  Capped 를 판정한다              **마지막 줄이 표시 줄인가**           R14
   ④  from 부터 연다                  아래 2.2 의 세 갈래
   ⑤  Progress 와 본문을 돌려준다
```

## 2.2 ④ 의 세 갈래

```text
   그 단계의 파일이 없다     Total 0 · Attempt 0 · Capped false · 빈 본문     R18
                          **404 가 아니다.**  「아직」과 「없다」를 폴링하는
                          쪽이 가르기 쉽게 한다 (decisions.md 의 GET 라우트 행)

   from >= Total           빈 본문.  Total 은 그대로.  정상이다 — 폴링이
                          따라잡은 상태다

   from <  Total           from 부터 끝까지
```

**`from > Total` 도 오류가 아니다.** 총 길이가 뒤로 간 뒤의 폴링이 그 모양이다
(3절 · R19).

---

# 3. 총 길이가 뒤로 가는 두 경로

**읽는 쪽이 이것을 「응답이 깨졌다」로 읽으면 안 된다.** 경로가 둘이고 **갈리는
표시가 다르다.**

```text
   경로 1 — 시도가 바뀌었다
     일어나는 것   AppendProgress ④ 의 셋째 갈래.  새 파일이라 Total 이 0 부터
     읽는 쪽이 아는 법   **Attempt 가 바뀐다.**  화면이 그 값으로 카드를 비운다
                        (application-design.md 1.1 이 링의 gen 과 같은 답을 썼다)

   경로 2 — 트리가 걷혔다 (봉인 · 종료 · 쓸기)
     일어나는 것   DropProgress.  파일이 통째로 없다
     읽는 쪽이 아는 법   **Attempt 가 안 바뀐다.**  0 으로 간다.
                        Total 도 0 이다 — 둘이 함께 0 인 것이 이 경로의 표시다
```

**규칙은 R19 · R20 이다.** 경로 2 를 안 적으면 화면이 카드를 안 비우고 오래된
본문을 계속 들고 있는다.

---

# 4. `DropProgress` — 걷는 길

```go
func (s *Store) DropProgress(runID string) error
```

```text
   ①  그 Run 의 키들을 잠근다        드롭과 쓰기가 안 겹친다              R31
   ②  <Root>/progress/run-<safe(id)> 를 통째로 지운다
   ③  뮤텍스 맵에서 그 Run 의 키를 걷는다
   ④  없으면 아무것도 안 한다 — **멱등이다**                             R21
```

**Run 단위다. 단계 단위가 아니다.** 봉인도 종료도 쓸기도 Run 단위로 일어난다.

---

# 5. `Seal` 안에서 `DropProgress` 가 서는 자리

## 5.1 오늘의 `Seal`

```go
// record.go:92
func (s *Store) Seal(runID string, ...) error {
	if s.Sealed(runID) {
		return nil          // 봉인은 멱등이다
	}
	if err := s.Open(runID); err != nil { ... }
	... manifest · steps · verdict 쓰기 ...
	return seal(d)          // 0444 · 0555
}
```

## 5.2 어디에 넣나 — **조기 반환보다 앞이다**

```text
   func Seal(...) {
       DropProgress(runID)     <- **여기다.**  실패해도 안 멈춘다 (R25)
       if Sealed(runID) { return nil }
       Open(runID)
       ...
       return seal(d)
   }
```

**Q8 = A 가 이 자리를 가능하게 했다.** `Open` 이 진행 트리를 안 만들므로
(`AppendProgress` 가 게으르게 만든다) **`Open` 뒤에 둘 이유가 사라졌다.**
Q8 = B 였으면 `Open` 이 트리를 다시 만들어 드롭을 `Open` 뒤로 밀어야 했다.

## 5.3 왜 조기 반환보다 **앞**인가 — 늦은 청크의 고아 트리

`putLog` 는 **DB 의 `run.State`** 로 410 을 낸다 (`api.go:879`). `Sealed()` 를
안 본다. 그래서 창이 하나 열린다.

```text
   t0   PUT 이 들어와 run.State 검사를 지난다        아직 RUNNING 이다
   t1   Run 이 종료 상태가 되고 sealRecord 가 돈다
   t2   Seal 이 DropProgress 로 트리를 지운다
   t3   t0 의 PUT 이 AppendProgress 에 닿는다        **트리가 다시 생긴다**
   t4   아무도 다시 Seal 을 안 부른다                 고아 트리가 남는다
```

**조기 반환보다 앞에 두면 t4 가 막힌다.** 이미 봉인된 Run 에 `Seal` 이 다시
불려도(`sealExpired` 가 주기마다 훑는다 — `reap.go:148` 이 `Sealed` 면 건너뛰지만
다른 경로가 부를 수 있다) 드롭이 돈다.

**t4 를 완전히 막는 것은 6절의 쓸기다.** 5.2 의 자리는 값싼 한 겹이고 쓸기가
바닥이다. **둘을 함께 둔다** — `application-design.md` 4절이 「순서는 값이고
자리는 안전장치다」로 적은 것과 같은 결이다.

---

# 6. 진행 파일의 수명 — 태어나는 자리 하나 · 끊는 자리 셋 (Q5 = A)

## 6.1 그림

```text
   태어난다
     AppendProgress 의 ②          게으른 MkdirAll.  쓸 때만            R3

   끊는다 — 셋이다
     ① record.Seal 의 첫 줄        봉인이 실제로 돌 때                  R22
     ② store.sealRecord            **Seal 의 성공과 무관하게** 부른다.
                                   종료 상태에 이르면 끊는다             R23
     ③ 고아 쓸기                    살아 있는 Run 이 없는 진행 트리를
                                   기동과 주기마다 걷는다                R24
```

## 6.2 왜 셋인가 — 구멍이 **둘**이다

`Seal` 하나에 매달면 다음 둘에서 트리가 영원히 남는다. **코드로 확인했다.**

| | 구멍 | 어디 | 어느 끊는 자리가 막나 |
|---|---|---|---|
| ① | `sealExpired` 의 창이 한 시간이다 — `ended_at > now() - interval '1 hour'` | `reap.go:126` | **③ 쓸기** |
| ② | `sealExpired` 는 회수된 임대가 있을 때만 돈다 — `if n > 0` | `reap.go:104` | **③ 쓸기** |

**Part 1 이 적은 셋째는 구멍이 아니었다.** 반증을 함께 적는다 (R26).

## 6.3 ② 와 ③ 이 `internal/store` 에 사는 이유

```text
   ② 도 ③ 도 Run 의 상태를 알아야 한다
   internal/record 는 internal/store 를 임포트하지 않는다 — 오늘의 방향이다
   그래서 record 는 DropProgress 만 내주고 부르는 쪽이 store 에 앉는다
```

**그 대가가 파일 행렬과 품질 게이트를 깬다.** `business-rules.md` 10절이 그
문장을 확정으로 적는다.

---

# 7. 재시도의 순서 — 다섯이 맞물린다

`application-design.md` 4절의 것을 U3 의 자리까지 조였다.

```text
   ①  attempt 가 오른다                        노드.  claim.go 의 step.Attempt
   ②  링이 비워진다                            U2.  Ring.Reset 이 gen 을 올린다
   ③  첫 청크가 새 attempt 를 싣고 온다         U7 -> U4 -> U3
   ④  **AppendProgress 가 앞 시도를 걷고        U3.  ④ 의 셋째 갈래.  R9 · R16
       새 이름으로 연다**
   ⑤  응답의 Attempt 가 바뀌어 화면이 비운다     U4 의 헤더 · U5 · U8
```

**④ 가 이 유닛의 몫이다.** ①②③⑤ 는 다른 유닛이다.

---

# 8. `AppendLog` 과 `AppendProgress` 가 갈리는 자리 (Q2 = A)

**같은 파일의 두 함수가 상한 규칙이 다르다.** 이유를 안 적으면 다음 사람이
한쪽을 버그로 읽는다.

| | `AppendLog` (`logs/`) | `AppendProgress` (`progress/`) |
|---|---|---|
| 부르는 횟수 | **단계마다 한 번** (`claim.go:647` · `:796`) | **청크마다** |
| 상한의 뜻 | 이번 호출의 바이트 | **총 길이** |
| 자르는 자리 | 정확히 상한 | **상한 앞 마지막 개행** |
| 표시 | `... log truncated at N bytes` (텍스트) | `{"type":"enode.capped",...}` (NDJSON) |
| 시도 | **이름에 없다.** 재시도가 이어 붙는다 | **이름에 있다.** 시도마다 파일 |
| 봉인 | 된다 | **안 된다** |

## 8.1 U3 이 `AppendLog` 에 하는 것은 **하나뿐이다**

```text
   한다      반환값의 뜻을 「이번 호출의 바이트」에서 「붙인 뒤의 총 길이」로 (D4)
   안 한다   상한을 총 길이로 바꾸는 것.  자르는 자리.  표시.  이름에 시도
```

**그래서 봉인되는 기록의 거동이 안 바뀐다.** Q2 = A 가 산 것이 이것이다.

## 8.2 그 하나가 낳는 함정 하나

반환값이 총 길이가 되면 **잘림 판정을 그 값으로 하면 안 된다.**

```text
   오늘      n, _ := io.Copy(f, io.LimitReader(r, limit))
             if n == limit { 표시 }
             return n                     <- n 이 둘 다를 겸했다

   D4 뒤     written := io.Copy(...)      이번 호출의 바이트
             total   := 파일 크기          돌려줄 값
             if <잘림 판정> { 표시 }       **written 으로 한다.  total 이 아니다**
             return total
```

`total` 로 적으면 재시도가 깨진다 — `logs/NN-*.log` 는 시도가 이름에 없어
이어 붙으므로 (`domain-entities.md` 2.2 의 비대칭), 시도 2 에서 `total` 이 상한을
넘어 **안 잘렸는데 잘렸다고 표시**된다. 규칙은 R27 이다.
