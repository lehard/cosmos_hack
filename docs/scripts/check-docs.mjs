#!/usr/bin/env node
// Проверка документации «Главного» (эпик 47; FR-117, NFR-DOC-1, NFR-DOC-3; критерий Т5).
//
//   node docs/scripts/check-docs.mjs            — проверить (код выхода 1 при нарушениях)
//   node docs/scripts/check-docs.mjs --selftest — плюс убедиться, что каждая проверка краснеет
//
// Что проверяется:
//   1. Ссылки и якоря. В README.md и docs/**/*.md (кроме docs/licenses) каждая
//      относительная ссылка [текст](путь#якорь) ведёт на существующий файл или
//      каталог, а якорь — на заголовок целевого markdown (правило якорей GitHub).
//      Ссылки внутри блоков кода и `кода` не проверяются.
//   2. Руководства ролей. docs/guides/‹раздел›.md и встроенная справка
//      frontend/src/shared/help/content/‹раздел›.md — один и тот же текст
//      (побайтно), разделы совпадают в обе стороны. Справка разбирается
//      упрощённым markdown (shared/help: заголовки, абзацы, списки «- »), поэтому
//      в ней нет таблиц, блоков кода, `кода`, **выделения**, ссылок и нумерованных списков.
//      У каждого стола normative/desks/‹роль›.yaml есть help_key, и раздел справки
//      с таким именем существует (FR-117: из интерфейса открывается справка своей роли;
//      роль-наследник получает help_key вместе со столом базовой роли).
//   3. Путеводитель и README. Пути репозитория в `коде` (backend/…, frontend/…,
//      contracts/…, normative/…, scenarios/…, deploy/…, docs/…, extension/…,
//      third_party/…, Makefile, compose.yaml) существуют; цели `make ‹цель›`
//      есть в Makefile. Пути с заполнителями (‹…›, *, …) не проверяются.
//   4. NFR-DOC-3. У каждого пакета Go в backend/ (кроме vendor и testdata) есть
//      заголовочный комментарий — блок `//` прямо перед `package` хотя бы в одном
//      файле пакета, не считая строки «Code generated».
//
// Зависимостей нет: только стандартная библиотека Node ≥ 18.
import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs'
import { dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const REPO = resolve(dirname(fileURLToPath(import.meta.url)), '../..')
const GUIDES = 'docs/guides'
const HELP = 'frontend/src/shared/help/content'

/** Все файлы каталога рекурсивно (относительные пути от REPO). */
function walk(dir, skip = () => false) {
  const out = []
  const abs = join(REPO, dir)
  if (!existsSync(abs)) return out
  for (const name of readdirSync(abs).sort()) {
    const rel = join(dir, name)
    if (skip(rel, name)) continue
    if (statSync(join(REPO, rel)).isDirectory()) out.push(...walk(rel, skip))
    else out.push(rel)
  }
  return out
}

/** Текст без блоков кода ``` (строки заменены пустыми — номера строк сохраняются). */
function stripFences(text) {
  let inFence = false
  return text
    .split('\n')
    .map((line) => {
      if (/^\s*(```|~~~)/.test(line)) {
        inFence = !inFence
        return ''
      }
      return inFence ? '' : line
    })
    .join('\n')
}

/** Якорь заголовка по правилу GitHub: нижний регистр, без знаков, пробел → «-». */
export function slug(heading) {
  const text = heading
    .replace(/!\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/<[^>]+>/g, '')
    .replace(/[`*_~]/g, (m) => (m === '_' ? '_' : ''))
    .trim()
    .toLowerCase()
  return text.replace(/[^\p{L}\p{N}\p{M} _-]/gu, '').replace(/ /g, '-')
}

/** Якоря markdown-файла (с суффиксами -1, -2 у повторов). */
export function anchorsOf(text) {
  const seen = new Map()
  const out = new Set()
  for (const line of stripFences(text).split('\n')) {
    const m = /^#{1,6}\s+(.*?)\s*#*\s*$/.exec(line)
    if (!m) continue
    const base = slug(m[1])
    const n = seen.get(base) ?? 0
    seen.set(base, n + 1)
    out.add(n === 0 ? base : `${base}-${n}`)
  }
  for (const m of text.matchAll(/<a\s+(?:id|name)="([^"]+)"/g)) out.add(m[1])
  return out
}

/** Ссылки строки вне `кода`: [текст](цель). */
function linksOf(line) {
  const clean = line.replace(/`[^`]*`/g, '')
  const out = []
  for (const m of clean.matchAll(/!?\[(?:[^\]\\]|\\.)*\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g)) out.push(m[1])
  return out
}

/** 1. Ссылки и якоря. */
export function checkLinks(files, read = (f) => readFileSync(join(REPO, f), 'utf8'), exists = (p) => existsSync(join(REPO, p))) {
  const problems = []
  const anchorCache = new Map()
  const anchorsFor = (f) => {
    if (!anchorCache.has(f)) anchorCache.set(f, anchorsOf(read(f)))
    return anchorCache.get(f)
  }
  for (const file of files) {
    const lines = stripFences(read(file)).split('\n')
    lines.forEach((line, i) => {
      for (const target of linksOf(line)) {
        if (/^[a-z][a-z0-9+.-]*:/i.test(target)) continue // http:, https:, mailto:
        const [rawPath, anchor] = target.split('#')
        const path = decodeURI(rawPath)
        const dest = path ? relative(REPO, resolve(REPO, dirname(file), path)) : file
        const where = `${file}:${i + 1}`
        if (path && !exists(dest)) {
          problems.push(`${where}: ссылка на несуществующий путь «${target}»`)
          continue
        }
        if (anchor !== undefined && anchor !== '') {
          if (!dest.endsWith('.md')) continue
          if (!anchorsFor(dest).has(decodeURIComponent(anchor))) problems.push(`${where}: в ${dest} нет якоря «#${anchor}»`)
        }
      }
    })
  }
  return problems
}

/** 2. Руководства ролей = встроенная справка; формат справки; help_key столов. */
export function checkGuides(guides, help, desks = new Map()) {
  const problems = []
  for (const [desk, text] of desks) {
    const key = /^help_key:\s*([a-z][a-z0-9_]*)/m.exec(text)?.[1]
    if (!key) problems.push(`normative/desks/${desk}: нет help_key — раздел справки роли не выбран`)
    else if (!help.has(`${key}.md`)) problems.push(`normative/desks/${desk}: help_key «${key}» — нет раздела ${HELP}/${key}.md`)
  }
  const names = (m) => new Set([...m.keys()])
  for (const name of names(guides)) {
    if (!help.has(name)) problems.push(`${GUIDES}/${name}: нет раздела встроенной справки ${HELP}/${name}`)
    else if (help.get(name) !== guides.get(name)) problems.push(`${GUIDES}/${name}: текст расходится со встроенной справкой ${HELP}/${name}`)
  }
  for (const name of names(help)) {
    if (!guides.has(name)) problems.push(`${HELP}/${name}: нет руководства ${GUIDES}/${name}`)
    const lines = help.get(name).split('\n')
    lines.forEach((line, i) => {
      const where = `${HELP}/${name}:${i + 1}`
      const t = line.trim()
      if (/^(```|~~~)/.test(t)) problems.push(`${where}: блок кода — справка его не показывает`)
      else if (/^\|/.test(t)) problems.push(`${where}: таблица — справка её не показывает`)
      else if (/^\d+[.)]\s/.test(t)) problems.push(`${where}: нумерованный список — справка склеит его в абзац, нужен «- »`)
      else if (/^[*+]\s/.test(t)) problems.push(`${where}: пункт списка не через «- »`)
      else if (/`/.test(t)) problems.push(`${where}: \`код\` — справка покажет обратные кавычки`)
      else if (/\*\*|__/.test(t)) problems.push(`${where}: **выделение** — справка покажет звёздочки`)
      else if (/\[[^\]]*\]\([^)]*\)/.test(t)) problems.push(`${where}: ссылка — справка покажет её разметку`)
    })
    if (!/^# \S/.test(help.get(name))) problems.push(`${HELP}/${name}: первая строка — заголовок «# …»`)
  }
  return problems
}

const PATH_RE = /^(?:backend|frontend|contracts|normative|scenarios|deploy|docs|extension|third_party)\/\S*$|^(?:Makefile|compose\.yaml|README\.md)$/

/** 3. Пути в `коде` и цели make. */
export function checkRefs(files, read, exists, targets) {
  const problems = []
  for (const file of files) {
    const text = read(file)
    const lines = text.split('\n')
    let inFence = false
    lines.forEach((line, i) => {
      const where = `${file}:${i + 1}`
      if (/^\s*(```|~~~)/.test(line)) {
        inFence = !inFence
        return
      }
      const spans = inFence ? [line] : [...line.matchAll(/`([^`]+)`/g)].map((m) => m[1])
      for (const span of spans) {
        for (const m of span.matchAll(/(?:^|[\s;(&|])make\s+([a-z][a-z0-9-]*)/g)) {
          if (!targets.has(m[1])) problems.push(`${where}: цели «make ${m[1]}» нет в Makefile`)
        }
        if (inFence) continue
        const token = span.trim().replace(/[.,;:]$/, '')
        if (!PATH_RE.test(token) || /[‹›*…{}<>]/.test(token)) continue
        const path = token.replace(/\/$/, '')
        if (!exists(path)) problems.push(`${where}: пути «${token}» нет в репозитории`)
      }
    })
  }
  return problems
}

/** Цели Makefile. */
function makeTargets(text) {
  const out = new Set()
  for (const m of text.matchAll(/^([a-zA-Z0-9_-]+)\s*:(?!=)/gm)) out.add(m[1])
  return out
}

/** 4. NFR-DOC-3: заголовок у каждого пакета Go. */
export function checkGoHeaders(dirs) {
  const problems = []
  for (const [dir, files] of dirs) {
    const has = files.some((src) => {
      const lines = src.split('\n')
      const i = lines.findIndex((l) => l.startsWith('package '))
      if (i < 0) return false
      const block = []
      for (let j = i - 1; j >= 0 && lines[j].startsWith('//') && !lines[j].startsWith('//go:'); j--) block.push(lines[j])
      return block.length > 0 && !block.some((l) => /Code generated|DO NOT EDIT/.test(l))
    })
    if (!has) problems.push(`${dir}: у пакета Go нет заголовочного комментария (NFR-DOC-3)`)
  }
  return problems
}

/** Пакеты Go: каталог → тексты файлов без _test.go. */
function goPackages() {
  const files = walk('backend', (rel, name) => name === 'vendor' || name === 'testdata' || name === 'node_modules' || name.startsWith('.'))
  const dirs = new Map()
  for (const f of files) {
    if (!f.endsWith('.go') || f.endsWith('_test.go')) continue
    const d = dirname(f)
    if (!dirs.has(d)) dirs.set(d, [])
    dirs.get(d).push(readFileSync(join(REPO, f), 'utf8'))
  }
  return dirs
}

function markdownFiles() {
  return ['README.md', ...walk('docs', (rel) => rel.startsWith('docs/licenses'))].filter((f) => f.endsWith('.md'))
}

/** Столы ролей: имя файла → текст YAML. */
function deskFiles() {
  const out = new Map()
  for (const f of walk('normative/desks')) if (f.endsWith('.yaml')) out.set(f.slice('normative/desks/'.length), readFileSync(join(REPO, f), 'utf8'))
  return out
}

function readDir(dir) {
  const out = new Map()
  for (const f of walk(dir)) if (f.endsWith('.md') && !f.endsWith('/README.md')) out.set(f.slice(dir.length + 1), readFileSync(join(REPO, f), 'utf8'))
  return out
}

function run() {
  const read = (f) => readFileSync(join(REPO, f), 'utf8')
  const exists = (p) => existsSync(join(REPO, p))
  const md = markdownFiles()
  const targets = makeTargets(read('Makefile'))
  const refFiles = ['README.md', 'docs/reviewer-guide.md', ...walk(GUIDES).filter((f) => f.endsWith('.md'))].filter(exists)
  const sections = [
    ['ссылки и якоря', checkLinks(md, read, exists)],
    ['руководства = справка', checkGuides(readDir(GUIDES), readDir(HELP), deskFiles())],
    ['пути и цели make', checkRefs(refFiles, read, exists, targets)],
    ['заголовки пакетов Go', checkGoHeaders(goPackages())],
  ]
  let failed = 0
  for (const [title, problems] of sections) {
    if (problems.length) {
      failed += problems.length
      console.log(`✗ ${title}: ${problems.length}`)
      for (const p of problems) console.log(`  ${p}`)
    } else console.log(`✓ ${title}`)
  }
  console.log(`markdown-файлов: ${md.length}, пакетов Go: ${goPackages().size}`)
  return failed
}

/** Самопроверка: каждая проверка ловит заведомое нарушение. */
function selftest() {
  const files = { 'docs/a.md': '# Заголовок раз\n## 2. Второй — раздел\n[b](b.md#нет) [c](c.md) [ok](#2-второй--раздел)\n`[x](nope.md)`\n' }
  const read = (f) => files[f] ?? '# B\n'
  const exists = (p) => p in files || p === 'docs/b.md'
  const links = checkLinks(['docs/a.md'], read, exists)
  const guides = checkGuides(
    new Map([['a.md', '# A\n'], ['b.md', '# B\n']]),
    new Map([['a.md', '# A\nтекст `код`\n1. пункт\n'], ['c.md', '# C\n']]),
    new Map([['x.yaml', 'role: x\n'], ['y.yaml', 'help_key: nope\n'], ['z.yaml', 'help_key: a\n']]),
  )
  const refs = checkRefs(['README.md'], () => '`backend/нет/такого` `make nosuch-target` `backend/‹модуль›`\n', () => false, new Set(['up']))
  const heads = checkGoHeaders(new Map([['backend/x', ['package x\n']], ['backend/y', ['// Code generated. DO NOT EDIT.\npackage y\n']], ['backend/z', ['// Пакет z.\npackage z\n']]]))
  const expect = [
    ['ссылки', links.length === 2],
    ['руководства', guides.length === 7],
    ['пути и make', refs.length === 2],
    ['заголовки Go', heads.length === 2],
    ['якорь GitHub', slug('2. Второй — раздел') === '2-второй--раздел' && slug('Сборка в `cmd/ant` (для интегратора)') === 'сборка-в-cmdant-для-интегратора'],
  ]
  const bad = expect.filter(([, ok]) => !ok)
  for (const [name] of bad) console.log(`✗ самопроверка: проверка «${name}» не ловит нарушение`)
  if (!bad.length) console.log('✓ самопроверка: все проверки ловят нарушения')
  return bad.length
}

let failed = 0
if (process.argv.includes('--selftest')) failed += selftest()
failed += run()
process.exit(failed ? 1 : 0)
