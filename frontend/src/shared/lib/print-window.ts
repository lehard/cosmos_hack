/**
 * Новая вкладка под печатную форму документа (FR-139): HTML с рамкой и QR
 * рисует сервер (`documents.paper.print_view`, AD-12), интерфейс только
 * показывает его. Вкладку открываем сразу по нажатию — иначе блокировщик
 * всплывающих окон её не пустит, — а наполняем, когда сервер ответил.
 */
export interface PrintWindow {
  /** Показать страницу для печати. */
  write(html: string): void
  /** Закрыть пустую вкладку, если печать не удалась. */
  close(): void
}

/** Открыть пустую вкладку для печатной формы; блокировщик не пустил — вызовы ничего не делают. */
export function openPrintWindow(): PrintWindow {
  const w = window.open('', '_blank')
  return {
    write(html) {
      if (!w) return
      w.document.open()
      w.document.write(html)
      w.document.close()
      w.focus()
    },
    close() {
      w?.close()
    },
  }
}
