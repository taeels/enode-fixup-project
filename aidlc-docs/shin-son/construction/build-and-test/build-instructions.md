# Build Instructions — queue

## Prerequisites
- **Build Tool**: Go 1.26 (`go.mod` 의 `toolchain go1.26.6`)
- **Dependencies**: `go.mod` 의 직접 의존 셋 — `pgx/v5` · `x/sys` · `yaml.v3`. 이 유닛이 더한 의존 0
- **Environment Variables**: 빌드에는 없음. 시험은 `ENODE_TEST_DATABASE_URL` (unit-test-instructions)
- **System Requirements**: macOS · Linux · Windows 어느 쪽이든 Go 툴체인. 크로스 빌드 네 쌍이 선다

## Build Steps

### 1. Install Dependencies
```bash
go mod download
```

### 2. Configure Environment
빌드에는 설정이 없다. 시험용 DSN 은 `docs/testdb-setup.md` 의 한 줄이다.

### 3. Build All Units
```bash
go build ./... && go vet ./... && gofmt -l ./internal ./cmd ./scripts
```

### 4. Verify Build Success
- **Expected Output**: 셋 다 출력 없음 (`gofmt -l` 이 파일 이름을 내면 정렬 안 된 파일이다)
- **Build Artifacts**: `cmd/{mediator,enode,enodectl,runctl,iapadapter}` 바이너리. `go build -o <dir> ./cmd/...`
- **Common Warnings**: 없음

## Troubleshooting

### Build Fails with Dependency Errors
- **Cause**: 모듈 캐시 없음 · 프록시 차단
- **Solution**: `go mod download` 를 먼저. `go.sum` 이 잠금이라 판이 안 움직인다

### Build Fails with Compilation Errors
- **Cause**: `internal/store` 의 겉면(`Woken` · `CreateQueuedRun` 서명)을 다른 유닛이 다르게 부른다
- **Solution**: `aidlc-docs/shin-son/construction/plans/queue-code-generation-plan.md` 3절의 겉면이 정본이다
