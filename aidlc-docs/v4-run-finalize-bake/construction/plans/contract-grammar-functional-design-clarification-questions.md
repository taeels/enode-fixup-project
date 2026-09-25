# `contract-grammar` — Functional Design 되물음

계획(`contract-grammar-functional-design-plan.md`)의 답 일곱을 받았다 (2026-09-25T03:56:21Z).

```text
   1 A   굽기 판정은 produced — build 는 manifest, merge 는 merged
   2 A   effect 와 budget 을 받는 종류는 엄격한 표
   3 A   굽기 계약은 build 와 merge 두 단계만.  다른 단계가 있으면 400
   4 A   칸 이름 ir · 환경 변수 ENODE_IR · 좁은 문자 집합
   5 A   "discover": true.  run · agent 만.  상한은 노드가 정한다
   6 B   계획도 굽기 단계를 지을 수 있다.  Grammar 가 여섯 종류를 모두 가르친다
   7 C   굽기 예시는 명령을 자리표시로 둔다
```

모호한 답은 없다. **답 둘이 함께 성립하지 않는 자리가 하나**, **답 하나가 시험에 걸리는
자리가 하나** 있다. 물음 셋으로 닫는다. 권장을 **A** 에 둔다.

---

## 되물음 1 — 답 3 과 답 6 이 부딪친다

계획은 계약 안의 계획 단계(`expands: true` 인 agent 단계)가 지어 **같은 계약에 붙인다**
(ADR-022 · 계획 위임). 붙인 뒤의 계약이 `Validate` 를 다시 받는다. 그래서 계획이 build 와
merge 를 지으면 계약은 「계획 단계 + build + merge」 셋 이상이 된다. 답 3 (다른 단계가 있으면
400)이 그 계약을 거절하므로, 답 6 (계획도 굽기 단계를 짓는다)이 쓰일 수 있는 경우가 없다.

A) **굽기 앞에 계획 단계를 허용한다.** 굽기 계약은 「계획 단계들 + build + merge」 다. 계획
단계들은 계획을 짓는 agent 단계(`expands`)와 그 계획을 사람이 승인하는 ask 단계뿐이다.
build 와 merge 는 계획이 짓거나 사람이 처음부터 쓴다. merge 뒤와 build · merge 사이에는
아무 단계도 없다. 그 밖의 섞기는 여전히 400 이다. 계약 작성자는 계획이 지을 단계 이름을
`produces` 로 미리 약속하고 `success_when` 에서 `manifest` · `merged` 를 건다 (ADR-049 —
계획이 반드시 지을 단계 이름을 미리 적는 칸)

B) **답 3 을 B 로 바꾼다.** 굽기 계약에 다른 단계를 섞을 수 있다. build 와 merge 사이에 같은
역할(uses)의 단계만 못 온다

C) **답 6 을 A 로 바꾼다.** 계획은 굽기 단계를 못 짓는다. Grammar 는 굽기를 가르치지 않는다

D) Other (please describe after [Answer]: tag below)

[Answer]: A — 사용자 「권장대로.」 (2026-09-25)

---

## 되물음 2 — 계획이 지은 굽기를 사람이 승인해야 하나

되물음 1 이 C 가 아니면 쓰인다. 계획은 사람의 승인(ask 단계의 `adopts`)을 거쳐 붙거나,
`adopt: "yolo"` 로 묻지 않고 붙는다 (ADR-061 — 계획을 묻지 않고 채택하는 선언). 굽기는
공용 lower 에 합치므로 되돌리기 어렵다 (실행 계획의 위험도 High).

A) **굽기 단계를 짓는 계획은 사람이 승인해야 붙는다.** 계획이 build 나 merge 를 지었는데 그
계획 단계가 `adopt: "yolo"` 이면 붙일 때 거절한다. ask 의 `adopts` 로 사람이 본 계획만 붙는다

B) **다른 계획과 같다.** `yolo` 로도 붙는다. 계약 작성자가 `yolo` 를 적은 것이 곧 사람의
결정이다 (ADR-061 의 논리)

C) Other (please describe after [Answer]: tag below)

[Answer]: A — 사용자 「권장대로.」 (2026-09-25)

---

## 되물음 3 — 자리표시 예시가 예시 시험을 통과하려면

답 7 = C 로 `sync` 와 `builds[].command` 는 `"<your sync command>"` 같은 자리표시가 된다.
그런데 예시 시험(`TestExamples_ParseAndValidate`, `internal/contract/example_test.go:14`)은 모든
예시에 `Validate` 를 돌린다. `ir` 은 답 4 의 문자 집합을, 구성 이름은 `[a-z0-9-]` 를 지켜야
하므로 `<` 와 빈칸을 쓸 수 없다. 명령 칸은 문자열이면 무엇이든 된다.

A) **`ir` 과 이름은 규칙을 지키는 자리표시로 쓴다.** `"ir": "your-ir-tag"`, 구성 이름은
`"config-a"` · `"config-b"`. 명령만 `<...>` 자리표시다. 예시 시험을 고치지 않는다

B) **모두 `<...>` 로 쓰고 예시 시험에서 굽기 예시만 `Validate` 를 건너뛴다.** 보기에는
자리표시가 분명하지만, 예시가 실제 문법을 어겨도 시험이 못 잡는다

C) Other (please describe after [Answer]: tag below)

[Answer]: A — 사용자 「권장대로.」 (2026-09-25)

---

## 답이 bake 유닛에 넘기는 것 하나

답 7 = C 라 예시가 poky 를 쓰지 않는다. 그래서 계획 1.4 의 「`.repo` 가 없으면 워크스페이스
git 의 HEAD 태그를 본다」를 이 유닛이 정하지 않는다. IR 대조는 계획 2.3 대로
`.repo/manifests` 의 HEAD 를 본다. **SunnyVM 의 시험 lower 는 poky 를 git 으로 받았다** —
조각 5 ~ 8 (굽기 장면의 검증 단위)을 그 lower 에서 돌리려면 repo manifest 로 감싸거나 대조가
git 을 알아야 한다. bake 유닛의 Functional Design 이 닫는다.
