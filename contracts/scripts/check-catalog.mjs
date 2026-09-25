// Проверки каталога типов записей (AD-40) и словаря статусов (AD-30):
//   формат catalog.yaml; у каждого типа есть схема data каждой версии и ровно один эмитент;
//   эмитент = владелец семейства (кроме emitter_exceptions); ось меняет только её владелец;
//   у критического действия есть группа CA; нет схем без строки каталога; словарь статусов = enum в defs.
import fs from 'node:fs';
import path from 'node:path';
import { CONTRACTS, readYaml, readJson, rel, fail, report, walk, makeAjv, formatAjvErrors } from './lib.mjs';

const catPath = path.join(CONTRACTS, 'events/catalog.yaml');
const cat = readYaml(catPath);
const ajv = makeAjv();
const validateCat = ajv.getSchema('https://ant.invalid/contracts/events/catalog.schema.json');
if (!validateCat(cat)) fail(rel(catPath), `формат: ${formatAjvErrors(validateCat.errors)}`);

const statuses = readYaml(path.join(CONTRACTS, 'statuses.yaml'));
const axisOwner = Object.fromEntries(Object.entries(statuses.axes).map(([k, v]) => [k, v.owner]));
const modules = new Set(Object.keys(cat.modules || {}));
const roles = new Set(Object.keys(cat.roles || {}));
const streams = new Set(Object.keys(cat.streams || {}));
const families = cat.families || {};

for (const [fam, f] of Object.entries(families)) if (!modules.has(f.owner)) fail(`families.${fam}`, `неизвестный модуль ${f.owner}`);
for (const t of Object.keys(cat.emitter_exceptions || {})) if (!cat.types?.[t]) fail(`emitter_exceptions.${t}`, 'нет такого типа');

const seenSchemas = new Set();
const titles = new Map();
for (const [name, t] of Object.entries(cat.types || {})) {
  const w = `types.${name}`;
  const fam = name.split('.')[0];
  const famDef = families[fam];
  if (!famDef) { fail(w, `семейство «${fam}» не объявлено (AD-40)`); continue; }
  if (!modules.has(t.emitter)) fail(w, `эмитент ${t.emitter} — не модуль`);
  const exc = cat.emitter_exceptions?.[name];
  if (t.emitter !== famDef.owner && !exc) fail(w, `эмитент ${t.emitter} ≠ владелец семейства ${famDef.owner} и не в emitter_exceptions (AD-40)`);
  if (t.emitter === famDef.owner && exc) fail(w, 'лишнее исключение: эмитент совпадает с владельцем');
  if (!roles.has(t.role)) fail(w, `неизвестная роль ${t.role}`);
  if (!streams.has(t.stream)) fail(w, `неизвестный вид потока ${t.stream}`);
  if (t.axis !== 'none' && axisOwner[t.axis] !== t.emitter) fail(w, `ось ${t.axis} принадлежит ${axisOwner[t.axis]}, а эмитент ${t.emitter} (AD-30)`);
  if (t.critical && !t.ca_group) fail(w, 'критическое действие без группы CA (AD-28)');
  if (!t.critical && t.ca_group) fail(w, 'группа CA у некритического типа');
  if (t.kind === 'fact' && t.action_class !== 'record') fail(w, 'факт — не действие: класс record (AD-27)');
  if (t.kind === 'reaction' && !(t.provenance.length === 1 && t.provenance[0] === 'server_attested')) fail(w, 'реакции подписывает ключ движка: provenance [server_attested] (AD-3)');
  if (t.kind === 'decision' && t.provenance.includes('server_attested')) fail(w, 'решение человека не может быть server_attested (AD-2)');
  if (t.action_class === 'permissive' && t.kind === 'fact' && t.provenance.every((p) => p === 'server_attested')) fail(w, 'разрешающее действие только на server_attested (AD-2)');
  if (!t.versions.includes(t.current_version) || t.current_version !== Math.max(...t.versions)) fail(w, 'current_version должна быть наибольшей из versions');
  for (const v of t.versions) {
    const p = path.join(CONTRACTS, 'events', fam, `${name}.v${v}.json`);
    seenSchemas.add(p);
    if (!fs.existsSync(p)) { fail(w, `нет схемы ${rel(p)}`); continue; }
    const s = readJson(p);
    if (titles.has(s.title)) fail(rel(p), `title ${s.title} уже у ${titles.get(s.title)}`); else titles.set(s.title, rel(p));
    if (s.type !== 'object') fail(rel(p), 'data — объект');
  }
}
// Схемы без строки каталога (кроме общих).
for (const p of walk(path.join(CONTRACTS, 'events'), (f) => /\.v[0-9]+\.json$/.test(f))) {
  if (p.includes(`${path.sep}common${path.sep}`) || p.includes(`${path.sep}examples${path.sep}`)) continue;
  if (!seenSchemas.has(p)) fail(rel(p), 'схема без строки в catalog.yaml');
}

// Словарь статусов == enum осей и словарей в defs.v1.json.
const defs = readJson(path.join(CONTRACTS, 'events/common/defs.v1.json')).definitions;
const cmp = (kind, name, vals) => {
  const d = defs[`${kind}_${name}`];
  if (!d) { fail('defs.v1.json', `нет ${kind}_${name} (словарь statuses.yaml)`); return; }
  const a = JSON.stringify(d.enum), b = JSON.stringify(vals.map((x) => x.code));
  if (a !== b) fail('defs.v1.json', `${kind}_${name}: ${a} ≠ statuses.yaml ${b}`);
};
for (const [k, v] of Object.entries(statuses.axes)) cmp('axis', k, v.values);
for (const [k, v] of Object.entries(statuses.dictionaries || {})) cmp('dict', k, v.values);
for (const [k, v] of [...Object.entries(statuses.axes), ...Object.entries(statuses.dictionaries || {})])
  for (const x of v.values) if (!statuses.palette[x.tone]) fail(`statuses.yaml ${k}.${x.code}`, `тон ${x.tone} не в палитре`);
if (Object.keys(statuses.axes).length !== 6) fail('statuses.yaml', 'осей §3b должно быть шесть');
for (const [k, v] of Object.entries(statuses.axes)) if (!modules.has(v.owner) && v.owner) fail(`statuses.yaml ${k}`, `владелец ${v.owner} — не модуль`);

const n = Object.keys(cat.types || {}).length;
report(`каталог: ${n} типов, у каждого схема и один эмитент; словарь статусов`);
