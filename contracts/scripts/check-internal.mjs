// Контракты собственных процессов (AD-46): OpenAPI 3.1 хранителя и demo-signer валидны,
// перечень уровня 1 — факты каталога, схемы классов пакетов существуют.
import fs from 'node:fs';
import path from 'node:path';
import { validate } from '@readme/openapi-parser';
import { CONTRACTS, REPO, readYaml, rel, fail, report } from './lib.mjs';

for (const f of ['internal/keeper.openapi.yaml', 'internal/demo-signer.openapi.yaml']) {
  const p = path.join(CONTRACTS, f);
  try {
    const r = await validate(p);
    if (r && r.valid === false) fail(rel(p), (r.errors || []).map((e) => e.message).join('; '));
  } catch (e) { fail(rel(p), e.message.split('\n')[0]); }
}
const cat = readYaml(path.join(CONTRACTS, 'events/catalog.yaml'));
const l1 = readYaml(path.join(CONTRACTS, 'internal/token-agent/level1-actions.yaml'));
for (const t of l1.event_types) {
  const c = cat.types[t];
  if (!c) fail('level1-actions.yaml', `нет типа ${t}`);
  else if (c.kind !== 'fact' || c.critical) fail('level1-actions.yaml', `${t}: уровень 1 — только некритичные факты (AD-13)`);
}
const pc = readYaml(path.join(CONTRACTS, 'crypto/payload-classes.yaml'));
for (const [k, v] of Object.entries(pc.classes)) if (!fs.existsSync(path.join(REPO, v.schema))) fail('payload-classes.yaml', `${k}: нет ${v.schema}`);
const prof = readYaml(path.join(CONTRACTS, 'crypto/profiles.yaml'));
for (const k of Object.keys(prof.object_profiles)) if (!pc.classes[k]) fail('profiles.yaml', `класс ${k} не объявлен в payload-classes.yaml`);
report('контракты собственных процессов: OpenAPI keeper и demo-signer, уровень 1, классы пакетов');
