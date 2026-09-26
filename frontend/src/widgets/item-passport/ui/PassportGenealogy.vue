<script setup lang="ts">
/**
 * Генеалогия изделия (FR-45): куда входит, из чего состоит, партии и выписки
 * паспортов партнёров. «Происхождение не подтверждено» у выписки показывается
 * как есть (AD-19). Экземпляр открывается своим паспортом.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { SummaryTag, splitGenealogy, type GenealogyNode, type ItemGenealogy } from '@/entities/item'

const props = defineProps<{ genealogy: ItemGenealogy }>()
const emit = defineEmits<{ 'open-item': [itemId: string] }>()
const { t } = useI18n()

const view = computed(() => splitGenealogy(props.genealogy))
const sections = computed(() => [
  { key: 'up', title: 'passport.genealogyUp', nodes: view.value.up },
  { key: 'down', title: 'passport.genealogyDown', nodes: view.value.down },
  { key: 'lots', title: 'widgets.passport.genealogy.lots', nodes: view.value.lots },
  { key: 'extracts', title: 'passport.supplierExtract', nodes: view.value.extracts },
])
const openable = (n: GenealogyNode) => n.kind === 'item'
</script>

<template>
  <section class="genealogy" data-testid="passport-genealogy">
    <div v-for="s in sections" :key="s.key" class="block" :data-section="s.key">
      <h5>{{ t(s.title) }}</h5>
      <p v-if="!s.nodes.length" class="muted">{{ t('empty.noRecords') }}</p>
      <ul v-else>
        <li v-for="n in s.nodes" :key="n.ref" :data-ref="n.ref" :data-kind="n.kind">
          <button v-if="openable(n)" type="button" class="linklike" @click="emit('open-item', n.ref)">{{ n.label }}</button>
          <span v-else>{{ n.label }}</span>
          <span v-if="n.position" class="muted">· {{ n.position }}</span>
          <SummaryTag v-if="n.summary" :code="n.summary" />
          <span v-if="n.provenance" class="muted" data-testid="provenance">· {{ n.provenance }}</span>
        </li>
      </ul>
    </div>
  </section>
</template>

<style scoped>
.genealogy {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 8px 16px;
}

h5 {
  margin: 0 0 4px;
}

ul {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding: 0;
  list-style: none;
}

li {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: baseline;
}

.muted {
  margin: 0;
  color: #6b7280;
  font-size: 12px;
}

.linklike {
  padding: 0;
  border: 0;
  background: none;
  color: #2f6fdb;
  font: inherit;
  cursor: pointer;
}
</style>
