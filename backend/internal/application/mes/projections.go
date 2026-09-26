package mes

import (
	"encoding/json"
	"slices"

	engineapp "ant/internal/application/engine"
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/mes"
)

// Проекции модуля mes (AD-45, писатель — mes; роль projector).
const (
	// ProjectionBlock — блок или снятие в MES по бизнес-ключу (dom.Block).
	ProjectionBlock = "mes.block"
	// ProjectionBlockIndex — бизнес-ключи блоков в порядке формирования (ключ all).
	ProjectionBlockIndex = "mes.block_index"
	// ProjectionJob — задание MES по job_id.
	ProjectionJob = "mes.job"
	// ProjectionJobIndex — задания в порядке получения (ключ all).
	ProjectionJobIndex = "mes.job_index"
	// IndexLimit — сколько последних записей держит индекс.
	IndexLimit = 2000
)

// Projections — проекции модуля mes для реестра движка.
func Projections() []engineapp.GlobalProjection {
	return []engineapp.GlobalProjection{
		{Name: ProjectionBlock, Writer: dom.Module, Keys: blockKeys, Step: blockStep},
		{Name: ProjectionBlockIndex, Writer: dom.Module, Keys: indexOf(blockOf), Step: indexStep(blockOf)},
		{Name: ProjectionJob, Writer: dom.Module, Keys: jobKeys, Step: jobStep},
		{Name: ProjectionJobIndex, Writer: dom.Module, Keys: indexOf(jobOf), Step: indexStep(jobOf)},
	}
}

// MustRegister регистрирует проекции mes в реестре движка (cmd/ant, engineRegistry).
func MustRegister(r *engineapp.Registry) {
	for _, p := range Projections() {
		if err := r.AddGlobal(p); err != nil {
			panic(err)
		}
	}
}

func blockOf(r kernel.Record) string {
	if r.Type != catalog.MesHoldRequested {
		return ""
	}
	return dom.BlockKey(r)
}

func jobOf(r kernel.Record) string {
	if r.Type != catalog.MesJobReceived {
		return ""
	}
	var d struct {
		JobID string `json:"job_id"`
	}
	if json.Unmarshal(r.Data, &d) != nil {
		return ""
	}
	return d.JobID
}

func blockKeys(r kernel.Record) []string {
	if k := dom.BlockKey(r); k != "" {
		return []string{k}
	}
	return nil
}

func blockStep(_ string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	var b dom.Block
	if len(prev) > 0 {
		if err := json.Unmarshal(prev, &b); err != nil {
			return nil, err
		}
	}
	b, err := b.Apply(r)
	if err != nil {
		return nil, err
	}
	return json.Marshal(b)
}

func jobKeys(r kernel.Record) []string {
	if k := jobOf(r); k != "" {
		return []string{k}
	}
	return nil
}

func jobStep(_ string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	if len(prev) > 0 {
		return prev, nil
	}
	var d ev.MesJobReceivedV1
	if err := json.Unmarshal(r.Data, &d); err != nil {
		return nil, err
	}
	return json.Marshal(dom.JobView{Data: d, ReceivedAt: r.OccurredAt, Seq: r.Seq})
}

func indexOf(key func(kernel.Record) string) func(kernel.Record) []string {
	return func(r kernel.Record) []string {
		if key(r) != "" {
			return []string{"all"}
		}
		return nil
	}
}

func indexStep(key func(kernel.Record) string) func(string, json.RawMessage, kernel.Record) (json.RawMessage, error) {
	return func(_ string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
		var ks []string
		if len(prev) > 0 {
			if err := json.Unmarshal(prev, &ks); err != nil {
				return nil, err
			}
		}
		if k := key(r); !slices.Contains(ks, k) {
			ks = append(ks, k)
		}
		if len(ks) > IndexLimit {
			ks = ks[len(ks)-IndexLimit:]
		}
		return json.Marshal(ks)
	}
}
