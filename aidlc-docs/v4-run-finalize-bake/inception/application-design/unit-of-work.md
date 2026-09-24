# 유닛 정의 — 굽기 회차

유닛 하나는 브랜치 하나(`unit/<이름>`) · PR 하나다. 유닛마다 Functional Design 부터 Code
Generation 까지 따로 진행하고 따로 `main` 에 병합한다.

- **입력**: `plans/unit-of-work-plan.md` (답 일곱 · 2.2 · 2.3) · `requirements.md` · `stories.md` ·
  이 폴더의 설계 다섯
- **작성 시각**: 2026-09-24T12:14:10Z · 회차 브랜치 `v4-run-finalize-bake` · HEAD `1b4e606`
- **정한 것**: 유닛 여덟 (Q1 = A) · 조각은 그 기능을 마지막으로 완성하는 유닛이 맡는다 (Q2 = A) ·
  결과 adapter 순연 (Q3 = A) · 한 줄 순서 (Q4 = B) · agent 단계의 전체 훑기를 끈다 (Q7 = A) ·
  굽기 계약이 구울 IR 을 적는다 (2.3)

순서와 의존은 `unit-of-work-dependency.md`, 파일은 `unit-of-work-file-matrix.md`, 스토리 ·
완료 조건 · 조각의 대응은 `unit-of-work-story-map.md` 에 있다.

---

## 0. 한눈에

| 순서 | 유닛 | 하는 일 | FR | 맡는 조각 | 병합 조건 |
|---|---|---|---|---|---|
| 1 | `contract-grammar` 계약 문법 | 작업 성격 · 예산 · 굽기 두 단계 · 구울 IR · 명시 훑기 · 400 검증 · 계약 작성 도구 | 1 · 3 · 5 · 9 의 문법 | 없음 | 코드 검사 |
| 2 | `step-phase` Mediator 진행 구간 | 종료 보고 받기 · phase 칸 셋 · 결과의 새 칸 · 진행 조회 · 대기 사유 | 2 (Mediator) | 0 | 조각 0 |
| 3 | `finalize` 결과 확정 | 전체 훑기 끄기 · 명시 훑기 · 진단 · 종료 보고 보내기 · 예산 둘 · 업로드 client | 1 · 2 (노드) · 3 | 1 · 2 · 3 | 조각 1 · 2 · 3 |
| 4 | `trash` 버리기와 여유 공간 | trash 로 옮기기 · 배경 삭제자 · 여유 부족 drain · arch 조건 제거 · 상태 파일과 제어판 | 4 | 4 | 조각 4 |
| 5 | `merge-rules` 합치기 규칙 | 합치기 표 · 시작 전 확인 · 가짜 트리 재개 시험 | 7 (규칙) | 7 의 기계 부분 | 코드 검사 + 재개 시험 |
| 6 | `lower-state` 아래층 상태와 잠금 | 신원 · 상태 파일 · 잠금 · 후보 잠금 · 광고 키 · 준비도 점검 | 8 (상태 · 잠금 · drain · 점검) · 9 (광고) | 없음 | 코드 검사 |
| 7 | `bake` 굽기 단계 | build · merge 단계 · IR 대조 · 시작 때 정리와 재개 · metadata · 대기 로그 · 합치기 조각 스크립트 | 5 (노드) · 6 · 7 (연결) · 8 (나머지) · 9 (쓰기) | 5 · 6 · 7 · 8 | 조각 5 · 6 · 7 · 8 |
| 8 | `checkpoint` 실패한 단계 보존 | 받아들임 · spool · 크기 판정 · TTL · 재시작 조정 · 조회 명령 | 10 | 9 | 조각 9 |

**코드 검사** = 빌드 · 기본 `go test` · 패키지별 커버리지 80% 이상 · 코드 경계 시험 · 크로스 빌드
셋(windows/amd64 · linux/arm · darwin/arm64) · U+2605 0 · 출력 문자열의 장식 문자 0 ·
`enodectl.exe` 심볼 상한 (`requirements.md` 5.1).

**모든 유닛이 조각 0 을 연기 시험으로 돌린다.** 조각 0 을 맡는 것은 라우트를 더하는
`step-phase` 하나다.

---

## 1. `contract-grammar` — 계약 문법

**하는 일**

- 작업 성격 `effect` — `read` · `edit` · `build` · `prepare`. 안 적으면 명령 단계는 `build`,
  agent 단계는 `edit`
- 예산 `budget.finalize` · `budget.upload` — Go duration. 1분 아래와 0 이하는 400
- 굽기 build 단계 — `effect: prepare` · `sync` · `builds[]`(이름 · 명령) · **구울 IR**(필수 · 값)
- 굽기 merge 단계 — `merge.wait`(기본 4시간) · `needs: build`. 명령이 아니라 노드의 내장 단계
- 명시로 켜는 훑기 `discover` 의 문법 (FR-1)
- 400 검증 — 준비 단계 뒤에 같은 역할의 merge 단계 없음 · 구성 이름 규칙(소문자 · 숫자 · `-`) ·
  이름 겹침 · 구울 IR 없음 · 기간 형식
- 계약 작성 도구 (계획 2.2)
  - `runctl example bake` — **공개 도구로 쓴 굽기 예시.** 사내 명령을 담지 않는다. SunnyVM
    시험이 쓴 poky 가 후보다
  - `runctl lint` 의 판정 조건 제안이 build · merge 단계를 알게 한다
  - `contract.Grammar`(계획을 짓는 agent 에게 주는 문법 문장)에 `effect` 와 예산을 더한다

**만지는 자리** — `internal/contract` · `cmd/runctl/shape.go` 한 파일

**하지 않는 것** — 노드의 실행 · Mediator 의 저장 · `produce` 칸(adapter 순연)

**맡는 조각** — 없음. 조각 5 의 「잘못된 계약은 400」 절반은 이 유닛의 시험이 확인한다.
조각 5 전체(바꾼 대기 상한이 실제로 적용되는지)는 `bake` 가 맡는다.

**받치는 스토리** — US-8 (예산) · US-9 (effect 로 diff 를 계속 받는 길) · US-15 (대기 상한)

**Functional Design — 한다.** 닫을 것:

- **굽기 Run 의 성공 판정** (계획 2.2). build · merge 는 명령 단계가 아니라 `exit_code` 를
  못 건다. 후보는 셋이다 — 두 단계에 「성공으로 끝났다」는 조건을 연다 · `fleet_has` 로 광고를
  본다 · build manifest 를 `$OUT` 산출물로 내고 `produced` 로 본다
- 구울 IR 칸의 이름과 허용 문자 (형식 규칙은 두지 않는다 · 계획 2.3)
- effect 의 기본값 규칙과 400 규칙의 경계값

**NFR Requirements · NFR Design — 건너뛴다.** 성능 표면이 없다. 팩 보안 표의 「계약이 노드
설정이나 host 경로를 지정하지 못한다」는 그런 칸을 만들지 않는 문법 규칙이라 Functional
Design 이 닫는다.

---

## 2. `step-phase` — Mediator 진행 구간

**하는 일**

- `steps` 표에 칸 셋 — `phase` · `phase_since` · `exit` (`ALTER TABLE … ADD COLUMN IF NOT EXISTS`)
- claim 이 `running` 을, merge 종류면 `waiting` 을 적는다
- 종료 보고 받기 `POST /v1/runs/{run}/steps/{seq}/exited` — 노드 · `claimed_instance` · attempt
  를 대조한다. 다르면 거절, 같은 보고의 재전송과 끝난 단계는 조용히 성공한다. 판정 · 다음 단계 ·
  정산 · 임대 해제를 하지 않는다
- 결과 보고가 새 칸을 담아 봉인한다 — 종료 시각 · 결과 확정 시각 · 예산 결과 · 원인 코드 ·
  진단 · 보존 상태 · build manifest · merge 결과
- 진행 조회의 단계마다 `phase` · `phase_since` · `exit`
- 대기(QUEUED) 사유 — 요구 줄마다 후보 수 셋(살아 있음 · 다른 Run 을 맡음 · drain 중).
  `internal/match` 를 두 번 부를 뿐 매칭 규칙은 안 바꾼다
- Record 의 단계 기록에 시각 둘 (노드 시계)

**만지는 자리** — `internal/store` · `internal/api` · `internal/record`

**하지 않는 것** — 노드 쪽 전부 · `internal/match` 변경 · 결과 보고의 인스턴스 대조(잔여로 둔다)

**맡는 조각** — 0 (기동 · 라우트 등록 18 -> 19). 기대는 조각 없음.

**완료 조건** — 4 (대기가 다른 Run 때문인지 drain 때문인지). **스토리** — US-7

**Functional Design — 한다.** 종료 보고의 수락 규칙(재전송 · 늦은 도착 · 끝난 단계) · 두 시계
(실행 계획 7절 ② — 진행 조회의 한 줄에 Mediator 시계와 노드 시계가 섞인다) · phase 전이 ·
후보 수의 계산.

**NFR — 한다 (최소).** 보안 — 종료 보고의 인스턴스 대조(팩 보안 표 첫 줄). 성능 — 대기 중인
Run 을 조회할 때마다 매칭을 두 번 부르는 비용.

---

## 3. `finalize` — 결과 확정

**하는 일**

- 단계 세션의 `Harvest` 를 `Finalize` 로 바꾸고, `Close` 가 위층의 행선지를 받게 한다
  (행선지를 실제로 쓰는 것은 `trash` · `bake` · `checkpoint`)
- 명령 단계 — effect 가 build 면 diff 와 전체 훑기를 끈다. edit 면 오늘의 diff 경로로 낸다
- agent 단계 — 전체 훑기를 끈다. diff 는 adapter 가 생길 때까지 둔다 (Q7)
- 명시로 켜는 훑기 — 방문 수 · 시간 · 메모리 · 결과 크기에 상한. 상한에 닿으면 부분 관찰로 적는다
- 재지 않은 변경 목록을 「바뀐 파일이 없다」로 쓰지 않는다. 빠진 산출물과 collect 실패는 결과의
  진단 칸과 단계 로그 끝에 남긴다. 단계 로그 끝에 「diff 와 바뀐 파일 목록을 이제 안 낸다」 안내
- 종료 보고 보내기 — 명령이 끝나면 따로 보낸다. 실패하면 결과 확정과 나란히 다시 보낸다
- 예산 둘 — 결과 확정 1분 · 업로드 3분 기본. 넘으면 `finalize_timeout` · `upload_timeout`
- 업로드 전용 client — 요청마다의 30초 제한 없이 업로드 예산이 마감을 정한다. 파일을 통째로
  읽지 않고 흘려 보낸다

**만지는 자리** — `internal/enode` (세션 · 명령과 agent 단계 · 업로드) · `cmd/enode` (client)

**하지 않는 것** — trash 로 옮기기(`trash`) · 결과 adapter(순연) · 체크포인트(`checkpoint`)

**맡는 조각** — 1 (기계) · 2 (사람 · 스크래치 Mediator) · 3 (기계). 기대는 조각 — 0.

**완료 조건** — 3 ④ (명령 단계가 diff 를 안 낸다) · 3 ⑤ (agent 단계의 전체 훑기가 꺼진다).
**스토리** — US-8 · US-9 · US-10

**NFR 값** — **N3** (업로드 예산과 요청의 관계 · 흘려 보내기의 상한)

**Functional Design — 한다.** 결과 확정과 닫기의 수명과 예산 경계(`services.md` 1절) · 명시
훑기의 상한과 부분 관찰 표기 · 진단 칸의 모양 · 종료 보고의 재전송 · 두 시계(`step-phase` 와 맞춘다) ·
명시 훑기가 `steps[].workspace` 를 요구하는지.

**NFR — 한다.** 성능 — 임대 창이 트리 크기와 무관한지(조각 1) · N3.

**커버리지** — `internal/enode` 의 여유가 1.5점이다. 이 유닛의 새 문장은 대부분 이 패키지에
들어가므로 규칙은 순수 함수로 떼어 기본 `go test` 에서 시험한다.

---

## 4. `trash` — 버리기와 여유 공간

**하는 일**

- 새 패키지 `internal/scratch` 의 trash 부분 — 이름 바꾸기 한 번으로 옮기기 · namespace 안에서
  지우기(symlink 를 따라가지 않고 trash 밖을 지우지 않는다) · 배경 삭제자(노드 시작 때 한 번,
  결과 보고 뒤마다) · 양(trash · 삭제 중인지 · 측정 시각)
- 작업 폴더를 지우는 다섯 자리를 trash 로 옮기기로 바꾼다 — 열기 실패 · 닫기 · 중단 · helper
  정리 · 준비도 점검 (`runc_overlay_linux.go`)
- trash-helper 입구 (`enode trash-helper`)
- 여유가 `min_free_gb` 아래면 노드 전체가 graceful drain 을 스스로 알린다. 소유자 정책과
  합쳐 센 쪽을 광고한다. 되찾으면 푼다
- `arch` 키에서 디스크 조건문을 지운다 (`detect.go:82` ~ `:98`)
- 상태 파일에 drain 의 출처와 trash 양, 제어판에 그 표시와 「누가 풀 수 있나」
- 예시 설정 넷의 `min_free_gb` 주석을 고친다
- 코드 경계 시험에 세 줄 — Mediator 가 `internal/enode` 를 못 가져다 쓴다 · `internal/scratch`
  를 못 가져다 쓴다 · `internal/scratch` 는 표준 라이브러리와 x/sys 만 쓴다

**만지는 자리** — `internal/scratch` (새) · `internal/enode` (세션 닫기 · 광고 · 탐지 · 상태 파일 ·
설정) · `internal/panel` · `cmd/enode` · `packaging/macos/examples`

**하지 않는 것** — 굽기 출처의 drain(`lower-state`) · spool 과 체크포인트(`checkpoint`)

**맡는 조각** — 4 (사람 · SunnyVM). 기대는 조각 — 0.

**완료 조건** — 1 의 여유 부족 출처 · 2 의 trash 양 · 3 ① (예시 주석). **스토리** — US-1 · US-2 의
일부 · US-3

**NFR 값** — **N1** (배경 삭제자가 한 번에 지우는 양과 속도)

**Functional Design — 한다.** 삭제의 경계 · drain 세기를 합치는 규칙과 광고 응답에 보이는 모양 ·
상태 파일을 쓰는 조건 · 여유를 확인하는 시점.

**NFR — 한다.** N1 · 보안(trash 삭제 줄) · 성능(보고 전에는 이름 바꾸기 한 번만 — 조각 4).

---

## 5. `merge-rules` — 합치기 규칙

**하는 일**

- 새 패키지 `internal/merge` — 시작 전 확인(같은 filesystem · metacopy 꺼짐 · 디렉터리 이름 바꿈
  기록 없음) · 위층 항목 읽기(whiteout 은 문자 장치 0/0 과 xattr 형식, opaque 는
  `user.overlay.opaque`) · ADR-077 §4 의 표대로 합치기 · 종류가 바뀐 항목(아래층 쪽을 trash 로
  옮긴 뒤 대체)
- 표시는 효과를 적용한 뒤에만 지운다. 그래서 끊겨도 같은 절차를 다시 돌리면 결과가 같다
- **가짜 트리 재개 시험** — 표시 종류를 모두 담고 종류가 바뀐 항목을 담은 가짜 트리에서 1 ~ 30번째
  연산 뒤 끊고 다시 돌려, 한 번에 끝낸 목록과 비교한다. SunnyVM 시제품(`merge.py`)의 시험을 옮긴다.
  기본 `go test` 에서 돈다
- 코드 경계 시험에 두 줄 — Mediator 금지 · 표준 라이브러리와 x/sys 만

**만지는 자리** — `internal/merge` (새) · `internal/panel/boundary_test.go`

**하지 않는 것** — 잠금 · 상태 파일 · namespace · 굽기 단계. 이 패키지는 부르는 쪽이 배타 잠금을
쥐고 온다고 믿는다

**맡는 조각** — 7 의 기계 부분(가짜 트리 재개 시험). 조각 7 전체는 `bake` 가 맡는다.

**실행 계획이 건 제약 둘을 이 유닛이 채운다** — 합치기 규칙이 자기 유닛이고(제약 첫째), 재개
시험이 `bake` 보다 먼저 `main` 에 들어간다(제약 둘째 · 실제 아래층에 처음 합치기 전).

**Functional Design — 한다.** 표의 줄마다 동작 · 종류가 바뀐 항목을 `merge.py` 가 어떻게 다뤘는지
확인 · 가짜 표시를 CI(`ubuntu-latest`)에서 다시 확인 (user xattr 이 없으면 시험이 실패해야 한다).

**NFR — 한다 (최소).** 보안 — 합치는 경로가 아래층 밖으로 안 나간다 · 같은 filesystem. 성능 —
비용이 바뀐 항목 수를 따르고 바이트 수와 무관하다.

---

## 6. `lower-state` — 아래층 상태와 잠금

**하는 일**

- 새 패키지 `internal/lower` — 신원(statfs 의 filesystem 번호와 inode) · 상태 자리
  `~/.local/state/enode/lowers/<fsid>-<ino>/` (노드 사용자 전용) · `lower.json` · `state.json` ·
  잠금 둘(`lower.lock` 형제 공유 / 합치기 배타 · `bake.lock` 굽기 배타) · 쥔 사람 기록 ·
  metadata 읽기와 쓰기 · 준비도 점검 셋의 판정
- 후보 잠금 — 노드가 매칭 후보인 동안 공유 잠금을 쥔다. drain 이 받아들여졌고 맡은 Run 이 없으면
  놓는다. 준비 단계를 맡으면 놓는다
- 굽기 출처의 drain — 상태가 pending · merging 이거나 공유 잠금을 못 쥐면 graceful drain
- 광고 키 — `workspace.writes`(runtime 에서) · `ir` · `repo.built.<이름>` · `bake.run` ·
  `bake.resumed`(metadata 에서)
- 준비도 점검 셋 — scratch 와 워크스페이스가 같은 filesystem 인가 · 아래층 루트 소유 uid 가 노드
  uid 인가 · `lower.json` 신원이 맞는가. 어긋나면 not ready 이고 어긋난 것을 이름으로 말한다.
  `internal/environment` 에는 「노드 쪽이 Fact 를 더 낼 수 있다」는 좁은 이음매 하나만 더한다
- 상태 파일과 제어판에 굽기 출처
- 코드 경계 시험에 두 줄

**만지는 자리** — `internal/lower` (새) · `internal/enode` (광고 · 후보 잠금 · 준비도 점검) ·
`internal/environment/check.go` · `internal/panel` · `cmd/enode`

**하지 않는 것** — build · merge 단계와 시작 때 재개(`bake`). Mediator 는 아래층을 모른다

**맡는 조각** — 없음. 코드 검사로 병합한다. 조각 8 의 준비도 점검 부분은 `bake` 가 조각 8 을
돌릴 때 확인한다.

**완료 조건** — 1 (굽기 출처까지 붙어 완성) · 3 ② (scratch 가 다른 filesystem 이면 not ready).
**스토리** — US-1 · US-4

**Functional Design — 한다.** 상태 기계와 전이마다의 잠금 · 공유 잠금을 놓는 조건(몇 번 연속
받아들여졌을 때) · 쥔 사람 기록의 형식과 살아 있는 기록만 읽는 법 · 「이 아래층의 마운트 0」의
증거 · 점검 결과를 어느 State 로 적나.

**NFR — 한다 (최소).** 보안 — 상태 자리 권한. 동시성 — 같은 기계의 여러 노드가 같은 파일과
잠금을 쓴다.

---

## 7. `bake` — 굽기 단계

**하는 일**

- build 단계 — 굽기 잠금(주인이 살아 있으면 곧바로 `bake_in_progress`) · 상태 building ·
  위층에서 sync 와 builds 를 차례로(항목마다 시각과 종료 코드) · **계약의 IR 을 환경 변수로 넘기고
  sync 뒤 HEAD 와 대조**(다르면 실패 · 합치지 않음 · 결과에 HEAD 의 태그와 커밋) · pinned
  manifest · 위층을 대기 자리로 · 상태 pending · 실패하면 위층은 trash, `last_attempt` 에 남김
- merge 단계 — 상태 확인 · 배타 잠금 대기(마감 = 받은 시각 + `merge.wait`) · 기다리는 동안
  로그에 「어느 노드가 어느 Run 으로 잡고 있나 · 남은 시간」 · 마감이면 `merge_wait_timeout`
  (위층 trash · committed · drain 해제) · 잡으면 merging · 시작 전 확인 · merge-helper 로 합치기 ·
  metadata · committed
- merge-helper 입구 (`enode merge-helper`)
- 노드 시작 순서 — 상태 자리 열기 · 굽기 잠금 시도 · 낡은 상태 정리와 끊긴 합치기 재개(metadata
  에 `resumed` 와 원래 Run) · 삭제자 첫 회 · 광고 시작
- 합치기 조각(6 · 7 · 8)의 스크립트 — 굽기를 내기 전에 대상 아래층을 출력하고, 운영 아래층이면
  멈춘다 (완료 조건 8)

**만지는 자리** — `internal/enode` (단계 분기 · 굽기 흐름 · 세션 닫기의 대기 자리) · `cmd/enode`
(시작 순서 · helper) · 조각 스크립트

**하지 않는 것** — 합치기 규칙(`merge-rules`) · 상태와 잠금의 모양(`lower-state`) · 계약 문법

**맡는 조각** — 5 (기계) · 6 · 7 · 8 (사람 · SunnyVM · 버려도 되는 아래층).
기대는 조각 — 1 · 2 · 4 (조각 6 이 먼저 서길 바라는 것 중 자기가 맡지 않는 것).

**완료 조건** — 5 (merge 가 누구를 언제까지 기다리나) · 6 (재개로 합쳐졌나) · 8 (운영 아래층이면
멈춤) · 9 (build 성공과 merge 사유가 따로) · 10 (계약의 IR 과 HEAD 가 어긋난 이유).
**스토리** — US-6 · US-12 · US-13 · US-15 · US-16 · US-17 · US-18 · US-19

**Functional Design — 한다.** build · merge 의 실패 경로와 상태 되돌림 · IR 대조의 환경 변수 이름과
원인 코드 · 대기 로그의 모양과 상한 시각의 시계 · 재개가 굽기 잠금을 먼저 잡은 하나에게 가는 규칙 ·
시작 전 확인이 어긋났을 때 상태를 어디로 돌리나. 굽기 Run 의 성공 판정은 `contract-grammar` 가
닫은 것을 따른다.

**NFR — 한다 (최소).** 보안 — 계약의 명령이 격리 실행 환경 안에서만 돈다. 성능 — 형제가 기다리는
시간은 합치기(하루치 1.49초 실측)가 아니라 형제 Run 이 정한다.

---

## 8. `checkpoint` — 실패한 단계 보존

**하는 일**

- `internal/scratch` 의 Checkpoint Store — 보고 전 받아들임(여유 하한과 보존 총량만) · 위층을
  spool 로 이름 바꾸기 한 번 · 보고 뒤 크기 판정과 퇴출 · TTL · 재시작 때 미완료와 만료 조정 · 목록
- 단계 런타임이 보존 지원 여부를 낸다 (native 는 `unsupported`)
- 영수증의 `checkpoint_capture` — 상태 다섯 · 원인 코드 · 불투명 ID · 범위 · 보장 · 노드 · 만료 시각.
  host 경로는 담지 않는다
- 노드 설정의 checkpoint 블록 — 기본 on-failure · 48시간 · 보존 용량 20%. 원격 계약은 못 바꾼다
- `enode checkpoint list | show` — 노드의 보존본 조회
- 상태 파일과 제어판에 spool 양과 보존 수

**만지는 자리** — `internal/scratch` · `internal/enode` (세션 닫기 · 결과 · 런타임 · 설정 · 상태
파일) · `internal/panel` · `cmd/enode`

**하지 않는 것** — 중앙 보관 · 다른 노드에서 복원(순연 · ADR-076 §6)

**맡는 조각** — 9 (사람 · SunnyVM). 기대는 조각 — 4.

**완료 조건** — 2 (spool 양까지 붙어 완성) · 3 ③ (기본 on-failure · 48시간 · 20%) · 7 (captured 의
뜻). **스토리** — US-2 · US-5 · US-11

**NFR 값** — **N2** (체크포인트 하나의 상한과 inode 한도)

**Functional Design — 한다.** 상태 다섯과 원인 여섯의 전이 · 받아들임 규칙 · 퇴출 순서 · 재시작
조정.

**NFR — 한다.** N2 · 보안(spool 은 주인만 읽는다 · 바깥에 host 경로 없음) · 성능(보고 전에는 이름
바꾸기 한 번).

---

## 9. 이번 범위에서 뺀 것

```text
   FR-11 결과 adapter 둘       순연 (Q3 = A).  사내는 Git changeset 을 쓸 일이 없고 빌드는 사내
                             스크립트가 시작한다.  함께 빠지는 것 — 장면 4 · 조각 10 · 계약의
                             produce 칸 · 광고 키 producer.<이름> · changeset 설명 칸.
                             되살릴 조건과 설계 빚(받는 쪽 · base 의 뜻 · 인터페이스)은
                             requirements.md 6.1 · 10절 4-16
   FR-12 ADR-073 잔여          유닛 없음.  코드가 0 이다.  조각 11 은 Build and Test 에서 사용자가
                             운영 host 에서 돈다.  ADR-073 §11 문서 고침은 진행자가 정본에 올린다
   FR-13 ADR-072 결정 조건      유닛 없음.  조각 12 는 Build and Test 에서 돈다 (조각 6 뒤).
                             오늘의 diff 경로(steps[].workspace 와 in.from)로 돈다
```

---

## 10. 크로스 빌드 — Linux 전용 코드

CI 는 windows/amd64 · linux/arm · darwin/arm64 로도 짓는다 (실행 계획 7절 ④). Linux 전용
시스템 호출을 쓰는 유닛은 `_linux.go` 와 `_other.go` 를 짝으로 두고, `_other.go` 는 「지원 안 함」
을 돌려준다 (`runc_overlay_other.go` 가 선례다).

```text
   trash         namespace 안의 지우기 · 여유 측정
   merge-rules   mknod 표시 읽기 · user xattr · 같은 filesystem 확인
   lower-state   statfs 의 filesystem 번호 · flock · 소유 uid
   bake          merge-helper 를 unshare 뒤에서 여는 입구
   checkpoint    statfs 여유 · 보존 지원 여부
```

linux/arm 은 32비트다. filesystem 번호와 inode 를 담는 타입이 그 위에서도 컴파일되는지
`lower-state` 의 Code Generation 계획이 확인한다.

---

## 11. 확장 준수 요약 — Units Generation 단계

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩의 보안 표는 유닛마다의 NFR 칸에 나눠 적었다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 합치기 재개 시험은 기존 컨벤션 안에서 가짜 트리로 돈다 (`merge-rules`) |
