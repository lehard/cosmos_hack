<script setup lang="ts">
/**
 * Реестр процессов (эпик 39; UI-11, FR-22…FR-25, UJ-4): на странице —
 * процессы (действующая версия, число версий, состояние, изделия в работе) и
 * «Создать процесс»; щелчок по строке открывает процесс в правом окне (Д-70)
 * со списком версий, модельером, отличиями и листом утверждения. Открытое
 * окно — в адресе: `?open=process:‹process_id›`, выбранная версия —
 * `&version=‹version_id›`, поэтому ссылка ведёт в то же окно.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { NButton } from 'naive-ui'
import type { WidgetProps } from '@/shared/config/widget'
import { useMomentStore } from '@/shared/model/moment'
import { DataTable, EmptyState, WidgetFrame } from '@/shared/ui'
import { useCanOnVersion } from '../model/access'
import { useProcesses } from '../model/source'
import CreateProcessDialog from './CreateProcessDialog.vue'
import ProcessDrawer from './ProcessDrawer.vue'

const props = defineProps<WidgetProps>()
const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const canOnVersion = useCanOnVersion()
const moment = useMomentStore()

const OPEN = 'process:'
const src = useProcesses()
const rows = computed(() => src.processes.value ?? [])
const creating = ref(false)

const openId = computed(() => {
  const o = route.query.open
  return typeof o === 'string' && o.startsWith(OPEN) ? o.slice(OPEN.length) : null
})
const versionId = computed(() => (typeof route.query.version === 'string' && route.query.version ? route.query.version : null))
/** Открытый процесс; только что созданного ещё нет в списке — заглушка до перечитывания. */
const opened = computed(() => {
  const id = openId.value
  if (!id) return null
  return rows.value.find((p) => p.process_id === id) ?? { process_id: id, name: id, versions: 0, status: 'draft' as const, is_default: false, items_in_work: 0 }
})

function open(processId: string | null, version: string | null = null): void {
  const query = { ...route.query }
  delete query.version
  if (processId) query.open = OPEN + processId
  else delete query.open
  if (processId && version) query.version = version
  void router.replace({ query })
}

async function onCreated(pid: string): Promise<void> {
  creating.value = false
  await src.query.refetch()
  open(pid)
}

const canCreate = computed(() => !moment.isReplay && canOnVersion('process.version.draft'))
/** Создавать нельзя — объясняем, кто создаёт и как версия вступает в силу (разделение ролей PRD, FR-22, FR-23). */
const showCreateNote = computed(() => !moment.isReplay && !canCreate.value)
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="props.density"
    :mode="src.mode.value"
    :loading="src.query.isLoading.value"
    :error="src.query.error.value ?? undefined"
    :empty="false"
    :data-widget="widgetId"
  >
    <template v-if="canCreate" #actions>
      <NButton size="small" type="primary" data-action="create" @click="creating = true">{{ t('processEditor.actions.create') }}</NButton>
    </template>
    <p v-if="showCreateNote" class="create-note ant-wrap" data-testid="create-note">{{ t('processEditor.registry.createNote') }}</p>
    <!-- Пусто — своё состояние: окно процесса и «Создать процесс» должны оставаться доступны. -->
    <EmptyState v-if="!rows.length" compact :title="t('empty.noRecords')" />
    <DataTable v-else :caption="t('processEditor.registry.title')">
      <thead>
        <tr>
          <th scope="col">{{ t('processEditor.registry.columns.name') }}</th>
          <th scope="col">{{ t('processEditor.registry.columns.active') }}</th>
          <th scope="col" class="num">{{ t('processEditor.registry.columns.versions') }}</th>
          <th scope="col">{{ t('processEditor.registry.columns.status') }}</th>
          <th scope="col" class="num">{{ t('processEditor.registry.columns.items') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="p in rows"
          :key="p.process_id"
          class="row"
          :class="{ opened: p.process_id === openId }"
          tabindex="0"
          :data-process="p.process_id"
          @click="open(p.process_id)"
          @keydown.enter="open(p.process_id)"
        >
          <td>
            <strong class="ant-wrap">{{ p.name }}</strong>
            <span v-if="p.is_default" class="tag">{{ t('processEditor.registry.default') }}</span>
          </td>
          <td>{{ p.active_version?.label ?? '—' }}</td>
          <td class="num">{{ p.versions }}</td>
          <td>{{ t(`processEditor.registry.status.${p.status}`) }}</td>
          <td class="num">{{ p.items_in_work }}</td>
        </tr>
      </tbody>
    </DataTable>
    <ProcessDrawer :process="opened" :version-id="versionId" @close="open(null)" @select-version="(v) => open(openId, v)" />
    <CreateProcessDialog :show="creating" :processes="rows" @close="creating = false" @created="onCreated" />
  </WidgetFrame>
</template>

<style scoped>
.create-note {
  margin: 0 0 var(--ant-space-3);
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

.row {
  cursor: pointer;
}

.row.opened td {
  background: var(--ant-accent-soft);
}

.num {
  text-align: right;
}

.tag {
  margin-left: var(--ant-space-2);
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}
</style>
