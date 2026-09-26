<script setup lang="ts">
/**
 * Терминал исполнителя — представление (FR-137, FR-55; PRD §3a): своё рабочее
 * место — предупреждения оборудования («ресурс инструмента 73/75 — заменить
 * после этой детали»), текущая операция и «остановить», «начать операцию»
 * (изделие из очереди шага; ждущее контроля — «сначала контроль»), изделия у
 * поста с подтверждением перемещения в изолятор, «сообщить об отклонении»,
 * «запросить контроль». Крупно: мастеру и исполнителю — крупные кнопки.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NInput, NRadio, NRadioGroup, NSelect, NTag } from 'naive-ui'
import { SummaryTag } from '@/entities/item'
import type { ProcessStep } from '@/entities/live-map'
import type { RunProfile } from '@/entities/equipment'
import type { FinishOperationCompletion } from '@/entities/operation'
import { IsolatorMoveConfirm } from '@/features/task-actions'
import type { Density } from '@/shared/config/widget'
import { naiveSizeOf } from '@/shared/config/widget'
import { codeToKey } from '@/shared/i18n'
import { useProblemText } from '@/shared/i18n/problem'
import { formatMinutes } from '@/shared/lib/duration'
import { ActionButton, EmptyState, FormField, SectionPanel, StatusTag } from '@/shared/ui'
import type { ItemRow } from '@/entities/operation'
import type { PostItem, TerminalWarning } from '../model/terminal'

const props = withDefaults(
  defineProps<{
    workplace: { id: string; title: string } | null
    shiftTitle?: string | null
    warnings: readonly TerminalWarning[]
    run: RunProfile | null
    runStep: ProcessStep | null
    runMinutes: number | null
    runOver: boolean | null
    runNorm: string
    operations: readonly ProcessStep[]
    stepKey: string | null
    candidates: readonly { row: ItemRow; blocked: boolean }[]
    itemsError?: unknown
    items: readonly PostItem[]
    canAct?: boolean
    busy?: boolean
    error?: unknown
    result?: string | null
    density?: Density
  }>(),
  { shiftTitle: null, itemsError: undefined, canAct: true, busy: false, error: undefined, result: null, density: 'large' },
)
const emit = defineEmits<{
  'update:stepKey': [key: string]
  start: [itemId: string]
  finish: [completion: FinishOperationCompletion]
  deviation: [description: string, itemId: string | null]
  inspection: [itemId: string | null]
  open: [itemId: string]
}>()

const { t } = useI18n()
const problemText = useProblemText()
const size = computed(() => naiveSizeOf(props.density))

const stepOptions = computed(() => props.operations.map((s) => ({ label: s.operationCode ? `${s.operationCode} · ${s.name}` : s.name, value: s.stepKey })))
const itemOptions = computed(() =>
  props.candidates.map((c) => ({
    label: c.blocked ? `${c.row.label} — ${t('terminal.inspectionFirst')}` : c.row.label,
    value: c.row.item_id,
    disabled: c.blocked,
  })),
)
const startItem = ref<string | null>(null)
const completion = ref<FinishOperationCompletion>('completed')
const deviationText = ref('')
const deviationItem = ref<string | null>(null)
const inspectionItem = ref<string | null>(null)
const itemChoice = computed(() => props.items.map((i) => ({ label: i.label, value: i.item_id })))
const actionsOff = computed(() => !props.canAct || !props.workplace || props.busy)

function warningText(w: TerminalWarning): string {
  switch (w.kind) {
    case 'tool_life':
      return `${w.equipment}: ${t('terminal.toolLifeWarning', { used: w.used, limit: w.limit })}`
    case 'equipment':
      return `${w.equipment}: ${w.warning.text}`
    case 'unusable':
      return `${w.equipment}: ${t(`widgets.shopFloor.unusable.${codeToKey(w.reason)}`)}`
    case 'no_data':
      return t('widgets.shopFloor.why.equipmentNoData', { equipment: w.equipment })
  }
  return ''
}

function submitDeviation(): void {
  const text = deviationText.value.trim()
  if (!text) return
  emit('deviation', text, deviationItem.value)
  deviationText.value = ''
}
</script>

<template>
  <div class="terminal" data-testid="terminal">
    <div class="line" data-testid="workplace">
      <strong class="ant-wrap">{{ workplace ? t('common.header.workplace', { workplace: workplace.title }) : t('widgets.shopFloor.terminal.noWorkplace') }}</strong>
      <span v-if="shiftTitle" class="ant-muted ant-wrap">· {{ t('common.words.shift') }}: {{ shiftTitle }}</span>
    </div>
    <p class="ant-muted ant-wrap">{{ t('terminal.onlyOwnWorkplace') }}</p>

    <div v-if="warnings.length" class="stack" data-testid="warnings">
      <NAlert v-for="(w, i) in warnings" :key="i" :type="w.kind === 'unusable' ? 'error' : 'warning'" :bordered="false" :data-warning="w.kind">
        {{ warningText(w) }}
      </NAlert>
    </div>

    <!-- Текущая операция и «остановить». -->
    <SectionPanel :title="t('widgets.shopFloor.terminal.currentOperation')" variant="subtle" data-testid="current-run">
      <template v-if="run">
        <div class="line">
          <strong class="ant-wrap">{{ runStep?.name ?? run.step_key }}</strong>
          <ActionButton
            text
            type="primary"
            :size="size"
            :label="items.find((i) => i.item_id === run!.item_id)?.label ?? run.item_id"
            :hint="t('common.actions.openPassport')"
            @click="emit('open', run.item_id)"
          />
          <span v-if="runMinutes !== null" :class="{ over: runOver }">{{ t('widgets.shopFloor.station.runFor', { time: formatMinutes(t, runMinutes) }) }}</span>
          <span class="ant-muted">{{ runNorm }}</span>
          <NTag v-if="runOver" size="small" type="error" :bordered="false">{{ t('widgets.shopFloor.station.overNorm') }}</NTag>
        </div>
        <NRadioGroup v-model:value="completion" :size="size">
          <NRadio value="completed">{{ t('widgets.shopFloor.terminal.completion.completed') }}</NRadio>
          <NRadio value="interrupted">{{ t('widgets.shopFloor.terminal.completion.interrupted') }}</NRadio>
        </NRadioGroup>
        <ActionButton type="warning" :size="size" block :disabled="actionsOff || !!run.finished_at" :label="t('terminal.stopOperation')" data-testid="stop" @click="emit('finish', completion)" />
      </template>
      <EmptyState v-else compact :title="t('widgets.shopFloor.station.noRun')" />
    </SectionPanel>

    <!-- Начать операцию. -->
    <SectionPanel :title="t('terminal.startOperation')" variant="subtle" data-testid="start">
      <FormField :label="t('common.words.operation')">
        <NSelect
          :value="stepKey"
          :options="stepOptions"
          :size="size"
          :placeholder="t('common.words.operation')"
          :consistent-menu-width="false"
          data-testid="step"
          @update:value="(v: string) => emit('update:stepKey', v)"
        />
      </FormField>
      <NAlert v-if="itemsError" type="error" :bordered="false">{{ problemText(itemsError) }}</NAlert>
      <FormField v-else :label="t('common.words.item')" :hint="t('terminal.inspectionFirst')">
        <NSelect
          v-model:value="startItem"
          :options="itemOptions"
          :size="size"
          :placeholder="t('widgets.shopFloor.terminal.pickItem')"
          filterable
          :consistent-menu-width="false"
          data-testid="start-item"
        />
      </FormField>
      <ActionButton
        type="primary"
        :size="size"
        block
        :disabled="actionsOff || !stepKey || !startItem || !!run"
        :label="t('terminal.startOperation')"
        data-testid="start-operation"
        @click="startItem && emit('start', startItem)"
      />
      <p v-if="run" class="ant-muted ant-wrap">{{ t('widgets.shopFloor.terminal.finishFirst') }}</p>
    </SectionPanel>

    <!-- Изделия у поста: статус и перемещение в изолятор (FR-55). -->
    <SectionPanel :title="t('terminal.currentItems')" variant="subtle" data-testid="items">
      <EmptyState v-if="!items.length" compact :title="t('empty.noRecords')" />
      <div v-for="i in items" :key="i.item_id" class="stack" :data-item="i.item_id">
        <div class="line">
          <ActionButton text type="primary" :size="size" :label="i.label" :hint="t('common.actions.openPassport')" @click="emit('open', i.item_id)" />
          <StatusTag v-if="i.row" axis="position" :code="i.row.status.position" />
          <SummaryTag v-if="i.row && i.row.status.summary !== 'in_process'" :code="i.row.status.summary" />
        </div>
        <IsolatorMoveConfirm :item-id="i.item_id" :can-act="canAct && !!workplace" :density="density" />
      </div>
    </SectionPanel>

    <!-- Сообщить об отклонении и запросить контроль. -->
    <SectionPanel :title="t('terminal.reportDeviation')" variant="subtle" data-testid="deviation">
      <form class="stack" @submit.prevent="submitDeviation">
        <FormField v-if="itemChoice.length" :label="t('common.words.item')">
          <NSelect v-model:value="deviationItem" :options="itemChoice" :size="size" clearable :placeholder="t('common.words.item')" data-testid="deviation-item" />
        </FormField>
        <FormField :label="t('common.words.comment')" required>
          <NInput
            v-model:value="deviationText"
            type="textarea"
            :size="size"
            :autosize="{ minRows: 2, maxRows: 6 }"
            :maxlength="2000"
            :placeholder="t('widgets.shopFloor.terminal.deviationPlaceholder')"
            data-testid="deviation-text"
          />
        </FormField>
        <ActionButton
          type="warning"
          secondary
          :size="size"
          block
          attr-type="submit"
          :disabled="actionsOff || !deviationText.trim()"
          :label="t('terminal.reportDeviation')"
          data-testid="report-deviation"
        />
      </form>
    </SectionPanel>

    <SectionPanel :title="t('terminal.requestInspection')" variant="subtle" data-testid="inspection">
      <FormField v-if="itemChoice.length" :label="t('common.words.item')">
        <NSelect v-model:value="inspectionItem" :options="itemChoice" :size="size" clearable :placeholder="t('common.words.item')" data-testid="inspection-item" />
      </FormField>
      <ActionButton
        secondary
        type="primary"
        :size="size"
        block
        :disabled="actionsOff || !stepKey"
        :label="t('terminal.requestInspection')"
        data-testid="request-inspection"
        @click="emit('inspection', inspectionItem)"
      />
    </SectionPanel>

    <NAlert v-if="result" type="success" :bordered="false" data-testid="result">{{ result }}</NAlert>
    <NAlert v-if="error" type="error" :bordered="false" data-testid="command-error">{{ problemText(error) }}</NAlert>
  </div>
</template>

<style scoped>
.terminal,
.stack {
  display: flex;
  flex-direction: column;
  gap: var(--ant-gap);
  min-width: 0;
}

.stack {
  gap: var(--ant-space-2);
}

.line {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-2);
  align-items: center;
  min-width: 0;
}

.over {
  color: var(--ant-status-danger-text);
}
</style>
