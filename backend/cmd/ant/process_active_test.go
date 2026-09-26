package main

import (
	"bytes"
	"context"
	"io/fs"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	itemapp "ant/internal/application/item"
	"ant/internal/application/item/itemtest"
	"ant/internal/application/platform"
	processapp "ant/internal/application/process"
	dj "ant/internal/domain/journal"
	dp "ant/internal/domain/process"
	"ant/internal/infrastructure/fixtures/world"
)

// Хвост эпика 39 (FR-22, AD-17): после ввода в действие новой версии процесса
// новое изделие закрепляет её хеш, а изделие, запущенное раньше, остаётся на
// своей версии. Адаптер activeProcess — над хранилищем версий process.
func TestNewItemsFollowActiveProcessVersion(t *testing.T) {
	ctx := itemtest.Principal(context.Background(), "qc-7")
	seedXML, err := fs.ReadFile(world.Inputs(), processSeedFile)
	if err != nil {
		t.Fatal(err)
	}
	versions := &processapp.MemVersions{}
	t0 := time.Date(2026, 9, 21, 5, 0, 0, 0, time.UTC)
	seed, err := processapp.EnsureSeed(ctx, versions, seedXML, t0)
	if err != nil {
		t.Fatal(err)
	}
	active := &activeProcess{store: versions}

	j := enginemem.New(nil)
	codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: dj.ZeroLink.String(), Partitions: 4}
	w := itemtest.New(j, codec, j, "active-process")
	w.Item = itemapp.NewLive(itemapp.Config{Codec: codec, Projections: j, Bundles: itemapp.Bundles{Env: itemtest.Env},
		Writer: itemapp.JournalWriter{Journal: j, DomainBuild: codec.DomainBuild, Partitions: codec.Partitions}, Clock: w.Clock, Env: itemtest.Env,
		ProcessVersion: "streebog256:" + string(bytes.Repeat([]byte("0"), 64)), ActiveProcess: active})

	register := func(local string) string {
		t.Helper()
		if _, err := w.Item.Register(ctx, itemapp.RegisterItem{CommandHeader: w.Cmd(), LocalID: local, ItemTypeID: "FL-100.00.000", ItemRevision: "Б"}); err != nil {
			t.Fatal(err)
		}
		if err := w.Settle(ctx); err != nil {
			t.Fatal(err)
		}
		return "ENT01:" + local
	}
	version := func(id string) string {
		t.Helper()
		p, err := w.Item.Passport(ctx, id, platform.Moment{})
		if err != nil {
			t.Fatal(err)
		}
		return p.ProcessVersion
	}

	old := register("F-901")
	if got := version(old); got != seed.Hash {
		t.Fatalf("до новой версии: %s, ждали стартовую %s", got, seed.Hash)
	}

	// Новая версия процесса утверждена и введена в действие (как ActivateVersion:
	// новая — active, прежняя — retired).
	xml2 := append(bytes.Clone(seedXML), []byte("\n<!-- v2: норма очереди сварки -->\n")...)
	at := t0.Add(48 * time.Hour)
	v2 := processapp.VersionRecord{ID: "flange-2", Label: "v2", Status: dp.StatusDraft, BaseVersionID: seed.ID, Author: "TEC-01",
		Hash: dp.VersionHash(xml2), XML: xml2, CreatedAt: at}
	if err := versions.Save(ctx, v2); err != nil {
		t.Fatal(err)
	}
	v2.Status, v2.EffectiveFrom = dp.StatusActive, &at
	if err := versions.Update(ctx, v2); err != nil {
		t.Fatal(err)
	}
	seed.Status = dp.StatusRetired
	if err := versions.Update(ctx, seed); err != nil {
		t.Fatal(err)
	}

	fresh := register("F-902")
	if got := version(fresh); got != v2.Hash {
		t.Fatalf("новое изделие после ввода v2: %s, ждали %s", got, v2.Hash)
	}
	if got := version(old); got != seed.Hash {
		t.Fatalf("изделие в работе сменило версию: %s, ждали %s", got, seed.Hash)
	}

	// Имена узлов — из действующей версии (UI-21).
	names, err := active.StepNames(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if names["welding.weld"] != "Сварка фланца с патрубком" {
		t.Fatalf("имя узла welding.weld: %q", names["welding.weld"])
	}
}
