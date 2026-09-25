// Проверка таблиц контракта, ссылающихся на каталог: уровни доверия анализатора (AD-29).
import path from 'node:path';
import { CONTRACTS, readYaml, fail, report } from './lib.mjs';

const cat = readYaml(path.join(CONTRACTS, 'events/catalog.yaml'));
const tl = readYaml(path.join(CONTRACTS, 'analyzer-trust-levels.yaml'));
const levels = tl.levels.map((l) => l.level);
if (JSON.stringify(levels) !== '[0,1,2,3,4]') fail('analyzer-trust-levels.yaml', 'уровни должны быть 0…4 по порядку');
for (const l of tl.levels) for (const a of l.actions) {
  const t = cat.types[a.type];
  if (!t) { fail(`analyzer-trust-levels.yaml уровень ${l.level}`, `нет типа ${a.type} в каталоге`); continue; }
  if (t.action_class === 'irreversible') fail(`уровень ${l.level}`, `${a.type} — необратимое, автоматике запрещено (AD-27)`);
  if (t.action_class === 'permissive' && l.level < 4) fail(`уровень ${l.level}`, `${a.type} — разрешающее, только с уровня 4 (AD-29)`);
}
for (const f of tl.forbidden_always) if (!cat.types[f.type]) fail('analyzer-trust-levels.yaml forbidden_always', `нет типа ${f.type}`);
report('таблица уровней доверия анализатора (AD-29)');
