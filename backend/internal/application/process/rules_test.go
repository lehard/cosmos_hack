package process_test

import (
	"os"
	"slices"
	"testing"

	"go.yaml.in/yaml/v3"

	dp "ant/internal/domain/process"
)

// Переменные языка условий домена совпадают с contracts/bpmn-ext/rules.yaml
// (conditions.variables): один источник правил (AD-20) — загрузчик движка
// применяет те же правила, что проверка контрактов (check-bpmn.mjs).
func TestVariablesMatchRules(t *testing.T) {
	b, err := os.ReadFile("../../../../contracts/bpmn-ext/rules.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var rules struct {
		ExpressionLanguage string `yaml:"expression_language"`
		Conditions         struct {
			Variables map[string]struct {
				Type   string   `yaml:"type"`
				Values []string `yaml:"values"`
			} `yaml:"variables"`
		} `yaml:"conditions"`
	}
	if err := yaml.Unmarshal(b, &rules); err != nil {
		t.Fatal(err)
	}
	if rules.ExpressionLanguage != dp.ExprLanguage {
		t.Fatalf("язык условий: %s", rules.ExpressionLanguage)
	}
	if len(rules.Conditions.Variables) != len(dp.Variables) {
		t.Fatalf("переменных в rules.yaml %d, в домене %d", len(rules.Conditions.Variables), len(dp.Variables))
	}
	for name, v := range rules.Conditions.Variables {
		d, ok := dp.Variables[name]
		if !ok || d.Type != v.Type || !slices.Equal(d.Values, v.Values) {
			t.Fatalf("переменная %s: rules.yaml %+v, домен %+v", name, v, d)
		}
	}
}
