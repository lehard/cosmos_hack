package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"uuid"

	"ant/internal/application/ingest"
	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
)

// Шина безопасности поверх журнала (AD-24, FR-118): события безопасности —
// записи журнала семейства security; источники пишут их через этот модуль
// (IngestBus, Emitter), подписчики — глобальные потребители журнала (AD-45)
// со своим курсором. Новый подписчик подключается строкой в сборке (cmd/ant)
// без изменения источников: он получает все события с начала журнала.
// Журнал критических действий шиной не питается (AD-8).

// IngestBus — реализация порта application/ingest.SecurityBus (эпик 06):
// записи security.signature.invalid и security.idempotency.conflict в ту же
// пачку Append, что и факт карантина. Запись журнала критических действий
// для конфликта (критический тип, группа admin_security) строит CriticalHook
// в транзакции Append — с настоящими номером CA и commit.
type IngestBus struct {
	Enc Encoder
}

var _ ingest.SecurityBus = IngestBus{}

var (
	reSource = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/@-]{0,127}$`)
	reKeyRef = regexp.MustCompile(`^[a-z0-9][a-z0-9._:-]{0,95}@[1-9][0-9]{0,5}$`)
	reUUID   = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

// SignatureInvalid — AD-2: ошибка подписи дополнительно даёт событие security.
func (b IngestBus) SignatureInvalid(ctx context.Context, f ingest.SignatureFailure) ([]appjournal.Pending, []appjournal.Pending, error) {
	data := map[string]any{"failure": f.Failure}
	if reSource.MatchString(f.SourceID) {
		data["source_id"] = f.SourceID
	}
	if reKeyRef.MatchString(f.KeyRef) {
		data["key_ref"] = f.KeyRef
	}
	if reUUID.MatchString(f.EventID) {
		data["subject_event_id"] = f.EventID
	}
	if reUUID.MatchString(f.QuarantineID) {
		data["quarantine_event_id"] = f.QuarantineID
	}
	p, err := b.Enc.Encode(ctx, Out{EventID: uuid.NewV7().String(), Type: catalog.SecuritySignatureInvalid, Stream: "global",
		OccurredAt: f.OccurredAt, Causation: f.QuarantineID, Data: data})
	if err != nil {
		return nil, nil, err
	}
	return []appjournal.Pending{p}, nil, nil
}

// IdempotencyConflict — FR-31, AD-7: конфликт целостности — событие security
// в потоке источника; запись CA — CriticalHook в той же транзакции.
func (b IngestBus) IdempotencyConflict(ctx context.Context, c ingest.IdempotencyConflict) ([]appjournal.Pending, []appjournal.Pending, error) {
	p, err := b.Enc.Encode(ctx, Out{EventID: uuid.NewV7().String(), Type: catalog.SecurityIdempotencyConflict, Stream: "source:" + c.SourceID,
		OccurredAt: c.OccurredAt, Causation: c.QuarantineID, Data: map[string]any{
			"source_id": c.SourceID, "event_id": c.EventID, "first_payload_digest": c.FirstDigest, "conflicting_payload_digest": c.ConflictDigest,
		}})
	if err != nil {
		return nil, nil, err
	}
	return []appjournal.Pending{p}, nil, nil
}

// Emitter — запись событий безопасности модуля security (тревоги хранителя,
// отчёт верификатора, нарушения целостности) через journal.Append.
type Emitter struct {
	Journal appjournal.JournalStore
	Enc     Encoder
}

// Emit пишет события одной пачкой.
func (e Emitter) Emit(ctx context.Context, outs ...Out) (appjournal.AppendResult, error) {
	rq := appjournal.AppendRequest{}
	for _, o := range outs {
		if o.EventID == "" {
			o.EventID = uuid.NewV7().String()
		}
		if o.Stream == "" {
			o.Stream = "global"
		}
		p, err := e.Enc.Encode(ctx, o)
		if err != nil {
			return appjournal.AppendResult{}, err
		}
		rq.Batch = append(rq.Batch, p)
	}
	return e.Journal.Append(ctx, rq)
}

// Subscriber — подписчик шины безопасности (AD-24): получает события
// семейства security по порядку seq; у каждого свой курсор потребителя
// `security.bus.‹имя›`. Ошибка — повтор той же пачки (доставка «хотя бы раз»).
type Subscriber interface {
	Name() string
	Deliver(ctx context.Context, events []SecurityEvent) error
}

// Bus — шина безопасности: подписчики поверх потребителя журнала (AD-45).
type Bus struct {
	Consumer    appjournal.Consumer
	Journal     appjournal.JournalStore
	Subscribers []Subscriber
	Log         *slog.Logger
}

// ConsumerName — имя курсора подписчика.
func ConsumerName(sub string) string { return "security.bus." + sub }

// Run запускает подписчиков; возвращается при отмене ctx или ошибке.
func (b *Bus) Run(ctx context.Context) error {
	log := b.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	var wg sync.WaitGroup
	errs := make(chan error, len(b.Subscribers))
	for _, s := range b.Subscribers {
		wg.Go(func() {
			err := b.Consumer.Consume(ctx, ConsumerName(s.Name()), appjournal.Scope{Global: true},
				func(ctx context.Context, batch []jc.JournalEntry) (appjournal.AppendRequest, error) {
					evs, err := b.Events(ctx, batch)
					if err != nil {
						return appjournal.AppendRequest{}, err
					}
					if len(evs) > 0 {
						if err := s.Deliver(ctx, evs); err != nil {
							return appjournal.AppendRequest{}, fmt.Errorf("подписчик %s: %w", s.Name(), err)
						}
					}
					return appjournal.AppendRequest{}, nil
				})
			if err != nil && ctx.Err() == nil {
				log.Error("шина безопасности: подписчик остановлен", "subscriber", s.Name(), "err", err)
				errs <- err
			}
		})
	}
	wg.Wait()
	close(errs)
	return errors.Join(func() []error {
		var out []error
		for e := range errs {
			out = append(out, e)
		}
		return out
	}()...)
}

// Events — события безопасности из пачки записей журнала (семейство security,
// кроме журнала критических действий).
func (b *Bus) Events(ctx context.Context, batch []jc.JournalEntry) ([]SecurityEvent, error) {
	var out []SecurityEvent
	for _, e := range batch {
		if !IsBusEvent(e) {
			continue
		}
		ev, err := open(ctx, b.Journal, e)
		if err != nil {
			return nil, err
		}
		out = append(out, ToSecurityEvent(e, ev))
	}
	return out, nil
}

// IsBusEvent — запись — событие шины безопасности.
func IsBusEvent(e jc.JournalEntry) bool {
	return (e.Chain == "" || e.Chain == jc.JournalEntryChainMain) && strings.HasPrefix(e.EventType, "security.") &&
		e.EventType != string(catalog.SecurityCriticalActionRecorded)
}

// FileExport — подписчик «экспорт во внешний мониторинг ИБ» (AD-24; в MVP —
// JSON-строки в файл): одна строка на событие, дописывание. Внешняя SIEM
// читает файл (или syslog-пересылка файла средствами ОС).
type FileExport struct {
	Path string
	mu   sync.Mutex
}

// Name — имя подписчика.
func (*FileExport) Name() string { return "export_file" }

// ExportLine — строка экспорта.
type ExportLine struct {
	Seq        int64           `json:"seq"`
	EventID    string          `json:"event_id"`
	EventType  string          `json:"event_type"`
	OccurredAt string          `json:"occurred_at"`
	Severity   string          `json:"severity"`
	SourceID   string          `json:"source_id,omitempty"`
	Summary    string          `json:"summary"`
	Data       json.RawMessage `json:"data,omitempty"`
	Exported   string          `json:"exported_at"`
}

// Deliver дописывает события в файл.
func (f *FileExport) Deliver(_ context.Context, evs []SecurityEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	fh, err := os.OpenFile(f.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	enc := json.NewEncoder(fh)
	for _, e := range evs {
		if err := enc.Encode(ExportLine{Seq: e.Seq, EventID: e.EventID, EventType: e.EventType,
			OccurredAt: e.OccurredAt.UTC().Format("2006-01-02T15:04:05.000Z"), Severity: e.Severity, SourceID: e.SourceID,
			Summary: e.Summary, Data: e.Data, Exported: now}); err != nil {
			_ = fh.Close()
			return err
		}
	}
	return fh.Close()
}

// LogSubscriber — подписчик «журнал процесса» (slog): тревоги — уровнем Warn.
type LogSubscriber struct{ Log *slog.Logger }

// Name — имя подписчика.
func (LogSubscriber) Name() string { return "log" }

// Deliver пишет события в журнал процесса.
func (l LogSubscriber) Deliver(_ context.Context, evs []SecurityEvent) error {
	for _, e := range evs {
		lvl := slog.LevelInfo
		if e.Severity == "alarm" {
			lvl = slog.LevelWarn
		}
		l.Log.Log(context.Background(), lvl, "событие безопасности", "event_type", e.EventType, "seq", e.Seq, "summary", e.Summary)
	}
	return nil
}
