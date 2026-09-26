/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { CloseBlocker } from './closeBlocker';
import type { FactorRef } from './factorRef';
import type { IncidentSummaryStage } from './incidentSummaryStage';
import type { IncidentSummaryStatus } from './incidentSummaryStatus';
import type { KnownCountsView } from './knownCountsView';

export interface IncidentSummary {
  /** Почему расследование нельзя закрыть: те же коды, что вернёт POST /incidents/{id}/close со scope=investigation (422). Пусто — можно. */
  close_blockers: CloseBlocker[];
  common_factor?: FactorRef;
  /** Изделия текущей версии области по оси «что известно». */
  counts: KnownCountsView;
  /** Ключ группы /analysis/groups (вид дефекта × операция × оборудование) ведущего несоответствия. */
  group_key?: string;
  incident_id: string;
  /** @minimum 0 */
  initial_size: number;
  label: string;
  /** Время последней записи по инциденту (версия области, гипотеза, причина, мера). */
  last_event_at?: string;
  /** Несоответствия инцидента: по ним — /nonconformities/{nc_id}/circumstances и /hypotheses. */
  nc_ids: string[];
  /**
     * Что сделать следующим — текст сервера; null — делать нечего.
     * @nullable
     */
  next_step: string | null;
  opened_at: string;
  /** Несоответствие с гипотезами — вход «Почему могло произойти». */
  primary_nc_id?: string;
  /** @minimum 0 */
  scope_version: number;
  /** @minimum 0 */
  size: number;
  /** Стадия расследования: словарь investigation_stage (contracts/statuses.yaml). */
  stage: IncidentSummaryStage;
  /** Область риска: open — идёт, closed — решение по изделиям принято (расследование — stage). */
  status: IncidentSummaryStatus;
}
