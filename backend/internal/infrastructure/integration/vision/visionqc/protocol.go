package visionqc

// Протокол внешней системы VisionQC — contracts/integrations/vision/visionqc/result.v1.json
// (по образцу ResultDataType OPC UA for Machine Vision, OPC 40100-1). Типы —
// внешний формат: дальше адаптера не выходят (AD-18). Соответствие типов
// схеме проверяет контрактный тест на эталонных сообщениях (examples/).

// Result — результат системы по одному срабатыванию камеры.
type Result struct {
	ResultID        string         `json:"result_id"`
	Seq             int64          `json:"seq"`
	SystemID        string         `json:"system_id"`
	CameraID        string         `json:"camera_id"`
	RunID           string         `json:"run_id,omitempty"`
	CreationTime    string         `json:"creation_time"`
	IsSimulated     bool           `json:"is_simulated,omitempty"`
	PartID          string         `json:"part_id,omitempty"`
	PartIDKind      string         `json:"part_id_kind,omitempty"`
	MeasID          string         `json:"meas_id,omitempty"`
	Station         string         `json:"station,omitempty"`
	ResultState     int            `json:"result_state"`
	Recipe          Recipe         `json:"recipe"`
	Configuration   *Configuration `json:"configuration,omitempty"`
	Software        Software       `json:"software"`
	ProductRevision string         `json:"product_revision,omitempty"`
	Frame           *Frame         `json:"frame,omitempty"`
	Verdict         string         `json:"verdict,omitempty"`
	ConfidenceBP    *int           `json:"confidence_bp,omitempty"`
	Stages          []Stage        `json:"stages,omitempty"`
	Findings        []Finding      `json:"findings,omitempty"`
	Zones           []string       `json:"zones,omitempty"`
	Recommendation  string         `json:"recommendation,omitempty"`
	Images          []Image        `json:"images,omitempty"`
}

// Recipe — карта контроля (рецепт) системы.
type Recipe struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// Configuration — камера и свет, калибровка, профиль порогов.
type Configuration struct {
	CameraConfig     string `json:"camera_config,omitempty"`
	Calibration      string `json:"calibration,omitempty"`
	ThresholdProfile string `json:"threshold_profile,omitempty"`
}

// Software — версии программ системы.
type Software struct {
	AnalyzerVersion string `json:"analyzer_version,omitempty"`
	AppVersion      string `json:"app_version,omitempty"`
	ContractVersion string `json:"contract_version,omitempty"`
}

// Frame — качество кадра и помехи.
type Frame struct {
	QualityBP int      `json:"quality_bp"`
	Issues    []string `json:"issues,omitempty"`
}

// Stage — ступень анализатора.
type Stage struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	ConfidenceBP *int   `json:"confidence_bp,omitempty"`
	Output       string `json:"output,omitempty"`
}

// Finding — найденный признак.
type Finding struct {
	Class        string `json:"class"`
	Zone         string `json:"zone"`
	Location     string `json:"location,omitempty"`
	Severity     string `json:"severity,omitempty"`
	ConfidenceBP *int   `json:"confidence_bp,omitempty"`
	Note         string `json:"note,omitempty"`
}

// Image — снимок у системы.
type Image struct {
	URI       string `json:"uri"`
	MediaType string `json:"media_type,omitempty"`
}

// Состояния обработки ResultState (OPC 40100-1, 12.19).
const (
	StateUndefined  = 0
	StateCompleted  = 1
	StateProcessing = 2
	StateAborted    = 3
	StateFailed     = 4
)
