# 결정표 — 이미 정해진 것. 다시 논의하지 않는다

이 표는 2026-09-21~23 설계 검토에서 **사용자가 확정한 값**과 그 근거다. 게이트에서는 확인만
하고 새로 묻지 않는다. 결정마다 정본의 절(ADR)과 원장 항목(`EN-…`, Office Task `f806768d`)을
단다. 정본과 어긋나면 `enode-design/protocol/INVARIANTS.md`가 이긴다(`canon.md`).

**게이트 규칙 한 줄 — 권장을 벗어나면 근거를 적는다.**

| 절 | 내용 |
|---|---|
| 0 | 정본이 이미 지는 값. 팩이 고를 자리가 아니다 |
| 1 | 단계 결과와 finalizing (ADR-075) |
| 2 | 실행 상태, trash, checkpoint (ADR-076) |
| 3 | 굽기 (ADR-077) |
| 4 | 기각과 순연 |
| 5 | 팩이 채운 권장값 |
| 6 | 이월 — 구현 계획에서 닫는 값 |
| 7 | 정본에 되돌려 올리는 것 |

ADR 링크의 짧은 이름은 다음 파일을 가리킨다.

| 이름 | 파일 |
|---|---|
| ADR-075 | [`ADR-075-a-step-returns-a-commit-set-not-a-workspace.md`](../../enode-design/adr/ADR-075-a-step-returns-a-commit-set-not-a-workspace.md) |
| ADR-076 | [`ADR-076-a-checkpoint-preserves-runtime-state-not-a-step-result.md`](../../enode-design/adr/ADR-076-a-checkpoint-preserves-runtime-state-not-a-step-result.md) |
| ADR-077 | [`ADR-077-a-bake-builds-in-the-upper-and-merges-into-the-lower.md`](../../enode-design/adr/ADR-077-a-bake-builds-in-the-upper-and-merges-into-the-lower.md) |
| ADR-072 | [`ADR-072-the-node-says-which-ir-it-stands-on.md`](../../enode-design/adr/ADR-072-the-node-says-which-ir-it-stands-on.md) |
| ADR-073 | [`ADR-073-the-profile-prepares-the-environment-the-runtime-runs-the-step.md`](../../enode-design/adr/ADR-073-the-profile-prepares-the-environment-the-runtime-runs-the-step.md) |
| mediator-api | [`protocol/mediator-api.md`](../../enode-design/protocol/mediator-api.md) |

---

# 0. 정본이 이미 지는 값

| 값 | 정본 |
|---|---|
| 노드 하나가 한 번에 Run 하나를 쥔다. 이 팩은 `I1`을 고치지 않는다 | `INVARIANTS` `I1` |
| 봉인되지 않은 것은 Record가 아니고, 봉인된 Record는 append-only다 | `INVARIANTS` `I4` |
| 매칭은 순수 라벨의 부분집합 비교이고 first available이다. 술어나 부정 조건을 열지 않는다 | [ADR-011](../../enode-design/adr/ADR-011-lineage-and-position.md) |
| 임대 갱신은 노드 광고의 응답이 나른다 | [ADR-016](../../enode-design/adr/ADR-016-heartbeat-carries-the-lease.md) |
| 광고에서 키가 없으면 그 노드는 그것을 못 한다는 뜻이다 | [ADR-012](../../enode-design/adr/ADR-012-enode-self-advertises.md) |
| drain은 graceful이 기본이고 draining 노드는 후보에서 빠진다 | [ADR-063](../../enode-design/adr/ADR-063-the-node-says-what-it-is-the-owner-says-who-may-use-it.md) |
| 계약이 지목한 경로만 `stat`하는 `Result.Changed`는 판정 증거다 | [ADR-037](../../enode-design/adr/ADR-037-the-predicate-must-not-be-authored-by-the-agent.md) |
| `StepRuntime` session이 `Run`부터 결과 확정까지 같은 namespace를 소유한다 | ADR-073 |

---

# 1. 단계 결과와 finalizing (ADR-075)

| # | 결정 | 근거 | 정본 | 원장 |
|---|---|---|---|---|
| 1-1 | 단계 봉투는 receipt, commit set, predicate evidence, diagnostics 넷이다. cache, intermediate, 분류되지 않은 workspace 쓰기는 정상 결과가 아니다 | 파일시스템 변경에는 결과의 뜻이 없다. writable은 publishable과 동의어가 아니다 | ADR-075 §2, §4 | `EN-aaddb63f` |
| 1-2 | build/test에서 자동 `workspace.diff`와 전수 `workspace.changed`를 만들지 않는다. `Result.Changed`는 유지한다 | 비용이 결과가 아니라 workspace 크기를 따랐다. SunnyVM 300만 파일에서 `RecordDiff`가 12.79초를 1분 31초로 늘렸다 | ADR-075 §1, §8, §14 | `EN-6f0a6803`, `EN-4ffc8c90` |
| 1-3 | actor와 effect를 가른다. 계약 단계의 `effect`는 명시 필드이고, prepare 몫이 이 팩에서 확정된다 | 노드가 upper를 버릴지 합칠지가 이 값으로 갈리므로 유도하지 않고 Record에 보이게 둔다 | ADR-075 §5, ADR-077 §2 | `EN-bde53bef` |
| 1-4 | `Harvest`의 수명은 살리고 의미를 Finalize로 바꾼다. 모든 operation은 lease-aware context와 예산을 받는다 | 정상 경로 비용이 commit set과 지목 경로 수를 따라야 한다 | ADR-075 §9 | — |
| 1-5 | 명령의 종료 status가 정해지면 노드가 Finalize 전에 멱등 종료 보고를 한 번 보낸다. outcome과 종료 시각만 싣는다 | 명령이 끝난 단계와 도는 단계를 밖에서 가른다. 사내 증상이 이 창이었다 | ADR-075 §10.2, mediator-api | `EN-b66e20a9` |
| 1-6 | 단계 상태는 `CLAIMED`로 두고 phase 칸을 붙인다. phase는 `running`, `finalizing`, `waiting`이다 | 새 상태 값은 진행 중 판정을 곳곳에서 고치게 하는데 바뀌는 Mediator 판정은 없다 | ADR-075 §10.3 | `EN-57b57465`, `EN-dad477ec` |
| 1-7 | 종료 보고는 판정이 아니다. 대조, 다음 단계 생성, 정산을 하지 않는다. 종결 전이는 result 하나다 | 판정 재료는 Finalize가 끝나야 나온다 | ADR-075 §10.2 | `EN-b66e20a9` |
| 1-8 | 임대는 풀지 않는다 | namespace가 살아 있고 다음 단계는 commit set을 기다린다 | ADR-075 §10.2 | `EN-b66e20a9` |
| 1-9 | 종료 보고의 키는 Run, seq, attempt, 노드 instance다. 재전송과 늦은 도착은 조용히 성공하고, result는 종료 보고의 수락에 기대지 않는다 | 유실돼도 봉인된 사실이 같아야 한다 | ADR-075 §10.4 | `EN-b66e20a9` |
| 1-10 | Finalize 예산 기본 1분, 계약이 늘릴 수 있다. 넘으면 `finalize_timeout` | 사용자 결정 | ADR-075 §10.2 | `EN-dad477ec` |
| 1-11 | 업로드 예산 기본 3분, 계약이 바꿀 수 있다. 넘으면 `upload_timeout` | 산출물 크기는 계약 작성자가 안다. descriptor와 payload 전송의 선과 같다 | ADR-075 §10.2 | `EN-dad477ec` |
| 1-12 | merge 단계는 claim되는 순간 phase가 `waiting`이다. 합치기 몇 초는 따로 알리지 않고 result가 닫는다 | 4시간짜리 running은 기다리는지 합치는지 모른다 | ADR-075 §10.3 | `EN-dad477ec` |
| 1-13 | Record의 단계 기록은 종료 시각과 Finalize가 끝난 시각을 따로 가진다 | Finalize 비용이 모든 Record에 남는다 | ADR-075 §4, §10.2 | `EN-b66e20a9` |

---

# 2. 실행 상태, trash, checkpoint (ADR-076)

| # | 결정 | 근거 | 정본 | 원장 |
|---|---|---|---|---|
| 2-1 | capture, 보관, publication은 소유자가 다르다. checkpoint는 결과가 아니다. 봉투에 다섯째 종류를 더하지 않는다 | 결과 의미를 ADR-075가 닫았다 | ADR-076 §2 | `EN-19b217f9` |
| 2-2 | capture 상태는 다섯(`not_requested`, `unsupported`, `rejected`, `captured`, `failed`)으로 닫고 원인은 `reason` 코드로 둔다. unsupported는 노드와 runtime에 걸린 원인, rejected는 옮기기 전 단계별 원인, failed는 옮기기 시작한 뒤의 원인이다 | 상태는 소비자가 분기하는 닫힌 집합이고 원인은 는다 | ADR-076 §2 | `EN-0054b84f` |
| 2-3 | `reason` 코드는 `runtime`, `cross_filesystem`, `quota`, `free_space`, `lease_budget`, `io`다 | 보고 전 순회와 암호화 전제를 걷으며 `admission_bound`, `no_admission`, `no_block_encryption`을 뺐다 | ADR-076 §2 | `EN-e4ed4726`, `EN-c6184167` |
| 2-4 | 임대 창에서는 O(1) 소유권 이전만 한다. 크기에 비례하는 일은 보고 뒤 store가 한다 | 보고 전 창이 사내 증상의 자리다 | ADR-076 §4 | `EN-19b217f9` |
| 2-5 | capture 정책과 무관하게 `runRoot`는 `<scratch>/trash/`로 `rename`하고 배경 삭제자가 보고 뒤에 지운다. 삭제자는 user namespace 안, idle IO, 데몬 시작 때 한 번 | 임대 창의 비용이 upper 크기와 무관해지고 모든 경로가 한 기제를 쓴다. 실측으로는 18만 항목 삭제가 1.0초라 급하지 않고 근거는 구조다 | ADR-076 §4.1 | `EN-d273e9c3`, `EN-14f8ad1c` |
| 2-6 | trash가 쌓여 여유가 `min_free_gb` 아래면 노드가 광고에서 빠진다 | 이미 있는 광고 조건이다 | ADR-076 §4.1 | `EN-d273e9c3` |
| 2-7 | 보존 판정을 둘로 나눈다. 보고 전에는 `statfs` 여유 하한과 store의 보존 총량만 본다. 크기 판정은 보고 뒤 store가 하고 넘으면 trash로 퇴출한다 | 판정하는 순간 bytes는 이미 디스크에 있어 판정은 보관 기간만 정한다 | ADR-076 §5 | `EN-e4ed4726` |
| 2-8 | 보고 뒤 크기 측정은 upper 순회다. project quota는 쓰지 않는다. 실행 중 scratch에 한도를 강제하지 않는다 | 순회가 2.6~4초로 싸다. project quota 조회는 root와 관리자 준비가 필요하다. 강제하면 `EDQUOT`가 단계 결과를 바꾼다 | ADR-076 §5 | `EN-c6184167` |
| 2-9 | 노드 로컬 spool에 보존 중 암호화를 요구하지 않는다. 중앙 store는 전송·저장 암호화와 접근 기록을 요구한다 | 같은 내용이 lower와 scratch에 같은 소유자로 평문으로 있다 | ADR-076 §5 | `EN-c6184167` |
| 2-10 | 첫 기본값: 정책 on-failure, TTL 48시간, 노드 보존 용량 (보존 총량 + 여유)의 20%, 여유 하한 `min_free_gb`, 퇴출은 오래된 것부터, 삭제자는 idle IO | 사용자 결정. 노드 소유자가 바꿀 수 있다 | ADR-076 §10 | `EN-c6184167` |
| 2-11 | 원격 계약이나 agent 출력은 TTL을 늘리거나 host 경로를 지정하거나 보존을 강제하지 못한다 | 정책은 노드 소유자의 것이다 | ADR-076 §4 | — |

---

# 3. 굽기 (ADR-077)

| # | 결정 | 근거 | 정본 | 원장 |
|---|---|---|---|---|
| 3-1 | profile이 적용된 노드는 전부 overlay 노드이고 공유 lower 위에 native 노드를 두지 않는다 | lower를 지키기 위한 전제다. 사내 native 환경에는 빌드 도구가 있고, 도구가 빠진 곳은 runc rootfs 안이었다 | ADR-077 §1 | `EN-c25dc3b5` |
| 3-2 | 굽기는 일반 Run의 언어로 돈다. 새 실행 기계가 없다 | ADR-070 §4 | ADR-077 §2 | `EN-bde53bef` |
| 3-3 | 굽기는 overlay의 upper에서 짓고 성공하면 lower에 합친다(방식 다) | 굽기가 실패하면 upper만 버린다. lower가 바뀌는 창이 합치기로 준다. 작업 단계와 같은 rootfs와 경로에서 짓는다 | ADR-077 §3 | `EN-bde53bef` |
| 3-4 | 층을 쌓아 두지 않는다. 날마다 합친다 | 층을 남기면 ADR-070 §5.6이 갈래 가를 기각한 근거가 걸린다 | ADR-077 §9 | `EN-47157628` |
| 3-5 | 합치기는 파일 단위 `rename`이다. 표시는 lower에 적용한 뒤에만 치운다 | upper가 곧 남은 일의 기록이라 끊겨도 다시 돌리면 같다. 사용자는 「일단 해본다」로 정했다 | ADR-077 §4 | `EN-47157628`, `EN-04af5676` |
| 3-6 | lower와 upper는 사용자 마운트포인트 안에 둔다. 같은 filesystem을 강제하고 `env check`가 `st_dev`로 확인한다 | 다른 filesystem 마운트포인트가 없는 환경이다 | ADR-077 §4, §5 | `EN-bde53bef` |
| 3-7 | 합치기만 미룬다. 형제의 Run이 끝날 때까지 기다린다. 단계가 아니라 Run 단위다 | build는 lower를 읽기만 한다. Run 도중에 바닥 IR이 바뀌면 안 된다 | ADR-077 §6 | `EN-dd51586d` |
| 3-8 | 합치기는 굽기 Run의 둘째 단계다. 기다림은 merge 단계 안에서 한다 | Record에 남고 Finalize 예산에 걸리지 않는다 | ADR-077 §2, §6 | `EN-47157628` |
| 3-9 | 잠금은 둘이다. lower 잠금(형제 공유, merge 배타)과 굽기 잠금(building부터 committed까지). prepare를 담은 Run은 공유 잠금을 잡지 않는다. 잠금 승격은 쓰지 않는다 | 자기 잠금을 피한다. `flock` 승격은 원자적이지 않다 | ADR-077 §6 | `EN-47157628` |
| 3-10 | 상태가 pending이나 merging이면 형제가 광고에 graceful drain을 싣는다. Mediator는 바뀌지 않는다 | draining은 이미 후보에서 빠진다 | ADR-077 §6, §8 | `EN-47157628` |
| 3-11 | merge 대기 상한은 기본 4시간이고 계약이 바꿀 수 있다. 넘으면 `merge_wait_timeout`. 합치기 본체에는 상한이 없다 | 사용자 결정. 가장 긴 빌드 실측이 79분이었다. 본체를 끊으면 lower가 merging에 묶인다 | ADR-077 §6 | `EN-dad477ec` |
| 3-12 | 상태가 committed가 아니고 굽기 잠금의 주인이 살아 있으면 새 굽기는 `bake_in_progress`로 곧바로 실패한다. 주인이 없으면 낡은 상태를 정리한다 | 기다리면 임대를 쥔 채 drain을 늘린다. 계기가 다음 주기에 다시 낸다 | ADR-077 §7 | `EN-47157628` |
| 3-13 | 끊긴 합치기는 같은 lower의 어느 노드든 시작할 때 재개한다. 재개로 끝낸 합치기는 metadata에 `resumed`와 원래 Run을 남긴다 | lower가 merging에 묶이면 형제가 전부 멈춘다 | ADR-077 §7 | `EN-47157628` |
| 3-14 | 상태와 잠금의 자리는 `~/.local/state/enode/lowers/<fsid>-<ino>/`이다. 한 lower의 노드는 한 사용자가 같은 home을 보는 환경에서 띄운다 | lower 옆은 bind 별칭마다 갈리고 부모가 root 소유인 기계가 있다. inode 키는 별칭에도 같다 | ADR-077 §5 | `EN-98622f9b` |
| 3-15 | 전이 상태는 lower 밖에, 확정 metadata `.enode-metadata.json`은 lower의 repo root에 합치기 창 안에서만 쓴다 | 마운트된 lower를 고치는 것은 overlayfs가 정의하지 않는다 | ADR-077 §5 | `EN-47157628` |
| 3-16 | metadata는 source(url, branch, repo_id, head, ir, pinned, sync_command, synced_at), builds[](name, command, started_at, finished_at, exit_code), environment, workspace_target, bake(run, node, merged_at, resumed, previous_ir)를 담는다 | 사용자가 정한 항목(원본 주소, 브랜치, 다운로드 시점, 빌드 시점, exit code, 원본 명령)과 재현에 필요한 것 | ADR-077 §5 | `EN-ce349ec8` |
| 3-17 | build 단계는 `sync`와 이름 붙은 `builds[]`를 구조로 싣는다. 명령을 그대로 적는다 | 결정론을 말하기에 명령을 적는 것이 가장 간단하다(사용자) | ADR-077 §2 | `EN-ce349ec8` |
| 3-18 | sync나 빌드 하나라도 실패하면 합치지 않는다. 실패한 시도는 Run Record와 `state.json`의 `last_attempt`에 남는다 | 일부만 합치면 빠진 구성을 요구하는 계약이 422로 영구 거절된다 | ADR-077 §2 | `EN-ce349ec8` |
| 3-19 | 구성 이름은 굽기 계약을 쓰는 쪽이 명령 옆에서 짓는다. 문자 규칙과 중복만 검사하고 어기면 400이다 | 이름과 명령이 한 자리에 있어야 어긋나지 않는다. 광고된 이름은 `GET /v1/capabilities`에 보인다 | ADR-077 §2 | `EN-ce349ec8` |
| 3-20 | IR은 계약 칸이 아니라 sync 뒤 manifest HEAD의 태그에서 유도한다. 없으면 null이다 | sync 명령이 이미 어느 IR인지 말한다. 같은 사실을 두 곳에 두지 않는다 | ADR-077 §5 | `EN-ce349ec8` |
| 3-21 | 광고: `workspace.writes`(runc-overlay면 isolated, native면 in-place, 모든 노드), `ir`, `repo.built.<name>` | runtime 종류가 광고에 없었다. 키가 없으면 못 한다는 뜻이 되므로 모든 노드가 낸다 | ADR-077 §8 | `EN-bde53bef` |
| 3-22 | `env check`가 lower 루트 소유 uid와 노드 uid의 일치, `lower.json` 신원을 확인한다 | 3-14의 전제를 지킨다 | ADR-077 §5 | `EN-98622f9b` |

---

# 4. 기각과 순연

| # | 무엇 | 판정 | 되살릴 조건 또는 여는 조건 | 정본 |
|---|---|---|---|---|
| 4-1 | finalizing을 새 단계 상태로 둔다 | 기각 | Finalize 구간에서만 다르게 동작해야 하는 Mediator 판정이 생긴다 | ADR-075 §13 |
| 4-2 | 광고에 phase를 싣는다 | 기각 | 종료 사실을 Record에 남길 필요가 없어진다 | ADR-075 §13 |
| 4-3 | Mediator가 Finalize 예산을 강제한다 | 순연 | 광고를 계속하면서 예산을 넘기는 노드가 관측된다. 강제해도 계약의 예산을 쓰므로 제출자와 충돌하지 않는다 | ADR-075 §10.2 |
| 4-4 | 공유 lower 위에 native 굽기 노드를 둔다 | 기각 | lower를 나누지 않는 기계 | ADR-077 §9 |
| 4-5 | build 단계의 Finalize에서 기다렸다 합친다 | 기각 | 없다 | ADR-077 §9 |
| 4-6 | Run 밖에서 합친다, 봉인된 Record에 이어 붙인다 | 기각 | `I4` | ADR-077 §9 |
| 4-7 | Mediator가 lower 점유 행을 갖는다 | 기각 | 기계 밖에서 lower를 공유하는 구성 | ADR-077 §9 |
| 4-8 | 다른 filesystem 사이의 합치기 | 순연 | 다른 마운트포인트를 고려해야 하는 환경 | ADR-077 §9 |
| 4-9 | 합치기 대기를 Run 경계에서 멈췄다 잇기 | 순연 | 기다림이 굽기 주기를 넘는 사례. ADR-064 §3.4가 먼저다 | ADR-077 §9 |
| 4-10 | 첫 준비도 Run으로 | 순연 | 빈 lower에서 repo 광고가 없는 노드를 굽기 계약이 고르는 법. 실행 쪽 조건은 실측으로 섰다 | ADR-077 §9, §12 |
| 4-11 | 여러 사용자나 환경이 한 lower를 나눈다 | 순연 | 그런 구성이 요구된다 | ADR-077 §9 |
| 4-12 | 저장소가 구성 이름과 명령을 정의 파일로 갖는다 | 순연 | 같은 저장소를 여럿이 굽기 계약으로 구워 이름이 갈라진다 | ADR-077 §9 |
| 4-13 | 중앙 checkpoint store, 다른 노드로 재개 | 순연 | 대용량 직접 전송과 post-step 권한, portable format | ADR-076 §6, §8 |
| 4-14 | Windows `StepRuntime` | 이 팩 밖 | Linux 쪽이 닫히고 실제 Windows 노드에서 W1~W7 probe | [ADR-074 §6~§7](../../enode-design/adr/ADR-074-windows-isolation-is-a-host-sandbox-not-a-rootfs.md) |

---

# 5. 팩이 채운 권장값

| 항목 | 권장 | 근거 |
|---|---|---|
| 착수 순서 | 기능 1~3과 기능 4를 먼저(서로 기대지 않는다), 그다음 기능 5~9, 그다음 기능 10, 기능 11. 기능 12~13은 병행 | 굽기가 effect, phase waiting, trash에 기댄다. capture는 trash 뒤다 |
| 실측 환경 | SunnyVM(`sunnyvm`, Hyper-V 노트북 VM). 합치기 시험은 버려도 되는 lower(`~/yocto-fresh`)에서 하고 `/srv/yocto`는 읽기만 한다 | 1-8 실측이 그렇게 돌았다 |
| 합치기 구현 | Go로 옮기되 `merge.py`의 규칙과 시험(가짜 트리의 표시 종류 전부, 1~30번째 연산 뒤 강제 종료)을 그대로 테스트로 삼는다 | 시제품이 실측에서 맞았다 |
| AI-DLC 확장 opt-in | 앞 회차들의 선택(security-baseline 켬)을 잇는 것을 권한다. Requirements Analysis가 묻는다 | 사용자가 이번 팩에 대해 아직 답하지 않았다 |

---

# 6. 이월 — 구현 계획에서 닫는 값

| 값 | 정본 |
|---|---|
| 두 예산과 merge 대기 상한의 계약 표기, 노드 쪽 상한을 둘지 | ADR-075 §16 |
| 명령 앞에서 임대를 쥐는 준비 구간을 phase에 올릴지 | ADR-075 §16 |
| checkpoint 하나의 상한과 inode 한도 | ADR-076 §10 |
| checkpoint descriptor와 조회 API, `reason`의 wire 표기 | ADR-076 §10 |
| 배경 삭제자가 한 번에 지우는 양 | ADR-076 §10 |
| manifest HEAD에 태그가 여럿일 때 IR 형식 규칙의 자리. 사내 형식은 `IR<YYMMDD>_<HHMMSS>` | ADR-077 §11 |
| Git changeset의 wire format과 10 MiB 초과 정책, producer adapter의 등록과 versioning | ADR-075 §16 |

---

# 7. 정본에 되돌려 올리는 것

| 무엇 | 자리 |
|---|---|
| 계약 문법: `effect`, 단계 예산, `sync`, `builds[]`, merge kind, merge 대기 상한 | [`protocol/run-contract.md`](../../enode-design/protocol/run-contract.md). 아직 반영되지 않았다 |
| ADR-073의 E5 상태 | ADR-073 §11 끝의 「남은 것」. SunnyVM 제품 경로 완주(`EN-bf4045e7`)를 반영한다 |
| ADR-072의 결정 승격과 §8 「그 값을 누가 적는가」 닫기 | 기능 13 조각이 초록이 되면 |
| ADR-076, ADR-077의 결정 승격 | 이 팩의 게이트가 초록이 되면 |
| enode-design의 `unit/runtime-environment-profile`을 enode-design `main`으로 | ADR-075~077이 enode-design `main`에 없다 |
