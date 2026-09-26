package reference

import (
	"slices"
	"time"
)

// Оборудование и поверка на дату (FR-17, ГОСТ Р 58876 п. 7.1.5.2): операция
// на оборудовании, чья поверка на дату операции истекла, блокируется
// предусловием процесса. Функции ниже отвечают «действует ли оборудование на
// t» и дают интервалы действия — вход предусловий процесса (domain/process
// сравнивает дату операции с интервалами).

// Причины недействующего оборудования (EquipmentStatus.Reason).
const (
	ReasonUnknown            = "unknown"              // нет в справочнике на эту дату
	ReasonNotVerified        = "not_verified"         // средство измерений без поверки
	ReasonVerificationFailed = "verification_invalid" // последняя поверка — «непригодно»
	ReasonExpired            = "verification_expired" // срок поверки истёк
)

// EquipmentStatus — оборудование на момент t.
type EquipmentStatus struct {
	Equipment Equipment
	// Verification — последняя поверка не позже t; nil — не было.
	Verification *Verification
	// VerifiedUntil — конец действия поверки (конец дня valid_until по
	// местному времени); nil — поверки нет или она бессрочна.
	VerifiedUntil *time.Time
	// Valid — оборудование можно использовать на t.
	Valid bool
	// Reason — почему нельзя (Reason*), пусто — можно.
	Reason string
}

// EquipmentAt — версия оборудования, действующая на t.
func (b Book) EquipmentAt(id string, t time.Time) (Equipment, bool) {
	return governing(b.Equipment, id, func(x Equipment) string { return string(x.Data.EquipmentID) },
		func(x Equipment) Meta { return x.Meta }, func(x Equipment) Validity { return x.Validity }, t)
}

// EquipmentIDs — все оборудование среза (по id).
func (b Book) EquipmentIDs() []string {
	return sortedKeys(b.Equipment, func(x Equipment) string { return string(x.Data.EquipmentID) })
}

// verifications — поверки оборудования по (At, seq).
func (b Book) verifications(id string) []Verification {
	var vs []Verification
	for _, v := range b.Verifications {
		if string(v.Data.EquipmentID) == id {
			vs = append(vs, v)
		}
	}
	slices.SortStableFunc(vs, func(a, c Verification) int {
		if !a.At.Equal(c.At) {
			return a.At.Compare(c.At)
		}
		return cmpInt(a.Seq, c.Seq)
	})
	return vs
}

// VerificationEnd — конец действия поверки: начало следующих местных суток
// после valid_until («действует до 31.03.2027» — весь день 31 марта). Нет
// даты — nil (бессрочно).
func VerificationEnd(v Verification) *time.Time {
	if v.Data.ValidUntil == nil {
		return nil
	}
	d, err := time.ParseInLocation(DateLayout, string(*v.Data.ValidUntil), Local)
	if err != nil {
		return nil
	}
	end := d.AddDate(0, 0, 1).UTC()
	return &end
}

// EquipmentStatusAt — действует ли оборудование на t (FR-17): есть в
// справочнике на эту дату; если это средство измерений или у него вообще
// ведётся поверка — последняя поверка не позже t пригодна и не истекла.
func (b Book) EquipmentStatusAt(id string, t time.Time) EquipmentStatus {
	eq, ok := b.EquipmentAt(id, t)
	st := EquipmentStatus{Equipment: eq}
	if !ok {
		st.Reason = ReasonUnknown
		return st
	}
	vs := b.verifications(id)
	var last *Verification
	for i := range vs {
		if vs[i].At.After(t) {
			break
		}
		last = &vs[i]
	}
	if !eq.Data.IsMeasuringInstrument && len(vs) == 0 {
		st.Valid = true
		return st
	}
	if last == nil {
		st.Reason = ReasonNotVerified
		return st
	}
	v := *last
	st.Verification = &v
	st.VerifiedUntil = VerificationEnd(v)
	switch {
	case v.Data.Result != "valid":
		st.Reason = ReasonVerificationFailed
	case st.VerifiedUntil != nil && !t.Before(*st.VerifiedUntil):
		st.Reason = ReasonExpired
	default:
		st.Valid = true
	}
	return st
}

// EquipmentValidity — интервалы, когда оборудование действует (по срезу):
// между соседними границами (даты действия версий, моменты поверок и концы
// их сроков) состояние постоянно, поэтому достаточно проверить начало каждого
// отрезка. Вход предусловий процесса (process.Reference.Equipment).
func (b Book) EquipmentValidity(id string) []Validity {
	var bounds []time.Time
	for _, e := range b.Equipment {
		if string(e.Data.EquipmentID) != id {
			continue
		}
		bounds = append(bounds, e.From)
		if e.Until != nil {
			bounds = append(bounds, *e.Until)
		}
	}
	for _, v := range b.verifications(id) {
		bounds = append(bounds, v.At)
		if end := VerificationEnd(v); end != nil {
			bounds = append(bounds, *end)
		}
	}
	return intervals(bounds, func(t time.Time) bool { return b.EquipmentStatusAt(id, t).Valid })
}

// intervals — объединение отрезков между границами bounds, на которых
// выполняется pred (значение на отрезке — по его началу); последний отрезок
// бессрочный.
func intervals(bounds []time.Time, pred func(time.Time) bool) []Validity {
	slices.SortFunc(bounds, func(a, c time.Time) int { return a.Compare(c) })
	bounds = slices.CompactFunc(bounds, func(a, c time.Time) bool { return a.Equal(c) })
	var out []Validity
	open := false
	for i, t := range bounds {
		ok := pred(t)
		switch {
		case ok && !open:
			out = append(out, Validity{From: t})
			open = true
		case !ok && open:
			end := bounds[i]
			out[len(out)-1].Until = &end
			open = false
		}
	}
	return out
}

// QualificationValidity — интервалы действия квалификаций по сотрудникам
// (FR-17, FR-80): person → [(qualification_id, интервал)]. Выдача действует
// [valid_from, valid_until); отзыв, записанный позже выдачи, прекращает её с
// effective_from. ok=false — в срезе нет ни одной записи о квалификациях
// (справочник не подключён — «не проверено», а не «нарушено»).
func (b Book) QualificationValidity() (map[string][]QualificationInterval, bool) {
	if len(b.Qualifications) == 0 && len(b.Revocations) == 0 {
		return nil, false
	}
	out := map[string][]QualificationInterval{}
	for _, g := range b.Qualifications {
		v := validity(g.Data.ValidFrom, g.Data.ValidUntil)
		for _, r := range b.Revocations {
			if r.Data.PersonID != g.Data.PersonID || r.Data.QualificationID != g.Data.QualificationID || r.Seq < g.Seq {
				continue
			}
			end := r.Data.EffectiveFrom.Time().UTC()
			if v.Until == nil || end.Before(*v.Until) {
				v.Until = &end
			}
		}
		if v.Until != nil && !v.From.Before(*v.Until) {
			continue
		}
		p := string(g.Data.PersonID)
		out[p] = append(out[p], QualificationInterval{Qualification: string(g.Data.QualificationID), Validity: v})
	}
	return out, true
}

// QualificationInterval — квалификация с интервалом действия.
type QualificationInterval struct {
	Qualification string
	Validity
}

// LotUsableAt — партия годна на t по сроку годности (FR-17: «сроки годности
// материалов и их партий»): срок — до конца дня expiry_date по местному
// времени. ok=false — партии нет в срезе.
func (b Book) LotUsableAt(lotID string, t time.Time) (usable bool, ok bool) {
	for _, l := range b.LotsLatest() {
		if string(l.Data.LotID) != lotID {
			continue
		}
		if l.Data.ExpiryDate == nil {
			return true, true
		}
		d, err := time.ParseInLocation(DateLayout, string(*l.Data.ExpiryDate), Local)
		if err != nil {
			return false, true
		}
		return t.Before(d.AddDate(0, 0, 1)), true
	}
	return false, false
}
