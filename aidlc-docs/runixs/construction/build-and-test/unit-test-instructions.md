# 단위·경계 검사

저장소 루트에서 전용 DB를 준비한다. CI와 동일한 Claude 스텁을 먼저 PATH에 둔다.
운영 DB와 공유하지 않는다. 시스템 Go가 없으면 Mac mini의 별도 검증 디렉터리에서
실행한다. 스텁은 검사 환경에만 적용하며 운영 Bedrock 설정에 넣지 않는다.

```sh
eval "$(scripts/testdb.sh)"
export PATH="$PWD/.github/ci-stubs:$PATH"
chmod +x .github/ci-stubs/claude
go test ./... -count=1 -race -coverpkg=./... -coverprofile=/tmp/gallery-cover.out -json > /tmp/gallery-tests.jsonl
node --test internal/api/ui/tests/*.test.mjs
python3 -m unittest discover -s scripts/gallery_demo -p 'test_*.py'
```

합격 조건은 실패·예상 밖 skip 0이다. .github/workflows/ci.yml의 커버리지 블록
집계로 각 패키지 80% 이상을 확인한다. 전체 합계만으로 하한을 대신하지 않는다.
표준 CI는 race 없는 전체 coverage도 별도로 실행한다.

추가 경계는 대상/소유 UUID 검증, 고정 계약, 게시 본문 변경 거부, HMAC 위조·만료,
POST 전 시도 기록, 중복·재시작·응답 유실 뒤 재게시 방지, MCP 도구/인자 범위다.
UI 검사는 프롬프트 한 번·실제 게시 결과·초안 전용·되묻기·같은 요청 재시도·
새로고침 후 중복 접수 방지를 다룬다. 새 댓글 grant의 대상/본문 고정과 한 번 게시,
MCP의 목록 선행 조회·목록 밖 대상 거부도 포함한다.

최신 main의 제어판 검사도 함께 실행한다. macOS의 셸 exec 최적화로 설정 argv를
잃던 보조 프로세스는 전용 테스트 프로세스로 고쳤다. cmd/enode의 실제 panel
진입점에서 설정·정책 오류와 인증 없는 외부 바인딩·포트 충돌이 실패하는지 확인한다.
제품 제어판 코드는 바꾸지 않는다. 결과와 수정 이유는 [요약](build-and-test-summary.md)에 있다.
