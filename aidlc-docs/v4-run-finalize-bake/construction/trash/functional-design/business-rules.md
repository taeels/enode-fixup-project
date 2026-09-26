# `trash` — 규칙

노드가 단계의 작업 폴더를 어떻게 버리고, 배경에서 어떻게 지우고, 여유가 모자랄 때 어떻게 스스로 빠지나의 규칙이다.
로그 줄 · 상태 파일의 값 · helper 의 출력은 영어다 (`CONVENTIONS.md` 2.1 — 밖으로 나간다). 제어판 화면의 문구는 오늘
화면의 언어(한국어)를 따른다. 타입과 이름은 `domain-entities.md` 에 있다.

**원칙 셋**

- **보고 전 창에는 rename 한 번만 든다** (`requirements.md` 5.4 · `components.md` 6절). 걷는 일 — 지우기와 측정 — 은 전부
  보고 뒤이고 helper 안이다
- **삭제는 판정이 아니다.** 지우기에 실패해도 단계의 결과는 안 바뀐다. 항목이 trash 에 남고 노드 로그에 이유가 남는다
  (등급은 노드 · `components.md` 5절)
- **노드가 스스로 거는 drain 은 새 어휘가 아니다.** 광고의 `policy.drain` 에 실리는 값 하나이고, Mediator 는 오늘처럼
  draining 노드를 후보에서 뺀다 (ADR-063). 출처는 노드 쪽(상태 파일과 제어판)에만 있다

---

## 1. 작업 폴더를 버리는 다섯 자리 (FR-4)

| 자리 | 오늘 | 이 유닛 |
|---|---|---|
| ① `Open` 실패 갈래 (`:144` · `:160` · `:165` · `:171`) | 노드 uid 로 `RemoveAll` | `Trash.Move` 한 뒤 잠금을 놓는다. 막 만든 빈 폴더라도 같은 길로 간다 — 자리가 하나면 빠지는 경로가 없다 |
| ② 세션 `Close` (`:489` · abort 뒤 `:457`) | helper 를 닫고 남은 것을 `RemoveAll` | helper 를 닫고(unmount 만) `Trash.Move` 한 뒤 잠금을 놓는다 |
| ③ `abort` (`:503`) | helper 를 죽이고 `RemoveAll` | helper 를 죽이고 `Trash.Move` 한 뒤 잠금을 놓는다 |
| ④ helper `cleanup` (`:1094`) | namespace 안에서 unmount 뒤 `RemoveAll` | unmount 만 한다. 지우지 않는다 |
| ⑤ 준비도 smoke (`:1339` · `:1352`) | ② 를 지난다 · 닫은 뒤 폴더가 없는지 본다 | ② 를 지난다 · 닫은 뒤 폴더가 **원래 자리에** 없는지 본다(trash 로 갔다). 문구는 오늘 그대로 |

- **`Move` 가 실패하면 지우지 않는다.** 오류는 `runtime cleanup: move runtime session to trash: <err>` 로 결과에 남는다
  (오늘의 `runtime cleanup:` 머리말). 같은 scratch 안의 rename 이라 늘 성립해야 하고, 실패하면 설정이나 권한이 틀린 것이다
- **이름이 겹치면 뒤에 `-1` · `-2` … 를 붙인다.** 작업 폴더 이름은 `MkdirTemp` 로 유일하지만, 지운 이름이 다시 나오는 동안
  옛 항목이 trash 에 남아 있을 수 있다
- **trash 는 `<scratch>/trash` 이고 0700 이다.** 없으면 `Move` 가 만든다
- **잠금은 rename 뒤에 놓는다.** 앞에 놓으면 그 사이에 다른 데몬의 기동 청소가 같은 폴더를 옮기려 한다

---

## 2. 닫기와 Finalize 예산 (답 1 = A · finalize 가 넘긴 일)

```text
   exited_at --[ Finalize 예산 ------------------------------ ]
             Finalize (collect · stat · diff · 훑기)   닫기 (unmount · rename)   finalized_at
```

- 닫기는 Finalize 예산 안이다. **닫기 자체는 마감으로 끊지 않는다** — 닫기는 Worker 의 ctx(마감 없음)로 돈다. 반쯤 닫은
  세션은 helper 와 마운트를 남긴다
- **닫기가 끝난 시각이 마감을 넘었으면 `finalize_timeout` 이다.** Finalize 가 이미 넘겼으면 그대로 timeout 이다. 문구는 finalize
  유닛의 `finalize budget of %s exceeded` 그대로이고 단계 로그 끝 줄도 같다
- 임대가 끝났으면 오늘처럼 timeout 이 아니다 (`settle` 의 규칙 그대로)
- `finalized_at` 은 닫기가 끝난 시각이다 — 뜻이 안 바뀐다. 이제 그 안에 지우는 시간이 없다

---

## 3. 남은 작업 폴더 — 기동 청소 (답 2 = A)

데몬이 죽으면 helper 는 `Pdeathsig` 로 죽고 작업 폴더가 scratch 에 남는다 (`:157`).

- **데몬이 기동할 때 한 번** scratch 의 `enode-runc-*` 를 본다. runc-overlay 노드만이다 — native 노드는 scratch 가 없다
- 잠금 파일(`.enode-session.lock`)이 있고 **잠금을 쥘 수 있으면** 남은 것이다 — 쥔 프로세스가 없다. trash 로 옮긴다
- 잠금을 못 쥐면 살아 있는 세션이다 — 다른 데몬의 단계나 도는 `enode env check` 의 smoke 다. 건드리지 않는다
- **잠금 파일이 없으면** 이 유닛 전의 enode 가 남긴 폴더다. 만든 지 1시간이 지났으면 남은 것으로 치고 옮긴다. 1시간 안이면
  건드리지 않는다 — `MkdirTemp` 와 잠금 사이의 짧은 창을 비킨다
- 옮긴 것마다 노드 로그 info `orphaned runtime session moved to trash` (name) · 못 옮기면 warn `cannot move an orphaned runtime
  session to trash` (name · err)
- `enode-smoke-io-*` 는 대상이 아니다 — 노드 uid 가 만든 작은 폴더이고 smoke 가 오늘처럼 지운다

---

## 4. 배경 삭제자 (답 3 = A)

### 4.1 언제 깨나

| 때 | 한다 |
|---|---|
| 데몬 기동 | 3절의 청소 뒤 곧바로 한 번. **배경에서** 돈다 — 광고는 기다리지 않는다 (답 10 = A) |
| 결과 보고 뒤 | `Kick`. Worker 가 보고를 끝낸 뒤(닿았든 포기했든) 부른다. 막지 않는다 — 이미 깨어 있으면 하나로 합친다 |
| trash 에 항목이 남아 있을 때 | 10분마다. 실패한 항목과 데몬 밖에서 들어온 항목(`enode env check` 의 smoke)을 거둔다 |

**설계가 더한 것** — 계획의 답은 「실패한 항목이 남았으면 10분마다」였다. 데몬이 도는 동안 `enode env check` 가 남긴 항목도
같은 자리에서 거두도록 「항목이 남아 있으면」으로 넓혔다. 비어 있으면 깨지 않는다.

### 4.2 한 번에 무엇을

- trash 의 이름을 읽어(노드 uid) **하나씩 차례로** helper 하나로 지운다. 동시에 둘을 안 연다
- helper 는 먼저 측정하고(4.4) 그다음 지운다. 한 번에 지우는 양은 항목 하나다 — 그 값과 속도의 판단은 NFR (N1) 이다
- **도는 단계가 있어도 멈추지 않는다.** helper 는 IO 우선순위 idle · CPU 우선순위 19 로 돈다
- **IO 우선순위 idle 은 BFQ 스케줄러의 디스크에서만 먹는다.** `none` · `mq-deadline` 디스크에서는 삭제자가 도는 단계와 보통
  우선순위로 IO 를 나눈다 (이 기계의 블록 장치는 `none` 이다). 사용자가 그대로 두기로 했다 (2026-09-26 · 쉬는 안 B 를 고르지
  않았다) — 바쁜 노드도 trash 가 비워지는 쪽을 골랐다. 삭제 속도와 SunnyVM 의 스케줄러는 Code Generation 과 조각 4 에서 측정한다
- Worker 가 끝나면(데몬 종료) 도는 helper 를 죽이고 멈춘다 — 반쯤 지운 항목은 trash 에 남고 다음 기동이 거둔다

### 4.3 결과마다

| helper 의 결과 | 삭제자 | 노드 로그 |
|---|---|---|
| exit 0 · `removed: true` | 측정값을 지운다 | debug `trash entry removed` (name · bytes · took) |
| `removed: false` · `left` | 항목이 남는다 | warn `trash entry left in part; it reaches into another filesystem` (name · left) |
| `error` · exit 1 | 항목이 남는다. 다음 깸에 다시 한다 | warn `cannot remove trash entry; it stays in trash` (name · err) |
| helper 를 못 띄웠다 (unshare 없음 등) | 이번 깸을 멈춘다 | warn `cannot start the trash helper; trash is not emptied` (err). 원인이 바뀔 때만 다시 적는다 |

### 4.4 양을 측정하는 법 (답 5 = A)

- helper 가 지우기 전에 같은 걷기 규칙(5절)으로 **블록 수 x 512 의 합과 항목 수**를 낸다 (`{"measured": …}`). 삭제자가 그 줄을
  받는 순간 `Usage` 를 고친다 — 지우는 동안 「곧 지워질 양」이 보인다
- `Usage` — `trash_entries` 는 trash 에 남은 이름 수(노드 uid 로 읽는다) · `trash_bytes` 는 그 가운데 측정한 것의 합 ·
  `trash_unsized` 는 아직 측정하지 않은 수 · `deleting` 은 helper 가 도는 중인가 · `measured_at` 은 마지막으로 고친 시각
- 측정값은 메모리에만 있다. 재시작 뒤 남은 항목은 다시 「크기 모름」이다

---

## 5. 삭제의 경계 (답 4 = A · 팩 보안 표의 trash 삭제 줄)

- helper 는 **trash 의 바로 아래 이름 하나**만 받는다. 빈 이름 · `.` · `..` · `/` 가 든 이름은 `invalid trash entry name` 으로 거절한다
- trash 가 디렉터리가 아니거나 symlink 면 `trash is not a plain directory` 로 거절한다
- 걷기는 `lstat` 으로만 본다. **symlink 는 링크 자체만 지운다** — 가리키는 곳으로 안 들어간다
- **권한 000 인 디렉터리**(overlay 의 `work/work`)는 들어가기 전에 0700 으로 푼다. namespace 안의 root 가 권한 검사를 넘을 수
  있다는 가정에 코드가 기대지 않는다
- **trash 와 st_dev 가 다른 디렉터리에는 들어가지 않는다.** 그 경로를 `left` 에 적고 남긴다 — 남은 마운트로 다른 filesystem 을
  지우는 일을 막는다. trash-helper 는 마운트 namespace 를 안 연다
- 측정도 같은 경계를 지킨다

---

## 6. 여유 부족 drain (답 6 · 7 = A)

### 6.1 거는 노드와 측정하는 자리

- **`min_free_gb` 를 가진 모든 노드가 건다.** 기본값 10 그대로다 — native · runc-overlay · agent 만 하는 노드 모두
- 여유는 **워크스페이스의 filesystem** 에서 광고 주기마다 측정한다 (statfs · 오늘의 `freeBytes`). runc-overlay 노드는 FR-8 의
  env check 가 scratch 와 워크스페이스를 같은 filesystem 으로 강제한다
- 워크스페이스가 없는 노드는 여유를 안 보고 걸지 않는다 (오늘의 `hasRoom` 과 같다)
- 여유를 측정하지 못하면 넉넉한 것으로 치고 warn `cannot measure free disk; assuming enough` 를 원인이 바뀔 때만 적는다 (오늘 문구)

### 6.2 걸고 푸는 조건

| 앞 광고 | 여유 | 이번 광고 |
|---|---|---|
| 안 걸었다 | < `min_free_gb` | 건다 — graceful · 노드 로그 warn `free disk below min_free_gb; draining this node` (free_gb · min_gb) |
| 안 걸었다 | >= `min_free_gb` | 안 건다 |
| 걸었다 | < `min_free_gb` + 1 GB | 그대로 건다 |
| 걸었다 | >= `min_free_gb` + 1 GB | 푼다 — 노드 로그 info `free disk recovered; lifting the disk drain` (free_gb · min_gb) |

GB 는 오늘처럼 2^30 바이트의 몫(내림)이다.

**Detail 문구** — 걸 때 `free 7 GB < min 10 GB` · 푸는 선 아래에서 걸어 둔 동안 `free 10 GB < min 10 GB + 1 GB to lift`.

### 6.3 합치기

| 소유자 (정책 파일) | 여유 부족 | 광고의 `policy.drain` |
|---|---|---|
| 없음 | 없음 | 없음 |
| 없음 | graceful | graceful |
| graceful | graceful | graceful |
| at-boundary | graceful | at-boundary |
| graceful 또는 at-boundary | 없음 | 소유자 값 |

- 세기는 at-boundary > graceful > 없음. 출처 목록을 받아 센 쪽 하나를 낸다 — lower-state 유닛이 bake 출처를 더해도 이 표가
  그대로다
- 소유자 출처 — `Kind: owner` · `Owner: true` · Detail `set in the policy file`
- 여유 부족 출처 — `Kind: disk` · `Owner: false` · Detail 은 6.2
- **소유자가 drain 을 풀어도 여유 부족이 남으면 노드는 빠진 채다.** 그 사실은 제어판이 보인다 (8절)

### 6.4 arch 키

`arch` 와 `arch.<이름>` 키는 **툴체인 탐지만 따른다.** 여유가 모자라도 광고에 남는다 — 노드 전체가 drain 으로 빠지므로 키를
빼서 막을 일이 없다. `detect.go` 의 조건문과 경고 줄(`not enough free disk; dropping build capability …`)을 지운다.

---

## 7. 상태 파일 (답 8 = A)

- 칸 넷 — `caps` · `at` (오늘) · `drain` (실린 값 · 출처 · 그 광고의 시각) · `scratch` (Usage). scratch 가 없는 노드는 `scratch` 가 없다
- **쓰는 때** — 어느 칸이든 값이 바뀌면 쓴다. 광고 주기(caps 의 시각 · drain)와 삭제자(측정 · 지움 시작 · 지움 끝)에서
- 같은 값이면 안 쓴다. 쓰기는 임시 파일에 쓰고 이름을 바꾼다 (오늘 그대로)
- 못 쓰면 warn `cannot write the status file; the panel will show capabilities as unknown` 를 원인이 바뀔 때만 적는다 (오늘 문구).
  광고와 삭제는 안 막는다

---

## 8. 제어판 (답 9 = A)

### 8.1 drain 칸

- **데몬이 돌고**(잠금 파일 · proc — 오늘의 `ProcView`) 상태 파일에 `drain` 칸이 있으면 **그 값**을 보인다. 아니면 오늘처럼 정책 파일의 값이다
- 출처마다 한 줄 — 이름 · 값 · Detail · 누가 풀 수 있나

| 출처 | 이름 | 누가 풀 수 있나 |
|---|---|---|
| owner | 소유자 정책 | 여기서 풀 수 있다 |
| disk | 여유 부족 | 저절로 풀린다 — trash 가 비거나 디스크가 늘면 |

- 「drain 풀기」 버튼은 **소유자 출처가 있을 때만** 보인다. 다른 출처가 남아 있으면 버튼 곁에 「풀어도 여유 부족 drain 이
  남아 노드는 빠져 있다」를 적는다 (남은 출처의 이름을 넣는다)
- 「drain 걸기」는 오늘 그대로다 — 소유자 정책을 쓴다

### 8.2 trash 칸

`scratch` 칸이 있을 때만 — 「trash 9.0 GiB · 항목 2 (1 개는 크기 모름) · 지우는 중 · 측정 12:03:11」. 크기 표기는 오늘 화면의 표기를 따른다.

---

## 9. 예시 설정 넷 (완료 조건 3 ①)

`packaging/macos/examples` 의 `local.yaml` · `colima.yaml` · `qemu.yaml` · `zephyr.yaml` 의 주석을 바꾼다.

```text
   오늘   디스크가 이 아래로 떨어지면 빌드 능력을 광고에서 뺀다 (기본 10).
   바꿈   워크스페이스의 여유가 이 아래로 떨어지면 노드 전체가 drain 한다 (기본 10 GB).
          이 값 + 1 GB 를 되찾으면 풀린다.  빌드 능력(arch 키)은 여유와 무관하게 광고에 남는다.
```

---

## 10. 코드 경계 시험 (유닛 정의 4절)

`internal/panel/boundary_test.go` 의 표에 더한다.

```text
   금지 둘    cmd/mediator 는 internal/enode 를 못 가져다 쓴다
             cmd/mediator 는 internal/scratch 를 못 가져다 쓴다
   봉인 하나   internal/scratch 는 표준 라이브러리와 golang.org/x/sys 만 쓴다
```

---

## 11. 확장 준수 — Functional Design

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 trash 삭제 줄은 5절이 닫는다 — namespace 안 · symlink 를 안 따라감 · trash 밖을 안 지움 · 다른 filesystem 으로 안 넘어감 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 한 줄 |
