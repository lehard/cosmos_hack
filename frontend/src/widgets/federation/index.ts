/**
 * Виджет «federation» — раздел «Партнёры и выписки» столов администратора и
 * начальника ОТК (эпик 41; FR-131, FR-132, AD-19, Д-70): предприятия-партнёры
 * с корнями доверия, входящие и исходящие выписки паспорта со статусом
 * происхождения; подробности — в правом окне записи `?open=extract:‹отпечаток›`
 * и `?open=partner:‹код›` (app/record/kinds/ExtractRecord.vue, PartnerRecord.vue).
 */
export { default } from './ui/FederationWidget.vue'
