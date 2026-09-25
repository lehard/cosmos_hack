// Сборка и проверка contracts/events/asyncapi.yaml (AsyncAPI 3.0) из catalog.yaml и схем (AD-20).
//   node asyncapi.mjs --write   — пересобрать файл;
//   node asyncapi.mjs           — проверить, что файл собран из текущего каталога, и провалидировать парсером @asyncapi/parser.
import fs from 'node:fs';
import path from 'node:path';
import YAML from 'yaml';
import { Parser, fromFile } from '@asyncapi/parser';
import { CONTRACTS, readYaml, readJson, rel, fail, report } from './lib.mjs';

const EV = path.join(CONTRACTS, 'events');
const OUT = path.join(EV, 'asyncapi.yaml');
const cat = readYaml(path.join(EV, 'catalog.yaml'));

function build() {
  const channels = {};
  const operations = {};
  const messages = {};
  const ingestMsgs = {};
  const byFamily = {};
  for (const [name, t] of Object.entries(cat.types)) {
    const fam = name.split('.')[0];
    for (const v of t.versions) {
      const key = `${name}.v${v}`;
      const schemaRef = `./${fam}/${name}.v${v}.json`;
      const schema = readJson(path.join(EV, fam, `${name}.v${v}.json`));
      messages[key] = {
        name: key,
        title: t.title,
        summary: schema.description,
        contentType: 'application/json',
        payload: {
          allOf: [
            { $ref: './common/envelope.v1.json' },
            {
              type: 'object',
              properties: {
                event_type: { const: name },
                schema_version: { const: v },
                data: { $ref: schemaRef },
              },
            },
          ],
        },
        'x-ant-kind': t.kind,
        'x-ant-emitter': t.emitter,
        'x-ant-stream': t.stream,
        'x-ant-axis': t.axis,
        'x-ant-action-class': t.action_class,
        'x-ant-critical': t.critical,
        'x-ant-current': v === t.current_version,
      };
      (byFamily[fam] ||= {})[key] = { $ref: `#/components/messages/${key}` };
      if (t.kind === 'fact') ingestMsgs[key] = { $ref: `#/components/messages/${key}` };
    }
  }
  for (const [fam, msgs] of Object.entries(byFamily)) {
    const f = cat.families[fam];
    channels[fam] = {
      address: `journal/${fam}`,
      title: `Журнал: семейство ${fam}`,
      description: `${f.title}. Владелец схемы — модуль ${f.owner}. Записи читают потребители журнала (AD-45); порядок — по seq.`,
      messages: msgs,
    };
    operations[`journal.${fam}`] = {
      action: 'send',
      title: `ant записывает записи семейства ${fam}`,
      channel: { $ref: `#/channels/${fam}` },
      messages: Object.keys(msgs).map((k) => ({ $ref: `#/channels/${fam}/messages/${k}` })),
    };
  }
  channels.facts = {
    address: '/api/v1/ingest/events',
    title: 'Приём фактов от источников',
    description: 'Подписанные пакеты DSSE (ключ источника, source_seq) от edge-агента, шлюзов, терминалов и импорта; пачки — той же операцией приёма из openapi.yaml (AD-46). Проверка тела — этими же схемами (AD-20).',
    messages: ingestMsgs,
  };
  operations['ingest.facts'] = {
    action: 'receive',
    title: 'ant принимает факты',
    channel: { $ref: '#/channels/facts' },
    messages: Object.keys(ingestMsgs).map((k) => ({ $ref: `#/channels/facts/messages/${k}` })),
  };
  messages.EntityChanged = {
    name: 'EntityChanged',
    title: 'Сущность изменилась',
    summary: 'Живое обновление: сущность, id, seq (AD-21).',
    contentType: 'application/json',
    payload: { $ref: './common/sse-entity-changed.v1.json' },
  };
  channels.sse = {
    address: '/api/v1/stream',
    title: 'SSE: живые обновления столов',
    description: 'Server-Sent Events: `id:` = seq, `event: entity_changed`, `data:` — EntityChanged. Фронтенд инвалидирует ключ Vue Query [сущность, id] (AD-21); переподключение — с Last-Event-ID.',
    messages: { EntityChanged: { $ref: '#/components/messages/EntityChanged' } },
  };
  operations['sse.entityChanged'] = {
    action: 'send',
    title: 'ant отправляет живые обновления',
    channel: { $ref: '#/channels/sse' },
    messages: [{ $ref: '#/channels/sse/messages/EntityChanged' }],
  };
  const doc = {
    asyncapi: '3.0.0',
    info: {
      title: 'ant — события журнала, приём фактов и живые обновления',
      version: `${cat.catalog_version}.0.0`,
      description: 'Контракт событий v1 (кейс §4.4–4.7, FR-27…29, FR-110). Файл собирается из contracts/events/catalog.yaml и схем командой `node contracts/scripts/asyncapi.mjs --write` — руками не правится. Конверт — common/envelope.v1.json, данные типа — ‹семейство›/‹тип›.v‹N›.json.',
    },
    defaultContentType: 'application/json',
    servers: {
      ant: { host: 'localhost:8443', protocol: 'https', description: 'Экземпляр ant (в демо — localhost с сертификатом от ant init).' },
    },
    channels,
    operations,
    components: { messages },
  };
  return '# Сгенерировано contracts/scripts/asyncapi.mjs из catalog.yaml — не редактировать руками.\n' + YAML.stringify(doc, { lineWidth: 0 });
}

const text = build();
if (process.argv.includes('--write')) {
  fs.writeFileSync(OUT, text);
  console.log(`записан ${rel(OUT)}`);
} else {
  if (!fs.existsSync(OUT) || fs.readFileSync(OUT, 'utf8') !== text) fail(rel(OUT), 'не собран из текущего catalog.yaml — запустите node contracts/scripts/asyncapi.mjs --write');
  const parser = new Parser();
  const { document, diagnostics } = await fromFile(parser, OUT).parse();
  const errs = diagnostics.filter((d) => d.severity === 0);
  for (const d of errs) fail(rel(OUT), `${d.code}: ${d.message} @ ${(d.path || []).join('.')}`);
  if (!document && !errs.length) fail(rel(OUT), 'парсер не вернул документ');
  report(`AsyncAPI 3.0 валиден и собран из каталога (${Object.keys(cat.types).length} типов, SSE-канал)`);
}
