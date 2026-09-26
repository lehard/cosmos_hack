/**
 * Виджет «documents-registry» — публичный вход (FSD): реестр документов
 * раздела «Документы» столов (FR-65, FR-66, FR-139; AD-12, AD-43) и окно
 * документа для реестра окна записи (app/record, тип `document`).
 * Оболочка грузит виджет через widgets/registry.ts.
 */
export { default } from './ui/DocumentsRegistryWidget.vue'
export { default as DocumentDrawer } from './ui/DocumentDrawer.vue'
