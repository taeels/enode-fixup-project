# U3 `progress-store` — Functional Design 계획

**유닛** `progress-store` · **브랜치** `unit/progress-store` · **담당** taeels ·
**지는 게이트** 없다 (코드 게이트만. CB3 의 재료다) · **딛는 게이트** 없다 ·
**NFR 요구** 돈다 — **N2 를 진다**

정본 입력은 `aidlc-docs/v3-run-transcript/inception/application-design/` 의 넷과
`requirements/transcript/` 팩이다. 이 계획은 그 위에서 **진행 파일의 형식과
규칙**만 짓는다 — 코드는 다음 단계다.

---

## 0. 이 단계가 닫는 것과 안 닫는 것

```text
   닫는다    진행 파일의 실제 표현 — 자리 · 이름 · 권한 · 무엇이 한 줄인가
             시도를 어디에 적나 (Application Design 이 「곁파일」까지만 적고 넘겼다)
             시도가 바뀔 때의 비우기 규칙과 뒤늦은 청크의 처리
             총 길이 상한의 규칙 — 어디서 자르고 무엇을 남기나
             D4 의 범위 — AppendLog 에도 거나, AppendProgress 에만 거나
             N2 의 규칙 모양 — 진행 파일의 수명을 누가 언제 끊나
             Seal 안에서 DropProgress 가 서는 자리
             동시 쓰기 규칙 — record 에 오늘 락이 0 이다

   안 닫는다  N2 의 숫자           NFR Requirements.  주기 · 나이 상한을 여기서 안 짓는다
             라우트 · 쿼리 · 헤더   U4.  HTTP 는 이 유닛의 것이 아니다
             입력 검증의 400 문구   U4 (SECURITY-05)
             업로더의 주기와 버퍼    U7.  decisions.md 2절이 값으로 닫았다
             사건 모형의 필드       U1
             시험 목록과 파일별 diff  이 유닛의 Code Generation
```

---

## 1. 실측 — 코드가 지금 어떻게 생겼나 (2026-09-15)

계획을 짓기 전에 `internal/record/record.go` 전문과 호출자 전부를 읽고
`go build ./...` 과 `go test ./internal/record/` 를 돌렸다. 둘 다 초록이다.
**아래 여덟이 문서와 코드가 갈린 자리이고 이것이 이 단계의 산출이다.**

### 1.1 D4 의 근거가 낡았다 — 청크가 `AppendLog` 로 안 온다

`requirements.md` 2.4 가 D4 의 근거다.

> 오늘은 단계마다 한 번 부르므로 `MaxBlobBytes`(기본 10 MiB)가 단계 상한이다.
> 청크로 바꾸면 **청크마다 10 MiB** 가 되어 상한이 사실상 사라진다.

그 문장은 팩의 원설계(청크를 `logs/` 에 붙인다)를 전제한다. **질문 1 = B 가 그
전제를 무르고 파일을 둘로 갈랐다.** 그 뒤의 호출 셈은 이렇다.

```text
   AppendProgress   청크마다.  U7 의 업로더가 2초 · 64 KiB 로 민다
   AppendLog        단계마다 한 번.  오늘과 같다
                    claim.go:647  명령 단계.  buf 를 통째로 한 번
                    claim.go:796  하네스 단계.  selectLogs 결과를 통째로 한 번
```

`grep -rn AppendLog` 의 비테스트 호출자는 `internal/api/api.go:887` **하나**다.
**「청크마다 10 MiB」가 되는 경로가 오늘 설계에 없다.** D4 를 그대로 두면
근거 없는 변경이 되고, 빼면 1.3 이 남는다 — 물음 2 가 그 자리다.

**D4 의 나머지 절반(반환값의 뜻)은 그대로 참이다.** `api.go:887` 이
`if _, err :=` 로 값을 버리므로 호출자가 안 깨지고 시그니처가 같아 컴파일도
안 깨진다. 실측으로 확인했다.

### 1.2 잘림 표시가 두 자리에서 틀린다 — 옳은 쪽이 같은 파일에 있다

```go
// record.go:65
n, err := io.Copy(f, io.LimitReader(r, limit))
if n == limit {
	_, _ = fmt.Fprintf(f, "\n... log truncated at %d bytes\n", limit)
}
```

```text
   틀린 것 하나   n == limit 는 「잘렸다」가 아니라 「딱 맞았다」도 참이다.
                 정확히 상한 바이트인 로그에 안 잘렸는데 잘렸다고 적힌다

   틀린 것 둘     여러 번 부르면 부를 때마다 박힌다.  requirements.md 2.4 가
                 「파일 중간에 여러 번 박힌다」로 이미 이름 붙인 자리다

   옳은 쪽        같은 파일의 WriteBlob (record.go:287) 이 limit+1 을 읽고
                 n > limit 로 센다.  초과를 한 겹 더 읽어 가른다
```

**옳은 패턴이 같은 파일 안에 이미 있다.** `AppendProgress` 는 그쪽을 쓰고
`AppendLog` 도 함께 고칠지가 물음 2 · 3 · 4 다.

### 1.3 `logs/NN-*.log` 에 시도가 없다 — D4 가 봉인 기록의 거동을 바꾼다

```text
   logs/     %02d-%s.log        시도가 없다 (record.go:58)
   blobs/    %02d.%d-%s         시도가 있다 (record.go:271).  이유가 주석에 있다
```

`UploadLog` 도 시도를 안 싣는다 (`claim.go:145` 의 URL 에 `attempt` 가 없다).
그래서 **재시도가 같은 `logs/NN-이름.log` 에 이어 붙는다.** 오늘은 상한이
호출마다라 시도마다 10 MiB 를 받는다. **D4 가 상한을 총 길이로 바꾸면 시도 2
이후의 봉인 로그가 오늘보다 일찍 잘린다.**

**이것은 진행 파일이 아니라 봉인되는 기록의 거동 변화다.** 어느 문서도 안 적었다.
`constraints.md` §5 가 「Record 의 `logs/NN-<단계>.log` 이름과 자리」를 안 건드린다고
했고 상한은 그 목록에 없지만, 봉인 묶음의 내용이 줄어드는 것은 성질 4(자기충족)에
닿는다. 물음 2 가 이 자리다.

### 1.4 D1 의 인용 한 줄이 다른 줄을 가리킨다

`application-design.md` D1 의 근거가 `record.go:156` · `:196` 이다.

```text
   :156   맞다.  func seal(dir string) error 의 첫 줄이다
   :174   0444 를 거는 자리.  인용에 없다
   :180   0555 를 거는 자리.  인용에 없다
   :196   unseal 의 Walk 안 return err 다.  0444 도 0555 도 아니다
   :189-191  「보관 정책이 없다」의 실제 근거.  unseal 의 주석이다 —
             "장기 보관 정책은 MVP 밖, 쌓아두기만 한다 ... 보관 정책이 생기면
              여기가 그 입구다"
```

**D1 의 판단은 그대로 참이고 인용만 빗나갔다.** 이 저장소는 줄 번호로 근거를
대는 집이라 그 자리를 적는다 — 고치는 것은 진행자의 몫이다 (5절).

### 1.5 N2 의 오늘 값 — 상한이 없는 것이 아니라 **지우는 코드가 0** 이다

N2 는 「진행 파일이 디스크에 앉아 있는 기간의 상한」이다. 오늘 값을 코드로 쟀다.

```text
   기록 루트의 회수    grep -rn "RemoveAll" internal/ 이 <Root> 를 지우는 줄을
                     0 개 낸다.  run-<id> 를 지우는 코드가 저장소에 없다
   봉인의 회수         없다.  Seal 은 잠글 뿐 안 지운다
   unseal              테스트 정리에만 쓴다 (주석).  제품 경로에서 안 불린다
```

**그러므로 진행 파일의 수명은 `DropProgress` 를 무엇이 부르느냐가 전부다.**
`Seal` 하나에 매달면 구멍이 셋 남고, 셋 다 코드로 확인했다.

```text
   ①  sealExpired 의 창이 한 시간이다 (reap.go:126)
         WHERE state IN ('SUCCEEDED','FAILED') AND ended_at > now() - interval '1 hour'
       Mediator 가 한 시간 넘게 죽어 있던 사이에 끝난 Run 은 여기서 안 잡힌다.
       영원히 안 봉인되고 진행 트리가 영원히 남는다

   ②  sealExpired 는 n > 0 일 때만 돈다 (reap.go:104).  회수된 임대가 0 이면
       그 주기에는 아예 안 훑는다

   ③  sealRecord 가 s.Records == nil 이면 아무것도 안 하고 nil 을 돌려준다
       (seal.go:177).  그 구성에서는 봉인이 0 이므로 DropProgress 도 0 이다
```

**N2 를 「봉인이 끊는다」로만 적으면 상한이 여전히 없다.** 물음 5 가 이 자리다.
값(주기 · 나이)은 NFR Requirements 로 넘긴다.

### 1.6 `Seal` 의 조기 반환이 `DropProgress` 의 자리를 정한다

```go
// record.go:92
func (s *Store) Seal(...) error {
	if s.Sealed(runID) {
		return nil // 봉인은 멱등이다
	}
	if err := s.Open(runID); err != nil { ... }
```

`application-design.md` 4절이 「봉인의 순서 — `DropProgress` 가 가장 먼저다」로
적었는데 **「가장 먼저」가 조기 반환보다 앞인지 뒤인지가 안 적혀 있다.**

```text
   조기 반환보다 앞   이미 봉인된 Run 을 다시 Seal 해도 진행 트리를 지운다.
                    늦게 도착한 청크가 만든 고아 트리를 두 번째 봉인이 걷는다
   조기 반환보다 뒤   첫 봉인만 지운다.  두 번째 Seal 은 return nil 로 끝난다
```

**늦게 도착한 청크가 실제로 있다.** `putLog` 는 DB 의 `run.State` 로 410 을 내고
(`api.go:879`) `Sealed()` 를 안 본다. 상태 전이와 `sealRecord` 사이에 검사를 이미
지난 요청이 있으면 `DropProgress` 뒤에 디렉터리가 다시 생긴다. 물음 5 의 대가에
이 자리를 적었다.

### 1.7 `internal/record` 에 락이 하나도 없다

`internal/record` 는 `record.go` 하나이고 `sync` 를 임포트하지 않는다.
`AppendLog` 은 `O_APPEND` 로 열어 `io.Copy` 한다 — 커널이 append 위치를 원자로
잡아주므로 덮어쓰기는 없지만 **32 KiB 단위의 쓰기가 섞일 수는 있다.**

```text
   오늘 안 부딪치는 이유   단계마다 한 번이고 단계마다 파일이 다르다.
                        한 노드는 한 번에 한 단계다 (transcript.go 의 Ring 주석)

   청크 아래서 부딪치는 자리
     총 길이를 무엇으로 재나   쓰기 뒤 Stat 은 남의 바이트까지 센다
     시도가 바뀌는 순간        비우기(자르기)와 이어 붙이기가 같은 파일에서 겹친다
```

링은 같은 문제를 `sync.Mutex` 로 이미 풀었다 (`transcript.go:40`). 물음 7 이 이
자리다.

### 1.8 `safe()` 는 경로 탈출만 막는다 — 돌려서 확인했다

D1 의 경로가 `safe()` 를 전제하므로 실제로 돌렸다 (`record.go:38` 의 함수를
그대로 떼어 실행).

```text
   "../../etc/passwd"  ->  "____etc_passwd"     탈출이 막힌다
   "a/../b"            ->  "a___b"              막힌다
   "..."               ->  "_."                 남는다.  탈출은 아니다
   ""                  ->  ""                   빈 이름 -> "01-.log"
   "step\nname"        ->  "step\nname"         개행이 그대로 파일 이름에 간다
   "step\x00name"      ->  "step\x00name"       NUL 이 그대로.  os 가 뒤에서 거절한다
   300자                ->  300자               길이 상한이 없다.  ENAMETOOLONG
```

**경로 탈출은 확실히 막힌다** (`TestPathTraversalIsBlocked` 가 그것을 잰다).
막히지 않는 것은 제어문자와 길이이고 **이것은 오늘도 참인 기존 사실이다** —
`name` 은 `r.URL.Query().Get("name")` 에서 온다 (`api.go:884`). U3 이 만든 결함이
아니지만 진행 트리가 같은 규칙을 물려받는다. 물음 9 가 그 자리다.

### 1.9 진행 트리가 `seal` 의 `Walk` 밖이라는 D1 의 안전장치를 확인했다

```text
   seal 과 Tar 가 도는 곳   s.dir(runID) = <Root>/run-<safe(id)>   (record.go:32)
   진행 트리                 <Root>/progress/run-<safe(id)>/        (D1)
```

**형제다.** `seal` 의 `Walk` 도 `Tar` 의 `Walk` 도 `<Root>/progress` 를 안 지난다.
`dir()` 이 언제나 `run-` 을 앞에 붙이므로 어떤 `runID` 로도 디렉터리 이름이
`progress` 가 될 수 없다 — 이름 충돌도 0 이다. **D1 의 「순서가 틀려도 원문이
영구가 되지 않는다」가 코드로 참이다.**

### 1.10 `Open` 이 만드는 것과 안 만드는 것

`Store.Open` 은 `steps` · `logs` · `blobs` 셋만 `0755` 로 만든다 (`record.go:45`).
`internal/store/store.go:427` 이 Run 을 만들 때 부른다. **진행 트리는 아무도 안
만든다.** 그리고 `Seal` 자신이 `s.Open(runID)` 를 부른다 (`record.go:96`) —
`Open` 에 진행 트리를 더하면 **봉인이 지우기 직전에 다시 만든다.** 물음 8 이 이
자리다.

### 1.11 커버리지 계약이 이 패키지에 거는 값

```text
   .coverage-contract.yml:130   internal/record   statements 159 · covered 131
   오늘 비율                    82.4%
   CI 의 하한                   패키지별 80%
```

새 문장 N 개를 더하고 C 개를 덮으면 `(131+C)/(159+N) >= 0.80` 이어야 한다.
겉면 셋에 곁파일까지 들어오므로 **새 문장의 약 3/4 이상을 덮어야 안 빨개진다.**
시험 목록은 Code Generation 이 지지만 **그 예산이 물음 1 의 답에 달려 있다** —
곁파일이 늘면 문장이 는다.

### 1.12 안 만져도 되는 것을 확인했다

```text
   internal/store         diff 0.  sealRecord 는 record.Seal 을 그대로 부른다.
                          DropProgress 를 record 안에서 부르면 store 가 안 바뀐다
   internal/api           diff 0 (U3 기준).  api.go:887 이 값을 버려 D4 가 안 깬다
   기존 시험 둘이 D4 아래서 산다 — 한 번 호출이라 총 길이와 호출 바이트가 같다
     record_test.go:282   TestLogTruncationIsMarked      100 바이트 · 상한 10
     api/log_test.go:194  TestLog_Oversize...NotRefused  256 바이트 · 상한 64
   record -> store 임포트가 오늘 0 이다.  곁파일을 store 에 안 묻는 근거가 이것이다
```

---

## 2. 물음 아홉

답을 `[Answer]:` 뒤에 적는다. **권장이 있는 물음은 권장을 A 에 둔다.**
각 선택지 끝의 한 줄이 그 선택의 대가다.

### Question 1

**시도를 어디에 적나.** `component-methods.md` 2.1 이 「진행 파일 옆의 작은
곁파일이다. 이름과 형식은 Functional Design 이 닫는다」까지만 적었다.
읽는 쪽은 `AppendProgress` 가 「파일에 적힌 시도」와 인자를 대조해야 한다.

A) **파일 이름에 넣는다** — `NN-<safe(name)>.<attempt>.log`. 곁파일이 0 이다.
비우기는 「새 이름으로 연다」이고 총 길이가 자연히 0 부터 다시 센다. 같은 파일의
`blobPath` 가 이미 `%02d.%d-%s` 로 같은 답을 쓴다 (`record.go:271`).
**대가** — 앞 시도의 파일이 봉인까지 디스크에 남는다. 읽는 쪽이 「가장 큰 시도」를
고르는 한 겹이 필요하다 (`OpenBlob` 이 이미 그 모양이다)

B) **곁파일 하나** — `NN-<safe(name)>.attempt` 에 십진수 한 줄.
**대가** — 파일이 두 배가 되고 둘의 원자성을 따로 지켜야 한다. 곁파일만 남고
로그가 없는 상태가 생긴다

C) **진행 파일 머리에 적는다** — 링과 같은 고정 머리(매직 · 시도 · 총 길이).
파일 하나가 스스로를 설명한다. `transcript.go` 의 링이 이미 그 모양이다.
**대가** — 파일이 순수한 바이트열이 아니게 되어 `from` 오프셋이 머리만큼 밀린다.
U4 와 U7 이 그 밀림을 알아야 한다

D) **Run 당 JSON 하나** — `<Root>/progress/run-<id>/attempts.json` 에 seq -> attempt.
**대가** — 단계 둘이 동시에 그 한 파일을 고친다. 1.7 의 락 문제를 Run 전체로 넓힌다

E) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2

**D4 의 상한을 어디까지 거나.** 1.1 이 D4 의 근거가 낡은 것을, 1.3 이 그대로
걸면 봉인 로그의 거동이 바뀌는 것을 보였다.

A) **`AppendProgress` 에만 총 길이 상한. `AppendLog` 은 반환값의 뜻만 바꾼다.**
청크가 오는 쪽에만 걸고 봉인되는 기록의 거동을 안 바꾼다. D4 의 「시그니처는
그대로 · 반환값은 총 길이」는 지킨다.
**대가** — 같은 파일의 두 함수가 상한 규칙이 다르다. 그 이유를 주석과
`business-rules.md` 에 적지 않으면 다음 사람이 한쪽을 버그로 읽는다

B) **둘 다 총 길이 상한.** D4 를 글자 그대로 집행한다.
**대가** — 시도 2 이후의 봉인 로그가 오늘보다 일찍 잘린다 (1.3). 성질 4 에 닿는
변화이고 이 회차의 어느 문서도 그 대가를 안 적었다

C) **둘 다 총 길이 상한 + `logs/` 파일 이름에도 시도를 넣는다.** 시도마다 파일이
갈려 B 의 대가가 사라진다.
**대가** — `constraints.md` §5 의 「Record 의 `logs/NN-<단계>.log` 이름과 자리」를
정면으로 무른다. `UploadLog` 시그니처와 봉인 tar 의 파일 이름이 함께 바뀌어
U7 · CB3 의 `cmp` 명령까지 번진다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3

**상한에 닿는 청크를 어떻게 자르나.** 진행 파일의 내용은 NDJSON 이고 읽는 쪽이
줄 단위 파서다 (`internal/transcript`).

A) **상한 앞 마지막 개행까지만 쓰고 나머지를 버린다.** 파일이 언제나 온전한
줄로 끝나 파서가 마지막에 반쪽 사건을 안 본다.
**대가** — 총 길이가 상한보다 작은 자리에서 멈춘다. 「닿았다」와 「상한 값」이
안 같아져 헤더를 읽는 쪽이 둘을 비교하면 안 된다

B) **정확히 상한 바이트에서 자른다.** 오늘 `AppendLog` 과 같다.
**대가** — 마지막 줄이 반쪽으로 남는다. 파서가 그것을 `raw` 로 올려 화면에 깨진
JSON 이 한 줄 뜬다. `Parse` 의 `truncated` 는 **앞**이 잘린 경우의 인자라 뒤를
안 막는다

C) **넘치는 청크는 통째로 안 쓴다.** `WriteBlob` 의 태도다 — 자른 것은 안 받는다.
**대가** — 마지막 청크가 크면 상한보다 한참 앞에서 멈춘다. 64 KiB 청크면 최대
64 KiB 를 덜 보고 사람이 그 차이를 「멈췄다」로 읽을 수 있다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4

**진행 파일에 잘림 표시 줄을 박나.** 오늘 `AppendLog` 은 텍스트 한 줄을 박는다
(`\n... log truncated at %d bytes\n`). NC-4 는 「응답과 화면에 적힌다」이고
`X-Enode-Log-Capped` 가 이미 그 값을 진다.

A) **안 박는다. `Capped` 헤더 하나로 말한다.** 진행 파일은 순수한 하네스 원문으로
남아 봉인 전후의 바이트 비교가 단순해진다.
**대가** — 파일만 떼어 본 사람은 잘린 것을 모른다. 다만 진행 파일은 봉인 때
지워지므로 떼어 볼 사람이 GET 을 지나는 사람뿐이고 그는 헤더를 본다

B) **오늘과 같은 텍스트 한 줄을 박는다.** 두 파일의 규칙이 같아진다.
**대가** — NDJSON 파일 안에 JSON 이 아닌 줄이 하나 섞인다. 파서가 그것을 `raw`
로 올려 화면 맨 끝에 접힌 원문 한 줄이 뜬다

C) **NDJSON 한 줄로 박는다** — `{"type":"enode.capped","bytes":N}`. 파서가 아는
줄이 되어 화면이 문장으로 그린다. `enode.elided` 가 이미 같은 결의 선례다.
**대가** — `internal/transcript` 가 아는 종류가 하나 는다. U1 의 `Kind` 여섯에
없는 값이라 U1 의 산출물을 되열어야 한다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5

**N2 — 진행 파일의 수명을 무엇이 끊나.** 1.5 가 「봉인만」으로는 상한이 없음을
코드로 보였다. **여기서 정하는 것은 규칙의 모양이고 숫자는 NFR Requirements 다.**

A) **봉인 + 종료 상태 + 고아 쓸기 셋.** `Seal` 이 부르고, 봉인이 실패하거나
안 돌아도 Run 이 종료 상태에 이르면 끊고, 기동과 주기마다 「살아 있는 Run 이
없는 진행 트리」를 걷는다. 1.5 의 구멍 ① ② ③ 이 전부 막힌다.
**대가** — 쓸기가 Run 의 상태를 알아야 하는데 `record` 는 `store` 를 임포트하지
않는다 (1.12). 쓸기를 `store` 쪽에 두고 `record` 는 `DropProgress` 만 주는 배치가
되어 **`internal/store` 의 diff 가 0 이 아니게 된다** — `unit-of-work-file-matrix.md`
4절의 「`internal/store` 코드 diff 0」과 품질 게이트 2 가 그 자리에서 깨진다

B) **봉인 + 나이 쓸기 둘.** 쓸기가 Run 상태를 안 묻고 **마지막 쓰기로부터 얼마**
만 본다. `record` 안에서 닫히고 `store` 가 안 바뀐다.
**대가** — 아주 오래 도는 단계의 진행 파일을 살아 있는 채로 지울 수 있다. 그
나이 값이 곧 N2 이고 NFR 이 정한다. 지워진 뒤 총 길이가 0 으로 뒤로 가는데
시도는 안 바뀌어서 **화면이 카드를 안 비운다** — 그 자리를 규칙으로 함께 닫아야
한다 (아래 「함께 닫는 규칙」)

C) **봉인만. 구멍 셋을 잔여로 적고 넘긴다.** 이 회차의 변경이 가장 작다.
**대가** — N2 에 상한이 없다. 이 유닛이 N2 를 지기로 한 것과 정면으로 어긋나고
`requirements.md` 5.4 잔여 ① 의 「봉인 때 지워지지만」이 거짓이 되는 경로가 남는다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

**함께 닫는 규칙 (답이 A 나 B 면 `business-rules.md` 가 진다)** — 진행 트리가
봉인 없이 사라지면 `ReadProgress` 의 총 길이가 0 으로 되돌아간다. 시도는 안
바뀌므로 `X-Enode-Log-Attempt` 가 그 자리를 못 잡는다. **총 길이가 `from` 보다
작아진 것을 읽는 쪽이 「비워졌다」로 읽게 하는 규칙**을 이 단계가 적는다.

### Question 6

**뒤늦은 시도의 청크를 어떻게 하나.** 시도 2 가 시작한 뒤 시도 1 의 버퍼가
도착할 수 있다 (U7 의 업로더는 자기 고루틴이고 실패한 청크를 다음에 이어 보낸다).
그대로 받으면 시도 2 의 파일에 시도 1 의 바이트가 섞인다.

A) **버린다. 지금 `Progress` 를 그대로 돌려준다.** 시도는 앞으로만 간다.
노드는 돌아온 총 길이로 자기 오프셋을 맞추므로 (FR-5) 다음 청크에서 저절로
따라잡는다.
**대가** — 노드가 「보냈는데 안 들어갔다」를 모른다. 총 길이가 자기 계산과
어긋난 것으로만 안다

B) **오류로 돌려 U4 가 409 를 낸다.** 노드가 명시로 안다.
**대가** — 새 상태 코드가 하나 는다. `component-methods.md` 4.2 의 PUT 응답에
없는 값이라 U4 의 산출물을 되열어야 한다

C) **받아서 이어 붙인다.** 시도를 안 본다.
**대가** — D7 이 막으려던 것이 그대로 일어난다. 시도 1 의 문장이 시도 2 의 카드에
섞여 CB4 의 「같은 문장」이 깨진다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 7

**동시 쓰기를 무엇으로 막나.** 1.7 이 `record` 에 락이 0 인 것과 청크 아래서
부딪치는 자리 둘을 보였다.

A) **`Store` 에 (run, seq, name) 별 뮤텍스를 둔다.** 비우기와 이어 붙이기가 한
단계 안에서 직렬이 되고 총 길이를 락 안에서 잰다. 다른 단계는 안 기다린다.
**대가** — 맵이 자라고 언제 비우나가 는다. `DropProgress` 가 그 자리를 함께
걷어야 한다

B) **`Store` 에 뮤텍스 하나.** 가장 단순하다. `record` 의 쓰기는 원래 드물다.
**대가** — 단계가 여럿인 Run 에서 청크 PUT 이 서로 기다린다. 2초 주기라 실제로는
안 밀릴 가능성이 높지만 그 판단이 측정이 아니라 짐작이다

C) **락 없이 간다.** `O_APPEND` 를 믿고 총 길이는 쓰기 직후 `Stat` 으로 잰다.
**대가** — 비우기(자르기)와 이어 붙이기가 겹치는 순간이 안 막힌다. Q6 = A 로
뒤늦은 시도를 버려도 **같은 시도의 첫 청크와 비우기**가 겹칠 수 있다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 8

**진행 트리를 언제 만들고 권한을 무엇으로 두나.** 1.10 이 `Seal` 자신이
`s.Open(runID)` 를 부르는 것을 보였다. `requirements.md` 5.4 가 **파일**은
0600 으로 이미 못 박았고 디렉터리와 생성 시점이 안 적혀 있다.

A) **`AppendProgress` 가 게으르게 `MkdirAll(0700)` 한다. 파일은 0600.**
`Open` 을 안 건드리므로 `Seal` 의 `s.Open` 이 지운 트리를 다시 만드는 일이 없다.
쓴 적 없는 Run 은 디렉터리도 0 이다.
**대가** — 청크마다 `MkdirAll` 을 부른다. 이미 있으면 syscall 하나이고 2초 주기라
무시할 수 있지만 그 판단을 근거로 적어야 한다

B) **`Open` 이 Run 시작에 함께 만든다.** 자리가 한 곳에 모인다.
**대가** — `Seal` 이 `s.Open` 을 먼저 부르므로 (`record.go:96`) `DropProgress` 를
그 뒤에 두어야만 맞는다. **「DropProgress 가 가장 먼저다」와 정면으로 부딪친다.**
그리고 트랜스크립트를 안 쓰는 Run 도 빈 디렉터리를 남긴다

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 9

**`record` 가 `name` 을 스스로 검증하나.** 1.8 이 `safe()` 가 경로 탈출만 막는
것을 보였다. `unit-of-work.md` 가 입력 검증(SECURITY-05)을 **U4** 에 배정했다.

A) **U4 를 믿는다. `record` 는 `safe()` 만 쓴다.** 규칙이 한 자리에 있고 두 벌이
안 된다 (`CONVENTIONS.md` 1.4 의 결). 오늘 `logs/` 와 같은 태도다.
**대가** — `record` 를 직접 부르는 다음 호출자가 생기면 방어가 0 이다. 오늘
호출자는 `api.go` 하나뿐이라 지금은 참이다

B) **`record` 도 건다** — 길이 상한과 제어문자 거절을 `AppendProgress` 안에.
표면 두 겹이 각자 막는다.
**대가** — 같은 규칙이 두 자리에 산다. 값이 갈리면 U4 는 400 을 내는데 `record`
는 받는(또는 그 반대) 상태가 된다

C) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## 2.1 질문 범주 여덟 — 전부 평가했다

`functional-design.md` Step 3 이 범주 여덟을 전부 보라고 한다.
**안 묻는 범주는 왜 안 묻는지를 적는다.**

| 범주 | 물음 | 안 물으면 그 이유 |
|---|---|---|
| Business Logic Modeling | 2 · 3 · 5 | — |
| Domain Model | 1 | — |
| Business Rules | 6 · 9 | — |
| Data Flow | 3 · 5 · 7 | — |
| Integration Points | **안 묻는다** | 이 유닛의 접점이 `Store` 의 메서드 호출뿐이다. HTTP · 라우트 · 헤더는 U4 의 것이고 업로더는 U7 이다. `record -> store` 임포트가 오늘 0 이고 이 유닛이 그것을 안 바꾼다 (Q5 = A 만 예외이며 그 대가를 선택지에 적었다) |
| Error Handling | 4 · 6 | 실패 등급은 `application-design.md` 2.1 이 이미 값으로 닫았다 — 진행 트리 지우기는 **보조**다. 남은 갈림 둘만 물었다 |
| Business Scenarios | 6 · 8 | 재시도 · 늦은 청크 · 봉인 뒤 도착이 그 자리이고 물음 둘이 덮는다 |
| Frontend Components | **해당 없음** | 이 유닛은 화면을 0 줄 만진다. 파일 행렬이 `internal/record/record.go` 하나만 준다. 카드는 U5 · U8 |

**추가로 묻지 않은 것 — 코드가 답을 줬다.**

```text
   진행 파일의 자리와 이름     D1 이 값으로 닫았고 1.9 가 그것이 seal 의 Walk 밖임을 확인했다
   safe 를 다시 지을 것인가    안 짓는다.  같은 패키지에 이미 있다 (CONVENTIONS 1.4)
   DropProgress 의 멱등성      component-methods.md 2.1 이 닫았다.  I4 와 같은 결
   Progress 의 필드 셋         application-design.md 3.2 가 닫았다
   AppendLog 의 시그니처       D4 가 닫았고 api.go:887 이 값을 버리는 것을 실측했다
   진행 파일의 권한 0600       requirements.md 5.4 가 값으로 닫았다.  디렉터리만 물었다
```

---

## 2.2 모순 검사 — 답 아홉을 서로 대 봤다 (2026-09-15)

답은 **Q1 ~ Q9 전부 A** 다 (사용자 「권장대로」). `functional-design.md` Step 5 가
산출물을 짓기 전에 요구하는 검사를 돌렸다.

### 2.2.1 서로 부정하는 답 — 0

맞물리는 셋을 먼저 봤고 나머지 쌍도 훑었다. **답끼리 부정하는 자리가 0 이다.**
다만 셋 다 **파생 규칙**을 낳으므로 그것을 산출물이 적어야 한다.

| 맞물림 | 판정 | 낳은 파생 규칙 |
|---|---|---|
| Q1 시도를 파일 이름에 · Q6 뒤늦은 청크를 버린다 | 모순 0 | 곁파일이 없으므로 **「지금 시도」가 디렉터리 스캔의 최댓값**이 된다. `ReadProgress` 에 attempt 인자가 없어 강제된다. `OpenBlob` 이 이미 같은 스캔이다 (`record.go:326`) |
| Q2 상한은 `AppendProgress` 만 · Q3 자르는 자리 · Q4 표시 | 모순 0 | Q3 · Q4 는 `AppendProgress` 만 말한다. **`AppendLog` 의 반환값은 총 길이인데 잘림 판정은 이번 호출 바이트로 해야 한다** — `total == limit` 로 적으면 재시도에서 깨진다 |
| Q5 수명을 끊는 셋 · Q8 게으른 생성 | 모순 0 | Q8 = A 가 `Seal` 의 `s.Open` (`record.go:96`) 이 진행 트리를 다시 안 만들게 하므로 **`DropProgress` 를 조기 반환 앞에 둘 수 있다.** 1.6 이 열어 둔 자리가 Q8 로 닫힌다 |

**나머지 쌍에서 나온 파생 규칙 다섯.**

```text
   Q1     safe(name) 에 점이 들어갈 수 있다.  이름을 **오른쪽에서** 파싱해야 한다 —
          parseBlobName 은 왼쪽 파싱이라 재사용이 안 된다.  새 헬퍼가 하나 늘고
          1.11 의 커버리지 예산이 그만큼 는다

   Q3     개행 정렬은 **상한에 닿을 때만** 한다.  평소 청크를 개행에 맞추면 Total 이
          노드가 보낸 것보다 뒤처지고 노드가 그 차이를 다시 보내 루프가 돈다

   Q3     남은 예산 안에 개행이 하나도 없으면 0 바이트를 쓰고 상한 도달로 넘어간다

   Q7     뮤텍스 키에 attempt 를 **안 넣는다.**  「지금 시도 고르기」와 쓰기가 한
          임계 구역이어야 하므로 키는 (run, seq, name) 이다

   Q7·Q5  드롭이 맵에서 뮤텍스를 지우는 순간 그것을 쥔 쓰기가 있으면 다음 호출이
          새 뮤텍스를 만들어 둘이 동시에 돈다.  드롭과 쓰기가 안 겹치는 것을
          불변식으로 적어야 한다
```

### 2.2.2 검사가 뒤집은 것 — 1.5 의 구멍 ③ 은 없다

Part 1 이 N2 의 구멍을 셋으로 적었다. 검사하면서 배선을 따라가 **셋째가 성립하지
않는 것을 확인했다.**

```text
   api.go:50     records: st.Records   API 와 store 가 **같은 포인터**를 든다
   api.go:56     needRecords 가 nil 이면 503 을 내고 putLog 가 그것을 먼저 부른다
   결과          Records 가 nil 이면 PUT 자체가 503 이라 진행 트리가 애초에 안 생긴다.
                 지울 것이 없으므로 구멍이 아니다
```

**구멍은 둘이다** — `reap.go:126` 의 한 시간 창과 `reap.go:104` 의 `n > 0`.
1.5 의 셋째 줄은 산출물에서 「구멍이 아니었다」로 반증과 함께 적는다.

### 2.2.3 이름 붙인 긴장 하나 — Q1 = A 가 D7 의 보안 credit 을 되돌린다

모순은 아니고 **대가가 상류 문서의 근거와 어긋나는 자리**다. 사용자가 Q1 = A 의
대가 줄(「앞 시도의 파일이 봉인까지 디스크에 남는다」)을 보고 골랐으므로 답을
바꾸지 않는다. **적어 두는 것이 이 단계의 몫이다.**

```text
   상류가 적은 것   application-design.md 8절 SECURITY-03
                   「Q1 = A 가 노출 기간을 Run 전체에서 **시도 하나로 줄인다**」
                   (그 Q1 은 D7 이다 — 시도가 바뀌면 진행 파일을 비운다)

   이 답이 하는 것   비우기가 「자르기」가 아니라 「새 이름으로 연다」가 된다.
                   시도 1 의 원문이 봉인 또는 쓸기까지 디스크에 남는다 —
                   노출 기간이 시도 하나가 아니라 사실상 Run 전체로 돌아간다

   좁히는 길        새 시도의 첫 청크가 앞 시도의 파일을 지우면 Q1 = A 의 이름
                   규칙을 그대로 두고 credit 을 되찾는다.  읽는 쪽은 안 다친다 —
                   시도가 바뀌면 화면이 카드를 비우므로 앞 시도를 아무도 안 읽는다
```

**짐작으로 고르지 않는다.** 아래 물음 11 이 그 자리다.

**닫혔다 — 물음 11 = B (2026-09-15).** 새 시도의 첫 청크가 앞 시도의 파일을
걷는다. 노출 기간이 다시 시도 하나가 되어 **`application-design.md` 8절의
SECURITY-03 판정과 `requirements.md` 5.4 의 잔여 ① 은 그대로 선다 — 안 고친다.**
5.1 의 미확정분에서 그 둘을 걷어냈다. Q1 = A 의 대가 줄(「앞 시도의 파일이
봉인까지 디스크에 남는다」)은 **이 답으로 무효가 됐다** — 이름 규칙만 남는다.

### 2.2.4 막는 것 하나 — Q3 = A 와 Q4 = A 가 합쳐지면 GET 이 상한 도달을 못 안다

**이것이 산출물을 지금 못 짓게 하는 이유다.** 답 둘이 각각은 멀쩡한데 **합쳐지면
`Progress.Capped` 를 읽는 쪽에서 채울 근거가 사라진다.**

```text
   Q3 = A    상한 앞 마지막 개행까지만 쓴다  ->  닿았을 때 Total < limit 다
   Q4 = A    진행 파일에 표시 줄을 안 박는다  ->  파일 안에 흔적이 0 이다
   합치면     「닿아서 멈춘 파일」과 「그냥 아직 작은 파일」이 둘 다 Total < limit 이고
             파일만 봐서는 안 갈린다
```

겉면이 그 자리를 못 메운다 — `component-methods.md` 3.2 가 서명을 이미 닫았다.

```go
func (s *Store) ReadProgress(runID string, seq int, name string,
	from int64) (Progress, io.ReadCloser, error)   // limit 인자가 없다
```

**limit 을 넘겨도 못 푼다.** Q3 = A 아래서는 `Total >= limit` 가 판정이 아니기
때문이다. Part 1 의 Q3 = A 대가 줄이 그것을 이미 적었다 — 「헤더를 읽는 쪽이
둘을 비교하면 안 된다」.

```text
   AppendProgress   limit 을 인자로 받으므로 Capped 를 낼 수 있다.  U7 이 여기서 안다
   ReadProgress     낼 근거가 0 이다.  U4 의 GET 이 여기서 막힌다
```

**US-7 과 NC-4 가 그 자리에서 죽는다.** NC-4 의 문장이 「**응답과** 화면에
적힌다」이고 화면 둘(U5 · U8)은 GET 이 실은 값을 그린다. GET 이 못 말하면
사람이 「멈췄다」와 「끝났다」를 못 가르고, 그것이 US-7 이 막으려던 오독이다.

**Part 1 에서는 안 보였다.** Q3 과 Q4 가 아직 답이 없어 조합이 없었다.
`functional-design.md` Step 5 가 「새 모호가 있으면 진행하지 말라」고 한 자리라
**산출물 셋을 여기서 멈추고 물음 10 을 낸다.**

**닫혔다 — 물음 10 = B (2026-09-15).** 상한에 닿을 때 진행 파일에 NDJSON 한
줄을 박는다. **파일이 스스로를 설명하므로 `ReadProgress` 가 서명을 안 바꾸고
`Capped` 를 낼 수 있다.** 응답과 화면이 한 출처에서 서고 NC-4 와 US-7 이 산다.

```text
   사용자가 본 값   상한의 오늘 값을 먼저 쟀다 — MaxBlobBytes 기본 **10 MiB**
                   (config.go:100 의 10 << 20).  mediator yaml 의 max_blob_bytes
                   로 바뀌고 api.go:887 이 그 값을 그대로 넘긴다.
                   **닿는 일이 드물다**는 것을 보고 골랐다

   산 것            NC-4 가 요구한 둘(응답 · 화면)이 한 출처에서 선다
   치른 것          Q4 = A 가 피하려던 「NDJSON 아닌 줄이 섞인다」.
                   다만 B 는 NDJSON 이라 그 대가가 아니고, 남는 것은
                   internal/transcript 의 Kind 가 하나 느는 것뿐이다

   대가가 지금 거의 0 인 이유   계획이 「U1 이 이미 병합됐을 수 있다」를 대가로
                   적었는데 **U1 은 병합이 0 이다.**  진행자가 U1 에 같은 지시를
                   병렬로 보내 Kind 를 늘린다
```

**U3 이 지는 것은 「내보내는 쪽」뿐이다.** `Kind` 와 파싱 규칙은 U1 이 진다 —
두 벌로 두지 않는다 (`CONVENTIONS.md` 1.4).

### Question 10

**상한에 닿았다는 사실을 어디에 남기나.** 2.2.4 가 GET 이 그것을 못 아는 것을
보였다. 답이 없으면 `domain-entities.md` 의 `Progress.Capped` 행과
`business-rules.md` 의 상한 규칙을 못 짓는다.

A) **Q3 을 B 로 되돌린다** — 정확히 상한 바이트에서 자른다. `Total == limit` 가
판정이 되어 `ReadProgress` 가 limit 없이도... 못 푼다. **limit 인자가 필요하다.**
겉면이 한 칸 늘고 `component-methods.md` 3.2 를 되연다.
**대가** — 마지막 줄이 반쪽으로 남아 파서가 깨진 JSON 을 `raw` 로 올린다.
Q3 = B 의 대가가 그대로 돌아온다

B) **Q4 를 C 로 되돌린다** — 상한에 닿을 때 NDJSON 한 줄을 박는다
(`{"type":"enode.capped","bytes":N}`). 파일이 스스로를 설명하고 `ReadProgress`
가 마지막 줄로 안다. 파서가 아는 종류가 되어 화면이 문장으로 그린다.
**대가** — `internal/transcript` 의 `Kind` 가 하나 는다. **U1 의 산출물을 되열어야
한다** (U1 은 이미 병합됐을 수 있다). `enode.elided` 가 같은 결의 선례다

C) **곁파일 하나를 둔다** — `NN-<safe(name)>.<attempt>.capped` 빈 파일.
`AppendProgress` 가 닿을 때 만들고 `ReadProgress` 가 있으면 참이다.
`DropProgress` 가 트리째 걷으므로 생명주기가 따로 안 는다.
**대가** — Q1 = A 가 고른 「곁파일이 0 이다」를 무른다. 파일 수가 단계마다
최대 두 배가 된다

D) **`Store` 가 상한을 필드로 든다** — `record.New(root)` 옆에 진행 상한을 세워
`ReadProgress` 가 비교한다.
**대가** — A 와 같은 이유로 **비교가 판정이 아니다** (Q3 = A). 그리고
`cmd/mediator/main.go:184` 가 바뀌어 파일 행렬의 「`cmd/` 다섯 코드 diff 0」이 깨진다

E) Other (please describe after [Answer]: tag below)

[Answer]: B

### Question 11

**새 시도가 앞 시도의 진행 파일을 지우나.** 2.2.3 의 긴장이다. Q1 = A 의
이름 규칙은 어느 쪽이든 안 바뀐다.

A) **안 지운다. 봉인 또는 쓸기까지 남는다.** Q1 = A 의 대가 줄 그대로다.
**대가** — `application-design.md` 8절이 D7 에 준 「노출 기간이 시도 하나」가
거짓이 된다. 그 문장을 고치고 `requirements.md` 5.4 의 잔여 ① 을 넓혀 적어야 한다

B) **지운다.** 새 시도의 첫 청크가 그 단계의 앞 시도 파일을 걷는다. D7 의
보안 credit 이 그대로 서고 이름 규칙은 Q1 = A 그대로다.
**대가** — 지우는 자리가 하나 는다. 그 실패는 보조여야 하고(진행을 안 막는다)
「앞 시도를 아무도 안 읽는다」가 전제가 된다 — 오늘 화면이 시도 바뀌면 카드를
비우므로 참이지만 **전제를 규칙으로 적어야** 한다

C) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## 3. 답을 받은 뒤 낼 산출물

`aidlc-docs/taeels/construction/progress-store/functional-design/` 아래 셋이다.

**2026-09-15 셋 다 냈다.** 2.2.4 가 연 물음 10 이 **B** 로, 2.2.3 의 긴장이 물음
11 이 **B** 로 닫힌 뒤에 지었다. 규칙 **서른여섯(R1 ~ R36)** 에 번호와 근거를
달았고, 세 문서가 가리키는 규칙 번호가 전부 정의로 풀린다 — **가리키는데 정의가
없는 것 0 · 정의만 있고 아무도 안 가리키는 것 0.**

- [x] **`domain-entities.md`** — 이 유닛이 드는 형식
  - [x] 진행 트리의 자리표 — `<Root>/progress/run-<safe(id)>/` 아래 무엇이 있나.
        `<Root>/run-<id>/` 와의 관계와 **`seal` · `Tar` 의 `Walk` 밖임을 그림으로**
  - [x] 진행 파일 한 줄의 모양 — NDJSON. 표시 줄이 있으면 그 줄도 (물음 4)
  - [x] 시도의 표현 (물음 1) — 파일 이름 · 곁파일 · 머리 중 답이 고른 것의 전문
  - [x] `Progress{Total, Attempt, Capped}` 의 각 필드가 **언제 무엇을 담나**.
        파일이 없을 때 · 비워진 직후 · 상한에 닿은 뒤의 값을 표로
  - [x] 권한표 — 진행 파일 0600 · 디렉터리 (물음 8). `logs/` 의 0644 · 0755 와
        왜 다른지를 `requirements.md` 5.4 로 근거 지어
  - [x] 이름 규칙 — `safe()` 를 그대로 쓰고 안 짓는다. 1.8 의 실측표를 근거로

- [x] **`business-logic-model.md`** — 순서와 흐름
  - [x] `AppendProgress` 의 ① ~ ⑦ — 시도 대조 · 비우기 · 상한 확인 · 쓰기 ·
        총 길이 재기 · `Capped` 판정 · 반환. **락의 자리를 순서 안에 표시** (물음 7)
  - [x] `ReadProgress` 의 흐름 — 파일 없음 · `from` 이 총 길이보다 큼 · 비워진 뒤
  - [x] `DropProgress` 의 흐름과 멱등성
  - [x] **`Seal` 의 순서 그림** — `Sealed()` 조기 반환 · `s.Open` · `DropProgress`
        셋의 자리 (1.6 · 물음 8). 늦게 온 청크가 만드는 고아 트리의 경로도 함께
  - [x] **진행 파일의 수명 그림** (물음 5) — 태어나는 자리 하나와 끊는 자리 N 개.
        1.5 의 구멍 ① ② ③ 이 각각 어느 화살표로 막히는지를 이름으로
  - [x] 재시도의 순서 — attempt 상승 · 링 Reset · 첫 청크 · 비우기 · 헤더 ·
        카드 비우기. `application-design.md` 4절의 것을 U3 의 자리까지 조여서
  - [x] `AppendLog` 과 `AppendProgress` 가 갈리는 자리 (물음 2 의 답)

- [x] **`business-rules.md`** — 규칙과 불변식
  - [x] 시도 규칙 — **앞으로만 간다.** 뒤늦은 청크의 처리 (물음 6)
  - [x] 상한 규칙 — 총 길이로 잰다 · 어디서 자르나 (물음 3) · 표시 (물음 4) ·
        **한 번만 박힌다**. 1.2 의 `n == limit` 오판을 고치는 규칙을 문장으로
  - [x] 비우기 규칙 — 무엇이 0 으로 돌아가고 무엇이 안 돌아가나
  - [x] **총 길이가 뒤로 갈 때의 규칙** — 시도가 바뀐 경우와 트리가 쓸린 경우.
        둘째는 시도가 안 바뀌므로 읽는 쪽이 총 길이로 안다 (물음 5 의 함께 닫는 규칙)
  - [x] 수명 규칙 (물음 5) — 끊는 조건을 **잴 수 있는 문장**으로. 숫자는 비워 두고
        **「이 값은 NFR Requirements 가 채운다」를 그 자리에 적는다**
  - [x] 실패 등급 — 진행 트리 지우기는 보조다 (`application-design.md` 2.1).
        `DropProgress` 실패가 봉인을 막지 않는 규칙
  - [x] 동시성 불변식 (물음 7) — 무엇이 직렬이고 무엇이 아닌가
  - [x] 검증 규칙 (물음 9) — `record` 가 거는 것과 U4 에 맡기는 것의 경계
  - [x] 경로 불변식 — **진행 트리는 봉인 묶음에 안 들어간다.** 1.9 를 시험할 수 있는
        문장으로 (`Tar` 의 출력에 `progress/` 가 0 개다)

---

## 4. 이 단계가 안 만드는 것

```text
   코드          한 줄도 안 쓴다.  다음 단계다
   시험 목록      Code Generation 계획이 든다.  1.11 의 커버리지 예산도 거기서 센다
   N2 의 숫자     NFR Requirements.  이 단계는 규칙의 모양까지다
   라우트와 헤더   U4.  HTTP 는 이 유닛의 것이 아니다
   업로더         U7.  주기와 버퍼는 decisions.md 2절이 값으로 닫았다
   GLOSSARY.md   5절을 본다 — 이 유닛이 들여오는 새 글자가 0 이다
```

---

## 5. 파장 — 이 유닛 밖으로 가는 것

### 5.1 고쳐야 할 앞선 문서 — 셋

**전부 진행자의 몫이다.** U3 은 자기 산출물에 근거만 적는다.

```text
   application-design.md D1   근거의 줄 번호.  :196 이 0444 · 0555 를 안 가리킨다.
                              참값은 :174 와 :180 이고 「보관 정책이 없다」는
                              :189-191 이다 (1.4)

   application-design.md D4   근거 문장.  「청크마다 10 MiB」가 되는 경로가
                              질문 1 = B 뒤에 없다 (1.1).  물음 2 의 답이
                              A 면 D4 의 범위 자체가 좁아진다

   requirements.md 5.4        막는 것 표의 「진행 파일은 봉인 때 지운다」.
                              1.5 의 구멍 **둘**에서 그 문장이 거짓이 된다
                              (2.2.2 가 셋째를 반증했다).  물음 5 = A 가 이 줄을 고친다
```

**물음 10 = B · 11 = B 가 이 목록을 확정했다 (2026-09-15).**

```text
   는 것 — 물음 10 = B 가 연 둘.  **둘 다 U1 이 지고 U3 은 내보내는 쪽만 진다**

     internal/transcript 의 Kind     enode.capped 가 하나 는다.  U1 의 산출물이
                                     그 종류와 파싱 규칙을 진다
     transcript/decisions.md 2절      사건 종류 행에 enode.capped 를 더한다.
                                     팩 문서라 진행자가 올린다.
                                     enode.elided 가 같은 자리의 선례다

   안 는 것 — 물음 11 = B 가 닫은 둘

     application-design.md 8절 SECURITY-03   그대로 선다.  D7 의 「노출 기간이
                                             시도 하나」가 참으로 남는다
     requirements.md 5.4 잔여 ①               그대로.  넓혀 적을 것이 없다

   안 는 것 — 물음 10 이 A · D 였으면 열렸을 둘

     component-methods.md 3.2 의 ReadProgress 서명   **안 바뀐다.**  B 가 파일
                                                    안에 근거를 두어 인자가 안 는다
     cmd/mediator/main.go:184                       **안 바뀐다.**  파일 행렬의
                                                    「cmd/ 다섯 코드 diff 0」이 산다
```

### 5.2 파일 행렬이 넓어진다 — 물음 5 = A 가 확정했다

**사용자가 이 대가를 보고 A 를 골랐다.** 깨지는 것 둘을 문장으로 확정한다.

| | 오늘 적힌 문장 | Q5 = A 뒤의 값 | 왜 |
|---|---|---|---|
| 1 | `unit-of-work-file-matrix.md` 5절 — 「`internal/store` 코드 diff 0. `sealRecord` 가 부르는 `Seal` 이 진행 트리를 지우지만 **그 코드는 `internal/record` 에 산다**」 | **거짓이 된다.** `internal/store` 에 줄이 는다 | 끊는 자리 셋 중 둘이 Run 의 상태를 알아야 하고 **`record` 는 `store` 를 임포트하지 않는다** (1.12). 종료 상태 끊기는 `seal.go` 의 `sealRecord` 에, 고아 쓸기는 `reap.go` 에 앉는다 |
| 2 | `execution-plan.md` 6절 품질 게이트 2 — 「`internal/store` 와 `internal/contract` 의 diff 가 0 이다」 | **`internal/store` 쪽이 0 이 아니다.** `internal/contract` 는 그대로 0 | 게이트가 재는 값 자체가 바뀐다. 초록·빨강이 아니라 **기준선이 바뀌는 것**이라 게이트 문서를 고쳐야 한다 |

```text
   그대로인 것   internal/record/record.go (+ 새 파일) 는 여전히 U3 하나만 만진다.
                internal/contract 는 diff 0 그대로다.  cmd/ 다섯도 0 이다
                (물음 10 = D 를 고르면 cmd/ 가 깨진다 — 5.1 을 본다)

   고치는 사람   **진행자다.**  행렬과 실행 계획은 회차 문서라 U3 의 문서 루트 밖이다
                (CONVENTIONS.md 3.4 의 「안 싣는 것」).  U3 은 근거만 적는다

   고치는 때     이 단계의 승인 뒤.  U3 의 Code Generation 이 돌기 전이다 —
                게이트 2 가 기준선인 채로 유닛이 병합되면 그 자리에서 빨개진다
```

### 5.3 팩 문서 — 되돌려 올릴 것 0

```text
   decisions.md    행이 0 이다.  「상한 — MaxBlobBytes 와 잘림 표시 그대로」는
                   물음 2 · 3 · 4 가 그 뜻을 조이지만 값을 안 바꾼다
   scene-gates.md  CB3 의 명령이 안 바뀐다.  실측으로 확인했다 —
                   CB3 의 cmp 는 **봉인 뒤**의 GET 과 tar 의 logs/ 를 견준다.
                   그 자리는 진행 파일이 이미 지워진 뒤라 질문 1 = B 와 안 부딪친다
   canon.md        INVARIANTS 와 어긋나는 자리 0.  I4 를 안 건드린다 —
                   진행 파일은 Record 가 아니다 (ADR-025 §5)
```

### 5.4 GLOSSARY — 이 유닛이 들여오는 새 글자가 0 이다

`CLAUDE.md` 가 「그 글자를 처음 들여오는 커밋이 적는다」로 걸었다. 셌다.

```text
   U3 · CB3 · N2 · D1 · D4 · D7 · FR-5 · NC-4   전부 앞선 커밋이 들여왔다.
                                                U3 이 처음 쓰는 것이 0 이다
   AppendProgress · ReadProgress · DropProgress   Go 식별자다.  축약어가 아니다
   진행 파일 · 곁파일                              푼 말 그대로다.  줄일 계획이 없다
```

**다만 `GLOSSARY.md` 가 아직 역사에 없다.** `git cat-file -e main:GLOSSARY.md`
가 「does not exist in 'main'」이고 이 브랜치에도 없다. 초안은 진행자의 작업
트리에 **커밋 안 된 채로** 있고 (`enode-fixup-project` 의 `?? GLOSSARY.md`)
`W0` ~ `W4` · `CP` · `CA` · `handle` 넷을 적어 두었다.

**그 초안에 이 회차의 글자 둘이 아직 없다.**

```text
   CB0 ~ CB6   requirements/transcript/scene-gates.md 2절이 들여왔다.
               그 팩 커밋의 몫이다
   N1 · N2     이 회차의 Inception 이 들여왔다.
               unit-of-work.md 9절 · 10절이 그 어휘를 진다
```

**U3 이 그 줄을 적을 자리가 아니다.** 둘 다 U3 보다 앞선 커밋이 들여왔고,
`CB` 의 푼 말은 `CP` · `CA` 와 같은 자리다 — **U3 이 모른다.** `CLAUDE.md` 가
「푼 말을 모르면 짐작해 적지 말고 묻는다」로 못 박았으므로 **진행자에게 넘긴다.**

---

## 6. 브랜치를 회차에서 땄다

`unit/progress-store` 는 `v3-run-transcript` 에서 땄다 (`CONVENTIONS.md` 3.1).
`unit-of-work.md` 가 U3 의 「딛는 게이트」를 없음으로, `unit-of-work-dependency.md`
가 착수 순서를 ④ 로 적었지만 **코드로는 아무도 안 기다린다** — U3 은 바이트를
다루지 사건을 안 다룬다. `internal/record/record.go` 를 만지는 유닛이 U3 하나라
파일 행렬에 직렬이 0 이다.

기준선은 `4bc4146` 이고 그 자리에서 `go build ./...` 과
`go test ./internal/record/` 가 초록인 것을 확인하고 시작했다.
