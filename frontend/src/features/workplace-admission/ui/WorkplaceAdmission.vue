<script setup lang="ts">
/**
 * Допуск к рабочему месту (барьер 2, FR-83; эпик 37): открыть допуск на своём
 * посту и снять его. Сервер проверяет всё сам — зону по СКУД, роль в области
 * поста, квалификацию на дату, назначение на смену, ключ и PIN — и отвечает
 * отказом с перечнем невыполненного (`access.admission_denied`,
 * `access.not_in_zone`); отказ фиксируется в шине безопасности.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert } from 'naive-ui'
import { useSession } from '@/entities/session'
import { PRESENCE_TEXT, useAdmission, usePosts } from '@/entities/workplace'
import type { Density } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { newCommandId } from '@/shared/lib/command-id'
import { useTokenInfo, useTokenStatus } from '@/shared/lib/token-agent'

import { admissionKey, myPosts } from '../model/admission'

const props = withDefaults(defineProps<{ runId?: string; canAct?: boolean; density?: Density }>(), {
  runId: undefined,
  canAct: true,
  density: 'large',
})

const { t } = useI18n()
const problemText = useProblemText()
const session = useSession()
const postsQ = usePosts(computed(() => (props.runId ? { run_id: props.runId } : {})))
const status = useTokenStatus()
const info = useTokenInfo()
const admission = useAdmission()
const result = ref<string | null>(null)

const me = computed(() => session.data.value?.data.user.id ?? null)
const workplace = computed(() => session.data.value?.data.workplace ?? null)
const posts = computed(() => myPosts(postsQ.data.value?.data, me.value))
const key = computed(() => (me.value ? admissionKey(status.value, info.value, me.value) : null))

function meta() {
  return { command_id: newCommandId(), basis_seq: 0, policy_seq: session.data.value?.data.policy_seq ?? 0 }
}

async function admit(workplaceId: string): Promise<void> {
  result.value = null
  const res = await admission
    .mutateAsync({ kind: 'admit', workplace_id: workplaceId, body: { ...meta(), workplace_id: workplaceId, ...(key.value ?? {}) } })
    .catch(() => null)
  if (res) result.value = t('widgets.shopFloor.terminal.admission.opened', { seq: res.data.seq })
}

async function release(workplaceId: string): Promise<void> {
  result.value = null
  const res = await admission.mutateAsync({ kind: 'release', workplace_id: workplaceId, body: { ...meta(), workplace_id: workplaceId } }).catch(() => null)
  if (res) result.value = t('widgets.shopFloor.terminal.admission.released', { seq: res.data.seq })
}
</script>

<template>
  <!-- На посту — одна тихая строка; не на посту — крупно «встаньте на пост» и посты кнопками. -->
  <section class="admission" data-testid="admission">
    <p v-if="workplace" class="quiet line">
      <span>{{ t('widgets.shopFloor.terminal.admission.admitted', { workplace: workplace.title }) }}</span>
      <button type="button" class="link" :disabled="!canAct || admission.isPending.value" data-testid="admission-release" @click="release(workplace.id)">
        {{ t('widgets.shopFloor.terminal.admission.release') }}
      </button>
    </p>
    <template v-else>
      <p class="head">Встаньте на пост</p>
      <p v-if="!key" class="quiet" data-testid="admission-key">{{ t('widgets.shopFloor.terminal.admission.keyMissing') }}</p>
      <p v-if="!posts.length" class="quiet">{{ t('widgets.shopFloor.terminal.admission.noPosts') }}</p>
      <ul v-else class="cards">
        <li v-for="p in posts" :key="p.workplace_id" class="card" data-testid="admission-post">
          <p class="title">{{ p.station }}</p>
          <p class="quiet">{{ t(PRESENCE_TEXT[p.presence]) }}</p>
          <button type="button" class="go" :disabled="!canAct || admission.isPending.value" data-testid="admission-open" @click="admit(p.workplace_id)">
            {{ t('widgets.shopFloor.terminal.admission.open') }} →
          </button>
        </li>
      </ul>
      <p class="quiet" :title="t('widgets.shopFloor.terminal.admission.hint')">Допуск — по назначению мастера на смену, СКУД и ключу</p>
    </template>
    <NAlert v-if="admission.error.value" type="error" :bordered="false" data-testid="admission-error">{{ problemText(admission.error.value) }}</NAlert>
  </section>
</template>

<style scoped>
.admission {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
  min-width: 0;
}

p {
  margin: 0;
}

.head {
  font-size: var(--ant-fs-title);
}

.quiet {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.line {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  align-items: baseline;
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

.title {
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
}

.go,
.link {
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

.go:hover {
  background: var(--ant-surface-hover);
}

.link {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
}

.go:disabled,
.link:disabled {
  opacity: 0.5;
  cursor: default;
}
</style>
