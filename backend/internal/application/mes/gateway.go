package mes

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	engineapp "ant/internal/application/engine"
	"ant/internal/contracts/catalog"
	"ant/internal/domain/engine"
	dom "ant/internal/domain/mes"
)

// GatewaySource — source_id шлюза MES.
func GatewaySource(system string) string { return "mes." + system }

// Gateway — шлюз входящих MES (AD-18: входящие — через обычный приём):
// задания → mes.job.received, события операций → operation.run.* изделия.
// Факты — вида «внешняя система», класс server-attested (AD-2); event_id —
// от номера сообщения MES: повторная доставка — дубль (AD-7). Сообщение без
// соответствия (изделие, шаг) откладывается с причиной — факт не
// выдумывается (FR-123).
type Gateway struct {
	Channel Channel
	Intake  Intake
	// Env — соответствия и шаги процесса на момент опроса (nil — пустые:
	// тогда только задания и события с нашим ID изделия).
	Env func(ctx context.Context) (dom.Env, error)
	Log *slog.Logger
}

// Pulled — итог опроса.
type Pulled struct {
	Submitted int
	Deferred  []dom.Deferred
}

// PullOnce — один опрос MES: новые факты — в приём.
func (g *Gateway) PullOnce(ctx context.Context) (Pulled, error) {
	var out Pulled
	in, err := g.Channel.Pull(ctx)
	if err != nil {
		return out, err
	}
	env := dom.Env{}
	if g.Env != nil {
		if env, err = g.Env(ctx); err != nil {
			return out, err
		}
	}
	out.Deferred = append(out.Deferred, in.Rejected...)
	var facts []dom.Fact
	for _, j := range in.Jobs {
		facts = append(facts, dom.JobFacts(j)...)
	}
	for _, e := range in.Events {
		fs, def := dom.EventFacts(env, e)
		if def != nil {
			out.Deferred = append(out.Deferred, *def)
			continue
		}
		facts = append(facts, fs...)
	}
	if g.Log != nil {
		for _, d := range out.Deferred {
			g.Log.Warn("MES: сообщение отложено", "message_id", d.MessageID, "ref", d.Ref, "reason", d.Reason)
		}
	}
	if len(facts) == 0 {
		return out, nil
	}
	src := GatewaySource(g.Channel.Info().System)
	envs := make([][]byte, 0, len(facts))
	for _, f := range facts {
		b, err := Envelope(src, "gateway-mes@1", f)
		if err != nil {
			return out, err
		}
		envs = append(envs, b)
	}
	if err := g.Intake.Submit(ctx, src, envs); err != nil {
		return out, err
	}
	out.Submitted = len(facts)
	return out, nil
}

// Envelope — конверт DSSE факта шлюза MES (events/common/envelope.v1):
// источник — внешняя система, изделие — наш item_id по соответствию; демо без
// подписи (Д-28).
func Envelope(source, keyRef string, f dom.Fact) ([]byte, error) {
	info, ok := catalog.Lookup(f.Type)
	if !ok {
		return nil, fmt.Errorf("mes: тип %q не из каталога", f.Type)
	}
	data, err := json.Marshal(f.Data)
	if err != nil {
		return nil, err
	}
	env := map[string]any{
		"event_id": f.EventID, "event_type": string(f.Type), "schema_version": info.CurrentVersion,
		"source_id": source, "source_kind": "external_system", "reliability": "high",
		"occurred_at": engineapp.FormatTime(f.OccurredAt), "correlation_id": f.EventID, "causation_id": nil,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{strings.ToLower(keyRef)}},
		"data":      json.RawMessage(data),
	}
	if f.ItemID != "" {
		env["item_id"] = f.ItemID
	}
	payload, err := engine.Canonical(env)
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{"payloadType": engineapp.PayloadTypeEvent,
		"payload": base64.StdEncoding.EncodeToString(payload), "signatures": []any{}})
}

// StepsFromBPMN — код операции → step_key из версии процесса (BPMN
// `ant:properties/@operationCode`): по нему событие операции MES попадает
// на шаг процесса (AD-17).
func StepsFromBPMN(src []byte) (map[string]string, error) {
	steps := map[string]string{}
	dec := xml.NewDecoder(bytes.NewReader(src))
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("mes: BPMN: %w", err)
		}
		if t, ok := tok.(xml.StartElement); ok && t.Name.Local == "properties" {
			var key, code string
			for _, a := range t.Attr {
				switch a.Name.Local {
				case "stepKey":
					key = a.Value
				case "operationCode":
					code = a.Value
				}
			}
			if key != "" && code != "" {
				if _, dup := steps[code]; !dup {
					steps[code] = key
				}
			}
		}
	}
	return steps, nil
}
