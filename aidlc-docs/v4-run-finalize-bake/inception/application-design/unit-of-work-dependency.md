# 유닛 의존과 순서

**혼자 한 줄 순서로 만든다** (Q4 = B · 「병렬이어야 할 이유는 없다. 혼자 작업하기 때문에.」).
그래서 이 문서는 웨이브 표 대신 한 줄 순서를 적고, 사람이 확인하는 조각을 기다리는 동안 무엇을
먼저 할 수 있는지를 함께 적는다.

- **작성 시각**: 2026-09-24T12:14:10Z
- **병합 규칙**: 조각은 그 기능을 마지막으로 완성하는 유닛이 맡는다. 앞선 유닛은 코드 검사로
  병합한다 (Q2 = A · `unit-of-work.md` 0절)

---

## 1. 의존 행렬

세로가 기다리는 유닛, 가로가 먼저 있어야 하는 유닛이다.

| | 계약 문법 | 진행 구간 | 결과 확정 | trash | 합치기 규칙 | 아래층 상태 | 굽기 | 보존 |
|---|---|---|---|---|---|---|---|---|
| **1 contract-grammar** | — | | | | | | | |
| **2 step-phase** | 코드 | — | | | | | | |
| **3 finalize** | 코드 | 조각 | — | | | | | |
| **4 trash** | | | 파일 | — | | | | |
| **5 merge-rules** | | | | | — | | | |
| **6 lower-state** | | | 파일 | 코드 | | — | | |
| **7 bake** | 코드 | 조각 | 코드 | 코드 | 코드 | 코드 | — | |
| **8 checkpoint** | | | 코드 | 코드 | | 파일 | 파일 | — |

```text
   코드   그 유닛의 코드가 main 에 있어야 빌드가 된다.  그래서 그것이 main 에 들어간 뒤에 시작한다
   파일   같은 파일을 고친다.  먼저 시작해도 되지만 병합은 그 유닛 뒤에 한다
   조각   코드로는 안 기대지만, 그 유닛이 맡은 조각이 통과해야 이 유닛의 조각을 돌릴 수 있다
```

**칸마다의 이유**

```text
   step-phase  <- contract-grammar   merge 종류를 알아야 claim 이 phase = waiting 을 적는다
   finalize    <- contract-grammar   effect · 예산 · 명시 훑기의 문법
               <- step-phase (조각)  조각 2 가 종료 보고 라우트를 쓴다
   trash       <- finalize (파일)    세션 닫기(runc_overlay_linux.go)와 cmd/enode/main.go
   lower-state <- trash              drain 출처를 합치는 틀과 상태 파일의 칸
               <- finalize (파일)    단계 런타임의 능력 칸(runtime.go)
   bake        <- 다섯 (코드)        굽기 문법 · 결과 확정과 닫기의 행선지 · trash 와 대기 자리 ·
                                    합치기 규칙 · 아래층 상태와 잠금
               <- step-phase (조각)  merge 단계의 waiting 과 조각 2
   checkpoint  <- trash · finalize   같은 이름 바꾸기와 삭제자 · 닫기의 행선지와 결과 칸
               <- lower-state · bake (파일)   runtime.go · claim.go · runc_overlay_linux.go
```

**의존이 0 인 유닛은 둘이다** — `contract-grammar` 와 `merge-rules`. `merge-rules` 는 표준
라이브러리와 x/sys 만 쓰고 부르는 쪽을 모른다.

---

## 2. 한 줄 순서

```text
   1  contract-grammar   계약 문법            코드 검사
   2  step-phase         Mediator 진행 구간   조각 0 (기계)
   3  finalize           결과 확정            조각 1 · 3 (기계) · 조각 2 (사람 · 스크래치)
   4  trash              버리기와 여유 공간    조각 4 (사람 · SunnyVM)
   5  merge-rules        합치기 규칙          코드 검사 + 가짜 트리 재개 시험
   6  lower-state        아래층 상태와 잠금    코드 검사
   7  bake               굽기 단계            조각 5 (기계) · 조각 6 · 7 · 8 (사람 · SunnyVM)
   8  checkpoint         실패한 단계 보존      조각 9 (사람 · SunnyVM)
```

**이 순서를 고른 이유**

- **팩의 권장 순서를 따른다** (`decisions.md` 5절) — 결과 쪽(기능 1 ~ 3 과 4)을 먼저, 굽기
  쪽(5 ~ 9)을 다음에, 보존(10)을 마지막에. 「굽기가 effect · waiting · trash 에 기대고, 보존은
  trash 뒤다」
- **계약 문법이 가장 앞이다.** 노드와 Mediator 가 함께 쓰는 문법이라, 먼저 있어야 둘 다 빌드된다
- **merge-rules 가 bake 앞이다.** 재개 시험이 실제 아래층에 처음 합치기 전에 통과해야 한다
  (실행 계획 1.4 제약 둘째)
- **사내 증상을 고치는 두 유닛(finalize · trash)이 앞쪽 절반에 있다.** 명령 단계의 전체 훑기와
  보고 전 삭제가 사내에서 임대를 붙잡은 원인이었다

---

## 3. 시작 조건 (Q2 의 시작 규칙)

유닛은 **자기가 맡는 조각이 기대는 앞 조각이 통과했거나 보류일 때** 시작한다. 자기가 맡는 조각은
빼고 센다. 코드로 기대는 유닛은 그것이 `main` 에 들어간 뒤 시작한다.

| 유닛 | 코드로 기대는 것이 main 에 | 앞 조각이 통과 또는 보류 |
|---|---|---|
| contract-grammar | — | — |
| step-phase | contract-grammar | — |
| finalize | contract-grammar | 조각 0 |
| trash | — (finalize 와는 파일 순서만) | 조각 0 |
| merge-rules | — | — |
| lower-state | trash | — |
| bake | contract-grammar · finalize · trash · merge-rules · lower-state | 조각 1 · 2 · 4 |
| checkpoint | trash · finalize | 조각 4 |

**보류는 통과가 아니다** (`scene-gates.md` 4절). 보류인 조각을 맡은 유닛은 병합하지 않는다.
그래서 그 유닛에 코드로 기대는 유닛은 시작하지 못한다.

---

## 4. 사람 조각을 기다리는 동안 할 수 있는 것

SunnyVM 은 노트북 VM 이라 꺼져 있을 수 있다. 사람이 확인해야 병합할 수 있는 유닛은 넷이다.
그동안 한 줄 순서를 멈추지 않도록, 기다리는 유닛에 코드로 기대지 않는 다음 유닛을 먼저 진행한다.

| 기다리는 것 | 누가 | 그동안 할 수 있는 것 | 못 하는 것 |
|---|---|---|---|
| 조각 2 (스크래치 Mediator) | finalize | trash — 파일 순서만 겹친다. 병합은 finalize 뒤 | bake · checkpoint |
| 조각 4 (SunnyVM) | trash | merge-rules — 의존 0 | lower-state · bake · checkpoint |
| 조각 6 · 7 · 8 (SunnyVM) | bake | checkpoint — bake 와는 파일 순서만. 병합은 bake 뒤 | — |
| 조각 9 (SunnyVM) | checkpoint | Build and Test 의 조각 11 · 12 준비 | — |

**조각 4 가 가장 긴 대기를 만든다.** trash 에 코드로 기대는 유닛이 셋(lower-state · bake ·
checkpoint)이기 때문이다. 조각 4 는 SunnyVM 이 켜지는 첫 기회에 돌린다.

---

## 5. 조각이 통과하는 순서

```text
   step-phase 뒤      조각 0        기계 · 스크래치
   finalize 뒤        조각 1 · 3    기계 · 스크래치
                      조각 2        사람 · 스크래치 Mediator (Mediator 를 잠깐 멈춰 보고를 잃게 한다)
   trash 뒤           조각 4        사람 · SunnyVM
   merge-rules 뒤     조각 7 의 기계 부분 (가짜 트리 재개 시험)
   bake 뒤            조각 5        기계 · 스크래치
                      조각 6 · 7 · 8 사람 · SunnyVM · 버려도 되는 아래층
   checkpoint 뒤      조각 9        사람 · SunnyVM
   Build and Test     조각 11       사람 · 운영 host (유닛 없음)
                      조각 12       사람 · SunnyVM 또는 사내 · 조각 6 뒤 (유닛 없음)
   해당 없음           조각 10       adapter 순연 (보류가 아니다)
```

---

## 6. 실행 계획이 건 제약 둘

```text
   합치기 규칙이 자기 유닛인가            그렇다 — merge-rules.  상태 · 잠금 · drain 은 lower-state,
                                       굽기 단계는 bake 로 따로 있다.  하나를 되돌려도 다른 것은 남는다
   재개 시험이 실제 아래층보다 먼저인가     그렇다 — merge-rules(5번째)의 병합 조건이 가짜 트리 재개
                                       시험이고, 실제 아래층에 처음 합치는 조각 6 은 bake(7번째)가 맡는다
```

---

## 7. 되돌리기의 단위

```text
   굽기 단계를 물리려면           bake 만 되돌린다.  준비도 점검과 후보 잠금(lower-state)은 남는다
   준비도 점검을 물리려면         lower-state 를 되돌리려면 bake 도 함께다 (코드로 기댄다)
   보존을 끄려면                 checkpoint 만 되돌린다.  trash 와 삭제자는 남는다
   Mediator 를 물리려면           step-phase 만 되돌린다.  종료 보고를 못 받는 동안 노드의 단계는
                                running 에 머물고 결과 보고로 끝난다 (옛 노드와 같다)
   가장 되돌리기 어려운 것         실제 아래층에 적용된 합치기.  trash 를 지운 뒤에는 물릴 길이 없다.
                                그래서 합치기 조각은 버려도 되는 아래층에서만 돈다 (완료 조건 8)
```

---

## 8. 병합이 겹치는 파일

여러 유닛이 고치는 파일은 열하나다. 혼자 한 줄 순서로 하므로 동시에 부딪치지 않고, 위의 순서대로
병합하면 된다. 전체 목록과 순서는 `unit-of-work-file-matrix.md` 2절에 있다.

실행 계획이 짚은 다섯 중 셋(`contract.go` · `api.go` · `schema.sql`)은 **한 유닛만 고친다.**
계약 문법과 Mediator 를 각각 한 유닛에 모았기 때문이다.
