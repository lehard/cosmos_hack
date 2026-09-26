/**
 * Права в интерфейсе (AD-15 «Для фронтенда», FR-85): интерфейс сам права не
 * вычисляет. Сервер отдаёт плоский список «действие → объект», вычисленный тем же
 * Enforce, что проверяет команды; здесь он превращается в правила @casl/ability,
 * а @casl/vue показывает или прячет действия (`<Can>`, `useAbility`).
 *
 * Действие — x-ant-action id `‹модуль›.‹объект›.‹действие›`, объект — вид
 * сущности (EntityKind) или `all`. В воспроизведении (as_of задан) в правила
 * попадает только чтение — «действия выключены правилом прав» (AD-21).
 */
import { createMongoAbility, type MongoAbility, type RawRuleOf } from '@casl/ability'
import type { Permission } from '@/shared/api/generated/model'

/** Способность пользователя: [действие, объект]. */
export type AppAbility = MongoAbility<[string, string]>

/** Пустая способность: пока список прав не пришёл, ничего нельзя. */
export const createAppAbility = (): AppAbility => createMongoAbility<[string, string]>([])

/**
 * Правила CASL из списка сервера.
 * @param permissions — список `access.permission.list`
 * @param replay — воспроизведение: оставить только чтение
 */
export function rulesFrom(permissions: readonly Permission[], replay: boolean): RawRuleOf<AppAbility>[] {
  return permissions
    .filter((p) => !replay || p.action_class === 'read')
    .map((p) => (p.object_id ? { action: p.action, subject: p.subject, conditions: { id: p.object_id } } : { action: p.action, subject: p.subject }))
}
