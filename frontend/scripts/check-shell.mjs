// Проверки оболочки интерфейса для make check (эпик 03).
//
//   node scripts/check-shell.mjs             — проверить
//   node scripts/check-shell.mjs --selftest  — плюс убедиться, что проверка краснеет
//
// 1. Реестр виджетов (src/widgets/registry.ts, читается синтаксическим деревом
//    TypeScript): у каждого id есть папка src/widgets/‹id›/index.ts и ленивый
//    импорт './‹id›'; других папок виджетов нет; ключ заголовка есть в текстах.
// 2. Столы ролей normative/desks/*.yaml (AD-21, NFR-EXT-1): схема desk.schema.json,
//    имя файла = роль, id вкладок и слотов уникальны, каждый виджет есть в реестре
//    («неизвестный виджет ловит make check»), ключи названий есть в текстах.
//    Роли столов сверяются со стартовой политикой normative/policy/policy.v1.yaml.
// 3. Тексты (NFR-UI-3, Д-12): ru.json и ru.shell.json — без повторяющихся ключей,
//    ключи двух файлов не пересекаются.
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs'
import { basename, dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import Ajv from 'ajv'
import ts from 'typescript'
import { parse } from 'yaml'

const FRONTEND = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const REPO = resolve(FRONTEND, '..')
const WIDGETS = join(FRONTEND, 'src/widgets')
const DESKS = join(REPO, 'normative/desks')
const I18N = join(FRONTEND, 'src/shared/i18n')

/** Реестр виджетов из registry.ts: id → { titleKey, epic, importPath }. */
export function readRegistry(file = join(WIDGETS, 'registry.ts')) {
  const src = ts.createSourceFile(file, readFileSync(file, 'utf8'), ts.ScriptTarget.Latest, true)
  let literal = null
  src.forEachChild((node) => {
    if (!ts.isVariableStatement(node)) return
    for (const decl of node.declarationList.declarations) {
      if (decl.name.getText(src) !== 'widgetRegistry' || !decl.initializer) continue
      let init = decl.initializer
      while (ts.isSatisfiesExpression(init) || ts.isAsExpression(init) || ts.isParenthesizedExpression(init)) init = init.expression
      if (ts.isObjectLiteralExpression(init)) literal = init
    }
  })
  if (!literal) throw new Error(`${relative(REPO, file)}: не найден объект widgetRegistry`)
  const out = new Map()
  for (const prop of literal.properties) {
    if (!ts.isPropertyAssignment(prop) || !ts.isObjectLiteralExpression(prop.initializer)) {
      throw new Error(`${relative(REPO, file)}: строка реестра не в формате «'id': { … }»: ${prop.getText(src)}`)
    }
    const id = ts.isStringLiteral(prop.name) || ts.isIdentifier(prop.name) ? prop.name.text : null
    const entry = { titleKey: null, epic: null, importPath: null }
    for (const f of prop.initializer.properties) {
      if (!ts.isPropertyAssignment(f)) continue
      const name = f.name.getText(src)
      if (name === 'titleKey' && ts.isStringLiteral(f.initializer)) entry.titleKey = f.initializer.text
      if (name === 'epic' && ts.isNumericLiteral(f.initializer)) entry.epic = Number(f.initializer.text)
      if (name === 'load') {
        f.initializer.forEachChild(function find(n) {
          if (ts.isCallExpression(n) && n.expression.kind === ts.SyntaxKind.ImportKeyword && n.arguments[0] && ts.isStringLiteral(n.arguments[0])) {
            entry.importPath = n.arguments[0].text
          }
          n.forEachChild(find)
        })
      }
    }
    out.set(id, entry)
  }
  return out
}

/** Все пути листьев объекта текстов: 'a.b.c'. */
function leafPaths(obj, prefix = '') {
  return Object.entries(obj).flatMap(([k, v]) => (v && typeof v === 'object' ? leafPaths(v, `${prefix}${k}.`) : [`${prefix}${k}`]))
}

/** Разбор JSON с запретом повторяющихся ключей (YAML — надмножество JSON). */
function parseStrictJson(file) {
  return parse(readFileSync(file, 'utf8'), { uniqueKeys: true })
}

/** Есть ли ключ текста (путь в объекте сообщений). */
const hasKey = (messages, key) => key.split('.').reduce((o, k) => (o && typeof o === 'object' ? o[k] : undefined), messages) !== undefined

/**
 * Проверить стол.
 * @returns список ошибок
 */
export function checkDesk(name, desk, { validate, registry, messages }) {
  const errors = []
  if (!validate(desk)) {
    for (const e of validate.errors) errors.push(`${name}: схема: ${e.instancePath || '/'} ${e.message}`)
    return errors
  }
  const role = basename(name, '.yaml')
  if (desk.role !== role) errors.push(`${name}: role «${desk.role}» не совпадает с именем файла «${role}»`)
  const keys = [desk.title_key]
  const tabIds = new Set()
  for (const tab of desk.tabs) {
    if (tabIds.has(tab.id)) errors.push(`${name}: вкладка «${tab.id}» повторяется`)
    tabIds.add(tab.id)
    keys.push(tab.title_key)
    const slotIds = new Set()
    for (const slot of tab.slots) {
      if (slotIds.has(slot.id)) errors.push(`${name}: вкладка «${tab.id}»: слот «${slot.id}» повторяется`)
      slotIds.add(slot.id)
      if (!registry.has(slot.widget)) {
        errors.push(`${name}: вкладка «${tab.id}», слот «${slot.id}»: неизвестный виджет «${slot.widget}» — его нет в frontend/src/widgets/registry.ts`)
      }
    }
  }
  for (const k of keys) if (!hasKey(messages, k)) errors.push(`${name}: нет текста интерфейса «${k}»`)
  return errors
}

function main() {
  const errors = []
  const ok = (msg) => console.log(`  ок: ${msg}`)

  // 3. Тексты.
  const ru = parseStrictJson(join(I18N, 'ru.json'))
  const shell = parseStrictJson(join(I18N, 'ru.shell.json'))
  const ruPaths = new Set(leafPaths(ru))
  const overlap = leafPaths(shell).filter((p) => ruPaths.has(p) || [...ruPaths].some((r) => r.startsWith(`${p}.`) || p.startsWith(`${r}.`)))
  for (const p of overlap) errors.push(`ru.shell.json: ключ «${p}» пересекается с ru.json — тексты процессной сессии правятся в ru.json`)
  const messages = { ...ru, ...shell }
  if (!overlap.length) ok(`тексты: ${ruPaths.size} + ${leafPaths(shell).length} ключей, повторов и пересечений нет`)

  // 1. Реестр.
  const registry = readRegistry()
  const folders = readdirSync(WIDGETS).filter((f) => statSync(join(WIDGETS, f)).isDirectory())
  for (const [id, e] of registry) {
    if (!/^[a-z][a-z0-9-]*$/.test(id ?? '')) errors.push(`registry.ts: id «${id}» — только латиница в нижнем регистре, цифры и дефис`)
    if (e.importPath !== `./${id}`) errors.push(`registry.ts: «${id}» грузит «${e.importPath}», ожидается './${id}'`)
    if (!existsSync(join(WIDGETS, id, 'index.ts'))) errors.push(`registry.ts: у «${id}» нет src/widgets/${id}/index.ts`)
    if (!e.titleKey || !hasKey(messages, e.titleKey)) errors.push(`registry.ts: у «${id}» нет текста заголовка «${e.titleKey}»`)
    if (!Number.isInteger(e.epic)) errors.push(`registry.ts: у «${id}» не указан эпик-владелец`)
  }
  for (const f of folders) if (!registry.has(f)) errors.push(`src/widgets/${f}: папка виджета не записана в registry.ts`)
  ok(`реестр: ${registry.size} виджетов, папки на месте`)

  // 2. Столы.
  const ajv = new Ajv({ allErrors: true, strict: false })
  const validate = ajv.compile(JSON.parse(readFileSync(join(DESKS, 'desk.schema.json'), 'utf8')))
  const files = readdirSync(DESKS).filter((f) => f.endsWith('.yaml')).sort()
  if (!files.length) errors.push('normative/desks: нет ни одного стола')
  for (const f of files) {
    let desk
    try {
      desk = parse(readFileSync(join(DESKS, f), 'utf8'), { uniqueKeys: true })
    } catch (e) {
      errors.push(`${f}: не разбирается: ${e.message}`)
      continue
    }
    errors.push(...checkDesk(f, desk, { validate, registry, messages }))
  }
  ok(`столы: ${files.map((f) => basename(f, '.yaml')).join(', ')}`)

  // Столы ↔ роли стартовой политики: стол — только у существующей роли; у каждой
  // базовой роли (без inherits) стол есть, наследники берут стол базовой (AD-15).
  const policyFile = join(REPO, 'normative/policy/policy.v1.yaml')
  if (existsSync(policyFile)) {
    const policy = parse(readFileSync(policyFile, 'utf8'))
    const roles = policy.roles ?? []
    const ids = new Set(roles.map((r) => r.id))
    const deskRoles = new Set(files.map((f) => basename(f, '.yaml')))
    // Общие роли (их наследуют другие: employee, staff) и роль субъекта без
    // сеанса (unauthenticated_role: device_source, Д-63) — не роли людей за
    // столом: стол им не нужен.
    const inherited = new Set(roles.flatMap((r) => r.inherits ?? []))
    const noDesk = (id) => inherited.has(id) || id === policy.unauthenticated_role
    for (const r of deskRoles) if (!ids.has(r)) errors.push(`normative/desks/${r}.yaml: роли «${r}» нет в normative/policy/policy.v1.yaml`)
    for (const r of roles) {
      if (!(r.inherits ?? []).length && !noDesk(r.id) && !deskRoles.has(r.id)) errors.push(`normative/desks: у базовой роли «${r.id}» нет стола`)
    }
    ok(`столы ↔ политика: ${ids.size} ролей, у базовых столы есть`)
  }

  // Самопроверка: неизвестный виджет обязан краснеть.
  if (process.argv.includes('--selftest')) {
    const probe = { version: 1, role: 'zz_probe', title_key: 'desks.dashboard', density: 'comfortable', tabs: [{ id: 't', title_key: 'desks.dashboard', layout: 'single', slots: [{ id: 's', area: 'main', widget: 'no-such-widget' }] }] }
    const got = checkDesk('zz_probe.yaml', probe, { validate, registry, messages })
    if (!got.some((e) => e.includes('неизвестный виджет'))) errors.push('самопроверка: стол с неизвестным виджетом не пойман')
    const badArea = structuredClone(probe)
    badArea.tabs[0].slots[0] = { id: 's', area: 'left', widget: [...registry.keys()][0] }
    if (!checkDesk('zz_probe.yaml', badArea, { validate, registry, messages }).length) errors.push('самопроверка: слот вне областей раскладки не пойман')
    else ok('самопроверка: неизвестный виджет и чужая область раскладки краснеют')
  }

  if (errors.length) {
    console.error(`\nОшибки оболочки (${errors.length}):`)
    for (const e of errors) console.error(`  ${e}`)
    process.exit(1)
  }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main()
