// Примеры сообщений контракта (FR-29, FR-110): пять случаев изменения контракта проверяются эмуляцией
// правил приёма из таблицы AD-20, версии v1 → v2 — схемами и таблицей повышателя.
import path from 'node:path';
import { CONTRACTS, readYaml, readJson, rel, fail, report, walk, makeAjv, formatAjvErrors } from './lib.mjs';

const ajv = makeAjv();
const cat = readYaml(path.join(CONTRACTS, 'events/catalog.yaml'));
const crit = readYaml(path.join(CONTRACTS, 'events/safety-critical-enums.yaml')).fields;
const env = ajv.getSchema('https://ant.invalid/contracts/events/common/envelope.v1.json');
const dataSchema = (t, v) => ajv.getSchema(`https://ant.invalid/contracts/events/${t.split('.')[0]}/${t}.v${v}.json`);
for (const [t, ptrs] of Object.entries(crit)) if (!cat.types[t]) fail('safety-critical-enums.yaml', `нет типа ${t}`); else void ptrs;

const matchPtr = (pattern, ptr) => new RegExp('^' + pattern.replace(/\*/g, '[0-9]+') + '$').test(ptr);
// Эмуляция приёма по таблице AD-20 (реальный приём — эпик 06, те же схемы через santhosh-tekuri/jsonschema).
export function ingest(msg) {
  const t = cat.types[msg.event_type];
  if (!t) return { outcome: 'quarantined', code: 'ingest.unknown_event_type' };
  if (!t.versions.includes(msg.schema_version)) return { outcome: 'quarantined', code: 'ingest.unknown_schema_version' };
  if (!env(msg)) return { outcome: 'quarantined', code: 'ingest.schema_violation', detail: formatAjvErrors(env.errors) };
  const v = dataSchema(msg.event_type, msg.schema_version);
  if (v(msg.data)) return { outcome: 'accepted' };
  const errs = v.errors;
  const req = errs.find((e) => e.keyword === 'required');
  if (req) return { outcome: 'quarantined', code: 'ingest.missing_required_field', field: `/data${req.instancePath}/${req.params.missingProperty}` };
  const en = errs.filter((e) => e.keyword === 'enum');
  if (en.length && en.length === errs.length) {
    const ptr = en[0].instancePath;
    if ((crit[msg.event_type] || []).some((p) => matchPtr(p, ptr))) return { outcome: 'quarantined', code: 'ingest.unknown_enum_value_critical', field: `/data${ptr}` };
    const val = ptr.split('/').slice(1).reduce((o, k) => o?.[k], msg.data);
    return { outcome: 'accepted_with_flag', code: 'ingest.unknown_enum_value', field: `/data${ptr}`, stored_as: `UNKNOWN(${val})` };
  }
  return { outcome: 'quarantined', code: 'ingest.schema_violation', detail: formatAjvErrors(errs) };
}

const cases = walk(path.join(CONTRACTS, 'events/examples/contract-change'), (p) => p.endsWith('.json'));
const seen = new Set();
for (const f of cases) {
  const ex = readJson(f);
  const got = ingest(ex.message);
  for (const [k, v] of Object.entries(ex.expect)) if (got[k] !== v) fail(rel(f), `ожидалось ${k}=${v}, получено ${got[k]} (${got.detail || ''})`);
  seen.add(ex.expect.code || ex.expect.outcome);
}
for (const need of ['ingest.unknown_schema_version', 'accepted', 'ingest.missing_required_field', 'ingest.unknown_enum_value', 'ingest.unknown_enum_value_critical'])
  if (!seen.has(need)) fail('examples/contract-change', `нет примера для случая ${need} (FR-29)`);

// Версии v1 → v2 и повышатель.
const V = path.join(CONTRACTS, 'events/examples/versions');
const v1 = readJson(path.join(V, 'equipment.state.changed.v1.json'));
const v2 = readJson(path.join(V, 'equipment.state.changed.v2.json'));
const up = readJson(path.join(V, 'equipment.state.changed.v1-upcast-to-v2.json'));
if (ingest(v1).outcome !== 'accepted') fail('versions v1', JSON.stringify(ingest(v1)));
if (ingest(v2).outcome !== 'accepted') fail('versions v2', JSON.stringify(ingest(v2)));
const u = readYaml(path.join(CONTRACTS, 'events/upcasters/equipment.state.changed.v1-to-v2.yaml'));
const res = structuredClone(v1);
res.schema_version = u.to;
const src = structuredClone(v1.data);
for (const m of u.map) {
  const val = src[m.from.slice(1)];
  const tgt = m.cases[val] || m.default;
  for (const [p, x] of Object.entries(tgt)) res.data[p.slice(1)] = x;
}
for (const [p, x] of Object.entries(u.set || {})) res.data[p.slice(1)] = x;
for (const p of u.remove || []) delete res.data[p.slice(1)];
if (JSON.stringify(Object.entries(res.data).sort()) !== JSON.stringify(Object.entries(up.data).sort()) || res.schema_version !== up.schema_version)
  fail('versions', 'повышатель v1 → v2 даёт не то, что в v1-upcast-to-v2.json');
if (ingest(up).outcome !== 'accepted') fail('versions upcast', 'результат повышения не проходит схему v2');
if (!cat.types['equipment.state.changed'].versions.includes(2)) fail('catalog', 'equipment.state.changed без версии 2');
report(`примеры: 5 случаев изменения контракта (FR-29) и equipment.state.changed v1 → v2 с повышателем (FR-110)`);
