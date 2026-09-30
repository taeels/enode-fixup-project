# `checkpoint` — 규칙

칸과 타입은 `domain-entities.md` (아래 「엔티티 N절」), 흐름은 `business-logic-model.md` (「흐름 N절」) 에 있다. **답 N** 은 계획
물음 N 의 답 (모두 A) 이다. 문구 초안 가운데 영어는 CLI · 로그 · 오류 · diagnostics 이고 (CONVENTIONS 2.1), 한국어는 제어판 화면
안의 글이다 (오늘 제어판과 같다 · `panel/page.go:275`).

---

## 1. 보존을 요구하는가 (답 1 · 2 · 3)

노드는 Run 의 성공을 모른다 (불변식 I3 — 성공은 계약의 조건이 정한다). 아래는 보존 여부만 정하고 결과에 안 들어간다.

| 단계의 끝 | off | on-failure | always |
|---|---|---|---|
| 명령 단계 — exit 0 · 산출물이 다 있음 · Finalize 오류 없음 | 안 함 | 안 함 | 요구 |
| 명령 단계 — exit ≠ 0 · signal · 빠진 산출물 (`diagnostics.missing`) · Finalize 오류 가운데 하나 | 안 함 | 요구 | 요구 |
| agent 단계 — 하네스 완주 · 산출물이 다 있음 · Finalize 오류 없음 | 안 함 | 안 함 | 요구 |
| agent 단계 — 하네스 미완주 · 빠진 산출물 · Finalize 오류 가운데 하나 | 안 함 | 요구 | 요구 |
| 굽기 build — `failEnd` 넷 (명령 실패 · IR 어긋남 · IR 대조를 못 함 · pinned 실패) | 안 함 | 요구 | 요구 |
| 굽기 build — 성공 끝 (upper 가 대기 자리로 간다 · 그 뒤 버려져도) | 안 함 | 안 함 | 안 함 |

- **굽기 build 는 위 표의 굽기 두 줄만 따른다** (답 3 이 답 1 보다 좁은 자리를 정한다). 성공 끝의 upper 는 merge 단계의 것이라 `always` 여도
  옮기지 않는다. 성공 끝이 Close 뒤에 버려지는 갈래 (Finalize 오류 · 마감 · 초안 쓰기 실패 · `bake_build.go:449-466`) 는 upper 가
  이미 대기 자리에 있다 — 답 3 의 「대기 자리 뒤」와 같아 보존하지 않는다
- **Finalize 오류에는 마감도 든다.** 그때는 요구하되 2절 ③ 에서 `rejected`(`lease_budget`) 로 닫힌다
- 요구하지 않았으면 `not_requested` 다. native 노드도 같다 (답 2)

---

## 2. 판정 차례와 전이 (계획 3절 1번 · ADR-076 §2 표)

| 차례 | 보는 것 | 걸리면 | `diagnostics.checkpoint` 문장 (초안) |
|---|---|---|---|
| ① | 1절이 요구하나 | `not_requested` | 없음 |
| ② | 런타임의 `Capture.Supported` | `unsupported`(`runtime`) | `this node runs steps without an isolated upper; there is nothing to keep` |
| ② | `<scratch>` 와 `<scratch>/spool` 의 filesystem 번호 (spool 은 lstat — symlink 면 따라가지 않고 ⑤ 가 거절한다 · NFR 답 3 으로 고침) | `unsupported`(`cross_filesystem`) | `the spool is on another filesystem than the step's upper; the upper is never copied` |
| ③ | Finalize 가 마감으로 끝났거나 지금이 마감 뒤 | `rejected`(`lease_budget`) | `the finalize budget ran out before the upper could be moved into the spool` |
| ④ | statfs 여유 < `min_free_gb` | `rejected`(`free_space`) | `free space 8.0 GiB is below min_free_gb 10` |
| ④ | 보존 총량 ≥ 몫 (5절) | `rejected`(`quota`) | `kept checkpoints already use 41.2 GiB, capacity_percent 20 of kept plus free space` |
| ⑤ | 세션을 열 때의 예약 (항목 폴더 · 잠금 · `reserved` 기록) 이 실패했다 — 그때의 errno 를 들고 있다가 여기서 알린다 (NFR Design 답 1 로 고침) | `failed`(`io`) | `reserving a place in the spool failed: <errno>` |
| ⑤ | spool 자리를 거절했다 — symlink · 디렉터리 아님 · 남의 것 (11절 · NFR 답 3 으로 고침) | `failed`(`io`) | `the spool is not a private directory of this node; nothing is kept` |
| ⑤ | `Close(Keep{Upper, By, Result})` 뒤 `KeepResult.Late` | `rejected`(`lease_budget`) | ③ 과 같다 |
| ⑤ | `KeepResult.Err` | `failed`(`io`) | `moving the upper into the spool failed: <errno>` · abort 면 `the session was aborted before the upper could be kept` |
| ⑥ | 옮긴 뒤 지금이 마감 뒤 | `failed`(`lease_budget`) | `the finalize budget ran out after the upper was moved; the checkpoint was discarded` |
| ⑥ | `kept` 기록 쓰기가 실패 | `failed`(`io`) | `writing the checkpoint record failed: <errno>` |
| ⑦ | 확정 | `captured` | 없음 |

- **④ 는 여유를 먼저 본다** — 노드 전체의 여유를 보존 몫보다 먼저 지킨다
- **⑤ · ⑥ 에서 실패한 예약은 그 자리에서 버린다** — 항목 폴더째 `Trash.Move` 한 번 (상수 시간) 하고 잠금을 놓는다. 버리기도
  실패하면 `reserved` 로 남아 9절 조정이 거둔다 (ADR-076 §4 「spool 미완료 항목은 store 가 정리한다」)
- **예약은 세션을 열 때 한다** (NFR Design 답 1 로 고침) — `claim.go:712` · `bake_build.go:196` 뒤. 정책이 off 가 아니고 runtime 이
  지원하고 spool 을 받아들인 노드에서 단계마다. 판정의 창에는 ① ~ ④ 의 stat · statfs · 요약 읽기와 Close 안의 rename 만 남고, ⑥ ⑦ 의
  확정은 `closedAt` 뒤다. 요구하지 않았거나 칸이 없는 끝 (4절) 의 예약은 보고 뒤 버린다. 요구하는지는 여전히 닫을 때 1절이 정한다
  (`nfr-design-patterns.md` 1절)
- **`<errno>` 는 오류의 뜻만이다** — `syscall.Errno` 의 글 (`file exists` 처럼). `PathError` · `LinkError` 의 문장은 경로를 담으므로
  쓰지 않는다 (11절)

---

## 3. keep 의 실패와 Close 의 오류 (계획 3절 2번 · ADR-076 §4)

ADR-076 §4 의 두 문장이 함께 성립해야 한다 — 「capture 가 실패해도 exit status 는 바뀌지 않는다」와 「Close 오류를 무시해 성공으로
봉인하지 않는 기존 수명 규칙은 유지한다」.

- **떼는 것은 spool 로 옮기기 하나다.** `Keep.Result` 가 있으면 runtime 의 Close 는 옮기기의 결과 (옮김 · 늦음 · 실패) 를 그
  칸에 적고, 자기가 돌려주는 오류에 싣지 않는다. 그 결과가 2절 ⑤ 의 상태다
- **나머지 Close 오류는 오늘 그대로다** — helper 의 close 답 오류 · helper 가 안 끝남 · unmount · 작업 폴더를 trash 로 옮기기 ·
  잠금. finalize 칸 error 와 error 문장 `runtime cleanup: …` 이 되어 단계가 완주가 아니다 (`finalize.go:230-235`)
- **keep 이 실패한 뒤 upper 는 작업 폴더와 함께 trash 로 간다.** 그 옮기기가 실패하면 그것은 Close 오류다 — 보존의 실패가 아니다
- **굽기 대기 자리의 keep 은 오늘 그대로다** — `Keep.Result` 가 없고 실패는 Close 오류다 (`runc_overlay_linux.go:588-593`)
- keep 은 첫 Close 에서만 본다 (`runtime.go:253-260`) — 뒤의 안전망 Close 는 spool 로 옮긴 upper 를 건드리지 않는다

---

## 4. 칸이 있는 때 (계획 3절 3번)

- 세션을 열고 closeOut 에 닿은 단계만 `checkpoint_capture` 를 싣는다. 명령 단계 · agent 단계 · 굽기 build 의 `failEnd` 와 성공 끝
- 칸이 없다 — 임대 만료 · 실행 실패로 Finalize 없이 닫는 끝 (`claim.go:777-780` · `:1082`) · 하네스 앞의 거절 · runtime open 실패 ·
  굽기 build 의 `stopEarly` · merge 단계. 칸이 없는 것이 곧 그 구간에 닿지 않았다는 뜻이다
- `Worker.Checkpoints` 가 nil 이면 (옛 조립 · 시험) 칸을 싣지 않는다
- `enode env check` 의 smoke 는 단계가 아니다 — 보존하지 않는다 (FR-4 · smoke 는 trash). 보존 지원은 광고 · 매칭 어휘에 안 오른다
  (ADR-076 §5 — capability 는 새 매칭 속성이 아니다)

---

## 5. 받아들임 — 보고 전 (계획 3절 4번 · 결정 2-7 · 2-10)

- **보는 것은 둘이다.** statfs 의 여유 (`freeBytes` · `disk_unix.go:7`) 와 spool 의 요약 `usage.json` 의 보존 총량 (닫을 때 읽는다 ·
  잠금 없이 · 없거나 못 읽으면 0). 둘 다 upper 크기와 무관하다 (NFR Design D3 로 고침)
- **보존 총량** = `kept` 항목의 측정한 바이트의 합. 어느 노드의 판정이든 spool 전체를 읽고 요약을 쓰므로 형제의 측정한 보존본이
  든다. 측정 전 항목은 0 으로 센다 — 받아들임은 그만큼 느슨하고 6절이 정확히 막는다 (NFR Design D3 로 고침)
- **몫** = `capacity_percent` / 100 x (보존 총량 + 여유)
- 예약은 spool 잠금을 쥐지 않는다 — 보고 뒤 판정이 측정하는 동안 (몇 초) 보고 전 창이 기다리지 않게 한다. 항목 잠금만 쥔다

---

## 6. 보고 뒤 판정과 퇴출 (계획 3절 5번 · 답 7 · 결정 2-8)

차례대로 한다. 보고 전 창과 임대 밖이다. spool 잠금은 1 (읽기) 과 3 ~ 7 (적기) 에만 쥐고 2 의 측정은 잠금 밖이다 — 적기 전에 기록을
다시 읽어 그새 형제가 퇴출하거나 만료한 항목의 측정값은 버린다 (NFR Design D2 로 고침).

| 차례 | 하는 일 |
|---|---|
| 1 | 기록을 다 읽는다. 만료 (지금 ≥ `expires_at`) 인 `kept` · `evicted` 는 항목 폴더째 `Trash.Move` |
| 2 | 측정 전인 `kept` 를 오래된 것부터 helper 안에서 걷는다 — 바이트 (블록 수 x 512) 와 항목 수. `Measure` 의 경계 그대로 (symlink 를 안 따라가고 다른 filesystem 에 안 들어간다) |
| 3 | 하나가 `max_gb` 를 넘으면 그 `upper/` 를 `Trash.Move` · 기록은 `evicted` · 사유 `larger than max_gb` |
| 4 | 몫을 한 번 정한다 — 바이트는 `capacity_percent` x (보존 총량 + 여유), inode 는 10절의 한도. 퇴출한 것은 여유로 돌아올 것이므로 몫은 판정 동안 그대로다 (NFR 답 2 로 고침) |
| 5 | 보존 총량 > 몫이면 오래된 `kept` 부터 퇴출 · 사유 `over capacity_percent`. 새 것도 예외가 아니다 |
| 6 | inode 합 > inode 한도 (10절) 면 오래된 `kept` 부터 퇴출 · 사유 `over max_total_inodes`. statfs 의 전체 inode 가 0 이면 이 차례를 건너뛴다 (NFR 답 2 로 고침) |
| 7 | 요약 `usage.json` (`kept` 의 바이트 합 · inode 합 · 시각 · 0600 · fsync 없음) 과 상태 파일의 spool 칸을 쓴다. trash 에 넣었으면 삭제자를 깨운다 (NFR Design D3 로 고침) |

- **측정이 실패하면** 그 항목은 측정 전으로 남고 다음 판정에서 다시 걷는다. helper 를 못 띄우면 (`LaunchError`) 그 판정은 멈추고
  다음 때를 기다린다 — 삭제자와 같은 규칙 (trash `business-rules.md` 4.3)
- **receipt 는 안 바뀐다.** 퇴출의 사유는 store 기록에만 있다 (ADR-076 §2 끝 · §6)
- **inode 한도는 보존본 전체에 건다** (ADR-076 §5). 하나의 상한은 바이트뿐이다

**판정의 때** — 결과 보고 뒤 (`Worker.AfterReport`) · 기동의 조정 뒤 · 10분마다 (만료를 거두려고 · 삭제자의 `DefaultEvery` 와 같은 값).
배경의 고루틴 하나가 돈다. 판정 중에 새 깸이 오면 끝난 뒤 한 번 더 돈다.

---

## 7. TTL 과 기록의 끝 (계획 3절 6번 · 답 7)

- `expires_at` = 확정한 시각 + 그때의 `ttl_hours`. 설정을 뒤에 바꿔도 이미 잡은 것의 만료는 그대로다 — receipt 의 값이 사실로 남는다
- `evicted` 의 기록은 원래 `expires_at` 까지 남고, 그때 항목 폴더째 trash 로 간다
- 만료된 항목은 기록째 없다. `show` 는 모르는 ID 와 같게 답한다 (13절)

---

## 8. 보고의 성패 (답 10 · ADR-076 §4 끝)

- `Worker.report` 가 끝나면 그 끝을 기록의 `report` 에 적는다 — `delivered` · `rejected` · `lease_ended` · `unknown` (엔티티 3절)
- **어느 끝이든 TTL 까지 남긴다.** ADR-076 §4 끝의 「정책에 따라 만료 · 정리」를 TTL 로 읽는다 (정본 되돌림 · 흐름 12절)
- 기록 쓰기가 실패하면 노드 로그에 warn 한 줄 · 기록은 그대로 둔다 (조정이 `unknown` 으로 채운다)

---

## 9. 재시작 조정 (계획 3절 7번 · ADR-076 §2 끝 · §5 끝)

기동 차례 — 기동 청소 → 삭제자 (배경) → **조정** → 광고 (`services.md` 4절). 조정은 spool 잠금을 쥐고 돈다.

| 찾은 것 | 하는 일 |
|---|---|
| ID 모양 폴더에 잠금 파일조차 없다 | 만든 지 1시간이 지났을 때만 아래 줄과 같이 다룬다 — 예약의 mkdir 와 잠금 사이를 비킨다 (trash 규칙 3절 선례 · NFR R2 로 고침) |
| ID 모양 폴더이고 기록이 없거나 · 비었거나 · 읽을 수 없거나 (11절 · 1 MiB 넘음 포함) · `reserved` 이며 항목 잠금을 쥘 수 있다 | 주인이 없는 미완료다 — 전원이 나간 뒤에도 생긴다 (기록에 fsync 가 없다 · 흐름 1절). 항목 폴더째 trash (NFR R2 로 고침) |
| 위와 같은데 항목 잠금을 못 쥔다 | 형제가 쓰는 중이다. 건드리지 않는다 |
| `kept` · `evicted` 이고 만료 | 항목 폴더째 trash |
| `kept` 이고 `report` 가 비었다 | `unknown` 으로 적는다 |
| 이름이 ID 모양이 아닌 폴더 | 건드리지 않고 warn 한 줄 (사람이 둔 것일 수 있다 · 읽을 수 없는 기록은 위 줄로 옮겼다 · NFR R2 로 고침) |

조정 뒤 곧바로 보고 뒤 판정 (6절) 을 한 번 돈다 — 측정 전 항목을 걷는다. 소유자가 항목 폴더를 trash 로 옮겼으면 (답 6) 폴더도 기록도
없으므로 할 일이 없다.

---

## 10. 설정 (답 4 · 계획 3절 8번)

| 키 | 기본 | 받는 값 | 틀리면 (노드가 안 뜬다 · `config.go:161` 선례) |
|---|---|---|---|
| `policy` | `on-failure` | `off` · `on-failure` · `always` | `checkpoint: policy "X" is not off, on-failure or always` |
| `ttl_hours` | 48 | 1 이상 | `checkpoint: ttl_hours must be at least 1` |
| `capacity_percent` | 20 | 1 ~ 100 | `checkpoint: capacity_percent must be between 1 and 100` |
| `max_gb` | 32 (NFR 답 1 로 고침) | 1 이상 | `checkpoint: max_gb must be at least 1` |
| `max_total_inodes` | 몫 — 아래 (NFR 답 2 로 고침) | 1 이상 | `checkpoint: max_total_inodes must be at least 1` |

- 블록이 없거나 값이 0 이면 기본값이다 (`min_free_gb` 선례 · `config.go:141`). 보존을 끄는 길은 `policy: off` 하나다
- **inode 한도의 기본은 몫이다** — `capacity_percent` x (`kept` 의 측정한 inode 합 + statfs 의 남은 inode). statfs 의 전체 inode 가 0 인
  filesystem (btrfs) 에서는 보지 않는다. 숫자를 적으면 그 숫자다 (NFR 답 2 로 고침 · `nfr-requirements.md` 1절)
- `max_gb` 의 기본 32 는 측정한 가장 큰 upper (처음부터 굽기 23 GB · ADR-077 §12) 를 받는다 (NFR 답 1 로 고침)
- 음수는 틀린 값이다. 문장의 머리 `config <path>: ` 는 오늘 `LoadLocal` 이 붙인다
- 계약과 agent 출력에는 칸이 없다 (결정 2-11). native 노드도 블록을 읽는다 — 상태 파일에 정책을 보이려고 (12절)

---

## 11. spool 과 보안 (계획 3절 10 · 11번 · `requirements.md` 5.3)

- spool 과 항목 폴더는 0700, 기록은 0600 — 노드 uid 만 읽는다
- **spool 자리를 기동 때 확인한다** (NFR 답 3 으로 고침) — lstat 로 본다. 없으면 0700 으로 만든다. 남에게 열린 비트 (0o077) 는
  fchmod 0700 으로 좁히고 로그 한 줄. symlink · 디렉터리 아님 · 주인이 노드 uid 가 아님은 고치지 않고 거절한다 — 요구한 단계는 2절
  ⑤ 의 `failed`(`io`) 와 문장, 기동 로그 warn 한 줄. 노드는 뜨고 drain 하지 않는다 (lower-state 상태 자리의 규칙과 같다 ·
  `lower-state-code-generation-plan.md:145-162`). 예약은 spool 을 `O_NOFOLLOW` 로 연다 — 기동 뒤에 바뀐 자리도 거절한다
- 기록은 `O_NOFOLLOW` · 보통 파일 · 1 MiB 까지만 읽는다. 넘으면 읽을 수 없는 기록이다 (9절 · NFR C7 로 고침)
- host 경로가 나가지 않는 자리 — receipt · `diagnostics.checkpoint` · 단계 로그 (Mediator 로 올라간다) · 광고. 나가는 자리는
  `enode checkpoint show` 와 `list --json` (소유자 화면) 과 노드 로그 (기계 안) 뿐이다
- 보존 중 암호화는 하지 않는다 (결정 2-9). TTL 에 지운다 (7절)
- 신원 (lower · environment · head · ir) 은 옮기기 앞에 정한다 (ADR-076 §4 의 3). lower metadata 는 읽기만 한다 — capture 중 lower 를
  옮기거나 만지지 않는다 (ADR-076 §5)

---

## 12. 보이는 자리 (답 5 · 완료 조건 2 · 3 ③ · US-5)

**상태 파일** — 엔티티 7절의 칸. 기동에 `checkpoint` 블록을 한 번 쓰고, spool 칸은 판정마다 바뀌면 쓴다.

**제어판** (한국어 · 오늘의 trash 줄 아래 · `page.go:273-279` 의 모양):

| 조건 | 문구 초안 |
|---|---|
| spool 줄 (scratch 가 있을 때) | `보존 3 개 · spool 4.2 GiB (1 개는 크기 모름) · 측정 13:02:15` |
| 정책 줄 (on-failure · always) | `보존 정책 on-failure · 48시간 · (보존 + 여유) 의 20% — 바꾸려면 설정 파일의 checkpoint: 블록` |
| 안내 줄 (on-failure · always) | `실패한 단계의 upper 가 이 기계에 남는다. 도구가 쓴 자격증명 캐시가 들어 있을 수 있다. 여는 법은 enode checkpoint show <ID>` |
| off | `보존 정책 off — 단계의 upper 를 남기지 않는다. 켜려면 설정 파일의 checkpoint: 블록` |
| native (`unsupported: runtime`) | `이 노드는 단계를 격리 없이 돌려 보존하지 않는다 (runtime)` |
| 상태 파일에 블록이 없다 | 줄 없음 (데몬이 아직 안 썼다) |

**데몬 기동 로그** (영어 · info 한 줄):

- overlay — `checkpoint policy on-failure, ttl 48h, capacity 20 percent of kept plus free space; change it under checkpoint: in <config>`
- off — `checkpoint policy off; failed steps leave nothing behind`
- native — `checkpoint: this node runs steps without an isolated upper and keeps nothing (runtime)`

---

## 13. 조회 — `enode checkpoint list | show <ID>` (답 6 · 7 · 8 · 계획 3절 13번)

- 입구 `enode checkpoint list|show [ID] --config PATH [--json]` — `enode env` 와 같다 (`environment.go:21`). 설정에서 scratch 를 찾는다
- 기록만 읽는다. spool 잠금을 쥐지 않는다. 오래된 것부터 — 확정한 시각, `reserved` 는 기록을 만든 시각으로 줄 세운다

**`list`** — 한 줄에 하나 (초안):

```text
ID            STATE                          RUN      STEP        CAPTURED              EXPIRES               SIZE        REPORT
3f9a1c0b7d2e  kept                           r-7a1c   3 build     2026-09-30T13:02:11Z  2026-10-02T13:02:11Z  4.2 GiB     delivered
7a01be55c9d3  evicted: larger than max_gb    r-7a1c   5 test      2026-09-30T14:10:40Z  2026-10-02T14:10:40Z  31.0 GiB    lease_ended
c41e09aa7b21  kept                           r-81f0   2 agent     2026-09-30T15:00:02Z  2026-10-02T15:00:02Z  unmeasured  unknown
```

항목 잠금이 쥐어진 `reserved` (도는 단계의 예약) 는 목록에서 뺀다 — 잠금을 `LOCK_NB` 로 시험만 한다. 주인이 없는 `reserved` 만 STATE
`incomplete` 다. `show` 도 같다 (NFR Design 답 1 로 고침). 비었으면 `no checkpoints on this node`. scratch 가 없는 노드 (native) 는
`this node has no scratch; checkpoints need the runc-overlay runtime` — 둘 다 exit 0.

**`show <ID>`** — 칸마다 한 줄 (초안):

```text
id           3f9a1c0b7d2e
state        kept
node         <node id>
run          <run id>   step 3 build   attempt 1
runtime      runc-overlay (overlay-upper)
scope        workspace-upper, inspect-only
lower        <lower identity>
environment  <prepared environment>
head         <commit>   ir <IR>
captured     2026-09-30T13:02:11Z
expires      2026-10-02T13:02:11Z
size         4.2 GiB, 157982 entries (measured 2026-09-30T13:02:15Z)
report       delivered
path         <scratch>/spool/3f9a1c0b7d2e/upper
open         entries written as the container root belong to subordinate uids; read them inside
             unshare --user --map-root-user --map-auto (the same mapping the helpers use)
discard      move <scratch>/spool/3f9a1c0b7d2e into <scratch>/trash/; the background deleter removes it
```

- `evicted` 면 path · open · discard 대신 `evicted     larger than max_gb at <시각>; the tree is gone and this record stays until <expires>`
- 기록에 없는 칸 (head · ir · 측정 전 크기) 은 줄째 뺀다. 측정 전이면 `size        unmeasured`
- 모르는 ID — stderr `no checkpoint <ID> on this node: it expired or was never captured here` · exit 1 (답 7)
- `--json` — `list` 는 기록의 배열, `show` 는 기록 하나. 둘 다 `path` 칸을 더한다 (엔티티 3절)

---

## 14. 노드 로그 (영어)

| 때 | 수준 · 문구 · 칸 |
|---|---|
| 확정 | info `checkpoint captured` — id · run · step · expires_at |
| 요구했는데 못 함 | info `checkpoint not captured` — state · reason · detail |
| 퇴출 | info `checkpoint evicted` — id · reason |
| 만료 | info `checkpoint expired` — id |
| 판정 · 조정의 오류 | warn `checkpoint settle failed` · `checkpoint reconcile failed` — err (경로를 담아도 된다 · 기계 안) |
| 조정이 남긴 폴더 | warn `unknown entry left in the spool` — name |
| 보고의 성패 쓰기 실패 | warn `cannot record the report outcome of a checkpoint` — id · err |
| 기동 · spool 을 좁힘 | info `checkpoint spool permissions narrowed to 0700` — mode (NFR 답 3 으로 고침) |
| 기동 · spool 을 거절 | warn `the checkpoint spool is not a private directory of this node; nothing will be kept` — why (`symlink` · `not a directory` · `owned by uid N`) (NFR 답 3 으로 고침) |
