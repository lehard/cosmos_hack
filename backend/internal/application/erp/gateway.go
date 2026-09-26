package erp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/contracts/catalog"
	"ant/internal/domain/engine"
)

// Шлюз входящих порта учёта (AD-18: входящие — через обычный приём):
// задания, номенклатура, партии и соответствия внешних ID читаются у учётной
// системы и уходят в приём как факты источника-шлюза (класс server-attested,
// AD-2). event_id — UUIDv5 от внешнего ключа и версии данных: повторное
// чтение не даёт новых записей; source_seq ведётся на источник и не
// меняется при повторной подаче того же факта.

// GatewaySource — source_id шлюза учётной системы.
func GatewaySource(system string) string { return "erp." + system }

// GatewayKey — подписант пакетов шлюза (демо без подписи — Д-28; ключ
// шлюза — эпик 05).
func GatewayKey(system string) string { return "gateway-" + system + "@1" }

// GatewaySeen — какие факты шлюз уже подал и с каким source_seq.
type GatewaySeen interface {
	Seen(ctx context.Context, source string, ids []string) (map[string]int64, error)
	MarkSeen(ctx context.Context, source string, seqs map[string]int64) error
}

func (o *Outbox) pullLoop(ctx context.Context) {
	t := time.NewTicker(o.PullEvery)
	defer t.Stop()
	for {
		if ch, ok, err := o.Store.Channel(ctx, o.system()); err == nil && ok && ch.State == ChannelOK {
			if n, err := o.PullOnce(ctx); err != nil && ctx.Err() == nil {
				o.Log.Warn("входящие: опрос учётной системы", "err", err)
			} else if n > 0 {
				o.Log.Info("входящие: приняты факты учётной системы", "count", n)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// PullOnce — один опрос входящих: новые факты — в приём; возвращает число поданных.
func (o *Outbox) PullOnce(ctx context.Context) (int, error) {
	o.init()
	facts, err := o.Ledger.Pull(ctx)
	if err != nil || len(facts) == 0 {
		return 0, err
	}
	src := GatewaySource(o.system())
	seen, _ := o.Store.(GatewaySeen)
	ids := make([]string, 0, len(facts))
	for _, f := range facts {
		ids = append(ids, f.EventID)
	}
	known := map[string]int64{}
	if seen != nil {
		if known, err = seen.Seen(ctx, src, ids); err != nil {
			return 0, err
		}
	}
	var fresh []Inbound
	for _, f := range facts {
		if _, ok := known[f.EventID]; !ok {
			fresh = append(fresh, f)
		}
	}
	if len(fresh) == 0 {
		return 0, nil
	}
	first, err := o.Store.NextSourceSeq(ctx, src, len(fresh))
	if err != nil {
		return 0, err
	}
	envs := make([][]byte, 0, len(fresh))
	marks := map[string]int64{}
	for i, f := range fresh {
		seq := first + int64(i)
		b, err := Envelope(src, GatewayKey(o.system()), seq, f)
		if err != nil {
			return 0, err
		}
		envs = append(envs, b)
		marks[f.EventID] = seq
	}
	if _, err := o.Intake.Submit(ctx, src, envs); err != nil {
		return 0, err
	}
	if seen != nil {
		if err := seen.MarkSeen(ctx, src, marks); err != nil {
			return 0, err
		}
	}
	return len(fresh), nil
}

// Envelope — конверт DSSE факта шлюза (контракт events/common/envelope.v1):
// источник — внешняя система, демо без подписи (пустой список подписей, Д-28).
func Envelope(source, keyRef string, seq int64, f Inbound) ([]byte, error) {
	info, _ := catalog.Lookup(f.Type)
	data, err := json.Marshal(f.Data)
	if err != nil {
		return nil, err
	}
	env := map[string]any{
		"event_id": f.EventID, "event_type": string(f.Type), "schema_version": info.CurrentVersion,
		"source_id": source, "source_seq": seq, "source_kind": "external_system", "reliability": "high",
		"occurred_at": engineapp.FormatTime(f.OccurredAt), "correlation_id": f.EventID, "causation_id": nil,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{strings.ToLower(keyRef)}},
		"data":      json.RawMessage(data),
	}
	payload, err := engine.Canonical(env)
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{"payloadType": engineapp.PayloadTypeEvent,
		"payload": base64.StdEncoding.EncodeToString(payload), "signatures": []any{}})
}
