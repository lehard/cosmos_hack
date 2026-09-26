<script setup lang="ts">
/**
 * Встроенная справка роли (FR-78, AD-21): руководство роли из бандла (работает
 * без сети) и перечень того, что стоит на столе — по данным стола, так что
 * справка не расходится со столом.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useDesk } from '@/entities/desk'
import { useSession } from '@/entities/session'
import { helpGuide, parseHelp } from '@/shared/help'
import { codeToKey } from '@/shared/i18n'
import { AppPage, EmptyState, KeyValue, KeyValueList, SectionPanel } from '@/shared/ui'
import { isWidgetId, widgetRegistry } from '@/widgets/registry'

const { t, te } = useI18n()
const route = useRoute()
const session = useSession()
const desk = useDesk()

const ownRole = computed(() => session.data.value?.data.role.id ?? '')
const role = computed(() => (typeof route.params.role === 'string' && route.params.role ? route.params.role : ownRole.value))
const roleTitle = computed(() => {
  const key = `roles.${codeToKey(role.value)}`
  return te(key) ? t(key) : role.value
})
const roleTask = computed(() => {
  const key = `roles.tasks.${codeToKey(role.value)}`
  return te(key) ? t(key) : ''
})
const guide = computed(() => {
  const d = desk.data.value?.data
  const md = helpGuide(role.value === ownRole.value && d?.help_key ? d.help_key : role.value)
  return md ? parseHelp(md) : null
})

/** Вкладки и виджеты своего стола. */
const contents = computed(() =>
  role.value === ownRole.value
    ? (desk.data.value?.data.tabs ?? []).map((tab) => ({
        id: tab.id,
        title: t(tab.title_key),
        widgets: tab.slots.map((s) => (isWidgetId(s.widget) ? t(widgetRegistry[s.widget].titleKey) : s.widget)),
      }))
    : [],
)
</script>

<template>
  <AppPage class="help" :title="`${t('shell.help.title')}: ${roleTitle}`" :subtitle="roleTask || undefined">
    <template #meta>{{ t('shell.help.offline') }}</template>

    <SectionPanel>
      <article v-if="guide" class="guide ant-wrap" data-testid="help-guide">
        <template v-for="(b, i) in guide" :key="i">
          <h2 v-if="b.kind === 'h' && b.level === 2">{{ b.text }}</h2>
          <h3 v-else-if="b.kind === 'h'">{{ b.text }}</h3>
          <p v-else-if="b.kind === 'p'">{{ b.text }}</p>
          <ul v-else-if="b.kind === 'ul'">
            <li v-for="(item, j) in b.items" :key="j">{{ item }}</li>
          </ul>
        </template>
      </article>
      <EmptyState v-else :title="t('shell.help.noGuide')" />
    </SectionPanel>

    <SectionPanel v-if="contents.length" :title="t('shell.help.deskContents')">
      <KeyValueList>
        <KeyValue v-for="tab in contents" :key="tab.id" :label="tab.title" :value="tab.widgets.join(', ')" />
      </KeyValueList>
    </SectionPanel>
  </AppPage>
</template>

<style scoped>
.help {
  max-width: 960px;
}

.guide {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
  line-height: var(--ant-lh-normal);
}

.guide h2 {
  margin-top: var(--ant-space-2);
  font-size: var(--ant-fs-xl);
}

.guide h3 {
  margin-top: var(--ant-space-1);
  font-size: var(--ant-fs-lg);
}

.guide ul {
  margin: 0;
  padding-left: var(--ant-space-5);
}

.guide li + li {
  margin-top: var(--ant-space-1);
}
</style>
