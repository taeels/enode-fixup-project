# 응용 설계 계획 — Application Design

입력은 `requirements.md` (Comprehensive · FR-1 ~ FR-13 · 8절의 ① ~ ⑬) · `stories.md`
(스토리 열아홉 · 완료 조건 열) · `plans/execution-plan.md` (7절의 넷) 이다.

- **회차 브랜치**: `v4-run-finalize-bake` · HEAD `b7f7c7a`
- **작성 시각**: 2026-09-24T01:23:48Z
- **닫을 것**: `requirements.md` 8절 ① ~ ⑫ (⑬ 은 Units Generation) · 완료 조건 1 · 2 · 4 · 5 · 6
  의 노드 밖 경로 · `execution-plan.md` 7절의 넷 · `decisions.md` 6절의 준비 구간 한 줄

---

## 1. 착수 전 실측 — 코드를 먼저 읽었다

**`requirements.md` 8절이 「사람에게 물을 것이 아니다」라고 적었다.** 그래서 물을 것을
고르기 전에 답이 코드와 측정에 이미 있는지부터 봤다. 열이 나왔다. **여섯은 설계가
답했고 넷이 질문을 만들었다.**

### 1.1 임대는 매칭 때 생기고 노드는 나중에 안다

```text
   store/claim.go:305     ClaimStep 은 이미 있는 leases 행을 읽는다.  임대는 claim 이 아니라
                          매칭(queue 의 promote)이 만든다.  steps.node_id 도 그때 정해진다
   cmd/enode/main.go:233  노드는 광고 응답(Held.Set)이나 claim 응답(Held.Add)으로 그 임대를 안다
   api/api.go:299 · :335  광고 처리는 drain 을 먼저 커밋하고(UpsertAdvert) 임대 목록을 따로
                          읽는다(RenewLeases).  한 트랜잭션이 아니다
```

**매칭과 노드가 그것을 아는 순간 사이에 창이 있다.** 그 창에서 노드가 아무 잠금도 안
쥐었으면 merge 의 배타 잠금이 잡히고 lower 가 바뀐다. 결정 3-24(「매칭된 뒤 첫 단계 전에
lower 가 바뀌지 않는다」)가 그 자리에서 깨진다. Q1 이 이 자리다.

### 1.2 합치기와 삭제는 namespace 안에서 돈다

```text
   runc_overlay_linux.go:1151   컨테이너 사용자는 노드 uid 로, 컨테이너 root 는 subordinate uid 로
                                매핑된다.  upper 에 subordinate uid 소유의 디렉터리가 생긴다
   runc_overlay_linux.go:210    그것을 다루는 기존 입구가 unshare --map-root-user --map-auto 뒤의
                                enode runtime-helper 다
```

노드 uid 로는 subordinate uid 소유 디렉터리 안의 항목을 `rename` 하지도 지우지도 못한다.
**합치기와 배경 삭제자는 같은 매핑의 helper 프로세스로 돈다.** 규칙은 순수 Go 이고
helper 는 그것을 부르기만 한다 — 그래서 규칙은 기본 `go test` 에서 잰다 (5.1). Q2 가 이
위에 선다.

### 1.3 가짜 표시는 특권 없이 만들어진다 — 8절 ⑫ 을 닫는다

이 기계(커널 6.5 · ext4 · uid 1000 · namespace 없음)에서 쟀다.

```text
   mknod c 0 0 (whiteout)           성공
   mknod c 1 3                      EPERM   whiteout 만 예외라는 커널 규칙과 맞다
   setxattr user.overlay.opaque     성공
   setxattr trusted.overlay.opaque  EPERM   쓰지 않는다.  userxattr 마운트가 user. 이름을 쓴다
```

**가짜 트리의 시험이 기본 `go test` 에서 선다.** 남은 조건 하나 — `user.` xattr 을 못 받는
filesystem(커널 6.6 앞의 tmpfs)에서는 시험이 **실패해야 한다.** 스킵하면 허용목록 밖
스킵 0 의 게이트(5.1)에 걸리고, 조용히 넘기면 표시 없는 트리를 잰다. CI 의
`ubuntu-latest` 에서 한 번 더 재는 것은 Functional Design 이 한다.

### 1.4 env check 에 이미 좁은 이음매가 있다

```text
   environment/check.go:82    RuntimeVerifier — 「environment 가 StepRuntime 구현을 알지 않도록
                              이 좁은 seam 만 둔다」
   cmd/enode/environment.go:16 · main.go:138
                              노드 쪽이 internal/enode 의 구현을 넣는다
   environment/check.go:37    결과는 이름 있는 Fact 의 목록이다
```

**lower 확인 셋(scratch 와 워크스페이스의 `st_dev` · lower 루트 소유 uid · `lower.json`
신원)은 이 이음매로 노드 쪽이 낸다.** `internal/environment` 는 새 내부 임포트 없이
「노드 쪽이 Fact 를 더 낼 수 있다」는 자리 하나만 얻는다. Mediator 는 lower 코드를
링크하지 않고(5.2) `enodectl env check` 도 enode 를 자식으로 부르므로 같은 Fact 를 본다.
`execution-plan.md` 7절 ③ 이 이것으로 닫힌다. 질문이 없다.

### 1.5 계약의 기간 표기와 단계 종류에 선례가 있다

```text
   contract.go:175    ask.timeout.after 가 Go duration 문자열("72h")이다
   contract.go:798    단계 종류는 필드의 존재로 정해진다 — agent · run · acquire · ask.
                      kind 필드는 없다
   contract.go:293    명령 자체에는 시간 상한이 없다.  lease 는 광고가 갱신하는 한 산다
```

**표기는 선례를 따른다.** 예산은 `budget: {finalize: "5m", upload: "10m"}`, merge 는
`merge: {wait: "6h"}` 객체가 있으면 merge 종류다. 남는 것은 상한을 둘지 하나다. Q6 이
이 자리다.

### 1.6 진단이 산출물로 나간다

```text
   claim.go:913       uploadProduced 가 먼저 $OUT 에 workspace.changed 를 쓰고 그것을 걷어 올린다.
                      그래서 그 이름이 produced 에 든다
   diff.go:163        workspace.diff 도 같은 길이다
   저장소 안 소비자     0.  계약 예시 · 시험 데이터 · 훅 어디에서도 이 두 이름을 입력으로 안 읽는다
   정본                run-contract.md:392 · :413 이 저자에게 workspace.changed 를 보라고 권한다
```

ADR-075 §4 는 단계 봉투를 넷으로 가르고 진단을 commit set 과 따로 둔다. 오늘 코드는 진단을
commit set 쪽(produced)에 싣는다. Q5 가 이 자리다.

### 1.7 QUEUED 의 두 집합이 이미 따로 있다 — 완료 조건 4

```text
   store/queue.go:229 ~ :238   busyIn 과 drainingIn 을 따로 읽고 합쳐서 매칭에 넘긴다
   match/match.go:67           Match 는 제외할 노드 집합을 인자로 받는다
```

**두 집합을 따로 넘겨 두 번 부르면 「점유라서」와 「drain 이라서」가 갈린다.**
`internal/match` 의 diff 는 0 이다 (품질 게이트 3). 진행 조회의 요구 줄(`RequireView`)이
후보 수를 셋(살아 있음 · 점유 · drain)으로 싣는다. 질문이 없다.

### 1.8 결과 보고의 wire 는 정본이 이미 적었다

```text
   mediator-api.md:452     exited 의 outcome 은 exit | signal | timeout
   mediator-api.md:471~476 result 가 exited_at · finalize · upload(ok | timeout | error)를 싣는다
   mediator-api.md:286     표는 POST /v1/runs/{run}/steps/{seq}/exited 로 적고
   mediator-api.md:446     절 제목은 POST /v1/steps/{id}/exited 로 적는다.  둘이 갈렸다
```

**구현은 표의 경로를 딛는다** — 기존 result 라우트와 같은 모양이고 `requirements.md`
FR-2 가 그것을 적었다. 절 제목의 갈림은 되돌림 목록(`requirements.md` 11절 mediator-api
줄)에 더한다. reason 코드(`finalize_timeout` · `upload_timeout` · `merge_wait_timeout` ·
`bake_in_progress`)는 result 의 `reason` 칸에, 정본의 `finalize` · `upload` 는 그대로 싣는다.

### 1.9 업로드는 파일을 통째로 읽고 요청마다 30초에 끊긴다

```text
   claim.go:919            uploadProduced 가 os.ReadFile 로 산출물 전체를 메모리에 올린다
   cmd/enode/main.go:206   그것을 PutBlob 이 요청마다 30초인 client 로 보낸다
   cmd/enode/main.go:209   롱폴은 이미 타임아웃 없는 client 를 따로 쓴다 (선례)
```

**업로드 예산의 자리는 client 다.** 업로드는 요청마다의 제한이 없는 client 를 따로 쓰고,
그 단계의 업로드 예산이 context 의 마감으로 모든 요청을 함께 묶는다. 스트림으로 보내는
것도 같은 자리다 — ADR-075 §9 「payload 를 상한 뒤에 통째로 버퍼링하지 않는다」.
`execution-plan.md` 7절 ① 의 자리가 이것으로 닫히고 값은 NFR 값 N3 이다. 질문이 없다.

### 1.10 lower 상태 자리의 근거 하나가 틀렸다 — 결정은 선다

다른 세션이 원장과 트랜스크립트를 대조하다 찾아 알려 왔다. SunnyVM 에서 다시 쟀다
(2026-09-24T01:26Z).

```text
   /srv                 sunny:sunny 755.  원장 두 항목과 ADR-077 §5 가 적은 root:root 가 아니다
   /srv/yocto           sunny:sunny 755
   /work                /srv/yocto 의 bind mount.  findmnt 의 SOURCE 가 /dev/sda2[/srv/yocto]
                        둘 다 ino 6168147 · fsid b517932fee9168d5
   / (/work 의 부모)     root:root 755
```

**결정 3-14(상태와 잠금을 `~/.local/state/enode/lowers/<fsid>-<ino>/` 에)는 그대로 선다.**
근거 둘이 남는다 — lower 옆에 두면 별칭마다 부모(`/` 와 `/srv`)가 달라 자리가 둘로 갈리고,
`/work` 별칭의 부모 `/` 는 노드 사용자가 못 쓴다. 틀린 것은 ADR-077 §5 의 「부모가 root
소유인 기계(SunnyVM 의 `/srv`)」 한 구절이다. 정본 되돌림 목록에 한 줄을 더한다. 원장은
정정 `EN-bb1a4a28` 이 두 항목을 대체했다. 질문이 없다.

---

## 2. 이 단계가 정하는 것 · 안 정하는 것

**정한다.**

```text
   컴포넌트와 패키지 자리        새 노드 쪽 기제 넷과 adapter 둘이 어디 사는가 (⑤ · Q2)
   컴포넌트의 겉면              메서드 시그니처와 입출력 타입.  규칙은 안 적는다
   흐름                        명령 단계 · build · merge · 노드 기동 · 광고 주기 · Mediator 의 여섯
   의존과 경계                  임포트 방향과 boundary_test 에 더할 줄
   사람이 읽는 자리              완료 조건 1 · 2 · 4 · 5 · 6 이 어디에 보이는가 (Q3 · Q4)
```

**설계가 답하는 것 — 묻지 않는다.** 근거가 1절이나 정본에 있다. 반대하면 답에 적어 주세요.

```text
   ①   effect 의 wire 값      read · edit · build · prepare.  ADR-075 §5 의 네 줄을 소문자 한 단어로.
                              안 적으면 run 단계는 build, agent 단계는 edit.
                              명령 단계가 edit 이면 changeset 은 오늘의 git diff 경로(diff.go)다.
                              Git adapter 가 서면 agent 와 같은 adapter 로 옮긴다
   ②   표기                   1.5 의 선례.  budget.finalize · budget.upload · merge.wait
   ④   drain 의 세기          at-boundary > graceful > 없음.  광고는 소유자 것과 노드 것 중 센 쪽을
                              싣는다.  노드가 정책 파일을 고치지 않는다 — 정본은 소유자의 파일이다
   ⑦   phase 칸               steps 에 phase · phase_since · exit 세 칸.  ALTER ... ADD COLUMN IF NOT
                              EXISTS 선례.  claim 이 running 을, merge 단계면 waiting 을 적는다
   ⑧   checkpoint descriptor  receipt 의 checkpoint_capture 가 state · reason · id · scope ·
                              guarantee · node · expires_at 을 싣는다.  host 경로는 안 싣는다.
                              조회는 노드의 CLI(enode checkpoint)다.  Mediator 조회 API 는 안 둔다 —
                              중앙 store 가 순연이고(4-13) 복원 약속이 없다
   ⑨   상한과 양의 자리        노드 설정의 checkpoint · trash 블록.  값은 NFR 값 N1 · N2
   ⑪   Git changeset          patch blob 과 descriptor(base · digest · size · complete).
                              10 MiB 를 넘으면 complete 가 false 이고 patch 를 안 올린다.
                              잘린 patch 는 적용하면 틀리므로 싣지 않는다.  요약은 진단으로 간다.
       producer adapter       노드 바이너리에 든 등록표.  이름과 판(yocto · 1).  평평한 광고 키
                              producer.<이름>=<판> 으로 내고 계약이 그 키를 요구한다 (ADR-012)
   ⑫   가짜 표시              1.3 으로 닫혔다
   7절 ①  업로드 client        1.9
   7절 ③  env check 의 자리    1.4
   7절 ④  크로스 빌드          Linux syscall 을 쓰는 파일은 _linux.go 와 _other.go 짝.
                              _other 는 unsupported 를 돌려준다.  runc_overlay_other.go 가 선례다
   6절    준비 구간의 phase     안 올린다.  스토리와 조각이 요구하지 않는다.  phase 는 열린 어휘라
                              (ADR-075 §10.3) 뒤에 더해도 스키마와 판정이 안 바뀐다
```

**⑪ 은 범위를 자를 후보다.** FR-11 의 두 adapter 는 이 회차에서 가장 큰 새 표면이고
조각 10 만 받는다. 설계는 겉면까지 내고, 자를지는 Units Generation 의 승인이 정한다
(`requirements.md` 9절).

**안 정한다** — 건드리면 옮겨 적기가 된다.

```text
   팩과 요구가 값으로 닫은 것    기본값 넷(1분 · 3분 · 4시간 · 48시간) · reason 코드 · 상태 다섯 ·
                               lower 상태 넷 · 광고 키 셋 · metadata 필드
   Functional Design 몫         합치기 표의 줄마다 동작 · lower 상태 기계와 전이마다의 잠금 ·
                               Q1 의 놓는 조건(울타리) · 두 시계(7절 ②) · 예산의 경계 ·
                               capture 전이 · 삭제자의 경계 · CI 에서의 가짜 표시 재측정
   NFR Requirements 몫          N1 삭제자가 한 번에 지우는 양 · N2 checkpoint 상한과 inode 한도 ·
                               N3 업로드 예산과 요청의 관계
   Units Generation 몫          유닛 분해 · 파일 행렬 · 범위 자르기
```

---

## 3. 산출물 계획 (체크박스)

필수 산출물 다섯이다. **4절 질문의 답이 들어온 뒤 생성한다.** 자리는
`aidlc-docs/v4-run-finalize-bake/inception/application-design/`.

- [x] `components.md` — 컴포넌트 정의와 책임
  - [x] 새 컴포넌트 — Q2 의 답에 따른 노드 쪽 패키지.  무엇을 알고 무엇을 모르나
  - [x] 기존 컴포넌트의 새 책임 — `enode` · `contract` · `store` · `api` · `record` ·
        `environment` · `cmd/enode` (Q3 = A 면 `panel`)
  - [x] 컴포넌트마다 **실패 등급** — 단계를 죽이는 것 · 보조로 남는 것 · 노드를 drain 시키는 것
  - [x] namespace 안에서 도는 것과 밖에서 도는 것의 경계 (1.2)
- [x] `component-methods.md` — 메서드 시그니처 (비즈 규칙은 Functional Design)
  - [x] `StepSession` 의 새 수명 — Finalize 와 Close 의 갈림 (upper 를 trash · 대기 자리 · spool 로)
  - [x] lower 상태와 잠금 · 합치기 · trash · checkpoint store 의 겉면
  - [x] `Client.Exited` 와 업로드 client (1.9)
  - [x] 계약 타입 — `Effect` · `Budget` · `Build` · `Merge` 와 검증 오류
  - [x] store 와 api — `MarkExited` · phase 칸 · 진행 조회 · 요구 줄의 후보 셋 (1.7)
  - [x] environment 의 이음매 한 자리 (1.4)
- [x] `services.md` — 오케스트레이션
  - [x] 명령 단계 — 종료 · exited · Finalize · 닫기 · 업로드 · 보고 · 삭제자.  **예산이 어디서 재나**
  - [x] 굽기 build 단계 — prepare claim · sync · builds · manifest · 대기 자리
  - [x] merge 단계 — waiting · drain · 배타 잠금 · 시작 전 확인 · 합치기 · metadata · 확정.  상한 경로
  - [x] 노드 기동 — 굽기 잠금 · 낡은 상태 정리 · 재개 · 삭제자 첫 회 · checkpoint 조정
  - [x] 광고 주기 — drain 합성 · 후보 잠금 · 상태 파일 · 광고 키
  - [x] Mediator — exited 수락 · merge claim 의 waiting · QUEUED 의 사유
- [x] `component-dependency.md` — 의존과 통신
  - [x] 임포트 그림과 텍스트 대안.  **Mediator 가 새 패키지를 링크하지 않음을 표로**
  - [x] boundary_test 에 더할 줄 (오늘 Mediator 쪽 금지 0 줄)
  - [x] 프로세스 경계 — 데몬 · namespace helper · 형제 노드 프로세스 사이의 잠금과 파일
  - [x] 자료 흐름 — 명령 종료에서 Record 까지 · upper 에서 lower 까지
- [x] `application-design.md` — 위 넷의 통합본
- [x] 검증 — 8절 ① ~ ⑫ · 7절의 넷 · 완료 조건 1 · 2 · 4 · 5 · 6 에 답이 다 붙었나 ·
      5.2 경계 · 5.5 불변식과 대조 · 표기(`emphasis-check.py`)

---

## 4. 결정이 필요한 것 — `[Answer]:` 태그

**일곱이다.** 1절의 실측이 여섯을 답했고(1.3 · 1.4 · 1.7 · 1.8 · 1.9 · 1.10) 2절의
「설계가 답하는 것」이 열둘을 닫았다. 남은 일곱은 값이 아니라 갈림이고, 고르는 쪽에 따라 사람이
보는 것이나 되돌리는 단위가 달라진다.

각 질문에 권장과 근거를 붙였다. `[Answer]:` 뒤에 기호를 적어 주세요. 「권장대로」라고
답하셔도 됩니다.

---

### Q1. lower 공유 잠금을 언제 잡고 언제 놓나 (8절 ③ · 결정 3-24)

형제 overlay 노드는 Run 을 쥔 동안 lower 공유 잠금을 쥐고, merge 는 배타 잠금을 잡는다
(결정 3-9). 1.1 이 그 사이의 창을 보였다 — 임대는 매칭 때 생기고 노드는 나중에 안다.

A) **후보인 동안 잡는다.** 노드는 drain 없이 광고하는 동안과, prepare 가 아닌 Run 의
임대를 쥔 동안 공유 잠금을 쥔다. drain 광고를 Mediator 가 받아 적었다는 응답을 받았고
쥔 임대가 없을 때 놓는다. prepare 단계를 claim 하면 놓는다. Mediator 변경 0

B) 임대가 노드에 닿는 순간(광고 응답 또는 claim) 잡고 prepare 단계를 claim 하면 놓는다.
Mediator 변경 0

C) 임대가 effect 를 싣는다. Mediator 가 임대 행에 prepare 여부를 실어 노드가 닿을 때 잡을지
정한다

D) Other (please describe after [Answer]: tag below)

**권장 A.** 근거 셋이다.

```text
   창을 닫는 것은 A 뿐이다   B 와 C 는 매칭과 노드가 임대를 아는 순간 사이에 잠금이 없다.
                           그 창에서 merge 가 배타를 잡으면 광고한 ir 과 다른 lower 위에서
                           첫 단계가 돈다.  A 는 후보인 노드가 이미 쥐고 있으므로 매칭이
                           잠금 없는 노드에 떨어지지 않는다
   Mediator 가 안 바뀐다     C 는 임대 행과 광고 응답의 wire 를 바꾼다.  FR-4 · FR-8 의 drain 은
                           「Mediator 변경 없음」이다.  그리고 C 도 창을 못 닫는다
   완료 조건 5 의 재료가 된다  잠금을 쥔 노드가 그 옆에 (노드 · Run) 기록을 남기면 merge 가 누구를
                           기다리는지 말할 수 있다 (Q4)
```

A 의 대가 하나 — **놓는 조건이 정확해야 한다.** 광고 처리가 drain 커밋과 임대 읽기를
한 트랜잭션으로 안 한다(1.1). drain 을 읽기 전에 시작한 매칭이 응답 뒤에 커밋되면 그
임대는 그 응답에 없다. 놓는 조건(예: drain 이 받아 적힌 응답을 두 번 연속 받고 임대 0)은
Functional Design 이 닫는다. **대가를 안 보이게 두지 않는다.**

**[Answer]:** A — 채팅에서 논의한 뒤 「권장안대로」(2026-09-24T04:36:54Z)

논의가 더한 것 셋이다. 설계 문서가 이것을 진다.

```text
   불변식의 문장    「매칭된 뒤 첫 단계 전」의 첫 단계는 그 노드가 그 Run 에서 처음 claim 하는
                   단계다.  지킬 것은 「매칭된 순간부터 그 노드가 그 Run 의 임대를 놓을 때까지
                   lower 가 안 바뀐다」이고 decisions.md 3-24 의 문구를 그 뜻으로 고칠 것을 제안한다
   틈은 둘이다      합치는 동안의 매칭(형제가 drain 을 광고하기 전)과 합친 뒤 낡은 광고(ir=X)로
                   들어오는 매칭.  노드의 단계가 Run 의 뒤쪽이면 claim 은 needs 가 풀릴 때까지
                   안 돌아오므로 노드가 임대를 먼저 아는 것은 광고 응답이다
   광고의 순서      합치기가 끝나면 형제는 잠금을 다시 잡고 · metadata 에서 새 ir 을 읽고 ·
                   drain 을 푸는 그 광고에 새 ir 을 싣는다.  둘째 틈을 이 순서가 닫는다
```

---

### Q2. 새 노드 쪽 코드는 어디 사나 (8절 ⑤ · 5.2)

새 기제 넷(trash 삭제자 · lower 상태와 잠금 · 합치기 · checkpoint store)과 env check 의
lower 확인이다. 셋 다 Mediator 가 링크하지 않는 자리여야 한다(5.2). 1.2 대로 합치기와
삭제는 namespace helper 에서 돌고 규칙은 순수 Go 다.

A) **새 패키지 셋.** `internal/lower`(신원 · 상태 · 두 잠금 · metadata · env check 의 Fact) ·
`internal/merge`(합치기 규칙 · 재개 · 시작 전 확인) · `internal/scratch`(trash · 배경 삭제자 ·
checkpoint store). `internal/enode` 는 이들을 부르고 helper 를 연다

B) 새 패키지 하나. `internal/bake` 에 넷을 다 둔다

C) `internal/enode` 안에 파일로 둔다

D) Other (please describe after [Answer]: tag below)

**권장 A.** 근거 셋이다.

```text
   되돌리기 단위가 선다     execution-plan 이 Units Generation 에 「합치기 규칙을 자기 유닛으로」를
                          걸었다.  패키지 경계가 곧 유닛 경계라 합치기를 물려도 lower 상태가 안
                          딸려 간다.  이 회차에서 되돌리기가 가장 어려운 코드다
   커버리지 여유가 안 준다   internal/enode 는 81.5% 다(하한 80).  C 는 새 문장을 거의 다 그 1.5점
                          위에 얹는다.  A 는 패키지마다 자기 80% 를 진다
   trash 는 굽기가 아니다    trash 와 삭제자는 모든 overlay 단계가 쓴다 (ADR-076 §4.1 「한 기제」).
                          B 는 그것을 굽기 이름 아래 넣어, 굽기를 안 쓰는 노드도 굽기 패키지에
                          기대게 만든다
```

A 의 대가 — boundary_test 에 세 줄, 크로스 빌드 대상 셋. 2절의 `_other.go` 규칙이 셋에
다 걸린다.

**[Answer]:** A — 채팅에서 논의한 뒤 「다음」(2026-09-24T04:41:39Z). 반대 없이 넘어간 것을 권장 A 로 읽었다

논의가 더한 것 둘이다.

```text
   패키지 사이의 약속   internal/merge 는 「부르는 쪽이 배타 잠금을 쥐고 왔다」를 전제로 한다.
                       그 약속은 internal/enode 의 merge 단계가 지킨다.  셋은 서로를 임포트하지 않는다
   마운트 0 의 증거     helper 마다 unshare --mount 로 자기 마운트 namespace 를 연다
                       (runc_overlay_linux.go:210 · :818).  형제의 overlay 마운트가 호스트의
                       /proc/self/mountinfo 에 안 보일 수 있다.  Q1 의 규칙에서는 배타 잠금이
                       「도는 형제 세션 0」의 증거가 된다.  다른 프로세스의 mountinfo 를 읽을 수
                       있는지는 Functional Design 이 잰다
```

---

### Q3. 노드 소유자가 drain 의 출처와 trash · spool 의 양을 어디서 보나 (완료 조건 1 · 2 · 8절 ④)

drain 의 출처는 셋이다 — 소유자 정책 · 여유 부족 · 형제의 굽기. 소유자 것만 소유자가
풀어야 풀리고 나머지 둘은 저절로 풀린다 (US-1).

A) **노드 상태 파일(`<stem>.status.yaml`)에 싣고 제어판이 그린다.** 광고와 Mediator 는 안
바뀐다

B) 새 CLI `enode status` 가 출력한다. 제어판은 안 바뀐다

C) 광고가 drain 의 출처를 싣고 `GET /v1/nodes` 가 보인다

D) Other (please describe after [Answer]: tag below)

**권장 A.** 근거 셋이다.

```text
   통로가 이미 있다         광고 주기가 상태 파일을 쓰고(advertise.go:237) 제어판이 읽는다
                          (panel/view.go:86).  ADR-068 = A 가 그렇게 세웠다
   푸는 자리에서 본다        제어판이 소유자가 drain 을 걸고 푸는 자리다 (panel.go:86 · :87).
                          draining 을 본 사람이 거기서 「이것은 내가 풀 것이 아니다」를 같이 봐야
                          여유 부족을 고장으로 읽고 디스크를 늘리지 않는다 (US-2)
   광고 어휘가 안 는다       C 는 FR-4 의 「새 광고 어휘도 Mediator 변경도 없다」를 깬다.
                          완료 조건 1 은 노드 쪽에서 선다고 적었다
```

A 의 대가 — `internal/panel` 이 만지는 패키지에 든다 (`execution-plan.md` 1.1 의 「고르면」
줄). 제어판을 LAN 에 안 연 소유자도 그 기계에서는 본다.

**[Answer]:** A — 채팅에서 논의한 뒤 「그래 다음」(2026-09-24T04:44:47Z)

논의가 더한 것 셋이다.

```text
   오늘 표시가 거짓이 된다   제어판의 drain 은 정책 파일만 읽는다 (panel/view.go:97).  노드가 스스로
                          건 drain 은 제어판에 「없음」으로 보이고 undrain 은 지울 것이 없다.
                          Q3 은 더 보이는 자리가 아니라 거짓이 되는 표시를 고치는 자리다
   상태 파일을 쓰는 조건     오늘은 탐지 시각이 바뀔 때만 쓴다 (advertise.go:237).  drain 과 scratch
                          값이 바뀔 때도 쓴다
   trash 의 양은 늦은 값     보고 전 창에 순회를 안 들인다 (5.4).  삭제자가 지우며 세거나 보고 뒤
                          배경에서 잰다.  값에 잰 시각이 붙는다.  여유(statfs)는 즉시 읽는다
```

---

### Q4. 노드 기계 밖에서 굽기의 기다림과 재개를 어떻게 보나 (완료 조건 5 · 6)

Mediator 는 lower 를 모른다 (4-7 기각). 굽기 담당(P4)은 노드 기계에 들어간다고 가정하지
않는 사람이다. 봐야 할 것은 둘이다 — merge 가 **누구를** 기다리는지(US-12 · US-15), 그리고
FAILED 로 봉인된 굽기가 **재개로 합쳐졌는지**(US-18).

A) **기존 통로 둘로 나른다.** 기다림은 merge 단계의 실시간 로그가 말한다 — 형제가 잠금
옆에 남긴 (노드 · Run) 기록을 merge 가 읽어 주기마다 한 줄 쓰고 대기 상한 시각을 함께
쓴다. 재개는 lower 의 마지막 굽기를 정보 키(`bake.run` · `bake.resumed`)로 `ir` 옆에
광고해 `GET /v1/nodes` 에서 보인다

B) 둘 다 노드 제어판에만 보인다

C) Mediator 에 새 표면을 둔다 — merge 단계가 기다리는 상대를 보고하는 필드와 재개 표시

D) Other (please describe after [Answer]: tag below)

**권장 A.** 근거 셋이다.

```text
   Mediator 가 lower 를 안 배운다   단계 로그는 앞 회차가 세운 실시간 업로드(claim.go:329)로
                                  중앙 화면에 뜬다.  광고의 정보 키는 machine · ws 가
                                  선례다(detect.go:64 · :77).  둘 다 Mediator 에는 글자다
   id 로 맞춘다                    FAILED 인 자기 Run 의 id 가 그 노드의 bake.run 에 있으면
                                  합쳐진 것이다.  노드 기계에 안 들어간다 (완료 조건 6)
   B 는 사람을 못 만난다            제어판은 노드 소유자의 자리다.  P4 는 노드 소유자가 아닐 수
                                  있다 (순연 4-15 의 전제)
```

A 의 대가 — 광고 키 둘이 는다. 매칭이 그 키를 요구할 수 있게 되지만 부정 조건이 없어
해가 없다 (ADR-011). 기다림의 한 줄은 구조가 아니라 글이다 — 사람이 읽는 자리라서 그렇게
둔다.

**[Answer]:** A — 채팅에서 논의한 뒤 「그래 다음」(2026-09-24T06:21:06Z)

논의가 더한 것 넷이다.

```text
   풀어 쓴 뜻        merge 가 기다리는 동안 밖에서 「누구의 어느 Run 이 끝나기를 기다리나」와
                    「언제 포기하나」를 본다.  설계 문서는 이 문장으로 적는다
   후보도 보인다      Q1 의 울타리로 drain 확인 전인 형제는 「candidate」로 한 줄에 선다.
                    기다림의 한두 주기가 거기서 설명된다
   마지막 굽기 하나   bake.run 은 lower 의 마지막 굽기만 보인다.  다음 굽기가 지나가면 가려지고
                    그 뒤는 metadata 의 previous_ir 사슬에만 남는다
   상한 시각의 시계   merge 의 phase_since 는 Mediator 가 claim 때 적는다.  로그의 상한 시각은
                    노드 시계다.  한 줄에 어느 시계를 쓸지는 Functional Design (7절 ②)
```

---

### Q5. build/test 단계의 누락 산출물과 collect 실패는 어디에 남나 (8절 ⑥ · FR-1)

걷기를 끄면 `workspace.changed` 의 셋째 절(바뀐 파일 목록)이 사라진다. 앞의 두 절(계약이
요구했는데 없는 이름 · collect 가 못 걷은 이유)은 남아야 한다. 1.6 대로 오늘 그 파일은
produced 에 든다.

A) **결과 보고의 진단 칸에 구조로 싣고(Record 에 봉인) 같은 문장을 그 단계 로그 끝에 한 번
쓴다.** 로그의 그 자리에 「변경 목록을 재지 않았다 — effect 가 build 다. 바뀐 파일을 남기려면
effect: edit 를 적는다」를 쓴다. `workspace.changed` 산출물은 걷기가 돈 단계(agent 단계 ·
bounded discovery)에만 나온다

B) 오늘처럼 `workspace.changed` 산출물에 쓰고 셋째 절만 「재지 않았다」로 바꾼다

C) 새 산출물 이름(`step.notes`)으로 쓴다

D) Other (please describe after [Answer]: tag below)

**권장 A.** 근거 셋이다.

```text
   정본의 봉투대로 간다     ADR-075 §4 는 진단을 commit set 과 따로 둔다.  B 와 C 는 결과가
                          아닌 것을 produced 에 계속 싣는다.  B 는 이름(changed)이 내용과 어긋난다
   완료 조건 3 ④ 가 선다    명령 단계의 workspace.diff 가 조용히 안 나오는 것을 겪는 사람이 읽는
                          자리가 지금 없다.  A 는 그 사람이 이미 보는 로그에 그 한 줄을 둔다
   깨지는 소비자가 없다     저장소 안에서 두 이름을 입력으로 읽는 곳이 0 이다 (1.6)
```

A 의 대가 — 정본 `run-contract.md:392 · :413` 의 두 문장을 고쳐야 한다. 이미 되돌림
목록에 있다 (`requirements.md` 11절). agent 단계는 안 바뀐다 (FR-1).

**[Answer]:** A — 채팅에서 논의한 뒤 「그래 다음」(2026-09-24T06:25:26Z)

논의가 더한 것 셋이다.

```text
   진단의 모양        result.diagnostics 에 missing · collect(name · why) · changes(not_measured 와
                     그 effect).  로그 끝의 문장은 같은 사실을 사람 말로.  effect: edit 안내를 붙인다
   훅은 안 걸린다      훅은 workspace.changed 파일을 안 읽고 같은 걷기를 스스로 돈다 (hook.go:281).
                     requirements.md FR-1 의 「훅이 workspace.changed 를 읽는다」는 파일이 아니라
                     걷기를 가리킨다
   잃는 것 하나        in.from 은 산출물 blob 만 깐다.  build/test 단계의 진단을 다음 단계가 입력으로
                     못 받는다.  저장소 안에 그렇게 쓰는 계약은 0 이다
```

---

### Q6. 예산 둘과 merge 대기 상한에 노드 쪽 상한을 두나 (8절 ② · `decisions.md` 6절)

표기는 1.5 의 선례로 닫혔다. 남은 것은 계약이 준 값을 누가 어디서 제한하는가다.

A) **노드 쪽 상한을 두지 않는다.** 계약 검증이 형식과 하한만 막는다 — Finalize 예산은 기본
1분 아래로 못 내린다(「늘릴 수 있다」). 소유자의 수단은 오늘처럼 drain 이다

B) 노드 설정에 상한을 둔다. 계약이 넘으면 노드가 상한으로 자르고 진단에 적는다

C) Mediator 의 계약 검증이 고정 상한을 둔다. 넘으면 400

D) Other (please describe after [Answer]: tag below)

**권장 A.** 근거 셋이다.

```text
   같은 위협의 문 하나만 닫는다   명령 자체에 시간 상한이 없다 (contract.go 에 run 의 timeout 없음).
                               계약은 오늘도 명령으로 노드를 원하는 만큼 쥔다.  예산에만 상한을
                               두면 한 문만 닫고 막았다고 적게 된다
   근거 없는 숫자가 된다         B 와 C 의 상한값을 받칠 측정이 없다.  정본의 기본값 넷은 사용자
                               결정과 실측(79분 빌드)으로 섰다
   소유자의 수단이 이미 있다      drain(at-boundary)은 도는 Run 을 끝까지 두고 다음을 안 받는다
                               (ADR-063).  노드를 되찾는 길이 예산 상한과 별개로 선다
```

A 의 대가 — 한 계약이 예산을 크게 적어 Finalize 에서 오래 머물 수 있다. 그 사실은
`phase_since` 와 계약의 예산으로 밖에서 보인다 (US-8). Mediator 가 강제하는 길은 순연 4-3
의 여는 조건이 그대로 진다.

**[Answer]:** A — 채팅에서 논의한 뒤 「그래 다음」(2026-09-24T06:27:03Z)

논의가 더한 것 둘이다.

```text
   검증의 모양        budget.finalize 는 1분 아래가 400 (늘릴 수만 있다).  budget.upload 와 merge.wait 는
                     0 이하가 400.  셋 다 Go duration 이다
   merge.wait 의 성질  merge 가 기다리는 동안 형제는 drain 이다.  큰 merge.wait 는 그 lower 위 노드의
                     수용력을 오래 묶는다.  노드 쪽 제한은 순연 4-15(노드 소유자가 굽기를 받을지)와
                     같은 갈래라 그 행에 한 줄을 더하고 함께 연다.  그동안 소유자의 수단은 굽기 Run
                     취소다 — upper 는 trash 로, lower 는 committed 로, drain 이 풀린다
```

---

### Q7. IR 태그를 가려내는 형식 규칙은 어디 사나 (8절 ⑩ · 완료 조건 10)

IR 은 sync 뒤 manifest HEAD 에 정확히 붙은 태그에서 유도한다 (결정 3-20). HEAD 에 태그가
여럿일 수 있다. 사내 형식은 `IR<YYMMDD>_<HHMMSS>` 다. 유도 결과의 규칙은 설계가 닫는다 —
형식에 맞는 태그가 하나면 그것, 0 이면 null(`no_ir_tag`), 둘 이상이면 null
(`ambiguous_ir_tag`). 완료 조건 10 이 그 이유를 merge 결과에서 본다.

A) **굽기 계약의 build 단계에 `sync` 옆에 둔다** — `ir_tag`(정규식). 안 적으면 사내 형식이
기본이다

B) 노드 설정에 둔다

C) 코드에 사내 형식을 고정한다

D) Other (please describe after [Answer]: tag below)

**권장 A.** 근거 셋이다.

```text
   읽는 규칙이 쓰는 명령 옆에 산다   어느 IR 을 받을지는 sync 명령이 말한다(refs/tags/IR…).
                                  그것을 되읽는 규칙이 같은 자리에 있어야 둘이 안 어긋난다.
                                  결정 3-19 의 「이름과 명령이 한 자리에」와 같은 이유다
   굽기마다 한 값이다               B 는 굽기 노드의 소유자마다 규칙이 갈릴 수 있다.  유도한 ir 은
                                  metadata 로 그 lower 의 모든 형제가 광고한다
   IR 칸이 아니다                  결정 3-20 은 IR 값을 계약에 안 싣는 것이다.  이것은 값이 아니라
                                  읽는 규칙이고 노드 설정도 host 경로도 아니다 (5.3)
```

A 의 대가 — 계약 문법이 한 칸 는다(정규식은 RE2 라 선형 시간이다). 컴파일되지 않으면
400 이다. 정본 `run-contract.md` 의 되돌림에 이 칸이 붙는다.

**[Answer]:** A — 채팅에서 논의한 뒤 「그래」(2026-09-24T06:33:21Z)

논의가 더한 것 둘이다.

```text
   시험 lower        SunnyVM 의 시험 lower 는 poky 다 (EN-8e9db691).  태그가 사내 형식이 아니라
                    C 였으면 장면 2 의 5 를 재려고 시험 저장소에 사내 형식 태그를 억지로 긋는다
   같은 커밋의 IR 둘   manifest 가 안 바뀐 채 IR 을 두 번 그으면 둘이 같은 트리를 가리킨다.
                    null 로 둘지 늦은 쪽을 고를지는 Functional Design 이 사내 IR 긋기 방식을
                    확인하고 정한다.  늦은 쪽의 정렬 규칙도 형식 곁(ir_tag)에 산다
```

---

## 5. 답이 들어온 뒤의 순서

```text
   1   답 일곱을 읽고 모순 · 모호를 본다.  있으면 clarification 파일을 낸다
       (question-format-guide.md 의 MANDATORY)
   2   3절의 체크박스를 순서대로 돌리고 그 자리에서 [x] 로 바꾼다
   3   산출물 다섯을 낸다.  8절 ① ~ ⑫ · 7절의 넷 · 완료 조건 다섯에 답이 다 붙었는지
       마지막에 센다
   4   확장 준수 요약을 붙인다 — 셋 다 꺼져 있다.  팩의 보안 표는 요구로 대조한다
   5   승인을 받고 커밋한다.  다음은 Units Generation 이다
```

---

## 6. 답과 분석 (2026-09-24T06:33:21Z)

일곱을 채팅에서 하나씩 논의했고 **일곱 다 A 로 닫혔다.** 사용자의 말은 Q1 「권장안대로」 ·
Q2 「다음」 · Q3 ~ Q6 「그래 다음」 · Q7 「그래」다. Q2 는 반대 없이 넘어간 것을 A 로 읽었고
채팅에서 그 읽기를 밝혔다.

```text
   Q1 = A   lower 공유 잠금은 노드가 매칭 후보인 동안 쥔다
   Q2 = A   새 패키지 셋 — internal/lower · internal/merge · internal/scratch
   Q3 = A   drain 의 출처와 scratch 의 양은 상태 파일에 싣고 제어판이 그린다
   Q4 = A   merge 의 기다림은 단계 로그로, 재개는 광고 정보 키(bake.run · bake.resumed)로
   Q5 = A   build/test 의 진단은 result 의 진단 칸과 단계 로그로.  produced 에서 뺀다
   Q6 = A   예산과 merge 대기에 노드 쪽 상한을 두지 않는다.  검증은 형식과 하한만
   Q7 = A   IR 태그 규칙은 굽기 계약의 build 단계 ir_tag.  기본은 사내 형식
```

### 모순 · 모호 분석 (Step 8 · MANDATORY)

**추가 질문 0.** 짝마다 대 봤다.

```text
   Q1 x Q2   후보 잠금의 자리는 internal/lower, 쥐고 놓는 시점은 internal/enode (광고 루프와
             prepare claim).  패키지 경계와 책임이 겹치지 않는다
   Q1 x Q3   drain 출처 셋째(형제의 굽기)에 「노드가 뜰 때 lower 가 merging 이라 잠금을 못 잡음」이
             든다.  상태 파일이 같은 출처로 보인다
   Q1 x Q4   잠금 옆의 쥔 사람 기록 하나가 둘에 쓰인다 — merge 의 기다림 로그와 후보의 표시
   Q3 x Q4   보는 사람이 다르다 — 노드 소유자는 제어판, 굽기 담당은 Mediator 의 로그와 광고.
             bake.run 은 상태 파일의 능력 목록에도 저절로 보인다
   Q5 x 완료 조건 3 ④   로그 끝의 한 줄이 그 조건의 자리다
   Q6 x Q1   큰 merge.wait 는 형제의 drain 을 길게 한다.  Q1 의 잠금 규칙은 그 길이를 안 바꾼다.
             노드 쪽 제한은 순연 4-15 에 붙였다
   Q7 x Q2   IR 유도는 build 단계(internal/enode)가 세션 안에서, 기록은 internal/lower 의 metadata 가
```

### 답이 낳은 파생 결정

```text
   Q1   합치기가 끝나면 형제는 잠금을 다시 잡고 · metadata 에서 새 ir 을 읽고 · drain 을 푸는
        그 광고에 새 ir 을 싣는다.  decisions.md 3-24 의 문구를 「매칭된 순간부터 그 노드가 그 Run
        의 임대를 놓을 때까지」로 고칠 것을 제안한다
   Q3   상태 파일을 쓰는 조건이 「탐지 시각이 바뀔 때」에서 「drain 이나 scratch 값이 바뀔 때도」로
        넓어진다
   Q6   순연 4-15 에 한 줄 — merge 대기의 노드 쪽 제한도 그 정책과 함께 연다
```

### 정본 되돌림에 더할 것 (이 단계가 찾은 것)

```text
   ADR-077 §5        「부모가 root 소유인 기계(SunnyVM 의 /srv)」 -> /work 별칭의 부모 / (1.10)
   mediator-api.md   exited 절 제목의 경로를 표와 같게 (1.8)
   run-contract.md   budget · merge.wait · ir_tag · effect 값 넷 (Q6 · Q7 · 2절 ①)
   ADR-077 §4        확인 넷 중 「마운트 0」은 배타 잠금이 증거를 진다 — Functional Design 이 확인하면
```
