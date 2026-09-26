/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { InjectionInjection } from './injectionInjection';

export interface Injection {
  /** Доступна в текущем состоянии прогона и профиле (tamper_outside — только fixtures и demo). */
  available: boolean;
  /** Что произойдёт и где это видно (столы, табло). */
  description?: string;
  injection: InjectionInjection;
  /** Нужна целевая запись (target_event_id). */
  needs_target: boolean;
  title: string;
}
