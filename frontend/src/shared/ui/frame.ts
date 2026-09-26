/**
 * Как рамка виджета встроена в окружение (Д-70, UI-6). Ставит тот, кто
 * размещает виджет: вкладка стола (заголовок панели совпадает с заголовком
 * страницы или вкладки — второй не нужен) и окно записи (у окна свой заголовок
 * и вкладки, рамка — без обводки).
 */
import type { InjectionKey } from 'vue'

export interface FrameContext {
  /** Не показывать заголовок рамки: его уже показывает страница, вкладка или окно. */
  hideTitle?: boolean
  /** Без обводки и тени — виджет внутри окна записи. */
  plain?: boolean
}

export const WIDGET_FRAME_CONTEXT: InjectionKey<FrameContext> = Symbol('widget-frame-context')
