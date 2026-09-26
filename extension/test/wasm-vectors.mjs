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

  // Ключи многих персон разом под один PIN (Д-72, демо из одного браузера):
  // argon2id один раз, по хранилищу на ключ; подпись — ключами одного человека.
  if (v.key_files?.length) {
    const many = call('sealEach', { files: v.key_files, storage_variant: 'extension', now: v.context.now }, v.pin)
    check(many.ok && many.sealed.length === v.key_files.length && new Set(many.sealed.map((x) => x.kdf.salt_b64)).size === 1, `${v.name}: ${v.key_files.length} ключа под одним PIN, одна соль`)
    const person = many.sealed[0].person_id
    const mine = many.sealed.filter((x) => x.person_id === person)
    const other = many.sealed.find((x) => x.person_id !== person)
    const ctx = { ...v.context, profile: 'hybrid' }
    const hp = call('prepare', v.block, mine, ctx)
    const hs = hp.ok && call('sign', { block: v.block, sealed: mine, dk_b64: many.dk_b64, context: ctx, confirmed_digest: hp.prepared.doc_digest, journal: [] })
    check(hs && hs.ok && hs.envelope.signatures.length === 2, `${v.name}: hybrid ключами ${person} из двух хранилищ, один ключ сеанса`)
    const gp = call('prepare', v.block, mine, v.context)
    check(gp.ok && gp.prepared.doc_digest === v.doc_digest, `${v.name}: gost из набора — тот же отпечаток, что у Go`)
    const again = call('sealEach', { files: [v.key_files[0]], now: v.context.now, existing: many.sealed[0] }, v.pin + '0')
    check(!again.ok && again.code === 'signing.pin_wrong', `${v.name}: дозагрузка под другим PIN отклонена`)
    if (other) {
      const mixed = call('prepare', v.block, [mine[0], other], v.context)
      check(!mixed.ok, `${v.name}: ключи двух людей в одной подписи отклонены («суперключа» нет)`)
    }
  }
}
process.exit(failed ? 1 : 0)
