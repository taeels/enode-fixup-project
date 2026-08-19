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
| S4 | ★ 두 연결 ★ 하트비트 + `claim` 롱폴 | `O6` — **가장 안 검증된 것** |
| S5 | 명령 단계 실행 + `result` + 상태기계 |  |
| S6 | Record 봉인 + `GET record` (tar) | `O1` |
| S7 | `runctl` — 제출 · 상태 · Record 읽기 | `O7` |
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
   internal/api        HTTP 표면. 라우팅은 표준 라이브러리만 (Go 1.22+ ServeMux)
   internal/enode      신원 · 잠금(unix/windows) · 능력 탐지 · 광고 루프
   cmd/mediator  cmd/enode        (cmd/runctl 는 아직)
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
