package analytics

import (
	"encoding/json"
	"strings"
	"time"
)

// Данные записей, которые читает функция вклада. Свои узкие структуры вместо
// сгенерированных типов: читаются только нужные поля, неизвестные и
// отсутствующие поля не ломают свёртку (FR-29: новое необязательное поле
// принимается). Data уже повышена до текущей версии схемы (AD-20).

type runStartedData struct {
	RunID     string     `json:"operation_run_id"`
	Code      string     `json:"operation_code"`
	Step      string     `json:"step_key"`
	Line      string     `json:"line_id"`
	Station   string     `json:"station_id"`
	Equipment string     `json:"equipment_id"`
	Operator  *string    `json:"operator_id"`
	ReworkOf  string     `json:"rework_of"`
	StartedAt *time.Time `json:"operation_started_at"`
}

type durationData struct {
	Value   int64  `json:"value"`
	Unit    string `json:"unit"`
	Meaning string `json:"meaning"`
	Origin  string `json:"origin"`
}

// seconds — длительность источника в секундах (единицы ms, s, min, h).
func (d durationData) seconds() int64 {
	switch d.Unit {
	case "ms":
		return d.Value / 1000
	case "min":
		return d.Value * 60
	case "h":
		return d.Value * 3600
	}
	return d.Value
}

type runFinishedData struct {
	RunID      string        `json:"operation_run_id"`
	Completion string        `json:"completion"`
	StartedAt  *time.Time    `json:"operation_started_at"`
	FinishedAt *time.Time    `json:"operation_finished_at"`
	Reported   *durationData `json:"reported_duration"`
}

type runRefData struct {
	RunID string `json:"operation_run_id"`
}

type defectData struct {
	Type     string `json:"defect_type_code"`
	Zone     string `json:"zone_id"`
	Location string `json:"location"`
	Severity string `json:"severity"`
}

type inspectionData struct {
	ObservationID string       `json:"observation_id"`
	Phase         string       `json:"phase"`
	Step          string       `json:"step_key"`
	RunID         string       `json:"operation_run_id"`
	Outcome       string       `json:"outcome"`
	UnableReason  string       `json:"unable_reason"`
	Processing    string       `json:"processing_state"`
	ZoneIDs       []string     `json:"zone_ids"`
	Defects       []defectData `json:"defects"`
	Equipment     string       `json:"equipment_id"`
	Inspector     *string      `json:"inspector_id"`
}

type presentationData struct {
	Step       string `json:"step_key"`
	No         int    `json:"presentation_no"`
	Resolution string `json:"resolution"`
}

type movementReceivedData struct {
	Step       string `json:"step_key"`
	To         string `json:"to_location_id"`
	Inspection string `json:"inspection_on_receipt"`
}

type ncData struct {
	NC         string   `json:"nc_id"`
	Signals    []string `json:"signal_ids"`
	DefectType *string  `json:"defect_type_code"`
	RunID      string   `json:"operation_run_id"`
	Step       string   `json:"step_key"`
}

type dispositionData struct {
	NC          string `json:"nc_id"`
	Disposition string `json:"disposition"`
}

type registeredData struct {
	ItemType string `json:"item_type_id"`
	Entry    string `json:"entry_step_key"`
}

// decode — разбор data записи; ошибка разбора — пустое значение и false:
// показатели не останавливают свёртку изделия из-за чужого поля.
func decode[T any](raw json.RawMessage) (T, bool) {
	var v T
	if len(raw) == 0 {
		return v, false
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return v, false
	}
	return v, true
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// label — подпись изделия: локальный ID без кода предприятия.
func label(itemID string) string {
	if _, local, ok := strings.Cut(itemID, ":"); ok {
		return local
	}
	return itemID
}
