package visionqc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	appvision "ant/internal/application/vision"
	dom "ant/internal/domain/vision"
)

// System — имя системы в локальном входе edge-агента: POST /v1/vision/visionqc.
const System = "visionqc"

// Station — точка контроля, на которой стоит камера (конфигурация установки,
// FR-103): точка, шаг процесса и фаза контроля. Камеры, которой нет в
// конфигурации, адаптер не выдумывает: точка берётся из сообщения, шаг пуст.
type Station struct {
	InspectionPoint string
	StepKey         string
	Phase           string
}

// DefaultStations — камеры демо-процесса фланца (normative/process/flange-process.bpmn:
// ant:inspection method="camera").
func DefaultStations() map[string]Station {
	return map[string]Station{
		"CAM-KT1":  {InspectionPoint: "KT-1", StepKey: "incoming.kt1_camera", Phase: "incoming"},
		"CAM-KT2":  {InspectionPoint: "KT-2", StepKey: "machining.kt2_camera", Phase: "after_operation"},
		"CAM-KT3":  {InspectionPoint: "KT-3", StepKey: "welding.kt3_camera", Phase: "after_operation"},
		"CAM-KT4D": {InspectionPoint: "KT-4d", StepKey: "assembly.kt4d_zone_camera", Phase: "before_zone_closure"},
		"CAM-KT4":  {InspectionPoint: "KT-4", StepKey: "assembly.kt4_camera", Phase: "assembly"},
		"CAM-KT5":  {InspectionPoint: "KT-5", StepKey: "final.kt5_camera", Phase: "final"},
	}
}

// Adapter — адаптер VisionQC (FR-97, AD-18): переводит результат системы в
// наблюдение на нашем языке. Реализует порт application/vision.SignalAdapter;
// работает на краю (edge-агент), в ядро идут только события контракта.
type Adapter struct {
	Stations map[string]Station
}

// New — адаптер с камерами stations (nil — DefaultStations).
func New(stations map[string]Station) *Adapter {
	if stations == nil {
		stations = DefaultStations()
	}
	return &Adapter{Stations: stations}
}

var _ appvision.SignalAdapter = (*Adapter)(nil)

// System — имя системы.
func (*Adapter) System() string { return System }

// Decode — результат системы по закрытой схеме: лишнее поле — отказ (AD-10).
func Decode(raw []byte) (Result, error) {
	var r Result
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return r, fmt.Errorf("VisionQC: результат не по контракту result.v1: %w", err)
	}
	if r.ResultID == "" || r.CameraID == "" || r.Recipe.ID == "" || r.CreationTime == "" {
		return r, fmt.Errorf("VisionQC: нет result_id, camera_id, recipe.id или creation_time")
	}
	return r, nil
}

// ContractMajor — мажорная версия протокола, которую понимает адаптер.
const ContractMajor = "1"

// CheckContract — сверка версии контракта ответной стороны (AD-18): система
// другой мажорной версии не принимается — сообщение отвергается целиком, а не
// толкуется наугад. Версия не сообщена — наблюдение идёт с unknown.
func CheckContract(system string, sw Software) error {
	v := strings.TrimSpace(sw.ContractVersion)
	if v == "" {
		return nil
	}
	if major, _, _ := strings.Cut(v, "."); major != ContractMajor {
		return fmt.Errorf("%s: версия контракта системы %s несовместима с адаптером (%s.x) — канал деградирован, сообщение не принято", system, v, ContractMajor)
	}
	return nil
}

// frameText — помехи кадра по-русски.
var frameText = map[string]string{"glare": "блик", "blur": "смаз", "out_of_frame": "деталь вне кадра", "dirty_lens": "грязный объектив",
	"underexposed": "недосвет", "overexposed": "пересвет", "occluded": "зона закрыта"}

// carrier — тип носителя по виду part_id (AD-16): значение метки — носитель, не item_id.
func carrier(kind string) string {
	switch kind {
	case "datamatrix":
		return "dpm_datamatrix"
	case "qr":
		return "tag_qr"
	}
	return "internal_id"
}

// ItemRef — ссылка на изделие из PartId; нет — наблюдение без изделия
// (привязку по «станция + время» делает стадия, AD-41).
func ItemRef(partID, kind string) *appvision.ItemRef {
	if partID == "" {
		return nil
	}
	return &appvision.ItemRef{CarrierType: carrier(kind), Value: partID, IdentificationLevel: "unique"}
}

// Translate — наблюдения из результата (FR-38): состояние обработки отдельно
// от вывода — «прервано» и «сбой» не выдаются за «признаков нет»; помехи
// кадра — в ограничениях; вектор версий — из рецепта, конфигурации и версий программ.
func (a *Adapter) Translate(raw []byte) (appvision.Signals, error) {
	r, err := Decode(raw)
	if err != nil {
		return appvision.Signals{}, err
	}
	if err := CheckContract("VisionQC", r.Software); err != nil {
		return appvision.Signals{}, err
	}
	at, err := time.Parse(time.RFC3339Nano, r.CreationTime)
	if err != nil {
		return appvision.Signals{}, fmt.Errorf("VisionQC: creation_time: %w", err)
	}
	st, known := a.Stations[r.CameraID]
	in := appvision.Inspection{ObservationID: r.SystemID + "/" + r.ResultID, OccurredAt: at.UTC(), RunID: r.RunID,
		Item: ItemRef(r.PartID, r.PartIDKind), EquipmentID: r.CameraID, Point: r.Station,
		ConfidenceBP: r.ConfidenceBP, Recommendation: recommendation(r.Recommendation), Zones: r.Zones, Simulated: r.IsSimulated}
	if known {
		in.StepKey, in.Phase = st.StepKey, st.Phase
		if in.Point == "" {
			in.Point = st.InspectionPoint
		}
	} else {
		in.Limitations = append(in.Limitations, "камера "+r.CameraID+" не описана в установке: шаг процесса не определён")
	}
	var issues []string
	if r.Frame != nil {
		q := r.Frame.QualityBP
		in.QualityBP = &q
		for _, i := range r.Frame.Issues {
			issues = append(issues, frameText[i])
		}
		if len(issues) > 0 {
			in.Limitations = append(in.Limitations, "помехи кадра: "+strings.Join(issues, ", "))
		}
	}
	in.ProcessingState, in.Outcome, in.UnableReason = interpret(r)
	if r.ResultState == StateProcessing || r.ResultState <= StateUndefined {
		in.Limitations = append(in.Limitations, "состояние обработки у системы: "+strconv.Itoa(r.ResultState))
	}
	if len(r.Images) > 0 {
		// Снимки системы в ant не передаются (кейс §5.4: тихая потеря снимков —
		// пометка, а не ошибка); ссылка на снимок у системы — в ограничениях.
		in.Limitations = append(in.Limitations, "снимок у системы: "+r.Images[0].URI)
	}
	for _, s := range r.Stages {
		in.Stages = append(in.Stages, appvision.Stage{Name: s.Name, Version: s.Version, ConfidenceBP: s.ConfidenceBP, Output: s.Output})
	}
	for _, f := range r.Findings {
		sev := f.Severity
		if sev == "" {
			sev = "unknown"
		}
		in.Findings = append(in.Findings, appvision.Finding{DefectCode: f.Class, Zone: f.Zone, Location: f.Location, Severity: sev,
			ConfidenceBP: f.ConfidenceBP, Note: f.Note})
	}
	in.Versions = Versions(r.ProductRevision, r.Recipe, r.Configuration, r.Software)
	return appvision.Signals{System: System, Inspections: []appvision.Inspection{in}}, nil
}

// Versions — вектор версий наблюдения из полей протокола (AD-29).
func Versions(rev string, rc Recipe, c *Configuration, sw Software) dom.Versions {
	v := dom.Versions{ItemRevision: rev, RecipeRef: dom.RecipeRef(rc.ID, rc.Version), AnalyzerVersion: sw.AnalyzerVersion,
		ContractVersion: sw.ContractVersion, AppVersion: sw.AppVersion}
	if c != nil {
		v.CameraConfig, v.Calibration, v.ThresholdProfile = c.CameraConfig, c.Calibration, c.ThresholdProfile
	}
	return v
}

// interpret — состояние обработки, исход источника и причина «оценка невозможна».
func interpret(r Result) (state, outcome, unable string) {
	switch r.ResultState {
	case StateCompleted:
		state = "completed"
	case StateAborted, StateProcessing:
		state = "aborted"
	default:
		state = "failed"
	}
	switch r.Verdict {
	case "defects":
		outcome = "defect_indicated"
	case "no_defects":
		outcome = "no_defect_indicated"
	default:
		outcome = "unable_to_assess"
	}
	if state != "completed" {
		outcome = "unable_to_assess"
	}
	if outcome != "unable_to_assess" {
		return state, outcome, ""
	}
	switch {
	case state == "failed":
		unable = "analyzer_failure"
	case state == "aborted":
		unable = "processing_aborted"
	case r.Frame != nil && hasAny(r.Frame.Issues, "occluded", "out_of_frame"):
		unable = "zone_occluded"
	case r.Frame != nil && len(r.Frame.Issues) > 0:
		unable = "poor_image"
	default:
		unable = "other"
	}
	return state, outcome, unable
}

func hasAny(xs []string, ys ...string) bool {
	for _, x := range xs {
		for _, y := range ys {
			if x == y {
				return true
			}
		}
	}
	return false
}

// recommendation — совет системы в словаре контракта (только совет, FR-48).
func recommendation(s string) string {
	switch s {
	case "pass":
		return "pass_to_next"
	case "review":
		return "manual_review"
	case "reject":
		return "isolate"
	case "none":
		return "none"
	}
	return ""
}
