<script setup lang="ts">
/**
 * Окно поста (Д-70): заголовок — название поста и цех; внутри — что пост знает
 * сейчас (назначен, на месте ли, текущее изделие — строка панели «Посты»),
 * назначения в текущей смене (`access.assignment.list`) и история поста —
 * записи журнала в потоке `workplace:‹id›` (назначения и снятия, ключ,
 * допуск, отклонения присутствия). Сотрудник и изделие открываются своими окнами.
 * Чтения без права у роли — честная ошибка в своём блоке, а не пустой список.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { PRESENCE_TEXT, PRESENCE_TONE, useAssignments, usePosts, useWorkplaceHistory, type JournalEntryView } from '@/entities/workplace'
import { useDrillDown } from '@/features/drill-down'
import { statusPalette } from '@/shared/api/generated/statuses'
import { statusOf } from '@/shared/api/problem'
import { eventCatalog } from '@/shared/contracts/catalog'
import { useProblemText } from '@/shared/i18n/problem'
import { DataTable, EmptyState, KeyValue, KeyValueList, RecordDrawer, SectionPanel } from '@/shared/ui'

const props = defineProps<{ id: string; show: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { t, d } = useI18n()
const route = useRoute()
const drill = useDrillDown()
const problemText = useProblemText()

const runId = computed(() => (typeof route.query.run === 'string' && route.query.run ? route.query.run : undefined))
const runParams = computed(() => (runId.value ? { run_id: runId.value } : {}))

// Строка панели «Посты»: тот же запрос, что у виджета, — кэш общий.
const postsQ = usePosts(runParams)
const posts = computed(() => postsQ.data.value?.data ?? null)
const post = computed(() => posts.value?.find((r) => r.workplace_id === props.id) ?? null)
/** Имя сотрудника по псевдониму — из той же панели; нет — псевдоним. */
const personName = (personId: string) => posts.value?.find((r) => r.assigned?.person_id === personId)?.assigned?.display ?? personId

const assignmentsQ = useAssignments(runParams)
const assignments = computed(() => (assignmentsQ.data.value?.data.items ?? []).filter((a) => a.workplace_id === props.id))

const historyQ = useWorkplaceHistory(() => props.id, runId)
const truncated = computed(() => !!historyQ.data.value?.data.next_cursor)
const catalogTitle = (type: string) => (eventCatalog as Record<string, { title: string } | undefined>)[type]?.title ?? type

interface HistoryRow {
  key: string
  at: string
  event: string
  personId: string | null
}
function eventText(e: JournalEntryView): string {
  const title = catalogTitle(e.event_type)
  const present = e.data?.present
  if (e.event_type === 'access.token.presence_changed' && typeof present === 'boolean') {
    return `${title}: ${t(present ? 'shell.record.workplace.tokenIn' : 'shell.record.workplace.tokenOut')}`
  }
  return title
}
// Новые записи сверху.
const history = computed<HistoryRow[]>(() =>
  [...(historyQ.data.value?.data.items ?? [])]
    .sort((a, b) => b.seq - a.seq)
    .map((e) => ({
      key: e.event_id,
      at: d(new Date(e.occurred_at), 'dateTime'),
      event: eventText(e),
      personId: typeof e.data?.person_id === 'string' && e.data.person_id ? e.data.person_id : null,
    })),
)
/** Операции ещё нет на сервере — история появится вместе с ней. */
const historyPending = computed(() => statusOf(historyQ.error.value) === 501)

const subtitle = computed(() => (post.value?.workshop ? `${t('common.words.workshop')}: ${post.value.workshop}` : ''))
const yesNo = (v: boolean) => t(v ? 'shell.record.workplace.yes' : 'shell.record.workplace.no')
const roleText = (role: string) => t(role === 'quality_inspector' ? 'widgets.shopFloor.shift.inspector' : 'widgets.shopFloor.shift.performer')

const openPerson = (id: string) => drill.open({ entity: 'person', id })
const openItem = (id: string) => drill.open({ entity: 'item', id })
</script>

<template>
  <RecordDrawer
    :show="show"
    :kind-label="t('common.words.post')"
    :number="post?.station ?? id"
    :subtitle="subtitle"
    :loading="postsQ.isPending.value && !posts"
    data-record="workplace"
    @close="emit('close')"
  >
    <template v-if="post" #status>
      <span class="presence ant-box" :data-presence="post.presence">
        <span class="dot" :style="{ background: statusPalette[PRESENCE_TONE[post.presence]] }" aria-hidden="true" />
        <span class="ant-ellipsis">{{ t(PRESENCE_TEXT[post.presence]) }}</span>
      </span>
    </template>

    <div class="sections ant-box">
      <SectionPanel :title="t('shell.record.workplace.now')" variant="plain" :padded="false" data-testid="workplace-now">
        <KeyValueList v-if="post">
          <KeyValue :label="t('liveMap.posts.station')" :value="post.station" />
          <KeyValue :label="t('common.words.workshop')" :value="post.workshop ?? null" />
          <KeyValue :label="t('liveMap.posts.assigned')">
            <button v-if="post.assigned" type="button" class="link ant-wrap" data-testid="workplace-person" @click="openPerson(post.assigned.person_id)">
              {{ post.assigned.display }}
            </button>
            <span v-else class="muted ant-wrap">{{ t('liveMap.posts.notAssigned') }}</span>
          </KeyValue>
          <KeyValue :label="t('liveMap.posts.presence')" :value="t(PRESENCE_TEXT[post.presence])" />
          <KeyValue :label="t('liveMap.posts.currentItem')">
            <button
              v-if="post.current_item"
              type="button"
              class="link mono ant-wrap"
              data-testid="workplace-item"
              :title="t('common.actions.openPassport')"
              @click="openItem(post.current_item.item_id)"
            >
              {{ post.current_item.label }}
            </button>
            <span v-else class="muted">—</span>
          </KeyValue>
          <KeyValue :label="t('shell.record.workplace.code')" :value="post.workplace_id" mono />
        </KeyValueList>
        <EmptyState v-else-if="postsQ.error.value" compact :title="t('errors.loadFailed')" :description="problemText(postsQ.error.value)" />
        <EmptyState v-else compact :title="t('shell.record.workplace.notInPanel')" />
      </SectionPanel>

      <SectionPanel :title="t('shell.record.workplace.shiftAssignments')" variant="plain" :padded="false" data-testid="workplace-assignments">
        <EmptyState
          v-if="assignmentsQ.error.value"
          compact
          :title="t('shell.record.workplace.assignmentsUnavailable')"
          :description="problemText(assignmentsQ.error.value)"
        />
        <EmptyState v-else-if="assignmentsQ.data.value && !assignments.length" compact :title="t('shell.record.workplace.noShiftAssignments')" />
        <DataTable v-else-if="assignments.length">
          <thead>
            <tr>
              <th>{{ t('shell.record.workplace.person') }}</th>
              <th>{{ t('shell.record.workplace.roleOnPost') }}</th>
              <th>{{ t('shell.record.workplace.admitted') }}</th>
              <th>{{ t('shell.record.workplace.qualification') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in assignments" :key="`${a.shift_id}:${a.person_id}:${a.assignee_role}`">
              <td>
                <button type="button" class="link ant-wrap" @click="openPerson(a.person_id)">{{ personName(a.person_id) }}</button>
              </td>
              <td><span class="ant-wrap">{{ roleText(a.assignee_role) }}</span></td>
              <td><span class="ant-wrap">{{ yesNo(a.admitted) }}</span></td>
              <td><span class="ant-wrap">{{ yesNo(a.qualification_ok) }}</span></td>
            </tr>
          </tbody>
        </DataTable>
      </SectionPanel>

      <SectionPanel
        :title="t('shell.record.workplace.history')"
        :subtitle="t('shell.record.workplace.historyHint')"
        variant="plain"
        :padded="false"
        data-testid="workplace-history"
      >
        <EmptyState v-if="historyPending" compact :title="t('shell.record.workplace.historyPending')" data-testid="workplace-history-pending" />
        <EmptyState
          v-else-if="historyQ.error.value"
          compact
          :title="t('shell.record.workplace.historyUnavailable')"
          :description="problemText(historyQ.error.value)"
          data-testid="workplace-history-error"
        />
        <EmptyState v-else-if="historyQ.data.value && !history.length" compact :title="t('empty.noRecords')" />
        <template v-else-if="history.length">
          <DataTable>
            <thead>
              <tr>
                <th>{{ t('common.words.time') }}</th>
                <th>{{ t('shell.record.workplace.event') }}</th>
                <th>{{ t('shell.record.workplace.person') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="h in history" :key="h.key" data-testid="workplace-history-row">
                <td><span class="ant-ellipsis" :title="h.at">{{ h.at }}</span></td>
                <td><span class="ant-wrap">{{ h.event }}</span></td>
                <td>
                  <button v-if="h.personId" type="button" class="link ant-wrap" @click="openPerson(h.personId)">{{ personName(h.personId) }}</button>
                  <span v-else class="muted">—</span>
                </td>
              </tr>
            </tbody>
          </DataTable>
          <p v-if="truncated" class="muted note ant-wrap">{{ t('shell.record.workplace.historyTruncated', { n: history.length }) }}</p>
        </template>
      </SectionPanel>
    </div>
  </RecordDrawer>
</template>

<style scoped>
.sections {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-6);
}

.presence {
  display: inline-flex;
  gap: var(--ant-space-2);
  align-items: center;
}

.dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.link {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.link:hover,
.link:focus-visible {
  text-decoration: underline;
}

.mono {
  font-family: var(--ant-font-mono);
}

.muted {
  color: var(--ant-text-3);
}

.note {
  margin: var(--ant-space-2) 0 0;
  font-size: var(--ant-fs-meta);
}
</style>
