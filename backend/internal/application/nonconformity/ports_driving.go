package nonconformity

import (
	"context"

	"ant/internal/application/platform"
)

// QueueFilter — фильтр очереди «Ждут моего решения».
type QueueFilter struct {
	// Kind — presentation | signal | isolated; пусто — все.
	Kind string
	// Sort — risk | deadline (по умолчанию risk, затем срок).
	Sort string
}

// NCFilter — фильтр списка несоответствий.
type NCFilter struct {
	ItemID string
	Status string
}

// Queries — ведущий порт чтения модуля nonconformity (AD-36).
type Queries interface {
	// Queue — очередь «Ждут моего решения» (nonconformity.queue.list, PRD §3a).
	Queue(ctx context.Context, f QueueFilter, m platform.Moment, p platform.Page) (DecisionQueue, error)
	// Card — карточка несоответствия (nonconformity.card.read, FR-51).
	Card(ctx context.Context, ncID string, m platform.Moment) (NCCard, error)
	// List — несоответствия (nonconformity.nonconformity.list).
	List(ctx context.Context, f NCFilter, m platform.Moment, p platform.Page) (NCList, error)
	// Concessions — разрешения на отклонение, применимые к изделию (nonconformity.concession.list, FR-54).
	Concessions(ctx context.Context, itemID string, m platform.Moment) (ConcessionList, error)
	// Presentation — точка предъявления изделия для решения (nonconformity.presentation.read, FR-19).
	Presentation(ctx context.Context, itemID string, m platform.Moment) (NCPresentationView, error)
}

// Commands — ведущий порт команд модуля nonconformity (AD-39): решения
// контролёра и уполномоченных по изделию, несоответствию и сдерживанию.
type Commands interface {
	Confirm(ctx context.Context, ncID string, in ConfirmNonconformity) (platform.Receipt, error)
	RejectSignal(ctx context.Context, itemID string, in RejectSignal) (platform.Receipt, error)
	RequestRecheck(ctx context.Context, itemID string, in RequestRecheck) (platform.Receipt, error)
	Isolate(ctx context.Context, itemID string, in IsolateItem) (platform.Receipt, error)
	ResolvePresentation(ctx context.Context, itemID string, in ResolvePresentation) (platform.Receipt, error)
	ReviewPresentation(ctx context.Context, itemID string, in ReviewPresentation) (platform.Receipt, error)
	ResolveLot(ctx context.Context, lotID string, in ResolveLot) (platform.Receipt, error)
	SetDisposition(ctx context.Context, ncID string, in SetDisposition) (platform.Receipt, error)
	VerifyDisposition(ctx context.Context, ncID string, in VerifyDisposition) (platform.Receipt, error)
	SetContainment(ctx context.Context, itemID string, in SetContainment) (platform.Receipt, error)
	ReleaseContainment(ctx context.Context, itemID string, in ReleaseContainment) (platform.Receipt, error)
	RevokeConcession(ctx context.Context, concessionID string, in RevokeConcession) (platform.Receipt, error)
	GrantConcession(ctx context.Context, in GrantConcession) (platform.Receipt, error)
	WaiveReworkLimit(ctx context.Context, itemID string, in WaiveReworkLimit) (platform.Receipt, error)
	SetProcessHold(ctx context.Context, in SetProcessHold) (platform.Receipt, error)
	ReleaseProcessHold(ctx context.Context, holdID string, in ReleaseProcessHold) (platform.Receipt, error)
	Close(ctx context.Context, ncID string, in CloseNonconformity) (platform.Receipt, error)
}

// Unimplemented — заглушка портов nonconformity: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures.
type Unimplemented struct{}

func ni(op string) error { return platform.NotImplemented(op) }

func nr(op string) (platform.Receipt, error) { return platform.Receipt{}, ni(op) }

func (Unimplemented) Queue(context.Context, QueueFilter, platform.Moment, platform.Page) (DecisionQueue, error) {
	return DecisionQueue{}, ni("nonconformity.queue.list")
}
func (Unimplemented) Card(context.Context, string, platform.Moment) (NCCard, error) {
	return NCCard{}, ni("nonconformity.card.read")
}
func (Unimplemented) List(context.Context, NCFilter, platform.Moment, platform.Page) (NCList, error) {
	return NCList{}, ni("nonconformity.nonconformity.list")
}
func (Unimplemented) Concessions(context.Context, string, platform.Moment) (ConcessionList, error) {
	return ConcessionList{}, ni("nonconformity.concession.list")
}
func (Unimplemented) Presentation(context.Context, string, platform.Moment) (NCPresentationView, error) {
	return NCPresentationView{}, ni("nonconformity.presentation.read")
}
func (Unimplemented) Confirm(context.Context, string, ConfirmNonconformity) (platform.Receipt, error) {
	return nr("nonconformity.nonconformity.confirm")
}
func (Unimplemented) RejectSignal(context.Context, string, RejectSignal) (platform.Receipt, error) {
	return nr("nonconformity.signal.reject")
}
func (Unimplemented) RequestRecheck(context.Context, string, RequestRecheck) (platform.Receipt, error) {
	return nr("nonconformity.recheck.request")
}
func (Unimplemented) Isolate(context.Context, string, IsolateItem) (platform.Receipt, error) {
	return nr("nonconformity.item.isolate")
}
func (Unimplemented) ResolvePresentation(context.Context, string, ResolvePresentation) (platform.Receipt, error) {
	return nr("nonconformity.presentation.resolve")
}
func (Unimplemented) ReviewPresentation(context.Context, string, ReviewPresentation) (platform.Receipt, error) {
	return nr("nonconformity.presentation.review")
}
func (Unimplemented) ResolveLot(context.Context, string, ResolveLot) (platform.Receipt, error) {
	return nr("nonconformity.lot.resolve")
}
func (Unimplemented) SetDisposition(context.Context, string, SetDisposition) (platform.Receipt, error) {
	return nr("nonconformity.disposition.set")
}
func (Unimplemented) VerifyDisposition(context.Context, string, VerifyDisposition) (platform.Receipt, error) {
	return nr("nonconformity.disposition.verify")
}
func (Unimplemented) SetContainment(context.Context, string, SetContainment) (platform.Receipt, error) {
	return nr("nonconformity.containment.set")
}
func (Unimplemented) ReleaseContainment(context.Context, string, ReleaseContainment) (platform.Receipt, error) {
	return nr("nonconformity.containment.release")
}
func (Unimplemented) RevokeConcession(context.Context, string, RevokeConcession) (platform.Receipt, error) {
	return nr("nonconformity.concession.revoke")
}
func (Unimplemented) GrantConcession(context.Context, GrantConcession) (platform.Receipt, error) {
	return nr("nonconformity.concession.grant")
}
func (Unimplemented) WaiveReworkLimit(context.Context, string, WaiveReworkLimit) (platform.Receipt, error) {
	return nr("nonconformity.rework_limit.waive")
}
func (Unimplemented) SetProcessHold(context.Context, SetProcessHold) (platform.Receipt, error) {
	return nr("nonconformity.process_hold.set")
}
func (Unimplemented) ReleaseProcessHold(context.Context, string, ReleaseProcessHold) (platform.Receipt, error) {
	return nr("nonconformity.process_hold.release")
}
func (Unimplemented) Close(context.Context, string, CloseNonconformity) (platform.Receipt, error) {
	return nr("nonconformity.nonconformity.close")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
