# 응용 설계 — 굽기 회차 (통합본)

넷의 통합본이다 — `components.md` · `component-methods.md` · `services.md` ·
`component-dependency.md`. 여기는 **무엇을 정했고 어디에 적었나**를 한 장으로 보인다.

- **입력**: `requirements.md` (FR-1 ~ FR-13 · 8절 ① ~ ⑬) · `stories.md` (완료 조건 열) ·
  `plans/execution-plan.md` (7절의 넷) · `plans/application-design-plan.md` (착수 전 실측 열 · 답 일곱)
- **작성 시각**: 2026-09-24T06:33:21Z · 회차 브랜치 `v4-run-finalize-bake` · HEAD `b7f7c7a`
- **답**: 일곱 다 A. 채팅에서 하나씩 논의했다 (계획 6절)

---

## 1. 한 문단

노드 쪽에 새 패키지 셋이 선다 — `internal/lower`(신원 · 상태 · 잠금 · metadata) ·
`internal/merge`(합치기 규칙) · `internal/scratch`(trash · 삭제자 · checkpoint store).
`internal/enode` 는 그것을 이어 붙이고, 걷는 일은 namespace helper 에서 돈다. 형제 노드는
**매칭 후보인 동안** lower 공유 잠금을 쥐어 매칭과 첫 단계 사이의 틈을 닫는다. Mediator 는
lower 를 모른 채 칸 셋과 라우트 하나만 는다. 사람이 읽는 자리는 기존 통로를 탄다 — 노드
소유자는 제어판, 굽기 담당은 단계 로그와 광고 키, 계약 작성자는 단계 로그와 진행 조회.

---

## 2. 닫은 것 — 어디에 적었나

### 2.1 `requirements.md` 8절

| # | 무엇 | 답 | 자리 |
|---|---|---|---|
| ① | effect 표기 · 명령 단계 edit 의 changeset | `read` · `edit` · `build` · `prepare`. 명령 단계 edit 은 오늘의 git diff 경로 | 계획 2절 · methods 5절 |
| ② | 예산 둘과 merge 대기의 표기 · 노드 쪽 상한 | `budget.finalize` · `budget.upload` · `merge.wait` (Go duration). 상한 없음 · 형식과 하한만 | Q6 · methods 5절 |
| ③ | lower 공유 잠금의 시점 | 매칭 후보인 동안 쥔다. drain 이 받아 적히고 임대가 없으면 놓는다. prepare claim 에 놓는다 | Q1 · services 5절 |
| ④ | 스스로 건 drain 과 소유자 drain | 센 쪽을 싣는다. 출처는 상태 파일과 제어판 | 계획 2절 · Q3 |
| ⑤ | 새 노드 쪽 코드의 자리 | 새 패키지 셋 | Q2 · components 2절 |
| ⑥ | 「바뀐 파일이 없다」 뒤 진단의 자리 | result 의 진단 칸과 단계 로그. produced 에서 뺀다 | Q5 · methods 4.1 |
| ⑦ | phase 칸 | steps 에 phase · phase_since · exit | 계획 2절 · methods 7절 |
| ⑧ | checkpoint descriptor 와 조회 · reason 표기 | receipt 의 `checkpoint_capture` 일곱 칸. 조회는 `enode checkpoint`. Mediator API 없음 | 계획 2절 · methods 3.2 |
| ⑨ | checkpoint 상한 · 삭제자의 양 | 자리는 노드 설정. 값은 NFR N1 · N2 | 계획 2절 |
| ⑩ | IR 태그 규칙의 자리 | **Units Generation 이 고쳤다** — 굽기 계약 build 단계가 구울 IR 을 정확한 값으로 적는다(필수 · 기본값 없음). 노드가 환경 변수로 넘기고 sync 뒤 대조한다. 처음 판은 `ir_tag`(규칙) · 기본 사내 형식 | Q7 · services 2절 |
| ⑪ | Git changeset wire · 10 MiB · producer 등록 | descriptor(base · digest · size · complete) · 넘으면 complete=false 로 patch 안 올림 · 노드 바이너리의 등록표와 광고 `producer.<이름>=<판>` | 계획 2절 · methods 4.2 |
| ⑫ | 가짜 whiteout · opaque 를 특권 없이 | 된다 — 이 기계에서 실측 | 계획 1.3 |
| ⑬ | 유닛 분해 | Units Generation 몫. 7절에 넘길 것 | — |

### 2.2 `execution-plan.md` 7절

| # | 무엇 | 답 |
|---|---|---|
| ① | 업로드 예산과 요청마다의 30초 | 업로드 전용 client(요청 제한 없음) · 예산이 context 마감 · 스트림. 값은 NFR N3 (계획 1.9) |
| ② | 두 시계 | 자리를 적었다 — started_at · ended_at 은 Mediator, exited_at · finalized_at 은 노드. 규칙은 Functional Design |
| ③ | env check 의 자리 | `RuntimeVerifier` 곁의 이음매로 노드 쪽이 낸다. environment 는 새 내부 임포트 0 (계획 1.4) |
| ④ | 크로스 빌드 | 새 셋은 `_linux.go` 와 `_other.go` 짝. `_other` 는 unsupported |

### 2.3 완료 조건 (`stories.md` 2절)

| 조건 | 누가 무엇을 | 자리 |
|---|---|---|
| 1 | 노드 소유자가 drain 의 출처 셋을 가른다 | 상태 파일의 drain 출처 · 제어판 (Q3) |
| 2 | 노드 소유자가 trash · spool 의 양과 삭제자를 본다 | 상태 파일의 scratch · 제어판 (Q3). 늦은 값이고 잰 시각이 붙는다 |
| 3 ① | min_free_gb 의 뜻이 바뀐다 | `packaging/macos/examples` 넷의 주석 (components 3.9) |
| 3 ② | scratch 가 다른 filesystem 이면 not ready | env check 의 Fact 이름 `binding.scratch_filesystem` (methods 1.5) |
| 3 ③ | checkpoint 기본 on-failure · 48시간 · 20% | 노드 설정의 checkpoint 블록 — 켜는 자리가 끄는 자리. 제어판에 보존 중인 수 (Q3) |
| 3 ④ | 명령 단계가 diff 를 안 낸다 | 단계 로그 끝의 한 줄과 effect: edit 안내 (Q5) |
| 3 ⑤ | agent 단계의 걷기가 adapter 뒤에 꺼진다 | adapter 가 켜지는 판에 같은 자리(단계 로그)에 한 줄. FR-11 이 범위에 남으면 |
| 4 | QUEUED 가 점유 때문인지 drain 때문인지 | 진행 조회의 요구 줄 후보 셋 (methods 7절 · services 6절) |
| 5 | merge 가 누구를 언제까지 기다리나 | merge 단계의 실시간 로그 (Q4 · services 3절) |
| 6 | FAILED 굽기가 재개로 합쳐졌나 | 광고 키 `bake.run` · `bake.resumed` 를 `GET /v1/nodes` 에서 (Q4) |
| 7 | `captured` 의 뜻 | descriptor 가 scope · guarantee · node · expires_at 을 싣는다 (methods 3.2) |
| 8 | 합치기 조각이 운영 lower 면 멈춘다 | 조각의 스크립트 — Build and Test 몫. 설계는 안 막는다 |
| 9 | merge_wait_timeout 의 굽기에서 build 성공과 merge 사유가 따로 | 두 단계가 각자 result 를 갖는다 (services 2 · 3절) |
| 10 | 계약의 IR 과 sync 뒤 HEAD 가 어긋난 이유 | build 단계 결과 — HEAD 의 태그와 커밋 (Units Generation 이 고쳤다. 처음 판은 metadata 의 `ir_reason`) |

### 2.4 `decisions.md` 6절의 한 줄

「명령 앞에서 임대를 쥐는 준비 구간을 phase 에 올릴지」 — **안 올린다.** 스토리와 조각이
요구하지 않고, phase 는 열린 어휘라(ADR-075 §10.3) 뒤에 더해도 스키마와 판정이 안 바뀐다.

---

## 3. 컴포넌트 · 흐름 · 의존 — 요약

```text
   새 셋            internal/lower · internal/merge · internal/scratch
                    표준 라이브러리와 x/sys 만.  서로 임포트 안 함.  Mediator 가 링크 안 함
   책임이 느는 것     internal/enode (이어 붙이기) · contract (문법) · store (칸 셋 · 수락 · 후보 셋) ·
                    api (라우트 하나) · record (시각 둘) · environment (이음매 하나) ·
                    panel (거짓이 되는 drain 표시) · cmd/enode (기동 · helper · CLI) ·
                    packaging/macos/examples (주석 넷)
   안 만지는 것      match · schema · proc · transcript · transcriptui · runctl · build ·
                    cmd 의 mediator · enodectl · runctl · iapadapter
   흐름 여섯         명령 단계 · build · merge · 노드 기동 · 광고 주기 · Mediator (services.md)
   namespace        걷는 일(삭제 · 합치기 · 크기 측정 · 세션)은 helper 안.  임대 창에는 rename 뿐
   라우트            mux.HandleFunc 18 -> 19 · internal/api 전체 27 -> 28
```

---

## 4. 요구와 대조

### 4.1 임포트 경계 (5.2)

`go list -deps ./cmd/mediator` 의 내부 패키지 열둘에 새 셋이 안 든다. 경계 시험에 금지
넷(cmd/mediator 에서 enode · lower · merge · scratch)과 봉인 셋(새 셋은 표준 라이브러리와
x/sys 만)을 더한다 (`component-dependency.md` 2절).

### 4.2 불변식 (5.5)

| 불변식 | 이 설계에서 |
|---|---|
| I1 · 새 점유 단위 없음 | lower 조율은 기계 안의 flock 과 광고 drain 이다. Mediator 에 행이 없다 |
| 단계 상태 어휘 | 안 바뀐다. 새 구간은 phase 칸이다 |
| 매칭 규칙 | 안 바뀐다. 새 광고 키는 평평한 문자열이다. QUEUED 의 후보 셋은 Match 를 두 번 부를 뿐이다 |
| Record append-only | 재개의 흔적은 metadata 가 진다. 봉인된 Record 를 안 고친다 |
| 마운트된 lower | metadata 는 배타 잠금 창(마운트 0)에서만 쓴다 |
| 옛 노드 | exited 를 안 보내면 phase 는 running 에 머문다. Mediator 를 먼저 올려도 안 깨진다 |
| arch 광고 (바뀜) | 툴체인만 본다. 여유 부족은 노드 전체 drain |
| runc-overlay 준비도 (바뀜) | scratch 와 워크스페이스의 st_dev 가 다르면 not ready. 사유를 이름으로 |
| 시험 안전 | 합치기 규칙은 가짜 트리에서. 실제 합치기 조각은 Build and Test 의 스크립트가 lower 를 먼저 보인다 |

### 4.3 팩의 보안 표 (5.3)

| 자리 | 이 설계에서 |
|---|---|
| 종료 보고 | 노드 · claimed_instance · attempt 대조. 다르면 거절 (services 6절) |
| trash 삭제 | trash-helper 가 namespace 안에서. symlink 안 따라감 · trash 밖 안 지움 (`scratch.Remove`) |
| 합치기 | `merge.Preflight` 가 같은 filesystem 을 본다. 합치는 경로가 lower 밖으로 안 나간다 (`merge.Apply`) |
| 상태 자리 | `lower.Open` 이 노드 사용자 전용으로 만든다 |
| spool | 소유자만 읽는다. descriptor 에 host 경로 없음. TTL 에 지운다 (`scratch.Store`) |
| 계약의 명령 | sync 와 builds 는 build 단계의 세션 안에서 돈다. 구울 IR 값은 노드 설정도 host 경로도 아니다 (Q7) |
| 정책 | 계약에 checkpoint 칸이 없다. TTL · 보존은 노드 설정만 |

**잔여** — result 는 오늘처럼 노드만 대조한다 (5.3). 이 설계가 좁히지 않는다.

---

## 5. 확장 준수 요약 — Application Design 단계

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩의 보안 표는 4.3 에서 요구로 대조했다 |
| resiliency-baseline | N/A (꺼짐) | 질문 5 = B |
| property-based-testing | N/A (꺼짐) | 질문 6 = X. 합치기 재개 시험은 기존 컨벤션 안에서 가짜 트리로 (`merge.Options.OnOp`) |

---

## 6. 뒤 단계에 넘기는 것

### 6.1 Units Generation

```text
   패키지 경계       새 셋이 유닛의 자연스러운 이음매다.  합치기 규칙(internal/merge)은 자기 유닛으로
                    (실행 계획의 제약 ①)
   직렬 병합 지점     실행 계획의 다섯에 셋이 는다
                      internal/contract/contract.go · internal/enode/claim.go ·
                      internal/enode/runc_overlay_linux.go · internal/store/schema.sql · internal/api/api.go
                      + internal/enode/advertise.go · internal/enode/detect.go (drain 합성과 새 키)
                      + internal/enode/status.go (상태 파일의 새 칸 — panel 과 맞물린다)
   범위를 자를 후보   FR-11 의 adapter 둘 (계획 2절 ⑪).  조각 10 만 받는다
   먼저 설 것        internal/contract (노드와 Mediator 가 함께 딛는다) ·
                    가짜 트리의 재개 시험 (실행 계획의 제약 ② — 실제 lower 에 처음 돌리기 전에)
```

### 6.2 Functional Design

```text
   Q1 의 놓는 울타리          drain 이 받아 적힌 응답을 몇 번 연속 받고 임대가 0 이면 놓나 (계획 Q1 답)
   쥔 사람 기록의 형식         살아 있는 기록만 읽는 법
   마운트 0 의 증거           다른 프로세스의 mountinfo 를 읽을 수 있나.  못 읽으면 배타 잠금이 증거 (Q2 답)
   합치기 표                 종류가 바뀐 항목 한 줄 · merge.py 의 처리 확인 (FR-7)
   lower 상태 기계            전이마다의 잠금 · 낡은 상태 정리 · 시작 전 확인이 어긋났을 때의 상태
   두 시계                   한 줄에 어느 시계를 쓰나 (실행 계획 7절 ②)
   예산의 경계               Finalize 예산이 닫기까지 · 업로드 예산이 로그까지 (services 1절)
   IR 대조                   IR 칸 이름 · 환경 변수 이름 · 어긋났을 때의 원인 코드 (Units Generation 2.3)
   env check Fact 의 State   어긋났을 때 어느 State 로 적나
   CI 의 가짜 표시            ubuntu-latest 에서 한 번 더 잰다.  user xattr 이 없으면 실패 (계획 1.3)
```

### 6.3 NFR Requirements

```text
   N1   배경 삭제자가 한 번에 지우는 양과 속도
   N2   checkpoint 하나의 상한과 inode 한도
   N3   업로드 예산과 요청 사이의 관계 · 스트림의 상한
```

---

## 7. 정본과 팩에 되돌릴 것

### 7.1 정본 (`requirements.md` 11절에 더한다)

```text
   ADR-077 §5        「부모가 root 소유인 기계(SunnyVM 의 /srv)」 -> /work 별칭의 부모 / (계획 1.10)
   ADR-077 §4        확인 넷 중 「마운트 0」은 배타 잠금이 증거를 진다 — Functional Design 이 확인하면
   ADR-077 §5 · §11  IR 을 계약 칸으로 싣는다 · 형식 규칙의 자리 없음 (Units Generation 2.3)
   mediator-api.md   exited 절 제목의 경로를 표와 같게 (계획 1.8)
   run-contract.md   effect 값 넷 · budget · merge.wait · 구울 IR · discover (produce 는 순연)
```

### 7.2 `decisions.md` 에 더할 행과 고칠 문구

```text
   3-24   문구 — 「매칭된 순간부터 그 노드가 그 Run 의 임대를 놓을 때까지 lower 가 안 바뀐다」
          (첫 단계는 그 노드가 그 Run 에서 처음 claim 하는 단계였다.  계획 Q1 답)
   4-15   한 줄 — merge 대기의 노드 쪽 제한도 이 정책과 함께 연다 (계획 Q6 답)
```

옮기는 것은 팩을 고치는 쪽의 몫이다 — 이 문서는 행을 짓고 옮기지 않는다 (`stories.md`
3절과 같은 규칙).

---

## 8. Units Generation 이 고친 것 (2026-09-24)

Units Generation 의 답과 결정이 이 설계를 넷 고쳤다. 줄마다 「Units Generation 이 고쳤다」를
적었고, 여기에 한데 모은다. 근거는 `plans/unit-of-work-plan.md` 5절 · 2.2 · 2.3 이다.

```text
   결과 adapter 순연 (Q3 = A)
     FR-11 의 두 adapter 를 이번에 만들지 않는다.  그래서 아래 겉면은 만들지 않는다 —
     FinalizeSpec 의 Changeset · Produce 칸, ChangesetDescriptor, 계약의 produce 칸,
     광고 키 producer.<이름>.  2절 ⑪ 과 4절의 「결과 adapter 둘」 줄은 순연 행 4-16 으로 간다
   agent 단계의 전체 훑기를 끈다 (Q7 = A)
     4절 Worker agent 단계의 「오늘 그대로」가 바뀐다.  Discover 는 끄고 RecordDiff 는 둔다
   굽기 계약이 구울 IR 을 적는다 (2.3)
     ⑩ 의 ir_tag(규칙) · 기본 사내 형식을 거둔다.  계약이 IR 을 값으로 적고, 노드가 환경
     변수로 넘기고, sync 뒤 HEAD 와 대조한다.  ir_reason 이 없어진다
   계약 작성 도구 (2.2)
     cmd/runctl 을 「안 만지는 것」에서 한 파일(shape.go · lint 의 조건 제안) 뺀다.
     굽기 Run 의 성공 판정(success_when)을 계약 문법 유닛의 Functional Design 이 닫는다
