package signing

import "encoding/json"

// CommandEvent — событие-команда, которое подписывает человек (агент токена,
// demo-signer): «payload_b64 — событие-команда» протокола агента (AD-14,
// contracts/internal/token-agent). Сервер сверяет тип, command_id, изделие и
// data с исполняемой командой; occurred_at под подписью — время клиента
// (client_signed_at), в порядок журнала оно не входит (AD-37).
type CommandEvent struct {
	EventType      string
	SchemaVersion  int
	CommandID      string
	ItemID         string
	RunID          string
	SourceID       string
	Data           json.RawMessage
	Level          int
	BasisSeq       int64
	PolicySeq      int64
	GuardStreams   []string
	WorkplaceID    string
	SeenCheckpoint int64
	ClientSignedAt string
	Profile        string
	Signers        []string
}

// Canonical — канонический JSON события-команды (RFC 8785).
func (c CommandEvent) Canonical() ([]byte, error) {
	if c.SchemaVersion == 0 {
		c.SchemaVersion = 1
	}
	if c.Profile == "" {
		c.Profile = ProfileGost
	}
	gs := c.GuardStreams
	if gs == nil {
		gs = []string{}
	}
	cmd := map[string]any{"command_id": c.CommandID, "basis_seq": c.BasisSeq, "guard_streams": gs, "policy_seq": c.PolicySeq,
		"signature_level": c.Level}
	if c.WorkplaceID != "" {
		cmd["workplace_id"] = c.WorkplaceID
	}
	if c.SeenCheckpoint > 0 {
		cmd["seen_checkpoint"] = c.SeenCheckpoint
	}
	if c.ClientSignedAt != "" {
		cmd["client_signed_at"] = c.ClientSignedAt
	}
	ev := map[string]any{"event_id": c.CommandID, "event_type": c.EventType, "schema_version": c.SchemaVersion,
		"source_id": c.SourceID, "source_kind": "manual_entry", "occurred_at": c.ClientSignedAt, "correlation_id": c.CommandID,
		"causation_id": nil, "command": cmd,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": c.Profile, "signers": c.Signers},
		"data":      c.Data}
	if c.ItemID != "" {
		ev["item_id"] = c.ItemID
	}
	if c.RunID != "" {
		ev["run_id"] = c.RunID
	}
	return CanonicalOf(ev)
}

// HeaderFields — поля заголовка команды (AD-7, AD-39): в подписываемое
// содержимое запроса не входят — их ставит клиент в момент отправки, а
// подпись — в поле signature.
var HeaderFields = []string{"command_id", "basis_seq", "policy_seq", "workplace_id", "signature"}

// RequestData — data события-команды по соглашению подписи запроса: операция,
// параметры пути и тело без полей заголовка. Одно и то же вычисляют клиент
// (агент токена, demo-signer) и сервер (Expect.Data) — подписанное совпадает
// с исполняемым побайтно после JCS.
func RequestData(operation string, params map[string]string, body map[string]any) (json.RawMessage, error) {
	b := make(map[string]any, len(body))
	for k, v := range body {
		b[k] = v
	}
	for _, h := range HeaderFields {
		delete(b, h)
	}
	if params == nil {
		params = map[string]string{}
	}
	return CanonicalOf(map[string]any{"operation": operation, "params": params, "body": b})
}
