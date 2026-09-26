# `trash` — 코드 요약

**유닛** `trash` (버리기와 여유 공간 · 한 줄 순서의 넷째) · **브랜치** `unit/trash` · **기준** `dd27618`
(Functional Design 커밋) · **계획** `construction/plans/trash-code-generation-plan.md` (단계 열여덟)

단계가 남긴 작업 폴더를 지우지 않고 rename 한 번으로 trash 에 넣는다. 지우는 것은 보고 뒤 배경 삭제자이고,
namespace 안의 helper 가 지운다. 워크스페이스의 여유가 `min_free_gb` 아래면 노드가 스스로 graceful drain 을
싣는다. Mediator 는 한 줄도 안 바뀌었다.

아래의 **FD** 는 이 유닛의 Functional Design 산출물 셋(`construction/trash/functional-design/`)의 줄임이다 —
「FD 규칙 6.2」는 `business-rules.md` 6.2절, 「FD 흐름 7절」은 `business-logic-model.md` 7절이다. 「계획 4절 ③」은
계획 파일 4절의 셋째 결정이다. **조각 4** 는 이 유닛의 병합 조건인 장면 확인(trash · 사람이 SunnyVM 에서)이다.

---

## 1. 파일

| 파일 | 새 · 고침 | 무엇 |
|---|---|---|
| `internal/scratch/trash.go` | 새 (101 줄) | 패키지 문서 · `TrashName` · `Trash` · `TrashIn` · `Move` · `Entries` · `CheckEntry` |
| `internal/scratch/session_unix.go` · `session_other.go` | 새 (116 · 22 줄) | `SessionLockName` · `SessionLock` · `HoldSession` · `Release` · `Orphans` (1시간 규칙). 다른 OS 는 할 일이 없는 잠금과 빈 목록 |
| `internal/scratch/remove_unix.go` · `remove_other.go` | 새 (160 · 14 줄) | `Measure` · `Remove` · 판정 함수를 받는 안쪽 `remove` · 함께 쓰는 걷기 `walker`. 다른 OS 는 「지원하지 않음」 |
| `internal/scratch/deleter.go` | 새 (258 줄) | `Size` · `Usage` · `LaunchError` · `LeftError` · `DefaultEvery` · `Deleter` (`Run` · `Kick` · `Usage`) |
| `internal/enode/trash_linux.go` · `trash_other.go` | 새 (220 · 32 줄) | `RunTrashHelper` · `trashHelperArgv` · `TrashLauncher` · `SweepOrphanSessions` · 우선순위 내리기 |
| `internal/enode/runc_overlay_linux.go` | 고침 (+73 −36) | 다섯 자리 — `Open` 의 잠금과 실패 갈래(`discard`) · 세션 `Close` · `abort` · `release` · helper `cleanup` 은 unmount 만 · smoke 의 확인은 그대로 |
| `internal/enode/policy.go` | 고침 (+72) | `DrainOwner` · `DrainDisk` · `DrainSource` · `DrainStatus` · `OwnerDrain` · `combineDrain` · `diskDrain` |
| `internal/enode/advertise.go` | 고침 (+81 −31) | `Advertiser.Local` · `Advertiser.Status` · `drain` · `diskSource` · `status`. `writeStatus` 와 그 칸 둘을 지웠다 |
| `internal/enode/status.go` | 고침 (+102 −7) | `Status.Drain` · `Status.Scratch` · `WriteStatus(configPath, Status)` · `StatusBook` (`NewStatusBook` · `SetCaps` · `SetDrain` · `SetScratch`) |
| `internal/enode/detect.go` | 고침 (+15 −33) | arch 키의 여유 조건문 · `hasRoom` · 경고 줄을 지웠다 |
| `internal/enode/claim.go` | 고침 (행렬 밖) | `Worker.AfterReport` · `report` 가 끝나면 부른다 · `afterExit` 가 닫기가 늦었는지를 settle 과 로그 끝 줄에 넘긴다 |
| `internal/enode/finalize.go` | 고침 (행렬 밖) | `settleIn.closedLate` · `settle` 이 받는다 |
| `internal/enode/runtime.go` | 고침 (행렬 밖 · 주석만) | `Keep` 과 `StepSession.Close` 의 주석 — 닫기가 Finalize 예산 안이다 |
| `internal/enode/config.go` · `setup.go` | 고침 (행렬 밖 · 문구만) | `MinFreeGB` 의 주석 · `enode setup --min-free-gb` 의 도움말 (4절 ⑧) |
| `cmd/enode/main.go` | 고침 (+35 −1) | `trash-helper` 입구 · `StatusBook` · 기동 청소 · 삭제자 · `AfterReport` |
| `internal/panel/view.go` · `page.go` | 고침 | `State.DrainSources` · `DrainFrom` · `Scratch` · `drainView` · 출처 줄과 풀기 버튼 조건 · trash 줄 |
| `packaging/macos/examples/*.yaml` 넷 | 고침 | `min_free_gb` 주석 (FD 규칙 9절의 두 줄) |
| `scripts/finalize-bake/slice-4.sh` | 새 (273 줄) | 조각 4 — 사람이 SunnyVM 에서 돈다 |
| 시험 | 새 여섯 · 고침 열 | 새 `internal/scratch/*_test.go` 넷 · `trash_test.go` · `status_test.go`. 고침 `drain_test.go` · `enode_test.go` · `finalize_test.go` · `finalize_worker_test.go` · `runc_overlay_linux_test.go` · `runc_overlay_integration_test.go` (`integration` 태그 · CI 밖) · `cmd/enode/main_test.go` · `internal/panel/panel_test.go` · `boundary_test.go` |

`internal/enode/runc_overlay_other.go` 는 안 고쳤다 — 세션이 없다. `config.go` 에 새 칸은 없다 (주석만).
`internal/store` · `internal/api` · `internal/contract` · `cmd/mediator` 는 안 고쳤다.

**없앤 이름** — `hasRoom` · `Advertiser.writeStatus` · `lastStatusAt` · `statusWarn` · 경고 줄
`not enough free disk; dropping build capability from the advertisement`.

---

## 2. 규칙이 어디에 있나

| 규칙 (FD) | 자리 |
|---|---|
| 다섯 자리 · 옮기기가 실패하면 안 지운다 · 이름 겹침 (규칙 1절) | `RuncOverlayRuntime.discard` · 세션 `release` · `Close` · `abort` · helper `cleanup` · `Trash.Move` |
| 잠금은 rename 뒤에 놓는다 (규칙 1절) | `discard` · `release` |
| 닫기는 Finalize 예산 안 · 마감으로 끊지 않는다 (규칙 2절) | `Worker.afterExit` 의 `closedLate` · `settle` · `logTail` 의 인자 |
| 기동 청소 (규칙 3절) | `scratch.Orphans` · `enode.SweepOrphanSessions` · `cmd/enode/main.go` |
| 삭제자의 때 · 한 번에 무엇을 · 결과마다 (규칙 4.1 ~ 4.3) | `Deleter.Run` · `pass` · `report` · `Worker.AfterReport` |
| 양을 측정하는 법 (규칙 4.4) | `scratch.Measure` · `Deleter.update` · `RunTrashHelper` 의 `measured` 줄 |
| 삭제의 경계 (규칙 5절) | `CheckEntry` · `walker.remove` · `walker.children` · `openTrash` |
| 여유 부족 drain — 거는 노드 · 걸고 푸는 선 · 합치기 (규칙 6.1 ~ 6.3) | `Advertiser.diskSource` · `diskDrain` · `combineDrain` · `OwnerDrain` |
| arch 키 (규칙 6.4) | `cheapAttrs` |
| 상태 파일 (규칙 7절) | `StatusBook` |
| 제어판 (규칙 8절) | `drainView` · `page.go` 의 drain 칸 · `drainName` · `drainLift` |
| 예시 넷 (규칙 9절) · 경계 시험 (규칙 10절) | `packaging/macos/examples` · `boundary_test.go` |

---

## 3. 계획 4절의 결정 여덟 — 지은 모양

```text
   ①  StatusBook 을 누가     cmd/enode/main.go 가 만들어 Advertiser.Status 와 삭제자의 Changed 에 준다.
                             Advertiser.Status 가 nil 이면 첫 광고에서 스스로 만든다 (설정 경로가 있을 때만)
   ②  삭제자와 trash        RuncOverlayRuntime 이 scratch.TrashIn(binding.Scratch) 를 든다.  main 은 runc-overlay
                             일 때만 기동 청소 · 삭제자 · AfterReport 를 잇는다
   ③  닫기가 늦었나          afterExit — 닫기가 끝난 시각이 마감 뒤이고 임대가 살아 있으면 참.  settle 은 Finalize 가
                             오류 없이 끝났을 때만 이것을 timeout 으로 적는다 (Finalize 의 다른 오류가 더 자세하다)
   ④  helper 의 최종 방어    cleanup 이 unmount 만 한다.  폴더는 scratch 에 남고 다음 기동 청소가 거둔다
   ⑤  Orphans 의 1시간       now 가 인자다.  SweepOrphanSessions 가 time.Now() 를 넘긴다
   ⑥  trash-helper 의 argv   trashHelperArgv(helper, trash, entry).  --mount 가 없다.  시험이 모양을 본다
   ⑦  제어판 문구            소유자 정책 · 여유 부족 · 「여기서 풀 수 있다」 · 「저절로 풀린다 — trash 가 비거나
                             디스크가 늘면」 · 「풀어도 여유 부족 drain 이 남아 노드는 빠져 있다」 · trash 는 GiB 한 자리
   ⑧  조각 4 의 대상         이미 도는 runc-overlay 노드.  M · T · NODE_CONFIG · WS 를 받는다.  재시작은 사람이 한다
```

---

## 4. 계획과 다르게 된 자리

**① `CheckEntry` 는 OS 와 무관한 `trash.go` 에 있다.** 계획 2절은 `remove_unix.go` 였다. 이름 검사와 lstat 은 어느
OS 에서나 같아 두 벌로 둘 까닭이 없다.

**② `TrashLauncher` 는 로그 대신 trash 를 받는다** (`TrashLauncher(trash)`). 계획 4절 ② 는 `TrashLauncher(log)` 였다.
Launch 는 항목 이름만 받으므로 trash 의 자리를 알아야 하고, 결과마다의 로그는 삭제자가 적는다.

**③ 새 이름이 몇 더 있다** — `scratch.TrashIn` · `TrashName` (trash 의 자리 규칙을 한 곳에) · `scratch.LaunchError` ·
`LeftError` (FD 규칙 4.3 의 표에서 결과의 종류를 나누는 자리) · `enode.OwnerDrain` · `DrainOwner` · `DrainDisk`
(제어판이 데몬이 멈췄을 때 정책 파일에서 소유자 출처를 그린다) · `SweepOrphanSessions` 가 쓰는 상수 `runcSessionPrefix`.

**④ helper 를 못 띄운 것을 둘로 알아본다.** `cmd.Start` 가 실패했을 때와, helper 가 줄 하나 없이 끝났을 때다. 뒤의 것은
unshare 가 uid 매핑을 못 한 경우다 (이 기계가 그렇다 — `newuidmap: write to uid_map failed`). 둘 다 항목의 잘못이
아니므로 `LaunchError` 다.

**⑤ 우선순위는 프로세스 그룹에 건다.** FD 엔티티 5절은 「ioprio_set · setpriority」까지였다. linux 에서 두 값은
스레드마다이고 Go 런타임은 스레드가 여럿이라 프로세스 하나에 걸면 일부 스레드만 내려간다. launcher 가 Setpgid 로
새 그룹을 만들므로 그룹에는 unshare 와 helper 만 있다.

**⑥ 삭제는 dirfd 에 기대어 내려간다** (`openat` · `O_NOFOLLOW` · `fstatat` · `unlinkat`). FD 는 「lstat 으로만 본다」였다.
보고 걷는 사이에 경로가 바뀌어도 symlink 로 내려가지 않는다. 남는 창 하나 — 권한을 푸는 `fchmodat` 은 마지막 조각의
symlink 를 따를 수 있다. 그 사이에 바꿔 끼울 프로세스가 없다 (단계의 프로세스는 닫기에서 다 죽었고 trash 는 노드만
쓴다). 코드 주석에 적었다.

**⑦ `Changed` 를 부르는 때** — 깸의 처음(목록 · 지우는 중 켜짐) · 측정 줄이 올 때 · 항목 하나가 끝날 때 · 깸의 끝(다시
읽은 목록 · 지우는 중 꺼짐)이다. trash 가 비어 있으면 처음 한 번만 부른다. 계획 Step 5 의 「목록 · 측정 · 지움 시작과
끝」과 같은 뜻이고, 항목마다의 시작에는 부르지 않는다 (이미 켜져 있다).

**⑧ 옛 뜻을 적은 자리 둘을 고쳤다.** `config.go` 의 `MinFreeGB` 주석과 `enode setup --min-free-gb` 의 도움말
(「stop advertising build capacity below this」)이 이제 거짓이었다. 계획은 `config.go` 를 안 고친다고 했다 — 새 칸이 없다는
뜻이었고 칸은 그대로다. `setup.go` 는 계획 2절의 표에 없던 파일이다.

**⑨ 여유가 적어도 arch 키가 남는 시험은 `enode_test.go` 에 있다.** 계획은 `detectarch_test.go` 였다. `hasRoom` 에 기대던
`TestDetectDropsBuildWhenDiskLow` 가 거기 있어 그 자리에서 `TestDetectKeepsBuildWhenDiskLow` 로 바꿨다.

**⑩ 로그 줄 하나를 더했다** — trash 를 못 읽을 때 `cannot list trash; trash is not emptied` (원인이 바뀔 때만). FD 규칙 4.3
의 표에 그 경우가 없었다.

**⑪ `Orphans` 는 잠금 파일을 못 열면 살아 있는 쪽으로 친다.** 틀리면 도는 단계의 폴더를 옮기게 되고, 반대로 틀리면 다음
기동이 다시 본다. `skipped` 는 「잠금 파일이 없고 1시간 안」 하나만 담는다.

---

## 5. 코드 검사 (`unit-of-work.md` 0절 · CI 와 같은 명령)

| 검사 | 결과 |
|---|---|
| `gofmt -l .` | 빈 출력 |
| `go vet ./...` · `go vet -tags integration ./internal/enode/` · `go build ./...` | exit 0 |
| `go test ./... -count=1` (`-coverpkg=./...` · `-json` · 시험 DB 로) | 통과 2,088 · 실패 0 · 스킵 0 (Step 1 기준선 2,028) |
| 패키지마다 커버리지 (CI 의 awk 그대로) | 스물한 패키지 전부 80% 이상 · 전체 86.4% (기준선 86.0%). 새 `internal/scratch` 93.5% · `internal/enode` 82.7% -> 83.6% (3,140/3,755) · `internal/panel` 88.5% -> 89.1% · `cmd/enode` 83.0% -> **80.4%** (193/240 · 하한에 가깝다 — 아래) |
| 새 함수 | `internal/scratch` 의 함수 서른하나 중 스물넷 100% · `Move` 93.3% · `pass` 88.6% · `setNames` 85.7% · `openTrash` 83.3% · `walker.measure` 72.7% · `walker.remove` 88.9% · `walker.children` 66.7% (못 여는 디렉터리 같은 오류 갈래). `trash_linux.go` — `RunTrashHelper` 91.3% · `runTrashHelper` 97.1% · 나머지 100%. `StatusBook` 전부 100% · `diskDrain` · `combineDrain` · `OwnerDrain` · `settle` · `afterExit` · `drainView` · `discard` · `release` · `abort` 100% · `diskSource` 95.8% · 세션 `Close` 66.7% (helper 가 안 끝나는 갈래) |
| 크로스 빌드 셋 | windows/amd64 · linux/arm (GOARM=7) · darwin/arm64 exit 0 |
| `enodectl.exe` 심볼 | crypto/tls 1 · net/http 6 (상한 10 · 50) |
| U+2605 · `glyphscan` | 0 · 「144 files scanned, no decorative glyph in any string literal」 |
| `golangci-lint` v2.13.2 | 38 건 — 기준선 39 에서 하나 줄었다 (`status.go` 의 S1016 — `WriteStatus` 가 `Status` 를 받게 되어 사라졌다). 이 유닛이 더한 경고 0 (errcheck 일곱을 고쳤다) |
| **조각 0** (기동이 안 깨졌다) | build · vet · test 통과 · `internal/api/api.go` 의 `mux.HandleFunc` 19 (기준 `dd27618` 그대로) |
| `cmd/enodectl/probe.lock` | 측정이 바꿔 되돌렸다 (Reverse Engineering 이 적은 알려진 부채) |

**`cmd/enode` 가 80.4% 다.** `main.go` 의 runc-overlay 갈래(기동 청소 · 삭제자 · 넷째 고루틴 · `scratchDir`)는 준비된 실행
환경이 있어야 닿는다 — 기본 `go test` 에는 없다. 새 문장 열여섯 중 아홉이 거기서 닿지 않는다. 하한은 넘지만 여유가 0.4 점이라 뒤
유닛(bake · checkpoint)이 `main.go` 에 갈래를 더하면 넘지 못할 수 있다. 그때는 runc-overlay 의 기동을 `internal/enode` 의
함수 하나로 옮겨 시험으로 덮는 것이 길이다.

---

## 6. 조각

**조각 0 (기동이 안 깨졌다)** — 초록. 5절 — build · vet · test 통과 · 라우트 19 그대로.

**조각 4 (trash · 사람 · SunnyVM)** — **초록.** 2026-09-26 에이전트가 돌렸고 (사용자 지시 「미뤄둔 실측 있으면 지금 해.」)
사용자가 판정했다 (15:24:25Z · 「초록 · 승인 · PR 올림」). 계획을 쓸 때는 SunnyVM 에 ssh 가 닿지 않았는데 (「No route to host」) 15:07Z 에는 닿았다 — 6일째 켜져
있었고 막힌 것은 길이었다.

차림 — 이 브랜치로 빌드한 스크래치 Mediator (이 기계의 LAN 주소 `:18080` · DB `enode_slice4` · 광고 주기 5초) 와 SunnyVM 의
**조각 전용 노드** (이 브랜치 빌드 · 설정 `slice4.yaml` · 워크스페이스와 scratch 를 따로 · 준비된 실행 환경 store 는 읽기만).
사용자의 yocto 노드와 공유 rootfs 는 건드리지 않았다. 재시작은 스크립트가 묻는 자리마다 에이전트가 했다. 결과 (노드 시계):

```text
   기동    준비도 smoke 의 작업 폴더가 trash 로 가고 삭제자가 지웠다 (53,248 바이트 · 5 ms)
   ①      큰 upper (파일 150,000 · 9 GB)   finalized_at - exited_at = 208 ms · finalize ok
          작은 upper                       43 ms · finalize ok.  둘 다 1초 안쪽
   ②      보고 직후 trash 에 작업 폴더 하나 · 상태 파일 scratch 칸이 「1 항목 · 크기 모름 · 지우는 중」 ->
          「9,667,428,352 바이트 · 지우는 중」 -> 「0 항목 · 끝」.  비기까지 1,037 ms (노드 로그 took=1.087s)
          SunnyVM 의 디스크 sda 는 스케줄러 none — bfq 가 아니라 idle IO 는 먹지 않는다
   ③      노드 로그에 「cannot remove trash entry」 0.  upper 의 항목은 노드 uid 소유다 (아래)
   ④      단계가 도는 동안 SIGKILL -> scratch 에 enode-runc-549514570 이 남았다 -> 재시작 때
          「orphaned runtime session moved to trash」 -> 지워짐 (65,536 바이트 · 5 ms)
   ⑤      min_free_gb 328 (여유 322 GB) -> GET /v1/nodes draining graceful · 상태 파일 출처 disk
          「free 322 GB < min 328 GB」 · 노드 로그 「free disk below min_free_gb; draining this node」 ·
          Mediator 가 받아 적었다.  되돌리고 재시작 -> draining ""
          덧붙임 — 이 노드는 arch 키가 처음부터 없다 (툴체인도 arch: 도 없다).  설정에 arch: armv7 을 더해
          drain 인 채로 다시 봤다 -> arch=armv7 · arch.armv7=yes 가 광고에 남는다
   ⑥      env check 의 smoke 폴더가 trash 에서 다음 보고의 Kick 을 기다렸다가 지워졌다 (8절의 한계 그대로).
          Open 실패 — 스크립트는 공유 rootfs 를 옮겨야 해서 건너뛰었다.  대신 노드를 쉼표가 든 TMPDIR 로
          띄웠다 -> 「runtime open: open runtime namespace: $IN path contains a character unsafe for overlay
          options」 · 작업 폴더(잠금 파일만 · 4,096 바이트)가 trash 를 거쳐 지워졌다.  abort 는 기본 go test 가 본다
   끝      scratch 에 enode-runc-* 0 · trash 0
```

**돌리며 고친 것** (스크립트) — Run 의 끝 상태를 `DONE` 으로 적어 첫 Run 뒤에 멈췄다 (`SUCCEEDED` · `FAILED` 로) ·
runctl 은 git 의 `user.email` 을 요청자로 쓴다 (없으면 처음에 알리게 · 이번에는 `GIT_CONFIG_*` 환경 변수로 줬다) · 스케줄러는
실제 디스크만 찍는다 · ③ 의 「subordinate uid 소유」 문구가 틀렸다 (아래) · ⑤ 에서 arch 키가 없는 노드면 알린다.

돌리는 법 (SunnyVM 에서 · 이미 도는 runc-overlay 노드로):

```bash
export M=http://<mediator>:8080 T=<bootstrap token>
export NODE_CONFIG=<그 노드의 설정 파일> WS=<그 노드의 워크스페이스>
export ROOTFS=<준비된 rootfs 경로>     # ⑥ 의 Open 실패.  비우면 건너뛴다
export ENODE=<그 노드의 enode 실행 파일>
scripts/finalize-bake/slice-4.sh
```

스크립트가 순서대로 도는 것 — FD 흐름 7절의 ① ~ ⑥:

```text
   ①  큰 upper (파일 150,000 · 9 GB) 단계와 작은 upper 단계의 finalized_at - exited_at (ms)
   ②  보고 직후부터 trash 와 상태 파일의 scratch 칸이 바뀌는 대로 · 비워지기까지 걸린 ms · 이 기계의 IO 스케줄러
   ③  큰 upper 에는 subordinate uid 소유 파일 · 권한 000 work/work · whiteout 이 있다 — 다 지워졌는지 노드 로그로
   ④  단계가 도는 동안 데몬을 SIGKILL.  scratch 에 남은 작업 폴더 -> 사람이 다시 띄운다 -> trash 로 가고 빈다
   ⑤  사람이 min_free_gb 를 여유 + 5 로 두고 다시 띄운다 -> GET /v1/nodes 의 draining · arch 키 · 상태 파일의 출처.
       되돌리고 다시 띄우면 풀린다
   ⑥  ROOTFS 를 잠시 옮겨 Open 실패 -> trash.  env check 의 smoke -> trash -> 다음 보고의 Kick 에 빈다
```

**기계로 본 것** (기본 `go test` · CI 에서 돈다) — FD 흐름 7절의 「기계」 표 그대로다.

```text
   Move          TestMoveRenamesOnceAndCreatesTheTrash · TestMoveNumbersAClashingName · TestMoveFailsWithoutRemovingAnything
   Orphans       TestOrphansFindsSessionsNobodyHolds · TestOrphansLeavesWhatItCannotJudge
   CheckEntry    TestCheckEntryRejectsNamesAndTrashes
   Measure · Remove   TestMeasureSumsBlocksAndCountsEntries · TestRemoveDeletesTheEntryAndNothingOutside ·
                 TestRemoveOfASymlinkEntryRemovesOnlyTheLink · TestRemoveLeavesAnotherFilesystemInPlace · …
   Deleter       TestDeleterEmptiesTheTrashAtStart · …RetriesAFailedEntry · …SleepsOnAnEmptyTrashUntilKicked ·
                 …StopsThePassWhenTheHelperCannotStart · …ShowsWhatIsBeingDeleted · …StopsWithItsContext
   다섯 자리      TestRuncOverlayCloseMovesTheSessionToTrash · …OpenFailuresMoveTheSessionToTrash (셋) ·
                 …FinalizeAbortsAHelperThatDoesNotAnswer (abort) · TestRuntimeHelperCleanupIsIdempotentAndRemovesNothing ·
                 …CloseDoesNotFallBackToRemoving
   trash-helper  TestRunTrashHelperMeasuresThenRemoves · TestTrashLauncherRunsTheHelperAndReadsItsLines ·
                 TestTrashLauncherOutcomes · …KillsTheHelperWithItsContext · TestTrashHelperArgvOpensNoMountNamespace
   기동 청소      TestSweepOrphanSessionsMovesOnlyWhatNobodyHolds
   diskDrain · combineDrain   TestDiskDrainHoldsUntilOneGBAboveTheMinimum · TestCombineDrainTakesTheStrongest ·
                 TestDrain_LowDiskDrainsTheNodeAndLiftsAtTheLine · …TheAdvertCarriesTheCombinedValueAndTheStatusBothSources ·
                 …NoWorkspaceOrNoMeasureMeansNoDiskDrain
   StatusBook    TestStatusBookWritesOnlyWhenSomethingChanged · …KeepsEveryFieldUnderConcurrentWriters ·
                 …RetriesAFailedWriteAndWarnsOnce · TestAnOldStatusFileStillReads
   arch 키        TestDetectKeepsBuildWhenDiskLow
   닫기와 예산     TestSettle 의 세 줄 · TestFinalize_ClosingPastTheDeadlineIsAFinalizeTimeout
   보고 뒤        TestReport_CallsAfterReportOnceTheReportIsDone
   제어판         TestDrainViewShowsWhoDrainedTheNode · TestStateCarriesTheTrashUsage · TestIndexDrawsDrainSourcesAndTrash
   입구           TestRun_TrashHelperIsReachedBeforeAnyConfigIsLooked
```

제어판의 그리기(JS)는 Go 시험이 문자열만 본다. 네 경우(소유자만 · 여유 부족만 · 둘 다 · 데몬이 멈춤)를 가짜 DOM 으로
node 에서 한 번 그려 보았다 — 풀기 버튼은 소유자 출처가 있을 때만 · 「풀어도 여유 부족 drain 이 남아 …」는 둘 다일 때만 ·
trash 줄 「trash 9.0 GiB · 항목 2 (1 개는 크기 모름) · 지우는 중 · 측정 …」. 그 하네스는 저장소에 넣지 않았다.

**사람이 SunnyVM 에서 도는 시험** (`integration` 태그 · CI 밖) — `TestRuncOverlayRuntimeIntegration` 이 이제 닫은
작업 폴더가 trash 에 권한 000 `work/work` 와 whiteout 째로 있는지 · trash-helper 가 namespace 안에서 지우는지 ·
Worker 장면이 남긴 것을 삭제자가 비우는지 본다. **2026-09-26 SunnyVM 에서 초록** — 준비된 rootfs (samsung-eabsp ·
사용자 `enode`) 로 돌렸다. helper 가 15 항목 (53,248 바이트) 을 4.7 ms 에 지웠다. 돌리며 둘을 고쳤다 — rootfs 의 사용자
이름을 `ENODE_RUNC_USER` 로 받는다 (기본 `sunny`) · 노드 uid 로 먼저 지워 보던 갈래를 없앴다 (그러면 helper 가 남은 몇
항목만 지워 시험이 약해진다).

**노드 uid 소유다.** 시험이 찍은 소유자 — `upper` · `upper/.gate-marker` · whiteout · `work` · `work/work` 모두 uid 1000
(노드 사용자). 단계 사용자(컨테이너 uid 1000)가 노드 uid 로 매핑되기 때문이다 (`runtimeIDMappings`). 그래서 FD 엔티티 3절의
「노드 uid 는 subordinate uid 소유 디렉터리 … 안을 못 걷는다」는 이 매핑에서는 생기지 않았다. 노드 uid 가 못 지우는 것은
권한 000 인 `work/work` 하나이고, 그것도 소유자라 권한을 풀면 지울 수 있다. namespace 안에서 지우는 설계는 그대로 맞고
해가 없다 (항목마다 unshare 한 번 · 약 5 ms). 팩 보안 표의 「namespace 안」 줄이 그것을 요구한다.

---

## 7. 측정 — NFR 을 건너뛰어 계획 3절이 받은 것 (N1 · 한 번에 지우는 양과 속도)

```text
   BenchmarkRemove     디렉터리 100 x 파일 1,500 (150,101 항목) 을 Measure 와 Remove 로 한 번에
                       3.14 초 (3회 평균) — 초당 약 48,000 항목 (걷기 두 번을 합친 값)
   이 기계             Intel N100 · 4 코어 · ext4 (LVM thin 볼륨) · sda 의 스케줄러 mq-deadline ·
                       루트는 dm 장치.  bfq 가 아니므로 idle IO 는 여기서 안 먹는다
```

**SunnyVM** (2026-09-26 · Intel Core Ultra 7 258V · 8 코어 · ext4 · sda 의 스케줄러 none):

```text
   BenchmarkRemove     150,101 항목에 0.99 초 (3회 평균) — 초당 약 152,000 항목
   조각 4 ②            9,667,428,352 바이트 · 파일 150,000 의 작업 폴더를 helper 가 1.087 초에 측정하고 지웠다
   idle IO             bfq 가 아니라 먹지 않는다 — 삭제자는 도는 단계와 보통 우선순위로 IO 를 나눈다 (FD 규칙 4.2)
```

값은 바꾸지 않았다 — 한 번에 항목 하나 · idle IO · CPU 19 · 10분. 팩의 9 GB · 15만 파일 upper 가 두 기계 모두 몇 초 안에
비워진다. 틀려도 trash 가 늦게 비워져 여유 부족 drain 이 걸리는 안전한 쪽이다.

**finalize 유닛이 넘긴 측정 둘도 닫았다** (그 유닛 code-summary 7절 · SunnyVM):

```text
   BenchmarkWalkWorkspace   초당 약 811,700 방문 (항목 100,100 을 5 번 · 데워진 트리) — 방문 상한 2,000,000 에
                            약 2.5 초에 닿는다.  이 기계(약 304,000)보다 빠르다.  시간 상한 30초보다 한참 앞이다
   helper 의 마감 뒤 돌아오기  overlay 세션에서 80 µs · 115 µs (integration 시험 두 번) — 여유 5초가 네 자리 넘게 크다
```

---

## 8. 알려진 한계

**namespace 삭제가 실제로 필요한 경우가 좁다.** 6절 — 이 런타임의 uid 매핑에서는 upper 의 항목이 노드 uid 소유다. 되돌릴
일은 아니다 (팩 보안 표가 namespace 안의 삭제를 요구한다). 정본에 사실로 적는다 (10절).

**비어 있던 trash 에 데몬 밖에서 들어온 항목은 다음 깸을 기다린다.** 삭제자는 깸의 끝에 trash 가 비어 있으면 10분 시계를
걸지 않는다 (FD 규칙 4.1 「비어 있으면 깨지 않는다」). 그래서 데몬이 쉬는 동안 `enode env check` 가 남긴 smoke 의 작업
폴더는 다음 결과 보고의 Kick 이나 다음 기동까지 trash 에 있다. FD 규칙 4.1 의 「데몬이 도는 동안 env check 가 남긴 것도
거둔다」는 이 조건에서만 참이다. 크기는 smoke 한 번의 작은 폴더다. 조각 4 ⑥ 이 그 모양을 보인다.

**여유 부족 drain 은 광고 주기에 걸리고 풀린다.** 광고 주기(기본 60초) 사이에 디스크가 차면 다음 광고까지 새 일을 집을 수
있다 — 오늘의 arch 키 규칙과 같은 늦음이다.

---

## 9. 뒤 유닛에 넘기는 것 (FD 흐름 10절 그대로)

```text
   lower-state   combineDrain 의 출처 목록에 bake 출처를 더한다 (Kind: bake · Owner: false).  상태 파일의 drain 칸과
                 제어판의 출처 줄이 그대로 받는다 — page.go 의 drainName · drainLift 에 「굽기」 · 「저절로 풀린다 —
                 합치기가 끝나면」 한 줄씩
   bake          Keep.Upper 에 대기 자리.  merge_wait_timeout 의 upper 는 Trash.Move 로 · 합치기의 lower 쪽 항목도
                 Trash.Move 로 (같은 filesystem · FR-7)
   checkpoint    Keep.Upper 에 spool 자리 · 받아들임은 닫기 안이고 Finalize 예산의 남은 몫을 쓴다 · Usage 에
                 spool_bytes · checkpoints 를 더한다 · 퇴출은 Trash.Move
```

---

## 10. 정본에 되돌려 올릴 것 (FD 흐름 11절 · 진행자가 올린다)

```text
   ADR-063 §4       노드가 스스로 거는 drain (여유 부족) · 소유자 drain 과 센 쪽 합치기 · 푸는 선 min + 1 GB ·
                    출처는 노드 쪽(상태 파일과 제어판)에만
   ADR-068          상태 파일의 새 칸 drain · scratch 와 쓰는 조건
   ADR-076 §4.1     trash 의 자리와 이름 · 세션 잠금으로 남은 작업 폴더를 알아본다 · 삭제자의 때(기동 · 보고 뒤 ·
                    항목이 남으면 10분) · 삭제 경계 (다른 filesystem 으로 안 넘어간다) · 측정 뒤 삭제 ·
                    우선순위는 helper 의 프로세스 그룹에 (4절 ⑤) · 8절의 한계 · 단계 사용자가 노드 uid 로 매핑되어
                    upper 가 노드 uid 소유라는 사실 (6절)
   노드 설정 문서      min_free_gb 의 뜻 — 노드 전체 drain · 되찾는 선 · arch 키와 무관
```

---

## 11. 확장 준수 — Code Generation

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 trash 삭제 줄은 FD 규칙 5절 · 4절 ⑥ 의 dirfd 걷기 · 시험 (6절의 Measure · Remove · CheckEntry) 이 닫는다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 한 줄 |
