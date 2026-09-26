package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ant/internal/application/ingest/inmem"
	"ant/internal/application/platform"
	app "ant/internal/application/signing"
	dom "ant/internal/domain/signing"
	"ant/internal/infrastructure/integration/signing/paperscan"
	"ant/internal/infrastructure/security/profiles"
	storesigning "ant/internal/infrastructure/storage/signing"
)

var scenariosDir = filepath.Join("..", "..", "..", "scenarios")

// AD-26: подписывается только запрос, совпадающий с шагом определения
// (операция, персона после сведения имён, буквальные значения).
func TestStepMatch(t *testing.T) {
	st, err := Steps{Root: scenariosDir}.Find(context.Background(), "S02", "analysis.item.assess#1")
	if err != nil {
		t.Fatal(err)
	}
	if st.Actor != "TEC-01" {
		t.Fatalf("персона после сведения имён: %s", st.Actor)
	}
	ok := Request{Operation: "analysis.item.assess", Params: map[string]string{"incident_id": "RS-0a1b2c3d"},
		Body: map[string]any{"assessment": "excluded", "item_id": "ENT01:F-027", "evidence_event_ids": []any{"0192…"}, "command_id": "x"}}
	if err := Match(st, "TEC-01", ok); err != nil {
		t.Fatal(err)
	}
	bad := ok
	bad.Body = map[string]any{"assessment": "confirmed", "item_id": "ENT01:F-027", "evidence_event_ids": []any{"0192…"}}
	if err := Match(st, "TEC-01", bad); !errors.Is(err, ErrMismatch) {
		t.Fatalf("другое решение: %v", err)
	}
	if err := Match(st, "INS-01", ok); !errors.Is(err, ErrMismatch) {
		t.Fatalf("другая персона: %v", err)
	}
	other := ok
	other.Operation = "nonconformity.disposition.set"
	if err := Match(st, "TEC-01", other); !errors.Is(err, ErrMismatch) {
		t.Fatalf("другая операция: %v", err)
	}
	if _, err := (Steps{Root: scenariosDir}).Find(context.Background(), "S02", "нет-такого"); !errors.Is(err, ErrMismatch) {
		t.Fatal(err)
	}
}

// fakeAnt — ant в миниатюре для demo-signer: вход персоной, спецификация
// операций, и приём команды — настоящая проверка подписи модулем signing
// (реестр из файла затравки, который написал demo-signer init).
func fakeAnt(t *testing.T, svc *app.Service, scans map[string][]byte) *httptest.Server {
	spec := map[string]any{"paths": map[string]any{
		"/api/v1/incidents/{incident_id}/items": map[string]any{"post": map[string]any{"operationId": "analysis.item.assess",
			"x-ant-action": map[string]any{"emits": []string{"incident.item.assessed"}, "signature_level": 2, "critical": true}}},
		"/api/v1/documents/{document_id}/paper-signatures": map[string]any{"post": map[string]any{"operationId": AttestOperation,
			"x-ant-action": map[string]any{"emits": []string{"document.signature.recorded"}, "signature_level": 2}}},
		"/api/v1/journal/head":    map[string]any{"get": map[string]any{"operationId": "journal.head.read"}},
		"/api/v1/auth/session":    map[string]any{"get": map[string]any{"operationId": "access.session.read"}},
		"/api/v1/crypto-profiles": map[string]any{"get": map[string]any{"operationId": "signing.profile.list"}},
	}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/auth/session", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			PersonaID string `json:"persona_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		http.SetCookie(w, &http.Cookie{Name: "ant_session", Value: in.PersonaID})
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("GET /api/v1/openapi.json", func(w http.ResponseWriter, _ *http.Request) { _ = json.NewEncoder(w).Encode(spec) })
	mux.HandleFunc("GET /api/v1/journal/head", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"seq":7}`)) })
	mux.HandleFunc("GET /api/v1/auth/session", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"policy_seq":3}`)) })
	mux.HandleFunc("GET /api/v1/crypto-profiles", func(w http.ResponseWriter, r *http.Request) {
		v, _ := svc.Profiles(r.Context(), platform.Moment{})
		_ = json.NewEncoder(w).Encode(v)
	})
	reply := func(w http.ResponseWriter, err error) {
		if err != nil {
			var pe *platform.Error
			errors.As(err, &pe)
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]string{"code": string(pe.Code)})
			return
		}
		_, _ = w.Write([]byte(`{"seq":8}`))
	}
	mux.HandleFunc("POST /api/v1/incidents/{incident_id}/items", func(w http.ResponseWriter, r *http.Request) {
		ck, _ := r.Cookie("ant_session")
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		sig, _ := json.Marshal(body["signature"])
		exp, err := app.ExpectRequest("analysis.item.assess", map[string]string{"incident_id": r.PathValue("incident_id")}, body,
			"incident.item.assessed", body["command_id"].(string), "", ck.Value, 2, true)
		if err != nil {
			t.Fatal(err)
		}
		_, err = svc.CheckCommand(platform.WithPrincipal(r.Context(), platform.Principal{PersonID: ck.Value}), sig, exp)
		reply(w, err)
	})
	mux.HandleFunc("POST /api/v1/documents/{document_id}/paper-signatures", func(w http.ResponseWriter, r *http.Request) {
		ck, _ := r.Cookie("ant_session")
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		sig, _ := json.Marshal(body["signature"])
		exp := app.Expect{Class: dom.ClassDocumentSignature, DocumentID: r.PathValue("document_id"), DocDigest: dom.Digest([]byte("акт о браке")),
			Stage: 1, Signer: "INS-01", PaperAllowed: true, Actor: ck.Value, Level: 2}
		_, err := svc.CheckCommand(r.Context(), sig, exp)
		reply(w, err)
	})
	return httptest.NewServer(mux)
}

type scanStore map[string][]byte

func (s scanStore) Scan(_ context.Context, a string) ([]byte, error) {
	if b, ok := s[a]; ok {
		return b, nil
	}
	return nil, errors.New("нет")
}

// AD-26, AD-33: demo-signer — обычный клиент: init пишет ключи и затравку
// реестра; шаг сценария подписывается ключом персоны и принимается ant той
// же проверкой, что подпись агента токена (метод demo_signer, класс
// scenario); заверение бумаги персоной-заверителем — с настоящим QR на скане.
func TestEndToEnd(t *testing.T) {
	dir := t.TempDir()
	set, err := EnsureKeys(filepath.Join(dir, "keys"), fill(append([]Persona(nil), DefaultPersonas...)))
	if err != nil {
		t.Fatal(err)
	}
	boot := filepath.Join(dir, "bootstrap.json")
	if err := WriteBootstrap(boot, set); err != nil {
		t.Fatal(err)
	}
	regs, err := storesigning.LoadBootstrap(boot)
	if err != nil || len(regs) != 2*len(DefaultPersonas) {
		t.Fatalf("%d %v", len(regs), err)
	}
	clock := inmem.NewClock(time.Date(2026, 9, 23, 14, 25, 0, 0, time.UTC))
	j := inmem.NewJournal(clock.At)
	scans := scanStore{}
	svc := app.NewService(app.WithConfig(app.Config{Profile: "demo"}),
		app.WithDeps(app.Deps{Journal: j, Registry: app.NewRegistry(j, regs, nil), Authorities: app.DefaultAuthorities(),
			Scans: scans, QR: paperscan.Reader{}, Now: clock.At}))
	svc.UseCrypto(profiles.Verifier{Keys: svc.Registry()})
	ant := fakeAnt(t, svc, scans)
	defer ant.Close()
	s := &Signer{Keys: set, Steps: Steps{Root: scenariosDir}, Ant: &Client{Base: ant.URL, HTTP: ant.Client()}}
	rq := StepRequest{RunID: "ms-1-demo", ScenarioID: "S02", StepID: "analysis.item.assess#1", PersonaID: "TEC-01",
		Request: &Request{Operation: "analysis.item.assess", Params: map[string]string{"incident_id": "RS-0a1b2c3d"},
			Body: map[string]any{"assessment": "excluded", "item_id": "ENT01:F-027", "evidence_event_ids": []any{"01929a2b-7c3d-7e4f-8a5b-6c7d8e9f0a1b"}}}}
	res, err := s.Execute(context.Background(), rq)
	if err != nil || res.Status != "signed" || res.Detail != "" || !strings.HasPrefix(res.DocDigest, dom.HashPrefix) {
		t.Fatalf("шаг: %+v %v", res, err)
	}
	// Запрос не совпал с шагом — не подписывается и не отправляется.
	rq.Request.Body = map[string]any{"assessment": "confirmed", "item_id": "ENT01:F-027", "evidence_event_ids": []any{"x"}}
	if res, err := s.Execute(context.Background(), rq); !errors.Is(err, ErrMismatch) || res.Status != "mismatch" {
		t.Fatalf("несовпадение: %+v %v", res, err)
	}
	// Заверение бумаги: настоящий QR на скане. Шаг attest_paper — служебный
	// шаг сценария, проверяем подпись заверения напрямую.
	digest := dom.Digest([]byte("акт о браке"))
	png, _ := paperscan.PNG(dom.QRText("DOC-7", digest), 300)
	scans[dom.Digest(png)] = png
	body := map[string]any{"command_id": "c1", "version": json.Number("1"), "doc_digest": digest, "stage": json.Number("1"),
		"signer_person_id": "INS-01", "scan_address": dom.Digest(png), "paper_original_no": "ОТК-2026-0042"}
	p, _ := set.Persona("FOR-WC")
	areq := StepRequest{PersonaID: "FOR-WC", Request: &Request{Operation: AttestOperation, Params: map[string]string{"document_id": "DOC-7"}, Body: body}}
	r, err := s.attest(p, areq, body)
	if err != nil || r.Status != "attested" {
		t.Fatal(err)
	}
	st, out, err := s.Ant.Do(context.Background(), "FOR-WC", AttestOperation, areq.Request.Params, body)
	if err != nil || st != http.StatusOK {
		t.Fatalf("заверение: %d %s %v", st, out, err)
	}
	// Скан с чужим QR.
	png2, _ := paperscan.PNG(dom.QRText("DOC-8", dom.Digest([]byte("другой"))), 300)
	scans[dom.Digest(png2)] = png2
	body2 := map[string]any{}
	for k, v := range body {
		body2[k] = v
	}
	body2["scan_address"] = dom.Digest(png2)
	_, _ = s.attest(p, areq, body2)
	st, out, _ = s.Ant.Do(context.Background(), "FOR-WC", AttestOperation, areq.Request.Params, body2)
	if st != http.StatusUnprocessableEntity || !strings.Contains(string(out), "signing.qr_mismatch") {
		t.Fatalf("чужой QR: %d %s", st, out)
	}
	// Заверитель = подписант — demo-signer не подписывает сам.
	pi, _ := set.Persona("INS-01")
	if _, err := s.attest(pi, areq, body); !errors.Is(err, ErrMismatch) {
		t.Fatal("заверитель = подписант")
	}
}
