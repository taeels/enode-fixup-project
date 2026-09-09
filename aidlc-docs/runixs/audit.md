# AI-DLC 감사 기록 — runixs

## Codex v1 설정과 기존 Construction 맥락 복원

**Timestamp**: 2026-09-08T11:07:02Z

**User Input**:

```text
codex용 v1 aidlc 셋업해줘. 그리고 현재 이 repo에 진행중이던 aidlc를 이어갈껀데, 내 역할은 runixs 이거야. aidlc-docs/ 에보면 있을거야.
```

**AI Response**: 기존 v1.0.1 규칙을 읽는 Codex 진입점 `AGENTS.md` 와 체크아웃
전용 `local/aidlc-context.md` 를 설정했다. 담당은 runixs, 문서 루트는
`aidlc-docs/runixs/`, 유닛은 ui · demo-back 으로 복원했다. detached HEAD
`c010841` 에서 `unit/runixs-ui` 브랜치를 만들었다. README 에 재개 방법을 적고
담당 상태에 확인 결과와 기존 확장 선택을 기록했다.

**Context**: 설정과 맥락 복원이다. 개발 단계 완료나 승인으로 기록하지 않는다.
체크아웃에서 obs 목록 API 와 queue 구현이 없음을 확인했다. ui 실 모드 FD 는
obs 병합 뒤, demo-back 은 queue 병합 뒤이며 선행 게이트 통과는 미확인이다.
cardnews 회차가 낸 정적 UI 와 기존 `/ui/` 마운트는 후속 FD 의 입력으로 남긴다.

---

## 설정 검증

**Timestamp**: 2026-09-08T11:08:51Z

**AI Response**: `enode-design` 을 저장소에 고정된
`29c89cd39f14c3edb4042fcee57bcc04d91bfb06` 리비전으로 초기화했다. 설정과 재개에
필요한 파일 21개의 존재, VERSION 1.0.1, runixs 로컬 설정, AGENTS.md 크기와
override 부재를 확인했다. 변경 문서의 펜스·공백·표기 검사와
`enode-design/scripts/emphasis-check.py`, `git diff --check` 가 통과했다.
`local/aidlc-context.md` 가 기존 `/local/` ignore 규칙에 들어가는 것도 확인했다.

**Context**: 설정·문서만 검증했다. 제품 코드는 변경하지 않았고 CP0·CP1·CP2 등
장면 게이트는 실행하지 않았다. 커밋·푸시·PR 은 실행하지 않았다.

---

## obs 의미 설명 — 대화 기록 보완

**Timestamp**: 2026-09-08T14:16:04Z (기록 시각)

**User Input**:

```text
obs가 뭐야?
```

**AI Response**: obs 는 taeels 가 담당하는 관측 API 유닛이며, 노드·Run 목록과
Run 상세 확장으로 runixs 의 UI 가 표시할 실제 데이터를 제공한다고 설명했다.

---

## obs 전달 전 준비 범위와 결합 안정성 검토

**Timestamp**: 2026-09-08T14:16:04Z

**User Input — 중단된 요청**:

```text
내가 obs 전달 전에 미리 할 수 있는 일은 없을까??
```

**User Input — 이어진 요청**:

```text
내가 obs 전달 전에 미리 할 수 있는 일은 없을까?? 합쳤을 떄의 결합안정성을 해치는 일일까?
```

**AI Response**: API 정본·유닛 의존표·기존 정적 UI 를 검토했다. 화면 상태 설계,
소비 API 계약과 테스트 데이터 정리, 통합 검증 항목 준비는 선행 구현 없이도
가능하다. UI 를 미리 구현하려면 사전 작업 범위를 계획에 명시하고, 담당 파일
안에서 정본 계약을 소비하도록 구성하며 실제 obs 응답으로 검증해야 한다.
특히 draining 의 문자열 의미, lease 의 관측 의미, submitter 추가 요구와
sandbox 출처 미정을 임의로 추측하면 결합 시 동작이 어긋날 수 있다.

**Context**: 기술 검토와 제안이다. 사용자가 선행 구현이나 게이트 변경을 승인한
것으로 기록하지 않는다. 기존 obs 이후 W1 착수 규칙과 실제 CP1·CP2 검증 조건은
유지한다. UI·obs·공유 접점 코드는 수정하지 않았다.

---

## 사전 준비 승인과 실행

**Timestamp**: 2026-09-08T14:24:58Z (계획 기록 시각)

**User Input**:

```text
aidlc 스킬이 codex에 안잡히는데 괜찮은건가??? 일단 너가 말한대로 진행해줘. 혹시 세션을 다시 열어야 되면 그렇게할게
```

**AI Response**: 별도 aidlc 스킬을 설치한 방식이 아니라 AGENTS.md 로 저장소의
v1.0.1 규칙을 읽는 방식임을 설명했다. 현재 세션에서 규칙을 직접 읽어 적용하므로
사전 준비를 위해 재시작할 필요가 없다고 안내했다. 사용자 승인 범위는 앞서
제안한 UI 설계 초안·API 계약표·샘플 응답·통합 검증 항목으로 기록했다.

**Context**: 담당별 계획에 사전 준비 범위를 추가했다. 정식 FD 완료나 코드 생성,
공용 착수·병합 게이트 변경으로 해석하지 않았다. 확장 선택은 상속했고
resiliency-baseline·property-based-testing 은 기존 비활성 결정을 유지했다.

---

## 사전 준비 산출물과 검증 결과

**Timestamp**: 2026-09-08T14:30:56Z

**AI Response**: `construction/ui/preparation/` 에 읽기 안내, UI 설계 초안,
API 계약표, 샘플 사용 안내, 통합 검증 항목을 작성했다. 테스트 입력은
`internal/api/ui/testdata/obs-contract/` 에 합성 응답 17개와 manifest 1개로 작성했다.
중첩 attrs 예시와 기존 평탄 직렬화의 불일치를 발견해 D1 으로 기록했고,
lease 부재·verdict·submitter·chosen·sandbox·UI 모드 경계를 D2~D7 로 남겼다.

**검증**: JSON 구문과 manifest 대응, 노드·임대·Run 연결, created_at 정렬,
QUEUED 의 요구 능력을 가진 노드가 점유 중인지, requires 후보 의미 일치,
분기 의존·chosen 값, ASKED 와 만료 시각, 오류 HTTP 코드가 일관적인지 검사했다.
Markdown 링크·공백·펜스와 emphasis-check, git diff --check 가 통과했다.

**미실행**: `go test -count=1 -cover ./internal/api/ui` 는 go 명령을 찾을 수 없어
실행되지 않았다. 표준 설치·캐시 위치에도 Go 바이너리가 없었다. Go 커버리지와
실제 CP 게이트는 통과로 기록하지 않는다. 제품의 추적된 소스 코드는 변경하지
않았으며 테스트 입력은 기존 embed static 경로 밖이다.

**다음**: obs 반영 뒤 실제 응답을 D1~D5 와 대조하고 정식 ui FD 를 완성한다.
D6·D7 은 해당 FD 에서 닫는다. 준비 상태와 남은 검증은 담당 state 에 반영했다.

---

## 다음 진행 — 상세 설계와 로컬 도구 준비

**Timestamp**: 2026-09-08T14:44:17Z

**User Input**:

```text
다음 진행해줘
```

**AI Response**: 현재 진행 상태와 선행 코드를 다시 확인했다. origin을 fetch하자
main에 배정 문서 5개를 바꾼 `81b526c`가 있어 `git merge --ff-only origin/main`으로
반영했다. 담당의 기존 미커밋 변경은 보존했다. 최신 roster의 ui 패키지는
runixs(ui)·nacl1119(card-news) 공유다. obs와 queue 구현은 여전히 없어 실제
연동·게이트를 완료로 기록하지 않았다.

**산출물**: `construction/ui/functional-design/`에 업무 흐름·표시 규칙·자료 모델·
화면 구성 네 문서를 작성했다. `construction/plans/ui-functional-design-plan.md`와
`ui-code-generation-plan.md`에 검토 대상·파일별 8단계 순서를 남겼다. 사전 독립
표시 구현(1~3)과 obs 뒤 어댑터·랜딩·통합 검증(4~8)을 구분한 제안이다.
전체 ui 유닛·W3 데모 설계 완료나 코드 생성 승인으로 기록하지 않는다.

**결정안과 근거**: design README·pen 변수/프레임·S1/S2 export를 읽었고
frontend-design 스킬을 기존 시안 검토에 적용했다. sandbox는 명시적
capabilities[].attrs.sandbox의 능력별 광고 값을 읽고 없으면 미제공으로 표시한다.
기존 Cancel의 verdict.checks[].note, VERIFYING/CLAIMED, 노드가 아닌 Run에
연결되는 ASKED 의미를 설계에 반영했다. `/ui/`의 기존 관리자 폼을 활성화해
인증 성공 후 `/ui/fleet/`으로 이동하는 결정안으로 CP1 진입 조건을 보존했다.
공유 랜딩 접점은 통합 단계로 남기고 기존 Go 마운트를 재사용한다.

**도구와 검증**: 공식 go.dev 다운로드 메타데이터로 go1.26.6 darwin/arm64
아카이브의 크기(64772321 bytes)와 SHA256
`2dc95ce4675829f2df0e86b28bcef3283635902062a5f0580ca659bf570f3204`를 대조한 뒤
`local/toolchains/go1.26.6/go/`에 설치했다. 전역 PATH나 go.mod는 바꾸지 않았다.
`GOTOOLCHAIN=local ./local/toolchains/go1.26.6/go/bin/go test -count=1 -cover ./internal/api/ui`
실행이 성공했고 coverage는 92.3% statements다. 이전 Go 부재를 해결한 기존
UI 기준선이며, 새 JS 동작·브라우저·실제 obs·CP 게이트를 검증한 결과는 아니다.

**Context**: 제품 런타임 소스·공용 상태·design/*.pen은 수정하지 않았다. 지금
제시한 설계/코드 계획에 대한 사용자의 명시적 승인 응답은 아직 없다.

---

## W1 설계·코드 생성 계획 검토 요청

**Timestamp**: 2026-09-08T14:44:17Z

**Approval Prompt**: 이 설계와 전체 구현 계획을 승인하고, obs 전달 전 독립 화면
구현부터 진행할까요?

**대상**: `construction/plans/ui-functional-design-plan.md` Q1 및
`construction/plans/ui-code-generation-plan.md` 전체 8단계. 승인 후 1~3 실행을
제안하며 4~8은 obs 실제 응답 대조 조건을 유지한다.

**요청 근거**: 저장소 v1 규칙의 `construction/functional-design.md` Step 8과
`construction/code-generation.md` Part 1 Step 7은 설계와 코드 생성 계획의 명시적
승인을 요구한다. 후자의 원문은 “Do not proceed until the user explicitly approves
the unit code generation plan”이다. 현재 사용자의 “다음 진행해줘”는 새 계획을
보기 전의 요청이므로 승인 완료로 기록하지 않는다.

**Approval Status**: 대기. 사용자 응답 없음.

---

## 상세 설계 산출물 검증

**Timestamp**: 2026-09-08T14:45:57Z (검증 완료 후 기록 시각)

**검증 결과**: 담당 문서와 Codex 진입점/README 총 16개 Markdown의 상대 링크·
펜스·공백·emphasis-check, 기존 합성 JSON 18개의 구문, git diff --check가 통과했다.
FD 네 문서가 존재하며 추적된 internal/·cmd/ 런타임 소스와 design/ 파일의 변경이
없음을 확인했다. 이 결과는 문서·입력 검증이며 미구현 화면의 기능 검증이 아니다.

---

## 팀원 데모 pen과 현재 계획의 타깃 확인

**Timestamp**: 2026-09-08T14:51:03Z

**User Input**:

```text
너 그 demo-pen 우리 팀원이 그려놓은 것도 디자인에 반영했어??? 이번에 계획한건 그 타겟이 아닌가?
```

**AI Response**: 직전 8단계는 enode-ux.pen 기반 W1 실 함대 부분 계획이고,
팀원의 enode-demo.pen D1~D5를 구현 대상으로 직접 반영하지 않았다고 밝혔다.
runixs의 최종 범위에는 그 데모 화면이 포함되며, 백엔드 의존으로 디자인 준비까지
미룰 필요는 없다고 정리했다. enode-demo.pen의 프레임·문구·변수와 D2/D5 export를
읽고 requirements/decisions §8, features §3.4, 최신 roster와 대조했다.

**산출물**: `construction/ui/functional-design/design-targets.md`에 D1 게스트,
D2 함대 3D, D3의 네 투어 화면, D4 모달, D5 작업 3D, 웹캠·submitter를 현재 계획과
연결했다. 기존 FD·두 계획·담당 state에 부분 계획이라는 설명과 대조 링크를 추가했다.
데모 레이아웃/조작은 사전 준비 가능하고 실제 상태·제출·방송은 서비스 연결 뒤
검증한다는 차이를 기록했다. 원본 pen과 제품 코드는 변경하지 않았다.

**Approval Status**: 이 질문은 계획 승인·W1 취소로 해석하지 않는다. 데모 FD와
데모 코드 생성 계획을 완료했다고 기록하지 않으며 기존 승인 상태는 대기다.

---

## 3d-view 워크트리 재사용 검토

**Timestamp**: 2026-09-08T15:00:06Z

**User Input**:

```text
그리고 이거 진행하기 전에 /Users/runixs/orca/workspaces/enode-fixup-project/3d-view 워크트리에서 3d view를 좀 더 디테일하게 만든 코드가 있는데 그것도 확인해서 반영할지 확인해줘
```

**AI Response**: orca-cli 스킬과 설치된 CLI의 전체 가이드를 읽고 지정 워크트리를
확인했다. 실제 브랜치는 unit/ux-runtime, HEAD는 8985694다. 현재 작업과 공통
조상은 06215ff이며 소스는 3D 외에 API·큐·호스트 제어·요구 변경을 포함한다.
app.js·model.js·CSS·테스트·기존 실행 기록을 읽고 SVG 등각 표현과 그래프 탐색
기능을 선별 재사용하는 방안을 권고했다. 최종 화면 기준은 팀원의 demo pen이다.

**직접 검증**: Node v24.1.0 모델 테스트 3개 통과. 로컬 Go 1.26.6 UI 패키지
테스트 통과, 100.0% statements(정적 Handler). 실행 중인 로컬 서비스의 app.js가
검토한 파일과 같음을 확인했다. Orca 별도 탭에서 노드 6개·20단계/18 DONE/2 SKIPPED,
Fit·100%·125% 확대, 갱신 후 스크롤 유지와 2D/3D별 위치 복원을 확인했다.
Run 그래프에서 목록이 없어지는 차이도 직접 확인했다. 계약 형태·만료 우선순위·
누락 의존의 모델 탐침 결과와 인증/폴링 코드 검토 결과는 보고서에 구분해 남겼다.

**산출물**: `construction/ui/functional-design/3d-view-reuse-review.md`에 출처·
선별 범위·수정 조건·검증 한계를 기록했다. 디자인 타깃·화면 설계·FD 계획·담당
state와 코드 계획 1~3을 보완했다. 신규 scene.mjs에서 SVG를 분리하고 기존
모델·뷰를 재사용하는 제안이며, 소스 UI 전체나 backend를 가져오는 계획은 아니다.

**보존/상태**: 원본 워크트리 소스·.omc·터미널·실행 서비스는 변경하지 않았다.
검토 탭은 닫았다. Run 제출/취소/답변·노드 제어는 실행하지 않았다. 현재 체크아웃도
문서만 보완했고 코드 복사·cherry-pick·merge는 하지 않았다. 이전 계획 승인 대기를
이번 검토 요청으로 승인 완료 처리하지 않는다.

---

## AI-DLC 현재 단계 확인

**Timestamp**: 2026-09-08T15:15:28Z

**User Input**:

```text
aidlc 과정대로 하고 있는거지?? 지금 어느 단계인지 말해봐
```

**AI Response**: 담당 state·설계 계획·코드 계획·승인 로그와 승인된 실행 계획을
대조했다. 현재 공식 단계는 CONSTRUCTION의 ui Functional Design 보완·검토다.
W1 설계 검토안과 demo pen 범위 대조·3d-view 재사용 검토가 있으며, 데모 전체
FD와 설계 승인은 아직 없다. 미리 작성한 코드 계획은 초안이며 공식 Code
Generation 진입이나 제품 구현 완료가 아니다. 기존 코드 테스트·브라우저 검증도
재사용을 위한 사전 검증으로, 신규 기능의 Build and Test/CP 통과와 구분했다.

**기록 보완**: state의 현재 단계와 코드 계획의 상태를 위 구분으로 명확히 했다.
다음 순서는 데모 pen·3D 재사용 범위를 포함한 FD 보완/리뷰, 설계 승인, 코드 계획
승인 후 코드 생성, 실제 연동을 포함한 Build and Test다. 기존 실행 계획의
NFR Requirements·NFR Design·Infrastructure Design SKIP 선택을 유지한다.
사용자의 상태 질문을 단계 승인으로 기록하지 않았다.

---

## runixs 담당 작업과 팀 리드 역할 확인

**Timestamp**: 2026-09-08T15:19:27Z

**User Input**:

```text
너가 말한 순서는 우리 과제를 리드하는 최태양님이 해야하는게 아니고 내가 해야하는 일이 맞아???
```

**AI Response**: construction-roster §2·§5·§6, CLAUDE의 담당별 Construction 규칙,
유닛 의존/파일 행렬과 실행 계획의 열린 미정을 확인했다. ui·demo-back의 상세 설계·
코드·검증은 runixs 담당이며, 문서·코드는 에이전트가 작성하는 흐름이라고 설명했다.
사용자가 설명한 리드 최태양님의 공용 결정·접점 조율과, taeels에 배정된 obs·mcp
구현을 구분했다. 담당 유닛의 계획 검토가 팀 전체 설계나 공용 계약 승인은 아니다.

**보완**: sandbox 표시 출처는 roster §6에 따라 진행자와 닫아야 하는 결정안임을
state와 FD 검토 질문에 명확히 했다. 기존 제안을 확정된 공용 결정으로 올리지
않았으며 요구 팩·공용 상태·타 담당 문서는 수정하지 않았다. 이 질문도 단계 승인으로
기록하지 않는다.

---

## 담당 범위 구체화

**Timestamp**: 2026-09-08T15:42:32Z

**User Input**:

```text
내 범위를 구체화하자
```

**AI Response**: 배정표·유닛 정의·요구 팩·팀원 demo pen·3d-view 검토를 바탕으로
`construction/runixs-scope.md`를 작성했다. 실 함대 UI, 공개 데모 UI, demo-back을
작업 8개·파일 접점·완료 조건으로 구체화했다. obs·queue·진행자와의 책임을 나누고,
기존 W1 8단계가 전체 데모 계획은 아님을 FD 계획·state에 연결했다.

**Context**: 문서 연결 작업 중 다음 obs 병합 확인 요청을 받아 같은 작업을 이어갔다.
기존 팀 배정·공용 상태·pen 원본을 바꾸거나 범위 구체화를 단계 승인으로 기록하지 않았다.

---

## obs 병합 확인 · rebase · 계약 인수

**Timestamp**: 2026-09-08T15:42:32Z

**User Input**:

```text
obs작업이 완료 되었대. origin/main 확인해서 obs 브랜치 내용이 머지되었으면 그쪽으로 rebase해서 작업 이어가자
```

**AI Response**: fetch 후 origin/main의 PR #4 병합 `0a159a4`와 obs 브랜치
`ccc8073` 포함을 확인했다. 담당 미커밋 작업만 stash에 보관하고 `git rebase
origin/main` 후 복원했다. 충돌 없이 HEAD가 81b526c에서 0a159a4로 이동했다.
385개 기존 변경/미추적 파일 중 384개는 해시 동일, README는 로컬 추가와 upstream
수정 공존을 확인한 뒤 임시 stash를 제거했다. 설계 gitlink 29c89cd는 유지했다.

**계약 반영**: 중첩 requires.attrs, lease:null, 목록 Verdict/submitter,
필수 chosen, requires 읽기 실패의 200+warnings, Config.Demo의 공개 읽기 셋,
429/Retry-After/no-store를 담당 FD·계약표·계획에 반영했다. submitter 저장·컨텍스트
접점은 obs 구현을 재사용하도록 범위를 좁혔다. 샘플은 잘못된 후보를 오류 입력으로
바꾸고 429·부분 응답을 추가했다. 전부 합성이며 실제 함대 캡처로 표시하지 않았다.

**검증**: 기존 관측 테스트 API 17개·store 21개가 전용 임시 PostgreSQL 16에서
실패·스킵 없이 통과했다. scripts/testdb.sh의 지정 DSN 경로를 사용했고 DB는 종료했다.
UI/config/runctl 테스트 통과, 단독 커버리지 92.3%/81.8%/50.5%다. runctl의 80% 하한
통과나 전체 CP0 통과로 기록하지 않는다. 전체 빌드는 최초 로컬 Go 배포본의 자체
테스트 탐색 때문에 실패했으며, ignored local/toolchains/go.mod로 분리 후 통과했다.
루트 go.mod·제품 코드는 수정하지 않았다.

**현재 단계**: CONSTRUCTION → ui Functional Design 보완·검토를 유지한다.
obs 전달 대기는 해제됐다. 데모 전체 FD·제출 계약을 이어가며 queue·실제 장면·
sandbox 진행자 확인은 남아 있다. 담당 범위 요청을 마무리하면서 인수 결과를
`construction/ui/preparation/obs-integration-review.md`와 state에 연결했다.

---

## UI·demo-back 상세 설계 재개와 전체 계획 검토

**Timestamp**: 2026-09-08T15:58:58Z

**User Input**:

```text
AI-DLC를 이어가 보자.
```

**AI Response**: 현재 브랜치·담당 state·규칙을 확인하고 Functional Design을
이어갔다. 팀원 demo-pen 화면 사양과 요구 팩, 기존 Guest·obs·submit·example
코드를 대조했다. UI 네 문서에 D1~D5, 네 단계 투어, 두 시나리오 요청 버튼,
웹캠 resize/zoom/pan/fit·미니맵·플레이어 조작과 모드별 공용 표시를 보완했다.

**demo-back**: 세 FD 문서와 공동 submission-contract를 작성했다.
POST /v1/demo/runs의 scenario_id·submitter·request_id, 엄격한 입력 검증,
같은 의도의 동일 Run 식별, 미확인 응답의 같은 요청 재시도를 설계했다.
기존 서버 토큰·submitterKey·submit·DB 저장을 재사용하고 공개 쓰기 버킷은
읽기와 분리한 2 req/s·burst 4로 제안했다. 새 DB 스키마·미디어 중계는 없다.

**계획**: 기존 W1 8단계를 전체 UI 11단계로 대체하고 demo-back 7단계를 별도로
작성했다. construction/design-review.md에 설계·파일·검증·공용 입력을 연결했다.
이번 작업은 상세 설계와 검토 가능한 코드 계획 작성이며 코드 생성 실행은 아니다.

**비동기 확인 질문**:

```text
진행자와 정해진 데모 연결 정보가 있으면 알려주세요: ① LED·음원 시나리오 파일 경로 ② 웹캠 방송 공급자·주소 ③ sandbox를 capabilities[].attrs.sandbox에서 표시하기로 합의했는지. 아직 정해지지 않은 항목은 ‘미정’이라고 해도 됩니다.
```

**질문 상태**: 현재 회신 없음. generic example 셋만 확인했고 실제 LED/음원
픽스처·방송 입력·sandbox 진행자 확인을 수령했다고 기록하지 않았다. 공통 UI
구현과 실제 연결의 선행 조건을 분리했다. 외부 담당에게 메시지를 보내지 않았다.

**검증**: 이름 길이/형식 경계·UUID v4·정규 요청의 결정적 ID와 필드 변경 시
분리를 검산했다. 담당 문서 25개의 링크·코드 블록·공백, 표기와 git diff --check를
확인했다. 런타임 변경이 없어 기존 앱/DB 테스트를 재실행하지 않았다.
기존 제품 코드·pen·공용 상태·다른 담당 기록은 수정하지 않았다.

**제시할 검토 질문**:

```text
이 설계와 구현 순서로, 준비된 UI 코드부터 생성할까요?
```

**검토 상태**: 미응답. 저장소 AI-DLC construction/code-generation.md의 계획
명시적 승인 규칙에 따른 최종 검토다. 이번 재개 요청을 새로 작성한 결과의
소급 승인으로 처리하지 않는다. 통합 리뷰 §4가 현재 단일 질문이며 이전 W1 Q1을
중복으로 묻지 않는다. 승인 뒤 실제 입력이 준비된 단계부터 실행한다.

---

## 담당 설계·코드 생성 계획 승인

**Timestamp**: 2026-09-08T20:11:36Z

**User Input**:

```text
응 시작해
```

**AI Response**: 직전 통합 리뷰와 UI 11단계·demo-back 7단계 순서의 승인으로
기록하고 Code Generation을 시작한다. 먼저 공용 UI 모델·SVG·관측 클라이언트를
구현한다. queue·시나리오·방송·sandbox 공용 확인은 해당 연결 단계의 조건으로
유지하며, 같은 승인을 다시 요청하지 않는다.

---

## 승인된 UI 코드 생성과 독립 검증

**Timestamp**: 2026-09-08T20:41:21Z

**직전 사용자 입력**: 위의 “응 시작해” 승인을 계속 적용했다. 새로운 승인을
요청하거나 외부 입력 수령으로 해석하지 않았다.

**AI Response**: 승인된 ui 코드 계획 1~8을 구현했다. 공용 관측 모델·클라이언트,
2D/3D 함대/Run 그래프와 상세, 실 함대 세션 진입, D1 Guest·D2/D5 공개 화면,
투어·동일 의도 재시도 모달·웹캠 표시 조작을 작성했다. demo pen export와
3d-view@8985694의 선별 재사용 기준을 따랐다. 원본 워크트리·pen·API·store·
queue·config·DB 스키마는 수정하지 않았다.

**검증**: Node 42 통과, UI Go 98.4%. 전용 PostgreSQL 16.14 DB와 CI 스텁/정리한
PATH에서 전체 Go 표준 커버리지 실행 1,090 통과·0 스킵, 16 패키지 모두 80% 이상.
vet/build/표기/Windows enodectl·net/http 심볼 6개/govulncheck 통과.
초기 크로스 컴파일러 탐지 테스트 실패와 저장소 내부 Go 툴체인 스캔 문제는
검증 환경을 정리해 해결했다. 테스트가 만든 probe.lock PID 변경만 원복했다.

**브라우저**: 실제 api.Handler와 별도 DB에 등록한 광고 3개·예제 Run을 소비해
S0의 401/정상 진입·새 탭 세션 경계·logout·obs 갱신·그래프를 확인했다.
기존 Guest/cardnews 분기, 4단계 투어·선택/포커스 복원, 모달 차단/닫기/실제 404,
테스트 전송의 503 재시도·202 선택과 가짜 행 방지, 웹캠 크기·배율·이동·Fit,
390px iframe viewport를 검증했다. 시간 동결의 자원 간 간섭과 긴 그래프 Fit
하한 문제를 추가 수정·검증했다. 실제 하드웨어 실행·방송이라고 기록하지 않았다.

**남은 선행**: 현재 구현 기준의 queue 부재, 진행자 LED/음원 픽스처·이름 주입
필드·방송 주소·sandbox 합의 미수령. demo-back은 승인된 계획 1의 선행 인수에서
대기한다. UI 9·10 공동 장면, 전체 Code Generation 완료 리뷰·이후 commit/PR
게이트는 유지한다. 코드/검증 요약은 construction/ui/code/ 아래에 작성했다.
공용 상태·다른 담당 기록은 수정하지 않았고 팀원에게 메시지를 보내지 않았다.

---

## 독립 UI 구현 커밋 요청

**Timestamp**: 2026-09-08T23:28:22Z

**User Input**:

```text
runixs92@gmail.com으로 커밋해줘
```

**AI Response**: 현재 독립 UI 코드와 runixs 담당 문서를 `unit/runixs-ui`에
커밋한다. 기존 이름 Taewan Kim을 유지하고 작성자·커미터 이메일을
`runixs92@gmail.com`으로 명시한다. 작업과 무관한 untracked 파일은 보존한다.
직전 검증 결과와 보류된 외부 선행을 유지한다. 이 요청을 실제 데모 연결·
공동 게이트 완료나 원격 push·PR 병합 승인으로 확대하지 않는다.

---

## 구현된 UI 미리보기 요청

**Timestamp**: 2026-09-09T00:31:03Z

**User Input**:

```text
그러면 지금 디자인한 UI를 출력해줄 수 있어?
```

**AI Response**: 현재 커밋의 실제 UI embed와 API Handler를 전용 로컬 DB에
연결해 실 모드 18092와 공개 모드 18093으로 열었다. 미리보기용 노드 광고
3개를 로컬 API로 등록하고 3D 함대, 2D 함대, 새 작업 모달의 실제 브라우저
캡처를 ignored `local/ui-preview/screenshots/`에 저장했다. 방송은 미설정이며
노드와 Run은 로컬 검증 데이터다. 실제 장비 실행이나 demo-back 연결 완료로
기록하지 않는다. 사용자가 조작할 수 있도록 미리보기 서버를 유지한다.
추가 작업 그래프 캡처는 대기 문자열 불일치와 Orca runtime 연결 종료로
완료하지 못했다. 제품 코드나 기존 단계의 완료 상태는 변경하지 않았다.

---

## 데모 영상과 AI-DLC 연결 방식 상담

**Timestamp**: 2026-09-09T00:40:06Z

**User Input**:

```text
데모용으로 영상을 만들고 있었는데 이것도 aidlc workflow에 태워서 하면 좋나?
```

**AI Response**: 기존 영상 HANDOFF를 확인했다. 영상은 현재 소프트웨어
AI-DLC에 연결한 별도 제작 작업으로 관리하되, 기존 HANDOFF와 자산을 정본으로
유지하고 AI-DLC에는 범위·구현 의존성·검토 결과를 연결하는 방식을 권한다.
0–30초 채택본 보존과 30초 이후 웹 작업 후 재개 결정은 유지한다. 현재 공식
데모는 LED·음원이며 영상의 승인·이미지 전달·퓨징 이야기와 다르므로 장면별
현재 구현·미연결·설명용 표현을 대조하는 작업을 먼저 제안한다. 콘티와 현재
UI 자료 준비는 가능하지만 실제 완주 촬영은 해당 기능의 통합 검증 뒤다.
새 유닛 배정, 공용 계획 변경, 영상 생성·크레딧 사용은 실행하지 않았다.

---

## 웹 동작 영상 도구 비교 상담

**Timestamp**: 2026-09-09T00:42:53Z

**User Input**:

```text
Kling이 애니메이션이나 실제 영상 제작 말고 웹의 동작을 애니메이션화해서 영상을 만드는데에도 좋나? 다른 좋은게 뭐가 있을까?
```

**AI Response**: Kling, Remotion, Screen Studio, Jitter, Adobe 공식 자료를
조회했다. 웹 동작의 정확한 표현은 실제 화면 녹화와 시간·레이어를 직접
제어하는 제작 도구가 적합하다는 판단을 설명한다. enode에는 실제 웹 UI
녹화와 Remotion 합성을 권하고, 빠른 직접 녹화에는 Screen Studio, 디자인
레이어 애니메이션에는 Jitter, 움직이는 기기 화면 합성에는 After Effects를
비교한다. Kling은 기존 인물·배경 연출에 활용할 수 있다. 기존 한글 UI
합성 수정 이력도 판단 근거로 삼았다. 도구 설치·유료 생성은 실행하지 않았다.

**참고**:
- https://kling.ai/quickstart/klingai-video-3-model-user-guide
- https://www.remotion.dev/docs/the-fundamentals
- https://screen.studio/
- https://jitter.video/product/
- https://www.adobe.com/products/aftereffects/motion-tracking.html

---

## Remotion 에이전트와 MCP 지원 확인

**Timestamp**: 2026-09-09T00:46:35Z

**User Input**:

```text
Remotion도 MCP나 에이전트가 동작하나??
```

**AI Response**: 공식 문서에서 Agent Skills를 통한 영상 구성·미리보기·
렌더링 지원을 확인했다. 기존 문서 검색용 Remotion MCP는 deprecated이며
신규 설치 대신 remotion-docs 스킬을 권고한다. 별도로 Studio를 조회하고
재생·탐색하는 WebMCP가 있으며 호환 브라우저 에이전트가 필요하다. 이 세션의
노출 도구에는 Remotion 전용 도구가 없으므로 WebMCP 연결 완료로 주장하지
않는다. 현재 작업에는 공식 스킬과 CLI를 사용하는 구성을 권한다. 설치나
새 영상 생성은 실행하지 않았다.

**참고**:
- https://www.remotion.dev/docs/ai/skills
- https://www.remotion.dev/docs/ai/coding-agents
- https://www.remotion.dev/docs/ai/mcp
- https://www.remotion.dev/docs/ai/webmcp

---

## Remotion UI 샘플 제작 착수

**Timestamp (기록 시각)**: 2026-09-09T01:02:31+00:00

**User Input**:

```text
응 한번 해주ㅝ
```

**AI Response**: 직전 제안한 공식 스킬 설치와 별도 15초 UI 샘플 제작·미리보기·출력의 승인으로 해석했다. 공식 Remotion 스킬 7개를 설치하고 별도 프로젝트에 실제 UI 캡처, 한글 폰트, 시간 기반 확대·전환·자막·효과음을 구성했다. 대표 프레임 검수 후 MP4 출력 중이다. 기존 영상과 제품 유닛 단계는 유지한다.

---

## Kling 캐릭터 참조 시트 요청

**Timestamp (기록 시각)**: 2026-09-09T01:02:31+00:00

**User Input**:

```text
그리고 Kling에서 처음에 내가 보여준 캐릭터가 진행됨에 따라서 달라지더라고, 캐릭터 시트를 각 캐릭터별로 만든 후에 그걸 같이 input에 넣어서 줘야할 것 같아.
```

**AI Response**: 원본 포스터와 첫 장면, 기존 캐릭터 변화 검토 기록을 확인했다. 파란 요청자·금색 동료·흰 로봇 말의 개별 시트와 서대현님의 손·휴대폰 참조를 내장 imagegen으로 제작한다. 1차 시트의 과한 3D 표현과 구형 폰 차이를 확인해 원본 2D 스타일의 수정본을 생성 중이다. 장면 시작 프레임과 캐릭터 참조를 구분해 다음 Kling 입력 계획에 기록한다. Kling 업로드·Element 등록·영상 생성은 실행하지 않는다.

---

## 영상 보류 · queue 연동 재개

**Timestamp (기록 시각)**: 2026-09-09T01:07:43.279624+00:00

**User Input**:

```text
곧 queue가 머지된데, 지금 영상은 뒤로 미루고 일단 이거 해야할 것 같아.
```

**AI Response**: 영상 산출물을 보존하고 작업을 보류한다. fetch 결과 queue PR #5가 2026-09-09T01:00:41Z에 main@310c22d로 병합된 것을 확인했다. 기존 승인된 CONSTRUCTION 코드 생성 계획에 따라 담당 브랜치를 rebase하고 실제 queue 접점·시나리오 선행을 인수해 UI·demo-back 작업을 이어간다.

---

## UI 개선 요청 반영

**Timestamp (기록 시각)**: 2026-09-09T01:14:22.085984+00:00

**User Input (JSON, 원문 공백 보존)**:

```json
"- 그리고 웹캠뷰 닫기 버튼 필요해. \n- 사용자이름 한글 두단어로 바꿔주세요\n- 카드나 뭔가를 눌러서 나타난 플로팅 카드는 해당 카드 밖에 이전 레이어를 클릭하거나 액션이 생기면 자동으로 닫혀야해. \n\n\n위와같은 개선사항 요청이 왔어. aidlc에 반영해서 진행하자"
```

**AI Response**: 요청한 세 개선의 구현 지시로 기록한다. `construction/plans/ui-feedback-plan.md`에 변경 범위·수용 조건·검증을 적고 기존 승인된 코드 생성의 보완으로 실행한다. 한글 두 단어 이름을 신규 제출 계약에 반영하되 이전 미확인 요청의 원문을 보존해 재시도 시 Run 식별이 바뀌지 않게 한다. 웹캠 닫기/복원과 노드·단계 상세·모달·투어의 바깥 조작 닫기를 구현한다. queue 인수는 유지하고 실제 픽스처·방송 게이트 보류를 유지한다.

---

## queue 인수와 UI 개선 검증 결과

**Timestamp (기록 시각)**: 2026-09-09T01:20:31.137887+00:00

**AI Response**: queue@310c22d로 rebase 완료. 실제 201/202/200·거절·승격과 UI 선택 유지, 별도 DB에서 CreateQueuedRun의 submitter 저장·목록·승격 보존을 확인했다. rebase 기준 전체 Go 1,108 통과·0 스킵·16패키지 80% 이상. 이후 UI 개선은 Node 46·Go UI 98.4%·브라우저 24개 동작 단언 통과. 초기 브라우저 도구의 프로그램식 click은 포커스를 옮기지 않아 반환 포커스 기대가 틀렸고, 실제 클릭과 같이 먼저 포커스를 주어 재검증했다. 제품 포커스 동작은 유지했다. 모달 내부 조작과 폴링은 상세를 닫지 않으며 미확인 요청은 원문대로 재시도한다. 담당 FD/코드 계획/상태/검증 기록을 갱신했다. 영상 산출물은 보존·보류. 실제 픽스처·방송 입력과 공동 CP9/10/11은 보류이며 공용 문서·pen·다른 유닛 코드는 수정하지 않았다. 이번 보완은 아직 커밋·push·PR 하지 않았다.

---

## 담당 위치와 다음 작업 확인

**Timestamp (기록 시각)**: 2026-09-09T01:24:05.264422+00:00

**User Input**:

```text
그러면 내 작업은 어디까지 된건지 내 위치를 알려줘. 다음 구현을 기다려?
```

**AI Response**: 담당 상태와 git 상태를 대조하고 origin을 다시 fetch했다. main은 여전히 queue 병합 310c22d이며 실제 LED/음원 examples는 없다. runixs는 CONSTRUCTION 코드 생성 단계다. UI 독립 구현 1~8과 요청한 개선은 구현·로컬 검증 완료, queue 인수 완료다. 다음 구현인 demo-back도 runixs 담당이고 아직 구현하지 않았다. 실제 시나리오 계약 파일·매핑을 진행자에게 받아 연결하며 drain·panel·mcp 전체 완료를 기다리는 의존은 없다. 화면 구현 완료와 실제 데모·공동 게이트 완료를 구분하고, 최신 개선 변경은 아직 미커밋임을 안내한다.

---

## UI 개선과 queue 인수 기록 커밋·push 승인

**Timestamp (기록 시각)**: 2026-09-09T01:25:46.761181+00:00

**User Input**:

```text
그러면 일단 commit해서 push할까
```

**AI Response**: 완료한 UI 개선과 queue 인수 관련 담당 기록의 커밋·push 지시로 해석한다. 작성자·커미터는 Taewan Kim <runixs92@gmail.com>을 사용한다. 영상 파일·영상 별도 계획·이미지·로컬 검증 산출물은 보존하고 이번 커밋에서 제외한다. fetch 결과 원격 unit/runixs-ui는 7e4b4c7cf2e78de73d599a36069d05dc60562aa5이며, range-diff로 원격 두 커밋과 rebase한 f3a82b9/4c1265d의 내용이 같음을 확인했다. 다른 사람의 새 커밋이 없으므로 정확한 원격 SHA를 조건으로 force-with-lease push한다. Node 46·Go UI 98.4%·브라우저 24개 검증 결과를 유지하고 제출 코드 변경 없이 staged diff와 문서 범위를 확인한다. 이번 승인은 독립 변경의 저장·공유이며 Code Generation 전체 완료·공동 장면 통과·main 병합 승인은 아니다.

---

## sandbox 출처 승인 전달과 작업 재개

**Timestamp (기록 시각)**: 2026-09-09T01:29:41Z

**User Input**:

```text
샌드박스는 최태양님이 승인 했는데, 좀 더 진행할까?
```

**AI Response**: 최태양님의 기존 sandbox 출처 결정안 승인을 사용자가 전달한 것으로 기록한다. capabilities[].attrs.sandbox의 능력별 광고 원문을 표시하고 없으면 미제공으로 처리한다. 요구 값을 노드 광고로 복사하거나 격리 보장으로 해석하지 않는다. 담당 FD·상태·계획에서 출처 승인 대기를 해제한다. 실제 장비의 설정·광고·격리 동작 검증과 공용 roster 편집은 별개다. 기존 표시 경로를 실제 로컬 API·브라우저로 확인한다.

---

## queue 선행에 관한 팀 코멘트 수령

**Timestamp (기록 시각)**: 2026-09-09T01:29:41Z

**User Input**:

```text
4. queue 는 이미 풀렸는데 runixs 가 모른다
runixs 기록은 전부 "CreateQueuedRun·WakeQueued 는 아직 없다" 기준이다. PR #5 가 어제 main@310c22d 로 들어왔고 internal/store/queue.go:92,133 에 있다. 알려주면 UI 9단계와 demo-back 접수가 바로 풀린다. 여기서 태양님 몫은 하나 — 큐 생성 경로가 submitter 를 보존하는지 확인(obs 가 낸 컬럼과 api.submitterKey 접점이다).


라고 코멘트가 왔어
```

**AI Response**: fetch 결과 main은 310c22d이고 현재 브랜치는 이미 그 위에 있다. ce8bc14에 queue 인수·202/승격·목록 submitter 보존의 로컬 검증 기록을 커밋·push했으나 design-review와 obs 인수 문서에 이전 미구현 문구가 남아 있었다. 현재 상태와 역사 기록을 구분해 바로잡는다. 기존 store 직접 입력 검증을 보완해 submitterKey→기존 auth/postRuns→enqueue→CreateQueuedRun→목록→승격 경로를 담당 demo-back 인수 회귀 테스트로 검증한다. queue와 sandbox 승인은 더 이상 대기 조건이 아니다. 실제 LED/음원 파일은 main examples에 없으므로 해당 연결과 방송 장면은 완료로 표시하지 않는다.

---

## push된 UI 범위의 Draft PR 생성

**Timestamp (기록 시각)**: 2026-09-09T01:32:21.819927+00:00

**User Input**:

```text
지금 push된 부분까지 PR올려줘
```

**AI Response**: 별도 대화에서 요청한 PR 생성만 실행했다. 원격 unit/runixs-ui의 ce8bc147a27f3c1456d2525f6a18a77442dd272b까지 3개 커밋으로 main 대상 Draft PR #6(https://github.com/taeels/enode-fixup-project/pull/6)을 생성하고 head·base·Draft·OPEN 상태를 재조회해 확인했다. 기존 UI·queue 검증 기록과 실제 demo-back/하드웨어/방송 연결의 미완료 범위를 PR 본문에 적었다. 현재 로컬의 추가 테스트·계획 변경·영상 산출물은 커밋하거나 push하지 않았다. 제품 코드·브랜치·인덱스·담당 단계는 변경하지 않았으며 이 요청·결과만 담당 감사 로그에 덧붙였다. 장면 게이트 완료나 main 병합은 수행하지 않았다.

---

## queue 코멘트 정정과 sandbox 후속 검증 결과

**Timestamp (기록 시각)**: 2026-09-09T01:34:05.658973+00:00

**AI Response**: 담당 현재 상태와 설계 리뷰·FD/코드 계획·이전 obs 인수 기록의 오래된 대기 문구를 정리했다. queue·sandbox 승인 대기를 해제하고 실제 시나리오·방송·장비 장면은 남겼다. internal/api/demo_queue_test.go에서 내부 submitterKey→auth/postRuns→queue→공개 목록→승격의 이름 보존과 재접수 시 최초 이름 유지, 외부 헤더 무시를 검증했다. 전용 PostgreSQL 16.14 + scripts/testdb.sh에서 API 전체 140 통과·0 실패·0 스킵, API vet·glyphscan 89파일 통과. 실제 로컬 광고 API를 소비하는 브라우저에서 2D/3D sandbox 13개 단언 통과. 제품 표시 코드를 수정할 필요는 없었다. 담당 문서와 테스트를 기존 커밋·push 지시에 따라 같은 이메일로 unit/runixs-ui에 반영한다. main·공용 roster·pen과 미추적 영상 산출물은 수정하지 않았다.

---

## 영상 별도 세션 병렬 작업 가능 여부

**Timestamp (기록 시각)**: 2026-09-09T01:41:46.749631+00:00

**User Input**:

```text
다음 구현 진행하기 전에, 동영상 제작관련해서는 다른 세션에서 병렬로 진행해도 괜찮아??? conflic없어?
```

**AI Response**: 영상 파일은 output/enode-video, 별도 계획은 construction/video 아래이며 제품 구현과 파일 범위를 나눌 수 있음을 확인했다. 별도 영상 브랜치·워크트리를 권장한다. 영상 산출물은 미추적이므로 새 워크트리에 자동으로 나타나지 않아 복사 인수가 필요하다. 같은 체크아웃에서는 Git 인덱스·브랜치와 파일을 공유하므로 두 세션의 커밋/rebase가 충돌할 수 있다. 영상 전용 파일과 기록은 영상 세션, 제품 코드·담당 통합 state/audit는 현재 세션이 맡는 경계를 제안한다. UI 캡처는 특정 커밋 기준으로 고정하고 실제 제출 연동 장면은 demo-back 구현 뒤 갱신한다. 이 질문만으로 영상 생성이나 유료 요청을 재개하지 않았다.

---

## 담당 구현 우선·영상 순연과 demo-back 공통 구현 재개

**Timestamp (기록 시각)**: 2026-09-09T01:45:24.968335+00:00

**User Input**:

```text
그건 이미 이 worktree에 작업이 되어있어서 힘들고, 일단 구현 부터하고 내 할당일을 다 처리한 후에 작업하자.ㅏ
```

**AI Response**: 현재 워크트리에서 runixs 배정 구현을 먼저 진행하고 영상은 그 뒤로 미룬다는 지시다. 영상 파일은 이동하지 않고 보존한다. main@310c22d의 obs·queue를 인수했으므로 기존 승인된 demo-back 설계의 공통 백엔드를 구현한다. 앞선 계획이 실제 픽스처 인수를 모든 코드 생성의 앞에 둬 과도하게 대기하던 순서를 보완한다. 입력 검증·같은 의도 식별·전용 한도·서버 토큰/submitterKey·기존 submit 연결을 먼저 구현하고 실제 픽스처 인수는 활성화 및 공동 장면의 조건으로 유지한다. 승인된 FD의 미설정 503을 구현하므로 임의 하드웨어 계약이나 새 설정 경로를 만들지 않는다. 성공/queue 경로는 테스트 전용 기존 generic example으로 실제 DB에서 검증하고 실제 하드웨어 증거와 구분한다.

---

## 진행자가 추가한 실제 데모 픽스처 인수

**Timestamp (기록 시각)**: 2026-09-09T01:51:59.901225+00:00

**User Input**:

```text
내 PR위에 팀장님이 커밋을 추가하셨다는데 그거 가져와서 반영하자.
```

**AI Response**: PR #6의 origin/unit/runixs-ui에 taeels의 c7a237d가 추가된 것을 fetch로 확인했다. 변경은 진행자 README와 실제 example 두 파일이다. 현재 demo-back 작업 파일과 겹치지 않아 fast-forward했고 진행 중 6개 파일의 SHA-256이 그대로임을 확인했다. LED는 heartbeat→persistent 한 Run, 음원은 mac synthesize→board play와 봉인 blob이다. 이름은 synthesize.in.prompt의 정확한 자리표시자 한 곳에만 주입한다. Work는 승인된 제출 계약대로 요청별 manual 식별로 준비하고 원본 계약의 requires·needs·success_when·실행 argv와 기본 run ledger 범위를 보존한다. 실제 고정 매핑을 활성화하고 example·서버·UI 통합 검증을 다시 진행한다. 하드웨어 명령/송출 운영은 진행자 README의 담당 범위에 남긴다.

---

## 실제 demo-back 구현·연동 검증 결과

**Timestamp (기록 시각)**: 2026-09-09T01:59:44.928902+00:00

**AI Response**: c7a237d 위에서 demo.go와 회귀 테스트를 작성하고 api.go에 데모 전용 등록 3줄을 추가했다. 두 실제 example·정확한 prompt 이름 주입·본문/형식/같은 의도 식별·전용 한도·서버 토큰·기존 submit/queue 연결을 구현했다. 표준 Go 전체 1,173 통과·실패/스킵 0, 16패키지 80% 이상(api 84.539%, demo.go 96.0%). 동시 4개 재시도 race·vet·전체/교차 빌드·govulncheck 통과. 로컬 시험 도구가 탐색에 포함되던 예비 검사는 모듈 경계로 분리하고 16제품패키지 기준으로 재실행했다. 실제 Handler+fixture+로컬 광고+DB의 브라우저에서 LED 201·음원 202·승격·이름/claim·3D·목록을 검증했다. 최종 화면에서 승격 뒤 낡은 QUEUED 안내가 남는 것을 발견해 demo.js에서 관측 후 숨기도록 수정하고 Node 46·Go UI 98.4%·브라우저 14개 단언을 재검증했다. 테스트 유발 probe.lock PID는 3671709로 복구했다. 구현/검증 문서와 계획·상태를 갱신한다. 기존 커밋·push 지시에 따라 동일 이메일로 담당 브랜치에 반영하고 PR #6의 설명을 최종 범위로 갱신하되 Draft를 유지한다. 문태호의 보드/스피커 래퍼, 손신의 mac Claude+higgsfield, 최태양의 공개 webcam URL 및 실제 공동 장면은 남아 있다. 영상·공용 상태·진행자 fixture 원문은 보존했다.

---
