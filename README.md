# enode

AI-Native SDLC 실행 모델의 구현. `runctl` · `Mediator` · `enode` 세 컴포넌트.

**설계 정본은 여기가 아니다.** [`enode-design`](../enode-design) 의
`adr/` · `protocol/` 이 정본이고, 이 저장소는 그것의 구현이다.
어긋나면 `protocol/INVARIANTS.md` 가 이긴다.

## 왜 코드가 늦게 시작했나

ADR 20건 · 구조적 미결 0 · 표면 13개를 먼저 고정했다. 남은 미정은 전부
**돌려봐야 아는 값**(임대 기간 · 하트비트 주기 · `max_turns` · 디스크 임계값)이고,
그래서 지금 코드가 시작된다.

## 슬라이스

**합격 기준을 우리 원칙으로 정한다** — "돌아간다" 가 아니라 발표자료 §15 의
`O1~O9` 중 무엇이 켜지는가로 잰다.

| | 슬라이스 | 켜는 것 |
|---|---|---|
| **S1** | 계약 타입 + 매처 (순수 함수) | — I/O 0. 넷의 공통 어휘 |
| **S2** | Mediator: DB 스키마 + `POST /v1/runs` | **`I5` · `I1`(=`O7`)** |
| **S3** | enode: 신원 + 로컬 잠금 + 광고 루프 | **`O8` · `O9`** |
| **S4** | ★ 두 연결 ★ 하트비트 + `claim` 롱폴 + 명령 실행 | **`O6` · `I2`** |
| **S5** | 계약 조건 대조(⑩) — `success_when` | **`O4`** |
| **S6** | Record 봉인 + `GET record` (tar) | **`O1`·`O2`·`O3` · `I4`** |
| **S7** | `runctl` — 제출 · 상태 · Record · 취소 | **`O5`·`O7`** |
| ══ | **여기까지가 스켈레톤** | |
| **S8** | blob 별 모양 + ★ 스키마 검증 ★ | **`I4`** |
| **S9** | agent 어댑터 + ★ ⑥ 의 재시도 루프 ★ |  |
| **S10** | 워크스페이스 준비 · `GET /v1/capabilities` |  |

**S4 가 최대 위험이다.** `ADR-016`(하트비트가 임대를 나른다)은 설계만 있고
한 번도 안 돌았으며, 미정 값 셋이 전부 거기서 정해진다.

## 배치

```
   internal/contract   계약 · 광고 타입. ★ 넷의 공통 어휘 ★
   internal/match      요구 → 노드. ★ 순수 함수 ★ (ADR-014 결정 3)
   internal/config     ADR-015 §4 의 우선순위. ★ 사용자 경로가 /etc 를 이긴다 ★
   internal/store      PostgreSQL. ★ 매칭 로직은 여기 없다 ★
   internal/record     ★ Run Record — DB 가 아니라 파일시스템 ★ 봉인 · blob · tar
   internal/schema     ★ 형식만 ★ 검증한다. 판정 키워드는 계약을 400 으로 거절.
   internal/enode      … + agent 어댑터(사출·기동·수확) · 하네스 봉투 정규화
   internal/runctl     제출 · 상태 · Record · 취소. ★ 무상태다 ★
   internal/api        HTTP 표면. 라우팅은 표준 라이브러리만 (Go 1.22+ ServeMux)
   internal/enode      신원 · 잠금(unix/windows) · 탐지 · ★ 광고 루프 + claim 루프 ★
   cmd/mediator  cmd/enode  cmd/runctl        ★ 셋이 다 있다 ★
```

### ★ I1 은 애플리케이션 로직이 아니라 기본키가 강제한다 ★

```sql
CREATE TABLE leases (
    node_id text PRIMARY KEY,   -- ★ 이 한 줄이 I1 이다 ★
    run_id  text NOT NULL REFERENCES runs(run_id) ON DELETE CASCADE, ...
```

`ADR-019` 결정 2 가 임대 키를 `(노드, capability)` 에서 `(노드)` 로 붕괴시킨 것의
직접 표현이다. 두 번째 Run 이 같은 노드를 잡으려 하면 코드가 아니라 **DB 가 막는다.**

```text
   ERROR: duplicate key value violates unique constraint "leases_pkey"
   DETAIL: Key (node_id)=(board-042) already exists.
```

`I5`(전부 아니면 전무)는 그 충돌에 **롤백**을 붙여 얻는다 — 손으로 해제할 것이 없다.
※ `claim`(S4)의 `SKIP LOCKED` 와는 **다른 기계**다. 저쪽은 대기열에서 하나를 집는
것이고 이쪽은 여러 자원을 한꺼번에 잡거나 전부 포기하는 것이다.

### ★ 설정 파일이 곧 신원이다 ★

```text
   node_id = hash(email ∥ hostname ∥ realpath(config))[:12]
```

한 기계에서 둘을 띄우려면 설정이 이미 둘이어야 한다 — 같은 `local.yaml` 을
두 프로세스가 읽으면 같은 자원을 둘 다 광고해 `I1` 이 깨지기 때문이다.
**이미 유일해야 하는 것을 신원으로 쓰는 데는 비용이 0 이다** (`ADR-015` §2).

실물 확인 — 같은 노트북, 설정 둘:

```text
   8d73234fac52  taeels@CT103:ws-a   arch=armv7 repo=gerrit.corp/kernel/linux
   7a02313b8c10  taeels@CT103:ws-b   board=SoC-X tag=board-042
```

`repo` 는 워크스페이스의 `git remote` 에서 유도된 것이다 —
`ssh://git@gerrit.corp:29418/kernel/linux` → `gerrit.corp/kernel/linux`.
**사람이 저장소 주소를 안 적는다.**

중복 실행은 **로컬 잠금**이 막는다 (`flock` / `LockFileEx`). Mediator 에게 재시작과
중복은 똑같이 "같은 node_id 의 새 광고" 라 구분할 정보가 없기 때문이다.

### ★ 시간이 감시자다 — 실측 ★

```text
   12:09:50  Run RUNNING · 임대 1건        Mediator kill
   12:10:05  ★ enode 가 실행 중인 단계를 스스로 중단 ★   not_after 가 지났다
   12:10:15  Mediator 재시작 → 재시작 스캔이 회수
             run=FAILED  why="임대 만료 — 갱신이 끊겼다"  leases=0
             그리고 새 Run 이 같은 자원을 201 로 받는다
```

**감시자를 새로 만들지 않았다** (`ADR-008`). 갱신이 끊기면 `not_after` 가 지나고,
Mediator 쪽은 회수 스캔이, enode 쪽은 워치독이 각자 멈춘다.

#### 실측이 구멍 하나를 쟀다

`ADR-010` 은 *단계를 **시작하기 전에** `not_after` 를 확인한다* 였는데,
그것만으로는 **긴 단계가 임대보다 오래 산다.** 첫 시험에서 임대 회수 후
**7초를 더 돌았고**, 그동안 Mediator 는 그 자원을 새 Run 에 줄 수 있었다 —
`ADR-008` 이 *옛 Run 의 flash 가 아직 돌고 있다* 로 이름 붙인 충돌이다.

→ **실행 중에도 임대를 감시해 만료되면 그 단계를 죽인다.** 창이 단계 길이가
아니라 워치독 주기(1초)로 유계가 됐다. 권위는 여전히 `not_after` 이므로
하트비트 한 번 실패로는 안 죽는다 (`ADR-016`).

### ★ 완주와 성공은 다르다 ★

이 구분이 `ADR-004` 를 지탱한다.

```text
   DONE    프로세스가 끝나고 결과를 보고했다. ★ 종료코드가 무엇이든 ★
   FAILED  아예 못 돌았다 — 프로세스를 못 띄웠거나 임대가 끝나 중단됐다
```

`exit 2` 로 끝난 빌드는 **완주한 것**이고, 그게 성공인지는 `success_when` 이 판정한다.
enode 나 핸들러가 종료코드로 미리 판정하면 **계약이 할 일을 코드가 가로채고**,
그러면 `O4` 가 성립하지 않는다.

#### O4 — 실측

```jsonc
// 단계는 exit 0 으로 완주했고 test_result 를 냈다. 내용은 "not ok 1 - spi cdiv".
"success_when": [{"step":"parent_observe","produced":["test_result"]}]
```

```json
{"state":"SUCCEEDED",
 "verdict":{"state":"SUCCEEDED","checks":[
   {"step":"parent_observe","what":"produced",
    "want":["test_result"],"got":["serial_log","test_result"],"ok":true}]}}
```

**테스트는 실패했는데 Run 은 성공이다.** 결과값을 통과 기준에 넣으면
**회귀를 증명한 Run 이 `FAILED`** 가 되어 `ADR-004` 의 네 결과표가 뒤집힌다.

반대로 `exit_code: 0` 을 물은 계약에서 빌드가 `exit 2` 면:

```json
{"state":"FAILED","verdict":{"checks":[
  {"step":"b","what":"exit_code","want":0,"got":2,"ok":false},
  {"step":"b","what":"produced","want":["build_log"],"got":["build_log"],"ok":true}]}}
```

**`verdict` 가 무엇을 왜 로 남는다** — `ADR-005` 의 *실패 원인이 Record 에 있다*.

### ★ I4 는 파일시스템이 강제한다 ★

애플리케이션이 "고치지 않기로 한다" 가 아니라 **쓰기 비트를 내린다**.
`ADR-015` §3 이 Record 를 DB 가 아니라 디렉터리에 둔 논거가 이것이다 —
봉인된 디렉터리는 권한으로 불변이 되지만 **봉인된 행은 애플리케이션 규약일 뿐**이다.

```text
   dr-xr-xr-x  run-gerrit-12345-ps3/
   -r--r--r--    manifest.json      Run · Work · 요청자 · ★ 계약 전문 ★
   dr-xr-xr-x    steps/
   -r--r--r--      01-baseline_build.json   ★ node id + label ★
   -r--r--r--      02-parent_observe.json
   dr-xr-xr-x    logs/
   -r--r--r--      01-baseline_build.log    원문 그대로
   -r--r--r--      02-parent_observe.log
   dr-xr-xr-x    blobs/                     (S8)
   -r--r--r--    verdict.json       ⑩ 의 대조 결과

   $ echo tampered > verdict.json     → 허가 거부
   $ touch steps/03-injected.json     → 허가 거부
```

#### 한 묶음에서 `O1`·`O3`·`O4` 가 동시에 보인다

```text
   steps/   8d73234fac52  taeels@CT103:ws-a   baseline_build   ┐ ★ O1 ★
            7a02313b8c10  taeels@CT103:ws-b   parent_observe   ┘ 서로 다른 기계

   logs/02  not ok 1 - spi_cdiv_readback          ★ O3 — 테스트는 실패했다 ★
   verdict  {"state":"SUCCEEDED", …}              ★ O4 — 그런데 Run 은 성공 ★
```

**계약 전문이 `manifest.json` 에 들어간다** — 성질 4(자기충족)의 핵심이고,
`ADR-020` 이 스키마를 인라인으로 둔 덕에 **무엇으로 검증했는지까지** 함께 남는다.

#### 운영상 함의 하나

**봉인은 삭제까지 막는다.** 그것이 `I4` 의 값이지만 디스크가 차면 사람이 지우지도
못한다. `ADR-005` 가 *장기 보관 정책은 MVP 밖, 쌓아두기만 한다* 로 미뤄둔 자리다.

### runctl — CLI 종료코드는 0~3 이다

와이어는 HTTP 를 쓰고 CLI 는 넷을 쓴다. **경계가 둘이라 하나로 통일하지 않는다**
(`INVARIANTS` §4). 셸이 필요로 하는 구분은 *일이 실패했나 / 내 요청이 틀렸나 /
시스템이 죽었나* 다.

```text
   0  Run 이 SUCCEEDED (또는 아직 진행 중)
   1  ★ Run 이 FAILED ★ — 요청은 정상이었다
   2  ★ 요청이 거절됐다 ★ — 계약을 고치거나(400·422) 나중에 다시(409)
   3  ★ Mediator 에 못 닿았다 ★
```

```console
$ runctl submit s6.json --wait
gerrit-12345-ps3  SUCCEEDED
  builder    8d73234fac52  taeels@CT103:ws-a
  board      7a02313b8c10  taeels@CT103:ws-b
  ok  baseline_build exit_code
  ok  parent_observe produced
$ echo $?
0
```

**신원은 `git config user.email` 에서 읽는다** — enode 는 `--global` 이지만
runctl 은 **해석되는 형태**를 쓴다. 저장소 안에서 실행되므로 *그 저장소에서
커밋할 신원*과 같아야 자연스럽고, ⑫ 의 코멘트가 그 이름으로 달린다 (`ADR-015` §1).

#### ★ 조용한 무시가 가장 나쁘다 ★

표준 `flag` 는 첫 위치인자에서 파싱을 멈춘다. 그런데 사람은
`runctl submit x.json --wait` 라고 쓴다. 그 순서를 안 받으면 플래그가
**조용히 무시된다** — 인자를 재배열해서 받는다.

### 취소 — 하트비트가 통보한다

```text
   runctl cancel <run-id>
        ▼  Mediator: * → FAILED · 단계 FAILED · ★ 임대 삭제 ★ (I2)
        ▼  다음 하트비트 응답의 임대 목록에서 ★ 빠진다 ★
        ▼  enode 워치독이 실행 중인 단계를 중단한다

   실측: 취소 → enode "임대가 끝났다" → leases=0 → 자원 재사용 가능
         verdict 에 "사람이 취소했다: taeels@gmail.com" 이 봉인된다
```

**목록에서 빠지는 것이 곧 통보다** (`ADR-016`) — 별도의 취소 신호가 없다.

### ★ ADR-020 의 경계선을 400 으로 만들었다 ★

ADR-020 이 그은 선은 산문이었다 — *에이전트가 정직하게 답했을 때 통과하지 못할 수
있으면 그건 판정이다.* **산문으로 두면 새어나간다.** 누군가 `confidence >= 0.8` 을
쓰는 순간 `ADR-004`(기계적 판정만)가 스키마를 통해 무너진다.

그래서 허용 어휘를 좁히고 **나머지는 계약 검증에서 400 으로 거절한다.**

```text
   ○ type · required · properties · enum · items · additionalProperties
   ✗ minimum · maxLength · pattern · format · minItems …

   400 step "hypothesis" 의 hypothesis: 스키마 confidence 의 minimum 는 쓸 수 없다
       — 값의 크기로 판정한다 (ADR-020: 스키마는 형식만 제약한다)
```

`I1` 을 기본키로, `I4` 를 chmod 로 강제한 것과 같은 결이다.

### ★ 어긴 산출물은 산출물이 아니다 ★

```text
   PUT blob ──▶ 스키마 위반 ──▶ 422 · ★ 저장하지 않는다 ★
                                    ▼
                          produced 에 그 이름이 없다
                                    ▼
                          success_when 이 불만족 → Run FAILED
```

**`success_when` 에 `schema_ok` 같은 새 조건이 안 생긴다.** 판정 기준은 여전히
`produced` 하나다. 위반 내역은 `feedback` 으로 되먹여진다 — 되먹이는 것이
LLM 의 의견이 아니라 **검증기의 출력**이라는 것이 OpenHands 와의 차이다.

### 별 모양 — 노드끼리 직접 주고받지 않는다

```text
   PUT  /v1/runs/{run}/steps/{seq}/blob/{name}   생산자는 자기가 몇 번째인지 안다
   GET  /v1/runs/{run}/blob/{name}               소비자는 이름만 안다 — ★ 최신 ★ 을 준다
```

경로가 비대칭인 이유가 있다. 계약에서 `parent_build` 와 `patch_build` 가
**둘 다 `artifact` 를 낸다.** 이름만으로 키를 잡으면 뒤엣것이 앞엣것을 덮어
**차분 반증의 두 아티팩트를 봉인된 기록에서 구분할 수 없게 된다.**

```text
   실측:  blobs/01-artifact  ELF-parent-…      ┐ ★ 둘 다 남는다 ★
          blobs/03-artifact  ELF-patch         ┘
          board 노드의 로그: "받은 것: ELF-parent-…" → "받은 것: ELF-patch"
```

`GET` 은 `ADR-018` 대로 나중에 302 로 저장소를 가리킬 수 있다 — `http.Client` 가
리다이렉트를 따르므로 **enode 코드는 그때도 안 바뀐다.**

### agent 어댑터 — 넷으로 쪼갠 것 중 셋

```text
   ① 사출   $IN 에 이전 산출물 · 프롬프트에 ★ 배출 규약 + 스키마 + 되먹임 ★
   ② 기동   claude -p --output-format json --max-turns N
   ③ 되묻기 ★ 비어 있다 ★ — ask:never (ADR-013 이 --interactive 를 400 으로 거절했다)
   ④ 수확   $OUT 파일 → blob 업로드. 올라간 것만 produced.
```

`ADR-013` 이 [미정] 으로 남긴 **프롬프트에 배출 규약을 어떻게 심나**를 닫았다 —
규약 · 스키마 · 되먹임 · 요청 순서로 쌓는다. **되먹임을 요청 바로 앞에** 두면
무엇을 고쳐야 하는지가 가장 가깝게 놓인다.

#### 하네스 봉투는 계약이 될 수 있다

`ADR-013` 결정 3 의 부분 정정(`ADR-020`)이 코드가 됐다.

```text
   harness_error · timeout   ★ 완주가 아니다 ★ — 크래시는 반쯤 쓴 파일을 남긴다
   max_turns · max_tokens    ★ 완주다 ★ — produced 가 판정. 단 Record 에 남긴다.
   ok                        produced 가 판정

   Record:  {"reason":"ok","turns":3,"cost_usd":0.42}   ← 예산 신호
```

봉투를 **줄 단위로 찾으면 안 된다** — 여러 줄로 예쁘게 찍혀 올 수 있다.
출력 끝의 마지막 유효 JSON 객체를 집는다.

### ★ ⑥ 의 재시도 루프 — Mediator 가 돈다 ★

```text
   write_test(agent) ──▶ Mediator ──▶ parent_build(builder)
          ▲                                 │ exit 2
          └───── feedback: build_log ───────┘
```

**두 노드는 서로를 모른다.** 이것이 `ADR-014` 결정 1(Mediator 가 시퀀서)의
결정적 근거였고, 여기가 그것이 실물이 되는 자리다.

`INVARIANTS` §2 의 *재실행하지 않는다* 와 부딪히지 않는다 — 그건 **크래시 후 재개**에
대한 것이고, 이 루프는 **계약이 미리 선언한 것**이다. 계약 작성자가 반복해도 안전한
단계만 넣을 책임을 진다(보드를 두 번 flash 하는 단계를 검증자로 쓰면 안 된다).

**소진은 `verdict` 가 잡는다** — 루프는 제어 흐름이고 성패는 `within_attempts` 가 정한다.

#### ★ 실측이 의미 버그를 잡았다 ★

blob 최신성을 **가장 큰 순번**으로 정했더니 재시도가 깨졌다.

```text
   1회차   write_test(1) BROKEN → parent_build(2) 가 같은 이름으로 복사
   2회차   write_test(1) GOOD   → ★ 순번이 작아서 앞 회차의 BROKEN 이 이긴다 ★
```

**순번 순서는 전진만 할 때의 규칙이고 재시도 루프는 뒤로 돌아간다.**
→ 파일 이름에 회차를 넣고(`01.1-test_source`) **(회차, 순번)** 순서로 고른다.
mtime 은 같은 순간에 쓰이면 순서가 안 정해져 쓰지 않는다. 재시도는 대상과
검증자의 회차를 **함께** 올리므로 한 회차 안에서는 순번이 순서다.

```text
   blobs/  01.0-test_source  01.1-test_source     ★ 회차가 전부 남는다 ★
           02.0-build_log    02.1-build_log       "왜 두 번 시도했는가" 가 재구성된다
```

### 워크스페이스 준비 — ★ 순서가 셋이고 뒤바꾸면 안 된다 ★

```text
   ① reset --hard   추적 변경을 버린다 — 안 하면 checkout 이 거절된다
   ② checkout       목표 리비전으로 (없으면 fetch)
   ③ clean -df      ★ 목표 리비전의 .gitignore 로 ★ 청소한다
```

③ 을 ② 앞에 두면 **이전 리비전의 무시 규칙으로 청소**하게 되고, 그 리비전에
`.gitignore` 가 없거나 다르면 **데워둔 빌드 캐시가 날아간다.** 실측에서 밟았다 —
`ADR-017` 이 `-x` 를 뺀 이유가 **순서에도 걸려 있었다.**

```text
   실측:  drv.o (무시됨)   → CACHE_SURVIVED   ★ 캐시는 산다 ★
          junk.c (추적 안 됨) → JUNK_REMOVED
          drv.c (추적 변경) → 커밋된 내용으로 되돌아감
```

리비전이 로컬에 없으면 **받아온다** (`ADR-017` 의 [미정] 을 닫는다) — 새 패치는
로컬에 없는 것이 정상이라 실패시키면 본편 시나리오가 아예 안 돈다. 시간이 드는데
**임대가 그 시간을 묶는다**(`not_after`).

계약이 **다른 저장소**를 가리키면 거절한다 — 조용히 틀린 것을 빌드하면 안 된다.

### `runctl capabilities` — 계약을 쓰기 전에 어휘를 읽는다

```console
$ runctl capabilities
agent.reason  nodes: 2
  arch       armv7
  board      SoC-X
  harness    claude
  repo       gerrit.corp/kernel/linux
  tag        board-042

※ nodes 는 총수(존재)다. 지금 비어 있는지는 알려주지 않는다 —
  속성 조합으로 세려면 dry-run 을 쓴다.
```

**`ADR-012` 가 어휘를 창발시켰기 때문에 이것이 필요하다** — capability 이름과 속성이
중앙에 선언되지 않고 enode 광고로만 존재하므로, 읽는 경로가 없으면 **계약을 쓰는
쪽이 문자열을 추측한다.** 계약은 사람이 아니라 에이전트가 쓴다.

**여유는 안 알려준다** — 알려주면 그것을 보고 제출하는데 그 사이 다른 Run 이
가져가 아무것도 보장하지 않는 확인이 되고, `first available` 아래서는 지명도 못 한다.

## 테스트

```bash
eval "$(scripts/testdb.sh)"   # docker 로 Postgres 하나
go test ./...
```

`ENODE_TEST_DATABASE_URL` 이 없으면 DB 테스트는 `t.Skip` 한다.

**단위 테스트는 자기 데이터베이스(`enode_test`)를 쓴다.** 손으로 띄워둔 enode 와
같은 DB 를 공유하면 그 enode 가 계속 광고해서 *"함대에 없다(422)"* 를 기대한
테스트가 **조용히 201 을 받는다** — 실제로 밟았다.
**CI 에는 그 스킵을 잡는 단계가 따로 있다** — 조용히 안 도는 것이 가장 나쁘다.

## 테스트가 곧 명세다

각 테스트는 어느 ADR 을 기계적으로 지키는지 이름에 달고 있다.
`ADR-004`(기계적 판정만)를 우리 개발 자신에게 적용한 것이다.

```bash
go test ./...
```

## 도구

```text
   Go        1.26.6 — ~/sdk/go1.26.6, ~/.local/bin 에 심링크 (.profile 이 이미 PATH 에 둔다)
             apt 의 1.19 는 안 건드렸다. 사용자 로컬 설치라 sudo 가 필요 없다.
   Postgres  S2 부터 (ADR-015 §3)
```

### ★ 버전은 저장소가 강제한다 ★

```text
   go 1.26            이보다 낮은 툴체인은 ★ 빌드를 거절한다 ★ (하한 + 언어 버전)
   toolchain go1.26.6 실제로 쓸 것. GOTOOLCHAIN=auto(기본) 면 ★ 자동으로 받아온다 ★
```

**둘을 같은 버전으로 묶어 슬랙을 없앴다.** 넷이 서로 다른 컴파일러로 짜면
"내 기계에서는 되는데" 가 나온다. **명시적 실패가 조용한 드리프트보다 낫다** —
`ADR-015` 가 `user.email` 이 없으면 그 자리에서 죽기로 한 것과 같은 판단이다.

확인된 동작:

```text
   go1.19  → go: errors parsing go.mod: unknown directive: toolchain   ★ 거절 ★
   go1.26.6 → ok
```

**한 가지 대가** — `GOTOOLCHAIN=auto` 는 네트워크를 전제한다. 프록시 뒤에서
모듈 프록시에 못 닿으면 받아오지 못한다. 그런 환경에서는 `GOTOOLCHAIN=local` 로
두면 `go 1.26` 이 하한만 강제하고 정확한 고정은 풀린다. **사내 배포 때 확인할 것.**

## CI

```text
   test    gofmt -l 이 비어 있는가 · go vet · go test
   cross   ★ ADR-015 가 Go 를 고른 이유를 검증한다 ★
           GOOS=windows · linux/arm(Pi 2) · darwin/arm64 크로스 빌드
           "리눅스 CI 에서 exe 가 나온다" 가 깨지면 윈도우 enode 배포가 무너진다
```

버전은 `go-version-file: go.mod` 로 읽는다 — **버전을 두 곳에 적지 않는다.**

## 정해진 것 — 설계 문서의 미정을 닫은 것

```text
   명령 단계의 셸    ★ argv 배열 ★ (ADR-019 미정 하나를 닫는다)
                     "run": ["make", "-j8", "modules"]
                     윈도우 enode 에 sh 가 없고, 셸 인젝션 표면이 사라진다
   속성 값의 타입    ★ 문자열만 ★ — 숫자를 허용하면 범위 비교로 미끄러지고
                     그게 우리가 두 번 기각한 표현식 언어의 시작이다 (ADR-011)
```
