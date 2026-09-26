<script setup lang="ts">
/**
 * Виджет «Посты» (FR-6) — контейнер: читает посты через entities/workplace на
 * момент из useMomentStore; вынутый ключ приходит событием SSE `workplace` и
 * перечитывает панель (проверка FR-6: не позднее 2 с). Срез: `workshop` — цех,
 * `run_id` — прогон сценария.
 * Щелчок по посту, назначенному сотруднику, изделию — правое окно записи
 * (Д-70): `workplace`, `person`, `item`; нет окна поста или сотрудника — текст.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePosts } from '@/entities/workplace'
import { useDrillDown } from '@/features/drill-down'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { WidgetFrame } from '@/shared/ui'
import { postsState } from '../model/presence'
import PostsTable from './PostsTable.vue'

const props = defineProps<WidgetProps>()
const drill = useDrillDown()

const str = (v: unknown) => (typeof v === 'string' && v ? v : undefined)
const query = usePosts(
  computed(() => {
    const p: { workshop?: string; run_id?: string } = {}
    const workshop = str(props.slice.workshop)
    const run = str(props.slice.run_id)
    if (workshop) p.workshop = workshop
    if (run) p.run_id = run
    return p
  }),
)
const rows = computed(() => query.data.value?.data ?? null)
const { t } = useI18n()

/**
 * Сначала исключения (руководителю — только проблемы ресурсов): ключ не вставлен,
 * человек отсутствует, на посту никого, а изделие ждёт; остальные — «Показать все».
 * Включается срезом стола `exceptions_first: true` (стол руководителя).
 */
const PROBLEM = new Set(['key_missing', 'owner_absent', 'absent'])
const isProblem = (r: NonNullable<typeof rows.value>[number]) => PROBLEM.has(r.presence) || (r.presence === 'not_assigned' && Boolean(r.current_item))
const problems = computed(() => (rows.value ?? []).filter(isProblem))
const showAll = ref(props.slice.exceptions_first !== true)
const shown = computed(() => (showAll.value ? rows.value : problems.value))
// Id не важен: окно ставится на вид записи целиком.
const canOpenPost = computed(() => drill.canOpen({ entity: 'workplace', id: '-' }))
const canOpenPerson = computed(() => drill.canOpen({ entity: 'person', id: '-' }))
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(query.data.value)"
    :state="postsState(rows)"
    :loading="query.isPending.value && !rows"
    :error="rows ? undefined : query.error.value"
    :empty="!!rows && !rows.length"
    :data-widget="widgetId"
  >
    <p v-if="rows" class="summary" data-testid="posts-summary">
      {{ problems.length ? t('widgets.posts.problems', { n: problems.length }) : t('widgets.posts.allOk') }}
      <button v-if="rows.length > problems.length" type="button" class="toggle" data-testid="posts-toggle" @click="showAll = !showAll">
        {{ showAll ? t('widgets.posts.onlyProblems') : t('widgets.posts.showAll', { n: rows.length }) }}
      </button>
    </p>
    <PostsTable
      v-if="shown && shown.length"
      :rows="shown"
      :can-open-post="canOpenPost"
      :can-open-person="canOpenPerson"
      @open-post="(id) => drill.open({ entity: 'workplace', id })"
      @open-person="(id) => drill.open({ entity: 'person', id })"
      @open-item="(id) => drill.open({ entity: 'item', id })"
    />
  </WidgetFrame>
</template>

<style scoped>
.summary {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2) var(--ant-space-5);
  align-items: baseline;
  margin: 0 0 var(--ant-space-4);
  font-weight: var(--ant-fw-bold);
}

.toggle {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  font: inherit;
  font-weight: normal;
  cursor: pointer;
}
</style>
