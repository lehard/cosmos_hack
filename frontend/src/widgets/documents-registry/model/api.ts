/**
 * Данные и команды реестра документов (FR-65, FR-66, FR-136, FR-139; AD-12,
 * AD-43): список с отбором (`documents.document.list` без объекта),
 * документ с маршрутом (`documents.document.read`), отрисовка
 * (`documents.document.render`), подпись этапа (`documents.document.sign` —
 * пакет DSSE от расширения «Главный — подпись»; демо-профиль без ключа —
 * `documents.signature.record`, Д-30), отказ с замечанием
 * (`documents.signature.decline`), печать с QR (`documents.paper.print` →
 * печатная форма `documents.paper.print_view`). Ключи кэша — по соглашению
 * shared/api/keys.ts: SSE по документу сбрасывает `['document', id]` и списки.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import {
  documentsDocumentList,
  documentsDocumentRead,
  documentsDocumentRender,
  documentsDocumentSign,
  documentsPaperPrint,
  documentsPaperPrintView,
  documentsSignatureDecline,
  documentsSignatureRecord,
} from '@/shared/api/generated/client'
import type { DocumentSummary, DocumentView, DsseEnvelope } from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import { backendModeOf } from '@/shared/api/response'
import { newCommandId } from '@/shared/lib/command-id'
import { useMomentStore } from '@/shared/model/moment'
import { useSession } from '@/entities/session'
import { filterParams, type RegistryFilter } from './registry'

export type { DocumentSummary, DocumentView }

const keys = entityKeys('document')

/** Реестр документов с отбором. */
export function useDocumentRegistry(filter: MaybeRefOrGetter<RegistryFilter>) {
  const moment = useMomentStore()
  const params = computed(() => ({ ...filterParams(toValue(filter)), ...moment.params, limit: 500 }))
  const q = useQuery({
    queryKey: computed(() => keys.list('registry', params.value)),
    queryFn: ({ signal }) => documentsDocumentList(params.value, { signal }),
  })
  return {
    query: q,
    rows: computed<DocumentSummary[] | null>(() => q.data.value?.data.items ?? null),
    mode: computed(() => backendModeOf(q.data.value)),
  }
}

/** Документ с маршрутом подписей; version 0 — последняя. */
export function useDocument(id: MaybeRefOrGetter<string>, version: MaybeRefOrGetter<number> = 0) {
  const moment = useMomentStore()
  const params = computed(() => ({ ...moment.params, ...(toValue(version) ? { version: toValue(version) } : {}) }))
  return useQuery({
    queryKey: computed(() => keys.one(toValue(id), params.value)),
    queryFn: ({ signal }) => documentsDocumentRead(toValue(id), params.value, { signal }),
    enabled: computed(() => Boolean(toValue(id))),
  })
}

/** Каноническая отрисовка HTML версии (AD-12). */
export function useRendering(id: MaybeRefOrGetter<string>, version: MaybeRefOrGetter<number>) {
  return useQuery({
    queryKey: computed(() => keys.one(toValue(id), 'rendering', toValue(version))),
    queryFn: ({ signal }) => documentsDocumentRender(toValue(id), toValue(version) ? { version: toValue(version) } : {}, { signal }),
    enabled: computed(() => Boolean(toValue(id))),
    staleTime: Infinity,
  })
}

/** Команда над документом: подпись, отказ, печать. */
export type DocumentCommand =
  | { kind: 'sign'; doc: DocumentView; stage: number; signature?: DsseEnvelope }
  | { kind: 'sign_demo'; doc: DocumentView; stage: number }
  | { kind: 'decline'; doc: DocumentView; stage: number; comment: string }
  | { kind: 'print'; doc: DocumentView }

/** Команды окна документа (один command_id на намерение, AD-7); после — перечитать документ и реестр. */
export function useDocumentCommand() {
  const qc = useQueryClient()
  const session = useSession()
  return useMutation({
    mutationFn: async (c: DocumentCommand) => {
      const s = session.data.value?.data
      const header = {
        command_id: newCommandId(),
        basis_seq: c.doc.basis_seq,
        policy_seq: s?.policy_seq ?? 0,
        ...(s?.workplace?.id ? { workplace_id: s.workplace.id } : {}),
      }
      const id = c.doc.document_id
      switch (c.kind) {
        case 'sign':
          return documentsDocumentSign(id, { ...header, version: c.doc.version, stage: c.stage, doc_digest: c.doc.doc_digest, ...(c.signature ? { signature: c.signature } : {}) })
        case 'sign_demo':
          return documentsSignatureRecord(id, { ...header, version: c.doc.version, stage: c.stage, doc_digest: c.doc.doc_digest })
        case 'decline':
          return documentsSignatureDecline(id, { ...header, version: c.doc.version, stage: c.stage, doc_digest: c.doc.doc_digest, comment: c.comment })
        case 'print':
          return documentsPaperPrint(id, { ...header, version: c.doc.version })
      }
    },
    onSuccess: (_r, c) => {
      void qc.invalidateQueries({ queryKey: keys.one(c.doc.document_id) })
      void qc.invalidateQueries({ queryKey: keys.list() })
    },
  })
}

/** Печатная форма версии: HTML с рамкой и QR (FR-139). */
export async function fetchPrintView(id: string, version: number): Promise<string> {
  return (await documentsPaperPrintView(id, { version })).data.html
}
