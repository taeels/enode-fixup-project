# aidlc-docs — AI-DLC 산출물이 여기 쌓인다

**지금은 비어 있다. 이것이 정상이다.**

이 저장소는 브라운필드다 — 코드가 이미 있고 설계 정본이 `enode-design/` 에
있다. AI-DLC 는 **Workspace Detection** 으로 그 사실을 확인하고 **Reverse
Engineering** 부터 돈다. 그 산출물이 이 디렉터리의 첫 내용이 된다.

**낡은 산출물을 미리 실어 두지 않았다.** 실어 두면 그 단계를 실행한 것이
아니라 물려받은 것이 되고, 물려받은 것은 코드가 움직인 만큼 조용히 거짓이
된다. 앞선 회차에서 실제로 그렇게 됐다 — 스캔 기준선과 작업 기준선이 마흔여덟
커밋 벌어진 채로 산출물 아홉을 물려 썼고, 그중 셋이 통째로 거짓이 되어 있었다.

## 무엇이 어디에 쌓이나

`.aidlc/aidlc-rules/` 의 규칙이 자리를 정한다. 요약하면 이렇다.

```text
   aidlc-docs/
   ├── inception/
   │   ├── plans/
   │   ├── reverse-engineering/     브라운필드의 첫 산출물
   │   ├── requirements/
   │   ├── user-stories/
   │   └── application-design/
   ├── construction/
   │   ├── plans/
   │   ├── <유닛>/
   │   │   ├── functional-design/
   │   │   ├── nfr-requirements/
   │   │   ├── nfr-design/
   │   │   ├── infrastructure-design/
   │   │   └── code/
   │   └── build-and-test/
   ├── operations/
   ├── aidlc-state.md               단계 진행
   └── audit.md                     사용자 입력과 결정의 기록
```

**애플리케이션 코드는 여기 안 들어온다.** 저장소 뿌리에 들어간다.

## 요구 팩은 이미 있다

`requirements/` 의 여섯 파일과 `design/` 이 이 회차가 만들 것의 요구와
화면이다. AI-DLC 의 Requirements Analysis 는 그것을 **입력으로** 읽는다.
