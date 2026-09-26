<script setup lang="ts">
/**
 * Экран входа (FR-128, UI-1, UI-2): вход в корпоративную систему «Главный».
 *
 * - Центрированная карточка ограниченной ширины: знак и имя системы, логин,
 *   пароль, «Войти»; ошибка входа — под полями.
 * - «Подать заявку на доступ» — та же карточка переключается на заявку
 *   (access.account.request): учётную запись активирует администратор.
 * - Ниже — свёрнутый блок «Демо-вход: выберите сотрудника». Он есть только
 *   там, где сервер отдаёт демо-персон (профили demo и fixtures; в остальных
 *   access.persona.list отвечает ошибкой — блока нет). Развёрнутый — персоны
 *   по ролям компактным списком с поиском.
 *
 * После входа — стол своей роли (или страница, с которой пришли).
 */
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { NAlert, NButton, NIcon, NInput, NSpin } from 'naive-ui'
import { ArrowLeft, ChevronDown, ChevronRight, Search, Users } from '@vicons/tabler'
import { useAccessRequest, useLogin, usePersonas, type DemoPersona } from '@/entities/session'
import { codeToKey } from '@/shared/i18n'
import { useProblemText } from '@/shared/i18n/problem'
import { BrandMark, EmptyState, FormField } from '@/shared/ui'

const { t, te } = useI18n()
const route = useRoute()
const router = useRouter()
const problemText = useProblemText()

const personas = usePersonas()
const login = useLogin()
const request = useAccessRequest()

/** Что показывает карточка: вход или заявка на доступ. */
const view = ref<'login' | 'request'>('login')

// --- вход по логину и паролю ---

const form = reactive({ login: '', password: '' })
const formErrors = reactive<{ login: string | null; password: string | null }>({ login: null, password: null })
/** Откуда была последняя попытка входа — туда и ошибка. */
const attempt = ref<'form' | 'persona' | null>(null)

const loginError = computed(() => (login.isError.value ? problemText(login.error.value) : null))

async function enter(body: { persona_id?: string; login?: string; password?: string }) {
  await login.mutateAsync({ data: body })
  const next = typeof route.query.next === 'string' ? route.query.next : '/desk'
  await router.replace(next)
}

function byLogin(): void {
  formErrors.login = form.login.trim() ? null : t('shell.login.loginRequired')
  formErrors.password = form.password ? null : t('shell.login.passwordRequired')
  if (formErrors.login || formErrors.password) return
  attempt.value = 'form'
  enter({ login: form.login.trim(), password: form.password }).catch(() => undefined)
}

// --- заявка на доступ (контракт AccountRequest) ---

const LOGIN_PATTERN = /^[a-z0-9._-]{3,64}$/
const req = reactive({ display_name: '', login: '', password: '' })
const reqErrors = reactive<{ display_name: string | null; login: string | null; password: string | null }>({ display_name: null, login: null, password: null })

function sendRequest(): void {
  reqErrors.display_name = req.display_name.trim() ? null : t('shell.login.displayNameRequired')
  reqErrors.login = LOGIN_PATTERN.test(req.login.trim()) ? null : t('shell.login.loginHint')
  reqErrors.password = req.password.length >= 8 ? null : t('shell.login.passwordHint')
  if (reqErrors.display_name || reqErrors.login || reqErrors.password) return
  request.mutate({ data: { display_name: req.display_name.trim(), login: req.login.trim(), password: req.password } })
}

function openRequest(): void {
  request.reset()
  view.value = 'request'
}

// --- демо-вход ---

/** Демо-вход есть, только если сервер отдал персон (профили demo и fixtures). */
const demoAvailable = computed(() => personas.isSuccess.value && (personas.data.value?.data.items.length ?? 0) > 0)
const demoOpen = ref(false)
const query = ref('')

const roleTitle = (p: DemoPersona) => {
  const key = `roles.${codeToKey(p.role.id)}`
  return te(key) ? t(key) : p.role.title
}

/** Персоны по ролям в порядке политики, с поиском по имени, роли и области. */
const groups = computed(() => {
  const q = query.value.trim().toLocaleLowerCase('ru')
  const map = new Map<string, { title: string; items: DemoPersona[] }>()
  for (const p of personas.data.value?.data.items ?? []) {
    const title = roleTitle(p)
    if (q && ![p.name, title, p.scope ?? ''].some((s) => s.toLocaleLowerCase('ru').includes(q))) continue
    const g = map.get(p.role.id) ?? { title, items: [] }
    g.items.push(p)
    map.set(p.role.id, g)
  }
  return [...map.entries()].map(([id, g]) => ({ id, ...g }))
})

const personaCount = computed(() => personas.data.value?.data.items.length ?? 0)

/** Инициалы для знака персоны: «Контролёр ОТК 1» → «КО». */
const initials = (name: string) =>
  name
    .split(/\s+/)
    .filter((w) => /^\p{L}/u.test(w))
    .slice(0, 2)
    .map((w) => w[0]!.toLocaleUpperCase('ru'))
    .join('')

const pendingPersona = computed(() => (login.isPending.value ? login.variables.value?.data.persona_id : undefined))

function byPersona(p: DemoPersona): void {
  attempt.value = 'persona'
  enter({ persona_id: p.id }).catch(() => undefined)
}
</script>

<template>
  <main class="login-screen">
    <div class="column">
      <BrandMark size="lg" tagline class="brand" />

      <section class="card" data-testid="login-card">
        <template v-if="view === 'login'">
          <header class="card-head">
            <h1 class="ant-wrap">{{ t('shell.login.title') }}</h1>
            <p class="lead ant-wrap">{{ t('shell.login.subtitle') }}</p>
          </header>

          <form class="fields" novalidate data-testid="login-form" @submit.prevent="byLogin">
            <FormField :label="t('access.username')" for="login-name" :error="formErrors.login">
              <NInput
                v-model:value="form.login"
                size="large"
                :status="formErrors.login ? 'error' : undefined"
                :input-props="{ id: 'login-name', autocomplete: 'username', name: 'username' }"
                @update:value="formErrors.login = null"
              />
            </FormField>
            <FormField :label="t('access.password')" for="login-password" :error="formErrors.password">
              <NInput
                v-model:value="form.password"
                type="password"
                size="large"
                show-password-on="click"
                :status="formErrors.password ? 'error' : undefined"
                :input-props="{ id: 'login-password', autocomplete: 'current-password', name: 'password' }"
                @update:value="formErrors.password = null"
              />
            </FormField>
            <NAlert v-if="loginError && attempt === 'form'" type="error" :bordered="false" data-testid="login-error">
              <span class="ant-wrap">{{ loginError }}</span>
            </NAlert>
            <NButton type="primary" size="large" block attr-type="submit" :loading="login.isPending.value && attempt === 'form'" data-testid="login-submit">
              {{ t('common.actions.login') }}
            </NButton>
          </form>

          <footer class="card-foot">
            <button type="button" class="link" data-testid="request-open" @click="openRequest">{{ t('shell.login.requestAccess') }}</button>
          </footer>
        </template>

        <template v-else>
          <header class="card-head">
            <button type="button" class="link back" @click="view = 'login'">
              <NIcon :size="16"><ArrowLeft /></NIcon>
              <span>{{ t('shell.login.backToLogin') }}</span>
            </button>
            <h1 class="ant-wrap">{{ t('shell.login.requestTitle') }}</h1>
            <p class="lead ant-wrap">{{ t('shell.login.requestHint') }}</p>
          </header>

          <NAlert v-if="request.isSuccess.value" type="success" :bordered="false" data-testid="request-sent">
            <span class="ant-wrap">{{ t('access.registrationSent') }}</span>
          </NAlert>
          <form v-else class="fields" novalidate data-testid="request-form" @submit.prevent="sendRequest">
            <FormField :label="t('shell.login.displayName')" for="request-name" :error="reqErrors.display_name">
              <NInput v-model:value="req.display_name" size="large" :maxlength="128" :input-props="{ id: 'request-name', autocomplete: 'name' }" @update:value="reqErrors.display_name = null" />
            </FormField>
            <FormField :label="t('access.username')" for="request-login" :hint="t('shell.login.loginHint')" :error="reqErrors.login">
              <NInput v-model:value="req.login" size="large" :maxlength="64" :input-props="{ id: 'request-login', autocomplete: 'username' }" @update:value="reqErrors.login = null" />
            </FormField>
            <FormField :label="t('access.password')" for="request-password" :hint="t('shell.login.passwordHint')" :error="reqErrors.password">
              <NInput
                v-model:value="req.password"
                type="password"
                size="large"
                show-password-on="click"
                :input-props="{ id: 'request-password', autocomplete: 'new-password' }"
                @update:value="reqErrors.password = null"
              />
            </FormField>
            <NAlert v-if="request.isError.value" type="error" :bordered="false">
              <span class="ant-wrap">{{ problemText(request.error.value) }}</span>
            </NAlert>
            <NButton type="primary" size="large" block attr-type="submit" :loading="request.isPending.value">{{ t('shell.login.sendRequest') }}</NButton>
          </form>
        </template>
      </section>

      <section v-if="demoAvailable" class="demo" :data-open="demoOpen || undefined" data-testid="demo-personas">
        <button type="button" class="demo-toggle" :aria-expanded="demoOpen" aria-controls="demo-list" data-testid="demo-toggle" @click="demoOpen = !demoOpen">
          <NIcon :size="18" class="demo-icon"><Users /></NIcon>
          <span class="demo-title ant-ellipsis">{{ t('shell.login.demoTitle') }}</span>
          <span class="demo-count">{{ t('shell.login.demoCount', personaCount) }}</span>
          <NIcon :size="16" class="chevron"><ChevronDown /></NIcon>
        </button>

        <div v-if="demoOpen" id="demo-list" class="demo-body" data-testid="demo-list">
          <p class="demo-hint ant-wrap">{{ t('shell.login.demoHint') }}</p>
          <NInput v-model:value="query" clearable :placeholder="t('shell.login.demoSearch')" data-testid="demo-search">
            <template #prefix>
              <NIcon><Search /></NIcon>
            </template>
          </NInput>
          <NAlert v-if="loginError && attempt === 'persona'" type="error" :bordered="false">
            <span class="ant-wrap">{{ loginError }}</span>
          </NAlert>

          <div class="groups">
            <section v-for="g in groups" :key="g.id" class="group" :data-role="g.id">
              <h3 class="group-title ant-ellipsis" :title="g.title">{{ g.title }}</h3>
              <ul class="people">
                <li v-for="p in g.items" :key="p.id">
                  <button
                    type="button"
                    class="persona"
                    :data-persona="p.id"
                    :disabled="login.isPending.value"
                    :title="[t('shell.login.enterAs', { name: p.name }), p.scope ? t('shell.login.scope', { scope: p.scope }) : ''].filter(Boolean).join('\n')"
                    @click="byPersona(p)"
                  >
                    <span class="avatar" aria-hidden="true">{{ initials(p.name) }}</span>
                    <span class="persona-text">
                      <span class="persona-name ant-ellipsis">{{ p.name }}</span>
                      <span v-if="p.scope" class="persona-scope ant-ellipsis">{{ p.scope }}</span>
                    </span>
                    <NSpin v-if="pendingPersona === p.id" :size="14" />
                    <NIcon v-else :size="16" class="go"><ChevronRight /></NIcon>
                  </button>
                </li>
              </ul>
            </section>
            <EmptyState v-if="!groups.length" compact :title="t('shell.login.demoNotFound', { query })" />
          </div>
        </div>
      </section>

      <p class="legal ant-wrap">{{ t('shell.login.footer') }}</p>
    </div>
  </main>
</template>

<style scoped>
.login-screen {
  display: flex;
  justify-content: center;
  min-height: 100vh;
  padding: 10vh var(--ant-space-4) var(--ant-space-8);
  background:
    radial-gradient(1200px 480px at 50% -10%, var(--ant-n-0), transparent 70%),
    var(--ant-bg-app);
}

.column {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-4);
  width: min(var(--ant-w-auth), 100%);
}

.brand {
  align-self: center;
  margin-bottom: var(--ant-space-4);
}

.card {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-5);
  padding: var(--ant-space-8) var(--ant-space-8) var(--ant-space-6);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-lg);
  background: var(--ant-surface);
  box-shadow: var(--ant-shadow-md);
}

.card-head {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
}

.card-head h1 {
  font-size: var(--ant-fs-xl);
}

.lead {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-sm);
}

.fields {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-4);
}

.card-foot {
  display: flex;
  justify-content: center;
  padding-top: var(--ant-space-4);
  border-top: 1px solid var(--ant-border);
}

.link {
  display: inline-flex;
  gap: var(--ant-space-1);
  align-items: center;
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  font: inherit;
  font-size: var(--ant-fs-sm);
  cursor: pointer;
}

.link:hover {
  text-decoration: underline;
}

.back {
  align-self: flex-start;
  margin-bottom: var(--ant-space-3);
  color: var(--ant-text-2);
}

/* --- демо-вход --- */

.demo {
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-lg);
  background: var(--ant-surface);
}

.demo-toggle {
  display: flex;
  gap: var(--ant-space-2);
  align-items: center;
  width: 100%;
  min-width: 0;
  padding: var(--ant-space-3) var(--ant-space-4);
  border: 0;
  border-radius: var(--ant-radius-lg);
  background: none;
  color: var(--ant-text);
  font: inherit;
  font-size: var(--ant-fs-sm);
  text-align: left;
  cursor: pointer;
}

.demo-toggle:hover {
  background: var(--ant-surface-subtle);
}

.demo-icon {
  flex: none;
  color: var(--ant-text-3);
}

.demo-title {
  flex: 1 1 auto;
  font-weight: var(--ant-fw-bold);
}

.demo-count {
  flex: none;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
  white-space: nowrap;
}

.chevron {
  flex: none;
  color: var(--ant-text-3);
  transition: transform 0.15s ease;
}

.demo[data-open] .chevron {
  transform: rotate(180deg);
}

.demo-body {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
  padding: var(--ant-space-1) var(--ant-space-4) var(--ant-space-4);
  border-top: 1px solid var(--ant-border);
}

.demo-hint {
  padding-top: var(--ant-space-2);
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
}

.groups {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
  max-height: min(56vh, 520px);
  margin: 0 calc(-1 * var(--ant-space-2));
  padding: 0 var(--ant-space-2);
  overflow-y: auto;
}

.group-title {
  margin-bottom: 2px;
  padding: 0 var(--ant-space-2);
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
  font-weight: var(--ant-fw-bold);
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.people {
  margin: 0;
  padding: 0;
  list-style: none;
}

.persona {
  display: flex;
  gap: var(--ant-space-3);
  align-items: center;
  width: 100%;
  min-width: 0;
  padding: 6px var(--ant-space-2);
  border: 0;
  border-radius: var(--ant-radius-md);
  background: none;
  color: var(--ant-text);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.persona:hover:not(:disabled),
.persona:focus-visible {
  background: var(--ant-surface-hover);
}

.persona:disabled {
  cursor: progress;
  opacity: 0.7;
}

.avatar {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--ant-n-100);
  color: var(--ant-text-2);
  font-size: var(--ant-fs-xs);
  font-weight: var(--ant-fw-bold);
}

.persona-text {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  min-width: 0;
  line-height: var(--ant-lh-tight);
}

.persona-name {
  font-size: var(--ant-fs-sm);
}

.persona-scope {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
}

.go {
  flex: none;
  color: var(--ant-n-300);
}

.persona:hover .go {
  color: var(--ant-text-2);
}

.legal {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
  text-align: center;
}
</style>
