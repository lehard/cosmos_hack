// Права в интерфейсе (AD-15, AD-21): список сервера → правила CASL;
// в воспроизведении остаётся только чтение.
import { describe, expect, it } from 'vitest'
import type { Permission } from '@/shared/api/generated/model'
import { createAppAbility, rulesFrom } from '..'

const perms: Permission[] = [
  { action: 'item.item.read', action_class: 'read', subject: 'item' },
  { action: 'nonconformity.signal.reject', action_class: 'protective', subject: 'nonconformity' },
  { action: 'nonconformity.nonconformity.confirm', action_class: 'protective', subject: 'nonconformity', object_id: 'NC-1' },
]

describe('права @casl', () => {
  it('сейчас: разрешено то, что в списке сервера', () => {
    const a = createAppAbility()
    a.update(rulesFrom(perms, false))
    expect(a.can('item.item.read', 'item')).toBe(true)
    expect(a.can('nonconformity.signal.reject', 'nonconformity')).toBe(true)
    expect(a.can('nonconformity.disposition.set', 'nonconformity')).toBe(false)
  })

  it('воспроизведение: действия выключены правилом прав, чтение остаётся', () => {
    const a = createAppAbility()
    a.update(rulesFrom(perms, true))
    expect(a.can('item.item.read', 'item')).toBe(true)
    expect(a.can('nonconformity.signal.reject', 'nonconformity')).toBe(false)
  })

  it('право по объекту — только на этот объект', () => {
    const rules = rulesFrom(perms, false)
    expect(rules[2]).toEqual({ action: 'nonconformity.nonconformity.confirm', subject: 'nonconformity', conditions: { id: 'NC-1' } })
  })
})
