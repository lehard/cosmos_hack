/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { JournalRecordRef } from './journalRecordRef';
import type { Reason } from './reason';
import type { ScopeBreakdown } from './scopeBreakdown';
import type { ScopeTrigger } from './scopeTrigger';
import type { ScopeVersionChange } from './scopeVersionChange';
import type { ScopeVersionKeyClass } from './scopeVersionKeyClass';

export interface ScopeVersion {
  /**
     * Псевдоним; null — правило системы.
     * @nullable
     */
  author: string | null;
  /** Имя автора версии из справочника людей; нет — правило системы или справочник недоступен. */
  author_name?: string;
  breakdown: ScopeBreakdown;
  /** incident.scope.computed / expanded / narrowed. */
  change: ScopeVersionChange;
  /** Доказательства версии словами (те же записи, что evidence_event_ids). */
  evidence: JournalRecordRef[];
  evidence_event_ids: string[];
  /** Изделия, вошедшие в область этой версией (item_id); у первой версии — все. */
  items_added: string[];
  /** Изделия, исключённые этой версией (item_id). */
  items_removed: string[];
  /** Класс ключа подписи сужения (AD-10). */
  key_class?: ScopeVersionKeyClass;
  reason?: Reason;
  recorded_at: string;
  /** @minimum 1 */
  scope_version: number;
  /** Кто подписал сужение (псевдоним). */
  signed_by?: string;
  /** @minimum 0 */
  size: number;
  /** Повод ступени — вместо разбора reason.code. */
  trigger?: ScopeTrigger;
}
