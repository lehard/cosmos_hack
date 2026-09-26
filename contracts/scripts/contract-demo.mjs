// make contract-demo (FR-111, кейс §6.2, критерии О7, О8): воспроизводимое
// несовместимое изменение контракта обнаруживается сборкой ДО отправки
// результата в 1С. Копия contracts/events, в копии у схемы исходящего учётного
// сообщения erp.posting.requested v1 удалено обязательное поле action (без новой
// мажорной версии — так делать нельзя, AD-20), затем @asyncapi/diff сравнивает
// копию с текущим контрактом. Демо успешно, если изменение признано ломающим.
// Правильный путь того же изменения — новая мажорная версия схемы и повышатель
// (пример: equipment.state.changed v1 → v2, contracts/events/examples).
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { Parser, fromFile } from '@asyncapi/parser';
import { breakingChanges } from './compat-rules.mjs';
import { CONTRACTS } from './lib.mjs';

const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'contract-demo-'));
const next = path.join(tmp, 'events');
fs.cpSync(path.join(CONTRACTS, 'events'), next, { recursive: true });
const schemaPath = path.join(next, 'erp/erp.posting.requested.v1.json');
const schema = JSON.parse(fs.readFileSync(schemaPath, 'utf8'));
delete schema.properties.action;
schema.required = schema.required.filter((f) => f !== 'action');
fs.writeFileSync(schemaPath, JSON.stringify(schema, null, 2));
console.log('contract-demo: в копии контракта у erp.posting.requested v1 удалено обязательное поле action (без новой версии)');

const parser = new Parser();
const load = async (p) => {
  const { document, diagnostics } = await fromFile(parser, p).parse();
  if (!document) throw new Error(`${p}: ${diagnostics.map((d) => d.message).join('; ')}`);
  return document.json();
};
const breaking = breakingChanges(await load(path.join(CONTRACTS, 'events/asyncapi.yaml')), await load(path.join(next, 'asyncapi.yaml')))
  .filter((c) => c.path.includes('erp.posting.requested'));
fs.rmSync(tmp, { recursive: true, force: true });
if (!breaking.length) {
  console.error('✗ contract-demo: ломающее изменение НЕ обнаружено');
  process.exit(1);
}
for (const c of breaking) console.log(`  ломающее: ${c.action} ${c.path}`);
console.log('✓ contract-demo: несовместимое изменение контракта обнаружено сборкой до отправки в 1С (make check-compat краснеет на такой ветке)');
