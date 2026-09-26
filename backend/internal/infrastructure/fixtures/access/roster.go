package access

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"time"

	app "ant/internal/application/access"
	"ant/internal/application/platform"
	refapp "ant/internal/application/reference"
	"ant/internal/contracts/errcodes"
	"ant/internal/infrastructure/fixtures/loader"
)

// План смены на заготовках (интерфейс 6; FR-80, FR-81): назначения на посты,
// кандидаты на пост и команды «Назначить на пост» / «Снять с поста». План —
// из мира заготовок (access.assignment.list), команды мастера меняют его
// сессионным наложением (loader.Runtime.Record): факт в памяти процесса
// поверх ответа мира, до перезапуска или нового прогона пульта.

const (
	opAssignmentSet   = "access.assignment.set"
	opAssignmentClear = "access.assignment.clear"
	kindWorkplace     = "workplace"
)

// plan — план смен мира с наложением сессии на момент m.
func plan(ctx context.Context, m *platform.Moment) (app.AccessAssignmentList, error) {
	v, err := respond[app.AccessAssignmentList](ctx, "access.assignment.list", nil, m)
	if err != nil {
		return v, err
	}
	rt, err := runtime()
	if err != nil {
		return v, err
	}
	for _, f := range rt.Facts(ctx, m, kindWorkplace) {
		switch b := f.Body.(type) {
		case app.SetAssignment:
			v.Items = slices.DeleteFunc(v.Items, func(a app.AccessAssignment) bool {
				return a.WorkplaceID == f.ID && a.ShiftID == b.ShiftID && a.PersonID == b.PersonID
			})
			v.Items = append(v.Items, app.AccessAssignment{WorkplaceID: f.ID, ShiftID: b.ShiftID, PersonID: b.PersonID, AssigneeRole: b.AssigneeRole,
				ApprovalDocumentID: b.ApprovalDocumentID, QualificationOK: true})
			v.BasisSeq = max(v.BasisSeq, f.Seq)
		case app.ClearAssignment:
			v.Items = slices.DeleteFunc(v.Items, func(a app.AccessAssignment) bool {
				return a.WorkplaceID == f.ID && a.ShiftID == b.ShiftID && a.PersonID == b.PersonID
			})
			v.BasisSeq = max(v.BasisSeq, f.Seq)
		}
	}
	return v, nil
}

// shiftsOf — смены справочника мира заготовок в окне шага (reference.shift.list).
func shiftsOf(ctx context.Context, m *platform.Moment) []refapp.RefShift {
	v, err := respond[refapp.RefShiftList](ctx, "reference.shift.list", nil, m)
	if err != nil {
		return nil
	}
	return v.Items
}

// resolveShifts — смены плана по параметру: `‹шаблон›@дата` — эта смена;
// `‹шаблон›` — ближайшая такая (идущая или следующая); пусто — идущие смены,
// а между сменами — следующая.
func resolveShifts(ctx context.Context, shiftID string, m *platform.Moment) map[string]bool {
	if strings.Contains(shiftID, "@") {
		return map[string]bool{shiftID: true}
	}
	now := time.Now()
	if rt, err := runtime(); err == nil {
		if t, err := rt.Clock(ctx); err == nil {
			now = t
		}
	}
	if m != nil && m.AsOf != nil {
		now = *m.AsOf
	}
	out := map[string]bool{}
	var next *refapp.RefShift
	shifts := shiftsOf(ctx, m)
	for i, s := range shifts {
		base, _, _ := strings.Cut(s.ShiftID, "@")
		if shiftID != "" && base != shiftID {
			continue
		}
		if !s.StartsAt.After(now) && s.EndsAt.After(now) {
			out[s.ShiftID] = true
		}
		if s.StartsAt.After(now) && (next == nil || s.StartsAt.Before(next.StartsAt)) {
			next = &shifts[i]
		}
	}
	if len(out) == 0 && next != nil {
		out[next.ShiftID] = true
	}
	return out
}

// workshopOf — цех поста по панели «Посты» мира заготовок.
func workshopOf(ctx context.Context, m *platform.Moment) map[string]string {
	out := map[string]string{}
	if v, err := respond[app.PostList](ctx, "access.workplace.list", nil, m); err == nil {
		for _, r := range v.Items {
			out[r.WorkplaceID] = r.Workshop
		}
	}
	return out
}

// Assignments — назначения на посты в смене (access.assignment.list, FR-81):
// смена — `‹шаблон›@дата`, `‹шаблон›` (ближайшая) или пусто (идущая); цех — WS-….
func (Adapter) Assignments(ctx context.Context, shiftID, workshop string, m platform.Moment) (app.AccessAssignmentList, error) {
	v, err := plan(ctx, &m)
	if err != nil {
		return v, err
	}
	want := resolveShifts(ctx, shiftID, &m)
	ws := map[string]string{}
	if workshop != "" {
		ws = workshopOf(ctx, &m)
	}
	out := app.AccessAssignmentList{Items: []app.AccessAssignment{}, BasisSeq: v.BasisSeq}
	for _, a := range v.Items {
		if want[a.ShiftID] && (workshop == "" || ws[a.WorkplaceID] == workshop) {
			out.Items = append(out.Items, a)
		}
	}
	return out, nil
}

// Candidates — кандидаты на пост в смене (access.candidate.list): вердикты
// квалификации — из мира заготовок, «уже назначен» — по плану смены с наложением.
func (a Adapter) Candidates(ctx context.Context, workplaceID, shiftID string, m platform.Moment) (app.WorkplaceCandidateList, error) {
	v, err := respond[app.WorkplaceCandidateList](ctx, "access.candidate.list", map[string]string{"workplace_id": workplaceID}, &m)
	if err != nil {
		return v, err
	}
	as, err := a.Assignments(ctx, shiftID, "", m)
	if err != nil {
		return v, err
	}
	v.ShiftID = shiftID
	if shiftID == "" {
		for _, x := range as.Items {
			v.ShiftID = x.ShiftID
			break
		}
	}
	v.BasisSeq = max(v.BasisSeq, as.BasisSeq)
	for i := range v.Items {
		c := &v.Items[i]
		for _, x := range as.Items {
			if x.PersonID != c.PersonID {
				continue
			}
			if x.WorkplaceID == workplaceID {
				c.AssignedHere = true
			} else {
				c.AssignedElsewhere = x.WorkplaceID
			}
		}
	}
	app.SortCandidates(v.Items)
	return v, nil
}

// SetAssignment — назначить на пост (access.assignment.set): гард как у live
// (роль в области поста, квалификация исполнителя, согласование контролёра)
// по кандидатам мира заготовок; назначение ложится в план смены сессии.
func (a Adapter) SetAssignment(ctx context.Context, in app.SetAssignment) (platform.Receipt, error) {
	if strings.TrimSpace(in.ShiftID) == "" {
		return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "shift_id", "reason", "смена")
	}
	cands, err := a.Candidates(ctx, in.WorkplaceID, in.ShiftID, platform.Moment{})
	if err != nil {
		return platform.Receipt{}, err
	}
	i := slices.IndexFunc(cands.Items, func(c app.WorkplaceCandidate) bool { return c.PersonID == in.PersonID })
	switch {
	case i < 0 || cands.Items[i].Role != in.AssigneeRole:
		return platform.Receipt{}, platform.Fail(errcodes.AccessWrongWorkplace, "workplace", in.WorkplaceID)
	case in.AssigneeRole == "quality_inspector" && in.ApprovalDocumentID == "":
		return platform.Receipt{}, platform.Fail(errcodes.AccessControllerApprovalRequired, "workplace", in.WorkplaceID)
	case !cands.Items[i].Allowed:
		return platform.Receipt{}, platform.Fail(errcodes.AccessNotQualified, "person", in.PersonID, "workplace", in.WorkplaceID)
	}
	return record(ctx, opAssignmentSet, in.WorkplaceID, in.CommandMeta(), in)
}

// ClearAssignment — снять с поста (access.assignment.clear): назначение
// должно быть в плане смены (с наложением сессии).
func (Adapter) ClearAssignment(ctx context.Context, in app.ClearAssignment) (platform.Receipt, error) {
	v, err := plan(ctx, nil)
	if err != nil {
		return platform.Receipt{}, err
	}
	if !slices.ContainsFunc(v.Items, func(a app.AccessAssignment) bool {
		return a.WorkplaceID == in.WorkplaceID && a.ShiftID == in.ShiftID && a.PersonID == in.PersonID
	}) {
		return platform.Receipt{}, platform.Fail(errcodes.ApiNotFound, "object", "назначение", "id", in.PersonID+" · "+in.WorkplaceID+" · "+in.ShiftID)
	}
	return record(ctx, opAssignmentClear, in.WorkplaceID, in.CommandMeta(), in)
}

// record — команда с сессионным наложением: факт над постом; живые
// обновления — пост и панель «Посты».
func record(ctx context.Context, op, workplaceID string, meta platform.CommandMeta, body any) (platform.Receipt, error) {
	rt, err := runtime()
	if err != nil {
		return platform.Receipt{}, err
	}
	return rt.Record(ctx, op, loader.ObjectRef{Kind: kindWorkplace, ID: workplaceID}, meta, body,
		loader.Change{Entity: kindWorkplace, ID: "global"})
}

// planned — назначение плана текущей смены с именем сотрудника.
type planned struct {
	app.AccessAssignment
	display string
}

// sessionPlan — план идущей смены по постам, если в сессии есть назначения
// мастера (иначе ok = false: панель и карточка — как в мире заготовок).
func (a Adapter) sessionPlan(ctx context.Context, m platform.Moment) (map[string][]planned, bool) {
	rt, err := runtime()
	if err != nil || len(rt.Facts(ctx, &m, kindWorkplace)) == 0 {
		return nil, false
	}
	as, err := a.Assignments(ctx, "", "", m)
	if err != nil {
		return nil, false
	}
	names := map[string]string{}
	if ps, err := respond[app.DemoPersonaList](ctx, "access.persona.list", nil, nil); err == nil {
		for _, p := range ps.Items {
			names[p.ID] = p.Name
		}
	}
	out := map[string][]planned{}
	for _, x := range as.Items {
		out[x.WorkplaceID] = append(out[x.WorkplaceID], planned{AccessAssignment: x, display: cmp.Or(names[x.PersonID], x.PersonID)})
	}
	return out, true
}

// overlayRow — строка панели «Посты» по плану смены с наложением: назначен
// другой — присутствие неизвестно до данных СКУД и ключа; никого — «не назначен».
func overlayRow(r *app.PostRow, list []planned) {
	if len(list) == 0 {
		r.Assigned, r.Presence = nil, "not_assigned"
		return
	}
	if r.Assigned != nil && slices.ContainsFunc(list, func(p planned) bool { return p.PersonID == r.Assigned.PersonID }) {
		return
	}
	r.Assigned = &app.PostPerson{PersonID: list[0].PersonID, Display: list[0].display}
	r.Presence = "unknown"
}
