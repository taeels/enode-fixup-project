#!/usr/bin/env bash
# 테스트용 Postgres 하나. ADR-015 §3 이 고른 저장소를 그대로 쓴다.
#
#   eval "$(scripts/testdb.sh)" && go test ./...
set -euo pipefail

# 이미 잡혀 있으면 그것이 이긴다 — 그대로 되돌려 준다.
#
# 대회장에서 Docker 설치가 병목이라 각자 기계에 Postgres 를 직접 깐다
# (docs/testdb-setup.md). 그 기계에는 Docker 가 없으므로 아래를 돌 수 없는데,
# CP0 의 첫 줄은 네 기계에서 같은 글자여야 한다. 이 갈래가 그것을 지킨다 —
# 깐 방식이 무엇이든 eval "$(scripts/testdb.sh)" 하나로 선다.
if [ -n "${ENODE_TEST_DATABASE_URL:-}" ]; then
  echo "export ENODE_TEST_DATABASE_URL='${ENODE_TEST_DATABASE_URL}'"
  exit 0
fi

NAME=enode-test-pg
PORT=${ENODE_TEST_PG_PORT:-55434}

# Docker 가 없으면 조용히 넘어가지 않는다.
#
# 여기서 빈 출력을 내면 eval 이 아무것도 안 하고, 테스트는 URL 이 없는 채로
# 돌아 internal/store 가 t.Fatal 로 죽는다. 원인은 "DB 를 못 만들었다" 인데
# 메시지가 그 말을 안 한다. 그래서 stderr 로 이유와 갈 곳을 적고 실패한다.
if ! command -v docker >/dev/null 2>&1; then
  echo "testdb: docker is not installed and ENODE_TEST_DATABASE_URL is unset" >&2
  echo "testdb: install PostgreSQL 17 locally instead - see docs/testdb-setup.md" >&2
  exit 1
fi

# 단위 테스트는 자기 데이터베이스를 쓴다
# 같은 DB 를 손으로 띄운 enode 와 공유하면, 그 enode 가 계속 광고해서
# "함대에 없다(422)" 를 기대한 테스트가 조용히 201 을 받는다. 실제로 밟았다.
DB=${ENODE_TEST_DB:-enode_test}

if ! docker inspect "$NAME" >/dev/null 2>&1; then
  docker run -d --name "$NAME" \
    -e POSTGRES_USER=enode -e POSTGRES_PASSWORD=enode -e POSTGRES_DB=enode \
    -p "127.0.0.1:${PORT}:5432" postgres:17 >/dev/null
else
  docker start "$NAME" >/dev/null
fi

for _ in $(seq 1 30); do
  docker exec "$NAME" pg_isready -U enode >/dev/null 2>&1 && break
  sleep 1
done

docker exec "$NAME" psql -U enode -d postgres -tAc \
  "SELECT 1 FROM pg_database WHERE datname='${DB}'" 2>/dev/null | grep -q 1 ||
  docker exec "$NAME" createdb -U enode "$DB" >/dev/null 2>&1 || true

echo "export ENODE_TEST_DATABASE_URL='postgres://enode:enode@127.0.0.1:${PORT}/${DB}?sslmode=disable'"
