package engine

import (
	"maps"
	"slices"
	"strings"

	"ant/internal/domain/analysis"
	"ant/internal/domain/documents"
	"ant/internal/domain/item"
	"ant/internal/domain/kernel"
	"ant/internal/domain/machinelogs"
	"ant/internal/domain/nonconformity"
	"ant/internal/domain/notifications"
	"ant/internal/domain/process"
	"ant/internal/domain/quality"
	"ant/internal/domain/vision"
)

// Order — фиксированный порядок композиции свёртки изделия (AD-40).
var Order = []kernel.Module{
	item.Module, process.Module, vision.Module, quality.Module, machinelogs.Module,
	documents.Module, nonconformity.Module, analysis.Module, notifications.Module,
}

// Snapshot — состояние всех модулей композиции для одного изделия после
// свёртки его входа (AD-5).
type Snapshot struct {
	Item          item.State
	Process       process.State
	Vision        vision.State
	Quality       quality.State
	Machinelogs   machinelogs.State
	Documents     documents.State
	Nonconformity nonconformity.State
	Analysis      analysis.State
	Notifications notifications.State
	// BasisSeq — наибольший seq свёрнутого входа (AD-39).
	BasisSeq int64
}

// Bundle — закреплённый при запуске изделия нормативный слой, разложенный по
// модулям (AD-17): версия процесса, план контроля, карта реакций, шаблоны…
// Собирает application/engine из версии нормативного слоя.
type Bundle struct {
	Item          item.Env
	Process       process.Env
	Vision        vision.Env
	Quality       quality.Env
	Machinelogs   machinelogs.Env
	Documents     documents.Env
	Nonconformity nonconformity.Env
	Analysis      analysis.Env
	Notifications notifications.Env
}

// Fold — свёртка входа изделия целиком (AD-5): версия (Bundle) + вход →
// состояние + реакции. Вход сортируется по occurred_at → received_at →
// event_id; реакции в свёртку не входят (AD-3). Одинаково вызывают воркер,
// воспроизведение, заготовки при регрессии и верификатор (AD-9).
func Fold(b Bundle, input []kernel.Record) (Snapshot, []kernel.Reaction) {
	in := slices.Clone(input)
	slices.SortStableFunc(in, func(x, y kernel.Record) int {
		switch {
		case kernel.Less(x, y):
			return -1
		case kernel.Less(y, x):
			return 1
		}
		return 0
	})
	var s Snapshot
	bySlot := map[string]kernel.Reaction{}
	for _, r := range in {
		var out kernel.Output
		s, out = Step(s, b, r)
		for _, re := range out.Reactions {
			bySlot[re.Slot.Key()+"\x1f"+string(re.Type)] = re
		}
	}
	keys := slices.Sorted(maps.Keys(bySlot))
	reactions := make([]kernel.Reaction, 0, len(keys))
	for _, k := range keys {
		reactions = append(reactions, bySlot[k])
	}
	return s, reactions
}

// Step — один шаг свёртки: Reduce всех модулей по порядку композиции, затем
// React; намерения применяют модули-владельцы в конце шага (AD-40).
func Step(s Snapshot, b Bundle, r kernel.Record) (Snapshot, kernel.Output) {
	if r.Seq > s.BasisSeq {
		s.BasisSeq = r.Seq
	}
	s.Item = item.Reduce(s.Item, r, b.Item)
	s.Process = process.Reduce(s.Process, r, b.Process, process.Upstream{Item: &s.Item})
	s.Vision = vision.Reduce(s.Vision, r, b.Vision, vision.Upstream{Item: &s.Item, Process: &s.Process})
	s.Quality = quality.Reduce(s.Quality, r, b.Quality, s.qualityUp())
	s.Machinelogs = machinelogs.Reduce(s.Machinelogs, r, b.Machinelogs, s.machinelogsUp())
	s.Documents = documents.Reduce(s.Documents, r, b.Documents, s.documentsUp())
	s.Nonconformity = nonconformity.Reduce(s.Nonconformity, r, b.Nonconformity, s.nonconformityUp())
	s.Analysis = analysis.Reduce(s.Analysis, r, b.Analysis, s.analysisUp())
	s.Notifications = notifications.Reduce(s.Notifications, r, b.Notifications, s.notificationsUp())

	var out kernel.Output
	out.Merge(item.React(s.Item, b.Item))
	out.Merge(process.React(s.Process, b.Process, process.Upstream{Item: &s.Item}))
	out.Merge(vision.React(s.Vision, b.Vision, vision.Upstream{Item: &s.Item, Process: &s.Process}))
	out.Merge(quality.React(s.Quality, b.Quality, s.qualityUp()))
	out.Merge(machinelogs.React(s.Machinelogs, b.Machinelogs, s.machinelogsUp()))
	out.Merge(documents.React(s.Documents, b.Documents, s.documentsUp()))
	out.Merge(nonconformity.React(s.Nonconformity, b.Nonconformity, s.nonconformityUp()))
	out.Merge(analysis.React(s.Analysis, b.Analysis, s.analysisUp()))
	out.Merge(notifications.React(s.Notifications, b.Notifications, s.notificationsUp()))

	for _, in := range out.Intents {
		switch in.Target {
		case process.Module:
			s.Process = process.Apply(s.Process, in)
		case quality.Module:
			s.Quality = quality.Apply(s.Quality, in)
		case documents.Module:
			s.Documents = documents.Apply(s.Documents, in)
		case nonconformity.Module:
			s.Nonconformity = nonconformity.Apply(s.Nonconformity, in)
		}
	}
	return s, out
}

// Guard — гард операции над состоянием изделия на basis_seq (AD-39): вызывает
// гард модуля-владельца операции (первый сегмент id). Операции модулей вне
// композиции проверяют свои гарды сами.
func Guard(s Snapshot, b Bundle, cmd kernel.Command) error {
	owner, _, _ := strings.Cut(cmd.Action, ".")
	switch kernel.Module(owner) {
	case item.Module:
		return item.Guard(s.Item, b.Item, cmd)
	case process.Module:
		return process.Guard(s.Process, b.Process, process.Upstream{Item: &s.Item}, cmd)
	case vision.Module:
		return vision.Guard(s.Vision, b.Vision, vision.Upstream{Item: &s.Item, Process: &s.Process}, cmd)
	case quality.Module:
		return quality.Guard(s.Quality, b.Quality, s.qualityUp(), cmd)
	case machinelogs.Module:
		return machinelogs.Guard(s.Machinelogs, b.Machinelogs, s.machinelogsUp(), cmd)
	case documents.Module:
		return documents.Guard(s.Documents, b.Documents, s.documentsUp(), cmd)
	case nonconformity.Module:
		return nonconformity.Guard(s.Nonconformity, b.Nonconformity, s.nonconformityUp(), cmd)
	case analysis.Module:
		return analysis.Guard(s.Analysis, b.Analysis, s.analysisUp(), cmd)
	case notifications.Module:
		return notifications.Guard(s.Notifications, b.Notifications, s.notificationsUp(), cmd)
	}
	return nil
}

func (s *Snapshot) qualityUp() quality.Upstream {
	return quality.Upstream{Item: &s.Item, Process: &s.Process, Vision: &s.Vision}
}

func (s *Snapshot) machinelogsUp() machinelogs.Upstream {
	return machinelogs.Upstream{Item: &s.Item, Process: &s.Process, Vision: &s.Vision, Quality: &s.Quality}
}

func (s *Snapshot) documentsUp() documents.Upstream {
	return documents.Upstream{Item: &s.Item, Process: &s.Process, Vision: &s.Vision, Quality: &s.Quality, Machinelogs: &s.Machinelogs}
}

func (s *Snapshot) nonconformityUp() nonconformity.Upstream {
	return nonconformity.Upstream{Item: &s.Item, Process: &s.Process, Vision: &s.Vision, Quality: &s.Quality, Machinelogs: &s.Machinelogs, Documents: &s.Documents}
}

func (s *Snapshot) analysisUp() analysis.Upstream {
	return analysis.Upstream{Item: &s.Item, Process: &s.Process, Vision: &s.Vision, Quality: &s.Quality, Machinelogs: &s.Machinelogs, Documents: &s.Documents, Nonconformity: &s.Nonconformity}
}

func (s *Snapshot) notificationsUp() notifications.Upstream {
	return notifications.Upstream{Item: &s.Item, Process: &s.Process, Vision: &s.Vision, Quality: &s.Quality, Machinelogs: &s.Machinelogs, Documents: &s.Documents, Nonconformity: &s.Nonconformity, Analysis: &s.Analysis}
}
