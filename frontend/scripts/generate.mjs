// Генерация клиентского кода интерфейса ant из контрактов (AD-20, NFR-DEV-2, критерий О7).
//
//   node scripts/generate.mjs          — перегенерировать src/shared/api/generated/
//   node scripts/generate.mjs --check  — сгенерировать во временную папку и сверить
//                                        с зафиксированной (make check: «клиент не устарел»)
//
// Что генерируется:
//   client.ts + model/ — клиент orval + Vue Query (fetch) из OpenAPI;
//   statuses.ts        — словарь статусов и палитра тонов (AD-30) из statuses.yaml;
//   errors.ts          — код ошибки → ключ текста интерфейса из errors.yaml (FR-28);
//   stream.ts          — адрес SSE-канала живых обновлений из events/asyncapi.yaml (AD-21).
//
// Источники — контракты в contracts/ (корень репозитория). openapi.yaml генерирует
// эпик 02 из Go-описаний Huma; пока его нет, берётся черновик
// frontend/dev/openapi.draft.yaml, а переключение на настоящий файл — само, по его
// появлению. Источник пишется в шапку каждого файла.
import { existsSync, mkdtempSync, mkdirSync, readFileSync, readdirSync, rmSync, statSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { generate } from 'orval'
import { parse } from 'yaml'

const FRONTEND = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const REPO = resolve(FRONTEND, '..')
const GENERATED = join(FRONTEND, 'src/shared/api/generated')
const check = process.argv.includes('--check')

/** Настоящий контракт; если его ещё нет и есть черновик — черновик из frontend/dev. */
function source(contract, draft = null) {
  const real = join(REPO, 'contracts', contract)
  const path = existsSync(real) || !draft ? real : join(FRONTEND, 'dev', draft)
  if (!existsSync(path)) throw new Error(`нет контракта ${relative(REPO, real)}`)
  return { path, rel: relative(REPO, path), draft: path !== real }
}

const sources = {
  openapi: source('openapi.yaml', 'openapi.draft.yaml'),
  statuses: source('statuses.yaml'),
  errors: source('errors.yaml'),
  asyncapi: source('events/asyncapi.yaml'),
}

const header = (src) =>
  `// СГЕНЕРИРОВАНО frontend/scripts/generate.mjs — руками не править (AD-20).\n` +
  `// Источник: ${src.rel}${src.draft ? ' (черновик до появления контракта)' : ''}\n`

async function run(outDir) {
  rmSync(outDir, { recursive: true, force: true })
  mkdirSync(outDir, { recursive: true })

  // Клиент: fetch без ручного кода (единственный ручной сетевой модуль — shared/api/sse).
  // forceSuccessResponse — ответ не 2xx бросает ошибку с info = problem+json и status,
  // так Vue Query видит ошибку входа; includeHttpResponseReturnType — у ответа есть
  // headers, из них читается режим Ant-Backend (fixtures | live) для метки виджета.
  await generate({
    input: { target: sources.openapi.path },
    output: {
      mode: 'split',
      target: join(outDir, 'client.ts'),
      schemas: join(outDir, 'model'),
      client: 'vue-query',
      httpClient: 'fetch',
      clean: false,
      prettier: false,
      override: {
        header: () => [
          'СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).',
          `Источник: ${sources.openapi.rel}${sources.openapi.draft ? ' (черновик до появления contracts/openapi.yaml)' : ''}`,
        ],
        fetch: { includeHttpResponseReturnType: true, forceSuccessResponse: true },
        query: { version: 5, signal: true, shouldExportKeys: true },
      },
    },
  })

  // Словарь статусов (AD-30): коды осей, тексты по умолчанию и тона цвета.
  const st = parse(readFileSync(sources.statuses.path, 'utf8'))
  const axes = {}
  for (const [axis, def] of Object.entries(st.axes ?? {})) {
    axes[axis] = { title: def.title, owner: def.owner, values: Object.fromEntries(def.values.map((v) => [v.code, { label: v.label, tone: v.tone }])) }
  }
  const dictionaries = {}
  for (const [name, def] of Object.entries(st.dictionaries ?? {})) {
    dictionaries[name] = { title: def.title, values: Object.fromEntries(def.values.map((v) => [v.code, { label: v.label, tone: v.tone }])) }
  }
  const palette = Object.fromEntries(Object.entries(st.palette).map(([tone, p]) => [tone, p.color]))
  writeFileSync(
    join(outDir, 'statuses.ts'),
    header(sources.statuses) +
      `\n/** Палитра тонов: цвет — только для статуса и главного действия (NFR-UI-2). */\n` +
      `export const statusPalette = ${JSON.stringify(palette, null, 2)} as const\n\n` +
      `/** Тон цвета статуса. */\nexport type StatusTone = keyof typeof statusPalette\n\n` +
      `/** Оси статуса изделия (PRD §3b): у каждой один модуль-владелец. */\n` +
      `export const statusAxes = ${JSON.stringify(axes, null, 2)} as const\n\n` +
      `/** Ось статуса. */\nexport type StatusAxis = keyof typeof statusAxes\n\n` +
      `/** Дополнительные словари — не оси, но едины для всех экранов. */\n` +
      `export const statusDictionaries = ${JSON.stringify(dictionaries, null, 2)} as const\n`,
  )

  // Коды ошибок (FR-28): код → HTTP-статус и ключ текста интерфейса.
  const er = parse(readFileSync(sources.errors.path, 'utf8'))
  const codes = Object.fromEntries(
    Object.entries(er.codes).map(([code, c]) => [code, { status: c.status, title: c.title, uiKey: c.ui_key ?? null }]),
  )
  writeFileSync(
    join(outDir, 'errors.ts'),
    header(sources.errors) +
      `\n/** Каталог кодов ошибок: код → HTTP-статус, заголовок и ключ текста интерфейса. */\n` +
      `export const errorCatalog = ${JSON.stringify(codes, null, 2)} as const\n\n` +
      `/** Код ошибки из каталога. */\nexport type ErrorCode = keyof typeof errorCatalog\n`,
  )
  // Канал живых обновлений (AD-21): адрес — из AsyncAPI, а не из имени операции.
  const aa = parse(readFileSync(sources.asyncapi.path, 'utf8'))
  const address = aa.channels?.sse?.address
  if (typeof address !== 'string') throw new Error(`${sources.asyncapi.rel}: нет канала sse с адресом`)
  writeFileSync(
    join(outDir, 'stream.ts'),
    header(sources.asyncapi) +
      `\n/** Адрес SSE-канала живых обновлений столов (канал \`sse\`). */\n` +
      `export const SSE_ADDRESS = ${JSON.stringify(address)}\n`,
  )
  writeFileSync(join(outDir, '.gitattributes'), '* linguist-generated=true\n')
}

function listFiles(dir, base = dir) {
  return readdirSync(dir).flatMap((name) => {
    const p = join(dir, name)
    return statSync(p).isDirectory() ? listFiles(p, base) : [relative(base, p)]
  })
}

for (const [kind, src] of Object.entries(sources)) {
  console.log(`${kind}: ${src.rel}${src.draft ? ' — черновик' : ''}`)
}

if (!check) {
  await run(GENERATED)
  console.log(`сгенерировано: ${relative(REPO, GENERATED)}`)
} else {
  const tmp = mkdtempSync(join(tmpdir(), 'ant-gen-'))
  const out = join(tmp, 'generated')
  try {
    await run(out)
    const want = listFiles(out).sort()
    const have = existsSync(GENERATED) ? listFiles(GENERATED).filter((f) => f !== '.gitkeep').sort() : []
    const diff = []
    for (const f of new Set([...want, ...have])) {
      const a = want.includes(f) ? readFileSync(join(out, f), 'utf8') : null
      const b = have.includes(f) ? readFileSync(join(GENERATED, f), 'utf8') : null
      if (a !== b) diff.push(a === null ? `лишний: ${f}` : b === null ? `нет: ${f}` : `устарел: ${f}`)
    }
    if (diff.length) {
      console.error('Сгенерированный клиент не совпадает с контрактом — выполните npm run generate (make generate-frontend):')
      for (const d of diff) console.error(`  ${d}`)
      process.exit(1)
    }
    console.log('сгенерированный клиент совпадает с контрактом')
  } finally {
    rmSync(tmp, { recursive: true, force: true })
  }
}
