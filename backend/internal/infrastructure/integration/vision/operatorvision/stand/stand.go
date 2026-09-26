package stand

import (
	"fmt"
	"time"

	"ant/internal/infrastructure/integration/vision/operatorvision"
	"ant/internal/infrastructure/integration/vision/standkit"
	"ant/internal/infrastructure/integration/vision/visionqc"
)

// Сцены — гипотезы о действиях (FR-126).
var Scenes = []string{"step_missed", "sequence_broken", "extra_step", "tool_missing", "tool_returned", "manual_intervention"}

// steps — шаг ТП, к которому относится гипотеза сцены.
var steps = map[string]string{
	"step_missed":         "Затяжка крепежа крышки по схеме «крест»",
	"sequence_broken":     "Установка уплотнения S-1 до крышки",
	"extra_step":          "Повторная установка крышки",
	"tool_missing":        "Динамометрический ключ на месте",
	"tool_returned":       "Динамометрический ключ на месте",
	"manual_intervention": "Подгонка крышки вручную",
}

// Camera — обзорная камера рабочего места и допущенная конфигурация.
type Camera struct {
	SystemID    string
	CameraID    string
	WorkplaceID string
	Revision    string
	Recipe      visionqc.Recipe
	Config      visionqc.Configuration
	Software    visionqc.Software
}

// Assembly — камера рабочего места сборки; вектор версий — как у паспорта
// AP-OV-ASM-1 затравки (ov-asm@1).
func Assembly() Camera {
	return Camera{SystemID: "ov-asm", CameraID: "CAM-OV-ASM", WorkplaceID: "WP-ASM-1", Revision: "Б",
		Recipe:   visionqc.Recipe{ID: "ov-asm", Version: "1"},
		Config:   visionqc.Configuration{CameraConfig: "CAM-OV-ASM/обзор", Calibration: "CAL-OV-2026-01-10", ThresholdProfile: "TP-OV-1"},
		Software: visionqc.Software{AnalyzerVersion: "ov-asm 0.9.4", AppVersion: "ov-gw 0.9.0", ContractVersion: "1.0"}}
}

func bp(v int) *int { return &v }

// Action — гипотеза системы для сцены.
func Action(c Camera, seq int64, at time.Time, sh standkit.Shot) (operatorvision.Action, error) {
	st, ok := steps[sh.Kind]
	if !ok {
		return operatorvision.Action{}, fmt.Errorf("OperatorVision stand: сцена %q не поддерживается", sh.Kind)
	}
	a := operatorvision.Action{HypothesisID: fmt.Sprintf("%s-%06d", c.CameraID, seq), Seq: seq, SystemID: c.SystemID, CameraID: c.CameraID,
		RunID: sh.RunID, CreationTime: at.Format("2006-01-02T15:04:05.000Z"), PartID: sh.PartID, PartIDKind: sh.PartIDKind,
		WorkplaceID: c.WorkplaceID, TPStep: st, Action: sh.Kind, ConfidenceBP: bp(7400), FrameQualityBP: bp(8600),
		Recipe: c.Recipe, Configuration: &c.Config, Software: c.Software, ProductRevision: c.Revision}
	if a.PartID != "" && a.PartIDKind == "" {
		a.PartIDKind = "internal"
	}
	return a, nil
}

// Options — настройки stand-а.
type Options struct {
	Name     string
	EdgeURL  string
	Interval time.Duration
	Camera   Camera
	// PartID — деталь на рабочем месте в программе (пусто — без детали).
	PartID string
}

// New — stand OperatorVision с программой «шаг пропущен → порядок нарушен → нет инструмента».
func New(o Options) *standkit.Stand {
	if o.Camera.CameraID == "" {
		o.Camera = Assembly()
	}
	if o.PartID == "" {
		o.PartID = "ENT01:F-231"
	}
	prog := []standkit.Shot{}
	for _, k := range []string{"step_missed", "sequence_broken", "tool_missing"} {
		prog = append(prog, standkit.Shot{Kind: k, PartID: o.PartID, PartIDKind: "internal"})
	}
	c := o.Camera
	return &standkit.Stand{Name: o.Name, Path: "/v1/vision/" + operatorvision.System, EdgeURL: o.EdgeURL, Interval: o.Interval,
		Emulates: "OperatorVision, камера " + c.CameraID + " рабочего места " + c.WorkplaceID + ": гипотезы о действиях → edge-агент",
		Boundary: "нет камеры и модели действий, лица не распознаются — гипотезы заданы программой; протокол action.v1 → edge-агент настоящий",
		Kinds:    Scenes, Program: prog,
		Build: func(seq int64, at time.Time, sh standkit.Shot) (any, error) { return Action(c, seq, at, sh) }}
}
