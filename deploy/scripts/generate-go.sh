#!/usr/bin/env bash
# Цепочка генерации Go (make generate, AD-20; О7, О8). Выполняется внутри golang:1.27.1
# из Makefile (/src — корень репозитория). Порядок:
#   1) archgen: правила слоёв backend/.golangci.yml из tools/archgen/layers.json;
#   2) contractgen: каталог типов, коды ошибок, статусы, константы, повышатели,
#      реестр «тип+версия → Go-тип», структуры и XSD расширения BPMN, копия схем для встраивания;
#   3) go-jsonschema: Go-типы из JSON Schema (события, запись журнала, криптоформаты,
#      контракты собственных процессов, затравка нормативного слоя, problem+json);
#   4) sqlc: для каждого backend/internal/infrastructure/storage/*/sqlc.yaml;
#   5) contracts/openapi.yaml из операций Huma (ant -openapi).
# Результат детерминирован: make check перегенерирует и требует git diff --exit-code.
set -euo pipefail
cd /src/backend
bin=$(/src/deploy/scripts/go-tools.sh ./archgen ./contractgen github.com/atombender/go-jsonschema github.com/sqlc-dev/sqlc/cmd/sqlc)

step() { printf '== generate: %s\n' "$*" >&2; }

step "archgen → backend/.golangci.yml"
(cd tools && "$bin/archgen" -layers archgen/layers.json -out ../.golangci.yml)

step "contractgen"
(cd tools && "$bin/contractgen" -root /src)

# go-jsonschema: шапка «источник» дописывается после строки Code generated.
B=https://ant.invalid/contracts
OUT=/src/backend/internal/contracts
JS=("$bin/go-jsonschema" --only-models -t --tags json --capitalization ID --capitalization URL --capitalization UUID --capitalization JSON --capitalization HTTP)

stamp() { # stamp <файл> <источник>
  sed -i "1a // Источник: $2 (make generate). Руками не править (AD-20)." "$1"
}

cd /src/contracts
step "go-jsonschema: events"
mkdir -p "$OUT/events"
"${JS[@]}" -p events -o "$OUT/events/events_gen.go" events/common/*.json events/*/*.v*.json
stamp "$OUT/events/events_gen.go" "contracts/events/common/*.json, contracts/events/‹семейство›/*.v‹N›.json"

step "go-jsonschema: crypto, journal, procs, normative, problem"
mkdir -p "$OUT/crypto" "$OUT/journal" "$OUT/procs" "$OUT/normative" "$OUT/problem"
args=()
map() { # map <путь схемы от contracts> <пакет>
  args+=(--schema-package="$B/$1=ant/internal/contracts/$2" --schema-output="$B/$1=$OUT/$2/$2_gen.go")
  files+=("$1")
}
files=()
map crypto/dsse-envelope.schema.json crypto
map crypto/checkpoint.schema.json crypto
map journal/entry.schema.json journal
for f in internal/token-agent/*.json internal/verifier/*.json internal/stands/*.json; do map "$f" procs; done
for f in normative/*.schema.json; do map "$f" normative; done
map problem.schema.json problem
"${JS[@]}" -e -p misc "${args[@]}" "${files[@]}"
stamp "$OUT/crypto/crypto_gen.go" "contracts/crypto/{dsse-envelope,checkpoint}.schema.json"
stamp "$OUT/journal/journal_gen.go" "contracts/journal/entry.schema.json"
stamp "$OUT/procs/procs_gen.go" "contracts/internal/{token-agent,verifier,stands}/*.json"
stamp "$OUT/normative/normative_gen.go" "contracts/normative/*.schema.json"
stamp "$OUT/problem/problem_gen.go" "contracts/problem.schema.json"

cd /src/backend
step "sqlc"
shopt -s nullglob
for cfg in internal/infrastructure/storage/*/sqlc.yaml; do
  (cd "$(dirname "$cfg")" && "$bin/sqlc" generate -f sqlc.yaml)
done

step "gofmt сгенерированного"
gofmt -w internal/contracts

step "contracts/openapi.yaml из операций Huma"
go run ./cmd/ant -openapi /src/contracts/openapi.yaml
