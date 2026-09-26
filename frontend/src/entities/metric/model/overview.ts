/**
 * Раскладка показателей раздела «Аналитика» (кейс §2.4, §5.2; FR-86…FR-89):
 * строки `analytics.overview.read` → разделы экрана. Чистые функции, без
 * пересчёта чисел: сервер считает, экран раскладывает (AD-21, AD-45).
 *
 * Правила раскладки:
 * - дефекты и изделия с дефектами — две отдельные колонки, не сумма (кейс §5.2);
 * - входной брак, оборудование, исполнители и гипотезы — четыре графы
 *   раздельного учёта (FR-87), входной брак не попадает в производственные дефекты;
 * - повторные и незавершённые операции — отдельно (кейс §5.2);
 * - сравнение сопоставимых работ — таблица «срез × показатель» по одному
 *   измерению (исполнитель, оборудование…), а не рейтинг;
 * - строка, которую не удалось отнести к разделу, не теряется — раздел «Прочее».
 */
import type { ContributionRow, MetricRow, MetricSlice, MetricSliceDimension, MetricValue } from '@/shared/api/generated/model'
import { ACCOUNTS, accountOf, type Account } from './catalog'

export type { MetricRow, MetricSlice }

/** Таблица сравнения сопоставимых работ по одному измерению. */
export interface ComparisonTable {
  dimension: MetricSliceDimension
  /** Показатели — столбцы. */
  columns: MetricRow[]
  /** Срезы — строки; нет значения у показателя — null («не передано», не ноль). */
  rows: { key: string; label: string; cells: (MetricSlice | null)[] }[]
}

/** Разделы экрана «Аналитика». */
export interface OverviewModel {
  inspection: MetricRow[]
  /** Дефекты (число дефектов). */
  defects: MetricRow[]
  /** Изделия с дефектами (число изделий). */
  items: MetricRow[]
  /** Повторные и незавершённые операции. */
  operations: MetricRow[]
  accounts: Record<Account, MetricRow[]>
  causes: MetricRow[]
  time: MetricRow[]
  comparison: ComparisonTable[]
  other: MetricRow[]
}

const isOperationsMetric = (id: string) => /rework|unfinished|repeat|representation/.test(id)
const isItemsMetric = (id: string) => /items/.test(id)

/**
 * Разложить строки показателей по разделам.
 * @param rows — `items` ответа `analytics.overview.read`
 */
export function toOverviewModel(rows: readonly MetricRow[]): OverviewModel {
  const m: OverviewModel = {
    inspection: [],
    defects: [],
    items: [],
    operations: [],
    accounts: Object.fromEntries(ACCOUNTS.map((a) => [a, []])) as unknown as Record<Account, MetricRow[]>,
    causes: [],
    time: [],
    comparison: [],
    other: [],
  }
  const comparisonRows: MetricRow[] = []
  for (const row of rows) {
    const account = accountOf(row)
    if (account) {
      m.accounts[account].push(row)
      continue
    }
    switch (row.group) {
      case 'inspection':
        m.inspection.push(row)
        break
      case 'defects':
        if (isOperationsMetric(row.metric_id)) m.operations.push(row)
        else if (isItemsMetric(row.metric_id)) m.items.push(row)
        else m.defects.push(row)
        break
      case 'causes':
        m.causes.push(row)
        break
      case 'time':
        m.time.push(row)
        break
      case 'comparison':
        comparisonRows.push(row)
        break
      default:
        m.other.push(row)
    }
  }
  // Сравнение: сопоставимые работы + показатели исполнителей и оборудования с
  // тем же измерением среза (ошибки считаются на те же работы).
  const peers = rows.filter((r) => r.group === 'people' || r.group === 'equipment')
  m.comparison = comparisonTables(comparisonRows, peers)
  // Строка сравнения без срезов не попадёт в таблицу — показываем её в «Прочем».
  for (const r of comparisonRows) if (!r.slices.length) m.other.push(r)
  return m
}

/** Таблицы сравнения: по одной на измерение срезов строк сравнения. */
function comparisonTables(base: readonly MetricRow[], peers: readonly MetricRow[]): ComparisonTable[] {
  const dims: MetricSliceDimension[] = []
  for (const r of base) for (const s of r.slices) if (!dims.includes(s.dimension)) dims.push(s.dimension)
  return dims.map((dimension) => {
    const has = (r: MetricRow) => r.slices.some((s) => s.dimension === dimension)
    const columns = [...base.filter(has), ...peers.filter(has)]
    const order: { key: string; label: string }[] = []
    for (const c of columns)
      for (const s of c.slices) if (s.dimension === dimension && !order.some((o) => o.key === s.key)) order.push({ key: s.key, label: s.label })
    return {
      dimension,
      columns,
      rows: order.map((o) => ({ ...o, cells: columns.map((c) => c.slices.find((s) => s.dimension === dimension && s.key === o.key) ?? null) })),
    }
  })
}

/** Срезы строки, сгруппированные по измерению (в порядке прихода). */
export function slicesByDimension(row: Pick<MetricRow, 'slices'>): { dimension: MetricSliceDimension; slices: MetricSlice[] }[] {
  const out: { dimension: MetricSliceDimension; slices: MetricSlice[] }[] = []
  for (const s of row.slices) {
    const g = out.find((x) => x.dimension === s.dimension)
    if (g) g.slices.push(s)
    else out.push({ dimension: s.dimension, slices: [s] })
  }
  return out
}

/** Итог проверки «сумма вкладов = итог». */
export type SumCheck = 'match' | 'mismatch'

/**
 * Сверка раскрытия (AD-45: показатель — сумма вкладов изделий): для штучных
 * показателей сумма строк вклада должна совпасть с итогом. null — сверять
 * нельзя: доли и длительности не складываются, или загружены не все страницы.
 * @param total — итог показателя
 * @param rows — все строки вклада
 * @param complete — загружены все страницы
 */
export function sumCheck(total: MetricValue, rows: readonly Pick<ContributionRow, 'value'>[], complete: boolean): SumCheck | null {
  if (!complete || total.unit !== 'pcs' || rows.some((r) => r.value.unit !== 'pcs' || r.value.scale !== total.scale)) return null
  const sum = rows.reduce((acc, r) => acc + r.value.value, 0)
  return sum === total.value ? 'match' : 'mismatch'
}
