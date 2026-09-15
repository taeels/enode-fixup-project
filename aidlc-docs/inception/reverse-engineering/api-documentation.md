# API 문서 — 오늘의 표면

이 문서는 코드가 **지금 등록하는 것**을 적는다. 정본은 `enode-design` 의
`protocol/mediator-api.md` 이고, 어긋나면 정본이 아니라 이 문서가 낡은 것이다 —
여기는 측정값이다.

**2026-09-15 전면 재측정.** 2026-09-08 판은 라우트 15 를 적었다. 오늘은 26 이다.

---

## 1. Mediator REST — 라우트 26

등록 자리는 셋이다.

```text
   internal/api/api.go:88-113          열아홉.  핵심 + 데모 제출 + 정적 마운트 둘
   internal/api/demo_gallery.go:75-80  여섯.  데모 갤러리.  설정 둘이 다 켜져야 등록된다
   합계                                26
```

`api.go` 한 파일의 `mux.HandleFunc` 만 세면 17 이다. 그 셈법은 `mux.Handle(` 로
붙인 셋(데모 제출 · 리다이렉트 · 정적)과 갤러리 여섯을 놓친다.

### 1.1 보호

```text
   s.auth      Bearer 토큰 상수시간 비교 + X-Enode-Principal 을 컨텍스트에 넣는다
               토큰이 설정에 없으면(cfg.Token == "") 전부 401 이다
   read(...)   조건부 래퍼.  데모 모드면 s.limit (무인증 + 전역 토큰버킷),
               아니면 s.auth.  감싸는 것은 nodes · runs · run 상세 셋
   s.limit     무인증 + 초당 120 · 버스트 240 의 전역 버킷.  넘으면 429 + Retry-After: 1
   무인증      GET /{$} 와 GET /ui/ 아래 정적 파일
```

`X-Enode-Principal` 은 **식별이지 인가가 아니다** (ADR-015 §1). 검증하지 않는다 —
`~/.gitconfig` 는 사용자가 쓰는 파일이라 자기 신고다. 예외가 하나 있다:
`POST .../answer` 는 계약이 답할 사람을 지정했으면 그 값으로 403 을 낸다.

### 1.2 핵심 라우트

| 메서드 · 경로 | 핸들러 | 보호 | 성공 | 거절 |
|---|---|---|---|---|
| `POST /v1/nodes` | postNodes | auth | 200 | 400 · 503 |
| `GET /v1/nodes` | getNodes | read | 200 | 400(인자 있음) · 503 |
| `GET /v1/runs` | getRuns | read | 200 | 400 · 503 |
| `POST /v1/nodes/{id}/claim` | postClaim | auth | 200 · 204 | 503 |
| `POST /v1/runs/{run}/steps/{seq}/result` | postResult | auth | 200 | 400 · 409 · 503 |
| `POST /v1/runs` | postRuns | auth | 201 · 202 | 400 · 409 · 422 · 503 |
| `POST /v1/runs/dry-run` | postDryRun | auth | 200 | 400 · 503 |
| `GET /v1/runs/{id}` | getRun | read | 200 | 404 · 503 |
| `GET /v1/capabilities` | getCapabilities | auth | 200 | 503 |
| `GET /v1/asks` | getAsks | auth | 200 | 503 |
| `POST /v1/runs/{run}/steps/{seq}/answer` | postAnswer | auth | 200 | 400 · 403 · 409 · 413 · 422 · 503 |
| `GET /v1/runs/{id}/ledger` | getLedger | auth | 200 | 404 · 503 |
| `GET /v1/runs/{id}/record` | getRecord | auth | 200 (tar) | 404 · 409 |
| `POST /v1/runs/{id}/cancel` | postCancel | auth | 200 | 404 · 503 |
| `PUT /v1/runs/{run}/steps/{seq}/log` | putLog | auth | 204 | 400 · 404 · 410 · 503 |
| `PUT /v1/runs/{run}/steps/{seq}/blob/{name}` | putBlob | auth | 204 | 400 · 404 · 410 · 413 · 422 · 503 |
| `GET /v1/runs/{run}/blob/{name}` | getBlob | auth | 200 | 404 |

### 1.3 데모 라우트 — 설정이 켜져야 등록된다

```text
   POST /v1/demo/runs                        cfg.Demo 일 때만.  고정 시나리오 제출
   GET  /v1/demo/gallery/projects            cfg.Demo && cfg.DemoGallery && 토큰 있음
   POST /v1/demo/gallery/runs                     "
   POST /v1/demo/gallery/comments                 "
   GET  /v1/demo/gallery/runs/{id}                "
   GET  /v1/demo/gallery/history/{id}        위 조건 + s.auth
   POST /v1/demo/gallery/runs/{id}/publish        "
```

### 1.4 정적 마운트 둘

```text
   GET /{$}    302 -> /ui/          http.RedirectHandler
   /ui/        internal/api/ui      임베드 정적 파일.  보안 헤더 다섯을 싣는다
```

---

## 2. 자세히 — 이 회차가 딛는 라우트

### 2.1 `PUT /v1/runs/{run}/steps/{seq}/log`

그 단계가 뱉은 것을 남긴다 (ADR-005 의 `logs/`).

```text
   요청       본문은 text/plain.  ?name=<단계 이름> (없으면 "step")
   저장       record.Store.AppendLog -> <root>/run-<id>/logs/<seq02>-<name>.log
              O_CREATE|O_WRONLY|O_APPEND — 이어 붙이기다
   상한       cfg.Artifacts.MaxBlobBytes.  호출마다 건다.  넘으면 잘라 저장하고
              "... log truncated at N bytes" 한 줄을 붙인다
   응답       204.  본문이 없다
   거절       400 순번 · 404 없는 Run · 410 이미 종료된 Run (I4) · 503
```

**오늘 부르는 쪽은 단계 끝 한 번이다.** `claim.go` 가 에이전트 단계와 명령 단계
각각 한 자리에서 전체 버퍼를 올린다. 도중 호출을 막는 것은 없다 — `putLog` 는
봉인 전이면 언제나 받는다.

**`AppendLog` 는 총 길이를 안 돌려준다.** 반환값은 그 호출이 쓴 바이트 수다.
상한도 호출마다 걸리므로, 여러 번 나눠 올리면 파일 전체는 상한을 넘을 수 있다.

### 2.2 `GET /v1/runs/{id}/record`

봉인된 tar 를 낸다.

```text
   성공       200 · application/x-tar
   409        "run is not sealed yet" — I4.  봉인 전에는 Record 가 아니다
   404        없는 Run
   내용       manifest.json · steps/NN-*.json · logs/NN-*.log ·
              blobs/NN.A-<name> · verdict.json
```

**진행 중인 것을 내려받는 라우트는 없다.** 올리는 `PUT`은 있고 내려받는 `GET`은
없다 — 그래서 제어판이 지난 트랜스크립트를 tar 로 받아 풀어 `logs/` 만 꺼낸다.

### 2.3 `GET /v1/runs`

```text
   질의       limit(기본 100 · 상한 1000) · since(RFC3339 · 포함 하한) ·
              state · work.  모르는 인자는 무시한다
   응답       {"observed_at": <DB 시계>, "runs": [RunRow...]}  최신이 앞
   RunRow     run_id · state · verdict · work_id · created_at · ended_at ·
              assigned[] · submitter.  steps 는 안 싣는다
   헤더       Cache-Control: no-store
```

### 2.4 `GET /v1/runs/{id}`

```text
   응답       run_id · state · assigned · reject · verdict · steps[] ·
              warnings · requires
   StepView   seq · id · state · uses · node · needs[] · attempt · chosen ·
              started_at · ended_at.  본문도 판정도 안 싣는다 (ADR-025 §5)
```

### 2.5 `GET /v1/nodes`

```text
   질의       받지 않는다.  하나라도 오면 400
   응답       {"observed_at": <DB 시계>, "nodes": [NodeView...]}
   NodeView   node_id · label · instance · capabilities[] · seen_at ·
              expires_at · lease(없으면 null) · draining
   헤더       Cache-Control: no-store
```

---

## 3. 노드 제어판 REST (`enode panel`)

Mediator 가 아니다. 노드 기계에서 도는 별개 프로세스이고 기본 바인딩은
`127.0.0.1:8081` 이다.

| 메서드 · 경로 | 하는 일 | 출처 |
|---|---|---|
| `GET /` | 화면 한 장 | 임베드 HTML 문자열 (`page.go`) |
| `GET /api/state` | 신원 · 능력 · 프로세스 · 작업 · drain · Mediator | 로컬 파일 넷 + Mediator 두 홉 |
| `POST /api/drain?mode=` | `graceful` 또는 `at-boundary` 를 정책 파일에 쓴다 | 로컬 파일 |
| `POST /api/undrain` | 푼다 | 로컬 파일 |
| `POST /api/stop` | 도는 Run 을 cancel 하고 데몬에 신호 | Mediator + 로컬 프로세스 |
| `POST /api/start` | 자기 실행파일을 `--config` 로 다시 띄운다 | 로컬 프로세스 |
| `GET /api/logs` | 데몬 로그 꼬리 200 줄 | 로컬 파일 |
| `GET /api/transcript` | 링 파일 스냅샷 | 로컬 파일 |
| `GET /api/runs` | 이 노드가 assigned 인 Run 만 | `GET /v1/runs` 를 받아 거른다 |
| `GET /api/record?run=` | tar 에서 `logs/*.log` 만 꺼낸다 | `GET /v1/runs/{id}/record` |

인증은 바인딩이 정한다 — loopback 이면 없고, 아니면 정책 파일의 `panel_token` 이
필수이며 없으면 서버가 뜨기를 거부한다.

### 3.1 `GET /api/transcript` 의 응답

```text
   링이 없거나 못 읽으면   {"available": false}
   있으면                  {"available": true, "generation": N, "total": N,
                           "data": "<오래된 것 -> 새것 순서의 바이트>"}
```

`generation` 은 단계가 바뀔 때마다 오른다 — 화면이 갈리는 순간을 이 값으로 안다.
`total` 이 용량(512 KiB)을 넘으면 `data` 는 최근 용량만큼이고 첫 줄이 잘려 있다.

**오늘 이 라우트가 에이전트 단계에서 내는 것은 빈 링이다.** 하네스 stdout 의 링
tee 가 꺼져 있다 (`runner.go` 의 ⑥). 명령 단계만 흐른다.

---

## 4. 노드가 Mediator 를 부르는 순서 (`internal/enode.Client`)

```text
   POST /v1/nodes                    광고 = 하트비트 = 갱신.  기본 60초
   POST /v1/nodes/{id}/claim         롱폴.  최대 2h.  일이 없으면 204
   PUT  .../steps/{seq}/log          단계 끝에 한 번.  result 보다 먼저
   PUT  .../steps/{seq}/blob/{name}  산출물마다
   POST .../steps/{seq}/result       마지막
```

`ENODE_TOKEN` 은 하네스 환경으로 안 넘어간다 — 열쇠가 하나라는 전제가 I1 이
서 있는 자리다.

---

## 5. 내부 API — 이 회차가 만지는 것

### `internal/enode`

```text
   Harness.Argv(p, io) []string          순수 함수.  오늘 이미
                                         -p --output-format stream-json --verbose 를 낸다
   Harness.Decode(r, exitCode, emit)     시그니처는 스트림을 받는다.  구현은 아직 배치다 —
                                         io.ReadAll 로 EOF 까지 읽고 final 사건 하나만 낸다
   ParseClaude(stdout, exitCode)         lastJSONObject 로 마지막 JSON 객체를 집는다.
                                         type != "result" 면 harness_error (⑯)
   OpenRing(path, capacity) (*Ring, …)   고정 크기 링.  WriteAt 만 쓴다
   Ring.Write / Reset / Close            io.Writer.  실패를 삼킨다
   ReadRing(path) (Snapshot, error)      제어판이 읽는다.  잠금 없음
   TranscriptPath(configPath) string     <설정 경로에서 확장자 뺀 것>.transcript
   Client.UploadLog(ctx, run, seq, …)    PUT .../log 한 번
```

### `internal/record`

```text
   Store.AppendLog(run, seq, name, r, limit) (int64, error)
        이어 붙인다.  돌려주는 것은 이번 호출의 바이트 수다 — 파일 총 길이가 아니다
   Store.Tar(run, w) error         봉인된 묶음 전체
   Store.Sealed(run) bool          verdict.json 의 쓰기 비트로 판정
```

### `internal/store`

```text
   Steps(ctx, run) ([]StepView, error)        실행 중 관측 (ADR-025)
   Nodes(ctx) ([]NodeView, time.Time, error)  관측된 함대 (ADR-065)
   Runs(ctx, RunFilter) ([]RunRow, …)         목록.  최신이 앞
```

---

## 6. 데이터 모델 — 봉인 기록의 모양

```text
   run-<id>/
     manifest.json          계약 전문 + 버전 열 + 노드 정보 (성질 4 자기충족)
     steps/NN-<name>.json   단계 하나의 사실.  node id 와 label 이 반드시 든다
     logs/NN-<name>.log     그 단계가 뱉은 것.  2026-09-12 부터 허용목록으로 걸러진다
     blobs/NN.A-<name>      산출물.  NN 은 순번, A 는 회차
     verdict.json           판정.  이 파일의 쓰기 비트가 봉인 여부다
```

`logs/` 의 내용이 2026-09-12 에 바뀌었다 (`runner.go` 의 `selectLogs`).

```text
   전문으로 남는 것    첫 system/init 줄 · 마지막 type=="result" 줄 · stderr 전체
   껍데기로 남는 것    그 밖의 모든 사건 — {"type","subtype","tools","ok","tokens"}
   안 남는 것          JSON 객체가 아닌 줄 · type 이 문자열이 아닌 줄.  세기만 한다
   꼬리표              {"type":"enode.elided","events":N,"bytes":N} 한 줄
```
