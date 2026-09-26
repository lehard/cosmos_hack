<script setup lang="ts">
/**
 * Виджет «Партнёры и выписки» (эпик 41; FR-131, FR-132, AD-19) — контейнер:
 * `federation.partner.list`, `federation.extract.list`; щелчок по строке
 * открывает окно записи справа (Д-70). Сверху — «Принять выписку партнёра»
 * (порт межзаводского обмена: файл пакета DSSE; получатель сам проверяет
 * подписи, изменённая выписка отклоняется) и «Зарегистрировать партнёра».
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NInput, NSelect } from 'naive-ui'
import { usePartners, useExtracts, useReceiveExtract, useRegisterPartner } from '@/entities/federation'
import { useSession } from '@/entities/session'
import { useDrillDown, type DrillRef } from '@/features/drill-down'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { newCommandId } from '@/shared/lib/command-id'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton, SectionPanel, WidgetFrame } from '@/shared/ui'
import FederationView from './FederationView.vue'

defineProps<WidgetProps>()
const { t } = useI18n()
const problemText = useProblemText()
const moment = useMomentStore()
const session = useSession()
const partnersQ = usePartners()
const extractsQ = useExtracts()
const partners = computed(() => partnersQ.data.value?.data.items ?? null)
const extracts = computed(() => extractsQ.data.value?.data.items ?? null)
const drill = useDrillDown()

// Непроверенное происхождение у входящей выписки — не «норма».
const state = computed(() => (extracts.value?.some((e) => e.direction === 'incoming' && e.origin_status === 'unverified') ? 'defect_indication' : 'normal'))

/** Окна записей (типы вне EntityKind контракта — только оболочки). */
const openExtract = (digest: string) => drill.open({ entity: 'extract', id: digest } as unknown as DrillRef)
const openPartner = (code: string) => drill.open({ entity: 'partner', id: code } as unknown as DrillRef)

function meta() {
  const s = session.data.value?.data
  return { command_id: newCommandId(), basis_seq: 0, policy_seq: s?.policy_seq ?? 0, ...(s?.workplace?.id ? { workplace_id: s.workplace.id } : {}) }
}

// ── приём выписки ──
const panel = ref<'receive' | 'register' | null>(null)
const receiveM = useReceiveExtract()
const partnerCode = ref<string | null>(null)
const envelope = ref('')
const received = ref(false)
const partnerOptions = computed(() => (partners.value ?? []).map((p) => ({ label: `${p.name} (${p.partner_code})`, value: p.partner_code })))

/** Отправитель из содержимого пакета (sender): подставляет партнёра, если он ещё не выбран. */
function senderOf(text: string): string | null {
  try {
    const env = JSON.parse(text) as { payload?: string }
    const p = JSON.parse(atob(env.payload ?? '')) as { sender?: string }
    return p.sender ?? null
  } catch {
    return null
  }
}
watch(envelope, (text) => {
  const s = senderOf(text)
  if (s && !partnerCode.value && partners.value?.some((p) => p.partner_code === s)) partnerCode.value = s
})

async function readFile(ev: Event): Promise<void> {
  const f = (ev.target as HTMLInputElement).files?.[0]
  if (f) envelope.value = await f.text()
}

async function receive(): Promise<void> {
  if (!partnerCode.value || !envelope.value.trim()) return
  received.value = false
  try {
    await receiveM.mutateAsync({ ...meta(), partner_code: partnerCode.value, envelope: envelope.value })
    received.value = true
    envelope.value = ''
  } catch {
    // Текст отказа — из receiveM.error (federation.extract_tampered).
  }
}

// ── регистрация партнёра ──
const registerM = useRegisterPartner()
const reg = ref({ code: '', name: '', roots: '', endpoint: '', doc: '' })
const registered = ref(false)
const canRegister = computed(() => /^[A-Z0-9][A-Z0-9_-]{0,15}$/.test(reg.value.code) && reg.value.name.trim() && reg.value.roots.trim() && reg.value.doc.trim())

async function register(): Promise<void> {
  if (!canRegister.value) return
  registered.value = false
  try {
    await registerM.mutateAsync({
      ...meta(),
      partner_code: reg.value.code,
      name: reg.value.name.trim(),
      root_fingerprints: reg.value.roots.split(/[\s,]+/).filter(Boolean),
      document_id: reg.value.doc.trim(),
      ...(reg.value.endpoint.trim() ? { endpoint: reg.value.endpoint.trim() } : {}),
    })
    registered.value = true
    reg.value = { code: '', name: '', roots: '', endpoint: '', doc: '' }
  } catch {
    // Текст — из registerM.error.
  }
}

const toggle = (p: 'receive' | 'register') => {
  panel.value = panel.value === p ? null : p
  received.value = registered.value = false
}
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(partnersQ.data.value)"
    :state="state"
    :loading="partnersQ.isPending.value && !partners"
    :error="partners ? undefined : partnersQ.error.value"
    :data-widget="widgetId"
  >
    <div class="federation ant-box">
      <div v-if="!moment.isReplay" class="bar">
        <ActionButton size="small" data-testid="federation-receive-open" :label="t('widgets.federation.receive')" @click="toggle('receive')" />
        <ActionButton size="small" data-testid="federation-register-open" :label="t('widgets.federation.register')" @click="toggle('register')" />
      </div>

      <SectionPanel v-if="panel === 'receive'" :title="t('widgets.federation.receive')" :subtitle="t('widgets.federation.receiveHint')" data-testid="federation-receive">
        <div class="form">
          <NSelect v-model:value="partnerCode" :options="partnerOptions" :placeholder="t('widgets.federation.partner')" data-testid="receive-partner" />
          <input type="file" accept=".json,application/json" data-testid="receive-file" @change="readFile" />
          <NInput v-model:value="envelope" type="textarea" :autosize="{ minRows: 3, maxRows: 8 }" :placeholder="t('widgets.federation.envelopePlaceholder')" data-testid="receive-envelope" />
          <div>
            <ActionButton
              type="primary"
              :disabled="receiveM.isPending.value || !partnerCode || !envelope.trim()"
              data-testid="receive-submit"
              :label="t('widgets.federation.receiveSubmit')"
              @click="receive"
            />
          </div>
          <NAlert v-if="receiveM.error.value" type="error" :bordered="false" :show-icon="false" data-testid="receive-error">{{ problemText(receiveM.error.value) }}</NAlert>
          <NAlert v-else-if="received" type="success" :bordered="false" :show-icon="false" data-testid="receive-done">{{ t('widgets.federation.received') }}</NAlert>
        </div>
      </SectionPanel>

      <SectionPanel v-if="panel === 'register'" :title="t('widgets.federation.register')" :subtitle="t('widgets.federation.registerHint')" data-testid="federation-register">
        <div class="form">
          <NInput v-model:value="reg.code" :placeholder="t('widgets.federation.code')" :maxlength="16" data-testid="register-code" />
          <NInput v-model:value="reg.name" :placeholder="t('widgets.federation.name')" :maxlength="256" data-testid="register-name" />
          <NInput v-model:value="reg.roots" type="textarea" :autosize="{ minRows: 2, maxRows: 4 }" :placeholder="t('widgets.federation.roots')" data-testid="register-roots" />
          <NInput v-model:value="reg.endpoint" :placeholder="t('widgets.federation.endpoint')" :maxlength="512" />
          <NInput v-model:value="reg.doc" :placeholder="t('widgets.federation.act')" :maxlength="128" data-testid="register-doc" />
          <div>
            <ActionButton type="primary" :disabled="registerM.isPending.value || !canRegister" data-testid="register-submit" :label="t('widgets.federation.registerSubmit')" @click="register" />
          </div>
          <NAlert v-if="registerM.error.value" type="error" :bordered="false" :show-icon="false">{{ problemText(registerM.error.value) }}</NAlert>
          <NAlert v-else-if="registered" type="success" :bordered="false" :show-icon="false" data-testid="register-done">{{ t('widgets.federation.registered') }}</NAlert>
        </div>
      </SectionPanel>

      <FederationView v-if="partners" :partners="partners" :extracts="extracts ?? []" @open-extract="openExtract" @open-partner="openPartner" />
    </div>
  </WidgetFrame>
</template>

<style scoped>
.federation {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
}

.bar {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
}

.form {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  max-width: 720px;
}
</style>
