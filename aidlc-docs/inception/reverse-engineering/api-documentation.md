# API 문서 — 오늘 있는 표면

이 문서는 **지금 코드에 있는 것** 만 적는다. 팩이 더할 표면(`GET /v1/nodes`,
`GET /v1/runs` 목록, `GET /v1/runs/{id}` 의 `requires`, `202`/`QUEUED`)은 오늘
없으며, 없다는 사실을 각 절 끝에서 명시한다.

---

## REST APIs

Mediator 의 HTTP 표면 전체는 `internal/api` 한 패키지다. Go 1.22+ 의
`http.ServeMux` 위에 라우트 **15개** 가 `internal/api/api.go:61-75` 에서 연속으로
등록된다. 라우터 의존성은 없다.

전부 `s.auth(...)` 로 감싸여 있다 (`api.go:108-123`). 인증은 `Authorization:
Bearer <token>` 하나이며, 접두사 확인 뒤 `subtle.ConstantTimeCompare` 로 토큰을
상수 시간 비교한다 (`api.go:115-116`). 토큰이 비었거나 접두사가 없거나 틀리면
`401`. `X-Enode-Principal: <email>` 은 식별이지 인증이 아니다 — 검증하지 않고
컨텍스트에 그대로 담아(`api.go:120`) 취소 기록 · 답변자 판정에만 쓴다
(`api.go:100`, ADR-015 §1).

실패 응답 본문은 `errBody` 로 통일된다 (`api.go:132-137`):

```json
{ "error": { "code": 0, "reason": "" } }
```

`fail(w, code, reason)` 이 `code` 와 `reason` 을 둘 다 채운다 (`api.go:139-143`).
성공 JSON 은 `write(w, code, v)` 가 낸다 (`api.go:145-149`).

### 등록된 라우트 15개 (api.go:61-75)

| # | Method | Path | Handler |
|---|--------|------|---------|
| 1 | POST | `/v1/nodes` | `postNodes` |
| 2 | POST | `/v1/nodes/{id}/claim` | `postClaim` |
| 3 | POST | `/v1/runs/{run}/steps/{seq}/result` | `postResult` |
| 4 | POST | `/v1/runs` | `postRuns` |
| 5 | POST | `/v1/runs/dry-run` | `postDryRun` |
| 6 | GET | `/v1/runs/{id}` | `getRun` |
| 7 | GET | `/v1/capabilities` | `getCapabilities` |
| 8 | GET | `/v1/asks` | `getAsks` |
| 9 | POST | `/v1/runs/{run}/steps/{seq}/answer` | `postAnswer` |
| 10 | GET | `/v1/runs/{id}/ledger` | `getLedger` |
| 11 | GET | `/v1/runs/{id}/record` | `getRecord` |
| 12 | POST | `/v1/runs/{id}/cancel` | `postCancel` |
| 13 | PUT | `/v1/runs/{run}/steps/{seq}/log` | `putLog` |
| 14 | PUT | `/v1/runs/{run}/steps/{seq}/blob/{name}` | `putBlob` |
| 15 | GET | `/v1/runs/{run}/blob/{name}` | `getBlob` |

9번의 경로 문자열은 `answerRoute` 상수(`api.go:84`)에서 나온다. 같은 상수를
`answerPath` 가 되묻기 알림의 직링크에도 쓴다 (`api.go:92-95`) — 출처가 하나다.

### enode -> Mediator (노드가 부른다)

**1. POST /v1/nodes** — `postNodes` (api.go:194-243)
- 목적: 광고 + 하트비트. 응답이 임대의 갱신이자 취소 통보다 (ADR-016).
- 요청: JSON `contract.Advert` (`node_id`, `label`, `instance`, `capabilities[]`).
  본문 파싱 실패면 `400`, `node_id` 가 비면 `400`.
- 응답: `200` `advertResponse` (`api.go:183-192`, 242):
  `{ "leases": [LeaseRow], "renew_seconds": int }`.
  `LeaseRow` 는 `{run_id, node, capability, not_after, nonce}` (`claim.go:20-26`);
  `capability` 는 항상 `"agent.reason"` 로 채워진다 (`claim.go:58`, ADR-019).
  저장 · 갱신 실패면 `503`. 목록은 델타가 아니라 전부이므로, 목록에 없는 임대가
  곧 취소된 임대다. `instance` 가 있으면 재시작 판정(`FailRestarted`)이 갱신보다
  먼저 돈다 (`api.go:217-231`, ADR-030).

**2. POST /v1/nodes/{id}/claim** — `postClaim` (api.go:253-283)
- 목적: 유일한 비멱등 지점. 롱폴 pull.
- 요청: 본문 없음. path `{id}` 가 `node_id`. 헤더 `X-Enode-Instance` (ADR-030,
  없어도 옛 enode 로 동작).
- 응답: 할 일이 있으면 `200` `store.Claimed` (`claim.go:65-`):
  `{step_id, run_id, seq, name, uses, kind, agent?, run?, env?, collect?,
  check_changed?, workspace?, in?, out?, schema?, roles?, ...}`.
  `Claim.LongPollSeconds` 데드라인까지 1초 ticker 로 다시 보고 할 일이 없으면
  `204`. `ErrNoWork` 외의 오류면 `503`. `main.go` 는 이 롱폴 때문에 의도적으로
  `WriteTimeout` 을 걸지 않는다.

**3. POST /v1/runs/{run}/steps/{seq}/result** — `postResult` (api.go:294-341)
- 목적: 단계 결과 보고. 이 보고를 받은 Mediator 가 다음 단계를 만든다
  (`RUNNING -> RUNNING`, 주체는 Mediator, ADR-014).
- 요청: JSON `{ "node": string, ...store.StepResult, "produced": [string],
  "error": string }`. `seq` 가 양의 정수가 아니면 `400`, 본문 파싱 실패면 `400`.
  성패는 여기서 판정하지 않는다 — `error == ""` 이면 완주로 본다(`api.go:317`).
- 응답: `ReportStep` 이 되돌림(rolled)이면 `200`
  `{run_id, seq, rolled_back: true}`; 아니면 `SettleIfDone` 뒤 `200`
  `{run_id, seq, run_state}`. `ReportStep` 오류는 `409`, settle 실패는 `503`.

**13. PUT /v1/runs/{run}/steps/{seq}/log** — `putLog` (api.go:645-679)
- 목적: 그 단계가 뱉은 것을 원문 로그로 남긴다. result 보다 먼저 올린다.
- 요청: 본문이 로그 바이트. query `?name=` (없으면 `"step"`). `AppendLog` 는
  `MaxBlobBytes` 에서 자르고 표시한다(로그는 잘려도 로그다).
- 응답: `204`. Run 이 `SUCCEEDED`/`FAILED`(봉인)면 `410` (I4). Run 없으면 `404`,
  Record 저장소 없으면 `503`.

**14. PUT /v1/runs/{run}/steps/{seq}/blob/{name}** — `putBlob` (api.go:695-779)
- 목적: 단계 사이를 오가는 산출물 업로드. 스키마 검증 지점이다 (ADR-020).
- 요청: 본문이 blob 바이트. 회차(`attempt`)는 클라이언트가 아니라 Mediator 가
  `StepAttempt` 로 안다(`api.go:722`). `schemaFor` 로 그 단계 · 그 이름에 스키마가
  달렸으면(`api.go:801-806`) 전부 버퍼링해 `schema.Validate` 로 검사한다.
- 응답: `204`. 스키마 위반이면 `422`(저장 안 함 -> produced 불만족 -> 단계 실패).
  스키마 있는 경로에서 크기 초과면 `413`, 스키마 없는 스트리밍 경로에서 초과면
  `record.ErrTooBig` -> `413`(자르지 않는다). Run 봉인이면 `410`, Run 없으면
  `404`, step 없으면 `404`.

**15. GET /v1/runs/{run}/blob/{name}** — `getBlob` (api.go:781-798)
- 목적: 이름으로 최신 blob 을 내려준다(소비자는 이름만 안다). ADR-018 로 나중에
  `302` 리다이렉트가 되지만 오늘은 직접 서빙한다.
- 요청: 본문 없음. path `{run}`, `{name}`.
- 응답: `200` `application/octet-stream` + `Content-Length`, 본문이 blob.
  없으면 `404`, Record 저장소 없으면 `503`. 헤더를 보낸 뒤의 복사 오류는 로그만
  남기고 클라이언트에는 안 드러난다.

### runctl · 사람 -> Mediator

**4. POST /v1/runs** — `postRuns` -> `submit(dry=false)` (api.go:345, 352-477)
- 목적: Run 제출. 매칭 -> 임대 발급 -> `RUNNING` 생성.
- 요청: JSON `contract.Contract`. 파싱 실패나 `Validate()` 실패면 `400`.
- 응답 분기:
  - 같은 `run_id` 재제출이면 `200` + 기존 `runView` (`api.go:372-373`, 멱등).
  - 새로 만들면 `201` `runView` (`api.go:474-476`).
  - 매처 거절이면 `rej.Code`(`409` `CodeAllBusy` 또는 `422` `CodeNoCandidate`)로
    실패하고, `CreateRejectedRun` 으로 `state=FAILED` 인 runs 행을 남긴다
    (`api.go:399-408`).
  - 폭 상한(`MaxPerRun`) 초과면 `422` `CodeNoCandidate` + 거절 기록 (`api.go:413-432`).
  - 배정 도중 다른 Run 이 노드를 가져가 `ErrNodeTaken` 이면 `409` "another run
    took the node during allocation" — 트랜잭션이 전부 롤백하므로 **runs 행이
    아예 남지 않는다** (`api.go:459-463`).
- `runView` 형태 (`api.go:151-164`):
  `{run_id, state, assigned?, reject?, verdict?, steps?, warnings?}`.
  `steps` 는 `store.StepView` 배열(실행 중 관측, ADR-025).
- **오늘 없는 것**: 후보는 있으나 전부 점유된 일시 상황에 `202`/`QUEUED` 를
  내지 않는다 — 그 경우는 위의 `409`(`ErrNodeTaken` 또는 `CodeAllBusy`)로 끝난다.
  `WakeQueued` 심볼은 코드에 없다. 팩이 ADR-064 로 더한다.

**5. POST /v1/runs/dry-run** — `postDryRun` -> `submit(dry=true)` (api.go:350)
- 목적: 본문도 매처도 `POST /v1/runs` 와 같고 점유만 안 본다. 존재는 답하고
  여유는 안 답한다.
- 요청: 같은 `contract.Contract`.
- 응답: `200` `runView{state: "DRY_RUN", assigned, warnings}` (`api.go:433-436`).
  점유를 보지 않으므로(`busy` 는 빈 맵) `409` 가 나오지 않고, 후보가 없으면
  `422` 다. `DRY_RUN` 은 와이어 전용 값이며 DB 에 저장되지 않는다.

**6. GET /v1/runs/{id}** — `getRun` (api.go:481-493)
- 목적: Run 진행 관측(Record 가 아니다, ADR-025).
- 요청: path `{id}` 가 `run_id`, 본문 없음.
- 응답: `200` `runView`(`state` + `steps[]`). 없으면 `404`, 조회 실패면 `503`.
- **오늘 없는 것**: `requires` 필드는 나오지 않는다 (ADR-069 미구현) — 대시보드가
  "왜 기다리나" 를 답하려면 팩이 더해야 한다.

**8. GET /v1/asks** — `getAsks` (api.go:500-518)
- 목적: 되묻기 인박스. 대기 중인 것만 든다(답한 것은 봉인에 있다). 폴링 인박스가
  정본이다 (ADR-032).
- 요청: 본문 없음. `X-Enode-Principal` 로 `can_answer` 관점을 채운다.
- 응답: `200` `{ "asks": [AskView] }`. `AskView` (`ask.go:134-154`):
  `{run_id, seq, step, prompt, schema, answerers?, asked_at?, deadline?,
  can_answer, shown?, proposes?}`. `can_answer` 는 저장값이 아니라 보는 사람
  기준으로 서버가 채운다(`api.go:508-516`).

**9. POST /v1/runs/{run}/steps/{seq}/answer** — `postAnswer` (api.go:524-572)
- 목적: 답은 주소 있는 단일 쓰기다 — 본문이 곧 답이고 산출물이 된다 (ADR-032).
- 요청: 본문(임의 바이트, blob 처럼 검증). `seq` 가 양의 정수가 아니면 `400`.
  `MaxBlobBytes` 초과면 `413`.
- 응답: 되돌림이면 `200` `{run_id, seq, rolled_back: true}`, 아니면
  `SettleIfDone` 뒤 `200` `{run_id, seq, run_state}`. `ErrNoAsk` -> `409`,
  `ErrNotAnswerer` -> `403`, `SchemaViolation` -> `422`(저장 안 함, 질문은 열린 채
  남는다), Record 저장소 없으면 `503`.

**10. GET /v1/runs/{id}/ledger** — `getLedger` (api.go:585-597)
- 목적: 그 Run 이 지금 발견할 수 있는 것의 목록. 본문이 아니다 (ADR-023 §6.3).
  종료 전에도 답한다(Record 가 아니므로 `409` 를 우회하지 않는다).
- 응답: `200` `{ "entries": [LedgerEntry] }`. `LedgerEntry` (`observe.go:70-85`):
  `{run_id?, seq, attempt, name, by, at, bytes, schema_ok?}`. 없으면 `404`.

**11. GET /v1/runs/{id}/record** — `getRecord` (api.go:814-832)
- 목적: 봉인된 Run Record 를 tar 로 돌려준다(성질 4, 자기충족).
- 응답: 봉인됐으면 `200` `application/x-tar` +
  `Content-Disposition: attachment; filename="run-<id>.tar"`.
  아직 안 봉인됐으면 `409` "run is not sealed yet" (`s.records.Sealed` 이
  `verdict.json` 의 쓰기 비트가 꺼졌는지로 판정). Run 없으면 `404`, Record 저장소
  없으면 `503`. `CreateRejectedRun` 으로 만든 `FAILED` 거절 Run 은 Record 를
  봉인하지 않으므로 여기서 `409` 가 난다.

**12. POST /v1/runs/{id}/cancel** — `postCancel` (api.go:622-636)
- 목적: 사람이 Run 을 세운다 (ADR-009). 멱등이며 이미 종료면 그 상태로 `200`.
- 요청: 본문 없음. `X-Enode-Principal` 은 누가 취소했는지 기록에만 쓴다.
- 응답: `200` `{run_id, state}`. 없으면 `404`, 실패면 `503`.

**7. GET /v1/capabilities** — `getCapabilities` (api.go:607-615)
- 목적: 함대의 속성 어휘. 우리 층의 tools/list 다 (ADR-012 대칭).
- 응답: `200` `{ "capabilities": [CapabilityView] }`. `CapabilityView`
  (`store.go:384-388`): `{capability, nodes, attrs: map[string][]string}`.
  `nodes` 는 총수이지 지금 비어 있는 수가 아니다(여유는 답하지 않는다). `attrs`
  값 목록은 함대 전체의 합집합이다.

### 오늘 없는 라우트 (팩이 추가할 것)

- **GET /v1/nodes** — 등록이 없다. `/v1/nodes` 는 `POST` 뿐이다(`api.go:61`).
  ADR-065 의 운영 관측 스냅샷(`draining` 포함)은 미구현이며, 구현은 아직
  병합되지 않은 enode 브랜치에 있다.
- **GET /v1/runs (목록)** — 없다. `runs` 에는 id 스코프 GET 만 있다
  (`/v1/runs/{id}`, `.../ledger`, `.../record`, `.../blob/{name}`;
  `api.go:66,70,71,75`). canon 이 그린 `?state=&since=&work=&limit=` 목록은
  미구현이다.
- **GET /v1/runs/{id} 의 `requires` 필드** — 없다 (ADR-069 미구현).
- **`202` / `QUEUED`** — 없다(위 4번 참고).
- **`/ui/` 정적 파일 서빙** — 없다. `cmd/mediator` 는 `api.New(...).Handler()`
  하나만 마운트한다(`FileServer`/`embed` 없음).

---

## Internal APIs

### internal/runctl.Client (client.go)

무상태 HTTP 클라이언트다 — 제출하고 잊는다(`client.go:1-5`).
`Client{Base, Token, Principal, HTTP}` (`client.go:19-24`). 모든 호출은
`do(ctx, method, path, body)` 를 지난다(`client.go:52-86`): 요청에
`Authorization: Bearer <token>`(`client.go:64`)와 `X-Enode-Principal`
(`client.go:65`)를 붙이고, 응답이 `2xx` 가 아니면 본문의 `error` 를 읽어
`*Fail{Code, Reason}` 을 돌려준다(`client.go:85`). CLI 종료코드로의 번역은
`cmd/runctl` 이 한다.

과제가 지목한 다섯 메서드:

| 메서드 | 시그니처 | 부르는 라우트 |
|--------|----------|---------------|
| `Submit` | `Submit(ctx, contract []byte, dry bool) (*Run, error)` (client.go:118) | `POST /v1/runs` 또는 `/v1/runs/dry-run` |
| `Status` | `Status(ctx, runID string) (*Run, error)` (client.go:177) | `GET /v1/runs/{id}` |
| `Cancel` | `Cancel(ctx, runID string) (*Run, error)` (client.go:187) | `POST /v1/runs/{id}/cancel` |
| `Record` | `Record(ctx, runID string, w io.Writer) error` (client.go:197) | `GET /v1/runs/{id}/record` (tar 를 `w` 로 복사) |
| `Capabilities` | `Capabilities(ctx) ([]Capability, error)` (client.go:240) | `GET /v1/capabilities` |

같은 클라이언트에 `Asks`(`GET /v1/asks`, client.go:154), `Answer`(client.go:167),
`Wait`(종료 상태까지 폴링, client.go:214), 패키지 함수 `Terminal(state)`
(`state == "SUCCEEDED" || state == "FAILED"`, client.go:208), `Principal()`
(`git config --get user.email`, client.go:34)이 함께 있다. DTO 는 `Run`
(`{run_id, state, assigned?, verdict?, steps?, warnings?}`, client.go:88-98),
`Step`, `Assigned`, `AskItem`, `ShownArtifact`, `Capability`
(`{capability, nodes, attrs}`, client.go:231-236)다.

### internal/store 주요 함수

`store` 는 라우트를 모른다 — 위 핸들러가 이 메서드들을 부른다. 상태 저장소는
경쟁·가변 사실(광고·점유·Run·단계)을 쥔다.

**Cancel** — `func (s *Store) Cancel(ctx, runID, by string) (string, error)`
(reap.go:170-). 새 상태를 만들지 않고 `* -> FAILED` 로 간다. 둘째 인자는 자유
서술 사유가 아니라 `by`(취소자)이며, 핸들러가 `principal(r)` 을 넘긴다
(`api.go:624`). verdict 노트에 `"cancelled by: " + by` 를 적고(reap.go:179),
Run 을 `FAILED` 로, `PENDING`/`CLAIMED`/`ASKED` 단계를 `FAILED` 로 만든 뒤
`DELETE FROM leases`(=다음 하트비트에서 빠지는 것이 곧 enode 에 대한 취소 통보,
ADR-016) 하고 봉인한다. 이미 종료면 아무것도 하지 않고 그 상태를 돌려준다(멱등,
reap.go:175-176).

**Reap** — `func (s *Store) Reap(ctx, *slog.Logger) (int, error)` (reap.go:19-95).
만료된 임대를 회수한다(ADR-008 — 시간이 감시자다). 순서:
1. `ExpireAsks` 로 기한 지난 되묻기를 먼저 정리한다(reap.go:22-33).
2. `not_after <= now()` 인 임대를 가진 Run 을 `FAILED` 로 만든다 — 단
   `ASKED` 단계가 있는 Run 은 `NOT EXISTS (... st.state = 'ASKED')` 로 제외한다
   (reap.go:57-63, ADR-047: 답을 기다리는 동안 노드에서 아무것도 안 도므로 I1
   충돌이 없다).
3. 종료한 Run 의 `PENDING`/`CLAIMED`/`ASKED` 단계를 `FAILED` 로 정리한다.
4. 종료한 Run 의 임대를 전부 `DELETE` 한다(I2).
5. 회수된 Run 을 `sealExpired` 로 봉인한다.
`RunReaper(ctx, every, log)` 가 이 스캔을 주기적으로, 그리고 시작 시 한 번 먼저
돈다(재시작 스캔, reap.go:143-157).

**ask 전이** (ADR-032):
- `raiseAsks(ctx, tx, runID)` (ask.go:24-82) — `needs` 가 전부 `DONE`/`SKIPPED`
  인 `PENDING` `kind='ask'` 단계를 `UPDATE steps SET state='ASKED',
  started_at=now(), ask_deadline=$3` 로 올린다(ask.go:67-70). Run 생성 시와 각
  단계 뒤(`afterStep`)에 호출된다. `ask_deadline` 은 `ask.timeout.after` 에서
  계산하고, 없으면 `NULL`(무한 대기). `AskEvent` 를 만들어 푸시 웹훅에 실을 수
  있게 하되 인박스가 정본이다(재시도 없음).
- `AnswerStep(ctx, runID, seq, principal, body, limit) (bool, error)` — 답을
  blob 처럼 검증·저장한다. `ErrNoAsk`/`ErrNotAnswerer`/`SchemaViolation` 을
  낸다(핸들러가 각각 `409`/`403`/`422` 로 옮긴다).
- `PendingAsks(ctx) ([]AskView, error)` (ask.go:202) — 대기 중인 되묻기만
  돌려준다(인박스 정본).
- `ExpireAsks(ctx) ([]string, error)` (ask.go:488) — 기한 지난 ask 를 정리하고
  (then:"fail") 영향받은 `run_id` 들을 돌려준다. `Reap` 의 첫 단계다.

**describeWant** — `func describeWant(r contract.Require) string`
(verdict.go:47-58). 함대 조건을 `"fleet_has <capability> k=v ..."` 로 사람이 읽게
렌더한다 — 속성 어휘가 창발하므로(ADR-012) 무엇을 요구했는지가 안 보이면 원인을
못 찾는다. 키를 정렬해서 같은 조건이 매번 같게 봉인되게 한다. `Verify` 의 함대
조건 분기에서만 불리므로(verdict.go 내부), 봉인 직전 검증 시점에 실행된다.

---

## Data Models

`schema.sql` 은 `//go:embed` 로 실려 `Migrate` 가 파일 전체를 그대로 `exec`
한다 — 버전드 마이그레이션 도구 없이 `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`
로 열을 더한다. 경쟁이 있고 계속 바뀌는 것(광고·점유·Run·단계)만 DB 에 있고,
봉인되는 것(Run Record 디렉터리·blob 본문)은 파일시스템에 있다(schema.sql:1-8).
매칭은 SQL 이 하지 않는다 — 순수 함수 `match.Match` 가 하고, DB 는 광고와 점유를
돌려줄 뿐이다.

### nodes (schema.sql:10-20, instance 는 68)

| 열 | 타입 | 비고 |
|----|------|------|
| `node_id` | text | **PRIMARY KEY** |
| `label` | text NOT NULL | 사람이 읽는 이름 |
| `principal` | text NOT NULL | 식별이지 인증 아님 (ADR-015 §1) |
| `capabilities` | jsonb NOT NULL | `[{capability, attrs}]`, 델타 아닌 매번 전부 -> 통째 교체 |
| `expires_at` | timestamptz NOT NULL | 광고는 만료된다 (ADR-012) |
| `seen_at` | timestamptz DEFAULT now() | |
| `instance` | text (ADR-030) | 프로세스의 「이번 생」 표식, 재시작 판정용 |

### runs (schema.sql:22-47)

| 열 | 타입 | 비고 |
|----|------|------|
| `run_id` | text | **PRIMARY KEY**. runctl 이 만든다 — 재제출이 멱등 (INVARIANTS §4) |
| `state` | text NOT NULL | 아래 상태 어휘 참고 |
| `principal` | text NOT NULL | |
| `contract` | jsonb NOT NULL | 제출 전문(v1), Record 의 manifest 가 된다 |
| `assigned` | jsonb | `[{as, nodes:[{node,label}]}]`, ALLOCATING 을 지난 뒤 |
| `reject` | jsonb | 거절 사유(422/409), FAILED 의 원인 |
| `verdict` | jsonb | 대조 결과, verdict.json 이 된다 |
| `created_at` | timestamptz DEFAULT now() | |
| `ended_at` | timestamptz | |
| `work_id` | text (ADR-023) | Run 을 넘어 사는 유일한 식별자. index `runs_work_idx (work_id, created_at)` |
| `contract_versions` | jsonb NOT NULL DEFAULT '[]' | 실행 중 자란 판. `liveContract = coalesce(contract_versions -> -1, contract)` (store.go:177) |

### leases (schema.sql:57-64) — 점유 장부, I1 을 스키마로 강제

| 열 | 타입 | 비고 |
|----|------|------|
| `node_id` | text | **PRIMARY KEY** — I1(한 노드에 한 Run)의 직접 표현 (ADR-019: 임대 키 `(노드, capability)` 가 `(노드)` 로 붕괴) |
| `run_id` | text NOT NULL | **REFERENCES runs(run_id) ON DELETE CASCADE** |
| `not_after` | timestamptz NOT NULL | 허가 아티팩트의 만료 (ADR-010) |
| `nonce` | text NOT NULL | |
| `granted_at` | timestamptz DEFAULT now() | |

index `leases_run_idx (run_id)`. 두 번째 Run 이 같은 노드를 잡으려 하면 애플리케이션
로직이 아니라 **기본키 충돌** 이 막는다. I5(전부 아니면 전무)는 그 충돌에 트랜잭션
롤백을 붙여 얻으며, 그때 store 가 `ErrNodeTaken` 을 낸다(store.go:231). `node_id`
는 PK 일 뿐 `nodes` 로의 외래키가 아니다 — 광고와 점유는 생명주기가 다르다.

### steps (schema.sql:71-144)

| 열 | 타입 | 비고 |
|----|------|------|
| `run_id` | text NOT NULL | **REFERENCES runs(run_id) ON DELETE CASCADE**, PK 의 일부 |
| `seq` | int NOT NULL | 계약 `steps[]` 순서. step_id 는 `run_id#NN` 으로 노출. PK 의 일부 |
| `name` | text NOT NULL | 계약 `steps[].id` |
| `uses` | text NOT NULL | 역할 이름 |
| `kind` | text NOT NULL | `agent` \| `run` (주석). 실제로는 `ask`, `acquire` 도 쓴다 (`raiseAsks` 가 `kind='ask'` 를 고른다) |
| `state` | text NOT NULL | `PENDING` \| `CLAIMED` \| `DONE` \| `FAILED` \| `SKIPPED` (+ `ASKED`). CHECK 없음 — 어휘가 늘 때 마이그레이션을 강요하지 않으려고 |
| `node_id` | text | 배정된 노드. claim 이 채운다 |
| `attempt` | int NOT NULL DEFAULT 0 | |
| `started_at` / `ended_at` | timestamptz | |
| `result` | jsonb | `exit_code · produced · harness` (ADR-020) |
| `needs` | text[] NOT NULL DEFAULT '{}' | 이 단계가 기다리는 단계 이름들. 게이트의 술어. 빈 배열은 "안 기다린다" |
| `ledger_at` | text[] (ADR-023 §6.4) | 워터마크 — 집을 때 원장에 있던 것들 |
| `claimed_instance` | text (ADR-030) | 어느 「생」이 집었는가 — 재전달(같은 생) vs 재시작(다른 생) |
| `ask_deadline` | timestamptz (ADR-032) | `NULL` 이면 무한 대기 |
| `envelope_key` | text (ADR-050) | 되먹임 봉투 열쇠, 단계마다 다르다 |
| `chosen` | boolean NOT NULL DEFAULT false (ADR-060 §3) | 갈림길에 골라진 적 있는가 |

`PRIMARY KEY (run_id, seq)`.

`chosen` 은 하나의 `SKIPPED` 로 구분 못 하는 둘을 가른다: **안 고른 SKIPPED**
(경로가 갈렸다 -> 조건은 공허하게 참)와 **고른 뒤에도 SKIPPED**(골랐는데 못 닿았다
-> 목표 미달, FAILED). `applyDispatch` 가 `SET chosen = true` 로 세우고
(dispatch.go:74) 되돌리지 않는다. `StepView`/`Steps`/`StepFiles` 는 이 열을
select 하지 않는다 — `Verify` 가 판정용으로 내부에서만 읽는다.

### 관계

```text
runs (run_id PK)
  |-- 1:N --> steps  (run_id FK, ON DELETE CASCADE)   PK (run_id, seq)
  |-- 1:N --> leases (run_id FK, ON DELETE CASCADE)   PK (node_id)

nodes (node_id PK)   독립 테이블
  leases.node_id 는 nodes 로의 FK 가 아니다 (광고와 점유는 다른 생명주기)
```

### 상태 어휘와 실제로 저장되는 상태

**Run 상태** — 상수는 `store.go:156-163` 에, 어휘 정의는 canon 의 INVARIANTS §1.1
에 있다.

| 상수 | 상수 정의됨 | runs.state 에 실제로 쓰이나 |
|------|-------------|------------------------------|
| `RESOLVING` | 예 (store.go:157) | 아니오 — 죽은 상수, 어디에도 write 안 함 |
| `ALLOCATING` | 예 (store.go:158) | 아니오 — 죽은 상수 |
| `RUNNING` | 예 (store.go:159) | 예 — `CreateRun` 이 넣는다 (api.go:454) |
| `VERIFYING` | 예 (store.go:160) | 예 — `SettleIfDone` 이 넣는다 (reap.go:279, tx 밖 `pool.Exec`) |
| `SUCCEEDED` | 예 (store.go:161) | 예 — 종료 (reap.go) |
| `FAILED` | 예 (store.go:162) | 예 — 종료. `CreateRejectedRun` 도 `FAILED` 로 넣는다 (store.go:344) |

즉 `runs.state` 에 실제로 나타나는 값은 `RUNNING`, `VERIFYING`, `SUCCEEDED`,
`FAILED` 넷뿐이다. `VERIFYING` 은 종료 write 트랜잭션 밖에서 먼저 쓰이므로
관측자가 `RUNNING -> VERIFYING` 전이를 커밋 전에 볼 수 있다.

**오늘 없는 상태**: `QUEUED` 는 코드에 없다(주석 `reap.go:162` 에만 등장). `CREATED`
상수도 없다. `DRY_RUN` 은 dry-run 응답의 와이어 값일 뿐 저장되지 않는다. `QUEUED`
와 `202` 는 팩이 ADR-064 로 더한다.

**단계 상태** (verdict.go:19-32) — `PENDING`, `CLAIMED`, `DONE`, `FAILED`,
`SKIPPED`, `ASKED`. 여섯 전부 `steps.state` 에 실제로 쓰인다. `reap` 의 집계는
`PENDING`/`CLAIMED`/`ASKED` 를 「남은 것」, `FAILED` 를 「실패」로 세고,
`DONE`/`SKIPPED` 는 종료-비실패다. 그래서 새 단계 상태를 더하면 그 열거를 반드시
다시 봐야 한다 — `ASKED` 를 빠뜨려 질문이 열린 채 Run 이 끝나는 결함을 실제로
밟았다. `SKIPPED` 는 dispatch 가 안 간 경로이고, `ASKED` 단계는 노드에 안 간다
(claim 이 `node_id` 로 거른다).
