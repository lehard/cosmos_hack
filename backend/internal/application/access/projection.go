package access

import (
	"context"
	"slices"
	"sync"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	accessdom "ant/internal/domain/access"
)

// Projection — проекция политики (AD-15, AD-45): затравка (или пустая
// политика, если журнал начинается с политики генезиса) плюс записи
// policy.* и access.* журнала после неё. Копии api догоняют журнал сами:
// сигнал «есть новое» (LISTEN/NOTIFY) — перечитать записи после своего seq,
// поэтому отзыв роли доходит до всех копий (AD-15).
type Projection struct {
	seed accessdom.Policy
	log  PolicyLog
	// Every — наименьший интервал между перечитываниями журнала (InfraClock).
	Every time.Duration
	// Now — инфраструктурные часы (AD-37); nil — системные.
	Now func() time.Time

	mu     sync.Mutex
	cur    accessdom.Policy
	loaded bool
	// base, recs — начало свёртки (затравка или пустая политика генезиса) и
	// применённые записи: политика на любой seq (At) — для права подписанта
	// на момент подписи (FR-85, AD-9) и истории выдачи прав.
	base     accessdom.Policy
	recs     []accessdom.Record
	lastHead int64
	lastRead time.Time
}

// NewProjection — проекция над затравкой seed и журналом log (nil — только затравка).
func NewProjection(seed accessdom.Policy, log PolicyLog) *Projection {
	return &Projection{seed: seed.Clone(), log: log, Every: time.Second}
}

var _ PolicySource = (*Projection)(nil)

// Policy — действующая политика; при ошибке чтения журнала — последняя
// известная (и ошибка, если журнал ещё ни разу не прочитан).
func (p *Projection) Policy(ctx context.Context) (accessdom.Policy, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.log == nil {
		return p.seed, nil
	}
	now := time.Now()
	if p.Now != nil {
		now = p.Now()
	}
	head := p.log.Head()
	due := !p.loaded || (head > p.lastHead && now.Sub(p.lastRead) >= p.Every) || (head == 0 && now.Sub(p.lastRead) >= p.Every)
	if !due {
		return p.cur, nil
	}
	if err := p.refresh(ctx); err != nil {
		if !p.loaded {
			return accessdom.Policy{}, err
		}
		return p.cur, nil
	}
	p.lastHead, p.lastRead = head, now
	return p.cur, nil
}

// Refresh — перечитать журнал сейчас (после своей записи политики — чтобы
// следующий запрос видел её без ожидания интервала).
func (p *Projection) Refresh(ctx context.Context) error {
	if p.log == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.refresh(ctx)
}

func (p *Projection) refresh(ctx context.Context) error {
	base := p.cur
	if !p.loaded {
		base = p.seed
	}
	recs, err := p.log.Since(ctx, base.Seq)
	if err != nil {
		return err
	}
	if len(recs) == 0 && p.loaded {
		return nil
	}
	next := base.Clone()
	if !p.loaded {
		p.base = p.seed
		if genesis(recs) {
			// Генезис (эпик 05) записал роли в журнал — затравка не нужна (AD-33);
			// справочная часть нормативного слоя (сферы, маршрут выдачи) остаётся.
			p.base = accessdom.Policy{Root: p.seed.Root, Unauthenticated: p.seed.Unauthenticated, Catalog: p.seed.Catalog, Audit: p.seed.Catalog.Audit}
		}
		next = p.base.Clone()
	}
	for _, r := range recs {
		if err := next.Apply(r); err != nil {
			return platform.Fail(errcodes.ApiInternalError, "detail", err.Error())
		}
	}
	p.cur, p.loaded = next, true
	p.recs = append(p.recs, recs...)
	return nil
}

// At — политика на позиции журнала seq (AD-9, FR-85: право подписанта на
// момент подписи): начало свёртки плюс записи с seq ≤ seq. seq не меньше
// действующей версии — действующая политика.
func (p *Projection) At(ctx context.Context, seq int64) (accessdom.Policy, error) {
	cur, err := p.Policy(ctx)
	if err != nil {
		return cur, err
	}
	if seq >= cur.Seq || p.log == nil {
		return cur, nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return accessdom.PolicyAt(p.base, p.recs, seq)
}

// Records — записи политики, применённые проекцией, по возрастанию seq
// (история выдачи прав у Аудитора ИБ). Вызывающий их не меняет.
func (p *Projection) Records(ctx context.Context) ([]accessdom.Record, error) {
	if _, err := p.Policy(ctx); err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.recs), nil
}

// genesis — журнал начинается с политики генезиса (записи с происхождением genesis, AD-33).
func genesis(recs []accessdom.Record) bool {
	for _, r := range recs {
		if r.Genesis {
			return true
		}
	}
	return false
}
