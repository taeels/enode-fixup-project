# `lower-state` — Code Generation 요약

**유닛** `lower-state` (아래층 상태와 잠금) · **브랜치** `unit/lower-state` · **기준** `a3b1536` (Functional Design 커밋) ·
**계획** `construction/plans/lower-state-code-generation-plan.md` (열여덟 단계) · **맡는 조각** 없음 — 코드 검사로 병합한다

새 패키지 `internal/lower` 가 lower 의 신원 · 상태 자리 · 잠금 셋 · 쥔 사람 기록 · metadata · 마운트 훑기 · 준비도 점검 셋의 판정을
담는다. `internal/enode` 의 `LowerGuard` 가 광고 주기와 claim 에서 공유 잠금을 쥐고 놓고, 굽기 출처(bake)와 상태 자리 출처(lower)
의 drain 과 광고 키 다섯을 싣는다. 준비도 점검은 이음매 `FactSource` 로 Fact 셋을 더하고, smoke 도 공유를 쥔다. Mediator 는 실행 중
획득이 drain 을 보는 한 줄과 원인 코드 하나(`lower_changed`)만 바뀐다. 굽기 흐름(build · merge 단계 · 기동 때 재개)은 bake 유닛이다 —
이 유닛은 함수와 부르는 틀까지다.

---

## 1. 파일

| 파일 | 새 · 고침 | 무엇 |
|---|---|---|
| `internal/lower/lower.go` | 새 | 패키지 문서 · 타입 전부 (`Key` · `Root` · `Dir` · `Identity` · `Verdict` · `Phase` · `State` · `Holder` · `Shared` · `Exclusive` · `Bake` · `Waiting` · `Metadata` · `Marker` · `Mount` · `MountScan` · `Finding` · `Cause`) · `ErrUnsupported` · `pathError` |
| `internal/lower/judge.go` | 새 | 판정 (모든 플랫폼) — 키 글자 · `ParseKey` · 신원 · 권한 · mountinfo 풀기 · overlay 찾기 · 점검 셋의 문구 · state 와 metadata 거르기 |
| `internal/lower/root_linux.go` | 새 | `ReadRoot` (statx · statfs · 마운트 번호가 없으면 mountinfo) |
| `internal/lower/perm_linux.go` | 새 | 자리와 파일을 symlink 없이 열기 · 주인과 종류 확인 · 좁히기 · JSON 읽기 (1 MiB) · 임시 파일 + rename 쓰기 |
| `internal/lower/dir_linux.go` | 새 | `Open` · `Peek` · lower.json 판정과 쓰기 · `ReadState` · `ReadMetadata` · `WriteMetadata` |
| `internal/lower/lock_linux.go` | 새 | `TryShared` · `Update` · `Release` · `WaitShared` · `Exclusive` · `TryBake` · `Bake.WriteState` · `Holders` |
| `internal/lower/mounts_linux.go` | 새 | `ForeignMounts` — 가짜 /proc 을 받는 `scanMounts` 위에 |
| `internal/lower/check_linux.go` | 새 | `Check` |
| `internal/lower/lower_other.go` | 새 | linux 밖 — `ErrUnsupported` |
| `internal/lower/*_test.go` | 새 | `judge_test.go` (모든 플랫폼) · `root` · `perm` · `dir` · `lock` · `mounts` · `check` · `concurrency` 의 `_linux_test.go` · `lower_integration_test.go` (`integration` 태그) |
| `internal/enode/lowerguard.go` | 새 | `LowerGuard` · `StartLowerGuard` · 출처 둘 · 늦게 보인 임대 · `LowerChangedError` |
| `internal/enode/advertkeys.go` | 새 | 예약 키 다섯 · metadata 에서 키 · 광고 능력의 복사본에 싣기 |
| `internal/enode/lowercheck.go` | 새 | `ExecutionRuntimeVerifier` 의 type (옮김 · `Notice`) · `Facts` · smoke 잠금 · `LogNotice` |
| `internal/enode/runtime.go` | 고침 | `RuntimeCapability` · `StepRuntime.Capability()` · native 는 in-place · `WorkspaceWrites` |
| `internal/enode/runc_overlay_linux.go` | 고침 | `Capability()` 는 isolated · `Verify` 가 세션 앞에서 공유를 쥐고 닫은 뒤 놓는다 · type 줄을 옮겼다 |
| `internal/enode/advertise.go` | 고침 | `Guard` · `Writes` · 라벨 경고의 기억 · 출처와 키 · 응답 뒤 `AfterResponse` |
| `internal/environment/check.go` | 고침 | `FactSource` · host 점검 뒤 · prepared environment 앞 |
| `internal/panel/boundary_test.go` | 고침 | 금지 `{"cmd/mediator", "internal/lower"}` · 봉인 `{"internal/lower", x/sys}` |
| `cmd/enode/main.go` · `environment.go` | 고침 | `StartLowerGuard` 한 줄 · 칸 넷 · verifier 의 `Notice` |
| 행렬 밖 아홉 | 고침 | 8절 |
| 시험 (고침 · 새) | | `internal/enode` 의 `runtime_test.go` · `finalize_worker_test.go` (가짜 런타임 넷에 `Capability`) · `drain_test.go` · `paths_test.go` · 새 `advertkeys_test.go` · `lowerguard_linux_test.go` · `lowercheck_linux_test.go` · `lowerclaim_linux_test.go` · `runc_overlay_integration_test.go` / `internal/environment/check_test.go` / `internal/contract/bake_test.go` / `internal/store/exited_test.go` / `internal/api/api_test.go` / `internal/panel/panel_test.go` / `cmd/enode/environment_test.go` |

`detect.go` · `leases.go` · `status.go` · `internal/panel/view.go` 는 확인만 했고 안 고쳤다 (계획 2절). 계획 2절 밖의 파일은 Step 18 이
고친 회차 문서 `component-methods.md` 하나다. 시험이 바꾼 `cmd/enodectl/probe.lock` 은 되돌렸다.

---

## 2. 규칙의 자리

```text
   FD 규칙 1절 (키와 자리 · 권한)          Key.String · ParseKey · Open (ensureDir · openLock) · Peek (peekDir)
   FD 규칙 2절 (lower.json 판정표)         identityVerdict · Dir.inspect · Dir.record · clearLastAttempt
   FD 규칙 3절 (상태 기계 · 쓰기 차례)       Bake.WriteState · writeJSON (임시 파일 · fsync · rename · 디렉터리 fsync) · checkState
   FD 규칙 4절 (잠금 셋 · 배타 대기)         TryShared · WaitShared · Exclusive (pollEvery) · TryBake · 모든 잠금 fd 는 O_CLOEXEC
   FD 규칙 5절 (후보 잠금)                 LowerGuard — BeforeAdvert (5.1) · AfterResponse (5.2 · 5.4) · OnClaim (5.3 · 5.4) · start (5.5)
   FD 규칙 6절 (획득이 drain 을 본다)        internal/store/acquire.go tryGrab
   FD 규칙 7절 (쥔 사람 기록)               TryShared · Shared.write · Update · Holders · alive · validNode
   FD 규칙 8절 (마운트 0 · smoke)           ForeignMounts · scanMounts · overlaysOn · smokeLock
   FD 규칙 9절 (bake · lower 출처)          lowerSource · bakeSource · sourcesLocked · page.go 의 두 줄
   FD 규칙 10절 (합치기 없이 끝난 굽기)       bakeGoneLocked · HoldBake · DropBake
   FD 규칙 11절 (광고 키)                  reservedKey · metadataKeys · advertCaps · Advertiser.labelIgnored · keysLocked
   FD 규칙 12절 (점검 셋)                  Check · scratchFinding · ownerFinding · identityFinding · Facts · factOf
   FD 규칙 13절 (문구)                     judge.go · lowerguard.go · lowercheck.go 의 영어 문장 그대로
   FD 규칙 14절 (경계 시험)                 internal/panel/boundary_test.go
```

---

## 3. 계획 4절의 결정이 어떻게 들어갔나

- **① 임시 파일 이름** — `os.CreateTemp` 로 `.<파일>.tmp-<무작위>`. `Holders` 는 점으로 시작하는 이름을 건너뛴다. 죽은 임시 파일은
  metadata 의 것만 `WriteMetadata` 가 지운다
- **② fsync** — state.json · lower.json · metadata 는 파일과 디렉터리를 fsync 한다. 쥔 사람 기록은 안 한다
- **③ 잠금 파일** — `os.OpenFile(O_RDWR|O_CREATE|O_NOFOLLOW, 0600)`. 디렉터리는 `unix.Openat(… O_CLOEXEC)`. 셸 자식의 fd 목록에 상태
  자리의 파일이 없다 (`TestLockFdsDoNotLeak` · O_CLOEXEC 를 빼는 변이를 잡았다)
- **④ 광고 키와 상태 파일** — `advertCaps` 가 `maps.Clone` 한 복사본에 싣는다. 상태 파일은 탐지 능력 그대로다
- **⑤ 예약 키** — 복사본에서 빼고 키마다 처음 한 번 `label ignored; <key> is set by the node`. 뺀 뒤 `hasCapability` 가 거짓이면 그
  능력을 뺀다. `detect.go` 는 안 고쳤다
- **⑥ workspace.writes** — `Advertiser.Writes` 가 `""` 면 in-place. `main.go` 가 `WorkspaceWrites(stepRuntime)` 로 채운다
- **⑦ verifier 의 type** — `lowercheck.go` 로 옮겼다. `Verify` 는 두 파일에 그대로 있다
- **⑧ main.go** — 바꾼 문장 둘 · 더한 문장 하나 (`StartLowerGuard`) · 칸만 셋 (`Guard` 둘 · `Writes`). `cmd/enode` 80.4% -> 80.5%
- **⑨ OnClaim 의 자리** — 509행 임대 확인 뒤 · 링 비우기 앞. 곧바로 `defer w.Guard.StepDone(step)`. 패닉에도 짝이 맞는다
  (`TestLowerClaim_PanicStillPairsStepDone`)
- **⑩ 거절 보고** — `Result{Node, Error, Reason}`. 워크스페이스 준비 · $IN · 세션 · exited 가 없다 (`trackingRuntime` 의 Open 0 번)
- **⑪ HoldBake 부터 DropBake 까지** — `BeforeAdvert` 와 늦게 보인 임대가 공유를 잡지 않는다. `HoldBake` 는 쥔 공유를 놓는다
- **⑫ 임대가 0 이면 candidate** — 쥔 채 목록이 비면 역할 candidate · Run 을 비운다
- **⑬ Reused 의 last_attempt** — `TryBake` 로 잠깐 쥐고 지운다. 못 쥐면 `Dir.Notes` 에 한 줄
- **⑭ metadata 타입 두 벌** — `TestMetadataRecordsMatchTheContract` 가 `contract.BuildRecord` · `Pinned` 와 `DisallowUnknownFields` 로 오간다
- **⑮ bake.run 상한** — 128 바이트. 넘으면 `bake.run` · `bake.resumed` 를 빼고 한 줄 (원인이 바뀔 때만)
- **⑯ Check 와 home** — `lowers` 가 `""` 면 신원을 안 낸다. `Facts` 가 `LowersDir` 오류를 받아 신원 줄을 external-blocked 로 낸다
- **⑰ smoke 의 잠금 자리** — `newRuncOverlayRuntime` 뒤 · `MkdirAll(scratch)` 앞. 자리나 lower.lock 이 없으면 안 잠근다
- **⑱ 배타 대기의 1초** — 패키지 변수 `pollEvery`. smoke 쪽은 `smokePoll` (1초) · `smokeNotice` (30초)
- **⑲ contract 판정** — `buildName` · `irProblem` 을 `ValidBuildName` · `IRProblem` 으로 이름만 바꿨다
- **⑳ 주인이 다른 자리** — root 소유의 `/` 를 lowers 로 넘겨 `Open` · `Peek` · `Check` 가 아무것도 안 만들고 거절한다. root 로 돌면
  표 시험만 본다 (스킵하지 않는다)

---

## 4. 계획 · FD 와 다른 자리

정본과 FD 의 결정을 바꾼 자리는 없다. 겉면에 이름을 몇 개 더했고 FD 가 말하지 않은 갈래를 정했다.

- **새로 내보낸 이름 셋.** `Dir.Notes` (Open 이 고친 것) · `Dir.Loose` (Peek 이 본 느슨한 비트) · `Shared.Recorded()` (기록 없이 쥔
  공유인가). `internal/lower` 는 로그를 모르므로 알릴 것을 값으로 내고 `LowerGuard` 가 로그에 쓴다. `component-methods.md` 1.1 · 1.3 에 적었다
- **metadata 의 권한은 0644 다.** 계획은 권한을 말하지 않았다. CreateTemp 의 0600 이면 lower 안에서 그 파일만 단계 사용자가 못 읽는다 —
  lower 의 다른 파일과 같게 두었다
- **FD 13절에 없는 로그 넷.** `lower state directory fixed on open` (note) · `holding the lower lock without a holder record; a waiting
  merge cannot name this node` · `cannot update the lower holder record` · `ignoring values in .enode-metadata.json; not advertising those
  keys` (why). merging 을 처음 보면 FD 의 `the lower lock is held by a merge; draining this node` 를 쓴다
- **점검 문구 셋을 더했다.** 워크스페이스를 못 읽으면 앞의 두 Fact 가 `cannot read the workspace: <원인>` (invalid) · scratch 자리를
  못 읽으면 `cannot read the scratch: <원인>` · home 을 못 찾은 신원 줄의 remediation 은 `set HOME for the node user; the lower state
  lives under $HOME/.local/state/enode/lowers` 다. FD 12.3 은 이 셋을 적지 않았다
- **같은 node_id 의 기록 잠금을 남이 쥐고 있으면** 공유만 쥐고 기록은 안 쓴다 (FD 7절은 이름 규칙 밖만 적었다)
- **굽기 Run 의 임대를 늦게 보인 임대로 치지 않는다.** 광고 응답의 임대에는 단계 종류가 없다. `OnClaim` 이 prepare · merge 단계를
  본 Run 을 기억하고 (`baking`), `HoldBake` 의 Run 과 함께 늦게 보인 임대에서 뺀다. build claim 전에 처음 보인 굽기 Run 의 임대는
  보통 임대처럼 역할 run 이 되고 build claim 에서 놓는다
- **`DropBake` 가 표지를 새로 적는다.** 이 노드가 합친 lower 의 표지가 놓을 때의 것과 달라, 다음에 늦게 보인 임대를 거절하지 않게
- **`WaitShared` 는 다시 볼 때마다 notice 를 부르고**, 처음과 30초마다로 줄이는 것은 `smokeLock` 이다
- **마운트 훑기의 실패 갈래는 함수 값 대신 가짜 /proc 폴더로 시험했다** (`scanMounts(proc, …)`). 계획은 커버리지가 모자랄 때만 함수
  값으로 떼라고 했다 — 93.2% 라 떼지 않았다
- **3.2 표 다섯째 줄(배타가 굶지 않는다)** 은 계획의 말대로 Step 10 의 `TestLowerGuard_PendingDrainsAndTheMergeGetsTheLock` 이다
- **linux 에서만 도는 광고 시험 하나** (`TestDrain_BakeSourceKeysAndFailedAdverts` — bake 출처 · metadata 키 · 실패한 광고는 셈을 안
  바꾼다) 는 `drain_test.go` 가 아니라 `lowerguard_linux_test.go` 에 두었다. `drain_test.go` 는 모든 플랫폼 파일이다
- **runc integration 에 `TestMain` 을 두었다.** `Verify` 는 `os.Executable()` 을 helper 로 쓰므로 시험 바이너리가 `runtime-helper`
  입구를 알아야 smoke 가 진짜로 돈다
- **`advertCaps` 는 능력이 0 이면 nil 이다** — 오늘처럼 광고 본문의 capabilities 가 null 이다
- **린트** — 새 코드가 처음에 110 건이었다. 버린 오류 71 을 `_ =` 로 드러내고 staticcheck 하나(드모르간)를 고쳐 38 건 · Step 1 과 같은
  목록으로 되돌렸다

---

## 5. 코드 검사 (CI 와 같은 명령)

| 검사 | 결과 |
|---|---|
| `gofmt -l .` | 빈 출력 |
| `go vet ./...` · `go vet -tags integration ./internal/lower/ ./internal/enode/` · `go build ./...` | exit 0 |
| U+2605 · `go run ./scripts/glyphscan.go` | 0 · 160 파일에 장식 문자 없음 |
| `go test ./... -count=1` (`-coverpkg=./...` · `-json` · 시험 DB · 가짜 claude 스텁) | 통과 2,271 (Step 1 2,178) · 실패 0 · 스킵 0 · 스킵 감시 24 패키지 |
| 패키지마다 커버리지 (CI 의 awk 그대로) | 스물세 패키지 전부 80% 이상 · 전체 86.4% -> 87.1%. 새 `internal/lower` 93.2% (671/720) |
| Step 1 과 댄 패키지 | `internal/enode` 83.6 -> 84.5 · `cmd/enode` 80.4 -> 80.5 · `internal/environment` 82.2 -> 82.5 · `internal/api` 82.5 -> 82.7 · `internal/store` 83.1 · `internal/panel` 89.1 · `internal/contract` 92.5 |
| `go test -race -count=1 ./internal/lower/ ./internal/enode/` (가짜 claude 스텁) | 통과 607 · 경합 0 |
| 크로스 빌드 셋 | windows/amd64 · linux/arm (GOARM=7) · darwin/arm64 exit 0 — linux/arm 은 `Fsid.Val` 이 `[2]int32` 인 자리 (유닛 정의 10절) |
| `enodectl.exe` 심볼 | crypto/tls 1 · net/http 6 (상한 10 · 50) |
| `golangci-lint run ./...` (v2.13.2) | 38 건 — Step 1 과 같은 목록 |

파일마다 — `judge.go` · `lower.go` · `advertkeys.go` · `lowercheck.go` 100% · `dir_linux.go` 95.8% · `check_linux.go` 95.8% · `mounts_linux.go`
94.1% · `lowerguard.go` 92.8% · `lock_linux.go` 90.7% · `root_linux.go` 89.7% · `perm_linux.go` 82.5%. 남은 것은 fstat · fchmod · fsync
실패 같은 오류 갈래다.

**조각 0** — build · vet · test 초록 · 라우트 19 그대로.

**변이 셋을 넣어 시험이 잡는지 보았다.** 모두 잡혔고 되돌렸다.

- `tryGrab` 에서 drain 을 busy 에 합치는 줄을 뺀다 -> `TestAcquire_ADrainingNodeIsNotGrabbed` 의 graceful · at-boundary 가 실패
- 잠금 파일을 O_CLOEXEC 없이 연다 -> `TestLockFdsDoNotLeak` 가 셸의 fd 목록에서 lower.lock · 기록 잠금 · bake.lock 을 본다
- `internal/lower` 가 `internal/scratch` 를 임포트한다 -> 경계 시험의 봉인이 실패

**알아 둘 것 하나** — `GOOS=windows` · `darwin` 으로 `internal/enode` 의 **시험** 을 vet 하면 `overlay_test.go` 의 `overlayKernel` 이
없어 멈춘다. 이 유닛 전부터 그렇다 (CI 는 크로스로 시험을 vet 하지 않는다). 새 시험 중 진짜 잠금을 쓰는 것은 모두 `_linux_test.go` 에 두었다.

**진행자의 다시 돌리기 (승인 전)** — 서브에이전트가 끝낸 뒤 진행자가 다시 돌렸다.

- `TestLocks` (`internal/lower/lock_linux_test.go`) 가 6번 중 5번 실패했다 — 이 유닛이 들인 흔들림이다. `Shared.Release` 는 쥔 사람
  기록을 먼저 닫고 `lower.lock` 을 나중에 닫는다 (잡은 차례의 반대). 그 사이에 배타 대기가 한 번 더 보면 `Unnamed` 를 본다. 시험이 마지막
  관찰을 `sb.Release` 앞의 것으로 여겼다. 고친 것은 시험이다 — 놓기 전에 마지막 관찰을 떠 둔다. 제품 코드는 그대로다. 고친 뒤
  `-count=30 -run TestLocks` · `-count=5 ./internal/lower/` 통과
- 고친 뒤 `go test ./... -count=1` (시험 DB) 세 번 모두 통과
- CPU 를 채운 채 `internal/enode` 를 돌리면 시험 셋이 가끔 실패한다 — `TestRuncOverlayOpenIncludesHelperStderr` ·
  `TestFinalize_ACommandKilledByASignal` · `TestFinalize_AnOldMediatorGetsOneReport`. 같은 부하에서 이 유닛 전 (`a3b1536`) 의 나무도
  6번 중 5번 실패했다 (앞의 둘). 이 유닛이 들인 것이 아니다. 부하 없이는 다섯 번 모두 통과

---

## 6. 측정 — NFR 을 건너뛰어 계획 3절이 받은 것

**보안 (3.1 · 상태 자리 권한).** 시험이 본 것.

```text
   새 자리의 권한                 umask 0000 · 0002 · 0022 · 0077 모두 디렉터리 0700 · 파일 0600
   남에게 열린 자리               Open 이 lowers · <key> · holders 를 0700, lower.lock 을 0600 으로 좁히고 한 줄씩 남긴다.
                                 Peek 은 안 좁히고 Loose 로 알린다 -> 점검의 observed 끝에 「the node narrows loose permissions on start」
   symlink                       lowers · <key> · holders · lower.lock · bake.lock 어느 층이든 거절 · 밖에 아무것도 안 생긴다
                                 (가리키는 곳이 없어도 · 디렉터리를 가리켜도).  잠금을 쥐는 자리(TryShared · WaitShared · Exclusive)도 같다
   주인이 다른 자리               / 를 lowers 로 — Open · Peek · Check 가 「owned by uid 0」로 거절 · / 아래에 아무것도 없다
   읽는 파일                     디렉터리 · 1 MiB 넘는 파일 · symlink 는 못 읽은 것.  0644 기록은 읽는다 (읽기는 안 좁힌다)
```

**동시성 (3.2).** 이 기계 (Intel N100 · 커널 6.5 · ext4).

```text
   자식 프로세스 형제       자식이 공유를 쥔 동안 Exclusive 가 기다리고, SIGKILL 뒤 1.8 ms 에 잡았다 (pollEvery 2 ms 로 줄인 시험).
                           제품의 1초 간격이면 1초 안이다
   Open 여럿 동시           고루틴 16 (반은 다른 경로) — 모두 성공 · lower.json 은 JSON · 신원 칸 같다
   쓰기와 읽기 동시          WriteState 200 번 + 기록 Update 200 번 · 읽는 쪽 넷 — 읽기 103,724 · 깨진 읽기 0 · 오류 0
   fd 새지 않음             공유 · 기록 · 굽기 · 배타를 쥔 채 띄운 셸의 fd 에 상태 자리의 파일이 없다
   배타가 굶지 않는다        pending 이면 LowerGuard 가 공유를 새로 안 잡는다 -> 두 응답 뒤 놓자 기다리던 Exclusive 가 잡았다 (약 1초 — 제품의 pollEvery)
   경합                    -race 로 internal/lower · internal/enode — 0
```

**벤치마크 셋.** 3회 · `-benchtime 2s`. 기본 `go test` 에서는 안 돈다.

```text
                            이 기계 (N100 4코어)     SunnyVM (Core Ultra 7 258V · 8코어 · 커널 7.0)   계획의 추정
   BenchmarkBeforeAdvert     24 ~ 26 µs              9.1 µs                                       1 ms 아래
   BenchmarkHolders          0.12 ms                 0.037 ms                                     0.07 ms (측정 8)
   BenchmarkForeignMounts    4.9 ~ 5.4 ms            3.6 ms                                       4.7 ms (FD 계획 2.6)
                             namespace 1 · 못 읽음 5  namespace 3 · 못 읽음 1
```

셋 모두 추정의 한 자리 안이다. 광고마다 metadata 를 읽는 몫(`BeforeAdvert`)은 광고 주기 60초에 견주어 무시할 만하다.

---

## 7. integration 시험 (`integration` 태그 · CI 밖)

```text
   시험                                          이 기계                         SunnyVM (2026-09-27)
   TestForeignMountsIntegration                  초록 — 찾음 1 · namespace 2      초록 — 찾음 1 · namespace 4 · 5.2 ms
                                                 · 9.3 ms · 자식이 끝나면 0         · 자식이 끝나면 0
   TestBindAliasCheckIntegration                 초록                             초록
     bind 별칭 -> same filesystem, different mount · 원래 경로 -> same mount · 별칭으로 rename 이 EXDEV (invalid cross-device link)
   TestRuncOverlaySmokeWaitsForAMerge            못 돈다 — 이 기계는 unshare         초록 — 배타를 쥔 동안 기다리고 Notice 한 줄,
                                                 --map-auto 가 거절된다               놓은 뒤 68 ms 에 smoke 가 돌았다 · 닫은 뒤 공유를 놓았다
```

- helper 모양의 자식은 워크스페이스를 lower-ro 와 별칭 두 곳에 bind 하고 lower-ro 만 overlay 의 lowerdir 로 썼다 — 찾은 것은 overlay
  하나다 (bind 별칭만으로는 안 찾는다)
- SunnyVM 에서는 에이전트가 버려도 되는 `~/lower-it-*` 에서만 돌렸다. smoke 의 rootfs 는 그 안에 busybox 하나로 지었고 HOME 도 그
  안으로 옮겨 상태 자리가 그 안에 생겼다. `/srv/yocto` 와 떠 있는 두 노드의 설정 · 상태 자리 · rootfs 는 안 건드렸다. 끝나고 폴더를 지웠다
- 이 결과로 조각 8 (배타와 대기 · 사람 · bake 유닛) 을 대신하지 않는다

**훑기가 못 잡는 것 하나를 더 봤다.** overlay 의 lowerdir 가 symlink 경로로 적혀 있으면 (이 기계의 docker 가 `…/overlay2/l/<짧은 이름>`
으로 쓴다) mountinfo 에는 그 글자 그대로 남아 lower 에 닿는지 모른다. 제품의 helper 는 푼 경로를 쓰므로 해당이 없다. 그물이라 증거가
아니다 — 배타 잠금이 증거다 (FD 규칙 8.5 옆에 둔다).

---

## 8. 행렬 밖 파일의 diff (계획 2절의 아홉)

```text
   internal/store/acquire.go        +10  tryGrab — busyIn 뒤에 drainingIn 을 busy 에 합친다 (queue.go 와 같은 모양) · Mediator
   internal/store/claim.go          ±1   OutOfVocabulary 의 reason 목록에 contract.ReasonLowerChanged · Mediator
   internal/contract/result.go      +3   ReasonLowerChanged = "lower_changed" 와 주석
   internal/contract/bake.go        +12 -5  buildName -> ValidBuildName · irProblem -> IRProblem (이름과 주석만 · 규칙 그대로)
   internal/enode/claim.go          +22  Worker.Guard 칸 · execute 의 임대 확인 뒤 OnClaim · defer StepDone · 거절이면 보고하고 돌아간다
   internal/enode/policy.go         +6 -1  DrainBake · DrainLower · DrainSource.Kind 주석 (owner | disk | bake | lower)
   internal/enode/paths.go          +13  LowersDir — $HOME/.local/state/enode/lowers · ENODE_STATEDIR 를 안 본다
   internal/enode/runc_overlay_other.go  +6 -2  Capability() isolated · type 선언을 lowercheck.go 로 옮겼다
   internal/panel/page.go           +3 -2  drainName 에 bake 「굽기」 · lower 「아래층 상태」, drainLift 에 두 문구
```

Mediator 쪽은 앞의 둘이다. 획득 쪽 바뀐 동작 — drain 중인 노드만 남은 획득은 못 잡음(`unavailable`) 갈래로 간다. drain 없는 노드는 오늘
그대로 잡힌다 (`TestAcquire_ADrainingNodeIsNotGrabbed` 의 세 갈래).

---

## 9. 다른 유닛에 넘기는 것 (FD 흐름 12절 그대로 · 이 단계가 더한 것)

```text
   bake          FD 흐름 12절의 전부 — merge Preflight 의 마운트 확인 (merge_linux.go:81 · :86 옆) · merge 단계와 재개의 흐름 ·
                   state.json 을 쓰는 때 · HoldBake · DropBake 와 abandon 몸통 · WriteMetadata 를 부르는 자리 · 옛 판 노드의 장면
                 (더함) 차례 — 배타를 기다리기 전에 HoldBake(run, abandon).  committed 를 쓰고 두 잠금을 놓은 뒤 DropBake
                   (계획 4절 ⑪ · DropBake 가 새 표지를 적는다)
                 (더함) Exclusive 의 watch 는 막혀 있는 동안 처음과 every 마다 불린다.  Waiting.Unnamed 는 살아 있는 기록이 0 인데 막혔다
                 (더함) ForeignMounts 가 오류를 돌려주면 (self mountinfo 를 못 읽음) 그물을 못 친 것이다 — 합칠지는 bake 가 정한다
                 (더함) WriteMetadata 는 lower 뿌리의 옛 .enode-metadata.json.tmp-* 를 먼저 지우고 0644 로 쓴다.  schema 0 이면 1 로 채운다
                 (더함) lower_changed 로 끝난 Run 은 다시 내면 된다 — 조각에서 보이면 그 뜻

   checkpoint    RuntimeCapability 에 Capture 칸 · StepRuntime.Capability() 가 그 값을 낸다
```

---

## 10. 정본과 회차 문서

**정본** — FD 흐름 13절 그대로 진행자가 `enode-design` 에 올린다. 이 단계가 더하는 사실 둘:

- ADR-077 §5 — metadata 는 lower 뿌리에 0644 로 쓴다 (임시 파일 + rename · 옛 임시 파일을 먼저 지운다)
- ADR-077 §6 — 마운트 훑기는 lowerdir 가 symlink 경로로 적힌 overlay 를 못 잡는다 (7절). 증거는 배타 잠금 그대로

**회차 문서** — `component-methods.md` 1.1 · 1.3 · 1.5 · 4.1 · 4.3 · 4.4 에 코드의 겉면이 달라진 줄을 적었다 (`ReadRoot` · `ParseKey` ·
`Dir.Notes` · `Dir.Loose` · `Shared.Recorded` · `Check` 의 `lowers` · `WorkspaceWrites` · `StartLowerGuard` · `LowerChangedError` ·
verifier 의 `Notice` · `LogNotice`).
