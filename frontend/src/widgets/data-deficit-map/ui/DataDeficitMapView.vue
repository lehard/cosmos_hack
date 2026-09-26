<script setup lang="ts">
/**
 * Каркас страницы «Карта дефицита данных» (FR-143; наполнение — эпик 42):
 * каких сведений чаще всего не хватало в расследованиях и какая цифровизация
 * сузила бы область риска.
 *
 * Сейчас из контракта доступно одно: узлы, где оценка невозможна — нет данных
 * источника (`data_gaps` счётчиков узлов, считает сервер). Сводки «нехватки
 * сведений» по расследованиям в контракте v1 нет — строки видов сведений
 * показаны с «не передано», а не с нулём (NFR-UI-4).
 */
import { useI18n } from 'vue-i18n'

defineProps<{ dataGaps: string[] }>()
const { t } = useI18n()

/** Виды недостающих сведений (тексты разбора, эпик 12). */
const MISSING = ['toolUnknown', 'cycleEndTimeUnknown', 'noObservationAfterOperation', 'noObservationBeforeOperation', 'operatorUnknown', 'equipmentLogMissing'] as const
</script>

<template>
  <div class="deficit">
    <p class="hint">{{ t('analytics.metrics.dataDeficitMap.hint') }}</p>

    <section class="section" data-section="gaps">
      <h3>{{ t('widgets.analytics.deficit.gapsTitle') }}</h3>
      <p class="note">{{ t('hints.unableToAssess') }}</p>
      <ul v-if="dataGaps.length" class="gaps" data-testid="gaps">
        <li v-for="s in dataGaps" :key="s" :data-step="s"><code>{{ s }}</code> — {{ t('inspection.outcome.unableToAssess') }}</li>
      </ul>
      <p v-else class="empty" data-testid="no-gaps">{{ t('widgets.analytics.deficit.noGaps') }}</p>
    </section>

    <section class="section" data-section="investigations">
      <h3>{{ t('widgets.analytics.deficit.investigationsTitle') }}</h3>
      <table class="table">
        <thead>
          <tr>
            <th scope="col">{{ t('widgets.analytics.deficit.what') }}</th>
            <th scope="col">{{ t('widgets.analytics.deficit.inInvestigations') }}</th>
            <th scope="col">{{ t('widgets.analytics.deficit.narrowing') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="m in MISSING" :key="m" :data-missing="m">
            <th scope="row">{{ t(`widgets.analysis.missing.${m}`) }}</th>
            <td class="absent">{{ t('widgets.analytics.notProvided') }}</td>
            <td class="absent">{{ t('widgets.analytics.notProvided') }}</td>
          </tr>
        </tbody>
      </table>
      <p class="note" data-testid="pending">{{ t('widgets.analytics.deficit.pending') }}</p>
    </section>
  </div>
</template>

<style scoped>
.deficit {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.hint,
.note,
.empty {
  margin: 0;
  color: #6b7280;
  font-size: 12px;
}

.section h3 {
  margin: 0 0 6px;
  font-size: 15px;
}

.gaps {
  margin: 6px 0 0;
  padding-left: 18px;
}

.table {
  width: 100%;
  margin-bottom: 6px;
  border-collapse: collapse;
  font-size: 13px;
}

.table th,
.table td {
  padding: 4px 8px;
  border-bottom: 1px solid #f0f1f3;
  text-align: left;
}

.table thead th {
  color: #4b5563;
  font-weight: 500;
}

.table tbody th {
  font-weight: 400;
}

.absent {
  color: #9ca3af;
}
</style>
