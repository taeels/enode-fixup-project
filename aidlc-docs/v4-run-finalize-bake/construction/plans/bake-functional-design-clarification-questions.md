# `bake` — Functional Design 되물음

QA 검수 (2026-09-27T08:40:08Z · 판정 「고친 뒤 승인」) 가 산출물 셋에서 사용자가 정해야 할 자리를 찾았다. 물음은 다섯이다. 답을
`[Answer]:` 뒤에 적는다. 권장을 **A** 에 둔다. 산출물은 권장대로 적어 두고, 그 문장마다 「되물음 N 의 답에 따른다」로 표시했다 —
답을 받으면 그 표시를 지우거나 문장을 바꾼다.

**답** — 2026-09-27T10:38:41Z · 다섯 모두 A (「권장대로」). 되물음 2 는 2026-09-27T11:59:51Z 에 B 로 다시 답했다 (QA 재검 D1 — A 는 I3 과
부딪친다). 산출물의 표시를 「되물음 N 답 A」 · 「되물음 2 답 B」로 확정했다 (audit 의 「되물음 답 반영」 · 「되물음 2 를 B 로 반영」 항목).

**경로** — 규칙 · 흐름 · 엔티티는 `construction/bake/functional-design/` 의 `business-rules.md` · `business-logic-model.md` ·
`domain-entities.md`. 계획은 `construction/plans/bake-functional-design-plan.md`.

**내기 전에 대 본 것** — 네 흠에 물음마다 대 봤다.

```text
   흠                            대 본 결과
   이미 정한 것을 다시 묻기         물음 4 는 물음 2 의 A 에 묶여 있던 셋째 갈래를 따로 묻는다 — QA 가 얻는 것이 작다고 했고, 정본 되돌림이
                                 그 갈래에 걸린다.  진행자가 물으라고 했다.  물음 2 의 C · 물음 3 의 C 는 contract-grammar 가 정한 것
                                 (판정 조건은 lint 경고 · 계약 문법) 을 다시 묻는 선택지라 그 옆에 적었다.  물음 1 은 계획 3.13 이 묻지
                                 않고 정한 「기동 때만」을 다시 연다 — 근거 문장 (「다음 굽기가 치운다」) 이 코드에서 거짓이었다.
                                 물음 5 의 다섯은 답 뒤에 산출물이 정한 것이라 처음 묻는다
   측정 없는 근거                  물음 1 · 2 · 4 는 코드 줄을 단다.  물음 3 은 이 기계에서 측정했다 (dash 에 source 가 없다).  물음 5 는
                                 항목마다 규칙의 절을 단다
   넘김과 어긋난 선택지             물음 2 의 A 는 contract-grammar 흐름 3절이 bake 에 넘긴 「합칠 것이 없음을 어떻게 알리나」 안이다.
                                 물음 1 의 A · B 는 lower-state 가 넘긴 몸통 (pending 에서 임대가 사라짐 · 산 주인) 을 건드리지 않고 주인이
                                 죽은 갈래만 더한다
   요구를 빼는 선택지               물음 1 의 C 는 조각 8 (배타와 대기) 의 「pending 에서 죽으면 다른 노드가 정리하고 drain 이 풀린다」
                                 (scene-gates.md:53) 를 다른 노드의 재시작으로만 채운다 — 그 선택지 안에 적었다.  물음 2 의 B 는 판정
                                 조건 없는 굽기 계약이 SUCCEEDED 가 되는 것을 그대로 둔다 (contract-grammar 의 결정 안)
```

---

## Question 1

주인이 죽은 낡은 상태 (building · pending) 를 광고 주기가 치우나? (QA S1 · 답 2 재개의 때의 글자 밖 · ADR-077:289 「이 정리와 재개는 … 시작할 때 한다」를 바꾼다)

지금 산출물은 계획 3.13 대로 기동과 build claim 만 치운다. 그 근거로 적은 「다음 굽기가 치운다」가 거짓이었다 — pending 이면 그 lower 의
모든 노드가 굽기 drain 을 싣고 (`internal/enode/lowerguard.go:212` ~ `:216`), drain 인 노드는 매칭에서 빠진다 (`internal/store/queue.go:302` ~ `:308`).
소유자는 굽기 drain 을 풀 수 없고 (`lowerguard.go:145` ~ `:148`), 굽기 Run 을 취소해도 치우는 몸통은 굽기를 쥔 프로세스 안에서만 돈다
(`lowerguard.go:419` ~ `:433`). 푸는 길은 그 lower 의 노드 하나를 다시 띄우는 것뿐이다. 살아 있는 데몬도 pending 을 남긴다 — pending 을 쓴 뒤
merge claim 전에 SIGTERM 을 받았을 때 · 몸통 자신의 쓰기가 실패했을 때. building 은 형제에 drain 을 걸지 않는다.

A) 광고 주기가 committed 가 아닌 낡은 상태를 모두 기동과 같이 다룬다 — state 가 building · pending · merging 이고 TryBake 가 되면 (주인이 죽었다)
building · pending 은 정리 (upper 를 trash · committed · last_attempt), merging 은 재개. 실패하면 그 노드는 10분 뒤 (답 2 와 같은 간격)

B) pending 만 더한다 — merging 은 답 2 대로 재개하고, building 은 형제에 drain 이 없어 기동과 build claim 에 둔다

C) 기동 때만 (계획 3.13 그대로) — 거짓 문장만 고치고, 사람의 수단 (그 lower 의 노드 하나를 다시 띄운다) 을 drain 문구와 운영 문서에 적는다.
조각 8 은 「다른 노드를 시작」으로 확인한다

X) Other (please describe after [Answer]: tag below)

**권장 A.** 규칙이 하나다 — 광고 주기와 기동이 같은 표를 쓴다 (규칙 12.2). 주인이 살아 있으면 TryBake 가 안 되므로 산 굽기를 건드리지
않는다. 비용은 광고마다 flock 하나다 (lower-state `BeforeAdvert` 25 µs 안). B 는 규칙이 둘로 나뉘고, C 는 pending 이 merge 대기 상한
(기본 4시간) 동안 형제를 모두 붙잡을 수 있는데 사람이 볼 신호가 정상과 같다 (drain 문구 `lower pending a merge` · 제어판 「저절로 풀린다」).

[Answer]: A

## Question 2

판정 조건 (success_when) 이 없는 굽기 계약에서 sync 나 build 가 실패하면 Run 을 어떻게 보이나? (QA S8)

build 명령의 0 아닌 exit 는 완주라 build 단계가 DONE 이다 (INVARIANTS.md:292). 산출물은 merge 단계의 「합칠 것 없음」도 DONE 으로 두었다
(계획 3.8 · 규칙 7.1). `Verify` 는 SUCCEEDED 에서 시작하므로 (`internal/store/verdict.go:128` ~ `:129`) 판정 조건 없는 굽기 계약은 sync 가
실패해도 Run 이 SUCCEEDED 다. contract-grammar 는 그런 계약을 400 이 아니라 lint 경고로 두었다 (contract-grammar 규칙 8절 끝 · ADR-061 §2
경고 먼저).

A) 합칠 것 없음은 merge 단계 FAILED — 원인 코드 없음 · error `nothing to merge; the build step left no pending upper (see its exit and builds)`.
FAILED 단계가 있으면 Run 이 판정 조건과 무관하게 FAILED 다 (`internal/store/reap.go:326` ~ `:343`). build 단계는 DONE 그대로라 명령의 exit 가
Record 에 남는다

B) 그대로 — 합칠 것 없음은 DONE (계획 3.8). 판정 조건 없는 계약은 SUCCEEDED 가 될 수 있고, lint 경고 · Record 의 build 칸 · merge 단계 로그가 알린다

C) 굽기 계약은 판정 조건을 필수로 한다 (400) — contract-grammar 가 lint 경고로 정한 것을 다시 묻는 선택지다

X) Other (please describe after [Answer]: tag below)

**권장 A.** contract-grammar 흐름 3절이 「합칠 것이 없음을 어떻게 알리나」를 bake 에 넘겼다 — A 는 그 안에서 닫고 계약 문법을 안 건드린다.
merge 단계가 할 일의 전제 (대기 upper) 가 없어 노드가 멈춘 것이라 bake_in_progress 와 같은 등급이다. 판정 조건을 건 계약은 결과가 같다
(어차피 FAILED). 대가 — 계획 3.8 의 「합칠 것 없음은 DONE」과 ADR-077 §6 되돌림 한 줄이 바뀐다.

[Answer]: B (2026-09-27T11:59:51Z 다시 답함 — 처음 답은 A 였다. A 는 정본 불변식 I3 「Run 의 성공과 실패는 계약에 선언된 조건으로만」 (INVARIANTS.md:148) 과 부딪친다 — QA 재검 D1)

## Question 3

sync 와 builds 의 명령 문자열을 무슨 셸로 돌리나? (QA S4)

계약의 sync · builds 는 문자열이다 (ADR-077 §2 · 명령을 그대로 적는다). 정본 예시는 `source build/envsetup.sh && …` 다 (ADR-077:198) —
`source` 는 bash 의 내장어다. 이 기계 (Debian 12) 의 `/bin/sh` 는 dash 이고 `sh -c 'source /dev/null'` 은 `source: not found` 로 실패했다
(`. /dev/null` 은 된다 · 2026-09-27 측정). 준비 환경 예시는 Ubuntu 계열이다 (execution-environment.md:475). INVARIANTS.md:293 은 명령 단계의
셸을 argv 배열로 정했다 — 굽기의 셸 문자열은 그와 다르므로 어느 쪽이든 정본 되돌림에 한 줄이 생긴다.

A) `bash -c` — 준비 환경의 rootfs 에 bash 가 있어야 한다. 없으면 sync 전에 노드 쪽 오류로 멈춘다
(`cannot run the bake commands: the prepared rootfs has no bash`). 정본 예시가 그대로 돈다

B) `sh -c` — POSIX 셸만. 계약은 `. build/envsetup.sh` 로 쓴다. 정본 예시와 조각 스크립트를 그 글자로 고친다

C) 계약이 셸을 적는 새 칸 — contract-grammar 의 계약 문법을 다시 묻는 선택지다

X) Other (please describe after [Answer]: tag below)

**권장 A.** 계약 작성자가 정본 예시 그대로 쓸 수 있다. bash 가 없으면 명령을 돌리기 전에 이름을 대고 멈추므로 조용히 틀리지 않는다.
B 는 예시를 본 작성자가 `source` 를 쓰면 sync 가 `source: not found` 로 실패하고 그것이 명령 실패 (DONE · exit 127) 로 보인다.
노드가 짓는 IR 대조의 고정 셸 한 줄은 어느 답이든 POSIX sh 로 쓴다 — 이 물음과 무관하다.

[Answer]: A (2026-09-27T11:59:51Z 다시 확인 — 노드가 bash 래퍼를 갖는 것은 run-contract.md:1146 과 맞다. INVARIANTS 의 argv 줄은 정본 되돌림 — QA 재검 D2)

## Question 4

물음 2 (재개의 때) 의 A 에 묶였던 셋째 갈래 — build claim 이 주인이 죽은 merging 을 만나면 재개를 배경에 열고 build 는 bake_in_progress 로
끝낸다 — 를 유지하나? (QA 참고 · ADR-077 §7 「주인이 없으면 낡은 상태 정리를 먼저 하고 진행한다」를 merging 갈래에서 바꾼다)

이 갈래는 좁은 창에서만 생긴다. pending · merging 이면 그 lower 의 모든 노드가 굽기 drain 이라 새 굽기가 매칭되지 않는다
(`lowerguard.go:212` ~ `:216`). 남는 창은 굽는 노드가 building 인 동안 (형제에 drain 이 없다) 형제에 매칭된 다른 굽기의 build claim 이,
굽는 노드가 merging 에서 죽은 뒤에 닿는 때다.

A) 유지 — 재개를 배경에 열고 build 는 bake_in_progress (답 2 그대로)

B) 정본 글자대로 — 그 자리에서 재개를 끝낸 뒤 build 를 잇는다 (배타를 기다린다 · 상한 없음)

C) bake_in_progress 로만 끝내고 재개는 열지 않는다 — 광고 주기가 다음 주기에 연다 (답 2 의 살아 있는 동안 재개)

X) Other (please describe after [Answer]: tag below)

**권장 A.** 정본 되돌림은 살아 있는 동안 재개 때문에 어차피 §7 에 생긴다 — 이 갈래가 더하는 것은 문장 하나다. B 는 드문 창에서 굽기 Run 의
임대가 형제 Run 을 기다리며 묶인다 — 물음 2 에서 B 를 권하지 않은 까닭 그대로다. C 는 A 와 결과가 같고 재개가 한 광고 주기 (기본 60초)
늦는다 — 잡은 잠금을 넘기는 코드가 하나 준다. 물음 5 의 (17) 문장은 A 일 때의 글자다.

[Answer]: A

## Question 5

답 뒤에 산출물이 정한 것 중 사용자에게 보이는 다섯을 산출물대로 두나? (QA S9 · 괄호의 번호는 audit 의 「답에 없던 설계 결정」 번호)

```text
   (11) head_tags        대조가 HEAD 와 태그를 읽었으면 배열 (태그가 없으면 []) · 못 읽었거나 대조 전이면 null        엔티티 6절 · 규칙 15절
   (8)  last_attempt     합쳐 committed 로 가면 state.json 의 last_attempt 를 지운다 — 앞의 실패는 그 Run 의 Record 에   규칙 2 · 14절
   (3)  정리 오류         metadata 를 쓴 뒤 committed 쓰기나 대기 자리 옮기기가 실패해도 merge 단계는 DONE (merged)      규칙 7절
                         — 남은 정리는 재개가 한다
   (9)  $OUT/manifest    build 단계가 올리는 이름은 manifest 하나 · 굽기가 성공하지 않았으면 명령이 쓴 manifest 를 지운다   규칙 15절
   (17) bake_in_progress 주인이 죽은 merging 을 만난 build 의 문장 — an interrupted merge of run <Run> is left on this        규칙 16.1
                         lower; this node started resuming it; submit the bake again after it finishes
```

A) 다섯 모두 산출물대로

B) 다섯 모두 되돌린다 — (11) 대조 전에도 [] · (8) last_attempt 를 남긴다 · (3) FAILED · (9) 명령이 쓴 manifest 도 올린다 · (17) 주인이 살아 있을
때와 같은 문장 하나

C) 번호마다 고른다 — [Answer]: 에 번호와 A · B 를 적는다 (예: `11A 8B 3A 9A 17A`)

X) Other (please describe after [Answer]: tag below)

**권장 A.** (11) 재지 않은 것을 「없다」로 쓰지 않는다 — 결과 확정 유닛의 교훈 (FR-1). (8) 합친 뒤에도 옛 실패가 「마지막 시도」로 보이면
lower 가 실패한 굽기 위에 있는 것처럼 읽힌다. (3) metadata 가 쓰였으면 lower 는 새 IR 에 있다 — FAILED 로 보이면 US-18 (재개로 합쳐졌나) 의
굽기 담당이 같은 굽기를 다시 낸다. (9) 명령이 쓴 manifest 가 build 의 판정 조건을 참으로 만들지 못한다. (17) US-16 (다시 내면 되는 거절) —
언제 다시 내면 되는지를 말한다. 다만 재개가 같은 오류로 되풀이되면 다시 내도 같은 거절이다 (QA 참고).

[Answer]: A
