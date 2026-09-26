<script setup lang="ts">
/**
 * Терминал исполнителя — представление (FR-137, FR-55; PRD §3a): своё рабочее
 * место — предупреждения оборудования («ресурс инструмента 73/75 — заменить
 * после этой детали»), текущая операция и «остановить», «начать операцию»
 * (изделие из очереди шага; ждущее контроля — «сначала контроль»), изделия у
 * поста с подтверждением перемещения в изолятор, «сообщить об отклонении»,
 * «запросить контроль». Крупно: мастеру и исполнителю — крупные кнопки.
 *
 * Действия (UI-41, Д-70): большие кнопки «начать операцию», «сообщить об
 * отклонении», «запросить контроль», «остановить операцию» — форма открывается
 * по нажатию в правом окне, кнопка отправки — внизу окна. Нет поста в сеансе —
 * это сказано прямо, действия выключены с причиной (UI-39, UI-40).
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
import { ActionButton, EmptyState, FormField, RecordDrawer, SectionPanel, StatusTag } from '@/shared/ui'
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

/** Открытая форма действия в правом окне. */
type Panel = 'start' | 'finish' | 'deviation' | 'inspection'
const panel = ref<Panel | null>(null)
const PANEL_TITLE: Record<Panel, string> = {
  start: 'terminal.startOperation',
  finish: 'terminal.stopOperation',
  deviation: 'terminal.reportDeviation',
  inspection: 'terminal.requestInspection',
}
/** Почему действия выключены — сказать прямо, а не молча серыми кнопками. */
const offReason = computed(() => (!props.workplace ? t('widgets.shopFloor.terminal.noPostActions') : !props.canAct ? t('common.modes.replayReadOnly') : null))

function submitStart(): void {
  if (!startItem.value) return
  emit('start', startItem.value)
  panel.value = null
}

function submitFinish(): void {
  emit('finish', completion.value)
  panel.value = null
}

function submitDeviation(): void {
  const text = deviationText.value.trim()
  if (!text) return
  emit('deviation', text, deviationItem.value)
  deviationText.value = ''
  panel.value = null
}

function submitInspection(): void {
  emit('inspection', inspectionItem.value)
  panel.value = null
}
</script>

<template>
  <div class="terminal" data-testid="terminal">
    <div class="line" data-testid="workplace">
      <strong class="ant-wrap">{{ workplace ? t('common.header.workplace', { workplace: workplace.title }) : t('widgets.shopFloor.terminal.noWorkplace') }}</strong>
      <span v-if="shiftTitle" class="ant-muted ant-wrap">· {{ t('common.words.shift') }}: {{ shiftTitle }}</span>
    </div>
    <p v-if="!workplace" class="no-post ant-wrap" data-testid="no-post">{{ t('widgets.shopFloor.terminal.noPostHint') }}</p>
    <p v-else class="ant-muted ant-wrap">{{ t('terminal.onlyOwnWorkplace') }}</p>

    <NAlert v-if="result" type="success" :bordered="false" data-testid="result">{{ result }}</NAlert>
    <NAlert v-if="error" type="error" :bordered="false" data-testid="command-error">{{ problemText(error) }}</NAlert>

    <div v-if="warnings.length" class="stack" data-testid="warnings">
      <NAlert v-for="(w, i) in warnings" :key="i" :type="w.kind === 'unusable' ? 'error' : 'warning'" :bordered="false" :data-warning="w.kind">
        {{ warningText(w) }}
      </NAlert>
    </div>

    <!-- Текущая операция: что идёт, сколько против нормы, большая кнопка «остановить». -->
    <SectionPanel v-if="workplace" :title="t('widgets.shopFloor.terminal.currentOperation')" variant="subtle" data-testid="current-run">
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
        <p v-if="runOver" class="ant-wrap" data-testid="over-hint">{{ t('widgets.shopFloor.terminal.overNormHint') }}</p>
        <ActionButton type="warning" :size="size" block :disabled="actionsOff || !!run.finished_at" :label="t('terminal.stopOperation')" data-testid="stop" @click="panel = 'finish'" />
      </template>
      <EmptyState v-else compact :title="t('widgets.shopFloor.station.noRun')" />
    </SectionPanel>

    <!-- Действия — большими кнопками, форма — в правом окне. -->
    <section class="actions" data-testid="actions">
      <ActionButton type="primary" :size="size" block :disabled="actionsOff || !!run" :label="t('terminal.startOperation')" data-testid="open-start" @click="panel = 'start'" />
      <ActionButton type="warning" secondary :size="size" block :disabled="actionsOff" :label="t('terminal.reportDeviation')" data-testid="open-deviation" @click="panel = 'deviation'" />
      <ActionButton type="primary" secondary :size="size" block :disabled="actionsOff || !stepKey" :label="t('terminal.requestInspection')" data-testid="open-inspection" @click="panel = 'inspection'" />
    </section>
    <p v-if="offReason" class="ant-muted ant-wrap" data-testid="off-reason">{{ offReason }}</p>
    <p v-else-if="run" class="ant-muted ant-wrap">{{ t('widgets.shopFloor.terminal.finishFirst') }}</p>

    <!-- Изделия у поста: статус и перемещение в изолятор (FR-55). -->
    <SectionPanel v-if="workplace" :title="t('terminal.currentItems')" variant="subtle" data-testid="items">
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

    <RecordDrawer :show="!!panel" :kind-label="panel ? t(PANEL_TITLE[panel]) : ''" :subtitle="workplace?.title ?? ''" data-testid="terminal-drawer" @close="panel = null">
      <div class="stack form" :data-panel="panel ?? undefined">
        <template v-if="panel === 'start'">
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
          <p v-else-if="stepKey && !candidates.length" class="ant-muted ant-wrap" data-testid="no-candidates">{{ t('widgets.shopFloor.terminal.noQueue') }}</p>
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
        </template>

        <template v-else-if="panel === 'finish' && run">
          <p class="ant-wrap"><strong>{{ runStep?.name ?? run.step_key }}</strong></p>
          <FormField :label="t('widgets.shopFloor.terminal.completion.title')">
            <NRadioGroup v-model:value="completion" :size="size" data-testid="completion">
              <NRadio value="completed">{{ t('widgets.shopFloor.terminal.completion.completed') }}</NRadio>
              <NRadio value="interrupted">{{ t('widgets.shopFloor.terminal.completion.interrupted') }}</NRadio>
            </NRadioGroup>
          </FormField>
        </template>

        <template v-else-if="panel === 'deviation'">
          <FormField v-if="itemChoice.length" :label="t('common.words.item')">
            <NSelect v-model:value="deviationItem" :options="itemChoice" :size="size" clearable :placeholder="t('common.words.item')" data-testid="deviation-item" />
          </FormField>
          <FormField :label="t('common.words.comment')" required>
            <NInput
              v-model:value="deviationText"
              type="textarea"
              :size="size"
              :autosize="{ minRows: 3, maxRows: 8 }"
              :maxlength="2000"
              :placeholder="t('widgets.shopFloor.terminal.deviationPlaceholder')"
              data-testid="deviation-text"
            />
          </FormField>
        </template>

        <template v-else-if="panel === 'inspection'">
          <FormField v-if="itemChoice.length" :label="t('common.words.item')">
            <NSelect v-model:value="inspectionItem" :options="itemChoice" :size="size" clearable :placeholder="t('common.words.item')" data-testid="inspection-item" />
          </FormField>
          <p class="ant-muted ant-wrap">{{ t('widgets.shopFloor.terminal.inspectionHint') }}</p>
        </template>
      </div>
      <template #actions>
        <ActionButton v-if="panel === 'start'" type="primary" :size="size" :disabled="actionsOff || !stepKey || !startItem || !!run" :label="t('terminal.startOperation')" data-testid="start-operation" @click="submitStart" />
        <ActionButton v-else-if="panel === 'finish'" type="warning" :size="size" :disabled="actionsOff || !run" :label="t('terminal.stopOperation')" data-testid="confirm-stop" @click="submitFinish" />
        <ActionButton v-else-if="panel === 'deviation'" type="warning" :size="size" :disabled="actionsOff || !deviationText.trim()" :label="t('terminal.reportDeviation')" data-testid="report-deviation" @click="submitDeviation" />
        <ActionButton v-else-if="panel === 'inspection'" type="primary" :size="size" :disabled="actionsOff || !stepKey" :label="t('terminal.requestInspection')" data-testid="request-inspection" @click="submitInspection" />
      </template>
    </RecordDrawer>
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

/* Большие кнопки действий — сеткой, переносятся, без горизонтальной прокрутки. */
.actions {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: var(--ant-space-3);
}

.no-post {
  padding: var(--ant-space-3) var(--ant-space-4);
  border-left: 4px solid var(--ant-status-attention);
  border-radius: 0 var(--ant-radius-md) var(--ant-radius-md) 0;
  background: var(--ant-status-attention-soft);
  color: var(--ant-status-attention-text);
  font-weight: var(--ant-fw-bold);
}

.form {
  gap: var(--ant-space-4);
}
</style>
