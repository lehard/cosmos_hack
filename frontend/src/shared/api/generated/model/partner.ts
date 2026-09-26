/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { PartnerChannel } from './partnerChannel';

export interface Partner {
  /** Состояние канала обмена. */
  channel: PartnerChannel;
  /** Акт регистрации партнёра (маршрут: администратор безопасности → начальник ОТК). */
  document_id: string;
  /** Адрес порта межзаводского обмена. */
  endpoint?: string;
  name: string;
  /** Код предприятия-партнёра. */
  partner_code: string;
  registered_at: string;
  /** Отпечатки корней партнёра из нашего акта регистрации. */
  root_fingerprints: string[];
}
