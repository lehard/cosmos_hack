<script setup lang="ts">
/**
 * Окно документа (Д-70; FR-65, FR-66, FR-136, FR-139; AD-12, AD-13, AD-43;
 * Д-72): заголовок — вид, номер, состояние; вкладки:
 * - «Маршрут подписей» — этапы по порядку: кто подписал, когда, каким
 *   ключом (физический ключ / ключ в браузере) или на бумаге с заверением,
 *   отказы с замечаниями, кто ещё должен подписать;
 * - «Содержимое» — каноническая отрисовка версии (documents.document.render)
 *   и сводка для подписи с отпечатком;
 * - «Версии и бумага» — версии с отпечатками, бумажные экземпляры.
 * Внизу — кнопки по правам (@casl/vue): «Подписать» (порт подписи —
 * расширение «Главный — подпись»; в демо-профиле без ключа — демо-подпись,
 * Д-30), «Отказать» (замечание обязательно), «Печать с QR», «Скачать».
 * В воспроизведении прошлого команды выключены (AD-21).
 */
import { computed, inject, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ABILITY_TOKEN } from '@casl/vue'
import { NInput } from 'naive-ui'
import { useSession } from '@/entities/session'
import { useDrillDown } from '@/features/drill-down'
import { payloadTypeOf, toPayloadB64, useSigningPort } from '@/features/sign-decision'
import { statusPalette } from '@/shared/api/generated/statuses'
import type { DocumentRouteStage, DocumentStageSignature } from '@/shared/api/generated/model'
import { useProblemText } from '@/shared/i18n/problem'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton, EmptyState, KeyValue, KeyValueList, RecordDrawer, SectionPanel, type RecordDrawerTab } from '@/shared/ui'
import { fetchPrintView, useDocument, useDocumentCommand, useRendering } from '../model/api'
import { STATE_TONE, docState, itemLabel, keyStorageKey, kindKey, subjectKey, templateId } from '../model/registry'

const props = defineProps<{ id: string; show: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { t, te, d } = useI18n()
const drill = useDrillDown()
const problemText = useProblemText()
const moment = useMomentStore()
const session = useSession()
const port = useSigningPort()
const ability = inject(ABILITY_TOKEN, null)
const can = (action: string) => ability?.can(action, 'document') ?? false

const version = ref(0)
watch(
  () => props.id,
  () => {
    version.value = 0
    tab.value = 'route'
  },
)
const docQ = useDocument(() => props.id, version)
const doc = computed(() => docQ.data.value?.data ?? null)
const renderQ = useRendering(() => props.id, () => doc.value?.version ?? 0)
const html = computed(() => renderQ.data.value?.data.html ?? '')

const tab = ref('route')
const tabs = computed<RecordDrawerTab[]>(() => [
  { id: 'route', label: t('docRegistry.tabs.route') },
  { id: 'content', label: t('docRegistry.tabs.content') },
  { id: 'versions', label: t('docRegistry.tabs.versions') },
])

const state = computed(() => (doc.value ? docState({ state: undefined, status: doc.value.status, paper_status: lastPaper.value }) : 'draft'))
const lastPaper = computed(() => {
  const marks = (doc.value?.paper ?? []).filter((p) => p.version === doc.value?.version)
  return marks.length ? marks[marks.length - 1]!.paper_status : undefined
})
const kindLabel = computed(() => {
  const ref = doc.value?.template ?? ''
  return te(kindKey(ref)) ? t(kindKey(ref)) : templateId(ref) || t('docRegistry.kind')
})
const subjectText = computed(() => {
  const s = doc.value?.subject
  if (!s) return ''
  const kind = te(subjectKey(s.entity)) ? t(subjectKey(s.entity)) : s.entity
  return `${kind}: ${doc.value?.subject_label || (s.entity === 'item' ? itemLabel(s.id) : s.id)}`
})
const route = computed<DocumentRouteStage[]>(() => doc.value?.route ?? [])

/** Первый незакрытый этап — тот, что ждёт подписи сейчас. */
const nextStage = computed(() => {
  const dv = doc.value
  if (!dv || dv.live || dv.status === 'route_closed' || dv.status === 'annulled' || dv.status === 'returned') return null
  return route.value.find((s) => !s.done) ?? null
})
const me = computed(() => session.data.value?.data?.user.id ?? '')
const candidatesOf = (stage: number) => doc.value?.stages.find((s) => s.stage === stage)?.candidates ?? []
const myStage = computed(() => (nextStage.value && candidatesOf(nextStage.value.stage).includes(me.value) ? nextStage.value : null))
const replay = computed(() => moment.isReplay)
const demo = computed(() => session.data.value?.data?.demo === true)

const cmd = useDocumentCommand()
const error = ref<unknown>(undefined)
const declining = ref(false)
const comment = ref('')
const done = ref<string | null>(null)
watch(
  () => props.id,
  () => {
    error.value = undefined
    declining.value = false
    comment.value = ''
    done.value = null
  },
)

async function sign(withKey: boolean): Promise<void> {
  const dv = doc.value
  const st = myStage.value
  if (!dv || !st) return
  error.value = undefined
  try {
    if (withKey) {
      // Уровень 2: окно подтверждения — в расширении; отпечаток и сводку оно считает само (AD-14).
      const signature = await port.sign({
        level: 2,
        payload_type: payloadTypeOf('document-signature'),
        payload_b64: toPayloadB64(dv.content),
        template_ref: dv.template,
        doc_format_version: dv.doc_format_version,
        expected_doc_digest: dv.doc_digest,
      })
      await cmd.mutateAsync({ kind: 'sign', doc: dv, stage: st.stage, signature })
    } else {
      await cmd.mutateAsync({ kind: 'sign_demo', doc: dv, stage: st.stage })
    }
    done.value = t('docRegistry.done.signed')
  } catch (err) {
    if ((err as { info?: { code?: string } } | null)?.info?.code !== 'signing.cancelled') error.value = err
  }
}

async function decline(): Promise<void> {
  const dv = doc.value
  const st = myStage.value
  if (!dv || !st || !comment.value.trim()) return
  error.value = undefined
  try {
    await cmd.mutateAsync({ kind: 'decline', doc: dv, stage: st.stage, comment: comment.value.trim() })
    declining.value = false
    comment.value = ''
    done.value = t('docRegistry.done.declined')
  } catch (err) {
    error.value = err
  }
}

/** Печать с QR: запись «напечатан» и печатная форма в новом окне (FR-139). */
async function print(): Promise<void> {
  const dv = doc.value
  if (!dv) return
  error.value = undefined
  const w = window.open('', '_blank')
  try {
    if (can('documents.paper.print') && !replay.value) await cmd.mutateAsync({ kind: 'print', doc: dv })
    const page = await fetchPrintView(dv.document_id, dv.version)
    if (w) {
      w.document.open()
      w.document.write(page)
      w.document.close()
    }
  } catch (err) {
    w?.close()
    error.value = err
  }
}

/** Скачать каноническую отрисовку версии файлом HTML. */
function download(): void {
  const dv = doc.value
  if (!dv || !html.value) return
  const page = `<!doctype html><html lang="ru"><head><meta charset="utf-8"><title>${dv.document_id}</title></head><body>${html.value}</body></html>`
  const url = URL.createObjectURL(new Blob([page], { type: 'text/html;charset=utf-8' }))
  const a = document.createElement('a')
  a.href = url
  a.download = `${dv.document_id}-v${dv.version}.html`
  a.click()
  URL.revokeObjectURL(url)
}

const when = (iso?: string | null) => (iso ? d(new Date(iso), 'dateTime') : '—')
const methodText = (s: DocumentStageSignature): string => {
  if (s.method === 'paper') return t('docRegistry.method.paper')
  if (s.method === 'source_decision') return t('docRegistry.method.source')
  if (s.method === 'demo_signer') return t('docRegistry.method.demo')
  const key = keyStorageKey(s.key_storage)
  return key ? t(key) : t('docRegistry.method.key')
}
const declinesOf = (stage: number) => (doc.value?.declines ?? []).filter((x) => x.stage === stage)
const stageState = (s: DocumentRouteStage): 'done' | 'declined' | 'next' | 'waiting' =>
  s.done ? 'done' : declinesOf(s.stage).length ? 'declined' : nextStage.value?.stage === s.stage ? 'next' : 'waiting'
const STAGE_TONE = { done: 'success', declined: 'danger', next: 'attention', waiting: 'neutral' } as const

/** Отрисовка — в изолированном фрейме: HTML сервера со своими простыми стилями. */
const srcdoc = computed(
  () =>
    `<!doctype html><html lang="ru"><head><meta charset="utf-8"><style>body{font:14px/1.45 'PT Sans',system-ui,sans-serif;color:#1f2329;margin:12px}` +
    `h1{font-size:18px;margin:0 0 4px}h2{font-size:15px;margin:16px 0 6px}.ant-doc-id{color:#6b7280;margin:0}dl{display:grid;grid-template-columns:minmax(8rem,35%) 1fr;gap:4px 12px;margin:0}` +
    `dt{color:#6b7280}dd{margin:0}table{border-collapse:collapse;width:100%;font-size:13px}th,td{border:1px solid #e5e7eb;padding:3px 6px;text-align:left;vertical-align:top}` +
    `ol{padding-left:20px;margin:0}</style></head><body>${html.value}</body></html>`,
)
</script>

<template>
  <RecordDrawer
    v-model:tab="tab"
    :show="show"
    :kind-label="kindLabel"
    :number="id"
    :subtitle="doc?.title ?? ''"
    :tabs="tabs"
    :loading="docQ.isPending.value && !doc && !docQ.error.value"
    data-record="document"
    @close="emit('close')"
  >
    <template v-if="doc" #status>
      <span class="state-tag" :data-state="state">
        <span class="dot" :style="{ background: statusPalette[STATE_TONE[state]] }" aria-hidden="true" />
        <span class="ant-ellipsis">{{ t(`docRegistry.states.${state}`) }}</span>
      </span>
      <span v-if="doc.live" class="muted">{{ t('docRegistry.live') }}</span>
    </template>
    <template v-if="doc" #links>
      <button v-if="doc.subject.entity === 'item' || doc.subject.entity === 'nonconformity'" type="button" class="link ant-wrap" data-testid="doc-subject" @click="drill.open(doc.subject)">
        {{ subjectText }}
      </button>
      <span v-else class="muted ant-wrap">{{ subjectText }}</span>
    </template>

    <EmptyState v-if="!doc && docQ.error.value" compact :title="t('docRegistry.unavailable')" :description="problemText(docQ.error.value)" />

    <div v-else-if="doc" class="sections ant-box">
      <!-- Маршрут подписей по этапам -->
      <template v-if="tab === 'route'">
        <p v-if="doc.live" class="muted ant-wrap">{{ t('docRegistry.liveHint') }}</p>
        <ol class="stages ant-box" data-testid="doc-route">
          <li v-for="s in route" :key="s.stage" class="stage ant-box" :data-stage="s.stage" :data-stage-state="stageState(s)">
            <div class="stage-head ant-box">
              <span class="dot" :style="{ background: statusPalette[STAGE_TONE[stageState(s)]] }" aria-hidden="true" />
              <strong class="ant-wrap">{{ t('docRegistry.stage', { n: s.stage }) }} · {{ s.authority_label }}</strong>
              <span class="muted">{{ t(`docRegistry.stageState.${stageState(s)}`) }}</span>
            </div>
            <ul v-if="s.signatures.length" class="sigs ant-box">
              <li v-for="g in s.signatures" :key="g.event_id" class="sig ant-box" :data-key-storage="g.key_storage || undefined">
                <span class="ant-wrap"><strong>{{ g.signer_id }}</strong> · {{ when(g.signed_at) }}</span>
                <span class="muted ant-wrap">
                  {{ methodText(g) }}<template v-if="g.attested_by"> · {{ t('docRegistry.attestedBy', { who: g.attested_by }) }}</template
                  ><template v-if="g.paper_original_ref"> · {{ t('docRegistry.paperNo', { no: g.paper_original_ref }) }}</template
                  ><template v-if="g.key_ref"> · <span class="ant-mono">{{ g.key_ref }}</span></template>
                  · {{ t('docRegistry.level', { n: g.level }) }}
                  <template v-if="!g.counted"> · {{ t('docRegistry.notCounted') }}</template>
                </span>
              </li>
            </ul>
            <ul v-if="declinesOf(s.stage).length" class="sigs ant-box">
              <li v-for="x in declinesOf(s.stage)" :key="x.event_id" class="sig declined ant-box">
                <span class="ant-wrap"><strong>{{ x.signer_id }}</strong> · {{ t('docRegistry.declinedAt', { at: when(x.at) }) }}</span>
                <span class="ant-wrap">«{{ x.comment }}»</span>
              </li>
            </ul>
            <p v-if="!s.done && !declinesOf(s.stage).length && candidatesOf(s.stage).length" class="muted ant-wrap">
              {{ t('docRegistry.whoCan', { who: candidatesOf(s.stage).join(', ') }) }}
            </p>
            <p v-if="s.paper_allowed && !s.done" class="muted ant-wrap">{{ t('docRegistry.paperAllowed') }}</p>
          </li>
        </ol>
      </template>

      <!-- Содержимое: каноническая отрисовка и сводка для подписи -->
      <template v-else-if="tab === 'content'">
        <iframe v-if="html" class="rendering" sandbox="" :srcdoc="srcdoc" :title="doc.title" data-testid="doc-rendering" />
        <EmptyState v-else-if="renderQ.error.value" compact :title="t('docRegistry.unavailable')" :description="problemText(renderQ.error.value)" />
        <SectionPanel :title="t('docRegistry.summary')" variant="plain" :padded="false">
          <KeyValueList>
            <KeyValue v-for="f in doc.summary_fields" :key="f.key" :label="f.label" :value="f.value" />
            <KeyValue :label="t('docRegistry.digest')" :value="doc.doc_digest" mono />
            <KeyValue :label="t('docRegistry.template')" :value="doc.template" mono />
          </KeyValueList>
        </SectionPanel>
      </template>

      <!-- Версии и бумажные экземпляры -->
      <template v-else>
        <SectionPanel :title="t('docRegistry.versions')" variant="plain" :padded="false">
          <p v-if="!doc.versions?.length" class="muted">{{ t('docRegistry.noVersions') }}</p>
          <ul v-else class="versions ant-box">
            <li v-for="v in doc.versions" :key="v.version" class="ant-box" :data-version="v.version">
              <button type="button" class="link" :aria-current="v.version === doc.version" @click="version = v.version">
                {{ t('docRegistry.version', { n: v.version }) }}
              </button>
              <span class="muted">{{ t(`docRegistry.versionStatus.${v.status}`) }} · {{ when(v.drafted_at) }} · {{ t('docRegistry.signatures', { n: v.signatures }) }}</span>
              <span class="ant-mono muted ant-wrap">{{ v.doc_digest }}</span>
            </li>
          </ul>
        </SectionPanel>
        <SectionPanel :title="t('docRegistry.paper')" variant="plain" :padded="false">
          <p v-if="!doc.paper?.length" class="muted">{{ t('docRegistry.noPaper') }}</p>
          <ul v-else class="versions ant-box">
            <li v-for="p in doc.paper" :key="p.event_id" class="ant-box">
              {{ t(`docRegistry.paperStatus.${p.paper_status}`) }} · {{ t('docRegistry.version', { n: p.version }) }}
              <template v-if="p.copy_no"> · {{ t('docRegistry.copy', { n: p.copy_no }) }}</template> · {{ when(p.at) }}
            </li>
          </ul>
        </SectionPanel>
      </template>

      <div v-if="declining" class="decline ant-box" data-testid="doc-decline-form">
        <NInput v-model:value="comment" type="textarea" :autosize="{ minRows: 2, maxRows: 6 }" :placeholder="t('docRegistry.declineComment')" />
      </div>
      <p v-if="done" class="ok ant-wrap" data-testid="doc-done">{{ done }}</p>
      <p v-if="error" class="err ant-wrap" data-testid="doc-error">{{ problemText(error) }}</p>
    </div>

    <template v-if="doc" #actions>
      <template v-if="myStage && !replay && can('documents.document.sign')">
        <template v-if="!declining">
          <ActionButton type="primary" :loading="cmd.isPending.value" data-action="sign" :label="t('docRegistry.actions.sign')" @click="sign(true)" />
          <ActionButton v-if="demo" secondary :disabled="cmd.isPending.value" data-action="sign-demo" :label="t('docRegistry.actions.signDemo')" @click="sign(false)" />
          <ActionButton v-if="can('documents.signature.decline')" secondary data-action="decline" :label="t('docRegistry.actions.decline')" @click="declining = true" />
        </template>
        <template v-else>
          <ActionButton type="error" :disabled="!comment.trim()" :loading="cmd.isPending.value" data-action="decline-send" :label="t('docRegistry.actions.declineSend')" @click="decline" />
          <ActionButton secondary data-action="decline-cancel" :label="t('docRegistry.actions.cancel')" @click="declining = false" />
        </template>
      </template>
      <ActionButton v-if="!doc.live" secondary data-action="print" :label="t('docRegistry.actions.print')" @click="print" />
      <ActionButton secondary :disabled="!html" data-action="download" :label="t('docRegistry.actions.download')" @click="download" />
    </template>
  </RecordDrawer>
</template>

<style scoped>
.sections {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-4);
}

.stages {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
  margin: 0;
  padding: 0;
  list-style: none;
}

.stage {
  padding: var(--ant-space-3);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
}

.stage[data-stage-state='next'] {
  border-color: var(--ant-status-attention);
}

.stage-head {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  align-items: center;
}

.sigs,
.versions {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  margin: var(--ant-space-2) 0 0;
  padding: 0;
  list-style: none;
}

.sig,
.versions li {
  display: flex;
  flex-direction: column;
}

.sig.declined {
  color: var(--ant-status-danger);
}

.stage p {
  margin: var(--ant-space-2) 0 0;
}

.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.state-tag {
  display: inline-flex;
  align-items: center;
  gap: var(--ant-space-2);
}

.rendering {
  width: 100%;
  min-height: 28rem;
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface);
}

.link {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.link[aria-current='true'] {
  font-weight: var(--ant-fw-bold);
}

.ok {
  margin: 0;
  color: var(--ant-status-success);
}

.err {
  margin: 0;
  color: var(--ant-status-danger);
}
</style>
