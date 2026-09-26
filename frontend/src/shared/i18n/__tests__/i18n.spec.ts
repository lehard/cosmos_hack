// Тексты интерфейса (NFR-UI-3, Д-10, Д-11, Д-12).
import { describe, expect, it } from 'vitest'
import { codeToKey, i18n, ruPlural } from '..'

const { t } = i18n.global

describe('тексты интерфейса', () => {
  it('русское множественное число', () => {
    expect([1, 2, 5, 11, 21, 22, 25, 111].map((n) => ruPlural(n, 3))).toEqual([0, 1, 2, 2, 0, 1, 2, 2])
  })

  it('код контракта → ключ', () => {
    expect(codeToKey('ingest.unknown_schema_version')).toBe('ingest.unknownSchemaVersion')
  })

  it('Д-11: показатель «с первого раза» — формулировка кейса', () => {
    expect(t('analytics.metrics.firstPassYield.title')).toBe('Прохождение контроля с первого раза')
  })

  it('Д-10: названия модулей', () => {
    expect(t('common.modules.visionQc')).toBe('Визуальный контроль')
    expect(t('common.modules.machineLogs')).toBe('Журналы оборудования')
    expect(t('common.modules.criticalActions')).toBe('Критические действия')
    expect(t('common.modules.operatorVision')).toBe('Контроль действий оператора')
  })

  it('ссылки @: и тексты оболочки работают вместе', () => {
    expect(t('shell.header.logout')).toBe('Выйти')
  })
})
