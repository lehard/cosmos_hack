// Сверка сборки WASM с Go (AD-12: «отпечаток сервера = отпечаток агента»).
// Векторы пишет Go-тест cmd/token-agent/internal/agent (TestWasmVectors) тем же
// кодом, которым сервер ждёт подпись; здесь тот же запрос проходит через
// signer.wasm — отпечаток и подписываемые байты обязаны совпасть, «годен»
// уровнем 1 — отклониться, PIN — открыть хранилище.
//
// Запуск: node extension/test/wasm-vectors.mjs ‹каталог с signer.wasm и wasm_exec.js›
import { readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

const here = fileURLToPath(new URL('.', import.meta.url))
const dir = resolve(process.argv[2] || join(here, '..'))
await import(pathToFileURL(join(dir, 'wasm_exec.js')).href)
const go = new globalThis.Go()
const ready = new Promise((resolve) => (globalThis.__glavnySignerReady = resolve))
const { instance } = await WebAssembly.instantiate(readFileSync(join(dir, 'signer.wasm')), go.importObject)
go.run(instance)
await ready
const s = globalThis.glavnySigner
const call = (name, ...args) => JSON.parse(s[name](...args.map((a) => (typeof a === 'string' ? a : JSON.stringify(a)))))

let failed = 0
const check = (ok, msg) => {
  console.log((ok ? 'ok   ' : 'FAIL ') + msg)
  if (!ok) failed++
}

const vectors = JSON.parse(readFileSync(join(here, 'vectors.json'), 'utf8'))
for (const v of vectors) {
  const p = call('prepare', v.block, v.sealed, v.context)
  check(p.ok && p.prepared.doc_digest === v.doc_digest, `${v.name}: отпечаток WASM = Go (${p.prepared?.doc_digest ?? p.message})`)
  check(p.ok && p.prepared.payload_b64 === v.payload_b64, `${v.name}: подписываемые байты WASM = Go`)
  const l1 = call('prepare', { ...v.block, level: 1 }, v.sealed, v.context)
  check(!l1.ok && l1.code === v.level1_refusal_code, `${v.name}: уровень 1 отклонён (${l1.code})`)
  const u = call('unlock', v.sealed, v.pin)
  check(u.ok, `${v.name}: PIN открывает хранилище`)
  const bad = call('unlock', v.sealed, v.pin + '0')
  check(!bad.ok && bad.code === 'signing.pin_wrong', `${v.name}: неверный PIN отклонён`)
  const sg = call('sign', { block: v.block, sealed: v.sealed, dk_b64: u.dk_b64, context: v.context, confirmed_digest: v.doc_digest, journal: [] })
  check(sg.ok && sg.envelope.signatures.length === 1 && sg.journal.length === 1, `${v.name}: подпись и запись локального журнала`)
}
process.exit(failed ? 1 : 0)
