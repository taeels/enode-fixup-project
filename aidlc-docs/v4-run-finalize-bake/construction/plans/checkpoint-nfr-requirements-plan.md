# `checkpoint` — NFR Requirements 계획

**유닛** `checkpoint` (실패한 단계 보존) · **브랜치** `unit/checkpoint` · **앞 단계** Functional Design (승인 · `e6ec0d3`) ·
**다음** NFR Design (유닛 정의 8절)

- 작성 시각 2026-09-30T14:13:30Z · 규칙 `.aidlc/aidlc-rules/aws-aidlc-rule-details/construction/nfr-requirements.md` Step 1 ~ 4
- 범위 — 유닛 정의 8절의 NFR 셋 (N2 · 보안 · 성능) 과 FD 흐름 11절의 넘김. FD 문서는 `construction/checkpoint/functional-design/`
  이고 아래에서 「엔티티 · 규칙 · 흐름 N절」로 가리킨다
- N2 는 NFR 값 둘째 — `max_gb` (보존본 하나의 바이트 상한) 과 `max_total_inodes` (보존본 전체의 inode 한도) 의 기본값이다

---

## 1. 받는 일

| # | 일 | 출처 |
|---|---|---|
| 1 | N2 — `max_gb` · `max_total_inodes` 의 기본값 | `unit-of-work.md:333` · 규칙 10절 · ADR-076 §10 |
| 2 | 보안 — 0700 · 0600 · host 경로가 나가는 자리 셋 · `unshare` 안내 · 자격증명 캐시 안내 | 규칙 11 · 12절 · `requirements.md` 5.3 |
| 3 | 성능 — 보고 전 창에 더한 일 (stat 둘 · statfs · 예약 · 기록 쓰기 · rename) 의 시간과 Finalize 예산 (기본 1분) 안의 몫 | 흐름 1절 · `requirements.md` 5.4 |
| 4 | 성능 — 기록 쓰기에 fsync 가 없다 | 흐름 1절 |
| 5 | 성능 — 판정이 spool 잠금을 쥔 채 측정한다 · 판정 주기 10분 | 규칙 6절 |
| 6 | 성능 — 형제와 나눈 spool 에서 받아들임이 느슨하다 | 규칙 5절 |
| 7 | 기술 선택 — 저장소가 정했다. 묻지 않고 목록만 | `requirements.md` 5.6 · `boundary_test.go:102` |

확장 셋은 꺼져 있다 (security-baseline · resiliency-baseline · property-based-testing) — 6절.

---

## 2. 측정 (2026-09-30 · 이 기계)

**방법** — 임시 자리 (`/home/sunny/.claude/jobs/da000f06/tmp`, 끝난 뒤 지웠다 · 작업 트리 불변) 에서 Go 1.26 프로그램 하나. 한 번에
흐름 1절의 일을 차례로 한다 — `Stat` 둘 · `Statfs` · `Mkdir 0700` · 잠금 파일과 `flock` · 기록 (JSON 약 600 바이트) 을 `CreateTemp`
· `Write` · `Close` · `Rename` · 디렉터리 하나를 `renameat2(RENAME_NOREPLACE)` · 기록 한 번 더 · 잠금 놓기. 쓰기 부하는 같은 filesystem
에 `dd` 3 GiB 를 쓰고 `sync` 하는 동안이다.

**filesystem** — ext4 (Proxmox VM 디스크 · LVM) · 157,934,522,368 바이트 · inode 9,830,400 개 · inode 하나에 16,066 바이트 (ext4 기본
비율 16 KiB) · 가용 95.6 GB · 남은 inode 8,820,007.

| 조건 | fsync | n | p50 | p90 | p99 | 최대 |
|---|---|---|---|---|---|---|
| 한가함 | 없음 | 2000 | 0.16 ms | 0.20 ms | 0.46 ms | 24 ms |
| 한가함 | 있음 | 500 | 2.2 ms | 2.9 ms | 8.3 ms | 53 ms |
| 3 GiB 쓰기 중 | 없음 | 3000 | 0.25 ms | 17 ms | 87 ms | 4.2 s (예약의 한 번) |
| 3 GiB 쓰기 중 | 있음 | 300 | 9.9 ms | 56 ms | 4.4 s | 7.2 s |

한가할 때 fsync 없이 나눈 값 (p50) — stat 둘과 statfs 6 µs · 예약 84 µs · rename 10 µs · 확정 62 µs.

**upper 한 항목의 평균 크기** (ADR-076 §4.1 · ADR-077 §12 의 수로 셈) — 하루치 9.3 GB 에 184,528 항목이면 50 KB · 파일만 157,982 개면
59 KB · 굽기 B 8.6 GB 에 147,893 파일이면 58 KB · 처음부터 23 GB 에 681,670 파일이면 34 KB. 모두 ext4 의 16 KiB 보다 크다.

**이 기계에 20% 몫을 대면** — 바이트 몫 19.1 GB (가용의 20%) · inode 몫 1,764,001 (남은 inode 의 20%). 하루치 upper 는 바이트로 둘,
inode 로 아홉이 들어간다. 처음부터 굽기 (23 GB) 는 몫 하나로 퇴출된다. 바이트 몫보다 inode 몫이 먼저 닿는 것은 항목 평균이 약
10.8 KB (가용 바이트 / 남은 inode) 보다 작은 upper 뿐이다 — git 의 느슨한 객체 · 패키지 관리자의 파일 더미 같은 작은 파일 더미.

**앞 유닛 기록** — bake 조각 6 · 7 · 8 은 작은 합성 트리라 (61 항목 · 50,000 · 200,000 항목) N2 의 근거가 못 된다. trash 조각 4 는 9.67 GB
를 보고 뒤 1.09 초에 지웠고 훑기는 초당 약 811,700 항목이다 (`aidlc-state.md:254-257`).

**SunnyVM** (진행자가 읽기만 · 2026-09-30 · `df -T` · `df -i` · 노드의 state 자리와 home 이 같은 filesystem) — ext4 `/dev/sda2` ·
539,407,257,600 바이트 · 가용 344.7 GB · inode 33,488,896 개 · 남은 inode 28,285,100 · inode 하나에 16,107 바이트 (16 KiB 비율). 20% 몫을
대면 바이트 68.9 GB · inode 5,657,020. 하루치 upper 는 바이트로 일곱, 처음부터 굽기 (23 GB) 는 둘이 들어간다.

**못 한 것**

| 무엇 | 까닭 | 물음 |
|---|---|---|
| 노드 uid 로 못 여는 항목을 `unshare` 로 읽기 | 이 기계는 `newuidmap` 이 `Operation not permitted` (FD 계획 2.2) | 4 |
| 보고 뒤 측정의 찬 캐시 · 사내 규모 시간 | 정본도 측정하지 못했다 (ADR-076 §4.1) — 판정이 spool 잠금을 쥐는 시간이 된다 | 4 |
| 전원이 나간 뒤의 기록 | 끊어 볼 수 없다. 이론으로만 (3.3) | 4 |
| btrfs · xfs 의 statfs inode 값 | 이 기계에 없다. btrfs 는 inode 수를 0 으로 낸다고 알려져 있다 | 2 |

---

## 3. 묻지 않고 정한 것

FD 가 정한 것은 다시 묻지 않고 **확인할 조건** 으로 적는다. Step 6 의 `nfr-requirements.md` 가 이 표를 요구로 옮긴다.

### 3.1 성능

| # | 요구 | 확인 |
|---|---|---|
| P1 | 보존이 보고 전 창에 더하는 일은 upper 크기와 무관하다 (결정 2-4) | integration — 큰 upper 와 빈 upper 의 exited_at → finalized_at 이 같은 수준 (조각 4 의 방식) |
| P2 | **보존은 단계의 finalize 판정을 바꾸지 않는다** (FR-10 끝 줄). 쓰기 부하에서 예약이 4.2 초 멈췄다 (2절) — 창의 끝에 걸리면 finalize_timeout 을 부를 수 있다 | 패턴은 NFR Design (4절 넘김). 시험 — 느린 예약을 끼운 가짜 store 로 finalize 칸이 그대로다 |
| P3 | 한가할 때 창 안의 보존 일은 p99 1 ms 아래 (측정 0.46 ms). 게이트가 아니라 기록이다 | integration 에 시간 한 줄 |
| P4 | 기록에 fsync 하지 않는다 — 쓰기 부하에서 fsync 는 p99 4.4 초 · 최대 7.2 초 (2절) | 시험 없음 · 3.3 R2 가 받는다 |
| P5 | spool 잠금을 기다리는 것은 형제의 판정과 조정뿐이다 — 보고 전 창 · 받아들임 · 조회는 안 기다린다 (규칙 5 · 13절). 측정은 한 upper 에 2.6 ~ 3.95 초 (ADR-077 §12) | 시험 — 잠금을 쥔 채 예약 · 조회가 끝난다 |
| P6 | 만료는 늦어도 10분과 측정 시간 뒤에 거둔다 (규칙 6절) | 표 시험 (시계를 넣는다) |
| P7 | 받아들임의 넘침은 다음 판정 (보고 뒤 몇 초) 까지다. 여유 하한은 늘 그 순간의 statfs 로 본다 | 표 시험 |

### 3.2 보안 (규칙 11 · 12절 · 결정 2-11)

| # | 요구 | 확인 |
|---|---|---|
| C1 | spool · 항목 폴더 0700, 기록 0600 — umask 와 무관 | 시험 — umask 0 으로 만든 뒤 mode |
| C2 | host 경로가 receipt · `diagnostics.checkpoint` · 단계 로그 · 광고에 없다 | 시험 — 표지 글자를 담은 scratch 경로로 규칙 2절의 갈래를 모두 돌려 result JSON 과 올린 로그에 그 글자가 없다 |
| C3 | 오류 문장은 errno 의 글만 | 시험 — `PathError` · `LinkError` 를 넣는다 |
| C4 | `show` 의 `unshare` 안내로 컨테이너 root 가 쓴 0600 파일이 읽힌다 | integration · 조각 9 (물음 4) |
| C5 | 자격증명 캐시 안내가 on-failure · always 에서만 보인다 | 제어판 시험 |
| C6 | 계약과 agent 출력에 보존 칸이 없다 | 시험 한 줄 — 계약 문법에 `checkpoint` 칸이 없다 |
| C7 | 기록은 symlink 를 안 따라 읽고 1 MiB 까지 (lower-state 의 선례) | 시험 · 물음 3 |

### 3.3 신뢰 · 가용 · 규모 · 유지

| # | 요구 |
|---|---|
| R1 | 보존의 어떤 오류도 단계를 멈추거나 광고를 막지 않는다. 예외는 규칙 3절의 Close 오류 (보존이 아니다) |
| R2 | 전원이 나가 기록이 없거나 비면 조정이 미완료로 거둔다 — **FD 규칙 9절이 「읽을 수 없는 기록」을 건드리지 않게 적어 이것과 어긋난다** (5절 · 진행자) |
| R3 | 판정과 조정의 오류는 로그 warn 이고 다음 때에 다시 돈다 |
| S1 | 항목 수는 몫과 TTL 이 막는다 (이 기계면 하루치 upper 둘). 판정과 조회는 항목 수에 비례하고 upper 크기와 무관하다 |
| M1 | 패키지별 커버리지 80% 이상 — `internal/scratch` · `internal/enode` · `cmd/enode` (80.8%). 표 시험 · 새 의존 0 · 크로스 빌드 셋 · `glyphscan` · 문구는 영어 (제어판만 한국어) |
| U1 | 보이는 문구는 FD 규칙 12 · 13 · 14절 그대로다 (승인됨) |

### 3.4 기술 선택 — `tech-stack-decisions.md` 에 적을 목록

- Go 1.26 (`go.mod:8`). 새 언어 · 새 모듈 0 (`requirements.md` 5.6)
- `internal/scratch` 는 표준 라이브러리와 `golang.org/x/sys` v0.47.0 (`go.mod:14`) 만 — `unix.Flock` · `Renameat2` · `Statfs` ·
  `O_NOFOLLOW`. 봉인은 `boundary_test.go:102`
- ID 는 `crypto/rand` · 기록은 `encoding/json` 과 `os.CreateTemp` + `Rename` · 측정은 오늘의 trash-helper (`unshare`)
- 시험은 표준 `testing` — 단언 · 모킹 라이브러리 없음
- `_unix` (또는 `_linux`) 와 `_other` 짝 · 크로스 빌드 windows/amd64 · linux/arm · darwin/arm64 (linux/arm 은 32비트 — `Statfs_t` 칸의
  타입을 lower-state 가 확인한 방식으로)

### 3.5 NFR Design 에 넘기는 패턴 (여기서 묻지 않는다)

- P2 를 지키는 모양 — 후보: 예약을 세션을 열 때로 옮겨 창 밖에 두기 · 남은 예산에 여유를 두기 · finalize 판정에서 보존 시간을 빼기
- spool 잠금의 범위 — 측정을 잠금 밖에서 하고 결과만 잠금 안에서 적을지 (P5)
- 받아들임의 보존 총량을 형제의 보존본까지 따라잡는 법 (P7)
- 판정 깸이 겹칠 때 하나로 합치기 · 측정 helper 를 여는 모양

---

## 4. 물음 넷

답을 `[Answer]:` 뒤에 적는다. 권장을 **A** 에 둔다.

**내기 전에 대 본 것**

| 흠 | 대 본 결과 |
|---|---|
| 이미 정한 것을 다시 묻기 | 보안 칸 (0700 · 0600 · 경로 · 안내) · fsync 없음 · 판정 주기 · 받아들임의 느슨함 · 문구는 FD 가 정했다 — 3절에 확인 조건으로만 둔다. 기술 선택은 저장소가 정했다 |
| 측정 없는 근거 | 물음 1 · 2 는 2절의 비율과 몫 셈에 기댄다. SunnyVM 의 디스크 수는 진행자가 읽어 2절에 더했다 |
| 앞 유닛 넘김과 어긋남 | 물음 3 의 A 는 lower-state 의 권한 규칙을 그대로 따른다. 물음 1 · 2 의 선택지는 FD 규칙 10절의 키와 「0 이면 기본값」을 바꾸지 않는다 |
| 요구를 빼는 선택지 | 「하나의 상한이 없다」는 ADR-076 §5 가 요구한 자리를 빼므로 넣지 않았다. P2 를 느슨하게 하는 선택지 (드물면 finalize 판정이 바뀌어도 된다) 는 FR-10 끝 줄을 빼므로 넣지 않았다 |

### Question 1 — `max_gb` 의 기본값 (보존본 하나의 바이트 상한)

몫 (20%) 과 따로, 새 보존본 하나가 오래된 보존본을 모두 밀어내는 것을 막는 상한이다 (ADR-076 §5). 측정한 가장 큰 upper 는
처음부터 굽기의 23 GB, 하루치는 9.3 · 8.6 GB 다 (ADR-077 §12). 몫이 더 작으면 몫이 먼저 막는다 (이 기계 19.1 GB · SunnyVM 68.9 GB · 2절).

A) 32 — 측정한 모든 upper 가 들어간다. 큰 디스크에서 처음부터 굽기가 실패한 트리도 남는다

B) 16 — 하루치는 들어가고 처음부터 굽기는 하나의 상한에서 퇴출된다

C) 기본은 몫의 절반 — 디스크에 따라 바뀐다. 하나가 몫을 혼자 차지하지 못해 적어도 둘이 남는다

D) Other (please describe after [Answer]: tag below)

**권장 A.** 측정한 가장 큰 upper 를 받는 가장 작은 2 의 거듭제곱이다. 굽기의 실패도 보존하기로 했으므로 (FD 답 3) 그 트리를
하나의 상한이 먼저 버리지 않게 한다. 작은 디스크에서는 몫이 막는다. C 는 설정의 숫자와 실제 상한이 달라 소유자가 읽기 어렵다.

[Answer]: A

### Question 2 — `max_total_inodes` 의 기본값 (보존본 전체의 inode 한도)

filesystem 의 inode 가 바닥나면 바이트가 남아도 lower 와 다음 단계가 파일을 못 만든다. Yocto upper 는 항목 평균이 34 ~ 59 KB 라
ext4 의 16 KiB 비율에서 바이트 몫이 먼저 닿는다 (2절). inode 한도가 일하는 것은 작은 파일 더미다.

A) 몫과 같은 비율 — `capacity_percent` x (보존본의 inode 합 + filesystem 의 남은 inode). filesystem 이 inode 수를 내지 않으면
(statfs 의 전체 inode 가 0 — btrfs) 보지 않는다. 설정에 숫자를 적으면 그 숫자다

B) 고정 2,000,000 — 처음부터 굽기 upper (681,670 파일) 셋쯤. 이 기계 전체 inode 의 20%, 100 GB ext4 (inode 약 610만) 면 33%

C) 고정 1,000,000

D) Other (please describe after [Answer]: tag below)

**권장 A.** 바이트 몫과 같은 규칙이라 디스크 크기를 따라가고, 소유자가 설정할 것이 하나 (`capacity_percent`) 로 남는다. 고정값은
작은 디스크에서 inode 를 너무 많이, 큰 디스크에서 너무 적게 잡는다. 대가 — 퇴출 사유가 설정에 없는 키 (`over max_total_inodes`)
를 부를 수 있다. FD 규칙 10절과 엔티티 1절의 「0 이면 N2 의 기본값」 줄에 이 규칙을 적는다 (5절).

[Answer]: A

### Question 3 — 이미 있는 spool 자리가 symlink · 남의 것 · 느슨한 권한일 때

FD 는 spool 을 데몬이 0700 으로 만든다고만 적었다 (규칙 11절). 이미 있는 자리의 모양은 정하지 않았다. lower-state 의 상태 자리는
느슨한 비트를 좁히고, symlink · 디렉터리가 아님 · 남의 것은 거절한다 (`lower-state-code-generation-plan.md:145-162`).

A) lower-state 와 같다 — 느슨한 비트는 기동 때 0700 으로 좁히고 로그 한 줄. symlink · 디렉터리가 아님 · 주인이 노드 uid 가 아니면
고치지 않고 보존을 못 한다 — 요구한 단계는 `failed`(`io`) 와 문장 `the spool is not a private directory of this node; nothing is kept`,
기동 로그 warn 한 줄. 노드는 뜨고 drain 하지 않는다

B) 노드가 안 뜬다 — 설정이 틀린 것과 같게 다룬다

C) Other (please describe after [Answer]: tag below)

**권장 A.** 보존은 결과에 안 들어가는 보조다 (FR-10 끝 줄). 보조 때문에 노드를 세우지 않는다. 남이 놓은 symlink 를 따라가면 밖에
파일이 생긴다는 것을 lower-state 가 보였다 — 그래서 고치지 않고 거절한다. 문장 하나와 로그 한 줄이 새로 보인다.

[Answer]: A

### Question 4 — 확인하지 못한 채로 두는 것 (2절의 표)

`unshare` 로 읽기 · 찬 캐시와 사내 규모의 측정 시간 · 전원이 나간 뒤의 기록 · btrfs 와 xfs 의 inode 값.

A) 받아들인다 — `unshare` 읽기는 integration 과 조각 9 가 확인하고 (빨가면 그때 문구나 값을 고친다), 나머지
셋은 잔여로 적는다. 어느 것도 설계의 모양을 바꾸지 않는다

B) Code Generation 앞에 SunnyVM 에서 먼저 확인한다 — `unshare` 읽기 · 큰 upper 의 찬 캐시 측정 (사람이 돌린다)

C) Other (please describe after [Answer]: tag below)

**권장 A.** 조각 9 가 이 유닛의 병합 게이트라 SunnyVM 을 어차피 거친다. 먼저 확인해도 설계의 모양은 바뀌지 않는다 —
빨가면 고치는 것은 문구와 값이다. 잔여 셋은 정본도 측정하지 않은 것이다 (ADR-076 §4.1).

[Answer]: A

---

## 5. 산출물 계획 (Step 5 이후)

자리는 `aidlc-docs/v4-run-finalize-bake/construction/checkpoint/nfr-requirements/` 이다.

- [x] 답을 읽고 모호함과 서로 막히는 짝을 본다 (물음 1 과 2 — 두 한도의 모양이 한쪽만 몫이면 설정의 뜻이 어긋나지 않나) — 2026-09-30 ·
      넷 모두 A · 어긋나지 않는다 (전체에 거는 둘은 몫, 하나에 거는 상한만 숫자) · 되물음 파일 없음
- [x] `nfr-requirements.md` — 3.1 ~ 3.3 을 요구와 확인 조건으로 · N2 의 두 값 (답 1 · 2) · 물음 3 의 규칙 · 잔여 (답 4) · 2절의 측정
- [x] `tech-stack-decisions.md` — 3.4 의 목록
- [x] FD 에서 고칠 곳을 고쳤다 (진행자 지시 · 표지 「NFR … 로 고침」 · FD 흐름 10절 표) — 처음 계획은 진행자에게 넘긴다였다 — 규칙 9절 「읽을 수 없는 기록」 (R2) · 규칙 10절과 엔티티 1절의 N2 기본값 (답 1 · 2) ·
      물음 3 의 자리 확인 (규칙 2절 ⑤ 문장 · 11절 · 14절 로그 줄)
- [x] NFR Design 에 넘기는 것 (`nfr-requirements.md` 6절) — 3.5
- [x] 검사 — `enode-design/scripts/emphasis-check.py` · 말투 grep · 사내 이름 grep

---

## 6. 확장 준수 — 이 단계

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 spool 줄은 3.2 가 확인 조건으로 받는다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B. R1 ~ R3 은 팩의 요구에서 온다 |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 표 시험 (M1) |
