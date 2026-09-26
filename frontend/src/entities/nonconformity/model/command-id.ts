/**
 * Идентификатор команды — UUIDv7 клиента (RFC 9562 §5.7; соглашение
 * «Идентификаторы», AD-7): повтор команды с тем же `command_id` возвращает
 * прежний ответ, поэтому id создаётся один раз на решение, а не на попытку.
 */

/**
 * Новый UUIDv7: 48 бит миллисекунд, версия 7, вариант RFC, остальное —
 * криптостойкий случайный шум.
 * @param now — время, мс (для тестов)
 */
export function uuidv7(now: number = Date.now()): string {
  const b = new Uint8Array(16)
  crypto.getRandomValues(b)
  let ms = Math.floor(now)
  for (let i = 5; i >= 0; i--) {
    b[i] = ms % 256
    ms = Math.floor(ms / 256)
  }
  b[6] = (b[6]! & 0x0f) | 0x70
  b[8] = (b[8]! & 0x3f) | 0x80
  const h = [...b].map((x) => x.toString(16).padStart(2, '0')).join('')
  return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`
}
