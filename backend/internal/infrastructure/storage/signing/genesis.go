package signing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ant/internal/application/journal"
	app "ant/internal/application/signing"
	"ant/internal/contracts/catalog"
	dom "ant/internal/domain/signing"
	"ant/internal/infrastructure/security/pki"
	"ant/internal/infrastructure/security/profiles"
)

// Роль ant init (AD-33, AD-11, AD-32; FR-10, FR-109; критерий Т1): ключи в
// тома (0400), сертификаты mTLS, блок генезиса по составу AD-33 из затравки
// normative/, уничтожение ключа-якоря, файл trust-anchors в томе хранителя
// (и его копия в томе верификатора). Генезис уже есть — журнал не трогается
// (только сверка и, если нужно, дозапись trust-anchors).

// Volumes — тома ключей (пути внутри контейнера init).
type Volumes struct {
	// AntKeys — том ant: ключи движка и шлюзов (‹key_ref›.key.json).
	AntKeys string
	// Keeper — том хранителя: keys/keeper.{gost,pq}, pki/, trust-anchors.json.
	Keeper string
	// Verifier — том верификатора: keys/verifier.{gost,pq}, pki/, копия trust-anchors.json.
	Verifier string
	// AntPKI — сертификат mTLS ant (pki/ant.{crt,key}, ca.crt).
	AntPKI string
	// DemoSigner — ключи демо-персон класса scenario (том demo-signer).
	DemoSigner string
	// TokenAgent — ключи интерактивных демо-персон для агента токена
	// (./.demo-keys/token-agent/ вне репозитория); пусто или не пишется — пропуск.
	TokenAgent string
	// Devices — ключи устройств для edge-агентов (‹source_id›.key).
	Devices string
	// Partner — корни партнёра федерации (демо: второе предприятие — эта же система).
	Partner string
}

// AnchorsFile — имя файла trust-anchors в томах keeper и verifier (как у эпика 29).
const AnchorsFile = "trust-anchors.json"

// Имена ключей системы (AD-11): движок и шлюзы — в томе ant.
const (
	EngineKeyRef            = "engine@1"
	GatewayKeyRef           = "gateway-ingest@1"
	EnterpriseGatewayKeyRef = "gateway-enterprise@1"
	// GatewaySource — source_id шлюза приёма (application/ingest GatewaySource).
	GatewaySource = "ant-ingest"
	// EngineSubject — субъект ключа движка.
	EngineSubject = "ant-engine"
)

// InteractivePersonas — демо-персоны для агента токена (жюри подписывает
// интерактивно, AD-33): держатели пяти ролей кейса и подписанты актов.
// Ключи — ‹псевдоним›-ta@1 и ‹псевдоним›-ta-pq@1, субъект demo_persona.
var InteractivePersonas = []string{"INS-01", "FOR-WC", "TEC-01", "PM-01", "ADM-01", "HQC-01", "AUD-01"}

// PersonaClasses — классы пакетов ключей демо-персон (как у demo-signer).
var PersonaClasses = []string{dom.ClassEvent, dom.ClassDocumentSignature, dom.ClassPaperAttestation, dom.ClassShiftReport, dom.ClassKeyAct}

// DemoValidFrom — начало действия стартовой политики, справочников и ключей
// в профилях затравки: прогоны сценариев идут в виртуальном времени 2026 года.
var DemoValidFrom = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// ProvisionConfig — параметры ant init.
type ProvisionConfig struct {
	Volumes
	// Profile — профиль окружения (prod — без демо-персон и паспортов затравки).
	Profile string
	// ClockMode — режим часов журнала (system | scenario).
	ClockMode string
	// ProcessVersionID — id стартовой версии процесса.
	ProcessVersionID string
	DomainBuild      string
	Partitions       int
	// Seed — затравка normative/; nil — встроенная копия.
	Seed fs.FS
	// KeeperHosts — имена хранителя в серверном сертификате.
	KeeperHosts []string
	// Now — InfraClock; nil — time.Now.
	Now func() time.Time
	Log *slog.Logger
}

// Provisioned — итог ant init.
type Provisioned struct {
	// Written — блок записан сейчас (false — генезис уже был).
	Written bool
	Header  dom.GenesisHeader
	Digest  string
}

// Provision — ant init над журналом j (миграции уже применены).
func Provision(ctx context.Context, j journal.JournalStore, cfg ProvisionConfig) (Provisioned, error) {
	log := cfg.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	now := time.Now
	if cfg.Now != nil {
		now = cfg.Now
	}
	fsys := cfg.Seed
	if fsys == nil {
		fsys = Seed()
	}
	st, err := app.FindGenesis(ctx, j)
	if err != nil {
		return Provisioned{}, err
	}
	if st.Present {
		// AD-33: генезис есть — ничего не делаем; сверяем блок по якорю и
		// дописываем trust-anchors, если прошлый запуск оборвался после записи.
		pin, have := pinned(cfg.Keeper)
		rep, err := app.VerifyGenesis(ctx, j, profiles.Verifier{}, pin)
		if err != nil {
			return Provisioned{}, err
		}
		if !have {
			if err := writeAnchors(cfg, rep.Header, rep.Digest); err != nil {
				return Provisioned{}, err
			}
		}
		log.Info("генезис уже есть — ничего не делаю", "block_size", rep.Header.BlockSize, "genesis_digest", rep.Digest)
		return Provisioned{Header: rep.Header, Digest: rep.Digest}, nil
	}
	for _, d := range [][2]string{{"ant", cfg.AntKeys}, {"keeper", cfg.Keeper}, {"verifier", cfg.Verifier}, {"устройств", cfg.Devices},
		{"партнёра", cfg.Partner}, {"demo-signer", cfg.DemoSigner}} {
		if d[1] == "" && (d[0] != "demo-signer" || cfg.Profile != "prod") {
			return Provisioned{}, fmt.Errorf("init: не задан том ключей %s", d[0])
		}
	}
	// Журнал не пуст, а генезиса нет — ключи не создаём (AD-33: блок — только seq 1…k).
	if es, err := j.Read(ctx, journal.ReadQuery{Limit: 1}); err != nil || len(es) > 0 {
		if err == nil {
			err = app.ErrJournalNotEmpty
		}
		return Provisioned{}, err
	}
	seed, err := app.LoadSeed(fsys)
	if err != nil {
		return Provisioned{}, fmt.Errorf("затравка normative/: %w", err)
	}
	validFrom := DemoValidFrom
	if cfg.Profile == "prod" {
		validFrom = now().UTC()
	}
	keys, quorum, keeperFP, err := provisionKeys(cfg, seed, validFrom, log)
	if err != nil {
		return Provisioned{}, err
	}
	var block app.GenesisBlock
	written, err := app.EnsureGenesis(ctx, j, func() (app.GenesisBlock, error) {
		// Ключ-якорь (hybrid) — только в памяти init: на диск не пишется и
		// после подписи блока выходит из области видимости — уничтожен (AD-33).
		anchorG, err := profiles.Generate(dom.AnchorRefs[0], dom.ProfileGost)
		if err != nil {
			return app.GenesisBlock{}, err
		}
		anchorP, err := profiles.Generate(dom.AnchorRefs[1], dom.ProfilePQ)
		if err != nil {
			return app.GenesisBlock{}, err
		}
		anchor := []dom.AnchorKey{{KeyRef: anchorG.Ref, ProfileID: anchorG.Profile, PublicB64: anchorG.PublicB64()},
			{KeyRef: anchorP.Ref, ProfileID: anchorP.Profile, PublicB64: anchorP.PublicB64()}}
		spec, err := app.ComposeGenesis(app.GenesisInput{Profile: cfg.Profile, ClockMode: cfg.ClockMode,
			ProcessVersionID: cfg.ProcessVersionID, Keys: keys, KeeperFingerprint: keeperFP, ValidFrom: validFrom, Anchor: anchor}, seed)
		if err != nil {
			return app.GenesisBlock{}, err
		}
		signer := profiles.Signer{Keys: profiles.NewKeyring(append(quorum, anchorG, anchorP)...)}
		block, err = app.BuildGenesis(ctx, spec, signer, app.GenesisConfig{DomainBuild: cfg.DomainBuild, Partitions: cfg.Partitions, Now: now})
		return block, err
	})
	if err != nil {
		return Provisioned{}, err
	}
	if !written {
		return Provisioned{}, errors.New("генезис записан другим процессом одновременно — повторите ant init")
	}
	if err := writeAnchors(cfg, block.Header, block.Digest); err != nil {
		return Provisioned{}, err
	}
	log.Info("генезис записан; ключ-якорь уничтожен", "block_size", block.Header.BlockSize, "anchor", block.Header.AnchorFingerprint,
		"genesis_digest", block.Digest, "keys", len(keys))
	return Provisioned{Written: true, Header: block.Header, Digest: block.Digest}, nil
}

// pinned — отпечаток якоря, закреплённый в trust-anchors тома хранителя, и
// есть ли в файле отпечаток генезиса.
func pinned(keeper string) (string, bool) {
	if keeper == "" {
		return "", true
	}
	t, err := profiles.LoadTrustAnchors(filepath.Join(keeper, AnchorsFile))
	if err != nil {
		return "", false
	}
	return t.AnchorFingerprint, t.GenesisDigest != ""
}

// provisionKeys — ключи всех субъектов блока в их тома и данные регистрации;
// quorum — ключи демо-персон (подписи кворума нормативного слоя).
func provisionKeys(cfg ProvisionConfig, seed app.Seed, validFrom time.Time, log *slog.Logger) ([]app.RegistrationData, []*profiles.PrivateKey, string, error) {
	var regs []app.RegistrationData
	from := validFrom.UTC().Truncate(time.Millisecond).Format(app.TimeLayout)
	reg := func(k *profiles.PrivateKey, kind, subject string, classes ...string) {
		regs = append(regs, app.RegistrationData{KeyRef: k.Ref, SubjectKind: kind, SubjectID: subject, ProfileID: k.Profile,
			Algorithm: k.Algorithm(), PublicKeyB64: k.PublicB64(), Fingerprint: k.Fingerprint(), PayloadClasses: classes,
			SubjectConfirmation: dom.ConfirmGenesis, ValidFrom: from})
	}
	// Движок и шлюзы — том ant.
	ring, err := profiles.LoadDir(cfg.AntKeys)
	if err != nil {
		return nil, nil, "", err
	}
	code := seed.Policy.EnterpriseCode
	for _, x := range []struct{ ref, kind, subject string }{{EngineKeyRef, dom.SubjectEngine, EngineSubject},
		{GatewayKeyRef, dom.SubjectGateway, GatewaySource}, {EnterpriseGatewayKeyRef, dom.SubjectEnterpriseGateway, code}} {
		k, err := ring.Ensure(cfg.AntKeys, x.ref, dom.ProfileGost)
		if err != nil {
			return nil, nil, "", err
		}
		classes := []string{dom.ClassEvent}
		if x.kind == dom.SubjectEnterpriseGateway {
			classes = append(classes, dom.ClassPassportExtract)
		}
		reg(k, x.kind, x.subject, classes...)
	}
	// Хранитель и верификатор — свои тома (раскладка эпика 29).
	kg, kp, err := profiles.EnsureHybrid(filepath.Join(cfg.Keeper, "keys"), "keeper")
	if err != nil {
		return nil, nil, "", err
	}
	reg(kg, dom.SubjectKeeper, "keeper", dom.ClassCheckpoint)
	reg(kp, dom.SubjectKeeper, "keeper", dom.ClassCheckpoint)
	vg, vp, err := profiles.EnsureHybrid(filepath.Join(cfg.Verifier, "keys"), "verifier")
	if err != nil {
		return nil, nil, "", err
	}
	reg(vg, dom.SubjectVerifier, "verifier", dom.ClassVerifierReport)
	reg(vp, dom.SubjectVerifier, "verifier", dom.ClassVerifierReport)
	// Устройства — ключи edge-агентов (источники оборудования справочника).
	for _, e := range seed.Equipment.Equipment {
		if e.SourceID == nil || *e.SourceID == "" {
			continue
		}
		k, err := profiles.EnsureDeviceKey(cfg.Devices, *e.SourceID)
		if err != nil {
			return nil, nil, "", err
		}
		reg(k, dom.SubjectDevice, *e.SourceID, dom.ClassEvent)
	}
	// Корни партнёра федерации (поставщики с кодом партнёра).
	for _, s := range seed.Lots.Suppliers {
		if s.PartnerCode == nil || *s.PartnerCode == "" {
			continue
		}
		pg, pp, err := profiles.EnsureHybrid(cfg.Partner, "partner-"+strings.ToLower(*s.PartnerCode))
		if err != nil {
			return nil, nil, "", err
		}
		reg(pg, dom.SubjectPartnerRoot, *s.PartnerCode, dom.ClassPassportExtract, dom.ClassKeyAct)
		reg(pp, dom.SubjectPartnerRoot, *s.PartnerCode, dom.ClassPassportExtract, dom.ClassKeyAct)
	}
	var quorum []*profiles.PrivateKey
	if cfg.Profile != "prod" {
		// Демо-персоны сценариев (класс scenario) — том demo-signer (AD-26, AD-33).
		pr, err := profiles.LoadDir(cfg.DemoSigner)
		if err != nil {
			return nil, nil, "", err
		}
		for _, p := range seed.Policy.Persons {
			for _, x := range []struct{ ref, profile string }{{app.PersonaKeyRef(p.ID), dom.ProfileGost}, {app.PersonaPQRef(p.ID), dom.ProfilePQ}} {
				k, err := pr.Ensure(cfg.DemoSigner, x.ref, x.profile)
				if err != nil {
					return nil, nil, "", err
				}
				reg(k, dom.SubjectDemoPersona, p.ID, PersonaClasses...)
				if x.profile == dom.ProfileGost {
					quorum = append(quorum, k)
				}
			}
		}
		// Интерактивные демо-персоны — ./.demo-keys/token-agent/ (вне репозитория).
		if dir := cfg.TokenAgent; dir != "" {
			if err := writable(dir); err != nil {
				log.Warn("ключи интерактивных демо-персон не записаны: каталог недоступен (make keys)", "dir", dir, "err", err)
			} else {
				tr, err := profiles.LoadDir(dir)
				if err != nil {
					return nil, nil, "", err
				}
				for _, p := range InteractivePersonas {
					low := strings.ToLower(p)
					for _, x := range []struct{ ref, profile string }{{low + "-ta@1", dom.ProfileGost}, {low + "-ta-pq@1", dom.ProfilePQ}} {
						k, err := tr.Ensure(dir, x.ref, x.profile)
						if err != nil {
							return nil, nil, "", err
						}
						reg(k, dom.SubjectDemoPersona, p, PersonaClasses...)
					}
				}
			}
		}
	}
	return regs, quorum, kg.Fingerprint(), nil
}

// writable — каталог создаётся и в него можно писать.
func writable(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".probe-*")
	if err != nil {
		return err
	}
	name := f.Name()
	_ = f.Close()
	return os.Remove(name)
}

// writeAnchors — trust-anchors в томах хранителя и верификатора (AD-33):
// ключи хранителя, верификатора и якоря, ожидаемые профили, отпечатки якоря
// и генезиса; затем сертификаты mTLS (хранитель — сервер; ant и verifier —
// клиенты).
func writeAnchors(cfg ProvisionConfig, h dom.GenesisHeader, digest string) error {
	if cfg.Keeper == "" {
		return nil
	}
	t := profiles.TrustAnchors{FormatVersion: 1, Profiles: dom.DefaultObjectProfiles(), AnchorFingerprint: h.AnchorFingerprint,
		GenesisDigest: digest, Note: "Демо: создан ant init одного хоста (AD-33); в промышленной эксплуатации файл подписывает " +
			"Аудитор ИБ и передаёт на отчуждаемом носителе."}
	for _, sub := range []struct{ dir, subject string }{{cfg.Keeper, "keeper"}, {cfg.Verifier, "verifier"}} {
		if sub.dir == "" {
			continue
		}
		g, p, err := profiles.EnsureHybrid(filepath.Join(sub.dir, "keys"), sub.subject)
		if err != nil {
			return err
		}
		t.Keys = append(t.Keys, profiles.AnchorOf(sub.subject, g), profiles.AnchorOf(sub.subject, p))
	}
	for _, k := range h.AnchorKeys {
		t.Keys = append(t.Keys, profiles.AnchorFromPublic(dom.AnchorSubject, k.KeyRef, k.ProfileID, dom.AnchorKeyBytes(k)))
	}
	for _, dir := range []string{cfg.Keeper, cfg.Verifier} {
		if dir == "" {
			continue
		}
		if err := profiles.WriteTrustAnchors(filepath.Join(dir, AnchorsFile), t); err != nil {
			return err
		}
	}
	var parts []pki.Participant
	if cfg.AntPKI != "" {
		parts = append(parts, pki.Participant{Name: "ant", Dir: cfg.AntPKI})
	}
	if cfg.Verifier != "" {
		parts = append(parts, pki.Participant{Name: "verifier", Dir: cfg.Verifier})
	}
	hosts := cfg.KeeperHosts
	if len(hosts) == 0 {
		hosts = []string{"keeper", "localhost"}
	}
	return pki.Init(cfg.Keeper, hosts, parts)
}

// Genesis — блок генезиса журнала, проверенный по якорю (ядро при старте).
func Genesis(ctx context.Context, j journal.JournalStore) (app.GenesisReport, error) {
	return app.VerifyGenesis(ctx, j, profiles.Verifier{}, "")
}

// ErrNoGenesis — в журнале нет генезиса (ant init не выполнялся).
var ErrNoGenesis = errors.New("генезиса в журнале нет (ant init)")

// GenesisProcess — байты стартового процесса из генезиса (FR-10, AD-17, AD-33):
// нормативный слой v1 закреплён записью normative.version.loaded блока
// (подписи якоря и кворума); байты берутся из затравки fsys и принимаются,
// только если их отпечаток совпал с закреплённым. Возвращает id версии и XML.
func GenesisProcess(ctx context.Context, j journal.JournalStore, fsys fs.FS) (string, []byte, error) {
	st, err := app.FindGenesis(ctx, j)
	if err != nil {
		return "", nil, err
	}
	if !st.Present {
		return "", nil, ErrNoGenesis
	}
	rep, err := Genesis(ctx, j)
	if err != nil {
		return "", nil, err
	}
	ds, err := app.GenesisData(ctx, j, rep, catalog.NormativeVersionLoaded)
	if err != nil || len(ds) == 0 {
		return "", nil, fmt.Errorf("генезис без normative.version.loaded: %v", err)
	}
	var d struct {
		VersionID  string `json:"version_id"`
		Hash       string `json:"process_version_hash"`
		Components []struct {
			Path   string `json:"path"`
			Digest string `json:"digest"`
		} `json:"components"`
	}
	if err := json.Unmarshal(ds[0], &d); err != nil {
		return "", nil, err
	}
	if fsys == nil {
		fsys = Seed()
	}
	xml, err := fs.ReadFile(fsys, app.SeedProcessFile)
	if err != nil {
		return "", nil, err
	}
	if dom.Digest(xml) != d.Hash {
		return "", nil, fmt.Errorf("стартовый процесс сборки не совпадает с закреплённым генезисом (%s)", d.Hash)
	}
	return d.VersionID, xml, nil
}

// IngestSigning — проверка подписи приёма и подпись шлюза из генезиса
// (FR-26, AD-11): реестр ключей — свёртка key.* журнала (генезис — первые
// записи), подпись служебных записей — ключ gateway-ingest@1 тома ant.
// Ключа в томе нет — signer nil (записи шлюза без подписи, как до генезиса).
func IngestSigning(j journal.JournalStore, antKeys string) (profiles.Verifier, IngestKeys, *profiles.Signer, error) {
	reg := app.NewRegistry(j, nil, nil)
	v := profiles.Verifier{Keys: reg}
	keys := IngestKeys{Registry: reg, Sources: []string{GatewaySource}}
	ring, err := profiles.LoadDir(antKeys)
	if err != nil {
		return v, keys, nil, err
	}
	if _, ok := ring.Key(GatewayKeyRef); !ok {
		return v, keys, nil, nil
	}
	return v, keys, &profiles.Signer{Keys: ring}, nil
}
