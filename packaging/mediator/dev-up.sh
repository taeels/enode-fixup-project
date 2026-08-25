#!/usr/bin/env bash
# 실측용 Mediator 를 CT103 에 띄운다. 시연·실측 전용이고 운영 배포가 아니다.
#
#   packaging/mediator/dev-up.sh          띄운다 (postgres 포함)
#   packaging/mediator/dev-up.sh down     내린다
#
# 테스트 DB 와 갈라 쓴다 — scripts/testdb.sh 의 enode_test 를 공유하면
# 여기서 띄운 enode 가 계속 광고해서 "함대에 없다(422)" 를 기대한 단위 테스트가
# 조용히 201 을 받는다 (README 가 실제로 밟았다고 적어둔 사고다).
set -euo pipefail

cd "$(dirname "$0")/../.."
ROOT=$(pwd)

NAME=enode-dev-pg
PORT=${ENODE_DEV_PG_PORT:-55435}
DB=enode_dev
STATE=${ENODE_DEV_STATE:-$HOME/.local/state/enode-dev}
CONF=$STATE/mediator.yaml
LOG=$STATE/mediator.log
PIDF=$STATE/mediator.pid

if [ "${1:-up}" = "down" ]; then
  [ -f "$PIDF" ] && kill "$(cat "$PIDF")" 2>/dev/null && rm -f "$PIDF" && echo "mediator 내렸다"
  docker stop "$NAME" >/dev/null 2>&1 && echo "postgres 내렸다"
  exit 0
fi

mkdir -p "$STATE" "$STATE/artifacts"

echo "== postgres =="
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
  docker exec "$NAME" createdb -U enode "$DB"
echo "   ok  127.0.0.1:${PORT}/${DB}"

echo "== 토큰 =="
# 토큰은 한 번 뽑아 파일에 남긴다 — 맥 설정에 같은 값을 적어야 하는데
# 매번 새로 뽑으면 맥이 조용히 401 을 받는다.
if [ ! -f "$STATE/token" ]; then
  head -c 24 /dev/urandom | base64 | tr -d '/+=' > "$STATE/token"
  chmod 600 "$STATE/token"
fi
TOKEN=$(cat "$STATE/token")
echo "   $STATE/token"

echo "== 설정 =="
cat > "$CONF" <<YAML
# listen 이 전 인터페이스여야 한다 — 127.0.0.1 로 좁히면 맥에서 못 닿는다.
listen: ":8080"
token: "$TOKEN"
database:
  url: "postgres://enode:enode@127.0.0.1:${PORT}/${DB}?sslmode=disable"
artifacts:
  root: "$STATE/artifacts"
YAML
chmod 600 "$CONF"
echo "   $CONF"

echo "== mediator =="
if [ -f "$PIDF" ] && kill -0 "$(cat "$PIDF")" 2>/dev/null; then
  echo "   이미 돈다 (pid=$(cat "$PIDF"))"
else
  go build -o "$STATE/mediator" ./cmd/mediator
  nohup "$STATE/mediator" --config "$CONF" >> "$LOG" 2>&1 &
  echo $! > "$PIDF"
  sleep 2
  kill -0 "$(cat "$PIDF")" 2>/dev/null || { echo "✗ 못 떴다:"; tail -20 "$LOG"; exit 1; }
  echo "   떴다 pid=$(cat "$PIDF")"
fi

IP=$(ip -4 -o addr show scope global 2>/dev/null | awk '{print $4}' | cut -d/ -f1 | grep -v '^172\.' | head -1)
echo
echo "== 맥에 적을 것 =="
echo "   mediator: http://${IP:-<이 기계의 LAN IP>}:8080"
echo "   token:    $TOKEN"
echo
echo "== 여기서 확인 =="
echo "   export ENODE_MEDIATOR=http://127.0.0.1:8080 ENODE_TOKEN=$TOKEN"
echo "   runctl capabilities"
echo "   로그: $LOG"
