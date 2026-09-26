/**
 * Источник данных виджета «circumstances»: проекция `analysis.circumstances`
 * (операция `analysis.circumstances.read`, AD-29, FR-153) по несоответствию в
 * разборе — выбранному, из адреса `?nc=` или первому в группе.
 */
import { useCircumstances, useFocusedNc } from '@/entities/incident'

/** Данные виджета «circumstances». */
export function useCircumstancesSource() {
  const ncId = useFocusedNc()
  return { ncId, ...useCircumstances(ncId) }
}
