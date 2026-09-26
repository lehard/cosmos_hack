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
/** Очередь выбранной операции — карточками. */
const queue = computed(() => props.candidates)
const currentStepName = computed(() => props.operations.find((o) => o.stepKey === props.stepKey)?.name ?? '')

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
    <!-- Не на посту — всё скажет допуск выше; здесь ничего лишнего. -->
    <template v-if="workplace">
      <p class="meta" data-testid="workplace">
        {{ t('common.header.workplace', { workplace: workplace.title }) }}<template v-if="shiftTitle"> · {{ t('common.words.shift') }}: {{ shiftTitle }}</template>
      </p>

      <NAlert v-if="error" type="error" :bordered="false" data-testid="command-error">{{ problemText(error) }}</NAlert>
      <p v-else-if="result" class="meta ok" data-testid="result">{{ result }}</p>
      <p v-for="(w, i) in warnings" :key="i" class="warn" :data-warning="w.kind" data-testid="warnings">{{ warningText(w) }}</p>

      <!-- Идёт операция: одна главная вещь и одна кнопка. -->
      <section v-if="run" class="card now" data-testid="current-run">
        <p class="quiet">Идёт</p>
        <p class="title">{{ runStep?.name ?? run.step_key }} · {{ items.find((i) => i.item_id === run!.item_id)?.label ?? run.item_id }}</p>
        <p class="quiet" :class="{ over: runOver }">
          <template v-if="runMinutes !== null">{{ t('widgets.shopFloor.station.runFor', { time: formatMinutes(t, runMinutes) }) }} · </template>{{ runNorm }}
        </p>
        <div class="row">
          <button type="button" class="go primary" :disabled="actionsOff || !!run.finished_at" data-testid="stop" @click="emit('finish', 'completed')">Выполнено</button>
          <button type="button" class="link" :disabled="actionsOff || !!run.finished_at" data-testid="interrupt" @click="panel = 'finish'">Прервать…</button>
        </div>
      </section>

      <!-- Нет операции: очередь — карточки «Начать». -->
      <template v-else>
        <p class="head">{{ queue.length ? 'Можно начинать' : 'Очередь пуста' }}</p>
        <div v-if="operations.length > 1" class="row">
          <button v-for="o in operations" :key="o.stepKey" type="button" class="tab" :data-on="o.stepKey === stepKey || undefined" @click="emit('update:stepKey', o.stepKey)">{{ o.name }}</button>
        </div>
        <NAlert v-if="itemsError" type="error" :bordered="false">{{ problemText(itemsError) }}</NAlert>
        <ul v-else-if="queue.length" class="cards" data-testid="queue">
          <li v-for="c in queue" :key="c.row.item_id" class="card" :data-blocked="c.blocked || undefined">
            <p class="title">{{ c.row.label }}</p>
            <p class="quiet">{{ currentStepName }}</p>
            <p v-if="c.blocked" class="quiet">{{ t('terminal.inspectionFirst') }}</p>
            <button v-else type="button" class="go" :disabled="actionsOff" data-testid="open-start" @click="emit('start', c.row.item_id)">{{ t('terminal.startOperation') }} →</button>
          </li>
        </ul>
      </template>

      <!-- Изделия у поста, которым нужно в изолятор (FR-55). -->
      <section v-if="items.length" class="block" data-testid="items">
        <p class="quiet">{{ t('terminal.currentItems') }}</p>
        <div v-for="i in items" :key="i.item_id" class="row" :data-item="i.item_id">
          <button type="button" class="link" @click="emit('open', i.item_id)">{{ i.label }}</button>
          <SummaryTag v-if="i.row && i.row.status.summary !== 'in_process'" :code="i.row.status.summary" />
          <IsolatorMoveConfirm :item-id="i.item_id" :can-act="canAct && !!workplace" :density="density" />
        </div>
      </section>

      <p class="row quiet" data-testid="actions">
        <button type="button" class="link" :disabled="actionsOff" data-testid="open-deviation" @click="panel = 'deviation'">{{ t('terminal.reportDeviation') }}</button>
        <span>·</span>
        <button type="button" class="link" :disabled="actionsOff || !stepKey" data-testid="open-inspection" @click="panel = 'inspection'">{{ t('terminal.requestInspection') }}</button>
      </p>
    </template>

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
.terminal {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-4);
  min-width: 0;
}

p {
  margin: 0;
}

.head {
  font-size: var(--ant-fs-title);
}

.meta,
.quiet {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.ok {
  color: var(--ant-status-success-text);
}

.warn {
  color: var(--ant-status-attention-text);
}

.over {
  color: var(--ant-status-danger-text);
}

.row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2) var(--ant-space-3);
  align-items: center;
}

.block {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
}

.cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: var(--ant-space-3);
  margin: 0;
  padding: 0;
  list-style: none;
}

.card {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  padding: var(--ant-space-3) var(--ant-space-4);
  border: 1px solid var(--ant-border);
  border-left: 4px solid var(--ant-accent);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface);
}

.card[data-blocked] {
  border-left-color: var(--ant-border);
}

.now {
  border-left-color: var(--ant-status-success);
}

.title {
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
}

.go,
.link,
.tab {
  font: inherit;
  cursor: pointer;
}

.go {
  align-self: flex-start;
  margin-top: var(--ant-space-2);
  padding: var(--ant-space-1) var(--ant-space-3);
  border: 1px solid var(--ant-accent);
  border-radius: var(--ant-radius-md);
  background: var(--ant-accent-soft);
  color: var(--ant-accent);
  font-weight: var(--ant-fw-bold);
}

.go.primary {
  background: var(--ant-accent);
  color: var(--ant-surface);
}

.go:hover {
  background: var(--ant-surface-hover);
  color: var(--ant-accent);
}

.link,
.tab {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
}

.tab {
  color: var(--ant-text-2);
}

.tab[data-on] {
  color: var(--ant-text);
  font-weight: var(--ant-fw-bold);
}

.go:disabled,
.link:disabled {
  opacity: 0.5;
  cursor: default;
}

.form {
  gap: var(--ant-space-4);
}

.stack {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
}
</style>
