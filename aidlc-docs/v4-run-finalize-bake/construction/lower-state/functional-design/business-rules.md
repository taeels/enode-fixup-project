# `lower-state` — 규칙

같은 기계의 형제 노드가 한 lower 를 함께 쓸 때, 누가 언제 어떤 잠금을 쥐고 놓나, 굽기가 lower 를 바꾸는 동안 형제가 어떻게
빠지나, 준비도 점검이 무엇을 보고 어떻게 말하나의 규칙이다. 오류 · 로그 문구는 영어다 (`CONVENTIONS.md` 2.1). 타입과 이름은
`domain-entities.md` 에 있다.

**원칙 셋**

- **잠금 하나가 증거다.** 이 lower 를 마운트하는 모든 자리(단계 세션 · smoke)가 lower.lock 을 공유로 쥔다. 그래서 배타를 잡은
  merge 는 「이 lower 의 overlay 마운트 0」을 안다 (Application Design Q2 · 답 4 · 5). 훑기는 그물이고 증거가 아니다
- **매칭된 순간부터 그 노드가 그 Run 의 임대를 놓을 때까지 lower 가 안 바뀐다** (결정 3-24 · Application Design 이 제안한 문구).
  지키지 못하는 경로가 생기면 그 Run 의 단계를 돌리지 않는다 — 바뀐 lower 위에서 조용히 돌지 않는다 (답 1)
- **형제는 파일과 flock 으로만 만난다.** Mediator 는 lower 를 모른다. 이 유닛이 Mediator 에서 고치는 한 줄(6절)도 Mediator 가
  lower 를 알게 하지 않는다 — drain 을 모든 매칭 경로가 보게 할 뿐이다

---

## 1. 키와 자리 (답 9 · 계획 3.2)

- **자리의 뿌리는 `$HOME/.local/state/enode/lowers` 다.** `ENODE_STATEDIR` 를 따르지 않는다 — 두 데몬이 그 변수를 다르게 두면
  lower.lock 이 두 파일이 되어 서로를 못 본다 (계획 2.1 · 2.9). home 을 못 찾으면 lower 상태를 쓰는 일을 못 한다 (9절 Kind lower)
- **lower 루트는 노드 설정의 워크스페이스다** (runc-overlay 의 binding). symlink 를 푼 경로로 읽는다
- **키는 `<fsid>-<ino>` 다.** fsid 는 `stat -f -c %i` 와 같은 16진 (앞의 0 없이 · 16 자리까지), ino 는 10진이다 (계획 2.2 ·
  Code Generation 에서 고침 2026-09-27 — `stat` 은 앞의 0 을 안 찍어 16 자리로 채우면 글자가 어긋난다)
- **권한.** 만들 때 디렉터리는 0700, 파일은 0600 이다 (FR-8 · 보안 표). 이미 있는 것의 권한을 어떻게 확인할지는 NFR 이다
- **데몬만 자리를 만든다** (`Open`). env check 와 smoke 는 만들지 않는다 (`Peek`) — 점검은 고치지 않는다 (ADR-073)

```text
   ~/.local/state/enode/lowers/<key>/
     lower.json          신원 (2절)
     state.json          상태 (3절).  없으면 committed
     lower.lock          공유 · 배타 (4절)
     bake.lock           굽기 잠금 (4절)
     holders/<node>.lock 쥔 사람 기록의 잠금 (7절)
     holders/<node>.json 쥔 사람 기록
```

---

## 2. 신원 — lower.json (답 7)

**쓰는 때** — 데몬이 `Open` 할 때. 대조한 뒤 판정에 따라 쓴다.

| 판정 | 조건 | 데몬 (`Open`) | env check (`lower.identity`) |
|---|---|---|---|
| New | lower.json 이 없다 | 새로 쓴다 | ready — `not recorded yet; the node records it on start` |
| Match | fsid · ino 가 자리 이름과 같고, btime 이 같거나 한쪽이 0 | 경로가 새면 paths 에 더해 쓴다 | ready |
| Reused | btime 이 다르고 상태가 committed | 새로 쓴다 · state.json 의 last_attempt 를 지운다 · 로그 한 줄 | ready — 「노드가 뜰 때 새로 쓴다」 |
| Foreign | btime 이 다르고 상태가 committed 가 아니다 | 오류 — 자리를 열지 않는다 | not ready (external-blocked) |
| Broken | 못 읽는다 · JSON 이 아니다 · fsid 나 ino 가 자리 이름과 다르다 | 오류 | not ready (external-blocked) |

- **btime 이 다르면 다른 디렉터리다.** ext4 는 지운 디렉터리의 inode 번호를 곧바로 다시 쓴다 (계획 2.3). 끊긴 굽기가 남은 채
  다른 디렉터리를 만나면 재개가 새 디렉터리에 옛 upper 를 합친다 — 그래서 Foreign 은 사람이 본다
- **Reused 에서 last_attempt 를 지우는 까닭** — 옛 디렉터리의 실패한 굽기를 새 디렉터리의 것처럼 제어판에 보이지 않게 한다
- **paths 는 대조하지 않는다.** bind 별칭마다 경로가 다르고 (계획 2.2), 정보로만 남는다. 두 데몬이 거의 같은 때에 쓰면 마지막
  것이 남는다 — 신원 값(fsid · ino · btime)은 둘이 같고 paths 하나를 잃을 뿐이다
- btime 이 한쪽이라도 0 이면 대조하지 않는다 — 그 filesystem 은 알려 주지 않는다

---

## 3. 상태 기계 (계획 3.1 · ADR-077 §5 · §7)

```text
   전이                              쥐는 잠금                    쓰는 쪽과 때
   committed -> building             굽기 잠금                    굽는 노드.  build 단계를 claim 한 뒤
   building  -> pending              굽기 잠금                    굽는 노드.  pending_upper 를 적는다
   pending   -> merging              굽기 잠금 + lower 배타       굽는 노드 · 재개하는 노드.  첫 합치기 동작 전에
   merging   -> committed            굽기 잠금 + lower 배타       같다.  metadata 가 합치기의 마지막 동작이고 그 뒤에
   building · pending -> committed   굽기 잠금                    실패 · merge 대기 상한 · 10절 · 기동 때 낡은 상태 정리
```

- **쓰는 쪽은 굽기 잠금의 주인 하나다** (`Bake.WriteState`). 형제는 잠금 없이 읽는다 — 파일은 rename 으로만 바뀐다 (계획 2.5)
- **쓰기 차례** — 같은 자리의 임시 파일에 쓰고 · fsync · rename · 자리 디렉터리를 fsync. merging 은 첫 합치기 동작 전에
  디스크에 있어야 한다. pending 으로 남은 채 lower 가 반쯤 바뀌면 기동 정리가 upper 를 버린다
- **없으면 committed 다.** 못 읽으면(권한 · 깨진 JSON) 그 노드는 lower 출처로 drain 한다 (9절) — 모르는 상태를 committed 로
  읽지 않는다
- 전이를 누가 언제 일으키는지(build · merge 단계 · 기동 정리)는 bake 유닛이다. 이 유닛은 쓰는 법과 잠금의 짝을 정한다

---

## 4. 잠금 셋

```text
   lower.lock    공유    형제 — 후보인 동안 · prepare 가 아닌 Run 의 임대를 쥔 동안 (5절) · smoke (8.3)
                 배타    merge 단계 · 기동 때 재개.  잡히면 이 lower 를 마운트한 자리가 0 이다
   bake.lock     배타    굽기 Run 의 노드 — building 부터 committed 까지 · 재개하는 노드.  주인이 살아 있다는 증거 (결정 3-12)
   holders/*.lock 배타   쥔 사람 기록마다 (7절)
```

- **승격(공유에서 배타로)은 쓰지 않는다** (결정 3-9). 굽는 노드는 build 단계를 claim 하면 공유를 놓는다 — 한 프로세스의 fd 둘도
  서로 부딪친다 (계획 2.1 다섯째 줄)
- **형제는 state.json 을 먼저 읽는다.** pending · merging 이면 공유를 잡지 않고 bake 출처로 drain 한다. 기다리는 배타에 우선권이
  없어서다 (계획 2.1 셋째 줄). 예외는 둘이다 — 놓은 뒤에 보인 임대(5.4)와 smoke(8.3). 둘 다 이미 매칭됐거나 몇 초 쥐고 놓는다
- **잠금 파일과 기록 파일은 `O_CLOEXEC` 로 연다.** fd 가 helper 나 단계 프로세스로 새면 데몬이 죽어도 잠금이 남는다 (계획 2.1)
- **배타를 기다리는 법** — 1초마다 `LOCK_EX|LOCK_NB` 를 다시 건다. 막힌 채 기다리는 flock 은 ctx 로 끊을 수 없다. 1초는 하루치
  합치기(1 ~ 2초)에 견주어 작다
- **쥔 프로세스가 죽으면 커널이 푼다** — 측정한 값은 2 ms 안이다 (계획 2.1)

---

## 5. 후보 잠금 (Application Design Q1 · 답 1)

runc-overlay 노드만 한다. `LowerGuard` 가 광고 주기와 claim 에서 쥐고 놓는다.

### 5.1 쥐는 때

- **실을 drain 이 없으면 광고 전에 공유를 쥔다** — 소유자 · 여유 부족 · bake · lower 출처를 합친 값이 비어 있을 때. 기록의
  역할은 candidate 다
- **prepare 가 아닌 Run 의 임대를 쥔 동안 놓지 않는다.** 광고 응답이나 claim 으로 임대를 보면 기록의 역할을 run 으로 바꾼다
- **못 쥐면 (배타가 쥐어져 있다)** bake 출처(`lower is being merged`)를 더해 drain 한다 — 그 광고는 drain 을 싣는다

### 5.2 놓는 울타리 — 두 번 연속

```text
   센다        광고 응답마다 — 응답의 drain 이 비어 있지 않고 (graceful · at-boundary) 임대 목록이 0 이면 하나 는다
   되돌린다     그런 응답이 아니면 0 으로
   그대로 둔다   광고가 실패하면 — 응답이 없으면 본 것이 없다.  셈한 두 응답 사이의 시간은 늘기만 한다
   놓는다       셈이 2 가 되고 · 도는 단계가 없을 때 (Worker 가 세션을 닫았다)
```

- **두 번의 뜻** — 첫 응답 전에 drain 을 읽은 매칭이 한 주기(기본 60초) 안에 커밋하면 둘째 응답에 그 임대가 보인다 (계획 2.7)
- **도는 단계가 없어야 놓는다.** 취소된 Run 의 임대는 목록에서 먼저 빠지지만 그 단계의 세션은 조금 뒤에 닫힌다 — Worker 가 임대를
  1초마다 보고 단계를 끊는다 (`claim.go:643`). 세션이 열린 채 놓으면 마운트 0 의 증거가 깨진다
- **놓을 때 표지를 적어 둔다** — 그때의 metadata 표지(bake.run · merged_at). 5.4 가 쓴다
- 놓은 뒤에 drain 이 풀리면(합치기가 끝났다 · 소유자가 풀었다) 5.1 로 다시 쥔다 — 공유를 잡고 · metadata 에서 새 ir 을 읽고 ·
  drain 없는 그 광고에 싣는다 (Application Design Q1 의 순서)

### 5.3 prepare 단계를 claim 하면 놓는다 (결정 3-9)

- claim 한 단계의 effect 가 prepare 이거나 종류가 merge 면 공유를 곧바로 놓는다. 울타리를 세지 않는다 — 노드의 임대는 하나라
  (I1 — 노드마다 임대 하나) 다른 Run 이 이 노드에 붙어 있지 않다
- 굽기 Run 의 세션은 공유 대신 굽기 잠금이 지킨다 — 굽기 잠금 없이 합치기는 없다 (결정 3-12)

### 5.4 놓은 뒤에 보인 임대 (답 1)

놓은 순간 그 노드의 임대는 0 이다. 그 뒤 광고 응답이나 claim 으로 prepare 가 아닌 Run 의 임대가 보이면 **늦게 보인 임대**다
(늦게 커밋한 매칭 · 실행 중 획득).

```text
   state.json 이 merging                        거절 — lower 가 반쯤 바뀌었을 수 있다
   metadata 표지가 놓을 때와 다르다              거절 — 그 사이 합쳐졌다
   그 밖 (committed · building · pending)       공유를 잡는다 (역할 run).  pending 이어도 잡는다 — merge 는 그 Run 을 더 기다린다
     못 잡았다 (배타가 쥐어져 있다)              거절
```

- **거절한 Run** — 그 Run 의 단계를 claim 하면 돌리지 않고 곧바로 보고한다. `reason: lower_changed` · error
  `the lower changed after this run was matched on this node; resubmit the run`. 준비(워크스페이스 · $IN)도 하지 않는다
- 거절 표는 그 Run 의 임대가 사라지면 지운다
- **prepare 단계와 merge 단계는 거절하지 않는다** — 굽기는 sync 로 다시 맞추고, 굽기끼리는 굽기 잠금이 막는다 (5.3)
- **상태 자리를 못 연 노드**(9절 Kind lower)는 공유를 못 쥐므로 prepare 가 아닌 단계를 돌리지 않는다. 오류 `cannot open the lower
  state directory: <원인>` 으로 보고하고 원인 코드는 달지 않는다 — lower 가 바뀐 것을 안 것이 아니다

### 5.5 기동 때

- 데몬이 뜰 때 「놓은 상태」로 시작한다 — 그때의 metadata 표지를 적는다. 첫 광고 전에 5.1 로 쥔다 (`services.md` 4절 5)
- 첫 응답에 임대가 보이면(ADR-030 이 끝내지 않은 Run) 쥐었으면 그대로, 못 쥐었으면 5.4 로 다룬다

---

## 6. 실행 중 획득도 drain 을 본다 (답 2 · Mediator 한 줄)

- **고치는 곳** — `internal/store/acquire.go` 의 `tryGrab`. busy 에 drain 목록(`drainingIn`)을 합친다. 제출(`api.go:596`)과 대기열
  승격(`queue.go:302`)이 이미 하는 모양 그대로다
- **파일 행렬 밖이다.** Code Generation 계획에 적는다. Mediator 를 고치는 줄이라 Application Design Q1 의 근거 「Mediator 변경 0」과
  부딪친다 — 그 근거는 drain 한 노드에 매칭이 안 떨어진다는 전제 위에 있었고, 획득 경로에서 그 전제가 거짓이었다 (계획 2.7)
- **까닭은 ADR-063 이다.** §2.1 이 drain 을 「이 노드로 새 임대를 내보내지 않는다」로, §6 이 「draining 노드는 busy 와 같이 후보에서
  빠진다」로 정했다. 획득은 그 둘을 어겼다 — 굽기와 무관하게 소유자의 drain 도 안 먹었다
- **바뀌는 동작** — drain 중인 노드만 남은 획득은 「못 잡음」(`unavailable`) 갈래로 간다. 오늘 다른 Run 에 묶인 노드와 같은 길이다
- 이것으로도 5.4 는 남는다 — 제출과 승격의 늦은 커밋은 그대로다

---

## 7. 쥔 사람 기록 (답 3)

- **한 쌍** — `holders/<node_id>.lock` 을 공유를 쥔 동안 배타 flock 으로 든다. 내용은 `holders/<node_id>.json` 에 임시 파일 + rename
  으로 쓴다 (칸은 `domain-entities.md` 4절)
- **차례** — 잡을 때 lower.lock (공유) 다음 기록, 놓을 때 기록 다음 lower.lock. 그래서 살아 있는 기록이 있으면 그 노드는 lower.lock
  을 쥐고 있다
- **다시 쓰는 때** — 역할이 바뀔 때 (candidate · run) · Run 이 바뀔 때 · `acks` 가 바뀔 때 (5.2 의 셈)
- **읽는 법** — `.json` 마다 짝 `.lock` 에 `LOCK_SH|LOCK_NB` 를 건다. 막히면 살아 있는 기록, 잡히면 죽은 기록이라 건너뛴다.
  건 잠금은 곧바로 놓는다. **지우지 않는다** — 같은 노드가 다시 뜨면 덮어쓴다. 지우면 막 잡은 쪽과 경쟁한다
- node_id 는 파일 이름이 된다 — `[0-9a-z-]` 밖의 글자가 있으면 기록을 안 쓰고 로그 한 줄 (공유 잠금은 그대로 쥔다)
- **smoke 는 기록을 안 남긴다.** 몇 초 쥐고 놓고, 같은 node_id 의 데몬이 이미 그 이름의 잠금을 쥐고 있을 수 있다. 그래서 배타를
  기다리는 쪽은 「막혔는데 살아 있는 기록이 0」을 `Waiting.Unnamed` 로 받는다 — 기록 없는 쥔 쪽(smoke · 옛 판의 프로세스)이다

---

## 8. 「이 lower 의 overlay 마운트 0」 (답 4 · 5)

### 8.1 증거

**배타 잠금이다** (Application Design · 그대로). lower 를 마운트하는 자리는 둘이고 (단계 세션 · smoke), 둘 다 공유를 쥔다. merge-rules
의 `Preflight` 는 마운트를 보지 않는다.

### 8.2 배타 뒤의 그물 — `ForeignMounts`

부르는 쪽(bake 의 merge 단계와 기동 때 재개)이 배타를 잡은 뒤 · merging 을 적기 전에 부른다.

```text
   이 lower 의 자리     /proc/self/mountinfo 에서 lower 루트를 담은 마운트를 찾아 (장치, filesystem 안의 경로 L) 을 얻는다
   훑는 프로세스         /proc 의 pid 중 주인이 이 uid 인 것.  마운트 namespace (ns/mnt 링크) 마다 한 번 읽는다
   찾는 것             fstype 이 overlay 인 줄.  super option 의 lowerdir= (':' 로 나눈 것) · lowerdir+= · datadir+= 의 경로 P 마다
                       같은 namespace 에서 P 를 담은 마운트 M 을 찾아 (M 의 장치, M 의 root + P 의 나머지) 를 얻는다
                       장치가 같고 그 경로가 L 이거나 L 안이거나 L 을 담으면 — 이 lower 에 닿는 overlay 다
   못 읽은 것          권한이면 Unreadable 로 센다.  사라졌거나 좀비면 (ns 링크가 없다) 건너뛴다
```

- **overlay 줄만 본다.** bind 줄만으로 보면 lower 를 그대로 bind 한 호스트의 마운트(SunnyVM 의 `/work`)와 모든 namespace 의 `/`
  마운트가 걸린다 (계획 2.2 · 2.4)
- **찾으면 합치지 않는다.** 배타를 놓고 다시 기다린다 — 기다림의 상한은 그대로다 (merge 단계면 merge.wait, 재개면 없음). 로그에
  찾은 pid 와 마운트 자리를 쓴다. 문장과 다시 볼 간격은 bake 유닛이 정한다
- **0 이어도 잠금 없이 합치지 않는다** — 그물이다
- 비용은 이 기계에서 4.7 ms 였다 (pid 451 · namespace 2 · 계획 2.6)

### 8.3 smoke 도 공유를 쥔다 (답 5)

- `ExecutionRuntimeVerifier.Verify` 가 세션을 열기 전에 워크스페이스의 자리를 `Peek` 한다. **자리가 없으면 잠그지 않는다** — 이
  lower 에 굽기가 한 번도 없었다. 합치기는 자리 없이 일어나지 않는다
- 자리가 있으면 `WaitShared` 로 공유를 쥔다. 배타가 쥐어져 있으면(합치는 중) 1초마다 다시 보며 기다린다. 기다림의 상한은 점검의
  ctx 다. 기다리기 시작할 때와 30초마다 `Notice` 에 한 줄을 쓴다 (13절). pending 이어도 쥔다 — 몇 초 뒤 놓는다
- 세션을 닫은 뒤 놓는다. 기록은 남기지 않는다 (7절)
- `enode env check` · `enode env apply` · 데몬 기동의 점검이 모두 이 길을 지난다

### 8.4 규칙 — 한 lower 의 노드를 모두 새 판으로 올린 뒤에 굽는다

이 유닛 전의 노드는 잠금도 굽기 drain 도 없다 — pending 에도 새 일을 받고, 합치는 동안 세션을 열 수 있다 (merge-rules 가 넘긴 틈).
제품은 그런 노드를 알아볼 길이 없다. 배포 문서와 정본(ADR-077 §6)에 규칙으로 적는다. SunnyVM 의 yocto 노드는 2026-09-22 판이다
(계획 2.11).

### 8.5 못 잡는 것

```text
   합치는 1 ~ 2초 사이에 옛 판의 노드가 새로 연 세션      잠금을 모른다.  그물은 합치기 전에 한 번 본다
   dumpable 이 꺼진 같은 uid 의 프로세스                 mountinfo 를 못 읽는다 (이 기계에서 6).  Unreadable 로 센다
   다른 uid 의 프로세스                                 훑지 않는다.  한 lower 는 한 사용자가 쓴다 (결정 3-14 · 12.2 가 막는다)
```

---

## 9. 굽기 출처와 lower 출처의 drain (계획 3.4 · trash 가 넘긴 일)

```text
   Kind    거는 조건                                    값          Detail (영어 한 줄)
   bake    state.json 이 pending                         graceful    lower pending a merge (run <Run>)
           state.json 이 merging                         graceful    lower being merged (run <Run>)
           공유를 못 쥐었다 (배타가 쥐어져 있다)           graceful    lower is being merged
   lower   상태 자리를 못 열었다 · state.json 을 못 읽었다  graceful    cannot open the lower state: <원인>
           home 을 못 찾았다                              graceful    cannot find the home directory: <원인>
```

- **Owner 는 거짓이다** — 소유자가 정책 파일에서 풀 수 없다. 합치기는 trash 유닛의 합치기 그대로 센 쪽 하나다 (at-boundary > graceful)
- **푸는 때** — bake 는 committed 로 돌아오고 공유를 다시 쥔 그 광고에서. lower 는 자리를 연 광고에서 (광고 주기마다 다시 연다)
- **제어판** — `page.go` 의 `drainName` · `drainLift` 에 두 줄씩

```text
   bake    「굽기」          「저절로 풀린다 — 합치기가 끝나면」         (trash 가 넘긴 문구 그대로)
   lower   「아래층 상태」    「상태 자리를 고치면 다음 광고에서 풀린다」
```

- 상태 자리를 못 연 노드를 bake 로 적지 않는다 — 「합치기가 끝나면」이 거짓이 된다
- 상태 파일은 새 칸이 없다 — drain 의 출처 목록이 그대로 싣는다 (trash 유닛의 `DrainStatus`)

---

## 10. 굽기 Run 이 합치기 없이 끝나면 (계획 3.3)

- **조건** — 이 프로세스가 굽기 잠금을 쥐고 (`HoldBake`), state.json 이 **pending** 이고 주인이 그 Run 인데, 광고 응답의 임대 목록에
  그 Run 이 없다 (취소 · 소유자의 at-boundary · 임대 만료 — 목록이 전부라 없으면 끝난 것이다 · ADR-016)
- **하는 일** — bake 가 등록한 몸통을 한 번 부른다: upper 를 trash 로 · 상태 committed · 굽기 잠금을 놓는다. 광고 루프를 막지 않게
  따로 돈다. 형제의 drain 은 다음 광고에서 풀린다
- **building 이면 여기서 하지 않는다.** build 단계의 세션이 upper 를 쓰는 중이다 — Worker 가 임대가 사라진 것을 보고 단계를 끊고,
  build 단계가 끝에서 같은 몸통으로 치운다 (bake 유닛)
- merging 이면 하지 않는다 — 합치기 본체는 끊지 않는다 (결정 3-11)

---

## 11. 광고 키 (계획 3.5 · ADR-077 §8 · FR-9 · Application Design Q4)

| 키 | 값 | 어느 노드 | 출처 |
|---|---|---|---|
| `workspace.writes` | `isolated` · `in-place` | 모든 노드 | 런타임의 `Capability().Writes`. 런타임이 없는 노드는 `in-place` |
| `ir` | IR 이름 | runc-overlay | metadata `source.ir` |
| `repo.built.<이름>` | `yes` | runc-overlay | metadata `builds[].name` 마다 |
| `bake.run` | Run id | runc-overlay | metadata `bake.run` |
| `bake.resumed` | `true` · `false` | runc-overlay | metadata `bake.resumed`. `bake.run` 이 있으면 늘 함께 |

- **예약 키다 — 라벨이 못 덮는다.** 노드 설정의 labels 에 이 이름이 있으면 광고에서 빼고 로그 한 줄 (처음 한 번). 오늘 라벨은
  탐지한 키만 못 덮는다 (`detect.go` 의 capabilities) — 이 다섯은 탐지가 아니라 광고 때 더하므로 따로 막는다. 라벨로 `ir` 을
  적으면 굽지 않은 IR 을 광고하게 된다
- **metadata 는 광고마다 읽는다** — 공유 결정 뒤에 (5.2 의 순서). 파일 하나다
- **metadata 를 읽는 규칙** — `O_NOFOLLOW` 로 연다 · 보통 파일 · 1 MiB 까지 · schema 1. 어긋나거나 JSON 이 아니면 네 키를 다 안 싣고
  로그 한 줄 (원인이 바뀔 때만). 노드는 계속 일한다 — ir 을 요구하지 않는 계약은 그대로 받는다
- **이름과 IR 은 계약의 규칙으로 거른다** (`contract.ValidBuildName` · `contract.IRProblem`). 어긋난 이름은 그 키만 빼고, 어긋난
  IR 이면 `ir` 을 뺀다. lower 는 같은 사용자라면 누구나 쓸 수 있는 자리다
- **pending · merging 동안에는 옛 metadata 를 그대로 싣는다** — drain 이 새 매칭을 막는다
- `producer.<이름>` 은 싣지 않는다 (결정 4-16 · 결과 adapter 순연)

---

## 12. 준비도 점검 셋 (답 6 · 7 · 8)

### 12.1 자리

- `ExecutionRuntimeVerifier` 가 `environment.FactSource` 를 구현한다. `CheckWithRuntime` 은 host 점검(패키지 · subordinate id ·
  user namespace) 뒤, prepared environment 와 smoke 앞에서 그 Fact 를 더한다. 하나라도 ready 가 아니면 smoke 를 안 돈다
  (`check.go:291`)
- runc-overlay 일 때만 낸다. Source 칸은 `/runtime` 이다 (binding Fact 의 선례)
- 아무것도 만들지 않는다. scratch 가 아직 없으면 가장 가까운 있는 조상으로 본다 — smoke 가 그 아래에 만든다

### 12.2 셋

| Fact | ready 의 조건 | 어긋나면 | State |
|---|---|---|---|
| `binding.scratch_filesystem` | scratch 와 워크스페이스의 st_dev 가 같고 마운트도 같다 (답 6) | 다른 filesystem · 같은 filesystem 의 다른 마운트 | invalid |
| `lower.owner_uid` | 워크스페이스 루트의 소유 uid = 노드 uid | 다른 uid | invalid |
| `lower.identity` | 2절의 New · Match · Reused | Foreign · Broken · home 을 못 찾음 | external-blocked |

- **마운트는 statx 의 `STATX_MNT_ID` 로 댄다.** 커널이 안 주면 `/proc/self/mountinfo` 에서 경로가 가장 길게 겹치는 마운트를 찾는다.
  st_dev 가 같아도 마운트가 다르면 rename 이 EXDEV 로 거절된다 (계획 2.4)
- **State 의 뜻** (답 8) — invalid 는 「설정이 쓸 수 없는 자리를 가리킨다」(선례 `check.go:172`), external-blocked 는 「profile
  밖에 남은 상태가 막는다」. 넷 다 env apply 를 멈추므로 (`apply.go:74`) 동작은 같고, 읽는 사람이 고칠 곳을 안다
- 데몬은 뜰 때 같은 점검을 돌고 ready 가 아니면 뜨지 않는다 (`main.go:148` 오늘 그대로)

### 12.3 문구 (영어)

```text
   binding.scratch_filesystem
     required     same filesystem and mount as the workspace
     observed     same mount
                  different filesystem (scratch dev <maj:min>, workspace dev <maj:min>)
                  same filesystem, different mount (scratch mount <id>, workspace mount <id>); is the workspace a bind alias?
     remediation  put environment.scratch on the same mount as the workspace, outside it; a bind alias of the workspace is a different mount

   lower.owner_uid
     required     uid <n> (the node user)
     observed     uid <m>
     remediation  run this node as the owner of the workspace; one lower is shared by one user

   lower.identity
     required     the recorded identity of this lower
     observed     not recorded yet; the node records it on start
                  matches
                  recorded for another directory with the same inode; the node rewrites it on start
                  recorded for another directory and a bake is <phase> (run <Run>)
                  cannot read <path>: <원인>
                  cannot find the home directory: <원인>
     remediation  inspect <state dir>; remove it only when no bake of that directory is left
```

---

## 13. 로그와 오류 문구 (영어)

```text
   광고 주기 (노드 로그)
     cannot open the lower state; draining this node                           dir · err            원인이 바뀔 때만
     the lower is pending a merge; draining this node                          run                  phase 가 바뀔 때만
     the lower lock is held by a merge; draining this node
     released the lower lock: drain acknowledged twice and no lease            mark
     took the lower lock again                                                 role
     a lease arrived after the lower lock was released; taking it again        run
     refusing a run: the lower changed after it was matched                    run · why
     label ignored; <key> is set by the node                                   처음 한 번
     cannot read .enode-metadata.json; not advertising ir, repo.built and bake keys   err   원인이 바뀔 때만
     the bake run's lease is gone while the lower is pending; discarding the upper    run

   단계 보고 (result.error)
     the lower changed after this run was matched on this node; resubmit the run      reason lower_changed
     cannot open the lower state directory: <원인>

   smoke (Notice)
     env check: waiting for the lower merge to finish (run <Run> on node <node>, since <시각>)

   internal/lower
     lower: <path>: not a directory
     lower: state directory <dir>: <원인>
     lower: <dir>/lower.json records a different directory (birth time differs) and its state is <phase>
     lower: <dir>/lower.json does not match its key
     lower is supported on linux only
```

---

## 14. 코드 경계 시험 (유닛 정의 6절)

`internal/panel/boundary_test.go` 의 표에 더한다.

```text
   금지 하나   cmd/mediator 는 internal/lower 를 못 가져다 쓴다
   봉인 하나   internal/lower 는 표준 라이브러리와 golang.org/x/sys 만 쓴다 (internal/scratch · internal/merge 도 안 된다)
```

`internal/store/acquire.go` 의 한 줄(6절)은 경계를 안 바꾼다 — 같은 패키지의 `drainingIn` 을 부를 뿐이다.

---

## 15. 확장 준수 — Functional Design

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 상태 자리 줄(노드 사용자 전용 권한)은 1절이 닫고 확인은 NFR 이다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 저장소의 표 시험 관례를 따른다 |
