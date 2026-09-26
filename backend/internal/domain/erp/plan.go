package erp

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Env — окружение правил модуля erp: цеха и склады процесса и учётная
// система, в которую уходят сообщения (включённые системы — конфигурация, AD-18).
type Env struct {
	Topology Topology
	// System — учётная система: onec (по умолчанию) или galaktika.
	System string
}

func (e Env) system() ev.ErpPostingRequestedV1ExternalSystem {
	if e.System == "" {
		return ev.ErpPostingRequestedV1ExternalSystemOnec
	}
	return ev.ErpPostingRequestedV1ExternalSystem(e.System)
}

// Trigger — запись журнала, на которую модуль erp формирует учётные
// сообщения: сама запись и, для реакции процесса, её слот (AD-3) — новая
// версия того же слота даёт тот же бизнес-ключ.
type Trigger struct {
	Record kernel.Record
	// Slot — ключ слота реакции-триггера (kernel.Slot.Key); пусто у фактов и решений.
	Slot string
}

// Draft — учётное сообщение до версии: бизнес-ключ, действие и содержимое.
type Draft struct {
	Key    string
	Action Action
	Data   ev.ErpPostingRequestedV1
	Cause  kernel.Record
}

// Subject — субъект записи-триггера: изделие или партия (пусто — запись не
// меняет учёт ни одного субъекта).
func Subject(r kernel.Record) (string, bool, error) {
	switch r.Type {
	case catalog.ErpLotReceived:
		var d ev.ErpLotReceivedV1
		return string(d.LotID), true, json.Unmarshal(r.Data, &d)
	case catalog.GenealogyLotIssued:
		var d ev.GenealogyLotIssuedV1
		return string(d.LotID), true, json.Unmarshal(r.Data, &d)
	case catalog.DecisionLotResolved:
		var d ev.DecisionLotResolvedV1
		return string(d.LotID), true, json.Unmarshal(r.Data, &d)
	case catalog.ItemItemRegistered:
		var d ev.ItemItemRegisteredV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return "", false, err
		}
		return string(d.ItemID), false, nil
	}
	if Is(r.Type, Triggers) {
		return r.ItemID, false, nil
	}
	return "", false, nil
}

// Plan — учётные сообщения, которые даёт запись-триггер, и новый учёт
// субъекта (чистая функция, AD-4). Сообщения уходят только на закрывающих
// точках (кейс §3.3): по событиям-сообщениям процесса «в 1С» (BPMN),
// решениям на закрывающих точках (результат контроля), решению по партии и
// проверке исполнения переделки (возврат из брака, Д-17). Повторное решение о
// переделке, пока изделие в браке, нового движения не даёт; возврат из
// брака без брака не уходит.
func Plan(env Env, b Books, t Trigger) ([]Draft, Books, error) {
	r := t.Record
	switch r.Type {
	case catalog.ItemItemRegistered:
		var d ev.ItemItemRegisteredV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return nil, b, err
		}
		b.ItemTypeID = string(d.ItemTypeID)
		if d.OrderID != nil {
			b.OrderID = string(*d.OrderID)
		}
		return nil, b, nil
	case catalog.ErpLotReceived:
		var d ev.ErpLotReceivedV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return nil, b, err
		}
		b.Lot, b.ItemTypeID = true, string(d.ItemTypeID)
		b.Received += d.Quantity
		if b.Warehouse == "" {
			b.Warehouse = env.Topology.FirstWarehouse()
		}
		return nil, b, nil
	case catalog.GenealogyLotIssued:
		var d ev.GenealogyLotIssuedV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return nil, b, err
		}
		b.Lot = true
		b.Issued += d.Quantity
		return nil, b, nil
	case catalog.DecisionDispositionSet:
		var d ev.DecisionDispositionSetV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return nil, b, err
		}
		x := &Disposition{NCID: string(d.NcID), Disposition: string(d.Disposition)}
		if d.ClaimBasis != nil {
			x.ClaimBasis = *d.ClaimBasis
		}
		if d.ConcessionID != nil {
			x.ConcessionID = string(*d.ConcessionID)
			if d.Disposition == ev.DecisionDispositionSetV1DispositionUseAsIs || d.Disposition == ev.DecisionDispositionSetV1DispositionRepair {
				b.ConcessionID = x.ConcessionID
			}
		}
		b.Disposition = x
		return nil, b, nil
	case catalog.OperationMessageThrown:
		return planThrown(env, b, t)
	case catalog.DecisionPresentationResolved:
		return planPresentation(env, b, r)
	case catalog.DecisionLotResolved:
		return planLot(env, b, r)
	case catalog.DecisionDispositionVerified:
		if !b.InDefect || b.DefectAction != ScrapRework {
			return nil, b, nil
		}
		var d ev.DecisionDispositionVerifiedV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return nil, b, err
		}
		return returnFromDefect(env, b, r, "", []string{string(d.NcID)})
	}
	return nil, b, nil
}

func planThrown(env Env, b Books, t Trigger) ([]Draft, Books, error) {
	r := t.Record
	if r.ItemID == "" {
		return nil, b, nil
	}
	var d ev.OperationMessageThrownV1
	if err := json.Unmarshal(r.Data, &d); err != nil {
		return nil, b, err
	}
	step := string(d.StepKey)
	a := Action(d.ErpAction)
	if !a.Movement() {
		return nil, b, fmt.Errorf("erp: событие-сообщение %s: учётное действие %q не из порта учёта", step, d.ErpAction)
	}
	point := step
	if n := b.occurrence(step, t.Slot); n > 1 {
		point = step + ".r" + strconv.Itoa(n)
	}
	basis := append([]string{r.EventID}, uuids(d.ClosingBasis)...)
	topo := env.Topology
	lane, inLane := topo.LaneOf(step)
	switch a {
	case AcceptIntoWork:
		// Выдача годного в производство (ЗТ-1): со склада приёмки на склад
		// цеха, который начинает обработку (следующая дорожка).
		to := topo.StepWarehouse(step)
		if inLane && lane == 0 {
			to = topo.Warehouse(1)
		}
		dr := draft(env, b, a, point, step, basis, r)
		dr.Data.FromWarehouseID, dr.Data.ToWarehouseID = oid(topo.FirstWarehouse()), oid(to)
		if b.Warehouse == "" {
			b.Warehouse = to
		}
		return []Draft{dr}, b, nil
	case WarehouseTransfer, Release:
		// Смена склада при передаче между цехами (FR-130): на склад цеха-
		// получателя (дорожка события) со склада, где изделие числится; если
		// прежних сообщений нет — со склада предыдущей дорожки.
		to := topo.StepWarehouse(step)
		from := b.Warehouse
		if from == "" || from == to {
			if inLane {
				from = topo.Warehouse(lane - 1)
			}
		}
		dr := draft(env, b, a, point, step, basis, r)
		dr.Data.FromWarehouseID, dr.Data.ToWarehouseID = oid(from), oid(to)
		if a == Release {
			rw := b.Reworked
			dr.Data.AfterRework = &rw
			dr.Data.ConcessionID = oid(b.ConcessionID)
		}
		b.Warehouse = to
		return []Draft{dr}, b, nil
	case ScrapRework, ScrapWriteoff, ScrapReprocess:
		if b.InDefect && b.DefectAction == a {
			// Повторное решение того же вида, пока изделие в браке, нового
			// движения не даёт (каталог процессной сессии, ЗТ-Р).
			return nil, b, nil
		}
		if !b.InDefect {
			b.DefectCycle++
		}
		dr := draft(env, b, a, defectPoint(b.DefectCycle), step, basis, r)
		dr.Data.FromWarehouseID = oid(b.Warehouse)
		dr.Data.NcIds = ncIDs(b.Disposition)
		b.InDefect, b.DefectAction = true, a
		return []Draft{dr}, b, nil
	case ReturnToSupplier:
		dr := draft(env, b, a, point, step, basis, r)
		dr.Data.FromWarehouseID = oid(b.Warehouse)
		dr.Data.NcIds = ncIDs(b.Disposition)
		if b.Disposition != nil && b.Disposition.ClaimBasis != "" {
			cb := b.Disposition.ClaimBasis
			dr.Data.ClaimBasis = &cb
		}
		return []Draft{dr}, b, nil
	case ReturnFromDefect:
		if !b.InDefect {
			return nil, b, nil
		}
		return returnFromDefect(env, b, r, step, basis[1:])
	}
	return nil, b, nil
}

// returnFromDefect — «возврат из брака в производство» (Д-17): один на цикл
// брака; его дают и событие-сообщение процесса, и проверка исполнения
// переделки — бизнес-ключ у обоих один, второе ничего не добавляет (AD-7).
func returnFromDefect(env Env, b Books, r kernel.Record, step string, extra []string) ([]Draft, Books, error) {
	basis := append([]string{r.EventID}, extra...)
	dr := draft(env, b, ReturnFromDefect, defectPoint(b.DefectCycle), step, nil, r)
	dr.Data.BasisEventIds = uuidsOf(basis)
	dr.Data.ToWarehouseID = oid(b.Warehouse)
	dr.Data.NcIds = ncIDs(b.Disposition)
	b.InDefect, b.DefectAction, b.Reworked = false, "", true
	return []Draft{dr}, b, nil
}

func planPresentation(env Env, b Books, r kernel.Record) ([]Draft, Books, error) {
	if r.ItemID == "" {
		return nil, b, nil
	}
	var d ev.DecisionPresentationResolvedV1
	if err := json.Unmarshal(r.Data, &d); err != nil {
		return nil, b, err
	}
	point := b.decisionPoint(r.EventID, r.Corrects, d.ClosingPoint+".p"+strconv.Itoa(d.PresentationNo))
	dr := draft(env, b, InspectionResult, point, string(d.StepKey), append([]string{r.EventID}, uuids(d.MethodEventIds)...), r)
	res := ev.ErpPostingRequestedV1Resolution(d.Resolution)
	dr.Data.Resolution = &res
	n := d.PresentationNo
	dr.Data.PresentationNo = &n
	if d.ConcessionID != nil {
		dr.Data.ConcessionID = oid(string(*d.ConcessionID))
		if d.Resolution == ev.DecisionPresentationResolvedV1ResolutionAcceptWithConcession {
			b.ConcessionID = string(*d.ConcessionID)
		}
	}
	cp := d.ClosingPoint
	dr.Data.ClosingPoint = &cp
	return []Draft{dr}, b, nil
}

func planLot(env Env, b Books, r kernel.Record) ([]Draft, Books, error) {
	var d ev.DecisionLotResolvedV1
	if err := json.Unmarshal(r.Data, &d); err != nil {
		return nil, b, err
	}
	b.Lot = true
	basis := append([]string{r.EventID}, uuids(d.MethodEventIds)...)
	point := b.decisionPoint(r.EventID, r.Corrects, "ZT-1")
	ir := draft(env, b, InspectionResult, point, "", basis, r)
	res := ev.ErpPostingRequestedV1Resolution(d.Resolution)
	ir.Data.Resolution = &res
	zt := "ZT-1"
	ir.Data.ClosingPoint = &zt
	out := []Draft{ir}
	if d.Resolution == ev.DecisionLotResolvedV1ResolutionReject {
		// Брак партии на входном контроле → «возврат поставщику» остатка
		// партии с основанием претензии (ЗТ-1 → ЗТ-Р).
		ret := draft(env, b, ReturnToSupplier, point, "", basis, r)
		ret.Data.FromWarehouseID = oid(b.Warehouse)
		if q := b.Remaining(); q > 0 {
			ret.Data.Quantity = &q
		}
		if d.Reason != nil && d.Reason.Text != "" {
			cb := string(d.Reason.Text)
			ret.Data.ClaimBasis = &cb
		}
		ret.Data.ClosingPoint = &zt
		out = append(out, ret)
	}
	return out, b, nil
}

// draft — сообщение по субъекту учёта b: бизнес-ключ, действие, точка, шаг, основание.
func draft(env Env, b Books, a Action, point, step string, basis []string, cause kernel.Record) Draft {
	key := BusinessKey(b.Subject, a, point)
	d := ev.ErpPostingRequestedV1{BusinessKey: key, ExternalSystem: env.system(), Action: ev.ErpPostingRequestedV1Action(a)}
	if b.Lot {
		d.LotID = oid(b.Subject)
	} else {
		it := ev.ItemID(b.Subject)
		d.ItemID = &it
	}
	if point != "" {
		p := point
		d.ClosingPoint = &p
	}
	if step != "" {
		s := ev.StepKey(step)
		d.StepKey = &s
	}
	d.OrderID = oid(b.OrderID)
	if len(basis) > 0 {
		d.BasisEventIds = uuidsOf(basis)
	}
	return Draft{Key: key, Action: a, Data: d, Cause: cause}
}

func defectPoint(n int) string { return "defect." + strconv.Itoa(n) }

// BusinessKey — бизнес-ключ исходящего сообщения (AD-7): субъект, учётное
// действие, закрывающая точка — `‹субъект›/‹действие›/‹точка›`. Ключ входит
// в поток `erp_message:‹ключ›` (не длиннее 128 знаков из [A-Za-z0-9._:/@-]):
// слишком длинную точку, а затем и субъект заменяет их UUIDv5.
func BusinessKey(subject string, a Action, point string) string {
	clean := func(s string) string {
		return strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._:/@-", r) {
				return r
			}
			return '_'
		}, s)
	}
	short := func(s string) string {
		return "h" + strings.ReplaceAll(kernel.UUIDv5(constants.NsAnt, s), "-", "")[:16]
	}
	subject, point = clean(subject), clean(point)
	key := subject + "/" + string(a) + "/" + point
	if len(key) > 128 {
		point = short(point)
		key = subject + "/" + string(a) + "/" + point
	}
	if len(key) > 128 {
		key = short(subject) + "/" + string(a) + "/" + point
	}
	return key
}

// Stream — поток сообщения `erp_message:‹бизнес-ключ›` (AD-39).
func Stream(key string) string { return "erp_message:" + key }

// MessageID — номер сообщения для учётной системы (`X-Message-Id`): UUIDv5
// от бизнес-ключа и версии; повтор и ручная переотправка — с тем же номером.
func MessageID(key string, version int) string {
	return kernel.UUIDv5(constants.NsAnt, "erp.message\x1f"+key+"\x1f"+strconv.Itoa(version))
}

func oid(s string) *ev.ObjectID {
	if s == "" {
		return nil
	}
	x := ev.ObjectID(s)
	return &x
}

func ncIDs(d *Disposition) []ev.ObjectID {
	if d == nil || d.NCID == "" {
		return nil
	}
	return []ev.ObjectID{ev.ObjectID(d.NCID)}
}

func uuids(xs []ev.UUID) []string {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		out = append(out, string(x))
	}
	return out
}

func uuidsOf(xs []string) []ev.UUID {
	var out []ev.UUID
	seen := map[string]bool{}
	for _, x := range xs {
		if x == "" || seen[x] {
			continue
		}
		seen[x] = true
		out = append(out, ev.UUID(x))
	}
	return out
}
