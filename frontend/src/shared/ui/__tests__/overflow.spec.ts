// Ни один текст в общих элементах shared/ui не выходит за границы (UI-2).
// Браузера на машине сборки нет, поэтому проверка статическая, по шаблонам:
// - каждая вставка текста `{{ … }}` и каждый слот `<slot>` лежат в HTML-элементе
//   с одним из служебных классов переполнения (theme/base.css): многоточие,
//   перенос, ограничение строк или сжимаемый контейнер;
// - текст не передаётся в компоненты Naive UI свойствами (title, label, …) —
//   там его переполнением не управляем; текст идёт в слот, в свой элемент.
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { parse } from 'vue/compiler-sfc'
import { describe, expect, it } from 'vitest'

const UI = resolve(__dirname, '..')
const OVERFLOW = ['ant-ellipsis', 'ant-wrap', 'ant-clamp-2', 'ant-box']
const TEXT_PROPS = ['title', 'label', 'description', 'tab', 'content', 'header']

// Типы узлов шаблона Vue (@vue/compiler-core).
const ELEMENT = 1
const INTERPOLATION = 5
const ATTRIBUTE = 6
const DIRECTIVE = 7
const TAG_ELEMENT = 0
const TAG_COMPONENT = 1
const TAG_SLOT = 2
const TAG_TEMPLATE = 3

interface Node {
  type: number
  tag?: string
  tagType?: number
  props?: { type: number; name: string; value?: { content: string }; arg?: { content: string }; exp?: { content: string } }[]
  children?: Node[]
  loc: { start: { line: number }; source: string }
}

/** Классы элемента: статические и из выражения :class. */
function classesOf(el: Node): string {
  return (el.props ?? [])
    .map((p) => (p.type === ATTRIBUTE && p.name === 'class' ? (p.value?.content ?? '') : p.type === DIRECTIVE && p.name === 'bind' && p.arg?.content === 'class' ? (p.exp?.content ?? '') : ''))
    .join(' ')
}

const hasOverflowClass = (el: Node) => OVERFLOW.some((c) => classesOf(el).includes(c))

/** Нарушения в шаблоне одного файла. */
export function violations(template: string, file = 'шаблон'): string[] {
  const { descriptor, errors } = parse(`<template>${template}</template>`, { filename: file })
  if (errors.length) throw new Error(`${file}: ${String(errors[0])}`)
  const out: string[] = []
  let checked = 0

  /** Ближайший предок, отвечающий за переполнение (template и fallback слота прозрачны). */
  const owner = (stack: Node[]) => [...stack].reverse().find((n) => n.tagType !== TAG_TEMPLATE && n.tagType !== TAG_SLOT)

  function walk(node: Node, stack: Node[]): void {
    const isSlot = node.type === ELEMENT && node.tagType === TAG_SLOT
    if (node.type === INTERPOLATION || isSlot) {
      checked++
      const o = owner(stack)
      const what = isSlot ? `<slot>` : node.loc.source
      if (!o || o.tagType !== TAG_ELEMENT) out.push(`${file}:${node.loc.start.line}: ${what} — внутри компонента <${o?.tag ?? '?'}>, а не в своём элементе с классом переполнения`)
      else if (!hasOverflowClass(o)) out.push(`${file}:${node.loc.start.line}: ${what} — у <${o.tag}> нет класса переполнения (${OVERFLOW.join(', ')})`)
    }
    if (node.type === ELEMENT && node.tagType === TAG_COMPONENT && /^N[A-Z]/.test(node.tag ?? '')) {
      for (const p of node.props ?? []) {
        const name = p.type === ATTRIBUTE ? p.name : p.type === DIRECTIVE && p.name === 'bind' ? p.arg?.content : undefined
        if (name && TEXT_PROPS.includes(name)) out.push(`${file}:${node.loc.start.line}: <${node.tag} ${name}> — текст свойством компонента Naive UI; перенесите в слот со своим элементом`)
      }
    }
    for (const c of node.children ?? []) walk(c, node.type === ELEMENT ? [...stack, node] : stack)
  }

  const root = descriptor.template!.ast as unknown as Node
  for (const c of root.children ?? []) walk(c, [])
  if (!checked && /\{\{|<slot/.test(template)) out.push(`${file}: проверка ничего не нашла — разбор шаблона сломан`)
  return out
}

const files = readdirSync(UI).filter((f) => f.endsWith('.vue'))

describe('shared/ui: текст не выходит за границы', () => {
  it('самопроверка: голый текст и текст свойством Naive UI ловятся', () => {
    expect(violations('<div>{{ label }}</div>')).toHaveLength(1)
    expect(violations('<NButton>{{ label }}</NButton>')).toHaveLength(1)
    expect(violations('<section><slot /></section>')).toHaveLength(1)
    expect(violations('<NCard :title="t(x)"><div class="ant-box"><slot /></div></NCard>')).toHaveLength(1)
    expect(violations('<span class="ant-ellipsis">{{ label }}</span>')).toEqual([])
    expect(violations('<span :class="wrap ? \'ant-wrap\' : \'ant-ellipsis\'"><template v-if="x">{{ a }}</template></span>')).toEqual([])
  })

  it('элементы найдены', () => {
    expect(files).toEqual(expect.arrayContaining(['ActionButton.vue', 'DataTable.vue', 'FormField.vue', 'StatusTag.vue', 'WidgetFrame.vue']))
  })

  for (const f of files) {
    it(f, () => {
      const { descriptor } = parse(readFileSync(join(UI, f), 'utf8'), { filename: f })
      const tpl = descriptor.template?.content ?? ''
      expect(violations(tpl, f)).toEqual([])
    })
  }
})
