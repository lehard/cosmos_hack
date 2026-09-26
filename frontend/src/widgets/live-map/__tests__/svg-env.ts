/**
 * Заглушки SVG-API, которых нет в happy-dom, — для настоящего просмотрщика bpmn-js
 * в юнит-тестах. Влияют только на геометрию отрисовки (матрицы, размеры текста),
 * не на разбор XML, реестр элементов и наложения. Тот же приём, что в
 * contracts/scripts/check-bpmn-js.mjs (jsdom).
 */
class Matrix {
  a = 1; b = 0; c = 0; d = 1; e = 0; f = 0
  multiply(): Matrix { return new Matrix() }
  inverse(): Matrix { return new Matrix() }
  translate(): Matrix { return new Matrix() }
  scale(): Matrix { return new Matrix() }
  rotate(): Matrix { return new Matrix() }
}

/** Поставить заглушки один раз на окно теста. */
export function installSvgStubs(): void {
  const w = window as unknown as Record<string, unknown>
  if (w.__antSvgStubs) return
  w.__antSvgStubs = true
  w.SVGMatrix = Matrix
  const transform = (m?: Matrix) => ({ matrix: m ?? new Matrix(), setMatrix() {}, setTranslate() {}, setScale() {}, setRotate() {} })
  const list = () => {
    const items: unknown[] = []
    return {
      get numberOfItems() { return items.length },
      appendItem(t: unknown) { items.push(t); return t },
      clear() { items.length = 0 },
      initialize(t: unknown) { items.length = 0; items.push(t); return t },
      getItem(i: number) { return items[i] },
      consolidate() { return items[0] ?? null },
      createSVGTransformFromMatrix: transform,
    }
  }
  const proto = window.SVGElement.prototype as unknown as Record<string, unknown>
  // happy-dom объявляет transform у SVGGraphicsElement — перекрываем на обоих прототипах.
  const graphics = (window as unknown as { SVGGraphicsElement?: { prototype: object } }).SVGGraphicsElement?.prototype
  for (const p of [proto, graphics].filter(Boolean) as object[]) {
    Object.defineProperty(p, 'transform', {
      configurable: true,
      get(this: { __t?: unknown }) {
        if (!this.__t) this.__t = { baseVal: list() }
        return this.__t
      },
    })
  }
  proto.getBBox = () => ({ x: 0, y: 0, width: 10, height: 10 })
  proto.getComputedTextLength = () => 10
  proto.getScreenCTM = () => new Matrix()
  proto.getCTM = () => new Matrix()
  const svg = (window.SVGSVGElement?.prototype ?? proto) as unknown as Record<string, unknown>
  svg.createSVGMatrix = () => new Matrix()
  svg.createSVGTransform = () => transform()
  svg.createSVGTransformFromMatrix = transform
  svg.createSVGPoint = () => ({ x: 0, y: 0, matrixTransform: () => ({ x: 0, y: 0 }) })
}
