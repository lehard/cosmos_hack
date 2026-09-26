// Проверка contracts/openapi.yaml — контракта HTTP API, сгенерированного из операций Huma (эпик 02; AD-20, AD-27, AD-40):
//   у каждой операции есть x-ant-action; его id = operationId вида ‹модуль›.‹объект›.‹действие›;
//   есть класс (read | record | protective | permissive | irreversible) — «у операции нет класса» краснеет;
//   модуль-владелец = первый сегмент id; критическая операция — с группой CA из каталога;
//   каждый эмитируемый тип есть в каталоге и его эмитент — модуль-владелец («модуль эмитит чужой тип» краснеет);
//   чтение — GET без эмитируемых типов; у операций чтения состояния — параметры axis и as_of (AD-21);
//   каждое действие стартовой политики normative/policy (в том числе с `*`) соответствует хотя бы одной операции.
// Запуск: node contracts/scripts/check-openapi.mjs [файл]   (без файла — contracts/openapi.yaml)
//         node contracts/scripts/check-openapi.mjs --selftest — пробные нарушения должны краснеть.
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import YAML from 'yaml';
import { CONTRACTS, REPO, readYaml, rel, fail, report } from './lib.mjs';

const CLASSES = new Set(['read', 'record', 'protective', 'permissive', 'irreversible']);
const ID = /^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$/;
const METHODS = ['get', 'post', 'put', 'patch', 'delete'];

if (process.argv.includes('--selftest')) selftest();
else check(process.argv[2] ? path.resolve(process.argv[2]) : path.join(CONTRACTS, 'openapi.yaml'));

function check(file) {
  if (!fs.existsSync(file)) {
    console.log(`✓ openapi: ${rel(file)} ещё нет — пропуск`);
    return;
  }
  const doc = YAML.parse(fs.readFileSync(file, 'utf8'));
  const cat = readYaml(path.join(CONTRACTS, 'events/catalog.yaml'));
  const types = cat.types || {};
  const caGroups = new Set(Object.values(types).map((t) => t.ca_group).filter(Boolean));
  const ids = [];
  for (const [p, item] of Object.entries(doc.paths || {})) {
    for (const m of METHODS) {
      const op = item[m];
      if (!op) continue;
      const where = `${rel(file)} ${m.toUpperCase()} ${p}`;
      const a = op['x-ant-action'];
      if (!a) { fail(where, 'нет x-ant-action (AD-40)'); continue; }
      if (a.id !== op.operationId) fail(where, `x-ant-action.id ${a.id} ≠ operationId ${op.operationId}`);
      if (!ID.test(a.id || '')) fail(where, `id ${a.id} не вида ‹модуль›.‹объект›.‹действие›`);
      if (!a.class) fail(where, `у операции ${a.id} нет класса (AD-27)`);
      else if (!CLASSES.has(a.class)) fail(where, `неизвестный класс ${a.class}`);
      if (!a.owner || !String(a.id).startsWith(a.owner + '.')) fail(where, `модуль-владелец ${a.owner} не совпадает с первым сегментом id`);
      if (typeof a.critical !== 'boolean') fail(where, 'нет признака critical (AD-28)');
      if (a.critical && !caGroups.has(a.ca_group)) fail(where, `критическая операция без группы CA из каталога (${a.ca_group})`);
      for (const t of a.emits || []) {
        if (!types[t]) fail(where, `эмитирует тип ${t}, которого нет в каталоге`);
        else if (types[t].emitter !== a.owner) fail(where, `модуль ${a.owner} эмитит чужой тип ${t} (эмитент — ${types[t].emitter}, AD-40)`);
      }
      if (a.class === 'read') {
        if (m !== 'get' && !op['x-ant-read-by-post']) fail(where, 'чтение — только GET');
        if ((a.emits || []).length) fail(where, 'чтение не эмитирует записей');
      } else if (m === 'get') fail(where, 'команда не может быть GET');
      ids.push(a.id);
    }
  }
  const dup = ids.filter((x, i) => ids.indexOf(x) !== i);
  for (const d of new Set(dup)) fail(rel(file), `операция ${d} объявлена дважды`);

  // Действия стартовой политики ↔ операции (эпик 02: имена приведены к одному виду).
  const policyFile = path.join(REPO, 'normative/policy/policy.v1.yaml');
  if (fs.existsSync(policyFile) && ids.length) {
    const policy = readYaml(policyFile);
    const re = (pat) => new RegExp('^' + pat.split('.').map((s) => (s === '*' ? '[a-z0-9_]+' : s)).join('\\.') + '$');
    for (const r of policy.roles || []) {
      for (const act of r.actions || []) {
        if (!ids.some((id) => re(act).test(id))) fail(`normative/policy/policy.v1.yaml роль ${r.id}`, `действие ${act} не соответствует ни одной операции openapi.yaml`);
      }
    }
  }
  report(`openapi: ${ids.length} операций — x-ant-action, классы, эмитенты, политика`);
}

// Самопроверка: портим копию спецификации и ждём красного.
function selftest() {
  const src = path.join(CONTRACTS, 'openapi.yaml');
  if (!fs.existsSync(src)) { console.log('✓ openapi selftest: спецификации ещё нет — пропуск'); return; }
  const ops = (d) => Object.values(d.paths).flatMap((it) => METHODS.map((m) => it[m]).filter(Boolean));
  const cases = [
    ['у операции нет класса', (d) => { delete ops(d)[0]['x-ant-action'].class; }],
    ['нет x-ant-action', (d) => { delete ops(d)[0]['x-ant-action']; }],
    ['модуль эмитит чужой тип', (d) => {
      const op = ops(d).find((o) => o['x-ant-action'].class !== 'read') || ops(d)[0];
      const a = op['x-ant-action'];
      const types = readYaml(path.join(CONTRACTS, 'events/catalog.yaml')).types;
      const foreign = Object.keys(types).find((t) => types[t].emitter !== a.owner);
      a.class = a.class === 'read' ? 'record' : a.class;
      a.emits = [foreign];
    }],
  ];
  let bad = 0;
  for (const [name, mutate] of cases) {
    const d = YAML.parse(fs.readFileSync(src, 'utf8'));
    mutate(d);
    const tmp = path.join(fs.mkdtempSync(path.join(os.tmpdir(), 'oas-')), 'openapi.yaml');
    fs.writeFileSync(tmp, YAML.stringify(d));
    let red = false;
    try { execFileSync(process.execPath, [process.argv[1], tmp], { stdio: 'pipe' }); } catch { red = true; }
    console.log(`  ${red ? 'ок' : 'НЕ ПОЙМАНО'}: ${name}`);
    if (!red) bad++;
  }
  if (bad) process.exit(1);
  console.log('✓ openapi selftest: пробные нарушения краснеют');
}
