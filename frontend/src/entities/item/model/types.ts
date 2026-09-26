/**
 * Данные паспорта изделия (эпик 11; FR-42, FR-43, FR-45, FR-46, FR-140) —
 * сгенерированные типы контракта (contracts/openapi.yaml → shared/api/generated):
 * `item.passport.read`, `item.history.list` (журнал изменений паспорта),
 * `item.genealogy.read`. Руками здесь только псевдонимы и сужения.
 *
 * Необязательные поля в ответе отсутствуют вовсе — читаются как `?? null`.
 * Смысл — только контрактный (NFR-UI-4): «оценка невозможна» ≠ «годно»,
 * «заблокировано» ≠ «признано дефектным», ручная отметка ≠ данные станка.
 */
import type {
  ItemPassport,
  ItemStatus,
  PassportEntry,
  PassportEntryKind,
  PassportEntryReliability,
  PassportEntrySourceKind,
  ItemSignature,
  ItemSignatureCheck,
  ItemSignatureClass,
} from '@/shared/api/generated/model'

export type {
  GenealogyNode,
  GenealogyNodeKind,
  ItemCarrier,
  ItemCarrierCarrierType,
  ItemCarrierState,
  ItemDocumentRef,
  ItemDocumentRefStatus,
  ItemGenealogy,
  ItemHistory,
  ItemHistoryEntry,
  ItemPassport,
  ItemPassportIdentification,
  ItemSignature,
  ItemSignatureCheck,
  ItemSignatureClass,
  ItemStatus,
  ItemZone,
  PassportEntry,
  PassportEntryKind,
} from '@/shared/api/generated/model'

/** Вид источника факта (FR-140). */
export type SourceKind = PassportEntrySourceKind
/** Надёжность факта по источнику (FR-140). */
export type Reliability = PassportEntryReliability

/** Пять осей статуса изделия (PRD §3b) — без сводного статуса. */
export type ItemStatuses = Omit<ItemStatus, 'summary'>

/** Значение оси «Состояние качества». */
export type QualityStatus = ItemStatus['quality']

/** Слой записи на экране — вид записи в журнале (AD-2). */
export type RecordLayer = PassportEntryKind

/** Запись, у которой можно показать пометку источника. */
export type SourcedRecord = Pick<PassportEntry, 'kind'> & { source_kind?: SourceKind; reliability?: Reliability }

/** Итог проверки подписи. */
export type SignatureCheck = ItemSignatureCheck

/** Класс происхождения подписи. */
export type SignatureClass = ItemSignatureClass

/** Подпись записи. */
export type RecordSignature = ItemSignature

/** Паспорт, из которого строится шапка. */
export type PassportHead = Pick<ItemPassport, 'item_id' | 'label' | 'item_type_id' | 'item_revision' | 'process_version' | 'status' | 'identification'>
