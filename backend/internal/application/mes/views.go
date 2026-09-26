package mes

import "time"

// MesJob — задание MES (mes.job.received, FR-92).
type MesJob struct {
	JobID          string     `json:"job_id"`
	ExternalNumber string     `json:"external_number"`
	OrderID        string     `json:"order_id,omitempty"`
	OperationCode  string     `json:"operation_code"`
	StationID      string     `json:"station_id,omitempty"`
	PlannedStart   *time.Time `json:"planned_start,omitempty"`
	ReceivedAt     time.Time  `json:"received_at"`
}

// MesJobList — задания MES.
type MesJobList struct {
	Items      []MesJob `json:"items"`
	NextCursor string   `json:"next_cursor,omitempty"`
}

// MesBlock — блокировка изделия или партии в MES (mes.hold.*, FR-93, AD-30):
// реакция на блок в журнале, подтверждение — квитанцией MES.
type MesBlock struct {
	BusinessKey string     `json:"business_key"`
	ItemID      string     `json:"item_id,omitempty"`
	LotID       string     `json:"lot_id,omitempty"`
	Hold        bool       `json:"hold" doc:"true — заблокировать, false — снять."`
	RequestedAt time.Time  `json:"requested_at"`
	Outcome     string     `json:"outcome,omitempty" enum:"accepted,duplicate,rejected" doc:"Квитанция MES; пусто — ждём."`
	ErrorCode   string     `json:"error_code,omitempty"`
	RespondedAt *time.Time `json:"responded_at,omitempty"`
}

// MesBlockList — блокировки в MES.
type MesBlockList struct {
	Items []MesBlock `json:"items"`
}
