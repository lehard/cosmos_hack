// Открытая запись в адресе (Д-70): `?open=‹тип›:‹id›`, «назад» закрывает окно,
// без окна в оболочке ссылки ведут на страницы.
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { RECORD_DRAWER, formatRecord, parseRecord, useRecordLink, withoutRecord } from '../record'

describe('ссылка на запись в адресе', () => {
  it('тип — до первого двоеточия, id может содержать двоеточие', () => {
    expect(parseRecord('item:ENT:FL-0042')).toEqual({ entity: 'item', id: 'ENT:FL-0042' })
    expect(parseRecord(['nonconformity:NC-0142', 'item:x'])).toEqual({ entity: 'nonconformity', id: 'NC-0142' })
    expect(formatRecord({ entity: 'item', id: 'ENT:FL-0042' })).toBe('item:ENT:FL-0042')
    for (const bad of [undefined, null, '', 'item', ':NC-1', 'item:', 42]) expect(parseRecord(bad)).toBeNull()
    expect(withoutRecord({ open: 'item:x', run: 'R1' })).toEqual({ run: 'R1' })
  })

  async function setup(kinds: string[] | null, start = '/desk?tab=queue') {
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:p*', component: { template: '<div />' } }] })
    await router.push(start)
    let link!: ReturnType<typeof useRecordLink>
    const Probe = defineComponent({
      setup() {
        link = useRecordLink()
        return () => h('div')
      },
    })
    mount(Probe, { global: { plugins: [router], provide: kinds ? { [RECORD_DRAWER as symbol]: { kinds: new Set(kinds) } } : {} } })
    return { router, link }
  }

  it('открыть — новый шаг истории с сохранением запроса; закрыть — шаг назад', async () => {
    const { router, link } = await setup(['item'])
    expect(link.open({ entity: 'item', id: 'ENT:FL-0042' })).toBe(true)
    await router.isReady()
    await new Promise((r) => setTimeout(r))
    expect(router.currentRoute.value.query).toEqual({ tab: 'queue', open: 'item:ENT:FL-0042' })
    expect(link.current.value).toEqual({ entity: 'item', id: 'ENT:FL-0042' })
    // Тип, которого нет в окне, — false: вызывающий откроет страницу.
    expect(link.canOpen({ entity: 'lot', id: 'L-1' })).toBe(false)
    expect(link.open({ entity: 'lot', id: 'L-1' })).toBe(false)
  })

  it('окно открыто по прямой ссылке — закрытие убирает параметр, а не уходит со страницы', async () => {
    const { router, link } = await setup(['item'], '/desk?open=item:ENT:FL-0042')
    expect(link.current.value).toEqual({ entity: 'item', id: 'ENT:FL-0042' })
    link.close()
    await new Promise((r) => setTimeout(r))
    expect(router.currentRoute.value.fullPath).toBe('/desk')
  })

  it('без окна в оболочке (тест виджета, вход) открыть нельзя', async () => {
    const { link } = await setup(null)
    expect(link.canOpen({ entity: 'item', id: 'x' })).toBe(false)
    expect(link.open({ entity: 'item', id: 'x' })).toBe(false)
  })
})
