/**
 * Доступ к серверу (слой shared, FSD; AD-20, AD-21).
 *
 * - `generated/` — клиент orval + Vue Query и таблицы контрактов; руками не правится,
 *   перегенерация — `npm run generate` (make generate-frontend).
 * - `sse/` — единственный ручной сетевой модуль: живые обновления.
 * - здесь — соглашение о ключах кэша, разбор ошибок и режим ответа.
 */
export * from './keys'
export * from './problem'
export * from './response'
