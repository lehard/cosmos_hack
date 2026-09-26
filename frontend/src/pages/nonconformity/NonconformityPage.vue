<script setup lang="ts">
/**
 * Карточка несоответствия на всю страницу (общий экран PRD §3a, FR-7): те же
 * виджеты nc-card и decision-panel, что на столе контролёра, со срезом `nc_id`,
 * и компактный паспорт изделия справа. Сюда ведут ссылки «несоответствие» с
 * живой карты, из ленты тревог и из паспорта (маршрут `nonconformity`).
 */
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useNcCard } from '@/entities/nonconformity'
import WidgetHost from '@/widgets/WidgetHost.vue'

const route = useRoute()
const ncId = computed(() => String(route.params.id ?? ''))
const card = useNcCard(ncId)
const itemId = computed(() => card.data.value?.data?.item_id ?? '')
const slice = computed(() => ({ nc_id: ncId.value, view: 'full' }))
const passportSlice = computed(() => ({ item_id: itemId.value, view: 'compact' }))
</script>

<template>
  <div class="nc-page">
    <div class="main">
      <WidgetHost widget="nc-card" slot-id="page-card" :slice="slice" density="comfortable" />
      <WidgetHost widget="decision-panel" slot-id="page-decision" :slice="slice" density="comfortable" />
    </div>
    <aside class="side">
      <WidgetHost v-if="itemId" widget="item-passport" slot-id="page-passport" :slice="passportSlice" density="comfortable" />
    </aside>
  </div>
</template>

<style scoped>
.nc-page {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(320px, 400px);
  gap: 16px;
  align-items: start;
}

.main,
.side {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

@media (max-width: 1279px) {
  .nc-page {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
