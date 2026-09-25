#!/usr/bin/env bash
# Проверка происхождения GoGOST 7.0.0 в third_party/gogost (см. SOURCE).
#
#   verify.sh              офлайн-проверки: sha256 архива, подпись ed25519 из
#                          .meta4, подпись ключа SSH ключом PGP автора (если есть
#                          gpg), побайтовое совпадение вложенного кода с архивом
#   verify.sh --hybrid     + подпись .sig (ssh-mldsa44-ed25519) — OpenSSH ≥ 10.4
#                          в одноразовом контейнере alpine:edge (нужны docker и сеть)
#   verify.sh --deckhouse  + сверка кода с github.com/deckhouse/gogost/v6 v6.2.0
#                          (proxy.golang.org, нужна сеть)
#
# Нужны на машине: bash, tar с zstd, sha256sum, ssh-keygen (OpenSSH ≥ 8.1).
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd)
up="$here/upstream"
archive="$up/gogost-7.0.0.tar.zst"

# Закреплённые значения (дублируют SOURCE).
SHA256=225aba5cc409546401a7ae37a12d506ee06ce91a4c29b164d8c5d30b7f9ce659
SSH_ED25519_FP=SHA256:u8X9rPDOhxpyzGs/IugbxXbDeOu/0AttKY+LGAvHBH0
SSH_HYBRID_FP=SHA256:BNLMGBzQJNfWuPDr602Cn7ExhEip3sqLSKJgpAlGgLE
PGP_PRIMARY=12AD32689C660D426967FD75CB8205632107AD8A
DECKHOUSE_ZIP_SHA256=deb2cd7287916e1f712cc9a315599808623eb9ab145067a1d2c56bc0c9d02614

hybrid=0 deckhouse=0
for a in "$@"; do
  case "$a" in
    --hybrid) hybrid=1 ;;
    --deckhouse) deckhouse=1 ;;
    *) echo "неизвестный флаг $a" >&2; exit 2 ;;
  esac
done

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
ok() { echo "  ок: $*"; }

echo "== GoGOST 7.0.0: проверка происхождения"

# 1. sha256 архива.
got=$(sha256sum "$archive" | cut -d' ' -f1)
[[ $got == "$SHA256" ]] || { echo "sha256 архива $got ≠ $SHA256" >&2; exit 1; }
ok "sha256 архива $SHA256"
grep -q "<hash type=\"sha-256\">$SHA256</hash>" "$up/gogost-7.0.0.tar.zst.meta4" \
  || { echo "sha256 не совпадает с .meta4" >&2; exit 1; }
ok "sha256 совпадает с .meta4 автора"

# 2. Подпись ed25519 из .meta4 ключом PUBKEY-SSH.pub.
sed -n '/-----BEGIN SSH SIGNATURE-----/,/-----END SSH SIGNATURE-----/p' "$up/gogost-7.0.0.tar.zst.meta4" > "$work/meta4.sig"
printf 'gogost@stargrave.org %s\n' "$(cut -d' ' -f2- "$here/PUBKEY-SSH.pub")" > "$work/allowed"
fp=$(ssh-keygen -lf "$here/PUBKEY-SSH.pub" | awk '{print $2}')
[[ $fp == "$SSH_ED25519_FP" ]] || { echo "отпечаток PUBKEY-SSH.pub $fp ≠ $SSH_ED25519_FP" >&2; exit 1; }
ssh-keygen -Y verify -f "$work/allowed" -I gogost@stargrave.org -n file -s "$work/meta4.sig" < "$archive" >/dev/null
ok "подпись ed25519 (.meta4) верна, ключ $SSH_ED25519_FP"

# 3. PUBKEY-SSH.pub подписан ключом PGP автора (Sergey Matveev).
if command -v gpg >/dev/null 2>&1; then
  export GNUPGHOME="$work/gnupg"; mkdir -m 700 "$GNUPGHOME"
  gpg --quiet --import "$up"/stargrave-pgp-*.asc 2>/dev/null
  status=$(gpg --status-fd 1 --verify "$here/PUBKEY-SSH.pub.asc" "$here/PUBKEY-SSH.pub" 2>/dev/null || true)
  grep -q "^\[GNUPG:\] VALIDSIG .* $PGP_PRIMARY\$" <<<"$status" \
    || { echo "подпись PGP ключа SSH не подтверждена ключом $PGP_PRIMARY" >&2; exit 1; }
  ok "PUBKEY-SSH.pub подписан ключом PGP автора $PGP_PRIMARY"
else
  echo "  пропуск: нет gpg — подпись PGP ключа SSH не проверена"
fi

# 4. Вложенный код побайтно совпадает с архивом.
tar --zstd -xf "$archive" -C "$work"
src="$work/gogost-7.0.0"
n=0
while IFS= read -r f; do
  rel=${f#"$here"/}
  cmp -s "$f" "$src/$rel" || { echo "файл $rel отличается от архива" >&2; exit 1; }
  n=$((n + 1))
done < <(find "$here" -type f ! -path "$up/*" ! -name SOURCE ! -name verify.sh)
ok "$n файлов third_party/gogost побайтно совпадают с архивом"

# 5. Гибридная подпись .sig (ssh-mldsa44-ed25519): OpenSSH ≥ 10.4.
if [[ $hybrid == 1 ]]; then
  docker_sh="$here/../../deploy/scripts/docker.sh"
  out=$("$docker_sh" run --rm -v "$up:/w:ro" alpine:edge sh -c \
    'apk add -q openssh-keygen >/dev/null 2>&1 && ssh -V 2>/dev/null; ssh-keygen -Y check-novalidate -n file -s /w/gogost-7.0.0.tar.zst.sig < /w/gogost-7.0.0.tar.zst' 2>&1)
  grep -q "Good \"file\" signature with MLDSA44-ED25519 key $SSH_HYBRID_FP" <<<"$out" \
    || { echo "гибридная подпись не подтверждена:"; echo "$out"; exit 1; } >&2
  ok "подпись .sig (ML-DSA-44 + Ed25519) верна, ключ $SSH_HYBRID_FP"
fi

# 6. Сверка с github.com/deckhouse/gogost/v6 v6.2.0 (зеркало на proxy.golang.org).
if [[ $deckhouse == 1 ]]; then
  curl -fsS -m 60 -o "$work/dh.zip" https://proxy.golang.org/github.com/deckhouse/gogost/v6/@v/v6.2.0.zip
  [[ $(sha256sum "$work/dh.zip" | cut -d' ' -f1) == "$DECKHOUSE_ZIP_SHA256" ]] || { echo "sha256 zip deckhouse изменился" >&2; exit 1; }
  (cd "$work" && mkdir dh && cd dh && unzip -q ../dh.zip)
  dh="$work/dh/github.com/deckhouse/gogost/v6@v6.2.0"
  mkdir "$work/n1" "$work/n2"
  for p in gogost.go gost3410 gost34112012256 gost34112012512 internal/gost34112012 gost28147 gost341194; do
    mkdir -p "$work/n1/$(dirname $p)" "$work/n2/$(dirname $p)"
    cp -r "$src/$p" "$work/n1/$p"; cp -r "$dh/$p" "$work/n2/$p"
  done
  chmod -R u+w "$work/n1" "$work/n2"
  find "$work/n1" "$work/n2" -name '*.go' -exec sed -i -E \
    's|github.com/deckhouse/gogost/v6|go.stargrave.org/gogost/v7|g; s|Copyright \(C\) 2015-20[0-9]{2}|Copyright (C) 2015-YYYY|' {} +
  (cd "$work" && diff -r n2 n1 > diff.txt || true)
  diff -q "$work/diff.txt" "$up/deckhouse-v6.2.0-normalized.diff" >/dev/null \
    || { echo "отличия от deckhouse v6.2.0 изменились:"; cat "$work/diff.txt"; exit 1; } >&2
  ok "код совпадает с deckhouse/gogost/v6 v6.2.0 с точностью до путей импорта, годов и upstream/deckhouse-v6.2.0-normalized.diff"
fi
