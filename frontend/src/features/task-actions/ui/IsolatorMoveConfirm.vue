<script setup lang="ts">
/**
 * Подтверждение физического перемещения в изолятор (FR-55, FR-137, UJ-7):
 * расхождение «изолировано в системе, физически не перемещено» видно, пока нет
 * приёмки в изоляторе; кнопка записывает приёмку (`process.movement.receive`,
 * `destination_kind = isolator`). После записи карточка перечитывается — и
 * расхождение исчезает по данным сервера, а не по догадке экрана.
 */
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NInput, NRadio, NRadioGroup, NSelect } from 'naive-ui'
import type { Density } from '@/shared/config/widget'
import { naiveSizeOf } from '@/shared/config/widget'
import { codeToKey } from '@/shared/i18n'
import { useProblemText } from '@/shared/i18n/problem'
import { newCommandId } from '@/shared/lib/command-id'
import { ActionButton, FormField } from '@/shared/ui'
import { INSPECTION_ON_RECEIPT, type IsolatorMoveDraft } from '../model/isolation'
import { useIsolatorMove } from '../model/use-isolator-move'

const props = withDefaults(
  defineProps<{
    itemId: string
    /** Можно ли действовать (не воспроизведение, есть право). */
    canAct?: boolean
    density?: Density
    /** Показывать ли «не изолировано» и «нет сведений» (иначе — только расхождение и итог). */
    verbose?: boolean
  }>(),
  { canAct: true, density: 'comfortable', verbose: false },
)
const emit = defineEmits<{ done: [seq: number] }>()

const { t } = useI18n()
const problemText = useProblemText()
const move = useIsolatorMove(() => props.itemId)
const size = computed(() => naiveSizeOf(props.density))

const draft = reactive<IsolatorMoveDraft>({ to_location_id: '', inspection_on_receipt: 'no_damage' })
/** id команды — один на намерение: повтор после ошибки уходит с тем же id (AD-7). */
const commandId = ref(newCommandId())
const recordedSeq = ref<number | null>(null)

// Изолятор по умолчанию — из решения «изолировать», иначе первый изолятор цеха.
watch(
  [move.decidedIsolator, move.isolators],
  ([decided, list]) => {
    if (!draft.to_location_id) draft.to_location_id = decided ?? list[0]?.location_id ?? ''
  },
  { immediate: true },
)

const options = computed(() => {
  const list = move.isolators.value.map((l) => ({ label: `${l.name} (${l.location_id})`, value: l.location_id }))
  const decided = move.decidedIsolator.value
  if (decided && !list.some((o) => o.value === decided)) list.unshift({ label: decided, value: decided })
  return list
})

const ready = computed(() => props.canAct && !move.busy.value && !!draft.to_location_id.trim())

async function submit(): Promise<void> {
  if (!ready.value) return
  try {
    const res = await move.confirm({ ...draft }, commandId.value)
    recordedSeq.value = res.data.seq
    commandId.value = newCommandId()
    emit('done', res.data.seq)
  } catch {
    // Текст ошибки — из move.commandError; id команды сохраняется для повтора.
  }
}
</script>

<template>
  <div class="isolator-move" :data-item="itemId" :data-isolation="move.state.value" data-testid="isolator-move">
    <template v-if="move.state.value === 'not_moved'">
      <NAlert type="warning" :bordered="false" data-testid="not-moved">
        {{ t('decisions.containment.notMovedWarning') }}: {{ move.itemLabel.value }}
      </NAlert>
      <NAlert v-if="recordedSeq !== null" type="info" :bordered="false" data-testid="receipt-pending">
        {{ t('widgets.shopFloor.isolator.pending', { seq: recordedSeq }) }}
      </NAlert>
      <form class="form" @submit.prevent="submit">
        <FormField :label="t('widgets.shopFloor.isolator.to')">
          <NSelect v-if="options.length" v-model:value="draft.to_location_id" :size="size" :options="options" :consistent-menu-width="false" data-testid="isolator" />
          <NInput v-else v-model:value="draft.to_location_id" :size="size" data-testid="isolator" />
        </FormField>
        <FormField :label="t('widgets.shopFloor.isolator.inspection')">
          <NRadioGroup v-model:value="draft.inspection_on_receipt" :size="size" data-testid="inspection-on-receipt">
            <NRadio v-for="v in INSPECTION_ON_RECEIPT" :key="v" :value="v">{{ t(`widgets.shopFloor.isolator.receipt.${codeToKey(v)}`) }}</NRadio>
          </NRadioGroup>
        </FormField>
        <ActionButton
          type="primary"
          :size="size"
          :disabled="!ready"
          :loading="move.busy.value"
          :label="t('decisions.containment.confirmIsolatorMove')"
          data-testid="confirm-isolator-move"
          @click="submit"
        />
        <NAlert v-if="move.commandError.value" type="error" :bordered="false" data-testid="command-error">
          {{ problemText(move.commandError.value) }}
        </NAlert>
      </form>
    </template>
    <NAlert v-else-if="move.state.value === 'moved' && (recordedSeq !== null || verbose)" type="success" :bordered="false" data-testid="moved">
      {{ t('timeline.operation.isolatorConfirmed') }}<template v-if="recordedSeq !== null"> · {{ t('widgets.shopFloor.recorded', { seq: recordedSeq }) }}</template>
    </NAlert>
    <p v-else-if="verbose && move.state.value === 'none'" class="ant-muted" data-testid="not-isolated">{{ t('widgets.shopFloor.isolator.notIsolated') }}</p>
    <p v-else-if="verbose && move.state.value === 'unknown' && !move.loading.value" class="ant-muted">{{ t('empty.noDataUnknown') }}</p>
  </div>
</template>

<style scoped>
.isolator-move,
.form {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
}
</style>
