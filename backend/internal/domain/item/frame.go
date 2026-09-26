package item

import (
	"ant/internal/domain/kernel"
)

// Module — имя модуля-эмитента (AD-40).
const Module kernel.Module = "item"

// State — состояние модуля item в свёртке одного изделия (AD-5). Заполняет
// эпик-владелец модуля; свёртка — чистая функция входа изделия, поэтому State
// — значение, без ссылок на внешние ресурсы.
type State struct{}

// Env — закреплённая при запуске изделия часть нормативного слоя, нужная
// модулю item (AD-17: процесс, план контроля, карта реакций, шаблоны …), и срез
// справочников на occurred_at (AD-31). Собирает движок из пакета версии.
type Env struct{}

// Reduce применяет запись входа изделия (факт, решение, адресованную запись
// стадии) к состоянию модуля (AD-5). Реакции в свёртку не входят (AD-3).
func Reduce(s State, r kernel.Record, env Env) State {
	_, _ = r, env
	return s
}

// React вычисляет реакции модуля по состоянию после записи и намерения к
// ранним модулям (AD-3, AD-40). Реакции строятся только через kernel.NewReaction
// с собственными типами модуля.
func React(s State, env Env) kernel.Output {
	_, _ = s, env
	return kernel.Output{}
}

// Guard — доменный гард операций модуля item (AD-39): состояние изделия на
// basis_seq и команда → nil или *kernel.Refusal с кодом из contracts/errors.yaml.
// Его вызывают api до записи, свёртка при применении и верификатор.
func Guard(s State, env Env, cmd kernel.Command) error {
	_, _, _ = s, env, cmd
	return nil
}
