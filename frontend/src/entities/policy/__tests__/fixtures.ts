// Заготовки политики доступа: сотрудники, роли и полномочия, клейма, история выдачи.
import type { AccessGrantEntry, AccessPerson, AccessRoleList, AccessStamp } from '@/shared/api/generated/model'

export const persons = (): AccessPerson[] => [
  {
    person_id: 'P-QC-1',
    display_name: 'Контролёр К.',
    login: 'qc1',
    account_status: 'active',
    policy_seq: 7,
    roles: [{ role_id: 'quality_inspector', scope: 'ent01/b1/qa', valid_from: '2026-01-01T00:00:00Z', valid_until: '2026-12-31T00:00:00Z' }],
  },
  { person_id: 'P-NEW', display_name: 'Новичок Н.', account_status: 'pending', policy_seq: 7, roles: [] },
]

export const roles = (): AccessRoleList => ({
  policy_seq: 7,
  items: [
    { id: 'quality_inspector', title: 'Контролёр качества', case_role: true, inherits: [], actions: ['quality.*.read', 'nonconformity.signal.reject'] },
    { id: 'head_of_qc', title: 'Начальник ОТК', case_role: false, inherits: ['quality_inspector'], actions: ['vision.passport.admit'] },
  ],
  authorities: [{ id: 'qc_acceptance', title: 'Приёмка на точке предъявления ОТК' }],
  stamp_kinds: ['ВИК', 'РК'],
})

export const stamps = (): AccessStamp[] => [
  { stamp_id: 'ST-7', person_id: 'P-QC-1', inspection_kind: 'ВИК', order_ref: 'Приказ 12', scope: 'ent01/b1/qa', status: 'active', valid_from: '2026-01-01T00:00:00Z', valid_until: '2026-12-31T00:00:00Z' },
]

export const grants = (): AccessGrantEntry[] => [
  { event_id: 'g-1', seq: 501, at: '2026-09-20T09:00:00Z', action: 'granted', kind: 'role', subject_id: 'head_of_qc', person_id: 'P-QC-2', scope: 'ent01', by: 'P-ADM-1', second_signature_by: 'P-PM-1', ca_ref: 'CA-3', document_id: 'DOC-1' },
  { event_id: 'g-2', seq: 502, at: '2026-09-21T09:00:00Z', action: 'revoked', kind: 'stamp', subject_id: 'ST-5', person_id: 'P-QC-3', by: 'P-ADM-1' },
]
