/** Типы свойств общих элементов интерфейса (слой shared, FSD). */

/** Столбец таблицы данных (`DataTable`). */
export interface DataColumn {
  /** Ключ поля строки и имя слота ячейки `cell-‹key›`. */
  key: string
  /** Заголовок столбца (уже переведённый текст). */
  title: string
  /** Выравнивание: числа — вправо. */
  align?: 'left' | 'right' | 'center'
  /** Ширина столбца (CSS). */
  width?: string
  /** Коды и номера — моноширинно. */
  mono?: boolean
  /** Одна строка с многоточием и подсказкой вместо переноса. */
  ellipsis?: boolean
}

/** Как элемент обходится с длинной подписью. */
export type TextOverflow = 'ellipsis' | 'wrap'

/** Вид панели секции. */
export type PanelVariant = 'card' | 'subtle' | 'plain'
