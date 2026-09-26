/**
 * Порт подписи решения (FR-66, FR-69, FR-139; AD-13, AD-14, AD-43). Один порт —
 * адаптеры за ним (Д-72): ключ в браузере (расширение «Главный — подпись»,
 * пакет подписи WASM) и физический ключ (то же расширение — транслятор к
 * агенту токена по Native Messaging), плюс бумага с заверением вторым человеком.
 *
 * Сервер закрытых ключей не хранит (FR-69): подпись делает расширение
 * (contracts/internal/token-agent, `sign` / `sign_batch`); окно подтверждения
 * уровня 2 — окно расширения или агента, сводку в нём пакет подписи считает
 * сам. Расширения нет — `sign` отвечает `signing.agent_not_found`, и интерфейс
 * предлагает бумагу. Тесты подставляют свой порт через `provideSigningPort`.
 */
import { inject, provide, readonly, ref, type InjectionKey, type Ref } from 'vue'
import type { DsseEnvelope } from '@/shared/api/generated/model'
import type { ApiError } from '@/shared/api/problem'
import { PAYLOAD_TYPE_TEMPLATE } from '@/shared/contracts/constants'
import type { SignBlock } from '@/shared/contracts/procs'
import { extensionRequest, SIGN_TIMEOUT_MS, useTokenStatus, type TokenStatus } from '@/shared/lib/token-agent'

/**
 * Запрос подписи: блок `sign` протокола агента. Для команды решения —
 * `command_request` (операция, параметры пути, изделие) и тело команды в
 * `payload_b64`: событие-команду, отпечаток и сводку расширение собирает само
 * тем же пакетом, что сервер (AD-12, AD-14). Страница показывает только свою
 * копию «проверьте перед подписью».
 */
export type SignRequest = Pick<
  SignBlock,
  'level' | 'payload_type' | 'payload_b64' | 'event_type' | 'template_ref' | 'doc_format_version' | 'expected_doc_digest' | 'command_request'
>

/** Результат подписи — конверт DSSE для поля `signature` команды (AD-10). */
export type SignResult = DsseEnvelope

/** Порт подписи. */
export interface SigningPort {
  /** Состояние ключа на рабочем месте. */
  status: Readonly<Ref<TokenStatus>>
  /** Подписать; окно подтверждения — в расширении или агенте. */
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

/**
 * Отказ расширения или агента в форме ошибки клиента API: текст — по коду из
 * каталога ошибок (signing.level_not_allowed, signing.pin_wrong, signing.cancelled…).
 * @param code — код ошибки
 * @param message — пояснение расширения или агента
 */
export function agentError(code: string, message?: string): ApiError {
  if (code === 'signing.agent_not_found') return agentNotFoundError()
  return Object.assign(new Error(message ?? code), {
    status: 422,
    info: { type: `urn:ant:problem:${code}`, title: message ?? code, status: 422, code, detail: message },
  })
}

/** Заглушка порта: агента нет — остаётся подпись на бумаге (AD-43). */
export function createStubSigningPort(): SigningPort {
  return {
    status: readonly(ref<TokenStatus>('agent_missing')),
    sign: async () => {
      throw agentNotFoundError()
    },
  }
}

/**
 * Порт через расширение «Главный — подпись» (эпик 38): уровень 2 — окно
 * расширения (или агента токена) со сводкой, посчитанной пакетом подписи;
 * уровень 1 вне перечня расширение отклоняет само.
 */
export function createExtensionSigningPort(): SigningPort {
  return {
    status: useTokenStatus(),
    sign: async (request) => {
      const r = await extensionRequest({ type: 'sign', sign: request }, SIGN_TIMEOUT_MS)
      const env = r.type === 'signed' ? r.signed?.[0]?.envelope : undefined
      if (env) return env as SignResult
      throw agentError(r.error?.code ?? 'signing.agent_not_found', r.error?.message)
    },
  }
}

const SIGNING_PORT: InjectionKey<SigningPort> = Symbol('signing-port')

/** Подставить реализацию порта (тесты, другие адаптеры). */
export const provideSigningPort = (port: SigningPort): void => provide(SIGNING_PORT, port)

/** Порт подписи: подставленный или через расширение. */
export const useSigningPort = (): SigningPort => inject(SIGNING_PORT, null) ?? createExtensionSigningPort()

/** Можно ли подписать ключом сейчас: ключ готов или PIN спросит окно подписи. */
export const tokenReady = (status: TokenStatus): boolean => status === 'inserted' || status === 'locked'

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
