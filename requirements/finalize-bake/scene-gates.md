# 장면 조각 게이트 — 진행의 단위는 유닛 완료가 아니라 장면 완주다

장면 셋을 조각 열셋으로 잘랐다. **조각마다 실행 명령이 있고, 조각 게이트가 초록이 아니면 다음
유닛을 착수하지 않는다.** 이것이 요구사항의 수용 기준이다. 조각은 이 팩 안에서 「조각 N」으로
부르고, 다른 팩과 함께 말할 때는 「finalize-bake 조각 N」으로 부른다.

집행자는 **그 유닛을 구현하지 않은 사람**이다. 사람이 보는 조각은 SunnyVM(Hyper-V 노트북
VM이라 꺼져 있을 수 있다)과 스크래치 Mediator로 돈다. 합치기를 하는 조각은 버려도 되는 lower
(`~/yocto-fresh` 같은 것)에서만 돌고 `/srv/yocto`는 읽기만 한다.

---

# 1. 장면

## 장면 1 — 결정론 빌드 한 단계가 끝난다

1. overlay 노드에 결정론 BitBake 단계를 낸다.
2. 명령이 끝나는 즉시 진행 조회에 `finalizing`과 exit code가 보인다. 명령이 아직 도는 단계와 갈린다.
3. workspace가 수백만 파일이어도 결과 확정은 그 크기를 걷지 않는다.
4. 결과 보고 뒤 `runRoot`는 trash에 있고 곧 지워진다.
5. 봉인된 Record에 종료 시각과 Finalize가 끝난 시각이 따로 있다.

## 장면 2 — 하루치 굽기

1. 빈 lower에서 처음부터 굽는다(A). 합치기는 rename 몇 번으로 끝난다.
2. 형제 overlay 노드가 긴 Run을 도는 중에 증분 굽기(B)를 낸다.
3. B의 build는 곧바로 돈다. merge는 `waiting`이고 형제는 draining이다.
4. 형제의 Run이 끝나자 합치기가 몇 초에 끝난다.
5. 형제가 새 `ir`과 `repo.built.<name>`을 광고한다.
6. 합친 lower의 목록이 합치기 전 merged view와 같다.

## 장면 3 — 실패를 들여다본다

1. capture 정책이 on-failure인 노드에서 단계가 실패한다.
2. receipt에 `captured`와 불투명 ID가 남는다. 성공한 단계는 `not_requested`다.
3. 보고 뒤 store가 크기를 재고 한도 안이면 보관한다. 넘으면 퇴출하고 조회가 그것을 보인다.
4. TTL이 지나면 사라지고, 봉인된 Record의 `captured`는 당시의 사실로 남는다.

---

# 2. 조각

| 조각 | 이름 | 확인 (실동작) | 집행자 | 대상 | 재는 기능 | 먼저 서는 조각 |
|---|---|---|---|---|---|---|
| 0 | 기동이 안 깨졌다 | `go build ./...`, `go vet ./...`, `go test ./...`가 통과한다. ADR-073의 기존 제품 게이트가 회귀하지 않는다 | 기계 | 스크래치 | 바닥 | 없음 |
| 1 | 걷지 않는다 | 합성 300만 파일 workspace에서 결정론 no-op 단계의 시간이 빈 workspace와 같은 수준이다. `success_when.changed`가 지목한 N개만 `stat`한다. `$OUT` named output은 그대로 봉인된다 | 기계 | 스크래치 | 1 | 0 |
| 2 | 보인다 | exit 1로 끝나고 업로드가 긴 단계에서, 종료 즉시 진행 조회에 `finalizing`과 exit 1이 보인다. Mediator를 잠깐 멈춰 종료 보고를 유실시켜도 봉인된 Record가 같다. 재전송과 늦은 도착이 종결을 흔들지 않는다. 종료 보고를 보내지 않는 옛 노드는 `running`에 머문다. Record의 단계 기록에 두 시각이 있다 | 사람 | 스크래치 | 2 | 0 |
| 3 | 예산 | Finalize 예산을 넘기면 `finalize_timeout`, 업로드 예산을 넘기면 `upload_timeout`이다. 계약이 늘린 단계는 늘린 값으로 잰다. 둘 다 명령 실패와 원인이 갈린다 | 기계 | 스크래치 | 3 | 2 |
| 4 | trash | 1-8 규모(15만 파일, 9 GB) upper를 남기는 단계에서 종료부터 보고까지의 시간이 upper 크기와 무관하다. 보고 뒤 삭제자가 비운다. 권한 000인 `work/work`와 subordinate uid 항목도 지운다. 데몬을 재시작하면 남은 trash를 비운다. 여유가 `min_free_gb` 아래면 광고에서 빠졌다가 돌아온다 | 사람 | SunnyVM | 4 | 0 |
| 5 | 굽기 계약 | prepare 단계 뒤에 merge 단계가 없는 계약, 구성 이름이 규칙을 어긴 계약, 이름이 겹친 계약이 400이다. merge 대기 상한을 바꾼 계약이 그 값으로 기다린다 | 기계 | 스크래치 | 5 | 1 |
| 6 | 굽기가 돈다 | 장면 2의 1~6. 합친 lower의 목록이 merged view와 일치한다(`listing.py`식 비교). `.enode-metadata.json`의 필드가 다 있다. 빌드 하나를 일부러 실패시키면 합치지 않고 `last_attempt`에 남는다. manifest HEAD에 IR 태그가 없으면 `ir`은 null이고 광고하지 않는다 | 사람 | SunnyVM, 버려도 되는 lower | 5, 6, 7, 8, 9 | 1, 2, 4, 5 |
| 7 | 끊겨도 된다 | 합치기 도중 여러 지점에서 노드를 강제 종료하면 같은 lower의 다른 노드가 시작 때 재개하고 결과 목록이 한 번에 끝낸 것과 같다. metadata에 `resumed`와 원래 Run이 있다. 표시 종류를 모두 담은 가짜 트리에서 1~30번째 연산 뒤 강제 종료하는 시험이 Go 테스트로 돈다 | 기계, 사람 | 스크래치, SunnyVM | 7, 8 | 6 |
| 8 | 배타와 대기 | 굽기 중 두 번째 굽기가 `bake_in_progress`로 곧바로 실패한다. 굽기 노드가 pending에서 죽으면 다른 노드가 낡은 상태를 정리하고 drain이 풀린다. 짧게 준 대기 상한을 형제 Run이 넘기면 `merge_wait_timeout`으로 upper가 trash로 가고 상태가 committed로 돌고 drain이 풀린다. bind 별칭으로 같은 lower를 가리키는 두 노드가 같은 상태 자리를 쓴다. 다른 사용자의 노드와 다른 filesystem의 scratch는 `env check`가 not ready다 | 사람 | SunnyVM | 8 | 6 |
| 9 | 보존 | 장면 3의 1~4. native 노드는 `unsupported`(`runtime`)다. 여유 하한 미달은 `rejected`(`free_space`)다. 단계 outcome과 commit set이 capture 성패와 무관하다. 재시작 뒤 미완료 capture가 조정된다 | 사람 | SunnyVM | 10 | 4 |
| 10 | adapter | agent edit의 추가·수정·삭제가 정확한 base에 적용되는 changeset이고 compiler 부산물이 없다. Yocto 빌드에서 deploy output만 commit set에 있다. bounded discovery가 상한에서 부분 관찰임을 적는다 | 사람 | SunnyVM | 11 | 1 |
| 11 | ADR-073 잔여 | 실제 최대 PyInstaller extraction이 256 MiB tmpfs에 든다. 운영 대상 host의 subordinate-ID 배치를 확인한다. ADR-073 §11의 E5 문장이 갱신됐다 | 사람 | 운영 host | 12 | 0 |
| 12 | ADR-072 결정 조건 | rev를 뽑은 판에서 agent와 build가 같은 IR 위에서 diff를 주고받는 실물 Run이 완주한다 | 사람 | SunnyVM 또는 사내 | 13 | 6 |

---

# 3. 조각마다의 명령

값은 시작값이다. 유닛의 Code Generation 계획이 실제 스크립트로 굳힌다.

```text
   게이트를 돌리기 전에
     eval "$(scripts/testdb.sh)"
     export M=http://<Mediator 주소>:8080
     export T=<bootstrap 토큰>
     SunnyVM 조각: ssh sunnyvm (ConnectTimeout 30).  안 붙으면 노트북이 꺼진 것이다

   조각 1   합성 트리 3,000,000 파일 workspace 노드에 결정론 no-op 계약을 낸다
            빈 workspace 노드와 단계 시간을 비교한다 (step finished took)
            success_when.changed 에 경로 N 개를 적고 노드 로그의 stat 수를 센다

   조각 2   runctl submit (exit 1 로 끝나고 큰 $OUT 을 올리는 명령 단계)
            명령이 끝나는 순간 curl -s -H "Authorization: Bearer $T" "$M/v1/runs/<id>" | jq '.steps[]|{phase,exit}'
            Mediator 를 10초 멈춘 채 같은 계약을 다시 → 봉인 뒤 runctl record 의 steps/NN-*.json 비교

   조각 4   SunnyVM 에서 bitbake -C compile virtual/kernel 단계 → 종료 시각과 보고 시각의 차
            ls <scratch>/trash 가 보고 직후 차 있고 잠시 뒤 비어 있다
            enode 재시작 뒤 남은 trash 가 비워진다

   조각 6   ~/yocto-fresh 같은 빈 lower 위에서 굽기 A 계약 → 굽기 B 계약 (형제 노드에 긴 Run 을 먼저 낸다)
            합치기 전: user namespace 에서 ro overlay(lowerdir=upper:lower) 목록
            합친 뒤: lower 목록 → diff 가 비어야 한다
            curl "$M/v1/capabilities" 에 ir 과 repo.built.<name> 이 보인다

   조각 7   merge 단계가 도는 중 enode 를 SIGKILL (여러 번, 다른 시점)
            다른 노드를 시작 → state.json 이 committed, metadata 의 resumed 가 true

   조각 8   굽기 중 같은 lower 로 두 번째 굽기 계약 → 곧바로 FAILED, 사유 bake_in_progress
            merge 대기 상한을 1분으로 준 계약 + 형제에 5분짜리 Run → merge_wait_timeout
```

---

# 4. 게이트가 빨간 채로 다음 유닛을 착수하면

착수하지 않는다. 빨간 이유가 그 유닛 밖이면 진행자가 이 회차의 `audit.md`에 적고 조각을
**보류**로 표시한 뒤 넘어간다. **보류는 통과가 아니다.**

SunnyVM이 꺼져 있어 못 돈 조각은 보류다. 켜진 뒤 돌린다. 사람이 보는 조각(2, 4, 6, 8, 9)을
코드 테스트가 초록이라는 것으로 대신하지 않는다.
