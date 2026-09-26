<script lang="ts">
/** Запись, открытая в правом окне: паспорт, пропуск брака или наблюдение. */
export interface OpenRef {
  kind: 'passport' | 'escape' | 'observation'
  id: string
}
</script>

<script setup lang="ts">
/**
 * Правое окно записи страницы «Адаптация VisionQC» (Д-70; эпик 40):
 *  - паспорт допуска — вектор версий, уровень доверия и допустимые действия,
 *    приостановка автооткатом (триггер, пояснение, основания), контроль дрейфа,
 *    история; кнопки внизу: «Вернуть в работу» (только начальник ОТК — иначе
 *    сервер отказывает с кодом analyzer.reinstate_requires_head_of_qc),
 *    «Допустить новую версию» (тень → пилот → работа), «Вывести»;
 *  - пропуск брака — ранние «признаков нет» с версиями и изделия на перепроверку (FR-100);
 *  - наблюдение — «какими версиями и почему» (FR-98).
 * До общего элемента правого окна в shared/ui — NDrawer здесь по тем же правилам.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NDrawer, NDrawerContent, NInput, NRadioButton, NRadioGroup, NSelect } from 'naive-ui'
import {
  useAnalyzerChecks,
  useAnalyzerPassport,
  useObservationAccount,
  usePassportCommand,
  type AdaptationEscape,
} from '@/entities/analyzer-passport'
import { useSession } from '@/entities/session'
import type { Density } from '@/shared/config/widget'
import { naiveSizeOf } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { newCommandId } from '@/shared/lib/command-id'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton, EmptyState, FormField, KeyValue, KeyValueList, SectionPanel } from '@/shared/ui'
import { width } from '@/shared/ui/theme/tokens'
import TrustStatus from './TrustStatus.vue'

const props = withDefaults(defineProps<{ open: OpenRef | null; runId?: string; escapes: AdaptationEscape[]; density?: Density }>(), {
  runId: undefined,
  density: 'comfortable',
})
const emit = defineEmits<{ close: []; open: [ref: OpenRef] }>()
const { t, d } = useI18n()
const problemText = useProblemText()
const moment = useMomentStore()
const session = useSession()

const passportId = computed(() => (props.open?.kind === 'passport' ? props.open.id : null))
const observationId = computed(() => (props.open?.kind === 'observation' ? props.open.id : null))
const escape = computed(() => (props.open?.kind === 'escape' ? (props.escapes.find((e) => e.event_id === props.open?.id) ?? null) : null))

const passportQ = useAnalyzerPassport(passportId, () => props.runId)
const checksQ = useAnalyzerChecks(passportId, () => props.runId)
const accountQ = useObservationAccount(observationId)
const passport = computed(() => passportQ.data.value?.data ?? null)
const checks = computed(() => checksQ.data.value?.data?.items ?? [])
const account = computed(() => accountQ.data.value?.data ?? null)

const command = usePassportCommand()
const action = ref<'reinstate' | 'admit' | 'retire' | null>(null)
const reason = ref('')
const done = ref<string | null>(null)
const admitForm = ref({ version: '', stage: 'shadow' as 'shadow' | 'pilot' | 'active', level: 0, document: '' })
watch(
  () => props.open,
  () => {
    action.value = null
    reason.value = ''
    done.value = null
    command.reset()
  },
)

const time = (iso?: string | null) => (iso ? d(new Date(iso), 'dateTime') : '—')
const share = (bp?: number | null) => (bp === undefined || bp === null ? '—' : (bp / 10000).toFixed(2).replace('.', ','))
const versionKeys = ['item_revision', 'recipe_ref', 'camera_config', 'calibration', 'analyzer_version', 'threshold_profile', 'contract_version', 'app_version']
const levels = computed(() => [0, 1, 2, 3, 4].map((n) => ({ value: n, label: t(`widgets.visionAdaptation.levels.${n}`) })))

const title = computed(() => {
  if (props.open?.kind === 'passport') return passport.value?.title ?? props.open.id
  if (props.open?.kind === 'escape') return t('widgets.visionAdaptation.escapes.one', { item: escape.value?.item_id ?? '' })
  if (props.open?.kind === 'observation') return t('widgets.visionAdaptation.observation.title')
  return ''
})

function meta() {
  const s = session.data.value?.data
  return {
    command_id: newCommandId(),
    basis_seq: passport.value?.basis_seq ?? 0,
    policy_seq: s?.policy_seq ?? 0,
    ...(s?.workplace?.id ? { workplace_id: s.workplace.id } : {}),
  }
}

async function submit(): Promise<void> {
  const p = passport.value
  if (!p || !action.value) return
  try {
    if (action.value === 'reinstate' && p.suspension) {
      await command.mutateAsync({
        kind: 'reinstate',
        passportId: p.passport_id,
        body: { ...meta(), suspension_event_id: p.suspension.event_id, reason: { text: reason.value.trim() }, ...(props.runId ? { run_id: props.runId } : {}) },
      })
    } else if (action.value === 'retire') {
      await command.mutateAsync({ kind: 'retire', passportId: p.passport_id, body: { ...meta(), reason: { text: reason.value.trim() } } })
    } else if (action.value === 'admit') {
      const f = admitForm.value
      const versions = { ...p.versions, analyzer_version: f.version.trim() }
      await command.mutateAsync({
        kind: 'admit',
        body: {
          ...meta(),
          passport_id: `${p.analyzer_id}-${f.version.trim().replace(/[^\p{L}\p{N}.-]+/gu, '-')}-${f.stage}`.slice(0, 128),
          analyzer_id: p.analyzer_id,
          stage: f.stage,
          trust_level: f.level,
          recipe_ref: p.recipe_ref,
          versions,
          previous_passport_id: p.passport_id,
          document_id: f.document.trim(),
          ...(p.analyzer_kind ? { analyzer_kind: p.analyzer_kind } : {}),
          ...(p.title ? { title: p.title } : {}),
        },
      })
    }
    done.value = action.value
    action.value = null
  } catch {
    // Текст отказа (с кодом) — из command.error.
  }
}

const canSubmit = computed(() => {
  if (command.isPending.value || moment.isReplay) return false
  if (action.value === 'admit') return !!admitForm.value.version.trim() && !!admitForm.value.document.trim()
  return !!reason.value.trim()
})

function startAdmit(): void {
  const p = passport.value
  admitForm.value = { version: p?.versions.analyzer_version ?? '', stage: 'shadow', level: 0, document: '' }
  action.value = 'admit'
}
</script>

<template>
  <NDrawer :show="!!open" :width="width.side + width.queue / 3" placement="right" @update:show="(v: boolean) => !v && emit('close')">
    <NDrawerContent :title="title" closable :native-scrollbar="false" data-testid="adaptation-drawer">
      <!-- Паспорт допуска -->
      <template v-if="open?.kind === 'passport'">
        <p v-if="passportQ.error.value" class="error">{{ problemText(passportQ.error.value) }}</p>
        <div v-else-if="passport" class="body" :data-passport="passport.passport_id">
          <div class="head">
            <TrustStatus :status="passport.status" :stage="passport.stage" :level="passport.trust_level" />
            <span class="muted ant-mono ant-wrap">{{ passport.passport_id }}</span>
          </div>
          <NAlert v-if="passport.suspension" type="error" :bordered="false" :show-icon="false" data-testid="suspension">
            <strong class="ant-wrap">{{ t(`widgets.visionAdaptation.trigger.${passport.suspension.trigger}`) }} · {{ time(passport.suspension.at) }}</strong>
            <p v-if="passport.suspension.note" class="ant-wrap">{{ passport.suspension.note }}</p>
            <!-- Что сделала система сама — списком (разбор роли администратора). -->
            <p class="done-title">{{ t('widgets.visionAdaptation.systemDid.title') }}</p>
            <ul class="done" data-testid="system-did">
              <li>✓ {{ t('widgets.visionAdaptation.systemDid.stopped') }}</li>
              <li>✓ {{ t(`widgets.visionAdaptation.fallback.${passport.suspension.fallback}`, { id: passport.suspension.fallback_passport_id ?? '' }) }}</li>
              <li v-if="passport.suspension.trigger === 'escape_detected'">✓ {{ t('widgets.visionAdaptation.systemDid.recheck') }}</li>
            </ul>
            <p class="ant-wrap" data-testid="only-head-of-qc">🔒 {{ t('widgets.visionAdaptation.onlyHeadOfQc') }}</p>
          </NAlert>
          <p v-if="passport.provenance === 'genesis'" class="muted ant-wrap">{{ t('widgets.visionAdaptation.genesis') }}</p>
          <SectionPanel variant="subtle" :title="t('widgets.visionAdaptation.versions')">
            <KeyValueList>
              <KeyValue v-for="k in versionKeys" :key="k" :label="t(`widgets.visionAdaptation.vector.${k}`)" :value="passport.versions[k] ?? null" mono />
            </KeyValueList>
          </SectionPanel>
          <SectionPanel variant="subtle" :title="t('widgets.visionAdaptation.allowed')">
            <p class="ant-wrap">{{ passport.allowed_auto_actions.map((a) => t(`widgets.visionAdaptation.actions.${a}`)).join(' · ') }}</p>
            <p class="muted ant-wrap">{{ t('widgets.visionAdaptation.forbidden') }}</p>
          </SectionPanel>
          <SectionPanel v-if="passport.monitor" variant="subtle" :title="t('widgets.visionAdaptation.monitor.title')">
            <p class="ant-wrap">{{ t('widgets.visionAdaptation.monitor.rule', { n: passport.monitor.window, q: share(passport.monitor.quality_min_bp) }) }}</p>
            <p class="ant-wrap" data-testid="monitor">
              {{ t('widgets.visionAdaptation.monitor.recent') }}:
              <template v-if="passport.monitor.recent_quality_bp.length">{{ passport.monitor.recent_quality_bp.map(share).join(' · ') }}</template>
              <template v-else>—</template>
            </p>
          </SectionPanel>
          <SectionPanel v-if="checks.length" variant="subtle" :title="t('widgets.visionAdaptation.checks')">
            <ul class="list">
              <li v-for="c in checks" :key="c.event_id" class="ant-wrap">
                {{ t(`widgets.visionAdaptation.checkKind.${c.check_kind}`) }} · {{ c.passed ? t('widgets.visionAdaptation.passed') : t('widgets.visionAdaptation.failed') }} ·
                {{ t('widgets.visionAdaptation.rates', { escape: share(c.escape_rate_bp), alarm: share(c.false_alarm_rate_bp), disagree: share(c.disagreement_rate_bp) }) }}
              </li>
            </ul>
          </SectionPanel>
          <SectionPanel v-if="passport.history?.length" variant="subtle" :title="t('widgets.visionAdaptation.history')">
            <ul class="list">
              <li v-for="h in passport.history" :key="h.event_id" class="ant-wrap">
                {{ time(h.at) }} · {{ t(`statuses.analyzerPassport.${h.status}`) }}<template v-if="h.trigger"> · {{ t(`widgets.visionAdaptation.trigger.${h.trigger}`) }}</template>
              </li>
            </ul>
          </SectionPanel>
          <form v-if="action" class="form" @submit.prevent="submit">
            <template v-if="action === 'admit'">
              <FormField :label="t('widgets.visionAdaptation.admit.version')">
                <NInput v-model:value="admitForm.version" :size="naiveSizeOf(density)" data-testid="admit-version" />
              </FormField>
              <FormField :label="t('widgets.visionAdaptation.admit.stage')" :hint="t('widgets.visionAdaptation.admit.stageHint')">
                <NRadioGroup v-model:value="admitForm.stage" :size="naiveSizeOf(density)">
                  <NRadioButton value="shadow">{{ t('statuses.analyzerPassport.shadow') }}</NRadioButton>
                  <NRadioButton value="pilot">{{ t('statuses.analyzerPassport.pilot') }}</NRadioButton>
                  <NRadioButton value="active">{{ t('statuses.analyzerPassport.active') }}</NRadioButton>
                </NRadioGroup>
              </FormField>
              <FormField :label="t('widgets.visionAdaptation.admit.level')">
                <NSelect v-model:value="admitForm.level" :options="levels" :size="naiveSizeOf(density)" />
              </FormField>
              <FormField :label="t('widgets.visionAdaptation.admit.document')" :hint="t('widgets.visionAdaptation.admit.documentHint')">
                <NInput v-model:value="admitForm.document" :size="naiveSizeOf(density)" data-testid="admit-document" />
              </FormField>
            </template>
            <FormField v-else :label="t('widgets.visionAdaptation.reason')">
              <NInput v-model:value="reason" type="textarea" :autosize="{ minRows: 2 }" :size="naiveSizeOf(density)" data-testid="reason" />
            </FormField>
          </form>
          <p v-if="done" class="ok ant-wrap" data-testid="done">{{ t(`widgets.visionAdaptation.done.${done}`) }}</p>
          <NAlert v-if="command.error.value" type="error" :bordered="false" :show-icon="false" data-testid="command-error">
            <span class="ant-wrap">{{ problemText(command.error.value) }}</span>
          </NAlert>
        </div>
      </template>

      <!-- Пропуск брака -->
      <template v-else-if="open?.kind === 'escape'">
        <EmptyState v-if="!escape" compact :title="t('widgets.visionAdaptation.escapes.gone')" />
        <div v-else class="body" data-testid="escape">
          <KeyValueList>
            <KeyValue :label="t('widgets.visionAdaptation.col.item')" :value="escape.item_id" mono />
            <KeyValue :label="t('widgets.visionAdaptation.escapes.defect')" :value="escape.defect_id" mono />
            <KeyValue :label="t('widgets.visionAdaptation.col.version')" :value="escape.analyzer_version ?? null" mono />
            <KeyValue :label="t('widgets.visionAdaptation.escapes.found')" :value="time(escape.recorded_at)" />
          </KeyValueList>
          <p v-if="!escape.method_covers_defect" class="muted ant-wrap">{{ t('widgets.visionAdaptation.escapes.methodBlindLong') }}</p>
          <SectionPanel variant="subtle" :title="t('widgets.visionAdaptation.escapes.missed')">
            <ul class="list">
              <li v-for="m in escape.missed" :key="m.event_id">
                <button type="button" class="link ant-wrap" @click="emit('open', { kind: 'observation', id: m.event_id })">
                  {{ m.point ?? '—' }} · {{ time(m.occurred_at) }} · {{ m.versions.analyzer_version ?? '—' }}
                </button>
              </li>
            </ul>
          </SectionPanel>
          <SectionPanel variant="subtle" :title="t('widgets.visionAdaptation.escapes.recheckTitle')" :subtitle="t('widgets.visionAdaptation.escapes.recheckHint')">
            <EmptyState v-if="!escape.recheck.length" compact :title="t('widgets.visionAdaptation.escapes.recheckNone')" />
            <ul v-else class="list" data-testid="recheck">
              <li v-for="r in escape.recheck" :key="r.item_id" class="ant-wrap">
                <span class="ant-mono">{{ r.item_id }}</span> · {{ time(r.last_at) }}
              </li>
            </ul>
          </SectionPanel>
        </div>
      </template>

      <!-- Наблюдение: какими версиями и почему -->
      <template v-else-if="open?.kind === 'observation'">
        <p v-if="accountQ.error.value" class="error">{{ problemText(accountQ.error.value) }}</p>
        <div v-else-if="account" class="body" data-testid="account">
          <KeyValueList>
            <KeyValue :label="t('widgets.visionAdaptation.col.item')" :value="account.item_id ?? null" mono />
            <KeyValue :label="t('widgets.visionAdaptation.observation.at')" :value="time(account.occurred_at)" />
            <KeyValue :label="t('widgets.visionAdaptation.observation.passport')" :value="account.passport_id ?? null" mono />
            <KeyValue :label="t('widgets.visionAdaptation.observation.levelThen')" :value="account.level_then" />
          </KeyValueList>
          <SectionPanel variant="subtle" :title="t('widgets.visionAdaptation.observation.why')">
            <ol class="list">
              <li v-for="(r, i) in account.reasons" :key="i" class="ant-wrap">{{ r }}</li>
            </ol>
          </SectionPanel>
          <SectionPanel variant="subtle" :title="t('widgets.visionAdaptation.versions')">
            <KeyValueList>
              <KeyValue v-for="k in versionKeys" :key="k" :label="t(`widgets.visionAdaptation.vector.${k}`)" :value="account.versions[k] ?? null" mono />
            </KeyValueList>
          </SectionPanel>
        </div>
      </template>

      <template v-if="open?.kind === 'passport' && passport" #footer>
        <div class="footer">
          <template v-if="action">
            <ActionButton :label="t('widgets.visionAdaptation.cancel')" :size="naiveSizeOf(density)" @click="action = null" />
            <ActionButton
              type="primary"
              :label="t(`widgets.visionAdaptation.confirm.${action}`)"
              :size="naiveSizeOf(density)"
              :disabled="!canSubmit"
              :loading="command.isPending.value"
              data-testid="submit"
              @click="submit"
            />
          </template>
          <template v-else>
            <ActionButton
              v-if="passport.suspension"
              type="primary"
              :label="t('widgets.visionAdaptation.reinstate')"
              :size="naiveSizeOf(density)"
              :disabled="moment.isReplay"
              data-testid="reinstate"
              @click="action = 'reinstate'"
            />
            <ActionButton :label="t('widgets.visionAdaptation.admitNew')" :size="naiveSizeOf(density)" :disabled="moment.isReplay" data-testid="admit" @click="startAdmit" />
            <ActionButton
              v-if="passport.status !== 'retired'"
              :label="t('widgets.visionAdaptation.retire')"
              :size="naiveSizeOf(density)"
              :disabled="moment.isReplay"
              data-testid="retire"
              @click="action = 'retire'"
            />
          </template>
        </div>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<style scoped>
.body {
  display: flex;
  flex-direction: column;
  gap: var(--ant-gap);
  font-size: var(--ant-fs-body);
}

.head {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  align-items: center;
}

.list {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  margin: 0;
  padding-left: var(--ant-space-5);
}

.form {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
}

.footer {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  justify-content: flex-end;
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

p {
  margin: 0;
}

.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.error {
  color: var(--ant-status-danger);
}

.ok {
  color: var(--ant-status-success);
}
.done-title {
  margin: var(--ant-space-2) 0 0;
  font-weight: var(--ant-fw-bold);
}

.done {
  margin: 0;
  padding: 0;
  list-style: none;
}
</style>
