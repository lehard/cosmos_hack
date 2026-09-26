package signing

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"go.stargrave.org/gogost/v7/gost34112012256"

	"ant/internal/application/journal"
	app "ant/internal/application/signing"
	"ant/internal/contracts/catalog"
	dom "ant/internal/domain/signing"
	"ant/internal/infrastructure/security/profiles"
)

// Детерминированные ключи демо-персон (Д-82, Д-72): в профилях затравки
// (demo, fixtures) ключи персон для агента токена (‹псевдоним›-ta@1,
// ‹псевдоним›-ta-pq@1) и демо-подписчика сценариев (‹псевдоним›@1,
// ‹псевдоним›-pq@1) выводятся из постоянного демо-зерна репозитория, а не из
// случайного источника: на любой машине и после любого сброса стенда открытые
// ключи одинаковы — файл ключа, однажды загруженный в расширение браузера, не
// нужно перевыпускать и перезагружать.
//
// Зерно публично (лежит в репозитории), поэтому ключи демо-персон — НЕ секрет:
// это осознанное свойство демо-стенда. В профиле prod (и load) демо-персон нет
// или ключи случайные; вывод отказывает для любого профиля, кроме demo и fixtures.

// demoKeySeed — постоянное демо-зерно ключей персон. ТОЛЬКО ДЕМО: не для
// рабочего профиля; смена строки меняет все демо-ключи (их придётся
// перезагрузить в расширение), поэтому версия зерна — в самой строке.
const demoKeySeed = "ant/glavny demo-persona-keys seed v1 — ТОЛЬКО ДЕМО, публично, не для профиля prod"

// demoKeyLabel — метка KDF (Р 50.1.113-2016, KDF_GOSTR3411_2012_256).
var demoKeyLabel = []byte("ant/demo-persona-key")

// ErrNotDemoProfile — детерминированный демо-ключ запрошен вне профилей затравки.
var ErrNotDemoProfile = errors.New("детерминированные демо-ключи — только в профилях demo и fixtures")

// DeterministicDemoKeys — профиль окружения выводит ключи демо-персон из
// демо-зерна (только demo и fixtures; prod и load — нет).
func DeterministicDemoKeys(envProfile string) bool {
	return envProfile == "demo" || envProfile == "fixtures"
}

// DemoPersonaKey — детерминированный ключ демо-персоны personID с key_ref ref
// (номер версии ключа — в ref: …@1) профиля подписи keyProfile (gost | pq):
// материал = KDF_GOSTR3411_2012_256(демо-зерно, метка, person_id ‖ ref ‖
// профиль ‖ счётчик); ГОСТ — скаляр по модулю q, нулевой отбрасывается
// (следующий счётчик); ML-DSA-65 — материал как 32-байтное зерно.
// Вне профилей demo и fixtures — ErrNotDemoProfile.
func DemoPersonaKey(envProfile, personID, ref, keyProfile string) (*profiles.PrivateKey, error) {
	if !DeterministicDemoKeys(envProfile) {
		return nil, fmt.Errorf("%w (профиль %q)", ErrNotDemoProfile, envProfile)
	}
	kdf := gost34112012256.NewKDF([]byte(demoKeySeed))
	for ctr := uint32(0); ctr < 16; ctr++ {
		var in bytes.Buffer
		for _, s := range []string{personID, ref, keyProfile} {
			in.WriteString(s)
			in.WriteByte(0)
		}
		_ = binary.Write(&in, binary.BigEndian, ctr)
		k, err := profiles.FromSeed(ref, keyProfile, kdf.Derive(nil, demoKeyLabel, in.Bytes()))
		if errors.Is(err, profiles.ErrZeroScalar) {
			continue
		}
		return k, err
	}
	return nil, fmt.Errorf("демо-ключ %s: материал KDF подряд даёт нулевой скаляр", ref)
}

// demoKeys — ключи демо-персон одного каталога (том demo-signer или
// ./.demo-keys/token-agent) при генезисе.
type demoKeys struct {
	dir        string
	envProfile string
	ring       *profiles.Keyring
	// kept — существующие файлы, отличные от детерминированных (старые случайные).
	kept []string
}

func loadDemoKeys(dir, envProfile string) (*demoKeys, error) {
	ring, err := profiles.LoadDir(dir)
	if err != nil {
		return nil, err
	}
	return &demoKeys{dir: dir, envProfile: envProfile, ring: ring}, nil
}

// ensure — ключ ref персоны personID. Существующий файл главнее: совпал с
// детерминированным — оставляется; отличается (выпущен случайно до Д-82 и,
// возможно, уже загружен человеком в расширение) — тоже оставляется, без
// перезаписи, и попадает в предупреждение; файла нет — в профилях затравки
// выводится детерминированно, в остальных — случайный, как раньше.
func (d *demoKeys) ensure(personID, ref, keyProfile string) (*profiles.PrivateKey, error) {
	if !DeterministicDemoKeys(d.envProfile) {
		return d.ring.Ensure(d.dir, ref, keyProfile)
	}
	want, err := DemoPersonaKey(d.envProfile, personID, ref, keyProfile)
	if err != nil {
		return nil, err
	}
	if k, ok := d.ring.Key(ref); ok {
		if k.Fingerprint() != want.Fingerprint() {
			d.kept = append(d.kept, ref)
		}
		return k, nil
	}
	if err := profiles.Save(d.dir, want); err != nil {
		return nil, err
	}
	return want, nil
}

// warn — одно предупреждение на каталог о сохранённых недетерминированных ключах.
func (d *demoKeys) warn(log *slog.Logger) {
	if len(d.kept) == 0 {
		return
	}
	log.Warn("ключи демо-персон в каталоге выпущены случайно (до Д-82) и оставлены как есть: уже загруженные в расширение "+
		"продолжают работать, но на другой машине или после очистки каталога ключи будут другими (детерминированными); "+
		"перейти — удалить эти файлы, пересоздать стенд (down -v, make keys) и один раз перезагрузить ключи в расширение",
		"dir", d.dir, "keys", len(d.kept), "refs", strings.Join(d.kept, ","))
}

// restoreDemoKeys — генезис уже есть, а файлов ключей демо-персон в каталоге
// нет (каталог потерян или очищен): в профилях затравки недостающий файл
// выводится заново, если его открытый ключ совпадает с зарегистрированным
// генезисом; иначе (генезис со случайным ключом) — предупреждение: ключ не
// восстановить, нужен новый стенд.
func restoreDemoKeys(ctx context.Context, j journal.JournalStore, rep app.GenesisReport, cfg ProvisionConfig, log *slog.Logger) error {
	if !DeterministicDemoKeys(cfg.Profile) {
		return nil
	}
	type want struct{ dir, person, ref, profile string }
	var ws []want
	if cfg.TokenAgent != "" {
		for _, p := range InteractivePersonas {
			for _, x := range interactiveRefs(p) {
				ws = append(ws, want{cfg.TokenAgent, p, x.ref, x.profile})
			}
		}
	}
	var missing []want
	for _, w := range ws {
		if _, err := os.Stat(filepath.Join(w.dir, profiles.FileName(w.ref))); errors.Is(err, os.ErrNotExist) {
			missing = append(missing, w)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	if err := writable(cfg.TokenAgent); err != nil {
		log.Warn("ключи интерактивных демо-персон не восстановлены: каталог недоступен", "dir", cfg.TokenAgent, "err", err)
		return nil
	}
	ds, err := app.GenesisData(ctx, j, rep, catalog.KeyRegistrationRecorded)
	if err != nil {
		return err
	}
	registered := map[string]string{}
	for _, raw := range ds {
		if g, err := app.RegistrationFromData(raw); err == nil {
			registered[g.KeyRef] = dom.Digest(g.PublicKey())
		}
	}
	var restored, lost []string
	for _, w := range missing {
		k, err := DemoPersonaKey(cfg.Profile, w.person, w.ref, w.profile)
		if err != nil {
			return err
		}
		if registered[w.ref] != k.Fingerprint() {
			lost = append(lost, w.ref)
			continue
		}
		if err := profiles.Save(w.dir, k); err != nil {
			return err
		}
		restored = append(restored, w.ref)
	}
	if len(restored) > 0 {
		log.Info("ключи демо-персон восстановлены из демо-зерна (Д-82)", "dir", cfg.TokenAgent, "keys", len(restored))
	}
	if len(lost) > 0 {
		log.Warn("ключей демо-персон нет в каталоге, а генезис регистрировал случайные ключи — восстановить нельзя; "+
			"пересоздайте стенд (down -v, make keys)", "dir", cfg.TokenAgent, "refs", strings.Join(lost, ","))
	}
	return nil
}

// interactiveRefs — ключи персоны для агента токена: ‹псевдоним›-ta@1 (gost)
// и ‹псевдоним›-ta-pq@1 (pq).
func interactiveRefs(persona string) []struct{ ref, profile string } {
	low := strings.ToLower(persona)
	return []struct{ ref, profile string }{{low + "-ta@1", dom.ProfileGost}, {low + "-ta-pq@1", dom.ProfilePQ}}
}
