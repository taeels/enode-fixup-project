# Requirements 확인 질문 — 굽기 (v4-run-finalize-bake)

팩의 `decisions.md` 는 **값을 거의 다 닫았다.** 1 ~ 3절의 마흔여섯 줄이 사용자 결정이고
6절이 구현 계획에 넘긴 것을 이름으로 적는다. 그 값들은 다시 묻지 않는다.

그런데 Reverse Engineering 이 팩의 전제와 어긋나는 측정값 여섯을 냈다
(`aidlc-state.md` 의 ① ~ ⑥). 그것을 오늘 코드에 한 번 더 댔다. **여섯 중 하나만 값이
아니라 결정이다** — 팩의 한 문장이 코드에서 두 쪽으로 갈라져 어느 쪽을 살릴지 사람이
골라야 한다. 나머지 다섯은 팩의 뜻 안에서 닫히므로 이 문서 끝의 「확인된 사실」에
적는다 (게이트 규칙 — 권장을 벗어나면 근거를 적는다).

그 밖에 묻는 것은 운영 둘과 AI-DLC 확장 셋이다.

```text
   질문 1        범위     min_free_gb 가 노드를 광고에서 빼는가, 빌드 능력만 빼는가
   질문 2        운영     이 회차를 어디까지 도나, 몇 손인가
   질문 3        운영     합쳐 둔 실행 환경 구현이 main 에 어느 길로 가나
   질문 4 ~ 6    확장     security-baseline · resiliency-baseline · property-based-testing
```

각 질문의 `[Answer]:` 뒤에 글자를 적는다. 맞는 것이 없으면 X 를 고르고 설명을 적는다.

---

## Question 1 — 여유가 `min_free_gb` 아래면 무엇이 빠지나

팩의 기능 4 와 결정 2-6 은 한 문장이다 — 「trash 가 쌓여 여유가 `min_free_gb` 아래면
노드가 광고에서 빠진다. **새 기제는 없다.** 이미 있는 광고 조건이다」.

오늘 코드에서 그 둘이 함께 서지 않는다.

```go
// internal/enode/detect.go:82 — 탐지 주기(기본 5분)마다
if archs := detectArchs(l); len(archs) > 0 {
    if ok, free := hasRoom(l, log); ok {   // :387  l.Workspace 의 여유만 잰다
        attrs["arch"] = archs[0]
        for _, a := range archs { attrs[archPrefix+a] = "yes" }
    } else {
        log.Warn("not enough free disk; dropping build capability ...")
    }
}
```

```text
   빠지는 것      arch 와 arch.<이름> 키뿐이다.  노드는 광고에 그대로 있다
   안 빠지는 것   arch 를 요구하지 않는 계약.  agent 단계와 대부분의 명령 단계가 그렇다
   재는 자리      워크스페이스.  scratch 는 안 잰다
                  (기능 8 의 env check 가 둘의 st_dev 일치를 강제하므로
                   이 팩 뒤에는 같은 filesystem 이다.  이 절반은 저절로 닫힌다)
```

그래서 「광고에서 빠진다」를 살리려면 기제가 하나 생기고, 「새 기제는 없다」를 살리면
빠지는 것이 빌드 능력뿐이다.

광고를 아예 멈추는 길은 없다. 광고가 곧 하트비트이고 임대 갱신이라(ADR-016) 멈추면
그 노드가 쥔 Run 의 임대가 끊긴다.

A) **노드가 스스로 graceful drain 을 싣는다** — 여유가 하한 아래면 광고의
`policy.drain` 에 `graceful` 을 싣고, 되찾으면 푼다. 굽기의 pending · merging 이 형제를
drain 시키는 길(결정 3-10)과 **같은 길**이다. 새 광고 어휘도 Mediator 변경도 없고,
새 것은 「노드가 스스로 거는 drain 의 계기」 하나다. 조각 4 의 「광고에서 빠졌다가
돌아온다」가 글자 그대로 선다. 소유자가 정책 파일로 건 drain 과 겹치면 센 쪽을 따른다
(어느 쪽이 센지는 Functional Design 이 닫는다). 팩 문장은 「새 기제는 없다」가 「새 계기
하나」로 고쳐진다. (권장)

B) **오늘 기제 그대로 — 빌드 능력만 뺀다** — 코드 0 줄이다. 팩 문장을 「빌드 능력이
광고에서 빠진다」로 고친다. 대가는 arch 를 요구하지 않는 계약이 계속 이 노드에 배정되는
것이다. trash 가 디스크를 채우는 동안 그 단계가 `ENOSPC` 로 실패할 수 있고, 조각 4 는
「arch 키가 빠졌다가 돌아온다」를 재도록 고친다.

X) Other (please describe after [Answer]: tag below)

[Answer]: A. "arch" 광고가 빌드 기능과 강하게 결합되어 있나보네. 극 초반 설계에서 arch를 넣어놨기 때문에 그것이 퍼진 것이 아닌가 싶은데 이제 이런 단순 조건문은 제거되어야 한다.

---

## Question 2 — 이 회차를 어디까지 도나, 그리고 몇 손인가

앞 두 회차는 **B** 로 답했고 유닛이 그렇게 `main` 에 들어갔다. 이 팩은 더 넓다.

```text
   기능          열셋 (코드 열하나 · 확인과 문서 둘 — 기능 12 · 13)
   조각          열셋.  사람이 보는 조각 여덟 (2 · 4 · 6 · 8 · 9 · 10 · 11 · 12)
   실측 자리     SunnyVM(노트북 VM — 꺼져 있을 수 있다) · 조각 11 은 사내 운영 host
   정본 상태     ADR-075 결정 · ADR-076 초안 · ADR-077 초안
```

A) **Inception 을 끝까지 돌고 멈춘다** — Units Generation 까지 내고 유닛 분해와 파일
행렬을 본 뒤에 배정과 범위를 정한다. Construction 착수는 별도 승인이다.

B) **Inception 을 돌고 그대로 Construction 까지 한 손(taeels)으로** — 앞 두 회차와
같은 모양이다. 접점 충돌이 없고 병합 순서가 곧 일정이다. 사람이 보는 조각은 사용자가
SunnyVM 과 사내 host 에서 돌린다. 범위를 자를 자리는 B 에서도 남는다 — Units
Generation 승인이 그 자리다. (권장)

C) **지금 담당을 여럿으로 정한다** — 유닛 분해가 나오는 즉시 `construction-roster.md`
에 행을 더한다. 굽기(기능 5 ~ 9)와 결과 좁히기(기능 1 ~ 3)가 접점 파일
`internal/enode/claim.go` · `runc_overlay_linux.go` 를 함께 만지므로 직렬 병합이 는다.

X) Other (please describe after [Answer]: tag below)

[Answer]: B.

---

## Question 3 — 합쳐 둔 실행 환경 구현은 `main` 에 어느 길로 가나

이 회차 브랜치는 `unit/runtime-environment-profile` 을 합쳐 두었다(`b5659ae`). 회차
README 가 이 선택을 열어 두었다 — 「먼저 `main` 에 병합하는 쪽을 고르면 이 합치기를
되돌리고 `main` 을 합친다」.

```text
   origin/main..origin/unit/runtime-environment-profile    19 커밋
   그 브랜치의 PR                                          없다 (gh pr list, 2026-09-23)
   enode-design 369270a                                    enode-design 의 같은 이름 브랜치에만 있다.
                                                           ADR-075 ~ 077 이 enode-design main 에 없다
```

`CONVENTIONS.md` 3.1 은 회차 브랜치를 **Inception 이 닫히면** PR 로 `main` 에 올린다.
그때 이 합치기가 그대로면 Inception PR 이 실행 환경 구현 전부를 싣고 간다.

A) **실행 환경 브랜치를 먼저 자기 PR 로 `main` 에 올린다** — enode-design 의 같은 이름
브랜치도 enode-design `main` 으로 먼저 간다(`decisions.md` 7절 끝줄). 그 뒤 회차 브랜치가
`main` 을 합치고, 회차의 Inception PR 에는 이 회차의 문서만 실린다. 코드가 유닛 PR 로
`main` 에 닿는다는 3.3 의 길 그대로다. 병합 커밋으로 병합하면 되돌릴 것이 없고, squash 로
병합하면 README 의 되돌리기가 필요하다. (권장)

B) **회차의 Inception PR 에 실어 보낸다** — 한 번에 끝난다. 대가는 19 커밋의 코드가 유닛
PR 이 아니라 Inception PR 로 `main` 에 닿는 것이다. 이 회차의 게이트는 그 코드를 조각 0
의 회귀로만 재고, 그 브랜치 자신의 게이트 판정은 이 PR 에 안 보인다.

X) Other (please describe after [Answer]: tag below)

[Answer]: 병합을 안 하면 회차 진행에 문제가 되는지?

---

## Question 4 — Security Extensions

이 프로젝트에 security extension 규칙을 강제하는가?

팩이 보안 표를 가지고 있다 — 종료 보고의 발신자 대조, trash 삭제의 경계, 합치기의
filesystem 경계, 상태 자리와 spool 의 권한, 계약이 낸 명령의 격리. 앞 회차들이 켰고
`decisions.md` 5절이 그 선택을 잇기를 권한다.

A) Yes — SECURITY 규칙 전부를 차단성 제약으로 강제한다 (production 급 애플리케이션에 권장). (권장)

B) No — SECURITY 규칙을 모두 건너뛴다 (PoC · 시제품 · 실험 프로젝트에 맞다)

X) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Question 5 — Resiliency Extensions

이 프로젝트에 resiliency baseline 을 적용하는가?

**이 확장이 무엇인가.** 켜면 회복력 있는 시스템을 짓기 위한 **방향성 있는 설계 시점
모범 사례**를 적용한다. **AWS Well-Architected Framework(Reliability Pillar)** 와 회복력
검토 지침에서 왔다. 요구 · 설계 · 코드를 장애 내성 · 고가용성 · 관측 가능성 · 복구
가능성 쪽으로 이끌고, 사업 목표 · 변경 관리 · 관측 · 고가용성 · 재해 복구 · 지속 개선에
걸친 실천 영역 15개를 다룬다.

**이 확장이 무엇이 아닌가.** 켠다고 워크로드가 production 준비가 되지 않고, 어떤 가용성
· RTO · RPO 목표도 인증하거나 보장하지 않는다. 좋은 회복력 결정을 일찍 세우는 **출발점**
이지 지어진 시스템에 대한 정식 **AWS Well-Architected Review** 를 대신하지 않는다.

결과물은 검증하고 다듬을 **회복력 자세의 첫 초안**으로 다룬다.

이 팩에 대한 판단 — 회복력이 이 팩의 본문이다(끊긴 합치기의 재개, 재시작 때의 조정,
데몬 시작 때 trash 비우기, merge 대기 상한). 그 요구는 이미 팩이 값으로 적었다. 이
기준선의 나머지 영역(다중 가용 영역 · 재해 복구 · RTO/RPO)은 Mediator 하나와 기계 안
lower 하나인 이 구성에서 대부분 해당 없음으로 떨어진다.

A) Yes — resiliency baseline 을 방향성 있는 모범 사례와 설계 시점 지침으로 적용한다 (업무상 중요한 워크로드에 권장. go-live 전에 검증하고 다질 출발점으로)

B) No — resiliency baseline 을 건너뛴다 (PoC · 시제품 · 빠른 반복이 신뢰성보다 중요한 실험 프로젝트에 맞다). (권장 — 위 판단)

X) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Question 6 — Property-Based Testing Extension

이 프로젝트에 property-based testing(PBT) 규칙을 강제하는가?

이 팩에서 PBT 가 가장 맞는 자리는 합치기다 — 「아무 트리 · 아무 지점에서 끊어도 다시
돌린 결과가 한 번에 끝낸 결과와 같다」가 그대로 성질이다. 팩은 그것을 **고정된 가짜
트리 위의 전수 시험**(표시 종류 전부 · 1 ~ 30번째 연산 뒤 강제 종료)으로 이미 요구한다
(`decisions.md` 5절).

켜면 드는 비용 — 규칙 PBT-09 가 shrinking 을 지원하는 PBT 프레임워크를 **프로젝트
의존성으로** 요구한다. 표준 라이브러리의 `testing/quick` 은 shrinking 이 없다. 이
저장소의 직접 의존은 셋(`pgx` · `x/sys` · `yaml.v3`)이다. 그리고 A 면 모든 유닛의 설계
단계가 성질을 식별해야 한다(PBT-01).

합치기에만 걸고 싶으면 X 를 고르고 그렇게 적는다.

A) Yes — PBT 규칙 전부를 차단성 제약으로 강제한다 (업무 로직 · 데이터 변환 · 직렬화 · 상태 있는 컴포넌트를 가진 프로젝트에 권장)

B) Partial — 순수 함수와 직렬화 왕복에만 PBT 규칙을 강제한다 (알고리즘 복잡도가 제한된 프로젝트에 맞다)

C) No — PBT 규칙을 모두 건너뛴다 (단순 CRUD · UI 전용 · 업무 로직이 거의 없는 얇은 통합 층에 맞다). (권장 — 합치기의 성질은 팩의 전수 시험이 지고, 그 시험이 종류가 바뀐 항목까지 담도록 사실 7 이 요구를 더한다)

X) Other (please describe after [Answer]: tag below)

[Answer]: X. 기존 테스트 컨벤션을 따른다.

---

# 확인된 사실 — 묻지 않고 적는 것

R/E 의 측정값 여섯 가운데 질문 1 로 간 ① 을 뺀 다섯과, 이번에 팩을 코드에 대며 더 찾은
넷이다. **모두 팩의 뜻 안에서 닫히거나 설계 단계가 닫을 자리라 묻지 않는다.**
`requirements.md` 가 이것을 요구로 옮긴다.

## 사실 1 — 종료 보고는 인스턴스까지 대조한다. result 는 오늘 그대로다 (②)

```go
// internal/store/claim.go:782 — ReportStep
UPDATE steps SET state=$4, ended_at=now(), result=$5
 WHERE run_id=$1 AND seq=$2 AND node_id=$3 AND state='CLAIMED'
```

result 는 노드만 본다. 팩의 보안 표는 종료 보고에 「claim 한 노드와 **인스턴스**만」을
요구하고 「인증은 다른 노드 표면과 같다」고 적는데, 다른 표면에 인스턴스 대조가 없다.

대조할 재료는 이미 있다 — `steps.claimed_instance` 를 claim 이 채운다
(`internal/store/claim.go:321`). 그리고 결정 1-9 가 종료 보고의 키를 Run · seq · attempt ·
노드 instance 로 정했으므로 본문이 인스턴스를 싣는다.

교정 — **종료 보고는 노드와 `claimed_instance` 를 함께 대조하고, result 는 바꾸지
않는다.** result 를 넓히는 것은 팩 밖이고, 재시작한 노드의 옛 인스턴스는 ADR-030 이
Run 을 이미 실패로 돌려 `state='CLAIMED'` 조건에서 걸린다. 두 표면의 비대칭은 잔여로
적는다.

## 사실 2 — Mediator 가 `internal/environment` 를 링크한다 (③)

`internal/store/claim.go` 가 `execenv.Record` 타입 하나 때문에 `internal/environment` 를
import 한다. 그래서 Mediator 바이너리에 profile · `env check` · `env apply` 코드가
들어가고, `internal/panel/boundary_test.go` 의 금지 표에 그 패키지가 없다.

이 팩은 노드 쪽에 lower 상태 · 잠금 · 합치기 · trash 삭제자 · spool 을 더한다.
**그 코드는 Mediator 가 링크하는 패키지에 두지 않는다** — 자리는 Application Design 이
정하고, 새 패키지면 경계 시험의 금지 표에 올린다. 이미 있는 `store -> environment` 한
줄을 끊을지는 이 팩 밖이다.

## 사실 3 — `internal/enode` 의 커버리지 여유가 얇다 (④)

```text
   internal/enode              81.5%   (하한 80 · .coverage-contract.yml)
   runc_overlay_linux.go       60.8%
   overlay_linux.go            48.4%
   실제 namespace 경로          integration 태그 시험 하나.  CI 의 go test 는 이 태그를 안 붙인다
```

기능 1 · 4 · 6 · 7 · 10 이 이 패키지에 문장을 더한다. **namespace 가 없어도 되는 규칙
(합치기 순회 · lower 상태 전이 · trash 삭제 경계 · capture 판정)은 기본 `go test` 에서
돌 수 있는 자리에 둔다.** 조각 7 의 「1 ~ 30번째 연산 뒤 강제 종료」 Go 테스트가 그
기준이다 — integration 태그 뒤에 두면 CI 밖이라 하한에 안 든다. 가짜 트리의 whiteout 과
opaque 표시를 특권 없이 만들 수 있는지는 Functional Design 이 실측으로 확인한다.

## 사실 4 — `runRoot` 를 지우는 자리는 다섯이다. 다섯 다 trash 로 간다 (⑤)

```text
   runc_overlay_linux.go:143 ~ :170   Open 의 실패 갈래
                         :437         세션 Close
                         :451         abort
                         :1017        helper cleanup
                         :1212        준비도 smoke 의 probe 디렉터리
```

팩은 뒤의 셋만 적는다. 결정 2-5 의 근거가 「**모든 경로가 한 기제를 쓴다**」이므로
다섯 다 `<scratch>/trash/` 로 `rename` 한다. smoke 는 `enode env check` 로 데몬 밖에서도
돌지만, 거기 남은 것은 다음 데몬 시작의 한 번 비우기가 치운다.

## 사실 5 — ADR-071 정본이 코드보다 늦다 (⑥)

ADR-071 의 머리가 「결정 (2026-09-17) · **미구현**」인데 코드는 `70d6258` 에서 구현했다.
`decisions.md` 7절(정본에 되돌려 올리는 것)은 ADR-073 의 E5 만 적는다. **ADR-071 의 상태
줄도 그 목록에 더한다.**

## 사실 6 — 기능 1 이 끄는 걷기를 정본 계약 문서가 저자에게 권한다

```text
   enode-design/protocol/run-contract.md:392   정말 처음 보는 빌드라면 — 한 번 돌리고
                                               workspace.changed 를 보면 된다
                                        :413   못 찾으면 안 걷고 이유만 workspace.changed 에 적는다
```

기능 1 은 build/test 단계의 전수 `workspace.changed` 를 없앤다. 그 대체인 「명시적으로
켜는 bounded discovery」는 기능 11 에 있고 팩의 착수 순서(`decisions.md` 5절)에서
**마지막**이다. 그 순서대로면 기능 1 과 기능 11 사이에 계약 저자가 산출물 경로를 찾을
길이 없다.

교정 — **bounded discovery 를 기능 1 과 함께 세운다.** 5절은 팩이 채운 권장값이라
근거를 적고 벗어난다. `collect` 의 「못 찾으면 이유를 적는다」도 `workspace.changed` 가
아닌 자리(diagnostics)로 옮긴다. `run-contract.md` 의 두 문장은 7절의 계약 문법 되돌림과
함께 고친다.

기능 1 의 범위는 팩이 적은 그대로 **build/test 단계**다. agent 단계의 `RecordDiff` 와
`Discover`(`claim.go:876-882`)는 기능 11 의 Git changeset adapter 가 설 때까지 오늘
그대로 둔다 — 훅이 `workspace.changed` 를 읽는다.

## 사실 7 — 합치기 표가 종류가 바뀐 항목을 따로 적지 않는다

ADR-077 §4 의 표는 「파일 · symlink 는 같은 경로로 `rename` 해 대체한다」이다. overlay 에서
lower 의 디렉터리를 지우고 같은 이름으로 파일을 만들면 upper 에는 whiteout 이 아니라
**그냥 파일**이 남는다. 그 파일을 lower 의 디렉터리 자리로 `rename(2)` 하면 `EISDIR` 로
실패한다(대상이 디렉터리이고 원본이 디렉터리가 아닐 때).

교정 — **종류가 바뀐 항목은 lower 쪽을 trash 로 옮긴 뒤 대체한다.** 조각 7 의 가짜
트리가 이 경우를 담는다. `merge.py` 가 이것을 어떻게 다뤘는지는 SunnyVM 에서 확인하고,
정본 표에 한 줄을 더하는 것을 7절 목록에 올린다.

## 사실 8 — 임대는 Run 의 effect 를 모른다

```text
   internal/enode/advertise.go:53   type Lease — run_id · node · capability · not_after · nonce
                                    단계도 effect 도 없다
```

결정 3-9 는 「형제는 **임대를 쥔 동안** 공유 잠금을 쥔다. prepare 를 담은 Run 은 잡지
않는다」이다. 임대가 노드에 닿는 순간 노드는 그 Run 에 prepare 단계가 있는지 모른다 —
그것은 첫 claim 이 단계를 받을 때 안다. 그 사이에 합치기가 배타 잠금을 잡으면 형제의
Run 이 옛 `ir` 로 매칭되고 새 lower 위에서 돈다.

잠금을 언제 잡고 언제 놓는지(임대가 닿는 순간 잡고 prepare 단계를 claim 하면 놓는 길,
또는 임대가 effect 를 싣는 길)는 **Functional Design 이 닫는다.** 요구는 「매칭된 뒤
첫 단계 전에 lower 가 바뀌지 않는다」이다.

## 사실 9 — scratch 를 다른 filesystem 에 둔 runc-overlay 노드는 not ready 가 된다

scratch 는 노드 설정의 `environment.scratch` 로 소유자가 적고 기본값이 없다
(`internal/enode/config.go:117`). 기능 8 의 `env check` 가 scratch 와 워크스페이스의
`st_dev` 일치를 확인하면, 둘을 다른 filesystem 에 둔 기존 노드는 이 팩 뒤에 not
ready 다. 결정 3-6 의 뜻(「같은 filesystem 을 강제한다」 · 「다른 filesystem 마운트포인트가
없는 환경이다」) 그대로라 묻지 않는다. `env check` 의 not ready 사유가 그 이유를 이름으로
말해야 한다.

---

# 이 문서가 닫히면

질문 여섯의 답을 받아 `requirements.md` 를 낸다. 질문 1 의 답이 기능 4 · 결정 2-6 ·
조각 4 의 문구를 정하고, 질문 4 ~ 6 의 답이 `aidlc-state.md` 의 Extension Configuration
표가 된다. 질문 2 와 3 의 답은 Workflow Planning 과 Inception PR 의 모양을 정한다.

---

# 답 — 2026-09-23T14:35:56Z

```text
   1   A    덧붙인 말 — arch 를 디스크에 묶은 단순 조건문은 제거한다
   2   B
   3   —    되물음 「병합을 안 하면 회차 진행에 문제가 되는지?」.  재질문으로 간다
   4   B    권장(A)을 벗어났다
   5   B
   6   X    「기존 테스트 컨벤션을 따른다」
```

## 답의 모호함 분석 (Step 6 의 의무)

**재질문은 질문 3 하나다.** 나머지 다섯은 값이 하나로 떨어진다.

```text
   1   A 에 덧붙인 말이 범위를 하나 더한다 — 아래
   2   B.  앞 두 회차와 같다
   3   선택이 아니라 되물음이다.  답하고 다시 묻는다 (requirement-clarification-questions.md)
   4   B.  확장의 baseline 규칙을 차단성으로 걸지 않는다.
       팩의 보안 표(features.md 3절)는 확장과 무관하게 이 팩의 요구로 남는다 —
       확장이 끄는 것은 SECURITY-NN 규칙이지 팩이 적은 요구가 아니다
   5   B.  권장 그대로
   6   X 를 이 저장소의 테스트 규약으로 읽는다 — 표준 testing · 단언과 모킹 도구 없음 ·
       .coverage-contract.yml 의 측정 명령과 패키지별 하한 80 · 허용목록 밖 스킵 0.
       PBT 프레임워크를 의존으로 더하지 않으므로 확장은 꺼진다(PBT-09 가 설 자리가 없다).
       팩이 요구한 합치기의 전수 시험과 사실 7 의 종류가 바뀐 항목은 그 규약 안의 시험이다
```

## 질문 1 의 덧붙인 말 — 정본 하나를 고친다

사용자의 짐작이 맞다. 그 조건문은 **ADR-017 결정 3**(확정 2026-08-19)이고, 같은 날
`1249c35`(S3 — 능력 탐지 · 광고 루프)가 코드로 옮겼다.

```text
   ADR-017 결정 3    여유공간은 매칭 조건이 아니라 광고 조건이다
                     디스크가 모자라면 그 capability 를 광고하지 않는다 -> 후보에서 사라진다
   ADR-068 §3.3      ADR-017③ 디스크가 모자라면 그 능력을 광고에서 뺀다     그대로
   ADR-076 §4.1      min_free_gb 는 이미 있는 광고 조건이라 새 기제가 없다
```

그때 「능력」이 빌드 하나였고 빌드 능력의 이름이 arch 였다. 그래서 디스크 판정이 arch
키에 붙었다.

답이 정하는 것은 둘이다.

```text
   코드   hasRoom 이 arch 키를 가리지 않는다.  arch 와 arch.<이름> 은 툴체인 탐지만 따른다.
          여유 하한은 노드 전체의 drain 계기 하나로 옮긴다 (A)
   정본   ADR-017 결정 3 의 기제가 「능력을 뺀다」에서 「노드가 drain 한다」로 바뀐다.
          매칭 조건이 아니라는 결정의 본체는 그대로다.
          ADR-068 §3.3 의 인용 줄과 ADR-076 §4.1 의 「새 기제는 없다」도 함께 고친다.
          decisions.md 7절(정본에 되돌려 올리는 것)에 더한다
```

**남는 것 없음.** 「디스크가 모자라면 광고에서 뺀다」는 결정의 뜻(동적인 여유를
매칭에 안 넣는다)은 A 에서도 선다 — 노드가 스스로 판단해 스스로 빠진다. 바뀌는 것은
빠지는 단위(키 하나에서 노드 전체로)다.
