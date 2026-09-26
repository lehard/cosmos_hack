// Обнаружение несовместимого изменения контракта событий (AD-20, FR-29, критерий О8):
// @asyncapi/diff сравнивает contracts/events/asyncapi.yaml с базовой версией
// (прошлый тег контракта или ветка main — выбирает make check-compat) и краснеет
// на ломающих изменениях. Совместимые (новый тип, новое необязательное поле)
// проходят. HTTP-контракт сравнивает oasdiff breaking (make check-compat).
//
//   node contracts/scripts/check-compat.mjs ‹база asyncapi.yaml› [‹новый asyncapi.yaml›]
import fs from 'node:fs';
import path from 'node:path';
import { Parser, fromFile } from '@asyncapi/parser';
import { diff } from '@asyncapi/diff';
import { breakingChanges } from './compat-rules.mjs';
import { CONTRACTS, rel } from './lib.mjs';

const [base, next = path.join(CONTRACTS, 'events/asyncapi.yaml')] = process.argv.slice(2);
if (!base || !fs.existsSync(base) || fs.statSync(base).size === 0) {
  console.log('✓ asyncapi diff: базовой версии нет — сравнивать не с чем');
  process.exit(0);
}
const parser = new Parser();
async function load(p) {
  const { document, diagnostics } = await fromFile(parser, p).parse();
  if (!document) {
    console.error(`✗ asyncapi diff: ${p} не разобран: ${diagnostics.filter((d) => d.severity === 0).map((d) => d.message).join('; ')}`);
    process.exit(1);
  }
  return byName(document.json());
}
// Сообщения операции сравниваются по имени, а не по позиции в массиве:
// новый тип в середине отсортированного списка — совместимое добавление
// (FR-29), а не «правка» всех следующих за ним сообщений.
function byName(doc) {
  for (const op of Object.values(doc.operations ?? {})) {
    if (!Array.isArray(op.messages)) continue;
    const m = {};
    for (const [i, msg] of op.messages.entries()) {
      const key = msg?.name ?? msg?.messageId ?? msg?.['x-parser-message-name'] ?? msg?.payload?.allOf?.[1]?.properties?.event_type?.const ?? String(i);
      m[key] = msg;
    }
    op.messages = m;
  }
  return doc;
}
const b = await load(base);
const n = await load(next);
const out = diff(b, n, { outputType: 'json' });
const breaking = breakingChanges(b, n);
if (breaking.length) {
  console.error(`✗ asyncapi diff: ломающие изменения контракта событий (нужна новая мажорная версия схемы и повышатель, AD-20):`);
  for (const c of breaking) console.error(`  - ${c.action} ${c.path}`);
  process.exit(1);
}
console.log(`✓ asyncapi diff: ломающих изменений нет (${rel(next)} против базы; совместимых: ${out.nonBreaking().length})`);
