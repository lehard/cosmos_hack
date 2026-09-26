/**
 * Геометрия контрольной карты Шухарта (FR-5, FR-89; ГОСТ Р ИСО 7870-2): точки
 * по порядку выполнений, центральная линия и контрольные границы — как пришли
 * с сервера (`analytics.control_chart.read`). Границы и признак выхода
 * (`out_of_control`) считает сервер; здесь только перевод в координаты SVG.
 */
import type { ControlChart, ControlChartPoint } from '@/shared/api/generated/model'
import { numeric } from './value'

export type { ControlChart, ControlChartPoint }

/** Размеры области рисования. */
export interface ChartBox {
  width: number
  height: number
  /** Поля: слева под подписи оси, сверху/снизу/справа — воздух. */
  left: number
  right: number
  top: number
  bottom: number
}

export const DEFAULT_BOX: ChartBox = { width: 640, height: 220, left: 56, right: 16, top: 12, bottom: 24 }

/** Точка в координатах SVG. */
export interface PlotPoint {
  x: number
  y: number
  point: ControlChartPoint
  index: number
}

/** Карта в координатах SVG. */
export interface ChartGeometry {
  box: ChartBox
  points: PlotPoint[]
  /** y центральной линии и границ. */
  center: number
  upper: number
  lower: number
  /** Полилиния ряда: «x,y x,y …». */
  path: string
  /** Пределы оси значений. */
  min: number
  max: number
}

/**
 * Координаты карты. Ось значений охватывает границы и все точки с запасом 10 %,
 * чтобы выход за границу был виден; ось x — порядок выполнений.
 * @param chart — карта контракта
 * @param box — размеры
 */
export function chartGeometry(chart: Pick<ControlChart, 'points' | 'center' | 'upper' | 'lower'>, box: ChartBox = DEFAULT_BOX): ChartGeometry {
  const values = chart.points.map((p) => numeric(p.value))
  const lo = Math.min(numeric(chart.lower), ...values)
  const hi = Math.max(numeric(chart.upper), ...values)
  const pad = (hi - lo || Math.abs(hi) || 1) * 0.1
  const min = lo - pad
  const max = hi + pad
  const plotW = box.width - box.left - box.right
  const plotH = box.height - box.top - box.bottom
  const y = (v: number) => round(box.top + ((max - v) / (max - min)) * plotH)
  const n = chart.points.length
  const x = (i: number) => round(box.left + (n <= 1 ? plotW / 2 : (i / (n - 1)) * plotW))
  const points = chart.points.map((point, index) => ({ x: x(index), y: y(numeric(point.value)), point, index }))
  return {
    box,
    points,
    center: y(numeric(chart.center)),
    upper: y(numeric(chart.upper)),
    lower: y(numeric(chart.lower)),
    path: points.map((p) => `${p.x},${p.y}`).join(' '),
    min,
    max,
  }
}

const round = (v: number) => Math.round(v * 10) / 10
