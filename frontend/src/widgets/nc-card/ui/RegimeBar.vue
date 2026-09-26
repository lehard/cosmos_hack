<script setup lang="ts">
/**
 * Полоса режима на ленте «Как было дело» (UI-24): коридор уставки и наблюдённый
 * диапазон параметра операции на одной шкале; часть за коридором — красным,
 * подпись крупно — «макс. 182 А — на 12 А выше допуска». Числа — из
 * `NCRecordRef.reading` (значение = число × 10^−scale), ничего не досчитывается
 * за сервер, кроме разницы с границей уставки. Ширина — по контейнеру, без
 * горизонтальной прокрутки.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { codeToKey } from '@/shared/i18n'
import type { NCParameterReading } from '@/entities/nonconformity'

const props = defineProps<{ reading: NCParameterReading }>()
const { t, te, n } = useI18n()

const k = computed(() => 10 ** -props.reading.scale)
const v = (x: number | undefined) => (x == null ? null : x * k.value)
const r = computed(() => ({
  setMin: v(props.reading.setpoint_min),
  setMax: v(props.reading.setpoint_max),
  nominal: v(props.reading.setpoint_nominal),
  obsMin: v(props.reading.observed_min),
  obsMax: v(props.reading.observed_max),
}))

/** Единицы UCUM → текст с числом (`common.units.*`); неизвестная — как есть. */
const UNIT_KEY: Record<string, string> = { A: 'ampere', V: 'volt', mm: 'mm', um: 'um', s: 'seconds', min: 'minutes', Cel: 'celsius', 'N.m': 'newtonMeter', '%': 'percent' }
const fmt = (x: number) => n(x, { maximumFractionDigits: props.reading.scale })
const withUnit = (x: number) => {
  const key = UNIT_KEY[props.reading.unit]
  return key ? t(`common.units.${key}`, { value: fmt(x) }) : `${fmt(x)} ${props.reading.unit}`
}
const parameter = computed(() => {
  const key = `ncCard.regime.parameter.${codeToKey(props.reading.parameter)}`
  return te(key) ? t(key) : null
})

/** Шкала: от меньшего из границ и наблюдений до большего, с запасом 15 % по краям. */
const scale = computed(() => {
  const xs = [r.value.setMin, r.value.setMax, r.value.obsMin, r.value.obsMax, r.value.nominal].filter((x): x is number => x != null)
  if (!xs.length) return null
  const lo = Math.min(...xs)
  const hi = Math.max(...xs)
  const pad = Math.max((hi - lo) * 0.15, 1)
  return { lo: lo - pad, hi: hi + pad }
})
const pos = (x: number) => (scale.value ? ((x - scale.value.lo) / (scale.value.hi - scale.value.lo)) * 100 : 0)
const span = (a: number, b: number) => ({ left: `${pos(Math.min(a, b))}%`, width: `${Math.max(pos(Math.max(a, b)) - pos(Math.min(a, b)), 0.8)}%` })

const corridor = computed(() => (r.value.setMin != null && r.value.setMax != null ? span(r.value.setMin, r.value.setMax) : null))
const observed = computed(() => {
  const a = r.value.obsMin ?? r.value.obsMax
  const b = r.value.obsMax ?? r.value.obsMin
  return a != null && b != null ? { a, b } : null
})
/** Части наблюдения внутри коридора и за ним. */
const parts = computed(() => {
  const o = observed.value
  if (!o) return []
  const lo = r.value.setMin ?? -Infinity
  const hi = r.value.setMax ?? Infinity
  const out: { style: Record<string, string>; out: boolean }[] = []
  if (o.a < lo) out.push({ style: span(o.a, Math.min(o.b, lo)), out: true })
  const inA = Math.max(o.a, lo)
  const inB = Math.min(o.b, hi)
  if (inB >= inA) out.push({ style: span(inA, inB), out: false })
  if (o.b > hi) out.push({ style: span(Math.max(o.a, hi), o.b), out: true })
  return out
})

/** Итог крупно: насколько вышли за уставку; в пределах — так и пишем. */
const verdict = computed(() => {
  const o = observed.value
  if (!o) return null
  if (r.value.setMax != null && o.b > r.value.setMax) return { out: true, text: t('ncCard.regime.above', { value: withUnit(o.b), delta: withUnit(o.b - r.value.setMax) }) }
  if (r.value.setMin != null && o.a < r.value.setMin) return { out: true, text: t('ncCard.regime.below', { value: withUnit(o.a), delta: withUnit(r.value.setMin - o.a) }) }
  return { out: false, text: t('ncCard.regime.within') }
})
const setpointText = computed(() => {
  const { setMin, setMax, nominal } = r.value
  if (setMin != null && setMax != null) return t('ncCard.regime.setpointRange', { from: fmt(setMin), to: withUnit(setMax) }) + (nominal != null ? ` · ${t('ncCard.regime.nominal', { value: withUnit(nominal) })}` : '')
  return nominal != null ? t('ncCard.regime.nominal', { value: withUnit(nominal) }) : null
})
const observedText = computed(() => {
  const o = observed.value
  if (!o) return null
  return o.a === o.b ? withUnit(o.a) : t('ncCard.regime.range', { from: fmt(o.a), to: withUnit(o.b) })
})
</script>

<template>
  <div v-if="scale" class="regime" :data-out="verdict?.out || undefined" data-testid="regime-bar">
    <p v-if="verdict" class="verdict ant-wrap" data-testid="regime-verdict">
      <template v-if="parameter">{{ parameter }}: </template>{{ verdict.text }}
    </p>
    <div class="track" aria-hidden="true">
      <span v-if="corridor" class="corridor" :style="corridor" />
      <span v-for="(p, i) in parts" :key="i" class="observed" :data-out="p.out || undefined" :style="p.style" />
    </div>
    <p class="legend ant-wrap">
      <span v-if="setpointText" class="legend-item"><span class="swatch corridor-swatch" />{{ setpointText }}</span>
      <span v-if="observedText" class="legend-item"><span class="swatch observed-swatch" :data-out="verdict?.out || undefined" />{{ t('ncCard.regime.observed', { value: observedText }) }}</span>
    </p>
  </div>
</template>

<style scoped>
.regime {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  min-width: 0;
  margin-top: var(--ant-space-1);
  padding: var(--ant-space-2) var(--ant-space-3);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface);
}

p {
  margin: 0;
}

.verdict {
  font-weight: var(--ant-fw-bold);
}

.regime[data-out] .verdict {
  color: var(--ant-status-danger-text);
}

/* Шкала: коридор уставки — мягкий зелёный, наблюдение — полоса поверх, за коридором — красным. */
.track {
  position: relative;
  height: 18px;
  overflow: hidden;
  border-radius: var(--ant-radius-sm);
  background: var(--ant-surface-subtle);
}

.corridor {
  position: absolute;
  top: 0;
  bottom: 0;
  background: var(--ant-status-success-soft);
  border-right: 2px solid var(--ant-status-success);
  border-left: 2px solid var(--ant-status-success);
}

.observed {
  position: absolute;
  top: 5px;
  bottom: 5px;
  border-radius: var(--ant-radius-sm);
  background: var(--ant-text-2);
}

.observed[data-out] {
  background: var(--ant-status-danger);
}

.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 2px var(--ant-space-4);
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

.legend-item {
  display: inline-flex;
  gap: var(--ant-space-1);
  align-items: center;
  min-width: 0;
}

.swatch {
  flex: none;
  width: 12px;
  height: 8px;
  border-radius: 2px;
}

.corridor-swatch {
  border: 1px solid var(--ant-status-success);
  background: var(--ant-status-success-soft);
}

.observed-swatch {
  background: var(--ant-text-2);
}

.observed-swatch[data-out] {
  background: var(--ant-status-danger);
}
</style>
