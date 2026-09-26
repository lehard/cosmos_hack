// Какие кнопки доступны в окне интеграции (FR-157, AD-47).
import { describe, expect, it } from 'vitest'
import { integrationActions } from '..'
import { integrations } from './fixtures'

describe('кнопки интеграции', () => {
  const [onec, galaktika, vqc] = integrations().items
  it('стенд без реального адреса: только «Выключить»', () => {
    expect(integrationActions(onec!)).toEqual({ enable: null, disable: true, toggle: null })
  })
  it('неустановленная — без кнопок', () => {
    expect(integrationActions(galaktika!)).toEqual({ enable: null, disable: false, toggle: null })
  })
  it('выключенная — «Включить» в прежний режим', () => {
    expect(integrationActions(vqc!).enable).toBe('enabled')
  })
  it('реальная со стендом — «Переключить на стенд»', () => {
    expect(integrationActions({ ...vqc!, state: 'enabled' }).toggle).toBe('stand')
  })
})
