# `trash` — Code Generation 계획

**유닛** `trash` (버리기와 여유 공간 · 한 줄 순서의 넷째) · **브랜치** `unit/trash` · **기준** `dd27618`
(Functional Design 커밋) · **맡는 조각** 4 (trash · 사람 · SunnyVM) · **병합 조건** 조각 4 가 초록

**이 계획이 이 유닛 Code Generation 의 기준이다.** 여기 없는 것은 짓지 않는다. 설계는
`construction/trash/functional-design/` 의 셋이다. 아래의 **FD** 는 그 셋의 줄임이다 — 「FD 규칙 6.2」는
`business-rules.md` 6.2절, 「FD 흐름 3절」은 `business-logic-model.md` 3절, 「FD 엔티티 4절」은 `domain-entities.md` 4절이다.

- **작성 시각**: 2026-09-26T14:24:01Z 이후 (Functional Design 승인 뒤)
- **입력**: FD 셋 · 유닛 정의 `unit-of-work.md` 4절 · 파일 행렬 1.3 의 ④ 열 · 팩 `scene-gates.md` 조각 4 · `requirements.md` 6절
- **NFR 두 단계** (NFR Requirements · NFR Design — 비기능 요구와 그 설계): 사용자 결정으로 건너뛰었다
  (2026-09-26T14:24:01Z). 그 둘이 맡기로 했던 N1 을 이 계획의 3절이 받는다

---

## 1. 유닛 맥락

**하는 일** — 단계가 남긴 작업 폴더를 rename 한 번으로 trash 에 넣고, 보고 뒤 배경에서 namespace 안에서 지운다.
여유가 `min_free_gb` 아래면 노드가 스스로 graceful drain 을 싣는다. 노드 쪽만 바뀐다 — Mediator 는 그대로다.

**완성하는 스토리** — US-1 (누가 건 drain 인지 · 이 유닛은 소유자와 여유 부족 둘) · US-2 (얼마가 곧 지워질 trash 인지 ·
checkpoint 몫은 checkpoint 유닛) · US-3 (여유가 모자라면 노드 전체가 빠진다). **완료 조건** 1 · 2 · 3 ①.

**새 이름**

```text
   internal/scratch (새 패키지)   Trash · Move · Entries · SessionLockName · HoldSession · SessionLock · Orphans ·
                                 CheckEntry · Size · Measure · Remove · Deleter · Usage
   internal/enode                RunTrashHelper · TrashLauncher · SweepOrphanSessions · DrainSource · DrainStatus ·
                                 StatusBook · NewStatusBook · Worker.AfterReport · Advertiser.Status
                                 (안 내보내는 것) combineDrain · diskDrain · runcOverlaySession.lock
   internal/panel                State.DrainSources · State.DrainFrom · State.Scratch
```

**없애는 이름** — `hasRoom` (`detect.go:387`) 과 arch 키를 가리던 조건문 · 경고 줄 (FD 규칙 6.4).

**경계** — `internal/scratch` 는 표준 라이브러리와 `golang.org/x/sys` 만 쓴다. Mediator(`cmd/mediator`)는 `internal/enode` 와
`internal/scratch` 를 안 가져다 쓴다 (FD 규칙 10절 — 시험 줄 셋을 더한다). `go.mod` 를 안 움직인다 — `x/sys` 는 이미 있다.

**크로스 빌드** — `internal/scratch` 의 잠금 · 측정 · 삭제는 unix 전용이다 (flock · st_dev · st_blocks). `_unix.go` 와
`_other.go` 짝으로 두고, `_other` 는 「지원하지 않음」을 돌려준다 (실행 계획 7절 ④). trash 와 삭제자는 runc-overlay
노드(linux)만 쓰므로 다른 OS 에서 불릴 일이 없다.

---

## 2. 고치는 파일

| 파일 | 새 · 고침 | 행렬 | 무엇 |
|---|---|---|---|
| `internal/scratch/trash.go` | 새 | 있음 | `Trash` · `Move` · `Entries` |
| `internal/scratch/session_unix.go` · `session_other.go` | 새 | 있음 | 세션 잠금 · `Orphans` |
| `internal/scratch/remove_unix.go` · `remove_other.go` | 새 | 있음 | `CheckEntry` · `Measure` · `Remove` |
| `internal/scratch/deleter.go` | 새 | 있음 | `Deleter` · `Usage` |
| `internal/scratch/*_test.go` | 새 | 시험 | 규칙마다 표 시험 · 벤치마크 하나 (3절) |
| `internal/enode/trash_linux.go` · `trash_other.go` | 새 | 있음 (trash-helper 새 파일) | `RunTrashHelper` · `TrashLauncher` · `SweepOrphanSessions` |
| `internal/enode/runc_overlay_linux.go` | 고침 | 있음 | 다섯 자리 · 세션 잠금 · helper cleanup 은 unmount 만 |
| `internal/enode/policy.go` | 고침 | 있음 | `DrainSource` · `DrainStatus` · `combineDrain` · `diskDrain` |
| `internal/enode/advertise.go` | 고침 | 있음 | 광고 주기의 drain 합치기 · `Advertiser.Status` |
| `internal/enode/detect.go` | 고침 | 있음 | arch 키의 여유 조건문과 `hasRoom` 을 지운다 |
| `internal/enode/status.go` | 고침 | 있음 | `Status` 의 새 칸 둘 · `StatusBook` |
| `internal/enode/claim.go` | 고침 | 밖 (FD 흐름 9절) | `Worker.AfterReport` · `afterExit` 가 닫기가 늦었는지 넘긴다 |
| `internal/enode/finalize.go` | 고침 | 밖 (FD 흐름 9절) | `settle` 의 입력 하나 |
| `cmd/enode/main.go` | 고침 | 있음 | `trash-helper` 입구 · 기동 청소 · 삭제자 · `StatusBook` 을 잇는다 |
| `internal/panel/view.go` · `page.go` | 고침 | 있음 | drain 출처 · 풀기 버튼 조건 · trash 칸 |
| `internal/panel/boundary_test.go` | 고침 | 밖 (FD 흐름 9절) | 금지 둘 · 봉인 하나 |
| `packaging/macos/examples/*.yaml` 넷 | 고침 | 있음 | `min_free_gb` 주석 |
| `scripts/finalize-bake/slice-4.sh` | 새 | 있음 (조각 스크립트) | 조각 4 — 사람이 SunnyVM 에서 |
| `internal/enode` 의 시험 | 새 · 고침 | 시험 | `trash_test.go` (새 · 인자와 줄) · `drain_test.go` (고침 · 합치기와 걸고 풀기) · `status_test.go` (새 · StatusBook) · `detectarch_test.go` (고침 · 여유와 무관) · `runc_overlay_linux_test.go` (고침 · 다섯 자리) · `finalize_worker_test.go` (고침 · 닫기가 늦었다) · `runc_overlay_integration_test.go` (고침 · namespace 안의 삭제 · CI 밖) |
| `internal/panel/panel_test.go` | 고침 | 시험 | 출처 · DrainFrom · 버튼 조건 · trash 칸 |

`internal/enode/config.go` · `runc_overlay_other.go` 는 안 고친다 — 새 칸이 없고 세션이 없다 (FD 흐름 9절). 확인만 한다.

---

## 3. NFR 두 단계를 건너뛰어서 이 계획이 받는 것

유닛 정의가 NFR 에 적은 셋 — 보안(trash 삭제 줄)은 FD 규칙 5절이, 성능(보고 전에는 rename 한 번)은 FD 규칙 1 · 2절과
조각 4 ① 이 닫았다. 남는 것은 **N1 — 삭제자가 한 번에 지우는 양과 속도**다. 틀려도 trash 가 늦게 비워져 여유 부족 drain 이
걸리는 안전한 쪽이다. 측정 둘로 확인하고 숫자를 `code-summary.md` 에 적는다 (Step 15).

- `BenchmarkRemove` (`internal/scratch`) — 보통 권한의 트리 (디렉터리 100 x 파일 1,500 = 150,000) 를 `Measure` 와 `Remove` 로
  한 번 지우는 시간. 이 기계의 디스크(스케줄러 `none`)에서. 기본 `go test` 에서는 안 돈다
- 조각 4 ② 에서 SunnyVM 의 스케줄러(`/sys/block/*/queue/scheduler`)와 9 GB upper 를 지우는 시간을 함께 본다 — 스크립트가 출력한다

값을 바꾸지 않는다 — 한 번에 항목 하나 · idle IO · CPU 19 · 10분 간격. idle IO 는 BFQ 에서만 먹는다 (FD 규칙 4.2).

---

## 4. 이 계획이 정한 것 — FD 에 적히지 않은 자리

**① `StatusBook` 을 누가 만드나.** `cmd/enode/main.go` 가 만들어 `Advertiser.Status` 와 삭제자의 `Changed` 에 준다. `Advertiser.Status`
가 nil 이면 Advertiser 가 첫 광고에서 자기 것을 만든다 — 설정 경로가 있을 때만 (오늘의 `writeStatus` 와 같은 조건). 그래서 오늘의
Advertiser 시험이 그대로 돈다.

**② 삭제자와 trash 를 누가 만드나.** `enode.SweepOrphanSessions(scratch, log)` 와 `enode.TrashLauncher(log)` 를 `trash_linux.go` 에 둔다.
`RuncOverlayRuntime` 은 생성 때 `scratch.Trash{Dir: <binding.Scratch>/trash}` 를 든다. `cmd/enode/main.go` 는 runc-overlay 일 때만 청소 ·
삭제자 · `worker.AfterReport` 를 잇는다.

**③ 닫기가 늦었는가.** `settleIn` 에 `closedLate bool` 한 칸 — `afterExit` 가 닫기가 끝난 시각이 `spec.Deadline` 뒤이고 임대가 살아
있으면 참으로 넘긴다. `settle` 은 Finalize 가 ok 인데 `closedLate` 면 timeout 과 같은 문구를 낸다. 단계 로그 끝의 줄도 같다 —
`logTail` 의 `finalizeTimeout` 인자가 이 경우를 받는다.

**④ helper cleanup 의 최종 방어 경로.** helper 는 stdin 이 끝나도(부모가 죽었다) cleanup 을 부른다. 이제 cleanup 은 unmount 만 하므로
작업 폴더는 scratch 에 남고 다음 기동 청소가 거둔다 (FD 규칙 3절). 따로 할 일이 없다.

**⑤ `Orphans` 의 「1시간」은 인자다.** 시험이 시각을 준다. `SweepOrphanSessions` 가 `time.Now()` 를 넘긴다.

**⑥ trash-helper 의 argv.** `runcOverlayHelperArgv` 옆에 `trashHelperArgv(helper, trash, entry)` — `--mount` 가 없다. 시험이 모양을 본다.

**⑦ 제어판 문구.** 오늘 화면의 한국어를 따른다 (FD 규칙 8절). 출처 이름 — 「소유자 정책」 · 「여유 부족」. 누가 풀 수 있나 —
「여기서 풀 수 있다」 · 「저절로 풀린다 — trash 가 비거나 디스크가 늘면」. 남은 출처 문구 — 「풀어도 여유 부족 drain 이 남아
노드는 빠져 있다」. trash 칸 크기 표기 — GiB 한 자리.

**⑧ 조각 4 스크립트의 대상.** SunnyVM 에서 이미 도는 runc-overlay 노드를 쓴다 — 스크립트가 노드를 띄우지 않는다. 받는 것 — `M` · `T` ·
`NODE_CONFIG` (그 노드의 설정 경로 · 상태 파일과 scratch 를 거기서 찾는다) · `WS` (그 노드의 워크스페이스 · 계약의 `ws`). 재시작은
사람이 한다 — 스크립트가 멈추고 할 일을 출력한다 (`enodectl stop` · `enodectl start`).

---

## 5. 단계 — 열여덟

### Step 1 — 기준선

- [x] `unit/trash` 가 `dd27618` 위에 있다
- [x] CI 의 awk 로 `internal/enode` · `internal/panel` · `cmd/enode` 커버리지를 적는다 · `golangci-lint` 경고 수를 적는다 (39)

### Step 2 — `internal/scratch` 의 trash (FD 엔티티 1절 · FD 규칙 1절)

- [x] `trash.go` — `Trash.Move` (trash 가 없으면 0700 으로 만든다 · 마지막 조각을 이름으로 · 겹치면 `-1` · `-2` … · rename 한 번 ·
      실패하면 지우지 않고 오류) · `Entries` (바로 아래 이름 · 이름순)
- [x] 패키지 문서 주석 — 무엇을 담고 무엇을 모르나 (lower · 합치기 · 굽기 · Mediator)

### Step 3 — 세션 잠금과 남은 폴더 (FD 엔티티 2절 · FD 규칙 3절)

- [x] `session_unix.go` — `HoldSession` (잠금 파일 0600 · `flock(LOCK_EX|LOCK_NB)`) · `Release` · `Orphans(scratch, prefix, now)`
      (잠금 파일이 있고 쥘 수 있으면 남은 것 · 쥔 뒤 놓는다 · 없으면 1시간 규칙 · 못 쥐면 살아 있다)
- [x] `session_other.go` — `HoldSession` 은 할 일이 없는 잠금을 돌려주고 `Orphans` 는 빈 목록 (다른 OS 는 runc-overlay 가 없다)

### Step 4 — 삭제와 측정 (FD 엔티티 3절 · FD 규칙 5절)

- [x] `remove_unix.go` — `CheckEntry` · `Measure` · `Remove`. 걷기는 `Lstat` · 디렉터리는 들어가기 전에 0700 · trash 와 st_dev 가 다르면
      안 들어가고 `Left` 에 · symlink 는 링크만. 블록 수 x 512 의 합 · 항목 수(디렉터리 포함)
- [x] `remove_other.go` — 「지원하지 않음」

### Step 5 — 배경 삭제자 (FD 엔티티 4절 · FD 규칙 4절 · FD 흐름 3절)

- [x] `deleter.go` — `Run` (곧바로 한 번 · Kick · 항목이 남으면 `Every`) · `Kick` (크기 1 채널) · `Usage` · 결과마다 로그 (FD 규칙 4.3 의
      영어 문구) · helper 를 못 띄운 원인은 바뀔 때만 적는다 · `Changed` 를 부르는 때 셋 (목록 · 측정 · 지움 시작과 끝)
- [x] `Launch` 를 바꿔 끼우는 시험 자리는 칸 그대로다 (`Deleter.Launch`)

### Step 6 — `internal/scratch` 시험

- [x] `Move` — 옮긴 뒤 inode 가 같다(rename) · 이름 겹침 · trash 를 만든다 · 없는 경로면 오류이고 지우지 않는다
- [x] `Orphans` — 쥔 잠금은 안 나온다(다른 프로세스 대신 같은 프로세스의 다른 fd 로 쥔다 — flock 은 fd 마다다) · 풀린 잠금은 나온다 ·
      잠금 파일 없고 1시간 전이면 나온다 · 1시간 안이면 `skipped` · prefix 밖은 안 본다
- [x] `CheckEntry` — 거절 넷(빈 이름 · `.` · `..` · `/` 든 이름) · trash 가 symlink · 파일
- [x] `Measure` · `Remove` — symlink 가 가리키는 trash 밖의 파일이 남는다 · 권한 000 디렉터리가 지워진다 · 블록 합과 항목 수 ·
      다른 st_dev 는 시험이 만들 수 없어 판정 함수를 인자로 떼어 가짜로 본다 (`Remove` 의 안쪽 걷기가 받는다)
- [x] `Deleter` — 가짜 `Launch` 로: 기동 첫 회 · Kick 합치기 · 실패한 항목이 남고 다음 깸에 다시 · 항목이 없으면 `Every` 에 안 깬다 ·
      measured 가 `Usage` 에 들어간다 · `Deleting` 이 켜지고 꺼진다 · ctx 로 멈춘다
- [x] `BenchmarkRemove` (3절)

### Step 7 — trash-helper (FD 엔티티 5절 · FD 흐름 3절 · 4절 ⑥)

- [x] `trash_linux.go` — `RunTrashHelper(args, out, errOut)` (ioprio idle · setpriority 19 · CheckEntry · Measure 줄 · Remove 줄 · exit) ·
      `trashHelperArgv` · `TrashLauncher(log)` (unshare 로 띄우고 · 프로세스 그룹 · Pdeathsig · stdout 줄 읽기 · ctx 에 그룹 SIGKILL) ·
      `SweepOrphanSessions(scratch, log)`
- [x] `trash_other.go` — `RunTrashHelper` 은 1 · 나머지는 할 일이 없다
- [x] `cmd/enode/main.go` — `enode trash-helper <trash> <entry>` 입구 (runtime-helper 옆 · 플래그 앞)
- [x] `trash_test.go` — 인자 모자람 · 이름 거절 줄 · 보통 권한의 trash 에서 줄 셋과 exit 0 (namespace 없이 `RunTrashHelper` 을 부른다) ·
      argv 모양 (`--mount` 가 없다)

### Step 8 — 다섯 자리와 세션 잠금 (FD 규칙 1절 · FD 흐름 1절)

- [x] `RuncOverlayRuntime` 이 trash 를 든다. `Open` — `MkdirTemp` 바로 뒤 `HoldSession` · 실패 갈래 넷은 `Move` 한 뒤 `Release`
- [x] 세션 `Close` — helper 를 닫은 뒤 `Move` · `Release`. abort 뒤도 같다 (`RemoveAll` 두 자리가 `Move` 가 된다). `Move` 오류는
      `move runtime session to trash: %w`
- [x] `abort` — helper 를 죽인 뒤 `Move` · `Release`
- [x] helper `cleanup` — unmount 만. `RemoveAll(runRoot)` 을 지운다
- [x] 준비도 smoke — 코드는 그대로 (`Close` 를 지난다). 닫은 뒤 원래 자리에 폴더가 없는지 보는 확인이 그대로 초록인지
- [x] `runc_overlay_linux_test.go` — 가짜 helper 로: 닫은 뒤 trash 에 작업 폴더가 있고 원래 자리에 없다 · abort 뒤도 · Open 실패(가짜 helper 가
      opened 에 오류) 뒤도 · 잠금이 풀렸다 (다른 fd 로 쥘 수 있다)

### Step 9 — 닫기를 Finalize 예산 안으로 (FD 규칙 2절 · 4절 ③)

- [x] `finalize.go` — `settleIn.closedLate` · `settle` 이 받는다
- [x] `claim.go` — `afterExit` 가 닫기 뒤 시각을 `spec.Deadline` 과 비교해 넘기고 `logTail` 에도 넘긴다
- [x] `finalize_test.go` 의 `TestSettle` 에 줄 둘 (닫기가 늦었다 · 늦었지만 임대가 끝났다) · `finalize_worker_test.go` 에 가짜 세션 하나
      (Close 가 마감을 넘긴다 → `finalize_timeout` · 단계 로그 끝 줄)

### Step 10 — drain 합치기와 여유 (FD 엔티티 6절 · FD 규칙 6절)

- [x] `policy.go` — `DrainSource` · `DrainStatus` · `combineDrain` · `diskDrain` (Detail 문구 · GB 는 2^30 의 몫)
- [x] `advertise.go` — 광고 주기: 소유자 출처 · 여유 측정 (워크스페이스가 있을 때만 · 측정하지 못하면 넉넉함과 경고 한 번) · `diskHeld` ·
      걸기와 풀기가 바뀌면 로그 한 줄 · `ad.Policy` 에 합친 값 · `Status.SetDrain`
- [x] `detect.go` — arch 조건문 · `hasRoom` · 경고 줄을 지운다
- [x] `drain_test.go` — FD 규칙 6.2 의 네 줄 · 6.3 의 다섯 줄 · 워크스페이스 없음 · 여유를 측정하지 못한 경우 · 광고 본문의 `policy.drain` 이
      합친 값이다 (시험 서버) · 응답의 drain 이 Worker 에 닿는다 (오늘 경로 그대로)
- [x] `detectarch_test.go` — 여유가 0 이어도 툴체인이 있으면 arch 키가 실린다 (hasRoom 을 기대던 시험을 고친다)

### Step 11 — 상태 파일 (FD 엔티티 7절 · FD 규칙 7절 · 4절 ①)

- [x] `status.go` — `Status.Drain` · `Status.Scratch` · `StatusBook` (`NewStatusBook(configPath, log)` · `SetCaps` · `SetDrain` · `SetScratch` ·
      같은 값이면 안 씀 · 쓰기 실패 경고는 원인이 바뀔 때만)
- [x] `advertise.go` — `writeStatus` 가 `Status.SetCaps` 가 된다 · `Advertiser.Status` 가 nil 이면 스스로 만든다 (설정 경로가 있을 때만)
- [x] `status_test.go` — 같은 값이면 파일이 안 바뀐다(mtime) · 한 칸만 바뀌어도 쓴다 · 두 고루틴이 동시에 불러도 칸을 안 덮는다 ·
      옛 상태 파일(칸 둘)이 읽힌다

### Step 12 — 기동을 잇는다 (FD 흐름 2절 · 4절 ②)

- [x] `claim.go` — `Worker.AfterReport func()` · `report` 가 끝나면 부른다 (닿았든 포기했든)
- [x] `cmd/enode/main.go` — `StatusBook` 을 만들어 `adv.Status` 에 · runc-overlay 면 `SweepOrphanSessions` · `scratch.Deleter` 를 만들어
      `go Run` (WaitGroup 에 하나 더) · `worker.AfterReport = deleter.Kick` · 광고는 기다리지 않는다
- [x] `cmd/enode/main_test.go` — `trash-helper` 입구가 플래그 앞에서 갈리는지 (runtime-helper 의 시험이 있으면 그 옆)

### Step 13 — 제어판 (FD 엔티티 8절 · FD 규칙 8절 · 4절 ⑦)

- [x] `view.go` — 데몬이 돌고 상태 파일에 drain 이 있으면 effective · 출처 · `DrainFrom: status`. 아니면 정책 파일과 소유자 출처 하나 ·
      `DrainFrom: policy`. `Scratch` 는 상태 파일에서
- [x] `page.go` — 출처 줄 · 풀기 버튼은 소유자 출처가 있을 때만 · 남은 출처 문구 · trash 줄 · 기존 「소유자가 풀어야 후보로 돌아온다」 문구를
      출처에 맞게 바꾼다
- [x] `panel_test.go` — 네 경우 (소유자만 · 여유 부족만 · 둘 다 · 데몬이 안 돎) 의 State 와 페이지 문자열 · trash 칸

### Step 14 — 예시 주석 · 경계 시험 (FD 규칙 9 · 10절)

- [x] `packaging/macos/examples` 넷의 `min_free_gb` 주석을 FD 규칙 9절의 문장으로
- [x] `internal/panel/boundary_test.go` — `{"cmd/mediator", "internal/enode"}` · `{"cmd/mediator", "internal/scratch"}` · 봉인에 `internal/scratch`
      (허용 목록 `golang.org/x/sys`)

### Step 15 — 측정과 integration 시험 (3절)

- [x] `BenchmarkRemove` 를 한 번 돌리고 숫자를 적는다 · 이 기계의 스케줄러를 적는다
- [x] `runc_overlay_integration_test.go` — 실제 세션이 subordinate uid 소유 항목과 권한 000 `work/work` 를 남기게 하고 닫은 뒤 trash 에 있는지 ·
      `TrashLauncher` 로 지워지는지 · whiteout 도 · `go vet -tags integration` 로 컴파일만 확인 (CI 밖 · 사람이 SunnyVM 에서)

### Step 16 — 조각 4 스크립트 (FD 흐름 7절 · 4절 ⑧)

- [x] `scripts/finalize-bake/slice-4.sh` — `M` · `T` · `NODE_CONFIG` · `WS` 를 받는다. FD 흐름 7절의 ① ~ ⑥ 을 순서대로 — 큰 upper 단계와 작은 upper
      단계의 두 시각 차 · 보고 직후와 잠시 뒤의 trash 목록과 상태 파일의 scratch 칸 · 권한 000 과 subordinate uid 항목 · 재시작은 사람에게 ·
      `min_free_gb` 를 올려 draining 과 arch 키 · 스케줄러 출력. 사람이 보는 조각이라 판정하지 않고 멈춰 무엇을 볼지 출력한다
- [x] `bash -n` · 사내 이름 0 · 출력 영어 · 주석 한국어

### Step 17 — 코드 검사와 조각

- [x] `gofmt -l .` · `go vet ./...` · `go vet -tags integration ./internal/enode/` · `go build ./...`
- [x] `go test ./... -count=1` — 실패 0 · 스킵 0 · 패키지마다 80% 이상 (`internal/scratch` 포함)
- [x] 크로스 빌드 셋 · `enodectl.exe` 심볼 상한 · U+2605 0 · glyphscan · 린트 수가 Step 1 보다 늘지 않는다
- [x] **조각 0** — build · vet · test · 라우트 19 (안 바뀐다)
- [x] **조각 4** — 사람의 조각이다. 스크립트만 준비한다. SunnyVM 이 꺼져 있으면 보류로 적는다 — 보류는 통과가 아니고 병합 지점도 아니다
      (2026-09-26 확인 때 꺼져 있었다)
- [x] `git status` 로 2절 밖의 파일이 없는지 · `cmd/enodectl/probe.lock` 이 바뀌면 되돌린다

### Step 18 — 요약 · 상태 · 감사 · 커밋

- [x] `construction/trash/code/code-summary.md` — 파일 · 규칙의 자리 · 4절의 결정 · 계획과 다른 자리 · 코드 검사 숫자 · 측정 숫자 · 조각 4 의
      돌리는 법 · 넘기는 것 · 정본 되돌림 (FD 흐름 11절)
- [x] 표기 검사 · 사용자가 싫어한 말투 · 사내 이름
- [x] 이 계획의 체크박스 · `aidlc-state.md` 의 U4 절 · `audit.md`
- [x] 한 커밋 — 코드 · 시험 · 스크립트 · 예시 · 이 계획 · code-summary · 상태 · 감사. 승인 뒤에 넣는다 (CONVENTIONS 3.3)

---

## 6. 이 단계가 하지 않는 것

```text
   bake 출처의 drain · 후보 잠금                         lower-state 유닛
   Keep.Upper 의 대기 자리 · merge 의 trash                bake 유닛
   spool · checkpoint · Usage 의 spool 칸                 checkpoint 유닛
   env check 의 scratch filesystem 확인                   lower-state 유닛
   Mediator                                             바뀌지 않는다
   조각 4 를 돌리는 일                                    사람의 조각이다.  SunnyVM 이 켜진 뒤 사용자가 돈다
   정본(enode-design) 되돌림                              진행자가 올린다 (FD 흐름 11절)
   PR 과 병합                                            조각 4 가 초록인 뒤.  올리기 전에 묻는다
```

---

## 7. 확장 준수 — Code Generation

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 trash 삭제 줄은 FD 규칙 5절이 닫고 Step 6 · 15 가 시험으로 확인한다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 한 줄 |
