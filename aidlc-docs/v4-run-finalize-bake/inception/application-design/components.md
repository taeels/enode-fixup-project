# 컴포넌트 — 굽기 회차

입력은 `requirements.md` (FR-1 ~ FR-13) · `stories.md` (완료 조건 열) ·
`plans/application-design-plan.md` 의 답 일곱(전부 A)이다. 여기는 **무엇이 어디 살고 무엇을
아는가**만 적는다. 규칙은 Functional Design 이, 값은 NFR Requirements 가 진다.

- **작성 시각**: 2026-09-24T06:33:21Z · 회차 브랜치 `v4-run-finalize-bake` · HEAD `b7f7c7a`

---

## 1. 지도

노드 쪽에 새 패키지 셋이 서고(Q2), 기존 패키지 여덟이 책임을 더한다. Mediator 쪽은 칸과
라우트 하나가 는다.

```text
   노드 기계 (한 사용자)
     enode 데몬 (cmd/enode)
       Advertiser        drain 합성 · 후보 잠금 · 광고 키 · 상태 파일          internal/enode
       Worker            명령 · agent · build · merge 단계                    internal/enode
       Deleter           trash 를 보고 뒤에 비운다                            internal/scratch
       Checkpoint Store  spool 의 보고 뒤 판정 · TTL · 재시작 조정              internal/scratch
     namespace helper 셋 (unshare --user --map-root-user --map-auto 뒤)
       runtime-helper    단계의 overlay 세션.  오늘 있다                       internal/enode
       trash-helper      trash 항목 하나를 지운다.  새로                        internal/scratch 의 규칙
       merge-helper      대기 upper 를 lower 에 합친다.  새로                   internal/merge 의 규칙
     형제 데몬들          같은 lower 의 다른 노드 프로세스.  파일과 flock 으로만 만난다
     공유 파일            ~/.local/state/enode/lowers/<fsid>-<ino>/               internal/lower
                         <lower>/.enode-metadata.json                          internal/lower
     제어판 (enode panel) 상태 파일과 정책 파일을 읽는다                         internal/panel

   Mediator
     api                 POST .../exited 하나가 는다 · result 가 칸을 더 싣는다     internal/api
     store               steps 에 phase 칸 셋 · 종료 수락 · QUEUED 의 후보 셋       internal/store
     record              단계 기록에 시각 둘                                     internal/record
```

**Mediator 는 lower · 합치기 · trash 를 모른다.** 새 패키지 셋은 Mediator 가 링크하지
않는다 (5.2 · `component-dependency.md` 2절).

---

## 2. 새 컴포넌트 셋 (Q2 = A)

셋 다 표준 라이브러리와 `golang.org/x/sys/unix` 말고는 임포트하지 않는다. 서로도 임포트하지
않는다. Linux 전용 부분은 `_linux.go`, 나머지 플랫폼은 `_other.go` 가 unsupported 를 돌려준다
(`runc_overlay_other.go` 선례 · 7절 ④).

### 2.1 `internal/lower` — lower 의 신원 · 상태 · 잠금 · metadata

```text
   책임       lower 루트의 신원(statfs f_fsid 와 inode) 과 그 키의 상태 자리
              ~/.local/state/enode/lowers/<fsid>-<ino>/ 의 lower.json · state.json
              두 잠금 — lower.lock (형제 공유 · merge 배타) · bake.lock (굽기 Run 배타)
              잠금 옆 쥔 사람 기록 — 누가(노드) 무엇으로(후보 · Run) 언제부터 (Q1 · Q4)
              <lower>/.enode-metadata.json 을 읽고 쓴다
              env check 의 확인 셋 — scratch 와 워크스페이스의 st_dev · lower 루트 소유 uid ·
              lower.json 신원 (1.4)
   아는 것    경로 · 파일 · flock · 상태 넷(committed · building · pending · merging)
   모르는 것  합치기 규칙 · trash · 단계 · Run 의 계약 · Mediator · environment 의 타입
   실패       상태 자리를 못 열면 그 노드는 굽기와 후보 잠금을 못 한다 -> 노드가 drain 한다.
              단계를 죽이지 않는다
```

**env check 의 결과를 자기 타입으로 낸다.** `environment.Fact` 로 옮기는 것은 노드 쪽
verifier(`internal/enode`)다. 그래야 `internal/lower` 가 Mediator 가 링크하는 패키지를
임포트하지 않는다.

**상태 자리는 결정 3-14 그대로다.** 근거는 계획 1.10 이 다시 쟀다 — 별칭마다 부모가
갈리고(`/work` 와 `/srv/yocto` 는 같은 inode), `/work` 의 부모 `/` 는 root 소유다.

### 2.2 `internal/merge` — 합치기 규칙

```text
   책임       대기 upper 를 걸으며 ADR-077 §4 의 표를 lower 에 적용한다.  종류가 바뀐 항목
              (FR-7 · 사실 7)은 lower 쪽을 trash 로 옮긴 뒤 대체한다
              표시(whiteout · opaque)는 효과를 lower 에 적용한 뒤에만 치운다 — 재개가 같은 절차다
              시작 전 확인 — 같은 filesystem · metacopy 표시 없음 · redirect 표시 없음
              whiteout 읽기 — 문자 장치 0/0 과 xattr 형식.  opaque 는 user.overlay.opaque
   아는 것    경로 셋(upper · lower · trash) · rename · lstat · xattr · rmdir · 속성 맞추기
   모르는 것  상태 파일 · 잠금 · 광고 · namespace · Run
   약속       부르는 쪽이 lower 의 배타 잠금을 쥐고 온다.  그 약속이 「이 lower 의 overlay
              마운트 0」의 증거다 (계획 Q2 답 · helper 마다 마운트 namespace 가 따로라 호스트
              mountinfo 에 형제의 마운트가 안 보일 수 있다.  Functional Design 이 잰다)
   실패       시작 전 확인이 어긋나면 시작하지 않는다.  도중 오류는 멈추고 남은 upper 가 곧
              남은 일의 기록이다 — 같은 절차를 다시 돌리면 끝난다
```

**규칙은 기본 `go test` 에서 잰다.** 가짜 whiteout(`mknod c 0 0`)과 `user.overlay.opaque`
는 특권 없이 만들어진다 (계획 1.3). namespace 는 helper 만의 몫이다.

### 2.3 `internal/scratch` — trash · 배경 삭제자 · Checkpoint Store

```text
   책임       trash — 경로 하나를 <scratch>/trash/ 로 rename 한 번 옮긴다 (FR-4 의 다섯 자리)
              배경 삭제자 — 결과 보고 뒤에, idle IO 로, 새 항목이 오면 깨어나고 데몬 시작 때 한 번
              삭제의 경계 — symlink 를 안 따라가고 trash 밖을 안 지운다 (5.3)
              양 — trash 와 spool 이 차지한 양 · 보존 중인 checkpoint 수 · 삭제자가 도는 중인가 (Q3)
              Checkpoint Store — 보고 전 받아들임(statfs 여유 · 보존 총량) · upper 를 spool 로 rename ·
              보고 뒤 크기 판정과 퇴출 · TTL · 재시작 조정 · 목록과 조회
   아는 것    scratch 경로 · spool 경로 · 정책 값 · rename · 걷기
   모르는 것  lower · 합치기 · 굽기 · Mediator · 단계의 결과
   실패       삭제가 실패하면 항목이 trash 에 남는다 — 여유가 min_free_gb 아래로 가면 노드가
              drain 한다 (FR-4).  capture 가 실패해도 단계 outcome 과 commit set 은 안 바뀐다 (FR-10)
```

**trash 는 굽기 기능이 아니다.** 모든 overlay 단계가 쓴다 (ADR-076 §4.1 「한 기제」).
그래서 `internal/lower` 와 따로 선다.

---

## 3. 기존 컴포넌트의 새 책임

### 3.1 `internal/enode` — 이어 붙이는 쪽

```text
   StepRuntime · 세션      Harvest 를 Finalize 로 바꾼다 (ADR-075 §9 가 「effect 와 producer adapter
                          를 정한 뒤 한 번에」라고 적었고 이 회차가 둘을 정한다).  Close 가 upper 의
                          행선지(trash · 대기 자리 · spool)를 받는다.  runtime 이 capture capability 와
                          workspace.writes 값을 낸다
   runRoot 삭제 다섯 자리   runc_overlay_linux.go 의 Open 실패 갈래(:143 ~ :170) · Close(:437) ·
                          abort(:451) · helper cleanup(:1017) · 준비도 smoke(:1212) — 다섯 다 trash
   Worker 명령 단계         effect 에 따라 수확을 좁힌다 (claim.go:691 ~ :694).  exited 를 보낸다.
                          Finalize 예산과 업로드 예산을 context 로 건다.  진단을 result 로 (Q5)
   Worker agent 단계        오늘 그대로 (claim.go:876 ~ :882).  Git changeset adapter 가 서면 끈다
   Worker build 단계        새로.  sync 뒤 builds 를 차례로 · 항목마다 시각과 exit · IR 유도(ir_tag) ·
                          pinned manifest · upper 를 대기 자리로
   Worker merge 단계        새로.  배타 잠금을 기다리며 쥔 사람을 로그에 쓴다 (Q4) · merge-helper ·
                          metadata · 상태 확정
   Advertiser              drain 합성 — 소유자 정책 · 여유 부족 · 형제의 굽기 중 센 쪽 (2절 ④ · Q3).
                          후보 잠금 — 후보인 동안 쥐고 drain 이 받아 적히면 놓는다 (Q1).
                          광고 키 — workspace.writes · ir · repo.built.<name> · bake.run · bake.resumed ·
                          producer.<name>.  arch 키의 디스크 조건문을 지운다 (detect.go:82 ~ :98)
   상태 파일                drain 의 출처와 scratch 의 양을 더 싣는다.  쓰는 조건이 넓어진다 (Q3)
   helper 입구              trash-helper · merge-helper 를 unshare 뒤에서 연다
   env check verifier      internal/lower 의 확인 셋을 environment.Fact 로 옮긴다 (1.4)
   Client                  Exited 한 메서드 · 업로드 전용 client (요청마다 제한 없음 · 예산 마감) (1.9)
   결과 adapter 둘          Git changeset (edit) · Yocto producer — 범위를 자를 후보 (계획 2절 ⑪)
```

**이 패키지에는 이어 붙이는 코드만 더한다.** 여유가 1.5점(81.5%)이라 규칙은 새 패키지
셋에 두고 여기는 호출과 흐름만 둔다 (Q2 근거).

### 3.2 `internal/contract` — 계약 문법

```text
   effect           read · edit · build · prepare.  안 적으면 run 단계 build · agent 단계 edit
   budget           finalize · upload.  Go duration (ask.timeout.after 선례 · 1.5)
   build 단계        effect: prepare · sync · builds[](name · command) · ir_tag.  종류 build
   merge 단계        merge: {wait}.  종류 merge.  명령이 아니라 노드의 내장 단계
   discover         명시로 켜는 bounded discovery (FR-1)
   produce          producer adapter 를 이름과 판으로 부른다 (⑪)
   검증 (400)        prepare 뒤 같은 역할의 merge 없음 · 이름 규칙(소문자 · 숫자 · -) · 이름 겹침 ·
                    ir_tag 컴파일 실패 · 기간 형식 · finalize 1분 아래 · upload 와 wait 0 이하 (Q6)
```

**상한은 두지 않는다** (Q6). 계약이 노드 설정이나 host 경로를 지정하는 칸은 여전히 없다
(5.3). `ir_tag` 는 값이 아니라 읽는 규칙이다 (Q7).

### 3.3 `internal/store` — 종료 · phase · 대기 사유

```text
   steps 칸 셋       phase · phase_since · exit.  ALTER TABLE ... ADD COLUMN IF NOT EXISTS
   claim            phase 를 적는다 — running, merge 종류면 waiting (ADR-075 §10.3)
   종료 수락         노드 · claimed_instance · attempt 대조.  멱등.  종결된 단계면 조용히 성공 (FR-2)
   result           exited_at · finalize · upload · reason · 진단 · receipt 칸을 봉인한다
   진행 조회         단계마다 phase · phase_since · exit
   QUEUED 의 사유    요구 줄마다 후보 셋 — 살아 있음 · 점유 · drain (완료 조건 4 · 1.7)
```

**`internal/match` 는 안 바뀐다.** 점유 집합과 drain 집합을 따로 넘겨 두 번 부른다.

### 3.4 `internal/api` — 라우트 하나

`POST /v1/runs/{run}/steps/{seq}/exited` 한 줄과 그 핸들러다. `mux.HandleFunc` 18 -> 19,
`internal/api` 전체 등록 27 -> 28. 경로는 정본 표의 것이다 (1.8). result 핸들러는 새 칸을
그대로 store 로 넘긴다.

### 3.5 `internal/record` — 시각 둘

단계 기록(`StepFile`)이 종료 시각과 Finalize 가 끝난 시각을 따로 가진다 (FR-2 · 결정 1-13).
**두 시계가 섞인다** — started_at · ended_at 은 Mediator 의 시각이고 둘은 노드의 시각이다.
어느 칸이 어느 시계인지 적는 규칙은 Functional Design 이 닫는다 (실행 계획 7절 ②).

### 3.6 `internal/environment` — 이음매 한 자리

verifier 가 Fact 를 더 낼 수 있다는 좁은 인터페이스 하나다. **새 내부 임포트가 없다.**
Mediator 는 여전히 이 패키지를 링크하지만 lower 코드는 안 딸려 온다 (1.4 · 5.2).

### 3.7 `internal/panel` — 거짓이 되는 표시를 고친다 (Q3 = A)

오늘 제어판의 drain 은 정책 파일만 읽는다 (`panel/view.go:97`). 노드가 스스로 건 drain 이
「없음」으로 보이게 된다. 상태 파일의 drain 출처와 scratch 의 양을 그리고, 출처마다 누가
풀 수 있는지 붙인다. undrain 버튼은 소유자 정책에만 먹는다는 것이 화면에서 보인다.

### 3.8 `cmd/enode` — 기동과 입구

```text
   기동 순서        상태 자리 열기 -> 굽기 잠금 시도 -> 낡은 상태 정리 · 끊긴 합치기 재개 ->
                  삭제자 첫 회 -> checkpoint 조정 -> 광고 시작 (services.md 4절)
   helper 입구     runtime-helper(오늘) 옆에 trash-helper · merge-helper
   CLI            enode checkpoint — 노드의 보존본 목록과 조회 (⑧).  Mediator 조회 API 는 없다
   client          업로드 전용 client 를 따로 만든다 (main.go:206 의 30초 client 는 그대로)
```

### 3.9 `packaging/macos/examples` — 거짓이 되는 주석 넷

예시 넷의 「디스크가 이 아래로 떨어지면 빌드 능력을 광고에서 뺀다」가 거짓이 된다 (완료
조건 3 ①). 「노드 전체가 drain 한다」로 고친다.

---

## 4. 안 만지는 컴포넌트

```text
   internal/match         매칭 규칙 불변 (품질 게이트 3)
   internal/schema        산출물의 형식 검증 (ADR-020).  진단은 산출물이 아니라 여기를 안 지난다 (Q5)
   internal/proc · internal/transcript · internal/transcriptui · internal/runctl · internal/build
   cmd/mediator · cmd/enodectl · cmd/runctl · cmd/iapadapter
```

`cmd/enodectl` 은 안 바뀌지만 `enodectl env check` 는 enode 를 자식으로 부르므로 새 Fact 셋을
그대로 본다 (1.4).

---

## 5. 실패 등급

단계와 노드에 무엇이 일어나는가로 가른다. 보조로 남는 것은 로그와 진단에 적고 흐름을 막지
않는다.

| 자리 | 실패하면 | 등급 |
|---|---|---|
| exited 전송 | Finalize 와 나란히 재시도. result 가 같은 사실을 다시 싣는다 | 보조 |
| Finalize 예산 초과 | 단계 FAILED · reason `finalize_timeout`. 닫기와 보고는 계속한다 | 단계 |
| 업로드 예산 초과 | 단계 FAILED · reason `upload_timeout` | 단계 |
| trash 로 rename | 같은 scratch 안이라 늘 성립한다. 실패는 세션 닫기 오류로 결과에 남는다 (오늘 규칙) | 단계 |
| 배경 삭제 | 항목이 trash 에 남는다. 여유가 모자라면 노드가 drain 한다 | 노드 |
| capture | receipt 에 `rejected` · `failed` 와 reason. outcome 과 commit set 불변 | 보조 |
| build 의 sync · builds | 하나라도 0 이 아니면 build 단계 실패 · 합치지 않음 · `last_attempt` | 단계 |
| 새 굽기 중복 | 주인이 살아 있으면 곧바로 실패 · reason `bake_in_progress` | 단계 |
| merge 대기 상한 | 굽기 Run FAILED · reason `merge_wait_timeout` · upper 는 trash · committed · drain 풀림 | 단계 |
| merge 시작 전 확인 | 시작하지 않는다. 상태를 어디로 돌리는지는 Functional Design | 단계 |
| 합치기 도중 | 멈춘다. 재개가 남은 upper 로 끝낸다 (재개는 굽기 잠금을 먼저 잡은 노드가) | 노드 |
| 상태 자리 · 잠금 | 굽기와 후보 잠금을 못 한다. 노드가 drain 한다 | 노드 |
| env check 확인 셋 | not ready · 어긋난 것을 이름으로 (FR-8 · 사실 9) | 노드 |
| 상태 파일 쓰기 | 제어판이 「모름」으로 그린다. 광고는 안 막는다 (오늘 규칙) | 보조 |

---

## 6. namespace 안과 밖

1.2 의 사실에서 온다 — upper 에 subordinate uid 소유 항목이 생기고, 노드 uid 는 그 안을
못 걷는다.

```text
   밖 (노드 uid)          디렉터리 하나를 옮기는 rename — runRoot 를 trash 로 · upper 를 대기 자리로 ·
                          upper 를 spool 로.  부모가 노드 uid 소유라 성립한다
                          상태 파일 · 잠금 · metadata · 광고 · 보고
   안 (helper · 같은 매핑)  걷는 일 — 삭제 · 합치기 · 보고 뒤 크기 측정 · 단계 세션
```

**임대 창에 드는 것은 밖의 rename 뿐이다.** 걷는 일은 전부 보고 뒤이거나(삭제 · 측정)
굽기 Run 의 merge 단계 안이다(합치기). 5.4 의 「임대 창이 upper 크기와 무관하다」가 이
경계로 선다.
