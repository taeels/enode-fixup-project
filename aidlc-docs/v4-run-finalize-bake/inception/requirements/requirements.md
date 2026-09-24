# Requirements — 굽기 (v4-run-finalize-bake)

```text
   입력        requirements/finalize-bake/ 다섯 파일 (팩)
               aidlc-docs/inception/reverse-engineering/ 여덟 문서 (2026-09-23 전면 갱신 · 기준 195a5d0)
               requirement-verification-questions.md 의 답 여섯 (2026-09-23T14:35:56Z)
               requirement-clarification-questions.md 의 답 하나 (2026-09-23T14:41:53Z)
   정본        enode-design/.  어긋나면 protocol/INVARIANTS.md 가 이긴다
   확장        security-baseline · resiliency-baseline · property-based-testing 모두 끔
   깊이        Comprehensive.  공유 lower 를 바꾸는 코드가 들어오고, 틀리면 그 기계의
               모든 형제 노드가 잘못된 바닥 위에서 돈다
```

---

# 1. 의도 분석

```text
   사용자 요청   「requirements/finalize-bake 로부터 brownfield aidlc 를 시작.」 (13:49:57Z)
                「workspace detection의 reverse engineering 산출물 승인. 요구사항 분석으로
                 넘어간다.」 (14:12:44Z)
                질문 파일의 답 (14:35:56Z) · 「그럼 먼저 main에 기준선을 병합하고 진행하도록
                하지.」 (14:41:53Z)
   요청 종류     Enhancement 와 New Feature 가 섞였다.
                단계 결과 좁히기 · phase · 예산 · trash 는 기존 실행 경로를 고치고,
                굽기 · 합치기 · lower 상태 · checkpoint 는 새 기능이다
   범위 추정     System-wide.  노드(enode) · Mediator(api · store) · 계약(contract) ·
                실행 환경(environment) · Record 가 다 움직이고 정본 ADR 다섯이 딸려 온다
   복잡도 추정   Complex.  공유 lower 를 바꾸는 합치기, 형제 노드 사이의 잠금과 drain,
                끊겨도 재개되는 절차, 사람이 보는 조각 여덟
```

## 1.1 왜 지금인가

사내 증상 하나에서 시작했다 — **결정론적 BitBake 는 끝났는데 임대가 안 풀린다.**
2026-09-21 ~ 23 설계 검토가 그 뿌리를 넷으로 갈랐고(`features.md` 1.2) 정본에 ADR
셋(075 · 076 · 077)이 섰다.

```text
   Worker 가 계약과 무관하게 workspace 전체를 걸어 결과를 추측한다       ADR-075
   명령 종료부터 결과 보고까지가 밖에서 CLAIMED 하나로만 보인다          ADR-075 §10
   끝난 단계의 upper 를 보고 전에 지운다 — 삭제가 항목 수에 비례한다      ADR-076 §4.1
   굽기를 받을 노드가 없다 — overlay 는 쓰기를 버리고 native 는 lower 를 바꾼다   ADR-077
```

## 1.2 가치의 고정점

`scene-gates.md` 1절의 장면 셋이다.

**명령이 끝나면 밖에서 곧바로 보이고, 결과 확정이 workspace 크기를 걷지 않고,
하루치 굽기가 형제를 멈추지 않은 채 lower 에 합쳐지며, 실패한 단계를 보고 뒤에
들여다볼 수 있다.**

그중 **임대 창이 upper 크기와 무관하다**가 이 회차의 값이다. 사내 증상이 그 창에서
났다.

---

# 2. 착수 전 실측 — 팩을 코드에 댔다

R/E 가 팩의 전제와 어긋나는 측정값 여섯을 냈고(`aidlc-state.md` ① ~ ⑥), 질문
단계에서 팩을 코드에 다시 대며 넷을 더 찾았다. 전문과 근거는
`requirement-verification-questions.md` 의 「확인된 사실」 1 ~ 9 이고, 여기는 그것이
요구로 어떻게 옮겨졌는지만 적는다.

| # | 사실 | 처리 | 요구 |
|---|---|---|---|
| ① | `min_free_gb` 가 arch 키만 뺀다 (`detect.go:82 · :387`) | 질문 1 = A | FR-4 |
| ② | result 가 인스턴스를 안 본다 (`store/claim.go:782`) | 종료 보고만 인스턴스까지 대조 | FR-2 · 5.3 |
| ③ | Mediator 가 `internal/environment` 를 링크한다 | 새 노드 쪽 코드는 Mediator 밖 | 5.2 |
| ④ | `internal/enode` 81.5% · namespace 경로는 CI 밖 | 규칙은 기본 `go test` 에서 돈다 | 5.1 · 5.6 |
| ⑤ | `runRoot` 삭제 자리가 다섯 | 다섯 다 trash | FR-4 |
| ⑥ | ADR-071 이 「미구현」인데 구현됐다 | 정본 되돌림에 더한다 | 11절 |
| 6 | 정본 계약 문서가 저자에게 `workspace.changed` 를 권한다 (`run-contract.md:392 · :413`) | bounded discovery 를 FR-1 로 당긴다 | FR-1 |
| 7 | 합치기 표에 종류가 바뀐 항목이 없다 — `rename(2)` 이 `EISDIR` | lower 쪽을 trash 로 옮긴 뒤 대체 | FR-7 |
| 8 | 임대가 Run 의 effect 를 모른다 (`advertise.go:53`) | 잠금 시점은 Functional Design | FR-8 · 8절 |
| 9 | scratch 를 다른 filesystem 에 둔 노드가 not ready 가 된다 | 결정 3-6 그대로. 사유를 이름으로 | FR-8 · 5.5 |

하나 더 — **`workspace.changed` 한 파일이 세 가지를 싣는다** (`claim.go:961`
`writeHarvestNote`).

```text
   계약이 요구했는데 $OUT 에 없는 이름      판정 재료가 아니라 사람이 읽을 사실
   collect 가 못 걷은 이름과 이유           run-contract.md:413 이 가리키는 자리
   기준 시각 뒤 바뀐 파일 목록              Discover 가 켜졌을 때만
```

걷기를 끄면 셋째가 사라진다. 그런데 **목록이 비면 오늘 코드는 「no files changed in
the workspace」라고 쓴다** — 안 잰 것을 「안 바뀌었다」로 말하게 된다. FR-1 이 이것을
막는다.

---

# 3. 사용자 결정 — 질문의 답

```text
   1   A    여유 하한 아래면 노드가 스스로 graceful drain 을 싣는다.
            덧붙인 말 — arch 를 디스크에 묶은 단순 조건문은 제거한다
   2   B    Inception 을 돌고 그대로 Construction 까지 한 손(taeels)으로
   3   A    실행 환경 브랜치를 먼저 자기 PR 로 main 에 올린다 (재질문의 답)
   4   B    security-baseline 끔 (권장을 벗어났다)
   5   B    resiliency-baseline 끔
   6   X    「기존 테스트 컨벤션을 따른다」 — property-based-testing 끔
```

**질문 1 의 덧붙인 말이 정본 하나를 고친다.** 그 조건문은 ADR-017 결정 3(확정
2026-08-19, 코드 `1249c35`)이다. 그때 광고되는 능력이 빌드 하나였고 그 이름이
arch 였다. 결정의 본체(동적인 여유를 매칭에 안 넣고 노드가 스스로 판단한다)는
그대로 서고, **빠지는 단위가 키 하나에서 노드 전체로** 바뀐다.

**질문 4 = B 는 팩의 보안 표를 끄지 않는다.** 확장이 끄는 것은 SECURITY-NN
기준선 규칙의 차단성 집행이다. 팩이 적은 보안 요구(`features.md` 3절)는 이 팩의
요구이고 5.3 이 그대로 진다.

**질문 3 = A 의 집행** — 2026-09-23 에 진행했다.

```text
   enode-design        taeels/enode-design#15          병합 a2c4ac6.  369270a 가 main 에서 닿고 나무가 같다
   이 저장소           taeels/enode-fixup-project#59   CI 셋(test · cross · bounded-demo) 초록 뒤 병합 826b40f
   회차 브랜치          main 을 합쳤다 (4facffb — 나무 변화 0).  핀을 a2c4ac6 으로 옮겼다 (afcdc71)
```

둘 다 병합 커밋이다. 그래서 `b5659ae` 를 되돌리지 않았고, 회차의 Inception PR 에는
이 회차의 문서와 핀 한 줄만 실린다.

---

# 4. 기능 요구사항

FR 번호는 팩의 기능 번호와 같다. 팩의 문장을 되풀이하지 않고, **팩에 더하거나
고친 것**을 굵게 적는다. 결정의 근거는 `decisions.md` 가 진다.

## FR-1 (기능 1) 단계 결과를 Finalize 로 닫는다

- build/test effect 의 단계에서 `RecordDiff` 와 전수 `Discover` 를 끈다.
  오늘 명령 단계는 둘 다 켠다 (`claim.go:691-694`)
- 계약이 지목한 경로만 `stat` 하는 `Result.Changed` 는 그대로다 (ADR-037)
- `$OUT` 의 named output 과 `collect` 는 그대로 결과다
- 계약 단계에 `effect` 필드를 둔다. 값은 ADR-075 §5 의 read/analyze · edit ·
  build/test · prepare 다. 명시가 없으면 명령 단계는 build/test, agent 단계는
  edit 다. prepare 는 굽기의 build 단계가 쓴다 (FR-5)
- **agent 단계도 전수 `Discover` 를 끈다** (Units Generation Q7 = A · 2026-09-24T09:50:00Z). `RecordDiff`
  는 FR-11 의 Git changeset adapter 가 설 때까지 그대로다 (`claim.go:876-882`).
  처음 판은 둘 다 그대로 두었고 근거가 「훅이 `workspace.changed` 를 읽는다」였다. 그 파일을
  읽는 코드는 0 이고 훅은 산출물이 빠졌을 때 스스로 걷는다 (`hook.go:281`). 걷기의 산물은
  결과도 checkpoint 도 아니다 (원장 `EN-43c3e7d8` 의 세 갈래)
- **명시적으로 켜는 bounded discovery 를 이 FR 에 둔다** (팩은 기능 11). 방문 수 ·
  시간 · 메모리 · 결과 크기에 상한이 있고, 상한에 닿으면 부분 관찰임을 적는다.
  정상 결과가 아니라 진단이다. 기능 1 이 끄는 걷기의 대체가 같은 때 서야 계약
  저자가 산출물 경로를 찾을 길이 안 끊긴다
- **변경 목록을 재지 않았으면 「바뀐 파일이 없다」고 쓰지 않는다.** 누락 산출물과
  collect 실패 이유는 오늘처럼 사람이 읽는 자리에 남는다. 그 자리의 이름과 모양은
  Application Design 이 정한다
- 결과 의미는 runtime 과 무관하다 — native 와 runc-overlay 에서 같은 actor 와
  effect 는 같은 commit set 을 낸다

## FR-2 (기능 2) 명령 종료 보고와 phase

- 명령의 종료 status 가 정해지면 노드가 Finalize 전에
  `POST /v1/runs/{run}/steps/{seq}/exited` 를 한 번 보낸다. outcome 과 종료 시각만
  싣는다. 키는 Run · seq · attempt · 노드 instance 다
- **Mediator 는 보낸 노드와 인스턴스를 `steps.node_id` · `steps.claimed_instance`
  에 대조한다.** 다르면 거절한다. `claimed_instance` 는 claim 이 이미 채운다
  (`store/claim.go:321`). result 의 대조는 바꾸지 않는다 — 비대칭은 잔여로 적는다
  (5.3)
- 재전송과 늦은 도착은 조용히 성공한다. 종결 전이는 result 하나이고 result 는
  종료 보고의 수락에 기대지 않는다
- 종료 보고는 판정이 아니다. 대조 · 다음 단계 생성 · 정산을 하지 않고 임대를 풀지
  않는다
- 단계 상태는 `CLAIMED` 그대로이고 phase 칸을 붙인다 — `running` · `finalizing` ·
  `waiting`. 종료 보고를 받으면 `finalizing` 과 `phase_since = exited_at` 이다.
  merge 단계는 claim 되는 순간 `waiting` 이다
- 진행 조회(`GET /v1/runs/{id}`)의 단계마다 `phase` · `phase_since` · `exit` 를 싣는다
- result 가 `exited_at` · `finalize` · `upload` 를 다시 싣는다. 종료 보고가 유실돼도
  봉인된 사실이 같다
- Record 의 단계 기록이 종료 시각과 Finalize 가 끝난 시각을 따로 가진다
- 종료 보고를 안 보내는 옛 노드의 단계는 `running` 에 머문다
- 라우트가 하나 는다 — `internal/api/api.go` 의 `mux.HandleFunc` **18 -> 19**,
  `internal/api` 전체 등록 **27 -> 28**

## FR-3 (기능 3) finalizing 구간의 예산 둘

- Finalize 예산 기본 1분, 계약이 단계마다 늘릴 수 있다. 넘으면 `finalize_timeout`
- 업로드 예산 기본 3분, 계약이 단계마다 바꿀 수 있다. 넘으면 `upload_timeout`
- 둘 다 명령 실패와 구분되는 원인으로 Record 에 남는다
- 노드가 스스로 지킨다. Mediator 는 강제하지 않는다 (순연 4-3)
- 계약 표기와 노드 쪽 상한을 둘지는 Application Design 이 닫는다 (`decisions.md` 6절)

## FR-4 (기능 4 · 질문 1 = A) scratch 는 trash 로, 삭제는 보고 뒤에

- `runRoot` 를 `RemoveAll` 하지 않고 `<scratch>/trash/` 로 `rename` 한다.
  **바꿀 자리는 다섯이다** — `runc_overlay_linux.go` 의 `Open` 실패 갈래(:143 ~ :170) ·
  세션 `Close`(:437) · `abort`(:451) · helper `cleanup`(:1017) · 준비도 smoke(:1212).
  capture 정책과 무관하다
- 노드 데몬의 배경 삭제자가 결과 보고 뒤에 지운다. runtime 과 같은 uid 매핑의 user
  namespace 안에서, idle IO 우선순위로 돈다. 데몬 시작 때 한 번 돌아 남은 trash 를
  비운다. smoke 가 데몬 밖(`enode env check`)에서 남긴 것도 그때 치운다
- **여유가 `min_free_gb` 아래면 노드가 광고의 `policy.drain` 에 `graceful` 을 스스로
  싣고, 되찾으면 푼다.** 새 광고 어휘도 Mediator 변경도 없다 — draining 노드는 이미
  후보에서 빠진다(ADR-063). 굽기의 pending · merging drain(FR-8)과 같은 길이다
- **arch 와 `arch.<이름>` 키는 툴체인 탐지만 따른다.** `hasRoom` 이 그 키를 가리는
  조건문(`detect.go:83` ~ `:98`)을 제거한다
- 소유자가 정책 파일로 건 drain 과 노드가 스스로 건 drain 이 겹치면 센 쪽을 따른다.
  어느 쪽이 센지와 광고 응답이 그것을 어떻게 보이는지는 Functional Design 이 닫는다
- 여유를 재는 자리는 워크스페이스 그대로다. FR-8 의 `env check` 가 scratch 와
  워크스페이스의 `st_dev` 일치를 강제하므로 같은 filesystem 이다

## FR-5 (기능 5) 굽기 계약

- 굽기는 일반 Run 이고 두 단계다 — build(`effect: prepare` · `sync` · `builds[]`)와
  merge(내장 kind · `needs: build`). 같은 역할을 써서 같은 노드에 앉는다
- merge 는 명령이 아니라 enode 가 수행하는 내장 단계다. ask(ADR-032)가 선례이나
  ask 는 Mediator 쪽이고 **merge 는 노드 쪽**이다
- `builds[]` 의 항목은 `name` 과 `command` 다. 이름은 굽기 계약을 쓰는 쪽이 짓는다
- 계약 검증이 400 으로 막는 것 — prepare 단계 뒤에 같은 역할의 merge 단계가 없음,
  구성 이름이 문자 규칙(소문자 · 숫자 · `-`)을 어김, 이름이 겹침, **구울 IR 을 안 적음**
  (FR-9 · Units Generation 이 더했다)
- merge 대기 상한 기본 4시간. 계약의 merge 단계가 바꿀 수 있다
- 계약이 노드 설정이나 host 경로를 지정하지 못한다

## FR-6 (기능 6) build 단계는 upper 에 짓는다

- 평범한 overlay 단계와 똑같이 돈다. lower 는 읽기 전용이고 rootfs 와
  `workspace_target` 이 작업 단계와 같다
- 노드가 `sync` 를 먼저, `builds` 를 적힌 순서대로 돌리며 항목마다 시작 · 끝 시각과
  exit code 를 기록한다. 이것이 prepare effect 의 결과 manifest 다
- 하나라도 0 이 아니면 build 단계가 실패하고 합치지 않는다. upper 는 trash 로 가고
  실패한 시도는 `state.json` 의 `last_attempt` 와 Run Record 에 남는다
- 성공하면 Finalize 가 upper 만 합치기 대기 자리로 `rename` 하고 나머지 `runRoot` 는
  trash 로 보낸다
- `repo manifest -r` 로 뜬 pinned manifest 를 남긴다

## FR-7 (기능 7) 합치기

- upper 를 lower 에 파일 단위 `rename` 으로 합친다. 규칙은 ADR-077 §4 의 표다
- **종류가 바뀐 항목은 lower 쪽을 trash 로 옮긴 뒤 대체한다.** lower 의 디렉터리
  자리에 upper 의 파일 · symlink 가 오면 `rename(2)` 이 `EISDIR` 로 실패한다. 표에 이
  줄이 없다. SunnyVM 의 `merge.py` 가 이것을 어떻게 다뤘는지 Functional Design 이
  확인한다
- 표시(whiteout · opaque)는 그 효과를 lower 에 적용한 뒤에만 치운다. 어느 지점에서
  끊겨도 남은 upper 에 같은 절차를 다시 돌리면 한 번에 끝낸 결과와 같다
- 시작 전 확인 넷 — 같은 filesystem, `metacopy` 꺼짐, redirect 표시 없음, 이 lower 의
  overlay 마운트 0. 하나라도 어긋나면 시작하지 않는다
- 합치는 경로가 lower 밖으로 나가지 않는다
- 마지막 동작으로 `.enode-metadata.json` 을 쓴다 (FR-9)
- Go 로 옮기되 `merge.py` 의 규칙과 시험을 그대로 테스트로 삼는다 (5.6)

## FR-8 (기능 8) lower 의 상태 · 잠금 · drain · 재개

- 상태 자리는 `~/.local/state/enode/lowers/<fsid>-<ino>/` 이고 `lower.json` ·
  `state.json` · `lower.lock` · `bake.lock` 을 둔다. 키는 lower 루트의 `statfs`
  `f_fsid` 와 inode 다. 노드 사용자 전용 권한이다
- 전이 상태는 committed · building · pending · merging 이다. lower 안에 쓰지 않는다
- 형제 overlay 노드는 임대를 쥔 동안 lower 공유 잠금을 쥔다(Run 단위). prepare 를 담은
  Run 은 잡지 않는다. merge 는 배타 잠금을 잡는다. 잠금 승격은 쓰지 않는다
- **요구 — 매칭된 뒤 첫 단계 전에 lower 가 바뀌지 않는다.** 임대는 Run 의 effect 를
  싣지 않으므로(`advertise.go:53`) 공유 잠금을 언제 잡고 언제 놓는지는 Functional
  Design 이 닫는다 (8절 ③)
- 상태가 pending 이나 merging 이면 형제가 광고에 graceful drain 을 싣는다
- merge 대기가 상한을 넘으면 `merge_wait_timeout` 으로 upper 를 trash 로 보내고
  상태를 committed 로 돌리고 drain 을 푼다. 합치기 본체에는 상한이 없다
- 상태가 committed 가 아니고 굽기 잠금의 주인이 살아 있으면 새 굽기는
  `bake_in_progress` 로 곧바로 실패한다. 주인이 없으면 낡은 상태를 정리한다
- 같은 lower 의 어느 노드든 시작할 때 merging 을 재개한다. 재개로 끝낸 합치기는
  metadata 에 `resumed` 와 원래 Run 을 남긴다
- `env check` 가 셋을 확인한다 — scratch 와 워크스페이스의 `st_dev` 일치, lower 루트
  소유 uid 와 노드 uid 의 일치, `lower.json` 신원. **not ready 의 사유가 어긋난 것을
  이름으로 말한다** (사실 9 — scratch 를 다른 filesystem 에 둔 기존 노드가 여기 걸린다)

## FR-9 (기능 9) metadata 와 광고

- `.enode-metadata.json` 은 `source`(url · branch · repo_id · head · ir · pinned ·
  sync_command · synced_at) · `builds[]`(name · command · started_at · finished_at ·
  exit_code) · `environment` · `workspace_target` · `bake`(run · node · merged_at ·
  resumed · previous_ir)를 담는다
- **굽기 계약의 build 단계가 구울 IR 태그를 정확한 값으로 적는다 (필수).** 노드는 그
  값을 sync 와 builds 명령에 환경 변수로 넘기고, sync 뒤 `.repo/manifests` HEAD 에 그
  태그가 정확히 붙었는지 확인한다. 다르면 build 단계가 실패하고 합치지 않는다. 제품에는
  IR 태그의 기본 형식이 없다 — 사내 규약이라 제품이 모른다. Units Generation(2026-09-24)
  이 고쳤다. 처음 판은 「IR 은 계약 칸이 아니라 sync 뒤 HEAD 태그에서 유도하고, 없으면
  null」이었다
- 광고 키 — `workspace.writes`(runc-overlay 면 `isolated`, native 면 `in-place`, 모든
  노드가 낸다) · `ir` · `repo.built.<name>=yes`. 평평한 문자열 키다

## FR-10 (기능 10) checkpoint capture 와 Checkpoint Store

- `StepRuntime` 이 capture capability(지원 여부 · 범위 `workspace-upper` · 보장
  `inspect-only`)를 낸다. native 는 `unsupported`(`runtime`)다
- 정책이 요구하면 보고 전에는 `statfs` 여유 하한과 store 의 현재 보존 총량만 보고
  upper 를 spool 로 `rename` 한다. 크기 판정은 보고 뒤 store 가 upper 순회로 하고,
  넘으면 trash 로 퇴출한다
- receipt 의 `checkpoint_capture` 에 상태 다섯(`not_requested` · `unsupported` ·
  `rejected` · `captured` · `failed`)과 `reason` 코드(`runtime` · `cross_filesystem` ·
  `quota` · `free_space` · `lease_budget` · `io`), diagnostics 에 상세를 둔다. receipt 는
  의미상 자리다 — wire 이름은 ADR-075 §4 대로 오늘 필드를 바꾸지 않고 더한다
- 기본값 — 정책 on-failure · TTL 48시간 · 노드 보존 용량 (보존 총량 + 여유)의 20% ·
  퇴출은 오래된 것부터. 노드 소유자가 바꿀 수 있고 원격 계약과 agent 출력은 못 바꾼다
- spool 은 소유자만 읽는다. 외부 descriptor 에 host 경로와 비밀을 싣지 않는다. 보존 중
  암호화는 요구하지 않는다
- 재시작 때 미완료 capture 와 만료 항목을 조정한다
- 단계 outcome 과 commit set 은 capture 성패와 무관하다

## FR-11 (기능 11) 결과 adapter — 순연

**순연 (Units Generation Q3 = A · 2026-09-24T09:06:19Z).** 사내는 Git changeset 을 쓸 일이 없고
빌드는 커스텀 스크립트가 시작한다. 조각 10 은 해당 없음이다. 아래는 팩의 문장이고 되살릴 때의
출발점이다. 설계 빚은 6.1 에 있다.

- agent edit 를 위한 Git changeset adapter. 정확한 base identity 를 가진다. **이것이
  서면 agent 단계의 `RecordDiff` 와 전수 `Discover` 를 끈다** (FR-1)
- Yocto producer adapter. 같은 namespace 안에서 target 의 deploy output 을 질의해
  named output 으로 승격한다
- bounded discovery 는 FR-1 로 옮겼다

## FR-12 (기능 12) ADR-073 잔여

- 실제 최대 PyInstaller extraction 이 256 MiB tmpfs 에 드는지 확인한다
- 운영 대상 host 의 subordinate-ID 배치를 확인한다
- ADR-073 §11 끝 「남은 것」의 E5 문장을 SunnyVM 제품 경로 완주(`EN-bf4045e7`)로
  갱신한다

## FR-13 (기능 13) ADR-072 의 결정 조건

- rev 를 뽑은 판으로 실물 Run 을 돌려 agent 와 build 가 같은 IR 위에서 diff 를
  주고받는 것을 본다. 통과하면 ADR-072 를 결정으로 올릴 근거가 선다

---

# 5. 비기능 요구사항

## 5.1 차단 게이트 — 그대로 걸린다

```text
   패키지별 커버리지                      하한 80%   .coverage-contract.yml 의 측정 명령
   허용목록 밖의 스킵                     상한 0
   U+2605 을 담은 파일 수                 상한 0
   출력 문자열의 장식 문자                 별도 스텝
   crypto/tls T 심볼 (enodectl.exe)      상한 10
   net/http  T 심볼 (enodectl.exe)       상한 50
```

**`internal/enode` 의 여유가 1.5 포인트다** (81.5%). 이 회차가 그 패키지에 가장 많은
문장을 더한다. namespace 없이 도는 규칙 — 합치기 순회 · lower 상태 전이 · trash 삭제
경계 · capture 판정 · drain 계기 — 은 기본 `go test` 에서 도는 자리에 둔다.
integration 태그 뒤에 두면 CI 밖이라 하한에 안 든다.

## 5.2 임포트 경계

```text
   오늘      internal/store -> internal/environment      execenv.Record 하나 때문에
             Mediator 바이너리가 profile · env check · env apply 를 링크한다
   요구      lower 상태 · 잠금 · 합치기 · trash 삭제자 · spool 코드는
             Mediator 가 링크하는 패키지에 두지 않는다
   집행      자리는 Application Design 이 정한다.  새 패키지면
             internal/panel/boundary_test.go 의 금지 표에 줄을 더한다
   안 하는 것  store -> environment 한 줄을 끊는 것.  이 팩 밖이다
```

## 5.3 보안 — 팩의 보안 표

security-baseline 확장은 꺼져 있다(질문 4 = B). 아래는 **팩의 요구**이고 확장과
무관하게 선다.

| 자리 | 요구 |
|---|---|
| 종료 보고 | 그 단계를 claim 한 노드와 인스턴스만 보낼 수 있다. `steps.claimed_instance` 로 대조한다. 인증은 다른 노드 표면과 같은 bearer 다 |
| trash 삭제 | user namespace 안에서 지운다. symlink 를 따라가지 않는다. trash 밖을 지우지 않는다 |
| 합치기 | lower · 대기 upper · trash 가 같은 filesystem 이어야 시작한다. 합치는 경로가 lower 밖으로 나가지 않는다 |
| 상태 자리 | `~/.local/state/enode/lowers/…` 는 노드 사용자 전용 권한이다 |
| spool | 소유자만 읽는다. TTL 에 지운다. 외부 descriptor 에 host 경로와 비밀을 싣지 않는다 |
| 계약의 명령 | `sync` 와 `builds[].command` 는 격리 runtime 안에서 돈다. 계약이 노드 설정이나 host 경로를 지정하지 못한다 |
| 정책 | 원격 계약과 agent 출력은 checkpoint 의 TTL 을 늘리거나 host 경로를 지정하거나 보존을 강제하지 못한다 (결정 2-11) |

**잔여** — result(`POST .../result`)는 오늘처럼 노드만 대조한다. 재시작한 노드의 옛
인스턴스가 보낸 result 는 ADR-030 이 Run 을 이미 실패로 돌려 `state='CLAIMED'`
조건에서 걸린다. 두 표면의 대조 강도가 다른 것을 이 회차는 좁히지 않는다.

## 5.4 성능 — 임대 창이 크기와 무관하다

```text
   Finalize        workspace 크기를 걷지 않는다.  비용은 commit set 과 지목 경로 수를 따른다
                   조각 1 — 합성 300만 파일 no-op 단계가 빈 workspace 와 같은 수준
   보고 전 창       trash 와 spool 은 rename 한 번이다.  삭제와 크기 측정은 보고 뒤다
                   조각 4 — 15만 파일 · 9 GB upper 에서 종료부터 보고까지가 크기와 무관
   합치기           비용은 upper 에서 바뀐 항목 수를 따르고 byte 수와 무관하다.
                   새 디렉터리는 하위 트리가 커도 rename 한 번이다
   배경 삭제자      idle IO.  한 번에 지우는 양은 Application Design 이 정한다
```

## 5.5 호환과 불변식

```text
   I1                    안 고친다.  새 점유 단위를 Mediator 에 안 만든다.
                         lower 조율은 기계 안의 flock 과 광고 drain 이다
   단계 상태 어휘         안 바꾼다.  새 구간은 phase 칸이다.  진행 중 판정 그대로
   매칭 규칙              안 바꾼다.  새 광고 키는 평평한 문자열 키다
   Record                append-only.  재개의 흔적은 metadata 가 진다
   마운트된 lower         고치지 않는다.  metadata 는 마운트 0 인 합치기 창에서만 쓴다
   옛 노드               종료 보고를 안 보내면 그 단계는 running 에 머문다
   arch 광고 (바뀜)       여유와 무관하게 툴체인만 따른다.  여유가 모자란 노드는
                         빌드 능력이 아니라 노드 전체가 drain 으로 빠진다
   runc-overlay 준비도 (바뀜)  scratch 와 워크스페이스가 다른 filesystem 이면 not ready
   시험 안전             합치기 시험은 버려도 되는 lower 에서.  SunnyVM 의 /srv/yocto 는 읽기만
```

## 5.6 테스트 — 저장소의 규약을 따른다 (질문 6 = X)

```text
   도구          표준 testing.  단언 라이브러리와 모킹 도구를 안 들인다.  새 의존 0
   측정          .coverage-contract.yml 이 못 박은 명령 · env · flags · platform
   합치기        merge.py 의 시험을 Go 로 옮긴다 — 표시 종류를 모두 담은 가짜 트리에서
                 1 ~ 30번째 연산 뒤 강제 종료하고 다시 돌려 한 번에 끝낸 목록과 비교한다.
                 가짜 트리가 종류가 바뀐 항목을 담는다 (FR-7).  기본 go test 에서 돈다
   namespace     실제 user namespace 와 overlay 가 필요한 경로는 integration 태그 시험이다.
                 CI 밖이고 사람이 SunnyVM 에서 돈다.  그 결과로 사람 조각을 대신하지 않는다
   가짜 표시      whiteout 과 opaque 를 특권 없이 만들 수 있는지는 Functional Design 이
                 실측으로 확인한다.  못 만들면 그 시험의 자리를 다시 정한다
```

## 5.7 언어와 표기

`CONVENTIONS.md` 1 · 2 절 그대로다. 에러 문자열 · 로그 · CLI 출력 · 테스트 이름과
메시지는 영어, 주석은 한국어. 코드와 커밋 메시지에 장식 문자를 넣지 않는다. 새
reason 코드(`finalize_timeout` · `upload_timeout` · `merge_wait_timeout` ·
`bake_in_progress`)는 wire 값이라 영어 소문자다.

---

# 6. 수용 기준 — 장면 조각 게이트

`scene-gates.md` 의 조각 0 ~ 12 다. 아래는 **팩에서 바뀌거나 잴 것이 는 자리**만
적는다. 나머지는 팩의 문구 그대로다.

| 조각 | 이름 | 바뀐 것 |
|---|---|---|
| 0 | 기동이 안 깨졌다 | 바닥이 `main` 이 된다(질문 3 = A). `grep -c 'mux.HandleFunc' internal/api/api.go` 가 **18 -> 19** 다 |
| 1 | 걷지 않는다 | **잴 것이 는다.** bounded discovery 를 켠 단계가 상한에서 부분 관찰임을 적는다. 걷기를 안 켠 단계의 기록에 「바뀐 파일이 없다」가 없다 |
| 6 | 굽기가 돈다 | **바뀐다** (Units Generation). 「manifest HEAD 에 IR 태그가 없으면 `ir` 은 null 이고 광고하지 않는다」 대신 — sync 가 계약의 IR 에 닿지 않으면 build 단계가 실패하고 합치지 않으며, 결과에 HEAD 의 태그와 커밋이 보인다 |
| 2 | 보인다 | **잴 것이 는다.** 다른 인스턴스(같은 노드를 재시작한 뒤)가 보낸 종료 보고가 거절된다 |
| 4 | trash | **재는 방법이 바뀐다.** 여유가 `min_free_gb` 아래면 그 노드가 `draining` 으로 후보에서 빠지고(`GET /v1/nodes`), trash 가 비면 풀린다. 그동안 `arch.<이름>` 키는 광고에 남는다. 다섯 자리 모두 trash 로 간다 |
| 7 | 끊겨도 된다 | **잴 것이 는다.** 가짜 트리에 종류가 바뀐 항목이 있고, 그 시험이 기본 `go test` 에서 돈다 |
| 8 | 배타와 대기 | **잴 것이 는다.** `env check` 의 not ready 사유가 어긋난 것(`st_dev` · 소유 uid · `lower.json`)을 이름으로 말한다. 형제 노드가 임대를 받은 뒤 첫 단계 전에 합치기가 lower 를 바꾸지 않는다 |
| 11 | ADR-073 잔여 | 대상이 사내 운영 host 다. 못 닿으면 보류이고 보류는 통과가 아니다 |

**사람이 보는 조각(2 · 4 · 6 · 8 · 9 · 10 · 11 · 12)을 코드 테스트 초록으로 대신하지
않는다.** 집행자는 그 유닛을 구현하지 않은 사람이다 — 이 회차에서 구현은 에이전트가
하고 사람 조각은 사용자가 SunnyVM 과 사내 host 에서 돈다. SunnyVM 은 노트북 VM 이라
꺼져 있을 수 있고, 그때 그 조각은 보류다.

## 6.1 장면 4 — agent 가 고친 것을 build 가 받는다 (Units Generation 에서 더함 · 순연)

**순연 (2026-09-24T09:06:19Z).** Units Generation 질문 3 = A 로 결과 adapter 둘을 순연했고
이 장면도 함께 간다. 사내는 Git changeset 을 쓸 일이 없고 빌드는 커스텀 스크립트가 시작한다
(`unit-of-work-plan.md` Q3 의 사내 대조). 이 절은 되살릴 때 잴 장면과 설계 빚으로 남긴다.
아래 「재는 조각」의 두 줄은 적용하지 않는다 — 조각 10 은 해당 없음이고 조각 12 는 조각 6 만
딛는다.

**2026-09-24T07:48:31Z 사용자 지시로 더했다.** Units Generation 질문 3 이 FR-11 을 자를
근거로 「장면 셋 어디도 adapter 를 안 지난다」를 들었고, 사용자가 그 자리에 장면을 만들라고
했다. 팩의 `scene-gates.md` 1절에 옮기는 것은 팩을 고치는 쪽의 몫이다.

정본은 ADR-017 결정 6 · ADR-072 §9 · ADR-075 §6 · §7 · §15 다. ADR-017 결정 6 이 이미
그린 길(agent 노드가 고친 diff 를 같은 base 의 build 노드가 받아 짓는다)을, 이 팩이 바꾼
결과 경계 위에서 끝까지 돈다.

1. IR X 에 선 overlay 노드에 agent 단계(effect edit)를 낸다. agent 가 커널 소스의 파일을
   추가 · 수정 · 삭제하고, 고친 것을 확인하려고 compiler 도 돌린다.
2. 하네스가 끝나는 즉시 진행 조회에 `finalizing` 이 보인다. 결과 확정이 수백만 파일 트리를
   걷지 않는다.
3. 결과는 base 가 붙은 changeset 하나다. compiler 부산물이 없다.
4. 다음 build 단계가 IR X 에 선 노드에 앉아 그 changeset 을 받아 적용하고 BitBake 로 짓는다.
5. 계약이 경로를 적지 않아도 deploy output 만 commit set 에 봉인된다. intermediate 는 없다.
6. base 가 다른 노드에서는 그 changeset 이 조용히 붙지 않는다.

**재는 조각.**

```text
   조각 10    1 · 3 · 5 를 따로 잰다 (오늘의 확인 셋 그대로).  2 를 더한다 — 300만 파일
              워크스페이스에서 agent 단계의 결과 확정이 빈 워크스페이스와 같은 수준이다
   조각 12    1 ~ 6 을 한 Run 으로 끝까지 잰다.  먼저 서는 조각이 6 에서 6 · 10 으로 는다
```

**이 장면이 드러낸 빈자리 셋.** 팩과 앞 단계가 안 닫았다.

```text
   ①  받는 쪽      FR-11 은 changeset 을 만드는 쪽만 적는다.  받아 적용하는 쪽이 없다.
                  계약의 in.diff 는 자리만 있고 런타임이 안 읽는다 (contract.go:905 ·
                  run-contract.md:125 의 @work.patch_rev).  계약 명령의 git apply 로
                  할지 노드가 적용할지 정해야 한다.  6 이 거기 달렸다
   ②  base 의 뜻    Application Design 은 base 를 「git HEAD」로 적었다.  repo 로 여러 git
                  저장소를 묶은 Yocto 트리에서 어느 저장소의 HEAD 인지, IR 과 어떻게
                  맞물리는지가 안 정해졌다 (ADR-072 는 정합 단위를 IR 로 올렸다)
   ③  인터페이스    adapter 가 끼울 자리가 코드에도 설계에도 없다.  설계는 둘을 FinalizeSpec
                  의 칸(Changeset bool · Produce *ProduceSpec)으로 박았고 ProduceSpec 과
                  등록표의 모양은 적지 않았다.  ADR-075 는 둘 다 구현이 느는 자리로 적는다 —
                  Git 밖의 filesystem changeset adapter (§6), Bazel · CMake · Meson (§7)
```

---

# 7. 범위

## 7.1 하는 것 — 열셋

```text
   FR-1    수확 좁히기 · effect · bounded discovery      contract · enode
   FR-2    종료 보고 · phase                             api · store · enode · record
   FR-3    예산 둘                                      contract · enode · record
   FR-4    trash · 배경 삭제자 · 여유 drain · arch 분리    enode
   FR-5    굽기 계약                                    contract
   FR-6    build 단계                                   enode
   FR-7    합치기                                       enode 또는 새 패키지 (5.2)
   FR-8    lower 상태 · 잠금 · drain · 재개 · env check   enode · environment
   FR-9    metadata · 광고 키                            enode
   FR-10   checkpoint capture · store                   enode
   FR-11   Git changeset · Yocto producer adapter       enode
   FR-12   ADR-073 잔여                                  사람 확인 · 정본 문서
   FR-13   ADR-072 결정 조건                              사람 확인 · 정본 문서
```

## 7.2 안 하는 것

`constraints.md` 1절의 열둘 그대로다. 요지는 이렇다.

```text
   Windows StepRuntime                      Linux 쪽이 먼저 닫혀야 한다
   중앙 checkpoint store · 다른 노드로 복원    대용량 직접 전송과 portable format 이 없다
   checkpoint 의 restorable 보장            첫 보장은 inspect-only 다
   Mediator 가 Finalize 예산을 강제          노드가 스스로 지킨다
   첫 준비를 Run 으로                        빈 lower 노드를 고르는 법이 없다
   한 lower 를 여러 사용자가 나눈다            상태 자리가 사용자 home 아래다
   다른 filesystem 사이의 합치기              사용자 마운트포인트 하나에 둔다
   upper 를 층으로 쌓기                      날마다 합친다
   project quota 로 크기 세기                순회가 싸다
   노드 로컬 spool 의 보존 중 암호화           같은 내용이 lower 와 scratch 에 평문으로 있다
   저장소 소유의 구성 정의 파일               이름과 명령은 굽기 계약이 싣는다
   합치기 대기 중 Run 을 경계에서 멈췄다 잇기    graceful 로 기다린다
```

이 회차가 더한 것 둘.

```text
   result 의 인스턴스 대조                   종료 보고만 한다.  잔여 (5.3)
   store -> environment 임포트 끊기          새 코드만 Mediator 밖에 둔다 (5.2)
```

---

# 8. Application Design 이 닫을 것

**사람에게 물을 것이 아니다.** 답이 설계와 실측에서 나온다.

```text
   ①  effect 의 계약 표기와, 명령 단계가 effect: edit 일 때 changeset 을 무엇으로
      만드는가 (Git adapter 전에는 오늘의 git diff 가 후보다)
   ②  예산 둘과 merge 대기 상한의 계약 표기, 노드 쪽 상한을 둘지 (decisions.md 6절)
   ③  lower 공유 잠금을 언제 잡고 언제 놓나 — 임대가 닿는 순간 잡고 prepare 단계를
      claim 하면 놓는 길, 또는 임대가 effect 를 싣는 길 (사실 8)
   ④  노드가 스스로 건 drain 과 소유자 drain 의 세기 순서, 광고 응답에 보이는 모양
   ⑤  새 노드 쪽 코드의 패키지 자리 (5.2)
   ⑥  「바뀐 파일이 없다」를 안 쓰게 된 뒤 누락 산출물 · collect 실패가 남는 자리의
      이름과 모양 (FR-1)
   ⑦  phase 칸의 스키마 — steps 에 칸을 더하는가, 진행 조회가 무엇을 읽는가
   ⑧  checkpoint descriptor 와 조회 API, reason 의 wire 표기 (decisions.md 6절)
   ⑨  checkpoint 하나의 상한과 inode 한도 · 배경 삭제자가 한 번에 지우는 양
   ⑩  manifest HEAD 에 태그가 여럿일 때 IR 형식 규칙의 자리 (사내 형식 IR<YYMMDD>_<HHMMSS>)
       — Units Generation 이 없앴다.  계약이 IR 을 정확한 값으로 적는다 (FR-9)
   ⑪  Git changeset 의 wire format 과 10 MiB 초과 정책, producer adapter 의 등록과 versioning
   ⑫  가짜 트리의 whiteout 과 opaque 를 특권 없이 만들 수 있는지 (5.6)
   ⑬  유닛 분해와 파일 행렬.  Units Generation 의 몫이다
```

---

# 9. 회차 운영 — 질문 2 = B · 질문 3 = A

```text
   Inception      aidlc-docs/v4-run-finalize-bake/.  브랜치 v4-run-finalize-bake
   Construction   aidlc-docs/taeels/.  유닛마다 unit/<유닛> 브랜치를 회차 브랜치에서 딴다
   손             하나 (taeels).  앞 두 회차와 같은 모양이다
   병합           둘 다 PR 로 main.  유닛은 그 유닛의 장면 게이트가 초록인 뒤에
   바닥           실행 환경 구현이 main 에 먼저 갔다 (#59 · 826b40f).  회차 브랜치가 main 을
                  합쳤으므로 Inception PR 에는 이 회차의 문서만 실린다
   User Stories   돈다.  굽기 계약을 쓰는 사람 · 노드를 가진 사람 · 진행을 보는 사람이
                  갈리고, 계약 문법과 진행 조회가 바뀐다 — core-workflow.md 의
                  「ALWAYS Execute」 지표(새 사용자 기능 · 사용자 워크플로 변경 · 여러
                  페르소나)에 걸린다
   범위를 자를 자리  Units Generation 승인 (질문 2 의 B 가 그 자리를 남긴다)
```

---

# 10. `decisions.md` 에 더할 행

이 회차가 실측과 답으로 정한 것이다. 번호는 팩의 절을 따라 붙인다.

```text
   1-14   agent 단계의 Discover 는 끈다.  RecordDiff 는 Git changeset adapter 까지   Q7 = A
          그대로 (처음 판 「둘 다 그대로」의 근거가 거짓이었다)
   1-15   bounded discovery 를 기능 1 과 함께 세운다.  5절의 순서를 벗어난다       사실 6
   1-16   재지 않은 변경 목록을 「바뀐 파일이 없다」로 쓰지 않는다                  2절
   1-17   종료 보고는 노드와 claimed_instance 를 대조한다.  result 는 그대로       사실 1
   2-12   runRoot 삭제 자리는 다섯이고 다섯 다 trash 로 간다                     사실 4
   2-13   여유 하한 아래면 노드가 스스로 graceful drain 한다                      질문 1 = A
   2-14   arch 키는 툴체인만 따른다.  디스크 조건문을 제거한다                     질문 1 덧붙인 말
   3-23   종류가 바뀐 항목은 lower 쪽을 trash 로 옮긴 뒤 대체한다                  사실 7
   3-24   매칭된 뒤 첫 단계 전에 lower 가 바뀌지 않는다.  잠금 시점은 설계가 닫는다   사실 8
   3-25   env check 의 not ready 사유가 어긋난 것을 이름으로 말한다               사실 9
   5-·    확장 셋 모두 끔.  팩의 보안 표는 요구로 남는다                          질문 4 ~ 6
   5-·    실행 환경 구현을 먼저 main 에 병합한다                                 질문 3 = A
   3-26   굽기 계약이 구울 IR 을 정확한 값으로 적는다.  노드가 환경 변수로 넘기고   Units Gen.
          sync 뒤 HEAD 와 대조한다.  제품에 태그 형식의 기본값이 없다
   1-14'  agent 단계의 Discover 는 끈다.  RecordDiff 는 adapter 까지 그대로        Units Gen. Q7
   4-16   결과 adapter 둘(Git changeset · Yocto producer)과 장면 4 를 순연한다     Q3 = A
          되살릴 조건 — agent 가 고친 것을 build 로 넘기는 계약이 생길 때, 또는
          산출물 경로를 빌드 시스템에 물어야 하는 계약이 생길 때
```

---

# 11. 정본에 되돌려 올리는 것

`decisions.md` 7절의 목록에 여섯이 는다. 앞의 다섯은 팩이 이미 적었다.

```text
   팩이 적은 것
     run-contract.md    effect · 단계 예산 · sync · builds[] · merge kind · merge 대기 상한
     ADR-073 §11        E5 상태
     ADR-072            결정 승격과 §8 닫기 (조각 12 초록 뒤)
     ADR-076 · 077      결정 승격 (이 팩의 게이트 초록 뒤)
     enode-design main  같은 이름 브랜치를 main 으로 — 2026-09-23 #15 로 끝났다

   이 회차가 더한 것
     ADR-017 결정 3     여유 부족의 기제가 「능력을 뺀다」에서 「노드가 drain 한다」로.
                        매칭 조건이 아니라는 본체는 그대로 (질문 1)
     ADR-068 §3.3       「ADR-017③ 디스크가 모자라면 그 능력을 광고에서 뺀다 — 그대로」 줄
     ADR-076 §4.1       「min_free_gb 는 이미 있는 광고 조건이라 새 기제가 없다」
     ADR-077 §4         합치기 표에 종류가 바뀐 항목 한 줄 (사실 7)
     run-contract.md    :392 · :413 — workspace.changed 를 권하는 두 문장을 bounded
                        discovery 와 새 진단 자리로 (사실 6)
     ADR-071            머리의 「미구현」을 구현(70d6258)으로 (사실 5)
     mediator-api       exited 가 인스턴스까지 대조한다는 것과 result 와의 비대칭
     ADR-077 §5 · §11   IR 을 계약 칸으로 싣는다 — 노드가 환경 변수로 sync 에 넘겨 사실이
                        한 곳에서 온다.  형식 규칙의 자리는 없어진다 (결정 3-26)
     ADR-077 · run-contract   굽기 Run 의 성공 판정(success_when) — 계약 문법 유닛이 닫는다
```
