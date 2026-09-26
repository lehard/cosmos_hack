/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCAnalyzerStage } from './nCAnalyzerStage';
import type { NCRecordRef } from './nCRecordRef';
import type { NCSourceSignalBasisKind } from './nCSourceSignalBasisKind';
import type { NCSourceSignalSeverity } from './nCSourceSignalSeverity';
import type { NCSourceSignalVersions } from './nCSourceSignalVersions';

export interface NCSourceSignal {
  /**
     * Уверенность анализатора, б. п.; ≠ вероятность брака.
     * @minimum 0
     * @maximum 10000
     */
  analyzer_confidence_bp?: number;
  basis_kind: NCSourceSignalBasisKind;
  defect_type_code?: string;
  defect_type_known: boolean;
  /** Вид дефекта по-русски — из классификатора видов дефектов; нет в классификаторе — поля нет. */
  defect_type_label?: string;
  /** Адреса материалов: кадры, иллюстрация. */
  evidence_refs: string[];
  /**
     * Качество наблюдения, б. п.
     * @minimum 0
     * @maximum 10000
     */
  observation_quality_bp?: number;
  record: NCRecordRef;
  severity: NCSourceSignalSeverity;
  signal_id: string;
  stages: NCAnalyzerStage[];
  /** Вектор версий наблюдения (AD-29). */
  versions?: NCSourceSignalVersions;
  zone_id?: string;
  /** Зона по-русски — из зон типа изделия по КД (справочник номенклатуры); нет в справочнике — поля нет. */
  zone_label?: string;
}
