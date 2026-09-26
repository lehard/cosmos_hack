/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { CriticalActionSignerKeyClass } from './criticalActionSignerKeyClass';
import type { CriticalActionSignerKeyStorage } from './criticalActionSignerKeyStorage';

export interface CriticalActionSigner {
  /** Подписант словами. */
  display?: string;
  /** Класс подписи (AD-10): personal — ключ человека, server_attested — заверено сервером, paper — бумага с заверением, scenario — ключ сценария. */
  key_class: CriticalActionSignerKeyClass;
  /** Ключ key_id@версия. */
  key_id?: string;
  /** Класс хранения ключа человека (AD-14, Д-72): физический ключ или ключ в браузере под PIN. */
  key_storage?: CriticalActionSignerKeyStorage;
  /** Подписант: псевдоним сотрудника или источник (ant — заверение сервером). */
  signer_id: string;
}
