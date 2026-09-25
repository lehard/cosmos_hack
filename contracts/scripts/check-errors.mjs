// Проверка каталога ошибок contracts/errors.yaml (FR-28, RFC 9457):
//   формат; префикс кода — семейство ошибок; рабочие коды процессной сессии сведены один раз;
//   все коды, упомянутые в контрактах (`семейство.причина`), объявлены.
import fs from 'node:fs';
import path from 'node:path';
import { CONTRACTS, readYaml, rel, fail, report, walk, makeAjv, formatAjvErrors } from './lib.mjs';

const P = path.join(CONTRACTS, 'errors.yaml');
const doc = readYaml(P);
const ajv = makeAjv();
const v = ajv.getSchema('https://ant.invalid/contracts/errors.schema.json');
if (!v(doc)) fail(rel(P), formatAjvErrors(v.errors));
const PREFIXES = new Set(['api', 'ingest', 'journal', 'access', 'signing', 'nonconformity', 'process', 'incident', 'erp', 'mes', 'analyzer', 'federation', 'simulation', 'document', 'item', 'quality', 'security', 'reference', 'ops']);
const codes = doc.codes || {};
const aliases = new Map();
for (const [c, d] of Object.entries(codes)) {
  const pfx = c.split('.')[0];
  if (!PREFIXES.has(pfx)) fail(`errors.yaml ${c}`, `префикс «${pfx}» не семейство ошибок`);
  if (d.quarantine && pfx !== 'ingest') fail(`errors.yaml ${c}`, 'карантин — только у кодов приёма');
  if ((d.severity ?? 'error') === 'error' && d.status < 400) fail(`errors.yaml ${c}`, 'ошибка со статусом < 400');
  for (const a of d.aliases || []) { if (aliases.has(a)) fail(`errors.yaml ${c}`, `рабочий код ${a} уже сведён к ${aliases.get(a)}`); aliases.set(a, c); }
}
for (const a of ['E_MISSING_FIELD', 'E_UNSUPPORTED_VERSION', 'E_UNKNOWN_ENUM', 'E_ID_CONFLICT', 'E_REWORK_LIMIT', 'E_PERMIT_REQUIRED', 'E_REF_NOT_FOUND'])
  if (!aliases.has(a)) fail('errors.yaml', `рабочий код процессной сессии ${a} не сведён`);
// Упоминания кодов в контрактах: `prefix.reason` в обратных кавычках (кроме префиксов erp и decision —
// у процессной сессии так называются события, см. таблицу сведения имён в events/README.md).
const re = /`((?:api|ingest|journal|access|signing|nonconformity|reference|process|incident|mes|analyzer|federation|simulation)\.[a-z][a-z0-9_]*)`/g;
for (const f of walk(CONTRACTS, (p) => /\.(json|ya?ml|md)$/.test(p) && !p.endsWith('package-lock.json'))) {
  const txt = fs.readFileSync(f, 'utf8');
  for (const m of txt.matchAll(re)) {
    const code = m[1];
    if (code.split('.').length !== 2) continue;
    if (!codes[code]) fail(rel(f), `упомянут необъявленный код ${code}`);
  }
}
report(`каталог ошибок: ${Object.keys(codes).length} кодов RFC 9457`);
