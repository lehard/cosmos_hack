package analysis

import (
	"ant/internal/domain/documents"
	"ant/internal/domain/item"
	"ant/internal/domain/kernel"
	"ant/internal/domain/machinelogs"
	"ant/internal/domain/nonconformity"
	"ant/internal/domain/process"
	"ant/internal/domain/quality"
	"ant/internal/domain/vision"
)

// Module — имя модуля-эмитента (AD-40).
const Module kernel.Module = "analysis"

// State — состояние модуля analysis в свёртке одного изделия (AD-5). Заполняет
// эпик-владелец модуля; свёртка — чистая функция входа изделия, поэтому State
// — значение, без ссылок на внешние ресурсы.
type State struct{}

// Env — закреплённая при запуске изделия часть нормативного слоя, нужная
// модулю analysis (AD-17: процесс, план контроля, карта реакций, шаблоны …), и срез
// справочников на occurred_at (AD-31). Собирает движок из пакета версии.
type Env struct{}

// Upstream — состояния модулей раньше analysis в композиции на этом шаге
// (только чтение, AD-40): поздний модуль видит вывод раннего, обратно — только
// через функцию-намерение раннего модуля.
type Upstream struct {
	Item          *item.State
	Process       *process.State
	Vision        *vision.State
	Quality       *quality.State
	Machinelogs   *machinelogs.State
	Documents     *documents.State
	Nonconformity *nonconformity.State
}

// Reduce применяет запись входа изделия (факт, решение, адресованную запись
// стадии) к состоянию модуля (AD-5). Реакции в свёртку не входят (AD-3).
func Reduce(s State, r kernel.Record, env Env, up Upstream) State {
	_, _ = r, env
	_ = up
	return s
}

// React вычисляет реакции модуля по состоянию после записи и намерения к
// ранним модулям (AD-3, AD-40). Реакции строятся только через kernel.NewReaction
// с собственными типами модуля.
func React(s State, env Env, up Upstream) kernel.Output {
	_, _ = s, env
	_ = up
	return kernel.Output{}
}

// Guard — доменный гард операций модуля analysis (AD-39): состояние изделия на
// basis_seq и команда → nil или *kernel.Refusal с кодом из contracts/errors.yaml.
// Его вызывают api до записи, свёртка при применении и верификатор.
func Guard(s State, env Env, up Upstream, cmd kernel.Command) error {
	_, _, _ = s, env, cmd
	_ = up
	return nil
}
