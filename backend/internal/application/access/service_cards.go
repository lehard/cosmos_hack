package access

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	accessdom "ant/internal/domain/access"
)

// Карточки окон «Пост» и «Сотрудник» раздела «Посты» (UI-16; FR-6, FR-81):
// отдельные чтения с минимальными данными, чтобы руководителю и мастеру не
// выдавать чтение учётных записей (access.person.read) и всего журнала
// (journal.entry.list).

// WorkplaceLog — ведомый порт записей потока поста `workplace:‹id›` из журнала
// (назначения, токен, допуск, отклонения присутствия): адаптер —
// infrastructure/storage/access над JournalStore и кодеком записей.
type WorkplaceLog interface {
	// Stream — записи потока поста по возрастанию seq на момент m (AD-22, AD-38).
	Stream(ctx context.Context, workplaceID string, m platform.Moment) ([]accessdom.Record, error)
}

// WithWorkplaceLog подключает историю поста из журнала (режим live).
func WithWorkplaceLog(l WorkplaceLog) Option { return func(s *Service) { s.wplog = l } }

// WorkplaceCard — карточка поста (access.workplace.read): строка панели «Посты»
// и назначения текущей смены поста. В режиме fixtures — у заготовок.
func (s *Service) WorkplaceCard(ctx context.Context, workplaceID string, m platform.Moment) (WorkplaceCard, error) {
	if !s.live || s.policy == nil || s.dir == nil || len(s.dir.Workplaces) == 0 {
		return s.Queries.WorkplaceCard(ctx, workplaceID, m)
	}
	wp, ok := s.dir.Workplace(workplaceID)
	if !ok {
		return WorkplaceCard{}, platform.Fail(errcodes.ApiNotFound, "object", "пост", "id", workplaceID)
	}
	posts, err := s.Workplaces(ctx, wp.Workshop, m)
	if err != nil {
		return WorkplaceCard{}, err
	}
	out := WorkplaceCard{PostRow: PostRow{WorkplaceID: wp.ID, Station: wp.Name, Workshop: wp.Workshop, WorkshopName: wp.WorkshopName, Presence: "not_assigned"}, Scope: wp.Scope,
		Assignments: []WorkplaceAssignee{}}
	for _, r := range posts.Items {
		if r.WorkplaceID == wp.ID {
			out.PostRow = r
		}
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return WorkplaceCard{}, err
	}
	at := s.at(ctx, m)
	for _, a := range pol.PostsIn("") {
		if a.WorkplaceID != wp.ID {
			continue
		}
		v := WorkplaceAssignee{PersonID: a.PersonID, PersonDisplay: displayOf(pol, a.PersonID), ShiftID: a.ShiftID, AssigneeRole: a.AssigneeRole, QualificationOK: true}
		if a.AssigneeRole == accessdom.AssigneePerformer {
			_, v.QualificationOK = pol.QualifiedAt(a.PersonID, wp.Scope, at)
		}
		out.ShiftID = a.ShiftID
		out.Assignments = append(out.Assignments, v)
	}
	return out, nil
}

// WorkplaceHistory — история поста (access.workplace.history): записи потока
// поста, новые сверху. В режиме fixtures — у заготовок.
func (s *Service) WorkplaceHistory(ctx context.Context, workplaceID string, m platform.Moment, p platform.Page) (WorkplaceHistory, error) {
	if !s.live || s.wplog == nil || s.dir == nil {
		return s.Queries.WorkplaceHistory(ctx, workplaceID, m, p)
	}
	if _, ok := s.dir.Workplace(workplaceID); !ok {
		return WorkplaceHistory{}, platform.Fail(errcodes.ApiNotFound, "object", "пост", "id", workplaceID)
	}
	recs, err := s.wplog.Stream(ctx, workplaceID, m)
	if err != nil {
		return WorkplaceHistory{}, err
	}
	var pol accessdom.Policy
	if s.policy != nil {
		if pol, err = s.policy.Policy(ctx); err != nil {
			return WorkplaceHistory{}, err
		}
	}
	recs, err = s.withZonePasses(ctx, workplaceID, recs, pol, m)
	if err != nil {
		return WorkplaceHistory{}, err
	}
	items := WorkplaceEvents(recs, func(id string) string { return displayOf(pol, id) })
	page, next := PageOf(items, p)
	return WorkplaceHistory{WorkplaceID: workplaceID, Items: page, NextCursor: next}, nil
}

// withZonePasses — записи потока поста и проходы СКУД через зону поста тех,
// кто на пост назначался или назначен (эпик 37: присутствие видно в истории поста).
func (s *Service) withZonePasses(ctx context.Context, workplaceID string, recs []accessdom.Record, pol accessdom.Policy, m platform.Moment) ([]accessdom.Record, error) {
	zone := s.zoneOf(workplaceID)
	if s.presence == nil || zone == "" {
		return recs, nil
	}
	people := map[string]bool{}
	for _, r := range recs {
		var d struct {
			PersonID string `json:"person_id"`
		}
		if json.Unmarshal(r.Data, &d) == nil && d.PersonID != "" {
			people[d.PersonID] = true
		}
	}
	for _, a := range pol.PostsIn("") {
		if a.WorkplaceID == workplaceID {
			people[a.PersonID] = true
		}
	}
	passes, err := s.presence.Passes(ctx)
	if err != nil {
		return nil, err
	}
	out := slices.Clone(recs)
	for _, r := range passes {
		if m.AsOf != nil && r.OccurredAt.After(*m.AsOf) {
			continue
		}
		var d ev.AccessZonePassedV1
		if json.Unmarshal(r.Data, &d) != nil || string(d.ZoneID) != zone || !people[string(d.PersonID)] {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

// WorkplaceEvents — события поста из записей его потока, новые сверху; записи
// других типов (факты исполнителя без изделия и т. п.) пропускаются. Сотрудник
// завершения допуска — из открывшей его записи (по workplace_session_id).
func WorkplaceEvents(recs []accessdom.Record, display func(personID string) string) []WorkplaceEvent {
	out := []WorkplaceEvent{}
	sessions := map[string]string{}
	for _, r := range recs {
		e := WorkplaceEvent{Seq: r.Seq, At: r.OccurredAt.UTC(), EventType: r.Type}
		switch catalog.Type(r.Type) {
		case catalog.AccessAssignmentSet:
			var d ev.AccessAssignmentSetV1
			if json.Unmarshal(r.Data, &d) != nil {
				continue
			}
			e.Kind, e.PersonID, e.ShiftID = "assigned", string(d.PersonID), string(d.ShiftID)
		case catalog.AccessAssignmentCleared:
			var d ev.AccessAssignmentClearedV1
			if json.Unmarshal(r.Data, &d) != nil {
				continue
			}
			e.Kind, e.PersonID, e.ShiftID = "cleared", string(d.PersonID), string(d.ShiftID)
			if d.Reason != nil {
				e.Reason = string(d.Reason.Text)
			}
		case catalog.AccessTokenPresenceChanged:
			var d ev.AccessTokenPresenceChangedV1
			if json.Unmarshal(r.Data, &d) != nil {
				continue
			}
			e.Kind, e.PersonID = "token_out", string(d.PersonID)
			if d.Present {
				e.Kind = "token_in"
			}
		case catalog.AccessWorkplaceAdmitted:
			var d ev.AccessWorkplaceAdmittedV1
			if json.Unmarshal(r.Data, &d) != nil {
				continue
			}
			e.Kind, e.PersonID = "admitted", string(d.PersonID)
			if d.ShiftID != nil {
				e.ShiftID = string(*d.ShiftID)
			}
			sessions[string(d.WorkplaceSessionID)] = e.PersonID
		case catalog.AccessWorkplaceReleased:
			var d ev.AccessWorkplaceReleasedV1
			if json.Unmarshal(r.Data, &d) != nil {
				continue
			}
			e.Kind, e.PersonID = "released", sessions[string(d.WorkplaceSessionID)]
		case catalog.AccessWorkplaceRevoked:
			var d ev.AccessWorkplaceRevokedV1
			if json.Unmarshal(r.Data, &d) != nil {
				continue
			}
			e.Kind, e.PersonID, e.Reason = "revoked", sessions[string(d.WorkplaceSessionID)], string(d.Cause)
		case catalog.AccessZonePassed:
			var d ev.AccessZonePassedV1
			if json.Unmarshal(r.Data, &d) != nil {
				continue
			}
			e.Kind, e.PersonID, e.Reason = "zone_out", string(d.PersonID), string(d.ZoneID)
			if d.Direction == ev.AccessZonePassedV1DirectionEnter {
				e.Kind = "zone_in"
			}
		case catalog.SecurityPresenceDeviation:
			var d ev.SecurityPresenceDeviationV1
			if json.Unmarshal(r.Data, &d) != nil {
				continue
			}
			e.Kind, e.PersonID, e.Reason = "presence_deviation", string(d.PersonID), string(d.Deviation)
		default:
			continue
		}
		if e.PersonID != "" && display != nil {
			e.PersonDisplay = display(e.PersonID)
		}
		out = append(out, e)
	}
	slices.SortStableFunc(out, func(a, b WorkplaceEvent) int {
		switch {
		case a.Seq > b.Seq:
			return -1
		case a.Seq < b.Seq:
			return 1
		}
		return 0
	})
	return out
}

// PersonCard — карточка сотрудника (access.person.card): имя, подразделение,
// роли и квалификации на момент чтения, текущие посты; без логина. В режиме
// fixtures — у заготовок (посты мира заготовок не в журнале).
func (s *Service) PersonCard(ctx context.Context, personID string, m platform.Moment) (PersonCard, error) {
	if !s.live || s.policy == nil {
		return s.Queries.PersonCard(ctx, personID, m)
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return PersonCard{}, err
	}
	x, ok := pol.Person(personID)
	if !ok {
		return PersonCard{}, platform.Fail(errcodes.ApiNotFound, "object", "сотрудник", "id", personID)
	}
	v := personView(pol, x, nil, s.at(ctx, m))
	out := PersonCard{PersonID: x.ID, DisplayName: x.Name, OrgUnit: x.OrgUnit, Roles: v.Roles, Qualifications: []AccessQualification{},
		Posts: []PersonPost{}, PolicySeq: pol.Seq}
	qs, err := s.Qualifications(ctx, personID, m)
	if err != nil {
		return PersonCard{}, err
	}
	out.Qualifications = append(out.Qualifications, qs.Items...)
	for _, a := range pol.PostsIn("") {
		if a.PersonID != personID {
			continue
		}
		pp := PersonPost{WorkplaceID: a.WorkplaceID, Station: a.WorkplaceID, ShiftID: a.ShiftID, AssigneeRole: a.AssigneeRole}
		if s.dir != nil {
			if wp, ok := s.dir.Workplace(a.WorkplaceID); ok && wp.Name != "" {
				pp.Station = wp.Name
			}
		}
		out.Posts = append(out.Posts, pp)
	}
	slices.SortFunc(out.Posts, func(a, b PersonPost) int { return strings.Compare(a.WorkplaceID, b.WorkplaceID) })
	return out, nil
}

// displayOf — отображаемое имя сотрудника по политике; нет — псевдоним.
func displayOf(pol accessdom.Policy, personID string) string {
	if x, ok := pol.Person(personID); ok && x.Name != "" {
		return x.Name
	}
	return personID
}
