// Окно подтверждения уровня 2 за портом подписи и бумажный путь с QR и
// заверением (FR-66, FR-69, FR-139; AD-13, AD-14, AD-43).
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { i18n } from '@/shared/i18n'
import { agentNotFoundError, createStubSigningPort, payloadTypeOf, toPayloadB64 } from '@/features/sign-decision'
import { problemMessage } from '@/shared/api/problem'
import PaperSignPanel from '../ui/PaperSignPanel.vue'
import SignConfirmPanel from '../ui/SignConfirmPanel.vue'

const summary = [
  { labelKey: 'widgets.signing.fields.action', valueKey: 'decisions.signal.rejectSignal' },
  { labelKey: 'common.words.item', value: 'FL-0042' },
  { labelKey: 'common.words.basis', value: 'Блик на кромке' },
]
const mountConfirm = (props = {}) => mount(SignConfirmPanel, { props: { summary, tokenStatus: 'agent_missing', ...props }, global: { plugins: [i18n] } })

describe('окно подтверждения уровня 2', () => {
  it('сводка «проверьте перед подписью» — поля решения', () => {
    const w = mountConfirm({ tokenStatus: 'inserted' })
    expect(w.text()).toContain('Проверьте перед подписью')
    expect(w.text()).toContain('Уровень 2 — закрывающее решение: сводка и явное подтверждение')
    const dd = w.findAll('[data-testid="summary"] dd').map((x) => x.text())
    expect(dd).toEqual(['Отклонить сигнал — изделие продолжает маршрут', 'FL-0042', 'Блик на кромке'])
  })

  it('токен вставлен — подпись касанием токена', async () => {
    const w = mountConfirm({ tokenStatus: 'inserted' })
    expect(w.find('[data-testid="confirm-token"]').attributes('disabled')).toBeUndefined()
    await w.find('[data-testid="confirm-token"]').trigger('click')
    expect(w.emitted('confirm-token')).toHaveLength(1)
  })

  it('агента нет — подписать токеном нельзя, остаётся бумага', () => {
    const w = mountConfirm()
    expect(w.find('[data-testid="confirm-token"]').attributes('disabled')).toBeDefined()
    expect(w.find('[data-testid="token-note"]').text()).toContain('Агент токена не найден — доступна подпись на бумаге')
    expect(w.find('[data-testid="sign-paper"]').exists()).toBe(true)
  })

  it('ни агента, ни бумаги — решение не подписывается (FR-69)', () => {
    const w = mountConfirm({ paperAllowed: false })
    expect(w.find('[data-testid="no-path"]').text()).toContain('Нет ни агента токена, ни подписи на бумаге')
  })

  it('демонстрационный профиль: подтверждение без агента названо прямо', async () => {
    const w = mountConfirm({ paperAllowed: false, demoUnsigned: true })
    expect(w.find('[data-testid="no-path"]').exists()).toBe(false)
    expect(w.find('[data-testid="demo-note"]').text()).toContain('«подпись не проверялась»')
    await w.find('[data-testid="confirm-unsigned"]').trigger('click')
    expect(w.emitted('confirm-unsigned')).toHaveLength(1)
  })
})

describe('порт подписи — заглушка до эпика 38', () => {
  it('агент «не найден» — ошибка с кодом signing.agent_not_found', async () => {
    const port = createStubSigningPort()
    expect(port.status.value).toBe('agent_missing')
    await expect(port.sign({ level: 2, payload_type: payloadTypeOf('event'), payload_b64: '' })).rejects.toMatchObject({ info: { code: 'signing.agent_not_found' } })
    expect(problemMessage(agentNotFoundError()).key).toBe('errors.signing.agentNotFound')
  })

  it('payloadType по шаблону контракта; содержимое — UTF-8 в base64', () => {
    expect(payloadTypeOf('event')).toBe('application/vnd.ant.event+json; v=1')
    expect(atob(toPayloadB64({ a: 1 }))).toBe('{"a":1}')
    expect(new TextDecoder().decode(Uint8Array.from(atob(toPayloadB64({ t: 'шов' })), (c) => c.charCodeAt(0)))).toBe('{"t":"шов"}')
  })
})

describe('подпись на бумаге (FR-139)', () => {
  const doc = { document_id: 'DOC-NCD-142', doc_digest: 'streebog256:c0ffee', version: 1 }

  it('печать: QR с отпечатком документа и ожидаемый подписант', async () => {
    const w = mount(PaperSignPanel, { props: { document: doc, mode: 'print', expectedSigner: 'vp-01' }, global: { plugins: [i18n] } })
    expect(w.find('[data-testid="qr-payload"]').text()).toBe('ant:doc:DOC-NCD-142:streebog256:c0ffee')
    expect(w.find('[data-testid="expected-signer"]').text()).toContain('vp-01')
    await w.find('[data-testid="print"]').trigger('click')
    expect(w.emitted('print')).toHaveLength(1)
  })

  it('заверитель ≠ подписант: подписант не может заверить свой скан', () => {
    const w = mount(PaperSignPanel, { props: { document: doc, mode: 'attest', expectedSigner: 'vp-01', currentUser: 'vp-01' }, global: { plugins: [i18n] } })
    expect(w.find('[data-testid="attester-is-signer"]').text()).toBe('Заверитель не может быть подписантом')
    expect(w.find('[data-testid="attest"]').attributes('disabled')).toBeDefined()
  })

  it('заверение: скан и учётный номер оригинала обязательны', async () => {
    const w = mount(PaperSignPanel, { props: { document: doc, mode: 'attest', expectedSigner: 'vp-01', currentUser: 'master-07' }, global: { plugins: [i18n] } })
    expect(w.find('[data-testid="attest"]').attributes('disabled')).toBeDefined()
    const file = new File(['%PDF'], 'scan.pdf', { type: 'application/pdf' })
    const input = w.find('[data-testid="scan-file"]')
    Object.defineProperty(input.element, 'files', { value: [file] })
    await input.trigger('change')
    await w.find('[data-testid="archive-no"] input').setValue('ОТК-2026-0142')
    expect(w.find('[data-testid="attest"]').attributes('disabled')).toBeUndefined()
    await w.find('[data-testid="attest"]').trigger('click')
    expect(w.emitted('attest')?.[0]?.[0]).toMatchObject({ file, archive_no: 'ОТК-2026-0142' })
  })

  it('этап запрещает бумагу — так и написано', () => {
    const w = mount(PaperSignPanel, { props: { document: doc, mode: 'print', paperAllowed: false }, global: { plugins: [i18n] } })
    expect(w.find('[data-testid="paper-forbidden"]').text()).toBe('Для этого этапа подпись на бумаге запрещена')
  })
})
