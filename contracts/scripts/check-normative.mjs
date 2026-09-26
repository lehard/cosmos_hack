// Затравка нормативного слоя /normative (AD-17, AD-31, AD-33): файлы по схемам contracts/normative/*
// и перекрёстные ссылки между ними (классификатор ↔ карта реакций, политика, номенклатура, места, оборудование).
import fs from 'node:fs';
import path from 'node:path';
import { REPO, readYaml, rel, fail, report, walk, makeAjv, formatAjvErrors } from './lib.mjs';

const N = path.join(REPO, 'normative');
const ajv = makeAjv();
const SCHEMA = {
  'defects/classifier': 'classifier', 'reactions/reaction-map': 'reaction-map', 'policy/policy': 'policy',
  'documents/templates': 'templates', 'reference/flange/locations': 'locations', 'reference/flange/item-types': 'item-types',
  'reference/flange/equipment': 'equipment', 'reference/flange/calendar': 'calendar', 'reference/flange/shifts': 'shifts',
  'reference/flange/lot-templates': 'lot-templates',
  'vision/analyzer-passports': 'analyzer-passports', 'vision/illustrations': 'illustrations',
};
export function loadNormative() {
  const out = {};
  // Столы ролей normative/desks/*.yaml (эпик 03) проверяет frontend/scripts/check-shell.mjs
  // по своей схеме normative/desks/desk.schema.json и реестру виджетов.
  for (const f of walk(N, (p) => p.endsWith('.yaml') && !p.endsWith('.steps.yaml') && !p.includes(`${path.sep}desks${path.sep}`))) {
    const r = path.relative(N, f).replace(/\.v[0-9]+\.yaml$|\.yaml$/, '');
    out[r] = { file: f, data: readYaml(f) };
  }
  return out;
}
const docs = loadNormative();
for (const [key, name] of Object.entries(SCHEMA)) if (!docs[key]) fail('normative/', `нет файла ${key}`);
for (const [key, d] of Object.entries(docs)) {
  const s = SCHEMA[key];
  if (!s) { fail(rel(d.file), 'файл без схемы в contracts/normative (добавьте в check-normative.mjs)'); continue; }
  const v = ajv.getSchema(`https://ant.invalid/contracts/normative/${s}.schema.json`);
  if (!v(d.data)) fail(rel(d.file), formatAjvErrors(v.errors));
}
const cls = docs['defects/classifier']?.data, rm = docs['reactions/reaction-map']?.data, pol = docs['policy/policy']?.data;
const loc = docs['reference/flange/locations']?.data, it = docs['reference/flange/item-types']?.data, eq = docs['reference/flange/equipment']?.data;
const lots = docs['reference/flange/lot-templates']?.data;
const uniq = (arr, w) => { const s = new Set(); for (const x of arr) { if (s.has(x)) fail(w, `повтор ${x}`); s.add(x); } return s; };
if (cls && rm) {
  const codes = uniq(cls.defect_types.map((d) => d.code), 'classifier');
  for (const r of rm.rules) for (const c of r.match.defect_types || []) if (!codes.has(c)) fail(`reaction-map ${r.id}`, `вида ${c} нет в классификаторе`);
  uniq(rm.rules.map((r) => r.id), 'reaction-map');
}
if (pol) {
  const roles = uniq(pol.roles.map((r) => r.id), 'policy.roles');
  const CASE = ['quality_inspector', 'site_foreman', 'technologist', 'production_manager', 'administrator'];
  for (const c of CASE) if (!pol.roles.find((r) => r.id === c && r.case_role)) fail('policy', `нет роли кейса ${c}`);
  for (const r of pol.roles) for (const i of r.inherits || []) if (!roles.has(i)) fail(`policy.roles.${r.id}`, `наследует неизвестную ${i}`);
  const auth = uniq(pol.authorities.map((a) => a.id), 'policy.authorities');
  const persons = uniq(pol.persons.map((p) => p.id), 'policy.persons');
  const scopeOk = (s) => s === pol.scopes.root || s.startsWith(pol.scopes.root + '/');
  for (const p of pol.persons) for (const r of p.roles) { if (!roles.has(r.role)) fail(`policy.persons.${p.id}`, `роль ${r.role}`); if (!scopeOk(r.scope)) fail(`policy.persons.${p.id}`, `область ${r.scope}`); }
  for (const g of pol.grants.authorities) { if (!persons.has(g.person)) fail('policy.grants', `нет сотрудника ${g.person}`); if (!auth.has(g.authority)) fail('policy.grants', `нет полномочия ${g.authority}`); }
  for (const g of pol.grants.stamps) { if (!persons.has(g.person)) fail('policy.stamps', g.person); if (!pol.stamp_kinds.includes(g.kind)) fail('policy.stamps', `вид ${g.kind}`); }
  for (const g of pol.grants.qualifications) if (!persons.has(g.person)) fail('policy.qualifications', g.person);
  for (const v of Object.values(pol.second_signature)) if (!auth.has(v)) fail('policy.second_signature', v);
  if (pol.persons.filter((p) => p.roles.some((r) => r.role === 'performer')).length < 2) fail('policy', 'кейс §4.2: не менее двух операторов');
  if (loc) for (const l of loc.locations) if (!scopeOk(l.scope)) fail(`locations.${l.id}`, `область ${l.scope} вне корня политики`);
}
if (loc) {
  const ids = uniq(loc.locations.map((l) => l.id), 'locations');
  for (const l of loc.locations) {
    if (l.parent && !ids.has(l.parent)) fail(`locations.${l.id}`, `нет родителя ${l.parent}`);
    if (l.warehouse_id && !ids.has(l.warehouse_id)) fail(`locations.${l.id}`, `нет склада ${l.warehouse_id}`);
  }
  if (eq) for (const e of eq.equipment) if (!ids.has(e.location_id)) fail(`equipment.${e.id}`, `нет места ${e.location_id}`);
}
if (it) {
  const types = uniq(it.item_types.map((t) => t.id), 'item-types');
  for (const t of it.item_types) {
    for (const c of t.components || []) if (!types.has(c.item_type_id)) fail(`item-types.${t.id}`, `нет компонента ${c.item_type_id}`);
    const zones = uniq((t.zones || []).map((z) => z.zone_id), `item-types.${t.id}.zones`);
    for (const l of t.links || []) for (const z of [...l.zones, ...(l.closes_access_to || [])]) if (!zones.has(z)) fail(`item-types.${t.id}.links.${l.link_id}`, `нет зоны ${z}`);
  }
  if (lots) for (const l of lots.lots) if (!types.has(l.item_type_id)) fail(`lot-templates.${l.id}`, `нет типа ${l.item_type_id}`);
}
if (eq) {
  const ids = uniq(eq.equipment.map((e) => e.id), 'equipment');
  for (const v of eq.verifications) if (!ids.has(v.equipment_id)) fail('equipment.verifications', v.equipment_id);
}
report(`затравка нормативного слоя: ${Object.keys(docs).length} файлов по схемам и перекрёстные ссылки`);
