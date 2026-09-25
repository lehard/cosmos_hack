// Перепроверка тест-векторов contracts/crypto/test-vectors/vectors.v1.json без ГОСТ-библиотеки:
// JCS (RFC 8785), DSSE PAE, UUIDv5 от NS_ANT, пример события и записи цепочки — по схемам.
// Стрибог-256 и подписи проверяют Go-тесты криптоядра (эпик 05) на этих же векторах.
import path from 'node:path';
import crypto from 'node:crypto';
import { CONTRACTS, readJson, readYaml, fail, report, makeAjv, formatAjvErrors } from './lib.mjs';

export function jcs(v) {
  if (v === null || typeof v !== 'object') {
    if (typeof v === 'number' && !Number.isSafeInteger(v)) throw new Error(`число вне контракта: ${v}`);
    return JSON.stringify(v);
  }
  if (Array.isArray(v)) return '[' + v.map(jcs).join(',') + ']';
  const keys = Object.keys(v).sort((a, b) => (a < b ? -1 : a > b ? 1 : 0)); // сравнение по UTF-16, как в RFC 8785
  return '{' + keys.map((k) => JSON.stringify(k) + ':' + jcs(v[k])).join(',') + '}';
}
const pae = (t, body) => Buffer.concat([Buffer.from(`DSSEv1 ${Buffer.byteLength(t)} ${t} ${body.length} `), body]);
function uuid5(ns, name) {
  const nsb = Buffer.from(ns.replace(/-/g, ''), 'hex');
  const h = crypto.createHash('sha1').update(nsb).update(Buffer.from(name, 'utf8')).digest().subarray(0, 16);
  h[6] = (h[6] & 0x0f) | 0x50; h[8] = (h[8] & 0x3f) | 0x80;
  const x = h.toString('hex');
  return `${x.slice(0, 8)}-${x.slice(8, 12)}-${x.slice(12, 16)}-${x.slice(16, 20)}-${x.slice(20)}`;
}

const V = readJson(path.join(CONTRACTS, 'crypto/test-vectors/vectors.v1.json'));
const W = 'vectors.v1.json';
for (const [i, t] of V.jcs.entries()) if (jcs(JSON.parse(t.input)) !== t.canonical_utf8) fail(W, `jcs[${i}] не совпал`);
const ev = V.sample_event.json;
const evc = Buffer.from(jcs(ev), 'utf8');
if (evc.toString('utf8') !== V.sample_event.canonical_utf8) fail(W, 'sample_event.canonical_utf8');
if (Buffer.from(V.dsse_pae.payload_b64, 'base64').toString('utf8') !== V.sample_event.canonical_utf8) fail(W, 'dsse_pae.payload_b64');
if (pae(V.dsse_pae.payload_type, evc).toString('hex') !== V.dsse_pae.pae_hex) fail(W, 'dsse_pae.pae_hex');
if (V.mldsa65.context !== `ant/${V.dsse_pae.payload_type}`) fail(W, 'контекст ML-DSA ≠ ant/‹payloadType›');
const consts = readYaml(path.join(CONTRACTS, 'constants.yaml'));
for (const u of V.uuid5) if (uuid5(u.namespace, u.name_utf8) !== u.uuid) fail(W, `uuid5 ${u.name_utf8}`);
if (!V.uuid5.some((u) => u.uuid === consts.NS_ANT && u.name_utf8 === 'urn:ant:ns:1')) fail('constants.yaml', 'NS_ANT ≠ UUIDv5(NAMESPACE_URL, "urn:ant:ns:1")');
const docIn = { content: JSON.parse(V.doc_digest.content_json), rendering_hash: V.doc_digest.rendering_hash, template_ref: V.doc_digest.template_ref, doc_format_version: Number(V.doc_digest.doc_format_version) };
if (jcs(docIn) !== V.doc_digest.digest_input_canonical_utf8) fail(W, 'doc_digest: вход отпечатка');

const ajv = makeAjv();
const envV = ajv.getSchema('https://ant.invalid/contracts/events/common/envelope.v1.json');
if (!envV(ev)) fail(W, `sample_event не по конверту: ${formatAjvErrors(envV.errors)}`);
const cat = readYaml(path.join(CONTRACTS, 'events/catalog.yaml'));
const t = cat.types[ev.event_type];
if (!t) fail(W, `sample_event: тип ${ev.event_type} не в каталоге`);
else {
  const dv = ajv.getSchema(`https://ant.invalid/contracts/events/${ev.event_type.split('.')[0]}/${ev.event_type}.v${ev.schema_version}.json`);
  if (!dv(ev.data)) fail(W, `sample_event.data: ${formatAjvErrors(dv.errors)}`);
}
const dsseV = ajv.getSchema('https://ant.invalid/contracts/crypto/dsse-envelope.schema.json');
if (!dsseV(V.dsse_envelope_hybrid)) fail(W, `dsse_envelope_hybrid: ${formatAjvErrors(dsseV.errors)}`);
if (jcs(V.dsse_envelope_hybrid) !== V.chain_v1.envelope_canonical_utf8) fail(W, 'chain_v1.envelope_canonical_utf8');
const entryV = ajv.getSchema('https://ant.invalid/contracts/journal/entry.schema.json');
for (const [i, e] of V.chain_v1.entries.entries()) {
  const entry = { ...JSON.parse(e.open_fields_json), link: e.link, sealed: { aead: 'kuznyechik_mgm', dek_id: 'dek-1', nonce_b64: 'AAAA', ciphertext_b64: 'AAAA' } };
  if (!entryV(entry)) fail(W, `chain_v1.entries[${i}] не по entry.schema.json: ${formatAjvErrors(entryV.errors)}`);
  if (JSON.parse(e.open_fields_json).commit !== e.commit) fail(W, `chain_v1.entries[${i}].commit`);
  if (i > 0 && e.prev_link !== V.chain_v1.entries[i - 1].link) fail(W, `chain_v1.entries[${i}].prev_link`);
}
if (V.chain_v1.entries[0].prev_link !== 'streebog256:' + '0'.repeat(64)) fail(W, 'prev_link первой записи — нули');
if (V.gost3410_2012_256.verify !== true || V.gost3410_2012_256.tampered_verify !== false) fail(W, 'ГОСТ: ожидаемые исходы проверки');
if (Buffer.from(V.gost3410_2012_256.signature_b64, 'base64').length !== 64) fail(W, 'подпись ГОСТ — 64 байта');
if (Buffer.from(V.gost3410_2012_256.public_key_b64, 'base64').length !== 64) fail(W, 'ключ ГОСТ — 64 байта');
if (Buffer.from(V.mldsa65.signature_b64, 'base64').length !== 3309) fail(W, 'подпись ML-DSA-65 — 3309 байт');
if (Buffer.from(V.mldsa65.public_key_b64, 'base64').length !== 1952) fail(W, 'ключ ML-DSA-65 — 1952 байта');
report('тест-векторы crypto: JCS, PAE, UUIDv5, пример события и цепочки по схемам');
