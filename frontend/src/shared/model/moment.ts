/**
 * Момент, на который смотрит интерфейс (AD-21, AD-22, AD-37) — состояние
 * интерфейса, поэтому в Pinia (серверных данных здесь нет).
 *
 * - `axis`: `occurred` — «как было» (по умолчанию), `recorded` — «что мы знали»;
 * - `asOf`: null — «сейчас», иначе момент воспроизведения (RFC 3339 UTC).
 *
 * Параметры уходят во все запросы чтения; в воспроизведении действия выключены
 * правилом прав (shared/lib/access) и рамкой виджета. Меняет момент таймлайн
 * (эпик 10) и пульт сценариев (эпик 14).
 */
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { Axis } from '@/shared/api/generated/model'

/** Параметры момента для запросов чтения. */
export interface MomentParams {
  axis: Axis
  as_of?: string
}

export const useMomentStore = defineStore('moment', () => {
  const axis = ref<Axis>('occurred')
  const asOf = ref<string | null>(null)

  /** Воспроизведение — смотрим в прошлое, действия недоступны (FR-4). */
  const isReplay = computed(() => asOf.value !== null)

  /** Параметры для сгенерированных запросов чтения. */
  const params = computed<MomentParams>(() => (asOf.value ? { axis: axis.value, as_of: asOf.value } : { axis: axis.value }))

  /** Перейти к моменту `at` по оси `by`. */
  function travel(at: string, by: Axis = axis.value): void {
    asOf.value = at
    axis.value = by
  }

  /** Вернуться к «сейчас». */
  function goLive(): void {
    asOf.value = null
    axis.value = 'occurred'
  }

  return { axis, asOf, isReplay, params, travel, goLive }
})
