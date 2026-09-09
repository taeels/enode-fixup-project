# obs 인수 — origin/main 반영과 소비 계약

**기록 범위**: obs 인수 당시의 검증 기록이다. 현재는 queue PR #5/main@310c22d를
이미 인수했고 UI 구현도 진행했다. [최신 queue 인수·접점 증거](../code/queue-integration-review.md)와
[담당 현재 상태](../../../aidlc-state.md)를 우선한다. sandbox 출처 승인도 수령했다.

**당시 기준**: 2026-09-09 KST, `origin/main@0a159a4`.
PR #4가 `unit/obs@ccc8073`를 병합했다. `unit/runixs-ui`를 `81b526c`에서
이 커밋으로 rebase했고 충돌은 없었다. 제품 UI 구현 단계로 넘어간 기록은 아니다.

## 1. 작업 보존

담당 문서·샘플·Codex 설정 변경만 임시 stash에 보관했다가 복원했다.
사전 해시를 기록한 385개 파일 중 384개가 동일하며, README는 기존 추가분과
upstream 변경이 함께 남았음을 확인했다. 검증 뒤 임시 stash를 제거했다.
설계 서브모듈은 `29c89cd` 그대로다. 커밋·push·main 변경은 하지 않았다.

## 2. UI가 사용할 계약

| 접점 | 병합된 코드에서 확인한 값 | 담당 설계에 반영 |
|---|---|---|
| D1 requires | `store.RequireView`의 중첩 `attrs`, 빈 속성은 `{}`, count 생략 가능 | 관측은 중첩 형태만 소비. 제출 계약의 평탄 속성과 구분 |
| D2 lease | 키가 항상 있고 미임대는 `null` | 필드 생략 후보는 잘못된 응답 검증용으로 전환 |
| D3 verdict | 목록은 Verdict 객체 또는 `null`; `checks[].note` 보존 | 종료 사유를 텍스트로 표시. 실제 drain 장면은 후속 검증 |
| D4 submitter | 목록에 문자열 키가 항상 있음, 과거/미지정은 `""` | 목록에서 읽음. 상세에는 이 필드가 없으므로 목록 자료와 결합 |
| D5 chosen | 단계에 boolean 키 항상 포함, false도 생략하지 않음 | 누락을 false로 바꾸지 않음 |
| D5 부분 응답 | 정상 상세는 현재 유효 계약의 requires. 읽기 실패 시 200+warnings, requires 생략 | Run 상태는 보여주고 요구 정보 미확인을 표시. 빈 요구로 바꾸지 않음 |
| D7 공개 읽기 | `Config.Demo` / `ENODE_DEMO_MODE=1`에서 nodes·runs·detail 셋만 무인증 | 데모는 토큰 없이 이 셋을 소비. asks는 실 함대 모드에서만 조회 |
| 조회 제한 | 공개 읽기 셋이 전역 120 req/s, burst 240 공유. 429와 `Retry-After: 1` | 한도 초과 표시, Retry-After 이전 자동/수동 재요청 방지 |
| 캐시·질의 | 읽기 셋 성공에 `Cache-Control: no-store`. nodes는 질의 인자 전부 400 | 캐시 무효화용 `?t=` 금지. runs 기본 100, 상한 1000, 페이지네이션 없음 |

코드 근거는 `internal/api/{nodes,runs,ratelimit,api}.go`,
`internal/store/{observe,store}.go`, `internal/config/config.go`다.
obs의 설계·검증 기록은 `aidlc-docs/taeels/construction/obs/`를 대조했다.
공개 읽기의 정본 예외와 노출 정책은 obs가 진행자에게 남긴 기록을 상속한다.
이 인수로 진행자 결정까지 완료 처리하지 않는다.

## 3. demo-back에 이미 마련된 연결

obs가 `runs.submitter` 컬럼뿐 아니라 `store.Run.Submitter`, 기존 Run 생성·거절
저장, `api.submitterKey` 컨텍스트를 읽는 submit 경로도 제공했다. demo-back은
Guest 이름을 검증하고 이 컨텍스트에 넣어 기존 접수를 사용한다. 이미 구현된
저장을 다시 만들지 않는다. `Config.Demo`도 같은 필드를 사용한다.

obs 인수 당시에는 `CreateQueuedRun`·`WakeQueued`가 없었다. 이후 PR #5/main@310c22d의
202/QUEUED·재접수·승격·submitter 보존을 확인하여 queue 선행을 인수했다.
실제 고정 시나리오 연결은 별도로 남아 있다.
고정 시나리오 allow-list와 스텝 이름 주입은 여전히 runixs 구현 범위다.

## 4. 이 체크아웃에서 실행한 검증

- Go 1.26.6, 전용 임시 PostgreSQL 16 클러스터·DB로 기존 관측 테스트 실행.
  `scripts/testdb.sh`의 지정 DSN 경로 사용. API 17개·store 21개 통과, 실패·스킵 0.
  빈/임대 함대, requires/chosen, 부분 응답, submitter, 공개 읽기·인증·한도 포함.
- `go test -count=1 -cover ./internal/api/ui ./internal/config ./internal/runctl` 통과.
  각각 92.3%·81.8%·50.5%. 이 단독 실행의 runctl 커버리지는 80% 미만이므로
  전체 CP0 커버리지 통과로 기록하지 않는다.
- `go build ./...` 통과. 첫 실행은 로컬 설치한 Go 배포본의 자체 테스트를
  프로젝트로 탐색해 실패했다. ignored `local/toolchains/go.mod`로 도구 디렉터리를
  별도 모듈 경계로 분리한 뒤 재실행했다. 루트 go.mod와 제품 코드는 그대로다.
- 임시 PostgreSQL은 테스트 뒤 종료했다. 기존 실행 서비스·사용자 DB는 사용하지
  않았다. 로그·실행 명령은 ignored `local/obs-integration-20260909/`에 보존한다.

PostgreSQL 17 기준의 전체 CP0 재검증, 실제 데몬 함대, queue·UI 장면은 이번
검증에 포함되지 않는다. obs 담당 기록의 CP1 부분 측정과도 구분한다.

## 5. 이어갈 작업

obs 전달을 기다리는 상태는 해제한다. W1 실 함대 설계는 실제 계약으로 갱신하고,
[담당 범위](../../runixs-scope.md)에 따라 데모 전체 FD와 제출 계약을 보완한다.
당시 계획했던 UI 어댑터·브라우저와 queue 연동은 후속 구현·검증 기록을 따른다.
queue 인수와 sandbox 출처 승인은 완료했다. 실제 고정 시나리오·방송·공동 CP 장면은 남아 있다.
