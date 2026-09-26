/**
 * Порт подписи решения (FR-66, FR-69, FR-139; AD-13, AD-14, AD-43). Один порт —
 * два пути: агент токена (уровень 2 — доверенное окно агента и касание токена)
 * и бумага с заверением вторым человеком.
 *
 * Сервер закрытых ключей не хранит и кода подписи не отдаёт (FR-69): подпись
 * делает агент токена через браузерное расширение по Native Messaging
 * (contracts/internal/token-agent, `sign` / `sign_batch`). Пока расширения нет
 * (эпик 38), порт — заглушка: агент «не найден», `sign` отвечает кодом
 * `signing.agent_not_found`, и интерфейс предлагает бумагу. Эпик 38 подставляет
 * настоящую реализацию через `provideSigningPort`, не трогая виджеты.
 */
import { inject, provide, type InjectionKey, type Ref } from 'vue'
import type { ApiError } from '@/shared/api/problem'
import { PAYLOAD_TYPE_TEMPLATE } from '@/shared/contracts/constants'
import type { SignBlock } from '@/shared/contracts/procs'
import { useTokenStatus, type TokenStatus } from '@/shared/lib/token-agent'

/**
 * Запрос подписи уровня 2: блок `sign` протокола агента. Сводку для окна агент
 * считает сам из содержимого (AD-14) — страница только показывает свою копию
 * «проверьте перед подписью».
 */
export type SignRequest = Pick<SignBlock, 'level' | 'payload_type' | 'payload_b64' | 'event_type' | 'template_ref' | 'doc_format_version' | 'expected_doc_digest'>

/** Результат подписи агентом. */
export interface SignResult {
  method: 'token_agent'
  /** Ключ `key_id@версия`. */
  key_ref: string
  /** Подписанный пакет (base64). */
  signature_b64: string
}

/** Порт подписи. */
export interface SigningPort {
  /** Состояние токена на рабочем месте. */
  status: Readonly<Ref<TokenStatus>>
  /** Подписать агентом токена; окно подтверждения — в агенте. */
  sign(request: SignRequest): Promise<SignResult>
}

/**
 * Ошибка «агент токена не найден» в форме ошибки клиента API: текст берётся
 * по коду из каталога ошибок (`errors.signing.agentNotFound`).
 */
export function agentNotFoundError(): ApiError {
  return Object.assign(new Error('Агент токена не найден'), {
    status: 503,
    info: {
      type: 'urn:ant:problem:signing.agent_not_found',
      title: 'Агент токена не найден',
      status: 503,
      code: 'signing.agent_not_found',
    },
  })
}

/** Заглушка порта: агента нет — остаётся подпись на бумаге (AD-43). */
export function createStubSigningPort(): SigningPort {
  return {
    status: useTokenStatus(),
    sign: async () => {
      throw agentNotFoundError()
    },
  }
}

const SIGNING_PORT: InjectionKey<SigningPort> = Symbol('signing-port')

/** Подставить реализацию порта (эпик 38 — расширение агента токена; тесты). */
export const provideSigningPort = (port: SigningPort): void => provide(SIGNING_PORT, port)

/** Порт подписи: подставленный или заглушка. */
export const useSigningPort = (): SigningPort => inject(SIGNING_PORT, null) ?? createStubSigningPort()

/** Можно ли подписать агентом прямо сейчас. */
export const tokenReady = (status: TokenStatus): boolean => status === 'inserted'

/** Класс подписанного пакета (contracts/crypto/payload-classes.yaml). */
export type PayloadClass = 'event' | 'document-signature' | 'paper-attestation'

/**
 * payloadType пакета: `application/vnd.ant.‹класс›+json; v=‹версия›`
 * (contracts/constants.yaml). Решение человека — событие журнала (`event`),
 * подпись документа — `document-signature`, заверение бумаги — `paper-attestation`.
 */
export const payloadTypeOf = (cls: PayloadClass, version = 1): string =>
  PAYLOAD_TYPE_TEMPLATE.replace('{class}', cls).replace('{version}', String(version))

/**
 * Каноническое содержимое команды в base64 для блока `sign`. Отпечаток и сводку
 * агент считает сам; здесь — только упаковка (UTF-8 → base64).
 * @param value — команда
 */
export function toPayloadB64(value: unknown): string {
  const bytes = new TextEncoder().encode(JSON.stringify(value))
  let bin = ''
  for (const b of bytes) bin += String.fromCharCode(b)
  return btoa(bin)
}
