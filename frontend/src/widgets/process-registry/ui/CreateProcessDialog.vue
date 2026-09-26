<script setup lang="ts">
/**
 * «Создать процесс» (эпик 39; FR-10, FR-13, FR-25): с чистого листа — схема
 * «начало → контроль ОТК → конец» с новым id процесса, или загрузкой файла
 * .bpmn (BPMN 2.0 XML с нашими свойствами urn:ant:bpmn-ext:1). Итог —
 * черновик версии (`process.version.draft`); процесс определяется id главного
 * bpmn:process, поэтому отдельной операции «создать процесс» нет (решение
 * эпика 39). Сервер проверяет описание при загрузке и отвечает кодом и id
 * элемента — перечень нарушений показывается здесь же.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NInput, NModal, NRadioButton, NRadioGroup, useMessage } from 'naive-ui'
import { problemOf } from '@/shared/api/problem'
import { useProblemText } from '@/shared/i18n/problem'
import { FormField, SectionPanel } from '@/shared/ui'
import { blankProcessXml, BpmnFileError, newProcessId, readBpmnFile, type BpmnFileInfo } from '../model/bpmn-file'
import { useLifecycle, type ProcessSummary } from '../model/source'

const props = defineProps<{ show: boolean; processes: readonly ProcessSummary[] }>()
const emit = defineEmits<{ close: []; created: [processId: string] }>()
const { t } = useI18n()
const message = useMessage()
const problemText = useProblemText()
const cmd = useLifecycle()

const mode = ref<'blank' | 'upload'>('blank')
const name = ref('')
const fileText = ref<string | null>(null)
const fileInfo = ref<BpmnFileInfo | null>(null)
const fileError = ref<string | null>(null)
const details = ref<string | null>(null)

watch(
  () => props.show,
  (s) => {
    if (!s) return
    mode.value = 'blank'
    name.value = ''
    fileText.value = null
    fileInfo.value = null
    fileError.value = null
    details.value = null
  },
)

const existing = computed(() => (fileInfo.value ? (props.processes.find((p) => p.process_id === fileInfo.value!.processId) ?? null) : null))
const ready = computed(() => (mode.value === 'blank' ? name.value.trim().length > 0 : Boolean(fileText.value && fileInfo.value)))

async function onFile(ev: Event): Promise<void> {
  const file = (ev.target as HTMLInputElement).files?.[0]
  fileText.value = null
  fileInfo.value = null
  fileError.value = null
  details.value = null
  if (!file) return
  const text = await file.text()
  try {
    fileInfo.value = readBpmnFile(text)
    fileText.value = text
  } catch (e) {
    fileError.value = e instanceof BpmnFileError ? t(`processEditor.fileErrors.${e.key}`) : String(e)
  }
}

async function submit(): Promise<void> {
  details.value = null
  let xml: string
  let pid: string
  let label = 'v1'
  if (mode.value === 'blank') {
    pid = newProcessId()
    xml = blankProcessXml(pid, name.value.trim())
  } else {
    xml = fileText.value!
    pid = fileInfo.value!.processId
    if (existing.value) label = `v${existing.value.versions + 1}`
  }
  try {
    await cmd.mutateAsync({ kind: 'draft', label, bpmn_xml: xml })
    message.success(t('processEditor.create.created', { name: mode.value === 'blank' ? name.value.trim() : fileInfo.value!.name }))
    emit('created', pid)
  } catch (e) {
    message.error(problemText(e))
    const p = problemOf(e)
    details.value = p?.detail ? String(p.detail) : null
  }
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="t('processEditor.create.title')"
    style="width: min(96vw, 640px)"
    :mask-closable="!cmd.isPending.value"
    data-testid="create-process"
    @update:show="(s: boolean) => !s && emit('close')"
  >
    <div class="form ant-box">
      <NRadioGroup v-model:value="mode" name="create-mode">
        <NRadioButton value="blank" data-mode="blank">{{ t('processEditor.create.blank') }}</NRadioButton>
        <NRadioButton value="upload" data-mode="upload">{{ t('processEditor.create.upload') }}</NRadioButton>
      </NRadioGroup>
      <FormField v-if="mode === 'blank'" :label="t('processEditor.create.name')" :hint="t('processEditor.create.blankHint')" required>
        <NInput v-model:value="name" :maxlength="200" data-field="name" />
      </FormField>
      <FormField v-else :label="t('processEditor.create.file')" :hint="t('processEditor.create.fileHint')" :error="fileError" required>
        <input type="file" accept=".bpmn,.xml,application/xml,text/xml" data-field="file" @change="onFile" />
      </FormField>
      <template v-if="mode === 'upload' && fileInfo">
        <p class="note ant-wrap">{{ fileInfo.name }} · <span class="ant-mono">{{ fileInfo.processId }}</span></p>
        <p v-if="existing" class="note ant-wrap" data-testid="existing">{{ t('processEditor.create.existing', { name: existing.name }) }}</p>
        <p v-if="!fileInfo.hasAntExtension" class="warn ant-wrap" data-testid="no-extension">{{ t('processEditor.create.noExtension') }}</p>
      </template>
      <SectionPanel v-if="details" variant="subtle" :title="t('processEditor.fileErrors.details')">
        <ul class="list">
          <li v-for="(x, i) in details.split('; ')" :key="i" class="ant-wrap ant-mono">{{ x }}</li>
        </ul>
      </SectionPanel>
    </div>
    <template #footer>
      <div class="buttons">
        <NButton quaternary @click="emit('close')">{{ t('processEditor.actions.cancel') }}</NButton>
        <NButton type="primary" :disabled="!ready" :loading="cmd.isPending.value" data-action="create" @click="submit">
          {{ t('processEditor.create.submit') }}
        </NButton>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.form {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
}

.note {
  margin: 0;
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

.warn {
  margin: 0;
  color: var(--ant-status-danger, var(--ant-text));
  font-size: var(--ant-fs-meta);
}

.list {
  margin: 0;
  padding-left: var(--ant-space-5);
}

.buttons {
  display: flex;
  gap: var(--ant-space-2);
  justify-content: flex-end;
}
</style>
