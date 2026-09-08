# 샘플 응답 안내

경로는 `internal/api/ui/testdata/obs-contract/` 이다. 전부 **합성 데이터**이며
현재 서버에서 캡처한 결과가 아니다. `manifest.json` 에 기준 리비전·HTTP 상태·
계약 검증 결과·폴링 실패 시나리오를 기록했다. 2026-09-09 obs `0a159a4` 인수로
중첩 requires와 lease:null을 정상 형식으로 좁혔다.

기준 시계는 **2026-09-08T12:00:00Z**다. 실제 오늘 시계로 계산하면 모든 광고가
만료돼 테스트 의미가 사라진다. 후속 화면 테스트에서 시계를 주입한다.
복구 응답은 새 observed_at·seen_at·expires_at 을 만들어 사용한다.

## 1. 사례

| 파일 | 목적 | 기대되는 표시·판정 |
|---|---|---|
| nodes-empty.json | 정상 빈 함대 | 오류가 아닌 빈 상태 |
| nodes-states.json | 여섯 노드 | idle·실행·draining 실행·draining 대기·만료 임박·사람 대기 |
| nodes-lease-omitted.json | 잘못된 응답 | 필수 lease 키 생략. 관측 계약 불일치로 거부 |
| runs-empty.json | 정상 빈 Run 목록 | 작업 없음 |
| runs-states.json | QUEUED·RUNNING·SUCCEEDED·FAILED | created_at 내림차순, QUEUED 배정 없음. verdict/submitter 는 D3/D4 후보 |
| run-queued-nested.json | ADR-069 의 attrs 중첩 예시 | requires[].as 와 capability·attrs 표시. D1/D5 |
| run-queued-flat.json | 잘못된 관측 응답 | 평탄 속성은 계약 입력 형식. UI 관측 응답으로 거부 |
| run-branch.json | 선택된 갈래와 미선택 갈래 | apply 는 CLAIMED+chosen=true, alternate 는 SKIPPED+false |
| run-drain.json | draining 중 실행 | node-03 의 work 단계는 진행 중 |
| run-asked.json | 사람 대기 단계 | node-06 은 ASKED. run-asked 자체는 RUNNING |
| run-chosen-unreachable.json | 방어적 표시 사례 | SKIPPED+true 를 보통 미선택 분기와 구별. 현재 제출 경로가 이 상태를 만든다는 주장이 아님 |
| asks-empty.json | 정상 빈 인박스 | 열린 질문 없음 |
| asks-pending.json | 기한 없는 질문 | reviewer@example.invalid, asked_at 이후 경과 시간. 남은 시간은 없음 |
| error-401.json | 기존 auth 오류 | 토큰 재입력, 보호된 화면 숨김 |
| error-404.json | 기존 getRun 미존재 | 선택한 Run 을 찾을 수 없음 |
| error-503.json | 기존 조회 실패 | stale/실패 횟수 처리. 정상 빈 데이터로 대체 금지 |
| error-429.json | 공개 읽기 한도 초과 | Retry-After: 1 헤더와 함께 사용. 재요청 제한·한도 표시 |
| run-requires-unavailable.json | 200 부분 응답 | Run/단계는 표시, warnings와 요구 정보 미확인 표시 |
| nodes-invalid-draining.json | JSON 은 유효하지만 draining 이 boolean | 계약 검사에서 거부. 성공한 폴링으로 세지 않음 |

응답 본문 19개와 manifest 1개다. error 파일은 본문뿐이므로 HTTP 상태·헤더도
manifest 값으로 설정해야 한다. 네트워크 실패·timeout 은 JSON 으로 흉내 내지
않고 후속 테스트의 전송 계층에서 주입한다.

## 2. 함께 쓰는 스냅샷

nodes-states + runs-states + run-branch/run-drain/run-asked + asks-pending 을
같은 기준 시각에 사용한다. QUEUED 의 요구는 board=demo-board 이고 그 속성을
가진 node-02 만 다른 Run 에 임대돼 있다. 임대 없는 다른 노드를 보고
“왜 대기 중인가”가 모순되지 않게 구성했다.

node-05 는 광고 만료까지 20초라 흐려진다. node-06 의 not_after 는 기준 시각보다
30초 전이지만 ASKED 이므로 일반 lease 만료 카운트다운을 보이지 않는다.
종료 Run 에 기록된 과거 배정만으로 현재 노드 카드에 그 Run 을 다시 얹지 않는다.
현재 배정 표시는 nodes[].lease 가 출발점이다.

run-chosen-unreachable 은 별도의 표시 검증 사례이며 위 함대 스냅샷에 섞지 않는다.
invalid 응답도 정상 fixture 집합에 합치지 않는다.

## 3. obs 인수 뒤의 사용 기준

- lease:null과 중첩 requires가 병합된 코드의 형태다. 생략/평탄 파일은
  manifest의 invalid-response로 바꿨다. 정상 파서가 두 형태를 함께 수용하지 않는다.
- chosen은 현재 StepView가 false도 포함해 내보낸다. 누락을 false로 바꾸지 않는다.
- 목록 Verdict 객체·null과 submitter 문자열을 코드·관측 테스트로 확인했다.
  실제 drain 장면과 Guest 제출·QUEUED 이름 보존은 후속 검증이다.
- QUEUED 파일은 queue의 후속 동작을 위한 합성 입력이며 현재 생성 가능한
  상태의 캡처가 아니다. run-chosen-unreachable도 방어적 표시 사례다.
- submitter는 합성 Guest 라벨이다. 과거 Run의 실제 기본값은 빈 문자열이다.
- sandbox 와 데모 제출 endpoint 는 미확정이어서 샘플에 만들지 않았다.

## 4. 사용 경계

이 디렉터리는 `//go:embed static` 에 포함되지 않는다. 테스트에서만 명시적으로
읽는다. API 장애 시 자동 fallback, 실제 데이터와 혼합, 샘플 응답을 게이트
통과 증거로 사용하는 동작은 추가하지 않는다. 배포 자산으로 복사하지 않는다.
