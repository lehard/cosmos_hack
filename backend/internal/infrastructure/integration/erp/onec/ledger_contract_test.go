package onec_test

import (
	"encoding/json"
	"errors"
	"testing"

	app "ant/internal/application/erp"
	"ant/internal/application/erp/ledgertest"
	appingest "ant/internal/application/ingest"
	domingest "ant/internal/domain/ingest"
)

// Общий контрактный тест порта учёта (эпик 31, AD-35): тот же внутренний
// сигнал, что принимает адаптер Галактики, принимает и адаптер 1С со своим
// stand-ом; входящие факты шлюза проходят схемы приёма.
func TestLedgerContract(t *testing.T) {
	e := setup(t)
	ledgertest.Run(t, e.client, ledgertest.Options{System: "onec", Envelope: func(f app.Inbound) error {
		raw, err := app.Envelope(app.GatewaySource("onec"), app.GatewayKey("onec"), 1, f)
		if err != nil {
			return err
		}
		var d struct {
			Payload []byte `json:"payload"`
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return err
		}
		dec, err := appingest.ValidateEnvelope(d.Payload)
		if err != nil {
			return err
		}
		if dec.Outcome != domingest.OutcomeAccepted {
			return errors.New(string(dec.Code) + " " + dec.Field + " " + dec.Detail)
		}
		return nil
	}})
}
