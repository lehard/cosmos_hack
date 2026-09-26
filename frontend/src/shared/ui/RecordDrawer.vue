<script setup lang="ts">
/**
 * Правое окно записи (Д-70, UI-7, UI-8) — единый способ открыть запись с любого
 * стола: стол показывает списки и обзор, щелчок по строке открывает справа это
 * окно. В нём:
 * - заголовок: тип записи, номер, статус (слот `status`) и пояснение;
 * - вкладки (карточка, паспорт изделия, история…) — прокручивается только тело;
 * - нижняя панель действий (слот `actions`) — кнопки решения и команд всегда
 *   видны, не уезжают при прокрутке.
 * Закрытие — крестик, Esc, щелчок мимо окна. Ширина — токены `--ant-w-drawer*`:
 * доля экрана между минимумом и максимумом, на узком экране — весь экран.
 *
 * Когда брать: любая запись из списка стола (несоответствие, изделие, заявка…).
 * Состояние «открыто» держит адрес (`?open=‹тип›:‹id›`, shared/model/record) —
 * окно само его не знает, только показывает и сообщает о закрытии.
 */
import { NDrawer, NDrawerContent, NSpin } from 'naive-ui'

export interface RecordDrawerTab {
  id: string
  /** Подпись вкладки (готовый текст). */
  label: string
}

withDefaults(
  defineProps<{
    show: boolean
    /** Тип записи для людей: «Несоответствие», «Изделие»… */
    kindLabel: string
    /** Номер записи для людей. */
    number?: string
    /** Пояснение под заголовком (изделие, этап…). */
    subtitle?: string
    tabs?: RecordDrawerTab[]
    /** Активная вкладка (v-model:tab). */
    tab?: string
    loading?: boolean
  }>(),
  { number: '', subtitle: '', tabs: () => [], tab: '', loading: false },
)
const emit = defineEmits<{ close: []; 'update:tab': [id: string] }>()

function onShow(v: boolean): void {
  if (!v) emit('close')
}
</script>

<template>
  <NDrawer
    :show="show"
    placement="right"
    width="min(100vw, clamp(var(--ant-w-drawer-min), 52vw, var(--ant-w-drawer)))"
    :auto-focus="false"
    close-on-esc
    mask-closable
    class="record-drawer"
    @update:show="onShow"
  >
    <NDrawerContent closable :native-scrollbar="false" class="record-drawer-content" body-content-class="record-drawer-body" footer-class="record-drawer-footer">
      <template #header>
        <div class="head ant-box" data-testid="record-drawer-head">
          <div class="head-line ant-box">
            <span class="kind ant-ellipsis" :title="kindLabel">{{ kindLabel }}</span>
            <strong v-if="number" class="number ant-ellipsis" :title="number">{{ number }}</strong>
            <span v-if="$slots.status" class="status ant-box"><slot name="status" /></span>
          </div>
          <p v-if="subtitle" class="subtitle ant-clamp-2" :title="subtitle">{{ subtitle }}</p>
          <div v-if="$slots.links" class="links ant-box"><slot name="links" /></div>
          <div v-if="tabs.length > 1" class="tabs ant-box" role="tablist">
            <button
              v-for="x in tabs"
              :key="x.id"
              type="button"
              role="tab"
              class="tab ant-ellipsis"
              :aria-selected="x.id === tab"
              :data-tab="x.id"
              :title="x.label"
              @click="emit('update:tab', x.id)"
            >
              {{ x.label }}
            </button>
          </div>
        </div>
      </template>

      <div class="body ant-box" data-testid="record-drawer-body" :data-tab="tab || undefined">
        <div v-if="loading" class="center ant-box"><NSpin size="small" /></div>
        <slot v-else />
      </div>

      <template v-if="$slots.actions" #footer>
        <div class="actions ant-box" data-testid="record-drawer-actions"><slot name="actions" /></div>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<style scoped>
.head {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
}

.head-line {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-3);
  align-items: baseline;
}

.kind {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-sm);
  font-weight: normal;
}

.number {
  max-width: 100%;
  font-size: var(--ant-fs-lg);
}

.status {
  display: inline-flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  align-items: center;
  font-size: var(--ant-fs-sm);
  font-weight: normal;
}

.subtitle {
  margin: 0;
  color: var(--ant-text-2);
  font-size: var(--ant-fs-sm);
  font-weight: normal;
}

.links {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  font-size: var(--ant-fs-sm);
  font-weight: normal;
}

.tabs {
  display: flex;
  gap: var(--ant-space-1);
  margin-bottom: calc(-1 * var(--ant-space-4));
  border-bottom: 1px solid var(--ant-border);
}

.tab {
  max-width: 280px;
  padding: var(--ant-space-2) var(--ant-space-3);
  border: 0;
  border-bottom: 2px solid transparent;
  background: none;
  color: var(--ant-text-2);
  font: inherit;
  font-size: var(--ant-fs-body);
  font-weight: normal;
  cursor: pointer;
}

.tab:hover {
  color: var(--ant-text);
}

.tab[aria-selected='true'] {
  border-bottom-color: var(--ant-accent);
  color: var(--ant-accent);
  font-weight: var(--ant-fw-bold);
}

.body {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-4);
  min-width: 0;
}

.center {
  display: flex;
  justify-content: center;
  padding: var(--ant-space-8) 0;
}

.actions {
  width: 100%;
  min-width: 0;
}
</style>

<style>
/* Нижняя панель действий: не выше половины окна, дальше — своя прокрутка
   (кнопки решения наверху панели остаются видны). */
.record-drawer-content .record-drawer-footer {
  display: block;
  max-height: 50vh;
  overflow: auto;
  padding: var(--ant-space-2) var(--ant-space-5) var(--ant-space-3);
  border-top: 1px solid var(--ant-border);
  background: var(--ant-surface-subtle);
}

.record-drawer-content .record-drawer-body {
  padding: var(--ant-space-4) var(--ant-space-5);
}
</style>
