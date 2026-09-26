package main

import (
	"slices"
	"testing"

	"ant/cmd/internal/enginewire"
	engineapp "ant/internal/application/engine"
)

// AD-9: верификатор сверяет те же проекции, что пишет воркер: реестр
// cmd/internal/enginewire совпадает с engineRegistry.
func TestVerifierRegistryMatches(t *testing.T) {
	names := func(r *engineapp.Registry) []string {
		var out []string
		for _, p := range r.Items() {
			out = append(out, "item:"+p.Name)
		}
		for _, p := range r.Globals() {
			out = append(out, "global:"+p.Name)
		}
		slices.Sort(out)
		return out
	}
	if a, b := names(engineRegistry()), names(enginewire.Registry()); !slices.Equal(a, b) {
		t.Fatalf("реестр верификатора расходится с cmd/ant:\n ant:      %v\n verifier: %v", a, b)
	}
}
