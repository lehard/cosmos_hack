// Образец карточки «требуется ваше решение» — только в тестах (PRD §11.10).
// Представитель заказчика согласует «как есть» по разрешению на отклонение
// (FR-136, режим 5 FR-50); контролёр ОТК уже подписал.
import type { DecisionRequest } from '@/entities/document'
import { at } from '@/entities/item/__tests__/fixtures'

export const useAsIsRequest = (over: Partial<DecisionRequest> = {}): DecisionRequest => ({
  document: {
    document_id: 'DOC-NCD-142',
    version: 1,
    template_ref: 'nc_decision@2',
    doc_type: 'nonconformity_decision',
    doc_digest: 'streebog256:c0ffee',
    status: 'in_route',
    drafted_at: at('10:00'),
    route: [
      {
        stage: 1,
        authority_id: 'qc.disposition',
        authority_label: 'Контролёр ОТК',
        quorum: 'one',
        signature_level: 2,
        paper_allowed: false,
        external_party: 'none',
        required: 1,
        signatures: [
          { event_id: 's-1', signed_at: at('10:05'), counted: true, check: 'valid', class: 'personal', level: 2, signer_id: 'qc-03' },
          { event_id: 's-0', signed_at: at('09:50'), counted: false, previous_version: true, check: 'valid', class: 'personal', level: 2, signer_id: 'qc-03' },
        ],
      },
      {
        stage: 2,
        authority_id: 'customer.concession',
        authority_label: 'Представитель заказчика',
        quorum: 'one',
        signature_level: 2,
        paper_allowed: true,
        attester_authority_id: 'paper.attest',
        external_party: 'customer_representative',
        required: 1,
        signatures: [],
      },
    ],
  },
  proposal: {
    kind: 'disposition',
    code: 'use_as_is',
    summary: 'Принять FL-0042 «как есть» по разрешению РО-12/26',
    item_id: 'ENT:FL-0042',
    item_label: 'FL-0042',
    nc_id: 'NC-0142',
    nc_number: 'НС-0142',
  },
  escalation: { reason: 'Решение «как есть» требует согласования с представителем заказчика', automation_mode: 5, rule_id: 'AUTH-use-as-is' },
  evidence: [
    {
      event_id: 'e-kt3',
      event_type: 'inspection.result.recorded',
      occurred_at: at('08:52'),
      evidence_refs: [
        { material_address: 'streebog256:beef01', media_type: 'image/png', kind: 'photo', is_illustration: false },
        { material_address: 'streebog256:beef02', media_type: 'image/png', kind: 'illustration', is_illustration: true, source_note: 'открытый набор' },
      ],
    },
  ],
  similar_accepted: [{ ref_id: 'NC-0117', number: 'НС-0117', summary: '«Как есть», поры 0,3 мм', outcome: 'accepted', decided_at: at('07:00') }],
  similar_rejected: [{ ref_id: 'NC-0098', number: 'НС-0098', summary: 'Отказ: поры в зоне уплотнения', outcome: 'rejected', decided_at: at('06:00') }],
  my_stage: 2,
  expected_signer: 'vp-01',
  due_at: at('16:00'),
  ...over,
})
