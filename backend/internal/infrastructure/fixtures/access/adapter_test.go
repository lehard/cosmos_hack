package access

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"

	app "ant/internal/application/access"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	"ant/internal/infrastructure/fixtures/loader"
)

const common = `format: 1
default_scenario: sc
responses:
  - op: access.persona.list
    body: {"items": [{"id": "INS-01", "name": "Контролёр ОТК 1", "role": {"id": "quality_inspector", "title": "Контролёр качества"}}]}
  - op: access.session.read
    params: {persona: INS-01}
    body: {"user": {"id": "INS-01", "name": "Контролёр ОТК 1"}, "role": {"id": "quality_inspector", "title": "Контролёр качества"}, "policy_seq": 0, "demo": true}
  - op: access.desk.read
    params: {role: quality_inspector}
    body: {"version": 1, "role": "quality_inspector", "title_key": "desks.decisionQueue", "density": "comfortable", "tabs": [{"id": "queue", "title_key": "x", "layout": "single", "slots": []}]}
`

const manifest = `format: 1
id: sc
title: Тест
initial_step: 0
enterprise: ENT01
local_ids: ['F-\d{3}']
steps:
  - {step: 0, clock: 2026-09-18T05:00:00Z, title: "шаг 0"}
`

func setup(t *testing.T) {
	t.Helper()
	fsys := fstest.MapFS{
		"common/responses.yaml": {Data: []byte(common)},
		"sc/scenario.yaml":      {Data: []byte(manifest)},
		"sc/steps/00.yaml":      {Data: []byte("format: 1\nstep: 0\nclock: 2026-09-18T05:00:00Z\ntitle: шаг 0\n")},
	}
	lib, err := loader.LoadFS(fsys, ".")
	if err != nil {
		t.Fatal(err)
	}
	rt := loader.New(lib, loader.NewMemoryCursor())
	old := runtime
	runtime = func() (*loader.Runtime, error) { return rt, nil }
	t.Cleanup(func() { runtime = old })
}

func TestSessionAndDesk(t *testing.T) {
	setup(t)
	a := New()
	ctx := context.Background()
	if _, err := a.Session(ctx); !isCode(err, errcodes.AccessUnauthenticated) {
		t.Fatalf("анонимный сеанс: %v", err)
	}
	if _, _, err := a.OpenSession(ctx, app.SessionCreate{}); !isCode(err, errcodes.AccessLoginFailed) {
		t.Fatalf("вход без персоны: %v", err)
	}
	s, token, err := a.OpenSession(ctx, app.SessionCreate{PersonaID: "INS-01"})
	if err != nil || token != "demo.INS-01" || s.Role.ID != "quality_inspector" {
		t.Fatalf("вход: %+v %q %v", s, token, err)
	}
	ctx = platform.WithPrincipal(ctx, platform.Principal{PersonID: "INS-01", Role: "quality_inspector"})
	if s, err := a.Session(ctx); err != nil || s.User.ID != "INS-01" {
		t.Fatalf("сеанс: %+v %v", s, err)
	}
	if d, err := a.Desks(ctx); err != nil || d.Role != "quality_inspector" {
		t.Fatalf("стол: %+v %v", d, err)
	}
	if p, err := a.Personas(context.Background()); err != nil || len(p.Items) != 1 {
		t.Fatalf("персоны: %+v %v", p, err)
	}
}

func isCode(err error, code errcodes.Code) bool {
	var e *platform.Error
	return errors.As(err, &e) && e.Code == code
}
