/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { FactorRef } from './factorRef';
import type { KnownGood } from './knownGood';
import type { NarrowOption } from './narrowOption';
import type { ScopeItem } from './scopeItem';
import type { ScopeVersion } from './scopeVersion';
import type { TimeWindow } from './timeWindow';

export interface RiskScope {
  basis_seq: number;
  common_factor?: FactorRef;
  /** Ключ группы /analysis/groups (вид дефекта × операция × оборудование) ведущего несоответствия. */
  group_key?: string;
  incident_id: string;
  incident_label: string;
  /** Изделия текущей версии. */
  items: ScopeItem[];
  last_known_good?: KnownGood;
  /** Готовые сужения области по данным (кнопка в один щелчок, FR-61): изделия и основание из журнала; решение и подпись — человека (incident.scope.narrow). */
  narrow_options?: NarrowOption[];
  /** Несоответствия инцидента: по ним — /nonconformities/{nc_id}/circumstances и /hypotheses. */
  nc_ids: string[];
  /** Несоответствие с гипотезами — вход «Почему могло произойти». */
  primary_nc_id?: string;
  shipped_to_partners?: number;
  /** По возрастанию. */
  versions: ScopeVersion[];
  window?: TimeWindow;
}
