/**
 * Источник данных виджета «cause-analysis»: группы несоответствий «вид
 * дефекта × операция × оборудование» (операция `analysis.group.list`).
 */
import { useNcGroups } from '@/entities/incident'

/** Данные виджета «cause-analysis». */
export const useNcGroupsSource = useNcGroups
