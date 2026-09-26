/**
 * Воспроизведение истории на карте (FR-4, FR-155; AD-22, AD-37).
 *
 * Воспроизведение — это только движение момента `as_of` в useMomentStore:
 * все виджеты перечитывают своё состояние на этот момент теми же запросами,
 * ничего не пишется и никуда не отправляется. Скорость ×1…×1000 — во сколько
 * раз доменное время идёт быстрее настоящего; шаг — раз в `tickMs` настоящего
 * времени (часы — setInterval, в тестах подменяются).
 */
import { onScopeDispose, ref, type Ref } from 'vue'
import { useMomentStore } from '@/shared/model/moment'

/** Скорости воспроизведения (FR-4). */
export const SPEEDS = [1, 10, 100, 1000] as const
export type Speed = (typeof SPEEDS)[number]

/** Момент в формате контракта: RFC 3339 UTC, ровно три знака после секунд. */
export const toMoment = (ms: number): string => new Date(ms).toISOString()

/**
 * Следующий момент воспроизведения.
 * @param atMs — текущий момент, мс
 * @param speed — скорость
 * @param realMs — сколько прошло настоящего времени
 * @param endMs — конец истории («сейчас» сервера)
 */
export function advance(atMs: number, speed: Speed, realMs: number, endMs: number): { at: number; ended: boolean } {
  const at = atMs + speed * realMs
  return at >= endMs ? { at: endMs, ended: true } : { at, ended: false }
}

/** Границы истории, мс. */
export interface PlaybackRange {
  from: number
  to: number
}

/** Управление воспроизведением. */
export interface Playback {
  playing: Readonly<Ref<boolean>>
  speed: Ref<Speed>
  /** Пуск: из «сейчас» — с начала истории (прогона), иначе — с текущего момента. */
  play(): void
  pause(): void
  /** Переход к моменту (пауза). */
  jump(at: string | number): void
  /** Вернуться к «сейчас». */
  goLive(): void
}

/**
 * Воспроизведение над хранилищем момента.
 * @param range — границы истории; null — неизвестны (пуск невозможен)
 * @param tickMs — шаг настоящего времени
 */
export function usePlayback(range: () => PlaybackRange | null, tickMs = 1000): Playback {
  const moment = useMomentStore()
  const playing = ref(false)
  const speed = ref<Speed>(100)
  let timer: ReturnType<typeof setInterval> | null = null

  const stop = () => {
    if (timer !== null) clearInterval(timer)
    timer = null
    playing.value = false
  }

  function tick() {
    const r = range()
    if (!r || !moment.asOf) return stop()
    const next = advance(Date.parse(moment.asOf), speed.value, tickMs, r.to)
    if (next.ended) {
      // Дошли до «сейчас» — дальше живой режим (FR-4: «сейчас» ↔ воспроизведение).
      stop()
      moment.goLive()
      return
    }
    moment.travel(toMoment(next.at))
  }

  function play() {
    const r = range()
    if (!r || playing.value) return
    if (!moment.asOf || Date.parse(moment.asOf) >= r.to) moment.travel(toMoment(r.from))
    playing.value = true
    timer = setInterval(tick, tickMs)
  }

  function jump(at: string | number) {
    stop()
    moment.travel(typeof at === 'number' ? toMoment(at) : at)
  }

  function goLive() {
    stop()
    moment.goLive()
  }

  onScopeDispose(stop)
  return { playing, speed, play, pause: stop, jump, goLive }
}
