# User Stories Assessment — 굽기 (v4-run-finalize-bake)

`user-stories.md` Step 1 이 요구하는 판정이다. `requirements.md` 9절이 「돈다」고
적었고, 여기는 그 근거와 **이 단계가 무엇을 재야 값이 나오는지**를 적는다.

---

## Request Analysis

```text
   원래 요청     결정론 빌드가 끝나면 밖에서 곧바로 보이고 임대 창이 upper 크기와 무관하다.
                하루치 굽기가 형제를 멈추지 않고 lower 에 합쳐진다.  실패한 단계를 보고 뒤에
                들여다본다 (requirements.md 1.2)
   사용자 영향   Direct.  진행 조회에 phase 가 생기고, 계약 문법이 늘고(effect · 예산 · 굽기),
                노드가 스스로 drain 하고, receipt 에 checkpoint_capture 가 실린다
   복잡도        Complex.  lower 상태 넷 · 잠금 둘 · 재개 · 사람 조각 여덟
   이해관계자    노드 소유자 · 계약 작성자 · 진행자.  굽기를 내는 사람을 따로 셀지는 계획의 Q1
```

## Assessment Criteria Met

**High Priority — 넷이 걸린다.**

- [x] **New User Features** — 굽기 계약(`sync` · `builds[]` · merge)과 checkpoint 는
      사람이 새로 쓰고 새로 읽는 표면이다
- [x] **User Experience Changes** — 계약을 안 고친 사람의 경험도 바뀐다. 명령 단계가
      `workspace.diff` 를 더 안 내고, 여유가 모자란 노드는 빌드 능력이 아니라 통째로
      빠진다
- [x] **Multi-Persona Systems** — 굽기가 형제를 drain 시키는 순간 한 사람의 굽기가
      다른 사람의 Run 을 기다리게 한다
- [x] **Complex Business Logic** — committed · building · pending · merging 과 재개,
      `bake_in_progress` · `merge_wait_timeout` 이 서로 다른 시나리오다

**Medium Priority — 하나가 더 걸린다.**

- [x] **Testing** — 조각 열셋 중 여덟이 사람 조각이다. 사람이 보는 것이 곧 판정이다

## Skip 사유에 걸리는 것이 있나

**없다.** 순수 리팩터링도, 고립된 버그 수정도, 사람에게 안 보이는 인프라 변경도 아니다.

---

## Decision

**Execute User Stories**: Yes

**Depth**: minimal — 앞 두 회차와 같은 태도다. `scene-gates.md` 의 조각 0 ~ 12 와
`requirements.md` 6절이 수용 기준을 실행 명령으로 이미 적었다. 스토리가 행복 경로를
다시 쓰면 약한 사본이 되고, 갈리는 날 어느 쪽이 이기는지 정할 자리가 없다.

## 조각 열셋이 안 재는 자리

**이 팩은 기다림을 만든다.** 새 상태의 대부분이 「무엇이 아직 안 된다」다. 조각을
하나씩 대 보니 **상태가 옳게 들어가고 나오는지는 거의 다 재는데, 기다리게 된 사람이
그것이 어느 기다림인지 아는지는 거의 안 잰다.**

```text
   새 기다림 · 부재              기다리는 사람        상태 전이를     그 사람이 이유를
                                                   재는 조각       아는지를 재는 조각
   finalizing                   계약 작성자           2 · 3         2  (phase · exit)
   merge 단계 waiting            굽기를 낸 사람         6             0  누가 lower 를 쥐고 있나
   형제의 drain (pending · merging)  형제에 낼 계약 작성자   6 · 8         0  Run 은 QUEUED 뿐이다
   여유 부족 drain               노드 소유자           4             0  소유자 drain 과 갈리나
   not ready (st_dev 등)         노드 소유자           8             8  (사유를 이름으로)
   bake_in_progress              굽기를 낸 사람         8             8  (사유 코드)
   재개로 합친 lower               굽기를 낸 사람         7             0  Run 은 FAILED 인데 lower 는 새 ir
   ir 이 null                    ir 을 요구하는 작성자    6             0
   trash 가 아직 안 비었다          노드 소유자           4             0  디스크가 왜 찼나
   captured                     계약 작성자           9             0  보존본이 어디에 있고 누가 여나
```

실측 셋이 이 표를 받친다.

- **QUEUED 에 사유가 없다.** Mediator 는 drain 중인 노드를 점유된 노드와 같은 편으로
  합친다(`internal/store/queue.go:233` ~ `:239`). 진행 조회의 `StepView` 에 대기 사유
  칸이 없다(`internal/store/observe.go:21`). 형제가 굽기 때문에 빠진 것과 그냥 바쁜
  것이 밖에서 같다
- **Record 와 lower 가 갈리는 것은 정본이 이미 안다.** ADR-077 §7 — 「Record 는 merge
  단계 실패를 말하는데 lower 는 새 `ir` 에 선다」. 그 둘을 잇는 것이
  `.enode-metadata.json` 의 `resumed` 와 원래 Run 이다. **그 파일은 노드 기계 안에
  있다.** 굽기를 낸 사람이 FAILED 인 Run 에서 출발해 그것에 닿는 길을 재는 조각이 0 이다
- **노드 소유자가 받을지 정하는 자리가 없다.** 정책 파일에서 데몬이 읽는 것은 `drain`
  하나다(`internal/enode/policy.go:22` ~ `:29`). 팩은 누가 굽기를 낼 수 있는지 적지
  않는다. 계획의 Q5 가 이것을 묻는다

## 둘째 빈자리 — 계약을 안 고친 사람의 어제와 오늘

기다림과 다른 갈래가 하나 더 있다. **아무것도 안 고친 사람의 동작이 바뀐다.** 조각은
새 동작이 도는지를 재고, 그 사람이 바뀐 줄 아는지는 안 잰다.

```text
   누가           어제                                  오늘                                  재는 조각
   계약 작성자     명령 단계가 workspace.diff 를 낸다      effect 기본 build/test.  안 낸다        1  (안 걷는다만)
                 (changed.go:36 「사람이 리뷰한다」)
   노드 소유자     여유가 모자라면 arch 키만 빠진다        노드 전체가 drain.  agent 도 안 받는다    4  (빠졌다 돌아온다만)
   노드 소유자     scratch 를 다른 filesystem 에 둬도 ready  not ready                              8  (not ready 만)
   노드 소유자     끝난 단계의 runRoot 가 곧 지워진다        trash 에 남았다가 보고 뒤 지워진다        4
   노드 소유자     (없음)                                 실패한 단계의 upper 가 48시간 남는다       9  (capture 가 도는지만)
                                                        기본 정책 on-failure (decisions 2-10)
```

**마지막 줄이 앞 회차의 US-3 과 같은 모양이다** — 내 기계에 무엇이 남는지를 노드
소유자가 모르면, 안 적은 잔여와 같다.

---

## Expected Outcomes

```text
   ①  조각이 안 재는 자리를 이름으로 만든다.  그 자리가 새 완료 조건이 된다
   ②  Units Generation 이 스토리와 완료 조건을 유닛에 앉힐 때 배정 안 된 것이 0 임을 센다
   ③  requirements.md 8절이 Application Design 에 넘긴 ④ · ⑥ 에 사람 쪽 기준을 준다 —
      「광고 응답에 보이는 모양」과 「누락 산출물이 남는 자리」가 누구에게 무엇을
      말해야 하는가
```
