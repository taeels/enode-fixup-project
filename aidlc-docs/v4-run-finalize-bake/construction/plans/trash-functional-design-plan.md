# `trash` — Functional Design 계획

**유닛** `trash` (버리기와 여유 공간) · **브랜치** `unit/trash` · **담당** taeels ·
**회차** `v4-run-finalize-bake` (굽기) · **순서** 여덟 중 넷째 · **앞 유닛** finalize (`main` `3f98c8f` 에 병합) ·
**맡는 조각** 4 (trash · 사람 · SunnyVM) · **완료 조건** 1 (drain 의 출처 셋을 나눠 본다 — 이 유닛은 소유자 ·
여유 부족 둘) · 2 (trash 의 양과 삭제자를 본다) · 3 ① (`min_free_gb` 의 뜻이 바뀐 것을 예시 설정이 적는다) ·
**스토리** US-1 (누가 건 drain 인지) · US-2 (디스크 중 얼마가 곧 지워질 trash 인지) · US-3 (여유가 모자라면 이제
노드 전체가 빠진다)

입력은 유닛 정의(`unit-of-work.md` 4절) · 요구(`requirements.md` FR-4 trash 와 여유 drain · 5.3 보안 표의 trash 삭제 줄 ·
5.4 보고 전 창) · 설계(`services.md` 1절 닫기 · 4절 노드 기동 · 5절 광고 주기 · `components.md` 2.3 · 3.1 · 3.7 · 3.9 ·
5절 실패 등급 · 6절 namespace 안과 밖 · `component-methods.md` 3.1 · 4.3 · 4.4 · 8절 · `component-dependency.md` 3절) ·
조각 정의(`requirements/finalize-bake/scene-gates.md` 2절 조각 4 · `requirements.md` 6절이 바꾼 확인 방법) ·
팩의 결정 2-5 (trash 와 배경 삭제자) · 2-6 (여유 부족이면 광고에서 빠진다) · 앞 유닛이 넘긴 일
(`construction/finalize/functional-design/business-logic-model.md` 11절)이다. 이 계획은 그 위에서
**단계가 남긴 작업 폴더를 어떻게 버리고, 여유가 모자랄 때 노드가 어떻게 스스로 빠지나**만 짓는다. 코드는 다음 단계다.

---

## 0. 이 단계가 닫는 것과 안 닫는 것

```text
   닫는다     작업 폴더(runRoot)를 버리는 다섯 자리를 trash 로 옮기기 한 번으로 — 이름 · 자리 · 실패
             닫기를 Finalize 예산 안으로 (finalize 가 넘긴 일)
             데몬이 죽어 남은 작업 폴더를 어떻게 알아보고 치우나
             배경 삭제자 — 언제 깨나 · 무엇을 한 번에 · namespace 안의 삭제 경계 · 실패하면
             trash 의 양을 측정하는 법과 그 값이 가는 자리
             여유 부족 drain — 거는 조건과 푸는 조건 · 어느 노드가 거나 · 소유자 drain 과 합치는 규칙
             arch 키에서 디스크 조건문을 지운다
             상태 파일의 새 칸(drain 의 출처 · trash 의 양)과 쓰는 조건
             제어판 — drain 의 출처와 누가 풀 수 있나 · trash 의 양
             예시 설정 넷의 min_free_gb 주석
             코드 경계 시험에 더할 줄
             조각 4 의 확인 모양

   안 닫는다   굽기 출처의 drain (bake) · 후보 잠금                              lower-state 유닛
             upper 를 대기 자리로 (Keep.Upper 에 경로)                            bake 유닛
             spool 과 checkpoint (Keep.Upper 에 spool 자리 · 보존 중인 수)          checkpoint 유닛
             env check 의 scratch filesystem 확인                               lower-state 유닛
             값의 크기 — 삭제자가 한 번에 지우는 양과 속도 (N1)                     이 유닛의 NFR Requirements
             Mediator                                                           바뀌지 않는다
```

---

## 1. 실측 — 코드가 지금 어떻게 생겼나 (2026-09-26 · `3f98c8f`)

### 1.1 작업 폴더를 지우는 다섯 자리 (`internal/enode/runc_overlay_linux.go`)

작업 폴더는 `os.MkdirTemp(<scratch>, "enode-runc-")` 다 (`:138`). scratch 는 노드 설정의
`environment.scratch` 다 (`config.go:117`).

```text
   ①  Open 실패 갈래     :144 · :160 · :165 · :171   막 만든 빈 폴더.  노드 uid 로 지운다
   ②  세션 Close         :489 (보통) · :457 (abort 뒤)  helper 가 닫은 뒤 남은 것을 노드 uid 로 지운다
   ③  abort              :503                          helper 를 죽이고 노드 uid 로 지운다
   ④  helper cleanup     :1094                         namespace 안에서 unmount 뒤 RemoveAll.  upper 크기에 비례한다
   ⑤  준비도 smoke        :1339 (Close) · :1352         smoke 의 세션도 ② 를 지난다.  닫은 뒤 폴더가 없는지 본다
```

**오늘 큰 비용은 ④ 다.** upper 에는 subordinate uid 소유 항목이 생겨 노드 uid 는 그 안을 못 걷는다
(`components.md` 6절). 그래서 helper 가 namespace 안에서 지우고, 그 시간이 보고 전 창에 든다 —
finalize 유닛은 이것을 Finalize 예산 밖에 두었다 (`finalized_at` 에 그 시간이 들어 있다).

**데몬이 죽으면 작업 폴더가 scratch 에 남는다.** helper 는 `Pdeathsig` 로 죽고(`:157`), 그 뒤 ④ 가 안 돈다.
오늘 그것을 치우는 코드는 없다.

### 1.2 여유와 arch 키 (`internal/enode/detect.go`)

```text
   :82 ~ :98   툴체인이 있어도 hasRoom 이 거짓이면 arch · arch.<이름> 키를 안 싣는다.  노드는 그대로 광고한다
   :387        hasRoom — 워크스페이스의 statfs 여유(GB) >= min_free_gb.  측정하지 못하면 넉넉한 것으로 친다
   config.go:139   min_free_gb 의 기본값은 10 이다 — 적지 않은 노드도 10 GB 를 본다
```

**오늘 여유가 모자라면 빌드만 안 받고 agent 단계는 계속 받는다** (US-3). native 노드도 같다.

### 1.3 drain 과 상태 파일 (`internal/enode`)

```text
   advertise.go:183    광고의 policy 는 정책 파일에서 읽은 소유자 drain 하나다 (policy.go 의 policyReader)
   advertise.go:175    상태 파일은 탐지 시각(At)이 바뀔 때만 쓴다.  칸은 caps 와 at 둘
   leases.go:85        응답의 drain 을 Worker 에 나른다.  받아 적힌 drain 이면 Worker 가 claim 을 멈춘다
   contract/advert.go  drain 어휘 셋 — "" · graceful (새 임대만 막는다) · at-boundary (경계에서 닫는다)
```

### 1.4 제어판 (`internal/panel`)

```text
   view.go:98        drain 은 정책 파일만 읽는다 — 노드가 스스로 건 drain 이 「없음」으로 보이게 된다
   handlers.go:30    drain 걸기 · 풀기는 정책 파일을 쓴다.  소유자 drain 에만 먹는다
   page.go:243       「drain 풀기」 버튼
```

### 1.5 finalize 가 넘긴 일 (`construction/finalize/functional-design/business-logic-model.md` 11절)

```text
   닫기를 rename 으로 바꾸는 커밋에서 닫기를 Finalize 예산 안으로.  afterExit 의 session.Close 에 Finalize ctx
   Keep.Upper 가 "" 면 runRoot 를 trash 로.  finalized_at 의 뜻(닫기가 끝난 때)은 그대로
   runcOverlaySession.Close 의 RemoveAll 두 자리(보통 · abort 뒤)가 바뀔 자리다
```

### 1.6 예시 설정 넷 (`packaging/macos/examples`)

`local.yaml` · `colima.yaml` · `qemu.yaml` · `zephyr.yaml` 이 「디스크가 이 아래로 떨어지면 빌드 능력을
광고에서 뺀다」고 적는다. 이 유닛 뒤에 거짓이 된다.

---

## 2. 물음 열

답을 `[Answer]:` 뒤에 적는다. 권장을 **A** 에 둔다. 기호 옆에 뜻을 적었다.

### Question 1 — 닫기를 Finalize 예산 안으로 (finalize 가 넘긴 일)

```text
   종료 status          닫기 (unmount · trash 로 rename)      finalized_at
        |-- Finalize --|-- 닫기 --|                             |
   A    [ Finalize 예산 ---------- ]   닫기는 마감으로 끊지 않는다
   B    [ Finalize 예산 ]              닫기는 예산 밖 그대로
```

A) **닫기를 Finalize 예산 안에 넣는다. 다만 닫기 자체는 마감으로 끊지 않는다** — 반쯤 닫은 세션은 마운트나
   helper 를 남긴다. 닫기가 끝난 시각이 마감을 넘었으면 `finalize_timeout` 이다 (Finalize 가 이미 넘겼으면
   그대로). 닫기가 rename 한 번과 unmount 라 짧아서 예산 안에 둘 수 있다 — `services.md` 1절의 모양이 된다.
   `finalized_at` 의 뜻(닫기가 끝난 때)은 그대로다

B) 닫기를 예산 밖에 그대로 둔다. rename 이라 짧으므로 차이가 거의 없다. 대신 `services.md` 1절과 모양이
   다르게 남는다

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2 — trash 의 자리 · 이름 · 데몬이 죽어 남은 작업 폴더

같은 scratch 를 두 데몬이 쓸 수도 있다 — 노드 설정이 경로를 정하고, 한 기계의 형제 노드가 같은 경로를
적을 수 있다. 남은 폴더를 치울 때 다른 데몬의 살아 있는 세션을 옮기면 그 단계가 깨진다.

A) **trash 는 `<scratch>/trash/` 이고 항목 이름은 작업 폴더 이름 그대로다** (`enode-runc-XXXX` — 이미 유일하다).
   **세션마다 작업 폴더 안의 잠금 파일을 Open 이 쥐고(flock) 닫을 때 놓는다.** 데몬이 기동할 때 scratch 의
   `enode-runc-*` 중 **잠금을 쥘 수 있는 것만** — 쥔 프로세스가 죽었다는 뜻이다 — trash 로 옮긴다. 다른 데몬의
   살아 있는 세션과 도는 `enode env check` 의 smoke 는 잠금이 쥐어져 있어 건드리지 않는다. 잠금은 커널이
   프로세스가 죽을 때 푼다

B) trash 자리와 이름은 A 와 같다. 기동 때 scratch 의 `enode-runc-*` 를 **전부** trash 로 옮긴다 — 한 scratch 를
   한 데몬만 쓴다는 전제다. 설정이 그 전제를 어기면 다른 데몬의 세션을 깬다

C) 남은 작업 폴더는 오늘처럼 두고, trash 로 들어온 것만 지운다. 사람이 치운다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3 — 배경 삭제자가 언제 · 무엇을 · 어떻게 지우나

trash 의 항목은 subordinate uid 소유라 노드 uid 가 못 지운다 — runtime 과 같은 uid 매핑의 user namespace 안에서
지운다 (FR-4 · `component-methods.md` 4.4 의 `enode trash-helper <trash> <entry>`).

A) **항목 하나에 helper 하나, 한 번에 하나씩 차례로 지운다.** helper 는 IO 우선순위를 idle 로, CPU 우선순위를
   가장 낮게 내리고 돈다. **도는 단계가 있어도 멈추지 않는다** — idle 이라 단계에 양보한다. 깨는 때는 셋 —
   데몬 기동 · 결과 보고 뒤(Kick) · 실패한 항목이 남았으면 10분마다. 지우기에 실패한 항목은 trash 에 남고
   노드 로그에 이유를 남긴다 (`components.md` 5절 — 등급은 노드). 한 번에 지우는 양과 속도의 값은 NFR (N1) 이다

B) A 와 같되 **임대가 없을 때만** 지운다 — 단계가 도는 동안은 쉰다. 단계의 IO 와 절대 안 겹친다. 대신 쉬지
   않고 일이 들어오는 노드는 trash 가 안 비어 여유 drain 에 걸린다

C) helper 하나로 trash 전체를 한 번에 비운다 (항목마다 helper 를 안 연다). helper 를 여는 비용이 준다. 대신
   한 항목의 실패가 나머지를 막지 않게 helper 안에서 나눠야 한다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4 — 삭제의 경계 (보안 표의 trash 삭제 줄)

팩 보안 표 — 「user namespace 안에서 지운다. symlink 를 따라가지 않는다. trash 밖을 지우지 않는다」.

A) **helper 는 항목 이름을 trash 의 바로 아래 이름 하나로만 받는다** — 빈 이름 · `.` · `..` · `/` 가 든 이름은
   거절한다. trash 자체가 symlink 면 거절한다. 지우기는 디렉터리를 열어 그 안의 이름을 지우는 방식이라
   symlink 를 따라가지 않는다(링크 자체만 지운다). **권한 000 인 디렉터리**(overlay 의 `work/work`)는 들어가기
   전에 권한을 풀어 지운다 — namespace 안의 root 는 매핑된 uid 의 파일에 권한 검사를 넘을 수 있지만, 그
   가정을 코드가 기대지 않게 한다. 마운트가 남아 있으면(helper 가 unmount 를 못 했다) 지우지 않고 그 항목을
   남긴다 — 다른 filesystem 으로 넘어가지 않는다

B) A 와 같되 마운트 확인을 하지 않는다 — trash-helper 는 마운트 namespace 를 안 여니 남은 마운트가 보이면
   그대로 지운다

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5 — trash 의 양을 어떻게 재나 (US-2 · 완료 조건 2)

노드 uid 는 subordinate uid 소유 디렉터리와 권한 000 디렉터리 안을 못 걷는다 — 양도 namespace 안에서 측정한다.
보고 전 창에는 걷기를 들이지 않는다 (5.4).

A) **삭제자가 항목을 지우기 전에 같은 helper 안에서 크기(블록 수 합)와 항목 수를 먼저 측정한다** — 걷기가 한 번 더
   들지만 idle 이고 보고 뒤다. 상태 파일에는 trash 에 남은 항목들의 합 · 항목 수 · 지우는 중인지 · 측정한 시각을
   싣는다. 아직 측정하지 않은 항목은 「크기 모름」으로 센다 (수는 알고 크기는 모른다)

B) 크기를 미리 측정하지 않고 지우면서 센다 — 「지금까지 지운 양」만 보인다. 걷기가 한 번이다. 대신 「곧 지워질
   양」을 모른다 (US-2 의 앞 절반이 빈다)

C) 크기 대신 trash 의 항목 수와 filesystem 여유만 싣는다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6 — 여유 부족 drain 을 거는 노드

`min_free_gb` 의 기본값은 10 이다 — 적지 않은 노드도 10 GB 를 본다. arch 조건문이 사라지면 여유를 지키는
길은 drain 하나다.

A) **`min_free_gb` 를 가진 모든 노드가 건다** (기본 10 그대로). native 빌드 노드도 여유가 모자라면 빠진다 —
   arch 조건이 하던 보호를 이것이 넘겨받는다. agent 만 하는 노트북도 10 GB 아래면 빠진다 — US-3 의 「노드 전체가
   빠진다」가 그 뜻이다. 예시 설정의 주석과 단계 로그가 아닌 **노드 로그 한 줄**이 그것을 알린다

B) scratch 가 있는 runc-overlay 노드만 건다 — trash 가 쌓이는 곳이다. native 노드는 여유를 안 본다. native 빌드
   노드는 디스크가 차면 빌드가 실패하는 것으로 알게 된다

C) 모든 노드가 걸되 기본값을 끈다 — `min_free_gb` 를 적은 노드만 본다. 오늘 기본 10 을 믿던 노드가 조용히
   보호를 잃는다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 7 — drain 을 합치는 규칙 · 거는 값 · 푸는 조건

drain 의 세기는 at-boundary (경계에서 닫는다) > graceful (새 임대만 막는다) > 없음 이다.

A) **여유 부족은 graceful 로 건다. 소유자 drain 과 합칠 때 센 쪽을 싣는다** (소유자 at-boundary + 여유 부족 ->
   at-boundary). **거는 조건은 여유 < `min_free_gb`, 푸는 조건은 여유 >= `min_free_gb` + 1 GB** — 삭제자가 조금씩
   비우는 동안 광고마다 걸고 풀기를 되풀이하지 않게 한다. 여유를 측정하지 못하면 오늘처럼 넉넉한 것으로 치고 노드 로그에
   한 번 적는다. 출처를 합치는 틀은 출처 목록을 받게 짓는다 — lower-state 유닛이 bake 출처를 더한다

B) A 와 같되 푸는 조건도 여유 >= `min_free_gb` 다 (같은 값). 경계에서 광고마다 걸고 풀 수 있다

C) 여유 부족을 at-boundary 로 건다 — 도는 단계를 경계에서 닫는다. 디스크가 더 차기 전에 멈춘다. 대신 긴 빌드가
   중간에 끊긴다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 8 — 상태 파일을 쓰는 자리와 조건

오늘은 광고 주기에서 탐지 시각이 바뀔 때만 쓴다. 새 칸은 drain 의 출처(광고 주기에 정해진다)와 trash 의 양(삭제자가
바꾼다)이다.

A) **칸을 한 자리에 모으고, 어느 칸이든 바뀌면 그 자리가 파일을 쓴다** — 광고 주기와 삭제자의 사건(측정 · 지움 시작 ·
   지움 끝) 둘 다에서. 쓰는 곳은 하나라 두 고루틴이 서로의 칸을 덮지 않는다. 쓰기는 오늘처럼 임시 파일에 쓰고
   이름을 바꾼다. 못 써도 광고와 삭제를 안 막는다

B) 광고 주기에서만 쓴다. trash 의 양은 광고 주기(기본 60초)만큼 늦게 보인다

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 9 — 제어판이 보이는 것 (US-1 · 완료 조건 1 · 2)

A) **drain 칸은 상태 파일의 실린 값(effective)과 출처 목록을 보인다.** 출처마다 한 줄 — 무엇인지(소유자 · 여유 부족) ·
   값(graceful · at-boundary) · 설명(「여유 7 GB < 최소 10 GB」) · **소유자가 풀 수 있나.** 「drain 풀기」 버튼은
   소유자 출처가 있을 때만 보이고, 다른 출처가 남아 있으면 버튼 곁에 「풀어도 여유가 돌아올 때까지 빠져 있다」를
   적는다. trash 칸 — 남은 양 · 항목 수 · 지우는 중인지 · 측정한 시각. 데몬이 안 돌면(상태 파일이 없거나 낡았다) 오늘처럼
   정책 파일만 보인다

B) 출처 목록만 더하고 버튼은 오늘처럼 늘 보인다

C) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 10 — 기동 순서와 조각 4 의 확인 모양

`services.md` 4절은 「삭제자 첫 회 -> 광고 시작」 순서다. 남은 trash 가 크면 첫 회가 몇 분 걸리고 그동안 노드가
함대에 없다.

A) **삭제자 첫 회를 배경에서 시작하고 광고는 기다리지 않는다** — 여유가 모자라면 여유 부족 drain 이 새 일을 막는다.
   **조각 4 는 사람과 기계 둘로 확인한다.** 사람(SunnyVM) — 15만 파일 · 9 GB upper 를 남기는 단계와 작은 upper 를
   남기는 단계의 `finalized_at - exited_at` 이 비슷하다 · 보고 직후 trash 가 차 있고 곧 빈다 · 권한 000 인 `work/work` 와
   subordinate uid 항목도 지워진다 · 데몬을 재시작하면 남은 trash 를 비운다 · `min_free_gb` 를 여유보다 크게 두면
   `GET /v1/nodes` 에 draining 이고 여유가 돌아오면 풀리며, 그동안 `arch.<이름>` 키는 남는다. 기계(기본 `go test`) —
   옮기기가 rename 한 번(같은 inode) · 이름 거절 · symlink 를 안 따라감 · drain 합치기 표 · 걸고 푸는 조건 · 상태 파일을
   쓰는 조건 · arch 키가 여유와 무관함. namespace 안의 삭제는 `integration` 태그 시험 (CI 밖 · 사람이 SunnyVM 에서)

B) 기동 순서는 `services.md` 4절 그대로 — 첫 회가 끝난 뒤 광고한다. 조각 4 의 모양은 A 와 같다

C) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## 3. 산출물 계획 (체크박스)

답이 들어오고 모호함이 풀린 뒤에 채운다. 자리는
`aidlc-docs/v4-run-finalize-bake/construction/trash/functional-design/` 이다.

- [x] 답을 읽고 모호함을 확인한다 — 있으면 되물음 파일을 만든다 (2026-09-26T12:40:16Z · 답 열 모두 A · 답끼리 막는 자리 없음 · 되물음 없음)
- [x] `domain-entities.md` — `internal/scratch` 의 `Trash` · `Remove` · `Deleter` · `Usage` · 세션 잠금 ·
      trash-helper 의 입구와 인자 · `DrainSource` 와 drain 합치기의 입력과 출력 · 상태 파일의 새 칸 ·
      제어판 State 의 새 칸 · 설정에서 읽는 값 (`min_free_gb` · scratch)
- [x] `business-rules.md` — 다섯 자리의 새 동작 · 닫기와 Finalize 예산 · 남은 작업 폴더를 알아보는 규칙 ·
      삭제의 경계 · 삭제자의 깨는 때와 실패 · 양을 측정하는 법 · 여유 부족 drain 을 거는 노드 · 걸고 푸는 조건 · 합치기 표 ·
      arch 키 · 상태 파일을 쓰는 조건 · 제어판 문구 (영어 · 화면 문구는 오늘 화면의 언어를 따른다) · 로그 줄 (영어)
- [x] `business-logic-model.md` — 닫기의 흐름 · 기동의 흐름 · 삭제자 goroutine 과 helper · 광고 주기의 drain 합치기 ·
      상태 파일 쓰기 · 제어판이 읽는 흐름 · 조각 4 의 확인 모양 · 파일 행렬 밖 자리 · 다른 유닛에 넘기는 것 ·
      정본 되돌림
- [x] 코드 경계 시험에 더할 줄 셋 — Mediator 가 `internal/enode` 를 못 가져다 쓴다 · `internal/scratch` 를 못 가져다 쓴다 ·
      `internal/scratch` 는 표준 라이브러리와 x/sys 만 쓴다 (유닛 정의 4절)
- [x] 커버리지 — 규칙을 순수 함수로 떼는 자리를 적는다 (`internal/enode` 82.3% · 새 패키지 `internal/scratch` 는 80% 이상)
- [x] 표기 검사 (`enode-design/scripts/emphasis-check.py`) · 사용자가 싫어한 말투 검사 · 사내 이름 검사

---

## 4. 확장 준수 — 이 단계

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 trash 삭제 줄(namespace 안 · symlink 를 안 따라감 · trash 밖을 안 지움)은 팩의 요구로 남아 물음 4 가 닫는다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 저장소의 표 시험 관례를 따른다 |

유닛 정의는 이 유닛의 NFR Requirements 를 「한다」로 적었다 — N1 (삭제자가 한 번에 지우는 양과 속도) · 보안(trash
삭제 줄) · 성능(보고 전에는 이름 바꾸기 한 번만 — 조각 4). Functional Design 이 닫힌 뒤에 정한다.
