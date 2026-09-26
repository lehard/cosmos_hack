/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { KeyActSignature } from './keyActSignature';
import type { KeyView } from './keyView';

export interface KeyDetails {
  basis_seq: number;
  /** Прежние версии ключа субъекта по ротации. */
  history: KeyView[];
  key: KeyView;
  /** Подписи акта регистрации (владение, подтверждение субъекта, вторая подпись). */
  signatures: KeyActSignature[];
}
