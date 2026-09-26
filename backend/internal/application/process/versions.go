package process

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	dp "ant/internal/domain/process"
)

// Версии процесса (FR-22, FR-23, AD-17): хранилище подписываемых байтов XML
// и утверждение кворумом. XML хранится как загружен; хеш версии записан
// отдельно (при загрузке) — изменение байтов в обход системы обнаруживается
// сравнением H(байты) с хешем, закреплённым за изделием в журнале.

// Стартовая версия (seed, FR-10): загружается из normative/process при
// первом запуске и считается подписанной генезисом (AD-33; демо-трек —
// настоящие подписи генезиса и кворума — эпики 05, 27).
const (
	SeedVersionID    = "flange-1"
	SeedVersionLabel = "v1"
	SeedAuthor       = "TEC-01"
)

// VersionRecord — версия процесса в хранилище модуля.
type VersionRecord struct {
	ID            string
	Label         string
	Status        string
	BaseVersionID string
	Author        string
	// Hash — хеш XML при загрузке (H(байты как загружены)); по нему изделие
	// находит закреплённую версию.
	Hash string
	// XML — байты версии как хранятся сейчас (могли быть изменены в обход системы).
	XML           []byte
	CreatedAt     time.Time
	EffectiveFrom *time.Time
	// Genesis — стартовая версия, подписанная генезисом.
	Genesis bool
	// Signatures, RouteClosedEventID — подписи кворума листа утверждения (AD-43).
	Signatures         []dp.Signature
	RouteClosedEventID string
	// ApprovalDocumentID — лист утверждения версии (документ с маршрутом
	// кворума, эпик 28), заводится при отправке на утверждение (FR-23).
	ApprovalDocumentID string
}

// VersionStore — ведомый порт хранилища версий (адаптер Postgres —
// infrastructure/storage/process; в памяти — MemVersions).
type VersionStore interface {
	// ByHash — версия по хешу, закреплённому за изделием.
	ByHash(ctx context.Context, hash string) (VersionRecord, bool, error)
	// ByID — версия по id.
	ByID(ctx context.Context, id string) (VersionRecord, bool, error)
	// List — все версии по времени создания.
	List(ctx context.Context) ([]VersionRecord, error)
	// Save — сохранить новую версию (черновик или стартовую); существующую
	// версию с тем же id не перезаписывает (версии неизменны).
	Save(ctx context.Context, v VersionRecord) error
	// Update — жизненный цикл версии (FR-22): статус, дата вступления в силу,
	// подписи кворума, закрытие маршрута, лист утверждения. Байты XML и хеш
	// не меняются никогда.
	Update(ctx context.Context, v VersionRecord) error
}

// QuorumVerifier — ведомый порт проверки подписей кворума версии (FR-23,
// AD-43): эпики 05 и 27 проверяют ключи, отзыв и полномочия подписантов на
// seq подписи. До них — RecordedQuorum.
type QuorumVerifier interface {
	Approval(ctx context.Context, v VersionRecord) (dp.Approval, error)
}

// RecordedQuorum — демо-реализация QuorumVerifier: стартовая версия подписана
// генезисом; у остальных — подписи, записанные при утверждении (без проверки
// ключей — эпики 05, 27).
type RecordedQuorum struct{}

// Approval — утверждение версии по записи хранилища.
func (RecordedQuorum) Approval(_ context.Context, v VersionRecord) (dp.Approval, error) {
	return dp.Approval{VersionID: v.ID, Hash: v.Hash, Genesis: v.Genesis, Signatures: v.Signatures, RouteClosedEventID: v.RouteClosedEventID}, nil
}

// SeedRecord — стартовая версия из байтов normative/process (FR-10).
func SeedRecord(xml []byte, at time.Time) VersionRecord {
	t := at.UTC()
	return VersionRecord{ID: SeedVersionID, Label: SeedVersionLabel, Status: dp.StatusActive, Author: SeedAuthor,
		Hash: dp.VersionHash(xml), XML: slices.Clone(xml), CreatedAt: t, EffectiveFrom: &t, Genesis: true}
}

// EnsureSeed — загрузить стартовую версию при первом запуске (FR-10: seed на
// чистой базе без ручных шагов). Уже загруженная не перезаписывается.
func EnsureSeed(ctx context.Context, store VersionStore, xml []byte, at time.Time) (VersionRecord, error) {
	if v, ok, err := store.ByID(ctx, SeedVersionID); err != nil || ok {
		return v, err
	}
	v := SeedRecord(xml, at)
	return v, store.Save(ctx, v)
}

// Active — действующая версия: последняя введённая в действие.
func Active(vs []VersionRecord) (VersionRecord, bool) {
	var best VersionRecord
	found := false
	for _, v := range vs {
		if v.Status != dp.StatusActive || v.EffectiveFrom == nil {
			continue
		}
		if !found || v.EffectiveFrom.After(*best.EffectiveFrom) {
			best, found = v, true
		}
	}
	return best, found
}

// MemVersions — хранилище версий в памяти (тесты, режим без БД).
type MemVersions struct {
	mu sync.Mutex
	vs []VersionRecord
}

var _ VersionStore = (*MemVersions)(nil)

// ByHash — версия по хешу.
func (m *MemVersions) ByHash(_ context.Context, hash string) (VersionRecord, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, v := range m.vs {
		if v.Hash == hash {
			return v, true, nil
		}
	}
	return VersionRecord{}, false, nil
}

// ByID — версия по id.
func (m *MemVersions) ByID(_ context.Context, id string) (VersionRecord, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, v := range m.vs {
		if v.ID == id {
			return v, true, nil
		}
	}
	return VersionRecord{}, false, nil
}

// List — все версии.
func (m *MemVersions) List(context.Context) ([]VersionRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.vs), nil
}

// Save — новая версия (существующая не перезаписывается).
func (m *MemVersions) Save(_ context.Context, v VersionRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, x := range m.vs {
		if x.ID == v.ID {
			return nil
		}
	}
	m.vs = append(m.vs, v)
	return nil
}

// Update — жизненный цикл версии; XML и хеш не трогаются.
func (m *MemVersions) Update(_ context.Context, v VersionRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.vs {
		if m.vs[i].ID == v.ID {
			x := &m.vs[i]
			x.Status, x.EffectiveFrom, x.Signatures, x.RouteClosedEventID, x.ApprovalDocumentID = v.Status, v.EffectiveFrom, slices.Clone(v.Signatures), v.RouteClosedEventID, v.ApprovalDocumentID
			return nil
		}
	}
	return errors.New("process: нет версии " + v.ID)
}

// Tamper — изменить байты XML версии в обход системы (демо-инструмент
// подделки FR-152 и тесты FR-23): хеш записи не меняется.
func (m *MemVersions) Tamper(id string, xml []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.vs {
		if m.vs[i].ID == id {
			m.vs[i].XML = slices.Clone(xml)
		}
	}
}
