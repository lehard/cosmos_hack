package erp

// Books — учёт субъекта (изделия или партии) глазами модуля erp: где он
// числится по уже сформированным учётным сообщениям, в браке ли, сколько было
// циклов брака, по какому разрешению на отклонение принят. Это состояние
// реакции «учётное сообщение сформировано» (роль projector): чистая свёртка
// записей журнала по субъекту, её пишет только erp (AD-45).
type Books struct {
	// Subject — изделие (`item_id`) или партия (`lot_id`).
	Subject string `json:"subject"`
	// Lot — субъект — партия.
	Lot bool `json:"lot,omitempty"`
	// OrderID, ItemTypeID — задание и номенклатура (из регистрации изделия или поступления партии).
	OrderID    string `json:"order_id,omitempty"`
	ItemTypeID string `json:"item_type_id,omitempty"`
	// Warehouse — склад, где субъект числится по учётным сообщениям (FR-130).
	Warehouse string `json:"warehouse,omitempty"`
	// InDefect, DefectAction, DefectCycle — субъект в браке, вид последнего
	// перевода и номер цикла брака (перевод в брак → возврат из брака).
	InDefect     bool   `json:"in_defect,omitempty"`
	DefectAction Action `json:"defect_action,omitempty"`
	DefectCycle  int    `json:"defect_cycle,omitempty"`
	// Reworked — изделие возвращалось из брака: выпуск — с признаком after_rework.
	Reworked bool `json:"reworked,omitempty"`
	// ConcessionID — разрешение на отклонение, по которому изделие принято.
	ConcessionID string `json:"concession_id,omitempty"`
	// Disposition — последнее решение по несоответствию изделия.
	Disposition *Disposition `json:"disposition,omitempty"`
	// Occurrences — события-сообщения процесса по шагам: слот реакции → номер
	// прохождения шага (повторный проход — новый бизнес-ключ, новая версия
	// того же слота — тот же ключ).
	Occurrences []Occurrence `json:"occurrences,omitempty"`
	// Decisions — решения на закрывающих точках и их точка бизнес-ключа
	// (исправление решения — новая версия того же сообщения).
	Decisions []DecisionPoint `json:"decisions,omitempty"`
	// Received, Issued — партия: поступило и выдано в производство.
	Received int `json:"received,omitempty"`
	Issued   int `json:"issued,omitempty"`
}

// Occurrence — прохождение шага процесса: слот события-сообщения.
type Occurrence struct {
	Step string `json:"step"`
	Slot string `json:"slot"`
}

// DecisionPoint — решение на закрывающей точке и точка его сообщения.
type DecisionPoint struct {
	EventID string `json:"event_id"`
	Point   string `json:"point"`
}

// Disposition — решение по несоответствию: основание перевода в брак и возврата.
type Disposition struct {
	NCID         string `json:"nc_id"`
	Disposition  string `json:"disposition"`
	ClaimBasis   string `json:"claim_basis,omitempty"`
	ConcessionID string `json:"concession_id,omitempty"`
}

// occurrence — номер прохождения шага step для слота slot (1…); новый слот
// дописывается.
func (b *Books) occurrence(step, slot string) int {
	n := 0
	for _, o := range b.Occurrences {
		if o.Step != step {
			continue
		}
		n++
		if slot != "" && o.Slot == slot {
			return n
		}
	}
	b.Occurrences = append(b.Occurrences, Occurrence{Step: step, Slot: slot})
	return n + 1
}

// decisionPoint — точка сообщения по решению (исправление — точка исправляемого).
func (b *Books) decisionPoint(eventID, corrects, point string) string {
	if corrects != "" {
		for _, d := range b.Decisions {
			if d.EventID == corrects {
				point = d.Point
				break
			}
		}
	}
	b.Decisions = append(b.Decisions, DecisionPoint{EventID: eventID, Point: point})
	return point
}

// Remaining — партия: остаток на складе (поступило − выдано), не меньше нуля.
func (b Books) Remaining() int { return max(b.Received-b.Issued, 0) }
