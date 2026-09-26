package cad

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/contracts/catalog"
	"ant/internal/domain/engine"
)

// Fact — факт импорта для приёма: тип, детерминированный event_id, время и data.
type Fact struct {
	Type       catalog.Type
	EventID    string
	OccurredAt time.Time
	Data       any
}

// Envelope — конверт DSSE факта импорта (контракт events/common/envelope.v1):
// источник вида «импорт» (AD-2, FR-140), без source_seq (импорт файла — не
// поток устройства); демо без подписи — пустой список подписей (Д-28),
// подписант — загрузивший файл (`‹псевдоним›@1`) или ключ шлюза.
func Envelope(source, signer string, f Fact) ([]byte, error) {
	info, _ := catalog.Lookup(f.Type)
	data, err := json.Marshal(f.Data)
	if err != nil {
		return nil, err
	}
	if signer == "" {
		signer = "gateway-kompas@1"
	}
	env := map[string]any{
		"event_id": f.EventID, "event_type": string(f.Type), "schema_version": info.CurrentVersion,
		"source_id": source, "source_kind": "import", "reliability": "high",
		"occurred_at": engineapp.FormatTime(f.OccurredAt), "correlation_id": f.EventID, "causation_id": nil,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{strings.ToLower(signer)}},
		"data":      json.RawMessage(data),
	}
	payload, err := engine.Canonical(env)
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{"payloadType": engineapp.PayloadTypeEvent,
		"payload": base64.StdEncoding.EncodeToString(payload), "signatures": []any{}})
}
