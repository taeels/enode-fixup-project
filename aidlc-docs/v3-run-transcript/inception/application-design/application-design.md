# 응용 설계 — 트랜스크립트 (v3-run-transcript)

```text
   입력    requirements.md (Comprehensive · FR-1 ~ FR-8)
           user-stories.md (스토리 열하나 · NC-1 ~ NC-6)
           plans/execution-plan.md (미결 D1 ~ D7)
           plans/application-design-plan.md (답 넷 · 전부 A)
   정본    enode-design/.  어긋나면 protocol/INVARIANTS.md 가 이긴다
   확장    security-baseline 켬
   나눠 적은 곳   components.md · component-methods.md · services.md ·
                 component-dependency.md.  여기는 그 넷의 통합본이다
```

---

# 1. 미결 일곱이 닫혔다 — D1 ~ D7

**이 단계가 지는 표다.** 하나라도 빈 채로 넘기면 유닛이 그 자리에서 지어낸다.

| | 미결 | 답 | 근거 |
|---|---|---|---|
| D1 | 진행 파일의 이름과 자리 | `<Root>/progress/run-<safe(id)>/NN-<safe(name)>.log`. **기록 디렉터리 밖이다** | Q2 = A. `seal(dir)` 이 `Walk` 로 0444 · 0555 로 굳히고 보관 정책이 없다 (`record.go:156` 의 `seal` · 0444 는 `:174` · 0555 는 `:180` · 보관 정책이 없다는 주석은 `:189-191`. **앞 판이 `:196` 으로 적었는데 그것은 `unseal` 안이다** — U3 의 Functional Design 이 고쳤다). 안에 두면 지우기를 한 번 놓쳤을 때 원문이 영구가 된다 |
| D2 | PUT 을 어떻게 가르나 | 기존 라우트에 쿼리 둘 — `?progress=1&attempt=<n>`. 새 라우트 0 | CB0 의 셈이 17 -> 18 이고 는 것은 GET 하나뿐이다 |
| D3 | 출처를 말하는 헤더 | `X-Enode-Log-Source: progress \| sealed` | `Sealed(runID)` 한 줄이 갈린다 (`record.go:76`). NC-5 가 이 헤더를 그린다 |
| D4 | `AppendLog` 의 시그니처 | 그대로 둔다. **반환값의 뜻만 바뀐다** — 이번 호출 바이트에서 총 길이로. 상한도 총 길이로 건다 | `api.go:887` 이 값을 버린다 (`if _, err :=`). 시그니처가 같아 컴파일도 안 깨진다. **근거 문장 하나가 낡았다** — `requirements.md` 2.4 의 「청크마다 10 MiB」는 청크가 `logs/` 로 간다는 전제인데 질문 1 = B 가 파일을 갈랐다. U3 의 Functional Design 이 상한을 `AppendProgress` 에만 걸었고 (물음 2 = A) `AppendLog` 의 `n == limit` 오판은 **코드의 잔여로 남긴다** |
| D5 | `selectLogs` 가 파서를 쓰나 | **쓴다.** 껍데기를 짓는 셋을 `internal/transcript` 로 옮기고 `selectLogs` 가 임포트한다 | Q4 = A. `enode -> transcript` 는 허용이고 반대가 금지다. 옮기는 방향이 유일하게 금지를 안 건드린다 |
| D6 | 유닛 분해와 파일 행렬 | **여기서 안 가른다.** Units Generation 의 몫이다 | `constraints.md` 의 구조 불변식. 팩이 유닛을 안 준다 |
| D7 | 재시도가 어떻게 갈리나 | **시도가 바뀌면 진행 파일을 비운다.** 링과 같은 답이다 | Q1 = A. B · C 는 링이 비워진 뒤에도 진행 파일에 시도 1 이 남아 CB4 의 「같은 문장」이 깨진다 |

## 1.1 D7 이 낳은 파생 결정 하나

**비우면 총 길이가 뒤로 간다.** 읽는 쪽이 그것을 「응답이 깨졌다」로 읽으면 안 된다.

링이 같은 자리를 `gen` 으로 이미 풀었다 (`transcript.go:148`). 같은 답을 쓴다 —
**응답이 `X-Enode-Log-Attempt` 를 싣고 그 값이 바뀌면 화면이 카드를 비운다.**
설계가 답한 것이고 새로 묻지 않았다.

## 1.2 이 단계가 찾은 일곱째 경로

`requirements.md` 7.3 이 새 코드가 사는 자리를 여섯으로 적었다. **일곱이다.**

`internal/panel` 은 Mediator 를 `runctl.Client` 로만 부른다 (`panel.go` 의
패키지 주석). FR-4 가 지난 것의 출처를 GET 라우트로 바꾸라고 했으므로
**`internal/runctl` 에 클라이언트 메서드 하나가 는다** — `StepLog`.

```text
   안 적으면 무엇이 되나   유닛이 그 자리에서 net/http 를 제어판에 직접 들여온다.
                        패키지 주석이 세운 「조회는 runctl.Client 로」가 그 순간 깨진다
   게이트에 걸리나        안 걸린다.  cmd/enodectl 이 internal/runctl 을
                        임포트하지 않는다 (실측).  enodectl.exe 의 심볼 상한과 무관하다
```

---

# 2. 컴포넌트 일곱

전문은 `components.md`.

```text
   internal/transcript   신규.  사건의 모양을 지는 하나.  **짓는 쪽과 읽는 쪽이 여기 산다**
                         표준 라이브러리만.  시계를 모른다 (Q3 = A)
   internal/enode        Decode 배출 · 링 tee 되살리기 · 업로더.  selectLogs 가 파서를 쓴다
   internal/record       진행 파일의 주인.  쓰기 · 시도 비우기 · 총 길이 · 봉인 때 지우기
   internal/api          GET log 신규 하나 · PUT 의 쿼리 갈림.  api.go 는 등록 줄 하나
   internal/panel        카드가 파서를 쓴다 · 지난 것의 출처 전환 · 보안 헤더 다섯
   internal/api/ui       Run 상세 단계 카드.  Go 변경 0 — 정적 파일만 는다
   internal/runctl       클라이언트 메서드 하나.  1.2 가 찾은 것
```

## 2.1 실패 등급 — 무엇이 실행을 죽이나

```text
   치명    Decode 의 실패.  오늘과 같다 — HarnessResult 가 크래시로 나온다
   보조    링 쓰기 · 청크 PUT · 업로더 Close · 진행 트리 지우기
```

**관측 표면이 실행을 막으면 그것은 관측이 아니라 의존이다.** 청크 PUT 이
실패해도 하네스의 stdout 이 안 막히고 링도 안 막힌다 (FR-5).

---

# 3. 겉면

전문은 `component-methods.md`. **여기는 겉면이고 필드의 값은 Functional Design 이
닫는다.**

## 3.1 파서

```go
func Parse(b []byte, truncated bool) Result   // 읽는 쪽. 시계를 안 받는다

func ParseLine(line []byte) (map[string]any, string, bool)  // 옛 parseEventLine
func Shell(obj map[string]any, typ string) []byte           // 옛 eventShell
func ElidedMarker(events, bytes int) []byte                 // 옛 elidedMarker
```

## 3.2 진행 파일

```go
type Progress struct{ Total int64; Attempt int; Capped bool }

func (s *Store) AppendProgress(runID string, seq int, name string,
	attempt int, r io.Reader, limit int64) (Progress, error)
func (s *Store) ReadProgress(runID string, seq int, name string,
	from int64) (Progress, io.ReadCloser, error)
func (s *Store) DropProgress(runID string) error
```

## 3.3 라우트

```text
   GET /v1/runs/{run}/steps/{seq}/log?from=&as=&name=       신규.  17 -> 18
   PUT /v1/runs/{run}/steps/{seq}/log?progress=1&attempt=   기존 라우트의 갈림

   응답 헤더 넷   X-Enode-Log-Bytes     총 길이
                 X-Enode-Log-Source    progress | sealed   D3 · NC-5
                 X-Enode-Log-Attempt   지금 시도           D7 의 대가를 닫는다
                 X-Enode-Log-Capped    상한 도달           NC-4
```

---

# 4. 순서

전문은 `services.md`. **세 순서가 이 설계를 지탱한다.**

```text
   단계가 도는 순서    링 Reset -> 업로더 -> tee 갈래 둘 -> 하네스 -> 청크 -> Close
                      -> 선별본 업로드.  **청크와 선별본이 다른 파일로 간다**
   재시도의 순서       attempt 가 오른다 -> 링이 비워진다 -> 첫 청크가 attempt 를 싣는다
                      -> Mediator 가 진행 파일을 비운다 -> 화면이 헤더로 알고 카드를 비운다
   봉인의 순서         DropProgress 가 가장 먼저다.  다만 Q2 = A 라 **순서가 틀려도
                      원문이 영구가 되지 않는다** — 진행 트리는 seal 의 Walk 밖이다
```

**순서는 값이고 자리는 안전장치다. 둘을 함께 둔다.**

---

# 5. 의존

전문은 `component-dependency.md`.

```text
   는 임포트 셋       enode -> transcript · api -> transcript · panel -> transcript
   깨는 금지 0        여섯 전부 안 깬다.  transcript 의 행이 전부 금지인 것이 뼈다
   미는 경로 0        Mediator 는 노드에 접속하지 않는다 (ADR-014 결정 2)
```

## 5.1 「파서는 하나다」가 언어 경계를 넘는 법

JS 는 Go 패키지를 임포트할 수 없다. 그래도 파서가 둘이 되면 안 된다.

```text
   제어판   handleTranscript 가 링을 읽어 **사건 배열로** 낸다.  원문 토글용 data 도 함께
   현황판   as=events 로 받는다.  Mediator 가 같은 파서를 거친다.  원문 토글은 as=raw
   결과     JS 가 JSON 줄을 해석하는 자리가 0 이다.  둘 다 이미 판 것을 그린다
```

**이것이 CB1 · CB2 · CB4 의 「같은 모양」을 세우는 장치다.**

---

# 6. NC-1 ~ NC-6 이 어느 자리인가

`user-stories.md` 가 「보인다」까지만 적고 자리를 이 단계에 넘겼다.
**표시의 모양은 Functional Design 이 닫는다.**

| | 완료 조건 | 자리 | 값을 내는 쪽 |
|---|---|---|---|
| NC-1 | 마지막 사건 이후 경과 | 제어판 카드 | 화면이 뺀다 (Q3 = A) |
| NC-2 | 링이 감겨 앞이 잘렸다 | 제어판 카드 | `ReadRing` 의 `total` 이 이미 난다 |
| NC-3 | 링에 원문이 남는다 | 제어판 카드 | 문장이다. 값이 아니다 |
| NC-4 | 진행 파일이 상한에 닿았다 | 응답과 화면 둘 | `X-Enode-Log-Capped` |
| NC-5 | 봉인 전후의 갈림 | `mediator-api` 문서와 응답 | `X-Enode-Log-Source` |
| NC-6 | 마지막 갱신 시각 | 현황판 단계 카드 | 화면이 뺀다 (Q3 = A) |

**배정 안 된 NC 가 0 이다.** 어느 유닛이 지는지는 Units Generation 이 센다.

---

# 7. 이 단계가 안 한 것

```text
   유닛 분해와 파일 행렬   D6.  Units Generation 의 몫이다
   사건 모형의 필드       Functional Design.  Kind 여섯의 이름만 여기서 고정했다
   진행 파일의 형식       Functional Design.  자리와 이름 규칙만 여기서 고정했다
   곁파일의 이름과 형식    Functional Design.  「시도를 파일 옆에 적는다」만 여기서 정했다
   NC 표시의 모양         Functional Design.  자리와 값을 내는 쪽만 여기서 정했다
   N1 함대 규모 · N2 수명  NFR Requirements.  값이 없는 것을 여기서 지어내지 않는다
   폴링 간격 · 청크 주기    팩과 요구가 값으로 닫았다.  안 건드린다
```

---

# 8. 확장 준수 요약 — Application Design 단계

`security-baseline` 만 켜져 있다 (`decisions.md` §1 · 사용자 결정 2026-09-11).

| 규칙 | 판정 | 근거 |
|---|---|---|
| SECURITY-03 민감정보 | 준수 | **원문이 앉는 자리 둘을 그림에 주황으로 표시하고 이름으로 적었다** (`component-dependency.md` 2절). Q2 = A 가 잔여 ① 의 최악(영구화)을 구조로 막는다 — 진행 트리가 `seal` 의 `Walk` 밖이다. Q1 = A 가 노출 기간을 Run 전체에서 시도 하나로 줄인다 |
| SECURITY-05 입력 검증 | 준수 | GET 의 인자 넷(`seq` · `from` · `as` · `name`)과 PUT 의 `attempt` 에 규칙을 박고 밖은 `400` 으로 적었다. 화면은 본문을 텍스트로만 그린다 |
| SECURITY-11 보안 설계 | 준수 | 새 표면 넷의 실패 등급을 표로 갈랐다. **보조 실패가 실행을 안 막는 것**이 설계의 값으로 적혀 있다 |
| SECURITY-13 신뢰 경계 | 준수 | 파서의 입력이 신뢰할 수 없는 하네스 출력임을 적고, 아는 키만 읽는 `ParseLine` 의 규율을 **`internal/transcript` 로 옮겨** 짓는 쪽과 읽는 쪽이 같은 규율을 쓰게 했다 |
| 나머지 열하나 | N/A | 설계 문서는 코드 · 네트워크 · 자격증명 표면을 실제로 만들지 않는다. 그 규칙들은 Construction 에서 다시 걸린다 |

**차단 findings 0.**
