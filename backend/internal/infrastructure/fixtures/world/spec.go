package world

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Spec — world.yaml: компактное описание мира сценария (см. комментарии в файле).
type Spec struct {
	Format           int               `yaml:"format"`
	ID               string            `yaml:"id"`
	Title            string            `yaml:"title"`
	Description      string            `yaml:"description"`
	Covers           []string          `yaml:"covers"`
	Case             []string          `yaml:"case"`
	Enterprise       string            `yaml:"enterprise"`
	Month            string            `yaml:"month"`
	TZ               string            `yaml:"tz"`
	InitialStep      int               `yaml:"initial_step"`
	LocalIDs         []string          `yaml:"local_ids"`
	People           []PersonRef       `yaml:"people"`
	Lines            []Line            `yaml:"lines"`
	Suppliers        []Named           `yaml:"suppliers"`
	Orders           []Order           `yaml:"orders"`
	Lots             []Lot             `yaml:"lots"`
	Background       Background        `yaml:"background"`
	Items            []ItemSpec        `yaml:"items"`
	Rework           []ReworkSpec      `yaml:"rework"`
	Signals          []SignalSpec      `yaml:"signals"`
	NCs              []NCSpec          `yaml:"ncs"`
	Incidents        []IncidentSpec    `yaml:"incidents"`
	Reviews          []ReviewSpec      `yaml:"reviews"`
	ProcessHolds     []ProcessHold     `yaml:"process_holds"`
	Interventions    []Intervention    `yaml:"interventions"`
	ComponentUnlinks []ComponentUnlink `yaml:"component_unlinks"`
	Sources          []SourceSpec      `yaml:"sources"`
	LateEvents       []LateEvent       `yaml:"late_events"`
	Quarantine       []QuarantineSpec  `yaml:"quarantine"`
	ERP              []ERPSpec         `yaml:"erp"`
	Integrity        IntegritySpec     `yaml:"integrity"`
	Steps            []StepSpec        `yaml:"steps"`
}

// PersonRef — человек сценария процессной сессии и псевдоним политики.
type PersonRef struct {
	Ref       string `yaml:"ref"`
	Person    string `yaml:"person"`
	Workplace string `yaml:"workplace"`
	Shift     string `yaml:"shift"`
}

// Line — линия (пост сварки и источник).
type Line struct {
	ID     string `yaml:"id"`
	Name   string `yaml:"name"`
	Post   string `yaml:"post"`
	Source string `yaml:"source"`
}

// Named — id и название.
type Named struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

// Order — задание 1С.
type Order struct {
	ID       string `yaml:"id"`
	Label    string `yaml:"label"`
	Qty      int    `yaml:"qty"`
	Due      string `yaml:"due"`
	Received T      `yaml:"received"`
	Items    string `yaml:"items"`
	ERPRef   string `yaml:"erp_ref"`
}

// Lot — партия.
type Lot struct {
	ID       string `yaml:"id"`
	Label    string `yaml:"label"`
	ItemType string `yaml:"item_type"`
	Qty      int    `yaml:"qty"`
	Supplier string `yaml:"supplier"`
	Heat     string `yaml:"heat"`
	Cert     string `yaml:"cert"`
	Received T      `yaml:"received"`
	Accepted T      `yaml:"accepted"`
	Rings    string `yaml:"rings"`
}

// Background — фоновые выпущенные изделия.
type Background struct {
	Items      string   `yaml:"items"`
	Welders    []string `yaml:"welders"`
	Sources    []string `yaml:"sources"`
	From       T        `yaml:"from"`
	ReleasedBy T        `yaml:"released_by"`
}

// ItemSpec — изделие: вехи маршрута (пустые — генератор не ставит).
type ItemSpec struct {
	ID            string `yaml:"id"`
	Weld          Span   `yaml:"weld"`
	Src           string `yaml:"src"`
	Welder        string `yaml:"welder"`
	Ring          string `yaml:"ring"`
	EdgePrep      T      `yaml:"edge_prep"`
	Xray          T      `yaml:"xray"`
	ZT3           T      `yaml:"zt3"`
	ZT3Incomplete bool   `yaml:"zt3_incomplete"`
	ToAC          T      `yaml:"to_ac"`
	Asm           Span   `yaml:"asm"`
	ZT4           T      `yaml:"zt4"`
	Test          Span   `yaml:"test"`
	ZT5           T      `yaml:"zt5"`
	KT5           T      `yaml:"kt5"`
	ZT6           T      `yaml:"zt6"`
	Rel           T      `yaml:"rel"`
	Reference     bool   `yaml:"reference"`
	Anchor        bool   `yaml:"anchor"`
	// Moves — явные положения сверх маршрута по вехам (сборка приостановлена и т. п.).
	Moves []MoveSpec `yaml:"moves"`
}

// MoveSpec — явное положение изделия: шаг, положение, место.
type MoveSpec struct {
	At   T      `yaml:"at"`
	Step string `yaml:"step"`
	Pos  string `yaml:"pos"`
	Loc  string `yaml:"loc"`
}

// ReworkSpec — повторное выполнение сварки и дальнейший маршрут.
type ReworkSpec struct {
	Item string `yaml:"item"`
	Weld Span   `yaml:"weld"`
	Xray T      `yaml:"xray"`
	ZT3  T      `yaml:"zt3"`
	ToAC T      `yaml:"to_ac"`
	Asm  Span   `yaml:"asm"`
	ZT4  T      `yaml:"zt4"`
	Test Span   `yaml:"test"`
}

// SignalSpec — сигнал без несоответствия: отклонён или «оценка невозможна».
type SignalSpec struct {
	ID           string `yaml:"id"`
	Item         string `yaml:"item"`
	At           T      `yaml:"at"`
	Source       string `yaml:"source"`
	Zone         string `yaml:"zone"`
	Kind         string `yaml:"kind"`
	Unable       bool   `yaml:"unable"`
	ConfidenceBP int    `yaml:"confidence_bp"`
	QualityBP    int    `yaml:"quality_bp"`
	Rejected     T      `yaml:"rejected"`
	Resolved     T      `yaml:"resolved"`
	By           string `yaml:"by"`
	Reason       string `yaml:"reason"`
	Scenario     string `yaml:"scenario"`
}

// Defect — дефект несоответствия: зона и вид (FR-37).
type Defect struct {
	Zone string `yaml:"zone"`
	Kind string `yaml:"kind"`
	Side string `yaml:"side"`
}

// Disposition — решение по изделию.
type Disposition struct {
	Kind string   `yaml:"kind"`
	At   T        `yaml:"at"`
	By   []string `yaml:"by"`
}

// Cause — подтверждённая причина.
type Cause struct {
	Category string `yaml:"category"`
	At       T      `yaml:"at"`
	Ref      string `yaml:"ref"`
}

// NCSpec — несоответствие.
type NCSpec struct {
	ID          string       `yaml:"id"`
	Label       string       `yaml:"label"`
	Item        string       `yaml:"item"`
	Items       []string     `yaml:"items"`
	Lot         string       `yaml:"lot"`
	Component   string       `yaml:"component"`
	Signal      T            `yaml:"signal"`
	Confirmed   T            `yaml:"confirmed"`
	By          string       `yaml:"by"`
	Source      string       `yaml:"source"`
	Report      string       `yaml:"report"`
	Isolated    T            `yaml:"isolated"`
	Kind        string       `yaml:"kind"`
	Defects     []Defect     `yaml:"defects"`
	Severity    string       `yaml:"severity"`
	Cause       *Cause       `yaml:"cause"`
	Disposition *Disposition `yaml:"disposition"`
	Scenario    string       `yaml:"scenario"`
}

// AllItems — изделия несоответствия.
func (n NCSpec) AllItems() []string {
	if n.Item != "" {
		return []string{n.Item}
	}
	return n.Items
}

// IncidentSpec — инцидент и версии области риска.
type IncidentSpec struct {
	ID             string         `yaml:"id"`
	Label          string         `yaml:"label"`
	Opened         T              `yaml:"opened"`
	Trigger        string         `yaml:"trigger"`
	Factors        []string       `yaml:"factors"`
	Versions       []ScopeVersion `yaml:"versions"`
	Closed         T              `yaml:"closed"`
	CauseConfirmed T              `yaml:"cause_confirmed"`
}

// ScopeVersion — версия области риска: правило построения и основание.
type ScopeVersion struct {
	V         int      `yaml:"v"`
	At        T        `yaml:"at"`
	By        string   `yaml:"by"`
	Rule      string   `yaml:"rule"`
	Source    string   `yaml:"source"`
	Before    T        `yaml:"before"`
	Unknown   []string `yaml:"unknown"`
	Confirmed []string `yaml:"confirmed"`
	Exclude   []string `yaml:"exclude"`
	Lot       string   `yaml:"lot"`
	LateEvent string   `yaml:"late_event"`
	Basis     string   `yaml:"basis"`
}

// ReviewSpec — пересмотр подписанной приёмки.
type ReviewSpec struct {
	Item      string `yaml:"item"`
	Gate      string `yaml:"gate"`
	Flagged   T      `yaml:"flagged"`
	Completed T      `yaml:"completed"`
	Outcome   string `yaml:"outcome"`
	By        string `yaml:"by"`
	LateEvent string `yaml:"late_event"`
}

// ProcessHold — остановка точки процесса (оборудования).
type ProcessHold struct {
	Equipment        string `yaml:"equipment"`
	Set              T      `yaml:"set"`
	By               string `yaml:"by"`
	ProposedBy       string `yaml:"proposed_by"`
	Reason           string `yaml:"reason"`
	ReleaseCondition string `yaml:"release_condition"`
}

// Intervention — вскрытие собранного изделия.
type Intervention struct {
	Item     string `yaml:"item"`
	Opened   T      `yaml:"opened"`
	By       string `yaml:"by"`
	Note     string `yaml:"note"`
	BackToWC T      `yaml:"back_to_wc"`
}

// ComponentUnlink — компонент снят с изделия.
type ComponentUnlink struct {
	Item      string `yaml:"item"`
	Component string `yaml:"component"`
	At        T      `yaml:"at"`
	Note      string `yaml:"note"`
}

// SourceSpec — источник событий: потеря связи и пачка после восстановления.
type SourceSpec struct {
	ID        string `yaml:"id"`
	Equipment string `yaml:"equipment"`
	Lost      T      `yaml:"lost"`
	Alert     T      `yaml:"alert"`
	Restored  T      `yaml:"restored"`
	Batch     struct {
		Records       int    `yaml:"records"`
		Duplicates    int    `yaml:"duplicates"`
		Late          int    `yaml:"late"`
		Lost          int    `yaml:"lost"`
		LostWindow    Span   `yaml:"lost_window"`
		Gap           string `yaml:"gap"`
		MaxDelayHours int    `yaml:"max_delay_hours"`
	} `yaml:"batch"`
}

// LateEvent — ключевая опоздавшая запись.
type LateEvent struct {
	ID       string `yaml:"id"`
	Source   string `yaml:"source"`
	Occurred T      `yaml:"occurred"`
	Received T      `yaml:"received"`
	CurrentA int    `yaml:"current_a"`
	Setpoint string `yaml:"setpoint"`
	RunItem  string `yaml:"run_item"`
}

// QuarantineSpec — сообщение в карантине приёма.
type QuarantineSpec struct {
	ID     string `yaml:"id"`
	At     T      `yaml:"at"`
	Code   string `yaml:"code"`
	Field  string `yaml:"field"`
	Value  string `yaml:"value"`
	Source string `yaml:"source"`
	Fixed  T      `yaml:"fixed"`
	By     string `yaml:"by"`
}

// ERPSpec — исходящее сообщение 1С и попытки доставки.
type ERPSpec struct {
	ID       string       `yaml:"id"`
	Kind     string       `yaml:"kind"`
	Items    []string     `yaml:"items"`
	Lot      string       `yaml:"lot"`
	Qty      int          `yaml:"qty"`
	Route    string       `yaml:"route"`
	At       T            `yaml:"at"`
	Basis    string       `yaml:"basis"`
	Attempts []ERPAttempt `yaml:"attempts"`
}

// ERPAttempt — попытка доставки и ответ 1С.
type ERPAttempt struct {
	At     T      `yaml:"at"`
	Result string `yaml:"result"`
	Code   string `yaml:"code"`
	HTTP   int    `yaml:"http"`
	Doc    string `yaml:"doc"`
}

// IntegritySpec — проверки целостности и найденные нарушения.
type IntegritySpec struct {
	IntervalMinutes int `yaml:"interval_minutes"`
	Violations      []struct {
		At     T      `yaml:"at"`
		Record string `yaml:"record"`
		Note   string `yaml:"note"`
	} `yaml:"violations"`
}

// StepSpec — шаг курсора.
type StepSpec struct {
	At        T         `yaml:"at"`
	Title     string    `yaml:"title"`
	Scenarios []string  `yaml:"scenarios"`
	Wait      *WaitSpec `yaml:"wait"`
}

// WaitSpec — ожидание решения человека.
type WaitSpec struct {
	Action string `yaml:"action"`
	Role   string `yaml:"role"`
	Object struct {
		Kind string `yaml:"kind"`
		ID   string `yaml:"id"`
	} `yaml:"object"`
	Title string `yaml:"title"`
}

// ─────────────────────────────── время ───────────────────────────────

// T — момент «ДД ЧЧ:ММ» (день месяца сценария, местное время) или пусто.
type T struct {
	raw string
	t   time.Time
}

// Span — интервал «ДД ЧЧ:ММ-ЧЧ:ММ» (конец раньше начала — следующий день).
type Span struct {
	raw      string
	From, To T
}

// UnmarshalYAML читает «ДД ЧЧ:ММ».
func (t *T) UnmarshalYAML(unmarshal func(any) error) error {
	return unmarshal(&t.raw)
}

// UnmarshalYAML читает «ДД ЧЧ:ММ-ЧЧ:ММ».
func (s *Span) UnmarshalYAML(unmarshal func(any) error) error {
	return unmarshal(&s.raw)
}

// IsZero — момент не задан.
func (t T) IsZero() bool { return t.t.IsZero() }

// Time — момент в UTC.
func (t T) Time() time.Time { return t.t }

// IsZero — интервал не задан.
func (s Span) IsZero() bool { return s.From.IsZero() }

var reT = regexp.MustCompile(`^(\d{1,2}) (\d{2}):(\d{2})$`)

// clock — разбор времён описания в месяце и поясе сценария.
type clock struct {
	year  int
	month time.Month
	loc   *time.Location
}

func newClock(month, tz string) (clock, error) {
	ym, err := time.Parse("2006-01", month)
	if err != nil {
		return clock{}, fmt.Errorf("month %q: %w", month, err)
	}
	off, err := time.Parse("-07:00", tz)
	if err != nil {
		return clock{}, fmt.Errorf("tz %q: %w", tz, err)
	}
	_, sec := off.Zone()
	return clock{year: ym.Year(), month: ym.Month(), loc: time.FixedZone(tz, sec)}, nil
}

func (c clock) at(day, hh, mm int) time.Time {
	return time.Date(c.year, c.month, day, hh, mm, 0, 0, c.loc).UTC()
}

func (c clock) parseT(t *T) error {
	if t.raw == "" {
		return nil
	}
	m := reT.FindStringSubmatch(strings.TrimSpace(t.raw))
	if m == nil {
		return fmt.Errorf("время %q: ожидается «ДД ЧЧ:ММ»", t.raw)
	}
	d, _ := strconv.Atoi(m[1])
	h, _ := strconv.Atoi(m[2])
	mi, _ := strconv.Atoi(m[3])
	t.t = c.at(d, h, mi)
	return nil
}

func (c clock) parseSpan(s *Span) error {
	if s.raw == "" {
		return nil
	}
	parts := strings.SplitN(strings.TrimSpace(s.raw), "-", 2)
	if len(parts) != 2 {
		return fmt.Errorf("интервал %q: ожидается «ДД ЧЧ:ММ-ЧЧ:ММ»", s.raw)
	}
	s.From.raw = parts[0]
	if err := c.parseT(&s.From); err != nil {
		return err
	}
	day := strings.Fields(parts[0])[0]
	s.To.raw = day + " " + parts[1]
	if err := c.parseT(&s.To); err != nil {
		return err
	}
	if s.To.t.Before(s.From.t) {
		s.To.t = s.To.t.Add(24 * time.Hour)
	}
	return nil
}

// expandRange — «F-001..F-040» → F-001…F-040.
func expandRange(r string) ([]string, error) {
	if r == "" {
		return nil, nil
	}
	a, b, ok := strings.Cut(r, "..")
	if !ok {
		return []string{r}, nil
	}
	i := strings.LastIndexByte(a, '-')
	j := strings.LastIndexByte(b, '-')
	if i < 0 || j < 0 || a[:i] != b[:j] {
		return nil, fmt.Errorf("диапазон %q", r)
	}
	from, err1 := strconv.Atoi(a[i+1:])
	to, err2 := strconv.Atoi(b[j+1:])
	if err1 != nil || err2 != nil || to < from {
		return nil, fmt.Errorf("диапазон %q", r)
	}
	w := len(a) - i - 1
	out := make([]string, 0, to-from+1)
	for n := from; n <= to; n++ {
		out = append(out, fmt.Sprintf("%s-%0*d", a[:i], w, n))
	}
	return out, nil
}
