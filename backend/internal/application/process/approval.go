package process

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	dp "ant/internal/domain/process"
)

// Путь утверждения версии (эпик 39; FR-22, FR-23, FR-24, AD-17, AD-43):
// черновик → на утверждении (лист утверждения — документ эпика 28 с
// маршрутом кворума по шаблону process-version-approval, в нём — читаемая
// разница с действующей версией) → действующая (только после закрытия
// маршрута: document.route.closed, подписи всех ролей кворума) → выведена.
// Изделия в работе остаются на своей версии: версия закреплена хешем при
// запуске изделия (AD-17), вывод версии их не трогает.

// ApprovalTemplate — шаблон листа утверждения версии процесса (normative/documents).
const ApprovalTemplate = "process-version-approval"

// ApprovalRoute — состояние маршрута листа утверждения.
type ApprovalRoute struct {
	// Closed — маршрут закрыт реакцией document.route.closed (AD-43).
	Closed             bool
	RouteClosedEventID string
	// Signatures — засчитанные подписи этапов (роль этапа, подписант).
	Signatures []dp.Signature
	// Have, Need — засчитано и нужно подписей по маршруту.
	Have, Need int
}

// ApprovalDocs — ведомый порт листа утверждения (documents, эпик 28):
// завести документ с маршрутом кворума и прочитать состояние маршрута.
// Подписи кворума ставятся операциями documents (маршрут подписей,
// вторая подпись — политика эпика 26).
type ApprovalDocs interface {
	Request(ctx context.Context, v VersionRecord, decision, comment string, meta platform.CommandMeta) (documentID string, err error)
	Route(ctx context.Context, documentID string) (ApprovalRoute, error)
}

func conflict(field, reason string) error {
	e := platform.Fail(errcodes.ApiValidationFailed, "field", field, "reason", reason)
	e.Detail = reason
	return e
}

var statusText = map[string]string{dp.StatusDraft: "черновик", dp.StatusOnApproval: "на утверждении", dp.StatusActive: "действующая", dp.StatusRetired: "выведена"}

func wantStatus(v VersionRecord, want string) error {
	if v.Status == want {
		return nil
	}
	return conflict("status", fmt.Sprintf("версия %s — %s, ожидается «%s»", v.ID, statusText[v.Status], statusText[want]))
}

// DiffText — читаемая разница для листа утверждения (FR-24), как строки
// process.diff.* интерфейса: «Добавлена точка предъявления после «…»».
func DiffText(entries []ProcessDiffEntry) string {
	if len(entries) == 0 {
		return "Отличий от действующей версии нет"
	}
	parts := make([]string, 0, len(entries))
	for _, e := range entries {
		switch e.Kind {
		case "presentationPointAdded":
			parts = append(parts, "Добавлена точка предъявления после «"+e.Step+"»")
		case "elementAdded":
			parts = append(parts, "Добавлен элемент «"+e.Element+"»")
		case "elementRemoved":
			parts = append(parts, "Удалён элемент «"+e.Element+"»")
		case "thresholdChanged":
			parts = append(parts, fmt.Sprintf("Порог уверенности для вида «%s»: %v → %v", e.DefectType, e.From, e.To))
		case "propertyChanged":
			parts = append(parts, fmt.Sprintf("«%s»: %s %v → %v", e.Element, e.Property, show(e.From), show(e.To)))
		}
	}
	return strings.Join(parts, "; ")
}

func show(v any) any {
	if v == nil {
		return "—"
	}
	return v
}

// cut — не длиннее n символов (поля документа ограничены контрактом).
func cut(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func (s *LiveService) today(ctx context.Context) (time.Time, error) {
	if s.Clock != nil {
		return s.Clock(ctx)
	}
	return time.Now().UTC(), nil
}

func (s *LiveService) record(ctx context.Context, t catalog.Type, v VersionRecord, meta platform.CommandMeta, at time.Time, data map[string]any) (platform.Receipt, error) {
	if s.Recorder == nil {
		return platform.Receipt{CommandID: meta.CommandID, EventIDs: []string{v.ID}, RecordedAt: at}, nil
	}
	return s.Recorder.Record(ctx, t, VersionStream(v.ID), platform.PrincipalFrom(ctx).PersonID, meta, at, data)
}

// SubmitVersion — отправить черновик на утверждение кворумом
// (process.version.submit, FR-23): лист утверждения с маршрутом кворума и
// читаемой разницей с действующей версией процесса; normative.version.submitted.
func (s *LiveService) SubmitVersion(ctx context.Context, versionID string, in SubmitVersion) (platform.Receipt, error) {
	if s.Library == nil || s.Approvals == nil {
		return platform.Receipt{}, platform.NotImplemented("process.version.submit")
	}
	v, err := s.version(ctx, versionID)
	if err != nil {
		return platform.Receipt{}, err
	}
	if err := wantStatus(v, dp.StatusDraft); err != nil {
		return platform.Receipt{}, err
	}
	proc := ProcessOf(v)
	base, err := s.resolve(ctx, proc.ID, "")
	if err != nil {
		return platform.Receipt{}, err
	}
	if base.Status != dp.StatusActive {
		base = v // действующей версии нет — первая версия процесса
	}
	entries := []ProcessDiffEntry{}
	if base.ID != v.ID {
		nd, err := definition(v)
		if err != nil {
			return platform.Receipt{}, err
		}
		od, err := definition(base)
		if err != nil {
			return platform.Receipt{}, err
		}
		entries = DiffDefinitions(od, nd)
	}
	comment := DiffText(entries)
	if in.Note != "" {
		comment = in.Note + "\n" + comment
	}
	decision := fmt.Sprintf("Ввести в действие версию %s процесса «%s» вместо %s", v.Label, proc.Name, base.Label)
	if base.ID == v.ID {
		// Новый процесс (эпик 39): действующей версии ещё нет — первая версия.
		decision = fmt.Sprintf("Ввести в действие первую версию %s нового процесса «%s»", v.Label, proc.Name)
		comment = strings.Replace(comment, "Отличий от действующей версии нет", "Новый процесс: действующей версии нет", 1)
	}
	docID, err := s.Approvals.Request(ctx, v, cut(decision, 128), cut(comment, 2000), in.CommandMeta())
	if err != nil {
		return platform.Receipt{}, err
	}
	at, err := s.today(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	v.Status, v.ApprovalDocumentID = dp.StatusOnApproval, docID
	if err := s.Library.Update(ctx, v); err != nil {
		return platform.Receipt{}, err
	}
	data := map[string]any{"version_id": v.ID, "process_version_hash": v.Hash, "bundle_digest": v.Hash, "approval_document_id": docID}
	if base.ID != v.ID {
		data["base_version_id"] = base.ID
	}
	return s.record(ctx, catalog.NormativeVersionSubmitted, v, in.CommandMeta(), at, data)
}

// ActivateVersion — ввести версию в действие (process.version.activate,
// FR-23, AD-17): только после закрытия маршрута листа утверждения и с
// подписями всех ролей кворума; прежняя действующая версия того же процесса
// выводится — изделия, запущенные по ней, доделываются по своей версии.
func (s *LiveService) ActivateVersion(ctx context.Context, versionID string, in ActivateVersion) (platform.Receipt, error) {
	if s.Library == nil || s.Approvals == nil {
		return platform.Receipt{}, platform.NotImplemented("process.version.activate")
	}
	v, err := s.version(ctx, versionID)
	if err != nil {
		return platform.Receipt{}, err
	}
	if err := wantStatus(v, dp.StatusOnApproval); err != nil {
		return platform.Receipt{}, err
	}
	route, err := s.Approvals.Route(ctx, v.ApprovalDocumentID)
	if err != nil {
		return platform.Receipt{}, err
	}
	appr := dp.Approval{VersionID: v.ID, Hash: v.Hash, Signatures: route.Signatures, RouteClosedEventID: route.RouteClosedEventID}
	if missing := appr.Missing(); !route.Closed || len(missing) > 0 {
		who := strings.Join(missing, ", ")
		if who == "" {
			who = "маршрут листа утверждения не закрыт"
		}
		e := platform.Fail(errcodes.ProcessQuorumIncomplete, "who", who)
		e.Detail = "Не хватает подписей кворума: " + who
		return platform.Receipt{}, e
	}
	if in.RouteClosedEventID != "" && !strings.EqualFold(in.RouteClosedEventID, route.RouteClosedEventID) {
		return platform.Receipt{}, conflict("route_closed_event_id", "маршрут закрыт записью "+route.RouteClosedEventID)
	}
	at, err := s.today(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	all, err := s.Library.List(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	prev, hadPrev := Active(VersionsOf(all, ProcessOf(v).ID))
	v.Status, v.EffectiveFrom, v.Signatures, v.RouteClosedEventID = dp.StatusActive, &at, route.Signatures, route.RouteClosedEventID
	if err := s.Library.Update(ctx, v); err != nil {
		return platform.Receipt{}, err
	}
	if hadPrev && prev.ID != v.ID {
		prev.Status = dp.StatusRetired
		if err := s.Library.Update(ctx, prev); err != nil {
			return platform.Receipt{}, err
		}
	}
	return s.record(ctx, catalog.NormativeVersionActivated, v, in.CommandMeta(), at,
		map[string]any{"version_id": v.ID, "process_version_hash": v.Hash, "route_closed_event_id": route.RouteClosedEventID})
}

// RetireVersion — вывести версию (process.version.retire, FR-22): новые
// изделия по ней не запускаются, изделия в работе доделываются по ней.
func (s *LiveService) RetireVersion(ctx context.Context, versionID string, in RetireVersion) (platform.Receipt, error) {
	if s.Library == nil {
		return platform.Receipt{}, platform.NotImplemented("process.version.retire")
	}
	v, err := s.version(ctx, versionID)
	if err != nil {
		return platform.Receipt{}, err
	}
	if err := wantStatus(v, dp.StatusActive); err != nil {
		return platform.Receipt{}, err
	}
	at, err := s.today(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	v.Status = dp.StatusRetired
	if err := s.Library.Update(ctx, v); err != nil {
		return platform.Receipt{}, err
	}
	return s.record(ctx, catalog.NormativeVersionRetired, v, in.CommandMeta(), at,
		map[string]any{"version_id": v.ID, "reason": map[string]any{"text": in.ReasonText}})
}
