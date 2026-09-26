/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ScopeItemAction } from './scopeItemAction';
import type { ScopeItemKnown } from './scopeItemKnown';
import type { ScopeItemLocation } from './scopeItemLocation';

export interface ScopeItem {
  /** Словарь incident_action. */
  action: ScopeItemAction;
  item_id: string;
  /** Ось incident словаря статусов. */
  known: ScopeItemKnown;
  label: string;
  location: ScopeItemLocation;
}
