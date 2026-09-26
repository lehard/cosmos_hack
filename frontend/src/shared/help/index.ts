/**
 * Встроенная справка ролей (AD-21, FR-78: «из интерфейса открывается раздел
 * справки своей роли»). Руководства — markdown в help/content/‹роль›.md,
 * собираются в бандл и открываются без сети. Каркас — эпик 03; полные
 * руководства ролей пишет эпик 47, заменяя эти файлы.
 */
const guides = import.meta.glob('./content/*.md', { query: '?raw', import: 'default', eager: true }) as Record<string, string>

/** Руководство роли или null, если его ещё нет. */
export function helpGuide(role: string): string | null {
  return guides[`./content/${role}.md`] ?? null
}

/** Роли, для которых есть руководство. */
export const helpRoles = (): string[] =>
  Object.keys(guides)
    .map((p) => p.replace('./content/', '').replace('.md', ''))
    .sort()

/** Блок упрощённого markdown: заголовок, абзац или список. */
export type HelpBlock = { kind: 'h'; level: 2 | 3; text: string } | { kind: 'p'; text: string } | { kind: 'ul'; items: string[] }

/**
 * Разбор упрощённого markdown (`#`, `##`, `- `, абзацы) без HTML: текст выводится
 * узлами Vue, v-html не нужен.
 */
export function parseHelp(md: string): HelpBlock[] {
  const blocks: HelpBlock[] = []
  let para: string[] = []
  let list: string[] | null = null
  const flush = () => {
    if (para.length) blocks.push({ kind: 'p', text: para.join(' ') })
    if (list) blocks.push({ kind: 'ul', items: list })
    para = []
    list = null
  }
  for (const raw of md.split('\n')) {
    const line = raw.trim()
    if (!line) {
      flush()
    } else if (line.startsWith('#')) {
      flush()
      const level = line.startsWith('##') ? 3 : 2
      blocks.push({ kind: 'h', level, text: line.replace(/^#+\s*/, '') })
    } else if (line.startsWith('- ')) {
      if (para.length) flush()
      list = [...(list ?? []), line.slice(2)]
    } else {
      if (list) flush()
      para.push(line)
    }
  }
  flush()
  return blocks
}
