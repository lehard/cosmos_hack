package analytics_test

import (
	"context"
	"testing"

	"ant/internal/application/analytics"
	"ant/internal/application/platform"
)

// Имена узлов действующей версии процесса (заглушка порта StepNames).
type names map[string]string

func (n names) StepNames(context.Context) (map[string]string, error) { return n, nil }

// UI-21: у срезов «узел» подпись — имя узла BPMN действующей версии, ключ —
// по-прежнему step_key; у контрольной карты — step_name рядом со step_key.
// Без порта подпись остаётся кодом.
func TestStepSliceLabelsAreNodeNames(t *testing.T) {
	p := world(t)
	ov := p.overview(platform.Moment{})
	byName := names{}
	keys := map[string]string{}
	for _, r := range ov.Items {
		for _, s := range r.Slices {
			if s.Dimension != "step" {
				continue
			}
			if s.Label == "" {
				t.Fatalf("%s: пустая подпись среза %s", r.MetricID, s.Key)
			}
			step := s.Label // без порта подпись — код шага
			byName[step] = "Узел «" + step + "»"
			keys[r.MetricID+"|"+s.Key] = step
		}
	}
	if len(keys) == 0 {
		t.Fatal("в мире нет срезов по узлам")
	}
	p.svc = analytics.NewService(analytics.WithStore(analytics.MemStore{Src: p.j, Items: func() []string { return p.items }}),
		analytics.WithClock(p.clock), analytics.WithStepNames(byName))
	ov = p.overview(platform.Moment{})
	var step string
	for _, r := range ov.Items {
		for _, s := range r.Slices {
			if s.Dimension != "step" {
				continue
			}
			code, ok := keys[r.MetricID+"|"+s.Key]
			if !ok {
				t.Fatalf("%s: ключ среза изменился: %s", r.MetricID, s.Key)
			}
			if want := byName[code]; s.Label != want {
				t.Errorf("%s/%s: подпись %q, ждали %q", r.MetricID, s.Key, s.Label, want)
			}
			step = code
		}
	}
	cc, err := p.svc.ControlChart(context.Background(), step, "", analytics.PeriodQuery{Kind: "shift"}, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	if cc.StepKey != step || cc.StepName == nil || *cc.StepName != byName[step] {
		t.Errorf("контрольная карта: step_key %q, step_name %v", cc.StepKey, cc.StepName)
	}
}
