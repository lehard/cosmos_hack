/**
 * Текст ошибки API для пользователя: ключ из каталога кодов (contracts/errors.yaml,
 * поле ui_key), параметры из problem+json; нет текста по ключу — заголовок ответа.
 */
import { useI18n } from 'vue-i18n'
import { problemMessage } from '@/shared/api/problem'

/** Функция «ошибка клиента → русский текст». */
export function useProblemText(): (err: unknown) => string {
  const { t, te } = useI18n()
  return (err) => {
    const m = problemMessage(err)
    if (te(m.key)) return t(m.key, m.params)
    // Нет текста по ключу — пояснение сервера (у 409 гардов оно и есть ответ человеку), затем заголовок.
    return m.detail || m.fallback || t('errors.generic')
  }
}
