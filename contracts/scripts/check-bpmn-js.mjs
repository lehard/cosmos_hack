// BPMN нормативного слоя открывается в bpmn-js 18.30.1 без предупреждений импорта (эпик 00, «Что ожидаем в итоге»).
// Настоящий Viewer из дистрибутива bpmn-js в jsdom; недостающие в jsdom SVG-API — минимальные заглушки
// (они влияют только на отрисовку, не на разбор XML и DI). Предупреждения импорта — те же, что видит пользователь.
import fs from 'node:fs';
import path from 'node:path';
import { createRequire } from 'node:module';
import { JSDOM, VirtualConsole } from 'jsdom';
import { CONTRACTS, REPO, readJson, rel, fail, report, walk } from './lib.mjs';

const require = createRequire(import.meta.url);
const viewerSrc = fs.readFileSync(require.resolve('bpmn-js/dist/bpmn-viewer.production.min.js'), 'utf8');
const ant = readJson(path.join(CONTRACTS, 'bpmn-ext/ant.json'));

function makeWindow() {
  const vc = new VirtualConsole(); // «not implemented» jsdom (canvas) — не предупреждения bpmn-js
  const dom = new JSDOM('<!DOCTYPE html><div id="c"></div>', { pretendToBeVisual: true, runScripts: 'outside-only', virtualConsole: vc });
  const w = dom.window;
  w.structuredClone = globalThis.structuredClone;
  class SVGMatrix { constructor() { Object.assign(this, { a: 1, b: 0, c: 0, d: 1, e: 0, f: 0 }); } multiply() { return new SVGMatrix(); } inverse() { return new SVGMatrix(); } translate() { return new SVGMatrix(); } scale() { return new SVGMatrix(); } rotate() { return new SVGMatrix(); } }
  w.SVGMatrix = SVGMatrix;
  const tr = (m) => ({ matrix: m || new SVGMatrix(), setMatrix() {}, setTranslate() {}, setScale() {}, setRotate() {} });
  const tlist = () => { const it = []; return { get numberOfItems() { return it.length; }, appendItem(t) { it.push(t); return t; }, clear() { it.length = 0; }, initialize(t) { it.length = 0; it.push(t); return t; }, getItem(i) { return it[i]; }, consolidate() { return it[0] || null; }, createSVGTransformFromMatrix: tr }; };
  const P = w.SVGElement.prototype;
  Object.defineProperty(P, 'transform', { get() { if (!this.__t) this.__t = { baseVal: tlist() }; return this.__t; } });
  P.getBBox = () => ({ x: 0, y: 0, width: 10, height: 10 });
  P.getComputedTextLength = () => 10;
  P.getScreenCTM = () => new SVGMatrix();
  P.getCTM = () => new SVGMatrix();
  const S = w.SVGSVGElement ? w.SVGSVGElement.prototype : P;
  S.createSVGMatrix = () => new SVGMatrix();
  S.createSVGTransform = () => tr();
  S.createSVGTransformFromMatrix = tr;
  S.createSVGPoint = () => ({ x: 0, y: 0, matrixTransform() { return { x: 0, y: 0 }; } });
  w.eval(viewerSrc);
  return w;
}

let n = 0;
for (const f of walk(path.join(REPO, 'normative/process'), (p) => p.endsWith('.bpmn'))) {
  const w = makeWindow();
  const viewer = new w.BpmnJS({ container: w.document.getElementById('c'), moddleExtensions: { ant } });
  try {
    const { warnings } = await viewer.importXML(fs.readFileSync(f, 'utf8'));
    for (const x of warnings) fail(rel(f), `bpmn-js: ${x.message}`);
    n += viewer.get('elementRegistry').getAll().length;
  } catch (e) { fail(rel(f), `bpmn-js не открыл: ${e.message.split('\n')[0]}`); }
}
report(`BPMN открывается в bpmn-js 18.30.1 без предупреждений (${n} элементов на холсте)`);
