# 정본 — `enode-design` 과 이 팩의 연결

설계 정본은 이 저장소의 서브모듈 `enode-design/` 이다. **어긋나면
`enode-design/protocol/INVARIANTS.md` 가 이긴다.** 이 팩의 문서는 정본을
요약하거나 값을 채운 것이지 정본을 대체하지 않는다.

```text
   서브모듈 고정       enode-design @ ee13099  (origin/main)
   갱신                git submodule update --remote 는 진행자만.  갱신하면 이 문서의
                       표를 다시 본다
```

---

# 1. 이 팩이 딛는 정본

| 문서 | 상태 | 이 팩에서 쓰는 자리 |
|---|---|---|
| `protocol/INVARIANTS.md` | 정본 | §1.1 상태 어휘 · §1.2 유예 상태(`QUEUED` 유예 해제) · §2 전이표(`ALLOCATING -> QUEUED` 가 는다) · `I4` `I5` |
| `protocol/mediator-api.md` | 정본 | §4 `GET /v1/nodes` · §5 `GET /v1/runs?state=&since=&limit=` 자리 · §1.1 인증과 식별 |
| `ADR-015` 신원과 저장 | 확정 | §1 토큰은 인증 · 이메일은 식별. `principal` 에 권한을 걸지 않는다 |
| `ADR-012` 자기 광고 | 확정 | 광고는 매번 전부이고 만료된다. 정책은 그 옆의 다른 축 |
| `ADR-017` 워크스페이스는 노드 | 확정 | 결정 3 — 능력을 빼고 보내는 것이 「지금은 못 한다」 |
| `ADR-042` 경계는 노드 | 확정 | 제어 표면이 노드에 사는 근거 |
| `ADR-045` 아는 쪽이 적는다 | 확정 | `roles` 는 노드 역할. 역할 주입과 층이 다르다 (이월) |
| `ADR-060` 가지는 고른 것에 닿아야 | 확정 | `steps.chosen` — DB 에 있고 뷰에 없다. `StepView.chosen` 을 더한다 |
| `ADR-063` 노드는 무엇인지 · 소유자는 누가 쓸지 | **초안** | drain 의 근거. §4 미결을 `decisions.md` 1절이 닫았다 |
| `ADR-064` 대기열을 연다 · 선점은 안 연다 | **초안** | `QUEUED` 의 근거. §6 크래시 복구를 `decisions.md` 2절이 닫았다 |
| `ADR-065` 운용 관측은 배정 판정이 아니다 | 결정 | `GET /v1/nodes` 응답 모양. `principal` 안 냄 · 필터 없음 |
| `ADR-066` 도구는 입력의 모양을 말한다 | 결정 | `runctl shape` — 계약 작성의 어휘. 이 팩이 손대지 않는다 |

---

# 2. 정본이 코드와 어긋나 있는 자리

**`ADR-065` 는 「구현됨 `enode`」라고 적혀 있으나 이 저장소의 코드에
`GET /v1/nodes` 가 없다.** 라우트는 15개 그대로다. 구현은 정본의 병합되지
않은 가지에 있으나 **이 팩은 그 가지를 참조하지 않는다** — 이 저장소의
`main` 이 보는 것만 본다. 중앙 현황판은 `ADR-065` §2 의 응답 모양에서
새로 만든다.

**커밋을 글자로 박지 않는다.** 이 저장소는 정본에서 갈라져 나왔고 `upstream`
원격이 그것을 가리킨다. 「정본과 얼마나 떨어졌나」는 문서가 아니라
`git log HEAD..upstream/main` 이 답한다 — 문서에 적으면 그 줄이 조용히
낡는다.

---

# 3. 선행 조건 — 정본 쪽에 먼저 착지해야 하는 것

```text
   ADR-065 개정      GET /v1/nodes 노드 항목에 draining 필드를 더한다.
                     중앙 현황판의 draining 배지가 이 필드를 읽는다.
                     개정이 main 에 들어가기 전에는 배지의 근거가 기획뿐이다 —
                     결정이 문서에 없는 채로 화면이 앞서간 것이 이미 한 번 비용을 냈다

   ADR-063 개정      §2.1 「drain 은 Run 을 죽이지 않는다」를 graceful 에 한정하고,
                     at-boundary 는 「현재 단계를 끝내고 취소 경로로 닫는다. 산출은
                     Record 에 남는다」로 갈라 적는다 (decisions.md 1절, 사용자 결정
                     2026-09-04). 이 팩은 개정된 문장을 따른다

   ADR-063 · 064     실물을 만들기 전에 결정으로 올린다는 조건이 각 §6 · §8 에 있다.
                     decisions.md 가 그 미결에 값을 줬으므로 올릴 수 있다.
                     올리는 것은 enode-design 의 PR 이고 이 저장소의 일이 아니다
```

---

# 4. 이 팩이 정본에 더하는 것 (역방향 요약)

정본이 아직 말하지 않은 것을 이 팩이 값으로 정했다. 이 회차가 끝나면 정본으로
올린다.

```text
   ALLOCATING -> QUEUED 전이 · 202              decisions.md 1절
   drain 두 모드의 뜻 · 해제 · 통보 경로          decisions.md 1절
   GET /v1/runs 응답 모양                         decisions.md 2절
   QUEUED 크래시 복구                             decisions.md 2절
   호스트 제어판의 위치 · 바인딩 · 포트            decisions.md 1 · 2절
```
