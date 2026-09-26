/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCSummaryDisposition } from './nCSummaryDisposition';
import type { NCSummarySeverity } from './nCSummarySeverity';
import type { NCSummaryStatus } from './nCSummaryStatus';

export interface NCSummary {
  defect_type_code?: string;
  disposition: NCSummaryDisposition;
  found_at: string;
  item_id: string;
  item_label: string;
  nc_id: string;
  number: string;
  severity: NCSummarySeverity;
  status: NCSummaryStatus;
  step_key: string;
}
