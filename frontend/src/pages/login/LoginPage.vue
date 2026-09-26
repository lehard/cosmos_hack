<script setup lang="ts">
/**
 * Экран входа (FR-128). Демо-трек (эпик 08, заметка): сначала — выбор демо-персоны
 * без пароля; ниже — вход по логину, поле пароля заложено, но пока не обязательно.
 * После входа — стол своей роли (или страница, с которой пришли).
 */
import { computed, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { NAlert, NButton, NCard, NCollapse, NCollapseItem, NForm, NFormItem, NInput, NSpin, NText } from 'naive-ui'
import { useLogin, usePersonas, type DemoPersona } from '@/entities/session'
import { codeToKey } from '@/shared/i18n'
import { useProblemText } from '@/shared/i18n/problem'

const { t, te } = useI18n()
const route = useRoute()
const router = useRouter()
const problemText = useProblemText()

const personas = usePersonas()
const login = useLogin()
const form = reactive({ login: '', password: '' })

/** Персоны, сгруппированные по роли — в порядке политики. */
const groups = computed(() => {
  const map = new Map<string, { title: string; items: DemoPersona[] }>()
  for (const p of personas.data.value?.data.items ?? []) {
    const key = `roles.${codeToKey(p.role.id)}`
    const g = map.get(p.role.id) ?? { title: te(key) ? t(key) : p.role.title, items: [] }
    g.items.push(p)
    map.set(p.role.id, g)
  }
  return [...map.entries()].map(([id, g]) => ({ id, ...g }))
})

async function enter(body: { persona_id?: string; login?: string; password?: string }) {
  await login.mutateAsync({ data: body })
  const next = typeof route.query.next === 'string' ? route.query.next : '/desk'
  await router.replace(next)
}

const byPersona = (p: DemoPersona) => enter({ persona_id: p.id }).catch(() => undefined)
const byLogin = () =>
  enter({ login: form.login.trim(), ...(form.password ? { password: form.password } : {}) }).catch(() => undefined)
</script>

<template>
  <main class="login">
    <NCard :title="t('access.loginTitle')" class="card">
      <template #header-extra>
        <NText depth="3">{{ t('common.appName') }}</NText>
      </template>

      <NAlert v-if="login.isError.value" type="error" :bordered="false" class="gap">{{ problemText(login.error.value) }}</NAlert>

      <section data-testid="demo-personas">
        <h3>{{ t('shell.login.demoTitle') }}</h3>
        <NText depth="3">{{ t('shell.login.demoHint') }}</NText>
        <NSpin v-if="personas.isPending.value" size="small" class="gap" />
        <NAlert v-else-if="personas.isError.value" type="default" :bordered="false" class="gap">
          {{ t('shell.login.noPersonas') }} · {{ problemText(personas.error.value) }}
        </NAlert>
        <div v-else class="groups">
          <div v-for="g in groups" :key="g.id" class="group">
            <NText strong>{{ g.title }}</NText>
            <NButton
              v-for="p in g.items"
              :key="p.id"
              block
              secondary
              :loading="login.isPending.value && login.variables.value?.data.persona_id === p.id"
              :title="p.scope ? t('shell.login.scope', { scope: p.scope }) : undefined"
              @click="byPersona(p)"
            >
              {{ t('shell.login.enterAs', { name: p.name }) }}
            </NButton>
          </div>
        </div>
      </section>

      <NCollapse class="gap" :default-expanded-names="personas.isError.value ? ['login'] : []">
        <NCollapseItem :title="t('shell.login.byLogin')" name="login">
          <NForm @submit.prevent="byLogin">
            <NFormItem :label="t('access.username')" path="login">
              <NInput v-model:value="form.login" autocomplete="username" />
            </NFormItem>
            <NFormItem :label="t('shell.login.passwordOptional')" path="password">
              <NInput v-model:value="form.password" type="password" show-password-on="click" autocomplete="current-password" />
            </NFormItem>
            <NButton type="primary" attr-type="submit" :disabled="!form.login.trim()" :loading="login.isPending.value">
              {{ t('common.actions.login') }}
            </NButton>
          </NForm>
        </NCollapseItem>
      </NCollapse>
    </NCard>
  </main>
</template>

<style scoped>
.login {
  display: flex;
  align-items: flex-start;
  justify-content: center;
  min-height: 100%;
  padding: 48px 24px;
  box-sizing: border-box;
}

.card {
  width: min(960px, 100%);
}

.gap {
  margin-top: 12px;
}

.groups {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
  margin-top: 16px;
}

.group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

h3 {
  margin: 0 0 4px;
  font-size: 16px;
}
</style>
