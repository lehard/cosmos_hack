/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCPresentationAction } from './nCPresentationAction';
import type { NCPresentationPoint } from './nCPresentationPoint';
import type { NCPresentationReview } from './nCPresentationReview';
import type { NCRecommendation } from './nCRecommendation';
import type { NCRecordRef } from './nCRecordRef';

export interface NCPresentationView {
  /** Решения на экране: только их показывает интерфейс — доступность для вошедшего, почему, и последствия, вычисленные сервером. */
  actions?: NCPresentationAction[];
  /** seq, на котором построен ответ (basis_seq команды, AD-39). */
  basis_seq: number;
  item_id: string;
  /** Номер детали для людей. */
  item_label: string;
  /** Результаты методов контроля, на которых решение (как happened.after в карточке НС). */
  method_results: NCRecordRef[];
  presentation: NCPresentationPoint;
  /** Рекомендация системы — отдельно от политики: что система предлагает и почему; решает человек. */
  recommendation?: NCRecommendation;
  /** Пересмотр решения, принятого до новых данных (строка очереди kind = review). */
  review?: NCPresentationReview;
}
