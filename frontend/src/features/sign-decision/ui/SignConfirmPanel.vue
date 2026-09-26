<script setup lang="ts">
/**
 * Окно подтверждения уровня 2 — заглушка за портом подписи (FR-66, PRD §11.16;
 * AD-13, AD-14). Закрывающее решение: сводка из 3–7 полей и явное
 * подтверждение касанием токена. Доверенная сводка и касание — в окне агента
 * токена (AD-21: «окно подтверждения уровня 2 — в расширении агента»); здесь —
 * копия «проверьте перед подписью» и выбор пути: агент или бумага (FR-139).
 * Без агента и без разрешённой бумаги решение не подписывается
 * (`signing.no_signature_path`).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert } from 'naive-ui'
import type { SummaryField } from '@/entities/document'
import type { TokenStatus } from '@/shared/lib/token-agent'
import { useProblemText } from '@/shared/i18n/problem'
import { tokenReady } from '../model/port'
import { ActionButton } from '@/shared/ui'

const props = withDefaults(
  defineProps<{
    /** Сводка решения. */
    summary: SummaryField[]
    /** Состояние токена на рабочем месте. */
    tokenStatus: TokenStatus
    /** Этап маршрута разрешает бумагу с заверением. */
    paperAllowed?: boolean
    /** Идёт подпись или отправка. */
    busy?: boolean
    /** Ошибка подписи или команды. */
    error?: unknown
    /** Сколько объектов в пачке (одно окно, одно касание, AD-13); 1 — одиночная подпись. */
    batch?: number
    /**
     * Демонстрационный профиль без агента токена: команда уходит без подписи
     * агента, в паспорте её подпись — «не проверялась» (не «действительна»).
     */
    demoUnsigned?: boolean
  }>(),
  { paperAllowed: true, busy: false, error: undefined, batch: 1, demoUnsigned: false },
)
const emit = defineEmits<{
  /** Подписать агентом токена. */
  'confirm-token': []
  /** Подписать на бумаге. */
  'sign-paper': []
  /** Демо: подтвердить без подписи агента. */
  'confirm-unsigned': []
  cancel: []
}>()

const { t } = useI18n()
const problemText = useProblemText()

const value = (f: SummaryField) => f.value ?? (f.valueKey ? t(f.valueKey, f.valueParams ?? {}) : t('common.words.unknown'))
const ready = computed(() => tokenReady(props.tokenStatus))
/** Нет ни агента, ни бумаги — подписать нечем. */
const noPath = computed(() => !ready.value && !props.paperAllowed && !props.demoUnsigned)
const tokenNote = computed(() =>
  props.tokenStatus === 'missing' ? t('errors.signing.tokenMissing') : props.tokenStatus === 'agent_missing' ? t('common.header.tokenAgentMissing') : null,
)
</script>

<template>
  <section class="sign-confirm" role="dialog" :aria-label="t('decisions.signature.summaryTitle')" data-testid="sign-confirm">
    <h3>{{ t('decisions.signature.summaryTitle') }}</h3>
    <p class="muted">{{ t('decisions.signature.level2') }}</p>
    <p v-if="batch > 1" class="muted" data-testid="batch">{{ t('widgets.signing.batch', { n: batch }) }}</p>

    <dl class="summary" data-testid="summary">
      <template v-for="f in summary" :key="f.labelKey">
        <dt>{{ t(f.labelKey) }}</dt>
        <dd>{{ value(f) }}</dd>
      </template>
    </dl>

    <p class="muted" :title="t('hints.tokenAgent')">{{ t('widgets.signing.trustedWindow') }}</p>
    <p v-if="tokenNote" class="muted" data-testid="token-note">{{ tokenNote }}</p>
    <p v-if="demoUnsigned && !ready" class="muted" data-testid="demo-note">{{ t('widgets.signing.demoUnsignedNote') }}</p>
    <NAlert v-if="noPath" type="error" :bordered="false" :show-icon="false" data-testid="no-path">{{ t('errors.signing.noSignaturePath') }}</NAlert>
    <NAlert v-if="error" type="error" :bordered="false" :show-icon="false" data-testid="sign-error">{{ problemText(error) }}</NAlert>

    <footer class="buttons">
      <ActionButton overflow="wrap" type="primary" :disabled="!ready || busy" :loading="busy" data-testid="confirm-token" @click="emit('confirm-token')" :label="t('decisions.signature.confirmWithToken')" />
      <ActionButton overflow="wrap" v-if="demoUnsigned && !ready" type="primary" secondary :disabled="busy" :loading="busy" data-testid="confirm-unsigned" @click="emit('confirm-unsigned')" :label="t('widgets.signing.demoUnsigned')" />
      <ActionButton overflow="wrap" v-if="paperAllowed" :disabled="busy" data-testid="sign-paper" @click="emit('sign-paper')" :label="t('decisions.signature.signOnPaper')" />
      <ActionButton overflow="wrap" quaternary :disabled="busy" data-testid="cancel" @click="emit('cancel')" :label="t('common.actions.cancel')" />
    </footer>
  </section>
</template>

<style scoped>
.sign-confirm {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

h3 {
  margin: 0;
  font-size: var(--ant-fs-lg);
}

.muted {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.summary {
  display: grid;
  grid-template-columns: max-content minmax(0, 1fr);
  gap: 4px 12px;
  margin: 0;
  padding: 8px 10px;
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
}

.summary dt {
  color: var(--ant-text-3);
}

.summary dd {
  margin: 0;
  font-weight: var(--ant-fw-bold);
}

.buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
</style>
