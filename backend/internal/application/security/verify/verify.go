// Пакет verify — независимый верификатор журнала (AD-9, FR-73, UJ-5): чем
// проверяется, что подмену любой записи нельзя сделать незаметно. Сценарий
// приложения без ввода-вывода по сети: журнал читает роль БД ant_verifier
// (только SELECT), контрольные точки и звенья — у хранителя (их подписи
// проверяет точка входа cmd/verifier по файлу trust-anchors вне БД и вне ant),
// реакции и проекции пересчитывает тот же доменный код, что у воркера.
//
// Проверки (report.v1.json): цепочки и обязательства; журнал критических
// действий один к одному с решениями; контрольные точки и переданные
// хранителю звенья; подписи и их классы; реакции по basis_seq той же
// свёрткой; покрытие; непрерывность source_seq; проекции против журнала;
// задержка записи. Чего ещё нет в системе (генезис, реестр ключей, права на
// момент, кворум версии, отрисовка документов, перечень сборок) — «не
// проверяемо» с причиной; общий вердикт тогда «цело с оговорками».
//
// Слой: application (модуль security). Владелец: эпик 29 (доверие).
package verify

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	app "ant/internal/application/security"
	"ant/internal/contracts/procs"
	"ant/internal/domain/engine"
)

// Journal — чтение журнала верификатором: записи цепочек и содержимое без
// сверки commit (её делает сам верификатор и сообщает как нарушение).
type Journal interface {
	Read(ctx context.Context, q appjournal.ReadQuery) ([]jcEntry, error)
	OpenUnverified(ctx context.Context, e jcEntry) (salt, envelope []byte, err error)
}

// Projections — сохранённые проекции изделия и курсоры воркера (схемы
// engine и journal_state, только чтение).
type Projections interface {
	// ItemRows — проекции изделия: имя → значение (jsonb как текст).
	ItemRows(ctx context.Context, itemID string) (map[string][]byte, error)
	// WorkerCursor — курсор воркера партиции (вход до него обработан).
	WorkerCursor(ctx context.Context, partition int) (int64, error)
}

// Checkpoint — контрольная точка хранителя с итогом проверки подписи.
type Checkpoint struct {
	app.Checkpoint
	// SigErr — подпись hybrid не проверилась по trust-anchors.
	SigErr error
}

// Input — всё, что нужно верификатору.
type Input struct {
	Journal     Journal
	Codec       *engineapp.Codec
	Registry    *engineapp.Registry
	Bundles     engineapp.BundleSource
	Fold        engine.Folder
	Projections Projections
	// Checkpoints — точки хранителя по возрастанию номера; CheckpointsErr —
	// хранитель недоступен.
	Checkpoints    []Checkpoint
	CheckpointsErr error
	// KeeperLinks — звенья, принятые хранителем (цепочка → seq → link).
	KeeperLinks map[string]map[int64]string
	// MaxGap — предельная задержка передачи записи хранителю (policy.audit.max_gap).
	MaxGap time.Duration
	// LateWrite — порог «задержки записи» факта устройства.
	LateWrite time.Duration
	// Partitions — P (курсор воркера по партиции изделия).
	Partitions int
	// RunID — проверять только прогон (пусто — весь журнал).
	RunID string
	// Genesis — итог проверки блока генезиса целиком по якорю (AD-33, эпик
	// 05): cmd/verifier вызывает signing.VerifyGenesis с anchor_fingerprint
	// из trust-anchors. nil — генезиса нет и якорь не закреплён («не проверяемо»).
	Genesis *GenesisResult
}

// GenesisResult — проверка блока генезиса: записей в блоке, отпечаток,
// ошибка (подпись якоря или кворума, чужой якорь, второй генезис, генезиса
// нет при закреплённом якоре — нарушение).
type GenesisResult struct {
	Records int
	Digest  string
	Err     error
}

// genesis — проверка блока генезиса (AD-33) и отпечатка в первой контрольной точке хранителя.
func (v *run) genesis() {
	g := v.in.Genesis
	if g == nil {
		return
	}
	c := v.checks["genesis"]
	c.checked = g.Records
	if g.Err != nil {
		c.reject("genesis.invalid", g.Err.Error(), "main", 1, "")
		return
	}
	if n := len(v.in.Checkpoints); n > 0 {
		if d := v.in.Checkpoints[0].Payload.GenesisDigest; d != nil && *d != g.Digest {
			c.reject("genesis.checkpoint_mismatch", fmt.Sprintf("первая контрольная точка хранителя закрепила генезис %s, в журнале — %s", *d, g.Digest), "main", 1, "")
		}
	}
}

// Report — итог проверки (без подписи: её ставит cmd/verifier).
type Report = procs.VerifierReportV1

// check — накопитель одной проверки.
type check struct {
	name     procs.VerifierReportV1ChecksElemCheck
	checked  int
	rejected bool
	unverif  bool
	findings []procs.VerifierReportV1ChecksElemFindingsElem
}

// maxFindings — предел находок на проверку в отчёте.
const maxFindings = 200

func (c *check) add(status, code, detail string, chain string, seq int64, caRef string) {
	switch status {
	case "rejected":
		c.rejected = true
	case "not_verifiable":
		c.unverif = true
	}
	if len(c.findings) >= maxFindings {
		return
	}
	f := procs.VerifierReportV1ChecksElemFindingsElem{Code: code, Detail: clip(detail)}
	if chain != "" {
		ch := procs.VerifierReportV1ChecksElemFindingsElemChain(chain)
		f.Chain = &ch
	}
	if seq > 0 {
		s := int(seq)
		f.Seq = &s
	}
	if caRef != "" {
		f.CaRef = &caRef
	}
	c.findings = append(c.findings, f)
}

func (c *check) reject(code, detail, chain string, seq int64, caRef string) {
	c.add("rejected", code, detail, chain, seq, caRef)
}

func (c *check) unverifiable(code, detail string) { c.add("not_verifiable", code, detail, "", 0, "") }

func (c *check) row() procs.VerifierReportV1ChecksElem {
	st := procs.VerifierReportV1ChecksElemStatusIntact
	switch {
	case c.rejected:
		st = procs.VerifierReportV1ChecksElemStatusRejected
	case c.unverif:
		st = procs.VerifierReportV1ChecksElemStatusNotVerifiable
	}
	f := c.findings
	if f == nil {
		f = []procs.VerifierReportV1ChecksElemFindingsElem{}
	}
	return procs.VerifierReportV1ChecksElem{Check: c.name, Status: st, Checked: c.checked, Findings: f}
}

func clip(s string) string {
	r := []rune(s)
	if len(r) > 2000 {
		return string(r[:2000])
	}
	return s
}

// Run — проверка журнала целиком. Возвращает отчёт без подписантов, хеша
// бинарника и отпечатка trust-anchors (их ставит cmd/verifier).
func Run(ctx context.Context, in Input) (Report, error) {
	v := &run{in: in, checks: map[procs.VerifierReportV1ChecksElemCheck]*check{}}
	for _, n := range []procs.VerifierReportV1ChecksElemCheck{"chains", "checkpoints", "signatures", "signing_moment", "authority",
		"bpmn_quorum", "reactions", "coverage", "source_seq", "rendering", "late_write", "build", "projections", "genesis"} {
		v.checks[n] = &check{name: n}
		v.order = append(v.order, n)
	}
	if err := v.readChains(ctx); err != nil {
		return Report{}, err
	}
	v.criticalActions()
	v.checkpoints()
	v.signatures()
	v.sourceSeq()
	v.lateWrite()
	if err := v.items(ctx); err != nil {
		return Report{}, err
	}
	v.genesis()
	v.pending()
	return v.report(), nil
}

// ErrNoJournal — журнал пуст.
var ErrNoJournal = errors.New("verify: журнал пуст")

func (v *run) report() Report {
	r := Report{FormatVersion: 1, CryptoProfile: procs.VerifierReportV1CryptoProfileHybrid, PaperRegister: []procs.VerifierReportV1PaperRegisterElem{}}
	if v.in.RunID != "" {
		id := v.in.RunID
		r.RunID = &id
	}
	r.VirtualTime = v.virtual
	r.Range = procs.VerifierReportV1Range{MainFromSeq: int(min(1, v.lastSeq("main"))), MainToSeq: int(v.lastSeq("main")), CaToSeq: int(v.lastSeq("ca"))}
	if n := len(v.in.Checkpoints); n > 0 {
		r.Range.LastCheckpointNo = v.in.Checkpoints[n-1].Payload.CheckpointNo
	}
	r.SignatureClasses = v.classes
	rejected, unverif := false, false
	for _, n := range v.order {
		c := v.checks[n]
		r.Checks = append(r.Checks, c.row())
		rejected = rejected || c.rejected
		unverif = unverif || (!c.rejected && c.unverif)
	}
	switch {
	case rejected:
		r.Verdict = procs.VerifierReportV1VerdictViolated
	case unverif:
		r.Verdict = procs.VerifierReportV1VerdictIntactWithReservations
	default:
		r.Verdict = procs.VerifierReportV1VerdictIntact
	}
	return r
}

// Summary — вывод отчёта по-русски для CLI и журнала процесса.
func Summary(r Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Верификатор: %s — журнал main 1…%d, ca 1…%d, контрольная точка №%d\n",
		strings.ToUpper(app.VerdictText(string(r.Verdict))), r.Range.MainToSeq, r.Range.CaToSeq, r.Range.LastCheckpointNo)
	status := map[string]string{"intact": "цело", "rejected": "ОТВЕРГНУТО", "not_verifiable": "не проверяемо"}
	for _, c := range r.Checks {
		fmt.Fprintf(&b, "  %-15s %-14s проверено %d\n", c.Check, status[string(c.Status)], c.Checked)
		shown := 0
		for _, f := range c.Findings {
			if c.Status == "rejected" || shown < 3 {
				fmt.Fprintf(&b, "      · %s\n", f.Detail)
				shown++
			}
			if shown >= 12 {
				fmt.Fprintf(&b, "      · … всего находок: %d\n", len(c.Findings))
				break
			}
		}
	}
	return b.String()
}

func uniq(xs []string) []string {
	slices.Sort(xs)
	return slices.Compact(xs)
}

func errorsAs(err error, target any) bool { return errors.As(err, target) }
