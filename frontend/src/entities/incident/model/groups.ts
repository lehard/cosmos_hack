/**
 * Группы несоответствий «вид дефекта × операция × оборудование» (стол технолога,
 * «Разбор причин»): порядок строк.
 */
import { toMs } from './timescale'
import type { NcGroup } from './types'

/** Больше несоответствий — выше; при равенстве — свежее. */
export const sortGroups = (gs: readonly NcGroup[]): NcGroup[] =>
  [...gs].sort((a, b) => b.nc_count - a.nc_count || toMs(b.last_found_at) - toMs(a.last_found_at))
