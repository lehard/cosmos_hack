// Линтер диалекта JSON Schema (AD-20) и компиляция всех схем контрактов Ajv.
// Правила — contracts/README.md, раздел «Диалект JSON Schema».
import path from 'node:path';
import { CONTRACTS, schemaFiles, readJson, rel, fail, report, makeAjv, idFor } from './lib.mjs';

const BANNED = ['oneOf', 'anyOf', 'allOf', 'not', 'if', 'then', 'else', 'dependencies', 'dependentSchemas', 'dependentRequired', '$defs', 'unevaluatedProperties'];
const BANNED_FORMATS = ['date', 'time', 'duration'];
const MAXI = 9007199254740991;
const CYR = /[А-Яа-яЁё]/;
const NAME = /^[a-z][a-z0-9_]*$/;
const ENUM_VAL = /^[a-z][a-z0-9_]*$/;

// Открытые схемы — события (FR-29); закрытые — всё остальное (AD-10).
function isOpen(file) {
  const r = path.relative(CONTRACTS, file).split(path.sep);
  return r[0] === 'events' && !r[r.length - 1].startsWith('catalog');
}
// Мета-схемы файлов конфигурации (catalog.schema.json) — не контракт данных, правила имён к ним не применяются.
function isMeta(file) { return file.endsWith('catalog.schema.json'); }

function lintNode(node, where, file, ctx) {
  if (node === null || typeof node !== 'object') return;
  if (Array.isArray(node)) { node.forEach((n, i) => lintNode(n, `${where}[${i}]`, file, ctx)); return; }
  for (const k of BANNED) if (k in node) fail(where, `запрещённое ключевое слово «${k}» (диалект AD-20)`);
  if ('$ref' in node) {
    const others = Object.keys(node).filter((k) => k !== '$ref' && k !== 'description');
    if (others.length) fail(where, `рядом с $ref игнорируются ключи ${others.join(', ')}`);
    return;
  }
  const t = node.type;
  const types = Array.isArray(t) ? t : t ? [t] : [];
  if (types.includes('number')) fail(where, 'type: number запрещён — целое + масштаб (AD-4)');
  if (node.format && BANNED_FORMATS.includes(node.format)) fail(where, `format: ${node.format} запрещён — date-time или pattern`);
  for (const b of ['minimum', 'maximum']) if (b in node && Math.abs(node[b]) > MAXI) fail(where, `${b} вне ±(2^53−1)`);
  if (Array.isArray(node.enum) && !ctx.meta) {
    for (const v of node.enum) if (typeof v === 'string' && !ENUM_VAL.test(v)) fail(where, `значение enum «${v}» не snake_case`);
  }
  if (types.includes('object') || node.properties) {
    const open = ctx.open;
    if (open && node.additionalProperties === false) fail(where, 'схема события закрыта (additionalProperties: false) — новые необязательные поля должны приниматься (FR-29)');
    if (!open && !ctx.meta && node.properties && node.additionalProperties !== false) fail(where, 'закрытая схема: нужен additionalProperties: false (AD-10)');
    for (const [pn, pv] of Object.entries(node.properties || {})) {
      const pw = `${where}.properties.${pn}`;
      if (!ctx.meta && !NAME.test(pn)) fail(pw, 'имя свойства не snake_case');
      if (pv && typeof pv === 'object' && !('$ref' in pv)) {
        if (!pv.description) fail(pw, 'нет description');
        else if (!ctx.meta && !CYR.test(pv.description)) fail(pw, 'description не на русском');
      }
      lintNode(pv, pw, file, ctx);
    }
    for (const r of node.required || []) if (node.properties && !(r in node.properties)) fail(where, `required «${r}» нет в properties`);
    if (node.additionalProperties && typeof node.additionalProperties === 'object') lintNode(node.additionalProperties, `${where}.additionalProperties`, file, ctx);
    if (node.patternProperties) for (const [k, v] of Object.entries(node.patternProperties)) lintNode(v, `${where}.patternProperties.${k}`, file, ctx);
  }
  if (node.items) lintNode(node.items, `${where}.items`, file, ctx);
  if (node.definitions) for (const [k, v] of Object.entries(node.definitions)) {
    if (v && typeof v === 'object' && !('$ref' in v) && !v.description) fail(`${where}.definitions.${k}`, 'нет description');
    lintNode(v, `${where}.definitions.${k}`, file, ctx);
  }
  if (node.propertyNames) lintNode(node.propertyNames, `${where}.propertyNames`, file, ctx);
}

const files = schemaFiles();
let n = 0;
for (const f of files) {
  let s;
  try { s = readJson(f); } catch (e) { fail(rel(f), `не JSON: ${e.message}`); continue; }
  if (!s.$schema) continue; // не схема (например, конфигурация)
  n++;
  const where = rel(f);
  if (s.$schema !== 'http://json-schema.org/draft-07/schema#') fail(where, `$schema должен быть draft-07, а не ${s.$schema}`);
  if (s.$id !== idFor(f)) fail(where, `$id должен повторять путь: ${idFor(f)}`);
  if (!s.title || !/^[A-Z][A-Za-z0-9]*$/.test(s.title)) fail(where, 'нет title в PascalCase (имя генерируемого типа)');
  if (!s.description || !CYR.test(s.description)) fail(where, 'нет description на русском у корня');
  lintNode(s, where, f, { open: isOpen(f), meta: isMeta(f) });
}

// Компиляция: все $ref разрешаются, схемы корректны для Ajv (strict).
try {
  const ajv = makeAjv();
  for (const f of files) {
    const s = readJson(f);
    if (!s.$schema) continue;
    try { ajv.getSchema(s.$id) || ajv.compile(s); } catch (e) { fail(rel(f), `Ajv: ${e.message}`); }
  }
} catch (e) { fail('ajv', e.message); }

report(`диалект схем AD-20 и компиляция (${n} схем)`);
