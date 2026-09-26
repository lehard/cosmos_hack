/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DrillRef } from './drillRef';
import type { GrantApprovalStage } from './grantApprovalStage';
import type { GrantAssessmentDomain } from './grantAssessmentDomain';

export interface GrantAssessment {
  /** Расширяются права администратора или аудита — привилегированная выдача, кто бы её ни делал. */
  admin_expansion: boolean;
  /** Решение документа (grant:‹вид›:‹что›:‹кому›:‹область›) — поле decision запроса решения. */
  decision: string;
  /** Сфера выдачи (AD-11): обычная, ОТК, производство, администраторы и аудит. */
  domain: GrantAssessmentDomain;
  /** Права, которые добавляет выдача: action:шаблон@область, authority:id@область, stamp:вид@область. */
  expands: string[];
  policy_seq: number;
  reason: string;
  /** Полномочие второй подписи; пусто — одной подписью администратора безопасности. */
  second_authority?: string;
  /** Кто ставит вторую подпись (по-русски). */
  second_signer: string;
  /** Выдача себе: изменение расширяет права инициатора (по эффективным правам). */
  self_grant: boolean;
  /** Этапы подписей документа выдачи (RequiredApprovals, AD-43). */
  stages: GrantApprovalStage[];
  /** Объект документа — сотрудник. */
  subject: DrillRef;
  /** Шаблон документа «Выдача ролей, полномочий, клейм» (шаблон@версия). */
  template?: string;
}
