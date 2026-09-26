package loader

import (
	"context"
	"slices"
	"sync"
	"time"

	"ant/internal/application/platform"
)

// Сессионное наложение мира заготовок (FR-129, AD-36).
//
// Мир заготовок детерминирован: ответы — заготовки шага курсора, файлы на
// диске и встроенный мир команды не меняют. Но стенд в режиме заготовок
// должен заметно отвечать на действия людей: «Выполнено» у задачи закрывает
// её, подпись закрывает этап маршрута, начатая с терминала операция видна на
// терминале. Для этого команда записывает факт в память процесса (Record), а
// чтения модульных адаптеров накладывают факты поверх ответа мира (Facts).
//
// Правила:
//   - ключ факта — вид и id объекта (без префикса прогона) в пределах прогона
//     пульта: факты другого прогона не видны;
//   - сброс — перезапуск процесса, старт и остановка прогона пульта (Start,
//     Stop): прогон всегда начинается с чистого мира;
//   - номер квитанции уникален: seq шага курсора и поверх него монотонный
//     счётчик процесса (n·SeqPerStep + SeqSessionAt + k), а не один seq шага
//     на все команды;
//   - повтор команды с тем же command_id возвращает прежнюю квитанцию
//     (Replayed) и второй факт не пишет (AD-7);
//   - чтение на момент в прошлом (as_of раньше факта) факт не видит;
//   - изменения объектов фактов уходят живым обновлениям (SessionChanges):
//     открытые экраны перечитывают их сами.

// SeqSessionAt — начало диапазона seq квитанций сессионных команд внутри шага:
// записи журнала шага — n·SeqPerStep+1…, квитанции — n·SeqPerStep+SeqSessionAt+k,
// изменения для SSE — с n·SeqPerStep+SeqChangesAt.
const SeqSessionAt = 5000

// seqSessionSpan — сколько квитанций помещается в шаге до диапазона изменений.
const seqSessionSpan = SeqChangesAt - SeqSessionAt - 1

// Fact — факт сессии: команда человека над объектом мира заготовок.
type Fact struct {
	// Op — операция команды (documents.document.sign, notifications.task.acknowledge…).
	Op string
	// Kind и ID — объект команды; ID без префикса прогона.
	Kind, ID string
	// RunID — прогон пульта, в котором записан факт.
	RunID string
	// Actor и Role — псевдоним и активная роль субъекта команды.
	Actor, Role string
	// At — доменное время квитанции (часы шага курсора).
	At time.Time
	// Seq — номер квитанции.
	Seq int64
	// CommandID — id команды клиента.
	CommandID string
	// Body — тело команды (тип модуля) для наложения на ответы.
	Body any
}

// sessionChange — изменение для живых обновлений с номером версии сессии.
type sessionChange struct {
	ver int64
	SeqChange
}

// session — факты прогона в памяти процесса.
type session struct {
	mu      sync.Mutex
	runID   string
	counter int64
	facts   []Fact
	byCmd   map[string]platform.Receipt
	ver     int64
	changes []sessionChange
}

// reset — чистая сессия прогона runID (старт, остановка, смена прогона).
// Счётчик квитанций и версия изменений не сбрасываются: номера не повторяются.
func (s *session) reset(runID string) {
	s.runID = runID
	s.facts = nil
	s.byCmd = map[string]platform.Receipt{}
	s.ver++
	s.changes = append(s.changes[:0], sessionChange{ver: s.ver, SeqChange: SeqChange{Change: Change{Entity: string(platform.EntityNotification), ID: "global"}, RunID: runID}})
}

// nextSeq — уникальный номер квитанции на шаге n.
func (s *session) nextSeq(n int) int64 {
	s.counter++
	k := (s.counter-1)%seqSessionSpan + 1
	return int64(n)*SeqPerStep + SeqSessionAt + k
}

// resetSession — сброс сессионного наложения (старт и остановка прогона).
func (r *Runtime) resetSession(runID string) {
	r.sess.mu.Lock()
	defer r.sess.mu.Unlock()
	r.sess.reset(runID)
}

// Record — команда человека с наложением: квитанция как у Decide (шаг
// ожидания именно этого решения двигает сценарий) и факт в памяти процесса;
// touched — объекты, которые команда меняет сверх самого объекта (для живых
// обновлений: список задач, очередь решений…). Повтор command_id — прежняя
// квитанция без второго факта.
func (r *Runtime) Record(ctx context.Context, op string, obj ObjectRef, meta platform.CommandMeta, body any, touched ...Change) (platform.Receipt, error) {
	if rc, ok := r.replayed(meta.CommandID); ok {
		return rc, nil
	}
	rc, err := r.Decide(ctx, op, obj, meta)
	if err != nil {
		return rc, err
	}
	st, _, err := r.State(ctx)
	if err != nil {
		return rc, err
	}
	p := platform.PrincipalFrom(ctx)
	f := Fact{Op: op, Kind: obj.Kind, ID: StripRun(obj.ID, st.RunID), RunID: st.RunID, Actor: p.PersonID, Role: p.Role,
		At: rc.RecordedAt, Seq: rc.Seq, CommandID: meta.CommandID, Body: body}
	r.sess.mu.Lock()
	defer r.sess.mu.Unlock()
	if r.sess.byCmd == nil || r.sess.runID != st.RunID {
		r.sess.reset(st.RunID)
	}
	r.sess.facts = append(r.sess.facts, f)
	if meta.CommandID != "" {
		r.sess.byCmd[meta.CommandID] = rc
	}
	changes := touched
	if obj.Kind != "" && obj.ID != "" {
		changes = append([]Change{{Entity: obj.Kind, ID: obj.ID}}, touched...)
	}
	r.sess.ver++
	sc, _ := r.lib.Scenario(st.Scenario)
	for _, c := range changes {
		id := c.ID
		if sc != nil && id != "global" {
			id = sc.PrefixID(StripRun(id, st.RunID), st.RunID)
		}
		r.sess.changes = append(r.sess.changes, sessionChange{ver: r.sess.ver, SeqChange: SeqChange{Change: Change{Entity: c.Entity, ID: id}, Seq: rc.Seq, RunID: st.RunID}})
	}
	return rc, nil
}

// replayed — квитанция уже принятой команды с тем же command_id.
func (r *Runtime) replayed(commandID string) (platform.Receipt, bool) {
	if commandID == "" {
		return platform.Receipt{}, false
	}
	r.sess.mu.Lock()
	defer r.sess.mu.Unlock()
	rc, ok := r.sess.byCmd[commandID]
	if ok {
		rc.Replayed = true
	}
	return rc, ok
}

// Facts — факты текущего прогона видов kinds (все виды, если пусто) в порядке
// записи; m — момент чтения: факты позже as_of не видны, чтение другого
// прогона (m.RunID) фактов не видит.
func (r *Runtime) Facts(ctx context.Context, m *platform.Moment, kinds ...string) []Fact {
	st, _, err := r.State(ctx)
	if err != nil {
		return nil
	}
	if m != nil && m.RunID != "" && m.RunID != st.RunID {
		return nil
	}
	r.sess.mu.Lock()
	defer r.sess.mu.Unlock()
	if r.sess.runID != st.RunID {
		return nil
	}
	var out []Fact
	for _, f := range r.sess.facts {
		if len(kinds) > 0 && !slices.Contains(kinds, f.Kind) {
			continue
		}
		if m != nil && m.AsOf != nil && m.AsOf.Before(f.At) {
			continue
		}
		out = append(out, f)
	}
	return out
}

// FactsOf — факты текущего прогона над объектом kind/id (id с префиксом
// прогона или без него).
func (r *Runtime) FactsOf(ctx context.Context, m *platform.Moment, kind, id string) []Fact {
	st, _, _ := r.State(ctx)
	id = StripRun(id, st.RunID)
	var out []Fact
	for _, f := range r.Facts(ctx, m, kind) {
		if f.ID == id {
			out = append(out, f)
		}
	}
	return out
}

// SessionVersion — версия изменений сессии (для подписки живых обновлений).
func (r *Runtime) SessionVersion() int64 {
	r.sess.mu.Lock()
	defer r.sess.mu.Unlock()
	return r.sess.ver
}

// SessionChanges — изменения объектов сессионных фактов после версии after и
// новая версия: подписка journal отдаёт их как живые обновления.
func (r *Runtime) SessionChanges(after int64) ([]SeqChange, int64) {
	r.sess.mu.Lock()
	defer r.sess.mu.Unlock()
	var out []SeqChange
	for _, c := range r.sess.changes {
		if c.ver > after {
			out = append(out, c.SeqChange)
		}
	}
	// Хвост изменений ограничен: подписки читают его раз в полсекунды.
	if len(r.sess.changes) > 1024 {
		r.sess.changes = append([]sessionChange(nil), r.sess.changes[len(r.sess.changes)-512:]...)
	}
	return out, r.sess.ver
}

// Local — id объекта без префикса текущего прогона (ключ фактов сессии).
func (r *Runtime) Local(ctx context.Context, id string) string {
	st, _, err := r.State(ctx)
	if err != nil {
		return id
	}
	return StripRun(id, st.RunID)
}

// Holders — кто вправе выполнить действие (псевдонимы по стартовой политике):
// подписанты запроса решения на заготовках. Регистрирует встроенный мир
// (пакет world); nil — неизвестно.
var Holders func(action string) []string
