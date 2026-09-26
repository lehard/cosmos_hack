<script setup lang="ts">
/**
 * «Почему вы можете / не можете это сделать» и «Запросить решение» (FR-146,
 * FR-136). Объяснение — текст сервера (`access.permission.explain`): ваши
 * полномочия, что разрешено, что нет, чья подпись нужна. Интерфейс его не
 * сочиняет; код отказа показывается названием из каталога ошибок.
 * Если полномочий не хватает, вместо действия — «Запросить решение».
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NSpin } from 'naive-ui'
import { errorCatalog } from '@/shared/api/generated/errors'
import type { Explanation } from '@/shared/api/generated/model'
import { useProblemText } from '@/shared/i18n/problem'

const props = withDefaults(
  defineProps<{
    /** Действие доступно; null — список прав ещё не пришёл. */
    allowed: boolean | null
    /** Объяснение сервера, если его запросили. */
    explanation?: Explanation | null
    /** Объяснение раскрыто. */
    open?: boolean
    loading?: boolean
    error?: unknown
    /** Можно запросить решение вместо действия. */
    canRequest?: boolean
    /** Подписи действий из `allowed_actions`: x-ant-action id → текст. */
    actionLabels?: Record<string, string>
    /** Действия недоступны (воспроизведение). */
    disabled?: boolean
    size?: 'small' | 'medium' | 'large'
  }>(),
  { explanation: null, open: false, loading: false, error: undefined, canRequest: false, actionLabels: () => ({}), disabled: false, size: 'small' },
)
const emit = defineEmits<{
  /** Раскрыть или свернуть объяснение. */
  toggle: []
  /** Запросить решение у того, у кого есть полномочия. */
  'request-decision': []
}>()

const { t } = useI18n()
const problemText = useProblemText()

const catalog = errorCatalog as Record<string, { title: string } | undefined>
const codeTitle = computed(() => {
  const code = props.explanation?.code
  return code ? (catalog[code]?.title ?? `UNKNOWN(${code})`) : null
})
const alternatives = computed(() => (props.explanation?.allowed_actions ?? []).map((a) => props.actionLabels[a] ?? a))
/** Итог по объяснению сервера, если оно есть, иначе — по списку прав. */
const can = computed(() => props.explanation?.allowed ?? props.allowed)
</script>

<template>
  <div class="authority" :data-allowed="can === null ? 'unknown' : String(can)" data-testid="authority">
    <div class="row">
      <button v-if="can !== null" type="button" class="linklike" data-testid="why" @click="emit('toggle')">
        {{ t(can ? 'decisions.authority.whyCan' : 'decisions.authority.whyCannot') }}
      </button>
      <NButton
        v-if="can === false && canRequest"
        :size="size"
        type="primary"
        secondary
        :disabled="disabled"
        data-testid="request-decision"
        @click="emit('request-decision')"
      >
        {{ t('decisions.authority.requestDecision') }}
      </NButton>
    </div>
    <div v-if="open" class="details" data-testid="explanation">
      <NSpin v-if="loading" size="small" />
      <p v-else-if="error" class="muted">{{ problemText(error) }}</p>
      <template v-else-if="explanation">
        <p v-if="explanation.reason" class="reason">{{ explanation.reason }}</p>
        <p v-if="codeTitle" class="muted" data-testid="explanation-code">{{ codeTitle }}</p>
        <p v-if="alternatives.length" class="muted">
          {{ t('decisions.authority.allowed') }}: {{ alternatives.join(', ') }}
        </p>
      </template>
    </div>
  </div>
</template>

<style scoped>
.authority {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
}

.row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.details p {
  margin: 0;
}

.reason {
  color: #1f2937;
}

.muted {
  color: #6b7280;
}

.linklike {
  padding: 0;
  border: 0;
  background: none;
  color: #2f6fdb;
  font: inherit;
  cursor: pointer;
}

.linklike:hover {
  text-decoration: underline;
}
</style>
