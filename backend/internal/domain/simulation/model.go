package simulation

// Модель определений сценариев (scenarios/definitions, AD-26): мир, прогон,
// карточки сценариев. Определения — данные (YAML); инфраструктура разбирает их
// в эти структуры через JSON (json-теги), домен только считает по ним.
//
// Идентификаторы в определениях — машинные ID процессной сессии
// (process/scenarios-flange.md: F-017, WLD-02, WS-2, U2) — так карточки
// остаются «один к одному» с исходными таблицами; в события и запросы они
// попадают только через сведение имён World.Aliases (решение пользователя:
// имена процессной сессии — через сведение с контрактом, эпик 00).

// World — мир сценариев: предприятие, изделие, источники, люди, линии,
// нормы маршрута. Один мир на семейство прогонов (фланец ФЛ-100).
type World struct {
	ID           string `json:"id"`
	Version      string `json:"version"`
	Title        string `json:"title"`
	Enterprise   string `json:"enterprise"`
	ItemType     string `json:"item_type"`
	ItemRevision string `json:"item_revision"`
	// Aliases — сведение ID процессной сессии с ID контракта и затравки
	// normative/ (люди → персоны генезиса, оборудование, зоны, роли).
	Aliases Aliases `json:"aliases"`
	// Sources — источники событий по ключу определения; source_id в прогоне —
	// ‹run_id›/‹id› (AD-38).
	Sources map[string]Source `json:"sources"`
	// Lines — линии и их сварочные посты (кейс §1.4, §2.4).
	Lines map[string]Line `json:"lines"`
	// Route — нормы маршрута фоновых изделий (длительности, шаги, исполнители).
	Route Route `json:"route"`
	// Shifts — смены: A и B, местное время начала и конца.
	Shifts map[string]Shift `json:"shifts"`
}

// Aliases — сведение имён процессной сессии с контрактом (эпик 00).
type Aliases struct {
	People    map[string]string `json:"people"`
	Equipment map[string]string `json:"equipment"`
	Zones     map[string]string `json:"zones"`
	Roles     map[string]string `json:"roles"`
	Codes     map[string]string `json:"codes"`
	Other     map[string]string `json:"other"`
}

// Source — источник событий (устройство, шлюз, терминал, stand внешней системы).
type Source struct {
	// ID — локальный source_id (без префикса прогона): edge-weld-2, term-weld-2.
	ID          string `json:"id"`
	Kind        string `json:"kind"`        // source_kind (AD-2, FR-140)
	Reliability string `json:"reliability"` // high | medium | low | unknown
	Equipment   string `json:"equipment,omitempty"`
	Station     string `json:"station,omitempty"`
	Workplace   string `json:"workplace,omitempty"`
	// SeqStart — первый source_seq источника в прогоне (по умолчанию 1).
	SeqStart int64  `json:"seq_start,omitempty"`
	Note     string `json:"note,omitempty"`
}

// Line — линия (кейс §2.4): свой сварочный пост, остальные участки общие.
type Line struct {
	Title         string `json:"title"`
	WeldingSource string `json:"welding_source"` // ключ источника сварочного аппарата
	WeldTerminal  string `json:"weld_terminal"`  // ключ терминала поста
	Workplace     string `json:"workplace"`
	Equipment     string `json:"equipment"` // WS-1 | WS-2 (ID процессной сессии)
}

// Shift — смена: начало и конец по местному времени «ЧЧ:ММ».
type Shift struct {
	Starts string `json:"starts"`
	Ends   string `json:"ends"`
}

// Route — нормы маршрута фонового фланца: длительности этапов в минутах,
// исполнители по умолчанию. Порядок этапов и формы событий — в route.go.
type Route struct {
	MachiningMin  int    `json:"machining_min"`
	CMMMin        int    `json:"cmm_min"`
	EdgePrepMin   int    `json:"edge_prep_min"`
	WeldMin       int    `json:"weld_min"`
	XRayAfterMin  int    `json:"xray_after_min"`
	ZT3AfterMin   int    `json:"zt3_after_min"`
	AssemblyMin   int    `json:"assembly_min"`
	LeakTestMin   int    `json:"leak_test_min"`
	FinalAfterMin int    `json:"final_after_min"`
	CNCOperator   string `json:"cnc_operator"`
	CNCOperatorB  string `json:"cnc_operator_b"`
	CMMOperator   string `json:"cmm_operator"`
	QCMachining   string `json:"qc_machining"`
	QCWelding     string `json:"qc_welding"`
	QCAssembly    string `json:"qc_assembly"`
	QCFinal       string `json:"qc_final"`
	Customer      string `json:"customer"`
	NDT           string `json:"ndt"`
	Assembler     string `json:"assembler"`
	Tester        string `json:"tester"`
	Storekeeper   string `json:"storekeeper"`
	MasterMC      string `json:"master_mc"`
	MasterWC      string `json:"master_wc"`
	MasterAC      string `json:"master_ac"`
	Program       string `json:"program"`
	WireLot       string `json:"wire_lot"`
	// CurrentNominal, CurrentTol — уставка тока сварки, А (160 ± 10, [ПП]).
	CurrentNominal int `json:"current_nominal"`
	CurrentTol     int `json:"current_tol"`
}

// RunDef — определение прогона: главная история, отдельный прогон, сбой.
type RunDef struct {
	ID          string   `json:"id"`
	Version     string   `json:"version"`
	Title       string   `json:"title"`
	Kind        string   `json:"kind"` // demo | standalone | failure | load
	Description string   `json:"description,omitempty"`
	CaseRefs    []string `json:"case_refs,omitempty"`
	World       string   `json:"world"`
	Seed        int64    `json:"seed"`
	// Anchor — начало виртуальных часов прогона по определению (RFC 3339 с
	// зоной). Даты определения — смещения от него: при старте прогон
	// сдвигается на целое число недель к доменному «сейчас» (AD-37, AD-38).
	Anchor string `json:"anchor"`
	// End — конец прогона (виртуальное время).
	End string `json:"end"`
	// Speed — скорость по умолчанию ×1…×1000.
	Speed int `json:"speed,omitempty"`
	// Scenarios — карточки, чьи шаги входят в прогон (порядок — как в карточке прогона).
	Scenarios []string    `json:"scenarios"`
	Orders    []OrderPlan `json:"orders,omitempty"`
	Lots      []LotPlan   `json:"lots,omitempty"`
	Items     []ItemPlan  `json:"items,omitempty"`
	// Noise — случайный шум фона (б. п.); главная история — только заданный
	// карточками шум (всё по 0), прогоны сбоев — заданные доли.
	Noise Noise `json:"noise,omitempty"`
	// Refs — ссылки на объекты, которые рождает система (несоответствия,
	// инциденты, сигналы): вычисляются во время прогона теми же Queries.
	Refs map[string]Ref `json:"refs,omitempty"`
	// Stops — решения, на которых интерактивный прогон ждёт стол роли
	// (FR-129); остальные решения подписывает demo-signer и в интерактиве.
	Stops []string `json:"stops,omitempty"`
	// Live — живая часть прогона (сценарий показа, Д-85); nil — весь прогон
	// идёт по часам, решения — по Stops и stop карточек.
	Live *LivePart `json:"live,omitempty"`
}

// LivePart — живая часть прогона (сценарий показа, Д-85): до From — готовая
// история (решения подписывает demo-signer, интерактивный прогон проигрывает
// её сразу, без часов); с From — только руками: каждое решение человека —
// остановка прогона до нажатия на столе роли, demo-signer не подписывает ничего.
type LivePart struct {
	From string `json:"from"`
}

// OrderPlan — задание 1С (Е-01): приходит через stand 1С.
type OrderPlan struct {
	ID       string `json:"id"` // ORD-0917
	At       string `json:"at"`
	External string `json:"external"` // номер в 1С
	Quantity int    `json:"quantity"`
	Due      string `json:"due,omitempty"`
	Items    string `json:"items,omitempty"` // «F-001…F-040» — справочно
	// Lots — партии компонентов изделий задания по умолчанию:
	// blank, ring, cover, seal, fastener, valve.
	Lots map[string]string `json:"lots,omitempty"`
}

// LotPlan — партия (Е-03…Е-08): поступление из 1С, приёмка, входной контроль, ЗТ-1.
type LotPlan struct {
	ID          string `json:"id"`        // LOT-R-117
	Component   string `json:"component"` // тип компонента (номенклатура)
	Supplier    string `json:"supplier"`
	Quantity    int    `json:"quantity"`
	Certificate string `json:"certificate,omitempty"`
	Heat        string `json:"heat,omitempty"`
	External    string `json:"external,omitempty"`
	Arrived     string `json:"arrived"`
	Accepted    string `json:"accepted,omitempty"` // ЗТ-1; пусто — вместе с поступлением + 2 ч
	// Units — экземпляры партии с номерами (R-101…R-110), если учитываются поштучно.
	Units string `json:"units,omitempty"`
	Note  string `json:"note,omitempty"`
	// Issues — выдачи партии на участок без изделия (проволока на сварочный участок).
	Issues []LotIssue `json:"issues,omitempty"`
}

// LotIssue — выдача партии на место.
type LotIssue struct {
	At       string `json:"at"`
	To       string `json:"to"`
	Quantity int    `json:"quantity"`
}

// ItemPlan — план фланца: запуск, мехобработка, сварка и докуда маршрут
// строится генератором; дальше — шаги карточек.
type ItemPlan struct {
	ID     string `json:"id"` // F-017
	Order  string `json:"order"`
	Launch string `json:"launch"` // Е-09: выдача заготовки
	// Machining — начало мехобработки; пусто — Launch + 30 мин.
	Machining string `json:"machining,omitempty"`
	// MachiningMin — длительность мехобработки, мин (по умолчанию — норма маршрута).
	MachiningMin int `json:"machining_min,omitempty"`
	// Weld — сварка (главный параметр области риска §2.6).
	Weld *WeldPlan `json:"weld,omitempty"`
	// Until — последний этап, который строит маршрут (см. Stages).
	Until string `json:"until"`
	// Skip — этапы, которые вместо маршрута дают шаги карточек.
	Skip []string `json:"skip,omitempty"`
	// At — явные времена этапов (перекрывают расчёт по нормам).
	At map[string]string `json:"at,omitempty"`
	// Components — кольцо, крышка, клапан (ID экземпляров) и партии.
	Ring      string `json:"ring,omitempty"`
	RingLot   string `json:"ring_lot,omitempty"`
	Cover     string `json:"cover,omitempty"`
	Valve     string `json:"valve,omitempty"`
	BlankLot  string `json:"blank_lot,omitempty"`
	Note      string `json:"note,omitempty"`
	Reference bool   `json:"reference,omitempty"` // эталон для сравнения (Ф-001)
}

// WeldPlan — сварка фланца с кольцом.
type WeldPlan struct {
	At      string `json:"at"`
	Minutes int    `json:"minutes,omitempty"`
	Station string `json:"station"` // WS-1 | WS-2
	Welder  string `json:"welder"`  // WLD-02 | WLD-03
	// Current — ток по журналу, А: [мин, макс].
	Current []int `json:"current,omitempty"`
	// OutFrom — с какого момента ток вне уставки (пусто — весь шов в уставке).
	OutFrom string `json:"out_from,omitempty"`
	// Arc — окно горения дуги [от, до] (сводки источника — только в нём);
	// пусто — вся сварка.
	Arc []string `json:"arc,omitempty"`
}

// Noise — доли шума фона в базисных пунктах (0…10000) и параметры.
type Noise struct {
	DuplicateBP  int `json:"duplicate_bp,omitempty"`
	LateBP       int `json:"late_bp,omitempty"`
	LateMin      int `json:"late_min,omitempty"`
	LossBP       int `json:"loss_bp,omitempty"`
	ReorderBP    int `json:"reorder_bp,omitempty"`
	ClockSkewSec int `json:"clock_skew_s,omitempty"`
	// Sources — ключи источников, к которым применяется шум (пусто — ко всем фоновым).
	Sources []string `json:"sources,omitempty"`
}

// Ref — ссылка на объект, рождённый системой: чтение той же операцией API.
type Ref struct {
	Operation string         `json:"operation"`
	Params    map[string]any `json:"params,omitempty"`
	// Path — путь в ответе (board.go: /a/b, [поле=значение], /0).
	Path string `json:"path"`
	Note string `json:"note,omitempty"`
}

// ScenarioDef — карточка сценария «один к одному» (process/scenarios-flange.md).
type ScenarioDef struct {
	ID      string   `json:"id"`
	Version string   `json:"version"`
	Title   string   `json:"title"`
	Goal    string   `json:"goal,omitempty"`
	Kind    string   `json:"kind"` // situation | check | demo | failure | extra
	Case    CaseRefs `json:"case,omitempty"`
	// Run — прогон, в котором идёт карточка (MS-1, S10B, S13, S14, F-…).
	Run    string `json:"run"`
	Where  string `json:"where,omitempty"`
	Source string `json:"source,omitempty"` // откуда взято (документ процессной сессии, раздел)
	Steps  []Step `json:"steps"`
	// Outcome, MustNot — «что должно получиться» и «чего не должно случиться»
	// словами карточки; машинные утверждения — scenarios/expected/‹id›.yaml.
	Outcome []string `json:"outcome,omitempty"`
	MustNot []string `json:"must_not,omitempty"`
}

// CaseRefs — какие места кейса закрывает карточка.
type CaseRefs struct {
	S42   []string `json:"s4_2,omitempty"`
	S51   []string `json:"s5_1,omitempty"`
	Demo  []string `json:"demo,omitempty"`
	Other []string `json:"other,omitempty"`
}

// Step — строка таблицы карточки: факт через обычный приём, решение человека
// через API, служебное действие (сбой stand-а, подделка демо-инструментом),
// поток журнала оборудования или вывод системы (только для сверки глазами).
type Step struct {
	At    string `json:"at"`
	Label string `json:"label,omitempty"`
	Note  string `json:"note,omitempty"`
	Ref   string `json:"ref,omitempty"` // номер события каталога процессной сессии: «Е-44»

	// Факт (AD-26: stand → edge-агент → приём).
	Fact          string         `json:"fact,omitempty"`
	SchemaVersion int            `json:"schema_version,omitempty"`
	Source        string         `json:"source,omitempty"`
	Item          string         `json:"item,omitempty"`
	Lot           string         `json:"lot,omitempty"`
	Carrier       string         `json:"carrier,omitempty"` // tag | dm | none | ‹значение›
	Level         string         `json:"identification_level,omitempty"`
	Data          map[string]any `json:"data,omitempty"`
	// Mutate — сообщение не по контракту или с изменениями контракта (S12,
	// кейс §4.7): поля конверта убираются или заменяются после сборки.
	Mutate *Mutation `json:"mutate,omitempty"`
	// Skew — часы источника спешат: occurred_at = at + skew (S15).
	Skew string `json:"skew,omitempty"`
	// Deliver — когда сообщение дошло (опоздание, S07); пусто — сразу.
	Deliver string `json:"deliver,omitempty"`
	// Duplicates — повторные доставки того же сообщения (S06): моменты.
	Duplicates []string `json:"duplicates,omitempty"`
	// Conflict — повтор с тем же event_id и другим содержимым (S06, вариант).
	Conflict map[string]any `json:"conflict,omitempty"`
	// Lost — запись потеряна у источника: номер получен, сообщение не доставлено.
	Lost bool `json:"lost,omitempty"`
	// SeqReset — источник перезапущен: нумерация source_seq начинается заново с 1.
	SeqReset bool `json:"seq_reset,omitempty"`

	// Решение человека (AD-26: интерактивно — стол роли, автосверка — demo-signer).
	Decision string         `json:"decision,omitempty"` // operationId
	Role     string         `json:"role,omitempty"`
	Actor    string         `json:"actor,omitempty"`
	Stop     bool           `json:"stop,omitempty"`
	Params   map[string]any `json:"params,omitempty"`
	Body     map[string]any `json:"body,omitempty"`
	// Refusal — ожидаемый отказ (код контракта): шаг проверяет запрет.
	Refusal string `json:"refusal,omitempty"`

	// Служебное.
	Stand  *StandAction `json:"stand,omitempty"`
	Tamper *Tamper      `json:"tamper,omitempty"`
	Stream *Stream      `json:"stream,omitempty"`
	// Weld — сварка шагом карточки (переварка S10, отдельные прогоны S10B, S14):
	// подтверждение режима, начало, сводки источника по минутам, завершение.
	Weld *WeldStep `json:"weld,omitempty"`
	// Route — строка карточки, которую даёт маршрут изделия: «F-001:weld»
	// (карточка остаётся один к одному с таблицей, событие не дублируется).
	Route string `json:"route,omitempty"`
	// System — вывод системы из таблицы карточки (Е-80, Е-84…): его делает
	// движок; генератор его не шлёт, он проверяется утверждениями expected.
	System string `json:"system,omitempty"`
}

// Mutation — правка собранного конверта: пути вида «item_ref», «data/outcome».
type Mutation struct {
	Drop []string       `json:"drop,omitempty"`
	Set  map[string]any `json:"set,omitempty"`
	// Invalid — сообщение не проходит схемы контракта (генератор это проверяет).
	Invalid bool `json:"invalid,omitempty"`
	// Quarantine — приём должен положить сообщение в карантин (FR-30); иначе
	// принять с флагом UNKNOWN(значение) (AD-20, случай «неизвестное значение»).
	Quarantine bool `json:"quarantine,omitempty"`
}

// StandAction — служебный порт stand-а (AD-18): сбой или его снятие.
type StandAction struct {
	Stand  string `json:"stand"`            // onec | equipment:‹id› | …
	Fault  string `json:"fault,omitempty"`  // offline | error | delay | duplicate | drop | reorder | clock_skew | corrupt
	Param  int64  `json:"param,omitempty"`  // мс, код ответа или доля
	Match  string `json:"match,omitempty"`  // какое сообщение (бизнес-ключ, вид)
	Until  string `json:"until,omitempty"`  // до какого момента
	Clear  bool   `json:"clear,omitempty"`  // снять сбои stand-а
	Detail string `json:"detail,omitempty"` // «ошибка 422: не найден договор»
}

// Tamper — подделка в обход системы (S09): только демо-инструмент cmd/tamper
// с отдельным подключением суперпользователя БД (AD-26, AD-28).
type Tamper struct {
	Kind   string         `json:"kind"`   // update_in_place | update_and_rechain | projection_update
	Target string         `json:"target"` // метка события или представление
	Change map[string]any `json:"change,omitempty"`
}

// WeldStep — сварка шагом карточки.
type WeldStep struct {
	Item    string `json:"item"`
	Run     string `json:"run"` // SV-017-2
	Station string `json:"station"`
	Welder  string `json:"welder"`
	Minutes int    `json:"minutes"`
	// Current — ток по журналу, А: [мин, макс]; пусто — в уставке.
	Current  []int  `json:"current,omitempty"`
	ReworkOf string `json:"rework_of,omitempty"` // прежнее выполнение (FR-47)
	Program  string `json:"program,omitempty"`
	// NoConfirm — без отдельного «режим по карте сверен» перед сваркой
	// (сценарий показа: сварщик жмёт только «Начать» и «Выполнено»).
	NoConfirm bool `json:"no_confirm,omitempty"`
}

// Stream — поток записей журнала оборудования за окно (S04, S06, S07):
// сводки циклов по минутам во время сварок, сводки простоя до числа
// Records, отклонения на своё время; пачка доставляется в Deliver.
type Stream struct {
	Source    string `json:"source"`
	Equipment string `json:"equipment"`
	From      string `json:"from"`
	To        string `json:"to"`
	// Records — всего записей у источника за окно (вместе с потерянными).
	Records int `json:"records"`
	// Deliver — момент досылки пачки (связь восстановлена); DeliverSpanSec — за сколько секунд пачка ушла.
	Deliver        string `json:"deliver"`
	DeliverSpanSec int    `json:"deliver_span_s,omitempty"`
	// Duplicates — сколько записей пачки придут повторно (тот же номер).
	Duplicates int `json:"duplicates,omitempty"`
	// Lost — окно, записи которого потеряны у источника (буфер переполнен).
	Lost *LostWindow `json:"lost,omitempty"`
	// Deviations — записи «ток вне уставки» (equipment.deviation.detected).
	Deviations []Deviation `json:"deviations,omitempty"`
}

// LostWindow — потерянные записи и номер первой потерянной у источника.
type LostWindow struct {
	From     string `json:"from"`
	To       string `json:"to"`
	FirstSeq int64  `json:"first_seq,omitempty"`
}

// Deviation — отклонение параметра на своё время.
type Deviation struct {
	At        string `json:"at"`
	Run       string `json:"run"` // выполнение операции: SV-015-1
	Parameter string `json:"parameter"`
	Value     int    `json:"value"`
	Unit      string `json:"unit"`
	Label     string `json:"label,omitempty"`
}

// Bundle — загруженные определения одного прогона: мир, прогон и его карточки.
type Bundle struct {
	World     World
	Run       RunDef
	Scenarios []ScenarioDef
}

// Catalog — каталог сценариев scenarios/definitions/index.yaml: что можно
// запустить с пульта (FR-129) и какое место кейса закрывает каждый сценарий
// (8 ситуаций §4.2, 9 проверок §5.1, демо-сценарии PRD, сбои каталога).
type Catalog struct {
	Version string         `json:"version"`
	Title   string         `json:"title"`
	Source  []string       `json:"source,omitempty"`
	Entries []CatalogEntry `json:"scenarios"`
	// Coverage — место кейса → сценарии (для жюри и путеводителя, FR-117).
	Coverage map[string][]CoverageRow `json:"coverage"`
	// Reproduction — набор воспроизведения (FR-119, кейс §6.4): что в нём,
	// версии, допущения, параметры и команды запуска.
	Reproduction *Reproduction `json:"reproduction,omitempty"`
}

// Reproduction — набор воспроизведения сценариев (FR-119).
type Reproduction struct {
	Contents    []string `json:"contents"`
	Versions    []string `json:"versions"`
	Parameters  []string `json:"parameters"`
	Assumptions []string `json:"assumptions"`
	Commands    []string `json:"commands"`
}

// CatalogEntry — строка пульта: карточка или прогон целиком.
type CatalogEntry struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// Kind — situation (§4.2) | check (§5.1) | demo | failure | extra | run.
	Kind string `json:"kind"`
	// Run — определение прогона, которое запускается (runs/‹run›.yaml).
	Run string `json:"run"`
	// Show — сценарий показа: пульт по умолчанию показывает только их
	// (simulation.scenario.list; остальные — all=true и запуск по id).
	Show bool `json:"show,omitempty"`
	// Cards — карточки, чьи утверждения показывает табло (пусто — только своя).
	Cards    []string `json:"cards,omitempty"`
	CaseRefs []string `json:"case_refs,omitempty"`
	Note     string   `json:"note,omitempty"`
}

// CoverageRow — одно место кейса и сценарии, которые его показывают.
type CoverageRow struct {
	Case      string   `json:"case"`
	Scenarios []string `json:"scenarios"`
	Note      string   `json:"note,omitempty"`
}
