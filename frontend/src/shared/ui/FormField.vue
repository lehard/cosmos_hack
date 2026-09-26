<script setup lang="ts">
/**
 * Поле формы: подпись, элемент управления (слот), подсказка и ошибка под полем.
 * Ошибка вытесняет подсказку и читается программами чтения экрана (role=alert).
 *
 * Когда брать: любая форма (вход, заявка, решение, фильтр с подписью). Для поля
 * без подписи в панели инструментов — элемент Naive UI напрямую.
 */
import { useId } from 'vue'

withDefaults(
  defineProps<{
    /** Подпись поля. */
    label: string
    /** id элемента управления — для связи подписи с полем. */
    for?: string
    /** Подсказка под полем. */
    hint?: string
    /** Текст ошибки; есть — поле в состоянии ошибки. */
    error?: string | null
    /** Обязательное поле — звёздочка у подписи. */
    required?: boolean
  }>(),
  { for: undefined, hint: undefined, error: null, required: false },
)

const messageId = useId()
</script>

<template>
  <div class="form-field" :data-invalid="error ? 'true' : undefined">
    <label class="label ant-wrap" :for="$props.for">
      {{ label }}<span v-if="required" class="required" aria-hidden="true">*</span>
    </label>
    <div class="control ant-box" :aria-describedby="error || hint ? messageId : undefined">
      <slot :invalid="!!error" :message-id="messageId" />
    </div>
    <p v-if="error" :id="messageId" class="message error ant-wrap" role="alert">{{ error }}</p>
    <p v-else-if="hint" :id="messageId" class="message ant-wrap">{{ hint }}</p>
  </div>
</template>

<style scoped>
.form-field {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  min-width: 0;
}

.label {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-sm);
  font-weight: var(--ant-fw-bold);
}

.required {
  margin-left: 2px;
  color: var(--ant-status-danger);
}

.message {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
  line-height: var(--ant-lh-tight);
}

.message.error {
  color: var(--ant-status-danger);
}
</style>
