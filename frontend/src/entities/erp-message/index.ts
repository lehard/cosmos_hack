/**
 * Обмен с внешними системами (1С, Галактика, MES): ключи кэша и каналы обмена
 * для стола администратора — «статус обмена» (эпик 14, FR-30; AD-26: stand-ы кейса).
 *
 * Операция — `erp.channel.list` (contracts/openapi.yaml), сгенерированный клиент;
 * ключ `[erp_message, '@list', 'channels']` перечитывает SSE по сообщениям обмена.
 */
import { useQuery } from '@tanstack/vue-query'
import { erpChannelList } from '@/shared/api/generated/client'
import type { ErpChannel, ErpChannelState } from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'

export type { ErpChannel, ErpChannelState }

export const erpMessageKeys = entityKeys('erp_message')

/** Каналы обмена: состояние, последний обмен, очередь, карантин (`erp.channel.list`). */
export function useErpChannels() {
  return useQuery({
    queryKey: erpMessageKeys.list('channels'),
    queryFn: ({ signal }) => erpChannelList({ signal }),
    retry: false,
    refetchInterval: 15_000,
  })
}
