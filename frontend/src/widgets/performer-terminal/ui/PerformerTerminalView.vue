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
import { NAlert, NButton, NDivider, NEllipsis, NFlex, NInput, NRadio, NRadioGroup, NSelect, NTag, NText } from 'naive-ui'
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
import { StatusTag } from '@/shared/ui'
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
const tagSize = computed(() => (props.density === 'large' ? 'medium' : 'small'))

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
  <NFlex vertical :size="12" data-testid="terminal">
    <NFlex :size="8" align="center" :wrap="true" data-testid="workplace">
      <NText strong>{{ workplace ? t('common.header.workplace', { workplace: workplace.title }) : t('widgets.shopFloor.terminal.noWorkplace') }}</NText>
      <NText v-if="shiftTitle" depth="3">· {{ t('common.words.shift') }}: {{ shiftTitle }}</NText>
    </NFlex>
    <NText depth="3">{{ t('terminal.onlyOwnWorkplace') }}</NText>

    <NFlex v-if="warnings.length" vertical :size="6" data-testid="warnings">
      <NAlert v-for="(w, i) in warnings" :key="i" :type="w.kind === 'unusable' ? 'error' : 'warning'" :bordered="false" :data-warning="w.kind">
        {{ warningText(w) }}
      </NAlert>
    </NFlex>

    <!-- Текущая операция и «остановить». -->
    <section data-testid="current-run">
      <NDivider title-placement="left">{{ t('widgets.shopFloor.terminal.currentOperation') }}</NDivider>
      <NFlex v-if="run" vertical :size="8">
        <NFlex :size="8" align="center" :wrap="true">
          <NText strong><NEllipsis :tooltip="{ width: 360 }">{{ runStep?.name ?? run.step_key }}</NEllipsis></NText>
          <NButton text type="primary" :size="size" @click="emit('open', run.item_id)">{{ items.find((i) => i.item_id === run!.item_id)?.label ?? run.item_id }}</NButton>
          <NText v-if="runMinutes !== null" :type="runOver ? 'error' : undefined">{{ t('widgets.shopFloor.station.runFor', { time: formatMinutes(t, runMinutes) }) }}</NText>
          <NText depth="3">{{ runNorm }}</NText>
          <NTag v-if="runOver" :size="tagSize" type="error" :bordered="false">{{ t('widgets.shopFloor.station.overNorm') }}</NTag>
        </NFlex>
        <NRadioGroup v-model:value="completion" :size="size">
          <NFlex :size="16" :wrap="true">
            <NRadio value="completed">{{ t('widgets.shopFloor.terminal.completion.completed') }}</NRadio>
            <NRadio value="interrupted">{{ t('widgets.shopFloor.terminal.completion.interrupted') }}</NRadio>
          </NFlex>
        </NRadioGroup>
        <NButton type="warning" :size="size" block :disabled="actionsOff || !!run.finished_at" data-testid="stop" @click="emit('finish', completion)">
          {{ t('terminal.stopOperation') }}
        </NButton>
      </NFlex>
      <NText v-else depth="3">{{ t('widgets.shopFloor.station.noRun') }}</NText>
    </section>

    <!-- Начать операцию. -->
    <section data-testid="start">
      <NDivider title-placement="left">{{ t('terminal.startOperation') }}</NDivider>
      <NFlex vertical :size="8">
        <NSelect
          :value="stepKey"
          :options="stepOptions"
          :size="size"
          :placeholder="t('common.words.operation')"
          :consistent-menu-width="false"
          data-testid="step"
          @update:value="(v: string) => emit('update:stepKey', v)"
        />
        <NText v-if="itemsError" type="error">{{ problemText(itemsError) }}</NText>
        <NSelect
          v-else
          v-model:value="startItem"
          :options="itemOptions"
          :size="size"
          :placeholder="t('widgets.shopFloor.terminal.pickItem')"
          filterable
          :consistent-menu-width="false"
          data-testid="start-item"
        />
        <NButton type="primary" :size="size" block :disabled="actionsOff || !stepKey || !startItem || !!run" data-testid="start-operation" @click="startItem && emit('start', startItem)">
          {{ t('terminal.startOperation') }}
        </NButton>
        <NText v-if="run" depth="3">{{ t('widgets.shopFloor.terminal.finishFirst') }}</NText>
      </NFlex>
    </section>

    <!-- Изделия у поста: статус и перемещение в изолятор (FR-55). -->
    <section data-testid="items">
      <NDivider title-placement="left">{{ t('terminal.currentItems') }}</NDivider>
      <NText v-if="!items.length" depth="3">{{ t('empty.noRecords') }}</NText>
      <NFlex v-for="i in items" :key="i.item_id" vertical :size="6" :data-item="i.item_id">
        <NFlex :size="8" align="center" :wrap="true">
          <NButton text type="primary" :size="size" @click="emit('open', i.item_id)">{{ i.label }}</NButton>
          <StatusTag v-if="i.row" axis="position" :code="i.row.status.position" />
          <SummaryTag v-if="i.row && i.row.status.summary !== 'in_process'" :code="i.row.status.summary" />
        </NFlex>
        <IsolatorMoveConfirm :item-id="i.item_id" :can-act="canAct && !!workplace" :density="density" />
      </NFlex>
    </section>

    <!-- Сообщить об отклонении и запросить контроль. -->
    <section data-testid="deviation">
      <NDivider title-placement="left">{{ t('terminal.reportDeviation') }}</NDivider>
      <form @submit.prevent="submitDeviation">
        <NFlex vertical :size="8">
          <NSelect v-if="itemChoice.length" v-model:value="deviationItem" :options="itemChoice" :size="size" clearable :placeholder="t('common.words.item')" data-testid="deviation-item" />
          <NInput
            v-model:value="deviationText"
            type="textarea"
            :size="size"
            :autosize="{ minRows: 2, maxRows: 6 }"
            :maxlength="2000"
            :placeholder="t('widgets.shopFloor.terminal.deviationPlaceholder')"
            data-testid="deviation-text"
          />
          <NButton type="warning" secondary :size="size" block attr-type="submit" :disabled="actionsOff || !deviationText.trim()" data-testid="report-deviation">
            {{ t('terminal.reportDeviation') }}
          </NButton>
        </NFlex>
      </form>
    </section>

    <section data-testid="inspection">
      <NDivider title-placement="left">{{ t('terminal.requestInspection') }}</NDivider>
      <NFlex vertical :size="8">
        <NSelect v-if="itemChoice.length" v-model:value="inspectionItem" :options="itemChoice" :size="size" clearable :placeholder="t('common.words.item')" data-testid="inspection-item" />
        <NButton secondary type="primary" :size="size" block :disabled="actionsOff || !stepKey" data-testid="request-inspection" @click="emit('inspection', inspectionItem)">
          {{ t('terminal.requestInspection') }}
        </NButton>
      </NFlex>
    </section>

    <NAlert v-if="result" type="success" :bordered="false" data-testid="result">{{ result }}</NAlert>
    <NAlert v-if="error" type="error" :bordered="false" data-testid="command-error">{{ problemText(error) }}</NAlert>
  </NFlex>
</template>
