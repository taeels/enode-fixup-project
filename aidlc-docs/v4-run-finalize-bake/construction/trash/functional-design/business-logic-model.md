# `trash` — 흐름

타입은 `domain-entities.md`, 규칙과 문구는 `business-rules.md` 에 있다. 여기는 그 규칙이 어느 자리에서 어느 순서로
도나, 조각 4 를 어떻게 확인하나, 그리고 다른 유닛에 무엇을 넘기나다.

---

## 1. 단계의 닫기 (runc-overlay · `Worker.afterExit` 의 닫기)

```text
   Worker                                   helper (runtime-helper · namespace 안)
   session.Finalize(fctx, spec)       -->   [Finalize 예산]
   session.Close(ctx0, Keep{})        -->   {op: close}  ->  unmount 넷.  지우지 않는다
                                      <--   {op: closed}
     stdin 닫기 · helper 가 끝나기를 기다린다 (오늘 그대로 · 3초에 죽인다)
     Trash.Move(runRoot)                      <scratch>/enode-runc-XXXX  ->  <scratch>/trash/enode-runc-XXXX
     lock.Release()                           rename 뒤
   finalized_at := now
   닫기가 마감 뒤에 끝났나?  -> settle 이 finalize_timeout 으로 적는다
   ... 업로드 · 보고 (finalize 유닛 그대로)
   w.report 가 끝나면 Deleter.Kick()
```

- `ctx0` 은 Worker 의 ctx 다 — 닫기는 마감으로 끊지 않는다 (답 1 = A)
- abort 뒤의 닫기(Finalize 가 helper 를 죽였다)는 helper 에 말하지 않고 `Trash.Move` 와 잠금 놓기만 한다 — finalize 유닛의
  abort 표시 그대로이고 그 안의 `RemoveAll` 이 `Move` 가 된다
- native 세션은 작업 폴더가 없어 닫기가 할 일이 없다 (오늘 그대로)

**Open 의 처음과 실패 갈래**

```text
   MkdirTemp(<scratch>, "enode-runc-")
   HoldSession(runRoot)                 곧바로.  실패하면 Move · 오류 (`runtime open:` 오늘 머리말)
   ... helper 띄우기 · open 요청
   실패하면  lock.Release 전에 Move · 그다음 Release     (1절 ① · 순서는 business-rules.md 1절)
```

---

## 2. 노드 기동 (`cmd/enode/main.go`)

```text
   설정 · 잠금 · client · 탐지 (오늘 그대로)
   runtime 이 runc-overlay 면
      trash := scratch.Trash{Dir: <scratch>/trash}
      scratch.Orphans(<scratch>, "enode-runc-", now)  ->  남은 것마다 trash.Move        (business-rules.md 3절)
      deleter := scratch.Deleter{Trash, Launch: enode.TrashLauncher(...), Changed: book.SetScratch, Every: 10분}
      go deleter.Run(ctx)                                곧바로 첫 회.  배경 (답 10 = A)
      worker.AfterReport = deleter.Kick
   go advertiser.Run(ctx)                                기다리지 않는다
   worker.Run(ctx)
```

- native 노드는 삭제자가 없다. 여유 부족 drain 은 native 에도 있다 (광고 주기 · 4절)
- `cmd/enode/main.go` 에 입구 하나 — `enode trash-helper <trash> <entry>` (runtime-helper 옆)

---

## 3. 배경 삭제자

```text
   Run(ctx)
     loop
        pass:
           names := trash.Entries()                          노드 uid 로 읽는다
           Usage 를 고친다 (항목 수 · 측정 안 된 수) -> Changed
           for name in names (차례로):
              Deleting = true -> Changed
              Launch(ctx, name, measured)                    helper 하나
                 measured(Size) 가 오면  -> sizes[name] = Size -> Changed
              결과마다 로그 (business-rules.md 4.3)
              지워졌으면 sizes 에서 뺀다
           Deleting = false · Usage 를 다시 고친다 -> Changed
        기다린다:  Kick | (항목이 남았으면) Every | ctx 끝
```

- `Kick` 은 크기 1 의 채널이다 — 보고가 여럿 겹쳐도 한 번 더 도는 것으로 합친다
- helper 를 못 띄우면 그 pass 를 멈추고 다음 깸까지 기다린다 (같은 원인은 한 번만 적는다)
- ctx 가 끝나면 도는 helper 를 죽인다 (`Launch` 의 ctx) — 반쯤 지운 항목은 다음 기동이 거둔다

**Launch (`internal/enode/trash_linux.go`)**

```text
   argv   unshare --user --map-root-user --map-auto --fork --kill-child -- <os.Executable> trash-helper <trash> <name>
   프로세스 그룹 · Pdeathsig — runtime-helper 와 같다 (데몬이 죽으면 같이 죽는다)
   stdout 줄을 읽는다   measured -> 콜백 · removed · error
   exit 을 기다린다     ctx 가 끝나면 그룹에 SIGKILL
```

**trash-helper (`RunTrashHelper`)**

```text
   ioprio_set(IDLE) · setpriority(19)
   scratch.CheckEntry(trash, name)       실패 -> {"error"} · exit 1
   scratch.Measure(trash, name)          -> {"measured": {...}}
   scratch.Remove(trash, name)           -> {"removed": true} · exit 0
                                         또는 {"removed": false, "left": [...]} · exit 1
```

---

## 4. 광고 주기의 drain (`Advertiser.Run`)

```text
   snap := Caps()                                   arch 키는 툴체인만 (여유 조건문을 지웠다)
   owner := policyReader.Read()                     오늘 그대로
   free, err := freeBytes(Local.Workspace)          워크스페이스가 있을 때만
   disk, ok := diskDrain(free, MinFreeGB, a.diskHeld)
   a.diskHeld = ok ;  걸기 · 풀기가 바뀌었으면 노드 로그 한 줄
   sources := [owner 출처?, disk 출처?]
   effective := combineDrain(sources)
   ad.Policy = {Drain: effective}
   book.SetCaps(snap) · book.SetDrain({effective, sources, now})
   POST /v1/nodes  ...  응답의 drain -> Held.SetDrain (오늘 그대로 · Worker 가 claim 을 멈춘다)
```

- 스스로 건 drain 도 오늘의 소유자 drain 과 같은 길을 지난다 — Mediator 가 받아 적고 응답으로 돌려주면 Worker 가 새 일을
  안 집는다. 여유를 되찾아 푼 광고 뒤에 Mediator 가 빈 drain 을 돌려주면 Worker 가 다시 집는다
- 여유 부족은 삭제자와 맞물린다 — trash 가 쌓이면 여유가 내려가 drain 이 걸리고, 삭제자가 비우면 다음 광고에서 풀린다
  (`services.md` 5절)

---

## 5. 상태 파일 (`statusBook`)

```text
   Advertiser  --SetCaps · SetDrain-->   statusBook   --값이 바뀌었으면-->  <stem>.status.yaml (tmp -> rename)
   Deleter     --SetScratch--------->        (mu)
   제어판       <------------------------------ ReadStatus
```

오늘의 `Advertiser.writeStatus`(탐지 시각이 바뀔 때만)가 `statusBook.SetCaps` 가 된다. 설정 경로가 없으면(시험) 쓰지 않는다.

---

## 6. 제어판이 읽는 흐름 (`internal/panel`)

```text
   state()
     상태 파일 읽기  ->  Caps (오늘) · Drain · Scratch
     프로세스        ->  Running (오늘)
     drain 칸:  Running 이고 상태 파일에 drain 이 있으면  effective · sources · DrainFrom="status"
                아니면                                      정책 파일의 값 · 소유자 출처 하나 · DrainFrom="policy"
     trash 칸:  Scratch 가 있으면
   page.go   출처 줄 · 풀기 버튼의 조건 · 남은 출처 문구 · trash 줄 (business-rules.md 8절)
```

---

## 7. 조각 4 의 확인 모양 (답 10 = A)

### 사람 (SunnyVM · 노트북 VM 이라 꺼져 있을 수 있다 — 그때는 보류)

Code Generation 이 `scripts/finalize-bake/slice-4.sh` 로 굳힌다. 보는 것은 `scene-gates.md` 2절과 `requirements.md` 6절이 바꾼 것이다.

```text
   ①  15만 파일 · 9 GB upper 를 남기는 단계와 작은 upper 를 남기는 단계 — finalized_at - exited_at 이 비슷하다
       (1초 안쪽).  오늘은 앞의 것에 upper 를 지우는 시간이 든다
   ②  보고 직후 <scratch>/trash 가 차 있고 곧 빈다.  상태 파일의 scratch 칸이 측정 · 지우는 중 · 빈 것으로 바뀐다
   ③  권한 000 인 work/work 와 subordinate uid 소유 항목도 지워진다
   ④  데몬을 SIGKILL 로 죽이고 재시작하면 남은 작업 폴더가 trash 로 가고 비워진다
   ⑤  min_free_gb 를 여유보다 크게 두면 GET /v1/nodes 에 draining · 제어판에 여유 부족 출처.
       그동안 arch.<이름> 키는 광고에 남는다.  min_free_gb 를 되돌리면(또는 여유를 되찾으면) 다음 광고에 풀린다
   ⑥  다섯 자리 모두 trash 로 간다 — Open 실패(없는 rootfs 로) · 보통 닫기 · abort (Finalize 예산을 넘기는 단계) ·
       helper cleanup (지우지 않는다) · env check 의 smoke
```

### 기계 (기본 `go test` · CI 에서 돈다)

```text
   Move          rename 한 번 — 옮긴 뒤 inode 가 같다 · 이름이 겹치면 -1 · trash 가 없으면 만든다
   Orphans       잠금을 쥔 폴더는 안 나온다 · 풀린 잠금은 나온다 · 잠금 파일이 없고 1시간 전이면 나온다 · 1시간 안이면 안 나온다
   CheckEntry    빈 이름 · . · .. · / 든 이름 · trash 가 symlink 거절
   Measure · Remove   보통 권한의 가짜 트리 — symlink 는 링크만 지운다(가리키는 파일이 남는다) · 권한 000 디렉터리를 풀고 지운다 ·
                 블록 합과 항목 수
   Deleter       가짜 Launch 로 — 기동 첫 회 · Kick 합치기 · 실패한 항목이 남고 다음 깸에 다시 · 항목이 없으면 안 깬다 ·
                 measured 가 Usage 에 들어간다 · Changed 가 불린다
   diskDrain     6.2 의 표 네 줄 · 워크스페이스 없음 · 여유를 못 측정함
   combineDrain  6.3 의 표
   statusBook    같은 값이면 안 쓴다 · 한 칸만 바뀌어도 쓴다 · 두 고루틴이 동시에 불러도 칸을 안 덮는다
   arch 키        여유가 0 이어도 툴체인이 있으면 실린다
   닫기와 예산     닫기가 마감 뒤에 끝나면 finalize_timeout (가짜 세션)
   제어판         출처 줄 · 버튼 조건 · DrainFrom
```

### 사람이 SunnyVM 에서 도는 시험 (`integration` 태그 · CI 밖)

namespace 안의 삭제 — subordinate uid 소유 항목과 권한 000 `work/work` 를 실제 overlay 세션이 남기게 하고 trash-helper 로
지운다. whiteout(문자 장치 0:0)도 지워진다.

---

## 8. 순수 함수와 커버리지

`internal/enode` 는 82.3% 다. 새 규칙은 순수 함수로 떼어 표 시험으로 덮는다.

```text
   internal/scratch   CheckEntry · Move · Orphans · Measure · Remove · Deleter (Launch 를 바꿔 끼운다) — 새 패키지 80% 이상
   internal/enode     diskDrain · combineDrain · statusBook · 닫기가 늦었는가 (settle 의 입력 하나)
```

namespace 를 여는 `Launch` 와 `RunTrashHelper` 의 unshare 경로는 `integration` 태그 시험이다. `RunTrashHelper` 의 인자 검사 ·
줄 쓰기는 기본 `go test` 에서 돈다 (namespace 없이 보통 권한의 trash 로).

---

## 9. 파일 행렬 밖의 자리

행렬(`unit-of-work-file-matrix.md` 1.3 의 ④ 열)이 준 파일은 `runc_overlay_linux.go` · `runc_overlay_other.go` · `advertise.go` ·
`detect.go` · `policy.go` · `status.go` · `config.go` · trash-helper 새 파일 · `cmd/enode/main.go` 와 `internal/scratch` ·
`internal/panel` · `packaging/macos/examples` 다.

```text
   internal/enode/claim.go       afterExit — 닫기가 마감 뒤에 끝났는지를 settle 에 넘긴다.  report 뒤 AfterReport
   internal/enode/finalize.go    settle 의 입력 하나 (닫기가 늦었다)
   internal/panel/boundary_test.go   금지 둘과 봉인 하나 (business-rules.md 10절)
   config.go                     안 바꿀 수 있다 — 새 칸이 없다 (domain-entities.md 10절).  Code Generation 이 확인한다
   runc_overlay_other.go         안 바꿀 수 있다 — 세션이 없다.  trash_other.go 가 RunTrashHelper 의 짝이다
```

---

## 10. 다른 유닛에 넘기는 것

```text
   lower-state   combineDrain 의 출처 목록에 bake 출처를 더한다 (Kind: bake · Owner: false).  상태 파일의 drain 칸과
                 제어판의 출처 줄이 그대로 받는다 — 이름 「굽기」 · 「저절로 풀린다 — 합치기가 끝나면」
   bake          Keep.Upper 에 대기 자리.  merge_wait_timeout 의 upper 는 Trash.Move 로 · 합치기의 lower 쪽 항목도
                 Trash.Move 로 (같은 filesystem · FR-7)
   checkpoint    Keep.Upper 에 spool 자리 · 받아들임은 닫기 안이고 Finalize 예산의 남은 몫을 쓴다 · Usage 에
                 spool_bytes · checkpoints 를 더한다 · 퇴출은 Trash.Move
   이 유닛의 NFR  N1 — 한 번에 지우는 양(오늘 항목 하나)과 속도 · idle IO 로 충분한가 · 10분의 간격
```

---

## 11. 정본에 되돌려 올리는 것

진행자가 `enode-design` 에 올린다.

```text
   ADR-063 §4       노드가 스스로 거는 drain (여유 부족) · 소유자 drain 과 센 쪽 합치기 · 푸는 선 min + 1 GB ·
                    출처는 노드 쪽(상태 파일과 제어판)에만
   ADR-068          상태 파일의 새 칸 drain · scratch 와 쓰는 조건
   ADR-076 §4.1     trash 의 자리와 이름 · 세션 잠금으로 남은 작업 폴더를 알아본다 · 삭제자의 때(기동 · 보고 뒤 ·
                    항목이 남으면 10분) · 삭제 경계 (다른 filesystem 으로 안 넘어간다) · 측정 뒤 삭제
   노드 설정 문서      min_free_gb 의 뜻 — 노드 전체 drain · 되찾는 선 · arch 키와 무관
```

---

## 12. 확장 준수 — Functional Design

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 trash 삭제 줄은 `business-rules.md` 5절이 닫는다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 한 줄 |

다음 단계는 유닛 정의대로면 이 유닛의 NFR Requirements (N1 · 보안 · 성능)다.
