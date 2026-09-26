// Правила совместимости контракта событий поверх @asyncapi/diff (AD-20, FR-29).
// Библиотека классифицирует изменения только на уровне сообщения целиком, а
// изменения внутри payload оставляет «unclassified». Здесь — наши правила
// пяти случаев FR-29 для потребителей журнала:
//   ломающие: удаление канала, сообщения или свойства; изменение списка
//     обязательных полей; изменение type / const / pattern / format / $ref;
//     удаление значения перечисления;
//   совместимые: новое необязательное свойство, новое значение перечисления
//     (приём даёт UNKNOWN(значение) с флагом), тексты описаний.
//   исключения (WIDENED_PATTERNS): расширение шаблона строки, при котором все
//     прежние значения остаются допустимыми, — это расширение допустимых
//     значений, совместимый случай FR-29 (как новое значение перечисления).
//     Разрешена только точная пара «было → стало» на названных полях.
import { diff } from '@asyncapi/diff';

/**
 * Разрешённые расширения шаблонов (решения дирижёра):
 *  - Д-66: классы подписанных пакетов key.registration.recorded.payload_classes
 *    и key.profile.registered.object_classes — шаблон `code` → `class`
 *    (допускает дефис: `document-signature`, `key-act` из
 *    contracts/crypto/payload-classes.yaml). Прежние значения по-прежнему
 *    допустимы; шаблон `code` для остальных полей не меняется.
 */
const WIDENED_PATTERNS = [
  { path: /\/properties\/(payload_classes|object_classes)\/items\/pattern$/, from: '^[a-z][a-z0-9_]*$', to: '^[a-z][a-z0-9_-]*$' },
];

function widened(c) {
  return c.action === 'edit' && WIDENED_PATTERNS.some((w) => w.path.test(c.path) && c.before === w.from && c.after === w.to);
}

const TEXT = /\/(description|title|summary|x-parser-[a-z-]+)(\/|$)/;
const STRICT = /\/(type|const|pattern|format|\$ref|additionalProperties)(\/\d+)?$/;

/**
 * Списки сообщений операций — в словари по имени сообщения: @asyncapi/diff
 * сравнивает массивы по индексу, и новый тип в середине семейства (порядок
 * каталога) выглядел бы как правка и удаление всех следующих сообщений.
 * По имени новый тип — добавление, удалённый — удаление.
 */
function byMessageName(doc) {
  const d = structuredClone(doc);
  for (const op of Object.values(d.operations ?? {})) {
    if (!Array.isArray(op?.messages)) continue;
    op.messages = Object.fromEntries(op.messages.map((m, i) => [m?.name ?? m?.['x-parser-message-name'] ?? `#${i}`, m]));
  }
  return d;
}

/** Изменения, ломающие потребителей: библиотечные breaking + наши правила. */
export function breakingChanges(baseDoc, nextDoc) {
  const out = diff(byMessageName(baseDoc), byMessageName(nextDoc), { outputType: 'json' });
  const lib = new Set(out.breaking());
  const all = [...out.breaking(), ...out.nonBreaking(), ...out.unclassified()];
  return all.filter((c) => {
    if (lib.has(c)) return true;
    if (TEXT.test(c.path)) return false;
    if (/\/required(\/\d+)?$/.test(c.path)) return true;
    if (/\/enum\/\d+$/.test(c.path)) return c.action === 'remove' || c.action === 'edit';
    if (c.action === 'remove') return true;
    if (widened(c)) return false;
    if (c.action === 'edit' && STRICT.test(c.path)) return true;
    return false;
  });
}
