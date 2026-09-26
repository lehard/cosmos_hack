/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { CadCharacteristic } from './cadCharacteristic';
import type { CadComponent } from './cadComponent';
import type { CadConstraint } from './cadConstraint';
import type { CadDiscrepancy } from './cadDiscrepancy';
import type { CadLink } from './cadLink';
import type { CadZone } from './cadZone';

export interface CadAssembly {
  /** Например ФЛ-100.00.000 СБ. */
  assembly_designation: string;
  assembly_item_type_id?: string;
  assembly_name?: string;
  characteristics?: CadCharacteristic[];
  components: CadComponent[];
  constraints?: CadConstraint[];
  discrepancies?: CadDiscrepancy[];
  /** Отпечаток файла (streebog256:…). */
  file_digest: string;
  geometry_note?: string;
  /** Геометрия не импортируется (geometry: null). */
  geometry_present: boolean;
  imported_at: string;
  lifecycle_letter?: string;
  links: CadLink[];
  revision: string;
  /** Позиция записи импорта в журнале. */
  seq?: number;
  source_system?: string;
  zones?: CadZone[];
}
