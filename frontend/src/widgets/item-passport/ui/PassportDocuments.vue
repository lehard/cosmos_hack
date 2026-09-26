<script setup lang="ts">
/**
 * Документы изделия (FR-65, AD-12): собраны из истории по шаблонам шагов —
 * название, шаблон с версией, отпечаток, статус маршрута подписей.
 */
import { useI18n } from 'vue-i18n'
import { DOCUMENT_STATUS_TEXT, codeText, type ItemDocumentRef } from '@/entities/item'

defineProps<{ documents: ItemDocumentRef[] }>()
const { t } = useI18n()
</script>

<template>
  <section class="documents" data-testid="passport-documents">
    <p v-if="!documents.length" class="muted">{{ t('empty.noDocuments') }}</p>
    <ul v-else>
      <li v-for="doc in documents" :key="doc.document_id" :data-status="doc.status">
        <strong>{{ doc.title }}</strong>
        <span class="status">{{ codeText(DOCUMENT_STATUS_TEXT, doc.status, t) }}</span>
        <span class="muted">{{ t('documents.generatedFromHistory') }} · {{ doc.template }}</span>
        <span class="muted">{{ t('documents.fingerprint') }}: <code>{{ doc.digest }}</code></span>
      </li>
    </ul>
  </section>
</template>

<style scoped>
ul {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 0;
  padding: 0;
  list-style: none;
}

li {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 10px;
  align-items: baseline;
}

.status {
  padding: 0 6px;
  border-radius: var(--ant-radius-lg);
  background: var(--ant-n-100);
  font-size: var(--ant-fs-meta);
}

.muted {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
  word-break: break-all;
}
</style>
