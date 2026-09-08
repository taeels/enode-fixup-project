# aidlc-docs — AI-DLC 산출물이 여기 쌓인다

이 저장소는 브라운필드다 — 코드가 이미 있고 설계 정본이 `enode-design/` 에
있다. AI-DLC 는 **Workspace Detection** 으로 그 사실을 확인하고 **Reverse
Engineering** 부터 돈다. 그 산출물이 `inception/reverse-engineering/` 이고,
아래 §1 이 설명하듯 이것만 회차를 가리지 않고 공용이다.

**낡은 산출물을 미리 실어 두지 않았다.** 실어 두면 그 단계를 실행한 것이
아니라 물려받은 것이 되고, 물려받은 것은 코드가 움직인 만큼 조용히 거짓이
된다. 앞선 회차에서 실제로 그렇게 됐다 — 스캔 기준선과 작업 기준선이 마흔여덟
커밋 벌어진 채로 산출물 아홉을 물려 썼고, 그중 셋이 통째로 거짓이 되어 있었다.

## 1. 문서는 사람이 아니라 회차(브랜치) 이름으로 나뉜다 (2026-09-08 부터)

`CLAUDE.md` 의 "AI-DLC 문서 루트는 `aidlc-docs/<브랜치 이름>/`" 규칙이
정본이다 — **새 회차부터** 적용한다. 요약하면 이렇다.

```text
   aidlc-docs/
   ├── inception/
   │   └── reverse-engineering/     공용.  회차를 안 가린다 (아래 §1.1)
   ├── <브랜치 이름>/                새 회차부터.  예: v2-run-<이름>
   │   ├── inception/
   │   │   ├── plans/
   │   │   ├── requirements/
   │   │   ├── user-stories/
   │   │   └── application-design/
   │   ├── construction/
   │   │   ├── plans/
   │   │   ├── <유닛>/
   │   │   │   ├── functional-design/
   │   │   │   ├── nfr-requirements/
   │   │   │   ├── nfr-design/
   │   │   │   ├── infrastructure-design/
   │   │   │   └── code/
   │   │   └── build-and-test/
   │   ├── operations/
   │   ├── aidlc-state.md           이 회차의 단계 진행.  진행자만 고친다
   │   └── audit.md                 이 회차의 사용자 입력과 결정의 기록
   └── <다른 브랜치 이름>/           그다음 회차가 여기 새로 쌓는다
```

**이번 회차(`v1-run-dhseo-cardnews`)의 기존 산출물은 옮기지 않았다** —
`inception/` · `construction/` · `aidlc-state.md` · `audit.md` 가 지금도
평평한 자리에 있다. 이미 착지한 산출물을 규칙이 바뀌었다고 소급해서
옮기면, 그 옮김 자체가 이번 회차의 스코프 밖 diff 가 된다. 이 규칙은
**다음에 새로 여는 회차**부터 적용된다.

**애플리케이션 코드는 여기 안 들어온다.** 저장소 뿌리에 들어간다.

### 1.1 왜 Reverse Engineering 만 공용인가

RE 산출물은 **코드베이스에 대한 관측**이라 어느 회차가 실행하든 같은
질문("이 함수는 무엇을 하나")에 같은 답이 나와야 한다. 회차마다 새로
스캔하면 실측이 갈라져도 아무도 모른다 — 공용 자리 하나에 두고, 코드가
크게 움직였을 때만(다음 브라운필드 스캔에서) 갱신한다.

Inception 의 나머지(requirements · user-stories · application-design)와
Construction 전부는 **그 회차가 무엇을 만들기로 했는가의 기록**이라
회차마다 다르다 — 사람이 아니라 회차로 나누는 이유는, 같은 사람이 여러
회차를 겹쳐 돌릴 수 있고(동시에 다른 기능을 진행) 회차가 병합·폐기되는
단위이기 때문이다.

### 1.2 `aidlc-state.md` · `design/*.pen` 은 진행자 한 사람만 고친다

둘 다 **통째로 다시 쓰거나 자동으로 합칠 수 없는 파일**이다.
`aidlc-state.md` 는 그 회차의 현재 단계를 요약하는 스냅샷이고, `.pen`
은 그림 도구의 바이너리에 가까운 형식이라 diff 가 안 갈린다. 그래서:

```text
   병합 뒤 정리      unit/<유닛> 이 자기 몫을 적어 올리고, 진행자가
                    병합한 뒤 aidlc-state.md 로 옮긴다 (CONVENTIONS.md 3.2)
   시안은 새 파일    design/ 의 화면 시안을 고칠 때는 기존 .pen 을
                    고치지 않고 회차마다 새 .pen 파일로 낸다 — 예:
                    enode-ux.pen(정본) 과 enode-cardnews.pen(이번 회차)
```

`audit.md` 는 다르다 — **이어 붙이는 로그**라 여러 유닛이 같은 파일에
써도 병합이 가능하다. `.gitattributes` 의 `merge=union` 이 그 병합을
git 에게 맡긴다.

## 요구 팩은 이미 있다

`requirements/` 의 여섯 파일과 `design/` 이 이 회차가 만들 것의 요구와
화면이다. AI-DLC 의 Requirements Analysis 는 그것을 **입력으로** 읽는다.
