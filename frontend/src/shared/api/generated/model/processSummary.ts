/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ProcessSummaryStatus } from './processSummaryStatus';
import type { ProcessVersionRef } from './processVersionRef';

export interface ProcessSummary {
  /** Действующая версия; нет — ни одна версия не в действии. */
  active_version?: ProcessVersionRef;
  /** Основной процесс: его показывают живая карта и список версий без параметра процесса. */
  is_default: boolean;
  /**
     * Изделий в работе по всем версиям процесса.
     * @minimum 0
     */
  items_in_work: number;
  /** Название процесса (имя bpmn:process действующей версии). */
  name: string;
  /** id главного bpmn:process; параметр process_id живой карты и списка версий. */
  process_id: string;
  /** active — есть действующая версия; draft — ни одна версия ещё не введена; retired — все введённые выведены. */
  status: ProcessSummaryStatus;
  /**
     * Число версий процесса (все статусы).
     * @minimum 0
     */
  versions: number;
}
