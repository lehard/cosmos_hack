package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
	"uuid"

	dom "ant/internal/domain/signing"
	"ant/internal/infrastructure/security/profiles"
)

// Signer — выполнение шагов: сверка с определением, отпечаток, подпись,
// обычный вызов API.
type Signer struct {
	Keys  *KeySet
	Steps Steps
	Ant   *Client
	// Now — часы клиента (client_signed_at, в порядок журнала не входит — AD-37).
	Now func() time.Time
}

// AttestOperation — операция заверения бумажной подписи (эпик 28, AD-43):
// шаг attest_paper сценария — заверитель-персона подписывает заверение.
const AttestOperation = "documents.paper.attest"

func (s *Signer) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// Execute — шаг сценария от имени персоны.
func (s *Signer) Execute(ctx context.Context, rq StepRequest) (StepResult, error) {
	p, ok := s.Keys.Persona(rq.PersonaID)
	if !ok {
		return StepResult{Status: "mismatch", Detail: "персоны " + rq.PersonaID + " нет"}, ErrMismatch
	}
	if rq.Request == nil {
		return StepResult{Status: "skipped", Detail: "в запросе нет request — нечего подписывать"}, nil
	}
	st, err := s.Steps.Find(ctx, rq.ScenarioID, rq.StepID)
	if err == nil {
		err = Match(st, p.PersonaID, *rq.Request)
	}
	if err != nil {
		return StepResult{Status: "mismatch", Detail: err.Error()}, err
	}
	body, err := s.header(ctx, p.PersonaID, rq)
	if err != nil {
		return StepResult{Status: "skipped", Detail: err.Error()}, err
	}
	var res StepResult
	if rq.Request.Operation == AttestOperation {
		res, err = s.attest(p, rq, body)
	} else {
		res, err = s.signCommand(ctx, p, rq, body)
	}
	if err != nil {
		return res, err
	}
	status, out, err := s.Ant.Do(ctx, p.PersonaID, rq.Request.Operation, rq.Request.Params, body)
	if err != nil {
		return StepResult{Status: "skipped", Detail: err.Error()}, err
	}
	if status/100 != 2 {
		res.Detail = fmt.Sprintf("ant ответил %d: %s", status, string(out))
		return res, nil
	}
	return res, nil
}

// header — заголовок команды (AD-7, AD-39): command_id, basis_seq, policy_seq
// — как у стола роли, если шаг их не задал.
func (s *Signer) header(ctx context.Context, persona string, rq StepRequest) (map[string]any, error) {
	body := map[string]any{}
	for k, v := range rq.Request.Body {
		body[k] = v
	}
	if _, ok := body["command_id"]; !ok {
		body["command_id"] = uuid.NewV7().String()
	}
	if v, ok := body["basis_seq"]; !ok || fmt.Sprint(v) == "0" {
		if h, err := s.Ant.Read(ctx, persona, "journal.head.read", nil); err == nil && h["seq"] != nil {
			body["basis_seq"] = h["seq"]
		} else {
			body["basis_seq"] = 0
		}
	}
	if v, ok := body["policy_seq"]; !ok || fmt.Sprint(v) == "0" {
		if ss, err := s.Ant.Read(ctx, persona, "access.session.read", nil); err == nil && ss["policy_seq"] != nil {
			body["policy_seq"] = ss["policy_seq"]
		} else {
			body["policy_seq"] = 0
		}
	}
	return body, nil
}

// required — обязательный профиль пакетов класса event на ant (смена профиля, AD-32).
func (s *Signer) required(ctx context.Context, persona, class string) string {
	m, err := s.Ant.Read(ctx, persona, "signing.profile.list", nil)
	if err != nil {
		return dom.ProfileGost
	}
	b, _ := json.Marshal(m["items"])
	var items []struct {
		ProfileID     string   `json:"profile_id"`
		ObjectClasses []string `json:"object_classes"`
	}
	_ = json.Unmarshal(b, &items)
	for _, it := range items {
		for _, c := range it.ObjectClasses {
			if c == class {
				return it.ProfileID
			}
		}
	}
	return dom.ProfileGost
}

// signCommand — событие-команда по соглашению «подписан запрос»
// (domain/signing.RequestData) ключом персоны; отпечаток считает сам.
func (s *Signer) signCommand(ctx context.Context, p Persona, rq StepRequest, body map[string]any) (StepResult, error) {
	op, err := s.Ant.Op(ctx, rq.Request.Operation)
	if err != nil {
		return StepResult{Status: "skipped", Detail: err.Error()}, err
	}
	if op.SignatureLevel == 0 || len(op.Emits) == 0 {
		return StepResult{Status: "skipped", Detail: "операция без уровня подписи — отправлена как есть"}, nil
	}
	data, err := dom.RequestData(rq.Request.Operation, rq.Request.Params, body)
	if err != nil {
		return StepResult{Status: "skipped", Detail: err.Error()}, err
	}
	refs := []string{GostRef(p.PersonaID)}
	profile := s.required(ctx, p.PersonaID, dom.ClassEvent)
	if profile == dom.ProfileHybrid {
		refs = append(refs, PQRef(p.PersonaID))
	}
	level := op.SignatureLevel
	if level > dom.Level2 {
		level = dom.Level2
	}
	ce := dom.CommandEvent{EventType: op.Emits[0], CommandID: fmt.Sprint(body["command_id"]), ItemID: rq.Request.Params["item_id"],
		RunID: rq.RunID, SourceID: p.PersonaID, Data: data, Level: level, ClientSignedAt: s.now().Format("2006-01-02T15:04:05.000Z"),
		Profile: profile, Signers: refs}
	c, err := ce.Canonical()
	if err != nil {
		return StepResult{Status: "skipped", Detail: err.Error()}, err
	}
	env, err := profiles.Signer{Keys: s.Keys.Ring}.Envelope(dom.PayloadType(dom.ClassEvent, 1), c, refs...)
	if err != nil {
		return StepResult{Status: "skipped", Detail: err.Error()}, err
	}
	body["signature"] = env
	return StepResult{Status: "signed", CommandID: ce.CommandID, DocDigest: dom.Digest(c)}, nil
}

// attest — заверение бумажной подписи персоной-заверителем (AD-33, шаг
// attest_paper): подпись уровня 2 класса paper-attestation над документом,
// подписантом, сканом и учётным номером оригинала.
func (s *Signer) attest(p Persona, rq StepRequest, body map[string]any) (StepResult, error) {
	str := func(k string) string { v, _ := body[k].(string); return v }
	num := func(k string) int {
		switch v := body[k].(type) {
		case json.Number:
			n, _ := v.Int64()
			return int(n)
		case float64:
			return int(v)
		case int:
			return v
		}
		return 0
	}
	pa := dom.PaperAttestation{FormatVersion: 1, CryptoProfile: dom.ProfileGost, Signers: []string{GostRef(p.PersonaID)},
		DocumentID: rq.Request.Params["document_id"], Version: num("version"), DocDigest: str("doc_digest"), Stage: num("stage"),
		SignerPersonID: str("signer_person_id"), AttestedBy: p.PersonaID, ScanAddress: str("scan_address"),
		PaperOriginalNo: str("paper_original_no"), SignatureLevel: dom.Level2, ClientSignedAt: s.now().Format("2006-01-02T15:04:05.000Z")}
	if pa.SignerPersonID == p.PersonaID {
		return StepResult{Status: "mismatch", Detail: "заверитель = подписант (AD-43)"}, fmt.Errorf("%w: заверитель = подписант", ErrMismatch)
	}
	c, err := dom.CanonicalOf(pa)
	if err != nil {
		return StepResult{}, err
	}
	env, err := profiles.Signer{Keys: s.Keys.Ring}.Envelope(dom.PayloadType(dom.ClassPaperAttestation, 1), c, GostRef(p.PersonaID))
	if err != nil {
		return StepResult{}, err
	}
	body["signature"] = env
	return StepResult{Status: "attested", CommandID: fmt.Sprint(body["command_id"]), DocDigest: pa.DocDigest}, nil
}

// Handler — API шагов (contracts/internal/demo-signer.openapi.yaml).
func (s *Signer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /v1/personas", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.Keys.Personas)
	})
	mux.HandleFunc("POST /v1/steps", func(w http.ResponseWriter, r *http.Request) {
		var rq StepRequest
		dec := json.NewDecoder(r.Body)
		dec.UseNumber()
		if err := dec.Decode(&rq); err != nil {
			problem(w, http.StatusBadRequest, "api.validation_failed", err.Error())
			return
		}
		res, err := s.Execute(r.Context(), rq)
		if errors.Is(err, ErrMismatch) {
			problem(w, http.StatusConflict, "signing.document_changed", res.Detail)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	})
	return mux
}

func problem(w http.ResponseWriter, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "urn:ant:problem:" + code, "title": "Шаг не подписан", "status": status,
		"code": code, "detail": detail})
}
