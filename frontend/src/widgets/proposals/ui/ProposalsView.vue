<script setup lang="ts">
/**
 * Каркас страницы «Предложения» (FR-63; наполнение — эпик 42). Предложения
 * формируют подключаемые генераторы; каждое — запись журнала с основаниями и
 * ответственным, ведёт на карту процесса; ничего не применяется автоматически.
 *
 * Операции списка предложений в контракте v1 нет — список пуст и так и
 * сказано, без выдуманных строк (FR-150: заготовок в коде интерфейса нет).
 * Уже доступный вход генератора «ограничение линии» — узел-ограничение из
 * счётчиков узлов (считает сервер) — показан как основание, не как предложение.
 */
import { useI18n } from 'vue-i18n'
import type { Bottleneck } from '@/shared/api/generated/model'

defineProps<{ bottleneck: Bottleneck | null }>()
const { t } = useI18n()

/** Генераторы предложений FR-63. */
const GENERATORS = ['bottleneck', 'riskScope', 'reactionRules', 'visionAdaptation'] as const
</script>

<template>
  <div class="proposals">
    <p class="principle" data-testid="principle">{{ t('liveMap.proposals.systemChangesNothing') }}. {{ t('widgets.analytics.proposals.principle') }}</p>

    <section class="section" data-section="list">
      <h3>{{ t('desks.proposals') }}</h3>
      <p class="empty" data-testid="no-proposals">{{ t('empty.noProposals') }}</p>
    </section>

    <section class="section" data-section="generators">
      <h3>{{ t('widgets.analytics.proposals.generators') }}</h3>
      <ul class="generators">
        <li v-for="g in GENERATORS" :key="g" :data-generator="g">
          <span class="name">{{ t(`widgets.analytics.proposals.generator.${g}`) }}</span>
          <span class="status">{{ t('widgets.analytics.proposals.notConnected') }}</span>
        </li>
      </ul>
    </section>

    <section class="section" data-section="inputs">
      <h3>{{ t('widgets.analytics.proposals.inputs') }}</h3>
      <p v-if="bottleneck" data-testid="bottleneck">
        {{ t('liveMap.bottleneck') }}: <code>{{ bottleneck.step_key }}</code
        ><template v-if="bottleneck.wait"> · {{ t('widgets.analytics.proposals.wait', { wait: bottleneck.wait }) }}</template>
      </p>
      <p v-else class="empty" data-testid="no-bottleneck">{{ t('widgets.analytics.proposals.noBottleneck') }}</p>
    </section>
  </div>
</template>

<style scoped>
.proposals {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.principle {
  margin: 0;
  padding: 8px 10px;
  border-left: 3px solid var(--ant-status-neutral);
  background: var(--ant-surface-subtle);
  font-size: var(--ant-fs-body);
}

.section h3 {
  margin: 0 0 6px;
  font-size: var(--ant-fs-title);
}

.empty {
  margin: 0;
  color: var(--ant-text-3);
}

.generators {
  margin: 0;
  padding: 0;
  list-style: none;
}

.generators li {
  display: flex;
  gap: 12px;
  justify-content: space-between;
  padding: 4px 0;
  border-bottom: 1px solid var(--ant-n-100);
}

.status {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}
</style>
