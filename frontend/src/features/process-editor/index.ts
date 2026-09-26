/**
 * Фича «Редактор процесса» (эпик 39; FR-25, FR-12, AD-17) — публичный вход
 * (FSD): модельер bpmn-js с расширением `urn:ant:bpmn-ext:1` и панелью наших
 * свойств; провайдер панели сверяется с дескриптором тестом.
 */
export * from './model/properties'
export * from './model/editor'
export { default as ProcessModeler } from './ui/ProcessModeler.vue'
export { default as PropertiesPanel } from './ui/PropertiesPanel.vue'
