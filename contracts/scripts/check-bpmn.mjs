// Проверка BPMN нормативного слоя по дескриптору urn:ant:bpmn-ext:1 (AD-17, FR-11…13; решения Д-4…Д-8):
// разбор bpmn-moddle с дескриптором без предупреждений (как импорт bpmn-js), DI у каждого узла и стрелки,
// поддерживаемое подмножество, нет неявных слияний, step_key у каждого узла, значения и ссылки по rules.yaml,
// нормативные опоры из norm-anchors.yaml, условия по грамматике urn:ant:expr:1, список шагов *.steps.yaml.
//   node check-bpmn.mjs            — проверить;
//   node check-bpmn.mjs --write    — пересобрать normative/process/*.steps.yaml из BPMN.
import fs from 'node:fs';
import path from 'node:path';
import YAML from 'yaml';
import { BpmnModdle } from 'bpmn-moddle';
import { CONTRACTS, REPO, readYaml, readJson, rel, fail, report, walk } from './lib.mjs';

const ext = readJson(path.join(CONTRACTS, 'bpmn-ext/ant.json'));
const rules = readYaml(path.join(CONTRACTS, 'bpmn-ext/rules.yaml'));
const anchors = readYaml(path.join(CONTRACTS, 'bpmn-ext/norm-anchors.yaml'));
const cat = readYaml(path.join(CONTRACTS, 'events/catalog.yaml'));
const nrm = (p) => { const f = walk(path.join(REPO, 'normative'), (x) => x.includes(p)); return f.length ? readYaml(f[0]) : null; };
const classifier = nrm('classifier.v'), policy = nrm('policy.v'), itemTypes = nrm('item-types.yaml'), equipment = nrm('equipment.yaml');
const reactionMap = nrm('reaction-map.v'), templates = nrm('templates.v');
const defectCodes = new Set((classifier?.defect_types || []).map((d) => d.code));
const zones = new Set((itemTypes?.item_types || []).flatMap((t) => (t.zones || []).map((z) => z.zone_id)));
const roles = new Set((policy?.roles || []).map((r) => r.id));
const authorities = new Set((policy?.authorities || []).map((a) => a.id));
const stampKinds = new Set(policy?.stamp_kinds || []);
const qualifications = new Set((policy?.grants?.qualifications || []).map((q) => q.qualification));
const equipmentIds = new Set((equipment?.equipment || []).map((e) => e.id));
const templateIds = new Set((templates?.templates || []).map((t) => t.id));
const reactionRef = reactionMap ? `${reactionMap.id}@${reactionMap.version}` : null;
const E = rules.enums;
const FLOW = new Set(rules.supported_elements.flow_nodes);
const OTHER = new Set(rules.supported_elements.other);
const STEP_RE = new RegExp(rules.step_key.pattern);

// ── язык условий urn:ant:expr:1 ──
export function parseCondition(src, vars) {
  const toks = [];
  const re = /\s*(?:(==|!=|<=|>=|<|>|\(|\))|('(?:[^'])*')|(-?[0-9]+)|([a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)*))/y;
  let i = 0;
  while (i < src.length) {
    if (/^\s*$/.test(src.slice(i))) break;
    re.lastIndex = i;
    const m = re.exec(src);
    if (!m) throw new Error(`неожиданный символ в позиции ${i}: «${src.slice(i, i + 10)}»`);
    i = re.lastIndex;
    if (m[1]) toks.push({ t: m[1] }); else if (m[2]) toks.push({ t: 'str', v: m[2].slice(1, -1) });
    else if (m[3]) toks.push({ t: 'int', v: Number(m[3]) });
    else if (['and', 'or', 'not', 'true', 'false'].includes(m[4])) toks.push({ t: m[4] }); else toks.push({ t: 'id', v: m[4] });
  }
  let p = 0;
  const peek = () => toks[p]?.t, eat = (t) => { if (peek() !== t) throw new Error(`ожидалось «${t}», получено «${peek() ?? 'конец'}»`); return toks[p++]; };
  const OPS = ['==', '!=', '<', '<=', '>', '>='];
  function cmp() {
    const f = eat('id').v;
    const op = toks[p]?.t; if (!OPS.includes(op)) throw new Error(`после поля ${f} нужна операция сравнения`); p++;
    const lit = toks[p++]; if (!lit) throw new Error('нет литерала');
    const v = vars[f]; if (!v) throw new Error(`поле «${f}» не из состояния изделия и решения (rules.yaml conditions.variables)`);
    if (v.type === 'enum') { if (lit.t !== 'str') throw new Error(`${f}: нужен строковый литерал`); if (!v.values.includes(lit.v)) throw new Error(`${f}: значения «${lit.v}» нет в перечне`); if (!['==', '!='].includes(op)) throw new Error(`${f}: для перечисления только == и !=`); }
    else if (v.type === 'boolean') { if (!['true', 'false'].includes(lit.t)) throw new Error(`${f}: нужен true или false`); if (!['==', '!='].includes(op)) throw new Error(`${f}: для булева только == и !=`); }
    else if (v.type === 'integer') { if (lit.t !== 'int' || !Number.isSafeInteger(lit.v)) throw new Error(`${f}: нужен целый литерал`); }
  }
  function primary() { if (peek() === '(') { p++; expr(); eat(')'); } else cmp(); }
  function notE() { if (peek() === 'not') { p++; notE(); } else primary(); }
  function andE() { notE(); while (peek() === 'and') { p++; notE(); } }
  function expr() { andE(); while (peek() === 'or') { p++; andE(); } }
  expr();
  if (p !== toks.length) throw new Error(`лишнее после выражения: «${toks[p].t}»`);
}

function clauseOk(std, clause) {
  const list = anchors.standards[std];
  if (!list) return false;
  const parts = clause.split(',').map((s) => s.trim());
  return parts.every((c, idx) => list.includes(c) || (anchors.qualifiers.includes(c) && idx > 0));
}

const WRITE = process.argv.includes('--write');
const files = walk(path.join(REPO, 'normative/process'), (p) => p.endsWith('.bpmn'));
if (!files.length) fail('normative/process', 'нет ни одного BPMN');
let nodesTotal = 0;
for (const file of files) {
  const W = rel(file);
  const xml = fs.readFileSync(file, 'utf8');
  const moddle = new BpmnModdle({ ant: ext });
  let res;
  try { res = await moddle.fromXML(xml, 'bpmn:Definitions'); } catch (e) { fail(W, `не разбирается: ${e.message}`); continue; }
  for (const w of res.warnings) fail(W, `предупреждение импорта: ${w.message}`);
  const defs = res.rootElement;
  if (defs.expressionLanguage !== rules.expression_language) fail(W, `expressionLanguage должен быть ${rules.expression_language} (Д-7)`);

  // DI
  const di = new Map();
  for (const d of defs.diagrams || []) for (const pe of d.plane.planeElement || []) {
    const id = pe.bpmnElement?.id;
    if (!id) { fail(W, `DI ${pe.id} без bpmnElement`); continue; }
    if (di.has(id)) fail(W, `у ${id} несколько DI`); di.set(id, pe);
  }
  const steps = [];
  const keys = new Map();
  const laneOf = new Map();
  const processes = defs.rootElements.filter((r) => r.$type === 'bpmn:Process');
  for (const r of defs.rootElements) if (!OTHER.has(r.$type.replace('bpmn:', '').replace(/^./, (c) => c.toLowerCase()))) fail(W, `неподдерживаемый корневой элемент ${r.$type} ${r.id}`);
  const collab = defs.rootElements.find((r) => r.$type === 'bpmn:Collaboration');
  for (const part of collab?.participants || []) if (!di.has(part.id)) fail(W, `нет DI участника ${part.id}`);
  const props = (el) => (el.extensionElements?.values || []).filter((v) => v.$type === 'ant:Properties');
  const extOf = (el, t) => (el.extensionElements?.values || []).filter((v) => v.$type === t);

  function walkProcess(proc, procId) {
    for (const ls of proc.laneSets || []) for (const lane of ls.lanes || []) {
      if (!di.has(lane.id)) fail(W, `нет DI дорожки ${lane.id}`);
      const lp = props(lane)[0];
      for (const ref of lane.flowNodeRef || []) laneOf.set(ref.id, { lane: lane.id, workshop: lp?.workshop });
    }
    for (const el of proc.flowElements || []) {
      const t = el.$type.replace('bpmn:', ''), tl = t.charAt(0).toLowerCase() + t.slice(1);
      const w = `${W} ${el.id}`;
      if (tl === 'sequenceFlow') {
        if (!di.has(el.id)) fail(w, 'нет DI стрелки');
        if (el.conditionExpression) {
          try { parseCondition(el.conditionExpression.body || '', rules.conditions.variables); }
          catch (e) { fail(w, `process.condition_invalid: ${e.message} — «${el.conditionExpression.body}»`); }
        }
        continue;
      }
      if (!FLOW.has(tl)) { fail(w, `process.unsupported_element: ${t}`); continue; }
      nodesTotal++;
      if (!di.has(el.id)) fail(w, 'нет DI узла');
      for (const ed of el.eventDefinitions || []) {
        const et = ed.$type.replace('bpmn:', ''); const etl = et.charAt(0).toLowerCase() + et.slice(1);
        if (!(rules.supported_elements.event_definitions[tl] || []).includes(etl)) fail(w, `process.unsupported_element: ${et} у ${t}`);
      }
      if (!tl.endsWith('Gateway') && (el.incoming || []).length > 1) fail(w, `process.implicit_merge: ${el.incoming.length} входящих стрелок без шлюза (Д-6)`);
      const ps = props(el);
      if (ps.length !== 1) { fail(w, `process.step_key_missing: нужен ровно один ant:properties, найдено ${ps.length}`); continue; }
      const pr = ps[0];
      if (!pr.stepKey || !STEP_RE.test(pr.stepKey)) fail(w, `step_key «${pr.stepKey}» не по шаблону`);
      if (keys.has(pr.stepKey)) fail(w, `step_key «${pr.stepKey}» уже у ${keys.get(pr.stepKey)}`); keys.set(pr.stepKey, el.id);
      if (rules.step_kind.required_on.includes(tl) && !rules.step_kind.values.includes(pr.stepKind)) fail(w, `stepKind «${pr.stepKind}» не из ${rules.step_kind.values}`);
      for (const [k, vals] of Object.entries(E)) {
        const [obj, attr] = k.split('.');
        if (obj !== 'properties') continue;
        if (pr[attr] !== undefined && !vals.includes(pr[attr])) fail(w, `${attr}=«${pr[attr]}» не из ${vals}`);
      }
      const insp = extOf(el, 'ant:Inspection');
      if (insp.length > 1) fail(w, 'больше одного ant:inspection');
      for (const i of insp) {
        if (!E['inspection.method'].includes(i.method)) fail(w, `метод ${i.method}`);
        if (!E['inspection.phase'].includes(i.phase)) fail(w, `фаза ${i.phase}`);
        for (const c of (i.coverage || '').split(/\s+/).filter(Boolean)) if (!defectCodes.has(c)) fail(w, `вида дефекта ${c} нет в классификаторе`);
      }
      if (pr.stepKind === 'automated_inspection' && !insp.length) fail(w, 'автоматизированный контроль без ant:inspection');
      for (const z of extOf(el, 'ant:ZoneRef')) if (!zones.has(z.zone)) fail(w, `нет зоны ${z.zone}`);
      for (const z of (pr.closesZoneAccess || '').split(/\s+/).filter(Boolean)) if (!zones.has(z)) fail(w, `closesZoneAccess: нет зоны ${z}`);
      const pp = extOf(el, 'ant:PresentationPoint');
      if (pr.closingPoint && tl === 'userTask' && !pp.length) fail(w, 'process.presentation_point_without_role: закрывающая точка без ant:presentationPoint');
      for (const x of pp) {
        if (!authorities.has(x.authority)) fail(w, `полномочия ${x.authority} нет в политике`);
        if (x.repeatAuthority && !authorities.has(x.repeatAuthority)) fail(w, `полномочия ${x.repeatAuthority} нет в политике`);
        if (x.role && !roles.has(x.role)) fail(w, `роли ${x.role} нет в политике`);
        if (x.stampKind && !stampKinds.has(x.stampKind)) fail(w, `вида клейма ${x.stampKind} нет в политике`);
      }
      if (pr.paperAttester && !roles.has(pr.paperAttester)) fail(w, `роль заверителя ${pr.paperAttester} нет в политике`);
      for (const c of extOf(el, 'ant:Precondition')) {
        if (!E['precondition.kind'].includes(c.kind)) fail(w, `предусловие ${c.kind}`);
        if (!E['precondition.mode'].includes(c.mode)) fail(w, `режим предусловия ${c.mode}`);
        if (c.kind === 'qualification' && !qualifications.has(c.ref)) fail(w, `квалификации ${c.ref} нет в политике`);
        if (c.kind === 'equipment_verification') for (const e of (c.ref || '').split('|')) if (!equipmentIds.has(e)) fail(w, `оборудования ${e} нет в справочнике`);
        if (c.kind === 'zone_check' && !zones.has(c.ref)) fail(w, `зоны ${c.ref} нет`);
      }
      for (const d of extOf(el, 'ant:Document')) if (!templateIds.has(d.template)) fail(w, `шаблона ${d.template} нет в normative/documents`);
      for (const r of extOf(el, 'ant:ReactionMap')) if (r.ref !== reactionRef) fail(w, `карта реакций ${r.ref} ≠ ${reactionRef}`);
      for (const n of extOf(el, 'ant:NormRef')) {
        if (!clauseOk(n.standard, n.clause || '')) fail(w, `опоры «${n.standard} ${n.clause}» нет в norm-anchors.yaml`);
        if (!n.check || !n.systemAction) fail(w, 'ant:normRef без check или systemAction');
      }
      const msg = (el.eventDefinitions || []).some((d) => d.$type === 'bpmn:MessageEventDefinition');
      if (msg && ['startEvent', 'intermediateCatchEvent'].includes(tl)) {
        if (!pr.triggerEventType || !cat.types[pr.triggerEventType]) fail(w, `событие-сообщение без triggerEventType из каталога (Д-5): «${pr.triggerEventType}»`);
      }
      if (msg && tl === 'intermediateThrowEvent' && !pr.erpAction) fail(w, 'событие «в 1С» без erpAction');
      if (tl === 'boundaryEvent') {
        const td = (el.eventDefinitions || [])[0]?.timeDuration?.body;
        if (!td || !/^P(T?[0-9]+[HMSD])+$/.test(td)) fail(w, `таймер без timeDuration ISO 8601: «${td}»`);
        if (!pr.timerScope) fail(w, 'граничный таймер без timerScope (Д-8)');
      }
      const lane = laneOf.get(el.id);
      steps.push({ step_key: pr.stepKey, bpmn_id: el.id, bpmn_type: tl, name: el.name || '', process: procId, workshop: lane?.workshop || null, step_kind: pr.stepKind || null, closing_point: pr.closingPoint || null, inspection_point: pr.inspectionPoint || null });
      if (tl === 'subProcess') walkProcess(el, procId);
    }
  }
  for (const proc of processes) walkProcess(proc, proc.id);
  // итог подпроцесса: если условия используют nc.outcome, у концов вызываемых процессов есть outcome
  const called = new Set(processes.flatMap((p) => (p.flowElements || []).filter((e) => e.$type === 'bpmn:CallActivity').map((e) => e.calledElement)));
  for (const proc of processes.filter((p) => called.has(p.id))) for (const e of proc.flowElements || [])
    if (e.$type === 'bpmn:EndEvent' && !props(e)[0]?.outcome) fail(`${W} ${e.id}`, 'конец вызываемого подпроцесса без outcome');

  const stepsFile = file.replace(/\.bpmn$/, '.steps.yaml');
  const text = '# Сгенерировано contracts/scripts/check-bpmn.mjs --write из ' + path.basename(file) + ' — не редактировать руками.\n'
    + '# Список step_key (AD-17): карта, заготовки, счётчики узлов. Изделия прежних версий — на узлах своего step_key.\n'
    + YAML.stringify({ source: path.basename(file), steps }, { lineWidth: 0 });
  if (WRITE) { fs.writeFileSync(stepsFile, text); console.log(`записан ${rel(stepsFile)}`); }
  else if (!fs.existsSync(stepsFile) || fs.readFileSync(stepsFile, 'utf8') !== text) fail(rel(stepsFile), 'не совпадает с BPMN — node contracts/scripts/check-bpmn.mjs --write');
}
if (!WRITE) report(`BPMN нормативного слоя по дескриптору urn:ant:bpmn-ext:1: ${files.length} файл(ов), ${nodesTotal} узлов, у каждого step_key`);
