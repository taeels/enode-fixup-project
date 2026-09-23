# API 문서 — 오늘의 표면

이 문서는 코드가 **지금 등록하는 것**을 적는다. 정본은 `enode-design` 의
`protocol/mediator-api.md` 이고, 어긋나면 정본이 아니라 이 문서가 낡은 것이다 —
여기는 측정값이다. 단 정본이 앞으로의 라우트를 먼저 적어 둔 자리는 반대로
코드가 아직 없는 것이다 (5절).

**2026-09-23 전면 재측정.** 2026-09-15 판은 라우트 26 을 적었다. 오늘은 27 이다.
기준 커밋 `195a5d0`.

---

## 1. Mediator REST — 라우트 27

등록 자리는 둘이다.

```text
   internal/api/api.go:88-114          스물하나.  핵심 + 데모 제출 + 정적 마운트 둘
   internal/api/demo_gallery.go:75-80  여섯.  데모 갤러리.  설정 둘이 다 켜져야 등록된다
   합계                                27
```

`mux.HandleFunc` 와 `mux.Handle(` 을 둘 다 센다. 앞의 것만 세면 데모 제출 ·
리다이렉트 · 정적 셋을 놓친다.

### 1.1 보호

```text
   s.auth      Bearer 토큰 상수시간 비교 + X-Enode-Principal 을 컨텍스트에 넣는다
               토큰이 설정에 없으면(cfg.Token == "") 전부 401 이다
   read(...)   조건부 래퍼.  데모 모드면 s.limit (무인증 + 전역 토큰버킷),
               아니면 s.auth.  감싸는 것은 넷 — nodes · runs · run 상세 · GET log
   s.limit     무인증 + 초당 120 · 버스트 240 의 전역 버킷.  넘으면 429 + Retry-After: 1
   무인증      GET /{$} 와 GET /ui/ 아래 정적 파일
```

`X-Enode-Principal` 은 **식별이지 인가가 아니다** (ADR-015 §1). 검증하지 않는다.
예외가 하나 있다: `POST .../answer` 는 계약이 답할 사람을 지정했으면 그 값으로
403 을 낸다.

**노드 표면도 같은 토큰이다.** `POST /v1/nodes` · `claim` · `result` · `PUT log` ·
`PUT blob` 은 전부 `s.auth` 하나를 지난다 — 노드를 가리는 별도 자격이 없다. 결과를
받는 자리는 인증이 아니라 상태로 거른다 — `ReportStep` 이
`WHERE run_id=$1 AND seq=$2 AND node_id=$3 AND state='CLAIMED'` 로 갱신하고, 0 행이면
거절한다. `node_id` 는 본문의 `node` 값이다 — 노드의 자기 신고다. 인스턴스
(`claimed_instance`)는 claim 의 재전달과 재시작 판정에만 쓰이고 결과 보고에서는 안
대조한다.

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
| `PUT /v1/runs/{run}/steps/{seq}/log` | putLog | auth | **200** | 400 · 404 · 410 · 503 |
| `GET /v1/runs/{run}/steps/{seq}/log` | getLog | read | 200 | 400 · 404 · 503 |
| `PUT /v1/runs/{run}/steps/{seq}/blob/{name}` | putBlob | auth | 204 | 400 · 404 · 410 · 413 · 422 · 503 |
| `GET /v1/runs/{run}/blob/{name}` | getBlob | auth | 200 | 404 |

`PUT log` 의 성공이 204 에서 **200** 으로 바뀌었다 — 응답 헤더 `X-Enode-Log-Bytes` 가
파일의 총 길이를 싣는다.

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
   /ui/        internal/api/ui      임베드 정적 파일.  보안 헤더 다섯을 싣는다.
                                    카드 렌더러(internal/transcriptui)를 /ui/shared/ 아래로 낸다
```

---

## 2. 자세히 — 단계 하나가 지나는 라우트

### 2.1 `POST /v1/nodes/{id}/claim`

롱폴이다. 일이 없으면 최대 대기 뒤 204, 있으면 200 에 단계 하나. 단계 상태가
`PENDING -> CLAIMED` 로 가고 `claimed_instance` 에 노드 인스턴스가 박힌다. 같은
인스턴스가 다시 물으면 같은 단계를 재전달하고, 다른 인스턴스가 나타나면 그 단계를
실패시킨다 (ADR-030).

### 2.2 `PUT /v1/runs/{run}/steps/{seq}/log`

갈래가 쿼리 하나로 갈린다. 새 라우트가 아니다.

```text
   쿼리 없음        logs/<seq02>-<name>.log 에 이어 붙인다.  단계 끝의 선별본이다
                    AppendLog 가 파일 총 길이를 낸다 -> X-Enode-Log-Bytes
   ?progress=1     진행 파일에 이어 붙인다.  도는 동안의 원문이다
     &attempt=<n>  필수.  0 부터다.  시도가 바뀌면 앞 시도의 바이트를 걷는다
                    응답 헤더 X-Enode-Log-Bytes (총 길이) · X-Enode-Log-Attempt ·
                    X-Enode-Log-Capped: 1 (총 길이 상한에 닿았으면)
   상한             cfg.Artifacts.MaxBlobBytes (기본 10 MiB).  넘으면 표시 줄 하나를 붙이고 멈춘다
   거절             400 순번 · 400 attempt · 404 없는 Run · 410 이미 종료된 Run (I4) · 503
```

**노드는 서버가 준 총 길이로 자기 오프셋을 맞춘다** — 자기가 보낸 바이트 수를 안
믿는다 (`upload.go` 의 `LogChunkAck`).

### 2.3 `GET /v1/runs/{run}/steps/{seq}/log` (신규)

도는 중에도 답한다. **Record 가 아니다** — `GET record` 는 봉인 전에 여전히 409 다.

```text
   쿼리         name (없으면 "step") · from=<바이트 오프셋> · as=raw | events (기본 raw)
   출처         봉인 전 = 진행 파일 (원문).  봉인 뒤 = logs/NN-*.log (선별본)
   헤더         X-Enode-Log-Bytes (파일 총 길이 · 조각 길이가 아니다) ·
                X-Enode-Log-Source: progress | sealed ·
                X-Enode-Log-Attempt (봉인 뒤는 0 — 값이 없다는 뜻) ·
                X-Enode-Log-Capped: 1
   한 응답      최대 1 MiB (maxLogSliceBytes).  설정 키가 없다 — 보호이지 손잡이가 아니다
   as=events    같은 조각을 transcript.Parse 에 넣어 사건 배열로 낸다.  from 이 줄 한가운데면
                첫 줄을 버린다
   없는 seq     404 가 아니라 200 에 총 길이 0.  404 는 없는 Run 하나뿐이다
```

### 2.4 `POST /v1/runs/{run}/steps/{seq}/result`

단계를 끝내는 유일한 전이다. 본문은 노드의 `Result` 다.

```text
   node · exit_code · produced[] · harness (agent 만) · workspace (Prep) ·
   changed[] (계약이 지목한 경로 중 바뀐 것.  ADR-037) · error · environment (신규)
```

`environment` 는 이 단계를 실제로 실행한 준비 산출물과 runtime 정책이다
(`execenv.Record` — `profile_sha256` · `prepared_environment_id` · `runtime` ·
`workspace_target` · `uid` · `gid` · `ssh` · `tmp_size` · `tmp_executable`). 봉인되는
`steps/NN-*.json` 에 그대로 들어간다.

**종료 시각 · Finalize 시각 · 업로드 시각이 없다.** `steps.ended_at` 은 result 가
닿은 순간의 `now()` 다. 명령이 끝난 순간은 결과 보고가 닿는 순간과 밖에서 구별되지
않는다 (5절).

### 2.5 `GET /v1/runs/{id}`

```text
   응답       run_id · state · assigned · reject · verdict · steps[] ·
              warnings · requires
   StepView   seq · id · state · uses · node · needs[] · attempt · chosen ·
              started_at · ended_at.  본문도 판정도 안 싣는다 (ADR-025 §5)
```

**phase 가 없다.** `CLAIMED` 인 단계가 명령을 도는 중인지, 명령이 끝나 수확 · 업로드
중인지 이 응답으로 못 가른다.

### 2.6 `GET /v1/runs/{id}/record`

봉인된 tar 를 낸다. 봉인 전 409 (I4). 내용은 6절.

### 2.7 `GET /v1/runs` · `GET /v1/nodes`

```text
   GET /v1/runs    limit(기본 100 · 상한 1000) · since · state · work.  최신이 앞
                   RunRow = run_id · state · verdict · work_id · created_at · ended_at ·
                   assigned[] · submitter.  steps 는 안 싣는다
   GET /v1/nodes   질의 인자를 안 받는다 (하나라도 오면 400)
                   NodeView = node_id · label · instance · capabilities[] · seen_at ·
                   expires_at · lease(없으면 null) · draining
   둘 다           {"observed_at": <DB 시계>, ...} · Cache-Control: no-store
```

---

## 3. 노드 광고 — `POST /v1/nodes` 의 속성 어휘

광고는 평평한 문자열 맵이고 매처는 부분집합 비교만 한다 (ADR-011). 오늘 노드가 내는
키는 이렇다. **굵게 적은 셋이 2026-09-15 판 뒤에 생겼다.**

```text
   값싼 쪽 (광고마다)   os · host_arch · ws · **machine** · board · tag ·
                        arch (예전 어휘.  첫째 하나) · **arch.<이름>=yes** (할 줄 아는 것마다)
   비싼 쪽 (탐지 시계)  harness · harness.<이름>=1 · repo · mcp.<이름> ·
                        **overlay=kernel|userns|fuse**
   정책                 policy.drain (drain 이 걸렸을 때)
   사람이 적는 것       labels.  탐지값을 못 덮는다
```

`arch` 와 `arch.<이름>` 은 **워크스페이스의 여유 디스크가 `min_free_gb`(기본 10) 이상일
때만** 실린다. 모자라면 그 둘만 빠지고 나머지 광고는 그대로 나간다 — 노드가 광고에서
통째로 빠지는 기제는 없다.

**없는 키** — `workspace.writes` · `ir` · `repo.built.<이름>` 을 내는 코드가 없다
(ADR-072 §6.4 · ADR-077 §8 이 정했다).

---

## 4. 노드 제어판 REST (`enode panel`)

Mediator 가 아니다. 노드 기계에서 도는 별개 프로세스이고 기본 바인딩은
`127.0.0.1:8081` 이다. **라우트 열하나** (2026-09-15 판은 열).

| 메서드 · 경로 | 하는 일 | 출처 |
|---|---|---|
| `GET /` | 화면 한 장 | 임베드 HTML 문자열 (`page.go`) |
| `GET /api/state` | 신원 · 능력 · 프로세스 · 작업 · drain · Mediator | 로컬 파일 넷 + Mediator 두 홉 |
| `POST /api/drain?mode=` | `graceful` 또는 `at-boundary` 를 정책 파일에 쓴다 | 로컬 파일 |
| `POST /api/undrain` | 푼다 | 로컬 파일 |
| `POST /api/stop` | 도는 Run 을 cancel 하고 데몬에 신호 | Mediator + 로컬 프로세스 |
| `POST /api/start` | 자기 실행파일을 `--config` 로 다시 띄운다 | 로컬 프로세스 |
| `GET /api/logs` | 데몬 로그 꼬리 200 줄 | 로컬 파일 |
| `GET /api/transcript` | 링 파일 스냅샷 (원문과 사건 열) | 로컬 파일 |
| `GET /api/runs` | 이 노드가 assigned 인 Run 만 | `GET /v1/runs` 를 받아 거른다 |
| `GET /api/record?run=` | 지난 단계의 로그 | `GET .../log` (`runctl.StepLog`) |
| `GET /static/card.mjs` | 카드 렌더러 | `internal/transcriptui` — 현황판과 같은 바이트 |

모든 응답에 보안 헤더 다섯이 붙는다 (`headers.go`). 인증은 바인딩이 정한다 —
loopback 이면 없고, 아니면 정책 파일의 `panel_token` 이 필수이며 없으면 서버가
뜨기를 거부한다.

---

## 5. 정본이 적었으나 코드에 없는 표면

`protocol/mediator-api.md` (서브모듈 `369270a`) 가 이미 적은 것이다. 이 목록은
어긋남의 기록이지 요구가 아니다 — 요구는 회차의 팩이 진다.

```text
   POST /v1/runs/{run}/steps/{seq}/exited   명령 종료 보고 (ADR-075 결정 7).  라우트가 없다
   result 의 exited_at · finalize · upload    Result 에 필드가 없다
   진행 조회의 phase · phase_since · exit     StepView 에 필드가 없다.  steps 표에 열이 없다
```

---

## 6. 노드가 Mediator 를 부르는 순서 (`internal/enode.Client`)

```text
   POST /v1/nodes                          광고 = 하트비트 = 갱신.  기본 60초
   POST /v1/nodes/{id}/claim               롱폴.  일이 없으면 204
   GET  /v1/runs/{run}/blob/{name}         $IN 에 깔 입력마다.  404 는 부재로 기록한다
   PUT  .../steps/{seq}/log?progress=1     도는 동안.  2초 또는 64 KiB 마다.  실행을 안 막는다
   PUT  .../steps/{seq}/blob/{name}        $OUT 의 산출물마다 (수확 뒤)
   PUT  .../steps/{seq}/log                단계 끝의 선별본 한 번
   POST .../steps/{seq}/result             마지막.  닿을 때까지 다시 보낸다
```

command 단계는 blob 을 log 보다 먼저, agent 단계는 log 를 blob 보다 먼저 올린다
(`claim.go` 의 두 경로가 순서를 따로 쓴다). 둘 다 result 는 마지막이다.

`ENODE_TOKEN` 은 하네스 환경으로 안 넘어간다 — 열쇠가 하나라는 전제가 I1 이
서 있는 자리다.

---

## 7. 내부 API — 노드의 실행 경계 (신규)

### `internal/enode` — `StepRuntime`

```text
   StepRuntime.Open(ctx, RuntimeSpec) (StepSession, error)
       RuntimeSpec = RunID · StepID · Dir(워크스페이스) · In · Out · Record
   StepSession.Paths() RuntimePaths
       단계가 보는 경로.  native 는 host 경로 그대로,
       runc-overlay 는 workspace_target · /run/enode/in · /run/enode/out
   StepSession.Project(ctx, FrameworkProjectionSpec) (FrameworkProjection, error)
       하네스 · enode · 계장 디렉터리 · 자격증명 helper 를 runtime 안 경로로 투영한다
   StepSession.Run(ctx, ProcessSpec) (exitCode int, error)
       ProcessSpec = Argv · Dir · Env · Stdin · Stdout · Stderr.  ctx 취소가 프로세스를 죽인다
   StepSession.Harvest(ctx, HarvestSpec) (HarvestResult, error)
       HarvestSpec = Workspace · Out · RecordDiff · Discover · Collect · Check · Stamp
       HarvestResult = Changed · Collected · Notes · Workspace[] · WorkspaceN ·
                       DiffBytes · DiffError · ChangedError
   StepSession.Close() error
   StepSession.Environment() *execenv.Record
```

`Harvest` 가 부르는 것은 native 의 수확 하나다 — runc-overlay helper 도 merged view 를
워크스페이스로 바꿔 같은 함수를 부른다. 그래서 두 runtime 의 수확 의미가 같다.

```text
   RecordDiff   git diff --binary 를 $OUT/workspace.diff 로.  상한 maxBlobBytes
   Discover     기준 시각 이후 바뀐 파일을 워크스페이스 전체에서 걷는다 (changedSince, 상위 2000)
   Collect      계약의 collect 글롭을 $OUT 으로 옮긴다
   Check        계약의 success_when.changed 경로만 stat 한다 (ADR-037)
```

### `internal/environment`

```text
   Load(path) (Document, error) · Parse(b) (Document, error)      profile
   Check(ctx, doc, binding, inspector) Report
   CheckWithRuntime(ctx, doc, binding, inspector, verifier) Report   ready 면 smoke 까지
   Preparer.Apply(ctx, doc, binding) (Manifest, error)             env apply
   CurrentManifest(store, name) (Manifest, error)
   PreparedRootFS(store, preparedEnvironmentID) string            <store>/.../rootfs
   RecordFor(doc, manifest) Record
```

### `internal/record`

```text
   Store.AppendLog(run, seq, name, r, limit) (int64, error)
        이어 붙인다.  돌려주는 것이 파일 총 길이로 바뀌었다 (트랜스크립트 회차의 D4)
   Store.AppendProgress(run, seq, name, attempt, r, limit) (Progress, error)
   Store.DropProgress(run) error    봉인이 먼저 부른다
   Store.Tar(run, w) error          봉인된 묶음 전체
   Store.Sealed(run) bool           verdict.json 의 쓰기 비트로 판정
```

### `internal/store`

```text
   Steps(ctx, run) ([]StepView, error)        실행 중 관측 (ADR-025)
   Nodes(ctx) ([]NodeView, time.Time, error)  관측된 함대 (ADR-065)
   Runs(ctx, RunFilter) ([]RunRow, …)         목록.  최신이 앞
```

---

## 8. 데이터 모델

### 8.1 봉인 기록의 모양

```text
   run-<id>/
     manifest.json          계약 전문 + 버전 열 + 노드 정보 (성질 4 자기충족)
     steps/NN-<name>.json   단계 하나의 사실.  node id 와 label 이 반드시 든다.
                            environment 가 실린다 (신규)
     logs/NN-<name>.log     그 단계가 뱉은 것.  크기 절단 선별본 (ADR-071)
     blobs/NN.A-<name>      산출물.  NN 은 순번, A 는 회차.
                            workspace.diff · workspace.changed 도 여기 blob 으로 온다
     verdict.json           판정.  이 파일의 쓰기 비트가 봉인 여부다

   <Root>/progress/run-<id>/   진행 파일.  기록 디렉터리의 형제다.  봉인이 지우고
                               고아는 회수기가 6시간 뒤 쓴다.  Record 가 아니다
```

`logs/` 의 선별 규칙이 2026-09-17 에 바뀌었다 (ADR-071 — 정본은 아직 「미구현」으로
적는다).

```text
   전문으로 남는 것    첫 system/init 줄 · 마지막 type=="result" 줄 · stderr 전체 ·
                       assistant 의 말 · 도구 호출
   512 바이트로 자르는 것  도구 결과 · 도구 인자
   안 싣는 것          thinking
   꼬리표              {"type":"enode.elided","events":N,"bytes":N}.  bytes 는 지운 만큼이다
```

### 8.2 노드 설정의 실행 환경 바인딩 (신규)

```text
   environment:
     profile: <경로>      공유 profile.  상대 경로면 설정 파일 옆
     store:   <경로>      준비 산출물 store
     scratch: <경로>      runtime 세션 자리.  runRoot 가 여기 생긴다
   credentials:
     ssh_dir: <경로>      runc-overlay 가 읽기 전용으로 싣는다 (profile 이 ssh: readonly 일 때)
   min_free_gb: <수>      워크스페이스의 여유.  빌드 능력 키만 가린다
```

### 8.3 실행 환경 profile (신규)

```text
   api_version: enode.dev/v1alpha1
   kind: execution-environment
   name · host{provider: apt, packages[], require{subuid_size, subgid_size, unprivileged_userns}} ·
   rootfs{builder: debootstrap, release, arch, mirror, apt{components[], packages[]}, locale,
          user{name, uid, gid}} ·
   runtime{driver: native | runc-overlay, workspace_target, tmp{size, executable},
           credentials{ssh: readonly}} ·
   verify{executables[], locale}
```

목록은 정렬되고 중복이 없어야 한다. YAML 별칭 · 앵커 · merge 키 · 문서 둘 이상 ·
모르는 키를 거절한다.
