#!/bin/sh
# Разовая подготовка секретов демо-стенда (compose-служба secrets-init).
# Пароль БД генерируется при первом запуске и хранится только в томе secrets:
#   postgres/db_password — владелец postgres (uid 70 в образе alpine), 0400;
#   ant/db_password      — владелец ant (uid 65532, distroless nonroot), 0400.
# В репозитории и в переменных окружения секретов нет (AD-25).
set -eu
umask 077
if [ ! -s /secrets/postgres/db_password ]; then
  mkdir -p /secrets/postgres /secrets/ant
  pw=$(head -c 48 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 32)
  printf '%s' "$pw" > /secrets/postgres/db_password
  printf '%s' "$pw" > /secrets/ant/db_password
  echo "secrets-init: создан пароль БД"
else
  echo "secrets-init: секреты уже есть"
fi
chown -R 70:70 /secrets/postgres
chown -R 65532:65532 /secrets/ant
chmod 0500 /secrets/postgres /secrets/ant
chmod 0400 /secrets/postgres/db_password /secrets/ant/db_password
chmod 0755 /secrets
