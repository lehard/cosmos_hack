# deploy/k8s

Конфигурация масштабирования промышленного окружения для Kubernetes (эпик 35; AD-6, AD-25,
FR-112). Описание, без локального запуска: `kubectl kustomize --load-restrictor
LoadRestrictionsNone deploy/k8s | kubectl apply -f -` (конфигурация `ant.yaml` берётся из
`deploy/config`) при готовых
секретах и внешнем Postgres. Правила масштабирования и узкие места —
[docs/scaling.md](../../docs/scaling.md); как проверено, что смысл от числа обработчиков
не зависит, — `make load` ([scenarios/load](../../scenarios/load/README.md)).

| Файл | Что | Копий | Как масштабируется |
|---|---|---|---|
| `migrate-job.yaml` | разовая роль `migrate` (роли БД, миграции), ConfigMap `ant-env` (профиль `prod`) | Job | перед выкладкой версии |
| `api.yaml` | роль `api`, Service, HPA, PDB, том материалов (RWX) | 2…8 | горизонтально по CPU: без состояния, сеансы в Postgres |
| `worker.yaml` | роль `worker`, HPA, PDB | 2…16 (≤ P = 64) | партиции `hash(item_id) mod P` делятся арендами с эпохой; изменение N — перераспределение, окно стабилизации 1–5 мин |
| `leaders.yaml` | `crossitem, projector, scheduler, outbox, security` | 2 (активная + резерв) | не масштабируются: одна активная копия по аренде — порядок межизделийных расчётов и единственность отправки |
| `trust.yaml` | хранитель (StatefulSet, свой том) и верификатор (читает реплику) | по 1 | границы доверия, не масштабируются |
| `policies.yaml` | NetworkPolicy: всё входящее закрыто, api — от ingress, хранитель — от лидеров и верификатора | — | — |

Секреты, которые готовит эксплуатация (`kubectl create secret generic …`): `ant-db`
(`db_password`), `ant-kek` (`kek`), `ant-pki` (сертификат mTLS ant), `ant-verifier`
(ключ и trust-anchors верификатора). Postgres — внешний кластер (например, CloudNativePG):
сервис `ant-postgres-rw` (primary) и `ant-postgres-ro` (реплики чтения для верификатора,
аналитики и воспроизведения, AD-6).

**Соединения с Postgres.** Каждая копия держит свой пул (`ANT_DB_MAX_CONNS`): api 16,
worker 12, лидеры 24. При максимуме HPA (8 api + 16 worker + 2 лидера) это до 368
соединений — нужен `max_connections` с запасом или PgBouncer в режиме transaction
перед primary (LISTEN у каждой копии — отдельное соединение в обход пула).

**Что не описано здесь.** stands и пульт сценариев (профили demo, load) в prod не
запускаются; edge-агенты стоят у оборудования, а не в кластере.
