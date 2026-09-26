package stand

import (
	"fmt"
	"strconv"
	"time"

	"ant/internal/infrastructure/integration/vision/standkit"
	"ant/internal/infrastructure/integration/vision/visionqc"
)

// Сцены программы КТ-3.
const (
	SceneWeldOK          = "weld_ok"
	SceneWeldPores       = "weld_pores"
	SceneWeldUndercut    = "weld_undercut"
	SceneWeldBurnThrough = "weld_burnthrough"
	// SceneBadFrame — «испорченный кадр»: качество наблюдения 0,3, а система
	// всё равно говорит «признаков нет» — ant не должен принять это за годность.
	SceneBadFrame = "bad_frame"
	// SceneGlare — блик на шве: система сама говорит «оценить нельзя».
	SceneGlare = "glare"
	// SceneAborted — обработка прервана (ResultState = 3).
	SceneAborted = "aborted"
)

// Scenes — все сцены stand-а.
var Scenes = []string{SceneWeldOK, SceneWeldPores, SceneWeldUndercut, SceneWeldBurnThrough, SceneBadFrame, SceneGlare, SceneAborted}

// MainStory — главная история КТ-3 по кругу.
var MainStory = []string{SceneWeldOK, SceneWeldPores, SceneBadFrame, SceneWeldOK, SceneGlare, SceneWeldBurnThrough, SceneWeldUndercut}

// Camera — камера точки контроля и допущенная конфигурация контура (AD-29).
type Camera struct {
	SystemID string
	CameraID string
	Station  string
	Revision string
	Recipe   visionqc.Recipe
	Config   visionqc.Configuration
	Software visionqc.Software
	Zones    []string
}

// KT3 — камера сварного шва КТ-3; вектор версий совпадает с паспортом
// AP-KT3-WELD-1 затравки (карта контроля kt3-weld@1).
func KT3() Camera {
	return Camera{SystemID: "vqc-kt3", CameraID: "CAM-KT3", Station: "KT-3", Revision: "Б",
		Recipe:   visionqc.Recipe{ID: "kt3-weld", Version: "1"},
		Config:   visionqc.Configuration{CameraConfig: "CAM-KT3/свет-R2", Calibration: "CAL-KT3-2026-01-10", ThresholdProfile: "TP-KT3-5"},
		Software: visionqc.Software{AnalyzerVersion: "vqc-weld 2.3.1", AppVersion: "vqc-gw 1.2.0", ContractVersion: "1.0"},
		Zones:    []string{"W-1.U1", "W-1.U2", "W-1.U3", "W-1.U4", "W-1.U5", "W-1.U6", "W-1.U7", "W-1.U8"}}
}

func bp(v int) *int { return &v }

// Result — сообщение системы для сцены: ступени с версией и уверенностью,
// качество кадра отдельно от уверенности (FR-38).
func Result(c Camera, seq int64, at time.Time, sh standkit.Shot) (visionqc.Result, error) {
	ver := c.Software.AnalyzerVersion
	r := visionqc.Result{ResultID: fmt.Sprintf("%s-%06d", c.CameraID, seq), Seq: seq, SystemID: c.SystemID, CameraID: c.CameraID,
		RunID: sh.RunID, CreationTime: at.Format("2006-01-02T15:04:05.000Z"), PartID: sh.PartID, PartIDKind: sh.PartIDKind,
		Station: c.Station, ResultState: visionqc.StateCompleted, Recipe: c.Recipe, Configuration: &c.Config, Software: c.Software,
		ProductRevision: c.Revision, Zones: c.Zones}
	if r.PartID != "" && r.PartIDKind == "" {
		r.PartIDKind = "internal"
	}
	stages := func(loc, cls int, locOut, clsOut string) []visionqc.Stage {
		return []visionqc.Stage{{Name: "localize", Version: ver, ConfidenceBP: bp(loc), Output: locOut},
			{Name: "classify", Version: ver, ConfidenceBP: bp(cls), Output: clsOut}}
	}
	defect := func(class, zone, loc, sev string, conf int, note string) {
		r.Verdict, r.ConfidenceBP, r.Recommendation = "defects", bp(conf), "reject"
		r.Frame = &visionqc.Frame{QualityBP: 9100}
		r.Stages = stages(9300, conf, "признак на участке "+zone, class)
		r.Findings = []visionqc.Finding{{Class: class, Zone: zone, Location: loc, Severity: sev, ConfidenceBP: bp(conf), Note: note}}
	}
	switch sh.Kind {
	case SceneWeldOK:
		r.Verdict, r.ConfidenceBP, r.Recommendation = "no_defects", bp(9600), "pass"
		r.Frame = &visionqc.Frame{QualityBP: 9200}
		r.Stages = stages(9500, 9600, "признаков не найдено", "—")
	case SceneWeldPores:
		defect("W-PORE-S", "W-1.U3", "90–110 мм", "major", 7200, "поры Ø0,5–0,8 мм, 3 шт.")
		r.Recommendation = "review"
	case SceneWeldUndercut:
		defect("W-UNDERCUT", "W-1.U5", "175–190 мм", "major", 8100, "подрез глубиной ≈0,4 мм (оценка по изображению)")
	case SceneWeldBurnThrough:
		defect("W-BURNTHRU", "W-1.U2", "52–58 мм", "critical", 9100, "прожог Ø≈2 мм")
	case SceneBadFrame:
		r.Verdict, r.ConfidenceBP, r.Recommendation = "no_defects", bp(8800), "pass"
		r.Frame = &visionqc.Frame{QualityBP: 3000}
		r.Stages = stages(4100, 8800, "шов найден частично", "признаков не найдено")
	case SceneGlare:
		r.Verdict, r.Recommendation = "cannot_evaluate", "review"
		r.Frame = &visionqc.Frame{QualityBP: 4500, Issues: []string{"glare"}}
		r.Stages = []visionqc.Stage{{Name: "localize", Version: ver, ConfidenceBP: bp(3100), Output: "блик на участке У4 — зона не читается"}}
	case SceneAborted:
		r.ResultState, r.Recommendation = visionqc.StateAborted, "review"
	default:
		return r, fmt.Errorf("VisionQC stand: сцена %q не поддерживается", sh.Kind)
	}
	return r, nil
}

// Options — настройки stand-а.
type Options struct {
	Name     string
	EdgeURL  string
	Interval time.Duration
	Camera   Camera
	// PartPrefix, FirstPart — детали главной истории: ‹префикс›‹номер›, номер растёт.
	PartPrefix string
	FirstPart  int
}

// New — stand камеры КТ-3 с главной историей по кругу.
func New(o Options) *standkit.Stand {
	if o.Camera.CameraID == "" {
		o.Camera = KT3()
	}
	if o.PartPrefix == "" {
		o.PartPrefix, o.FirstPart = "ENT01:F-", 231
	}
	prog := make([]standkit.Shot, 0, len(MainStory))
	for i, k := range MainStory {
		prog = append(prog, standkit.Shot{Kind: k, PartID: o.PartPrefix + strconv.Itoa(o.FirstPart+i), PartIDKind: "internal"})
	}
	c := o.Camera
	return &standkit.Stand{Name: o.Name, Path: "/v1/vision/" + visionqc.System, EdgeURL: o.EdgeURL, Interval: o.Interval,
		Emulates: "VisionQC, камера " + c.CameraID + " (" + c.Station + "): ступени анализатора → edge-агент",
		Boundary: "нет камеры, света и нейросети — выводы ступеней заданы программой; протокол result.v1 → edge-агент настоящий",
		Kinds:    Scenes, Program: prog,
		Build: func(seq int64, at time.Time, sh standkit.Shot) (any, error) { return Result(c, seq, at, sh) }}
}
