# U3 `progress-store` — 이 유닛이 드는 형식

**여기는 형식이다.** 흐름은 `business-logic-model.md`, 규칙은 `business-rules.md`
가 진다. 겉면(서명)은 `application-design.md` 3.2 와 `component-methods.md` 2.1
이 이미 닫았고 **이 문서는 그 겉면이 드는 값의 실제 표현**을 짓는다.

답은 `plans/progress-store-functional-design-plan.md` 의 물음 열하나다 —
**Q1 ~ Q9 = A · Q10 = B · Q11 = B**.

---

# 1. 진행 트리의 자리표

## 1.1 어디에 사나

```text
   <Root>/                                  Artifacts.Root.  config.go 의 artifacts.root
   ├── run-<safe(id)>/                      기록.  Store.dir() 이 짓는다 (record.go:32)
   │   ├── manifest.json
   │   ├── steps/NN-*.json
   │   ├── logs/NN-<safe(name)>.log         **선별본.  봉인된다**
   │   ├── blobs/NN.A-<safe(name)>
   │   └── verdict.json
   └── progress/                            **이 유닛이 만드는 것**
       └── run-<safe(id)>/
           └── NN-<safe(name)>.<attempt>.log   **원문.  안 봉인된다**
```

**형제다. 부모와 자식이 아니다.** D1(Q2 = A)이 고른 자리이고 그것이 이 유닛의
안전장치다.

## 1.2 왜 그 자리가 안전장치인가 — 실측으로 확인했다

```text
   seal(dir) 이 도는 곳   filepath.Walk(d) · d = s.dir(runID)     record.go:156 · :159
   Tar 가 도는 곳         filepath.Walk(d) · 같은 d               record.go:229
   진행 트리              <Root>/progress/...                     둘 다의 Walk 밖이다
```

**이름 충돌도 0 이다.** `dir()` 이 언제나 `"run-"` 을 앞에 붙이므로 어떤 `runID`
로도 디렉터리 이름이 `progress` 가 될 수 없다.

**그래서 지우기를 한 번 놓쳐도 원문이 0444 로 굳지 않는다.** `seal` 이 쓰기
비트를 내리는 것은 `<Root>/run-<id>/` 안뿐이다 (`record.go:174` 의 0444 ·
`:180` 의 0555). 진행 파일은 언제나 지울 수 있다 — **N2 가 성립할 수 있는 근거가
이 한 줄이다.**

## 1.3 봉인 묶음에 안 들어간다

`Tar` 의 출력에 `progress/` 가 0 개다. 이것이 질문 1 = B 가 산 값이다 —
봉인되는 것은 `logs/` 의 선별본이고 원문은 봉인 전에 사라진다.

---

# 2. 진행 파일

## 2.1 이름

```text
   NN-<safe(name)>.<attempt>.log

   NN         %02d.  단계 순번.  logs/ 와 같은 규칙이다
   safe(name) record.go:38 의 safe.  **다시 짓지 않는다** (CONVENTIONS 1.4)
   attempt    %d.  시도 회차.  Q1 = A 가 곁파일 대신 이름에 넣었다
   .log       꼬리
```

**보기** — `01-build.0.log` · `03-agent-review.2.log`

## 2.2 왜 이름에 넣나 (Q1 = A)

```text
   곁파일이 0 이다      파일 둘의 원자성을 지킬 자리가 안 생긴다
   비우기가 공짜다      시도가 바뀌면 새 이름으로 연다.  자르기가 없다
   총 길이가 저절로 0   새 파일이라 0 부터 다시 센다
   선례가 같은 파일에    blobPath 가 %02d.%d-%s 로 이미 시도를 이름에 넣는다
                       (record.go:270).  그 주석이 이유까지 적어 뒀다
```

## 2.3 이름을 **오른쪽에서** 읽는다

`safe()` 는 점을 안 바꾼다 (2.6 의 실측표). 그래서 `safe(name)` 안에 점이 들어갈
수 있다.

```text
   "01-build.step.2.log"      ->  단계 이름 "build.step" · 시도 2
   왼쪽에서 읽으면            ->  시도가 "step" 이 되어 깨진다
```

**규칙은 `business-rules.md` R6 이다.** 여기서는 그 결과 하나를 적는다 —
**`parseBlobName`(`record.go:263`)을 재사용할 수 없다.** 그쪽은 `%02d.%d-이름`
이라 왼쪽에서 읽는다. **새 헬퍼가 하나 는다.**

## 2.4 한 줄의 모양

```text
   보통의 줄   하네스가 낸 stream-json 한 줄.  **원문 그대로다.**  이 유닛은
              내용을 안 본다 — 바이트를 다루지 사건을 안 다룬다
   표시 줄     상한에 닿았을 때 **마지막에 한 줄** (Q10 = B).  아래 2.5
   줄 경계     청크 경계는 줄 경계가 아니다.  도는 동안 파일의 꼬리가 반쪽 줄인
              것이 정상이다 — 링도 같다.  읽는 쪽(U1 의 파서)이 그것을 다룬다
```

**끝이 개행이라는 보장이 없다.** 상한에 닿은 뒤에만 보장된다 (R12).

## 2.5 표시 줄 — `enode.capped`

Q10 = B 가 고른 것이고 **모양을 `internal/enode/runner.go:415` 의 `elidedMark`
에 맞춘다.** 두 벌로 짓지 않는다.

```text
   찍는 것    {"type":"enode.capped","bytes":<상한>}  뒤에 개행 하나
   선례       elidedMarker 가 {"type":"enode.elided","events":N,"bytes":B} 에
              개행 하나를 붙여 쓴다 (runner.go:361-362 · :425)
```

**`type` 에 점을 넣는 근거가 그대로 선다.** `elidedMark` 의 주석이 적어 뒀다 —
실측한 하네스의 `type` 이 전부 홑단어라(`system` · `assistant` · `user` ·
`result` · `rate_limit_event`) 부딪칠 수 없다.

**`bytes` 가 무엇인가 — 상한이다.** 잘린 자리가 아니다.

```text
   근거    같은 저장소의 기존 표시가 상한을 적는다 —
           AppendLog 의 "... log truncated at %d bytes" 의 %d 가 limit 이다
           (record.go:72).  읽는 사람이 알고 싶은 것은 「어디서 멈췄나」가 아니라
           「무엇에 걸렸나」다.  멈춘 자리는 Total 이 이미 말한다
```

**U3 이 지는 것은 찍는 쪽뿐이다.** 이 줄을 `Kind` 로 받아 그리는 규칙은 **U1 이
진다** — `internal/transcript` 가 사건의 모양을 지는 하나다
(`application-design.md` 2절). U3 은 파싱 규칙을 안 짓는다.

## 2.6 이름이 막는 것과 안 막는 것 — 실측 (2026-09-15)

`record.go:38` 의 `safe` 를 떼어 돌린 값이다.

| 넣은 것 | 나온 것 | 판정 |
|---|---|---|
| `../../etc/passwd` | `____etc_passwd` | 탈출이 **막힌다** |
| `a/../b` | `a___b` | **막힌다** |
| `...` | `_.` | 남는다. 탈출은 아니다 |
| (빈 문자열) | (빈 문자열) | `01-.0.log` 가 된다 |
| `step\nname` | `step\nname` | **개행이 그대로 간다** |
| `step\x00name` | `step\x00name` | **NUL 이 그대로.** `os` 가 뒤에서 거절한다 |
| 300자 | 300자 | **길이 상한이 0.** `ENAMETOOLONG` |

**경로 탈출만 막는다.** Q9 = A 가 나머지를 U4 에 맡겼으므로 그 전제를
`business-rules.md` 8절이 규칙으로 적는다 — **전제를 안 적으면 U4 가 그것을
모른다.**

## 2.7 권한 (Q8 = A)

| 자리 | 값 | 근거 |
|---|---|---|
| `<Root>/progress/` | `0700` | 디렉터리는 진행 파일보다 열려 있을 이유가 없다 |
| `<Root>/progress/run-<id>/` | `0700` | 같다 |
| `NN-<name>.<attempt>.log` | `0600` | `requirements.md` 5.4 가 값으로 닫았다 — 「진행 파일의 권한은 0600. Mediator 프로세스만」 |

**`logs/` 와 다르다.** `logs/` 는 파일 0644 · 디렉터리 0755 다 (`record.go:59` ·
`:47`). **더 닫는 쪽으로 다르고 이유가 내용에 있다** — `logs/` 는 선별본이고
진행 파일은 원문이다 (`requirements.md` 5.4 의 잔여 ①).

---

# 3. `Progress` — 세 필드가 무엇을 담나

겉면은 `component-methods.md` 2.1 이 닫았다.

```go
type Progress struct {
	Total   int64 // 파일의 총 길이
	Attempt int   // 지금 시도
	Capped  bool  // 상한에 닿았다
}
```

## 3.1 `Total` 의 정의는 하나다 — **파일의 크기**

**예외가 0 이다. 표시 줄도 든다.**

```text
   그래서 참인 것   from 은 언제나 이 파일의 진짜 바이트 오프셋이다.
                  Total 과 파일 크기가 갈리는 순간이 없다
   그래서 거짓인 것  Total <= 상한 이 아니다.  닿은 뒤에는 표시 줄만큼 넘는다
```

**넘겨 적는 것이 이 집의 결이다.** 같은 파일의 `WriteBlob` 이 `limit+1` 을 읽어
초과를 가른다 (`record.go:287`) — **넘었음을 알려면 넘겨야 한다.** 표시 줄도
같다. 근거와 규칙은 `business-rules.md` R13 · R14 다.

## 3.2 상태마다의 값

| 상태 | `Total` | `Attempt` | `Capped` |
|---|---|---|---|
| 그 단계의 파일이 하나도 없다 | `0` | `0` | `false` |
| 도는 중 | 파일 크기 | 이름에서 읽은 최대 시도 | `false` |
| 시도가 바뀐 직후 | `0` (새 파일) | 새 시도 | `false` |
| 상한에 닿았다 | 파일 크기 (**표시 줄 포함**) | 그 시도 | `true` |
| 닿은 뒤 또 PUT 이 왔다 | 안 변한다 | 안 변한다 | `true` |
| 뒤늦은 시도의 청크가 왔다 | **지금 시도의** 크기 | **지금** 시도 | 지금 시도의 값 |
| 봉인·쓸기로 트리가 사라졌다 | `0` | `0` | `false` |

**마지막 두 줄이 읽는 쪽을 헷갈리게 하는 자리다.** 총 길이가 뒤로 간다.
`business-rules.md` 4절이 그 규칙을 진다.

## 3.3 `Attempt` 가 0 일 수 있다

**「시도 0」과 「파일이 없다」가 같은 값이다.** 갈라 쓰지 않는다 — 시도 0 은 실제
값이기도 하다 (`claim.go:73` 의 `Attempt int` 는 0 부터다. `expands` 로 붙은
단계가 시도 0 이라는 것을 `claim.go:712-713` 의 주석이 적는다).

**갈리는 자리는 `Total` 이다.** 파일이 없으면 `Total` 이 0 이고 본문이 비었다.

---

# 4. 뮤텍스 — Q7 = A 의 자료구조

## 4.1 오늘은 락이 0 이다

`internal/record` 는 `record.go` 하나이고 `sync` 를 임포트하지 않는다.
`AppendLog` 은 `O_APPEND` 하나로 버틴다 — **단계마다 한 번이고 단계마다 파일이
달라서** 오늘은 부딪치지 않는다.

**청크가 오면 부딪친다.** 같은 단계에 여러 번 쓰고, 시도가 바뀌는 순간 지우기와
쓰기가 겹친다.

## 4.2 어디 사는가 · 키가 무엇인가 · 언제 나고 죽는가

```text
   어디     Store 의 필드다.  Store 가 오늘 struct{ Root string } 이므로
            필드 둘이 는다 — 맵 하나와 그 맵을 지키는 뮤텍스 하나

   키       (runID, seq, name) 셋이다.  **attempt 를 안 넣는다**
            넣으면 「지금 시도 고르기」와 「쓰기」가 다른 임계 구역으로 갈려
            시도가 바뀌는 순간이 안 막힌다 (R28 의 근거)

   난다     그 키로 처음 AppendProgress 나 ReadProgress 가 올 때

   죽는다   DropProgress 가 그 Run 의 키를 전부 걷을 때.  **트리와 함께 죽는다**
```

## 4.3 걷는 순간의 함정을 이름으로 적는다

**뮤텍스를 맵에서 지우는 순간 그것을 쥔 쓰기가 있으면**, 다음 호출이 새 뮤텍스를
만들어 둘이 같은 파일에서 동시에 돈다. 맵을 쓰는 코드의 고전적인 함정이다.

**불변식으로 막는다 (R31)** — 드롭과 쓰기는 안 겹친다. 어떤 자료구조로 그것을
세울지는 Code Generation 이 고른다. **이 문서가 지는 것은 「겹치면 안 된다」가
규칙이라는 사실이다.**

## 4.4 무엇이 직렬이고 무엇이 아닌가

```text
   직렬이다      같은 (run, seq, name) 의 쓰기 · 읽기 · 시도 바뀌기
   직렬이 아니다  다른 단계.  다른 Run.  **키가 다르면 안 기다린다**
   선례          Ring 이 같은 자리를 sync.Mutex 하나로 풀었다 (transcript.go:40)
```

---

# 5. 이 문서가 안 짓는 것

```text
   Kind 와 파싱 규칙       U1.  internal/transcript 가 사건의 모양을 지는 하나다.
                          U3 은 enode.capped 를 **찍는 쪽**만 진다
   라우트 · 쿼리 · 헤더    U4.  X-Enode-Log-* 넷은 이 유닛의 것이 아니다
   업로더의 주기와 버퍼     U7.  decisions.md 2절이 값으로 닫았다
   N2 의 숫자             NFR Requirements.  business-rules.md 5절이 규칙의
                          모양까지만 짓고 값의 자리를 비워 둔다
   화면의 문장            U5 · U8
```
