/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCSummaryContainment } from './nCSummaryContainment';
import type { NCSummaryDisposition } from './nCSummaryDisposition';
import type { NCSummaryInvestigationStatus } from './nCSummaryInvestigationStatus';
import type { NCSummarySeverity } from './nCSummarySeverity';
import type { NCSummaryStatus } from './nCSummaryStatus';

export interface NCSummary {
  /** Решение — комиссия (FR-151). */
  commission?: boolean;
  /** Сдерживание изделия. */
  containment?: NCSummaryContainment;
  defect_type_code?: string;
  /** Вид дефекта по-русски — из классификатора видов дефектов; нет в классификаторе — поля нет. */
  defect_type_label?: string;
  disposition: NCSummaryDisposition;
  found_at: string;
  /** Системное расследование (FR-51). */
  investigation_status?: NCSummaryInvestigationStatus;
  item_id: string;
  item_label: string;
  nc_id: string;
  number: string;
  severity: NCSummarySeverity;
  status: NCSummaryStatus;
  step_key: string;
}
