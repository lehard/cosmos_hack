#!/usr/bin/env bash
# Своя БД для каждого агента (make dev-db; AD-25): общий dev-postgres на машине
# (контейнер ant-dev-postgres, 127.0.0.1:55432) и отдельная база ant_<ветка>.
#
#   dev-db.sh create   поднять dev-postgres при необходимости и создать БД
#   dev-db.sh drop     удалить БД ветки
#   dev-db.sh psql     psql в БД ветки
#
# Параметры подключения пишутся в .dev/db.env (переменные ANT_DB_*); контейнеры
# Go из Makefile подхватывают их сами и подключаются к сети docker ant-dev, где
# сервер доступен по имени ant-dev-postgres:5432 (не через loopback машины —
# на машине сборки он может быть закрыт для пользователя). С машины сервер
# доступен на 127.0.0.1:55432. Пароль — файлом ~/.config/ant-dev/pg_password
# (0600), в переменных его нет.
set -euo pipefail
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
DOCKER="$ROOT/deploy/scripts/docker.sh"
CONTAINER=${ANT_DEV_PG_CONTAINER:-ant-dev-postgres}
PORT=${ANT_DEV_PG_PORT:-55432}
IMAGE=${PG_IMAGE:-postgres:18.6-alpine}
ADMIN=ant_admin
NETWORK=${ANT_DEV_NETWORK:-ant-dev}
SECRETS="$HOME/.config/ant-dev"
PWFILE="$SECRETS/pg_password"

branch=$(git -C "$ROOT" rev-parse --abbrev-ref HEAD 2>/dev/null || echo local)
db="ant_$(tr '[:upper:]' '[:lower:]' <<<"$branch" | sed -E 's/[^a-z0-9]+/_/g; s/^_+|_+$//g')"
db=${ANT_DEV_DB_NAME:-${db:0:63}}

ensure_server() {
  # Несколько агентов могут звать make dev-db одновременно — запуск сервера
  # и создание БД идут под своей блокировкой.
  mkdir -p "$HOME/.cache"
  exec 9>"$HOME/.cache/ant-dev-db.lock"
  flock 9
  mkdir -p "$SECRETS"; chmod 700 "$SECRETS"
  if [[ ! -s $PWFILE ]]; then
    (umask 077; head -c 32 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 32 > "$PWFILE")
  fi
  "$DOCKER" network inspect "$NETWORK" >/dev/null 2>&1 \
    || "$DOCKER" network create --label ant.project=ant "$NETWORK" >/dev/null
  state=$("$DOCKER" inspect -f '{{.State.Status}}' "$CONTAINER" 2>/dev/null || true)
  case "$state" in
    running) ;;
    "")
      echo "dev-db: запускаю $CONTAINER ($IMAGE) на 127.0.0.1:$PORT"
      # Пароль передаётся переменной только внутрь контейнера dev-сервера:
      # файл 0600 владельца машины не читается пользователем postgres в образе.
      "$DOCKER" --lock run -d --name "$CONTAINER" --restart unless-stopped \
        --memory 384m -p "127.0.0.1:$PORT:5432" --network "$NETWORK" \
        -e POSTGRES_USER=$ADMIN -e POSTGRES_DB=postgres -e POSTGRES_PASSWORD="$(cat "$PWFILE")" \
        --label ant.project=ant -v ant-dev-pgdata:/var/lib/postgresql \
        "$IMAGE" postgres -c shared_buffers=64MB -c max_connections=100 -c work_mem=4MB >/dev/null
      ;;
    *) "$DOCKER" start "$CONTAINER" >/dev/null ;;
  esac
  # Сервер, созданный до появления сети, подключаем к ней (повторно — без ошибки).
  "$DOCKER" network connect "$NETWORK" "$CONTAINER" >/dev/null 2>&1 || true
  for _ in $(seq 60); do
    "$DOCKER" exec "$CONTAINER" pg_isready -q -U $ADMIN -d postgres && return 0
    sleep 1
  done
  echo "dev-db: $CONTAINER не ответил за 60 с" >&2; exit 1
}

psql_admin() { "$DOCKER" exec -i "$CONTAINER" psql -v ON_ERROR_STOP=1 -qtA -U $ADMIN -d postgres "$@"; }

write_env() {
  mkdir -p "$ROOT/.dev"
  cat > "$ROOT/.dev/db.env" <<ENV
# make dev-db: БД ветки $branch. Адрес и путь к паролю — для контейнеров
# Makefile (сеть $NETWORK, каталог $SECRETS → /run/ant-dev).
# С машины: postgres://$ADMIN@127.0.0.1:$PORT/$db, пароль в $PWFILE.
ANT_DB_HOST=$CONTAINER
ANT_DB_PORT=5432
ANT_DB_NAME=$db
ANT_DB_USER=$ADMIN
ANT_DB_PASSWORD_FILE=/run/ant-dev/pg_password
ENV
}

case "${1:-create}" in
  create)
    ensure_server
    if [[ -z "$(psql_admin -c "SELECT 1 FROM pg_database WHERE datname = '$db'")" ]]; then
      psql_admin -c "CREATE DATABASE \"$db\"" >/dev/null
      echo "dev-db: создана БД $db"
    else
      echo "dev-db: БД $db уже есть"
    fi
    write_env
    echo "dev-db: параметры → .dev/db.env"
    echo "  postgres://$ADMIN@127.0.0.1:$PORT/$db  (пароль: $PWFILE)"
    ;;
  drop)
    ensure_server
    psql_admin -c "DROP DATABASE IF EXISTS \"$db\" WITH (FORCE)" >/dev/null
    rm -f "$ROOT/.dev/db.env"
    echo "dev-db: удалена БД $db"
    ;;
  psql)
    ensure_server
    flock -u 9
    exec "$DOCKER" exec -it "$CONTAINER" psql -U $ADMIN -d "$db"
    ;;
  *) echo "использование: $0 create|drop|psql" >&2; exit 2 ;;
esac
