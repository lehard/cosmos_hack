/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCCardApprovalsStatus } from './nCCardApprovalsStatus';
import type { NCCardInvestigationStatus } from './nCCardInvestigationStatus';
import type { NCCardOrigin } from './nCCardOrigin';
import type { NCCardResolution } from './nCCardResolution';
import type { NCCardStatus } from './nCCardStatus';
import type { NCContainmentSource } from './nCContainmentSource';
import type { NCEvidence } from './nCEvidence';
import type { NCHandoff } from './nCHandoff';
import type { NCHappened } from './nCHappened';
import type { NCIsolation } from './nCIsolation';
import type { NCItemAxes } from './nCItemAxes';
import type { NCPresentationContext } from './nCPresentationContext';
import type { NCRecordRef } from './nCRecordRef';
import type { NCSystemAnalysis } from './nCSystemAnalysis';
import type { NCToDecide } from './nCToDecide';

export interface NCCard {
  /** Подписи маршрута решения (режим 4): pending — решение не исполняется; demo_stub — демо, подписи не проверялись. */
  approvals_status?: NCCardApprovalsStatus;
  axes: NCItemAxes;
  /** seq, на котором построена карточка (для basis_seq команд, AD-39). */
  basis_seq: number;
  /** Решение принимает комиссия (специальный процесс, FR-151). */
  commission?: boolean;
  /** Действующие основания сдерживания. */
  containment?: NCContainmentSource[];
  evidence: NCEvidence;
  /** Изделия группового несоответствия: окно нарушения специального процесса — одно несоответствие на все изделия окна, решение комиссии приходит каждому (FR-151). */
  group_item_ids?: string[];
  /** Кому передано исполнение решения по изделию и в каком оно состоянии; нет решения — поля нет. */
  handoff?: NCHandoff;
  happened: NCHappened;
  /** Решения людей с подписью (отдельно от вывода системы). */
  human_decisions: NCRecordRef[];
  /** Второй статус несоответствия — «системное расследование» (FR-51): закрытие по изделию его не закрывает. */
  investigation_status?: NCCardInvestigationStatus;
  isolation?: NCIsolation;
  item_id: string;
  item_label: string;
  nc_id: string;
  /** Номер для людей. */
  number: string;
  /** Черновик по сигналу или регистрация окна нарушения специального процесса (FR-151). */
  origin?: NCCardOrigin;
  /** «Изолировано в системе, физически не перемещено» (FR-55). */
  physically_not_moved?: boolean;
  /** Контекст точки предъявления, если изделие ждёт решения на ней. */
  presentation?: NCPresentationContext;
  /** Исход несоответствия, закрытого без решения по изделию. */
  resolution?: NCCardResolution;
  /** Ревизия правила, построившего черновик (карта реакций). */
  rule_rev?: string;
  status: NCCardStatus;
  system_analysis: NCSystemAnalysis;
  to_decide: NCToDecide;
}
