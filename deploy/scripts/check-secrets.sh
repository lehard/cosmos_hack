#!/usr/bin/env bash
# Проверка «в репозитории нет закрытых ключей и секретов» для make check
# (NFR-SEC-1, дополнение PRD §11.22 п. 13; кейс §3.4 «ключи отделены от кода»).
# Без сети и без новых зависимостей: только git и grep на хосте, секунды.
#
# Что ищется — в файлах под git и в новых, ещё не добавленных (не игнорируемых):
#   PEM_PRIVATE  — PEM-блок закрытого ключа: BEGIN … PRIVATE KEY (RSA, EC,
#                  OPENSSH, ENCRYPTED, PGP PRIVATE KEY BLOCK);
#   JSON_PRIVATE — JSON с закрытой частью ключа: secret_hex (формат *.key.json
#                  агента токена и demo-signer), private_key*, privateKey, secret_key
#                  со значением от 16 символов, в том числе в экранированном JSON;
#   JWK_PRIVATE  — закрытый компонент JWK ("d": длинное base64url-значение);
#   TOKEN        — токены сервисов по префиксу: github_pat_, ghp_/gho_/ghs_/ghu_/ghr_,
#                  gv_ (GitVerse), AKIA/ASIA (AWS), glpat- (GitLab), xox?- (Slack),
#                  sk-ant- (Anthropic), AIza (Google), npm_;
#   KEY_SEED     — зашитое в код зерно ключей (константа …KeySeed со строкой);
#   KEY_FILE     — файл ключа по имени: *.pem, *.key, *.p12, *.pfx, *.jks,
#                  *.keystore, id_rsa, id_ecdsa, id_ed25519, *.key.json, .env.
#
# Исключения — только осознанные, строкой в deploy/scripts/check-secrets.allow:
#   ПРАВИЛО | путь | причина
# Устаревшее исключение (правило больше не срабатывает на этом пути) — тоже
# ошибка: список не зарастает. Найденные строки печатаются без содержимого,
# только путь, номер строки и правило, чтобы секрет не попал в журнал проверки.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
ALLOW="deploy/scripts/check-secrets.allow"

# Правила по содержимому: имя и расширенное регулярное выражение (POSIX ERE).
# \\? — необязательная обратная косая перед кавычкой (JSON внутри JSON-строки).
declare -A RULES=(
  [PEM_PRIVATE]='-----BEGIN ([A-Z0-9]+ )*PRIVATE KEY( BLOCK)?-----'
  [JSON_PRIVATE]='\\?"(secret_hex|secret_key|private_key[a-z_]*|privateKey|privKey)\\?"[[:space:]]*:[[:space:]]*\\?"[0-9A-Za-z+/=_-]{16,}'
  [JWK_PRIVATE]='\\?"d\\?"[[:space:]]*:[[:space:]]*\\?"[A-Za-z0-9_-]{32,}'
  [TOKEN]='(github_pat_[A-Za-z0-9_]{20,}|gh[opsur]_[A-Za-z0-9]{30,}|(^|[^A-Za-z0-9_])gv_[A-Za-z0-9]{16,}|(AKIA|ASIA)[0-9A-Z]{16}|glpat-[A-Za-z0-9_-]{20,}|xox[abpr]-[A-Za-z0-9-]{10,}|sk-ant-[A-Za-z0-9_-]{20,}|AIza[0-9A-Za-z_-]{35}|npm_[A-Za-z0-9]{36})'
  [KEY_SEED]='[Kk]ey_?[Ss]eed[A-Za-z_]*[[:space:]]*(:?=|:)[[:space:]]*"'
)
# Правило по имени файла (ERE по пути).
KEY_FILE_RE='(^|/)(id_rsa|id_ecdsa|id_ed25519|\.env)$|\.(pem|key|p12|pfx|jks|keystore)$|\.key\.json$'

# Находки: строки «ПРАВИЛО|путь|строка».
hits=()
for rule in "${!RULES[@]}"; do
  # git grep: отслеживаемые и новые неигнорируемые файлы, двоичные пропускаются.
  while IFS= read -r m; do
    [[ -n "$m" ]] && hits+=("$rule|${m%%:*}|$(cut -d: -f2 <<<"$m")")
  done < <(git grep --untracked -nIE -e "${RULES[$rule]}" -- . | cut -d: -f1,2 || true)
done
while IFS= read -r f; do
  [[ -n "$f" ]] && hits+=("KEY_FILE|$f|-")
done < <(git ls-files --cached --others --exclude-standard | grep -E "$KEY_FILE_RE" || true)

# Исключения: «ПРАВИЛО | путь | причина»; пустые строки и # — комментарии.
declare -A allowed=() used=()
if [[ -f "$ALLOW" ]]; then
  while IFS='|' read -r r p why; do
    r="${r//[[:space:]]/}"; p="$(xargs <<<"$p")"
    [[ -z "$r" || "$r" == \#* ]] && continue
    if [[ -z "$(xargs <<<"${why:-}")" ]]; then
      echo "check-secrets: исключение без причины: $r | $p" >&2; exit 2
    fi
    allowed["$r|$p"]=1
  done < "$ALLOW"
fi

bad=0
for h in "${hits[@]}"; do
  key="${h%|*}"
  if [[ -n "${allowed[$key]:-}" ]]; then
    used["$key"]=1
  else
    IFS='|' read -r r p l <<<"$h"
    echo "СЕКРЕТ? $p:$l — правило $r"
    bad=1
  fi
done
for key in "${!allowed[@]}"; do
  if [[ -z "${used[$key]:-}" ]]; then
    echo "устаревшее исключение (больше не срабатывает): ${key/|/ | } — удалите из $ALLOW"
    bad=1
  fi
done

if (( bad )); then
  echo "check-secrets: найдены возможные закрытые ключи или секреты — уберите их из репозитория или внесите осознанное исключение с причиной в $ALLOW" >&2
  exit 1
fi
echo "check-secrets: закрытых ключей и секретов нет (исключений по списку: ${#allowed[@]})"
