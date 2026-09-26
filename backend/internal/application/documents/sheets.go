package documents

import (
	"context"

	"ant/internal/application/platform"
	dom "ant/internal/domain/documents"
)

// Лист утверждения версии процесса (каталог документов: «Нормативный слой —
// лист утверждения версии процесса», AD-43) для модуля process (эпик 17):
// отправка версии на кворум — «Запросить решение» с действием
// process.version.submit (шаблон process-version-approval, объект
// process_version:‹id›); активация — по закрытому маршруту листа. Модуль
// process читает итог через свой порт QuorumVerifier; адаптер к нему
// собирается в cmd/ant из ApprovalSheet.

// SheetSignature — засчитанная подпись кворума.
type SheetSignature struct {
	Stage     int
	Person    string
	Role      string
	Authority string
	EventID   string
}

// Sheet — состояние листа утверждения объекта.
type Sheet struct {
	DocumentID string
	Version    int
	DocDigest  string
	// Decision, Comment — что утверждается (из запроса: хеш версии, читаемая разница).
	Decision string
	Comment  string
	Closed   bool
	// RouteClosedEventID — реакция document.route.closed текущей версии листа.
	RouteClosedEventID string
	Signatures         []SheetSignature
}

// ApprovalSheet — последний лист утверждения объекта subjectRef
// (`process_version:‹id›`); ok=false — листа нет.
func (s *Service) ApprovalSheet(ctx context.Context, subjectRef string) (Sheet, bool, error) {
	if !s.live() {
		return Sheet{}, false, platform.NotImplemented("documents")
	}
	found, err := s.bySubject(ctx, subjectRef, platform.Moment{})
	if err != nil {
		return Sheet{}, false, err
	}
	for i := len(found) - 1; i >= 0; i-- {
		f := found[i]
		if f.doc.DocType != dom.DocGeneric {
			continue
		}
		cur := f.doc.Current()
		if cur == nil {
			continue
		}
		sh := Sheet{DocumentID: f.doc.ID, Version: cur.No, DocDigest: cur.Digest, Decision: f.doc.Context.Decision, Comment: f.doc.Context.Comment, Closed: cur.Closed}
		ev := dom.Evaluate(cur, f.view.Env.People, f.view.State.Participants)
		for _, st := range ev.Stages {
			for _, sg := range st.Counted {
				sh.Signatures = append(sh.Signatures, SheetSignature{Stage: st.Stage.Stage, Person: sg.Person, Role: st.Stage.Role, Authority: st.Stage.AuthorityID, EventID: sg.EventID})
			}
		}
		for _, r := range f.view.Reactions {
			if r.Slot == dom.Slot(dom.RuleRoute, f.doc.ID, cur.No) {
				sh.RouteClosedEventID = r.EventID
			}
		}
		return sh, true, nil
	}
	return Sheet{}, false, nil
}
