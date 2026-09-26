# `finalize` — 규칙

노드가 명령이 끝난 뒤 무엇을 거두고, 두 예산을 어떻게 지키고, 종료 보고를 어떻게 보내고, 결과의
새 칸을 어떻게 채우나의 규칙이다. 로그 줄과 error 칸의 문구는 영어다 (`CONVENTIONS.md` 2.1 — 밖으로
나간다). 타입과 이름은 `domain-entities.md` 에 있다.

**원칙 셋**

- **결과 확정은 판정이 아니다** (ADR-004 · I3). 노드는 사실을 적고, 성패는 Mediator 의
  `success_when` 이 정한다
- **예산을 넘긴 것만 판정을 바꾼다** (FR-3). 예산 초과는 error 칸을 채워 단계를 실패시킨다. 업로드 한
  건의 전송 실패 · 훑기의 상한 · collect 실패는 사실로만 적고 판정을 바꾸지 않는다 — 오늘과 같다
- **종료 보고는 Finalize 를 막지 않는다** (ADR-075 §10.4). 보고가 유실돼도 result 가 같은 사실을 싣는다

---

## 1. effect 가 정하는 수확 (contract-grammar 의 표 · 답 4 = A)

effect 는 `contract.Step.EffectOrDefault` 로 채운 값이다.

| 단계 · effect | diff (`workspace.diff`) | 명시 훑기 | collect · 지목 경로 stat · `$OUT` |
|---|---|---|---|
| run · build (기본) | 없음 | `discover` 일 때만 | 있다 |
| run · read | 없음 | `discover` 일 때만 | 있다 |
| run · edit | 오늘 조건 그대로 | `discover` 일 때만 | 있다 |
| agent · edit (기본) | 오늘 조건 그대로 | `discover` 일 때만 | 있다 |
| agent · read | 없음 | `discover` 일 때만 | 있다 |

- **오늘 조건** — 노드에 워크스페이스가 있고(`Local.Workspace`) 계약이 `workspace` 를 적었다
  (`claim.go:692`). agent 의 diff 는 Git changeset adapter 가 생길 때까지 그대로 둔다 (Units Generation 물음 7)
- **전체 훑기는 어느 단계에서도 늘 켜지 않는다.** 켜는 것은 계약의 `discover: true` 하나다
- **지목 경로 stat 은 늘 한다** (ADR-037). 판정 재료이고 비용이 계약이 적은 경로 수를 따른다
- **agent 가 완주하지 못했을 때** (하네스가 ok · cannot 이 아닌 이유로 끝났다) — collect 와 diff 는 안
  한다 (오늘 그대로 · 반쯤 쓴 파일을 믿을 수 없다). 지목 경로 stat 은 한다. `discover` 를 켰으면 훑기도
  한다 — 실패한 sandbox 를 조사하는 진단이다 (ADR-075 §8)

---

## 2. 두 예산의 경계 (답 1 = A)

```text
   종료 status 확정 = exited_at
        |
        +-- [ Finalize 예산 ]  session.Finalize — collect · 지목 경로 stat · diff · 명시 훑기
        |
        +-- 닫기               session.Close.  이 유닛에서는 예산 밖 (trash 유닛이 안으로 옮긴다)
        |
        +-- finalized_at
        |
        +-- [ 업로드 예산 ]    단계 로그 -> $OUT 의 이름들
        |
        +-- result 보고        두 예산 밖.  임대가 죽을 때까지 재시도 (오늘 그대로)
```

- 예산 값은 `contract.Step.Budgets()` 다 — 계약이 안 적었으면 Finalize 1분 · 업로드 3분
- **Finalize 예산의 ctx** 는 `context.WithDeadline(runCtx, exited_at + finalize)` 다. runCtx 는 임대를
  감시한다 (`claim.go:604`). 그래서 임대가 끝나도 Finalize 가 멈춘다 (ADR-075 §9 — 모든 연산은 임대를
  아는 ctx 를 받는다)
- **업로드 예산의 ctx** 는 닫기가 끝난 때부터 `context.WithTimeout(runCtx, upload)` 다
- **닫기의 ctx** 는 Worker 의 ctx 다 (마감 없음). 닫기가 runRoot 를 지우는 동안은 upper 크기에 비례하는
  시간이 걸린다. 그 시간은 `finalized_at` 에 그대로 적힌다
- **임대가 이미 끝났으면** (명령이 임대 만료로 죽었다) Finalize 와 업로드를 건너뛴다. 닫기와 result
  보고만 한다 — 오늘의 "aborted: lease expired" 그대로. Run 은 이미 회수되고 있다
- **프로세스가 뜨지 않았으면** (ADR-075 §10.2 ①) 종료 보고도 Finalize 도 없다. 닫기 · 단계 로그 업로드 ·
  result 보고만 한다. result 의 error 는 오늘의 문구다

---

## 3. 예산을 넘겼을 때 (답 2 = A)

### 3.1 무엇이 멈추고 무엇이 계속되나

| 자리 | 넘으면 |
|---|---|
| native 의 Finalize | ctx 가 collect (파일 사이와 복사 중) · git diff · 훑기 · stat 을 멈춘다. collect 는 `.part` 에 쓰고 rename 하므로 반쯤 복사한 것은 그 이름으로 남지 않는다 |
| overlay 의 Finalize | helper 가 요청의 `Deadline` 으로 자기 ctx 를 만들어 같은 자리에서 멈춘다. Worker 는 마감 + **5초**를 기다리고, 답이 없으면 helper 를 죽인다 (오늘의 abort — 프로세스 그룹에 SIGKILL) |
| 닫기 | 계속한다 |
| 업로드 (Finalize 초과 뒤) | 계속한다. 단계 로그와 `$OUT` 의 이름을 올린다. 이름이 `.part` 로 끝나는 파일은 안 올린다 — 죽은 helper 가 남긴 반쪽이다 |
| 업로드 (업로드 초과) | 진행 중 요청을 끊고 남은 이름을 안 올린다. 못 올린 이름은 produced 에 없다 |
| result 보고 | 계속한다. 임대가 죽을 때까지 재시도한다 |

Finalize 가 끝내 결과를 못 돌려줬으면 (helper 를 죽였다) 지목 경로 stat 의 결과(`changed`)는 비고,
진단 칸의 `missing` 은 Worker 가 `$OUT` 을 직접 읽어 채운다. `collect` 와 `discovered` 는 비고 `changes` 는
`not_measured` 다.

### 3.2 `finalize` · `upload` · `reason` · `error` 를 정하는 표

순수 함수 하나가 정한다 (`business-logic-model.md` 9절).

```text
   finalize   timeout   Finalize ctx 의 마감이 지났고 runCtx 는 살아 있다
                        (overlay 에서 helper 를 죽인 경우도)
              error     Finalize 가 마감이 아닌 오류를 돌려줬거나 닫기가 오류를 돌려줬다
              ok        그 밖
   upload     timeout   업로드 ctx 의 마감이 지났고 runCtx 는 살아 있다
              error     예산 안에서 전송 실패가 한 건이라도 있었다.  Mediator 의 거절(4xx)은 아니다
              ok        그 밖
   reason     finalize_timeout   finalize 가 timeout
              upload_timeout     그 밖에 upload 가 timeout
              ""                 그 밖
```

**error 칸의 문구** — 일어난 순서대로 `; ` 로 잇는다.

| 경우 | 문구 |
|---|---|
| Finalize 초과 | `finalize budget of %s exceeded` (값은 Go duration — `1m0s`) |
| Finalize 오류 | `runtime finalize: %v` (오늘의 `runtime harvest: %v` 를 바꾼다) |
| 닫기 오류 | `runtime cleanup: %v` (오늘 그대로) |
| 업로드 초과 | `upload budget of %s exceeded` |

임대 만료(`aborted: lease expired`)와 실행 실패(`code < 0` 인 runErr — 프로세스가 안 떴거나 native 에서
signal 로 죽었다)는 오늘처럼 위 문구를 **덮어쓴다** — 명령이 완주하지 않았다는 사실을 먼저 적는다. signal 로
죽은 명령은 오늘처럼 완주가 아니다. 종료 보고만 그것을 signal 로 알린다.

error 칸이 차면 Mediator 는 완주가 아닌 것으로 받는다 (`api.go:432`). 명령의 `exit_code` 는 result 에
그대로 남아 명령 실패와 나뉜다 (조각 3 — 예산).

---

## 4. 종료 보고 (답 3 = A)

### 4.1 보내는 조건 — `exitOutcome(code, runErr)`

| session.Run 이 돌려준 것 | outcome | 보내나 |
|---|---|---|
| `code >= 0` | `{kind: exit, code}` | 보낸다 |
| `code < 0` 이고 `runErr` 가 `*exec.ExitError` 이며 signal 로 끝났다 (native) | `{kind: signal, code: 신호 번호}` | 보낸다 |
| 그 밖의 `code < 0` — 프로세스가 안 떴거나 overlay 가 이유를 문자열로만 준다 | 없음 | 안 보낸다 |

- overlay 는 container 안의 프로세스가 signal 로 죽으면 runc 가 128+n 을 종료 코드로 준다 — 첫 줄에
  걸려 `exit` 로 간다. 다시 해석하지 않는다
- **임대가 끝나 죽인 명령에는 안 보낸다** — `session.Run` 이 돌아온 순간 `runCtx.Err()` 가 있으면 건너뛴다
- `timeout` 종류는 이 유닛이 안 쓴다 — 오늘 명령 timeout 이 없다
- **agent 단계**는 하네스 프로세스가 끝난 순간에 같은 조건으로 보낸다 (`Job.Exited` 콜백 — 봉투 해석과
  버전 확인 앞). 하네스가 완주하지 못했어도 프로세스는 끝났으므로 보낸다
- **보내는 것** — `{node: Ident.NodeID, instance: Client.Instance, attempt: Step.Attempt, outcome,
  exited_at}`. `exited_at` 은 `session.Run` 이 돌아온 순간의 노드 시계이고, result 의 `exited_at` 과 같은 값이다

### 4.2 응답과 재전송

| 응답 | 한다 | 노드 로그 |
|---|---|---|
| 200 (`accepted` 가 true 든 false 든) | 멈춘다 | debug `exit reported` |
| 409 | 멈춘다 | warn `exit report rejected; not retrying` 와 본문 |
| 400 | 멈춘다 | error `exit report was malformed; not retrying` 와 본문 — 노드의 결함이다 |
| 404 | 멈춘다 | info `the mediator does not accept exit reports for this step; not retrying` — 옛 Mediator 도 여기다 |
| 그 밖의 4xx | 멈춘다 | warn `exit report rejected; not retrying` |
| 5xx · 끊김 · 요청 제한 30초 | 다시 보낸다 | debug `exit report failed; retrying` |

- 다시 보내는 간격 — 1초에서 두 배씩, 최대 10초 (1 · 2 · 4 · 8 · 10 · 10 …)
- **그만두는 때** — result 보고를 시작하기 직전 (`exitReporter.Stop`). result 가 같은 사실을 싣는다.
  Worker 의 ctx 가 끝나도 그만둔다
- 멈춤은 **그 단계에만** 걸린다. 단계마다 한 번 보내므로 Run 단위로 기억하지 않는다 — 옛 Mediator 에서
  단계마다 요청 하나를 더 쓰는 비용이다

---

## 5. 명시 훑기 (답 5 = A)

### 5.1 언제 · 어디를

- 계약이 `discover: true` 를 적었을 때만. effect 와 무관하고, agent 가 완주하지 못했어도 돈다 (1절)
- **native** — 워크스페이스(`Stamp.Root` = `Local.Workspace`)를 걷는다. 기준 시각(`Stamp.At`) 뒤에 수정된
  보통 파일을 모은다 (오늘 방법)
- **overlay** — `runRoot/upper` 만 걷는다. upper 에 있는 것이 곧 이 단계가 쓴 것이다. 보통 파일을 모으고,
  whiteout (지운 항목)은 목록에 안 넣고 `Deleted` 로 센다. opaque 표시와 디렉터리는 안 센다
- 둘 다 `.git` 과 `.repo` 디렉터리 아래로 안 들어간다 (오늘 규칙)
- 워크스페이스가 없는 노드(native 에서 `Local.Workspace` 가 비었다)면 훑지 않고 `Skipped` 에 이유를 적는다

### 5.2 상한 넷

| 상한 | 값 | 닿으면 |
|---|---|---|
| 방문 | 2,000,000 항목 (디렉터리 포함) | 걷기를 멈춘다. `visits` |
| 시간 | Finalize 예산의 절반 (기본 30초). 훑기를 시작한 때부터 | 걷기를 멈춘다. `time` |
| 메모리 | 들고 있는 항목 200,000 | 걷기를 멈춘다. `memory` |
| 결과 크기 | 경로 2,000 개 · 경로 글자 합 256 KiB | 걷기를 다 한 뒤 큰 파일부터 담다가 자른다. `size` |

- **먼저 닿은 하나**만 `discovery_limit` 에 적는다. 결과 크기는 걷기를 다 한 뒤에만 볼 수 있으므로 다른
  상한이 먼저 닿았으면 적히지 않는다
- 시간 상한이 예산의 절반이라 훑기 혼자서 단계를 `finalize_timeout` 으로 만들지 않는다. 계약이
  `budget.finalize` 를 늘리면 훑기 시간도 같이 는다 — 계약은 `discover` 를 켜기만 한다는 contract-grammar 의
  규칙을 지킨다
- 순서는 오늘처럼 큰 파일부터, 같은 크기면 경로 순이다

### 5.3 `changes` 의 값

| 경우 | `changes` |
|---|---|
| `discover` 를 안 켰다 | `not_measured` |
| 켰는데 워크스페이스가 없다 · Finalize 가 결과를 못 돌려줬다 | `not_measured` |
| 켰고 상한에 닿았다 | `partial` |
| 켰고 다 훑었다 | `measured` |

`not_measured` 는 「바뀐 파일이 없다」가 아니다 (FR-1). 칸이 늘 있다 — `omitempty` 가 아니다
(`contract.Diagnostics`).

---

## 6. 진단 칸과 단계 로그 끝 (답 6 = A)

### 6.1 `diagnostics` 칸 — Finalize 를 돈 단계마다 하나

| 칸 | 채우는 법 |
|---|---|
| `missing` | 계약의 `out` 에 있는데 Finalize 뒤 `$OUT` 에 없는 이름. `.part` 는 없는 것으로 친다 |
| `collect` | collect 가 못 걷은 이름과 이유 — `HarvestNote` 를 `CollectNote` 로 옮긴다 |
| `changes` | 5.3 |
| `effect` | 기본값을 채운 effect |
| `discovered` | 명시 훑기의 목록 (5.2 의 결과 크기 안쪽) |
| `discovery_limit` | 닿은 상한. 안 닿았으면 없다 |

### 6.2 `workspace.changed` 를 없앤다

- `$OUT` 에 `workspace.changed` 를 더 쓰지 않는다. produced 에 그 이름이 더 안 나온다
- 「no files changed in the workspace」 문장이 사라진다 — 안 확인한 것을 「안 바뀌었다」로 말하지 않는다
- `writeHarvestNote` · `writeChangedNote` · `changedName` 을 지운다. 훅의 걷기(`hook.go:283`)와
  `summarize` 는 그대로 둔다 — 하네스가 도는 동안의 것이고 Finalize 와 무관하다
- `workspace.diff` 는 1절의 표에서 diff 가 있는 단계에 오늘처럼 `$OUT` 의 산출물로 나온다

### 6.3 단계 로그 끝의 줄

단계 로그(Record 의 `logs/` 에 들어가는 선별본)의 끝에 붙인다. 줄마다 `enode: ` 로 시작한다. 같은 내용을
노드 로그(slog)에도 남긴다 (오늘 그대로). 순서는 아래 표의 순서다.

| 경우 | 줄 |
|---|---|
| 빠진 산출물 | `enode: required by the contract but missing from $OUT: a, b` |
| collect 실패 (이름마다) | `enode: collect could not gather a: <why>` |
| diff 실패 | `enode: workspace.diff was not produced: <err>` |
| 훑기 — 다 훑었다 | `enode: discover listed N files created or modified by this step` |
| 훑기 — 상한에 닿았다 | `enode: discover listed N files created or modified by this step before it stopped at the <limit> limit; the list is partial` |
| 훑기 — overlay 의 지운 항목 | 위 줄 끝에 `, and M deletions` |
| 훑기 — 목록 | 큰 것부터 20 줄. `enode:   <path>  <size>` (크기는 오늘의 `human` 표기) |
| 훑기 — 워크스페이스 없음 | `enode: discover was requested but this node has no workspace directory; nothing was listed` |
| Finalize 초과 | `enode: finalize budget of 1m0s exceeded` |
| 바뀐 기본값 안내 | 7절 |

- **업로드의 결과는 로그 끝에 없다** — 단계 로그를 먼저 올리므로(2절) 그 뒤의 일은 담을 수 없다. 업로드
  결과는 result 의 `upload` · `reason` · `error` 칸과 노드 로그에 있다
- agent 단계도 같다. 단계 로그 업로드를 수확 뒤로 옮긴다 (`claim.go:874` 의 자리)

---

## 7. 바뀐 기본값의 안내 (완료 조건 3 ④ ⑤ · 답 7 = A)

단계 로그 끝의 마지막 줄이다. 기한을 두지 않는다.

| 조건 | 줄 |
|---|---|
| run 단계가 `effect` 를 안 적었고 `discover` 도 안 켰다 | `enode: this run step used the default effect build, so it returns no workspace.diff and no list of changed files. Write "effect": "edit" if the command changes source files, or "discover": true to list what changed.` |
| agent 단계가 `discover` 를 안 켰다 | `enode: agent steps no longer return workspace.changed. Write "discover": true to list what changed.` |

`effect` 를 적은 run 단계와 `discover` 를 켠 단계에는 안 쓴다 — 적은 사람은 이미 안다.

---

## 8. 업로드 (답 8 = A)

- **client** — `Client.Upload` (요청마다의 제한 없음). 마감은 업로드 예산의 ctx 다. `PutBlob` 과
  `UploadLog` 가 쓴다. 진행 청크 · 종료 보고 · result · claim · `GetBlob` 은 30초 client 그대로다
- **순서** — 단계 로그 먼저, 그다음 `$OUT` 의 이름들 (디렉터리를 읽은 순서 — 이름순). `.enode-` 로
  시작하는 이름(오늘)과 `.part` 로 끝나는 이름(새로)은 건너뛴다
- **흘려 보내기** — `PutBlob` 은 파일을 열어 크기를 `Content-Length` 로 싣고 흘려 보낸다. 통째로 읽지 않는다
- **크기 사전 검사 없음** — Mediator 가 상한+1 바이트에서 413 으로 끊는다 (`api.go:1056` 이후). 보내는 양이
  상한을 안 넘는다. 노드는 Mediator 의 상한 값을 모른다
- **재시도 없음** (오늘 그대로)
- **이름마다의 결과**

| 결과 | produced 에 | `upload` 칸 | 노드 로그 |
|---|---|---|---|
| 2xx | 넣는다 | 영향 없음 | - |
| 4xx (`BlobRejected` — 413 · 422 등) | 안 넣는다 | 영향 없음 — Mediator 의 판정이다 | warn `blob rejected; not listed in produced` (오늘 그대로) |
| 업로드 ctx 의 마감 | 안 넣는다. 남은 이름도 안 올린다 | timeout | warn `upload budget exceeded; not uploaded` 와 이름들 |
| 그 밖의 전송 실패 | 안 넣는다. 다음 이름으로 간다 | error | warn `blob upload failed` |

단계 로그의 업로드도 같은 표를 따른다. 로그는 produced 가 아니므로 첫 열이 없다.

---

## 9. result 의 새 칸 — 경로마다

| 경로 | 종료 보고 | `exited_at` | Finalize | `finalized_at` | `finalize` | `upload` | `reason` | `diagnostics` | error |
|---|---|---|---|---|---|---|---|---|---|
| 명령 앞 실패 (워크스페이스 · `$IN` · runtime open · 빈 argv) | 없음 | - | 없음 | - | - | - | - | - | 오늘 문구 |
| 프로세스가 안 떴다 | 없음 | - | 없음 | - | - | - | - | - | 오늘 문구 (runErr) |
| 임대가 끝나 죽였다 | 없음 | - | 없음 | - | - | - | - | - | `aborted: lease expired` |
| native 에서 signal 로 죽었다 | 보낸다 (signal) | 있다 | 돈다 | 있다 | ok · error | ok · error | - | 있다 | 오늘 문구 (runErr — `signal: killed` 등) |
| 정상 | 보낸다 | 있다 | 돈다 | 있다 | ok · error | ok · error | - | 있다 | 없음 또는 런타임 오류 |
| Finalize 초과 | 보낸다 | 있다 | 멈춘다 | 있다 | timeout | ok · error · timeout | `finalize_timeout` | 있다 | 3.2 |
| 업로드 초과 | 보낸다 | 있다 | 돈다 | 있다 | ok · error | timeout | `upload_timeout` | 있다 | 3.2 |
| agent 미완주 | 보낸다 | 있다 | 돈다 (stat · 훑기) | 있다 | ok · error | ok · error | - | 있다 | `harness: …` (오늘 문구) |

`-` 는 칸이 없다는 뜻이다. 명령 앞 실패 · 안 뜬 프로세스 · 임대 만료의 단계 로그 업로드도 업로드 client 와
업로드 예산으로 하지만, 새 칸은 채우지 않는다 — 여섯은 Finalize 를 돈 단계에만 있다 (`domain-entities.md` 4절).

---

## 10. 확장 준수 — Functional Design

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표에서 이 유닛에 닿는 줄은 없다. 노드는 종료 보고에 자기 instance 를 싣기만 한다 — 대조는 step-phase 가 Mediator 에서 한다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 한 줄 |
