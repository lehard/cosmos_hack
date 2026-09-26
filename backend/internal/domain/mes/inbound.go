package mes

import (
	"strings"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Module — модуль-эмитент семейства mes (AD-40).
const Module kernel.Module = "mes"

// Job — задание MES на нашем языке (B2MML ProcessOperationsSchedule →
// OperationsRequest → SegmentRequirement).
type Job struct {
	// MessageID — номер сообщения MES (BODID).
	MessageID string
	// RequestID, SegmentID — ID запроса и требования сегмента в MES.
	RequestID, SegmentID string
	// OperationCode — код операции (ProcessSegmentID).
	OperationCode string
	// Material — обозначение КД изготавливаемого (MaterialDefinitionID), Quantity — количество.
	Material string
	Quantity int
	// Station — оборудование или участок MES (EquipmentID).
	Station      string
	PlannedStart time.Time
}

// Категории событий операций MES.
const (
	EventStart  = "start"
	EventEnd    = "end"
	EventPause  = "pause"
	EventResume = "resume"
	EventAbort  = "abort"
)

// OperationEvent — событие операции MES на нашем языке (B2MML NotifyOperationsEvent).
type OperationEvent struct {
	MessageID string
	// EventID — ID события в MES.
	EventID string
	// Category — start | end | pause | resume | abort.
	Category string
	// RunRef — выполнение сегмента в MES (SegmentResponseID): начало и конец
	// одного выполнения несут один RunRef.
	RunRef        string
	OperationCode string
	// SubLot — экземпляр изделия в MES (MaterialSubLot.ID).
	SubLot    string
	Equipment string
	Personnel string
	At        time.Time
}

// Env — соответствия и нормативный слой для перевода входящих MES (AD-18):
// шаг процесса по коду операции (BPMN `operationCode`) и записи соответствий
// внешних ID MES нашим (reference.external_id.mapped, система mes).
type Env struct {
	// Steps — код операции → step_key.
	Steps map[string]string
	// Items — экземпляр MES (MaterialSubLot.ID) → наш item_id.
	Items map[string]string
	// Persons — сотрудник MES → псевдоним исполнителя.
	Persons map[string]string
	// Equipment — оборудование MES → наш ID оборудования.
	Equipment map[string]string
}

// Fact — входящий факт для приёма: тип, детерминированный event_id,
// изделие (для фактов изделия), время и data по схеме контракта.
type Fact struct {
	Type       catalog.Type
	EventID    string
	ItemID     string
	OccurredAt time.Time
	Data       any
}

// Deferred — сообщение MES, которое нельзя перевести без соответствия
// (FR-123: не додумывать): причина для администратора.
type Deferred struct {
	MessageID, Ref, Reason string
}

func uid(kind, key string) string { return kernel.UUIDv5(constants.NsAnt, kind+"\x1f"+key) }

// JobID — наш ID задания MES: `MES-‹запрос›-‹сегмент›` латиницей.
func JobID(j Job) string { return "MES-" + latin(j.RequestID) + "-" + latin(j.SegmentID) }

// JobFacts — факт mes.job.received по заданию (event_id — от сообщения и
// сегмента: повторная доставка того же сообщения — дубль, AD-7).
func JobFacts(j Job) []Fact {
	d := ev.MesJobReceivedV1{JobID: ev.ObjectID(JobID(j)), ExternalNumber: j.RequestID, OperationCode: trimTo(j.OperationCode, 64)}
	if s := latin(j.Station); s != "" {
		x := ev.ObjectID(s)
		d.StationID = &x
	}
	if !j.PlannedStart.IsZero() {
		t := ev.NewTimestamp(j.PlannedStart)
		d.PlannedStart = &t
	}
	return []Fact{{Type: catalog.MesJobReceived, EventID: uid("mes.job", j.MessageID+"\x1f"+j.RequestID+"\x1f"+j.SegmentID), Data: d}}
}

// RunID — наш ID выполнения операции по выполнению сегмента MES.
func RunID(ref string) string { return "MES-" + latin(ref) }

// EventFacts — факт операции изделия по событию MES: шаг — по коду операции,
// изделие, исполнитель и оборудование — по соответствиям. Нет шага или
// изделия — Deferred (факт не выдумывается); неизвестный исполнитель — null
// («неизвестно», FR-123).
func EventFacts(env Env, e OperationEvent) ([]Fact, *Deferred) {
	defer1 := func(reason string) ([]Fact, *Deferred) {
		return nil, &Deferred{MessageID: e.MessageID, Ref: e.EventID, Reason: reason}
	}
	item := env.Items[e.SubLot]
	if item == "" && isItemID(e.SubLot) {
		item = e.SubLot
	}
	if item == "" {
		return defer1("нет соответствия экземпляра MES «" + e.SubLot + "» изделию ant (reference.external_id.mapped, система mes)")
	}
	if e.RunRef == "" {
		return defer1("нет выполнения сегмента (SegmentResponseID) — начало и конец не связать")
	}
	run := ev.ObjectID(RunID(e.RunRef))
	id := uid("mes.operation_event", e.MessageID+"\x1f"+e.EventID)
	f := Fact{EventID: id, ItemID: item, OccurredAt: e.At}
	switch e.Category {
	case EventStart:
		step := env.Steps[e.OperationCode]
		if step == "" {
			return defer1("код операции MES «" + e.OperationCode + "» не найден в нормативном слое (operationCode шага BPMN)")
		}
		d := ev.OperationRunStartedV1{OperationRunID: run, OperationCode: trimTo(e.OperationCode, 64), StepKey: ev.StepKey(step)}
		if p := env.Persons[e.Personnel]; p != "" {
			d.OperatorID = &p
		}
		if q := env.Equipment[e.Equipment]; q != "" {
			x := ev.ObjectID(q)
			d.EquipmentID = &x
		}
		t := ev.NewTimestamp(e.At)
		d.OperationStartedAt = &t
		f.Type, f.Data = catalog.OperationRunStarted, d
	case EventEnd, EventAbort:
		c := ev.OperationRunFinishedV1CompletionCompleted
		if e.Category == EventAbort {
			c = ev.OperationRunFinishedV1CompletionInterrupted
		}
		t := ev.NewTimestamp(e.At)
		f.Type, f.Data = catalog.OperationRunFinished, ev.OperationRunFinishedV1{OperationRunID: run, Completion: c, OperationFinishedAt: &t}
	case EventPause:
		f.Type, f.Data = catalog.OperationRunPaused, ev.OperationRunPausedV1{OperationRunID: run, PauseReason: ev.OperationRunPausedV1PauseReasonUnknown}
	case EventResume:
		f.Type, f.Data = catalog.OperationRunResumed, ev.OperationRunResumedV1{OperationRunID: run}
	default:
		return defer1("категория события «" + e.Category + "» не из подмножества mes.isa95.v1")
	}
	return []Fact{f}, nil
}

// isItemID — строка уже наш ID изделия `код_предприятия:локальный_id` (MES
// хранит наш ID, полученный в блокировке).
func isItemID(s string) bool {
	ent, local, ok := strings.Cut(s, ":")
	if !ok || ent == "" || local == "" || len(ent) > 16 {
		return false
	}
	for _, r := range ent {
		if !(r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	for _, r := range local {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || strings.ContainsRune("._/-", r)) {
			return false
		}
	}
	return true
}

var tr = map[rune]string{'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D", 'Е': "E", 'Ё': "E", 'Ж': "ZH", 'З': "Z", 'И': "I", 'Й': "Y",
	'К': "K", 'Л': "L", 'М': "M", 'Н': "N", 'О': "O", 'П': "P", 'Р': "R", 'С': "S", 'Т': "T", 'У': "U", 'Ф': "F", 'Х': "KH", 'Ц': "TS",
	'Ч': "CH", 'Ш': "SH", 'Щ': "SHCH", 'Ъ': "", 'Ы': "Y", 'Ь': "", 'Э': "E", 'Ю': "YU", 'Я': "YA"}

// latin — ID MES латиницей (кириллица транслитерируется, прочее — «-»).
func latin(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		u := r
		if r >= 'а' && r <= 'я' || r == 'ё' {
			u = r - 'а' + 'А'
			if r == 'ё' {
				u = 'Ё'
			}
		}
		switch {
		case r < 128 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._-", r)):
			b.WriteRune(r)
		case tr[u] != "" || u == 'Ъ' || u == 'Ь':
			b.WriteString(tr[u])
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func trimTo(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}
