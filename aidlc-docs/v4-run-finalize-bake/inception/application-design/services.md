# 흐름 — 누가 어떤 순서로 부르나

흐름이 여섯이다. 흐름마다 **예산이 어디서 재고 어디서 죽을 수 있나**를 함께 적는다. 순서의
세부와 경계값은 Functional Design 이 닫는다.

- **작성 시각**: 2026-09-24T06:33:21Z · 답 일곱(전부 A)을 딛는다

```text
   1   명령 단계          오늘 경로를 좁히고 종료 보고와 예산을 더한다
   2   굽기 build 단계     새로.  upper 에 짓고 대기 자리로
   3   굽기 merge 단계     새로.  형제를 기다려 lower 에 합친다
   4   노드 기동           낡은 상태 정리 · 끊긴 합치기 재개 · 삭제자 첫 회
   5   광고 주기           drain 합성 · 후보 잠금 · 광고 키 · 상태 파일
   6   Mediator           종료 수락 · merge 의 waiting · QUEUED 의 사유
```

---

## 1. 명령 단계 (effect 기본 build)

```text
   claim                     Held.Add.  후보 잠금은 이미 쥐어져 있다 — 역할만 Run 으로 (Q1)
   준비 · 세션 열기             오늘 그대로.  Open 실패 갈래는 runRoot 를 trash 로 (FR-4)
   명령                       session.Run
   종료 status 확정            여기부터 phase 가 finalizing 이다
     exited 를 보낸다          goroutine.  실패하면 Finalize 와 나란히 재시도.  Finalize 를 안 막는다
   Finalize                   [Finalize 예산 시작]  $OUT 의 named output · collect · 지목 경로 stat ·
                              (edit) changeset · (produce) producer adapter · (discover) bounded 걷기 ·
                              진단 (Q5).  build/test 면 워크스페이스를 걷지 않는다 (FR-1)
   닫기                       unmount.  정책이 요구하면 받아들임(statfs · 보존 총량) -> upper 를 spool 로.
                              나머지 runRoot 를 trash 로.  둘 다 rename 한 번  [Finalize 예산 끝]
   업로드                      [업로드 예산 시작]  named output 과 로그.  업로드 client · 스트림  [끝]
   보고                       result — exited_at · finalized_at · finalize · upload · reason ·
                              diagnostics · checkpoint_capture.  임대가 죽을 때까지 재시도 (오늘 규칙)
   보고 뒤                     삭제자 Kick · Checkpoint Store Settle.  둘 다 임대 밖
```

**Finalize 예산이 닫기까지 덮는다.** capture 의 남은 예산은 Finalize 예산의 남은 몫이다
(ADR-075 §10.2 · ADR-076 §4). 닫기가 rename 뿐이라 예산 안에 든다.

**죽을 수 있는 자리.**

```text
   Finalize 예산 초과     reason finalize_timeout · finalize=timeout.  닫기와 보고는 계속한다
   업로드 예산 초과       reason upload_timeout · upload=timeout.  못 올린 이름은 produced 에 없다
   Finalize 중 노드 죽음   Mediator 에 phase=finalizing 과 exit 가 남는다.  ADR-030 이 Run 을 닫고
                         Record 에 명령은 끝났고 Finalize 는 못 끝났다는 사실이 exit 와 함께 남는다
   보고 전 노드 죽음       trash 와 spool 의 미완료가 남는다.  기동이 치운다 (4절)
```

agent 단계는 오늘 그대로다 — 수확을 안 좁히고(FR-1) exited 는 하네스 종료에서 보낸다.
진단 칸과 예산은 같이 받는다.

---

## 2. 굽기 build 단계 (effect: prepare)

```text
   claim                     LowerGuard.OnClaim — prepare 면 후보 잠금을 놓는다 (결정 3-9)
   굽기 잠금                   TryBake.  못 잡으면(주인이 살아 있음) 곧바로 reason bake_in_progress 로 보고
                              잡았는데 상태가 committed 가 아니면 낡은 상태다 — 정리하고 간다 (결정 3-12)
   상태 building              state.json — 주인(Run · 단계 · 노드 · 인스턴스)
   세션 열기                   평범한 overlay 단계와 같다.  lower 는 읽기 전용 (FR-6)
   sync                       sh -c 로.  시작 · 끝 시각과 exit 를 적는다
   builds                     적힌 순서대로.  항목마다 같은 기록
     하나라도 0 이 아니면       upper 를 trash 로 · last_attempt · 상태 committed · 굽기 잠금 놓음 ·
                              build 단계 실패로 보고.  합치지 않는다 (결정 3-18)
   IR 유도                    세션 안에서 manifest HEAD 에 정확히 붙은 태그를 ir_tag 로 가린다 (Q7).
                              하나면 그것 · 0 이면 null(no_ir_tag) · 둘 이상이면 null(ambiguous_ir_tag)
   pinned manifest            repo manifest -r 을 워크스페이스에 떠 둔다 — upper 에 남아 합쳐진다
   Finalize                   결과는 파일 목록이 아니라 build manifest 다 (ADR-075 §5 의 prepare 줄)
   닫기                       Keep{Upper: 대기 자리} — upper 만 <scratch>/pending/<run> 으로 rename.
                              나머지 runRoot 는 trash 로
   상태 pending               state.json 에 대기 upper 의 자리.  굽기 잠금은 쥔 채다
   보고                       build 단계 DONE.  build manifest 를 싣는다
```

**pending 이 되는 순간 형제가 drain 을 싣기 시작한다** (5절). merge 단계는 Mediator 가 다음
단계로 만든다.

---

## 3. 굽기 merge 단계

```text
   claim                     Mediator 가 phase 를 waiting 으로 적는다 (ADR-075 §10.3)
   상태 확인                   pending 이고 주인이 이 Run 인가.  아니면 실패로 보고
   배타 잠금 기다림             Exclusive(ctx 마감 = 받은 시각 + merge.wait).  기다리는 동안 주기마다
                              쥔 사람 기록을 읽어 단계 로그에 한 줄씩 쓴다 (Q4)
                                waiting for the lower lock; deadline <시각> (<남은 시간> left)
                                  node <노드>  run <Run>  holding since <시각>
                                  node <노드>  candidate  drain not acknowledged yet
   마감                       upper 를 trash 로 · 상태 committed · 굽기 잠금 놓음 · drain 풀림 ·
                              reason merge_wait_timeout 으로 보고 (결정 3-11)
   잡았다                      형제가 전부 후보에서 빠졌고 도는 Run 이 없다 (Q1)
   상태 merging               state.json.  이 순간부터 끊기면 재개 대상이다
   시작 전 확인                merge.Preflight — 같은 filesystem · metacopy · redirect.
                              마운트 0 은 쥔 배타 잠금이 증거다
   합치기                      merge-helper (namespace 안) -> merge.Apply.  lower 에서 없앨 것은 trash 로
   metadata                  합치기의 마지막 동작.  bake.run · merged_at · resumed=false · previous_ir
   상태 committed             대기 upper 자리를 지우고 굽기 잠금 · 배타 잠금 놓음
   보고                       merge 단계 DONE.  ir · ir_reason · 셈을 싣는다
   보고 뒤                     삭제자 Kick
```

**합치기 본체에는 상한이 없다** (결정 3-11). 끊으면 lower 가 merging 에 묶인다.

**Finalize 예산이 안 걸린다.** 기다림은 merge 단계 안의 일이고 Record 에 남는다 (결정 3-8).

---

## 4. 노드 기동

runc-overlay 노드만 이 흐름을 탄다. native 노드는 scratch 가 없어 삭제자도 checkpoint 도 없고,
lower 가 없어 후보 잠금도 없다.

```text
   1   상태 자리 열기            lower.Open(워크스페이스).  못 열면 노드가 drain (components 5절)
   2   굽기 잠금 시도            TryBake.  잡았고 상태가 committed 가 아니면 주인이 죽은 낡은 상태다
         building · pending    대기 upper 를 trash 로 · 상태 committed
         merging               재개 — 배타 잠금을 배경에서 기다려 merge.Apply 를 남은 upper 에 다시 돌린다.
                               metadata 에 resumed=true 와 원래 Run.  광고 루프를 안 막는다
       못 잡으면               다른 노드가 잡았다.  그쪽이 정리한다
   3   삭제자 첫 회              남은 trash 를 비운다.  env check 의 smoke 가 남긴 것도 여기서 (FR-4)
   4   checkpoint 조정          미완료 capture 와 만료 항목 (FR-10)
   5   광고 시작                5절.  첫 광고 전에 후보 잠금을 시도한다
```

**재개는 굽기 잠금을 먼저 잡은 하나가 한다** (결정 3-13). 그동안 상태가 merging 이라 형제는
drain 을 싣고, 재개가 배타 잠금을 기다리는 동안 형제의 도는 Run 이 끝난다.

---

## 5. 광고 주기

```text
   정책 파일 읽기            소유자 정책 (오늘 그대로)
   스스로의 사유             statfs 여유 < min_free_gb -> disk
                           lower 상태가 pending · merging -> bake
                           공유 잠금을 못 잡음(합치기 중) -> bake
   drain 합성               LowerGuard.BeforeAdvert — 셋 중 센 쪽 (at-boundary > graceful > 없음)
   후보 잠금                 실을 drain 이 없으면 공유 잠금을 쥔다 (Q1)
   광고 키                  workspace.writes · ir · repo.built.<name> · bake.run · bake.resumed ·
                           producer.<name>.  ir 과 bake 는 metadata 에서, arch 는 툴체인만 보고 (FR-4)
   광고                     POST /v1/nodes
   응답                     LowerGuard.AfterResponse — drain 이 받아 적혔고 임대가 없으면 놓는다
                           (놓는 울타리는 Functional Design).  임대가 있으면 Run 역할로 쥔 채
   상태 파일                 drain 의 출처 · scratch 의 양이 바뀌었으면 쓴다 (Q3)
```

**drain 을 푸는 광고의 순서가 정해져 있다** (Q1 답). 합치기가 끝나면 공유 잠금을 다시
잡고 · metadata 에서 새 ir 을 읽고 · drain 을 푸는 그 광고에 새 ir 을 싣는다. Mediator 는
drain 없는 광고로만 매칭하므로 낡은 ir 로 매칭할 틈이 없다.

**여유 부족 drain 은 삭제자와 맞물린다.** trash 가 쌓이면 여유가 내려가 drain 이 걸리고,
삭제자가 비우면 다음 광고에서 풀린다 (FR-4).

---

## 6. Mediator

```text
   POST .../exited          인증(오늘의 bearer) -> 본문 -> store.MarkExited
                              노드 · claimed_instance · attempt 가 맞고 CLAIMED   phase=finalizing ·
                                                                                phase_since=exited_at · exit
                              같은 키의 재전송 · 종결된 단계                       200.  아무것도 안 바뀐다
                              인스턴스가 다르다                                   거절 (FR-2 · 5.3)
                            판정 · 다음 단계 · 정산 · 임대 해제를 하지 않는다 (결정 1-7 · 1-8)
   claim                    merge 종류면 phase=waiting, 나머지는 running
   POST .../result          오늘 경로.  새 칸을 봉인한다.  종결 전이는 이것 하나다
   GET /v1/runs/{id}        단계마다 phase · phase_since · exit.
                            QUEUED 면 요구 줄마다 후보 셋 — match.Match 를 점유 집합과 drain 집합으로
                            따로 두 번 부른다 (완료 조건 4 · internal/match 불변)
```

**result 의 대조는 바꾸지 않는다** — 노드만 본다. 종료 보고와의 비대칭은 잔여다 (5.3).
