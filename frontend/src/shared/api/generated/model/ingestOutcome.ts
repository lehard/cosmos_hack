/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { IngestOutcomeStatus } from './ingestOutcomeStatus';

export interface IngestOutcome {
  /** Код из contracts/errors.yaml (ingest.*): неизвестная версия, нет поля, подпись… */
  code?: string;
  event_id?: string;
  /** Флаги аномалий (ingest.anomaly.flagged): future_timestamp, clock_skew, sequence_violation, unknown_enum_value, unknown_defect_type, late_write. */
  flags: string[];
  /**
     * Позиция конверта в пачке.
     * @minimum 0
     */
  index: number;
  quarantine_id?: string;
  /** Позиция записи в журнале (для accepted). */
  seq?: number;
  /** false — подпись не проверялась (профиль demo до эпика 05, заметка эпика 06). */
  signature_checked: boolean;
  /** accepted — записан; duplicate — тот же payload уже есть (повтор не меняет показатели, AD-7); quarantined — в карантине с кодом; rejected — отказ без карантина. */
  status: IngestOutcomeStatus;
}
