/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { CadComponent } from './cadComponent';
import type { CadLink } from './cadLink';

export interface CadAssembly {
  /** Например ФЛ-100.00.000 СБ. */
  assembly_designation: string;
  components: CadComponent[];
  /** Отпечаток файла (streebog256:…). */
  file_digest: string;
  /** Геометрия не импортируется (geometry: null). */
  geometry_present: boolean;
  imported_at: string;
  links: CadLink[];
  revision: string;
}
