# `bake` — 규칙

노드가 굽기의 build · merge 단계를 받으면 어느 차례로 무엇을 부르나, 실패하면 upper · state.json · last_attempt 를 어디로
돌리나, 끊긴 합치기를 누가 언제 잇나의 규칙이다. 오류 · 로그 문구는 영어다 (`CONVENTIONS.md` 2.1). 타입과 이름은
`domain-entities.md`, 흐름과 시험은 `business-logic-model.md` 에 있다.

**되물음 다섯은 답을 받았다** (`construction/plans/bake-functional-design-clarification-questions.md` · 2026-09-27T10:38:41Z · 되물음 2 는
2026-09-27T11:59:51Z 에 B 로 다시 답함). 그 답이 정한 문장에는 「되물음 N 답 A · B」를 근거로 달았다 — 1 A 광고 주기가 주인이 죽은 building ·
pending · merging 을 기동과 같은 표로 다룬다 · 2 B 합칠 것 없음은 merge 단계 DONE 이고 판정은 계약의 조건이 한다 (I3) · 3 A 셸은 노드의 래퍼
bash · 4 A build claim 이 낡은 merging 을 만나면 재개를 열고 bake_in_progress · 5 A 답 뒤에 정한 다섯을 산출물대로.

**원칙 셋**

- **merging 에서 committed 로 가는 길은 합치기가 끝나는 것 하나다.** merging 을 쓰기 전의 실패는 모두 lower 를 그대로 두고
  committed 로 돌린다 (upper 는 trash). merging 을 쓴 뒤에는 lower 가 반쯤 바뀌었을 수 있으므로 되돌리지 않고, 재개가 끝낸다
  (계획 3.1 · 답 2 재개의 때)
- **굽기 하나에 주인 하나, 치우는 몸통 한 번.** state.json 을 쓰는 쪽은 굽기 잠금(bake.lock)의 주인 하나다 (lower-state 규칙 3절).
  치우는 몸통은 여러 길에서 불려도 한 번만 돈다 (계획 3.12)
- **판정은 계약의 조건이 하고, 노드는 완주했나만 말한다** (I3 · INVARIANTS.md:148 · :292 — result.error 가 비면 완주). 원인 코드는 노드가
  붙이는 까닭이다. 대개는 노드가 스스로 멈춘 FAILED 에 붙는다 — 예산 둘 · lower_changed · bake_in_progress · merge_wait_timeout.
  **ir_mismatch 는 완주 (DONE) 에 붙는다** — builds 를 돌리지 않고 manifest 를 내지 않은 까닭을 말하고, Run 은 계약의 produced ["manifest"]
  조건이 FAILED 로 판정한다 (물음 1 답 B · 결정 54). 명령의 0 아닌 exit 도 완주라 DONE 이다

---

## 1. 단계 분기와 거절 (계획 3.2 · 받는 일 8 · 9 · 11 · 20)

- **분기 자리** — `execute` (`internal/enode/claim.go:508`) 에서 `OnClaim` · `StepDone` 짝(:524 ~ :525)과 임대 감시(:655) 뒤, 빈 argv
  확인(:676 ~ :679) 앞. kind 가 `build` 면 build 흐름, `merge` 면 merge 흐름이다. 굽기 단계는 workspace 와 in 을 받지 않으므로
  (`contract/bake.go` 의 checkBuildStep · checkMergeStep) 그 앞에 준비된 것은 `$OUT` · `$IN` 임시 폴더뿐이다
- **병합 뒤의 창이 닫힌다** — 오늘 굽기 단계는 `run step has an empty argv` 로 끝난다 (contract-grammar code-summary 7절). 계획이
  지은 굽기 (Grammar 의 굽기 절) 도 kind 가 같으므로 같은 분기로 들어온다
- **Baker 가 없는 노드는 거절한다** — native · 런타임 없음 · lower 루트가 없는 runc-overlay. 준비 없이 곧바로 FAILED, 원인 코드
  없음 (노드 설정의 일이다):
  `a bake step needs a node that writes to an upper (workspace.writes=isolated); this node writes to its workspace in place`
- **상태 자리를 못 여는 노드** (`LowerGuard.Dir()` 이 nil) 와 **잠금 파일이나 state.json 을 못 읽은 때** (`TryBake` · `ReadState` 의
  오류 · `lock_linux.go:195` ~ `:206`) 는 곧바로 FAILED — `cannot open the lower state directory: <원인>`. 「주인이 살아 있다」
  (bake_in_progress) 로 읽지 않는다. build 단계가 방금 잡은 굽기 잠금이면 놓는다. merge 단계처럼 이 프로세스가 그 굽기를 쥐고
  (`Baker.held`) 있으면 치우는 몸통으로 끝낸다 (11절 · 결정 44)
- 거절과 이 오류들은 굽기를 시작하지 않았다 — exited 도 last_attempt 도 없다. 몸통으로 끝낸 merge 단계만 last_attempt 가 있다

---

## 2. 전이와 쓰는 때 (계획 3.1 · ADR-077 §5 상태 기계 · lower-state 규칙 3절 · 받는 일 39)

```text
   때                                                     전이                        state.json 에 적는 것
   build claim · 굽기 잠금 · 대기 자리를 만든 뒤              committed -> building       owner (Run · 단계 seq · 노드 · 인스턴스) ·
                                                                                     pending_upper · since
   build 성공 · 닫기 · 초안 뒤 (Finalize 예산 안)             building -> pending         since (owner · pending_upper 그대로)
   merge · 배타 · 그물 · 시작 전 확인 뒤 · 첫 Apply 전         pending -> merging         since
   metadata 뒤 (대기 자리는 이 뒤에 옮긴다 · 12.4)            merging -> committed        owner · pending_upper 를 비운다 · last_attempt 를 지운다
   치우는 몸통 · 기동 정리 · build claim 의 낡은 상태 정리       building · pending ->       owner · pending_upper 를 비운다 · last_attempt
                                                             committed
```

- **pending_upper 를 building 때 적는다** — 세션을 닫는 rename 과 pending 쓰기 사이에 죽으면 기동 정리가 upper 의 자리를 알아야 한다.
  lower-state 규칙 3절은 building -> pending 에 적는다고 썼다 — 앞당겼다 (흐름 13.3)
- **building · pending · merging 을 쓸 때 있던 last_attempt 를 그대로 싣는다** — 새 굽기가 앞 굽기의 abandoned 기록을 지우지 않게 한다.
  재시작한 노드의 merge claim 이 그 기록으로 자기 Run 을 알아본다 (7.1 · 결정 49). 지우는 것은 합쳐 committed 로 갈 때뿐이다
- **합쳐 committed 로 가면 last_attempt 를 지운다** (되물음 5 답 A · (8)). 마지막 시도가 성공했다 — metadata 의 bake 칸이 그 시도다. 앞의
  실패는 그 Run 의 Record 에 남는다 (결정 3-18 실패한 시도의 기록)
- **committed 를 쓴 뒤에 대기 자리를 옮긴다** — 초안이 committed 전까지 남아 있어야 재개가 「이미 끝났나」를 초안으로 대 본다 (12.4)
- 쓰는 법은 lower-state 의 `Bake.WriteState` 그대로다 — 임시 파일 · fsync · rename · 자리 디렉터리 fsync. merging 은 첫 합치기 동작
  전에 디스크에 있다
- 재개는 state.json 을 merging 그대로 두고 시작한다 — 끝나면 committed 하나를 쓴다

---

## 3. build 의 차례 (계획 3.3 · 3.6 · FR-6 build 는 upper 에 짓는다 · ADR-077 §2 굽기 계약 · 답 3 첫 실패에서 멈춤)

```text
   1   굽기 잠금       이 프로세스가 이미 굽기를 쥐었거나 재개 중이다 -> bake_in_progress
                      TryBake 오류 -> cannot open the lower state directory (1절)
                      TryBake 가 안 된다 (주인이 살아 있다) -> bake_in_progress
   2   state 를 읽는다  오류 -> 잠금을 놓고 cannot open the lower state directory
                      committed -> 3
                      building · pending (주인이 죽었다) -> 낡은 상태 정리 (12.2 의 표) 뒤 3
                      merging (주인이 죽었다) -> 잡은 잠금을 넘겨 재개를 배경에 연다 (13절) · bake_in_progress (답 2 · 되물음 4 답 A)
   3   previous_ir     lower 의 metadata 의 source.ir — 초안에 적는다.  못 읽으면 null 과 노드 로그 경고 한 줄
   4   대기 자리        MkdirTemp(<scratch>/pending/<lower 키>/).  실패 -> 몸통 · FAILED (결정 44)
   5   building        WriteState(building).  실패 -> 몸통 · FAILED (결정 44)
   6   세션            RuntimeSpec 은 명령 단계와 같다 — Dir 은 워크스페이스 · In · Out · Record
   6'  bash 확인       ["sh", "-c", "command -v bash >/dev/null"] (노드의 고정 한 줄).  0 이 아니면 노드 쪽 오류 — 계약의 명령을
                      하나도 안 돌렸다 (되물음 3 답 A).  exit 의 갈래는 아래 문단 (결정 45)
   7   sync            ["bash", "-c", <sync>].  0 이 아니면 builds 를 안 돈다 -> 실패 끝 (6절)
   8   IR 대조          4절.  어긋나면 실패 끝 — DONE · reason ir_mismatch (물음 1 답 B) · 못 대 보면 노드 쪽 오류
   9   pinned          repo 모양일 때만 — repo manifest -r -o .enode-manifest.xml (세션 안 · 워크스페이스 뿌리)
   10  builds          적힌 차례로.  하나가 0 이 아니면 멈추고 남은 구성마다 skipping 줄 -> 실패 끝
   11  성공 끝          exited · Finalize · Close(Keep{Upper: <대기 자리>/upper}) · pinned 의 sha256 · 초안 · WriteState(pending)
                      — 여기까지 Finalize 예산 안.  넘겼으면 pending 을 쓰지 않는다 (6절) —
                      HoldBake(run, 몸통) · $OUT/manifest · 업로드 · 보고 DONE (exit 0 · build 칸)
```

- **명령마다 세션의 `Run` 한 번** — argv 는 `["bash", "-c", <계약의 명령 문자열>]` 이다 (되물음 3 답 A — 정본 예시
  `source build/envsetup.sh && …` 가 bash 의 내장어를 쓴다 · ADR-077:198 · 이 기계의 dash 는 `source: not found`). 작업 폴더는 워크스페이스 자리 · 시작과 끝 시각(노드 시계)과
  exit 를 `lower.BuildRecord` 로 적는다 (sync 도 같은 모양 · 이름 `sync`). **명령에는 시간 상한이 없다** — 임대가 끝나면 끊긴다
  (오늘의 임대 감시). 셸 문자열로 도는 것은 INVARIANTS.md:293 (명령 단계의 셸은 argv 배열) 과 다르다 — 정본 되돌림 (흐름 13.1)
- **bash 는 sync 전에 확인한다** (되물음 3 답 A) — 세션을 연 뒤 노드의 고정 POSIX sh 한 줄 `command -v bash >/dev/null` 을 돌린다.
  갈래는 넷이다 (결정 45) — 모두 계약의 명령이 돌기 전이라 exited 가 없고, 끝은 몸통이다 (upper 는 trash · committed · last_attempt 는 그 문장)

```text
   exit 0                    통과 — sync 로 간다
   exit 1 이상               FAILED · cannot run the bake commands: the prepared rootfs has no bash, or sh cannot run there (exit <n>)
                             dash 의 command -v 는 없으면 127, bash 는 1 (이 기계 측정 · 2026-09-27).  sh 자체를 못 띄우면 runc 가
                             1 을 낸다 (QA 재검 측정 · runc 1.1.5 — 이 기계의 runc 도 1.1.5 이나 세션을 못 열어 다시 재지 못했다)
   -1 (세션 오류)             FAILED · runtime run: <원인> — 노드 쪽 오류 (`runc_overlay_linux.go:380` ~ `:382`)
   runCtx 가 끝났다          임대면 FAILED aborted: lease expired · 데몬이 멈췄으면 보고 없음 (6절)
```

  **앞에서 확인하는 까닭** — bash 가 없는 rootfs 에서 `bash -c` 는 runc 가 실행 파일을 못 찾아 exit 1 이 된다 (QA 재검 측정). sync 가 흔히
  내는 실패 exit 1 과 같은 값이라 명령 실패 (DONE) 와 구별되지 않는다. 처음 판의 「exit 127」은 틀렸다. pinned 의 `repo manifest -r` 도
  bash 로 돈다
- **환경** — 오늘의 명령 단계와 같은 목록 (`commandEnv` (`env.go:201`) 와 계약의 `env` 이름) 에 `OUT` · `IN` · `ENODE_IR`
  (`contract.EnvIR`) 을 더한다. 이름은 contract-grammar 의 코드가 닫았다
- **출력** — 명령의 stdout · stderr 는 오늘처럼 단계 로그의 버퍼와 진행 청크(`w.transcript`)로 간다. 명령마다 머리 한 줄과 끝 한 줄을
  노드가 같은 자리에 쓴다 (16.2)
- **sync 가 실패하면 builds 를 안 돈다** — 맞지 않은 트리 위의 빌드는 뜻이 없다 (계획 3.3). **build 하나가 실패하면 남은 builds 도 안
  돈다** (답 3) — 실패한 굽기의 upper 는 trash 로 가므로 뒤 구성의 산출물과 캐시가 남지 않는다. 이 근거는 실패한 build 의 upper 를
  보존하지 않는다는 전제 위에 있다 — checkpoint 유닛이 보존하면 다시 본다 (흐름 12절)
- **Finalize 는 effect prepare 의 몫이다** (finalize 가 넘긴 「build 단계의 수확」) — `finalizeSpecFor` (`finalize.go:63`) 그대로 diff
  없음 · 훑기 없음 · 지목 경로 stat 만. 올리는 이름은 manifest 하나다 (15절). merge 단계는 Finalize 를 돌리지 않는다 — 수확할 명령이 없다
- **pinned 는 IR 이 맞은 바로 뒤, builds 전에 뜬다** — builds 는 저장소의 판을 안 바꾸고, 실패하면 빌드 전에 알린다. git 모양이면
  pinned 는 null 이다 (ADR-077:218 — manifest 가 없는 단일 git workspace). 파일은 upper 에 남아 lower 로 합쳐진다
- **pinned 의 sha256 은 닫은 뒤 대기 자리에서 읽는다** — upper 의 파일은 컨테이너 사용자가 썼고 호스트에서 노드 uid 소유다
  (`runc_overlay_linux.go:1269` ~ `:1275` 의 매핑)
- **닫기** — `Close(Keep{Upper: 경로})` 는 helper 가 unmount 한 뒤 `<runRoot>/upper` 를 그 경로로 rename 한 번, 나머지 runRoot 는
  trash (trash 가 넘긴 일 · 받는 일 21). 오늘 `Close` 는 Keep 을 받기만 한다 (`runc_overlay_linux.go:486`). 성공이 아닌 끝은 늘 `Keep{}`
  — upper 가 runRoot 와 함께 trash 로 간다
- **HoldBake 는 pending 을 쓴 바로 뒤** 다 (11절). 넘기는 함수는 `func() { hold.abandon("the bake run ended before the merge", <초안의 builds>) }`
  모양이다 — `HoldBake(run string, abandon func())` (`lowerguard.go:490`) 에 맞춘다

### 3.1 초안의 칸 (계획 3.6)

```text
   source.url            IR 대조가 읽은 remote.origin.url.  비밀번호는 지운다 (아래).  origin 이 없으면 "" (결정 47)
   source.branch         git rev-parse --abbrev-ref HEAD.  detached ("HEAD") 면 ""
   source.repo_id        DetectRepo (repoid.go:77) 의 규칙을 세션 안의 값에 그대로 — repo 모양이면 CanonicalRepoID(url) + "#" + branch
                         (branch 가 비었으면 # 없이) · git 모양이면 CanonicalRepoID(url).  합친 뒤 detect 가 광고할 repo= 와 같다.
                         url 이 "" 면 "" — detect 도 origin 이 없으면 "" 다 (repoid.go:81 ~ :84)
   source.head           IR 대조의 HEAD
   source.ir             계약의 IR (맞았으므로)
   source.pinned         {file: .enode-manifest.xml, sha256} · git 모양이면 null
   source.sync_command   계약의 sync 그대로
   source.synced_at      sync 의 끝 시각
   builds                돈 builds (성공이면 모두 exit 0 — ADR-077 §5)
   environment           Record.PreparedEnvironment
   workspace_target      Record.WorkspaceTarget
   run · node            굽기 Run · 이 노드
   previous_ir           3 의 값.  못 읽어 null 인 것과 옛 ir 이 없어 null 인 것은 metadata 로는 구별하지 못한다 — 노드 로그가 구별한다
```

- **url 의 비밀번호를 지운다** — `scheme://user:password@host/...` 모양이면 `password` 만 빼고 `user` 는 둔다. scp 모양
  (`user@host:path`) 은 그대로다. metadata 는 lower 뿌리의 0644 파일이라 그 lower 위의 모든 Run 이 읽는다. repo_id 는 이미 자격
  증명을 뺀다 (`CanonicalRepoID`)
- 초안을 쓰는 법과 읽는 법은 `domain-entities.md` 4절이다

---

## 4. IR 대조 (물음 1 답 B · 계획 3.4 · FR-9 metadata 쓰기와 IR 대조 · 완료 조건 10 IR 과 HEAD 가 어긋난 이유)

```text
   보는 git       DetectRepo (repoid.go:77) 와 같은 차례 — <ws>/.repo/manifests 가 디렉터리면 그것 (repo 모양) ·
                 아니고 <ws>/.git 이 있으면 워크스페이스 뿌리 (git 모양) · 둘 다 없으면 대조를 못 한다
   자리           세션 안 · 노드가 지은 고정된 POSIX sh 한 줄 (셸 답과 무관 — 노드의 한 줄은 늘 sh) · 작업 폴더는 워크스페이스 자리
   환경           ENODE_IR 는 세션 환경으로 넘긴다.  LC_ALL=C · GIT_TERMINAL_PROMPT=0 · GIT_CEILING_DIRECTORIES=<워크스페이스
                 자리의 부모> 는 그 셸 한 줄 안에서 export 한다 — 세션 환경은 LANG · LC_ALL 을 프로필의 locale 로 덮는다
                 (runc_overlay_linux.go:1280 ~ :1284)
   명령           git rev-parse HEAD · git rev-parse -q --verify "refs/tags/$ENODE_IR^{commit}" · git tag --points-at HEAD ·
                 git config --get remote.origin.url · git rev-parse --abbrev-ref HEAD.  앞의 셋이 실패하면 「대조를 못 함」이고, 뒤의
                 둘은 실패해도 빈 값이다 — origin 이 없는 저장소 (git init · fetch 로 받은 것) 도 대조한다 (결정 47)
   판정           태그의 커밋 == HEAD            맞음.  HEAD 에 다른 태그가 함께 있어도 된다 (계획 2.1 · 한 커밋에 태그 둘)
                 태그의 커밋이 없다 (빈 출력)     로컬에 없음 — ir_mismatch
                 태그의 커밋 != HEAD            다른 커밋 — ir_mismatch
                 보는 git 이 없다 · git 이 실패   대조를 못 함 — 노드 쪽 오류 (원인 코드 없음)
```

- **세션 안에서 도는 까닭** — upper 는 호스트의 보통 디렉터리지만 (`runc_overlay_linux.go:899` ~ `:911`), git 이 읽을 것은 lower 와
  upper 를 합친 트리이고 그 트리는 helper 의 마운트 namespace 안에만 있다 (overlay 마운트 `:931`). 새 helper 동작이 필요 없고 격리
  runtime 을 벗어나지 않는다 (팩 보안 표의 계약 명령 줄)
- **git 모양 (`.repo` 없음) 도 대조한다** — SunnyVM 의 시험 lower 는 poky 를 git 으로 받아 `.repo` 가 없다 (contract-grammar 되물음 파일 끝 ·
  받는 일 4). 정본의 IR 문장은 `.repo/manifests` 만 적는다 (ADR-077:227 ~ :230). 근거는 이 단계가 고친 FR-9 (`requirements.md` · 없으면
  워크스페이스 뿌리의 `.git`) 와 contract-grammar 가 넘긴 일이다 — 정본 차이는 흐름 13.1 §5 에 올린다. ADR-077:218 은 metadata 칸의
  문장이다 (git 모양이면 pinned 가 null)
- **`^{commit}` 로 벗긴다** — annotated 태그와 가벼운 태그가 같은 값으로 대조된다 (계획 2.1)
- **부모 저장소를 읽지 않는다** — `GIT_CEILING_DIRECTORIES` 를 두면 워크스페이스 뿌리 위로 올라가지 않는다 (계획 2.1 끝 줄)
- **safe.directory 를 건드리지 않는다** — 컨테이너 사용자가 파일 주인과 같은 uid 로 보인다 (3절의 매핑). git 이 소유를 거절하면
  「대조를 못 함」이고 그 stderr 가 문장에 실린다. 조각 6 (굽기가 끝까지 돈다) 이 처음 확인한다 (계획 2.6)
- **rootfs 에 git 이 있어야 한다** — sync 가 git 이나 repo 를 쓰면 이미 있다. 없으면 exit 127 이 「대조를 못 함」이다
- **sync 바로 뒤, builds 전이다** — 빌드 하나가 30 ~ 80 분이다 (ADR-077 §12 실측 · 1,892 초 · 4,760 초)
- **어긋나면** (물음 1 답 B · 2026-09-27T12:54:46Z) — builds 를 안 돈다 (남은 구성마다 skipping 줄) · exited (sync 의 것) · Finalize ·
  `Close(Keep{})` · 치우는 몸통 (upper 는 trash · committed · last_attempt reason `ir_mismatch`) · `$OUT/manifest` 없음 · 단계 로그 끝에 문장
  한 줄 (아래 두 갈래 중 하나) · 보고 **DONE** (error 없음 · exit 0 — sync 의 것 · reason `ir_mismatch` · build 칸에 sync 의 기록 · head ·
  head_tags · ir null). merge 는 needs 를 따라 와서 합칠 것 없음 (DONE · 7.1 다섯째 줄) 이다
- **판정은 계약의 조건이 한다 — I3 을 지킨다** (I3 · INVARIANTS.md:148) — manifest 는 모든 명령이 0 으로 끝나고 IR 이 맞았을 때만 나오므로
  (`internal/contract/bake.go:36` ~ `:37`) 계약의 produced ["manifest"] 조건이 Run 을 FAILED 로 판정한다. 노드는 완주했다는 것과 까닭
  (reason) 만 말한다. 명령 실패 (답 3 의 첫 실패에서 멈춤) 와 같은 모양이다 — 둘 다 DONE 이고 manifest 가 없다
- **대조를 못 했으면** head 는 "" · head_tags 는 null 이다 — 읽지 못한 것을 [] 로 쓰지 않는다 (되물음 5 답 A · (11))
- contract-grammar 흐름 3절의 「merge 는 needs 를 따라 와서 합칠 것이 없다」는 명령 실패와 IR 어긋남 두 갈래에 그대로다 (물음 1 답 B) —
  그 절이 알리는 법을 bake 에 넘겼다 (받는 일 10) — 그 알리는 법은 merge 단계 DONE · merged 없음 · 단계 로그 한 줄이고, 판정은 계약의
  조건이 한다 (되물음 2 답 B · 7.1)

문장 (영어) — 어긋남의 두 갈래는 단계 로그의 글자다 (DONE 이라 error 가 없다 · 물음 1 답 B). 못 함은 error 와 단계 로그에 같은 글자다.
태그 목록은 빈칸으로 잇고, 없으면 `none`.

```text
   로컬에 없음    ir <IR> is not in the local repository after sync; the sync command must fetch that tag (HEAD is <commit>, tags at HEAD: <tags>)
   다른 커밋      ir <IR> points at <commit 1>, but sync left HEAD at <commit 2> (tags at HEAD: <tags>)
   못 함         cannot verify ir: the workspace has neither .repo/manifests nor .git
                cannot verify ir: git exited <n>: <stderr 의 마지막 줄>
   맞음 (로그)    bake: ir <IR> matches HEAD <commit> (tags at HEAD: <tags>)
```

---

## 5. exited 와 예산 (계획 3.5 · 받는 일 12 · 13 · 18 · 19)

- **exited 는 계약의 명령이 더 돌지 않게 된 때 한 번** — 마지막으로 돈 명령 (실패한 것 · IR 이 어긋나거나 pinned 가 실패했으면 sync ·
  성공이면 마지막 build) 의 결과로 보낸다. `Client.Exited` 와 `startExitReport` (`finalize.go:140`) 를 명령 단계처럼 쓴다.
  exited_at 은 그 명령이 끝난 때다
- **안 보내는 때** — 계약의 명령이 하나도 안 돌았다 (거절 · bake_in_progress · 상태 자리를 못 엶 · 세션을 못 엶 · bash 가 없음) · 명령이 임대로
  끊겼다 (오늘 명령 단계의 규칙 · `claim.go:749` 의 갈래) · 명령 도중 helper 가 죽었다 (`runtime run:` · exit -1 — 명령이 끝나지 않았다 ·
  6절 표) · 데몬이 멈췄다. 계획 3.5 는 「하나도 안 돌았으면」만 적었다 — 뒤의 셋은 오늘 명령 단계의 규칙을 따른 것이다
- IR 대조 · pinned · git 조회는 노드의 명령이라 세지 않는다
- **두 예산은 오늘 규칙 그대로다** — Finalize 예산이 결과 확정 · 닫기(Keep 의 rename) · pinned 의 sha256 · 초안 · pending 쓰기를 덮고,
  업로드 예산이 단계 로그와 manifest 를 덮는다. 계약이 build 의 budget 을 늘릴 수 있다 (contract-grammar). 예산은 kind 를 안 보므로
  `contractStep` 을 넓혀도 build 의 값은 같다 (finalize 가 넘긴 일)
- **merge 단계는 exited 를 안 보내고 예산을 안 쓴다** — 기다림은 merge.wait, 합치기 본체는 상한이 없다 (결정 3-11 대기 상한 기본
  4시간 · 본체에 상한 없음). Mediator 도 merge 의 exited 를 받지 않는다 (`store/claim.go:793` · `:847`)

---

## 6. build 이 합칠 upper 를 못 남기면 (계획 3.7 · 물음 1 답 B · 답 2 · 받는 일 10 · 47)

| 갈래 | 단계 | 원인 코드 | upper | state.json | last_attempt | exited | merge 단계 |
|---|---|---|---|---|---|---|---|
| Baker 가 없는 노드 · 상태 자리 · 잠금 파일 · state.json 을 못 엶 | FAILED | 없음 | 없음 | 안 건드린다 | 안 남긴다 | 없음 | 안 온다 |
| bash 확인 실패 — bash 없음 · sh 를 못 띄움 (되물음 3 답 A · 결정 45) | FAILED | 없음 | runRoot 째 trash · 대기 자리 trash | committed | 남긴다 | 없음 | 안 온다 |
| 굽기 잠금의 주인이 살아 있다 · 이 프로세스가 이미 굽기를 쥐었다 | FAILED | `bake_in_progress` | 없음 | 그대로 | 안 남긴다 | 없음 | 안 온다 |
| 낡은 merging (주인이 죽었다 · 답 2 · 되물음 4 답 A) | FAILED | `bake_in_progress` | 없음 | merging 그대로 — 이 노드의 배경 재개가 잇는다 | 안 남긴다 | 없음 | 안 온다 |
| 낡은 building · pending (주인이 죽었다) | 정리하고 진행한다 (ADR-077 §7 굽기끼리의 배타와 재개) | — | 옛 대기 자리를 그 자리의 scratch 의 trash | committed (옛 주인의 last_attempt) 뒤 building | 옛 주인의 것 | — | — |
| 세션을 못 엶 | FAILED | 없음 | runRoot 째 trash · 대기 자리 trash | committed | 남긴다 | 없음 | 안 온다 |
| sync 가 0 아님 | DONE (exit 는 sync 의 값) | 없음 | trash | committed | 남긴다 | sync | 온다 — 합칠 것 없음 · DONE (되물음 2 답 B) |
| IR 이 어긋남 (물음 1 답 B) | DONE (exit 0 — sync 의 것) | `ir_mismatch` | trash | committed | 남긴다 (`ir_mismatch`) | sync | 온다 — 합칠 것 없음 · DONE |
| IR 을 못 대 봄 · pinned 실패 | FAILED | 없음 | trash | committed | 남긴다 | sync | 안 온다 |
| build 하나가 0 아님 (답 3) | DONE (exit 는 그 build 의 값) | 없음 | trash | committed | 남긴다 | 그 build | 온다 — 합칠 것 없음 · DONE (되물음 2 답 B) |
| 임대가 끝남 (명령 중) | FAILED (`aborted: lease expired`) | 없음 | trash | committed | 남긴다 | 없음 | 안 온다 |
| 명령 · 대조 · pinned 도중 helper 가 죽음 (`runtime run: <원인>` · exit -1) | FAILED | 없음 | trash | committed | 남긴다 | 없음 (명령이 끝나지 않았다) | 안 온다 |
| 데몬이 멈춤 (SIGTERM · 명령 중) | 보고 없음 — 임대가 끝나거나 재시작이 Run 을 닫는다 (ADR-030) | — | trash | committed | 남긴다 (`node stopped`) | 없음 | — |
| pending 을 쓰기 전의 노드 쪽 오류 (닫기 · 초안 · pending 쓰기) | FAILED | 없음 | trash | committed | 남긴다 | 마지막 build | 안 온다 |
| pending 을 쓰기 전에 Finalize 예산을 넘김 | FAILED | `finalize_timeout` | trash — pending 을 안 쓴다 | committed | 남긴다 | 마지막 build | 안 온다 |
| pending 을 쓴 뒤 업로드 예산을 넘김 | FAILED | `upload_timeout` | trash (몸통) | committed | 남긴다 | 마지막 build | 안 온다 |
| 성공 | DONE (exit 0) | 없음 | 대기 자리 | pending | 안 남긴다 | 마지막 build | 온다 — 합친다 |

- **명령의 0 아닌 exit 는 완주다** (INVARIANTS.md:292 · 오늘의 명령 단계). build 단계의 성공은 success_when 의 produced manifest 가
  판정한다 (contract-grammar 답 1). **판정 조건을 건 계약이면 Run 은 build 의 조건에서 실패다.** 판정 조건이 없는 굽기 계약은 400 이
  아니라 lint 경고라 (contract-grammar 규칙 8절 끝) `Verify` 가 SUCCEEDED 에서 시작하면 (`internal/store/verdict.go:128` ~ `:129`) Run 이
  SUCCEEDED 가 될 수 있다 — **그대로 둔다** (되물음 2 답 B). 판정은 계약의 조건이 한다 (I3 · INVARIANTS.md:148 · :292 ·
  `internal/store/claim.go:977` ~ `:984`). 그 결과는 lint 경고 · Record 의 build 칸 (명령의 exit) · merge 단계 로그의 「nothing to merge」가
  알린다
- **IR 어긋남도 같은 모양이다** (물음 1 답 B · 결정 54) — build 는 DONE 이고 manifest 가 없으므로 produced ["manifest"] 를 건 계약이면
  Run 은 build 의 조건에서 실패다. 조건이 없는 굽기 계약이면 SUCCEEDED 가 될 수 있다 — 그대로 둔다. 그때는 Record 의 reason
  `ir_mismatch` · head_tags 와 단계 로그의 문장이 알린다
- **계약 작성자는 produced 조건을 건다** (QA S8 을 받은 자리) — `runctl example bake` 의 계약은 build 에 produced ["manifest"], merge 에
  produced ["merged"] 를 건다 (`internal/contract/examples/bake.json:29` ~ `:32`). 조건이 없는 단계는 `runctl lint` 가 경고하고
  (`cmd/runctl/shape.go:104` · 「no condition judges step」) 두 조건을 제안한다 (contract-grammar 규칙 9절)
- **노드 쪽 오류는 오늘의 `runtime open:` · `workspace:` 와 같은 등급이다** — 완주가 아니다
- **FAILED 인 갈래는 build 칸을 채운다** — 계약의 명령이 하나라도 돌았으면 늘 (15절)
- 표의 「trash」는 세션의 runRoot (upper 포함) 와 building 때 만든 대기 자리 둘이다. 치우는 몸통이 대기 자리를 옮긴다 (11절)
- **Finalize 예산은 pending 쓰기까지 덮는다** — 마감을 넘겼으면 pending 을 쓰지 않고 `Keep` 으로 옮긴 upper 째 몸통이 trash 로 보낸다. 오늘
  `settle` 이 그 갈래를 `finalize_timeout` 으로 낸다 (`finalize.go:212` ~ `:215`)
- **pending 을 쓴 뒤 업로드 예산을 넘기면 build 단계가 몸통을 곧바로 부른다** — FAILED 로 Run 이 끝나므로 merge 가 오지 않는다. 광고
  응답이 임대가 사라진 것을 볼 때까지 기다리지 않는다

---

## 7. merge 의 차례 (계획 3.8 · 받는 일 6 · 13 · 17 · 30 · 31 · 42)

```text
   1   거절 확인            Baker 가 없다 -> 거절.  상태 자리 · 잠금 파일 · state.json 을 못 엶 -> FAILED (1절 — held 가 있으면 몸통)
   2   claim 확인           7.1
   3   마감                 claim 을 받은 때 (노드 시계) + MergeWait
   4   배타                 Exclusive(ctx = runCtx 에 마감을 건 것, 10초, watch) — 8절
   5   그물                 ForeignMounts — 찾으면 배타를 놓고 60초 뒤 4 로 (8.4)
   6   trash                <대기 upper 의 scratch>/trash 를 MkdirAll (0700) — Preflight 가 진짜 디렉터리를 요구한다.
                            실패 -> 배타 Release · 몸통 · FAILED (결정 44)
   7   helper preflight     10절.  어긋남 · 읽기 실패 -> 배타 Release · 몸통 · FAILED
   8   합치기 시작 표지      startMerging — 몸통이 먼저 돌았으면 배타 Release 하고 끝 (보고는 닿지 않는다 — 임대가 끝났다)
   9   merging              WriteState(merging).  못 쓰면 합치기 전이다 — 표지를 거두고 배타 Release · 몸통 · FAILED
   10  helper apply         merge.Apply.  ctx 로 끊지 않는다 (본체에 상한 없음)
   11  upper 뿌리 rmdir      Apply 가 끝났다는 표지를 겸한다 (12.4)
   12  metadata             WriteMetadata(초안 + bake{run · node · merged_at · resumed false · previous_ir}) — 합치기의 마지막 동작
   13  committed            WriteState(committed)
   14  정리                 대기 자리 (빈 자리와 초안) 를 그 scratch 의 trash 로 Trash.Move 한 번 — committed 뒤다 (2절)
   15  놓기                 배타 Release · 굽기 Release · DropBake · Baker.held 를 비운다
   16  보고                 $OUT/merged · 업로드 · 보고 DONE (merge 칸)
```

- **merging 을 쓴 뒤 멈추면 (10 ~ 12 의 오류)** — merging 에 둔다 · 배타 Release · 굽기 Release · DropBake · Baker.held 를 비운다 ·
  FAILED, 원인 코드 없음, error 에 호출과 경로 (답 2). 다음 광고에서 보통 그 노드 자신이 잇는다 (13절)
- **committed 를 못 쓰면 대기 자리를 옮기지 않는다** — 13 이 실패하면 14 를 건너뛴다. merging 인 동안 초안이 있어야 재개가 「끝났나」를
  대 본다 (12.4)
- **metadata 를 쓴 뒤의 오류 (13 · 14)** — 합치기는 끝났다. 경고 한 줄을 쓰고 15 · 16 으로 간다 — `$OUT/merged` 를 내고 DONE
  (되물음 5 답 A · (3)). committed 를 못 썼으면 merging 이 남고 재개가 metadata 와 초안을 대 보고 committed 만 쓴다 (12.4). 대기 자리를 못
  옮겼으면 버려진 자리다 — 기동의 committed 정리가 거둔다 (12.2 · 광고 주기의 committed 는 대기 자리를 안 치운다 — 13절). Record 가 합친 굽기를 실패로 말하지 않게 한다 (US-18 재개로 합쳐졌나)
- **`WriteMetadata` 는 lower-state 가 지은 그대로 부른다** — lower 뿌리의 옛 임시 파일을 먼저 지우고 0644 · schema 0 이면 1 · 호스트
  (노드 uid) 에서 부른다 (lower-state code-summary 9절 · 받는 일 42)
- **previous_ir 은 초안의 값이다** (`domain-entities.md` 4절)
- **merge.Result 를 로그에 칸 그대로 쓰고, merge 칸에는 셈 일곱으로 옮긴다** (`mergeOpsOf`). metadata 에는 싣지 않는다 — metadata 의
  칸은 결정 3-16 (metadata 가 담는 것) 그대로다
- **합치는 중 임대가 사라지면** — Apply 는 끝까지 간다. 보고는 닿지 않을 수 있다. lower 는 committed 로 가고 광고의 bake.run 이 그
  Run 이다 (완료 조건 6 재개로 합쳐졌나를 광고로 본다 — 와 같은 길)
- **데몬이 멈추면 (SIGTERM)** — 배타를 기다리는 중이면 몸통 (last_attempt `node stopped`) · 보고 없음. Apply 중이면 helper 를 기다려
  끝낸다 (하루치 1 ~ 2 초 · merge-rules code-summary 6절 1.11 초). SIGKILL 이면 helper 도 죽고 (`--kill-child`) merging 이 남는다 —
  재개가 잇는다. pending 을 쓴 뒤 merge claim 전에 멈추면 몸통이 안 돈다 — pending 이 남고 형제의 광고 주기가 치운다 (11절 · 13절 ·
  되물음 1 답 A)

### 7.1 claim 확인 (받는 일 10 · 17)

| 이 프로세스가 이 Run 의 굽기를 쥐었나 | state.json | 결과 |
|---|---|---|
| 예 | pending · 주인이 이 Run | 진행한다 |
| 예 | 그 밖 | FAILED `the lower state does not match this bake: <phase> (run <Run>)`. 상태를 건드리지 않고 굽기 잠금을 놓는다 · DropBake · Baker.held 를 비운다. last_attempt 는 남기지 않는다 — 상태가 이 굽기의 것이 아니다 |
| 아니오 | 주인이 이 Run 이고 pending · merging | FAILED `the pending upper of this run is left for the stale-bake cleanup; the node restarted or could not record the end of the build step` — 정리가 아직 못 돌았다 (상태 자리를 못 열었다 · TryBake 가 오류였다) |
| 아니오 | 지금 state 의 phase 와 무관하게 last_attempt 의 Run 이 이 Run 이고 reason 이 `abandoned: no process held the bake while it was pending` | FAILED `the pending upper of this run was discarded as a stale bake; the node restarted or could not record the end of the build step` — build 보고와 merge claim 사이에 재시작했다. `FailRestarted` 는 CLAIMED 단계만 실패시키므로 (`store/claim.go:599` ~ `:607`) PENDING 이던 merge 는 새 생이 받고, 그 전에 기동 정리나 형제의 광고 주기가 pending 을 치웠다 (되물음 1 답 A). 그 사이 다른 굽기가 building · pending · merging 이어도 last_attempt 는 남는다 (2절) |
| 아니오 | 그 밖 (committed — 명령 실패 · IR 어긋남 (last_attempt ir_mismatch) · 다른 굽기 · 주인이 이 Run 인 building · building 을 치운 abandoned 기록) | 합칠 것 없음 — DONE · merged 없음 · error 없음 · 단계 로그 `bake: nothing to merge; the build step left no pending upper` (되물음 2 답 B · 물음 1 답 B · 계획 3.8 그대로) |

- **merge 는 build 를 한 노드로만 간다** — 두 단계가 같은 역할이고 (`contract/bake.go:273`) 역할의 노드가 단계 행에 박힌다
  (`store/store.go:389` ~ `:393` · `store/claim.go:299`). 그 노드에서 다른 프로세스가 merge 를 받는 길은 재시작뿐이다
- **합칠 것 없음은 DONE 이다** (되물음 2 답 B · 계획 3.8 그대로) — 확인하고 할 일이 없었던 것은 완주다 (INVARIANTS.md:292). 판정은 계약의
  조건이 한다 (I3 · INVARIANTS.md:148 · `internal/store/claim.go:977` ~ `:984`) — merge 에 produced ["merged"] 를 건 계약은 그 조건으로
  FAILED 이고, build 에 produced ["manifest"] 를 건 계약은 build 의 조건에서 이미 FAILED 다 (contract-grammar 흐름 3절)
- **재시작 두 줄과 어긋남 줄이 FAILED 인 까닭 — 판정이 아니라 미완주다** (ADR-021 · INVARIANTS.md:292). build 단계는 DONE 으로 대기
  upper 를 남겼는데 재시작이 그것을 잃게 했다 — merge 단계가 제 입력을 잃어 끝까지 못 돈 것이고,
  `FailRestarted` 가 CLAIMED 단계에 붙이는 「node restarted; in-flight step cannot be trusted」 (`store/claim.go:599` ~ `:607`) 와 같은 뜻이다.
  어긋남 줄은 노드의 상태가 이 굽기와 맞지 않아 돌리지 않은 것이다. I3 에 걸리지 않는다
- **재시작 줄은 pending 갈래뿐이다** (결정 52 · 마지막 QA) — 대기 upper 는 pending 을 쓴 뒤에만 있다. 주인이 이 Run 인 building 이나
  building 을 치운 abandoned 기록은 명령이 실패한 뒤 몸통이 committed 를 못 쓰고 놓은 자리다 — 이 Run 에는 대기 upper 가 없었으므로
  「the pending upper of this run …」은 거짓이 된다. 합칠 것 없음 (DONE) 으로 간다. build 단계에서 재시작하면 build 가 CLAIMED 라
  `FailRestarted` 가 Run 을 닫아 merge claim 이 오지 않는다 (`store/claim.go:599` ~ `:607`)
- **합칠 것 없음과 겹치지 않는다** — 재시작 줄은 「주인이 이 Run 이고 pending · merging」 이거나 「last_attempt 가 이 Run 이고 pending 을 치운
  abandoned」 일 때이고, 합칠 것 없음은 그 밖이다. 명령 실패 뒤의 몸통은 reason 이 `sync exited 1` 같은 글자라 abandoned 로 시작하지 않고,
  몸통이 committed 를 못 쓴 뒤의 정리는 building 을 치운 기록이다. 표는 위에서부터 처음 맞는 줄이라 재시작 줄이 먼저다 (결정 50 · 52).
  reason 의 글자는 bakerule.go 의 상수 하나가 짓고 7.1 이 같은 상수로 대 본다
- 명령 실패 뒤의 merge 가 이 줄에 닿으려면 몸통이 `Baker.held` 를 비워야 한다 (11절) — 남아 있으면 둘째 줄 (어긋남) 로 잘못 간다
- 첫째 줄 밖의 넷은 7 의 앞이라 굽기 잠금을 새로 잡지 않는다. 표는 위에서부터 처음 맞는 줄이다
- **남는 가장자리** — 재시작 뒤 끼어든 다른 굽기가 합쳐 last_attempt 를 지웠으면 exit 0 인 build 의 merge 도 「nothing to merge」로 DONE 이
  된다. 그때 Run 의 결과는 계약의 조건이 정한다 — merge 에 produced ["merged"] 가 걸려 있으면 FAILED, 없으면 SUCCEEDED 이고 lint 가 그 계약을
  경고했다 (되물음 2 답 B)

---

## 8. 기다림 — 시계 · 간격 · 로그 · 그물 (계획 3.9 · 완료 조건 5 merge 가 누구를 언제까지 기다리나 · US-12 · US-15 · 받는 일 35 ~ 38 · 49)

### 8.1 시계

- **상한 시각은 노드 시계다.** 마감을 지키는 쪽이 노드이고, 로그는 노드가 언제 포기하는지를 말해야 한다. 한 줄에 마감 시각
  (UTC · `node clock`) 과 남은 시간을 함께 쓴다 — 남은 시간은 두 기계의 시계 차이와 무관하다. 진행 조회의 phase_since 는 Mediator
  시계 그대로다 (Application Design Q4 논의)

### 8.2 간격

- `Exclusive` 는 1초마다 다시 걸고 (lower-state `pollEvery`) watch 를 10초마다 부른다 (막혀 있는 동안 처음과 every 마다 ·
  `lock_linux.go:157`)
- 줄을 쓰는 때 — 처음 · 쥔 쪽의 모습 (노드 · 역할 · Run · acks · 기록 없는 쥔 쪽) 이 바뀔 때 · 그 밖에는 5분마다 한 번. 4시간이면
  바뀜을 빼고 48 번이다

### 8.3 줄 (영어)

```text
   waiting for the lower lock; deadline 2026-09-27T09:00:00Z node clock (3h52m left)
     node box-a holds it for run R-8790 since 2026-09-27T04:10:00Z
     node box-b holds it as a candidate; its drain was acknowledged 1 of 2 times
     a holder left no record (an env check smoke, or an older enode that takes no lower lock)
   took the lower lock after 41m12s
```

- 노드 이름은 `Holder.Label` (없으면 node_id) 이다. 기록 없는 쥔 쪽 줄은 `Waiting.Unnamed` 일 때다 — 살아 있는 기록이 0 인데 막혔다
  (lower-state code-summary 9절)

### 8.4 그물 — `ForeignMounts` (lower-state 규칙 8.2)

- **배타를 잡은 뒤 · 시작 전 확인 앞에 부른다.** 찾으면 배타를 놓고, 찾은 마운트마다 한 줄 (pid · namespace · 마운트 자리 ·
  lowerdir) 을 쓰고, 60초 뒤 다시 배타를 기다린다 (마감은 그대로). 곧바로 다시 잡으면 잠금을 모르는 옛 판 노드의 마운트를 1초마다
  훑는다
- **그물이 오류면** (자기 mountinfo 를 못 읽음) 경고 한 줄을 쓰고 합친다 — 증거는 배타 잠금이다 (lower-state 규칙 8.1). 못 읽은
  프로세스(`Unreadable`)가 있으면 그 수를 한 줄 쓴다
- 찾은 것은 실패가 아니다 — 재개에서도 재시도 간격(10분)을 걸지 않는다

```text
   found 1 overlay mount of this lower in another mount namespace; releasing the lock and waiting 60s
     pid 41822 namespace mnt:[4026532871] mount /home/u/scratch/enode-runc-x/merged lowerdir /home/u/scratch/enode-runc-x/lower-ro
   cannot scan mounts of other processes: <원인>; merging on the lower lock alone
   2 processes could not be read while scanning mounts
```

### 8.5 끝나는 셋

| 끝 | 알아보는 법 | 결과 |
|---|---|---|
| 마감 | ctx 의 마감이 지났다 | 몸통 · FAILED `gave up waiting for the lower lock after 4h0m0s (merge.wait)` · reason `merge_wait_timeout`. build 의 DONE 은 그대로라 Record 에서 둘이 따로 보인다 (완료 조건 9) |
| 임대가 사라짐 | runCtx 가 임대 감시로 끝났다 (`claim.go:667` 의 cancel) | 몸통 · FAILED `aborted: lease expired` |
| 데몬이 멈춤 | Worker 의 ctx 가 끝났다 | 몸통 · 보고 없음 |

- 셋 모두 upper 는 trash · committed · drain 이 다음 광고에서 풀린다 (ADR-077:265 ~ :268 대기 상한)
- 마감의 문장은 `MergeWait` 의 `time.Duration` 글자다 — 기본이면 `4h0m0s`, 계약이 `2s` 면 `2s`

### 8.6 어디에 쓰나

- **merge 단계의 줄은 단계 로그다** — 버퍼와 진행 청크(`w.transcript`) 둘로 간다. 굽기 담당이 기다리는 동안 Mediator 에서 본다
  (US-15 · 받는 일 49). 끝에 오늘처럼 `UploadLog` 로 올린다
- 노드 로그에는 셋만 — 기다리기 시작 · 잡음 · 마감 (노드 · Run · 마감 시각)
- **재개도 같다** — 마감이 없어 마감 줄 대신 `waiting for the lower lock to resume the merge of run <Run>` 이고, 줄은 노드 로그로 간다
  (단계가 없다)

---

## 9. merge-helper 와 trash (계획 3.10 · 받는 일 23 ~ 26 · 32)

- **입구** `enode merge-helper` 를 trash-helper 와 같은 모양으로 연다 — `unshare --user --map-root-user --map-auto --fork --kill-child`,
  `--mount` 없음 (`trash_linux.go:104`). namespace 안의 root 로 돌고 권한을 내려놓지 않는다 (merge-rules)
- **한 합치기에 두 번 연다** — preflight · apply. 그 사이에 호스트가 merging 을 쓴다. 한 번 여는 데 약 5 ms 다 (trash code-summary 6절).
  처음과 재개가 같은 두 줄이다 (merge-rules 가 넘긴 일)
- 입력은 stdin 의 JSON 한 줄, 출력은 stdout 의 JSON 한 줄이다 (`domain-entities.md` 8절)
- **trash 는 대기 upper 가 있는 scratch 의 trash 다** — `pending_upper` 에서 네 단계 위. 재개하는 노드가 다른 scratch 를 써도 같은 자리로
  간다. Preflight 전에 `MkdirAll` 로 만든다 (7 의 6). 치우는 몸통 · 기동 정리가 대기 자리를 보내는 trash 도 이것이다 (계획 3.13) — 다른
  마운트의 trash 로는 rename 이 EXDEV 다
- **`Discard` 는 `scratch.Trash.Move` 로 항목마다 한 번** — 합치기 하나의 항목을 한 디렉터리에 모으지 않는다. 삭제자는 항목이 쓰이는
  중인지 모르고 도는 단계가 있어도 10분마다 깬다 (`scratch/deleter.go:48` ~ `:60`) — 모은 디렉터리는 채우는 중에 지워질 수 있다.
  낱개의 값 — Move 는 merge-helper 안의 rename 이라 helper 를 더 열지 않는다. 비용은 뒤의 삭제다: 항목마다 trash-helper 하나
  (약 5 ms) 라 하루치 526 항목이면 배경에서 약 2.6 초다
- 같은 끝 조각의 항목이 여럿이면 trash 이름이 `-1` · `-2` 로 늘어난다 (`scratch/trash.go:43` 의 Move)
- **Apply 뒤** 빈 upper 뿌리를 호스트에서 rmdir 하고 (뿌리는 helper 가 0700 으로 만든 노드 uid 소유다 · merge-rules 가 넘긴 일),
  committed 를 쓴 뒤에 대기 자리 (초안 포함) 를 `Trash.Move` 한 번으로 옮긴다
- **trash 를 비우는 쪽** — 합친 노드의 삭제자가 자기 scratch 의 trash 를 비운다. 재개가 다른 scratch 의 trash 에 넣은 항목은 그
  scratch 를 쓰는 노드의 삭제자가 비운다 (같은 scratch 를 나눠 쓰는 형제 · 그 노드의 재시작)

---

## 10. 시작 전 확인 — 마운트 줄과 어긋났을 때 (계획 3.11 · lower-state 답 6 · 받는 일 27 · 28 · 34 · 51)

- **마운트 줄을 더한다** — `merge_linux.go` 의 `resolve` 에서 셋의 `STATX_MNT_ID` 를 읽고 st_dev 확인 (`:86`) 다음에 댄다. 새 Check
  `mount` · 문장 `merge preflight: upper, lower and trash must share one mount (upper mount <n>, lower mount <n>, trash mount <n>);
  a bind alias of the lower cannot take a rename`. 번호를 못 읽으면 `*PreflightError` 가 아니라 확인을 못 한 오류다 (`domain-entities.md` 9절).
  정본은 이미 「한 마운트포인트 안」을 요구한다 (ADR-077:133) — 코드가 그것을 지키게 한다
- **Preflight 가 스스로 확인해야 재개 때도 막힌다** — env check 의 점검 셋은 그 노드의 scratch 와 워크스페이스를 보지만, 재개는 다른
  노드의 scratch 에 있는 대기 upper 를 합친다

```text
   때                   *merge.PreflightError                                          그 밖의 오류 (읽기 실패 · helper 를 못 띄움)
                        (path · directory · overlap · filesystem · mount · mark)
   merge 단계 (pending)  merging 을 안 쓴다.  lower 는 그대로다 — 배타 Release ·          같다.  다시 해 보지 않는다 (노드 쪽 오류 · 6절)
                        몸통 (upper 를 trash · committed · 굽기 Release · DropBake ·
                        last_attempt reason 은 그 문장) · FAILED (error 에 문장 ·
                        원인 코드 없음)
   재개 (merging)        upper 를 버리지 않는다 — lower 가 반쯤 바뀌었을 수 있다.          같다 (답 2)
                        merging 에 둔다 · 두 잠금을 놓는다 · 그 노드는 10분 뒤 (답 2)
```

- **merge 단계에서 Preflight 를 merging 보다 먼저 두는 까닭** — Preflight 는 아무것도 바꾸지 않는다. merging 을 쓴 뒤에 어긋나면
  committed 로 돌릴 길이 없다 (원칙 첫째). lower-state 흐름 5절은 merging 을 먼저 그렸다 — 뒤집었다 (흐름 13.3)
- merge-rules 가 넘긴 「읽기 실패는 다시 해 볼 일」(받는 일 28) 은 재개에서 받는다 — merge 단계는 pending 이라 다시 해 볼 Run 이 없다.
  Run 은 FAILED 로 끝나고 계기가 다시 낸다
- 워크스페이스가 bind 별칭인 노드는 다른 노드의 대기 upper 를 재개하지 못한다 — 마운트 줄에 늘 걸린다. 그 노드의 재시도는 10분마다
  로그 없이 (오류가 같으므로) 되풀이되고, 다른 노드가 잇는다

---

## 11. 치우는 몸통 · HoldBake · DropBake (계획 3.12 · 받는 일 22 · 40 · 41 · 52)

- **HoldBake 는 pending 을 쓴 바로 뒤** — lower-state 의 차례 그대로다 (`lowerguard.go:490` 의 주석 · 배타를 기다리기 전). 그 전 building
  동안 이 노드가 공유를 candidate 로 다시 쥐는 것은 해가 없다 (계획 2.3 · `lowerguard.go:170`) — 굽기 잠금이 다른 굽기와 재개를 막고,
  배타를 기다리는 쪽이 없다. HoldBake 가 놓는다. 팩 결정 3-9 (prepare 를 담은 Run 은 공유를 잡지 않는다) 와 다르다 — 정본 되돌림 (흐름 13.1)
- **DropBake 는 HoldBake 뒤 굽기 잠금을 놓는 모든 길의 끝** — 합침 · 대기 상한 · 임대가 사라짐 · Preflight 어긋남 · merging 뒤 멈춤
  (답 2) · pending 뒤 예산을 넘긴 build · 치우는 몸통 · 7.1 의 어긋남. HoldBake 전의 길(building 실패)에서 불러도 해가 없다 — 표지를
  새로 적는데 lower 가 안 바뀌었으므로 같은 표지다. 계획 3.12 는 「HoldBake 뒤」만 적었다
- **치우는 몸통은 굽기 하나에 하나다.** `heldBake` 의 mutex 와 「합치기 시작」 표지로 한 번만 돈다. 광고 응답의 몸통
  (`bakeGoneLocked`, `lowerguard.go:419`) · build 단계의 실패 끝 · merge 단계의 실패 길이 같은 함수를 부른다. merge 단계는 merging 을
  쓰기 전에 같은 mutex 아래에서 표지를 적고, 몸통이 먼저 돌았으면 거기서 멈춘다. 몸통은 표지가 있으면 아무것도 안 한다 — 합치기
  본체를 끊지 않는다
- **몸통이 하는 일** — 대기 자리를 그 자리의 scratch 의 trash 로 · `WriteState(committed)` + last_attempt · 굽기 Release · DropBake ·
  `Baker.mu` 아래에서 `Baker.held` 를 비운다 (그 굽기일 때만). 소유자가 굽기 Run 을 취소한 길이 이것이다 (Application Design Q6 논의 ·
  소유자의 수단은 굽기 Run 취소) — upper 는 trash · committed · drain 이 다음 광고에서 풀린다. held 를 비우지 않으면 그 노드는 다음
  merge 를 7.1 둘째 줄로 실패시키고 모든 굽기를 bake_in_progress 로 거절한다
- **held 가 있는 동안의 모든 오류 길은 몸통으로 끝난다** (결정 44 · QA 재검) — build 의 MkdirTemp · building 쓰기 · 세션 · bash 확인 ·
  명령 · 대조 · pinned · 닫기 · 초안 · pending 쓰기의 오류, merge 의 ReadState · trash 만들기 · Preflight 의 오류, 대기 상한 · 임대 · 데몬 멈춤.
  길 하나라도 몸통을 건너뛰면 `g.bake` 와 held 가 남고, 광고 응답의 몸통은 pending 이 아니면 `g.bake` 를 비우지 않아 (`lowerguard.go:424` ~
  `:425`) 그 노드는 재시작까지 모든 굽기를 거절한다. 예외는 merging 을 쓴 뒤 (7절 — merging 에 둔다) 와 7.1 둘째 줄 (상태가 이 굽기의 것이
  아니다 — 잠금만 놓고 held 를 비운다) 둘이다
- **몸통의 쓰기가 실패하면** 할 수 있는 데까지 한다 — 대기 자리를 못 옮기면 노드 로그 경고 한 줄 뒤 committed 를 쓴다 (그 자리는
  버려진 것이고 기동의 committed 정리가 거둔다 · 12.2). committed 를 못 쓰면 경고 한 줄 뒤 잠금을 놓는다 — state.json 은 낡은 상태가
  되고, 기동이나 형제의 광고 주기가 낡은 상태로 치운다 (13절 · 되물음 1 답 A)
- **building 에서 임대가 사라지면 build 단계가 끝에서 치운다** — 세션이 upper 를 쓰는 중이다 (lower-state 규칙 10절). 광고 응답의 몸통은
  pending 일 때만 돈다 (`bakeGoneLocked` 가 phase 를 본다)
- **pending 을 쓴 뒤 merge claim 전에 데몬이 멈추면 (SIGTERM) 몸통이 안 돈다** — 굽기를 쥔 프로세스가 사라지고 pending 이 남는다.
  광고 응답의 몸통은 그 프로세스 안에서만 돈다 (`lowerguard.go:419` ~ `:433`). 굽기 잠금이 풀렸으므로 형제의 광고 주기가 TryBake 로 잡고
  낡은 pending 으로 치운다 (13절 · 되물음 1 답 A)
- **재개는 HoldBake · DropBake 를 부르지 않는다** (`domain-entities.md` 10절). DropBake 는 표지를 지금 metadata 의 것으로 새로 적는다
  (`lowerguard.go:504` ~ `:513`). 재개하는 노드가 합친 뒤에 부르면, 합치기 전에 그 노드에 매칭된 늦은 임대가 표지 비교
  (`lateLocked` · `:398`) 를 통과해 바뀐 lower 위에서 돈다. 부르지 않으면 표지가 합치기 전의 것으로 남아 그 임대는 lower_changed 로
  거절된다. HoldBake 는 그 짝이라 부르지 않는다 — merging 동안은 어느 노드도 공유를 새로 잡지 않으므로 (광고 `:170` · `:215` ·
  늦은 임대 `:396` · 기동 `:95` ~ `:100`) HoldBake 가 놓을 공유도 없다

---

## 12. 기동 — 낡은 상태 정리와 재개 (계획 3.13 · ADR-077 §7 · 결정 3-12 주인이 살아 있으면 bake_in_progress · 3-13 시작할 때 재개 · 받는 일 50 · 광고 주기도 같은 표 — 되물음 1 답 A)

### 12.1 자리

`cmd/enode/main.go` 의 `StartLowerGuard` (`:249`) 뒤 · `SweepOrphanSessions` (`:312`) 와 삭제자 첫 회 앞 · 광고 시작 (`:329`) 전.
조립은 `internal/enode` 의 `StartBaker` 하나이고 main.go 에는 부르는 줄만 둔다 (`cmd/enode` 80.5% · 하한에 가깝다 · lower-state
code-summary 5절 표 `:144`). 대기 자리는 `enode-runc-` 로 시작하지 않으므로 기동 청소가 건드리지 않는다 (`trash_linux.go:23` · `:202`).

- **`StartBaker` 의 첫 줄이 `guard.OnStale(baker.onStale)` 등록이다** — 굽기 잠금을 못 잡아도, 자리를 못 열어도 광고 주기의 정리와 재개는
  열려 있어야 한다
- 자리를 못 열었거나 (`guard.Dir()` 이 nil) TryBake · ReadState 가 오류면 로그 한 줄 (원인이 바뀔 때만) 과 함께 기동 정리를 건너뛴다

### 12.2 TryBake 가 되면

| state.json | 하는 일 |
|---|---|
| committed | 자기 scratch 의 `pending/<lower 키>/` 아래 모든 항목을 trash 로 · 놓는다 |
| building · pending | 기록된 대기 자리를 그 자리의 scratch 의 trash 로, 자기 scratch 의 `pending/<lower 키>/` 아래 나머지를 자기 trash 로 · `WriteState(committed)` + last_attempt · 놓는다 |
| merging | 자기 scratch 의 `pending/<lower 키>/` 아래 기록된 자리 밖의 항목을 trash 로 · 재개를 배경에 연다 — 잠금은 재개가 쥔다 |

- **안 되면 다른 노드가 쥐었다** — 그쪽이 한다. **먼저 잡은 하나가 한다** 는 flock 이 준다 (ADR-077:289 · 받는 일 50)
- **광고 주기도 이 표를 쓴다** (되물음 1 답 A · 13절) — 주인이 죽은 building · pending · merging 셋 모두. 다만 committed 일 때 자기 scratch 의
  버려진 대기 자리를 치우는 첫 줄은 기동만 한다 — 광고 주기는 committed 를 보면 TryBake 를 하지 않는다
- **대기 자리를 보내는 trash 는 그 자리가 있는 scratch 의 trash 다** (계획 3.13) — 기록된 자리가 다른 노드의 scratch 에 있을 수 있고,
  다른 마운트의 trash 로는 rename 이 EXDEV 다
- **committed 인데 대기 자리가 남는 때** — MkdirTemp 뒤 building 쓰기 전에 죽었다 · committed 를 쓴 뒤 옮기기 전에 죽었다 · 옮기기가
  실패했다. 굽기 잠금을 쥐었으므로 도는 굽기가 없다 — 그 lower 키 아래는 모두 버려진 것이다
- **낡은 상태 정리의 last_attempt** — reason `abandoned: no process held the bake while it was <phase>` · builds 는 초안이 있으면
  (pending) 거기서, 없으면 비운다 · Run 은 옛 주인. merge claim 이 이 reason 으로 자기 Run 을 알아본다 (7.1). 주인이 죽었는지 committed 를
  못 쓰고 놓았는지는 정리하는 쪽이 모르므로 문장은 둘 다 참인 글자다 (결정 49 · 처음 판은 「stopped」였다)
- **기록된 대기 자리가 이미 없으면 옮겨진 것으로 보고 committed 를 쓴다** (결정 46 · QA 재검 B1). 다른 오류로 못 옮겨도 committed 를 쓴다 —
  몸통과 같다 (13절 · 결정 53). 자리가 없는 까닭은 몸통이 옮긴 뒤 committed 를 못 썼거나,
  옮기기와 committed 쓰기 사이에 죽었다. 없는 자리를 옮기려다 나는 ENOENT (`internal/scratch/trash.go:54` ~ `:60`) 를 실패로 치면 pending 이
  영영 남는다
- build claim 이 낡은 building · pending 을 만나면 같은 몸통으로 정리하고 진행한다 (3 의 2). 형제의 광고 주기가 먼저 치웠으면 committed 를 본다
- **pending_upper 의 모양을 확인한다** — `<scratch>/pending/<이 lower 의 키>/<이름>/upper` 인 절대 경로가 아니면 그 자리를 옮기지도 합치지도
  않는다. 정리면 경로를 로그에 남기고 상태만 committed 로, 재개면 실패로 친다 (13절 · 10분 뒤)

### 12.3 재개의 차례

```text
   1   배타                 Exclusive(데몬의 ctx · 마감 없음, 10초, watch) — 줄은 노드 로그 (8.6)
   2   그물                 ForeignMounts — 찾으면 놓고 60초 뒤 1
   3   모양                 pending_upper 의 모양 (12.2 끝 줄).  틀리면 실패
   4   초안                 대기 자리의 bake.json 을 읽는다.  없거나 못 읽으면 실패 (Apply 를 시작하지 않는다)
   5   끝났나               metadata 의 bake.run 이 주인 Run 이고 source.synced_at · source.head 가 초안과 같으면 10 으로 (12.4)
   6   upper 뿌리           없으면 9 로 (Apply 와 rmdir 이 끝났다)
   7   합치기               trash 를 MkdirAll · helper preflight (어긋남 · 읽기 실패 -> 실패 · 10절) · helper apply
   8   upper 뿌리 rmdir
   9   metadata             초안 + bake{run: 주인 Run · node: 초안의 Node · merged_at: 지금 · resumed true · previous_ir: 초안}
   10  committed            WriteState(committed)
   11  정리                 대기 자리를 그 scratch 의 trash 로 Trash.Move
   12  놓기                 배타 Release · 굽기 Release  (HoldBake · DropBake 는 안 부른다 · 11절)
```

- 실패하면 (3 · 4 · 7 · 8 · 9 · 10 의 오류) — merging 에 둔다 · 두 잠금을 놓는다 · 이 노드의 `retryAt` 을 10분 뒤로 · 오류가 바뀔 때만
  로그 한 줄 (13절). 11 의 오류는 경고 한 줄이다 — 합치기는 끝났고 committed 도 썼다. 남은 자리는 버려진 것이다 (12.2)
- 데몬이 멈추면 — 기다리는 중이면 두 잠금을 놓고 끝 (merging 그대로). Apply 중이면 helper 를 기다린다. main 이 `Baker.Wait` 로 기다린다

### 12.4 끝났는지 먼저 본다

```text
   초안이 없다                                            실패 — 사람이 봐야 한다.  초안은 pending 보다 먼저 fsync 로 디스크에 있고
                                                         committed 를 쓴 뒤에만 옮기므로 (2절), merging 인데 없으면 사람이 지웠거나
                                                         디스크가 깨진 것이다.  사람의 수단은 state.json 을 고치는 것뿐이다
   metadata 의 bake.run 이 주인 Run 이고                   합치기와 metadata 가 끝났다 (그 뒤에 죽었다) — committed · 정리
   source.synced_at · source.head 가 초안과 같다
   그 밖이고 upper 뿌리가 없다                             Apply 와 rmdir 이 끝났다 — 9 (metadata) 부터
   그 밖이고 upper 뿌리가 있다                             7 부터.  비어 있어도 된다 — 빈 upper 의 Apply 는 할 일이 없다
```

- **run_id 한 칸으로 「끝났다」를 정하지 않는다** — runs.run_id 는 Mediator DB 하나 안에서만 겹치지 않고 (`store/schema.sql:23`) 제출자가
  적는 값이다 (`contract.go:1087`). 다른 Mediator 에 붙은 노드도 같은 lower 에 있을 수 있다. 같은 run_id 를 다시 쓴 lower 에서 끊긴 합치기를
  「끝났다」로 보면 조용히 망가진다 — 초안의 synced_at 과 head 까지 대 본다

### 12.5 재개의 metadata

- resumed true · run 은 주인 Run · node 는 주인 노드 (그 굽기를 한 노드 · US-6 lower 를 마지막으로 바꾼 굽기) · merged_at 은 재개가 끝난
  때 · previous_ir 은 초안의 값 (`domain-entities.md` 4절 — 반쯤 합친 lower 에서 읽지 않는다)
- bake.node 는 합친 노드가 아니라 구운 노드다 — 정본 §5 의 예시만으로는 두 뜻이 구별되지 않는다 (흐름 13.1)
- 재개한 노드는 자기 노드 로그에 남는다 — `resumed the merge of run <Run>` (재개한 노드 · 셈 · 걸린 시간)
- 원래 Run 은 FAILED 로 봉인됐다 — 재시작 (ADR-030) 이거나 merge 단계의 Apply 오류 (답 2). 광고의 bake.run · bake.resumed 가 둘을
  잇는다 (완료 조건 6 · US-18)

---

## 13. 살아 있는 동안 정리와 재개 (답 2 · 되물음 1 답 A · 받는 일 29)

```text
   여는 쪽                 조건                                                      하는 일
   광고 주기                state.json 이 building · pending · merging ·                   TryBake.  되면 state 를 다시 읽고 12.2 의 표대로
   (LowerGuard.OnStale)     이 프로세스가 굽기를 안 쥐었고 재개 중이 아니다 ·                  (building · pending -> 정리 · merging -> 재개 ·
                           retryAt 이 지났다 (되물음 1 답 A)                              committed -> 놓는다).  안 되면 아무것도 안 한다
   build claim             TryBake 가 됐고 state.json 이 merging                          잡은 잠금을 넘겨 재개를 배경에 연다 · build 는
                                                                                       bake_in_progress (되물음 4 답 A)
   기동                    TryBake 가 됐고 state.json 이 committed 가 아니다               12.2
```

- **TryBake 뒤 state 를 다시 읽는다** — 광고가 읽은 state 는 잠금 전의 것이다. 그 사이 다른 노드가 정리나 재개를 끝냈을 수 있다
- **주인이 살아 있으면 TryBake 가 안 되므로 산 굽기를 건드리지 않는다** — building 인 굽기의 주인은 building 부터 committed 까지 잠금을 놓지
  않는다 (3절). 형제는 굽기 동안 광고마다 TryBake 한 번을 헛되이 한다 — flock 하나다
- **정리는 몸통과 같이 할 수 있는 데까지 한다** (결정 53 · 마지막 QA) — 대기 자리를 못 옮기면 로그 한 줄 뒤 committed + last_attempt 를 쓴다.
  남은 자리는 버려진 것이고, 그 scratch 를 쓰는 노드의 기동 정리 (12.2 의 committed 줄) 가 거둔다. 자리가 이미 없으면 옮겨진 것으로 본다
  (결정 46). committed 를 못 쓴 것만 실패로 치고 재개와 같은 간격이다 — 그 노드는 10분 뒤 · 로그는 오류가 바뀔 때만
- **광고 주기의 정리는 HoldBake · DropBake 를 부르지 않는다** — lower 를 안 바꾸므로 그 노드의 표지가 그대로 맞다. 굽기 잠금만 잡았다 놓는다
- **재시도 간격은 노드마다 10분이다** — 재개가 실패한 노드만 기다린다. 다른 노드는 자기 광고에서 해 본다. 하루치 upper 의 재시도는 한
  노드에서 시간당 Preflight 만으로 1.4 ~ 3.8 초다 (계획 물음 2 · Preflight 0.23 ~ 0.63 초) — 노드 N 개면 N 배이고, Apply 까지 가서 멈추는
  오류면 더 든다
- **merge 단계의 Apply 가 멈춘 노드는 10분을 기다리지 않는다** — `retryAt` 을 걸지 않으므로 다음 광고에서 곧바로 해 본다 (답 2 의
  「보통 그 노드 자신이 잇는다」). 그 재개가 실패하면 그때부터 10분이다
- **실패가 아닌 것** — TryBake 가 안 됨 · 그물이 찾음 · 배타를 기다림. 셋은 간격을 걸지 않는다. TryBake 의 오류는 1절처럼 읽기
  실패다 — 로그 한 줄 (원인이 바뀔 때만) 이고 간격은 걸지 않는다
- **로그는 오류가 바뀔 때만** — `resume failed; trying again in 10m` (run · err). 같은 오류가 되풀이되면 (예 — 매핑 밖 uid 로 chown ·
  merge-rules 규칙 9절) 한 번만 남고 lower 는 merging 에 머문다. 형제는 drain 이다 — 사람이 원인을 고치면 다음 시도에 풀린다
- **광고 루프를 막지 않는다** — `OnStale` 은 따로 돌고, TryBake 는 flock 한 번이다 (lower-state 의 `BeforeAdvert` 25 µs 안에 든다)
- **merging 동안은 어느 노드도 공유를 새로 잡지 않는다** — 굽기 drain 이 걸려 있다 (`lowerguard.go:189` 의 `sourcesLocked` · `:215`).
  merging 을 쓴 합치기가 이미 모두 놓게 했으므로 재개의 배타는 보통 곧바로 잡힌다 — 기다리는 것은 smoke 가 쥐는 몇 초와 기록 없는 쥔 쪽
  (옛 판) 뿐이다
- **building · pending 의 낡은 상태도 광고 주기가 치운다** (되물음 1 답 A · ADR-077:289 되돌림). 계획 3.13 의 근거로 적었던 「살아 있는
  데몬은 남기지 않는다 · 다음 굽기가 치운다」는 거짓이었다 — pending 을 쓴 뒤 merge claim 전의 SIGTERM 과 몸통의 쓰기 실패가 pending 을
  남기고 (11절), pending 이면 그 lower 의 모든 노드가 drain 이라 다음 굽기가 매칭되지 않는다 (`lowerguard.go:212` ~ `:216` ·
  `internal/store/queue.go:302` ~ `:308`). 소유자는 굽기 drain 을 풀 수 없다 (`lowerguard.go:145` ~ `:148`). 이제 두 광고 주기 (기본 120초)
  안에 풀린다 — 정리를 연 광고는 이미 drain 을 싣고 나가고 (`lowerguard.go:212` ~ `:214` 다음에 OnStale), 그다음 광고에서 풀린다

---

## 14. last_attempt (계획 3.14)

- **남기는 때** — building 을 쓴 굽기가 합치지 않고 끝난 모든 길 (6 · 8.5 · 10 · 11절) 과 낡은 상태 정리 (12.2). bake_in_progress 와
  거절은 굽기를 시작하지 않았으므로 안 남긴다. 7.1 의 둘째 줄은 state.json 이 이 굽기의 것이 아니므로 안 남긴다 — 쓰면 남의 상태를 덮는다
- **칸** — Run · 때 (노드 시계) · reason (원인 코드가 있으면 그것, 없으면 영어 한 줄) · builds (돈 것)

```text
   reason 의 예       ir_mismatch · merge_wait_timeout · finalize_timeout · upload_timeout
                     sync exited 1 · build config-a exited 2 · aborted: lease expired · node stopped
                     cannot verify ir: <원인> · cannot pin the manifest: <원인> · merge preflight: <문장>
                     cannot run the bake commands: the prepared rootfs has no bash
                     the bake run ended before the merge · abandoned: no process held the bake while it was pending
```

- **지우는 때** — 합쳐 committed 로 갈 때 (2절 · 되물음 5 답 A · (8)). lower-state 의 신원 판정 Reused 도 지운다 (lower-state 규칙 2절)

---

## 15. 결과 칸과 `$OUT` (계획 3.6 · 3.15 · 받는 일 5 · 6 · 14 · 15)

- **result 의 build 칸은 계약의 명령이 하나라도 돌았으면 늘 채운다** (step-phase 가 넘긴 물음 · 받는 일 15) — sync 의 기록 · 돈 builds ·
  head · ir (맞았을 때) · pinned · head_tags (대조가 HEAD 와 태그를 읽었을 때만 배열 · 그 밖은 null · 되물음 5 답 A · (11)). 완료 조건 9
  (build 의 성공이 따로 보인다) · 10 (IR 과 HEAD 의 차이) 이 이 칸을 본다
- **실패의 기록은 셋이다** — result 의 build 칸 · result 의 진단 칸 (effect prepare · 오늘 결과 확정 규칙 그대로) · state.json 의
  last_attempt. contract-grammar 흐름 3절이 넘긴 「실패의 기록은 result 진단과 last_attempt」 를 받는다
- **merge 칸은 합쳤을 때만** — ir (초안의 IR) · previous_ir · merged_at · resumed false · ops
- **`$OUT/manifest` 는 굽기가 성공했을 때만** — 모든 명령이 0 이고 IR 이 맞았다. 내용은 build 칸과 같은 JSON 이다
  (`contract.ArtifactManifest` · 받는 일 5)
- **`$OUT/merged` 는 합쳤을 때만** — 내용은 merge 칸과 같은 JSON 이다 (`contract.ArtifactMerged` · 받는 일 6)
- **build 단계의 `$OUT` 은 노드의 것이다** (되물음 5 답 A · (9)) — 계약이 build 에 out 을 적지 못하므로 (`contract/bake.go` 의 checkBuildStep)
  올릴 이름은 manifest 하나다. 명령이 `$OUT` 에 쓴 것은 올리지 않고, 굽기가 성공하지 않았으면 `$OUT/manifest` 를 지운 뒤 올린다 — 명령이
  쓴 manifest 가 build 의 조건을 참으로 만들지 못한다. merge 단계에는 명령이 없다
- 판정은 오늘의 produced 대조다 (`Verify` 불변 · contract-grammar 흐름 3절)

---

## 16. 오류와 로그 문구 (영어)

### 16.1 단계 보고 (result.error)

```text
   a bake step needs a node that writes to an upper (workspace.writes=isolated); this node writes to its workspace in place
   cannot open the lower state directory: <원인>                                      상태 자리 · 잠금 파일 · state.json 을 못 읽음
   another bake holds this lower: run <Run> on node <node> since <시각> (<phase>); submit the bake again later       reason bake_in_progress
   an interrupted merge of run <Run> is left on this lower; this node started resuming it; submit the bake again after it finishes
                                                                                     reason bake_in_progress · 되물음 4 · 5 답 A (17)
   cannot run the bake commands: the prepared rootfs has no bash, or sh cannot run there (exit <n>)
                                                                                     되물음 3 답 A — sync 전의 확인 (결정 45)
   runtime run: <원인>                                                                 명령 · 확인 도중 helper 가 죽음 (결정 48)
   cannot verify ir: the workspace has neither .repo/manifests nor .git
   cannot verify ir: git exited <n>: <stderr 의 마지막 줄>
   cannot pin the manifest: repo manifest -r exited <n>: <stderr 의 마지막 줄>
   cannot write the bake draft: <원인>
   cannot write the lower state: <원인>
   gave up waiting for the lower lock after <merge.wait> (merge.wait)                                              reason merge_wait_timeout
   aborted: lease expired
   the lower state does not match this bake: <phase> (run <Run>)
   the pending upper of this run was discarded as a stale bake; the node restarted or could not record the end of the build step
   the pending upper of this run is left for the stale-bake cleanup; the node restarted or could not record the end of the build step
   merge preflight: <merge 의 문장>
   merge stopped: <merge.OpError 의 문장>; the lower stays merging and a node on this lower resumes it
```

- bake_in_progress 의 주인은 state.json 의 owner 에서 읽는다 (US-16 다시 내면 되는 거절). owner 가 없으면 `run unknown`
- IR 어긋남은 이 목록에 없다 — DONE 에 reason `ir_mismatch` 만 싣고 문장은 단계 로그에 쓴다 (16.2 · 물음 1 답 B)

### 16.2 단계 로그 (버퍼와 진행 청크)

```text
   bake: sync started
   bake: sync exited 0 after 3m12s
   bake: ir <IR> matches HEAD <commit> (tags at HEAD: <tags>)
   bake: pinned the manifest to .enode-manifest.xml
   bake: build config-a started
   bake: build config-a exited 0 after 41m45s
   bake: skipping config-b; config-a failed                          답 3 · sync 가 실패했으면 "sync failed"
   bake: skipping config-a; ir <IR> did not match                    물음 1 답 B — 구성마다 한 줄
   bake: ir <IR> is not in the local repository after sync; the sync command must fetch that tag (HEAD is <commit>, tags at HEAD: <tags>)
   bake: ir <IR> points at <commit 1>, but sync left HEAD at <commit 2> (tags at HEAD: <tags>)
                                                                     어긋남의 두 갈래 — 단계 로그 끝 · reason ir_mismatch (물음 1 답 B)
   bake: the upper is pending the merge step
   bake: nothing to merge; the build step left no pending upper
   (merge 단계) 8.3 · 8.4 의 줄 · merged: <merge.Result 의 칸 그대로> in <걸린 시간>
   (merge 단계) the merge finished but cleaning up failed: <원인>; the next resume writes committed
```

- 단계 로그에 host 경로를 쓰지 않는다 — 대기 자리의 경로는 노드 로그에만 (그물의 줄은 마운트 자리를 보이는 것이 목적이라 예외다).
  계획은 이 규칙을 적지 않았다 — 결과 Record 로 가는 글자에 노드의 경로를 싣지 않는다

### 16.3 노드 로그

```text
   bake: holding the lower for run <Run>                               굽기 잠금을 잡았다 · building
   bake: cannot read the previous metadata; previous_ir is null         err
   bake: cleaned a stale bake                                          run · phase · 자리
   bake: cannot move the pending upper to trash; it is left for the start-up cleanup   path · err
   bake: cannot write the lower state while discarding the bake          err
   bake: resuming an interrupted merge                                 run · 주인 노드 · 여는 쪽 (start · advert · build claim)
   waiting for the lower lock to resume the merge of run <Run>         8.2 의 때 (재개)
   bake: resumed the merge of run <Run>                                셈 · 걸린 시간
   resume failed; trying again in 10m                                  run · err — 오류가 바뀔 때만
   bake: the pending upper path does not look like one this node writes; leaving it    path
   waiting for the lower lock (merge step)                             run · deadline — 시작 · 잡음 · 마감에 한 줄씩
```

---

## 17. 확장 준수 — Functional Design

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 계약 명령 줄 (격리 runtime 안에서만) 은 3 · 4절이 세션 안에서 도는 것으로 닫고, 확인은 이 유닛의 NFR 이다. 합치기 줄 (같은 filesystem · lower 밖으로 안 나감) 은 10절과 merge-rules 가 닫는다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 한 줄 — 저장소의 관례다 |
