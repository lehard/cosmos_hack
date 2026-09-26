package simulation

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	appingest "ant/internal/application/ingest"
	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/constants"
	"ant/internal/contracts/crypto"
	jc "ant/internal/contracts/journal"
	domingest "ant/internal/domain/ingest"
	"ant/internal/domain/kernel"
	sim "ant/internal/domain/simulation"
)

// Адаптеры ведомых портов simulation поверх портов других модулей
// application (сборка — в cmd/ant): приём событий через ведущий порт приёма,
// служебные записи через journal.Append, сбои через служебный порт stand-ов.

// IngestGateway — события источников прогона в обычный приём через ведущий
// порт ingest.Commands (операция ingest.batch.submit) пачками по источнику.
// Конверт DSSE без подписи — профиль demo до эпика 05 (приём ставит
// signature_checked = false, класс происхождения scenario); с edge-агентом,
// принимающим несколько источников, подпись поставит ключ устройства (AD-26).
type IngestGateway struct {
	Ingest appingest.Commands
	// SentAt — сообщать приёму время отправки пачки по часам источника (FR-33):
	// момент доставки по виртуальным часам прогона.
	SentAt bool
}

var _ Gateway = IngestGateway{}

// Deliver — пачки по источнику в порядке доставки.
func (g IngestGateway) Deliver(ctx context.Context, _ string, batch []sim.Emission) ([]Delivered, error) {
	var out []Delivered
	for i := 0; i < len(batch); {
		j := i
		for j < len(batch) && batch[j].SourceID == batch[i].SourceID {
			j++
		}
		part := batch[i:j]
		in := appingest.IngestBatch{SourceID: part[0].SourceID}
		for _, e := range part {
			in.Envelopes = append(in.Envelopes, crypto.DsseEnvelope{PayloadType: appingest.PayloadTypeEvent,
				Payload: base64.StdEncoding.EncodeToString(e.Event), Signatures: []crypto.DsseEnvelopeSignaturesElem{}})
		}
		if g.SentAt {
			t := part[len(part)-1].DeliverAt
			in.SentAt = &t
		}
		res, err := g.Ingest.SubmitBatch(ctx, in)
		if err != nil {
			return out, err
		}
		for k, e := range part {
			d := Delivered{EventID: e.EventID, Status: DeliveryRejected}
			if k < len(res.Items) {
				it := res.Items[k]
				d.Status = DeliveryStatus(it.Status)
				if it.Code != nil {
					d.Code = *it.Code
				}
				if it.Seq != nil {
					d.Seq = *it.Seq
				}
			}
			out = append(out, d)
		}
		i = j
	}
	return out, nil
}

// JournalRecorder — служебные записи прогона (simulation.run.*,
// time.clock.ticked) единственной функцией записи journal.Append (AD-44).
// event_id — UUIDv5 от содержимого записи: повтор после сбоя опознаётся как
// дубль (AD-7). Подпись — ключом сервера сценариев, когда он будет (эпик 05);
// до того конверт DSSE без подписей, как у приёма в профиле demo.
type JournalRecorder struct {
	Store appjournal.JournalStore
	// ScenarioClock — журнал в режиме часов scenario (AD-37): recorded_at
	// записи прогона — виртуальное время.
	ScenarioClock bool
	// Now — InfraClock для received_at (AD-37).
	Now func() time.Time
	mu  sync.Mutex
}

var _ Recorder = (*JournalRecorder)(nil)

// SourceSimulation — source_id служебных записей прогона.
const SourceSimulation = "ant-simulation"

// Record — запись.
func (r *JournalRecorder) Record(ctx context.Context, rec Record) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	data, err := json.Marshal(rec.Data)
	if err != nil {
		return 0, err
	}
	occurred := sim.FormatTime(rec.OccurredAt)
	id := kernel.UUIDv5(constants.NsAnt, fmt.Sprintf("simulation-record|%s|%s|%s|%s", rec.RunID, rec.Type, occurred, data))
	env := map[string]any{"event_id": id, "event_type": rec.Type, "schema_version": 1, "source_id": SourceSimulation,
		"occurred_at": occurred, "correlation_id": id, "causation_id": nil, "run_id": rec.RunID,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{"simulation.ant@1"}},
		"data":      json.RawMessage(data)}
	raw, err := json.Marshal(env)
	if err != nil {
		return 0, err
	}
	canon, err := domingest.Canonicalize(raw)
	if err != nil {
		return 0, err
	}
	dsse, err := json.Marshal(map[string]any{"payloadType": appingest.PayloadTypeEvent,
		"payload": base64.StdEncoding.EncodeToString(canon), "signatures": []any{}})
	if err != nil {
		return 0, err
	}
	now := time.Now
	if r.Now != nil {
		now = r.Now
	}
	run := rec.RunID
	e := jc.JournalEntry{Chain: jc.JournalEntryChainMain, EntryKind: jc.JournalEntryEntryKindService, EventType: rec.Type,
		SchemaVersion: 1, EventID: id, SourceID: SourceSimulation, Stream: "run:" + rec.RunID, OccurredAt: occurred,
		ReceivedAt: sim.FormatTime(now()), CorrelationID: id, ProvenanceClass: jc.JournalEntryProvenanceClassScenario, RunID: &run}
	if r.ScenarioClock {
		e.RecordedAt = occurred
	}
	res, err := r.Store.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{{Entry: e, Envelope: dsse}}})
	if errors.Is(err, appjournal.ErrDuplicate) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if len(res.Seqs) == 0 {
		return 0, nil
	}
	return res.Seqs[0], nil
}

// StandControl — служебный порт stand-ов приёма (application/ingest.StandControl, AD-18).
type StandControl struct {
	Control appingest.StandControl
}

var _ Stands = StandControl{}

// SetFault — включить сбой stand-а.
func (s StandControl) SetFault(ctx context.Context, stand string, a sim.StandAction, until time.Time) error {
	return s.Control.SetFault(ctx, stand, appingest.Fault{Kind: appingest.FaultKind(a.Fault), Param: a.Param, Until: until})
}

// ClearFaults — снять сбои stand-а.
func (s StandControl) ClearFaults(ctx context.Context, stand string) error {
	return s.Control.ClearFaults(ctx, stand)
}

// PendingTamperer — ЗАГЛУШКА порта Tamperer до слияния эпика 29 (cmd/tamper,
// make tamper, отдельное подключение суперпользователя БД — AD-26, AD-28):
// подделку не выполняет и прямо говорит об этом — шаг кнопки «пропущен»,
// строки табло «подделку поймали» ждут. При сведении дирижёр подставляет в
// Deps.Tamper реализацию эпика 29; сервис берёт заглушку, только если порт пуст.
type PendingTamperer struct{}

var _ Tamperer = PendingTamperer{}

// ErrTamperPending — демо-инструмент подделки ещё не подключён.
var ErrTamperPending = errors.New("демо-инструмент cmd/tamper ещё не подключён (эпик 29): подделка не выполнена")

// Apply — подделка не выполняется (заглушка).
func (PendingTamperer) Apply(context.Context, string, sim.Tamper, string) error {
	return ErrTamperPending
}
