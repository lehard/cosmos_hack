/** Кнопки окна версии по жизненному циклу (FR-22, FR-23, Д-70). */
import { describe, expect, it } from 'vitest'
import { actionsFor, nextLabel } from '../model/lifecycle'

const all = () => true
const idle = { dirty: false, routeClosed: false, editing: false }

describe('кнопки окна версии', () => {
  it('черновик: «Править схему» (на весь экран, UI-34) и «Отправить» — только сохранённое', () => {
    expect(actionsFor('draft', all, idle)).toEqual([
      { action: 'edit', enabled: true },
      { action: 'submit', enabled: true },
    ])
    expect(actionsFor('draft', all, { ...idle, dirty: true }).find((a) => a.action === 'submit')?.enabled).toBe(false)
  })

  it('на утверждении: ввести в действие — только после закрытия маршрута кворума', () => {
    expect(actionsFor('on_approval', all, idle)).toEqual([{ action: 'activate', enabled: false }])
    expect(actionsFor('on_approval', all, { ...idle, routeClosed: true })).toEqual([{ action: 'activate', enabled: true }])
  })

  it('права решают, какие кнопки видны', () => {
    const technologist = (a: string) => a === 'process.version.draft' || a === 'process.version.submit'
    expect(actionsFor('active', technologist, idle).map((a) => a.action)).toEqual(['newDraft'])
    expect(actionsFor('on_approval', technologist, { ...idle, routeClosed: true })).toEqual([])
  })

  it('правка из действующей версии — «Сохранить черновик»', () => {
    expect(actionsFor('active', all, { ...idle, editing: true, dirty: true })).toEqual([{ action: 'saveDraft', enabled: true }])
  })

  it('метка новой версии', () => {
    expect(nextLabel(['v1', 'v3', 'черновик'])).toBe('v4')
    expect(nextLabel([])).toBe('v1')
  })
})
