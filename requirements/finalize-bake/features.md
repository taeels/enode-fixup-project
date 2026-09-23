# finalize-bake 팩 — 단계 결과를 좁히고, 굽기를 같은 언어로 돌린다

2026-09-21~23 설계 검토의 후속이다. 사내 증상 「결정론적 BitBake 는 끝났는데 임대가
안 풀린다」에서 출발해 정본에 ADR 셋이 섰다. 이 팩은 **그 셋과 딸린 잔여를 제품 코드로
옮긴다.**

| 파일 | 역할 |
|---|---|
| `features.md` | 이 문서. 요구와 수용 기준의 주소 |
| `decisions.md` | 이미 정해진 값. 결정마다 ADR 절과 원장 항목을 단다 |
| `constraints.md` | 안 만드는 것, 구조 불변식, 접점, 코드 기준선 |
| `scene-gates.md` | 장면 조각 게이트. 이 문서의 수용 기준이다 |
| `canon.md` | enode-design 정본과의 연결. 어긋나면 INVARIANTS가 이긴다 |

---

# 1. 개요

## 1.1 무엇을 만드나

세 갈래다.

- **단계 결과를 좁힌다.** 격리 단계가 끝난 뒤 workspace 전체를 걸어 결과를 추측하지 않는다.
  결과는 commit set이고, 명령이 끝난 순간과 결과 확정이 밖에서 갈린다
  ([ADR-075](../../enode-design/adr/ADR-075-a-step-returns-a-commit-set-not-a-workspace.md)).
- **실행 상태를 결과와 따로 다룬다.** 끝난 단계의 scratch는 임대 창에서 지우지 않고 trash로
  옮기며, 실패 조사용 보존본(checkpoint)은 결과가 아니다
  ([ADR-076](../../enode-design/adr/ADR-076-a-checkpoint-preserves-runtime-state-not-a-step-result.md)).
- **주기 굽기를 일반 Run으로 돌린다.** 매일 repo sync와 빌드를 overlay 노드가 upper에서 짓고,
  형제 노드가 끝나기를 기다려 lower에 합친다
  ([ADR-077](../../enode-design/adr/ADR-077-a-bake-builds-in-the-upper-and-merges-into-the-lower.md)).

## 1.2 왜 그런가

| 관찰 | 뿌리 | 정본 |
|---|---|---|
| 결정론적 빌드가 끝난 뒤 임대가 오래 남는다 | Worker가 계약과 무관하게 workspace 전체를 걸어 `workspace.changed`를 만든다. `RecordDiff`가 켜지면 `git diff --binary`를 통째로 담는다 | ADR-075 §1 |
| 그 창이 밖에서 안 보인다 | 명령 종료부터 결과 보고까지 단계는 `CLAIMED` 하나이고 광고는 단계가 무엇을 하는지 싣지 않는다 | ADR-075 §10.1 |
| 끝난 단계의 upper를 보고 전에 지운다 | helper cleanup과 `Close`가 `runRoot`를 `RemoveAll`한다. 삭제는 항목 수에 비례한다 | ADR-076 §4.1 |
| 굽기를 받을 노드가 없다 | overlay 노드는 쓰기를 버리고, 공유 lower 위의 native 노드는 광고로 일반 계약과 갈리지 않으며, 굽는 동안 형제를 막을 장치가 없다. SunnyVM에서 native 노드가 overlay 노드의 lower 파일 5,649개를 바꿨다 | ADR-077 §1 |

## 1.3 코드 기준선

**이 팩이 넓히는 코드는 아직 `main`에 없다.** ADR-073의 실행 환경 구현(profile, `env
check`/`env apply`, runc-overlay `StepRuntime`, 제품 경로 연결)은 브랜치
`unit/runtime-environment-profile`(상위 `0a67716`)에만 있다. 이 팩의 줄 번호와 파일 경로는
그 브랜치 기준이다. 회차가 Workspace Detection을 돌기 전에 그 코드가 회차 브랜치에 있어야
한다(`constraints.md` 4절).

## 1.4 정본

`canon.md`. 설계 정본은 서브모듈 `enode-design/`(회차 브랜치가 `369270a`를 가리킨다)이고
어긋나면 `protocol/INVARIANTS.md`가 이긴다.

---

# 2. 기능

기능마다 무엇을 받는지는 `scene-gates.md`의 조각이 실동작으로 잰다. 결정의 근거는
`decisions.md`에 있고 여기서는 되풀이하지 않는다.

## 기능 1. 단계 결과를 Finalize로 닫는다

- build/test 단계에서 자동 `workspace.diff`와 전수 `workspace.changed` 수확을 없앤다. 지금
  Worker는 `RecordDiff`와 `Discover: true`로 수확한다(`internal/enode/claim.go:692-693`).
- 계약이 지목한 경로만 `stat`하는 `Result.Changed`(ADR-037)는 그대로 둔다.
- `$OUT`의 named output은 그대로 결과다.
- 계약 단계에 `effect` 필드를 둔다. 첫 값은 prepare(기능 5)이고, edit와 build/test의 기본값은
  ADR-075 §5의 표를 따른다.
- 받는 것: 조각 1
- 정본: [ADR-075 §4~§9](../../enode-design/adr/ADR-075-a-step-returns-a-commit-set-not-a-workspace.md)

## 기능 2. 명령 종료 보고와 phase

- 명령의 종료 status가 정해지면 노드가 Finalize 전에 멱등 종료 보고
  `POST /v1/runs/{run}/steps/{seq}/exited`를 한 번 보낸다. outcome과 종료 시각만 싣는다.
- Mediator는 단계 상태를 `CLAIMED`로 둔 채 `phase`를 `finalizing`으로, `phase_since`를 종료
  시각으로 둔다. merge 단계는 claim되는 순간 `waiting`이다.
- 진행 조회(`GET /v1/runs/{id}`)에 `phase`, `phase_since`, `exit`를 싣는다.
- result가 `exited_at`, `finalize`, `upload`를 다시 싣는다. 종료 보고가 유실돼도 봉인된 사실은
  같다.
- Record의 단계 기록이 종료 시각과 Finalize가 끝난 시각을 따로 가진다.
- 받는 것: 조각 2
- 정본: [ADR-075 §10](../../enode-design/adr/ADR-075-a-step-returns-a-commit-set-not-a-workspace.md),
  [mediator-api](../../enode-design/protocol/mediator-api.md)의 `POST /v1/steps/{id}/exited` 절

## 기능 3. finalizing 구간의 예산 둘

- Finalize 예산: 기본 1분. 계약이 단계마다 늘릴 수 있다. 넘으면 `finalize_timeout`.
- 업로드 예산: 기본 3분. 계약이 단계마다 바꿀 수 있다. 넘으면 `upload_timeout`.
- 둘 다 명령 실패와 구분되는 원인으로 Record에 남는다.
- 받는 것: 조각 3
- 정본: [ADR-075 §10.2](../../enode-design/adr/ADR-075-a-step-returns-a-commit-set-not-a-workspace.md)

## 기능 4. scratch는 trash로, 삭제는 보고 뒤에

- capture 정책과 무관하게 `runRoot`를 `RemoveAll`하지 않고 `<scratch>/trash/`로 `rename`한다.
  바꿀 자리는 helper cleanup, session `Close`, `abort` 셋이다
  (`internal/enode/runc_overlay_linux.go:1017`, `:437`, `:451`).
- 노드 데몬의 배경 삭제자가 결과 보고 뒤에 지운다. runtime과 같은 uid 매핑의 user namespace
  안에서, idle IO 우선순위로 돈다. 데몬 시작 때 한 번 돌아 남은 trash를 비운다.
- trash가 쌓여 여유가 `min_free_gb` 아래면 노드가 광고에서 빠진다. 새 기제는 없다.
- 받는 것: 조각 4
- 정본: [ADR-076 §4.1](../../enode-design/adr/ADR-076-a-checkpoint-preserves-runtime-state-not-a-step-result.md)

## 기능 5. 굽기 계약

- 굽기는 일반 Run이고 두 단계다. build(`effect: prepare`, `sync`, `builds[]`)와 merge(내장
  kind, `needs: build`)이며 같은 역할을 써서 같은 노드에 앉는다.
- `builds[]`의 항목은 `name`과 `command`다. 이름은 굽기 계약을 쓰는 쪽이 짓는다.
- 계약 검증이 400으로 막는 것: prepare 단계 뒤에 merge 단계가 없음, 구성 이름이 문자 규칙
  (소문자, 숫자, `-`)을 어김, 이름이 겹침.
- merge 대기 상한(기본 4시간)을 계약의 merge 단계가 바꿀 수 있다.
- 받는 것: 조각 5
- 정본: [ADR-077 §2](../../enode-design/adr/ADR-077-a-bake-builds-in-the-upper-and-merges-into-the-lower.md)

## 기능 6. build 단계는 upper에 짓는다

- build 단계는 평범한 overlay 단계와 똑같이 돈다. lower는 읽기 전용이다.
- 노드가 `sync`를 먼저, `builds`를 적힌 순서대로 돌리며 항목마다 시작·끝 시각과 exit code를
  기록한다. 이것이 prepare effect의 결과 manifest다.
- 하나라도 0이 아닌 exit code면 build 단계가 실패하고 합치지 않는다. upper는 trash로 가고
  실패한 시도는 `state.json`의 `last_attempt`에 남는다.
- 성공하면 Finalize가 upper만 합치기 대기 자리로 `rename`하고 나머지 `runRoot`는 trash로 보낸다.
- build 단계는 `repo manifest -r`로 떠 둔 pinned manifest를 남긴다.
- 받는 것: 조각 6
- 정본: [ADR-077 §2~§3](../../enode-design/adr/ADR-077-a-bake-builds-in-the-upper-and-merges-into-the-lower.md)

## 기능 7. 합치기

- merge 단계가 upper를 lower에 파일 단위 `rename`으로 합친다. 파일·symlink는 대체, lower에 없는
  디렉터리는 하위 트리째 한 번, 양쪽에 있는 디렉터리는 안으로 들어가 속성을 맞춘다. whiteout과
  opaque 디렉터리는 lower 쪽을 trash로 옮긴 뒤 표시를 치운다.
- 표시는 lower에 적용한 뒤에만 치운다. 그래서 어느 지점에서 끊겨도 남은 upper에 같은 절차를
  다시 돌리면 같은 결과가 된다.
- 시작 전 확인: 같은 filesystem, metacopy·redirect 표시 없음, 이 lower의 마운트 0.
- 마지막 동작으로 `.enode-metadata.json`을 쓴다(기능 9).
- 시제품: SunnyVM `~/bake-measure/merge.py`. 가짜 트리와 실제 BitBake 트리에서 merged view와
  목록이 일치하고 도중 강제 종료 뒤 재개도 일치했다.
- 받는 것: 조각 6, 조각 7
- 정본: [ADR-077 §4](../../enode-design/adr/ADR-077-a-bake-builds-in-the-upper-and-merges-into-the-lower.md)

## 기능 8. lower의 상태, 잠금, drain, 재개

- 상태 자리는 `~/.local/state/enode/lowers/<fsid>-<ino>/`이고 `lower.json`, `state.json`,
  `lower.lock`, `bake.lock`을 둔다. 키는 lower 루트의 `statfs` `f_fsid`와 inode다.
- 전이 상태는 committed, building, pending, merging이다. lower 안에 쓰지 않는다.
- 형제 overlay 노드는 임대를 쥔 동안 lower 공유 잠금을 쥔다(Run 단위, `Held`). prepare를 담은
  Run은 잡지 않는다. merge는 배타 잠금을 잡는다.
- 상태가 pending이나 merging이면 형제가 광고에 graceful drain을 싣는다.
- merge 대기가 상한을 넘으면 `merge_wait_timeout`으로 upper를 trash로 보내고 상태를 committed로
  돌리고 drain을 푼다. 합치기 본체에는 상한이 없다.
- 상태가 committed가 아니고 굽기 잠금의 주인이 살아 있으면 새 굽기는 `bake_in_progress`로 곧바로
  실패한다. 주인이 없으면 낡은 상태를 정리한다.
- 같은 lower의 어느 노드든 시작할 때 merging을 재개한다. 재개로 끝낸 합치기는 metadata에
  `resumed`와 원래 Run을 남긴다.
- `env check`가 scratch와 workspace의 `st_dev` 일치, lower 루트 소유 uid와 노드 uid의 일치,
  `lower.json` 신원을 확인한다.
- 받는 것: 조각 6, 조각 7, 조각 8
- 정본: [ADR-077 §5~§7](../../enode-design/adr/ADR-077-a-bake-builds-in-the-upper-and-merges-into-the-lower.md)

## 기능 9. metadata와 광고

- `.enode-metadata.json`은 `source`(url, branch, repo_id, head, ir, pinned, sync_command,
  synced_at), `builds[]`(name, command, started_at, finished_at, exit_code), `environment`,
  `workspace_target`, `bake`(run, node, merged_at, resumed, previous_ir)를 담는다.
- IR은 계약 칸이 아니라 sync 뒤 `.repo/manifests` HEAD에 정확히 붙은 태그에서 유도한다. 없으면
  null이고 `ir=`을 광고하지 않는다.
- 광고 키: `workspace.writes`(runc-overlay면 isolated, native면 in-place, 모든 노드가 낸다),
  `ir`, `repo.built.<name>=yes`.
- 받는 것: 조각 6
- 정본: [ADR-077 §5, §8](../../enode-design/adr/ADR-077-a-bake-builds-in-the-upper-and-merges-into-the-lower.md),
  [ADR-072 §6.4, §8](../../enode-design/adr/ADR-072-the-node-says-which-ir-it-stands-on.md)

## 기능 10. checkpoint capture와 Checkpoint Store

- `StepRuntime`이 capture capability(지원 여부, 범위 `workspace-upper`, 보장 `inspect-only`)를
  낸다. native는 `unsupported`(runtime)다.
- 정책이 요구하면 보고 전에는 `statfs` 여유 하한과 store의 현재 보존 총량만 보고 upper를 spool로
  `rename`한다. 크기 판정은 보고 뒤 store가 upper 순회로 하고, 넘으면 trash로 퇴출한다.
- receipt의 `checkpoint_capture`에 상태 다섯(`not_requested`, `unsupported`, `rejected`,
  `captured`, `failed`)과 `reason` 코드를, diagnostics에 상세를 둔다.
- 기본값: 정책 on-failure, TTL 48시간, 노드 보존 용량 (보존 총량 + 여유)의 20%, 퇴출은 오래된 것부터.
- spool은 소유자만 읽고, 외부 descriptor에 host 경로와 비밀을 싣지 않는다. 보존 중 암호화는
  요구하지 않는다.
- 재시작 때 미완료 capture와 만료 항목을 조정한다.
- 받는 것: 조각 9
- 정본: [ADR-076 §2~§5, §10](../../enode-design/adr/ADR-076-a-checkpoint-preserves-runtime-state-not-a-step-result.md)

## 기능 11. 결과 adapter

- agent edit를 위한 Git changeset adapter. 정확한 base identity를 가진다.
- Yocto producer adapter. 같은 namespace 안에서 target의 deploy output을 질의해 named output으로
  승격한다.
- 명시적으로 켜는 bounded discovery. 방문 수, 시간, 메모리, 결과 크기에 상한이 있고 상한에
  닿으면 부분 관찰임을 적는다. 정상 결과가 아니다.
- 받는 것: 조각 10
- 정본: [ADR-075 §6~§8](../../enode-design/adr/ADR-075-a-step-returns-a-commit-set-not-a-workspace.md)

## 기능 12. ADR-073 잔여

- 실제 최대 PyInstaller extraction이 256 MiB tmpfs에 드는지 확인한다.
- 운영 대상 host의 subordinate-ID 배치를 확인한다.
- ADR-073 문서가 E5(BitBake 완주)를 남은 것으로 적는데, SunnyVM 제품 경로에서 완주가 기록됐다
  (원장 `EN-bf4045e7`). 문서를 갱신한다.
- 받는 것: 조각 11
- 정본: [ADR-073](../../enode-design/adr/ADR-073-the-profile-prepares-the-environment-the-runtime-runs-the-step.md)
  §11 끝의 「남은 것」

## 기능 13. ADR-072의 결정 조건

- rev를 뽑은 판으로 실물 Run을 돌려, agent와 build가 같은 IR 위에서 diff를 주고받는 것을 본다.
  통과하면 ADR-072를 결정으로 올릴 근거가 선다.
- 받는 것: 조각 12
- 정본: [ADR-072 §9](../../enode-design/adr/ADR-072-the-node-says-which-ir-it-stands-on.md)

---

# 3. 보안

| 자리 | 요구 |
|---|---|
| 종료 보고 | 그 단계를 claim한 노드와 인스턴스(ADR-030)만 보낼 수 있다. 다른 노드의 보고는 거절한다. 인증은 다른 노드 표면과 같다 |
| trash 삭제 | user namespace 안에서 지운다. symlink를 따라가지 않는다. trash 밖을 지우지 않는다 |
| 합치기 | lower, 대기 upper, trash가 같은 filesystem이어야 시작한다. 합치는 경로가 lower 밖으로 나가지 않는다 |
| 상태 자리 | `~/.local/state/enode/lowers/…`는 노드 사용자 전용 권한이다 |
| spool | 소유자만 읽는다. TTL에 지운다. 외부 descriptor에 host 경로와 비밀을 싣지 않는다 |
| 계약의 명령 | `sync`와 `builds[].command`는 계약 제출자가 낸 명령을 격리 runtime 안에서 돌린다. 노드 설정이나 host 경로를 계약이 지정하지 못한다 |

---

# 4. 수용 기준

`scene-gates.md`의 조각 0~12다. 조각이 초록이 아니면 다음 유닛을 착수하지 않는다. 사람이
봐야 하는 조각은 보류로 넘기지 않는다.
