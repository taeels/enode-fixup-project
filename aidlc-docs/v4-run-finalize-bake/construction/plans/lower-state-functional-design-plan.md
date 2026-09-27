# `lower-state` — Functional Design 계획

**유닛** `lower-state` (아래층 상태와 잠금) · **브랜치** `unit/lower-state` · **담당** taeels ·
**회차** `v4-run-finalize-bake` (굽기) · **순서** 여덟 중 여섯째 · **앞 유닛** merge-rules (`main` `56ce415` 에 병합) ·
**맡는 조각** 없음 — 코드 검사로 병합한다. 조각 8 (배타와 대기 · 사람 · SunnyVM) 의 준비도 점검 부분은 bake 유닛이 조각 8 을
돌릴 때 확인한다 · **완료 조건** 1 (노드 소유자가 drain 의 출처를 나눠 본다 — 이 유닛은 굽기 출처) · 3 ② (scratch 가 다른
filesystem 이면 not ready 이고 이유를 이름으로 말한다) · **스토리** US-1 (누가 건 drain 인지) · US-4 (not ready 가 무엇이
어긋나서인지) · **요구** FR-8 (기능 8 — 아래층의 상태 · 잠금 · drain · 준비도 점검) · FR-9 (기능 9 — 광고 키)

입력은 유닛 정의(`unit-of-work.md` 6절 · 0절 코드 검사 · 10절 크로스 빌드) · 요구(`requirements.md` FR-8 · FR-9 · 5.2 임포트
경계 · 5.3 보안 표의 상태 자리 줄 · 5.5 호환 · 6절 조각 8) · 설계(`components.md` 2.1 · 3.1 · 3.6 · 5절 실패 등급 ·
`component-methods.md` 1절 · 4.3 · 6절 · `services.md` 2 ~ 5절 · `component-dependency.md` 2 · 3절 · Application Design 계획의
Q1 · Q2 · Q4 답과 1.1 · 1.4 · 1.10) · 정본(ADR-077 §5 상태와 metadata · §6 잠금 둘과 drain · §7 굽기끼리의 배타와 재개 · §8 광고 ·
ADR-063 §4 · §6 drain) · 팩의 결정 3-9 ~ 3-15 · 3-21 · 3-22 · 앞 유닛이 넘긴 일(1절)이다. 이 계획은 그 위에서 **같은 기계의
형제 노드가 한 아래층(lower)을 어떻게 함께 쓰고, 굽기가 그것을 바꾸는 동안 누가 무엇을 쥐나** 만 짓는다. build · merge 단계와
기동 때 재개는 bake 유닛이고, 코드는 다음 단계다.

---

## 0. 이 단계가 닫는 것과 안 닫는 것

```text
   닫는다     신원 — 키의 모양 · lower.json 이 무엇을 대조하나 (물음 7)
             상태 자리의 뿌리 (물음 9) · 권한 · 파일을 쓰는 법
             상태 넷(committed · building · pending · merging)의 전이와 전이마다 쥐는 잠금
             후보 잠금 — 쥐는 때 · 놓는 울타리 · 놓은 뒤에 보인 임대 (물음 1 · 2)
             쥔 사람 기록의 모양과 살아 있는 기록만 읽는 법 (물음 3)
             「이 아래층의 overlay 마운트 0」의 증거와 그 틈 둘 (물음 4 · 5 · merge-rules 가 넘긴 일)
             굽기 출처의 drain — 거는 조건 · 푸는 순서 · 상태 파일과 제어판의 줄
             광고 키 다섯 — 어디서 읽고 언제 싣나
             준비도 점검 셋 — 무엇을 보고 (물음 6 · 7) 어느 State 로 어느 자리에 적나 (물음 8)
             코드 경계 시험 두 줄 · 크로스 빌드 짝

   안 닫는다   build · merge 단계의 흐름 · 대기 로그의 문장 · 기동 때 정리와 재개          bake 유닛
             .enode-metadata.json 쓰기                                            bake 유닛
             합치기 규칙                                                           merge-rules (병합됨)
             RuntimeCapability 의 보존 지원 절반 (Capture)                           checkpoint 유닛
             값의 크기 · 권한과 동시성의 확인                                        이 유닛의 NFR Requirements
```

---

## 1. 받는 일 — 앞 단계와 앞 유닛이 넘긴 것

```text
   보낸 곳              받는 일                                                        적힌 자리
   Application Design   공유 잠금을 놓는 울타리 — 「drain 이 받아 적힌 응답을 몇 번       계획 Q1 답 · services.md 5절 ·
                          연속 받으면 놓나」                                            unit-of-work.md 6절
                        쥔 사람 기록의 형식 · 살아 있는 기록만 읽는 법                   계획 Q4 답 · component-dependency.md 3.2
                        다른 프로세스의 mountinfo 를 읽을 수 있는지                      계획 Q2 답
                        env check 의 이음매(FactSource)를 어느 단계에서 더하고            component-methods.md 6절
                          어느 State 로 적나
                        결정 3-24 의 문구를 「매칭된 순간부터 그 노드가 그 Run 의          계획 6절 파생 결정 (정본 되돌림)
                          임대를 놓을 때까지 lower 가 안 바뀐다」로 고치자는 제안
   contract-grammar     workspace.writes 광고 — 굽기 예시의 requires 와 lint 경고가        FD 흐름 8절 · code-summary 8절
                          이 키를 쓴다.  이 광고 전에는 예시를 베낀 계약이 422 (후보 없음)
   finalize             RuntimeCapability 를 짓지 않았다 — Writes 절반이 이 유닛          FD domain-entities.md 2절
   trash                drain 출처 목록(combineDrain)에 bake 출처 (Kind bake ·            FD 흐름 10절 · code-summary 9절
                          Owner false).  상태 파일의 drain 칸과 제어판 줄이 그대로
                          받는다 — page.go 의 drainName · drainLift 에 「굽기」 ·
                          「저절로 풀린다 — 합치기가 끝나면」
   merge-rules          「마운트 0 의 증거」 — 그 계획 1.3 의 측정 (형제 helper 의 마운트는  FD 흐름 7절 · code-summary 8절
                          호스트 mountinfo 에 없고 /proc/<pid>/mountinfo 에 있다) ·
                          옛 노드의 틈 (이 유닛 전의 노드는 잠금을 쥐지 않는다)
   유닛 정의 10절        linux/arm 은 32비트 — 신원을 담는 타입이 컴파일되는지            Code Generation 계획 (2.10 에 미리 측정)
```

step-phase 는 이 유닛에 넘긴 일이 없다. 대기(QUEUED) 사유의 draining 셈은 굽기 drain 도 그대로 센다.

---

## 2. 실측 (2026-09-26)

이 기계(커널 6.5 · ext4 · uid 1000 · 특권 없음)에서 측정했다. SunnyVM(커널 7.0)은 읽기만 했다. 측정 폴더와 프로그램은
스크래치에 두었고 끝나고 지웠다. 작업 트리는 바꾸지 않았다.

### 2.1 flock — util-linux `flock(1)` 과 python `fcntl.flock`

```text
   공유를 쥔 동안 다른 프로세스의 공유 (LOCK_SH|LOCK_NB)      된다
   공유를 쥔 동안 배타 (LOCK_EX|LOCK_NB)                       거절 (EWOULDBLOCK)
   배타가 기다리는 동안 새 공유 (LOCK_SH|LOCK_NB)               된다 — 기다리는 배타에 우선권이 없다
   쥔 프로세스를 kill -9                                      기다리던 배타가 2 ms 안에 잡는다
   한 프로세스의 fd 둘 — 하나로 공유를 쥔 채 다른 fd 로 배타     거절 — 자기와 부딪친다
   fd 를 물려받은 자식                                        부모가 죽어도 자식이 끝날 때까지 잠금이 남는다
   /proc/locks                                                쥔 것은 「FLOCK ADVISORY READ <pid> <장치:inode>」,
                                                              기다리는 것은 「-> FLOCK ADVISORY WRITE ...」 로 보인다
```

- 셋째 줄 — merge 가 배타를 기다리는 동안에도 공유는 계속 잡힌다. 형제가 **state.json 을 먼저 읽고** pending · merging 이면
  공유를 잡지 않아야 배타가 굶지 않는다
- 다섯째 줄 — 굽는 노드가 자기 공유를 놓지 않으면 자기 merge 를 막는다. 결정 3-9 (잠금 둘 · prepare 를 담은 Run 은 공유를
  잡지 않는다) 의 「자기 잠금을 피한다」를 확인했다
- 여섯째 줄 — 잠금 fd 가 helper 나 단계 프로세스로 새면 데몬이 죽어도 잠금이 남는다. Go 의 `os.OpenFile` 은 `O_CLOEXEC` 로 연다

### 2.2 신원 — statfs 의 `f_fsid` 와 statx

```text
   이 기계 /home (ext4)     f_fsid.Val = [3b167980 fdd76b7b]   stat -f -c %i = 3b167980fdd76b7b
   이 기계 /dev/shm (tmpfs) f_fsid.Val = [b55c2d5a c4833425]   0 이 아니다
   SunnyVM /srv/yocto       stat -f -c %i = b517932fee9168d5   filesystem UUID 의 앞 8 바이트와 뒤 8 바이트를 XOR 한 값과 같다
   bind 별칭                user+mount namespace 안에서 mount --bind 로 만든 두 경로 —
                            f_fsid · st_dev(252:15) · inode · 생성 시각(btime)이 같고 마운트 번호(mnt_id)만 다르다
```

- 키 문자열을 `stat -f -c %i` 와 같은 순서(Val[0] 다음 Val[1], 각 8 자리 16진)로 쓰면 사람이 그 명령으로 자리를 찾는다.
  두 값을 `lo | hi<<32` 로 합쳐 찍으면 순서가 뒤집힌다 (`fdd76b7b3b167980`)
- ext4 의 `f_fsid` 는 superblock 의 UUID 에서 나온다 — 재부팅에 안 바뀐다. 다른 filesystem 종류는 측정하지 못했다 (2.11)

### 2.3 inode 재사용 — ext4

```text
   mkdir lower                  ino 2905094   btime ...210.337965693
   rmdir lower; mkdir lower     ino 2905094   btime ...210.341965733     번호가 같고 생성 시각이 다르다
```

아래층을 지우고 같은 자리에 다시 받으면 키(`<fsid>-<ino>`)가 같을 수 있다. 옛 자리의 state.json 이 merging 으로 남아 있으면
기동 때 재개가 **새 디렉터리에 옛 upper 를 합친다** (물음 7).

### 2.4 bind 별칭과 rename — st_dev 가 같아도 EXDEV

```text
   scratch 에서 bind 별칭 경로(/work 모양)의 lower 안으로 rename    EXDEV (다른 장치 사이라며 거절)
   scratch 에서 같은 lower 의 원래 경로 안으로 rename               된다
```

- 같은 filesystem(같은 st_dev)이어도 **마운트가 다르면 rename 이 안 된다.** merge-rules 의 Preflight 는 st_dev 만 본다
  (`internal/merge/merge_linux.go:81` · `:86`). 별칭 경로를 워크스페이스로 둔 노드는 점검과 Preflight 를 다 통과하고 합치기의 첫
  rename 에서 멈춘다. 재개도 같은 자리에서 멈춘다 (물음 6)
- SunnyVM 의 yocto 노드는 워크스페이스 `/srv/yocto` · scratch `/srv/enode-env/scratch` 로 둘 다 마운트 `/` 에 있다. `/work` 는
  `/srv/yocto` 의 bind 별칭이고 따로 마운트 `/work` 다 (`findmnt -T`)

### 2.5 state.json 쓰기 — 제자리 쓰기와 임시 파일 + rename

읽는 goroutine 넷이 2초 동안 읽으며 JSON 으로 풀었다.

```text
   제자리 (os.WriteFile)      쓰기 21,559   읽기 265,358   비었거나 반쯤이거나 없던 읽기 192,255
   임시 파일 + rename         쓰기 13,308   읽기  88,790   그런 읽기 0
```

형제는 잠금 없이 state.json 을 읽는다. rename 으로만 바꾼다 (`component-methods.md` 1.2 의 「임시 파일에 쓰고 rename」 그대로).
쥔 사람 기록도 같은 규칙을 따른다.

### 2.6 다른 프로세스의 마운트 — merge-rules 계획 1.3 을 이 기계에서 다시

helper 와 같은 모양(`unshare --user --map-root-user --mount` 안에서 lower 를 bind 한 뒤 그 자리를 lowerdir 로 overlay)을 띄우고
밖에서 훑었다.

```text
   이 셸의 /proc/self/mountinfo 에 그 lower 가 나오는 줄      0
   helper 의 /proc/<pid>/mountinfo                          bind 줄(root 칸이 lower 경로 · 장치 252:15)과 overlay 줄이 읽힌다
   같은 uid 프로세스를 마운트 namespace 마다 한 번씩 훑기      pid 451 · 이 uid 365 · namespace 2 · 4.7 ms
   못 읽은 것                                              권한 6 (dumpable 이 꺼진 프로세스) ·
                                                           좀비 233 (ns 링크가 없다 · 마운트를 쥐지 않는다)
```

그리고 준비도 점검의 smoke 가 실제 워크스페이스를 lowerdir 로 마운트한다 — `ExecutionRuntimeVerifier.Verify` 가
`runtimeImpl.Open(ctx, RuntimeSpec{Dir: binding.Workspace, ...})` 로 세션을 연다 (`runc_overlay_linux.go:1341`). 데몬이 뜰 때도
같은 점검을 돈다 (`cmd/enode/main.go:147`). 단계 세션 말고 lower 를 마운트하는 자리가 하나 더 있다 (물음 5).

### 2.7 매칭의 세 경로와 drain — 코드와 시험 DB

```text
   경로                  drain 을 보나   임대를 만드는 곳                                   drain 을 읽은 뒤 임대를 커밋하기까지
   제출                  본다           api.go:596 DrainingNodes -> :674 CreateRun          한 요청 안.  시간 상한을 두는 코드가 없다
                                        (따로 트랜잭션)
   대기열 승격            본다           queue.go:302 drainingIn -> :326 promoteIn            그 트랜잭션 안
   실행 중 획득(acquire)  안 본다        acquire.go:138 LiveAdverts · :142 busyIn -> :160     -
                                        Match

   광고 처리              api.go:303 UpsertAdvert (drain 커밋) -> :341 RenewLeases (따로 읽기) -> 응답
```

- 광고 응답의 임대 목록은 drain 을 커밋한 **뒤에** 읽는다. 그 전에 drain 을 읽은 매칭이 응답 뒤에 커밋하면 그 임대는 그 응답에
  없고 다음 응답(광고 주기 뒤 · 기본 60초 · `internal/config/config.go:104`)에 나온다. Application Design 1.1 이 본 창이다
- 실행 중 획득을 시험 DB 로 돌렸다 (`go test -overlay` 로 작업 트리를 안 바꾸고 · 측정 뒤 지웠다). **graceful drain 이 받아 적힌
  노드를 획득이 잡았고** 다음 광고 응답에 그 임대가 왔다. 같은 노드를 처음부터 요구한 제출은 202 (대기) 였다
- 그래서 Application Design Q1 답의 근거 「후보인 노드가 이미 쥐고 있으므로 매칭이 잠금 없는 노드에 떨어지지 않는다」가 획득
  경로에서는 성립하지 않는다. 소유자가 건 drain 도 획득에는 안 먹는다 (물음 1 · 2)

### 2.8 준비도 점검의 State 가 하는 일 — 코드

```text
   check.go:291              runtime smoke 는 앞의 Fact 가 모두 ready 일 때만 돈다
   apply.go:74               invalid · unsupported · external-blocked · admin-required 면 env apply 가 아무것도 안 하고 멈춘다
                             installable · stale 이면 절차(operations)를 돌리려 든다 — 이 유닛의 셋은 절차가 고칠 일이 아니다
   cmd/enode/main.go:148     데몬이 뜰 때 같은 점검이 ready 가 아니면 뜨지 않는다
   check.go:172 · :309       binding 이 잘못이면 binding Fact 를 invalid 로 적는다 (선례)
```

### 2.9 상태 자리의 뿌리 — 코드

```text
   internal/enode/paths.go:41   StateDir — ENODE_STATEDIR 가 이기고, 없으면 $HOME/.local/state/enode.  로그와 pid 의 자리
                                (같은 파일의 ConfDir 주석 — 「시연에서 한 기계에 여러 벌을 두는 데 쓰인다」)
   사용자에게 알리는 곳           cmd/enodectl/main.go:95 · packaging/macos/install.sh:15 · packaging/self-update.sh:21
   SunnyVM                      ~/.local/state/enode 에 노드 로그 셋이 있다 (0755).  lowers/ 는 아직 없다
```

두 데몬이 ENODE_STATEDIR 를 다르게 두면 lower.lock 이 서로 다른 파일이 된다. flock 은 같은 파일에서만 만난다 (물음 9).

### 2.10 linux/arm (32비트)

x/sys v0.47.0 에서 `Statfs_t.Fsid.Val` 은 `[2]int32`, `Stat_t.Ino` · `Statx_t.Ino` 는 `uint64` 다. statfs · statx 를 쓰는 측정
프로그램이 linux/arm · amd64 · arm64 로 빌드된다. Code Generation 계획이 제품 코드로 다시 확인한다.

### 2.11 못 한 것

- `f_fsid` 가 ext4 밖(xfs · btrfs 등)에서 재부팅 뒤에도 같은지 — 특권 없이 그런 filesystem 을 만들 수 없다 (user namespace 에서
  마운트되는 것은 tmpfs · overlay 같은 것뿐이다)
- 전원이 나간 뒤 state.json 이 어떻게 남는지 — 측정 대신 3.1 의 규칙(fsync)으로 둔다
- 옛 노드(SunnyVM `~/bin/enode` · 2026-09-22 22:36 +0900 판 · yocto 노드가 그것으로 떠 있다)가 있는 lower 에서 새 노드가 굽는
  장면 — bake 유닛의 조각에서 본다

---

## 3. 이 계획이 정하는 것 — 묻지 않는다

근거가 정본 · 설계 · 2절에 있다. 반대하면 답에 적어 주세요.

### 3.1 상태 기계와 잠금 (ADR-077 §5 · §6 · §7 · `services.md` 2 ~ 4절)

```text
   전이                              쥐는 잠금                    state.json 을 쓰는 쪽과 때
   committed -> building             굽기 잠금 (배타)             굽는 노드.  build 단계를 claim 한 뒤
   building  -> pending              굽기 잠금                    굽는 노드.  대기 upper 의 자리를 적는다
   pending   -> merging              굽기 잠금 + lower 배타       굽는 노드 또는 재개하는 노드.  첫 합치기 동작 전에
   merging   -> committed            굽기 잠금 + lower 배타       같다.  metadata 가 합치기의 마지막 동작이고 그 뒤에
   building · pending -> committed   굽기 잠금                    실패 · merge 대기 상한 · 3.3 · 기동 때 낡은 상태 정리
```

- state.json 은 굽기 잠금을 쥔 프로세스만 쓴다. 형제는 잠금 없이 읽는다 (2.5)
- 쓰기는 임시 파일 · fsync · rename · 디렉터리 fsync 다. merging 은 첫 합치기 동작 전에 디스크에 있어야 한다 — pending 으로
  남은 채 lower 가 반쯤 바뀌면 기동 정리가 upper 를 버린다
- 공유는 형제가 쥐고 배타는 merge 와 재개만 쥔다. 승격(공유에서 배타로)은 쓰지 않는다 (결정 3-9)
- 형제는 state.json 을 먼저 읽고, pending · merging 이면 공유를 잡지 않고 굽기 drain 을 싣는다 (2.1 셋째 줄)
- 잠금 파일과 기록 파일은 `O_CLOEXEC` 로 연다 (2.1 여섯째 줄)

### 3.2 키와 자리

- 키는 `<fsid>-<ino>` — fsid 는 `stat -f -c %i` 와 같은 16 자리 16진, ino 는 10진 (2.2)
- 자리의 권한은 디렉터리 0700 · 파일 0600 이다 (FR-8 · 보안 표). 확인은 NFR
- env check 는 자리를 만들지 않는다 — 점검은 고치지 않는다 (ADR-073 의 「start 는 환경을 고치지 않는다」). 자리는 데몬이 뜰 때 연다

### 3.3 굽기 Run 이 합치기 없이 끝나면

pending 동안 굽기 Run 이 취소되거나, 소유자의 at-boundary drain 이나 임대 만료로 닫히면 merge 단계가 오지 않는다. 굽기 잠금을
쥔 그 노드는 광고 응답에서 그 Run 의 임대가 사라진 것을 본다 (목록이 전부라 없으면 끝난 것이다 — ADR-016). 그러면 곧바로
upper 를 trash 로 · 상태를 committed 로 · 굽기 잠금을 놓는다. 형제의 drain 은 다음 광고에서 풀린다.

Application Design Q6 논의가 적은 결과(「취소하면 upper 는 trash 로, lower 는 committed 로, drain 이 풀린다」)를 내는 자리다.
방아쇠는 임대를 보는 LowerGuard (이 유닛)이고, 치우는 몸통은 기동 때 낡은 상태 정리와 같은 것 (bake 유닛이 쓴다) 이다.

### 3.4 굽기 출처의 drain

```text
   건다      state.json 이 pending · merging            Kind bake  · graceful · 「lower pending (run <Run>)」 모양의 영어 한 줄
             공유를 못 잡았다 (배타가 쥐어져 있다)        Kind bake  · graceful · 「lower is being merged」 모양
             상태 자리를 못 열었다 (components.md 5절)    Kind lower · graceful · 오류 한 줄
   푼다      committed 로 돌아왔다 — 공유를 다시 잡고 · metadata 에서 새 ir 을 읽고 · drain 없는 그 광고에 새 ir 을 싣는다
             (Application Design Q1 답의 순서)
   제어판     bake  — 「굽기」 · 「저절로 풀린다 — 합치기가 끝나면」 (trash 가 넘긴 문구)
             lower — 「아래층 상태」 · 「상태 자리를 고치면 다음 광고에서 풀린다」
```

- 세기는 trash 유닛의 합치기 그대로다 — at-boundary > graceful > 없음
- 상태 자리를 못 연 노드를 bake 로 적지 않는 까닭 — 제어판의 「합치기가 끝나면」이 거짓이 된다

### 3.5 광고 키 (ADR-077 §8 · FR-9 · Application Design Q4 답)

```text
   workspace.writes    runc-overlay 면 isolated · 그 밖(native · environment 블록 없음)은 in-place.  모든 노드
                       런타임이 낸다 — RuntimeCapability 의 Writes.  광고는 세션 밖이라 세션이 아니라 런타임에서 읽는다
   ir                  metadata 의 source.ir
   repo.built.<이름>   yes.  metadata 의 builds[].name 마다
   bake.run            metadata 의 bake.run
   bake.resumed        metadata 의 bake.resumed — "true" | "false".  bake.run 이 있으면 늘 함께 싣는다
```

- 아래 넷은 runc-overlay 노드만 싣고, 광고마다 metadata 파일 하나를 읽는다. 파일이 없으면 넷 다 안 싣는다. 깨졌으면 안 싣고 노드
  로그에 한 번 쓰고 상태 파일에 적는다. 노드는 계속 일한다 — ir 을 요구하지 않는 계약은 그대로 받는다
- pending · merging 동안에는 옛 metadata 를 그대로 싣는다. drain 이 새 매칭을 막는다
- `producer.<이름>` 은 싣지 않는다 — 결과 adapter 순연 (결정 4-16)

### 3.6 코드 경계와 크로스 빌드

- `internal/panel/boundary_test.go` 에 두 줄 — Mediator 가 `internal/lower` 를 못 가져다 쓴다 · `internal/lower` 는 표준
  라이브러리와 x/sys 만 쓴다 (봉인). 봉인이 `internal/scratch` · `internal/merge` 임포트도 막는다 (`components.md` 2절)
- Linux 전용 호출(statfs · statx · flock · /proc 읽기)은 `_linux.go`, 그 밖은 `_other.go` 가 unsupported 를 돌려준다
- 후보 잠금 · 굽기 drain · 준비도 점검 셋은 runc-overlay 노드만 한다. native 노드는 lower 가 없다 (`services.md` 4절)

---

## 4. 물음 아홉

답을 `[Answer]:` 뒤에 적는다. 권장을 **A** 에 둔다. 기호 옆에 뜻을 적었다.

**내기 전에 대 본 것** — U5 에서 「질문 아홉을 다시 검토」 해 나온 흠 넷 (감사 로그 2026-09-26T15:55:03Z) 에 물음마다 대 봤다.

```text
   흠                                      대 본 결과
   Application Design 이 정한 것을 다시 묻기   후보 잠금(Q1 = A) · 배타 잠금이 마운트 0 의 증거(Q2) · 기다림은 merge 로그로(Q4) 를
                                            다시 묻지 않는다.  물음 1 · 3 · 4 · 8 은 유닛 정의가 FD 에 넘긴 것이고
                                            2 · 5 · 6 · 7 · 9 는 2절의 측정이 연 것이다
   측정 없는 근거                             물음마다 2절의 번호를 단다.  측정하지 못한 것은 2.11 에 적었다
   앞 유닛의 넘김과 어긋난 선택지              trash 의 bake 출처와 문구 · merge-rules 의 「잠금이 증거」 · contract-grammar 의
                                            workspace.writes 를 그대로 받는 선택지만 둔다
   요구를 빼는 선택지                          결정 3-24 (매칭된 뒤 첫 단계 전에 lower 가 안 바뀐다) 를 어느 경로에서든 어기는
                                            선택지는 뺐다 (물음 1 · 2 · 5).  물음 6 의 B 와 물음 7 의 C 는 요구 문구 그대로이고,
                                            남는 위험을 선택지 안에 적었다
```

### Question 1 — 공유 잠금을 놓는 울타리와, 놓은 뒤에 보인 임대

Application Design Q1 = A — 노드는 매칭 후보인 동안 공유 잠금을 쥐고, drain 이 받아 적혔고 임대가 없으면 놓는다. 놓는 울타리는
이 단계의 몫이다. 2.7 — 광고 응답의 임대는 drain 커밋 뒤에 읽지만 그 전에 drain 을 읽은 매칭이 응답 뒤에 커밋할 수 있고 (제출에는
시간 상한이 없다), 실행 중 획득은 drain 을 아예 안 본다. 그래서 몇 번을 기다려도 **놓은 뒤에 임대가 오는 일**이 남는다. 그 규칙을
함께 정한다.

두 선택지에 공통인 규칙 — 놓는 순간 그 노드의 임대는 0 이다. 그 뒤 광고 응답이나 claim 으로 임대가 보이면 곧바로 공유를 다시
잡는다. 잡았고 놓은 뒤로 lower 가 합쳐지지 않았으면 (metadata 의 bake.run · merged_at 이 그대로) 평소처럼 돈다 — merge 는 그 Run
을 더 기다린다. 못 잡았거나 (합치는 중) 그 사이 합쳐졌으면 그 Run 의 단계를 돌리지 않고 실패로 보고한다. 원인 코드가 하나 는다
(`lower_changed` · wire 값이라 정본에 올린다).

A) **두 번 연속** — drain 이 받아 적히고 임대가 0 인 응답을 두 번 연속 받으면 놓는다 (첫 응답에서 한 광고 주기 뒤 · 기본 60초).
   첫 응답 전에 drain 을 읽은 매칭이 한 주기 안에 커밋하면 둘째 응답에 보여 놓지 않는다. merge 의 기다림이 한 주기 는다.
   `lower_changed` 는 한 주기를 넘겨 커밋한 매칭과 실행 중 획득에서만 난다

B) **한 번** — 첫 응답에서 놓는다. merge 가 한 주기 일찍 잡는다. 대신 drain 과 거의 같은 때 들어온 제출이 그 사이 합치기를 만나면
   `lower_changed` 로 실패한다

C) Other (please describe after [Answer]: tag below)

**권장 A.** 2.7 — 제출과 승격의 창은 한 요청 · 한 트랜잭션이라 한 주기면 거의 다 닫힌다. ADR-077 §12 — 형제가 기다리는 창은
합치기가 아니라 형제 Run 이 정한다. 한 주기(60초)는 형제 Run 에 비해 작고, `lower_changed` 로 끝난 Run 은 다시 내야 한다.

[Answer]: A

### Question 2 — 실행 중 획득(acquire)이 drain 을 안 본다

2.7 의 측정이다. ADR-063 §6 은 「draining 노드는 busy 와 같이 후보에서 빠진다」고 적었는데, 제출과 대기열 승격만 drain 을 보고
(`api.go:596` · `queue.go:302`) 획득은 안 본다 (`acquire.go:138` ~ `:160`). 굽기와 무관하게 소유자의 drain 도 획득에는 안 먹는다.
물음 1 의 규칙이 있으면 lower 는 지켜진다. 남는 것은 drain 한 노드가 새 일을 받는다는 것과, 굽기 중이면 그 Run 이 `lower_changed`
로 실패하거나 merge 가 그 Run 을 더 기다린다는 것이다.

A) **이 유닛이 획득 경로에 drain 을 넣는다** — `internal/store/acquire.go` 의 tryGrab 이 drain 목록을 busy 에 합친다 (제출 · 승격과
   같은 모양의 한 줄) + 시험 하나 (drain 한 노드를 획득이 안 잡고 그 단계는 unavailable 갈래로 간다). 파일 행렬 밖이라 Code
   Generation 계획에 적는다. Application Design Q1 의 근거 「Mediator 변경 0」과 부딪치지만, 그 근거는 drain 이 모든 매칭에서
   빠진다는 전제 위에 있었다

B) Mediator 는 두고 노드 쪽 규칙(물음 1)만 둔다. 어긋남은 정본 되돌림(ADR-063 · ADR-024 실행 중 획득)과 잔여로 적는다

C) 이 회차 밖의 PR 로 따로 고친다 — bake 유닛 전에 main 에 들어가게. 이 유닛은 B 와 같다

D) Other (please describe after [Answer]: tag below)

**권장 A.** 2.7 의 시험 — graceful drain 이 받아 적힌 노드를 획득이 잡았다. ADR-063 §2.1 은 drain 을 「이 노드로 새 임대를 내보내지
않는다」로 정의한다. 고칠 자리가 한 줄이고 같은 모양이 두 곳에 이미 있다. B 는 소유자 drain 의 약속을 획득에서 계속 어긴다.

[Answer]: A

### Question 3 — 쥔 사람 기록의 모양과 살아 있는 기록만 읽는 법

Application Design Q4 = A — merge 가 기다리는 동안 단계 로그에 「노드 · Run · 언제부터」와 「candidate · drain 이 아직 안 받아
적힘」을 쓴다. 그 재료인 기록의 모양과 살아 있는 것만 읽는 법은 이 단계의 몫이다 (`component-dependency.md` 3.2).

A) **노드마다 잠금 파일과 기록 파일 한 쌍.** `holders/<node_id>.lock` 을 공유를 쥔 동안 배타 flock 으로 들고, 내용(노드 · 라벨 ·
   역할 candidate 또는 run · Run · 언제부터 · pid)은 `holders/<node_id>.json` 에 임시 파일 + rename 으로 쓴다. 읽는 쪽은 `.lock` 에
   `LOCK_SH|LOCK_NB` 를 걸어 막히면 살아 있는 기록, 잡히면 죽은 기록으로 보고 건너뛴다. 지우지 않는다 — 같은 노드가 다시 뜨면
   덮어쓴다. 잠금과 기록을 한 파일에 두지 않는 까닭 — rename 은 inode 를 바꿔 flock 이 따라가지 않고, 제자리 쓰기는 반쯤 읽힌다 (2.5)

B) 기록 하나에 pid 와 프로세스 시작 시각을 적고 `/proc/<pid>/stat` 로 살아 있는지 본다. pid 가 다시 쓰여도 시작 시각으로 구별한다

C) `/proc/locks` 에서 lower.lock 을 쥔 pid 를 읽고 기록의 pid 와 맞춘다 (2.1 — 쥔 것과 기다리는 것이 보인다)

D) Other (please describe after [Answer]: tag below)

**권장 A.** 2.1 — 커널이 죽은 프로세스의 잠금을 2 ms 안에 푼다. 살아 있음의 증거가 lower.lock 과 같은 기제라 새 가정이 없다.
B 와 C 는 /proc 의 형식과 pid 에 기대고, C 는 기록을 쓰기 전과 뒤의 틈에서 pid 와 기록이 어긋날 수 있다.

[Answer]: A

### Question 4 — 「이 아래층의 overlay 마운트 0」과 옛 노드의 틈

Application Design 은 부르는 쪽이 쥔 lower 배타 잠금을 증거로 정했다 (`components.md` 2.2). merge-rules 답 1 = A 가 옛 노드의 틈을
이 유닛에 넘겼다 — 이 유닛 전의 노드는 잠금도 굽기 drain 도 없어서, pending 에도 새 일을 받고 합치는 동안 세션을 열 수 있다.
SunnyVM 의 yocto 노드가 2026-09-22 판으로 `/srv/yocto` 에 떠 있다.

A) **배타 잠금이 증거 (그대로) + 틈을 두 겹으로 좁힌다.** (1) merge 가 배타를 잡은 뒤 merging 을 적기 전에, 같은 uid 프로세스의
   마운트 namespace 를 한 번씩 훑어 이 lower 를 가리키는 마운트(같은 장치 · root 칸이 lower 이거나 그 안이나 그 위)가 있으면
   합치지 않는다 — 배타를 놓고 다시 기다린다 (대기 상한은 그대로). 훑기는 증거가 아니라 그물이다 — 0 이어도 잠금 없이 합치지 않는다.
   훑는 코드는 `internal/lower` 의 Linux 파일이다. (2) 규칙 — 한 lower 의 노드를 모두 새 판으로 올린 뒤에 굽는다. 배포 문서와
   ADR-077 §6 에 적는다

B) 배타 잠금이 증거 (그대로) + 규칙 (A 의 (2)) 만. 제품 코드는 훑지 않는다

C) Other (please describe after [Answer]: tag below)

**권장 A.** 2.6 — 호스트 mountinfo 에는 안 보이고 helper 의 mountinfo 는 같은 uid 가 읽는다. 훑기는 4.7 ms 다. 옛 노드가 합치기
직전에 열어 둔 세션이라는 가장 흔한 경우를 잡는다. 못 잡는 것 둘은 FD 에 적는다 — 합치는 1 ~ 2초 사이에 옛 노드가 새로 연 세션 ·
dumpable 이 꺼진 프로세스 (이 기계에서 6).

[Answer]: A

### Question 5 — 준비도 점검의 smoke 도 lower 를 마운트한다

2.6 — `enode env check` 의 smoke 는 실제 워크스페이스를 lowerdir 로 세션을 연다. 사람이 점검을 돌릴 때와 데몬이 뜰 때 돈다. 잠금을
쥐지 않으면 「배타 잠금이 마운트 0 의 증거」가 이 자리에서 깨진다 — 합치는 동안 새 노드를 띄우면 그 노드의 점검이 lower 를
마운트한다.

A) **smoke 도 공유 잠금을 잡는다.** 배타가 쥐어져 있으면(합치는 중) 풀릴 때까지 기다렸다 돈다 — 기다림의 상한은 점검의 ctx 다.
   pending 이어도 잡는다 (몇 초 쥐고 놓으므로 merge 가 그만큼 더 기다린다). 기다리는 동안 점검 출력에 한 줄을 쓴다

B) smoke 도 공유 잠금을 잡되 기다리지 않는다 — 합치는 중이면 smoke 를 돌리지 않고 runtime.smoke Fact 를 not ready
   (external-blocked · 「합치는 중이다 · 다시 돌려라」)로 적는다. 그때 데몬은 뜨지 않는다 (`main.go:148`)

C) smoke 가 실제 워크스페이스 대신 빈 임시 디렉터리를 lowerdir 로 쓴다 — 잠금이 필요 없다. 대신 smoke 가 확인하던 것 하나(워크스페이스
   자리의 쓰기가 upper 로 가고 lower 에 안 닿는다 — 마커 파일)가 실제 워크스페이스에서 빠진다

D) Other (please describe after [Answer]: tag below)

**권장 A.** 합치기 본체는 하루치가 1 ~ 2초다 (merge-rules 의 SunnyVM 벤치마크 Apply 1.11 초 · ADR-077 §12 1.49 초). 기다려도 점검이
거의 안 늦고, 잠금 하나가 모든 마운트의 증거로 남는다. B 는 합치는 순간에 뜬 데몬을 이유 없이 떨어뜨린다.

[Answer]: A

### Question 6 — 「scratch 와 워크스페이스가 같은 filesystem」을 무엇으로 확인하나

FR-8 과 ADR-077 §4 는 st_dev 일치로 적었다. 2.4 — st_dev 가 같아도 마운트가 다르면 rename 이 EXDEV 로 거절된다. merge-rules 의
Preflight 도 st_dev 만 본다.

A) **마운트까지 같아야 ready.** env check 가 st_dev 와 마운트 번호(statx 의 `STATX_MNT_ID` · 커널 5.8 부터 · 없으면
   `/proc/self/mountinfo` 에서 찾는다)를 함께 본다. 어긋나면 not ready 이고 사유가 「같은 filesystem 이지만 다른 마운트 (bind
   별칭?)」를 말한다. bake 유닛에 넘긴다 — merge 의 Preflight 에도 같은 확인 (merge-rules 코드 한 곳 · 행렬 밖이라 bake 의 Code
   Generation 계획이 적는다)

B) st_dev 만 (요구 문구 그대로). 별칭 경로를 워크스페이스로 둔 노드도 ready 가 되고, 첫 굽기가 합치기의 첫 rename 에서 EXDEV 로
   멈춘다. 재개도 같은 자리에서 멈춰 상태가 merging 에 머물고 형제는 drain 에 남는다 — 사람이 풀어야 한다

C) Other (please describe after [Answer]: tag below)

**권장 A.** 2.4 — 같은 st_dev 에서 EXDEV 를 봤다. SunnyVM 의 yocto 노드는 워크스페이스와 scratch 가 같은 마운트라 A 에서도 ready 다.

[Answer]: A

### Question 7 — `lower.json` 의 신원은 무엇을 대조하나

Application Design 의 모양(`component-methods.md` 1.1)은 schema · fsid · ino · paths · owner_uid · seen_at 이다. 자리 이름이 키라
fsid · ino 는 늘 맞는다 — 대조가 무엇을 잡을지가 비어 있다. 2.3 — ext4 는 지운 디렉터리의 inode 번호를 곧바로 다시 쓴다.

A) **생성 시각(statx 의 btime)을 더해 inode 재사용을 잡는다.** 다르면 — 옛 상태가 committed 면 노드가 뜰 때 자리를 새로 쓰고 (옛
   last_attempt 는 버린다 · 로그 한 줄), committed 가 아니면 not ready 다 (끊긴 굽기가 다른 디렉터리의 것이다 · 사람이 본다).
   paths 는 별칭을 더할 뿐 대조하지 않는다. btime 을 못 읽는 filesystem 이면 대조하지 않는다

B) A 에 더해 키가 바뀐 것도 잡는다 — 이 lower 의 경로가 다른 키 자리의 lower.json 에 committed 가 아닌 상태로 적혀 있으면 not ready.
   `f_fsid` 가 재부팅에 바뀌는 filesystem 에서 끊긴 합치기를 잃지 않게 한다. 비용은 자리 목록을 한 번 읽는 것이다

C) Application Design 의 모양 그대로 — 파일이 깨졌거나 키와 다를 때만 not ready. inode 재사용을 못 잡는다 (2.3)

D) Other (please describe after [Answer]: tag below)

**권장 A.** 2.3 — 재사용이 실제로 나고 btime 은 다르다. 2.2 — ext4 의 fsid 는 UUID 에서 나와 재부팅에 안 바뀐다. B 의 키 이동은 ext4
밖에서만 생길 수 있고 이 계획은 그것을 측정하지 못했다 (2.11). 그런 filesystem 을 쓰는 곳이 생기면 B 를 더한다.

[Answer]: A

### Question 8 — 점검 셋을 어느 State 로, 어느 자리에서 적나

`component-methods.md` 6절이 이 단계에 넘겼다. 두 선택지에 공통 — 이음매(FactSource)의 Fact 는 host 점검 뒤, prepared environment
와 smoke 앞에 더한다. 어긋나면 smoke 를 안 돈다 (smoke 는 lower 를 마운트하고 작업 폴더를 남긴다). runc-overlay 일 때만 낸다.
Fact 이름은 `binding.scratch_filesystem` · `lower.owner_uid` · `lower.identity` (`component-methods.md` 1.5).

A) **사유에 따라 둘로 적는다.** scratch 의 filesystem 과 소유 uid 는 `invalid` — binding 이 쓸 수 없는 자리를 가리킨다 (선례
   `check.go:172`). lower.json 신원은 `external-blocked` — profile 밖에 남은 상태가 막는다. 고침 안내(remediation)가 상태 자리의
   경로를 말한다

B) 셋 다 `invalid`

C) Other (please describe after [Answer]: tag below)

**권장 A.** 2.8 — 이 넷 중 무엇이든 env apply 는 멈추므로 동작은 같다. 사람이 읽는 State 가 고칠 곳(설정인가 남은 상태인가)과 맞는다.
installable 과 stale 은 apply 가 절차를 돌리려 들어 틀린 안내가 된다.

[Answer]: A

### Question 9 — 상태 자리의 뿌리

정본과 요구는 `~/.local/state/enode/lowers/<fsid>-<ino>/` 다 (결정 3-14 — 한 lower 의 노드는 한 사용자가 같은 home 을 보는
환경에서 띄운다). 코드에는 로그의 자리 StateDir 이 이미 있고 ENODE_STATEDIR 가 이긴다 (2.9).

A) **lowers 는 ENODE_STATEDIR 를 따르지 않는다** — 늘 사용자의 home($HOME) 아래 `.local/state/enode/lowers` 다. 함수 하나로
   `internal/enode/paths.go` 에 둔다 (「노드가 쓰는 자리를 한 곳에만 적는다」는 그 파일의 규칙)

B) StateDir() 아래 `lowers/` — ENODE_STATEDIR 가 이긴다. 한 기계에 여러 벌을 띄우는 쪽이 따로 나뉜다

C) Other (please describe after [Answer]: tag below)

**권장 A.** 2.1 · 2.9 — 두 데몬이 ENODE_STATEDIR 를 다르게 두면 lower.lock 이 두 파일이 되어 서로를 못 보고, merge 가 형제의 Run
아래에서 lower 를 바꾼다. 로그를 나누려고 둔 변수가 잠금을 나누면 안 된다. 한 기계의 여러 벌이 서로 다른 lower 를 쓰면 A 에서도
키가 달라 이미 나뉜다.

[Answer]: A

---

## 5. 산출물 계획 (체크박스)

답이 들어오고 모호함이 풀린 뒤에 채운다. 자리는 `aidlc-docs/v4-run-finalize-bake/construction/lower-state/functional-design/` 이다.

- [x] 답을 읽고 모호함을 확인한다 — 있으면 되물음 파일을 만든다 (2026-09-26T18:14:21Z · 아홉 모두 A · 답끼리 막는 짝 없음 · 되물음 없음)
- [x] `domain-entities.md` — `Key` · `Dir` · `Identity` (답 7) · `Phase` · `State` · `Owner` · `LastAttempt` · `Holder` (답 3) ·
      `Shared` · `Exclusive` · `Bake` · metadata 읽기 · 마운트 훑기 (답 4) · `Finding` · `LowerGuard` 와 그 입력 · drain 출처의 새
      Kind 둘 · 광고 키 · 이음매(FactSource)와 Fact 셋 · 상태 자리의 뿌리 (답 9) · linux 와 그 밖의 짝
- [x] `business-rules.md` — 키와 자리 · 권한 · 파일 쓰기 · 상태 기계와 전이마다의 잠금 · 후보 잠금 (쥐는 때 · 울타리 답 1 · 놓은
      뒤의 임대 · 획득 답 2) · 쥔 사람 기록 · 마운트 0 (답 4 · 5) · 굽기 drain 과 푸는 순서 · 합치기 없이 끝난 굽기 · 광고 키 ·
      점검 셋 (답 6 · 7 · 8) · 오류와 로그 문구 (영어)
- [x] `business-logic-model.md` — 광고 주기 한 바퀴의 LowerGuard · claim 때 · 임대가 사라질 때 · merge 가 기다릴 때 기록을 읽는 법 ·
      smoke 의 잠금 · 기동 때 · env check 흐름 · 시험 모양 · 파일 행렬 밖 자리 · 다른 유닛에 넘기는 것 · 정본 되돌림
- [x] 코드 경계 시험 두 줄 — Mediator 가 `internal/lower` 를 못 가져다 쓴다 · `internal/lower` 는 표준 라이브러리와 x/sys 만
      (`business-rules.md` 14절에 적었다 · 코드는 Code Generation)
- [x] 커버리지 — `internal/lower` 80% 이상 · `internal/enode` 에는 잇는 코드만 (지금 값은 Code Generation 계획이 적는다 ·
      `business-logic-model.md` 10절 — cmd/enode 80.4% 가 하한에 가깝다)
- [x] 표기 검사 (`enode-design/scripts/emphasis-check.py`) · 말투 검사 · 사내 이름 검사 (새 문서 셋과 고친 문서 모두 exit 0 · 말투 0 ·
      사내 이름 0)

---

## 6. 확장 준수 — 이 단계

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 상태 자리 줄(노드 사용자 전용 권한)은 팩의 요구로 남아 3.2 가 닫는다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 저장소의 표 시험 관례를 따른다 |

유닛 정의는 이 유닛의 NFR Requirements 를 「한다 (최소)」로 적었다 — 보안(상태 자리 권한) · 동시성(같은 기계의 여러 노드가 같은
파일과 잠금을 쓴다). Functional Design 이 닫힌 뒤에 정한다.
