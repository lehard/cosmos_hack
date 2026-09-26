/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { TimelineMark } from './timelineMark';

export interface TimelineData {
  /** Начало доступной истории (или прогона). */
  from: string;
  marks: TimelineMark[];
  /** Конец — «сейчас» на сервере. */
  to: string;
}
