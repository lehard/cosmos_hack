import { describe, expect, it } from 'vitest'
import type { PostRow } from '@/shared/api/generated/model'
import { admissionKey, keyRefOf, myPosts } from '../model/admission'

const posts: PostRow[] = [
  { workplace_id: 'WP-WELD-1', station: 'Пост сварки 1', presence: 'key_missing', assigned: { person_id: 'W21', display: 'Сварщик' } },
  { workplace_id: 'WP-WELD-2', station: 'Пост сварки 2', presence: 'not_assigned' },
]

describe('допуск к рабочему месту (эпик 37)', () => {
  it('мои посты — где я назначен', () => {
    expect(myPosts(posts, 'W21').map((p) => p.workplace_id)).toEqual(['WP-WELD-1'])
    expect(myPosts(posts, null)).toEqual([])
  })

  it('ключ сотрудника среди ключей расширения', () => {
    expect(keyRefOf({ key_refs: ['w22@1', 'w21@2'] } as never, 'W21')).toBe('w21@2')
    expect(keyRefOf(null, 'W21')).toBeNull()
  })

  it('ключ вставлен и открыт PIN-ом — pin_verified; заблокирован — нет; нет ключа — null', () => {
    expect(admissionKey('inserted', { key_refs: ['w21@1'] } as never, 'W21')).toEqual({ key_ref: 'w21@1', pin_verified: true })
    expect(admissionKey('locked', null, 'W21')).toEqual({ key_ref: 'w21@1', pin_verified: false })
    expect(admissionKey('missing', null, 'W21')).toBeNull()
    expect(admissionKey('agent_missing', null, 'W21')).toBeNull()
  })
})
