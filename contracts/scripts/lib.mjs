// Общие помощники проверок контрактов (contracts/scripts/check.sh).
// Слой: инструменты сборки, не код продукта. AD-20: один источник на вид контракта.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import YAML from 'yaml';
import Ajv from 'ajv';
import addFormats from 'ajv-formats';

export const SCRIPTS_DIR = path.dirname(fileURLToPath(import.meta.url));
export const CONTRACTS = path.resolve(SCRIPTS_DIR, '..');
export const REPO = path.resolve(CONTRACTS, '..');
export const SCHEMA_BASE = 'https://ant.invalid/contracts/';

const errors = [];
export function fail(where, msg) { errors.push(`${where}: ${msg}`); }
export function report(title) {
  if (errors.length) {
    console.error(`✗ ${title}: ${errors.length} наруш.`);
    for (const e of errors) console.error('  - ' + e);
    process.exit(1);
  }
  console.log(`✓ ${title}`);
}

export const rel = (p) => path.relative(REPO, p);
export const readYaml = (p) => YAML.parse(fs.readFileSync(p, 'utf8'));
export const readJson = (p) => JSON.parse(fs.readFileSync(p, 'utf8'));

export function walk(dir, pred = () => true, out = []) {
  if (!fs.existsSync(dir)) return out;
  for (const ent of fs.readdirSync(dir, { withFileTypes: true })) {
    if (ent.name === 'node_modules' || ent.name.startsWith('.')) continue;
    const p = path.join(dir, ent.name);
    if (ent.isDirectory()) walk(p, pred, out);
    else if (pred(p)) out.push(p);
  }
  return out.sort();
}

// Все JSON Schema контрактов: *.json с ключом $schema (draft-07).
export function schemaFiles() {
  return walk(CONTRACTS, (p) => p.endsWith('.json') && !p.includes(`${path.sep}examples${path.sep}`)
    && !p.includes(`${path.sep}test-vectors${path.sep}`) && !p.endsWith('package.json') && !p.endsWith('package-lock.json')
    && !p.endsWith(`bpmn-ext${path.sep}ant.json`));
}

// Ajv с загруженными схемами контрактов: ссылки $ref разрешаются по $id (= путь файла).
export function makeAjv() {
  const ajv = new Ajv({ strict: true, strictRequired: false, allErrors: true, allowUnionTypes: true });
  addFormats(ajv, ['date-time', 'uri', 'email']);
  for (const f of schemaFiles()) {
    const s = readJson(f);
    if (s.$schema !== 'http://json-schema.org/draft-07/schema#') continue;
    ajv.addSchema(s, s.$id || SCHEMA_BASE + path.relative(CONTRACTS, f).split(path.sep).join('/'));
  }
  return ajv;
}

export function idFor(file) {
  return SCHEMA_BASE + path.relative(CONTRACTS, file).split(path.sep).join('/');
}

export function formatAjvErrors(errs) {
  return (errs || []).map((e) => `${e.instancePath || '/'} ${e.message}${e.params && e.params.additionalProperty ? ` (${e.params.additionalProperty})` : ''}`).join('; ');
}
