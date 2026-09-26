<script setup lang="ts">
/**
 * Носители идентификатора (AD-16): тип, значение, временный ли, состояние
 * (нанесён / проверен / нечитаем / снят / заменён), когда нанесён и снят.
 * Идентичность изделия от носителя отделена: носитель можно заменить.
 */
import { useI18n } from 'vue-i18n'
import { CARRIER_STATE_TEXT, CARRIER_TYPE_TEXT, codeText, type ItemCarrier } from '@/entities/item'

defineProps<{ carriers: ItemCarrier[] }>()
const { t, d } = useI18n()
const time = (x: string) => d(new Date(x), 'dateTime')
</script>

<template>
  <section class="carriers" data-testid="passport-carriers">
    <p v-if="!carriers.length" class="muted">{{ t('empty.noRecords') }}</p>
    <ul v-else>
      <li v-for="c in carriers" :key="`${c.carrier_type}:${c.value}`" :data-state="c.state">
        <span>{{ codeText(CARRIER_TYPE_TEXT, c.carrier_type, t) }}</span>
        <code>{{ c.value }}</code>
        <span class="status">{{ codeText(CARRIER_STATE_TEXT, c.state, t) }}</span>
        <span v-if="c.temporary" class="muted">{{ t('widgets.passport.carrier.temporary') }}</span>
        <span class="muted">{{ time(c.applied_at) }}<template v-if="c.removed_at"> — {{ time(c.removed_at) }}</template></span>
      </li>
    </ul>
  </section>
</template>

<style scoped>
ul {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0;
  padding: 0;
  list-style: none;
}

li {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 10px;
  align-items: baseline;
}

.status {
  padding: 0 6px;
  border-radius: 8px;
  background: #f3f4f6;
  font-size: 12px;
}

.muted {
  margin: 0;
  color: #6b7280;
  font-size: 12px;
}
</style>
