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
| S8 | blob 별 모양 (2단계 Run) |  |
| S9 | agent 어댑터 (하네스 봉투 + `$OUT` 수확) |  |
| S10 | 워크스페이스 · 스키마 검증 · `dry-run` · `capabilities` |  |

**S4 가 최대 위험이다.** `ADR-016`(하트비트가 임대를 나른다)은 설계만 있고
한 번도 안 돌았으며, 미정 값 셋이 전부 거기서 정해진다.

## 배치

```
   internal/contract   계약 · 광고 타입. ★ 넷의 공통 어휘 ★
   internal/match      요구 → 노드. ★ 순수 함수 ★ (ADR-014 결정 3)
   internal/config     ADR-015 §4 의 우선순위. ★ 사용자 경로가 /etc 를 이긴다 ★
   internal/store      PostgreSQL. ★ 매칭 로직은 여기 없다 ★
   internal/record     ★ Run Record — DB 가 아니라 파일시스템 ★ 봉인 · tar
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

## 테스트

```bash
eval "$(scripts/testdb.sh)"   # docker 로 Postgres 하나
go test ./...
```

`ENODE_TEST_DATABASE_URL` 이 없으면 DB 테스트는 `t.Skip` 한다.
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
