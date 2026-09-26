package main

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ant/cmd/internal/config"
	"ant/cmd/internal/db"
	appingest "ant/internal/application/ingest"
	processapp "ant/internal/application/process"
	appsigning "ant/internal/application/signing"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/clock"
	storesigning "ant/internal/infrastructure/storage/signing"
)

// Роль init — `ant init` (эпик 05; AD-33, AD-11, AD-32; FR-10, FR-109;
// критерий Т1): разовый шаг перед остальными службами compose — миграции,
// ключи в тома (0400), сертификаты mTLS, блок генезиса (seq 1…k класса
// genesis, подписан ключом-якорем hybrid) по затравке normative/, уничтожение
// якоря, trust-anchors в томе хранителя. Генезис уже есть — журнал не
// трогается. Логика — infrastructure/storage/signing.Provision; здесь — только
// сборка: тома из переменных окружения (ANT_*_DIR, пути томов compose).
//
// Здесь же — то, что другие роли берут из генезиса вместо временных путей:
// ключи приёма и подпись шлюза (genesisIngest), стартовый процесс
// (genesisProcessXML); паспорта анализаторов и политика — записи блока.

// Роль регистрируется здесь, а не в реестре main.go, чтобы не править общий файл.
func init() { roles["init"] = role{run: runInit, oneShot: true} }

// genesisBatchMax — предел пачки для записи блока генезиса одной транзакцией
// (блок — несколько сотен записей, больше journal.batch_max по умолчанию).
const genesisBatchMax = 4096

// initVolumes — тома ключей (переменные окружения; по умолчанию — пути томов compose).
func initVolumes() storesigning.Volumes {
	return storesigning.Volumes{
		AntKeys:    envOr("ANT_KEYS_DIR", "/var/lib/ant-keys"),
		Keeper:     envOr("ANT_KEEPER_DIR", "/var/lib/keeper"),
		Verifier:   envOr("ANT_VERIFIER_DIR", "/var/lib/verifier"),
		AntPKI:     envOr("ANT_PKI_DIR", "/var/lib/ant-pki"),
		DemoSigner: envOr("ANT_DEMO_SIGNER_KEYS", "/var/lib/demo-signer/keys"),
		TokenAgent: os.Getenv("ANT_TOKEN_AGENT_KEYS"),
		Devices:    envOr("ANT_DEVICE_KEYS", "/var/lib/edge-agent/keys"),
		Partner:    envOr("ANT_PARTNER_KEYS", "/var/lib/ant-keys/partner"),
	}
}

// clockMode — режим часов журнала для генезиса (AD-37): в профиле demo журнал
// ведётся в режиме scenario (прогоны сценариев задают доменное «сейчас»).
func clockMode(cfg *config.Config) string {
	if cfg.Profile == config.ProfileDemo {
		return appsigning.ClockScenario
	}
	return appsigning.ClockSystem
}

// runInit — разовая роль init.
func runInit(ctx context.Context, env *environment) error {
	if err := runMigrate(ctx, env); err != nil {
		return err
	}
	pc, err := db.Config(env.cfg.DB, "ant-init")
	if err != nil {
		return err
	}
	pool, err := journalstore.NewAppPool(ctx, pc)
	if err != nil {
		return err
	}
	defer pool.Close()
	// TODO(29): trustOptions(env.cfg, env) — записи CA и шифрование при
	// хранении, как у ядра (после слияния эпика 29).
	j := journalstore.NewStore(pool, clock.System{}, journalstore.WithBatchMax(genesisBatchMax))
	res, err := storesigning.Provision(ctx, j, storesigning.ProvisionConfig{Volumes: initVolumes(), Profile: env.cfg.Profile,
		ClockMode: clockMode(env.cfg), ProcessVersionID: processapp.SeedVersionID, DomainBuild: domainBuild(),
		Partitions: env.cfg.Engine.Partitions, KeeperHosts: strings.Split(envOr("ANT_KEEPER_HOSTS", "keeper,localhost"), ","), Log: env.log})
	if err != nil {
		return err
	}
	if os.Geteuid() == 0 {
		// Тома, которых нет в образе, docker создаёт от root: созданное —
		// владельцу-процессу (65532), ключи агента токена — пользователю машины.
		v := initVolumes()
		owner, agent := envOr("ANT_INIT_OWNER", "65532:65532"), envOr("ANT_TOKEN_AGENT_OWNER", "65532:65532")
		for _, x := range []struct{ dir, owner string }{{v.AntKeys, owner}, {v.Keeper, owner}, {v.Verifier, owner}, {v.AntPKI, owner},
			{v.DemoSigner, owner}, {v.Devices, owner}, {v.Partner, owner}, {v.TokenAgent, agent}} {
			if err := chownTree(x.dir, x.owner); err != nil {
				return err
			}
		}
	}
	env.log.Info("init: готово", "genesis_written", res.Written, "block_size", res.Header.BlockSize,
		"anchor", res.Header.AnchorFingerprint, "genesis_digest", res.Digest)
	return nil
}

// genesisIngest — порты подписи приёма из генезиса (FR-26, AD-11): реестр
// ключей — свёртка key.* журнала (генезис — первые записи), проверка подписи
// профилями gost/pq/hybrid, подпись служебных записей ключом шлюза
// gateway-ingest@1 из тома ant (ANT_KEYS_DIR). Ключа в томе нет — записи
// шлюза без подписи (как до эпика 05).
func genesisIngest(env *environment, c *core) (appsigning.Verifier, appingest.KeyRegistry, appsigning.Signer) {
	v, keys, signer, err := storesigning.IngestSigning(c.journal, envOr("ANT_KEYS_DIR", "/var/lib/ant-keys"))
	if err != nil {
		env.log.Warn("приём: ключи тома ant не читаются — служебные записи без подписи шлюза", "err", err)
	}
	if signer == nil {
		return v, keys, nil
	}
	return v, keys, signer
}

// genesisProcessXML — стартовая версия процесса из генезиса (FR-10, AD-17,
// AD-33): отпечаток закреплён записью normative.version.loaded блока, байты
// сверяются с ним. Генезиса нет (разработка и тесты без ant init) — fallback
// (встроенная копия normative/) с предупреждением.
func genesisProcessXML(ctx context.Context, c *core, env *environment, fallback func() ([]byte, error)) ([]byte, error) {
	id, xml, err := storesigning.GenesisProcess(ctx, c.journal, nil)
	switch {
	case errors.Is(err, storesigning.ErrNoGenesis):
		env.log.Warn("процесс: генезиса нет (ant init) — стартовая версия из встроенной копии normative/")
		return fallback()
	case err != nil:
		return nil, err
	case id != processapp.SeedVersionID:
		return nil, errors.New("процесс: генезис закрепил версию " + id + ", ожидалась " + processapp.SeedVersionID)
	}
	return xml, nil
}

// chownTree — владелец uid:gid каталога dir и всего внутри (нет каталога — ничего).
func chownTree(dir, owner string) error {
	if dir == "" {
		return nil
	}
	u, g, ok := strings.Cut(owner, ":")
	uid, err1 := strconv.Atoi(u)
	gid, err2 := strconv.Atoi(g)
	if !ok || err1 != nil || err2 != nil {
		return errors.New("init: владелец " + owner + " — ожидается uid:gid")
	}
	err := filepath.WalkDir(dir, func(p string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Lchown(p, uid, gid)
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
