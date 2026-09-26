package main

import (
	domdocs "ant/internal/domain/documents"
	"reflect"
	"slices"
	"testing"

	"ant/cmd/internal/enginewire"
	engineapp "ant/internal/application/engine"
	processapp "ant/internal/application/process"
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

// Цепочка нормативного слоя изделия у верификатора та же, что у воркера.
func TestVerifierBundlesMatch(t *testing.T) {
	chain := func(v any) []string {
		var out []string
		for rv := reflect.ValueOf(v); rv.IsValid(); {
			for rv.Kind() == reflect.Interface || rv.Kind() == reflect.Pointer {
				if rv.IsNil() {
					return out
				}
				rv = rv.Elem()
			}
			out = append(out, rv.Type().String())
			if rv.Kind() != reflect.Struct {
				break
			}
			rv = rv.FieldByName("Next")
		}
		return out
	}
	c := &core{bundles: &processapp.Bundles{}, codec: &engineapp.Codec{}}
	vb, err := enginewire.Bundles(nil, c.codec, domdocs.VerificationDemo)
	if err != nil {
		t.Fatal(err)
	}
	if a, b := chain(c.bundleSource()), chain(vb); !slices.Equal(a, b) {
		t.Fatalf("нормативный слой верификатора расходится с cmd/ant:\n ant:      %v\n verifier: %v", a, b)
	}
}
