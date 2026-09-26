package operatorvision

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	appvision "ant/internal/application/vision"
	"ant/internal/infrastructure/integration/vision/visionqc"
)

// System — имя системы в локальном входе edge-агента: POST /v1/vision/operatorvision.
const System = "operatorvision"

// Action — гипотеза системы о действии у рабочего места —
// contracts/integrations/vision/operatorvision/action.v1.json. Сведений о
// человеке в протоколе нет: закрытая схема отвергает любое лишнее поле,
// в том числе попытку передать личность или лицо (FR-126).
type Action struct {
	HypothesisID    string                  `json:"hypothesis_id"`
	Seq             int64                   `json:"seq"`
	SystemID        string                  `json:"system_id"`
	CameraID        string                  `json:"camera_id"`
	RunID           string                  `json:"run_id,omitempty"`
	CreationTime    string                  `json:"creation_time"`
	IsSimulated     bool                    `json:"is_simulated,omitempty"`
	PartID          string                  `json:"part_id,omitempty"`
	PartIDKind      string                  `json:"part_id_kind,omitempty"`
	WorkplaceID     string                  `json:"workplace_id"`
	TPStep          string                  `json:"tp_step,omitempty"`
	Action          string                  `json:"action"`
	ConfidenceBP    *int                    `json:"confidence_bp,omitempty"`
	FrameQualityBP  *int                    `json:"frame_quality_bp,omitempty"`
	Recipe          visionqc.Recipe         `json:"recipe"`
	Configuration   *visionqc.Configuration `json:"configuration,omitempty"`
	Software        visionqc.Software       `json:"software"`
	ProductRevision string                  `json:"product_revision,omitempty"`
}

// Adapter — адаптер OperatorVision («Контроль действий оператора», FR-126):
// гипотеза о действии → operator.action.observed. Исполнителя не называет —
// его определяет ant по входу на рабочее место (domain/vision.ExecutorAt).
type Adapter struct{}

// New — адаптер OperatorVision.
func New() *Adapter { return &Adapter{} }

var _ appvision.SignalAdapter = (*Adapter)(nil)

// System — имя системы.
func (*Adapter) System() string { return System }

// Decode — гипотеза по закрытой схеме.
func Decode(raw []byte) (Action, error) {
	var a Action
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&a); err != nil {
		msg := err.Error()
		if strings.Contains(msg, "unknown field") {
			msg += " (сведений о человеке и лицах протокол не принимает: исполнитель — по входу на рабочее место, FR-126)"
		}
		return a, fmt.Errorf("OperatorVision: гипотеза не по контракту action.v1: %s", msg)
	}
	if a.HypothesisID == "" || a.WorkplaceID == "" || a.Action == "" || a.CreationTime == "" {
		return a, fmt.Errorf("OperatorVision: нет hypothesis_id, workplace_id, action или creation_time")
	}
	return a, nil
}

// Translate — гипотеза о действии на нашем языке.
func (*Adapter) Translate(raw []byte) (appvision.Signals, error) {
	a, err := Decode(raw)
	if err != nil {
		return appvision.Signals{}, err
	}
	if err := visionqc.CheckContract("OperatorVision", a.Software); err != nil {
		return appvision.Signals{}, err
	}
	at, err := time.Parse(time.RFC3339Nano, a.CreationTime)
	if err != nil {
		return appvision.Signals{}, fmt.Errorf("OperatorVision: creation_time: %w", err)
	}
	return appvision.Signals{System: System, Actions: []appvision.OperatorAction{{
		HypothesisID: a.SystemID + "/" + a.HypothesisID, OccurredAt: at.UTC(), RunID: a.RunID,
		Item: visionqc.ItemRef(a.PartID, a.PartIDKind), EquipmentID: a.CameraID, WorkplaceID: a.WorkplaceID,
		TPStep: a.TPStep, Action: a.Action, ConfidenceBP: a.ConfidenceBP, QualityBP: a.FrameQualityBP,
		Versions: visionqc.Versions(a.ProductRevision, a.Recipe, a.Configuration, a.Software), Simulated: a.IsSimulated,
	}}}, nil
}
