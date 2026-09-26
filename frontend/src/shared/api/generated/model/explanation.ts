/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ExplainAuthority } from './explainAuthority';
import type { ExplainHolder } from './explainHolder';
import type { ExplainRole } from './explainRole';
import type { ExplainStamp } from './explainStamp';

export interface Explanation {
  /** x-ant-action id. */
  action: string;
  allowed: boolean;
  /** Что можно сделать вместо (например, «Запросить решение»). */
  allowed_actions: string[];
  /** Ваши действующие полномочия — рамки доменных гардов (FR-50, FR-78). */
  authorities?: ExplainAuthority[];
  /** Вы — редкий подписант: доступны только адресованные вам карточки решения (FR-136). */
  card_only?: boolean;
  /** Код отказа из contracts/errors.yaml. */
  code?: string;
  /** Какая ваша роль разрешает действие: назначенная роль, базовая роль с этим действием, область. */
  granted_by?: ExplainRole;
  /** Место операции (область), по которому принято решение (барьер 3). */
  place?: string;
  /** Версия политики объяснения (AD-39). */
  policy_seq?: number;
  /** Объяснение по-русски: роль, область, полномочие, клеймо, разделение обязанностей. */
  reason?: string;
  /** Ваши действующие роли в областях. */
  roles?: ExplainRole[];
  /** Ваши действующие цифровые клейма (FR-145). */
  stamps?: ExplainStamp[];
  /** Кто может выполнить действие: роли политики и сотрудники с ними. */
  who_can?: ExplainHolder[];
}
