<script setup lang="ts">
/**
 * Согласования назначения контролёра на пост (PRD §11.18): документы поста по
 * шаблону «назначение контролёра» (`documents.document.list`) и их статус;
 * маршрут закрыт (начальник ОТК согласовал) — можно назначить по документу.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NFlex, NTag, NText } from 'naive-ui'
import { CONTROLLER_ASSIGNMENT_TEMPLATE, useWorkplaceDocuments, type DocumentSummary } from '@/entities/workplace'
import { codeToKey } from '@/shared/i18n'
import { isControllerApproval } from '../model/shift'

const props = withDefaults(defineProps<{ workplaceId: string; canAct?: boolean; size?: 'small' | 'medium' | 'large' }>(), { canAct: true, size: 'medium' })
const emit = defineEmits<{ use: [doc: DocumentSummary] }>()
const { t, d } = useI18n()
const docsQ = useWorkplaceDocuments(() => props.workplaceId)
const docs = computed(() =>
  (docsQ.data.value?.data ?? []).filter((doc) => doc.subject.id === props.workplaceId && isControllerApproval(doc, CONTROLLER_ASSIGNMENT_TEMPLATE)),
)
const tagType = (s: DocumentSummary['status']) => (s === 'route_closed' ? 'success' : s === 'annulled' ? 'default' : 'warning')
</script>

<template>
  <NFlex v-if="docs.length" vertical :size="4" data-testid="approvals">
    <NFlex v-for="doc in docs" :key="doc.document_id" :size="6" align="center" :wrap="true" :data-document="doc.document_id" :data-status="doc.status">
      <NTag size="small" :bordered="false" :type="tagType(doc.status)">{{ t(`widgets.shopFloor.shift.approvalStatus.${codeToKey(doc.status)}`) }}</NTag>
      <NText depth="3">{{ doc.document_id }}<template v-if="doc.closed_at"> · {{ d(new Date(doc.closed_at), 'dateTime') }}</template></NText>
      <NButton v-if="doc.status === 'route_closed' && canAct" :size="size" secondary type="primary" data-testid="use-approval" @click="emit('use', doc)">
        {{ t('widgets.shopFloor.shift.assignByApproval') }}
      </NButton>
    </NFlex>
  </NFlex>
</template>
