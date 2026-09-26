// «Почему вы можете / не можете» и «Запросить решение» вместо «Подтвердить»
// (FR-146): объяснение — текст сервера, интерфейс прав не вычисляет (FR-85).
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { i18n } from '@/shared/i18n'
import { AuthorityNote, REQUEST_DECISION_ACTION, canRequestDecision } from '@/features/decision-authority'

describe('почему вы можете / не можете', () => {
  it('нет полномочий — «Почему вы не можете» и «Запросить решение»', async () => {
    const w = mount(AuthorityNote, { props: { allowed: false, canRequest: true }, global: { plugins: [i18n] } })
    expect(w.find('[data-testid="why"]').text()).toBe('Почему вы не можете это сделать')
    await w.find('[data-testid="request-decision"]').trigger('click')
    expect(w.emitted('request-decision')).toHaveLength(1)
  })

  it('есть полномочия — «Почему вы можете», без «Запросить решение»', () => {
    const w = mount(AuthorityNote, { props: { allowed: true, canRequest: true }, global: { plugins: [i18n] } })
    expect(w.find('[data-testid="why"]').text()).toBe('Почему вы можете это сделать')
    expect(w.find('[data-testid="request-decision"]').exists()).toBe(false)
  })

  it('объяснение сервера: причина, код отказа названием каталога, что можно вместо', () => {
    const w = mount(AuthorityNote, {
      props: {
        allowed: false,
        open: true,
        explanation: {
          action: 'nonconformity.disposition.set',
          allowed: false,
          code: 'access.separation_of_duties',
          reason: 'Вы участвовали в изготовлении FL-0042; нужна подпись начальника ОТК',
          allowed_actions: [REQUEST_DECISION_ACTION],
        },
        actionLabels: { [REQUEST_DECISION_ACTION]: 'Запросить решение' },
      },
      global: { plugins: [i18n] },
    })
    const e = w.find('[data-testid="explanation"]')
    expect(e.text()).toContain('Вы участвовали в изготовлении FL-0042')
    expect(e.find('[data-testid="explanation-code"]').text()).toBe('Разделение обязанностей')
    expect(e.text()).toContain('Разрешено: Запросить решение')
  })

  it('права ещё не пришли — ни «можете», ни «не можете»', () => {
    const w = mount(AuthorityNote, { props: { allowed: null }, global: { plugins: [i18n] } })
    expect(w.find('[data-testid="why"]').exists()).toBe(false)
    expect(w.find('[data-testid="authority"]').attributes('data-allowed')).toBe('unknown')
  })

  it('запросить решение можно, если это назвал сервер или дал право', () => {
    expect(canRequestDecision({ action: 'x', allowed: false, allowed_actions: [REQUEST_DECISION_ACTION] }, null)).toBe(true)
    expect(canRequestDecision(null, new Set([REQUEST_DECISION_ACTION]))).toBe(true)
    expect(canRequestDecision(null, new Set())).toBe(false)
  })
})
