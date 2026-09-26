<script setup lang="ts">
/**
 * «Назначить доп. проверку» прямо в задаче контролёра (R-01, R-02: «оценка
 * невозможна» — кадр хуже порога карты, кейс §4.5): по изделию, без
 * несоответствия — `nonconformity.recheck.request` (POST /items/{id}/recheck).
 * Метод и основание — из формы; задачу снимает сервер, когда доп. проверка
 * назначена.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NInput, NSelect } from 'naive-ui'
import { METHOD_TEXT, useDecisionCommand, type InspectionMethod } from '@/entities/nonconformity'
import { useSession } from '@/entities/session'
import type { Density } from '@/shared/config/widget'
import { naiveSizeOf } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { newCommandId } from '@/shared/lib/command-id'
import { ActionButton, FormField } from '@/shared/ui'

const props = withDefaults(defineProps<{ itemId: string; basisSeq?: number; reason?: string; density?: Density }>(), {
  basisSeq: 0,
  reason: '',
  density: 'large',
})

const { t } = useI18n()
const problemText = useProblemText()
const size = computed(() => naiveSizeOf(props.density))
const session = useSession()
const command = useDecisionCommand()

const method = ref<InspectionMethod>('radiography')
const text = ref(props.reason)
const commandId = ref(newCommandId())
const done = ref(false)
const methodOptions = computed(() => (Object.keys(METHOD_TEXT) as InspectionMethod[]).map((m) => ({ label: t(METHOD_TEXT[m]), value: m })))
const error = computed(() => (command.error.value ? problemText(command.error.value) : null))

async function submit(): Promise<void> {
  const res = await command
    .mutateAsync({
      action: 'request_recheck',
      nc_id: '',
      item_id: props.itemId,
      body: {
        command_id: commandId.value,
        basis_seq: props.basisSeq,
        policy_seq: session.data.value?.data.policy_seq ?? 0,
        method: method.value,
        reason: { text: text.value.trim() || t('taskActions.verb.recheck') },
      },
    })
    .catch(() => null)
  if (res) done.value = true
  else commandId.value = newCommandId()
}
</script>

<template>
  <div class="recheck" data-testid="task-recheck">
    <FormField :label="t('inspection.method.title')">
      <NSelect v-model:value="method" :size="size" :options="methodOptions" data-testid="recheck-method" />
    </FormField>
    <FormField :label="t('processHoldAction.reason')">
      <NInput v-model:value="text" :size="size" type="textarea" :autosize="{ minRows: 1, maxRows: 3 }" data-testid="recheck-reason" />
    </FormField>
    <NAlert v-if="error" type="error" :bordered="false">{{ error }}</NAlert>
    <ActionButton
      :size="size"
      type="primary"
      :disabled="done"
      :loading="command.isPending.value"
      :label="t('taskActions.verb.recheck')"
      data-testid="recheck-submit"
      @click="submit"
    />
  </div>
</template>

<style scoped>
.recheck {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2, 8px);
}
</style>
