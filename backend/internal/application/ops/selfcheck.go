package ops

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	dom "ant/internal/domain/ops"
)

// SelfCheck — самопроверка после старта (FR-109, AD-25): журнал читается и
// его записи открываются, генезис (если есть — один), миграции модулей
// применены, роли БД на месте и журнал защищён от изменения, роли процесса
// работают. Итог — строка лога «инициализация без критических ошибок» или
// перечень находок и ответ /readyz.
type SelfCheck struct {
	Journal appjournal.JournalStore
	// Database — миграции и роли БД (nil — проверки пропущены с предупреждением).
	Database Database
	// RequireGenesis — генезис обязателен (роль init в сборке): его
	// отсутствие — критично; иначе — предупреждение для непустого журнала.
	RequireGenesis bool
	// Runtime — аренды: взяли ли роли-лидеры процесса свою аренду.
	Runtime Runtime
	// Roles — роли этого процесса; Pending — заглушки до своих эпиков.
	Roles   []string
	Pending []string
	// Idle — роли-лидеры, которые в этой конфигурации аренду не берут
	// (outbox без включённой 1С, security без хранителя).
	Idle []string
	// LeaderWait — сколько ждать аренды ролей-лидеров (0 — не ждать).
	LeaderWait time.Duration
	// Now — InfraClock; nil — time.Now.
	Now func() time.Time
	Log *slog.Logger
}

// Run выполняет проверки и пишет итог в лог.
func (c *SelfCheck) Run(ctx context.Context) SelfCheckView {
	now := time.Now
	if c.Now != nil {
		now = c.Now
	}
	var fs []dom.Finding
	head, jf := c.journal(ctx)
	fs = append(fs, jf...)
	if jf == nil {
		fs = append(fs, c.genesis(ctx, head)...)
	}
	if c.Database != nil {
		fs = append(fs, dom.MigrationFindings(c.Database.Migrations(ctx))...)
		missing, privs, err := c.Database.DBRoles(ctx)
		if err != nil {
			fs = append(fs, dom.Finding{Check: dom.CheckDBRoles, Severity: dom.Warning, Text: "роли БД не проверены: " + err.Error()})
		} else {
			fs = append(fs, dom.DBRoleFindings(missing, privs)...)
		}
	} else {
		fs = append(fs, dom.Finding{Check: dom.CheckMigrations, Severity: dom.Warning, Text: "миграции и роли БД не проверялись (нет подключения)"})
	}
	fs = append(fs, dom.RoleFindings(c.processRoles(ctx))...)

	v := SelfCheckView{OK: !dom.HasCritical(fs), Summary: dom.Summary(fs), Findings: []SelfCheckFinding{}, CheckedAt: now().UTC()}
	for _, f := range fs {
		v.Findings = append(v.Findings, SelfCheckFinding{Check: f.Check, Severity: string(f.Severity), Text: f.Text})
	}
	c.report(ctx, v)
	return v
}

func (c *SelfCheck) report(ctx context.Context, v SelfCheckView) {
	log := c.Log
	if log == nil {
		return
	}
	for _, f := range v.Findings {
		lvl := slog.LevelWarn
		if f.Severity == string(dom.Critical) {
			lvl = slog.LevelError
		}
		log.Log(ctx, lvl, "самопроверка: "+f.Text, "check", f.Check, "severity", f.Severity)
	}
	if v.OK {
		log.Info(dom.MsgOK, "warnings", len(v.Findings))
		return
	}
	log.Error(v.Summary, "findings", len(v.Findings))
}

// journal — голова читается, первая и последняя записи открываются (конверт
// расшифровывается KEK этой установки, AD-23).
func (c *SelfCheck) journal(ctx context.Context) (int64, []dom.Finding) {
	crit := func(format string, a ...any) []dom.Finding {
		return []dom.Finding{{Check: dom.CheckJournal, Severity: dom.Critical, Text: fmt.Sprintf(format, a...)}}
	}
	h, err := c.Journal.Head(ctx)
	if err != nil {
		return 0, crit("головы цепочек не читаются: %v", err)
	}
	if h.MainSeq == 0 {
		return 0, nil
	}
	for _, q := range []appjournal.ReadQuery{{Limit: 1}, {Backward: true, Limit: 1}} {
		es, err := c.Journal.Read(ctx, q)
		if err != nil {
			return h.MainSeq, crit("записи не читаются: %v", err)
		}
		if len(es) == 0 {
			return h.MainSeq, crit("голова seq %d есть, записей нет", h.MainSeq)
		}
		if _, err := c.Journal.Open(ctx, es[0]); err != nil {
			return h.MainSeq, crit("запись seq %d не открывается (KEK, AD-23): %v", es[0].Seq, err)
		}
	}
	return h.MainSeq, nil
}

// genesis — записи journal.genesis.recorded (AD-33).
func (c *SelfCheck) genesis(ctx context.Context, head int64) []dom.Finding {
	es, err := c.Journal.Read(ctx, appjournal.ReadQuery{EventType: string(catalog.JournalGenesisRecorded), Limit: 10})
	if err != nil {
		return []dom.Finding{{Check: dom.CheckGenesis, Severity: dom.Critical, Text: "генезис не читается: " + err.Error()}}
	}
	seqs := make([]int64, 0, len(es))
	for _, e := range es {
		seqs = append(seqs, int64(e.Seq))
	}
	return dom.GenesisFindings(head, seqs, c.RequireGenesis)
}

// processRoles — роли процесса; для ролей-лидеров ждёт аренду до LeaderWait.
func (c *SelfCheck) processRoles(ctx context.Context) []dom.ProcessRole {
	roles := make([]dom.ProcessRole, 0, len(c.Roles))
	for _, r := range c.Roles {
		roles = append(roles, dom.ProcessRole{Name: r, Pending: slices.Contains(c.Pending, r),
			Leader: slices.Contains(dom.LeaderRoles, r) && !slices.Contains(c.Pending, r) && !slices.Contains(c.Idle, r)})
	}
	if c.Runtime == nil {
		for i := range roles {
			roles[i].Held = true
		}
		return roles
	}
	deadline := time.Now().Add(c.LeaderWait)
	for {
		held := map[string]bool{}
		if ls, err := c.Runtime.Leases(ctx); err == nil {
			now := time.Now()
			if c.Now != nil {
				now = c.Now()
			}
			for _, l := range ls {
				if l.ExpiresAt.After(now) {
					held[l.Name] = true
				}
			}
		}
		all := true
		for i := range roles {
			roles[i].Held = held[roles[i].Name]
			all = all && (!roles[i].Leader || roles[i].Held)
		}
		if all || !time.Now().Before(deadline) {
			return roles
		}
		select {
		case <-ctx.Done():
			return roles
		case <-time.After(250 * time.Millisecond):
		}
	}
}
