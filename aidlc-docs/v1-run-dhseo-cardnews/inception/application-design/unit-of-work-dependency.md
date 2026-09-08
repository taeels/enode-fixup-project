# Unit of Work Dependency — v1-run-dhseo-cardnews

## 의존 매트릭스

| 유닛 | 의존하는 유닛 | 비고 |
|---|---|---|
| `cardnews-guest-login` | (없음) | 함대를 안 본다. CP0 뒤 아무 때나 착수 가능 |

이 실행이 유닛을 하나만 내므로 유닛 간 병렬 스케줄링은 적용되지 않는다.
`CONVENTIONS.md` 3.1 의 「의존 없는 유닛부터 병렬로 착수」 규칙은 다음에
3.1.x ~ 3.3.x 유닛이 이 팩에서 나올 때 이 유닛과 나란히 적용된다.

## 파일 행렬 — 다른(향후) 유닛과 만나는 자리

`constraints.md` 가 요구하는 파일 행렬이다. 이 실행 시점에는 3.1.x 등
다른 유닛이 아직 존재하지 않으므로, 아래는 **향후 유닛이 착수할 때
참조할 접점**을 미리 적어 둔 것이다(`decisions.md` 8.5).

| 파일 | 만지는 유닛 | 접점 처리 |
|---|---|---|
| `internal/api/ui/**` (패키지 자체) | `cardnews-guest-login` (이번 실행이 먼저 만든다) · 향후 3.1.1 유닛(자기 화면 S0~S5 를 더한다) | 디렉터리가 겹치지 않는다 — `cardnews-guest-login` 은 `static/{index.html, cardnews/, demo/, shared/}` 만 만진다. 3.1.1 은 `static/index.html` 의 관리자 로그인 placeholder 를 실제 동작으로 바꾸는 것과 자기 화면 파일(예 `static/dashboard/`)을 더하는 것 — **같은 파일을 동시에 고치는 것은 `static/index.html` 하나뿐**이라 그 한 파일만 사람이 직렬로 병합한다 |
| `internal/api/api.go` 의 `Handler()` 등록 줄 | `cardnews-guest-login` (`GET /ui/` 한 줄 추가) · 향후 3.1.1 유닛(같은 줄을 또 추가하려 할 수 있다) | **3.1.1 유닛은 이 줄을 새로 추가하지 않는다** — 이미 있으면 그대로 두고 자기 화면 파일만 `static/` 아래 더한다. 이 유닛이 먼저 병합되면 3.1.1 은 이 줄에 diff 를 내면 안 된다(사람이 리뷰에서 걸러낸다) |

이 유닛 자체는 위 표의 두 파일 모두에서 **유일한 저자**다 — 오늘 이
순간에는 부딪힐 다른 유닛이 없다. 표는 나중을 위한 것이다.

## 병렬로 실제로 돌았는가

아니다. 이 실행은 단일 실행자가 전 단계를 순서대로 돈다
(`aidlc-docs/v1-run-dhseo-cardnews/audit.md` 「실행 방식 적응」). 유닛이 하나뿐이라 병렬성을
측정할 대상 자체가 없다.
