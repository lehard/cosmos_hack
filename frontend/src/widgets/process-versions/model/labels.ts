/**
 * Подписи свойств расширения BPMN (`ant:properties`, contracts/bpmn-ext) и их
 * значений для читаемого представления версии процесса. Код показывается,
 * только когда перевода нет — не путь ключа текста и не пустое место.
 */
import { useI18n } from 'vue-i18n'
import { eventCatalog } from '@/shared/contracts/catalog'
import { codeToKey } from '@/shared/i18n'
import type { ProcessPropertyValue } from '@/entities/process-version'

/** Свойство → словарь его значений (префикс ключа текста). */
const valueDictionaries: Record<string, string> = {
  stepKind: 'process.values.stepKind',
  reworkLimitScope: 'process.values.reworkLimitScope',
  erpAction: 'process.values.erpAction',
  timerScope: 'process.values.timerScope',
  outcome: 'process.values.outcome',
  paperAttester: 'roles',
}

/** Название типа события из каталога (AD-40) или null. */
const eventTitle = (code: string): string | null =>
  (eventCatalog as Record<string, { title?: string } | undefined>)[code]?.title ?? null

/** Подписи свойств и значений для текущего языка. */
export function usePropertyLabels() {
  const { t, te, n } = useI18n()

  /** Текст по ключу, если он есть и это строка (не раздел словаря). */
  const known = (key: string): string | null => {
    if (!te(key)) return null
    const s = t(key)
    return s && s !== key ? s : null
  }

  /** Подпись свойства; нет перевода — имя атрибута BPMN как есть. */
  const propLabel = (prop: string): string => known(`process.properties.${prop}`) ?? prop

  /** Значение свойства словами; нет перевода — код. */
  function propValue(prop: string, v: ProcessPropertyValue | undefined): string {
    if (v === true) return t('widgets.analysis.process.valueYes')
    if (v === false) return t('widgets.analysis.process.valueNo')
    if (v === null || v === undefined || v === '') return t('widgets.analysis.process.valueEmpty')
    if (typeof v === 'number') return n(v)
    if (prop === 'triggerEventType') return eventTitle(v) ?? v
    const dict = valueDictionaries[prop]
    return (dict && known(`${dict}.${codeToKey(v)}`)) ?? v
  }

  return { propLabel, propValue }
}
