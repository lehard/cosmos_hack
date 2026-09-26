/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ItemSignatureCheck } from './itemSignatureCheck';
import type { ItemSignatureClass } from './itemSignatureClass';

export interface ItemSignature {
  /** Заверитель бумажной подписи (AD-43). */
  attested_by?: string;
  /** Статус проверки: цело / отвергнуто / не проверяемо / не проверялась (демо без подписей). */
  check: ItemSignatureCheck;
  /** Класс происхождения подписи (AD-2). */
  class: ItemSignatureClass;
  /**
     * Уровень подписи (AD-13).
     * @minimum 0
     * @maximum 3
     */
  level: number;
  /** Учётный номер бумажного оригинала в архиве ОТК (AD-43). */
  paper_original_ref?: string;
  /** Адрес скана в хранилище материалов: streebog256:… (AD-23, AD-43). */
  scan_address?: string;
  /** Псевдоним подписанта или id устройства. */
  signer_id: string;
}
