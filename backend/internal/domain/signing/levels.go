package signing

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// Уровни подписи (FR-66, AD-13, PRD §11.16). Уровень определяет, КАК
// собирается одна подпись; кто, сколько и в каком порядке подписывает —
// маршрут шаблона документа (AD-43).
const (
	// Level0 — устройство подписывает своё сообщение (edge-агент).
	Level0 = 0
	// Level1 — рутинное действие человека личным ключом без окна (PIN раз за
	// смену): только факты исполнителя и физические подтверждения мастера.
	Level1 = 1
	// Level2 — доверенное отображение (сводка 3–7 полей) и явное подтверждение;
	// все заключения ОТК и все критические действия; допускается пачка.
	Level2 = 2
	// Level3 — сменный рапорт одной подписью над корнем Меркла подписей смены.
	Level3 = 3
)

// level1Actions — перечень действий уровня 1, вкомпилированный в агент токена
// (contracts/internal/token-agent/level1-actions.yaml; тест сверяет копию с
// контрактом). Всё остальное на уровне 1 агент и сервер отклоняют:
// взломанный сервер не должен получать тихие подписи «годен» (PRD §11.16).
var level1Actions = []string{
	"genealogy.lot.issued",
	"item.assembly.recorded",
	"item.carrier.applied",
	"item.carrier.removed",
	"item.presentation.recorded",
	"operation.movement.received",
	"operation.movement.sent",
	"operation.run.finished",
	"operation.run.paused",
	"operation.run.resumed",
	"operation.run.started",
	"operator.deviation.reported",
	"operator.inspection.requested",
	"operator.step.confirmed",
}

// Level1Actions — перечень действий уровня 1 (копия).
func Level1Actions() []string { return slices.Clone(level1Actions) }

// Level1Allowed — тип события допускает подпись уровня 1.
func Level1Allowed(eventType string) bool {
	_, ok := slices.BinarySearch(level1Actions, eventType)
	return ok
}

// ErrLevel — уровень подписи не допускается (signing.level_not_allowed).
var ErrLevel = errors.New("signing.level_not_allowed")

// MinLevel — минимальный уровень подписи для записи: устройство — 0; факты
// из перечня уровня 1 — 1; заключения ОТК, решения и критические действия —
// 2; сменный рапорт — 3. required — уровень операции из каталога
// (x-ant-action), он может только повысить минимум.
func MinLevel(eventType string, required int, critical bool) int {
	m := Level2
	if Level1Allowed(eventType) {
		m = Level1
	}
	if critical && m < Level2 {
		m = Level2
	}
	if required > m {
		m = required
	}
	return m
}

// CheckLevel — AD-13: подпись уровня actual допустима для записи eventType.
// Уровень 1 — только для типов перечня; ниже минимума — отказ. Уровень 0
// (устройство) людям не допускается, уровень 3 — только сменный рапорт.
func CheckLevel(eventType string, required, actual int, critical bool) error {
	switch {
	case actual < Level0 || actual > Level3:
		return fmt.Errorf("%w: уровень %d вне 0–3", ErrLevel, actual)
	case actual == Level3:
		return fmt.Errorf("%w: уровень 3 — только сменный рапорт", ErrLevel)
	case actual == Level1 && !Level1Allowed(eventType):
		return fmt.Errorf("%w: %s не входит в перечень уровня 1 — нужна подпись уровня 2", ErrLevel, eventType)
	}
	if need := MinLevel(eventType, required, critical); actual < need {
		return fmt.Errorf("%w: %s требует уровня %d, подписано уровнем %d", ErrLevel, eventType, need, actual)
	}
	return nil
}

// BatchItem — одна подпись пачки уровня 2: событие-команда в каноническом виде.
type BatchItem struct {
	Payload []byte
}

// BatchSummary — сводка окна пачки: «подписать: N изделий, операция X, годен»
// (AD-13, PRD §11.16) — одно окно агента, одно касание токена.
type BatchSummary struct {
	Count     int
	EventType string
	// Decision — общее содержимое решения без изделия (одно на всю пачку).
	Decision string
	// Items — изделия пачки по порядку.
	Items []string
}

// ErrBatch — пачка неоднородна.
var ErrBatch = errors.New("signing: пачка уровня 2 неоднородна")

// Summarize — сводка пачки: все элементы — события одного типа уровня 2 с
// одинаковым решением (data без полей изделия), разные изделия. Иначе пачка
// отклоняется: подписант видел одно окно и должен подписать ровно то, что в
// нём сказано.
func Summarize(items []BatchItem) (BatchSummary, error) {
	var s BatchSummary
	if len(items) == 0 {
		return s, fmt.Errorf("%w: пачка пуста", ErrBatch)
	}
	var common []byte
	for i, it := range items {
		var ev struct {
			EventType string `json:"event_type"`
			ItemID    string `json:"item_id"`
			Command   struct {
				SignatureLevel int `json:"signature_level"`
			} `json:"command"`
			Data map[string]json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(it.Payload, &ev); err != nil {
			return s, fmt.Errorf("%w: элемент %d: %v", ErrBatch, i, err)
		}
		if ev.ItemID == "" {
			return s, fmt.Errorf("%w: элемент %d без изделия", ErrBatch, i)
		}
		if ev.Command.SignatureLevel != Level2 {
			return s, fmt.Errorf("%w: пачка — только уровень 2, у изделия %s уровень %d", ErrBatch, ev.ItemID, ev.Command.SignatureLevel)
		}
		if slices.Contains(s.Items, ev.ItemID) {
			return s, fmt.Errorf("%w: изделие %s дважды", ErrBatch, ev.ItemID)
		}
		for _, k := range []string{"item_id", "operation_run_id", "observation_id", "signal_id", "signal_ids", "nc_id"} {
			delete(ev.Data, k)
		}
		c, err := CanonicalOf(ev.Data)
		if err != nil {
			return s, err
		}
		switch {
		case i == 0:
			s.EventType, common = ev.EventType, c
		case ev.EventType != s.EventType:
			return s, fmt.Errorf("%w: %s и %s в одной пачке", ErrBatch, s.EventType, ev.EventType)
		case !bytes.Equal(c, common):
			return s, fmt.Errorf("%w: у изделия %s другое решение", ErrBatch, ev.ItemID)
		}
		s.Items = append(s.Items, ev.ItemID)
	}
	s.Count, s.Decision = len(items), string(common)
	return s, nil
}
