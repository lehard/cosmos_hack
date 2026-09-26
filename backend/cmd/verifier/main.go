// Команда verifier — независимый верификатор журнала (AD-9, AD-46, FR-73,
// UJ-5): отдельный процесс по границе доверия. Журнал читает ролью БД
// ant_verifier (только SELECT), корни доверия берёт из файла trust-anchors
// своего тома (вне БД и вне ant), контрольные точки и звенья — только у
// хранителя (mTLS). Переигрывает журнал тем же доменным кодом, что воркер, и
// выдаёт «цело», «цело с оговорками» или место и тип нарушения. Отчёт
// подписывает своим ключом (hybrid) с хешем своего бинарника и сдаёт
// хранителю; ant забирает его оттуда и журналирует security.integrity.checked.
//
//	verifier -once        — одна проверка: отчёт в stdout (и -out), сдать хранителю;
//	verifier              — по расписанию (-interval);
//	make verify           — одна проверка в демо-стенде.
//
// Ограничение (AD-9): ошибки и закладки в самом общем доменном коде
// верификатор не обнаруживает (смягчение — воспроизводимая сборка и хеш в
// нормативном слое; вторая реализация — описание).
//
// Слой: точка входа (cmd/*). Владелец: эпик 29 (доверие).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ant/cmd/internal/config"
	"ant/cmd/internal/db"
	"ant/cmd/internal/enginewire"
	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	app "ant/internal/application/security"
	"ant/internal/application/security/verify"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
	"ant/internal/infrastructure/integration/security/keeper"
	"ant/internal/infrastructure/observability/logging"
	"ant/internal/infrastructure/security/atrest"
	"ant/internal/infrastructure/security/hybrid"
	"ant/internal/infrastructure/security/mtls"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/clock"
)

var version = "dev"

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

type options struct {
	data, keeperURL, kek, out, runID string
	once, submit                     bool
	interval, maxGap, lateWrite      time.Duration
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("verifier", flag.ContinueOnError)
	fs.SetOutput(stderr)
	cfgPath := fs.String("config", envOr("ANT_CONFIG", "/etc/ant/ant.yaml"), "ant.yaml: подключение к БД (роль ant_verifier), параметры аудита")
	var o options
	fs.StringVar(&o.data, "data", envOr("ANT_VERIFIER_DATA", "/var/lib/verifier"), "том верификатора: ключи, trust-anchors, сертификат mTLS")
	fs.StringVar(&o.keeperURL, "keeper", envOr("ANT_VERIFIER_KEEPER_URL", ""), "адрес хранителя (по умолчанию security.keeper_url)")
	fs.StringVar(&o.kek, "kek", envOr("ANT_VERIFIER_KEK", ""), "файл KEK (по умолчанию security.kek_file): без него commit зашифрованных записей не сверяется")
	fs.StringVar(&o.out, "out", "", "записать отчёт (JSON) в файл")
	fs.StringVar(&o.runID, "run", "", "проверять изделия только этого прогона сценария")
	fs.BoolVar(&o.once, "once", false, "одна проверка и выход")
	fs.BoolVar(&o.submit, "submit", true, "сдать подписанный отчёт хранителю")
	fs.DurationVar(&o.interval, "interval", 0, "интервал проверок (по умолчанию security.verifier_interval)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := config.Load(*cfgPath, os.LookupEnv)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if o.keeperURL == "" {
		o.keeperURL = cfg.Security.KeeperURL
	}
	if o.kek == "" {
		o.kek = cfg.Security.KEKFile
	}
	if o.interval <= 0 {
		o.interval = cfg.Security.VerifierInterval
	}
	o.maxGap, o.lateWrite = cfg.Security.MaxGap, cfg.Security.LateWrite
	log := logging.New(stderr, cfg.Log.Level, "verifier")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	v, err := open(ctx, cfg, o, log)
	if err != nil {
		log.Error("верификатор не собран", "err", err)
		return 1
	}
	defer v.pool.Close()
	if o.once {
		r, err := v.check(ctx, stdout)
		if err != nil {
			log.Error("проверка не выполнена", "err", err)
			return 1
		}
		if r.Verdict == "violated" {
			return 3
		}
		return 0
	}
	if o.interval <= 0 {
		o.interval = time.Minute
	}
	t := time.NewTicker(o.interval)
	defer t.Stop()
	for {
		if _, err := v.check(ctx, io.Discard); err != nil && ctx.Err() == nil {
			log.Error("проверка не выполнена", "err", err)
		}
		select {
		case <-ctx.Done():
			return 0
		case <-t.C:
		}
	}
}

type verifier struct {
	o        options
	log      *slog.Logger
	pool     *pgxpool.Pool
	store    *journalstore.Store
	codec    *engineapp.Codec
	bundles  engineapp.BundleSource
	registry *engineapp.Registry
	keeper   app.Keeper
	signer   *hybrid.Signer
	anchors  hybrid.Anchors
	digest   string
	build    string
	parts    int
}

func open(ctx context.Context, cfg *config.Config, o options, log *slog.Logger) (*verifier, error) {
	pc, err := db.Config(cfg.DB, "ant-verifier")
	if err != nil {
		return nil, err
	}
	// Только чтение: каждое соединение — роль ant_verifier (AD-1, AD-9).
	pc.AfterConnect = journalstore.AfterConnectRole("ant_verifier")
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, err
	}
	var opts []journalstore.Option
	if k, err := atrest.Load(o.kek); err == nil {
		opts = append(opts, journalstore.WithCipher(k))
	} else if !errors.Is(err, atrest.ErrNoKEK) {
		return nil, err
	} else {
		log.Warn("KEK не смонтирован: commit зашифрованных записей не сверяется", "kek", o.kek)
	}
	store := journalstore.NewStore(pool, clock.System{}, opts...)
	v := &verifier{o: o, log: log, pool: pool, store: store, registry: enginewire.Registry(), parts: cfg.Engine.Partitions}
	v.codec = &engineapp.Codec{Store: lenient{store}, KeyRef: "verifier@1", Profile: "gost", Partitions: cfg.Engine.Partitions}
	if v.bundles, err = enginewire.Bundles(pool, v.codec); err != nil {
		return nil, err
	}
	if v.signer, err = hybrid.Load(filepath.Join(o.data, "keys"), "verifier"); err != nil {
		return nil, fmt.Errorf("ключи верификатора (keeper -init -init-verifier): %w", err)
	}
	if v.anchors, v.digest, err = hybrid.LoadAnchors(filepath.Join(o.data, "trust-anchors.json")); err != nil {
		return nil, err
	}
	if o.keeperURL != "" {
		tlsCfg, err := mtls.Client(mtls.Files{Dir: filepath.Join(o.data, "pki"), Name: "verifier"})
		if err != nil {
			return nil, err
		}
		v.keeper = keeper.New(o.keeperURL, tlsCfg)
	}
	v.build = selfHash()
	return v, nil
}

// selfHash — хеш бинарника верификатора (AD-46).
func selfHash() string {
	p, err := os.Executable()
	if err != nil {
		return dj.H([]byte("verifier"), []byte(version)).String()
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return dj.H([]byte("verifier"), []byte(version)).String()
	}
	return dj.H(b).String()
}

// lenient — журнал для свёртки верификатора: содержимое записи без сверки
// commit (расхождение commit верификатор сообщает сам как нарушение, а
// переигрывает то, что лежит в базе).
type lenient struct{ *journalstore.Store }

func (l lenient) Open(ctx context.Context, e jc.JournalEntry) (appjournal.Envelope, error) {
	salt, env, err := l.OpenUnverified(ctx, e)
	if err != nil {
		return appjournal.Envelope{}, err
	}
	return appjournal.Envelope{Raw: env, Salt: salt}, nil
}

// projections — проекции изделия и курсоры воркера (только чтение).
type projections struct{ pool *pgxpool.Pool }

func (p projections) ItemRows(ctx context.Context, itemID string) (map[string][]byte, error) {
	rows, err := p.pool.Query(ctx, "SELECT name, value::text FROM engine.projections WHERE item_id = $1 AND key = $1", itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]byte{}
	for rows.Next() {
		var n, v string
		if err := rows.Scan(&n, &v); err != nil {
			return nil, err
		}
		out[n] = []byte(v)
	}
	return out, rows.Err()
}

func (p projections) WorkerCursor(ctx context.Context, partition int) (int64, error) {
	var seq int64
	err := p.pool.QueryRow(ctx, "SELECT COALESCE(MAX(seq), 0) FROM journal_state.consumer_offsets WHERE name = $1 AND partition = $2",
		appjournal.WorkerConsumer, partition).Scan(&seq)
	return seq, err
}

func (v *verifier) check(ctx context.Context, stdout io.Writer) (verify.Report, error) {
	started := time.Now()
	in := verify.Input{Journal: v.store, Codec: v.codec, Registry: v.registry, Bundles: v.bundles,
		Projections: projections{v.pool}, MaxGap: v.o.maxGap, LateWrite: v.o.lateWrite, Partitions: v.parts, RunID: v.o.runID}
	if v.keeper == nil {
		in.CheckpointsErr = errors.New("адрес хранителя не задан")
	} else {
		in.Checkpoints, in.KeeperLinks, in.CheckpointsErr = v.fromKeeper(ctx)
	}
	r, err := verify.Run(ctx, in)
	if err != nil {
		return r, err
	}
	r.GeneratedAt = dj.FormatTime(time.Now())
	r.Signers = v.signer.KeyIDs()
	r.VerifierBuild = v.build
	r.TrustAnchorsDigest = v.digest
	payload, err := json.Marshal(r)
	if err != nil {
		return r, err
	}
	if payload, err = dj.Canonical(payload); err != nil {
		return r, err
	}
	env, err := v.signer.Sign("verifier-report", payload)
	if err != nil {
		return r, err
	}
	_, digest, _ := app.PayloadOf(env)
	_, _ = fmt.Fprint(stdout, verify.Summary(r))
	_, _ = fmt.Fprintf(stdout, "Отчёт %s подписан ключами %v, сборка %s, за %s\n", digest, r.Signers, v.build, time.Since(started).Round(time.Millisecond))
	if v.o.out != "" {
		if err := os.WriteFile(v.o.out, env, 0o644); err != nil {
			return r, err
		}
	}
	if v.o.submit && v.keeper != nil {
		if err := v.keeper.SubmitReport(ctx, env); err != nil {
			return r, fmt.Errorf("отчёт не сдан хранителю: %w", err)
		}
		_, _ = fmt.Fprintln(stdout, "Отчёт сдан хранителю.")
	}
	v.log.Info("проверка", "verdict", r.Verdict, "main_to_seq", r.Range.MainToSeq, "report_digest", digest, "took", time.Since(started).String())
	return r, nil
}

// fromKeeper — контрольные точки (подпись hybrid проверяется по
// trust-anchors) и звенья, принятые хранителем.
func (v *verifier) fromKeeper(ctx context.Context) ([]verify.Checkpoint, map[string]map[int64]string, error) {
	var out []verify.Checkpoint
	after := int64(0)
	for {
		page, err := v.keeper.Checkpoints(ctx, after, 1000)
		if err != nil {
			return nil, nil, err
		}
		for _, c := range page {
			_, sigErr := v.anchors.Verify(c.Envelope, "checkpoint", "keeper")
			out = append(out, verify.Checkpoint{Checkpoint: c, SigErr: sigErr})
			after = int64(c.Payload.CheckpointNo)
		}
		if len(page) < 1000 {
			break
		}
	}
	links := map[string]map[int64]string{}
	for _, ch := range app.Chains {
		links[ch] = map[int64]string{}
		after := int64(0)
		for {
			page, err := v.keeper.Links(ctx, ch, after, 10000)
			if err != nil {
				return nil, nil, err
			}
			for _, l := range page {
				links[ch][l.Seq] = l.Link
				after = l.Seq
			}
			if len(page) < 10000 {
				break
			}
		}
	}
	return out, links, nil
}
