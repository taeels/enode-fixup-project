# demo-back 자료 모델

**상태**: 검토안. 추가 테이블 없이 기존 Contract·Run·submitter를 사용한다.
wire 값은 [제출 계약](submission-contract.md)이 정한다.

| 자료 | 값 | 소유와 수명 |
|---|---|---|
| DemoSubmission | scenarioId, submitter, requestId | 검증한 한 요청. 임의 계약 필드 없음 |
| ScenarioDefinition | 공개 별칭, 고정 example 이름, 이름 주입 함수, 검증 결과 | 서버가 소유하는 고정 allow-list. 요청으로 변경 불가 |
| PreparedContract | RunID, Work, Requires, Steps, SuccessWhen 등 기존 Contract | 매번 example을 새로 읽어 준비. 공유 원본을 mutate하지 않음 |
| SubmissionIdentity | demo- + 정규 요청의 SHA-256 | 재시작·재접수에도 동일. 인증 자격 아님 |
| SubmissionContext | submitterKey의 Guest 이름 | 서버 내부 요청 컨텍스트. 기존 submit이 Run.Submitter로 전달 |
| AdmissionResult | 기존 HTTP status, runView 또는 error | 새 상태 enum을 추가하지 않음 |

Work는 이 데모 요청을 하나의 manual 작업으로 식별하도록 RunID와 같은 식별을
work.change_id와 work.id.change_id에 넣는다. 서로 다른 관람객의 작업이 example의
고정 work 키를 공유하여 산출물을 섞지 않게 한다. ledger는 고정 픽스처의 정책을
검토하고 이 데모에서는 Run 사이 산출 공유가 필요하지 않음을 확인한다.

submitter 컬럼·CreateRun·CreateRejectedRun 저장은 obs가 이미 제공했다.
CreateQueuedRun도 PR #5/main@310c22d 인수에서 submitter 저장·재접수·승격 보존을
확인했다. [인수 증거](../../ui/code/queue-integration-review.md). UI가 쓰는 이름 조회는 Run 목록이다.
GetRun에 새 표시 필드를 요구하거나 ui에서 store를 import하지 않는다.
