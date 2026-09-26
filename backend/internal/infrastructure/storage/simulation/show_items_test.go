package simulation

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	sim "ant/internal/domain/simulation"
)

// TestShowItemsOneIdentity — у каждого фланца показа SHOW-IS2 одно изделие на
// весь путь: ровно одна регистрация (binds), все решения по нему (перемещение,
// приёмка, сварка, остановки) ссылаются на {item:‹тот же›}, command_id решений
// уникальны и не зависят от номеров шагов (item_id регистрации — из
// command_id: номер шага сдвигается вставкой истории, метка — нет).
func TestShowItemsOneIdentity(t *testing.T) {
	ctx := context.Background()
	f := NewFiles(filepath.Join(repo, "scenarios"))
	b, err := f.Bundle(ctx, "SHOW-IS2")
	if err != nil {
		t.Fatal(err)
	}
	p, err := sim.Generate(b, sim.Params{RunID: GoldenRunID("SHOW-IS2", b.Run.Seed)})
	if err != nil {
		t.Fatal(err)
	}
	binds := map[string]int{}
	cmds := map[string]string{}
	for _, a := range p.Actions {
		if a.Kind != sim.ActionDecision {
			continue
		}
		if a.Binds != "" {
			binds[a.Binds]++
		}
		id := p.IDs.ActionCommandID(a.Label, a.Seq)
		if prev, ok := cmds[id]; ok {
			t.Fatalf("command_id шагов %q и %q совпал", prev, a.Label)
		}
		cmds[id] = a.Label
		// Решение по изделию определения ссылается только на своё изделие.
		if a.Item != "" {
			ref := "{item:" + a.Item + "}"
			for k, v := range a.Params {
				if s, ok := v.(string); ok && strings.HasPrefix(s, "{item:") && s != ref {
					t.Errorf("%s: %s = %s, изделие шага %s", a.Label, k, s, a.Item)
				}
			}
		}
	}
	for _, it := range []string{"F-001", "F-002", "F-003", "F-101", "F-131"} {
		if binds[it] != 1 {
			t.Errorf("регистраций %s: %d", it, binds[it])
		}
	}
	// Метка — основа command_id: регистрация Ф-001 не совпадает ни с одной
	// регистрацией истории при любой нумерации шагов.
	reg := func(item string) string {
		for _, a := range p.Actions {
			if a.Binds == item {
				return p.IDs.ActionCommandID(a.Label, a.Seq)
			}
		}
		return ""
	}
	for i := 101; i <= 131; i++ {
		if reg(fmt.Sprintf("F-%d", i)) == reg("F-001") {
			t.Fatalf("регистрация F-%d = регистрация F-001", i)
		}
	}
	if p.IDs.ActionCommandID("F-001/register", 114) != p.IDs.ActionCommandID("F-001/register", 136) {
		t.Fatal("command_id зависит от номера шага")
	}
}
