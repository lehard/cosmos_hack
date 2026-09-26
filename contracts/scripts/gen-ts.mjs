// Генерация TypeScript-типов из JSON Schema контрактов (AD-20, NFR-DEV-2; make generate).
//
//   node contracts/scripts/gen-ts.mjs          — перегенерировать frontend/src/shared/contracts/
//
// Что генерируется (json-schema-to-typescript, версия закреплена в package.json):
//   events.ts    — данные всех типов записей журнала, конверт, общие определения, сообщение SSE;
//   journal.ts   — запись журнала (AD-44);
//   crypto.ts    — конверт DSSE и контрольная точка хранителя (AD-10, AD-8);
//   procs.ts     — контракты собственных процессов: агент токена, отчёт верификатора, телеметрия (AD-46);
//   constants.ts — константы контракта (contracts/constants.yaml);
//   catalog.ts   — каталог типов записей: эмитент, вид, поток, ось, класс, критичность (AD-40).
// Клиент API (orval), словарь статусов и коды ошибок генерирует frontend/scripts/generate.mjs.
// Источник пишется в шапку каждого файла; руками не править.
import { mkdirSync, readdirSync, readFileSync, statSync, writeFileSync } from 'node:fs'
import { dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { compile } from 'json-schema-to-typescript'
import { parse } from 'yaml'

const CONTRACTS = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const REPO = resolve(CONTRACTS, '..')
const OUT = join(REPO, 'frontend/src/shared/contracts')

const banner = (src) =>
  `// СГЕНЕРИРОВАНО contracts/scripts/gen-ts.mjs (make generate) — руками не править (AD-20).\n// Источник: ${src}\n`

function walk(dir) {
  return readdirSync(dir).flatMap((n) => {
    const p = join(dir, n)
    return statSync(p).isDirectory() ? walk(p) : [p]
  })
}

/** Схемы группы — одним корнем с definitions: общие типы объявляются один раз. */
async function group(name, files, cwd, src) {
  const definitions = {}
  for (const f of files.sort()) {
    const s = JSON.parse(readFileSync(f, 'utf8'))
    definitions[s.title] = { $ref: './' + relative(cwd, f) }
  }
  const root = { title: `${name}Contracts`, type: 'object', additionalProperties: false, properties: {}, definitions }
  let ts = await compile(root, `${name}Contracts`, {
    cwd,
    bannerComment: banner(src),
    unreachableDefinitions: true,
    declareExternallyReferenced: true,
    additionalProperties: false,
    format: false,
    strictIndexSignatures: true,
    $refOptions: { resolve: { external: true } },
  })
  // Пустой корневой интерфейс-контейнер не нужен.
  ts = ts.replace(new RegExp(`export interface ${name}Contracts \\{\\s*\\}\\n?`), '')
  writeFileSync(join(OUT, `${name.toLowerCase()}.ts`), ts)
}

mkdirSync(OUT, { recursive: true })

const ev = join(CONTRACTS, 'events')
const eventFiles = walk(ev).filter((f) => /\.v\d+\.json$/.test(f) && !f.includes('/examples/'))
await group('Events', eventFiles, ev, 'contracts/events/common/*.json, contracts/events/‹семейство›/*.v‹N›.json')
await group('Journal', [join(CONTRACTS, 'journal/entry.schema.json')], join(CONTRACTS, 'journal'), 'contracts/journal/entry.schema.json')
await group('Crypto', [join(CONTRACTS, 'crypto/dsse-envelope.schema.json'), join(CONTRACTS, 'crypto/checkpoint.schema.json')], join(CONTRACTS, 'crypto'), 'contracts/crypto/*.schema.json')
const internal = join(CONTRACTS, 'internal')
await group('Procs', walk(internal).filter((f) => f.endsWith('.json')), internal, 'contracts/internal/**/*.json')

// Константы.
const consts = parse(readFileSync(join(CONTRACTS, 'constants.yaml'), 'utf8'))
let c = banner('contracts/constants.yaml') + '\n'
for (const [k, v] of Object.entries(consts)) c += `export const ${k} = ${JSON.stringify(v)} as const\n`
writeFileSync(join(OUT, 'constants.ts'), c)

// Каталог типов записей.
const cat = parse(readFileSync(join(CONTRACTS, 'events/catalog.yaml'), 'utf8'))
const rows = Object.fromEntries(
  Object.entries(cat.types)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([t, d]) => [t, { title: d.title, emitter: d.emitter, kind: d.kind, stream: d.stream, axis: d.axis, actionClass: d.action_class, critical: d.critical, caGroup: d.ca_group ?? null, currentVersion: d.current_version }]),
)
writeFileSync(
  join(OUT, 'catalog.ts'),
  banner('contracts/events/catalog.yaml') +
    `\n/** Каталог типов записей журнала (AD-40): эмитент, вид, поток, ось статуса, класс действия, критичность. */\n` +
    `export const eventCatalog = ${JSON.stringify(rows, null, 2)} as const\n\n` +
    `/** Тип записи журнала \`семейство.сущность.действие\`. */\nexport type EventType = keyof typeof eventCatalog\n`,
)
writeFileSync(join(OUT, '.gitattributes'), '* linguist-generated=true\n')
console.log(`сгенерировано: ${relative(REPO, OUT)}`)
