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
import { naiveSizeOf } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { newCommandId } from '@/shared/lib/command-id'
import { useTokenInfo, useTokenStatus } from '@/shared/lib/token-agent'
import { ActionButton, EmptyState, SectionPanel } from '@/shared/ui'
import { admissionKey, myPosts } from '../model/admission'

const props = withDefaults(defineProps<{ runId?: string; canAct?: boolean; density?: Density }>(), {
  runId: undefined,
  canAct: true,
  density: 'large',
})

const { t } = useI18n()
const problemText = useProblemText()
const size = computed(() => naiveSizeOf(props.density))
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
  <SectionPanel :title="t('widgets.shopFloor.terminal.admission.title')" :subtitle="t('widgets.shopFloor.terminal.admission.hint')" variant="subtle" data-testid="admission">
    <div v-if="workplace" class="line">
      <strong class="ant-wrap">{{ t('widgets.shopFloor.terminal.admission.admitted', { workplace: workplace.title }) }}</strong>
      <ActionButton
        :size="size"
        :disabled="!canAct || admission.isPending.value"
        :label="t('widgets.shopFloor.terminal.admission.release')"
        data-testid="admission-release"
        @click="release(workplace.id)"
      />
    </div>
    <template v-else>
      <NAlert v-if="!key" type="info" :bordered="false" data-testid="admission-key">{{ t('widgets.shopFloor.terminal.admission.keyMissing') }}</NAlert>
      <EmptyState v-if="!posts.length" compact :title="t('widgets.shopFloor.terminal.admission.noPosts')" />
      <div v-for="p in posts" :key="p.workplace_id" class="line" data-testid="admission-post">
        <span class="ant-wrap">{{ p.station }}</span>
        <span class="ant-muted">{{ t(PRESENCE_TEXT[p.presence]) }}</span>
        <ActionButton
          type="primary"
          :size="size"
          :disabled="!canAct || admission.isPending.value"
          :label="t('widgets.shopFloor.terminal.admission.open')"
          data-testid="admission-open"
          @click="admit(p.workplace_id)"
        />
      </div>
    </template>
    <NAlert v-if="admission.error.value" type="error" :bordered="false" data-testid="admission-error">{{ problemText(admission.error.value) }}</NAlert>
    <NAlert v-else-if="result" type="success" :bordered="false">{{ result }}</NAlert>
  </SectionPanel>
</template>

<style scoped>
.line {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
</style>
