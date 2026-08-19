#!/usr/bin/env bash
# 테스트용 Postgres 하나. ADR-015 §3 이 고른 저장소를 그대로 쓴다.
#
#   eval "$(scripts/testdb.sh)" && go test ./...
set -euo pipefail

NAME=enode-test-pg
PORT=${ENODE_TEST_PG_PORT:-55434}

# ★ 단위 테스트는 자기 데이터베이스를 쓴다 ★
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
