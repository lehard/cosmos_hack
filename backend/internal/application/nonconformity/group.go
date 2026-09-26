package nonconformity

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
	"uuid"

	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/application/security"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/nonconformity"
	domsecurity "ant/internal/domain/security"
)

// Групповое несоответствие (FR-151, S05 «НС-И1», NC-G1): несоответствие окна
// нарушения специального процесса — одно на окно, зарегистрировано в потоке
// каждого изделия окна. Решение комиссии по нему одно и приходит каждому
// изделию группы: одна команда — по записи-решению в поток каждого изделия,
// одной транзакцией journal.Append (все или ни одной, AD-39, AD-44).

// ncGroup — изделия несоответствия ncID на момент m: изделия записей
// регистрации с этим nc_id (групповое), иначе — единственное изделие
// несоответствия.
func (s *Service) ncGroup(ctx context.Context, ncID string, m platform.Moment) ([]string, error) {
	es, err := s.readAll(ctx, appjournal.ReadQuery{EventType: string(catalog.DecisionNonconformityRegistered), Moment: m})
	if err != nil {
		return nil, err
	}
	var items []string
	for _, e := range es {
		if e.ItemID == nil || *e.ItemID == "" || slices.Contains(items, *e.ItemID) {
			continue
		}
		d, err := s.d.Codec.Decode(ctx, e)
		if err != nil {
			continue
		}
		var x struct {
			NCID string `json:"nc_id"`
		}
		if json.Unmarshal(d.Record.Data, &x) == nil && x.NCID == ncID {
			items = append(items, *e.ItemID)
		}
	}
	if len(items) > 0 {
		slices.Sort(items)
		return items, nil
	}
	it, err := s.ncItem(ctx, ncID)
	if err != nil {
		return nil, err
	}
	return []string{it}, nil
}

// groupEventID — event_id записи групповой команды в потоке изделия:
// UUIDv5 команды и изделия (повтор команды даёт те же id, AD-7).
func groupEventID(commandID, itemID string) string {
	return kernel.UUIDv5(constants.NsAnt, "nonconformity.group\x1f"+commandID+"\x1f"+itemID)
}

// group — команда по несоответствию ncID: у несоответствия одного изделия —
// как раньше (s.nc); у группового — гард и запись в каждом изделии группы,
// где несоответствие уже есть и pick (nil — все) его выбирает. Изделие, по
// которому команда уже исполнена (гард: недопустимый переход), пропускается;
// любой другой отказ гарда — отказ всей команды.
func (s *Service) group(ctx context.Context, ncID string, c itemCommand, pick func(v *itemView) bool) (platform.Receipt, error) {
	items, err := s.ncGroup(ctx, ncID, platform.Moment{})
	if err != nil {
		return platform.Receipt{}, err
	}
	if len(items) == 1 {
		return s.nc(ctx, ncID, c)
	}
	cmdID := strings.ToLower(c.Meta.CommandID)
	if _, err := uuid.Parse(cmdID); err != nil || cmdID == "" {
		cmdID = uuid.NewV7().String()
	}
	for _, it := range items {
		if r, ok, err := s.replayed(ctx, "item:"+it, c.Type, groupEventID(cmdID, it)); err != nil || ok {
			if ok {
				return s.groupReceipt(ctx, c.Type, cmdID, items)
			}
			return r, err
		}
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	var (
		batch    []appjournal.Pending
		checks   []appjournal.Check
		ids      []string
		firstErr error
		at       time.Time
	)
	for _, it := range items {
		v, err := s.loadItem(ctx, it, platform.Moment{})
		if err != nil {
			return platform.Receipt{}, err
		}
		if _, ok := v.State().NC(ncID); !ok || (pick != nil && !pick(v)) {
			continue
		}
		now, err := s.now(ctx, v.RunID)
		if err != nil {
			return platform.Receipt{}, err
		}
		stream := "item:" + it
		// basis_seq клиента относится к изделию, по которому он смотрел
		// карточку; у остальных изделий группы — их состояние на сейчас.
		basis := v.BasisSeq
		cmd := kernel.Command{Action: c.Action, CommandID: cmdID, Actor: actor, Object: stream, BasisSeq: basis,
			PolicySeq: c.Meta.PolicySeq, GuardStreams: []string{stream}, OccurredAt: now, SignatureLevel: 2, Payload: c.Data}
		if err := dom.Guard(v.State(), v.Env, v.Upstream(), cmd); err != nil {
			var rf *kernel.Refusal
			if errors.As(err, &rf) && rf.Code == errcodes.NonconformityInvalidTransition {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			return platform.Receipt{}, err
		}
		more := []appjournal.Check{{Stream: stream, BasisSeq: basis}}
		if c.Extra != nil {
			x, err := c.Extra(v, now)
			if err != nil {
				return platform.Receipt{}, err
			}
			more = append(more, x...)
		}
		meta := c.Meta
		meta.CommandID, meta.BasisSeq = cmdID, basis
		id := groupEventID(cmdID, it)
		p, err := s.pending(ctx, decision{Type: c.Type, Stream: stream, ItemID: it, RunID: v.RunID, Data: c.Data, Meta: meta,
			Actor: actor, OccurredAt: now, SignatureLevel: 2, GuardStreams: []string{stream}}, id, "")
		if err != nil {
			return platform.Receipt{}, err
		}
		batch, checks, ids = append(batch, p), append(checks, more...), append(ids, id)
		if at.IsZero() {
			at = now
		}
	}
	if len(batch) == 0 {
		if firstErr != nil {
			return platform.Receipt{}, firstErr
		}
		e := platform.Fail(errcodes.ApiNotFound, "object", "Несоответствие", "id", ncID)
		e.Detail = "Несоответствие " + ncID + ": нет изделий, к которым относится команда"
		return platform.Receipt{}, e
	}
	var res appjournal.AppendResult
	err = security.Execute(ctx, domsecurity.Command{ActorID: actor, PolicySeq: c.Meta.PolicySeq}, func(ctx context.Context) error {
		var err error
		res, err = s.d.Journal.Append(ctx, appjournal.AppendRequest{Batch: batch, Checks: checks})
		return err
	})
	if errors.Is(err, appjournal.ErrDuplicate) {
		return s.groupReceipt(ctx, c.Type, cmdID, items)
	}
	if err != nil {
		if pe, ok := appjournal.Problem(err); ok {
			return platform.Receipt{}, pe
		}
		return platform.Receipt{}, err
	}
	r := platform.Receipt{CommandID: cmdID, EventIDs: ids, RecordedAt: at}
	if len(res.Seqs) > 0 {
		r.Seq = res.Seqs[0]
	}
	if !s.cfg.ScenarioClock && !res.Committed.IsZero() {
		r.RecordedAt = res.Committed.UTC()
	}
	return r, nil
}

// groupReceipt — прежняя квитанция групповой команды (AD-7): записи этой
// команды в потоках изделий группы.
func (s *Service) groupReceipt(ctx context.Context, t catalog.Type, cmdID string, items []string) (platform.Receipt, error) {
	r := platform.Receipt{CommandID: cmdID, Replayed: true}
	for _, it := range items {
		x, err := s.replay(ctx, decision{Stream: "item:" + it, Type: t}, groupEventID(cmdID, it))
		if err != nil {
			return platform.Receipt{}, err
		}
		if x.Seq == 0 {
			continue
		}
		r.EventIDs = append(r.EventIDs, x.EventIDs...)
		if r.Seq == 0 {
			r.Seq, r.RecordedAt = x.Seq, x.RecordedAt
		}
	}
	return r, nil
}
