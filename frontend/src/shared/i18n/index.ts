/**
 * Тексты интерфейса ant (слой shared, FSD) — vue-i18n, единственный источник
 * текстов интерфейса (NFR-UI-3, решение дирижёра Д-12).
 *
 * - `ru.json` — копия ui-texts/ru.json процессной сессии; обновления от неё
 *   вливает дирижёр. В копии применено решение Д-11: показатель называется
 *   «Прохождение контроля с первого раза» (кейс §2.4). Названия модулей —
 *   по Д-10, в файле они уже такие (`common.modules.*`).
 * - `ru.shell.json` — тексты оболочки, которых в ru.json нет (разделы `shell`,
 *   `widgets`). Ключи двух файлов не пересекаются — это проверяет
 *   scripts/check-shell.mjs в make check.
 *
 * Код подключения — по README текстов процессной сессии.
 */
import { createI18n } from 'vue-i18n'
import ru from './ru.json'
import shell from './ru.shell.json'

/** Схема сообщений: ru.json + тексты оболочки. */
export type MessageSchema = typeof ru & typeof shell

/**
 * Русское множественное число: 1 изделие / 2 изделия / 5 изделий.
 * @param choice — число
 * @param choicesLength — сколько форм в сообщении
 */
export function ruPlural(choice: number, choicesLength: number): number {
  const n = Math.abs(choice) % 100
  const n1 = n % 10
  const idx = n > 10 && n < 20 ? 2 : n1 === 1 ? 0 : n1 >= 2 && n1 <= 4 ? 1 : 2
  return Math.min(idx, choicesLength - 1)
}

/**
 * Код контракта → ключ текста: каждый сегмент snake_case → camelCase.
 * `ingest.unknown_schema_version` → `ingest.unknownSchemaVersion`.
 */
export const codeToKey = (code: string): string =>
  code
    .split('.')
    .map((s) => s.replace(/_([a-z0-9])/g, (_, c: string) => c.toUpperCase()))
    .join('.')

/** Часовой пояс завода — допущение процессной сессии (README текстов, вопрос 11). */
export const PLANT_TIME_ZONE = 'Europe/Moscow'

export const i18n = createI18n({
  legacy: false,
  locale: 'ru',
  fallbackLocale: 'ru',
  messages: { ru: { ...ru, ...shell } },
  pluralRules: { ru: ruPlural },
  numberFormats: {
    ru: {
      decimal2: { minimumFractionDigits: 2, maximumFractionDigits: 2 }, // уверенность, качество наблюдения
      integer: { maximumFractionDigits: 0 },
      percent: { style: 'percent', maximumFractionDigits: 1 },
    },
  },
  datetimeFormats: {
    ru: {
      date: { day: '2-digit', month: '2-digit', year: 'numeric', timeZone: PLANT_TIME_ZONE },
      dateTime: { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit', timeZone: PLANT_TIME_ZONE },
      // с секундами — журнал: порядок записей внутри минуты (AD-37)
      dateTimeSec: { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit', timeZone: PLANT_TIME_ZONE },
      time: { hour: '2-digit', minute: '2-digit', timeZone: PLANT_TIME_ZONE },
    },
  },
  missingWarn: import.meta.env.DEV,
  fallbackWarn: false,
})
