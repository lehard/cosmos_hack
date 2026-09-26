/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ItemSignatureCheck } from './itemSignatureCheck';
import type { ItemSignatureClass } from './itemSignatureClass';
import type { ItemSignatureKeyStorage } from './itemSignatureKeyStorage';
import type { ItemSignatureMethod } from './itemSignatureMethod';

export interface ItemSignature {
  /** Заверитель бумажной подписи (AD-43). */
  attested_by?: string;
  /** Статус проверки: цело / отвергнуто / не проверяемо / не проверялась (демо без подписей). */
  check: ItemSignatureCheck;
  /** Класс происхождения подписи (AD-2). */
  class: ItemSignatureClass;
  /** Класс хранения ключа подписанта (AD-14, Д-72): физический ключ или ключ в браузере. */
  key_storage?: ItemSignatureKeyStorage;
  /**
     * Уровень подписи (AD-13).
     * @minimum 0
     * @maximum 3
     */
  level: number;
  /** Способ подписи решения: агент токена или ключ в браузере, бумага с заверением, демо-подписант (Д-59). */
  method?: ItemSignatureMethod;
  /** Учётный номер бумажного оригинала в архиве ОТК (AD-43). */
  paper_original_ref?: string;
  /** Адрес скана в хранилище материалов: streebog256:… (AD-23, AD-43). */
  scan_address?: string;
  /** Псевдоним подписанта или id устройства. */
  signer_id: string;
}
