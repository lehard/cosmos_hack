/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { PlanEntryKind } from './planEntryKind';

export interface PlanEntry {
  /** Доменное время по плану. Часы прогона стоят, пока он ждёт человека, — время следующих строк от нажатия не «убегает». */
  at: string;
  /** Уже произошло. */
  done: boolean;
  /** Изделие прогона. */
  item_id?: string;
  /** decision — решение человека; event — событие машины или внешней системы; alert — запланированный сбой (видно заранее); stand, tamper — служебное. */
  kind: PlanEntryKind;
  /** Метка шага определения (строка карточки). */
  label?: string;
  /** Объект решения (изделие, пост, несоответствие…), если уже известен. */
  object_id?: string;
  /** operationId решения (x-ant-action). */
  operation?: string;
  /** Персона (псевдоним) решения. */
  persona?: string;
  /** Роль стола решения. */
  role?: string;
  /** Прогон ждёт нажатия человека на его столе. */
  stop: boolean;
  /** Что произойдёт, по-русски. */
  title: string;
  /** Конец группы однотипных событий (сводки тока по минутам). */
  until?: string;
  /** Прогон ждёт именно этого решения сейчас. */
  waiting: boolean;
}
