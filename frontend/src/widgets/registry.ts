/**
 * Реестр виджетов (AD-21, NFR-UI-1, NFR-EXT-1): id → заголовок, эпик-владелец,
 * ленивый импорт. Столы ролей (normative/desks/‹роль›.yaml) ссылаются на
 * виджеты только по id отсюда; неизвестный id ловит make check
 * (frontend/scripts/check-shell.mjs).
 *
 * Реестр заполнен заранее всеми виджетами, которые называют эпики 10–15
 * («Владеет»), — фронт-эпики наполняют готовые папки widgets/‹id›/ и сюда
 * не пишут. Новый виджет — новая строка здесь и папка с index.ts.
 *
 * Формат строк важен: check-shell.mjs читает этот файл синтаксическим деревом
 * TypeScript (ключ — строковый литерал, titleKey — строка, epic — число,
 * load — `() => import('./‹id›')`).
 */
import type { Component } from 'vue'

/** Описание виджета в реестре. */
export interface WidgetDefinition {
  /** Ключ текста заголовка (ru.json / ru.shell.json). */
  titleKey: string
  /** Эпик, который наполняет виджет. */
  epic: number
  /** Ленивая загрузка компонента: виджеты не утяжеляют вход. */
  load: () => Promise<{ default: Component }>
  /**
   * Виджет занимает всю высоту раздела стола (живая карта): страница не
   * прокручивается, виджет растягивается на оставшуюся высоту окна.
   */
  fill?: boolean
}

export const widgetRegistry = {
  // ── эпик 10: Живая карта и стол руководителя ──
  'live-map': { titleKey: 'liveMap.title', epic: 10, load: () => import('./live-map'), fill: true },
  'map-timeline': { titleKey: 'widgets.mapTimeline', epic: 10, load: () => import('./map-timeline') },
  'posts': { titleKey: 'liveMap.posts.title', epic: 10, load: () => import('./posts') },
  'attention': { titleKey: 'liveMap.attention.title', epic: 10, load: () => import('./attention') },
  'alerts-feed': { titleKey: 'liveMap.alerts.title', epic: 10, load: () => import('./alerts-feed') },
  // ── эпик 11: Паспорт изделия и решение ──
  'item-passport': { titleKey: 'desks.passport', epic: 11, load: () => import('./item-passport') },
  'nc-card': { titleKey: 'desks.ncCard', epic: 11, load: () => import('./nc-card') },
  'decision-queue': { titleKey: 'desks.decisionQueue', epic: 11, load: () => import('./decision-queue') },
  'decision-panel': { titleKey: 'decisions.panelTitle', epic: 11, load: () => import('./decision-panel') },
  'decision-request-card': { titleKey: 'desks.decisionCard', epic: 11, load: () => import('./decision-request-card') },
  // ── эпик 12: Разбор и область риска ──
  'cause-analysis': { titleKey: 'desks.causeAnalysis', epic: 12, load: () => import('./cause-analysis') },
  'circumstances': { titleKey: 'desks.circumstances', epic: 12, load: () => import('./circumstances') },
  'common-factors': { titleKey: 'desks.commonFactors', epic: 12, load: () => import('./common-factors') },
  'hypothesis': { titleKey: 'ncCard.hypotheses.title', epic: 12, load: () => import('./hypothesis') },
  'risk-scope': { titleKey: 'riskScope.title', epic: 12, load: () => import('./risk-scope') },
  'process-versions': { titleKey: 'desks.process', epic: 12, load: () => import('./process-versions') },
  // ── эпик 13: Участок, терминал и задачи ──
  'station-posts': { titleKey: 'desks.station', epic: 13, load: () => import('./station-posts') },
  'people-equipment': { titleKey: 'desks.peopleAndEquipment', epic: 13, load: () => import('./people-equipment') },
  'tasks': { titleKey: 'desks.tasks', epic: 13, load: () => import('./tasks') },
  'shift-assignments': { titleKey: 'desks.shift', epic: 13, load: () => import('./shift-assignments') },
  'performer-terminal': { titleKey: 'desks.terminal', epic: 13, load: () => import('./performer-terminal') },
  // ── эпик 14: Администрирование, аудит и тестовые сценарии ──
  'access-admin': { titleKey: 'desks.access', epic: 14, load: () => import('./access-admin') },
  'sources': { titleKey: 'desks.sources', epic: 14, load: () => import('./sources') },
  'quarantine': { titleKey: 'desks.quarantine', epic: 14, load: () => import('./quarantine') },
  'system-health': { titleKey: 'desks.components', epic: 14, load: () => import('./system-health') },
  'scenario-console': { titleKey: 'desks.testScenarios', epic: 14, load: () => import('./scenario-console') },
  'verification-board': { titleKey: 'desks.verificationBoard', epic: 14, load: () => import('./verification-board') },
  'integrity-reports': { titleKey: 'audit.verifier.title', epic: 14, load: () => import('./integrity-reports') },
  'critical-actions-log': { titleKey: 'desks.criticalActionsLog', epic: 14, load: () => import('./critical-actions-log') },
  'security-events': { titleKey: 'audit.securityEvents.title', epic: 14, load: () => import('./security-events') },
  'grants-history': { titleKey: 'widgets.grantsHistory', epic: 14, load: () => import('./grants-history') },
  // ── эпик 15: Аналитика ──
  'metric-tiles': { titleKey: 'widgets.metricTiles', epic: 15, load: () => import('./metric-tiles') },
  'control-chart': { titleKey: 'widgets.controlChart', epic: 15, load: () => import('./control-chart') },
  'metric-drilldown': { titleKey: 'widgets.metricDrilldown', epic: 15, load: () => import('./metric-drilldown') },
  'analytics-overview': { titleKey: 'desks.analytics', epic: 15, load: () => import('./analytics-overview') },
  'proposals': { titleKey: 'desks.proposals', epic: 15, load: () => import('./proposals') },
  'data-deficit-map': { titleKey: 'desks.dataDeficit', epic: 15, load: () => import('./data-deficit-map') },
  // ── эпик 42: Предложения, меры и карта дефицита данных ──
  'corrective-actions': { titleKey: 'widgets.quality.title', epic: 42, load: () => import('./corrective-actions') },
  // ── эпик 40: Адаптация VisionQC ──
  'vision-adaptation': { titleKey: 'widgets.visionAdaptation.title', epic: 40, load: () => import('./vision-adaptation') },
} satisfies Record<string, WidgetDefinition>

/** id виджета из реестра. */
export type WidgetId = keyof typeof widgetRegistry

/** Есть ли такой виджет (стол пришёл с сервера строкой). */
export const isWidgetId = (id: string): id is WidgetId => Object.hasOwn(widgetRegistry, id)

/** Виджет занимает всю высоту раздела стола (признак `fill`). */
export const fillsSection = (id: string): boolean => isWidgetId(id) && (widgetRegistry[id] as WidgetDefinition).fill === true
